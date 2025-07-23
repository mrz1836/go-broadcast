package transform

import (
	"path/filepath"
	"strings"
)

// binaryExtensions contains common binary file extensions
//
//nolint:gochecknoglobals // This is a read-only lookup table
var binaryExtensions = map[string]bool{
	// Images
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".ico":  true,
	".svg":  true,
	".webp": true,

	// Archives
	".zip": true,
	".tar": true,
	".gz":  true,
	".bz2": true,
	".xz":  true,
	".7z":  true,
	".rar": true,

	// Executables
	".exe":   true,
	".dll":   true,
	".so":    true,
	".dylib": true,
	".a":     true,
	".o":     true,

	// Media
	".mp3":  true,
	".mp4":  true,
	".avi":  true,
	".mov":  true,
	".wav":  true,
	".flac": true,
	".ogg":  true,

	// Documents
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".ppt":  true,
	".pptx": true,

	// Other
	".jar":    true,
	".war":    true,
	".ear":    true,
	".class":  true,
	".pyc":    true,
	".pyo":    true,
	".wasm":   true,
	".db":     true,
	".sqlite": true,
}

// IsBinary checks if a file is likely binary based on its extension and content
func IsBinary(filePath string, content []byte) bool {
	// Check extension first (fast)
	ext := strings.ToLower(filepath.Ext(filePath))
	if binaryExtensions[ext] {
		return true
	}

	// Check content (slower, but more accurate)
	return isBinaryContent(content)
}

// isBinaryContent checks if content appears to be binary
func isBinaryContent(content []byte) bool {
	// Empty files are not binary
	if len(content) == 0 {
		return false
	}

	// Check first 8KB for binary indicators
	checkLen := len(content)
	if checkLen > 8192 {
		checkLen = 8192
	}

	// Count non-text characters
	nonTextBytes := 0

	for i := 0; i < checkLen; i++ {
		b := content[i]

		// Null byte is a strong indicator of binary content
		if b == 0 {
			return true
		}

		// Check for non-text characters (excluding common whitespace)
		if (b < 32 && b != '\t' && b != '\n' && b != '\r') || b > 127 {
			nonTextBytes++
		}
	}

	// If more than 30% of checked bytes are non-text, consider it binary
	threshold := checkLen * 30 / 100
	return nonTextBytes > threshold
}

// binaryTransformer is a no-op transformer for binary files
type binaryTransformer struct{}

// NewBinaryTransformer creates a transformer that skips binary files
func NewBinaryTransformer() Transformer {
	return &binaryTransformer{}
}

// Name returns the name of this transformer
func (b *binaryTransformer) Name() string {
	return "binary-file-skipper"
}

// Transform returns content unchanged if it's binary
func (b *binaryTransformer) Transform(content []byte, ctx Context) ([]byte, error) {
	if IsBinary(ctx.FilePath, content) {
		// Return content unchanged
		return content, nil
	}

	// For non-binary files, this transformer does nothing
	// (other transformers in the chain will process the file)
	return content, nil
}
