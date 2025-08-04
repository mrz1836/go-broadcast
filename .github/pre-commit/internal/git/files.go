// Package git provides intelligent file filtering and classification for the pre-commit system
package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
)

// FileClassifier provides intelligent file classification and filtering
type FileClassifier struct {
	config *config.Config
}

// NewFileClassifier creates a new file classifier
func NewFileClassifier(cfg *config.Config) *FileClassifier {
	return &FileClassifier{config: cfg}
}

// FileInfo contains information about a file
type FileInfo struct {
	Path      string
	IsText    bool
	IsBinary  bool
	Language  string
	Size      int64
	IsGoFile  bool
	Generated bool
	Excluded  bool
}

// ClassifyFiles analyzes and classifies a list of files
func (fc *FileClassifier) ClassifyFiles(ctx context.Context, files []string) ([]FileInfo, error) {
	result := make([]FileInfo, 0, len(files))

	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			info, err := fc.classifyFile(file)
			if err != nil {
				// Log error but continue with other files
				continue
			}
			result = append(result, info)
		}
	}

	return result, nil
}

// FilterGoFiles returns only Go source files, excluding generated and test files if specified
func (fc *FileClassifier) FilterGoFiles(ctx context.Context, files []string, excludeTests bool) ([]string, error) {
	classified, err := fc.ClassifyFiles(ctx, files)
	if err != nil {
		return nil, err
	}

	var goFiles []string
	for _, info := range classified {
		if info.IsGoFile && !info.Generated && !info.Excluded {
			if excludeTests && strings.HasSuffix(info.Path, "_test.go") {
				continue
			}
			goFiles = append(goFiles, info.Path)
		}
	}

	return goFiles, nil
}

// FilterTextFiles returns only text files that are not excluded
func (fc *FileClassifier) FilterTextFiles(ctx context.Context, files []string) ([]string, error) {
	classified, err := fc.ClassifyFiles(ctx, files)
	if err != nil {
		return nil, err
	}

	var textFiles []string
	for _, info := range classified {
		if info.IsText && !info.Excluded && !info.Generated {
			textFiles = append(textFiles, info.Path)
		}
	}

	return textFiles, nil
}

// ExcludeByPatterns filters out files matching exclude patterns
func (fc *FileClassifier) ExcludeByPatterns(files, patterns []string) []string {
	if len(patterns) == 0 {
		return files
	}

	var filtered []string
	for _, file := range files {
		excluded := false
		for _, pattern := range patterns {
			if fc.matchesPattern(file, pattern) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, file)
		}
	}

	return filtered
}

