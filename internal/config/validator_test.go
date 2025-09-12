package config

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_Validate tests basic config validation
func TestConfig_Validate(t *testing.T) {
	t.Run("valid config with groups", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Groups: []Group{
				{
					Name: "test-group",
					ID:   "test",
					Source: SourceConfig{
						Repo: "org/template",
					},
					Targets: []TargetConfig{
						{
							Repo: "org/service",
							Files: []FileMapping{
								{Src: "file.txt", Dest: "dest.txt"},
							},
						},
					},
				},
			},
		}

		// Test that config is not nil
		assert.NotNil(t, config)
		assert.Equal(t, 1, config.Version)
		assert.Len(t, config.Groups, 1)
	})

	t.Run("empty config", func(t *testing.T) {
		config := &Config{}
		assert.NotNil(t, config)
		assert.Empty(t, config.Groups)
	})
}

// TestValidateFileLists tests the validateFileLists function with delete operations
func TestValidateFileLists(t *testing.T) {
	tests := []struct {
		name        string
		fileLists   []FileList
		expectValid bool
		errorMsg    string
		description string
	}{
		{
			name: "valid file list with regular files",
			fileLists: []FileList{
				{
					ID:   "regular-files",
					Name: "Regular Files",
					Files: []FileMapping{
						{Src: "source.txt", Dest: "dest.txt", Delete: false},
						{Src: "another.txt", Dest: "another.txt", Delete: false},
					},
				},
			},
			expectValid: true,
			description: "Regular file lists should validate correctly",
		},
		{
			name: "valid file list with delete operations",
			fileLists: []FileList{
				{
					ID:   "delete-files",
					Name: "Files to Delete",
					Files: []FileMapping{
						{Src: "", Dest: ".github/.prettierignore", Delete: true},
						{Src: "", Dest: ".github/.prettierrc.yml", Delete: true},
					},
				},
			},
			expectValid: true,
			description: "Delete operations should allow empty source paths",
		},
		{
			name: "mixed file list with sync and delete operations",
			fileLists: []FileList{
				{
					ID:   "mixed-files",
					Name: "Mixed Operations",
					Files: []FileMapping{
						{Src: "new-file.txt", Dest: "new-file.txt", Delete: false},
						{Src: "", Dest: "old-file.txt", Delete: true},
						{Src: "another-new.txt", Dest: "another-new.txt", Delete: false},
					},
				},
			},
			expectValid: true,
			description: "Mixed sync and delete operations should be valid",
		},
		{
			name: "invalid - empty source for non-delete operation",
			fileLists: []FileList{
				{
					ID:   "invalid-files",
					Name: "Invalid Files",
					Files: []FileMapping{
						{Src: "", Dest: "dest.txt", Delete: false}, // This should fail
					},
				},
			},
			expectValid: false,
			errorMsg:    "source path cannot be empty",
			description: "Non-delete operations should require source path",
		},
		{
			name: "invalid - empty destination for delete operation",
			fileLists: []FileList{
				{
					ID:   "invalid-delete",
					Name: "Invalid Delete",
					Files: []FileMapping{
						{Src: "", Dest: "", Delete: true}, // This should fail - empty dest
					},
				},
			},
			expectValid: false,
			errorMsg:    "destination path cannot be empty",
			description: "Delete operations should require destination path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version:   1,
				FileLists: tt.fileLists,
			}

			err := config.validateFileLists(context.Background(), nil)
			if tt.expectValid {
				assert.NoError(t, err, tt.description)
			} else {
				require.Error(t, err, tt.description)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			}
		})
	}
}

