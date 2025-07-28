package state

import (
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/stretchr/testify/require"
)

func TestSyncStatus_String(t *testing.T) {
	tests := []struct {
		status   SyncStatus
		expected string
	}{
		{StatusUnknown, "unknown"},
		{StatusUpToDate, "up-to-date"},
		{StatusBehind, "behind"},
		{StatusPending, "pending"},
		{StatusConflict, "conflict"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			require.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestState_Structure(t *testing.T) {
	now := time.Now()
	state := State{
		Source: SourceState{
			Repo:         "org/template-repo",
			Branch:       "main",
			LatestCommit: "abc123",
			LastChecked:  now,
		},
		Targets: map[string]*TargetState{
			"org/service-a": {
				Repo: "org/service-a",
				SyncBranches: []SyncBranch{
					{
						Name: "sync/template-20240101-120000-abc123",
						Metadata: &BranchMetadata{
							Timestamp: now.Add(-24 * time.Hour),
							CommitSHA: "abc123",
							Prefix:    "sync/template",
						},
					},
				},
				OpenPRs: []gh.PR{
					{
						Number: 42,
						State:  "open",
						Title:  "Sync from template",
					},
				},
				LastSyncCommit: "abc123",
				LastSyncTime:   &now,
				Status:         StatusUpToDate,
			},
			"org/service-b": {
				Repo:           "org/service-b",
				SyncBranches:   []SyncBranch{},
				OpenPRs:        []gh.PR{},
				LastSyncCommit: "def456",
				LastSyncTime:   nil,
				Status:         StatusBehind,
			},
		},
	}

	// Verify source state
	require.Equal(t, "org/template-repo", state.Source.Repo)
	require.Equal(t, "main", state.Source.Branch)
	require.Equal(t, "abc123", state.Source.LatestCommit)
	require.Equal(t, now, state.Source.LastChecked)

	// Verify targets
	require.Len(t, state.Targets, 2)

	// Verify service-a
	serviceA := state.Targets["org/service-a"]
	require.NotNil(t, serviceA)
	require.Equal(t, "org/service-a", serviceA.Repo)
	require.Len(t, serviceA.SyncBranches, 1)
	require.Len(t, serviceA.OpenPRs, 1)
	require.Equal(t, "abc123", serviceA.LastSyncCommit)
	require.NotNil(t, serviceA.LastSyncTime)
	require.Equal(t, StatusUpToDate, serviceA.Status)

	// Verify service-b
	serviceB := state.Targets["org/service-b"]
	require.NotNil(t, serviceB)
	require.Equal(t, "org/service-b", serviceB.Repo)
	require.Empty(t, serviceB.SyncBranches)
	require.Empty(t, serviceB.OpenPRs)
	require.Equal(t, "def456", serviceB.LastSyncCommit)
	require.Nil(t, serviceB.LastSyncTime)
	require.Equal(t, StatusBehind, serviceB.Status)
}

func TestSourceState_DefaultValues(t *testing.T) {
	var source SourceState
	require.Empty(t, source.Repo)
	require.Empty(t, source.Branch)
	require.Empty(t, source.LatestCommit)
	require.True(t, source.LastChecked.IsZero())
}

func TestTargetState_DefaultValues(t *testing.T) {
	var target TargetState
	require.Empty(t, target.Repo)
	require.Nil(t, target.SyncBranches)
	require.Nil(t, target.OpenPRs)
	require.Empty(t, target.LastSyncCommit)
	require.Nil(t, target.LastSyncTime)
	require.Equal(t, SyncStatus(""), target.Status) // Zero value
}

func TestSyncBranch_Structure(t *testing.T) {
	timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	branch := SyncBranch{
		Name: "sync/template-20240101-120000-abc123",
		Metadata: &BranchMetadata{
			Timestamp: timestamp,
			CommitSHA: "abc123",
			Prefix:    "sync/template",
		},
	}

	require.Equal(t, "sync/template-20240101-120000-abc123", branch.Name)
	require.NotNil(t, branch.Metadata)
	require.Equal(t, timestamp, branch.Metadata.Timestamp)
	require.Equal(t, "abc123", branch.Metadata.CommitSHA)
	require.Equal(t, "sync/template", branch.Metadata.Prefix)
}

func TestBranchMetadata_Structure(t *testing.T) {
	metadata := &BranchMetadata{
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		CommitSHA: "abc123def456",
		Prefix:    "sync/template",
	}

	require.Equal(t, 2024, metadata.Timestamp.Year())
	require.Equal(t, time.January, metadata.Timestamp.Month())
	require.Equal(t, 1, metadata.Timestamp.Day())
	require.Equal(t, 12, metadata.Timestamp.Hour())
	require.Equal(t, 0, metadata.Timestamp.Minute())
	require.Equal(t, 0, metadata.Timestamp.Second())
	require.Equal(t, "abc123def456", metadata.CommitSHA)
	require.Equal(t, "sync/template", metadata.Prefix)
}

func TestState_EmptyTargets(t *testing.T) {
	state := State{
		Source: SourceState{
			Repo: "org/template",
		},
		Targets: map[string]*TargetState{},
	}

	require.Empty(t, state.Targets)
	require.Equal(t, "org/template", state.Source.Repo)
}

func TestState_NilTargets(t *testing.T) {
	state := State{
		Source: SourceState{
			Repo: "org/template",
		},
		Targets: nil,
	}

	require.Nil(t, state.Targets)
}

func TestTargetState_MultipleOpenPRs(t *testing.T) {
	target := TargetState{
		Repo: "org/service",
		OpenPRs: []gh.PR{
			{Number: 1, State: "open", Title: "First PR"},
			{Number: 2, State: "open", Title: "Second PR"},
			{Number: 3, State: "open", Title: "Third PR"},
		},
		Status: StatusPending,
	}

	require.Len(t, target.OpenPRs, 3)
	require.Equal(t, 1, target.OpenPRs[0].Number)
	require.Equal(t, 2, target.OpenPRs[1].Number)
	require.Equal(t, 3, target.OpenPRs[2].Number)
	require.Equal(t, StatusPending, target.Status)
}

func TestTargetState_MultipleSyncBranches(t *testing.T) {
	now := time.Now()
	target := TargetState{
		Repo: "org/service",
		SyncBranches: []SyncBranch{
			{
				Name: "sync/template-20240101-120000-abc123",
				Metadata: &BranchMetadata{
					Timestamp: now.Add(-48 * time.Hour),
					CommitSHA: "abc123",
					Prefix:    "sync/template",
				},
			},
			{
				Name: "sync/template-20240102-120000-def456",
				Metadata: &BranchMetadata{
					Timestamp: now.Add(-24 * time.Hour),
					CommitSHA: "def456",
					Prefix:    "sync/template",
				},
			},
			{
				Name: "sync/template-20240103-120000-ghi789",
				Metadata: &BranchMetadata{
					Timestamp: now,
					CommitSHA: "ghi789",
					Prefix:    "sync/template",
				},
			},
		},
	}

	require.Len(t, target.SyncBranches, 3)

	// Verify branches are in expected order
	require.Equal(t, "abc123", target.SyncBranches[0].Metadata.CommitSHA)
	require.Equal(t, "def456", target.SyncBranches[1].Metadata.CommitSHA)
	require.Equal(t, "ghi789", target.SyncBranches[2].Metadata.CommitSHA)
}

func TestSyncBranch_NilMetadata(t *testing.T) {
	branch := SyncBranch{
		Name:     "sync/template-invalid",
		Metadata: nil,
	}

	require.Equal(t, "sync/template-invalid", branch.Name)
	require.Nil(t, branch.Metadata)
}

func TestBranchMetadata_EmptyCommitSHA(t *testing.T) {
	metadata := &BranchMetadata{
		Timestamp: time.Now(),
		CommitSHA: "",
		Prefix:    "sync/template",
	}

	require.Empty(t, metadata.CommitSHA)
	require.Equal(t, "sync/template", metadata.Prefix)
	require.False(t, metadata.Timestamp.IsZero())
}

func TestSyncStatus_Validation(t *testing.T) {
	// Test all valid statuses
	validStatuses := []SyncStatus{
		StatusUnknown,
		StatusUpToDate,
		StatusBehind,
		StatusPending,
		StatusConflict,
	}

	for _, status := range validStatuses {
		require.NotEmpty(t, status)
	}

	// Test that status values are distinct
	statusMap := make(map[SyncStatus]bool)
	for _, status := range validStatuses {
		require.False(t, statusMap[status], "Duplicate status value: %s", status)
		statusMap[status] = true
	}
}

func TestState_ComplexScenario(t *testing.T) {
	// Create a complex state with multiple targets in different states
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	state := State{
		Source: SourceState{
			Repo:         "org/template",
			Branch:       "develop",
			LatestCommit: "latest123",
			LastChecked:  now,
		},
		Targets: map[string]*TargetState{
			// Up-to-date target
			"org/service-current": {
				Repo:           "org/service-current",
				LastSyncCommit: "latest123",
				LastSyncTime:   &now,
				Status:         StatusUpToDate,
				SyncBranches:   []SyncBranch{},
				OpenPRs:        []gh.PR{},
			},
			// Behind target with open PR
			"org/service-behind": {
				Repo:           "org/service-behind",
				LastSyncCommit: "old456",
				LastSyncTime:   &lastWeek,
				Status:         StatusPending,
				SyncBranches: []SyncBranch{
					{
						Name: "sync/template-20240101-120000-latest123",
						Metadata: &BranchMetadata{
							Timestamp: yesterday,
							CommitSHA: "latest123",
							Prefix:    "sync/template",
						},
					},
				},
				OpenPRs: []gh.PR{
					{
						Number: 99,
						State:  "open",
						Title:  "Sync from template",
						Head: struct {
							Ref string `json:"ref"`
							SHA string `json:"sha"`
						}{
							Ref: "sync/template-20240101-120000-latest123",
						},
					},
				},
			},
			// Conflicted target
			"org/service-conflict": {
				Repo:           "org/service-conflict",
				LastSyncCommit: "conflict789",
				LastSyncTime:   &yesterday,
				Status:         StatusConflict,
				SyncBranches: []SyncBranch{
					{
						Name: "sync/template-20240101-120000-latest123",
						Metadata: &BranchMetadata{
							Timestamp: now,
							CommitSHA: "latest123",
							Prefix:    "sync/template",
						},
					},
				},
				OpenPRs: []gh.PR{
					{
						Number: 100,
						State:  "open",
						Title:  "Sync from template (conflicts)",
						Labels: []struct {
							Name string `json:"name"`
						}{
							{Name: "conflicts"},
						},
					},
				},
			},
			// Unknown status target (never synced)
			"org/service-new": {
				Repo:           "org/service-new",
				LastSyncCommit: "",
				LastSyncTime:   nil,
				Status:         StatusUnknown,
				SyncBranches:   []SyncBranch{},
				OpenPRs:        []gh.PR{},
			},
		},
	}

	// Verify we have all targets
	require.Len(t, state.Targets, 4)

	// Verify each target's status
	require.Equal(t, StatusUpToDate, state.Targets["org/service-current"].Status)
	require.Equal(t, StatusPending, state.Targets["org/service-behind"].Status)
	require.Equal(t, StatusConflict, state.Targets["org/service-conflict"].Status)
	require.Equal(t, StatusUnknown, state.Targets["org/service-new"].Status)

	// Verify sync times
	require.NotNil(t, state.Targets["org/service-current"].LastSyncTime)
	require.NotNil(t, state.Targets["org/service-behind"].LastSyncTime)
	require.NotNil(t, state.Targets["org/service-conflict"].LastSyncTime)
	require.Nil(t, state.Targets["org/service-new"].LastSyncTime)
}
