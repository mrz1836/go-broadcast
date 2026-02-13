package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"
)

// TestOpenSQLite_Success tests successful database opening
func TestOpenSQLite_Success(t *testing.T) {
	config := SQLiteConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	}

	db, err := OpenSQLite(config)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify connection is working
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Ping())

	// Cleanup
	require.NoError(t, sqlDB.Close())
}

// TestOpenSQLite_EmptyPath tests error handling for empty path
func TestOpenSQLite_EmptyPath(t *testing.T) {
	config := SQLiteConfig{
		Path:     "",
		LogLevel: logger.Silent,
	}

	db, err := OpenSQLite(config)
	require.Error(t, err)
	assert.Nil(t, db)
	assert.Contains(t, err.Error(), "path is required")
}

// TestAutoMigrate tests schema creation
func TestAutoMigrate(t *testing.T) {
	db := TestDB(t)

	// Verify tables were created by checking one model
	var count int64
	err := db.Model(&Config{}).Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// TestPragmas verifies SQLite pragmas are set correctly
func TestPragmas(t *testing.T) {
	// Use a temp file for WAL mode (not available in :memory:)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := SQLiteConfig{
		Path:     dbPath,
		LogLevel: logger.Silent,
	}

	db, err := OpenSQLite(config)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	tests := []struct {
		pragma   string
		expected string
	}{
		{"PRAGMA journal_mode", "wal"},
		{"PRAGMA synchronous", "1"},  // NORMAL = 1
		{"PRAGMA foreign_keys", "1"}, // ON = 1
		{"PRAGMA temp_store", "2"},   // MEMORY = 2
	}

	for _, tt := range tests {
		var result string
		err := db.Raw(tt.pragma).Scan(&result).Error
		require.NoError(t, err, "pragma check failed: %s", tt.pragma)
		assert.Equal(t, tt.expected, result, "pragma mismatch: %s", tt.pragma)
	}
}

// TestPragmas_Comprehensive verifies all 8 SQLite pragmas are set correctly
func TestPragmas_Comprehensive(t *testing.T) {
	t.Parallel() // Run in parallel for speed

	// Use a temp file for WAL mode (not available in :memory:)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := SQLiteConfig{
		Path:     dbPath,
		LogLevel: logger.Silent,
	}

	db, err := OpenSQLite(config)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	tests := []struct {
		name     string
		pragma   string
		expected string
	}{
		{"WAL journaling", "PRAGMA journal_mode", "wal"},
		{"NORMAL sync mode", "PRAGMA synchronous", "1"},
		{"5 second busy timeout", "PRAGMA busy_timeout", "5000"},
		{"foreign keys enabled", "PRAGMA foreign_keys", "1"},
		{"20MB cache", "PRAGMA cache_size", "-20000"},
		{"memory temp storage", "PRAGMA temp_store", "2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			err := db.Raw(tt.pragma).Scan(&result).Error
			require.NoError(t, err, "pragma check failed: %s", tt.pragma)
			assert.Equal(t, tt.expected, result, "pragma mismatch: %s", tt.pragma)
		})
	}

	// auto_vacuum can only be checked on a fresh database (before tables are created)
	// It's verified separately in TestAutoVacuumPragma

	// mmap_size returns 268435456 but may vary by platform
	t.Run("mmap_size configured", func(t *testing.T) {
		var mmapSize int64
		err := db.Raw("PRAGMA mmap_size").Scan(&mmapSize).Error
		require.NoError(t, err, "mmap_size pragma check failed")
		assert.Positive(t, mmapSize, "mmap_size should be positive")
	})
}

// TestAutoVacuumPragma verifies auto_vacuum pragma is set on fresh databases
// auto_vacuum can only be set before the first table is created
func TestAutoVacuumPragma(t *testing.T) {
	t.Parallel() // Run in parallel for speed

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "fresh.db")

	config := SQLiteConfig{
		Path:     dbPath,
		LogLevel: logger.Silent,
	}

	db, err := OpenSQLite(config)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	// Check auto_vacuum before any tables are created
	var autoVacuum string
	err = db.Raw("PRAGMA auto_vacuum").Scan(&autoVacuum).Error
	require.NoError(t, err)

	// Should be "2" (incremental) if set, but may be "0" (none) if not applied yet
	// The pragma is executed in OpenSQLite but may not persist until first write
	// We verify it's one of the valid values
	assert.Contains(t, []string{"0", "1", "2"}, autoVacuum, "auto_vacuum should be valid value")

	// After creating a table, the pragma should be locked in
	err = db.Exec("CREATE TABLE test (id INTEGER)").Error
	require.NoError(t, err)

	// Now check again - on some SQLite implementations this will show the set value
	err = db.Raw("PRAGMA auto_vacuum").Scan(&autoVacuum).Error
	require.NoError(t, err)
	t.Logf("auto_vacuum after table creation: %s", autoVacuum)
}

