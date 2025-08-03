// Package runner provides the check execution engine for the pre-commit system
package runner

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/checks"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
)

// Runner executes pre-commit checks
type Runner struct {
	config   *config.Config
	repoRoot string
	registry *checks.Registry
}

// Options configures a check run
type Options struct {
	Files               []string
	OnlyChecks          []string
	SkipChecks          []string
	Parallel            int
	FailFast            bool
	ProgressCallback    ProgressCallback
	GracefulDegradation bool
}

// Results contains the results of a check run
type Results struct {
	CheckResults  []CheckResult
	Passed        int
	Failed        int
	Skipped       int
	TotalDuration time.Duration
	TotalFiles    int
}

// CheckResult contains the result of a single check
type CheckResult struct {
	Name       string
	Success    bool
	Error      string
	Output     string
	Duration   time.Duration
	Files      []string
	Suggestion string
	CanSkip    bool
	Command    string
}

// ProgressCallback is called during check execution for progress updates
type ProgressCallback func(checkName string, status string)

// New creates a new Runner
func New(cfg *config.Config, repoRoot string) *Runner {
	return &Runner{
		config:   cfg,
		repoRoot: repoRoot,
		registry: checks.NewRegistryWithConfig(cfg),
	}
}

// Run executes checks based on the provided options
func (r *Runner) Run(ctx context.Context, opts Options) (*Results, error) {
	start := time.Now()

	// Process SKIP environment variables and combine with CLI skip options
	opts.SkipChecks = r.combineSkipSources(opts.SkipChecks)

	// Determine which checks to run
	checksToRun, err := r.determineChecks(opts)
	if err != nil {
		return nil, err
	}

	// Determine parallelism
	parallel := opts.Parallel
	if parallel <= 0 {
		parallel = r.config.Performance.ParallelWorkers
		if parallel <= 0 {
			parallel = runtime.NumCPU()
		}
	}

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(r.config.Timeout)*time.Second)
	defer cancel()

	// Run checks
	results := &Results{
		CheckResults: make([]CheckResult, 0, len(checksToRun)),
		TotalFiles:   len(opts.Files),
	}

	if opts.FailFast {
		// Sequential execution with fail-fast
		for _, check := range checksToRun {
			if opts.ProgressCallback != nil {
				opts.ProgressCallback(check.Name(), "running")
			}

			result := r.runCheck(ctxWithTimeout, check, opts.Files, opts.GracefulDegradation)
			results.CheckResults = append(results.CheckResults, result)

			if result.Success {
				results.Passed++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(check.Name(), "passed")
				}
			} else if result.CanSkip && opts.GracefulDegradation {
				results.Skipped++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(check.Name(), "skipped")
				}
			} else {
				results.Failed++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(check.Name(), "failed")
				}
				break // Stop on first failure
			}
		}
	} else {
		// Parallel execution
		resultsChan := make(chan CheckResult, len(checksToRun))
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, parallel)

		for _, check := range checksToRun {
			wg.Add(1)
			go func(c checks.Check) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				if opts.ProgressCallback != nil {
					opts.ProgressCallback(c.Name(), "running")
				}

				result := r.runCheck(ctxWithTimeout, c, opts.Files, opts.GracefulDegradation)
				resultsChan <- result
			}(check)
		}

		wg.Wait()
		close(resultsChan)

		// Collect results
		for result := range resultsChan {
			results.CheckResults = append(results.CheckResults, result)
			if result.Success {
				results.Passed++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(result.Name, "passed")
				}
			} else if result.CanSkip && opts.GracefulDegradation {
				results.Skipped++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(result.Name, "skipped")
				}
			} else {
				results.Failed++
				if opts.ProgressCallback != nil {
					opts.ProgressCallback(result.Name, "failed")
				}
			}
		}
	}

	results.TotalDuration = time.Since(start)
	return results, nil
}

// runCheck executes a single check
func (r *Runner) runCheck(ctx context.Context, check checks.Check, files []string, gracefulDegradation bool) CheckResult {
	start := time.Now()

	// Filter files for this check
	filteredFiles := check.FilterFiles(files)
	if len(filteredFiles) == 0 {
		return CheckResult{
			Name:     check.Name(),
			Success:  true,
			Duration: time.Since(start),
			Files:    filteredFiles,
		}
	}

	// Run the check
	err := check.Run(ctx, filteredFiles)

	result := CheckResult{
		Name:     check.Name(),
		Success:  err == nil,
		Duration: time.Since(start),
		Files:    filteredFiles,
	}

	if err != nil {
		result.Error = err.Error()

		// Check if this is an enhanced CheckError with context
		var checkErr *prerrors.CheckError
		if errors.As(err, &checkErr) {
			result.Suggestion = checkErr.Suggestion
			result.CanSkip = checkErr.CanSkip
			result.Command = checkErr.Command
			result.Output = checkErr.Output

			// If graceful degradation is enabled and this error can be skipped
			if gracefulDegradation && checkErr.CanSkip {
				result.Success = true // Mark as success but with warning info
			}
		}
	}

	return result
}

