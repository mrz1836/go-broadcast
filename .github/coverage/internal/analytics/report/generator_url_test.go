package report

import (
	"testing"
)

func TestGenerator_stripModulePrefix(t *testing.T) {
	g := &Generator{}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.stripModulePrefix(tt.fullPath)
			if result != tt.expected {
				t.Errorf("stripModulePrefix(%q) = %q, want %q", tt.fullPath, result, tt.expected)
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
			g := &Generator{}

			// Simulate the logic from generateData
			fullPath := tt.packagePath + "/" + tt.fileName
			relativePath := g.stripModulePrefix(fullPath)
			fileURL := "https://github.com/" + tt.owner + "/" + tt.repo + "/blob/" + tt.branch + "/" + relativePath

			if fileURL != tt.expectedURL {
				t.Errorf("File URL construction failed:\ngot:  %q\nwant: %q", fileURL, tt.expectedURL)
			}
		})
	}
}
