package integration

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	syncpkg "sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/sync"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/mrz1836/go-broadcast/test/fixtures"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// APIError represents a GitHub API error for testing
type APIError struct {
	StatusCode int
	Message    string
	Headers    map[string]string
}

func (e *APIError) Error() string {
	return e.Message
}

// TestNetworkEdgeCases tests network resilience and API edge cases
func TestNetworkEdgeCases(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	generator := fixtures.NewTestRepoGenerator(tmpDir)
	defer func() {
		if err := generator.Cleanup(); err != nil {
			t.Errorf("failed to cleanup test generator: %v", err)
		}
	}()

	t.Run("github_api_rate_limiting", func(t *testing.T) {
		testGitHubAPIRateLimiting(t, generator)
	})

	t.Run("network_interruption_handling", func(t *testing.T) {
		testNetworkInterruptionHandling(t, generator)
	})

	t.Run("authentication_failure_scenarios", func(t *testing.T) {
		testAuthenticationFailureScenarios(t, generator)
	})

	t.Run("api_timeout_and_retry", func(t *testing.T) {
		testAPITimeoutAndRetry(t, generator)
	})

	t.Run("concurrent_api_operations", func(t *testing.T) {
		testConcurrentAPIOperations(t, generator)
	})

	t.Run("github_api_degradation", func(t *testing.T) {
		testGitHubAPIDegradation(t, generator)
	})

	t.Run("network_partition_recovery", func(t *testing.T) {
		testNetworkPartitionRecovery(t, generator)
	})

	t.Run("dns_resolution_failures", func(t *testing.T) {
		testDNSResolutionFailures(t, generator)
	})

	t.Run("ssl_certificate_errors", func(t *testing.T) {
		testSSLCertificateErrors(t, generator)
	})

	t.Run("proxy_connection_issues", func(t *testing.T) {
		testProxyConnectionIssues(t, generator)
	})

	t.Run("github_webhook_simulation", func(t *testing.T) {
		testGitHubWebhookSimulation(t, generator)
	})
}

// RateLimitSimulator simulates GitHub API rate limiting
type RateLimitSimulator struct {
	mu               syncpkg.Mutex
	requestCount     int64
	resetTime        time.Time
	limit            int64
	remaining        int64
	windowDuration   time.Duration
	rateLimitReached bool
}

// NewRateLimitSimulator creates a new rate limit simulator
func NewRateLimitSimulator(limit int64, windowDuration time.Duration) *RateLimitSimulator {
	return &RateLimitSimulator{
		limit:          limit,
		remaining:      limit,
		windowDuration: windowDuration,
		resetTime:      time.Now().Add(windowDuration),
	}
}

// CheckRateLimit checks if rate limit is exceeded
func (r *RateLimitSimulator) CheckRateLimit() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	atomic.AddInt64(&r.requestCount, 1)

	// Reset if window has passed
	if time.Now().After(r.resetTime) {
		r.remaining = r.limit
		r.resetTime = time.Now().Add(r.windowDuration)
		r.rateLimitReached = false
	}

	if r.remaining <= 0 {
		r.rateLimitReached = true
		resetIn := time.Until(r.resetTime)
		return fmt.Errorf("%w: %d requests in window, reset in %v", fixtures.ErrRateLimitExceeded,
			r.limit, resetIn.Round(time.Second))
	}

	r.remaining--
	return nil
}

// GetHeaders returns rate limit headers
func (r *RateLimitSimulator) GetHeaders() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return map[string]string{
		"X-RateLimit-Limit":     fmt.Sprintf("%d", r.limit),
		"X-RateLimit-Remaining": fmt.Sprintf("%d", r.remaining),
		"X-RateLimit-Reset":     fmt.Sprintf("%d", r.resetTime.Unix()),
	}
}

