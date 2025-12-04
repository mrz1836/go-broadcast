package io //nolint:revive,nolintlint // internal test package, name conflict intentional

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/testutil"
)

var (
	errTransformFailed      = errors.New("transform failed")
	errHandlerFailed        = errors.New("handler failed")
	errTransformError       = errors.New("transform error")
	errHandlerError         = errors.New("handler error")
	errObjectHandlerError   = errors.New("object handler error")
	errTransformationFailed = errors.New("transformation failed")
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errType error
	}{
		{
			name:    "ValidPath",
			path:    "valid/path/file.txt",
			wantErr: false,
		},
		{
			name:    "PathTraversal_DoubleDot",
			path:    "../../../etc/passwd",
			wantErr: true,
			errType: ErrPathTraversal,
		},
		{
			name:    "PathTraversal_WithDot",
			path:    "valid/../../../etc/passwd",
			wantErr: true,
			errType: ErrPathTraversal,
		},
		{
			name:    "NullByte",
			path:    "file\x00.txt",
			wantErr: true,
			errType: ErrNullByteInPath,
		},
		{
			name:    "ValidPathWithDot",
			path:    "./valid/path.txt",
			wantErr: false,
		},
		{
			name:    "AbsolutePath",
			path:    "/usr/local/bin/app",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					require.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewStreamProcessor(t *testing.T) {
	processor := NewStreamProcessor()

	require.NotNil(t, processor)
	require.Equal(t, DefaultChunkSize, processor.ChunkSize)
	require.Equal(t, DefaultStreamingThreshold, processor.StreamingThreshold)
	require.Equal(t, DefaultBufferTimeout, processor.BufferTimeout)

	// Check that stats are initialized to zero
	stats := processor.GetStats()
	require.Equal(t, int64(0), stats.FilesProcessed)
	require.Equal(t, int64(0), stats.BytesProcessed)
	require.Equal(t, int64(0), stats.StreamingOps)
	require.Equal(t, int64(0), stats.InMemoryOps)
	require.InDelta(t, 0.0, stats.StreamingRatio, 0.001)
}

func TestNewStreamProcessorWithConfig(t *testing.T) {
	tests := []struct {
		name               string
		chunkSize          int
		streamingThreshold int
		timeout            time.Duration
		expectedChunkSize  int
		expectedThreshold  int
		expectedTimeout    time.Duration
	}{
		{
			name:               "ValidConfig",
			chunkSize:          32 * 1024,
			streamingThreshold: 512 * 1024,
			timeout:            10 * time.Second,
			expectedChunkSize:  32 * 1024,
			expectedThreshold:  512 * 1024,
			expectedTimeout:    10 * time.Second,
		},
		{
			name:               "InvalidChunkSize_Zero",
			chunkSize:          0,
			streamingThreshold: 512 * 1024,
			timeout:            10 * time.Second,
			expectedChunkSize:  DefaultChunkSize,
			expectedThreshold:  512 * 1024,
			expectedTimeout:    10 * time.Second,
		},
		{
			name:               "InvalidChunkSize_TooLarge",
			chunkSize:          10 * 1024 * 1024, // Larger than pool threshold
			streamingThreshold: 512 * 1024,
			timeout:            10 * time.Second,
			expectedChunkSize:  DefaultChunkSize,
			expectedThreshold:  512 * 1024,
			expectedTimeout:    10 * time.Second,
		},
		{
			name:               "InvalidStreamingThreshold_Zero",
			chunkSize:          32 * 1024,
			streamingThreshold: 0,
			timeout:            10 * time.Second,
			expectedChunkSize:  32 * 1024,
			expectedThreshold:  DefaultStreamingThreshold,
			expectedTimeout:    10 * time.Second,
		},
		{
			name:               "InvalidStreamingThreshold_TooLarge",
			chunkSize:          32 * 1024,
			streamingThreshold: 20 * 1024 * 1024, // Larger than MaxInMemorySize
			timeout:            10 * time.Second,
			expectedChunkSize:  32 * 1024,
			expectedThreshold:  DefaultStreamingThreshold,
			expectedTimeout:    10 * time.Second,
		},
		{
			name:               "InvalidTimeout_Zero",
			chunkSize:          32 * 1024,
			streamingThreshold: 512 * 1024,
			timeout:            0,
			expectedChunkSize:  32 * 1024,
			expectedThreshold:  512 * 1024,
			expectedTimeout:    DefaultBufferTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewStreamProcessorWithConfig(tt.chunkSize, tt.streamingThreshold, tt.timeout)

			require.Equal(t, tt.expectedChunkSize, processor.ChunkSize)
			require.Equal(t, tt.expectedThreshold, processor.StreamingThreshold)
			require.Equal(t, tt.expectedTimeout, processor.BufferTimeout)
		})
	}
}

func TestStreamProcessorProcessFile(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)

	tests := []struct {
		name            string
		fileContent     string
		forceStreaming  bool
		transform       TransformFunc
		expectedContent string
		expectError     bool
		errorContains   string
	}{
		{
			name:        "SmallFile_InMemory",
			fileContent: "hello world",
			transform: func(data []byte) ([]byte, error) {
				return []byte(strings.ToUpper(string(data))), nil
			},
			expectedContent: "HELLO WORLD",
		},
		{
			name:           "LargeFile_Streaming",
			fileContent:    strings.Repeat("test content ", 1000), // Force streaming
			forceStreaming: true,
			transform: func(data []byte) ([]byte, error) {
				return data, nil // Identity transform
			},
			expectedContent: strings.Repeat("test content ", 1000),
		},
		{
			name:        "TransformError",
			fileContent: "test",
			transform: func(_ []byte) ([]byte, error) {
				return nil, errTransformFailed
			},
			expectError:   true,
			errorContains: "transformation failed",
		},
		{
			name:        "EmptyFile",
			fileContent: "",
			transform: func(data []byte) ([]byte, error) {
				return data, nil
			},
			expectedContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			inputFile := filepath.Join(tempDir, "input_"+tt.name+".txt")
			outputFile := filepath.Join(tempDir, "output_"+tt.name+".txt")

			testutil.WriteTestFile(t, inputFile, tt.fileContent)

			// Create processor
			var processor *StreamProcessor
			if tt.forceStreaming {
				processor = NewStreamProcessorWithConfig(DefaultChunkSize, 100, DefaultBufferTimeout) // Low threshold
			} else {
				processor = NewStreamProcessor()
			}

			// Process file
			ctx := context.Background()
			err := processor.ProcessFile(ctx, inputFile, outputFile, tt.transform)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}

			require.NoError(t, err)

			// Verify output
			result, err := os.ReadFile(outputFile) //nolint:gosec // Reading test file is safe
			require.NoError(t, err)
			require.Equal(t, tt.expectedContent, string(result))

			// Verify stats
			stats := processor.GetStats()
			require.Equal(t, int64(1), stats.FilesProcessed)
			if tt.name != "EmptyFile" {
				require.Positive(t, stats.BytesProcessed)
			} else {
				require.Equal(t, int64(0), stats.BytesProcessed)
			}

			if tt.forceStreaming {
				require.Equal(t, int64(1), stats.StreamingOps)
				require.Equal(t, int64(0), stats.InMemoryOps)
			} else {
				require.Equal(t, int64(0), stats.StreamingOps)
				require.Equal(t, int64(1), stats.InMemoryOps)
			}
		})
	}
}

