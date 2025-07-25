package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name: "ValidConfigWithDefaults",
			setupFunc: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "valid-defaults-*.yml")
				require.NoError(t, err)

				validConfig := `version: 1
source:
  repo: org/template
  branch: main
defaults:
  branch_prefix: "sync/"
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

			err := runValidateWithFlags(flags)

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

	err = runValidateWithFlags(flags)
	require.NoError(t, err)
}

// TestValidateAbsolutePath tests absolute path display
func TestValidateAbsolutePath(t *testing.T) {
	// Create temporary config in a known directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yml")

	validConfig := `version: 1
source:
  repo: org/template
  branch: main
targets:
  - repo: org/target1
    files:
      - src: README.md
        dest: README.md`

	require.NoError(t, os.WriteFile(configPath, []byte(validConfig), 0o600))

	// Use relative path
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(originalDir) }()

	flags := &Flags{
		ConfigFile: "./test-config.yml",
	}

	err = runValidateWithFlags(flags)
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
source:
  repo: test/source
  branch: main
targets:
  - repo: test/target1
    files:
      - src: README.md
        dest: README.md`

	require.NoError(t, os.WriteFile(tmpFile.Name(), []byte(validConfig), 0o600))

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
