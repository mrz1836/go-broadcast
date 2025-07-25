package benchmark

import (
	"runtime"
	"testing"
	"time"
)

// Result captures comprehensive performance metrics
type Result struct {
	Name        string
	Operations  int64
	NsPerOp     int64
	AllocsPerOp int64
	BytesPerOp  int64
	MemoryUsed  int64
	StartTime   time.Time
	EndTime     time.Time
}

// MemoryStats captures memory usage before and after operation
type MemoryStats struct {
	Before runtime.MemStats
	After  runtime.MemStats
}

// CaptureMemoryStats runs a function and captures memory statistics
func CaptureMemoryStats(fn func()) MemoryStats {
	var stats MemoryStats
	runtime.GC()
	runtime.ReadMemStats(&stats.Before)

	fn()

	runtime.GC()
	runtime.ReadMemStats(&stats.After)
	return stats
}

// ReportResults generates a formatted benchmark report
func ReportResults(b *testing.B, results []Result) {
	b.Helper()
	for _, r := range results {
		b.Logf("Benchmark: %s", r.Name)
		b.Logf("  Operations: %d", r.Operations)
		b.Logf("  ns/op: %d", r.NsPerOp)
		b.Logf("  allocs/op: %d", r.AllocsPerOp)
		b.Logf("  bytes/op: %d", r.BytesPerOp)
	}
}

// GenerateTestData creates test data of various sizes
func GenerateTestData(size string) []byte {
	sizes := map[string]int{
		"small":  1024,             // 1KB
		"medium": 1024 * 100,       // 100KB
		"large":  1024 * 1024,      // 1MB
		"xlarge": 1024 * 1024 * 10, // 10MB
	}

	if bytes, ok := sizes[size]; ok {
		data := make([]byte, bytes)
		for i := range data {
			data[i] = byte(i % 256)
		}
		return data
	}
	return nil
}

// MeasureOperation measures an operation and returns timing metrics
func MeasureOperation(name string, fn func()) Result {
	result := Result{
		Name:      name,
		StartTime: time.Now(),
	}

	// Capture memory stats
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Run the operation
	fn()

	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	result.EndTime = time.Now()
	result.Operations = 1
	result.NsPerOp = result.EndTime.Sub(result.StartTime).Nanoseconds()
	result.AllocsPerOp = int64(memAfter.Mallocs - memBefore.Mallocs)      //nolint:gosec // Memory stats unlikely to overflow in tests
	result.BytesPerOp = int64(memAfter.TotalAlloc - memBefore.TotalAlloc) //nolint:gosec // Memory stats unlikely to overflow in tests
	result.MemoryUsed = int64(memAfter.Alloc)                             //nolint:gosec // Memory stats unlikely to overflow in tests

	return result
}

// RunWithMemoryTracking executes a benchmark with detailed memory tracking
func RunWithMemoryTracking(b *testing.B, _ string, fn func()) {
	b.Helper()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	allocsBefore := memStats.Mallocs
	bytesBefore := memStats.TotalAlloc

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fn()
	}
	b.StopTimer()

	runtime.ReadMemStats(&memStats)
	allocsAfter := memStats.Mallocs
	bytesAfter := memStats.TotalAlloc

	b.ReportAllocs()
	b.ReportMetric(float64(allocsAfter-allocsBefore)/float64(b.N), "allocs/op")
	b.ReportMetric(float64(bytesAfter-bytesBefore)/float64(b.N), "bytes/op")
}

// CreateTempRepo creates a temporary git repository for testing
func CreateTempRepo(b *testing.B) string {
	b.Helper()
	return b.TempDir()
}

// Sizes returns standard size configurations for benchmarks
func Sizes() []struct {
	Name string
	Size string
} {
	return []struct {
		Name string
		Size string
	}{
		{"Small", "small"},
		{"Medium", "medium"},
		{"Large", "large"},
		{"XLarge", "xlarge"},
	}
}
