package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/state"
)

func TestRepositorySync_generatePRBodyWithDirectories(t *testing.T) {
	repoSync := &RepositorySync{
		sourceState: &state.SourceState{
			Repo:         "org/template",
			Branch:       "master",
			LatestCommit: "abc123",
		},
		target: config.TargetConfig{
			Repo: "org/target",
			Directories: []config.DirectoryMapping{
				{
					Src:     ".github/workflows",
					Dest:    ".github/workflows",
					Exclude: []string{"*.tmp", "*.backup"},
				},
				{
					Src:     "docs",
					Dest:    "documentation",
					Exclude: []string{"*.draft"},
				},
			},
		},
		syncMetrics: &PerformanceMetrics{
			StartTime: time.Now().Add(-30 * time.Second),
			EndTime:   time.Now(),
			FileMetrics: FileProcessingMetrics{
				FilesProcessed:   28, // 5 individual files + 23 directory files (15+8)
				FilesChanged:     3,  // Actual changes
				FilesSkipped:     25, // Total processed - changed (28-3)
				ProcessingTimeMs: 1500,
			},
			DirectoryMetrics: map[string]DirectoryMetrics{
				".github/workflows": {
					FilesProcessed:     15,
					FilesChanged:       2, // 2 files actually changed in this directory
					FilesExcluded:      3,
					StartTime:          time.Now().Add(-20 * time.Second),
					EndTime:            time.Now().Add(-10 * time.Second),
					BinaryFilesSkipped: 2,
					BinaryFilesSize:    1024,
				},
				"docs": {
					FilesProcessed: 8,
					FilesChanged:   1, // 1 file actually changed in this directory
					FilesExcluded:  1,
					StartTime:      time.Now().Add(-15 * time.Second),
					EndTime:        time.Now().Add(-5 * time.Second),
				},
			},
			APICallsSaved: 25,
			CacheHits:     12,
			CacheMisses:   3,
		},
	}

	files := []FileChange{
		{Path: "README.md", IsNew: false},
		{Path: ".github/workflows/ci.yml", IsNew: true},
		{Path: "documentation/api.md", IsNew: false},
	}

	body := repoSync.generatePRBody("commit456", files, nil)

	// Verify all sections are present
	assert.Contains(t, body, "## What Changed")
	assert.Contains(t, body, "## Directory Synchronization Details")
	assert.Contains(t, body, "## Performance Metrics")
	assert.Contains(t, body, "## Why It Was Necessary")
	assert.Contains(t, body, "## Testing Performed")
	assert.Contains(t, body, "## Impact / Risk")

	// Verify directory sync details
	assert.Contains(t, body, "### `.github/workflows` → `.github/workflows`")
	assert.Contains(t, body, "### `docs` → `documentation`")
	assert.Contains(t, body, "**Files synced**: 2")    // Now shows actual changes, not processed
	assert.Contains(t, body, "**Files examined**: 15") // Shows files that were examined
	assert.Contains(t, body, "**Files synced**: 1")    // For docs directory
	assert.Contains(t, body, "**Files examined**: 8")  // For docs directory
	assert.Contains(t, body, "**Files excluded**: 3")
	assert.Contains(t, body, "**Binary files skipped**: 2")
	assert.Contains(t, body, "**Exclusion patterns**: `*.tmp`, `*.backup`")

	// Verify performance metrics (now unified)
	assert.Contains(t, body, "**Files processed**: 28 (3 changed, 0 deleted, 25 skipped)")
	assert.NotContains(t, body, "**Directory files processed**") // Should not show separate directory section
	assert.Contains(t, body, "**API calls saved**: 25")
	assert.Contains(t, body, "**Cache hit rate**: 80.0% (12 hits, 3 misses)")

	// Verify enhanced change description
	assert.Contains(t, body, "Updated 1 individual file(s) to synchronize with the source repository")
	assert.Contains(t, body, "Synchronized 2 file(s) from directory mappings")

	// Verify metadata contains directory information
	assert.Contains(t, body, "directories:")
	assert.Contains(t, body, "- src: .github/workflows")
	assert.Contains(t, body, "  dest: .github/workflows")
	assert.Contains(t, body, "  excluded: [\"*.tmp\", \"*.backup\"]")
	assert.Contains(t, body, "  files_synced: 2")    // Now shows actual changes
	assert.Contains(t, body, "  files_examined: 15") // Shows files examined
	assert.Contains(t, body, "  files_excluded: 3")

	// Verify performance metadata
	assert.Contains(t, body, "performance:")
	assert.Contains(t, body, "total_files: 28") // 5 + 15 + 8
	assert.Contains(t, body, "api_calls_saved: 25")
	assert.Contains(t, body, "cache_hits: 12")
}

