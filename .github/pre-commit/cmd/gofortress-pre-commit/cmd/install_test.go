package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallCmd_ParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T)
	}{
		{
			name: "force flag",
			args: []string{"install", "--force"},
			validate: func(t *testing.T) {
				assert.True(t, force)
			},
		},
		{
			name: "force flag short",
			args: []string{"install", "-f"},
			validate: func(t *testing.T) {
				assert.True(t, force)
			},
		},
		{
			name: "hook-type flag single",
			args: []string{"install", "--hook-type", "pre-push"},
			validate: func(t *testing.T) {
				assert.Equal(t, []string{"pre-push"}, hookTypes)
			},
		},
		{
			name: "hook-type flag multiple",
			args: []string{"install", "--hook-type", "pre-commit", "--hook-type", "pre-push"},
			validate: func(t *testing.T) {
				assert.Equal(t, []string{"pre-commit", "pre-push"}, hookTypes)
			},
		},
		{
			name: "default hook type",
			args: []string{"install"},
			validate: func(t *testing.T) {
				assert.Equal(t, []string{"pre-commit"}, hookTypes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			force = false
			// For tests with explicit hook-type flags, start with empty slice
			// For default test, use the default value
			if tt.name == "default hook type" {
				hookTypes = []string{"pre-commit"}
			} else {
				hookTypes = []string{}
			}

			// Parse command properly through execute to handle subcommand flags
			rootCmd.SetArgs(tt.args)
			cmd, err := rootCmd.ExecuteC()
			if err != nil {
				// For testing flag parsing, we expect parse errors but not execution errors
				// Since we can't actually run install without proper git repo setup
				require.Contains(t, err.Error(), "failed to load configuration")
			}
			assert.Equal(t, "install", cmd.Name())

			// Validate
			tt.validate(t)
		})
	}
}

func TestInstallCmd_CommandStructure(t *testing.T) {
	// Verify command exists and has correct structure
	cmd, _, err := rootCmd.Find([]string{"install"})
	require.NoError(t, err)
	assert.Equal(t, "install", cmd.Name())
	assert.Contains(t, cmd.Short, "Install")

	// Check flags exist
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)

	hookTypeFlag := cmd.Flags().Lookup("hook-type")
	assert.NotNil(t, hookTypeFlag)
}
