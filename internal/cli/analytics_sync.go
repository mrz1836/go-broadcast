package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/mrz1836/go-broadcast/internal/analytics"
	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// parseRepoName parses "owner/name" format and returns owner and name.
// Returns an error if the format is invalid.
func parseRepoName(fullName string) (owner, name string, err error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository format %q, expected owner/name (e.g., owner/repo)", fullName) //nolint:err113 // user-facing CLI error
	}
	return parts[0], parts[1], nil
}

// determineSyncType returns the sync type based on flags
func determineSyncType(securityOnly bool) string {
	if securityOnly {
		return "security_only"
	}
	return "full"
}

// newAnalyticsSyncCmd creates the analytics sync command
func newAnalyticsSyncCmd() *cobra.Command {
	var (
		org            string
		repo           string
		allOrgs        bool
		securityOnly   bool
		full           bool
		dryRun         bool
		progress       bool
		rateLimit      float64
		burst          int
		interRepoDelay int
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync repository analytics data",
		Long: `Collect repo stats from GitHub API using batched GraphQL for metadata
and concurrent REST for security alerts. Syncs 60-75 repos across multiple orgs
with change detection to minimize database writes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Validate flag combinations
			if repo != "" && org != "" {
				return fmt.Errorf("cannot specify both --repo and --org flags") //nolint:err113 // user-facing CLI error
			}

			// Parse repo name if provided
			var repoOwner, repoName string
			if repo != "" {
				var err error
				repoOwner, repoName, err = parseRepoName(repo)
				if err != nil {
					return err
				}
			}

			// Initialize logger from context (set by PersistentPreRunE)
			logger, ok := ctx.Value(loggerContextKey{}).(*logrus.Logger)
			if !ok || logger == nil {
				logger = logrus.New()
			}

			// Open database
			database, err := openDatabase()
			if err != nil {
				return err
			}
			defer func() { _ = database.Close() }()

			gormDB := database.DB()
			analyticsRepo := db.NewAnalyticsRepo(gormDB)
			repoRepo := db.NewRepoRepository(gormDB)
			orgRepo := db.NewOrganizationRepository(gormDB)

			// Initialize GitHub client
			ghClient, err := gh.NewClient(ctx, logger, &logging.LogConfig{})
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			// Create shared throttle for rate limiting
			throttleCfg := analytics.DefaultThrottleConfig()
			throttleCfg.RequestsPerSecond = rateLimit
			throttleCfg.BurstSize = burst
			throttleCfg.InterRepoDelay = time.Duration(interRepoDelay) * time.Millisecond
			throttle := analytics.NewThrottle(throttleCfg, logger)

			// Create analytics pipeline
			pipeline := analytics.NewPipeline(ghClient, analyticsRepo, repoRepo, orgRepo, logger, throttle)

			// Snapshot the rate limit before any GitHub request, so the run's API usage
			// can be measured and recorded. Reading the rate limit is free.
			usageMeter, usageErr := gh.StartUsageMeter(ctx, ghClient)
			if usageErr != nil {
				logger.WithError(usageErr).Debug("GitHub API usage will not be measured")
			}

			// Pre-flight rate limit check
			if showProgress := progress; showProgress {
				if usageMeter == nil {
					output.Warn(fmt.Sprintf("Could not check rate limit: %v", usageErr))
				} else {
					rateLimitInfo := analytics.NewRateLimitInfo(usageMeter.Baseline())
					analytics.DisplayRateLimitInfo(rateLimitInfo)

					// Estimate cost and warn if budget is low
					repoCount := 0
					if repo != "" {
						repoCount = 1
					}
					// For org/full sync, we'll warn with a conservative estimate
					if repoCount == 0 {
						repoCount = 50 // Conservative default estimate
					}
					estimate := analytics.EstimateSyncCost(repoCount)
					analytics.WarnIfBudgetLow(rateLimitInfo, estimate)
				}

				output.Info(fmt.Sprintf("Throttle: %.1f req/s, burst %d, %dms inter-repo delay",
					throttleCfg.RequestsPerSecond, throttleCfg.BurstSize,
					throttleCfg.InterRepoDelay.Milliseconds()))
			}

			// Determine sync scope
			syncType := determineSyncType(securityOnly)

			// Start sync run tracking
			syncRun, err := pipeline.StartSyncRun(ctx, syncType, org, repo)
			if err != nil {
				return fmt.Errorf("failed to start sync run: %w", err)
			}

			// Execute sync based on flags
			var syncErr error
			if repo != "" {
				// Single repository sync
				syncErr = syncSingleRepository(ctx, pipeline, analyticsRepo, syncRun, repoOwner, repoName, progress, dryRun, full)
			} else if org != "" {
				// Organization sync
				syncErr = syncOrganization(ctx, pipeline, analyticsRepo, syncRun, org, progress, dryRun, full)
			} else if allOrgs {
				// Explicit org-wide sync: discover all repos in all organizations (legacy behavior)
				syncErr = syncAllOrganizations(ctx, pipeline, analyticsRepo, syncRun, progress, dryRun, full)
			} else {
				// Default: sync only managed repos from DB targets (from sync.yaml groups)
				syncErr = syncManagedRepos(ctx, gormDB, pipeline, analyticsRepo, syncRun, progress, dryRun, full)
			}

			// Determine final status
			status := "completed"
			if syncRun.ReposFailed > 0 {
				status = "partial"
			}
			if syncErr != nil {
				status = "failed"
			}

			// Record the run's API usage before completing it, so it is persisted with the run
			apiUsage := measureRunAPIUsage(ctx, usageMeter, logger)
			applyAPIUsage(syncRun, apiUsage)

			// Complete sync run
			if completeErr := pipeline.CompleteSyncRun(ctx, syncRun, status); completeErr != nil {
				output.Warn(fmt.Sprintf("Failed to update sync run: %v", completeErr))
			}

			// Display summary
			throttleStats := throttle.Stats()
			displaySyncSummary(syncRun, status, &throttleStats, apiUsage)

			// Return original sync error if any
			return syncErr
		},
	}

	cmd.Flags().StringVar(&org, "org", "", "Sync specific owner (organization or user account)")
	cmd.Flags().BoolVar(&allOrgs, "all-orgs", false, "Sync all repositories in all organizations (org-wide discovery, not just managed repos)")
	cmd.Flags().StringVar(&repo, "repo", "", "Sync specific repository only (owner/name)")
	cmd.Flags().BoolVar(&securityOnly, "security-only", false, "Sync security alerts only")
	cmd.Flags().BoolVar(&full, "full", false, "Force full sync (ignore change detection and re-download unchanged CI artifacts)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be synced without writing to DB")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress output")
	cmd.Flags().Float64Var(&rateLimit, "rate-limit", 1.0, "Max GitHub API requests per second")
	cmd.Flags().IntVar(&burst, "burst", 3, "Max burst size for rate limiter")
	cmd.Flags().IntVar(&interRepoDelay, "inter-repo-delay", 500, "Delay between repos in milliseconds")

	return cmd
}

// syncSingleRepository syncs a single repository
func syncSingleRepository(
	ctx context.Context,
	pipeline *analytics.Pipeline,
	analyticsRepo db.AnalyticsRepo,
	syncRun *db.SyncRun,
	owner, name string,
	showProgress, isDryRun, forceFull bool,
) error {
	fullName := fmt.Sprintf("%s/%s", owner, name)

	if showProgress {
		output.Info(fmt.Sprintf("Syncing repository: %s", fullName))
	}

	if isDryRun {
		output.Info(fmt.Sprintf("[DRY RUN] Would sync repository: %s", fullName))
		output.Info("")
		output.Info("  Actions that would be performed:")
		output.Info("  • Collect repository metadata (stars, forks, issues, PRs)")
		output.Info("  • Fetch security alerts (Dependabot, code scanning, secrets)")
		output.Info("  • Collect CI metrics from GoFortress workflow")
		output.Info("  • Create database snapshot (if changes detected)")
		output.Info("")
		output.Success("✓ Dry-run complete. Remove --dry-run flag to perform actual sync.")

		// Mark as processed for dry-run summary
		syncRun.ReposProcessed++
		return nil
	}

	// Collect metadata via Pipeline
	metadata, err := pipeline.SyncRepository(ctx, owner, name)
	if err != nil {
		pipeline.RecordSyncRunError(ctx, syncRun, fullName, err)
		if showProgress {
			output.Error(fmt.Sprintf("✗ %s: failed to collect metadata: %v", fullName, err))
		}
		return fmt.Errorf("failed to sync repository %s: %w", fullName, err)
	}

	// Upsert organization
	owner, _ = parseOwnerAndName(metadata.FullName)
	org := &db.Organization{
		Name:        owner,
		Description: "", // Optional: could fetch org details
	}
	if orgErr := analyticsRepo.UpsertOrganization(ctx, org); orgErr != nil {
		pipeline.RecordSyncRunError(ctx, syncRun, fullName, orgErr)
		if showProgress {
			output.Error(fmt.Sprintf("✗ %s: failed to upsert organization: %v", fullName, orgErr))
		}
		return fmt.Errorf("failed to upsert organization: %w", orgErr)
	}

	// Fetch repo settings via REST (allow_auto_merge, security settings, etc.)
	repoSettings, settingsErr := pipeline.GetGHClient().GetRepository(ctx, fullName)
	if settingsErr != nil {
		output.Warn(fmt.Sprintf("Failed to fetch repo settings for %s: %v", fullName, settingsErr))
	}

	// Upsert repository
	repo := buildRepo(metadata, org.ID)
	applyRepoSettings(repo, repoSettings)
	if repoErr := analyticsRepo.UpsertRepository(ctx, repo); repoErr != nil {
		pipeline.RecordSyncRunError(ctx, syncRun, fullName, repoErr)
		if showProgress {
			output.Error(fmt.Sprintf("✗ %s: failed to upsert repository: %v", fullName, repoErr))
		}
		return fmt.Errorf("failed to upsert repository: %w", repoErr)
	}

	// Build current snapshot
	currentSnapshot := buildRepositorySnapshot(metadata, repo.ID)
	currentSnapshot.Contributors = fetchContributorCount(ctx, pipeline.GetGHClient(), fullName)

	// Check for changes (unless --full flag forces write)
	shouldWrite := forceFull
	if !forceFull {
		latestSnapshot, snapErr := analyticsRepo.GetLatestSnapshot(ctx, repo.ID)
		if snapErr != nil && !errors.Is(snapErr, gorm.ErrRecordNotFound) {
			output.Warn(fmt.Sprintf("Failed to get latest snapshot: %v", snapErr))
		}
		if analytics.HasChanged(currentSnapshot, latestSnapshot) {
			shouldWrite = true
		}
	}

	// Create snapshot if changed
	if shouldWrite {
		if snapErr := analyticsRepo.CreateSnapshot(ctx, currentSnapshot); snapErr != nil {
			pipeline.RecordSyncRunError(ctx, syncRun, fullName, snapErr)
			output.Warn(fmt.Sprintf("Failed to create snapshot: %v", snapErr))
		} else {
			syncRun.SnapshotsCreated++
		}
	} else {
		syncRun.ReposSkipped++
		if showProgress {
			output.Info(fmt.Sprintf("  %s: no changes, snapshot skipped", fullName))
		}
	}

	// Collect and upsert security alerts
	secResult, err := collectSecurityAlerts(ctx, pipeline, analyticsRepo, repo.ID, fullName)
	if err != nil {
		output.Warn(fmt.Sprintf("Failed to collect security alerts for %s: %v", fullName, err))
	}
	if secResult != nil {
		syncRun.AlertsUpserted += secResult.AlertCount
		// Surface warnings to user
		for _, w := range secResult.Warnings {
			output.Warn(fmt.Sprintf("  %s: %s", fullName, w))
		}
	}

	// Update snapshot alert counts from collected data
	if currentSnapshot != nil {
		updateSnapshotAlertCounts(ctx, analyticsRepo, repo.ID, currentSnapshot)
	}

	// Collect and create CI metrics snapshot
	recordCIMetrics(ctx, pipeline, analyticsRepo, syncRun, repo.ID, fullName, showProgress, forceFull)

	syncRun.ReposProcessed++

	// Update repo's last analytics sync timestamp
	if tsErr := analyticsRepo.UpdateRepoSyncTimestamp(ctx, repo.ID, time.Now(), syncRun.ID); tsErr != nil {
		output.Warn(fmt.Sprintf("Failed to update sync timestamp for %s: %v", fullName, tsErr))
	}

	if showProgress {
		totalAlerts := getAlertTotal(ctx, analyticsRepo, repo.ID)
		output.Success(fmt.Sprintf(
			"✓ %s: %d stars, %d forks, %d open issues, %d alerts",
			fullName,
			metadata.Stars,
			metadata.Forks,
			metadata.OpenIssues,
			totalAlerts,
		))
	}

	return nil
}

// recordCIMetrics collects the repository's CI metrics snapshot and counts it on
// the sync run when it reused an unchanged workflow run's metrics. Failures are
// reported as warnings and never fail the repository sync.
func recordCIMetrics(
	ctx context.Context,
	pipeline *analytics.Pipeline,
	analyticsRepo db.AnalyticsRepo,
	syncRun *db.SyncRun,
	repoID uint,
	fullName string,
	showProgress, forceFull bool,
) {
	reused, err := collectCIMetrics(ctx, pipeline, analyticsRepo, repoID, fullName, forceFull)
	if err != nil {
		output.Warn(fmt.Sprintf("Failed to collect CI metrics for %s: %v", fullName, err))
		return
	}
	if !reused {
		return
	}

	syncRun.CISnapshotsReused++
	if showProgress {
		output.Info(fmt.Sprintf("  %s: CI workflow run unchanged, CI metrics reused", fullName))
	}
}

// buildRepo constructs a Repo from metadata
func buildRepo(metadata *analytics.RepoMetadata, orgID uint) *db.Repo {
	// Parse name from FullName (owner/name format)
	_, name := parseOwnerAndName(metadata.FullName)

	// Convert topics slice to JSON string
	var topicsJSON string
	if len(metadata.Topics) > 0 {
		if jsonBytes, err := json.Marshal(metadata.Topics); err == nil {
			topicsJSON = string(jsonBytes)
		}
	}

	return &db.Repo{
		OrganizationID:        orgID,
		Name:                  name,
		FullNameStr:           metadata.FullName,
		Description:           metadata.Description,
		DefaultBranch:         metadata.DefaultBranch,
		Language:              metadata.Language,
		IsPrivate:             metadata.IsPrivate,
		IsFork:                metadata.IsFork,
		ForkParent:            metadata.ForkParent,
		IsArchived:            metadata.IsArchived,
		HTMLURL:               metadata.HTMLURL,
		HomepageURL:           metadata.HomepageURL,
		Topics:                topicsJSON,
		License:               metadata.License,
		DiskUsageKB:           metadata.DiskUsageKB,
		HasIssuesEnabled:      metadata.HasIssuesEnabled,
		HasWikiEnabled:        metadata.HasWikiEnabled,
		HasDiscussionsEnabled: metadata.HasDiscussionsEnabled,
		SSHURL:                metadata.SSHURL,
		CloneURL:              metadata.CloneURL,
		GitHubCreatedAt:       parseTime(metadata.CreatedAt),
		LastPushedAt:          parseTime(metadata.PushedAt),
		GitHubUpdatedAt:       parseTime(metadata.UpdatedAt),
	}
}

// applyRepoSettings copies REST-sourced settings from gh.Repository into the db.Repo.
func applyRepoSettings(repo *db.Repo, settings *gh.Repository) {
	if settings == nil {
		return
	}
	repo.AutoMergeEnabled = settings.AllowAutoMerge
	repo.UpdateBranchEnabled = settings.AllowUpdateBranch
	repo.DependabotEnabled = settings.SecurityAndAnalysis.DependabotSecurityUpdates.Status == "enabled"
	repo.SecretScanningEnabled = settings.SecurityAndAnalysis.SecretScanning.Status == "enabled"
	repo.PushProtectionEnabled = settings.SecurityAndAnalysis.SecretScanningPushProtection.Status == "enabled"

	// Merge strategy settings (backfilled from REST API at zero cost)
	repo.AllowSquashMerge = settings.AllowSquashMerge
	repo.AllowMergeCommit = settings.AllowMergeCommit
	repo.AllowRebaseMerge = settings.AllowRebaseMerge
	repo.DeleteBranchOnMerge = settings.DeleteBranchOnMerge
	repo.SquashMergeCommitTitle = settings.SquashMergeCommitTitle
	repo.SquashMergeCommitMessage = settings.SquashMergeCommitMessage
}

// fetchContributorCount returns the contributor count for a repo, logging warnings on error.
func fetchContributorCount(ctx context.Context, ghClient gh.Client, fullName string) int {
	count, err := ghClient.GetContributorCount(ctx, fullName)
	if err != nil {
		output.Warn(fmt.Sprintf("Failed to fetch contributor count for %s: %v", fullName, err))
		return 0
	}
	return count
}

// parseOwnerAndName splits "owner/name" format into owner and name
func parseOwnerAndName(fullName string) (owner, name string) {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", fullName
}

// buildRepositorySnapshot constructs a RepositorySnapshot from metadata
func buildRepositorySnapshot(metadata *analytics.RepoMetadata, repoID uint) *db.RepositorySnapshot {
	now := time.Now()
	return &db.RepositorySnapshot{
		RepositoryID:    repoID,
		SnapshotAt:      now,
		Stars:           metadata.Stars,
		Forks:           metadata.Forks,
		Watchers:        metadata.Watchers,
		OpenIssues:      metadata.OpenIssues,
		OpenPRs:         metadata.OpenPRs,
		BranchCount:     metadata.BranchCount,
		LatestRelease:   metadata.LatestRelease,
		LatestReleaseAt: parseTimePtr(metadata.LatestReleaseAt),
		LatestTag:       metadata.LatestTag,
		LatestTagAt:     parseTimePtr(metadata.LatestTagAt),
		RepoUpdatedAt:   parseTime(metadata.UpdatedAt),
		PushedAt:        parseTime(metadata.PushedAt),
		// Alert counts will be set later after collecting security alerts
		DependabotAlertCount:     0,
		CodeScanningAlertCount:   0,
		SecretScanningAlertCount: 0,
	}
}

// securityCollectionOutput holds the result of security alert collection for a repo
type securityCollectionOutput struct {
	AlertCount int
	Warnings   []string
}

// collectSecurityAlerts collects and upserts security alerts for a repository.
// Returns the number of alerts upserted and any warnings for display.
func collectSecurityAlerts(
	ctx context.Context,
	pipeline *analytics.Pipeline,
	analyticsRepo db.AnalyticsRepo,
	repoID uint,
	fullName string,
) (*securityCollectionOutput, error) {
	// Use SecurityCollector to fetch alerts
	securityCollector := analytics.NewSecurityCollector(pipeline.GetGHClient(), pipeline.GetLogger(), pipeline.GetThrottle())

	// Create RepoInfo for collector
	parts := strings.Split(fullName, "/")
	repoInfo := gh.RepoInfo{
		FullName: fullName,
		Owner: struct {
			Login string `json:"login"`
		}{Login: parts[0]},
		Name: parts[1],
	}

	resultMap, err := securityCollector.CollectAlerts(ctx, []gh.RepoInfo{repoInfo})
	if err != nil {
		return nil, err
	}

	result, ok := resultMap[fullName]
	// Always initialize out, even when no open alerts were returned.
	// If the repo has 0 open alerts on GitHub it won't appear in resultMap, but
	// we still need to call CloseStaleAlerts below to clear any stale DB records.
	out := &securityCollectionOutput{}
	if ok {
		out.Warnings = result.Warnings
	}

	// Upsert each currently-open alert and collect their numbers
	var currentAlertNumbers []int
	for _, alert := range result.Alerts {
		dbAlert := convertSecurityAlert(alert, repoID)
		if upsertErr := analyticsRepo.UpsertAlert(ctx, dbAlert); upsertErr != nil {
			return out, fmt.Errorf("failed to upsert alert: %w", upsertErr)
		}
		out.AlertCount++
		currentAlertNumbers = append(currentAlertNumbers, alert.AlertNumber)
	}

	// Reconcile: mark stale open alerts as auto_resolved
	resolved, resolveErr := analyticsRepo.CloseStaleAlerts(ctx, repoID, currentAlertNumbers)
	if resolveErr != nil {
		return out, fmt.Errorf("failed to close stale alerts: %w", resolveErr)
	}
	if resolved > 0 {
		out.Warnings = append(out.Warnings, fmt.Sprintf("auto-resolved %d stale alert(s) no longer open on GitHub", resolved))
	}

	return out, nil
}

// updateSnapshotAlertCounts queries the DB for current alert counts and updates the snapshot.
// If the snapshot was never persisted (skipped due to no changes), it fetches the latest
// persisted snapshot for this repo and updates that instead.
func updateSnapshotAlertCounts(
	ctx context.Context,
	analyticsRepo db.AnalyticsRepo,
	repoID uint,
	snapshot *db.RepositorySnapshot,
) {
	counts, err := analyticsRepo.GetAlertCountsByType(ctx, repoID)
	if err != nil || len(counts) == 0 {
		return // No alerts to update
	}

	// If the snapshot wasn't persisted (ID=0, e.g. skipped for no changes),
	// fetch the latest persisted snapshot to update its counts instead
	target := snapshot
	if snapshot.ID == 0 {
		latest, latestErr := analyticsRepo.GetLatestSnapshot(ctx, repoID)
		if latestErr != nil || latest == nil {
			return // No snapshot to update
		}
		target = latest
	}

	target.DependabotAlertCount = counts["dependabot"]
	target.CodeScanningAlertCount = counts["code_scanning"]
	target.SecretScanningAlertCount = counts["secret_scanning"]

	if err := analyticsRepo.UpdateSnapshotAlertCounts(ctx, target); err != nil {
		output.Warn(fmt.Sprintf("Failed to update snapshot alert counts: %v", err))
	}
}

// getAlertTotal returns the total open alert count for a repository from the DB.
// Used for display purposes to ensure accurate counts regardless of snapshot state.
func getAlertTotal(ctx context.Context, analyticsRepo db.AnalyticsRepo, repoID uint) int {
	counts, err := analyticsRepo.GetAlertCountsByType(ctx, repoID)
	if err != nil {
		return 0
	}
	total := 0
	for _, c := range counts {
		total += c
	}
	return total
}

// convertSecurityAlert converts analytics.SecurityAlert to db.SecurityAlert
func convertSecurityAlert(alert analytics.SecurityAlert, repoID uint) *db.SecurityAlert {
	var createdAt time.Time
	if t := parseTime(alert.CreatedAt); t != nil {
		createdAt = *t
	}
	return &db.SecurityAlert{
		RepositoryID:   repoID,
		AlertType:      string(alert.AlertType),
		AlertNumber:    alert.AlertNumber,
		State:          alert.State,
		Severity:       alert.Severity,
		Summary:        alert.Title,
		HTMLURL:        alert.HTMLURL,
		AlertCreatedAt: createdAt,
		DismissedAt:    parseTimePtr(alert.DismissedAt),
		FixedAt:        parseTimePtr(alert.FixedAt),
	}
}

// collectCIMetrics collects and creates CI metrics snapshot for a repository.
// Unless forceFull is set, an unchanged newest workflow run is recorded from the
// latest snapshot instead of downloading its artifacts again; reused reports that.
func collectCIMetrics(
	ctx context.Context,
	pipeline *analytics.Pipeline,
	analyticsRepo db.AnalyticsRepo,
	repoID uint,
	fullName string,
	forceFull bool,
) (reused bool, err error) {
	// Use CICollector to fetch metrics
	ciCollector := analytics.NewCICollector(pipeline.GetGHClient(), pipeline.GetLogger(), pipeline.GetThrottle())

	parts := strings.Split(fullName, "/")
	repoInfo := gh.RepoInfo{
		FullName: fullName,
		Owner: struct {
			Login string `json:"login"`
		}{Login: parts[0]},
		Name: parts[1],
	}

	// Offer the latest snapshot as a baseline, so an unchanged newest workflow run is
	// recorded without downloading its artifacts again. --full always re-downloads.
	var baselines map[string]*analytics.CIMetrics
	if !forceFull {
		latest, snapErr := analyticsRepo.GetLatestCISnapshot(ctx, repoID)
		switch {
		case snapErr == nil:
			baselines = map[string]*analytics.CIMetrics{fullName: ciMetricsFromSnapshot(latest)}
		case !errors.Is(snapErr, gorm.ErrRecordNotFound):
			output.Warn(fmt.Sprintf("Failed to get latest CI snapshot for %s: %v", fullName, snapErr))
		}
	}

	metricsMap, err := ciCollector.CollectCIMetricsWithBaselines(ctx, []gh.RepoInfo{repoInfo}, baselines)
	if err != nil {
		return false, err
	}

	metrics, ok := metricsMap[fullName]
	if !ok || metrics == nil {
		return false, nil // No CI metrics available
	}

	// Create CI snapshot
	snapshot := &db.CIMetricsSnapshot{
		RepositoryID:       repoID,
		SnapshotAt:         time.Now(),
		WorkflowRunID:      metrics.WorkflowRunID,
		WorkflowRunAttempt: metrics.WorkflowRunAttempt,
		Branch:             metrics.Branch,
		CommitSHA:          metrics.CommitSHA,
		ArtifactsComplete:  metrics.ArtifactsComplete,
		Reused:             metrics.Reused,
		GoFilesLOC:         metrics.GoFilesLOC,
		TestFilesLOC:       metrics.TestFilesLOC,
		GoFilesCount:       metrics.GoFilesCount,
		TestFilesCount:     metrics.TestFilesCount,
		TestCount:          metrics.TestCount,
		BenchmarkCount:     metrics.BenchmarkCount,
		CoveragePercent:    metrics.Coverage,
	}

	if createErr := analyticsRepo.CreateCISnapshot(ctx, snapshot); createErr != nil {
		return false, createErr
	}
	return metrics.Reused, nil
}

// ciMetricsFromSnapshot converts a stored CI snapshot back into collector metrics,
// for use as the baseline of the next collection.
func ciMetricsFromSnapshot(snapshot *db.CIMetricsSnapshot) *analytics.CIMetrics {
	return &analytics.CIMetrics{
		WorkflowRunID:      snapshot.WorkflowRunID,
		WorkflowRunAttempt: snapshot.WorkflowRunAttempt,
		Branch:             snapshot.Branch,
		CommitSHA:          snapshot.CommitSHA,
		GoFilesLOC:         snapshot.GoFilesLOC,
		TestFilesLOC:       snapshot.TestFilesLOC,
		GoFilesCount:       snapshot.GoFilesCount,
		TestFilesCount:     snapshot.TestFilesCount,
		TestCount:          snapshot.TestCount,
		BenchmarkCount:     snapshot.BenchmarkCount,
		Coverage:           snapshot.CoveragePercent,
		ArtifactsComplete:  snapshot.ArtifactsComplete,
	}
}

// parseTime parses ISO 8601 timestamp string to *time.Time
func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		return nil
	}
	return &t
}

// parseTimePtr parses *string timestamp to *time.Time
func parseTimePtr(s *string) *time.Time {
	if s == nil {
		return nil
	}
	return parseTime(*s)
}

// syncOrganization syncs all repositories in an organization
func syncOrganization(
	ctx context.Context,
	pipeline *analytics.Pipeline,
	analyticsRepo db.AnalyticsRepo,
	syncRun *db.SyncRun,
	org string,
	showProgress, isDryRun, forceFull bool,
) error {
	if showProgress {
		output.Info(fmt.Sprintf("Starting sync for owner: %s", org))
	}

	// Discover repos via Pipeline
	metadata, err := pipeline.SyncOrganization(ctx, org)
	if err != nil {
		return fmt.Errorf("failed to sync owner %s: %w", org, err)
	}

	if showProgress {
		output.Info(fmt.Sprintf("Discovered %d repositories for %s", len(metadata), org))
	}

	if isDryRun {
		output.Info(fmt.Sprintf("[DRY RUN] Would sync %d repositories", len(metadata)))
		output.Info("")
		output.Info("  Actions that would be performed for each repository:")
		output.Info("  • Collect repository metadata via GraphQL")
		output.Info("  • Fetch security alerts concurrently")
		output.Info("  • Collect CI metrics from workflows")
		output.Info("  • Create database snapshots (with change detection)")
		output.Info("")
		output.Success(fmt.Sprintf("✓ Dry-run complete. Would process %d repositories. Remove --dry-run to sync.", len(metadata)))

		// Mark repos as processed for dry-run summary
		syncRun.ReposProcessed = len(metadata)
		return nil
	}

	// Process each repository
	repoIndex := 0
	for fullName, meta := range metadata {
		parts := strings.Split(fullName, "/")
		if len(parts) != 2 {
			continue
		}

		if err := syncRepositoryMetadata(ctx, pipeline, analyticsRepo, syncRun, meta, showProgress, forceFull); err != nil {
			output.Warn(fmt.Sprintf("Failed to sync %s: %v", fullName, err))
			continue
		}

		// Inter-repo delay to avoid rate-limit pressure
		repoIndex++
		if throttle := pipeline.GetThrottle(); throttle != nil && repoIndex < len(metadata) {
			_ = throttle.WaitInterRepo(ctx)
		}
	}

	return nil
}

// syncAllOrganizations syncs all organizations in the database
func syncAllOrganizations(
	ctx context.Context,
	pipeline *analytics.Pipeline,
	analyticsRepo db.AnalyticsRepo,
	syncRun *db.SyncRun,
	showProgress, isDryRun, forceFull bool,
) error {
	orgs, err := analyticsRepo.ListOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("failed to list organizations: %w", err)
	}

	if len(orgs) == 0 {
		return fmt.Errorf("no organizations found in database (import configuration first)") //nolint:err113 // user-facing CLI error
	}

	if showProgress {
		output.Info(fmt.Sprintf("Starting full sync for %d organizations", len(orgs)))
	}

	if isDryRun {
		output.Info(fmt.Sprintf("[DRY RUN] Would sync %d organizations", len(orgs)))
		for _, org := range orgs {
			output.Info(fmt.Sprintf("  • %s", org.Name))
		}
		output.Info("")
		output.Info("  Actions that would be performed:")
		output.Info("  • Discover all repositories in each organization")
		output.Info("  • Collect metadata via batched GraphQL queries")
		output.Info("  • Fetch security alerts concurrently")
		output.Info("  • Collect CI metrics from workflows")
		output.Info("  • Create database snapshots (with change detection)")
		output.Info("")
		output.Success(fmt.Sprintf("✓ Dry-run complete. Would process %d organizations. Remove --dry-run to sync.", len(orgs)))

		// Mark orgs as processed for dry-run summary
		syncRun.ReposProcessed = len(orgs)
		return nil
	}

	// Sync each organization
	for _, org := range orgs {
		if err := syncOrganization(ctx, pipeline, analyticsRepo, syncRun, org.Name, showProgress, false, forceFull); err != nil {
			output.Warn(fmt.Sprintf("Failed to sync organization %s: %v", org.Name, err))
			continue
		}
	}

	return nil
}

// syncManagedRepos syncs only the repositories managed by enabled sync groups.
// It reads the repo list from DB targets (populated via `db import` from sync.yaml)
// rather than discovering all repos in each organization via GitHub API.
// This ensures analytics data covers exactly the set of managed repos.
func syncManagedRepos(
	ctx context.Context,
	gormDB *gorm.DB,
	pipeline *analytics.Pipeline,
	analyticsRepo db.AnalyticsRepo,
	syncRun *db.SyncRun,
	showProgress, isDryRun, forceFull bool,
) error {
	// Query unique managed repos from enabled groups
	type managedRepo struct {
		OrgName  string
		RepoName string
	}

	var managedRepos []managedRepo

	result := gormDB.Raw(`
		SELECT DISTINCT o.name AS org_name, r.name AS repo_name
		FROM targets t
		JOIN repos r ON t.repo_id = r.id
		JOIN organizations o ON r.organization_id = o.id
		JOIN groups g ON t.group_id = g.id
		WHERE g.enabled = 1
		ORDER BY o.name, r.name
	`).Scan(&managedRepos)
	if result.Error != nil {
		return fmt.Errorf("failed to query managed repos: %w", result.Error)
	}

	if len(managedRepos) == 0 {
		return fmt.Errorf("no managed repos found in database (run `go-broadcast db import` to import configuration)") //nolint:err113 // user-facing CLI error
	}

	if showProgress {
		output.Info(fmt.Sprintf("Syncing %d managed repositories from database configuration", len(managedRepos)))
	}

	if isDryRun {
		output.Info(fmt.Sprintf("[DRY RUN] Would sync %d managed repositories:", len(managedRepos)))
		for _, r := range managedRepos {
			output.Info(fmt.Sprintf("  • %s/%s", r.OrgName, r.RepoName))
		}
		output.Info("")
		output.Info("  Actions that would be performed for each repository:")
		output.Info("  • Collect repository metadata via GraphQL")
		output.Info("  • Fetch security alerts concurrently")
		output.Info("  • Collect CI metrics from workflows")
		output.Info("  • Create database snapshots (with change detection)")
		output.Info("")
		output.Success(fmt.Sprintf("✓ Dry-run complete. Would process %d repositories. Remove --dry-run to sync.", len(managedRepos)))

		syncRun.ReposProcessed = len(managedRepos)
		return nil
	}

	// Sync each managed repository
	for i, r := range managedRepos {
		fullName := fmt.Sprintf("%s/%s", r.OrgName, r.RepoName)

		meta, err := pipeline.SyncRepository(ctx, r.OrgName, r.RepoName)
		if err != nil {
			output.Warn(fmt.Sprintf("Failed to collect metadata for %s: %v", fullName, err))
			pipeline.RecordSyncRunError(ctx, syncRun, fullName, err)
			continue
		}

		if err := syncRepositoryMetadata(ctx, pipeline, analyticsRepo, syncRun, meta, showProgress, forceFull); err != nil {
			output.Warn(fmt.Sprintf("Failed to store metadata for %s: %v", fullName, err))
			continue
		}

		// Inter-repo delay to avoid rate-limit pressure
		if throttle := pipeline.GetThrottle(); throttle != nil && i < len(managedRepos)-1 {
			_ = throttle.WaitInterRepo(ctx)
		}
	}

	return nil
}

// syncRepositoryMetadata is a helper for org/full sync modes
func syncRepositoryMetadata(
	ctx context.Context,
	pipeline *analytics.Pipeline,
	analyticsRepo db.AnalyticsRepo,
	syncRun *db.SyncRun,
	metadata *analytics.RepoMetadata,
	showProgress, forceFull bool,
) error {
	fullName := metadata.FullName

	// Upsert organization
	orgOwner, _ := parseOwnerAndName(metadata.FullName)
	org := &db.Organization{
		Name:        orgOwner,
		Description: "",
	}
	if err := analyticsRepo.UpsertOrganization(ctx, org); err != nil {
		pipeline.RecordSyncRunError(ctx, syncRun, fullName, err)
		if showProgress {
			output.Error(fmt.Sprintf("✗ %s: failed to upsert organization: %v", fullName, err))
		}
		return fmt.Errorf("failed to upsert organization: %w", err)
	}

	// Fetch repo settings via REST (allow_auto_merge, security settings, etc.)
	repoSettings, settingsErr := pipeline.GetGHClient().GetRepository(ctx, fullName)
	if settingsErr != nil {
		output.Warn(fmt.Sprintf("Failed to fetch repo settings for %s: %v", fullName, settingsErr))
	}

	// Upsert repository
	repo := buildRepo(metadata, org.ID)
	applyRepoSettings(repo, repoSettings)
	if err := analyticsRepo.UpsertRepository(ctx, repo); err != nil {
		pipeline.RecordSyncRunError(ctx, syncRun, fullName, err)
		if showProgress {
			output.Error(fmt.Sprintf("✗ %s: failed to upsert repository: %v", fullName, err))
		}
		return fmt.Errorf("failed to upsert repository: %w", err)
	}

	// Build current snapshot
	currentSnapshot := buildRepositorySnapshot(metadata, repo.ID)
	currentSnapshot.Contributors = fetchContributorCount(ctx, pipeline.GetGHClient(), fullName)

	// Check for changes (unless --full flag forces write)
	shouldWrite := forceFull
	if !forceFull {
		latestSnapshot, snapErr := analyticsRepo.GetLatestSnapshot(ctx, repo.ID)
		if snapErr != nil && !errors.Is(snapErr, gorm.ErrRecordNotFound) {
			output.Warn(fmt.Sprintf("Failed to get latest snapshot: %v", snapErr))
		}
		if analytics.HasChanged(currentSnapshot, latestSnapshot) {
			shouldWrite = true
		}
	}

	// Create snapshot if changed
	if shouldWrite {
		if snapErr := analyticsRepo.CreateSnapshot(ctx, currentSnapshot); snapErr != nil {
			pipeline.RecordSyncRunError(ctx, syncRun, fullName, snapErr)
			output.Warn(fmt.Sprintf("Failed to create snapshot: %v", snapErr))
		} else {
			syncRun.SnapshotsCreated++
		}
	} else {
		syncRun.ReposSkipped++
		if showProgress {
			output.Info(fmt.Sprintf("  %s: no changes, snapshot skipped", fullName))
		}
	}

	// Collect and upsert security alerts
	secResult, secErr := collectSecurityAlerts(ctx, pipeline, analyticsRepo, repo.ID, fullName)
	if secErr != nil {
		output.Warn(fmt.Sprintf("Failed to collect security alerts for %s: %v", fullName, secErr))
	}
	if secResult != nil {
		syncRun.AlertsUpserted += secResult.AlertCount
		for _, w := range secResult.Warnings {
			output.Warn(fmt.Sprintf("  %s: %s", fullName, w))
		}
	}

	// Update snapshot alert counts from collected data
	if currentSnapshot != nil {
		updateSnapshotAlertCounts(ctx, analyticsRepo, repo.ID, currentSnapshot)
	}

	// Collect and create CI metrics snapshot
	recordCIMetrics(ctx, pipeline, analyticsRepo, syncRun, repo.ID, fullName, showProgress, forceFull)

	syncRun.ReposProcessed++

	// Update repo's last analytics sync timestamp
	if tsErr := analyticsRepo.UpdateRepoSyncTimestamp(ctx, repo.ID, time.Now(), syncRun.ID); tsErr != nil {
		output.Warn(fmt.Sprintf("Failed to update sync timestamp for %s: %v", fullName, tsErr))
	}

	if showProgress {
		totalAlerts := getAlertTotal(ctx, analyticsRepo, repo.ID)
		output.Success(fmt.Sprintf(
			"✓ %s: %d stars, %d forks, %d open issues, %d alerts",
			fullName,
			metadata.Stars,
			metadata.Forks,
			metadata.OpenIssues,
			totalAlerts,
		))
	}

	return nil
}

// measureRunAPIUsage returns the GitHub API usage since meter started, or nil when
// it cannot be measured (no meter, or the rate limit could not be read). It
// measures even after ctx is canceled, so an interrupted run is still recorded.
func measureRunAPIUsage(ctx context.Context, meter *gh.UsageMeter, logger *logrus.Logger) *gh.APIUsage {
	if meter == nil {
		return nil
	}

	usage, err := meter.MeasureDetached(ctx)
	if err != nil {
		if logger != nil {
			logger.WithError(err).Debug("GitHub API usage could not be measured")
		}
		return nil
	}
	return &usage
}

// applyAPIUsage copies measured API usage onto the sync run record. Unmeasured
// usage (nil) leaves the run's counts at zero.
func applyAPIUsage(syncRun *db.SyncRun, usage *gh.APIUsage) {
	if usage == nil {
		return
	}
	syncRun.APICallsMade = usage.CoreRequests
	syncRun.GraphQLPointsUsed = usage.GraphQLPoints
	syncRun.APIUsageLowerBound = usage.CoreLowerBound || usage.GraphQLLowerBound
}

// displaySyncSummary displays a user-friendly summary of the sync operation.
// apiUsage is the run's measured GitHub API usage, or nil when it was not measured.
func displaySyncSummary(syncRun *db.SyncRun, status string, throttleStats *analytics.ThrottleStats, apiUsage *gh.APIUsage) {
	// Don't show summary for dry-run (already shown inline)
	if syncRun.SyncType == "full" && syncRun.ReposProcessed > 0 && syncRun.SnapshotsCreated == 0 && syncRun.AlertsUpserted == 0 && syncRun.DurationMs < 10 {
		// This looks like a dry-run, skip summary
		return
	}

	output.Plain("\n" + strings.Repeat("─", 60))
	output.Info("Sync Summary")
	output.Plain(strings.Repeat("─", 60))

	// Status with color
	switch status {
	case "completed":
		output.Success(fmt.Sprintf("Status: %s", status))
	case "partial":
		output.Warn(fmt.Sprintf("Status: %s (some repositories failed)", status))
	case "failed":
		output.Error(fmt.Sprintf("Status: %s", status))
	default:
		output.Info(fmt.Sprintf("Status: %s", status))
	}

	// Metrics
	output.Info(fmt.Sprintf("Repositories Processed: %d", syncRun.ReposProcessed))
	output.Info(fmt.Sprintf("Repositories Skipped: %d (no changes)", syncRun.ReposSkipped))
	if syncRun.ReposFailed > 0 {
		output.Warn(fmt.Sprintf("Repositories Failed: %d", syncRun.ReposFailed))
	}
	output.Info(fmt.Sprintf("Snapshots Created: %d", syncRun.SnapshotsCreated))
	if syncRun.CISnapshotsReused > 0 {
		output.Info(fmt.Sprintf("CI Snapshots Reused: %d (newest workflow run unchanged)", syncRun.CISnapshotsReused))
	}
	output.Info(fmt.Sprintf("Security Alerts Upserted: %d", syncRun.AlertsUpserted))

	// Duration
	if syncRun.DurationMs > 0 {
		duration := time.Duration(syncRun.DurationMs) * time.Millisecond
		output.Info(fmt.Sprintf("Duration: %s", duration.Round(time.Millisecond)))
	}

	// GitHub API usage and throttle stats
	switch {
	case apiUsage != nil:
		output.Info(fmt.Sprintf("GitHub API Usage: %s", apiUsage))
		output.Info(fmt.Sprintf("REST Budget Remaining: %d/%d (resets at %s)",
			apiUsage.CoreRemaining, apiUsage.CoreLimit, apiUsage.CoreResetAt.Format("15:04:05")))
	case throttleStats != nil && throttleStats.TotalCalls > 0:
		// Without a rate-limit reading, report the operations the throttle admitted.
		// One operation can make several requests (pagination, artifact downloads).
		output.Info(fmt.Sprintf("GitHub API Calls: %d throttled operations (rate limit unavailable, usage not measured)",
			throttleStats.TotalCalls))
	}
	if throttleStats != nil {
		if throttleStats.TotalRetries > 0 {
			output.Warn(fmt.Sprintf("Rate-Limit Retries: %d", throttleStats.TotalRetries))
		}
		if throttleStats.TotalWaitedMs > 0 {
			waitDuration := time.Duration(throttleStats.TotalWaitedMs) * time.Millisecond
			output.Info(fmt.Sprintf("Throttle Wait Time: %s", waitDuration.Round(time.Millisecond)))
		}
	}

	output.Plain(strings.Repeat("─", 60))
}