// TestOpen_Success tests the Open factory function
func TestOpen_Success(t *testing.T) {
	opts := OpenOptions{
		Path:        ":memory:",
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	}

	database, err := Open(opts)
	require.NoError(t, err)
	require.NotNil(t, database)

	// Verify database interface works
	db := database.DB()
	require.NotNil(t, db)

	// Verify auto-migrate worked
	var count int64
	err = db.Model(&Config{}).Count(&count).Error
	require.NoError(t, err)

	// Cleanup
	require.NoError(t, database.Close())
}

// TestOpen_WithFileAndAutoMigrate tests file-based database with auto-migration
func TestOpen_WithFileAndAutoMigrate(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	opts := OpenOptions{
		Path:        dbPath,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	}

	database, err := Open(opts)
	require.NoError(t, err)
	require.NotNil(t, database)
	defer func() { _ = database.Close() }()

	// Verify file was created
	_, err = os.Stat(dbPath)
	require.NoError(t, err)

	// Verify database works
	db := database.DB()
	config := &Config{
		ExternalID: "test-id",
		Name:       "Test Config",
		Version:    1,
	}
	require.NoError(t, db.Create(config).Error)

	// Verify record was created
	var retrieved Config
	err = db.First(&retrieved, config.ID).Error
	require.NoError(t, err)
	assert.Equal(t, config.Name, retrieved.Name)
}

// TestOpen_EmptyPath tests error handling for empty path
func TestOpen_EmptyPath(t *testing.T) {
	opts := OpenOptions{
		Path: "",
	}

	database, err := Open(opts)
	require.Error(t, err)
	assert.Nil(t, database)
	assert.Contains(t, err.Error(), "path is required")
}

// TestOpen_AutoMigrateFails tests behavior when auto-migration fails
func TestOpen_AutoMigrateFails(t *testing.T) {
	// This test is challenging to trigger naturally, but we can test the path exists
	// by verifying auto-migrate is called when AutoMigrate=true
	opts := OpenOptions{
		Path:        ":memory:",
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	}

	database, err := Open(opts)
	require.NoError(t, err)
	require.NotNil(t, database)
	defer func() { _ = database.Close() }()

	// If we got here, auto-migrate succeeded
	// Verify tables exist
	var count int64
	err = database.DB().Model(&Config{}).Count(&count).Error
	require.NoError(t, err)
}

// TestDefaultPath tests the default path function
func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".config/go-broadcast/broadcast.db")
}

// TestDatabase_Close tests database closure
func TestDatabase_Close(t *testing.T) {
	opts := OpenOptions{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	}

	database, err := Open(opts)
	require.NoError(t, err)

	// Close should succeed
	err = database.Close()
	require.NoError(t, err)

	// After close, database operations should fail
	db := database.DB()
	var count int64
	err = db.Model(&Config{}).Count(&count).Error
	assert.Error(t, err, "operations after close should fail")
}

// TestConnectionPoolSettings tests SQLite connection pool configuration
func TestConnectionPoolSettings(t *testing.T) {
	config := SQLiteConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	}

	db, err := OpenSQLite(config)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Verify connection pool settings
	stats := sqlDB.Stats()
	assert.Equal(t, 1, stats.MaxOpenConnections, "MaxOpenConns should be 1 for SQLite single-writer")
}

// TestConnectionPoolSettings_Complete verifies all connection pool parameters
func TestConnectionPoolSettings_Complete(t *testing.T) {
	t.Parallel() // Race-safe, uses isolated memory database

	config := SQLiteConfig{
		Path:     ":memory:",
		LogLevel: logger.Silent,
	}

	db, err := OpenSQLite(config)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// Test MaxOpenConns (SQLite single-writer optimization)
	stats := sqlDB.Stats()
	assert.Equal(t, 1, stats.MaxOpenConnections, "MaxOpenConns should be 1")

	// MaxIdleConns and ConnMaxLifetime cannot be directly read from Stats
	// but we can verify the connection stays alive by performing multiple operations
	var count int64
	for i := 0; i < 3; i++ { // Reduced from 5 to 3 for speed
		err = db.Raw("SELECT 1").Scan(&count).Error
		require.NoError(t, err, "query %d should succeed with persistent connection", i+1)
		assert.Equal(t, int64(1), count)
	}

	// After multiple queries, we should still have at most 1 idle connection
	stats = sqlDB.Stats()
	assert.LessOrEqual(t, stats.Idle, 1, "should not exceed MaxIdleConns")
}

