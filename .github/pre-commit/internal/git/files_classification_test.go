package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
)

// TestNewFileClassifier tests the file classifier creation
func TestNewFileClassifier(t *testing.T) {
	cfg := &config.Config{
		MaxFileSize: 1024 * 1024,
	}

	fc := NewFileClassifier(cfg)
	assert.NotNil(t, fc)
	assert.Equal(t, cfg, fc.config)

	// Test with nil config
	fc2 := NewFileClassifier(nil)
	assert.NotNil(t, fc2)
	assert.Nil(t, fc2.config)
}

// TestIsGoFile tests Go file detection
func TestIsGoFile(t *testing.T) {
	fc := NewFileClassifier(nil)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Go source file", "main.go", true},
		{"Go test file", "main_test.go", true},
		{"Go file in subdir", "cmd/app/main.go", true},
		{"Go file with path", "internal/pkg/file.go", true},
		{"Not Go file", "main.py", false},
		{"No extension", "main", false},
		{"Vendor file", "vendor/github.com/pkg/file.go", false},
		{"File in vendor", "internal/vendor/file.go", false},
		{"Vendor at start", "vendor/file.go", false},
		{"Go mod file", "go.mod", false},
		{"Hidden Go file", ".hidden.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fc.isGoFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDetectLanguage tests language detection
func TestDetectLanguage(t *testing.T) {
	fc := NewFileClassifier(nil)

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		// Go files
		{"Go source", "main.go", "go"},
		{"Go mod", "go.mod", "go-mod"},
		{"Go sum", "go.sum", "go-sum"},

		// Common languages
		{"Python", "script.py", "python"},
		{"JavaScript", "app.js", "javascript"},
		{"TypeScript", "app.ts", "typescript"},
		{"Java", "Main.java", "java"},
		{"C", "main.c", "c"},
		{"C++", "main.cpp", "cpp"},
		{"C++ alt", "main.cc", "cpp"},
		{"C++ alt2", "main.cxx", "cpp"},
		{"C header", "header.h", "c"},
		{"C++ header", "header.hpp", "cpp"},
		{"Rust", "main.rs", "rust"},
		{"Ruby", "script.rb", "ruby"},
		{"PHP", "index.php", "php"},
		{"Swift", "app.swift", "swift"},
		{"Kotlin", "app.kt", "kotlin"},
		{"Scala", "app.scala", "scala"},
		{"C#", "Program.cs", "csharp"},

		// Shell scripts
		{"Shell", "script.sh", "shell"},
		{"Bash", "script.bash", "shell"},
		{"Zsh", "script.zsh", "shell"},
		{"Fish", "script.fish", "shell"},
		{"PowerShell", "script.ps1", "powershell"},

		// Data formats
		{"SQL", "query.sql", "sql"},
		{"Markdown", "README.md", "markdown"},
		{"Text", "notes.txt", "text"},
		{"YAML", "config.yml", "yaml"},
		{"YAML alt", "config.yaml", "yaml"},
		{"JSON", "data.json", "json"},
		{"XML", "data.xml", "xml"},
		{"HTML", "index.html", "html"},
		{"HTML alt", "index.htm", "html"},
		{"CSS", "style.css", "css"},
		{"SCSS", "style.scss", "scss"},
		{"Sass", "style.sass", "sass"},
		{"Less", "style.less", "less"},
		{"Protobuf", "api.proto", "protobuf"},
		{"TOML", "config.toml", "toml"},
		{"INI", "config.ini", "ini"},
		{"Config", "app.cfg", "config"},
		{"Config alt", "app.conf", "config"},
		{"Env", ".env", "env"},

		// Special files
		{"Makefile", "Makefile", "make"},
		{"Makefile lower", "makefile", "make"},
		{"Dockerfile", "Dockerfile", "docker"},
		{"Dockerfile lower", "dockerfile", "docker"},
		{"Jenkinsfile", "Jenkinsfile", "groovy"},
		{"Jenkinsfile lower", "jenkinsfile", "groovy"},
		{"Vagrantfile", "Vagrantfile", "ruby"},
		{"Vagrantfile lower", "vagrantfile", "ruby"},
		{"Gitignore", ".gitignore", "gitignore"},
		{"Dockerignore", ".dockerignore", "dockerignore"},
		{"Editorconfig", ".editorconfig", "editorconfig"},

		// Unknown
		{"Unknown extension", "file.xyz", "unknown"},
		{"No extension", "README", "unknown"},
		{"Binary", "app.exe", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fc.detectLanguage(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsGeneratedFile tests generated file detection
func TestIsGeneratedFile(t *testing.T) {
	fc := NewFileClassifier(nil)

	// Create temp directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		fileName  string
		content   string
		generated bool
	}{
		// Generated patterns
		{"Protobuf", "api.pb.go", "", true},
		{"gRPC gateway", "api.pb.gw.go", "", true},
		{"Stringer", "types_string.go", "", true},
		{"EasyJSON", "model_easyjson.go", "", true},
		{"FFJson", "model_ffjson.go", "", true},
		{"GoMock", "mock_service.go", "", true},
		{"GoMock alt", "service_mock.go", "", true},
		{"Generic gen", "types_gen.go", "", true},
		{"Generic generated", "generated.go", "", true},
		{"Bindata", "bindata.go", "", true},
		{"Bindata alt", "assets_bindata.go", "", true},
		{"Statik", "statik.go", "", true},
		{"Wire", "wire_gen.go", "", true},
		{"Wire alt", "providers_wire.go", "", true},
		{"VTProtobuf", "message_vtproto.pb.go", "", true},
		{"Swagger JSON", "api.swagger.json", "", true},
		{"Swagger Go", "api_swagger.go", "", true},

		// Files with generated markers
		{"With marker 1", "custom.go", "// Code generated by tool. DO NOT EDIT.\npackage main", true},
		{"With marker 2", "custom.go", "// This file was automatically generated\npackage main", true},
		{"With marker 3", "custom.go", "// Auto-generated file\npackage main", true},
		{"With marker 4", "custom.go", "// Autogenerated by tool\npackage main", true},
		{"With marker 5", "custom.go", "// Generated by protoc-gen-go\npackage main", true},
		{"Marker lowercase", "custom.go", "// code generated - do not edit\npackage main", true},
		{"Marker in comment", "custom.go", "/*\nThis file is automatically generated\n*/\npackage main", true},

		// Non-generated files
		{"Regular Go file", "main.go", "package main\n\nfunc main() {}", false},
		{"Regular test", "main_test.go", "package main\n\nimport \"testing\"", false},
		{"Similar name", "stringify.go", "package main", false},
		{"Mock in name", "mockery.go", "package main", false},
		{"Gen in path", "generator/main.go", "package main", false},

		// Edge cases
		{"Empty file", "empty.go", "", false},
		{"Marker after line 10", "late.go", strings.Repeat("package main\n", 11) + "// Code generated", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file if content is provided
			filePath := tt.fileName
			if tt.content != "" {
				filePath = filepath.Join(tempDir, tt.fileName)
				// Create directory if needed
				dir := filepath.Dir(filePath)
				err := os.MkdirAll(dir, 0o750)
				require.NoError(t, err)
				err = os.WriteFile(filePath, []byte(tt.content), 0o600)
				require.NoError(t, err)
			}

			result := fc.isGeneratedFile(filePath)
			assert.Equal(t, tt.generated, result, "File: %s", tt.fileName)
		})
	}
}

// TestIsExcludedPath tests path exclusion logic
func TestIsExcludedPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		config   *config.Config
		excluded bool
	}{
		// Default excludes
		{"Vendor dir", "vendor/github.com/pkg/file.go", nil, true},
		{"Git dir", ".git/config", nil, true},
		{"Node modules", "node_modules/package/index.js", nil, true},
		{"VSCode", ".vscode/settings.json", nil, true},
		{"IDEA", ".idea/workspace.xml", nil, true},
		{"Temp file", "file.tmp", nil, true},
		{"Temp file 2", "file.temp", nil, true},
		{"Log file", "app.log", nil, true},
		{"Cache file", "cache.cache", nil, true},
		{"DS_Store", ".DS_Store", nil, true},
		{"Thumbs.db", "Thumbs.db", nil, true},
		{"Backup", "file.bak", nil, true},
		{"Orig", "file.orig", nil, true},
		{"Reject", "file.rej", nil, true},
		{"Tilde", "file~", nil, true},
		{"Emacs temp", "#file#", nil, true},
		{"Emacs lock", ".#file", nil, true},
		{"Vim swap", "file.swp", nil, true},
		{"Vim swap 2", "file.swo", nil, true},
		{"Coverage", "coverage.out", nil, true},
		{"Test binary", "main.test", nil, true},
		{"Profile", "cpu.prof", nil, true},
		{"PProf", "mem.pprof", nil, true},

		// Not excluded by default
		{"Regular file", "main.go", nil, false},
		{"Similar name", "vendor.go", nil, false},
		{"Git in name", "gitutil.go", nil, false},

		// Custom excludes
		{
			"Custom pattern",
			"build/output.txt",
			&config.Config{
				Git: struct {
					HooksPath       string
					ExcludePatterns []string
				}{
					HooksPath:       ".git/hooks",
					ExcludePatterns: []string{"build/"},
				},
			},
			true,
		},
		{
			"Multiple patterns",
			"test.custom",
			&config.Config{
				Git: struct {
					HooksPath       string
					ExcludePatterns []string
				}{
					HooksPath:       ".git/hooks",
					ExcludePatterns: []string{"*.custom", "dist/"},
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := NewFileClassifier(tt.config)
			result := fc.isExcludedPath(tt.path)
			assert.Equal(t, tt.excluded, result)
		})
	}
}

// TestIsTextContent tests text vs binary detection
func TestIsTextContent(t *testing.T) {
	fc := NewFileClassifier(nil)

	tests := []struct {
		name    string
		content []byte
		isText  bool
	}{
		// Text content
		{"Empty", []byte{}, true},
		{"ASCII text", []byte("Hello, World!"), true},
		{"With newlines", []byte("Line 1\nLine 2\n"), true},
		{"With tabs", []byte("Name\tValue\n"), true},
		{"With CR LF", []byte("Windows\r\nLine"), true},
		{"UTF-8", []byte("Hello, 世界!"), true}, //nolint:gosmopolitan // Intentional non-Latin test case
		{"Code", []byte("func main() {\n\tfmt.Println(\"test\")\n}"), true},

		// Binary content
		{"Null bytes", []byte{0x00, 0x01, 0x02}, false},
		{"Invalid UTF-8", []byte{0xFF, 0xFE, 0xFD}, false},
		{"High control chars", []byte{0x01, 0x02, 0x03, 0x04, 0x05}, false},
		{"Mixed with null", []byte("text\x00binary"), false},

		// Edge cases
		{"Just printable", []byte(strings.Repeat("a", 100)), true},
		{"Mostly control", append(bytes.Repeat([]byte{0x01}, 70), bytes.Repeat([]byte("a"), 30)...), false},
		{"30% control boundary", append(bytes.Repeat([]byte{0x01}, 30), bytes.Repeat([]byte("a"), 70)...), false},
		{"29% control", append(bytes.Repeat([]byte{0x01}, 29), bytes.Repeat([]byte("a"), 71)...), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fc.isTextContent(tt.content)
			assert.Equal(t, tt.isText, result)
		})
	}
}

// TestMatchesPattern tests pattern matching
func TestMatchesPattern(t *testing.T) {
	fc := NewFileClassifier(nil)

	tests := []struct {
		name    string
		str     string
		pattern string
		matches bool
	}{
		// Exact matches
		{"Exact match", "file.go", "file.go", true},
		{"No match", "file.go", "other.go", false},

		// Directory patterns
		{"Dir pattern start", "vendor/file.go", "vendor/", true},
		{"Dir pattern middle", "src/vendor/file.go", "vendor/", true},
		{"Dir pattern no match", "file.go", "vendor/", false},

		// Simple globs
		{"Star at end", "file.go", "*.go", true},
		{"Star at end no match", "file.py", "*.go", false},
		{"Star at start", "main.go", "*main.go", true},
		{"Star at start no match", "main.go", "*test.go", false},
		{"Star in middle", "file_test.go", "file_*.go", true},
		{"Star in middle no match", "main.go", "file_*.go", false},

		// Contains matches
		{"Contains", "path/to/file.go", "to/file", true},
		{"Not contains", "path/to/file.go", "from/file", false},

		// Complex patterns
		{"Multiple stars", "path/to/file_test.go", "*/*_test.go", true},
		{"Extension only", "any.go", "*.go", true},
		{"Prefix and suffix", "test_file.go", "test_*.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fc.matchesPattern(tt.str, tt.pattern)
			assert.Equal(t, tt.matches, result)
		})
	}
}