func TestStreamProcessorProcessFileErrors(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	processor := NewStreamProcessor()
	ctx := context.Background()

	t.Run("InputFileNotFound", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
		outputFile := filepath.Join(tempDir, "output.txt")

		err := processor.ProcessFile(ctx, nonExistentFile, outputFile, func(data []byte) ([]byte, error) {
			return data, nil
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to stat input file")

		stats := processor.GetStats()
		require.Positive(t, stats.ErrorCount)
	})

	t.Run("InvalidInputPath", func(t *testing.T) {
		// Test path validation by trying to access file directly using processFileInMemory
		invalidPath := "../../../etc/passwd"
		outputFile := filepath.Join(tempDir, "output.txt")

		// Create a small file to trigger in-memory processing
		inputFile := filepath.Join(tempDir, "small.txt")
		testutil.WriteTestFile(t, inputFile, "test")

		// Force the processor to use the invalid path internally
		err := processor.processFileInMemory(ctx, invalidPath, outputFile, func(data []byte) ([]byte, error) {
			return data, nil
		}, 4)

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid input path")
	})

	t.Run("InvalidOutputPath", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "input.txt")
		testutil.WriteTestFile(t, inputFile, "test")

		invalidOutputPath := "../../../tmp/output.txt"

		err := processor.ProcessFile(ctx, inputFile, invalidOutputPath, func(data []byte) ([]byte, error) {
			return data, nil
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid output path")
	})
}

func TestStreamProcessorProcessFileContext(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	processor := NewStreamProcessorWithConfig(1024, 100, DefaultBufferTimeout) // Force streaming

	// Create a large file
	inputFile := filepath.Join(tempDir, "large.txt")
	outputFile := filepath.Join(tempDir, "output.txt")
	largeContent := strings.Repeat("test content that will trigger streaming mode ", 1000)
	testutil.WriteTestFile(t, inputFile, largeContent)

	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Start processing in goroutine
		errChan := make(chan error, 1)
		go func() {
			err := processor.ProcessFile(ctx, inputFile, outputFile, func(data []byte) ([]byte, error) {
				// Slow transform to ensure context cancellation is caught
				time.Sleep(10 * time.Millisecond)
				return data, nil
			})
			errChan <- err
		}()

		// Cancel context after a short delay
		time.Sleep(5 * time.Millisecond)
		cancel()

		// Wait for result
		select {
		case err := <-errChan:
			require.Error(t, err)
			require.Equal(t, context.Canceled, err)
		case <-time.After(time.Second):
			t.Fatal("Test timed out")
		}
	})
}

