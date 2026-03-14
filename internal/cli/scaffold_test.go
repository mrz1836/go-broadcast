package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

func TestRunScaffold_DryRun(t *testing.T) {
	ctx := context.Background()
	preset := config.DefaultPreset()

	opts := ScaffoldOptions{
		Name:    "my-repo",
		Owner:   "acme",
		Preset:  &preset,
		DryRun:  true,
		Topics:  []string{"go", "library"},
		NoClone: false,
	}

	result, err := RunScaffold(ctx, nil, opts)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "acme/my-repo", result.RepoFullName)
	assert.False(t, result.Created)
}

func TestRunScaffold_CreateSuccess(t *testing.T) {
	ctx := context.Background()
	preset := config.DefaultPreset()

	mockClient := new(gh.MockClient)
	mockClient.On("CreateRepository", ctx, gh.CreateRepoOptions{
		Name:        "acme/my-repo",
		Description: "A test repo",
		Private:     true,
	}).Return(&gh.Repository{Name: "my-repo"}, nil)
	mockClient.On("UpdateRepoSettings", ctx, "acme/my-repo", presetToRepoSettings(&preset)).Return(nil)
	mockClient.On("SetTopics", ctx, "acme/my-repo", []string{"go"}).Return(nil)
	mockClient.On("SyncLabels", ctx, "acme/my-repo", presetLabelsToGH(preset.Labels)).Return(nil)
	for _, rc := range preset.Rulesets {
		mockClient.On("CreateOrUpdateRuleset", ctx, "acme/my-repo", configRulesetToGH(&rc)).Return(nil)
	}

	opts := ScaffoldOptions{
		Name:        "my-repo",
		Description: "A test repo",
		Owner:       "acme",
		Preset:      &preset,
		Topics:      []string{"go"},
	}

	result, err := RunScaffold(ctx, mockClient, opts)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Created)
	assert.Equal(t, "acme/my-repo", result.RepoFullName)
	mockClient.AssertExpectations(t)
}

func TestRunScaffold_CreateFailure(t *testing.T) {
	ctx := context.Background()
	preset := config.DefaultPreset()

	mockClient := new(gh.MockClient)
	mockClient.On("CreateRepository", ctx, gh.CreateRepoOptions{
		Name:        "acme/fail-repo",
		Description: "fail",
		Private:     true,
	}).Return((*gh.Repository)(nil), assert.AnError)

	opts := ScaffoldOptions{
		Name:        "fail-repo",
		Description: "fail",
		Owner:       "acme",
		Preset:      &preset,
	}

	result, err := RunScaffold(ctx, mockClient, opts)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create repository")
}

func TestPresetToRepoSettings(t *testing.T) {
	preset := &config.SettingsPreset{
		HasIssues:                true,
		HasWiki:                  false,
		AllowSquashMerge:         true,
		AllowMergeCommit:         false,
		AllowRebaseMerge:         false,
		DeleteBranchOnMerge:      true,
		AllowAutoMerge:           true,
		AllowUpdateBranch:        true,
		SquashMergeCommitTitle:   "PR_TITLE",
		SquashMergeCommitMessage: "PR_BODY",
	}

	settings := presetToRepoSettings(preset)
	assert.True(t, settings.HasIssues)
	assert.False(t, settings.HasWiki)
	assert.True(t, settings.AllowSquashMerge)
	assert.False(t, settings.AllowMergeCommit)
	assert.True(t, settings.DeleteBranchOnMerge)
	assert.Equal(t, "PR_TITLE", settings.SquashMergeCommitTitle)
	assert.Equal(t, "PR_BODY", settings.SquashMergeCommitMessage)
}

func TestPresetLabelsToGH(t *testing.T) {
	labels := []config.LabelSpec{
		{Name: "bug", Color: "d73a4a", Description: "Something isn't working"},
		{Name: "enhancement", Color: "a2eeef", Description: "New feature"},
	}

	ghLabels := presetLabelsToGH(labels)
	require.Len(t, ghLabels, 2)
	assert.Equal(t, "bug", ghLabels[0].Name)
	assert.Equal(t, "d73a4a", ghLabels[0].Color)
	assert.Equal(t, "enhancement", ghLabels[1].Name)
}

func TestPresetLabelsToGH_Empty(t *testing.T) {
	ghLabels := presetLabelsToGH(nil)
	assert.Empty(t, ghLabels)
}

func TestConfigRulesetToGH_Branch(t *testing.T) {
	rc := &config.RulesetConfig{
		Name:    "branch-protection",
		Target:  "branch",
		Include: []string{"~DEFAULT_BRANCH"},
		Rules:   []string{"required_signatures", "pull_request"},
	}

	ruleset := configRulesetToGH(rc)
	assert.Equal(t, "branch-protection", ruleset.Name)
	assert.Equal(t, "branch", ruleset.Target)
	assert.Equal(t, "active", ruleset.Enforcement)
}

func TestConfigRulesetToGH_Tag(t *testing.T) {
	rc := &config.RulesetConfig{
		Name:    "tag-protection",
		Target:  "tag",
		Include: []string{"v*"},
		Rules:   []string{"required_signatures"},
	}

	ruleset := configRulesetToGH(rc)
	assert.Equal(t, "tag-protection", ruleset.Name)
	assert.Equal(t, "tag", ruleset.Target)
	assert.Equal(t, "active", ruleset.Enforcement)
}

func TestDbPresetToConfigPreset(t *testing.T) {
	compat := &dbSettingsPresetCompat{
		ExternalID:          "test-preset",
		Name:                "Test",
		HasIssues:           true,
		AllowSquashMerge:    true,
		DeleteBranchOnMerge: true,
		Labels: []dbLabelCompat{
			{Name: "bug", Color: "d73a4a", Description: "Bug"},
		},
		Rulesets: []dbRulesetCompat{
			{Name: "branch-protection", Target: "branch", Include: []string{"~DEFAULT_BRANCH"}, Rules: []string{"pull_request"}},
		},
	}

	preset := dbPresetToConfigPreset(compat)
	assert.Equal(t, "test-preset", preset.ID)
	assert.Equal(t, "Test", preset.Name)
	assert.True(t, preset.HasIssues)
	assert.True(t, preset.AllowSquashMerge)
	assert.True(t, preset.DeleteBranchOnMerge)
	require.Len(t, preset.Labels, 1)
	assert.Equal(t, "bug", preset.Labels[0].Name)
	require.Len(t, preset.Rulesets, 1)
	assert.Equal(t, "branch-protection", preset.Rulesets[0].Name)
}
