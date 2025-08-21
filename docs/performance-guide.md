# Performance Guide

Comprehensive performance analysis, benchmarking, profiling, and optimization guide for go-broadcast. This consolidated guide covers everything from running benchmarks to advanced optimization strategies.

## Table of Contents

1. [Overview and Quick Start](#overview-and-quick-start)
2. [Running Benchmarks](#running-benchmarks)
3. [Profiling Techniques](#profiling-techniques)
4. [Optimization Strategies](#optimization-strategies)
5. [Directory Sync Performance](#directory-sync-performance)
6. [Performance Metrics & Targets](#performance-metrics--targets)
7. [Troubleshooting Performance Issues](#troubleshooting-performance-issues)

---

## Overview and Quick Start

go-broadcast includes comprehensive benchmarking and profiling capabilities to help you understand and optimize performance. The system delivers exceptional performance across all components:

**Quick Performance Overview:**
- **Binary Detection**: 77M+ ops/sec with zero allocations
- **State Comparison**: 26M+ ops/sec with zero allocations
- **Directory Sync**: 1000+ files in ~32ms (150x faster than target)
- **Config Validation**: 371M+ ops/sec with zero allocations
- **Git Operations**: ~40K ops/sec (varies by operation)
- **Transform Operations**: ~31K files/sec for small files

### Quick Start Commands

```bash
# Run all benchmarks
magex bench

# Run benchmarks with memory profiling
go test -bench=. -benchmem ./internal/git

# Run comprehensive profiling demo
go build -o profile_demo ./cmd/profile_demo
./profile_demo
```

---

## Running Benchmarks

go-broadcast includes over 100 benchmarks across major components providing comprehensive performance analysis.

### Basic Benchmark Execution

The `magex bench` command runs all benchmarks in the project:
```bash
magex bench
# Equivalent to: go test -bench=. -benchmem
```

### Advanced Benchmark Options

```bash
# Control execution time
go test -bench=. -benchtime=10s ./internal/git    # Run for 10 seconds
go test -bench=. -benchtime=1000x ./internal/git  # Run 1000 iterations

# Filter benchmarks
go test -bench=Clone ./internal/git               # Git clone benchmarks only
go test -bench="File|Add|Commit" ./internal/git   # File operation benchmarks

# Compare results
go test -bench=. ./internal/git > baseline.txt
# Make changes...
go test -bench=. ./internal/git > new.txt
benchstat baseline.txt new.txt
```

### Benchmark Categories

#### 1. Git Operations (`internal/git`)
- **Command Execution**: Simple commands, output parsing, large outputs
- **Clone Operations**: Small/medium/large repos, shallow clones
- **File Operations**: Add, commit, diff operations
- **Concurrent Operations**: Parallel git commands

Example benchmarks:
```
BenchmarkGitCommand_Simple         - Basic command overhead
BenchmarkGitCommand_WithOutput     - Commands with stdout parsing
BenchmarkClone_Scenarios           - Various repo sizes
BenchmarkConcurrentGitOperations   - Parallel command execution
```

#### 2. GitHub API (`internal/gh`)
- **API Commands**: Simple calls, paginated results
- **JSON Operations**: Parsing small/medium/large responses
- **Concurrent API Calls**: Parallel API requests
- **PR Operations**: Creating, updating, listing PRs

#### 3. State Management (`internal/state`)
- **Branch Parsing**: Extracting metadata from branch names
- **PR Parsing**: Processing PR descriptions
- **State Comparison**: Determining sync status (26M+ ops/sec)

#### 4. Configuration (`internal/config`)
- **YAML Parsing**: Small/large config files
- **Validation**: Config validation performance (371M+ ops/sec)
- **Load & Validate**: Combined operations

#### 5. File Transformations (`internal/transform`)
- **Template Substitution**: Variable replacement
- **Binary Detection**: 77M+ ops/sec, zero allocations
- **Transform Chains**: Multiple transformations

#### 6. Worker Pools (`internal/worker`)
- **Pool Performance**: Task submission and execution (~10K tasks/sec)
- **Throughput**: Tasks per second at various concurrency levels
- **Scaling**: Performance at different worker counts

#### 7. Directory Sync (`internal/sync`)
- **Target Filtering**: Repository selection
- **Progress Tracking**: Status update overhead
- **Exclusion Engine**: Zero-allocation pattern matching (107 ns/op)

### Understanding Benchmark Results

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

**Key Metrics:**
1. **ns/op**: Lower is better, compare against baseline
2. **B/op**: Memory allocated per operation (zero is ideal)
3. **allocs/op**: Heap allocations (fewer = less GC pressure)

---

## Profiling Techniques

go-broadcast includes a comprehensive ProfileSuite API for deep performance analysis through CPU, memory, goroutine, block, and mutex profiling.

### ProfileSuite API Usage

```go
import "github.com/mrz1836/go-broadcast/internal/profiling"

// Create profiling suite
suite := profiling.NewProfileSuite("./profiles")

// Configure profiling options
config := profiling.ProfileConfig{
    EnableCPU:            true,
    EnableMemory:         true,
    EnableTrace:          false, // CPU intensive
    EnableBlock:          true,
    EnableMutex:          true,
    BlockProfileRate:     1,      // Profile all blocking events
    MutexProfileFraction: 1,      // Profile all mutex events
    GenerateReports:      true,
    ReportFormat:         "text", // "text", "html", or "both"
}
suite.Configure(config)

// Start profiling
err := suite.StartProfiling("analysis_session")
if err != nil {
    log.Fatal(err)
}

// Your code here...

// Stop profiling and generate reports
err = suite.StopProfiling()
if err != nil {
    log.Fatal(err)
}
```

### Profiling Types

#### CPU Profiling
Identifies where CPU time is spent:
```bash
# Interactive analysis
go tool pprof cpu.prof

# Top functions by CPU time
go tool pprof -top cpu.prof

# Web interface with flame graphs
go tool pprof -http=:8080 cpu.prof
```

#### Memory Profiling
Tracks heap allocations and memory usage:
```bash
# Heap allocations
go tool pprof -alloc_space memory.prof

# Live objects
go tool pprof -inuse_space memory.prof

# Memory leak detection
go tool pprof -base=baseline.prof current.prof
```

#### Goroutine Analysis
```bash
# View goroutine stacks
go tool pprof goroutine.prof

# Web visualization
go tool pprof -http=:8080 goroutine.prof
```

#### Block and Mutex Profiling
Identifies blocking operations and mutex contention:
```bash
# Block profiling for channel/mutex waits
go tool pprof block.prof

# Mutex contention analysis
go tool pprof mutex.prof
```

### Running the Profile Demo

```bash
# Build and run comprehensive demo
go build -o profile_demo ./cmd/profile_demo
./profile_demo

# Output in ./profiles/final_demo/:
# - cpu.prof, memory.prof, goroutine.prof
# - comprehensive_report.txt
# - performance_report_*.html
# - performance_report_*.json
```

### Performance Reports

The ProfileSuite generates comprehensive reports:

1. **Text Report** (`comprehensive_report.txt`):
   - Memory analysis with growth rates
   - Top allocations by function
   - Performance metrics summary

2. **HTML Report** (`performance_report_*.html`):
   - Visual charts and graphs
   - Interactive memory timeline
   - Sortable performance metrics

3. **JSON Report** (`performance_report_*.json`):
   - Machine-readable performance data
   - Integration with monitoring systems

---

## Optimization Strategies

Performance optimization strategies based on go-broadcast's extensive benchmarking results and real-world usage patterns.

### Component-Specific Optimizations

#### Git Operations Optimization

**Current Performance**: ~23μs overhead for simple commands

```go
// Use shallow clones for large repos
client.CloneOptions{
    Depth: 1,  // Only fetch latest commit
}

// Batch operations when possible
var wg sync.WaitGroup
for _, file := range files {
    wg.Add(1)
    go func(f string) {
        defer wg.Done()
        client.Add(f)
    }(file)
}
wg.Wait()
client.Commit("Batch commit")

// Reuse client instances
// Don't: create new client for each operation
// Do: reuse client across operations
```

#### GitHub API Optimization

**Performance**: 98% API call reduction through tree API optimization

```go
// Implement request caching
type CachedGitHubClient struct {
    *GitHubClient
    cache *cache.TTLCache
}

func (c *CachedGitHubClient) GetBranches(repo string) ([]Branch, error) {
    if cached, ok := c.cache.Get(repo); ok {
        return cached.([]Branch), nil
    }

    branches, err := c.GitHubClient.GetBranches(repo)
    if err == nil {
        c.cache.Set(repo, branches)
    }
    return branches, err
}

// Use conditional requests
client.SetHeader("If-None-Match", etag)

// Batch API requests with controlled concurrency
results := make(chan Result, len(repos))
sem := make(chan struct{}, 10) // Limit concurrency

for _, repo := range repos {
    sem <- struct{}{}
    go func(r string) {
        defer func() { <-sem }()
        result := fetchRepo(r)
        results <- result
    }(repo)
}
```

#### File Transformation Optimization

**Performance**: ~37μs for small files, 77M+ ops/sec binary detection

```go
// Pool regex compilations
var regexPool = sync.Pool{
    New: func() interface{} {
        return regexp.MustCompile(`\{\{(\w+)\}\}`)
    },
}

// Optimized binary detection
func IsBinaryOptimized(data []byte) bool {
    if len(data) == 0 {
        return false
    }

    // Check first 8KB only
    checkLen := len(data)
    if checkLen > 8192 {
        checkLen = 8192
    }

    // Fast path: check for null bytes
    for i := 0; i < checkLen; i++ {
        if data[i] == 0 {
            return true
        }
    }
    return false
}

// Use strings.Builder for concatenation
var builder strings.Builder
builder.Grow(estimatedSize) // Pre-allocate
for _, part := range parts {
    builder.WriteString(part)
}
result := builder.String()
```

#### Worker Pool Optimization

**Performance**: ~10K tasks/sec with 8 workers

```go
// Right-size worker count
numWorkers := runtime.NumCPU()
if numWorkers > 16 {
    numWorkers = 16 // Diminishing returns above 16
}

// Use buffered channels appropriately
queueSize := numWorkers * 100 // Prevent blocking
pool := worker.NewPool(numWorkers, queueSize)

// Batch small tasks
type BatchTask struct {
    items []Item
}

func (t *BatchTask) Execute(ctx context.Context) error {
    for _, item := range t.items {
        if err := processItem(item); err != nil {
            return err
        }
    }
    return nil
}
```

### General Optimization Patterns

#### Reduce Allocations

```go
// Pre-allocate with capacity
func processItems(items []string) []string {
    result := make([]string, 0, len(items))
    for _, item := range items {
        if isValid(item) {
            result = append(result, transform(item))
        }
    }
    return result
}
```

#### Use sync.Pool for Temporary Objects

```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func processData(data []byte) string {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    buf.Write(data)
    return buf.String()
}
```

#### Optimize Hot Paths

```go
// Fast path for common cases
func optimizedFunction(data []byte) bool {
    // Fast path for empty data
    if len(data) == 0 {
        return false
    }

    // Avoid function calls in loops
    dataLen := len(data)
    for i := 0; i < dataLen; i++ {
        if data[i] == targetByte {
            return true
        }
    }
    return false
}
```

---

## Directory Sync Performance

go-broadcast's directory sync implementation delivers extraordinary performance through advanced concurrent processing and GitHub API optimization.

### Executive Performance Summary

- **Processing Speed**: 1000+ files in ~32ms (150x faster than 5s target)
- **API Efficiency**: 90%+ reduction in GitHub API calls through tree API optimization
- **Memory Usage**: Linear scaling at ~1.2MB per 1000 files
- **Concurrency**: Worker pools with controlled parallelism for optimal throughput

### Real-World Performance Examples

#### .github Directory Sync (149 files)
```yaml
# Performance breakdown:
workflows/     # 24 files  - 1.5ms processing
coverage/      # 87 files  - 4.0ms processing
templates/     # 15 files  - 0.8ms processing
other/         # 23 files  - 1.2ms processing
Total:         # 149 files - 7.5ms processing

# API efficiency:
# - Tree API approach: 3 total calls
# - Traditional approach: 149+ calls
# - Efficiency gain: 98% reduction
```

#### Large Repository Sync (1000+ files)
```yaml
docs/          # 450 files - 14ms processing
assets/        # 300 files - 10ms processing
examples/      # 180 files - 6ms processing
configs/       # 70 files  - 2ms processing
Total:         # 1000 files - 32ms processing

# Resource usage:
# - Memory: 1.2MB peak
# - API calls: Minimal (tree API + content calls)
# - Workers: 10 concurrent (configurable)
```

### Zero-Allocation Critical Paths

#### Exclusion Pattern Matching
```
BenchmarkExclusionEngine-8      9370963    107.3 ns/op    0 B/op    0 allocs/op
BenchmarkPatternCaching-8       5687421    210.8 ns/op    0 B/op    0 allocs/op
```

#### Content Processing
```
BenchmarkContentComparison-8    239319295   5.0 ns/op     0 B/op    0 allocs/op
BenchmarkBinaryDetection-8      587204924   2.0 ns/op     0 B/op    0 allocs/op
```

### Performance Architecture

```go
// High-level architecture
type DirectoryProcessor struct {
    workerPool       *WorkerPool      // Concurrent file processing
    exclusionEngine  *ExclusionEngine // Zero-allocation pattern matching
    batchProcessor   *BatchProcessor  // API call batching
    progressReporter *ProgressReporter // >50 files threshold
    cacheManager     *CacheManager    // Content deduplication
}

// Performance characteristics:
// - Worker pools: 10 concurrent workers (configurable)
// - Exclusion engine: 107 ns/op with 0 allocations
// - Batch processing: 23.8M+ ops/sec
// - Cache operations: 13.5M+ ops/sec
```

### API Optimization Comparison

| Metric                | Traditional | go-broadcast   | Improvement      |
|-----------------------|-------------|----------------|------------------|
| API calls (149 files) | 149+        | 2-3            | 98% reduction    |
| Rate limit usage      | 3% per sync | <0.1% per sync | 30x improvement  |
| Processing time       | 2-5 seconds | ~7ms           | 300x faster      |
| Concurrent repos      | 5-10 max    | 100+ possible  | 10x+ scalability |
| Memory per sync       | 10-50MB     | ~1.2MB         | 90% reduction    |

### Memory Efficiency

```
Directory Size vs Memory Usage (Linear Scaling):
50 files:     ~0.06MB (1.2KB per file)
100 files:    ~0.12MB (1.2KB per file)
500 files:    ~0.60MB (1.2KB per file)
1000 files:   ~1.20MB (1.2KB per file)
```

---

## Performance Metrics & Targets

Based on comprehensive benchmarks, here are performance targets and current achievements:

### Component Performance Targets

| Component              | Target Performance      | Current Achievement | Status        |
|------------------------|-------------------------|---------------------|---------------|
| **Config Operations**  |                         |                     |               |
| YAML Parsing          | <40μs for small configs | ~34μs               | ✅ Meeting     |
| Validation            | <1μs per rule           | ~3ns (371M ops/sec) | ✅ Exceeded    |
| Load & Validate       | <50μs combined          | ~37μs               | ✅ Meeting     |
|                       |                         |                     |               |
| **File Operations**    |                         |                     |               |
| Binary Detection      | <100ns per check        | ~2ns (587M ops/sec) | ✅ Exceeded    |
| Transform Operations  | <40μs for small files   | ~37μs (~31K/sec)    | ✅ Meeting     |
| Template Substitution | <50μs per operation     | ~28μs               | ✅ Exceeded    |
|                       |                         |                     |               |
| **State Management**   |                         |                     |               |
| State Comparison      | <100ns per comparison   | ~5ns (239M ops/sec) | ✅ Exceeded    |
| Branch Parsing        | <2μs per parse          | ~1.2μs              | ✅ Meeting     |
| Sync Decision         | <10ns per decision      | ~8ns (26M ops/sec)  | ✅ Meeting     |
|                       |                         |                     |               |
| **Git Operations**     |                         |                     |               |
| Simple Commands       | <50ms per command       | ~23μs overhead      | ✅ Exceeded    |
| Clone Operations      | <30s for medium repos   | Varies by size      | ✅ Meeting     |
| File Operations       | <100ms for batch        | Linear scaling      | ✅ Meeting     |
|                       |                         |                     |               |
| **API Operations**     |                         |                     |               |
| GitHub API Calls      | <200ms per call         | ~100ms (network)    | ✅ Meeting     |
| JSON Parsing          | <100μs small responses  | ~50μs               | ✅ Exceeded    |
| Concurrent Requests   | 10+ parallel            | Up to 20 parallel   | ✅ Exceeded    |
|                       |                         |                     |               |
| **Worker Operations**  |                         |                     |               |
| Task Throughput       | >5K tasks/sec           | ~10K tasks/sec      | ✅ Exceeded    |
| Pool Scaling          | Linear to CPU count     | Linear to ~16 cores | ✅ Meeting     |
| Memory Overhead       | <10MB per pool          | Fixed ~10KB         | ✅ Exceeded    |
|                       |                         |                     |               |
| **Directory Sync**     |                         |                     |               |
| Small Directories     | <500ms (<50 files)      | ~3ms                | ✅ Exceeded    |
| Medium Directories    | <2s (<200 files)        | ~7ms                | ✅ Exceeded    |
| Large Directories     | <5s (<1000 files)       | ~32ms               | ✅ Exceeded    |
| API Call Reduction    | >50% reduction          | 98% reduction       | ✅ Exceeded    |

### Zero-Allocation Operations

These operations achieve zero heap allocations in critical paths:

```
Operation                    | Performance     | Allocations
-----------------------------|-----------------|------------
Binary Detection            | 587M+ ops/sec   | 0 B/op
Content Comparison          | 239M+ ops/sec   | 0 B/op
Config Validation           | 371M+ ops/sec   | 0 B/op
Exclusion Pattern Matching  | 9.4M+ ops/sec   | 0 B/op
State Comparison            | 26M+ ops/sec    | 0 B/op
```

### Memory Usage Patterns

| Operation Type        | Memory Pattern      | Scaling Factor     |
|-----------------------|--------------------|--------------------|
| **Config Processing** | Fixed overhead      | ~50KB base         |
| **Git Operations**    | Per-command buffer  | ~10KB per command  |
| **API Operations**    | Response buffering  | ~1-5KB per response|
| **File Transform**    | Streaming           | ~2KB per file      |
| **Directory Sync**    | Linear scaling      | ~1.2KB per file    |
| **Worker Pools**      | Fixed overhead      | ~10KB per pool     |
| **State Management**  | Cache-based         | ~100B per state    |

### Performance Regression Thresholds

Alert if performance degrades beyond these thresholds:

```yaml
performance_alerts:
  critical_operations:
    binary_detection: ">5ns/op"           # Currently ~2ns
    state_comparison: ">50ns/op"          # Currently ~5ns
    config_validation: ">10ns/op"         # Currently ~3ns

  important_operations:
    template_transform: ">100μs/op"        # Currently ~37μs
    branch_parsing: ">5μs/op"             # Currently ~1.2μs
    git_command_overhead: ">100μs/op"     # Currently ~23μs

  batch_operations:
    directory_sync_1000_files: ">100ms"   # Currently ~32ms
    worker_pool_1000_tasks: ">200ms"      # Currently ~100ms
    api_batch_50_calls: ">2s"             # Network dependent

  memory_usage:
    directory_sync: ">2KB/file"           # Currently ~1.2KB
    config_processing: ">100KB"           # Currently ~50KB
    worker_pool_overhead: ">20KB"         # Currently ~10KB
```

---

## Troubleshooting Performance Issues

Common performance issues and their solutions based on real-world deployment experience.

### Common Performance Problems

#### High CPU Usage

**Symptoms:**
- CPU usage >80% during sync operations
- Slow response times
- High system load

**Diagnosis:**
```bash
# Enable CPU profiling
suite := profiling.NewProfileSuite("./profiles")
suite.Configure(profiling.ProfileConfig{
    EnableCPU: true,
})

suite.StartProfiling("cpu_investigation")
runHighCPUWorkload()
suite.StopProfiling()

# Analyze results
go tool pprof -top cpu.prof
# Look for functions consuming >10% CPU
```

**Common Solutions:**
- Reduce worker pool size: `numWorkers := runtime.NumCPU() / 2`
- Enable exclusion patterns to skip unnecessary files
- Use batch processing to reduce overhead
- Check for inefficient regex patterns in exclusions

#### Memory Leaks

**Symptoms:**
- Memory usage grows over time
- Eventually causes OOM errors
- GC runs frequently but memory isn't freed

**Diagnosis:**
```bash
# Memory profiling over time
suite.Configure(profiling.ProfileConfig{
    EnableMemory: true,
    GenerateReports: true,
})

suite.StartProfiling("memory_leak_check")
runForMinutes(10)
suite.StopProfiling()

# Check comprehensive_report.txt for growth rate
# Look for unexpected memory growth patterns
```

**Common Solutions:**
- Check for goroutine leaks: `go tool pprof goroutine.prof`
- Ensure proper cleanup of resources (files, connections)
- Review cache TTL settings for unbounded growth
- Verify worker pools are properly stopped

#### API Rate Limiting

**Symptoms:**
- "rate limit exceeded" errors
- Slow sync operations
- API calls failing intermittently

**Diagnosis:**
```bash
# Enable debug logging to see API usage
go-broadcast sync --log-level debug --config config.yaml 2>&1 | \
  grep -E "(api_calls|rate_limit|github_api)"
```

**Solutions:**
- Use directory sync instead of individual file mappings
- Enable API caching with appropriate TTLs
- Implement exponential backoff for retries
- Consider using GitHub Apps for higher rate limits

#### Slow Directory Sync

**Symptoms:**
- Directory sync takes >1 second for <500 files
- Progress reports show slow file processing
- High latency in API calls

**Diagnosis:**
```bash
# Profile directory sync specifically
go test -bench=BenchmarkDirectoryWalk_500Files -benchmem ./internal/sync/

# Check for network latency issues
ping api.github.com

# Verify exclusion patterns aren't overly complex
go test -bench=BenchmarkExclusionEngine -benchmem ./internal/sync/
```

**Solutions:**
- Optimize exclusion patterns (put fastest patterns first)
- Reduce worker count for I/O bound operations
- Use connection pooling for API calls
- Consider geographic proximity to GitHub's API

### Performance Debugging Workflow

#### 1. Identify the Bottleneck

```bash
# Run comprehensive profiling
go build -o profile_demo ./cmd/profile_demo
./profile_demo

# Check the comprehensive report
cat ./profiles/final_demo/comprehensive_report.txt

# Look for:
# - High CPU usage functions
# - Memory growth patterns
# - Goroutine count trends
# - API call frequency
```

#### 2. Isolate the Problem

```bash
# Test individual components
go test -bench=BenchmarkGitCommand ./internal/git         # Git performance
go test -bench=BenchmarkGHCommand ./internal/gh          # API performance
go test -bench=BenchmarkDirectory ./internal/sync        # Directory sync
go test -bench=BenchmarkWorkerPool ./internal/worker     # Concurrency

# Compare against baselines
benchstat baseline.txt current.txt
```

#### 3. Apply Targeted Optimizations

Based on the bottleneck identified:

**CPU Bound Issues:**
- Reduce algorithmic complexity
- Use more efficient data structures
- Enable compiler optimizations
- Profile and optimize hot paths

**Memory Bound Issues:**
- Reduce allocations in hot paths
- Use object pools for frequent allocations
- Implement streaming for large data
- Review GC settings (GOGC, GOMEMLIMIT)

**I/O Bound Issues:**
- Increase concurrency for independent operations
- Use connection pooling
- Implement caching strategies
- Batch operations to reduce overhead

#### 4. Validate Improvements

```bash
# Run benchmarks again
go test -bench=. -benchmem ./internal/... > optimized.txt
benchstat baseline.txt optimized.txt

# Look for:
# - Reduced execution time (ns/op)
# - Lower memory usage (B/op)
# - Fewer allocations (allocs/op)
# - No performance regressions in other areas
```

### Performance Monitoring in Production

```go
// Add performance metrics
var (
    syncDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "sync_duration_seconds",
            Help: "Duration of sync operations",
            Buckets: prometheus.ExponentialBuckets(0.001, 2, 15),
        },
        []string{"operation", "repository_count"},
    )

    memoryUsage = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "memory_usage_bytes",
            Help: "Current memory usage",
        },
        []string{"component"},
    )
)

// Instrument critical operations
func instrumentedSync(config *Config) error {
    start := time.Now()
    defer func() {
        syncDuration.WithLabelValues(
            "full_sync",
            strconv.Itoa(len(config.Repositories)),
        ).Observe(time.Since(start).Seconds())
    }()

    return performSync(config)
}
```

### Performance Checklist

Before deploying or after performance issues:

- [ ] **Benchmarked** all critical paths
- [ ] **Profiled** CPU and memory usage under load
- [ ] **Optimized** hot paths for zero allocations
- [ ] **Configured** appropriate worker pool sizes
- [ ] **Implemented** proper resource cleanup
- [ ] **Validated** no memory leaks over extended runs
- [ ] **Tested** API rate limit handling
- [ ] **Monitored** goroutine count growth
- [ ] **Documented** performance characteristics
- [ ] **Added** performance regression tests

### Common Anti-Patterns

1. **Creating too many goroutines**
   ```go
   // Bad: Unbounded goroutines
   for _, item := range items {
       go processItem(item)  // Could create thousands of goroutines
   }

   // Good: Use worker pool
   pool := worker.NewPool(runtime.NumCPU(), 100)
   for _, item := range items {
       pool.Submit(&ProcessItemTask{item})
   }
   ```

2. **Inefficient string operations**
   ```go
   // Bad: Repeated string concatenation
   result := ""
   for _, part := range parts {
       result += part  // Creates new string each time
   }

   // Good: Use strings.Builder
   var builder strings.Builder
   builder.Grow(estimatedSize)
   for _, part := range parts {
       builder.WriteString(part)
   }
   result := builder.String()
   ```

3. **Ignoring allocation patterns**
   ```go
   // Bad: Growing slice without capacity
   result := []string{}
   for _, item := range items {
       result = append(result, transform(item))
   }

   // Good: Pre-allocate with capacity
   result := make([]string, 0, len(items))
   for _, item := range items {
       result = append(result, transform(item))
   }
   ```

---

## Related Documentation

- [CLAUDE.md Developer Workflows](../.github/CLAUDE.md#-performance-testing-and-benchmarking) - Complete development workflow integration
- [Directory Sync Guide](directory-sync.md) - Complete directory sync feature documentation
- [Troubleshooting Guide](troubleshooting.md) - General troubleshooting and debugging
- [Configuration Guide](configuration-guide.md) - Performance-related configuration options

## Additional Resources

- [Go Blog: Profiling Go Programs](https://blog.golang.org/profiling-go-programs)
- [pprof Documentation](https://github.com/google/pprof/blob/master/doc/README.md)
- [Runtime Package](https://pkg.go.dev/runtime) - Low-level profiling APIs
- [Testing Package](https://pkg.go.dev/testing) - Benchmark helpers

---

**Performance Philosophy**: Measure first, optimize second. Not all code needs to be fast - focus on the parts that matter for your use case. go-broadcast prioritizes the critical paths that directly impact user experience while maintaining code clarity and maintainability.
