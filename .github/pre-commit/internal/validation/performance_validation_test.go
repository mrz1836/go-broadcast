package validation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/runner"
)

// PerformanceValidationTestSuite validates that the system meets the <2s performance target
type PerformanceValidationTestSuite struct {
	suite.Suite
	tempDir    string
	envFile    string
	originalWD string
}

// SetupSuite initializes the test environment
func (s *PerformanceValidationTestSuite) SetupSuite() {
	var err error
	s.originalWD, err = os.Getwd()
	s.Require().NoError(err)

	// Create temporary directory structure
	s.tempDir = s.T().TempDir()

	// Create .github directory
	githubDir := filepath.Join(s.tempDir, ".github")
	s.Require().NoError(os.MkdirAll(githubDir, 0o755))

	// Create optimized .env.shared file for performance testing
	s.envFile = filepath.Join(githubDir, ".env.shared")
	envContent := `# Performance-optimized configuration
ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=error
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=false
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=10
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=0
PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT=5
PRE_COMMIT_SYSTEM_EOF_TIMEOUT=5
PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB=10
PRE_COMMIT_SYSTEM_MAX_FILES_OPEN=100
PRE_COMMIT_SYSTEM_COLOR_OUTPUT=false
`
	s.Require().NoError(os.WriteFile(s.envFile, []byte(envContent), 0o644))

	// Change to temp directory for tests
	s.Require().NoError(os.Chdir(s.tempDir))

	// Initialize git repository
	s.Require().NoError(s.initGitRepo())
}

// TearDownSuite cleans up the test environment
func (s *PerformanceValidationTestSuite) TearDownSuite() {
	// Restore original working directory
	_ = os.Chdir(s.originalWD)
}

// initGitRepo initializes a git repository in the temp directory
func (s *PerformanceValidationTestSuite) initGitRepo() error {
	gitDir := filepath.Join(s.tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0o644)
}

// Test2SecondTargetSmallCommit validates <2s performance for small commits (1-3 files)
func (s *PerformanceValidationTestSuite) Test2SecondTargetSmallCommit() {
	const target = 2 * time.Second
	const iterations = 10

	// Create small commit scenario (1-3 files)
	files := s.createSmallCommitFiles()

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	var durations []time.Duration
	var allResults []*runner.Results

	for i := 0; i < iterations; i++ {
		r := runner.New(cfg, s.tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), target*2)

		start := time.Now()
		result, err := r.Run(ctx, runner.Options{
			Files: files,
		})
		duration := time.Since(start)
		cancel()

		s.Require().NoError(err, "Iteration %d should succeed", i)
		s.Require().NotNil(result, "Result %d should not be nil", i)

		durations = append(durations, duration)
		allResults = append(allResults, result)
	}

	// Calculate statistics
	avgDuration := s.calculateAverage(durations)
	p95Duration := s.calculatePercentile(durations, 95)
	maxDuration := s.calculateMax(durations)

	// Validate performance targets
	s.LessOrEqual(avgDuration, target,
		"Average duration should be ≤2s: %v", avgDuration)
	s.True(p95Duration <= target*120/100, // Allow 20% buffer for P95
		"P95 duration should be ≤2.4s: %v", p95Duration)
	s.True(maxDuration <= target*150/100, // Allow 50% buffer for max
		"Max duration should be ≤3s: %v", maxDuration)

	s.T().Logf("Small commit performance: avg=%v, p95=%v, max=%v (target=%v)",
		avgDuration, p95Duration, maxDuration, target)
}

// Test2SecondTargetTypicalCommit validates <2s performance for typical commits (5-10 files)
func (s *PerformanceValidationTestSuite) Test2SecondTargetTypicalCommit() {
	const target = 2 * time.Second
	const iterations = 5

	// Create typical commit scenario (5-10 files)
	files := s.createTypicalCommitFiles()

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	var durations []time.Duration

	for i := 0; i < iterations; i++ {
		r := runner.New(cfg, s.tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), target*2)

		start := time.Now()
		result, err := r.Run(ctx, runner.Options{
			Files: files,
		})
		duration := time.Since(start)
		cancel()

		s.Require().NoError(err, "Iteration %d should succeed", i)
		s.Require().NotNil(result, "Result %d should not be nil", i)

		durations = append(durations, duration)
	}

	// Calculate statistics
	avgDuration := s.calculateAverage(durations)
	maxDuration := s.calculateMax(durations)

	// Validate performance targets (slightly relaxed for typical commits)
	s.True(avgDuration <= target*120/100, // Allow 20% buffer for typical commits
		"Average duration should be ≤2.4s: %v", avgDuration)
	s.True(maxDuration <= target*150/100, // Allow 50% buffer for max
		"Max duration should be ≤3s: %v", maxDuration)

	s.T().Logf("Typical commit performance: avg=%v, max=%v (target=%v)",
		avgDuration, maxDuration, target)
}

