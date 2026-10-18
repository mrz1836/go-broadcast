package gh

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
)

// createPRArgs is the argument list the gh CLI is invoked with when creating a PR
func createPRArgs() []string {
	return []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}
}

// listPRArgs is the argument list used when reconciling against existing PRs
func listPRArgs() []string {
	return []string{"api", "repos/org/repo/pulls?state=open", "--paginate"}
}

// useFastRetries shrinks the retry backoff so tests do not sleep through the real
// multi-second delays. Tests using it must not run in parallel.
func useFastRetries(t *testing.T) {
	t.Helper()

	original := initialRetryDelay
	initialRetryDelay = time.Millisecond

	t.Cleanup(func() { initialRetryDelay = original })
}

// serverError builds an error shaped like the one gh returns for an HTTP 502
func serverError() error {
	return &CommandError{Stderr: "gh: Server Error (HTTP 502)\n"}
}

// emptyBodyError builds the error gh returns when GitHub sends an error status
// with an empty JSON body - gh fails to parse it and loses the status code
func emptyBodyError() error {
	return &CommandError{Stderr: "unexpected end of JSON input\n"}
}

// connectTimeoutError builds the error gh returns on macOS when the TCP connection
// to GitHub cannot be established - it says "timed out", never "timeout"
func connectTimeoutError() error {
	return &CommandError{
		Stderr: `Get "https://api.github.com/repos/org/repo/branches/main": dial tcp 140.82.114.6:443: connect: operation timed out` + "\n",
	}
}

// assigneesArgs is the argument list used when assigning users to PR 42
func assigneesArgs() []string {
	return []string{"api", "repos/org/repo/issues/42/assignees", "--method", "POST", "--input", "-"}
}

func testPRRequest() PRRequest {
	return PRRequest{
		Title: "Test PR",
		Body:  "Test description",
		Head:  "feature-branch",
		Base:  "main",
	}
}

func marshalPR(t *testing.T, pr PR) []byte {
	t.Helper()

	data, err := json.Marshal(pr)
	require.NoError(t, err)

	return data
}

// TestCreatePR_RetriesTransientServerError verifies that a 502 on the first attempt
// is retried rather than failing the sync, which is the failure this guards against.
func TestCreatePR_RetriesTransientServerError(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	success := marshalPR(t, PR{Number: 42, Title: "Test PR", State: "open"})

	// First attempt fails with 502, second succeeds
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(nil, serverError()).Once()
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(success, nil).Once()

	// The retry reconciles first and finds no existing PR
	mockRunner.On("Run", ctx, "gh", listPRArgs()).
		Return([]byte(`[]`), nil).Once()

	result, err := client.CreatePR(ctx, "org/repo", testPRRequest())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_AdoptsPRCreatedDespiteServerError covers the dangerous case: GitHub
// committed the write and then returned 502, so a blind retry would open a duplicate.
func TestCreatePR_AdoptsPRCreatedDespiteServerError(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	existing := PR{Number: 99, Title: "Test PR", State: "open"}
	existing.Head.Ref = "feature-branch"
	listOutput, err := json.Marshal([]PR{existing})
	require.NoError(t, err)

	// The create is attempted exactly once and appears to fail
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(nil, serverError()).Once()

	// Reconciling reveals the PR actually exists
	mockRunner.On("Run", ctx, "gh", listPRArgs()).
		Return(listOutput, nil).Once()

	result, err := client.CreatePR(ctx, "org/repo", testPRRequest())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 99, result.Number, "should adopt the existing PR rather than create a duplicate")

	// AssertExpectations fails if a second create was issued
	mockRunner.AssertExpectations(t)
	mockRunner.AssertNumberOfCalls(t, "RunWithInput", 1)
}

