package db

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Test-specific errors for simulating migration failures
var (
	errForcedMigrationFailure = fmt.Errorf("forced migration failure")
	errMigrationFailed        = fmt.Errorf("migration failed")
	errRollbackFailed         = fmt.Errorf("rollback failed intentionally")
)

// TestMigrationManager_ApplySingle tests applying a single migration
func TestMigrationManager_ApplySingle(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration := Migration{
		Version:     "001_create_test_table",
		Description: "Create test table",
		Up: func(tx *gorm.DB) error {
			return tx.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)").Error
		},
		Down: func(tx *gorm.DB) error {
			return tx.Exec("DROP TABLE test_table").Error
		},
	}

	mgr.Register(migration)

	// Apply migration
	err := mgr.Apply()
	require.NoError(t, err)

	// Verify table was created
	var count int64
	err = database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify SchemaMigration record was created
	var schemaMigration SchemaMigration
	err = database.Where("version = ?", "001_create_test_table").First(&schemaMigration).Error
	require.NoError(t, err)
	assert.Equal(t, "001_create_test_table", schemaMigration.Version)
	assert.Equal(t, "Create test table", schemaMigration.Description)
	assert.NotEmpty(t, schemaMigration.Checksum)
	assert.NotZero(t, schemaMigration.AppliedAt)
}

// TestMigrationManager_ApplyMultiple tests applying multiple migrations in order
func TestMigrationManager_ApplyMultiple(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration1 := Migration{
		Version:     "001_first",
		Description: "First migration",
		Up: func(tx *gorm.DB) error {
			return tx.Exec("CREATE TABLE table1 (id INTEGER PRIMARY KEY)").Error
		},
	}

	migration2 := Migration{
		Version:     "002_second",
		Description: "Second migration",
		Up: func(tx *gorm.DB) error {
			return tx.Exec("CREATE TABLE table2 (id INTEGER PRIMARY KEY)").Error
		},
	}

	migration3 := Migration{
		Version:     "003_third",
		Description: "Third migration",
		Up: func(tx *gorm.DB) error {
			return tx.Exec("CREATE TABLE table3 (id INTEGER PRIMARY KEY)").Error
		},
	}

	mgr.Register(migration1)
	mgr.Register(migration2)
	mgr.Register(migration3)

	// Apply all migrations
	err := mgr.Apply()
	require.NoError(t, err)

	// Verify all tables were created
	for i := 1; i <= 3; i++ {
		tableName := "table" + string(rune('0'+i))
		var count int64
		err = database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "table %s should exist", tableName)
	}

	// Verify all migration records exist
	versions, err := mgr.AppliedMigrations()
	require.NoError(t, err)
	assert.Len(t, versions, 3)
	assert.Equal(t, "001_first", versions[0])
	assert.Equal(t, "002_second", versions[1])
	assert.Equal(t, "003_third", versions[2])
}

// TestMigrationManager_SkipApplied tests that applied migrations are skipped
func TestMigrationManager_SkipApplied(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	var callCount int32
	migration := Migration{
		Version:     "001_test",
		Description: "Test migration",
		Up: func(tx *gorm.DB) error {
			atomic.AddInt32(&callCount, 1)
			return tx.Exec("CREATE TABLE test_skip (id INTEGER)").Error
		},
	}

	mgr.Register(migration)

	// Apply first time
	err := mgr.Apply()
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount), "migration should run once")

	// Apply again
	err = mgr.Apply()
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount), "migration should not run again")
}

// TestMigrationManager_ChecksumCalculation tests checksum generation
func TestMigrationManager_ChecksumCalculation(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration := Migration{
		Version:     "001_checksum_test",
		Description: "Checksum test migration",
		Up: func(_ *gorm.DB) error {
			return nil
		},
	}

	mgr.Register(migration)
	err := mgr.Apply()
	require.NoError(t, err)

	// Get the stored checksum
	var schemaMigration SchemaMigration
	err = database.Where("version = ?", "001_checksum_test").First(&schemaMigration).Error
	require.NoError(t, err)

	// Verify checksum format (should be 64-character hex string for SHA256)
	assert.Len(t, schemaMigration.Checksum, 64, "SHA256 checksum should be 64 hex characters")
	assert.Regexp(t, "^[a-f0-9]{64}$", schemaMigration.Checksum, "checksum should be lowercase hex")

	// Verify checksum is consistent
	database2 := TestDB(t)
	mgr2 := NewMigrationManager(database2)
	mgr2.Register(migration)
	err = mgr2.Apply()
	require.NoError(t, err)

	var schemaMigration2 SchemaMigration
	err = database2.Where("version = ?", "001_checksum_test").First(&schemaMigration2).Error
	require.NoError(t, err)

	assert.Equal(t, schemaMigration.Checksum, schemaMigration2.Checksum, "same migration should have same checksum")
}

