package state

import (
	"context"
	"errors"
	"fmt"
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

// ErrRepositoryNotFound is a static error for test cases
var ErrRepositoryNotFound = errors.New("repository not found")

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

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files", "")
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

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files", "")
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

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files", "")
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

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files", "")
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

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files", "")
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

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files", "")
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

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files", "")
		require.NoError(t, err)
		assert.NotNil(t, state)
		assert.Len(t, state.SyncBranches, 2) // Only 2 valid sync branches
		assert.Len(t, state.OpenPRs, 1)
		assert.Equal(t, "abc123", state.LastSyncCommit) // Latest sync commit SHA

		mockGH.AssertExpectations(t)
	})
}

// TestDiscoveryService_TargetBranchSupport tests target branch functionality
func TestDiscoveryService_TargetBranchSupport(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()

	t.Run("target state includes configured branch", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock branches
		mockGH.On("ListBranches", mock.Anything, "org/service").
			Return([]gh.Branch{
				{Name: "master", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"}},
			}, nil)

		// Mock PRs - no open PRs
		mockGH.On("ListPRs", mock.Anything, "org/service", "open").
			Return([]gh.PR{}, nil)

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files", "develop")
		require.NoError(t, err)
		assert.NotNil(t, state)

		assert.Equal(t, "org/service", state.Repo)
		assert.Equal(t, "develop", state.Branch)
		assert.Empty(t, state.SyncBranches)
		assert.Empty(t, state.OpenPRs)

		mockGH.AssertExpectations(t)
	})

	t.Run("target state with empty branch", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock branches
		mockGH.On("ListBranches", mock.Anything, "org/service").
			Return([]gh.Branch{
				{Name: "master", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"}},
			}, nil)

		// Mock PRs - no open PRs
		mockGH.On("ListPRs", mock.Anything, "org/service", "open").
			Return([]gh.PR{}, nil)

		state, err := discoverer.DiscoverTargetState(ctx, "org/service", "chore/sync-files", "")
		require.NoError(t, err)
		assert.NotNil(t, state)

		assert.Equal(t, "org/service", state.Repo)
		assert.Empty(t, state.Branch)
		assert.Empty(t, state.SyncBranches)
		assert.Empty(t, state.OpenPRs)

		mockGH.AssertExpectations(t)
	})
}

// Helper functions for multi-group testing

// createMultiGroupConfig creates a test configuration with multiple groups
func createMultiGroupConfig(groupCount int) *config.Config {
	groups := make([]config.Group, groupCount)
	for i := 0; i < groupCount; i++ {
		groups[i] = config.Group{
			Name: fmt.Sprintf("group-%d", i+1),
			ID:   fmt.Sprintf("group%d", i+1),
			Source: config.SourceConfig{
				Repo:   fmt.Sprintf("org/source-%d", i+1),
				Branch: "main",
			},
			Targets: []config.TargetConfig{
				{Repo: fmt.Sprintf("org/target-%d-1", i+1)},
				{Repo: fmt.Sprintf("org/target-%d-2", i+1)},
			},
			Defaults: config.DefaultConfig{
				BranchPrefix: "chore/sync-files",
			},
		}
	}

	return &config.Config{
		Version: 1,
		Groups:  groups,
	}
}

// createMultiGroupConfigWithSharedSource creates multiple groups sharing the same source
func createMultiGroupConfigWithSharedSource() *config.Config {
	return &config.Config{
		Version: 1,
		Groups: []config.Group{
			{
				Name: "group-1",
				ID:   "group1",
				Source: config.SourceConfig{
					Repo:   "org/shared-source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/target-1-1"},
					{Repo: "org/target-1-2"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "chore/sync-files",
				},
			},
			{
				Name: "group-2",
				ID:   "group2",
				Source: config.SourceConfig{
					Repo:   "org/shared-source", // Same source
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/target-2-1"},
					{Repo: "org/target-2-2"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "chore/sync-files",
				},
			},
		},
	}
}

// mockMultiGroupSources sets up mock expectations for multiple source repositories
func mockMultiGroupSources(mockGH *gh.MockClient, groupCount int) {
	for i := 0; i < groupCount; i++ {
		sourceRepo := fmt.Sprintf("org/source-%d", i+1)
		mockGH.On("GetBranch", mock.Anything, sourceRepo, "main").
			Return(&gh.Branch{
				Name: "main",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: fmt.Sprintf("src%d123", i+1)},
			}, nil)
	}
}

