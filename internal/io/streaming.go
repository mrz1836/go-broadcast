package io //nolint:revive,nolintlint // internal package, name conflict intentional

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mrz1836/go-broadcast/internal/pool"
)

// Stream processing errors
var (
	ErrInvalidJSONStructure = errors.New("expected JSON array or object")
	ErrPathTraversal        = errors.New("path traversal detected")
	ErrNullByteInPath       = errors.New("null byte detected in path")
)

// validatePath validates file paths to prevent directory traversal attacks
func validatePath(path string) error {
	// Clean the path to resolve . and .. elements
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts - looking for .. after cleaning
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("%w: %s", ErrPathTraversal, path)
	}

	// Check for null bytes which can be used to bypass filters
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("%w: %s", ErrNullByteInPath, path)
	}

	return nil
}

// StreamProcessor processes files and data in chunks without loading entire content into memory
type StreamProcessor struct {
	ChunkSize          int           // Size of chunks to read/write (default: 64KB)
	StreamingThreshold int           // File size threshold for streaming vs in-memory (default: 1MB)
	BufferTimeout      time.Duration // Timeout for buffer operations (default: 30s)

	// Statistics for monitoring
	stats struct {
		filesProcessed  int64
		bytesProcessed  int64
		streamingOps    int64
		inMemoryOps     int64
		chunksProcessed int64
		errorCount      int64
	}
}

// StreamingThresholds defines size limits for different processing modes
const (
	DefaultChunkSize          = 64 * 1024        // 64KB chunks - optimal for most file systems
	DefaultStreamingThreshold = 1024 * 1024      // 1MB - switch to streaming above this size
	MaxInMemorySize           = 10 * 1024 * 1024 // 10MB - absolute maximum for in-memory processing
	DefaultBufferTimeout      = 30 * time.Second
)

// NewStreamProcessor creates a new stream processor with optimized defaults
func NewStreamProcessor() *StreamProcessor {
	return &StreamProcessor{
		ChunkSize:          DefaultChunkSize,
		StreamingThreshold: DefaultStreamingThreshold,
		BufferTimeout:      DefaultBufferTimeout,
	}
}

// NewStreamProcessorWithConfig creates a stream processor with custom configuration
func NewStreamProcessorWithConfig(chunkSize, streamingThreshold int, timeout time.Duration) *StreamProcessor {
	// Validate and clamp configuration values
	if chunkSize <= 0 || chunkSize > pool.LargeBufferThreshold {
		chunkSize = DefaultChunkSize
	}
	if streamingThreshold <= 0 || streamingThreshold > MaxInMemorySize {
		streamingThreshold = DefaultStreamingThreshold
	}
	if timeout <= 0 {
		timeout = DefaultBufferTimeout
	}

	return &StreamProcessor{
		ChunkSize:          chunkSize,
		StreamingThreshold: streamingThreshold,
		BufferTimeout:      timeout,
	}
}

// TransformFunc represents a function that transforms data chunks
// It receives input data and returns transformed data and any error
type TransformFunc func([]byte) ([]byte, error)

// ProcessFile processes a file with the given transformation function
// Automatically chooses between streaming and in-memory processing based on file size
func (sp *StreamProcessor) ProcessFile(ctx context.Context, inputPath, outputPath string, transform TransformFunc) error {
	// Get file size to determine processing strategy
	fileInfo, err := os.Stat(inputPath)
	if err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("failed to stat input file %s: %w", inputPath, err)
	}

	fileSize := fileInfo.Size()

	// Choose processing strategy based on file size
	if fileSize <= int64(sp.StreamingThreshold) {
		atomic.AddInt64(&sp.stats.inMemoryOps, 1)
		return sp.processFileInMemory(ctx, inputPath, outputPath, transform, fileSize)
	}
	atomic.AddInt64(&sp.stats.streamingOps, 1)
	return sp.processFileStreaming(ctx, inputPath, outputPath, transform, fileSize)
}

// JSONStreamHandler represents a function that processes individual JSON objects
type JSONStreamHandler func(interface{}) error

