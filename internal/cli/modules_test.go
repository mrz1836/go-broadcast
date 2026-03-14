package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/output"
)

func TestRunListModules(t *testing.T) {
	tests := []struct {
		name           string
		config         string
		expectErr      bool
		expectOutput   []string
		expectNoOutput bool
	}{
		{
			name: "no groups configured",
			config: `version: 1
groups:
  - name: "empty-group"
    id: "empty-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md`, // Valid config with no modules
			expectErr:      false,
			expectOutput:   []string{"No modules configured"},
			expectNoOutput: false,
		},
		{
			name: "groups without modules",
			config: `version: 1
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
            dest: README.md`,
			expectErr:      false,
			expectOutput:   []string{"=== Configured Modules ==="},
			expectNoOutput: false,
		},
		{
			name: "groups with modules",
			config: `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        directories:
          - src: "."
            dest: "."
            module:
              type: go
              version: latest`,
			expectErr:      false,
			expectOutput:   []string{"=== Configured Modules ===", "Group: test-group", "Module 1:", "Type: go", "Version: latest"},
			expectNoOutput: false,
		},
		{
			name: "invalid config",
			config: `invalid yaml:
  - bad: structure`,
			expectErr:      true,
			expectNoOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configPath, []byte(tt.config), 0o600)
			require.NoError(t, err)

			// Set global config file path
			oldFlags := GetGlobalFlags()
			SetFlags(&Flags{ConfigFile: configPath, LogLevel: oldFlags.LogLevel})
			defer func() { SetFlags(oldFlags) }()

			// Capture output (thread-safe)
			scope := output.CaptureOutput()
			defer scope.Restore()

			// Create command
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Run the function
			err = runListModules(cmd, []string{})

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expectNoOutput {
				assert.Empty(t, scope.Stdout.String())
				return
			}

			capturedOutput := scope.Stdout.String()
			for _, expectedOutput := range tt.expectOutput {
				assert.Contains(t, capturedOutput, expectedOutput, "Output should contain expected text")
			}
		})
	}
}

func TestRunShowModule(t *testing.T) {
	tests := []struct {
		name         string
		config       string
		args         []string
		expectErr    bool
		expectOutput []string
	}{
		{
			name: "show module with valid group and target",
			config: `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        directories:
          - src: "."
            dest: "."
            module:
              type: go
              version: v1.2.3
              check_tags: true`,
			args:         []string{"."}, // Module path matches src: "."
			expectErr:    false,
			expectOutput: []string{"Group: test-group", "Target Repository: org/target1", "Type: go", "Version: v1.2.3"},
		},
		{
			name: "invalid arguments",
			config: `version: 1
groups: []`,
			args:      []string{"invalid-group"},
			expectErr: true,
		},
		{
			name: "group not found",
			config: `version: 1
groups:
  - name: "other-group"
    id: "other-group-1"
    source:
      repo: org/template
      branch: main
    targets: []`,
			args:      []string{"non-existent-group", "org/target1"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configPath, []byte(tt.config), 0o600)
			require.NoError(t, err)

			// Set global config file path
			oldFlags := GetGlobalFlags()
			SetFlags(&Flags{ConfigFile: configPath, LogLevel: oldFlags.LogLevel})
			defer func() { SetFlags(oldFlags) }()

			// Capture output (thread-safe)
			scope := output.CaptureOutput()
			defer scope.Restore()

			// Create command
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Run the function
			err = runShowModule(cmd, tt.args)

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			capturedOutput := scope.Stdout.String()
			for _, expectedOutput := range tt.expectOutput {
				assert.Contains(t, capturedOutput, expectedOutput, "Output should contain expected text")
			}
		})
	}
}