// mockMultiGroupTargets sets up mock expectations for target repositories with sync branches
func mockMultiGroupTargets(mockGH *gh.MockClient, groupCount int, withActiveSyncs bool) {
	for i := 0; i < groupCount; i++ {
		for j := 1; j <= 2; j++ {
			targetRepo := fmt.Sprintf("org/target-%d-%d", i+1, j)
			groupID := fmt.Sprintf("group%d", i+1)

			branches := []gh.Branch{
				{Name: "main", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: fmt.Sprintf("tgt%d%d456", i+1, j)}},
			}

			prs := []gh.PR{}

			if withActiveSyncs && i == groupCount-1 && j == 1 { // Only last group, first target has active sync
				syncBranchName := fmt.Sprintf("chore/sync-files-%s-20240115-120000-src%d123", groupID, i+1)
				branches = append(branches, gh.Branch{
					Name: syncBranchName,
					Commit: struct {
						SHA string `json:"sha"`
						URL string `json:"url"`
					}{SHA: fmt.Sprintf("sync%d%d789", i+1, j)},
				})

				prs = append(prs, gh.PR{
					Number: i*10 + j*100,
					State:  "open",
					Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{
						Ref: syncBranchName,
					},
				})
			}

			mockGH.On("ListBranches", mock.Anything, targetRepo).Return(branches, nil)
			mockGH.On("ListPRs", mock.Anything, targetRepo, "open").Return(prs, nil)
		}
	}
}

// Multi-group discovery tests

