package cli

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
)

// TestSync_FromDB_ResolvesFileListRefs asserts that file_list_refs entries are merged
// into target.Files when the config is loaded from DB via loadConfigFromDB (AC-2).
func TestSync_FromDB_ResolvesFileListRefs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// setupTestDB wires target1 (acme/test-repo-1) → TargetFileListRef → ai-files.
	// ai-files contains ".cursorrules". target1 also has inline ".editorconfig".
	// After loadConfigFromDB resolves refs, both must appear in target.Files.
	cfg, err := loadConfigFromDB()
	require.NoError(t, err)
	require.Len(t, cfg.Groups, 1)

	var target *config.TargetConfig
	for i := range cfg.Groups[0].Targets {
		if cfg.Groups[0].Targets[i].Repo == "acme/test-repo-1" {
			target = &cfg.Groups[0].Targets[i]
			break
		}
	}
	require.NotNil(t, target, "target acme/test-repo-1 not found in loaded config")

	dests := make(map[string]bool, len(target.Files))
	for _, f := range target.Files {
		dests[f.Dest] = true
	}
	assert.True(t, dests[".cursorrules"], "expected .cursorrules from ai-files file_list_ref to be merged")
	assert.True(t, dests[".editorconfig"], "expected .editorconfig from inline file mapping to be preserved")
}

// TestSync_FromDB_ResolvesDirectoryListRefs asserts that directory_list_refs entries are merged
// into target.Directories when the config is loaded from DB via loadConfigFromDB (AC-3).
func TestSync_FromDB_ResolvesDirectoryListRefs(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// setupTestDB wires target2 (acme/test-repo-2) → TargetDirectoryListRef → github-workflows.
	// github-workflows starts with no entries; add one so we can assert the merge.
	{
		database, err := openDatabase()
		require.NoError(t, err)
		gormDB := database.DB()

		var dl db.DirectoryList
		require.NoError(t, gormDB.Where("external_id = ?", "github-workflows").First(&dl).Error)

		require.NoError(t, gormDB.Create(&db.DirectoryMapping{
			OwnerType: "directory_list",
			OwnerID:   dl.ID,
			Src:       ".github/workflows",
			Dest:      ".github/workflows",
			Position:  0,
		}).Error)
		require.NoError(t, database.Close())
	}

	cfg, err := loadConfigFromDB()
	require.NoError(t, err)
	require.Len(t, cfg.Groups, 1)

	var target *config.TargetConfig
	for i := range cfg.Groups[0].Targets {
		if cfg.Groups[0].Targets[i].Repo == "acme/test-repo-2" {
			target = &cfg.Groups[0].Targets[i]
			break
		}
	}
	require.NotNil(t, target, "target acme/test-repo-2 not found in loaded config")

	require.NotEmpty(t, target.Directories, "expected directory from github-workflows directory_list_ref to be merged")
	dests := make(map[string]bool, len(target.Directories))
	for _, d := range target.Directories {
		dests[d.Dest] = true
	}
	assert.True(t, dests[".github/workflows"], "expected .github/workflows from github-workflows ref")
}