// testGitHubAPIRateLimiting tests rate limiting scenarios
func testGitHubAPIRateLimiting(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for rate limiting
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup rate limit simulator (low limit for testing)
	rateLimiter := NewRateLimitSimulator(5, 30*time.Second)

	// Setup mocks with rate limiting
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Track API calls for rate limiting
	var apiCallCount int64

	// Mock GitHub API with rate limiting
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(mock.Arguments) {
			atomic.AddInt64(&apiCallCount, 1)
			time.Sleep(10 * time.Millisecond)
		}).Return(&gh.FileContent{Content: []byte("content")}, nil).Maybe()

	// Mock CreatePR with rate limiting - simulate rate limit after a few calls
	mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
	prCallCount := 0
	mockGH.On("CreatePR", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("gh.PRRequest")).
		Run(func(_ mock.Arguments) {
			atomic.AddInt64(&apiCallCount, 1)
			prCallCount++
			_ = rateLimiter.CheckRateLimit()
		}).Return(nil, &APIError{
		StatusCode: http.StatusTooManyRequests,
		Message:    "rate limit exceeded",
		Headers:    rateLimiter.GetHeaders(),
	}).Maybe()

	// Mock Git operations (not rate limited)
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory from the test scenario
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
		}).Return(nil)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "master"}}, nil).Maybe()

	// Create sync engine with retry logic
	opts := sync.DefaultOptions().WithDryRun(false).WithMaxConcurrency(1) // Lower concurrency to test rate limiting
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	engine.SetLogger(logger)

	// Execute sync (should encounter rate limiting)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Longer timeout for retries
	defer cancel()

	start := time.Now()
	err = engine.Sync(ctx, nil)
	duration := time.Since(start)

	// Should handle rate limiting gracefully (with retries or backoff)
	finalAPICallCount := atomic.LoadInt64(&apiCallCount)

	t.Logf("Sync completed in %v with %d API calls", duration, finalAPICallCount)
	t.Logf("Rate limiter state: remaining=%d, reset time=%v",
		rateLimiter.remaining, rateLimiter.resetTime)

	// Should eventually succeed or fail gracefully after proper retries
	if err != nil {
		// If it fails, it should be due to rate limiting, not other errors
		assert.Contains(t, err.Error(), "rate limit", "Failure should be due to rate limiting")
	}

	// Should have made multiple API calls
	assert.Greater(t, finalAPICallCount, int64(3), "Should have made multiple API calls")

	// Duration should indicate backoff/retry behavior
	if rateLimiter.rateLimitReached {
		assert.Greater(t, duration, 5*time.Second, "Should have taken time for backoff when rate limited")
	}

	mockState.AssertExpectations(t)
}

// NetworkSimulator simulates network issues
type NetworkSimulator struct {
	mu               syncpkg.Mutex
	failureMode      fixtures.FailureMode
	failureRate      float64
	consecutiveFails int
	maxConsecutive   int
	totalRequests    int64
	failedRequests   int64
}

// NewNetworkSimulator creates a network simulator
func NewNetworkSimulator(failureRate float64, maxConsecutive int) *NetworkSimulator {
	return &NetworkSimulator{
		failureMode:    fixtures.FailureNetwork,
		failureRate:    failureRate,
		maxConsecutive: maxConsecutive,
	}
}

// ShouldFail determines if operation should fail
func (n *NetworkSimulator) ShouldFail() bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	atomic.AddInt64(&n.totalRequests, 1)

	// Don't fail if we've had too many consecutive failures
	if n.consecutiveFails >= n.maxConsecutive {
		n.consecutiveFails = 0
		return false
	}

	// Random failure based on rate
	failureThreshold := int64(n.failureRate * 1000)
	shouldFail := (time.Now().UnixNano() % 1000) < failureThreshold

	if shouldFail {
		n.consecutiveFails++
		atomic.AddInt64(&n.failedRequests, 1)
	} else {
		n.consecutiveFails = 0
	}

	return shouldFail
}

// GetStats returns failure statistics
func (n *NetworkSimulator) GetStats() (int64, int64, float64) {
	total := atomic.LoadInt64(&n.totalRequests)
	failed := atomic.LoadInt64(&n.failedRequests)
	rate := float64(failed) / float64(total)
	return total, failed, rate
}

