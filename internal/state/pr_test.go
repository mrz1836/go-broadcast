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
    from: file
directories:
  - src: .github/coverage
    dest: .github/coverage
    excluded: ["*.out", "*.test", "gofortress-coverage"]
    files_synced: 61
    files_excluded: 26
    processing_time_ms: 1523
performance:
  total_sync_time_ms: 2500
  total_files_processed: 87
  total_files_changed: 63
  total_files_skipped: 24
  api_calls_saved: 72
  cache_hits: 45
  cache_misses: 15
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
						From:        "file",
					},
				},
				Directories: []DirectoryMapping{
					{
						Source:           ".github/coverage",
						Destination:      ".github/coverage",
						Excluded:         []string{"*.out", "*.test", "gofortress-coverage"},
						FilesSynced:      61,
						FilesExcluded:    26,
						ProcessingTimeMs: 1523,
					},
				},
				Performance: &PerformanceInfo{
					TotalSyncTimeMs:     2500,
					TotalFilesProcessed: 87,
					TotalFilesChanged:   63,
					TotalFilesSkipped:   24,
					APICallsSaved:       72,
					CacheHits:           45,
					CacheMisses:         15,
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

<!-- go-broadcast:metadata
sync_metadata:
  source_repo: org/template
  source_commit: abc123
  target_repo: org/service
  sync_time: 2025-01-30T12:00:00Z
files:
  - src: README.md
    dest: README.md
    from: file
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
						From:        "file",
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
    from: file
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
					assert.Equal(t, expectedFile.From, result.Files[i].From)
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
					assert.Equal(t, expectedDir.ProcessingTimeMs, result.Directories[i].ProcessingTimeMs)
				}
			}

			// Verify performance metrics
			if tt.expected.Performance != nil {
				require.NotNil(t, result.Performance)
				assert.Equal(t, tt.expected.Performance.TotalSyncTimeMs, result.Performance.TotalSyncTimeMs)
				assert.Equal(t, tt.expected.Performance.TotalFilesProcessed, result.Performance.TotalFilesProcessed)
				assert.Equal(t, tt.expected.Performance.TotalFilesChanged, result.Performance.TotalFilesChanged)
				assert.Equal(t, tt.expected.Performance.TotalFilesSkipped, result.Performance.TotalFilesSkipped)
				assert.Equal(t, tt.expected.Performance.APICallsSaved, result.Performance.APICallsSaved)
				assert.Equal(t, tt.expected.Performance.CacheHits, result.Performance.CacheHits)
				assert.Equal(t, tt.expected.Performance.CacheMisses, result.Performance.CacheMisses)
			} else {
				assert.Nil(t, result.Performance)
			}
		})
	}
}

