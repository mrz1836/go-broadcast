package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/analytics"
	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

var errMockGH = errors.New("mock gh error")

// newTestPipeline builds an analytics.Pipeline backed by a mock GitHub client
// and the provided analytics repo. repoRepo/orgRepo are nil because the helpers
// under test do not use them. A silent logger and nil throttle keep it hermetic.
func newTestPipeline(ghClient gh.Client, repo db.AnalyticsRepo) *analytics.Pipeline {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	return analytics.NewPipeline(ghClient, repo, nil, nil, logger, nil)
}

func TestApplyRepoSettings(t *testing.T) {
	t.Parallel()

	t.Run("nil settings is a no-op", func(t *testing.T) {
		t.Parallel()
		repo := &db.Repo{AutoMergeEnabled: false}
		applyRepoSettings(repo, nil)
		assert.False(t, repo.AutoMergeEnabled)
		assert.False(t, repo.DependabotEnabled)
		assert.Empty(t, repo.SquashMergeCommitTitle)
	})

	t.Run("copies all settings", func(t *testing.T) {
		t.Parallel()
		settings := &gh.Repository{
			AllowAutoMerge:           true,
			AllowUpdateBranch:        true,
			AllowSquashMerge:         true,
			AllowMergeCommit:         true,
			AllowRebaseMerge:         true,
			DeleteBranchOnMerge:      true,
			SquashMergeCommitTitle:   "PR_TITLE",
			SquashMergeCommitMessage: "PR_BODY",
		}
		settings.SecurityAndAnalysis.DependabotSecurityUpdates.Status = "enabled"
		settings.SecurityAndAnalysis.SecretScanning.Status = "enabled"
		settings.SecurityAndAnalysis.SecretScanningPushProtection.Status = "enabled"

		repo := &db.Repo{}
		applyRepoSettings(repo, settings)

		assert.True(t, repo.AutoMergeEnabled)
		assert.True(t, repo.UpdateBranchEnabled)
		assert.True(t, repo.DependabotEnabled)
		assert.True(t, repo.SecretScanningEnabled)
		assert.True(t, repo.PushProtectionEnabled)
		assert.True(t, repo.AllowSquashMerge)
		assert.True(t, repo.AllowMergeCommit)
		assert.True(t, repo.AllowRebaseMerge)
		assert.True(t, repo.DeleteBranchOnMerge)
		assert.Equal(t, "PR_TITLE", repo.SquashMergeCommitTitle)
		assert.Equal(t, "PR_BODY", repo.SquashMergeCommitMessage)
	})

	t.Run("disabled security stays false", func(t *testing.T) {
		t.Parallel()
		settings := &gh.Repository{}
		settings.SecurityAndAnalysis.DependabotSecurityUpdates.Status = "disabled"
		repo := &db.Repo{}
		applyRepoSettings(repo, settings)
		assert.False(t, repo.DependabotEnabled)
	})
}

func TestGetAlertTotal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("sums counts", func(t *testing.T) {
		t.Parallel()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetAlertCountsByType", ctx, uint(1)).
			Return(map[string]int{"dependabot": 3, "code_scanning": 2}, nil)
		assert.Equal(t, 5, getAlertTotal(ctx, mockRepo, 1))
		mockRepo.AssertExpectations(t)
	})

	t.Run("error returns zero", func(t *testing.T) {
		t.Parallel()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetAlertCountsByType", ctx, uint(2)).
			Return(nil, errMockGH)
		assert.Equal(t, 0, getAlertTotal(ctx, mockRepo, 2))
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty counts returns zero", func(t *testing.T) {
		t.Parallel()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetAlertCountsByType", ctx, uint(3)).
			Return(map[string]int{}, nil)
		assert.Equal(t, 0, getAlertTotal(ctx, mockRepo, 3))
		mockRepo.AssertExpectations(t)
	})
}

func TestFetchContributorCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		ghMock.On("GetContributorCount", ctx, "org/repo").Return(7, nil)
		assert.Equal(t, 7, fetchContributorCount(ctx, ghMock, "org/repo"))
		ghMock.AssertExpectations(t)
	})

	t.Run("error returns zero", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		ghMock.On("GetContributorCount", ctx, "org/repo").Return(0, errMockGH)
		assert.Equal(t, 0, fetchContributorCount(ctx, ghMock, "org/repo"))
		ghMock.AssertExpectations(t)
	})
}

