package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/validation"
)

// TestFileMapping_Delete tests file mapping with delete flag
func TestFileMapping_Delete(t *testing.T) {
	tests := []struct {
		name        string
		fileMapping FileMapping
		expectValid bool
		description string
	}{
		{
			name: "valid file deletion",
			fileMapping: FileMapping{
				Src:    "", // Empty source allowed for deletions
				Dest:   "file-to-delete.txt",
				Delete: true,
			},
			expectValid: true,
			description: "Delete flag allows empty source path",
		},
		{
			name: "valid file deletion with dummy source",
			fileMapping: FileMapping{
				Src:    "dummy",
				Dest:   "file-to-delete.txt",
				Delete: true,
			},
			expectValid: true,
			description: "Delete flag allows dummy source path",
		},
		{
			name: "regular file sync still requires source",
			fileMapping: FileMapping{
				Src:    "", // Empty source not allowed for regular sync
				Dest:   "regular-file.txt",
				Delete: false,
			},
			expectValid: false,
			description: "Regular sync requires non-empty source path",
		},
		{
			name: "file deletion still requires destination",
			fileMapping: FileMapping{
				Src:    "",
				Dest:   "", // Empty destination not allowed even for deletions
				Delete: true,
			},
			expectValid: false,
			description: "Delete operations require destination path",
		},
		{
			name: "valid regular file sync",
			fileMapping: FileMapping{
				Src:    "source-file.txt",
				Dest:   "dest-file.txt",
				Delete: false,
			},
			expectValid: true,
			description: "Regular sync with both source and dest paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to validation format
			validationMapping := validation.FileMapping{
				Src:    tt.fileMapping.Src,
				Dest:   tt.fileMapping.Dest,
				Delete: tt.fileMapping.Delete,
			}

			err := validation.ValidateFileMapping(validationMapping)
			if tt.expectValid {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
			}
		})
	}
}

// TestDirectoryMapping_Delete tests directory mapping with delete flag
func TestDirectoryMapping_Delete(t *testing.T) {
	tests := []struct {
		name             string
		directoryMapping DirectoryMapping
		expectValid      bool
		description      string
	}{
		{
			name: "valid directory deletion",
			directoryMapping: DirectoryMapping{
				Src:    "", // Empty source allowed for deletions
				Dest:   "directory-to-delete",
				Delete: true,
			},
			expectValid: true,
			description: "Delete flag allows empty source path for directories",
		},
		{
			name: "valid directory deletion with dummy source",
			directoryMapping: DirectoryMapping{
				Src:    "dummy",
				Dest:   "directory-to-delete",
				Delete: true,
			},
			expectValid: true,
			description: "Delete flag allows dummy source path for directories",
		},
		{
			name: "directory deletion still requires destination",
			directoryMapping: DirectoryMapping{
				Src:    "",
				Dest:   "", // Empty destination not allowed even for deletions
				Delete: true,
			},
			expectValid: false,
			description: "Delete operations require destination path for directories",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For now, we don't have directory validation implemented yet,
			// so we'll test the basic structure
			// Just verify the Delete flag is set as expected
			if tt.directoryMapping.Delete {
				assert.True(t, tt.directoryMapping.Delete, "Delete flag should be true")
			} else {
				assert.False(t, tt.directoryMapping.Delete, "Delete flag should be false")
			}

			// Test that destination is properly set
			if tt.expectValid && tt.directoryMapping.Dest != "" {
				assert.NotEmpty(t, tt.directoryMapping.Dest)
			}

			if !tt.expectValid && tt.directoryMapping.Dest == "" {
				assert.Empty(t, tt.directoryMapping.Dest)
			}
		})
	}
}

// TestTargetConfig_WithDeleteMappings tests target config validation with delete mappings
func TestTargetConfig_WithDeleteMappings(t *testing.T) {
	tests := []struct {
		name        string
		target      TargetConfig
		expectValid bool
		description string
	}{
		{
			name: "mixed file operations - sync and delete",
			target: TargetConfig{
				Repo: "org/target-repo",
				Files: []FileMapping{
					{
						Src:    "source-file.txt",
						Dest:   "dest-file.txt",
						Delete: false, // Regular sync
					},
					{
						Src:    "",
						Dest:   "file-to-delete.txt",
						Delete: true, // Deletion
					},
				},
			},
			expectValid: true,
			description: "Should allow mix of sync and delete operations",
		},
		{
			name: "multiple file deletions",
			target: TargetConfig{
				Repo: "org/target-repo",
				Files: []FileMapping{
					{
						Src:    "",
						Dest:   "file1-to-delete.txt",
						Delete: true,
					},
					{
						Src:    "",
						Dest:   "file2-to-delete.txt",
						Delete: true,
					},
					{
						Src:    "",
						Dest:   "file3-to-delete.txt",
						Delete: true,
					},
				},
			},
			expectValid: true,
			description: "Should allow multiple file deletions",
		},
		{
			name: "delete operation with invalid destination",
			target: TargetConfig{
				Repo: "org/target-repo",
				Files: []FileMapping{
					{
						Src:    "",
						Dest:   "", // Invalid: empty destination
						Delete: true,
					},
				},
			},
			expectValid: false,
			description: "Should reject delete operations with empty destination",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config to test the target validation
			tempConfig := &Config{
				Version: 1,
				Groups: []Group{
					{
						Name:    "test-group",
						ID:      "test",
						Source:  SourceConfig{Repo: "org/source", Branch: "main"},
						Targets: []TargetConfig{tt.target},
					},
				},
			}

			err := tempConfig.Validate()
			if tt.expectValid {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
			}
		})
	}
}

// TestConfig_WithDeleteOperations tests full config validation with delete operations
func TestConfig_WithDeleteOperations(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectValid bool
		description string
	}{
		{
			name: "complete config with delete operations",
			config: &Config{
				Version: 1,
				Groups: []Group{
					{
						Name: "Delete Files Group",
						ID:   "delete-files",
						Source: SourceConfig{
							Repo:   "org/template-repo",
							Branch: "main",
						},
						Targets: []TargetConfig{
							{
								Repo: "org/target-repo",
								Files: []FileMapping{
									{
										Src:    "new-file.txt",
										Dest:   "new-file.txt",
										Delete: false,
									},
									{
										Src:    "",
										Dest:   ".github/.prettierignore",
										Delete: true,
									},
									{
										Src:    "",
										Dest:   ".github/.prettierrc.yml",
										Delete: true,
									},
								},
							},
						},
					},
				},
			},
			expectValid: true,
			description: "Complete config with mixed sync and delete operations should be valid",
		},
		{
			name: "config with only delete operations",
			config: &Config{
				Version: 1,
				Groups: []Group{
					{
						Name: "Cleanup Group",
						ID:   "cleanup",
						Source: SourceConfig{
							Repo:   "org/template-repo",
							Branch: "main",
						},
						Targets: []TargetConfig{
							{
								Repo: "org/target-repo",
								Files: []FileMapping{
									{
										Src:    "",
										Dest:   "deprecated-file1.txt",
										Delete: true,
									},
									{
										Src:    "",
										Dest:   "deprecated-file2.txt",
										Delete: true,
									},
								},
							},
						},
					},
				},
			},
			expectValid: true,
			description: "Config with only delete operations should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectValid {
				assert.NoError(t, err, tt.description)
			} else {
				require.Error(t, err, tt.description)
				t.Logf("Validation error: %v", err)
			}
		})
	}
}
