package state

import (
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPRMetadata(t *testing.T) {
	tests := []struct {
		name        string
		pr          gh.PR
		expectError bool
		expected    *PRMetadata
	}{
		{
			name: "valid metadata",
			pr: gh.PR{
				Body: `This PR syncs files from the template repository.

## Sync Details

- **Source**: org/template @ main
- **Commit**: abc123def
- **Created**: 2024-01-15T12:00:00Z

<!-- go-broadcast:metadata
source_commit: abc123def
source_repo: org/template
source_branch: main
created_at: 2024-01-15T12:00:00Z
files:
  - .github/workflows/ci.yml
  - Makefile
transforms_applied:
  - repository-name-replacer
  - template-variable-replacer
-->`,
			},
			expectError: false,
			expected: &PRMetadata{
				SourceCommit: "abc123def",
				SourceRepo:   "org/template",
				SourceBranch: "main",
				CreatedAt:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				Files: []string{
					".github/workflows/ci.yml",
					"Makefile",
				},
				TransformsApplied: []string{
					"repository-name-replacer",
					"template-variable-replacer",
				},
			},
		},
		{
			name: "minimal metadata",
			pr: gh.PR{
				Body: `Sync PR

<!-- go-broadcast:metadata
source_commit: xyz789
source_repo: org/template
source_branch: main
created_at: 2024-01-15T12:00:00Z
files: []
-->`,
			},
			expectError: false,
			expected: &PRMetadata{
				SourceCommit: "xyz789",
				SourceRepo:   "org/template",
				SourceBranch: "main",
				CreatedAt:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				Files:        []string{},
			},
		},
		{
			name: "no metadata block",
			pr: gh.PR{
				Body: "This is a regular PR without metadata",
			},
			expectError: true,
		},
		{
			name: "empty body",
			pr: gh.PR{
				Body: "",
			},
			expectError: true,
		},
		{
			name: "malformed metadata - not closed",
			pr: gh.PR{
				Body: `PR description

<!-- go-broadcast:metadata
source_commit: abc123
source_repo: org/template`,
			},
			expectError: true,
		},
		{
			name: "malformed metadata - invalid YAML",
			pr: gh.PR{
				Body: `PR description

<!-- go-broadcast:metadata
source_commit: abc123
source_repo: [unclosed bracket
source_branch main
-->`,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractPRMetadata(tt.pr)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.SourceCommit, result.SourceCommit)
			assert.Equal(t, tt.expected.SourceRepo, result.SourceRepo)
			assert.Equal(t, tt.expected.SourceBranch, result.SourceBranch)
			assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
			assert.Equal(t, tt.expected.Files, result.Files)
			assert.Equal(t, tt.expected.TransformsApplied, result.TransformsApplied)
		})
	}
}

func TestFormatPRMetadata(t *testing.T) {
	metadata := &PRMetadata{
		SourceCommit: "abc123def",
		SourceRepo:   "org/template",
		SourceBranch: "main",
		CreatedAt:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Files: []string{
			".github/workflows/ci.yml",
			"Makefile",
		},
		TransformsApplied: []string{
			"repository-name-replacer",
		},
	}

	result := FormatPRMetadata(metadata)

	// Check that it starts and ends correctly
	assert.True(t, strings.HasPrefix(result, "<!-- go-broadcast:metadata\n"))
	assert.True(t, strings.HasSuffix(result, "-->"))

	// Check that key fields are present
	assert.Contains(t, result, "source_commit: abc123def")
	assert.Contains(t, result, "source_repo: org/template")
	assert.Contains(t, result, "source_branch: main")
	assert.Contains(t, result, ".github/workflows/ci.yml")
	assert.Contains(t, result, "repository-name-replacer")
}

func TestGeneratePRDescription(t *testing.T) {
	metadata := &PRMetadata{
		SourceCommit: "abc123def",
		SourceRepo:   "org/template",
		SourceBranch: "main",
		CreatedAt:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Files: []string{
			".github/workflows/ci.yml",
			"Makefile",
		},
		TransformsApplied: []string{
			"repository-name-replacer",
			"template-variable-replacer",
		},
	}

	summary := "Sync latest changes from source repository"
	result := GeneratePRDescription(metadata, summary)

	// Check structure
	assert.Contains(t, result, summary)
	assert.Contains(t, result, "## Sync Details")
	assert.Contains(t, result, "### Files Synced")
	assert.Contains(t, result, "### Transforms Applied")

	// Check content
	assert.Contains(t, result, "**Source**: `org/template` @ `main`")
	assert.Contains(t, result, "**Commit**: `abc123def`")
	assert.Contains(t, result, "- `.github/workflows/ci.yml`")
	assert.Contains(t, result, "- `Makefile`")
	assert.Contains(t, result, "- repository-name-replacer")
	assert.Contains(t, result, "- template-variable-replacer")

	// Check metadata block is included
	assert.Contains(t, result, "<!-- go-broadcast:metadata")
	assert.Contains(t, result, "-->")
}

func TestPRMetadataRoundTrip(t *testing.T) {
	// Test that we can generate a PR description and extract the metadata back
	original := &PRMetadata{
		SourceCommit: "deadbeef123",
		SourceRepo:   "org/template-repo",
		SourceBranch: "develop",
		CreatedAt:    time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC),
		Files: []string{
			"src/config.go",
			"tests/integration_test.go",
		},
		TransformsApplied: []string{
			"custom-transform",
		},
	}

	// Generate PR description
	description := GeneratePRDescription(original, "Test sync")

	// Create a PR with this description
	pr := gh.PR{
		Body: description,
	}

	// Extract metadata back
	extracted, err := ExtractPRMetadata(pr)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.SourceCommit, extracted.SourceCommit)
	assert.Equal(t, original.SourceRepo, extracted.SourceRepo)
	assert.Equal(t, original.SourceBranch, extracted.SourceBranch)
	assert.Equal(t, original.CreatedAt, extracted.CreatedAt)
	assert.Equal(t, original.Files, extracted.Files)
	assert.Equal(t, original.TransformsApplied, extracted.TransformsApplied)
}
