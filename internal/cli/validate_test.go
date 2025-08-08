package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// TestValidateCmd tests validate command configuration
func TestValidateCmd(t *testing.T) {
	cmd := validateCmd

	assert.Equal(t, "validate", cmd.Use)
	assert.Equal(t, "Validate configuration file", cmd.Short)
	assert.Contains(t, cmd.Long, "YAML syntax is valid")
	assert.Contains(t, cmd.Example, "go-broadcast validate")
	assert.Contains(t, cmd.Aliases, "v")
	assert.Contains(t, cmd.Aliases, "check")
	assert.NotNil(t, cmd.RunE)
}

// TestRunValidate tests the main validate command execution
func TestRunValidate(t *testing.T) {
	t.Run("ConfigNotFound", func(t *testing.T) {
		// Save original config
		originalFlags := globalFlags
		globalFlags = &Flags{
			ConfigFile: "/non/existent/config.yml",
		}
		defer func() {
			globalFlags = originalFlags
		}()

		cmd := &cobra.Command{}
		err := runValidate(cmd, []string{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrConfigFileNotFound)
	})

	t.Run("ValidConfig", func(t *testing.T) {
		// Create temporary valid config
		tmpFile, err := os.CreateTemp("", "config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md`

		_, err = tmpFile.WriteString(validConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Save original flags
		originalFlags := globalFlags
		globalFlags = &Flags{
			ConfigFile: tmpFile.Name(),
		}
		defer func() {
			globalFlags = originalFlags
		}()

		cmd := &cobra.Command{}
		err = runValidate(cmd, []string{})
		require.NoError(t, err)
	})
}

// TestRunValidateWithFlags tests validate with flags parameter
func TestRunValidateWithFlags(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func() (string, func())
		expectError bool
		errorCheck  func(*testing.T, error)
		outputCheck func(*testing.T, string)
	}{
		{
			name: "FileNotFound",
			setupFunc: func() (string, func()) {
				return "/non/existent/config.yml", func() {}
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, ErrConfigFileNotFound)
			},
		},
		{
			name: "InvalidYAML",
			setupFunc: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "invalid-*.yml")
				require.NoError(t, err)

				invalidYAML := `invalid: yaml: content:`
				_, err = tmpFile.WriteString(invalidYAML)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "configuration parsing failed")
			},
		},
		{
			name: "InvalidConfig",
			setupFunc: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "invalid-config-*.yml")
				require.NoError(t, err)

				// Valid YAML but invalid config (missing required fields)
				invalidConfig := `version: 1
source:
  repo: org/template
targets:
  - name: target1`

				_, err = tmpFile.WriteString(invalidConfig)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "configuration parsing failed")
			},
		},
		{
			name: "ValidConfigMinimal",
			setupFunc: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "valid-minimal-*.yml")
				require.NoError(t, err)

				validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md`

				_, err = tmpFile.WriteString(validConfig)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			expectError: false,
			outputCheck: nil,
		},
		{
			name: "ValidConfigWithGlobal",
			setupFunc: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "valid-global-*.yml")
				require.NoError(t, err)

				validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    global:
      pr_labels:
        - "automated"
        - "sync"
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md`

				_, err = tmpFile.WriteString(validConfig)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			expectError: false,
			outputCheck: nil,
		},
		{
			name: "ValidConfigWithTransforms",
			setupFunc: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "valid-transforms-*.yml")
				require.NoError(t, err)

				validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md
        transform:
          repo_name: true
          variables:
            PROJECT: "test-project"
            VERSION: "1.0.0"`

				_, err = tmpFile.WriteString(validConfig)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			expectError: false,
			outputCheck: nil,
		},
		{
			name: "ValidConfigMultipleTargets",
			setupFunc: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "valid-multi-*.yml")
				require.NoError(t, err)

				validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md
          - src: .github/workflows/ci.yml
            dest: .github/workflows/ci.yml
      - repo: org/target2
        files:
          - src: README.md
            dest: docs/README.md
      - repo: org/target3
        files:
          - src: README.md
            dest: README.md`

				_, err = tmpFile.WriteString(validConfig)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			expectError: false,
			outputCheck: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configPath, cleanup := tc.setupFunc()
			defer cleanup()

			flags := &Flags{
				ConfigFile: configPath,
			}

			// Create a mock command with flags
			cmd := &cobra.Command{}
			cmd.Flags().Bool("skip-remote-checks", false, "Skip remote checks")
			cmd.Flags().Bool("source-only", false, "Source only")

			err := runValidateWithFlags(flags, cmd)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorCheck != nil {
					tc.errorCheck(t, err)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidateOutputFormatting tests the output formatting
func TestValidateOutputFormatting(t *testing.T) {
	// Create temporary valid config
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md`

	_, err = tmpFile.WriteString(validConfig)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	flags := &Flags{
		ConfigFile: tmpFile.Name(),
	}

	// Create a mock command with flags
	cmd := &cobra.Command{}
	cmd.Flags().Bool("skip-remote-checks", false, "Skip remote checks")
	cmd.Flags().Bool("source-only", false, "Source only")

	err = runValidateWithFlags(flags, cmd)
	require.NoError(t, err)
}

// TestValidateAbsolutePath tests absolute path display
func TestValidateAbsolutePath(t *testing.T) {
	// Create temporary config in a known directory
	tmpDir := testutil.CreateTempDir(t)
	configPath := filepath.Join(tmpDir, "test-config.yml")

	validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md`

	testutil.WriteTestFile(t, configPath, validConfig)

	// Use relative path
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(originalDir) }()

	flags := &Flags{
		ConfigFile: "./test-config.yml",
	}

	// Create a mock command with flags
	cmd := &cobra.Command{}
	cmd.Flags().Bool("skip-remote-checks", false, "Skip remote checks")
	cmd.Flags().Bool("source-only", false, "Source only")

	err = runValidateWithFlags(flags, cmd)
	require.NoError(t, err)
}