// determineChecks figures out which checks to run based on options and config
func (r *Runner) determineChecks(opts Options) ([]checks.Check, error) {
	// Get all available checks
	allChecks := r.registry.GetChecks()

	checksToRun := make([]checks.Check, 0, len(allChecks))

	// Filter based on options
	for _, check := range allChecks {
		name := check.Name()

		// Skip if disabled in config
		if !r.isCheckEnabled(name) {
			continue
		}

		// Handle --only flag
		if len(opts.OnlyChecks) > 0 {
			found := false
			for _, only := range opts.OnlyChecks {
				if only == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Handle --skip flag
		if len(opts.SkipChecks) > 0 {
			skip := false
			for _, skipName := range opts.SkipChecks {
				if skipName == name {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}

		checksToRun = append(checksToRun, check)
	}

	if len(checksToRun) == 0 {
		return nil, prerrors.ErrNoChecksToRun
	}

	return checksToRun, nil
}

// isCheckEnabled checks if a check is enabled in the configuration
func (r *Runner) isCheckEnabled(name string) bool {
	switch name {
	case "fumpt":
		return r.config.Checks.Fumpt
	case "lint":
		return r.config.Checks.Lint
	case "mod-tidy":
		return r.config.Checks.ModTidy
	case "whitespace":
		return r.config.Checks.Whitespace
	case "eof":
		return r.config.Checks.EOF
	default:
		return false
	}
}

// combineSkipSources processes SKIP environment variables and combines them with CLI skip options
func (r *Runner) combineSkipSources(cliSkips []string) []string {
	// Start with CLI skip options
	allSkips := make([]string, 0)
	if len(cliSkips) > 0 {
		allSkips = append(allSkips, cliSkips...)
	}

	// Process environment variables in order of precedence
	envSkips := r.processSkipEnvironment()
	if len(envSkips) > 0 {
		allSkips = append(allSkips, envSkips...)
	}

	// Remove duplicates and validate
	return r.deduplicateAndValidateSkips(allSkips)
}

// processSkipEnvironment reads and processes SKIP-related environment variables
func (r *Runner) processSkipEnvironment() []string {
	var skips []string

	// Check multiple environment variables in order of precedence
	skipEnvVars := []string{
		"SKIP",                   // Standard pre-commit convention
		"PRE_COMMIT_SYSTEM_SKIP", // GoFortress-specific
	}

	for _, envVar := range skipEnvVars {
		if value := strings.TrimSpace(os.Getenv(envVar)); value != "" {
			// Parse the skip value (comma-separated list)
			parsed := r.parseSkipValue(value)
			if len(parsed) > 0 {
				skips = append(skips, parsed...)
				// Use the first non-empty environment variable found
				break
			}
		}
	}

	return skips
}

// parseSkipValue parses a SKIP environment variable value
func (r *Runner) parseSkipValue(value string) []string {
	if value == "" {
		return nil
	}

	// Handle special values
	if strings.ToLower(value) == "all" {
		return []string{"fumpt", "lint", "mod-tidy", "whitespace", "eof"}
	}

	// Split by comma and clean up
	parts := strings.Split(value, ",")
	var skips []string
	for _, part := range parts {
		if cleaned := strings.TrimSpace(part); cleaned != "" {
			skips = append(skips, cleaned)
		}
	}

	return skips
}

// deduplicateAndValidateSkips removes duplicates and validates skip names
func (r *Runner) deduplicateAndValidateSkips(skips []string) []string {
	seen := make(map[string]bool)
	var result []string

	validChecks := map[string]bool{
		"fumpt":      true,
		"lint":       true,
		"mod-tidy":   true,
		"whitespace": true,
		"eof":        true,
	}

	for _, skip := range skips {
		skip = strings.TrimSpace(skip)
		if skip == "" {
			continue
		}

		// Skip duplicates
		if seen[skip] {
			continue
		}

		// Validate check name
		if !validChecks[skip] {
			// Log warning for invalid check names but don't fail
			// This allows for future extensibility
			continue
		}

		seen[skip] = true
		result = append(result, skip)
	}

	return result
}
