package analytics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/db"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

var (
	errGraphQLError           = errors.New("GraphQL error")
	errQueryTooComplex        = errors.New("query is too complex")
	errQueryCostExceeded      = errors.New("query cost exceeded")
	errComplexityLimitReached = errors.New("complexity limit reached")
	errNetworkTimeout         = errors.New("network timeout")
	errCreateSyncRun          = errors.New("create sync run failed")
	errUpdateSyncRun          = errors.New("update sync run failed")
	errOrgNotFoundDB          = errors.New("org not found in db")
	errRepoQueryFailed        = errors.New("repo query failed")
)

// --- Mock implementations for db interfaces ---

type mockAnalyticsRepoPipeline struct {
	mock.Mock
}

func (m *mockAnalyticsRepoPipeline) UpsertOrganization(ctx context.Context, org *db.Organization) error {
	return m.Called(ctx, org).Error(0)
}

func (m *mockAnalyticsRepoPipeline) GetOrganization(ctx context.Context, login string) (*db.Organization, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Organization), args.Error(1)
}

func (m *mockAnalyticsRepoPipeline) ListOrganizations(ctx context.Context) ([]db.Organization, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.Organization), args.Error(1)
}

func (m *mockAnalyticsRepoPipeline) UpsertRepository(ctx context.Context, repo *db.Repo) error {
	return m.Called(ctx, repo).Error(0)
}

func (m *mockAnalyticsRepoPipeline) GetRepository(ctx context.Context, fullName string) (*db.Repo, error) {
	args := m.Called(ctx, fullName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Repo), args.Error(1)
}

func (m *mockAnalyticsRepoPipeline) ListRepositories(ctx context.Context, orgLogin string) ([]db.Repo, error) {
	args := m.Called(ctx, orgLogin)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.Repo), args.Error(1)
}

func (m *mockAnalyticsRepoPipeline) CreateSnapshot(ctx context.Context, snap *db.RepositorySnapshot) error {
	return m.Called(ctx, snap).Error(0)
}

func (m *mockAnalyticsRepoPipeline) GetLatestSnapshot(ctx context.Context, repoID uint) (*db.RepositorySnapshot, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.RepositorySnapshot), args.Error(1)
}

func (m *mockAnalyticsRepoPipeline) GetSnapshotHistory(ctx context.Context, repoID uint, since time.Time) ([]db.RepositorySnapshot, error) {
	args := m.Called(ctx, repoID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.RepositorySnapshot), args.Error(1)
}

func (m *mockAnalyticsRepoPipeline) UpdateSnapshotAlertCounts(_ context.Context, _ *db.RepositorySnapshot) error {
	return nil
}

func (m *mockAnalyticsRepoPipeline) UpsertAlert(ctx context.Context, alert *db.SecurityAlert) error {
	return m.Called(ctx, alert).Error(0)
}

func (m *mockAnalyticsRepoPipeline) GetOpenAlerts(ctx context.Context, repoID uint, severity string) ([]db.SecurityAlert, error) {
	args := m.Called(ctx, repoID, severity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.SecurityAlert), args.Error(1)
}

func (m *mockAnalyticsRepoPipeline) GetAlertCounts(ctx context.Context, repoID uint) (map[string]int, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *mockAnalyticsRepoPipeline) GetAlertCountsByType(ctx context.Context, repoID uint) (map[string]int, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *mockAnalyticsRepoPipeline) CreateCISnapshot(ctx context.Context, snap *db.CIMetricsSnapshot) error {
	return m.Called(ctx, snap).Error(0)
}

func (m *mockAnalyticsRepoPipeline) GetLatestCISnapshot(ctx context.Context, repoID uint) (*db.CIMetricsSnapshot, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.CIMetricsSnapshot), args.Error(1)
}

func (m *mockAnalyticsRepoPipeline) CreateSyncRun(ctx context.Context, run *db.SyncRun) error {
	return m.Called(ctx, run).Error(0)
}

func (m *mockAnalyticsRepoPipeline) UpdateSyncRun(ctx context.Context, run *db.SyncRun) error {
	return m.Called(ctx, run).Error(0)
}

func (m *mockAnalyticsRepoPipeline) GetLatestSyncRun(ctx context.Context) (*db.SyncRun, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.SyncRun), args.Error(1)
}

type mockOrgRepository struct {
	mock.Mock
}

func (m *mockOrgRepository) Create(ctx context.Context, org *db.Organization) error {
	return m.Called(ctx, org).Error(0)
}

func (m *mockOrgRepository) GetByID(ctx context.Context, id uint) (*db.Organization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Organization), args.Error(1)
}

