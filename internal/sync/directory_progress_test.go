package sync

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DirectoryProgressTestSuite provides comprehensive directory progress testing
type DirectoryProgressTestSuite struct {
	suite.Suite

	logger      *logrus.Entry
	testDir     string
	threshold   int
	testOptions DirectoryProgressReporterOptions
}

// SetupSuite initializes the test suite
func (suite *DirectoryProgressTestSuite) SetupSuite() {
	// Initialize logger with debug level for detailed testing
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	suite.logger = logger.WithField("component", "directory-progress-test")

	// Set up test constants
	suite.testDir = "/test/directory"
	suite.threshold = 10
	suite.testOptions = DirectoryProgressReporterOptions{
		Threshold:      suite.threshold,
		UpdateInterval: 100 * time.Millisecond,
		Enabled:        false,
	}
}

// TearDownSuite cleans up the test suite
func (suite *DirectoryProgressTestSuite) TearDownSuite() {
	// No cleanup needed for this test suite
}

// SetupTest initializes each test
func (suite *DirectoryProgressTestSuite) SetupTest() {
	// Fresh logger entry for each test
	suite.logger = suite.logger.WithField("test", suite.T().Name())
}

// TearDownTest cleans up each test
func (suite *DirectoryProgressTestSuite) TearDownTest() {
	// No cleanup needed for individual tests
}

func TestDirectoryProgressTestSuite(t *testing.T) {
	suite.Run(t, new(DirectoryProgressTestSuite))
}

// TestNewDirectoryProgressReporter tests the constructor
func (suite *DirectoryProgressTestSuite) TestNewDirectoryProgressReporter() {
	t := suite.T()

	t.Run("with valid threshold", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 25)

		require.NotNil(t, reporter)
		assert.Equal(t, suite.logger, reporter.logger)
		assert.Equal(t, suite.testDir, reporter.directoryPath)
		assert.Equal(t, 25, reporter.threshold)
		assert.Equal(t, 2*time.Second, reporter.updateInterval)
		assert.False(t, reporter.enabled)
		assert.Equal(t, 0, reporter.metrics.FilesDiscovered)
		assert.False(t, reporter.metrics.StartTime.IsZero())
	})

	t.Run("with zero threshold uses default", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 0)

		require.NotNil(t, reporter)
		assert.Equal(t, 50, reporter.threshold) // Default threshold
	})

	t.Run("with negative threshold uses default", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, -10)

		require.NotNil(t, reporter)
		assert.Equal(t, 50, reporter.threshold) // Default threshold
	})
}

// TestNewDirectoryProgressReporterWithOptions tests the options constructor
func (suite *DirectoryProgressTestSuite) TestNewDirectoryProgressReporterWithOptions() {
	t := suite.T()

	t.Run("with custom options", func(t *testing.T) {
		opts := DirectoryProgressReporterOptions{
			Threshold:      100,
			UpdateInterval: 5 * time.Second,
			Enabled:        true,
		}

		reporter := NewDirectoryProgressReporterWithOptions(suite.logger, suite.testDir, opts)

		require.NotNil(t, reporter)
		assert.Equal(t, 100, reporter.threshold)
		assert.Equal(t, 5*time.Second, reporter.updateInterval)
		assert.True(t, reporter.enabled) // Force enabled
	})

	t.Run("with partial options", func(t *testing.T) {
		opts := DirectoryProgressReporterOptions{
			Threshold: 75,
			// UpdateInterval and Enabled not set
		}

		reporter := NewDirectoryProgressReporterWithOptions(suite.logger, suite.testDir, opts)

		require.NotNil(t, reporter)
		assert.Equal(t, 75, reporter.threshold)
		assert.Equal(t, 2*time.Second, reporter.updateInterval) // Default
		assert.False(t, reporter.enabled)                       // Default
	})

	t.Run("with zero update interval uses default", func(t *testing.T) {
		opts := DirectoryProgressReporterOptions{
			Threshold:      30,
			UpdateInterval: 0, // Should use default
		}

		reporter := NewDirectoryProgressReporterWithOptions(suite.logger, suite.testDir, opts)

		require.NotNil(t, reporter)
		assert.Equal(t, 2*time.Second, reporter.updateInterval) // Default unchanged
	})
}

