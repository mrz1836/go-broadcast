package pool

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain ensures test isolation by resetting the default pool stats before all tests
func TestMain(m *testing.M) {
	ResetStats()
	os.Exit(m.Run())
}

// TestNewBufferPool tests buffer pool creation
func TestNewBufferPool(t *testing.T) {
	bp := NewBufferPool()
	require.NotNil(t, bp)
	require.NotNil(t, bp.smallBufferPool)
	require.NotNil(t, bp.mediumBufferPool)
	require.NotNil(t, bp.largeBufferPool)
}

// TestGetBuffer tests buffer retrieval with different sizes
func TestGetBuffer(t *testing.T) {
	bp := NewBufferPool()

	testCases := []struct {
		name         string
		size         int
		expectedPool string
	}{
		{"SmallBuffer", 512, "small"},
		{"SmallBufferMax", SmallBufferThreshold, "small"},
		{"MediumBuffer", SmallBufferThreshold + 1, "medium"},
		{"MediumBufferMax", MediumBufferThreshold, "medium"},
		{"LargeBuffer", MediumBufferThreshold + 1, "large"},
		{"LargeBufferMax", LargeBufferThreshold, "large"},
		{"OversizedBuffer", LargeBufferThreshold + 1, "oversized"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := bp.GetBuffer(tc.size)
			require.NotNil(t, buf)
			assert.GreaterOrEqual(t, buf.Cap(), tc.size)

			// Verify buffer is ready to use
			assert.Equal(t, 0, buf.Len())
		})
	}
}

// TestPutBuffer tests buffer return to pool
func TestPutBuffer(t *testing.T) {
	bp := NewBufferPool()

	t.Run("ReturnSmallBuffer", func(t *testing.T) {
		buf := bp.GetBuffer(100)
		buf.WriteString("test data")

		// Buffer should have content before returning
		assert.Positive(t, buf.Len())

		bp.PutBuffer(buf)

		// Buffer should be reset after returning
		assert.Equal(t, 0, buf.Len())
	})

	t.Run("ReturnNilBuffer", func(_ *testing.T) {
		// Should not panic
		bp.PutBuffer(nil)
	})

	t.Run("ReturnOversizedBuffer", func(t *testing.T) {
		// Create oversized buffer directly
		buf := bytes.NewBuffer(make([]byte, 0, MaxPoolableSize+1))
		buf.WriteString("oversized data")

		initialOversized := bp.GetStats().Oversized
		bp.PutBuffer(buf)

		// Should increment oversized counter
		assert.Equal(t, initialOversized+1, bp.GetStats().Oversized)
	})
}

// TestBufferReuse tests that buffers are actually reused via pool statistics
func TestBufferReuse(t *testing.T) {
	bp := NewBufferPool()
	bp.ResetStats()

	// First cycle: get and return
	buf1 := bp.GetBuffer(100)
	buf1.WriteString("first use")
	bp.PutBuffer(buf1)
	_ = buf1 // Explicit: we no longer own this buffer after Put

	// Second cycle: get another buffer of same size
	buf2 := bp.GetBuffer(100)

	// Verify buffer is ready to use (reset)
	assert.Equal(t, 0, buf2.Len())

	// Verify pool statistics show proper usage
	// (We can't reliably assert pointer identity due to sync.Pool behavior)
	stats := bp.GetStats()
	assert.Equal(t, int64(2), stats.SmallPool.Gets)
	assert.Equal(t, int64(1), stats.SmallPool.Puts)

	bp.PutBuffer(buf2)
}

// TestDefaultPool tests the package-level default pool
func TestDefaultPool(t *testing.T) {
	// Reset stats to get clean counts
	ResetStats()

	buf := GetBuffer(100)
	require.NotNil(t, buf)

	PutBuffer(buf)

	stats := GetStats()
	assert.Equal(t, int64(1), stats.SmallPool.Gets)
	assert.Equal(t, int64(1), stats.SmallPool.Puts)
}

// TestWithBuffer tests the WithBuffer helper function
func TestWithBuffer(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var result string
		err := WithBuffer(1024, func(buf *bytes.Buffer) error {
			buf.WriteString("Hello, ")
			buf.WriteString("World!")
			result = buf.String()
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", result)
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("processing failed") //nolint:err113 // test error
		err := WithBuffer(1024, func(buf *bytes.Buffer) error {
			buf.WriteString("some data")
			return expectedErr
		})

		assert.Equal(t, expectedErr, err)
	})
}

// TestWithBufferResult tests the WithBufferResult helper function
func TestWithBufferResult(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		result, err := WithBufferResult[string](1024, func(buf *bytes.Buffer) (string, error) {
			buf.WriteString("Result data")
			return buf.String(), nil
		})

		require.NoError(t, err)
		assert.Equal(t, "Result data", result)
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("processing failed") //nolint:err113 // test error
		result, err := WithBufferResult[string](1024, func(_ *bytes.Buffer) (string, error) {
			return "", expectedErr
		})

		assert.Equal(t, expectedErr, err)
		assert.Empty(t, result)
	})
}

