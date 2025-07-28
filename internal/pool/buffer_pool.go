// Package pool provides buffer pooling and resource management utilities for efficient memory usage.
package pool

import (
	"bytes"
	"sync"
	"sync/atomic"
)

// BufferPool manages multiple tiers of buffer pools with statistics
type BufferPool struct {
	smallBufferPool  *sync.Pool
	mediumBufferPool *sync.Pool
	largeBufferPool  *sync.Pool

	// Pool statistics for monitoring and optimization
	stats struct {
		smallGets  int64
		smallPuts  int64
		mediumGets int64
		mediumPuts int64
		largeGets  int64
		largePuts  int64
		oversized  int64 // Buffers too large for pooling
		resets     int64 // Buffer resets performed
	}
}

// NewBufferPool creates a new buffer pool instance
func NewBufferPool() *BufferPool {
	return &BufferPool{
		smallBufferPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 1024)) // 1KB capacity
			},
		},
		mediumBufferPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 8192)) // 8KB capacity
			},
		},
		largeBufferPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 65536)) // 64KB capacity
			},
		},
	}
}

var (
	defaultBufferPool *BufferPool //nolint:gochecknoglobals // Package-level singleton pattern
	defaultPoolOnce   sync.Once   //nolint:gochecknoglobals // Package-level singleton pattern
)

// getDefaultPool returns the default buffer pool, creating it if necessary
func getDefaultPool() *BufferPool {
	defaultPoolOnce.Do(func() {
		defaultBufferPool = NewBufferPool()
	})
	return defaultBufferPool
}

// Size thresholds for pool selection
const (
	SmallBufferThreshold  = 1024   // 1KB
	MediumBufferThreshold = 8192   // 8KB
	LargeBufferThreshold  = 65536  // 64KB
	MaxPoolableSize       = 131072 // 128KB - don't pool buffers larger than this
)

// GetBuffer returns a buffer from the appropriate pool based on required size.
//
// The buffer selection is optimized for common usage patterns:
// - Small buffers (≤1KB): For short strings, config values, small JSON
// - Medium buffers (≤8KB): For file paths, moderate content, API responses
// - Large buffers (≤64KB): For file content, large transformations, diffs
//
// Parameters:
// - size: Minimum required buffer capacity in bytes
//
// Returns:
// - *bytes.Buffer from the appropriate pool, ready for use
//
// Performance Notes:
// - Buffers are pre-allocated with appropriate capacity to minimize reallocations
// - Pool selection is based on required size, not buffer content
// - Returned buffers may have larger capacity than requested (which is beneficial)
// GetBuffer returns a buffer from the appropriate pool based on required size.
func (bp *BufferPool) GetBuffer(size int) *bytes.Buffer {
	switch {
	case size <= SmallBufferThreshold:
		atomic.AddInt64(&bp.stats.smallGets, 1)
		return bp.smallBufferPool.Get().(*bytes.Buffer)
	case size <= MediumBufferThreshold:
		atomic.AddInt64(&bp.stats.mediumGets, 1)
		return bp.mediumBufferPool.Get().(*bytes.Buffer)
	case size <= LargeBufferThreshold:
		atomic.AddInt64(&bp.stats.largeGets, 1)
		return bp.largeBufferPool.Get().(*bytes.Buffer)
	default:
		// For very large requirements, create a new buffer without pooling
		// This prevents memory waste from pooling oversized buffers
		atomic.AddInt64(&bp.stats.oversized, 1)
		return bytes.NewBuffer(make([]byte, 0, size))
	}
}

// GetBuffer returns a buffer from the default pool based on required size.
//
// The buffer selection is optimized for common usage patterns:
// - Small buffers (≤1KB): For short strings, config values, small JSON
// - Medium buffers (≤8KB): For file paths, moderate content, API responses
// - Large buffers (≤64KB): For file content, large transformations, diffs
//
// Parameters:
// - size: Minimum required buffer capacity in bytes
//
// Returns:
// - *bytes.Buffer from the appropriate pool, ready for use
//
// Performance Notes:
// - Buffers are pre-allocated with appropriate capacity to minimize reallocations
// - Pool selection is based on required size, not buffer content
// - Returned buffers may have larger capacity than requested (which is beneficial)
func GetBuffer(size int) *bytes.Buffer {
	return getDefaultPool().GetBuffer(size)
}