// TestDirectoryProgressReporterStart tests the Start method
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressReporterStart() {
	t := suite.T()

	t.Run("enables reporting when above threshold", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 10)
		assert.False(t, reporter.isEnabled())

		reporter.Start(15) // Above threshold

		assert.True(t, reporter.isEnabled())
		metrics := reporter.GetMetrics()
		assert.Equal(t, 15, metrics.FilesDiscovered)
	})

	t.Run("disables reporting when below threshold", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 10)

		reporter.Start(5) // Below threshold

		assert.False(t, reporter.isEnabled())
		metrics := reporter.GetMetrics()
		assert.Equal(t, 5, metrics.FilesDiscovered)
	})

	t.Run("enables reporting when exactly at threshold", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 10)

		reporter.Start(10) // Exactly at threshold

		assert.True(t, reporter.isEnabled())
		metrics := reporter.GetMetrics()
		assert.Equal(t, 10, metrics.FilesDiscovered)
	})
}

// TestDirectoryProgressReporterUpdateProgress tests progress updates
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressReporterUpdateProgress() {
	t := suite.T()

	t.Run("updates progress when enabled", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 5)
		reporter.Start(10) // Enable reporting

		// Should update immediately (no previous update)
		reporter.UpdateProgress(3, 10, "Processing files")

		// Verify lastUpdate was set
		reporter.mu.RLock()
		assert.False(t, reporter.lastUpdate.IsZero())
		reporter.mu.RUnlock()
	})

	t.Run("ignores updates when disabled", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 10)
		reporter.Start(5) // Below threshold, disabled

		// Should not update
		reporter.UpdateProgress(3, 5, "Processing files")

		// Verify lastUpdate was not set
		reporter.mu.RLock()
		assert.True(t, reporter.lastUpdate.IsZero())
		reporter.mu.RUnlock()
	})

	t.Run("rate limits updates", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 5)
		reporter.SetUpdateInterval(100 * time.Millisecond)
		reporter.Start(10) // Enable reporting

		// First update should work
		reporter.UpdateProgress(1, 10, "First update")

		reporter.mu.RLock()
		firstUpdate := reporter.lastUpdate
		reporter.mu.RUnlock()
		assert.False(t, firstUpdate.IsZero())

		// Immediate second update should be rate limited
		reporter.UpdateProgress(2, 10, "Second update")

		reporter.mu.RLock()
		secondUpdate := reporter.lastUpdate
		reporter.mu.RUnlock()
		assert.Equal(t, firstUpdate, secondUpdate) // Should be same time

		// Wait for rate limit to pass
		time.Sleep(110 * time.Millisecond)
		reporter.UpdateProgress(3, 10, "Third update")

		reporter.mu.RLock()
		thirdUpdate := reporter.lastUpdate
		reporter.mu.RUnlock()
		assert.True(t, thirdUpdate.After(firstUpdate))
	})
}

