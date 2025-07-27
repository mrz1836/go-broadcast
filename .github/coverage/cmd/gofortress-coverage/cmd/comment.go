package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/coverage/internal/github"
	"github.com/mrz1836/go-broadcast/coverage/internal/history"
	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

var (
	// ErrGitHubTokenRequired indicates that a GitHub token is required but not provided
	ErrGitHubTokenRequired = errors.New("GitHub token is required")
	// ErrGitHubOwnerRequired indicates that a GitHub repository owner is required but not provided
	ErrGitHubOwnerRequired = errors.New("GitHub repository owner is required")
	// ErrGitHubRepoRequired indicates that a GitHub repository name is required but not provided
	ErrGitHubRepoRequired = errors.New("GitHub repository name is required")
	// ErrPRNumberRequired indicates that a pull request number is required but not provided
	ErrPRNumberRequired = errors.New("pull request number is required")
)

var commentCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "comment",
	Short: "Create PR coverage comment",
	Long:  `Create or update pull request comments with coverage information.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		prNumber, _ := cmd.Flags().GetInt("pr")
		inputFile, _ := cmd.Flags().GetString("coverage")
		badgeURL, _ := cmd.Flags().GetString("badge-url")
		reportURL, _ := cmd.Flags().GetString("report-url")
		createStatus, _ := cmd.Flags().GetBool("status")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Load configuration
		cfg := config.Load()

		// Validate GitHub configuration
		if cfg.GitHub.Token == "" {
			return ErrGitHubTokenRequired
		}
		if cfg.GitHub.Owner == "" {
			return ErrGitHubOwnerRequired
		}
		if cfg.GitHub.Repository == "" {
			return ErrGitHubRepoRequired
		}

		// Use PR number from config if not provided
		if prNumber == 0 {
			prNumber = cfg.GitHub.PullRequest
		}
		if prNumber == 0 {
			return ErrPRNumberRequired
		}

		// Set defaults
		if inputFile == "" {
			inputFile = cfg.Coverage.InputFile
		}
		if badgeURL == "" {
			badgeURL = cfg.GetBadgeURL()
		}
		if reportURL == "" {
			reportURL = cfg.GetReportURL()
		}

		// Parse coverage data
		p := parser.New()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		coverage, err := p.ParseFile(ctx, inputFile)
		if err != nil {
			return fmt.Errorf("failed to parse coverage file: %w", err)
		}

		// Get trend information if history is enabled
		var trend = "stable"
		if cfg.History.Enabled {
			historyConfig := &history.Config{
				StoragePath:    cfg.History.StoragePath,
				RetentionDays:  cfg.History.RetentionDays,
				MaxEntries:     cfg.History.MaxEntries,
				AutoCleanup:    cfg.History.AutoCleanup,
				MetricsEnabled: cfg.History.MetricsEnabled,
			}
			tracker := history.NewWithConfig(historyConfig)

			// Get latest entry to compare
			branch := cfg.GitHub.CommitSHA
			if branch == "" {
				branch = "main"
			}

			if latest, latestErr := tracker.GetLatestEntry(ctx, branch); latestErr == nil {
				if coverage.Percentage > latest.Coverage.Percentage {
					trend = "up"
				} else if coverage.Percentage < latest.Coverage.Percentage {
					trend = "down"
				}
			}
		}

		// Create GitHub client
		githubConfig := &github.Config{
			Token:      cfg.GitHub.Token,
			BaseURL:    "https://api.github.com",
			Timeout:    cfg.GitHub.Timeout,
			RetryCount: 3,
			UserAgent:  "gofortress-coverage/1.0",
		}
		client := github.NewWithConfig(githubConfig)

		// Generate coverage comment
		comment := client.GenerateCoverageComment(coverage.Percentage, trend, badgeURL)

		if dryRun {
			cmd.Printf("Dry run mode - would post the following comment:\n")
			cmd.Printf("===============================================\n")
			cmd.Println(comment)
			cmd.Printf("===============================================\n")
			cmd.Printf("PR: %d\n", prNumber)
			cmd.Printf("Repository: %s/%s\n", cfg.GitHub.Owner, cfg.GitHub.Repository)
			if createStatus {
				cmd.Printf("Would also create commit status on: %s\n", cfg.GitHub.CommitSHA)
			}
			return nil
		}

		// Create or update PR comment
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := client.CreateComment(ctx, cfg.GitHub.Owner, cfg.GitHub.Repository, prNumber, comment)
		if err != nil {
			return fmt.Errorf("failed to create PR comment: %w", err)
		}

		cmd.Printf("Coverage comment posted successfully!\n")
		cmd.Printf("Comment ID: %d\n", result.ID)
		cmd.Printf("Coverage: %.2f%% (%s trend)\n", coverage.Percentage, trend)

		// Create commit status if requested
		if createStatus && cfg.GitHub.CommitSHA != "" {
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
				TargetURL:   reportURL,
				Description: description,
				Context:     github.ContextCoverage,
			}

			err = client.CreateStatus(ctx, cfg.GitHub.Owner, cfg.GitHub.Repository, cfg.GitHub.CommitSHA, statusReq)
			if err != nil {
				cmd.Printf("Warning: failed to create commit status: %v\n", err)
			} else {
				cmd.Printf("Commit status created: %s\n", state)
			}
		}

		return nil
	},
}

func init() { //nolint:revive,gochecknoinits // CLI command initialization
	commentCmd.Flags().IntP("pr", "p", 0, "Pull request number (defaults to GITHUB_PR_NUMBER)")
	commentCmd.Flags().StringP("coverage", "c", "", "Coverage data file")
	commentCmd.Flags().String("badge-url", "", "Badge URL (auto-generated if not provided)")
	commentCmd.Flags().String("report-url", "", "Report URL (auto-generated if not provided)")
	commentCmd.Flags().Bool("status", false, "Also create commit status")
	commentCmd.Flags().Bool("dry-run", false, "Show what would be posted without actually posting")
}
