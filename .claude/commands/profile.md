---
allowed-tools: Task, Bash(go test -cpuprofile:*), Bash(go tool pprof:*), Write, Read
description: Performance profiling and optimization analysis
argument-hint: [cpu|memory|goroutine or specific operation]
---
# 🚀 Performance Profiling & Optimization

I need to profile and analyze performance for: ${ARGUMENTS:-all operations}

**Parallel profiling workflow:**

1. **Use the performance-profiler agent** to:
   - Generate CPU profiles for hot paths
   - Create memory allocation profiles
   - Analyze goroutine behavior
   - Profile specific operations
   - Generate pprof files for analysis
   - Identify bottlenecks

2. **Simultaneously, use the benchmark-analyst agent** to:
   - Run targeted benchmarks
   - Compare current vs baseline performance
   - Identify regression areas
   - Analyze allocation patterns
   - Track operation timings

**Profiling targets:**
- Binary detection operations (target: 587M+ ops/sec)
- Content comparison (target: 239M+ ops/sec)
- Directory sync performance
- Cache operations efficiency
- API call optimization
- Memory allocation hotspots

**Analysis deliverables:**
- CPU flame graphs
- Memory allocation reports
- Goroutine analysis
- Bottleneck identification
- Optimization recommendations
- Code refactoring suggestions

**Optimization strategies:**
- Algorithm improvements
- Concurrency enhancements
- Memory pooling opportunities
- Cache optimization
- I/O batching improvements

The agents will provide actionable insights with specific code changes to improve performance.
