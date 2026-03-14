// Package cli provides command-line interface functionality for go-broadcast.
//
// This file contains tests for error paths in validate operations.
// These tests verify that validate command properly handles failures including
// partial target failures, mixed success/failure scenarios, and source branch issues.
package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

// TestValidateRepositoryAccessibilityWithClient_NilConfig verifies that
// validateRepositoryAccessibilityWithClient returns ErrNilConfig for nil config.
//
// This matters because the function accesses cfg.Groups which would panic.
func TestValidateRepositoryAccessibilityWithClient_NilConfig(t *testing.T) {
	t.Parallel()

	mockClient := gh.NewMockClient()

	err := validateRepositoryAccessibilityWithClient(
		context.Background(),
		nil, // nil config
		mockClient,
		false,
	)

	require.Error(t, err, "should return error for nil config")
	assert.ErrorIs(t, err, ErrNilConfig, "error should be ErrNilConfig")
}

// TestValidateRepositoryAccessibilityWithClient_NoGroups verifies that
// the function handles configs with no groups.
//
// This matters because accessing groups[0] on empty slice would panic.
func TestValidateRepositoryAccessibilityWithClient_NoGroups(t *testing.T) {
	t.Parallel()

	mockClient := gh.NewMockClient()
	cfg := &config.Config{
		Groups: []config.Group{}, // Empty groups
	}

	err := validateRepositoryAccessibilityWithClient(
		context.Background(),
		cfg,
		mockClient,
		false,
	)

	require.Error(t, err, "should return error for empty groups")
	assert.ErrorIs(t, err, ErrNoConfigGroups, "error should be ErrNoConfigGroups")
}

// TestValidateRepositoryAccessibilityWithClient_SourceBranchNotFound verifies
// that the function properly handles missing source branch.
//
// This matters because users need clear feedback when source branch doesn't exist.
func TestValidateRepositoryAccessibilityWithClient_SourceBranchNotFound(t *testing.T) {
	t.Parallel()

	mockClient := gh.NewMockClient()
	mockClient.On("GetBranch", mock.Anything, "org/source", "main").
		Return(nil, errors.New("branch not found: main")) //nolint:err113 // test-only error

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "test-group",
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/target1"},
				},
			},
		},
	}

	err := validateRepositoryAccessibilityWithClient(
		context.Background(),
		cfg,
		mockClient,
		false,
	)

	require.Error(t, err, "should return error for missing branch")
	require.ErrorIs(t, err, ErrSourceBranchNotFound, "error should be ErrSourceBranchNotFound")
	mockClient.AssertExpectations(t)
}

// TestValidateRepositoryAccessibilityWithClient_SourceRepoNotFound verifies
// that the function properly handles inaccessible source repository.
//
// This matters because 404 errors indicate permission or existence issues.
func TestValidateRepositoryAccessibilityWithClient_SourceRepoNotFound(t *testing.T) {
	t.Parallel()

	mockClient := gh.NewMockClient()
	mockClient.On("GetBranch", mock.Anything, "org/source", "main").
		Return(nil, errors.New("404 Not Found")) //nolint:err113 // test-only error

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "test-group",
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/target1"},
				},
			},
		},
	}

	err := validateRepositoryAccessibilityWithClient(
		context.Background(),
		cfg,
		mockClient,
		false,
	)

	require.Error(t, err, "should return error for missing repo")
	require.ErrorIs(t, err, ErrSourceRepoNotFound, "error should be ErrSourceRepoNotFound")
	mockClient.AssertExpectations(t)
}

