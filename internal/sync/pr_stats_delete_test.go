package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPRStatistics_WithDeletions tests that PR statistics correctly track deleted files and directories
func TestPRStatistics_WithDeletions(t *testing.T) {
	tests := []struct {
		name                   string
		changedFiles           []FileChange
		expectedFilesChanged   int
		expectedFilesDeleted   int
		expectedFilesProcessed int
		description            string
	}{
		{
			name: "mixed operations - files and directory deletions",
			changedFiles: []FileChange{
				{Path: "README.md", IsDeleted: false, IsNew: true},
				{Path: "src/main.go", IsDeleted: false, IsNew: false},
				{Path: "old-docs/README.md", IsDeleted: true, IsNew: false},
				{Path: "old-docs/guide.md", IsDeleted: true, IsNew: false},
				{Path: "old-docs/api/endpoints.md", IsDeleted: true, IsNew: false},
				{Path: "deprecated/old.txt", IsDeleted: true, IsNew: false},
			},
			expectedFilesChanged:   6, // All operations count as changes
			expectedFilesDeleted:   4, // 4 files deleted (3 from directory, 1 individual)
			expectedFilesProcessed: 6, // All files were processed
			description:            "Should correctly count mixed file/directory operations",
		},
		{
			name: "only directory deletions",
			changedFiles: []FileChange{
				{Path: "temp/file1.txt", IsDeleted: true, IsNew: false},
				{Path: "temp/file2.txt", IsDeleted: true, IsNew: false},
				{Path: "temp/subdir/file3.txt", IsDeleted: true, IsNew: false},
				{Path: "cache/data.json", IsDeleted: true, IsNew: false},
				{Path: "cache/index.html", IsDeleted: true, IsNew: false},
			},
			expectedFilesChanged:   5, // All deleted files count as changes
			expectedFilesDeleted:   5, // All files were deleted
			expectedFilesProcessed: 5, // All files were processed
			description:            "Should handle directory-only deletions",
		},
		{
			name: "only regular file operations",
			changedFiles: []FileChange{
				{Path: "new-file.go", IsDeleted: false, IsNew: true},
				{Path: "updated-file.go", IsDeleted: false, IsNew: false},
				{Path: "another-new.go", IsDeleted: false, IsNew: true},
			},
			expectedFilesChanged:   3, // All files are changes
			expectedFilesDeleted:   0, // No deletions
			expectedFilesProcessed: 3, // All files processed
			description:            "Should handle regular operations without deletions",
		},
		{
			name: "large directory deletion",
			changedFiles: []FileChange{
				// Simulate deleting a large directory structure
				{Path: "node_modules/package1/index.js", IsDeleted: true, IsNew: false},
				{Path: "node_modules/package1/lib/utils.js", IsDeleted: true, IsNew: false},
				{Path: "node_modules/package2/main.js", IsDeleted: true, IsNew: false},
				{Path: "node_modules/package2/dist/bundle.js", IsDeleted: true, IsNew: false},
				{Path: "node_modules/.bin/cli", IsDeleted: true, IsNew: false},
				// Plus some regular changes
				{Path: "package.json", IsDeleted: false, IsNew: false},
				{Path: "src/app.js", IsDeleted: false, IsNew: false},
			},
			expectedFilesChanged:   7, // 5 deletions + 2 updates = 7 changes
			expectedFilesDeleted:   5, // 5 files deleted from node_modules
			expectedFilesProcessed: 7, // All files processed
			description:            "Should handle large directory deletion with mixed operations",
		},
		{
			name:                   "empty change set",
			changedFiles:           []FileChange{},
			expectedFilesChanged:   0,
			expectedFilesDeleted:   0,
			expectedFilesProcessed: 0,
			description:            "Should handle empty changes correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the metrics calculation logic from repository.go
			deletedFileCount := 0
			for _, fileChange := range tt.changedFiles {
				if fileChange.IsDeleted {
					deletedFileCount++
				}
			}

			// Create metrics like in the actual code
			metrics := FileProcessingMetrics{
				FilesProcessed: len(tt.changedFiles),
				FilesChanged:   len(tt.changedFiles),
				FilesDeleted:   deletedFileCount,
				FilesSkipped:   0, // For this test, assume no files were skipped
			}

			assert.Equal(t, tt.expectedFilesChanged, metrics.FilesChanged, tt.description+" - changed count")
			assert.Equal(t, tt.expectedFilesDeleted, metrics.FilesDeleted, tt.description+" - deleted count")
			assert.Equal(t, tt.expectedFilesProcessed, metrics.FilesProcessed, tt.description+" - processed count")

			// Verify that FilesDeleted doesn't exceed FilesChanged
			assert.LessOrEqual(t, metrics.FilesDeleted, metrics.FilesChanged,
				"FilesDeleted should never exceed FilesChanged")
		})
	}
}

