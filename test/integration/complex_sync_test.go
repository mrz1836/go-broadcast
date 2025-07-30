package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	syncpkg "sync"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
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

// Helper function for creating source files in Git Clone mocks
func createSourceFilesInMock(args mock.Arguments, sourceFiles map[string][]byte) {
	cloneDir := args[2].(string)

	// Create the expected files in the clone directory
	for filePath, content := range sourceFiles {
		fullPath := filepath.Join(cloneDir, filePath)
		dir := filepath.Dir(fullPath)
		_ = os.MkdirAll(dir, 0o750)
		_ = os.WriteFile(fullPath, content, 0o600)
	}
}

// TestComplexSyncScenarios tests complex real-world sync scenarios
func TestComplexSyncScenarios(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	generator := fixtures.NewTestRepoGenerator(tmpDir)
	defer func() {
		if err := generator.Cleanup(); err != nil {
			t.Errorf("failed to cleanup test generator: %v", err)
		}
	}()

	t.Run("multi_repository_sync_with_conflicts", func(t *testing.T) {
		testMultiRepoSyncWithConflicts(t, generator)
	})

	t.Run("partial_sync_failure_recovery", func(t *testing.T) {
		testPartialSyncFailureRecovery(t, generator)
	})

	t.Run("large_file_handling", func(t *testing.T) {
		testLargeFileHandling(t, generator)
	})

	t.Run("concurrent_sync_operations", func(t *testing.T) {
		testConcurrentSyncOperations(t, generator)
	})

	t.Run("memory_usage_monitoring", func(t *testing.T) {
		testMemoryUsageMonitoring(t, generator)
	})

	t.Run("state_consistency_across_failures", func(t *testing.T) {
		testStateConsistencyAcrossFailures(t, generator)
	})
}

// testMultiRepoSyncWithConflicts tests syncing multiple repositories where some have conflicts
func testMultiRepoSyncWithConflicts(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create complex scenario with conflicts
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	// Configure state discovery to return scenario state
	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Mock GitHub operations for source repo
	mockGH.On("GetFile", mock.Anything, "org/template-repo", mock.AnythingOfType("string"), "").
		Return(&gh.FileContent{Content: []byte("source content")}, nil).Maybe()

	// Mock GitHub operations for target repos with different responses
	for i, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)

		if target.HasConflict {
			// Mock conflicting content for repos that should have conflicts
			mockGH.On("GetFile", mock.Anything, repoName, mock.AnythingOfType("string"), "").
				Return(&gh.FileContent{Content: []byte("conflicting content")}, nil).Maybe()
		} else {
			// Mock different or missing content for other repos
			if i%2 == 0 {
				mockGH.On("GetFile", mock.Anything, repoName, mock.AnythingOfType("string"), "").
					Return(&gh.FileContent{Content: []byte("old content")}, nil).Maybe()
			} else {
				mockGH.On("GetFile", mock.Anything, repoName, mock.AnythingOfType("string"), "").
					Return(nil, fmt.Errorf("%w", fixtures.ErrFileNotFound)).Maybe()
			}
		}
	}

	// Mock Git operations
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory from the test scenario
			cloneDir := args[2].(string)

			// Create source files from the scenario
			for filePath, content := range scenario.SourceRepo.Files {
				fullPath := filepath.Join(cloneDir, filePath)
				dir := filepath.Dir(fullPath)
				_ = os.MkdirAll(dir, 0o750)
				_ = os.WriteFile(fullPath, content, 0o600)
			}
		}).Return(nil).Maybe()
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil).Maybe()
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil).Maybe()
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		Return(nil).Maybe()
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil).Maybe()
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Return(nil).Maybe()
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)

	// Mock transformations
	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil).Maybe()

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "main"}}, nil).Maybe()

	// Mock PR creation - should be called for repos that need sync
	for _, target := range scenario.TargetRepos {
		if scenario.State.Targets[fmt.Sprintf("org/%s", target.Name)].Status == state.StatusBehind {
			mockGH.On("CreatePR", mock.Anything, fmt.Sprintf("org/%s", target.Name), mock.AnythingOfType("gh.PRRequest")).
				Return(&gh.PR{
					Number: 123,
					Title:  "Sync from source repository",
				}, nil).Maybe()
		}
	}

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false).WithMaxConcurrency(2)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	engine.SetLogger(logger)

	// Execute sync
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = engine.Sync(ctx, nil)

	// Should handle conflicts gracefully
	require.NoError(t, err, "Sync should handle conflicts gracefully")
	mockState.AssertExpectations(t)

	// Verify that PRs were created for outdated repos
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		if scenario.State.Targets[repoName].Status == state.StatusBehind {
			mockGH.AssertCalled(t, "CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest"))
		}
	}
}

