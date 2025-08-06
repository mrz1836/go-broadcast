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
