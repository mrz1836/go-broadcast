package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

func TestComputeSettingsDiffs_NoChanges(t *testing.T) {
	current := &gh.RepoSettings{
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
		SquashMergeCommitMessage: "PR_BODY",
	}
	preset := &config.SettingsPreset{
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
		SquashMergeCommitMessage: "PR_BODY",
	}

	diffs := computeSettingsDiffs(current, preset)
	assert.Empty(t, diffs)
}

func TestComputeSettingsDiffs_AllDifferent(t *testing.T) {
	current := &gh.RepoSettings{
		HasIssues:           false,
		HasWiki:             true,
		HasProjects:         true,
		HasDiscussions:      true,
		AllowSquashMerge:    false,
		AllowMergeCommit:    true,
		AllowRebaseMerge:    true,
		DeleteBranchOnMerge: false,
		AllowAutoMerge:      false,
		AllowUpdateBranch:   false,
	}
	preset := &config.SettingsPreset{
		HasIssues:           true,
		HasWiki:             false,
		HasProjects:         false,
		HasDiscussions:      false,
		AllowSquashMerge:    true,
		AllowMergeCommit:    false,
		AllowRebaseMerge:    false,
		DeleteBranchOnMerge: true,
		AllowAutoMerge:      true,
		AllowUpdateBranch:   true,
	}

	diffs := computeSettingsDiffs(current, preset)
	assert.Len(t, diffs, 10) // All 10 boolean settings differ
}

func TestComputeSettingsDiffs_PartialChanges(t *testing.T) {
	current := &gh.RepoSettings{
		HasIssues:      true,
		HasWiki:        true, // differs
		AllowAutoMerge: false,
	}
	preset := &config.SettingsPreset{
		HasIssues:      true,
		HasWiki:        false,
		AllowAutoMerge: true,
	}

	diffs := computeSettingsDiffs(current, preset)
	require.Len(t, diffs, 2)
	assert.Equal(t, "has_wiki", diffs[0].Name)
	assert.Equal(t, "true", diffs[0].Current)
	assert.Equal(t, "false", diffs[0].Expected)
	assert.Equal(t, "allow_auto_merge", diffs[1].Name)
}

func TestComputeSettingsDiffs_StringFieldsSkippedWhenEmpty(t *testing.T) {
	current := &gh.RepoSettings{
		SquashMergeCommitTitle:   "COMMIT_OR_PR_TITLE",
		SquashMergeCommitMessage: "COMMIT_MESSAGES",
	}
	preset := &config.SettingsPreset{
		// Empty string fields should not produce diffs
		SquashMergeCommitTitle:   "",
		SquashMergeCommitMessage: "",
	}

	diffs := computeSettingsDiffs(current, preset)
	// No string diffs since expected values are empty
	for _, d := range diffs {
		assert.NotEqual(t, "squash_merge_commit_title", d.Name)
		assert.NotEqual(t, "squash_merge_commit_message", d.Name)
	}
}

func TestComputeSettingsDiffs_StringFieldsChanged(t *testing.T) {
	current := &gh.RepoSettings{
		SquashMergeCommitTitle:   "COMMIT_OR_PR_TITLE",
		SquashMergeCommitMessage: "COMMIT_MESSAGES",
	}
	preset := &config.SettingsPreset{
		SquashMergeCommitTitle:   "PR_TITLE",
		SquashMergeCommitMessage: "PR_BODY",
	}

	diffs := computeSettingsDiffs(current, preset)
	// Should have 2 string diffs
	stringDiffs := 0
	for _, d := range diffs {
		if d.Name == "squash_merge_commit_title" || d.Name == "squash_merge_commit_message" {
			stringDiffs++
		}
	}
	assert.Equal(t, 2, stringDiffs)
}

func TestRunSettingsApply_InvalidRepo(t *testing.T) {
	err := runSettingsApply(t.Context(), "invalid-repo", "mvp", "", "", false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repo format")
}

func TestRunSettingsApply_DryRun(t *testing.T) {
	err := runSettingsApply(t.Context(), "acme/my-repo", "mvp", "", "", false, true)
	require.NoError(t, err)
}

func TestRunSettingsApply_DryRun_WithTopics(t *testing.T) {
	err := runSettingsApply(t.Context(), "acme/repo", "mvp", "go,library", "", false, true)
	require.NoError(t, err)
}

func TestRunSettingsApply_DryRun_WithDescription(t *testing.T) {
	err := runSettingsApply(t.Context(), "acme/repo", "mvp", "", "My library", false, true)
	require.NoError(t, err)
}