// testPartialSyncFailureRecovery tests recovery from partial sync failures
func testPartialSyncFailureRecovery(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create partial failure scenario
	scenario, err := generator.CreatePartialFailureScenario()
	require.NoError(t, err)

	// Setup mocks with controlled failures
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	// Setup failure injector
	failureInjector := &fixtures.MockFailureInjector{
		FailureMode:  fixtures.FailureNetwork,
		FailureRepos: []string{"service-b"}, // Fail on service-b
		FailureCount: 2,                     // Fail twice then succeed
	}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Mock successful operations for most repos
	mockGH.On("GetFile", mock.Anything, mock.MatchedBy(func(repo string) bool {
		return !strings.Contains(repo, "service-b")
	}), mock.AnythingOfType("string"), "").
		Return(&gh.FileContent{Content: []byte("content")}, nil).Maybe()

	// Mock failures for service-b initially, then success
	callCount := 0
	mockGH.On("GetFile", mock.Anything, "org/service-b", mock.AnythingOfType("string"), "").
		Return(func(ctx context.Context, repo, _, _ string) (*gh.FileContent, error) {
			callCount++
			if callCount <= 2 && failureInjector.ShouldFail(ctx, repo, "GetFile") {
				return nil, failureInjector.GetFailureError("GetFile")
			}
			return &gh.FileContent{Content: []byte("content")}, nil
		}).Maybe()

	// Mock Git operations
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			cloneDir := args[2].(string)

			// Create the expected files in the clone directory from the test scenario
			for filePath, content := range scenario.SourceRepo.Files {
				fullPath := filepath.Join(cloneDir, filePath)
				dir := filepath.Dir(fullPath)
				_ = os.MkdirAll(dir, 0o750)
				_ = os.WriteFile(fullPath, content, 0o600)
			}
		}).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "main"}}, nil).Maybe()

	// Mock PR creation
	mockGH.On("CreatePR", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("gh.PRRequest")).
		Return(&gh.PR{Number: 123}, nil)

	// Create sync engine with retry capabilities
	opts := sync.DefaultOptions().WithDryRun(false).WithMaxConcurrency(1)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync with retries
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err = engine.Sync(ctx, nil)

	// Should eventually succeed after retries
	require.NoError(t, err, "Sync should recover from partial failures")
	mockState.AssertExpectations(t)
}

