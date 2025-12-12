package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// Static test errors for linter compliance
var (
	errRetryEarlyEOF          = errors.New("early EOF")
	errRetryConnectionReset   = errors.New("connection reset by peer")
	errRetryTimeout           = errors.New("timeout waiting for response")
	errRetryNetworkUnreach    = errors.New("network is unreachable")
	errRetryTempFailure       = errors.New("temporary failure in name resolution")
	errRetryConnectionRefused = errors.New("connection refused")
	errRetryEarlyEOFDetected  = errors.New("early EOF detected")
	errRetryRepoNotFound      = errors.New("repository not found")
	errRetryAuthFailed        = errors.New("authentication failed")
	errRetryGeneric           = errors.New("something went wrong")
	errRetryPermissionDenied  = errors.New("permission denied")
	errRetryInvalidURL        = errors.New("invalid url")
	errRetryFileNotFound      = errors.New("file not found")
	errRetrySyntaxError       = errors.New("syntax error")
	errRetryUnmappedTest      = errors.New("unmapped test error")
)

// getStaticError returns a static error for the given error message
func getStaticError(errMsg string) error {
	errorMap := map[string]error{
		"early eof":                errRetryEarlyEOF,
		"connection reset":         errRetryConnectionReset,
		"timeout":                  errRetryTimeout,
		"network is unreachable":   errRetryNetworkUnreach,
		"temporary failure":        errRetryTempFailure,
		"connection refused":       errRetryConnectionRefused,
		"EARLY EOF":                errRetryEarlyEOF,
		"Connection Reset By Peer": errRetryConnectionReset,
		"Network timeout occurred": errRetryTimeout,
		"authentication failed":    errRetryAuthFailed,
		"repository not found":     errRetryRepoNotFound,
		"permission denied":        errRetryPermissionDenied,
		"invalid url":              errRetryInvalidURL,
		"file not found":           errRetryFileNotFound,
		"syntax error":             errRetrySyntaxError,
	}

	if staticErr, exists := errorMap[errMsg]; exists {
		return staticErr
	}
	return errRetryUnmappedTest // fallback for unmapped errors
}

// TestIsRetryableNetworkError tests the network error detection function
func TestIsRetryableNetworkError(t *testing.T) {
	testCases := []struct {
		name        string
		err         error
		shouldRetry bool
	}{
		{
			name:        "nil error",
			err:         nil,
			shouldRetry: false,
		},
		{
			name:        "early eof error",
			err:         errRetryEarlyEOF,
			shouldRetry: true,
		},
		{
			name:        "connection reset error",
			err:         errRetryConnectionReset,
			shouldRetry: true,
		},
		{
			name:        "timeout error",
			err:         errRetryTimeout,
			shouldRetry: true,
		},
		{
			name:        "network unreachable error",
			err:         errRetryNetworkUnreach,
			shouldRetry: true,
		},
		{
			name:        "temporary failure error",
			err:         errRetryTempFailure,
			shouldRetry: true,
		},
		{
			name:        "connection refused error",
			err:         errRetryConnectionRefused,
			shouldRetry: true,
		},
		{
			name:        "mixed case early EOF",
			err:         errRetryEarlyEOFDetected,
			shouldRetry: true,
		},
		{
			name:        "non-retryable error",
			err:         errRetryRepoNotFound,
			shouldRetry: false,
		},
		{
			name:        "authentication error",
			err:         errRetryAuthFailed,
			shouldRetry: false,
		},
		{
			name:        "generic error",
			err:         errRetryGeneric,
			shouldRetry: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isRetryableNetworkError(tc.err)
			assert.Equal(t, tc.shouldRetry, result, "Expected isRetryableNetworkError(%v) to be %v", tc.err, tc.shouldRetry)
		})
	}
}

// TestGitClient_CloneWithRetry tests the clone retry logic
func TestGitClient_CloneWithRetry(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()

	t.Run("successful clone on first attempt", func(t *testing.T) {
		client, err := NewClient(logger, &logging.LogConfig{})
		require.NoError(t, err)

		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "test-repo")

		// Create a source repository to clone from
		sourceRepo := filepath.Join(tmpDir, "source")
		cmd := exec.CommandContext(ctx, "git", "init", "--bare", sourceRepo) //nolint:gosec // test command with controlled args
		require.NoError(t, cmd.Run())

		// Clone should succeed on first attempt
		err = client.Clone(ctx, sourceRepo, repoPath, nil)
		require.NoError(t, err)

		// Verify the repository was cloned
		_, statErr := os.Stat(filepath.Join(repoPath, ".git"))
		assert.NoError(t, statErr)
	})

	t.Run("repository already exists", func(t *testing.T) {
		client, err := NewClient(logger, &logging.LogConfig{})
		require.NoError(t, err)

		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "existing-repo")

		// Create the directory first
		require.NoError(t, os.MkdirAll(repoPath, 0o750))

		err = client.Clone(ctx, "https://example.com/repo.git", repoPath, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository already exists")
	})

	t.Run("clone with immediate context cancellation", func(t *testing.T) {
		client, err := NewClient(logger, &logging.LogConfig{})
		require.NoError(t, err)

		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "canceled-repo")

		// Cancel context immediately to test cancellation path reliably
		// This avoids timing-dependent behavior that causes flaky tests
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		// Use any URL - the canceled context should cause immediate failure
		err = client.Clone(cancelCtx, "https://example.com/repo.git", repoPath, nil)
		require.Error(t, err)
		// Should get context canceled error
		assert.ErrorIs(t, err, context.Canceled)
	})
}