func TestExtractPRMetadataBackwardCompatibility(t *testing.T) {
	// Test that legacy extraction still works when enhanced format is present
	enhancedPR := gh.PR{
		Body: `Enhanced format sync

<!-- go-broadcast-metadata
sync_metadata:
  source_repo: company/template-repo
  source_commit: abc123f7890
  target_repo: company/service
  sync_time: 2025-01-30T14:30:52Z
files:
  - src: .github/workflows/ci.yml
    dest: .github/workflows/ci.yml
    from: file
directories:
  - src: .github/coverage
    dest: .github/coverage
    files_synced: 61
    files_excluded: 26
-->`,
	}

	// Extract using legacy function - should convert enhanced to legacy
	legacy, err := ExtractPRMetadata(enhancedPR)
	require.NoError(t, err)
	require.NotNil(t, legacy)

	// Verify legacy format fields are populated
	assert.Equal(t, "abc123f7890", legacy.SourceCommit)
	assert.Equal(t, "company/template-repo", legacy.SourceRepo)
	assert.Equal(t, time.Date(2025, 1, 30, 14, 30, 52, 0, time.UTC), legacy.CreatedAt)
	assert.Empty(t, legacy.SourceBranch)      // Not available in enhanced format
	assert.Empty(t, legacy.TransformsApplied) // Not tracked in enhanced format

	// Files should include both individual files and directory placeholders
	assert.Contains(t, legacy.Files, ".github/workflows/ci.yml")
	assert.Contains(t, legacy.Files, ".github/coverage/*")
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
				From:        "file",
			},
		},
		Directories: []DirectoryMapping{
			{
				Source:           ".github/coverage",
				Destination:      ".github/coverage",
				Excluded:         []string{"*.out", "*.test"},
				FilesSynced:      61,
				FilesExcluded:    26,
				ProcessingTimeMs: 1523,
			},
		},
		Performance: &PerformanceInfo{
			TotalSyncTimeMs:     2500,
			TotalFilesProcessed: 87,
			TotalFilesChanged:   63,
			APICallsSaved:       72,
			CacheHits:           45,
			CacheMisses:         15,
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
	assert.Contains(t, result, ".github/coverage")
	assert.Contains(t, result, "files_synced: 61")
	assert.Contains(t, result, "total_sync_time_ms: 2500")
	assert.Contains(t, result, "api_calls_saved: 72")
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
				From:        "file",
			},
		},
		Directories: []DirectoryMapping{
			{
				Source:           ".github/coverage",
				Destination:      ".github/coverage",
				Excluded:         []string{"*.out", "*.test"},
				FilesSynced:      61,
				FilesExcluded:    26,
				ProcessingTimeMs: 1523,
			},
		},
		Performance: &PerformanceInfo{
			TotalSyncTimeMs:     2500,
			TotalFilesProcessed: 87,
			TotalFilesChanged:   63,
			TotalFilesSkipped:   24,
			APICallsSaved:       72,
			CacheHits:           45,
			CacheMisses:         15,
		},
	}

	summary := "Sync latest changes with directory support"
	result := GenerateEnhancedPRDescription(metadata, summary)

	// Check structure
	assert.Contains(t, result, summary)
	assert.Contains(t, result, "## Sync Details")
	assert.Contains(t, result, "### Individual Files Synced")
	assert.Contains(t, result, "### Directory Mappings")
	assert.Contains(t, result, "### Performance Metrics")

	// Check sync details
	assert.Contains(t, result, "**Source**: `company/template-repo` @ `abc123f7890`")
	assert.Contains(t, result, "**Target**: `company/service`")
	assert.Contains(t, result, "**Sync Commit**: `def456`")

	// Check individual files
	assert.Contains(t, result, "- `.github/workflows/ci.yml`")

	// Check directory mappings
	assert.Contains(t, result, "- `.github/coverage`")
	assert.Contains(t, result, "**Files Synced**: 61")
	assert.Contains(t, result, "**Files Excluded**: 26")
	assert.Contains(t, result, "**Processing Time**: 1523ms")
	assert.Contains(t, result, "**Exclusion Patterns**: *.out, *.test")

	// Check performance metrics
	assert.Contains(t, result, "**Total Sync Time**: 2500ms")
	assert.Contains(t, result, "**Files Processed**: 87")
	assert.Contains(t, result, "**Files Changed**: 63")
	assert.Contains(t, result, "**Files Skipped**: 24")
	assert.Contains(t, result, "**API Calls Saved**: 72")
	assert.Contains(t, result, "**Cache Hit Rate**: 75.0% (45 hits, 15 misses)")

	// Check metadata block is included
	assert.Contains(t, result, "<!-- go-broadcast-metadata")
	assert.Contains(t, result, "-->")
}