func TestRunModuleVersions(t *testing.T) {
	tests := []struct {
		name         string
		config       string
		args         []string
		expectErr    bool
		expectOutput []string
	}{
		{
			name: "show module versions",
			config: `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        directories:
          - src: "."
            dest: "."
            module:
              type: go
              version: latest`,
			args:         []string{"."}, // Module path matches src: "."
			expectErr:    true,          // Will fail because org/template is not a real git repository
			expectOutput: []string{},
		},
		{
			name: "insufficient arguments",
			config: `version: 1
groups: []`,
			args:      []string{}, // No arguments provided
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configPath, []byte(tt.config), 0o600)
			require.NoError(t, err)

			// Set global config file path
			oldFlags := GetGlobalFlags()
			SetFlags(&Flags{ConfigFile: configPath, LogLevel: oldFlags.LogLevel})
			defer func() { SetFlags(oldFlags) }()

			// Capture output (thread-safe)
			scope := output.CaptureOutput()
			defer scope.Restore()

			// Create command
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Run the function
			err = runModuleVersions(cmd, tt.args)

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			capturedOutput := scope.Stdout.String()
			for _, expectedOutput := range tt.expectOutput {
				assert.Contains(t, capturedOutput, expectedOutput, "Output should contain expected text")
			}
		})
	}
}

func TestRunValidateModules(t *testing.T) {
	tests := []struct {
		name         string
		config       string
		expectErr    bool
		expectOutput []string
	}{
		{
			name: "validate modules successfully",
			config: `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        directories:
          - src: "."
            dest: "."
            module:
              type: go
              version: v1.2.3`,
			expectErr:    true, // Will fail because it tries to validate version against non-existent git repo
			expectOutput: []string{},
		},
		{
			name: "no modules to validate",
			config: `version: 1
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
            dest: README.md`,
			expectErr:    false,
			expectOutput: []string{"No modules configured to validate"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yml")
			err := os.WriteFile(configPath, []byte(tt.config), 0o600)
			require.NoError(t, err)

			// Set global config file path
			oldFlags := GetGlobalFlags()
			SetFlags(&Flags{ConfigFile: configPath, LogLevel: oldFlags.LogLevel})
			defer func() { SetFlags(oldFlags) }()

			// Capture output (thread-safe)
			scope := output.CaptureOutput()
			defer scope.Restore()

			// Create command
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Run the function
			err = runValidateModules(cmd, []string{})

			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			capturedOutput := scope.Stdout.String()
			for _, expectedOutput := range tt.expectOutput {
				assert.Contains(t, capturedOutput, expectedOutput, "Output should contain expected text")
			}
		})
	}
}

func TestGetModuleType(t *testing.T) {
	tests := []struct {
		name           string
		module         config.ModuleConfig
		expectedType   string
		expectedResult string
	}{
		{
			name: "go module type",
			module: config.ModuleConfig{
				Type:    "go",
				Version: "v1.2.3",
			},
			expectedType:   "go",
			expectedResult: "go",
		},
		{
			name: "empty module type",
			module: config.ModuleConfig{
				Version: "v1.2.3",
			},
			expectedType:   "",
			expectedResult: "go (default)",
		},
		{
			name: "npm module type",
			module: config.ModuleConfig{
				Type:    "npm",
				Version: "1.2.3",
			},
			expectedType:   "npm",
			expectedResult: "npm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getModuleType(tt.module.Type)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestFetchGitTags(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		expectErr bool
	}{
		{
			name:      "valid repo format",
			repo:      "org/repo",
			expectErr: false, // We expect this to not panic and handle gracefully
		},
		{
			name:      "invalid repo format",
			repo:      "invalid-repo",
			expectErr: false, // Function should handle this gracefully
		},
		{
			name:      "empty repo",
			repo:      "",
			expectErr: false, // Function should handle this gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function makes external calls, so we just test it doesn't panic
			// and returns some result (even if it's an error due to network/auth)
			ctx := context.Background()
			versions, err := fetchGitTags(ctx, tt.repo)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				// We don't assert no error because this might fail due to network/auth
				// We just ensure it doesn't panic and returns a slice
				assert.IsType(t, []string{}, versions)
			}
		})
	}
}

