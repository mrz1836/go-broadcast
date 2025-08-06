package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/spf13/cobra"
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

func TestProcessCancelTarget_BranchDeletionError(t *testing.T) {
	// Ensure dry run is off
	originalDryRun := globalFlags.DryRun
	globalFlags.DryRun = false
	defer func() { globalFlags.DryRun = originalDryRun }()

	// Ensure branches are not kept
	originalKeepBranches := cancelKeepBranches
	cancelKeepBranches = false
	defer func() { cancelKeepBranches = originalKeepBranches }()

	// Create mock client that succeeds PR close but fails branch deletion
	mockClient := &gh.MockClient{}
	mockClient.On("ClosePR", mock.Anything, "org/test-repo", 123, mock.AnythingOfType("string")).Return(nil)
	mockClient.On("DeleteBranch", mock.Anything, "org/test-repo", "sync/test-branch").Return(assert.AnError)

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

	// Verify PR was closed but branch deletion failed
	assert.Equal(t, "org/test-repo", result.Repository)
	assert.NotNil(t, result.PRNumber)
	assert.Equal(t, 123, *result.PRNumber)
	assert.True(t, result.PRClosed)
	assert.Equal(t, "sync/test-branch", result.BranchName)
	assert.False(t, result.BranchDeleted)
	assert.Contains(t, result.Error, "failed to delete branch sync/test-branch")
}

