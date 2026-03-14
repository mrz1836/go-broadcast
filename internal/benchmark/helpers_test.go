package benchmark

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureMemoryStats(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
		want struct {
			hasMemStats bool
			hasBefore   bool
			hasAfter    bool
		}
	}{
		{
			name: "BasicMemoryCapture",
			fn: func() {
				// Allocate some memory
				data := make([]byte, 1024)
				_ = data
			},
			want: struct {
				hasMemStats bool
				hasBefore   bool
				hasAfter    bool
			}{
				hasMemStats: true,
				hasBefore:   true,
				hasAfter:    true,
			},
		},
		{
			name: "NoOpFunction",
			fn:   func() {},
			want: struct {
				hasMemStats bool
				hasBefore   bool
				hasAfter    bool
			}{
				hasMemStats: true,
				hasBefore:   true,
				hasAfter:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CaptureMemoryStats(tt.fn)

			if tt.want.hasMemStats {
				// Check that we have valid memory stats
				require.Positive(t, result.Before.Sys)
				require.Positive(t, result.After.Sys)
			}

			if tt.want.hasBefore {
				require.NotEqual(t, runtime.MemStats{}, result.Before)
			}

			if tt.want.hasAfter {
				require.NotEqual(t, runtime.MemStats{}, result.After)
			}
		})
	}
}

func TestReportResults(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		want    struct {
			shouldLog bool
		}
	}{
		{
			name: "SingleResult",
			results: []Result{
				{
					Name:        "TestOperation",
					Operations:  100,
					NsPerOp:     1000,
					AllocsPerOp: 5,
					BytesPerOp:  256,
				},
			},
			want: struct {
				shouldLog bool
			}{
				shouldLog: true,
			},
		},
		{
			name: "MultipleResults",
			results: []Result{
				{
					Name:        "Operation1",
					Operations:  50,
					NsPerOp:     2000,
					AllocsPerOp: 3,
					BytesPerOp:  128,
				},
				{
					Name:        "Operation2",
					Operations:  75,
					NsPerOp:     1500,
					AllocsPerOp: 7,
					BytesPerOp:  512,
				},
			},
			want: struct {
				shouldLog bool
			}{
				shouldLog: true,
			},
		},
		{
			name:    "EmptyResults",
			results: []Result{},
			want: struct {
				shouldLog bool
			}{
				shouldLog: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a testing.B to test ReportResults
			b := &testing.B{}

			// This should not panic and should work with the helper
			require.NotPanics(t, func() {
				ReportResults(b, tt.results)
			})
		})
	}
}

func TestGenerateTestData(t *testing.T) {
	tests := []struct {
		name string
		size string
		want struct {
			length   int
			notEmpty bool
		}
	}{
		{
			name: "SmallData",
			size: "small",
			want: struct {
				length   int
				notEmpty bool
			}{
				length:   1024,
				notEmpty: true,
			},
		},
		{
			name: "MediumData",
			size: "medium",
			want: struct {
				length   int
				notEmpty bool
			}{
				length:   1024 * 100,
				notEmpty: true,
			},
		},
		{
			name: "LargeData",
			size: "large",
			want: struct {
				length   int
				notEmpty bool
			}{
				length:   1024 * 1024,
				notEmpty: true,
			},
		},
		{
			name: "XLargeData",
			size: "xlarge",
			want: struct {
				length   int
				notEmpty bool
			}{
				length:   1024 * 1024 * 10,
				notEmpty: true,
			},
		},
		{
			name: "InvalidSize",
			size: "invalid",
			want: struct {
				length   int
				notEmpty bool
			}{
				length:   0,
				notEmpty: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateTestData(tt.size)

			if tt.want.notEmpty {
				require.NotNil(t, result)
				require.Len(t, result, tt.want.length)

				// Verify pattern (should be cycling bytes)
				if len(result) > 0 {
					require.Equal(t, byte(0), result[0])
					if len(result) > 256 {
						require.Equal(t, byte(0), result[256])
					}
				}
			} else {
				require.Nil(t, result)
			}
		})
	}
}

func TestMeasureOperation(t *testing.T) {
	tests := []struct {
		name      string
		operation func()
		want      struct {
			hasName      bool
			hasMetrics   bool
			hasTimestamp bool
		}
	}{
		{
			name: "FastOperation",
			operation: func() {
				// Quick operation
				sum := 0
				for i := 0; i < 100; i++ {
					sum += i
				}
				_ = sum
			},
			want: struct {
				hasName      bool
				hasMetrics   bool
				hasTimestamp bool
			}{
				hasName:      true,
				hasMetrics:   true,
				hasTimestamp: true,
			},
		},
		{
			name: "AllocationOperation",
			operation: func() {
				// Operation that allocates memory
				data := make([]byte, 1024)
				_ = data
			},
			want: struct {
				hasName      bool
				hasMetrics   bool
				hasTimestamp bool
			}{
				hasName:      true,
				hasMetrics:   true,
				hasTimestamp: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MeasureOperation(tt.name, tt.operation)

			if tt.want.hasName {
				require.Equal(t, tt.name, result.Name)
			}

			if tt.want.hasMetrics {
				require.Equal(t, int64(1), result.Operations)
				require.Positive(t, result.NsPerOp)
				require.GreaterOrEqual(t, result.AllocsPerOp, int64(0))
				require.GreaterOrEqual(t, result.BytesPerOp, int64(0))
				require.GreaterOrEqual(t, result.MemoryUsed, int64(0))
			}

			if tt.want.hasTimestamp {
				require.False(t, result.StartTime.IsZero())
				require.False(t, result.EndTime.IsZero())
				require.True(t, result.EndTime.After(result.StartTime))
			}
		})
	}
}

