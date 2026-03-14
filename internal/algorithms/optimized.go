// Package algorithms provides optimized data processing algorithms for the broadcast system.
package algorithms

import (
	"bytes"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mrz1836/go-broadcast/internal/pool"
)

// BinaryDetectionConfig configures binary file detection
type BinaryDetectionConfig struct {
	MaxSampleSize     int     // Maximum bytes to sample for detection
	NullByteThreshold int     // Number of null bytes to trigger binary classification
	NonPrintableRatio float64 // Ratio of non-printable characters to trigger binary
	UseQuickDetection bool    // Use quick detection for large files
}

// DefaultBinaryDetectionConfig returns sensible defaults
func DefaultBinaryDetectionConfig() BinaryDetectionConfig {
	return BinaryDetectionConfig{
		MaxSampleSize:     8192, // 8KB sample
		NullByteThreshold: 1,    // Any null byte triggers binary
		NonPrintableRatio: 0.3,  // 30% non-printable = binary
		UseQuickDetection: true,
	}
}

// IsBinaryOptimized performs optimized binary file detection with early exit patterns
func IsBinaryOptimized(content []byte) bool {
	return IsBinaryOptimizedWithConfig(content, DefaultBinaryDetectionConfig())
}

// IsBinaryOptimizedWithConfig performs binary detection with custom configuration
func IsBinaryOptimizedWithConfig(content []byte, config BinaryDetectionConfig) bool {
	if len(content) == 0 {
		return false
	}

	// Determine sample size
	sampleSize := len(content)
	if sampleSize > config.MaxSampleSize {
		sampleSize = config.MaxSampleSize
	}

	sample := content[:sampleSize]

	// Quick detection: check for null bytes first (most reliable indicator)
	nullCount := 0
	for i := 0; i < sampleSize; i++ {
		if sample[i] == 0 {
			nullCount++
			// Early exit: if we find enough null bytes, it's binary
			if nullCount >= config.NullByteThreshold {
				return true
			}
		}
	}

	// If quick detection is enabled and file is large, use simplified heuristics
	if config.UseQuickDetection && len(content) > config.MaxSampleSize {
		return checkBinaryHeuristics(sample, config)
	}

	// Full analysis for smaller files or when quick detection is disabled
	return analyzeBinaryContent(sample, config)
}

// checkBinaryHeuristics performs quick binary detection for large files
func checkBinaryHeuristics(sample []byte, config BinaryDetectionConfig) bool {
	// Check for common binary file signatures
	if hasBinarySignature(sample) {
		return true
	}

	// Count non-printable characters (excluding common whitespace)
	nonPrintable := 0
	for _, b := range sample {
		if !isPrintableOrWhitespace(b) {
			nonPrintable++
		}
	}

	ratio := float64(nonPrintable) / float64(len(sample))
	return ratio > config.NonPrintableRatio
}

// analyzeBinaryContent performs thorough binary content analysis
func analyzeBinaryContent(sample []byte, config BinaryDetectionConfig) bool {
	if len(sample) == 0 {
		return false
	}

	// Track character distribution
	var charCounts [256]int
	controlChars := 0
	highBitChars := 0

	for _, b := range sample {
		charCounts[b]++

		// Count control characters (excluding common whitespace)
		if b < 32 && b != '\t' && b != '\n' && b != '\r' && b != '\f' {
			controlChars++
		}

		// Count high-bit characters
		if b > 127 {
			highBitChars++
		}
	}

	sampleLen := len(sample)

	// Check various binary indicators
	controlRatio := float64(controlChars) / float64(sampleLen)
	highBitRatio := float64(highBitChars) / float64(sampleLen)

	// Binary if too many control characters
	if controlRatio > config.NonPrintableRatio {
		return true
	}

	// Binary if too many high-bit characters (unless it's valid UTF-8-like)
	if highBitRatio > 0.5 {
		return true
	}

	// Check for excessive character diversity (common in binary files)
	uniqueChars := 0
	for _, count := range charCounts {
		if count > 0 {
			uniqueChars++
		}
	}

	diversityRatio := float64(uniqueChars) / float64(sampleLen)
	if diversityRatio > 0.7 && sampleLen > 100 {
		return true
	}

	return false
}