// TestPerformanceUnderParallelism validates performance with different parallelism levels
func (s *PerformanceValidationTestSuite) TestPerformanceUnderParallelism() {
	const target = 2 * time.Second

	files := s.createTypicalCommitFiles()

	testCases := []struct {
		name     string
		parallel int
		target   time.Duration
	}{
		{
			name:     "Sequential",
			parallel: 1,
			target:   target * 120 / 100, // 2.4s for sequential
		},
		{
			name:     "Dual Core",
			parallel: 2,
			target:   target,
		},
		{
			name:     "Quad Core",
			parallel: 4,
			target:   target,
		},
		{
			name:     "Auto (NumCPU)",
			parallel: 0, // Auto-detect
			target:   target,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Load configuration
			cfg, err := config.Load()
			s.Require().NoError(err)

			r := runner.New(cfg, s.tempDir)

			ctx, cancel := context.WithTimeout(context.Background(), tc.target*2)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files:    files,
				Parallel: tc.parallel,
			})
			duration := time.Since(start)

			s.Require().NoError(err, "Should succeed with %d workers", tc.parallel)
			s.Require().NotNil(result, "Should have result")

			s.True(duration <= tc.target,
				"Duration with %d workers should be ≤%v: %v", tc.parallel, tc.target, duration)

			s.T().Logf("%s (%d workers): %v (target: %v)",
				tc.name, tc.parallel, duration, tc.target)
		})
	}
}

// TestPerformanceScaling validates performance scaling with file count
func (s *PerformanceValidationTestSuite) TestPerformanceScaling() {
	testCases := []struct {
		name      string
		fileCount int
		target    time.Duration
	}{
		{
			name:      "5 Files",
			fileCount: 5,
			target:    1 * time.Second,
		},
		{
			name:      "10 Files",
			fileCount: 10,
			target:    2 * time.Second,
		},
		{
			name:      "20 Files",
			fileCount: 20,
			target:    3 * time.Second,
		},
		{
			name:      "50 Files",
			fileCount: 50,
			target:    5 * time.Second,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create files for this test case
			files := s.createScalingTestFiles(tc.fileCount)

			// Load configuration
			cfg, err := config.Load()
			s.Require().NoError(err)

			r := runner.New(cfg, s.tempDir)

			ctx, cancel := context.WithTimeout(context.Background(), tc.target*2)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files: files,
			})
			duration := time.Since(start)

			s.Require().NoError(err, "Should succeed with %d files", tc.fileCount)
			require.NotNil(s.T(), result, "Should have result")

			s.True(duration <= tc.target,
				"Duration with %d files should be ≤%v: %v", tc.fileCount, tc.target, duration)

			s.T().Logf("%s: %v (target: %v, files: %d)",
				tc.name, duration, tc.target, len(files))

			// Clean up files for next test
			s.cleanupScalingTestFiles(tc.fileCount)
		})
	}
}

// TestColdStartPerformance validates performance on first run (cold start)
func (s *PerformanceValidationTestSuite) TestColdStartPerformance() {
	const target = 3 * time.Second // Allow slightly more time for cold start

	files := s.createTypicalCommitFiles()

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	// Create fresh runner (cold start)
	r := runner.New(cfg, s.tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), target*2)
	defer cancel()

	start := time.Now()
	result, err := r.Run(ctx, runner.Options{
		Files: files,
	})
	duration := time.Since(start)

	s.Require().NoError(err, "Cold start should succeed")
	require.NotNil(s.T(), result, "Should have result")

	s.True(duration <= target,
		"Cold start duration should be ≤3s: %v", duration)

	s.T().Logf("Cold start performance: %v (target: %v)", duration, target)
}

// TestWarmRunPerformance validates performance on subsequent runs (warm)
func (s *PerformanceValidationTestSuite) TestWarmRunPerformance() {
	const target = 1500 * time.Millisecond // Stricter target for warm runs

	files := s.createTypicalCommitFiles()

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	r := runner.New(cfg, s.tempDir)

	// Warm up run
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = r.Run(ctx, runner.Options{Files: files})
	cancel()
	s.Require().NoError(err, "Warm-up run should succeed")

	// Measure warm run performance
	const iterations = 3
	var durations []time.Duration

	for i := 0; i < iterations; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), target*2)

		start := time.Now()
		result, err := r.Run(ctx, runner.Options{
			Files: files,
		})
		duration := time.Since(start)
		cancel()

		s.Require().NoError(err, "Warm run %d should succeed", i)
		require.NotNil(s.T(), result, "Should have result")

		durations = append(durations, duration)
	}

	avgDuration := s.calculateAverage(durations)
	maxDuration := s.calculateMax(durations)

	s.True(avgDuration <= target,
		"Average warm run duration should be ≤1.5s: %v", avgDuration)
	s.True(maxDuration <= target*120/100,
		"Max warm run duration should be ≤1.8s: %v", maxDuration)

	s.T().Logf("Warm run performance: avg=%v, max=%v (target: %v)",
		avgDuration, maxDuration, target)
}

