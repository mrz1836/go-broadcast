package benchmark

import (
	"fmt"
	"os"
	"path/filepath"
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
	result.AllocsPerOp = int64(memAfter.Mallocs - memBefore.Mallocs)      //nolint:gosec // allocation counts are bounded in practice
	result.BytesPerOp = int64(memAfter.TotalAlloc - memBefore.TotalAlloc) //nolint:gosec // allocation bytes are bounded in practice
	result.MemoryUsed = int64(memAfter.Alloc)                             //nolint:gosec // memory values fit in int64 (max ~9 exabytes)

	return result
}

// RunWithMemoryTracking executes a benchmark with detailed memory tracking.
// The name parameter is reserved for future use (e.g., sub-benchmark labeling).
func RunWithMemoryTracking(b *testing.B, name string, fn func()) {
	b.Helper()
	_ = name // Reserved for future sub-benchmark labeling

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
	// Guard against division by zero when b.N is 0
	if b.N > 0 {
		b.ReportMetric(float64(allocsAfter-allocsBefore)/float64(b.N), "allocs/op")
		b.ReportMetric(float64(bytesAfter-bytesBefore)/float64(b.N), "bytes/op")
	}
}

// CreateTempRepo creates a temporary git repository for testing
func CreateTempRepo(b *testing.B) string {
	b.Helper()
	return b.TempDir()
}

// Size represents a size configuration for benchmarks
type Size struct {
	Name      string
	FileCount int
	FileSize  int
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

// StandardSizes returns consistent size configurations for benchmarks
func StandardSizes() []Size {
	return []Size{
		{Name: "Small", FileCount: 10, FileSize: 1024},
		{Name: "Medium", FileCount: 100, FileSize: 10240},
		{Name: "Large", FileCount: 1000, FileSize: 102400},
	}
}

// SetupBenchmarkFiles creates files for benchmark testing
func SetupBenchmarkFiles(b *testing.B, dir string, count int) []string {
	b.Helper()

	// Validate inputs
	if dir == "" {
		b.Fatal("directory path cannot be empty")
	}
	if count < 0 {
		count = 0
	}

	files := make([]string, count)
	for i := 0; i < count; i++ {
		fileName := fmt.Sprintf("bench_file_%d.txt", i)
		filePath := filepath.Join(dir, fileName)
		content := fmt.Sprintf("Benchmark test content %d", i)

		err := os.WriteFile(filePath, []byte(content), 0o600)
		if err != nil {
			b.Fatalf("failed to create benchmark file %s: %v", filePath, err)
		}
		files[i] = filePath
	}
	return files
}

// WithMemoryTracking runs benchmark with memory allocation tracking
func WithMemoryTracking(b *testing.B, fn func()) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		fn()
	}

	b.StopTimer()
}

// SetupBenchmarkRepo creates a temporary repository for benchmark testing
func SetupBenchmarkRepo(b *testing.B) string {
	b.Helper()
	return b.TempDir()
}
