package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUninstallCmd_ParseFlags(t *testing.T) {
	// Reset flags
	hookTypes = []string{"pre-commit"}

	// Parse command properly through execute to handle subcommand flags
	rootCmd.SetArgs([]string{"uninstall", "--hook-type", "pre-push"})
	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		// For testing flag parsing, we expect execution errors but not parse errors
		// Since we can't actually run uninstall without proper git repo setup
		require.Contains(t, err.Error(), "failed to")
	}
	assert.Equal(t, "uninstall", cmd.Name())

	assert.Equal(t, []string{"pre-push"}, hookTypes)
}

func TestUninstallCmd_CommandStructure(t *testing.T) {
	// Verify command exists and has correct structure
	cmd, _, err := rootCmd.Find([]string{"uninstall"})
	require.NoError(t, err)
	assert.Equal(t, "uninstall", cmd.Name())
	assert.Contains(t, cmd.Short, "Uninstall")

	// Check flags exist
	hookTypeFlag := cmd.Flags().Lookup("hook-type")
	assert.NotNil(t, hookTypeFlag)
}