// PutBuffer returns a buffer to the appropriate pool after use.
//
// The buffer is cleaned and reset before pooling to ensure it's ready
// for the next use. Pool selection is based on the buffer's actual
// capacity to ensure proper categorization.
//
// Parameters:
// - buf: Buffer to return to the pool (can be nil)
//
// Behavior:
// - Nil buffers are safely ignored
// - Buffers are reset to empty state before pooling
// - Oversized buffers are not pooled to prevent memory waste
// - Pool selection is based on buffer capacity, not original request size
// PutBuffer returns a buffer to the appropriate pool after use.
func (bp *BufferPool) PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}

	// Reset the buffer to empty state for reuse
	buf.Reset()
	atomic.AddInt64(&bp.stats.resets, 1)

	capacity := buf.Cap()

	// Return buffer to appropriate pool based on its capacity
	switch {
	case capacity <= SmallBufferThreshold:
		atomic.AddInt64(&bp.stats.smallPuts, 1)
		bp.smallBufferPool.Put(buf)
	case capacity <= MediumBufferThreshold:
		atomic.AddInt64(&bp.stats.mediumPuts, 1)
		bp.mediumBufferPool.Put(buf)
	case capacity <= LargeBufferThreshold:
		atomic.AddInt64(&bp.stats.largePuts, 1)
		bp.largeBufferPool.Put(buf)
	case capacity <= MaxPoolableSize:
		// For buffers between large threshold and max poolable size,
		// put them in the large pool to avoid waste
		atomic.AddInt64(&bp.stats.largePuts, 1)
		bp.largeBufferPool.Put(buf)
	default:
		// Don't pool very large buffers - let them be garbage collected
		// This prevents the pool from growing unbounded with oversized buffers
		atomic.AddInt64(&bp.stats.oversized, 1)
	}
}

// PutBuffer returns a buffer to the appropriate pool after use.
//
// The buffer is cleaned and reset before pooling to ensure it's ready
// for the next use. Pool selection is based on the buffer's actual
// capacity to ensure proper categorization.
//
// Parameters:
// - buf: Buffer to return to the pool (can be nil)
//
// Behavior:
// - Nil buffers are safely ignored
// - Buffers are reset to empty state before pooling
// - Oversized buffers are not pooled to prevent memory waste
// - Pool selection is based on buffer capacity, not original request size
func PutBuffer(buf *bytes.Buffer) {
	getDefaultPool().PutBuffer(buf)
}

// WithBuffer executes a function with a pooled buffer and ensures cleanup.
//
// This function provides automatic buffer lifecycle management using
// the defer pattern. The buffer is automatically returned to the pool
// after the function completes, even if an error occurs.
//
// Parameters:
// - size: Minimum required buffer capacity in bytes
// - fn: Function to execute with the buffer
//
// Returns:
// - Error returned by the provided function
//
// Usage Example:
//
//	err := pool.WithBuffer(1024, func(buf *bytes.Buffer) error {
//	    buf.WriteString("Hello, ")
//	    buf.WriteString("World!")
//	    result := buf.String()
//	    return processResult(result)
//	})
func WithBuffer(size int, fn func(*bytes.Buffer) error) error {
	buf := GetBuffer(size)
	defer PutBuffer(buf)
	return fn(buf)
}

// WithBufferResult executes a function with a pooled buffer and returns a result.
//
// This function is similar to WithBuffer but allows returning a result value
// along with an error. This is useful for operations that need to extract
// data from the buffer before it's returned to the pool.
//
// Parameters:
// - size: Minimum required buffer capacity in bytes
// - fn: Function to execute with the buffer, returning a result and error
//
// Returns:
// - Result of type T returned by the provided function
// - Error returned by the provided function
//
// Usage Example:
//
//	result, err := pool.WithBufferResult[string](1024, func(buf *bytes.Buffer) (string, error) {
//	    buf.WriteString("Processing data...")
//	    return buf.String(), nil
//	})
func WithBufferResult[T any](size int, fn func(*bytes.Buffer) (T, error)) (T, error) {
	buf := GetBuffer(size)
	defer PutBuffer(buf)
	return fn(buf)
}

