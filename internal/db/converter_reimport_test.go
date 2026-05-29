package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
)

// TestConverterImport_ReimportWithFileListRefs is a regression test for a bug
// where re-importing an existing config (the update path that calls
// deleteGroupAssociations) failed with "FOREIGN KEY constraint failed" because
// targets were deleted while child TargetFileListRef/TargetDirectoryListRef rows
// (real FK to Target) still referenced them, and polymorphic file/dir mappings
// and transforms were orphaned rather than cascaded.
func TestConverterImport_ReimportWithFileListRefs(t *testing.T) {
	db := TestDB(t)
	converter := NewConverter(db)
	ctx := context.Background()

	cfg := &config.Config{
		Version: 1,
		ID:      "reimport-test",
		Name:    "Reimport Test",
		FileLists: []config.FileList{
			{
				ID:   "common-files",
				Name: "Common Files",
				Files: []config.FileMapping{
					{Src: "README.md", Dest: "README.md"},
				},
			},
		},
		Groups: []config.Group{
			{
				ID:   "reimport-group",
				Name: "Reimport Group",
				Source: config.SourceConfig{
					Repo:   "mrz1836/test",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:   "mrz1836/target1",
						Branch: "main",
						Files: []config.FileMapping{
							{Src: "inline.txt", Dest: "inline.txt"},
						},
						FileListRefs: []string{"common-files"},
						Transform: config.Transform{
							RepoName: true,
						},
					},
				},
			},
		},
	}

	// First import succeeds.
	_, err := converter.ImportConfig(ctx, cfg)
	require.NoError(t, err)

	// Re-import the same config: exercises the update path (deleteGroupAssociations).
	// Before the fix this failed with "FOREIGN KEY constraint failed".
	_, err = converter.ImportConfig(ctx, cfg)
	require.NoError(t, err, "re-importing an existing config must not fail")

	// Verify the config round-trips correctly and no orphaned children accumulated.
	exported, err := converter.ExportConfig(ctx, cfg.ID)
	require.NoError(t, err)
	require.Len(t, exported.Groups, 1)
	require.Len(t, exported.Groups[0].Targets, 1)
	assert.Equal(t, []string{"common-files"}, exported.Groups[0].Targets[0].FileListRefs)
	assert.Len(t, exported.Groups[0].Targets[0].Files, 1)

	// No orphaned polymorphic rows should remain for targets.
	var targetCount int64
	require.NoError(t, db.Model(&Target{}).Count(&targetCount).Error)
	assert.Equal(t, int64(1), targetCount, "exactly one target after re-import")

	var refCount int64
	require.NoError(t, db.Model(&TargetFileListRef{}).Count(&refCount).Error)
	assert.Equal(t, int64(1), refCount, "exactly one file list ref after re-import")

	// Polymorphic file mappings owned by a target: 1 inline file (the file-list's
	// file is owned by the file_list, not the target).
	var targetFileMappings int64
	require.NoError(t, db.Model(&FileMapping{}).
		Where("owner_type = ?", "target").Count(&targetFileMappings).Error)
	assert.Equal(t, int64(1), targetFileMappings, "no orphaned/duplicated target file mappings")

	var targetTransforms int64
	require.NoError(t, db.Model(&Transform{}).
		Where("owner_type = ?", "target").Count(&targetTransforms).Error)
	assert.Equal(t, int64(1), targetTransforms, "no orphaned/duplicated target transforms")
}
