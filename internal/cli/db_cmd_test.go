package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// TestDBInit tests the db init command
func TestDBInit(t *testing.T) {
	t.Run("creates new database", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "test.db")

		// Save and restore original dbPath and flag
		oldDBPath := dbPath
		oldForce := dbInitForce
		defer func() {
			dbPath = oldDBPath
			dbInitForce = oldForce
		}()

		dbPath = tmpPath
		dbInitForce = false

		// Run init directly
		err := runDBInit(nil, nil)
		require.NoError(t, err)

		// Verify database file exists
		_, err = os.Stat(tmpPath)
		require.NoError(t, err, "database file should exist")
	})

	t.Run("fails if database already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "existing.db")

		// Create existing database
		_, err := os.Create(tmpPath) //nolint:gosec // test file in temp directory
		require.NoError(t, err)

		// Save and restore
		oldDBPath := dbPath
		oldForce := dbInitForce
		defer func() {
			dbPath = oldDBPath
			dbInitForce = oldForce
		}()

		dbPath = tmpPath
		dbInitForce = false

		// Run init (should fail)
		err = runDBInit(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("force flag recreates database", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "force.db")

		// Create existing database
		require.NoError(t, os.WriteFile(tmpPath, []byte("old data"), 0o600))

		// Save and restore
		oldDBPath := dbPath
		oldForce := dbInitForce
		defer func() {
			dbPath = oldDBPath
			dbInitForce = oldForce
		}()

		dbPath = tmpPath
		dbInitForce = true

		// Run init with force
		err := runDBInit(nil, nil)
		require.NoError(t, err)

		// Verify database was recreated (should be SQLite file, not "old data")
		content, err := os.ReadFile(tmpPath) //nolint:gosec // test file in temp directory
		require.NoError(t, err)
		assert.NotEqual(t, "old data", string(content))
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "nested", "dirs", "test.db")

		// Save and restore
		oldDBPath := dbPath
		oldForce := dbInitForce
		defer func() {
			dbPath = oldDBPath
			dbInitForce = oldForce
		}()

		dbPath = tmpPath
		dbInitForce = false

		// Run init
		err := runDBInit(nil, nil)
		require.NoError(t, err)

		// Verify all directories were created
		_, err = os.Stat(filepath.Dir(tmpPath))
		require.NoError(t, err)
	})

	t.Run("initial migration record created", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "versioned.db")

		// Save and restore
		oldDBPath := dbPath
		oldForce := dbInitForce
		defer func() {
			dbPath = oldDBPath
			dbInitForce = oldForce
		}()

		dbPath = tmpPath
		dbInitForce = false

		// Run init
		err := runDBInit(nil, nil)
		require.NoError(t, err)

		// Verify migration record exists
		database, err := db.Open(db.OpenOptions{Path: tmpPath})
		require.NoError(t, err)
		defer func() { _ = database.Close() }()

		var migration db.SchemaMigration
		err = database.DB().Order("applied_at DESC").First(&migration).Error
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", migration.Version)
		assert.Equal(t, "Initial schema via AutoMigrate", migration.Description)
		assert.NotEmpty(t, migration.Checksum)
	})
}

