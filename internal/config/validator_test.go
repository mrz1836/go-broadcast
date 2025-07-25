package config

import (
	"context"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantError   bool
		expectedErr error
	}{
		{
			name: "valid configuration",
			config: &Config{
				Version: 1,
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "main",
				},
				Defaults: DefaultConfig{
					BranchPrefix: "sync/template",
					PRLabels:     []string{"automated-sync"},
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
			wantError: false,
		},
		{
			name: "invalid version",
			config: &Config{
				Version: 2,
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "main",
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
			wantError:   true,
			expectedErr: ErrUnsupportedVersion,
		},
		{
			name: "no targets",
			config: &Config{
				Version: 1,
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "main",
				},
				Targets: []TargetConfig{},
			},
			wantError:   true,
			expectedErr: ErrNoTargets,
		},
		{
			name: "duplicate target repositories",
			config: &Config{
				Version: 1,
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "main",
				},
				Targets: []TargetConfig{
					{
						Repo: "org/service",
						Files: []FileMapping{
							{Src: "file1.txt", Dest: "dest1.txt"},
						},
					},
					{
						Repo: "org/service",
						Files: []FileMapping{
							{Src: "file2.txt", Dest: "dest2.txt"},
						},
					},
				},
			},
			wantError:   true,
			expectedErr: ErrDuplicateTarget,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				require.Error(t, err)
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_ValidateWithLogging(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		logConfig *logging.LogConfig
		wantError bool
	}{
		{
			name: "valid configuration with debug logging",
			config: &Config{
				Version: 1,
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "main",
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
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{
					Config: true,
				},
			},
			wantError: false,
		},
		{
			name: "valid configuration with nil log config",
			config: &Config{
				Version: 1,
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "main",
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
			logConfig: nil,
			wantError: false,
		},
		{
			name: "context cancellation",
			config: &Config{
				Version: 1,
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "main",
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
			logConfig: nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.name == "context cancellation" {
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel() // Cancel immediately
				ctx = cancelCtx
			}

			err := tt.config.ValidateWithLogging(ctx, tt.logConfig)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_validateSourceWithLogging(t *testing.T) {
	tests := []struct {
		name        string
		source      SourceConfig
		wantError   bool
		expectedErr error
	}{
		{
			name: "valid source",
			source: SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			wantError: false,
		},
		{
			name: "missing repository",
			source: SourceConfig{
				Repo:   "",
				Branch: "main",
			},
			wantError:   true,
			expectedErr: ErrSourceRepoRequired,
		},
		{
			name: "invalid repository format - no slash",
			source: SourceConfig{
				Repo:   "orgtemplate",
				Branch: "main",
			},
			wantError:   true,
			expectedErr: ErrInvalidRepoFormat,
		},
		{
			name: "invalid repository format - starts with slash",
			source: SourceConfig{
				Repo:   "/org/template",
				Branch: "main",
			},
			wantError:   true,
			expectedErr: ErrInvalidRepoFormat,
		},
		{
			name: "invalid repository format - ends with slash",
			source: SourceConfig{
				Repo:   "org/template/",
				Branch: "main",
			},
			wantError:   true,
			expectedErr: ErrInvalidRepoFormat,
		},
		{
			name: "invalid repository format - multiple slashes",
			source: SourceConfig{
				Repo:   "org/sub/template",
				Branch: "main",
			},
			wantError:   true,
			expectedErr: ErrInvalidRepoFormat,
		},
		{
			name: "missing branch",
			source: SourceConfig{
				Repo:   "org/template",
				Branch: "",
			},
			wantError:   true,
			expectedErr: ErrSourceBranchRequired,
		},
		{
			name: "invalid branch name - starts with special character",
			source: SourceConfig{
				Repo:   "org/template",
				Branch: "-main",
			},
			wantError:   true,
			expectedErr: ErrInvalidBranchName,
		},
		{
			name: "valid branch with slashes and dots",
			source: SourceConfig{
				Repo:   "org/template",
				Branch: "feature/test.branch",
			},
			wantError: false,
		},
		{
			name: "valid branch with hyphen",
			source: SourceConfig{
				Repo:   "org/template",
				Branch: "feature-branch",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version: 1,
				Source:  tt.source,
			}

			ctx := context.Background()
			err := config.validateSourceWithLogging(ctx, nil)

			if tt.wantError {
				require.Error(t, err)
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_validateDefaultsWithLogging(t *testing.T) {
	tests := []struct {
		name        string
		defaults    DefaultConfig
		wantError   bool
		expectedErr error
	}{
		{
			name: "valid defaults",
			defaults: DefaultConfig{
				BranchPrefix: "sync/template",
				PRLabels:     []string{"automated-sync", "enhancement"},
			},
			wantError: false,
		},
		{
			name: "empty defaults",
			defaults: DefaultConfig{
				BranchPrefix: "",
				PRLabels:     []string{},
			},
			wantError: false,
		},
		{
			name: "invalid branch prefix",
			defaults: DefaultConfig{
				BranchPrefix: "-invalid",
				PRLabels:     []string{"automated-sync"},
			},
			wantError:   true,
			expectedErr: ErrInvalidBranchPrefix,
		},
		{
			name: "empty PR label",
			defaults: DefaultConfig{
				BranchPrefix: "sync/template",
				PRLabels:     []string{"automated-sync", "", "enhancement"},
			},
			wantError:   true,
			expectedErr: ErrEmptyPRLabel,
		},
		{
			name: "whitespace only PR label",
			defaults: DefaultConfig{
				BranchPrefix: "sync/template",
				PRLabels:     []string{"automated-sync", "   ", "enhancement"},
			},
			wantError:   true,
			expectedErr: ErrEmptyPRLabel,
		},
		{
			name: "valid branch prefix with dots and hyphens",
			defaults: DefaultConfig{
				BranchPrefix: "sync.template-v1",
				PRLabels:     []string{"automated-sync"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version:  1,
				Defaults: tt.defaults,
			}

			ctx := context.Background()
			err := config.validateDefaultsWithLogging(ctx, nil)

			if tt.wantError {
				require.Error(t, err)
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTargetConfig_validateWithLogging(t *testing.T) {
	tests := []struct {
		name        string
		target      TargetConfig
		wantError   bool
		expectedErr error
	}{
		{
			name: "valid target",
			target: TargetConfig{
				Repo: "org/service",
				Files: []FileMapping{
					{Src: "src/file.txt", Dest: "dest/file.txt"},
				},
				Transform: Transform{
					RepoName:  true,
					Variables: map[string]string{"KEY": "value"},
				},
			},
			wantError: false,
		},
		{
			name: "missing repository",
			target: TargetConfig{
				Repo: "",
				Files: []FileMapping{
					{Src: "src/file.txt", Dest: "dest/file.txt"},
				},
			},
			wantError:   true,
			expectedErr: ErrRepoRequired,
		},
		{
			name: "invalid repository format",
			target: TargetConfig{
				Repo: "invalid-repo",
				Files: []FileMapping{
					{Src: "src/file.txt", Dest: "dest/file.txt"},
				},
			},
			wantError:   true,
			expectedErr: ErrInvalidRepoFormat,
		},
		{
			name: "no file mappings",
			target: TargetConfig{
				Repo:  "org/service",
				Files: []FileMapping{},
			},
			wantError:   true,
			expectedErr: ErrNoFileMappings,
		},
		{
			name: "duplicate destination files",
			target: TargetConfig{
				Repo: "org/service",
				Files: []FileMapping{
					{Src: "src/file1.txt", Dest: "dest/same.txt"},
					{Src: "src/file2.txt", Dest: "dest/same.txt"},
				},
			},
			wantError:   true,
			expectedErr: ErrDuplicateDestination,
		},
		{
			name: "valid transform configuration",
			target: TargetConfig{
				Repo: "org/service",
				Files: []FileMapping{
					{Src: "template.txt", Dest: "output.txt"},
				},
				Transform: Transform{
					RepoName: false,
					Variables: map[string]string{
						"PROJECT_NAME": "my-service",
						"VERSION":      "1.0.0",
					},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.target.validateWithLogging(ctx, nil, nil)

			if tt.wantError {
				require.Error(t, err)
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFileMapping_validateWithLogging(t *testing.T) {
	tests := []struct {
		name        string
		fileMapping FileMapping
		wantError   bool
		expectedErr error
	}{
		{
			name: "valid file mapping",
			fileMapping: FileMapping{
				Src:  "src/file.txt",
				Dest: "dest/file.txt",
			},
			wantError: false,
		},
		{
			name: "missing source path",
			fileMapping: FileMapping{
				Src:  "",
				Dest: "dest/file.txt",
			},
			wantError:   true,
			expectedErr: ErrSourcePathRequired,
		},
		{
			name: "missing destination path",
			fileMapping: FileMapping{
				Src:  "src/file.txt",
				Dest: "",
			},
			wantError:   true,
			expectedErr: ErrDestPathRequired,
		},
		{
			name: "absolute source path",
			fileMapping: FileMapping{
				Src:  "/absolute/path/file.txt",
				Dest: "dest/file.txt",
			},
			wantError:   true,
			expectedErr: ErrInvalidSourcePath,
		},
		{
			name: "source path with parent directory traversal",
			fileMapping: FileMapping{
				Src:  "../outside/file.txt",
				Dest: "dest/file.txt",
			},
			wantError:   true,
			expectedErr: ErrInvalidSourcePath,
		},
		{
			name: "absolute destination path",
			fileMapping: FileMapping{
				Src:  "src/file.txt",
				Dest: "/absolute/dest/file.txt",
			},
			wantError:   true,
			expectedErr: ErrInvalidDestPath,
		},
		{
			name: "destination path with parent directory traversal",
			fileMapping: FileMapping{
				Src:  "src/file.txt",
				Dest: "../outside/file.txt",
			},
			wantError:   true,
			expectedErr: ErrInvalidDestPath,
		},
		{
			name: "complex valid paths",
			fileMapping: FileMapping{
				Src:  "src/nested/deep/file.txt",
				Dest: "dest/different/structure/file.txt",
			},
			wantError: false,
		},
		{
			name: "paths with dots in filename",
			fileMapping: FileMapping{
				Src:  "src/file.config.yaml",
				Dest: "dest/app.config.yaml",
			},
			wantError: false,
		},
		{
			name: "source path with complex traversal attempt",
			fileMapping: FileMapping{
				Src:  "src/../../../etc/passwd",
				Dest: "dest/file.txt",
			},
			wantError:   true,
			expectedErr: ErrInvalidSourcePath,
		},
		{
			name: "destination path with complex traversal attempt",
			fileMapping: FileMapping{
				Src:  "src/file.txt",
				Dest: "dest/../../../tmp/malicious.txt",
			},
			wantError:   true,
			expectedErr: ErrInvalidDestPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.fileMapping.validateWithLogging(ctx, nil, nil)

			if tt.wantError {
				require.Error(t, err)
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidationWithCancellation(t *testing.T) {
	config := &Config{
		Version: 1,
		Source: SourceConfig{
			Repo:   "org/template",
			Branch: "main",
		},
		Targets: []TargetConfig{
			{
				Repo: "org/service",
				Files: []FileMapping{
					{Src: "file.txt", Dest: "dest.txt"},
				},
			},
		},
	}

	t.Run("canceled context in main validation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := config.ValidateWithLogging(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation canceled")
	})

	t.Run("canceled context in source validation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := config.validateSourceWithLogging(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source validation canceled")
	})

	t.Run("canceled context in defaults validation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := config.validateDefaultsWithLogging(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "defaults validation canceled")
	})

	t.Run("canceled context in target validation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		target := config.Targets[0]
		err := target.validateWithLogging(ctx, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target validation canceled")
	})

	t.Run("canceled context in file mapping validation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		fileMapping := config.Targets[0].Files[0]
		err := fileMapping.validateWithLogging(ctx, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file mapping validation canceled")
	})
}

func TestRegexValidation(t *testing.T) {
	t.Run("repository regex validation", func(t *testing.T) {
		validRepos := []string{
			"org/repo",
			"my-org/my-repo",
			"org123/repo456",
			"a/b",
			"test.org/test.repo",
			"org_name/repo_name",
		}

		for _, repo := range validRepos {
			t.Run("valid_repo_"+repo, func(t *testing.T) {
				assert.True(t, repoRegex.MatchString(repo), "Expected %s to be valid", repo)
			})
		}

		invalidRepos := []string{
			"",
			"repo",
			"/org/repo",
			"org/repo/",
			"org//repo",
			"org/",
			"/repo",
			"org repo/test",
			"org/repo test",
			"-org/repo",
			"org/-repo",
		}

		for _, repo := range invalidRepos {
			t.Run("invalid_repo_"+repo, func(t *testing.T) {
				assert.False(t, repoRegex.MatchString(repo), "Expected %s to be invalid", repo)
			})
		}
	})

	t.Run("branch regex validation", func(t *testing.T) {
		validBranches := []string{
			"main",
			"master",
			"feature/branch",
			"feature-branch",
			"v1.0.0",
			"hotfix/urgent.fix",
			"test_branch",
			"branch123",
			"a",
		}

		for _, branch := range validBranches {
			t.Run("valid_branch_"+branch, func(t *testing.T) {
				assert.True(t, branchRegex.MatchString(branch), "Expected %s to be valid", branch)
			})
		}

		invalidBranches := []string{
			"",
			"-branch",
			".branch",
			"/branch",
			"bra nch",
			"branch with spaces",
		}

		for _, branch := range invalidBranches {
			t.Run("invalid_branch_"+branch, func(t *testing.T) {
				assert.False(t, branchRegex.MatchString(branch), "Expected %s to be invalid", branch)
			})
		}
	})
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectedMsg string
	}{
		{
			name: "unsupported version error message",
			config: &Config{
				Version: 5,
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "main",
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
			expectedMsg: "unsupported config version: 5 (only version 1 is supported)",
		},
		{
			name: "invalid repo format error message",
			config: &Config{
				Version: 1,
				Source: SourceConfig{
					Repo:   "invalid-repo-format",
					Branch: "main",
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
			expectedMsg: "invalid repository format (expected: org/repo): invalid-repo-format",
		},
		{
			name: "duplicate target error message",
			config: &Config{
				Version: 1,
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "main",
				},
				Targets: []TargetConfig{
					{
						Repo: "org/duplicate",
						Files: []FileMapping{
							{Src: "file1.txt", Dest: "dest1.txt"},
						},
					},
					{
						Repo: "org/duplicate",
						Files: []FileMapping{
							{Src: "file2.txt", Dest: "dest2.txt"},
						},
					},
				},
			},
			expectedMsg: "duplicate target repository: org/duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedMsg)
		})
	}
}

// TestConfigValidateContextCancellation tests context cancellation during validation
func TestConfigValidateContextCancellation(t *testing.T) {
	// Create a config with multiple targets and files to test all cancellation points
	config := &Config{
		Version: 1,
		Source: SourceConfig{
			Repo:   "org/template",
			Branch: "main",
		},
		Defaults: DefaultConfig{
			BranchPrefix: "sync/template",
			PRLabels:     []string{"automated-sync"},
		},
		Targets: []TargetConfig{
			{
				Repo: "org/service-1",
				Files: []FileMapping{
					{Src: "file1.txt", Dest: "dest1.txt"},
					{Src: "file2.txt", Dest: "dest2.txt"},
					{Src: "file3.txt", Dest: "dest3.txt"},
				},
				Transform: Transform{
					Variables: map[string]string{
						"VAR1": "value1",
						"VAR2": "value2",
						"VAR3": "value3",
					},
				},
			},
			{
				Repo: "org/service-2",
				Files: []FileMapping{
					{Src: "file4.txt", Dest: "dest4.txt"},
					{Src: "file5.txt", Dest: "dest5.txt"},
				},
				Transform: Transform{
					Variables: map[string]string{
						"VAR4": "value4",
						"VAR5": "value5",
					},
				},
			},
			{
				Repo: "org/service-3",
				Files: []FileMapping{
					{Src: "file6.txt", Dest: "dest6.txt"},
				},
			},
		},
	}

	t.Run("context cancellation during target iteration", func(t *testing.T) {
		// Create a context that's already canceled
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := config.ValidateWithLogging(ctx, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation canceled")
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("context cancellation during file mapping validation", func(t *testing.T) {
		// Create a config that will pass initial validation but cancel during file processing
		// We can't easily control exactly when cancellation occurs, but we can test the error path
		ctx, cancel := context.WithCancel(context.Background())
		// Start validation in a goroutine and cancel after a very short time
		resultChan := make(chan error, 1)
		go func() {
			resultChan <- config.ValidateWithLogging(ctx, nil)
		}()

		// Cancel immediately to try to hit cancellation during processing
		cancel()

		err := <-resultChan
		// The error could be either cancellation or successful validation
		// depending on timing, but if it's an error, it should be cancellation-related
		if err != nil {
			errorMsg := err.Error()
			isCancellationError := assert.Contains(t, errorMsg, "validation canceled") ||
				assert.Contains(t, errorMsg, "file mapping validation canceled") ||
				assert.Contains(t, errorMsg, "context canceled")
			assert.True(t, isCancellationError, "Error should be cancellation-related: %v", err)
		}
	})

	t.Run("context timeout during validation", func(t *testing.T) {
		// Create a context with a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1) // 1 nanosecond timeout
		defer cancel()

		// Give the context time to timeout
		<-ctx.Done()

		err := config.ValidateWithLogging(ctx, nil)

		require.Error(t, err)
		errorMsg := err.Error()
		isTimeoutError := assert.Contains(t, errorMsg, "validation canceled") ||
			assert.Contains(t, errorMsg, "file mapping validation canceled") ||
			assert.Contains(t, errorMsg, "context deadline exceeded")
		assert.True(t, isTimeoutError, "Error should be timeout-related: %v", err)
	})
}

// TestConfigValidateContextCancellationEdgeCases tests specific edge cases for context cancellation
func TestConfigValidateContextCancellationEdgeCases(t *testing.T) {
	t.Run("cancellation with debug logging enabled", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Source: SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Targets: []TargetConfig{
				{
					Repo: "org/service",
					Files: []FileMapping{
						{Src: "file1.txt", Dest: "dest1.txt"},
						{Src: "file2.txt", Dest: "dest2.txt"},
					},
				},
			},
		}

		// Enable debug logging
		logConfig := &logging.LogConfig{
			Debug: logging.DebugFlags{
				Config: true,
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := config.ValidateWithLogging(ctx, logConfig)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation canceled")
	})

	t.Run("cancellation with empty targets", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Source: SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Targets: []TargetConfig{}, // Empty targets
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Should get cancellation error since context is checked early in validation
		err := config.ValidateWithLogging(ctx, nil)

		require.Error(t, err)
		// Should be cancellation error since context is checked first
		assert.Contains(t, err.Error(), "validation canceled")
	})

	t.Run("cancellation during transform variable validation", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Source: SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Targets: []TargetConfig{
				{
					Repo: "org/service",
					Files: []FileMapping{
						{Src: "file1.txt", Dest: "dest1.txt"},
					},
					Transform: Transform{
						Variables: map[string]string{
							"VAR1": "value1",
							"VAR2": "value2",
							"VAR3": "value3",
							"VAR4": "value4",
							"VAR5": "value5",
						},
					},
				},
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := config.ValidateWithLogging(ctx, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation canceled")
	})
}
