package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/coverage/internal/analysis"
	"github.com/mrz1836/go-broadcast/coverage/internal/badge"
	"github.com/mrz1836/go-broadcast/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/coverage/internal/github"
	"github.com/mrz1836/go-broadcast/coverage/internal/history"
	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
	"github.com/mrz1836/go-broadcast/coverage/internal/templates"
)

var (
	// ErrEnhancedGitHubTokenRequired indicates GitHub token was not provided
	ErrEnhancedGitHubTokenRequired = errors.New("GitHub token is required")
	// ErrEnhancedGitHubOwnerRequired indicates repository owner was not provided
	ErrEnhancedGitHubOwnerRequired = errors.New("GitHub repository owner is required")
	// ErrEnhancedGitHubRepoRequired indicates repository name was not provided
	ErrEnhancedGitHubRepoRequired  = errors.New("GitHub repository name is required")
	// ErrEnhancedPRNumberRequired indicates PR number was not provided
	ErrEnhancedPRNumberRequired    = errors.New("pull request number is required")
)

var commentEnhancedCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "comment-enhanced",
	Short: "Create enhanced PR coverage comment with analysis and templates",
	Long: `Create or update pull request comments with comprehensive coverage analysis,
comparison, dynamic templates, PR-specific badges, and status checks.

This enhanced version includes:
- Intelligent PR comment management with anti-spam features
- Coverage comparison and analysis between base and PR branches
- Dynamic template rendering with multiple template options
- PR-specific badge generation with unique naming
- GitHub status check integration for blocking PR merges
- Smart update logic and lifecycle management`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		prNumber, _ := cmd.Flags().GetInt("pr")
		inputFile, _ := cmd.Flags().GetString("coverage")
		baseCoverageFile, _ := cmd.Flags().GetString("base-coverage")
		badgeURL, _ := cmd.Flags().GetString("badge-url")
		reportURL, _ := cmd.Flags().GetString("report-url")
		templateName, _ := cmd.Flags().GetString("template")
		createStatus, _ := cmd.Flags().GetBool("status")
		blockOnFailure, _ := cmd.Flags().GetBool("block-merge")
		generateBadges, _ := cmd.Flags().GetBool("generate-badges")
		enableAnalysis, _ := cmd.Flags().GetBool("enable-analysis")
		compactMode, _ := cmd.Flags().GetBool("compact")
		antiSpam, _ := cmd.Flags().GetBool("anti-spam")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Load configuration
		cfg := config.Load()

		// Validate GitHub configuration
		if cfg.GitHub.Token == "" {
			return ErrEnhancedGitHubTokenRequired
		}
		if cfg.GitHub.Owner == "" {
			return ErrEnhancedGitHubOwnerRequired
		}
		if cfg.GitHub.Repository == "" {
			return ErrEnhancedGitHubRepoRequired
		}

		// Use PR number from config if not provided
		if prNumber == 0 {
			prNumber = cfg.GitHub.PullRequest
		}
		if prNumber == 0 {
			return ErrEnhancedPRNumberRequired
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
		_ = badgeURL  // TODO: Use in PR comment template
		_ = reportURL // TODO: Use in PR comment template
		if templateName == "" {
			if compactMode {
				templateName = "compact"
			} else {
				templateName = "comprehensive"
			}
		}

		// Parse current coverage data
		p := parser.New()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		coverage, err := p.ParseFile(ctx, inputFile)
		if err != nil {
			return fmt.Errorf("failed to parse coverage file: %w", err)
		}

		// Parse base coverage data for comparison (if provided)
		var baseCoverage *parser.CoverageData
		if baseCoverageFile != "" {
			baseCoverage, err = p.ParseFile(ctx, baseCoverageFile)
			if err != nil {
				fmt.Printf("Warning: failed to parse base coverage file: %v\n", err)
				baseCoverage = nil
			}
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
			UserAgent:  "gofortress-coverage/2.0",
		}
		client := github.NewWithConfig(githubConfig)

		// Initialize enhanced PR comment system
		prCommentConfig := &github.PRCommentConfig{
			MinUpdateIntervalMinutes: 5,
			MaxCommentsPerPR:         1,
			CommentSignature:         "gofortress-coverage-v2",
			IncludeTrend:             true,
			IncludeCoverageDetails:   true,
			IncludeFileAnalysis:      enableAnalysis,
			ShowCoverageHistory:      true,
			GeneratePRBadges:         generateBadges,
			EnableStatusChecks:       createStatus,
			FailBelowThreshold:       true,
			BlockMergeOnFailure:      blockOnFailure,
		}

		// Adjust settings for compact mode
		if compactMode {
			prCommentConfig.IncludeCoverageDetails = false
			prCommentConfig.IncludeFileAnalysis = false
			prCommentConfig.ShowCoverageHistory = false
		}

		// Adjust settings for anti-spam mode
		if antiSpam {
			prCommentConfig.MinUpdateIntervalMinutes = 15
			prCommentConfig.MaxCommentsPerPR = 1
		}

		prCommentManager := github.NewPRCommentManager(client, prCommentConfig)

		// Perform coverage comparison and analysis if base coverage is available
		var comparison *github.CoverageComparison
		if baseCoverage != nil && enableAnalysis {
			comparisonEngine := analysis.NewComparisonEngine(nil)

			// Convert parser data to comparison snapshots
			baseSnapshot := convertToSnapshot(baseCoverage, "main", "")
			prSnapshot := convertToSnapshot(coverage, "current", cfg.GitHub.CommitSHA)

			comparisonResult, compErr := comparisonEngine.CompareCoverage(ctx, baseSnapshot, prSnapshot)
			if compErr != nil {
				fmt.Printf("Warning: failed to perform coverage comparison: %v\n", compErr)
			} else {
				// Convert comparison result to PR comment format
				comparison = &github.CoverageComparison{
					BaseCoverage: github.CoverageData{
						Percentage:        baseCoverage.Percentage,
						TotalStatements:   baseCoverage.TotalLines,
						CoveredStatements: baseCoverage.CoveredLines,
						CommitSHA:         "",
						Branch:            "main",
						Timestamp:         time.Now(),
					},
					PRCoverage: github.CoverageData{
						Percentage:        coverage.Percentage,
						TotalStatements:   coverage.TotalLines,
						CoveredStatements: coverage.CoveredLines,
						CommitSHA:         cfg.GitHub.CommitSHA,
						Branch:            "current",
						Timestamp:         time.Now(),
					},
					Difference:       coverage.Percentage - baseCoverage.Percentage,
					TrendAnalysis:    convertTrendData(comparisonResult.TrendAnalysis),
					FileChanges:      convertFileChanges(comparisonResult.FileChanges),
					SignificantFiles: extractSignificantFiles(comparisonResult.FileChanges),
				}
			}
		}

		// Fall back to simple comparison if no base coverage or analysis disabled
		if comparison == nil {
			comparison = &github.CoverageComparison{
				PRCoverage: github.CoverageData{
					Percentage:        coverage.Percentage,
					TotalStatements:   coverage.TotalLines,
					CoveredStatements: coverage.CoveredLines,
					CommitSHA:         cfg.GitHub.CommitSHA,
					Branch:            "current",
					Timestamp:         time.Now(),
				},
				TrendAnalysis: github.TrendData{
					Direction:        trend,
					Magnitude:        "minor",
					PercentageChange: 0,
					Momentum:         "steady",
				},
			}
		}

		if dryRun {
			// Generate template preview for dry run
			templateEngine := templates.NewPRTemplateEngine(&templates.TemplateConfig{
				CompactMode:            compactMode,
				IncludeEmojis:          true,
				IncludeCharts:          true,
				MaxFileChanges:         20,
				MaxRecommendations:     5,
				UseMarkdownTables:      true,
				UseCollapsibleSections: true,
				IncludeProgressBars:    true,
				BrandingEnabled:        true,
			})

			templateData := buildTemplateData(cfg, prNumber, comparison, coverage)

			commentPreview, renderErr := templateEngine.RenderComment(ctx, templateName, templateData)
			if renderErr != nil {
				commentPreview = fmt.Sprintf("Error generating template preview: %v", renderErr)
			}

			fmt.Printf("Enhanced PR Comment Preview (Dry Run)\n")
			fmt.Printf("=====================================\n")
			fmt.Printf("Template: %s\n", templateName)
			fmt.Printf("PR: %d\n", prNumber)
			fmt.Printf("Repository: %s/%s\n", cfg.GitHub.Owner, cfg.GitHub.Repository)
			fmt.Printf("Coverage: %.2f%%\n", coverage.Percentage)
			if comparison.BaseCoverage.Percentage > 0 {
				fmt.Printf("Base Coverage: %.2f%%\n", comparison.BaseCoverage.Percentage)
				fmt.Printf("Difference: %+.2f%%\n", comparison.Difference)
			}
			fmt.Printf("Features enabled:\n")
			fmt.Printf("  - Analysis: %v\n", enableAnalysis)
			fmt.Printf("  - Status Checks: %v\n", createStatus)
			fmt.Printf("  - Badge Generation: %v\n", generateBadges)
			fmt.Printf("  - Merge Blocking: %v\n", blockOnFailure)
			fmt.Printf("  - Anti-spam: %v\n", antiSpam)
			fmt.Printf("=====================================\n")
			fmt.Println(commentPreview)
			fmt.Printf("=====================================\n")

			return nil
		}

		// Create or update enhanced PR comment
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		result, err := prCommentManager.CreateOrUpdatePRComment(ctx, cfg.GitHub.Owner, cfg.GitHub.Repository, prNumber, comparison)
		if err != nil {
			return fmt.Errorf("failed to create enhanced PR comment: %w", err)
		}

		fmt.Printf("Enhanced coverage comment %s successfully!\n", result.Action)
		fmt.Printf("Comment ID: %d\n", result.CommentID)
		fmt.Printf("Coverage: %.2f%%\n", comparison.PRCoverage.Percentage)
		if comparison.BaseCoverage.Percentage > 0 {
			fmt.Printf("Change: %+.2f%% vs base\n", comparison.Difference)
		}
		fmt.Printf("Action taken: %s (%s)\n", result.Action, result.Reason)

		// Generate PR-specific badges if requested
		if generateBadges {
			badgeGenerator := badge.New()
			prBadgeManager := badge.NewPRBadgeManager(badgeGenerator, nil)

			badgeRequest := &badge.PRBadgeRequest{
				Repository:   cfg.GitHub.Repository,
				Owner:        cfg.GitHub.Owner,
				PRNumber:     prNumber,
				Branch:       "current",
				CommitSHA:    cfg.GitHub.CommitSHA,
				BaseBranch:   "main",
				Coverage:     coverage.Percentage,
				BaseCoverage: comparison.BaseCoverage.Percentage,
				Trend:        determineBadgeTrend(comparison.TrendAnalysis.Direction),
				QualityGrade: calculateQualityGrade(coverage.Percentage),
				Types:        []badge.PRBadgeType{badge.PRBadgeCoverage, badge.PRBadgeTrend, badge.PRBadgeStatus},
				Timestamp:    time.Now(),
			}

			badgeResult, err := prBadgeManager.GenerateStandardPRBadges(ctx, badgeRequest)
			if err != nil {
				fmt.Printf("Warning: failed to generate PR badges: %v\n", err)
			} else {
				fmt.Printf("Generated %d PR-specific badges\n", badgeResult.TotalBadges)
				for badgeType, urls := range badgeResult.PublicURLs {
					if len(urls) > 0 {
						fmt.Printf("  %s: %s\n", badgeType, urls[0])
					}
				}
			}
		}

		// Create enhanced status checks if requested
		if createStatus && cfg.GitHub.CommitSHA != "" {
			statusManager := github.NewStatusCheckManager(client, nil)

			statusRequest := &github.StatusCheckRequest{
				Owner:      cfg.GitHub.Owner,
				Repository: cfg.GitHub.Repository,
				CommitSHA:  cfg.GitHub.CommitSHA,
				PRNumber:   prNumber,
				Branch:     "current",
				BaseBranch: "main",
				Coverage: github.CoverageStatusData{
					Percentage:        coverage.Percentage,
					TotalStatements:   coverage.TotalLines,
					CoveredStatements: coverage.CoveredLines,
					Change:            comparison.Difference,
					Trend:             comparison.TrendAnalysis.Direction,
				},
				Comparison: github.ComparisonStatusData{
					BasePercentage:    comparison.BaseCoverage.Percentage,
					CurrentPercentage: comparison.PRCoverage.Percentage,
					Difference:        comparison.Difference,
					IsSignificant:     comparison.Difference > 1.0 || comparison.Difference < -1.0,
					Direction:         comparison.TrendAnalysis.Direction,
				},
				Quality: github.QualityStatusData{
					Grade:     calculateQualityGrade(coverage.Percentage),
					Score:     coverage.Percentage,
					RiskLevel: calculateRiskLevel(coverage.Percentage),
				},
			}

			statusResult, err := statusManager.CreateStatusChecks(ctx, statusRequest)
			if err != nil {
				fmt.Printf("Warning: failed to create status checks: %v\n", err)
			} else {
				fmt.Printf("Created %d status checks\n", statusResult.TotalChecks)
				fmt.Printf("Passed: %d, Failed: %d, Errors: %d\n",
					statusResult.PassedChecks, statusResult.FailedChecks, statusResult.ErrorChecks)
				if statusResult.BlockingPR {
					fmt.Printf("⚠️ PR merge is blocked due to failed required checks\n")
				}
				if len(statusResult.RequiredFailed) > 0 {
					fmt.Printf("Failed required checks: %v\n", statusResult.RequiredFailed)
				}
			}
		}

		return nil
	},
}

