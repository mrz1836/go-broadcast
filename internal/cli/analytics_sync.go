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
		return "", "", fmt.Errorf("invalid repository format %q, expected owner/name (e.g., mrz1836/go-broadcast)", fullName) //nolint:err113 // user-facing CLI error
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

			// Pre-flight rate limit check
			if showProgress := progress; showProgress {
				rateLimitInfo, rateLimitErr := analytics.CheckRateLimit(ctx, ghClient)
				if rateLimitErr != nil {
					output.Warn(fmt.Sprintf("Could not check rate limit: %v", rateLimitErr))
				} else {
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
			} else {
				// Full sync (all organizations)
				syncErr = syncAllOrganizations(ctx, pipeline, analyticsRepo, syncRun, progress, dryRun, full)
			}

			// Determine final status
			status := "completed"
			if syncRun.ReposFailed > 0 {
				status = "partial"
			}
			if syncErr != nil {
				status = "failed"
			}

			// Complete sync run
			if completeErr := pipeline.CompleteSyncRun(ctx, syncRun, status); completeErr != nil {
				output.Warn(fmt.Sprintf("Failed to update sync run: %v", completeErr))
			}

			// Update API call count from throttle stats
			throttleStats := throttle.Stats()
			syncRun.APICallsMade = int(throttleStats.TotalCalls)

			// Display summary
			displaySyncSummary(syncRun, status, &throttleStats)

			// Return original sync error if any
			return syncErr
		},
	}

	cmd.Flags().StringVar(&org, "org", "", "Sync specific owner (organization or user account)")
	cmd.Flags().StringVar(&repo, "repo", "", "Sync specific repository only (owner/name)")
	cmd.Flags().BoolVar(&securityOnly, "security-only", false, "Sync security alerts only")
	cmd.Flags().BoolVar(&full, "full", false, "Force full sync (ignore change detection)")
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

	// Upsert repository
	repo := buildAnalyticsRepository(metadata, org.ID)
	if repoErr := analyticsRepo.UpsertRepository(ctx, repo); repoErr != nil {
		pipeline.RecordSyncRunError(ctx, syncRun, fullName, repoErr)
		if showProgress {
			output.Error(fmt.Sprintf("✗ %s: failed to upsert repository: %v", fullName, repoErr))
		}
		return fmt.Errorf("failed to upsert repository: %w", repoErr)
	}

	// Update config repos table with all GitHub metadata
	updateConfigRepoFromGitHub(ctx, analyticsRepo, fullName, metadata)

	// Build current snapshot
	currentSnapshot := buildRepositorySnapshot(metadata, repo.ID)

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
	alertCount, err := collectSecurityAlerts(ctx, pipeline, analyticsRepo, repo.ID, fullName)
	if err != nil {
		output.Warn(fmt.Sprintf("Failed to collect security alerts for %s: %v", fullName, err))
	} else {
		syncRun.AlertsUpserted += alertCount
	}

	// Collect and create CI metrics snapshot
	if err := collectCIMetrics(ctx, pipeline, analyticsRepo, repo.ID, fullName); err != nil {
		output.Warn(fmt.Sprintf("Failed to collect CI metrics for %s: %v", fullName, err))
	}

	syncRun.ReposProcessed++
	// API call count is tracked by the shared throttle

	if showProgress {
		totalAlerts := currentSnapshot.DependabotAlertCount + currentSnapshot.CodeScanningAlertCount + currentSnapshot.SecretScanningAlertCount
		output.Success(fmt.Sprintf("✓ %s: %d stars, %d forks, %d open issues, %d alerts",
			fullName,
			metadata.Stars,
			metadata.Forks,
			metadata.OpenIssues,
			totalAlerts,
		))
	}

	return nil
}

