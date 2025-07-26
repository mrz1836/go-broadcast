// Package io provides comprehensive benchmarking tests for stream processing functionality.
// These tests evaluate the performance characteristics of file processing, JSON handling,
// and streaming operations under various conditions and data sizes.
package io

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/benchmark"
	"github.com/mrz1836/go-broadcast/internal/memory"
	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// Test data generators for consistent benchmarking
func generateTestFile(t *testing.B, size int, pattern string) string {
	t.Helper()

	tmpDir := testutil.CreateBenchmarkTempDir(t)
	filename := filepath.Join(tmpDir, fmt.Sprintf("test_%d.txt", size))

	data := strings.Repeat(pattern, size/len(pattern)+1)[:size]

	testutil.WriteBenchmarkFile(t, filename, data)

	return filename
}

func generateTestJSONFile(t *testing.B, itemCount int) string {
	t.Helper()

	tmpDir := testutil.CreateBenchmarkTempDir(t)
	filename := filepath.Join(tmpDir, fmt.Sprintf("test_%d.json", itemCount))

	// Generate JSON array with test data
	items := make([]map[string]interface{}, itemCount)
	for i := 0; i < itemCount; i++ {
		items[i] = map[string]interface{}{
			"id":   i,
			"name": fmt.Sprintf("item_%d", i),
			"data": fmt.Sprintf("test_data_%d", i),
			"metadata": map[string]interface{}{
				"created_at": time.Now().Format(time.RFC3339),
				"version":    "1.0",
				"tags":       []string{"test", fmt.Sprintf("tag_%d", i%10)},
			},
		}
	}

	data, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	testutil.WriteBenchmarkFile(t, filename, string(data))

	return filename
}

// Simple transformation function for testing
func uppercaseTransform(data []byte) ([]byte, error) {
	return []byte(strings.ToUpper(string(data))), nil
}

// Identity transformation for baseline testing
func identityTransform(data []byte) ([]byte, error) {
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// BenchmarkFileProcessing compares streaming vs in-memory file processing
func BenchmarkFileProcessing(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"Small_1KB", 1024},
		{"Medium_100KB", 100 * 1024},
		{"Large_1MB", 1024 * 1024},
		{"XLarge_5MB", 5 * 1024 * 1024},
		{"XXLarge_10MB", 10 * 1024 * 1024},
	}

	processor := NewStreamProcessor()
	ctx := context.Background()

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			// Pre-generate test file
			inputFile := generateTestFile(b, size.size, "test data pattern ")

			b.Run("InMemory", func(b *testing.B) {
				// Force in-memory processing by setting high threshold
				testProcessor := NewStreamProcessorWithConfig(
					DefaultChunkSize,
					100*1024*1024, // 100MB threshold - forces in-memory
					DefaultBufferTimeout,
				)

				b.ReportAllocs()
				benchmark.WithMemoryTracking(b, func() {
					outputFile := filepath.Join(b.TempDir(), fmt.Sprintf("output_%d.txt", 1000))
					err := testProcessor.ProcessFile(ctx, inputFile, outputFile, identityTransform)
					if err != nil {
						b.Fatalf("ProcessFile failed: %v", err)
					}

					// Clean up output file to avoid disk space issues
					_ = os.Remove(outputFile)
				})
			})

			b.Run("Streaming", func(b *testing.B) {
				// Force streaming processing by setting low threshold
				testProcessor := NewStreamProcessorWithConfig(
					DefaultChunkSize,
					1024, // 1KB threshold - forces streaming
					DefaultBufferTimeout,
				)

				b.ReportAllocs()
				benchmark.WithMemoryTracking(b, func() {
					outputFile := filepath.Join(b.TempDir(), fmt.Sprintf("output_%d.txt", 1000))
					err := testProcessor.ProcessFile(ctx, inputFile, outputFile, identityTransform)
					if err != nil {
						b.Fatalf("ProcessFile failed: %v", err)
					}

					// Clean up output file to avoid disk space issues
					_ = os.Remove(outputFile)
				})
			})

			b.Run("Auto", func(b *testing.B) {
				// Use default processor with automatic selection
				b.ReportAllocs()
				benchmark.WithMemoryTracking(b, func() {
					outputFile := filepath.Join(b.TempDir(), fmt.Sprintf("output_%d.txt", 1000))
					err := processor.ProcessFile(ctx, inputFile, outputFile, identityTransform)
					if err != nil {
						b.Fatalf("ProcessFile failed: %v", err)
					}

					// Clean up output file to avoid disk space issues
					_ = os.Remove(outputFile)
				})
			})
		})
	}
}

