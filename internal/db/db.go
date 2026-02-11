// Package db provides database operations for go-broadcast configuration storage.
// It supports SQLite with GORM for structured config management.
// Package db provides database abstraction and models for go-broadcast configuration storage.
// It supports SQLite with WAL mode and provides GORM-based models for all configuration entities.
package db

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database is the interface for database operations
type Database interface {
	DB() *gorm.DB
	Close() error
	AutoMigrate() error
}

// database is the concrete implementation
type database struct {
	db *gorm.DB
}

// DB returns the underlying GORM database
func (d *database) DB() *gorm.DB {
	return d.db
}

// Close closes the database connection
func (d *database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// AutoMigrate runs schema migrations
func (d *database) AutoMigrate() error {
	return AutoMigrate(d.db)
}

// OpenOptions holds options for opening a database
type OpenOptions struct {
	Path        string          // Database file path (required)
	LogLevel    logger.LogLevel // GORM log level (default: Silent)
	AutoMigrate bool            // Run auto-migration on open (default: false)
}

// ErrEmptyPath is returned when database path is empty
var ErrEmptyPath = fmt.Errorf("database path is required")

// Open opens a database connection with the given options
// Supports SQLite only for now (Postgres support deferred to future)
func Open(opts OpenOptions) (Database, error) {
	if opts.Path == "" {
		return nil, ErrEmptyPath
	}

	// Create parent directory if it doesn't exist (unless :memory:)
	if opts.Path != ":memory:" {
		dir := filepath.Dir(opts.Path)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return nil, fmt.Errorf("failed to create database directory: %w", err)
			}
		}
	}

	// Open SQLite database
	config := SQLiteConfig{
		Path:     opts.Path,
		LogLevel: opts.LogLevel,
	}

	db, err := OpenSQLite(config)
	if err != nil {
		return nil, err
	}

	d := &database{db: db}

	// Run auto-migration if requested
	if opts.AutoMigrate {
		if err := d.AutoMigrate(); err != nil {
			_ = d.Close()
			return nil, fmt.Errorf("auto-migration failed: %w", err)
		}
	}

	return d, nil
}

// DefaultPath returns the default database path
// ~/.config/go-broadcast/broadcast.db
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".config/go-broadcast/broadcast.db"
	}
	return filepath.Join(home, ".config", "go-broadcast", "broadcast.db")
}
