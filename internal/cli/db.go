package cli

import (
	"github.com/spf13/cobra"

	"github.com/mrz1836/go-broadcast/internal/db"
)

//nolint:gochecknoglobals // Cobra commands are designed to be global variables
var (
	dbPath string
	dbCmd  = &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
		Long: `Manage the go-broadcast configuration database.

The database provides structured storage for configuration with:
• SQLite backend with WAL mode for concurrent access
• Write-time validation
• Queryable configuration (which repos sync this file?)
• Import/export to/from YAML for backwards compatibility

Common workflows:
  # Initialize a new database
  go-broadcast db init

  # Check database status
  go-broadcast db status

  # Import existing YAML config
  go-broadcast config import sync.yaml

  # Query configuration
  go-broadcast config query --file .github/workflows/ci.yml`,
	}
)

//nolint:gochecknoinits // Cobra commands require init() for flag registration
func init() {
	// Add subcommands
	dbCmd.AddCommand(dbInitCmd)
	dbCmd.AddCommand(dbStatusCmd)
}

// getDBPath returns the database path, using default if not specified
func getDBPath() string {
	if dbPath == "" {
		return db.DefaultPath()
	}
	return dbPath
}