// classifyFile analyzes a single file
func (fc *FileClassifier) classifyFile(filePath string) (FileInfo, error) {
	info := FileInfo{
		Path: filePath,
	}

	// Check if file exists and get size
	stat, err := os.Stat(filePath)
	if err != nil {
		return info, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	info.Size = stat.Size()

	// Check if file exceeds size limit
	if fc.config != nil && info.Size > fc.config.MaxFileSize {
		info.Excluded = true
		return info, nil
	}

	// Classify by extension and name
	info.IsGoFile = fc.isGoFile(filePath)
	info.Language = fc.detectLanguage(filePath)
	info.Generated = fc.isGeneratedFile(filePath)
	info.Excluded = fc.isExcludedPath(filePath)

	// Detect if file is text or binary
	if info.Size > 0 && info.Size < 1024*1024 { // Only check files up to 1MB
		content, err := fc.readFileHead(filePath, 512) // Read first 512 bytes
		if err == nil {
			info.IsText = fc.isTextContent(content)
			info.IsBinary = !info.IsText
		}
	}

	return info, nil
}

// isGoFile checks if the file is a Go source file
func (fc *FileClassifier) isGoFile(filePath string) bool {
	if !strings.HasSuffix(filePath, ".go") {
		return false
	}

	// Exclude vendor directories
	if strings.Contains(filePath, "/vendor/") || strings.HasPrefix(filePath, "vendor/") {
		return false
	}

	return true
}

// detectLanguage determines the programming language based on file extension
func (fc *FileClassifier) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":    "go",
		".mod":   "go-mod",
		".sum":   "go-sum",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "javascript",
		".tsx":   "typescript",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".cc":    "cpp",
		".cxx":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".rs":    "rust",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".cs":    "csharp",
		".sh":    "shell",
		".bash":  "shell",
		".zsh":   "shell",
		".fish":  "shell",
		".ps1":   "powershell",
		".sql":   "sql",
		".md":    "markdown",
		".txt":   "text",
		".yml":   "yaml",
		".yaml":  "yaml",
		".json":  "json",
		".xml":   "xml",
		".html":  "html",
		".htm":   "html",
		".css":   "css",
		".scss":  "scss",
		".sass":  "sass",
		".less":  "less",
		".proto": "protobuf",
		".toml":  "toml",
		".ini":   "ini",
		".cfg":   "config",
		".conf":  "config",
		".env":   "env",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	// Check for files without extensions
	base := strings.ToLower(filepath.Base(filePath))
	specialFiles := map[string]string{
		"makefile":      "make",
		"dockerfile":    "docker",
		"jenkinsfile":   "groovy",
		"vagrantfile":   "ruby",
		".gitignore":    "gitignore",
		".dockerignore": "dockerignore",
		".editorconfig": "editorconfig",
	}

	if lang, exists := specialFiles[base]; exists {
		return lang
	}

	return "unknown"
}

// isGeneratedFile checks if a file is generated code
func (fc *FileClassifier) isGeneratedFile(filePath string) bool {
	// Common generated file patterns
	generatedPatterns := []string{
		"*.pb.go",         // Protocol buffer generated files
		"*.pb.gw.go",      // gRPC gateway generated files
		"*_string.go",     // stringer generated files
		"*_easyjson.go",   // easyjson generated files
		"*_ffjson.go",     // ffjson generated files
		"*_mock.go",       // gomock generated files
		"mock_*.go",       // gomock generated files
		"*_gen.go",        // Generic generated files
		"generated.go",    // Generic generated files
		"bindata.go",      // go-bindata generated files
		"*_bindata.go",    // go-bindata generated files
		"statik.go",       // statik generated files
		"wire_gen.go",     // wire generated files
		"*_wire.go",       // wire generated files
		"*_vtproto.pb.go", // vtprotobuf generated files
		"*.swagger.json",  // Swagger generated files
		"*_swagger.go",    // Swagger generated files
	}

	fileName := filepath.Base(filePath)
	for _, pattern := range generatedPatterns {
		if fc.matchesPattern(fileName, pattern) {
			return true
		}
	}

	// Check for generated file markers in the first few lines
	if strings.HasSuffix(filePath, ".go") {
		if fc.hasGeneratedMarker(filePath) {
			return true
		}
	}

	return false
}

// hasGeneratedMarker checks if a Go file has standard generated file markers
func (fc *FileClassifier) hasGeneratedMarker(filePath string) bool {
	content, err := fc.readFileHead(filePath, 1024) // Read first 1KB
	if err != nil {
		return false
	}

	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Check first 10 lines for generated markers
	for i, line := range lines {
		if i >= 10 {
			break
		}

		line = strings.TrimSpace(strings.ToLower(line))

		// Standard generated file markers
		if strings.Contains(line, "code generated") ||
			strings.Contains(line, "do not edit") ||
			strings.Contains(line, "auto-generated") ||
			strings.Contains(line, "autogenerated") ||
			strings.Contains(line, "automatically generated") ||
			strings.Contains(line, "this file was automatically generated") ||
			strings.Contains(line, "generated by") {
			return true
		}
	}

	return false
}

