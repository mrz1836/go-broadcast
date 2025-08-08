// Package main implements a benchmark runner for go-broadcast.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
)

type benchmarkRun struct {
	name        string
	description string
	benchFilter string
	count       int
	outputFile  string
}

func main() {
	successColor := color.New(color.FgGreen, color.Bold)
	infoColor := color.New(color.FgCyan)
	errorColor := color.New(color.FgRed, color.Bold)
	headerColor := color.New(color.FgBlue, color.Bold)

	benchmarkName := ""
	if len(os.Args) > 1 {
		benchmarkName = os.Args[1]
	}

	outputDir := "benchmark-results"
	timestamp := time.Now().Format("20060102_150405")

	// Create output directory
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		_, _ = errorColor.Fprintf(os.Stderr, "Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	_, _ = headerColor.Println("Running Go-Broadcast Directory Sync Benchmarks")
	_, _ = headerColor.Println("================================================")
	_, _ = fmt.Fprintln(os.Stdout)

	var benchPattern string
	if benchmarkName != "" {
		_, _ = infoColor.Printf("Running specific benchmark: %s\n", benchmarkName)
		benchPattern = benchmarkName
	} else {
		_, _ = infoColor.Println("Running all benchmarks")
		benchPattern = "."
	}

	// Define all benchmark runs
	benchmarks := []benchmarkRun{
		{
			name:        "Basic benchmarks",
			description: "Running basic benchmarks...",
			benchFilter: benchPattern,
			count:       3,
			outputFile:  fmt.Sprintf("%s/basic_%s.txt", outputDir, timestamp),
		},
		{
			name:        "Memory profiling",
			description: "Running benchmarks with memory profiling...",
			benchFilter: benchPattern,
			count:       1,
			outputFile:  fmt.Sprintf("%s/profile_%s.txt", outputDir, timestamp),
		},
		{
			name:        "API efficiency",
			description: "Running API efficiency benchmarks...",
			benchFilter: "BenchmarkAPIEfficiency",
			count:       5,
			outputFile:  fmt.Sprintf("%s/api_efficiency_%s.txt", outputDir, timestamp),
		},
		{
			name:        "Cache performance",
			description: "Running cache performance benchmarks...",
			benchFilter: "BenchmarkCacheHitRates",
			count:       5,
			outputFile:  fmt.Sprintf("%s/cache_performance_%s.txt", outputDir, timestamp),
		},
		{
			name:        "Memory allocation",
			description: "Running memory allocation benchmarks...",
			benchFilter: "BenchmarkMemoryAllocationPatterns",
			count:       3,
			outputFile:  fmt.Sprintf("%s/memory_allocation_%s.txt", outputDir, timestamp),
		},
		{
			name:        "Real-world scenarios",
			description: "Running real-world scenario benchmarks...",
			benchFilter: "BenchmarkRealWorldScenarios",
			count:       3,
			outputFile:  fmt.Sprintf("%s/real_world_%s.txt", outputDir, timestamp),
		},
		{
			name:        "Performance regression",
			description: "Running performance regression baseline...",
			benchFilter: "BenchmarkPerformanceRegression",
			count:       5,
			outputFile:  fmt.Sprintf("%s/regression_baseline_%s.txt", outputDir, timestamp),
		},
	}

	// Run benchmarks
	ctx := context.Background()
	for i, bench := range benchmarks {
		_, _ = fmt.Fprintf(os.Stdout, "%d. %s\n", i+1, bench.description)

		args := []string{
			"test", "./internal/sync",
			fmt.Sprintf("-bench=%s", bench.benchFilter),
			"-benchmem",
			fmt.Sprintf("-count=%d", bench.count),
		}

		// Add profiling flags for memory profiling run
		if bench.name == "Memory profiling" {
			memProfile := fmt.Sprintf("%s/mem_%s.prof", outputDir, timestamp)
			cpuProfile := fmt.Sprintf("%s/cpu_%s.prof", outputDir, timestamp)
			args = append(args, fmt.Sprintf("-memprofile=%s", memProfile))
			args = append(args, fmt.Sprintf("-cpuprofile=%s", cpuProfile))
		}

		if err := runBenchmark(ctx, args, bench.outputFile); err != nil {
			_, _ = errorColor.Printf("Failed to run %s: %v\n", bench.name, err)
			// Continue with other benchmarks even if one fails
		}
	}

	// Show summary
	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = headerColor.Println("Benchmark Results Summary")
	_, _ = headerColor.Println("==========================")

	// Show basic results
	basicResultsFile := fmt.Sprintf("%s/basic_%s.txt", outputDir, timestamp)
	if results, err := readBenchmarkResults(basicResultsFile, 10); err == nil {
		_, _ = fmt.Fprintln(os.Stdout, "Basic Benchmark Results:")
		for _, line := range results {
			_, _ = fmt.Fprintln(os.Stdout, line)
		}
	}

	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = successColor.Printf("All results saved to: %s/\n", outputDir)

	if benchmarkName == "" || strings.Contains(benchmarkName, "Memory") {
		_, _ = infoColor.Printf("Memory profile: %s/mem_%s.prof\n", outputDir, timestamp)
		_, _ = infoColor.Printf("CPU profile: %s/cpu_%s.prof\n", outputDir, timestamp)

		_, _ = fmt.Fprintln(os.Stdout)
		_, _ = fmt.Fprintln(os.Stdout, "To analyze profiles, use:")
		_, _ = fmt.Fprintf(os.Stdout, "  go tool pprof %s/mem_%s.prof\n", outputDir, timestamp)
		_, _ = fmt.Fprintf(os.Stdout, "  go tool pprof %s/cpu_%s.prof\n", outputDir, timestamp)
	}

	_, _ = fmt.Fprintln(os.Stdout)
	_, _ = fmt.Fprintln(os.Stdout, "To compare benchmarks over time, use:")
	_, _ = fmt.Fprintf(os.Stdout, "  benchstat %s/basic_<old_timestamp>.txt %s/basic_%s.txt\n", outputDir, outputDir, timestamp)
}

func runBenchmark(ctx context.Context, args []string, outputFile string) error {
	cmd := exec.CommandContext(ctx, "go", args...)

	// Create output file
	file, err := os.Create(outputFile) // #nosec G304 -- outputFile is constructed from safe components
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Redirect both stdout and stderr to file
	cmd.Stdout = file
	cmd.Stderr = file

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("benchmark command failed: %w", err)
	}

	return nil
}

func readBenchmarkResults(filename string, maxLines int) ([]string, error) {
	file, err := os.Open(filename) // #nosec G304 -- filename is constructed from safe components
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var results []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Benchmark") {
			results = append(results, line)
			if len(results) >= maxLines {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
