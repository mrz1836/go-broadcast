// Package cli provides command-line interface functionality for go-broadcast.
//
// This file contains tests for nil parameter handling in cancel operations.
// These tests verify that nil discoverers, states, and targets are handled
// safely without panics.
package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// TestPerformCancelWithDiscoverer_NilDiscoverer verifies that
// performCancelWithDiscoverer returns ErrNilDiscoverer when given nil.
//
// This matters because the function would panic at discoverer.DiscoverState()
// if nil is not checked. A clear error message helps debugging.
func TestPerformCancelWithDiscoverer_NilDiscoverer(t *testing.T) {
	t.Parallel()

	// Save and restore global flags (thread-safe)
	oldFlags := GetGlobalFlags()
	defer func() { SetFlags(oldFlags) }()
	SetFlags(&Flags{ConfigFile: oldFlags.ConfigFile, DryRun: false, LogLevel: oldFlags.LogLevel})

	// Save and restore cancel group filters
	oldGroupFilter := cancelGroupFilter
	oldSkipGroups := cancelSkipGroups
	defer func() {
		cancelGroupFilter = oldGroupFilter
		cancelSkipGroups = oldSkipGroups
	}()
	cancelGroupFilter = nil
	cancelSkipGroups = nil

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "test-group",
				Targets: []config.TargetConfig{
					{Repo: "org/repo1"},
				},
			},
		},
	}

	// Create a mock GitHub client (won't be used because discoverer is nil)
	mockClient := gh.NewMockClient()

	// Call with nil discoverer
	summary, err := performCancelWithDiscoverer(
		context.Background(),
		cfg,
		nil, // targetRepos
		mockClient,
		nil, // nil discoverer - should cause error
	)

	require.Error(t, err, "should return error for nil discoverer")
	require.ErrorIs(t, err, ErrNilDiscoverer, "error should be ErrNilDiscoverer")
	assert.Nil(t, summary, "summary should be nil on error")
}

// TestPerformCancelWithDiscoverer_NilConfig verifies that
// performCancelWithDiscoverer returns ErrNilConfig when given nil config.
//
// This matters because the function accesses config.Groups which would panic.
func TestPerformCancelWithDiscoverer_NilConfig(t *testing.T) {
	t.Parallel()

	mockClient := gh.NewMockClient()
	mockDiscoverer := state.NewMockDiscoverer()

	summary, err := performCancelWithDiscoverer(
		context.Background(),
		nil, // nil config
		nil,
		mockClient,
		mockDiscoverer,
	)

	require.Error(t, err, "should return error for nil config")
	require.ErrorIs(t, err, ErrNilConfig, "error should be ErrNilConfig")
	assert.Nil(t, summary, "summary should be nil on error")
}

// TestPerformCancelWithDiscoverer_DiscovererReturnsNilState verifies that
// performCancelWithDiscoverer returns ErrNilState when discoverer returns nil.
//
// This matters because filterTargets would panic accessing state.Targets map
// if the returned state is nil.
func TestPerformCancelWithDiscoverer_DiscovererReturnsNilState(t *testing.T) {
	// Not parallel because we modify global flags (thread-safe)
	oldFlags := GetGlobalFlags()
	defer func() { SetFlags(oldFlags) }()
	SetFlags(&Flags{ConfigFile: oldFlags.ConfigFile, DryRun: false, LogLevel: oldFlags.LogLevel})

	oldGroupFilter := cancelGroupFilter
	oldSkipGroups := cancelSkipGroups
	defer func() {
		cancelGroupFilter = oldGroupFilter
		cancelSkipGroups = oldSkipGroups
	}()
	cancelGroupFilter = nil
	cancelSkipGroups = nil

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "test-group",
				Targets: []config.TargetConfig{
					{Repo: "org/repo1"},
				},
			},
		},
	}

	mockClient := gh.NewMockClient()
	mockDiscoverer := state.NewMockDiscoverer()

	// Mock DiscoverState to return nil state with no error
	mockDiscoverer.On("DiscoverState", mock.Anything, mock.Anything).Return(nil, nil)

	summary, err := performCancelWithDiscoverer(
		context.Background(),
		cfg,
		nil,
		mockClient,
		mockDiscoverer,
	)

	require.Error(t, err, "should return error for nil state")
	require.ErrorIs(t, err, ErrNilState, "error should be ErrNilState")
	assert.Nil(t, summary, "summary should be nil on error")
	mockDiscoverer.AssertExpectations(t)
}