// TestDBStatus tests the db status command
func TestDBStatus(t *testing.T) {
	t.Run("shows status for existing database", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "status.db")

		// Create and initialize database
		database, err := db.Open(db.OpenOptions{
			Path:        tmpPath,
			AutoMigrate: true,
		})
		require.NoError(t, err)
		require.NoError(t, database.Close())

		// Save and restore
		oldDBPath := dbPath
		oldJSON := dbStatusJSON
		defer func() {
			dbPath = oldDBPath
			dbStatusJSON = oldJSON
		}()

		dbPath = tmpPath
		dbStatusJSON = false

		// Run status
		err = runDBStatus(nil, nil)
		require.NoError(t, err)
	})

	t.Run("reports missing database", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "nonexistent.db")

		// Save and restore
		oldDBPath := dbPath
		oldJSON := dbStatusJSON
		defer func() {
			dbPath = oldDBPath
			dbStatusJSON = oldJSON
		}()

		dbPath = tmpPath
		dbStatusJSON = false

		// Run status (should report missing)
		err := runDBStatus(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("JSON output format", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "json.db")

		// Create and initialize database
		database, err := db.Open(db.OpenOptions{
			Path:        tmpPath,
			AutoMigrate: true,
		})
		require.NoError(t, err)

		// Add some test data
		gormDB := database.DB()
		cfg := &db.Config{
			ExternalID: "test-config",
			Name:       "Test Config",
			Version:    1,
		}
		require.NoError(t, gormDB.Create(cfg).Error)
		require.NoError(t, database.Close())

		// Save and restore
		oldDBPath := dbPath
		oldJSON := dbStatusJSON
		oldStdout := os.Stdout
		defer func() {
			dbPath = oldDBPath
			dbStatusJSON = oldJSON
			os.Stdout = oldStdout
		}()

		dbPath = tmpPath
		dbStatusJSON = true

		// Capture stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run status
		err = runDBStatus(nil, nil)
		require.NoError(t, err)

		// Restore stdout and read output
		_ = w.Close()
		os.Stdout = oldStdout
		var output bytes.Buffer
		_, _ = output.ReadFrom(r)

		// Parse JSON output
		var status DBStatus
		err = json.Unmarshal(output.Bytes(), &status)
		require.NoError(t, err)
		assert.Equal(t, tmpPath, status.Path)
		assert.True(t, status.Exists)
		assert.Positive(t, status.TableCounts["configs"])
	})

	t.Run("shows version after init", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "version.db")

		// Save and restore
		oldDBPath := dbPath
		oldForce := dbInitForce
		oldJSON := dbStatusJSON
		oldStdout := os.Stdout
		defer func() {
			dbPath = oldDBPath
			dbInitForce = oldForce
			dbStatusJSON = oldJSON
			os.Stdout = oldStdout
		}()

		// Initialize database using init command
		dbPath = tmpPath
		dbInitForce = false
		err := runDBInit(nil, nil)
		require.NoError(t, err)

		// Run status with JSON output
		dbStatusJSON = true

		// Capture stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err = runDBStatus(nil, nil)
		require.NoError(t, err)

		// Restore stdout and read output
		_ = w.Close()
		os.Stdout = oldStdout
		var output bytes.Buffer
		_, _ = output.ReadFrom(r)

		// Parse JSON output
		var status DBStatus
		err = json.Unmarshal(output.Bytes(), &status)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", status.Version)
	})
}

// TestDBFlagParsing tests the --db-path flag
func TestDBFlagParsing(t *testing.T) {
	t.Run("uses default path when not specified", func(t *testing.T) {
		// Save and restore
		oldPath := dbPath
		defer func() { dbPath = oldPath }()

		dbPath = ""

		path := getDBPath()
		assert.NotEmpty(t, path)
		assert.Contains(t, path, ".config/go-broadcast/broadcast.db")
	})

	t.Run("uses specified path", func(t *testing.T) {
		customPath := "/tmp/custom.db"

		// Save and restore
		oldPath := dbPath
		defer func() { dbPath = oldPath }()

		dbPath = customPath

		path := getDBPath()
		assert.Equal(t, customPath, path)
	})
}

// TestGetDBPath tests the getDBPath helper function
func TestGetDBPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func() string
	}{
		{
			name:  "empty path returns default",
			input: "",
			expected: func() string {
				return db.DefaultPath()
			},
		},
		{
			name:  "custom path returns custom",
			input: "/custom/path/db.sqlite",
			expected: func() string {
				return "/custom/path/db.sqlite"
			},
		},
		{
			name:  "relative path preserved",
			input: "./local.db",
			expected: func() string {
				return "./local.db"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore
			oldPath := dbPath
			defer func() { dbPath = oldPath }()

			dbPath = tt.input

			result := getDBPath()
			expected := tt.expected()
			assert.Equal(t, expected, result)
		})
	}
}

// TestDBCommandIntegration tests the full command integration
func TestDBCommandIntegration(t *testing.T) {
	t.Run("db command exists", func(t *testing.T) {
		assert.NotNil(t, dbCmd)
		assert.Equal(t, "db", dbCmd.Use)
	})

	t.Run("db init subcommand exists", func(t *testing.T) {
		assert.NotNil(t, dbInitCmd)
		assert.Equal(t, "init", dbInitCmd.Use)
	})

	t.Run("db status subcommand exists", func(t *testing.T) {
		assert.NotNil(t, dbStatusCmd)
		assert.Equal(t, "status", dbStatusCmd.Use)
	})

	t.Run("db command has subcommands", func(t *testing.T) {
		subcommands := dbCmd.Commands()
		assert.GreaterOrEqual(t, len(subcommands), 7, "should have at least 7 subcommands")

		// Find all expected commands
		var hasInit, hasStatus, hasImport, hasExport, hasDiff, hasQuery, hasValidate bool
		for _, cmd := range subcommands {
			switch cmd.Use {
			case "init":
				hasInit = true
			case "status":
				hasStatus = true
			case "import":
				hasImport = true
			case "export":
				hasExport = true
			case "diff":
				hasDiff = true
			case "query":
				hasQuery = true
			case "validate":
				hasValidate = true
			}
		}

		assert.True(t, hasInit, "should have init subcommand")
		assert.True(t, hasStatus, "should have status subcommand")
		assert.True(t, hasImport, "should have import subcommand")
		assert.True(t, hasExport, "should have export subcommand")
		assert.True(t, hasDiff, "should have diff subcommand")
		assert.True(t, hasQuery, "should have query subcommand")
		assert.True(t, hasValidate, "should have validate subcommand")
	})

	t.Run("init command has force flag", func(t *testing.T) {
		flag := dbInitCmd.Flags().Lookup("force")
		assert.NotNil(t, flag)
		assert.Equal(t, "bool", flag.Value.Type())
	})

	t.Run("status command has json flag", func(t *testing.T) {
		flag := dbStatusCmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "bool", flag.Value.Type())
	})
}