// BenchmarkFileTransformation tests transformation performance
func BenchmarkFileTransformation(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"Small_1KB", 1024},
		{"Medium_100KB", 100 * 1024},
		{"Large_1MB", 1024 * 1024},
	}

	transforms := []struct {
		name      string
		transform TransformFunc
	}{
		{"Identity", identityTransform},
		{"Uppercase", uppercaseTransform},
	}

	processor := NewStreamProcessor()
	ctx := context.Background()

	for _, size := range sizes {
		for _, transform := range transforms {
			b.Run(fmt.Sprintf("%s_%s", size.name, transform.name), func(b *testing.B) {
				inputFile := generateTestFile(b, size.size, "test data pattern ")

				b.ReportAllocs()
				benchmark.WithMemoryTracking(b, func() {
					outputFile := filepath.Join(b.TempDir(), fmt.Sprintf("output_%d.txt", 1000))
					err := processor.ProcessFile(ctx, inputFile, outputFile, transform.transform)
					if err != nil {
						b.Fatalf("ProcessFile failed: %v", err)
					}

					// Clean up output file
					_ = os.Remove(outputFile)
				})
			})
		}
	}
}

// BenchmarkJSONProcessing tests streaming JSON parsing performance
func BenchmarkJSONProcessing(b *testing.B) {
	itemCounts := []struct {
		name  string
		count int
	}{
		{"Small_10", 10},
		{"Medium_100", 100},
		{"Large_1000", 1000},
		{"XLarge_5000", 5000},
		{"XXLarge_10000", 10000},
	}

	processor := NewStreamProcessor()
	ctx := context.Background()

	for _, itemCount := range itemCounts {
		b.Run(itemCount.name, func(b *testing.B) {
			jsonFile := generateTestJSONFile(b, itemCount.count)

			b.Run("StreamingJSON", func(b *testing.B) {
				b.ReportAllocs()
				benchmark.WithMemoryTracking(b, func() {
					processedCount := 0
					handler := func(_ interface{}) error {
						processedCount++
						return nil
					}

					err := processor.ProcessLargeJSON(ctx, jsonFile, handler)
					if err != nil {
						b.Fatalf("ProcessLargeJSON failed: %v", err)
					}

					if processedCount != itemCount.count {
						b.Fatalf("Expected %d items, got %d", itemCount.count, processedCount)
					}
				})
			})

			b.Run("StandardJSON", func(b *testing.B) {
				b.ReportAllocs()
				benchmark.WithMemoryTracking(b, func() {
					data, err := os.ReadFile(jsonFile) //nolint:gosec // Reading test file in benchmark
					if err != nil {
						b.Fatalf("Failed to read JSON file: %v", err)
					}

					var items []interface{}
					if err := json.Unmarshal(data, &items); err != nil {
						b.Fatalf("Failed to unmarshal JSON: %v", err)
					}

					processedCount := 0
					for range items {
						processedCount++
					}

					if processedCount != itemCount.count {
						b.Fatalf("Expected %d items, got %d", itemCount.count, processedCount)
					}
				})
			})
		})
	}
}

// BenchmarkBatchProcessing tests batch file processing performance
func BenchmarkBatchProcessing(b *testing.B) {
	batchSizes := []struct {
		name      string
		fileCount int
		fileSize  int
	}{
		{"Small_10x1KB", 10, 1024},
		{"Medium_50x10KB", 50, 10 * 1024},
		{"Large_100x100KB", 100, 100 * 1024},
	}

	processor := NewStreamProcessor()
	ctx := context.Background()

	for _, batchSize := range batchSizes {
		b.Run(batchSize.name, func(b *testing.B) {
			// Pre-generate test files
			operations := make([]FileOperation, batchSize.fileCount)
			for i := 0; i < batchSize.fileCount; i++ {
				inputFile := generateTestFile(b, batchSize.fileSize, fmt.Sprintf("file_%d_pattern ", i))
				operations[i] = FileOperation{
					InputPath:  inputFile,
					OutputPath: filepath.Join(b.TempDir(), fmt.Sprintf("output_%d.txt", i)),
					Transform:  identityTransform,
				}
			}

			batchProcessor := NewBatchFileProcessor(processor, 10, 5*time.Minute)

			b.ReportAllocs()
			benchmark.WithMemoryTracking(b, func() {
				// Update output paths for each iteration to avoid conflicts
				for j := range operations {
					operations[j].OutputPath = filepath.Join(b.TempDir(), fmt.Sprintf("output_%d_%d.txt", 1000, j))
				}

				err := batchProcessor.ProcessBatch(ctx, operations)
				if err != nil {
					b.Fatalf("ProcessBatch failed: %v", err)
				}

				// Clean up output files
				for _, op := range operations {
					_ = os.Remove(op.OutputPath) // Ignore cleanup errors
				}
			})
		})
	}
}

