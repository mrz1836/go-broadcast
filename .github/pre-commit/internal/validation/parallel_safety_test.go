package validation

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/runner"
)

// ParallelSafetyTestSuite validates thread safety and parallel execution safety
type ParallelSafetyTestSuite struct {
	suite.Suite
	tempDir    string
	envFile    string
	originalWD string
	testFiles  []string
}

// SetupSuite initializes the test environment
func (s *ParallelSafetyTestSuite) SetupSuite() {
	var err error
	s.originalWD, err = os.Getwd()
	require.NoError(s.T(), err)

	// Create temporary directory structure
	s.tempDir = s.T().TempDir()

	// Create .github directory
	githubDir := filepath.Join(s.tempDir, ".github")
	require.NoError(s.T(), os.MkdirAll(githubDir, 0o755))

	// Create comprehensive .env.shared file for parallel testing
	s.envFile = filepath.Join(githubDir, ".env.shared")
	envContent := `# Test environment configuration for parallel safety testing
ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=info
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=false
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=120
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=4
PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT=30
PRE_COMMIT_SYSTEM_EOF_TIMEOUT=30
`
	require.NoError(s.T(), os.WriteFile(s.envFile, []byte(envContent), 0o644))

	// Change to temp directory for tests
	require.NoError(s.T(), os.Chdir(s.tempDir))

	// Initialize git repository
	require.NoError(s.T(), s.initGitRepo())

	// Create test files for parallel testing
	s.testFiles = s.createTestFiles()
}

// TearDownSuite cleans up the test environment
func (s *ParallelSafetyTestSuite) TearDownSuite() {
	// Restore original working directory
	_ = os.Chdir(s.originalWD)
}

// initGitRepo initializes a git repository in the temp directory
func (s *ParallelSafetyTestSuite) initGitRepo() error {
	gitDir := filepath.Join(s.tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0o644)
}

// createTestFiles creates a variety of test files for parallel processing
func (s *ParallelSafetyTestSuite) createTestFiles() []string {
	files := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`,
		"service.go": `package main

type Service struct {
	name string
}

func NewService(name string) *Service {
	return &Service{name: name}
}
`,
		"handler.go": `package main

import "net/http"

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
`,
		"model.go": `package main

type User struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}
`,
		"utils.go": `package main

func add(a, b int) int {
	return a + b
}
`,
		"README.md": `# Test Project

This is a test project for parallel safety validation.

## Features

- Parallel execution testing
- Thread safety validation
- Resource management testing
`,
		"CHANGELOG.md": `# Changelog

## v1.0.0
- Initial release
- Parallel execution support
`,
		"config.yaml": `
app:
  name: test-app
  version: 1.0.0
  parallel_workers: 4
`,
		"script.sh": `#!/bin/bash
echo "Test script for parallel execution"
exit 0
`,
		"data.txt": `Line 1
Line 2
Line 3
Line 4
Line 5
`,
		"go.mod": `module test-project

