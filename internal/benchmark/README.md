# benchmark Package

The `benchmark` package provides shared utilities and patterns for consistent benchmarking across the go-broadcast codebase. It helps standardize benchmark setup, execution, and reporting while reducing code duplication.

## Features

- **Memory tracking** - Standardized memory allocation tracking
- **File creation** - Consistent file setup for benchmark tests
- **Repository setup** - Temporary repository creation for benchmarks
- **Size configurations** - Predefined size categories for scaling tests
- **Performance reporting** - Standardized benchmark result formatting

## Core Utilities

### WithMemoryTracking
Wraps benchmark execution with memory allocation tracking:
```go
func WithMemoryTracking(b *testing.B, fn func())

// Usage
benchmark.WithMemoryTracking(b, func() {
    for i := 0; i < b.N; i++ {
        result := processData(testData)
        _ = result
    }
})
```

**Features:**
- Automatically calls `b.ReportAllocs()`
- Handles `b.ResetTimer()` and `b.StopTimer()`
- Provides consistent memory measurement

### SetupBenchmarkFiles
Creates multiple files for benchmark testing:
```go
func SetupBenchmarkFiles(b *testing.B, dir string, count int) []string

// Usage
files := benchmark.SetupBenchmarkFiles(b, tempDir, 100)
// Creates 100 files: bench_file_0.txt through bench_file_99.txt
```

### SetupBenchmarkRepo
Creates a temporary Git repository for benchmarks:
```go
func SetupBenchmarkRepo(b *testing.B) string

// Usage
repoPath := benchmark.SetupBenchmarkRepo(b)
// Returns path to temporary repository, automatically cleaned up
```

### CreateBenchmarkData
Generates test data for benchmarks:
```go
func CreateBenchmarkData(size int, template interface{}) []byte

// Usage
data := benchmark.CreateBenchmarkData(1000, map[string]interface{}{
    "id":   1,
    "name": "test",
})
// Creates JSON array with 1000 items
```

## Size Configurations

### BenchmarkSize
Standardized size configurations for scaling tests:
```go
type BenchmarkSize struct {
    Name      string
    FileCount int
    FileSize  int64
    DataSize  int
}
```

### StandardSizes
Predefined size configurations:
```go
func StandardSizes() []BenchmarkSize

// Usage
sizes := benchmark.StandardSizes()
for _, size := range sizes {
    b.Run(size.Name, func(b *testing.B) {
        // Use size.FileCount, size.FileSize, etc.
    })
}
```

**Standard sizes:**
- **Small**: 10 files, 1KB each, 100 data items
- **Medium**: 100 files, 10KB each, 1000 data items  
- **Large**: 1000 files, 100KB each, 10000 data items
- **XLarge**: 5000 files, 1MB each, 50000 data items

## Performance Monitoring

### MemoryStats
Captures memory statistics during benchmarks:
```go
type MemoryStats struct {
    AllocBytes      uint64
    TotalAllocBytes uint64
    SysBytes        uint64
    NumGC           uint32
}

func CaptureMemoryStats() MemoryStats
```

### BenchmarkResult
Structured benchmark result reporting:
```go
type BenchmarkResult struct {
    Name           string
    Iterations     int
    NsPerOp        int64
    BytesPerOp     int64
    AllocsPerOp    int64
    MemoryStats    MemoryStats
}

func RecordBenchmarkResult(b *testing.B, name string) BenchmarkResult
```

## Usage Examples

### Basic Benchmark with Memory Tracking
```go
func BenchmarkProcessData(b *testing.B) {
    data := generateTestData(1000)
    
    benchmark.WithMemoryTracking(b, func() {
        for i := 0; i < b.N; i++ {
            result := processData(data)
            _ = result // Prevent optimization
        }
    })
}
```

### File-based Benchmark
```go
func BenchmarkFileOperations(b *testing.B) {
    sizes := benchmark.StandardSizes()
    
    for _, size := range sizes {
        b.Run(size.Name, func(b *testing.B) {
            tempDir := b.TempDir()
            files := benchmark.SetupBenchmarkFiles(b, tempDir, size.FileCount)
            
            benchmark.WithMemoryTracking(b, func() {
                for i := 0; i < b.N; i++ {
                    for _, file := range files {
                        data, _ := os.ReadFile(file)
                        _ = data
                    }
                }
            })
        })
    }
}
```

### Git Repository Benchmark
```go
func BenchmarkGitOperations(b *testing.B) {
    repoPath := benchmark.SetupBenchmarkRepo(b)
    
    benchmark.WithMemoryTracking(b, func() {
        for i := 0; i < b.N; i++ {
            // Create a test file
            fileName := fmt.Sprintf("file_%d.txt", i)
            filePath := filepath.Join(repoPath, fileName)
            os.WriteFile(filePath, []byte("test content"), 0600)
            
            // Git operations
            exec.Command("git", "-C", repoPath, "add", fileName).Run()
            exec.Command("git", "-C", repoPath, "commit", "-m", "test").Run()
        }
    })
}
```

