package fixtures

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/state"
)

// TestNewTestRepoGenerator tests the creation of a new test repository generator
func TestNewTestRepoGenerator(t *testing.T) {
	baseDir := "/tmp/test-repos"
	generator := NewTestRepoGenerator(baseDir)

	assert.NotNil(t, generator)
	assert.Equal(t, baseDir, generator.BaseDir)
	assert.Empty(t, generator.TempDir)
}

// TestCreateRepo tests basic repository creation
func TestCreateRepo(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	files := map[string]string{
		"README.md":                "# Test Repository",
		"src/main.go":              "package main\n\nfunc main() {}",
		"docs/api/README.md":       "# API Documentation",
		".github/workflows/ci.yml": "name: CI\non: push",
	}

	repo, err := generator.CreateRepo("test-repo", "testowner", "main", files)
	require.NoError(t, err)
	require.NotNil(t, repo)

	// Verify repository metadata
	assert.Equal(t, "test-repo", repo.Name)
	assert.Equal(t, "testowner", repo.Owner)
	assert.Equal(t, "main", repo.Branch)
	assert.NotEmpty(t, repo.CommitSHA)
	assert.Len(t, repo.Files, len(files))
	assert.False(t, repo.HasConflict)
	assert.Positive(t, repo.Size)

	// Verify files were created and content matches
	for filePath, expectedContent := range files {
		fullPath := filepath.Join(repo.Path, filePath)
		assert.FileExists(t, fullPath)

		actualContent, err := os.ReadFile(fullPath) //nolint:gosec // Test file paths are controlled
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(actualContent))

		// Verify in-memory representation
		assert.Equal(t, []byte(expectedContent), repo.Files[filePath])
	}

	// Verify total size calculation
	expectedSize := int64(0)
	for _, content := range files {
		expectedSize += int64(len(content))
	}
	assert.Equal(t, expectedSize, repo.Size)
}

// TestCreateRepoEmptyFiles tests repository creation with empty files
func TestCreateRepoEmptyFiles(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	files := map[string]string{
		"empty.txt":     "",
		"README.md":     "# Test",
		"another-empty": "",
	}

	repo, err := generator.CreateRepo("empty-test", "owner", "main", files)
	require.NoError(t, err)

	assert.Equal(t, int64(6), repo.Size) // Only "# Test" contributes to size
	assert.Len(t, repo.Files, 3)

	// Verify empty files exist
	assert.FileExists(t, filepath.Join(repo.Path, "empty.txt"))
	assert.FileExists(t, filepath.Join(repo.Path, "another-empty"))
}

// TestCreateRepoDirectoryCreation tests nested directory creation
func TestCreateRepoDirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	files := map[string]string{
		"deep/nested/path/file.txt":             "content",
		"another/deep/nested/structure/test.go": "package test",
		"single-level.md":                       "# Single level",
	}

	repo, err := generator.CreateRepo("nested-test", "owner", "main", files)
	require.NoError(t, err)

	// Verify all nested directories were created
	for filePath := range files {
		fullPath := filepath.Join(repo.Path, filePath)
		assert.FileExists(t, fullPath)

		// Verify directory structure
		dir := filepath.Dir(fullPath)
		stat, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, stat.IsDir())
	}
}

// TestCreateLargeFileRepo tests creation of repositories with large files
func TestCreateLargeFileRepo(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	// Create a small test file (1MB) to avoid slow tests
	fileSizeMB := 1
	repo, err := generator.CreateLargeFileRepo("large-test", "owner", fileSizeMB)
	require.NoError(t, err)

	// Verify repository structure
	assert.Equal(t, "large-test", repo.Name)
	assert.Equal(t, "owner", repo.Owner)
	assert.Equal(t, "main", repo.Branch)

	// Verify large file was created
	largeFileName := "large_file_1mb.txt"
	assert.Contains(t, repo.Files, largeFileName)

	// Verify file size
	expectedSize := fileSizeMB * 1024 * 1024
	largeFileContent := repo.Files[largeFileName]
	assert.Len(t, largeFileContent, expectedSize)

	// Verify file content pattern (should be repeating A-Z)
	for i := 0; i < 100; i++ { // Check first 100 bytes
		expectedChar := byte('A' + (i % 26))
		assert.Equal(t, expectedChar, largeFileContent[i])
	}

	// Verify other standard files exist
	assert.Contains(t, repo.Files, "README.md")
	assert.Contains(t, repo.Files, ".github/workflows/ci.yml")

	// Verify total size is greater than large file size (includes other files)
	assert.Greater(t, repo.Size, int64(expectedSize))
}