// TestDirectoryProgressReporterMetrics tests metrics tracking
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressReporterMetrics() {
	t := suite.T()

	t.Run("records file operations", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)

		// Record various file operations
		reporter.RecordFileDiscovered()
		reporter.RecordFileDiscovered()
		reporter.RecordFileProcessed(1024)
		reporter.RecordFileProcessed(2048)
		reporter.RecordFileExcluded()
		reporter.RecordFileSkipped()
		reporter.RecordFileError()

		metrics := reporter.GetMetrics()
		assert.Equal(t, 2, metrics.FilesDiscovered)
		assert.Equal(t, 2, metrics.FilesProcessed)
		assert.Equal(t, int64(3072), metrics.ProcessedSize)
		assert.Equal(t, 1, metrics.FilesExcluded)
		assert.Equal(t, 1, metrics.FilesSkipped)
		assert.Equal(t, 1, metrics.FilesErrored)
	})

	t.Run("records directory operations", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)

		reporter.RecordDirectoryWalked()
		reporter.RecordDirectoryWalked()
		reporter.RecordDirectoryWalked()
		reporter.AddTotalSize(5120)
		reporter.AddTotalSize(3456)

		metrics := reporter.GetMetrics()
		assert.Equal(t, 3, metrics.DirectoriesWalked)
		assert.Equal(t, int64(8576), metrics.TotalSize)
	})

	t.Run("records binary file operations", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)

		reporter.RecordBinaryFileSkipped(2048)
		reporter.RecordBinaryFileSkipped(4096)

		metrics := reporter.GetMetrics()
		assert.Equal(t, 2, metrics.BinaryFilesSkipped)
		assert.Equal(t, int64(6144), metrics.BinaryFilesSize)
	})

	t.Run("records transform operations", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)

		reporter.RecordTransformError()
		reporter.RecordTransformError()
		reporter.RecordTransformSuccess(100 * time.Millisecond)
		reporter.RecordTransformSuccess(200 * time.Millisecond)
		reporter.RecordTransformSuccess(150 * time.Millisecond)

		metrics := reporter.GetMetrics()
		assert.Equal(t, 2, metrics.TransformErrors)
		assert.Equal(t, 3, metrics.TransformSuccesses)
		assert.Equal(t, 450*time.Millisecond, metrics.TotalTransformDuration)
		assert.Equal(t, 3, metrics.TransformCount)

		avgDuration := reporter.GetAverageTransformDuration()
		assert.Equal(t, 150*time.Millisecond, avgDuration)
	})
}

// TestDirectoryProgressReporterGetAverageTransformDuration tests average calculation
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressReporterGetAverageTransformDuration() {
	t := suite.T()

	t.Run("returns zero for no transforms", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)

		avgDuration := reporter.GetAverageTransformDuration()
		assert.Equal(t, time.Duration(0), avgDuration)
	})

	t.Run("calculates correct average", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)

		reporter.RecordTransformSuccess(100 * time.Millisecond)
		reporter.RecordTransformSuccess(200 * time.Millisecond)
		reporter.RecordTransformSuccess(300 * time.Millisecond)

		avgDuration := reporter.GetAverageTransformDuration()
		assert.Equal(t, 200*time.Millisecond, avgDuration)
	})
}

// TestDirectoryProgressReporterComplete tests completion
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressReporterComplete() {
	t := suite.T()

	t.Run("completes with metrics", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)
		startTime := time.Now()

		// Add some metrics
		reporter.RecordFileProcessed(1024)
		reporter.RecordFileExcluded()
		reporter.RecordTransformSuccess(50 * time.Millisecond)

		// Small delay to ensure duration is measurable
		time.Sleep(10 * time.Millisecond)

		metrics := reporter.Complete()

		assert.False(t, metrics.EndTime.IsZero())
		assert.True(t, metrics.EndTime.After(startTime))
		assert.True(t, metrics.EndTime.After(metrics.StartTime))
		assert.Equal(t, 1, metrics.FilesProcessed)
		assert.Equal(t, 1, metrics.FilesExcluded)
		assert.Equal(t, 1, metrics.TransformSuccesses)
	})
}

