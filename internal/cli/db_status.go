package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
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
• All table counts (automatically discovered)
• Last modification time
• Hierarchy: clients → organizations → repos

Output formats:
• Human-readable (default) - tables grouped logically
• JSON (--json flag) - all tables with counts

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
	err = gormDB.Order("applied_at DESC").First(&migration).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			status.Error = fmt.Sprintf("failed to get schema version: %v", err)
			return printStatus(status)
		}
		status.Version = "unknown"
	} else {
		status.Version = migration.Version
	}

	// Get all tables dynamically using GORM Migrator
	allTables, err := gormDB.Migrator().GetTables()
	if err != nil {
		status.Error = fmt.Sprintf("failed to get tables: %v", err)
		return printStatus(status)
	}

	// Filter out SQLite internal tables (sqlite_*)
	userTables := filterUserTables(allTables)

	// Count records for each table
	for _, tableName := range userTables {
		var count int64
		if err := gormDB.Table(tableName).Count(&count).Error; err != nil {
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
	// Get tables in logical display order
	orderedTables := orderTables(status.TableCounts)
	for _, table := range orderedTables {
		count := status.TableCounts[table]
		output.Info(fmt.Sprintf("  %-30s %d", table+":", count))
	}

	return nil
}

// filterUserTables removes internal SQLite tables
func filterUserTables(tables []string) []string {
	var filtered []string
	for _, table := range tables {
		// Skip SQLite internal tables
		if strings.HasPrefix(table, "sqlite_") {
			continue
		}
		filtered = append(filtered, table)
	}
	return filtered
}

// orderTables returns tables in logical display order, with unknown tables at the end alphabetically
func orderTables(tableCounts map[string]int64) []string {
	// Define logical ordering for known tables (hierarchy + grouped by purpose)
	preferredOrder := []string{
		// Hierarchy (top-down)
		"clients",
		"organizations",
		"repos",

		// Configuration
		"configs",
		"groups",
		"group_dependencies",
		"group_globals",
		"group_defaults",

		// Sources and Targets
		"sources",
		"targets",

		// File/Directory Lists
		"file_lists",
		"directory_lists",

		// Mappings and Transforms
		"file_mappings",
		"directory_mappings",
		"transforms",

		// Join Tables
		"target_file_list_refs",
		"target_directory_list_refs",

		// System
		"schema_migrations",
	}

	ordered := make([]string, 0, len(tableCounts))
	seen := make(map[string]bool)

	// Add tables in preferred order (if they exist)
	for _, table := range preferredOrder {
		if _, exists := tableCounts[table]; exists {
			ordered = append(ordered, table)
			seen[table] = true
		}
	}

	// Add any remaining tables alphabetically (future-proofing)
	remaining := make([]string, 0)
	for table := range tableCounts {
		if !seen[table] {
			remaining = append(remaining, table)
		}
	}
	sort.Strings(remaining)
	ordered = append(ordered, remaining...)

	return ordered
}
