package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecureRandInt(t *testing.T) {
	tests := []struct {
		name     string
		maxVal   int
		expected func(int) bool
	}{
		{
			name:   "zero max value",
			maxVal: 0,
			expected: func(result int) bool {
				return result == 0
			},
		},
		{
			name:   "negative max value",
			maxVal: -5,
			expected: func(result int) bool {
				return result == 0
			},
		},
		{
			name:   "positive max value",
			maxVal: 100,
			expected: func(result int) bool {
				return result >= 0 && result < 100
			},
		},
		{
			name:   "small max value",
			maxVal: 1,
			expected: func(result int) bool {
				return result == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := secureRandInt(tt.maxVal)
			assert.True(t, tt.expected(result), "secureRandInt(%d) = %d", tt.maxVal, result)
		})
	}
}

func TestSecureRandIntDistribution(t *testing.T) {
	maxVal := 10
	iterations := 1000
	results := make(map[int]int)

	for i := 0; i < iterations; i++ {
		result := secureRandInt(maxVal)
		assert.True(t, result >= 0 && result < maxVal, "Result %d out of range [0, %d)", result, maxVal)
		results[result]++
	}

	// Verify all valid values were generated at least once (with high probability)
	assert.Greater(t, len(results), maxVal/2, "Expected better distribution, got %d unique values", len(results))
}

func TestIntensiveTask_Name(t *testing.T) {
	task := &intensiveTask{id: 42}
	expected := "intensive_task_42"
	assert.Equal(t, expected, task.Name())
}

func TestIntensiveTask_Execute(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	task := &intensiveTask{
		id: 1,
		wg: &wg,
	}

	ctx := context.Background()
	err := task.Execute(ctx)

	require.NoError(t, err)

	// Wait for WaitGroup to ensure Done() was called
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - WaitGroup was decremented
	case <-time.After(time.Second):
		t.Fatal("Execute did not call wg.Done()")
	}
}

func TestIntensiveTask_ExecuteConcurrent(t *testing.T) {
	const numTasks = 10
	var wg sync.WaitGroup
	wg.Add(numTasks)

	tasks := make([]*intensiveTask, numTasks)
	for i := 0; i < numTasks; i++ {
		tasks[i] = &intensiveTask{
			id: i,
			wg: &wg,
		}
	}

	// Execute all tasks concurrently
	for _, task := range tasks {
		go func(task *intensiveTask) {
			err := task.Execute(context.Background())
			assert.NoError(t, err)
		}(task)
	}

	// Wait for completion with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Tasks did not complete within timeout")
	}
}

func TestGenerateTestData(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"zero size", 0},
		{"small size", 10},
		{"medium size", 100},
		{"large size", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTestData(tt.size)
			assert.Len(t, result, tt.size, "Expected length %d, got %d", tt.size, len(result))

			if tt.size > 0 {
				// Verify data contains expected characters
				validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
				for _, char := range result {
					assert.Contains(t, validChars, string(char), "Invalid character found: %c", char)
				}
			}
		})
	}
}

func TestGenerateBinaryData(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"zero size", 0},
		{"small size", 10},
		{"medium size", 100},
		{"large size", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateBinaryData(tt.size)
			assert.Len(t, result, tt.size, "Expected length %d, got %d", tt.size, len(result))

			if tt.size > 0 {
				// Check that some null bytes exist (binary characteristic)
				hasNullByte := false
				for _, b := range result {
					if b == 0 {
						hasNullByte = true
						break
					}
				}
				assert.True(t, hasNullByte, "Binary data should contain null bytes")
			}
		})
	}
}

func TestGenerateTextData(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"zero size", 0},
		{"small size", 10},
		{"medium size", 100},
		{"large size", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTextData(tt.size)
			assert.Len(t, result, tt.size, "Expected length %d, got %d", tt.size, len(result))

			if tt.size > 0 {
				// Verify data contains only text characters (including whitespace)
				validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 \n\t"
				for _, b := range result {
					assert.Contains(t, validChars, string(b), "Invalid character found: %c", b)
				}
			}
		})
	}
}

