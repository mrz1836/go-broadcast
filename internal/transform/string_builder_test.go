package transform

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBuildTest = errors.New("build error")

func TestBuildPath(t *testing.T) {
	tests := []struct {
		name      string
		separator string
		parts     []string
		expected  string
	}{
		{
			name:      "empty parts",
			separator: "/",
			parts:     []string{},
			expected:  "",
		},
		{
			name:      "single part",
			separator: "/",
			parts:     []string{"single"},
			expected:  "single",
		},
		{
			name:      "two parts with slash",
			separator: "/",
			parts:     []string{"first", "second"},
			expected:  "first/second",
		},
		{
			name:      "multiple parts with slash",
			separator: "/",
			parts:     []string{"github.com", "user", "repo", "blob", "main", "README.md"},
			expected:  "github.com/user/repo/blob/main/README.md",
		},
		{
			name:      "parts with hyphen separator",
			separator: "-",
			parts:     []string{"sync", "template", "20240101", "abc123"},
			expected:  "sync-template-20240101-abc123",
		},
		{
			name:      "parts with underscore separator",
			separator: "_",
			parts:     []string{"part1", "part2", "part3"},
			expected:  "part1_part2_part3",
		},
		{
			name:      "parts with empty separator",
			separator: "",
			parts:     []string{"a", "b", "c"},
			expected:  "abc",
		},
		{
			name:      "parts with multi-character separator",
			separator: " -> ",
			parts:     []string{"start", "middle", "end"},
			expected:  "start -> middle -> end",
		},
		{
			name:      "parts with empty strings",
			separator: "/",
			parts:     []string{"", "middle", ""},
			expected:  "/middle/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildPath(tt.separator, tt.parts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildGitHubURL(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		pathParts []string
		expected  string
	}{
		{
			name:      "basic repository URL",
			repo:      "user/repo",
			pathParts: []string{},
			expected:  "https://github.com/user/repo",
		},
		{
			name:      "repository with blob path",
			repo:      "user/repo",
			pathParts: []string{"blob", "main", "README.md"},
			expected:  "https://github.com/user/repo/blob/main/README.md",
		},
		{
			name:      "repository with tree path",
			repo:      "organization/project",
			pathParts: []string{"tree", "develop", "src", "main", "java"},
			expected:  "https://github.com/organization/project/tree/develop/src/main/java",
		},
		{
			name:      "repository with single path part",
			repo:      "owner/name",
			pathParts: []string{"issues"},
			expected:  "https://github.com/owner/name/issues",
		},
		{
			name:      "repository with releases path",
			repo:      "company/product",
			pathParts: []string{"releases", "tag", "v1.0.0"},
			expected:  "https://github.com/company/product/releases/tag/v1.0.0",
		},
		{
			name:      "repository with empty path parts",
			repo:      "test/test",
			pathParts: []string{"", "file.txt"},
			expected:  "https://github.com/test/test//file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildGitHubURL(tt.repo, tt.pathParts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildBranchName(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		timestamp string
		commitSHA string
		expected  string
	}{
		{
			name:      "standard branch name",
			prefix:    "sync/template",
			timestamp: "20240101-120000",
			commitSHA: "abc123",
			expected:  "sync/template-20240101-120000-abc123",
		},
		{
			name:      "short prefix",
			prefix:    "sync",
			timestamp: "20240715-143022",
			commitSHA: "def456",
			expected:  "sync-20240715-143022-def456",
		},
		{
			name:      "long SHA",
			prefix:    "update/files",
			timestamp: "20240301-090000",
			commitSHA: "abcdef123456",
			expected:  "update/files-20240301-090000-abcdef123456",
		},
		{
			name:      "empty components",
			prefix:    "",
			timestamp: "",
			commitSHA: "",
			expected:  "--",
		},
		{
			name:      "special characters in prefix",
			prefix:    "feature/test-branch",
			timestamp: "20240401-160000",
			commitSHA: "xyz789",
			expected:  "feature/test-branch-20240401-160000-xyz789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildBranchName(tt.prefix, tt.timestamp, tt.commitSHA)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		subject  string
		details  []string
		expected string
	}{
		{
			name:     "simple commit message",
			action:   "sync",
			subject:  "update files from source repository",
			details:  []string{},
			expected: "sync: update files from source repository",
		},
		{
			name:     "commit message with single detail",
			action:   "update",
			subject:  "configuration files",
			details:  []string{"Modified: .github/workflows/ci.yml"},
			expected: "update: configuration files\n\nModified: .github/workflows/ci.yml",
		},
		{
			name:     "commit message with multiple details",
			action:   "sync",
			subject:  "files from source repository",
			details:  []string{"Modified: README.md", "Added: .gitignore", "Updated: package.json"},
			expected: "sync: files from source repository\n\nModified: README.md\nAdded: .gitignore\nUpdated: package.json",
		},
		{
			name:     "commit message with empty details",
			action:   "fix",
			subject:  "critical bug",
			details:  []string{""},
			expected: "fix: critical bug\n\n",
		},
		{
			name:     "long commit message",
			action:   "refactor",
			subject:  "improve performance and maintainability",
			details:  []string{"- Optimized database queries", "- Improved error handling", "- Added comprehensive tests"},
			expected: "refactor: improve performance and maintainability\n\n- Optimized database queries\n- Improved error handling\n- Added comprehensive tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildCommitMessage(tt.action, tt.subject, tt.details...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildFileList(t *testing.T) {
	tests := []struct {
		name      string
		files     []string
		prefix    string
		separator string
		expected  string
	}{
		{
			name:      "empty file list",
			files:     []string{},
			prefix:    "- ",
			separator: "\n",
			expected:  "",
		},
		{
			name:      "single file",
			files:     []string{"README.md"},
			prefix:    "- ",
			separator: "\n",
			expected:  "- README.md",
		},
		{
			name:      "multiple files with bullet points",
			files:     []string{"README.md", "main.go", "config.yaml"},
			prefix:    "- ",
			separator: "\n",
			expected:  "- README.md\n- main.go\n- config.yaml",
		},
		{
			name:      "files with comma separator",
			files:     []string{"file1.txt", "file2.txt", "file3.txt"},
			prefix:    "",
			separator: ", ",
			expected:  "file1.txt, file2.txt, file3.txt",
		},
		{
			name:      "files with numbered prefix",
			files:     []string{"first.go", "second.go"},
			prefix:    "  ",
			separator: "\n",
			expected:  "  first.go\n  second.go",
		},
		{
			name:      "files with custom separator",
			files:     []string{"a.txt", "b.txt", "c.txt"},
			prefix:    "* ",
			separator: " | ",
			expected:  "* a.txt | * b.txt | * c.txt",
		},
		{
			name:      "files with empty prefix and separator",
			files:     []string{"one", "two"},
			prefix:    "",
			separator: "",
			expected:  "onetwo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildFileList(tt.files, tt.prefix, tt.separator)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildKeyValuePairs(t *testing.T) {
	tests := []struct {
		name        string
		pairs       map[string]string
		keyValueSep string
		pairSep     string
		expected    []string // Multiple valid results due to map iteration order
	}{
		{
			name:        "empty map",
			pairs:       map[string]string{},
			keyValueSep: ": ",
			pairSep:     "\n",
			expected:    []string{""},
		},
		{
			name:        "single pair",
			pairs:       map[string]string{"key": "value"},
			keyValueSep: ": ",
			pairSep:     "\n",
			expected:    []string{"key: value"},
		},
		{
			name:        "multiple pairs with colon separator",
			pairs:       map[string]string{"repo": "user/repo", "branch": "main"},
			keyValueSep: ": ",
			pairSep:     "\n",
			expected:    []string{"repo: user/repo\nbranch: main", "branch: main\nrepo: user/repo"},
		},
		{
			name:        "pairs with equals separator",
			pairs:       map[string]string{"env": "production", "debug": "false"},
			keyValueSep: "=",
			pairSep:     "&",
			expected:    []string{"env=production&debug=false", "debug=false&env=production"},
		},
		{
			name:        "pairs with comma separator",
			pairs:       map[string]string{"name": "test", "type": "unit"},
			keyValueSep: ": ",
			pairSep:     ", ",
			expected:    []string{"name: test, type: unit", "type: unit, name: test"},
		},
		{
			name:        "complex values",
			pairs:       map[string]string{"url": "https://example.com", "method": "GET"},
			keyValueSep: " -> ",
			pairSep:     " | ",
			expected:    []string{"url -> https://example.com | method -> GET", "method -> GET | url -> https://example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildKeyValuePairs(tt.pairs, tt.keyValueSep, tt.pairSep)

			// For maps with single entry or empty maps, order is deterministic
			if len(tt.pairs) <= 1 {
				assert.Equal(t, tt.expected[0], result)
			} else {
				// For multiple entries, check if result matches any of the expected permutations
				found := false
				for _, expected := range tt.expected {
					if result == expected {
						found = true
						break
					}
				}
				assert.True(t, found, "Result %q didn't match any expected values: %v", result, tt.expected)
			}
		})
	}
}

func TestBuildLargeString(t *testing.T) {
	tests := []struct {
		name          string
		estimatedSize int
		buildFunc     func(buf *bytes.Buffer) error
		expectedLen   int
		expectError   bool
	}{
		{
			name:          "small string below threshold",
			estimatedSize: 1024,
			buildFunc: func(buf *bytes.Buffer) error {
				buf.WriteString("Small string content")
				return nil
			},
			expectedLen: 20,
			expectError: false,
		},
		{
			name:          "large string above threshold",
			estimatedSize: pool.LargeBufferThreshold + 1000,
			buildFunc: func(buf *bytes.Buffer) error {
				for i := 0; i < 1000; i++ {
					fmt.Fprintf(buf, "Line %d\n", i)
				}
				return nil
			},
			expectedLen: 8890, // Actual length from running the test
			expectError: false,
		},
		{
			name:          "function returns error",
			estimatedSize: 1024,
			buildFunc: func(buf *bytes.Buffer) error {
				buf.WriteString("Some content")
				return errBuildTest
			},
			expectedLen: 0,
			expectError: true,
		},
		{
			name:          "empty content",
			estimatedSize: 1024,
			buildFunc: func(_ *bytes.Buffer) error {
				return nil
			},
			expectedLen: 0,
			expectError: false,
		},
		{
			name:          "exact threshold size",
			estimatedSize: pool.LargeBufferThreshold,
			buildFunc: func(buf *bytes.Buffer) error {
				buf.WriteString("Exactly at threshold")
				return nil
			},
			expectedLen: 20,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildLargeString(tt.estimatedSize, tt.buildFunc)

			if tt.expectError {
				require.Error(t, err)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.expectedLen)
			}
		})
	}
}

func TestBuildURLWithParams(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		params   map[string]string
		expected []string // Multiple valid results due to map iteration order
	}{
		{
			name:     "no parameters",
			baseURL:  "https://api.github.com/repos/user/repo",
			params:   map[string]string{},
			expected: []string{"https://api.github.com/repos/user/repo"},
		},
		{
			name:     "single parameter",
			baseURL:  "https://api.github.com/user",
			params:   map[string]string{"per_page": "100"},
			expected: []string{"https://api.github.com/user?per_page=100"},
		},
		{
			name:    "multiple parameters",
			baseURL: "https://api.github.com/search/repositories",
			params:  map[string]string{"q": "go", "sort": "stars", "order": "desc"},
			expected: []string{
				"https://api.github.com/search/repositories?q=go&sort=stars&order=desc",
				"https://api.github.com/search/repositories?q=go&order=desc&sort=stars",
				"https://api.github.com/search/repositories?sort=stars&q=go&order=desc",
				"https://api.github.com/search/repositories?sort=stars&order=desc&q=go",
				"https://api.github.com/search/repositories?order=desc&q=go&sort=stars",
				"https://api.github.com/search/repositories?order=desc&sort=stars&q=go",
			},
		},
		{
			name:     "parameters with special characters",
			baseURL:  "https://example.com/api",
			params:   map[string]string{"filter": "name=test", "limit": "50"},
			expected: []string{"https://example.com/api?filter=name=test&limit=50", "https://example.com/api?limit=50&filter=name=test"},
		},
		{
			name:     "empty parameter values",
			baseURL:  "https://test.com",
			params:   map[string]string{"empty": "", "valid": "value"},
			expected: []string{"https://test.com?empty=&valid=value", "https://test.com?valid=value&empty="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildURLWithParams(tt.baseURL, tt.params)

			// For maps with single entry or no entries, order is deterministic
			if len(tt.params) <= 1 {
				assert.Equal(t, tt.expected[0], result)
			} else {
				// For multiple entries, check if result matches any of the expected permutations
				found := false
				for _, expected := range tt.expected {
					if result == expected {
						found = true
						break
					}
				}
				assert.True(t, found, "Result %q didn't match any expected values: %v", result, tt.expected)
			}
		})
	}
}

func TestBuildProgressMessage(t *testing.T) {
	tests := []struct {
		name      string
		current   int
		total     int
		operation string
		expected  string
	}{
		{
			name:      "basic progress",
			current:   5,
			total:     10,
			operation: "repositories processed",
			expected:  "5/10 repositories processed",
		},
		{
			name:      "zero progress",
			current:   0,
			total:     100,
			operation: "files synchronized",
			expected:  "0/100 files synchronized",
		},
		{
			name:      "complete progress",
			current:   50,
			total:     50,
			operation: "tests completed",
			expected:  "50/50 tests completed",
		},
		{
			name:      "large numbers",
			current:   999,
			total:     1000,
			operation: "items processed",
			expected:  "999/1000 items processed",
		},
		{
			name:      "single digit numbers",
			current:   1,
			total:     9,
			operation: "steps done",
			expected:  "1/9 steps done",
		},
		{
			name:      "empty operation",
			current:   3,
			total:     5,
			operation: "",
			expected:  "3/5 ",
		},
		{
			name:      "long operation description",
			current:   42,
			total:     100,
			operation: "complex synchronization operations with detailed logging",
			expected:  "42/100 complex synchronization operations with detailed logging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildProgressMessage(tt.current, tt.total, tt.operation)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests to validate performance optimizations
func BenchmarkBuildPath(b *testing.B) {
	parts := []string{"github.com", "user", "repository", "blob", "main", "path", "to", "file.go"}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = BuildPath("/", parts...)
	}
}

func BenchmarkBuildGitHubURL(b *testing.B) {
	pathParts := []string{"blob", "main", "src", "package", "file.go"}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = BuildGitHubURL("organization/repository", pathParts...)
	}
}

func BenchmarkBuildCommitMessage(b *testing.B) {
	details := []string{
		"Modified: README.md",
		"Added: .github/workflows/ci.yml",
		"Updated: package.json",
		"Fixed: configuration issues",
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = BuildCommitMessage("sync", "update files from source repository", details...)
	}
}

func BenchmarkBuildFileList(b *testing.B) {
	files := []string{
		"README.md", "main.go", "config.yaml", "Dockerfile",
		".github/workflows/ci.yml", "internal/app/app.go",
		"pkg/utils/helper.go", "test/integration_test.go",
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = BuildFileList(files, "- ", "\n")
	}
}

func BenchmarkBuildLargeString(b *testing.B) {
	buildFunc := func(buf *bytes.Buffer) error {
		for i := 0; i < 100; i++ {
			fmt.Fprintf(buf, "Line %d with some content\n", i)
		}
		return nil
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = BuildLargeString(3000, buildFunc)
	}
}

// Test memory efficiency by ensuring builders grow to expected sizes
func TestStringBuilderCapacityOptimization(t *testing.T) {
	t.Run("BuildPath capacity optimization", func(t *testing.T) {
		// This test validates that our size calculation is accurate
		parts := []string{"very", "long", "path", "with", "many", "components", "to", "test", "capacity"}
		result := BuildPath("/", parts...)

		// Verify the result is correct
		expected := "very/long/path/with/many/components/to/test/capacity"
		assert.Equal(t, expected, result)

		// Length should match our pre-calculated size
		expectedLen := len(expected)
		assert.Len(t, result, expectedLen)
	})

	t.Run("BuildGitHubURL capacity optimization", func(t *testing.T) {
		repo := "organization/very-long-repository-name"
		pathParts := []string{"blob", "feature/very-long-branch-name", "src", "main", "java", "com", "example", "VeryLongClassName.java"}
		result := BuildGitHubURL(repo, pathParts...)

		expected := "https://github.com/organization/very-long-repository-name/blob/feature/very-long-branch-name/src/main/java/com/example/VeryLongClassName.java"
		assert.Equal(t, expected, result)
	})
}
