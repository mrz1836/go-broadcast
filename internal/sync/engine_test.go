package sync

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	errGitCloneFailed = stderrors.New("git clone failed")
	errCloneFailed    = stderrors.New("clone failed")
)

func TestNewEngine(t *testing.T) {
	cfg := &config.Config{}
	ghClient := &gh.MockClient{}
	gitClient := &git.MockClient{}
	stateDiscoverer := &state.MockDiscoverer{}
	transformChain := &transform.MockChain{}

	t.Run("with options", func(t *testing.T) {
		opts := &Options{DryRun: true}
		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)

		assert.NotNil(t, engine)
		assert.Equal(t, cfg, engine.config)
		assert.Equal(t, ghClient, engine.gh)
		assert.Equal(t, gitClient, engine.git)
		assert.Equal(t, stateDiscoverer, engine.state)
		assert.Equal(t, transformChain, engine.transform)
		assert.Equal(t, opts, engine.options)
	})

	t.Run("with nil options", func(t *testing.T) {
		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, nil)

		assert.NotNil(t, engine)
		assert.NotNil(t, engine.options)
		assert.Equal(t, DefaultOptions().DryRun, engine.options.DryRun)
	})
}

func TestEngineSync(t *testing.T) {
	// Setup test configuration
	cfg := &config.Config{
		Groups: []config.Group{{
			Source: config.SourceConfig{
				Repo:   "org/template",
				Branch: "master",
			},
			Targets: []config.TargetConfig{
				{
					Repo: "org/target-a",
					Files: []config.FileMapping{
						{Src: "file1.txt", Dest: "file1.txt"},
					},
				},
				{
					Repo: "org/target-b",
					Files: []config.FileMapping{
						{Src: "file2.txt", Dest: "file2.txt"},
					},
				},
			},
		}},
	}

	t.Run("successful sync with up-to-date targets", func(t *testing.T) {
		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		// Mock state discovery - all targets up-to-date
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template",
				Branch:       "master",
				LatestCommit: "abc123",
				LastChecked:  time.Now(),
			},
			Targets: map[string]*state.TargetState{
				"org/target-a": {
					Repo:           "org/target-a",
					LastSyncCommit: "abc123", // Same as source
					Status:         state.StatusUpToDate,
				},
				"org/target-b": {
					Repo:           "org/target-b",
					LastSyncCommit: "abc123", // Same as source
					Status:         state.StatusUpToDate,
				},
			},
		}

		stateDiscoverer.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		// Create engine
		opts := &Options{
			DryRun:         false,
			MaxConcurrency: 2,
		}
		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
		engine.SetLogger(logrus.New())

		// Execute sync
		err := engine.Sync(context.Background(), nil)

		// Assertions - should succeed without doing any sync work
		require.NoError(t, err)
		stateDiscoverer.AssertExpectations(t)
	})

	t.Run("state discovery failure", func(t *testing.T) {
		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		// Mock state discovery failure
		stateDiscoverer.On("DiscoverState", mock.Anything, cfg).
			Return(nil, errors.ErrTest)

		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, nil)

		// Execute sync
		err := engine.Sync(context.Background(), nil)

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to discover current state")
		stateDiscoverer.AssertExpectations(t)
	})

	t.Run("no targets to sync", func(t *testing.T) {
		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		// Mock state with up-to-date targets
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template",
				Branch:       "master",
				LatestCommit: "abc123",
			},
			Targets: map[string]*state.TargetState{
				"org/target-a": {
					Repo:           "org/target-a",
					LastSyncCommit: "abc123", // Same as source
					Status:         state.StatusUpToDate,
				},
				"org/target-b": {
					Repo:           "org/target-b",
					LastSyncCommit: "abc123", // Same as source
					Status:         state.StatusUpToDate,
				},
			},
		}

		stateDiscoverer.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, nil)

		// Execute sync
		err := engine.Sync(context.Background(), nil)

		// Assertions
		require.NoError(t, err)
		stateDiscoverer.AssertExpectations(t)
	})

	t.Run("target filtering", func(t *testing.T) {
		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template",
				LatestCommit: "abc123",
			},
			Targets: map[string]*state.TargetState{
				"org/target-a": {
					Repo:           "org/target-a",
					LastSyncCommit: "abc123", // Same as source - up to date
					Status:         state.StatusUpToDate,
				},
			},
		}

		stateDiscoverer.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, nil)

		// Execute sync with target filter - should skip since up-to-date
		err := engine.Sync(context.Background(), []string{"org/target-a"})

		// Should not error
		require.NoError(t, err)
		stateDiscoverer.AssertExpectations(t)
	})

	t.Run("invalid target filter", func(t *testing.T) {
		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template",
				LatestCommit: "abc123",
			},
			Targets: map[string]*state.TargetState{},
		}

		stateDiscoverer.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, nil)

		// Execute sync with invalid target filter
		err := engine.Sync(context.Background(), []string{"org/nonexistent"})

		// Should error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no targets match")
		stateDiscoverer.AssertExpectations(t)
	})
}

