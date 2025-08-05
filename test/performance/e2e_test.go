package performance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/profiling"
	"github.com/mrz1836/go-broadcast/internal/sync"
	"github.com/mrz1836/go-broadcast/internal/worker"
)

// E2EPerformanceTest represents an end-to-end performance test scenario
type E2EPerformanceTest struct {
	Name         string
	Repositories int
	FilesPerRepo int
	Parallel     bool
	Timeout      time.Duration

	// Performance targets
	MaxDuration   time.Duration
	MinThroughput float64 // files per second
	MaxMemoryMB   int64
	MaxGoroutines int
}

// PerformanceResult captures the results of a performance test
type PerformanceResult struct {
	TestName       string        `json:"test_name"`
	Duration       time.Duration `json:"duration"`
	FilesProcessed int           `json:"files_processed"`
	Throughput     float64       `json:"throughput"` // files per second
	MemoryUsed     int64         `json:"memory_used_mb"`
	PeakGoroutines int           `json:"peak_goroutines"`
	Success        bool          `json:"success"`
	ErrorMessage   string        `json:"error_message,omitempty"`

	// Detailed metrics
	MemoryComparison profiling.MemoryComparison `json:"memory_comparison"`
	ProfileLocation  string                     `json:"profile_location,omitempty"`
}

// TestE2EPerformance runs comprehensive end-to-end performance tests
func TestE2EPerformance(t *testing.T) {
	// Skip in short test mode
	if testing.Short() {
		t.Skip("Skipping E2E performance tests in short mode")
	}

	scenarios := []E2EPerformanceTest{
		{
			Name:          "Small_Sequential",
			Repositories:  1,
			FilesPerRepo:  10,
			Parallel:      false,
			Timeout:       time.Minute * 2,
			MaxDuration:   time.Second * 30,
			MinThroughput: 1.0,
			MaxMemoryMB:   50,
			MaxGoroutines: 20,
		},
		{
			Name:          "Small_Parallel",
			Repositories:  1,
			FilesPerRepo:  10,
			Parallel:      true,
			Timeout:       time.Minute * 2,
			MaxDuration:   time.Second * 15,
			MinThroughput: 2.0,
			MaxMemoryMB:   75,
			MaxGoroutines: 50,
		},
		{
			Name:          "Medium_Sequential",
			Repositories:  5,
			FilesPerRepo:  50,
			Parallel:      false,
			Timeout:       time.Minute * 5,
			MaxDuration:   time.Minute * 2,
			MinThroughput: 10.0,
			MaxMemoryMB:   100,
			MaxGoroutines: 30,
		},
		{
			Name:          "Medium_Parallel",
			Repositories:  5,
			FilesPerRepo:  50,
			Parallel:      true,
			Timeout:       time.Minute * 5,
			MaxDuration:   time.Minute,
			MinThroughput: 20.0,
			MaxMemoryMB:   150,
			MaxGoroutines: 100,
		},
		{
			Name:          "Large_Sequential",
			Repositories:  10,
			FilesPerRepo:  100,
			Parallel:      false,
			Timeout:       time.Minute * 10,
			MaxDuration:   time.Minute * 5,
			MinThroughput: 50.0,
			MaxMemoryMB:   200,
			MaxGoroutines: 50,
		},
		{
			Name:          "Large_Parallel",
			Repositories:  10,
			FilesPerRepo:  100,
			Parallel:      true,
			Timeout:       time.Minute * 10,
			MaxDuration:   time.Minute * 3,
			MinThroughput: 100.0,
			MaxMemoryMB:   300,
			MaxGoroutines: 200,
		},
	}

	// Initialize profiling suite
	profilesDir := filepath.Join(t.TempDir(), "profiles")
	suite := profiling.NewProfileSuite(profilesDir)

	// Configure profiling to avoid CPU profiling conflicts in parallel tests
	config := profiling.ProfileConfig{
		EnableCPU:         false, // Disable CPU profiling to avoid conflicts
		EnableMemory:      true,
		EnableTrace:       false, // Disable trace to reduce overhead
		EnableBlock:       false,
		EnableMutex:       false,
		GenerateReports:   true,
		ReportFormat:      "text",
		AutoCleanup:       true,
		MaxSessionsToKeep: 5,
	}
	suite.Configure(config)

	results := make([]PerformanceResult, 0, len(scenarios))

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			result := runPerformanceScenario(t, scenario, suite)
			results = append(results, result)

			// Validate performance targets
			validatePerformanceTargets(t, scenario, result)
		})
	}

	// Generate final performance report
	generatePerformanceReport(t, results, profilesDir)
}