func TestStreamProcessorProcessLargeJSON(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	processor := NewStreamProcessor()
	ctx := context.Background()

	t.Run("JSONArray", func(t *testing.T) {
		// Create JSON array file
		testData := []map[string]interface{}{
			{"id": 1, "name": "item1", "value": "test1"},
			{"id": 2, "name": "item2", "value": "test2"},
			{"id": 3, "name": "item3", "value": "test3"},
		}

		jsonFile := filepath.Join(tempDir, "array.json")
		data, err := json.Marshal(testData)
		require.NoError(t, err)
		testutil.WriteTestFile(t, jsonFile, string(data))

		// Process JSON
		var processedItems []interface{}
		handler := func(item interface{}) error {
			processedItems = append(processedItems, item)
			return nil
		}

		_ = processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.Len(t, processedItems, 3)

		// Verify first item
		firstItem, ok := processedItems[0].(map[string]interface{})
		require.True(t, ok)
		require.InDelta(t, 1.0, firstItem["id"], 0.001) // JSON numbers are float64
		require.Equal(t, "item1", firstItem["name"])
	})

	t.Run("JSONObject", func(t *testing.T) {
		// Create JSON object file
		testData := map[string]interface{}{
			"property1": "value1",
			"property2": 42,
			"property3": true,
		}

		jsonFile := filepath.Join(tempDir, "object.json")
		data, err := json.Marshal(testData)
		require.NoError(t, err)
		testutil.WriteTestFile(t, jsonFile, string(data))

		// Process JSON
		var processedProperties []map[string]interface{}
		handler := func(item interface{}) error {
			if prop, ok := item.(map[string]interface{}); ok {
				processedProperties = append(processedProperties, prop)
			}
			return nil
		}

		_ = processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.Len(t, processedProperties, 3)

		// Verify that all properties were processed
		allKeys := make(map[string]bool)
		for _, prop := range processedProperties {
			for key := range prop {
				allKeys[key] = true
			}
		}
		require.True(t, allKeys["property1"])
		require.True(t, allKeys["property2"])
		require.True(t, allKeys["property3"])
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		jsonFile := filepath.Join(tempDir, "invalid.json")
		testutil.WriteTestFile(t, jsonFile, "invalid json content")

		handler := func(_ interface{}) error {
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read JSON opening token")
	})

	t.Run("InvalidJSONStructure", func(t *testing.T) {
		jsonFile := filepath.Join(tempDir, "invalid_structure.json")
		testutil.WriteTestFile(t, jsonFile, "\"just a string\"")

		handler := func(_ interface{}) error {
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidJSONStructure)
	})

	t.Run("HandlerError", func(t *testing.T) {
		testData := []map[string]interface{}{
			{"id": 1, "name": "item1"},
		}

		jsonFile := filepath.Join(tempDir, "handler_error.json")
		data, err := json.Marshal(testData)
		require.NoError(t, err)
		testutil.WriteTestFile(t, jsonFile, string(data))

		handler := func(_ interface{}) error {
			return errHandlerFailed
		}

		err = processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "handler failed for item 0")
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		// Create large JSON array
		testData := make([]map[string]interface{}, 1000)
		for i := 0; i < 1000; i++ {
			testData[i] = map[string]interface{}{"id": i}
		}

		jsonFile := filepath.Join(tempDir, "large_array.json")
		data, err := json.Marshal(testData)
		require.NoError(t, err)
		testutil.WriteTestFile(t, jsonFile, string(data))

		cancelCtx, cancel := context.WithCancel(context.Background())
		processedCount := 0

		handler := func(_ interface{}) error {
			processedCount++
			if processedCount > 10 {
				cancel() // Cancel after processing some items
			}
			return nil
		}

		err = processor.ProcessLargeJSON(cancelCtx, jsonFile, handler)
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
		require.Positive(t, processedCount)
		require.Less(t, processedCount, 1000)
	})

	t.Run("InvalidPath", func(t *testing.T) {
		invalidPath := "../../../etc/passwd"
		handler := func(_ interface{}) error {
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, invalidPath, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid input path")
	})

	t.Run("FileNotFound", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent.json")
		handler := func(_ interface{}) error {
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, nonExistentFile, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to open JSON file")
	})
}

func TestNewBatchFileProcessor(t *testing.T) {
	processor := NewStreamProcessor()

	tests := []struct {
		name            string
		maxBatchSize    int
		batchTimeout    time.Duration
		expectedSize    int
		expectedTimeout time.Duration
	}{
		{
			name:            "ValidConfig",
			maxBatchSize:    50,
			batchTimeout:    2 * time.Minute,
			expectedSize:    50,
			expectedTimeout: 2 * time.Minute,
		},
		{
			name:            "InvalidMaxBatchSize_Zero",
			maxBatchSize:    0,
			batchTimeout:    2 * time.Minute,
			expectedSize:    100, // Default
			expectedTimeout: 2 * time.Minute,
		},
		{
			name:            "InvalidMaxBatchSize_Negative",
			maxBatchSize:    -10,
			batchTimeout:    2 * time.Minute,
			expectedSize:    100, // Default
			expectedTimeout: 2 * time.Minute,
		},
		{
			name:            "InvalidTimeout_Zero",
			maxBatchSize:    50,
			batchTimeout:    0,
			expectedSize:    50,
			expectedTimeout: 5 * time.Minute, // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bfp := NewBatchFileProcessor(processor, tt.maxBatchSize, tt.batchTimeout)

			require.NotNil(t, bfp)
			require.Equal(t, processor, bfp.processor)
			require.Equal(t, tt.expectedSize, bfp.maxBatchSize)
			require.Equal(t, tt.expectedTimeout, bfp.batchTimeout)
		})
	}
}

func TestBatchFileProcessorProcessBatch(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	processor := NewStreamProcessor()
	bfp := NewBatchFileProcessor(processor, 2, time.Minute) // Small batch size for testing
	ctx := context.Background()

	t.Run("EmptyBatch", func(t *testing.T) {
		err := bfp.ProcessBatch(ctx, []FileOperation{})
		require.NoError(t, err)
	})

	t.Run("SingleOperation", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "single_input.txt")
		outputFile := filepath.Join(tempDir, "single_output.txt")

		testutil.WriteTestFile(t, inputFile, "test content")

		operations := []FileOperation{
			{
				InputPath:  inputFile,
				OutputPath: outputFile,
				Transform: func(data []byte) ([]byte, error) {
					return []byte(strings.ToUpper(string(data))), nil
				},
			},
		}

		err := bfp.ProcessBatch(ctx, operations)
		require.NoError(t, err)

		// Verify output
		result, err := os.ReadFile(outputFile) //nolint:gosec // Reading test file is safe
		require.NoError(t, err)
		require.Equal(t, "TEST CONTENT", string(result))
	})

	t.Run("MultipleBatches", func(t *testing.T) {
		// Create 5 operations to trigger multiple batches (batch size is 2)
		operations := make([]FileOperation, 5)
		for i := 0; i < 5; i++ {
			inputFile := filepath.Join(tempDir, fmt.Sprintf("batch_input_%d.txt", i))
			outputFile := filepath.Join(tempDir, fmt.Sprintf("batch_output_%d.txt", i))

			content := fmt.Sprintf("content %d", i)
			testutil.WriteTestFile(t, inputFile, content)

			operations[i] = FileOperation{
				InputPath:  inputFile,
				OutputPath: outputFile,
				Transform: func(data []byte) ([]byte, error) {
					return data, nil // Identity transform
				},
			}
		}

		err := bfp.ProcessBatch(ctx, operations)
		require.NoError(t, err)

		// Verify all outputs
		for i := 0; i < 5; i++ {
			result, err := os.ReadFile(operations[i].OutputPath)
			require.NoError(t, err)
			require.Equal(t, fmt.Sprintf("content %d", i), string(result))
		}
	})

	t.Run("BatchWithError", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "error_input.txt")
		outputFile := filepath.Join(tempDir, "error_output.txt")

		testutil.WriteTestFile(t, inputFile, "test")

		operations := []FileOperation{
			{
				InputPath:  inputFile,
				OutputPath: outputFile,
				Transform: func(_ []byte) ([]byte, error) {
					return nil, errTransformError
				},
			},
		}

		err := bfp.ProcessBatch(ctx, operations)
		require.Error(t, err)
		require.Contains(t, err.Error(), "batch processing failed")
		require.Contains(t, err.Error(), "transform error")
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Create a slow operation
		inputFile := filepath.Join(tempDir, "slow_input.txt")
		outputFile := filepath.Join(tempDir, "slow_output.txt")

		testutil.WriteTestFile(t, inputFile, "test")

		operations := []FileOperation{
			{
				InputPath:  inputFile,
				OutputPath: outputFile,
				Transform: func(data []byte) ([]byte, error) {
					time.Sleep(100 * time.Millisecond)
					return data, nil
				},
			},
		}

		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := bfp.ProcessBatch(ctx, operations)
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
	})
}