// TestGlobMatch tests the glob matching function
func TestGlobMatch(t *testing.T) {
	fc := NewFileClassifier(nil)

	tests := []struct {
		name    string
		str     string
		pattern string
		matches bool
	}{
		// No wildcards
		{"Exact match", "file.go", "file.go", true},
		{"No match", "file.go", "other.go", false},

		// Single star patterns
		{"Star at end", "test.go", "*.go", true},
		{"Star at end no match", "test.py", "*.go", false},
		{"Star at start", "file.go", "*go", true},
		{"Star at start no match", "file.py", "*go", false},
		{"Star middle", "file_test.go", "file*.go", true},

		// Multiple stars
		{"Two stars", "path/file.go", "*/*.go", true},
		{"Two stars no match", "file.go", "*/*.go", false},

		// Edge cases
		{"Empty pattern", "file", "", false},
		{"Empty string", "", "*.go", false},
		{"Star only", "anything", "*", true},
		{"Complex pattern", "test_file_gen.go", "test_*_gen.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fc.globMatch(tt.str, tt.pattern)
			assert.Equal(t, tt.matches, result)
		})
	}
}

// TestReadFileHead tests reading file headers
func TestReadFileHead(t *testing.T) {
	fc := NewFileClassifier(nil)
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		content     string
		readBytes   int
		expected    string
		expectError bool
	}{
		{
			"Read full small file",
			"Hello",
			10,
			"Hello",
			false,
		},
		{
			"Read partial",
			"Hello, World!",
			5,
			"Hello",
			false,
		},
		{
			"Read exact",
			"12345",
			5,
			"12345",
			false,
		},
		{
			"Empty file",
			"",
			10,
			"",
			false,
		},
		{
			"Large content",
			strings.Repeat("a", 1000),
			100,
			strings.Repeat("a", 100),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tempDir, "test.txt")
			err := os.WriteFile(filePath, []byte(tt.content), 0o600)
			require.NoError(t, err)

			// Read file head
			result, err := fc.readFileHead(filePath, tt.readBytes)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, string(result))
			}
		})
	}

	// Test non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		_, err := fc.readFileHead("/non/existent/file", 10)
		assert.Error(t, err)
	})
}

