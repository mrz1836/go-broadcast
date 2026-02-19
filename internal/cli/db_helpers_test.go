package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
)

func TestGetDefaultConfig(t *testing.T) {
	t.Parallel()

	t.Run("not found returns user-facing error", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		ctx := context.Background()

		cfg, err := getDefaultConfig(ctx, gormDB)
		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "no configuration found")
	})

	t.Run("found returns config", func(t *testing.T) {
		t.Parallel()

		gormDB, seed := db.TestDBWithSeed(t)
		ctx := context.Background()

		cfg, err := getDefaultConfig(ctx, gormDB)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, seed.Config.ID, cfg.ID)
		assert.Equal(t, "Test Configuration", cfg.Name)
	})
}

func TestResolveGroup(t *testing.T) {
	t.Parallel()

	t.Run("not found returns helpful error", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		ctx := context.Background()

		group, err := resolveGroup(ctx, gormDB, "nonexistent-group")
		require.Error(t, err)
		assert.Nil(t, group)
		assert.Contains(t, err.Error(), `group "nonexistent-group" not found`)
	})

	t.Run("found returns group", func(t *testing.T) {
		t.Parallel()

		gormDB, seed := db.TestDBWithSeed(t)
		ctx := context.Background()

		group, err := resolveGroup(ctx, gormDB, "mrz-tools")
		require.NoError(t, err)
		require.NotNil(t, group)
		assert.Equal(t, seed.Groups[0].ID, group.ID)
		assert.Equal(t, "MrZ Tools", group.Name)
	})
}

func TestResolveTarget(t *testing.T) {
	t.Parallel()

	t.Run("not found returns helpful error", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		ctx := context.Background()

		target, err := resolveTarget(ctx, gormDB, 1, "org/nonexistent")
		require.Error(t, err)
		assert.Nil(t, target)
		assert.Contains(t, err.Error(), `target "org/nonexistent" not found`)
	})

	t.Run("found returns target", func(t *testing.T) {
		t.Parallel()

		gormDB, seed := db.TestDBWithSeed(t)
		ctx := context.Background()

		target, err := resolveTarget(ctx, gormDB, seed.Groups[0].ID, "mrz1836/test-repo-1")
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.Equal(t, seed.Targets[0].ID, target.ID)
	})
}

func TestResolveFileList(t *testing.T) {
	t.Parallel()

	t.Run("not found returns helpful error", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		ctx := context.Background()

		fl, err := resolveFileList(ctx, gormDB, "nonexistent-list")
		require.Error(t, err)
		assert.Nil(t, fl)
		assert.Contains(t, err.Error(), `file list "nonexistent-list" not found`)
	})

	t.Run("found returns file list", func(t *testing.T) {
		t.Parallel()

		gormDB, seed := db.TestDBWithSeed(t)
		ctx := context.Background()

		fl, err := resolveFileList(ctx, gormDB, "ai-files")
		require.NoError(t, err)
		require.NotNil(t, fl)
		assert.Equal(t, seed.FileLists[0].ID, fl.ID)
		assert.Equal(t, "AI Configuration Files", fl.Name)
	})
}

func TestResolveDirectoryList(t *testing.T) {
	t.Parallel()

	t.Run("not found returns helpful error", func(t *testing.T) {
		t.Parallel()

		gormDB := db.TestDB(t)
		ctx := context.Background()

		dl, err := resolveDirectoryList(ctx, gormDB, "nonexistent-dir-list")
		require.Error(t, err)
		assert.Nil(t, dl)
		assert.Contains(t, err.Error(), `directory list "nonexistent-dir-list" not found`)
	})

	t.Run("found returns directory list", func(t *testing.T) {
		t.Parallel()

		gormDB, seed := db.TestDBWithSeed(t)
		ctx := context.Background()

		dl, err := resolveDirectoryList(ctx, gormDB, "github-workflows")
		require.NoError(t, err)
		require.NotNil(t, dl)
		assert.Equal(t, seed.DirectoryLists[0].ID, dl.ID)
		assert.Equal(t, "GitHub Workflows", dl.Name)
	})
}
