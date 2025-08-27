package config

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestConfig_validateDirectoryLists(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		expectedErr string
	}{
		{
			name: "valid directory lists",
			config: &Config{
				DirectoryLists: []DirectoryList{
					{
						ID:   "list1",
						Name: "Test List 1",
						Directories: []DirectoryMapping{
							{
								Src:  "/source/path1",
								Dest: "/dest/path1",
							},
						},
					},
					{
						ID:   "list2",
						Name: "Test List 2",
						Directories: []DirectoryMapping{
							{
								Src:  "/source/path2",
								Dest: "/dest/path2",
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty list ID",
			config: &Config{
				DirectoryLists: []DirectoryList{
					{
						ID:   "", // Empty ID should cause error
						Name: "Test List",
						Directories: []DirectoryMapping{
							{
								Src:  "/source/path",
								Dest: "/dest/path",
							},
						},
					},
				},
			},
			expectError: true,
			expectedErr: "list ID cannot be empty",
		},
		{
			name: "duplicate list ID",
			config: &Config{
				DirectoryLists: []DirectoryList{
					{
						ID:   "duplicate",
						Name: "Test List 1",
						Directories: []DirectoryMapping{
							{
								Src:  "/source/path1",
								Dest: "/dest/path1",
							},
						},
					},
					{
						ID:   "duplicate", // Duplicate ID should cause error
						Name: "Test List 2",
						Directories: []DirectoryMapping{
							{
								Src:  "/source/path2",
								Dest: "/dest/path2",
							},
						},
					},
				},
			},
			expectError: true,
			expectedErr: "duplicate list ID",
		},
		{
			name: "empty list name",
			config: &Config{
				DirectoryLists: []DirectoryList{
					{
						ID:   "list1",
						Name: "", // Empty name should cause error
						Directories: []DirectoryMapping{
							{
								Src:  "/source/path",
								Dest: "/dest/path",
							},
						},
					},
				},
			},
			expectError: true,
			expectedErr: "list name cannot be empty",
		},
		{
			name: "empty source path in directory",
			config: &Config{
				DirectoryLists: []DirectoryList{
					{
						ID:   "list1",
						Name: "Test List",
						Directories: []DirectoryMapping{
							{
								Src:  "", // Empty source path should cause error
								Dest: "/dest/path",
							},
						},
					},
				},
			},
			expectError: true,
			expectedErr: "source path cannot be empty",
		},
		{
			name: "empty directories list (should warn but not error)",
			config: &Config{
				DirectoryLists: []DirectoryList{
					{
						ID:          "list1",
						Name:        "Test List",
						Directories: []DirectoryMapping{}, // Empty directories should warn but not error
					},
				},
			},
			expectError: false,
		},
		{
			name: "context cancellation",
			config: &Config{
				DirectoryLists: []DirectoryList{
					{
						ID:   "list1",
						Name: "Test List",
						Directories: []DirectoryMapping{
							{
								Src:  "/source/path",
								Dest: "/dest/path",
							},
						},
					},
				},
			},
			expectError: false, // Will be tested with canceled context separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logConfig := &logging.LogConfig{
				Debug: logging.DebugFlags{
					Config: true,
				},
			}

			err := tt.config.validateDirectoryLists(ctx, logConfig)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_validateDirectoryLists_ContextCancellation(t *testing.T) {
	config := &Config{
		DirectoryLists: []DirectoryList{
			{
				ID:   "list1",
				Name: "Test List",
				Directories: []DirectoryMapping{
					{
						Src:  "/source/path",
						Dest: "/dest/path",
					},
				},
			},
		},
	}

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	logConfig := &logging.LogConfig{
		Debug: logging.DebugFlags{
			Config: true,
		},
	}

	err := config.validateDirectoryLists(ctx, logConfig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation canceled")
	assert.Contains(t, err.Error(), "context canceled")
}

func TestConfig_validateDirectoryLists_MultipleDirectories(t *testing.T) {
	config := &Config{
		DirectoryLists: []DirectoryList{
			{
				ID:          "complex_list",
				Name:        "Complex Test List",
				Description: "A list with multiple directories",
				Directories: []DirectoryMapping{
					{
						Src:     "/source/path1",
						Dest:    "/dest/path1",
						Exclude: []string{"*.tmp"},
					},
					{
						Src:         "/source/path2",
						Dest:        "/dest/path2",
						IncludeOnly: []string{"*.go"},
					},
					{
						Src:  "/source/path3",
						Dest: "/dest/path3",
					},
				},
			},
		},
	}

	ctx := context.Background()
	logConfig := &logging.LogConfig{
		Debug: logging.DebugFlags{
			Config: true,
		},
	}

	err := config.validateDirectoryLists(ctx, logConfig)
	assert.NoError(t, err)
}

func TestConfig_validateDirectoryLists_NilLogConfig(t *testing.T) {
	config := &Config{
		DirectoryLists: []DirectoryList{
			{
				ID:          "empty_list",
				Name:        "Empty List",
				Directories: []DirectoryMapping{}, // Empty directories
			},
		},
	}

	ctx := context.Background()
	// Test with nil logConfig (should not panic)
	err := config.validateDirectoryLists(ctx, nil)
	assert.NoError(t, err)
}

func TestConfig_validateDirectoryLists_ErrorIndexing(t *testing.T) {
	config := &Config{
		DirectoryLists: []DirectoryList{
			{
				ID:   "good_list",
				Name: "Good List",
				Directories: []DirectoryMapping{
					{
						Src:  "/source/good",
						Dest: "/dest/good",
					},
				},
			},
			{
				ID:   "", // Error in second list
				Name: "Bad List",
				Directories: []DirectoryMapping{
					{
						Src:  "/source/bad",
						Dest: "/dest/bad",
					},
				},
			},
		},
	}

	ctx := context.Background()
	err := config.validateDirectoryLists(ctx, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "directory_list[1]") // Should reference the correct index
	assert.Contains(t, err.Error(), "list ID cannot be empty")
}

func TestConfig_validateDirectoryLists_DirectoryIndexing(t *testing.T) {
	config := &Config{
		DirectoryLists: []DirectoryList{
			{
				ID:   "test_list",
				Name: "Test List",
				Directories: []DirectoryMapping{
					{
						Src:  "/source/good",
						Dest: "/dest/good",
					},
					{
						Src:  "", // Error in second directory
						Dest: "/dest/bad",
					},
				},
			},
		},
	}

	ctx := context.Background()
	err := config.validateDirectoryLists(ctx, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "directory_list[0] (test_list) directory[1]") // Should reference correct indices
	assert.Contains(t, err.Error(), "source path cannot be empty")
}

// Benchmark for the validation function
func BenchmarkConfig_validateDirectoryLists(b *testing.B) {
	config := &Config{
		DirectoryLists: make([]DirectoryList, 100), // Create 100 directory lists
	}

	// Populate with valid data
	for i := 0; i < 100; i++ {
		config.DirectoryLists[i] = DirectoryList{
			ID:   fmt.Sprintf("list_%d", i),
			Name: fmt.Sprintf("Test List %d", i),
			Directories: []DirectoryMapping{
				{
					Src:  fmt.Sprintf("/source/path_%d", i),
					Dest: fmt.Sprintf("/dest/path_%d", i),
				},
			},
		}
	}

	ctx := context.Background()
	logConfig := &logging.LogConfig{
		Debug: logging.DebugFlags{
			Config: false, // Disable debug logging for benchmark
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.validateDirectoryLists(ctx, logConfig)
		if err != nil {
			b.Fatal(err)
		}
	}
}
