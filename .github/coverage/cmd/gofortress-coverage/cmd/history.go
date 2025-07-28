package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/coverage/internal/history"
	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

var historyCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "history",
	Short: "Manage coverage history",
	Long:  `Manage historical coverage data for trend analysis and tracking.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Get flags
		inputFile, _ := cmd.Flags().GetString("add")
		branch, _ := cmd.Flags().GetString("branch")
		commit, _ := cmd.Flags().GetString("commit")
		commitURL, _ := cmd.Flags().GetString("commit-url")
		showTrend, _ := cmd.Flags().GetBool("trend")
		showStats, _ := cmd.Flags().GetBool("stats")
		cleanup, _ := cmd.Flags().GetBool("cleanup")
		days, _ := cmd.Flags().GetInt("days")
		format, _ := cmd.Flags().GetString("format")

		// Load configuration
		cfg := config.Load()

		// Create history tracker
		historyConfig := &history.Config{
			StoragePath:    cfg.History.StoragePath,
			RetentionDays:  cfg.History.RetentionDays,
			MaxEntries:     cfg.History.MaxEntries,
			AutoCleanup:    cfg.History.AutoCleanup,
			MetricsEnabled: cfg.History.MetricsEnabled,
		}
		tracker := history.NewWithConfig(historyConfig)

		ctx := context.Background()

		// Handle different operations
		switch {
		case inputFile != "":
			return addToHistory(ctx, tracker, inputFile, branch, commit, commitURL, cfg, cmd)
		case showTrend:
			return showTrendData(ctx, tracker, branch, days, format, cmd)
		case showStats:
			return showStatistics(ctx, tracker, format, cmd)
		case cleanup:
			return cleanupHistory(ctx, tracker, cmd)
		default:
			return showLatestEntry(ctx, tracker, branch, format, cmd)
		}
	},
}

func addToHistory(ctx context.Context, tracker *history.Tracker, inputFile, branch, commit, commitURL string, cfg *config.Config, cmd *cobra.Command) error { //nolint:revive // function naming
	// Parse coverage data
	p := parser.New()
	coverage, err := p.ParseFile(ctx, inputFile)
	if err != nil {
		return fmt.Errorf("failed to parse coverage file: %w", err)
	}

	// Set defaults
	if branch == "" {
		branch = cfg.GitHub.CommitSHA
		if branch == "" {
			branch = "main"
		}
	}
	if commit == "" {
		commit = cfg.GitHub.CommitSHA
	}

	// Add to history
	var options []history.Option
	if branch != "" {
		options = append(options, history.WithBranch(branch))
	}
	if commit != "" {
		options = append(options, history.WithCommit(commit, commitURL))
	}
	if cfg.GitHub.Owner != "" {
		options = append(options, history.WithMetadata("project", cfg.GitHub.Owner+"/"+cfg.GitHub.Repository))
	}

	err = tracker.Record(ctx, coverage, options...)
	if err != nil {
		return fmt.Errorf("failed to record coverage in history: %w", err)
	}

	cmd.Printf("Coverage recorded successfully!\n")
	cmd.Printf("Branch: %s\n", branch)
	cmd.Printf("Commit: %s\n", commit)
	cmd.Printf("Coverage: %.2f%% (%d/%d lines)\n",
		coverage.Percentage, coverage.CoveredLines, coverage.TotalLines)

	return nil
}

func showTrendData(ctx context.Context, tracker *history.Tracker, branch string, days int, format string, cmd *cobra.Command) error { //nolint:revive // function naming
	if branch == "" {
		branch = "main"
	}
	if days == 0 {
		days = 30
	}

	var options []history.TrendOption
	options = append(options, history.WithTrendBranch(branch))
	options = append(options, history.WithTrendDays(days))

	trendData, err := tracker.GetTrend(ctx, options...)
	if err != nil {
		return fmt.Errorf("failed to get trend data: %w", err)
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(trendData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal trend data: %w", err)
		}
		cmd.Println(string(data))
	default:
		cmd.Printf("Coverage Trend Analysis\n")
		cmd.Printf("======================\n")
		cmd.Printf("Branch: %s\n", branch)
		cmd.Printf("Period: %d days\n", days)
		cmd.Printf("Total Entries: %d\n", trendData.Summary.TotalEntries)

		if trendData.Summary.TotalEntries > 0 {
			cmd.Printf("Average Coverage: %.2f%%\n", trendData.Summary.AveragePercentage)
			cmd.Printf("Min Coverage: %.2f%%\n", trendData.Summary.MinPercentage)
			cmd.Printf("Max Coverage: %.2f%%\n", trendData.Summary.MaxPercentage)
			cmd.Printf("Current Trend: %s\n", trendData.Summary.CurrentTrend)

			if trendData.Analysis.Volatility > 0 {
				cmd.Printf("Volatility: %.2f\n", trendData.Analysis.Volatility)
			}
			if trendData.Analysis.Momentum != 0 {
				cmd.Printf("Momentum: %.2f\n", trendData.Analysis.Momentum)
			}

			if trendData.Analysis.Prediction != nil {
				cmd.Printf("\nPrediction:\n")
				if pred := trendData.Analysis.Prediction.NextWeek; pred != nil {
					cmd.Printf("  Next Week: %.2f%% (%.2f-%.2f)\n",
						pred.Percentage, pred.Range.Min, pred.Range.Max)
				}
				if pred := trendData.Analysis.Prediction.NextMonth; pred != nil {
					cmd.Printf("  Next Month: %.2f%% (%.2f-%.2f)\n",
						pred.Percentage, pred.Range.Min, pred.Range.Max)
				}
				cmd.Printf("  Confidence: %.1f%%\n", trendData.Analysis.Prediction.Confidence)
			}
		}
	}

	return nil
}

func showStatistics(ctx context.Context, tracker *history.Tracker, format string, cmd *cobra.Command) error { //nolint:revive // function naming
	stats, err := tracker.GetStatistics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal statistics: %w", err)
		}
		cmd.Println(string(data))
	default:
		cmd.Printf("Coverage History Statistics\n")
		cmd.Printf("===========================\n")
		cmd.Printf("Total Entries: %d\n", stats.TotalEntries)
		cmd.Printf("Storage Size: %d bytes\n", stats.StorageSize)

		if stats.TotalEntries > 0 {
			cmd.Printf("Date Range: %s to %s\n",
				stats.OldestEntry.Format("2006-01-02"),
				stats.NewestEntry.Format("2006-01-02"))
		}

		if len(stats.UniqueProjects) > 0 {
			cmd.Printf("\nProjects:\n")
			for project, count := range stats.UniqueProjects {
				cmd.Printf("  %s: %d entries\n", project, count)
			}
		}

		if len(stats.UniqueBranches) > 0 {
			cmd.Printf("\nBranches:\n")
			for branch, count := range stats.UniqueBranches {
				cmd.Printf("  %s: %d entries\n", branch, count)
			}
		}
	}

	return nil
}

func cleanupHistory(ctx context.Context, tracker *history.Tracker, cmd *cobra.Command) error { //nolint:revive // function naming
	err := tracker.Cleanup(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup history: %w", err)
	}

	cmd.Println("History cleanup completed successfully!")
	return nil
}

func showLatestEntry(ctx context.Context, tracker *history.Tracker, branch, format string, cmd *cobra.Command) error { //nolint:revive // function naming
	if branch == "" {
		branch = "main"
	}

	entry, err := tracker.GetLatestEntry(ctx, branch)
	if err != nil {
		return fmt.Errorf("failed to get latest entry: %w", err)
	}

	switch format {
	case "json":
		data, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal entry: %w", err)
		}
		cmd.Println(string(data))
	default:
		cmd.Printf("Latest Coverage Entry\n")
		cmd.Printf("====================\n")
		cmd.Printf("Branch: %s\n", entry.Branch)
		cmd.Printf("Commit: %s\n", entry.CommitSHA)
		cmd.Printf("Timestamp: %s\n", entry.Timestamp.Format(time.RFC3339))
		cmd.Printf("Coverage: %.2f%% (%d/%d lines)\n",
			entry.Coverage.Percentage, entry.Coverage.CoveredLines, entry.Coverage.TotalLines)

		if len(entry.Metadata) > 0 {
			cmd.Printf("\nMetadata:\n")
			for key, value := range entry.Metadata {
				cmd.Printf("  %s: %s\n", key, value)
			}
		}
	}

	return nil
}

func init() { //nolint:gochecknoinits // CLI command initialization
	historyCmd.Flags().StringP("add", "a", "", "Add coverage data file to history")
	historyCmd.Flags().StringP("branch", "b", "", "Branch name (defaults to main)")
	historyCmd.Flags().StringP("commit", "c", "", "Commit SHA")
	historyCmd.Flags().String("commit-url", "", "Commit URL")
	historyCmd.Flags().Bool("trend", false, "Show trend analysis")
	historyCmd.Flags().Bool("stats", false, "Show history statistics")
	historyCmd.Flags().Bool("cleanup", false, "Cleanup old history entries")
	historyCmd.Flags().Int("days", 30, "Number of days for trend analysis")
	historyCmd.Flags().String("format", "text", "Output format (text, json)")
}
