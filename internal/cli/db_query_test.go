package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// TestDBQuery tests the db query command
func TestDBQuery(t *testing.T) {
	// Create test database with seed data
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "query.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(t, err)

	// Seed test data
	gormDB := database.DB()

	// Create config
	cfg := &db.Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	require.NoError(t, gormDB.Create(cfg).Error)

	// Create file list
	fileList := &db.FileList{
		ConfigID:   cfg.ID,
		ExternalID: "test-files",
		Name:       "Test Files",
	}
	require.NoError(t, gormDB.Create(fileList).Error)

	// Add file mapping to file list
	fileMapping := &db.FileMapping{
		OwnerType: "file_list",
		OwnerID:   fileList.ID,
		Src:       ".github/workflows/ci.yml",
		Dest:      ".github/workflows/ci.yml",
	}
	require.NoError(t, gormDB.Create(fileMapping).Error)

	// Create group
	group := &db.Group{
		ConfigID:   cfg.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	require.NoError(t, gormDB.Create(group).Error)

	// Create source
	source := &db.Source{
		GroupID: group.ID,
		Repo:    "mrz1836/source-repo",
		Branch:  "main",
	}
	require.NoError(t, gormDB.Create(source).Error)

	// Create target
	target := &db.Target{
		GroupID: group.ID,
		Repo:    "mrz1836/target-repo",
		Branch:  "main",
	}
	require.NoError(t, gormDB.Create(target).Error)

	// Add inline file mapping to target
	targetFileMapping := &db.FileMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       "README.md",
		Dest:      "README.md",
	}
	require.NoError(t, gormDB.Create(targetFileMapping).Error)

	// Add file list ref to target
	ref := &db.TargetFileListRef{
		TargetID:   target.ID,
		FileListID: fileList.ID,
	}
	require.NoError(t, gormDB.Create(ref).Error)

	require.NoError(t, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldFile := dbQueryFile
	oldRepo := dbQueryRepo
	oldFileList := dbQueryFileList
	oldContains := dbQueryContains
	oldJSON := dbQueryJSON
	defer func() {
		dbPath = oldDBPath
		dbQueryFile = oldFile
		dbQueryRepo = oldRepo
		dbQueryFileList = oldFileList
		dbQueryContains = oldContains
		dbQueryJSON = oldJSON
	}()

	dbPath = tmpPath

	t.Run("query by file", func(t *testing.T) {
		dbQueryFile = ".github/workflows/ci.yml"
		dbQueryRepo = ""
		dbQueryFileList = ""
		dbQueryContains = ""
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.NoError(t, err)
	})

	t.Run("query by file (not found)", func(t *testing.T) {
		dbQueryFile = "nonexistent-file.txt"
		dbQueryRepo = ""
		dbQueryFileList = ""
		dbQueryContains = ""
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.NoError(t, err)
	})

	t.Run("query by repo", func(t *testing.T) {
		dbQueryFile = ""
		dbQueryRepo = "mrz1836/target-repo"
		dbQueryFileList = ""
		dbQueryContains = ""
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.NoError(t, err)
	})

	t.Run("query by repo (not found)", func(t *testing.T) {
		dbQueryFile = ""
		dbQueryRepo = "nonexistent/repo"
		dbQueryFileList = ""
		dbQueryContains = ""
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.NoError(t, err)
	})

	t.Run("query by file list", func(t *testing.T) {
		dbQueryFile = ""
		dbQueryRepo = ""
		dbQueryFileList = "test-files"
		dbQueryContains = ""
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.NoError(t, err)
	})

	t.Run("query by file list (not found)", func(t *testing.T) {
		dbQueryFile = ""
		dbQueryRepo = ""
		dbQueryFileList = "nonexistent-list"
		dbQueryContains = ""
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.NoError(t, err)
	})

	t.Run("query by pattern", func(t *testing.T) {
		dbQueryFile = ""
		dbQueryRepo = ""
		dbQueryFileList = ""
		dbQueryContains = "workflows"
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.NoError(t, err)
	})

	t.Run("query by pattern (not found)", func(t *testing.T) {
		dbQueryFile = ""
		dbQueryRepo = ""
		dbQueryFileList = ""
		dbQueryContains = "zzz_nonexistent_zzz"
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.NoError(t, err)
	})

	t.Run("JSON output by file", func(t *testing.T) {
		dbQueryFile = ".github/workflows/ci.yml"
		dbQueryRepo = ""
		dbQueryFileList = ""
		dbQueryContains = ""
		dbQueryJSON = true

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runDBQuery(nil, nil)
		require.NoError(t, err)

		// Restore stdout and read output
		_ = w.Close()
		os.Stdout = oldStdout
		var output bytes.Buffer
		_, _ = output.ReadFrom(r)

		// Parse JSON output
		var result QueryResult
		err = json.Unmarshal(output.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result.Query, ".github/workflows/ci.yml")
		assert.GreaterOrEqual(t, result.Count, 1)
	})

	t.Run("JSON output by repo", func(t *testing.T) {
		dbQueryFile = ""
		dbQueryRepo = "mrz1836/target-repo"
		dbQueryFileList = ""
		dbQueryContains = ""
		dbQueryJSON = true

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runDBQuery(nil, nil)
		require.NoError(t, err)

		// Restore stdout and read output
		_ = w.Close()
		os.Stdout = oldStdout
		var output bytes.Buffer
		_, _ = output.ReadFrom(r)

		// Parse JSON output
		var result QueryResult
		err = json.Unmarshal(output.Bytes(), &result)
		require.NoError(t, err)
		assert.Contains(t, result.Query, "mrz1836/target-repo")
	})

	t.Run("requires one query flag", func(t *testing.T) {
		dbQueryFile = ""
		dbQueryRepo = ""
		dbQueryFileList = ""
		dbQueryContains = ""
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must specify one of")
	})

	t.Run("rejects multiple query flags", func(t *testing.T) {
		dbQueryFile = "test.txt"
		dbQueryRepo = "mrz1836/repo"
		dbQueryFileList = ""
		dbQueryContains = ""
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only one query flag")
	})

	t.Run("missing database", func(t *testing.T) {
		dbPath = filepath.Join(tmpDir, "nonexistent.db")
		dbQueryFile = "test.txt"
		dbQueryRepo = ""
		dbQueryFileList = ""
		dbQueryContains = ""
		dbQueryJSON = false

		err := runDBQuery(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

// TestDBQueryCommand tests query command structure
func TestDBQueryCommand(t *testing.T) {
	t.Run("command exists", func(t *testing.T) {
		assert.NotNil(t, dbQueryCmd)
		assert.Equal(t, "query", dbQueryCmd.Use)
	})

	t.Run("has required flags", func(t *testing.T) {
		flags := []string{"file", "repo", "file-list", "contains", "json"}
		for _, flagName := range flags {
			flag := dbQueryCmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "should have %s flag", flagName)
		}
	})

	t.Run("help text contains examples", func(t *testing.T) {
		assert.Contains(t, dbQueryCmd.Long, "Examples:")
		assert.Contains(t, dbQueryCmd.Long, "--file")
		assert.Contains(t, dbQueryCmd.Long, "--repo")
		assert.Contains(t, dbQueryCmd.Long, "--file-list")
		assert.Contains(t, dbQueryCmd.Long, "--contains")
	})
}

// Note: Benchmarks for db query are in db_query_benchmark_test.go with bench_heavy tag
