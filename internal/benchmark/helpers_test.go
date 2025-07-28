package benchmark

import (
	"runtime"
	"testing"
	"time"

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