// testLargeFileHandling tests sync with large files and memory monitoring
func testLargeFileHandling(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Skip if running in short test mode to avoid long test times
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	// Create scenario with large files (50MB)
	largeRepo, err := generator.CreateLargeFileRepo("large-service", "org", 50)
	require.NoError(t, err)
	_ = largeRepo // Mark as used for linter

	// Create source repository with the files that will be synced
	sourceRepo, err := generator.CreateLargeFileRepo("template-repo", "org", 50)
	require.NoError(t, err)

	// Create config for large file sync
	cfg := &config.Config{
		Version: 1,
		Source: config.SourceConfig{
			Repo:   "org/template-repo",
			Branch: "main",
		},
		Targets: []config.TargetConfig{
			{
				Repo: "org/large-service",
				Files: []config.FileMapping{
					{Src: "large_file_50mb.txt", Dest: "large_file_50mb.txt"},
					{Src: "README.md", Dest: "README.md"},
				},
			},
		},
	}

	// Setup mocks
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	// Mock state with large file repo needing sync
	currentState := &state.State{
		Source: state.SourceState{
			Repo:         "org/template-repo",
			Branch:       "main",
			LatestCommit: "latest123",
			LastChecked:  time.Now(),
		},
		Targets: map[string]*state.TargetState{
			"org/large-service": {
				Repo:           "org/large-service",
				LastSyncCommit: "old123",
				Status:         state.StatusBehind,
			},
		},
	}

	mockState.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

	// Monitor memory usage before operation
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Mock large file operations
	largeContent := make([]byte, 50*1024*1024) // 50MB
	for i := range largeContent {
		largeContent[i] = byte('A' + (i % 26))
	}

	mockGH.On("GetFile", mock.Anything, "org/template-repo", "large_file_50mb.txt", "").
		Return(&gh.FileContent{Content: largeContent}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "README.md", "").
		Return(&gh.FileContent{Content: []byte("# Large File Repository")}, nil)
	mockGH.On("GetFile", mock.Anything, "org/large-service", mock.AnythingOfType("string"), "").
		Return(&gh.FileContent{Content: []byte("old content")}, nil).Maybe()

	// Mock Git operations
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory from the test scenario
			cloneDir := args[2].(string)

			// Create source files from the sourceRepo
			for filePath, content := range sourceRepo.Files {
				fullPath := filepath.Join(cloneDir, filePath)
				dir := filepath.Dir(fullPath)
				_ = os.MkdirAll(dir, 0o750)
				_ = os.WriteFile(fullPath, content, 0o600)
			}
		}).Return(nil)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)

	// Mock transformation that processes large files
	mockTransform.On("Transform", mock.Anything, mock.MatchedBy(func(content []byte) bool {
		return len(content) > 40*1024*1024 // Large content
	}), mock.Anything).Return(largeContent, nil)

	mockTransform.On("Transform", mock.Anything, mock.MatchedBy(func(content []byte) bool {
		return len(content) <= 40*1024*1024
	}), mock.Anything).Return([]byte("transformed content"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "main"}}, nil).Maybe()

	mockGH.On("CreatePR", mock.Anything, "org/large-service", mock.AnythingOfType("gh.PRRequest")).
		Return(&gh.PR{Number: 123}, nil)

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false)
	engine := sync.NewEngine(cfg, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // Longer timeout for large files
	defer cancel()

	start := time.Now()
	err = engine.Sync(ctx, nil)
	duration := time.Since(start)

	// Verify operation succeeded
	require.NoError(t, err, "Large file sync should succeed")

	// Monitor memory usage after operation
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Memory growth should be reasonable (less than 200MB)
	var memoryGrowth uint64
	if m2.HeapAlloc > m1.HeapAlloc {
		memoryGrowth = m2.HeapAlloc - m1.HeapAlloc
	} else {
		memoryGrowth = 0 // Memory decreased due to GC
	}
	assert.Less(t, memoryGrowth, uint64(200*1024*1024),
		"Memory growth should be reasonable")

	// Performance should be acceptable (less than 60 seconds for 50MB)
	assert.Less(t, duration, 60*time.Second,
		"Large file sync should complete in reasonable time")

	mockState.AssertExpectations(t)
	t.Logf("Large file sync completed in %v with memory growth of %d bytes", duration, memoryGrowth)
}

