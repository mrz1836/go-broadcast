package strutil

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name     string
		elements []string
		expected string
	}{
		{
			name:     "MultipleElements",
			elements: []string{"path", "to", "file.txt"},
			expected: "path/to/file.txt",
		},
		{
			name:     "SingleElement",
			elements: []string{"file.txt"},
			expected: "file.txt",
		},
		{
			name:     "EmptyElements",
			elements: []string{},
			expected: "",
		},
		{
			name:     "WithEmptyStrings",
			elements: []string{"path", "", "file.txt"},
			expected: "path/file.txt",
		},
		{
			name:     "WithDotDot",
			elements: []string{"path", "..", "file.txt"},
			expected: "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JoinPath(tt.elements...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBaseName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "FilePath",
			path:     "path/to/file.txt",
			expected: "file.txt",
		},
		{
			name:     "DirectoryPath",
			path:     "path/to/directory/",
			expected: "directory",
		},
		{
			name:     "SingleFile",
			path:     "file.txt",
			expected: "file.txt",
		},
		{
			name:     "EmptyPath",
			path:     "",
			expected: "",
		},
		{
			name:     "RootPath",
			path:     "/",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBaseName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDirName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "FilePath",
			path:     "path/to/file.txt",
			expected: "path/to",
		},
		{
			name:     "DirectoryPath",
			path:     "path/to/directory/",
			expected: "path/to/directory",
		},
		{
			name:     "SingleFile",
			path:     "file.txt",
			expected: ".",
		},
		{
			name:     "EmptyPath",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDirName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAbsolutePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "RelativePath",
			path:     "path/to/file.txt",
			expected: false,
		},
		{
			name:     "CurrentDirectory",
			path:     "./file.txt",
			expected: false,
		},
		{
			name:     "ParentDirectory",
			path:     "../file.txt",
			expected: false,
		},
	}

	// Platform-specific absolute path tests
	if runtime.GOOS == "windows" {
		tests = append(tests, []struct {
			name     string
			path     string
			expected bool
		}{
			{
				name:     "WindowsAbsolutePath",
				path:     "C:\\path\\to\\file.txt",
				expected: true,
			},
			{
				name:     "WindowsUNCPath",
				path:     "\\\\server\\share\\file.txt",
				expected: true,
			},
		}...)
	} else {
		tests = append(tests, struct {
			name     string
			path     string
			expected bool
		}{
			name:     "UnixAbsolutePath",
			path:     "/path/to/file.txt",
			expected: true,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAbsolutePath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasPathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "SafePath",
			path:     "path/to/file.txt",
			expected: false,
		},
		{
			name:     "PathTraversalAttempt",
			path:     "path/../../../etc/passwd",
			expected: true,
		},
		{
			name:     "PathTraversalAtStart",
			path:     "../file.txt",
			expected: true,
		},
		{
			name:     "DotDotInPath",
			path:     "path/to/../file.txt",
			expected: false, // This is cleaned to "path/file.txt", which is safe
		},
		{
			name:     "MultipleDotDot",
			path:     "../../file.txt",
			expected: true,
		},
		{
			name:     "DotDotInFilename",
			path:     "path/to/file..txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPathTraversal(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsHiddenFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "HiddenFile",
			path:     ".gitignore",
			expected: true,
		},
		{
			name:     "HiddenFileInPath",
			path:     "path/to/.hidden",
			expected: true,
		},
		{
			name:     "RegularFile",
			path:     "file.txt",
			expected: false,
		},
		{
			name:     "CurrentDirectory",
			path:     ".",
			expected: false,
		},
		{
			name:     "ParentDirectory",
			path:     "..",
			expected: false,
		},
		{
			name:     "DotInFilename",
			path:     "file.name.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHiddenFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToUnixPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "UnixPath",
			path:     "path/to/file.txt",
			expected: "path/to/file.txt",
		},
		{
			name:     "EmptyPath",
			path:     "",
			expected: "",
		},
	}

	// Only test backslash conversion on Windows
	if runtime.GOOS == "windows" {
		tests = append(tests, []struct {
			name     string
			path     string
			expected string
		}{
			{
				name:     "WindowsPath",
				path:     "path\\to\\file.txt",
				expected: "path/to/file.txt",
			},
			{
				name:     "MixedPath",
				path:     "path\\to/file.txt",
				expected: "path/to/file.txt",
			},
		}...)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToUnixPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasExtension(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		extensions []string
		expected   bool
	}{
		{
			name:       "MatchingExtension",
			path:       "file.txt",
			extensions: []string{".txt", ".md", ".go"},
			expected:   true,
		},
		{
			name:       "CaseInsensitiveMatch",
			path:       "file.TXT",
			extensions: []string{".txt", ".md"},
			expected:   true,
		},
		{
			name:       "NoMatchingExtension",
			path:       "file.pdf",
			extensions: []string{".txt", ".md", ".go"},
			expected:   false,
		},
		{
			name:       "NoExtension",
			path:       "file",
			extensions: []string{".txt", ".md"},
			expected:   false,
		},
		{
			name:       "EmptyExtensions",
			path:       "file.txt",
			extensions: []string{},
			expected:   false,
		},
		{
			name:       "ExtensionWithoutDot",
			path:       "file.txt",
			extensions: []string{"txt", "md"},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasExtension(tt.path, tt.extensions...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureTrailingSlash(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "PathWithoutSlash",
			path:     "path/to/directory",
			expected: "path/to/directory/",
		},
		{
			name:     "PathWithSlash",
			path:     "path/to/directory/",
			expected: "path/to/directory/",
		},
		{
			name:     "EmptyPath",
			path:     "",
			expected: "/",
		},
		{
			name:     "RootPath",
			path:     "/",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsureTrailingSlash(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRemoveTrailingSlash(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "PathWithSlash",
			path:     "path/to/directory/",
			expected: "path/to/directory",
		},
		{
			name:     "PathWithoutSlash",
			path:     "path/to/directory",
			expected: "path/to/directory",
		},
		{
			name:     "RootPath",
			path:     "/",
			expected: "",
		},
		{
			name:     "EmptyPath",
			path:     "",
			expected: "",
		},
		{
			name:     "MultipleTrailingSlashes",
			path:     "path/to/directory///",
			expected: "path/to/directory",
		},
		{
			name:     "OnlySlashes",
			path:     "///",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveTrailingSlash(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "StandardPath",
			path:     "path/to/file.txt",
			expected: []string{"path", "to", "file.txt"},
		},
		{
			name:     "PathWithDotDot",
			path:     "path/to/../file.txt",
			expected: []string{"path", "file.txt"},
		},
		{
			name:     "PathWithDot",
			path:     "path/./to/file.txt",
			expected: []string{"path", "to", "file.txt"},
		},
		{
			name:     "EmptyPath",
			path:     "",
			expected: nil,
		},
		{
			name:     "SingleComponent",
			path:     "file.txt",
			expected: []string{"file.txt"},
		},
		{
			name:     "PathWithEmptyComponents",
			path:     "path//to///file.txt",
			expected: []string{"path", "to", "file.txt"},
		},
		{
			name:     "AbsoluteUnixPath",
			path:     "/path/to/file.txt",
			expected: []string{"path", "to", "file.txt"},
		},
		{
			name:     "RootOnly",
			path:     "/",
			expected: nil,
		},
	}

	// Add Windows-specific test only on Windows
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name     string
			path     string
			expected []string
		}{
			name:     "WindowsPath",
			path:     "path\\to\\file.txt",
			expected: []string{"path", "to", "file.txt"},
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitPath(tt.path)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