// TestMemoryEfficiencyPerformance validates memory usage impact on performance
func (s *PerformanceValidationTestSuite) TestMemoryEfficiencyPerformance() {
	const target = 2 * time.Second

	files := s.createTypicalCommitFiles()

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	// Measure memory before
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	r := runner.New(cfg, s.tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), target*2)
	defer cancel()

	start := time.Now()
	result, err := r.Run(ctx, runner.Options{
		Files: files,
	})
	duration := time.Since(start)

	s.Require().NoError(err, "Should succeed")
	require.NotNil(s.T(), result, "Should have result")

	// Measure memory after
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	memUsed := memAfter.Alloc - memBefore.Alloc

	// Performance should not be impacted by memory usage
	s.True(duration <= target,
		"Duration should be ≤2s despite memory usage: %v", duration)

	// Memory usage should be reasonable
	maxMemory := uint64(50 * 1024 * 1024) // 50MB
	s.True(memUsed <= maxMemory,
		"Memory usage should be reasonable: %d bytes (max: %d)", memUsed, maxMemory)

	s.T().Logf("Memory-efficient performance: %v, memory used: %d bytes",
		duration, memUsed)
}

// TestErrorHandlingPerformance validates that error handling doesn't impact performance
func (s *PerformanceValidationTestSuite) TestErrorHandlingPerformance() {
	const target = 2 * time.Second

	// Create files that will cause some checks to be skipped (but not fail)
	files := []string{
		"main.go",     // Valid Go file
		"README.md",   // Valid markdown
		"binary.exe",  // Should be filtered out
		"config.yaml", // Valid YAML
	}
	s.createBasicFiles(files)

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	r := runner.New(cfg, s.tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), target*2)
	defer cancel()

	start := time.Now()
	result, err := r.Run(ctx, runner.Options{
		Files: files,
	})
	duration := time.Since(start)

	s.Require().NoError(err, "Should succeed despite filtering")
	require.NotNil(s.T(), result, "Should have result")

	s.True(duration <= target,
		"Duration with file filtering should be ≤2s: %v", duration)

	s.T().Logf("Error handling performance: %v (with file filtering)", duration)
}

// TestResourceConstrainedPerformance validates performance under resource constraints
func (s *PerformanceValidationTestSuite) TestResourceConstrainedPerformance() {
	const target = 3 * time.Second // Allow more time under constraints

	files := s.createTypicalCommitFiles()

	// Create constrained configuration
	s.createConstrainedPerformanceConfig()

	// Load constrained configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	r := runner.New(cfg, s.tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), target*2)
	defer cancel()

	start := time.Now()
	result, err := r.Run(ctx, runner.Options{
		Files:    files,
		Parallel: 1, // Force single-threaded
	})
	duration := time.Since(start)

	s.Require().NoError(err, "Should succeed under constraints")
	require.NotNil(s.T(), result, "Should have result")

	s.True(duration <= target,
		"Duration under constraints should be ≤3s: %v", duration)

	s.T().Logf("Resource-constrained performance: %v (target: %v)", duration, target)

	// Restore original configuration
	s.restorePerformanceConfig()
}

// Helper methods for creating test files and configurations

func (s *PerformanceValidationTestSuite) createSmallCommitFiles() []string {
	files := []string{"main.go", "README.md", "config.yaml"}
	s.createBasicFiles(files)
	return files
}

func (s *PerformanceValidationTestSuite) createTypicalCommitFiles() []string {
	files := []string{
		"main.go", "service.go", "handler.go", "model.go",
		"utils.go", "README.md", "CHANGELOG.md", "config.yaml",
		"docker-compose.yml",
	}
	s.createBasicFiles(files)
	return files
}

func (s *PerformanceValidationTestSuite) createScalingTestFiles(count int) []string {
	var files []string

	for i := 0; i < count; i++ {
		var filename string
		switch i % 4 {
		case 0:
			filename = fmt.Sprintf("service_%d.go", i)
		case 1:
			filename = fmt.Sprintf("model_%d.go", i)
		case 2:
			filename = fmt.Sprintf("doc_%d.md", i)
		case 3:
			filename = fmt.Sprintf("config_%d.yaml", i)
		}
		files = append(files, filename)
	}

	s.createBasicFiles(files)
	return files
}

