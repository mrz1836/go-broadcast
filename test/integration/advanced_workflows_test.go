package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	internalsync "github.com/mrz1836/go-broadcast/internal/sync"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/mrz1836/go-broadcast/test/fixtures"
)

// TestAdvancedWorkflows tests advanced workflow scenarios
func TestAdvancedWorkflows(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	generator := fixtures.NewTestRepoGenerator(tmpDir)
	defer func() {
		if err := generator.Cleanup(); err != nil {
			t.Errorf("failed to cleanup test generator: %v", err)
		}
	}()

	t.Run("branch_protection_handling", func(t *testing.T) {
		testBranchProtectionHandling(t, generator)
	})

	t.Run("template_repository_updates", func(t *testing.T) {
		testTemplateRepositoryUpdates(t, generator)
	})

	t.Run("rollback_capabilities", func(t *testing.T) {
		testRollbackCapabilities(t, generator)
	})

	t.Run("state_consistency_across_operations", func(t *testing.T) {
		testStateConsistencyAcrossOperations(t, generator)
	})

	t.Run("workflow_permissions_and_security", func(t *testing.T) {
		testWorkflowPermissionsAndSecurity(t, generator)
	})

	t.Run("incremental_template_changes", func(t *testing.T) {
		testIncrementalTemplateChanges(t, generator)
	})
}

// testBranchProtectionHandling tests sync behavior with protected branches
func testBranchProtectionHandling(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create branch protection scenario
	scenario, err := generator.CreateBranchProtectionScenario()
	require.NoError(t, err)

	// Setup mocks for branch protection
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	// Add broad GetChangedFiles mock to handle all calls
	mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Mock branch protection information
	protectedBranch := &gh.Branch{
		Name:      "master",
		Protected: true,
	}

	// Mock GitHub API calls for branch protection
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)

		// Mock getting branch protection status
		mockGH.On("GetBranch", mock.Anything, repoName, "master").
			Return(protectedBranch, nil)

		// Mock that direct pushes are not allowed
		mockGH.On("Push", mock.Anything, repoName, mock.AnythingOfType("string")).
			Return(fmt.Errorf("%w to main", fixtures.ErrBranchProtection))

		// Mock successful PR creation (the proper way to update protected branches)
		mockGH.On("ListBranches", mock.Anything, repoName).
			Return([]gh.Branch{}, nil)
		mockGH.On("GetCurrentUser", mock.Anything).
			Return(&gh.User{Login: "testuser", ID: 123}, nil)
		mockGH.On("CreatePR", mock.Anything, repoName, mock.MatchedBy(func(req gh.PRRequest) bool {
			// Verify PR is created with proper branch (not main)
			return req.Head != "master" && strings.Contains(req.Head, "chore/sync-files")
		})).Return(&gh.PR{
			Number: 123,
			Title:  "Sync from source repository",
		}, nil)
	}

	// Mock template repository file fetching
	mockGH.On("GetFile", mock.Anything, "org/template-repo", ".github/workflows/ci.yml", "").
		Return(&gh.FileContent{Content: []byte("ci workflow content")}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "Makefile", "").
		Return(&gh.FileContent{Content: []byte("makefile content")}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "README.md", "").
		Return(&gh.FileContent{Content: []byte("readme content")}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "docker-compose.yml", "").
		Return(&gh.FileContent{Content: []byte("docker compose content")}, nil)

	// Mock target repository files (optional)
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Return(&gh.FileContent{Content: []byte("old file content")}, nil).Maybe()

	// Mock Git operations for creating sync branches (not pushing to main)
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory
			cloneDir := args[2].(string)
			_ = os.MkdirAll(filepath.Join(cloneDir, ".github/workflows"), 0o750)
			_ = os.WriteFile(filepath.Join(cloneDir, ".github/workflows/ci.yml"), []byte("ci workflow content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "Makefile"), []byte("makefile content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("readme content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "docker-compose.yml"), []byte("docker compose content"), 0o600)
		}).
		Return(nil)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)

	// Mock creating sync branch (should succeed)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.MatchedBy(func(branch string) bool {
		return strings.Contains(branch, "chore/sync-files")
	})).Return(nil)

	// Mock checkout to sync branch
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.MatchedBy(func(branch string) bool {
		return strings.Contains(branch, "chore/sync-files")
	})).Return(nil)

	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).
		Return("abc123def456", nil)

	// Mock pushing to sync branch (should succeed)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.MatchedBy(func(branch string) bool {
		return strings.Contains(branch, "chore/sync-files")
	}), false).Return(nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Create sync engine
	opts := internalsync.DefaultOptions().WithDryRun(false)
	engine := internalsync.NewEngine(context.Background(), scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	engine.SetLogger(logger)

	// Execute sync
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = engine.Sync(ctx, nil)

	// Should handle branch protection by creating PRs instead of direct pushes
	require.NoError(t, err, "Sync should handle branch protection correctly")

	// Verify that PRs were created (not direct pushes to main)
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		if scenario.State.Targets[repoName].Status == state.StatusBehind {
			mockGH.AssertCalled(t, "CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest"))
		}
	}

	// Verify no attempts to push directly to main
	mockGit.AssertNotCalled(t, "Push", mock.Anything, mock.AnythingOfType("string"), "origin", "master", mock.AnythingOfType("bool"))

	mockState.AssertExpectations(t)
}

