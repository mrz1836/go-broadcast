package db

import (
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SQLiteConfig holds configuration options for SQLite connection
type SQLiteConfig struct {
	Path     string          // Database file path (:memory: for in-memory)
	LogLevel logger.LogLevel // GORM log level
}

// OpenSQLite opens a SQLite database with production-optimized settings
func OpenSQLite(config SQLiteConfig) (*gorm.DB, error) {
	if config.Path == "" {
		return nil, ErrEmptyPath
	}

	// Build DSN with pragmas
	// Using github.com/glebarez/sqlite (pure Go, no CGO required)
	dsn := config.Path

	// Open database
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(config.LogLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		// PrepareStmt improves performance by caching prepared statements
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Get underlying SQL DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database: %w", err)
	}

	// SQLite single-writer optimization
	// MaxOpenConns=1 prevents write contention (SQLite allows single writer)
	// MaxIdleConns=1 keeps connection alive for faster subsequent queries
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Apply SQLite pragmas for production use
	pragmas := []string{
		"PRAGMA journal_mode=WAL",        // Write-Ahead Logging: concurrent reads while writing
		"PRAGMA synchronous=NORMAL",      // Good durability with WAL (faster than FULL)
		"PRAGMA busy_timeout=5000",       // 5s lock contention timeout
		"PRAGMA foreign_keys=ON",         // Enforce FK constraints
		"PRAGMA cache_size=-20000",       // 20MB page cache (negative = KB)
		"PRAGMA temp_store=MEMORY",       // Temp tables in memory (faster)
		"PRAGMA mmap_size=268435456",     // 256MB memory-mapped I/O (faster reads)
		"PRAGMA auto_vacuum=INCREMENTAL", // Reclaim space gradually
	}

	for _, pragma := range pragmas {
		if err := db.Exec(pragma).Error; err != nil {
			return nil, fmt.Errorf("failed to set pragma %q: %w", pragma, err)
		}
	}

	return db, nil
}

// AutoMigrate runs auto-migration for all models
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// Core models from T-19
		&Client{},
		&Organization{},
		&Repo{},
		&Config{},
		&Group{},
		&GroupDependency{},
		&Source{},
		&GroupGlobal{},
		&GroupDefault{},
		&Target{},
		&FileList{},
		&DirectoryList{},
		&FileMapping{},
		&DirectoryMapping{},
		&Transform{},
		&TargetFileListRef{},
		&TargetDirectoryListRef{},
		&SchemaMigration{},
		// Analytics models from T-22
		&RepositorySnapshot{},
		&SecurityAlert{},
		&SyncRun{},
		&CIMetricsSnapshot{},
		// Broadcast sync metrics models from T-98
		&BroadcastSyncRun{},
		&BroadcastSyncTargetResult{},
		&BroadcastSyncFileChange{},
	)
}
