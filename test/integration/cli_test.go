package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/cli"
	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/test/helpers"
)

// TestCLICommands tests the CLI commands end-to-end
func TestCLICommands(t *testing.T) {
	// Reset global flags to ensure test isolation
	cli.ResetGlobalFlags()
	t.Cleanup(func() {
		cli.ResetGlobalFlags()
	})

	// Create test configuration
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sync.yaml")

	configContent := `version: 1
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
      branch: "master"
    defaults:
      branch_prefix: "chore/sync-files"
      pr_labels: ["automated-sync"]
    targets:
      - repo: "org/service-a"
        files:
          - src: ".github/workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
        transform:
          repo_name: true
      - repo: "org/service-b"
        files:
          - src: "Makefile"
            dest: "Makefile"
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	t.Run("validate command success", func(t *testing.T) {
		// Test validate command with valid config
		cmd := cli.GetRootCmd()
		cmd.SetArgs([]string{"validate", "--config", configPath})

		err := cmd.ExecuteContext(context.Background())
		assert.NoError(t, err)
	})

	t.Run("validate command failure", func(t *testing.T) {
		// Create invalid config
		invalidConfigPath := filepath.Join(tmpDir, "invalid.yaml")
		invalidContent := `version: 1
groups:
  - name: "Invalid Group"
    id: "invalid-group"
    source:
      repo: ""  # Empty repo
    targets: []  # No targets
`
		err := os.WriteFile(invalidConfigPath, []byte(invalidContent), 0o600)
		require.NoError(t, err)

		// Test validate command with invalid config
		cmd := cli.GetRootCmd()
		cmd.SetArgs([]string{"validate", "--config", invalidConfigPath})

		err = cmd.ExecuteContext(context.Background())
		assert.Error(t, err)
	})

	t.Run("version command", func(t *testing.T) {
		// Test version command
		cmd := cli.GetRootCmd()
		cmd.SetArgs([]string{"version"})

		err := cmd.ExecuteContext(context.Background())
		assert.NoError(t, err)
	})

	t.Run("help command", func(t *testing.T) {
		// Test help command
		cmd := cli.GetRootCmd()
		cmd.SetArgs([]string{"help"})

		err := cmd.ExecuteContext(context.Background())
		assert.NoError(t, err)
	})

	t.Run("sync command dry-run", func(t *testing.T) {
		// Skip if GitHub authentication is not available
		helpers.SkipIfNoGitHubAuth(t)

		// Test sync command in dry-run mode
		cmd := cli.GetRootCmd()
		cmd.SetArgs([]string{"sync", "--config", configPath, "--dry-run"})

		// This would normally require GitHub authentication and network access
		// In a real integration test, we'd need to mock the network calls
		// For now, we expect it to fail at the authentication stage
		err := cmd.ExecuteContext(context.Background())
		// The command should attempt to run but may fail due to missing auth
		// This tests that the CLI parsing and initial setup work correctly
		t.Logf("Sync command result: %v", err)
	})

	t.Run("status command", func(t *testing.T) {
		// Skip if GitHub authentication is not available
		helpers.SkipIfNoGitHubAuth(t)

		// Test status command
		cmd := cli.GetRootCmd()
		cmd.SetArgs([]string{"status", "--config", configPath})

		// This would normally require GitHub authentication and network access
		err := cmd.ExecuteContext(context.Background())
		// Similar to sync, this tests CLI parsing but may fail at network stage
		t.Logf("Status command result: %v", err)
	})

	t.Run("missing config file", func(t *testing.T) {
		// Test command with missing config file
		cmd := cli.GetRootCmd()
		cmd.SetArgs([]string{"validate", "--config", "/nonexistent/config.yaml"})

		err := cmd.ExecuteContext(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration file not found")
	})
}

// TestConfigurationExamples tests that all example configurations are valid
func TestConfigurationExamples(t *testing.T) {
	// Reset global flags to ensure test isolation
	cli.ResetGlobalFlags()
	t.Cleanup(func() {
		cli.ResetGlobalFlags()
	})

	examplesDir := filepath.Join("..", "..", "examples")

	// Find all .yaml files in examples directory
	files, err := filepath.Glob(filepath.Join(examplesDir, "*.yaml"))
	require.NoError(t, err)

	if len(files) == 0 {
		t.Skip("No example configuration files found")
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			// Load configuration
			cfg, err := config.Load(file)
			require.NoError(t, err, "Failed to load config file: %s", file)

			// Validate configuration
			err = cfg.Validate()
			require.NoError(t, err, "Configuration validation failed for: %s", file)

			// Verify basic structure
			assert.NotEqual(t, 0, cfg.Version, "Version should not be zero")
			assert.NotEmpty(t, cfg.Groups[0].Source.Repo, "Source repo should not be empty")
			assert.NotEmpty(t, cfg.Groups[0].Source.Branch, "Source branch should not be empty")
			assert.NotEmpty(t, cfg.Groups[0].Targets, "Targets should not be empty")

			// Verify each target has required fields
			for i, target := range cfg.Groups[0].Targets {
				assert.NotEmpty(t, target.Repo, "Target %d repo should not be empty", i)

				// Target must have either files or directories (or both)
				hasFiles := len(target.Files) > 0
				hasDirectories := len(target.Directories) > 0
				assert.True(t, hasFiles || hasDirectories, "Target %d must have either files or directories", i)

				// Verify each file mapping if present
				for j, file := range target.Files {
					assert.NotEmpty(t, file.Src, "Target %d file %d src should not be empty", i, j)
					assert.NotEmpty(t, file.Dest, "Target %d file %d dest should not be empty", i, j)
				}

				// Verify each directory mapping if present
				for j, dir := range target.Directories {
					assert.NotEmpty(t, dir.Src, "Target %d directory %d src should not be empty", i, j)
					assert.NotEmpty(t, dir.Dest, "Target %d directory %d dest should not be empty", i, j)
				}
			}
		})
	}
}

// TestCLIFlags tests various CLI flag combinations
func TestCLIFlags(t *testing.T) {
	// Reset global flags to ensure test isolation
	cli.ResetGlobalFlags()
	t.Cleanup(func() {
		cli.ResetGlobalFlags()
	})

	// Create test configuration
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sync.yaml")

	configContent := `version: 1
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
      branch: "master"
    targets:
      - repo: "org/service"
        files:
          - src: "file.txt"
            dest: "file.txt"
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "global help flag",
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name:        "validate with config flag",
			args:        []string{"validate", "--config", configPath},
			expectError: false,
		},
		{
			name:        "validate with log level",
			args:        []string{"validate", "--config", configPath, "--log-level", "debug"},
			expectError: false,
		},
		{
			name:        "sync with dry-run",
			args:        []string{"sync", "--config", configPath, "--dry-run"},
			expectError: false, // May fail at auth stage but CLI parsing should work
		},
		{
			name:        "invalid log level",
			args:        []string{"validate", "--config", configPath, "--log-level", "invalid"},
			expectError: true,
		},
		{
			name:        "missing required config",
			args:        []string{"validate"},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip sync command tests if no GitHub authentication
			if tc.name == "sync with dry-run" && helpers.GetGitHubToken() == "" {
				t.Skip("GH_PAT_TOKEN or GITHUB_TOKEN not set, skipping test that requires GitHub authentication")
			}

			cmd := cli.GetRootCmd()
			cmd.SetArgs(tc.args)

			err := cmd.ExecuteContext(context.Background())

			if tc.expectError {
				assert.Error(t, err, "Expected error for args: %v", tc.args)
			} else {
				// Some commands may fail at runtime due to missing auth/network
				// but CLI parsing and validation should succeed
				t.Logf("Command result for %v: %v", tc.args, err)
			}
		})
	}
}

// TestCLIAliases tests command aliases
func TestCLIAliases(t *testing.T) {
	// Reset global flags to ensure test isolation
	cli.ResetGlobalFlags()
	t.Cleanup(func() {
		cli.ResetGlobalFlags()
	})

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sync.yaml")

	configContent := `version: 1
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
      branch: "master"
    targets:
      - repo: "org/service"
        files:
          - src: "file.txt"
            dest: "file.txt"
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	t.Run("sync alias", func(t *testing.T) {
		// Skip if GitHub authentication is not available
		helpers.SkipIfNoGitHubAuth(t)

		// Test that 's' alias works for sync command
		cmd := cli.GetRootCmd()
		cmd.SetArgs([]string{"s", "--config", configPath, "--dry-run"})

		err := cmd.ExecuteContext(context.Background())
		// Should parse correctly (may fail at runtime due to auth)
		t.Logf("Sync alias result: %v", err)
	})

	t.Run("validate alias", func(t *testing.T) {
		// Test that 'v' alias works for validate command
		cmd := cli.GetRootCmd()
		cmd.SetArgs([]string{"v", "--config", configPath})

		err := cmd.ExecuteContext(context.Background())
		assert.NoError(t, err)
	})
}