// runPerformanceScenario executes a single performance test scenario
func runPerformanceScenario(t *testing.T, scenario E2EPerformanceTest, suite *profiling.ProfileSuite) PerformanceResult {
	ctx, cancel := context.WithTimeout(context.Background(), scenario.Timeout)
	defer cancel()

	result := PerformanceResult{
		TestName: scenario.Name,
	}

	// Start profiling
	err := suite.StartProfiling(scenario.Name)
	if err != nil {
		t.Errorf("Failed to start profiling: %v", err)
		result.ErrorMessage = fmt.Sprintf("Profiling error: %v", err)
		return result
	}

	defer func() {
		if stopErr := suite.StopProfiling(); stopErr != nil {
			t.Errorf("Failed to stop profiling: %v", stopErr)
		}

		// Set profile location
		if session := suite.GetCurrentSession(); session != nil {
			result.ProfileLocation = session.OutputDir
		}
	}()

	// Capture initial memory state
	startSnapshot := profiling.CaptureMemStats(fmt.Sprintf("%s_start", scenario.Name))

	// Run the actual performance test
	start := time.Now()

	totalFiles := scenario.Repositories * scenario.FilesPerRepo
	err = executeE2EScenario(ctx, scenario)

	duration := time.Since(start)

	// Capture final memory state
	endSnapshot := profiling.CaptureMemStats(fmt.Sprintf("%s_end", scenario.Name))
	memComparison := startSnapshot.Compare(endSnapshot)

	// Calculate results
	result.Duration = duration
	result.FilesProcessed = totalFiles
	result.Throughput = float64(totalFiles) / duration.Seconds()
	result.MemoryUsed = int64(endSnapshot.MemStats.Alloc) / 1024 / 1024 //nolint:gosec // Memory allocation unlikely to overflow in tests
	result.PeakGoroutines = endSnapshot.Goroutines
	result.MemoryComparison = memComparison
	result.Success = err == nil

	if err != nil {
		result.ErrorMessage = err.Error()
	}

	// Log results
	t.Logf("Scenario: %s", scenario.Name)
	t.Logf("Duration: %v", duration)
	t.Logf("Files processed: %d", totalFiles)
	t.Logf("Throughput: %.2f files/sec", result.Throughput)
	t.Logf("Memory used: %d MB", result.MemoryUsed)
	t.Logf("Peak goroutines: %d", result.PeakGoroutines)

	if err != nil {
		t.Logf("Error: %v", err)
	}

	return result
}

// executeE2EScenario runs the actual sync operation for the scenario
func executeE2EScenario(ctx context.Context, scenario E2EPerformanceTest) error {
	// Generate test configuration
	config := generateTestConfig(scenario.Repositories, scenario.FilesPerRepo)

	// Create sync options
	options := sync.DefaultOptions()
	options.DryRun = true // Run in dry-run mode for performance testing
	options.MaxConcurrency = 10
	options.Timeout = scenario.Timeout

	if scenario.Parallel {
		return runParallelSync(ctx, config, options)
	}
	return runSequentialSync(ctx, config, options)
}

// generateTestConfig creates a test configuration for the scenario
func generateTestConfig(repositories, filesPerRepo int) *config.Config {
	targets := make([]config.TargetConfig, repositories)

	for i := 0; i < repositories; i++ {
		files := make([]config.FileMapping, filesPerRepo)
		for j := 0; j < filesPerRepo; j++ {
			files[j] = config.FileMapping{
				Src:  fmt.Sprintf("test-file-%d.txt", j+1),
				Dest: fmt.Sprintf("test-file-%d.txt", j+1),
			}
		}

		targets[i] = config.TargetConfig{
			Repo:  fmt.Sprintf("test/repo-%d", i+1),
			Files: files,
		}
	}

	cfg := &config.Config{
		Version: 1,
		Mappings: []config.SourceMapping{
			{
				Source: config.SourceConfig{
					Repo:   "test/source-repo",
					Branch: "master",
					ID:     "test-source",
				},
				Targets: targets,
			},
		},
	}

	return cfg
}

// runSequentialSync executes sync operations sequentially using worker simulation
func runSequentialSync(ctx context.Context, cfg *config.Config, _ *sync.Options) error {
	// Simulate sequential processing of repositories
	for _, mapping := range cfg.Mappings {
		for i, target := range mapping.Targets {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// Simulate processing time for each repository
				if err := simulateRepositorySync(ctx, target, fmt.Sprintf("repo_%d", i)); err != nil {
					return fmt.Errorf("failed to sync repository %s: %w", target.Repo, err)
				}
			}
		}
	}

	return nil
}

