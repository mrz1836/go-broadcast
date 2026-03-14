package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm"

	"github.com/mrz1836/go-broadcast/internal/db"
)

//nolint:gochecknoglobals // Package-level variables for CLI flags
var (
	metricsFlagsMu sync.RWMutex
	metricsLast    string
	metricsRepo    string
	metricsRunID   string
	metricsJSON    bool
)

// getMetricsFlags returns a snapshot of metrics flags (thread-safe)
func getMetricsFlags() (last, repo, runID string, jsonOut bool) {
	metricsFlagsMu.RLock()
	defer metricsFlagsMu.RUnlock()
	return metricsLast, metricsRepo, metricsRunID, metricsJSON
}

// initMetrics initializes metrics command flags
func initMetrics() {
	metricsCmd.Flags().StringVar(&metricsLast, "last", "", "Show runs from last period (e.g., 7d, 24h, 30d)")
	metricsCmd.Flags().StringVar(&metricsRepo, "repo", "", "Filter by target repo (owner/name)")
	metricsCmd.Flags().StringVar(&metricsRunID, "run", "", "Show details for specific run ID (external_id)")
	metricsCmd.Flags().BoolVar(&metricsJSON, "json", false, "Output as JSON")
}

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Query sync metrics and history",
	Long: `Display sync metrics and historical data from the broadcast sync tracking database.

Shows information about:
  • Total runs and success rate
  • Average sync duration
  • Recent sync activity
  • Per-repository sync history
  • Detailed run information

Modes:
  • Default: Summary statistics across all runs
  • --last <period>: Recent runs from the specified time period
  • --repo <owner/name>: Sync history for a specific target repository
  • --run <external_id>: Detailed information for a specific run`,
	Example: `  # Show summary statistics
  go-broadcast metrics

  # Show recent syncs (last 7 days)
  go-broadcast metrics --last 7d

  # Show sync history for specific repo
  go-broadcast metrics --repo mrz1836/go-paymail

  # Show details for specific run
  go-broadcast metrics --run SR-20260215-abc123

  # Output as JSON for automation
  go-broadcast metrics --json
  go-broadcast metrics --last 24h --json`,
	RunE: runMetrics,
}

// runMetrics executes the metrics command
func runMetrics(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Get flags
	last, repo, runID, jsonOut := getMetricsFlags()

	// Open database
	database, err := openDatabase()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			// Ignore close errors in CLI context
			_ = closeErr
		}
	}()

	// Get repository
	syncRepo := db.NewBroadcastSyncRepo(database.DB())

	// Route based on flags
	switch {
	case runID != "":
		return showRunDetails(ctx, syncRepo, runID, jsonOut)
	case repo != "":
		return showRepoHistory(ctx, syncRepo, database.DB(), repo, jsonOut)
	case last != "":
		return showRecentRuns(ctx, syncRepo, last, jsonOut)
	default:
		return showSummaryStats(ctx, syncRepo, jsonOut)
	}
}

// showSummaryStats displays aggregate statistics
func showSummaryStats(ctx context.Context, repo db.BroadcastSyncRepo, jsonOut bool) error {
	stats, err := repo.GetSyncRunSummaryStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get summary stats: %w", err)
	}

	if jsonOut {
		return outputJSONMetrics("summary", stats)
	}

	// Human-readable output
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Println("📊 Sync Metrics Summary")
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Println(strings.Repeat("─", 40))
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("Total Runs:           %d\n", stats.TotalRuns)
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("Success Rate:         %.1f%%\n", stats.SuccessRate)
	if stats.AvgDurationMs > 0 {
		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Printf("Avg Duration:         %s\n", formatMetricsDuration(stats.AvgDurationMs))
	}
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("Runs This Week:       %d\n", stats.RunsThisWeek)
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("Files Changed (7d):   %d\n", stats.FilesChangedThisWeek)
	if stats.LastRunAt != nil {
		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Printf("Last Run:             %s\n", formatMetricsTime(*stats.LastRunAt))
	}

	return nil
}

