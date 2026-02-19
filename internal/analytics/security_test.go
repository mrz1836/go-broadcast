package analytics

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

var errMockAPI = errors.New("mock API error")

// TestSecurityCollector_CollectAlerts_EmptyRepos tests empty repo list
func TestSecurityCollector_CollectAlerts_EmptyRepos(t *testing.T) {
	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	ctx := context.Background()
	results, err := collector.CollectAlerts(ctx, []gh.RepoInfo{})

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Empty(t, results, "empty repo list should return empty map")
}

// TestSecurityCollector_CollectAlerts_SingleRepo tests single repo collection
func TestSecurityCollector_CollectAlerts_SingleRepo(t *testing.T) {
	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	// Setup mock responses for all three alert types
	dependabotAlert := gh.DependabotAlert{
		Number:            1,
		State:             "open",
		DependencyPackage: "test-package",
		HTMLURL:           "https://github.com/test/repo/security/dependabot/1",
		CreatedAt:         time.Now().Add(-24 * time.Hour),
		UpdatedAt:         time.Now(),
	}
	dependabotAlert.SecurityVulnerability.Severity = "high"
	dependabotAlerts := []gh.DependabotAlert{dependabotAlert}

	codeScanningAlert := gh.CodeScanningAlert{
		Number:    2,
		State:     "open",
		HTMLURL:   "https://github.com/test/repo/security/code-scanning/2",
		CreatedAt: time.Now().Add(-12 * time.Hour),
		UpdatedAt: time.Now(),
	}
	codeScanningAlert.Rule.ID = "test-rule"
	codeScanningAlert.Rule.Severity = "medium"
	codeScanningAlert.Rule.Description = "Test security issue"
	codeScanningAlerts := []gh.CodeScanningAlert{codeScanningAlert}

	secretScanningAlerts := []gh.SecretScanningAlert{
		{ //nolint:gosec // G101: test data with fake credentials, not real secrets
			Number:                3,
			State:                 "open",
			SecretTypeDisplayName: "GitHub Token",
			HTMLURL:               "https://github.com/test/repo/security/secret-scanning/3",
			CreatedAt:             time.Now().Add(-6 * time.Hour),
		},
	}

	mockClient.On("GetDependabotAlerts", mock.Anything, "test/repo").Return(dependabotAlerts, nil)
	mockClient.On("GetCodeScanningAlerts", mock.Anything, "test/repo").Return(codeScanningAlerts, nil)
	mockClient.On("GetSecretScanningAlerts", mock.Anything, "test/repo").Return(secretScanningAlerts, nil)

	ctx := context.Background()
	repos := []gh.RepoInfo{{FullName: "test/repo"}}
	results, err := collector.CollectAlerts(ctx, repos)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Contains(t, results, "test/repo")

	alerts := results["test/repo"]
	require.Len(t, alerts, 3, "should have all three alert types")

	// Verify alert types
	assert.Equal(t, AlertTypeDependabot, alerts[0].AlertType)
	assert.Equal(t, AlertTypeCodeScanning, alerts[1].AlertType)
	assert.Equal(t, AlertTypeSecretScanning, alerts[2].AlertType)

	// Verify alert details
	assert.Equal(t, 1, alerts[0].AlertNumber)
	assert.Equal(t, "open", alerts[0].State)
	assert.Equal(t, "high", alerts[0].Severity)

	mockClient.AssertExpectations(t)
}

// TestSecurityCollector_CollectAlerts_MultipleRepos tests concurrent collection
func TestSecurityCollector_CollectAlerts_MultipleRepos(t *testing.T) {
	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	// Setup mocks for 5 repos
	repos := []gh.RepoInfo{
		{FullName: "org/repo1"},
		{FullName: "org/repo2"},
		{FullName: "org/repo3"},
		{FullName: "org/repo4"},
		{FullName: "org/repo5"},
	}

	for _, repo := range repos {
		mockClient.On("GetDependabotAlerts", mock.Anything, repo.FullName).Return([]gh.DependabotAlert{}, nil)
		mockClient.On("GetCodeScanningAlerts", mock.Anything, repo.FullName).Return([]gh.CodeScanningAlert{}, nil)
		mockClient.On("GetSecretScanningAlerts", mock.Anything, repo.FullName).Return([]gh.SecretScanningAlert{}, nil)
	}

	ctx := context.Background()
	results, err := collector.CollectAlerts(ctx, repos)

	require.NoError(t, err)
	assert.Empty(t, results, "repos with no alerts should not appear in results")

	mockClient.AssertExpectations(t)
}