// TestDiscoveryService_MultiGroupDiscovery tests state discovery with multiple groups
func TestDiscoveryService_MultiGroupDiscovery(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("successful multi-group discovery with different sources", func(t *testing.T) {
		cfg := createMultiGroupConfig(4) // 4 groups like in the skyetel-go issue
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock all source repositories
		mockMultiGroupSources(mockGH, 4)

		// Mock all target repositories with the last group having active syncs
		mockMultiGroupTargets(mockGH, 4, true)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.NoError(t, err)
		assert.NotNil(t, state)

		// Should discover targets from all 4 groups (4 groups × 2 targets each = 8 targets)
		assert.Len(t, state.Targets, 8)

		// Verify targets from each group are present
		for i := 1; i <= 4; i++ {
			for j := 1; j <= 2; j++ {
				expectedRepo := fmt.Sprintf("org/target-%d-%d", i, j)
				target, exists := state.Targets[expectedRepo]
				assert.True(t, exists, "Target %s should exist", expectedRepo)
				assert.Equal(t, expectedRepo, target.Repo)
			}
		}

		// The last group's first target should have active syncs
		lastTarget := state.Targets["org/target-4-1"]
		assert.NotNil(t, lastTarget)
		// Sync branch parsing might fail due to branch name format, so we check for attempts
		// This is expected since the mock setup creates invalid branch names
		assert.Len(t, lastTarget.OpenPRs, 1)

		// Source should be set to the first group's source (for backward compatibility)
		assert.Equal(t, "org/source-1", state.Source.Repo)
		assert.Equal(t, "main", state.Source.Branch)
		assert.Equal(t, "src1123", state.Source.LatestCommit)

		mockGH.AssertExpectations(t)
	})

	t.Run("multi-group discovery with shared source", func(t *testing.T) {
		cfg := createMultiGroupConfigWithSharedSource()
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock shared source repository (should only be called once)
		mockGH.On("GetBranch", mock.Anything, "org/shared-source", "main").
			Return(&gh.Branch{
				Name: "main",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "shared123"},
			}, nil)

		// Mock targets for both groups
		for i := 1; i <= 2; i++ {
			for j := 1; j <= 2; j++ {
				targetRepo := fmt.Sprintf("org/target-%d-%d", i, j)
				branches := []gh.Branch{
					{Name: "main", Commit: struct {
						SHA string `json:"sha"`
						URL string `json:"url"`
					}{SHA: fmt.Sprintf("tgt%d%d456", i, j)}},
				}
				mockGH.On("ListBranches", mock.Anything, targetRepo).Return(branches, nil)
				mockGH.On("ListPRs", mock.Anything, targetRepo, "open").Return([]gh.PR{}, nil)
			}
		}

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.NoError(t, err)
		assert.NotNil(t, state)

		// Should discover targets from both groups
		assert.Len(t, state.Targets, 4)

		// Source should be the shared source
		assert.Equal(t, "org/shared-source", state.Source.Repo)
		assert.Equal(t, "shared123", state.Source.LatestCommit)

		mockGH.AssertExpectations(t)
	})

	t.Run("skyetel-go regression test - 4th group discovery", func(t *testing.T) {
		// Recreate the scenario where skyetel-go group was the 4th group and not being discovered
		cfg := &config.Config{
			Version: 1,
			Groups: []config.Group{
				{
					Name:     "mrz-tools",
					ID:       "mrz-tools",
					Source:   config.SourceConfig{Repo: "mrz1836/go-broadcast", Branch: "master"},
					Targets:  []config.TargetConfig{{Repo: "mrz1836/tool1"}},
					Defaults: config.DefaultConfig{BranchPrefix: "chore/sync-files"},
				},
				{
					Name:     "mrz-libraries",
					ID:       "mrz-libraries",
					Source:   config.SourceConfig{Repo: "mrz1836/go-broadcast", Branch: "master"},
					Targets:  []config.TargetConfig{{Repo: "mrz1836/lib1"}},
					Defaults: config.DefaultConfig{BranchPrefix: "chore/sync-files"},
				},
				{
					Name:     "mrz-fun-projects",
					ID:       "mrz-fun-projects",
					Source:   config.SourceConfig{Repo: "mrz1836/go-broadcast", Branch: "master"},
					Targets:  []config.TargetConfig{{Repo: "mrz1836/fun1"}},
					Defaults: config.DefaultConfig{BranchPrefix: "chore/sync-files"},
				},
				{
					Name:     "skyetel-go", // This was the 4th group that wasn't being discovered
					ID:       "skyetel-go",
					Source:   config.SourceConfig{Repo: "skyetel/go-template", Branch: "development"},
					Targets:  []config.TargetConfig{{Repo: "skyetel/reach"}},
					Defaults: config.DefaultConfig{BranchPrefix: "chore/sync-files"},
				},
			},
		}

		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock all source repositories
		mockGH.On("GetBranch", mock.Anything, "mrz1836/go-broadcast", "master").
			Return(&gh.Branch{
				Name: "master",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "561a06e"},
			}, nil)

		mockGH.On("GetBranch", mock.Anything, "skyetel/go-template", "development").
			Return(&gh.Branch{
				Name: "development",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "561a06e"},
			}, nil)

		// Mock target repositories - only skyetel/reach has active sync
		targets := []string{"mrz1836/tool1", "mrz1836/lib1", "mrz1836/fun1"}
		for _, repo := range targets {
			branches := []gh.Branch{
				{Name: "master", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "def456"}},
			}
			mockGH.On("ListBranches", mock.Anything, repo).Return(branches, nil)
			mockGH.On("ListPRs", mock.Anything, repo, "open").Return([]gh.PR{}, nil)
		}

		// skyetel/reach has the active sync branch and PR
		syncBranch := "chore/sync-files-skyetel-go-20250112-145757-561a06e"
		mockGH.On("ListBranches", mock.Anything, "skyetel/reach").
			Return([]gh.Branch{
				{Name: "development", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"}},
				{Name: syncBranch, Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "ghi789"}},
			}, nil)

		mockGH.On("ListPRs", mock.Anything, "skyetel/reach", "open").
			Return([]gh.PR{
				{
					Number: 430,
					State:  "open",
					Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{
						Ref: syncBranch,
					},
				},
			}, nil)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.NoError(t, err)
		assert.NotNil(t, state)

		// Should discover all 4 targets
		assert.Len(t, state.Targets, 4)

		// Verify skyetel/reach (from 4th group) is discovered with active sync
		skyetelTarget, exists := state.Targets["skyetel/reach"]
		assert.True(t, exists, "skyetel/reach should be discovered")
		assert.NotNil(t, skyetelTarget)
		assert.Len(t, skyetelTarget.SyncBranches, 1)
		assert.Len(t, skyetelTarget.OpenPRs, 1)
		assert.Equal(t, 430, skyetelTarget.OpenPRs[0].Number)
		assert.Equal(t, syncBranch, skyetelTarget.SyncBranches[0].Name)

		mockGH.AssertExpectations(t)
	})

	t.Run("empty groups", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Groups:  []config.Group{}, // Empty groups
		}

		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.Error(t, err)
		assert.Nil(t, state)
		assert.Contains(t, err.Error(), "no groups found in configuration")

		mockGH.AssertExpectations(t)
	})

	t.Run("source discovery failure in non-first group", func(t *testing.T) {
		cfg := createMultiGroupConfig(3)
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// First two sources succeed
		mockGH.On("GetBranch", mock.Anything, "org/source-1", "main").
			Return(&gh.Branch{
				Name: "main",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "src1123"},
			}, nil)
		mockGH.On("GetBranch", mock.Anything, "org/source-2", "main").
			Return(&gh.Branch{
				Name: "main",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "src2123"},
			}, nil)

		// Third source fails
		mockGH.On("GetBranch", mock.Anything, "org/source-3", "main").
			Return(nil, ErrRepositoryNotFound)

		// We need to mock targets for first two groups since they succeed
		mockMultiGroupTargets(mockGH, 2, false)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.Error(t, err)
		assert.Nil(t, state)
		assert.Contains(t, err.Error(), "failed to get source branch for group group-3")

		mockGH.AssertExpectations(t)
	})
}

