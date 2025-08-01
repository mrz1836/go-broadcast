package sync

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinaryFileMetricsTracking(t *testing.T) {
	// Create a test logger
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.DebugLevel)

	// Create a directory progress reporter
	reporter := NewDirectoryProgressReporter(logger, "/test/directory", 1)

	// Test initial state
	initialMetrics := reporter.GetMetrics()
	assert.Equal(t, 0, initialMetrics.BinaryFilesSkipped)
	assert.Equal(t, int64(0), initialMetrics.BinaryFilesSize)
	assert.Equal(t, 0, initialMetrics.TransformErrors)
	assert.Equal(t, 0, initialMetrics.TransformSuccesses)
	assert.Equal(t, time.Duration(0), initialMetrics.TotalTransformDuration)

	// Record some binary files
	reporter.RecordBinaryFileSkipped(1024) // 1KB file
	reporter.RecordBinaryFileSkipped(2048) // 2KB file
	reporter.RecordBinaryFileSkipped(512)  // 512B file

	// Record some transform operations
	reporter.RecordTransformSuccess(100 * time.Millisecond)
	reporter.RecordTransformSuccess(200 * time.Millisecond)
	reporter.RecordTransformError()
	reporter.RecordTransformSuccess(50 * time.Millisecond)

	// Get updated metrics
	metrics := reporter.GetMetrics()

	// Verify binary file metrics
	assert.Equal(t, 3, metrics.BinaryFilesSkipped, "Should track 3 binary files")
	assert.Equal(t, int64(3584), metrics.BinaryFilesSize, "Should track total binary file size (1024+2048+512)")

	// Verify transform metrics
	assert.Equal(t, 1, metrics.TransformErrors, "Should track 1 transform error")
	assert.Equal(t, 3, metrics.TransformSuccesses, "Should track 3 transform successes")
	assert.Equal(t, 3, metrics.TransformCount, "Should track 3 transforms for averaging")
	assert.Equal(t, 350*time.Millisecond, metrics.TotalTransformDuration, "Should track total transform duration")

	// Verify average calculation
	avgDuration := reporter.GetAverageTransformDuration()
	expectedAvg := 350 * time.Millisecond / 3
	assert.Equal(t, expectedAvg, avgDuration, "Should calculate correct average transform duration")

	// Complete the reporter and verify final metrics
	finalMetrics := reporter.Complete()
	assert.Equal(t, metrics.BinaryFilesSkipped, finalMetrics.BinaryFilesSkipped)
	assert.Equal(t, metrics.BinaryFilesSize, finalMetrics.BinaryFilesSize)
	assert.Equal(t, metrics.TransformErrors, finalMetrics.TransformErrors)
	assert.Equal(t, metrics.TransformSuccesses, finalMetrics.TransformSuccesses)
	assert.False(t, finalMetrics.EndTime.IsZero(), "Should have end time set")
}

func TestBatchProgressWrapperEnhancedReporting(t *testing.T) {
	// Create a test logger
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.DebugLevel)

	// Create a directory progress reporter
	reporter := NewDirectoryProgressReporter(logger, "/test/directory", 1)

	// Create batch progress wrapper
	wrapper := NewBatchProgressWrapper(reporter)

	// Verify it implements both interfaces
	var basicReporter ProgressReporter = wrapper
	var enhancedReporter EnhancedProgressReporter = wrapper

	require.NotNil(t, basicReporter, "Should implement ProgressReporter")
	require.NotNil(t, enhancedReporter, "Should implement EnhancedProgressReporter")

	// Test enhanced reporting methods
	wrapper.RecordBinaryFileSkipped(2048)
	wrapper.RecordTransformSuccess(150 * time.Millisecond)
	wrapper.RecordTransformError()

	// Verify metrics were recorded
	metrics := reporter.GetMetrics()
	assert.Equal(t, 1, metrics.BinaryFilesSkipped)
	assert.Equal(t, int64(2048), metrics.BinaryFilesSize)
	assert.Equal(t, 1, metrics.TransformSuccesses)
	assert.Equal(t, 1, metrics.TransformErrors)
	assert.Equal(t, 150*time.Millisecond, metrics.TotalTransformDuration)
}

func TestDirectoryMetricsWithBinaryFiles(t *testing.T) {
	// Create a test logger
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.DebugLevel)

	// Create a directory progress reporter
	reporter := NewDirectoryProgressReporter(logger, "/test/directory", 1)
	reporter.Start(10) // 10 total files

	// Simulate processing with mixed file types
	reporter.RecordFileProcessed(1024)
	reporter.RecordBinaryFileSkipped(2048) // Binary file

	reporter.RecordFileProcessed(512)
	reporter.RecordTransformSuccess(100 * time.Millisecond) // Text file with transform

	reporter.RecordFileProcessed(256)
	reporter.RecordTransformError() // Text file with transform error

	// Complete and verify comprehensive metrics
	finalMetrics := reporter.Complete()

	assert.Equal(t, 10, finalMetrics.FilesDiscovered, "Should track discovered files from Start()")
	assert.Equal(t, 3, finalMetrics.FilesProcessed, "Should track processed files")
	assert.Equal(t, int64(1792), finalMetrics.ProcessedSize, "Should track processed size (1024+512+256)")

	// Binary file metrics
	assert.Equal(t, 1, finalMetrics.BinaryFilesSkipped, "Should track binary files")
	assert.Equal(t, int64(2048), finalMetrics.BinaryFilesSize, "Should track binary file size")

	// Transform metrics
	assert.Equal(t, 1, finalMetrics.TransformSuccesses, "Should track transform successes")
	assert.Equal(t, 1, finalMetrics.TransformErrors, "Should track transform errors")
	assert.Equal(t, 100*time.Millisecond, finalMetrics.TotalTransformDuration, "Should track transform duration")
	assert.Equal(t, 1, finalMetrics.TransformCount, "Should track transform count")

	assert.False(t, finalMetrics.StartTime.IsZero(), "Should have start time")
	assert.False(t, finalMetrics.EndTime.IsZero(), "Should have end time")
}