// TestDBInitValidation tests init command validation
func TestDBInitValidation(t *testing.T) {
	t.Run("accepts memory database path", func(t *testing.T) {
		// Memory databases are allowed in the db package, verify the behavior
		memPath := ":memory:"

		oldDBPath := dbPath
		oldForce := dbInitForce
		defer func() {
			dbPath = oldDBPath
			dbInitForce = oldForce
		}()

		dbPath = memPath
		dbInitForce = false

		// This should succeed as :memory: is valid for SQLite
		err := runDBInit(nil, nil)
		require.NoError(t, err)
	})
}

// TestDBStatusValidation tests status command validation
func TestDBStatusValidation(t *testing.T) {
	t.Run("handles corrupted database", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "corrupt.db")

		// Create a corrupted database file
		require.NoError(t, os.WriteFile(tmpPath, []byte("not a valid sqlite database"), 0o600))

		oldDBPath := dbPath
		oldJSON := dbStatusJSON
		defer func() {
			dbPath = oldDBPath
			dbStatusJSON = oldJSON
		}()

		dbPath = tmpPath
		dbStatusJSON = false

		// Should report error but not panic
		err := runDBStatus(nil, nil)
		require.Error(t, err)
	})
}

// TestDBCommandHelp tests help text
func TestDBCommandHelp(t *testing.T) {
	t.Run("db help contains expected text", func(t *testing.T) {
		assert.Contains(t, dbCmd.Long, "SQLite")
		assert.Contains(t, dbCmd.Long, "database")
	})

	t.Run("init help contains expected text", func(t *testing.T) {
		assert.Contains(t, dbInitCmd.Long, "Initialize")
		assert.Contains(t, dbInitCmd.Long, "--force")
	})

	t.Run("status help contains expected text", func(t *testing.T) {
		assert.Contains(t, dbStatusCmd.Long, "status")
		assert.Contains(t, dbStatusCmd.Long, "JSON")
	})
}

// TestFilterUserTables tests the filterUserTables helper function
func TestFilterUserTables(t *testing.T) {
	t.Run("filters out SQLite internal tables", func(t *testing.T) {
		input := []string{
			"clients",
			"sqlite_sequence",
			"configs",
			"sqlite_master",
			"groups",
			"sqlite_temp_master",
		}

		expected := []string{
			"clients",
			"configs",
			"groups",
		}

		result := filterUserTables(input)
		assert.Equal(t, expected, result)
	})

	t.Run("handles empty input", func(t *testing.T) {
		input := []string{}
		result := filterUserTables(input)
		assert.Empty(t, result)
	})

	t.Run("handles all SQLite internal tables", func(t *testing.T) {
		input := []string{
			"sqlite_sequence",
			"sqlite_master",
			"sqlite_temp_master",
		}
		result := filterUserTables(input)
		assert.Empty(t, result)
	})

	t.Run("passes through user tables unchanged", func(t *testing.T) {
		input := []string{
			"clients",
			"organizations",
			"repos",
			"schema_migrations",
		}
		expected := input
		result := filterUserTables(input)
		assert.Equal(t, expected, result)
	})
}