// hasBinarySignature checks for common binary file signatures
func hasBinarySignature(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// Common binary signatures
	signatures := [][]byte{
		{0xFF, 0xD8, 0xFF},       // JPEG
		{0x89, 0x50, 0x4E, 0x47}, // PNG
		{0x47, 0x49, 0x46, 0x38}, // GIF
		{0x50, 0x4B, 0x03, 0x04}, // ZIP/DOCX/etc
		{0x25, 0x50, 0x44, 0x46}, // PDF
		{0x7F, 0x45, 0x4C, 0x46}, // ELF
		{0x4D, 0x5A},             // PE/EXE
		{0xCA, 0xFE, 0xBA, 0xBE}, // Java class
		{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O
	}

	for _, sig := range signatures {
		if len(data) >= len(sig) && bytes.HasPrefix(data, sig) {
			return true
		}
	}

	return false
}

// isPrintableOrWhitespace checks if a byte is printable or common whitespace
func isPrintableOrWhitespace(b byte) bool {
	return (b >= 32 && b <= 126) || // ASCII printable
		b == '\t' || b == '\n' || b == '\r' || b == '\f' // Common whitespace
}

// DiffConfig configures diff operations
type DiffConfig struct {
	MaxDiffSize   int  // Maximum diff size to compute
	UseBufferPool bool // Use buffer pool for temporary allocations
	EarlyExit     bool // Exit early if diff would be too large
	ChunkSize     int  // Size of chunks for large file processing
}

// DefaultDiffConfig returns sensible defaults for diff operations
func DefaultDiffConfig() DiffConfig {
	return DiffConfig{
		MaxDiffSize:   1024 * 1024, // 1MB max diff
		UseBufferPool: true,
		EarlyExit:     true,
		ChunkSize:     4096,
	}
}

// DiffOptimized performs optimized diff computation with early exit patterns
func DiffOptimized(a, b []byte, maxDiff int) ([]byte, bool) {
	config := DefaultDiffConfig()
	config.MaxDiffSize = maxDiff
	return DiffOptimizedWithConfig(a, b, config)
}

// DiffOptimizedWithConfig performs diff computation with custom configuration
func DiffOptimizedWithConfig(a, b []byte, config DiffConfig) ([]byte, bool) {
	// Quick equality check
	if bytes.Equal(a, b) {
		return nil, true
	}

	// Early exit if size difference exceeds threshold
	if config.EarlyExit {
		sizeDiff := abs(len(a) - len(b))
		if sizeDiff > config.MaxDiffSize {
			return nil, false
		}
	}

	// Estimate diff size
	estimatedDiffSize := estimateDiffSize(a, b)
	if config.EarlyExit && estimatedDiffSize > config.MaxDiffSize {
		return nil, false
	}

	// Use buffer pool for diff result
	var buf *bytes.Buffer
	if config.UseBufferPool {
		buf = pool.GetBuffer(estimatedDiffSize)
		defer pool.PutBuffer(buf)
	} else {
		buf = bytes.NewBuffer(make([]byte, 0, estimatedDiffSize))
	}

	// Perform optimized diff computation
	success := computeOptimizedDiff(a, b, buf, config)
	if !success {
		return nil, false
	}

	// Return copy of result
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, true
}

// estimateDiffSize provides a rough estimate of diff size
func estimateDiffSize(a, b []byte) int {
	// Simple heuristic: assume worst case is sum of both files
	// In practice, diffs are usually much smaller
	maxSize := len(a) + len(b)

	// For large files, use a more conservative estimate
	if maxSize > 100*1024 { // 100KB
		return maxSize / 4 // Assume 25% diff
	}

	return maxSize
}

// computeOptimizedDiff computes the actual diff using an optimized algorithm
func computeOptimizedDiff(a, b []byte, buf *bytes.Buffer, config DiffConfig) bool {
	// Simple line-based diff for text content
	if isMostlyText(a) && isMostlyText(b) {
		return computeLineDiff(a, b, buf, config)
	}

	// Byte-based diff for binary content
	return computeByteDiff(a, b, buf, config)
}

// computeLineDiff performs line-based diff computation
func computeLineDiff(a, b []byte, buf *bytes.Buffer, config DiffConfig) bool {
	aLines := bytes.Split(a, []byte{'\n'})
	bLines := bytes.Split(b, []byte{'\n'})

	// Use a simplified diff algorithm suitable for most cases
	return computeLCS(aLines, bLines, buf, config)
}

// computeByteDiff performs byte-based diff computation
func computeByteDiff(a, b []byte, buf *bytes.Buffer, config DiffConfig) bool {
	// For binary files, use a simple block-based approach
	minLen := minInt(len(a), len(b))
	maxLen := maxInt(len(a), len(b))

	// Check if diff would be too large
	if maxLen-minLen > config.MaxDiffSize {
		return false
	}

	// Find common prefix
	commonPrefix := 0
	for commonPrefix < minLen && a[commonPrefix] == b[commonPrefix] {
		commonPrefix++
	}

	// Find common suffix
	commonSuffix := 0
	aIdx, bIdx := len(a)-1, len(b)-1
	for commonSuffix < minLen-commonPrefix && aIdx >= commonPrefix && bIdx >= commonPrefix && a[aIdx] == b[bIdx] {
		commonSuffix++
		aIdx--
		bIdx--
	}

	// Generate diff representation
	if commonPrefix > 0 {
		fmt.Fprintf(buf, "= %d bytes\n", commonPrefix)
	}

	if commonPrefix < len(a)-commonSuffix {
		deletedLen := len(a) - commonPrefix - commonSuffix
		fmt.Fprintf(buf, "- %d bytes from %d\n", deletedLen, commonPrefix)
	}

	if commonPrefix < len(b)-commonSuffix {
		addedLen := len(b) - commonPrefix - commonSuffix
		fmt.Fprintf(buf, "+ %d bytes at %d\n", addedLen, commonPrefix)
	}

	if commonSuffix > 0 {
		fmt.Fprintf(buf, "= %d bytes (suffix)\n", commonSuffix)
	}

	return buf.Len() <= config.MaxDiffSize
}

// computeLCS computes Longest Common Subsequence for line-based diff
func computeLCS(a, b [][]byte, buf *bytes.Buffer, config DiffConfig) bool {
	m, n := len(a), len(b)

	// For large files, use a simplified approach
	if m*n > 10000 { // Avoid O(m*n) memory for very large files
		return computeSimpleDiff(a, b, buf, config)
	}

	// Create LCS table
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}

	// Fill LCS table
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if bytes.Equal(a[i-1], b[j-1]) {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else {
				lcs[i][j] = maxInt(lcs[i-1][j], lcs[i][j-1])
			}
		}
	}

	// Generate diff from LCS
	return generateDiffFromLCS(a, b, lcs, buf, config)
}