// TestClassifyFile tests single file classification
func TestClassifyFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name       string
		fileName   string
		content    string
		size       int64
		config     *config.Config
		expected   FileInfo
		createFile bool
	}{
		{
			name:     "Go source file",
			fileName: "main.go",
			content:  "package main\n\nfunc main() {}\n",
			expected: FileInfo{
				Path:      filepath.Join(tempDir, "main.go"),
				IsText:    true,
				IsBinary:  false,
				Language:  "go",
				IsGoFile:  true,
				Generated: false,
				Excluded:  false,
			},
			createFile: true,
		},
		{
			name:     "Generated Go file",
			fileName: "types_string.go",
			content:  "// Code generated by stringer. DO NOT EDIT.\npackage main",
			expected: FileInfo{
				Path:      filepath.Join(tempDir, "types_string.go"),
				IsText:    true,
				IsBinary:  false,
				Language:  "go",
				IsGoFile:  true,
				Generated: true,
				Excluded:  false,
			},
			createFile: true,
		},
		{
			name:     "Binary file",
			fileName: "binary.dat",
			content:  string([]byte{0x00, 0x01, 0x02, 0x03, 0xFF}),
			expected: FileInfo{
				Path:      filepath.Join(tempDir, "binary.dat"),
				IsText:    false,
				IsBinary:  true,
				Language:  "unknown",
				IsGoFile:  false,
				Generated: false,
				Excluded:  false,
			},
			createFile: true,
		},
		{
			name:     "Excluded file",
			fileName: "file.tmp",
			content:  "temporary",
			expected: FileInfo{
				Path:      filepath.Join(tempDir, "file.tmp"),
				IsText:    true,
				IsBinary:  false,
				Language:  "unknown",
				IsGoFile:  false,
				Generated: false,
				Excluded:  true,
			},
			createFile: true,
		},
		{
			name:     "Large file excluded",
			fileName: "large.txt",
			content:  "small content", // Size will be overridden by config
			config: &config.Config{
				MaxFileSize: 10,
			},
			expected: FileInfo{
				Path:     filepath.Join(tempDir, "large.txt"),
				Excluded: true,
				Language: "text",
			},
			createFile: true,
		},
		{
			name:       "Non-existent file",
			fileName:   "nonexistent.go",
			createFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := NewFileClassifier(tt.config)
			filePath := filepath.Join(tempDir, tt.fileName)

			if tt.createFile {
				err := os.WriteFile(filePath, []byte(tt.content), 0o600)
				require.NoError(t, err)
			}

			result, err := fc.classifyFile(filePath)

			if !tt.createFile {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Update expected size
			if tt.expected.Size == 0 {
				tt.expected.Size = int64(len(tt.content))
			}

			assert.Equal(t, tt.expected.Path, result.Path)
			assert.Equal(t, tt.expected.IsText, result.IsText)
			assert.Equal(t, tt.expected.IsBinary, result.IsBinary)
			assert.Equal(t, tt.expected.Language, result.Language)
			assert.Equal(t, tt.expected.IsGoFile, result.IsGoFile)
			assert.Equal(t, tt.expected.Generated, result.Generated)
			assert.Equal(t, tt.expected.Excluded, result.Excluded)
			assert.Equal(t, tt.expected.Size, result.Size)
		})
	}
}