// TestMigrationManager_TransactionRollback tests rollback on migration failure
func TestMigrationManager_TransactionRollback(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration := Migration{
		Version:     "001_rollback_test",
		Description: "Test rollback",
		Up: func(tx *gorm.DB) error {
			// Create table successfully
			err := tx.Exec("CREATE TABLE temp_table (id INTEGER)").Error
			if err != nil {
				return err
			}

			// Force an error
			return errForcedMigrationFailure
		},
	}

	mgr.Register(migration)

	// Apply should fail
	err := mgr.Apply()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "forced migration failure")

	// Verify table was NOT created (transaction rolled back)
	var count int64
	err = database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='temp_table'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "table should not exist after rollback")
}

// TestMigrationManager_TransactionRollbackNoRecord tests that no SchemaMigration record is created on failure
func TestMigrationManager_TransactionRollbackNoRecord(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration := Migration{
		Version:     "001_no_record",
		Description: "Should not create record",
		Up: func(_ *gorm.DB) error {
			return errMigrationFailed
		},
	}

	mgr.Register(migration)

	// Apply should fail
	err := mgr.Apply()
	require.Error(t, err)

	// Verify no SchemaMigration record was created
	var count int64
	err = database.Model(&SchemaMigration{}).Where("version = ?", "001_no_record").Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "no migration record should exist")

	// Verify IsMigrationApplied returns false
	applied, err := mgr.IsMigrationApplied("001_no_record")
	require.NoError(t, err)
	assert.False(t, applied)
}

// TestMigrationManager_IsMigrationApplied tests checking migration status
func TestMigrationManager_IsMigrationApplied(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration := Migration{
		Version:     "001_applied_check",
		Description: "Check if applied",
		Up: func(*gorm.DB) error {
			return nil
		},
	}

	// Before applying
	applied, err := mgr.IsMigrationApplied("001_applied_check")
	require.NoError(t, err)
	assert.False(t, applied, "migration should not be applied yet")

	// Register and apply
	mgr.Register(migration)
	err = mgr.Apply()
	require.NoError(t, err)

	// After applying
	applied, err = mgr.IsMigrationApplied("001_applied_check")
	require.NoError(t, err)
	assert.True(t, applied, "migration should be applied")

	// Check non-existent migration
	applied, err = mgr.IsMigrationApplied("999_nonexistent")
	require.NoError(t, err)
	assert.False(t, applied, "non-existent migration should not be applied")
}

// TestMigrationManager_AppliedMigrations tests listing applied migrations
func TestMigrationManager_AppliedMigrations(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	// Initially empty
	versions, err := mgr.AppliedMigrations()
	require.NoError(t, err)
	assert.Empty(t, versions)

	// Apply migrations
	for i := 1; i <= 3; i++ {
		version := "00" + string(rune('0'+i)) + "_test"
		migration := Migration{
			Version:     version,
			Description: "Test migration " + string(rune('0'+i)),
			Up: func(*gorm.DB) error {
				return nil
			},
		}
		mgr.Register(migration)
	}

	err = mgr.Apply()
	require.NoError(t, err)

	// Verify order (should be ordered by applied_at ASC)
	versions, err = mgr.AppliedMigrations()
	require.NoError(t, err)
	require.Len(t, versions, 3)
	assert.Equal(t, "001_test", versions[0])
	assert.Equal(t, "002_test", versions[1])
	assert.Equal(t, "003_test", versions[2])
}

// TestMigrationManager_RollbackSuccess tests successful rollback
func TestMigrationManager_RollbackSuccess(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	var downCalled int32
	migration := Migration{
		Version:     "001_rollback_success",
		Description: "Test rollback",
		Up: func(tx *gorm.DB) error {
			return tx.Exec("CREATE TABLE rollback_test (id INTEGER)").Error
		},
		Down: func(tx *gorm.DB) error {
			atomic.StoreInt32(&downCalled, 1)
			return tx.Exec("DROP TABLE rollback_test").Error
		},
	}

	mgr.Register(migration)

	// Apply migration
	err := mgr.Apply()
	require.NoError(t, err)

	// Verify table exists
	var count int64
	err = database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='rollback_test'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Rollback
	err = mgr.Rollback()
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&downCalled), "Down function should be called")

	// Verify table was dropped
	err = database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='rollback_test'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "table should be dropped after rollback")
}