// TestCreateConflictingRepo tests creation of repositories that will have conflicts
func TestCreateConflictingRepo(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	repo, err := generator.CreateConflictingRepo("conflict-test", "owner")
	require.NoError(t, err)

	// Verify conflict flag is set
	assert.True(t, repo.HasConflict)
	assert.Equal(t, "conflict-test", repo.Name)
	assert.Equal(t, "owner", repo.Owner)
	assert.Equal(t, "main", repo.Branch)

	// Verify expected files exist
	expectedFiles := []string{"README.md", ".github/workflows/ci.yml", "Makefile"}
	for _, fileName := range expectedFiles {
		assert.Contains(t, repo.Files, fileName)
		assert.FileExists(t, filepath.Join(repo.Path, fileName))
	}

	// Verify conflict-indicating content
	readmeContent := string(repo.Files["README.md"])
	assert.Contains(t, readmeContent, "will conflict")
	assert.Contains(t, readmeContent, "conflict-test")

	ciContent := string(repo.Files[".github/workflows/ci.yml"])
	assert.Contains(t, ciContent, "CI (Modified)")
	assert.Contains(t, ciContent, "Custom test for target repo")
}

// TestCreateComplexScenario tests creation of complex multi-repository scenarios
func TestCreateComplexScenario(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)
	require.NotNil(t, scenario)

	// Verify scenario metadata
	assert.Equal(t, "Complex Multi-Repository Sync", scenario.Name)
	assert.NotEmpty(t, scenario.Description)

	// Verify source repository
	require.NotNil(t, scenario.SourceRepo)
	assert.Equal(t, "template-repo", scenario.SourceRepo.Name)
	assert.Equal(t, "org", scenario.SourceRepo.Owner)
	assert.Equal(t, "main", scenario.SourceRepo.Branch)

	// Verify source repo has expected files
	expectedSourceFiles := []string{
		"README.md",
		".github/workflows/ci.yml",
		".github/workflows/release.yml",
		"Makefile",
		"docker-compose.yml",
		"scripts/setup.sh",
		"docs/API.md",
	}
	for _, fileName := range expectedSourceFiles {
		assert.Contains(t, scenario.SourceRepo.Files, fileName)
	}

	// Verify target repositories
	require.Len(t, scenario.TargetRepos, 3)

	// Check first target (normal repo)
	assert.Equal(t, "service-a", scenario.TargetRepos[0].Name)
	assert.False(t, scenario.TargetRepos[0].HasConflict)

	// Check second target (conflicting repo)
	assert.Equal(t, "service-b", scenario.TargetRepos[1].Name)
	assert.True(t, scenario.TargetRepos[1].HasConflict)

	// Check third target (large file repo)
	assert.Equal(t, "service-c", scenario.TargetRepos[2].Name)
	assert.Greater(t, scenario.TargetRepos[2].Size, int64(50*1024*1024)) // > 50MB

	// Verify configuration
	require.NotNil(t, scenario.Config)
	assert.Equal(t, 1, scenario.Config.Version)
	assert.Equal(t, "org/template-repo", scenario.Config.Source.Repo)
	assert.Equal(t, "main", scenario.Config.Source.Branch)
	assert.Len(t, scenario.Config.Targets, 3)

	// Verify state
	require.NotNil(t, scenario.State)
	assert.Equal(t, "org/template-repo", scenario.State.Source.Repo)
	assert.Equal(t, "main", scenario.State.Source.Branch)
	assert.Equal(t, scenario.SourceRepo.CommitSHA, scenario.State.Source.LatestCommit)
	assert.Len(t, scenario.State.Targets, 3)

	// Verify first target is up-to-date, others are behind
	serviceAState := scenario.State.Targets["org/service-a"]
	require.NotNil(t, serviceAState)
	assert.Equal(t, state.StatusUpToDate, serviceAState.Status)
	assert.Equal(t, scenario.SourceRepo.CommitSHA, serviceAState.LastSyncCommit)

	serviceBState := scenario.State.Targets["org/service-b"]
	require.NotNil(t, serviceBState)
	assert.Equal(t, state.StatusBehind, serviceBState.Status)
	assert.NotEqual(t, scenario.SourceRepo.CommitSHA, serviceBState.LastSyncCommit)
}

