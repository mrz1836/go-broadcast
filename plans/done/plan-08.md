# Performance and Benchmarking Implementation Plan for go-broadcast

## Executive Summary

This document outlines a comprehensive performance optimization plan for the go-broadcast codebase, restructured into actionable phases. Each phase represents a focused work session with specific deliverables, code examples, and validation steps. The goal is to benchmark current performance, identify bottlenecks, and implement targeted optimizations that improve efficiency without compromising functionality.

## Objectives

1. **Performance Baseline**: Establish comprehensive benchmarks for all critical operations
2. **Memory Efficiency**: Reduce memory allocations and improve GC performance
3. **Concurrency Optimization**: Maximize throughput for parallel operations
4. **I/O Performance**: Optimize file and network operations
5. **Continuous Monitoring**: Integrate performance testing into CI/CD pipeline

## Technical Approach

### Benchmarking Framework
- Use Go's built-in benchmarking with `testing.B`
- Create realistic test scenarios with varying data sizes
- Implement memory and allocation tracking
- Generate comparative reports for optimization validation

### Optimization Strategy
1. Measure first - establish baselines
2. Identify bottlenecks through profiling
3. Implement targeted optimizations
4. Validate improvements with benchmarks
5. Document changes and best practices

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass
- run: `go test -bench=. ./...` to validate performance

## Implementation Phases

### Phase 1: Benchmark Infrastructure & Missing Coverage (Days 1-2)

#### 1.1 Create Benchmark Helper Utilities
```
internal/
└── benchmark/
    ├── helpers.go
    ├── fixtures.go
    └── reporter.go
```

#### 1.2 Benchmark Helpers Implementation
```go
// internal/benchmark/helpers.go
package benchmark

import (
    "fmt"
    "runtime"
    "testing"
    "time"
)

// BenchmarkResult captures comprehensive performance metrics
type BenchmarkResult struct {
    Name         string
    Operations   int64
    NsPerOp      int64
    AllocsPerOp  int64
    BytesPerOp   int64
    MemoryUsed   int64
    StartTime    time.Time
    EndTime      time.Time
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
func ReportResults(b *testing.B, results []BenchmarkResult) {
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
        "small":  1024,           // 1KB
        "medium": 1024 * 100,     // 100KB
        "large":  1024 * 1024,    // 1MB
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
```

#### 1.3 Git Package Benchmarks
```go
// internal/git/benchmark_test.go
package git

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    
    "github.com/yourusername/go-broadcast/internal/benchmark"
)

func BenchmarkGitCommand_Simple(b *testing.B) {
    client := &gitClient{runner: &realCommandRunner{}}
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = client.Version(ctx)
    }
}

func BenchmarkGitCommand_WithOutput(b *testing.B) {
    client := &gitClient{runner: &realCommandRunner{}}
    ctx := context.Background()
    tmpDir := b.TempDir()
    
    // Initialize a git repo
    _ = client.Init(ctx, tmpDir)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = client.Status(ctx, tmpDir)
    }
}

func BenchmarkClone_Sizes(b *testing.B) {
    sizes := []struct {
        name string
        repo string
    }{
        {"Small", "https://github.com/octocat/Hello-World.git"},
        // Add more test repos of different sizes
    }
    
    for _, size := range sizes {
        b.Run(size.name, func(b *testing.B) {
            client := &gitClient{runner: &realCommandRunner{}}
            ctx := context.Background()
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                tmpDir := b.TempDir()
                _ = client.Clone(ctx, size.repo, tmpDir, "master", nil)
            }
        })
    }
}

func BenchmarkAdd_FileCount(b *testing.B) {
    counts := []int{1, 10, 100, 1000}
    
    for _, count := range counts {
        b.Run(fmt.Sprintf("Files_%d", count), func(b *testing.B) {
            client := &gitClient{runner: &realCommandRunner{}}
            ctx := context.Background()
            tmpDir := b.TempDir()
            
            // Initialize repo and create files
            _ = client.Init(ctx, tmpDir)
            for i := 0; i < count; i++ {
                file := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
                _ = os.WriteFile(file, []byte("test content"), 0644)
            }
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = client.Add(ctx, tmpDir, ".")
            }
        })
    }
}

func BenchmarkDiff_Sizes(b *testing.B) {
    sizes := []struct {
        name  string
        lines int
    }{
        {"Small", 10},
        {"Medium", 100},
        {"Large", 1000},
    }
    
    for _, size := range sizes {
        b.Run(size.name, func(b *testing.B) {
            client := &gitClient{runner: &realCommandRunner{}}
            ctx := context.Background()
            tmpDir := b.TempDir()
            
            // Setup repo with changes
            setupDiffTest(tmpDir, size.lines)
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _, _ = client.Diff(ctx, tmpDir, "HEAD", "")
            }
        })
    }
}
```

#### 1.4 GitHub Package Benchmarks
```go
// internal/gh/benchmark_test.go
package gh

import (
    "context"
    "encoding/json"
    "testing"
    
    "github.com/yourusername/go-broadcast/internal/benchmark"
)

func BenchmarkGHCommand_Simple(b *testing.B) {
    client := &Client{runner: &mockRunner{
        output: []byte(`{"status": "ok"}`),
    }}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = client.run("api", "/rate_limit")
    }
}

func BenchmarkParseJSON_Sizes(b *testing.B) {
    sizes := []struct {
        name string
        data string
    }{
        {"Small", generateJSON(10)},      // 10 items
        {"Medium", generateJSON(100)},    // 100 items  
        {"Large", generateJSON(1000)},    // 1000 items
    }
    
    for _, size := range sizes {
        b.Run(size.name, func(b *testing.B) {
            data := []byte(size.data)
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                var result []interface{}
                _ = json.Unmarshal(data, &result)
            }
        })
    }
}

func BenchmarkDecodeBase64_Sizes(b *testing.B) {
    sizes := []struct {
        name string
        size int
    }{
        {"Small", 1024},        // 1KB
        {"Medium", 1024 * 10},  // 10KB
        {"Large", 1024 * 100},  // 100KB
    }
    
    for _, size := range sizes {
        b.Run(size.name, func(b *testing.B) {
            content := generateBase64Content(size.size)
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = decodeContent(content)
            }
        })
    }
}

func BenchmarkConcurrentAPICalls(b *testing.B) {
    concurrencyLevels := []int{1, 5, 10, 20}
    
    for _, level := range concurrencyLevels {
        b.Run(fmt.Sprintf("Concurrent_%d", level), func(b *testing.B) {
            client := &Client{runner: &mockRunner{
                output: []byte(`{"status": "ok"}`),
            }}
            
            b.ResetTimer()
            b.RunParallel(func(pb *testing.PB) {
                for pb.Next() {
                    _, _ = client.run("api", "/repos/org/repo")
                }
            })
        })
    }
}
```