func TestEngineFilterTargets(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.Group{{
			Targets: []config.TargetConfig{
				{Repo: "org/target-a"},
				{Repo: "org/target-b"},
				{Repo: "org/target-c"},
			},
		}},
	}

	currentState := &state.State{
		Targets: map[string]*state.TargetState{
			"org/target-a": {Status: state.StatusBehind},
			"org/target-b": {Status: state.StatusUpToDate},
			"org/target-c": {Status: state.StatusPending},
		},
	}

	engine := &Engine{
		config:  cfg,
		options: DefaultOptions(),
		logger:  logrus.New(),
	}

	t.Run("no filter", func(t *testing.T) {
		targets, err := engine.filterTargets(nil, currentState)

		require.NoError(t, err)
		// Should return targets that need sync (Behind and Pending since UpdateExistingPRs defaults to true)
		assert.Len(t, targets, 2)
		assert.Equal(t, "org/target-a", targets[0].Repo)
		assert.Equal(t, "org/target-c", targets[1].Repo)
	})

	t.Run("with filter", func(t *testing.T) {
		targets, err := engine.filterTargets([]string{"org/target-b"}, currentState)

		require.NoError(t, err)
		// Should return empty since target-b is up-to-date
		assert.Empty(t, targets)
	})

	t.Run("with force option", func(t *testing.T) {
		engine.options.Force = true
		targets, err := engine.filterTargets([]string{"org/target-b"}, currentState)

		require.NoError(t, err)
		// Should return target-b even though it's up-to-date (forced)
		assert.Len(t, targets, 1)
		assert.Equal(t, "org/target-b", targets[0].Repo)
	})

	t.Run("invalid filter", func(t *testing.T) {
		engine.options.Force = false
		_, err := engine.filterTargets([]string{"org/nonexistent"}, currentState)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no targets match")
	})
}

func TestEngineNeedsSync(t *testing.T) {
	engine := &Engine{
		options: DefaultOptions(),
		logger:  logrus.New(),
	}

	target := config.TargetConfig{Repo: "org/target"}
	currentState := &state.State{
		Source: state.SourceState{LatestCommit: "abc123"},
	}

	t.Run("no target state", func(t *testing.T) {
		needs := engine.needsSync(target, currentState)
		assert.True(t, needs)
	})

	t.Run("up to date", func(t *testing.T) {
		currentState.Targets = map[string]*state.TargetState{
			"org/target": {Status: state.StatusUpToDate},
		}

		needs := engine.needsSync(target, currentState)
		assert.False(t, needs)
	})

	t.Run("behind", func(t *testing.T) {
		currentState.Targets = map[string]*state.TargetState{
			"org/target": {Status: state.StatusBehind},
		}

		needs := engine.needsSync(target, currentState)
		assert.True(t, needs)
	})

	t.Run("pending with update PRs enabled", func(t *testing.T) {
		engine.options.UpdateExistingPRs = true
		currentState.Targets = map[string]*state.TargetState{
			"org/target": {Status: state.StatusPending},
		}

		needs := engine.needsSync(target, currentState)
		assert.True(t, needs)
	})

	t.Run("pending with update PRs disabled", func(t *testing.T) {
		engine.options.UpdateExistingPRs = false
		currentState.Targets = map[string]*state.TargetState{
			"org/target": {Status: state.StatusPending},
		}

		needs := engine.needsSync(target, currentState)
		assert.False(t, needs)
	})

	t.Run("conflict", func(t *testing.T) {
		currentState.Targets = map[string]*state.TargetState{
			"org/target": {Status: state.StatusConflict},
		}

		needs := engine.needsSync(target, currentState)
		assert.False(t, needs)
	})

	t.Run("unknown status", func(t *testing.T) {
		currentState.Targets = map[string]*state.TargetState{
			"org/target": {Status: state.StatusUnknown},
		}

		needs := engine.needsSync(target, currentState)
		assert.True(t, needs)
	})
}

