// Package main provides a comprehensive demonstration of the profiling capabilities.
// This demo showcases memory profiling, performance benchmarking, and resource monitoring
// across various operations including caching, worker pools, and batch processing.
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mrz1836/go-broadcast/internal/algorithms"
	"github.com/mrz1836/go-broadcast/internal/cache"
	"github.com/mrz1836/go-broadcast/internal/profiling"
	"github.com/mrz1836/go-broadcast/internal/reporting"
	"github.com/mrz1836/go-broadcast/internal/worker"
)

// secureRandInt generates a cryptographically secure random integer in range [0, maxVal)
func secureRandInt(maxVal int) int {
	if maxVal <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxVal)))
	if err != nil {
		// Fallback to current time for demo purposes
		return int(time.Now().UnixNano()) % maxVal
	}
	return int(n.Int64())
}

func main() {
	app := NewProfileDemoApp()
	if err := app.Run(); err != nil {
		log.Fatalf("Profile demo failed: %v", err)
	}
}

// ProfileDemoApp represents the main profiling demonstration application
type ProfileDemoApp struct {
	logger              Logger
	dirManager          DirectoryManager
	profileSuiteFactory ProfileSuiteFactory
	testRunner          TestRunner
	reportGenerator     ReportGenerator
}

// Logger defines the interface for logging operations
type Logger interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

// DirectoryManager defines the interface for directory operations
type DirectoryManager interface {
	MkdirAll(path string, perm os.FileMode) error
}

// ProfileSuiteFactory defines the interface for creating profile suites
type ProfileSuiteFactory interface {
	NewProfileSuite(profilesDir string) ProfileSuite
}

// ProfileSuite defines the interface for profiling operations
type ProfileSuite interface {
	Configure(config profiling.ProfileConfig)
	StartProfiling(sessionName string) error
	StopProfiling() error
}

// TestRunner defines the interface for running performance tests
type TestRunner interface {
	TestWorkerPool() time.Duration
	TestTTLCache() time.Duration
	TestAlgorithmOptimizations() time.Duration
	TestBatchProcessing() time.Duration
}

// ReportGenerator defines the interface for generating reports
type ReportGenerator interface {
	GenerateFinalReport(metrics map[string]float64, profilesDir string) error
}

// DefaultLogger implements Logger using the log package
type DefaultLogger struct{}

func (d *DefaultLogger) Println(v ...interface{}) {
	log.Println(v...)
}

