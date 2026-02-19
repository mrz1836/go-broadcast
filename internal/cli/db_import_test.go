package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
)

func TestCollectFileMetadata(t *testing.T) {
	t.Parallel()

	t.Run("empty path", func(t *testing.T) {
		t.Parallel()

		result := collectFileMetadata("")
		assert.Empty(t, result["path"])
		assert.Empty(t, result["abs_path"])
		assert.Empty(t, result["sha256"])
		assert.Equal(t, 0, result["size_bytes"])
	})

	t.Run("stdin indicator", func(t *testing.T) {
		t.Parallel()

		result := collectFileMetadata("-")
		assert.Equal(t, "-", result["path"])
		assert.Empty(t, result["abs_path"])
		assert.Equal(t, 0, result["size_bytes"])
	})

	t.Run("nonexistent file returns partial metadata", func(t *testing.T) {
		t.Parallel()

		result := collectFileMetadata("/nonexistent/path/file.yaml")
		assert.Equal(t, "/nonexistent/path/file.yaml", result["path"])
		assert.Empty(t, result["abs_path"])
		assert.Equal(t, 0, result["size_bytes"])
		assert.Empty(t, result["sha256"])
	})

	t.Run("real file returns complete metadata", func(t *testing.T) {
		t.Parallel()

		// Create a temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "test.yaml")
		require.NoError(t, os.WriteFile(tmpFile, []byte("version: 1\n"), 0o600))

		result := collectFileMetadata(tmpFile)
		assert.Equal(t, tmpFile, result["path"])
		assert.NotEmpty(t, result["abs_path"])
		assert.NotEmpty(t, result["sha256"])
		assert.NotEqual(t, 0, result["size_bytes"])
		assert.NotEmpty(t, result["modified_at"])
		assert.NotEmpty(t, result["rel_path"])
	})
}

func TestBuildCompleteMetadata(t *testing.T) {
	t.Parallel()

	t.Run("builds metadata structure", func(t *testing.T) {
		t.Parallel()

		cfg := &config.Config{
			Name:    "test-config",
			Version: 1,
			Groups: []config.Group{
				{
					ID:   "g1",
					Name: "Test Group",
					Source: config.SourceConfig{
						Repo:   "org/template",
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{Repo: "org/target1"},
					},
				},
			},
		}

		metadata := buildCompleteMetadata(cfg, "/nonexistent/sync.yaml")

		// Verify import context
		importCtx, ok := metadata["import"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "cli", importCtx["source_type"])
		assert.NotEmpty(t, importCtx["timestamp"])
		assert.Equal(t, "1.0", importCtx["enriched_version"])

		// Verify source file metadata exists
		_, ok = metadata["source_file"]
		assert.True(t, ok)

		// Verify metrics exist
		_, ok = metadata["metrics"]
		assert.True(t, ok)

		// Verify config analysis exists
		_, ok = metadata["config_analysis"]
		assert.True(t, ok)
	})

	t.Run("with real file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "sync.yaml")
		require.NoError(t, os.WriteFile(tmpFile, []byte("version: 1\n"), 0o600))

		cfg := &config.Config{Version: 1}
		metadata := buildCompleteMetadata(cfg, tmpFile)

		fileMetadata, ok := metadata["source_file"].(map[string]interface{})
		require.True(t, ok)
		assert.NotEmpty(t, fileMetadata["sha256"])
	})
}

// TestEnrichConfigWithMetadata tests the enrichConfigWithMetadata function
// using a real in-memory SQLite database.
func TestEnrichConfigWithMetadata(t *testing.T) {
	t.Parallel()

	t.Run("updates metadata on existing config", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		testDBPath := filepath.Join(tmpDir, "enrich_test.db")

		// Open database with auto-migrate
		database, err := db.Open(db.OpenOptions{
			Path:        testDBPath,
			LogLevel:    logger.Silent,
			AutoMigrate: true,
		})
		require.NoError(t, err)
		defer func() { _ = database.Close() }()

		// Create a config record directly
		dbConfig := db.Config{
			ExternalID: "test-enrich",
			Name:       "Enrich Test Config",
			Version:    1,
		}
		result := database.DB().Create(&dbConfig)
		require.NoError(t, result.Error)
		require.NotZero(t, dbConfig.ID)

		// Build metadata to apply
		metadata := db.Metadata{
			"import": map[string]interface{}{
				"source_type": "cli",
				"timestamp":   "2026-01-01T00:00:00Z",
			},
			"metrics": map[string]interface{}{
				"total_groups":  1,
				"total_targets": 3,
			},
		}

		// Enrich the config with metadata
		err = enrichConfigWithMetadata(database, dbConfig.ID, metadata)
		require.NoError(t, err)

		// Read back and verify metadata was stored
		var updatedConfig db.Config
		result = database.DB().First(&updatedConfig, dbConfig.ID)
		require.NoError(t, result.Error)
		assert.NotNil(t, updatedConfig.Metadata)

		// Verify metadata content
		importCtx, ok := updatedConfig.Metadata["import"]
		assert.True(t, ok, "metadata should contain 'import' key")
		if importMap, ok := importCtx.(map[string]interface{}); ok {
			assert.Equal(t, "cli", importMap["source_type"])
		}
	})

	t.Run("handles nil metadata gracefully", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		testDBPath := filepath.Join(tmpDir, "enrich_nil_test.db")

		database, err := db.Open(db.OpenOptions{
			Path:        testDBPath,
			LogLevel:    logger.Silent,
			AutoMigrate: true,
		})
		require.NoError(t, err)
		defer func() { _ = database.Close() }()

		// Create a config record
		dbConfig := db.Config{
			ExternalID: "test-nil-metadata",
			Name:       "Nil Metadata Test",
			Version:    1,
		}
		result := database.DB().Create(&dbConfig)
		require.NoError(t, result.Error)

		// Enrich with nil metadata (should not error)
		err = enrichConfigWithMetadata(database, dbConfig.ID, nil)
		assert.NoError(t, err)
	})

	t.Run("handles empty metadata", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		testDBPath := filepath.Join(tmpDir, "enrich_empty_test.db")

		database, err := db.Open(db.OpenOptions{
			Path:        testDBPath,
			LogLevel:    logger.Silent,
			AutoMigrate: true,
		})
		require.NoError(t, err)
		defer func() { _ = database.Close() }()

		// Create a config record
		dbConfig := db.Config{
			ExternalID: "test-empty-metadata",
			Name:       "Empty Metadata Test",
			Version:    1,
		}
		result := database.DB().Create(&dbConfig)
		require.NoError(t, result.Error)

		// Enrich with empty metadata
		err = enrichConfigWithMetadata(database, dbConfig.ID, db.Metadata{})
		assert.NoError(t, err)
	})

	t.Run("nonexistent config ID does not error with gorm update", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		testDBPath := filepath.Join(tmpDir, "enrich_missing_test.db")

		database, err := db.Open(db.OpenOptions{
			Path:        testDBPath,
			LogLevel:    logger.Silent,
			AutoMigrate: true,
		})
		require.NoError(t, err)
		defer func() { _ = database.Close() }()

		metadata := db.Metadata{"key": "value"}

		// GORM Update on nonexistent record does not return an error
		// (it returns RowsAffected=0 but no error)
		err = enrichConfigWithMetadata(database, 99999, metadata)
		assert.NoError(t, err)
	})
}
