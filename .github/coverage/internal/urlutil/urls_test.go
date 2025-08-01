package urlutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildGitHubCommitURL(t *testing.T) {
	tests := []struct {
		name      string
		owner     string
		repo      string
		commitSHA string
		expected  string
	}{
		{
			name:      "valid inputs",
			owner:     "mrz1836",
			repo:      "go-broadcast",
			commitSHA: "abc123def456",
			expected:  "https://github.com/mrz1836/go-broadcast/commit/abc123def456",
		},
		{
			name:      "empty owner",
			owner:     "",
			repo:      "go-broadcast",
			commitSHA: "abc123def456",
			expected:  "",
		},
		{
			name:      "empty repo",
			owner:     "mrz1836",
			repo:      "",
			commitSHA: "abc123def456",
			expected:  "",
		},
		{
			name:      "empty commitSHA",
			owner:     "mrz1836",
			repo:      "go-broadcast",
			commitSHA: "",
			expected:  "",
		},
		{
			name:      "all empty",
			owner:     "",
			repo:      "",
			commitSHA: "",
			expected:  "",
		},
		{
			name:      "special characters in inputs",
			owner:     "user-name",
			repo:      "repo.name",
			commitSHA: "1a2b3c4d5e6f",
			expected:  "https://github.com/user-name/repo.name/commit/1a2b3c4d5e6f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildGitHubCommitURL(tt.owner, tt.repo, tt.commitSHA)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildCoverageBadgeURL(t *testing.T) {
	tests := []struct {
		name       string
		percentage float64
		expected   string
	}{
		{
			name:       "perfect coverage",
			percentage: 100.0,
			expected:   "https://img.shields.io/badge/coverage-100.0%25-brightgreen",
		},
		{
			name:       "excellent coverage",
			percentage: 95.5,
			expected:   "https://img.shields.io/badge/coverage-95.5%25-brightgreen",
		},
		{
			name:       "minimum brightgreen",
			percentage: 90.0,
			expected:   "https://img.shields.io/badge/coverage-90.0%25-brightgreen",
		},
		{
			name:       "good coverage",
			percentage: 85.3,
			expected:   "https://img.shields.io/badge/coverage-85.3%25-green",
		},
		{
			name:       "minimum green",
			percentage: 80.0,
			expected:   "https://img.shields.io/badge/coverage-80.0%25-green",
		},
		{
			name:       "acceptable coverage",
			percentage: 75.7,
			expected:   "https://img.shields.io/badge/coverage-75.7%25-yellowgreen",
		},
		{
			name:       "minimum yellowgreen",
			percentage: 70.0,
			expected:   "https://img.shields.io/badge/coverage-70.0%25-yellowgreen",
		},
		{
			name:       "mediocre coverage",
			percentage: 65.2,
			expected:   "https://img.shields.io/badge/coverage-65.2%25-yellow",
		},
		{
			name:       "minimum yellow",
			percentage: 60.0,
			expected:   "https://img.shields.io/badge/coverage-60.0%25-yellow",
		},
		{
			name:       "poor coverage",
			percentage: 55.8,
			expected:   "https://img.shields.io/badge/coverage-55.8%25-orange",
		},
		{
			name:       "minimum orange",
			percentage: 50.0,
			expected:   "https://img.shields.io/badge/coverage-50.0%25-orange",
		},
		{
			name:       "very poor coverage",
			percentage: 45.1,
			expected:   "https://img.shields.io/badge/coverage-45.1%25-red",
		},
		{
			name:       "zero coverage",
			percentage: 0.0,
			expected:   "https://img.shields.io/badge/coverage-0.0%25-red",
		},
		{
			name:       "negative coverage",
			percentage: -5.0,
			expected:   "https://img.shields.io/badge/coverage--5.0%25-red",
		},
		{
			name:       "boundary 89.9",
			percentage: 89.9,
			expected:   "https://img.shields.io/badge/coverage-89.9%25-green",
		},
		{
			name:       "boundary 79.9",
			percentage: 79.9,
			expected:   "https://img.shields.io/badge/coverage-79.9%25-yellowgreen",
		},
		{
			name:       "boundary 69.9",
			percentage: 69.9,
			expected:   "https://img.shields.io/badge/coverage-69.9%25-yellow",
		},
		{
			name:       "boundary 59.9",
			percentage: 59.9,
			expected:   "https://img.shields.io/badge/coverage-59.9%25-orange",
		},
		{
			name:       "boundary 49.9",
			percentage: 49.9,
			expected:   "https://img.shields.io/badge/coverage-49.9%25-red",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildCoverageBadgeURL(tt.percentage)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildGitHubRepoURL(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		repo     string
		expected string
	}{
		{
			name:     "valid inputs",
			owner:    "mrz1836",
			repo:     "go-broadcast",
			expected: "https://github.com/mrz1836/go-broadcast",
		},
		{
			name:     "empty owner",
			owner:    "",
			repo:     "go-broadcast",
			expected: "",
		},
		{
			name:     "empty repo",
			owner:    "mrz1836",
			repo:     "",
			expected: "",
		},
		{
			name:     "both empty",
			owner:    "",
			repo:     "",
			expected: "",
		},
		{
			name:     "special characters",
			owner:    "user-name",
			repo:     "repo.name",
			expected: "https://github.com/user-name/repo.name",
		},
		{
			name:     "single character names",
			owner:    "a",
			repo:     "b",
			expected: "https://github.com/a/b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildGitHubRepoURL(tt.owner, tt.repo)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractRepoNameFromURL(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		expected string
	}{
		{
			name:     "standard owner/repo format",
			fullName: "mrz1836/go-broadcast",
			expected: "go-broadcast",
		},
		{
			name:     "multiple slashes",
			fullName: "github.com/mrz1836/go-broadcast",
			expected: "go-broadcast",
		},
		{
			name:     "trailing slash",
			fullName: "mrz1836/go-broadcast/",
			expected: "",
		},
		{
			name:     "just repo name",
			fullName: "go-broadcast",
			expected: "go-broadcast",
		},
		{
			name:     "empty string",
			fullName: "",
			expected: "",
		},
		{
			name:     "single slash at end",
			fullName: "mrz1836/",
			expected: "",
		},
		{
			name:     "multiple segments",
			fullName: "github.com/org/subgroup/repo-name",
			expected: "repo-name",
		},
		{
			name:     "complex path with dots",
			fullName: "github.com/user.name/repo.name",
			expected: "repo.name",
		},
		{
			name:     "path with hyphens and underscores",
			fullName: "my-org/my_repo-name",
			expected: "my_repo-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractRepoNameFromURL(tt.fullName)
			require.Equal(t, tt.expected, result)
		})
	}
}
