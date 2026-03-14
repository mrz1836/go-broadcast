package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
)

// setupPresetTestDB creates a test DB with preset support and returns cleanup
func setupPresetTestDB(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "test-preset.db")

	oldDBPath := dbPath
	dbPath = tmpPath

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(t, err)
	require.NoError(t, database.Close())

	return func() {
		dbPath = oldDBPath
	}
}

func TestPresetList_Empty(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	t.Run("json output", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runPresetList(true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "listed", resp.Action)
		assert.Equal(t, "preset", resp.Type)
		assert.Equal(t, 0, resp.Count)
	})

	t.Run("human output", func(t *testing.T) {
		err := runPresetList(false)
		require.NoError(t, err)
	})
}

func TestPresetCreate_And_List(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	// Create
	resp, err := captureJSON(t, func() error {
		return runPresetCreate("test-preset", "Test Preset", "A test preset", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "created", resp.Action)

	// List should show 1
	resp, err = captureJSON(t, func() error {
		return runPresetList(true)
	})
	require.NoError(t, err)
	assert.Equal(t, 1, resp.Count)
}

func TestPresetCreate_Duplicate(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	// Create first
	_, err := captureJSON(t, func() error {
		return runPresetCreate("dup-test", "Dup", "", true)
	})
	require.NoError(t, err)

	// Create duplicate
	resp, err := captureJSON(t, func() error {
		return runPresetCreate("dup-test", "Dup2", "", true)
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "already exists")
}

func TestPresetShow(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	// Create first
	_, _ = captureJSON(t, func() error {
		return runPresetCreate("show-test", "Show Test", "desc", true)
	})

	t.Run("found", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runPresetShow("show-test", true)
		})
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "show", resp.Action)
	})

	t.Run("not found", func(t *testing.T) {
		resp, err := captureJSON(t, func() error {
			return runPresetShow("nonexistent", true)
		})
		require.NoError(t, err)
		assert.False(t, resp.Success)
	})

	t.Run("human output", func(t *testing.T) {
		err := runPresetShow("show-test", false)
		require.NoError(t, err)
	})
}

