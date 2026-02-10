package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/output"
)

//nolint:gochecknoglobals // Cobra commands and flags are designed to be global variables
var (
	dbStatusJSON bool
	dbStatusCmd  = &cobra.Command{
		Use:   "status",
		Short: "Show database status",
		Long: `Display database status including version, table counts, and last sync time.

Shows:
• Database path and file size
• Schema version
• Table counts (configs, groups, targets, file lists, etc.)
• Last modification time

Output formats:
• Human-readable (default)
• JSON (--json flag)

Examples:
  # Show status (human-readable)
  go-broadcast db status

  # Show status as JSON
  go-broadcast db status --json

  # Status for custom database
  go-broadcast db status --path /tmp/my-config.db`,
		RunE: runDBStatus,
	}
)

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	dbStatusCmd.Flags().BoolVar(&dbStatusJSON, "json", false, "Output as JSON")
}

// DBStatus represents database status information
type DBStatus struct {
	Path         string           `json:"path"`
	Exists       bool             `json:"exists"`
	Size         int64            `json:"size_bytes,omitempty"`
	LastModified *time.Time       `json:"last_modified,omitempty"`
	Version      string           `json:"version,omitempty"`
	TableCounts  map[string]int64 `json:"table_counts,omitempty"`
	Error        string           `json:"error,omitempty"`
}

// runDBStatus executes the database status command
func runDBStatus(_ *cobra.Command, _ []string) error {
	path := getDBPath()
	status := DBStatus{
		Path:        path,
		TableCounts: make(map[string]int64),
	}

	// Check if database file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			status.Exists = false
			status.Error = "database does not exist (run 'go-broadcast db init' to create)"
			return printStatus(status)
		}
		status.Error = fmt.Sprintf("failed to stat database: %v", err)
		return printStatus(status)
	}

	status.Exists = true
	status.Size = info.Size()
	modTime := info.ModTime()
	status.LastModified = &modTime

	// Open database
	database, err := db.Open(db.OpenOptions{
		Path:     path,
		LogLevel: logger.Silent,
	})
	if err != nil {
		status.Error = fmt.Sprintf("failed to open database: %v", err)
		return printStatus(status)
	}
	defer func() { _ = database.Close() }()

	gormDB := database.DB()

	// Get schema version
	var migration db.SchemaMigration
	if err := gormDB.Order("applied_at DESC").First(&migration).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			status.Error = fmt.Sprintf("failed to get schema version: %v", err)
			return printStatus(status)
		}
		status.Version = "unknown"
	} else {
		status.Version = migration.Version
	}

	// Get table counts
	tables := map[string]interface{}{
		"configs":                    &db.Config{},
		"groups":                     &db.Group{},
		"sources":                    &db.Source{},
		"targets":                    &db.Target{},
		"file_lists":                 &db.FileList{},
		"directory_lists":            &db.DirectoryList{},
		"file_mappings":              &db.FileMapping{},
		"directory_mappings":         &db.DirectoryMapping{},
		"transforms":                 &db.Transform{},
		"group_dependencies":         &db.GroupDependency{},
		"group_globals":              &db.GroupGlobal{},
		"group_defaults":             &db.GroupDefault{},
		"target_file_list_refs":      &db.TargetFileListRef{},
		"target_directory_list_refs": &db.TargetDirectoryListRef{},
	}

	for tableName, model := range tables {
		var count int64
		if err := gormDB.Model(model).Count(&count).Error; err != nil {
			output.Warn(fmt.Sprintf("Failed to count %s: %v", tableName, err))
			continue
		}
		status.TableCounts[tableName] = count
	}

	return printStatus(status)
}

// printStatus prints the database status in the appropriate format
func printStatus(status DBStatus) error {
	if dbStatusJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(status)
	}

	// Human-readable output
	output.Info(fmt.Sprintf("Database: %s", status.Path))

	if !status.Exists {
		output.Error(status.Error)
		return fmt.Errorf("database does not exist") //nolint:err113 // user-facing CLI error
	}

	if status.Error != "" {
		output.Error(status.Error)
		return fmt.Errorf("database error: %s", status.Error) //nolint:err113 // user-facing CLI error
	}

	output.Info(fmt.Sprintf("Size: %.2f KB", float64(status.Size)/1024))
	if status.LastModified != nil {
		output.Info(fmt.Sprintf("Last Modified: %s", status.LastModified.Format(time.RFC3339)))
	}
	output.Info(fmt.Sprintf("Schema Version: %s", status.Version))

	// Print table counts (showing all tables for visibility)
	output.Info("\nTable Counts:")
	// Define ordered list of tables for consistent display
	tableOrder := []string{
		"configs", "groups", "sources", "targets",
		"file_lists", "directory_lists",
		"file_mappings", "directory_mappings",
		"transforms",
		"group_dependencies", "group_globals", "group_defaults",
		"target_file_list_refs", "target_directory_list_refs",
	}
	for _, table := range tableOrder {
		if count, exists := status.TableCounts[table]; exists {
			output.Info(fmt.Sprintf("  %-30s %d", table+":", count))
		}
	}

	return nil
}
