package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestPreset is a helper that creates a SettingsPreset with labels and rulesets
func createTestPreset(externalID, name string) *SettingsPreset {
	return &SettingsPreset{
		ExternalID:               externalID,
		Name:                     name,
		Description:              "Test preset: " + name,
		HasIssues:                true,
		HasWiki:                  false,
		HasProjects:              false,
		HasDiscussions:           false,
		AllowSquashMerge:         true,
		AllowMergeCommit:         false,
		AllowRebaseMerge:         false,
		DeleteBranchOnMerge:      true,
		AllowAutoMerge:           true,
		AllowUpdateBranch:        true,
		SquashMergeCommitTitle:   "PR_TITLE",
		SquashMergeCommitMessage: "COMMIT_MESSAGES",
		Labels: []SettingsPresetLabel{
			{Name: "bug", Color: "d73a4a", Description: "Something isn't working"},
			{Name: "enhancement", Color: "a2eeef", Description: "New feature or request"},
		},
		Rulesets: []SettingsPresetRuleset{
			{
				Name:        "branch-protection",
				Target:      "branch",
				Enforcement: "active",
				Include:     JSONStringSlice{"~DEFAULT_BRANCH"},
				Rules:       JSONStringSlice{"deletion", "pull_request"},
			},
		},
	}
}

func TestSettingsPresetRepository_CreateAndGetByID(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	preset := createTestPreset("mvp", "MVP")
	err := repo.Create(ctx, preset)
	require.NoError(t, err)
	assert.NotZero(t, preset.ID)

	// GetByID should return the preset with children
	got, err := repo.GetByID(ctx, preset.ID)
	require.NoError(t, err)
	assert.Equal(t, "mvp", got.ExternalID)
	assert.Equal(t, "MVP", got.Name)
	assert.Equal(t, "Test preset: MVP", got.Description)
	assert.True(t, got.HasIssues)
	assert.False(t, got.HasWiki)
	assert.True(t, got.AllowSquashMerge)
	assert.False(t, got.AllowMergeCommit)
	assert.True(t, got.DeleteBranchOnMerge)
	assert.Equal(t, "PR_TITLE", got.SquashMergeCommitTitle)
	assert.Equal(t, "COMMIT_MESSAGES", got.SquashMergeCommitMessage)

	// Labels
	require.Len(t, got.Labels, 2)
	assert.Equal(t, "bug", got.Labels[0].Name)
	assert.Equal(t, "d73a4a", got.Labels[0].Color)
	assert.Equal(t, "enhancement", got.Labels[1].Name)

	// Rulesets
	require.Len(t, got.Rulesets, 1)
	assert.Equal(t, "branch-protection", got.Rulesets[0].Name)
	assert.Equal(t, "branch", got.Rulesets[0].Target)
	assert.Equal(t, "active", got.Rulesets[0].Enforcement)
	assert.Equal(t, JSONStringSlice{"~DEFAULT_BRANCH"}, got.Rulesets[0].Include)
	assert.Equal(t, JSONStringSlice{"deletion", "pull_request"}, got.Rulesets[0].Rules)
}

func TestSettingsPresetRepository_GetByID_NotFound(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, 99999)
	require.ErrorIs(t, err, ErrRecordNotFound)
}

func TestSettingsPresetRepository_GetByExternalID(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	preset := createTestPreset("go-lib", "Go Library")
	require.NoError(t, repo.Create(ctx, preset))

	t.Run("found", func(t *testing.T) {
		got, err := repo.GetByExternalID(ctx, "go-lib")
		require.NoError(t, err)
		assert.Equal(t, preset.ID, got.ID)
		assert.Equal(t, "Go Library", got.Name)
		require.Len(t, got.Labels, 2)
		require.Len(t, got.Rulesets, 1)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := repo.GetByExternalID(ctx, "non-existent")
		require.ErrorIs(t, err, ErrRecordNotFound)
	})
}