func TestConvertToDirectorySyncInfo(t *testing.T) {
	enhanced := &EnhancedPRMetadata{
		SyncMetadata: &SyncMetadataInfo{
			SourceRepo:   "company/template-repo",
			SourceCommit: "abc123f7890",
			TargetRepo:   "company/service",
			SyncTime:     time.Date(2025, 1, 30, 14, 30, 52, 0, time.UTC),
		},
		Files: []FileMapping{
			{
				Source:      ".github/workflows/ci.yml",
				Destination: ".github/workflows/ci.yml",
				From:        "file",
			},
		},
		Directories: []DirectoryMapping{
			{
				Source:           ".github/coverage",
				Destination:      ".github/coverage",
				Excluded:         []string{"*.out", "*.test"},
				FilesSynced:      61,
				FilesExcluded:    26,
				ProcessingTimeMs: 1523,
			},
		},
		Performance: &PerformanceInfo{
			TotalSyncTimeMs:     2500,
			TotalFilesProcessed: 87,
			TotalFilesChanged:   63,
			TotalFilesSkipped:   24,
			APICallsSaved:       72,
			CacheHits:           45,
			CacheMisses:         15,
		},
	}

	result := ConvertToDirectorySyncInfo(enhanced)
	require.NotNil(t, result)

	// Check directory mappings
	require.Len(t, result.DirectoryMappings, 1)
	mapping := result.DirectoryMappings[0]
	assert.Equal(t, ".github/coverage", mapping.Source)
	assert.Equal(t, ".github/coverage", mapping.Destination)
	assert.Equal(t, []string{"*.out", "*.test"}, mapping.ExcludePatterns)
	assert.True(t, mapping.PreserveStructure)
	assert.False(t, mapping.IncludeHidden)
	assert.False(t, mapping.TransformApplied)

	// Check synced files info
	require.NotNil(t, result.SyncedFiles)
	assert.Equal(t, 1, result.SyncedFiles.IndividualFileCount)
	assert.Equal(t, 61, result.SyncedFiles.DirectoryFileCount)
	assert.Equal(t, 62, result.SyncedFiles.TotalFiles)
	assert.Contains(t, result.SyncedFiles.IndividualSyncedFiles, ".github/workflows/ci.yml")

	// Check performance metrics
	require.NotNil(t, result.PerformanceMetrics)
	assert.True(t, result.PerformanceMetrics.ExtractedFromPR)

	// Check overall metrics
	require.NotNil(t, result.PerformanceMetrics.OverallMetrics)
	overall := result.PerformanceMetrics.OverallMetrics
	assert.Equal(t, int64(2500), overall.ProcessingTimeMs)
	assert.Equal(t, 87, overall.TotalFilesProcessed)
	assert.Equal(t, 63, overall.TotalFilesChanged)
	assert.Equal(t, 24, overall.TotalFilesSkipped)
	assert.Equal(t, time.Duration(2500)*time.Millisecond, overall.Duration)

	// Check API metrics
	require.NotNil(t, result.PerformanceMetrics.APIMetrics)
	api := result.PerformanceMetrics.APIMetrics
	assert.Equal(t, 72, api.APICallsSaved)
	assert.Equal(t, 45, api.CacheHits)
	assert.Equal(t, 15, api.CacheMisses)
	assert.InDelta(t, 0.75, api.CacheHitRatio, 0.01)

	// Check directory-specific metrics
	require.NotNil(t, result.PerformanceMetrics.DirectoryMetrics)
	require.Contains(t, result.PerformanceMetrics.DirectoryMetrics, ".github/coverage")
	dirMetrics := result.PerformanceMetrics.DirectoryMetrics[".github/coverage"]
	assert.Equal(t, 61, dirMetrics.FilesProcessed)
	assert.Equal(t, 26, dirMetrics.FilesExcluded)
	assert.Equal(t, time.Duration(1523)*time.Millisecond, dirMetrics.ProcessingDuration)

	// Check last directory sync time
	require.NotNil(t, result.LastDirectorySync)
	assert.Equal(t, time.Date(2025, 1, 30, 14, 30, 52, 0, time.UTC), *result.LastDirectorySync)
}