func TestUpdateSnapshotAlertCounts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("no alert counts is no-op", func(t *testing.T) {
		t.Parallel()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetAlertCountsByType", ctx, uint(1)).
			Return(map[string]int{}, nil)
		snap := &db.RepositorySnapshot{}
		snap.ID = 5
		updateSnapshotAlertCounts(ctx, mockRepo, 1, snap)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error fetching counts is no-op", func(t *testing.T) {
		t.Parallel()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetAlertCountsByType", ctx, uint(1)).
			Return(nil, errMockGH)
		snap := &db.RepositorySnapshot{}
		snap.ID = 5
		updateSnapshotAlertCounts(ctx, mockRepo, 1, snap)
		mockRepo.AssertExpectations(t)
	})

	t.Run("persisted snapshot updated in place", func(t *testing.T) {
		t.Parallel()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetAlertCountsByType", ctx, uint(1)).
			Return(map[string]int{"dependabot": 2, "code_scanning": 1, "secret_scanning": 3}, nil)
		snap := &db.RepositorySnapshot{}
		snap.ID = 5
		updateSnapshotAlertCounts(ctx, mockRepo, 1, snap)
		assert.Equal(t, 2, snap.DependabotAlertCount)
		assert.Equal(t, 1, snap.CodeScanningAlertCount)
		assert.Equal(t, 3, snap.SecretScanningAlertCount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("unpersisted snapshot falls back to latest", func(t *testing.T) {
		t.Parallel()
		latest := &db.RepositorySnapshot{}
		latest.ID = 9
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetAlertCountsByType", ctx, uint(1)).
			Return(map[string]int{"dependabot": 4}, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(1)).Return(latest, nil)
		snap := &db.RepositorySnapshot{} // ID == 0
		updateSnapshotAlertCounts(ctx, mockRepo, 1, snap)
		assert.Equal(t, 4, latest.DependabotAlertCount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("unpersisted snapshot with no latest is no-op", func(t *testing.T) {
		t.Parallel()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetAlertCountsByType", ctx, uint(1)).
			Return(map[string]int{"dependabot": 4}, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(1)).Return(nil, errMockGH)
		snap := &db.RepositorySnapshot{} // ID == 0
		updateSnapshotAlertCounts(ctx, mockRepo, 1, snap)
		mockRepo.AssertExpectations(t)
	})
}

func TestCollectSecurityAlerts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	setupNoAlerts := func(ghMock *gh.MockClient, repo string) {
		// CollectAlerts runs in an errgroup which derives a child context, so
		// match the context argument loosely.
		ghMock.On("GetDependabotAlerts", mock.Anything, repo).Return([]gh.DependabotAlert{}, nil)
		ghMock.On("GetCodeScanningAlerts", mock.Anything, repo).Return([]gh.CodeScanningAlert{}, nil)
		ghMock.On("GetSecretScanningAlerts", mock.Anything, repo).Return([]gh.SecretScanningAlert{}, nil)
	}

	t.Run("no open alerts still reconciles stale", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		setupNoAlerts(ghMock, "org/repo")
		// mockAnalyticsRepo.CloseStaleAlerts is a fixed no-op returning (0, nil).
		mockRepo := new(mockAnalyticsRepo)

		pipe := newTestPipeline(ghMock, mockRepo)
		out, err := collectSecurityAlerts(ctx, pipe, mockRepo, 1, "org/repo")
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, 0, out.AlertCount)
		ghMock.AssertExpectations(t)
	})
}

func TestCollectCIMetrics(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("no GoFortress workflow returns nil error", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		ghMock.On("ListWorkflows", mock.Anything, "org/repo").Return([]gh.Workflow{}, nil)

		mockRepo := new(mockAnalyticsRepo)
		pipe := newTestPipeline(ghMock, mockRepo)
		err := collectCIMetrics(ctx, pipe, mockRepo, 1, "org/repo")
		require.NoError(t, err)
		// CreateCISnapshot must not be called when there is no workflow.
		mockRepo.AssertNotCalled(t, "CreateCISnapshot", mock.Anything, mock.Anything)
	})

	t.Run("list workflows error is swallowed by collector", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		ghMock.On("ListWorkflows", mock.Anything, "org/repo").Return(nil, errMockGH)

		mockRepo := new(mockAnalyticsRepo)
		pipe := newTestPipeline(ghMock, mockRepo)
		// Collector logs the error as a warning and returns nil metrics; no DB write.
		err := collectCIMetrics(ctx, pipe, mockRepo, 1, "org/repo")
		require.NoError(t, err)
		mockRepo.AssertNotCalled(t, "CreateCISnapshot", mock.Anything, mock.Anything)
	})

	t.Run("GoFortress workflow with run creates snapshot", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		ghMock.On("ListWorkflows", mock.Anything, "org/repo").
			Return([]gh.Workflow{{ID: 99, Name: "GoFortress"}}, nil)
		ghMock.On("GetWorkflowRuns", mock.Anything, "org/repo", int64(99), mock.Anything).
			Return([]gh.WorkflowRun{{ID: 1234, HeadBranch: "master", HeadSHA: "abc"}}, nil)
		// No artifacts -> metrics returned without coverage (fallback) -> snapshot created.
		ghMock.On("GetRunArtifacts", mock.Anything, "org/repo", int64(1234)).
			Return([]gh.Artifact{}, nil)

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("CreateCISnapshot", ctx, mock.Anything).Return(nil)
		pipe := newTestPipeline(ghMock, mockRepo)
		err := collectCIMetrics(ctx, pipe, mockRepo, 1, "org/repo")
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

func TestCollectSecurityAlerts_WithAlerts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ghMock := gh.NewMockClient()
	// One code-scanning alert; dependabot and secret scanning empty.
	ghMock.On("GetDependabotAlerts", mock.Anything, "org/repo").Return([]gh.DependabotAlert{}, nil)
	ghMock.On("GetCodeScanningAlerts", mock.Anything, "org/repo").
		Return([]gh.CodeScanningAlert{{Number: 7, State: "open"}}, nil)
	ghMock.On("GetSecretScanningAlerts", mock.Anything, "org/repo").Return([]gh.SecretScanningAlert{}, nil)

	mockRepo := new(mockAnalyticsRepo)
	mockRepo.On("UpsertAlert", ctx, mock.Anything).Return(nil)

	pipe := newTestPipeline(ghMock, mockRepo)
	out, err := collectSecurityAlerts(ctx, pipe, mockRepo, 1, "org/repo")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, 1, out.AlertCount)
	mockRepo.AssertExpectations(t)
}