func TestSettingsPresetRepository_Update(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	preset := createTestPreset("updatable", "Before Update")
	require.NoError(t, repo.Create(ctx, preset))

	// Modify fields
	preset.Name = "After Update"
	preset.Description = "Updated description"
	preset.HasWiki = true
	preset.AllowMergeCommit = true
	preset.SquashMergeCommitTitle = "COMMIT_OR_PR_TITLE"

	err := repo.Update(ctx, preset)
	require.NoError(t, err)

	// Reload and verify
	got, err := repo.GetByID(ctx, preset.ID)
	require.NoError(t, err)
	assert.Equal(t, "After Update", got.Name)
	assert.Equal(t, "Updated description", got.Description)
	assert.True(t, got.HasWiki)
	assert.True(t, got.AllowMergeCommit)
	assert.Equal(t, "COMMIT_OR_PR_TITLE", got.SquashMergeCommitTitle)
}

func TestSettingsPresetRepository_Delete(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	t.Run("soft delete", func(t *testing.T) {
		preset := createTestPreset("soft-del", "Soft Delete Me")
		require.NoError(t, repo.Create(ctx, preset))

		err := repo.Delete(ctx, preset.ID, false)
		require.NoError(t, err)

		// Should not be found via normal query
		_, err = repo.GetByID(ctx, preset.ID)
		require.ErrorIs(t, err, ErrRecordNotFound)

		// But should still exist in database (soft deleted)
		var count int64
		gormDB.Unscoped().Model(&SettingsPreset{}).Where("id = ?", preset.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("hard delete", func(t *testing.T) {
		preset := createTestPreset("hard-del", "Hard Delete Me")
		require.NoError(t, repo.Create(ctx, preset))

		presetID := preset.ID
		require.Len(t, preset.Labels, 2)
		require.Len(t, preset.Rulesets, 1)

		err := repo.Delete(ctx, presetID, true)
		require.NoError(t, err)

		// Should not be found at all, even unscoped
		var presetCount int64
		gormDB.Unscoped().Model(&SettingsPreset{}).Where("id = ?", presetID).Count(&presetCount)
		assert.Equal(t, int64(0), presetCount)

		// Children should also be hard deleted
		var labelCount int64
		gormDB.Unscoped().Model(&SettingsPresetLabel{}).Where("settings_preset_id = ?", presetID).Count(&labelCount)
		assert.Equal(t, int64(0), labelCount)

		var rulesetCount int64
		gormDB.Unscoped().Model(&SettingsPresetRuleset{}).Where("settings_preset_id = ?", presetID).Count(&rulesetCount)
		assert.Equal(t, int64(0), rulesetCount)
	})
}

func TestSettingsPresetRepository_List(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	// Create multiple presets
	preset1 := createTestPreset("alpha", "Alpha Preset")
	preset2 := createTestPreset("beta", "Beta Preset")
	preset3 := createTestPreset("gamma", "Gamma Preset")

	require.NoError(t, repo.Create(ctx, preset1))
	require.NoError(t, repo.Create(ctx, preset2))
	require.NoError(t, repo.Create(ctx, preset3))

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 3)

	// Should be ordered by external_id ASC
	assert.Equal(t, "alpha", list[0].ExternalID)
	assert.Equal(t, "beta", list[1].ExternalID)
	assert.Equal(t, "gamma", list[2].ExternalID)

	// Children should be preloaded
	for _, p := range list {
		assert.Len(t, p.Labels, 2, "preset %s should have 2 labels", p.ExternalID)
		assert.Len(t, p.Rulesets, 1, "preset %s should have 1 ruleset", p.ExternalID)
	}
}

func TestSettingsPresetRepository_List_Empty(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	list, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestSettingsPresetRepository_ImportFromConfig_CreateNew(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	preset := &SettingsPreset{
		ExternalID:               "new-import",
		Name:                     "Imported Preset",
		Description:              "Created via import",
		HasIssues:                true,
		AllowSquashMerge:         true,
		DeleteBranchOnMerge:      true,
		SquashMergeCommitTitle:   "PR_TITLE",
		SquashMergeCommitMessage: "COMMIT_MESSAGES",
		Labels: []SettingsPresetLabel{
			{Name: "imported-label", Color: "ff0000", Description: "Imported"},
		},
		Rulesets: []SettingsPresetRuleset{
			{
				Name:        "imported-ruleset",
				Target:      "branch",
				Enforcement: "active",
				Include:     JSONStringSlice{"~DEFAULT_BRANCH"},
				Rules:       JSONStringSlice{"pull_request"},
			},
		},
	}

	err := repo.ImportFromConfig(ctx, preset)
	require.NoError(t, err)
	assert.NotZero(t, preset.ID)

	// Verify it was created correctly
	got, err := repo.GetByExternalID(ctx, "new-import")
	require.NoError(t, err)
	assert.Equal(t, "Imported Preset", got.Name)
	assert.Equal(t, "Created via import", got.Description)
	require.Len(t, got.Labels, 1)
	assert.Equal(t, "imported-label", got.Labels[0].Name)
	require.Len(t, got.Rulesets, 1)
	assert.Equal(t, "imported-ruleset", got.Rulesets[0].Name)
}

func TestSettingsPresetRepository_ImportFromConfig_UpsertExisting(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	// Create initial preset
	initial := createTestPreset("upsert-test", "Initial Name")
	require.NoError(t, repo.Create(ctx, initial))
	originalID := initial.ID

	// Import with same external_id but different values
	updated := &SettingsPreset{
		ExternalID:               "upsert-test",
		Name:                     "Updated Name",
		Description:              "Updated description",
		HasIssues:                false,
		HasWiki:                  true,
		AllowSquashMerge:         false,
		AllowMergeCommit:         true,
		DeleteBranchOnMerge:      false,
		SquashMergeCommitTitle:   "COMMIT_OR_PR_TITLE",
		SquashMergeCommitMessage: "PR_BODY",
		Labels: []SettingsPresetLabel{
			{Name: "new-label-1", Color: "aabbcc", Description: "New label 1"},
			{Name: "new-label-2", Color: "ddeeff", Description: "New label 2"},
			{Name: "new-label-3", Color: "112233", Description: "New label 3"},
		},
		Rulesets: []SettingsPresetRuleset{
			{
				Name:        "new-ruleset",
				Target:      "tag",
				Enforcement: "evaluate",
				Include:     JSONStringSlice{"~ALL"},
				Rules:       JSONStringSlice{"deletion", "update"},
			},
			{
				Name:        "another-ruleset",
				Target:      "branch",
				Enforcement: "active",
				Include:     JSONStringSlice{"~DEFAULT_BRANCH"},
				Rules:       JSONStringSlice{"pull_request"},
			},
		},
	}

	err := repo.ImportFromConfig(ctx, updated)
	require.NoError(t, err)

	// Should reuse the same DB ID
	assert.Equal(t, originalID, updated.ID)

	// Verify all fields were updated
	got, err := repo.GetByID(ctx, originalID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", got.Name)
	assert.Equal(t, "Updated description", got.Description)
	assert.False(t, got.HasIssues)
	assert.True(t, got.HasWiki)
	assert.False(t, got.AllowSquashMerge)
	assert.True(t, got.AllowMergeCommit)
	assert.False(t, got.DeleteBranchOnMerge)
	assert.Equal(t, "COMMIT_OR_PR_TITLE", got.SquashMergeCommitTitle)
	assert.Equal(t, "PR_BODY", got.SquashMergeCommitMessage)

	// Old labels should be replaced with new ones
	require.Len(t, got.Labels, 3)
	assert.Equal(t, "new-label-1", got.Labels[0].Name)
	assert.Equal(t, "new-label-2", got.Labels[1].Name)
	assert.Equal(t, "new-label-3", got.Labels[2].Name)

	// Old rulesets should be replaced with new ones
	require.Len(t, got.Rulesets, 2)
	assert.Equal(t, "new-ruleset", got.Rulesets[0].Name)
	assert.Equal(t, "another-ruleset", got.Rulesets[1].Name)

	// Verify no orphaned children remain from original creation
	var labelCount int64
	gormDB.Model(&SettingsPresetLabel{}).Where("settings_preset_id = ?", originalID).Count(&labelCount)
	assert.Equal(t, int64(3), labelCount)

	var rulesetCount int64
	gormDB.Model(&SettingsPresetRuleset{}).Where("settings_preset_id = ?", originalID).Count(&rulesetCount)
	assert.Equal(t, int64(2), rulesetCount)
}

func TestSettingsPresetRepository_ImportFromConfig_ReplaceChildren(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	// Create initial preset with labels and rulesets
	initial := createTestPreset("replace-children", "Replace Children Test")
	require.NoError(t, repo.Create(ctx, initial))
	originalID := initial.ID

	// Verify initial children
	got, err := repo.GetByID(ctx, originalID)
	require.NoError(t, err)
	require.Len(t, got.Labels, 2)
	require.Len(t, got.Rulesets, 1)

	// Import with no children (empty slices)
	empty := &SettingsPreset{
		ExternalID:  "replace-children",
		Name:        "No Children",
		Description: "Preset with no children",
	}

	err = repo.ImportFromConfig(ctx, empty)
	require.NoError(t, err)
	assert.Equal(t, originalID, empty.ID)

	// Children should be gone
	got, err = repo.GetByID(ctx, originalID)
	require.NoError(t, err)
	assert.Empty(t, got.Labels)
	assert.Empty(t, got.Rulesets)
}

func TestSettingsPresetRepository_AssignPresetToRepo(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	// Create prerequisite Client -> Organization -> Repo
	client := &Client{
		Name:        "TestClient",
		Description: "Test client for preset assignment",
	}
	require.NoError(t, gormDB.Create(client).Error)

	org := &Organization{
		ClientID:    client.ID,
		Name:        "test-org",
		Description: "Test organization",
	}
	require.NoError(t, gormDB.Create(org).Error)

	testRepo := &Repo{
		OrganizationID: org.ID,
		Name:           "test-repo",
		Description:    "Test repository",
	}
	require.NoError(t, gormDB.Create(testRepo).Error)

	// Create a preset
	preset := createTestPreset("assign-test", "Assignable Preset")
	require.NoError(t, repo.Create(ctx, preset))

	// Assign preset to repo
	err := repo.AssignPresetToRepo(ctx, testRepo.ID, preset.ID)
	require.NoError(t, err)

	// GetPresetForRepo should return the preset with children
	got, err := repo.GetPresetForRepo(ctx, testRepo.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, preset.ID, got.ID)
	assert.Equal(t, "assign-test", got.ExternalID)
	assert.Equal(t, "Assignable Preset", got.Name)
	require.Len(t, got.Labels, 2)
	require.Len(t, got.Rulesets, 1)
}

func TestSettingsPresetRepository_GetPresetForRepo_NoPresetAssigned(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	// Create prerequisite Client -> Organization -> Repo (no preset assigned)
	client := &Client{
		Name:        "TestClient2",
		Description: "Test client for no preset",
	}
	require.NoError(t, gormDB.Create(client).Error)

	org := &Organization{
		ClientID:    client.ID,
		Name:        "test-org-2",
		Description: "Test organization 2",
	}
	require.NoError(t, gormDB.Create(org).Error)

	testRepo := &Repo{
		OrganizationID: org.ID,
		Name:           "no-preset-repo",
		Description:    "Repo with no preset",
	}
	require.NoError(t, gormDB.Create(testRepo).Error)

	// GetPresetForRepo should return nil, nil when no preset assigned
	got, err := repo.GetPresetForRepo(ctx, testRepo.ID)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestSettingsPresetRepository_GetPresetForRepo_RepoNotFound(t *testing.T) {
	gormDB := TestDB(t)
	repo := NewSettingsPresetRepository(gormDB)
	ctx := context.Background()

	_, err := repo.GetPresetForRepo(ctx, 99999)
	require.ErrorIs(t, err, ErrRecordNotFound)
}