func (d *DefaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// DefaultDirectoryManager implements DirectoryManager using os package
type DefaultDirectoryManager struct{}

func (d *DefaultDirectoryManager) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// DefaultProfileSuiteFactory implements ProfileSuiteFactory
type DefaultProfileSuiteFactory struct{}

func (d *DefaultProfileSuiteFactory) NewProfileSuite(profilesDir string) ProfileSuite {
	return &DefaultProfileSuiteWrapper{suite: profiling.NewProfileSuite(profilesDir)}
}

// DefaultProfileSuiteWrapper wraps the actual profiling.ProfileSuite
type DefaultProfileSuiteWrapper struct {
	suite interface {
		Configure(config profiling.ProfileConfig)
		StartProfiling(sessionName string) error
		StopProfiling() error
	}
}

func (d *DefaultProfileSuiteWrapper) Configure(config profiling.ProfileConfig) {
	d.suite.Configure(config)
}

func (d *DefaultProfileSuiteWrapper) StartProfiling(sessionName string) error {
	return d.suite.StartProfiling(sessionName)
}

func (d *DefaultProfileSuiteWrapper) StopProfiling() error {
	return d.suite.StopProfiling()
}

// DefaultTestRunner implements TestRunner using the original test functions
type DefaultTestRunner struct{}

func (d *DefaultTestRunner) TestWorkerPool() time.Duration {
	start := time.Now()
	testWorkerPool()
	return time.Since(start)
}

func (d *DefaultTestRunner) TestTTLCache() time.Duration {
	start := time.Now()
	testTTLCache()
	return time.Since(start)
}

func (d *DefaultTestRunner) TestAlgorithmOptimizations() time.Duration {
	start := time.Now()
	testAlgorithmOptimizations()
	return time.Since(start)
}

func (d *DefaultTestRunner) TestBatchProcessing() time.Duration {
	start := time.Now()
	testBatchProcessing()
	return time.Since(start)
}

// DefaultReportGenerator implements ReportGenerator using the original function
type DefaultReportGenerator struct{}

func (d *DefaultReportGenerator) GenerateFinalReport(metrics map[string]float64, profilesDir string) error {
	generateFinalReport(metrics, profilesDir)
	return nil
}

// NewProfileDemoApp creates a new ProfileDemoApp with default implementations
func NewProfileDemoApp() *ProfileDemoApp {
	return &ProfileDemoApp{
		logger:              &DefaultLogger{},
		dirManager:          &DefaultDirectoryManager{},
		profileSuiteFactory: &DefaultProfileSuiteFactory{},
		testRunner:          &DefaultTestRunner{},
		reportGenerator:     &DefaultReportGenerator{},
	}
}

// NewProfileDemoAppWithDependencies creates a new ProfileDemoApp with injectable dependencies
func NewProfileDemoAppWithDependencies(logger Logger, dirManager DirectoryManager, profileSuiteFactory ProfileSuiteFactory, testRunner TestRunner, reportGenerator ReportGenerator) *ProfileDemoApp {
	return &ProfileDemoApp{
		logger:              logger,
		dirManager:          dirManager,
		profileSuiteFactory: profileSuiteFactory,
		testRunner:          testRunner,
		reportGenerator:     reportGenerator,
	}
}

// Run executes the profiling demonstration
func (app *ProfileDemoApp) Run() error {
	app.logger.Println("Starting comprehensive profiling demonstration...")

	// Initialize profiling suite
	profilesDir := "./profiles/final_demo"
	if err := app.dirManager.MkdirAll(profilesDir, 0o750); err != nil {
		return fmt.Errorf("failed to create profiles directory: %w", err)
	}

	suite := app.profileSuiteFactory.NewProfileSuite(profilesDir)

	// Configure comprehensive profiling
	config := profiling.ProfileConfig{
		EnableCPU:            false, // Disabled to avoid conflicts
		EnableMemory:         true,
		EnableTrace:          false, // Disabled to reduce overhead
		EnableBlock:          false,
		EnableMutex:          false,
		BlockProfileRate:     1,
		MutexProfileFraction: 1,
		GenerateReports:      true,
		ReportFormat:         "text",
		AutoCleanup:          false,
		MaxSessionsToKeep:    10,
	}
	suite.Configure(config)

	// Start profiling session
	if err := suite.StartProfiling("final_optimization_demo"); err != nil {
		return fmt.Errorf("failed to start profiling: %w", err)
	}

	app.logger.Println("Profiling started - running optimization demonstrations...")

	// Create performance metrics collector
	metrics := make(map[string]float64)

	// Run performance tests
	app.logger.Println("1. Testing Worker Pool optimization...")
	duration := app.testRunner.TestWorkerPool()
	metrics["worker_pool_duration_ms"] = float64(duration.Nanoseconds()) / 1e6

	app.logger.Println("2. Testing TTL Cache optimization...")
	duration = app.testRunner.TestTTLCache()
	metrics["cache_duration_ms"] = float64(duration.Nanoseconds()) / 1e6

	app.logger.Println("3. Testing Algorithm optimizations...")
	duration = app.testRunner.TestAlgorithmOptimizations()
	metrics["algorithms_duration_ms"] = float64(duration.Nanoseconds()) / 1e6

	app.logger.Println("4. Testing Batch Processing optimization...")
	duration = app.testRunner.TestBatchProcessing()
	metrics["batch_processing_duration_ms"] = float64(duration.Nanoseconds()) / 1e6

	app.logger.Println("Optimization demonstrations completed - stopping profiling...")

	// Stop profiling
	if err := suite.StopProfiling(); err != nil {
		app.logger.Printf("Warning: failed to stop profiling: %v", err)
	}

	// Generate performance report
	app.logger.Println("Generating comprehensive performance report...")
	if err := app.reportGenerator.GenerateFinalReport(metrics, profilesDir); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	app.logger.Println("Final profiling demonstration completed successfully!")
	app.logger.Printf("Results available in: %s\n", profilesDir)

	return nil
}

func testWorkerPool() {
	// Create worker pool with optimal worker count
	pool, err := worker.NewPool(8, 100) // 8 workers, 100 queue size
	if err != nil {
		log.Printf("Failed to create worker pool: %v", err)
		return
	}

	pool.Start(context.Background())
	defer pool.Shutdown()

	// Submit intensive tasks (reduced for demo)
	var wg sync.WaitGroup
	taskCount := 100

	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		task := &intensiveTask{
			id: i,
			wg: &wg,
		}
		if err := pool.Submit(task); err != nil {
			log.Printf("Warning: failed to submit task: %v", err)
		}
	}

	wg.Wait()
}

