package io

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
)

var (
	errTransformFailed = errors.New("transform failed")
	errHandlerFailed   = errors.New("handler failed")
	errTransformError  = errors.New("transform error")
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
	tempDir := t.TempDir()

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

			err := os.WriteFile(inputFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			// Create processor
			var processor *StreamProcessor
			if tt.forceStreaming {
				processor = NewStreamProcessorWithConfig(DefaultChunkSize, 100, DefaultBufferTimeout) // Low threshold
			} else {
				processor = NewStreamProcessor()
			}

			// Process file
			ctx := context.Background()
			err = processor.ProcessFile(ctx, inputFile, outputFile, tt.transform)

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
	tempDir := t.TempDir()
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
		err := os.WriteFile(inputFile, []byte("test"), 0o600)
		require.NoError(t, err)

		// Force the processor to use the invalid path internally
		err = processor.processFileInMemory(ctx, invalidPath, outputFile, func(data []byte) ([]byte, error) {
			return data, nil
		}, 4)

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid input path")
	})

	t.Run("InvalidOutputPath", func(t *testing.T) {
		inputFile := filepath.Join(tempDir, "input.txt")
		err := os.WriteFile(inputFile, []byte("test"), 0o600)
		require.NoError(t, err)

		invalidOutputPath := "../../../tmp/output.txt"

		err = processor.ProcessFile(ctx, inputFile, invalidOutputPath, func(data []byte) ([]byte, error) {
			return data, nil
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid output path")
	})
}

func TestStreamProcessorProcessFileContext(t *testing.T) {
	tempDir := t.TempDir()
	processor := NewStreamProcessorWithConfig(1024, 100, DefaultBufferTimeout) // Force streaming

	// Create a large file
	inputFile := filepath.Join(tempDir, "large.txt")
	outputFile := filepath.Join(tempDir, "output.txt")
	largeContent := strings.Repeat("test content that will trigger streaming mode ", 1000)
	err := os.WriteFile(inputFile, []byte(largeContent), 0o600)
	require.NoError(t, err)

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
	tempDir := t.TempDir()
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
		err = os.WriteFile(jsonFile, data, 0o600)
		require.NoError(t, err)

		// Process JSON
		var processedItems []interface{}
		handler := func(item interface{}) error {
			processedItems = append(processedItems, item)
			return nil
		}

		err = processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.NoError(t, err)
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
		err = os.WriteFile(jsonFile, data, 0o600)
		require.NoError(t, err)

		// Process JSON
		var processedProperties []map[string]interface{}
		handler := func(item interface{}) error {
			if prop, ok := item.(map[string]interface{}); ok {
				processedProperties = append(processedProperties, prop)
			}
			return nil
		}

		err = processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.NoError(t, err)
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
		err := os.WriteFile(jsonFile, []byte("invalid json content"), 0o600)
		require.NoError(t, err)

		handler := func(_ interface{}) error {
			return nil
		}

		err = processor.ProcessLargeJSON(ctx, jsonFile, handler)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read JSON opening token")
	})

	t.Run("InvalidJSONStructure", func(t *testing.T) {
		jsonFile := filepath.Join(tempDir, "invalid_structure.json")
		err := os.WriteFile(jsonFile, []byte("\"just a string\""), 0o600)
		require.NoError(t, err)

		handler := func(_ interface{}) error {
			return nil
		}

		err = processor.ProcessLargeJSON(ctx, jsonFile, handler)
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
		err = os.WriteFile(jsonFile, data, 0o600)
		require.NoError(t, err)

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
		err = os.WriteFile(jsonFile, data, 0o600)
		require.NoError(t, err)

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
	tempDir := t.TempDir()
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

		err := os.WriteFile(inputFile, []byte("test content"), 0o600)
		require.NoError(t, err)

		operations := []FileOperation{
			{
				InputPath:  inputFile,
				OutputPath: outputFile,
				Transform: func(data []byte) ([]byte, error) {
					return []byte(strings.ToUpper(string(data))), nil
				},
			},
		}

		err = bfp.ProcessBatch(ctx, operations)
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
			err := os.WriteFile(inputFile, []byte(content), 0o600)
			require.NoError(t, err)

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

		err := os.WriteFile(inputFile, []byte("test"), 0o600)
		require.NoError(t, err)

		operations := []FileOperation{
			{
				InputPath:  inputFile,
				OutputPath: outputFile,
				Transform: func(_ []byte) ([]byte, error) {
					return nil, errTransformError
				},
			},
		}

		err = bfp.ProcessBatch(ctx, operations)
		require.Error(t, err)
		require.Contains(t, err.Error(), "batch processing failed")
		require.Contains(t, err.Error(), "transform error")
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Create a slow operation
		inputFile := filepath.Join(tempDir, "slow_input.txt")
		outputFile := filepath.Join(tempDir, "slow_output.txt")

		err := os.WriteFile(inputFile, []byte("test"), 0o600)
		require.NoError(t, err)

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

		err = bfp.ProcessBatch(ctx, operations)
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
	})
}

func TestStreamProcessorStats(t *testing.T) {
	processor := NewStreamProcessor()
	ctx := context.Background()
	tempDir := t.TempDir()

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
	err := os.WriteFile(smallFile, []byte("small content"), 0o600)
	require.NoError(t, err)

	err = processor.ProcessFile(ctx, smallFile, smallOutput, func(data []byte) ([]byte, error) {
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
	err = os.WriteFile(largeFile, []byte(largeContent), 0o600)
	require.NoError(t, err)

	err = streamingProcessor.ProcessFile(ctx, largeFile, largeOutput, func(data []byte) ([]byte, error) {
		return data, nil
	})
	require.NoError(t, err)

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
	tempDir := t.TempDir()

	// Create test files
	const numFiles = 10
	inputFiles := make([]string, numFiles)
	outputFiles := make([]string, numFiles)

	for i := 0; i < numFiles; i++ {
		inputFile := filepath.Join(tempDir, fmt.Sprintf("concurrent_input_%d.txt", i))
		outputFile := filepath.Join(tempDir, fmt.Sprintf("concurrent_output_%d.txt", i))

		content := fmt.Sprintf("content for file %d", i)
		err := os.WriteFile(inputFile, []byte(content), 0o600)
		require.NoError(t, err)

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
	tempDir := t.TempDir()
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

		err := os.WriteFile(inputFile, []byte("test content"), 0o600)
		require.NoError(t, err)

		// Transform that returns empty slice
		err = processor.ProcessFile(ctx, inputFile, outputFile, func(_ []byte) ([]byte, error) {
			return []byte{}, nil
		})
		require.NoError(t, err)

		// Verify empty output
		result, err := os.ReadFile(outputFile) //nolint:gosec // Reading test file is safe
		require.NoError(t, err)
		require.Empty(t, result)
	})
}