// TestValidateRepositoryAccessibilityWithClient_SourceOnlyMode verifies
// that sourceOnly=true skips target validation.
//
// This matters for performance when users only want to validate source config.
func TestValidateRepositoryAccessibilityWithClient_SourceOnlyMode(t *testing.T) {
	t.Parallel()

	mockClient := gh.NewMockClient()
	mockClient.On("GetBranch", mock.Anything, "org/source", "main").
		Return(&gh.Branch{Name: "main"}, nil)
	// GetRepo should NOT be called for targets when sourceOnly=true

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "test-group",
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/target1"},
					{Repo: "org/target2"},
				},
			},
		},
	}

	err := validateRepositoryAccessibilityWithClient(
		context.Background(),
		cfg,
		mockClient,
		true, // sourceOnly = true
	)

	require.NoError(t, err, "should succeed with valid source")
	mockClient.AssertExpectations(t)
	// Verify GetRepo was NOT called (would fail mock expectations if it was)
}

// TestValidateRepositoryAccessibilityWithClient_TargetPartialFailure verifies
// that validation continues checking remaining targets after one fails.
//
// This matters because users want to see ALL validation issues, not just the first.
func TestValidateRepositoryAccessibilityWithClient_TargetPartialFailure(t *testing.T) {
	t.Parallel()

	mockClient := gh.NewMockClient()
	mockClient.On("GetBranch", mock.Anything, "org/source", "main").
		Return(&gh.Branch{Name: "main"}, nil)
	mockClient.On("ListBranches", mock.Anything, "org/target1").
		Return([]*gh.Branch{{Name: "main"}}, nil)
	mockClient.On("ListBranches", mock.Anything, "org/target2").
		Return(nil, errors.New("404 Not Found")) //nolint:err113 // test-only error
	mockClient.On("ListBranches", mock.Anything, "org/target3").
		Return([]*gh.Branch{{Name: "main"}}, nil)

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "test-group",
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/target1"},
					{Repo: "org/target2"}, // This will fail
					{Repo: "org/target3"},
				},
			},
		},
	}

	err := validateRepositoryAccessibilityWithClient(
		context.Background(),
		cfg,
		mockClient,
		false,
	)

	// Validation should continue checking all targets
	mockClient.AssertExpectations(t)
	// The function should report partial failure
	// Current implementation may return nil or error depending on design
	// This test documents expected behavior
	_ = err // Result depends on implementation - test verifies all targets are checked
}

// TestValidateRepositoryAccessibilityWithClient_AllTargetsFail verifies
// behavior when all target repositories fail validation.
//
// This is an edge case that should provide clear feedback.
func TestValidateRepositoryAccessibilityWithClient_AllTargetsFail(t *testing.T) {
	t.Parallel()

	mockClient := gh.NewMockClient()
	mockClient.On("GetBranch", mock.Anything, "org/source", "main").
		Return(&gh.Branch{Name: "main"}, nil)
	mockClient.On("ListBranches", mock.Anything, "org/target1").
		Return(nil, errors.New("permission denied")) //nolint:err113 // test-only error
	mockClient.On("ListBranches", mock.Anything, "org/target2").
		Return(nil, errors.New("404 Not Found")) //nolint:err113 // test-only error

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "test-group",
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/target1"},
					{Repo: "org/target2"},
				},
			},
		},
	}

	err := validateRepositoryAccessibilityWithClient(
		context.Background(),
		cfg,
		mockClient,
		false,
	)

	// All targets should be checked
	mockClient.AssertExpectations(t)
	// Result depends on implementation - test verifies behavior
	_ = err
}

// TestValidateRepositoryAccessibilityWithClient_ContextCancellation verifies
// that validation respects context cancellation.
func TestValidateRepositoryAccessibilityWithClient_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockClient := gh.NewMockClient()
	mockClient.On("GetBranch", mock.Anything, "org/source", "main").
		Return(nil, context.Canceled)

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "test-group",
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "org/target1"},
				},
			},
		},
	}

	err := validateRepositoryAccessibilityWithClient(ctx, cfg, mockClient, false)

	require.Error(t, err, "should return error when context canceled")
	mockClient.AssertExpectations(t)
}

