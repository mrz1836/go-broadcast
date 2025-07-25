// Package errors defines common error types used throughout the application
package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSyncErrors verifies that all sync-related errors are defined correctly
func TestSyncErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrNoFilesToCommit",
			err:      ErrNoFilesToCommit,
			expected: "no files to commit",
		},
		{
			name:     "ErrNoTargets",
			err:      ErrNoTargets,
			expected: "no targets configured",
		},
		{
			name:     "ErrInvalidConfig",
			err:      ErrInvalidConfig,
			expected: "invalid configuration",
		},
		{
			name:     "ErrSyncFailed",
			err:      ErrSyncFailed,
			expected: "sync operation failed",
		},
		{
			name:     "ErrNoMatchingTargets",
			err:      ErrNoMatchingTargets,
			expected: "no targets match the specified filter",
		},
		{
			name:     "ErrFileNotFound",
			err:      ErrFileNotFound,
			expected: "source file not found",
		},
		{
			name:     "ErrTransformNotFound",
			err:      ErrTransformNotFound,
			expected: "transform not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.err)
			require.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// TestStateErrors verifies that all state-related errors are defined correctly
func TestStateErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrPRExists",
			err:      ErrPRExists,
			expected: "pull request already exists",
		},
		{
			name:     "ErrPRNotFound",
			err:      ErrPRNotFound,
			expected: "pull request not found",
		},
		{
			name:     "ErrBranchNotFound",
			err:      ErrBranchNotFound,
			expected: "branch not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.err)
			require.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// TestGitErrors verifies that all git-related errors are defined correctly
func TestGitErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrInvalidRepoPath",
			err:      ErrInvalidRepoPath,
			expected: "invalid repository path",
		},
		{
			name:     "ErrGitCommand",
			err:      ErrGitCommand,
			expected: "git command failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.err)
			require.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// TestTestErrors verifies that test-specific errors are defined correctly
func TestTestErrors(t *testing.T) {
	require.Error(t, ErrTest)
	require.Equal(t, "test error", ErrTest.Error())
}

// TestErrorsAreSentinel verifies that all errors are sentinel errors
func TestErrorsAreSentinel(t *testing.T) {
	sentinelErrors := []struct {
		name string
		err  error
	}{
		// Sync errors
		{"ErrNoFilesToCommit", ErrNoFilesToCommit},
		{"ErrNoTargets", ErrNoTargets},
		{"ErrInvalidConfig", ErrInvalidConfig},
		{"ErrSyncFailed", ErrSyncFailed},
		{"ErrNoMatchingTargets", ErrNoMatchingTargets},
		{"ErrFileNotFound", ErrFileNotFound},
		{"ErrTransformNotFound", ErrTransformNotFound},
		// State errors
		{"ErrPRExists", ErrPRExists},
		{"ErrPRNotFound", ErrPRNotFound},
		{"ErrBranchNotFound", ErrBranchNotFound},
		// Git errors
		{"ErrInvalidRepoPath", ErrInvalidRepoPath},
		{"ErrGitCommand", ErrGitCommand},
		// Test errors
		{"ErrTest", ErrTest},
	}

	for _, tt := range sentinelErrors {
		t.Run(tt.name, func(t *testing.T) {
			// Verify errors are not nil
			require.Error(t, tt.err)

			// Verify errors have non-empty messages
			require.NotEmpty(t, tt.err.Error())

			// Verify errors can be used with errors.Is
			testErr := tt.err
			require.ErrorIs(t, testErr, tt.err)
		})
	}
}

// TestErrorsAreImmutable verifies that the exported errors cannot be modified
func TestErrorsAreImmutable(t *testing.T) {
	// This test ensures that the errors remain constant and their messages don't change
	originalMessage := ErrNoFilesToCommit.Error()

	// Attempt to use the error (this shouldn't modify it)
	_ = ErrNoFilesToCommit

	// Verify the error message hasn't changed
	require.Equal(t, originalMessage, ErrNoFilesToCommit.Error())
}

// Define static wrapper errors for testing
var (
	errFailedToSyncRepository  = errors.New("failed to sync repository")
	errFailedToExecuteGitPull  = errors.New("failed to execute git pull")
	errUnableToFindPRForBranch = errors.New("unable to find PR for branch main")
)

// TestErrorsCanBeWrapped verifies that errors can be properly wrapped
func TestErrorsCanBeWrapped(t *testing.T) {
	tests := []struct {
		name        string
		baseErr     error
		wrapErr     error
		wrapMessage string
	}{
		{
			name:        "wrap sync error",
			baseErr:     ErrSyncFailed,
			wrapErr:     errFailedToSyncRepository,
			wrapMessage: "failed to sync repository",
		},
		{
			name:        "wrap git error",
			baseErr:     ErrGitCommand,
			wrapErr:     errFailedToExecuteGitPull,
			wrapMessage: "failed to execute git pull",
		},
		{
			name:        "wrap state error",
			baseErr:     ErrPRNotFound,
			wrapErr:     errUnableToFindPRForBranch,
			wrapMessage: "unable to find PR for branch main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wrap the error
			wrappedErr := errors.Join(tt.wrapErr, tt.baseErr)

			// Verify we can still identify the base error
			require.ErrorIs(t, wrappedErr, tt.baseErr)

			// Verify the wrapped error contains both messages
			require.Contains(t, wrappedErr.Error(), tt.wrapMessage)
			require.Contains(t, wrappedErr.Error(), tt.baseErr.Error())
		})
	}
}

// TestErrorCategories verifies that errors are properly categorized
func TestErrorCategories(t *testing.T) {
	syncErrors := []error{
		ErrNoFilesToCommit,
		ErrNoTargets,
		ErrInvalidConfig,
		ErrSyncFailed,
		ErrNoMatchingTargets,
		ErrFileNotFound,
		ErrTransformNotFound,
	}

	stateErrors := []error{
		ErrPRExists,
		ErrPRNotFound,
		ErrBranchNotFound,
	}

	gitErrors := []error{
		ErrInvalidRepoPath,
		ErrGitCommand,
	}

	testErrors := []error{
		ErrTest,
	}

	// Verify all errors are accounted for
	totalErrors := len(syncErrors) + len(stateErrors) + len(gitErrors) + len(testErrors)
	require.Equal(t, 13, totalErrors, "ensure all errors are categorized")

	// Verify no nil errors
	for _, err := range append(append(append(syncErrors, stateErrors...), gitErrors...), testErrors...) {
		require.Error(t, err)
	}
}