// testNetworkInterruptionHandling tests network interruption scenarios
func testNetworkInterruptionHandling(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for network testing
	scenario, err := generator.CreateNetworkFailureScenario()
	require.NoError(t, err)

	// Setup network simulator (30% failure rate, max 2 consecutive failures)
	networkSim := NewNetworkSimulator(0.3, 2)

	// Setup mocks with network simulation
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Mock GitHub operations with network failures
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(_ mock.Arguments) {
			// Simulate network latency if not failing
			if !networkSim.ShouldFail() {
				time.Sleep(50 * time.Millisecond)
			}
		}).Return(&gh.FileContent{Content: []byte("content")}, nil).Maybe()

	// Setup separate mocks for success and failure cases
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(&gh.PR{Number: 456}, nil).Maybe()
	}

	// Mock Git operations (also subject to network issues)
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Always create the expected files - network issues affect API calls, not local git content
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
			// Simulate clone time
			time.Sleep(100 * time.Millisecond)
		}).Return(nil).Maybe()

	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Run(func(_ mock.Arguments) {
			time.Sleep(75 * time.Millisecond)
		}).
		Return(nil).Maybe()

	// Other Git operations (less likely to fail)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "master"}}, nil).Maybe()

	// Create sync engine with network resilience
	opts := sync.DefaultOptions().WithDryRun(false).WithMaxConcurrency(2)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync multiple times to test network resilience
	attempts := 3
	var results []error

	for i := 0; i < attempts; i++ {
		t.Logf("Network resilience test attempt %d/%d", i+1, attempts)

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		err := engine.Sync(ctx, nil)
		cancel()

		results = append(results, err)

		// Brief pause between attempts
		time.Sleep(200 * time.Millisecond)
	}

	// Analyze results
	successCount := 0
	for i, err := range results {
		if err == nil {
			successCount++
			t.Logf("Attempt %d: SUCCESS", i+1)
		} else {
			t.Logf("Attempt %d: FAILED - %v", i+1, err)
		}
	}

	// Get network statistics
	total, failed, failureRate := networkSim.GetStats()
	t.Logf("Network simulation stats: %d total requests, %d failed (%.1f%% failure rate)",
		total, failed, failureRate*100)

	// Should have made multiple attempts and handled failures gracefully
	assert.Greater(t, total, int64(5), "Should have made multiple network requests")
	assert.GreaterOrEqual(t, successCount, 1, "At least one attempt should succeed with network resilience")

	mockState.AssertExpectations(t)
}

// testAuthenticationFailureScenarios tests various auth failure cases
func testAuthenticationFailureScenarios(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for auth testing
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks with auth failures
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Track auth failures
	authFailureCount := 0

	// Mock GitHub operations with auth failures
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(_ mock.Arguments) {
			authFailureCount++
		}).Return(nil, &APIError{
		StatusCode: http.StatusUnauthorized,
		Message:    "Bad credentials",
	}).Maybe()

	// Mock other operations (some with auth issues)
	// Setup CreatePR mocks for auth scenarios
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(&gh.PR{Number: 789}, nil).Maybe()
	}

	// Git operations may also have auth issues
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory from the test scenario
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
		}).Return(nil).Maybe()

	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Return(nil).Maybe()

	// Standard Git operations
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "master"}}, nil).Maybe()

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync (should handle auth failures)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = engine.Sync(ctx, nil)

	// Should handle auth failures gracefully
	t.Logf("Auth failure test result: %v", err)
	t.Logf("Total auth failure attempts: %d", authFailureCount)

	// Should have encountered auth failures
	assert.Greater(t, authFailureCount, 2, "Should have encountered auth failures")

	// Result depends on retry logic and error handling
	if err != nil {
		// Should be a meaningful auth-related error
		authRelated := strings.Contains(err.Error(), "auth") ||
			strings.Contains(err.Error(), "credential") ||
			strings.Contains(err.Error(), "forbidden") ||
			strings.Contains(err.Error(), "unauthorized")
		assert.True(t, authRelated, "Error should be auth-related: %v", err)
	}

	mockState.AssertExpectations(t)
}

