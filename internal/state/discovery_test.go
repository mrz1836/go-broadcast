package state

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestDiscoveryService_DiscoverState(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	cfg := &config.Config{
		Version: 1,
		Groups: []config.Group{
			{
				Name: "test-group",
				ID:   "test",
				Source: config.SourceConfig{
					Repo:   "org/template",
					Branch: "master",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/service-a"},
					{Repo: "org/service-b"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "chore/sync-files",
				},
			},
		},
	}

	t.Run("successful discovery", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock source branch
		mockGH.On("GetBranch", mock.Anything, "org/template", "master").
			Return(&gh.Branch{
				Name: "master",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"},
			}, nil)

		// Mock branches for service-a
		mockGH.On("ListBranches", mock.Anything, "org/service-a").
			Return([]gh.Branch{
				{Name: "master", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "def456"}},
				{Name: "chore/sync-files-default-20240115-120000-abc123", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "ghi789"}},
				{Name: "feature/something", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "jkl012"}},
			}, nil)

		// Mock PRs for service-a
		mockGH.On("ListPRs", mock.Anything, "org/service-a", "open").
			Return([]gh.PR{}, nil)

		// Mock branches for service-b
		mockGH.On("ListBranches", mock.Anything, "org/service-b").
			Return([]gh.Branch{
				{Name: "master", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "mno345"}},
				{Name: "chore/sync-files-default-20240114-100000-def789", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "pqr678"}},
			}, nil)

		// Mock PRs for service-b with an open sync PR
		mockGH.On("ListPRs", mock.Anything, "org/service-b", "open").
			Return([]gh.PR{
				{
					Number: 42,
					Title:  "Sync from source repository",
					State:  "open",
					Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{
						Ref: "chore/sync-files-default-20240115-140000-abc123",
						SHA: "stu901",
					},
				},
			}, nil)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.NoError(t, err)
		assert.NotNil(t, state)

		// Verify source state
		assert.Equal(t, "org/template", state.Source.Repo)
		assert.Equal(t, "master", state.Source.Branch)
		assert.Equal(t, "abc123", state.Source.LatestCommit)

		// Verify target states
		assert.Len(t, state.Targets, 2)

		// Check service-a state
		serviceA := state.Targets["org/service-a"]
		assert.NotNil(t, serviceA)
		assert.Equal(t, "org/service-a", serviceA.Repo)
		assert.Len(t, serviceA.SyncBranches, 1)
		assert.Equal(t, "chore/sync-files-default-20240115-120000-abc123", serviceA.SyncBranches[0].Name)
		assert.Equal(t, StatusUpToDate, serviceA.Status)
		assert.Equal(t, "abc123", serviceA.LastSyncCommit)

		// Check service-b state
		serviceB := state.Targets["org/service-b"]
		assert.NotNil(t, serviceB)
		assert.Equal(t, "org/service-b", serviceB.Repo)
		assert.Len(t, serviceB.SyncBranches, 1)
		assert.Len(t, serviceB.OpenPRs, 1)
		assert.Equal(t, StatusPending, serviceB.Status) // Has open PR

		mockGH.AssertExpectations(t)
	})

	t.Run("error getting source commits", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		mockGH.On("GetBranch", mock.Anything, "org/template", "master").
			Return(nil, assert.AnError)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.Error(t, err)
		assert.Nil(t, state)
		assert.Contains(t, err.Error(), "failed to get source branch")

		mockGH.AssertExpectations(t)
	})
}

func TestDiscoveryService_DiscoverTargetState(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()

	t.Run("repository with sync history", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock branches
		mockGH.On("ListBranches", mock.Anything, "org/service").
			Return([]gh.Branch{
				{Name: "master", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"}},
				{Name: "chore/sync-files-default-20240114-100000-abc123", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "def456"}},
				{Name: "chore/sync-files-default-20240115-120000-def456", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "ghi789"}},
				{Name: "chore/sync-files-invalid-format", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "jkl012"}}, // Invalid format
			}, nil)

		// Mock PRs
		mockGH.On("ListPRs", mock.Anything, "org/service", "open").
			Return([]gh.PR{
				{
					Number: 10,
					State:  "open",
					Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{
						Ref: "chore/sync-files-default-20240115-120000-def456",
					},
				},
			}, nil)

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files")
		require.NoError(t, err)
		assert.NotNil(t, state)

		assert.Equal(t, "org/service", state.Repo)
		assert.Len(t, state.SyncBranches, 2) // Only valid sync branches
		assert.Len(t, state.OpenPRs, 1)
		assert.Equal(t, "def456", state.LastSyncCommit) // Latest sync
		assert.NotNil(t, state.LastSyncTime)

		mockGH.AssertExpectations(t)
	})

	t.Run("repository with no sync history", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock branches - no sync branches
		mockGH.On("ListBranches", mock.Anything, "org/service").
			Return([]gh.Branch{
				{Name: "master", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"}},
				{Name: "feature/something", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "def456"}},
			}, nil)

		// Mock PRs - no open PRs
		mockGH.On("ListPRs", mock.Anything, "org/service", "open").
			Return([]gh.PR{}, nil)

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files")
		require.NoError(t, err)
		assert.NotNil(t, state)

		assert.Equal(t, "org/service", state.Repo)
		assert.Empty(t, state.SyncBranches)
		assert.Empty(t, state.OpenPRs)
		assert.Empty(t, state.LastSyncCommit)
		assert.Nil(t, state.LastSyncTime)
		assert.Equal(t, StatusUnknown, state.Status)

		mockGH.AssertExpectations(t)
	})
}

