package git

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// Define test errors as static wrapped errors
var (
	errCloneFailed     = errors.New("clone failed")
	errCheckoutFailed  = errors.New("checkout failed")
	errBranchFailed    = errors.New("branch creation failed")
	errAddFailed       = errors.New("add failed")
	errNothingToCommit = errors.New("nothing to commit")
	errPushFailed      = errors.New("push failed")
	errDiffFailed      = errors.New("diff failed")
	errTestError       = errors.New("test error")
	errNotGitRepo      = errors.New("not a git repository")
	errRemoteNotFound  = errors.New("remote not found")
)

// TestMockClientImplementation tests the MockClient implementation
func TestMockClientImplementation(t *testing.T) {
	ctx := context.Background()

	t.Run("Clone", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Clone", ctx, "https://github.com/test/repo.git", "/tmp/repo").Return(nil)

			err := mock.Clone(ctx, "https://github.com/test/repo.git", "/tmp/repo")
			require.NoError(t, err)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Clone", ctx, "https://github.com/test/repo.git", "/tmp/repo").Return(errCloneFailed)

			err := mock.Clone(ctx, "https://github.com/test/repo.git", "/tmp/repo")
			require.Error(t, err)
			require.Equal(t, errCloneFailed, err)
			mock.AssertExpectations(t)
		})

		t.Run("improperly configured mock", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Clone", ctx, "url", "path").Return()

			err := mock.Clone(ctx, "url", "path")
			require.Error(t, err)
			require.Contains(t, err.Error(), "mock not properly configured")
		})

		t.Run("mock returns non-error type", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Clone", ctx, "url", "path").Return("not an error")

			err := mock.Clone(ctx, "url", "path")
			require.Error(t, err)
			require.Contains(t, err.Error(), "mock returned non-error type")
		})
	})

	t.Run("Checkout", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Checkout", ctx, "/tmp/repo", "master").Return(nil)

			err := mock.Checkout(ctx, "/tmp/repo", "master")
			require.NoError(t, err)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Checkout", ctx, "/tmp/repo", "master").Return(errCheckoutFailed)

			err := mock.Checkout(ctx, "/tmp/repo", "master")
			require.Error(t, err)
			require.Equal(t, errCheckoutFailed, err)
			mock.AssertExpectations(t)
		})
	})

	t.Run("CreateBranch", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("CreateBranch", ctx, "/tmp/repo", "feature-branch").Return(nil)

			err := mock.CreateBranch(ctx, "/tmp/repo", "feature-branch")
			require.NoError(t, err)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("CreateBranch", ctx, "/tmp/repo", "feature-branch").Return(errBranchFailed)

			err := mock.CreateBranch(ctx, "/tmp/repo", "feature-branch")
			require.Error(t, err)
			require.Equal(t, errBranchFailed, err)
			mock.AssertExpectations(t)
		})
	})

	t.Run("Add", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockClient{}
			paths := []string{"file1.txt", "file2.txt"}
			mock.On("Add", ctx, "/tmp/repo", paths).Return(nil)

			err := mock.Add(ctx, "/tmp/repo", paths...)
			require.NoError(t, err)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockClient{}
			paths := []string{"file1.txt"}
			mock.On("Add", ctx, "/tmp/repo", paths).Return(errAddFailed)

			err := mock.Add(ctx, "/tmp/repo", paths...)
			require.Error(t, err)
			require.Equal(t, errAddFailed, err)
			mock.AssertExpectations(t)
		})
	})

	t.Run("Commit", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Commit", ctx, "/tmp/repo", "Test commit").Return(nil)

			err := mock.Commit(ctx, "/tmp/repo", "Test commit")
			require.NoError(t, err)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Commit", ctx, "/tmp/repo", "Test commit").Return(errNothingToCommit)

			err := mock.Commit(ctx, "/tmp/repo", "Test commit")
			require.Error(t, err)
			require.Equal(t, errNothingToCommit, err)
			mock.AssertExpectations(t)
		})
	})

	t.Run("Push", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Push", ctx, "/tmp/repo", "origin", "master", false).Return(nil)

			err := mock.Push(ctx, "/tmp/repo", "origin", "master", false)
			require.NoError(t, err)
			mock.AssertExpectations(t)
		})

		t.Run("error case with force", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Push", ctx, "/tmp/repo", "origin", "master", true).Return(errPushFailed)

			err := mock.Push(ctx, "/tmp/repo", "origin", "master", true)
			require.Error(t, err)
			require.Equal(t, errPushFailed, err)
			mock.AssertExpectations(t)
		})
	})

	t.Run("Diff", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockClient{}
			expectedDiff := "diff --git a/file.txt b/file.txt\n+added line"
			mock.On("Diff", ctx, "/tmp/repo", false).Return(expectedDiff, nil)

			diff, err := mock.Diff(ctx, "/tmp/repo", false)
			require.NoError(t, err)
			require.Equal(t, expectedDiff, diff)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Diff", ctx, "/tmp/repo", true).Return("", errDiffFailed)

			diff, err := mock.Diff(ctx, "/tmp/repo", true)
			require.Error(t, err)
			require.Equal(t, errDiffFailed, err)
			require.Empty(t, diff)
			mock.AssertExpectations(t)
		})

		t.Run("improperly configured mock - single argument", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Diff", ctx, "/tmp/repo", false).Return(errTestError)

			diff, err := mock.Diff(ctx, "/tmp/repo", false)
			require.Error(t, err)
			require.Equal(t, errTestError, err)
			require.Empty(t, diff)
		})

		t.Run("improperly configured mock - no arguments", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("Diff", ctx, "/tmp/repo", false).Return()

			diff, err := mock.Diff(ctx, "/tmp/repo", false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "mock not properly configured")
			require.Empty(t, diff)
		})
	})

	t.Run("GetCurrentBranch", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("GetCurrentBranch", ctx, "/tmp/repo").Return("master", nil)

			branch, err := mock.GetCurrentBranch(ctx, "/tmp/repo")
			require.NoError(t, err)
			require.Equal(t, "master", branch)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("GetCurrentBranch", ctx, "/tmp/repo").Return("", errNotGitRepo)

			branch, err := mock.GetCurrentBranch(ctx, "/tmp/repo")
			require.Error(t, err)
			require.Equal(t, errNotGitRepo, err)
			require.Empty(t, branch)
			mock.AssertExpectations(t)
		})
	})

	t.Run("GetRemoteURL", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockClient{}
			expectedURL := "https://github.com/test/repo.git"
			mock.On("GetRemoteURL", ctx, "/tmp/repo", "origin").Return(expectedURL, nil)

			url, err := mock.GetRemoteURL(ctx, "/tmp/repo", "origin")
			require.NoError(t, err)
			require.Equal(t, expectedURL, url)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockClient{}
			mock.On("GetRemoteURL", ctx, "/tmp/repo", "upstream").Return("", errRemoteNotFound)

			url, err := mock.GetRemoteURL(ctx, "/tmp/repo", "upstream")
			require.Error(t, err)
			require.Equal(t, errRemoteNotFound, err)
			require.Empty(t, url)
			mock.AssertExpectations(t)
		})
	})
}