// BenchmarkMemoryOperations tests memory-efficient operations
func BenchmarkMemoryOperations(b *testing.B) {
	b.Run("StringIntern", func(b *testing.B) {
		strings := []string{
			"repo1", "repo2", "repo3", "main", "develop", "feature/branch",
			"repo1", "repo2", "repo3", "main", "develop", "feature/branch", // Duplicates for cache hits
		}

		b.Run("WithIntern", func(b *testing.B) {
			intern := memory.NewStringIntern()

			b.ReportAllocs()
			benchmark.WithMemoryTracking(b, func() {
				for _, s := range strings {
					_ = intern.Intern(s)
				}
			})
		})

		b.Run("WithoutIntern", func(b *testing.B) {
			b.ReportAllocs()
			benchmark.WithMemoryTracking(b, func() {
				for _, s := range strings {
					// Simulate string copying that would normally happen
					result := make([]byte, len(s))
					copy(result, s)
					_ = string(result)
				}
			})
		})
	})

	b.Run("SlicePreallocation", func(b *testing.B) {
		sizes := []int{10, 100, 1000}

		for _, size := range sizes {
			b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
				b.Run("WithPrealloc", func(b *testing.B) {
					b.ReportAllocs()
					benchmark.WithMemoryTracking(b, func() {
						slice := memory.PreallocateSlice[int](size)
						for j := 0; j < size; j++ {
							slice = append(slice, j)
						}
						_ = slice
					})
				})

				b.Run("WithoutPrealloc", func(b *testing.B) {
					b.ReportAllocs()
					benchmark.WithMemoryTracking(b, func() {
						var slice []int
						for j := 0; j < size; j++ {
							slice = append(slice, j)
						}
						_ = slice
					})
				})
			})
		}
	})

	b.Run("MapPreallocation", func(b *testing.B) {
		sizes := []int{10, 100, 1000}

		for _, size := range sizes {
			b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
				b.Run("WithPrealloc", func(b *testing.B) {
					b.ReportAllocs()
					benchmark.WithMemoryTracking(b, func() {
						m := memory.PreallocateMap[string, int](size)
						for j := 0; j < size; j++ {
							m[fmt.Sprintf("key_%d", j)] = j
						}
						_ = m
					})
				})

				b.Run("WithoutPrealloc", func(b *testing.B) {
					b.ReportAllocs()
					benchmark.WithMemoryTracking(b, func() {
						m := make(map[string]int)
						for j := 0; j < size; j++ {
							m[fmt.Sprintf("key_%d", j)] = j
						}
						_ = m
					})
				})
			})
		}
	})
}