// setupNoSecurityOrCI wires the gh mock so the security + CI collectors return
// no alerts/metrics without errors (keeping the full sync path hermetic).
func setupNoSecurityOrCI(ghMock *gh.MockClient, repo string) {
	ghMock.On("GetDependabotAlerts", mock.Anything, repo).Return([]gh.DependabotAlert{}, nil)
	ghMock.On("GetCodeScanningAlerts", mock.Anything, repo).Return([]gh.CodeScanningAlert{}, nil)
	ghMock.On("GetSecretScanningAlerts", mock.Anything, repo).Return([]gh.SecretScanningAlert{}, nil)
	ghMock.On("ListWorkflows", mock.Anything, repo).Return([]gh.Workflow{}, nil)
	ghMock.On("GetContributorCount", mock.Anything, repo).Return(0, nil)
}

func TestSyncRepositoryMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	meta := &analytics.RepoMetadata{FullName: "org/repo", Stars: 5, Forks: 2}

	t.Run("success creates snapshot", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		ghMock.On("GetRepository", mock.Anything, "org/repo").Return(&gh.Repository{}, nil)
		setupNoSecurityOrCI(ghMock, "org/repo")

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("UpsertOrganization", ctx, mock.Anything).Return(nil)
		mockRepo.On("UpsertRepository", ctx, mock.Anything).Return(nil)
		// forceFull=true => CreateSnapshot called without GetLatestSnapshot.
		mockRepo.On("CreateSnapshot", ctx, mock.Anything).Return(nil)
		mockRepo.On("GetAlertCountsByType", ctx, mock.Anything).Return(map[string]int{}, nil)
		mockRepo.On("UpdateRepoSyncTimestamp", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pipe := newTestPipeline(ghMock, mockRepo)
		run := &db.SyncRun{}
		err := syncRepositoryMetadata(ctx, pipe, mockRepo, run, meta, false, true)
		require.NoError(t, err)
		assert.Equal(t, 1, run.SnapshotsCreated)
		assert.Equal(t, 1, run.ReposProcessed)
	})

	t.Run("upsert organization error", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("UpsertOrganization", ctx, mock.Anything).Return(errMockGH)

		pipe := newTestPipeline(ghMock, mockRepo)
		run := &db.SyncRun{}
		err := syncRepositoryMetadata(ctx, pipe, mockRepo, run, meta, true, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upsert organization")
	})

	t.Run("upsert repository error", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		ghMock.On("GetRepository", mock.Anything, "org/repo").Return(&gh.Repository{}, nil)
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("UpsertOrganization", ctx, mock.Anything).Return(nil)
		mockRepo.On("UpsertRepository", ctx, mock.Anything).Return(errMockGH)

		pipe := newTestPipeline(ghMock, mockRepo)
		run := &db.SyncRun{}
		err := syncRepositoryMetadata(ctx, pipe, mockRepo, run, meta, false, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upsert repository")
	})

	t.Run("no changes skips snapshot", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		ghMock.On("GetRepository", mock.Anything, "org/repo").Return(&gh.Repository{}, nil)
		setupNoSecurityOrCI(ghMock, "org/repo")

		// A previous snapshot identical to the new one => HasChanged is false.
		prev := buildRepositorySnapshot(meta, 0)
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("UpsertOrganization", ctx, mock.Anything).Return(nil)
		mockRepo.On("UpsertRepository", ctx, mock.Anything).Return(nil)
		mockRepo.On("GetLatestSnapshot", ctx, mock.Anything).Return(prev, nil)
		mockRepo.On("GetAlertCountsByType", ctx, mock.Anything).Return(map[string]int{}, nil)
		mockRepo.On("UpdateRepoSyncTimestamp", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pipe := newTestPipeline(ghMock, mockRepo)
		run := &db.SyncRun{}
		err := syncRepositoryMetadata(ctx, pipe, mockRepo, run, meta, true, false)
		require.NoError(t, err)
		assert.Equal(t, 1, run.ReposSkipped)
		mockRepo.AssertNotCalled(t, "CreateSnapshot", mock.Anything, mock.Anything)
	})
}

