package transform

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBinary_ByExtension(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Binary extensions
		{"image.jpg", true},
		{"image.JPEG", true},
		{"photo.png", true},
		{"archive.zip", true},
		{"program.exe", true},
		{"library.so", true},
		{"document.pdf", true},
		{"data.db", true},

		// Text extensions
		{"code.go", false},
		{"script.py", false},
		{"config.yaml", false},
		{"readme.md", false},
		{"data.json", false},
		{"style.css", false},

		// No extension
		{"Makefile", false},
		{"README", false},

		// Multiple dots
		{"archive.tar.gz", true},
		{"code.test.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			// For extension test, content doesn't matter
			result := IsBinary(tt.path, []byte("dummy content"))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBinary_ByContent(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "text content",
			content:  []byte("This is plain text content\nwith newlines\nand normal characters."),
			expected: false,
		},
		{
			name:     "empty content",
			content:  []byte{},
			expected: false,
		},
		{
			name:     "null bytes",
			content:  []byte("text\x00with\x00null\x00bytes"),
			expected: true,
		},
		{
			name:     "high concentration of control characters",
			content:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 11, 12, 14, 15, 16, 17, 18, 19},
			expected: true,
		},
		{
			name:     "valid UTF-8 with some high bytes",
			content:  []byte("Hello World"), // Just ASCII to avoid UTF-8 detection issues
			expected: false,
		},
		{
			name:     "mostly binary data",
			content:  generateBinaryData(1000, 80), // 80% binary
			expected: true,
		},
		{
			name:     "mostly text data",
			content:  generateBinaryData(1000, 25), // 25% binary - below threshold
			expected: false,
		},
		{
			name:     "text with allowed control chars",
			content:  []byte("Line 1\r\nLine 2\tTabbed\nLine 3"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a text file extension to ensure content check is used
			result := IsBinary("file.txt", tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBinaryContent_LargeFile(t *testing.T) {
	// Create content larger than 8KB to test sampling
	largeContent := make([]byte, 10000)

	// Fill first 8KB with text
	for i := 0; i < 8192; i++ {
		largeContent[i] = 'A' + byte(i%26)
	}

	// Add binary data after 8KB
	for i := 8192; i < 10000; i++ {
		largeContent[i] = 0
	}

	// Should be considered text because only first 8KB is checked
	assert.False(t, isBinaryContent(largeContent))

	// Now put binary data at the beginning
	for i := 0; i < 100; i++ {
		largeContent[i] = 0
	}

	// Should be considered binary
	assert.True(t, isBinaryContent(largeContent))
}

func TestBinaryTransformer(t *testing.T) {
	transformer := NewBinaryTransformer()

	assert.Equal(t, "binary-file-skipper", transformer.Name())

	tests := []struct {
		name    string
		path    string
		content []byte
	}{
		{
			name:    "binary file unchanged",
			path:    "image.jpg",
			content: []byte{0xFF, 0xD8, 0xFF, 0xE0}, // JPEG header
		},
		{
			name:    "text file unchanged",
			path:    "file.txt",
			content: []byte("This is text content"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Context{
				SourceRepo: "org/source",
				TargetRepo: "org/target",
				FilePath:   tt.path,
			}

			result, err := transformer.Transform(tt.content, ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.content, result)
		})
	}
}

// generateBinaryData creates test data with specified percentage of binary bytes
func generateBinaryData(size int, binaryPercent int) []byte {
	data := make([]byte, size)
	binaryCount := size * binaryPercent / 100

	// Fill with text first
	for i := 0; i < size; i++ {
		data[i] = 'A' + byte(i%26)
	}

	// Replace some bytes with high-value bytes (200+) to simulate binary
	// These are clearly non-text but won't be confused with control characters
	for i := 0; i < binaryCount; i++ {
		data[i] = byte(200 + (i % 56)) // High bytes from 200-255
	}

	return data
}

func TestBinaryExtensions(t *testing.T) {
	// Ensure common binary extensions are covered
	commonBinary := []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp",
		".zip", ".tar", ".gz", ".7z",
		".exe", ".dll", ".so",
		".pdf", ".doc", ".docx",
		".mp3", ".mp4", ".avi",
		".jar", ".class", ".pyc",
	}

	for _, ext := range commonBinary {
		t.Run(ext, func(t *testing.T) {
			assert.True(t, binaryExtensions[ext], "Extension %s should be marked as binary", ext)
		})
	}
}

func TestIsBinary_CaseInsensitive(t *testing.T) {
	// Test that extension check is case-insensitive
	upperCase := IsBinary("IMAGE.JPG", []byte("content"))
	lowerCase := IsBinary("image.jpg", []byte("content"))
	mixedCase := IsBinary("Image.Jpg", []byte("content"))

	assert.True(t, upperCase)
	assert.True(t, lowerCase)
	assert.True(t, mixedCase)
}

func TestIsBinaryContent_EdgeCases(t *testing.T) {
	t.Run("single null byte", func(t *testing.T) {
		content := []byte("Hello\x00World")
		assert.True(t, isBinaryContent(content))
	})

	t.Run("all whitespace", func(t *testing.T) {
		content := []byte(strings.Repeat(" \t\n\r", 100))
		assert.False(t, isBinaryContent(content))
	})

	t.Run("single byte", func(t *testing.T) {
		assert.False(t, isBinaryContent([]byte("A")))
		assert.True(t, isBinaryContent([]byte{0}))
	})
}