// BenchmarkRealWorldScenarios tests realistic usage patterns
func BenchmarkRealWorldScenarios(b *testing.B) {
	processor := NewStreamProcessor()
	ctx := context.Background()

	b.Run("RepositorySync", func(b *testing.B) {
		// Simulate processing multiple repository files
		repoFiles := []struct {
			name string
			size int
		}{
			{"README.md", 5 * 1024},         // 5KB
			{"main.go", 50 * 1024},          // 50KB
			{"large_file.json", 500 * 1024}, // 500KB
		}

		operations := make([]FileOperation, len(repoFiles))
		for i, file := range repoFiles {
			inputFile := generateTestFile(b, file.size, fmt.Sprintf("%s content ", file.name))
			operations[i] = FileOperation{
				InputPath:  inputFile,
				OutputPath: filepath.Join(b.TempDir(), fmt.Sprintf("sync_%s", file.name)),
				Transform: func(data []byte) ([]byte, error) {
					// Simulate template transformation
					content := strings.ReplaceAll(string(data), "{{REPO}}", "go-broadcast")
					content = strings.ReplaceAll(content, "{{BRANCH}}", "main")
					return []byte(content), nil
				},
			}
		}

		batchProcessor := NewBatchFileProcessor(processor, 10, 5*time.Minute)

		b.ReportAllocs()
		benchmark.WithMemoryTracking(b, func() {
			// Update output paths for each iteration
			for j := range operations {
				operations[j].OutputPath = filepath.Join(b.TempDir(), fmt.Sprintf("sync_%d_%s", 1000, repoFiles[j].name))
			}

			err := batchProcessor.ProcessBatch(ctx, operations)
			if err != nil {
				b.Fatalf("Repository sync simulation failed: %v", err)
			}

			// Clean up output files
			for _, op := range operations {
				_ = os.Remove(op.OutputPath) // Ignore cleanup errors
			}
		})
	})

	b.Run("GitHubAPIResponse", func(b *testing.B) {
		// Simulate processing GitHub API response with many branches
		jsonFile := generateTestJSONFile(b, 1000) // 1000 branches

		b.ReportAllocs()
		benchmark.WithMemoryTracking(b, func() {
			processedBranches := 0
			intern := memory.NewStringIntern()
			branches := memory.PreallocateSlice[string](1000)

			handler := func(item interface{}) error {
				// Simulate branch processing with string interning
				if branchMap, ok := item.(map[string]interface{}); ok {
					if name, ok := branchMap["name"].(string); ok {
						internedName := intern.Intern(name)
						branches = append(branches, internedName)
						processedBranches++
					}
				}
				return nil
			}

			err := processor.ProcessLargeJSON(ctx, jsonFile, handler)
			if err != nil {
				b.Fatalf("GitHub API simulation failed: %v", err)
			}

			if processedBranches != 1000 {
				b.Fatalf("Expected 1000 branches, got %d", processedBranches)
			}
		})
	})
}

// BenchmarkMemoryEfficiency measures memory usage patterns
func BenchmarkMemoryEfficiency(b *testing.B) {
	// Create a memory pressure monitor for testing
	monitor := memory.NewPressureMonitor(
		memory.DefaultThresholds(),
		func(alert memory.Alert) {
			b.Logf("Memory alert: %s - %s", alert.Type, alert.Message)
		},
	)

	b.Run("LargeFileProcessing", func(b *testing.B) {
		processor := NewStreamProcessor()
		ctx := context.Background()

		// Create a 10MB test file
		inputFile := generateTestFile(b, 10*1024*1024, "memory efficiency test pattern ")

		benchmark.WithMemoryTracking(b, func() {
			// Capture memory stats before processing
			startStats := monitor.GetCurrentMemStats()

			outputFile := filepath.Join(b.TempDir(), fmt.Sprintf("memory_output_%d.txt", 1000))
			err := processor.ProcessFile(ctx, inputFile, outputFile, identityTransform)
			if err != nil {
				b.Fatalf("ProcessFile failed: %v", err)
			}

			// Capture memory stats after processing
			endStats := monitor.GetCurrentMemStats()

			// Calculate memory usage
			memoryUsed := endStats.Alloc - startStats.Alloc
			b.ReportMetric(float64(memoryUsed), "bytes-used")

			// Clean up
			_ = os.Remove(outputFile)
		})
	})
}

// Example test to validate functionality
func TestStreamProcessorBasic(t *testing.T) {
	processor := NewStreamProcessor()
	ctx := context.Background()

	// Create test file
	tmpDir := testutil.CreateTempDir(t)
	inputFile := filepath.Join(tmpDir, "input.txt")
	outputFile := filepath.Join(tmpDir, "output.txt")

	testContent := "hello world test content"
	testutil.WriteTestFile(t, inputFile, testContent)

	// Process file with uppercase transformation
	err := processor.ProcessFile(ctx, inputFile, outputFile, uppercaseTransform)
	if err != nil {
		t.Fatalf("ProcessFile failed: %v", err)
	}

	// Verify result
	result, err := os.ReadFile(outputFile) //nolint:gosec // Reading test output file
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	expected := strings.ToUpper(testContent)
	if string(result) != expected {
		t.Errorf("Expected %q, got %q", expected, string(result))
	}

	// Check stats
	stats := processor.GetStats()
	if stats.FilesProcessed != 1 {
		t.Errorf("Expected 1 file processed, got %d", stats.FilesProcessed)
	}
}