// showRecentRuns displays recent sync runs
func showRecentRuns(ctx context.Context, repo db.BroadcastSyncRepo, period string, jsonOut bool) error {
	// Parse duration
	since, err := parseDuration(period)
	if err != nil {
		return fmt.Errorf("invalid period %q: %w", period, err)
	}

	runs, err := repo.ListRecentSyncRuns(ctx, since, 100)
	if err != nil {
		return fmt.Errorf("failed to list recent runs: %w", err)
	}

	if jsonOut {
		return outputJSONMetrics("recent_runs", runs)
	}

	// Human-readable output
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("📅 Recent Runs (since %s)\n", formatMetricsTime(since))
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Println(strings.Repeat("─", 100))

	if len(runs) == 0 {
		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Println("No runs found in the specified period.")
		return nil
	}

	// Table header
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("%-20s %-10s %-8s %-12s %-10s\n",
		"Run ID", "Status", "Targets", "Duration", "Started")
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Println(strings.Repeat("─", 100))

	// Table rows
	for _, run := range runs {
		status := formatMetricsStatus(run.Status)
		targets := fmt.Sprintf("%d/%d", run.SuccessfulTargets, run.TotalTargets)
		duration := formatMetricsDuration(run.DurationMs)
		started := formatMetricsTimeShort(run.StartedAt)

		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Printf("%-20s %-10s %-8s %-12s %-10s\n",
			run.ExternalID, status, targets, duration, started)
	}

	return nil
}

// showRepoHistory displays sync history for a specific repository
func showRepoHistory(ctx context.Context, repo db.BroadcastSyncRepo, gormDB *gorm.DB, repoName string, jsonOut bool) error {
	// Try to parse as uint first (repo ID)
	var repoID uint
	if _, err := fmt.Sscanf(repoName, "%d", &repoID); err != nil {
		// Try to resolve "org/repo" format to DB ID
		parts := strings.SplitN(repoName, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid repo format %q: use numeric ID or owner/name (e.g., --repo mrz1836/go-broadcast)", repoName) //nolint:err113 // user-facing CLI error
		}
		repoRepo := db.NewRepoRepository(gormDB)
		dbRepo, lookupErr := repoRepo.GetByFullName(ctx, parts[0], parts[1])
		if lookupErr != nil {
			return fmt.Errorf("repo %q not found in database: %w", repoName, lookupErr)
		}
		repoID = dbRepo.ID
	}

	runs, err := repo.ListSyncRunsByRepo(ctx, repoID, 100)
	if err != nil {
		return fmt.Errorf("failed to list runs for repo %d: %w", repoID, err)
	}

	if jsonOut {
		return outputJSONMetrics("repo_history", runs)
	}

	// Human-readable output
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("🔍 Sync History for Repo %s (ID %d)\n", repoName, repoID)
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Println(strings.Repeat("─", 100))

	if len(runs) == 0 {
		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Println("No sync runs found for this repository.")
		return nil
	}

	// Table header
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("%-20s %-10s %-8s %-10s %-10s\n",
		"Run ID", "Status", "Files", "Duration", "Started")
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Println(strings.Repeat("─", 100))

	// Table rows
	for _, run := range runs {
		status := formatMetricsStatus(run.Status)
		files := fmt.Sprintf("%d", run.TotalFilesChanged)
		duration := formatMetricsDuration(run.DurationMs)
		started := formatMetricsTimeShort(run.StartedAt)

		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Printf("%-20s %-10s %-8s %-10s %-10s\n",
			run.ExternalID, status, files, duration, started)
	}

	return nil
}