// ProcessLargeJSON processes large JSON files without loading the entire content into memory
// This is particularly useful for GitHub API responses with many items (branches, PRs, etc.)
func (sp *StreamProcessor) ProcessLargeJSON(ctx context.Context, inputPath string, handler JSONStreamHandler) error {
	// Validate input path
	if err := validatePath(inputPath); err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("invalid input path: %w", err)
	}

	file, err := os.Open(inputPath) //nolint:gosec // Opening file with user-provided path for streaming processing
	if err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("failed to open JSON file %s: %w", inputPath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Get file size for statistics
	fileInfo, err := file.Stat()
	if err == nil {
		defer func() {
			atomic.AddInt64(&sp.stats.filesProcessed, 1)
			atomic.AddInt64(&sp.stats.bytesProcessed, fileInfo.Size())
		}()
	}

	// Create buffered reader for optimal performance
	reader := bufio.NewReaderSize(file, sp.ChunkSize)
	decoder := json.NewDecoder(reader)

	// Read opening delimiter (array or object)
	token, err := decoder.Token()
	if err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("failed to read JSON opening token: %w", err)
	}

	// Handle both JSON arrays and objects
	var isArray bool
	if delim, ok := token.(json.Delim); ok && delim == '[' {
		isArray = true
	} else if delim, ok := token.(json.Delim); ok && delim == '{' {
		isArray = false
	} else {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("%w: got %T", ErrInvalidJSONStructure, token)
	}

	itemCount := int64(0)

	if isArray {
		// Process array elements one by one
		for decoder.More() {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				atomic.AddInt64(&sp.stats.errorCount, 1)
				return ctx.Err()
			default:
			}

			var item interface{}
			if err := decoder.Decode(&item); err != nil {
				atomic.AddInt64(&sp.stats.errorCount, 1)
				return fmt.Errorf("failed to decode JSON item %d: %w", itemCount, err)
			}

			if err := handler(item); err != nil {
				atomic.AddInt64(&sp.stats.errorCount, 1)
				return fmt.Errorf("handler failed for item %d: %w", itemCount, err)
			}

			itemCount++

			// Yield to other goroutines occasionally to prevent blocking
			if itemCount%100 == 0 {
				runtime.Gosched()
			}
		}
	} else {
		// Process object properties one by one
		for decoder.More() {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				atomic.AddInt64(&sp.stats.errorCount, 1)
				return ctx.Err()
			default:
			}

			// Read property name
			keyToken, err := decoder.Token()
			if err != nil {
				atomic.AddInt64(&sp.stats.errorCount, 1)
				return fmt.Errorf("failed to read property key: %w", err)
			}

			// Read property value
			var value interface{}
			if err := decoder.Decode(&value); err != nil {
				atomic.AddInt64(&sp.stats.errorCount, 1)
				return fmt.Errorf("failed to decode property value: %w", err)
			}

			// Create property object for handler
			property := map[string]interface{}{
				keyToken.(string): value,
			}

			if err := handler(property); err != nil {
				atomic.AddInt64(&sp.stats.errorCount, 1)
				return fmt.Errorf("handler failed for property %s: %w", keyToken, err)
			}

			itemCount++

			// Yield to other goroutines occasionally
			if itemCount%100 == 0 {
				runtime.Gosched()
			}
		}
	}

	// Read closing delimiter
	if _, err := decoder.Token(); err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("failed to read JSON closing token: %w", err)
	}

	atomic.AddInt64(&sp.stats.streamingOps, 1)

	return nil
}

// BatchFileProcessor handles multiple file operations efficiently
type BatchFileProcessor struct {
	processor    *StreamProcessor
	maxBatchSize int
	batchTimeout time.Duration
}