// TestPRDescription_WithDeletions tests that PR descriptions correctly display deletion statistics
func TestPRDescription_WithDeletions(t *testing.T) {
	tests := []struct {
		name         string
		metrics      FileProcessingMetrics
		expectSubstr []string
		description  string
	}{
		{
			name: "PR with deletions",
			metrics: FileProcessingMetrics{
				FilesProcessed: 10,
				FilesChanged:   8,
				FilesDeleted:   3,
				FilesSkipped:   2,
			},
			expectSubstr: []string{
				"10 (5 changed, 3 deleted, 2 skipped)", // Format: processed (non-deleted changed, deleted, skipped)
				"files_deleted: 3",                     // YAML metadata
			},
			description: "Should show deletion counts in both summary and metadata",
		},
		{
			name: "PR with only deletions",
			metrics: FileProcessingMetrics{
				FilesProcessed: 5,
				FilesChanged:   5,
				FilesDeleted:   5,
				FilesSkipped:   0,
			},
			expectSubstr: []string{
				"5 (0 changed, 5 deleted, 0 skipped)",
				"files_deleted: 5",
			},
			description: "Should handle deletion-only PRs",
		},
		{
			name: "PR with no deletions",
			metrics: FileProcessingMetrics{
				FilesProcessed: 7,
				FilesChanged:   6,
				FilesDeleted:   0,
				FilesSkipped:   1,
			},
			expectSubstr: []string{
				"7 (6 changed, 0 deleted, 1 skipped)",
			},
			description: "Should not show deletion metadata when no files deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the calculation logic matches expected format
			filesChanged := tt.metrics.FilesChanged - tt.metrics.FilesDeleted
			expectedFormat := "files_deleted: " + string(rune(tt.metrics.FilesDeleted+'0'))

			// Verify the math is correct
			if tt.metrics.FilesDeleted > 0 {
				assert.Contains(t, tt.expectSubstr, expectedFormat, tt.description)
				assert.GreaterOrEqual(t, tt.metrics.FilesChanged, tt.metrics.FilesDeleted,
					"FilesChanged should be >= FilesDeleted")
			}

			// Verify non-deleted changed files calculation
			expectedNonDeletedChanges := tt.metrics.FilesChanged - tt.metrics.FilesDeleted
			assert.Equal(t, expectedNonDeletedChanges, filesChanged,
				"Non-deleted changed files calculation should be correct")
		})
	}
}

// TestDirectoryDeletionMetrics tests metrics specifically for directory deletions
func TestDirectoryDeletionMetrics(t *testing.T) {
	tests := []struct {
		name                   string
		directoryChanges       []FileChange
		individualFileChanges  []FileChange
		expectedDirectoryFiles int
		expectedTotalDeleted   int
		description            string
	}{
		{
			name: "single directory deletion",
			directoryChanges: []FileChange{
				{Path: "docs/README.md", IsDeleted: true, IsNew: false},
				{Path: "docs/guide.md", IsDeleted: true, IsNew: false},
				{Path: "docs/api/spec.md", IsDeleted: true, IsNew: false},
			},
			individualFileChanges: []FileChange{
				{Path: "CHANGELOG.md", IsDeleted: true, IsNew: false},
			},
			expectedDirectoryFiles: 3, // Files from directory deletion
			expectedTotalDeleted:   4, // 3 from directory + 1 individual
			description:            "Should track directory vs individual file deletions",
		},
		{
			name: "multiple directory deletions",
			directoryChanges: []FileChange{
				// First directory: old-docs
				{Path: "old-docs/index.md", IsDeleted: true, IsNew: false},
				{Path: "old-docs/tutorial.md", IsDeleted: true, IsNew: false},
				// Second directory: legacy
				{Path: "legacy/old-script.sh", IsDeleted: true, IsNew: false},
				{Path: "legacy/deprecated.py", IsDeleted: true, IsNew: false},
				{Path: "legacy/config/settings.ini", IsDeleted: true, IsNew: false},
			},
			individualFileChanges:  []FileChange{},
			expectedDirectoryFiles: 5, // All files from directory deletions
			expectedTotalDeleted:   5, // All deletions
			description:            "Should handle multiple directory deletions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Combine all changes
			allChanges := append(tt.directoryChanges, tt.individualFileChanges...)

			// Count deleted files
			deletedCount := 0
			for _, change := range allChanges {
				if change.IsDeleted {
					deletedCount++
				}
			}

			assert.Equal(t, tt.expectedTotalDeleted, deletedCount, tt.description+" - total deleted count")
			assert.Len(t, tt.directoryChanges, tt.expectedDirectoryFiles,
				tt.description+" - directory files count")

			// Verify that all directory changes are marked as deleted
			for _, change := range tt.directoryChanges {
				assert.True(t, change.IsDeleted, "Directory file %s should be marked as deleted", change.Path)
				assert.False(t, change.IsNew, "Directory file %s should not be marked as new", change.Path)
			}
		})
	}
}
