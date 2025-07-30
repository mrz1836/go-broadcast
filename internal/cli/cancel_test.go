package cli

import (
	"context"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockTime returns a fixed time for testing
func mockTime() time.Time {
	return time.Date(2023, time.January, 15, 10, 30, 0, 0, time.UTC)
}

func TestGenerateCancelComment(t *testing.T) {
	comment := generateCancelComment()

	assert.Contains(t, comment, "Sync Operation Canceled")
	assert.Contains(t, comment, "Manual cancellation via CLI")
	assert.Contains(t, comment, "go-broadcast")
}

func TestFilterTargets(t *testing.T) {
	// Create test state
	s := &state.State{
		Targets: map[string]*state.TargetState{
			"org/repo1": {
				Repo: "org/repo1",
				OpenPRs: []gh.PR{
					{Number: 123, State: "open"},
				},
			},
			"org/repo2": {
				Repo: "org/repo2",
				SyncBranches: []state.SyncBranch{
					{Name: "sync/test-branch"},
				},
			},
			"org/repo3": {
				Repo: "org/repo3",
				// No active syncs
			},
		},
	}

	tests := []struct {
		name        string
		targetRepos []string
		wantCount   int
		wantRepos   []string
		wantError   bool
	}{
		{
			name:        "all targets with active syncs",
			targetRepos: []string{},
			wantCount:   2,
			wantRepos:   []string{"org/repo1", "org/repo2"},
		},
		{
			name:        "specific target with active sync",
			targetRepos: []string{"org/repo1"},
			wantCount:   1,
			wantRepos:   []string{"org/repo1"},
		},
		{
			name:        "specific target without active sync",
			targetRepos: []string{"org/repo3"},
			wantCount:   0,
			wantRepos:   []string{},
		},
		{
			name:        "nonexistent target",
			targetRepos: []string{"org/nonexistent"},
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := filterTargets(s, tt.targetRepos)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, targets, tt.wantCount)

			// Check that returned targets match expected repos
			for i, target := range targets {
				if i < len(tt.wantRepos) {
					found := false
					for _, expectedRepo := range tt.wantRepos {
						if target.Repo == expectedRepo {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected repo %s not found in results", target.Repo)
				}
			}
		})
	}
}

func TestProcessCancelTarget_DryRun(t *testing.T) {
	// Set global dry run flag
	originalDryRun := globalFlags.DryRun
	globalFlags.DryRun = true
	defer func() { globalFlags.DryRun = originalDryRun }()

	// Create mock client
	mockClient := &gh.MockClient{}

	// Create target state with PR and sync branch
	target := &state.TargetState{
		Repo: "org/test-repo",
		OpenPRs: []gh.PR{
			{Number: 123, State: "open", Title: "Test PR"},
		},
		SyncBranches: []state.SyncBranch{
			{
				Name: "sync/test-branch",
				Metadata: &state.BranchMetadata{
					Timestamp: mockTime(),
				},
			},
		},
	}

	result := processCancelTarget(context.Background(), mockClient, target)

	// In dry run mode, no actual API calls should be made
	mockClient.AssertNotCalled(t, "ClosePR")
	mockClient.AssertNotCalled(t, "DeleteBranch")

	// But result should show what would happen
	assert.Equal(t, "org/test-repo", result.Repository)
	assert.NotNil(t, result.PRNumber)
	assert.Equal(t, 123, *result.PRNumber)
	assert.True(t, result.PRClosed)
	assert.Equal(t, "sync/test-branch", result.BranchName)
	assert.True(t, result.BranchDeleted)
	assert.Empty(t, result.Error)
}

func TestProcessCancelTarget_RealExecution(t *testing.T) {
	// Ensure dry run is off
	originalDryRun := globalFlags.DryRun
	globalFlags.DryRun = false
	defer func() { globalFlags.DryRun = originalDryRun }()

	// Create mock client
	mockClient := &gh.MockClient{}
	mockClient.On("ClosePR", mock.Anything, "org/test-repo", 123, mock.AnythingOfType("string")).Return(nil)
	mockClient.On("DeleteBranch", mock.Anything, "org/test-repo", "sync/test-branch").Return(nil)

	// Create target state
	target := &state.TargetState{
		Repo: "org/test-repo",
		OpenPRs: []gh.PR{
			{Number: 123, State: "open", Title: "Test PR"},
		},
		SyncBranches: []state.SyncBranch{
			{
				Name: "sync/test-branch",
				Metadata: &state.BranchMetadata{
					Timestamp: mockTime(),
				},
			},
		},
	}

	result := processCancelTarget(context.Background(), mockClient, target)

	// Verify API calls were made
	mockClient.AssertCalled(t, "ClosePR", mock.Anything, "org/test-repo", 123, mock.AnythingOfType("string"))
	mockClient.AssertCalled(t, "DeleteBranch", mock.Anything, "org/test-repo", "sync/test-branch")

	// Verify results
	assert.Equal(t, "org/test-repo", result.Repository)
	assert.NotNil(t, result.PRNumber)
	assert.Equal(t, 123, *result.PRNumber)
	assert.True(t, result.PRClosed)
	assert.Equal(t, "sync/test-branch", result.BranchName)
	assert.True(t, result.BranchDeleted)
	assert.Empty(t, result.Error)
}

func TestProcessCancelTarget_KeepBranches(t *testing.T) {
	// Set keep branches flag
	originalKeepBranches := cancelKeepBranches
	cancelKeepBranches = true
	defer func() { cancelKeepBranches = originalKeepBranches }()

	// Ensure dry run is off
	originalDryRun := globalFlags.DryRun
	globalFlags.DryRun = false
	defer func() { globalFlags.DryRun = originalDryRun }()

	// Create mock client
	mockClient := &gh.MockClient{}
	mockClient.On("ClosePR", mock.Anything, "org/test-repo", 123, mock.AnythingOfType("string")).Return(nil)

	// Create target state
	target := &state.TargetState{
		Repo: "org/test-repo",
		OpenPRs: []gh.PR{
			{Number: 123, State: "open", Title: "Test PR"},
		},
		SyncBranches: []state.SyncBranch{
			{
				Name: "sync/test-branch",
				Metadata: &state.BranchMetadata{
					Timestamp: mockTime(),
				},
			},
		},
	}

	result := processCancelTarget(context.Background(), mockClient, target)

	// Verify PR was closed but branch was not deleted
	mockClient.AssertCalled(t, "ClosePR", mock.Anything, "org/test-repo", 123, mock.AnythingOfType("string"))
	mockClient.AssertNotCalled(t, "DeleteBranch", mock.Anything, mock.Anything, mock.Anything)

	// Verify results
	assert.Equal(t, "org/test-repo", result.Repository)
	assert.NotNil(t, result.PRNumber)
	assert.True(t, result.PRClosed)
	assert.Equal(t, "sync/test-branch", result.BranchName)
	assert.False(t, result.BranchDeleted) // Should not be deleted when keeping branches
	assert.Empty(t, result.Error)
}

func TestProcessCancelTarget_APIError(t *testing.T) {
	// Ensure dry run is off
	originalDryRun := globalFlags.DryRun
	globalFlags.DryRun = false
	defer func() { globalFlags.DryRun = originalDryRun }()

	// Create mock client that returns error
	mockClient := &gh.MockClient{}
	mockClient.On("ClosePR", mock.Anything, "org/test-repo", 123, mock.AnythingOfType("string")).Return(assert.AnError)

	// Create target state
	target := &state.TargetState{
		Repo: "org/test-repo",
		OpenPRs: []gh.PR{
			{Number: 123, State: "open", Title: "Test PR"},
		},
	}

	result := processCancelTarget(context.Background(), mockClient, target)

	// Verify error is captured
	assert.Equal(t, "org/test-repo", result.Repository)
	if result.PRNumber != nil {
		assert.Equal(t, 123, *result.PRNumber)
	}
	assert.False(t, result.PRClosed)
	assert.Contains(t, result.Error, "failed to close PR #123")
}