// TestSync_FromDB_MissingFileListRef_Errors asserts that a target whose file_list_refs
// point to a list absent from the config's FileLists surfaces ErrListReferenceNotFound
// with the "available file_lists" hint, matching YAML-loader error parity (AC-4).
func TestSync_FromDB_MissingFileListRef_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "missing-ref.db")

	oldDBPath := dbPath
	dbPath = tmpPath
	defer func() { dbPath = oldDBPath }()

	// Build a DB where the main config has a target whose TargetFileListRef points to a
	// file_list owned by a DIFFERENT config. ExportConfig for the main config will export
	// the ref's ExternalID ("ghost-list") but won't include it in config.FileLists (only
	// file_lists scoped to the main config_id are loaded). ApplyDefaultsAndResolve will
	// then fail with ErrListReferenceNotFound.
	database, err := db.Open(db.OpenOptions{Path: tmpPath, AutoMigrate: true})
	require.NoError(t, err)
	gormDB := database.DB()

	// Main config — loaded first by loadConfigFromDB (lowest ID)
	mainCfg := &db.Config{ExternalID: "main-config", Name: "Main Config", Version: 1}
	require.NoError(t, gormDB.Create(mainCfg).Error)

	// Auxiliary config — owns the ghost file_list
	auxCfg := &db.Config{ExternalID: "aux-config", Name: "Aux Config", Version: 1}
	require.NoError(t, gormDB.Create(auxCfg).Error)

	// File list scoped to auxCfg — will NOT appear in main config's exported FileLists
	ghostList := &db.FileList{
		ConfigID:   auxCfg.ID,
		ExternalID: "ghost-list",
		Name:       "Ghost List",
		Position:   0,
	}
	require.NoError(t, gormDB.Create(ghostList).Error)
	require.NoError(t, gormDB.Create(&db.FileMapping{
		OwnerType: "file_list",
		OwnerID:   ghostList.ID,
		Src:       "ghost.txt",
		Dest:      "ghost.txt",
		Position:  0,
	}).Error)

	// Minimal client / org / repo hierarchy for main config
	client := &db.Client{Name: "acme-ghost"}
	require.NoError(t, gormDB.Create(client).Error)
	org := &db.Organization{ClientID: client.ID, Name: "acme-ghost"}
	require.NoError(t, gormDB.Create(org).Error)
	srcRepo := &db.Repo{OrganizationID: org.ID, Name: "go-broadcast"}
	require.NoError(t, gormDB.Create(srcRepo).Error)
	tgtRepo := &db.Repo{OrganizationID: org.ID, Name: "target-repo"}
	require.NoError(t, gormDB.Create(tgtRepo).Error)

	enabled := true
	group := &db.Group{
		ConfigID:   mainCfg.ID,
		ExternalID: "test-group-ghost",
		Name:       "Test Group",
		Enabled:    &enabled,
		Position:   0,
	}
	require.NoError(t, gormDB.Create(group).Error)
	require.NoError(t, gormDB.Create(&db.Source{GroupID: group.ID, RepoID: srcRepo.ID, Branch: "main"}).Error)
	require.NoError(t, gormDB.Create(&db.GroupGlobal{GroupID: group.ID}).Error)
	require.NoError(t, gormDB.Create(&db.GroupDefault{GroupID: group.ID}).Error)

	target := &db.Target{GroupID: group.ID, RepoID: tgtRepo.ID, Branch: "main", Position: 0}
	require.NoError(t, gormDB.Create(target).Error)

	// Cross-config ref: target under mainCfg → ghost-list under auxCfg
	require.NoError(t, gormDB.Create(&db.TargetFileListRef{
		TargetID:   target.ID,
		FileListID: ghostList.ID,
		Position:   0,
	}).Error)

	require.NoError(t, database.Close())

	_, err = loadConfigFromDB()
	require.Error(t, err)
	assert.True(t, errors.Is(err, config.ErrListReferenceNotFound),
		"expected ErrListReferenceNotFound in error chain, got: %v", err)
	assert.Contains(t, err.Error(), "available file_lists",
		"error message should include 'available file_lists' hint from formatListNotFoundError")
}

// TestSync_FromDB_InlineOverridesRef_OnDestCollision asserts that when the same dest
// exists both as an inline target FileMapping and in a referenced file_list, the inline
// entry wins (matching parser.go:267-270 precedence), and exactly one entry remains (AC-7).
func TestSync_FromDB_InlineOverridesRef_OnDestCollision(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Add "shared.txt" to both ai-files (ref side, src="from-ref") and target1 inline
	// (src="from-inline"). After resolve, only the inline entry should survive.
	{
		database, err := openDatabase()
		require.NoError(t, err)
		gormDB := database.DB()

		var fl db.FileList
		require.NoError(t, gormDB.Where("external_id = ?", "ai-files").First(&fl).Error)

		var target1 db.Target
		require.NoError(t, gormDB.Where("position = ?", 0).First(&target1).Error)

		require.NoError(t, gormDB.Create(&db.FileMapping{
			OwnerType: "file_list",
			OwnerID:   fl.ID,
			Src:       "from-ref",
			Dest:      "shared.txt",
			Position:  1,
		}).Error)

		require.NoError(t, gormDB.Create(&db.FileMapping{
			OwnerType: "target",
			OwnerID:   target1.ID,
			Src:       "from-inline",
			Dest:      "shared.txt",
			Position:  1,
		}).Error)

		require.NoError(t, database.Close())
	}

	cfg, err := loadConfigFromDB()
	require.NoError(t, err)
	require.Len(t, cfg.Groups, 1)

	var target *config.TargetConfig
	for i := range cfg.Groups[0].Targets {
		if cfg.Groups[0].Targets[i].Repo == "acme/test-repo-1" {
			target = &cfg.Groups[0].Targets[i]
			break
		}
	}
	require.NotNil(t, target, "target acme/test-repo-1 not found")

	var sharedEntries []config.FileMapping
	for _, f := range target.Files {
		if f.Dest == "shared.txt" {
			sharedEntries = append(sharedEntries, f)
		}
	}
	require.Len(t, sharedEntries, 1, "expected exactly one entry for shared.txt after dedup")
	assert.Equal(t, "from-inline", sharedEntries[0].Src,
		"inline entry must override ref entry when dest collides")
}