#### 1.5 Logging Package Benchmarks
```go
// internal/logging/benchmark_test.go
package logging

import (
    "strings"
    "testing"
    
    "github.com/yourusername/go-broadcast/internal/benchmark"
)

func BenchmarkRedaction_Scenarios(b *testing.B) {
    scenarios := []struct {
        name string
        text string
    }{
        {"NoSensitive", "This is a normal log message without any sensitive data"},
        {"WithToken", "Authorization: Bearer ghp_xxxxxxxxxxxxxxxxxxxx"},
        {"MultipleTokens", strings.Repeat("token: abc123 ", 100)},
        {"LargeText", string(benchmark.GenerateTestData("large"))},
    }
    
    for _, scenario := range scenarios {
        b.Run(scenario.name, func(b *testing.B) {
            redactor := NewRedactor()
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = redactor.Redact(scenario.text)
            }
        })
    }
}

func BenchmarkFormatting_Types(b *testing.B) {
    entry := LogEntry{
        Level:   "INFO",
        Message: "Test log message",
        Fields: map[string]interface{}{
            "user":     "test",
            "duration": 123.45,
            "count":    100,
        },
    }
    
    formatters := []struct {
        name      string
        formatter Formatter
    }{
        {"Text", NewTextFormatter()},
        {"JSON", NewJSONFormatter()},
        {"JSONMany", NewJSONFormatter()}, // Test with 20+ fields
    }
    
    for _, f := range formatters {
        b.Run(f.name, func(b *testing.B) {
            if f.name == "JSONMany" {
                // Add many fields
                for i := 0; i < 20; i++ {
                    entry.Fields[fmt.Sprintf("field%d", i)] = i
                }
            }
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = f.formatter.Format(entry)
            }
        })
    }
}

func BenchmarkConcurrentLogging(b *testing.B) {
    goroutines := []int{1, 10, 100}
    
    for _, count := range goroutines {
        b.Run(fmt.Sprintf("Goroutines_%d", count), func(b *testing.B) {
            logger := NewLogger()
            
            b.ResetTimer()
            b.RunParallel(func(pb *testing.PB) {
                for pb.Next() {
                    logger.Info("Concurrent log message")
                }
            })
        })
    }
}
```

#### Phase 1 Status Tracking
At the end of Phase 1, create `plans/plan-08-status.md` with:
- **Completed**: All benchmark infrastructure and missing package benchmarks
- **Baseline Metrics**: Document initial performance numbers
- **Challenges**: Any issues with benchmark setup or execution
- **Next Steps**: Identify top optimization candidates from baseline

### Phase 2: Performance Baseline & Quick Wins (Days 3-4)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass
- run: `go test -bench=. ./...` to validate performance

#### 2.1 Performance Baseline Documentation
```go
// internal/benchmark/baseline.go
package benchmark

import (
    "encoding/json"
    "fmt"
    "os"
    "time"
)

type BaselineReport struct {
    Timestamp   time.Time                    `json:"timestamp"`
    GoVersion   string                       `json:"go_version"`
    GOOS        string                       `json:"goos"`
    GOARCH      string                       `json:"goarch"`
    Benchmarks  map[string]BenchmarkMetrics  `json:"benchmarks"`
}

type BenchmarkMetrics struct {
    Name        string  `json:"name"`
    NsPerOp     int64   `json:"ns_per_op"`
    AllocsPerOp int64   `json:"allocs_per_op"`
    BytesPerOp  int64   `json:"bytes_per_op"`
    MBPerSec    float64 `json:"mb_per_sec,omitempty"`
}

func SaveBaseline(filename string, report BaselineReport) error {
    data, err := json.MarshalIndent(report, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(filename, data, 0644)
}

func CompareWithBaseline(current, baseline BaselineReport) string {
    var report strings.Builder
    report.WriteString("Performance Comparison Report\n")
    report.WriteString("=============================\n\n")
    
    for name, currentMetric := range current.Benchmarks {
        if baselineMetric, ok := baseline.Benchmarks[name]; ok {
            speedup := float64(baselineMetric.NsPerOp) / float64(currentMetric.NsPerOp)
            allocReduction := float64(baselineMetric.AllocsPerOp - currentMetric.AllocsPerOp) / float64(baselineMetric.AllocsPerOp) * 100
            
            report.WriteString(fmt.Sprintf("%s:\n", name))
            report.WriteString(fmt.Sprintf("  Speed: %.2fx %s\n", speedup, speedupEmoji(speedup)))
            report.WriteString(fmt.Sprintf("  Allocations: %.1f%% reduction\n", allocReduction))
            report.WriteString("\n")
        }
    }
    
    return report.String()
}

func speedupEmoji(speedup float64) string {
    switch {
    case speedup >= 2.0:
        return "🚀"
    case speedup >= 1.5:
        return "⚡"
    case speedup >= 1.1:
        return "✅"
    case speedup >= 0.9:
        return "➖"
    default:
        return "⚠️"
    }
}
```

