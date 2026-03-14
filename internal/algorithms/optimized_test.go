package algorithms

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBinaryDetectionConfig(t *testing.T) {
	config := DefaultBinaryDetectionConfig()

	assert.Equal(t, 8192, config.MaxSampleSize)
	assert.Equal(t, 1, config.NullByteThreshold)
	assert.InDelta(t, 0.3, config.NonPrintableRatio, 0.001)
	assert.True(t, config.UseQuickDetection)
}

func TestIsBinaryOptimized(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: false,
		},
		{
			name:     "plain text",
			content:  []byte("Hello, World! This is a plain text file.\n"),
			expected: false,
		},
		{
			name:     "text with newlines",
			content:  []byte("Line 1\nLine 2\nLine 3\n"),
			expected: false,
		},
		{
			name:     "null byte content",
			content:  []byte("Hello\x00World"),
			expected: true,
		},
		{
			name:     "JPEG signature",
			content:  []byte{0xFF, 0xD8, 0xFF, 0xE0}, // Add 4th byte to meet minimum requirement
			expected: true,
		},
		{
			name:     "PNG signature with null byte",
			content:  []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00}, // Added null byte to trigger detection
			expected: true,
		},
		{
			name:     "high control characters",
			content:  bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x04}, 100),
			expected: true,
		},
		{
			name:     "mostly non-printable",
			content:  append([]byte("ABC"), bytes.Repeat([]byte{0x80, 0x81, 0x82}, 100)...),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBinaryOptimized(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBinaryOptimizedWithConfig(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		config   BinaryDetectionConfig
		expected bool
	}{
		{
			name:    "large file with PNG signature triggers quick detection",
			content: append([]byte{0x89, 0x50, 0x4E, 0x47}, bytes.Repeat([]byte{0x20}, 9000)...), // PNG signature + lots of spaces
			config: BinaryDetectionConfig{
				MaxSampleSize:     8192,
				NullByteThreshold: 1,
				NonPrintableRatio: 0.3,
				UseQuickDetection: true,
			},
			expected: true,
		},
		{
			name:    "custom null byte threshold",
			content: []byte("Hello\x00World\x00Test"),
			config: BinaryDetectionConfig{
				MaxSampleSize:     8192,
				NullByteThreshold: 3,
				NonPrintableRatio: 0.3,
				UseQuickDetection: true,
			},
			expected: false, // 2 null bytes < threshold of 3
		},
		{
			name:    "custom non-printable ratio",
			content: append([]byte("ABCDEFGHIJ"), bytes.Repeat([]byte{0x01}, 5)...),
			config: BinaryDetectionConfig{
				MaxSampleSize:     8192,
				NullByteThreshold: 10,
				NonPrintableRatio: 0.5, // 5/15 = 0.33 < 0.5
				UseQuickDetection: false,
			},
			expected: false,
		},
		{
			name:    "small sample size",
			content: append([]byte("TEXT"), bytes.Repeat([]byte{0xFF}, 1000)...),
			config: BinaryDetectionConfig{
				MaxSampleSize:     4,
				NullByteThreshold: 10,
				NonPrintableRatio: 0.3,
				UseQuickDetection: true,
			},
			expected: false, // Only samples "TEXT"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBinaryOptimizedWithConfig(tt.content, tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasBinarySignature(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{
			name:     "JPEG signature",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0}, // Add 4th byte to meet minimum requirement
			expected: true,
		},
		{
			name:     "PNG signature",
			data:     []byte{0x89, 0x50, 0x4E, 0x47},
			expected: true,
		},
		{
			name:     "GIF signature",
			data:     []byte{0x47, 0x49, 0x46, 0x38},
			expected: true,
		},
		{
			name:     "ZIP signature",
			data:     []byte{0x50, 0x4B, 0x03, 0x04},
			expected: true,
		},
		{
			name:     "PDF signature",
			data:     []byte{0x25, 0x50, 0x44, 0x46},
			expected: true,
		},
		{
			name:     "no signature",
			data:     []byte("Hello"),
			expected: false,
		},
		{
			name:     "too short",
			data:     []byte{0xFF},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasBinarySignature(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultDiffConfig(t *testing.T) {
	config := DefaultDiffConfig()

	assert.Equal(t, 1024*1024, config.MaxDiffSize)
	assert.True(t, config.UseBufferPool)
	assert.True(t, config.EarlyExit)
	assert.Equal(t, 4096, config.ChunkSize)
}

func TestDiffOptimized(t *testing.T) {
	tests := []struct {
		name        string
		a           []byte
		b           []byte
		maxDiff     int
		expectDiff  bool
		expectEmpty bool
	}{
		{
			name:        "identical content",
			a:           []byte("Hello, World!"),
			b:           []byte("Hello, World!"),
			maxDiff:     100,
			expectDiff:  true,
			expectEmpty: true,
		},
		{
			name:       "small text difference",
			a:          []byte("Hello, World!"),
			b:          []byte("Hello, Universe!"),
			maxDiff:    100,
			expectDiff: true,
		},
		{
			name:       "exceed max diff size",
			a:          []byte("small"),
			b:          bytes.Repeat([]byte("large"), 1000),
			maxDiff:    100,
			expectDiff: false,
		},
		{
			name:        "empty files",
			a:           []byte{},
			b:           []byte{},
			maxDiff:     100,
			expectDiff:  true,
			expectEmpty: true,
		},
		{
			name:       "one empty file",
			a:          []byte("content"),
			b:          []byte{},
			maxDiff:    100,
			expectDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, ok := DiffOptimized(tt.a, tt.b, tt.maxDiff)
			assert.Equal(t, tt.expectDiff, ok)
			if tt.expectEmpty && ok {
				assert.Nil(t, diff)
			}
			if ok && !tt.expectEmpty {
				assert.NotNil(t, diff)
			}
		})
	}
}

func TestDiffOptimizedWithConfig(t *testing.T) {
	tests := []struct {
		name       string
		a          []byte
		b          []byte
		config     DiffConfig
		expectDiff bool
	}{
		{
			name: "early exit disabled",
			a:    []byte("small"),
			b:    bytes.Repeat([]byte("large"), 100),
			config: DiffConfig{
				MaxDiffSize:   10,
				UseBufferPool: false,
				EarlyExit:     false,
				ChunkSize:     4096,
			},
			expectDiff: false,
		},
		{
			name: "text diff",
			a:    []byte("line1\nline2\nline3"),
			b:    []byte("line1\nmodified\nline3"),
			config: DiffConfig{
				MaxDiffSize:   1024,
				UseBufferPool: true,
				EarlyExit:     true,
				ChunkSize:     4096,
			},
			expectDiff: true,
		},
		{
			name: "binary diff",
			a:    []byte{0xFF, 0xD8, 0xFF, 0xE0},
			b:    []byte{0xFF, 0xD8, 0xFF, 0xE1},
			config: DiffConfig{
				MaxDiffSize:   1024,
				UseBufferPool: true,
				EarlyExit:     true,
				ChunkSize:     4096,
			},
			expectDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, ok := DiffOptimizedWithConfig(tt.a, tt.b, tt.config)
			assert.Equal(t, tt.expectDiff, ok)
			if ok && !bytes.Equal(tt.a, tt.b) {
				assert.NotNil(t, diff)
			}
		})
	}
}

func TestEstimateDiffSize(t *testing.T) {
	tests := []struct {
		name    string
		a       []byte
		b       []byte
		wantMin int
		wantMax int
	}{
		{
			name:    "small files",
			a:       make([]byte, 1000),
			b:       make([]byte, 2000),
			wantMin: 3000,
			wantMax: 3000,
		},
		{
			name:    "large files",
			a:       make([]byte, 100*1024),
			b:       make([]byte, 100*1024),
			wantMin: 50 * 1024,
			wantMax: 50 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := estimateDiffSize(tt.a, tt.b)
			assert.GreaterOrEqual(t, result, tt.wantMin)
			assert.LessOrEqual(t, result, tt.wantMax)
		})
	}
}

func TestIsMostlyText(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: true,
		},
		{
			name:     "plain text",
			content:  []byte("This is plain text with\nnewlines and\ttabs."),
			expected: true,
		},
		{
			name:     "mostly text with few control chars",
			content:  append([]byte(strings.Repeat("text", 100)), 0x01, 0x02),
			expected: true,
		},
		{
			name:     "binary content",
			content:  bytes.Repeat([]byte{0xFF, 0x00, 0x01, 0x02}, 100),
			expected: false,
		},
		{
			name:     "mixed content over threshold",
			content:  append([]byte("ABC"), bytes.Repeat([]byte{0x00}, 100)...),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMostlyText(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultBatchProcessorConfig(t *testing.T) {
	config := DefaultBatchProcessorConfig()

	assert.Equal(t, 100, config.BatchSize)
	assert.True(t, config.AutoFlush)
	assert.Equal(t, time.Second*5, config.FlushInterval)
	assert.Equal(t, time.Second*30, config.MaxWaitTime)
}

func TestBatchProcessor(t *testing.T) {
	t.Run("basic add and flush", func(t *testing.T) {
		var processed []interface{}
		var mu sync.Mutex

		processor := func(items []interface{}) error {
			mu.Lock()
			processed = append(processed, items...)
			mu.Unlock()
			return nil
		}

		config := BatchProcessorConfig{
			BatchSize:     3,
			AutoFlush:     false,
			FlushInterval: time.Hour,
			MaxWaitTime:   time.Hour,
		}

		bp := NewBatchProcessor(processor, config)

		// Add items
		require.NoError(t, bp.Add("item1"))
		require.NoError(t, bp.Add("item2"))

		// Should not process yet
		mu.Lock()
		assert.Empty(t, processed)
		mu.Unlock()

		// Add third item should trigger flush
		require.NoError(t, bp.Add("item3"))

		// Wait a bit for processing
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		assert.Equal(t, []interface{}{"item1", "item2", "item3"}, processed)
		mu.Unlock()

		// Add more items and manually flush
		require.NoError(t, bp.Add("item4"))
		require.NoError(t, bp.Add("item5"))
		require.NoError(t, bp.Flush())

		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		assert.Contains(t, processed, "item4")
		assert.Contains(t, processed, "item5")
		mu.Unlock()

		require.NoError(t, bp.Stop())
	})

	t.Run("batch add", func(t *testing.T) {
		var processed []interface{}
		var mu sync.Mutex

		processor := func(items []interface{}) error {
			mu.Lock()
			processed = append(processed, items...)
			mu.Unlock()
			return nil
		}

		config := BatchProcessorConfig{
			BatchSize:     2,
			AutoFlush:     false,
			FlushInterval: time.Hour,
			MaxWaitTime:   time.Hour,
		}

		bp := NewBatchProcessor(processor, config)

		items := []interface{}{"a", "b", "c", "d", "e"}
		require.NoError(t, bp.AddBatch(items))

		// Wait for processing
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		// Should have processed in batches of 2
		assert.Len(t, processed, 4) // First 4 items
		mu.Unlock()

		// Flush remaining
		require.NoError(t, bp.Flush())
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		assert.Len(t, processed, 5)
		mu.Unlock()

		require.NoError(t, bp.Stop())
	})

	t.Run("auto flush", func(t *testing.T) {
		var processed []interface{}
		var mu sync.Mutex

		processor := func(items []interface{}) error {
			mu.Lock()
			processed = append(processed, items...)
			mu.Unlock()
			return nil
		}

		config := BatchProcessorConfig{
			BatchSize:     10,
			AutoFlush:     true,
			FlushInterval: 50 * time.Millisecond,
			MaxWaitTime:   100 * time.Millisecond,
		}

		bp := NewBatchProcessor(processor, config)

		// Add items that won't trigger batch size flush
		require.NoError(t, bp.Add("item1"))
		require.NoError(t, bp.Add("item2"))

		// Wait for auto flush
		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		assert.Len(t, processed, 2)
		mu.Unlock()

		require.NoError(t, bp.Stop())
	})

	t.Run("stats", func(t *testing.T) {
		processor := func(_ []interface{}) error {
			return nil
		}

		config := BatchProcessorConfig{
			BatchSize:     5,
			AutoFlush:     false,
			FlushInterval: time.Hour,
			MaxWaitTime:   time.Hour,
		}

		bp := NewBatchProcessor(processor, config)

		require.NoError(t, bp.Add("item1"))
		require.NoError(t, bp.Add("item2"))

		stats := bp.Stats()
		assert.Equal(t, 2, stats.PendingItems)
		assert.NotZero(t, stats.LastFlushTime)
		assert.Greater(t, stats.TimeSinceFlush, time.Duration(0))

		require.NoError(t, bp.Stop())
	})

	t.Run("processor error", func(t *testing.T) {
		expectedErr := assert.AnError
		processor := func(_ []interface{}) error {
			return expectedErr
		}

		config := BatchProcessorConfig{
			BatchSize:     2,
			AutoFlush:     false,
			FlushInterval: time.Hour,
			MaxWaitTime:   time.Hour,
		}

		bp := NewBatchProcessor(processor, config)

		require.NoError(t, bp.Add("item1"))
		err := bp.Add("item2") // Should trigger flush with error
		assert.Equal(t, expectedErr, err)

		require.NoError(t, bp.Stop())
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("abs", func(t *testing.T) {
		assert.Equal(t, 5, abs(5))
		assert.Equal(t, 5, abs(-5))
		assert.Equal(t, 0, abs(0))
	})

	t.Run("minInt", func(t *testing.T) {
		assert.Equal(t, 3, minInt(3, 5))
		assert.Equal(t, 3, minInt(5, 3))
		assert.Equal(t, -5, minInt(-5, 0))
	})

	t.Run("maxInt", func(t *testing.T) {
		assert.Equal(t, 5, maxInt(3, 5))
		assert.Equal(t, 5, maxInt(5, 3))
		assert.Equal(t, 0, maxInt(-5, 0))
	})
}

func TestComputeLCS(t *testing.T) {
	tests := []struct {
		name     string
		a        [][]byte
		b        [][]byte
		maxDiff  int
		expectOk bool
	}{
		{
			name: "identical lines",
			a: [][]byte{
				[]byte("line1"),
				[]byte("line2"),
				[]byte("line3"),
			},
			b: [][]byte{
				[]byte("line1"),
				[]byte("line2"),
				[]byte("line3"),
			},
			maxDiff:  1024,
			expectOk: true,
		},
		{
			name: "one line changed",
			a: [][]byte{
				[]byte("line1"),
				[]byte("line2"),
				[]byte("line3"),
			},
			b: [][]byte{
				[]byte("line1"),
				[]byte("modified"),
				[]byte("line3"),
			},
			maxDiff:  1024,
			expectOk: true,
		},
		{
			name: "lines added",
			a: [][]byte{
				[]byte("line1"),
				[]byte("line2"),
			},
			b: [][]byte{
				[]byte("line1"),
				[]byte("line2"),
				[]byte("line3"),
				[]byte("line4"),
			},
			maxDiff:  1024,
			expectOk: true,
		},
		{
			name: "exceed max diff",
			a: [][]byte{
				[]byte("line1"),
			},
			b: [][]byte{
				[]byte(strings.Repeat("x", 100)),
				[]byte(strings.Repeat("y", 100)),
			},
			maxDiff:  50,
			expectOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			config := DiffConfig{
				MaxDiffSize: tt.maxDiff,
			}

			ok := computeLCS(tt.a, tt.b, buf, config)
			assert.Equal(t, tt.expectOk, ok)

			if ok && !equalByteSlices(tt.a, tt.b) {
				assert.NotEmpty(t, buf.String())
			}
		})
	}
}

func TestComputeSimpleDiff(t *testing.T) {
	tests := []struct {
		name     string
		a        [][]byte
		b        [][]byte
		maxDiff  int
		expectOk bool
	}{
		{
			name: "simple change",
			a: [][]byte{
				[]byte("unchanged1"),
				[]byte("old"),
				[]byte("unchanged2"),
			},
			b: [][]byte{
				[]byte("unchanged1"),
				[]byte("new"),
				[]byte("unchanged2"),
			},
			maxDiff:  1024,
			expectOk: true,
		},
		{
			name: "multiple changes",
			a: [][]byte{
				[]byte("a1"),
				[]byte("a2"),
				[]byte("common"),
				[]byte("a3"),
			},
			b: [][]byte{
				[]byte("b1"),
				[]byte("b2"),
				[]byte("common"),
				[]byte("b3"),
			},
			maxDiff:  1024,
			expectOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			config := DiffConfig{
				MaxDiffSize: tt.maxDiff,
			}

			ok := computeSimpleDiff(tt.a, tt.b, buf, config)
			assert.Equal(t, tt.expectOk, ok)

			if ok {
				assert.NotEmpty(t, buf.String())
			}
		})
	}
}

// Helper function for testing
func equalByteSlices(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

// Benchmarks
func BenchmarkIsBinaryOptimized(b *testing.B) {
	testCases := []struct {
		name    string
		content []byte
	}{
		{"small_text", []byte("Hello, World!")},
		{"large_text", bytes.Repeat([]byte("Hello, World!\n"), 1000)},
		{"small_binary", []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}},
		{"large_binary", bytes.Repeat([]byte{0xFF, 0x00, 0x01, 0x02}, 1000)},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = IsBinaryOptimized(tc.content)
			}
		})
	}
}

func BenchmarkDiffOptimized(b *testing.B) {
	testCases := []struct {
		name string
		a    []byte
		b    []byte
	}{
		{
			"small_identical",
			[]byte("Hello, World!"),
			[]byte("Hello, World!"),
		},
		{
			"small_different",
			[]byte("Hello, World!"),
			[]byte("Hello, Universe!"),
		},
		{
			"large_similar",
			bytes.Repeat([]byte("Hello, World!\n"), 100),
			append(bytes.Repeat([]byte("Hello, World!\n"), 99), []byte("Hello, Universe!\n")...),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = DiffOptimized(tc.a, tc.b, 1024*1024)
			}
		})
	}
}

func BenchmarkBatchProcessor(b *testing.B) {
	processor := func(_ []interface{}) error {
		// Simulate some work
		time.Sleep(time.Microsecond)
		return nil
	}

	config := BatchProcessorConfig{
		BatchSize:     100,
		AutoFlush:     false,
		FlushInterval: time.Hour,
		MaxWaitTime:   time.Hour,
	}

	bp := NewBatchProcessor(processor, config)
	defer func() { _ = bp.Stop() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bp.Add(i)
	}
	_ = bp.Flush()
}