func TestCreateEnhancedMetadataFromDirectorySync(t *testing.T) {
	// Create a sample DirectorySyncInfo
	syncInfo := &DirectorySyncInfo{
		DirectoryMappings: []DirectoryMappingInfo{
			{
				Source:            ".github/workflows",
				Destination:       ".github/workflows",
				ExcludePatterns:   []string{"*.bak"},
				PreserveStructure: true,
				IncludeHidden:     false,
				TransformApplied:  true,
			},
		},
		SyncedFiles: &SyncedFilesInfo{
			IndividualSyncedFiles: []string{"README.md", "LICENSE"},
			IndividualFileCount:   2,
			DirectoryFileCount:    15,
			TotalFiles:            17,
		},
		PerformanceMetrics: &DirectoryPerformanceMetrics{
			DirectoryMetrics: map[string]*DirectoryProcessingMetrics{
				".github/workflows": {
					FilesProcessed:     15,
					FilesExcluded:      3,
					ProcessingDuration: time.Duration(500) * time.Millisecond,
				},
			},
			OverallMetrics: &OverallSyncMetrics{
				ProcessingTimeMs:    1200,
				TotalFilesProcessed: 17,
				TotalFilesChanged:   12,
				TotalFilesSkipped:   5,
			},
			APIMetrics: &APISyncMetrics{
				APICallsSaved: 30,
				CacheHits:     20,
				CacheMisses:   10,
			},
		},
	}

	result := CreateEnhancedMetadataFromDirectorySync("org/template", "org/service", "abc123", syncInfo)
	require.NotNil(t, result)

	// Check sync metadata
	require.NotNil(t, result.SyncMetadata)
	assert.Equal(t, "org/template", result.SyncMetadata.SourceRepo)
	assert.Equal(t, "org/service", result.SyncMetadata.TargetRepo)
	assert.Equal(t, "abc123", result.SyncMetadata.SourceCommit)
	assert.WithinDuration(t, time.Now().UTC(), result.SyncMetadata.SyncTime, time.Second)

	// Check individual files
	require.Len(t, result.Files, 2)
	assert.Equal(t, "README.md", result.Files[0].Source)
	assert.Equal(t, "README.md", result.Files[0].Destination)
	assert.Equal(t, "file", result.Files[0].From)

	// Check directories
	require.Len(t, result.Directories, 1)
	dir := result.Directories[0]
	assert.Equal(t, ".github/workflows", dir.Source)
	assert.Equal(t, ".github/workflows", dir.Destination)
	assert.Equal(t, []string{"*.bak"}, dir.Excluded)
	assert.Equal(t, 15, dir.FilesSynced)
	assert.Equal(t, 3, dir.FilesExcluded)
	assert.Equal(t, int64(500), dir.ProcessingTimeMs)

	// Check performance
	require.NotNil(t, result.Performance)
	assert.Equal(t, int64(1200), result.Performance.TotalSyncTimeMs)
	assert.Equal(t, 17, result.Performance.TotalFilesProcessed)
	assert.Equal(t, 12, result.Performance.TotalFilesChanged)
	assert.Equal(t, 5, result.Performance.TotalFilesSkipped)
	assert.Equal(t, 30, result.Performance.APICallsSaved)
	assert.Equal(t, 20, result.Performance.CacheHits)
	assert.Equal(t, 10, result.Performance.CacheMisses)
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
				From:        "file",
			},
		},
		Directories: []DirectoryMapping{
			{
				Source:           "tests/integration",
				Destination:      "tests/integration",
				Excluded:         []string{"*.tmp"},
				FilesSynced:      42,
				FilesExcluded:    3,
				ProcessingTimeMs: 800,
			},
		},
		Performance: &PerformanceInfo{
			TotalSyncTimeMs:     1500,
			TotalFilesProcessed: 45,
			TotalFilesChanged:   38,
			TotalFilesSkipped:   7,
			APICallsSaved:       25,
			CacheHits:           18,
			CacheMisses:         7,
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
		assert.Equal(t, originalFile.From, extracted.Files[i].From)
	}

	// Compare directories
	require.Len(t, extracted.Directories, len(original.Directories))
	for i, originalDir := range original.Directories {
		assert.Equal(t, originalDir.Source, extracted.Directories[i].Source)
		assert.Equal(t, originalDir.Destination, extracted.Directories[i].Destination)
		assert.Equal(t, originalDir.Excluded, extracted.Directories[i].Excluded)
		assert.Equal(t, originalDir.FilesSynced, extracted.Directories[i].FilesSynced)
		assert.Equal(t, originalDir.FilesExcluded, extracted.Directories[i].FilesExcluded)
		assert.Equal(t, originalDir.ProcessingTimeMs, extracted.Directories[i].ProcessingTimeMs)
	}

	// Compare performance
	require.NotNil(t, extracted.Performance)
	assert.Equal(t, original.Performance.TotalSyncTimeMs, extracted.Performance.TotalSyncTimeMs)
	assert.Equal(t, original.Performance.TotalFilesProcessed, extracted.Performance.TotalFilesProcessed)
	assert.Equal(t, original.Performance.TotalFilesChanged, extracted.Performance.TotalFilesChanged)
	assert.Equal(t, original.Performance.TotalFilesSkipped, extracted.Performance.TotalFilesSkipped)
	assert.Equal(t, original.Performance.APICallsSaved, extracted.Performance.APICallsSaved)
	assert.Equal(t, original.Performance.CacheHits, extracted.Performance.CacheHits)
	assert.Equal(t, original.Performance.CacheMisses, extracted.Performance.CacheMisses)
}
