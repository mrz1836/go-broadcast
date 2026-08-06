package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAttachUpdateCommandWiring locks the seam between the root command and the
// go-selfupdate cobracmd package: the self-update command is registered under
// the name "update" with the "upgrade" alias (preserving the old command name),
// the check/force/verbose boolean flags, and a hidden, inert --use-binary flag.
// The command's behavior itself is covered by the library's own suites, so this
// asserts only the wiring.
func TestAttachUpdateCommandWiring(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "go-broadcast"}

	cmd := attachUpdateCommand(root)
	require.NotNil(t, cmd, "attachUpdateCommand returns the registered command")

	assert.Equal(t, "update", cmd.Name(), "the self-update command is named update")
	assert.Contains(t, cmd.Aliases, "upgrade", "the update command keeps the upgrade alias")
	assert.NotEmpty(t, cmd.Short, "the update command has a Short description")
	assert.NotEmpty(t, cmd.Long, "the update command has a Long description")
	assert.NotEmpty(t, cmd.Example, "the update command has an Example section")

	// The command is registered on root under the update name.
	var registered *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "update" {
			registered = c
			break
		}
	}
	require.NotNil(t, registered, "root registers an update command")

	for _, name := range []string{"check", "force", "verbose"} {
		flag := registered.Flags().Lookup(name)
		require.NotNilf(t, flag, "the update command registers --%s", name)
		assert.Equalf(t, "bool", flag.Value.Type(), "--%s is a boolean flag", name)
	}

	// The deprecated --use-binary flag is accepted for compatibility but hidden.
	useBinary := registered.Flags().Lookup("use-binary")
	require.NotNil(t, useBinary, "the deprecated --use-binary flag is registered")
	assert.True(t, useBinary.Hidden, "--use-binary is hidden")
}

// TestAttachUpdateCommandDropsCheckShorthand verifies that --check no longer
// carries the library's default -c shorthand, which would otherwise collide with
// go-broadcast's global --config/-c when cobra merges the inherited persistent
// flags. The collision would surface as a pflag panic while rendering help.
func TestAttachUpdateCommandDropsCheckShorthand(t *testing.T) {
	t.Parallel()

	// Reproduce go-broadcast's real setup: a persistent --config bound to -c.
	root := &cobra.Command{Use: "go-broadcast"}
	root.PersistentFlags().StringP("config", "c", "sync.yaml", "Path to configuration file")

	cmd := attachUpdateCommand(root)

	check := cmd.Flags().Lookup("check")
	require.NotNil(t, check, "the update command registers --check")
	assert.Empty(t, check.Shorthand, "--check's colliding -c shorthand is dropped")

	// Rendering help forces cobra to merge the inherited persistent flags. With
	// the shorthand still in place this panics with a redefined-shorthand error;
	// the drop keeps the merge safe.
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"update", "--help"})
	require.NotPanics(t, func() {
		require.NoError(t, root.Execute())
	})
	assert.Contains(t, out.String(), "update", "help renders for the update command")
}