// TestSecurityCollector_CollectAlerts_PartialFailure tests partial failure tolerance
func TestSecurityCollector_CollectAlerts_PartialFailure(t *testing.T) {
	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	// Setup: repo1 succeeds, repo2 fails, repo3 succeeds
	repos := []gh.RepoInfo{
		{FullName: "org/repo1"},
		{FullName: "org/repo2"},
		{FullName: "org/repo3"},
	}

	// repo1: success with alerts
	alert1 := gh.DependabotAlert{
		Number:            1,
		State:             "open",
		DependencyPackage: "pkg",
		HTMLURL:           "url",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	alert1.SecurityVulnerability.Severity = "high"
	mockClient.On("GetDependabotAlerts", mock.Anything, "org/repo1").Return([]gh.DependabotAlert{alert1}, nil)
	mockClient.On("GetCodeScanningAlerts", mock.Anything, "org/repo1").Return([]gh.CodeScanningAlert{}, nil)
	mockClient.On("GetSecretScanningAlerts", mock.Anything, "org/repo1").Return([]gh.SecretScanningAlert{}, nil)

	// repo2: fails
	mockClient.On("GetDependabotAlerts", mock.Anything, "org/repo2").Return(nil, errMockAPI)

	// repo3: success with alerts
	alert3 := gh.DependabotAlert{
		Number:            2,
		State:             "open",
		DependencyPackage: "pkg2",
		HTMLURL:           "url2",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	alert3.SecurityVulnerability.Severity = "medium"
	mockClient.On("GetDependabotAlerts", mock.Anything, "org/repo3").Return([]gh.DependabotAlert{alert3}, nil)
	mockClient.On("GetCodeScanningAlerts", mock.Anything, "org/repo3").Return([]gh.CodeScanningAlert{}, nil)
	mockClient.On("GetSecretScanningAlerts", mock.Anything, "org/repo3").Return([]gh.SecretScanningAlert{}, nil)

	ctx := context.Background()
	results, err := collector.CollectAlerts(ctx, repos)

	// Should not return error - partial failures are tolerated
	require.NoError(t, err)

	// Should have results for repo1 and repo3, but not repo2
	require.Len(t, results, 2)
	assert.Contains(t, results, "org/repo1")
	assert.Contains(t, results, "org/repo3")
	assert.NotContains(t, results, "org/repo2", "failed repo should not be in results")

	mockClient.AssertExpectations(t)
}

// TestSecurityCollector_CollectAlerts_WorkerPoolLimit tests concurrency limiting
func TestSecurityCollector_CollectAlerts_WorkerPoolLimit(t *testing.T) {
	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	// Track concurrent execution
	var activeWorkers int32
	var maxWorkers int32

	// Create 20 repos to exceed worker limit of 10
	repos := make([]gh.RepoInfo, 20)
	for i := 0; i < 20; i++ {
		repos[i] = gh.RepoInfo{FullName: "org/repo" + string(rune(i))}
	}

	// Setup mocks with artificial delay to test concurrency
	for _, repo := range repos {
		repoName := repo.FullName
		mockClient.On("GetDependabotAlerts", mock.Anything, repoName).Run(func(args mock.Arguments) {
			// Track concurrency
			current := atomic.AddInt32(&activeWorkers, 1)
			defer atomic.AddInt32(&activeWorkers, -1)

			// Update max if needed
			for {
				currentMax := atomic.LoadInt32(&maxWorkers)
				if current <= currentMax || atomic.CompareAndSwapInt32(&maxWorkers, currentMax, current) {
					break
				}
			}

			// Simulate work
			time.Sleep(10 * time.Millisecond)
		}).Return([]gh.DependabotAlert{}, nil)

		mockClient.On("GetCodeScanningAlerts", mock.Anything, repoName).Return([]gh.CodeScanningAlert{}, nil)
		mockClient.On("GetSecretScanningAlerts", mock.Anything, repoName).Return([]gh.SecretScanningAlert{}, nil)
	}

	ctx := context.Background()
	_, err := collector.CollectAlerts(ctx, repos)

	require.NoError(t, err)

	// Verify worker pool limit was respected
	maxConcurrent := atomic.LoadInt32(&maxWorkers)
	assert.LessOrEqual(t, maxConcurrent, int32(SecurityWorkerLimit), "should not exceed SecurityWorkerLimit")
	assert.Greater(t, maxConcurrent, int32(1), "should use concurrent workers")

	mockClient.AssertExpectations(t)
}

// TestSecurityCollector_CollectAlerts_ContextCancellation tests context cancellation
func TestSecurityCollector_CollectAlerts_ContextCancellation(t *testing.T) {
	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	// Create many repos to ensure some are still running when context is canceled
	repos := make([]gh.RepoInfo, 30)
	for i := 0; i < 30; i++ {
		repos[i] = gh.RepoInfo{FullName: "org/repo" + string(rune(i))}
	}

	// Create context that will be canceled immediately
	ctx, cancel := context.WithCancel(context.Background())

	// Setup mocks with delay and context checking
	for _, repo := range repos {
		repoName := repo.FullName
		mockClient.On("GetDependabotAlerts", mock.Anything, repoName).Run(func(args mock.Arguments) {
			// Check context before sleeping
			ctx := args.Get(0).(context.Context)
			select {
			case <-ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
				// Simulate work
			}
		}).Return([]gh.DependabotAlert{}, nil).Maybe()

		mockClient.On("GetCodeScanningAlerts", mock.Anything, repoName).Return([]gh.CodeScanningAlert{}, nil).Maybe()
		mockClient.On("GetSecretScanningAlerts", mock.Anything, repoName).Return([]gh.SecretScanningAlert{}, nil).Maybe()
	}

	// Cancel context very quickly to ensure some workers are interrupted
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := collector.CollectAlerts(ctx, repos)
	// Should return error due to context cancellation (or complete successfully on very fast machines)
	// The important thing is that it doesn't hang or panic
	if err != nil {
		assert.Contains(t, err.Error(), "context")
	}
}

// TestSecurityCollector_CollectAlerts_OnlyReposWithAlerts tests filtering
func TestSecurityCollector_CollectAlerts_OnlyReposWithAlerts(t *testing.T) {
	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	repos := []gh.RepoInfo{
		{FullName: "org/repo-with-alerts"},
		{FullName: "org/repo-without-alerts"},
	}

	// repo-with-alerts has one alert
	alertWithAlerts := gh.DependabotAlert{
		Number:            1,
		State:             "open",
		DependencyPackage: "pkg",
		HTMLURL:           "url",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	alertWithAlerts.SecurityVulnerability.Severity = "high"
	mockClient.On("GetDependabotAlerts", mock.Anything, "org/repo-with-alerts").Return([]gh.DependabotAlert{alertWithAlerts}, nil)
	mockClient.On("GetCodeScanningAlerts", mock.Anything, "org/repo-with-alerts").Return([]gh.CodeScanningAlert{}, nil)
	mockClient.On("GetSecretScanningAlerts", mock.Anything, "org/repo-with-alerts").Return([]gh.SecretScanningAlert{}, nil)

	// repo-without-alerts has no alerts
	mockClient.On("GetDependabotAlerts", mock.Anything, "org/repo-without-alerts").Return([]gh.DependabotAlert{}, nil)
	mockClient.On("GetCodeScanningAlerts", mock.Anything, "org/repo-without-alerts").Return([]gh.CodeScanningAlert{}, nil)
	mockClient.On("GetSecretScanningAlerts", mock.Anything, "org/repo-without-alerts").Return([]gh.SecretScanningAlert{}, nil)

	ctx := context.Background()
	results, err := collector.CollectAlerts(ctx, repos)

	require.NoError(t, err)

	// Only repo with alerts should be in results
	require.Len(t, results, 1)
	assert.Contains(t, results, "org/repo-with-alerts")
	assert.NotContains(t, results, "org/repo-without-alerts")

	mockClient.AssertExpectations(t)
}

// TestSecurityAlert_Types tests alert type constants
func TestSecurityAlert_Types(t *testing.T) {
	t.Parallel()

	assert.Equal(t, AlertTypeDependabot, SecurityAlertType("dependabot"))
	assert.Equal(t, AlertTypeCodeScanning, SecurityAlertType("code_scanning"))
	assert.Equal(t, AlertTypeSecretScanning, SecurityAlertType("secret_scanning"))
}

// TestSecurityCollector_Constants tests security collector constants
func TestSecurityCollector_Constants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3, SecurityWorkerLimit, "worker limit should be 3")
	assert.Equal(t, 100, RateLimitThreshold, "rate limit threshold should be 100")
}

// TestSecurityCollector_CollectAlerts_RaceCondition tests for race conditions
func TestSecurityCollector_CollectAlerts_RaceCondition(t *testing.T) {
	// Skip in short mode as this test is slower with race detector
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	// Create many repos to maximize concurrent access
	repos := make([]gh.RepoInfo, 50)
	for i := 0; i < 50; i++ {
		repos[i] = gh.RepoInfo{FullName: "org/repo" + string(rune(i))}
	}

	// Setup mocks
	for _, repo := range repos {
		repoName := repo.FullName
		raceAlert := gh.DependabotAlert{
			Number:            1,
			State:             "open",
			DependencyPackage: "pkg",
			HTMLURL:           "url",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		raceAlert.SecurityVulnerability.Severity = "high"
		mockClient.On("GetDependabotAlerts", mock.Anything, repoName).Return([]gh.DependabotAlert{raceAlert}, nil)
		mockClient.On("GetCodeScanningAlerts", mock.Anything, repoName).Return([]gh.CodeScanningAlert{}, nil)
		mockClient.On("GetSecretScanningAlerts", mock.Anything, repoName).Return([]gh.SecretScanningAlert{}, nil)
	}

	ctx := context.Background()
	results, err := collector.CollectAlerts(ctx, repos)

	require.NoError(t, err)
	assert.Len(t, results, 50, "all repos should have results")

	// Run with -race flag to detect data races
	mockClient.AssertExpectations(t)
}

func TestFormatTimePtr(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()

		result := formatTimePtr(nil)
		assert.Nil(t, result)
	})

	t.Run("valid time returns ISO 8601 string", func(t *testing.T) {
		t.Parallel()

		ts := time.Date(2026, 2, 19, 15, 30, 45, 0, time.UTC)
		result := formatTimePtr(&ts)
		require.NotNil(t, result)
		assert.Equal(t, "2026-02-19T15:30:45Z", *result)
	})

	t.Run("zero time returns formatted string", func(t *testing.T) {
		t.Parallel()

		ts := time.Time{}
		result := formatTimePtr(&ts)
		require.NotNil(t, result)
		assert.NotEmpty(t, *result)
	})
}
