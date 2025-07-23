package state

import (
	"context"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDiscoveryService_DiscoverState(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	cfg := &config.Config{
		Source: config.SourceConfig{
			Repo:   "org/template",
			Branch: "main",
		},
		Targets: []config.TargetConfig{
			{Repo: "org/service-a"},
			{Repo: "org/service-b"},
		},
	}

	t.Run("successful discovery", func(t *testing.T) {
		mockGH := &gh.MockClient{}
		discoverer := NewDiscoverer(mockGH, logger)

		// Mock source branch
		mockGH.On("GetBranch", mock.Anything, "org/template", "main").
			Return(&gh.Branch{
				Name: "main",
				Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"},
			}, nil)

		// Mock branches for service-a
		mockGH.On("ListBranches", mock.Anything, "org/service-a").
			Return([]gh.Branch{
				{Name: "main", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "def456"}},
				{Name: "sync/template-20240115-120000-abc123", Commit: struct {
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
				{Name: "main", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "mno345"}},
				{Name: "sync/template-20240114-100000-def789", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "pqr678"}},
			}, nil)

		// Mock PRs for service-b with an open sync PR
		mockGH.On("ListPRs", mock.Anything, "org/service-b", "open").
			Return([]gh.PR{
				{
					Number: 42,
					Title:  "Sync from template",
					State:  "open",
					Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{
						Ref: "sync/template-20240115-140000-abc123",
						SHA: "stu901",
					},
				},
			}, nil)

		state, err := discoverer.DiscoverState(ctx, cfg)
		require.NoError(t, err)
		assert.NotNil(t, state)

		// Verify source state
		assert.Equal(t, "org/template", state.Source.Repo)
		assert.Equal(t, "main", state.Source.Branch)
		assert.Equal(t, "abc123", state.Source.LatestCommit)

		// Verify target states
		assert.Len(t, state.Targets, 2)

		// Check service-a state
		serviceA := state.Targets["org/service-a"]
		assert.NotNil(t, serviceA)
		assert.Equal(t, "org/service-a", serviceA.Repo)
		assert.Len(t, serviceA.SyncBranches, 1)
		assert.Equal(t, "sync/template-20240115-120000-abc123", serviceA.SyncBranches[0].Name)
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
		discoverer := NewDiscoverer(mockGH, logger)

		mockGH.On("GetBranch", mock.Anything, "org/template", "main").
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
		discoverer := NewDiscoverer(mockGH, logger)

		// Mock branches
		mockGH.On("ListBranches", mock.Anything, "org/service").
			Return([]gh.Branch{
				{Name: "main", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "abc123"}},
				{Name: "sync/template-20240114-100000-abc123", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "def456"}},
				{Name: "sync/template-20240115-120000-def456", Commit: struct {
					SHA string `json:"sha"`
					URL string `json:"url"`
				}{SHA: "ghi789"}},
				{Name: "sync/template-invalid-format", Commit: struct {
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
						Ref: "sync/template-20240115-120000-def456",
					},
				},
			}, nil)

		state, err := discoverer.DiscoverTargetState(ctx, "org/service")
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
		discoverer := NewDiscoverer(mockGH, logger)

		// Mock branches - no sync branches
		mockGH.On("ListBranches", mock.Anything, "org/service").
			Return([]gh.Branch{
				{Name: "main", Commit: struct {
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

		state, err := discoverer.DiscoverTargetState(ctx, "org/service")
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
	discoverer := NewDiscoverer(mockGH, logger)

	t.Run("valid sync branch", func(t *testing.T) {
		metadata, err := discoverer.ParseBranchName("sync/template-20240115-120530-abc123")
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
	discoverer := &discoveryService{gh: mockGH, logger: logger}

	source := SourceState{
		Repo:         "org/template",
		Branch:       "main",
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