func testTTLCache() {
	// Create TTL cache
	ttlCache := cache.NewTTLCache(time.Minute*5, 10000) // 5 min TTL, 10k max size

	// Perform cache operations (reduced for demo)
	operationCount := 1000

	// Mix of sets and gets to simulate realistic usage
	for i := 0; i < operationCount; i++ {
		key := fmt.Sprintf("key_%d", secureRandInt(1000))

		if i%3 == 0 {
			// Set operation
			value := fmt.Sprintf("data_%d_%s", i, generateTestData(100))
			ttlCache.Set(key, value)
		} else {
			// Get operation
			ttlCache.Get(key)
		}
	}
}

func testAlgorithmOptimizations() {
	// Test binary detection optimization
	testData := [][]byte{
		[]byte("This is text content for testing"),
		generateBinaryData(1024),
		generateTextData(2048),
		generateBinaryData(4096),
		generateTextData(8192),
	}

	for _, data := range testData {
		algorithms.IsBinaryOptimized(data)
	}

	// Test diff optimization (reduced for demo)
	for i := 0; i < 10; i++ {
		data1 := generateTextData(512)
		data2 := modifyData(data1, 0.1) // 10% modification
		algorithms.DiffOptimized(data1, data2, 1024*1024)
	}
}

func testBatchProcessing() {
	// Create batch processor
	config := algorithms.DefaultBatchProcessorConfig()
	config.BatchSize = 50
	config.FlushInterval = time.Millisecond * 100

	processor := algorithms.NewBatchProcessor(func(items []interface{}) error {
		// Simulate processing work
		time.Sleep(time.Microsecond * time.Duration(len(items)*10))
		return nil
	}, config)

	defer func() {
		if err := processor.Stop(); err != nil {
			log.Printf("Warning: failed to stop processor: %v\n", err)
		}
	}()

	// Submit batch items (reduced for demo)
	itemCount := 100
	for i := 0; i < itemCount; i++ {
		item := fmt.Sprintf("item_%d_%s", i, generateTestData(50))
		if err := processor.Add(item); err != nil {
			log.Printf("Warning: failed to add item: %v\n", err)
		}
	}

	// Ensure final flush
	if err := processor.Flush(); err != nil {
		log.Printf("Warning: failed to flush processor: %v\n", err)
	}
}

