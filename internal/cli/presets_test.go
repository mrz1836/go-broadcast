package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// withEmptyConfigPath points the global config flag at an empty file inside
// t.TempDir() so config.Load returns no presets without polluting the working
// directory. Returns a restore function.
func withEmptyConfigPath(t *testing.T) func() {
	t.Helper()
	path := filepath.Join(t.TempDir(), "empty-sync.yaml")
	require.NoError(t, os.WriteFile(path, []byte("version: 1\ngroups: []\n"), 0o600))
	old := globalFlags.ConfigFile
	globalFlags.ConfigFile = path
	return func() { globalFlags.ConfigFile = old }
}

// writePresetYAML writes a single preset YAML file to dir and returns its path.
func writePresetYAML(t *testing.T, dir, filename, body string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func TestBundledPresets_AreGenericAndStable(t *testing.T) {
	bundled := BundledPresets()
	require.Len(t, bundled, 2)
	assert.Equal(t, "mvp", bundled[0].ID)
	assert.Equal(t, "go-lib", bundled[1].ID)
	for _, p := range bundled {
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.Description)
		assert.NotEmpty(t, p.Labels, "preset %q should ship default labels", p.ID)
	}
}

func TestRunPresetsList_AggregatesAllSources(t *testing.T) {
	cleanupDB := setupPresetTestDB(t)
	defer cleanupDB()
	cleanupCfg := withEmptyConfigPath(t)
	defer cleanupCfg()

	// Seed a single DB-only preset so we can prove the DB row is captured.
	database, err := db.Open(db.OpenOptions{Path: dbPath, AutoMigrate: true})
	require.NoError(t, err)
	presetRepo := db.NewSettingsPresetRepository(database.DB())
	require.NoError(t, presetRepo.Create(context.Background(), &db.SettingsPreset{
		ExternalID: "team-only",
		Name:       "Team Only",
	}))
	require.NoError(t, database.Close())

	resp, err := captureJSON(t, func() error {
		return runPresetsList(context.Background(), true, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "preset", resp.Type)

	entries, ok := resp.Data.([]interface{})
	require.True(t, ok, "expected list payload, got %T", resp.Data)
	require.GreaterOrEqual(t, len(entries), 3, "expected DB row + 2 bundled defaults")

	// Collect (id, source) pairs for assertions
	type seen struct{ id, source string }
	got := make([]seen, 0, len(entries))
	for _, e := range entries {
		m, ok := e.(map[string]interface{})
		require.True(t, ok)
		got = append(got, seen{
			id:     m["id"].(string),
			source: m["source"].(string),
		})
	}

	hasSource := func(id, source string) bool {
		for _, s := range got {
			if s.id == id && s.source == source {
				return true
			}
		}
		return false
	}

	assert.True(t, hasSource("team-only", presetSourceDB), "team-only should appear with db source")
	assert.True(t, hasSource("mvp", presetSourceBundled), "mvp should appear as bundled-default")
	assert.True(t, hasSource("go-lib", presetSourceBundled), "go-lib should appear as bundled-default")
}

func TestRunPresetsSeed_Idempotent(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	// First seed: creates 2 bundled rows
	resp, err := captureJSON(t, func() error {
		return runPresetsSeed(context.Background(), "", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	data := resp.Data.(map[string]interface{})
	assert.InDelta(t, 2.0, data["bundled_seeded"], 0.0001)

	// Second seed: 0 bundled rows added (idempotent)
	resp, err = captureJSON(t, func() error {
		return runPresetsSeed(context.Background(), "", true)
	})
	require.NoError(t, err)
	data = resp.Data.(map[string]interface{})
	assert.InDelta(t, 0.0, data["bundled_seeded"], 0.0001)

	// DB total should be exactly 2
	database, err := db.Open(db.OpenOptions{Path: dbPath, AutoMigrate: true})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()
	presets, err := db.NewSettingsPresetRepository(database.DB()).List(context.Background())
	require.NoError(t, err)
	assert.Len(t, presets, 2)
}

func TestRunPresetsSeed_FromDir(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	dir := t.TempDir()
	writePresetYAML(t, dir, "team.yaml", `
id: team-defaults
name: Team Defaults
description: Custom team preset
has_issues: true
allow_squash_merge: true
delete_branch_on_merge: true
labels:
  - name: chore
    color: cccccc
    description: Maintenance work
`)

	resp, err := captureJSON(t, func() error {
		return runPresetsSeed(context.Background(), dir, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	data := resp.Data.(map[string]interface{})
	assert.InDelta(t, 2.0, data["bundled_seeded"], 0.0001)
	assert.InDelta(t, 1.0, data["loaded_from_dir"], 0.0001)
	assert.InDelta(t, 0.0, data["overrides"], 0.0001)
	assert.Equal(t, dir, data["from_dir"])

	// Re-running is idempotent: bundled stays at 0, dir count is 1, but it now
	// counts as an override against the previously-imported row.
	resp, err = captureJSON(t, func() error {
		return runPresetsSeed(context.Background(), dir, true)
	})
	require.NoError(t, err)
	data = resp.Data.(map[string]interface{})
	assert.InDelta(t, 0.0, data["bundled_seeded"], 0.0001)
	assert.InDelta(t, 1.0, data["loaded_from_dir"], 0.0001)
	assert.InDelta(t, 1.0, data["overrides"], 0.0001)

	// DB total should be exactly 3 (2 bundled + 1 custom)
	database, err := db.Open(db.OpenOptions{Path: dbPath, AutoMigrate: true})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()
	presets, err := db.NewSettingsPresetRepository(database.DB()).List(context.Background())
	require.NoError(t, err)
	assert.Len(t, presets, 3)
}

func TestRunPresetsSeed_FromDir_OverrideBundled(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	// Pre-seed bundled defaults so the next --from import is an override.
	_, err := captureJSON(t, func() error {
		return runPresetsSeed(context.Background(), "", true)
	})
	require.NoError(t, err)

	dir := t.TempDir()
	writePresetYAML(t, dir, "mvp.yaml", `
id: mvp
name: MVP Override
description: Locally-modified mvp preset
has_issues: false
allow_squash_merge: true
delete_branch_on_merge: true
labels:
  - name: priority
    color: ff0000
    description: Top priority
`)

	resp, err := captureJSON(t, func() error {
		return runPresetsSeed(context.Background(), dir, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	data := resp.Data.(map[string]interface{})
	assert.InDelta(t, 1.0, data["loaded_from_dir"], 0.0001)
	assert.InDelta(t, 1.0, data["overrides"], 0.0001)

	// Confirm the bundled mvp row was overridden in the DB.
	database, err := db.Open(db.OpenOptions{Path: dbPath, AutoMigrate: true})
	require.NoError(t, err)
	defer func() { _ = database.Close() }()
	mvp, err := db.NewSettingsPresetRepository(database.DB()).GetByExternalID(context.Background(), "mvp")
	require.NoError(t, err)
	assert.Equal(t, "MVP Override", mvp.Name)
	assert.False(t, mvp.HasIssues, "override should drive HasIssues to false")
	require.Len(t, mvp.Labels, 1)
	assert.Equal(t, "priority", mvp.Labels[0].Name)
}

func TestParsePresetFile_MissingID(t *testing.T) {
	dir := t.TempDir()
	path := writePresetYAML(t, dir, "no-id.yaml", `
name: Nameless
description: Missing id field
`)

	preset, err := parsePresetFile(path)
	require.NoError(t, err)
	assert.Empty(t, preset.ID, "parser tolerates missing id; loader rejects it")
}