func TestSyncSingleRepository_DryRun(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ghMock := gh.NewMockClient()
	mockRepo := new(mockAnalyticsRepo)
	pipe := newTestPipeline(ghMock, mockRepo)
	run := &db.SyncRun{}

	err := syncSingleRepository(ctx, pipe, mockRepo, run, "org", "repo", true, true, false)
	require.NoError(t, err)
	assert.Equal(t, 1, run.ReposProcessed)
}

func TestSyncSingleRepository_FullPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("metadata fetch failure", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		// ExecuteGraphQL returns no usable repo data -> SyncRepository errors with
		// "metadata not returned".
		ghMock.On("ExecuteGraphQL", mock.Anything, mock.Anything).
			Return(map[string]interface{}{}, nil)
		mockRepo := new(mockAnalyticsRepo)
		pipe := newTestPipeline(ghMock, mockRepo)
		run := &db.SyncRun{}
		err := syncSingleRepository(ctx, pipe, mockRepo, run, "org", "repo", true, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to sync repository")
	})

	t.Run("success persists snapshot", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		// Return a parseable single-repo GraphQL batch response (alias repo0).
		ghMock.On("ExecuteGraphQL", mock.Anything, mock.Anything).Return(map[string]interface{}{
			"repo0": map[string]interface{}{
				"nameWithOwner":  "org/repo",
				"stargazerCount": float64(5),
				"forkCount":      float64(2),
			},
		}, nil)
		ghMock.On("GetRepository", mock.Anything, "org/repo").Return(&gh.Repository{}, nil)
		setupNoSecurityOrCI(ghMock, "org/repo")

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("UpsertOrganization", ctx, mock.Anything).Return(nil)
		mockRepo.On("UpsertRepository", ctx, mock.Anything).Return(nil)
		mockRepo.On("CreateSnapshot", ctx, mock.Anything).Return(nil)
		mockRepo.On("GetAlertCountsByType", ctx, mock.Anything).Return(map[string]int{}, nil)
		mockRepo.On("UpdateRepoSyncTimestamp", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		pipe := newTestPipeline(ghMock, mockRepo)
		run := &db.SyncRun{}
		// forceFull=true avoids GetLatestSnapshot; showProgress=true exercises the
		// success summary branch.
		err := syncSingleRepository(ctx, pipe, mockRepo, run, "org", "repo", true, false, true)
		require.NoError(t, err)
		assert.Equal(t, 1, run.ReposProcessed)
		assert.Equal(t, 1, run.SnapshotsCreated)
	})

	t.Run("upsert organization error", func(t *testing.T) {
		t.Parallel()
		ghMock := gh.NewMockClient()
		ghMock.On("ExecuteGraphQL", mock.Anything, mock.Anything).Return(map[string]interface{}{
			"repo0": map[string]interface{}{"nameWithOwner": "org/repo", "stargazerCount": float64(1)},
		}, nil)
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("UpsertOrganization", ctx, mock.Anything).Return(errMockGH)
		pipe := newTestPipeline(ghMock, mockRepo)
		run := &db.SyncRun{}
		err := syncSingleRepository(ctx, pipe, mockRepo, run, "org", "repo", false, false, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upsert organization")
	})
}

