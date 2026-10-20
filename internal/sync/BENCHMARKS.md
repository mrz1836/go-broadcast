# Directory Sync Benchmarks

This document describes the comprehensive benchmarking suite for the directory sync functionality in go-broadcast. The benchmarks are designed to measure performance, memory usage, and API efficiency across various scenarios.

## Overview

The benchmarks are located in `internal/sync/benchmark_test.go` (behind the `bench_heavy` build tag) and provide comprehensive performance testing for:

1. **Change Detection**: Settling target files from one Git tree request
2. **Memory Profiling**: Memory allocation patterns and usage
3. **Cache Performance**: Hit rates and memory efficiency
4. **Real-World Scenarios**: Using actual test fixtures
5. **Performance Regression Detection**: Baseline metrics for CI/CD

## Benchmark Categories

### 1. Directory Walking (`BenchmarkDirectoryWalk`)

Tests basic directory traversal performance across different directory sizes:
- SmallDirectory_10 (10 files)
- MediumDirectory_50 (50 files)
- LargeDirectory_100 (100 files)
- XLargeDirectory_500 (500 files)
- XXLargeDirectory_1000 (1000 files)

**Key Metrics**: Files processed per second, memory allocations

### 2. Exclusion Engine (`BenchmarkExclusionEngine`)

Tests pattern matching performance for file exclusions using common patterns like `*.log`, `node_modules/**`, etc.

**Key Metrics**: Pattern matches per second, memory per operation

### 3. Target Tree Change Detection (`BenchmarkTargetTreeChangeDetection`)

Settles a target whose mapped files are all unchanged, as a sync does before deciding
whether a target needs a PR: one recursive tree request, then a Git blob-ID comparison
per file. Runs with 10, 100 and 1000 files.

**Key Metrics**:
- `tree-requests/op`: Git Trees API requests per target (expected: 1)
- `file-requests/op`: Contents API requests per target (expected: 0 when nothing changed)

### 4. Cache Hit Rates (`BenchmarkCacheHitRates`)

Tests caching performance with different access patterns:
- **Sequential**: Linear access pattern
- **Random**: Random access pattern
- **Hotspot**: 80/20 access pattern (80% requests to 20% of files)

**Key Metrics**:
- `cache-hit-rate-%`: Percentage of cache hits
- `cache-size`: Number of cached entries
- `memory-usage-bytes`: Memory consumed by cache

### 5. Memory Allocation Patterns (`BenchmarkMemoryAllocationPatterns`)

Tests memory usage across different directory structures:
- Flat vs Deep directory structures
- Various file counts (50, 200, 1000)
- Directory depths (1 level to 10 levels)

**Key Metrics**:
- `bytes-alloc-per-op`: Memory allocated per operation
- `mallocs-per-op`: Number of allocations per operation

### 6. Concurrent Directory Processing (`BenchmarkConcurrentDirectoryProcessing`)

Tests memory usage during concurrent directory processing:
- Multiple directories processed simultaneously
- Various worker counts and file distributions

**Key Metrics**:
- `bytes-alloc-per-op`: Memory per operation
- `peak-memory-bytes`: Peak memory usage
- `worker-count`: Number of concurrent workers

### 7. Exclusion Pattern Memory (`BenchmarkExclusionPatternMemory`)

Tests memory efficiency of exclusion pattern matching:
- 10-100 patterns with 100-1000 test paths
- Complex nested patterns

**Key Metrics**:
- `bytes-alloc-per-op`: Memory per pattern match
- `pattern-count`: Number of exclusion patterns
- `test-path-count`: Number of paths tested

### 8. Real-World Scenarios (`BenchmarkRealWorldScenarios`)

Uses actual test fixtures to simulate realistic usage:
- **GitHubWorkflows**: `.github` directory structure
- **ComplexStructure**: Complex nested directories with special characters
- **LargeRepository**: Large repository simulation
- **MixedContent**: Mixed file types and binary content

**Key Metrics**:
- `files-discovered`: Number of files found and processed

### 9. Performance Regression (`BenchmarkPerformanceRegression`)

Establishes baseline metrics for performance regression detection:

#### BaselineDirectoryProcessing
- 500 files across 5 directory levels
- 10 concurrent workers
- 20 exclusion patterns

**Key Metrics**:
- `baseline-files-processed`: Files successfully processed
- `baseline-total-files`: Total files in test set
- `baseline-worker-count`: Worker configuration
- `baseline-processing-efficiency-%`: Processing success rate