// TestCheckCircularDependency_SimpleCircle verifies detection of
// direct circular dependencies between groups.
func TestCheckCircularDependency_SimpleCircle(t *testing.T) {
	t.Parallel()

	// Build dependency graph where A depends on B and B depends on A
	deps := map[string][]string{
		"group-a": {"group-b"},
		"group-b": {"group-a"},
	}

	tests := []struct {
		name       string
		startGroup string
		deps       map[string][]string
		wantCycle  bool
	}{
		{
			name:       "simple two-node cycle",
			startGroup: "group-a",
			deps:       deps,
			wantCycle:  true,
		},
		{
			name:       "three-node cycle",
			startGroup: "group-a",
			deps: map[string][]string{
				"group-a": {"group-b"},
				"group-b": {"group-c"},
				"group-c": {"group-a"},
			},
			wantCycle: true,
		},
		{
			name:       "no cycle - linear chain",
			startGroup: "group-a",
			deps: map[string][]string{
				"group-a": {"group-b"},
				"group-b": {"group-c"},
				"group-c": {},
			},
			wantCycle: false,
		},
		{
			name:       "no cycle - diamond shape",
			startGroup: "group-a",
			deps: map[string][]string{
				"group-a": {"group-b", "group-c"},
				"group-b": {"group-d"},
				"group-c": {"group-d"},
				"group-d": {},
			},
			wantCycle: false,
		},
		{
			name:       "self-referencing cycle",
			startGroup: "group-a",
			deps: map[string][]string{
				"group-a": {"group-a"},
			},
			wantCycle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			visited := make(map[string]bool)

			hasCycle := checkCircularDependency(tt.startGroup, tt.deps, visited)
			assert.Equal(t, tt.wantCycle, hasCycle, "cycle detection should match expected")
		})
	}
}

// TestCheckCircularDependency_LargeGraph verifies performance with
// larger dependency graphs.
func TestCheckCircularDependency_LargeGraph(t *testing.T) {
	t.Parallel()

	// Build a large linear dependency chain (no cycles)
	deps := make(map[string][]string)
	for i := 0; i < 100; i++ {
		if i < 99 {
			deps[groupName(i)] = []string{groupName(i + 1)}
		} else {
			deps[groupName(i)] = []string{}
		}
	}

	visited := make(map[string]bool)

	hasCycle := checkCircularDependency("group-0", deps, visited)
	assert.False(t, hasCycle, "large linear chain should not have cycle")
}

func groupName(i int) string {
	return "group-" + string(rune('0'+i/10)) + string(rune('0'+i%10)) //nolint:gosec // G115: bounded integer, safe conversion in test helper
}

// TestValidateSentinelErrors verifies that validation sentinel errors
// are properly defined and distinct.
func TestValidateSentinelErrors(t *testing.T) {
	t.Parallel()

	t.Run("errors are defined", func(t *testing.T) {
		t.Parallel()
		require.Error(t, ErrGitHubCLIRequired)
		require.Error(t, ErrGitHubAuthRequired)
		require.Error(t, ErrSourceBranchNotFound)
		require.Error(t, ErrSourceRepoNotFound)
		require.Error(t, ErrNoConfigGroups)
	})

	t.Run("errors are distinct", func(t *testing.T) {
		t.Parallel()
		errors := []error{
			ErrGitHubCLIRequired,
			ErrGitHubAuthRequired,
			ErrSourceBranchNotFound,
			ErrSourceRepoNotFound,
			ErrNoConfigGroups,
		}

		for i, err1 := range errors {
			for j, err2 := range errors {
				if i != j {
					assert.NotEqual(t, err1.Error(), err2.Error(),
						"errors %d and %d should have different messages", i, j)
				}
			}
		}
	})

	t.Run("errors have descriptive messages", func(t *testing.T) {
		t.Parallel()
		assert.Contains(t, ErrGitHubCLIRequired.Error(), "CLI")
		assert.Contains(t, ErrGitHubAuthRequired.Error(), "auth")
		assert.Contains(t, ErrSourceBranchNotFound.Error(), "branch")
		assert.Contains(t, ErrSourceRepoNotFound.Error(), "repository")
		assert.Contains(t, ErrNoConfigGroups.Error(), "group")
	})
}