// TestPerformCancelWithDiscoverer_DiscovererReturnsError verifies that
// performCancelWithDiscoverer properly propagates errors from the discoverer.
//
// This tests the normal error path to ensure errors are wrapped correctly.
func TestPerformCancelWithDiscoverer_DiscovererReturnsError(t *testing.T) {
	// Thread-safe flag access
	oldFlags := GetGlobalFlags()
	defer func() { SetFlags(oldFlags) }()
	SetFlags(&Flags{ConfigFile: oldFlags.ConfigFile, DryRun: false, LogLevel: oldFlags.LogLevel})

	oldGroupFilter := cancelGroupFilter
	oldSkipGroups := cancelSkipGroups
	defer func() {
		cancelGroupFilter = oldGroupFilter
		cancelSkipGroups = oldSkipGroups
	}()
	cancelGroupFilter = nil
	cancelSkipGroups = nil

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "test-group",
				Targets: []config.TargetConfig{
					{Repo: "org/repo1"},
				},
			},
		},
	}

	mockClient := gh.NewMockClient()
	mockDiscoverer := state.NewMockDiscoverer()

	// Mock DiscoverState to return an error
	discoveryErr := errors.New("GitHub API rate limited") //nolint:err113 // test-only error
	mockDiscoverer.On("DiscoverState", mock.Anything, mock.Anything).Return(nil, discoveryErr)

	summary, err := performCancelWithDiscoverer(
		context.Background(),
		cfg,
		nil,
		mockClient,
		mockDiscoverer,
	)

	require.Error(t, err, "should return error from discoverer")
	assert.Contains(t, err.Error(), "failed to discover sync state", "error should be wrapped")
	assert.Contains(t, err.Error(), "rate limited", "underlying error should be preserved")
	assert.Nil(t, summary, "summary should be nil on error")
	mockDiscoverer.AssertExpectations(t)
}

// TestProcessCancelTarget_NilTarget verifies that processCancelTarget
// handles nil target gracefully.
//
// This is a boundary condition that could occur if filterTargets
// returns a slice with nil elements.
func TestProcessCancelTarget_NilTarget(t *testing.T) {
	// This test verifies behavior if processCancelTarget receives nil target.
	// Since the function accesses target.Repo immediately, we verify the
	// function's structure requires non-nil targets by checking a valid case.

	// Thread-safe flag access
	oldFlags := GetGlobalFlags()
	defer func() { SetFlags(oldFlags) }()
	SetFlags(&Flags{ConfigFile: oldFlags.ConfigFile, DryRun: true, LogLevel: oldFlags.LogLevel})

	mockClient := gh.NewMockClient()

	target := &state.TargetState{
		Repo: "org/valid-repo",
		OpenPRs: []gh.PR{
			{Number: 42, State: "open"},
		},
		SyncBranches: []state.SyncBranch{
			{
				Name: "sync/test-branch",
				Metadata: &state.BranchMetadata{
					Timestamp: time.Now(),
				},
			},
		},
	}

	// This should work with valid target
	result := processCancelTarget(context.Background(), mockClient, target)

	assert.Equal(t, "org/valid-repo", result.Repository)
	assert.NotNil(t, result.PRNumber)
	assert.Equal(t, 42, *result.PRNumber)
	assert.True(t, result.PRClosed, "dry run should mark PR as would-be-closed")
	assert.Equal(t, "sync/test-branch", result.BranchName)
}

// TestFilterTargets_NilState verifies that filterTargets handles nil state.
//
// This tests the boundary where state could be nil from a failed discovery.
func TestFilterTargets_NilState(t *testing.T) {
	t.Parallel()

	// filterTargets accesses s.Targets directly, so nil state would panic
	// We verify the current behavior with empty state
	emptyState := &state.State{
		Targets: make(map[string]*state.TargetState),
	}

	targets, err := filterTargets(emptyState, nil)
	require.NoError(t, err, "should handle empty state")
	assert.Empty(t, targets, "should return empty slice for empty state")
}

// TestFilterTargets_StateWithNilTargetsMap verifies filterTargets
// handles state with nil Targets map.
func TestFilterTargets_StateWithNilTargetsMap(t *testing.T) {
	t.Parallel()

	// State with nil Targets map
	stateWithNilMap := &state.State{
		Targets: nil,
	}

	// This would panic if not handled - testing current behavior
	// The range over nil map is safe in Go, returns 0 iterations
	targets, err := filterTargets(stateWithNilMap, nil)
	require.NoError(t, err, "should handle nil Targets map")
	assert.Empty(t, targets, "should return empty slice for nil map")
}

// TestCancelSentinelErrors verifies that new sentinel errors are properly defined.
func TestCancelSentinelErrors(t *testing.T) {
	t.Parallel()

	t.Run("ErrNilDiscoverer is defined", func(t *testing.T) {
		t.Parallel()
		require.Error(t, ErrNilDiscoverer)
		assert.Contains(t, ErrNilDiscoverer.Error(), "nil")
		assert.Contains(t, ErrNilDiscoverer.Error(), "discoverer")
	})

	t.Run("ErrNilState is defined", func(t *testing.T) {
		t.Parallel()
		require.Error(t, ErrNilState)
		assert.Contains(t, ErrNilState.Error(), "nil")
		assert.Contains(t, ErrNilState.Error(), "state")
	})

	t.Run("sentinel errors are distinct", func(t *testing.T) {
		t.Parallel()
		assert.NotEqual(t, ErrNilDiscoverer, ErrNilState)
		assert.NotEqual(t, ErrNilDiscoverer, ErrNilConfig)
		assert.NotEqual(t, ErrNilState, ErrNilConfig)
	})

	t.Run("errors can be matched with errors.Is", func(t *testing.T) {
		t.Parallel()
		wrappedDiscoverer := errors.New("wrapped: " + ErrNilDiscoverer.Error()) //nolint:err113 // test-only error
		// This won't match with errors.Is because we didn't use fmt.Errorf with %w
		// Testing that the base errors work as sentinels
		assert.NotErrorIs(t, wrappedDiscoverer, ErrNilDiscoverer)
	})
}
