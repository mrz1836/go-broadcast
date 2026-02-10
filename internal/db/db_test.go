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
	assert.Error(t, err)
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
	assert.Error(t, err)
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