func TestEngineWithDryRun(t *testing.T) {
	cfg := &config.Config{
		Groups: []config.Group{{
			Source: config.SourceConfig{
				Repo:   "org/template",
				Branch: "master",
			},
			Targets: []config.TargetConfig{
				{
					Repo: "org/target",
					Files: []config.FileMapping{
						{Src: "file.txt", Dest: "file.txt"},
					},
				},
			},
		}},
	}

	// Setup mocks
	ghClient := &gh.MockClient{}
	gitClient := &git.MockClient{}
	stateDiscoverer := &state.MockDiscoverer{}
	transformChain := &transform.MockChain{}

	currentState := &state.State{
		Source: state.SourceState{
			Repo:         "org/template",
			Branch:       "master",
			LatestCommit: "abc123",
		},
		Targets: map[string]*state.TargetState{
			"org/target": {
				Repo:           "org/target",
				LastSyncCommit: "abc123", // Same as source - up to date
				Status:         state.StatusUpToDate,
			},
		},
	}

	stateDiscoverer.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

	// Mock file operations
	ghClient.On("GetFile", mock.Anything, "org/target", mock.Anything, "").
		Return(&gh.FileContent{Content: []byte("old content")}, nil).Maybe()

	// Mock transformations
	transformChain.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("new content"), nil).Maybe()

	// Create engine with dry-run enabled
	opts := &Options{DryRun: true}
	engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
	engine.SetLogger(logrus.New())

	// Execute sync
	err := engine.Sync(context.Background(), nil)

	// Should not error in dry-run mode
	require.NoError(t, err)
	stateDiscoverer.AssertExpectations(t)

	// Verify no actual operations were called (they would be mocked if called)
	ghClient.AssertNotCalled(t, "CreatePR")
	gitClient.AssertNotCalled(t, "Clone")
}