// generateDiffFromLCS generates diff output from LCS table
func generateDiffFromLCS(a, b [][]byte, lcs [][]int, buf *bytes.Buffer, config DiffConfig) bool {
	i, j := len(a), len(b)

	for i > 0 || j > 0 {
		if buf.Len() > config.MaxDiffSize {
			return false
		}

		if i > 0 && j > 0 && bytes.Equal(a[i-1], b[j-1]) {
			// Lines are equal
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			// Line added
			fmt.Fprintf(buf, "+%s\n", string(b[j-1]))
			j--
		} else if i > 0 {
			// Line deleted
			fmt.Fprintf(buf, "-%s\n", string(a[i-1]))
			i--
		}
	}

	return true
}

// computeSimpleDiff computes a simplified diff for large files
func computeSimpleDiff(a, b [][]byte, buf *bytes.Buffer, config DiffConfig) bool {
	// Simple approach: find blocks of changes
	aIdx, bIdx := 0, 0

	for aIdx < len(a) || bIdx < len(b) {
		if buf.Len() > config.MaxDiffSize {
			return false
		}

		if aIdx < len(a) && bIdx < len(b) && bytes.Equal(a[aIdx], b[bIdx]) {
			// Lines match, continue
			aIdx++
			bIdx++
		} else {
			// Find next matching line within reasonable distance
			matchFound := false
			searchLimit := minInt(20, minInt(len(a)-aIdx, len(b)-bIdx))

			for offset := 1; offset < searchLimit && !matchFound; offset++ {
				if aIdx+offset < len(a) && bIdx < len(b) && bytes.Equal(a[aIdx+offset], b[bIdx]) {
					// Found match after deleting some lines
					for i := 0; i < offset; i++ {
						fmt.Fprintf(buf, "-%s\n", string(a[aIdx+i]))
					}
					aIdx += offset
					matchFound = true
				} else if bIdx+offset < len(b) && aIdx < len(a) && bytes.Equal(a[aIdx], b[bIdx+offset]) {
					// Found match after adding some lines
					for i := 0; i < offset; i++ {
						fmt.Fprintf(buf, "+%s\n", string(b[bIdx+i]))
					}
					bIdx += offset
					matchFound = true
				}
			}

			if !matchFound {
				// No nearby match found, output as change
				if aIdx < len(a) {
					fmt.Fprintf(buf, "-%s\n", string(a[aIdx]))
					aIdx++
				}
				if bIdx < len(b) {
					fmt.Fprintf(buf, "+%s\n", string(b[bIdx]))
					bIdx++
				}
			}
		}
	}

	return true
}

// isMostlyText checks if content is mostly text
func isMostlyText(content []byte) bool {
	if len(content) == 0 {
		return true
	}

	// Sample check for text content
	sampleSize := minInt(1024, len(content))
	sample := content[:sampleSize]

	nonPrintable := 0
	for _, b := range sample {
		if !isPrintableOrWhitespace(b) {
			nonPrintable++
		}
	}

	return float64(nonPrintable)/float64(sampleSize) < 0.1
}

