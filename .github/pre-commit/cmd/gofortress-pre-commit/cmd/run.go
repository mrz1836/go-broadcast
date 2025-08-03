package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/git"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/runner"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Required by cobra
var (
	allFiles    bool
	files       []string
	skipChecks  []string
	onlyChecks  []string
	parallel    int
	failFast    bool
	showVersion bool
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
}

func runChecks(_ *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if pre-commit system is enabled
	if !cfg.Enabled {
		printWarning("Pre-commit system is disabled in configuration (ENABLE_PRE_COMMIT_SYSTEM=false)")
		return nil
	}

	// Get repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	// If show-checks flag is set, display available checks and exit
	if showVersion {
		return showAvailableChecks(cfg)
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
			return fmt.Errorf("failed to get all files: %w", err)
		}
	} else {
		// Staged files (default)
		repo := git.NewRepository(repoRoot)
		filesToCheck, err = repo.GetStagedFiles()
		if err != nil {
			return fmt.Errorf("failed to get staged files: %w", err)
		}
	}

	if len(filesToCheck) == 0 {
		printInfo("No files to check")
		return nil
	}

	// Create runner
	r := runner.New(cfg, repoRoot)

	// Configure runner options
	opts := runner.Options{
		Files:    filesToCheck,
		Parallel: parallel,
		FailFast: failFast,
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

	// Run checks
	if verbose {
		printInfo("Running checks on %d files", len(filesToCheck))
		if opts.Parallel > 0 {
			printInfo("Using %d parallel workers", opts.Parallel)
		}
	}

	results, err := r.Run(opts)
	if err != nil {
		return fmt.Errorf("failed to run checks: %w", err)
	}

	// Display results
	displayResults(results)

	// Return error if any checks failed
	if results.Failed > 0 {
		return fmt.Errorf("%w: %d", prerrors.ErrChecksFailed, results.Failed)
	}

	printSuccess("All checks passed!")
	return nil
}

func showAvailableChecks(cfg *config.Config) error {
	_, _ = os.Stdout.WriteString("Available checks:\n")
	_, _ = os.Stdout.WriteString("\n")

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
		status := "disabled"
		if check.enabled {
			status = "enabled"
		}
		_, _ = fmt.Fprintf(os.Stdout, "  %-12s %s [%s]\n", check.name, check.description, status)
	}

	return nil
}

func displayResults(results *runner.Results) {
	// Summary header
	_, _ = os.Stdout.WriteString("\n")
	_, _ = fmt.Fprintln(os.Stdout, "Check Results:")
	_, _ = fmt.Fprintln(os.Stdout, "─────────────")

	// Display each check result
	for _, result := range results.CheckResults {
		statusIcon := "✓"
		statusColor := printSuccess
		if !result.Success {
			statusIcon = "✗"
			statusColor = printError
		}

		// Check name and status
		statusColor("%s %s", statusIcon, result.Name)

		// Duration
		if verbose {
			_, _ = fmt.Fprintf(os.Stdout, " (%s)", result.Duration)
		}

		// Error message if failed
		if !result.Success && result.Error != "" {
			_, _ = fmt.Fprintf(os.Stdout, "\n  %s", strings.ReplaceAll(result.Error, "\n", "\n  "))
		}

		// Output if verbose
		if verbose && result.Output != "" {
			_, _ = fmt.Fprintln(os.Stdout, "\n  Output:")
			for _, line := range strings.Split(result.Output, "\n") {
				if line != "" {
					_, _ = fmt.Fprintf(os.Stdout, "    %s\n", line)
				}
			}
		}
	}

	// Summary
	_, _ = os.Stdout.WriteString("\n")
	_, _ = fmt.Fprintf(os.Stdout, "Summary: %d passed, %d failed, %d skipped (total time: %s)\n",
		results.Passed, results.Failed, results.Skipped, results.TotalDuration)
}