func TestDiscoveryService_ParseBranchName(t *testing.T) {
	logger := logrus.New()
	mockGH := &gh.MockClient{}
	discoverer := NewDiscoverer(mockGH, logger, nil)

	t.Run("valid sync branch", func(t *testing.T) {
		metadata, err := discoverer.ParseBranchName("chore/sync-files-default-20240115-120530-abc123")
		require.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.Equal(t, "abc123", metadata.CommitSHA)
		assert.Equal(t, time.Date(2024, 1, 15, 12, 5, 30, 0, time.UTC), metadata.Timestamp)
	})

	t.Run("non-sync branch", func(t *testing.T) {
		metadata, err := discoverer.ParseBranchName("feature/new-feature")
		require.Error(t, err)
		assert.Equal(t, ErrNotSyncBranch, err)
		assert.Nil(t, metadata)
	})
}

func TestDetermineSyncStatus(t *testing.T) {
	logger := logrus.New()
	mockGH := &gh.MockClient{}
	discoverer := &discoveryService{gh: mockGH, logger: logger, logConfig: nil}

	source := SourceState{
		Repo:         "org/template",
		Branch:       "master",
		LatestCommit: "abc123",
	}

	tests := []struct {
		name     string
		target   *TargetState
		expected SyncStatus
	}{
		{
			name: "up to date",
			target: &TargetState{
				LastSyncCommit: "abc123",
				OpenPRs:        []gh.PR{},
			},
			expected: StatusUpToDate,
		},
		{
			name: "behind",
			target: &TargetState{
				LastSyncCommit: "old123",
				OpenPRs:        []gh.PR{},
			},
			expected: StatusBehind,
		},
		{
			name: "pending with PR",
			target: &TargetState{
				LastSyncCommit: "old123",
				OpenPRs: []gh.PR{
					{Number: 1, State: "open"},
				},
			},
			expected: StatusPending,
		},
		{
			name: "no sync history",
			target: &TargetState{
				LastSyncCommit: "",
				OpenPRs:        []gh.PR{},
			},
			expected: StatusBehind,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := discoverer.determineSyncStatus(source, tt.target)
			assert.Equal(t, tt.expected, status)
		})
	}
}

// TestDiscoveryService_DiscoverStateWithDebugLogging tests state discovery with debug logging enabled
func TestDiscoveryService_DiscoverStateWithDebugLogging(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a LogConfig with debug state enabled
	logConfig := &logging.LogConfig{
		Debug: logging.DebugFlags{
			State: true,
		},
	}

	cfg := &config.Config{
		Version: 1,
		Groups: []config.Group{
			{
				Name: "test-group",
				ID:   "test",
				Source: config.SourceConfig{
					Repo:   "org/template",
					Branch: "master",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/service-a"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "chore/sync-files",
				},
			},
		},
	}

	t.Run("successful discovery with debug logging", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, logConfig)

		// Mock source branch
		mockGH.On("GetBranch", mock.Anything, "org/template", "master").
			Return(&gh.Branch{
				Name: "master",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"},
			}, nil)

		// Mock branches for service-a
		mockGH.On("ListBranches", mock.Anything, "org/service-a").
			Return([]gh.Branch{
				{Name: "master", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "def456"}},
			}, nil)

		// Mock PRs for service-a
		mockGH.On("ListPRs", mock.Anything, "org/service-a", "open").
			Return([]gh.PR{}, nil)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.NoError(t, err)
		assert.NotNil(t, state)

		mockGH.AssertExpectations(t)
	})

	t.Run("source branch error with debug logging", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, logConfig)

		mockGH.On("GetBranch", mock.Anything, "org/template", "master").
			Return(nil, assert.AnError)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.Error(t, err)
		assert.Nil(t, state)

		mockGH.AssertExpectations(t)
	})

	t.Run("target discovery error with debug logging", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, logConfig)

		// Mock successful source branch
		mockGH.On("GetBranch", mock.Anything, "org/template", "master").
			Return(&gh.Branch{
				Name: "master",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"},
			}, nil)

		// Mock error for target branches
		mockGH.On("ListBranches", mock.Anything, "org/service-a").
			Return(nil, assert.AnError)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.Error(t, err)
		assert.Nil(t, state)

		mockGH.AssertExpectations(t)
	})
}