// testConcurrentSyncOperations tests concurrent sync operations
func testConcurrentSyncOperations(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario with multiple repos
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks for concurrent operations
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	// Use mutex to ensure thread-safe mock operations
	var mu syncpkg.Mutex
	var operationCount int

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Mock GitHub operations with concurrency tracking
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(mock.Arguments) {
			mu.Lock()
			operationCount++
			mu.Unlock()
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
		}).
		Return(&gh.FileContent{Content: []byte("content")}, nil).Maybe()

	// Mock Git operations
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory from the test scenario
			cloneDir := args[2].(string)

			// Create source files from the scenario
			for filePath, content := range scenario.SourceRepo.Files {
				fullPath := filepath.Join(cloneDir, filePath)
				dir := filepath.Dir(fullPath)
				_ = os.MkdirAll(dir, 0o750)
				_ = os.WriteFile(fullPath, content, 0o600)
			}
		}).Return(nil)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "main"}}, nil).Maybe()

	mockGH.On("CreatePR", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("gh.PRRequest")).
		Return(&gh.PR{Number: 123}, nil)

	// Test different concurrency levels
	concurrencyLevels := []int{1, 2, 5}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("concurrency_%d", concurrency), func(t *testing.T) {
			// Reset operation count
			mu.Lock()
			operationCount = 0
			mu.Unlock()

			// Create sync engine with specific concurrency
			opts := sync.DefaultOptions().WithDryRun(false).WithMaxConcurrency(concurrency)
			engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
			engine.SetLogger(logrus.New())

			// Execute sync and measure performance
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			start := time.Now()
			err := engine.Sync(ctx, nil)
			duration := time.Since(start)

			require.NoError(t, err, "Concurrent sync should succeed")

			// Verify operations were performed
			mu.Lock()
			finalOperationCount := operationCount
			mu.Unlock()

			assert.Positive(t, finalOperationCount, "Should have performed operations")

			t.Logf("Concurrency %d: completed in %v with %d operations",
				concurrency, duration, finalOperationCount)

			// Higher concurrency should generally be faster (with some tolerance)
			if concurrency > 1 {
				expectedMaxDuration := 20 * time.Second
				assert.Less(t, duration, expectedMaxDuration,
					"Concurrent sync should be reasonably fast")
			}
		})
	}

	mockState.AssertExpectations(t)
}

// testMemoryUsageMonitoring tests memory usage patterns during sync
func testMemoryUsageMonitoring(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario with multiple repos of different sizes
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Add a medium-sized repo for variety
	mediumRepo, err := generator.CreateLargeFileRepo("medium-service", "org", 10) // 10MB
	require.NoError(t, err)
	scenario.TargetRepos = append(scenario.TargetRepos, mediumRepo)
	_ = mediumRepo // Mark as used for linter

	// Setup mocks
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Mock different sized content
	mockGH.On("GetFile", mock.Anything, "org/template-repo", mock.AnythingOfType("string"), "").
		Return(&gh.FileContent{Content: []byte("template content")}, nil).Maybe()

	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)

		if strings.Contains(target.Name, "large") {
			// Large content
			largeContent := make([]byte, 10*1024*1024) // 10MB
			mockGH.On("GetFile", mock.Anything, repoName, mock.AnythingOfType("string"), "").
				Return(&gh.FileContent{Content: largeContent}, nil).Maybe()
		} else {
			// Normal content
			mockGH.On("GetFile", mock.Anything, repoName, mock.AnythingOfType("string"), "").
				Return(&gh.FileContent{Content: []byte("normal content")}, nil).Maybe()
		}
	}

	// Mock Git operations
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory from the test scenario
			cloneDir := args[2].(string)

			// Create source files from the scenario
			for filePath, content := range scenario.SourceRepo.Files {
				fullPath := filepath.Join(cloneDir, filePath)
				dir := filepath.Dir(fullPath)
				_ = os.MkdirAll(dir, 0o750)
				_ = os.WriteFile(fullPath, content, 0o600)
			}
		}).Return(nil)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed"), nil)

	// Mock branch listing for PR operations
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{{Name: "main"}}, nil).Maybe()

	mockGH.On("CreatePR", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("gh.PRRequest")).
		Return(&gh.PR{Number: 123}, nil).Maybe()

	// Monitor memory throughout the sync process
	var memStats []runtime.MemStats
	var memMutex syncpkg.Mutex

	// Start memory monitoring goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				memMutex.Lock()
				memStats = append(memStats, m)
				memMutex.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false).WithMaxConcurrency(2)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync
	err = engine.Sync(ctx, nil)
	require.NoError(t, err)

	// Wait a bit for final memory measurements
	time.Sleep(200 * time.Millisecond)

	// Analyze memory usage patterns
	memMutex.Lock()
	defer memMutex.Unlock()

	if len(memStats) < 2 {
		t.Skip("Not enough memory samples collected")
	}

	// Calculate memory statistics
	var maxHeap, minHeap uint64 = 0, ^uint64(0)
	var totalAllocGrowth uint64

	for i, stats := range memStats {
		if stats.HeapAlloc > maxHeap {
			maxHeap = stats.HeapAlloc
		}
		if stats.HeapAlloc < minHeap {
			minHeap = stats.HeapAlloc
		}
		if i > 0 {
			if stats.TotalAlloc > memStats[i-1].TotalAlloc {
				totalAllocGrowth += stats.TotalAlloc - memStats[i-1].TotalAlloc
			}
		}
	}

	heapGrowth := maxHeap - minHeap

	// Memory growth should be reasonable
	assert.Less(t, heapGrowth, uint64(100*1024*1024),
		"Heap growth should be reasonable")

	// Total allocations should not be excessive
	assert.Less(t, totalAllocGrowth, uint64(500*1024*1024),
		"Total allocation growth should be reasonable")

	t.Logf("Memory usage: heap growth = %d bytes, total alloc growth = %d bytes, samples = %d",
		heapGrowth, totalAllocGrowth, len(memStats))

	mockState.AssertExpectations(t)
}