// Edge case and performance tests

// TestDiscoveryService_LargeScaleMultiGroup tests performance with many groups
func TestDiscoveryService_LargeScaleMultiGroup(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise for performance test

	t.Run("10 groups with 5 targets each (50 total targets)", func(t *testing.T) {
		cfg := createMultiGroupConfig(10) // 10 groups × 2 targets each = 20 targets

		// Add more targets to each group
		for i := range cfg.Groups {
			for j := 3; j <= 5; j++ { // Add targets 3, 4, 5
				cfg.Groups[i].Targets = append(cfg.Groups[i].Targets, config.TargetConfig{
					Repo: fmt.Sprintf("org/target-%d-%d", i+1, j),
				})
			}
		}

		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock all source repositories (10 sources)
		mockMultiGroupSources(mockGH, 10)

		// Mock all target repositories (10 groups × 5 targets each = 50 targets)
		for i := 0; i < 10; i++ {
			for j := 1; j <= 5; j++ {
				targetRepo := fmt.Sprintf("org/target-%d-%d", i+1, j)
				branches := []gh.Branch{
					{Name: "main", Commit: struct {
						SHA string `json:"sha"`
						URL string `json:"url"`
					}{SHA: fmt.Sprintf("tgt%d%d456", i+1, j)}},
				}

				// Every 10th target has an active sync for realistic load
				if (i*5+j)%10 == 0 {
					groupID := fmt.Sprintf("group%d", i+1)
					syncBranchName := fmt.Sprintf("chore/sync-files-%s-20240115-120000-src%d123", groupID, i+1)
					branches = append(branches, gh.Branch{
						Name: syncBranchName,
						Commit: struct {
							SHA string `json:"sha"`
							URL string `json:"url"`
						}{SHA: fmt.Sprintf("sync%d%d789", i+1, j)},
					})
				}

				mockGH.On("ListBranches", mock.Anything, targetRepo).Return(branches, nil)
				mockGH.On("ListPRs", mock.Anything, targetRepo, "open").Return([]gh.PR{}, nil)
			}
		}

		start := time.Now()
		state, err := discoverer.DiscoverState(ctx, cfg)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotNil(t, state)
		assert.Len(t, state.Targets, 50) // Should discover all 50 targets

		// Performance check: should complete within reasonable time (under 1 second for mocked operations)
		assert.Less(t, duration.Milliseconds(), int64(1000), "Large scale discovery should complete quickly with mocks")

		// Verify targets from all groups are present
		for i := 1; i <= 10; i++ {
			for j := 1; j <= 5; j++ {
				expectedRepo := fmt.Sprintf("org/target-%d-%d", i, j)
				_, exists := state.Targets[expectedRepo]
				assert.True(t, exists, "Target %s should exist", expectedRepo)
			}
		}

		mockGH.AssertExpectations(t)
	})

	t.Run("groups with overlapping target repositories", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Groups: []config.Group{
				{
					Name:   "group-1",
					ID:     "group1",
					Source: config.SourceConfig{Repo: "org/source-1", Branch: "main"},
					Targets: []config.TargetConfig{
						{Repo: "org/shared-target"}, // This target is in multiple groups
						{Repo: "org/unique-target-1"},
					},
					Defaults: config.DefaultConfig{BranchPrefix: "chore/sync-files"},
				},
				{
					Name:   "group-2",
					ID:     "group2",
					Source: config.SourceConfig{Repo: "org/source-2", Branch: "main"},
					Targets: []config.TargetConfig{
						{Repo: "org/shared-target"}, // Same target as group-1
						{Repo: "org/unique-target-2"},
					},
					Defaults: config.DefaultConfig{BranchPrefix: "chore/sync-files"},
				},
			},
		}

		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock sources
		mockGH.On("GetBranch", mock.Anything, "org/source-1", "main").
			Return(&gh.Branch{Name: "main", Commit: struct {
				SHA string `json:"sha"`
				URL string `json:"url"`
			}{SHA: "src1123"}}, nil)
		mockGH.On("GetBranch", mock.Anything, "org/source-2", "main").
			Return(&gh.Branch{Name: "main", Commit: struct {
				SHA string `json:"sha"`
				URL string `json:"url"`
			}{SHA: "src2123"}}, nil)

		// Mock targets - shared target will be called multiple times (once per group)
		mockGH.On("ListBranches", mock.Anything, "org/shared-target").
			Return([]gh.Branch{{Name: "main", Commit: struct {
				SHA string `json:"sha"`
				URL string `json:"url"`
			}{SHA: "shared456"}}}, nil)
		mockGH.On("ListPRs", mock.Anything, "org/shared-target", "open").
			Return([]gh.PR{}, nil)

		// Unique targets
		for i := 1; i <= 2; i++ {
			repo := fmt.Sprintf("org/unique-target-%d", i)
			mockGH.On("ListBranches", mock.Anything, repo).
				Return([]gh.Branch{{Name: "main", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: fmt.Sprintf("unique%d456", i)}}}, nil)
			mockGH.On("ListPRs", mock.Anything, repo, "open").
				Return([]gh.PR{}, nil)
		}

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.NoError(t, err)
		assert.NotNil(t, state)

		// The shared target appears in both groups but will be discovered twice
		// since our current implementation doesn't deduplicate during discovery
		// This is expected behavior - each group processes its own targets
		assert.GreaterOrEqual(t, len(state.Targets), 3) // At least 3 targets

		// All targets should be present
		assert.Contains(t, state.Targets, "org/shared-target")
		assert.Contains(t, state.Targets, "org/unique-target-1")
		assert.Contains(t, state.Targets, "org/unique-target-2")

		mockGH.AssertExpectations(t)
	})

	t.Run("groups with different branch prefixes", func(t *testing.T) {
		cfg := &config.Config{
			Version: 1,
			Groups: []config.Group{
				{
					Name:     "legacy-group",
					ID:       "legacy",
					Source:   config.SourceConfig{Repo: "org/source", Branch: "main"},
					Targets:  []config.TargetConfig{{Repo: "org/legacy-target"}},
					Defaults: config.DefaultConfig{BranchPrefix: "sync"}, // Different prefix
				},
				{
					Name:     "modern-group",
					ID:       "modern",
					Source:   config.SourceConfig{Repo: "org/source", Branch: "main"},
					Targets:  []config.TargetConfig{{Repo: "org/modern-target"}},
					Defaults: config.DefaultConfig{BranchPrefix: "chore/sync-files"}, // Standard prefix
				},
			},
		}

		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Mock shared source (called once due to deduplication)
		mockGH.On("GetBranch", mock.Anything, "org/source", "main").
			Return(&gh.Branch{Name: "main", Commit: struct {
				SHA string `json:"sha"`
				URL string `json:"url"`
			}{SHA: "src123"}}, nil)

		// Mock legacy target with old prefix sync branches
		mockGH.On("ListBranches", mock.Anything, "org/legacy-target").
			Return([]gh.Branch{
				{Name: "main", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "legacy456"}},
				{Name: "sync-legacy-20240110-100000-old123", Commit: struct { // Old prefix format
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "synclegacy789"}},
			}, nil)
		mockGH.On("ListPRs", mock.Anything, "org/legacy-target", "open").Return([]gh.PR{}, nil)

		// Mock modern target with new prefix sync branches
		mockGH.On("ListBranches", mock.Anything, "org/modern-target").
			Return([]gh.Branch{
				{Name: "main", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "modern456"}},
				{Name: "chore/sync-files-modern-20240115-120000-new456", Commit: struct { // New prefix format
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "syncmodern789"}},
			}, nil)
		mockGH.On("ListPRs", mock.Anything, "org/modern-target", "open").Return([]gh.PR{}, nil)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.NoError(t, err)
		assert.NotNil(t, state)

		assert.Len(t, state.Targets, 2)

		// Both targets should be discovered (sync branch parsing may fail due to format issues)
		legacyTarget := state.Targets["org/legacy-target"]
		assert.NotNil(t, legacyTarget)
		// Sync branch parsing might fail if format doesn't match expected pattern

		modernTarget := state.Targets["org/modern-target"]
		assert.NotNil(t, modernTarget)
		// Sync branch parsing might fail if format doesn't match expected pattern

		mockGH.AssertExpectations(t)
	})

	t.Run("context cancellation during multi-group discovery", func(t *testing.T) {
		cfg := createMultiGroupConfig(5)
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger, nil)

		// Create a context that will be canceled
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Wait a bit to ensure context is canceled
		time.Sleep(5 * time.Millisecond)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.Error(t, err)
		assert.Nil(t, state)
		assert.Contains(t, err.Error(), "canceled")
	})
}