// buildAnalyticsRepository constructs an AnalyticsRepository from metadata
func buildAnalyticsRepository(metadata *analytics.RepoMetadata, orgID uint) *db.AnalyticsRepository {
	// Parse owner from FullName (owner/name format)
	owner, name := parseOwnerAndName(metadata.FullName)

	// Convert topics slice to JSON string
	var topicsJSON string
	if len(metadata.Topics) > 0 {
		if jsonBytes, err := json.Marshal(metadata.Topics); err == nil {
			topicsJSON = string(jsonBytes)
		}
	}

	return &db.AnalyticsRepository{
		OrganizationID: orgID,
		Owner:          owner,
		Name:           name,
		FullName:       metadata.FullName,
		Description:    metadata.Description,
		DefaultBranch:  metadata.DefaultBranch,
		Language:       metadata.Language,
		IsPrivate:      metadata.IsPrivate,
		IsFork:         metadata.IsFork,
		ForkSource:     metadata.ForkParent,
		IsArchived:     metadata.IsArchived,
		URL:            metadata.HTMLURL,

		// Enhanced metadata fields
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

// collectSecurityAlerts collects and upserts security alerts for a repository
func collectSecurityAlerts(
	ctx context.Context,
	pipeline *analytics.Pipeline,
	analyticsRepo db.AnalyticsRepo,
	repoID uint,
	fullName string,
) (int, error) {
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

	alertMap, err := securityCollector.CollectAlerts(ctx, []gh.RepoInfo{repoInfo})
	if err != nil {
		return 0, err
	}

	alerts, ok := alertMap[fullName]
	if !ok || len(alerts) == 0 {
		return 0, nil
	}

	// Upsert each alert
	for _, alert := range alerts {
		dbAlert := convertSecurityAlert(alert, repoID)
		if upsertErr := analyticsRepo.UpsertAlert(ctx, dbAlert); upsertErr != nil {
			return 0, fmt.Errorf("failed to upsert alert: %w", upsertErr)
		}
	}

	return len(alerts), nil
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

// collectCIMetrics collects and creates CI metrics snapshot for a repository
func collectCIMetrics(
	ctx context.Context,
	pipeline *analytics.Pipeline,
	analyticsRepo db.AnalyticsRepo,
	repoID uint,
	fullName string,
) error {
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

	metricsMap, err := ciCollector.CollectCIMetrics(ctx, []gh.RepoInfo{repoInfo})
	if err != nil {
		return err
	}

	metrics, ok := metricsMap[fullName]
	if !ok || metrics == nil {
		return nil // No CI metrics available
	}

	// Create CI snapshot
	snapshot := &db.CIMetricsSnapshot{
		RepositoryID:    repoID,
		SnapshotAt:      time.Now(),
		WorkflowRunID:   metrics.WorkflowRunID,
		Branch:          metrics.Branch,
		CommitSHA:       metrics.CommitSHA,
		GoFilesLOC:      metrics.GoFilesLOC,
		TestFilesLOC:    metrics.TestFilesLOC,
		GoFilesCount:    metrics.GoFilesCount,
		TestFilesCount:  metrics.TestFilesCount,
		TestCount:       metrics.TestCount,
		BenchmarkCount:  metrics.BenchmarkCount,
		CoveragePercent: metrics.Coverage,
	}

	return analyticsRepo.CreateCISnapshot(ctx, snapshot)
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

	// Upsert repository
	repo := buildAnalyticsRepository(metadata, org.ID)
	if err := analyticsRepo.UpsertRepository(ctx, repo); err != nil {
		pipeline.RecordSyncRunError(ctx, syncRun, fullName, err)
		if showProgress {
			output.Error(fmt.Sprintf("✗ %s: failed to upsert repository: %v", fullName, err))
		}
		return fmt.Errorf("failed to upsert repository: %w", err)
	}

	// Build current snapshot
	currentSnapshot := buildRepositorySnapshot(metadata, repo.ID)

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
	alertCount, err := collectSecurityAlerts(ctx, pipeline, analyticsRepo, repo.ID, fullName)
	if err != nil {
		output.Warn(fmt.Sprintf("Failed to collect security alerts for %s: %v", fullName, err))
	} else {
		syncRun.AlertsUpserted += alertCount
	}

	// Collect and create CI metrics snapshot
	if err := collectCIMetrics(ctx, pipeline, analyticsRepo, repo.ID, fullName); err != nil {
		output.Warn(fmt.Sprintf("Failed to collect CI metrics for %s: %v", fullName, err))
	}

	syncRun.ReposProcessed++
	// API call count is tracked by the shared throttle

	if showProgress {
		totalAlerts := currentSnapshot.DependabotAlertCount + currentSnapshot.CodeScanningAlertCount + currentSnapshot.SecretScanningAlertCount
		output.Success(fmt.Sprintf("✓ %s: %d stars, %d forks, %d open issues, %d alerts",
			fullName,
			metadata.Stars,
			metadata.Forks,
			metadata.OpenIssues,
			totalAlerts,
		))
	}

	return nil
}

// displaySyncSummary displays a user-friendly summary of the sync operation
func displaySyncSummary(syncRun *db.SyncRun, status string, throttleStats *analytics.ThrottleStats) {
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
	output.Info(fmt.Sprintf("Security Alerts Upserted: %d", syncRun.AlertsUpserted))

	// Duration
	if syncRun.DurationMs > 0 {
		duration := time.Duration(syncRun.DurationMs) * time.Millisecond
		output.Info(fmt.Sprintf("Duration: %s", duration.Round(time.Millisecond)))
	}

	// API calls and throttle stats
	if syncRun.APICallsMade > 0 {
		output.Info(fmt.Sprintf("GitHub API Calls: %d", syncRun.APICallsMade))
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

// updateConfigRepoFromGitHub updates the config repos table with data from GitHub API
// This keeps the config repos table in sync with actual GitHub repo state
func updateConfigRepoFromGitHub(ctx context.Context, analyticsRepo db.AnalyticsRepo, fullName string, metadata *analytics.RepoMetadata) {
	// Get the GORM DB from the analytics repo interface
	// We need to access the underlying database to update the config repos table
	type gormGetter interface {
		GetDB() *gorm.DB
	}

	getter, ok := analyticsRepo.(gormGetter)
	if !ok {
		// Can't get DB, skip silently
		return
	}

	database := getter.GetDB()

	// Convert topics slice to JSON string
	var topicsJSON string
	if len(metadata.Topics) > 0 {
		if jsonBytes, err := json.Marshal(metadata.Topics); err == nil {
			topicsJSON = string(jsonBytes)
		}
	}

	// Update config repos table with all GitHub metadata
	result := database.WithContext(ctx).
		Exec(`
			UPDATE repos
			SET
				is_private = ?,
				is_archived = ?,
				default_branch = ?,
				language = ?,
				homepage_url = ?,
				topics = ?,
				license = ?,
				disk_usage_kb = ?,
				is_fork = ?,
				fork_parent = ?,
				has_issues_enabled = ?,
				has_wiki_enabled = ?,
				has_discussions_enabled = ?,
				html_url = ?,
				ssh_url = ?,
				clone_url = ?,
				github_created_at = ?,
				last_pushed_at = ?,
				github_updated_at = ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE id IN (
				SELECT r.id
				FROM repos r
				JOIN organizations o ON r.organization_id = o.id
				WHERE (o.name || '/' || r.name) = ?
			)
		`,
			metadata.IsPrivate,
			metadata.IsArchived,
			metadata.DefaultBranch,
			metadata.Language,
			metadata.HomepageURL,
			topicsJSON,
			metadata.License,
			metadata.DiskUsageKB,
			metadata.IsFork,
			metadata.ForkParent,
			metadata.HasIssuesEnabled,
			metadata.HasWikiEnabled,
			metadata.HasDiscussionsEnabled,
			metadata.HTMLURL,
			metadata.SSHURL,
			metadata.CloneURL,
			parseTime(metadata.CreatedAt),
			parseTime(metadata.PushedAt),
			parseTime(metadata.UpdatedAt),
			fullName,
		)

	if result.Error != nil {
		// Log warning but don't fail the sync
		output.Warn(fmt.Sprintf("Failed to update config repo from GitHub for %s: %v", fullName, result.Error))
	}
}