// TestStats tests statistics tracking
func TestStats(t *testing.T) {
	bp := NewBufferPool()
	bp.ResetStats()

	// Perform various operations
	smallBuf := bp.GetBuffer(100)
	bp.PutBuffer(smallBuf)

	mediumBuf := bp.GetBuffer(5000)
	bp.PutBuffer(mediumBuf)

	largeBuf := bp.GetBuffer(50000)
	bp.PutBuffer(largeBuf)

	oversizedBuf := bp.GetBuffer(200000)
	// Don't return oversized buffer
	_ = oversizedBuf

	stats := bp.GetStats()

	assert.Equal(t, int64(1), stats.SmallPool.Gets)
	assert.Equal(t, int64(1), stats.SmallPool.Puts)
	assert.Equal(t, int64(1), stats.MediumPool.Gets)
	assert.Equal(t, int64(1), stats.MediumPool.Puts)
	assert.Equal(t, int64(1), stats.LargePool.Gets)
	assert.Equal(t, int64(1), stats.LargePool.Puts)
	assert.Equal(t, int64(1), stats.Oversized)
	assert.Equal(t, int64(3), stats.Resets) // Three buffers were reset
}

// TestMetricsReturnRate tests return rate calculation
func TestMetricsReturnRate(t *testing.T) {
	testCases := []struct {
		name     string
		metrics  Metrics
		expected float64
	}{
		{
			name:     "PerfectReturnRate",
			metrics:  Metrics{Gets: 100, Puts: 100},
			expected: 100.0,
		},
		{
			name:     "HalfReturnRate",
			metrics:  Metrics{Gets: 100, Puts: 50},
			expected: 50.0,
		},
		{
			name:     "ZeroGets",
			metrics:  Metrics{Gets: 0, Puts: 0},
			expected: 0.0,
		},
		{
			name:     "ExternalBuffersAdded",
			metrics:  Metrics{Gets: 50, Puts: 60},
			expected: 120.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			returnRate := tc.metrics.ReturnRate()
			assert.InDelta(t, tc.expected, returnRate, 0.001)

			// Verify deprecated Efficiency() returns same value
			efficiency := tc.metrics.Efficiency()
			assert.InDelta(t, returnRate, efficiency, 0.001)
		})
	}
}

// TestConcurrentAccess tests thread-safe concurrent access
func TestConcurrentAccess(t *testing.T) {
	bp := NewBufferPool()
	bp.ResetStats()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Vary buffer sizes
				size := (id*100 + j*10) % 70000

				buf := bp.GetBuffer(size)
				buf.WriteString("concurrent test data")

				// Simulate some work
				_ = buf.String()

				bp.PutBuffer(buf)
			}
		}(i)
	}

	wg.Wait()

	stats := bp.GetStats()
	totalGets := stats.SmallPool.Gets + stats.MediumPool.Gets +
		stats.LargePool.Gets + stats.Oversized

	assert.Equal(t, int64(numGoroutines*numOperations), totalGets)
}

// TestEstimateBufferSize tests buffer size estimation
func TestEstimateBufferSize(t *testing.T) {
	testCases := []struct {
		operation string
		dataSize  int
		expected  int
	}{
		{"json_marshal", 100, SmallBufferThreshold},
		{"json_marshal", 1000, 2000},
		{"string_concat", 100, SmallBufferThreshold},
		{"template_transform", 1000, MediumBufferThreshold},
		{"file_content", 5000, MediumBufferThreshold},
		{"git_diff", 10000, 65536},
		{"unknown", 1000, MediumBufferThreshold},
	}

	for _, tc := range testCases {
		t.Run(tc.operation, func(t *testing.T) {
			size := EstimateBufferSize(tc.operation, tc.dataSize)
			assert.Equal(t, tc.expected, size)
		})
	}
}

// TestBufferCapacityPreservation tests that buffer capacity is maintained
func TestBufferCapacityPreservation(t *testing.T) {
	bp := NewBufferPool()

	// Get a buffer and check its capacity
	buf := bp.GetBuffer(SmallBufferThreshold)
	originalCap := buf.Cap()

	// Use the buffer
	for i := 0; i < 10; i++ {
		buf.WriteString("test data ")
	}

	// Return it to pool
	bp.PutBuffer(buf)

	// Get another buffer
	buf2 := bp.GetBuffer(SmallBufferThreshold)

	// Should have same capacity (reused buffer)
	assert.Equal(t, originalCap, buf2.Cap())
}

// TestPoolCategorizationByCapacity tests that buffers are returned to correct pool
func TestPoolCategorizationByCapacity(t *testing.T) {
	bp := NewBufferPool()
	bp.ResetStats()

	// Create a buffer with specific capacity
	buf := bytes.NewBuffer(make([]byte, 0, MediumBufferThreshold))

	// Return it to pool
	bp.PutBuffer(buf)

	// Should go to medium pool based on capacity
	stats := bp.GetStats()
	assert.Equal(t, int64(1), stats.MediumPool.Puts)
	assert.Equal(t, int64(0), stats.SmallPool.Puts)
	assert.Equal(t, int64(0), stats.LargePool.Puts)
}