// NewBatchFileProcessor creates a new batch file processor
func NewBatchFileProcessor(processor *StreamProcessor, maxBatchSize int, batchTimeout time.Duration) *BatchFileProcessor {
	if maxBatchSize <= 0 {
		maxBatchSize = 100 // Default batch size
	}
	if batchTimeout <= 0 {
		batchTimeout = 5 * time.Minute // Default timeout
	}

	return &BatchFileProcessor{
		processor:    processor,
		maxBatchSize: maxBatchSize,
		batchTimeout: batchTimeout,
	}
}

// FileOperation represents a single file processing operation
type FileOperation struct {
	InputPath  string
	OutputPath string
	Transform  TransformFunc
}

// ProcessBatch processes multiple files efficiently
func (bfp *BatchFileProcessor) ProcessBatch(ctx context.Context, operations []FileOperation) error {
	if len(operations) == 0 {
		return nil
	}

	// Process in batches to manage memory usage
	for i := 0; i < len(operations); i += bfp.maxBatchSize {
		end := i + bfp.maxBatchSize
		if end > len(operations) {
			end = len(operations)
		}

		batch := operations[i:end]

		// Create batch context with timeout
		batchCtx, cancel := context.WithTimeout(ctx, bfp.batchTimeout)

		// Process batch
		batchErr := bfp.processBatch(batchCtx, batch)
		cancel()

		if batchErr != nil {
			return fmt.Errorf("batch processing failed at batch starting index %d: %w", i, batchErr)
		}

		// Check for context cancellation between batches
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

// processBatch handles a single batch of file operations
func (bfp *BatchFileProcessor) processBatch(ctx context.Context, batch []FileOperation) error {
	for i, op := range batch {
		if err := bfp.processor.ProcessFile(ctx, op.InputPath, op.OutputPath, op.Transform); err != nil {
			return fmt.Errorf("operation %d failed (input: %s, output: %s): %w", i, op.InputPath, op.OutputPath, err)
		}
	}
	return nil
}

// StreamingStats contains statistics about streaming operations
type StreamingStats struct {
	FilesProcessed  int64   `json:"files_processed"`
	BytesProcessed  int64   `json:"bytes_processed"`
	StreamingOps    int64   `json:"streaming_operations"`
	InMemoryOps     int64   `json:"in_memory_operations"`
	ChunksProcessed int64   `json:"chunks_processed"`
	ErrorCount      int64   `json:"error_count"`
	StreamingRatio  float64 `json:"streaming_ratio"` // Percentage of operations that used streaming
}

// GetStats returns current streaming statistics
func (sp *StreamProcessor) GetStats() StreamingStats {
	filesProcessed := atomic.LoadInt64(&sp.stats.filesProcessed)
	bytesProcessed := atomic.LoadInt64(&sp.stats.bytesProcessed)
	streamingOps := atomic.LoadInt64(&sp.stats.streamingOps)
	inMemoryOps := atomic.LoadInt64(&sp.stats.inMemoryOps)
	chunksProcessed := atomic.LoadInt64(&sp.stats.chunksProcessed)
	errorCount := atomic.LoadInt64(&sp.stats.errorCount)

	totalOps := streamingOps + inMemoryOps
	var streamingRatio float64
	if totalOps > 0 {
		streamingRatio = float64(streamingOps) / float64(totalOps) * 100
	}

	return StreamingStats{
		FilesProcessed:  filesProcessed,
		BytesProcessed:  bytesProcessed,
		StreamingOps:    streamingOps,
		InMemoryOps:     inMemoryOps,
		ChunksProcessed: chunksProcessed,
		ErrorCount:      errorCount,
		StreamingRatio:  streamingRatio,
	}
}

// ResetStats resets all streaming statistics to zero
func (sp *StreamProcessor) ResetStats() {
	atomic.StoreInt64(&sp.stats.filesProcessed, 0)
	atomic.StoreInt64(&sp.stats.bytesProcessed, 0)
	atomic.StoreInt64(&sp.stats.streamingOps, 0)
	atomic.StoreInt64(&sp.stats.inMemoryOps, 0)
	atomic.StoreInt64(&sp.stats.chunksProcessed, 0)
	atomic.StoreInt64(&sp.stats.errorCount, 0)
}

// processFileInMemory handles small files entirely in memory for maximum performance
func (sp *StreamProcessor) processFileInMemory(_ context.Context, inputPath, outputPath string, transform TransformFunc, fileSize int64) error {
	// Use buffer pool for efficient memory management
	bufferSize := pool.EstimateBufferSize("file_content", int(fileSize))

	result, err := pool.WithBufferResult[[]byte](bufferSize, func(_ *bytes.Buffer) ([]byte, error) {
		// Validate input path
		if err := validatePath(inputPath); err != nil {
			return nil, fmt.Errorf("invalid input path: %w", err)
		}

		// Read entire file
		content, err := os.ReadFile(inputPath) //nolint:gosec // Reading file with user-provided path for processing
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", inputPath, err)
		}

		// Apply transformation
		transformed, err := transform(content)
		if err != nil {
			return nil, fmt.Errorf("transformation failed: %w", err)
		}

		// Copy result to avoid returning buffer pool memory
		result := make([]byte, len(transformed))
		copy(result, transformed)
		return result, nil
	})
	if err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return err
	}

	// Validate output path
	if err := validatePath(outputPath); err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Write result to output file
	if err := os.WriteFile(outputPath, result, 0o600); err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("failed to write output file %s: %w", outputPath, err)
	}

	atomic.AddInt64(&sp.stats.filesProcessed, 1)
	atomic.AddInt64(&sp.stats.bytesProcessed, fileSize)

	return nil
}

