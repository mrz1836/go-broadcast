package cli

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

// presetForTest returns the bundled "default" preset which resolvePreset can
// resolve without a database (the reserved id short-circuits DB lookup).
func presetForTest() *config.SettingsPreset {
	p := config.DefaultPreset()
	p.ID = reservedDefaultPresetID
	return &p
}

func TestRunSettingsApplyWithClient(t *testing.T) { //nolint:paralleltest // shared output globals
	ctx := context.Background()
	preset := presetForTest()

	t.Run("get settings error", func(t *testing.T) {
		ghMock := gh.NewMockClient()
		ghMock.On("GetRepoSettings", ctx, "acme/repo").Return(nil, errMockGH)

		err := runSettingsApplyWithClient(ctx, ghMock, "acme/repo", preset, "", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current settings")
		ghMock.AssertExpectations(t)
	})

	t.Run("no changes needed", func(t *testing.T) {
		// Point dbPath at a nonexistent file so updateRepoSettingsInDB no-ops.
		withDBPath(t, filepath.Join(t.TempDir(), "missing.db"))

		// Current settings already match the preset -> no UpdateRepoSettings call.
		current := presetToRepoSettings(preset)
		ghMock := gh.NewMockClient()
		ghMock.On("GetRepoSettings", ctx, "acme/repo").Return(&current, nil)
		ghMock.On("SyncLabels", ctx, "acme/repo", mock.Anything).Return(nil).Maybe()
		ghMock.On("CreateOrUpdateRuleset", ctx, "acme/repo", mock.Anything).Return(nil).Maybe()

		err := runSettingsApplyWithClient(ctx, ghMock, "acme/repo", preset, "", false)
		require.NoError(t, err)
		ghMock.AssertNotCalled(t, "UpdateRepoSettings", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("applies diff, topics, labels, rulesets", func(t *testing.T) {
		withDBPath(t, filepath.Join(t.TempDir(), "missing.db"))

		// Current settings differ from the preset so UpdateRepoSettings is called.
		current := gh.RepoSettings{} // all-false, will diff against preset
		ghMock := gh.NewMockClient()
		ghMock.On("GetRepoSettings", ctx, "acme/repo").Return(&current, nil)
		ghMock.On("UpdateRepoSettings", ctx, "acme/repo", presetToRepoSettings(preset)).Return(nil)
		ghMock.On("SetTopics", ctx, "acme/repo", []string{"go", "cli"}).Return(nil)
		ghMock.On("SyncLabels", ctx, "acme/repo", mock.Anything).Return(nil).Maybe()
		ghMock.On("CreateOrUpdateRuleset", ctx, "acme/repo", mock.Anything).Return(nil).Maybe()

		err := runSettingsApplyWithClient(ctx, ghMock, "acme/repo", preset, "go, cli", true)
		require.NoError(t, err)
		ghMock.AssertExpectations(t)
	})

	t.Run("update settings error propagates", func(t *testing.T) {
		withDBPath(t, filepath.Join(t.TempDir(), "missing.db"))

		current := gh.RepoSettings{}
		ghMock := gh.NewMockClient()
		ghMock.On("GetRepoSettings", ctx, "acme/repo").Return(&current, nil)
		ghMock.On("UpdateRepoSettings", ctx, "acme/repo", mock.Anything).Return(errMockGH)

		err := runSettingsApplyWithClient(ctx, ghMock, "acme/repo", preset, "", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to apply settings")
	})
}

func TestRunSettingsAuditWithClient(t *testing.T) { //nolint:paralleltest // shared output globals
	ctx := context.Background()

	t.Run("perfect score no failures", func(t *testing.T) {
		withDBPath(t, filepath.Join(t.TempDir(), "missing.db"))

		preset := presetForTest()
		current := presetToRepoSettings(preset)
		settings := &gh.RepoSettings{
			HasIssues: current.HasIssues, HasWiki: current.HasWiki,
			HasProjects: current.HasProjects, HasDiscussions: current.HasDiscussions,
			AllowSquashMerge: current.AllowSquashMerge, AllowMergeCommit: current.AllowMergeCommit,
			AllowRebaseMerge: current.AllowRebaseMerge, DeleteBranchOnMerge: current.DeleteBranchOnMerge,
			AllowAutoMerge: current.AllowAutoMerge, AllowUpdateBranch: current.AllowUpdateBranch,
			SquashMergeCommitTitle: current.SquashMergeCommitTitle, SquashMergeCommitMessage: current.SquashMergeCommitMessage,
		}

		ghMock := gh.NewMockClient()
		ghMock.On("GetRepoSettings", ctx, "acme/repo").Return(settings, nil)

		err := runSettingsAuditWithClient(ctx, ghMock, []string{"acme/repo"}, reservedDefaultPresetID, false, "table", false)
		require.NoError(t, err)
		ghMock.AssertExpectations(t)
	})

	t.Run("imperfect score returns failure error", func(t *testing.T) {
		withDBPath(t, filepath.Join(t.TempDir(), "missing.db"))

		ghMock := gh.NewMockClient()
		ghMock.On("GetRepoSettings", ctx, "acme/repo").Return(&gh.RepoSettings{}, nil)

		err := runSettingsAuditWithClient(ctx, ghMock, []string{"acme/repo"}, reservedDefaultPresetID, false, "table", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "completed with failures")
	})

	t.Run("get settings error recorded as failure", func(t *testing.T) {
		withDBPath(t, filepath.Join(t.TempDir(), "missing.db"))

		ghMock := gh.NewMockClient()
		ghMock.On("GetRepoSettings", ctx, "acme/repo").Return(nil, errMockGH)

		err := runSettingsAuditWithClient(ctx, ghMock, []string{"acme/repo"}, reservedDefaultPresetID, false, "json", true)
		require.Error(t, err)
		ghMock.AssertExpectations(t)
	})
}

func TestRunScaffoldWithClient(t *testing.T) { //nolint:paralleltest // shared output globals
	ctx := context.Background()

	t.Run("invalid repo format", func(t *testing.T) {
		err := runScaffoldWithClient(ctx, nil, "no-slash", "desc", reservedDefaultPresetID, "", true, true, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid repo format")
	})

	t.Run("dry run with nil client", func(t *testing.T) {
		err := runScaffoldWithClient(ctx, nil, "acme/repo", "desc", reservedDefaultPresetID, "go,cli", true, true, true)
		require.NoError(t, err)
	})

	t.Run("create flow with mock client", func(t *testing.T) {
		preset := presetForTest()
		ghMock := gh.NewMockClient()
		ghMock.On("CreateRepository", ctx, mock.Anything).Return(&gh.Repository{Name: "repo"}, nil)
		ghMock.On("UpdateRepoSettings", ctx, "acme/repo", mock.Anything).Return(nil)
		ghMock.On("SetTopics", ctx, "acme/repo", []string{"go"}).Return(nil)
		ghMock.On("RenameBranch", ctx, "acme/repo", "main", "master").Return(nil)
		ghMock.On("SyncLabels", ctx, "acme/repo", mock.Anything).Return(nil).Maybe()
		ghMock.On("CreateOrUpdateRuleset", ctx, "acme/repo", mock.Anything).Return(nil).Maybe()
		_ = preset

		// no-files, no-clone keeps the flow hermetic (no filesystem clone)
		err := runScaffoldWithClient(ctx, ghMock, "acme/repo", "desc", reservedDefaultPresetID, "go", true, true, false)
		require.NoError(t, err)
		ghMock.AssertExpectations(t)
	})

	t.Run("create failure propagates", func(t *testing.T) {
		ghMock := gh.NewMockClient()
		ghMock.On("CreateRepository", ctx, mock.Anything).Return(nil, errMockGH)

		err := runScaffoldWithClient(ctx, ghMock, "acme/repo", "desc", reservedDefaultPresetID, "", true, true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create repository")
	})
}

// TestRunScaffold_PublicDryRun covers the public runScaffold wrapper's dry-run
// branch, which delegates to runScaffoldWithClient with a nil client.
func TestRunScaffold_PublicDryRun(t *testing.T) { //nolint:paralleltest // shared output globals
	err := runScaffold(context.Background(), "acme/new-repo", "desc", reservedDefaultPresetID, "go,cli", "", false, false, true)
	require.NoError(t, err)
}