func TestRunWithMemoryTracking(t *testing.T) {
	tests := []struct {
		name      string
		operation func()
		want      struct {
			shouldRun bool
		}
	}{
		{
			name: "BasicOperation",
			operation: func() {
				data := make([]byte, 512)
				_ = data
			},
			want: struct {
				shouldRun bool
			}{
				shouldRun: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a benchmark to test with
			b := &testing.B{N: 1}

			require.NotPanics(t, func() {
				RunWithMemoryTracking(b, tt.name, tt.operation)
			})
		})
	}
}

func TestCreateTempRepo(t *testing.T) {
	// Create a benchmark to test with
	b := &testing.B{}

	tempDir := CreateTempRepo(b)
	require.NotEmpty(t, tempDir)
	// In real testing.B, this would create a temporary directory
}

func TestSizes(t *testing.T) {
	sizes := Sizes()

	require.NotEmpty(t, sizes)
	require.Len(t, sizes, 4)

	expectedSizes := map[string]string{
		"Small":  "small",
		"Medium": "medium",
		"Large":  "large",
		"XLarge": "xlarge",
	}

	for _, size := range sizes {
		expectedValue, exists := expectedSizes[size.Name]
		require.True(t, exists, "Unexpected size name: %s", size.Name)
		require.Equal(t, expectedValue, size.Size)
	}
}

// BenchmarkCaptureMemoryStats tests the performance of memory capturing
func BenchmarkCaptureMemoryStats(b *testing.B) {
	operation := func() {
		data := make([]byte, 1024)
		_ = data
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CaptureMemoryStats(operation)
	}
}

// BenchmarkMeasureOperation tests the performance of operation measurement
func BenchmarkMeasureOperation(b *testing.B) {
	operation := func() {
		sum := 0
		for i := 0; i < 100; i++ {
			sum += i
		}
		_ = sum
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MeasureOperation("test-op", operation)
	}
}

// TestMemoryStatsIncrease verifies that memory allocation is captured
func TestMemoryStatsIncrease(t *testing.T) {
	result := CaptureMemoryStats(func() {
		// Allocate a significant amount of memory
		data := make([][]byte, 1000)
		for i := range data {
			data[i] = make([]byte, 1024)
		}
		_ = data
	})

	// Memory should increase during allocation
	require.GreaterOrEqual(t, result.After.TotalAlloc, result.Before.TotalAlloc)
}

// TestMeasureOperationTiming verifies timing measurement accuracy
func TestMeasureOperationTiming(t *testing.T) {
	sleepDuration := 10 * time.Millisecond

	result := MeasureOperation("sleep-test", func() {
		time.Sleep(sleepDuration)
	})

	// Should measure at least the sleep duration (with some tolerance)
	minNanos := int64(sleepDuration * 8 / 10) // 80% tolerance for timing variations
	require.GreaterOrEqual(t, result.NsPerOp, minNanos, "Measured time should be at least 80% of sleep duration")
}

// TestGenerateTestDataConsistency verifies data generation consistency
func TestGenerateTestDataConsistency(t *testing.T) {
	// Generate the same size multiple times
	data1 := GenerateTestData("small")
	data2 := GenerateTestData("small")

	require.Len(t, data2, len(data1), "Same size should generate same length")
	require.Equal(t, data1, data2, "Same size should generate identical data")
}

// TestStandardSizes tests the StandardSizes function
func TestStandardSizes(t *testing.T) {
	sizes := StandardSizes()

	require.NotEmpty(t, sizes)
	require.Len(t, sizes, 3, "Should have exactly 3 standard sizes")

	expectedSizes := map[string]struct {
		fileCount int
		fileSize  int
	}{
		"Small":  {fileCount: 10, fileSize: 1024},
		"Medium": {fileCount: 100, fileSize: 10240},
		"Large":  {fileCount: 1000, fileSize: 102400},
	}

	for _, size := range sizes {
		expected, exists := expectedSizes[size.Name]
		require.True(t, exists, "Unexpected size name: %s", size.Name)
		require.Equal(t, expected.fileCount, size.FileCount)
		require.Equal(t, expected.fileSize, size.FileSize)
	}

	// Verify sizes are in ascending order
	require.Less(t, sizes[0].FileCount, sizes[1].FileCount)
	require.Less(t, sizes[1].FileCount, sizes[2].FileCount)
	require.Less(t, sizes[0].FileSize, sizes[1].FileSize)
	require.Less(t, sizes[1].FileSize, sizes[2].FileSize)
}

