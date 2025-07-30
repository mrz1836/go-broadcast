package state

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/stretchr/testify/require"
)

// Define test errors as static wrapped errors
var (
	errDiscoveryFailed       = errors.New("discovery failed")
	errTestError             = errors.New("test error")
	errTargetDiscoveryFailed = errors.New("target discovery failed")
	errInvalidBranchName     = errors.New("invalid branch name")
)

// TestMockDiscovererImplementation tests the MockDiscoverer implementation
func TestMockDiscovererImplementation(t *testing.T) {
	ctx := context.Background()

	t.Run("DiscoverState", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockDiscoverer{}
			expectedState := &State{
				Source: SourceState{
					Repo:   "test/repo",
					Branch: "main",
				},
			}

			mock.On("DiscoverState", ctx, (*config.Config)(nil)).Return(expectedState, nil)

			state, err := mock.DiscoverState(ctx, nil)
			require.NoError(t, err)
			require.Equal(t, expectedState, state)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockDiscoverer{}
			mock.On("DiscoverState", ctx, (*config.Config)(nil)).Return((*State)(nil), errDiscoveryFailed)

			state, err := mock.DiscoverState(ctx, nil)
			require.Error(t, err)
			require.Equal(t, errDiscoveryFailed, err)
			require.Nil(t, state)
			mock.AssertExpectations(t)
		})

		t.Run("improperly configured mock - single argument", func(t *testing.T) {
			mock := &MockDiscoverer{}
			// Simulate improper mock configuration with only one return value
			mock.On("DiscoverState", ctx, (*config.Config)(nil)).Return(errTestError)

			state, err := mock.DiscoverState(ctx, nil)
			require.Error(t, err)
			require.Nil(t, state)
			require.Equal(t, errTestError, err)
		})

		t.Run("improperly configured mock - no arguments", func(t *testing.T) {
			mock := &MockDiscoverer{}

			// Simulate improper mock configuration with no return values
			mock.On("DiscoverState", ctx, (*config.Config)(nil)).Return()

			state, err := mock.DiscoverState(ctx, nil)
			require.Error(t, err)
			require.Nil(t, state)
			require.Contains(t, err.Error(), "mock not properly configured")
		})
	})

	t.Run("DiscoverTargetState", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockDiscoverer{}
			expectedState := &TargetState{
				Repo:         "target/repo",
				SyncBranches: []SyncBranch{{Name: "sync-123"}},
			}

			mock.On("DiscoverTargetState", ctx, "target/repo", "chore/sync-files").Return(expectedState, nil)

			state, err := mock.DiscoverTargetState(ctx, "target/repo", "chore/sync-files")
			require.NoError(t, err)
			require.Equal(t, expectedState, state)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockDiscoverer{}
			mock.On("DiscoverTargetState", ctx, "target/repo", "chore/sync-files").Return((*TargetState)(nil), errTargetDiscoveryFailed)

			state, err := mock.DiscoverTargetState(ctx, "target/repo", "chore/sync-files")
			require.Error(t, err)
			require.Equal(t, errTargetDiscoveryFailed, err)
			require.Nil(t, state)
			mock.AssertExpectations(t)
		})

		t.Run("improperly configured mock", func(t *testing.T) {
			mock := &MockDiscoverer{}

			// Simulate improper mock configuration
			mock.On("DiscoverTargetState", ctx, "target/repo", "chore/sync-files").Return()

			state, err := mock.DiscoverTargetState(ctx, "target/repo", "chore/sync-files")
			require.Error(t, err)
			require.Nil(t, state)
			require.Contains(t, err.Error(), "mock not properly configured")
		})
	})

	t.Run("ParseBranchName", func(t *testing.T) {
		t.Run("success case", func(t *testing.T) {
			mock := &MockDiscoverer{}
			expectedMetadata := &BranchMetadata{
				Timestamp: time.Now(),
				CommitSHA: "abc123",
				Prefix:    "chore/sync-files",
			}

			mock.On("ParseBranchName", "sync-123-source-repo-main").Return(expectedMetadata, nil)

			metadata, err := mock.ParseBranchName("sync-123-source-repo-main")
			require.NoError(t, err)
			require.Equal(t, expectedMetadata, metadata)
			mock.AssertExpectations(t)
		})

		t.Run("error case", func(t *testing.T) {
			mock := &MockDiscoverer{}
			mock.On("ParseBranchName", "invalid-branch").Return((*BranchMetadata)(nil), errInvalidBranchName)

			metadata, err := mock.ParseBranchName("invalid-branch")
			require.Error(t, err)
			require.Equal(t, errInvalidBranchName, err)
			require.Nil(t, metadata)
			mock.AssertExpectations(t)
		})

		t.Run("improperly configured mock", func(t *testing.T) {
			mock := &MockDiscoverer{}

			// Simulate improper mock configuration
			mock.On("ParseBranchName", "test-branch").Return()

			metadata, err := mock.ParseBranchName("test-branch")
			require.Error(t, err)
			require.Nil(t, metadata)
			require.Contains(t, err.Error(), "mock not properly configured")
		})
	})
}

// TestMockDiscovererDefensiveProgramming tests the defensive programming in MockDiscoverer
func TestMockDiscovererDefensiveProgramming(t *testing.T) {
	ctx := context.Background()

	t.Run("handles nil returns gracefully", func(t *testing.T) {
		mock := &MockDiscoverer{}

		// Test nil state with nil error
		mock.On("DiscoverState", ctx, (*config.Config)(nil)).Return(nil, nil).Once()
		state, err := mock.DiscoverState(ctx, nil)
		require.NoError(t, err)
		require.Nil(t, state)

		// Test nil target state with nil error
		mock.On("DiscoverTargetState", ctx, "repo", "chore/sync-files").Return(nil, nil).Once()
		targetState, err := mock.DiscoverTargetState(ctx, "repo", "chore/sync-files")
		require.NoError(t, err)
		require.Nil(t, targetState)

		// Test nil metadata with nil error
		mock.On("ParseBranchName", "branch").Return(nil, nil).Once()
		metadata, err := mock.ParseBranchName("branch")
		require.NoError(t, err)
		require.Nil(t, metadata)

		mock.AssertExpectations(t)
	})
}

// TestMockDiscovererConcurrency tests that MockDiscoverer is safe for concurrent use
func TestMockDiscovererConcurrency(_ *testing.T) {
	ctx := context.Background()
	mock := &MockDiscoverer{}

	// Set up expectations for concurrent calls
	mock.On("DiscoverState", ctx, (*config.Config)(nil)).Return(&State{}, nil).Maybe()
	mock.On("DiscoverTargetState", ctx, "repo", "chore/sync-files").Return(&TargetState{}, nil).Maybe()
	mock.On("ParseBranchName", "branch").Return(&BranchMetadata{}, nil).Maybe()

	// Run concurrent operations
	done := make(chan bool, 3)

	go func() {
		for i := 0; i < 10; i++ {
			_, _ = mock.DiscoverState(ctx, nil)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_, _ = mock.DiscoverTargetState(ctx, "repo", "chore/sync-files")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			_, _ = mock.ParseBranchName("branch")
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	// No assertion needed - test passes if no race conditions or panics occur
}