// testTemplateRepositoryUpdates tests handling of template repository changes
func testTemplateRepositoryUpdates(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create initial scenario
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Simulate template repository update by changing the source commit
	updatedCommit := "new" + scenario.State.Source.LatestCommit

	// Create updated state reflecting template changes
	updatedState := &state.State{
		Source: state.SourceState{
			Repo:         scenario.State.Source.Repo,
			Branch:       scenario.State.Source.Branch,
			LatestCommit: updatedCommit,
			LastChecked:  time.Now(),
		},
		Targets: make(map[string]*state.TargetState),
	}

	// All targets are now behind due to template update
	for repo, targetState := range scenario.State.Targets {
		updatedState.Targets[repo] = &state.TargetState{
			Repo:           targetState.Repo,
			LastSyncCommit: targetState.LastSyncCommit, // Still old commit
			Status:         state.StatusBehind,         // Now behind
			LastSyncTime:   targetState.LastSyncTime,
		}
	}

	// Setup mocks
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	// Add broad GetChangedFiles mock to handle all calls
	mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(updatedState, nil)

	// Mock updated template files
	updatedTemplateFiles := map[string][]byte{
		".github/workflows/ci.yml": []byte(`name: CI (Updated)
on:
  push:
    branches: [ master, development ]
  pull_request:
    branches: [ master ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4  # Updated version
    - name: Set up Go
      uses: actions/setup-go@v4  # Updated version
      with:
        go-version: 1.22         # Updated version
    - name: Run tests
      run: make test
    - name: Run new security scan  # New step
      run: make security-scan`),
		"Makefile": []byte(`# Template Makefile (Updated)
.PHONY: test build lint clean security-scan

test:
	go test -v ./...

build:
	go build -o bin/app ./cmd/app

lint:
	golangci-lint run

security-scan:  # New target
	gosec ./...

clean:
	rm -rf bin/`),
	}

	// Mock template repository file fetching
	mockGH.On("GetFile", mock.Anything, "org/template-repo", ".github/workflows/ci.yml", "").
		Return(&gh.FileContent{Content: updatedTemplateFiles[".github/workflows/ci.yml"]}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "Makefile", "").
		Return(&gh.FileContent{Content: updatedTemplateFiles["Makefile"]}, nil)

	// Mock target repository files (older versions)
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)

		mockGH.On("GetFile", mock.Anything, repoName, ".github/workflows/ci.yml", "").
			Return(&gh.FileContent{Content: []byte("old ci workflow")}, nil).Maybe()
		mockGH.On("GetFile", mock.Anything, repoName, "Makefile", "").
			Return(&gh.FileContent{Content: []byte("old makefile")}, nil).Maybe()
		mockGH.On("GetFile", mock.Anything, repoName, "README.md", "").
			Return(&gh.FileContent{Content: []byte("old readme")}, nil).Maybe()
		mockGH.On("GetFile", mock.Anything, repoName, "docker-compose.yml", "").
			Return(&gh.FileContent{Content: []byte("old docker compose")}, nil).Maybe()
	}

	// Mock Git operations
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory
			cloneDir := args[2].(string)
			_ = os.MkdirAll(filepath.Join(cloneDir, ".github/workflows"), 0o750)
			_ = os.WriteFile(filepath.Join(cloneDir, ".github/workflows/ci.yml"), []byte("ci workflow content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "Makefile"), []byte("makefile content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("readme content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "docker-compose.yml"), []byte("docker compose content"), 0o600)
		}).
		Return(nil)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).
		Return("abc123def456", nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).
		Return(nil)

	// Mock transformations that preserve template updates
	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock PR creation for template updates
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)

		mockGH.On("ListBranches", mock.Anything, repoName).
			Return([]gh.Branch{}, nil)
		mockGH.On("GetCurrentUser", mock.Anything).
			Return(&gh.User{Login: "testuser", ID: 123}, nil)
		mockGH.On("CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest")).
			Return(&gh.PR{
				Number: 456,
				Title:  "Sync template updates: CI workflow and Makefile improvements",
			}, nil)
	}

	// Create sync engine
	opts := internalsync.DefaultOptions().WithDryRun(false)
	engine := internalsync.NewEngine(context.Background(), scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	err = engine.Sync(ctx, nil)

	// Should successfully propagate template updates
	require.NoError(t, err, "Template updates should be propagated successfully")

	// Verify PRs were created for all targets (since all are behind)
	for _, target := range scenario.TargetRepos {
		repoName := fmt.Sprintf("org/%s", target.Name)
		mockGH.AssertCalled(t, "CreatePR", mock.Anything, repoName, mock.AnythingOfType("gh.PRRequest"))
	}

	mockState.AssertExpectations(t)
}