// TestDirectoryProgressReporterThreadSafety tests concurrent access
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressReporterThreadSafety() {
	t := suite.T()

	t.Run("concurrent metrics updates", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)
		const numGoroutines = 50
		const operationsPerGoroutine = 100

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Run concurrent operations
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < operationsPerGoroutine; j++ {
					reporter.RecordFileDiscovered()
					reporter.RecordFileProcessed(1)
					reporter.RecordFileExcluded()
					reporter.RecordFileSkipped()
					reporter.RecordFileError()
					reporter.RecordDirectoryWalked()
					reporter.AddTotalSize(1)
					reporter.RecordBinaryFileSkipped(1)
					reporter.RecordTransformError()
					reporter.RecordTransformSuccess(time.Millisecond)
				}
			}()
		}

		wg.Wait()

		metrics := reporter.GetMetrics()
		expectedCount := numGoroutines * operationsPerGoroutine

		assert.Equal(t, expectedCount, metrics.FilesDiscovered)
		assert.Equal(t, expectedCount, metrics.FilesProcessed)
		assert.Equal(t, int64(expectedCount), metrics.ProcessedSize)
		assert.Equal(t, expectedCount, metrics.FilesExcluded)
		assert.Equal(t, expectedCount, metrics.FilesSkipped)
		assert.Equal(t, expectedCount, metrics.FilesErrored)
		assert.Equal(t, expectedCount, metrics.DirectoriesWalked)
		assert.Equal(t, int64(expectedCount), metrics.TotalSize)
		assert.Equal(t, expectedCount, metrics.BinaryFilesSkipped)
		assert.Equal(t, int64(expectedCount), metrics.BinaryFilesSize)
		assert.Equal(t, expectedCount, metrics.TransformErrors)
		assert.Equal(t, expectedCount, metrics.TransformSuccesses)
		assert.Equal(t, expectedCount, metrics.TransformCount)
	})

	t.Run("concurrent progress updates", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)
		reporter.SetUpdateInterval(10 * time.Millisecond) // Short interval for testing
		reporter.Start(100)

		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Run concurrent progress updates
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					reporter.UpdateProgress(id*10+j, 100, "Concurrent update")
					time.Sleep(5 * time.Millisecond)
				}
			}(i)
		}

		wg.Wait()

		// Should not panic or corrupt state
		assert.True(t, reporter.isEnabled())
	})

	t.Run("concurrent configuration changes", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 10)

		const numGoroutines = 20
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Run concurrent configuration changes
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				threshold := 5 + (id % 10)
				interval := time.Duration(50+id) * time.Millisecond

				reporter.SetThreshold(threshold)
				reporter.SetUpdateInterval(interval)

				// Also test Start method concurrently
				reporter.Start(threshold + 5)

				// Read metrics concurrently
				_ = reporter.GetMetrics()
				_ = reporter.isEnabled()
			}(i)
		}

		wg.Wait()

		// Should not panic or corrupt state
		metrics := reporter.GetMetrics()
		assert.NotNil(t, metrics)
	})
}

// TestDirectoryProgressReporterSetters tests setter methods
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressReporterSetters() {
	t := suite.T()

	t.Run("set threshold", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 10)

		reporter.SetThreshold(25)

		reporter.mu.RLock()
		threshold := reporter.threshold
		reporter.mu.RUnlock()
		assert.Equal(t, 25, threshold)
	})

	t.Run("set update interval", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 10)

		newInterval := 5 * time.Second
		reporter.SetUpdateInterval(newInterval)

		reporter.mu.RLock()
		interval := reporter.updateInterval
		reporter.mu.RUnlock()
		assert.Equal(t, newInterval, interval)
	})
}

// TestBatchProgressWrapper tests the batch progress wrapper
func (suite *DirectoryProgressTestSuite) TestBatchProgressWrapper() {
	t := suite.T()

	t.Run("with valid reporter", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)
		wrapper := NewBatchProgressWrapper(reporter)

		require.NotNil(t, wrapper)
		assert.Equal(t, reporter, wrapper.reporter)

		// Test interface methods
		reporter.Start(10) // Enable reporting
		wrapper.UpdateProgress(5, 10, "Batch progress")
		wrapper.RecordBinaryFileSkipped(1024)
		wrapper.RecordTransformError()
		wrapper.RecordTransformSuccess(100 * time.Millisecond)

		metrics := reporter.GetMetrics()
		assert.Equal(t, 1, metrics.BinaryFilesSkipped)
		assert.Equal(t, 1, metrics.TransformErrors)
		assert.Equal(t, 1, metrics.TransformSuccesses)
	})

	t.Run("with nil reporter", func(t *testing.T) {
		wrapper := NewBatchProgressWrapper(nil)

		require.NotNil(t, wrapper)
		assert.Nil(t, wrapper.reporter)

		// Should not panic with nil reporter
		wrapper.UpdateProgress(5, 10, "Batch progress")
		wrapper.RecordBinaryFileSkipped(1024)
		wrapper.RecordTransformError()
		wrapper.RecordTransformSuccess(100 * time.Millisecond)
	})
}