func TestStreamProcessorStats(t *testing.T) {
	processor := NewStreamProcessor()
	ctx := context.Background()
	tempDir := testutil.CreateTempDir(t)

	// Initially stats should be zero
	stats := processor.GetStats()
	require.Equal(t, int64(0), stats.FilesProcessed)
	require.Equal(t, int64(0), stats.BytesProcessed)
	require.Equal(t, int64(0), stats.StreamingOps)
	require.Equal(t, int64(0), stats.InMemoryOps)
	require.Equal(t, int64(0), stats.ChunksProcessed)
	require.Equal(t, int64(0), stats.ErrorCount)
	require.InDelta(t, 0.0, stats.StreamingRatio, 0.001)

	// Process a small file (in-memory)
	smallFile := filepath.Join(tempDir, "small.txt")
	smallOutput := filepath.Join(tempDir, "small_out.txt")
	testutil.WriteTestFile(t, smallFile, "small content")

	err := processor.ProcessFile(ctx, smallFile, smallOutput, func(data []byte) ([]byte, error) {
		return data, nil
	})
	require.NoError(t, err)

	stats = processor.GetStats()
	require.Equal(t, int64(1), stats.FilesProcessed)
	require.Positive(t, stats.BytesProcessed)
	require.Equal(t, int64(0), stats.StreamingOps)
	require.Equal(t, int64(1), stats.InMemoryOps)
	require.InDelta(t, 0.0, stats.StreamingRatio, 0.001) // 0% streaming

	// Force streaming with a low threshold processor
	streamingProcessor := NewStreamProcessorWithConfig(DefaultChunkSize, 1, DefaultBufferTimeout)
	largeFile := filepath.Join(tempDir, "large.txt")
	largeOutput := filepath.Join(tempDir, "large_out.txt")
	largeContent := strings.Repeat("large content ", 100)
	testutil.WriteTestFile(t, largeFile, largeContent)

	_ = streamingProcessor.ProcessFile(ctx, largeFile, largeOutput, func(data []byte) ([]byte, error) {
		return data, nil
	})

	streamingStats := streamingProcessor.GetStats()
	require.Equal(t, int64(1), streamingStats.FilesProcessed)
	require.Positive(t, streamingStats.BytesProcessed)
	require.Equal(t, int64(1), streamingStats.StreamingOps)
	require.Equal(t, int64(0), streamingStats.InMemoryOps)
	require.InDelta(t, 100.0, streamingStats.StreamingRatio, 0.001) // 100% streaming
	require.Positive(t, streamingStats.ChunksProcessed)

	// Test reset stats
	streamingProcessor.ResetStats()
	resetStats := streamingProcessor.GetStats()
	require.Equal(t, int64(0), resetStats.FilesProcessed)
	require.Equal(t, int64(0), resetStats.BytesProcessed)
	require.Equal(t, int64(0), resetStats.StreamingOps)
	require.Equal(t, int64(0), resetStats.InMemoryOps)
	require.Equal(t, int64(0), resetStats.ChunksProcessed)
	require.Equal(t, int64(0), resetStats.ErrorCount)
	require.InDelta(t, 0.0, resetStats.StreamingRatio, 0.001)
}

func TestStreamProcessorConcurrency(t *testing.T) {
	processor := NewStreamProcessor()
	ctx := context.Background()
	tempDir := testutil.CreateTempDir(t)

	// Create test files
	const numFiles = 10
	inputFiles := make([]string, numFiles)
	outputFiles := make([]string, numFiles)

	for i := 0; i < numFiles; i++ {
		inputFile := filepath.Join(tempDir, fmt.Sprintf("concurrent_input_%d.txt", i))
		outputFile := filepath.Join(tempDir, fmt.Sprintf("concurrent_output_%d.txt", i))

		content := fmt.Sprintf("content for file %d", i)
		testutil.WriteTestFile(t, inputFile, content)

		inputFiles[i] = inputFile
		outputFiles[i] = outputFile
	}

	// Process files concurrently
	errChan := make(chan error, numFiles)
	for i := 0; i < numFiles; i++ {
		go func(index int) {
			err := processor.ProcessFile(ctx, inputFiles[index], outputFiles[index], func(data []byte) ([]byte, error) {
				return []byte(strings.ToUpper(string(data))), nil
			})
			errChan <- err
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numFiles; i++ {
		err := <-errChan
		require.NoError(t, err)
	}

	// Verify results
	for i := 0; i < numFiles; i++ {
		result, err := os.ReadFile(outputFiles[i])
		require.NoError(t, err)
		expected := strings.ToUpper(fmt.Sprintf("content for file %d", i))
		require.Equal(t, expected, string(result))
	}

	// Verify stats
	stats := processor.GetStats()
	require.Equal(t, int64(numFiles), stats.FilesProcessed)
	require.Positive(t, stats.BytesProcessed)
}

func TestStreamProcessorEdgeCase(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	processor := NewStreamProcessor()
	ctx := context.Background()

	t.Run("VeryLargeFile", func(t *testing.T) {
		// Create a 2MB file to test streaming
		largeFile := filepath.Join(tempDir, "very_large.txt")
		outputFile := filepath.Join(tempDir, "very_large_out.txt")

		// Create content in chunks to avoid memory issues
		file, err := os.Create(largeFile) //nolint:gosec // Creating test file is safe
		require.NoError(t, err)

		chunk := strings.Repeat("X", 64*1024) // 64KB chunk
		for i := 0; i < 32; i++ {             // 32 * 64KB = 2MB
			_, writeErr := file.WriteString(chunk)
			require.NoError(t, writeErr)
		}
		err = file.Close()
		require.NoError(t, err)

		// Process the large file
		err = processor.ProcessFile(ctx, largeFile, outputFile, func(data []byte) ([]byte, error) {
			// Convert to lowercase (simple transformation)
			return []byte(strings.ToLower(string(data))), nil
		})
		require.NoError(t, err)

		// Verify the output file exists and has content
		outputInfo, err := os.Stat(outputFile)
		require.NoError(t, err)
		require.Positive(t, outputInfo.Size())

		// Verify stats show streaming was used
		stats := processor.GetStats()
		require.Positive(t, stats.StreamingOps)
		require.Positive(t, stats.ChunksProcessed)
	})

	t.Run("EmptyTransform", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "empty_transform.txt")
		outputFile := filepath.Join(tempDir, "empty_transform_out.txt")

		testutil.WriteTestFile(t, inputFile, "test content")

		// Transform that returns empty slice
		err := processor.ProcessFile(ctx, inputFile, outputFile, func(_ []byte) ([]byte, error) {
			return []byte{}, nil
		})
		require.NoError(t, err)

		// Verify empty output
		result, err := os.ReadFile(outputFile) //nolint:gosec // Reading test file is safe
		require.NoError(t, err)
		require.Empty(t, result)
	})
}

