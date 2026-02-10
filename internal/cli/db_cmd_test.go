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
		assert.NoError(t, err, "database file should exist")
	})

	t.Run("fails if database already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpPath := filepath.Join(tmpDir, "existing.db")

		// Create existing database
		_, err := os.Create(tmpPath)
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
		assert.Error(t, err)
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
		content, err := os.ReadFile(tmpPath)
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
		assert.NoError(t, err)
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
		assert.NoError(t, err)
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
		assert.Error(t, err)
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
		assert.NoError(t, err)
		assert.Equal(t, tmpPath, status.Path)
		assert.True(t, status.Exists)
		assert.Positive(t, status.TableCounts["configs"])
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
		assert.Len(t, subcommands, 2, "should have 2 subcommands")

		// Find init and status commands
		var hasInit, hasStatus bool
		for _, cmd := range subcommands {
			switch cmd.Use {
			case "init":
				hasInit = true
			case "status":
				hasStatus = true
			}
		}

		assert.True(t, hasInit, "should have init subcommand")
		assert.True(t, hasStatus, "should have status subcommand")
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
		assert.NoError(t, err)
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
		assert.Error(t, err)
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