// mockGitClientForRetryTesting provides controlled failure simulation
type mockGitClientForRetryTesting struct {
	*gitClient

	attemptCount  int
	maxFailures   int
	shouldSucceed bool
}

// simulateCloneWithRetry simulates the clone retry logic for testing
func (m *mockGitClientForRetryTesting) simulateClone(ctx context.Context, _, path string) error {
	// Check if path already exists
	if _, err := os.Stat(path); err == nil {
		return ErrRepositoryExists
	}

	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		m.attemptCount++

		// Simulate failures up to maxFailures
		if m.attemptCount <= m.maxFailures {
			var err error
			switch m.attemptCount {
			case 1:
				err = errRetryEarlyEOF
			case 2:
				err = errRetryConnectionReset
			case 3:
				err = errRetryTimeout
			default:
				err = errRetryNetworkUnreach
			}

			if isRetryableNetworkError(err) && attempt < maxRetries {
				// Clean up partial clone
				_ = os.RemoveAll(path)

				// Brief delay before retry
				select {
				case <-time.After(time.Duration(attempt) * time.Millisecond):
				case <-ctx.Done():
					return ctx.Err()
				}
				continue
			}

			return err
		}

		// Success case
		if m.shouldSucceed {
			// Create the directory to simulate successful clone
			if err := os.MkdirAll(filepath.Join(path, ".git"), 0o750); err != nil {
				return err
			}
			return nil
		}

		return errRetryRepoNotFound
	}

	return errRetryGeneric
}

// TestCloneRetryLogic tests the retry logic in isolation
func TestCloneRetryLogic(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	ctx := context.Background()

	t.Run("success after 2 failures", func(t *testing.T) {
		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "retry-success")

		mockClient := &mockGitClientForRetryTesting{
			maxFailures:   2,
			shouldSucceed: true,
		}

		err := mockClient.simulateClone(ctx, "https://example.com/repo.git", repoPath)
		require.NoError(t, err)
		assert.Equal(t, 3, mockClient.attemptCount) // 2 failures + 1 success

		// Verify the "repository" was created
		_, statErr := os.Stat(filepath.Join(repoPath, ".git"))
		assert.NoError(t, statErr)
	})

	t.Run("failure after max retries", func(t *testing.T) {
		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "retry-fail")

		mockClient := &mockGitClientForRetryTesting{
			maxFailures:   5, // More than max retries
			shouldSucceed: false,
		}

		err := mockClient.simulateClone(ctx, "https://example.com/repo.git", repoPath)
		require.Error(t, err)
		assert.Equal(t, 3, mockClient.attemptCount) // 3 attempts (max retries)
		assert.Contains(t, err.Error(), "timeout waiting for response")
	})

	t.Run("non-retryable error", func(t *testing.T) {
		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "non-retryable")

		mockClient := &mockGitClientForRetryTesting{
			maxFailures:   0,
			shouldSucceed: false,
		}

		err := mockClient.simulateClone(ctx, "https://example.com/repo.git", repoPath)
		require.Error(t, err)
		assert.Equal(t, 1, mockClient.attemptCount) // Only 1 attempt
		assert.Contains(t, err.Error(), "repository not found")
	})
}

// TestNetworkErrorDetection tests edge cases in network error detection
func TestNetworkErrorDetection(t *testing.T) {
	networkErrors := []string{
		"early eof",
		"connection reset",
		"timeout",
		"network is unreachable",
		"temporary failure",
		"connection refused",
		"EARLY EOF", // Case insensitive
		"Connection Reset By Peer",
		"Network timeout occurred",
	}

	nonNetworkErrors := []string{
		"authentication failed",
		"repository not found",
		"permission denied",
		"invalid url",
		"file not found",
		"syntax error",
	}

	for _, errMsg := range networkErrors {
		t.Run("should retry: "+errMsg, func(t *testing.T) {
			err := getStaticError(errMsg)
			assert.True(t, isRetryableNetworkError(err), "Expected '%s' to be retryable", errMsg)
		})
	}

	for _, errMsg := range nonNetworkErrors {
		t.Run("should not retry: "+errMsg, func(t *testing.T) {
			err := getStaticError(errMsg)
			assert.False(t, isRetryableNetworkError(err), "Expected '%s' to NOT be retryable", errMsg)
		})
	}
}