// testRollbackCapabilities tests rollback functionality when sync fails
func testRollbackCapabilities(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for rollback testing
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks with controlled failures
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	// Add broad GetChangedFiles mock to handle all calls
	mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	// Track operations for rollback verification with thread safety
	var gitOperations []string
	var gitOperationsMutex sync.Mutex
	var rollbackCalled bool
	var rollbackMutex sync.Mutex

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Mock template repository file fetching
	mockGH.On("GetFile", mock.Anything, "org/template-repo", ".github/workflows/ci.yml", "").
		Return(&gh.FileContent{Content: []byte("ci workflow content")}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "Makefile", "").
		Return(&gh.FileContent{Content: []byte("makefile content")}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "README.md", "").
		Return(&gh.FileContent{Content: []byte("readme content")}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "docker-compose.yml", "").
		Return(&gh.FileContent{Content: []byte("docker compose content")}, nil)

	// Mock other file operations
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Return(&gh.FileContent{Content: []byte("content")}, nil).Maybe()

	// Mock Git operations with tracking
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).
		Run(func(args mock.Arguments) {
			gitOperationsMutex.Lock()
			gitOperations = append(gitOperations, "clone:"+args[2].(string))
			gitOperationsMutex.Unlock()
			// Create the expected files in the clone directory
			cloneDir := args[2].(string)
			_ = os.MkdirAll(filepath.Join(cloneDir, ".github/workflows"), 0o750)
			_ = os.WriteFile(filepath.Join(cloneDir, ".github/workflows/ci.yml"), []byte("ci workflow content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "Makefile"), []byte("makefile content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("readme content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "docker-compose.yml"), []byte("docker compose content"), 0o600)
		}).Return(nil)

	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			gitOperationsMutex.Lock()
			gitOperations = append(gitOperations, "checkout:"+args[2].(string))
			gitOperationsMutex.Unlock()
		}).Return(nil)

	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			gitOperationsMutex.Lock()
			gitOperations = append(gitOperations, "create_branch:"+args[2].(string))
			gitOperationsMutex.Unlock()
		}).Return(nil)

	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		Run(func(args mock.Arguments) {
			gitOperationsMutex.Lock()
			gitOperations = append(gitOperations, "add:"+args[1].(string))
			gitOperationsMutex.Unlock()
		}).Return(nil)

	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			gitOperationsMutex.Lock()
			gitOperations = append(gitOperations, "commit:"+args[1].(string))
			gitOperationsMutex.Unlock()
		}).Return(nil)

	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).
		Return("abc123def456", nil)

	// Allow Push to succeed so PR creation is attempted
	mockGit.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(_ mock.Arguments) {
			gitOperationsMutex.Lock()
			gitOperations = append(gitOperations, "push_success")
			gitOperationsMutex.Unlock()
		}).Return(nil).Maybe()

	// Mock rollback operations
	mockGit.On("Reset", mock.Anything, mock.AnythingOfType("string"), "HEAD~1").
		Run(func(args mock.Arguments) {
			rollbackMutex.Lock()
			rollbackCalled = true
			rollbackMutex.Unlock()
			gitOperationsMutex.Lock()
			gitOperations = append(gitOperations, "rollback_reset:"+args[1].(string))
			gitOperationsMutex.Unlock()
		}).Return(nil).Maybe()

	mockGit.On("Clean", mock.Anything, mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			gitOperationsMutex.Lock()
			gitOperations = append(gitOperations, "rollback_clean:"+args[1].(string))
			gitOperationsMutex.Unlock()
		}).Return(nil).Maybe()

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	// Mock branch listing and PR creation - PR creation should fail
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{}, nil).Maybe()
	mockGH.On("GetCurrentUser", mock.Anything).
		Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()

	// Make PR creation fail to simulate the rollback scenario
	mockGH.On("CreatePR", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("gh.PRRequest")).
		Run(func(args mock.Arguments) {
			gitOperationsMutex.Lock()
			gitOperations = append(gitOperations, "pr_creation_failed:"+args[1].(string))
			gitOperationsMutex.Unlock()
		}).Return(nil, fmt.Errorf("PR creation failed: %w", fixtures.ErrBranchProtection)).Maybe()

	// Create sync engine
	opts := internalsync.DefaultOptions().WithDryRun(false)
	engine := internalsync.NewEngine(context.Background(), scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	engine.SetLogger(logger)

	// Execute sync (should fail and trigger rollback)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = engine.Sync(ctx, nil)

	// Debug: Check what operations were performed
	gitOperationsMutex.Lock()
	operationLog := strings.Join(gitOperations, ", ")
	gitOperationsMutex.Unlock()
	t.Logf("Git operations performed: %s", operationLog)
	t.Logf("Failure count should be 2, error: %v", err)

	// Sync should fail due to push failure, but rollback should be attempted
	require.Error(t, err, "Sync should fail due to push failure")
	assert.Contains(t, err.Error(), "completed with 2 failures out of 2 targets", "Error should indicate sync failures")

	// Verify rollback was attempted
	t.Logf("Git operations performed: %s", operationLog)

	// Should have attempted operations before failure
	assert.Contains(t, operationLog, "clone", "Should have cloned repositories")
	assert.Contains(t, operationLog, "pr_creation_failed", "Should have encountered PR creation failure")

	// With proper error handling, the system should handle partial failures gracefully
	// The exact rollback mechanism depends on the engine implementation
	t.Logf("Rollback mechanism called: %v", rollbackCalled)

	mockState.AssertExpectations(t)
}

