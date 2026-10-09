package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// cloneModeOf returns the clone_mode field of a JSON CLI response's data object
// (nil when omitted, i.e. the default).
func cloneModeOf(t *testing.T, resp CLIResponse) any {
	t.Helper()
	data, ok := resp.Data.(map[string]any)
	require.True(t, ok, "response data should be an object, got %T", resp.Data)
	return data["clone_mode"]
}

// updateTargetCloneMode runs `db target update --clone-mode` for acme/test-repo-1.
func updateTargetCloneMode(t *testing.T, mode string) (CLIResponse, error) {
	t.Helper()
	cmd := newDBTargetUpdateCmd()
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.Flags().Set("clone-mode", mode))
	return captureJSON(t, func() error {
		return runTargetUpdate(cmd, "my-tools", "acme/test-repo-1", "", "", "", "", "", "", true)
	})
}

func TestTargetCloneMode_CLI(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("defaults to empty (sparse) for existing targets", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runTargetGet(context.Background(), "my-tools", "acme/test-repo-1", true)
		})
		require.NoError(t, err)
		assert.Nil(t, cloneModeOf(t, resp))
	})

	t.Run("update sets full mode", func(t *testing.T) {
		resp, err := updateTargetCloneMode(t, "full")
		require.NoError(t, err)
		require.True(t, resp.Success, resp.Error)
		assert.Equal(t, "full", cloneModeOf(t, resp))

		got, err := captureJSON(t, func() error {
			return runTargetGet(context.Background(), "my-tools", "acme/test-repo-1", true)
		})
		require.NoError(t, err)
		assert.Equal(t, "full", cloneModeOf(t, got))
	})

	t.Run("update leaves clone mode unchanged when the flag is omitted", func(t *testing.T) {
		cmd := newDBTargetUpdateCmd()
		cmd.SetContext(context.Background())
		require.NoError(t, cmd.Flags().Set("branch", "main"))
		resp, err := captureJSON(t, func() error {
			return runTargetUpdate(cmd, "my-tools", "acme/test-repo-1", "main", "", "", "", "", "", true)
		})
		require.NoError(t, err)
		assert.Equal(t, "full", cloneModeOf(t, resp))
	})

	t.Run("list shows clone mode", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runTargetList(context.Background(), "my-tools", true)
		})
		require.NoError(t, err)
		items, ok := resp.Data.([]any)
		require.True(t, ok)
		modes := map[string]any{}
		for _, item := range items {
			entry, isMap := item.(map[string]any)
			require.True(t, isMap)
			modes[entry["repo"].(string)] = entry["clone_mode"]
		}
		assert.Equal(t, "full", modes["acme/test-repo-1"])
		assert.Nil(t, modes["acme/test-repo-2"])

		require.NoError(t, runTargetList(context.Background(), "my-tools", false), "human output")
	})

	t.Run("clone copies clone mode from the source target", func(t *testing.T) {
		cmd := newDBTargetCloneCmd()
		cmd.SetContext(context.Background())
		resp, err := captureJSON(t, func() error {
			return runTargetClone(cmd, "my-tools", "acme/test-repo-1", "acme/clone-mode-copy", "", "", "", "", "", "", "", true)
		})
		require.NoError(t, err)
		require.True(t, resp.Success, resp.Error)
		assert.Equal(t, "full", cloneModeOf(t, resp))
	})

	t.Run("invalid clone mode is rejected without changes", func(t *testing.T) {
		resp, err := updateTargetCloneMode(t, "shallow")
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Contains(t, resp.Error, "invalid clone_mode")
		assert.Contains(t, resp.Hint, "--clone-mode full")

		got, err := captureJSON(t, func() error {
			return runTargetGet(context.Background(), "my-tools", "acme/test-repo-1", true)
		})
		require.NoError(t, err)
		assert.Equal(t, "full", cloneModeOf(t, got))
	})

	t.Run("invalid clone mode in human mode returns an error", func(t *testing.T) {
		cmd := newDBTargetUpdateCmd()
		cmd.SetContext(context.Background())
		require.NoError(t, cmd.Flags().Set("clone-mode", "partial"))
		err := runTargetUpdate(cmd, "my-tools", "acme/test-repo-1", "", "", "", "", "", "", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid clone_mode")
	})

	t.Run("empty value resets to the default", func(t *testing.T) {
		resp, err := updateTargetCloneMode(t, "")
		require.NoError(t, err)
		require.True(t, resp.Success, resp.Error)
		assert.Nil(t, cloneModeOf(t, resp))
	})
}

// TestSync_FromDB_CarriesCloneMode asserts that `sync --from-db` hands each
// target's clone mode to the sync engine: an explicit "full" survives loading,
// and targets without a setting resolve to the sparse default.
func TestSync_FromDB_CarriesCloneMode(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, err := updateTargetCloneMode(t, config.CloneModeFull)
	require.NoError(t, err)

	cfg, err := loadConfigFromDB(context.Background())
	require.NoError(t, err)
	require.Len(t, cfg.Groups, 1)

	modes := map[string]string{}
	for _, target := range cfg.Groups[0].Targets {
		modes[target.Repo] = config.ResolveCloneMode(target.CloneMode)
	}
	assert.Equal(t, config.CloneModeFull, modes["acme/test-repo-1"])
	assert.Equal(t, config.CloneModeSparse, modes["acme/test-repo-2"])
}
