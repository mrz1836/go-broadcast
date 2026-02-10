package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

//nolint:gochecknoglobals // Cobra commands and flags are designed to be global variables
var (
	dbInitForce bool
	dbInitCmd   = &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		Long: `Initialize a new go-broadcast configuration database.

Creates the database file and directory structure, then runs schema migrations.
By default, fails if the database already exists. Use --force to recreate.

Examples:
  # Initialize at default path (~/.config/go-broadcast/broadcast.db)
  go-broadcast db init

  # Initialize at custom path
  go-broadcast db init --path /tmp/my-config.db

  # Force recreation (WARNING: destroys existing data)
  go-broadcast db init --force`,
		RunE: runDBInit,
	}
)

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	dbInitCmd.Flags().BoolVar(&dbInitForce, "force", false, "Force recreation (destroys existing data)")
}

// runDBInit executes the database initialization
func runDBInit(_ *cobra.Command, _ []string) error {
	path := getDBPath()

	// Check if database exists
	if _, err := os.Stat(path); err == nil && !dbInitForce {
		return fmt.Errorf("database already exists at %s (use --force to recreate)", path) //nolint:err113 // user-facing CLI error
	}

	// Remove existing database if force flag is set
	if dbInitForce {
		if _, err := os.Stat(path); err == nil {
			output.Info(fmt.Sprintf("Removing existing database: %s", path))
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove existing database: %w", err)
			}
		}
	}

	// Create parent directory
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		output.Info(fmt.Sprintf("Creating directory: %s", dir))
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Open database with auto-migration
	output.Info(fmt.Sprintf("Initializing database: %s", path))
	database, err := db.Open(db.OpenOptions{
		Path:        path,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() { _ = database.Close() }()

	output.Success(fmt.Sprintf("✓ Database initialized: %s", path))
	return nil
}