func TestSyncAllOrganizations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("list error", func(t *testing.T) {
		t.Parallel()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("ListOrganizations", ctx).Return(nil, errMockGH)
		pipe := newTestPipeline(gh.NewMockClient(), mockRepo)
		err := syncAllOrganizations(ctx, pipe, mockRepo, &db.SyncRun{}, false, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list organizations")
	})

	t.Run("no organizations", func(t *testing.T) {
		t.Parallel()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("ListOrganizations", ctx).Return([]db.Organization{}, nil)
		pipe := newTestPipeline(gh.NewMockClient(), mockRepo)
		err := syncAllOrganizations(ctx, pipe, mockRepo, &db.SyncRun{}, false, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no organizations found")
	})

	t.Run("dry run", func(t *testing.T) {
		t.Parallel()
		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("ListOrganizations", ctx).Return([]db.Organization{{Name: "org-a"}, {Name: "org-b"}}, nil)
		pipe := newTestPipeline(gh.NewMockClient(), mockRepo)
		run := &db.SyncRun{}
		err := syncAllOrganizations(ctx, pipe, mockRepo, run, true, true, false)
		require.NoError(t, err)
		assert.Equal(t, 2, run.ReposProcessed)
	})
}

func TestSyncOrganization(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Build a pipeline backed by a real (in-memory) DB so SyncOrganization can
	// query configured repos. The seeded org has no analytics repos, so
	// SyncOrganization returns an empty metadata map (no GraphQL needed).
	gormDB := db.TestDB(t)
	client := &db.Client{Name: "emptyorg"}
	require.NoError(t, gormDB.Create(client).Error)
	require.NoError(t, gormDB.Create(&db.Organization{ClientID: client.ID, Name: "emptyorg"}).Error)

	analyticsRepo := db.NewAnalyticsRepo(gormDB)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)
	pipe := analytics.NewPipeline(
		gh.NewMockClient(), analyticsRepo,
		db.NewRepoRepository(gormDB), db.NewOrganizationRepository(gormDB),
		logger, nil,
	)

	t.Run("no configured repos", func(t *testing.T) {
		t.Parallel()
		run := &db.SyncRun{}
		require.NoError(t, syncOrganization(ctx, pipe, analyticsRepo, run, "emptyorg", true, false, false))
	})

	t.Run("dry run", func(t *testing.T) {
		t.Parallel()
		run := &db.SyncRun{}
		require.NoError(t, syncOrganization(ctx, pipe, analyticsRepo, run, "emptyorg", true, true, false))
	})

	t.Run("org not found errors", func(t *testing.T) {
		t.Parallel()
		run := &db.SyncRun{}
		err := syncOrganization(ctx, pipe, analyticsRepo, run, "ghost", false, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to sync owner")
	})
}

func TestSyncManagedRepos(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("no managed repos", func(t *testing.T) {
		t.Parallel()
		gormDB := db.TestDB(t)
		mockRepo := new(mockAnalyticsRepo)
		pipe := newTestPipeline(gh.NewMockClient(), mockRepo)
		err := syncManagedRepos(ctx, gormDB, pipe, mockRepo, &db.SyncRun{}, false, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no managed repos found")
	})

	t.Run("dry run lists managed repos", func(t *testing.T) {
		t.Parallel()
		gormDB, _ := db.TestDBWithSeed(t)
		mockRepo := new(mockAnalyticsRepo)
		pipe := newTestPipeline(gh.NewMockClient(), mockRepo)
		run := &db.SyncRun{}
		err := syncManagedRepos(ctx, gormDB, pipe, mockRepo, run, true, true, false)
		require.NoError(t, err)
		assert.Positive(t, run.ReposProcessed)
	})

	t.Run("non-dry-run with metadata failures continues", func(t *testing.T) {
		t.Parallel()
		gormDB, _ := db.TestDBWithSeed(t)
		ghMock := gh.NewMockClient()
		// SyncRepository -> ExecuteGraphQL returns no usable data -> each repo
		// errors and is recorded, but the loop continues to completion.
		ghMock.On("ExecuteGraphQL", mock.Anything, mock.Anything).
			Return(map[string]interface{}{}, nil)
		mockRepo := new(mockAnalyticsRepo)
		pipe := newTestPipeline(ghMock, mockRepo)
		run := &db.SyncRun{}
		err := syncManagedRepos(ctx, gormDB, pipe, mockRepo, run, true, false, false)
		require.NoError(t, err)
		assert.Positive(t, run.ReposFailed)
	})
}
