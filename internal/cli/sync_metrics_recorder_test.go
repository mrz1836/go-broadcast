package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
	internalsync "github.com/mrz1836/go-broadcast/internal/sync"
)

// newMinimalEngine creates a minimal sync.Engine for unit-testing without
// requiring real GitHub/Git credentials. It uses nil clients which is safe
// as long as Sync() is never called.
func newMinimalEngine(t *testing.T) *internalsync.Engine {
	t.Helper()
	cfg := &config.Config{
		Groups: []config.Group{{
			Source: config.SourceConfig{Repo: "org/src", Branch: "main"},
			Targets: []config.TargetConfig{
				{Repo: "org/target"},
			},
		}},
	}
	opts := internalsync.DefaultOptions().WithDryRun(true)
	engine := internalsync.NewEngine(context.Background(), cfg, nil, nil, nil, nil, opts)
	return engine
}

// newTestDB creates a real SQLite database file in a temp directory and returns its path.
// The file (and directory) are removed automatically via t.Cleanup.
func newTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	database, err := db.Open(db.OpenOptions{
		Path:        path,
		AutoMigrate: true,
	})
	require.NoError(t, err, "failed to create test database")
	require.NoError(t, database.Close())

	return path
}

// setTestDBPath overrides the global dbPath for the duration of a test.
func setTestDBPath(t *testing.T, path string) {
	t.Helper()
	old := dbPath
	dbPath = path
	t.Cleanup(func() { dbPath = old })
}

// TestTryAttachMetricsRecorder_NoDB verifies that when no database file exists at the
// configured path, tryAttachMetricsRecorder returns a no-op closer and the engine
// remains without a metrics recorder.
func TestTryAttachMetricsRecorder_NoDB(t *testing.T) {
	setTestDBPath(t, filepath.Join(t.TempDir(), "does-not-exist.db"))

	engine := newMinimalEngine(t)
	log := logrus.New()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)

	closer := tryAttachMetricsRecorder(engine, log)
	require.NotNil(t, closer)
	closer() // must not panic

	assert.False(t, engine.HasMetricsRecorder(), "recorder should not be set when DB is absent")
	assert.Contains(t, buf.String(), "Database not found")
}

// TestTryAttachMetricsRecorder_ValidDB verifies that when a valid database file exists,
// tryAttachMetricsRecorder attaches a metrics recorder to the engine and returns a
// closer that can be safely called.
func TestTryAttachMetricsRecorder_ValidDB(t *testing.T) {
	dbFile := newTestDB(t)
	setTestDBPath(t, dbFile)

	engine := newMinimalEngine(t)
	log := logrus.New()
	log.SetOutput(os.Stderr)

	closer := tryAttachMetricsRecorder(engine, log)
	require.NotNil(t, closer)
	defer closer()

	assert.True(t, engine.HasMetricsRecorder(), "recorder should be set when DB is present and valid")
}

// TestTryAttachMetricsRecorder_InvalidDBFile verifies that when the path exists but is
// not a valid SQLite database, tryAttachMetricsRecorder returns a no-op closer,
// logs a warning, and the engine remains without a metrics recorder.
func TestTryAttachMetricsRecorder_InvalidDBFile(t *testing.T) {
	// Write a non-SQLite file at the DB path
	dir := t.TempDir()
	badPath := filepath.Join(dir, "not-a-db.db")
	require.NoError(t, os.WriteFile(badPath, []byte("this is not sqlite"), 0o600))
	setTestDBPath(t, badPath)

	engine := newMinimalEngine(t)
	log := logrus.New()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetLevel(logrus.WarnLevel)

	closer := tryAttachMetricsRecorder(engine, log)
	require.NotNil(t, closer)
	closer() // must not panic

	assert.False(t, engine.HasMetricsRecorder(), "recorder should not be set when DB cannot be opened")
	assert.Contains(t, buf.String(), "sync metrics will not be recorded")
}

// TestTryAttachMetricsRecorder_CloserReleasesDB verifies that calling the closer
// returned by tryAttachMetricsRecorder properly releases the database so the file
// can be removed (important on some platforms where open files cannot be deleted).
func TestTryAttachMetricsRecorder_CloserReleasesDB(t *testing.T) {
	dbFile := newTestDB(t)
	setTestDBPath(t, dbFile)

	engine := newMinimalEngine(t)
	log := logrus.New()
	log.SetOutput(os.Stderr)

	closer := tryAttachMetricsRecorder(engine, log)
	require.True(t, engine.HasMetricsRecorder())

	// Calling the closer must not panic and must release the file handle.
	assert.NotPanics(t, closer)

	// Calling it a second time should also be safe (no-op on already-closed DB).
	assert.NotPanics(t, closer)
}

// TestTryAttachMetricsRecorder_PreMigrationDB verifies that when the database
// was created before the broadcast_sync_runs table existed (pre-migration),
// tryAttachMetricsRecorder still attaches successfully because it now runs
// AutoMigrate to create missing tables.
func TestTryAttachMetricsRecorder_PreMigrationDB(t *testing.T) {
	// Create a database WITHOUT auto-migration (simulates a pre-T-98 database
	// that lacks the broadcast_sync_runs table).
	dir := t.TempDir()
	path := filepath.Join(dir, "pre-migration.db")

	database, err := db.Open(db.OpenOptions{
		Path:        path,
		AutoMigrate: false,
	})
	require.NoError(t, err, "failed to create pre-migration database")
	require.NoError(t, database.Close())

	setTestDBPath(t, path)

	engine := newMinimalEngine(t)
	log := logrus.New()
	log.SetOutput(os.Stderr)

	closer := tryAttachMetricsRecorder(engine, log)
	require.NotNil(t, closer)
	defer closer()

	assert.True(t, engine.HasMetricsRecorder(),
		"recorder should be attached even for pre-migration DB (AutoMigrate creates missing tables)")
}
