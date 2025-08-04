package runner

import (
	"context"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
)

// BenchmarkRunner_Performance_SmallCommit simulates a typical small commit (1-3 files)
func BenchmarkRunner_Performance_SmallCommit(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.CheckTimeouts.Fumpt = 30
	cfg.CheckTimeouts.Lint = 60
	cfg.CheckTimeouts.ModTidy = 30
	cfg.CheckTimeouts.Whitespace = 30
	cfg.CheckTimeouts.EOF = 30

	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	runner := New(cfg, "/tmp")
	files := []string{"main.go", "utils.go", "README.md"}

	opts := Options{
		Files: files,
	}

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		_, _ = runner.Run(context.Background(), opts)
	}

	duration := time.Since(start)
	b.Logf("Average execution time: %v", duration/time.Duration(b.N))
}

// BenchmarkRunner_Performance_TypicalCommit simulates a typical commit (5-10 files)
func BenchmarkRunner_Performance_TypicalCommit(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.CheckTimeouts.Fumpt = 30
	cfg.CheckTimeouts.Lint = 60
	cfg.CheckTimeouts.ModTidy = 30
	cfg.CheckTimeouts.Whitespace = 30
	cfg.CheckTimeouts.EOF = 30

	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true

	runner := New(cfg, "/tmp")
	files := []string{
		"cmd/main.go", "pkg/utils.go", "internal/handler.go",
		"internal/service.go", "internal/model.go", "README.md",
		"config.yaml", "Dockerfile", "go.mod",
	}

	opts := Options{
		Files: files,
	}

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		_, _ = runner.Run(context.Background(), opts)
	}

	duration := time.Since(start)
	b.Logf("Average execution time: %v", duration/time.Duration(b.N))
}

// BenchmarkRunner_Performance_LargeCommit simulates a large commit (20+ files)
func BenchmarkRunner_Performance_LargeCommit(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 120,
	}
	cfg.CheckTimeouts.Fumpt = 30
	cfg.CheckTimeouts.Lint = 60
	cfg.CheckTimeouts.ModTidy = 30
	cfg.CheckTimeouts.Whitespace = 30
	cfg.CheckTimeouts.EOF = 30

	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true

	runner := New(cfg, "/tmp")

	// Generate a large set of files
	files := make([]string, 25)
	for i := 0; i < 15; i++ {
		files[i] = "pkg/module" + string(rune('A'+i)) + "/service.go"
	}
	for i := 15; i < 20; i++ {
		files[i] = "cmd/tool" + string(rune('A'+i-15)) + "/main.go"
	}
	files[20] = "README.md"
	files[21] = "CHANGELOG.md"
	files[22] = "go.mod"
	files[23] = "go.sum"
	files[24] = "Makefile"

	opts := Options{
		Files: files,
	}

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		_, _ = runner.Run(context.Background(), opts)
	}

	duration := time.Since(start)
	b.Logf("Average execution time: %v", duration/time.Duration(b.N))
}

// BenchmarkRunner_Performance_AllChecks measures performance with all checks enabled
func BenchmarkRunner_Performance_AllChecks(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 120,
	}
	cfg.CheckTimeouts.Fumpt = 30
	cfg.CheckTimeouts.Lint = 60
	cfg.CheckTimeouts.ModTidy = 30
	cfg.CheckTimeouts.Whitespace = 30
	cfg.CheckTimeouts.EOF = 30

	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true
	cfg.Checks.ModTidy = true

	runner := New(cfg, "/tmp")
	files := []string{
		"main.go", "service.go", "handler.go", "model.go",
		"utils.go", "config.go", "README.md", "go.mod",
	}

	opts := Options{
		Files: files,
	}

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		_, _ = runner.Run(context.Background(), opts)
	}

	duration := time.Since(start)
	b.Logf("Average execution time: %v", duration/time.Duration(b.N))
}

// BenchmarkRunner_Performance_SmartFiltering tests performance improvements from smart filtering
func BenchmarkRunner_Performance_SmartFiltering(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 120,
	}
	cfg.CheckTimeouts.Fumpt = 30
	cfg.CheckTimeouts.Lint = 60
	cfg.CheckTimeouts.ModTidy = 30
	cfg.CheckTimeouts.Whitespace = 30
	cfg.CheckTimeouts.EOF = 30

	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true
	cfg.Checks.ModTidy = true

	runner := New(cfg, "/tmp")

	// Mix of files where only some are relevant to each check
	files := []string{
		"main.go",     // Relevant to fumpt, lint, mod-tidy
		"README.md",   // Relevant to whitespace, EOF only
		"config.json", // Relevant to whitespace, EOF only
		"Dockerfile",  // Relevant to whitespace, EOF only
		"binary.png",  // Should be filtered out by all checks
		"script.py",   // Relevant to whitespace, EOF only
		"styles.css",  // Relevant to whitespace, EOF only
		"data.sql",    // Relevant to whitespace, EOF only
	}

	opts := Options{
		Files: files,
	}

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		_, _ = runner.Run(context.Background(), opts)
	}

	duration := time.Since(start)
	b.Logf("Average execution time: %v", duration/time.Duration(b.N))
	b.Logf("Files processed: %d", len(files))
}

// BenchmarkRunner_Performance_TimeoutHandling tests timeout handling performance
func BenchmarkRunner_Performance_TimeoutHandling(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 30, // Short global timeout
	}
	// Very short timeouts to test timeout handling
	cfg.CheckTimeouts.Fumpt = 5
	cfg.CheckTimeouts.Lint = 10
	cfg.CheckTimeouts.ModTidy = 5
	cfg.CheckTimeouts.Whitespace = 5
	cfg.CheckTimeouts.EOF = 5

	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	runner := New(cfg, "/tmp")
	files := []string{"main.go", "README.md"}

	opts := Options{
		Files: files,
	}

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		_, _ = runner.Run(context.Background(), opts)
	}

	duration := time.Since(start)
	b.Logf("Average execution time: %v", duration/time.Duration(b.N))
}

// BenchmarkRunner_Performance_ParallelExecution tests parallel vs sequential execution
func BenchmarkRunner_Performance_ParallelExecution(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 120,
	}
	cfg.CheckTimeouts.Fumpt = 30
	cfg.CheckTimeouts.Lint = 60
	cfg.CheckTimeouts.ModTidy = 30
	cfg.CheckTimeouts.Whitespace = 30
	cfg.CheckTimeouts.EOF = 30

	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true
	cfg.Performance.ParallelWorkers = 4

	runner := New(cfg, "/tmp")
	files := []string{
		"cmd/main.go", "pkg/service.go", "internal/handler.go",
		"internal/model.go", "utils/helper.go", "README.md",
		"config.yaml", "go.mod", "scripts/deploy.sh",
	}

	opts := Options{
		Files:    files,
		Parallel: 4,
	}

	b.ResetTimer()
	start := time.Now()

	for i := 0; i < b.N; i++ {
		_, _ = runner.Run(context.Background(), opts)
	}

	duration := time.Since(start)
	b.Logf("Average execution time: %v", duration/time.Duration(b.N))
}
