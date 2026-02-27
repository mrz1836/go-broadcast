package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// setupBulkTestDB creates a temp SQLite DB, initializes it, and returns a cleanup function.
// It also seeds the data from TestDBWithSeed into the file-based DB so that group
// "mrz-tools" and file/directory lists exist for the "not-found list" tests.
func setupBulkTestDB(t *testing.T, seed bool) {
	t.Helper()

	tmpDir := t.TempDir()
	dbFilePath := filepath.Join(tmpDir, "bulk_test.db")

	// Save and restore package-level dbPath
	oldPath := dbPath
	t.Cleanup(func() { dbPath = oldPath })
	dbPath = dbFilePath

	// Initialize DB file
	oldForce := dbInitForce
	t.Cleanup(func() { dbInitForce = oldForce })
	dbInitForce = false

	require.NoError(t, runDBInit(nil, nil))

	if seed {
		// Open the file DB and insert realistic seed data via the bulk-friendly repo helpers.
		database, err := db.Open(db.OpenOptions{Path: dbFilePath})
		require.NoError(t, err)
		t.Cleanup(func() { _ = database.Close() })

		gormDB := database.DB()

		cfg := &db.Config{ExternalID: "test-cfg", Name: "Test Config", Version: 1}
		require.NoError(t, gormDB.Create(cfg).Error)

		client := &db.Client{Name: "test-client"}
		require.NoError(t, gormDB.Create(client).Error)

		org := &db.Organization{ClientID: client.ID, Name: "mrz1836"}
		require.NoError(t, gormDB.Create(org).Error)

		enabled := true
		group := &db.Group{
			ConfigID:   cfg.ID,
			ExternalID: "mrz-tools",
			Name:       "MrZ Tools",
			Enabled:    &enabled,
		}
		require.NoError(t, gormDB.Create(group).Error)

		fl := &db.FileList{
			ConfigID:   cfg.ID,
			ExternalID: "ai-files",
			Name:       "AI Files",
		}
		require.NoError(t, gormDB.Create(fl).Error)

		dl := &db.DirectoryList{
			ConfigID:   cfg.ID,
			ExternalID: "github-workflows",
			Name:       "GitHub Workflows",
		}
		require.NoError(t, gormDB.Create(dl).Error)
	}
}

// --- runBulkAddFileList ---

func TestRunBulkAddFileList_GroupNotFound(t *testing.T) {
	setupBulkTestDB(t, false)

	err := runBulkAddFileList("nonexistent-group", "any-list", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunBulkAddFileList_FileListNotFound(t *testing.T) {
	setupBulkTestDB(t, true)

	err := runBulkAddFileList("mrz-tools", "nonexistent-file-list", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- runBulkRemoveFileList ---

func TestRunBulkRemoveFileList_GroupNotFound(t *testing.T) {
	setupBulkTestDB(t, false)

	err := runBulkRemoveFileList("nonexistent-group", "any-list", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunBulkRemoveFileList_FileListNotFound(t *testing.T) {
	setupBulkTestDB(t, true)

	err := runBulkRemoveFileList("mrz-tools", "nonexistent-file-list", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- runBulkAddDirList ---

func TestRunBulkAddDirList_GroupNotFound(t *testing.T) {
	setupBulkTestDB(t, false)

	err := runBulkAddDirList("nonexistent-group", "any-list", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunBulkAddDirList_DirListNotFound(t *testing.T) {
	setupBulkTestDB(t, true)

	err := runBulkAddDirList("mrz-tools", "nonexistent-dir-list", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- runBulkRemoveDirList ---

func TestRunBulkRemoveDirList_GroupNotFound(t *testing.T) {
	setupBulkTestDB(t, false)

	err := runBulkRemoveDirList("nonexistent-group", "any-list", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunBulkRemoveDirList_DirListNotFound(t *testing.T) {
	setupBulkTestDB(t, true)

	err := runBulkRemoveDirList("mrz-tools", "nonexistent-dir-list", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- Command structure tests ---

func TestNewDBBulkCmd(t *testing.T) {
	t.Parallel()

	cmd := newDBBulkCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "bulk", cmd.Use)

	subNames := make([]string, 0, len(cmd.Commands()))
	for _, sub := range cmd.Commands() {
		subNames = append(subNames, sub.Use)
	}
	assert.Contains(t, subNames, "add-file-list")
	assert.Contains(t, subNames, "remove-file-list")
	assert.Contains(t, subNames, "add-dir-list")
	assert.Contains(t, subNames, "remove-dir-list")
}
