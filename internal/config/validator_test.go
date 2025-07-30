package config

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/validation"
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
					BranchPrefix: "chore/sync-files",
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
			wantError: true,
		},
		{
			name: "invalid repository format - no slash",
			source: SourceConfig{
				Repo:   "orgtemplate",
				Branch: "main",
			},
			wantError: true,
		},
		{
			name: "invalid repository format - starts with slash",
			source: SourceConfig{
				Repo:   "/org/template",
				Branch: "main",
			},
			wantError: true,
		},
		{
			name: "invalid repository format - ends with slash",
			source: SourceConfig{
				Repo:   "org/template/",
				Branch: "main",
			},
			wantError: true,
		},
		{
			name: "invalid repository format - multiple slashes",
			source: SourceConfig{
				Repo:   "org/sub/template",
				Branch: "main",
			},
			wantError: true,
		},
		{
			name: "missing branch",
			source: SourceConfig{
				Repo:   "org/template",
				Branch: "",
			},
			wantError: true,
		},
		{
			name: "invalid branch name - starts with special character",
			source: SourceConfig{
				Repo:   "org/template",
				Branch: "-main",
			},
			wantError: true,
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
				BranchPrefix: "chore/sync-files",
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
			wantError: true,
		},
		{
			name: "empty PR label",
			defaults: DefaultConfig{
				BranchPrefix: "chore/sync-files",
				PRLabels:     []string{"automated-sync", "", "enhancement"},
			},
			wantError: true,
		},
		{
			name: "whitespace only PR label",
			defaults: DefaultConfig{
				BranchPrefix: "chore/sync-files",
				PRLabels:     []string{"automated-sync", "   ", "enhancement"},
			},
			wantError: true,
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
			wantError: true,
		},
		{
			name: "invalid repository format",
			target: TargetConfig{
				Repo: "invalid-repo",
				Files: []FileMapping{
					{Src: "src/file.txt", Dest: "dest/file.txt"},
				},
			},
			wantError: true,
		},
		{
			name: "no file mappings",
			target: TargetConfig{
				Repo:  "org/service",
				Files: []FileMapping{},
			},
			wantError: true,
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
			wantError: true,
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

// TestValidationPackageIntegration tests that the validation package functions work correctly
func TestValidationPackageIntegration(t *testing.T) {
	tests := []struct {
		name        string
		fileMapping validation.FileMapping
		wantError   bool
	}{
		{
			name: "valid file mapping",
			fileMapping: validation.FileMapping{
				Src:  "src/file.txt",
				Dest: "dest/file.txt",
			},
			wantError: false,
		},
		{
			name: "missing source path",
			fileMapping: validation.FileMapping{
				Src:  "",
				Dest: "dest/file.txt",
			},
			wantError: true,
		},
		{
			name: "missing destination path",
			fileMapping: validation.FileMapping{
				Src:  "src/file.txt",
				Dest: "",
			},
			wantError: true,
		},
		{
			name: "absolute source path",
			fileMapping: validation.FileMapping{
				Src:  "/absolute/path/file.txt",
				Dest: "dest/file.txt",
			},
			wantError: true,
		},
		{
			name: "source path with parent directory traversal",
			fileMapping: validation.FileMapping{
				Src:  "../outside/file.txt",
				Dest: "dest/file.txt",
			},
			wantError: true,
		},
		{
			name: "absolute destination path",
			fileMapping: validation.FileMapping{
				Src:  "src/file.txt",
				Dest: "/absolute/dest/file.txt",
			},
			wantError: true,
		},
		{
			name: "destination path with parent directory traversal",
			fileMapping: validation.FileMapping{
				Src:  "src/file.txt",
				Dest: "../outside/file.txt",
			},
			wantError: true,
		},
		{
			name: "complex valid paths",
			fileMapping: validation.FileMapping{
				Src:  "src/nested/deep/file.txt",
				Dest: "dest/different/structure/file.txt",
			},
			wantError: false,
		},
		{
			name: "paths with dots in filename",
			fileMapping: validation.FileMapping{
				Src:  "src/file.config.yaml",
				Dest: "dest/app.config.yaml",
			},
			wantError: false,
		},
		{
			name: "source path with complex traversal attempt",
			fileMapping: validation.FileMapping{
				Src:  "src/../../../etc/passwd",
				Dest: "dest/file.txt",
			},
			wantError: true,
		},
		{
			name: "destination path with complex traversal attempt",
			fileMapping: validation.FileMapping{
				Src:  "src/file.txt",
				Dest: "dest/../../../tmp/malicious.txt",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateFileMapping(tt.fileMapping)

			if tt.wantError {
				require.Error(t, err)
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

		// Test cancellation through target validation which includes file mapping validation
		target := config.Targets[0]
		err := target.validateWithLogging(ctx, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target validation canceled")
	})
}

func TestRegexValidation(t *testing.T) {
	t.Run("repository validation", func(t *testing.T) {
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
				err := validation.ValidateRepoName(repo)
				assert.NoError(t, err, "Expected %s to be valid", repo)
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
			t.Run("invalid_repo_"+strings.ReplaceAll(repo, "/", "_slash_"), func(t *testing.T) {
				err := validation.ValidateRepoName(repo)
				assert.Error(t, err, "Expected %s to be invalid", repo)
			})
		}
	})

	t.Run("branch validation", func(t *testing.T) {
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
				err := validation.ValidateBranchName(branch)
				assert.NoError(t, err, "Expected %s to be valid", branch)
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
			t.Run("invalid_branch_"+strings.ReplaceAll(branch, " ", "_space_"), func(t *testing.T) {
				err := validation.ValidateBranchName(branch)
				assert.Error(t, err, "Expected %s to be invalid", branch)
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
			expectedMsg: "invalid format: repository name",
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
			BranchPrefix: "chore/sync-files",
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

// TestComplexConfigurationValidationEdgeCases tests complex nested configuration scenarios
func TestComplexConfigurationValidationEdgeCases(t *testing.T) {
	t.Run("deeply nested file structures with complex transforms", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Source: SourceConfig{
				Repo:   "enterprise-org/master-template-repository",
				Branch: "production/stable/v2.1.3",
			},
			Defaults: DefaultConfig{
				BranchPrefix: "automated-sync/chore",
				PRLabels:     []string{"automated-sync", "chore", "critical-infrastructure", "security-patch"},
			},
			Targets: []TargetConfig{
				{
					Repo: "enterprise-org/microservice-payment-gateway",
					Files: []FileMapping{
						{Src: "infrastructure/kubernetes/base/deployment.yaml", Dest: "k8s/overlays/production/deployment.yaml"},
						{Src: "infrastructure/kubernetes/base/service.yaml", Dest: "k8s/overlays/production/service.yaml"},
						{Src: "infrastructure/monitoring/prometheus/rules.yaml", Dest: "monitoring/prometheus/payment-gateway-rules.yaml"},
						{Src: "infrastructure/security/network-policies.yaml", Dest: "security/network/payment-gateway-policies.yaml"},
						{Src: "docs/api/openapi-spec-template.yaml", Dest: "docs/api/payment-gateway-openapi.yaml"},
						{Src: "testing/integration/test-framework.config.json", Dest: "tests/integration/payment-gateway.config.json"},
					},
					Transform: Transform{
						RepoName: true,
						Variables: map[string]string{
							"SERVICE_NAME":        "payment-gateway",
							"NAMESPACE":           "payment-services",
							"DATABASE_CONNECTION": "postgresql://payment-db:5432/payments",
							"REDIS_ENDPOINT":      "redis://payment-cache:6379",
							"API_VERSION":         "v2",
							"SECURITY_PROFILE":    "PCI-DSS-COMPLIANT",
							"MONITORING_LABELS":   "payment,gateway,financial",
							"REPLICAS":            "3",
							"RESOURCE_LIMITS":     "cpu=2,memory=4Gi",
							"HEALTH_CHECK_PATH":   "/health/payment-gateway",
						},
					},
				},
				{
					Repo: "enterprise-org/microservice-user-authentication",
					Files: []FileMapping{
						{Src: "infrastructure/kubernetes/base/deployment.yaml", Dest: "k8s/overlays/staging/deployment.yaml"},
						{Src: "infrastructure/kubernetes/base/configmap.yaml", Dest: "k8s/overlays/staging/auth-configmap.yaml"},
						{Src: "infrastructure/security/rbac-template.yaml", Dest: "security/rbac/auth-service-rbac.yaml"},
						{Src: "infrastructure/monitoring/grafana/dashboard-template.json", Dest: "monitoring/grafana/auth-service-dashboard.json"},
					},
					Transform: Transform{
						RepoName: true,
						Variables: map[string]string{
							"SERVICE_NAME":    "user-authentication",
							"NAMESPACE":       "auth-services",
							"JWT_SECRET_KEY":  "${AUTH_JWT_SECRET}",
							"OAUTH_PROVIDERS": "google,github,microsoft",
							"SESSION_TIMEOUT": "3600",
							"RATE_LIMIT":      "1000req/min",
							"ENCRYPTION_ALGO": "AES-256-GCM",
						},
					},
				},
			},
		}

		err := config.ValidateWithLogging(context.Background(), nil)
		require.NoError(t, err, "Complex nested configuration should be valid")
	})

	t.Run("maximum complexity configuration with edge case names", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Source: SourceConfig{
				Repo:   "edge-case.org/repo_with.special-chars123",
				Branch: "feature/complex.branch-name_with_underscores",
			},
			Defaults: DefaultConfig{
				BranchPrefix: "sync.automated-updates",
				PRLabels: []string{
					"automated-sync",
					"chore",
					"infrastructure-change",
					"security-enhancement",
					"performance-optimization",
					"documentation-update",
					"ci-cd-improvement",
					"dependency-update",
					"monitoring-enhancement",
					"logging-improvement",
				},
			},
			Targets: make([]TargetConfig, 50), // Create 50 targets to test scale
		}

		// Populate targets with complex file mappings
		for i := range config.Targets {
			config.Targets[i] = TargetConfig{
				Repo: fmt.Sprintf("edge-case.org/service-%d_special.name", i+1),
				Files: []FileMapping{
					{Src: fmt.Sprintf("templates/service-%d/config.yaml", i+1), Dest: fmt.Sprintf("configs/service-%d.yaml", i+1)},
					{Src: fmt.Sprintf("templates/service-%d/deployment.yaml", i+1), Dest: fmt.Sprintf("k8s/service-%d-deployment.yaml", i+1)},
					{Src: "shared/monitoring/base.yaml", Dest: fmt.Sprintf("monitoring/service-%d-monitoring.yaml", i+1)},
				},
				Transform: Transform{
					RepoName: true,
					Variables: map[string]string{
						"SERVICE_NAME":  fmt.Sprintf("service-%d", i+1),
						"SERVICE_PORT":  fmt.Sprintf("%d", 8000+i),
						"DATABASE_NAME": fmt.Sprintf("service_%d_db", i+1),
					},
				},
			}
		}

		err := config.ValidateWithLogging(context.Background(), nil)
		require.NoError(t, err, "Maximum complexity configuration should be valid")
	})

	t.Run("error recovery with partially invalid complex configuration", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Source: SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Defaults: DefaultConfig{
				BranchPrefix: "chore/sync-files",
				PRLabels:     []string{"automated-sync", "", "valid-label"}, // Empty label in middle
			},
			Targets: []TargetConfig{
				{
					Repo: "org/valid-service-1",
					Files: []FileMapping{
						{Src: "valid/file1.yaml", Dest: "dest/file1.yaml"},
						{Src: "valid/file2.yaml", Dest: "dest/file2.yaml"},
					},
				},
				{
					Repo: "invalid-repo-format", // Invalid repo format
					Files: []FileMapping{
						{Src: "file.yaml", Dest: "dest.yaml"},
					},
				},
				{
					Repo: "org/valid-service-2",
					Files: []FileMapping{
						{Src: "../traversal-attempt", Dest: "dest.yaml"}, // Path traversal
					},
				},
				{
					Repo: "org/duplicate-dest-service",
					Files: []FileMapping{
						{Src: "src1.yaml", Dest: "same-dest.yaml"},
						{Src: "src2.yaml", Dest: "same-dest.yaml"}, // Duplicate destination
					},
				},
				{
					Repo: "org/valid-service-3",
					Files: []FileMapping{
						{Src: "valid/final.yaml", Dest: "dest/final.yaml"},
					},
					Transform: Transform{
						Variables: map[string]string{
							"VALID_VAR":   "valid-value",
							"EMPTY_VAR":   "", // Empty variable value (should be allowed)
							"COMPLEX_VAR": "${DATABASE_URL}/api/v1/endpoint?timeout=30s&retries=3",
						},
					},
				},
			},
		}

		err := config.ValidateWithLogging(context.Background(), nil)
		require.Error(t, err, "Configuration with multiple errors should fail")

		// Should catch the first error encountered (empty PR label)
		errorMsg := err.Error()
		isExpectedError := strings.Contains(errorMsg, "cannot be empty") ||
			strings.Contains(errorMsg, "invalid format: repository name") ||
			strings.Contains(errorMsg, "path traversal detected") ||
			strings.Contains(errorMsg, "duplicate destination")

		assert.True(t, isExpectedError, "Should contain one of the expected error messages: %s", errorMsg)
	})

	t.Run("repository discovery edge cases with complex naming patterns", func(t *testing.T) {
		// Test complex but valid repository naming patterns
		validComplexRepos := []string{
			"enterprise.corporation/microservice.payment-gateway_v2",
			"github-org123/repo-name_with.multiple-separators",
			"a1b2c3.org/x9y8z7.repo",
			"org_with_underscores/repo.with.dots-and-hyphens_123",
			"12345.numbers/67890.in.names",
		}

		for i, repo := range validComplexRepos {
			t.Run(fmt.Sprintf("valid_complex_repo_%d", i+1), func(t *testing.T) {
				config := &Config{
					Version: 1,
					Source: SourceConfig{
						Repo:   repo,
						Branch: "main",
					},
					Targets: []TargetConfig{
						{
							Repo: "org/target",
							Files: []FileMapping{
								{Src: "file.txt", Dest: "dest.txt"},
							},
						},
					},
				}

				err := config.ValidateWithLogging(context.Background(), nil)
				assert.NoError(t, err, "Complex but valid repository name should pass: %s", repo)
			})
		}

		// Test edge cases that should fail
		invalidComplexRepos := []struct {
			repo   string
			reason string
		}{
			{"org/repo/extra/parts", "too many slashes"},
			{"org//double-slash", "double slash"},
			{"org/", "empty repo name"},
			{"/repo", "empty org name"},
			{"_org/repo", "org starts with underscore"},
			{"org/_repo", "repo starts with underscore"},
			{"org with spaces/repo", "org contains spaces"},
			{"org/repo with spaces", "repo contains spaces"},
		}

		for i, testCase := range invalidComplexRepos {
			t.Run(fmt.Sprintf("invalid_complex_repo_%d_%s", i+1, strings.ReplaceAll(testCase.reason, " ", "_")), func(t *testing.T) {
				config := &Config{
					Version: 1,
					Source: SourceConfig{
						Repo:   testCase.repo,
						Branch: "main",
					},
					Targets: []TargetConfig{
						{
							Repo: "org/target",
							Files: []FileMapping{
								{Src: "file.txt", Dest: "dest.txt"},
							},
						},
					},
				}

				err := config.ValidateWithLogging(context.Background(), nil)
				require.Error(t, err, "Invalid repository name should fail (%s): %s", testCase.reason, testCase.repo)
				assert.Contains(t, err.Error(), "repository name", "Error should mention repository name validation")
			})
		}
	})

	t.Run("complex branch name validation edge cases", func(t *testing.T) {
		// Test complex but valid branch names
		validComplexBranches := []string{
			"feature/JIRA-12345_implement.complex-feature_v2.1.0",
			"hotfix/urgent.security-patch_CVE-2023-12345",
			"release/v1.2.3-beta.4_pre-release.candidate",
			"maintenance/database.migration_phase-1_rollback.safe",
			"experiment/ai-ml.model_training-v3.2_gpu.optimization",
			"integration/third-party.api_v2.auth.oauth2-implementation",
		}

		for i, branch := range validComplexBranches {
			t.Run(fmt.Sprintf("valid_complex_branch_%d", i+1), func(t *testing.T) {
				config := &Config{
					Version: 1,
					Source: SourceConfig{
						Repo:   "org/template",
						Branch: branch,
					},
					Targets: []TargetConfig{
						{
							Repo: "org/target",
							Files: []FileMapping{
								{Src: "file.txt", Dest: "dest.txt"},
							},
						},
					},
				}

				err := config.ValidateWithLogging(context.Background(), nil)
				assert.NoError(t, err, "Complex but valid branch name should pass: %s", branch)
			})
		}

		// Test invalid complex branch names
		invalidComplexBranches := []struct {
			branch string
			reason string
		}{
			{"-feature/invalid-start", "starts with hyphen"},
			{".feature/invalid-start", "starts with dot"},
			{"/feature/invalid-start", "starts with slash"},
			{"feature branch with spaces", "contains spaces"},
			{"feature@branch#invalid", "contains special characters"},
			{"feature\\branch\\backslashes", "contains backslashes"},
			{"feature|branch|pipes", "contains pipes"},
			{"feature<branch>brackets", "contains angle brackets"},
		}

		for i, testCase := range invalidComplexBranches {
			t.Run(fmt.Sprintf("invalid_complex_branch_%d_%s", i+1, strings.ReplaceAll(testCase.reason, " ", "_")), func(t *testing.T) {
				config := &Config{
					Version: 1,
					Source: SourceConfig{
						Repo:   "org/template",
						Branch: testCase.branch,
					},
					Targets: []TargetConfig{
						{
							Repo: "org/target",
							Files: []FileMapping{
								{Src: "file.txt", Dest: "dest.txt"},
							},
						},
					},
				}

				err := config.ValidateWithLogging(context.Background(), nil)
				require.Error(t, err, "Invalid branch name should fail (%s): %s", testCase.reason, testCase.branch)
				assert.Contains(t, err.Error(), "branch name", "Error should mention branch name validation")
			})
		}
	})

	t.Run("extreme file path validation scenarios", func(t *testing.T) {
		// Test complex but valid file paths
		validComplexPaths := []struct {
			src  string
			dest string
		}{
			{
				"infrastructure/kubernetes/overlays/production/microservices/payment-gateway/deployment.v2.1.3.yaml",
				"k8s/production/services/payment/gateway-deployment.yaml",
			},
			{
				"docs/api-specifications/openapi/v3.0/payment-service/endpoints.swagger.json",
				"documentation/api/payment-service-openapi-v3.json",
			},
			{
				"testing/integration/data/fixtures/payment.gateway.test-data.large-volume.json",
				"tests/integration/fixtures/payment-gateway-bulk-data.json",
			},
			{
				"configuration/environments/production/secrets-template.encrypted.yaml",
				"config/prod/secrets-template.yaml",
			},
			{
				"monitoring/prometheus/rules/alerts/payment-gateway.high-availability.rules.yaml",
				"monitoring/alerts/payment-gateway-ha.yaml",
			},
		}

		for i, pathPair := range validComplexPaths {
			t.Run(fmt.Sprintf("valid_complex_paths_%d", i+1), func(t *testing.T) {
				config := &Config{
					Version: 1,
					Source: SourceConfig{
						Repo:   "org/template",
						Branch: "main",
					},
					Targets: []TargetConfig{
						{
							Repo: "org/target",
							Files: []FileMapping{
								{Src: pathPair.src, Dest: pathPair.dest},
							},
						},
					},
				}

				err := config.ValidateWithLogging(context.Background(), nil)
				assert.NoError(t, err, "Complex but valid file paths should pass: %s -> %s", pathPair.src, pathPair.dest)
			})
		}

		// Test sophisticated path traversal attempts
		sophisticatedTraversalAttempts := []struct {
			src    string
			dest   string
			reason string
		}{
			{
				"legitimate/path/../../../../../../../etc/passwd",
				"dest/file.txt",
				"multiple parent directory traversal in source",
			},
			{
				"src/file.txt",
				"legitimate/path/../../../../../../../tmp/malicious.sh",
				"multiple parent directory traversal in destination",
			},
			{
				"src/normal/../../../secret.file",
				"dest/file.txt",
				"traversal mixed with legitimate path in source",
			},
			{
				"src/file.txt",
				"dest/normal/../../../sensitive.data",
				"traversal mixed with legitimate path in destination",
			},
			{
				"configs/../configs/../configs/../../../root/.ssh/id_rsa",
				"dest/file.txt",
				"repeated directory traversal pattern in source",
			},
			{
				"src/file.txt",
				"data/../data/../data/../../../var/log/sensitive.log",
				"repeated directory traversal pattern in destination",
			},
		}

		for i, testCase := range sophisticatedTraversalAttempts {
			t.Run(fmt.Sprintf("sophisticated_traversal_%d", i+1), func(t *testing.T) {
				config := &Config{
					Version: 1,
					Source: SourceConfig{
						Repo:   "org/template",
						Branch: "main",
					},
					Targets: []TargetConfig{
						{
							Repo: "org/target",
							Files: []FileMapping{
								{Src: testCase.src, Dest: testCase.dest},
							},
						},
					},
				}

				err := config.ValidateWithLogging(context.Background(), nil)
				require.Error(t, err, "Sophisticated path traversal should be detected (%s)", testCase.reason)
				assert.Contains(t, err.Error(), "path traversal detected", "Error should mention path traversal detection")
			})
		}
	})

	t.Run("transform variable validation with complex scenarios", func(t *testing.T) {
		// Test complex but valid transform variables
		complexValidVariables := map[string]string{
			"SIMPLE_VAR":                   "simple-value",
			"EMPTY_VAR":                    "", // Empty values should be allowed
			"URL_VAR":                      "https://api.example.com/v1/endpoint?timeout=30s&retries=3",
			"DATABASE_CONNECTION_STRING":   "postgresql://user:password@hostname:5432/database?sslmode=require&pool_max_conns=10",
			"KUBERNETES_RESOURCE_TEMPLATE": "memory: ${MEMORY_LIMIT}Mi, cpu: ${CPU_LIMIT}m, replicas: ${REPLICA_COUNT}",
			"JSON_CONFIG":                  `{"timeout": 30, "retries": 3, "endpoints": ["api1", "api2"]}`,
			"YAML_FRAGMENT":                "key: value\nnested:\n  key: value",
			"ENVIRONMENT_SPECIFIC":         "${ENVIRONMENT}_${SERVICE_NAME}_${VERSION}",
			"MONITORING_LABELS":            "app=${APP_NAME},version=${VERSION},environment=${ENV},team=platform",
			"SECURITY_POLICY":              "network-policy: default-deny, ingress: allow-from-namespace, egress: allow-to-internet",
			"PERFORMANCE_TUNING":           "gc-percent=100,max-procs=${CPU_COUNT},gomaxprocs=${GOMAXPROCS}",
			"FEATURE_FLAGS":                "flag1=true,flag2=false,flag3=${EXPERIMENTAL_FEATURES}",
			"API_VERSIONS":                 "v1=/api/v1,v2=/api/v2,health=/health,metrics=/metrics",
			"REGEX_PATTERN":                "^[a-zA-Z0-9]([a-zA-Z0-9-_.])*[a-zA-Z0-9]$",
			"SPECIAL_CHARACTERS":           "!@#$%^&*()_+-=[]{}|;':,.<>?",
			"UNICODE_CONTENT":              "Service: 💳 Payment Gateway, Status: ✅ Active, Performance: 🚀 High",
			"MULTI_LINE_SCRIPT":            "#!/bin/bash\nset -e\necho 'Starting service'\n./start-service.sh",
			"BASE64_ENCODED":               "dGVzdC1kYXRhLWZvci12YWxpZGF0aW9uLXB1cnBvc2Vz",
			"XML_FRAGMENT":                 "<config><database>prod-db</database><timeout>30</timeout></config>",
			"SQL_QUERY_TEMPLATE":           "SELECT * FROM ${TABLE_NAME} WHERE environment = '${ENVIRONMENT}' AND active = true",
		}

		config := &Config{
			Version: 1,
			Source: SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Targets: []TargetConfig{
				{
					Repo: "org/target",
					Files: []FileMapping{
						{Src: "template.yaml", Dest: "output.yaml"},
					},
					Transform: Transform{
						RepoName:  true,
						Variables: complexValidVariables,
					},
				},
			},
		}

		err := config.ValidateWithLogging(context.Background(), nil)
		assert.NoError(t, err, "Complex but valid transform variables should pass")
	})

	t.Run("validation performance with large configuration", func(t *testing.T) {
		// Create a configuration with many targets to test validation performance
		config := &Config{
			Version: 1,
			Source: SourceConfig{
				Repo:   "org/template",
				Branch: "main",
			},
			Defaults: DefaultConfig{
				BranchPrefix: "chore/sync-files",
				PRLabels:     []string{"automated-sync", "performance-test"},
			},
			Targets: make([]TargetConfig, 100), // 100 targets
		}

		// Populate targets with unique configurations
		for i := range config.Targets {
			config.Targets[i] = TargetConfig{
				Repo: fmt.Sprintf("org/service-%03d", i+1),
				Files: []FileMapping{
					{Src: fmt.Sprintf("templates/service-%d.yaml", i+1), Dest: fmt.Sprintf("services/service-%d.yaml", i+1)},
					{Src: fmt.Sprintf("configs/service-%d.json", i+1), Dest: fmt.Sprintf("configs/service-%d-config.json", i+1)},
					{Src: "shared/monitoring.yaml", Dest: fmt.Sprintf("monitoring/service-%d-monitoring.yaml", i+1)},
					{Src: "shared/security.yaml", Dest: fmt.Sprintf("security/service-%d-security.yaml", i+1)},
					{Src: "shared/deployment.yaml", Dest: fmt.Sprintf("k8s/service-%d-deployment.yaml", i+1)},
				},
				Transform: Transform{
					RepoName: true,
					Variables: map[string]string{
						"SERVICE_NAME":  fmt.Sprintf("service-%d", i+1),
						"SERVICE_PORT":  fmt.Sprintf("%d", 8000+i),
						"SERVICE_INDEX": fmt.Sprintf("%d", i+1),
						"DATABASE_NAME": fmt.Sprintf("service_%d_db", i+1),
						"NAMESPACE":     fmt.Sprintf("services-group-%d", (i/10)+1),
					},
				},
			}
		}

		// Measure validation time
		start := time.Now()
		err := config.ValidateWithLogging(context.Background(), nil)
		duration := time.Since(start)

		require.NoError(t, err, "Large configuration should validate successfully")
		assert.Less(t, duration, 5*time.Second, "Validation should complete within reasonable time")
		t.Logf("Validation of 100 targets with 500 files completed in %v", duration)
	})
}