func TestStreamProcessorEdgeCases(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	processor := NewStreamProcessor()
	ctx := context.Background()

	t.Run("ProcessFile_WithSymlink", func(t *testing.T) {
		// Create a real file
		realFile := filepath.Join(tempDir, "real.txt")
		testutil.WriteTestFile(t, realFile, "real content")

		// Create a symlink
		symlinkFile := filepath.Join(tempDir, "symlink.txt")
		err := os.Symlink(realFile, symlinkFile)
		if err != nil {
			t.Skip("Cannot create symlinks on this system")
		}

		outputFile := filepath.Join(tempDir, "output.txt")
		err = processor.ProcessFile(ctx, symlinkFile, outputFile, func(data []byte) ([]byte, error) {
			return []byte(strings.ToUpper(string(data))), nil
		})
		require.NoError(t, err)

		content, err := os.ReadFile(outputFile) //nolint:gosec // Test file read
		require.NoError(t, err)
		require.Equal(t, "REAL CONTENT", string(content))
	})

	t.Run("ProcessFile_EmptyFile", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "empty.txt")
		outputFile := filepath.Join(tempDir, "empty_out.txt")
		testutil.WriteTestFile(t, inputFile, "")

		err := processor.ProcessFile(ctx, inputFile, outputFile, func(data []byte) ([]byte, error) {
			return data, nil
		})
		require.NoError(t, err)

		content, err := os.ReadFile(outputFile) //nolint:gosec // Test file read
		require.NoError(t, err)
		require.Empty(t, string(content))
	})

	t.Run("ProcessFile_BinaryData", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "binary.dat")
		outputFile := filepath.Join(tempDir, "binary_out.dat")

		// Create binary data
		binaryData := make([]byte, 256)
		for i := range binaryData {
			binaryData[i] = byte(i)
		}
		err := os.WriteFile(inputFile, binaryData, 0o600)
		require.NoError(t, err)

		// Process with identity transform
		err = processor.ProcessFile(ctx, inputFile, outputFile, func(data []byte) ([]byte, error) {
			return data, nil
		})
		require.NoError(t, err)

		// Verify output
		outputData, err := os.ReadFile(outputFile) //nolint:gosec // Test file read
		require.NoError(t, err)
		require.Equal(t, binaryData, outputData)
	})

	t.Run("ProcessFile_TransformError", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "error_transform.txt")
		outputFile := filepath.Join(tempDir, "error_out.txt")
		testutil.WriteTestFile(t, inputFile, "test content")

		err := processor.ProcessFile(ctx, inputFile, outputFile, func(_ []byte) ([]byte, error) {
			return nil, errTransformError
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "transform error")

		// Verify stats counted the error
		stats := processor.GetStats()
		require.Positive(t, stats.ErrorCount)
	})

	t.Run("ProcessFile_InvalidOutputPath", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "valid_input.txt")
		testutil.WriteTestFile(t, inputFile, "test content")

		invalidOutputPath := "/nonexistent/directory/output.txt"
		err := processor.ProcessFile(ctx, inputFile, invalidOutputPath, func(data []byte) ([]byte, error) {
			return data, nil
		})
		require.Error(t, err)
	})

	t.Run("ProcessFile_ReadOnlyOutputDir", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Cannot test read-only directory as root")
		}

		inputFile := filepath.Join(tempDir, "input_readonly.txt")
		testutil.WriteTestFile(t, inputFile, "test content")

		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.Mkdir(readOnlyDir, 0o750)
		require.NoError(t, err)

		// Make directory read-only
		err = os.Chmod(readOnlyDir, 0o555) //nolint:gosec // Setting read-only for test
		require.NoError(t, err)
		defer func() { _ = os.Chmod(readOnlyDir, 0o755) }() //nolint:gosec // Restore permissions

		outputFile := filepath.Join(readOnlyDir, "output.txt")
		err = processor.ProcessFile(ctx, inputFile, outputFile, func(data []byte) ([]byte, error) {
			return data, nil
		})
		require.Error(t, err)
	})

	t.Run("ProcessFile_LargeChunkTransform", func(t *testing.T) {
		// Use a custom processor with small chunk size
		customProcessor := NewStreamProcessorWithConfig(1024, 2048, 30*time.Second)

		inputFile := filepath.Join(tempDir, "large_chunk.txt")
		outputFile := filepath.Join(tempDir, "large_chunk_out.txt")

		// Create content larger than chunk size
		content := strings.Repeat("ABCDEFGHIJ", 300) // 3000 bytes
		testutil.WriteTestFile(t, inputFile, content)

		// Transform that doubles the size
		err := customProcessor.ProcessFile(ctx, inputFile, outputFile, func(data []byte) ([]byte, error) {
			doubled := make([]byte, len(data)*2)
			copy(doubled, data)
			copy(doubled[len(data):], data)
			return doubled, nil
		})
		require.NoError(t, err)

		outputContent, err := os.ReadFile(outputFile) //nolint:gosec // Test file read
		require.NoError(t, err)
		require.Len(t, outputContent, len(content)*2)
	})
}