// showRunDetails displays detailed information for a specific run
func showRunDetails(ctx context.Context, repo db.BroadcastSyncRepo, extID string, jsonOut bool) error {
	run, err := repo.GetSyncRunByExternalID(ctx, extID)
	if err != nil {
		return fmt.Errorf("failed to get run %q: %w", extID, err)
	}

	if jsonOut {
		return outputJSONMetrics("run_details", run)
	}

	// Human-readable output
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("🔍 Run Details: %s\n", run.ExternalID)
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Println(strings.Repeat("─", 80))

	// Basic info
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("Status:          %s\n", formatMetricsStatus(run.Status))
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("Started:         %s\n", formatMetricsTime(run.StartedAt))
	if run.EndedAt != nil {
		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Printf("Ended:           %s\n", formatMetricsTime(*run.EndedAt))
	}
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("Duration:        %s\n", formatMetricsDuration(run.DurationMs))
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("Trigger:         %s\n", run.Trigger)

	// Target stats
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("\nTargets:         %d total, %d successful, %d failed, %d skipped\n",
		run.TotalTargets, run.SuccessfulTargets, run.FailedTargets, run.SkippedTargets)

	// File stats
	//nolint:forbidigo // CLI output requires fmt.Printf for formatting
	fmt.Printf("\nFile Changes:    %d files, +%d/-%d lines\n",
		run.TotalFilesChanged, run.TotalLinesAdded, run.TotalLinesRemoved)

	// Target results
	if len(run.TargetResults) > 0 {
		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Printf("\n📋 Target Results\n")
		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Println(strings.Repeat("─", 80))
		for _, result := range run.TargetResults {
			//nolint:forbidigo // CLI output requires fmt.Printf for formatting
			fmt.Printf("  Target ID %d: %s", result.TargetID, formatMetricsStatus(result.Status))
			if result.FilesChanged > 0 {
				//nolint:forbidigo // CLI output requires fmt.Printf for formatting
				fmt.Printf(" (%d files changed)", result.FilesChanged)
			}
			if result.PRNumber != nil {
				//nolint:forbidigo // CLI output requires fmt.Printf for formatting
				fmt.Printf(" [PR #%d]", *result.PRNumber)
			}
			//nolint:forbidigo // CLI output requires fmt.Printf for formatting
			fmt.Println()

			if result.ErrorMessage != "" {
				//nolint:forbidigo // CLI output requires fmt.Printf for formatting
				fmt.Printf("    Error: %s\n", result.ErrorMessage)
			}
		}
	}

	// Error summary
	if run.ErrorSummary != "" {
		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Printf("\n❌ Error Summary\n")
		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Println(strings.Repeat("─", 80))
		//nolint:forbidigo // CLI output requires fmt.Printf for formatting
		fmt.Println(run.ErrorSummary)
	}

	return nil
}

// outputJSONMetrics outputs metrics data as JSON
func outputJSONMetrics(dataType string, data interface{}) error {
	output := struct {
		GeneratedAt string      `json:"generated_at"`
		Type        string      `json:"type"`
		Data        interface{} `json:"data"`
	}{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Type:        dataType,
		Data:        data,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// formatMetricsStatus returns a simple status string
func formatMetricsStatus(status string) string {
	switch status {
	case "success":
		return "✓ success"
	case "partial":
		return "⚠ partial"
	case "failed":
		return "✗ failed"
	case "running":
		return "⟳ running"
	case "pending":
		return "○ pending"
	case "skipped":
		return "- skipped"
	case "no_changes":
		return "- no changes"
	default:
		return status
	}
}

// formatMetricsDuration formats milliseconds as human-readable duration
func formatMetricsDuration(ms int64) string {
	if ms == 0 {
		return "-"
	}
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// formatMetricsTime formats a time as human-readable string
func formatMetricsTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05 MST")
}

// formatMetricsTimeShort formats a time as short string
func formatMetricsTimeShort(t time.Time) string {
	return t.Format("01-02 15:04")
}

var (
	// ErrEmptyDuration is returned when parsing an empty duration string
	ErrEmptyDuration = fmt.Errorf("empty duration")
	// ErrUnknownDurationUnit is returned when parsing an unknown duration unit
	ErrUnknownDurationUnit = fmt.Errorf("unknown duration unit")
)

// parseDuration parses a duration string like "7d", "24h", "30d"
func parseDuration(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, ErrEmptyDuration
	}

	// Parse number and unit
	var num int
	var unit string
	if _, err := fmt.Sscanf(s, "%d%s", &num, &unit); err != nil {
		return time.Time{}, fmt.Errorf("invalid format (expected <number><unit>, e.g., 7d): %w", err)
	}

	now := time.Now()
	switch unit {
	case "h":
		return now.Add(-time.Duration(num) * time.Hour), nil
	case "d":
		return now.AddDate(0, 0, -num), nil
	case "w":
		return now.AddDate(0, 0, -num*7), nil
	case "m":
		return now.AddDate(0, -num, 0), nil
	case "y":
		return now.AddDate(-num, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("%w: %q (valid: h, d, w, m, y)", ErrUnknownDurationUnit, unit)
	}
}