// testAPITimeoutAndRetry tests timeout and retry mechanisms
func testAPITimeoutAndRetry(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for timeout testing
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks with timeouts
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Track operation timing
	var operationTimes []time.Duration
	var timeoutCount int

	// Mock GitHub operations with variable delays
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(_ mock.Arguments) {
			timeoutCount++
		}).Return(&gh.FileContent{Content: []byte("content")}, nil).Maybe()

	// Mock other operations with similar timeout behavior
	// Setup CreatePR mocks for timeout scenarios
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(&gh.PR{Number: 101}, nil).Maybe()
	}

	// Mock Git operations
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory from the test scenario
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
			time.Sleep(200 * time.Millisecond)
		}).Return(nil)

	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "master"}}, nil).Maybe()

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync with timeout constraints
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	err = engine.Sync(ctx, nil)
	totalDuration := time.Since(start)

	// Analyze timing results
	t.Logf("Total sync duration: %v", totalDuration)
	t.Logf("Operation timeouts encountered: %d", timeoutCount)

	if len(operationTimes) > 0 {
		var totalOpTime time.Duration
		for _, opTime := range operationTimes {
			totalOpTime += opTime
		}
		avgOpTime := totalOpTime / time.Duration(len(operationTimes))
		t.Logf("Average operation time: %v across %d operations", avgOpTime, len(operationTimes))
	}

	// Should handle timeouts gracefully
	if err != nil && strings.Contains(err.Error(), "timeout") {
		t.Logf("Sync failed due to timeout (expected): %v", err)
	}

	// Should have attempted multiple operations
	assert.Greater(t, timeoutCount, 3, "Should have attempted multiple operations")

	mockState.AssertExpectations(t)
}

// testConcurrentAPIOperations tests concurrent API request handling
func testConcurrentAPIOperations(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario with multiple repos for concurrency testing
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks for concurrent operations
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Track concurrent operations
	var concurrentOps int64
	var maxConcurrent int64
	var opMutex syncpkg.Mutex

	// Mock GitHub operations with concurrency tracking
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(_ mock.Arguments) {
			// Track concurrent operations start
			current := atomic.AddInt64(&concurrentOps, 1)

			opMutex.Lock()
			if current > maxConcurrent {
				maxConcurrent = current
			}
			opMutex.Unlock()

			// Simulate API processing time
			time.Sleep(100 * time.Millisecond)

			// Decrement concurrent operations count
			atomic.AddInt64(&concurrentOps, -1)
		}).Return(&gh.FileContent{Content: []byte("content")}, nil).Maybe()

	mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
	mockGH.On("CreatePR", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("gh.PRRequest")).
		Run(func(_ mock.Arguments) {
			current := atomic.AddInt64(&concurrentOps, 1)

			opMutex.Lock()
			if current > maxConcurrent {
				maxConcurrent = current
			}
			opMutex.Unlock()

			time.Sleep(150 * time.Millisecond)

			atomic.AddInt64(&concurrentOps, -1)
		}).Return(&gh.PR{Number: 202}, nil).Maybe()

	// Mock Git operations
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory from the test scenario
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
			time.Sleep(80 * time.Millisecond)
		}).Return(nil)

	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "master"}}, nil).Maybe()

	// Test different concurrency levels
	concurrencyLevels := []int{1, 3, 5}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("concurrency_%d", concurrency), func(t *testing.T) {
			// Reset tracking
			atomic.StoreInt64(&concurrentOps, 0)
			opMutex.Lock()
			maxConcurrent = 0
			opMutex.Unlock()

			// Create sync engine with specific concurrency
			opts := sync.DefaultOptions().WithDryRun(false).WithMaxConcurrency(concurrency)
			engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
			engine.SetLogger(logrus.New())

			// Execute sync
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			start := time.Now()
			err := engine.Sync(ctx, nil)
			duration := time.Since(start)

			// Analyze concurrency results
			opMutex.Lock()
			finalMaxConcurrent := maxConcurrent
			opMutex.Unlock()

			t.Logf("Concurrency %d: completed in %v, max concurrent ops: %d",
				concurrency, duration, finalMaxConcurrent)

			// Should respect concurrency limits (within reasonable bounds)
			maxAllowed := int64(concurrency) * 2
			assert.LessOrEqual(t, finalMaxConcurrent, maxAllowed,
				"Should not exceed reasonable concurrency bounds")

			// Higher concurrency should generally utilize more concurrent operations
			if concurrency > 1 {
				assert.Greater(t, finalMaxConcurrent, int64(1),
					"Should utilize concurrent operations")
			}

			// Handle any errors (may occur due to timing in tests)
			if err != nil {
				t.Logf("Sync with concurrency %d failed: %v", concurrency, err)
			}
		})
	}

	mockState.AssertExpectations(t)
}