func TestModifyData(t *testing.T) {
	original := []byte("hello world test data")

	tests := []struct {
		name  string
		ratio float64
	}{
		{"no modification", 0.0},
		{"10% modification", 0.1},
		{"50% modification", 0.5},
		{"100% modification", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := modifyData(original, tt.ratio)

			// Length should remain the same
			assert.Len(t, result, len(original), "Modified data length should match original")

			// Count differences
			differences := 0
			for i := range original {
				if original[i] != result[i] {
					differences++
				}
			}

			expectedChanges := int(float64(len(original)) * tt.ratio)

			if tt.ratio == 0.0 {
				assert.Equal(t, 0, differences, "No changes expected for 0.0 ratio")
			} else {
				// Allow larger variance due to randomness - especially for small datasets
				tolerance := max(expectedChanges/2+1, 3) // At least 3 tolerance, or half + 1
				assert.InDelta(t, expectedChanges, differences, float64(tolerance),
					"Expected ~%d changes, got %d", expectedChanges, differences)
			}
		})
	}
}

func TestModifyDataPreservesLength(t *testing.T) {
	sizes := []int{0, 1, 10, 100, 1000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			original := make([]byte, size)
			for i := range original {
				original[i] = byte(i % 256)
			}

			result := modifyData(original, 0.5)
			assert.Len(t, result, size, "Length should be preserved")
		})
	}
}

func TestGenerateTestDataUniqueness(t *testing.T) {
	// Generate multiple strings and verify they're different (with high probability)
	size := 100
	results := make(map[string]bool)
	iterations := 10

	for i := 0; i < iterations; i++ {
		result := generateTestData(size)
		results[result] = true
	}

	// With randomness, we should get mostly unique results
	assert.Greater(t, len(results), iterations/2, "Expected more unique results, got %d", len(results))
}

func TestDataGenerationEdgeCases(t *testing.T) {
	t.Run("generate test data with size 1", func(t *testing.T) {
		result := generateTestData(1)
		assert.Len(t, result, 1)
	})

	t.Run("generate binary data with size 1", func(t *testing.T) {
		result := generateBinaryData(1)
		assert.Len(t, result, 1)
	})

	t.Run("generate text data with size 1", func(t *testing.T) {
		result := generateTextData(1)
		assert.Len(t, result, 1)
	})

	t.Run("modify single byte data", func(t *testing.T) {
		original := []byte{42}
		result := modifyData(original, 1.0)
		assert.Len(t, result, 1)
		// With 100% modification ratio, the single byte should change
		assert.NotEqual(t, original[0], result[0])
	})
}

func TestWorkerPool(t *testing.T) {
	t.Run("testWorkerPool runs without panic", func(t *testing.T) {
		// This is an integration test to ensure testWorkerPool doesn't panic
		require.NotPanics(t, func() {
			testWorkerPool()
		})
	})
}

func TestTTLCache(t *testing.T) {
	t.Run("testTTLCache runs without panic", func(t *testing.T) {
		// This is an integration test to ensure testTTLCache doesn't panic
		require.NotPanics(t, func() {
			testTTLCache()
		})
	})
}

func TestAlgorithmOptimizations(t *testing.T) {
	t.Run("testAlgorithmOptimizations runs without panic", func(t *testing.T) {
		// This is an integration test to ensure testAlgorithmOptimizations doesn't panic
		require.NotPanics(t, func() {
			testAlgorithmOptimizations()
		})
	})
}

func TestBatchProcessing(t *testing.T) {
	t.Run("testBatchProcessing runs without panic", func(t *testing.T) {
		// This is an integration test to ensure testBatchProcessing doesn't panic
		require.NotPanics(t, func() {
			testBatchProcessing()
		})
	})
}

func TestGenerateFinalReport(t *testing.T) {
	t.Run("generateFinalReport runs without panic", func(t *testing.T) {
		// Create temporary directory for test
		tmpDir := t.TempDir()

		// Create mock metrics with positive non-zero values to avoid division by zero
		metrics := map[string]float64{
			"worker_pool_duration_ms":      100.0,
			"cache_duration_ms":            50.0,
			"algorithms_duration_ms":       75.0,
			"batch_processing_duration_ms": 25.0,
		}

		// This is an integration test to ensure generateFinalReport doesn't panic
		require.NotPanics(t, func() {
			generateFinalReport(metrics, tmpDir)
		})
	})

	t.Run("generateFinalReport with minimal metrics", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Use minimal but valid metrics to avoid division by zero
		metrics := map[string]float64{
			"worker_pool_duration_ms":      1.0,
			"cache_duration_ms":            1.0,
			"algorithms_duration_ms":       1.0,
			"batch_processing_duration_ms": 1.0,
		}

		require.NotPanics(t, func() {
			generateFinalReport(metrics, tmpDir)
		})
	})
}