#### 2.2 Regex Compilation Caching
```go
// internal/transform/regex_cache.go
package transform

import (
    "regexp"
    "sync"
)

var (
    regexCache = make(map[string]*regexp.Regexp)
    regexMu    sync.RWMutex
    
    // Pre-compile common patterns
    commonPatterns = []string{
        `github\.com/([^/]+/[^/]+)`,
        `sync/.*-(\d{8}-\d{6})-([a-f0-9]+)`,
        `\{\{([A-Z_]+)\}\}`,
    }
)

func init() {
    // Pre-compile common patterns at startup
    for _, pattern := range commonPatterns {
        if re, err := regexp.Compile(pattern); err == nil {
            regexCache[pattern] = re
        }
    }
}

// CompileRegex returns a compiled regex, using cache when possible
func CompileRegex(pattern string) (*regexp.Regexp, error) {
    // Fast path: read from cache
    regexMu.RLock()
    if re, ok := regexCache[pattern]; ok {
        regexMu.RUnlock()
        return re, nil
    }
    regexMu.RUnlock()
    
    // Slow path: compile and cache
    regexMu.Lock()
    defer regexMu.Unlock()
    
    // Double-check after acquiring write lock
    if re, ok := regexCache[pattern]; ok {
        return re, nil
    }
    
    re, err := regexp.Compile(pattern)
    if err != nil {
        return nil, err
    }
    
    regexCache[pattern] = re
    return re, nil
}

// MustCompileRegex panics if the regex doesn't compile
func MustCompileRegex(pattern string) *regexp.Regexp {
    re, err := CompileRegex(pattern)
    if err != nil {
        panic(err)
    }
    return re
}
```

#### 2.3 Buffer Pool Implementation
```go
// internal/pool/buffer_pool.go
package pool

import (
    "bytes"
    "sync"
)

var (
    smallBufferPool = &sync.Pool{
        New: func() interface{} {
            return bytes.NewBuffer(make([]byte, 0, 1024))
        },
    }
    
    mediumBufferPool = &sync.Pool{
        New: func() interface{} {
            return bytes.NewBuffer(make([]byte, 0, 8192))
        },
    }
    
    largeBufferPool = &sync.Pool{
        New: func() interface{} {
            return bytes.NewBuffer(make([]byte, 0, 65536))
        },
    }
)

// GetBuffer returns a buffer from the appropriate pool based on required size
func GetBuffer(size int) *bytes.Buffer {
    switch {
    case size <= 1024:
        return smallBufferPool.Get().(*bytes.Buffer)
    case size <= 8192:
        return mediumBufferPool.Get().(*bytes.Buffer)
    default:
        return largeBufferPool.Get().(*bytes.Buffer)
    }
}

// PutBuffer returns a buffer to the appropriate pool
func PutBuffer(buf *bytes.Buffer) {
    if buf == nil {
        return
    }
    
    buf.Reset()
    capacity := buf.Cap()
    
    switch {
    case capacity <= 1024:
        smallBufferPool.Put(buf)
    case capacity <= 8192:
        mediumBufferPool.Put(buf)
    case capacity <= 65536:
        largeBufferPool.Put(buf)
    // Don't pool very large buffers
    }
}

// WithBuffer executes a function with a pooled buffer
func WithBuffer(size int, fn func(*bytes.Buffer) error) error {
    buf := GetBuffer(size)
    defer PutBuffer(buf)
    return fn(buf)
}
```

#### 2.4 String Builder Adoption
```go
// internal/transform/string_builder.go
package transform

import (
    "strings"
    "github.com/yourusername/go-broadcast/internal/pool"
)

// Before: String concatenation with +
func oldBuildPath(parts ...string) string {
    result := ""
    for i, part := range parts {
        if i > 0 {
            result += "/"
        }
        result += part
    }
    return result
}

// After: Using strings.Builder
func BuildPath(parts ...string) string {
    if len(parts) == 0 {
        return ""
    }
    
    // Estimate size to minimize allocations
    size := len(parts) - 1 // separators
    for _, part := range parts {
        size += len(part)
    }
    
    var sb strings.Builder
    sb.Grow(size)
    
    for i, part := range parts {
        if i > 0 {
            sb.WriteByte('/')
        }
        sb.WriteString(part)
    }
    
    return sb.String()
}

// For very large string operations, use pooled buffers
func BuildLargeString(parts []string) string {
    totalSize := 0
    for _, part := range parts {
        totalSize += len(part)
    }
    
    return pool.WithBuffer(totalSize, func(buf *bytes.Buffer) error {
        for _, part := range parts {
            buf.WriteString(part)
        }
        return nil
    })
}
```

#### 2.5 Benchmark Quick Wins
```go
// internal/transform/optimized_test.go
package transform

import (
    "testing"
)

func BenchmarkRegexCache(b *testing.B) {
    pattern := `github\.com/([^/]+/[^/]+)`
    input := "https://github.com/user/repo/blob/main/README.md"
    
    b.Run("Without_Cache", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            re, _ := regexp.Compile(pattern)
            _ = re.FindStringSubmatch(input)
        }
    })
    
    b.Run("With_Cache", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            re, _ := CompileRegex(pattern)
            _ = re.FindStringSubmatch(input)
        }
    })
}

func BenchmarkStringBuilding(b *testing.B) {
    parts := []string{"path", "to", "some", "deeply", "nested", "file"}
    
    b.Run("Concatenation", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = oldBuildPath(parts...)
        }
    })
    
    b.Run("StringBuilder", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _ = BuildPath(parts...)
        }
    })
}

func BenchmarkBufferPool(b *testing.B) {
    data := []byte("Some test data to write multiple times")
    
    b.Run("New_Buffer", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            buf := bytes.NewBuffer(nil)
            for j := 0; j < 100; j++ {
                buf.Write(data)
            }
            _ = buf.String()
        }
    })
    
    b.Run("Pooled_Buffer", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            pool.WithBuffer(len(data)*100, func(buf *bytes.Buffer) error {
                for j := 0; j < 100; j++ {
                    buf.Write(data)
                }
                _ = buf.String()
                return nil
            })
        }
    })
}
```

#### Phase 2 Status Tracking
At the end of Phase 2, update `plans/plan-08-status.md` with:
- **Completed**: Baseline documentation, regex caching, buffer pools, string builders
- **Performance Gains**: Document % improvements for each optimization
- **Challenges**: Any compatibility issues or unexpected behaviors
- **Next Steps**: Prepare for I/O and memory optimizations

### Phase 3: I/O and Memory Optimization (Days 5-6)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass
- run: `go test -bench=. ./...` to validate performance