// TestCreatePartialFailureScenario tests creation of partial failure scenarios
func TestCreatePartialFailureScenario(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	scenario, err := generator.CreatePartialFailureScenario()
	require.NoError(t, err)

	assert.Equal(t, "Partial Failure Recovery", scenario.Name)
	assert.Contains(t, scenario.Description, "partially fails")

	// Verify second repo has conflict status
	serviceBState := scenario.State.Targets["org/service-b"]
	require.NotNil(t, serviceBState)
	assert.Equal(t, state.StatusConflict, serviceBState.Status)
}

// TestCreateNetworkFailureScenario tests creation of network failure scenarios
func TestCreateNetworkFailureScenario(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	scenario, err := generator.CreateNetworkFailureScenario()
	require.NoError(t, err)

	assert.Equal(t, "Network Failure Resilience", scenario.Name)
	assert.Contains(t, scenario.Description, "network interruption")
}

// TestCreateBranchProtectionScenario tests creation of branch protection scenarios
func TestCreateBranchProtectionScenario(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	scenario, err := generator.CreateBranchProtectionScenario()
	require.NoError(t, err)

	assert.Equal(t, "Branch Protection Testing", scenario.Name)
	assert.Contains(t, scenario.Description, "protected branches")
}

// TestMockFailureInjectorNone tests failure injector with no failures
func TestMockFailureInjectorNone(t *testing.T) {
	injector := &MockFailureInjector{
		FailureMode: FailureNone,
	}

	ctx := context.Background()
	shouldFail := injector.ShouldFail(ctx, "test/repo", "push")
	assert.False(t, shouldFail)
}

// TestMockFailureInjectorCount tests failure injector with failure count
func TestMockFailureInjectorCount(t *testing.T) {
	injector := &MockFailureInjector{
		FailureMode:  FailureNetwork,
		FailureCount: 2,
	}

	ctx := context.Background()

	// First two calls should fail
	assert.True(t, injector.ShouldFail(ctx, "test/repo", "push"))
	assert.True(t, injector.ShouldFail(ctx, "test/repo", "clone"))

	// Third call should not fail (count exhausted)
	assert.False(t, injector.ShouldFail(ctx, "test/repo", "fetch"))
}

// TestMockFailureInjectorRepoFiltering tests failure injector with specific repos
func TestMockFailureInjectorRepoFiltering(t *testing.T) {
	injector := &MockFailureInjector{
		FailureMode:  FailureAuth,
		FailureCount: 10, // High count to ensure repo filtering is the limiting factor
		FailureRepos: []string{"service-a", "service-c"},
	}

	ctx := context.Background()

	// Should fail for repos in the list
	assert.True(t, injector.ShouldFail(ctx, "org/service-a", "push"))
	assert.True(t, injector.ShouldFail(ctx, "org/service-c", "clone"))

	// Should not fail for repos not in the list
	assert.False(t, injector.ShouldFail(ctx, "org/service-b", "push"))
	assert.False(t, injector.ShouldFail(ctx, "org/service-d", "clone"))
}

// TestMockFailureInjectorRate tests failure injector with failure rate
func TestMockFailureInjectorRate(t *testing.T) {
	// Test with 100% failure rate
	injector := &MockFailureInjector{
		FailureMode: FailureTimeout,
		FailureRate: 1.0,
	}

	ctx := context.Background()

	// Should always fail with 100% rate
	failureCount := 0
	totalCalls := 10
	for i := 0; i < totalCalls; i++ {
		if injector.ShouldFail(ctx, "test/repo", "operation") {
			failureCount++
		}
	}

	// With rate-based failures, we can't guarantee exact counts due to time-based randomness
	// But with 100% rate, most should fail
	assert.Greater(t, failureCount, totalCalls/2)

	// Test with 0% failure rate
	injector.FailureRate = 0.0
	assert.False(t, injector.ShouldFail(ctx, "test/repo", "operation"))
}