// TestMockClientDefensiveProgramming tests the defensive programming in MockClient
func TestMockClientDefensiveProgramming(t *testing.T) {
	ctx := context.Background()

	t.Run("handles nil returns gracefully", func(t *testing.T) {
		mock := &MockClient{}

		// Test methods that return only error
		mock.On("Clone", ctx, "url", "path").Return(nil).Once()
		err := mock.Clone(ctx, "url", "path")
		require.NoError(t, err)

		mock.On("Checkout", ctx, "repo", "branch").Return(nil).Once()
		err = mock.Checkout(ctx, "repo", "branch")
		require.NoError(t, err)

		mock.On("CreateBranch", ctx, "repo", "branch").Return(nil).Once()
		err = mock.CreateBranch(ctx, "repo", "branch")
		require.NoError(t, err)

		mock.On("Add", ctx, "repo", []string{"file"}).Return(nil).Once()
		err = mock.Add(ctx, "repo", "file")
		require.NoError(t, err)

		mock.On("Commit", ctx, "repo", "message").Return(nil).Once()
		err = mock.Commit(ctx, "repo", "message")
		require.NoError(t, err)

		mock.On("Push", ctx, "repo", "remote", "branch", false).Return(nil).Once()
		err = mock.Push(ctx, "repo", "remote", "branch", false)
		require.NoError(t, err)

		// Test methods that return value and error
		mock.On("Diff", ctx, "repo", false).Return("", nil).Once()
		diff, err := mock.Diff(ctx, "repo", false)
		require.NoError(t, err)
		require.Empty(t, diff)

		mock.On("GetCurrentBranch", ctx, "repo").Return("", nil).Once()
		branch, err := mock.GetCurrentBranch(ctx, "repo")
		require.NoError(t, err)
		require.Empty(t, branch)

		mock.On("GetRemoteURL", ctx, "repo", "origin").Return("", nil).Once()
		url, err := mock.GetRemoteURL(ctx, "repo", "origin")
		require.NoError(t, err)
		require.Empty(t, url)

		mock.AssertExpectations(t)
	})
}

// TestMockClientConcurrency tests that MockClient is safe for concurrent use
func TestMockClientConcurrency(_ *testing.T) {
	ctx := context.Background()
	mock := &MockClient{}

	// Set up expectations for concurrent calls
	mock.On("Clone", ctx, "url", "path").Return(nil).Maybe()
	mock.On("Checkout", ctx, "repo", "branch").Return(nil).Maybe()
	mock.On("CreateBranch", ctx, "repo", "branch").Return(nil).Maybe()
	mock.On("Add", ctx, "repo", []string{"file"}).Return(nil).Maybe()
	mock.On("Commit", ctx, "repo", "message").Return(nil).Maybe()
	mock.On("Push", ctx, "repo", "remote", "branch", false).Return(nil).Maybe()
	mock.On("Diff", ctx, "repo", false).Return("diff", nil).Maybe()
	mock.On("GetCurrentBranch", ctx, "repo").Return("master", nil).Maybe()
	mock.On("GetRemoteURL", ctx, "repo", "origin").Return("url", nil).Maybe()

	// Run concurrent operations
	done := make(chan bool, 9)

	// Launch goroutines for each method
	go func() {
		for i := 0; i < 10; i++ {
			_ = mock.Clone(ctx, "url", "path")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_ = mock.Checkout(ctx, "repo", "branch")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_ = mock.CreateBranch(ctx, "repo", "branch")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_ = mock.Add(ctx, "repo", "file")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_ = mock.Commit(ctx, "repo", "message")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_ = mock.Push(ctx, "repo", "remote", "branch", false)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_, _ = mock.Diff(ctx, "repo", false)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_, _ = mock.GetCurrentBranch(ctx, "repo")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_, _ = mock.GetRemoteURL(ctx, "repo", "origin")
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 9; i++ {
		<-done
	}

	// No assertion needed - test passes if no race conditions or panics occur
}
