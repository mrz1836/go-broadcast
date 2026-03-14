package state

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

func TestExtractEnhancedPRMetadata(t *testing.T) {
	tests := []struct {
		name        string
		pr          gh.PR
		expectError bool
		expected    *EnhancedPRMetadata
	}{
		{
			name: "enhanced metadata with directories",
			pr: gh.PR{
				Body: `This PR syncs files and directories from the template repository.

## Sync Details

- **Source**: company/template-repo @ abc123f7890
- **Target**: company/service
- **Sync Time**: 2025-01-30T14:30:52Z

<!-- go-broadcast-metadata
sync_metadata:
  source_repo: company/template-repo
  source_commit: abc123f7890
  target_repo: company/service
  sync_commit: def456
  sync_time: 2025-01-30T14:30:52Z
files:
  - src: .github/workflows/ci.yml
    dest: .github/workflows/ci.yml
directories:
  - src: .github/actions
    dest: .github/actions
    excluded: ["*.out", "*.test", "go-coverage"]
    files_synced: 61
    files_excluded: 26
    processing_time_ms: 1523
performance:
  total_files: 87
  api_calls_saved: 72
  transforms_applied:
    repository_name_replacer: true
  timing:
    total_time: "2.5s"
-->`,
			},
			expectError: false,
			expected: &EnhancedPRMetadata{
				SyncMetadata: &SyncMetadataInfo{
					SourceRepo:   "company/template-repo",
					SourceCommit: "abc123f7890",
					TargetRepo:   "company/service",
					SyncCommit:   "def456",
					SyncTime:     time.Date(2025, 1, 30, 14, 30, 52, 0, time.UTC),
				},
				Files: []FileMapping{
					{
						Source:      ".github/workflows/ci.yml",
						Destination: ".github/workflows/ci.yml",
					},
				},
				Directories: []DirectoryMapping{
					{
						Source:         ".github/actions",
						Destination:    ".github/actions",
						Excluded:       []string{"*.out", "*.test", "go-coverage"},
						FilesSynced:    61,
						FilesExcluded:  26,
						ProcessingTime: 1523,
					},
				},
				Performance: &PerformanceInfo{
					TotalFiles:    87,
					APICallsSaved: 72,
					TransformsApplied: map[string]bool{
						"repository_name_replacer": true,
					},
					TimingInfo: map[string]string{
						"total_time": "2.5s",
					},
				},
			},
		},
		{
			name: "minimal enhanced metadata",
			pr: gh.PR{
				Body: `Minimal sync

<!-- go-broadcast-metadata
sync_metadata:
  source_repo: org/template
  source_commit: xyz789
  target_repo: org/service
  sync_time: 2025-01-30T12:00:00Z
-->`,
			},
			expectError: false,
			expected: &EnhancedPRMetadata{
				SyncMetadata: &SyncMetadataInfo{
					SourceRepo:   "org/template",
					SourceCommit: "xyz789",
					TargetRepo:   "org/service",
					SyncTime:     time.Date(2025, 1, 30, 12, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "enhanced metadata with alternative marker",
			pr: gh.PR{
				Body: `Sync with alternative marker

<!-- go-broadcast metadata
sync_metadata:
  source_repo: org/template
  source_commit: abc123
  target_repo: org/service
  sync_time: 2025-01-30T12:00:00Z
files:
  - src: README.md
    dest: README.md
-->`,
			},
			expectError: false,
			expected: &EnhancedPRMetadata{
				SyncMetadata: &SyncMetadataInfo{
					SourceRepo:   "org/template",
					SourceCommit: "abc123",
					TargetRepo:   "org/service",
					SyncTime:     time.Date(2025, 1, 30, 12, 0, 0, 0, time.UTC),
				},
				Files: []FileMapping{
					{
						Source:      "README.md",
						Destination: "README.md",
					},
				},
			},
		},
		{
			name: "malformed enhanced metadata - missing sync_metadata",
			pr: gh.PR{
				Body: `Invalid enhanced metadata

<!-- go-broadcast-metadata
files:
  - src: test.txt
    dest: test.txt
-->`,
			},
			expectError: true,
		},
		{
			name: "malformed enhanced metadata - invalid YAML",
			pr: gh.PR{
				Body: `Invalid YAML

<!-- go-broadcast-metadata
sync_metadata:
  source_repo: org/template
  source_commit: [unclosed
-->`,
			},
			expectError: true,
		},
		{
			name: "no metadata block",
			pr: gh.PR{
				Body: `This PR has no metadata block.

Just a regular PR description.`,
			},
			expectError: true,
		},
		{
			name:        "empty body",
			pr:          gh.PR{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractEnhancedPRMetadata(tt.pr)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.SyncMetadata)

			// Verify sync metadata
			assert.Equal(t, tt.expected.SyncMetadata.SourceRepo, result.SyncMetadata.SourceRepo)
			assert.Equal(t, tt.expected.SyncMetadata.SourceCommit, result.SyncMetadata.SourceCommit)
			assert.Equal(t, tt.expected.SyncMetadata.TargetRepo, result.SyncMetadata.TargetRepo)
			assert.Equal(t, tt.expected.SyncMetadata.SyncCommit, result.SyncMetadata.SyncCommit)
			assert.Equal(t, tt.expected.SyncMetadata.SyncTime, result.SyncMetadata.SyncTime)

			// Verify files
			assert.Len(t, result.Files, len(tt.expected.Files))
			for i, expectedFile := range tt.expected.Files {
				if i < len(result.Files) {
					assert.Equal(t, expectedFile.Source, result.Files[i].Source)
					assert.Equal(t, expectedFile.Destination, result.Files[i].Destination)
				}
			}

			// Verify directories
			assert.Len(t, result.Directories, len(tt.expected.Directories))
			for i, expectedDir := range tt.expected.Directories {
				if i < len(result.Directories) {
					assert.Equal(t, expectedDir.Source, result.Directories[i].Source)
					assert.Equal(t, expectedDir.Destination, result.Directories[i].Destination)
					assert.Equal(t, expectedDir.Excluded, result.Directories[i].Excluded)
					assert.Equal(t, expectedDir.FilesSynced, result.Directories[i].FilesSynced)
					assert.Equal(t, expectedDir.FilesExcluded, result.Directories[i].FilesExcluded)
					assert.Equal(t, expectedDir.ProcessingTime, result.Directories[i].ProcessingTime)
				}
			}

			// Verify performance metrics
			if tt.expected.Performance != nil {
				require.NotNil(t, result.Performance)
				assert.Equal(t, tt.expected.Performance.TotalFiles, result.Performance.TotalFiles)
				assert.Equal(t, tt.expected.Performance.APICallsSaved, result.Performance.APICallsSaved)
				assert.Equal(t, tt.expected.Performance.TransformsApplied, result.Performance.TransformsApplied)
				assert.Equal(t, tt.expected.Performance.TimingInfo, result.Performance.TimingInfo)
			} else {
				assert.Nil(t, result.Performance)
			}
		})
	}
}

func TestFormatEnhancedPRMetadata(t *testing.T) {
	metadata := &EnhancedPRMetadata{
		SyncMetadata: &SyncMetadataInfo{
			SourceRepo:   "company/template-repo",
			SourceCommit: "abc123f7890",
			TargetRepo:   "company/service",
			SyncCommit:   "def456",
			SyncTime:     time.Date(2025, 1, 30, 14, 30, 52, 0, time.UTC),
		},
		Files: []FileMapping{
			{
				Source:      ".github/workflows/ci.yml",
				Destination: ".github/workflows/ci.yml",
			},
		},
		Directories: []DirectoryMapping{
			{
				Source:         ".github/actions",
				Destination:    ".github/actions",
				Excluded:       []string{"*.out", "*.test"},
				FilesSynced:    61,
				FilesExcluded:  26,
				ProcessingTime: 1523,
			},
		},
		Performance: &PerformanceInfo{
			TotalFiles:    87,
			APICallsSaved: 72,
			TransformsApplied: map[string]bool{
				"repository_name_replacer": true,
			},
			TimingInfo: map[string]string{
				"total_time": "2.5s",
			},
		},
	}

	result := FormatEnhancedPRMetadata(metadata)

	// Check that it starts and ends correctly
	assert.True(t, strings.HasPrefix(result, "<!-- go-broadcast-metadata\n"))
	assert.True(t, strings.HasSuffix(result, "-->"))

	// Check that key fields are present
	assert.Contains(t, result, "source_repo: company/template-repo")
	assert.Contains(t, result, "source_commit: abc123f7890")
	assert.Contains(t, result, "target_repo: company/service")
	assert.Contains(t, result, "sync_commit: def456")
	assert.Contains(t, result, ".github/workflows/ci.yml")
	assert.Contains(t, result, ".github/actions")
	assert.Contains(t, result, "files_synced: 61")
	assert.Contains(t, result, "total_files: 87")
	assert.Contains(t, result, "api_calls_saved: 72")
	assert.Contains(t, result, "repository_name_replacer: true")
}

func TestGenerateEnhancedPRDescription(t *testing.T) {
	metadata := &EnhancedPRMetadata{
		SyncMetadata: &SyncMetadataInfo{
			SourceRepo:   "company/template-repo",
			SourceCommit: "abc123f7890",
			TargetRepo:   "company/service",
			SyncCommit:   "def456",
			SyncTime:     time.Date(2025, 1, 30, 14, 30, 52, 0, time.UTC),
		},
		Files: []FileMapping{
			{
				Source:      ".github/workflows/ci.yml",
				Destination: ".github/workflows/ci.yml",
			},
			{
				Source:      "Makefile",
				Destination: "Makefile",
			},
		},
		Directories: []DirectoryMapping{
			{
				Source:         ".github/actions",
				Destination:    ".github/actions",
				Excluded:       []string{"*.out", "*.test"},
				FilesSynced:    61,
				FilesExcluded:  26,
				ProcessingTime: 1523,
			},
		},
		Performance: &PerformanceInfo{
			TotalFiles:    87,
			APICallsSaved: 72,
			TransformsApplied: map[string]bool{
				"repository_name_replacer": true,
			},
		},
	}

	summary := "Sync latest changes with directory support"
	result := GenerateEnhancedPRDescription(metadata, summary)

	// Check structure
	assert.Contains(t, result, summary)
	assert.Contains(t, result, "## What Changed")

	// Check sync details
	assert.Contains(t, result, "* Sync from company/template-repo (commit: abc123f7890)")
	assert.Contains(t, result, "* Total files synchronized: 87")
	assert.Contains(t, result, "* Directories synchronized: 1")
	assert.Contains(t, result, "* Files excluded: 26")
	assert.Contains(t, result, "* Transforms applied: repository_name_replacer")
	assert.Contains(t, result, "* API calls optimized: 72 calls saved")

	// Check file list (should show since we have 2 files, which is <= 10)
	assert.Contains(t, result, "## Files Synchronized")
	assert.Contains(t, result, "* `.github/workflows/ci.yml` → `.github/workflows/ci.yml`")
	assert.Contains(t, result, "* `Makefile` → `Makefile`")

	// Check directory mappings (should show since we have 1 directory, which is <= 5)
	assert.Contains(t, result, "## Directories Synchronized")
	assert.Contains(t, result, "* `.github/actions` → `.github/actions` (61 files, 26 excluded)")
	assert.Contains(t, result, "  - Excluded: *.out, *.test")

	// Check metadata block is included
	assert.Contains(t, result, "<!-- go-broadcast-metadata")
	assert.Contains(t, result, "-->")
}

func TestGenerateEnhancedPRDescriptionManyFiles(t *testing.T) {
	// Test with more than 10 files - should not list them individually
	files := make([]FileMapping, 15)
	for i := range files {
		files[i] = FileMapping{
			Source:      "file.txt",
			Destination: "file.txt",
		}
	}

	metadata := &EnhancedPRMetadata{
		SyncMetadata: &SyncMetadataInfo{
			SourceRepo:   "org/template",
			SourceCommit: "abc123",
			TargetRepo:   "org/service",
			SyncTime:     time.Now(),
		},
		Files: files,
	}

	result := GenerateEnhancedPRDescription(metadata, "Many files sync")

	// Should not include the file list section when > 10 files
	assert.NotContains(t, result, "## Files Synchronized")
	assert.Contains(t, result, "* Total files synchronized: 15")
}

func TestGenerateEnhancedPRDescriptionManyDirectories(t *testing.T) {
	// Test with more than 5 directories - should not list them individually
	dirs := make([]DirectoryMapping, 10)
	for i := range dirs {
		dirs[i] = DirectoryMapping{
			Source:      "src/dir",
			Destination: "dest/dir",
			FilesSynced: 5,
		}
	}

	metadata := &EnhancedPRMetadata{
		SyncMetadata: &SyncMetadataInfo{
			SourceRepo:   "org/template",
			SourceCommit: "abc123",
			TargetRepo:   "org/service",
			SyncTime:     time.Now(),
		},
		Directories: dirs,
		Performance: &PerformanceInfo{
			TotalFiles: 50,
		},
	}

	result := GenerateEnhancedPRDescription(metadata, "Many directories sync")

	// Should show summary message instead of listing directories
	assert.Contains(t, result, "## Directories Synchronized")
	assert.Contains(t, result, "10 directories synchronized (too many to list)")
	assert.Contains(t, result, "* Total files synchronized: 50")
}

// TestExtractMetadataYAML_EdgeCases tests edge cases in metadata extraction
func TestExtractMetadataYAML_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		marker      string
		expectError error
		expectEmpty bool
	}{
		{
			name:        "marker only - no content after",
			body:        "<!-- go-broadcast-metadata-->",
			marker:      "<!-- go-broadcast-metadata",
			expectError: ErrPRNoMetadataBlock,
		},
		{
			name:        "marker with only newline before close",
			body:        "<!-- go-broadcast-metadata\n-->",
			marker:      "<!-- go-broadcast-metadata",
			expectError: ErrPRNoMetadataBlock,
		},
		{
			name:        "marker with whitespace-only content",
			body:        "<!-- go-broadcast-metadata\n   \n   \n-->",
			marker:      "<!-- go-broadcast-metadata",
			expectError: ErrPRNoMetadataBlock,
		},
		{
			name:        "marker with tabs and spaces only",
			body:        "<!-- go-broadcast-metadata\n\t  \t\n-->",
			marker:      "<!-- go-broadcast-metadata",
			expectError: ErrPRNoMetadataBlock,
		},
		{
			name:        "unclosed metadata block",
			body:        "<!-- go-broadcast-metadata\nsome: yaml\nmore: content",
			marker:      "<!-- go-broadcast-metadata",
			expectError: ErrPRMetadataNotClosed,
		},
		{
			name:        "marker not found",
			body:        "This PR has no metadata\nJust regular text",
			marker:      "<!-- go-broadcast-metadata",
			expectError: ErrPRNoMetadataBlock,
		},
		{
			name:        "valid content - should not error",
			body:        "<!-- go-broadcast-metadata\nkey: value\n-->",
			marker:      "<!-- go-broadcast-metadata",
			expectError: nil,
		},
		{
			name:        "valid content with surrounding text",
			body:        "Prefix text\n<!-- go-broadcast-metadata\nkey: value\n-->\nSuffix text",
			marker:      "<!-- go-broadcast-metadata",
			expectError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractMetadataYAML(tt.body, tt.marker)

			if tt.expectError != nil {
				require.ErrorIs(t, err, tt.expectError)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestEnhancedPRMetadataRoundTrip(t *testing.T) {
	// Test that we can generate an enhanced PR description and extract the metadata back
	original := &EnhancedPRMetadata{
		SyncMetadata: &SyncMetadataInfo{
			SourceRepo:   "company/template-repo",
			SourceCommit: "deadbeef123",
			TargetRepo:   "company/service-a",
			SyncCommit:   "cafe1234",
			SyncTime:     time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC),
		},
		Files: []FileMapping{
			{
				Source:      "src/config.go",
				Destination: "src/config.go",
			},
		},
		Directories: []DirectoryMapping{
			{
				Source:         "tests/integration",
				Destination:    "tests/integration",
				Excluded:       []string{"*.tmp"},
				FilesSynced:    42,
				FilesExcluded:  3,
				ProcessingTime: 800,
			},
		},
		Performance: &PerformanceInfo{
			TotalFiles:    45,
			APICallsSaved: 25,
			TransformsApplied: map[string]bool{
				"test_replacer": true,
			},
			TimingInfo: map[string]string{
				"total": "1.5s",
			},
		},
	}

	// Generate PR description
	description := GenerateEnhancedPRDescription(original, "Test enhanced sync")

	// Create a PR with this description
	pr := gh.PR{
		Body: description,
	}

	// Extract metadata back
	extracted, err := ExtractEnhancedPRMetadata(pr)
	require.NoError(t, err)

	// Compare sync metadata
	require.NotNil(t, extracted.SyncMetadata)
	assert.Equal(t, original.SyncMetadata.SourceRepo, extracted.SyncMetadata.SourceRepo)
	assert.Equal(t, original.SyncMetadata.SourceCommit, extracted.SyncMetadata.SourceCommit)
	assert.Equal(t, original.SyncMetadata.TargetRepo, extracted.SyncMetadata.TargetRepo)
	assert.Equal(t, original.SyncMetadata.SyncCommit, extracted.SyncMetadata.SyncCommit)
	assert.Equal(t, original.SyncMetadata.SyncTime, extracted.SyncMetadata.SyncTime)

	// Compare files
	require.Len(t, extracted.Files, len(original.Files))
	for i, originalFile := range original.Files {
		assert.Equal(t, originalFile.Source, extracted.Files[i].Source)
		assert.Equal(t, originalFile.Destination, extracted.Files[i].Destination)
	}

	// Compare directories
	require.Len(t, extracted.Directories, len(original.Directories))
	for i, originalDir := range original.Directories {
		assert.Equal(t, originalDir.Source, extracted.Directories[i].Source)
		assert.Equal(t, originalDir.Destination, extracted.Directories[i].Destination)
		assert.Equal(t, originalDir.Excluded, extracted.Directories[i].Excluded)
		assert.Equal(t, originalDir.FilesSynced, extracted.Directories[i].FilesSynced)
		assert.Equal(t, originalDir.FilesExcluded, extracted.Directories[i].FilesExcluded)
		assert.Equal(t, originalDir.ProcessingTime, extracted.Directories[i].ProcessingTime)
	}

	// Compare performance
	require.NotNil(t, extracted.Performance)
	assert.Equal(t, original.Performance.TotalFiles, extracted.Performance.TotalFiles)
	assert.Equal(t, original.Performance.APICallsSaved, extracted.Performance.APICallsSaved)
	assert.Equal(t, original.Performance.TransformsApplied, extracted.Performance.TransformsApplied)
	assert.Equal(t, original.Performance.TimingInfo, extracted.Performance.TimingInfo)
}
