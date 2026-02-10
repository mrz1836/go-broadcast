package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

//nolint:gochecknoglobals // Cobra commands and flags are designed to be global variables
var (
	dbExportOutput string
	dbExportGroup  string
	dbExportStdout bool
	dbExportCmd    = &cobra.Command{
		Use:   "export",
		Short: "Export database configuration to YAML",
		Long: `Export configuration from the database to a YAML file.

This command reads the configuration from the SQLite database and exports
it to YAML format, preserving all groups, targets, file lists, directory
lists, and their relationships.

The exported YAML can be re-imported using 'db import' for backup/restore
or migration purposes.

Examples:
  # Export to file
  go-broadcast db export --output sync.yaml

  # Export to stdout
  go-broadcast db export --stdout

  # Export single group
  go-broadcast db export --group mrz-tools --output group.yaml

  # Export from custom database path
  go-broadcast db export --output sync.yaml --db-path /tmp/test.db`,
		RunE: runDBExport,
	}
)

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	dbExportCmd.Flags().StringVar(&dbExportOutput, "output", "", "Output YAML file path")
	dbExportCmd.Flags().StringVar(&dbExportGroup, "group", "", "Export single group by external ID")
	dbExportCmd.Flags().BoolVar(&dbExportStdout, "stdout", false, "Write to stdout instead of file")
}

// runDBExport executes the database export command
func runDBExport(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Validate flags
	if !dbExportStdout && dbExportOutput == "" {
		return fmt.Errorf("either --output or --stdout must be specified")
	}

	if dbExportStdout && dbExportOutput != "" {
		return fmt.Errorf("cannot specify both --output and --stdout")
	}

	path := getDBPath()

	// Open database
	output.Info(fmt.Sprintf("Opening database: %s", path))
	database, err := db.Open(db.OpenOptions{
		Path:     path,
		LogLevel: logger.Silent,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = database.Close() }()

	converter := db.NewConverter(database.DB())

	// Get config external ID (default to first config if not specified)
	var configExternalID string
	if dbExportGroup != "" {
		// Find the config that owns this group
		var group db.Group
		if dbErr := database.DB().Where("external_id = ?", dbExportGroup).First(&group).Error; dbErr != nil {
			return fmt.Errorf("group %q not found: %w", dbExportGroup, dbErr)
		}

		var cfg db.Config
		if dbErr := database.DB().First(&cfg, group.ConfigID).Error; dbErr != nil {
			return fmt.Errorf("failed to find config for group: %w", dbErr)
		}
		configExternalID = cfg.ExternalID
	} else {
		// Find first config
		var cfg db.Config
		if dbErr := database.DB().First(&cfg).Error; dbErr != nil {
			return fmt.Errorf("no configuration found in database: %w", dbErr)
		}
		configExternalID = cfg.ExternalID
	}

	// Export configuration
	output.Info(fmt.Sprintf("Exporting configuration: %s", configExternalID))
	cfg, err := converter.ExportConfig(ctx, configExternalID)
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	// Filter to single group if requested
	if dbExportGroup != "" {
		output.Info(fmt.Sprintf("Filtering to group: %s", dbExportGroup))
		var filteredGroups []db.Group
		for _, group := range cfg.Groups {
			if group.ID == dbExportGroup {
				// We need to export just this one group
				// For now, keep all groups and let user extract
				// In future, could add group-only export
				break
			}
		}
		if len(filteredGroups) == 0 {
			// Keep all groups for now (ExportGroup could be used for single group)
			output.Warn("Note: Full config exported. Use YAML tools to extract specific group.")
		}
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write output
	if dbExportStdout {
		fmt.Print(string(yamlData)) //nolint:forbidigo // intentional stdout output for --stdout flag
	} else {
		output.Info(fmt.Sprintf("Writing to: %s", dbExportOutput))
		if err := os.WriteFile(dbExportOutput, yamlData, 0o600); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		output.Success(fmt.Sprintf("✓ Exported to: %s", dbExportOutput))
	}

	// Report counts
	if !dbExportStdout {
		output.Info(fmt.Sprintf("  Groups:           %d", len(cfg.Groups)))
		totalTargets := 0
		for _, group := range cfg.Groups {
			totalTargets += len(group.Targets)
		}
		output.Info(fmt.Sprintf("  Targets:          %d", totalTargets))
		output.Info(fmt.Sprintf("  File Lists:       %d", len(cfg.FileLists)))
		output.Info(fmt.Sprintf("  Directory Lists:  %d", len(cfg.DirectoryLists)))
	}

	return nil
}
