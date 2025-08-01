package report

import (
	"testing"

	"github.com/mrz1836/go-broadcast/coverage/internal/urlutil"
)

func TestCleanModulePath(t *testing.T) {
	tests := []struct {
		name     string
		fullPath string
		expected string
	}{
		{
			name:     "standard github module path",
			fullPath: "github.com/mrz1836/go-broadcast/internal/algorithms/optimized.go",
			expected: "internal/algorithms/optimized.go",
		},
		{
			name:     "path without module prefix",
			fullPath: "internal/algorithms/optimized.go",
			expected: "internal/algorithms/optimized.go",
		},
		{
			name:     "nested package path",
			fullPath: "github.com/mrz1836/go-broadcast/cmd/go-broadcast/main.go",
			expected: "cmd/go-broadcast/main.go",
		},
		{
			name:     "test file path",
			fullPath: "github.com/mrz1836/go-broadcast/internal/algorithms/optimized_test.go",
			expected: "internal/algorithms/optimized_test.go",
		},
		{
			name:     "root level file",
			fullPath: "github.com/mrz1836/go-broadcast/main.go",
			expected: "main.go",
		},
		{
			name:     "different github repo",
			fullPath: "github.com/someuser/somerepo/pkg/util/helper.go",
			expected: "pkg/util/helper.go",
		},
		{
			name:     "non-github module",
			fullPath: "gitlab.com/user/repo/internal/file.go",
			expected: "gitlab.com/user/repo/internal/file.go",
		},
		{
			name:     "problematic path with cli prefix",
			fullPath: "github.com/mrz1836/go-broadcast/cli/internal/cli/cancel.go",
			expected: "internal/cli/cancel.go",
		},
		{
			name:     "path with cmd prefix repetition",
			fullPath: "github.com/mrz1836/go-broadcast/cmd/internal/cmd/main.go",
			expected: "internal/cmd/main.go",
		},
		{
			name:     "path with no repetition should remain unchanged",
			fullPath: "github.com/mrz1836/go-broadcast/cli/different/path.go",
			expected: "cli/different/path.go",
		},
		{
			name:     "path with repetition at end should remain unchanged",
			fullPath: "github.com/mrz1836/go-broadcast/internal/cli/cli",
			expected: "internal/cli/cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := urlutil.CleanModulePath(tt.fullPath)
			if result != tt.expected {
				t.Errorf("CleanModulePath(%q) = %q, want %q", tt.fullPath, result, tt.expected)
			}
		})
	}
}

func TestGenerator_FileURLConstruction(t *testing.T) {
	// Test that we construct correct GitHub URLs
	tests := []struct {
		name        string
		owner       string
		repo        string
		branch      string
		packagePath string
		fileName    string
		expectedURL string
	}{
		{
			name:        "standard file URL",
			owner:       "mrz1836",
			repo:        "go-broadcast",
			branch:      "master",
			packagePath: "github.com/mrz1836/go-broadcast/internal/algorithms",
			fileName:    "optimized.go",
			expectedURL: "https://github.com/mrz1836/go-broadcast/blob/master/internal/algorithms/optimized.go",
		},
		{
			name:        "nested package URL",
			owner:       "mrz1836",
			repo:        "go-broadcast",
			branch:      "main",
			packagePath: "github.com/mrz1836/go-broadcast/cmd/go-broadcast",
			fileName:    "main.go",
			expectedURL: "https://github.com/mrz1836/go-broadcast/blob/main/cmd/go-broadcast/main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from generateData using the new utility function
			fullPath := tt.packagePath + "/" + tt.fileName
			fileURL := urlutil.BuildGitHubFileURL(tt.owner, tt.repo, tt.branch, fullPath)

			if fileURL != tt.expectedURL {
				t.Errorf("File URL construction failed:\ngot:  %q\nwant: %q", fileURL, tt.expectedURL)
			}
		})
	}
}
