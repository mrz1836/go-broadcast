package strutil

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "EmptyString",
			input:    "",
			expected: true,
		},
		{
			name:     "WhitespaceOnly",
			input:    "   \t\n  ",
			expected: true,
		},
		{
			name:     "NonEmptyString",
			input:    "hello",
			expected: false,
		},
		{
			name:     "StringWithWhitespace",
			input:    "  hello  ",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsEmpty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNotEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "EmptyString",
			input:    "",
			expected: false,
		},
		{
			name:     "WhitespaceOnly",
			input:    "   \t\n  ",
			expected: false,
		},
		{
			name:     "NonEmptyString",
			input:    "hello",
			expected: true,
		},
		{
			name:     "StringWithWhitespace",
			input:    "  hello  ",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotEmpty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEmptyToDefault(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue string
		expected     string
	}{
		{
			name:         "EmptyValueReturnsDefault",
			value:        "",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "WhitespaceValueReturnsDefault",
			value:        "   ",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "NonEmptyValueReturnsValue",
			value:        "  actual  ",
			defaultValue: "default",
			expected:     "actual",
		},
		{
			name:         "NonEmptyValueTrimsWhitespace",
			value:        "actual",
			defaultValue: "default",
			expected:     "actual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EmptyToDefault(tt.value, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrimAndLower(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "UppercaseWithWhitespace",
			input:    "  HELLO  ",
			expected: "hello",
		},
		{
			name:     "MixedCase",
			input:    "Hello World",
			expected: "hello world",
		},
		{
			name:     "EmptyString",
			input:    "",
			expected: "",
		},
		{
			name:     "WhitespaceOnly",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrimAndLower(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		substrings []string
		expected   bool
	}{
		{
			name:       "ContainsFirstSubstring",
			text:       "hello world",
			substrings: []string{"hello", "foo", "bar"},
			expected:   true,
		},
		{
			name:       "ContainsMiddleSubstring",
			text:       "hello world",
			substrings: []string{"foo", "world", "bar"},
			expected:   true,
		},
		{
			name:       "ContainsNoSubstrings",
			text:       "hello world",
			substrings: []string{"foo", "bar", "baz"},
			expected:   false,
		},
		{
			name:       "EmptySubstrings",
			text:       "hello world",
			substrings: []string{},
			expected:   false,
		},
		{
			name:       "EmptyText",
			text:       "",
			substrings: []string{"hello"},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsAny(tt.text, tt.substrings...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasAnyPrefix(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		prefixes []string
		expected bool
	}{
		{
			name:     "HasFirstPrefix",
			text:     "hello world",
			prefixes: []string{"hello", "foo", "bar"},
			expected: true,
		},
		{
			name:     "HasSecondPrefix",
			text:     "hello world",
			prefixes: []string{"foo", "hell", "bar"},
			expected: true,
		},
		{
			name:     "HasNoPrefixes",
			text:     "hello world",
			prefixes: []string{"foo", "bar", "baz"},
			expected: false,
		},
		{
			name:     "EmptyPrefixes",
			text:     "hello world",
			prefixes: []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasAnyPrefix(tt.text, tt.prefixes...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasAnySuffix(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		suffixes []string
		expected bool
	}{
		{
			name:     "HasFirstSuffix",
			text:     "hello world",
			suffixes: []string{"world", "foo", "bar"},
			expected: true,
		},
		{
			name:     "HasSecondSuffix",
			text:     "hello world",
			suffixes: []string{"foo", "orld", "bar"},
			expected: true,
		},
		{
			name:     "HasNoSuffixes",
			text:     "hello world",
			suffixes: []string{"foo", "bar", "baz"},
			expected: false,
		},
		{
			name:     "EmptySuffixes",
			text:     "hello world",
			suffixes: []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasAnySuffix(tt.text, tt.suffixes...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatRepoName(t *testing.T) {
	tests := []struct {
		name     string
		org      string
		repo     string
		expected string
	}{
		{
			name:     "StandardRepoName",
			org:      "myorg",
			repo:     "myrepo",
			expected: "myorg/myrepo",
		},
		{
			name:     "EmptyOrg",
			org:      "",
			repo:     "myrepo",
			expected: "/myrepo",
		},
		{
			name:     "EmptyRepo",
			org:      "myorg",
			repo:     "",
			expected: "myorg/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatRepoName(tt.org, tt.repo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFilePath(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{
			name:     "MultipleParts",
			parts:    []string{"path", "to", "file.txt"},
			expected: "path/to/file.txt",
		},
		{
			name:     "SinglePart",
			parts:    []string{"file.txt"},
			expected: "file.txt",
		},
		{
			name:     "EmptyParts",
			parts:    []string{},
			expected: "",
		},
		{
			name:     "WithEmptyStrings",
			parts:    []string{"path", "", "file.txt"},
			expected: "path/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFilePath(tt.parts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "CleanDotDotPaths",
			path:     "path/to/../file.txt",
			expected: "path/file.txt",
		},
		{
			name:     "CleanDotPaths",
			path:     "path/./to/file.txt",
			expected: "path/to/file.txt",
		},
		{
			name:     "AlreadyNormalized",
			path:     "path/to/file.txt",
			expected: "path/to/file.txt",
		},
	}

	// Only test backslash conversion on Windows
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name     string
			path     string
			expected string
		}{
			name:     "BackslashesToForwardSlashes",
			path:     "path\\to\\file.txt",
			expected: "path/to/file.txt",
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeForFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ProblematicCharacters",
			input:    "file/name\\with:problems",
			expected: "file-name-with-problems",
		},
		{
			name:     "AllProblematicCharacters",
			input:    `file/\:*?"<>|name`,
			expected: "file---------name",
		},
		{
			name:     "WithWhitespace",
			input:    "  file name  ",
			expected: "file name",
		},
		{
			name:     "AlreadyClean",
			input:    "clean-filename",
			expected: "clean-filename",
		},
		{
			name:     "NullByte",
			input:    "file\x00name",
			expected: "file-name",
		},
		{
			name:     "ControlCharacters",
			input:    "file\x01\x02\x1fname",
			expected: "file---name",
		},
		{
			name:     "NewlineAndTab",
			input:    "file\nwith\ttabs",
			expected: "file-with-tabs",
		},
		{
			name:     "DELCharacter",
			input:    "file\x7fname",
			expected: "file-name",
		},
		{
			name:     "EmptyResult",
			input:    "   ",
			expected: "unnamed",
		},
		{
			name:     "AllProblematicResultsEmpty",
			input:    "///",
			expected: "---",
		},
		{
			name:     "EmptyInput",
			input:    "",
			expected: "unnamed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidGitHubURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "ValidGitHubURL",
			url:      "https://github.com/owner/repo",
			expected: true,
		},
		{
			name:     "InvalidScheme",
			url:      "http://github.com/owner/repo",
			expected: false,
		},
		{
			name:     "InvalidHost",
			url:      "https://gitlab.com/owner/repo",
			expected: false,
		},
		{
			name:     "PathTraversal",
			url:      "https://github.com/../../../etc/passwd",
			expected: false,
		},
		{
			name:     "EmptyURL",
			url:      "",
			expected: false,
		},
		{
			name:     "InvalidURL",
			url:      "not-a-url",
			expected: false,
		},
		{
			name:     "RepoNameWithDoubleDots",
			url:      "https://github.com/owner/my..repo",
			expected: true, // ".." in repo name is valid, not path traversal
		},
		{
			name:     "RepoNameWithTripleDots",
			url:      "https://github.com/owner/repo...name",
			expected: true, // "..." in repo name is valid
		},
		{
			name:     "ActualPathTraversalInPath",
			url:      "https://github.com/owner/../other/repo",
			expected: false, // Actual path traversal attempt
		},
		{
			name:     "ValidGitHubURLNoPath",
			url:      "https://github.com",
			expected: true, // Valid GitHub domain
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidGitHubURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceTemplateVars(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		replacements map[string]string
		expected     string
	}{
		{
			name:    "MultipleReplacements",
			content: "Hello {{NAME}}, welcome to {{SITE}}!",
			replacements: map[string]string{
				"{{NAME}}": "John",
				"{{SITE}}": "GitHub",
			},
			expected: "Hello John, welcome to GitHub!",
		},
		{
			name:    "NoReplacements",
			content: "Hello world!",
			replacements: map[string]string{
				"{{NAME}}": "John",
			},
			expected: "Hello world!",
		},
		{
			name:         "EmptyReplacements",
			content:      "Hello {{NAME}}!",
			replacements: map[string]string{},
			expected:     "Hello {{NAME}}!",
		},
		{
			name:    "EmptyContent",
			content: "",
			replacements: map[string]string{
				"{{NAME}}": "John",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ReplaceTemplateVars(tt.content, tt.replacements)
			assert.Equal(t, tt.expected, result)
		})
	}
}