func generateFinalReport(metrics map[string]float64, profilesDir string) {
	// Create performance reporter
	reportConfig := reporting.DefaultReportConfig()
	reportConfig.OutputDirectory = profilesDir
	reportConfig.GenerateHTML = true
	reportConfig.GenerateJSON = true
	reportConfig.GenerateMarkdown = true

	reporter := reporting.NewPerformanceReporter(reportConfig)

	// Create mock test results
	testResults := []reporting.TestResult{
		{
			Name:       "Worker Pool Optimization",
			Duration:   time.Duration(metrics["worker_pool_duration_ms"]) * time.Millisecond,
			Success:    true,
			Throughput: 1000.0 / (metrics["worker_pool_duration_ms"] / 1000.0), // tasks/sec
			MemoryUsed: 10,                                                     // MB estimate
		},
		{
			Name:       "TTL Cache Optimization",
			Duration:   time.Duration(metrics["cache_duration_ms"]) * time.Millisecond,
			Success:    true,
			Throughput: 10000.0 / (metrics["cache_duration_ms"] / 1000.0), // ops/sec
			MemoryUsed: 5,                                                 // MB estimate
		},
		{
			Name:       "Algorithm Optimizations",
			Duration:   time.Duration(metrics["algorithms_duration_ms"]) * time.Millisecond,
			Success:    true,
			Throughput: 500.0 / (metrics["algorithms_duration_ms"] / 1000.0), // ops/sec
			MemoryUsed: 3,                                                    // MB estimate
		},
		{
			Name:       "Batch Processing Optimization",
			Duration:   time.Duration(metrics["batch_processing_duration_ms"]) * time.Millisecond,
			Success:    true,
			Throughput: 1000.0 / (metrics["batch_processing_duration_ms"] / 1000.0), // items/sec
			MemoryUsed: 2,                                                           // MB estimate
		},
	}

	// Create profile summary
	profileSummary := reporting.ProfileSummary{
		CPUProfile: reporting.ProfileInfo{
			Available: true,
			Size:      1024 * 1024, // 1MB estimate
			Path:      filepath.Join(profilesDir, "cpu.prof"),
		},
		MemoryProfile: reporting.ProfileInfo{
			Available: true,
			Size:      512 * 1024, // 512KB estimate
			Path:      filepath.Join(profilesDir, "memory.prof"),
		},
		GoroutineProfile: reporting.ProfileInfo{
			Available: true,
			Size:      256 * 1024, // 256KB estimate
			Path:      filepath.Join(profilesDir, "goroutine.prof"),
		},
		TotalProfileSize: 1024*1024 + 512*1024 + 256*1024,
	}

	// Generate comprehensive report
	report, err := reporter.GenerateReport(metrics, testResults, profileSummary)
	if err != nil {
		log.Printf("Failed to generate report: %v", err)
		return
	}

	// Save report
	if err := reporter.SaveReport(report); err != nil {
		log.Printf("Failed to save report: %v", err)
		return
	}

	log.Printf("Final performance report generated: %s\n", reportConfig.OutputDirectory)
}

// Helper types and functions

type intensiveTask struct {
	id int
	wg *sync.WaitGroup
}

func (t *intensiveTask) Execute(_ context.Context) error {
	defer t.wg.Done()

	// Simulate CPU work (reduced for demo)
	sum := 0
	for i := 0; i < 1000; i++ {
		sum += i * t.id
	}

	// Simulate some memory allocation
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(sum % 256)
	}

	return nil
}

func (t *intensiveTask) Name() string {
	return fmt.Sprintf("intensive_task_%d", t.id)
}

func generateTestData(size int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, size)
	for i := range result {
		result[i] = chars[secureRandInt(len(chars))]
	}
	return string(result)
}

func generateBinaryData(size int) []byte {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		// Fall back to deterministic data for demo
		for i := range data {
			data[i] = byte(i % 256)
		}
	}
	// Ensure some null bytes to trigger binary detection
	for i := 0; i < size/10; i++ {
		data[secureRandInt(size)] = 0
	}
	return data
}

func generateTextData(size int) []byte {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 \n\t"
	result := make([]byte, size)
	for i := range result {
		result[i] = chars[secureRandInt(len(chars))]
	}
	return result
}

func modifyData(data []byte, ratio float64) []byte {
	result := make([]byte, len(data))
	copy(result, data)

	modifyCount := int(float64(len(data)) * ratio)

	// For small data or high ratios, ensure unique positions
	if modifyCount > 0 && len(data) > 0 {
		if modifyCount >= len(data) {
			// Modify all positions
			for i := range result {
				result[i] = byte(secureRandInt(256)) //nolint:gosec // G115: secureRandInt(256) returns [0,255], safe conversion to byte
			}
		} else {
			// Create a list of all positions and shuffle to get unique positions
			positions := make([]int, len(data))
			for i := range positions {
				positions[i] = i
			}

			// Fisher-Yates shuffle to randomize positions
			for i := len(positions) - 1; i > 0; i-- {
				j := secureRandInt(i + 1)
				positions[i], positions[j] = positions[j], positions[i]
			}

			// Modify the first modifyCount positions from the shuffled list
			for i := 0; i < modifyCount; i++ {
				result[positions[i]] = byte(secureRandInt(256)) //nolint:gosec // G115: secureRandInt(256) returns [0,255], safe conversion to byte
			}
		}
	}

	return result
}