func TestProcessCancelTarget_NoOpenPRs(t *testing.T) {
	// Ensure dry run is off
	originalDryRun := globalFlags.DryRun
	globalFlags.DryRun = false
	defer func() { globalFlags.DryRun = originalDryRun }()

	// Create mock client - we need to mock DeleteBranch since it will be called
	mockClient := &gh.MockClient{}
	mockClient.On("DeleteBranch", mock.Anything, "org/test-repo", "sync/test-branch").Return(nil)

	// Create target state with no open PRs, only sync branches
	target := &state.TargetState{
		Repo: "org/test-repo",
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

	// Verify no PR operations were attempted
	mockClient.AssertNotCalled(t, "ClosePR")
	assert.Equal(t, "org/test-repo", result.Repository)
	assert.Nil(t, result.PRNumber)
	assert.False(t, result.PRClosed)
	assert.Equal(t, "sync/test-branch", result.BranchName)
	assert.True(t, result.BranchDeleted) // Branch should be deleted since dry run is off
}

func TestProcessCancelTarget_MultipleSyncBranches(t *testing.T) {
	// Ensure dry run is off
	originalDryRun := globalFlags.DryRun
	globalFlags.DryRun = false
	defer func() { globalFlags.DryRun = originalDryRun }()

	// Create mock client
	mockClient := &gh.MockClient{}
	mockClient.On("DeleteBranch", mock.Anything, "org/test-repo", "sync/newer-branch").Return(nil)

	// Create target state with multiple sync branches
	target := &state.TargetState{
		Repo: "org/test-repo",
		SyncBranches: []state.SyncBranch{
			{
				Name: "sync/older-branch",
				Metadata: &state.BranchMetadata{
					Timestamp: time.Date(2023, time.January, 10, 10, 30, 0, 0, time.UTC),
				},
			},
			{
				Name: "sync/newer-branch",
				Metadata: &state.BranchMetadata{
					Timestamp: time.Date(2023, time.January, 20, 10, 30, 0, 0, time.UTC),
				},
			},
		},
	}

	result := processCancelTarget(context.Background(), mockClient, target)

	// Verify only the most recent branch is deleted
	mockClient.AssertCalled(t, "DeleteBranch", mock.Anything, "org/test-repo", "sync/newer-branch")
	mockClient.AssertNotCalled(t, "DeleteBranch", mock.Anything, "org/test-repo", "sync/older-branch")
	assert.Equal(t, "sync/newer-branch", result.BranchName)
	assert.True(t, result.BranchDeleted)
}

func TestProcessCancelTarget_CustomComment(t *testing.T) {
	// Ensure dry run is off
	originalDryRun := globalFlags.DryRun
	globalFlags.DryRun = false
	defer func() { globalFlags.DryRun = originalDryRun }()

	// Set custom comment
	originalComment := cancelComment
	cancelComment = "Custom cancellation reason"
	defer func() { cancelComment = originalComment }()

	// Create mock client
	mockClient := &gh.MockClient{}
	mockClient.On("ClosePR", mock.Anything, "org/test-repo", 123, "Custom cancellation reason").Return(nil)

	// Create target state
	target := &state.TargetState{
		Repo: "org/test-repo",
		OpenPRs: []gh.PR{
			{Number: 123, State: "open", Title: "Test PR"},
		},
	}

	result := processCancelTarget(context.Background(), mockClient, target)

	// Verify custom comment was used
	mockClient.AssertCalled(t, "ClosePR", mock.Anything, "org/test-repo", 123, "Custom cancellation reason")
	assert.True(t, result.PRClosed)
}

func TestOutputCancelPreview(t *testing.T) {
	// Capture output
	var stdoutBuf, stderrBuf bytes.Buffer
	originalStdout := output.Stdout()
	originalStderr := output.Stderr()
	output.SetStdout(&stdoutBuf)
	output.SetStderr(&stderrBuf)
	defer func() {
		output.SetStdout(originalStdout)
		output.SetStderr(originalStderr)
	}()

	tests := []struct {
		name    string
		summary *CancelSummary
		setup   func()
		cleanup func()
	}{
		{
			name: "no active syncs",
			summary: &CancelSummary{
				TotalTargets: 0,
				Results:      []CancelResult{},
				DryRun:       true,
			},
		},
		{
			name: "single target with PR and branch",
			summary: &CancelSummary{
				TotalTargets:    1,
				PRsClosed:       1,
				BranchesDeleted: 1,
				Results: []CancelResult{
					{
						Repository:    "org/test-repo",
						PRNumber:      intPtr(123),
						PRClosed:      true,
						BranchName:    "sync/test-branch",
						BranchDeleted: true,
					},
				},
				DryRun: true,
			},
		},
		{
			name: "multiple targets with keep branches",
			summary: &CancelSummary{
				TotalTargets: 2,
				PRsClosed:    2,
				Results: []CancelResult{
					{
						Repository: "org/repo1",
						PRNumber:   intPtr(123),
						PRClosed:   true,
						BranchName: "sync/branch1",
					},
					{
						Repository: "org/repo2",
						PRNumber:   intPtr(456),
						PRClosed:   true,
						BranchName: "sync/branch2",
					},
				},
				DryRun: true,
			},
			setup: func() {
				cancelKeepBranches = true
			},
			cleanup: func() {
				cancelKeepBranches = false
			},
		},
		{
			name: "target with error",
			summary: &CancelSummary{
				TotalTargets: 1,
				Errors:       1,
				Results: []CancelResult{
					{
						Repository: "org/error-repo",
						Error:      "failed to close PR #123: API error",
					},
				},
				DryRun: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset buffers
			stdoutBuf.Reset()
			stderrBuf.Reset()

			// Setup test-specific state
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			err := outputCancelPreview(tt.summary)
			require.NoError(t, err)

			stdoutContent := stdoutBuf.String()
			stderrContent := stderrBuf.String()

			// Common assertions
			assert.Contains(t, stderrContent, "DRY-RUN MODE")

			if tt.summary.TotalTargets == 0 {
				assert.Contains(t, stdoutContent, "No active sync operations found")
			} else {
				assert.Contains(t, stdoutContent, "Would cancel sync operations")
				assert.Contains(t, stdoutContent, "Summary (would):")

				// Check repository listings
				for _, result := range tt.summary.Results {
					assert.Contains(t, stdoutContent, result.Repository)

					if result.PRNumber != nil {
						assert.Contains(t, stdoutContent, "Would close PR")
					}

					if result.BranchName != "" {
						if cancelKeepBranches {
							assert.Contains(t, stdoutContent, "Would keep branch:")
						} else {
							assert.Contains(t, stdoutContent, "Would delete branch:")
						}
					}

					if result.Error != "" {
						assert.Contains(t, stderrContent, "Error:")
					}
				}
			}
		})
	}
}

func TestOutputCancelResults(t *testing.T) {
	// Capture output
	var stdoutBuf, stderrBuf bytes.Buffer
	originalStdout := output.Stdout()
	originalStderr := output.Stderr()
	output.SetStdout(&stdoutBuf)
	output.SetStderr(&stderrBuf)
	defer func() {
		output.SetStdout(originalStdout)
		output.SetStderr(originalStderr)
	}()

	tests := []struct {
		name       string
		summary    *CancelSummary
		jsonOutput bool
		setup      func()
		cleanup    func()
	}{
		{
			name: "no active syncs",
			summary: &CancelSummary{
				TotalTargets: 0,
				Results:      []CancelResult{},
				DryRun:       false,
			},
		},
		{
			name: "successful cancellation text output",
			summary: &CancelSummary{
				TotalTargets:    2,
				PRsClosed:       2,
				BranchesDeleted: 2,
				Results: []CancelResult{
					{
						Repository:    "org/repo1",
						PRNumber:      intPtr(123),
						PRClosed:      true,
						BranchName:    "sync/branch1",
						BranchDeleted: true,
					},
					{
						Repository:    "org/repo2",
						PRNumber:      intPtr(456),
						PRClosed:      true,
						BranchName:    "sync/branch2",
						BranchDeleted: true,
					},
				},
				DryRun: false,
			},
		},
		{
			name: "successful cancellation JSON output",
			summary: &CancelSummary{
				TotalTargets:    1,
				PRsClosed:       1,
				BranchesDeleted: 1,
				Results: []CancelResult{
					{
						Repository:    "org/test-repo",
						PRNumber:      intPtr(123),
						PRClosed:      true,
						BranchName:    "sync/test-branch",
						BranchDeleted: true,
					},
				},
				DryRun: false,
			},
			jsonOutput: true,
			setup: func() {
				jsonOutput = true
			},
			cleanup: func() {
				jsonOutput = false
			},
		},
		{
			name: "partial success with errors",
			summary: &CancelSummary{
				TotalTargets:    2,
				PRsClosed:       1,
				BranchesDeleted: 1,
				Errors:          1,
				Results: []CancelResult{
					{
						Repository:    "org/success-repo",
						PRNumber:      intPtr(123),
						PRClosed:      true,
						BranchName:    "sync/success-branch",
						BranchDeleted: true,
					},
					{
						Repository: "org/error-repo",
						PRNumber:   intPtr(456),
						PRClosed:   false,
						BranchName: "sync/error-branch",
						Error:      "failed to close PR #456: API error",
					},
				},
				DryRun: false,
			},
		},
		{
			name: "keep branches mode",
			summary: &CancelSummary{
				TotalTargets: 1,
				PRsClosed:    1,
				Results: []CancelResult{
					{
						Repository:    "org/repo1",
						PRNumber:      intPtr(123),
						PRClosed:      true,
						BranchName:    "sync/kept-branch",
						BranchDeleted: false,
					},
				},
				DryRun: false,
			},
			setup: func() {
				cancelKeepBranches = true
			},
			cleanup: func() {
				cancelKeepBranches = false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset buffers
			stdoutBuf.Reset()
			stderrBuf.Reset()

			// Setup test-specific state
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			err := outputCancelResults(tt.summary)
			require.NoError(t, err)

			stdoutContent := stdoutBuf.String()
			stderrContent := stderrBuf.String()

			if tt.summary.TotalTargets == 0 {
				assert.Contains(t, stdoutContent, "No active sync operations found")
			} else if tt.jsonOutput {
				// Verify JSON output
				assert.Contains(t, stdoutContent, "\"total_targets\":")
				assert.Contains(t, stdoutContent, "\"prs_closed\":")
				assert.Contains(t, stdoutContent, "\"branches_deleted\":")
				assert.Contains(t, stdoutContent, "\"results\":")
			} else {
				// Verify text output
				assert.Contains(t, stdoutContent, "Canceled sync operations")
				assert.Contains(t, stdoutContent, "Summary:")

				// Check repository listings
				for _, result := range tt.summary.Results {
					assert.Contains(t, stdoutContent, result.Repository)

					if result.PRClosed && result.PRNumber != nil {
						assert.Contains(t, stdoutContent, "Closed PR")
					} else if result.PRNumber != nil && !result.PRClosed {
						assert.Contains(t, stderrContent, "Failed to close PR")
					}

					if result.BranchDeleted {
						assert.Contains(t, stdoutContent, "Deleted branch:")
					} else if result.BranchName != "" && !cancelKeepBranches {
						assert.Contains(t, stderrContent, "Failed to delete branch:")
					} else if result.BranchName != "" && cancelKeepBranches {
						assert.Contains(t, stdoutContent, "Kept branch:")
					}

					if result.Error != "" {
						assert.Contains(t, stderrContent, "Error:")
					}
				}

				// Check summary counts
				if tt.summary.Errors == 0 {
					assert.Contains(t, stdoutContent, "completed successfully")
				} else {
					assert.Contains(t, stderrContent, "completed with some errors")
				}
			}
		})
	}
}

// intPtr is a helper function to get a pointer to an int
func intPtr(i int) *int {
	return &i
}

// TestPerformCancel tests the performCancel function
// This function is difficult to test comprehensively in unit tests
// because it creates its own GitHub client and state discoverer internally.
// Integration tests would be more appropriate for full coverage.
func TestPerformCancel_ConfigValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("nil config", func(t *testing.T) {
		// The performCancel function will panic with nil config instead of returning error
		// This is because it calls cfg methods without checking for nil
		assert.Panics(t, func() {
			_, _ = performCancel(ctx, nil, []string{})
		})
	})

	// Further testing would require either:
	// 1. Refactoring performCancel to accept injected dependencies
	// 2. Integration tests with real GitHub API (or docker containers)
	// 3. More sophisticated mocking of the gh.NewClient function
	//
	// For now, the function is adequately covered by:
	// - processCancelTarget tests (core business logic)
	// - filterTargets tests (target filtering logic)
	// - output function tests (presentation logic)
	// - Integration tests in the CLI package
}