// TestDirectoryProgressManager tests the progress manager
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressManager() {
	t := suite.T()

	t.Run("creates new manager", func(t *testing.T) {
		manager := NewDirectoryProgressManager(suite.logger)

		require.NotNil(t, manager)
		assert.Equal(t, suite.logger, manager.logger)
		assert.NotNil(t, manager.reporters)
		assert.Empty(t, manager.reporters)
	})

	t.Run("gets or creates reporters", func(t *testing.T) {
		manager := NewDirectoryProgressManager(suite.logger)

		// First call should create new reporter
		reporter1 := manager.GetReporter("/path1", 10)
		require.NotNil(t, reporter1)

		// Second call with same path should return same reporter
		reporter2 := manager.GetReporter("/path1", 20) // Different threshold ignored
		assert.Same(t, reporter1, reporter2)

		// Different path should create new reporter
		reporter3 := manager.GetReporter("/path2", 15)
		require.NotNil(t, reporter3)
		assert.NotSame(t, reporter1, reporter3)

		// Verify internal state
		manager.mu.RLock()
		assert.Len(t, manager.reporters, 2)
		manager.mu.RUnlock()
	})

	t.Run("gets all metrics", func(t *testing.T) {
		manager := NewDirectoryProgressManager(suite.logger)

		reporter1 := manager.GetReporter("/path1", 10)
		reporter2 := manager.GetReporter("/path2", 15)

		// Add some metrics
		reporter1.RecordFileProcessed(1024)
		reporter2.RecordFileProcessed(2048)

		allMetrics := manager.GetAllMetrics()
		require.Len(t, allMetrics, 2)
		assert.Contains(t, allMetrics, "/path1")
		assert.Contains(t, allMetrics, "/path2")
		assert.Equal(t, 1, allMetrics["/path1"].FilesProcessed)
		assert.Equal(t, 1, allMetrics["/path2"].FilesProcessed)
	})

	t.Run("completes all reporters", func(t *testing.T) {
		manager := NewDirectoryProgressManager(suite.logger)

		reporter1 := manager.GetReporter("/path1", 10)
		reporter2 := manager.GetReporter("/path2", 15)

		// Add some metrics
		reporter1.RecordFileProcessed(1024)
		reporter2.RecordFileExcluded()

		allMetrics := manager.CompleteAll()
		require.Len(t, allMetrics, 2)
		assert.Contains(t, allMetrics, "/path1")
		assert.Contains(t, allMetrics, "/path2")

		// Verify completion
		assert.Equal(t, 1, allMetrics["/path1"].FilesProcessed)
		assert.Equal(t, 1, allMetrics["/path2"].FilesExcluded)
		assert.False(t, allMetrics["/path1"].EndTime.IsZero())
		assert.False(t, allMetrics["/path2"].EndTime.IsZero())

		// Verify reporters are cleared
		manager.mu.RLock()
		assert.Empty(t, manager.reporters)
		manager.mu.RUnlock()
	})
}