// isExcludedPath checks if a file path should be excluded
func (fc *FileClassifier) isExcludedPath(filePath string) bool {
	// Default exclude patterns
	defaultExcludes := []string{
		"vendor/",
		".git/",
		"node_modules/",
		".vscode/",
		".idea/",
		"*.tmp",
		"*.temp",
		"*.log",
		"*.cache",
		".DS_Store",
		"Thumbs.db",
		"*.bak",
		"*.orig",
		"*.rej",
		"*~",
		"#*#",
		".#*",
		"*.swp",
		"*.swo",
		"coverage.out",
		"*.test",
		"*.prof",
		"*.pprof",
	}

	// Check default excludes
	for _, pattern := range defaultExcludes {
		if fc.matchesPattern(filePath, pattern) {
			return true
		}
	}

	// Check configured exclude patterns
	if fc.config != nil {
		for _, pattern := range fc.config.Git.ExcludePatterns {
			if fc.matchesPattern(filePath, pattern) {
				return true
			}
		}
	}

	return false
}

// isTextContent determines if content is text or binary
func (fc *FileClassifier) isTextContent(content []byte) bool {
	// Empty files are considered text
	if len(content) == 0 {
		return true
	}

	// Check for UTF-8 validity
	if !utf8.Valid(content) {
		return false
	}

	// Check for null bytes (common in binary files)
	if bytes.Contains(content, []byte{0}) {
		return false
	}

	// Count control characters
	controlChars := 0
	printableChars := 0

	for _, b := range content {
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			controlChars++
		} else if b >= 32 && b <= 126 {
			printableChars++
		}
	}

	// If more than 30% are control characters, likely binary
	if len(content) > 0 && float64(controlChars)/float64(len(content)) > 0.3 {
		return false
	}

	return true
}

// readFileHead reads the first n bytes of a file
func (fc *FileClassifier) readFileHead(filePath string, n int) ([]byte, error) {
	file, err := os.Open(filePath) //nolint:gosec // File path from git
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close() // Best effort close
	}()

	buffer := make([]byte, n)
	bytesRead, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	return buffer[:bytesRead], nil
}

// matchesPattern checks if a string matches a glob pattern
func (fc *FileClassifier) matchesPattern(str, pattern string) bool {
	// Handle directory patterns
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(str, pattern) || strings.Contains(str, "/"+pattern)
	}

	// Simple glob matching
	if strings.Contains(pattern, "*") {
		return fc.globMatch(str, pattern)
	}

	// Exact match or contains
	return str == pattern || strings.Contains(str, pattern)
}

// globMatch performs simple glob matching
func (fc *FileClassifier) globMatch(str, pattern string) bool {
	// Convert glob pattern to regex-like matching
	parts := strings.Split(pattern, "*")

	if len(parts) == 1 {
		return str == pattern
	}

	// Check if string starts with first part
	if len(parts[0]) > 0 && !strings.HasPrefix(str, parts[0]) {
		return false
	}

	// Check if string ends with last part
	if len(parts[len(parts)-1]) > 0 && !strings.HasSuffix(str, parts[len(parts)-1]) {
		return false
	}

	// For more complex patterns, use filepath.Match
	matched, err := filepath.Match(pattern, filepath.Base(str))
	if err != nil {
		return false
	}

	return matched
}

// GetFileStats returns statistics about the classified files
func (fc *FileClassifier) GetFileStats(ctx context.Context, files []string) (map[string]int, error) {
	classified, err := fc.ClassifyFiles(ctx, files)
	if err != nil {
		return nil, err
	}

	stats := map[string]int{
		"total":     len(classified),
		"text":      0,
		"binary":    0,
		"go":        0,
		"generated": 0,
		"excluded":  0,
	}

	languageStats := make(map[string]int)

	for _, info := range classified {
		if info.IsText {
			stats["text"]++
		}
		if info.IsBinary {
			stats["binary"]++
		}
		if info.IsGoFile {
			stats["go"]++
		}
		if info.Generated {
			stats["generated"]++
		}
		if info.Excluded {
			stats["excluded"]++
		}

		if info.Language != "unknown" {
			languageStats[info.Language]++
		}
	}

	// Merge language stats
	for lang, count := range languageStats {
		stats["lang_"+lang] = count
	}

	return stats, nil
}