func (s *PerformanceValidationTestSuite) cleanupScalingTestFiles(count int) {
	for i := 0; i < count; i++ {
		var filename string
		switch i % 4 {
		case 0:
			filename = fmt.Sprintf("service_%d.go", i)
		case 1:
			filename = fmt.Sprintf("model_%d.go", i)
		case 2:
			filename = fmt.Sprintf("doc_%d.md", i)
		case 3:
			filename = fmt.Sprintf("config_%d.yaml", i)
		}
		_ = os.Remove(filepath.Join(s.tempDir, filename))
	}
}

func (s *PerformanceValidationTestSuite) createBasicFiles(filenames []string) {
	for _, filename := range filenames {
		content := s.generateOptimizedFileContent(filename)
		fullPath := filepath.Join(s.tempDir, filename)
		s.Require().NoError(os.WriteFile(fullPath, []byte(content), 0o644))
	}
}

func (s *PerformanceValidationTestSuite) generateOptimizedFileContent(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".go":
		return `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	case ".md":
		return `# Test Document

This is a test document for performance validation.
`
	case ".yaml", ".yml":
		return `app:
  name: test-app
  version: 1.0.0
`
	default:
		return "Test content for performance validation.\n"
	}
}

func (s *PerformanceValidationTestSuite) createConstrainedPerformanceConfig() {
	constrainedConfig := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=error
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=5
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=1
PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT=2
PRE_COMMIT_SYSTEM_EOF_TIMEOUT=2
PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB=1
PRE_COMMIT_SYSTEM_MAX_FILES_OPEN=10
`
	s.Require().NoError(os.WriteFile(s.envFile, []byte(constrainedConfig), 0o644))
}

func (s *PerformanceValidationTestSuite) restorePerformanceConfig() {
	originalConfig := `ENABLE_PRE_COMMIT_SYSTEM=true
PRE_COMMIT_SYSTEM_LOG_LEVEL=error
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false
PRE_COMMIT_SYSTEM_ENABLE_LINT=false
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=false
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=10
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=0
PRE_COMMIT_SYSTEM_WHITESPACE_TIMEOUT=5
PRE_COMMIT_SYSTEM_EOF_TIMEOUT=5
PRE_COMMIT_SYSTEM_MAX_FILE_SIZE_MB=10
PRE_COMMIT_SYSTEM_MAX_FILES_OPEN=100
PRE_COMMIT_SYSTEM_COLOR_OUTPUT=false
`
	s.Require().NoError(os.WriteFile(s.envFile, []byte(originalConfig), 0o644))
}

// Statistical helper methods

func (s *PerformanceValidationTestSuite) calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func (s *PerformanceValidationTestSuite) calculateMax(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

func (s *PerformanceValidationTestSuite) calculatePercentile(durations []time.Duration, percentile int) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Simple percentile calculation (not perfectly accurate but sufficient for testing)
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	// Basic bubble sort for simplicity
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	index := int(float64(len(sorted)) * float64(percentile) / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// TestPerformanceRegression validates that performance doesn't regress over time
func (s *PerformanceValidationTestSuite) TestPerformanceRegression() {
	// This test establishes performance baselines that can be compared over time
	const target = 2 * time.Second
	const iterations = 5

	files := s.createTypicalCommitFiles()
	cfg, err := config.Load()
	s.Require().NoError(err)

	var durations []time.Duration

	for i := 0; i < iterations; i++ {
		r := runner.New(cfg, s.tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), target*2)
		start := time.Now()
		result, err := r.Run(ctx, runner.Options{Files: files})
		duration := time.Since(start)
		cancel()

		s.Require().NoError(err)
		require.NotNil(s.T(), result)
		durations = append(durations, duration)
	}

	avgDuration := s.calculateAverage(durations)
	maxDuration := s.calculateMax(durations)

	// Store baseline for regression testing
	s.T().Logf("PERFORMANCE_BASELINE: avg=%v, max=%v, target=%v", avgDuration, maxDuration, target)

	// Current validation
	s.True(avgDuration <= target, "Average should meet target: %v ≤ %v", avgDuration, target)
	s.True(maxDuration <= target*130/100, "Max should be within 30% of target: %v ≤ %v", maxDuration, target*130/100)
}

// TestSuite runs the performance validation test suite
func TestPerformanceValidationTestSuite(t *testing.T) {
	suite.Run(t, new(PerformanceValidationTestSuite))
}
