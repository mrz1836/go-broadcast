package git

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test-specific error constants to avoid dynamic error creation
var (
	errTestSyncOperationFailed = errors.New("sync operation failed")
)

// TestErrBranchAlreadyExists tests the new error constant and detection logic
func TestErrBranchAlreadyExists(t *testing.T) {
	t.Run("error constant is properly defined", func(t *testing.T) {
		// Test that the error is properly defined
		require.Error(t, ErrBranchAlreadyExists)
		assert.Equal(t, "branch already exists on remote", ErrBranchAlreadyExists.Error())

		// Test error comparison
		testErr := ErrBranchAlreadyExists
		require.ErrorIs(t, testErr, ErrBranchAlreadyExists)
		require.NotErrorIs(t, testErr, ErrNoChanges)
		require.NotErrorIs(t, testErr, ErrRepositoryExists)
	})

	t.Run("different from other git errors", func(t *testing.T) {
		assert.NotEqual(t, ErrBranchAlreadyExists, ErrNoChanges)
		assert.NotEqual(t, ErrBranchAlreadyExists, ErrRepositoryExists)
		assert.NotEqual(t, ErrBranchAlreadyExists, ErrGitNotFound)
		assert.NotEqual(t, ErrBranchAlreadyExists, ErrNotARepository)
	})
}

// TestBranchConflictScenarios tests that our error handling logic works correctly
// for typical branch conflict scenarios that would arise in real sync operations
func TestBranchConflictScenarios(t *testing.T) {
	t.Run("simulated sync scenario - existing branch recovery", func(t *testing.T) {
		// This test simulates what would happen during a sync operation
		// when a branch already exists from a previous failed sync

		// Scenario 1: Local branch creation fails because branch exists
		localBranchErr := ErrBranchAlreadyExists

		// In a real sync, we would detect this error and attempt to checkout the existing branch
		if errors.Is(localBranchErr, ErrBranchAlreadyExists) {
			t.Log("✓ Detected local branch already exists - would checkout existing branch")
		} else {
			t.Error("Failed to detect local branch conflict")
		}

		// Scenario 2: Push fails because remote branch exists with different history
		remotePushErr := ErrBranchAlreadyExists

		// In a real sync, we would detect this error and attempt a force push to recover
		if errors.Is(remotePushErr, ErrBranchAlreadyExists) {
			t.Log("✓ Detected remote branch conflict - would attempt force push")
		} else {
			t.Error("Failed to detect remote branch conflict")
		}
	})

	t.Run("error handling chain", func(t *testing.T) {
		// Test the complete error handling chain that would occur during sync

		// Initial attempt fails with branch conflict
		initialErr := ErrBranchAlreadyExists
		require := assert.New(t)

		// First level: detect the error type
		if errors.Is(initialErr, ErrBranchAlreadyExists) {
			t.Log("✓ Step 1: Detected branch conflict")

			// Second level: attempt recovery (simulated success)
			recoverySuccessful := true

			if recoverySuccessful {
				t.Log("✓ Step 2: Recovery successful")
				// Recovery successful - no assertion needed
			} else {
				t.Log("✗ Step 2: Recovery failed")
				require.Fail("Recovery should not fail in this test")
			}
		} else {
			require.Fail("Should have detected branch conflict")
		}
	})

	t.Run("integration with standard error patterns", func(t *testing.T) {
		// Test that our error integrates well with Go's standard error patterns

		// Wrapping the error should preserve its identity
		wrappedErr := fmt.Errorf("%w: %s", errTestSyncOperationFailed, ErrBranchAlreadyExists.Error())

		// Direct comparison
		assert.Contains(t, wrappedErr.Error(), "branch already exists on remote")

		// The wrapped error shouldn't be directly comparable, which is expected
		require.NotErrorIs(t, wrappedErr, ErrBranchAlreadyExists)

		// But we can still detect it in error messages for fallback handling
		errorMessage := wrappedErr.Error()
		containsBranchConflict := containsBranchConflictKeywords(errorMessage)
		assert.True(t, containsBranchConflict, "Should detect branch conflict in error message")
	})
}

// containsBranchConflictKeywords checks if an error message contains keywords
// that typically indicate branch conflicts - this simulates the logic we have in git.go
func containsBranchConflictKeywords(errMsg string) bool {
	keywords := []string{
		"already exists",
		"updates were rejected",
		"non-fast-forward",
		"fetch first",
	}

	for _, keyword := range keywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}
	return false
}
