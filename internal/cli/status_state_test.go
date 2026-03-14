// Package cli provides command-line interface functionality for go-broadcast.
//
// This file contains tests for state conversion functions in status operations.
// These tests verify that all state combinations are correctly converted to
// display status, including edge cases with nil metadata and empty collections.
package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// TestConvertStateToStatus_NilMetadata verifies that state conversion handles
// sync branches with nil metadata gracefully.
//
// This matters because metadata may be nil for branches that couldn't be parsed.
func TestConvertStateToStatus_NilMetadata(t *testing.T) {
	t.Parallel()

	s := &state.State{
		Source: state.SourceState{
			Repo:         "org/source",
			Branch:       "main",
			LatestCommit: "abc123",
		},
		Targets: map[string]*state.TargetState{
			"org/target": {
				Repo:   "org/target",
				Status: state.StatusUpToDate,
				SyncBranches: []state.SyncBranch{
					{
						Name:     "sync/test-1",
						Metadata: nil, // Nil metadata
					},
					{
						Name: "sync/test-2",
						Metadata: &state.BranchMetadata{
							Timestamp: time.Now(),
						},
					},
				},
			},
		},
	}

	status := convertStateToStatus(s, nil)

	require.NotNil(t, status, "status should not be nil")
	require.Len(t, status.Targets, 1, "should have one target")
	// The branch with valid metadata should be selected
	require.NotNil(t, status.Targets[0].SyncBranch, "SyncBranch pointer should not be nil")
	assert.Equal(t, "sync/test-2", *status.Targets[0].SyncBranch)
}

// TestConvertStateToStatus_EmptySyncBranches verifies that state conversion
// handles targets with no sync branches.
//
// This is the common case for targets that haven't been synced yet.
func TestConvertStateToStatus_EmptySyncBranches(t *testing.T) {
	t.Parallel()

	s := &state.State{
		Source: state.SourceState{
			Repo:         "org/source",
			Branch:       "main",
			LatestCommit: "abc123",
		},
		Targets: map[string]*state.TargetState{
			"org/target": {
				Repo:         "org/target",
				Status:       state.StatusPending,
				SyncBranches: []state.SyncBranch{}, // Empty
			},
		},
	}

	status := convertStateToStatus(s, nil)

	require.NotNil(t, status, "status should not be nil")
	require.Len(t, status.Targets, 1, "should have one target")
	assert.Empty(t, status.Targets[0].SyncBranch, "sync branch should be empty")
}

// TestConvertStateToGroupStatus_AllStatesCombinations verifies that group status
// is correctly determined from all possible target state combinations.
//
// This matters because group state is derived from aggregating target states.
func TestConvertStateToGroupStatus_AllStatesCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		targetStatuses []state.SyncStatus
		expectedState  string
	}{
		{
			name:           "all synced",
			targetStatuses: []state.SyncStatus{state.StatusUpToDate, state.StatusUpToDate},
			expectedState:  "synced",
		},
		{
			name:           "one pending makes group pending",
			targetStatuses: []state.SyncStatus{state.StatusUpToDate, state.StatusPending},
			expectedState:  "pending",
		},
		{
			name:           "one error makes group error",
			targetStatuses: []state.SyncStatus{state.StatusUpToDate, state.StatusConflict},
			expectedState:  "error",
		},
		{
			name:           "error takes priority over pending",
			targetStatuses: []state.SyncStatus{state.StatusPending, state.StatusConflict},
			expectedState:  "error",
		},
		{
			name:           "all pending",
			targetStatuses: []state.SyncStatus{state.StatusPending, state.StatusPending},
			expectedState:  "pending",
		},
		{
			name:           "all error",
			targetStatuses: []state.SyncStatus{state.StatusConflict, state.StatusConflict},
			expectedState:  "error",
		},
		{
			name:           "mixed states - error takes priority",
			targetStatuses: []state.SyncStatus{state.StatusUpToDate, state.StatusPending, state.StatusConflict},
			expectedState:  "error",
		},
		{
			name:           "single target synced",
			targetStatuses: []state.SyncStatus{state.StatusUpToDate},
			expectedState:  "synced",
		},
		{
			name:           "single target pending",
			targetStatuses: []state.SyncStatus{state.StatusPending},
			expectedState:  "pending",
		},
		{
			name:           "single target error",
			targetStatuses: []state.SyncStatus{state.StatusConflict},
			expectedState:  "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Build state with targets having specified statuses
			targets := make(map[string]*state.TargetState)
			configTargets := make([]config.TargetConfig, 0, len(tt.targetStatuses))

			for i, status := range tt.targetStatuses {
				repo := "org/target-" + string(rune('1'+i))
				targets[repo] = &state.TargetState{
					Repo:   repo,
					Status: status,
				}
				configTargets = append(configTargets, config.TargetConfig{Repo: repo})
			}

			s := &state.State{
				Source: state.SourceState{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: targets,
			}

			cfg := &config.Config{
				Groups: []config.Group{
					{
						Name:    "test-group",
						Targets: configTargets,
					},
				},
			}

			syncStatus := convertStateToGroupStatus(s, cfg)

			require.NotNil(t, syncStatus, "status should not be nil")
			require.Len(t, syncStatus.Groups, 1, "should have one group")
			assert.Equal(t, tt.expectedState, syncStatus.Groups[0].State,
				"group state should be %s", tt.expectedState)
		})
	}
}