### Complex Benchmark with Multiple Metrics
```go
func BenchmarkComplexOperation(b *testing.B) {
    sizes := benchmark.StandardSizes()
    
    for _, size := range sizes {
        b.Run(size.Name, func(b *testing.B) {
            // Setup
            data := benchmark.CreateBenchmarkData(size.DataSize, testTemplate)
            tempDir := b.TempDir()
            files := benchmark.SetupBenchmarkFiles(b, tempDir, size.FileCount)
            
            // Capture initial memory state
            initialMem := benchmark.CaptureMemoryStats()
            
            benchmark.WithMemoryTracking(b, func() {
                for i := 0; i < b.N; i++ {
                    // Process data
                    processedData := processLargeData(data)
                    
                    // File operations
                    for _, file := range files {
                        writeProcessedData(file, processedData)
                    }
                    
                    // Cleanup to prevent accumulation
                    cleanupProcessedData(processedData)
                }
            })
            
            // Record additional metrics
            finalMem := benchmark.CaptureMemoryStats()
            b.ReportMetric(float64(finalMem.AllocBytes-initialMem.AllocBytes), "bytes-delta")
            b.ReportMetric(float64(len(files)), "files-processed")
        })
    }
}
```

## Benchmark Patterns

### CPU-Intensive Benchmarks
```go
func BenchmarkCPUIntensive(b *testing.B) {
    input := generateComplexInput()
    
    benchmark.WithMemoryTracking(b, func() {
        for i := 0; i < b.N; i++ {
            result := cpuIntensiveOperation(input)
            // Use result to prevent optimization
            if result == nil {
                b.Fatal("unexpected nil result")
            }
        }
    })
}
```

### Memory-Intensive Benchmarks
```go
func BenchmarkMemoryIntensive(b *testing.B) {
    sizes := benchmark.StandardSizes()
    
    for _, size := range sizes {
        b.Run(size.Name, func(b *testing.B) {
            benchmark.WithMemoryTracking(b, func() {
                for i := 0; i < b.N; i++ {
                    // Allocate large data structures
                    data := make([]byte, size.FileSize)
                    processLargeBuffer(data)
                    
                    // Force deallocation
                    data = nil
                    runtime.GC()
                }
            })
        })
    }
}
```

### I/O Benchmarks
```go
func BenchmarkIOOperations(b *testing.B) {
    tempDir := b.TempDir()
    files := benchmark.SetupBenchmarkFiles(b, tempDir, 100)
    
    benchmark.WithMemoryTracking(b, func() {
        for i := 0; i < b.N; i++ {
            for _, file := range files {
                data, err := os.ReadFile(file)
                if err != nil {
                    b.Fatal(err)
                }
                
                // Process data
                processFileData(data)
            }
        }
    })
}
```

## Best Practices

1. **Use WithMemoryTracking** - Always wrap benchmark code for consistent measurement
2. **Leverage StandardSizes** - Use predefined sizes for comparable results
3. **Setup once, measure many** - Create test data outside the measured loop
4. **Prevent optimization** - Use results to prevent compiler optimization
5. **Clean up resources** - Use benchmark utilities for automatic cleanup
6. **Report custom metrics** - Use `b.ReportMetric()` for domain-specific measurements

## Performance Analysis

### Memory Usage Analysis
```go
func BenchmarkWithMemoryAnalysis(b *testing.B) {
    before := benchmark.CaptureMemoryStats()
    
    benchmark.WithMemoryTracking(b, func() {
        for i := 0; i < b.N; i++ {
            data := allocateData()
            processData(data)
            // Note: data not explicitly freed
        }
    })
    
    after := benchmark.CaptureMemoryStats()
    
    // Report memory growth
    memGrowth := after.AllocBytes - before.AllocBytes
    b.ReportMetric(float64(memGrowth), "memory-growth-bytes")
    
    // Report GC pressure
    gcCycles := after.NumGC - before.NumGC
    b.ReportMetric(float64(gcCycles), "gc-cycles")
}
```

### Throughput Measurement
```go
func BenchmarkThroughput(b *testing.B) {
    data := generateTestData(10000)
    
    start := time.Now()
    benchmark.WithMemoryTracking(b, func() {
        for i := 0; i < b.N; i++ {
            processData(data)
        }
    })
    duration := time.Since(start)
    
    // Calculate and report throughput
    itemsProcessed := b.N * len(data)
    throughput := float64(itemsProcessed) / duration.Seconds()
    b.ReportMetric(throughput, "items/sec")
}
```

## Integration with CI/CD

### Benchmark Comparison
```go
// Use consistent naming for benchmark comparison across commits
func BenchmarkStableAPI_ProcessRequest(b *testing.B) {
    benchmark.WithMemoryTracking(b, func() {
        // Stable benchmark for regression detection
    })
}
```

### Performance Budgets
```go
func BenchmarkPerformanceBudget(b *testing.B) {
    const maxAllocsPerOp = 100
    const maxNsPerOp = 1000000 // 1ms
    
    result := benchmark.RecordBenchmarkResult(b, "performance-budget")
    
    if result.AllocsPerOp > maxAllocsPerOp {
        b.Errorf("Too many allocations: %d > %d", result.AllocsPerOp, maxAllocsPerOp)
    }
    
    if result.NsPerOp > maxNsPerOp {
        b.Errorf("Too slow: %dns > %dns", result.NsPerOp, maxNsPerOp)
    }
}
```