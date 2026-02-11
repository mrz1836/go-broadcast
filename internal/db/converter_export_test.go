package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestConverterExportErrors tests export error paths
func TestConverterExportErrors(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Export config that doesn't exist", func(t *testing.T) {
		_, err := converter.ExportConfig(ctx, "does-not-exist")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Export group from non-existent config", func(t *testing.T) {
		_, err := converter.ExportGroup(ctx, 99999, "some-group")
		assert.Error(t, err)
	})
}

// TestConverterExport_FileListRefs tests exportFileListRefs edge cases
func TestConverterExport_FileListRefs(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		ID:      "filelist-refs-test",
		Name:    "FileList Refs Test",
		FileLists: []config.FileList{
			{
				ID:   "list1",
				Name: "List 1",
				Files: []config.FileMapping{
					{Src: "file1.txt", Dest: "file1.txt"},
				},
			},
			{
				ID:   "list2",
				Name: "List 2",
				Files: []config.FileMapping{
					{Src: "file2.txt", Dest: "file2.txt"},
				},
			},
		},
		Groups: []config.Group{
			{
				ID:   "test-group-refs",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:         "mrz1836/target",
						FileListRefs: []string{"list1", "list2"}, // Multiple refs
					},
				},
			},
		},
	}

	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)

	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)

	refs := exported.Groups[0].Targets[0].FileListRefs
	assert.Len(t, refs, 2)
	assert.ElementsMatch(t, []string{"list1", "list2"}, refs)
}

// TestConverterExport_NonExistent tests export error paths
func TestConverterExport_NonExistent(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Export non-existent config", func(t *testing.T) {
		_, err := converter.ExportConfig(ctx, "non-existent-config")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})

	t.Run("Export non-existent group", func(t *testing.T) {
		// Create a config first
		cfg := &config.Config{
			Version: 1,
			ID:      "export-test",
			Name:    "Export Test",
			Groups: []config.Group{
				{
					ID:   "existing-group",
					Name: "Existing Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{{Repo: "mrz1836/target"}},
				},
			},
		}

		importedConfig, err := converter.ImportConfig(ctx, cfg)
		require.NoError(t, err)

		_, err = converter.ExportGroup(ctx, importedConfig.ID, "non-existent-group")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}

// TestConverterValidateReferences_AllPaths tests all error paths in validateReferences
func TestConverterValidateReferences_AllPaths(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	t.Run("Missing directory list reference", func(t *testing.T) {
		cfg := &config.Config{
			Version:        1,
			ID:             "missing-dirlist-ref-test",
			Name:           "Missing DirList Ref Test",
			DirectoryLists: []config.DirectoryList{}, // Empty
			Groups: []config.Group{
				{
					ID:   "test-group",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo:              "mrz1836/target",
							DirectoryListRefs: []string{"non-existent-dir-list"}, // Missing reference
						},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrReferenceNotFound)
		assert.Contains(t, err.Error(), "directory list")
	})

	t.Run("Missing group dependency reference", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			ID:      "missing-dep-test",
			Name:    "Missing Dependency Test",
			Groups: []config.Group{
				{
					ID:        "group-a",
					Name:      "Group A",
					DependsOn: []string{"non-existent-group"}, // Missing dependency
					Source: config.SourceConfig{
						Repo:   "mrz1836/test",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "mrz1836/target"},
					},
				},
			},
		}

		_, err := converter.ImportConfig(ctx, cfg)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrReferenceNotFound)
		assert.Contains(t, err.Error(), "group dependency")
	})
}
