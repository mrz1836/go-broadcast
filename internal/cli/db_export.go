package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

//nolint:gochecknoglobals // Cobra commands and flags are designed to be global variables
var (
	dbExportOutput string
	dbExportGroup  string
	dbExportStdout bool
	dbExportFormat = exportFormatYAML
	dbExportCmd    = &cobra.Command{
		Use:   "export",
		Short: "Export database configuration to YAML or JSON",
		Long: `Export configuration from the database to a YAML or JSON file.

This command reads the configuration from the SQLite database and exports
it in the configuration file format, preserving all groups, targets, file
lists, directory lists, and their relationships.

The format is YAML by default; --format json writes the same document as
JSON, with the same keys in the same order, for programs that read JSON.
Either can be re-imported using 'db import' for backup/restore or
migration purposes.

With --stdout, the document is the only thing written to stdout: progress
messages go to stderr, so the output can be piped or parsed as is.

Examples:
  # Export to file
  go-broadcast db export --output sync.yaml

  # Export to stdout
  go-broadcast db export --stdout

  # Export to stdout as JSON
  go-broadcast db export --stdout --format json

  # Export single group
  go-broadcast db export --group my-tools --output group.yaml

  # Export from custom database path
  go-broadcast db export --output sync.yaml --db-path /tmp/test.db`,
		RunE: runDBExport,
	}
)

// Export formats accepted by --format.
const (
	exportFormatYAML = "yaml"
	exportFormatJSON = "json"
)

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	dbExportCmd.Flags().StringVar(&dbExportOutput, "output", "", "Output file path")
	dbExportCmd.Flags().StringVar(&dbExportGroup, "group", "", "Export single group by external ID")
	dbExportCmd.Flags().BoolVar(&dbExportStdout, "stdout", false, "Write to stdout instead of file (only the document goes to stdout)")
	dbExportCmd.Flags().StringVar(&dbExportFormat, "format", exportFormatYAML, "Output format: yaml or json")
}

// runDBExport executes the database export command
func runDBExport(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Validate flags
	if !dbExportStdout && dbExportOutput == "" {
		return fmt.Errorf("either --output or --stdout must be specified") //nolint:err113 // user-facing CLI validation error
	}

	if dbExportStdout && dbExportOutput != "" {
		return fmt.Errorf("cannot specify both --output and --stdout") //nolint:err113 // user-facing CLI validation error
	}

	format := strings.ToLower(strings.TrimSpace(dbExportFormat))
	if format != exportFormatYAML && format != exportFormatJSON {
		return fmt.Errorf("%w: %q (use %s or %s)", ErrUnsupportedExportFormat, dbExportFormat, exportFormatYAML, exportFormatJSON)
	}

	progress := exportLog()
	path := getDBPath()

	// Open database
	progress.Info(fmt.Sprintf("Opening database: %s", path))
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
	var cfg *config.Config
	if dbExportGroup != "" {
		// Export single group
		progress.Info(fmt.Sprintf("Exporting group: %s", dbExportGroup))

		var dbCfg db.Config
		if dbErr := database.DB().Where("external_id = ?", configExternalID).First(&dbCfg).Error; dbErr != nil {
			return fmt.Errorf("failed to find config: %w", dbErr)
		}

		group, gErr := converter.ExportGroup(ctx, dbCfg.ID, dbExportGroup)
		if gErr != nil {
			return fmt.Errorf("failed to export group %q: %w", dbExportGroup, gErr)
		}

		cfg = &config.Config{
			Version: 1,
			Groups:  []config.Group{*group},
		}
	} else {
		// Export full config
		progress.Info(fmt.Sprintf("Exporting configuration: %s", configExternalID))
		var eErr error
		cfg, eErr = converter.ExportConfig(ctx, configExternalID)
		if eErr != nil {
			return fmt.Errorf("export failed: %w", eErr)
		}
	}

	// Encode the document
	data, err := encodeExport(cfg, format)
	if err != nil {
		return err
	}

	// Write output
	if dbExportStdout {
		if _, err := output.Stdout().Write(data); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
	} else {
		progress.Info(fmt.Sprintf("Writing to: %s", dbExportOutput))
		if err := os.WriteFile(dbExportOutput, data, 0o600); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		progress.Success(fmt.Sprintf("✓ Exported to: %s", dbExportOutput))
	}

	// Report counts
	if !dbExportStdout {
		progress.Info(fmt.Sprintf("  Groups:           %d", len(cfg.Groups)))
		totalTargets := 0
		for _, group := range cfg.Groups {
			totalTargets += len(group.Targets)
		}
		progress.Info(fmt.Sprintf("  Targets:          %d", totalTargets))
		progress.Info(fmt.Sprintf("  File Lists:       %d", len(cfg.FileLists)))
		progress.Info(fmt.Sprintf("  Directory Lists:  %d", len(cfg.DirectoryLists)))
	}

	return nil
}

// exportLog is where export's progress messages go: stdout, unless the
// document itself goes to stdout (--stdout), in which case they go to stderr
// so that stdout carries the document alone.
func exportLog() output.Writer {
	if dbExportStdout {
		return output.NewColoredWriter(output.Stderr(), output.Stderr())
	}
	return output.NewColoredWriter(output.Stdout(), output.Stderr())
}

// encodeExport encodes the exported configuration in the requested format.
func encodeExport(cfg *config.Config, format string) ([]byte, error) {
	if format == exportFormatJSON {
		data, err := configToJSON(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return data, nil
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return data, nil
}