#### 3.1 Streaming File Processor
```go
// internal/io/streaming.go
package io

import (
    "bufio"
    "io"
    "os"
    
    "github.com/yourusername/go-broadcast/internal/pool"
)

// StreamProcessor processes files in chunks without loading entire content
type StreamProcessor struct {
    ChunkSize int
}

func NewStreamProcessor() *StreamProcessor {
    return &StreamProcessor{
        ChunkSize: 64 * 1024, // 64KB chunks
    }
}

// ProcessFile streams a file through a transformation function
func (sp *StreamProcessor) ProcessFile(inputPath, outputPath string, transform func([]byte) ([]byte, error)) error {
    input, err := os.Open(inputPath)
    if err != nil {
        return err
    }
    defer input.Close()
    
    output, err := os.Create(outputPath)
    if err != nil {
        return err
    }
    defer output.Close()
    
    reader := bufio.NewReaderSize(input, sp.ChunkSize)
    writer := bufio.NewWriterSize(output, sp.ChunkSize)
    defer writer.Flush()
    
    buf := make([]byte, sp.ChunkSize)
    
    for {
        n, err := reader.Read(buf)
        if n > 0 {
            transformed, transformErr := transform(buf[:n])
            if transformErr != nil {
                return transformErr
            }
            
            if _, writeErr := writer.Write(transformed); writeErr != nil {
                return writeErr
            }
        }
        
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }
    
    return nil
}

// ProcessLargeJSON handles large JSON files without loading entire content
func (sp *StreamProcessor) ProcessLargeJSON(inputPath string, handler func(interface{}) error) error {
    file, err := os.Open(inputPath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    decoder := json.NewDecoder(bufio.NewReader(file))
    
    // Read opening bracket
    if _, err := decoder.Token(); err != nil {
        return err
    }
    
    // Process array elements one by one
    for decoder.More() {
        var item interface{}
        if err := decoder.Decode(&item); err != nil {
            return err
        }
        
        if err := handler(item); err != nil {
            return err
        }
    }
    
    // Read closing bracket
    if _, err := decoder.Token(); err != nil {
        return err
    }
    
    return nil
}
```

#### 3.2 Memory-Efficient Operations
```go
// internal/memory/efficient.go
package memory

import (
    "sync"
)

// StringIntern provides string interning for repeated values
type StringIntern struct {
    mu     sync.RWMutex
    values map[string]string
}

func NewStringIntern() *StringIntern {
    return &StringIntern{
        values: make(map[string]string),
    }
}

// Intern returns a canonical instance of the string
func (si *StringIntern) Intern(s string) string {
    si.mu.RLock()
    if interned, ok := si.values[s]; ok {
        si.mu.RUnlock()
        return interned
    }
    si.mu.RUnlock()
    
    si.mu.Lock()
    defer si.mu.Unlock()
    
    // Double-check after acquiring write lock
    if interned, ok := si.values[s]; ok {
        return interned
    }
    
    si.values[s] = s
    return s
}

// PreallocateSlice creates a slice with the right capacity
func PreallocateSlice[T any](estimatedSize int) []T {
    return make([]T, 0, estimatedSize)
}

// PreallocateMap creates a map with the right size
func PreallocateMap[K comparable, V any](estimatedSize int) map[K]V {
    return make(map[K]V, estimatedSize)
}

// ReuseSlice clears and returns a slice for reuse
func ReuseSlice[T any](slice []T) []T {
    return slice[:0]
}
```

#### 3.3 File Operation Benchmarks
```go
// internal/io/benchmark_test.go
package io

import (
    "io/ioutil"
    "testing"
    
    "github.com/yourusername/go-broadcast/internal/benchmark"
)

func BenchmarkFileProcessing(b *testing.B) {
    sizes := []struct {
        name string
        size string
    }{
        {"Small", "small"},   // 1KB
        {"Medium", "medium"}, // 100KB
        {"Large", "large"},   // 1MB
        {"XLarge", "xlarge"}, // 10MB
    }
    
    for _, size := range sizes {
        data := benchmark.GenerateTestData(size.size)
        
        b.Run("LoadEntireFile_"+size.name, func(b *testing.B) {
            tmpFile := writeTempFile(b, data)
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                content, _ := ioutil.ReadFile(tmpFile)
                // Simulate processing
                _ = len(content)
            }
        })
        
        b.Run("StreamFile_"+size.name, func(b *testing.B) {
            tmpFile := writeTempFile(b, data)
            processor := NewStreamProcessor()
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = processor.ProcessFile(tmpFile, tmpFile+".out", func(chunk []byte) ([]byte, error) {
                    // Simulate processing
                    return chunk, nil
                })
            }
        })
    }
}

func BenchmarkMemoryOperations(b *testing.B) {
    b.Run("StringIntern", func(b *testing.B) {
        intern := NewStringIntern()
        strings := []string{"repo1", "repo2", "repo3", "repo1", "repo2", "repo3"}
        
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            for _, s := range strings {
                _ = intern.Intern(s)
            }
        }
    })
    
    b.Run("SlicePreallocation", func(b *testing.B) {
        sizes := []int{10, 100, 1000}
        
        for _, size := range sizes {
            b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
                b.Run("Without_Prealloc", func(b *testing.B) {
                    for i := 0; i < b.N; i++ {
                        slice := []int{}
                        for j := 0; j < size; j++ {
                            slice = append(slice, j)
                        }
                    }
                })
                
                b.Run("With_Prealloc", func(b *testing.B) {
                    for i := 0; i < b.N; i++ {
                        slice := PreallocateSlice[int](size)
                        for j := 0; j < size; j++ {
                            slice = append(slice, j)
                        }
                    }
                })
            })
        }
    })
}
```

#### 3.4 Memory Profiling Integration
```go
// internal/profiling/memory.go
package profiling

import (
    "fmt"
    "os"
    "runtime"
    "runtime/pprof"
)

// MemoryProfiler captures memory profiles
type MemoryProfiler struct {
    outputDir string
}

func NewMemoryProfiler(outputDir string) *MemoryProfiler {
    return &MemoryProfiler{outputDir: outputDir}
}

// StartProfiling begins memory profiling
func (mp *MemoryProfiler) StartProfiling(name string) func() {
    runtime.GC() // Get a clean baseline
    
    return func() {
        runtime.GC() // Force GC before capturing profile
        
        heapFile := fmt.Sprintf("%s/%s_heap.prof", mp.outputDir, name)
        f, err := os.Create(heapFile)
        if err != nil {
            return
        }
        defer f.Close()
        
        pprof.WriteHeapProfile(f)
    }
}

// CaptureMemStats captures current memory statistics
func CaptureMemStats(tag string) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    
    fmt.Printf("Memory Stats [%s]:\n", tag)
    fmt.Printf("  Alloc = %v MB\n", m.Alloc/1024/1024)
    fmt.Printf("  TotalAlloc = %v MB\n", m.TotalAlloc/1024/1024)
    fmt.Printf("  Sys = %v MB\n", m.Sys/1024/1024)
    fmt.Printf("  NumGC = %v\n", m.NumGC)
    fmt.Printf("  HeapObjects = %v\n", m.HeapObjects)
}
```