// processFileStreaming handles large files using streaming I/O to minimize memory usage
func (sp *StreamProcessor) processFileStreaming(ctx context.Context, inputPath, outputPath string, transform TransformFunc, _ int64) error {
	// Validate paths
	if err := validatePath(inputPath); err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("invalid input path: %w", err)
	}
	if err := validatePath(outputPath); err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Open input file
	input, err := os.Open(inputPath) //nolint:gosec // Opening file with user-provided path for streaming processing
	if err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("failed to open input file %s: %w", inputPath, err)
	}
	defer func() {
		_ = input.Close()
	}()

	// Create output file
	output, err := os.Create(outputPath) //nolint:gosec // Creating output file with user-provided path
	if err != nil {
		atomic.AddInt64(&sp.stats.errorCount, 1)
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer func() {
		if cerr := output.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Create buffered readers and writers for optimal performance
	reader := bufio.NewReaderSize(input, sp.ChunkSize)
	writer := bufio.NewWriterSize(output, sp.ChunkSize)
	defer func() {
		if ferr := writer.Flush(); ferr != nil && err == nil {
			err = ferr
		}
	}()

	// Process file in chunks
	chunk := make([]byte, sp.ChunkSize)
	var totalBytesProcessed int64

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			atomic.AddInt64(&sp.stats.errorCount, 1)
			return ctx.Err()
		default:
		}

		// Read chunk
		n, readErr := reader.Read(chunk)
		if n > 0 {
			// Apply transformation to chunk
			transformed, transformErr := transform(chunk[:n])
			if transformErr != nil {
				atomic.AddInt64(&sp.stats.errorCount, 1)
				return fmt.Errorf("transformation failed at offset %d: %w", totalBytesProcessed, transformErr)
			}

			// Write transformed chunk
			if _, writeErr := writer.Write(transformed); writeErr != nil {
				atomic.AddInt64(&sp.stats.errorCount, 1)
				return fmt.Errorf("failed to write at offset %d: %w", totalBytesProcessed, writeErr)
			}

			totalBytesProcessed += int64(n)
			atomic.AddInt64(&sp.stats.chunksProcessed, 1)
		}

		// Check for end of file
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			atomic.AddInt64(&sp.stats.errorCount, 1)
			return fmt.Errorf("failed to read at offset %d: %w", totalBytesProcessed, readErr)
		}
	}

	atomic.AddInt64(&sp.stats.filesProcessed, 1)
	atomic.AddInt64(&sp.stats.bytesProcessed, totalBytesProcessed)

	return nil
}