func (m *mockOrgRepository) GetByName(ctx context.Context, name string) (*db.Organization, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Organization), args.Error(1)
}

func (m *mockOrgRepository) Update(ctx context.Context, org *db.Organization) error {
	return m.Called(ctx, org).Error(0)
}

func (m *mockOrgRepository) Delete(ctx context.Context, id uint, hard bool) error {
	return m.Called(ctx, id, hard).Error(0)
}

func (m *mockOrgRepository) List(ctx context.Context, clientID uint) ([]*db.Organization, error) {
	args := m.Called(ctx, clientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*db.Organization), args.Error(1)
}

func (m *mockOrgRepository) ListWithRepos(ctx context.Context, clientID uint) ([]*db.Organization, error) {
	args := m.Called(ctx, clientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*db.Organization), args.Error(1)
}

func (m *mockOrgRepository) FindOrCreate(ctx context.Context, name string, clientID uint) (*db.Organization, error) {
	args := m.Called(ctx, name, clientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Organization), args.Error(1)
}

type mockRepoRepository struct {
	mock.Mock
}

func (m *mockRepoRepository) Create(ctx context.Context, repo *db.Repo) error {
	return m.Called(ctx, repo).Error(0)
}

func (m *mockRepoRepository) GetByID(ctx context.Context, id uint) (*db.Repo, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Repo), args.Error(1)
}

func (m *mockRepoRepository) GetByFullName(ctx context.Context, orgName, repoName string) (*db.Repo, error) {
	args := m.Called(ctx, orgName, repoName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Repo), args.Error(1)
}

func (m *mockRepoRepository) Update(ctx context.Context, repo *db.Repo) error {
	return m.Called(ctx, repo).Error(0)
}

func (m *mockRepoRepository) Delete(ctx context.Context, id uint, hard bool) error {
	return m.Called(ctx, id, hard).Error(0)
}

func (m *mockRepoRepository) List(ctx context.Context, organizationID uint) ([]*db.Repo, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*db.Repo), args.Error(1)
}

func (m *mockRepoRepository) FindOrCreateFromFullName(ctx context.Context, fullName string, defaultClientID uint) (*db.Repo, error) {
	args := m.Called(ctx, fullName, defaultClientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Repo), args.Error(1)
}

// --- Tests ---

func TestNewPipeline(t *testing.T) {
	mockClient := gh.NewMockClient()
	logger := logrus.New()

	pipeline := NewPipeline(mockClient, nil, nil, nil, logger, nil)
	require.NotNil(t, pipeline)
	assert.NotNil(t, pipeline.ghClient)
	assert.NotNil(t, pipeline.logger)
}

func TestPipelineGetters(t *testing.T) {
	t.Parallel()

	mockClient := gh.NewMockClient()
	logger := logrus.New()
	throttle := NewThrottle(DefaultThrottleConfig(), logger)

	pipeline := NewPipeline(mockClient, nil, nil, nil, logger, throttle)

	t.Run("GetGHClient returns correct client", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, mockClient, pipeline.GetGHClient())
	})

	t.Run("GetLogger returns correct logger", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, logger, pipeline.GetLogger())
	})

	t.Run("GetThrottle returns correct throttle", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, throttle, pipeline.GetThrottle())
	})

	t.Run("GetThrottle returns nil when not set", func(t *testing.T) {
		t.Parallel()
		p := NewPipeline(mockClient, nil, nil, nil, logger, nil)
		assert.Nil(t, p.GetThrottle())
	})
}

func TestSyncRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("successful single repo sync", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		pipeline := NewPipeline(mockClient, nil, nil, nil, nil, nil)

		// Mock GraphQL response for single repo
		graphQLData := map[string]interface{}{
			"repo0": map[string]interface{}{
				"nameWithOwner":  "owner/repo",
				"stargazerCount": float64(42),
				"forkCount":      float64(5),
				"description":    "Test repository",
			},
		}

		mockClient.On("ExecuteGraphQL", ctx, mock.Anything).Return(graphQLData, nil)

		result, err := pipeline.SyncRepository(ctx, "owner", "repo")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "owner/repo", result.FullName)
		assert.Equal(t, 42, result.Stars)
		assert.Equal(t, 5, result.Forks)
		assert.Equal(t, "Test repository", result.Description)

		mockClient.AssertExpectations(t)
	})

	t.Run("GraphQL error", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		pipeline := NewPipeline(mockClient, nil, nil, nil, nil, nil)

		mockClient.On("ExecuteGraphQL", ctx, mock.Anything).
			Return(nil, errGraphQLError)

		result, err := pipeline.SyncRepository(ctx, "owner", "error-repo")
		require.Error(t, err)
		assert.Nil(t, result)

		mockClient.AssertExpectations(t)
	})
}