// TestMaxInt tests the maxInt helper function
func TestMaxInt(t *testing.T) {
	testCases := []struct {
		a, b, expected int
	}{
		{1, 2, 2},
		{5, 3, 5},
		{-1, -2, -1},
		{0, 0, 0},
	}

	for _, tc := range testCases {
		result := maxInt(tc.a, tc.b)
		assert.Equal(t, tc.expected, result)
	}
}

// TestBufferSizeConstants tests that size constants are properly ordered
func TestBufferSizeConstants(t *testing.T) {
	assert.Less(t, SmallBufferThreshold, MediumBufferThreshold)
	assert.Less(t, MediumBufferThreshold, LargeBufferThreshold)
	assert.Less(t, LargeBufferThreshold, MaxPoolableSize)

	// Verify reasonable sizes
	assert.Equal(t, 1024, SmallBufferThreshold)  // 1KB
	assert.Equal(t, 8192, MediumBufferThreshold) // 8KB
	assert.Equal(t, 65536, LargeBufferThreshold) // 64KB
	assert.Equal(t, 131072, MaxPoolableSize)     // 128KB
}

// TestGetBufferNegativeSize tests that negative sizes don't cause panics
func TestGetBufferNegativeSize(t *testing.T) {
	bp := NewBufferPool()

	testCases := []struct {
		name string
		size int
	}{
		{"NegativeOne", -1},
		{"LargeNegative", -1000000},
		{"MinInt", -1 << 31},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				buf := bp.GetBuffer(tc.size)
				require.NotNil(t, buf)
				assert.GreaterOrEqual(t, buf.Cap(), 0)
				bp.PutBuffer(buf)
			})
		})
	}
}

// TestGetBufferNegativeSizeDefaultPool tests negative sizes with default pool
func TestGetBufferNegativeSizeDefaultPool(t *testing.T) {
	require.NotPanics(t, func() {
		buf := GetBuffer(-100)
		require.NotNil(t, buf)
		PutBuffer(buf)
	})
}

// TestEstimateBufferSizeEdgeCases tests overflow protection and edge cases
func TestEstimateBufferSizeEdgeCases(t *testing.T) {
	testCases := []struct {
		name      string
		operation string
		dataSize  int
	}{
		{"LargeDataSize", "git_diff", 1 << 30},
		{"VeryLargeDataSize", "json_marshal", 1 << 60},
		{"NegativeDataSize", "template_transform", -1},
		{"LargeNegativeDataSize", "file_content", -1 << 30},
		{"MaxInt", "string_concat", 1<<63 - 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				size := EstimateBufferSize(tc.operation, tc.dataSize)
				// Result should always be positive
				assert.Positive(t, size)
				// Result should be within reasonable bounds
				assert.LessOrEqual(t, size, MaxPoolableSize*5)
			})
		})
	}
}

// TestDefensiveTypeAssertion tests that the pool handles type assertion safely
func TestDefensiveTypeAssertion(t *testing.T) {
	bp := NewBufferPool()

	// Multiple rapid get/put cycles to stress test type assertions
	for i := 0; i < 100; i++ {
		sizes := []int{100, 5000, 50000, 100000}
		for _, size := range sizes {
			buf := bp.GetBuffer(size)
			require.NotNil(t, buf)
			buf.WriteString("test data")
			bp.PutBuffer(buf)
		}
	}

	stats := bp.GetStats()
	totalGets := stats.SmallPool.Gets + stats.MediumPool.Gets + stats.LargePool.Gets + stats.Oversized
	assert.Equal(t, int64(400), totalGets)
}

// TestBufferPoolImbalance documents behavior when buffers grow beyond original tier
func TestBufferPoolImbalance(t *testing.T) {
	bp := NewBufferPool()
	bp.ResetStats()

	// Get a small buffer
	buf := bp.GetBuffer(100)
	assert.LessOrEqual(t, buf.Cap(), SmallBufferThreshold)

	// Write enough data to force buffer growth beyond small threshold
	largeData := make([]byte, MediumBufferThreshold+1)
	buf.Write(largeData)

	// Buffer has grown - capacity is now larger than small threshold
	assert.Greater(t, buf.Cap(), SmallBufferThreshold)

	// Return buffer - it will go to a larger pool based on capacity
	bp.PutBuffer(buf)

	// Verify: buffer was gotten from small pool but returned to medium/large
	stats := bp.GetStats()
	assert.Equal(t, int64(1), stats.SmallPool.Gets)
	assert.Equal(t, int64(0), stats.SmallPool.Puts) // Not returned to small pool
	// It should have gone to medium or large pool based on new capacity
	assert.True(t, stats.MediumPool.Puts > 0 || stats.LargePool.Puts > 0,
		"Grown buffer should be returned to larger pool")
}