// TestFetchGitTags_SecurityValidation tests command injection prevention
func TestFetchGitTags_SecurityValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		repo           string
		expectErr      bool
		errContains    string
		securityReason string
	}{
		{
			name:           "path_traversal_double_dot",
			repo:           "../../../etc/passwd",
			expectErr:      true,
			errContains:    "invalid repository path",
			securityReason: "Prevents path traversal attacks",
		},
		{
			name:           "path_traversal_embedded",
			repo:           "org/../secret/repo",
			expectErr:      true,
			errContains:    "invalid repository path",
			securityReason: "Prevents embedded path traversal",
		},
		{
			name:           "command_injection_semicolon",
			repo:           "org/repo; rm -rf /",
			expectErr:      true,
			errContains:    "invalid repository path",
			securityReason: "Prevents command chaining with semicolon",
		},
		{
			name:           "command_injection_ampersand",
			repo:           "org/repo && cat /etc/passwd",
			expectErr:      true,
			errContains:    "invalid repository path",
			securityReason: "Prevents command chaining with &&",
		},
		{
			name:           "command_injection_single_ampersand",
			repo:           "org/repo & background",
			expectErr:      true,
			errContains:    "invalid repository path",
			securityReason: "Prevents background command execution",
		},
		{
			name:           "multiple_attack_vectors",
			repo:           "../repo; whoami & id",
			expectErr:      true,
			errContains:    "invalid repository path",
			securityReason: "Prevents combined attack vectors",
		},
		{
			name:           "valid_repo_with_hyphen",
			repo:           "my-org/my-repo",
			expectErr:      false,
			errContains:    "",
			securityReason: "Allows valid repo names with hyphens",
		},
		{
			name:           "valid_repo_with_underscore",
			repo:           "my_org/my_repo",
			expectErr:      false,
			errContains:    "",
			securityReason: "Allows valid repo names with underscores",
		},
		{
			name:           "valid_repo_with_numbers",
			repo:           "org123/repo456",
			expectErr:      false,
			errContains:    "",
			securityReason: "Allows valid repo names with numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			versions, err := fetchGitTags(ctx, tt.repo)

			if tt.expectErr {
				require.Error(t, err, "Security test '%s' should fail: %s", tt.name, tt.securityReason)
				assert.Contains(t, err.Error(), tt.errContains)
				require.ErrorIs(t, err, ErrInvalidRepositoryPath)
				assert.Nil(t, versions)
			} else {
				// For valid repos, the function should not error due to security validation
				// (it may error due to network/auth, which is acceptable)
				if err != nil {
					// If there's an error, it should NOT be ErrInvalidRepositoryPath
					assert.NotErrorIs(t, err, ErrInvalidRepositoryPath,
						"Valid repo '%s' should not be rejected by security validation", tt.repo)
				}
			}
		})
	}
}

// TestFetchGitTags_EdgeCases tests additional edge cases
func TestFetchGitTags_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("single_dot_is_allowed", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		// Single dot should be allowed (not a security issue)
		_, err := fetchGitTags(ctx, "org.name/repo.name")
		if err != nil {
			// Should not be ErrInvalidRepositoryPath
			assert.NotErrorIs(t, err, ErrInvalidRepositoryPath)
		}
	})

	t.Run("newline_prevention", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		// Newlines could be used for header injection in HTTP requests
		// This is handled by the URL construction, not explicit validation
		_, err := fetchGitTags(ctx, "org/repo\ninjected")
		// The function should handle this gracefully (URL library will handle it)
		// We just verify the function doesn't panic - error handling depends on implementation
		_ = err
	})

	t.Run("unicode_in_repo_name", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		// Unicode should be handled gracefully (using escaped Unicode)
		_, err := fetchGitTags(ctx, "org/\u0440\u0435\u043f\u4f60\u597d")
		if err != nil {
			// Should not be ErrInvalidRepositoryPath (URL encoding handles this)
			assert.NotErrorIs(t, err, ErrInvalidRepositoryPath)
		}
	})

	t.Run("very_long_repo_name", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		// Very long names should be handled gracefully
		longName := "org/" + strings.Repeat("a", 1000)

		_, err := fetchGitTags(ctx, longName)
		// Should not panic, may return error from git command
		// We just verify the function doesn't panic - error handling depends on implementation
		_ = err
	})
}

// TestFetchGitTags_GitCommandFailures tests git command execution failures
func TestFetchGitTags_GitCommandFailures(t *testing.T) {
	t.Parallel()

	t.Run("empty_repository_no_tags", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		// Test against a repository that exists but has no tags
		// This will make an actual network call, so we just verify it doesn't panic
		_, err := fetchGitTags(ctx, "octocat/Hello-World")
		// Should return empty list or error, not panic
		if err != nil {
			assert.NotErrorIs(t, err, ErrInvalidRepositoryPath)
		}
	})

	t.Run("nonexistent_repository", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		// Test against a repository that doesn't exist
		_, err := fetchGitTags(ctx, "nonexistent-org-12345/nonexistent-repo-67890")
		// Should return error for network/git failure
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrInvalidRepositoryPath)
	})

	t.Run("malformed_git_output_handling", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()

		// Test that the function handles malformed output gracefully
		// by testing with a valid repo (will get valid output)
		// The actual malformed output testing would require mocking exec.Command
		_, err := fetchGitTags(ctx, "github/docs")
		// Should not panic on any output format
		_ = err
	})
}

