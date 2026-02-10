package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

//nolint:gochecknoglobals // Cobra commands and flags are designed to be global variables
var (
	dbImportYAML  string
	dbImportForce bool
	dbImportCmd   = &cobra.Command{
		Use:   "import",
		Short: "Import YAML configuration into database",
		Long: `Import configuration from a YAML file into the database.

This command loads a YAML configuration file and imports it into the
SQLite database, converting all groups, targets, file lists, and directory
lists into database records.

The import process is transactional - if any error occurs during import,
all changes are rolled back and the database remains unchanged.

Examples:
  # Import from default sync.yaml
  go-broadcast db import

  # Import from specific YAML file
  go-broadcast db import --yaml my-config.yaml

  # Force import (replace existing config with same ID)
  go-broadcast db import --force

  # Import to custom database path
  go-broadcast db import --yaml sync.yaml --db-path /tmp/test.db`,
		RunE: runDBImport,
	}
)

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	dbImportCmd.Flags().StringVar(&dbImportYAML, "yaml", "sync.yaml", "Path to YAML configuration file")
	dbImportCmd.Flags().BoolVar(&dbImportForce, "force", false, "Force import (replace existing config)")
}

// runDBImport executes the database import command
func runDBImport(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	path := getDBPath()

	// Load YAML configuration
	output.Info(fmt.Sprintf("Loading YAML configuration: %s", dbImportYAML))
	cfg, err := config.Load(dbImportYAML)
	if err != nil {
		return fmt.Errorf("failed to load YAML: %w", err)
	}

	// Validate configuration before import
	output.Info("Validating configuration...")
	if err := cfg.ValidateWithLogging(ctx, nil); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Open database
	output.Info(fmt.Sprintf("Opening database: %s", path))
	database, err := db.Open(db.OpenOptions{
		Path:        path,
		LogLevel:    logger.Silent,
		AutoMigrate: true,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// Check if config already exists
	var existingConfig db.Config
	result := database.DB().Where("external_id = ?", cfg.ID).First(&existingConfig)
	if result.Error == nil && !dbImportForce {
		return fmt.Errorf("config %q already exists (use --force to replace)", cfg.ID)
	}

	// Import configuration
	output.Info(fmt.Sprintf("Importing configuration: %s (ID: %s)", cfg.Name, cfg.ID))
	converter := db.NewConverter(database.DB())
	dbConfig, err := converter.ImportConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Count imported records
	var groupCount, targetCount, fileListCount, dirListCount int64
	database.DB().Model(&db.Group{}).Where("config_id = ?", dbConfig.ID).Count(&groupCount)
	database.DB().Model(&db.Target{}).Joins("JOIN groups ON targets.group_id = groups.id").
		Where("groups.config_id = ?", dbConfig.ID).Count(&targetCount)
	database.DB().Model(&db.FileList{}).Where("config_id = ?", dbConfig.ID).Count(&fileListCount)
	database.DB().Model(&db.DirectoryList{}).Where("config_id = ?", dbConfig.ID).Count(&dirListCount)

	// Report success
	output.Success(fmt.Sprintf("✓ Import completed successfully"))
	output.Info(fmt.Sprintf("  Config:           %s (v%d)", dbConfig.Name, dbConfig.Version))
	output.Info(fmt.Sprintf("  Groups:           %d", groupCount))
	output.Info(fmt.Sprintf("  Targets:          %d", targetCount))
	output.Info(fmt.Sprintf("  File Lists:       %d", fileListCount))
	output.Info(fmt.Sprintf("  Directory Lists:  %d", dirListCount))

	return nil
}