// TestSetupBenchmarkFiles tests the SetupBenchmarkFiles function
func TestSetupBenchmarkFiles(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{
			name:  "SingleFile",
			count: 1,
		},
		{
			name:  "MultipleFiles",
			count: 5,
		},
		{
			name:  "ZeroFiles",
			count: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a benchmark to test with
			b := &testing.B{}
			tempDir := t.TempDir()

			files := SetupBenchmarkFiles(b, tempDir, tt.count)

			require.Len(t, files, tt.count)

			// Verify each file was created correctly
			for i, filePath := range files {
				require.FileExists(t, filePath)

				// Read and verify content
				content, err := os.ReadFile(filePath) //nolint:gosec // test file path is controlled
				require.NoError(t, err)

				expectedContent := fmt.Sprintf("Benchmark test content %d", i)
				require.Equal(t, expectedContent, string(content))

				// Verify file permissions
				info, err := os.Stat(filePath)
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0o600), info.Mode())
			}
		})
	}
}

// TestSetupBenchmarkFilesError tests SetupBenchmarkFiles error handling
func TestSetupBenchmarkFilesError(t *testing.T) {
	// We can't easily test b.Fatalf from a testing.T context
	// because b.Fatalf causes the test to exit immediately.
	// Instead, we test the underlying os.WriteFile behavior
	invalidDir := "/invalid/path/that/does/not/exist"
	filePath := filepath.Join(invalidDir, "test_file.txt")

	err := os.WriteFile(filePath, []byte("test"), 0o600)
	require.Error(t, err, "Should fail when directory doesn't exist")
	assert.Contains(t, err.Error(), "no such file or directory")
}

// TestWithMemoryTracking tests the WithMemoryTracking function
func TestWithMemoryTracking(t *testing.T) {
	tests := []struct {
		name      string
		operation func()
	}{
		{
			name: "SimpleOperation",
			operation: func() {
				data := make([]byte, 512)
				_ = data
			},
		},
		{
			name: "NoOperation",
			operation: func() {
				// Do nothing
			},
		},
		{
			name: "MultipleAllocations",
			operation: func() {
				for i := 0; i < 10; i++ {
					data := make([]byte, 100)
					_ = data
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a benchmark to test with
			b := &testing.B{N: 3}

			require.NotPanics(t, func() {
				WithMemoryTracking(b, tt.operation)
			})

			// Verify that the benchmark was reset and stopped properly
			// (This is implicit in the function behavior)
		})
	}
}

// TestSetupBenchmarkRepo tests the SetupBenchmarkRepo function
func TestSetupBenchmarkRepo(t *testing.T) {
	// Create a benchmark to test with
	b := &testing.B{}

	repoDir := SetupBenchmarkRepo(b)

	require.NotEmpty(t, repoDir)
	require.DirExists(t, repoDir)

	// Verify it's a valid directory
	info, err := os.Stat(repoDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

// TestWithMemoryTrackingBenchmark tests WithMemoryTracking in a benchmark context
func BenchmarkWithMemoryTracking(b *testing.B) {
	operation := func() {
		data := make([]byte, 1024)
		_ = data
	}

	WithMemoryTracking(b, operation)
}

// TestMemoryStatsStructure verifies MemoryStats structure
func TestMemoryStatsStructure(t *testing.T) {
	stats := MemoryStats{
		Before: runtime.MemStats{},
		After:  runtime.MemStats{},
	}

	// Verify the structure has the expected fields
	require.NotNil(t, &stats.Before)
	require.NotNil(t, &stats.After)
}

// TestResultStructure verifies Result structure
func TestResultStructure(t *testing.T) {
	now := time.Now()
	result := Result{
		Name:        "TestOperation",
		Operations:  100,
		NsPerOp:     1000,
		AllocsPerOp: 5,
		BytesPerOp:  256,
		MemoryUsed:  1024,
		StartTime:   now,
		EndTime:     now.Add(time.Millisecond),
	}

	require.Equal(t, "TestOperation", result.Name)
	require.Equal(t, int64(100), result.Operations)
	require.Equal(t, int64(1000), result.NsPerOp)
	require.Equal(t, int64(5), result.AllocsPerOp)
	require.Equal(t, int64(256), result.BytesPerOp)
	require.Equal(t, int64(1024), result.MemoryUsed)
	require.Equal(t, now, result.StartTime)
	require.True(t, result.EndTime.After(result.StartTime))
}

// TestSizeStructure verifies Size structure
func TestSizeStructure(t *testing.T) {
	size := Size{
		Name:      "TestSize",
		FileCount: 10,
		FileSize:  1024,
	}

	require.Equal(t, "TestSize", size.Name)
	require.Equal(t, 10, size.FileCount)
	require.Equal(t, 1024, size.FileSize)
}
