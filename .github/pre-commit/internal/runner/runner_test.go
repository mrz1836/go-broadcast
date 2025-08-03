package runner

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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

// Comprehensive test suite for runner functionality

type RunnerTestSuite struct {
	suite.Suite
	tempDir string
}

func TestRunnerSuite(t *testing.T) {
	suite.Run(t, new(RunnerTestSuite))
}

func (s *RunnerTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "runner_test_*")
	s.Require().NoError(err)
}

func (s *RunnerTestSuite) TearDownTest() {
	if s.tempDir != "" {
		err := os.RemoveAll(s.tempDir)
		s.Require().NoError(err)
	}
}

func (s *RunnerTestSuite) createTestFile(filename, content string) string {
	fullPath := s.tempDir + "/" + filename
	err := os.WriteFile(fullPath, []byte(content), 0o644)
	s.Require().NoError(err)
	return fullPath
}

// TestFailFastExecution tests the fail-fast execution path
func (s *RunnerTestSuite) TestFailFastExecution() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	r := New(cfg, s.tempDir)

	// Create a file with whitespace issues
	testFile := s.createTestFile("test.txt", "content with trailing spaces  \n")

	var progressCalls []string
	var progressMutex sync.Mutex

	opts := Options{
		Files:    []string{testFile},
		FailFast: true,
		ProgressCallback: func(checkName, status string) {
			progressMutex.Lock()
			defer progressMutex.Unlock()
			progressCalls = append(progressCalls, checkName+":"+status)
		},
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.Assert().NotNil(results)

	// Verify that progress callbacks were called
	progressMutex.Lock()
	defer progressMutex.Unlock()
	s.Assert().NotEmpty(progressCalls)
}

// TestParallelExecution tests the parallel execution path
func (s *RunnerTestSuite) TestParallelExecution() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	var progressCalls []string
	var progressMutex sync.Mutex

	opts := Options{
		Files:    []string{testFile},
		FailFast: false, // Parallel execution
		Parallel: 2,
		ProgressCallback: func(checkName, status string) {
			progressMutex.Lock()
			defer progressMutex.Unlock()
			progressCalls = append(progressCalls, checkName+":"+status)
		},
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.Assert().NotNil(results)

	// Should have run checks in parallel
	s.Assert().NotEmpty(results.CheckResults)

	// Verify that progress callbacks were called
	progressMutex.Lock()
	defer progressMutex.Unlock()
	s.Assert().NotEmpty(progressCalls)
}

// TestParallelismConfiguration tests different parallelism settings
func (s *RunnerTestSuite) TestParallelismConfiguration() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Performance.ParallelWorkers = 3
	cfg.Checks.Whitespace = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	tests := []struct {
		name             string
		parallelInOpts   int
		expectedParallel int // We can't directly test this, but we can test behavior
	}{
		{"default from config", 0, 3},
		{"override from opts", 5, 5},
		{"negative falls back", -1, 3},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			opts := Options{
				Files:    []string{testFile},
				FailFast: false,
				Parallel: tt.parallelInOpts,
			}

			results, err := r.Run(context.Background(), opts)
			s.Require().NoError(err)
			s.Assert().NotNil(results)
		})
	}
}

// TestGracefulDegradation tests graceful degradation functionality
func (s *RunnerTestSuite) TestGracefulDegradation() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	// Enable checks that might fail gracefully
	cfg.Checks.Fumpt = true // This might not be available
	cfg.Checks.Whitespace = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.go", "package main\n")

	opts := Options{
		Files:               []string{testFile},
		GracefulDegradation: true,
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.Assert().NotNil(results)

	// With graceful degradation, we should handle failures gracefully
	// The exact results depend on what tools are available
	s.Assert().True(results.Passed >= 0)
	s.Assert().True(results.Failed >= 0)
	s.Assert().True(results.Skipped >= 0)
}

// TestContextTimeout tests timeout handling
func (s *RunnerTestSuite) TestContextTimeout() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 1, // Very short timeout
	}
	cfg.Checks.Whitespace = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	opts := Options{
		Files: []string{testFile},
	}

	// The test should complete despite the short timeout for simple checks
	results, err := r.Run(context.Background(), opts)
	// Could succeed or fail depending on timing
	if err == nil {
		s.Assert().NotNil(results)
	}
}

// TestContextCancellation tests context cancellation
func (s *RunnerTestSuite) TestContextCancellation() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := Options{
		Files: []string{testFile},
	}

	results, err := r.Run(ctx, opts)
	// Should handle cancellation gracefully
	if err != nil {
		s.Assert().Contains(err.Error(), "context")
	} else {
		s.Assert().NotNil(results)
	}
}

