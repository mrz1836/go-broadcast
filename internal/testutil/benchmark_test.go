package testutil

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// BenchmarkCreateBenchmarkFiles tests the CreateBenchmarkFiles function
func BenchmarkCreateBenchmarkFiles(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tempDir := b.TempDir()
		b.StartTimer()

		files := CreateBenchmarkFiles(b, tempDir, 10)

		b.StopTimer()
		// Verify files were created correctly
		require.Len(b, files, 10)
		b.StartTimer()
	}
}

// BenchmarkCreateBenchmarkTempDir tests the CreateBenchmarkTempDir function
func BenchmarkCreateBenchmarkTempDir(b *testing.B) {
	for i := 0; i < b.N; i++ {
		dir := CreateBenchmarkTempDir(b)
		// Verify directory was created
		require.DirExists(b, dir)
	}
}

// BenchmarkWriteBenchmarkFile tests the WriteBenchmarkFile function
func BenchmarkWriteBenchmarkFile(b *testing.B) {
	tempDir := b.TempDir()
	content := "benchmark test content with some data to write"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filePath := filepath.Join(tempDir, "bench_file.txt")
		WriteBenchmarkFile(b, filePath, content)
	}
}

// BenchmarkRunBenchmarkCasesComplete tests RunBenchmarkCases with various scenarios
func BenchmarkRunBenchmarkCasesComplete(b *testing.B) {
	// Track whether setup and cleanup were called
	setupCalled := make(map[string]bool)
	cleanupCalled := make(map[string]bool)

	cases := []BenchmarkCase{
		{
			Name: "WithSetupAndCleanup",
			Size: 100,
			Setup: func() func() {
				setupCalled["WithSetupAndCleanup"] = true
				return func() {
					cleanupCalled["WithSetupAndCleanup"] = true
				}
			},
		},
		{
			Name: "WithoutSetup",
			Size: 200,
		},
		{
			Name: "WithSetupNoCleanup",
			Size: 300,
			Setup: func() func() {
				setupCalled["WithSetupNoCleanup"] = true
				return nil
			},
		},
		{
			Name: "LargeSize",
			Size: 10000,
			Setup: func() func() {
				setupCalled["LargeSize"] = true
				return func() {
					cleanupCalled["LargeSize"] = true
				}
			},
		},
	}

	// Run the benchmark cases
	RunBenchmarkCases(b, cases, func(b *testing.B, bc BenchmarkCase) {
		// Simulate some work based on size
		total := 0
		for i := 0; i < b.N; i++ {
			for j := 0; j < bc.Size; j++ {
				total += j
			}
		}
		// Prevent compiler optimization
		if total < 0 {
			b.Log("negative total")
		}
	})

	// Verify setup and cleanup were called for appropriate cases
	if !setupCalled["WithSetupAndCleanup"] {
		b.Error("setup not called for WithSetupAndCleanup")
	}
	if !cleanupCalled["WithSetupAndCleanup"] {
		b.Error("cleanup not called for WithSetupAndCleanup")
	}
	if !setupCalled["WithSetupNoCleanup"] {
		b.Error("setup not called for WithSetupNoCleanup")
	}
	if !setupCalled["LargeSize"] {
		b.Error("setup not called for LargeSize")
	}
	if !cleanupCalled["LargeSize"] {
		b.Error("cleanup not called for LargeSize")
	}
}

// TestBenchmarkFunctionsWithTestingT tests that benchmark functions work correctly
// This provides coverage for the benchmark-specific functions
func TestBenchmarkFunctionsWithTestingT(t *testing.T) {
	t.Run("CreateBenchmarkFiles", func(t *testing.T) {
		// Create a testing.B-like wrapper to test the function
		result := testing.Benchmark(func(b *testing.B) {
			tempDir := b.TempDir()
			files := CreateBenchmarkFiles(b, tempDir, 3)
			require.Len(b, files, 3)
			for i, file := range files {
				expectedName := filepath.Join(tempDir, "bench_file_"+string(rune('0'+i))+".txt")
				require.Equal(b, expectedName, file)
				require.FileExists(b, file)
			}
		})
		require.Positive(t, result.N)
	})

	t.Run("CreateBenchmarkTempDir", func(t *testing.T) {
		result := testing.Benchmark(func(b *testing.B) {
			dir := CreateBenchmarkTempDir(b)
			require.DirExists(b, dir)
		})
		require.Positive(t, result.N)
	})

	t.Run("WriteBenchmarkFile", func(t *testing.T) {
		result := testing.Benchmark(func(b *testing.B) {
			tempDir := b.TempDir()
			filePath := filepath.Join(tempDir, "test.txt")
			WriteBenchmarkFile(b, filePath, "test content")
			require.FileExists(b, filePath)
		})
		require.Positive(t, result.N)
	})

	t.Run("RunBenchmarkCases", func(t *testing.T) {
		setupRun := false
		cleanupRun := false
		caseRun := false

		result := testing.Benchmark(func(b *testing.B) {
			cases := []BenchmarkCase{
				{
					Name: "TestCase",
					Size: 10,
					Setup: func() func() {
						setupRun = true
						return func() {
							cleanupRun = true
						}
					},
				},
			}

			RunBenchmarkCases(b, cases, func(b *testing.B, bc BenchmarkCase) {
				caseRun = true
				require.Equal(b, "TestCase", bc.Name)
				require.Equal(b, 10, bc.Size)
			})
		})

		require.Positive(t, result.N)
		require.True(t, setupRun, "setup should have been called")
		require.True(t, cleanupRun, "cleanup should have been called")
		require.True(t, caseRun, "benchmark case should have been run")
	})
}