func TestRepositorySync_isDirectoryFile(t *testing.T) {
	repoSync := &RepositorySync{
		target: config.TargetConfig{
			Directories: []config.DirectoryMapping{
				{
					Src:  ".github/workflows",
					Dest: ".github/workflows",
				},
				{
					Src:  "docs",
					Dest: "documentation",
				},
			},
		},
	}

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "direct directory match",
			filePath: ".github/workflows",
			expected: true,
		},
		{
			name:     "file in directory",
			filePath: ".github/workflows/ci.yml",
			expected: true,
		},
		{
			name:     "file in mapped directory",
			filePath: "documentation/api.md",
			expected: true,
		},
		{
			name:     "file not in any directory",
			filePath: "README.md",
			expected: false,
		},
		{
			name:     "similar path but not in directory",
			filePath: ".github/workflow-config.yml",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repoSync.isDirectoryFile(tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRepositorySync_TrackingMethods(t *testing.T) {
	// Test with nil syncMetrics (should not panic)
	repoSync := &RepositorySync{}

	// These should not panic even with nil syncMetrics
	repoSync.TrackAPICallSaved(5)
	repoSync.TrackCacheHit()
	repoSync.TrackCacheMiss()
	repoSync.TrackAPIRequest()

	// Test with initialized syncMetrics
	repoSync.syncMetrics = &PerformanceMetrics{}

	repoSync.TrackAPICallSaved(10)
	repoSync.TrackCacheHit()
	repoSync.TrackCacheHit()
	repoSync.TrackCacheMiss()
	repoSync.TrackAPIRequest()
	repoSync.TrackAPIRequest()
	repoSync.TrackAPIRequest()

	assert.Equal(t, 10, repoSync.syncMetrics.APICallsSaved)
	assert.Equal(t, 2, repoSync.syncMetrics.CacheHits)
	assert.Equal(t, 1, repoSync.syncMetrics.CacheMisses)
	assert.Equal(t, 3, repoSync.syncMetrics.TotalAPIRequests)
}

func TestRepositorySync_writePerformanceMetricsEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		syncMetrics *PerformanceMetrics
		expected    []string
		notExpected []string
	}{
		{
			name:        "nil syncMetrics",
			syncMetrics: nil,
			expected:    []string{},
			notExpected: []string{"## Performance Metrics"},
		},
		{
			name: "empty metrics",
			syncMetrics: &PerformanceMetrics{
				FileMetrics: FileProcessingMetrics{},
			},
			expected:    []string{"## Performance Metrics"},
			notExpected: []string{"**Files processed**", "**API calls saved**"},
		},
		{
			name: "only file metrics",
			syncMetrics: &PerformanceMetrics{
				FileMetrics: FileProcessingMetrics{
					FilesProcessed:   3,
					FilesChanged:     2,
					FilesSkipped:     1,
					ProcessingTimeMs: 500,
				},
			},
			expected: []string{
				"## Performance Metrics",
				"**Files processed**: 3 (2 changed, 0 deleted, 1 skipped)",
				"**File processing time**: 500ms",
			},
			notExpected: []string{"**Directory files processed**"},
		},
		{
			name: "directories only with changes (PR scenario)",
			syncMetrics: &PerformanceMetrics{
				FileMetrics: FileProcessingMetrics{
					FilesProcessed:   4, // Updated to include directory files
					FilesChanged:     4, // Actual changes from directories
					FilesSkipped:     0, // No files skipped
					ProcessingTimeMs: 6511,
				},
				DirectoryMetrics: map[string]DirectoryMetrics{
					".github/workflows": {
						FilesProcessed: 2,
						FilesExcluded:  0,
					},
					".github/actions": {
						FilesProcessed: 1,
						FilesExcluded:  0,
					},
					".vscode": {
						FilesProcessed: 1,
						FilesExcluded:  0,
					},
				},
			},
			expected: []string{
				"## Performance Metrics",
				"**Files processed**: 4 (4 changed, 0 deleted, 0 skipped)", // Should show correct totals
				"**File processing time**: 6511ms",
			},
			notExpected: []string{"**Directory files processed**"}, // Should not show separate directory section
		},
		{
			name: "mixed files and directories",
			syncMetrics: &PerformanceMetrics{
				FileMetrics: FileProcessingMetrics{
					FilesProcessed:   10, // 2 individual files + 8 directory files
					FilesChanged:     6,  // Actual total changes
					FilesSkipped:     4,  // Some files were skipped
					ProcessingTimeMs: 1500,
				},
				DirectoryMetrics: map[string]DirectoryMetrics{
					".github/workflows": {
						FilesProcessed: 5,
						FilesExcluded:  2,
					},
					"docs": {
						FilesProcessed: 3,
						FilesExcluded:  1,
					},
				},
				APICallsSaved: 12,
				CacheHits:     8,
				CacheMisses:   2,
			},
			expected: []string{
				"## Performance Metrics",
				"**Files processed**: 10 (6 changed, 0 deleted, 4 skipped)",
				"**File processing time**: 1500ms",
				"**API calls saved**: 12 (through optimization)",
				"**Cache hit rate**: 80.0% (8 hits, 2 misses)",
			},
			notExpected: []string{"**Directory files processed**"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoSync := &RepositorySync{
				sourceState: &state.SourceState{
					Repo:         "org/template",
					LatestCommit: "abc123",
				},
				target: config.TargetConfig{
					Repo: "org/target",
				},
				syncMetrics: tt.syncMetrics,
			}

			body := repoSync.generatePRBody("commit456", []FileChange{}, nil)

			for _, expected := range tt.expected {
				assert.Contains(t, body, expected, "Expected content missing")
			}

			for _, notExpected := range tt.notExpected {
				assert.NotContains(t, body, notExpected, "Unexpected content found")
			}
		})
	}
}