go 1.21
`,
	}

	var createdFiles []string
	for filename, content := range files {
		filePath := filepath.Join(s.tempDir, filename)
		require.NoError(s.T(), os.WriteFile(filePath, []byte(content), 0o644))
		createdFiles = append(createdFiles, filename)
	}

	return createdFiles
}

// TestConcurrentRunnerExecution validates that multiple runner instances can execute safely
func (s *ParallelSafetyTestSuite) TestConcurrentRunnerExecution() {
	const numGoroutines = 10
	const numIterations = 5

	// Load configuration once
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	var wg sync.WaitGroup
	results := make(chan *runner.Results, numGoroutines*numIterations)
	errors := make(chan error, numGoroutines*numIterations)

	// Launch multiple goroutines running the same checks concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				// Create a new runner for each execution
				r := runner.New(cfg, s.tempDir)

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				result, err := r.Run(ctx, runner.Options{
					Files:    s.testFiles,
					Parallel: 2, // Use parallel execution within each runner
				})
				cancel()

				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)
	close(errors)

	// Validate results
	var allResults []*runner.Results
	for result := range results {
		allResults = append(allResults, result)
	}

	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	// Should have no errors
	assert.Empty(s.T(), allErrors, "Concurrent execution should not produce errors")

	// Should have expected number of results
	expectedResults := numGoroutines * numIterations
	assert.Len(s.T(), allResults, expectedResults, "Should have all expected results")

	// All results should be valid
	for i, result := range allResults {
		assert.NotNil(s.T(), result, "Result %d should not be nil", i)
		assert.True(s.T(), result.TotalDuration > 0, "Result %d should have positive duration", i)
	}

	s.T().Logf("Concurrent execution test completed: %d goroutines × %d iterations = %d total executions",
		numGoroutines, numIterations, len(allResults))
}

// TestParallelCheckExecution validates internal parallel check execution safety
func (s *ParallelSafetyTestSuite) TestParallelCheckExecution() {
	testCases := []struct {
		name            string
		parallelWorkers int
		description     string
	}{
		{
			name:            "Single Worker",
			parallelWorkers: 1,
			description:     "Sequential execution for baseline",
		},
		{
			name:            "Multiple Workers",
			parallelWorkers: 4,
			description:     "Parallel execution with multiple workers",
		},
		{
			name:            "Max Workers",
			parallelWorkers: runtime.NumCPU(),
			description:     "Parallel execution with CPU count workers",
		},
		{
			name:            "Excessive Workers",
			parallelWorkers: runtime.NumCPU() * 2,
			description:     "Parallel execution with more workers than CPUs",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Load configuration
			cfg, err := config.Load()
			require.NoError(s.T(), err)

			// Create runner
			r := runner.New(cfg, s.tempDir)

			// Execute with specified parallelism
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files:    s.testFiles,
				Parallel: tc.parallelWorkers,
			})
			duration := time.Since(start)

			// Validate results
			assert.NoError(s.T(), err, tc.description)
			assert.NotNil(s.T(), result, "Result should not be nil")
			assert.True(s.T(), duration > 0, "Execution should take measurable time")

			s.T().Logf("%s: %d workers, duration=%v, checks=%d",
				tc.name, tc.parallelWorkers, duration, len(result.CheckResults))
		})
	}
}

// TestMemoryUsageUnderParallelExecution validates memory usage and cleanup
func (s *ParallelSafetyTestSuite) TestMemoryUsageUnderParallelExecution() {
	// Record initial memory stats
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Load configuration
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	const numIterations = 20

	// Run multiple iterations to test memory cleanup
	for i := 0; i < numIterations; i++ {
		r := runner.New(cfg, s.tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, err := r.Run(ctx, runner.Options{
			Files:    s.testFiles,
			Parallel: 4,
		})
		cancel()

		require.NoError(s.T(), err, "Iteration %d should not fail", i)
		require.NotNil(s.T(), result, "Result %d should not be nil", i)

		// Occasional GC to help with memory measurement
		if i%5 == 0 {
			runtime.GC()
		}
	}

	// Force GC and measure final memory
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Calculate memory differences
	allocDiff := memAfter.Alloc - memBefore.Alloc
	totalAllocDiff := memAfter.TotalAlloc - memBefore.TotalAlloc

	s.T().Logf("Memory usage: before=%d, after=%d, diff=%d, total_alloc_diff=%d",
		memBefore.Alloc, memAfter.Alloc, allocDiff, totalAllocDiff)

	// Memory should not grow excessively (allow reasonable buffer)
	maxAllowedGrowth := uint64(50 * 1024 * 1024) // 50MB
	assert.True(s.T(), allocDiff < maxAllowedGrowth,
		"Memory growth should be reasonable: %d bytes (max: %d)", allocDiff, maxAllowedGrowth)
}

// TestResourceCleanupUnderParallelExecution validates resource cleanup
func (s *ParallelSafetyTestSuite) TestResourceCleanupUnderParallelExecution() {
	// Count initial goroutines
	initialGoroutines := runtime.NumGoroutine()

	// Load configuration
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	const numIterations = 10

	// Run multiple iterations with parallel execution
	for i := 0; i < numIterations; i++ {
		r := runner.New(cfg, s.tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, err := r.Run(ctx, runner.Options{
			Files:    s.testFiles,
			Parallel: 4,
		})
		cancel()

		require.NoError(s.T(), err, "Iteration %d should not fail", i)
		require.NotNil(s.T(), result, "Result %d should not be nil", i)

		// Brief pause to allow cleanup
		time.Sleep(10 * time.Millisecond)
	}

	// Allow time for cleanup
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	// Count final goroutines
	finalGoroutines := runtime.NumGoroutine()

	s.T().Logf("Goroutines: initial=%d, final=%d, diff=%d",
		initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)

	// Goroutine count should not grow significantly
	// Allow some tolerance for test environment variance
	maxAllowedGrowth := 5
	goroutineGrowth := finalGoroutines - initialGoroutines
	assert.True(s.T(), goroutineGrowth <= maxAllowedGrowth,
		"Goroutine count should not grow excessively: %d (max: %d)",
		goroutineGrowth, maxAllowedGrowth)
}

// TestRaceConditionDetection validates absence of race conditions
func (s *ParallelSafetyTestSuite) TestRaceConditionDetection() {
	// This test should be run with -race flag to detect race conditions
	// go test -race ./internal/validation

	const numGoroutines = 20
	var wg sync.WaitGroup

	// Load configuration once and share among goroutines
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	// Shared state that might cause race conditions
	var executionCount int64
	var mutex sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Create runner (this should be safe)
			r := runner.New(cfg, s.tempDir)

			// Execute check (this should be safe)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			result, err := r.Run(ctx, runner.Options{
				Files:    s.testFiles,
				Parallel: 2,
			})
			cancel()

			// Update shared state safely
			mutex.Lock()
			executionCount++
			mutex.Unlock()

			require.NoError(s.T(), err, "Goroutine %d should not fail", id)
			require.NotNil(s.T(), result, "Result from goroutine %d should not be nil", id)
		}(i)
	}

	wg.Wait()

	// Validate final state
	mutex.Lock()
	finalCount := executionCount
	mutex.Unlock()

	assert.Equal(s.T(), int64(numGoroutines), finalCount,
		"All goroutines should have executed successfully")

	s.T().Logf("Race condition test completed: %d concurrent executions", finalCount)
}

// TestContextCancellationSafety validates proper context cancellation handling
func (s *ParallelSafetyTestSuite) TestContextCancellationSafety() {
	// Load configuration
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	testCases := []struct {
		name        string
		timeout     time.Duration
		description string
	}{
		{
			name:        "Immediate Cancellation",
			timeout:     1 * time.Millisecond,
			description: "Context cancelled almost immediately",
		},
		{
			name:        "Short Timeout",
			timeout:     100 * time.Millisecond,
			description: "Context cancelled after short timeout",
		},
		{
			name:        "Medium Timeout",
			timeout:     1 * time.Second,
			description: "Context cancelled after medium timeout",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			r := runner.New(cfg, s.tempDir)

			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files:    s.testFiles,
				Parallel: 4,
			})
			duration := time.Since(start)

			// Should handle cancellation gracefully
			if err != nil {
				// Context cancellation is expected and acceptable
				assert.Contains(s.T(), err.Error(), "context",
					"Error should be context-related")
			}

			// Should not take significantly longer than timeout
			maxDuration := tc.timeout + 2*time.Second // Allow reasonable buffer
			assert.True(s.T(), duration <= maxDuration,
				"Execution should respect timeout: %v (max: %v)", duration, maxDuration)

			s.T().Logf("%s: timeout=%v, duration=%v, cancelled=%v",
				tc.name, tc.timeout, duration, err != nil)

			// Result might be nil or partial on cancellation - both are valid
			if result != nil {
				s.T().Logf("Partial result received with %d checks", len(result.CheckResults))
			}
		})
	}
}

// TestParallelExecutionConsistency validates that parallel execution produces consistent results
func (s *ParallelSafetyTestSuite) TestParallelExecutionConsistency() {
	// Load configuration
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	const numRuns = 10
	var results []*runner.Results

	// Run the same checks multiple times with parallel execution
	for i := 0; i < numRuns; i++ {
		r := runner.New(cfg, s.tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, err := r.Run(ctx, runner.Options{
			Files:    s.testFiles,
			Parallel: 4,
		})
		cancel()

		require.NoError(s.T(), err, "Run %d should not fail", i)
		require.NotNil(s.T(), result, "Result %d should not be nil", i)

		results = append(results, result)
	}

	// Validate consistency across runs
	firstResult := results[0]
	for i, result := range results[1:] {
		// Should have same number of checks
		assert.Equal(s.T(), len(firstResult.CheckResults), len(result.CheckResults),
			"Run %d should have same number of checks as first run", i+1)

		// Should have same total file count
		assert.Equal(s.T(), firstResult.TotalFiles, result.TotalFiles,
			"Run %d should process same number of files", i+1)

		// Check results should be consistent (names and general success pattern)
		for j, checkResult := range result.CheckResults {
			if j < len(firstResult.CheckResults) {
				assert.Equal(s.T(), firstResult.CheckResults[j].Name, checkResult.Name,
					"Check %d name should be consistent across runs", j)
			}
		}
	}

	s.T().Logf("Consistency test completed: %d runs, %d checks per run",
		numRuns, len(firstResult.CheckResults))
}

// TestParallelExecutionUnderLoad validates behavior under high load
func (s *ParallelSafetyTestSuite) TestParallelExecutionUnderLoad() {
	// Create additional test files to increase load
	largeTestFiles := make([]string, 0, len(s.testFiles)+20)
	largeTestFiles = append(largeTestFiles, s.testFiles...)

	// Generate additional files
	for i := 0; i < 20; i++ {
		filename := filepath.Join(s.tempDir, "generated_"+string(rune('A'+i))+".md")
		content := "# Generated Test File " + string(rune('A'+i)) + "\n\nContent for testing.\n"
		require.NoError(s.T(), os.WriteFile(filename, []byte(content), 0o644))
		largeTestFiles = append(largeTestFiles, "generated_"+string(rune('A'+i))+".md")
	}

	// Load configuration
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	// Test with different load levels
	testCases := []struct {
		name            string
		files           []string
		parallelWorkers int
		description     string
	}{
		{
			name:            "Normal Load",
			files:           s.testFiles,
			parallelWorkers: 2,
			description:     "Normal file count with moderate parallelism",
		},
		{
			name:            "High Load - Many Files",
			files:           largeTestFiles,
			parallelWorkers: 4,
			description:     "High file count with high parallelism",
		},
		{
			name:            "High Load - Max Workers",
			files:           largeTestFiles,
			parallelWorkers: runtime.NumCPU(),
			description:     "High file count with maximum workers",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			r := runner.New(cfg, s.tempDir)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files:    tc.files,
				Parallel: tc.parallelWorkers,
			})
			duration := time.Since(start)

			// Should complete successfully even under load
			assert.NoError(s.T(), err, tc.description)
			assert.NotNil(s.T(), result, "Result should not be nil")
			assert.True(s.T(), duration > 0, "Should have measurable duration")

			s.T().Logf("%s: %d files, %d workers, duration=%v",
				tc.name, len(tc.files), tc.parallelWorkers, duration)
		})
	}
}

// TestParallelExecutionErrorHandling validates error handling in parallel scenarios
func (s *ParallelSafetyTestSuite) TestParallelExecutionErrorHandling() {
	// Create configuration with very short timeouts to trigger errors
	githubDir := filepath.Join(s.tempDir, ".github")
	envFile := filepath.Join(githubDir, ".env.shared")
	shortTimeoutConfig := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=info
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=1
PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT=1
PRE_COMMIT_SYSTEM_EOF_TIMEOUT=1
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=4
`
	require.NoError(s.T(), os.WriteFile(envFile, []byte(shortTimeoutConfig), 0o644))

	// Load the configuration with short timeouts
	cfg, err := config.Load()
	require.NoError(s.T(), err)

	const numGoroutines = 5
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	results := make(chan *runner.Results, numGoroutines)

	// Launch multiple goroutines that may encounter timeouts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			r := runner.New(cfg, s.tempDir)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := r.Run(ctx, runner.Options{
				Files:    s.testFiles,
				Parallel: 4,
			})

			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(results)

	// Collect results
	var allErrors []error
	var allResults []*runner.Results

	for err := range errors {
		allErrors = append(allErrors, err)
	}

	for result := range results {
		allResults = append(allResults, result)
	}

	// Should handle errors gracefully without crashing
	totalExecutions := len(allErrors) + len(allResults)
	assert.Equal(s.T(), numGoroutines, totalExecutions,
		"All executions should complete (with success or error)")

	s.T().Logf("Error handling test: %d errors, %d successes out of %d executions",
		len(allErrors), len(allResults), numGoroutines)

	// Restore original configuration
	originalConfig := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=info
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=false
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=120
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=4
`
	require.NoError(s.T(), os.WriteFile(envFile, []byte(originalConfig), 0o644))
}

// TestSuite runs the parallel safety test suite
func TestParallelSafetyTestSuite(t *testing.T) {
	suite.Run(t, new(ParallelSafetyTestSuite))
}