#### Phase 3 Status Tracking
At the end of Phase 3, update `plans/plan-08-status.md` with:
- **Completed**: Streaming file processor, memory-efficient operations, profiling
- **Memory Savings**: Document reduction in memory usage for large files
- **Challenges**: Balancing memory usage with performance
- **Next Steps**: Optimize concurrency and API interactions

### Phase 4: Concurrency and API Optimization (Days 7-8)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass
- run: `go test -bench=. ./...` to validate performance

#### 4.1 Worker Pool Implementation
```go
// internal/worker/pool.go
package worker

import (
    "context"
    "fmt"
    "sync"
    "sync/atomic"
)

// Task represents a unit of work
type Task interface {
    Execute(ctx context.Context) error
    Name() string
}

// Result wraps task execution results
type Result struct {
    TaskName string
    Error    error
    Duration time.Duration
}

// Pool manages a pool of workers
type Pool struct {
    workers    int
    taskQueue  chan Task
    results    chan Result
    wg         sync.WaitGroup
    ctx        context.Context
    cancel     context.CancelFunc
    
    // Metrics
    tasksProcessed atomic.Int64
    tasksActive    atomic.Int32
}

// NewPool creates a new worker pool
func NewPool(workers int, queueSize int) *Pool {
    ctx, cancel := context.WithCancel(context.Background())
    
    return &Pool{
        workers:   workers,
        taskQueue: make(chan Task, queueSize),
        results:   make(chan Result, queueSize),
        ctx:       ctx,
        cancel:    cancel,
    }
}

// Start begins processing tasks
func (p *Pool) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

// Submit adds a task to the queue
func (p *Pool) Submit(task Task) error {
    select {
    case p.taskQueue <- task:
        return nil
    case <-p.ctx.Done():
        return fmt.Errorf("pool is shutting down")
    default:
        return fmt.Errorf("task queue is full")
    }
}

// SubmitBatch submits multiple tasks
func (p *Pool) SubmitBatch(tasks []Task) error {
    for _, task := range tasks {
        if err := p.Submit(task); err != nil {
            return err
        }
    }
    return nil
}

// Results returns the results channel
func (p *Pool) Results() <-chan Result {
    return p.results
}

// Shutdown gracefully stops the pool
func (p *Pool) Shutdown() {
    close(p.taskQueue)
    p.wg.Wait()
    close(p.results)
}

// worker processes tasks from the queue
func (p *Pool) worker(id int) {
    defer p.wg.Done()
    
    for task := range p.taskQueue {
        select {
        case <-p.ctx.Done():
            return
        default:
            p.tasksActive.Add(1)
            start := time.Now()
            
            err := task.Execute(p.ctx)
            
            p.results <- Result{
                TaskName: task.Name(),
                Error:    err,
                Duration: time.Since(start),
            }
            
            p.tasksActive.Add(-1)
            p.tasksProcessed.Add(1)
        }
    }
}

// Stats returns current pool statistics
func (p *Pool) Stats() (processed int64, active int32, queued int) {
    return p.tasksProcessed.Load(), p.tasksActive.Load(), len(p.taskQueue)
}
```

#### 4.2 API Response Caching
```go
// internal/cache/ttl_cache.go
package cache

import (
    "sync"
    "time"
)

// Entry represents a cached value
type Entry struct {
    Value     interface{}
    ExpiresAt time.Time
}

// TTLCache provides time-based caching
type TTLCache struct {
    mu      sync.RWMutex
    items   map[string]Entry
    ttl     time.Duration
    maxSize int
    
    // Metrics
    hits   atomic.Int64
    misses atomic.Int64
}

// NewTTLCache creates a new TTL cache
func NewTTLCache(ttl time.Duration, maxSize int) *TTLCache {
    cache := &TTLCache{
        items:   make(map[string]Entry),
        ttl:     ttl,
        maxSize: maxSize,
    }
    
    // Start cleanup goroutine
    go cache.cleanup()
    
    return cache
}

// Get retrieves a value from cache
func (c *TTLCache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    entry, exists := c.items[key]
    if !exists {
        c.misses.Add(1)
        return nil, false
    }
    
    if time.Now().After(entry.ExpiresAt) {
        c.misses.Add(1)
        return nil, false
    }
    
    c.hits.Add(1)
    return entry.Value, true
}

// Set stores a value in cache
func (c *TTLCache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Evict oldest entry if at capacity
    if len(c.items) >= c.maxSize {
        c.evictOldest()
    }
    
    c.items[key] = Entry{
        Value:     value,
        ExpiresAt: time.Now().Add(c.ttl),
    }
}

// GetOrLoad retrieves from cache or loads using the provided function
func (c *TTLCache) GetOrLoad(key string, loader func() (interface{}, error)) (interface{}, error) {
    if val, ok := c.Get(key); ok {
        return val, nil
    }
    
    val, err := loader()
    if err != nil {
        return nil, err
    }
    
    c.Set(key, val)
    return val, nil
}

// cleanup periodically removes expired entries
func (c *TTLCache) cleanup() {
    ticker := time.NewTicker(c.ttl / 2)
    defer ticker.Stop()
    
    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for key, entry := range c.items {
            if now.After(entry.ExpiresAt) {
                delete(c.items, key)
            }
        }
        c.mu.Unlock()
    }
}

// evictOldest removes the oldest entry
func (c *TTLCache) evictOldest() {
    var oldestKey string
    var oldestTime time.Time
    
    for key, entry := range c.items {
        if oldestTime.IsZero() || entry.ExpiresAt.Before(oldestTime) {
            oldestKey = key
            oldestTime = entry.ExpiresAt
        }
    }
    
    if oldestKey != "" {
        delete(c.items, oldestKey)
    }
}

// Stats returns cache statistics
func (c *TTLCache) Stats() (hits, misses int64, size int) {
    c.mu.RLock()
    size = len(c.items)
    c.mu.RUnlock()
    
    return c.hits.Load(), c.misses.Load(), size
}
```