// Helper functions for converting data structures

func convertToSnapshot(coverage *parser.CoverageData, branch, commitSHA string) *analysis.CoverageSnapshot {
	return &analysis.CoverageSnapshot{
		Branch:    branch,
		CommitSHA: commitSHA,
		Timestamp: time.Now(),
		OverallCoverage: analysis.CoverageMetrics{
			Percentage:        coverage.Percentage,
			TotalStatements:   coverage.TotalLines,
			CoveredStatements: coverage.CoveredLines,
			TotalLines:        coverage.TotalLines, // Approximation
			CoveredLines:      coverage.CoveredLines,
		},
		FileCoverage:    make(map[string]analysis.FileMetrics),
		PackageCoverage: make(map[string]analysis.PackageMetrics),
		TestMetadata: analysis.TestMetadata{
			TestDuration: 0,
			TestCount:    0,
		},
	}
}

func convertTrendData(trend analysis.TrendAnalysis) github.TrendData {
	return github.TrendData{
		Direction:        trend.Direction,
		Magnitude:        "minor", // Simplified
		PercentageChange: 0,       // Would need calculation
		Momentum:         trend.Momentum,
	}
}

func convertFileChanges(changes []analysis.FileChangeAnalysis) []github.FileChange {
	fileChanges := make([]github.FileChange, 0, len(changes))
	for _, change := range changes {
		fileChanges = append(fileChanges, github.FileChange{
			Filename:      change.Filename,
			BaseCoverage:  change.BasePercentage,
			PRCoverage:    change.PRPercentage,
			Difference:    change.PercentageChange,
			LinesAdded:    change.LinesAdded,
			LinesRemoved:  change.LinesRemoved,
			IsSignificant: change.IsSignificant,
		})
	}
	return fileChanges
}