// TestValidateCommandIntegration tests validate command as configured
func TestValidateCommandIntegration(t *testing.T) {
	// Test that command is properly wired
	cmd := validateCmd
	assert.NotNil(t, cmd.RunE)

	// Create a valid config
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: test/source
      branch: main
    targets:
      - repo: test/target1
        files:
          - src: README.md
            dest: README.md`

	testutil.WriteTestFile(t, tmpFile.Name(), validConfig)

	// Save original config
	originalFlags := globalFlags
	globalFlags = &Flags{
		ConfigFile: tmpFile.Name(),
	}
	defer func() {
		globalFlags = originalFlags
	}()

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err)
}

// TestValidateRepositoryAccessibilityPanicRecovery tests panic handling
func TestValidateRepositoryAccessibilityPanicRecovery(t *testing.T) {
	t.Run("nil config causes panic", func(t *testing.T) {
		// Test that validateRepositoryAccessibility panics with nil config
		ctx := context.Background()
		logConfig := &logging.LogConfig{LogLevel: "error"}

		require.Panics(t, func() {
			_ = validateRepositoryAccessibility(ctx, nil, logConfig, false)
		})
	})
}

// TestValidateSourceFilesExistGracefulHandling tests graceful error handling
func TestValidateSourceFilesExistGracefulHandling(t *testing.T) {
	t.Run("handles github client creation failure gracefully", func(t *testing.T) {
		ctx := context.Background()
		logConfig := &logging.LogConfig{LogLevel: "error"}
		cfg := &config.Config{
			Groups: []config.Group{{
				Name: "test-group",
				ID:   "test-group-1",
				Source: config.SourceConfig{
					Repo:   "test/repo",
					Branch: "main",
				},
			}},
		}

		// validateSourceFilesExist should handle GitHub client errors gracefully
		// and not panic (it's designed to be non-fatal)
		require.NotPanics(t, func() {
			validateSourceFilesExist(ctx, cfg, logConfig)
		})
	})
}

// TestValidateWithFlags tests runValidateWithFlags edge cases
func TestValidateWithFlagsEdgeCases(t *testing.T) {
	t.Run("skip remote checks flag", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md`

		_, err = tmpFile.WriteString(validConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		flags := &Flags{
			ConfigFile: tmpFile.Name(),
		}

		cmd := &cobra.Command{}
		cmd.Flags().Bool("skip-remote-checks", true, "Skip remote checks")
		cmd.Flags().Bool("source-only", false, "Source only")

		// This should skip the remote checks and succeed
		err = runValidateWithFlags(flags, cmd)
		require.NoError(t, err)
	})

	t.Run("source only flag", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md`

		_, err = tmpFile.WriteString(validConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		flags := &Flags{
			ConfigFile: tmpFile.Name(),
		}

		cmd := &cobra.Command{}
		cmd.Flags().Bool("skip-remote-checks", false, "Skip remote checks")
		cmd.Flags().Bool("source-only", true, "Source only")

		// This should validate only source and succeed
		err = runValidateWithFlags(flags, cmd)
		require.NoError(t, err)
	})
}

// TestDisplayGroupValidation tests the displayGroupValidation function
func TestDisplayGroupValidation(t *testing.T) {
	tests := []struct {
		name           string
		groups         []config.Group
		expectedOutput []string
	}{
		{
			name: "simple groups without dependencies",
			groups: []config.Group{
				{
					Name:     "group1",
					ID:       "group-1",
					Priority: 0,
					Source: config.SourceConfig{
						Repo:   "org/template",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "org/target1"},
					},
				},
				{
					Name:     "group2",
					ID:       "group-2",
					Priority: 1,
					Source: config.SourceConfig{
						Repo:   "org/template2",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "org/target2"},
					},
				},
			},
			expectedOutput: []string{
				"Groups: 2 configured",
				"✓ No circular dependencies detected",
				"Group 1: group1 (group-1)",
				"Priority: 0",
				"Group 2: group2 (group-2)",
				"Priority: 1",
				"✓ All groups have unique priorities",
			},
		},
		{
			name: "groups with circular dependencies",
			groups: []config.Group{
				{
					Name:      "group1",
					ID:        "group-1",
					Priority:  0,
					DependsOn: []string{"group-2"},
					Source: config.SourceConfig{
						Repo:   "org/template",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "org/target1"},
					},
				},
				{
					Name:      "group2",
					ID:        "group-2",
					Priority:  1,
					DependsOn: []string{"group-1"},
					Source: config.SourceConfig{
						Repo:   "org/template2",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "org/target2"},
					},
				},
			},
			expectedOutput: []string{
				"Groups: 2 configured",
				"✗ Circular dependency detected for group: group-1",
				"✗ Circular dependency detected for group: group-2",
			},
		},
		{
			name:   "empty groups",
			groups: []config.Group{},
			expectedOutput: []string{
				"Groups: 0 configured",
				"✓ No circular dependencies detected",
				"✓ All groups have unique priorities",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture both stdout and stderr
			var stdoutBuf, stderrBuf bytes.Buffer
			output.SetStdout(&stdoutBuf)
			output.SetStderr(&stderrBuf)
			defer output.SetStdout(os.Stdout)
			defer output.SetStderr(os.Stderr)

			displayGroupValidation(tt.groups)

			// Combine both outputs for testing
			capturedOutput := stdoutBuf.String() + stderrBuf.String()
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, capturedOutput, expected, "Output should contain expected text")
			}
		})
	}
}

// TestCheckCircularDependency tests the checkCircularDependency function
func TestCheckCircularDependency(t *testing.T) {
	tests := []struct {
		name           string
		groupID        string
		dependencyMap  map[string][]string
		expectCircular bool
	}{
		{
			name:    "no circular dependency",
			groupID: "group-1",
			dependencyMap: map[string][]string{
				"group-1": {"group-2"},
				"group-2": {"group-3"},
				"group-3": {},
			},
			expectCircular: false,
		},
		{
			name:    "direct circular dependency",
			groupID: "group-1",
			dependencyMap: map[string][]string{
				"group-1": {"group-2"},
				"group-2": {"group-1"},
			},
			expectCircular: true,
		},
		{
			name:    "indirect circular dependency",
			groupID: "group-1",
			dependencyMap: map[string][]string{
				"group-1": {"group-2"},
				"group-2": {"group-3"},
				"group-3": {"group-1"},
			},
			expectCircular: true,
		},
		{
			name:    "self dependency",
			groupID: "group-1",
			dependencyMap: map[string][]string{
				"group-1": {"group-1"},
			},
			expectCircular: true,
		},
		{
			name:    "no dependencies",
			groupID: "group-1",
			dependencyMap: map[string][]string{
				"group-1": {},
			},
			expectCircular: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visited := make(map[string]bool)
			result := checkCircularDependency(tt.groupID, tt.dependencyMap, visited)
			assert.Equal(t, tt.expectCircular, result)
		})
	}
}