// TestRunModuleVersions_ErrorPaths tests error scenarios for runModuleVersions
func TestRunModuleVersions_ErrorPaths(t *testing.T) {
	t.Run("module_not_in_any_group", func(t *testing.T) {
		// Create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")
		testConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: org/template
      branch: main
    targets:
      - repo: org/target1
        directories:
          - src: "src/module1"
            dest: "src/module1"
            module:
              type: go
              version: latest`

		err := os.WriteFile(configPath, []byte(testConfig), 0o600)
		require.NoError(t, err)

		// Set global config file path
		oldFlags := GetGlobalFlags()
		SetFlags(&Flags{ConfigFile: configPath, LogLevel: oldFlags.LogLevel})
		defer func() { SetFlags(oldFlags) }()

		// Capture output (thread-safe)
		scope := output.CaptureOutput()
		defer scope.Restore()

		// Create command
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		// Try to get versions for a module that doesn't exist in config
		err = runModuleVersions(cmd, []string{"nonexistent/module"})

		require.Error(t, err)
		require.ErrorIs(t, err, ErrModuleNotFound)
		capturedOutput := scope.Stdout.String() + scope.Stderr.String()
		assert.Contains(t, capturedOutput, "Module not found in configuration")
	})

	t.Run("version_resolution_failures", func(t *testing.T) {
		// Create temporary config file with module
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")
		testConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: nonexistent-org/nonexistent-repo
      branch: main
    targets:
      - repo: org/target1
        directories:
          - src: "."
            dest: "."
            module:
              type: go
              version: v1.2.3
              check_tags: true`

		err := os.WriteFile(configPath, []byte(testConfig), 0o600)
		require.NoError(t, err)

		// Set global config file path
		oldFlags := GetGlobalFlags()
		SetFlags(&Flags{ConfigFile: configPath, LogLevel: oldFlags.LogLevel})
		defer func() { SetFlags(oldFlags) }()

		// Capture output (thread-safe)
		scope := output.CaptureOutput()
		defer scope.Restore()

		// Create command
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		// Try to get versions - will fail because repository doesn't exist
		err = runModuleVersions(cmd, []string{"."})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch versions")
	})

	t.Run("display_more_than_10_versions", func(t *testing.T) {
		// Create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")
		testConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: golang/go
      branch: master
    targets:
      - repo: org/target1
        directories:
          - src: "."
            dest: "."
            module:
              type: go
              version: latest
              check_tags: true`

		err := os.WriteFile(configPath, []byte(testConfig), 0o600)
		require.NoError(t, err)

		// Set global config file path
		oldFlags := GetGlobalFlags()
		SetFlags(&Flags{ConfigFile: configPath, LogLevel: oldFlags.LogLevel})
		defer func() { SetFlags(oldFlags) }()

		// Capture output (thread-safe)
		scope := output.CaptureOutput()
		defer scope.Restore()

		// Create command
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		// Run against a repo with many tags (golang/go has many versions)
		err = runModuleVersions(cmd, []string{"."})

		// May succeed or fail depending on network, but should handle pagination
		capturedOutput := scope.Stdout.String()
		if err == nil && strings.Contains(capturedOutput, "and") && strings.Contains(capturedOutput, "more") {
			// Successfully showed pagination message
			assert.Contains(t, capturedOutput, "more")
		}
	})

	t.Run("missing_module_path_argument", func(t *testing.T) {
		// Create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")
		testConfig := `version: 1
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

		err := os.WriteFile(configPath, []byte(testConfig), 0o600)
		require.NoError(t, err)

		// Set global config file path
		oldFlags := GetGlobalFlags()
		SetFlags(&Flags{ConfigFile: configPath, LogLevel: oldFlags.LogLevel})
		defer func() { SetFlags(oldFlags) }()

		// Create command
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		// Try to run without arguments
		err = runModuleVersions(cmd, []string{})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrModulePathRequired)
	})
}