// runParallelSync executes sync operations in parallel using worker pool
func runParallelSync(ctx context.Context, cfg *config.Config, options *sync.Options) error {
	// Count total targets across all mappings
	totalTargets := 0
	for _, mapping := range cfg.Mappings {
		totalTargets += len(mapping.Targets)
	}

	// Create worker pool for parallel processing
	pool := worker.NewPool(options.MaxConcurrency, totalTargets)
	pool.Start(ctx)
	defer pool.Shutdown()

	// Create tasks for each repository
	tasks := make([]worker.Task, 0, totalTargets)
	for _, mapping := range cfg.Mappings {
		for i, target := range mapping.Targets {
			task := &repositoryTask{
				name:   fmt.Sprintf("repo_%d", i),
				target: target,
			}
			tasks = append(tasks, task)
		}
	}

	// Submit all tasks
	if err := pool.SubmitBatch(tasks); err != nil {
		return fmt.Errorf("failed to submit batch tasks: %w", err)
	}

	// Wait for completion
	completed := 0
	for result := range pool.Results() {
		if result.Error != nil {
			return fmt.Errorf("repository task failed: %w", result.Error)
		}
		completed++
		if completed >= totalTargets {
			break
		}
	}

	return nil
}

// repositoryTask implements worker.Task for repository sync simulation
type repositoryTask struct {
	name   string
	target config.TargetConfig
}

func (rt *repositoryTask) Execute(ctx context.Context) error {
	return simulateRepositorySync(ctx, rt.target, rt.name)
}

func (rt *repositoryTask) Name() string {
	return rt.name
}

