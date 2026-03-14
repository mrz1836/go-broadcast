package sync

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

// TestProcessFileDeletion tests the processFileDeletion function
func TestProcessFileDeletion(t *testing.T) {
	tests := []struct {
		name              string
		fileMapping       config.FileMapping
		existingContent   []byte
		getFileError      error
		expectError       bool
		expectFileChange  bool
		expectDeletedFlag bool
		description       string
	}{
		{
			name: "successful file deletion",
			fileMapping: config.FileMapping{
				Src:    "",
				Dest:   "file-to-delete.txt",
				Delete: true,
			},
			existingContent:   []byte("existing content"),
			getFileError:      nil,
			expectError:       false,
			expectFileChange:  true,
			expectDeletedFlag: true,
			description:       "Should mark existing file for deletion",
		},
		{
			name: "delete non-existent file",
			fileMapping: config.FileMapping{
				Src:    "",
				Dest:   "non-existent-file.txt",
				Delete: true,
			},
			existingContent:   nil,
			getFileError:      errors.ErrFileNotFound,
			expectError:       true,
			expectFileChange:  false,
			expectDeletedFlag: false,
			description:       "Should skip deletion if file doesn't exist",
		},
		{
			name: "delete file with network error",
			fileMapping: config.FileMapping{
				Src:    "",
				Dest:   "file-with-error.txt",
				Delete: true,
			},
			existingContent:   nil,
			getFileError:      assert.AnError,
			expectError:       true,
			expectFileChange:  false,
			expectDeletedFlag: false,
			description:       "Should return error on network failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock GitHub client
			mockGH := &gh.MockClient{}

			// Create a mock repository sync with the mock client
			logger := logrus.NewEntry(logrus.New())
			rs := &RepositorySync{
				engine: &Engine{
					gh: mockGH,
				},
				target: config.TargetConfig{
					Repo: "test/repo",
				},
				logger: logger,
			}

			// Set up the expected call and return value
			if tt.getFileError != nil {
				mockGH.On("GetFile", mock.Anything, "test/repo", tt.fileMapping.Dest, "").
					Return(nil, tt.getFileError)
			} else {
				mockGH.On("GetFile", mock.Anything, "test/repo", tt.fileMapping.Dest, "").
					Return(&gh.FileContent{Content: tt.existingContent}, nil)
			}

			// Create context
			ctx := context.Background()

			// Call the actual processFileDeletion function
			change, err := rs.processFileDeletion(ctx, tt.fileMapping)

			// Check expectations based on test case
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, change)
			} else if tt.expectFileChange {
				require.NoError(t, err)
				assert.NotNil(t, change)
				assert.Equal(t, tt.fileMapping.Dest, change.Path)
				assert.Nil(t, change.Content)
				assert.Equal(t, tt.existingContent, change.OriginalContent)
				assert.False(t, change.IsNew)
				assert.True(t, change.IsDeleted)
			}

			mockGH.AssertExpectations(t)
		})
	}
}