// TestDiscoveryService_DiscoverStateContextCancellation tests context cancellation handling
func TestDiscoveryService_DiscoverStateContextCancellation(t *testing.T) {
	logger := logrus.New()

	cfg := &config.Config{
		Version: 1,
		Groups: []config.Group{
			{
				Name: "test-group",
				ID:   "test",
				Source: config.SourceConfig{
					Repo:   "org/template",
					Branch: "master",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/service-a"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "chore/sync-files",
				},
			},
		},
	}

	t.Run("context canceled at start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "state discovery canceled")
		assert.Nil(t, state)
	})

	t.Run("context canceled during target discovery", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock source branch
		mockGH.On("GetBranch", mock.Anything, "org/template", "master").
			Return(&gh.Branch{
				Name: "master",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"},
			}, nil).Run(func(_ mock.Arguments) {
			// Cancel context after source is fetched
			cancel()
		})

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target discovery canceled")
		assert.Nil(t, state)

		mockGH.AssertExpectations(t)
	})
}

// TestDiscoveryService_DiscoverTargetStateWithDebugLogging tests target state discovery with debug logging
func TestDiscoveryService_DiscoverTargetStateWithDebugLogging(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a LogConfig with debug state enabled
	logConfig := &logging.LogConfig{
		Debug: logging.DebugFlags{
			State: true,
		},
	}

	t.Run("successful target discovery with debug logging", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, logConfig)

		// Mock branches with sync branches
		mockGH.On("ListBranches", mock.Anything, "org/service").
			Return([]gh.Branch{
				{Name: "master", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"}},
				{Name: "chore/sync-files-default-20240115-120000-def456", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "ghi789"}},
				{Name: "chore/sync-files-invalid", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "jkl012"}}, // Invalid format - will trigger parse error logging
			}, nil)

		// Mock PRs
		mockGH.On("ListPRs", mock.Anything, "org/service", "open").
			Return([]gh.PR{
				{
					Number: 10,
					State:  "open",
					Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{
						Ref: "chore/sync-files-default-20240115-120000-def456",
					},
				},
			}, nil)

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files")
		require.NoError(t, err)
		assert.NotNil(t, state)
		assert.Len(t, state.SyncBranches, 1) // Only valid sync branch

		mockGH.AssertExpectations(t)
	})

	t.Run("branch listing error with debug logging", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, logConfig)

		// Mock error when listing branches
		mockGH.On("ListBranches", mock.Anything, "org/service").
			Return(nil, assert.AnError)

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list branches")
		assert.Nil(t, state)

		mockGH.AssertExpectations(t)
	})

	t.Run("PR listing error with debug logging", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, logConfig)

		// Mock successful branch listing
		mockGH.On("ListBranches", mock.Anything, "org/service").
			Return([]gh.Branch{
				{Name: "master", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"}},
			}, nil)

		// Mock error when listing PRs
		mockGH.On("ListPRs", mock.Anything, "org/service", "open").
			Return(nil, assert.AnError)

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list PRs")
		assert.Nil(t, state)

		mockGH.AssertExpectations(t)
	})
}

// TestDiscoveryService_DiscoverTargetStateContextCancellation tests context cancellation in target discovery
func TestDiscoveryService_DiscoverTargetStateContextCancellation(t *testing.T) {
	logger := logrus.New()

	t.Run("context canceled at start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target discovery canceled")
		assert.Nil(t, state)
	})
}

// TestDiscoveryService_ComplexSyncBranchScenarios tests complex sync branch discovery scenarios
func TestDiscoveryService_ComplexSyncBranchScenarios(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	logConfig := &logging.LogConfig{
		Debug: logging.DebugFlags{
			State: true,
		},
	}

	t.Run("multiple sync branches", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, logConfig)

		// Mock branches with multiple sync branches - note that chore/sync-files-invalid will be filtered out
		mockGH.On("ListBranches", mock.Anything, "org/service").
			Return([]gh.Branch{
				{Name: "chore/sync-files-default-20240114-100000-abc123", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "def456"}},
				{Name: "chore/sync-files-default-20240115-110000-abc123", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "ghi789"}},
				{Name: "chore/sync-files-invalid", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "invalid"}}, // This will be filtered out due to invalid format
			}, nil)

		// Mock multiple PRs from different sync branches
		mockGH.On("ListPRs", mock.Anything, "org/service", "open").
			Return([]gh.PR{
				{
					Number: 10,
					State:  "open",
					Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{
						Ref: "chore/sync-files-default-20240115-110000-abc123",
					},
				},
			}, nil)

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files")
		require.NoError(t, err)
		assert.NotNil(t, state)
		assert.Len(t, state.SyncBranches, 2) // Only 2 valid sync branches
		assert.Len(t, state.OpenPRs, 1)
		assert.Equal(t, "abc123", state.LastSyncCommit) // Latest sync commit SHA

		mockGH.AssertExpectations(t)
	})
}