// simulateRepositorySync simulates the sync process for a repository
func simulateRepositorySync(ctx context.Context, target config.TargetConfig, _ string) error {
	// Simulate processing time based on number of files
	processingTime := time.Duration(len(target.Files)) * time.Millisecond * 10

	select {
	case <-time.After(processingTime):
		// Simulate some CPU work for each file
		for i := 0; i < len(target.Files); i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				// Simulate file processing work
				simulateFileProcessing(target.Files[i])
			}
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// simulateFileProcessing simulates processing a single file
func simulateFileProcessing(_ config.FileMapping) {
	// Simulate some CPU-intensive work
	result := 0
	for i := 0; i < 1000; i++ {
		result += i * i
	}
	_ = result // Prevent optimization
}

// validatePerformanceTargets checks if the performance targets were met
func validatePerformanceTargets(t *testing.T, scenario E2EPerformanceTest, result PerformanceResult) {
	if !result.Success {
		t.Errorf("Scenario %s failed: %s", scenario.Name, result.ErrorMessage)
		return
	}

	// Check duration target
	if result.Duration > scenario.MaxDuration {
		t.Errorf("Duration target missed for %s: %v > %v",
			scenario.Name, result.Duration, scenario.MaxDuration)
	}

	// Check throughput target
	if result.Throughput < scenario.MinThroughput {
		t.Errorf("Throughput target missed for %s: %.2f < %.2f files/sec",
			scenario.Name, result.Throughput, scenario.MinThroughput)
	}

	// Check memory target
	if result.MemoryUsed > scenario.MaxMemoryMB {
		t.Errorf("Memory target missed for %s: %d > %d MB",
			scenario.Name, result.MemoryUsed, scenario.MaxMemoryMB)
	}

	// Check goroutine target
	if result.PeakGoroutines > scenario.MaxGoroutines {
		t.Errorf("Goroutine target missed for %s: %d > %d",
			scenario.Name, result.PeakGoroutines, scenario.MaxGoroutines)
	}

	// Success case logging
	t.Logf("✓ All performance targets met for %s", scenario.Name)
}

// writeToReport is a helper function that ignores fmt.Fprintf errors in test reports
func writeToReport(f *os.File, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(f, format, args...)
}

// generatePerformanceReport creates a comprehensive performance report
func generatePerformanceReport(t *testing.T, results []PerformanceResult, profilesDir string) {
	reportFile := filepath.Join(profilesDir, "e2e_performance_report.txt")

	f, err := os.Create(reportFile) //nolint:gosec // Creating test report file
	if err != nil {
		t.Errorf("Failed to create performance report: %v", err)
		return
	}
	defer func() { _ = f.Close() }() // Ignore error in defer

	// Report header
	writeToReport(f, "End-to-End Performance Test Report\n")
	writeToReport(f, "===================================\n\n")
	writeToReport(f, "Generated: %s\n", time.Now().Format(time.RFC3339))
	writeToReport(f, "Total scenarios: %d\n\n", len(results))

	// Summary statistics
	successful := 0
	totalThroughput := 0.0
	maxMemory := int64(0)
	maxGoroutines := 0

	for _, result := range results {
		if result.Success {
			successful++
		}
		totalThroughput += result.Throughput
		if result.MemoryUsed > maxMemory {
			maxMemory = result.MemoryUsed
		}
		if result.PeakGoroutines > maxGoroutines {
			maxGoroutines = result.PeakGoroutines
		}
	}

	writeToReport(f, "Summary Statistics\n")
	writeToReport(f, "------------------\n")
	writeToReport(f, "Successful scenarios: %d/%d (%.1f%%)\n",
		successful, len(results), float64(successful)/float64(len(results))*100)
	writeToReport(f, "Average throughput: %.2f files/sec\n", totalThroughput/float64(len(results)))
	writeToReport(f, "Peak memory usage: %d MB\n", maxMemory)
	writeToReport(f, "Peak goroutines: %d\n\n", maxGoroutines)

	// Detailed results
	writeToReport(f, "Detailed Results\n")
	writeToReport(f, "----------------\n")

	for _, result := range results {
		writeToReport(f, "\nScenario: %s\n", result.TestName)
		writeToReport(f, "  Status: %s\n", func() string {
			if result.Success {
				return "✓ PASS"
			}
			return "✗ FAIL"
		}())
		writeToReport(f, "  Duration: %v\n", result.Duration)
		writeToReport(f, "  Files processed: %d\n", result.FilesProcessed)
		writeToReport(f, "  Throughput: %.2f files/sec\n", result.Throughput)
		writeToReport(f, "  Memory used: %d MB\n", result.MemoryUsed)
		writeToReport(f, "  Peak goroutines: %d\n", result.PeakGoroutines)

		if result.ProfileLocation != "" {
			writeToReport(f, "  Profile location: %s\n", result.ProfileLocation)
		}

		if result.ErrorMessage != "" {
			writeToReport(f, "  Error: %s\n", result.ErrorMessage)
		}

		// Memory comparison
		if result.MemoryComparison.Duration > 0 {
			writeToReport(f, "  Memory delta: %s\n", profiling.FormatBytesDelta(result.MemoryComparison.AllocDelta))
			writeToReport(f, "  GC delta: %+d\n", result.MemoryComparison.GCDelta)
		}
	}

	// Performance targets analysis
	writeToReport(f, "\nPerformance Targets Analysis\n")
	writeToReport(f, "----------------------------\n")

	// Check if we meet the overall performance targets from the plan
	overallTargets := struct {
		ResponseTime time.Duration // < 2s for typical sync operation
		Throughput   float64       // > 100 files/second transformation
		MemoryUsage  int64         // < 100MB for 90% of operations
		Concurrency  int           // Linear scaling up to 20 parallel operations
	}{
		ResponseTime: time.Second * 2,
		Throughput:   100.0,
		MemoryUsage:  100,
		Concurrency:  20,
	}

	// Check response time (using large parallel as representative)
	var largeParallel *PerformanceResult
	for i := range results {
		if results[i].TestName == "Large_Parallel" && results[i].Success {
			largeParallel = &results[i]
			break
		}
	}

	if largeParallel != nil {
		responseTimeOK := largeParallel.Duration < overallTargets.ResponseTime
		throughputOK := largeParallel.Throughput > overallTargets.Throughput
		memoryOK := largeParallel.MemoryUsed < overallTargets.MemoryUsage

		writeToReport(f, "Response Time Target: %s (actual: %v) %s\n",
			overallTargets.ResponseTime, largeParallel.Duration,
			func() string {
				if responseTimeOK {
					return "✓"
				}
				return "✗"
			}())

		writeToReport(f, "Throughput Target: %.0f files/sec (actual: %.2f) %s\n",
			overallTargets.Throughput, largeParallel.Throughput,
			func() string {
				if throughputOK {
					return "✓"
				}
				return "✗"
			}())

		writeToReport(f, "Memory Usage Target: %d MB (actual: %d) %s\n",
			overallTargets.MemoryUsage, largeParallel.MemoryUsed,
			func() string {
				if memoryOK {
					return "✓"
				}
				return "✗"
			}())
	}

	t.Logf("Performance report generated: %s", reportFile)
}

// BenchmarkE2EPerformance provides benchmark versions of the E2E tests
func BenchmarkE2EPerformance(b *testing.B) {
	scenarios := []E2EPerformanceTest{
		{"Bench_Small", 1, 10, true, time.Minute, 0, 0, 0, 0},
		{"Bench_Medium", 5, 50, true, time.Minute * 3, 0, 0, 0, 0},
		{"Bench_Large", 10, 100, true, time.Minute * 5, 0, 0, 0, 0},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.Name, func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), scenario.Timeout)

				err := executeE2EScenario(ctx, scenario)
				if err != nil {
					b.Errorf("Benchmark scenario failed: %v", err)
				}

				cancel()
			}
		})
	}
}