func TestPresetDelete(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	// Create
	_, _ = captureJSON(t, func() error {
		return runPresetCreate("del-test", "Delete Test", "", true)
	})

	// Delete
	resp, err := captureJSON(t, func() error {
		return runPresetDelete("del-test", false, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "soft-deleted", resp.Action)
}

func TestPresetDelete_NotFound(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runPresetDelete("nonexistent", false, true)
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)
}

func TestConfigPresetToDBPreset(t *testing.T) {
	cp := &config.SettingsPreset{
		ID:                  "test",
		Name:                "Test",
		Description:         "Test desc",
		HasIssues:           true,
		HasWiki:             false,
		AllowSquashMerge:    true,
		DeleteBranchOnMerge: true,
		Labels: []config.LabelSpec{
			{Name: "bug", Color: "d73a4a", Description: "Bug"},
			{Name: "enhancement", Color: "a2eeef"},
		},
		Rulesets: []config.RulesetConfig{
			{
				Name:    "branch-protection",
				Target:  "branch",
				Include: []string{"~DEFAULT_BRANCH"},
				Rules:   []string{"pull_request"},
			},
			{
				Name:        "tag-protection",
				Target:      "tag",
				Enforcement: "evaluate",
				Include:     []string{"v*"},
				Rules:       []string{"deletion"},
			},
		},
	}

	result := configPresetToDBPreset(cp)
	assert.Equal(t, "test", result.ExternalID)
	assert.Equal(t, "Test", result.Name)
	assert.True(t, result.HasIssues)
	assert.False(t, result.HasWiki)
	assert.True(t, result.AllowSquashMerge)
	assert.True(t, result.DeleteBranchOnMerge)

	require.Len(t, result.Labels, 2)
	assert.Equal(t, "bug", result.Labels[0].Name)
	assert.Equal(t, "d73a4a", result.Labels[0].Color)

	require.Len(t, result.Rulesets, 2)
	assert.Equal(t, "branch-protection", result.Rulesets[0].Name)
	assert.Equal(t, "active", result.Rulesets[0].Enforcement) // Default enforcement
	assert.Equal(t, "tag-protection", result.Rulesets[1].Name)
	assert.Equal(t, "evaluate", result.Rulesets[1].Enforcement) // Explicit enforcement
}

func TestPresetAssign_Success(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	// Create a preset first
	_, err := captureJSON(t, func() error {
		return runPresetCreate("assign-test", "Assign Test", "desc", true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runPresetAssign("assign-test", "owner/repo", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "assigned", resp.Action)
}

func TestPresetAssign_PresetNotFound(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	resp, err := captureJSON(t, func() error {
		return runPresetAssign("nonexistent-preset", "owner/repo", true)
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "not found")
}

func TestPresetAssign_JSONOutput(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	_, err := captureJSON(t, func() error {
		return runPresetCreate("json-assign", "JSON Assign", "", true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runPresetAssign("json-assign", "acme/my-repo", true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "assigned", resp.Action)
	assert.Equal(t, "preset", resp.Type)
}

func TestPresetImport_Success(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	configContent := `version: 1
groups: []
settings_presets:
  - id: import-preset
    name: Import Preset
    has_issues: true
    allow_squash_merge: true
`
	configPath := filepath.Join(t.TempDir(), "sync.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o600))

	resp, err := captureJSON(t, func() error {
		return runPresetImport(configPath, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "imported", resp.Action)
	assert.Equal(t, 1, resp.Count)
}

func TestPresetImport_NoPresetsInConfig(t *testing.T) {
	configContent := `version: 1
groups: []
`
	configPath := filepath.Join(t.TempDir(), "sync.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o600))

	resp, err := captureJSON(t, func() error {
		return runPresetImport(configPath, true)
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "no settings_presets")
}

func TestPresetImport_InvalidConfigPath(t *testing.T) {
	resp, err := captureJSON(t, func() error {
		return runPresetImport("/nonexistent/path/sync.yaml", true)
	})
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotEmpty(t, resp.Error)
}

func TestPresetImport_MultiplePresets(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	configContent := `version: 1
groups: []
settings_presets:
  - id: multi-preset-1
    name: Multi Preset 1
    has_issues: true
    allow_squash_merge: true
  - id: multi-preset-2
    name: Multi Preset 2
    has_issues: false
    allow_merge_commit: true
`
	configPath := filepath.Join(t.TempDir(), "sync.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0o600))

	resp, err := captureJSON(t, func() error {
		return runPresetImport(configPath, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, 2, resp.Count)
}

func TestPresetDelete_HardDelete(t *testing.T) {
	cleanup := setupPresetTestDB(t)
	defer cleanup()

	_, err := captureJSON(t, func() error {
		return runPresetCreate("hard-del-test", "Hard Delete Test", "", true)
	})
	require.NoError(t, err)

	resp, err := captureJSON(t, func() error {
		return runPresetDelete("hard-del-test", true, true)
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "hard-deleted", resp.Action)
}

func TestBuildPresetDetailResult(t *testing.T) {
	preset := &db.SettingsPreset{
		ExternalID:               "mvp",
		Name:                     "MVP",
		Description:              "Default",
		HasIssues:                true,
		AllowSquashMerge:         true,
		DeleteBranchOnMerge:      true,
		SquashMergeCommitTitle:   "PR_TITLE",
		SquashMergeCommitMessage: "PR_BODY",
		Labels: []db.SettingsPresetLabel{
			{Name: "bug", Color: "d73a4a", Description: "Bug"},
		},
		Rulesets: []db.SettingsPresetRuleset{
			{Name: "bp", Target: "branch", Enforcement: "active", Include: db.JSONStringSlice{"~DEFAULT_BRANCH"}, Rules: db.JSONStringSlice{"pull_request"}},
		},
	}

	result := buildPresetDetailResult(preset)
	assert.Equal(t, "mvp", result.ExternalID)
	assert.True(t, result.HasIssues)
	assert.Equal(t, "PR_TITLE", result.SquashMergeCommitTitle)
	require.Len(t, result.Labels, 1)
	assert.Equal(t, "bug", result.Labels[0].Name)
	require.Len(t, result.Rulesets, 1)
	assert.Equal(t, "bp", result.Rulesets[0].Name)
	assert.Equal(t, []string{"~DEFAULT_BRANCH"}, result.Rulesets[0].Include)
}