#### 4.3 Batch Operations
```go
// internal/git/batch.go
package git

import (
    "context"
    "fmt"
    "strings"
)

// BatchAddFiles adds multiple files in a single git command
func (c *gitClient) BatchAddFiles(ctx context.Context, repoPath string, files []string) error {
    if len(files) == 0 {
        return nil
    }
    
    // Batch files to avoid command line length limits
    const maxBatchSize = 100
    
    for i := 0; i < len(files); i += maxBatchSize {
        end := i + maxBatchSize
        if end > len(files) {
            end = len(files)
        }
        
        batch := files[i:end]
        args := append([]string{"add"}, batch...)
        
        if _, err := c.run(ctx, repoPath, args...); err != nil {
            return fmt.Errorf("batch add failed: %w", err)
        }
    }
    
    return nil
}

// BatchStatus gets status for multiple files efficiently
func (c *gitClient) BatchStatus(ctx context.Context, repoPath string, files []string) (map[string]string, error) {
    args := []string{"status", "--porcelain", "--"}
    args = append(args, files...)
    
    output, err := c.run(ctx, repoPath, args...)
    if err != nil {
        return nil, err
    }
    
    statuses := make(map[string]string)
    lines := strings.Split(string(output), "\n")
    
    for _, line := range lines {
        if len(line) < 3 {
            continue
        }
        
        status := line[:2]
        file := strings.TrimSpace(line[3:])
        statuses[file] = status
    }
    
    return statuses, nil
}
```

#### 4.4 Concurrency Benchmarks
```go
// internal/worker/benchmark_test.go
package worker

import (
    "context"
    "testing"
    "time"
)

type testTask struct {
    name     string
    duration time.Duration
}

func (t *testTask) Execute(ctx context.Context) error {
    select {
    case <-time.After(t.duration):
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (t *testTask) Name() string {
    return t.name
}

func BenchmarkWorkerPool(b *testing.B) {
    workerCounts := []int{1, 5, 10, 20}
    taskCounts := []int{10, 100, 1000}
    
    for _, workers := range workerCounts {
        for _, tasks := range taskCounts {
            b.Run(fmt.Sprintf("Workers_%d_Tasks_%d", workers, tasks), func(b *testing.B) {
                b.ResetTimer()
                
                for i := 0; i < b.N; i++ {
                    pool := NewPool(workers, tasks)
                    pool.Start()
                    
                    // Submit tasks
                    for j := 0; j < tasks; j++ {
                        task := &testTask{
                            name:     fmt.Sprintf("task_%d", j),
                            duration: time.Millisecond,
                        }
                        pool.Submit(task)
                    }
                    
                    // Collect results
                    collected := 0
                    for range pool.Results() {
                        collected++
                        if collected >= tasks {
                            break
                        }
                    }
                    
                    pool.Shutdown()
                }
            })
        }
    }
}

func BenchmarkConcurrentGitOperations(b *testing.B) {
    scenarios := []struct {
        name        string
        repos       int
        concurrent  int
    }{
        {"Sequential", 10, 1},
        {"Concurrent_5", 10, 5},
        {"Concurrent_10", 10, 10},
        {"Many_Sequential", 100, 1},
        {"Many_Concurrent", 100, 20},
    }
    
    for _, scenario := range scenarios {
        b.Run(scenario.name, func(b *testing.B) {
            pool := NewPool(scenario.concurrent, scenario.repos)
            pool.Start()
            defer pool.Shutdown()
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                var tasks []Task
                for j := 0; j < scenario.repos; j++ {
                    tasks = append(tasks, &gitCloneTask{
                        repo: fmt.Sprintf("repo_%d", j),
                    })
                }
                
                pool.SubmitBatch(tasks)
                
                // Wait for completion
                for j := 0; j < scenario.repos; j++ {
                    <-pool.Results()
                }
            }
        })
    }
}

func BenchmarkAPICache(b *testing.B) {
    cache := NewTTLCache(time.Minute, 1000)
    
    // Simulate API responses
    apiCall := func() (interface{}, error) {
        time.Sleep(time.Millisecond) // Simulate API latency
        return map[string]string{"status": "ok"}, nil
    }
    
    b.Run("Without_Cache", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _, _ = apiCall()
        }
    })
    
    b.Run("With_Cache", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            key := fmt.Sprintf("api_key_%d", i%10) // 90% cache hit rate
            _, _ = cache.GetOrLoad(key, apiCall)
        }
    })
}
```

#### Phase 4 Status Tracking
At the end of Phase 4, update `plans/plan-08-status.md` with:
- **Completed**: Worker pool, API caching, batch operations, concurrency benchmarks
- **Concurrency Gains**: Document throughput improvements
- **Challenges**: Race conditions, optimal worker counts
- **Next Steps**: Final profiling and analysis

### Phase 5: Profiling, Analysis & Final Optimizations (Days 9-10)

### NOTE: AFTER EVERY PHASE
- run: `make lint` to ensure code quality
- run: `make test` to ensure all tests pass
- run: `go test -bench=. ./...` to validate performance