// TestMockFailureInjectorGetFailureError tests error generation for different failure modes
func TestMockFailureInjectorGetFailureError(t *testing.T) {
	tests := []struct {
		name         string
		failureMode  FailureMode
		operation    string
		expectedText string
		expectedErr  error
	}{
		{
			name:         "Network failure",
			failureMode:  FailureNetwork,
			operation:    "git clone",
			expectedText: "network error: connection timeout",
			expectedErr:  ErrNetworkTimeout,
		},
		{
			name:         "Auth failure",
			failureMode:  FailureAuth,
			operation:    "git push",
			expectedText: "authentication failed: invalid credentials",
			expectedErr:  ErrAuthenticationFailed,
		},
		{
			name:         "Rate limit failure",
			failureMode:  FailureRateLimit,
			operation:    "api call",
			expectedText: "rate limit exceeded: too many requests",
			expectedErr:  ErrRateLimitExceeded,
		},
		{
			name:         "Partial failure",
			failureMode:  FailurePartial,
			operation:    "batch sync",
			expectedText: "partial failure: some operations failed",
			expectedErr:  ErrPartialFailure,
		},
		{
			name:         "Corruption failure",
			failureMode:  FailureCorruption,
			operation:    "data read",
			expectedText: "data corruption detected",
			expectedErr:  ErrDataCorruption,
		},
		{
			name:         "Timeout failure",
			failureMode:  FailureTimeout,
			operation:    "api request",
			expectedText: "operation timeout",
			expectedErr:  ErrOperationTimeout,
		},
		{
			name:         "Unknown failure",
			failureMode:  FailureMode(99), // Invalid mode
			operation:    "unknown op",
			expectedText: "unknown failure",
			expectedErr:  ErrUnknownFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			injector := &MockFailureInjector{
				FailureMode: tt.failureMode,
			}

			err := injector.GetFailureError(tt.operation)
			require.Error(t, err)

			assert.Contains(t, err.Error(), tt.expectedText)
			assert.Contains(t, err.Error(), tt.operation)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

// TestCleanup tests cleanup of generated test data
func TestCleanup(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	// Create some test repositories
	files := map[string]string{"README.md": "test"}
	_, err := generator.CreateRepo("test1", "owner", "main", files)
	require.NoError(t, err)

	_, err = generator.CreateRepo("test2", "owner", "main", files)
	require.NoError(t, err)

	// Verify directories exist
	assert.DirExists(t, tempDir)
	assert.DirExists(t, filepath.Join(tempDir, "owner-test1"))
	assert.DirExists(t, filepath.Join(tempDir, "owner-test2"))

	// Cleanup
	err = generator.Cleanup()
	require.NoError(t, err)

	// Verify directories are removed
	assert.NoDirExists(t, tempDir)
}

// TestCleanupEmptyBaseDir tests cleanup with empty base directory
func TestCleanupEmptyBaseDir(t *testing.T) {
	generator := &TestRepoGenerator{BaseDir: ""}
	err := generator.Cleanup()
	assert.NoError(t, err) // Should not error on empty base dir
}

// TestGenerateCommitSHA tests commit SHA generation
func TestGenerateCommitSHA(t *testing.T) {
	sha1 := generateCommitSHA()
	sha2 := generateCommitSHA()

	// Should generate different SHAs
	assert.NotEqual(t, sha1, sha2)

	// Should be 40 characters (hex representation of 20 bytes)
	assert.Len(t, sha1, 40)
	assert.Len(t, sha2, 40)

	// Should contain only hex characters
	for _, char := range sha1 {
		assert.True(t, (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f'))
	}
}

// TestStandardFileTemplates tests the standard file template functions
func TestStandardFileTemplates(t *testing.T) {
	tests := []struct {
		name     string
		function func() string
		contains []string
	}{
		{
			name:     "CI Workflow",
			function: getStandardCIWorkflow,
			contains: []string{"name: CI", "on:", "jobs:", "runs-on: ubuntu-latest", "make test", "make lint"},
		},
		{
			name:     "Release Workflow",
			function: getStandardReleaseWorkflow,
			contains: []string{"name: Release", "tags:", "create-release", "GITHUB_TOKEN"},
		},
		{
			name:     "Makefile",
			function: getStandardMakefile,
			contains: []string{".PHONY:", "test:", "build:", "lint:", "clean:", "go test", "go build"},
		},
		{
			name:     "Docker Compose",
			function: getStandardDockerCompose,
			contains: []string{"version:", "services:", "app:", "postgres:", "ports:", "environment:"},
		},
		{
			name:     "Setup Script",
			function: getStandardSetupScript,
			contains: []string{"#!/bin/bash", "set -e", "go mod download", "make test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function()
			assert.NotEmpty(t, result)

			for _, expectedText := range tt.contains {
				assert.Contains(t, result, expectedText)
			}
		})
	}
}

// TestErrorConstants tests that all error constants are properly defined
func TestErrorConstants(t *testing.T) {
	errors := []error{
		ErrNetworkTimeout,
		ErrAuthenticationFailed,
		ErrRateLimitExceeded,
		ErrPartialFailure,
		ErrDataCorruption,
		ErrOperationTimeout,
		ErrUnknownFailure,
		ErrNetworkPartition,
		ErrBranchProtection,
		ErrPushRejected,
		ErrUnauthorized,
		ErrForbidden,
		ErrFileNotFound,
		ErrNetworkFailure,
		ErrGitCloneFailed,
		ErrNetworkRefused,
		ErrNetworkUnreachable,
		ErrGitPushTimeout,
		ErrGitAuthRequired,
		ErrGitPermissionDenied,
		ErrRequestTimeout,
		ErrServiceDegradation,
	}

	for i, err := range errors {
		t.Run(err.Error(), func(t *testing.T) {
			require.Error(t, err, "Error %d should not be nil", i)
			assert.NotEmpty(t, err.Error(), "Error %d should have a message", i)
		})
	}
}

// TestFailureModeConstants tests failure mode constants
func TestFailureModeConstants(t *testing.T) {
	modes := []FailureMode{
		FailureNone,
		FailureNetwork,
		FailureAuth,
		FailureRateLimit,
		FailurePartial,
		FailureCorruption,
		FailureTimeout,
	}

	// Verify modes have distinct values
	seen := make(map[FailureMode]bool)
	for _, mode := range modes {
		assert.False(t, seen[mode], "Failure mode %v should be unique", mode)
		seen[mode] = true
	}

	// Verify FailureNone is 0
	assert.Equal(t, FailureNone, FailureMode(0))
}

// TestConfigGeneration tests the configuration generated in complex scenarios
func TestConfigGeneration(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	cfg := scenario.Config

	// Verify source configuration
	assert.Equal(t, "org/template-repo", cfg.Source.Repo)
	assert.Equal(t, "main", cfg.Source.Branch)

	// Verify defaults
	assert.Equal(t, "sync/template", cfg.Defaults.BranchPrefix)
	assert.Contains(t, cfg.Defaults.PRLabels, "automated-sync")
	assert.Contains(t, cfg.Defaults.PRLabels, "integration-test")

	// Verify targets have expected structure
	for i, target := range cfg.Targets {
		assert.True(t, strings.HasPrefix(target.Repo, "org/service-"))
		assert.NotEmpty(t, target.Files)
		assert.True(t, target.Transform.RepoName)
		assert.NotEmpty(t, target.Transform.Variables["SERVICE_NAME"])

		// Verify file mappings are valid
		for _, fileMapping := range target.Files {
			assert.NotEmpty(t, fileMapping.Src)
			assert.NotEmpty(t, fileMapping.Dest)
			// File mappings should be valid paths (can be relative paths with or without directories)
			assert.NotContains(t, fileMapping.Src, "..")
		}

		t.Logf("Target %d: %s with %d file mappings", i, target.Repo, len(target.Files))
	}
}

// TestStateGeneration tests the state generation in complex scenarios
func TestStateGeneration(t *testing.T) {
	tempDir := t.TempDir()
	generator := NewTestRepoGenerator(tempDir)

	scenario, err := generator.CreateComplexScenario()
	require.NoError(t, err)

	currentState := scenario.State

	// Verify source state
	assert.Equal(t, "org/template-repo", currentState.Source.Repo)
	assert.Equal(t, "main", currentState.Source.Branch)
	assert.Equal(t, scenario.SourceRepo.CommitSHA, currentState.Source.LatestCommit)
	assert.WithinDuration(t, time.Now(), currentState.Source.LastChecked, time.Minute)

	// Verify target states
	expectedRepos := []string{"org/service-a", "org/service-b", "org/service-c"}
	for _, repoName := range expectedRepos {
		targetState, exists := currentState.Targets[repoName]
		require.True(t, exists, "Target state should exist for %s", repoName)

		assert.Equal(t, repoName, targetState.Repo)
		assert.NotEmpty(t, targetState.LastSyncCommit)
		assert.NotNil(t, targetState.Status)
		assert.NotNil(t, targetState.LastSyncTime)
		assert.WithinDuration(t, time.Now(), *targetState.LastSyncTime, 24*time.Hour)
	}

	// Verify first repo is up-to-date
	serviceAState := currentState.Targets["org/service-a"]
	assert.Equal(t, state.StatusUpToDate, serviceAState.Status)
	assert.Equal(t, scenario.SourceRepo.CommitSHA, serviceAState.LastSyncCommit)

	// Verify other repos are behind
	serviceBState := currentState.Targets["org/service-b"]
	assert.Equal(t, state.StatusBehind, serviceBState.Status)
	assert.NotEqual(t, scenario.SourceRepo.CommitSHA, serviceBState.LastSyncCommit)
}