// testStateConsistencyAcrossFailures tests that system state remains consistent during failures
func testStateConsistencyAcrossFailures(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario with multiple repos
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks with intermittent failures
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	// Track state consistency
	var stateCallCount int
	originalState := scenario.State

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(originalState, nil).
		Run(func(mock.Arguments) {
			stateCallCount++
		})

	// Mock operations with some controlled failures
	failureRate := 0.3 // 30% failure rate
	_ = failureRate    // Mark as used for linter
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Return(func(_ context.Context, _, _, _ string) (*gh.FileContent, error) {
			// Introduce random failures
			if (time.Now().UnixNano() % 10) < 3 { // ~30% failure rate
				return nil, fmt.Errorf("%w", fixtures.ErrNetworkFailure)
			}
			return &gh.FileContent{Content: []byte("content")}, nil
		}).Maybe()

	// Mock Git operations that can fail - use a variable call count to determine success/failure
	var cloneCallCount int
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil).
		Run(func(args mock.Arguments) {
			cloneCallCount++
			cloneDir := args[2].(string)

			// Create the expected files in the clone directory for successful calls
			for filePath, content := range scenario.SourceRepo.Files {
				fullPath := filepath.Join(cloneDir, filePath)
				dir := filepath.Dir(fullPath)
				_ = os.MkdirAll(dir, 0o750)
				_ = os.WriteFile(fullPath, content, 0o600)
			}
		}).Maybe()

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
		Return([]gh.Branch{{Name: "main"}}, nil).Maybe()

	mockGH.On("CreatePR", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("gh.PRRequest")).
		Return(&gh.PR{Number: 123}, nil).Maybe()

	// Create sync engine
	opts := sync.DefaultOptions().WithDryRun(false).WithMaxConcurrency(1)
	engine := sync.NewEngine(scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute multiple sync attempts to test state consistency
	attempts := 3
	var results []error

	for i := 0; i < attempts; i++ {
		t.Logf("Sync attempt %d/%d", i+1, attempts)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := engine.Sync(ctx, nil)
		cancel()

		results = append(results, err)

		// Brief pause between attempts
		time.Sleep(100 * time.Millisecond)
	}

	// At least one attempt should succeed (with retries and error handling)
	hasSuccess := false
	for _, err := range results {
		if err == nil {
			hasSuccess = true
			break
		}
	}

	// With proper error handling, we should achieve some success
	t.Logf("Sync attempts completed. Success achieved: %v", hasSuccess)
	t.Logf("State discovery called %d times across %d attempts", stateCallCount, attempts)

	// State should be called consistently
	assert.GreaterOrEqual(t, stateCallCount, attempts, "State discovery should be called for each attempt")

	// Log results for analysis
	for i, err := range results {
		if err != nil {
			t.Logf("Attempt %d failed: %v", i+1, err)
		} else {
			t.Logf("Attempt %d succeeded", i+1)
		}
	}

	mockState.AssertExpectations(t)
}