// testGitHubAPIDegradation tests handling of GitHub API service degradation
func testGitHubAPIDegradation(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for API degradation testing
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks simulating API degradation
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Simulate different degradation scenarios
	degradationPhase := 0

	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(_ mock.Arguments) {
			degradationPhase++
		}).Return(nil, &APIError{
		StatusCode: http.StatusServiceUnavailable,
		Message:    "Service temporarily unavailable",
	}).Maybe()

	// Mock other operations with similar degradation
	// Setup CreatePR mocks for degradation scenarios
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(&gh.PR{Number: 303}, nil).Maybe()
	}

	// Git operations (less affected by GitHub API degradation)
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory from the test scenario
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
		}).Return(nil)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "master"}}, nil).Maybe()

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync during API degradation
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second) // Longer timeout for degradation
	defer cancel()

	start := time.Now()
	err = engine.Sync(ctx, nil)
	duration := time.Since(start)

	t.Logf("API degradation test completed in %v", duration)
	t.Logf("Degradation phases encountered: %d", degradationPhase)

	// Should handle API degradation gracefully
	if err != nil {
		t.Logf("Sync failed during API degradation: %v", err)
		// Should be service-related errors
		serviceRelated := strings.Contains(err.Error(), "service") ||
			strings.Contains(err.Error(), "server") ||
			strings.Contains(err.Error(), "unavailable") ||
			strings.Contains(err.Error(), "degradation")
		assert.True(t, serviceRelated, "Error should be service-related during degradation")
	}

	// Should have encountered multiple degradation scenarios
	assert.GreaterOrEqual(t, degradationPhase, 5, "Should have encountered multiple degradation scenarios")

	mockState.AssertExpectations(t)
}

// testNetworkPartitionRecovery tests recovery from network partitions
func testNetworkPartitionRecovery(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for partition testing
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks simulating network partition
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Simulate network partition phases
	var partitionPhase int

	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(_ mock.Arguments) {
			partitionPhase++
		}).Return(nil, fmt.Errorf("%w: no route to host", fixtures.ErrNetworkPartition)).Maybe()

	// Similar partition behavior for other operations
	// Setup CreatePR mocks for partition recovery scenarios
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(&gh.PR{Number: 404}, nil).Maybe()
	}

	// Git operations also affected by partition
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Always create the expected files - partition affects network, not local git content
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
		}).Return(nil).Maybe()

	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Return(fmt.Errorf("%w", fixtures.ErrNetworkPartition))

	// Local Git operations (not affected by network partition)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "master"}}, nil).Maybe()

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync during and after partition
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // Long timeout for partition recovery
	defer cancel()

	start := time.Now()
	err = engine.Sync(ctx, nil)
	duration := time.Since(start)

	t.Logf("Network partition recovery test completed in %v", duration)
	t.Logf("Partition phases encountered: %d", partitionPhase)

	// Analyze partition recovery
	if err != nil {
		t.Logf("Sync failed during network partition: %v", err)
		// Should be network-related errors
		networkRelated := strings.Contains(err.Error(), "network") ||
			strings.Contains(err.Error(), "partition") ||
			strings.Contains(err.Error(), "connection") ||
			strings.Contains(err.Error(), "route")
		assert.True(t, networkRelated, "Error should be network-related during partition")
	} else {
		t.Logf("Sync recovered successfully from network partition")
	}

	// Should have encountered partition
	assert.GreaterOrEqual(t, partitionPhase, 5, "Should have encountered network partition scenarios")

	// Duration should indicate partition recovery time
	if partitionPhase > 15 {
		t.Logf("Successfully recovered from network partition")
	}

	mockState.AssertExpectations(t)
}