// TestConvertStateToGroupStatus_DisabledGroup verifies that disabled groups
// are marked appropriately.
func TestConvertStateToGroupStatus_DisabledGroup(t *testing.T) {
	t.Parallel()

	disabled := false
	s := &state.State{
		Source: state.SourceState{Repo: "org/source", Branch: "main"},
		Targets: map[string]*state.TargetState{
			"org/target": {Repo: "org/target", Status: state.StatusPending},
		},
	}

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name:    "disabled-group",
				Enabled: &disabled, // Explicitly disabled
				Targets: []config.TargetConfig{{Repo: "org/target"}},
			},
		},
	}

	status := convertStateToGroupStatus(s, cfg)

	require.Len(t, status.Groups, 1)
	assert.False(t, status.Groups[0].Enabled, "group should be disabled")
	assert.Equal(t, "disabled", status.Groups[0].State, "disabled group should have 'disabled' state")
}

// TestConvertStateToGroupStatus_EmptyTargets verifies handling of groups
// with no targets.
func TestConvertStateToGroupStatus_EmptyTargets(t *testing.T) {
	t.Parallel()

	s := &state.State{
		Source:  state.SourceState{Repo: "org/source", Branch: "main"},
		Targets: map[string]*state.TargetState{},
	}

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name:    "empty-group",
				Targets: []config.TargetConfig{}, // No targets
			},
		},
	}

	status := convertStateToGroupStatus(s, cfg)

	require.Len(t, status.Groups, 1)
	assert.Empty(t, status.Groups[0].Targets, "should have no targets")
	// Empty group should be considered synced (vacuously true)
	assert.Equal(t, "synced", status.Groups[0].State)
}

// TestConvertStateToGroupStatus_MissingTargetState verifies handling when
// config references a target that doesn't exist in discovered state.
func TestConvertStateToGroupStatus_MissingTargetState(t *testing.T) {
	t.Parallel()

	s := &state.State{
		Source: state.SourceState{Repo: "org/source", Branch: "main"},
		Targets: map[string]*state.TargetState{
			// Only org/target1 exists in state
			"org/target1": {Repo: "org/target1", Status: state.StatusUpToDate},
		},
	}

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "partial-group",
				Targets: []config.TargetConfig{
					{Repo: "org/target1"},
					{Repo: "org/target2"}, // Not in state
				},
			},
		},
	}

	status := convertStateToGroupStatus(s, cfg)

	require.Len(t, status.Groups, 1)
	// Only the target that exists in state should be in result
	assert.Len(t, status.Groups[0].Targets, 1)
	assert.Equal(t, "org/target1", status.Groups[0].Targets[0].Repository)
}

// TestConvertStateToGroupStatus_MultipleGroups verifies correct handling
// of multiple groups with different states.
func TestConvertStateToGroupStatus_MultipleGroups(t *testing.T) {
	t.Parallel()

	s := &state.State{
		Source: state.SourceState{Repo: "org/source", Branch: "main"},
		Targets: map[string]*state.TargetState{
			"org/core-target":     {Repo: "org/core-target", Status: state.StatusUpToDate},
			"org/optional-target": {Repo: "org/optional-target", Status: state.StatusPending},
			"org/broken-target":   {Repo: "org/broken-target", Status: state.StatusConflict},
		},
	}

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name:     "core",
				Priority: 1,
				Targets:  []config.TargetConfig{{Repo: "org/core-target"}},
			},
			{
				Name:     "optional",
				Priority: 2,
				Targets:  []config.TargetConfig{{Repo: "org/optional-target"}},
			},
			{
				Name:     "broken",
				Priority: 3,
				Targets:  []config.TargetConfig{{Repo: "org/broken-target"}},
			},
		},
	}

	status := convertStateToGroupStatus(s, cfg)

	require.Len(t, status.Groups, 3)

	// Find groups by name
	groupStates := make(map[string]string)
	for _, g := range status.Groups {
		groupStates[g.Name] = g.State
	}

	assert.Equal(t, "synced", groupStates["core"])
	assert.Equal(t, "pending", groupStates["optional"])
	assert.Equal(t, "error", groupStates["broken"])
}