func extractSignificantFiles(changes []analysis.FileChangeAnalysis) []string {
	var significantFiles []string
	for _, change := range changes {
		if change.IsSignificant {
			significantFiles = append(significantFiles, change.Filename)
		}
	}
	return significantFiles
}

func buildTemplateData(cfg *config.Config, prNumber int, comparison *github.CoverageComparison, coverage *parser.CoverageData) *templates.TemplateData {
	return &templates.TemplateData{
		Repository: templates.RepositoryInfo{
			Owner:         cfg.GitHub.Owner,
			Name:          cfg.GitHub.Repository,
			DefaultBranch: "main",
			URL:           fmt.Sprintf("https://github.com/%s/%s", cfg.GitHub.Owner, cfg.GitHub.Repository),
		},
		PullRequest: templates.PullRequestInfo{
			Number:     prNumber,
			Title:      "",
			Branch:     "current",
			BaseBranch: "main",
			Author:     "",
			CommitSHA:  cfg.GitHub.CommitSHA,
			URL:        fmt.Sprintf("https://github.com/%s/%s/pull/%d", cfg.GitHub.Owner, cfg.GitHub.Repository, prNumber),
		},
		Timestamp: time.Now(),
		Coverage: templates.CoverageData{
			Overall: templates.CoverageMetrics{
				Percentage:        comparison.PRCoverage.Percentage,
				TotalStatements:   comparison.PRCoverage.TotalStatements,
				CoveredStatements: comparison.PRCoverage.CoveredStatements,
				Grade:             calculateQualityGrade(comparison.PRCoverage.Percentage),
				Status:            calculateCoverageStatus(comparison.PRCoverage.Percentage),
			},
			Summary: templates.CoverageSummary{
				Direction:     comparison.TrendAnalysis.Direction,
				Magnitude:     comparison.TrendAnalysis.Magnitude,
				OverallImpact: determineOverallImpact(comparison.Difference),
			},
		},
		Comparison: templates.ComparisonData{
			BasePercentage:    comparison.BaseCoverage.Percentage,
			CurrentPercentage: comparison.PRCoverage.Percentage,
			Change:            comparison.Difference,
			Direction:         comparison.TrendAnalysis.Direction,
			Magnitude:         comparison.TrendAnalysis.Magnitude,
			IsSignificant:     comparison.Difference > 1.0 || comparison.Difference < -1.0,
		},
		Quality: templates.QualityData{
			OverallGrade:  calculateQualityGrade(comparison.PRCoverage.Percentage),
			CoverageGrade: calculateQualityGrade(comparison.PRCoverage.Percentage),
			TrendGrade:    calculateTrendGrade(comparison.TrendAnalysis.Direction),
			RiskLevel:     calculateRiskLevel(comparison.PRCoverage.Percentage),
			Score:         comparison.PRCoverage.Percentage,
		},
	}
}