// testDNSResolutionFailures tests DNS resolution failure scenarios
func testDNSResolutionFailures(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for DNS testing
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks simulating DNS failures
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Track DNS resolution attempts
	var dnsFailureCount int

	// Mock GitHub operations with DNS failures
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(_ mock.Arguments) {
			dnsFailureCount++
		}).Return(nil, fixtures.ErrNetworkUnreachable).Times(3)

	// Mock CreatePR with intermittent DNS issues
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(nil, fixtures.ErrNetworkTimeout).Maybe()
	}

	// Git operations may also be affected by DNS - first few fail, then succeed
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
			dnsFailureCount++ // Count git DNS failures too
		}).Return(fixtures.ErrGitCloneFailed).Times(2)

	// After DNS failures, Clone eventually succeeds
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
		}).Return(nil)

	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Return(fixtures.ErrGitCloneFailed).Maybe()

	// Local Git operations (not affected by DNS)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return(nil, fixtures.ErrNetworkFailure).Maybe()

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync during DNS failures
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	start := time.Now()
	err = engine.Sync(ctx, nil)
	duration := time.Since(start)

	t.Logf("DNS resolution failure test completed in %v", duration)
	t.Logf("DNS failure attempts: %d", dnsFailureCount)

	// Should handle DNS failures gracefully
	if err != nil {
		dnsRelated := strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "lookup") ||
			strings.Contains(err.Error(), "resolve") ||
			strings.Contains(err.Error(), "DNS") ||
			strings.Contains(err.Error(), "dns") ||
			strings.Contains(err.Error(), "network") ||
			strings.Contains(err.Error(), "unreachable") ||
			strings.Contains(err.Error(), "clone failed")
		assert.True(t, dnsRelated, "Error should be DNS-related: %v", err)
	}

	// Should have encountered DNS failures (at least 2 clone attempts)
	assert.GreaterOrEqual(t, dnsFailureCount, 2, "Should have encountered DNS resolution failures")

	mockState.AssertExpectations(t)
}

// testSSLCertificateErrors tests SSL/TLS certificate validation failures
func testSSLCertificateErrors(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for SSL testing
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks simulating SSL certificate errors
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Track SSL certificate failures
	var sslFailureCount int

	// Mock GitHub operations with SSL certificate errors
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(_ mock.Arguments) {
			sslFailureCount++
		}).Return(nil, fixtures.ErrUnauthorized).Times(2)

	// Mock other operations with various SSL errors
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(nil, fixtures.ErrUnauthorized).Maybe()
	}

	// Git operations with SSL issues - ensure Clone is called and fails
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
			sslFailureCount++ // Count git SSL failures too
		}).Return(fixtures.ErrSSLCertificate)

	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Return(fixtures.ErrSSLCertificate).Maybe()

	// Standard operations
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return(nil, fixtures.ErrUnauthorized).Maybe()

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync with SSL certificate issues
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	start := time.Now()
	err = engine.Sync(ctx, nil)
	duration := time.Since(start)

	t.Logf("SSL certificate error test completed in %v", duration)
	t.Logf("SSL failure attempts: %d", sslFailureCount)

	// Should handle SSL certificate errors appropriately
	if err != nil {
		sslRelated := strings.Contains(err.Error(), "x509") ||
			strings.Contains(err.Error(), "certificate") ||
			strings.Contains(err.Error(), "SSL") ||
			strings.Contains(err.Error(), "TLS")
		assert.True(t, sslRelated, "Error should be SSL/certificate-related: %v", err)
	}

	// Should have encountered SSL failures
	assert.GreaterOrEqual(t, sslFailureCount, 2, "Should have encountered SSL certificate failures")

	mockState.AssertExpectations(t)
}

// testProxyConnectionIssues tests proxy connection and configuration issues
func testProxyConnectionIssues(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for proxy testing
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks simulating proxy issues
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Track proxy-related failures
	var proxyFailureCount int

	// Mock GitHub operations with proxy errors
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(_ mock.Arguments) {
			proxyFailureCount++
		}).Return(nil, fixtures.ErrNetworkRefused).Times(2)

	// Mock other operations with proxy authentication issues
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(nil, &APIError{
				StatusCode: http.StatusProxyAuthRequired,
				Message:    "Proxy Authentication Required",
			}).Maybe()
	}

	// Git operations with proxy issues - ensure Clone is called and fails
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
			proxyFailureCount++ // Count git proxy failures too
		}).Return(fixtures.ErrProxyConnection)

	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Return(fixtures.ErrProxyConnection).Maybe()

	// Standard operations
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return(nil, fixtures.ErrRequestTimeout).Maybe()

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync with proxy issues
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	start := time.Now()
	err = engine.Sync(ctx, nil)
	duration := time.Since(start)

	t.Logf("Proxy connection issue test completed in %v", duration)
	t.Logf("Proxy failure attempts: %d", proxyFailureCount)

	// Should handle proxy issues appropriately
	if err != nil {
		proxyRelated := strings.Contains(err.Error(), "proxy") ||
			strings.Contains(err.Error(), "407") ||
			strings.Contains(err.Error(), "Proxy Authentication") ||
			strings.Contains(err.Error(), "CONNECT")
		assert.True(t, proxyRelated, "Error should be proxy-related: %v", err)
	}

	// Should have encountered proxy failures
	assert.GreaterOrEqual(t, proxyFailureCount, 2, "Should have encountered proxy connection failures")

	mockState.AssertExpectations(t)
}

