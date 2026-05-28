package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
)

// TestDBDiff_ResolvesListRefs_OnDBSide verifies that the db_diff code path resolves
// file_list_refs on the DB-exported config before comparing, producing symmetric
// inputs for compareConfigs (AC-12).
//
// Case 1: identical YAML and DB state → zero diffs after both sides are resolved.
// Case 2: DB missing a file_list that YAML has → compareConfigs detects the gap.
func TestDBDiff_ResolvesListRefs_OnDBSide(t *testing.T) {
	tmpDir := t.TempDir()
	testDBPath := filepath.Join(tmpDir, "diff-resolve.db")

	database, err := db.Open(db.OpenOptions{
		Path:        testDBPath,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	converter := db.NewConverter(database.DB())

	// Seed DB with a config carrying file_list "fl1" and a target with file_list_refs: ["fl1"].
	seedCfg := &config.Config{
		Version: 1,
		Name:    "diff-test",
		ID:      "diff-test",
		FileLists: []config.FileList{
			{
				ID:   "fl1",
				Name: "File List 1",
				Files: []config.FileMapping{
					{Src: "shared.txt", Dest: "shared.txt"},
				},
			},
		},
		Groups: []config.Group{
			{
				Name: "Test Group",
				ID:   "diff-group",
				Source: config.SourceConfig{
					Repo:   "acme/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:         "acme/target1",
						FileListRefs: []string{"fl1"},
					},
				},
			},
		},
	}

	_, err = converter.ImportConfig(ctx, seedCfg)
	require.NoError(t, err)

	t.Run("zero diffs when YAML and DB carry matching file_list_refs", func(t *testing.T) {
		// Simulate the YAML side: apply defaults + resolve (what config.Load does).
		yamlCfg := &config.Config{
			Version: 1,
			Name:    "diff-test",
			ID:      "diff-test",
			FileLists: []config.FileList{
				{
					ID:   "fl1",
					Name: "File List 1",
					Files: []config.FileMapping{
						{Src: "shared.txt", Dest: "shared.txt"},
					},
				},
			},
			Groups: []config.Group{
				{
					Name: "Test Group",
					ID:   "diff-group",
					Source: config.SourceConfig{
						Repo:   "acme/source",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo:         "acme/target1",
							FileListRefs: []string{"fl1"},
						},
					},
				},
			},
		}
		require.NoError(t, config.ApplyDefaultsAndResolve(yamlCfg))

		// Simulate the DB side: export then resolve (what db_diff.go now does after Phase 1 fix).
		dbCfg, dbErr := converter.ExportConfig(ctx, "diff-test")
		require.NoError(t, dbErr)

		// Before resolve the target should have FileListRefs but empty Files.
		require.NotEmpty(t, dbCfg.Groups[0].Targets[0].FileListRefs,
			"exported config must carry file_list_refs before resolve")
		assert.Empty(t, dbCfg.Groups[0].Targets[0].Files,
			"exported config must NOT have Files before ApplyDefaultsAndResolve is called")

		require.NoError(t, config.ApplyDefaultsAndResolve(dbCfg))

		// After resolve the target must have files merged from fl1.
		assert.NotEmpty(t, dbCfg.Groups[0].Targets[0].Files,
			"target.Files must be non-empty after ApplyDefaultsAndResolve resolves the file_list_ref")

		// Structural comparison should show no diffs when both sides are resolved.
		diffs := compareConfigs(yamlCfg, dbCfg, false)
		assert.Empty(t, diffs, "expected zero compareConfigs diffs when YAML and DB carry matching resolved config")
	})

	t.Run("non-empty diffs when YAML has a file_list that DB is missing", func(t *testing.T) {
		// YAML declares an extra file_list "fl2" that is absent from the DB.
		// compareConfigs should detect the missing FileList on the DB side.
		yamlWithExtra := &config.Config{
			Version: 1,
			Name:    "diff-test",
			ID:      "diff-test",
			FileLists: []config.FileList{
				{
					ID:   "fl1",
					Name: "File List 1",
					Files: []config.FileMapping{
						{Src: "shared.txt", Dest: "shared.txt"},
					},
				},
				{
					ID:   "fl2",
					Name: "File List 2",
					Files: []config.FileMapping{
						{Src: "extra.txt", Dest: "extra.txt"},
					},
				},
			},
			Groups: []config.Group{
				{
					Name: "Test Group",
					ID:   "diff-group",
					Source: config.SourceConfig{
						Repo:   "acme/source",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo:         "acme/target1",
							FileListRefs: []string{"fl1"},
						},
					},
				},
			},
		}
		require.NoError(t, config.ApplyDefaultsAndResolve(yamlWithExtra))

		dbCfg, dbErr := converter.ExportConfig(ctx, "diff-test")
		require.NoError(t, dbErr)
		require.NoError(t, config.ApplyDefaultsAndResolve(dbCfg))

		diffs := compareConfigs(yamlWithExtra, dbCfg, false)
		assert.NotEmpty(t, diffs,
			"expected non-empty compareConfigs diffs when YAML has a file_list absent from DB")

		// Verify the diff mentions the missing file_list specifically.
		diffStr := ""
		for _, d := range diffs {
			diffStr += d
		}
		assert.Contains(t, diffStr, "FileList",
			"diff output should reference the missing FileList")
	})
}