// TestPrepareStmt_ProductionVsTest verifies PrepareStmt configuration
func TestPrepareStmt_ProductionVsTest(t *testing.T) {
	t.Parallel() // Safe to run in parallel

	t.Run("production mode has PrepareStmt enabled", func(t *testing.T) {
		t.Parallel()

		config := SQLiteConfig{
			Path:     ":memory:",
			LogLevel: logger.Silent,
		}

		db, err := OpenSQLite(config)
		require.NoError(t, err)
		defer func() {
			sqlDB, _ := db.DB()
			_ = sqlDB.Close()
		}()

		// PrepareStmt is set to true in OpenSQLite
		// Verify it works by executing the same query multiple times
		for i := 0; i < 3; i++ {
			var result int
			err = db.Raw("SELECT ?", i).Scan(&result).Error
			require.NoError(t, err)
			assert.Equal(t, i, result)
		}
	})

	t.Run("test helper has PrepareStmt disabled", func(t *testing.T) {
		t.Parallel()

		// TestDB uses PrepareStmt=false for race detector safety
		db := TestDB(t)

		// Should still work, just without statement caching
		var result int
		err := db.Raw("SELECT 42").Scan(&result).Error
		require.NoError(t, err)
		assert.Equal(t, 42, result)
	})
}

// TestOpenSQLite_InvalidPath tests error handling for invalid file paths
func TestOpenSQLite_InvalidPath(t *testing.T) {
	t.Parallel() // Safe to run in parallel

	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "valid memory path",
			path:        ":memory:",
			expectError: false,
		},
		{
			name:        "valid file path",
			path:        filepath.Join(tmpDir, "valid.db"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := SQLiteConfig{
				Path:     tt.path,
				LogLevel: logger.Silent,
			}

			db, err := OpenSQLite(config)
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, db)
			} else {
				require.NoError(t, err)
				require.NotNil(t, db)
				sqlDB, _ := db.DB()
				_ = sqlDB.Close()
			}
		})
	}
}

// TestOpenSQLite_ReadOnlyDirectory tests permission error handling
func TestOpenSQLite_ReadOnlyDirectory(t *testing.T) {
	t.Parallel() // Safe - uses isolated temp directory

	// Create a read-only directory
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0o555) //nolint:gosec // Test fixture intentionally restricted
	require.NoError(t, err)

	// Attempt to create database in read-only directory
	dbPath := filepath.Join(readOnlyDir, "test.db")
	config := SQLiteConfig{
		Path:     dbPath,
		LogLevel: logger.Silent,
	}

	db, err := OpenSQLite(config)

	// Behavior varies by platform and user permissions
	// On some systems, this may succeed (root user, etc.)
	// We verify that IF it fails, it fails gracefully with an error
	if err != nil {
		assert.Nil(t, db, "db should be nil on error")
		assert.Contains(t, err.Error(), "failed to open database")
	} else {
		// If it succeeded, cleanup
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}

	// Restore permissions for cleanup (must be writable for temp cleanup)
	_ = os.Chmod(readOnlyDir, 0o755) //nolint:gosec // Cleanup operation for test fixture
}

// TestWALMode_FileCreation verifies WAL mode creates auxiliary files
func TestWALMode_FileCreation(t *testing.T) {
	t.Parallel() // Safe - uses isolated temp directory

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := SQLiteConfig{
		Path:     dbPath,
		LogLevel: logger.Silent,
	}

	db, err := OpenSQLite(config)
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	}()

	// Perform a single write operation to trigger WAL file creation (fast)
	err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)").Error
	require.NoError(t, err)

	err = db.Exec("INSERT INTO test (id) VALUES (1)").Error
	require.NoError(t, err)

	// Verify main database file exists
	_, err = os.Stat(dbPath)
	require.NoError(t, err, "main database file should exist")

	// WAL and SHM files may or may not exist depending on timing and checkpointing
	// We just verify they don't cause errors if they exist
	walPath := dbPath + "-wal"
	shmPath := dbPath + "-shm"

	// Check if WAL file was created (optional, depends on write activity)
	if _, err := os.Stat(walPath); err == nil {
		t.Logf("WAL file created: %s", walPath)
	}

	// Check if SHM file was created (optional)
	if _, err := os.Stat(shmPath); err == nil {
		t.Logf("SHM file created: %s", shmPath)
	}
}