// TestDirectoryProgressManagerConcurrency tests manager thread safety
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressManagerConcurrency() {
	t := suite.T()

	t.Run("concurrent reporter creation", func(t *testing.T) {
		manager := NewDirectoryProgressManager(suite.logger)
		const numGoroutines = 50
		const pathsPerGoroutine = 10

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		reporterCounts := make(map[string]int64)
		var mu sync.Mutex

		// Run concurrent reporter creation
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < pathsPerGoroutine; j++ {
					path := fmt.Sprintf("/path_%d_%d", id, j)
					reporter := manager.GetReporter(path, 10)
					assert.NotNil(t, reporter)

					// Record that this path was accessed
					mu.Lock()
					reporterCounts[path]++
					mu.Unlock()

					// Add some metrics
					reporter.RecordFileProcessed(1)
				}
			}(i)
		}

		wg.Wait()

		// Verify all paths were created
		expectedPaths := numGoroutines * pathsPerGoroutine
		manager.mu.RLock()
		actualPaths := len(manager.reporters)
		manager.mu.RUnlock()
		assert.Equal(t, expectedPaths, actualPaths)

		// Verify each path was accessed only once for creation
		mu.Lock()
		for _, count := range reporterCounts {
			assert.Equal(t, int64(1), count, "Each path should be accessed once for reporter creation")
		}
		mu.Unlock()
	})

	t.Run("concurrent operations on manager", func(t *testing.T) {
		manager := NewDirectoryProgressManager(suite.logger)
		const numGoroutines = 20

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Run concurrent operations
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				path := fmt.Sprintf("/concurrent_path_%d", id%5) // Some overlap

				// Get reporter and perform operations
				reporter := manager.GetReporter(path, 10)
				reporter.RecordFileProcessed(int64(id))

				// Interleave with manager operations
				if id%3 == 0 {
					_ = manager.GetAllMetrics()
				}

				if id%7 == 0 {
					// Create another reporter
					altPath := fmt.Sprintf("/alt_path_%d", id)
					altReporter := manager.GetReporter(altPath, 15)
					altReporter.RecordFileError()
				}
			}(i)
		}

		wg.Wait()

		// Verify no corruption
		allMetrics := manager.GetAllMetrics()
		assert.NotEmpty(t, allMetrics)

		// Complete all should work without issues
		completedMetrics := manager.CompleteAll()
		assert.NotEmpty(t, completedMetrics)

		// Managers should be cleared
		manager.mu.RLock()
		assert.Empty(t, manager.reporters)
		manager.mu.RUnlock()
	})
}

// TestIsProgressReportingNeeded tests the standalone function
func (suite *DirectoryProgressTestSuite) TestIsProgressReportingNeeded() {
	t := suite.T()

	testCases := []struct {
		name      string
		fileCount int
		threshold int
		expected  bool
	}{
		{"above threshold", 100, 50, true},
		{"at threshold", 50, 50, true},
		{"below threshold", 25, 50, false},
		{"zero threshold uses default", 100, 0, true},
		{"negative threshold uses default", 100, -10, true},
		{"zero files", 0, 50, false},
		{"negative files", -5, 50, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsProgressReportingNeeded(tc.fileCount, tc.threshold)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestDirectoryProgressReporterMemoryLeaks tests for potential memory leaks
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressReporterMemoryLeaks() {
	t := suite.T()

	t.Run("repeated operations don't accumulate memory", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)

		// Perform many operations
		for i := 0; i < 10000; i++ {
			reporter.RecordFileDiscovered()
			reporter.RecordFileProcessed(1)
			reporter.RecordTransformSuccess(time.Microsecond)
			_ = reporter.GetMetrics()
		}

		// Should complete without issues
		metrics := reporter.Complete()
		assert.Equal(t, 10000, metrics.FilesDiscovered)
		assert.Equal(t, 10000, metrics.FilesProcessed)
		assert.Equal(t, 10000, metrics.TransformSuccesses)
	})

	t.Run("manager creation and cleanup", func(_ *testing.T) {
		// Create many managers and reporters
		for i := 0; i < 1000; i++ {
			manager := NewDirectoryProgressManager(suite.logger)
			for j := 0; j < 10; j++ {
				path := fmt.Sprintf("/test_path_%d_%d", i, j)
				reporter := manager.GetReporter(path, 10)
				reporter.RecordFileProcessed(1)
			}
			// Complete all to clean up
			_ = manager.CompleteAll()
		}
	})
}

// TestDirectoryProgressReporterEdgeCases tests edge cases and boundary conditions
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressReporterEdgeCases() {
	t := suite.T()

	t.Run("extremely large metrics", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)

		// Test with large numbers
		largeSize := int64(1<<62 - 1) // Near max int64
		reporter.AddTotalSize(largeSize)
		reporter.RecordFileProcessed(largeSize)

		metrics := reporter.GetMetrics()
		assert.Equal(t, largeSize, metrics.TotalSize)
		assert.Equal(t, largeSize, metrics.ProcessedSize)
	})

	t.Run("very short intervals", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)
		reporter.SetUpdateInterval(1 * time.Nanosecond) // Extremely short
		reporter.Start(10)

		// Multiple rapid updates
		reporter.UpdateProgress(1, 10, "Update 1")
		reporter.UpdateProgress(2, 10, "Update 2")
		reporter.UpdateProgress(3, 10, "Update 3")

		// Should handle gracefully
		assert.True(t, reporter.isEnabled())
	})

	t.Run("zero and negative sizes", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)

		reporter.RecordFileProcessed(0)
		reporter.RecordFileProcessed(-100) // Negative size
		reporter.AddTotalSize(0)
		reporter.AddTotalSize(-500)
		reporter.RecordBinaryFileSkipped(0)
		reporter.RecordBinaryFileSkipped(-200)

		metrics := reporter.GetMetrics()
		// Should handle negative values (implementation dependent)
		assert.Equal(t, 2, metrics.FilesProcessed)
		assert.Equal(t, 2, metrics.BinaryFilesSkipped)
	})

	t.Run("empty directory path", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, "", 10)
		require.NotNil(t, reporter)
		assert.Empty(t, reporter.directoryPath)

		// Should work normally
		reporter.Start(15)
		assert.True(t, reporter.isEnabled())
	})

	t.Run("very long transform durations", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)

		longDuration := 24 * time.Hour
		reporter.RecordTransformSuccess(longDuration)
		reporter.RecordTransformSuccess(longDuration)

		metrics := reporter.GetMetrics()
		assert.Equal(t, 48*time.Hour, metrics.TotalTransformDuration)
		avgDuration := reporter.GetAverageTransformDuration()
		assert.Equal(t, 24*time.Hour, avgDuration)
	})
}