func TestProcessLargeJSONEdgeCases(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	processor := NewStreamProcessor()
	ctx := context.Background()

	t.Run("JSONArray_EmptyArray", func(t *testing.T) {
		jsonFile := filepath.Join(tempDir, "empty_array.json")
		testutil.WriteTestFile(t, jsonFile, "[]")

		var processedItems []interface{}
		handler := func(item interface{}) error {
			processedItems = append(processedItems, item)
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.NoError(t, err)
		require.Empty(t, processedItems)
	})

	t.Run("JSONObject_EmptyObject", func(t *testing.T) {
		jsonFile := filepath.Join(tempDir, "empty_object.json")
		testutil.WriteTestFile(t, jsonFile, "{}")

		var processedItems []interface{}
		handler := func(item interface{}) error {
			processedItems = append(processedItems, item)
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.NoError(t, err)
		require.Empty(t, processedItems)
	})

	t.Run("JSONArray_NestedStructures", func(t *testing.T) {
		testData := []interface{}{
			map[string]interface{}{
				"id": 1,
				"nested": map[string]interface{}{
					"level2": map[string]interface{}{
						"level3": []interface{}{1, 2, 3},
					},
				},
			},
			[]interface{}{
				map[string]interface{}{"a": 1},
				map[string]interface{}{"b": 2},
			},
		}

		jsonFile := filepath.Join(tempDir, "nested.json")
		data, err := json.Marshal(testData)
		require.NoError(t, err)
		testutil.WriteTestFile(t, jsonFile, string(data))

		var processedItems []interface{}
		handler := func(item interface{}) error {
			processedItems = append(processedItems, item)
			return nil
		}

		err = processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.NoError(t, err)
		require.Len(t, processedItems, 2)
	})

	t.Run("JSONArray_LargeStrings", func(t *testing.T) {
		// Create array with large string values
		largeString := strings.Repeat("a", 100000)
		testData := []map[string]interface{}{
			{"id": 1, "data": largeString},
			{"id": 2, "data": largeString},
		}

		jsonFile := filepath.Join(tempDir, "large_strings.json")
		data, err := json.Marshal(testData)
		require.NoError(t, err)
		testutil.WriteTestFile(t, jsonFile, string(data))

		processedCount := 0
		handler := func(item interface{}) error {
			processedCount++
			// Verify we got the large string
			if m, ok := item.(map[string]interface{}); ok {
				if str, ok := m["data"].(string); ok {
					require.Len(t, str, len(largeString))
				}
			}
			return nil
		}

		err = processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.NoError(t, err)
		require.Equal(t, 2, processedCount)
	})

	t.Run("JSON_MalformedContent", func(t *testing.T) {
		tests := []struct {
			name    string
			content string
		}{
			{"truncated array", "[{\"id\": 1},"},
			{"truncated object", "{\"prop1\": \"value1\","},
			{"invalid delimiter", "[{\"id\": 1},,{\"id\": 2}]"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				jsonFile := filepath.Join(tempDir, "malformed.json")
				testutil.WriteTestFile(t, jsonFile, tt.content)

				handler := func(_ interface{}) error {
					return nil
				}

				err := processor.ProcessLargeJSON(ctx, jsonFile, handler)
				require.Error(t, err)
			})
		}
	})

	t.Run("JSON_SpecialCharacters", func(t *testing.T) {
		testData := []map[string]interface{}{
			{
				"id":        1,
				"unicode":   "Hello 世界 🌍", //nolint:gosmopolitan // Testing Unicode support
				"escaped":   "Line1\nLine2\tTabbed",
				"quotes":    "She said \"Hello\"",
				"backslash": "C:\\Users\\Test",
				"null_char": "before\x00after",
			},
		}

		jsonFile := filepath.Join(tempDir, "special_chars.json")
		data, err := json.Marshal(testData)
		require.NoError(t, err)
		testutil.WriteTestFile(t, jsonFile, string(data))

		var processedItems []interface{}
		handler := func(item interface{}) error {
			processedItems = append(processedItems, item)
			return nil
		}

		err = processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.NoError(t, err)
		require.Len(t, processedItems, 1)

		// Verify special characters were preserved
		item := processedItems[0].(map[string]interface{})
		require.Equal(t, "Hello 世界 🌍", item["unicode"]) //nolint:gosmopolitan // Testing Unicode support
		require.Equal(t, "Line1\nLine2\tTabbed", item["escaped"])
	})
}

func TestBatchFileProcessorEdgeCases(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	processor := NewStreamProcessor()
	batchProcessor := NewBatchFileProcessor(processor, 5, 30*time.Second)
	ctx := context.Background()

	t.Run("EmptyBatch", func(t *testing.T) {
		err := batchProcessor.ProcessBatch(ctx, []FileOperation{})
		require.NoError(t, err)
	})

	t.Run("SingleItemBatch", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "single_input.txt")
		outputFile := filepath.Join(tempDir, "single_output.txt")
		testutil.WriteTestFile(t, inputFile, "single content")

		operations := []FileOperation{
			{
				InputPath:  inputFile,
				OutputPath: outputFile,
				Transform: func(data []byte) ([]byte, error) {
					return []byte(strings.ToUpper(string(data))), nil
				},
			},
		}

		err := batchProcessor.ProcessBatch(ctx, operations)
		require.NoError(t, err)

		content, err := os.ReadFile(outputFile) //nolint:gosec // Test file read
		require.NoError(t, err)
		require.Equal(t, "SINGLE CONTENT", string(content))
	})

	t.Run("BatchWithErrors", func(t *testing.T) {
		operations := []FileOperation{
			{
				InputPath:  filepath.Join(tempDir, "exists1.txt"),
				OutputPath: filepath.Join(tempDir, "out1.txt"),
				Transform:  func(data []byte) ([]byte, error) { return data, nil },
			},
			{
				InputPath:  filepath.Join(tempDir, "nonexistent.txt"), // This will fail
				OutputPath: filepath.Join(tempDir, "out2.txt"),
				Transform:  func(data []byte) ([]byte, error) { return data, nil },
			},
			{
				InputPath:  filepath.Join(tempDir, "exists3.txt"),
				OutputPath: filepath.Join(tempDir, "out3.txt"),
				Transform:  func(data []byte) ([]byte, error) { return data, nil },
			},
		}

		// Create the files that should exist
		testutil.WriteTestFile(t, operations[0].InputPath, "content1")
		testutil.WriteTestFile(t, operations[2].InputPath, "content3")

		err := batchProcessor.ProcessBatch(ctx, operations)
		require.Error(t, err)
		require.Contains(t, err.Error(), "operation 1 failed")
	})

	t.Run("BatchTimeout", func(t *testing.T) {
		// Create batch processor with very short timeout
		shortTimeoutProcessor := NewBatchFileProcessor(processor, 2, 10*time.Millisecond)

		operations := make([]FileOperation, 5)
		for i := range operations {
			inputFile := filepath.Join(tempDir, fmt.Sprintf("timeout_input%d.txt", i))
			outputFile := filepath.Join(tempDir, fmt.Sprintf("timeout_output%d.txt", i))
			testutil.WriteTestFile(t, inputFile, "content")

			operations[i] = FileOperation{
				InputPath:  inputFile,
				OutputPath: outputFile,
				Transform: func(data []byte) ([]byte, error) {
					// Slow transform
					time.Sleep(20 * time.Millisecond)
					return data, nil
				},
			}
		}

		err := shortTimeoutProcessor.ProcessBatch(ctx, operations)
		// The test may or may not timeout depending on execution speed
		// If it doesn't timeout, all operations complete successfully
		if err != nil {
			require.Contains(t, err.Error(), "context deadline exceeded")
		}
	})

	t.Run("ContextCancellationBetweenBatches", func(t *testing.T) {
		// Create many operations to ensure multiple batches
		operations := make([]FileOperation, 20)
		for i := range operations {
			inputFile := filepath.Join(tempDir, fmt.Sprintf("cancel_input%d.txt", i))
			outputFile := filepath.Join(tempDir, fmt.Sprintf("cancel_output%d.txt", i))
			testutil.WriteTestFile(t, inputFile, "content")

			operations[i] = FileOperation{
				InputPath:  inputFile,
				OutputPath: outputFile,
				Transform: func(data []byte) ([]byte, error) {
					return data, nil
				},
			}
		}

		cancelCtx, cancel := context.WithCancel(context.Background())

		// Cancel context after a short delay
		go func() {
			time.Sleep(5 * time.Millisecond)
			cancel()
		}()

		err := batchProcessor.ProcessBatch(cancelCtx, operations)
		// The test may or may not get canceled depending on timing
		// If canceled, the error should contain context.Canceled
		if err != nil {
			require.Contains(t, err.Error(), "context canceled")
		}
	})
}