// TestMigrationManager_RollbackDeletesRecord tests that SchemaMigration record is deleted
func TestMigrationManager_RollbackDeletesRecord(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration := Migration{
		Version:     "001_delete_record",
		Description: "Test record deletion",
		Up: func(*gorm.DB) error {
			return nil
		},
		Down: func(*gorm.DB) error {
			return nil
		},
	}

	mgr.Register(migration)

	// Apply
	err := mgr.Apply()
	require.NoError(t, err)

	// Verify record exists
	var count int64
	err = database.Model(&SchemaMigration{}).Where("version = ?", "001_delete_record").Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Rollback
	err = mgr.Rollback()
	require.NoError(t, err)

	// Verify record was deleted
	err = database.Model(&SchemaMigration{}).Where("version = ?", "001_delete_record").Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "migration record should be deleted")
}

// TestMigrationManager_RollbackNoMigrations tests rollback with no migrations
func TestMigrationManager_RollbackNoMigrations(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	// Try to rollback with no migrations
	err := mgr.Rollback()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no migrations to rollback")
}

// TestMigrationManager_RollbackMigrationNotFound tests rollback when migration definition not found
func TestMigrationManager_RollbackMigrationNotFound(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration := Migration{
		Version:     "001_orphan",
		Description: "Orphan migration",
		Up: func(*gorm.DB) error {
			return nil
		},
		Down: func(*gorm.DB) error {
			return nil
		},
	}

	mgr.Register(migration)
	err := mgr.Apply()
	require.NoError(t, err)

	// Create new manager without registering the migration
	mgr2 := NewMigrationManager(database)

	// Rollback should fail
	err = mgr2.Rollback()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrMigrationNotFound)
	assert.Contains(t, err.Error(), "001_orphan")
}

// TestMigrationManager_RollbackNoDownFunction tests rollback when Down is nil
func TestMigrationManager_RollbackNoDownFunction(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration := Migration{
		Version:     "001_no_down",
		Description: "No down function",
		Up: func(_ *gorm.DB) error {
			return nil
		},
		Down: nil, // No rollback function
	}

	mgr.Register(migration)
	err := mgr.Apply()
	require.NoError(t, err)

	// Rollback should fail
	err = mgr.Rollback()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoDownFunction)
	assert.Contains(t, err.Error(), "001_no_down")
}

// TestMigrationManager_RollbackTransactionFailure tests rollback transaction failure
func TestMigrationManager_RollbackTransactionFailure(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration := Migration{
		Version:     "001_rollback_fail",
		Description: "Rollback failure test",
		Up: func(tx *gorm.DB) error {
			return tx.Exec("CREATE TABLE rollback_fail (id INTEGER)").Error
		},
		Down: func(tx *gorm.DB) error {
			// Try to drop table and then fail
			_ = tx.Exec("DROP TABLE rollback_fail").Error
			return errRollbackFailed
		},
	}

	mgr.Register(migration)

	// Apply
	err := mgr.Apply()
	require.NoError(t, err)

	// Rollback should fail
	err = mgr.Rollback()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rollback failed intentionally")

	// Due to transaction rollback, the table should still exist
	// (the DROP TABLE in Down() was rolled back)
	var count int64
	err = database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='rollback_fail'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "table should still exist after failed rollback transaction")

	// Migration record should still exist (deletion was rolled back)
	err = database.Model(&SchemaMigration{}).Where("version = ?", "001_rollback_fail").Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "migration record should still exist")
}

// TestMigrationManager_RollbackLastMigration tests rolling back the last of multiple migrations
func TestMigrationManager_RollbackLastMigration(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migrations := []Migration{
		{
			Version:     "001_first",
			Description: "First",
			Up: func(tx *gorm.DB) error {
				return tx.Exec("CREATE TABLE first (id INTEGER)").Error
			},
			Down: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE first").Error
			},
		},
		{
			Version:     "002_second",
			Description: "Second",
			Up: func(tx *gorm.DB) error {
				return tx.Exec("CREATE TABLE second (id INTEGER)").Error
			},
			Down: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE second").Error
			},
		},
		{
			Version:     "003_third",
			Description: "Third",
			Up: func(tx *gorm.DB) error {
				return tx.Exec("CREATE TABLE third (id INTEGER)").Error
			},
			Down: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE third").Error
			},
		},
	}

	for _, m := range migrations {
		mgr.Register(m)
	}

	// Apply all
	err := mgr.Apply()
	require.NoError(t, err)

	// Rollback last migration
	err = mgr.Rollback()
	require.NoError(t, err)

	// Verify third table is gone
	var count int64
	err = database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='third'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "third table should be removed")

	// Verify first and second still exist
	err = database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='first'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "first table should still exist")

	err = database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='second'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "second table should still exist")

	// Verify applied migrations
	versions, err := mgr.AppliedMigrations()
	require.NoError(t, err)
	assert.Len(t, versions, 2)
	assert.Equal(t, "001_first", versions[0])
	assert.Equal(t, "002_second", versions[1])
}

