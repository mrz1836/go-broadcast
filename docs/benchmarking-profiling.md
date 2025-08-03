# Benchmarking & Profiling Guide

go-broadcast includes comprehensive benchmarking and profiling capabilities to help you understand and optimize performance. This guide covers everything from running basic benchmarks to advanced profiling techniques.

## Table of Contents
- [Quick Start](#quick-start)
- [Running Benchmarks](#running-benchmarks)
- [Benchmark Categories](#benchmark-categories)
- [Understanding Results](#understanding-results)
- [Writing Custom Benchmarks](#writing-custom-benchmarks)
- [Performance Tips](#performance-tips)

## Quick Start

Run all benchmarks:
```bash
make bench
```

Run benchmarks for a specific package:
```bash
go test -bench=. ./internal/git
```

Run benchmarks with memory profiling:
```bash
go test -bench=. -benchmem ./internal/git
```

Run a specific benchmark:
```bash
go test -bench=BenchmarkGitCommand_Simple ./internal/git
```

## Running Benchmarks

### Basic Usage

The `make bench` command runs all benchmarks in the project:
```bash
make bench
# Equivalent to: go test -bench=. -benchmem
```

### Advanced Options

Control benchmark execution time:
```bash
# Run each benchmark for at least 10 seconds
go test -bench=. -benchtime=10s ./internal/git

# Run each benchmark for exactly 1000 iterations
go test -bench=. -benchtime=1000x ./internal/git
```

Compare benchmark results:
```bash
# Save baseline
go test -bench=. ./internal/git > baseline.txt

# Make changes, then compare
go test -bench=. ./internal/git > new.txt
benchstat baseline.txt new.txt
```

### Filtering Benchmarks

Run benchmarks matching a pattern:
```bash
# Run only Git clone benchmarks
go test -bench=Clone ./internal/git

# Run all file operation benchmarks
go test -bench="File|Add|Commit" ./internal/git
```

## Benchmark Categories

go-broadcast includes over 100 benchmarks across major components:

### 1. Git Operations (`internal/git`)
- **Command Execution**: Simple commands, output parsing, large outputs
- **Clone Operations**: Small/medium/large repos, shallow clones
- **File Operations**: Add, commit, diff operations
- **Concurrent Operations**: Parallel git commands
- **Batch Operations**: Multiple operations in sequence

Example benchmarks:
```
BenchmarkGitCommand_Simple         - Basic command overhead
BenchmarkGitCommand_WithOutput     - Commands with stdout parsing
BenchmarkClone_Scenarios           - Various repo sizes
BenchmarkAdd_FileCount             - Adding different numbers of files
BenchmarkDiff_Sizes                - Diff performance by change size
BenchmarkConcurrentGitOperations   - Parallel command execution
```

### 2. GitHub API (`internal/gh`)
- **API Commands**: Simple calls, paginated results
- **JSON Operations**: Parsing small/medium/large responses
- **File Operations**: Base64 encoding/decoding
- **Concurrent API Calls**: Parallel API requests
- **PR Operations**: Creating, updating, listing PRs

Example benchmarks:
```
BenchmarkGHCommand_Simple          - Basic API call overhead
BenchmarkListBranches_Sizes        - Branch listing performance
BenchmarkParseJSON_Sizes           - JSON parsing by response size
BenchmarkConcurrentAPICalls        - Parallel API performance
BenchmarkPROperations_Scenarios    - Full PR workflows
```

### 3. State Management (`internal/state`)
- **Branch Parsing**: Extracting metadata from branch names
- **PR Parsing**: Processing PR descriptions
- **State Comparison**: Determining sync status
- **State Aggregation**: Combining multiple states

Example benchmarks:
```
BenchmarkBranchParsing            - Branch name parsing speed
BenchmarkPRParsing                - PR metadata extraction
BenchmarkStateComparison          - Sync status determination
BenchmarkSyncBranchGeneration     - Creating sync branch names
```

### 4. Configuration (`internal/config`)
- **YAML Parsing**: Small/large config files
- **Validation**: Config validation performance
- **Load & Validate**: Combined operations

Example benchmarks:
```
BenchmarkLoadFromReader           - YAML parsing performance
BenchmarkValidate                 - Validation overhead
BenchmarkLoadAndValidate          - Full config processing
```

### 5. File Transformations (`internal/transform`)
- **Template Substitution**: Variable replacement
- **Repository Transforms**: Go module updates, file modifications
- **Binary Detection**: Identifying binary vs text files
- **Transform Chains**: Multiple transformations

Example benchmarks:
```
BenchmarkTemplateTransform_Small   - Small file transformations
BenchmarkRepoTransform_GoFile      - Go module path updates
BenchmarkBinaryDetection           - Binary file detection speed
BenchmarkChainTransform            - Multiple transform performance
```

### 6. Worker Pools (`internal/worker`)
- **Pool Performance**: Task submission and execution
- **Throughput**: Tasks per second at various concurrency levels
- **CPU-Intensive Tasks**: Heavy computation workloads
- **Memory Usage**: Pool memory overhead
- **Scaling**: Performance at different worker counts

Example benchmarks:
```
BenchmarkWorkerPool               - Basic pool operations
BenchmarkWorkerPoolThroughput     - Maximum throughput testing
BenchmarkWorkerPoolScaling        - Scaling characteristics
BenchmarkWorkerPoolBatchSubmission - Batch task submission
```

### 7. Caching (`internal/cache`)
- **Basic Operations**: Get, set, delete performance
- **Hit Rates**: Cache effectiveness measurement
- **Concurrency**: Parallel cache access
- **Memory Usage**: Cache memory overhead
- **Expiration**: TTL and eviction performance

Example benchmarks:
```
BenchmarkCacheBasicOperations     - Get/Set/Delete speed
BenchmarkCacheHitRates            - Hit/miss performance
BenchmarkCacheConcurrency         - Concurrent access patterns
BenchmarkCacheExpiration          - TTL handling overhead
```

### 8. Logging (`internal/logging`)
- **Redaction**: Secret removal performance
- **Formatting**: Text vs JSON output speed
- **Concurrent Logging**: Parallel log writes
- **Pattern Matching**: Regex performance for redaction

Example benchmarks:
```
BenchmarkRedaction_Scenarios      - Various redaction patterns
BenchmarkFormatting_Types         - Different output formats
BenchmarkConcurrentLogging        - Parallel logging performance
BenchmarkPatternMatching          - Regex matching speed
```

### 9. I/O Operations (`internal/io`)
- **File Processing**: Reading, writing, transforming files
- **JSON Processing**: Serialization/deserialization
- **Batch Processing**: Multiple file operations
- **Memory Efficiency**: Large file handling

Example benchmarks:
```
BenchmarkFileProcessing           - File I/O performance
BenchmarkJSONProcessing           - JSON operations
BenchmarkBatchProcessing          - Multiple file handling
BenchmarkMemoryEfficiency         - Memory usage patterns
```

### 10. Sync Operations (`internal/sync`)
- **Target Filtering**: Repository selection
- **Progress Tracking**: Status update overhead
- **Sync Decision**: Determining what needs syncing
- **Concurrent Progress**: Parallel progress updates

Example benchmarks:
```
BenchmarkFilterTargets            - Target selection speed
BenchmarkNeedsSync                - Sync decision performance
BenchmarkProgressTracking         - Progress update overhead
BenchmarkProgressConcurrent       - Parallel progress tracking
```

## Understanding Results

### Benchmark Output Format

```
BenchmarkGitCommand_Simple-8        50000      23456 ns/op     1024 B/op       12 allocs/op
│                          │          │           │              │               │
│                          │          │           │              │               └─ Allocations per operation
│                          │          │           │              └─ Bytes allocated per operation
│                          │          │           └─ Nanoseconds per operation
│                          │          └─ Number of iterations
│                          └─ GOMAXPROCS value (8 cores)
└─ Benchmark name
```

### Key Metrics

1. **ns/op (Nanoseconds per Operation)**
   - Lower is better
   - Compare against baseline for regressions
   - Consider expected operation time

2. **B/op (Bytes per Operation)**
   - Memory allocated per operation
   - Zero allocations is ideal for hot paths
   - Watch for unexpected increases

3. **allocs/op (Allocations per Operation)**
   - Number of heap allocations
   - Fewer allocations = less GC pressure
   - Critical for high-frequency operations

### Performance Targets

Based on current benchmarks, here are performance targets:

| Operation Type       | Target Performance        | Notes                     |
|----------------------|---------------------------|---------------------------|
| Config Parsing       | <40μs for small configs   | ~34K ops/sec              |
| Binary Detection     | <50ns per check           | 77M+ ops/sec, zero allocs |
| State Comparison     | <10ns per comparison      | 26M+ ops/sec, zero allocs |
| Git Commands         | <25ms for simple commands | Includes process overhead |
| API Calls            | <100ms per call           | Network dependent         |
| Transform Operations | <40μs for small files     | ~31K files/sec            |

## Writing Custom Benchmarks

### Basic Benchmark Structure

```go
func BenchmarkMyOperation(b *testing.B) {
    // Setup code (not timed)
    data := prepareTestData()

    b.ResetTimer() // Start timing here

    for i := 0; i < b.N; i++ {
        // Code to benchmark
        result := myOperation(data)

        // Prevent compiler optimization
        _ = result
    }
}
```

### Table-Driven Benchmarks

```go
func BenchmarkFileOperations(b *testing.B) {
    sizes := []struct {
        name string
        size int
    }{
        {"Small", 1024},
        {"Medium", 1024 * 100},
        {"Large", 1024 * 1024},
    }

    for _, tc := range sizes {
        b.Run(tc.name, func(b *testing.B) {
            data := generateData(tc.size)
            b.ResetTimer()

            for i := 0; i < b.N; i++ {
                processFile(data)
            }
        })
    }
}
```

### Memory Profiling in Benchmarks

```go
func BenchmarkMemoryIntensive(b *testing.B) {
    b.ReportAllocs() // Enable allocation reporting

    for i := 0; i < b.N; i++ {
        // Track specific allocations
        before := testing.AllocsPerRun(1, func() {
            // Warm-up run
            myOperation()
        })

        after := testing.AllocsPerRun(100, func() {
            myOperation()
        })

        if after > before*1.1 {
            b.Fatalf("Allocation regression: %f -> %f", before, after)
        }
    }
}
```

### Parallel Benchmarks

```go
func BenchmarkConcurrentOperation(b *testing.B) {
    b.RunParallel(func(pb *testing.PB) {
        // Each goroutine gets its own data
        data := prepareLocalData()

        for pb.Next() {
            result := concurrentOperation(data)
            _ = result
        }
    })
}
```

## Performance Tips

### 1. Identify Hot Paths
- Use CPU profiling to find bottlenecks
- Focus optimization on frequently called code
- Benchmark before and after changes

### 2. Reduce Allocations
- Reuse buffers with `sync.Pool`
- Pre-allocate slices with known capacity
- Use value types instead of pointers where appropriate

### 3. Optimize String Operations
- Use `strings.Builder` for concatenation
- Avoid repeated string conversions
- Consider `[]byte` for manipulation-heavy code

### 4. Efficient Concurrency
- Use worker pools for bounded parallelism
- Batch operations to reduce overhead
- Profile goroutine creation costs

### 5. Cache Wisely
- Cache expensive computations
- Use TTL to prevent stale data
- Monitor cache hit rates

### 6. Benchmark Regularly
- Add benchmarks for new features
- Run benchmarks in CI for regression detection
- Compare results across Git commits

## Developer Workflow Integration

For comprehensive go-broadcast development workflows, see [CLAUDE.md](../.github/CLAUDE.md#-performance-testing-and-benchmarking) which includes:
- **Essential development commands** for performance testing
- **Benchmark execution procedures** for different scenarios
- **Performance analysis workflows** for optimization
- **Troubleshooting quick reference** for performance issues

## Related Documentation

1. [Profiling Guide](profiling-guide.md) - Deep dive into CPU, memory, and goroutine profiling
2. [Performance Optimization](performance-optimization.md) - Best practices based on benchmark results
3. [Troubleshooting Guide](troubleshooting.md) - Debugging performance issues
4. [CLAUDE.md Developer Workflows](../.github/CLAUDE.md) - Complete development workflow integration

## Example Workflow

1. **Baseline Performance**
   ```bash
   go test -bench=. -benchmem ./internal/git > baseline.txt
   ```

2. **Make Optimizations**
   - Identify bottlenecks with profiling
   - Apply targeted optimizations
   - Maintain test coverage

3. **Measure Impact**
   ```bash
   go test -bench=. -benchmem ./internal/git > optimized.txt
   benchstat baseline.txt optimized.txt
   ```

4. **Validate Results**
   - Ensure tests still pass
   - Check for allocation regressions
   - Verify performance improvements

Remember: Measure first, optimize second. Not all code needs to be fast - focus on the parts that matter for your use case.