func TestStartSyncRun(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockAnalyticsRepoPipeline)
		mockRepo.On("CreateSyncRun", ctx, mock.AnythingOfType("*db.SyncRun")).
			Return(nil)

		logger := logrus.New()
		pipeline := NewPipeline(nil, mockRepo, nil, nil, logger, nil)

		run, err := pipeline.StartSyncRun(ctx, "full", "test-org", "")
		require.NoError(t, err)
		require.NotNil(t, run)
		assert.Equal(t, "running", run.Status)
		assert.Equal(t, "full", run.SyncType)
		assert.Equal(t, "test-org", run.OrgFilter)
		assert.False(t, run.StartedAt.IsZero())
		mockRepo.AssertExpectations(t)
	})

	t.Run("create error", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockAnalyticsRepoPipeline)
		mockRepo.On("CreateSyncRun", ctx, mock.AnythingOfType("*db.SyncRun")).
			Return(errCreateSyncRun)

		pipeline := NewPipeline(nil, mockRepo, nil, nil, nil, nil)

		run, err := pipeline.StartSyncRun(ctx, "full", "", "")
		require.Error(t, err)
		assert.Nil(t, run)
		assert.Contains(t, err.Error(), "failed to create sync run")
		mockRepo.AssertExpectations(t)
	})
}

func TestCompleteSyncRun(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success sets fields", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockAnalyticsRepoPipeline)
		mockRepo.On("UpdateSyncRun", ctx, mock.AnythingOfType("*db.SyncRun")).
			Return(nil)

		logger := logrus.New()
		pipeline := NewPipeline(nil, mockRepo, nil, nil, logger, nil)

		run := &db.SyncRun{StartedAt: time.Now().Add(-5 * time.Second)}

		err := pipeline.CompleteSyncRun(ctx, run, "completed")
		require.NoError(t, err)
		assert.Equal(t, "completed", run.Status)
		require.NotNil(t, run.CompletedAt)
		assert.Positive(t, run.DurationMs)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error propagates", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockAnalyticsRepoPipeline)
		mockRepo.On("UpdateSyncRun", ctx, mock.AnythingOfType("*db.SyncRun")).
			Return(errUpdateSyncRun)

		pipeline := NewPipeline(nil, mockRepo, nil, nil, nil, nil)
		run := &db.SyncRun{StartedAt: time.Now()}

		err := pipeline.CompleteSyncRun(ctx, run, "failed")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to complete sync run")
		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateSyncRunCounters(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockAnalyticsRepoPipeline)
		mockRepo.On("UpdateSyncRun", ctx, mock.AnythingOfType("*db.SyncRun")).
			Return(nil)

		pipeline := NewPipeline(nil, mockRepo, nil, nil, nil, nil)
		run := &db.SyncRun{ReposProcessed: 5}

		err := pipeline.UpdateSyncRunCounters(ctx, run)
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error propagates", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockAnalyticsRepoPipeline)
		mockRepo.On("UpdateSyncRun", ctx, mock.AnythingOfType("*db.SyncRun")).
			Return(errUpdateSyncRun)

		pipeline := NewPipeline(nil, mockRepo, nil, nil, nil, nil)
		run := &db.SyncRun{}

		err := pipeline.UpdateSyncRunCounters(ctx, run)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update sync run")
		mockRepo.AssertExpectations(t)
	})
}

func TestRecordSyncRunError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("nil errors field initializes", func(t *testing.T) {
		t.Parallel()

		logger := logrus.New()
		pipeline := NewPipeline(nil, nil, nil, nil, logger, nil)

		run := &db.SyncRun{}
		pipeline.RecordSyncRunError(ctx, run, "org/repo1", errNetworkTimeout)

		assert.Equal(t, 1, run.ReposFailed)
		assert.Equal(t, "org/repo1", run.LastProcessedRepo)
		require.NotNil(t, run.Errors)
		errorsArr, ok := run.Errors["errors"].([]interface{})
		require.True(t, ok)
		assert.Len(t, errorsArr, 1)
	})

	t.Run("existing errors appends", func(t *testing.T) {
		t.Parallel()

		logger := logrus.New()
		pipeline := NewPipeline(nil, nil, nil, nil, logger, nil)

		run := &db.SyncRun{}

		pipeline.RecordSyncRunError(ctx, run, "org/repo1", errNetworkTimeout)
		pipeline.RecordSyncRunError(ctx, run, "org/repo2", errGraphQLError)

		assert.Equal(t, 2, run.ReposFailed)
		assert.Equal(t, "org/repo2", run.LastProcessedRepo)
		errorsArr, ok := run.Errors["errors"].([]interface{})
		require.True(t, ok)
		assert.Len(t, errorsArr, 2)
	})
}