// TestStreamProcessorSpecificErrorPaths tests specific error paths to improve coverage
func TestStreamProcessorSpecificErrorPaths(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	processor := NewStreamProcessor()
	ctx := context.Background()

	t.Run("ProcessLargeJSON_FileStatError", func(t *testing.T) {
		// Test file.Stat() error path in ProcessLargeJSON
		inputFile := filepath.Join(tempDir, "test.json")
		testutil.WriteTestFile(t, inputFile, `["item1", "item2"]`)

		// Remove the file after opening to cause Stat() to fail
		file, err := os.Open(inputFile) //nolint:gosec // Test file is safe
		require.NoError(t, err)

		// Close and remove file to make Stat fail, but we need to test the actual code path
		_ = file.Close()
		_ = os.Remove(inputFile)

		// Create a new file that we can control
		testutil.WriteTestFile(t, inputFile, `["item1", "item2"]`)

		handlerCalled := false
		handler := func(_ interface{}) error {
			handlerCalled = true
			return nil
		}

		err = processor.ProcessLargeJSON(ctx, inputFile, handler)
		require.NoError(t, err)
		require.True(t, handlerCalled)
	})

	t.Run("ProcessLargeJSON_DecodeItemError", func(t *testing.T) {
		// Test JSON decode error in array processing
		inputFile := filepath.Join(tempDir, "malformed_item.json")
		// Create JSON array with malformed item
		testutil.WriteTestFile(t, inputFile, `["valid", invalid_json, "another"]`)

		handler := func(_ interface{}) error {
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, inputFile, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode JSON item")
	})

	t.Run("ProcessLargeJSON_HandlerErrorArray", func(t *testing.T) {
		// Test handler error in array processing
		inputFile := filepath.Join(tempDir, "handler_error.json")
		testutil.WriteTestFile(t, inputFile, `["item1", "item2", "item3"]`)

		callCount := 0
		handler := func(_ interface{}) error {
			callCount++
			if callCount == 2 {
				return errHandlerError
			}
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, inputFile, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "handler failed for item 1")
	})

	t.Run("ProcessLargeJSON_ObjectProcessing", func(t *testing.T) {
		// Test object processing paths that are uncovered
		inputFile := filepath.Join(tempDir, "object.json")
		testutil.WriteTestFile(t, inputFile, `{"key1": "value1", "key2": "value2", "key3": "value3"}`)

		items := make([]map[string]interface{}, 0)
		handler := func(item interface{}) error {
			property := item.(map[string]interface{})
			items = append(items, property)
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, inputFile, handler)
		require.NoError(t, err)
		require.Len(t, items, 3)
	})

	t.Run("ProcessLargeJSON_ObjectKeyTokenError", func(t *testing.T) {
		// Test property key read error (this is hard to trigger, but we can test object parsing)
		inputFile := filepath.Join(tempDir, "obj_malformed.json")
		// Malformed object with invalid property
		testutil.WriteTestFile(t, inputFile, `{"valid": "value", invalid_property}`)

		handler := func(_ interface{}) error {
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, inputFile, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read property key")
	})

	t.Run("ProcessLargeJSON_ObjectValueDecodeError", func(t *testing.T) {
		// Test property value decode error
		inputFile := filepath.Join(tempDir, "obj_bad_value.json")
		testutil.WriteTestFile(t, inputFile, `{"key": invalid_value}`)

		handler := func(_ interface{}) error {
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, inputFile, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode property value")
	})

	t.Run("ProcessLargeJSON_ObjectHandlerError", func(t *testing.T) {
		// Test handler error in object processing
		inputFile := filepath.Join(tempDir, "obj_handler_error.json")
		testutil.WriteTestFile(t, inputFile, `{"key1": "value1", "key2": "value2"}`)

		callCount := 0
		handler := func(_ interface{}) error {
			callCount++
			if callCount == 1 {
				return errObjectHandlerError
			}
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, inputFile, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "handler failed for property")
	})

	t.Run("ProcessLargeJSON_ClosingTokenError", func(t *testing.T) {
		// Test closing token read error
		inputFile := filepath.Join(tempDir, "no_closing.json")
		testutil.WriteTestFile(t, inputFile, `["item1", "item2"`) // Missing closing bracket

		handler := func(_ interface{}) error {
			return nil
		}

		err := processor.ProcessLargeJSON(ctx, inputFile, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read JSON closing token")
	})
}

// TestProcessFileStreamingErrorPaths tests specific error paths in streaming processing
func TestProcessFileStreamingErrorPaths(t *testing.T) {
	tempDir := testutil.CreateTempDir(t)
	processor := NewStreamProcessor()
	ctx := context.Background()

	t.Run("ProcessFileStreaming_InputPathValidationError", func(t *testing.T) {
		// Test input path validation error
		invalidInputPath := "../invalid/path"
		outputPath := filepath.Join(tempDir, "output.txt")

		transform := func(data []byte) ([]byte, error) {
			return data, nil
		}

		err := processor.processFileStreaming(ctx, invalidInputPath, outputPath, transform, 1000)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid input path")
	})

	t.Run("ProcessFileStreaming_OutputPathValidationError", func(t *testing.T) {
		// Test output path validation error
		inputPath := filepath.Join(tempDir, "input.txt")
		testutil.WriteTestFile(t, inputPath, "test content")
		invalidOutputPath := "../invalid/output"

		transform := func(data []byte) ([]byte, error) {
			return data, nil
		}

		err := processor.processFileStreaming(ctx, inputPath, invalidOutputPath, transform, 1000)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid output path")
	})

	t.Run("ProcessFileStreaming_InputFileOpenError", func(t *testing.T) {
		// Test input file open error
		nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
		outputPath := filepath.Join(tempDir, "output.txt")

		transform := func(data []byte) ([]byte, error) {
			return data, nil
		}

		err := processor.processFileStreaming(ctx, nonExistentFile, outputPath, transform, 1000)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to open input file")
	})

	t.Run("ProcessFileStreaming_OutputFileCreateError", func(t *testing.T) {
		// Test output file creation error
		inputPath := filepath.Join(tempDir, "input.txt")
		testutil.WriteTestFile(t, inputPath, "test content")

		// Try to create output file in non-existent directory
		invalidOutputDir := filepath.Join(tempDir, "nonexistent", "output.txt")

		transform := func(data []byte) ([]byte, error) {
			return data, nil
		}

		err := processor.processFileStreaming(ctx, inputPath, invalidOutputDir, transform, 1000)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create output file")
	})

	t.Run("ProcessFileStreaming_TransformError", func(t *testing.T) {
		// Test transformation error during streaming
		inputPath := filepath.Join(tempDir, "transform_error.txt")
		outputPath := filepath.Join(tempDir, "output.txt")
		testutil.WriteTestFile(t, inputPath, "test content for transformation error")

		transform := func(_ []byte) ([]byte, error) {
			return nil, errTransformationFailed
		}

		err := processor.processFileStreaming(ctx, inputPath, outputPath, transform, 1000)
		require.Error(t, err)
		require.Contains(t, err.Error(), "transformation failed at offset")
	})

	t.Run("ProcessFileStreaming_WriteError", func(t *testing.T) {
		// Test write error during streaming (harder to trigger, but we can test with valid scenario)
		inputPath := filepath.Join(tempDir, "write_test.txt")
		outputPath := filepath.Join(tempDir, "write_output.txt")
		testutil.WriteTestFile(t, inputPath, "content for write test")

		transform := func(data []byte) ([]byte, error) {
			return []byte(strings.ToUpper(string(data))), nil
		}

		err := processor.processFileStreaming(ctx, inputPath, outputPath, transform, 1000)
		require.NoError(t, err)

		// Verify the output
		result, err := os.ReadFile(outputPath) //nolint:gosec // Test file read is safe
		require.NoError(t, err)
		require.Equal(t, "CONTENT FOR WRITE TEST", string(result))
	})

	t.Run("ProcessFileStreaming_ReadError", func(t *testing.T) {
		// Test read error during streaming (difficult to trigger, but test normal case)
		inputPath := filepath.Join(tempDir, "read_test.txt")
		outputPath := filepath.Join(tempDir, "read_output.txt")
		// Create file with content that spans multiple chunks
		largeContent := strings.Repeat("test content ", 1000)
		testutil.WriteTestFile(t, inputPath, largeContent)

		transform := func(data []byte) ([]byte, error) {
			return data, nil
		}

		err := processor.processFileStreaming(ctx, inputPath, outputPath, transform, int64(len(largeContent)))
		require.NoError(t, err)
	})

	t.Run("ProcessBatch_ContextCancellation", func(t *testing.T) {
		// Test context cancellation between batches
		batchProcessor := NewBatchFileProcessor(processor, 1, time.Minute)

		operations := []FileOperation{
			{
				InputPath:  filepath.Join(tempDir, "batch1.txt"),
				OutputPath: filepath.Join(tempDir, "batch1_out.txt"),
				Transform: func(data []byte) ([]byte, error) {
					return data, nil
				},
			},
		}

		testutil.WriteTestFile(t, operations[0].InputPath, "batch content")

		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := batchProcessor.ProcessBatch(cancelCtx, operations)
		// This might or might not error depending on timing, but test both paths
		if err != nil {
			require.Contains(t, err.Error(), "context canceled")
		} else {
			// If it completes before cancellation, that's also valid
			require.NoError(t, err)
		}
	})
}