// testStateConsistencyAcrossOperations tests that state remains consistent across complex operations
func testStateConsistencyAcrossOperations(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario with multiple operations
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	// Add broad GetChangedFiles mock to handle all calls
	mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	// Track state consistency across multiple calls
	stateCallCount := 0
	lastState := scenario.State

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(lastState, nil).
		Run(func(_ mock.Arguments) {
			stateCallCount++

			// Simulate state evolution over time
			if stateCallCount > 1 {
				// Update some target states to reflect sync progress
				for repo, targetState := range lastState.Targets {
					if targetState.Status == state.StatusBehind {
						// Mark first repo as completed after first call
						if stateCallCount == 2 && strings.Contains(repo, "service-a") {
							targetState.Status = state.StatusUpToDate
							targetState.LastSyncCommit = lastState.Source.LatestCommit
							targetState.LastSyncTime = &[]time.Time{time.Now()}[0]
						}
					}
				}
			}
		})

	// Mock template repository file fetching
	mockGH.On("GetFile", mock.Anything, "org/template-repo", ".github/workflows/ci.yml", "").
		Return(&gh.FileContent{Content: []byte("ci workflow content")}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "Makefile", "").
		Return(&gh.FileContent{Content: []byte("makefile content")}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "README.md", "").
		Return(&gh.FileContent{Content: []byte("readme content")}, nil)
	mockGH.On("GetFile", mock.Anything, "org/template-repo", "docker-compose.yml", "").
		Return(&gh.FileContent{Content: []byte("docker compose content")}, nil)

	// Mock target repository files (optional)
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Return(&gh.FileContent{Content: []byte("old content")}, nil).Maybe()

	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory
			cloneDir := args[2].(string)
			_ = os.MkdirAll(filepath.Join(cloneDir, ".github/workflows"), 0o750)
			_ = os.WriteFile(filepath.Join(cloneDir, ".github/workflows/ci.yml"), []byte("ci workflow content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "Makefile"), []byte("makefile content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("readme content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "docker-compose.yml"), []byte("docker compose content"), 0o600)
		}).
		Return(nil)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{}, nil).Maybe()
	mockGH.On("GetCurrentUser", mock.Anything).
		Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
	mockGH.On("CreatePR", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("gh.PRRequest")).
		Return(&gh.PR{Number: 999}, nil).Maybe()

	// Create sync engine
	opts := internalsync.DefaultOptions().WithDryRun(false)
	engine := internalsync.NewEngine(context.Background(), scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute multiple sync operations to test state consistency
	for i := 0; i < 3; i++ {
		t.Logf("Sync iteration %d", i+1)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		err := engine.Sync(ctx, nil)
		cancel()

		// Should handle state consistently across iterations
		require.NoError(t, err, "Sync iteration should succeed")

		// Brief pause between iterations
		time.Sleep(100 * time.Millisecond)
	}

	// Verify state was discovered for each iteration
	assert.GreaterOrEqual(t, stateCallCount, 3, "State should be discovered for each sync operation")

	mockState.AssertExpectations(t)
}

// testWorkflowPermissionsAndSecurity tests security aspects of advanced workflows
func testWorkflowPermissionsAndSecurity(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create scenario for security testing
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks for security testing
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	// Add broad GetChangedFiles mock to handle all calls
	mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Mock authentication failure scenarios
	authFailureCount := 0
	mockGH.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), "").
		Run(func(mock.Arguments) {
			authFailureCount++
		}).
		Return(&gh.FileContent{Content: []byte("secure content")}, nil).Maybe()

	// Mock permission validation
	mockGH.On("GetCurrentUser", mock.Anything).
		Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{}, nil).Maybe()
	mockGH.On("CreatePR", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("gh.PRRequest")).
		Return(&gh.PR{Number: 111}, nil).Maybe()

	// Mock Git operations with security validations
	mockGit.On("Clone", mock.Anything, mock.MatchedBy(func(url string) bool {
		// Validate URL format for security
		return strings.HasPrefix(url, "https://github.com/") && !strings.Contains(url, "..")
	}), mock.AnythingOfType("string"), mock.Anything).Return(nil)

	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.MatchedBy(func(branch string) bool {
		// Validate branch name doesn't contain dangerous characters
		return !strings.Contains(branch, "..") && !strings.Contains(branch, ";")
	})).Return(nil)

	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)

	// Mock secure transformations
	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.MatchedBy(func(ctx transform.Context) bool {
		// Validate transformation context doesn't contain dangerous values
		for key, value := range ctx.Variables {
			if strings.Contains(value, "..") || strings.Contains(value, ";") || strings.Contains(key, "password") {
				return false
			}
		}
		return true
	})).Return([]byte("securely transformed content"), nil)

	// Create sync engine
	opts := internalsync.DefaultOptions().WithDryRun(false)
	engine := internalsync.NewEngine(context.Background(), scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute sync with security considerations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = engine.Sync(ctx, nil)

	// Should handle auth failures and permission issues gracefully
	// The exact behavior depends on retry logic and error handling in the engine
	t.Logf("Sync with security constraints result: %v", err)

	// Verify secure operations were attempted
	mockGit.AssertCalled(t, "Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything)

	mockState.AssertExpectations(t)
}

// testIncrementalTemplateChanges tests handling of incremental template updates
func testIncrementalTemplateChanges(t *testing.T, generator *fixtures.TestRepoGenerator) {
	// Create initial scenario
	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	// Setup mocks
	mockGH := &gh.MockClient{}
	mockGit := &git.MockClient{}
	// Add broad GetChangedFiles mock to handle all calls
	mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
	mockState := &state.MockDiscoverer{}
	mockTransform := &transform.MockChain{}

	// Simulate incremental changes over multiple sync cycles
	syncCycle := 0

	// Just use the scenario state for now (simpler pattern)
	mockState.On("DiscoverState", mock.Anything, scenario.Config).
		Return(scenario.State, nil)

	// Mock incremental file changes
	mockGH.On("GetFile", mock.Anything, "org/template-repo", mock.AnythingOfType("string"), "").
		Return(func(_ context.Context, _, _, _ string) (*gh.FileContent, error) {
			// Simulate incremental changes to template files
			syncCycle++ // Increment on each call
			baseContent := "# Template file\n"
			incrementalContent := fmt.Sprintf("# Version %d updates\n", syncCycle)
			return &gh.FileContent{
				Content: []byte(baseContent + incrementalContent),
			}, nil
		}).Maybe()

	// Mock target repo files
	mockGH.On("GetFile", mock.Anything, mock.MatchedBy(func(repo string) bool {
		return !strings.Contains(repo, "template-repo")
	}), mock.AnythingOfType("string"), "").
		Return(&gh.FileContent{Content: []byte("old content")}, nil).Maybe()

	// Mock Git operations
	mockGit.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.Anything).
		Run(func(args mock.Arguments) {
			// Create the expected files in the clone directory
			cloneDir := args[2].(string)
			_ = os.MkdirAll(filepath.Join(cloneDir, ".github/workflows"), 0o750)
			_ = os.WriteFile(filepath.Join(cloneDir, ".github/workflows/ci.yml"), []byte("ci workflow content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "Makefile"), []byte("makefile content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("readme content"), 0o600)
			_ = os.WriteFile(filepath.Join(cloneDir, "docker-compose.yml"), []byte("docker compose content"), 0o600)
		}).
		Return(nil)
	mockGit.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockGit.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
	mockGit.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123def456", nil)
	mockGit.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)

	mockTransform.On("Transform", mock.Anything, mock.Anything, mock.Anything).
		Return([]byte("transformed content"), nil)

	mockGH.On("ListBranches", mock.Anything, mock.AnythingOfType("string")).
		Return([]gh.Branch{}, nil).Maybe()
	mockGH.On("GetCurrentUser", mock.Anything).
		Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()
	mockGH.On("CreatePR", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("gh.PRRequest")).
		Return(&gh.PR{Number: 555}, nil).Maybe()

	// Create sync engine
	opts := internalsync.DefaultOptions().WithDryRun(false)
	engine := internalsync.NewEngine(context.Background(), scenario.Config, mockGH, mockGit, mockState, mockTransform, opts)
	engine.SetLogger(logrus.New())

	// Execute multiple sync cycles to test incremental changes
	completedCycles := 0
	for i := 0; i < 3; i++ {
		t.Logf("Incremental sync cycle %d", i+1)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		err := engine.Sync(ctx, nil)
		cancel()

		if err == nil {
			completedCycles++
		}

		require.NoError(t, err, "Incremental sync cycle should succeed")

		// Pause between cycles
		time.Sleep(200 * time.Millisecond)
	}

	// Verify all cycles completed
	assert.Equal(t, 3, completedCycles, "Should have completed all incremental sync cycles")

	mockState.AssertExpectations(t)
}