// BatchProcessor optimizes batch operations with configurable batching
type BatchProcessor struct {
	batchSize int
	processor func([]interface{}) error
	items     []interface{}
	mu        sync.Mutex

	// Configuration
	autoFlush     bool
	flushInterval time.Duration
	maxWaitTime   time.Duration

	// State
	lastFlush time.Time
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// BatchProcessorConfig configures batch processing behavior
type BatchProcessorConfig struct {
	BatchSize     int           // Maximum items per batch
	AutoFlush     bool          // Automatically flush batches
	FlushInterval time.Duration // How often to auto-flush
	MaxWaitTime   time.Duration // Maximum time to wait before forcing flush
}

// DefaultBatchProcessorConfig returns sensible defaults
func DefaultBatchProcessorConfig() BatchProcessorConfig {
	return BatchProcessorConfig{
		BatchSize:     100,
		AutoFlush:     true,
		FlushInterval: time.Second * 5,
		MaxWaitTime:   time.Second * 30,
	}
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(processor func([]interface{}) error, config BatchProcessorConfig) *BatchProcessor {
	bp := &BatchProcessor{
		batchSize:     config.BatchSize,
		processor:     processor,
		items:         make([]interface{}, 0, config.BatchSize),
		autoFlush:     config.AutoFlush,
		flushInterval: config.FlushInterval,
		maxWaitTime:   config.MaxWaitTime,
		lastFlush:     time.Now(),
		stopChan:      make(chan struct{}),
	}

	if config.AutoFlush {
		bp.wg.Add(1)
		go bp.autoFlushWorker()
	}

	return bp
}

// Add adds an item to the batch
func (bp *BatchProcessor) Add(item interface{}) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.items = append(bp.items, item)

	if len(bp.items) >= bp.batchSize {
		return bp.flush()
	}

	return nil
}

// AddBatch adds multiple items to the batch
func (bp *BatchProcessor) AddBatch(items []interface{}) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	for _, item := range items {
		bp.items = append(bp.items, item)

		if len(bp.items) >= bp.batchSize {
			if err := bp.flush(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Flush processes all pending items
func (bp *BatchProcessor) Flush() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	return bp.flush()
}

// Stop stops the batch processor
func (bp *BatchProcessor) Stop() error {
	if bp.autoFlush {
		close(bp.stopChan)
		bp.wg.Wait()
	}

	return bp.Flush()
}

// Stats returns batch processor statistics
func (bp *BatchProcessor) Stats() BatchProcessorStats {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	return BatchProcessorStats{
		PendingItems:   len(bp.items),
		LastFlushTime:  bp.lastFlush,
		TimeSinceFlush: time.Since(bp.lastFlush),
	}
}

// BatchProcessorStats contains batch processor statistics
type BatchProcessorStats struct {
	PendingItems   int           `json:"pending_items"`
	LastFlushTime  time.Time     `json:"last_flush_time"`
	TimeSinceFlush time.Duration `json:"time_since_flush"`
}

// flush processes pending items (must be called with lock held)
func (bp *BatchProcessor) flush() error {
	if len(bp.items) == 0 {
		return nil
	}

	itemsCopy := make([]interface{}, len(bp.items))
	copy(itemsCopy, bp.items)
	bp.items = bp.items[:0]
	bp.lastFlush = time.Now()

	// Process outside of lock to avoid blocking other operations.
	// Use a closure with defer so the lock is always re-acquired, even if processor panics (CWE-667).
	var err error
	func() {
		bp.mu.Unlock()
		defer bp.mu.Lock()
		err = bp.processor(itemsCopy)
	}()

	return err
}

// autoFlushWorker automatically flushes batches based on time intervals
func (bp *BatchProcessor) autoFlushWorker() {
	defer bp.wg.Done()

	ticker := time.NewTicker(bp.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bp.mu.Lock()
			shouldFlush := len(bp.items) > 0 &&
				(time.Since(bp.lastFlush) > bp.flushInterval ||
					time.Since(bp.lastFlush) > bp.maxWaitTime)

			if shouldFlush {
				if err := bp.flush(); err != nil {
					log.Printf("batch processor: periodic flush failed: %v", err)
				}
			}
			bp.mu.Unlock()

		case <-bp.stopChan:
			// Final flush before stopping
			if err := bp.Flush(); err != nil {
				log.Printf("batch processor: stop-time flush failed: %v", err)
			}
			return
		}
	}
}

// Helper functions

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
