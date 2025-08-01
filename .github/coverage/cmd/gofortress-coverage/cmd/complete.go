// Package cmd provides CLI commands for the GoFortress coverage tool
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/coverage/internal/analytics/dashboard"
	"github.com/mrz1836/go-broadcast/coverage/internal/analytics/report"
	"github.com/mrz1836/go-broadcast/coverage/internal/badge"
	"github.com/mrz1836/go-broadcast/coverage/internal/config"
	"github.com/mrz1836/go-broadcast/coverage/internal/github"
	"github.com/mrz1836/go-broadcast/coverage/internal/history"
	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

// getDefaultBranch returns the default branch name, checking environment variables first
func getDefaultBranch() string {
	if branch := os.Getenv("GITHUB_REF_NAME"); branch != "" {
		return branch
	}
	// Default to master (this repository's default branch)
	return history.DefaultBranch
}

// ErrCoverageBelowThreshold indicates that coverage percentage is below the configured threshold
var ErrCoverageBelowThreshold = errors.New("coverage is below threshold")

// ErrEmptyIndexHTML indicates that the generated index.html file is empty
var ErrEmptyIndexHTML = errors.New("generated index.html is empty")

var completeCmd = &cobra.Command{ //nolint:gochecknoglobals // CLI command
	Use:   "complete",
	Short: "Run complete coverage pipeline",
	Long: `Run the complete coverage pipeline: parse coverage, generate badge and report, 
update history, and create GitHub PR comment if in PR context.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
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

		// Create output directory structure for GitHub Pages
		// Structure depends on context:
		// - Branch: outputDir/reports/branch/{branchName}/
		// - PR: outputDir/pr/{prNumber}/
		branch := getDefaultBranch()
		var targetOutputDir string
		if cfg.IsPullRequestContext() {
			// PR context: outputDir/pr/{prNumber}/
			targetOutputDir = filepath.Join(outputDir, "pr", fmt.Sprintf("%d", cfg.GitHub.PullRequest))
		} else {
			// Branch context: outputDir/reports/branch/{branchName}/
			targetOutputDir = filepath.Join(outputDir, "reports", "branch", branch)
		}

		if cfg.Storage.AutoCreate && !dryRun {
			// Create the full directory structure
			if mkdirErr := os.MkdirAll(targetOutputDir, cfg.Storage.DirMode); mkdirErr != nil {
				return fmt.Errorf("failed to create output directory structure: %w", mkdirErr)
			}
			// Also ensure root output directory exists for root index.html
			if mkdirErr := os.MkdirAll(outputDir, cfg.Storage.DirMode); mkdirErr != nil {
				return fmt.Errorf("failed to create root output directory: %w", mkdirErr)
			}
		}

		// Step 2: Generate badge
		cmd.Printf("🏷️  Step 2: Generating coverage badge...\n")
		// Badge goes in target directory and also at root for easy access
		badgeFile := filepath.Join(targetOutputDir, cfg.Badge.OutputFile)
		rootBadgeFile := filepath.Join(outputDir, cfg.Badge.OutputFile)

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
			// Also write badge to root for easy access
			if writeErr := os.WriteFile(rootBadgeFile, svgContent, cfg.Storage.FileMode); writeErr != nil {
				cmd.Printf("   ⚠️  Failed to write root badge file: %v\n", writeErr)
			}
		}

		cmd.Printf("   ✅ Badge saved: %s\n", badgeFile)
		cmd.Printf("\n")

		// Step 3: Generate HTML report
		cmd.Printf("📊 Step 3: Generating HTML report...\n")

		reportConfig := &report.Config{
			OutputDir:       targetOutputDir,
			RepositoryOwner: cfg.GitHub.Owner,
			RepositoryName:  cfg.GitHub.Repository,
			BranchName:      getDefaultBranch(),
			CommitSHA:       cfg.GitHub.CommitSHA,
		}

		reportGen := report.NewGenerator(reportConfig)
		ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if !dryRun {
			if reportErr := reportGen.Generate(ctx, coverage); reportErr != nil {
				return fmt.Errorf("failed to generate report: %w", reportErr)
			}
		}

		cmd.Printf("   ✅ Report saved: %s/coverage.html\n", targetOutputDir)
		cmd.Printf("\n")

		// Step 4: Generate dashboard
		cmd.Printf("🎯 Step 4: Generating coverage dashboard...\n")

		// Prepare coverage data for dashboard
		// branch already declared earlier

		coverageData := &dashboard.CoverageData{
			ProjectName:    cfg.Report.Title,
			RepositoryURL:  fmt.Sprintf("https://github.com/%s/%s", cfg.GitHub.Owner, cfg.GitHub.Repository),
			Branch:         branch,
			CommitSHA:      cfg.GitHub.CommitSHA,
			PRNumber:       "",
			Timestamp:      time.Now(),
			TotalCoverage:  coverage.Percentage,
			TotalLines:     coverage.TotalLines,
			CoveredLines:   coverage.CoveredLines,
			MissedLines:    coverage.TotalLines - coverage.CoveredLines,
			TotalFiles:     0,
			CoveredFiles:   0,
			PartialFiles:   0,
			UncoveredFiles: 0,
		}

		// Detect workflow run context
		if runNumberStr := os.Getenv("GITHUB_RUN_NUMBER"); runNumberStr != "" {
			if runNumber, parseErr := strconv.Atoi(runNumberStr); parseErr == nil {
				coverageData.WorkflowRunNumber = runNumber
				// Consider it the first run if run number is 1-3 (allowing for a few initial failures)
				coverageData.IsFirstRun = runNumber <= 3
				coverageData.HasPreviousRuns = runNumber > 1
				cmd.Printf("   📊 Workflow run #%d detected\n", runNumber)
				if coverageData.IsFirstRun {
					cmd.Printf("   🚀 This appears to be one of the first workflow runs\n")
				}
			}
		}

		// Discover all eligible Go files to get accurate total count
		// Get repository root path - we're in .github/coverage/cmd/gofortress-coverage
		workingDir, wdErr := os.Getwd()
		if wdErr != nil {
			cmd.Printf("   ⚠️  Failed to get working directory: %v\n", wdErr)
		}
		repoRoot := filepath.Join(workingDir, "../../../../")
		repoRoot, pathErr := filepath.Abs(repoRoot)
		if pathErr != nil {
			cmd.Printf("   ⚠️  Failed to resolve repository root: %v\n", pathErr)
			repoRoot = "../../../../"
		}

		eligibleFiles, err := p.DiscoverEligibleFiles(ctx, repoRoot)
		if err != nil {
			cmd.Printf("   ⚠️  Failed to discover all Go files: %v\n", err)
			// Fall back to counting only files in coverage data
			totalFiles := 0
			for _, pkg := range coverage.Packages {
				totalFiles += len(pkg.Files)
			}
			coverageData.TotalFiles = totalFiles
		} else {
			coverageData.TotalFiles = len(eligibleFiles)
		}

		// Count coverage status for files that have coverage data
		// Any file with >0% coverage is considered "covered"
		filesInProfile := 0
		for _, pkg := range coverage.Packages {
			for _, file := range pkg.Files {
				filesInProfile++
				if file.Percentage > 0 {
					// Any coverage > 0% counts as "covered"
					coverageData.CoveredFiles++
				} else {
					// 0% coverage files in profile are uncovered
					coverageData.UncoveredFiles++
				}
			}
		}

		// Files not in coverage profile are considered uncovered
		if coverageData.TotalFiles > filesInProfile {
			additionalUncovered := coverageData.TotalFiles - filesInProfile
			coverageData.UncoveredFiles += additionalUncovered
		}

		// Debug output for file counting
		cmd.Printf("   📊 File Analysis:\n")
		cmd.Printf("      Total eligible files: %d\n", coverageData.TotalFiles)
		cmd.Printf("      Files in coverage profile: %d\n", filesInProfile)
		cmd.Printf("      Files with coverage >0%%: %d\n", coverageData.CoveredFiles)
		cmd.Printf("      Files with no coverage: %d\n", coverageData.UncoveredFiles)

		// Add package data
		coverageData.Packages = make([]dashboard.PackageCoverage, 0, len(coverage.Packages))
		for pkgName, pkg := range coverage.Packages {
			pkgCoverage := dashboard.PackageCoverage{
				Name:         pkgName,
				Path:         pkgName, // Use package name as path for now
				Coverage:     pkg.Percentage,
				TotalLines:   pkg.TotalLines,
				CoveredLines: pkg.CoveredLines,
				MissedLines:  pkg.TotalLines - pkg.CoveredLines,
			}

			// Add GitHub URL for package directory if we have GitHub info
			if cfg.GitHub.Owner != "" && cfg.GitHub.Repository != "" {
				pkgCoverage.GitHubURL = fmt.Sprintf("https://github.com/%s/%s/tree/%s/%s",
					cfg.GitHub.Owner, cfg.GitHub.Repository, branch, pkgName)
			}

			// Add file coverage if available
			if pkg.Files != nil {
				pkgCoverage.Files = make([]dashboard.FileCoverage, 0, len(pkg.Files))
				for fileName, file := range pkg.Files {
					fileCoverage := dashboard.FileCoverage{
						Name:         filepath.Base(fileName),
						Path:         fileName,
						Coverage:     file.Percentage,
						TotalLines:   file.TotalLines,
						CoveredLines: file.CoveredLines,
						MissedLines:  file.TotalLines - file.CoveredLines,
					}
					if cfg.GitHub.Owner != "" && cfg.GitHub.Repository != "" {
						fileCoverage.GitHubURL = fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s",
							cfg.GitHub.Owner, cfg.GitHub.Repository, branch, fileName)
					}
					pkgCoverage.Files = append(pkgCoverage.Files, fileCoverage)
				}
			}

			coverageData.Packages = append(coverageData.Packages, pkgCoverage)
		}

		// Set PR number if in PR context
		if cfg.IsPullRequestContext() {
			coverageData.PRNumber = fmt.Sprintf("%d", cfg.GitHub.PullRequest)
		}

		// Populate history data for dashboard
		// Always try to load history for display, even if history tracking is disabled
		// This ensures trends are shown when history data exists from previous runs
		{
			// branch already declared at function level

			// Resolve absolute path for history storage (same logic as Step 5)
			dashboardHistoryPath := cfg.History.StoragePath
			if resolvedPath, err := cfg.ResolveHistoryStoragePath(); err == nil {
				dashboardHistoryPath = resolvedPath
			}

			// Initialize history tracker to get historical data
			historyConfig := &history.Config{
				StoragePath:    dashboardHistoryPath,
				RetentionDays:  cfg.History.RetentionDays,
				MaxEntries:     cfg.History.MaxEntries,
				AutoCleanup:    false, // Don't cleanup when just reading for display
				MetricsEnabled: false, // Don't track metrics when just reading
			}
			tracker := history.NewWithConfig(historyConfig)

			// Get historical data for trends
			historyCtx, historyCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer historyCancel()

			trendData, err := tracker.GetTrend(historyCtx, history.WithTrendBranch(branch), history.WithTrendDays(30))

			// If no history for current branch and it's not master, try to get master branch history
			if (err != nil || trendData == nil || trendData.Summary.TotalEntries == 0) && branch != "master" {
				cmd.Printf("   📊 No history for branch '%s', checking master branch...\n", branch)
				if masterTrendData, masterErr := tracker.GetTrend(historyCtx, history.WithTrendBranch("master"), history.WithTrendDays(30)); masterErr == nil && masterTrendData != nil {
					// Use master branch data for comparison
					trendData = masterTrendData
					cmd.Printf("   ✅ Found %d history entries from master branch\n", trendData.Summary.TotalEntries)
				}
			}

			if err == nil && trendData != nil {
				// Populate trend data if we have enough entries
				if trendData.Summary.TotalEntries > 1 {
					// Use short-term trend analysis if available
					changePercent := 0.0
					direction := trendData.Summary.CurrentTrend
					if trendData.Analysis != nil && trendData.Analysis.ShortTermTrend != nil {
						changePercent = trendData.Analysis.ShortTermTrend.ChangePercent
						direction = trendData.Analysis.ShortTermTrend.Direction
					}

					coverageData.TrendData = &dashboard.TrendData{
						Direction:     direction,
						ChangePercent: changePercent,
						ChangeLines:   int(changePercent * float64(coverage.TotalLines) / 100),
					}
				}

				// Populate historical points from entries
				if len(trendData.Entries) > 0 {
					coverageData.History = make([]dashboard.HistoricalPoint, 0, len(trendData.Entries))
					for _, entry := range trendData.Entries {
						if entry.Coverage != nil {
							coverageData.History = append(coverageData.History, dashboard.HistoricalPoint{
								Timestamp:    entry.Timestamp,
								CommitSHA:    entry.CommitSHA,
								Coverage:     entry.Coverage.Percentage,
								TotalLines:   entry.Coverage.TotalLines,
								CoveredLines: entry.Coverage.CoveredLines,
							})
						}
					}
				}
			}

			cmd.Printf("   📊 History data loaded: %d entries, trend: %s\n",
				len(coverageData.History),
				func() string {
					if coverageData.TrendData != nil {
						return coverageData.TrendData.Direction
					}
					return "none"
				}())
		}

		// Generate dashboard
		dashboardConfig := &dashboard.GeneratorConfig{
			ProjectName:      cfg.Report.Title,
			RepositoryOwner:  cfg.GitHub.Owner,
			RepositoryName:   cfg.GitHub.Repository,
			OutputDir:        targetOutputDir, // Dashboard goes in target directory
			GeneratorVersion: "1.0.0",
			GitHubToken:      cfg.GitHub.Token,
		}

		dashboardGen := dashboard.NewGenerator(dashboardConfig)
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if !dryRun {
			if err := dashboardGen.Generate(ctx, coverageData); err != nil {
				cmd.Printf("   ⚠️  Failed to generate dashboard: %v\n", err)
			} else {
				cmd.Printf("   ✅ Dashboard saved: %s/index.html\n", targetOutputDir)

				// Also create dashboard.html for GitHub Pages deployment compatibility
				indexPath := filepath.Join(targetOutputDir, "index.html")
				dashboardPath := filepath.Join(targetOutputDir, "dashboard.html")

				// Verify index.html was created successfully
				if _, statErr := os.Stat(indexPath); statErr != nil {
					cmd.Printf("   ❌ index.html was not created successfully: %v\n", statErr)
					return fmt.Errorf("index.html generation failed: %w", statErr)
				}

				// Read the generated index.html and copy it to dashboard.html
				indexContent, readErr := os.ReadFile(indexPath) //nolint:gosec // path is constructed from validated config
				if readErr != nil {
					cmd.Printf("   ❌ Failed to read index.html for dashboard.html creation: %v\n", readErr)
					return fmt.Errorf("failed to read generated index.html: %w", readErr)
				}

				if len(indexContent) == 0 {
					cmd.Printf("   ❌ index.html is empty, cannot create dashboard.html\n")
					return ErrEmptyIndexHTML
				}

				if writeErr := os.WriteFile(dashboardPath, indexContent, cfg.Storage.FileMode); writeErr != nil {
					cmd.Printf("   ❌ Failed to create dashboard.html: %v\n", writeErr)
					return fmt.Errorf("failed to create dashboard.html: %w", writeErr)
				}

				// Verify dashboard.html was created successfully
				dashboardStat, statErr := os.Stat(dashboardPath)
				if statErr != nil {
					cmd.Printf("   ❌ dashboard.html was not created successfully: %v\n", statErr)
					return fmt.Errorf("dashboard.html creation verification failed: %w", statErr)
				}
				cmd.Printf("   ✅ Dashboard also saved as: %s (%d bytes)\n", dashboardPath, dashboardStat.Size())

				// Also save coverage data as JSON for pages deployment
				dataPath := filepath.Join(outputDir, "coverage-data.json")
				jsonData, err := json.Marshal(coverageData)
				if err != nil {
					cmd.Printf("   ⚠️  Failed to marshal coverage data: %v\n", err)
				}
				if err == nil && len(jsonData) > 0 {
					if err := os.WriteFile(dataPath, jsonData, cfg.Storage.FileMode); err != nil {
						cmd.Printf("   ⚠️  Failed to save coverage data: %v\n", err)
					}
				}
			}
		} else {
			cmd.Printf("   📊 Would generate dashboard at: %s/index.html\n", outputDir)
			cmd.Printf("   📊 Would also create: %s/dashboard.html\n", outputDir)
		}

		cmd.Printf("\n")

		// Step 5: Update history (if enabled)
		trend := "stable"
		cmd.Printf("📈 Step 5: Coverage history analysis...\n")
		cmd.Printf("   🔍 History enabled: %t\n", cfg.History.Enabled)
		cmd.Printf("   🔍 Skip history flag: %t\n", skipHistory)
		cmd.Printf("   🔍 History storage path: %s\n", cfg.History.StoragePath)

		if cfg.History.Enabled && !skipHistory {
			cmd.Printf("   📊 Proceeding with history update...\n")

			// Resolve absolute path for history storage to fix working directory issues
			historyStoragePath, pathErr := cfg.ResolveHistoryStoragePath()
			if pathErr != nil {
				cmd.Printf("   ⚠️  Failed to resolve history storage path: %v\n", pathErr)
				return fmt.Errorf("failed to resolve history storage path: %w", pathErr)
			}

			if historyStoragePath != cfg.History.StoragePath {
				cmd.Printf("   🔧 Resolved history path: %s -> %s\n", cfg.History.StoragePath, historyStoragePath)
			}

			historyConfig := &history.Config{
				StoragePath:    historyStoragePath,
				RetentionDays:  cfg.History.RetentionDays,
				MaxEntries:     cfg.History.MaxEntries,
				AutoCleanup:    cfg.History.AutoCleanup,
				MetricsEnabled: cfg.History.MetricsEnabled,
			}
			tracker := history.NewWithConfig(historyConfig)

			// Debug: Check if history directory exists and is writable
			if dirInfo, dirErr := os.Stat(historyStoragePath); dirErr != nil {
				cmd.Printf("   ⚠️  History directory check failed: %v\n", dirErr)
				cmd.Printf("   🔧 Attempting to create history directory: %s\n", historyStoragePath)
				if mkdirErr := os.MkdirAll(historyStoragePath, 0o750); mkdirErr != nil {
					cmd.Printf("   ❌ Failed to create history directory: %v\n", mkdirErr)
					return fmt.Errorf("failed to create history directory: %w", mkdirErr)
				}
				cmd.Printf("   ✅ History directory created: %s\n", historyStoragePath)
			} else {
				cmd.Printf("   ✅ History directory exists: %s (%s, %v)\n", historyStoragePath, dirInfo.Mode(), dirInfo.IsDir())
			}

			// Debug: List existing history files before adding new entry
			if historyFiles, err := filepath.Glob(filepath.Join(historyStoragePath, "*.json")); err == nil {
				cmd.Printf("   📊 Existing history entries: %d\n", len(historyFiles))
				if len(historyFiles) > 0 {
					cmd.Printf("   📝 Recent entries:\n")
					for i, file := range historyFiles {
						if i >= 3 { // Show only first 3 entries
							break
						}
						cmd.Printf("      - %s\n", filepath.Base(file))
					}
				}
			} else {
				cmd.Printf("   ⚠️  Failed to list history files: %v\n", err)
			}

			// Get trend before adding new entry
			// branch already declared at function level
			cmd.Printf("   🌿 Using branch: %s\n", branch)

			if latest, err := tracker.GetLatestEntry(ctx, branch); err == nil {
				commitDisplay := latest.CommitSHA
				if len(commitDisplay) > 8 {
					commitDisplay = commitDisplay[:8]
				}
				cmd.Printf("   📊 Previous coverage: %.2f%% (commit: %s)\n", latest.Coverage.Percentage, commitDisplay)
				if coverage.Percentage > latest.Coverage.Percentage {
					trend = "up"
					cmd.Printf("   📈 Trend: UP (+%.2f%%)\n", coverage.Percentage-latest.Coverage.Percentage)
				} else if coverage.Percentage < latest.Coverage.Percentage {
					trend = "down"
					cmd.Printf("   📉 Trend: DOWN (%.2f%%)\n", coverage.Percentage-latest.Coverage.Percentage)
				} else {
					cmd.Printf("   ➡️  Trend: STABLE (no change)\n")
				}
			} else {
				cmd.Printf("   🚀 No previous entry found (first run or new branch): %v\n", err)
			}

			// Add new entry
			if !dryRun {
				cmd.Printf("   📝 Recording new history entry...\n")
				var historyOptions []history.Option
				historyOptions = append(historyOptions, history.WithBranch(branch))
				cmd.Printf("   🔧 Branch: %s\n", branch)

				if cfg.GitHub.CommitSHA != "" {
					historyOptions = append(historyOptions, history.WithCommit(cfg.GitHub.CommitSHA, ""))
					cmd.Printf("   🔧 Commit SHA: %s\n", cfg.GitHub.CommitSHA)
				} else {
					cmd.Printf("   ⚠️  No commit SHA available\n")
				}

				if cfg.GitHub.Owner != "" {
					projectName := cfg.GitHub.Owner + "/" + cfg.GitHub.Repository
					historyOptions = append(historyOptions,
						history.WithMetadata("project", projectName))
					cmd.Printf("   🔧 Project: %s\n", projectName)
				} else {
					cmd.Printf("   ⚠️  No GitHub owner/repository info available\n")
				}

				cmd.Printf("   💾 Coverage data: %.2f%% (%d/%d lines)\n", coverage.Percentage, coverage.CoveredLines, coverage.TotalLines)

				if err := tracker.Record(ctx, coverage, historyOptions...); err != nil {
					cmd.Printf("   ❌ Failed to record history: %v\n", err)
					return fmt.Errorf("failed to record coverage history: %w", err)
				}

				cmd.Printf("   ✅ History entry recorded successfully\n")

				// Verify the entry was actually written
				if historyFiles, err := filepath.Glob(filepath.Join(historyStoragePath, "*.json")); err == nil {
					cmd.Printf("   📊 Total history entries after recording: %d\n", len(historyFiles))
					if len(historyFiles) > 0 {
						cmd.Printf("   📁 History files are located at: %s\n", historyStoragePath)
					}
				} else {
					cmd.Printf("   ⚠️  Failed to verify history files: %v\n", err)
				}
			} else {
				cmd.Printf("   🧪 DRY RUN: Would record history entry for branch %s\n", branch)
			}

			cmd.Printf("   ✅ History update completed (trend: %s)\n", trend)
			cmd.Printf("\n")
		} else {
			if !cfg.History.Enabled {
				cmd.Printf("   ℹ️  History tracking is disabled in configuration\n")
			}
			if skipHistory {
				cmd.Printf("   ℹ️  History tracking skipped by --skip-history flag\n")
			}
			cmd.Printf("   📈 Coverage history step skipped\n\n")
		}

		// Step 6: GitHub integration (if in GitHub context)
		if cfg.IsGitHubContext() && !skipGitHub {
			cmd.Printf("🐙 Step 6: GitHub integration...\n")

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

				// Create PR comment if in PR context - this is deprecated in favor of the comment command
				if cfg.IsPullRequestContext() && cfg.GitHub.PostComments {
					cmd.Printf("   ℹ️  PR comment creation is deprecated in complete command\n")
					cmd.Printf("   💡 Use 'gofortress-coverage comment' command for advanced PR comments\n")
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
			cmd.Printf("🐙 Step 6: GitHub integration (skipped)\n\n")
		}

		// Step 7: Copy critical files to root for GitHub Actions validation
		if !dryRun {
			cmd.Printf("📋 Step 7: Copying critical files to root output directory...\n")

			// Files to copy from target directory to root
			filesToCopy := []struct {
				filename string
				source   string
			}{
				{"index.html", filepath.Join(targetOutputDir, "index.html")},
				{"dashboard.html", filepath.Join(targetOutputDir, "dashboard.html")},
				{"coverage.html", filepath.Join(targetOutputDir, cfg.Report.OutputFile)},
			}

			for _, file := range filesToCopy {
				sourceFile := file.source
				destFile := filepath.Join(outputDir, file.filename)

				// Read source file
				content, err := os.ReadFile(sourceFile) //nolint:gosec // sourceFile is constructed from validated config paths
				if err != nil {
					cmd.Printf("   ⚠️  Failed to read %s: %v\n", file.filename, err)
					continue
				}

				// Write to root output directory
				if err := os.WriteFile(destFile, content, cfg.Storage.FileMode); err != nil {
					cmd.Printf("   ⚠️  Failed to copy %s to root: %v\n", file.filename, err)
				} else {
					cmd.Printf("   ✅ Copied %s to root output directory\n", file.filename)
				}
			}

			// Create root index.html redirect only if index.html copy failed and we're on master
			rootIndexPath := filepath.Join(outputDir, "index.html")
			if _, err := os.Stat(rootIndexPath); os.IsNotExist(err) && branch == "master" && !cfg.IsPullRequestContext() {
				cmd.Printf("   ℹ️  Creating fallback redirect for master branch\n")
				redirectHTML := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Coverage Report - Redirecting...</title>
    <meta http-equiv="refresh" content="0; url=reports/branch/master/">
    <script>window.location.href = "reports/branch/master/";</script>
</head>
<body>
    <p>Redirecting to <a href="reports/branch/master/">coverage report</a>...</p>
</body>
</html>`
				if err := os.WriteFile(rootIndexPath, []byte(redirectHTML), cfg.Storage.FileMode); err != nil {
					cmd.Printf("   ⚠️  Failed to create fallback root index.html: %v\n", err)
				} else {
					cmd.Printf("   ✅ Fallback root index.html redirect created\n")
				}
			}
			cmd.Printf("\n")
		}

		// Final summary
		cmd.Printf("✨ Pipeline Complete!\n")
		cmd.Printf("==================\n")
		cmd.Printf("Coverage: %.2f%% (%s)\n", coverage.Percentage,
			getStatusIcon(coverage.Percentage, cfg.Coverage.Threshold))
		cmd.Printf("Badge: %s\n", badgeFile)
		cmd.Printf("Report: %s/coverage.html\n", targetOutputDir)

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