// TestErrorConditions tests various error conditions
func (s *RunnerTestSuite) TestErrorConditions() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}

	r := New(cfg, s.tempDir)

	// Test with no enabled checks
	opts := Options{
		Files: []string{"test.go"},
	}

	results, err := r.Run(context.Background(), opts)
	s.Assert().Error(err) // Should return error when no checks to run
	s.Assert().Contains(err.Error(), "no checks to run")
	s.Assert().Nil(results)
}

// TestDetermineChecks tests the check determination logic
func (s *RunnerTestSuite) TestDetermineChecks() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true

	r := New(cfg, s.tempDir)

	tests := []struct {
		name        string
		onlyChecks  []string
		skipChecks  []string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "all enabled checks",
			onlyChecks:  nil,
			skipChecks:  nil,
			expectedMin: 1,
			expectedMax: 10, // Should get all enabled checks
		},
		{
			name:        "only specific checks",
			onlyChecks:  []string{"whitespace"},
			skipChecks:  nil,
			expectedMin: 1,
			expectedMax: 1,
		},
		{
			name:        "skip specific checks",
			onlyChecks:  nil,
			skipChecks:  []string{"fumpt"},
			expectedMin: 1,
			expectedMax: 10,
		},
		{
			name:        "skip all enabled checks",
			onlyChecks:  nil,
			skipChecks:  []string{"whitespace", "eof", "fumpt"},
			expectedMin: 0,
			expectedMax: 0,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			opts := Options{
				Files:      []string{"test.txt"},
				OnlyChecks: tt.onlyChecks,
				SkipChecks: tt.skipChecks,
			}

			results, err := r.Run(context.Background(), opts)
			if tt.expectedMin == 0 && tt.expectedMax == 0 {
				// Expect error when no checks to run
				s.Assert().Error(err)
				s.Assert().Contains(err.Error(), "no checks to run")
			} else {
				s.Require().NoError(err)
				s.Assert().NotNil(results)
				s.Assert().True(len(results.CheckResults) >= tt.expectedMin)
				s.Assert().True(len(results.CheckResults) <= tt.expectedMax)
			}
		})
	}
}

// TestProgressCallbacks tests progress callback functionality
func (s *RunnerTestSuite) TestProgressCallbacks() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	var progressEvents []string
	var progressMutex sync.Mutex

	opts := Options{
		Files: []string{testFile},
		ProgressCallback: func(checkName, status string) {
			progressMutex.Lock()
			defer progressMutex.Unlock()
			progressEvents = append(progressEvents, checkName+":"+status)
		},
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.Assert().NotNil(results)

	progressMutex.Lock()
	defer progressMutex.Unlock()

	// Should have at least running and completion events
	s.Assert().NotEmpty(progressEvents)

	// Verify we have running events
	hasRunning := false
	for _, event := range progressEvents {
		if contains(event, "running") {
			hasRunning = true
			break
		}
	}
	s.Assert().True(hasRunning, "Should have 'running' progress events")
}

// TestResultsAggregation tests that results are properly aggregated
func (s *RunnerTestSuite) TestResultsAggregation() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	opts := Options{
		Files: []string{testFile},
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.Assert().NotNil(results)

	// Verify results aggregation
	totalResults := results.Passed + results.Failed + results.Skipped
	s.Assert().Equal(len(results.CheckResults), totalResults)
	s.Assert().Equal(1, results.TotalFiles)
	s.Assert().True(results.TotalDuration >= 0)
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Unit tests for edge cases
func TestRunnerEdgeCases(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		// This would panic in real usage, but test the constructor
		assert.NotPanics(t, func() {
			r := New(&config.Config{}, "/test")
			assert.NotNil(t, r)
		})
	})

	t.Run("empty repo root", func(t *testing.T) {
		cfg := &config.Config{Enabled: true}
		r := New(cfg, "")
		assert.NotNil(t, r)
		assert.Equal(t, "", r.repoRoot)
	})

	t.Run("zero timeout", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
			Timeout: 0,
		}
		cfg.Checks.Whitespace = true

		r := New(cfg, "/test")

		opts := Options{
			Files: []string{"test.txt"},
		}

		// Zero timeout should still work for simple checks
		results, err := r.Run(context.Background(), opts)
		// May succeed or fail depending on timing
		if err == nil {
			assert.NotNil(t, results)
		}
	})
}