// TestCreatePR_ReconcileIgnoresUnrelatedBranches ensures a PR on a different branch
// is never mistaken for this one.
func TestCreatePR_ReconcileIgnoresUnrelatedBranches(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	other := PR{Number: 7, State: "open"}
	other.Head.Ref = "some-other-branch"
	listOutput, err := json.Marshal([]PR{other})
	require.NoError(t, err)

	success := marshalPR(t, PR{Number: 42, State: "open"})

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(nil, serverError()).Once()
	mockRunner.On("Run", ctx, "gh", listPRArgs()).
		Return(listOutput, nil).Once()
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(success, nil).Once()

	result, err := client.CreatePR(ctx, "org/repo", testPRRequest())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_ExhaustsRetriesThenFails verifies a persistent 502 still surfaces an
// error, and that the error keeps the head/base context callers rely on.
func TestCreatePR_ExhaustsRetriesThenFails(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(nil, serverError())
	mockRunner.On("Run", ctx, "gh", listPRArgs()).
		Return([]byte(`[]`), nil)

	result, err := client.CreatePR(ctx, "org/repo", testPRRequest())
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create PR with head 'org:feature-branch' and base 'main'")
	assert.Contains(t, err.Error(), "502")

	// maxRetries attempts, no more
	mockRunner.AssertNumberOfCalls(t, "RunWithInput", maxRetries)
}

// TestCreatePR_DoesNotRetryValidationFailure guards against burning retries and wall
// clock on a 422, which is permanent and already has its own recovery path.
func TestCreatePR_DoesNotRetryValidationFailure(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	validationErr := &CommandError{
		Stderr: "gh: Validation Failed (HTTP 422)\nA pull request already exists for org:feature-branch.",
	}

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(nil, validationErr)

	result, err := client.CreatePR(ctx, "org/repo", testPRRequest())
	require.Error(t, err)
	assert.Nil(t, result)
	require.ErrorIs(t, err, ErrPRValidationFailed)

	mockRunner.AssertNumberOfCalls(t, "RunWithInput", 1)
}

// TestCreatePR_DoesNotRetryGenericError confirms only transient errors are retried
func TestCreatePR_DoesNotRetryGenericError(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(nil, internalerrors.ErrTest)

	result, err := client.CreatePR(ctx, "org/repo", testPRRequest())
	require.Error(t, err)
	assert.Nil(t, result)

	mockRunner.AssertNumberOfCalls(t, "RunWithInput", 1)
}

// TestCreatePR_SucceedsWithoutReconcile confirms the happy path is unchanged: no
// extra API call is spent listing PRs when the first attempt works.
func TestCreatePR_SucceedsWithoutReconcile(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	success := marshalPR(t, PR{Number: 42, State: "open"})

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(success, nil).Once()

	result, err := client.CreatePR(ctx, "org/repo", testPRRequest())
	require.NoError(t, err)
	require.NotNil(t, result)

	mockRunner.AssertNumberOfCalls(t, "Run", 0)
	mockRunner.AssertExpectations(t)
}

// TestCreatePR_ReconcileFailureFallsBackToRetry ensures an unusable PR listing does
// not abort the retry - failing to confirm is not the same as confirming absence.
func TestCreatePR_ReconcileFailureFallsBackToRetry(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	success := marshalPR(t, PR{Number: 42, State: "open"})

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(nil, serverError()).Once()
	mockRunner.On("Run", ctx, "gh", listPRArgs()).
		Return(nil, internalerrors.ErrTest).Once()
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(success, nil).Once()

	result, err := client.CreatePR(ctx, "org/repo", testPRRequest())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_RespectsContextCancellation ensures a canceled context stops retrying
func TestCreatePR_RespectsContextCancellation(t *testing.T) {
	useFastRetries(t)

	ctx, cancel := context.WithCancel(context.Background())
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Run(func(mock.Arguments) { cancel() }).
		Return(nil, serverError()).Once()

	result, err := client.CreatePR(ctx, "org/repo", testPRRequest())
	require.Error(t, err)
	assert.Nil(t, result)

	mockRunner.AssertNumberOfCalls(t, "RunWithInput", 1)
}

// TestCreatePR_RetriesAssigneesOnEmptyErrorBody covers a GitHub blip right after PR
// creation that previously left the PR without its configured assignees.
func TestCreatePR_RetriesAssigneesOnEmptyErrorBody(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := testPRRequest()
	req.Assignees = []string{"user1"}

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(marshalPR(t, PR{Number: 42, State: "open"}), nil).Once()
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", assigneesArgs()).
		Return(nil, emptyBodyError()).Once()
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", assigneesArgs()).
		Return([]byte(`{}`), nil).Once()

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err)
	require.NotNil(t, result)

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_DoesNotRetryPermanentAssigneeFailure ensures an invalid assignee
// (HTTP 422) is not retried - it cannot succeed and would only add backoff delay.
func TestCreatePR_DoesNotRetryPermanentAssigneeFailure(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := testPRRequest()
	req.Assignees = []string{"not-a-collaborator"}

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", createPRArgs()).
		Return(marshalPR(t, PR{Number: 42, State: "open"}), nil).Once()
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", assigneesArgs()).
		Return(nil, &CommandError{Stderr: "gh: Validation Failed (HTTP 422)"})

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err, "assignee failures are logged, not fatal")
	require.NotNil(t, result)

	// One create, one assignee attempt
	mockRunner.AssertNumberOfCalls(t, "RunWithInput", 2)
}

func TestIsTransientServerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "nil error", err: nil, expected: false},
		{name: "gh 502 server error", err: serverError(), expected: true},
		{name: "503 service unavailable", err: &CommandError{Stderr: "gh: Service Unavailable (HTTP 503)"}, expected: true},
		{name: "504 gateway timeout", err: &CommandError{Stderr: "gh: Gateway Timeout (HTTP 504)"}, expected: true},
		{name: "bad gateway text", err: &CommandError{Stderr: "bad gateway"}, expected: true},
		{name: "connection reset", err: &CommandError{Stderr: "connection reset by peer"}, expected: true},
		{name: "connection refused", err: &CommandError{Stderr: "connection refused"}, expected: true},
		{name: "request timeout", err: &CommandError{Stderr: "request timeout"}, expected: true},
		{name: "macOS connect timeout", err: connectTimeoutError(), expected: true},
		{name: "i/o timeout", err: &CommandError{Stderr: "dial tcp 140.82.114.6:443: i/o timeout"}, expected: true},
		{name: "network unreachable", err: &CommandError{Stderr: "dial tcp 140.82.114.6:443: connect: network is unreachable"}, expected: true},
		{name: "dns lookup failure", err: &CommandError{Stderr: "dial tcp: lookup api.github.com: no such host"}, expected: true},
		{name: "empty error body", err: emptyBodyError(), expected: true},
		{name: "422 validation failed", err: &CommandError{Stderr: "gh: Validation Failed (HTTP 422)"}, expected: false},
		{name: "404 not found", err: &CommandError{Stderr: "gh: Not Found (HTTP 404)"}, expected: false},
		{name: "403 forbidden", err: &CommandError{Stderr: "gh: Forbidden (HTTP 403)"}, expected: false},
		{name: "generic error", err: internalerrors.ErrTest, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isTransientServerError(tt.err))
		})
	}
}

