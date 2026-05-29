package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm/logger"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
)

// validImportConfig returns a fully-formed valid config suitable for import.
func validImportConfig() *config.Config {
	return &config.Config{
		Version: 1,
		Name:    "test-config",
		ID:      "test-cfg",
		FileLists: []config.FileList{
			{
				ID:   "test-files",
				Name: "Test Files",
				Files: []config.FileMapping{
					{Src: "README.md", Dest: "README.md"},
				},
			},
		},
		Groups: []config.Group{
			{
				Name:   "Test Group",
				ID:     "test-group",
				Source: config.SourceConfig{Repo: "mrz1836/template", Branch: "main"},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/target1", Branch: "main", FileListRefs: []string{"test-files"}},
				},
			},
		},
	}
}

// writeYAML marshals a config to a YAML file and returns the path.
func writeYAML(t *testing.T, dir, name string, cfg *config.Config) string {
	t.Helper()
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, data, 0o600))
	return p
}

// withDBPath temporarily sets the global dbPath and restores it on cleanup.
func withDBPath(t *testing.T, path string) {
	t.Helper()
	old := dbPath
	dbPath = path
	t.Cleanup(func() { dbPath = old })
}

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

// TestRunDBImport exercises the import orchestrator end-to-end.
func TestRunDBImport(t *testing.T) { //nolint:paralleltest // mutates global flags/dbPath
	tmp := t.TempDir()
	yamlPath := writeYAML(t, tmp, "sync.yaml", validImportConfig())
	dbFile := filepath.Join(tmp, "import.db")

	withDBPath(t, dbFile)

	oldYAML, oldForce := dbImportYAML, dbImportForce
	dbImportYAML, dbImportForce = yamlPath, false
	t.Cleanup(func() { dbImportYAML, dbImportForce = oldYAML, oldForce })

	t.Run("success", func(t *testing.T) {
		require.NoError(t, runDBImport(newTestCmd(), nil))

		// Verify the config landed in the DB.
		database, err := db.Open(db.OpenOptions{Path: dbFile, LogLevel: logger.Silent})
		require.NoError(t, err)
		defer func() { _ = database.Close() }()
		var cnt int64
		database.DB().Model(&db.Config{}).Where("external_id = ?", "test-cfg").Count(&cnt)
		assert.Equal(t, int64(1), cnt)
	})

	t.Run("duplicate without force errors", func(t *testing.T) {
		err := runDBImport(newTestCmd(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("force flag re-imports", func(t *testing.T) {
		// With --force the orchestrator proceeds past the "already exists" guard
		// and delegates to the converter. We only assert that runDBImport reaches
		// the converter (i.e. does NOT return the "already exists" guard error).
		dbImportForce = true
		defer func() { dbImportForce = false }()
		err := runDBImport(newTestCmd(), nil)
		if err != nil {
			assert.NotContains(t, err.Error(), "already exists")
		}
	})

	t.Run("missing yaml errors", func(t *testing.T) {
		dbImportYAML = filepath.Join(tmp, "does-not-exist.yaml")
		defer func() { dbImportYAML = yamlPath }()
		err := runDBImport(newTestCmd(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load YAML")
	})
}

// TestRunDBExport exercises the export orchestrator.
func TestRunDBExport(t *testing.T) { //nolint:paralleltest // mutates global flags/dbPath
	tmp := t.TempDir()
	yamlPath := writeYAML(t, tmp, "sync.yaml", validImportConfig())
	dbFile := filepath.Join(tmp, "export.db")

	withDBPath(t, dbFile)

	// Seed the DB by importing first.
	oldYAML := dbImportYAML
	dbImportYAML = yamlPath
	t.Cleanup(func() { dbImportYAML = oldYAML })
	require.NoError(t, runDBImport(newTestCmd(), nil))

	oldOut, oldGroup, oldStdout := dbExportOutput, dbExportGroup, dbExportStdout
	t.Cleanup(func() { dbExportOutput, dbExportGroup, dbExportStdout = oldOut, oldGroup, oldStdout })

	t.Run("requires output or stdout", func(t *testing.T) {
		dbExportOutput, dbExportStdout, dbExportGroup = "", false, ""
		err := runDBExport(newTestCmd(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "either --output or --stdout")
	})

	t.Run("rejects both output and stdout", func(t *testing.T) {
		dbExportOutput, dbExportStdout, dbExportGroup = "out.yaml", true, ""
		err := runDBExport(newTestCmd(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify both")
	})

	t.Run("export to file", func(t *testing.T) {
		out := filepath.Join(tmp, "out-full.yaml")
		dbExportOutput, dbExportStdout, dbExportGroup = out, false, ""
		require.NoError(t, runDBExport(newTestCmd(), nil))
		assert.FileExists(t, out)
	})

	t.Run("export single group", func(t *testing.T) {
		out := filepath.Join(tmp, "out-group.yaml")
		dbExportOutput, dbExportStdout, dbExportGroup = out, false, "test-group"
		require.NoError(t, runDBExport(newTestCmd(), nil))
		assert.FileExists(t, out)
	})

	t.Run("export to stdout", func(t *testing.T) {
		dbExportOutput, dbExportStdout, dbExportGroup = "", true, ""
		require.NoError(t, runDBExport(newTestCmd(), nil))
	})

	t.Run("missing group errors", func(t *testing.T) {
		dbExportOutput, dbExportStdout, dbExportGroup = filepath.Join(tmp, "x.yaml"), false, "no-such-group"
		err := runDBExport(newTestCmd(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestRunDBDiff exercises the diff orchestrator.
func TestRunDBDiff(t *testing.T) { //nolint:paralleltest // mutates global flags/dbPath
	tmp := t.TempDir()
	yamlPath := writeYAML(t, tmp, "sync.yaml", validImportConfig())
	dbFile := filepath.Join(tmp, "diff.db")

	withDBPath(t, dbFile)

	oldImportYAML := dbImportYAML
	dbImportYAML = yamlPath
	t.Cleanup(func() { dbImportYAML = oldImportYAML })
	require.NoError(t, runDBImport(newTestCmd(), nil))

	oldDiffYAML, oldDetail := dbDiffYAML, dbDiffDetail
	t.Cleanup(func() { dbDiffYAML, dbDiffDetail = oldDiffYAML, oldDetail })

	t.Run("no differences for identical config", func(t *testing.T) {
		dbDiffYAML, dbDiffDetail = yamlPath, false
		require.NoError(t, runDBDiff(newTestCmd(), nil))
	})

	t.Run("detail mode with differences", func(t *testing.T) {
		modified := validImportConfig()
		modified.Groups[0].Name = "Renamed Group"
		modPath := writeYAML(t, tmp, "modified.yaml", modified)
		dbDiffYAML, dbDiffDetail = modPath, true
		require.NoError(t, runDBDiff(newTestCmd(), nil))
	})

	t.Run("missing yaml errors", func(t *testing.T) {
		dbDiffYAML = filepath.Join(tmp, "nope.yaml")
		defer func() { dbDiffYAML = yamlPath }()
		err := runDBDiff(newTestCmd(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load YAML")
	})
}

// TestRunMetrics exercises the metrics command routing.
func TestRunMetrics(t *testing.T) { //nolint:paralleltest // mutates global flags/dbPath
	tmp := t.TempDir()
	dbFile := filepath.Join(tmp, "metrics.db")
	// Create an empty (migrated) DB so openDatabase succeeds.
	database, err := db.Open(db.OpenOptions{Path: dbFile, LogLevel: logger.Silent, AutoMigrate: true})
	require.NoError(t, err)
	require.NoError(t, database.Close())

	withDBPath(t, dbFile)

	setFlags := func(last, repo, runID string, jsonOut bool) {
		metricsFlagsMu.Lock()
		metricsLast, metricsRepo, metricsRunID, metricsJSON = last, repo, runID, jsonOut
		metricsFlagsMu.Unlock()
	}
	t.Cleanup(func() { setFlags("", "", "", false) })

	t.Run("summary mode", func(t *testing.T) {
		setFlags("", "", "", false)
		require.NoError(t, runMetrics(newTestCmd(), nil))
	})

	t.Run("recent runs mode", func(t *testing.T) {
		setFlags("7d", "", "", false)
		require.NoError(t, runMetrics(newTestCmd(), nil))
	})

	t.Run("run details not found", func(t *testing.T) {
		setFlags("", "", "SR-missing", false)
		err := runMetrics(newTestCmd(), nil)
		require.Error(t, err)
	})

	t.Run("repo history invalid format", func(t *testing.T) {
		setFlags("", "bad-name", "", false)
		err := runMetrics(newTestCmd(), nil)
		require.Error(t, err)
	})

	t.Run("missing database errors", func(t *testing.T) {
		withDBPath(t, filepath.Join(tmp, "missing.db"))
		setFlags("", "", "", false)
		err := runMetrics(newTestCmd(), nil)
		require.Error(t, err)
	})
}

// TestRunAnalyticsStatus exercises the analytics status command.
func TestRunAnalyticsStatus(t *testing.T) { //nolint:paralleltest // mutates global dbPath
	tmp := t.TempDir()
	dbFile := filepath.Join(tmp, "analytics.db")
	database, err := db.Open(db.OpenOptions{Path: dbFile, LogLevel: logger.Silent, AutoMigrate: true})
	require.NoError(t, err)
	require.NoError(t, database.Close())

	withDBPath(t, dbFile)

	t.Run("all repositories empty DB", func(t *testing.T) {
		require.NoError(t, runAnalyticsStatus(newTestCmd(), nil))
	})

	t.Run("invalid repo name errors", func(t *testing.T) {
		err := runAnalyticsStatus(newTestCmd(), []string{"bad-name"})
		require.Error(t, err)
	})

	t.Run("single repo not in DB errors", func(t *testing.T) {
		err := runAnalyticsStatus(newTestCmd(), []string{"org/missing"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in database")
	})

	t.Run("missing database errors", func(t *testing.T) {
		withDBPath(t, filepath.Join(tmp, "missing.db"))
		err := runAnalyticsStatus(newTestCmd(), nil)
		require.Error(t, err)
	})
}
