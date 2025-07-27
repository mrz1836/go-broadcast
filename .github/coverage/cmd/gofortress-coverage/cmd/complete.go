package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/coverage/internal/badge"
	"github.com/mrz1836/go-broadcast/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/coverage/internal/github"
	"github.com/mrz1836/go-broadcast/coverage/internal/history"
	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
	"github.com/mrz1836/go-broadcast/coverage/internal/report"
)

// ErrCoverageBelowThreshold indicates that coverage percentage is below the configured threshold
var ErrCoverageBelowThreshold = errors.New("coverage is below threshold")

var completeCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "complete",
	Short: "Run complete coverage pipeline",
	Long: `Run the complete coverage pipeline: parse coverage, generate badge and report, 
update history, and create GitHub PR comment if in PR context.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		inputFile, _ := cmd.Flags().GetString("input")
		outputDir, _ := cmd.Flags().GetString("output")
		skipHistory, _ := cmd.Flags().GetBool("skip-history")
		skipGitHub, _ := cmd.Flags().GetBool("skip-github")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Load configuration
		cfg := config.Load()

		// Set defaults
		if inputFile == "" {
			inputFile = cfg.Coverage.InputFile
		}
		if outputDir == "" {
			outputDir = cfg.Coverage.OutputDir
		}

		// Validate configuration
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		cmd.Printf("Starting GoFortress Coverage Pipeline\n")
		cmd.Printf("====================================\n")
		cmd.Printf("Input: %s\n", inputFile)
		cmd.Printf("Output Directory: %s\n", outputDir)
		if dryRun {
			cmd.Printf("Mode: DRY RUN\n")
		}
		cmd.Printf("\n")

		// Step 1: Parse coverage data
		cmd.Printf("🔍 Step 1: Parsing coverage data...\n")
		parserConfig := &parser.Config{
			ExcludePaths:     cfg.Coverage.ExcludePaths,
			ExcludeFiles:     cfg.Coverage.ExcludeFiles,
			ExcludeGenerated: cfg.Coverage.ExcludeTests,
		}
		p := parser.NewWithConfig(parserConfig)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		coverage, err := p.ParseFile(ctx, inputFile)
		if err != nil {
			return fmt.Errorf("failed to parse coverage file: %w", err)
		}

		cmd.Printf("   ✅ Coverage: %.2f%% (%d/%d lines)\n",
			coverage.Percentage, coverage.CoveredLines, coverage.TotalLines)
		cmd.Printf("   📦 Packages: %d\n", len(coverage.Packages))

		// Check threshold
		if coverage.Percentage < cfg.Coverage.Threshold {
			cmd.Printf("   ⚠️  Below threshold %.2f%%\n", cfg.Coverage.Threshold)
		}
		cmd.Printf("\n")

		// Create output directory if needed
		if cfg.Storage.AutoCreate && !dryRun {
			if mkdirErr := os.MkdirAll(outputDir, cfg.Storage.DirMode); mkdirErr != nil {
				return fmt.Errorf("failed to create output directory: %w", mkdirErr)
			}
		}

		// Step 2: Generate badge
		cmd.Printf("🏷️  Step 2: Generating coverage badge...\n")
		badgeFile := filepath.Join(outputDir, cfg.Badge.OutputFile)

		var badgeOptions []badge.Option
		if cfg.Badge.Label != "coverage" {
			badgeOptions = append(badgeOptions, badge.WithLabel(cfg.Badge.Label))
		}
		if cfg.Badge.Style != "flat" {
			badgeOptions = append(badgeOptions, badge.WithStyle(cfg.Badge.Style))
		}
		if cfg.Badge.Logo != "" {
			badgeOptions = append(badgeOptions, badge.WithLogo(cfg.Badge.Logo))
		}

		badgeGen := badge.New()
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		svgContent, err := badgeGen.Generate(ctx, coverage.Percentage, badgeOptions...)
		if err != nil {
			return fmt.Errorf("failed to generate badge: %w", err)
		}

		if !dryRun {
			if writeErr := os.WriteFile(badgeFile, svgContent, cfg.Storage.FileMode); writeErr != nil {
				return fmt.Errorf("failed to write badge file: %w", writeErr)
			}
		}

		cmd.Printf("   ✅ Badge saved: %s\n", badgeFile)
		cmd.Printf("\n")

		// Step 3: Generate HTML report
		cmd.Printf("📊 Step 3: Generating HTML report...\n")
		reportFile := filepath.Join(outputDir, cfg.Report.OutputFile)

		reportConfig := &report.Config{
			Title:            cfg.Report.Title,
			Theme:            cfg.Report.Theme,
			ShowPackages:     cfg.Report.ShowPackages,
			ShowFiles:        cfg.Report.ShowFiles,
			ShowMissing:      cfg.Report.ShowMissing,
			InteractiveTrees: cfg.Report.Interactive,
		}
		reportGen := report.NewWithConfig(reportConfig)
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		htmlContent, err := reportGen.Generate(ctx, coverage)
		if err != nil {
			return fmt.Errorf("failed to generate report: %w", err)
		}

		if !dryRun {
			if err := os.WriteFile(reportFile, htmlContent, cfg.Storage.FileMode); err != nil {
				return fmt.Errorf("failed to write report file: %w", err)
			}
		}

		cmd.Printf("   ✅ Report saved: %s\n", reportFile)
		cmd.Printf("\n")

		// Step 4: Update history (if enabled)
		var trend = "stable"
		if cfg.History.Enabled && !skipHistory {
			cmd.Printf("📈 Step 4: Updating coverage history...\n")

			historyConfig := &history.Config{
				StoragePath:    cfg.History.StoragePath,
				RetentionDays:  cfg.History.RetentionDays,
				MaxEntries:     cfg.History.MaxEntries,
				AutoCleanup:    cfg.History.AutoCleanup,
				MetricsEnabled: cfg.History.MetricsEnabled,
			}
			tracker := history.NewWithConfig(historyConfig)

			// Get trend before adding new entry
			branch := "main"
			if cfg.GitHub.CommitSHA != "" {
				branch = cfg.GitHub.CommitSHA
			}

			if latest, err := tracker.GetLatestEntry(ctx, branch); err == nil {
				if coverage.Percentage > latest.Coverage.Percentage {
					trend = "up"
				} else if coverage.Percentage < latest.Coverage.Percentage {
					trend = "down"
				}
			}

			// Add new entry
			if !dryRun {
				var historyOptions []history.Option
				historyOptions = append(historyOptions, history.WithBranch(branch))
				if cfg.GitHub.CommitSHA != "" {
					historyOptions = append(historyOptions, history.WithCommit(cfg.GitHub.CommitSHA, ""))
				}
				if cfg.GitHub.Owner != "" {
					historyOptions = append(historyOptions,
						history.WithMetadata("project", cfg.GitHub.Owner+"/"+cfg.GitHub.Repository))
				}

				if err := tracker.Record(ctx, coverage, historyOptions...); err != nil {
					return fmt.Errorf("failed to record coverage history: %w", err)
				}
			}

			cmd.Printf("   ✅ History updated (trend: %s)\n", trend)
			cmd.Printf("\n")
		} else {
			cmd.Printf("📈 Step 4: Coverage history (skipped)\n\n")
		}

		// Step 5: GitHub integration (if in GitHub context)
		if cfg.IsGitHubContext() && !skipGitHub {
			cmd.Printf("🐙 Step 5: GitHub integration...\n")

			if cfg.GitHub.Token == "" {
				cmd.Printf("   ⚠️  Skipped: No GitHub token provided\n\n")
			} else {
				// Create GitHub client
				githubConfig := &github.Config{
					Token:      cfg.GitHub.Token,
					BaseURL:    "https://api.github.com",
					Timeout:    cfg.GitHub.Timeout,
					RetryCount: 3,
					UserAgent:  "gofortress-coverage/1.0",
				}
				client := github.NewWithConfig(githubConfig)

				// Create PR comment if in PR context
				if cfg.IsPullRequestContext() && cfg.GitHub.PostComments {
					badgeURL := cfg.GetBadgeURL()
					comment := client.GenerateCoverageComment(coverage.Percentage, trend, badgeURL)

					if dryRun {
						cmd.Printf("   📝 Would post PR comment to #%d\n", cfg.GitHub.PullRequest)
					} else {
						_, err := client.CreateComment(ctx, cfg.GitHub.Owner, cfg.GitHub.Repository,
							cfg.GitHub.PullRequest, comment)
						if err != nil {
							cmd.Printf("   ⚠️  Failed to post PR comment: %v\n", err)
						} else {
							cmd.Printf("   ✅ PR comment posted to #%d\n", cfg.GitHub.PullRequest)
						}
					}
				}

				// Create commit status
				if cfg.GitHub.CommitSHA != "" && cfg.GitHub.CreateStatuses {
					var state string
					var description string

					if coverage.Percentage >= cfg.Coverage.Threshold {
						state = github.StatusSuccess
						description = fmt.Sprintf("Coverage: %.2f%% ✅", coverage.Percentage)
					} else {
						state = github.StatusFailure
						description = fmt.Sprintf("Coverage: %.2f%% (below %.2f%% threshold)",
							coverage.Percentage, cfg.Coverage.Threshold)
					}

					statusReq := &github.StatusRequest{
						State:       state,
						TargetURL:   cfg.GetReportURL(),
						Description: description,
						Context:     github.ContextCoverage,
					}

					if dryRun {
						cmd.Printf("   📊 Would create commit status: %s\n", state)
					} else {
						err := client.CreateStatus(ctx, cfg.GitHub.Owner, cfg.GitHub.Repository,
							cfg.GitHub.CommitSHA, statusReq)
						if err != nil {
							cmd.Printf("   ⚠️  Failed to create commit status: %v\n", err)
						} else {
							cmd.Printf("   ✅ Commit status created: %s\n", state)
						}
					}
				}

				cmd.Printf("\n")
			}
		} else {
			cmd.Printf("🐙 Step 5: GitHub integration (skipped)\n\n")
		}

		// Final summary
		cmd.Printf("✨ Pipeline Complete!\n")
		cmd.Printf("==================\n")
		cmd.Printf("Coverage: %.2f%% (%s)\n", coverage.Percentage,
			getStatusIcon(coverage.Percentage, cfg.Coverage.Threshold))
		cmd.Printf("Badge: %s\n", badgeFile)
		cmd.Printf("Report: %s\n", reportFile)

		if cfg.GitHub.Owner != "" && cfg.GitHub.Repository != "" {
			cmd.Printf("Badge URL: %s\n", cfg.GetBadgeURL())
			cmd.Printf("Report URL: %s\n", cfg.GetReportURL())
		}

		// Return error if below threshold
		if coverage.Percentage < cfg.Coverage.Threshold {
			return fmt.Errorf("%w: %.2f%% is below threshold %.2f%%", ErrCoverageBelowThreshold, coverage.Percentage, cfg.Coverage.Threshold)
		}

		return nil
	},
}

func getStatusIcon(coverage, threshold float64) string {
	if coverage < threshold {
		return "🔴 Below Threshold"
	}
	switch {
	case coverage >= 90:
		return "🟢 Excellent"
	case coverage >= 80:
		return "🟡 Good"
	case coverage >= 70:
		return "🟠 Fair"
	default:
		return "🔴 Needs Improvement"
	}
}

func init() { //nolint:gochecknoinits // CLI command initialization
	completeCmd.Flags().StringP("input", "i", "", "Input coverage file")
	completeCmd.Flags().StringP("output", "o", "", "Output directory")
	completeCmd.Flags().Bool("skip-history", false, "Skip history tracking")
	completeCmd.Flags().Bool("skip-github", false, "Skip GitHub integration")
	completeCmd.Flags().Bool("dry-run", false, "Show what would be done without actually doing it")
}