// TestValidateDirectoryLists tests the validateDirectoryLists function with delete operations
func TestValidateDirectoryLists(t *testing.T) {
	tests := []struct {
		name           string
		directoryLists []DirectoryList
		expectValid    bool
		errorMsg       string
		description    string
	}{
		{
			name: "valid directory list with regular directories",
			directoryLists: []DirectoryList{
				{
					ID:   "regular-dirs",
					Name: "Regular Directories",
					Directories: []DirectoryMapping{
						{Src: ".github/workflows", Dest: ".github/workflows", Delete: false},
						{Src: ".vscode", Dest: ".vscode", Delete: false},
					},
				},
			},
			expectValid: true,
			description: "Regular directory lists should validate correctly",
		},
		{
			name: "valid directory list with delete operations",
			directoryLists: []DirectoryList{
				{
					ID:   "delete-dirs",
					Name: "Directories to Delete",
					Directories: []DirectoryMapping{
						{Src: "", Dest: "old-configs", Delete: true},
						{Src: "", Dest: "deprecated", Delete: true},
					},
				},
			},
			expectValid: true,
			description: "Delete operations should allow empty source paths for directories",
		},
		{
			name: "invalid - empty source for non-delete directory operation",
			directoryLists: []DirectoryList{
				{
					ID:   "invalid-dirs",
					Name: "Invalid Directories",
					Directories: []DirectoryMapping{
						{Src: "", Dest: "dest-dir", Delete: false}, // This should fail
					},
				},
			},
			expectValid: false,
			errorMsg:    "source path cannot be empty",
			description: "Non-delete directory operations should require source path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version:        1,
				DirectoryLists: tt.directoryLists,
			}

			err := config.validateDirectoryLists(context.Background(), nil)
			if tt.expectValid {
				assert.NoError(t, err, tt.description)
			} else {
				require.Error(t, err, tt.description)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			}
		})
	}
}