func calculateQualityGrade(percentage float64) string {
	switch {
	case percentage >= 95:
		return "A+"
	case percentage >= 90:
		return "A"
	case percentage >= 85:
		return "B+"
	case percentage >= 80:
		return "B"
	case percentage >= 70:
		return "C"
	case percentage >= 60:
		return "D"
	default:
		return "F"
	}
}

func calculateCoverageStatus(percentage float64) string {
	switch {
	case percentage >= 90:
		return "excellent"
	case percentage >= 80:
		return "good"
	case percentage >= 70:
		return "warning"
	default:
		return "critical"
	}
}

func calculateRiskLevel(percentage float64) string {
	switch {
	case percentage >= 80:
		return "low"
	case percentage >= 60:
		return "medium"
	case percentage >= 40:
		return "high"
	default:
		return "critical"
	}
}

func calculateTrendGrade(direction string) string {
	switch direction {
	case "up", "improved":
		return "A"
	case "stable":
		return "B"
	case "down", "degraded":
		return "D"
	default:
		return "C"
	}
}

func determineOverallImpact(difference float64) string {
	if difference > 1.0 {
		return "positive"
	} else if difference < -1.0 {
		return "negative"
	}
	return "neutral"
}

func determineBadgeTrend(direction string) badge.TrendDirection {
	switch strings.ToLower(direction) {
	case "up", "improved":
		return badge.TrendUp
	case "down", "degraded":
		return badge.TrendDown
	default:
		return badge.TrendStable
	}
}