// GetStats returns current buffer pool statistics.
//
// Returns:
// - Stats structure containing detailed usage metrics
//
// Usage:
// This function is useful for monitoring pool effectiveness, detecting
// memory usage patterns, and optimizing pool configurations.
// GetStats returns current buffer pool statistics.
func (bp *BufferPool) GetStats() Stats {
	return Stats{
		SmallPool: Metrics{
			Gets: atomic.LoadInt64(&bp.stats.smallGets),
			Puts: atomic.LoadInt64(&bp.stats.smallPuts),
		},
		MediumPool: Metrics{
			Gets: atomic.LoadInt64(&bp.stats.mediumGets),
			Puts: atomic.LoadInt64(&bp.stats.mediumPuts),
		},
		LargePool: Metrics{
			Gets: atomic.LoadInt64(&bp.stats.largeGets),
			Puts: atomic.LoadInt64(&bp.stats.largePuts),
		},
		Oversized: atomic.LoadInt64(&bp.stats.oversized),
		Resets:    atomic.LoadInt64(&bp.stats.resets),
	}
}

// GetStats returns current buffer pool statistics.
//
// Returns:
// - Stats structure containing detailed usage metrics
//
// Usage:
// This function is useful for monitoring pool effectiveness, detecting
// memory usage patterns, and optimizing pool configurations.
func GetStats() Stats {
	return getDefaultPool().GetStats()
}

// Stats contains buffer pool usage statistics
type Stats struct {
	SmallPool  Metrics `json:"small_pool"`
	MediumPool Metrics `json:"medium_pool"`
	LargePool  Metrics `json:"large_pool"`
	Oversized  int64   `json:"oversized"` // Operations with oversized buffers
	Resets     int64   `json:"resets"`    // Buffer resets performed
}

// Metrics contains metrics for an individual pool
type Metrics struct {
	Gets int64 `json:"gets"` // Number of buffers retrieved from pool
	Puts int64 `json:"puts"` // Number of buffers returned to pool
}

// Efficiency calculates the pool efficiency as a percentage
func (pm Metrics) Efficiency() float64 {
	if pm.Gets == 0 {
		return 0
	}
	return float64(pm.Puts) / float64(pm.Gets) * 100
}

// ResetStats resets all pool statistics to zero.
//
// This function is primarily useful for testing or when fresh
// statistics are needed for monitoring purposes.
// ResetStats resets all pool statistics to zero.
func (bp *BufferPool) ResetStats() {
	atomic.StoreInt64(&bp.stats.smallGets, 0)
	atomic.StoreInt64(&bp.stats.smallPuts, 0)
	atomic.StoreInt64(&bp.stats.mediumGets, 0)
	atomic.StoreInt64(&bp.stats.mediumPuts, 0)
	atomic.StoreInt64(&bp.stats.largeGets, 0)
	atomic.StoreInt64(&bp.stats.largePuts, 0)
	atomic.StoreInt64(&bp.stats.oversized, 0)
	atomic.StoreInt64(&bp.stats.resets, 0)
}

// ResetStats resets all pool statistics to zero.
//
// This function is primarily useful for testing or when fresh
// statistics are needed for monitoring purposes.
func ResetStats() {
	getDefaultPool().ResetStats()
}

// EstimateBufferSize provides size estimation for common operations.
//
// This function helps developers choose appropriate buffer sizes
// for different types of operations, improving pool utilization.
//
// Parameters:
// - operation: Type of operation being performed
// - dataSize: Size of input data (if applicable)
//
// Returns:
// - Estimated buffer size in bytes
func EstimateBufferSize(operation string, dataSize int) int {
	switch operation {
	case "json_marshal":
		// JSON typically expands by 20-50% due to quotes and structure
		return maxInt(dataSize*2, SmallBufferThreshold)
	case "string_concat":
		// String concatenation - use exact size plus some padding
		return maxInt(dataSize+256, SmallBufferThreshold)
	case "template_transform":
		// Template transformation can expand significantly
		return maxInt(dataSize*3, MediumBufferThreshold)
	case "file_content":
		// File content processing - use file size as baseline
		return maxInt(dataSize+1024, MediumBufferThreshold)
	case "git_diff":
		// Git diffs can be much larger than original content
		return maxInt(dataSize*5, LargeBufferThreshold)
	default:
		// Default to medium buffer for unknown operations
		return MediumBufferThreshold
	}
}

// Helper function for max calculation
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
