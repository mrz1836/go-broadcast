package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCmd_ShowChecks(t *testing.T) {
	// Save original
	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
	}()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run with show-checks flag
	os.Args = []string{"gofortress-pre-commit", "run", "--show-checks"}

	// Execute command
	rootCmd.SetArgs([]string{"run", "--show-checks"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should show available checks
	assert.Contains(t, output, "Available checks:")
	assert.Contains(t, output, "fumpt")
	assert.Contains(t, output, "lint")
	assert.Contains(t, output, "whitespace")
	assert.Contains(t, output, "eof")
	assert.Contains(t, output, "mod-tidy")
}

func TestRunCmd_DisabledSystem(t *testing.T) {
	// Save original env
	oldEnv := os.Getenv("ENABLE_PRE_COMMIT_SYSTEM")
	defer func() {
		if err := os.Setenv("ENABLE_PRE_COMMIT_SYSTEM", oldEnv); err != nil {
			t.Logf("Failed to restore ENABLE_PRE_COMMIT_SYSTEM: %v", err)
		}
	}()

	// Disable pre-commit system
	require.NoError(t, os.Setenv("ENABLE_PRE_COMMIT_SYSTEM", "false"))

	// Save original
	oldArgs := os.Args
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stderr = oldStderr
	}()

	// Capture stderr output since printWarning outputs to stderr when noColor is true
	r, w, _ := os.Pipe()
	os.Stderr = w
	noColor = true // Ensure we output to stderr

	// Execute command
	rootCmd.SetArgs([]string{"run"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should show warning about disabled system
	assert.Contains(t, output, "Pre-commit system is disabled")
}

func TestRunCmd_ParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T)
	}{
		{
			name: "all-files flag",
			args: []string{"run", "--all-files"},
			validate: func(t *testing.T) {
				assert.True(t, allFiles)
			},
		},
		{
			name: "files flag",
			args: []string{"run", "--files", "main.go,utils.go"},
			validate: func(t *testing.T) {
				assert.Equal(t, []string{"main.go", "utils.go"}, files)
			},
		},
		{
			name: "skip flag",
			args: []string{"run", "--skip", "lint,fumpt"},
			validate: func(t *testing.T) {
				assert.Equal(t, []string{"lint", "fumpt"}, skipChecks)
			},
		},
		{
			name: "only flag",
			args: []string{"run", "--only", "whitespace,eof"},
			validate: func(t *testing.T) {
				assert.Equal(t, []string{"whitespace", "eof"}, onlyChecks)
			},
		},
		{
			name: "parallel flag",
			args: []string{"run", "--parallel", "4"},
			validate: func(t *testing.T) {
				assert.Equal(t, 4, parallel)
			},
		},
		{
			name: "fail-fast flag",
			args: []string{"run", "--fail-fast"},
			validate: func(t *testing.T) {
				assert.True(t, failFast)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			allFiles = false
			files = nil
			skipChecks = nil
			onlyChecks = nil
			parallel = 0
			failFast = false

			// Parse command properly through execute to handle subcommand flags
			rootCmd.SetArgs(tt.args)
			cmd, err := rootCmd.ExecuteC()
			if err != nil {
				// For testing flag parsing, we expect execution errors but not parse errors
				// Since we can't actually run without proper git repo setup
				require.Contains(t, err.Error(), "failed to")
			}
			assert.Equal(t, "run", cmd.Name())

			// Validate
			tt.validate(t)
		})
	}
}

func TestRunCmd_SpecificCheck(t *testing.T) {
	// Save original
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Execute command with specific check
	rootCmd.SetArgs([]string{"run", "whitespace"})

	// This would fail in test environment as we're not in a git repo
	// but we can verify the command structure is correct
	cmd, _, err := rootCmd.Find([]string{"run", "whitespace"})
	require.NoError(t, err)
	assert.Equal(t, "run", cmd.Name())
}