// TestClassifyFiles tests batch file classification
func TestClassifyFiles(t *testing.T) {
	tempDir := t.TempDir()
	fc := NewFileClassifier(nil)
	ctx := context.Background()

	// Create test files
	files := []struct {
		name    string
		content string
	}{
		{"main.go", "package main"},
		{"test.py", "print('hello')"},
		{"data.json", `{"key": "value"}`},
		{"binary.dat", string([]byte{0x00, 0xFF})},
	}

	filePaths := make([]string, 0, len(files))
	for _, f := range files {
		path := filepath.Join(tempDir, f.name)
		err := os.WriteFile(path, []byte(f.content), 0o600)
		require.NoError(t, err)
		filePaths = append(filePaths, path)
	}

	// Test successful classification
	t.Run("Successful classification", func(t *testing.T) {
		results, err := fc.ClassifyFiles(ctx, filePaths)
		require.NoError(t, err)
		assert.Len(t, results, len(files))

		// Verify some classifications
		for i, result := range results {
			assert.Equal(t, filePaths[i], result.Path)
			assert.Positive(t, result.Size)
		}
	})

	// Test with context cancellation
	t.Run("Context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		results, err := fc.ClassifyFiles(cancelCtx, filePaths)
		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
		assert.Nil(t, results)
	})

	// Test with non-existent files
	t.Run("With non-existent files", func(t *testing.T) {
		mixedPaths := append(filePaths, "/non/existent/file.go")
		results, err := fc.ClassifyFiles(ctx, mixedPaths)

		// Should not error, just skip bad files
		require.NoError(t, err)
		assert.Len(t, results, len(files)) // Only valid files
	})
}

