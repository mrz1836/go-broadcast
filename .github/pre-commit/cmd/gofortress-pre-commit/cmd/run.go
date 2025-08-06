package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/git"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/output"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/runner"
)

//nolint:gochecknoglobals // Required by cobra
var (
	allFiles            bool
	files               []string
	skipChecks          []string
	onlyChecks          []string
	parallel            int
	failFast            bool
	showVersion         bool
	gracefulDegradation bool
	showProgress        bool
	quiet               bool
)

// runCmd represents the run command
//
//nolint:gochecknoglobals // Required by cobra
var runCmd = &cobra.Command{
	Use:   "run [check-name] [flags] [files...]",
	Short: "Run pre-commit checks",
	Long: `Run pre-commit checks on your code.

By default, runs all enabled checks on files staged for commit.
You can specify individual checks to run, or provide specific files to check.

Available checks:
  fumpt      - Format code with gofumpt
  lint       - Run golangci-lint
  mod-tidy   - Ensure go.mod and go.sum are tidy
  whitespace - Fix trailing whitespace
  eof        - Ensure files end with newline`,
	Example: `  # Run all checks on staged files
  gofortress-pre-commit run

  # Run specific check on staged files
  gofortress-pre-commit run lint

  # Run all checks on all files
  gofortress-pre-commit run --all-files

  # Run checks on specific files
  gofortress-pre-commit run --files main.go,utils.go

  # Skip specific checks
  gofortress-pre-commit run --skip lint,fumpt

  # Run only specific checks
  gofortress-pre-commit run --only whitespace,eof`,
	RunE: runChecks,
}

//nolint:gochecknoinits // Required by cobra
func init() {
	runCmd.Flags().BoolVarP(&allFiles, "all-files", "a", false, "Run on all files in the repository")
	runCmd.Flags().StringSliceVarP(&files, "files", "f", nil, "Specific files to check")
	runCmd.Flags().StringSliceVar(&skipChecks, "skip", nil, "Skip specific checks")
	runCmd.Flags().StringSliceVar(&onlyChecks, "only", nil, "Run only specific checks")
	runCmd.Flags().IntVarP(&parallel, "parallel", "p", 0, "Number of parallel workers (0 = auto)")
	runCmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first check failure")
	runCmd.Flags().BoolVar(&showVersion, "show-checks", false, "Show available checks and exit")
	runCmd.Flags().BoolVar(&gracefulDegradation, "graceful", false, "Skip checks that can't run instead of failing")
	runCmd.Flags().BoolVar(&showProgress, "progress", true, "Show progress indicators during execution")
	runCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress progress messages, show only errors and results")
}