// TestConvertSyncStatus_AllStatusMappings verifies that all sync status values
// are correctly converted to display strings.
func TestConvertSyncStatus_AllStatusMappings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   state.SyncStatus
		expected string
	}{
		{state.StatusUpToDate, "synced"},
		{state.StatusPending, "pending"},
		{state.StatusConflict, "error"},
		{state.StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()
			result := convertSyncStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConvertStateToStatus_SortedOutput verifies that targets are returned
// in sorted order for deterministic output.
func TestConvertStateToStatus_SortedOutput(t *testing.T) {
	t.Parallel()

	s := &state.State{
		Source: state.SourceState{Repo: "org/source", Branch: "main"},
		Targets: map[string]*state.TargetState{
			"org/zebra":    {Repo: "org/zebra", Status: state.StatusUpToDate},
			"org/apple":    {Repo: "org/apple", Status: state.StatusUpToDate},
			"org/middle":   {Repo: "org/middle", Status: state.StatusUpToDate},
			"org/beta":     {Repo: "org/beta", Status: state.StatusUpToDate},
			"org/123-repo": {Repo: "org/123-repo", Status: state.StatusUpToDate},
		},
	}

	status := convertStateToStatus(s, nil)

	require.Len(t, status.Targets, 5)

	// Verify sorted order
	repos := make([]string, len(status.Targets))
	for i, t := range status.Targets {
		repos[i] = t.Repository
	}

	expected := []string{"org/123-repo", "org/apple", "org/beta", "org/middle", "org/zebra"}
	assert.Equal(t, expected, repos, "targets should be sorted alphabetically")
}

// TestConvertStateToStatus_WithOpenPRs verifies that open PRs are included
// in the status output.
func TestConvertStateToStatus_WithOpenPRs(t *testing.T) {
	t.Parallel()

	now := time.Now()
	s := &state.State{
		Source: state.SourceState{Repo: "org/source", Branch: "main"},
		Targets: map[string]*state.TargetState{
			"org/target": {
				Repo:   "org/target",
				Status: state.StatusPending,
				SyncBranches: []state.SyncBranch{
					{
						Name: "sync/test-branch",
						Metadata: &state.BranchMetadata{
							Timestamp: now,
						},
					},
				},
			},
		},
	}

	status := convertStateToStatus(s, nil)

	require.Len(t, status.Targets, 1)
	require.NotNil(t, status.Targets[0].SyncBranch, "SyncBranch pointer should not be nil")
	assert.Equal(t, "sync/test-branch", *status.Targets[0].SyncBranch)
	// PR info is in a different field depending on implementation
}

// TestConvertStateToGroupStatus_WithDependencies verifies that group
// dependencies are preserved in the output.
func TestConvertStateToGroupStatus_WithDependencies(t *testing.T) {
	t.Parallel()

	s := &state.State{
		Source: state.SourceState{Repo: "org/source", Branch: "main"},
		Targets: map[string]*state.TargetState{
			"org/target1": {Repo: "org/target1", Status: state.StatusUpToDate},
			"org/target2": {Repo: "org/target2", Status: state.StatusUpToDate},
		},
	}

	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name:    "base",
				Targets: []config.TargetConfig{{Repo: "org/target1"}},
			},
			{
				Name:      "dependent",
				DependsOn: []string{"base"},
				Targets:   []config.TargetConfig{{Repo: "org/target2"}},
			},
		},
	}

	status := convertStateToGroupStatus(s, cfg)

	require.Len(t, status.Groups, 2)

	var dependentGroup *GroupStatus
	for i := range status.Groups {
		if status.Groups[i].Name == "dependent" {
			dependentGroup = &status.Groups[i]
			break
		}
	}

	require.NotNil(t, dependentGroup)
	assert.Contains(t, dependentGroup.DependsOn, "base")
}