// TestFilterGoFiles tests Go file filtering
func TestFilterGoFiles(t *testing.T) {
	tempDir := t.TempDir()
	fc := NewFileClassifier(nil)
	ctx := context.Background()

	// Create test files
	testFiles := []struct {
		name        string
		content     string
		isGo        bool
		isTest      bool
		isGenerated bool
	}{
		{"main.go", "package main", true, false, false},
		{"main_test.go", "package main", true, true, false},
		{"util.go", "package util", true, false, false},
		{"util_test.go", "package util", true, true, false},
		{"generated.pb.go", "// Code generated\npackage pb", true, false, true},
		{"doc.md", "# Documentation", false, false, false},
		{"script.py", "print('hello')", false, false, false},
	}

	allPaths := make([]string, 0, len(testFiles))
	expectedGo := make([]string, 0, len(testFiles))
	expectedGoNoTests := make([]string, 0, len(testFiles))

	for _, tf := range testFiles {
		path := filepath.Join(tempDir, tf.name)
		err := os.WriteFile(path, []byte(tf.content), 0o600)
		require.NoError(t, err)
		allPaths = append(allPaths, path)

		if tf.isGo && !tf.isGenerated {
			expectedGo = append(expectedGo, path)
			if !tf.isTest {
				expectedGoNoTests = append(expectedGoNoTests, path)
			}
		}
	}

	// Test without excluding tests
	t.Run("Include tests", func(t *testing.T) {
		results, err := fc.FilterGoFiles(ctx, allPaths, false)
		require.NoError(t, err)
		assert.ElementsMatch(t, expectedGo, results)
	})

	// Test excluding tests
	t.Run("Exclude tests", func(t *testing.T) {
		results, err := fc.FilterGoFiles(ctx, allPaths, true)
		require.NoError(t, err)
		assert.ElementsMatch(t, expectedGoNoTests, results)
	})

	// Test with context cancellation
	t.Run("Context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		results, err := fc.FilterGoFiles(cancelCtx, allPaths, false)
		require.Error(t, err)
		assert.Nil(t, results)
	})
}

// TestFilterTextFiles tests text file filtering
func TestFilterTextFiles(t *testing.T) {
	tempDir := t.TempDir()
	fc := NewFileClassifier(nil)
	ctx := context.Background()

	// Create test files
	testFiles := []struct {
		name       string
		content    []byte
		isText     bool
		isExcluded bool
	}{
		{"readme.md", []byte("# README"), true, false},
		{"main.go", []byte("package main"), true, false},
		{"config.json", []byte(`{"key": "value"}`), true, false},
		{"binary.exe", []byte{0x00, 0xFF, 0xFE}, false, false},
		{"image.png", []byte{0x89, 0x50, 0x4E, 0x47}, false, false},
		{".DS_Store", []byte("metadata"), true, true},
		{"cache.log", []byte("log data"), true, true},
		{"generated.pb.go", []byte("// Code generated\npackage pb"), true, false}, // Generated but still text
	}

	allPaths := make([]string, 0, len(testFiles))
	expectedText := make([]string, 0, len(testFiles))

	for _, tf := range testFiles {
		path := filepath.Join(tempDir, tf.name)
		err := os.WriteFile(path, tf.content, 0o600)
		require.NoError(t, err)
		allPaths = append(allPaths, path)

		// For this test, generated files are excluded in FilterTextFiles
		isGenerated := strings.Contains(tf.name, "generated")
		if tf.isText && !tf.isExcluded && !isGenerated {
			expectedText = append(expectedText, path)
		}
	}

	// Test text file filtering
	t.Run("Filter text files", func(t *testing.T) {
		results, err := fc.FilterTextFiles(ctx, allPaths)
		require.NoError(t, err)
		assert.ElementsMatch(t, expectedText, results)
	})
}

