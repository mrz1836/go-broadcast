---
allowed-tools: Task, Bash(magex bench:*), Bash(go test -bench:*), Write, Read
description: Run performance benchmarks and detect regressions
argument-hint: [specific benchmark pattern or leave empty for all]
---
!magex bench 2>&1 | grep -E "Benchmark|ns/op|allocs/op" | tail -20

# 📊 Performance Benchmarking & Analysis

I need to run comprehensive performance benchmarks and analyze for regressions.

**Step 1: Use the benchmark-runner agent** to:

1. **Execute benchmark suite**:
   - Run all benchmarks with multiple iterations
   - Capture CPU and memory profiles
   - Generate benchmark results in standard format
   - Save results for comparison

2. **Focus on critical performance paths**:
   - Binary detection (target: 587M+ ops/sec)
   - Content comparison (target: 239M+ ops/sec)
   - Directory sync (target: <32ms for 1000 files)
   - Cache operations (target: 13.5M+ ops/sec)

**Step 2: Use the benchmark-analyst agent** to:

1. **Compare with baseline**:
   - Load previous benchmark results
   - Calculate performance deltas
   - Flag regressions >5%
   - Identify improvements

2. **Provide detailed analysis**:
   - Statistical significance of changes
   - Memory allocation patterns
   - CPU usage trends
   - Recommendations for optimization

3. **Generate reports**:
   - Performance trend visualization
   - Regression alerts with root cause
   - Optimization opportunities
   - Update benchmark documentation

## Benchmark Target: $ARGUMENTS

The agents will ensure performance standards are maintained and any regressions are immediately identified with actionable fixes.
