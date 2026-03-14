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

	now := time.Now()
	dependabotAlerts := []gh.DependabotAlert{
		{
			Number:            1,
			State:             "open",
			DependencyPackage: "lodash",
			HTMLURL:           "https://github.com/test/repo/security/dependabot/1",
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}
	dependabotAlerts[0].SecurityVulnerability.Severity = "high"

	codeScanningAlerts := []gh.CodeScanningAlert{
		{
			Number:    2,
			State:     "open",
			HTMLURL:   "https://github.com/test/repo/security/code-scanning/2",
			CreatedAt: now.Add(-2 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour),
		},
	}
	codeScanningAlerts[0].Rule.ID = "go/sql-injection"
	codeScanningAlerts[0].Rule.Severity = "error"
	codeScanningAlerts[0].Rule.Description = "SQL injection vulnerability"

	secretScanningAlerts := []gh.SecretScanningAlert{
		{
			Number:                3,
			State:                 "open",
			SecretType:            "github_pat",
			SecretTypeDisplayName: "GitHub PAT",
			HTMLURL:               "https://github.com/test/repo/security/secret-scanning/3",
			CreatedAt:             now.Add(-6 * time.Hour),
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

	result := results["test/repo"]
	require.Len(t, result.Alerts, 3, "should have all three alert types")
	assert.Empty(t, result.Warnings, "no warnings expected when REST works")

	// Verify alert types
	assert.Equal(t, AlertTypeDependabot, result.Alerts[0].AlertType)
	assert.Equal(t, AlertTypeCodeScanning, result.Alerts[1].AlertType)
	assert.Equal(t, AlertTypeSecretScanning, result.Alerts[2].AlertType)

	// Verify alert details
	assert.Equal(t, 1, result.Alerts[0].AlertNumber)
	assert.Equal(t, "open", result.Alerts[0].State)
	assert.Equal(t, "high", result.Alerts[0].Severity)

	mockClient.AssertExpectations(t)
}

// TestSecurityCollector_CollectAlerts_GraphQLFallback tests that 404 triggers GraphQL fallback
func TestSecurityCollector_CollectAlerts_GraphQLFallback(t *testing.T) {
	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	now := time.Now()

	// REST Dependabot returns 404 (ErrSecurityNotAvailable)
	mockClient.On("GetDependabotAlerts", mock.Anything, "org/repo").
		Return(nil, gh.ErrSecurityNotAvailable)

	// GraphQL fallback succeeds
	graphqlAlerts := []gh.VulnerabilityAlert{
		{
			Number:            1,
			State:             "OPEN",
			CreatedAt:         now,
			Severity:          "MODERATE",
			PackageName:       "github.com/pion/dtls/v2",
			PackageEcosystem:  "GO",
			AdvisorySummary:   "Pion DTLS nonce reuse",
			AdvisoryPermalink: "https://github.com/advisories/GHSA-xxxx",
		},
	}
	mockClient.On("GetVulnerabilityAlertsGraphQL", mock.Anything, "org/repo").
		Return(graphqlAlerts, nil)

	// Other REST endpoints also 404
	mockClient.On("GetCodeScanningAlerts", mock.Anything, "org/repo").
		Return(nil, gh.ErrSecurityNotAvailable)
	mockClient.On("GetSecretScanningAlerts", mock.Anything, "org/repo").
		Return(nil, gh.ErrSecurityNotAvailable)

	ctx := context.Background()
	repos := []gh.RepoInfo{{FullName: "org/repo"}}
	results, err := collector.CollectAlerts(ctx, repos)

	require.NoError(t, err)
	require.Contains(t, results, "org/repo")

	result := results["org/repo"]
	require.Len(t, result.Alerts, 1, "should have GraphQL vulnerability alert")

	// Verify the alert was properly converted
	alert := result.Alerts[0]
	assert.Equal(t, AlertTypeDependabot, alert.AlertType)
	assert.Equal(t, "open", alert.State, "state should be lowercased from GraphQL")
	assert.Equal(t, "medium", alert.Severity, "MODERATE should normalize to medium")
	assert.Contains(t, alert.Title, "github.com/pion/dtls/v2")

	// Verify warnings were generated
	assert.NotEmpty(t, result.Warnings, "should have warnings about REST 404 and GraphQL fallback")

	// Should mention GraphQL fallback
	foundFallbackWarning := false
	for _, w := range result.Warnings {
		if assert.ObjectsAreEqual("dependabot: REST API unavailable (404), used GraphQL fallback (1 alerts found)", w) {
			foundFallbackWarning = true
		}
	}
	assert.True(t, foundFallbackWarning, "should warn about GraphQL fallback usage")

	mockClient.AssertExpectations(t)
}

// TestSecurityCollector_CollectAlerts_GraphQLFallbackFails tests when both REST and GraphQL fail
func TestSecurityCollector_CollectAlerts_GraphQLFallbackFails(t *testing.T) {
	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	// REST returns 404
	mockClient.On("GetDependabotAlerts", mock.Anything, "org/repo").
		Return(nil, gh.ErrSecurityNotAvailable)
	// GraphQL also fails
	mockClient.On("GetVulnerabilityAlertsGraphQL", mock.Anything, "org/repo").
		Return(nil, errMockAPI)
	mockClient.On("GetCodeScanningAlerts", mock.Anything, "org/repo").
		Return([]gh.CodeScanningAlert{}, nil)
	mockClient.On("GetSecretScanningAlerts", mock.Anything, "org/repo").
		Return([]gh.SecretScanningAlert{}, nil)

	ctx := context.Background()
	repos := []gh.RepoInfo{{FullName: "org/repo"}}
	results, err := collector.CollectAlerts(ctx, repos)

	require.NoError(t, err)
	require.Contains(t, results, "org/repo")

	result := results["org/repo"]
	assert.Empty(t, result.Alerts)
	assert.NotEmpty(t, result.Warnings)

	// Should mention both REST and GraphQL failure
	foundWarning := false
	for _, w := range result.Warnings {
		if assert.ObjectsAreEqual("dependabot: REST API returned 404, GraphQL fallback also failed: mock API error", w) {
			foundWarning = true
		}
	}
	assert.True(t, foundWarning, "should warn about both failures")

	mockClient.AssertExpectations(t)
}

// TestSecurityCollector_CollectAlerts_MultipleRepos tests concurrent collection
func TestSecurityCollector_CollectAlerts_MultipleRepos(t *testing.T) {
	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	// Setup mocks for 5 repos with no alerts
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
	assert.Len(t, results, 5, "all repos should have results")

	// All should have empty alerts and no warnings
	for _, repo := range repos {
		result := results[repo.FullName]
		require.NotNil(t, result)
		assert.Empty(t, result.Alerts)
		assert.Empty(t, result.Warnings)
	}

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

	// repo2: dependabot fails with real error (not 404)
	mockClient.On("GetDependabotAlerts", mock.Anything, "org/repo2").Return(nil, errMockAPI)
	mockClient.On("GetCodeScanningAlerts", mock.Anything, "org/repo2").Return([]gh.CodeScanningAlert{}, nil)
	mockClient.On("GetSecretScanningAlerts", mock.Anything, "org/repo2").Return([]gh.SecretScanningAlert{}, nil)

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

	// Should not return error - partial failures are captured as warnings
	require.NoError(t, err)

	// All repos should have results
	require.Len(t, results, 3)

	// repo1 should have alerts, no warnings
	assert.Len(t, results["org/repo1"].Alerts, 1)
	assert.Empty(t, results["org/repo1"].Warnings)

	// repo2 should have warnings about the failure
	assert.Empty(t, results["org/repo2"].Alerts)
	assert.NotEmpty(t, results["org/repo2"].Warnings)

	// repo3 should have alerts, no warnings
	assert.Len(t, results["org/repo3"].Alerts, 1)
	assert.Empty(t, results["org/repo3"].Warnings)

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

	// Create 20 repos to exceed worker limit
	repos := make([]gh.RepoInfo, 20)
	for i := 0; i < 20; i++ {
		repos[i] = gh.RepoInfo{FullName: "org/repo" + string(rune('a'+i))}
	}

	// Setup mocks with artificial delay to test concurrency
	for _, repo := range repos {
		repoName := repo.FullName
		mockClient.On("GetDependabotAlerts", mock.Anything, repoName).Run(func(_ mock.Arguments) {
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
		repos[i] = gh.RepoInfo{FullName: "org/repo" + string(rune('a'+i))}
	}

	// Create context that will be canceled immediately
	ctx, cancel := context.WithCancel(context.Background())

	// Setup mocks with delay and context checking
	for _, repo := range repos {
		repoName := repo.FullName
		mockClient.On("GetDependabotAlerts", mock.Anything, repoName).Run(func(args mock.Arguments) {
			// Check context before sleeping
			callCtx := args.Get(0).(context.Context)
			select {
			case <-callCtx.Done():
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

	// Both repos should be in results now
	require.Len(t, results, 2)

	// repo-with-alerts should have alerts
	assert.Len(t, results["org/repo-with-alerts"].Alerts, 1)

	// repo-without-alerts should have empty alerts
	assert.Empty(t, results["org/repo-without-alerts"].Alerts)

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
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	t.Parallel()

	mockClient := new(gh.MockClient)
	collector := NewSecurityCollector(mockClient, nil, nil)

	// Create many repos to maximize concurrent access
	repos := make([]gh.RepoInfo, 50)
	for i := 0; i < 50; i++ {
		repos[i] = gh.RepoInfo{FullName: "org/repo" + string(rune('a'+i))}
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

	// Each should have exactly 1 alert
	for _, repo := range repos {
		assert.Len(t, results[repo.FullName].Alerts, 1)
	}

	mockClient.AssertExpectations(t)
}

// TestNormalizeSeverity tests severity normalization from GraphQL to REST format
func TestNormalizeSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"CRITICAL", "critical"},
		{"HIGH", "high"},
		{"MODERATE", "medium"},
		{"LOW", "low"},
		{"critical", "critical"},
		{"high", "high"},
		{"moderate", "medium"},
		{"low", "low"},
		{"UNKNOWN", "unknown"},
		{"", ""},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, normalizeSeverity(tt.input), "normalizeSeverity(%q)", tt.input)
	}
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
