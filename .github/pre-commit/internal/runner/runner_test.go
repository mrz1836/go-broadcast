package runner

import (
	"context"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true
	cfg.Checks.ModTidy = false
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	r := New(cfg, "/test/repo")
	assert.NotNil(t, r)
	assert.Equal(t, cfg, r.config)
	assert.Equal(t, "/test/repo", r.repoRoot)
	assert.NotNil(t, r.registry)
}

func TestRunner_Run_NoFiles(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Fumpt = true

	r := New(cfg, "/test/repo")

	opts := Options{
		Files: []string{},
	}

	results, err := r.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.NotNil(t, results)
	// When no files are provided, checks still run but succeed immediately
	assert.Equal(t, 1, results.Passed)
	assert.Equal(t, 0, results.Failed)
	assert.Len(t, results.CheckResults, 1)
}

func TestRunner_Run_BasicFlow(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	// Enable only built-in checks that don't require external tools
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = false
	cfg.Checks.Lint = false
	cfg.Checks.ModTidy = false

	r := New(cfg, "/test/repo")

	// Create test files
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.txt"

	opts := Options{
		Files:      []string{testFile},
		OnlyChecks: []string{"whitespace", "eof"},
	}

	results, err := r.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.NotNil(t, results)
	// Should have results for the checks we requested
	assert.NotEmpty(t, results.CheckResults)
}

func TestRunner_Run_OnlyChecks(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	// Enable all checks
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true
	cfg.Checks.ModTidy = true

	r := New(cfg, "/test/repo")

	opts := Options{
		Files:      []string{"test.go"},
		OnlyChecks: []string{"whitespace"}, // Only run whitespace
	}

	results, err := r.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.NotNil(t, results)

	// Should only have 1 check result
	assert.Len(t, results.CheckResults, 1)
	assert.Equal(t, "whitespace", results.CheckResults[0].Name)
}

func TestRunner_Run_SkipChecks(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	// Enable multiple checks
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = false
	cfg.Checks.Lint = false
	cfg.Checks.ModTidy = false

	r := New(cfg, "/test/repo")

	opts := Options{
		Files:      []string{"test.go"},
		SkipChecks: []string{"whitespace"}, // Skip whitespace
	}

	results, err := r.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.NotNil(t, results)

	// Should not have whitespace check in results
	for _, result := range results.CheckResults {
		assert.NotEqual(t, "whitespace", result.Name)
	}
}

func TestOptions(t *testing.T) {
	opts := Options{
		Files:      []string{"a.go", "b.go"},
		OnlyChecks: []string{"lint"},
		SkipChecks: []string{"fumpt"},
		Parallel:   4,
		FailFast:   true,
	}

	assert.Len(t, opts.Files, 2)
	assert.Len(t, opts.OnlyChecks, 1)
	assert.Len(t, opts.SkipChecks, 1)
	assert.Equal(t, 4, opts.Parallel)
	assert.True(t, opts.FailFast)
}

func TestResults(t *testing.T) {
	results := &Results{
		CheckResults: []CheckResult{
			{
				Name:     "test1",
				Success:  true,
				Duration: 100 * time.Millisecond,
			},
			{
				Name:     "test2",
				Success:  false,
				Error:    "test error",
				Duration: 200 * time.Millisecond,
			},
		},
		Passed:        1,
		Failed:        1,
		Skipped:       0,
		TotalDuration: 300 * time.Millisecond,
	}

	assert.Len(t, results.CheckResults, 2)
	assert.Equal(t, 1, results.Passed)
	assert.Equal(t, 1, results.Failed)
	assert.Equal(t, 0, results.Skipped)
}

func TestCheckResult(t *testing.T) {
	result := CheckResult{
		Name:     "test-check",
		Success:  false,
		Error:    "check failed",
		Output:   "detailed output",
		Duration: 123 * time.Millisecond,
		Files:    []string{"a.go", "b.go"},
	}

	assert.Equal(t, "test-check", result.Name)
	assert.False(t, result.Success)
	assert.Equal(t, "check failed", result.Error)
	assert.Equal(t, "detailed output", result.Output)
	assert.Equal(t, 123*time.Millisecond, result.Duration)
	assert.Len(t, result.Files, 2)
}