func runChecks(_ *cobra.Command, args []string) error {
	// Load configuration first
	cfg, err := config.Load()
	if err != nil {
		// Use basic formatter for this error since config failed to load
		formatter := output.NewDefault()
		formatter.Error("Failed to load configuration: %v", err)
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create output formatter with config-based color settings
	formatter := output.New(output.Options{
		ColorEnabled: cfg.UI.ColorOutput && !noColor,
	})

	// Check if pre-commit system is enabled
	if !cfg.Enabled {
		formatter.Warning("Pre-commit system is disabled in configuration (ENABLE_PRE_COMMIT_SYSTEM=false)")
		return nil
	}

	// Get repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		formatter.Error("Failed to find git repository: %v", err)
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	// If show-checks flag is set, display available checks and exit
	if showVersion {
		return showAvailableChecks(cfg, formatter)
	}

	// Determine which files to check
	var filesToCheck []string
	if len(files) > 0 {
		// Specific files provided
		filesToCheck = files
	} else if allFiles {
		// All files in repository
		repo := git.NewRepository(repoRoot)
		filesToCheck, err = repo.GetAllFiles()
		if err != nil {
			formatter.Error("Failed to get all files: %v", err)
			return fmt.Errorf("failed to get all files: %w", err)
		}
	} else {
		// Staged files (default)
		repo := git.NewRepository(repoRoot)
		filesToCheck, err = repo.GetStagedFiles()
		if err != nil {
			formatter.Error("Failed to get staged files: %v", err)
			return fmt.Errorf("failed to get staged files: %w", err)
		}
	}

	if len(filesToCheck) == 0 {
		formatter.Info("No files to check")
		return nil
	}

	// Create runner
	r := runner.New(cfg, repoRoot)

	// Configure runner options
	opts := runner.Options{
		Files:               filesToCheck,
		Parallel:            parallel,
		FailFast:            failFast,
		GracefulDegradation: gracefulDegradation,
	}

	// Set up progress callback if progress is enabled and not in quiet mode
	if showProgress && !quiet {
		opts.ProgressCallback = func(checkName, status string) {
			switch status {
			case "running":
				formatter.Progress("Running %s check...", checkName)
			case "passed":
				formatter.Success("%s check passed", checkName)
			case "failed":
				formatter.Error("%s check failed", checkName)
			case "skipped":
				formatter.Warning("%s check skipped", checkName)
			}
		}
	}

	// Handle check selection
	if len(args) > 0 {
		// Specific check requested as positional argument
		opts.OnlyChecks = []string{args[0]}
	} else if len(onlyChecks) > 0 {
		// --only flag
		opts.OnlyChecks = onlyChecks
	} else if len(skipChecks) > 0 {
		// --skip flag
		opts.SkipChecks = skipChecks
	}

	// Show initial information (unless in quiet mode)
	if verbose && !quiet {
		formatter.Info("Running checks on %s", formatter.FormatFileList(filesToCheck, 3))
		if opts.Parallel > 0 {
			formatter.Info("Using %d parallel workers", opts.Parallel)
		}
		if gracefulDegradation {
			formatter.Info("Graceful degradation enabled - missing tools will be skipped")
		}
	}

	// Run checks
	results, err := r.Run(context.Background(), opts)
	if err != nil {
		formatter.Error("Failed to run checks: %v", err)
		return fmt.Errorf("failed to run checks: %w", err)
	}

	// Display results
	displayEnhancedResults(formatter, results, quiet)

	// Return error if any checks failed (unless they were gracefully skipped)
	if results.Failed > 0 {
		return fmt.Errorf("%w: %d", prerrors.ErrChecksFailed, results.Failed)
	}

	if results.Passed > 0 {
		formatter.Success("All checks passed! %s",
			formatter.FormatExecutionStats(results.Passed, results.Failed, results.Skipped, results.TotalDuration, results.TotalFiles))
	}

	return nil
}

// extractKeyErrorLines parses command output and extracts the most important error lines
func extractKeyErrorLines(output string) []string {
	var errorLines []string
	lines := strings.Split(output, "\n")

	// Compile regex once outside the loop
	goErrorRegex := regexp.MustCompile(`\w+\.go:\d+:\d+:`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip progress/status lines
		if strings.HasPrefix(line, "Running") ||
			strings.HasPrefix(line, "Checking") ||
			strings.HasPrefix(line, "Analyzing") ||
			strings.Contains(line, "files linted") {
			continue
		}

		// Look for actual error patterns
		isError := false

		// Go file errors (file.go:line:col: message)
		if goErrorRegex.MatchString(line) {
			isError = true
		}

		// Whitespace issues
		if strings.Contains(line, "trailing whitespace") ||
			strings.Contains(line, "mixed spaces and tabs") {
			isError = true
		}

		// General error indicators
		if strings.Contains(line, "error:") ||
			strings.Contains(line, "Error:") ||
			strings.Contains(line, "ERRO") ||
			strings.Contains(line, "level=error") ||
			strings.Contains(line, "✗") {
			isError = true
		}

		// Module tidy issues
		if strings.Contains(line, "go.mod") && strings.Contains(line, "not tidy") {
			isError = true
		}

		if isError {
			// Clean up ANSI color codes for cleaner display
			cleanLine := stripANSI(line)
			errorLines = append(errorLines, cleanLine)

			// Limit to first 10 error lines for non-verbose mode
			if len(errorLines) >= 10 {
				break
			}
		}
	}

	return errorLines
}

// stripANSI removes ANSI color codes from a string
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

func showAvailableChecks(cfg *config.Config, formatter *output.Formatter) error {
	formatter.Header("Available Checks")

	checks := []struct {
		name        string
		description string
		enabled     bool
	}{
		{"fumpt", "Format code with gofumpt", cfg.Checks.Fumpt},
		{"lint", "Run golangci-lint", cfg.Checks.Lint},
		{"mod-tidy", "Ensure go.mod and go.sum are tidy", cfg.Checks.ModTidy},
		{"whitespace", "Fix trailing whitespace", cfg.Checks.Whitespace},
		{"eof", "Ensure files end with newline", cfg.Checks.EOF},
	}

	for _, check := range checks {
		if check.enabled {
			formatter.Success("%-12s %s", check.name, check.description)
		} else {
			formatter.Detail("%-12s %s (disabled)", check.name, check.description)
		}
	}

	return nil
}

func displayEnhancedResults(formatter *output.Formatter, results *runner.Results, quietMode bool) {
	// In quiet mode, skip the header and only show failures
	if !quietMode {
		formatter.Header("Check Results")
	}

	// Collect failed checks for error summary
	var failedChecks []runner.CheckResult

	// Display each check result
	for _, result := range results.CheckResults {
		if result.Success {
			if !quietMode {
				if result.CanSkip && result.Suggestion != "" {
					// This was a gracefully skipped check
					formatter.Warning("%s - %s", result.Name, result.Error)
					if result.Suggestion != "" {
						formatter.SuggestAction(result.Suggestion)
					}
				} else {
					// Normal success
					formatter.Success("%s completed successfully", result.Name)
					if verbose {
						formatter.Detail("Duration: %s", formatter.Duration(result.Duration))
						if len(result.Files) > 0 {
							formatter.Detail("Files: %s", formatter.FormatFileList(result.Files, 3))
						}
					}
				}
			}
		} else {
			// Failed check - add to failed list for summary
			failedChecks = append(failedChecks, result)

			// Failed check
			formatter.Error("%s failed", result.Name)

			if verbose {
				formatter.Detail("Duration: %s", formatter.Duration(result.Duration))
				if len(result.Files) > 0 {
					formatter.Detail("Files: %s", formatter.FormatFileList(result.Files, 3))
				}
			}

			// Show error message
			if result.Error != "" {
				formatter.Detail("Error: %s", result.Error)
			}

			// Always show command output for failures to make errors visible
			// This is the key change - show actual errors even without verbose mode
			if result.Output != "" {
				// Parse and show key error lines
				errorLines := extractKeyErrorLines(result.Output)
				if len(errorLines) > 0 {
					for _, line := range errorLines {
						formatter.Detail("  %s", line)
					}
					if !verbose && len(errorLines) >= 10 {
						formatter.Detail("  ... (run with --verbose for full output)")
					}
				} else if verbose {
					// Fall back to full output in verbose mode
					formatter.Subheader("Command Output")
					formatter.CodeBlock(result.Output)
				}
			}

			// Show actionable suggestion
			if result.Suggestion != "" {
				formatter.SuggestAction(result.Suggestion)
			}
		}
	}

	// Summary (skip in quiet mode if all passed)
	if !quietMode || results.Failed > 0 {
		formatter.Subheader("Summary")
		stats := formatter.FormatExecutionStats(results.Passed, results.Failed, results.Skipped, results.TotalDuration, results.TotalFiles)
		if results.Failed > 0 {
			formatter.Error(stats)
		} else if results.Skipped > 0 {
			formatter.Warning(stats)
		} else {
			formatter.Success(stats)
		}
	}

	// Error Summary - show consolidated view of all failures
	if len(failedChecks) > 0 {
		formatter.Header("ERRORS FOUND")
		formatter.Error("%d check(s) failed with the following errors:", len(failedChecks))
		formatter.Detail("") // Empty line for spacing

		for _, check := range failedChecks {
			formatter.Error("━ %s ━", check.Name)

			// Show the specific errors for this check
			if check.Output != "" {
				errorLines := extractKeyErrorLines(check.Output)
				if len(errorLines) > 0 {
					for i, line := range errorLines {
						if i < 5 { // Show up to 5 error lines per check in summary
							formatter.Detail("  %s", line)
						}
					}
					if len(errorLines) > 5 {
						formatter.Detail("  ... and %d more errors", len(errorLines)-5)
					}
				} else {
					// No specific errors extracted, show generic message
					formatter.Detail("  %s", check.Error)
				}
			} else if check.Error != "" {
				formatter.Detail("  %s", check.Error)
			}

			// Show how to fix
			if check.Suggestion != "" {
				formatter.SuggestAction(fmt.Sprintf("To fix: %s", check.Suggestion))
			}
			formatter.Detail("") // Empty line for spacing
		}

		// Overall guidance
		formatter.Info("Run with --verbose flag to see full error details")
		formatter.Info("Fix the errors above and run the checks again")
	}
}

// resetRunFlags resets run command flags to their defaults for testing
func resetRunFlags() {
	allFiles = false
	files = nil
	skipChecks = nil
	onlyChecks = nil
	parallel = 0
	failFast = false
	showVersion = false
	gracefulDegradation = false
	showProgress = false
	quiet = false
}