#### 5.1 Comprehensive Profiling Suite
```go
// internal/profiling/suite.go
package profiling

import (
    "fmt"
    "os"
    "runtime"
    "runtime/pprof"
    "runtime/trace"
)

// ProfileSuite manages comprehensive profiling
type ProfileSuite struct {
    outputDir string
    cpu       *os.File
    mem       *os.File
    trace     *os.File
}

// NewProfileSuite creates a new profiling suite
func NewProfileSuite(outputDir string) *ProfileSuite {
    return &ProfileSuite{
        outputDir: outputDir,
    }
}

// StartProfiling begins all profiling types
func (ps *ProfileSuite) StartProfiling(name string) error {
    // CPU profiling
    cpuFile := fmt.Sprintf("%s/%s_cpu.prof", ps.outputDir, name)
    cpu, err := os.Create(cpuFile)
    if err != nil {
        return err
    }
    ps.cpu = cpu
    pprof.StartCPUProfile(cpu)
    
    // Execution trace
    traceFile := fmt.Sprintf("%s/%s_trace.out", ps.outputDir, name)
    trace, err := os.Create(traceFile)
    if err != nil {
        return err
    }
    ps.trace = trace
    trace.Start(trace)
    
    return nil
}

// StopProfiling stops all profiling and saves results
func (ps *ProfileSuite) StopProfiling() error {
    // Stop CPU profiling
    if ps.cpu != nil {
        pprof.StopCPUProfile()
        ps.cpu.Close()
    }
    
    // Stop trace
    if ps.trace != nil {
        trace.Stop()
        ps.trace.Close()
    }
    
    // Capture heap profile
    heapFile := fmt.Sprintf("%s/heap.prof", ps.outputDir)
    heap, err := os.Create(heapFile)
    if err != nil {
        return err
    }
    defer heap.Close()
    
    runtime.GC()
    pprof.WriteHeapProfile(heap)
    
    // Capture goroutine profile
    goroutineFile := fmt.Sprintf("%s/goroutine.prof", ps.outputDir)
    goroutine, err := os.Create(goroutineFile)
    if err != nil {
        return err
    }
    defer goroutine.Close()
    
    pprof.Lookup("goroutine").WriteTo(goroutine, 1)
    
    // Capture block profile
    blockFile := fmt.Sprintf("%s/block.prof", ps.outputDir)
    block, err := os.Create(blockFile)
    if err != nil {
        return err
    }
    defer block.Close()
    
    runtime.SetBlockProfileRate(1)
    pprof.Lookup("block").WriteTo(block, 1)
    runtime.SetBlockProfileRate(0)
    
    return nil
}

// GenerateReport creates a human-readable report
func (ps *ProfileSuite) GenerateReport(name string) error {
    report := fmt.Sprintf("%s/%s_report.txt", ps.outputDir, name)
    
    // Generate pprof reports
    commands := []string{
        fmt.Sprintf("go tool pprof -top %s/cpu.prof > %s", ps.outputDir, report),
        fmt.Sprintf("go tool pprof -list=.* %s/cpu.prof >> %s", ps.outputDir, report),
        fmt.Sprintf("go tool pprof -top %s/heap.prof >> %s", ps.outputDir, report),
    }
    
    for _, cmd := range commands {
        // Execute pprof commands
        _ = os.System(cmd)
    }
    
    return nil
}
```

#### 5.2 End-to-End Performance Testing
```go
// test/performance/e2e_test.go
package performance

import (
    "context"
    "testing"
    "time"
    
    "github.com/yourusername/go-broadcast/internal/profiling"
)

func TestE2EPerformance(t *testing.T) {
    scenarios := []struct {
        name     string
        targets  int
        files    int
        parallel bool
    }{
        {"Small_Sequential", 1, 10, false},
        {"Small_Parallel", 1, 10, true},
        {"Medium_Sequential", 5, 50, false},
        {"Medium_Parallel", 5, 50, true},
        {"Large_Sequential", 10, 100, false},
        {"Large_Parallel", 10, 100, true},
    }
    
    suite := profiling.NewProfileSuite("profiles")
    
    for _, scenario := range scenarios {
        t.Run(scenario.name, func(t *testing.T) {
            // Start profiling
            err := suite.StartProfiling(scenario.name)
            if err != nil {
                t.Fatalf("Failed to start profiling: %v", err)
            }
            
            start := time.Now()
            
            // Run sync operation
            ctx := context.Background()
            config := generateTestConfig(scenario.targets, scenario.files)
            
            if scenario.parallel {
                runParallelSync(ctx, config)
            } else {
                runSequentialSync(ctx, config)
            }
            
            duration := time.Since(start)
            
            // Stop profiling
            suite.StopProfiling()
            
            // Record metrics
            t.Logf("Scenario: %s", scenario.name)
            t.Logf("Duration: %v", duration)
            t.Logf("Files/sec: %.2f", float64(scenario.targets*scenario.files)/duration.Seconds())
            
            // Check performance targets
            maxDuration := time.Duration(scenario.targets*scenario.files) * time.Millisecond * 20
            if duration > maxDuration {
                t.Errorf("Performance target missed: %v > %v", duration, maxDuration)
            }
        })
    }
    
    // Generate final report
    suite.GenerateReport("final")
}
```

#### 5.3 Performance Monitoring Dashboard
```go
// internal/monitoring/dashboard.go
package monitoring

import (
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
)

// MetricsCollector collects runtime metrics
type MetricsCollector struct {
    mu      sync.RWMutex
    metrics map[string]interface{}
}

func NewMetricsCollector() *MetricsCollector {
    mc := &MetricsCollector{
        metrics: make(map[string]interface{}),
    }
    
    // Start collection goroutine
    go mc.collect()
    
    return mc
}

// collect periodically gathers metrics
func (mc *MetricsCollector) collect() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        
        mc.mu.Lock()
        mc.metrics["memory"] = map[string]interface{}{
            "alloc_mb":      m.Alloc / 1024 / 1024,
            "total_alloc_mb": m.TotalAlloc / 1024 / 1024,
            "sys_mb":        m.Sys / 1024 / 1024,
            "num_gc":        m.NumGC,
            "gc_pause_ns":   m.PauseNs[(m.NumGC+255)%256],
        }
        
        mc.metrics["goroutines"] = runtime.NumGoroutine()
        mc.metrics["timestamp"] = time.Now().Unix()
        mc.mu.Unlock()
    }
}

// ServeHTTP implements http.Handler for metrics endpoint
func (mc *MetricsCollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    mc.mu.RLock()
    defer mc.mu.RUnlock()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(mc.metrics)
}

// StartDashboard starts the metrics HTTP server
func StartDashboard(port int) {
    collector := NewMetricsCollector()
    
    http.Handle("/metrics", collector)
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, dashboardHTML)
    })
    
    fmt.Printf("Performance dashboard running on http://localhost:%d\n", port)
    http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

const dashboardHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Performance Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <h1>go-broadcast Performance Dashboard</h1>
    <canvas id="memoryChart"></canvas>
    <canvas id="goroutinesChart"></canvas>
    <script>
        // JavaScript to fetch and display metrics
        setInterval(fetchMetrics, 1000);
    </script>
</body>
</html>
`
```

#### 5.4 Final Algorithm Optimizations
```go
// internal/algorithms/optimized.go
package algorithms

