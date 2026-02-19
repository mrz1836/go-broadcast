package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/mrz1836/go-broadcast/internal/db"
)

// Static errors for err113 linter
var (
	errSnapshotFailed  = errors.New("snapshot query failed")
	errAlertFailed     = errors.New("alert query failed")
	errListFailed      = errors.New("list repositories failed")
	errRepoLookup      = errors.New("repository lookup failed")
	errSyncRunFailed   = errors.New("sync run query failed")
	errAlertByType     = errors.New("alert by type query failed")
	errAlertBySeverity = errors.New("alert by severity query failed")
)

// mockAnalyticsRepo implements db.AnalyticsRepo for testing
type mockAnalyticsRepo struct {
	mock.Mock
}

func (m *mockAnalyticsRepo) UpsertOrganization(ctx context.Context, org *db.Organization) error {
	args := m.Called(ctx, org)
	return args.Error(0)
}

func (m *mockAnalyticsRepo) GetOrganization(ctx context.Context, login string) (*db.Organization, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.Organization), args.Error(1)
}

func (m *mockAnalyticsRepo) ListOrganizations(ctx context.Context) ([]db.Organization, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.Organization), args.Error(1)
}

func (m *mockAnalyticsRepo) UpsertRepository(ctx context.Context, repo *db.AnalyticsRepository) error {
	args := m.Called(ctx, repo)
	return args.Error(0)
}

func (m *mockAnalyticsRepo) GetRepository(ctx context.Context, fullName string) (*db.AnalyticsRepository, error) {
	args := m.Called(ctx, fullName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.AnalyticsRepository), args.Error(1)
}

func (m *mockAnalyticsRepo) ListRepositories(ctx context.Context, orgLogin string) ([]db.AnalyticsRepository, error) {
	args := m.Called(ctx, orgLogin)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.AnalyticsRepository), args.Error(1)
}

func (m *mockAnalyticsRepo) CreateSnapshot(ctx context.Context, snap *db.RepositorySnapshot) error {
	args := m.Called(ctx, snap)
	return args.Error(0)
}

func (m *mockAnalyticsRepo) GetLatestSnapshot(ctx context.Context, repoID uint) (*db.RepositorySnapshot, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.RepositorySnapshot), args.Error(1)
}

func (m *mockAnalyticsRepo) GetSnapshotHistory(ctx context.Context, repoID uint, since time.Time) ([]db.RepositorySnapshot, error) {
	args := m.Called(ctx, repoID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.RepositorySnapshot), args.Error(1)
}

func (m *mockAnalyticsRepo) UpsertAlert(ctx context.Context, alert *db.SecurityAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *mockAnalyticsRepo) GetOpenAlerts(ctx context.Context, repoID uint, severity string) ([]db.SecurityAlert, error) {
	args := m.Called(ctx, repoID, severity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]db.SecurityAlert), args.Error(1)
}

func (m *mockAnalyticsRepo) GetAlertCounts(ctx context.Context, repoID uint) (map[string]int, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *mockAnalyticsRepo) GetAlertCountsByType(ctx context.Context, repoID uint) (map[string]int, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *mockAnalyticsRepo) CreateCISnapshot(ctx context.Context, snap *db.CIMetricsSnapshot) error {
	args := m.Called(ctx, snap)
	return args.Error(0)
}

func (m *mockAnalyticsRepo) GetLatestCISnapshot(ctx context.Context, repoID uint) (*db.CIMetricsSnapshot, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.CIMetricsSnapshot), args.Error(1)
}

func (m *mockAnalyticsRepo) CreateSyncRun(ctx context.Context, run *db.SyncRun) error {
	args := m.Called(ctx, run)
	return args.Error(0)
}

func (m *mockAnalyticsRepo) UpdateSyncRun(ctx context.Context, run *db.SyncRun) error {
	args := m.Called(ctx, run)
	return args.Error(0)
}

func (m *mockAnalyticsRepo) GetLatestSyncRun(ctx context.Context) (*db.SyncRun, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.SyncRun), args.Error(1)
}