// TestEngineConcurrentErrorScenarios tests error handling in concurrent sync operations
func TestEngineConcurrentErrorScenarios(t *testing.T) {
	// Base configuration with multiple targets for concurrent testing
	cfg := &config.Config{
		Groups: []config.Group{{
			Source: config.SourceConfig{
				Repo:   "org/template",
				Branch: "master",
			},
			Targets: []config.TargetConfig{
				{
					Repo: "org/target-a",
					Files: []config.FileMapping{
						{Src: "file1.txt", Dest: "file1.txt"},
					},
				},
				{
					Repo: "org/target-b",
					Files: []config.FileMapping{
						{Src: "file2.txt", Dest: "file2.txt"},
					},
				},
				{
					Repo: "org/target-c",
					Files: []config.FileMapping{
						{Src: "file3.txt", Dest: "file3.txt"},
					},
				},
			},
		}},
	}

	t.Run("multiple concurrent failures in errgroup", func(t *testing.T) {
		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		// Mock state discovery - all targets need sync
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template",
				Branch:       "master",
				LatestCommit: "new123",
				LastChecked:  time.Now(),
			},
			Targets: map[string]*state.TargetState{
				"org/target-a": {
					Repo:           "org/target-a",
					LastSyncCommit: "old123", // Behind source
					Status:         state.StatusBehind,
				},
				"org/target-b": {
					Repo:           "org/target-b",
					LastSyncCommit: "old123", // Behind source
					Status:         state.StatusBehind,
				},
				"org/target-c": {
					Repo:           "org/target-c",
					LastSyncCommit: "old123", // Behind source
					Status:         state.StatusBehind,
				},
			},
		}

		stateDiscoverer.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		// Mock all sync operations to fail with different errors
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.Anything).
			Return(errGitCloneFailed).Maybe()

		// Create engine with low concurrency to ensure predictable error handling
		opts := &Options{
			DryRun:         false,
			MaxConcurrency: 2, // Less than number of targets to test queuing
		}
		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
		engine.SetLogger(logrus.New())

		// Execute sync - should fail fast on first error due to errgroup behavior
		err := engine.Sync(context.Background(), nil)

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to complete sync operation")
		// Should contain one of the git clone failure messages
		errorMsg := err.Error()
		assert.True(t,
			assert.Contains(t, errorMsg, "git clone failed") ||
				assert.Contains(t, errorMsg, "clone failed"),
			"Error should contain git clone failure message")

		stateDiscoverer.AssertExpectations(t)
	})

	t.Run("partial concurrent failures with success mixed in", func(t *testing.T) {
		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		// Mock state - mixed statuses to test different code paths
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template",
				Branch:       "master",
				LatestCommit: "new123",
				LastChecked:  time.Now(),
			},
			Targets: map[string]*state.TargetState{
				"org/target-a": {
					Repo:           "org/target-a",
					LastSyncCommit: "new123", // Up to date - should be skipped
					Status:         state.StatusUpToDate,
				},
				"org/target-b": {
					Repo:           "org/target-b",
					LastSyncCommit: "old123", // Behind source - will fail
					Status:         state.StatusBehind,
				},
				"org/target-c": {
					Repo:           "org/target-c",
					LastSyncCommit: "old123", // Behind source - will fail
					Status:         state.StatusBehind,
				},
			},
		}

		stateDiscoverer.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		// Mock failures for targets that need sync
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.Anything).
			Return(errCloneFailed).Maybe()

		// Create engine
		opts := &Options{
			DryRun:         false,
			MaxConcurrency: 3, // Allow all to run concurrently
		}
		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
		engine.SetLogger(logrus.New())

		// Execute sync
		err := engine.Sync(context.Background(), nil)

		// Should fail due to errgroup failing on first error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to complete sync operation")

		stateDiscoverer.AssertExpectations(t)
	})

	t.Run("context cancellation during concurrent execution", func(t *testing.T) {
		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		// Mock state discovery - all targets behind
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template",
				Branch:       "master",
				LatestCommit: "new123",
				LastChecked:  time.Now(),
			},
			Targets: map[string]*state.TargetState{
				"org/target-a": {
					Repo:           "org/target-a",
					LastSyncCommit: "old123",
					Status:         state.StatusBehind,
				},
				"org/target-b": {
					Repo:           "org/target-b",
					LastSyncCommit: "old123",
					Status:         state.StatusBehind,
				},
				"org/target-c": {
					Repo:           "org/target-c",
					LastSyncCommit: "old123",
					Status:         state.StatusBehind,
				},
			},
		}

		stateDiscoverer.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		// Mock clone operations to check for context cancellation
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.Anything).
			Return(context.Canceled).Maybe()

		// Create engine
		opts := &Options{
			DryRun:         false,
			MaxConcurrency: 3,
		}
		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
		engine.SetLogger(logrus.New())

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Execute sync with timeout
		err := engine.Sync(ctx, nil)

		// Should fail due to context timeout
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to complete sync operation")
		// The underlying error should be context-related (any context timeout/cancellation is fine)
		errorMsg := err.Error()
		hasContextError := strings.Contains(errorMsg, "context deadline exceeded") ||
			strings.Contains(errorMsg, "context canceled") ||
			strings.Contains(errorMsg, "deadline exceeded") ||
			strings.Contains(errorMsg, "timeout") ||
			strings.Contains(errorMsg, "canceled")
		assert.True(t, hasContextError, "Error should be context-related: %v", err)

		stateDiscoverer.AssertExpectations(t)
	})

	t.Run("conflict status handling during concurrent sync", func(t *testing.T) {
		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		stateDiscoverer := &state.MockDiscoverer{}
		transformChain := &transform.MockChain{}

		// Mock state with conflict status to test the warning path
		currentState := &state.State{
			Source: state.SourceState{
				Repo:         "org/template",
				Branch:       "master",
				LatestCommit: "new123",
				LastChecked:  time.Now(),
			},
			Targets: map[string]*state.TargetState{
				"org/target-a": {
					Repo:   "org/target-a",
					Status: state.StatusConflict, // Should trigger warning log and be skipped
				},
				"org/target-b": {
					Repo:           "org/target-b",
					LastSyncCommit: "new123", // Same as source - up to date, will be skipped
					Status:         state.StatusUpToDate,
				},
				"org/target-c": {
					Repo:           "org/target-c",
					LastSyncCommit: "new123", // Same as source - up to date, will be skipped
					Status:         state.StatusUpToDate,
				},
			},
		}

		stateDiscoverer.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

		// Create engine
		opts := &Options{
			DryRun:         false,
			MaxConcurrency: 2,
		}
		engine := NewEngine(cfg, ghClient, gitClient, stateDiscoverer, transformChain, opts)
		engine.SetLogger(logrus.New())

		// Execute sync - should succeed since all targets are either conflicts (skipped with warning) or up-to-date
		err := engine.Sync(context.Background(), nil)

		// Should succeed - conflicts are just warnings, not errors, and up-to-date targets are skipped
		require.NoError(t, err)

		stateDiscoverer.AssertExpectations(t)
	})
}