// TestMigrationManager_ComplexScenario tests a complete migration workflow
func TestMigrationManager_ComplexScenario(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	// Register 3 migrations
	migrations := []Migration{
		{
			Version:     "001_users",
			Description: "Create users table",
			Up: func(tx *gorm.DB) error {
				return tx.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)").Error
			},
			Down: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE users").Error
			},
		},
		{
			Version:     "002_posts",
			Description: "Create posts table",
			Up: func(tx *gorm.DB) error {
				return tx.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER, title TEXT)").Error
			},
			Down: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE posts").Error
			},
		},
		{
			Version:     "003_comments",
			Description: "Create comments table",
			Up: func(tx *gorm.DB) error {
				return tx.Exec("CREATE TABLE comments (id INTEGER PRIMARY KEY, post_id INTEGER, text TEXT)").Error
			},
			Down: func(tx *gorm.DB) error {
				return tx.Exec("DROP TABLE comments").Error
			},
		},
	}

	for _, m := range migrations {
		mgr.Register(m)
	}

	// Apply all migrations
	err := mgr.Apply()
	require.NoError(t, err)

	// Verify all applied
	versions, err := mgr.AppliedMigrations()
	require.NoError(t, err)
	assert.Len(t, versions, 3)

	// Rollback last
	err = mgr.Rollback()
	require.NoError(t, err)

	versions, err = mgr.AppliedMigrations()
	require.NoError(t, err)
	assert.Len(t, versions, 2)

	// Apply again (should apply only the third migration)
	err = mgr.Apply()
	require.NoError(t, err)

	versions, err = mgr.AppliedMigrations()
	require.NoError(t, err)
	assert.Len(t, versions, 3)

	// Verify all tables exist
	for _, tableName := range []string{"users", "posts", "comments"} {
		var count int64
		err = database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "table %s should exist", tableName)
	}
}

// TestMigrationManager_EmptyUp tests migration with empty Up function
func TestMigrationManager_EmptyUp(t *testing.T) {
	t.Parallel()

	database := TestDB(t)
	mgr := NewMigrationManager(database)

	migration := Migration{
		Version:     "001_empty",
		Description: "Empty migration",
		Up: func(*gorm.DB) error {
			// Do nothing
			return nil
		},
	}

	mgr.Register(migration)
	err := mgr.Apply()
	require.NoError(t, err)

	// Verify record was created
	applied, err := mgr.IsMigrationApplied("001_empty")
	require.NoError(t, err)
	assert.True(t, applied)
}

// TestMigrationManager_ConcurrentApply tests that Apply is safe to call concurrently
func TestMigrationManager_ConcurrentApply(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// This test runs goroutines concurrently internally, so t.Parallel() is not used
	// to avoid resource contention with other database tests running in parallel

	database := TestDB(t)

	// Create the migration definition outside the loop
	// so each goroutine references the same migration object
	migration := Migration{
		Version:     "001_concurrent",
		Description: "Concurrent test",
		Up: func(tx *gorm.DB) error {
			return tx.Exec("CREATE TABLE concurrent_test (id INTEGER)").Error
		},
	}

	// Try to apply same migration concurrently
	// One should succeed, others should skip (already applied)
	done := make(chan error, 3)

	// Each goroutine gets its own migration manager
	// They share the same database connection, which is safe in SQLite with WAL mode
	for i := 0; i < 3; i++ {
		go func() {
			mgr := NewMigrationManager(database)
			// Create a copy of the migration to avoid any potential races
			migrationCopy := Migration{
				Version:     migration.Version,
				Description: migration.Description,
				Up:          migration.Up,
				Down:        migration.Down,
			}
			mgr.Register(migrationCopy)
			done <- mgr.Apply()
		}()
	}

	// Collect results
	var successCount int32
	for i := 0; i < 3; i++ {
		err := <-done
		// In concurrent scenarios, some might succeed and some might skip (already applied)
		// We just verify no fatal errors occurred
		if err == nil {
			atomic.AddInt32(&successCount, 1)
		}
	}

	// All should succeed (first applies, others skip because already applied)
	assert.Equal(t, int32(3), atomic.LoadInt32(&successCount), "all Apply calls should succeed (first applies, rest skip)")

	// Verify table exists
	var count int64
	err := database.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='concurrent_test'").Scan(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