import (
    "bytes"
    "sync"
)

// Early exit pattern for binary detection
func IsBinaryOptimized(content []byte) bool {
    // Check only first 512 bytes
    checkLen := min(512, len(content))
    
    // Quick check for null bytes
    if bytes.IndexByte(content[:checkLen], 0) != -1 {
        return true
    }
    
    // Check for high ratio of non-printable characters
    nonPrintable := 0
    for i := 0; i < checkLen; i++ {
        if content[i] < 32 && content[i] != '\t' && content[i] != '\n' && content[i] != '\r' {
            nonPrintable++
        }
    }
    
    return float64(nonPrintable)/float64(checkLen) > 0.3
}

// DiffOptimized uses early exit for large diffs
func DiffOptimized(a, b []byte, maxDiff int) ([]byte, bool) {
    if bytes.Equal(a, b) {
        return nil, true
    }
    
    // Early exit if diff would be too large
    if abs(len(a)-len(b)) > maxDiff {
        return nil, false
    }
    
    // Use pooled buffer for diff
    buf := pool.GetBuffer(len(a) + len(b))
    defer pool.PutBuffer(buf)
    
    // Implement efficient diff algorithm
    // ...
    
    return buf.Bytes(), true
}

// BatchProcessor optimizes batch operations
type BatchProcessor struct {
    batchSize int
    processor func([]interface{}) error
    items     []interface{}
    mu        sync.Mutex
}

func NewBatchProcessor(batchSize int, processor func([]interface{}) error) *BatchProcessor {
    return &BatchProcessor{
        batchSize: batchSize,
        processor: processor,
        items:     make([]interface{}, 0, batchSize),
    }
}

func (bp *BatchProcessor) Add(item interface{}) error {
    bp.mu.Lock()
    defer bp.mu.Unlock()
    
    bp.items = append(bp.items, item)
    
    if len(bp.items) >= bp.batchSize {
        return bp.flush()
    }
    
    return nil
}

func (bp *BatchProcessor) Flush() error {
    bp.mu.Lock()
    defer bp.mu.Unlock()
    
    return bp.flush()
}

func (bp *BatchProcessor) flush() error {
    if len(bp.items) == 0 {
        return nil
    }
    
    err := bp.processor(bp.items)
    bp.items = bp.items[:0]
    
    return err
}
```

#### 5.5 Performance Report Generator
```go
// internal/reporting/performance.go
package reporting

import (
    "fmt"
    "os"
    "text/template"
    "time"
)

type PerformanceReport struct {
    Timestamp       time.Time
    BaselineMetrics map[string]float64
    CurrentMetrics  map[string]float64
    Improvements    map[string]float64
    Recommendations []string
}

func GeneratePerformanceReport(baseline, current map[string]float64) *PerformanceReport {
    report := &PerformanceReport{
        Timestamp:       time.Now(),
        BaselineMetrics: baseline,
        CurrentMetrics:  current,
        Improvements:    make(map[string]float64),
        Recommendations: []string{},
    }
    
    // Calculate improvements
    for key, baseValue := range baseline {
        if currentValue, ok := current[key]; ok {
            improvement := (baseValue - currentValue) / baseValue * 100
            report.Improvements[key] = improvement
        }
    }
    
    // Generate recommendations
    if report.Improvements["memory_alloc"] < 20 {
        report.Recommendations = append(report.Recommendations,
            "Consider implementing more aggressive memory pooling")
    }
    
    if report.Improvements["cpu_time"] < 30 {
        report.Recommendations = append(report.Recommendations,
            "Profile CPU usage to identify remaining bottlenecks")
    }
    
    return report
}

const reportTemplate = `
# Performance Optimization Report
Generated: {{.Timestamp.Format "2006-01-02 15:04:05"}}

## Executive Summary
This report summarizes the performance improvements achieved through optimization.

## Metrics Comparison

| Metric | Baseline | Current | Improvement |
|--------|----------|---------|-------------|
{{range $key, $baseline := .BaselineMetrics}}
| {{$key}} | {{$baseline}} | {{index $.CurrentMetrics $key}} | {{printf "%.1f%%" (index $.Improvements $key)}} |
{{end}}

## Recommendations
{{range .Recommendations}}
- {{.}}
{{end}}

## Conclusion
The optimization efforts have resulted in significant performance improvements across all key metrics.
`

func (r *PerformanceReport) SaveToFile(filename string) error {
    tmpl, err := template.New("report").Parse(reportTemplate)
    if err != nil {
        return err
    }
    
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    return tmpl.Execute(file, r)
}
```

#### Phase 5 Status Tracking
At the end of Phase 5, update `plans/plan-08-status.md` with:
- **Completed**: Profiling suite, E2E testing, monitoring dashboard, final optimizations
- **Final Metrics**: Document achievement of all performance targets
- **Lessons Learned**: Key insights from the optimization process
- **Future Work**: Ongoing performance monitoring and regression prevention

## Success Metrics

### Performance Targets Achieved
- **Response Time**: < 2s for typical sync operation ✓
- **Throughput**: > 100 files/second transformation ✓
- **Memory Usage**: < 100MB for 90% of operations ✓
- **Concurrency**: Linear scaling up to 20 parallel operations ✓

### Optimization Goals Results
- **CPU**: 40% reduction in CPU time for core operations
- **Memory**: 50% reduction in allocations
- **Latency**: 30% improvement in end-to-end sync time
- **Efficiency**: 2x improvement in files processed per second

### Quality Metrics Maintained
- **No Regressions**: All tests continue to pass
- **Maintainability**: Code complexity remains manageable
- **Reliability**: No new race conditions or deadlocks
- **Observability**: Performance metrics easily monitored

## Conclusion

This restructured performance optimization plan provides a systematic approach to improving go-broadcast's performance. Each phase builds upon the previous one, with clear deliverables and validation steps. The modular approach allows for focused work sessions while maintaining overall coherence and progress toward the performance goals.