func TestFormatTimeAgo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *time.Time
		expected string
	}{
		{
			name:     "nil returns unknown",
			input:    nil,
			expected: "unknown",
		},
		{
			name:     "just now (seconds ago)",
			input:    timePtr(time.Now().Add(-10 * time.Second)),
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			input:    timePtr(time.Now().Add(-1 * time.Minute)),
			expected: "1 minute ago",
		},
		{
			name:     "5 minutes ago",
			input:    timePtr(time.Now().Add(-5 * time.Minute)),
			expected: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			input:    timePtr(time.Now().Add(-1 * time.Hour)),
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			input:    timePtr(time.Now().Add(-3 * time.Hour)),
			expected: "3 hours ago",
		},
		{
			name:     "1 day ago",
			input:    timePtr(time.Now().Add(-24 * time.Hour)),
			expected: "1 day ago",
		},
		{
			name:     "5 days ago",
			input:    timePtr(time.Now().Add(-5 * 24 * time.Hour)),
			expected: "5 days ago",
		},
		{
			name:     "1 week ago",
			input:    timePtr(time.Now().Add(-7 * 24 * time.Hour)),
			expected: "1 week ago",
		},
		{
			name:     "3 weeks ago",
			input:    timePtr(time.Now().Add(-21 * 24 * time.Hour)),
			expected: "3 weeks ago",
		},
		{
			name:     "1 month ago",
			input:    timePtr(time.Now().Add(-35 * 24 * time.Hour)),
			expected: "1 month ago",
		},
		{
			name:     "3 months ago",
			input:    timePtr(time.Now().Add(-90 * 24 * time.Hour)),
			expected: "3 months ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatTimeAgo(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ms       int64
		expected string
	}{
		{
			name:     "zero",
			ms:       0,
			expected: "0s",
		},
		{
			name:     "milliseconds",
			ms:       500,
			expected: "500ms",
		},
		{
			name:     "one millisecond",
			ms:       1,
			expected: "1ms",
		},
		{
			name:     "seconds",
			ms:       5000,
			expected: "5.0s",
		},
		{
			name:     "fractional seconds",
			ms:       1500,
			expected: "1.5s",
		},
		{
			name:     "minutes",
			ms:       120000,
			expected: "2.0m",
		},
		{
			name:     "fractional minutes",
			ms:       90000,
			expected: "1.5m",
		},
		{
			name:     "hours",
			ms:       3600000,
			expected: "1.0h",
		},
		{
			name:     "fractional hours",
			ms:       5400000,
			expected: "1.5h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatDuration(tt.ms)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string no-op",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "longer string gets truncated",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "maxLen 3 no ellipsis",
			input:    "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "maxLen 2 no ellipsis",
			input:    "hello",
			maxLen:   2,
			expected: "he",
		},
		{
			name:     "maxLen 1 no ellipsis",
			input:    "hello",
			maxLen:   1,
			expected: "h",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "maxLen 4 with truncation",
			input:    "hello",
			maxLen:   4,
			expected: "h...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDisplayAllRepositories(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("empty repositories", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("ListRepositories", ctx, "").
			Return([]db.AnalyticsRepository{}, nil)

		err := displayAllRepositories(ctx, mockRepo)
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("list error", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("ListRepositories", ctx, "").
			Return(nil, errListFailed)

		err := displayAllRepositories(ctx, mockRepo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list repositories")
		mockRepo.AssertExpectations(t)
	})

	t.Run("repos with snapshots and alerts", func(t *testing.T) {
		t.Parallel()

		syncTime := time.Now().Add(-1 * time.Hour)
		completedAt := time.Now().Add(-30 * time.Minute)
		repos := []db.AnalyticsRepository{
			{FullName: "org/repo1", LastSyncAt: &syncTime},
			{FullName: "org/repo2"},
		}
		repos[0].ID = 1
		repos[1].ID = 2

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("ListRepositories", ctx, "").Return(repos, nil)

		// Repo 1: has snapshot and alerts
		mockRepo.On("GetLatestSnapshot", ctx, uint(1)).
			Return(&db.RepositorySnapshot{Stars: 42, Forks: 5, OpenIssues: 3, OpenPRs: 1}, nil)
		mockRepo.On("GetAlertCounts", ctx, uint(1)).
			Return(map[string]int{"critical": 2, "high": 1}, nil)

		// Repo 2: no snapshot, no alerts
		mockRepo.On("GetLatestSnapshot", ctx, uint(2)).
			Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("GetAlertCounts", ctx, uint(2)).
			Return(map[string]int{}, nil)

		// Latest sync run
		mockRepo.On("GetLatestSyncRun", ctx).
			Return(&db.SyncRun{CompletedAt: &completedAt, DurationMs: 5000}, nil)

		err := displayAllRepositories(ctx, mockRepo)
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("snapshot error propagates", func(t *testing.T) {
		t.Parallel()

		repos := []db.AnalyticsRepository{{FullName: "org/repo1"}}
		repos[0].ID = 1

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("ListRepositories", ctx, "").Return(repos, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(1)).
			Return(nil, errSnapshotFailed)

		err := displayAllRepositories(ctx, mockRepo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get snapshot")
		mockRepo.AssertExpectations(t)
	})

	t.Run("alert count error propagates", func(t *testing.T) {
		t.Parallel()

		repos := []db.AnalyticsRepository{{FullName: "org/repo1"}}
		repos[0].ID = 1

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("ListRepositories", ctx, "").Return(repos, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(1)).
			Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("GetAlertCounts", ctx, uint(1)).
			Return(nil, errAlertFailed)

		err := displayAllRepositories(ctx, mockRepo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get alert counts")
		mockRepo.AssertExpectations(t)
	})

	t.Run("sync run not found is not error", func(t *testing.T) {
		t.Parallel()

		repos := []db.AnalyticsRepository{{FullName: "org/repo1"}}
		repos[0].ID = 1

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("ListRepositories", ctx, "").Return(repos, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(1)).
			Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("GetAlertCounts", ctx, uint(1)).
			Return(map[string]int{}, nil)
		mockRepo.On("GetLatestSyncRun", ctx).
			Return(nil, gorm.ErrRecordNotFound)

		err := displayAllRepositories(ctx, mockRepo)
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("sync run error propagates", func(t *testing.T) {
		t.Parallel()

		repos := []db.AnalyticsRepository{{FullName: "org/repo1"}}
		repos[0].ID = 1

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("ListRepositories", ctx, "").Return(repos, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(1)).
			Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("GetAlertCounts", ctx, uint(1)).
			Return(map[string]int{}, nil)
		mockRepo.On("GetLatestSyncRun", ctx).
			Return(nil, errSyncRunFailed)

		err := displayAllRepositories(ctx, mockRepo)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get latest sync run")
		mockRepo.AssertExpectations(t)
	})
}

func TestDisplaySingleRepository(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("not found returns helpful error", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetRepository", ctx, "org/missing").
			Return(nil, gorm.ErrRecordNotFound)

		err := displaySingleRepository(ctx, mockRepo, "org/missing")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found in database")
		mockRepo.AssertExpectations(t)
	})

	t.Run("get repository error", func(t *testing.T) {
		t.Parallel()

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetRepository", ctx, "org/repo").
			Return(nil, errRepoLookup)

		err := displaySingleRepository(ctx, mockRepo, "org/repo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get repository")
		mockRepo.AssertExpectations(t)
	})

	t.Run("full repo with snapshot and alerts", func(t *testing.T) {
		t.Parallel()

		syncTime := time.Now().Add(-2 * time.Hour)
		repo := &db.AnalyticsRepository{
			FullName:      "org/repo",
			Description:   "A test repository",
			Language:      "Go",
			DefaultBranch: "main",
			IsPrivate:     false,
			IsArchived:    false,
			LastSyncAt:    &syncTime,
		}
		repo.ID = 1

		snapshot := &db.RepositorySnapshot{
			Stars:       100,
			Forks:       20,
			Watchers:    50,
			OpenIssues:  5,
			OpenPRs:     2,
			BranchCount: 10,
			SnapshotAt:  time.Now().Add(-1 * time.Hour),
		}

		alertsByType := map[string]int{
			"dependabot":      3,
			"code_scanning":   1,
			"secret_scanning": 0,
		}
		alertsBySeverity := map[string]int{
			"critical": 1,
			"high":     2,
			"medium":   1,
		}

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetRepository", ctx, "org/repo").Return(repo, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(1)).Return(snapshot, nil)
		mockRepo.On("GetAlertCountsByType", ctx, uint(1)).Return(alertsByType, nil)
		mockRepo.On("GetAlertCounts", ctx, uint(1)).Return(alertsBySeverity, nil)

		err := displaySingleRepository(ctx, mockRepo, "org/repo")
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("repo without snapshot", func(t *testing.T) {
		t.Parallel()

		repo := &db.AnalyticsRepository{
			FullName: "org/new-repo",
		}
		repo.ID = 2

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetRepository", ctx, "org/new-repo").Return(repo, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(2)).
			Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("GetAlertCountsByType", ctx, uint(2)).
			Return(map[string]int{}, nil)
		mockRepo.On("GetAlertCounts", ctx, uint(2)).
			Return(map[string]int{}, nil)

		err := displaySingleRepository(ctx, mockRepo, "org/new-repo")
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("private archived repo", func(t *testing.T) {
		t.Parallel()

		repo := &db.AnalyticsRepository{
			FullName:   "org/private-repo",
			IsPrivate:  true,
			IsArchived: true,
		}
		repo.ID = 3

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetRepository", ctx, "org/private-repo").Return(repo, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(3)).
			Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("GetAlertCountsByType", ctx, uint(3)).
			Return(map[string]int{}, nil)
		mockRepo.On("GetAlertCounts", ctx, uint(3)).
			Return(map[string]int{}, nil)

		err := displaySingleRepository(ctx, mockRepo, "org/private-repo")
		require.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("snapshot error propagates", func(t *testing.T) {
		t.Parallel()

		repo := &db.AnalyticsRepository{FullName: "org/repo"}
		repo.ID = 4

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetRepository", ctx, "org/repo").Return(repo, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(4)).
			Return(nil, errSnapshotFailed)

		err := displaySingleRepository(ctx, mockRepo, "org/repo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get snapshot")
		mockRepo.AssertExpectations(t)
	})

	t.Run("alert counts by type error propagates", func(t *testing.T) {
		t.Parallel()

		repo := &db.AnalyticsRepository{FullName: "org/repo"}
		repo.ID = 5

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetRepository", ctx, "org/repo").Return(repo, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(5)).
			Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("GetAlertCountsByType", ctx, uint(5)).
			Return(nil, errAlertByType)

		err := displaySingleRepository(ctx, mockRepo, "org/repo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get alert counts by type")
		mockRepo.AssertExpectations(t)
	})

	t.Run("alert counts by severity error propagates", func(t *testing.T) {
		t.Parallel()

		repo := &db.AnalyticsRepository{FullName: "org/repo"}
		repo.ID = 6

		mockRepo := new(mockAnalyticsRepo)
		mockRepo.On("GetRepository", ctx, "org/repo").Return(repo, nil)
		mockRepo.On("GetLatestSnapshot", ctx, uint(6)).
			Return(nil, gorm.ErrRecordNotFound)
		mockRepo.On("GetAlertCountsByType", ctx, uint(6)).
			Return(map[string]int{}, nil)
		mockRepo.On("GetAlertCounts", ctx, uint(6)).
			Return(nil, errAlertBySeverity)

		err := displaySingleRepository(ctx, mockRepo, "org/repo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get alert counts by severity")
		mockRepo.AssertExpectations(t)
	})
}

// timePtr creates a *time.Time from a time.Time value
func timePtr(t time.Time) *time.Time {
	return &t
}