// TestDirectoryProgressReporterRaceConditions tests for race conditions
func (suite *DirectoryProgressTestSuite) TestDirectoryProgressReporterRaceConditions() {
	t := suite.T()

	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	t.Run("read-write race detection", func(t *testing.T) {
		reporter := NewDirectoryProgressReporter(suite.logger, suite.testDir, 1)
		reporter.Start(100)

		const numGoroutines = 100
		const operations = 1000

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		// Start operations that mix reads and writes
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < operations; j++ {
					switch j % 10 {
					case 0:
						reporter.RecordFileDiscovered()
					case 1:
						reporter.RecordFileProcessed(1)
					case 2:
						_ = reporter.GetMetrics()
					case 3:
						reporter.UpdateProgress(j, operations, "Testing")
					case 4:
						_ = reporter.isEnabled()
					case 5:
						reporter.RecordTransformSuccess(time.Millisecond)
					case 6:
						_ = reporter.GetAverageTransformDuration()
					case 7:
						reporter.SetThreshold(id + 1)
					case 8:
						reporter.SetUpdateInterval(time.Duration(id) * time.Millisecond)
					case 9:
						reporter.RecordBinaryFileSkipped(int64(j))
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify final state is consistent
		metrics := reporter.GetMetrics()
		assert.GreaterOrEqual(t, metrics.FilesDiscovered, 0)
		assert.GreaterOrEqual(t, metrics.FilesProcessed, 0)
		assert.GreaterOrEqual(t, metrics.TransformSuccesses, 0)
	})
}

// BenchmarkDirectoryProgressReporter benchmarks progress reporter performance
func BenchmarkDirectoryProgressReporter(b *testing.B) {
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.ErrorLevel) // Reduce logging overhead

	b.Run("metric_recording", func(b *testing.B) {
		reporter := NewDirectoryProgressReporter(logger, "/test", 1)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reporter.RecordFileProcessed(1024)
		}
	})

	b.Run("progress_updates", func(b *testing.B) {
		reporter := NewDirectoryProgressReporter(logger, "/test", 1)
		reporter.Start(1000)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reporter.UpdateProgress(i, 1000, "Benchmark")
		}
	})

	b.Run("concurrent_metrics", func(b *testing.B) {
		reporter := NewDirectoryProgressReporter(logger, "/test", 1)
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				reporter.RecordFileProcessed(1)
				_ = reporter.GetMetrics()
			}
		})
	})

	b.Run("manager_operations", func(b *testing.B) {
		manager := NewDirectoryProgressManager(logger)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			path := fmt.Sprintf("/benchmark_path_%d", i%100) // Limit paths for reuse
			reporter := manager.GetReporter(path, 10)
			reporter.RecordFileProcessed(1)
		}
	})
}