// TestExcludeByPatterns tests pattern-based exclusion
func TestExcludeByPatterns(t *testing.T) {
	fc := NewFileClassifier(nil)

	files := []string{
		"main.go",
		"main_test.go",
		"vendor/package/file.go",
		"internal/util.go",
		"build/output/app",
		"README.md",
		"docs/api.md",
		".git/config",
		"node_modules/pkg/index.js",
	}

	tests := []struct {
		name     string
		patterns []string
		expected []string
	}{
		{
			name:     "No patterns",
			patterns: []string{},
			expected: files,
		},
		{
			name:     "Single pattern",
			patterns: []string{"vendor/"},
			expected: []string{
				"main.go",
				"main_test.go",
				"internal/util.go",
				"build/output/app",
				"README.md",
				"docs/api.md",
				".git/config",
				"node_modules/pkg/index.js",
			},
		},
		{
			name:     "Multiple patterns",
			patterns: []string{"vendor/", "*.md", "build/"},
			expected: []string{
				"main.go",
				"main_test.go",
				"internal/util.go",
				".git/config",
				"node_modules/pkg/index.js",
			},
		},
		{
			name:     "Test files",
			patterns: []string{"*_test.go"},
			expected: []string{
				"main.go",
				"vendor/package/file.go",
				"internal/util.go",
				"build/output/app",
				"README.md",
				"docs/api.md",
				".git/config",
				"node_modules/pkg/index.js",
			},
		},
		{
			name:     "Hidden and system dirs",
			patterns: []string{".git/", "node_modules/"},
			expected: []string{
				"main.go",
				"main_test.go",
				"vendor/package/file.go",
				"internal/util.go",
				"build/output/app",
				"README.md",
				"docs/api.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fc.ExcludeByPatterns(files, tt.patterns)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

// TestGetFileStats tests file statistics collection
func TestGetFileStats(t *testing.T) {
	tempDir := t.TempDir()
	fc := NewFileClassifier(nil)
	ctx := context.Background()

	// Create diverse test files
	testFiles := []struct {
		name     string
		content  []byte
		language string
	}{
		{"main.go", []byte("package main"), "go"},
		{"util.go", []byte("package util"), "go"},
		{"main_test.go", []byte("package main"), "go"},
		{"generated.pb.go", []byte("// Code generated\npackage pb"), "go"},
		{"script.py", []byte("print('hello')"), "python"},
		{"app.js", []byte("console.log('hi')"), "javascript"},
		{"style.css", []byte("body { margin: 0; }"), "css"},
		{"data.json", []byte(`{"key": "value"}`), "json"},
		{"README.md", []byte("# Project"), "markdown"},
		{"binary.exe", []byte{0x00, 0xFF}, "unknown"},
		{".DS_Store", []byte("meta"), "unknown"},
		{"cache.tmp", []byte("temp"), "unknown"},
	}

	filePaths := make([]string, 0, len(testFiles))
	for _, tf := range testFiles {
		path := filepath.Join(tempDir, tf.name)
		err := os.WriteFile(path, tf.content, 0o600)
		require.NoError(t, err)
		filePaths = append(filePaths, path)
	}

	// Get file stats
	stats, err := fc.GetFileStats(ctx, filePaths)
	require.NoError(t, err)
	assert.NotNil(t, stats)

	// Verify counts
	assert.Equal(t, len(testFiles), stats["total"])
	assert.Equal(t, 9, stats["text"])      // All except binary.exe
	assert.Equal(t, 1, stats["binary"])    // binary.exe
	assert.Equal(t, 4, stats["go"])        // All .go files
	assert.Equal(t, 1, stats["generated"]) // generated.pb.go
	assert.Equal(t, 2, stats["excluded"])  // .DS_Store, cache.tmp

	// Verify language stats
	assert.Equal(t, 4, stats["lang_go"])
	assert.Equal(t, 1, stats["lang_python"])
	assert.Equal(t, 1, stats["lang_javascript"])
	assert.Equal(t, 1, stats["lang_css"])
	assert.Equal(t, 1, stats["lang_json"])
	assert.Equal(t, 1, stats["lang_markdown"])

	// Unknown language files are not included in language stats
	_, hasUnknown := stats["lang_unknown"]
	assert.False(t, hasUnknown)
}

// TestHasGeneratedMarker tests generated file marker detection
func TestHasGeneratedMarker(t *testing.T) {
	tempDir := t.TempDir()
	fc := NewFileClassifier(nil)

	tests := []struct {
		name      string
		content   string
		generated bool
	}{
		{
			name: "Standard generated marker",
			content: `// Code generated by tool. DO NOT EDIT.
package main

func main() {}`,
			generated: true,
		},
		{
			name: "Do not edit marker",
			content: `// This file is auto-generated. Do not edit.
package main`,
			generated: true,
		},
		{
			name: "Generated by marker",
			content: `// Generated by protoc-gen-go.
// source: api.proto
package pb`,
			generated: true,
		},
		{
			name: "Automatically generated",
			content: `/*
This file was automatically generated
*/
package main`,
			generated: true,
		},
		{
			name: "Multiple markers",
			content: `// Code generated - DO NOT EDIT
// This file is auto-generated
package main`,
			generated: true,
		},
		{
			name: "Marker in lowercase",
			content: `// code generated by tool
package main`,
			generated: true,
		},
		{
			name: "No marker",
			content: `// Regular source file
package main

// This function generates code
func generateCode() {}`,
			generated: false,
		},
		{
			name: "Marker after line 10",
			content: `package main

import "fmt"

// Line 5
// Line 6
// Line 7
// Line 8
// Line 9
// Line 10
// Line 11 - Code generated
func main() {}`,
			generated: false,
		},
		{
			name:      "Empty file",
			content:   "",
			generated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "test.go")
			err := os.WriteFile(filePath, []byte(tt.content), 0o600)
			require.NoError(t, err)

			result := fc.hasGeneratedMarker(filePath)
			assert.Equal(t, tt.generated, result)
		})
	}

	// Test with non-existent file
	t.Run("Non-existent file", func(t *testing.T) {
		result := fc.hasGeneratedMarker("/non/existent/file.go")
		assert.False(t, result)
	})
}

// TestClassifyFilesPerformance benchmarks file classification
func TestClassifyFilesPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tempDir := t.TempDir()
	fc := NewFileClassifier(nil)
	ctx := context.Background()

	// Create many test files
	const numFiles = 1000
	var filePaths []string

	for i := 0; i < numFiles; i++ {
		var name, content string
		switch i % 5 {
		case 0:
			name = fmt.Sprintf("file%d.go", i)
			content = fmt.Sprintf("package main\n// File %d", i)
		case 1:
			name = fmt.Sprintf("test%d_test.go", i)
			content = fmt.Sprintf("package main\n// Test %d", i)
		case 2:
			name = fmt.Sprintf("data%d.json", i)
			content = fmt.Sprintf(`{"id": %d}`, i)
		case 3:
			name = fmt.Sprintf("doc%d.md", i)
			content = fmt.Sprintf("# Document %d", i)
		default:
			name = fmt.Sprintf("script%d.py", i)
			content = fmt.Sprintf("# Script %d\nprint(%d)", i, i)
		}

		path := filepath.Join(tempDir, name)
		err := os.WriteFile(path, []byte(content), 0o600)
		require.NoError(t, err)
		filePaths = append(filePaths, path)
	}

	// Measure classification time
	start := time.Now()
	results, err := fc.ClassifyFiles(ctx, filePaths)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, results, numFiles)

	// Performance assertion - should process 1000 files in under 1 second
	assert.Less(t, duration, 1*time.Second,
		"Classification of %d files took %v, expected < 1s", numFiles, duration)

	t.Logf("Classified %d files in %v (%.2f files/ms)",
		numFiles, duration, float64(numFiles)/float64(duration.Milliseconds()))
}