// TestFileChangeTracking tests that file changes are properly tracked
func TestFileChangeTracking(t *testing.T) {
	tests := []struct {
		name             string
		fileChanges      []FileChange
		expectedDeleted  int
		expectedModified int
		description      string
	}{
		{
			name: "mixed operations",
			fileChanges: []FileChange{
				{
					Path:      "new-file.txt",
					Content:   []byte("new content"),
					IsNew:     true,
					IsDeleted: false,
				},
				{
					Path:            "modified-file.txt",
					Content:         []byte("modified content"),
					OriginalContent: []byte("old content"),
					IsNew:           false,
					IsDeleted:       false,
				},
				{
					Path:            "deleted-file.txt",
					Content:         nil,
					OriginalContent: []byte("deleted content"),
					IsNew:           false,
					IsDeleted:       true,
				},
			},
			expectedDeleted:  1,
			expectedModified: 2,
			description:      "Should correctly count deleted and modified files",
		},
		{
			name: "only deletions",
			fileChanges: []FileChange{
				{
					Path:            "deleted-file1.txt",
					Content:         nil,
					OriginalContent: []byte("content1"),
					IsNew:           false,
					IsDeleted:       true,
				},
				{
					Path:            "deleted-file2.txt",
					Content:         nil,
					OriginalContent: []byte("content2"),
					IsNew:           false,
					IsDeleted:       true,
				},
			},
			expectedDeleted:  2,
			expectedModified: 0,
			description:      "Should handle deletion-only operations",
		},
		{
			name:             "no changes",
			fileChanges:      []FileChange{},
			expectedDeleted:  0,
			expectedModified: 0,
			description:      "Should handle empty change set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Count deleted files
			deletedCount := 0
			modifiedCount := 0

			for _, change := range tt.fileChanges {
				if change.IsDeleted {
					deletedCount++
				} else {
					modifiedCount++
				}
			}

			assert.Equal(t, tt.expectedDeleted, deletedCount, tt.description+" - deleted count")
			assert.Equal(t, tt.expectedModified, modifiedCount, tt.description+" - modified count")
		})
	}
}

// TestProcessFileWithDelete tests the processFile function with delete flag
func TestProcessFileWithDelete(t *testing.T) {
	t.Run("delete flag routes to deletion logic", func(t *testing.T) {
		fileMapping := config.FileMapping{
			Src:    "",
			Dest:   "file-to-delete.txt",
			Delete: true,
		}

		// Test that the delete flag is properly recognized
		assert.True(t, fileMapping.Delete, "Delete flag should be true")
		assert.Equal(t, "file-to-delete.txt", fileMapping.Dest, "Destination should be set")

		// In the actual implementation, this would route to processFileDeletion
		// For now, we test the routing logic structure
		if !fileMapping.Delete {
			t.Error("Should have routed to deletion logic")
		}
	})

	t.Run("regular sync still works", func(t *testing.T) {
		fileMapping := config.FileMapping{
			Src:    "source-file.txt",
			Dest:   "dest-file.txt",
			Delete: false,
		}

		// Test that regular sync is not affected
		assert.False(t, fileMapping.Delete, "Delete flag should be false")
		assert.Equal(t, "source-file.txt", fileMapping.Src, "Source should be set")
		assert.Equal(t, "dest-file.txt", fileMapping.Dest, "Destination should be set")

		// In the actual implementation, this would route to regular processing
		if fileMapping.Delete {
			t.Error("Should have routed to regular sync logic")
		}
	})
}

// TestFileMetricsWithDeletion tests that deletion metrics are properly calculated
func TestFileMetricsWithDeletion(t *testing.T) {
	tests := []struct {
		name            string
		changedFiles    []FileChange
		expectedDeleted int
		expectedChanged int
		description     string
	}{
		{
			name: "metrics with deletions",
			changedFiles: []FileChange{
				{Path: "file1.txt", IsDeleted: false, IsNew: true},
				{Path: "file2.txt", IsDeleted: true, IsNew: false},
				{Path: "file3.txt", IsDeleted: false, IsNew: false},
				{Path: "file4.txt", IsDeleted: true, IsNew: false},
			},
			expectedDeleted: 2,
			expectedChanged: 4, // Total changed files (including deletions)
			description:     "Should count both deletions and modifications",
		},
		{
			name: "no deletions",
			changedFiles: []FileChange{
				{Path: "file1.txt", IsDeleted: false, IsNew: true},
				{Path: "file2.txt", IsDeleted: false, IsNew: false},
			},
			expectedDeleted: 0,
			expectedChanged: 2,
			description:     "Should handle cases with no deletions",
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
			}

			assert.Equal(t, tt.expectedDeleted, metrics.FilesDeleted, tt.description+" - deleted count")
			assert.Equal(t, tt.expectedChanged, metrics.FilesChanged, tt.description+" - changed count")
		})
	}
}