// TestGetBranch_RetriesConnectTimeout covers a sync target that failed after its
// branch was pushed because one base-branch check hit a dropped connection.
func TestGetBranch_RetriesConnectTimeout(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	args := []string{"api", "repos/org/repo/branches/main"}
	mockRunner.On("Run", ctx, "gh", args).
		Return(nil, connectTimeoutError()).Once()
	mockRunner.On("Run", ctx, "gh", args).
		Return([]byte(`{"name":"main"}`), nil).Once()

	branch, err := client.GetBranch(ctx, "org/repo", "main")
	require.NoError(t, err)
	require.NotNil(t, branch)
	assert.Equal(t, "main", branch.Name)

	mockRunner.AssertExpectations(t)
}

// TestGetBranch_DoesNotRetryNotFound ensures a missing branch is reported at once
// rather than after the retry backoff.
func TestGetBranch_DoesNotRetryNotFound(t *testing.T) {
	useFastRetries(t)

	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/branches/main"}).
		Return(nil, &CommandError{Stderr: "gh: Not Found (HTTP 404)"})

	branch, err := client.GetBranch(ctx, "org/repo", "main")
	require.ErrorIs(t, err, ErrBranchNotFound)
	assert.Nil(t, branch)

	mockRunner.AssertNumberOfCalls(t, "Run", 1)
}

// TestRateLimitedDoIf covers the retry predicate in isolation
func TestRateLimitedDoIf(t *testing.T) {
	useFastRetries(t)

	t.Run("stops immediately when error is not retryable", func(t *testing.T) {
		attempts := 0
		err := rateLimitedDoIf(context.Background(), 0, func(error) bool { return false }, func() error {
			attempts++
			return internalerrors.ErrTest
		})

		require.ErrorIs(t, err, internalerrors.ErrTest, "non-retryable errors must surface unwrapped")
		assert.Equal(t, 1, attempts)
	})

	t.Run("retries until success", func(t *testing.T) {
		attempts := 0
		err := rateLimitedDoIf(context.Background(), 0, func(error) bool { return true }, func() error {
			attempts++
			if attempts < 2 {
				return internalerrors.ErrTest
			}
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("gives up after maxRetries", func(t *testing.T) {
		attempts := 0
		err := rateLimitedDoIf(context.Background(), 0, func(error) bool { return true }, func() error {
			attempts++
			return internalerrors.ErrTest
		})

		require.Error(t, err)
		assert.Equal(t, maxRetries, attempts)
		assert.Contains(t, err.Error(), "after 3 attempts")
	})

	t.Run("rateLimitedDo still retries every error", func(t *testing.T) {
		attempts := 0
		err := rateLimitedDo(context.Background(), 0, func() error {
			attempts++
			return internalerrors.ErrTest
		})

		require.Error(t, err)
		assert.Equal(t, maxRetries, attempts)
	})
}
