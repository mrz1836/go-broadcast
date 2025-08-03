package runner

import (
	"fmt"
	"testing"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
)

// BenchmarkRunner_Run_SingleCheck measures performance of running a single check
func BenchmarkRunner_Run_SingleCheck(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true

	runner := New(cfg, "/tmp")
	files := []string{"test.go", "readme.md", "main.go"}

	opts := Options{
		Files: files,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(opts)
	}
}

// BenchmarkRunner_Run_MultipleChecks measures performance with multiple checks enabled
func BenchmarkRunner_Run_MultipleChecks(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true

	runner := New(cfg, "/tmp")
	files := []string{"test.go", "readme.md", "main.go", "pkg/util.go", "cmd/main.go"}

	opts := Options{
		Files: files,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(opts)
	}
}

// BenchmarkRunner_Run_LargeFileSet measures performance with many files
func BenchmarkRunner_Run_LargeFileSet(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true

	runner := New(cfg, "/tmp")

	// Generate a large set of files
	files := make([]string, 100)
	for i := 0; i < 100; i++ {
		files[i] = fmt.Sprintf("file%d.go", i)
	}

	opts := Options{
		Files: files,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.Run(opts)
	}
}

// BenchmarkRunner_New measures the cost of creating a new runner
func BenchmarkRunner_New(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		New(cfg, "/tmp")
	}
}

// BenchmarkRunner_Run_Parallel measures parallel execution performance
func BenchmarkRunner_Run_Parallel(b *testing.B) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	files := []string{"test.go", "readme.md", "main.go"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		runner := New(cfg, "/tmp")
		opts := Options{
			Files: files,
		}
		for pb.Next() {
			runner.Run(opts)
		}
	})
}