// TestOrderTables tests the orderTables helper function
func TestOrderTables(t *testing.T) {
	t.Run("orders hierarchy tables first", func(t *testing.T) {
		input := map[string]int64{
			"configs":       5,
			"repos":         3,
			"clients":       1,
			"organizations": 2,
			"groups":        4,
		}

		result := orderTables(input)

		// Hierarchy tables should be first
		assert.Equal(t, "clients", result[0])
		assert.Equal(t, "organizations", result[1])
		assert.Equal(t, "repos", result[2])

		// Config tables should follow
		assert.Equal(t, "configs", result[3])
		assert.Equal(t, "groups", result[4])
	})

	t.Run("places unknown tables at end alphabetically", func(t *testing.T) {
		input := map[string]int64{
			"clients":      1,
			"unknown_zulu": 99,
			"configs":      5,
			"unknown_alfa": 98,
		}

		result := orderTables(input)

		// Known tables first
		assert.Equal(t, "clients", result[0])
		assert.Equal(t, "configs", result[1])

		// Unknown tables at end, alphabetically
		assert.Equal(t, "unknown_alfa", result[2])
		assert.Equal(t, "unknown_zulu", result[3])
	})

	t.Run("handles all known tables in correct order", func(t *testing.T) {
		// Create a map with all known tables
		input := map[string]int64{
			"clients":                    1,
			"organizations":              2,
			"repos":                      3,
			"configs":                    4,
			"groups":                     5,
			"group_dependencies":         6,
			"group_globals":              7,
			"group_defaults":             8,
			"sources":                    9,
			"targets":                    10,
			"file_lists":                 11,
			"directory_lists":            12,
			"file_mappings":              13,
			"directory_mappings":         14,
			"transforms":                 15,
			"target_file_list_refs":      16,
			"target_directory_list_refs": 17,
			"schema_migrations":          18,
		}

		result := orderTables(input)

		// Verify the expected order (hierarchy first)
		expectedOrder := []string{
			"clients",
			"organizations",
			"repos",
			"configs",
			"groups",
			"group_dependencies",
			"group_globals",
			"group_defaults",
			"sources",
			"targets",
			"file_lists",
			"directory_lists",
			"file_mappings",
			"directory_mappings",
			"transforms",
			"target_file_list_refs",
			"target_directory_list_refs",
			"schema_migrations",
		}

		assert.Equal(t, expectedOrder, result)
	})

	t.Run("handles empty input", func(t *testing.T) {
		input := map[string]int64{}
		result := orderTables(input)
		assert.Empty(t, result)
	})

	t.Run("handles only unknown tables", func(t *testing.T) {
		input := map[string]int64{
			"zebra": 1,
			"apple": 2,
			"mango": 3,
		}

		result := orderTables(input)

		// Should be alphabetically sorted
		assert.Equal(t, []string{"apple", "mango", "zebra"}, result)
	})

	t.Run("maintains all tables from input", func(t *testing.T) {
		input := map[string]int64{
			"clients":       1,
			"unknown_one":   2,
			"configs":       3,
			"unknown_two":   4,
			"organizations": 5,
		}

		result := orderTables(input)

		// All tables should be present
		assert.Len(t, result, 5)
		assert.Contains(t, result, "clients")
		assert.Contains(t, result, "organizations")
		assert.Contains(t, result, "configs")
		assert.Contains(t, result, "unknown_one")
		assert.Contains(t, result, "unknown_two")
	})
}

// TestDBStatusDynamicDiscovery tests that db status discovers all tables dynamically
func TestDBStatusDynamicDiscovery(t *testing.T) {
	t.Run("discovers hierarchy tables", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "discovery.db")

		// Save and restore
		oldDBPath := dbPath
		oldStatusJSON := dbStatusJSON
		defer func() {
			dbPath = oldDBPath
			dbStatusJSON = oldStatusJSON
		}()

		dbPath = tmpPath
		dbStatusJSON = true

		// Initialize database
		dbInitForce = false
		err := runDBInit(nil, nil)
		require.NoError(t, err)

		// Capture status output
		var buf bytes.Buffer
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		defer func() { os.Stdout = oldStdout }()

		// Run status
		err = runDBStatus(nil, nil)
		require.NoError(t, err)

		// Restore stdout and read output
		_ = w.Close()
		_, _ = buf.ReadFrom(r)
		output := buf.String()

		// Parse JSON output
		var status DBStatus
		err = json.Unmarshal([]byte(output), &status)
		require.NoError(t, err)

		// Verify hierarchy tables are present
		_, hasClients := status.TableCounts["clients"]
		_, hasOrgs := status.TableCounts["organizations"]
		_, hasRepos := status.TableCounts["repos"]

		assert.True(t, hasClients, "clients table should be discovered")
		assert.True(t, hasOrgs, "organizations table should be discovered")
		assert.True(t, hasRepos, "repos table should be discovered")

		// Verify all expected tables are present
		expectedTables := []string{
			"clients", "organizations", "repos",
			"configs", "groups", "sources", "targets",
			"file_lists", "directory_lists",
			"file_mappings", "directory_mappings",
			"transforms",
			"group_dependencies", "group_globals", "group_defaults",
			"target_file_list_refs", "target_directory_list_refs",
			"schema_migrations",
		}

		for _, table := range expectedTables {
			_, exists := status.TableCounts[table]
			assert.True(t, exists, "table %s should be discovered", table)
		}
	})
}

// BenchmarkDBInit benchmarks database initialization
func BenchmarkDBInit(b *testing.B) {
	tmpBase := b.TempDir()

	oldDBPath := dbPath
	oldForce := dbInitForce
	defer func() {
		dbPath = oldDBPath
		dbInitForce = oldForce
	}()

	dbInitForce = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tmpPath := filepath.Join(tmpBase, fmt.Sprintf("bench_%d.db", i))
		dbPath = tmpPath

		if err := runDBInit(nil, nil); err != nil {
			b.Fatal(err)
		}
	}
}
