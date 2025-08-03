// Package runner provides the check execution engine for the pre-commit system
package runner

import (
	"context"
	"runtime"
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
	Files      []string
	OnlyChecks []string
	SkipChecks []string
	Parallel   int
	FailFast   bool
}

// Results contains the results of a check run
type Results struct {
	CheckResults  []CheckResult
	Passed        int
	Failed        int
	Skipped       int
	TotalDuration time.Duration
}

// CheckResult contains the result of a single check
type CheckResult struct {
	Name     string
	Success  bool
	Error    string
	Output   string
	Duration time.Duration
	Files    []string
}

// New creates a new Runner
func New(cfg *config.Config, repoRoot string) *Runner {
	return &Runner{
		config:   cfg,
		repoRoot: repoRoot,
		registry: checks.NewRegistry(),
	}
}

// Run executes checks based on the provided options
func (r *Runner) Run(opts Options) (*Results, error) {
	start := time.Now()

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.config.Timeout)*time.Second)
	defer cancel()

	// Run checks
	results := &Results{
		CheckResults: make([]CheckResult, 0, len(checksToRun)),
	}

	if opts.FailFast {
		// Sequential execution with fail-fast
		for _, check := range checksToRun {
			result := r.runCheck(ctx, check, opts.Files)
			results.CheckResults = append(results.CheckResults, result)

			if result.Success {
				results.Passed++
			} else {
				results.Failed++
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

				result := r.runCheck(ctx, c, opts.Files)
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
			} else {
				results.Failed++
			}
		}
	}

	results.TotalDuration = time.Since(start)
	return results, nil
}

// runCheck executes a single check
func (r *Runner) runCheck(ctx context.Context, check checks.Check, files []string) CheckResult {
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
