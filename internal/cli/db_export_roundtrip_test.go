package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
)

// TestDBExport_PreservesListRefs_Unresolved locks in the deliberate exception (call-site
// audit row #8): the db_export.go path calls converter.ExportConfig WITHOUT subsequently
// calling config.ApplyDefaultsAndResolve, so the exported YAML preserves file_list_refs
// and directory_list_refs as symbolic references rather than inlining the list entries.
// This protects the import→export round-trip authoring shape (AC-13).
func TestDBExport_PreservesListRefs_Unresolved(t *testing.T) {
	tmpDir := t.TempDir()
	testDBPath := filepath.Join(tmpDir, "export-roundtrip.db")

	database, err := db.Open(db.OpenOptions{
		Path:        testDBPath,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	converter := db.NewConverter(database.DB())

	// Seed: a config with one file_list "fl1" and one directory_list "dl1".
	// The single target references both via file_list_refs and directory_list_refs only —
	// it carries NO inline file or directory mappings.
	seedCfg := &config.Config{
		Version: 1,
		Name:    "roundtrip-test",
		ID:      "roundtrip-cfg",
		FileLists: []config.FileList{
			{
				ID:   "fl1",
				Name: "File List 1",
				Files: []config.FileMapping{
					{Src: "ref-file.txt", Dest: "ref-file.txt"},
				},
			},
		},
		DirectoryLists: []config.DirectoryList{
			{
				ID:   "dl1",
				Name: "Directory List 1",
				Directories: []config.DirectoryMapping{
					{Src: ".github/workflows", Dest: ".github/workflows"},
				},
			},
		},
		Groups: []config.Group{
			{
				Name: "Test Group",
				ID:   "roundtrip-group",
				Source: config.SourceConfig{
					Repo:   "acme/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{
						Repo:              "acme/target1",
						FileListRefs:      []string{"fl1"},
						DirectoryListRefs: []string{"dl1"},
						// Intentionally no inline Files or Directories.
					},
				},
			},
		},
	}

	_, err = converter.ImportConfig(ctx, seedCfg)
	require.NoError(t, err)

	t.Run("file_list_refs preserved un-inlined on export", func(t *testing.T) {
		// db_export.go calls ExportConfig without ApplyDefaultsAndResolve.
		exportedCfg, exportErr := converter.ExportConfig(ctx, "roundtrip-cfg")
		require.NoError(t, exportErr)

		target := exportedCfg.Groups[0].Targets[0]

		// The ref must be preserved as a symbolic reference.
		assert.Contains(t, target.FileListRefs, "fl1",
			"exported config must preserve file_list_refs verbatim")

		// The list entries must NOT be inlined into target.Files.
		for _, f := range target.Files {
			assert.NotEqual(t, "ref-file.txt", f.Dest,
				"ref-file.txt must not be inlined into target.files on db export")
		}

		// Marshal to YAML and verify the same invariant at the text level.
		yamlData, marshalErr := yaml.Marshal(exportedCfg)
		require.NoError(t, marshalErr)
		yamlStr := string(yamlData)

		assert.Contains(t, yamlStr, "fl1",
			"YAML output must contain file_list_refs entry 'fl1'")
		// "ref-file.txt" should only appear inside the file_lists section, not inline in targets.
		// A simple safeguard: the YAML target block should not contain the ref content inline.
		assert.Contains(t, yamlStr, "file_list_refs",
			"YAML output must contain file_list_refs key")
	})

	t.Run("directory_list_refs preserved un-inlined on export", func(t *testing.T) {
		exportedCfg, exportErr := converter.ExportConfig(ctx, "roundtrip-cfg")
		require.NoError(t, exportErr)

		target := exportedCfg.Groups[0].Targets[0]

		// The ref must be preserved as a symbolic reference.
		assert.Contains(t, target.DirectoryListRefs, "dl1",
			"exported config must preserve directory_list_refs verbatim")

		// The list entries must NOT be inlined into target.Directories.
		for _, d := range target.Directories {
			assert.NotEqual(t, ".github/workflows", d.Dest,
				".github/workflows must not be inlined into target.directories on db export")
		}

		yamlData, marshalErr := yaml.Marshal(exportedCfg)
		require.NoError(t, marshalErr)
		yamlStr := string(yamlData)

		assert.Contains(t, yamlStr, "dl1",
			"YAML output must contain directory_list_refs entry 'dl1'")
		assert.Contains(t, yamlStr, "directory_list_refs",
			"YAML output must contain directory_list_refs key")
	})
}
