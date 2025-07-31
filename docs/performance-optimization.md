# Performance Optimization Guide

This guide provides performance optimization strategies based on go-broadcast's extensive benchmarking results. Learn how to maximize performance across different components and operations.

## Table of Contents
- [Performance Overview](#performance-overview)
- [Component-Specific Optimizations](#component-specific-optimizations)
- [General Optimization Strategies](#general-optimization-strategies)
- [Real-World Performance Tips](#real-world-performance-tips)
- [Performance Checklist](#performance-checklist)

## Performance Overview

Based on comprehensive benchmarks, go-broadcast achieves:

| Component            | Performance     | Key Metric          |
|----------------------|-----------------|---------------------|
| Binary Detection     | 77M+ ops/sec    | Zero allocations    |
| State Comparison     | 26M+ ops/sec    | Zero allocations    |
| Config Validation    | 371M+ ops/sec   | Zero allocations    |
| Git Operations       | ~40K ops/sec    | Varies by operation |
| Transform Operations | ~31K files/sec  | For small files     |
| Cache Operations     | Sub-microsecond | For hits            |
| Worker Pool          | ~10K tasks/sec  | With 8 workers      |

## Component-Specific Optimizations

### 1. Git Operations

**Current Performance:**
- Simple commands: ~23μs overhead
- Clone operations: Varies by repo size
- Concurrent operations: Linear scaling to ~8 cores

**Optimization Strategies:**

```go
// 1. Use shallow clones for large repos
client.CloneOptions{
    Depth: 1,  // Only fetch latest commit
}

// 2. Batch operations when possible
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

// 3. Reuse client instances
// Don't: create new client for each operation
// Do: reuse client across operations
```

**Tips:**
- Use `--depth=1` for clones when history isn't needed
- Batch small operations into single commits
- Leverage concurrent operations for independent tasks
- Cache repository state to avoid repeated discoveries

### 2. GitHub API Operations

**Current Performance:**
- Simple API calls: ~100ms (network dependent)
- JSON parsing: ~50μs for small responses
- Concurrent calls: Up to 20 parallel requests

**Optimization Strategies:**

```go
// 1. Implement request caching
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

// 2. Use conditional requests
client.SetHeader("If-None-Match", etag)

// 3. Batch API requests
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

**Tips:**
- Cache API responses with appropriate TTLs
- Use ETags for conditional requests
- Implement exponential backoff for rate limits
- Batch related API calls
- Use GraphQL for complex queries to reduce round trips

### 3. File Transformations

**Current Performance:**
- Template transforms: ~37μs for small files
- Binary detection: 33ns for text, 16ns for binary
- Chain transforms: ~28μs for multiple operations

**Optimization Strategies:**

```go
// 1. Pool regex compilations
var (
    regexPool = sync.Pool{
        New: func() interface{} {
            return regexp.MustCompile(`\{\{(\w+)\}\}`)
        },
    }
)

// 2. Optimize binary detection
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

// 3. Use strings.Builder for concatenation
var builder strings.Builder
builder.Grow(estimatedSize) // Pre-allocate
for _, part := range parts {
    builder.WriteString(part)
}
result := builder.String()
```

**Tips:**
- Cache compiled regular expressions
- Use `strings.Builder` for string concatenation
- Pre-allocate buffers when size is known
- Process files in chunks for large transformations
- Skip binary files early in the pipeline

### 4. State Management

**Current Performance:**
- Branch parsing: ~1.2μs per parse
- State comparison: 8-9ns per comparison
- Zero allocations for core operations

**Optimization Strategies:**

```go
// 1. Pre-compile regex patterns
var (
    branchPattern = regexp.MustCompile(`^sync/(\w+)-(\d{8}-\d{6})-([a-f0-9]{7})$`)
)

// 2. Use efficient string operations
func needsSync(source, target string) bool {
    // Direct comparison instead of parsing
    return source != target
}

// 3. Cache parsed states
type StateCache struct {
    mu    sync.RWMutex
    cache map[string]*State
}

func (c *StateCache) Get(branch string) (*State, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    state, ok := c.cache[branch]
    return state, ok
}
```

**Tips:**
- Cache parsed branch metadata
- Use direct string comparisons when possible
- Implement efficient sorting for state aggregation
- Minimize allocations in hot paths

### 5. Worker Pool Optimization

**Current Performance:**
- ~10K tasks/sec with 8 workers
- Linear scaling up to CPU count
- Minimal overhead per task

**Optimization Strategies:**

```go
// 1. Right-size worker count
numWorkers := runtime.NumCPU()
if numWorkers > 16 {
    numWorkers = 16 // Diminishing returns above 16
}

// 2. Use buffered channels appropriately
queueSize := numWorkers * 100 // Prevent blocking
pool := worker.NewPool(numWorkers, queueSize)

// 3. Batch small tasks
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

**Tips:**
- Set worker count to CPU count for CPU-bound tasks
- Use 2-3x CPU count for I/O-bound tasks
- Buffer channels to prevent contention
- Batch small tasks to reduce overhead
- Monitor queue depth to detect bottlenecks

### 6. Cache Optimization

**Current Performance:**
- Basic operations: <1μs
- High concurrency: Linear scaling
- TTL overhead: Minimal

**Optimization Strategies:**

```go
// 1. Size cache appropriately
cache := cache.NewTTLCache(
    5*time.Minute,  // TTL
    10000,          // Max size based on working set
)

// 2. Use cache warming
func warmCache(cache *cache.TTLCache, keys []string) {
    for _, key := range keys {
        value := fetchExpensiveData(key)
        cache.Set(key, value)
    }
}

// 3. Implement cache layers
type LayeredCache struct {
    l1 *cache.TTLCache  // Hot data, short TTL
    l2 *cache.TTLCache  // Warm data, longer TTL
}
```

**Tips:**
- Size cache based on working set, not total data
- Use shorter TTLs for frequently changing data
- Implement cache warming for predictable access patterns
- Monitor hit rates and adjust size/TTL accordingly
- Consider LRU eviction for memory-constrained environments

## General Optimization Strategies

### 1. Reduce Allocations

```go
// Bad: Creates new slice each time
func processItems(items []string) []string {
    result := []string{}
    for _, item := range items {
        if isValid(item) {
            result = append(result, transform(item))
        }
    }
    return result
}

// Good: Pre-allocate with capacity
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

### 2. Use Sync.Pool for Temporary Objects

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
    
    // Use buffer
    buf.Write(data)
    return buf.String()
}
```

### 3. Optimize Hot Paths

```go
// Identify hot paths with profiling
// go tool pprof -top cpu.prof

// Optimize the top functions:
// 1. Inline small functions
// 2. Remove unnecessary checks
// 3. Use fast paths for common cases

func optimizedFunction(data []byte) bool {
    // Fast path for common case
    if len(data) == 0 {
        return false
    }
    
    // Avoid function calls in loops
    dataLen := len(data)
    for i := 0; i < dataLen; i++ {
        // Direct access instead of bounds checking
        if data[i] == targetByte {
            return true
        }
    }
    return false
}
```

### 4. Efficient Concurrency

```go
// Use worker pools for bounded concurrency
pool := worker.NewPool(runtime.NumCPU(), 1000)

// Use channels for coordination
done := make(chan struct{})
results := make(chan Result, 100) // Buffer to prevent blocking

// Batch operations to reduce overhead
const batchSize = 100
for i := 0; i < len(items); i += batchSize {
    end := i + batchSize
    if end > len(items) {
        end = len(items)
    }
    batch := items[i:end]
    
    pool.Submit(&BatchTask{items: batch})
}
```

## Real-World Performance Tips

### 1. Network Operations
- Implement connection pooling
- Use HTTP/2 for multiplexing
- Enable TCP keepalive
- Implement circuit breakers

### 2. File I/O
- Buffer reads and writes
- Use memory-mapped files for large data
- Process files concurrently
- Implement streaming for large files

### 3. JSON Processing
- Use json.RawMessage for partial parsing
- Consider alternatives like msgpack for internal APIs
- Stream large JSON responses
- Pre-allocate structures

### 4. Database Operations
- Use prepared statements
- Implement connection pooling
- Batch inserts and updates
- Use appropriate indexes

## Performance Checklist

Before deploying:

- [ ] **Benchmarked** critical paths
- [ ] **Profiled** CPU and memory usage
- [ ] **Optimized** allocations in hot paths
- [ ] **Sized** caches appropriately
- [ ] **Configured** appropriate concurrency limits
- [ ] **Tested** under expected load
- [ ] **Monitored** goroutine growth
- [ ] **Validated** no memory leaks
- [ ] **Documented** performance characteristics
- [ ] **Implemented** graceful degradation

## Monitoring in Production

```go
// Add metrics for key operations
var (
    syncDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "sync_duration_seconds",
            Help: "Duration of sync operations",
        },
        []string{"operation"},
    )
)

// Instrument critical paths
start := time.Now()
err := performSync()
syncDuration.WithLabelValues("full_sync").Observe(
    time.Since(start).Seconds(),
)
```

## Common Pitfalls

1. **Premature Optimization**
   - Profile first, optimize second
   - Focus on actual bottlenecks

2. **Over-Engineering**
   - Simple solutions often perform better
   - Complexity has a cost

3. **Ignoring GC Pressure**
   - Monitor allocation rate
   - Use GOGC and GOMEMLIMIT wisely

4. **Unbounded Concurrency**
   - Always limit goroutines
   - Use worker pools or semaphores

5. **Cache Invalidation**
   - Implement proper TTLs
   - Monitor cache effectiveness

## Developer Workflow Integration

For comprehensive go-broadcast development workflows, see [CLAUDE.md](../.github/CLAUDE.md#-performance-testing-and-benchmarking) which includes:
- **Performance testing procedures** for optimization validation  
- **Benchmark execution workflows** for measuring improvements
- **Development workflow integration** for performance-conscious development
- **Quality assurance commands** to maintain performance standards

## Related Documentation

1. [Benchmarking Guide](benchmarking-profiling.md) - Run your own benchmarks and measure performance
2. [Profiling Guide](profiling-guide.md) - Deep performance analysis and debugging
3. [Troubleshooting Guide](troubleshooting.md) - Debug performance issues and bottlenecks
4. [CLAUDE.md Developer Workflows](../.github/CLAUDE.md) - Complete development workflow integration

**Remember:** The best optimization is often better algorithms and data structures. Always measure the impact of your changes!