// TestConcurrentClassification tests thread safety
func TestConcurrentClassification(t *testing.T) {
	tempDir := t.TempDir()
	fc := NewFileClassifier(nil)
	ctx := context.Background()

	// Create test files
	const numFiles = 50
	var filePaths []string

	for i := 0; i < numFiles; i++ {
		name := fmt.Sprintf("file%d.go", i)
		content := fmt.Sprintf("package main\n// File %d", i)
		path := filepath.Join(tempDir, name)
		err := os.WriteFile(path, []byte(content), 0o600)
		require.NoError(t, err)
		filePaths = append(filePaths, path)
	}

	// Run concurrent classifications
	const numGoroutines = 10
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := fc.ClassifyFiles(ctx, filePaths)
			errors <- err
		}()
	}

	// Check all completed without error
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		assert.NoError(t, err)
	}
}

// TestFileClassifierIntegration tests real-world scenarios
func TestFileClassifierIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Create a mock project structure
	projectFiles := map[string]string{
		"main.go":                                "package main\n\nfunc main() {}",
		"internal/server/server.go":              "package server\n\ntype Server struct{}",
		"internal/server/server_test.go":         "package server\n\nimport \"testing\"",
		"pkg/utils/strings.go":                   "package utils\n\nfunc Reverse(s string) string { return s }",
		"pkg/utils/strings_test.go":              "package utils\n\nimport \"testing\"",
		"api/v1/api.pb.go":                       "// Code generated by protoc-gen-go. DO NOT EDIT.\npackage v1",
		"api/v1/api.pb.gw.go":                    "// Code generated by grpc-gateway. DO NOT EDIT.\npackage v1",
		"vendor/github.com/pkg/errors/errors.go": "package errors",
		"go.mod":                                 "module example.com/project\n\ngo 1.21",
		"go.sum":                                 "// go.sum file",
		"README.md":                              "# Example Project",
		"docs/API.md":                            "# API Documentation",
		".gitignore":                             "*.tmp\n*.log",
		"Makefile":                               "test:\n\tgo test ./...",
		"Dockerfile":                             "FROM golang:1.21",
		".github/workflows/ci.yml":               "name: CI\non: [push]",
		"scripts/build.sh":                       "#!/bin/bash\ngo build",
		"config/config.yaml":                     "server:\n  port: 8080",
		"web/index.html":                         "<html><body>Hello</body></html>",
		"web/style.css":                          "body { margin: 0; }",
		"web/app.js":                             "console.log('app');",
		"test-data/sample.json":                  `{"test": true}`,
		"coverage.out":                           "mode: set",
		".DS_Store":                              "meta",
		"debug.log":                              "2024-01-01 DEBUG message",
		"backup.bak":                             "old content",
	}

	// Create all files
	allPaths := make([]string, 0, len(projectFiles))
	for relPath, content := range projectFiles {
		fullPath := filepath.Join(tempDir, relPath)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0o750)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0o600)
		require.NoError(t, err)
		allPaths = append(allPaths, fullPath)
	}

	// Create file classifier
	cfg := &config.Config{
		MaxFileSize: 1024 * 1024, // 1MB
		Git: struct {
			HooksPath       string
			ExcludePatterns []string
		}{
			HooksPath:       ".git/hooks",
			ExcludePatterns: []string{"test-data/"},
		},
	}
	fc := NewFileClassifier(cfg)
	ctx := context.Background()

	t.Run("Complete project classification", func(t *testing.T) {
		stats, err := fc.GetFileStats(ctx, allPaths)
		require.NoError(t, err)

		// Verify counts
		assert.Equal(t, len(allPaths), stats["total"])
		assert.GreaterOrEqual(t, stats["text"], 20)
		assert.Greater(t, stats["go"], 5)
		assert.Greater(t, stats["generated"], 1)
		assert.Greater(t, stats["excluded"], 3)

		// Verify language diversity
		assert.Greater(t, stats["lang_go"], 5)
		assert.Greater(t, stats["lang_markdown"], 1)
		assert.Greater(t, stats["lang_yaml"], 1)
		assert.Positive(t, stats["lang_html"])
		assert.Positive(t, stats["lang_css"])
		assert.Positive(t, stats["lang_javascript"])
	})

	t.Run("Filter Go files for linting", func(t *testing.T) {
		goFiles, err := fc.FilterGoFiles(ctx, allPaths, true) // Exclude tests
		require.NoError(t, err)

		// Should get only non-test, non-generated, non-vendor Go files
		expectedFiles := []string{
			filepath.Join(tempDir, "main.go"),
			filepath.Join(tempDir, "internal/server/server.go"),
			filepath.Join(tempDir, "pkg/utils/strings.go"),
		}

		assert.ElementsMatch(t, expectedFiles, goFiles)
	})

	t.Run("Filter text files for whitespace check", func(t *testing.T) {
		textFiles, err := fc.FilterTextFiles(ctx, allPaths)
		require.NoError(t, err)

		// Should exclude binary, generated, and system files
		for _, file := range textFiles {
			assert.NotContains(t, file, ".DS_Store")
			assert.NotContains(t, file, "coverage.out")
			assert.NotContains(t, file, ".log")
			assert.NotContains(t, file, ".bak")
			assert.NotContains(t, file, ".pb.go")
			assert.NotContains(t, file, "test-data/") // Custom exclude
		}
	})
}

