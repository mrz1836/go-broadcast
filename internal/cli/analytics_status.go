package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// repoStatus holds aggregated status for a single repository
type repoStatus struct {
	FullName    string
	Stars       int
	Forks       int
	OpenIssues  int
	OpenPRs     int
	TotalAlerts int
	LastSyncAt  *time.Time
}

// newAnalyticsStatusCmd creates the analytics status command
func newAnalyticsStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [repo]",
		Short: "Show analytics status for repositories",
		Long: `Display current metrics, security alerts, and sync status for repositories.
Optionally specify a specific repo (owner/name) or show all repos.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runAnalyticsStatus,
	}

	return cmd
}

// runAnalyticsStatus implements the analytics status command
func runAnalyticsStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Open database
	database, err := openDatabase()
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	analyticsRepo := db.NewAnalyticsRepo(database.DB())

	// Mode detection: all repos vs single repo
	if len(args) == 0 {
		return displayAllRepositories(ctx, analyticsRepo)
	}

	// Parse and validate repo name
	fullName := args[0]
	if _, _, err := parseRepoName(fullName); err != nil {
		return err
	}

	return displaySingleRepository(ctx, analyticsRepo, fullName)
}

// displayAllRepositories shows a table of all repositories with their metrics
func displayAllRepositories(ctx context.Context, analyticsRepo db.AnalyticsRepo) error {
	// Query all repositories
	repos, err := analyticsRepo.ListRepositories(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(repos) == 0 {
		output.Warn("No repositories found. Run 'analytics sync' first.")
		return nil
	}

	// Collect status for each repo
	var statuses []repoStatus
	totalAlerts := 0
	reposWithAlerts := 0

	for _, repo := range repos {
		status := repoStatus{
			FullName:   repo.FullName(),
			LastSyncAt: repo.LastSyncAt,
		}

		// Get latest snapshot for metrics
		snapshot, snapshotErr := analyticsRepo.GetLatestSnapshot(ctx, repo.ID)
		if snapshotErr != nil && !errors.Is(snapshotErr, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to get snapshot for %s: %w", repo.FullName(), snapshotErr)
		}
		if snapshot != nil {
			status.Stars = snapshot.Stars
			status.Forks = snapshot.Forks
			status.OpenIssues = snapshot.OpenIssues
			status.OpenPRs = snapshot.OpenPRs
		}

		// Get alert counts
		alertCounts, countErr := analyticsRepo.GetAlertCounts(ctx, repo.ID)
		if countErr != nil && !errors.Is(countErr, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to get alert counts for %s: %w", repo.FullName(), countErr)
		}
		for _, count := range alertCounts {
			status.TotalAlerts += count
		}

		if status.TotalAlerts > 0 {
			reposWithAlerts++
			totalAlerts += status.TotalAlerts
		}

		statuses = append(statuses, status)
	}

	// Display header
	output.Plain("")
	output.Info("Analytics Status - All Repositories")
	output.Plain(strings.Repeat("─", 80))

	// Display table
	output.Plain(fmt.Sprintf("%-35s %6s %6s %7s %5s %7s  %s",
		"Repository", "Stars", "Forks", "Issues", "PRs", "Alerts", "Last Sync"))
	output.Plain(strings.Repeat("─", 80))

	for _, status := range statuses {
		lastSync := "Never"
		if status.LastSyncAt != nil {
			lastSync = formatTimeAgo(status.LastSyncAt)
		}

		output.Plain(fmt.Sprintf("%-35s %6d %6d %7d %5d %7d  %s",
			truncate(status.FullName, 35),
			status.Stars,
			status.Forks,
			status.OpenIssues,
			status.OpenPRs,
			status.TotalAlerts,
			lastSync))
	}

	// Display summary
	output.Plain("")
	if reposWithAlerts > 0 {
		output.Info(fmt.Sprintf("Total: %d repositories, %d with %d open alerts",
			len(statuses), reposWithAlerts, totalAlerts))
	} else {
		output.Info(fmt.Sprintf("Total: %d repositories, no open alerts", len(statuses)))
	}

	// Get last sync run info
	lastSync, err := analyticsRepo.GetLatestSyncRun(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get latest sync run: %w", err)
	}
	if lastSync != nil && lastSync.CompletedAt != nil {
		duration := formatDuration(lastSync.DurationMs)
		output.Info(fmt.Sprintf("Last sync: %s (%s)", formatTimeAgo(lastSync.CompletedAt), duration))
	}

	output.Plain(strings.Repeat("─", 80))
	output.Plain("")

	return nil
}

// displaySingleRepository shows detailed status for a specific repository
func displaySingleRepository(ctx context.Context, analyticsRepo db.AnalyticsRepo, fullName string) error {
	// Get repository
	repo, err := analyticsRepo.GetRepository(ctx, fullName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("repository %q not found in database. Run 'analytics sync' first", fullName) //nolint:err113 // user-facing CLI error
		}
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Get latest snapshot
	snapshot, err := analyticsRepo.GetLatestSnapshot(ctx, repo.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Get alert counts by type
	alertsByType, err := analyticsRepo.GetAlertCountsByType(ctx, repo.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get alert counts by type: %w", err)
	}

	// Get alert counts by severity
	alertsBySeverity, err := analyticsRepo.GetAlertCounts(ctx, repo.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to get alert counts by severity: %w", err)
	}

	// Display header
	output.Plain("")
	output.Info(fmt.Sprintf("Analytics Status - %s", repo.FullName()))
	output.Plain(strings.Repeat("─", 80))

	// Repository Info section
	output.Plain("Repository Info:")
	output.Plain(fmt.Sprintf("  Full Name:     %s", repo.FullName()))
	if repo.Description != "" {
		output.Plain(fmt.Sprintf("  Description:   %s", truncate(repo.Description, 60)))
	}
	if repo.Language != "" {
		output.Plain(fmt.Sprintf("  Language:      %s", repo.Language))
	}
	if repo.DefaultBranch != "" {
		output.Plain(fmt.Sprintf("  Default Branch: %s", repo.DefaultBranch))
	}
	visibility := "Public"
	if repo.IsPrivate {
		visibility = "Private"
	}
	output.Plain(fmt.Sprintf("  Visibility:    %s", visibility))
	status := "Active"
	if repo.IsArchived {
		status = "Archived"
	}
	output.Plain(fmt.Sprintf("  Status:        %s", status))

	// Current Metrics section
	output.Plain("")
	if snapshot != nil {
		snapshotAge := "unknown"
		if !snapshot.SnapshotAt.IsZero() {
			snapshotAge = formatTimeAgo(&snapshot.SnapshotAt)
		}
		output.Plain(fmt.Sprintf("Current Metrics (as of %s):", snapshotAge))
		output.Plain(fmt.Sprintf("  Stars:         %d", snapshot.Stars))
		output.Plain(fmt.Sprintf("  Forks:         %d", snapshot.Forks))
		output.Plain(fmt.Sprintf("  Watchers:      %d", snapshot.Watchers))
		output.Plain(fmt.Sprintf("  Open Issues:   %d", snapshot.OpenIssues))
		output.Plain(fmt.Sprintf("  Open PRs:      %d", snapshot.OpenPRs))
		if snapshot.BranchCount > 0 {
			output.Plain(fmt.Sprintf("  Branches:      %d", snapshot.BranchCount))
		}
	} else {
		output.Warn("No metrics available yet. Run 'analytics sync' to collect data.")
	}

	// Security Alerts section
	output.Plain("")
	output.Plain("Security Alerts:")

	totalAlerts := 0
	for _, count := range alertsByType {
		totalAlerts += count
	}

	if totalAlerts > 0 {
		// By Type
		if alertsByType != nil {
			parts := []string{}
			if count, ok := alertsByType["dependabot"]; ok && count > 0 {
				parts = append(parts, fmt.Sprintf("Dependabot: %d", count))
			}
			if count, ok := alertsByType["code_scanning"]; ok && count > 0 {
				parts = append(parts, fmt.Sprintf("Code Scanning: %d", count))
			}
			if count, ok := alertsByType["secret_scanning"]; ok && count > 0 {
				parts = append(parts, fmt.Sprintf("Secret Scanning: %d", count))
			}
			if len(parts) > 0 {
				output.Plain(fmt.Sprintf("  By Type:       %s", strings.Join(parts, ", ")))
			}
		}

		// By Severity
		if alertsBySeverity != nil {
			parts := []string{}
			if count, ok := alertsBySeverity["critical"]; ok && count > 0 {
				parts = append(parts, fmt.Sprintf("Critical: %d", count))
			}
			if count, ok := alertsBySeverity["high"]; ok && count > 0 {
				parts = append(parts, fmt.Sprintf("High: %d", count))
			}
			if count, ok := alertsBySeverity["medium"]; ok && count > 0 {
				parts = append(parts, fmt.Sprintf("Medium: %d", count))
			}
			if count, ok := alertsBySeverity["low"]; ok && count > 0 {
				parts = append(parts, fmt.Sprintf("Low: %d", count))
			}
			if len(parts) > 0 {
				output.Plain(fmt.Sprintf("  By Severity:   %s", strings.Join(parts, ", ")))
			}
		}
	} else {
		output.Success("  No security alerts")
	}

	// Last Sync section
	output.Plain("")
	output.Plain("Last Sync:")
	if repo.LastSyncAt != nil {
		output.Plain(fmt.Sprintf("  Completed:     %s", formatTimeAgo(repo.LastSyncAt)))
	} else {
		output.Warn("  Never synced")
	}

	output.Plain(strings.Repeat("─", 80))
	output.Plain("")

	return nil
}

// formatTimeAgo converts a timestamp to a human-readable "X ago" format
func formatTimeAgo(t *time.Time) string {
	if t == nil {
		return "unknown"
	}

	duration := time.Since(*t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case duration < 30*24*time.Hour:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}

// formatDuration converts milliseconds to a human-readable duration format
func formatDuration(ms int64) string {
	if ms == 0 {
		return "0s"
	}

	duration := time.Duration(ms) * time.Millisecond

	switch {
	case duration < time.Second:
		return fmt.Sprintf("%dms", ms)
	case duration < time.Minute:
		return fmt.Sprintf("%.1fs", duration.Seconds())
	case duration < time.Hour:
		return fmt.Sprintf("%.1fm", duration.Minutes())
	default:
		return fmt.Sprintf("%.1fh", duration.Hours())
	}
}

// truncate truncates a string to the specified length, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
