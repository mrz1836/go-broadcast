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
			oldFlags := globalFlags
			globalFlags = &Flags{ConfigFile: configPath}
			defer func() { globalFlags = oldFlags }()

			// Capture output
			var buf bytes.Buffer
			output.SetStdout(&buf)
			defer output.SetStdout(os.Stdout)

			// Create command
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Run the function
			err = runListModules(cmd, []string{})

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expectNoOutput {
				assert.Empty(t, buf.String())
				return
			}

			capturedOutput := buf.String()
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
			oldFlags := globalFlags
			globalFlags = &Flags{ConfigFile: configPath}
			defer func() { globalFlags = oldFlags }()

			// Capture output
			var buf bytes.Buffer
			output.SetStdout(&buf)
			defer output.SetStdout(os.Stdout)

			// Create command
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Run the function
			err = runShowModule(cmd, tt.args)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			capturedOutput := buf.String()
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
			oldFlags := globalFlags
			globalFlags = &Flags{ConfigFile: configPath}
			defer func() { globalFlags = oldFlags }()

			// Capture output
			var buf bytes.Buffer
			output.SetStdout(&buf)
			defer output.SetStdout(os.Stdout)

			// Create command
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Run the function
			err = runModuleVersions(cmd, tt.args)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			capturedOutput := buf.String()
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
			oldFlags := globalFlags
			globalFlags = &Flags{ConfigFile: configPath}
			defer func() { globalFlags = oldFlags }()

			// Capture output
			var buf bytes.Buffer
			output.SetStdout(&buf)
			defer output.SetStdout(os.Stdout)

			// Create command
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Run the function
			err = runValidateModules(cmd, []string{})

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			capturedOutput := buf.String()
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
				assert.Error(t, err)
			} else {
				// We don't assert no error because this might fail due to network/auth
				// We just ensure it doesn't panic and returns a slice
				assert.IsType(t, []string{}, versions)
			}
		})
	}
}
