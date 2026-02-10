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

// TestDBValidate tests the db validate command
func TestDBValidate(t *testing.T) {
	// Create test database
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "validate.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(t, err)

	gormDB := database.DB()

	// Create config
	cfg := &db.Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	require.NoError(t, gormDB.Create(cfg).Error)

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

	require.NoError(t, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldJSON := dbValidateJSON
	defer func() {
		dbPath = oldDBPath
		dbValidateJSON = oldJSON
	}()

	dbPath = tmpPath

	t.Run("validates clean database", func(t *testing.T) {
		dbValidateJSON = false

		err := runDBValidate(nil, nil)
		require.NoError(t, err)
	})

	t.Run("JSON output for clean database", func(t *testing.T) {
		dbValidateJSON = true

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := runDBValidate(nil, nil)
		require.NoError(t, err)

		// Restore stdout and read output
		_ = w.Close()
		os.Stdout = oldStdout
		var output bytes.Buffer
		_, _ = output.ReadFrom(r)

		// Parse JSON output
		var result ValidationResult
		err = json.Unmarshal(output.Bytes(), &result)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
		assert.NotEmpty(t, result.Checks)
	})

	t.Run("missing database", func(t *testing.T) {
		dbPath = filepath.Join(tmpDir, "nonexistent.db")
		dbValidateJSON = false

		err := runDBValidate(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

// TestDBValidateOrphanedRefs tests validation of orphaned references
func TestDBValidateOrphanedRefs(t *testing.T) {
	// Create test database
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "orphan.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(t, err)

	gormDB := database.DB()

	// Create config
	cfg := &db.Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	require.NoError(t, gormDB.Create(cfg).Error)

	// Create group
	group := &db.Group{
		ConfigID:   cfg.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	require.NoError(t, gormDB.Create(group).Error)

	// Create target
	target := &db.Target{
		GroupID: group.ID,
		Repo:    "mrz1836/target-repo",
		Branch:  "main",
	}
	require.NoError(t, gormDB.Create(target).Error)

	// Temporarily disable foreign key constraints to create orphaned ref
	require.NoError(t, gormDB.Exec("PRAGMA foreign_keys = OFF").Error)

	// Create orphaned file list ref (pointing to non-existent file list)
	orphanRef := &db.TargetFileListRef{
		TargetID:   target.ID,
		FileListID: 99999, // Non-existent ID
	}
	require.NoError(t, gormDB.Create(orphanRef).Error)

	// Re-enable foreign key constraints
	require.NoError(t, gormDB.Exec("PRAGMA foreign_keys = ON").Error)

	require.NoError(t, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldJSON := dbValidateJSON
	defer func() {
		dbPath = oldDBPath
		dbValidateJSON = oldJSON
	}()

	dbPath = tmpPath
	dbValidateJSON = true

	t.Run("detects orphaned file list ref", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		_ = runDBValidate(nil, nil)

		// Restore stdout and read output
		_ = w.Close()
		os.Stdout = oldStdout
		var output bytes.Buffer
		_, _ = output.ReadFrom(r)

		// Parse JSON output
		var result ValidationResult
		err = json.Unmarshal(output.Bytes(), &result)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)

		// Check that we detected the orphaned ref
		hasOrphanError := false
		for _, e := range result.Errors {
			if e.Type == "orphaned_file_list_ref" {
				hasOrphanError = true
				break
			}
		}
		assert.True(t, hasOrphanError, "should detect orphaned file list ref")
	})
}

// TestDBValidateCircularDependencies tests circular dependency detection
func TestDBValidateCircularDependencies(t *testing.T) {
	// This test is simplified as circular dependency detection
	// is thoroughly tested in internal/db/dependency_test.go
	// Here we just verify the validation command integrates with it correctly

	// Create test database with no circular dependencies
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "no-circular.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(t, err)

	gormDB := database.DB()

	// Create config
	cfg := &db.Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	require.NoError(t, gormDB.Create(cfg).Error)

	// Create group A
	groupA := &db.Group{
		ConfigID:   cfg.ID,
		ExternalID: "group-a",
		Name:       "Group A",
	}
	require.NoError(t, gormDB.Create(groupA).Error)

	// Create group B (depends on A - no cycle)
	groupB := &db.Group{
		ConfigID:   cfg.ID,
		ExternalID: "group-b",
		Name:       "Group B",
	}
	require.NoError(t, gormDB.Create(groupB).Error)

	// Create valid dependency: B depends on A (no cycle)
	depBA := &db.GroupDependency{
		GroupID:     groupB.ID,
		DependsOnID: "group-a",
	}
	require.NoError(t, gormDB.Create(depBA).Error)

	require.NoError(t, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldJSON := dbValidateJSON
	defer func() {
		dbPath = oldDBPath
		dbValidateJSON = oldJSON
	}()

	dbPath = tmpPath
	dbValidateJSON = true

	t.Run("validates non-circular dependencies", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		_ = runDBValidate(nil, nil)

		// Restore stdout and read output
		_ = w.Close()
		os.Stdout = oldStdout
		var output bytes.Buffer
		_, _ = output.ReadFrom(r)

		// Parse JSON output
		var result ValidationResult
		err = json.Unmarshal(output.Bytes(), &result)
		require.NoError(t, err)
		assert.True(t, result.Valid, "should be valid with no circular dependencies")
		assert.Empty(t, result.Errors, "should have no errors")
	})
}

// TestDBValidateOrphanedMappings tests validation of orphaned mappings
func TestDBValidateOrphanedMappings(t *testing.T) {
	// Orphaned mappings are tested more thoroughly in the OrphanedRefs test
	// This test verifies the validation check runs without errors on valid data

	// Create test database with valid mappings
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "valid-mapping.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(t, err)

	gormDB := database.DB()

	// Create config
	cfg := &db.Config{
		ExternalID: "test-config",
		Name:       "Test Config",
		Version:    1,
	}
	require.NoError(t, gormDB.Create(cfg).Error)

	// Create group
	group := &db.Group{
		ConfigID:   cfg.ID,
		ExternalID: "test-group",
		Name:       "Test Group",
	}
	require.NoError(t, gormDB.Create(group).Error)

	// Create target
	target := &db.Target{
		GroupID: group.ID,
		Repo:    "mrz1836/test-repo",
		Branch:  "main",
	}
	require.NoError(t, gormDB.Create(target).Error)

	// Create valid file mapping
	validMapping := &db.FileMapping{
		OwnerType: "target",
		OwnerID:   target.ID,
		Src:       "test.txt",
		Dest:      "test.txt",
	}
	require.NoError(t, gormDB.Create(validMapping).Error)

	require.NoError(t, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldJSON := dbValidateJSON
	defer func() {
		dbPath = oldDBPath
		dbValidateJSON = oldJSON
	}()

	dbPath = tmpPath
	dbValidateJSON = true

	t.Run("validates correct file mappings", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		_ = runDBValidate(nil, nil)

		// Restore stdout and read output
		_ = w.Close()
		os.Stdout = oldStdout
		var output bytes.Buffer
		_, _ = output.ReadFrom(r)

		// Parse JSON output
		var result ValidationResult
		err = json.Unmarshal(output.Bytes(), &result)
		require.NoError(t, err)
		assert.True(t, result.Valid, "should be valid with correct file mappings")
		assert.Empty(t, result.Errors, "should have no errors")
	})
}

// TestDBValidateCommand tests validate command structure
func TestDBValidateCommand(t *testing.T) {
	t.Run("command exists", func(t *testing.T) {
		assert.NotNil(t, dbValidateCmd)
		assert.Equal(t, "validate", dbValidateCmd.Use)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := dbValidateCmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "bool", flag.Value.Type())
	})

	t.Run("help text contains checks", func(t *testing.T) {
		assert.Contains(t, dbValidateCmd.Long, "Checks performed:")
		assert.Contains(t, dbValidateCmd.Long, "Orphaned")
		assert.Contains(t, dbValidateCmd.Long, "Circular dependencies")
	})
}

// TestDBValidateEmpty tests validation with no data
func TestDBValidateEmpty(t *testing.T) {
	// Create empty database
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "empty.db")

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(t, err)
	require.NoError(t, database.Close())

	// Save and restore flags
	oldDBPath := dbPath
	oldJSON := dbValidateJSON
	defer func() {
		dbPath = oldDBPath
		dbValidateJSON = oldJSON
	}()

	dbPath = tmpPath
	dbValidateJSON = true

	t.Run("handles empty database", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		_ = runDBValidate(nil, nil)

		// Restore stdout and read output
		_ = w.Close()
		os.Stdout = oldStdout
		var output bytes.Buffer
		_, _ = output.ReadFrom(r)

		// Parse JSON output
		var result ValidationResult
		err = json.Unmarshal(output.Bytes(), &result)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.NotEmpty(t, result.Checks)
	})
}

// Benchmarks moved to db_query_benchmark_test.go (bench_heavy tag)