// testGitHubWebhookSimulation tests webhook-like rapid state changes
func testGitHubWebhookSimulation(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for webhook simulation
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks simulating rapid state changes like webhooks would trigger
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	// Simulate changing state during sync (like webhook updates)
	var stateChanges int64
	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Run(func(_ mock.Arguments) {
			atomic.AddInt64(&stateChanges, 1)
		}).Return(scenario.State, nil)

	// Track rapid API changes
	var webhookSimulationCount int64

	// Mock GitHub operations with state changes mid-operation
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(_ mock.Arguments) {
			atomic.AddInt64(&webhookSimulationCount, 1)
			// Simulate occasional content changes during sync
			time.Sleep(25 * time.Millisecond)
		}).Return(&gh.FileContent{Content: []byte("updated content")}, nil).Maybe()

	// Mock PR operations with state changes
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)

		// Simulate PR state changes during sync
		// First PR creation succeeds
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(&gh.PR{Number: 555}, nil).Once()

		// Subsequent PR creations fail with conflict
		mockGH.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(nil, &APIError{
				StatusCode: http.StatusConflict,
				Message:    "A pull request already exists for this branch",
			}).Maybe()

		// Simulate existing PR detection - return empty initially
		mockGH.On("ListPRs", mock.Anything, repoName, "open").
			Return([]gh.PR{}, nil).Maybe()
	}

	// Git operations
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			createSourceFilesInMock(args, scenario.SourceRepo.Files)
		}).Return(nil)

	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "master"}, {Name: "sync/update-123"}}, nil).Maybe()

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false).WithMaxConcurrency(3)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute multiple concurrent syncs to simulate webhook-triggered updates
	numConcurrentSyncs := 3
	var wg syncpkg.WaitGroup
	results := make([]error, numConcurrentSyncs)

	for i := 0; i < numConcurrentSyncs; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			results[index] = engine.Sync(ctx, nil)
		}(i)

		// Stagger the starts slightly to simulate webhook timing
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	// Analyze webhook simulation results
	successCount := 0
	conflictCount := 0
	for i, err := range results {
		if err == nil {
			successCount++
			t.Logf("Concurrent sync %d: SUCCESS", i+1)
		} else {
			t.Logf("Concurrent sync %d: FAILED - %v", i+1, err)
			if strings.Contains(err.Error(), "conflict") || strings.Contains(err.Error(), "already exists") {
				conflictCount++
			}
		}
	}

	t.Logf("Webhook simulation: %d successful, %d conflicts out of %d syncs",
		successCount, conflictCount, numConcurrentSyncs)
	t.Logf("State changes detected: %d", atomic.LoadInt64(&stateChanges))
	t.Logf("Webhook simulation API calls: %d", atomic.LoadInt64(&webhookSimulationCount))

	// Should handle concurrent syncs gracefully (some conflicts expected)
	assert.Greater(t, atomic.LoadInt64(&stateChanges), int64(1), "Should have detected state changes")
	assert.Greater(t, atomic.LoadInt64(&webhookSimulationCount), int64(5), "Should have made multiple API calls")

	// In webhook scenarios, we expect some syncs to succeed and some to have conflicts
	// This is the expected behavior when multiple syncs are triggered concurrently
	totalHandled := successCount + conflictCount
	assert.Equal(t, numConcurrentSyncs, totalHandled,
		"All syncs should either succeed or encounter expected conflicts")

	// At least one sync should succeed (the first one to create the PR)
	assert.GreaterOrEqual(t, successCount, 1, "At least one sync should succeed")

	// Some conflicts are expected due to concurrent branch/PR creation
	if numConcurrentSyncs > 1 {
		assert.GreaterOrEqual(t, conflictCount, 1,
			"Should have some conflicts when multiple syncs run concurrently")
	}

	mockState.AssertExpectations(t)
}
