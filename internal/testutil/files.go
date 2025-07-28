// Package testutil provides shared testing utilities for file creation and mock handling.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// CreateTestFiles creates multiple test files with default content in the specified directory.
// Returns a slice of file paths that were created.
func CreateTestFiles(t *testing.T, dir string, count int) []string {
	t.Helper()

	files := make([]string, count)
	for i := 0; i < count; i++ {
		fileName := fmt.Sprintf("test_file_%d.txt", i)
		filePath := filepath.Join(dir, fileName)
		content := fmt.Sprintf("Test content for file %d\n", i)

		err := os.WriteFile(filePath, []byte(content), 0o600)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", filePath, err)
		}
		files[i] = filePath
	}
	return files
}

// CreateTestFilesWithNames creates test files with specific names and default content.
// Returns a slice of the file paths that were created.
func CreateTestFilesWithNames(t *testing.T, dir string, names []string) []string {
	t.Helper()

	files := make([]string, len(names))
	for i, name := range names {
		filePath := filepath.Join(dir, name)
		content := fmt.Sprintf("Content for %s", name)

		err := os.WriteFile(filePath, []byte(content), 0o600)
		if err != nil {
			t.Fatalf("failed to create test file %s: %v", filePath, err)
		}
		files[i] = filePath
	}
	return files
}

// CreateTestRepo creates a temporary repository directory with cleanup.
// Returns the directory path and a cleanup function.
func CreateTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "test_repo_*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(dir) // Ignore cleanup errors in tests
	}

	return dir, cleanup
}

// WriteTestFile creates a single test file with custom content.
func WriteTestFile(t *testing.T, filePath, content string) {
	t.Helper()

	err := os.WriteFile(filePath, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file %s: %v", filePath, err)
	}
}

// WriteTestFileWithFormat creates a single test file with formatted content.
func WriteTestFileWithFormat(t *testing.T, filePath, format string, args ...interface{}) {
	t.Helper()

	content := fmt.Sprintf(format, args...)
	err := os.WriteFile(filePath, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file %s: %v", filePath, err)
	}
}

// CreateBenchmarkFiles creates multiple test files for benchmark testing with specific content pattern.
// This is optimized for benchmark usage (uses b.Helper() and b.Fatalf).
func CreateBenchmarkFiles(b *testing.B, dir string, count int) []string {
	b.Helper()

	files := make([]string, count)
	for i := 0; i < count; i++ {
		fileName := fmt.Sprintf("bench_file_%d.txt", i)
		filePath := filepath.Join(dir, fileName)
		content := fmt.Sprintf("Benchmark test content %d", i)

		err := os.WriteFile(filePath, []byte(content), 0o600)
		if err != nil {
			b.Fatalf("failed to create benchmark file %s: %v", filePath, err)
		}
		files[i] = filePath
	}
	return files
}

// CreateTestDirectory creates a directory and returns its path.
// If the directory already exists, it does nothing.
func CreateTestDirectory(t *testing.T, dirPath string) {
	t.Helper()

	err := os.MkdirAll(dirPath, 0o750)
	if err != nil {
		t.Fatalf("failed to create directory %s: %v", dirPath, err)
	}
}

// CreateTempDir creates a temporary directory using t.TempDir() and returns the path.
// This leverages Go's built-in cleanup mechanism.
func CreateTempDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// CreateBenchmarkTempDir creates a temporary directory using b.TempDir() for benchmarks.
func CreateBenchmarkTempDir(b *testing.B) string {
	b.Helper()
	return b.TempDir()
}

// WriteBenchmarkFile creates a single file for benchmark testing.
// This is optimized for benchmark usage (uses b.Helper() and b.Fatalf).
func WriteBenchmarkFile(b *testing.B, filePath, content string) {
	b.Helper()

	err := os.WriteFile(filePath, []byte(content), 0o600)
	if err != nil {
		b.Fatalf("failed to create benchmark file %s: %v", filePath, err)
	}
}
