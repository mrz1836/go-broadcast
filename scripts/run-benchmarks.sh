#!/bin/bash

# Script to run comprehensive directory sync benchmarks
# Usage: ./scripts/run-benchmarks.sh [benchmark_name]

set -e

BENCHMARK_NAME=${1:-""}
OUTPUT_DIR="benchmark-results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo "Running Go-Broadcast Directory Sync Benchmarks"
echo "================================================"

if [[ -n "$BENCHMARK_NAME" ]]; then
    echo "Running specific benchmark: $BENCHMARK_NAME"
    BENCH_PATTERN="-bench=$BENCHMARK_NAME"
else
    echo "Running all benchmarks"
    BENCH_PATTERN="-bench=."
fi

# Basic benchmark run
echo "1. Running basic benchmarks..."
go test ./internal/sync $BENCH_PATTERN -benchmem -count=3 \
    > "$OUTPUT_DIR/basic_$TIMESTAMP.txt" 2>&1

# Memory profiling
echo "2. Running benchmarks with memory profiling..."
go test ./internal/sync $BENCH_PATTERN -benchmem -memprofile="$OUTPUT_DIR/mem_$TIMESTAMP.prof" \
    -cpuprofile="$OUTPUT_DIR/cpu_$TIMESTAMP.prof" \
    > "$OUTPUT_DIR/profile_$TIMESTAMP.txt" 2>&1

# API efficiency specific benchmarks
echo "3. Running API efficiency benchmarks..."
go test ./internal/sync -bench=BenchmarkAPIEfficiency -benchmem -count=5 \
    > "$OUTPUT_DIR/api_efficiency_$TIMESTAMP.txt" 2>&1

# Cache performance benchmarks
echo "4. Running cache performance benchmarks..."
go test ./internal/sync -bench=BenchmarkCacheHitRates -benchmem -count=5 \
    > "$OUTPUT_DIR/cache_performance_$TIMESTAMP.txt" 2>&1

# Memory allocation benchmarks
echo "5. Running memory allocation benchmarks..."
go test ./internal/sync -bench=BenchmarkMemoryAllocationPatterns -benchmem -count=3 \
    > "$OUTPUT_DIR/memory_allocation_$TIMESTAMP.txt" 2>&1

# Real-world scenario benchmarks
echo "6. Running real-world scenario benchmarks..."
go test ./internal/sync -bench=BenchmarkRealWorldScenarios -benchmem -count=3 \
    > "$OUTPUT_DIR/real_world_$TIMESTAMP.txt" 2>&1

# Performance regression baseline
echo "7. Running performance regression baseline..."
go test ./internal/sync -bench=BenchmarkPerformanceRegression -benchmem -count=5 \
    > "$OUTPUT_DIR/regression_baseline_$TIMESTAMP.txt" 2>&1

echo ""
echo "Benchmark Results Summary"
echo "========================="

# Show basic results
if [[ -f "$OUTPUT_DIR/basic_$TIMESTAMP.txt" ]]; then
    echo "Basic Benchmark Results:"
    grep -E "^Benchmark" "$OUTPUT_DIR/basic_$TIMESTAMP.txt" | head -10
fi

echo ""
echo "All results saved to: $OUTPUT_DIR/"
echo "Memory profile: $OUTPUT_DIR/mem_$TIMESTAMP.prof"
echo "CPU profile: $OUTPUT_DIR/cpu_$TIMESTAMP.prof"

echo ""
echo "To analyze profiles, use:"
echo "  go tool pprof $OUTPUT_DIR/mem_$TIMESTAMP.prof"
echo "  go tool pprof $OUTPUT_DIR/cpu_$TIMESTAMP.prof"

echo ""
echo "To compare benchmarks over time, use:"
echo "  benchstat $OUTPUT_DIR/basic_<old_timestamp>.txt $OUTPUT_DIR/basic_$TIMESTAMP.txt"