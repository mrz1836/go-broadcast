package errors //nolint:revive,nolintlint // internal test package, name conflict intentional

import (
	"errors"
	"sync"
	"testing"
)

// TestConcurrentErrorCreation verifies that all error creation functions
// are safe for concurrent use from multiple goroutines.
// Run with: go test -race ./internal/errors/...
func TestConcurrentErrorCreation(t *testing.T) {
	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 10)

	baseErr := errors.New("base error") //nolint:err113 // test-only error for race testing

	for i := 0; i < goroutines; i++ {
		// Test WrapWithContext
		go func() {
			defer wg.Done()
			err := WrapWithContext(baseErr, "concurrent operation")
			if err == nil {
				t.Error("expected non-nil error")
			}
		}()

		// Test InvalidFieldError
		go func() {
			defer wg.Done()
			err := InvalidFieldError("field", "value")
			if err == nil {
				t.Error("expected non-nil error")
			}
		}()

		// Test CommandFailedError
		go func() {
			defer wg.Done()
			err := CommandFailedError("test command", baseErr)
			if err == nil {
				t.Error("expected non-nil error")
			}
		}()

		// Test ValidationError
		go func() {
			defer wg.Done()
			err := ValidationError("item", "reason")
			if err == nil {
				t.Error("expected non-nil error")
			}
		}()

		// Test GitOperationError
		go func() {
			defer wg.Done()
			err := GitOperationError("clone", "repo", baseErr)
			if err == nil {
				t.Error("expected non-nil error")
			}
		}()

		// Test GitHubAPIError
		go func() {
			defer wg.Done()
			err := GitHubAPIError("create", "resource", baseErr)
			if err == nil {
				t.Error("expected non-nil error")
			}
		}()

		// Test FileOperationError
		go func() {
			defer wg.Done()
			err := FileOperationError("read", "/path", baseErr)
			if err == nil {
				t.Error("expected non-nil error")
			}
		}()

		// Test BatchOperationError
		go func() {
			defer wg.Done()
			err := BatchOperationError("process", 0, 10, baseErr)
			if err == nil {
				t.Error("expected non-nil error")
			}
		}()

		// Test APIResponseError
		go func() {
			defer wg.Done()
			err := APIResponseError(404, "not found")
			if err == nil {
				t.Error("expected non-nil error")
			}
		}()

		// Test reading sentinel error messages
		go func() {
			defer wg.Done()
			_ = ErrNoFilesToCommit.Error()
			_ = ErrSyncFailed.Error()
			_ = ErrPRNotFound.Error()
			_ = ErrGitCommand.Error()
		}()
	}

	wg.Wait()
}

// TestConcurrentSentinelErrorAccess verifies that sentinel errors
// can be safely accessed concurrently for identity checks.
func TestConcurrentSentinelErrorAccess(t *testing.T) {
	t.Parallel()
	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 4)

	sentinels := []error{
		ErrNoFilesToCommit,
		ErrNoChangesToSync,
		ErrNoTargets,
		ErrInvalidConfig,
		ErrSyncFailed,
		ErrNoMatchingTargets,
		ErrFileNotFound,
		ErrTransformNotFound,
		ErrPRExists,
		ErrPRNotFound,
		ErrBranchNotFound,
		ErrInvalidRepoPath,
		ErrGitCommand,
	}

	for i := 0; i < goroutines; i++ {
		// Read error messages
		go func() {
			defer wg.Done()
			for _, sentinel := range sentinels {
				_ = sentinel.Error()
			}
		}()

		// Perform errors.Is checks
		go func() {
			defer wg.Done()
			for _, sentinel := range sentinels {
				_ = errors.Is(sentinel, sentinel)
			}
		}()

		// Wrap and check
		go func() {
			defer wg.Done()
			for _, sentinel := range sentinels {
				wrapped := WrapWithContext(sentinel, "test")
				_ = errors.Is(wrapped, sentinel)
			}
		}()

		// Create and check multiple error types
		go func() {
			defer wg.Done()
			baseErr := errors.New("base") //nolint:err113 // test-only error for race testing
			_ = GitCloneError("repo", baseErr)
			_ = FileReadError("/path", baseErr)
			_ = JSONMarshalError("context", baseErr)
			_ = DirectoryCreateError("/dir", baseErr)
		}()
	}

	wg.Wait()
}

// TestConcurrentErrorWrapping verifies that error wrapping
// produces consistent results under concurrent access.
func TestConcurrentErrorWrapping(t *testing.T) {
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	baseErr := errors.New("base error") //nolint:err113 // test-only error for race testing

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// Create a chain of wrapped errors
			wrapped := baseErr
			for j := 0; j < 5; j++ {
				wrapped = WrapWithContext(wrapped, "level")
			}
			// Verify the original error is still findable
			if !errors.Is(wrapped, baseErr) {
				t.Error("errors.Is failed to find base error in chain")
			}
		}()
	}

	wg.Wait()
}