func init() { //nolint:revive,gochecknoinits // CLI command initialization
	commentEnhancedCmd.Flags().IntP("pr", "p", 0, "Pull request number (defaults to GITHUB_PR_NUMBER)")
	commentEnhancedCmd.Flags().StringP("coverage", "c", "", "Coverage data file")
	commentEnhancedCmd.Flags().String("base-coverage", "", "Base coverage data file for comparison")
	commentEnhancedCmd.Flags().String("badge-url", "", "Badge URL (auto-generated if not provided)")
	commentEnhancedCmd.Flags().String("report-url", "", "Report URL (auto-generated if not provided)")
	commentEnhancedCmd.Flags().String("template", "", "Comment template (comprehensive, compact, detailed, summary, minimal)")
	commentEnhancedCmd.Flags().Bool("status", false, "Create enhanced status checks")
	commentEnhancedCmd.Flags().Bool("block-merge", false, "Block PR merge on coverage failure")
	commentEnhancedCmd.Flags().Bool("generate-badges", false, "Generate PR-specific badges")
	commentEnhancedCmd.Flags().Bool("enable-analysis", true, "Enable detailed coverage analysis and comparison")
	commentEnhancedCmd.Flags().Bool("compact", false, "Use compact mode (shorter comments)")
	commentEnhancedCmd.Flags().Bool("anti-spam", true, "Enable anti-spam features")
	commentEnhancedCmd.Flags().Bool("dry-run", false, "Show preview of enhanced comment without posting")
}