#### BaselineCacheEffectiveness
Tests cache performance with realistic access patterns:

**Key Metrics**:
- `baseline-cache-hit-rate-%`: Cache hit percentage
- `baseline-cache-size`: Number of cached entries
- `baseline-memory-usage-bytes`: Cache memory usage

## Running Benchmarks

### Quick Start
```bash
# Run all benchmarks
go test -tags bench_heavy -run '^$' -bench=. ./internal/sync

# Run specific benchmark
go test -tags bench_heavy -run '^$' -bench=BenchmarkTargetTreeChangeDetection ./internal/sync

# Run with memory profiling
go test -tags bench_heavy -run '^$' -bench=. -benchmem ./internal/sync
```

### Comprehensive Benchmark Suite
```bash
# Build and use the benchmark tool
go build ./cmd/run-benchmarks
./run-benchmarks

# Or run specific category
./run-benchmarks BenchmarkCacheHitRates
```

### Memory and CPU Profiling
```bash
# Generate profiles
go test -tags bench_heavy -run '^$' -bench=. -benchmem \
  -memprofile=mem.prof \
  -cpuprofile=cpu.prof \
  ./internal/sync

# Analyze profiles
go tool pprof mem.prof
go tool pprof cpu.prof
```

## Interpreting Results

### Sample Output
```
BenchmarkTargetTreeChangeDetection/Files_100-12     200    27469 ns/op    0 file-requests/op    1.000 tree-requests/op    38926 B/op    515 allocs/op
BenchmarkTargetTreeChangeDetection/Files_1000-12    200   275682 ns/op    0 file-requests/op    1.000 tree-requests/op   470758 B/op   5018 allocs/op
```

### Key Metrics Explained
- **ns/op**: Nanoseconds per operation (lower is better)
- **tree-requests/op**: Git Trees API requests per target
- **file-requests/op**: Contents API requests per target
- **cache-hit-rate-%**: Percentage of successful cache lookups
- **bytes-alloc-per-op**: Memory allocated per benchmark operation
- **mallocs-per-op**: Number of memory allocations per operation

### Performance Targets
Based on the Phase 5 success criteria:

- **Change Detection**: One tree request and no file requests for an unchanged target
- **Cache Hit Rate**: Target 60%+ for realistic access patterns
- **Memory Efficiency**: Linear scaling with directory size
- **Concurrent Performance**: No significant degradation up to 20 workers

## CI/CD Integration

The benchmarks are designed for continuous performance monitoring:

### Performance Regression Detection
```bash
# Compare current vs previous results
benchstat old_results.txt new_results.txt

# Check for regressions (>10% performance loss)
if [[ $(benchstat -format=csv old.txt new.txt | grep -c "regression") -gt 0 ]]; then
  echo "Performance regression detected"
  exit 1
fi
```

### Automated Benchmark Runs
The benchmarks can be integrated into CI/CD pipelines to:
1. Detect performance regressions
2. Track memory usage trends
3. Validate API efficiency improvements
4. Monitor cache effectiveness

## Test Fixtures

The benchmarks use test fixtures from `test/fixtures/directories/`:
- **small/**: Basic directory structure
- **medium/**: Moderate complexity (20+ files)
- **large/**: Large repository simulation (100+ files)
- **complex/**: Special characters, unicode, edge cases
- **mixed/**: Binary files, executables, archives
- **github/**: Real GitHub workflow directory structure

## Troubleshooting

### Common Issues

1. **Fixture Path Not Found**
   ```
   Fixture path /path/to/fixtures does not exist
   ```
   Ensure test fixtures are generated

2. **Memory Profile Too Large**
   Reduce benchmark iterations or add `-benchtime=100ms`

3. **Inconsistent Results**
   Run multiple times with `-count=5` for statistical significance

### Debugging Performance Issues

1. **High Memory Usage**
   ```bash
   go test -tags bench_heavy -run '^$' -bench=BenchmarkMemory -memprofile=mem.prof ./internal/sync
   go tool pprof -top mem.prof
   ```

2. **Cache Misses**
   Verify cache size configuration and access patterns

## Contributing

When adding new benchmarks:

1. Follow naming convention: `BenchmarkFeatureName`
2. Include relevant metrics using `b.ReportMetric()`
3. Test with various input sizes
4. Add documentation to this file
5. Consider real-world usage patterns
6. Ensure benchmarks are deterministic and repeatable
