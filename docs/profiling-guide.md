# Profiling Guide

go-broadcast includes a comprehensive profiling suite that enables deep performance analysis through CPU profiling, memory profiling, goroutine analysis, and more. This guide covers how to use these powerful tools.

## Table of Contents
- [Overview](#overview)
- [ProfileSuite API](#profilesuite-api)
- [Running the Profile Demo](#running-the-profile-demo)
- [Profiling Types](#profiling-types)
- [Analyzing Profiles](#analyzing-profiles)
- [Performance Reports](#performance-reports)
- [Best Practices](#best-practices)

## Overview

The profiling system in go-broadcast provides:
- **Comprehensive profiling** - CPU, memory, goroutine, block, and mutex profiling
- **Automated reports** - HTML, JSON, and Markdown performance reports
- **Memory tracking** - Detailed heap analysis and allocation tracking
- **Session management** - Track multiple profiling sessions
- **Zero-overhead** - Profiling only when explicitly enabled

## ProfileSuite API

### Basic Usage

```go
import "github.com/mrz1836/go-broadcast/internal/profiling"

// Create a new profiling suite
suite := profiling.NewProfileSuite("./profiles")

// Configure profiling options
config := profiling.ProfileConfig{
    EnableCPU:            true,
    EnableMemory:         true,
    EnableTrace:          false, // Can be CPU intensive
    EnableBlock:          true,
    EnableMutex:          true,
    BlockProfileRate:     1,      // 1 = profile all blocking events
    MutexProfileFraction: 1,      // 1 = profile all mutex events
    GenerateReports:      true,
    ReportFormat:         "text", // "text", "html", or "both"
}
suite.Configure(config)

// Start profiling
err := suite.StartProfiling("my_analysis_session")
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

### Configuration Options

| Option                 | Description           | Default | Performance Impact     |
|------------------------|-----------------------|---------|------------------------|
| `EnableCPU`            | CPU profiling         | false   | Medium                 |
| `EnableMemory`         | Heap profiling        | false   | Low                    |
| `EnableTrace`          | Execution trace       | false   | High                   |
| `EnableBlock`          | Blocking profiling    | false   | Medium                 |
| `EnableMutex`          | Mutex contention      | false   | Medium                 |
| `BlockProfileRate`     | Block sampling rate   | 1       | Higher = more overhead |
| `MutexProfileFraction` | Mutex sampling rate   | 1       | Higher = more overhead |
| `GenerateReports`      | Auto-generate reports | true    | Low (post-execution)   |
| `ReportFormat`         | Output format         | "text"  | None                   |

## Running the Profile Demo

go-broadcast includes a comprehensive profiling demo that showcases the profiling capabilities:

```bash
# Build the profile demo
go build -o profile_demo ./cmd/profile_demo

# Run the demo
./profile_demo

# Output will be in ./profiles/final_demo/
```

The demo profiles:
- Worker pool operations
- TTL cache performance
- Algorithm optimizations
- Batch processing

### Demo Output

The demo generates:
- `cpu.prof` - CPU profile (if enabled)
- `memory.prof` - Memory profile
- `goroutine.prof` - Goroutine snapshot
- `block.prof` - Blocking profile (if enabled)
- `mutex.prof` - Mutex contention (if enabled)
- `comprehensive_report.txt` - Detailed analysis
- `performance_report_*.html` - Visual report
- `performance_report_*.json` - Machine-readable data

## Profiling Types

### 1. CPU Profiling

Identifies where CPU time is spent:

```go
// Enable CPU profiling
config.EnableCPU = true

// After profiling, analyze with:
// go tool pprof cpu.prof
```

Common commands:
```bash
# Interactive mode
go tool pprof cpu.prof

# Top 10 functions by CPU time
go tool pprof -top cpu.prof

# Generate flame graph
go tool pprof -http=:8080 cpu.prof

# Focus on specific function
go tool pprof -focus=worker cpu.prof
```

### 2. Memory Profiling

Tracks heap allocations and memory usage:

```go
// Enable memory profiling
config.EnableMemory = true

// The suite provides detailed memory analysis
type MemorySnapshot struct {
    Timestamp    time.Time
    AllocBytes   uint64  // Currently allocated
    TotalAlloc   uint64  // Total allocated (cumulative)
    NumGC        uint32  // Number of GC cycles
    NumGoroutine int     // Active goroutines
}
```

Analysis commands:
```bash
# Heap allocations
go tool pprof -alloc_space memory.prof

# Live objects
go tool pprof -inuse_space memory.prof

# Allocation sources
go tool pprof -source memory.prof
```

### 3. Goroutine Profiling

Captures goroutine stacks and states:

```go
// Automatically captured at session end
// Useful for detecting goroutine leaks
```

View goroutines:
```bash
# Text output
go tool pprof goroutine.prof

# Web visualization
go tool pprof -http=:8080 goroutine.prof
```

### 4. Block Profiling

Identifies blocking operations:

```go
// Enable block profiling
config.EnableBlock = true
config.BlockProfileRate = 1  // Profile all events

// Captures:
// - Channel operations
// - Mutex waits
// - Select statements
```

### 5. Mutex Profiling

Tracks mutex contention:

```go
// Enable mutex profiling
config.EnableMutex = true
config.MutexProfileFraction = 1  // Profile all contentions

// Shows which mutexes have high contention
```

## Analyzing Profiles

### Using pprof

Basic pprof commands:

```bash
# Interactive mode
go tool pprof [profile_file]

# Common interactive commands:
(pprof) top        # Show top functions
(pprof) list func  # Show source for function
(pprof) web        # Open in browser
(pprof) peek func  # Show callers/callees
(pprof) tree       # Show call tree
```

### Web Interface

The most powerful way to analyze profiles:

```bash
# Start web server
go tool pprof -http=:8080 cpu.prof

# Features:
# - Flame graphs
# - Source view
# - Peek view
# - Diff view (compare profiles)
```

### Comparing Profiles

Compare before/after optimization:

```bash
# Generate baseline
./myapp -profile=baseline.prof

# Make changes, generate new profile
./myapp -profile=optimized.prof

# Compare
go tool pprof -base=baseline.prof optimized.prof
```

### Memory Leak Detection

```go
// Use the MemoryProfiler for detailed tracking
profiler := profiling.NewMemoryProfiler("./profiles")

// Start session
session, _ := profiler.StartSession("memory_check")

// Periodic snapshots
for i := 0; i < 10; i++ {
    profiler.TakeSnapshot(fmt.Sprintf("iteration_%d", i))
    time.Sleep(time.Second)
}

// Analyze growth
analysis := profiler.AnalyzeSession(session)
if analysis.MemoryGrowthRate > 1.0 {
    log.Printf("Potential leak: %.2f MB/sec growth", 
        analysis.MemoryGrowthRate)
}
```

## Performance Reports

The ProfileSuite can generate comprehensive reports:

### Report Types

1. **Text Report** (`comprehensive_report.txt`)
   ```
   Comprehensive Profiling Report
   ==============================
   Start Time: 2024-01-15T10:30:00Z
   Duration: 5m30s
   
   Memory Analysis
   ---------------
   Initial: 10.5 MB
   Peak: 125.3 MB
   Final: 15.2 MB
   Growth Rate: 0.02 MB/s
   
   Top Allocations:
   1. worker.(*Pool).Submit - 45.2 MB
   2. cache.(*TTLCache).Set - 23.1 MB
   ...
   ```

2. **HTML Report** (`performance_report_*.html`)
   - Visual charts and graphs
   - Interactive memory timeline
   - Sortable performance metrics
   - Goroutine analysis

3. **JSON Report** (`performance_report_*.json`)
   ```json
   {
     "timestamp": "2024-01-15T10:30:00Z",
     "duration_ms": 330000,
     "memory_stats": {
       "initial_mb": 10.5,
       "peak_mb": 125.3,
       "final_mb": 15.2
     },
     "profile_summary": {
       "cpu_profile": {
         "available": true,
         "size_bytes": 1048576,
         "path": "profiles/cpu.prof"
       }
     }
   }
   ```

### Custom Report Generation

```go
import "github.com/mrz1836/go-broadcast/internal/reporting"

// Create reporter
config := reporting.DefaultReportConfig()
config.OutputDirectory = "./reports"
config.GenerateHTML = true
config.GenerateJSON = true
reporter := reporting.NewPerformanceReporter(config)

// Generate report
report, _ := reporter.GenerateReport(
    metrics,
    testResults,
    profileSummary,
)

// Save report
reporter.SaveReport(report)
```

## Best Practices

### 1. Profile in Production-Like Environment
- Use realistic data sizes
- Simulate actual concurrency levels
- Profile with typical system load

### 2. Focus on Hot Paths
- Profile the critical user paths first
- Don't optimize prematurely
- Measure impact of changes

### 3. Regular Profiling
- Add profiling to CI/CD pipeline
- Compare profiles across releases
- Track performance trends

### 4. Memory Profiling Tips
- Take multiple snapshots over time
- Look for unexpected growth patterns
- Check both allocations and live objects

### 5. CPU Profiling Tips
- Profile for at least 30 seconds
- Ensure representative workload
- Check both on-CPU and off-CPU time

### 6. Goroutine Health
- Monitor goroutine count trends
- Look for stuck goroutines
- Check for proper cleanup

## Common Scenarios

### Debugging High CPU Usage

```go
// 1. Enable CPU profiling
suite.Configure(profiling.ProfileConfig{
    EnableCPU: true,
})

// 2. Run workload
suite.StartProfiling("cpu_investigation")
runHighCPUWorkload()
suite.StopProfiling()

// 3. Analyze
// go tool pprof -top cpu.prof
// Look for functions consuming >10% CPU
```

### Finding Memory Leaks

```go
// 1. Enable memory profiling
suite.Configure(profiling.ProfileConfig{
    EnableMemory: true,
    GenerateReports: true,
})

// 2. Run for extended period
suite.StartProfiling("memory_leak_check")
runForMinutes(10)
suite.StopProfiling()

// 3. Check report for growth rate
// Look at comprehensive_report.txt
```

### Analyzing Concurrency Issues

```go
// 1. Enable block and mutex profiling
suite.Configure(profiling.ProfileConfig{
    EnableBlock: true,
    EnableMutex: true,
    BlockProfileRate: 1,
    MutexProfileFraction: 1,
})

// 2. Run concurrent workload
suite.StartProfiling("concurrency_check")
runConcurrentOperations()
suite.StopProfiling()

// 3. Check for high contention
// go tool pprof block.prof
// go tool pprof mutex.prof
```

## Integration Example

Here's how to add profiling to your application:

```go
package main

import (
    "flag"
    "log"
    "github.com/mrz1836/go-broadcast/internal/profiling"
)

var (
    profileDir = flag.String("profile-dir", "", "Enable profiling to directory")
    cpuProfile = flag.Bool("cpu-profile", false, "Enable CPU profiling")
    memProfile = flag.Bool("mem-profile", false, "Enable memory profiling")
)

func main() {
    flag.Parse()
    
    // Setup profiling if requested
    if *profileDir != "" {
        suite := profiling.NewProfileSuite(*profileDir)
        suite.Configure(profiling.ProfileConfig{
            EnableCPU:    *cpuProfile,
            EnableMemory: *memProfile,
            GenerateReports: true,
        })
        
        if err := suite.StartProfiling("app_profile"); err != nil {
            log.Fatal(err)
        }
        defer suite.StopProfiling()
    }
    
    // Your application logic here
    runApplication()
}
```

## Developer Workflow Integration

For comprehensive go-broadcast development workflows, see [CLAUDE.md](../.github/CLAUDE.md#-performance-testing-and-benchmarking) which includes:
- **Performance profiling procedures** for systematic analysis
- **Benchmark execution workflows** with memory profiling
- **Profile analysis commands** for optimization
- **Integration with development workflow** for continuous performance monitoring

## Related Documentation

1. [Benchmarking Guide](benchmarking-profiling.md) - Writing and running benchmarks
2. [Performance Optimization](performance-optimization.md) - Optimization strategies based on profiling results
3. [Troubleshooting Guide](troubleshooting.md) - Common performance issues and debugging
4. [CLAUDE.md Developer Workflows](../.github/CLAUDE.md) - Complete development workflow integration

## Additional Resources

- [Go Blog: Profiling Go Programs](https://blog.golang.org/profiling-go-programs)
- [pprof Documentation](https://github.com/google/pprof/blob/master/doc/README.md)
- [Runtime Package](https://pkg.go.dev/runtime) - Low-level profiling APIs
- [Testing Package](https://pkg.go.dev/testing) - Benchmark helpers