// Example usage demonstration
func ExampleFileClassifier_ClassifyFiles() {
	// Create a file classifier
	fc := NewFileClassifier(&config.Config{
		MaxFileSize: 10 * 1024 * 1024, // 10MB limit
	})

	// Classify some files
	files := []string{
		"main.go",
		"README.md",
		"vendor/package/file.go",
		"generated.pb.go",
	}

	results, err := fc.ClassifyFiles(context.Background(), files)
	if err != nil {
		panic(err)
	}

	// Process results
	for _, info := range results {
		if info.IsGoFile && !info.Generated && !info.Excluded {
			fmt.Printf("Go source file: %s\n", info.Path)
		}
	}
}

// Benchmark functions
func BenchmarkClassifyFile(b *testing.B) {
	tempDir := b.TempDir()
	fc := NewFileClassifier(nil)

	// Create a test file
	filePath := filepath.Join(tempDir, "test.go")
	content := "package main\n\nfunc main() {}\n"
	err := os.WriteFile(filePath, []byte(content), 0o600)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fc.classifyFile(filePath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIsTextContent(b *testing.B) {
	fc := NewFileClassifier(nil)

	// Test with typical source code
	content := []byte(`package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	os.Exit(0)
}
`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fc.isTextContent(content)
	}
}

func BenchmarkMatchesPattern(b *testing.B) {
	fc := NewFileClassifier(nil)

	patterns := []string{
		"*.go",
		"vendor/",
		"*_test.go",
		"internal/*/*.go",
	}

	files := []string{
		"main.go",
		"vendor/package/file.go",
		"internal/server/server_test.go",
		"README.md",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, file := range files {
			for _, pattern := range patterns {
				_ = fc.matchesPattern(file, pattern)
			}
		}
	}
}
