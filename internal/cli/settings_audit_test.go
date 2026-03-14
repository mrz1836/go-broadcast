package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

func TestRunAuditChecks_AllPass(t *testing.T) {
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

	checks := runAuditChecks(current, preset)
	require.Len(t, checks, 12) // 10 bools + 2 strings

	for _, c := range checks {
		assert.True(t, c.Pass, "expected %s to pass", c.Setting)
	}
}

func TestRunAuditChecks_AllFail(t *testing.T) {
	current := &gh.RepoSettings{
		HasIssues:                false,
		HasWiki:                  true,
		HasProjects:              true,
		HasDiscussions:           true,
		AllowSquashMerge:         false,
		AllowMergeCommit:         true,
		AllowRebaseMerge:         true,
		DeleteBranchOnMerge:      false,
		AllowAutoMerge:           false,
		AllowUpdateBranch:        false,
		SquashMergeCommitTitle:   "COMMIT_OR_PR_TITLE",
		SquashMergeCommitMessage: "COMMIT_MESSAGES",
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

	checks := runAuditChecks(current, preset)
	require.Len(t, checks, 12)

	for _, c := range checks {
		assert.False(t, c.Pass, "expected %s to fail", c.Setting)
	}
}

func TestRunAuditChecks_EmptyStringSkipped(t *testing.T) {
	current := &gh.RepoSettings{
		SquashMergeCommitTitle:   "PR_TITLE",
		SquashMergeCommitMessage: "PR_BODY",
	}
	preset := &config.SettingsPreset{
		// Empty strings should be skipped
		SquashMergeCommitTitle:   "",
		SquashMergeCommitMessage: "",
	}

	checks := runAuditChecks(current, preset)
	// Should only have 10 boolean checks, no string checks
	assert.Len(t, checks, 10)
}

func TestAuditSingleRepo_Error(t *testing.T) {
	mockClient := new(gh.MockClient)
	mockClient.On("GetRepoSettings", t.Context(), "owner/repo").Return((*gh.RepoSettings)(nil), assert.AnError)

	result := auditSingleRepo(t.Context(), mockClient, "owner/repo", "mvp")
	assert.NotEmpty(t, result.Error)
	assert.Equal(t, "owner/repo", result.Repo)
}

func TestAuditSingleRepo_PerfectScore(t *testing.T) {
	preset := config.DefaultPreset()
	current := &gh.RepoSettings{
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

	mockClient := new(gh.MockClient)
	mockClient.On("GetRepoSettings", t.Context(), "owner/repo").Return(current, nil)

	result := auditSingleRepo(t.Context(), mockClient, "owner/repo", "mvp")
	assert.Empty(t, result.Error)
	assert.Equal(t, 100, result.Score)
	assert.Equal(t, 0, result.Failed)
}

func TestResolveAuditRepos_WithRepos(t *testing.T) {
	repos, err := resolveAuditRepos(t.Context(), []string{"owner/repo1", "owner/repo2"}, "", false)
	require.NoError(t, err)
	assert.Equal(t, []string{"owner/repo1", "owner/repo2"}, repos)
}

func TestResolveAuditRepos_NoArgs(t *testing.T) {
	_, err := resolveAuditRepos(t.Context(), nil, "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "specify repos, --org, or --all")
}