// TestValidateDirectories tests the validateDirectories function
func TestValidateDirectories(t *testing.T) {
	tests := []struct {
		name        string
		target      TargetConfig
		expectErr   bool
		expectedErr error
	}{
		{
			name: "valid directories",
			target: TargetConfig{
				Repo: "org/target",
				Directories: []DirectoryMapping{
					{
						Src:  "src",
						Dest: "dest",
					},
				},
			},
			expectErr: false,
		},
		{
			name: "empty source path",
			target: TargetConfig{
				Repo: "org/target",
				Directories: []DirectoryMapping{
					{
						Src:  "",
						Dest: "dest",
					},
				},
			},
			expectErr:   true,
			expectedErr: ErrEmptySourcePath,
		},
		{
			name: "empty destination path",
			target: TargetConfig{
				Repo: "org/target",
				Directories: []DirectoryMapping{
					{
						Src:  "src",
						Dest: "",
					},
				},
			},
			expectErr:   true,
			expectedErr: ErrEmptyDestPath,
		},
		{
			name: "path traversal in source",
			target: TargetConfig{
				Repo: "org/target",
				Directories: []DirectoryMapping{
					{
						Src:  "../evil",
						Dest: "dest",
					},
				},
			},
			expectErr:   true,
			expectedErr: ErrPathTraversal,
		},
		{
			name: "path traversal in destination",
			target: TargetConfig{
				Repo: "org/target",
				Directories: []DirectoryMapping{
					{
						Src:  "src",
						Dest: "../evil",
					},
				},
			},
			expectErr:   true,
			expectedErr: ErrPathTraversal,
		},
		{
			name: "invalid exclusion pattern",
			target: TargetConfig{
				Repo: "org/target",
				Directories: []DirectoryMapping{
					{
						Src:     "src",
						Dest:    "dest",
						Exclude: []string{"[invalid"},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "valid exclusion patterns",
			target: TargetConfig{
				Repo: "org/target",
				Directories: []DirectoryMapping{
					{
						Src:     "src",
						Dest:    "dest",
						Exclude: []string{"*.tmp", "test/*"},
					},
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := logrus.NewEntry(logrus.StandardLogger())

			err := tt.target.validateDirectories(ctx, logger)

			if tt.expectErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.Contains(t, err.Error(), tt.expectedErr.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateFileDirectoryConflicts tests the validateFileDirectoryConflicts function
func TestValidateFileDirectoryConflicts(t *testing.T) {
	tests := []struct {
		name      string
		target    TargetConfig
		expectErr bool
	}{
		{
			name: "no conflicts",
			target: TargetConfig{
				Repo: "org/target",
				Files: []FileMapping{
					{Src: "file1.txt", Dest: "file1.txt"},
				},
				Directories: []DirectoryMapping{
					{Src: "dir1", Dest: "dir1"},
				},
			},
			expectErr: false,
		},
		{
			name: "file and directory conflict",
			target: TargetConfig{
				Repo: "org/target",
				Files: []FileMapping{
					{Src: "file1.txt", Dest: "conflicted"},
				},
				Directories: []DirectoryMapping{
					{Src: "dir1", Dest: "conflicted"},
				},
			},
			expectErr: true,
		},
		{
			name: "multiple files no conflicts",
			target: TargetConfig{
				Repo: "org/target",
				Files: []FileMapping{
					{Src: "file1.txt", Dest: "file1.txt"},
					{Src: "file2.txt", Dest: "file2.txt"},
				},
				Directories: []DirectoryMapping{
					{Src: "dir1", Dest: "dir1"},
					{Src: "dir2", Dest: "dir2"},
				},
			},
			expectErr: false,
		},
		{
			name: "no actual path conflict case",
			target: TargetConfig{
				Repo: "org/target",
				Files: []FileMapping{
					{Src: "src/file.txt", Dest: "config.yaml"},
				},
				Directories: []DirectoryMapping{
					{Src: "configs", Dest: "configs"},
				},
			},
			expectErr: false,
		},
		{
			name: "directory overwriting file",
			target: TargetConfig{
				Repo: "org/target",
				Files: []FileMapping{
					{Src: "file.txt", Dest: "app"},
				},
				Directories: []DirectoryMapping{
					{Src: "source", Dest: "app"},
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.target.validateFileDirectoryConflicts()

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestTargetConfig_BranchValidation tests target branch validation
func TestTargetConfig_BranchValidation(t *testing.T) {
	tests := []struct {
		name        string
		target      TargetConfig
		expectErr   bool
		expectedErr string
	}{
		{
			name: "valid target with no branch specified",
			target: TargetConfig{
				Repo: "org/target",
				Files: []FileMapping{
					{Src: "file.txt", Dest: "file.txt"},
				},
			},
			expectErr: false,
		},
		{
			name: "valid target with valid branch name",
			target: TargetConfig{
				Repo:   "org/target",
				Branch: "develop",
				Files: []FileMapping{
					{Src: "file.txt", Dest: "file.txt"},
				},
			},
			expectErr: false,
		},
		{
			name: "valid target with branch containing slashes",
			target: TargetConfig{
				Repo:   "org/target",
				Branch: "feature/new-feature",
				Files: []FileMapping{
					{Src: "file.txt", Dest: "file.txt"},
				},
			},
			expectErr: false,
		},
		{
			name: "valid target with branch containing dashes and numbers",
			target: TargetConfig{
				Repo:   "org/target",
				Branch: "release-1.2.3",
				Files: []FileMapping{
					{Src: "file.txt", Dest: "file.txt"},
				},
			},
			expectErr: false,
		},
		{
			name: "invalid target with branch containing spaces",
			target: TargetConfig{
				Repo:   "org/target",
				Branch: "feature branch",
				Files: []FileMapping{
					{Src: "file.txt", Dest: "file.txt"},
				},
			},
			expectErr:   true,
			expectedErr: "invalid target branch name",
		},
		{
			name: "invalid target with branch starting with special character",
			target: TargetConfig{
				Repo:   "org/target",
				Branch: "-feature",
				Files: []FileMapping{
					{Src: "file.txt", Dest: "file.txt"},
				},
			},
			expectErr:   true,
			expectedErr: "invalid target branch name",
		},
		{
			name: "invalid target with branch containing invalid characters",
			target: TargetConfig{
				Repo:   "org/target",
				Branch: "feature@branch",
				Files: []FileMapping{
					{Src: "file.txt", Dest: "file.txt"},
				},
			},
			expectErr:   true,
			expectedErr: "invalid target branch name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := logrus.NewEntry(logrus.StandardLogger())

			err := tt.target.validateWithLogging(ctx, nil, logger)

			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