func TestSyncOrganization(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("org not found", func(t *testing.T) {
		t.Parallel()

		mockOrgRepo := new(mockOrgRepository)
		mockOrgRepo.On("GetByName", ctx, "unknown-org").
			Return(nil, errOrgNotFoundDB)

		pipeline := NewPipeline(nil, nil, nil, mockOrgRepo, nil, nil)

		result, err := pipeline.SyncOrganization(ctx, "unknown-org")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "organization not found")
		mockOrgRepo.AssertExpectations(t)
	})

	t.Run("empty repos returns empty map", func(t *testing.T) {
		t.Parallel()

		org := &db.Organization{Name: "test-org"}
		org.ID = 1

		mockOrgRepo := new(mockOrgRepository)
		mockOrgRepo.On("GetByName", ctx, "test-org").Return(org, nil)

		mockRepoRepo := new(mockRepoRepository)
		mockRepoRepo.On("List", ctx, uint(1)).Return([]*db.Repo{}, nil)

		logger := logrus.New()
		pipeline := NewPipeline(nil, nil, mockRepoRepo, mockOrgRepo, logger, nil)

		result, err := pipeline.SyncOrganization(ctx, "test-org")
		require.NoError(t, err)
		assert.Empty(t, result)
		mockOrgRepo.AssertExpectations(t)
		mockRepoRepo.AssertExpectations(t)
	})

	t.Run("repo list error", func(t *testing.T) {
		t.Parallel()

		org := &db.Organization{Name: "test-org"}
		org.ID = 1

		mockOrgRepo := new(mockOrgRepository)
		mockOrgRepo.On("GetByName", ctx, "test-org").Return(org, nil)

		mockRepoRepo := new(mockRepoRepository)
		mockRepoRepo.On("List", ctx, uint(1)).Return(nil, errRepoQueryFailed)

		pipeline := NewPipeline(nil, nil, mockRepoRepo, mockOrgRepo, nil, nil)

		result, err := pipeline.SyncOrganization(ctx, "test-org")
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to query repos")
		mockOrgRepo.AssertExpectations(t)
		mockRepoRepo.AssertExpectations(t)
	})

	t.Run("successful batch with repos", func(t *testing.T) {
		t.Parallel()

		org := &db.Organization{Name: "test-org"}
		org.ID = 1

		repos := []*db.Repo{
			{Name: "repo1"},
			{Name: "repo2"},
		}

		mockOrgRepo := new(mockOrgRepository)
		mockOrgRepo.On("GetByName", ctx, "test-org").Return(org, nil)

		mockRepoRepo := new(mockRepoRepository)
		mockRepoRepo.On("List", ctx, uint(1)).Return(repos, nil)

		mockClient := gh.NewMockClient()
		graphQLData := map[string]interface{}{
			"repo0": map[string]interface{}{
				"nameWithOwner":  "test-org/repo1",
				"stargazerCount": float64(10),
				"forkCount":      float64(2),
			},
			"repo1": map[string]interface{}{
				"nameWithOwner":  "test-org/repo2",
				"stargazerCount": float64(20),
				"forkCount":      float64(4),
			},
		}
		mockClient.On("ExecuteGraphQL", ctx, mock.Anything).Return(graphQLData, nil)

		logger := logrus.New()
		pipeline := NewPipeline(mockClient, nil, mockRepoRepo, mockOrgRepo, logger, nil)

		result, err := pipeline.SyncOrganization(ctx, "test-org")
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "test-org/repo1")
		assert.Contains(t, result, "test-org/repo2")
		mockOrgRepo.AssertExpectations(t)
		mockRepoRepo.AssertExpectations(t)
		mockClient.AssertExpectations(t)
	})
}

func TestCollectMetadata_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	mockClient := gh.NewMockClient()
	logger := logrus.New()
	pipeline := NewPipeline(mockClient, nil, nil, nil, logger, nil)

	repos := []gh.RepoInfo{
		{Name: "repo1", FullName: "org/repo1", Owner: struct {
			Login string `json:"login"`
		}{Login: "org"}},
	}

	result, err := pipeline.collectMetadata(ctx, repos)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Empty(t, result)
}

func TestIsComplexityError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "complexity error",
			err:      errQueryTooComplex,
			expected: true,
		},
		{
			name:     "query cost error",
			err:      errQueryCostExceeded,
			expected: true,
		},
		{
			name:     "complexity limit",
			err:      errComplexityLimitReached,
			expected: true,
		},
		{
			name:     "other error",
			err:      errNetworkTimeout,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isComplexityError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
