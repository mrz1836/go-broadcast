package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

// TestSettingsIntegration_ScaffoldThenAudit tests the full flow:
// scaffold a repo → audit it → verify perfect score
func TestSettingsIntegration_ScaffoldThenAudit(t *testing.T) {
	ctx := context.Background()
	preset := config.DefaultPreset()

	// Build mock that tracks all calls
	mockClient := new(gh.MockClient)

	// Scaffold expects these calls
	mockClient.On("CreateRepository", ctx, gh.CreateRepoOptions{
		Name:        "mrz1836/integration-test",
		Description: "Integration test repo",
		Private:     true,
	}).Return(&gh.Repository{Name: "integration-test"}, nil)
	mockClient.On("UpdateRepoSettings", ctx, "mrz1836/integration-test", presetToRepoSettings(&preset)).Return(nil)
	mockClient.On("SyncLabels", ctx, "mrz1836/integration-test", presetLabelsToGH(preset.Labels)).Return(nil)
	for _, rc := range preset.Rulesets {
		mockClient.On("CreateOrUpdateRuleset", ctx, "mrz1836/integration-test", configRulesetToGH(&rc)).Return(nil)
	}

	// Step 1: Scaffold
	scaffoldOpts := ScaffoldOptions{
		Name:        "integration-test",
		Description: "Integration test repo",
		Owner:       "mrz1836",
		Preset:      &preset,
		NoClone:     true,
	}

	result, err := RunScaffold(ctx, mockClient, scaffoldOpts)
	require.NoError(t, err)
	assert.True(t, result.Created)

	// Step 2: Audit — mock GetRepoSettings returning matching values
	currentSettings := &gh.RepoSettings{
		HasIssues:                preset.HasIssues,
		HasWiki:                  preset.HasWiki,
		HasProjects:              preset.HasProjects,
		HasDiscussions:           preset.HasDiscussions,
		AllowSquashMerge:         preset.AllowSquashMerge,
		AllowMergeCommit:         preset.AllowMergeCommit,
		AllowRebaseMerge:         preset.AllowRebaseMerge,
		DeleteBranchOnMerge:      preset.DeleteBranchOnMerge,
		AllowAutoMerge:           preset.AllowAutoMerge,
		AllowUpdateBranch:        preset.AllowUpdateBranch,
		SquashMergeCommitTitle:   preset.SquashMergeCommitTitle,
		SquashMergeCommitMessage: preset.SquashMergeCommitMessage,
	}
	mockClient.On("GetRepoSettings", ctx, "mrz1836/integration-test").Return(currentSettings, nil)

	auditResult := auditSingleRepo(ctx, mockClient, "mrz1836/integration-test", preset.ID)
	assert.Empty(t, auditResult.Error)
	assert.Equal(t, 100, auditResult.Score)
	assert.Equal(t, 0, auditResult.Failed)
	assert.Equal(t, 12, auditResult.Total)

	mockClient.AssertExpectations(t)
}

// TestSettingsIntegration_AuditWithDrift tests auditing a repo that has drifted from preset
func TestSettingsIntegration_AuditWithDrift(t *testing.T) {
	ctx := context.Background()
	preset := config.DefaultPreset()

	mockClient := new(gh.MockClient)

	// Return settings with wiki enabled (drift from preset which has wiki=false)
	currentSettings := &gh.RepoSettings{
		HasIssues:                true,
		HasWiki:                  true, // DRIFT: should be false
		HasProjects:              false,
		HasDiscussions:           false,
		AllowSquashMerge:         true,
		AllowMergeCommit:         true, // DRIFT: should be false
		AllowRebaseMerge:         false,
		DeleteBranchOnMerge:      true,
		AllowAutoMerge:           true,
		AllowUpdateBranch:        true,
		SquashMergeCommitTitle:   preset.SquashMergeCommitTitle,
		SquashMergeCommitMessage: preset.SquashMergeCommitMessage,
	}
	mockClient.On("GetRepoSettings", ctx, "owner/drifted-repo").Return(currentSettings, nil)

	result := auditSingleRepo(ctx, mockClient, "owner/drifted-repo", preset.ID)
	assert.Empty(t, result.Error)
	assert.Less(t, result.Score, 100)
	assert.Equal(t, 2, result.Failed) // has_wiki + allow_merge_commit
	assert.Equal(t, 10, result.Passed)
}

// TestSettingsIntegration_DBPresetImportAndList tests importing presets from config
// and listing them back from the database
func TestSettingsIntegration_DBPresetImportAndList(t *testing.T) {
	tmpDir := t.TempDir()
	tmpPath := filepath.Join(tmpDir, "test-integration.db")

	oldDBPath := dbPath
	dbPath = tmpPath
	defer func() { dbPath = oldDBPath }()

	database, err := db.Open(db.OpenOptions{
		Path:        tmpPath,
		AutoMigrate: true,
	})
	require.NoError(t, err)

	ctx := context.Background()
	presetRepo := db.NewSettingsPresetRepository(database.DB())

	// Import a preset
	preset := &db.SettingsPreset{
		ExternalID:          "integration-test",
		Name:                "Integration Test",
		HasIssues:           true,
		AllowSquashMerge:    true,
		DeleteBranchOnMerge: true,
		Labels: []db.SettingsPresetLabel{
			{Name: "bug", Color: "d73a4a", Description: "Bug"},
		},
		Rulesets: []db.SettingsPresetRuleset{
			{
				Name: "bp", Target: "branch", Enforcement: "active",
				Include: db.JSONStringSlice{"~DEFAULT_BRANCH"},
				Rules:   db.JSONStringSlice{"pull_request"},
			},
		},
	}
	require.NoError(t, presetRepo.ImportFromConfig(ctx, preset))

	// List should return 1
	presets, err := presetRepo.List(ctx)
	require.NoError(t, err)
	require.Len(t, presets, 1)
	assert.Equal(t, "integration-test", presets[0].ExternalID)
	assert.Len(t, presets[0].Labels, 1)
	assert.Len(t, presets[0].Rulesets, 1)

	// Import again (upsert) — should still be 1 preset
	preset.Name = "Updated Integration Test"
	require.NoError(t, presetRepo.ImportFromConfig(ctx, preset))

	presets, err = presetRepo.List(ctx)
	require.NoError(t, err)
	require.Len(t, presets, 1)
	assert.Equal(t, "Updated Integration Test", presets[0].Name)

	require.NoError(t, database.Close())
}
