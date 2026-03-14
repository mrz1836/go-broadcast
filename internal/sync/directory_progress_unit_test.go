package sync

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsProgressReportingNeeded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fileCount int
		threshold int
		expected  bool
	}{
		{
			name:      "below threshold",
			fileCount: 10,
			threshold: 50,
			expected:  false,
		},
		{
			name:      "at threshold",
			fileCount: 50,
			threshold: 50,
			expected:  true,
		},
		{
			name:      "above threshold",
			fileCount: 100,
			threshold: 50,
			expected:  true,
		},
		{
			name:      "zero threshold defaults to 50",
			fileCount: 49,
			threshold: 0,
			expected:  false,
		},
		{
			name:      "zero threshold with 50 files",
			fileCount: 50,
			threshold: 0,
			expected:  true,
		},
		{
			name:      "negative threshold defaults to 50",
			fileCount: 50,
			threshold: -1,
			expected:  true,
		},
		{
			name:      "zero files",
			fileCount: 0,
			threshold: 50,
			expected:  false,
		},
		{
			name:      "custom small threshold",
			fileCount: 5,
			threshold: 3,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsProgressReportingNeeded(tt.fileCount, tt.threshold)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetThreshold(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	entry := logger.WithField("test", true)

	reporter := NewDirectoryProgressReporter(entry, "/tmp/test", 50)
	assert.Equal(t, 50, reporter.threshold)

	reporter.SetThreshold(100)
	assert.Equal(t, 100, reporter.threshold)

	reporter.SetThreshold(1)
	assert.Equal(t, 1, reporter.threshold)
}

func TestSetUpdateInterval(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	entry := logger.WithField("test", true)

	reporter := NewDirectoryProgressReporter(entry, "/tmp/test", 50)
	assert.Equal(t, 2*time.Second, reporter.updateInterval)

	reporter.SetUpdateInterval(5 * time.Second)
	assert.Equal(t, 5*time.Second, reporter.updateInterval)
}

func TestNewDirectoryProgressReporterWithOptions(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	entry := logger.WithField("test", true)

	t.Run("default options", func(t *testing.T) {
		t.Parallel()

		opts := DirectoryProgressReporterOptions{
			Threshold: 100,
		}
		reporter := NewDirectoryProgressReporterWithOptions(entry, "/tmp/test", opts)
		require.NotNil(t, reporter)
		assert.Equal(t, 100, reporter.threshold)
		assert.False(t, reporter.isEnabled())
	})

	t.Run("with update interval", func(t *testing.T) {
		t.Parallel()

		opts := DirectoryProgressReporterOptions{
			Threshold:      50,
			UpdateInterval: 5 * time.Second,
		}
		reporter := NewDirectoryProgressReporterWithOptions(entry, "/tmp/test", opts)
		require.NotNil(t, reporter)
		assert.Equal(t, 5*time.Second, reporter.updateInterval)
	})

	t.Run("with enabled flag", func(t *testing.T) {
		t.Parallel()

		opts := DirectoryProgressReporterOptions{
			Threshold: 10,
			Enabled:   true,
		}
		reporter := NewDirectoryProgressReporterWithOptions(entry, "/tmp/test", opts)
		require.NotNil(t, reporter)
		assert.True(t, reporter.isEnabled())
	})

	t.Run("zero threshold gets default", func(t *testing.T) {
		t.Parallel()

		opts := DirectoryProgressReporterOptions{
			Threshold: 0,
		}
		reporter := NewDirectoryProgressReporterWithOptions(entry, "/tmp/test", opts)
		require.NotNil(t, reporter)
		assert.Equal(t, 50, reporter.threshold) // Default
	})
}

func TestDirectoryProgressReporter_RecordMethods(t *testing.T) {
	t.Parallel()

	logger := logrus.New()
	entry := logger.WithField("test", true)

	t.Run("record binary file skipped", func(t *testing.T) {
		t.Parallel()

		reporter := NewDirectoryProgressReporter(entry, "/tmp/test", 50)
		reporter.RecordBinaryFileSkipped(1024)
		reporter.RecordBinaryFileSkipped(2048)

		metrics := reporter.GetMetrics()
		assert.Equal(t, 2, metrics.BinaryFilesSkipped)
		assert.Equal(t, int64(3072), metrics.BinaryFilesSize)
	})

	t.Run("record transform error", func(t *testing.T) {
		t.Parallel()

		reporter := NewDirectoryProgressReporter(entry, "/tmp/test", 50)
		reporter.RecordTransformError()
		reporter.RecordTransformError()

		metrics := reporter.GetMetrics()
		assert.Equal(t, 2, metrics.TransformErrors)
	})

	t.Run("record transform success", func(t *testing.T) {
		t.Parallel()

		reporter := NewDirectoryProgressReporter(entry, "/tmp/test", 50)
		reporter.RecordTransformSuccess(100 * time.Millisecond)
		reporter.RecordTransformSuccess(200 * time.Millisecond)

		metrics := reporter.GetMetrics()
		assert.Equal(t, 2, metrics.TransformSuccesses)
		assert.Equal(t, 2, metrics.TransformCount)
		assert.Equal(t, 300*time.Millisecond, metrics.TotalTransformDuration)
	})

	t.Run("average transform duration", func(t *testing.T) {
		t.Parallel()

		reporter := NewDirectoryProgressReporter(entry, "/tmp/test", 50)

		// No transforms yet
		assert.Equal(t, time.Duration(0), reporter.GetAverageTransformDuration())

		reporter.RecordTransformSuccess(100 * time.Millisecond)
		reporter.RecordTransformSuccess(300 * time.Millisecond)

		assert.Equal(t, 200*time.Millisecond, reporter.GetAverageTransformDuration())
	})
}
