# Directory Sync Benchmarks

This document describes the comprehensive benchmarking suite for the directory sync functionality in go-broadcast. The benchmarks are designed to measure performance, memory usage, and API efficiency across various scenarios.

## Overview

The benchmarks are located in `internal/sync/benchmark_test.go` and provide comprehensive performance testing for:

1. **API Call Efficiency**: GitHub API vs Tree API performance
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

### 3. API Efficiency (`BenchmarkAPIEfficiency`)

Compares GitHub Tree API vs individual file API calls:
- TreeAPI vs IndividualAPI for different repository sizes
- Includes failure simulation (5% failure rate)
- Tests with various API delays (10-30ms)

**Key Metrics**:
- `tree-api-calls`: Number of tree API calls
- `content-api-calls`: Number of individual file calls
- `total-api-calls`: Combined API calls

### 4. Cache Hit Rates (`BenchmarkCacheHitRates`)

Tests caching performance with different access patterns:
- **Sequential**: Linear access pattern
- **Random**: Random access pattern
- **Hotspot**: 80/20 access pattern (80% requests to 20% of files)

**Key Metrics**:
- `cache-hit-rate-%`: Percentage of cache hits
- `cache-size`: Number of cached entries
- `memory-usage-bytes`: Memory consumed by cache

### 5. Concurrent API Requests (`BenchmarkConcurrentAPIRequests`)

Tests API performance under concurrent load:
- LowConcurrency: 5 workers, 10 requests each
- MediumConcurrency: 10 workers, 20 requests each
- HighConcurrency: 20 workers, 50 requests each
- ExtremeConcurrency: 50 workers, 100 requests each

**Key Metrics**:
- `total-requests`: Total API requests made
- `worker-count`: Number of concurrent workers

### 6. Memory Allocation Patterns (`BenchmarkMemoryAllocationPatterns`)

Tests memory usage across different directory structures:
- Flat vs Deep directory structures
- Various file counts (50, 200, 1000)
- Directory depths (1 level to 10 levels)

**Key Metrics**:
- `bytes-alloc-per-op`: Memory allocated per operation
- `mallocs-per-op`: Number of allocations per operation

### 7. Concurrent Directory Processing (`BenchmarkConcurrentDirectoryProcessing`)

Tests memory usage during concurrent directory processing:
- Multiple directories processed simultaneously
- Various worker counts and file distributions

**Key Metrics**:
- `bytes-alloc-per-op`: Memory per operation
- `peak-memory-bytes`: Peak memory usage
- `worker-count`: Number of concurrent workers

### 8. Exclusion Pattern Memory (`BenchmarkExclusionPatternMemory`)

Tests memory efficiency of exclusion pattern matching:
- 10-100 patterns with 100-1000 test paths
- Complex nested patterns

**Key Metrics**:
- `bytes-alloc-per-op`: Memory per pattern match
- `pattern-count`: Number of exclusion patterns
- `test-path-count`: Number of paths tested

### 9. Real-World Scenarios (`BenchmarkRealWorldScenarios`)

Uses actual test fixtures to simulate realistic usage:
- **GitHubWorkflows**: `.github` directory structure
- **ComplexStructure**: Complex nested directories with special characters
- **LargeRepository**: Large repository simulation
- **MixedContent**: Mixed file types and binary content

**Key Metrics**:
- `files-discovered`: Number of files found and processed

### 10. Performance Regression (`BenchmarkPerformanceRegression`)

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

#### BaselineAPICallReduction
Measures API call efficiency improvements:

**Key Metrics**:
- `api-call-reduction-%`: Percentage reduction in API calls
- `time-reduction-%`: Time savings from tree API

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
go test -bench=. ./internal/sync

# Run specific benchmark
go test -bench=BenchmarkAPIEfficiency ./internal/sync

# Run with memory profiling
go test -bench=. -benchmem ./internal/sync
```

### Comprehensive Benchmark Suite
```bash
# Use the provided script
./scripts/run-benchmarks.sh

# Or run specific category
./scripts/run-benchmarks.sh BenchmarkCacheHitRates
```

### Memory and CPU Profiling
```bash
# Generate profiles
go test -bench=. -benchmem \
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
BenchmarkAPIEfficiency/TreeAPI_SmallRepo_10files-8         	    1000	   1234567 ns/op	   tree-api-calls:1	   content-api-calls:0	   total-api-calls:1
BenchmarkAPIEfficiency/IndividualAPI_SmallRepo_10files-8   	     100	  12345678 ns/op	   tree-api-calls:0	   content-api-calls:10	   total-api-calls:10
```

### Key Metrics Explained
- **ns/op**: Nanoseconds per operation (lower is better)
- **tree-api-calls**: GitHub Tree API calls made
- **content-api-calls**: Individual file API calls made
- **cache-hit-rate-%**: Percentage of successful cache lookups
- **bytes-alloc-per-op**: Memory allocated per benchmark operation
- **mallocs-per-op**: Number of memory allocations per operation

### Performance Targets
Based on the Phase 5 success criteria:

- **API Call Reduction**: Target 80%+ reduction using Tree API
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
   Ensure test fixtures are generated: `make generate-fixtures`

2. **Memory Profile Too Large**
   Reduce benchmark iterations or add `-benchtime=100ms`

3. **Inconsistent Results**
   Run multiple times with `-count=5` for statistical significance

### Debugging Performance Issues

1. **High Memory Usage**
   ```bash
   go test -bench=BenchmarkMemory -memprofile=mem.prof ./internal/sync
   go tool pprof -top mem.prof
   ```

2. **Slow API Calls**
   Check the mock API delay settings in `mockTreeAPIClient`

3. **Cache Misses**
   Verify cache size configuration and access patterns

## Contributing

When adding new benchmarks:

1. Follow naming convention: `BenchmarkFeatureName`
2. Include relevant metrics using `b.ReportMetric()`
3. Test with various input sizes
4. Add documentation to this file
5. Consider real-world usage patterns
6. Ensure benchmarks are deterministic and repeatablerepeatable