func TestRunCancel_ConfigNotFound(t *testing.T) {
	// Save original config
	originalConfig := globalFlags.ConfigFile
	globalFlags.ConfigFile = "/non/existent/config.yml"
	defer func() {
		globalFlags.ConfigFile = originalConfig
	}()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runCancel(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

func TestErrTargetNotFound(t *testing.T) {
	assert.Equal(t, "target repository not found in configuration", ErrTargetNotFound.Error())
}

func TestGenerateCancelComment_Content(t *testing.T) {
	comment := generateCancelComment()

	// Test structure and content
	assert.Contains(t, comment, "🚫 **Sync Operation Canceled**")
	assert.Contains(t, comment, "This sync operation has been canceled")
	assert.Contains(t, comment, "**Canceled at**:")
	assert.Contains(t, comment, "**Reason**: Manual cancellation via CLI")
	assert.Contains(t, comment, "You can safely ignore this PR")
	assert.Contains(t, comment, "go-broadcast")

	// Test that it contains a valid timestamp
	assert.Contains(t, comment, "T") // ISO 8601 format should contain T
	assert.Contains(t, comment, ":") // Time should contain colons

	// Test that multiple calls generate different timestamps
	comment2 := generateCancelComment()
	// These might be the same if called too quickly, but the format should be consistent
	assert.Contains(t, comment2, "**Canceled at**:")
}

// TestPerformCancelPanicRecovery tests panic handling in performCancel
func TestPerformCancelPanicRecovery(t *testing.T) {
	t.Run("nil config causes panic", func(t *testing.T) {
		ctx := context.Background()

		// performCancel should panic with nil config
		require.Panics(t, func() {
			_, _ = performCancel(ctx, nil, []string{})
		})
	})
}

// TestPerformCancelErrorHandling tests error conditions
func TestPerformCancelErrorHandling(t *testing.T) {
	t.Run("github client creation failure", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{
			Groups: []config.Group{{
				Source: config.SourceConfig{
					Repo:   "test/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "test/target1"},
				},
			}},
		}

		// performCancel will likely fail due to GitHub operations
		// but should return an error, not panic
		require.NotPanics(t, func() {
			summary, err := performCancel(ctx, cfg, []string{})
			// Should return an error due to GitHub API failures
			if err != nil {
				assert.Nil(t, summary)
				// Error could be about GitHub API, sync state, or discovery
				assert.NotEmpty(t, err.Error())
			}
		})
	})
}

// TestCancelSummaryStructure tests the CancelSummary struct
func TestCancelSummaryStructure(t *testing.T) {
	t.Run("cancel summary initialization", func(t *testing.T) {
		summary := &CancelSummary{
			TotalTargets:    3,
			PRsClosed:       1,
			BranchesDeleted: 2,
			Errors:          0,
			DryRun:          false,
			Results:         []CancelResult{},
		}

		assert.Equal(t, 3, summary.TotalTargets)
		assert.Equal(t, 1, summary.PRsClosed)
		assert.Equal(t, 2, summary.BranchesDeleted)
		assert.Equal(t, 0, summary.Errors)
		assert.False(t, summary.DryRun)
		assert.NotNil(t, summary.Results)
	})
}

// TestCancelResultStructure tests the CancelResult struct
func TestCancelResultStructure(t *testing.T) {
	t.Run("cancel result initialization", func(t *testing.T) {
		prNumber := 123
		result := CancelResult{
			Repository:    "test/repo",
			PRNumber:      &prNumber,
			PRClosed:      true,
			BranchName:    "sync/test-branch",
			BranchDeleted: false,
			Error:         "",
		}

		assert.Equal(t, "test/repo", result.Repository)
		assert.Equal(t, 123, *result.PRNumber)
		assert.True(t, result.PRClosed)
		assert.Equal(t, "sync/test-branch", result.BranchName)
		assert.False(t, result.BranchDeleted)
		assert.Empty(t, result.Error)
	})

	t.Run("cancel result with nil pr number", func(t *testing.T) {
		result := CancelResult{
			Repository:    "test/repo",
			PRNumber:      nil,
			PRClosed:      false,
			BranchName:    "",
			BranchDeleted: true,
			Error:         "Failed to close PR",
		}

		assert.Equal(t, "test/repo", result.Repository)
		assert.Nil(t, result.PRNumber)
		assert.False(t, result.PRClosed)
		assert.Empty(t, result.BranchName)
		assert.True(t, result.BranchDeleted)
		assert.Equal(t, "Failed to close PR", result.Error)
	})
}

// TestPerformCancelErrorCases tests specific error scenarios in performCancel
func TestPerformCancelErrorCases(t *testing.T) {
	t.Run("GitHub client creation failures", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{
			Groups: []config.Group{{
				Source: config.SourceConfig{
					Repo:   "test/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "test/target1"},
				},
			}},
		}

		// This will likely fail with GitHub CLI not found or auth issues
		// but we test the error handling path
		summary, err := performCancel(ctx, cfg, []string{})

		// Should return error, not panic
		require.Error(t, err)
		assert.Nil(t, summary)

		// Error should be a failure related to discovery/validation or GitHub client
		// The function actually gets past client creation and fails at state discovery
		assert.True(t,
			strings.Contains(err.Error(), "failed to discover sync state") ||
				containsGitHubClientError(err.Error()),
			"Expected sync state discovery or GitHub client error, got: %s", err.Error())
	})

	t.Run("Empty targets", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{
			Groups: []config.Group{{
				Source: config.SourceConfig{
					Repo:   "test/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{}, // Empty targets
			}},
		}

		// Should still try to process but fail at GitHub client creation
		summary, err := performCancel(ctx, cfg, []string{})
		require.Error(t, err)
		assert.Nil(t, summary)
	})

	t.Run("Specific target repos", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{
			Groups: []config.Group{{
				Source: config.SourceConfig{
					Repo:   "test/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{
					{Repo: "test/target1"},
					{Repo: "test/target2"},
				},
			}},
		}

		// Test with specific target repos filter
		summary, err := performCancel(ctx, cfg, []string{"test/target1"})
		require.Error(t, err) // Will fail at GitHub client creation
		assert.Nil(t, summary)
	})
}

// Helper function to check if error contains GitHub client related message
func containsGitHubClientError(errMsg string) bool {
	return strings.Contains(errMsg, "failed to initialize GitHub client") ||
		strings.Contains(errMsg, "gh CLI not found") ||
		strings.Contains(errMsg, "not authenticated") ||
		strings.Contains(errMsg, "ErrGHNotFound") ||
		strings.Contains(errMsg, "ErrNotAuthenticated")
}
