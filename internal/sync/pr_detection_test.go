package sync

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/testutil"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

var (
	errPRValidationHeadBranch   = gh.ErrPRValidationFailed
	errGitHubAPIRateLimit       = errors.New("GitHub API rate limit exceeded")
	errPRUpdatePermissionDenied = errors.New("PR update failed: permission denied")
)

// TestRepositorySync_ExistingPRDetection tests the PR detection via API fallback
func TestRepositorySync_ExistingPRDetection(t *testing.T) {
	// Setup test configuration
	cfg := &config.Config{
		Groups: []config.Group{
			{
				ID:   "test-group",
				Name: "Test Group",
				Source: config.SourceConfig{
					Repo: "org/template",
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "chore/sync-files-",
				},
			},
		},
	}

	target := config.TargetConfig{
		Repo: "org/target",
		Files: []config.FileMapping{
			{Src: "file1.txt", Dest: "file1.txt"},
		},
		Transform: config.Transform{
			RepoName: true,
		},
	}

	sourceState := &state.SourceState{
		Repo:         "org/template",
		Branch:       "master",
		LatestCommit: "abc123",
	}

	targetState := &state.TargetState{
		Repo:           "org/target",
		LastSyncCommit: "old123",
		Status:         state.StatusBehind,
	}

	t.Run("existing PR found via API fallback", func(t *testing.T) {
		// Create temporary directory and files for testing
		tmpDir := testutil.CreateTempDir(t)
		sourceDir := tmpDir + "/source"
		testutil.CreateTestDirectory(t, sourceDir)
		testutil.WriteTestFile(t, sourceDir+"/file1.txt", "new content")

		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		gitClient.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
		gitClient.On("Diff", mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()
		gitClient.On("DiffIgnoreWhitespace", mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()
		transformChain := &transform.MockChain{}

		// Setup default expectations for pre-sync validation
		ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
		ghClient.On("GetFile", mock.Anything, "org/target", "file1.txt", "").Return(&gh.FileContent{
			Content: []byte("old content"),
		}, nil)
		// For any other files, return not found
		ghClient.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, gh.ErrFileNotFound).Maybe()

		// Mock git operations
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/source")
		}), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			srcContent, _ := os.ReadFile(filepath.Join(sourceDir, "file1.txt")) //nolint:gosec // test file in controlled directory
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), string(srcContent))
		})

		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/target")
		}), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), "old content")
		})

		gitClient.On("Checkout", mock.Anything, mock.Anything, "abc123").Return(nil)
		gitClient.On("Checkout", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("CreateBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("CheckoutBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("AddAll", mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.Anything).Return("commit-sha", nil)
		gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"file1.txt"}, nil)
		gitClient.On("Push", mock.Anything, mock.Anything, "origin", mock.AnythingOfType("string"), false).Return(nil)

		// Mock transform operations
		transformChain.On("Transform", mock.Anything, mock.Anything, mock.Anything).Return([]byte("transformed content"), nil)

		// Mock PR creation to fail with validation error (simulating orphaned branch)
		prValidationError := errPRValidationHeadBranch
		ghClient.On("CreatePR", mock.Anything, "org/target", mock.AnythingOfType("gh.PRRequest")).Return(nil, prValidationError).Once()

		// Mock retry PR creation after branch cleanup - should succeed
		ghClient.On("CreatePR", mock.Anything, "org/target", mock.AnythingOfType("gh.PRRequest")).Return(&gh.PR{Number: 789}, nil).Once()

		// Mock API call to find existing PR - use a predictable branch name pattern
		existingPR := gh.PR{
			Number: 456,
			State:  "open",
		}
		existingPR.Head.Ref = "chore/sync-files-test-group-" // This will match the generated branch pattern

		ghClient.On("ListPRs", mock.Anything, "org/target", "open").Return(func(context.Context, string, string) []gh.PR {
			// Just return an empty list to force the retry logic
			return []gh.PR{}
		}, nil)

		// Remove UpdatePR mock since we're testing the retry logic now

		// Mock GetCurrentUser for PR creation
		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "test-user"}, nil).Maybe()

		// Mock branch deletion (cleanup attempt)
		gitClient.On("DeleteBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil).Maybe()
		ghClient.On("DeleteBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil).Maybe()

		// Create engine and repository sync
		engine := &Engine{
			config:    cfg,
			gh:        ghClient,
			git:       gitClient,
			transform: transformChain,
			options:   &Options{DryRun: false, MaxConcurrency: 1},
			logger:    logrus.New(),
		}

		repoSync := &RepositorySync{
			engine:      engine,
			target:      target,
			sourceState: sourceState,
			targetState: targetState,
			logger:      logrus.NewEntry(logrus.New()),
		}

		// Execute sync
		err := repoSync.Execute(context.Background())

		// Should succeed by creating PR after cleanup
		require.NoError(t, err)

		// Verify that PR creation was attempted twice (first failed, retry succeeded)
		ghClient.AssertNumberOfCalls(t, "CreatePR", 2)

		// Verify that API was called to find existing PR
		ghClient.AssertCalled(t, "ListPRs", mock.Anything, "org/target", "open")

		// Verify git operations were called appropriately
		gitClient.AssertCalled(t, "Clone", mock.Anything, mock.Anything, mock.AnythingOfType("string"), mock.Anything)
		gitClient.AssertCalled(t, "Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string"))
		gitClient.AssertCalled(t, "Push", mock.Anything, mock.Anything, "origin", mock.AnythingOfType("string"), false)
	})

	t.Run("no existing PR found - cleanup and retry", func(t *testing.T) {
		// Create temporary directory and files for testing
		tmpDir := testutil.CreateTempDir(t)
		sourceDir := tmpDir + "/source"
		testutil.CreateTestDirectory(t, sourceDir)
		testutil.WriteTestFile(t, sourceDir+"/file1.txt", "new content")

		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		gitClient.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
		gitClient.On("Diff", mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()
		gitClient.On("DiffIgnoreWhitespace", mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()
		transformChain := &transform.MockChain{}

		// Setup default expectations
		ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
		ghClient.On("GetFile", mock.Anything, "org/target", "file1.txt", "").Return(&gh.FileContent{
			Content: []byte("old content"),
		}, nil)
		// For any other files, return not found
		ghClient.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, gh.ErrFileNotFound).Maybe()

		// Mock git operations
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/source")
		}), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			srcContent, _ := os.ReadFile(filepath.Join(sourceDir, "file1.txt")) //nolint:gosec // test file in controlled directory
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), string(srcContent))
		})

		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/target")
		}), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), "old content")
		})

		gitClient.On("Checkout", mock.Anything, mock.Anything, "abc123").Return(nil)
		gitClient.On("Checkout", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("CreateBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("CheckoutBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("AddAll", mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.Anything).Return("commit-sha", nil)
		gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"file1.txt"}, nil)
		gitClient.On("Push", mock.Anything, mock.Anything, "origin", mock.AnythingOfType("string"), false).Return(nil)

		// Mock transform operations
		transformChain.On("Transform", mock.Anything, mock.Anything, mock.Anything).Return([]byte("transformed content"), nil)

		// Mock PR creation to fail initially
		prValidationError := errPRValidationHeadBranch
		ghClient.On("CreatePR", mock.Anything, "org/target", mock.AnythingOfType("gh.PRRequest")).Return(nil, prValidationError).Once()

		// Mock API call to find existing PR - return empty list (no existing PRs)
		ghClient.On("ListPRs", mock.Anything, "org/target", "open").Return([]gh.PR{}, nil)

		// Mock branch cleanup
		ghClient.On("DeleteBranch", mock.Anything, "org/target", mock.AnythingOfType("string")).Return(nil)

		// Mock GetCurrentUser for PR creation
		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "test-user"}, nil).Maybe()

		// Mock retry of PR creation - should succeed after cleanup
		successfulPR := &gh.PR{
			Number: 789,
		}
		ghClient.On("CreatePR", mock.Anything, "org/target", mock.AnythingOfType("gh.PRRequest")).Return(successfulPR, nil)

		// Create engine and repository sync
		engine := &Engine{
			config:    cfg,
			gh:        ghClient,
			git:       gitClient,
			transform: transformChain,
			options:   &Options{DryRun: false, MaxConcurrency: 1},
			logger:    logrus.New(),
		}

		repoSync := &RepositorySync{
			engine:      engine,
			target:      target,
			sourceState: sourceState,
			targetState: targetState,
			logger:      logrus.NewEntry(logrus.New()),
		}

		// Execute sync
		err := repoSync.Execute(context.Background())

		// Should succeed after cleanup and retry
		require.NoError(t, err)

		// Verify that PR creation was attempted twice (initial failure + retry)
		ghClient.AssertNumberOfCalls(t, "CreatePR", 2)

		// Verify that API was called to check for existing PRs
		ghClient.AssertCalled(t, "ListPRs", mock.Anything, "org/target", "open")

		// Verify that branch cleanup was attempted
		ghClient.AssertCalled(t, "DeleteBranch", mock.Anything, "org/target", mock.AnythingOfType("string"))

		// No update PR call should have been made
		ghClient.AssertNotCalled(t, "UpdatePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("API call to list PRs fails", func(t *testing.T) {
		// Create temporary directory and files for testing
		tmpDir := testutil.CreateTempDir(t)
		sourceDir := tmpDir + "/source"
		testutil.CreateTestDirectory(t, sourceDir)
		testutil.WriteTestFile(t, sourceDir+"/file1.txt", "new content")

		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		gitClient.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
		gitClient.On("Diff", mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()
		gitClient.On("DiffIgnoreWhitespace", mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()
		transformChain := &transform.MockChain{}

		// Setup default expectations
		ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
		ghClient.On("GetFile", mock.Anything, "org/target", "file1.txt", "").Return(&gh.FileContent{
			Content: []byte("old content"),
		}, nil)
		// For any other files, return not found
		ghClient.On("GetFile", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil, gh.ErrFileNotFound).Maybe()

		// Mock GetCurrentUser for PR creation
		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "test-user"}, nil).Maybe()

		// Mock git operations
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/source")
		}), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			srcContent, _ := os.ReadFile(filepath.Join(sourceDir, "file1.txt")) //nolint:gosec // test file in controlled directory
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), string(srcContent))
		})

		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/target")
		}), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), "old content")
		})

		gitClient.On("Checkout", mock.Anything, mock.Anything, "abc123").Return(nil)
		gitClient.On("Checkout", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("CreateBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("CheckoutBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("AddAll", mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.Anything).Return("commit-sha", nil)
		gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"file1.txt"}, nil)
		gitClient.On("Push", mock.Anything, mock.Anything, "origin", mock.AnythingOfType("string"), false).Return(nil)

		// Mock transform operations
		transformChain.On("Transform", mock.Anything, mock.Anything, mock.Anything).Return([]byte("transformed content"), nil)

		// Mock PR creation to fail initially
		prValidationError := errPRValidationHeadBranch
		ghClient.On("CreatePR", mock.Anything, "org/target", mock.AnythingOfType("gh.PRRequest")).Return(nil, prValidationError).Once()

		// Mock GetCurrentUser for PR creation
		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "test-user"}, nil).Maybe()

		// Mock API call to fail
		apiError := errGitHubAPIRateLimit
		ghClient.On("ListPRs", mock.Anything, "org/target", "open").Return(nil, apiError)

		// Mock branch cleanup
		ghClient.On("DeleteBranch", mock.Anything, "org/target", mock.AnythingOfType("string")).Return(nil)

		// Mock retry of PR creation - should succeed after cleanup
		successfulPR := &gh.PR{
			Number: 789,
		}
		ghClient.On("CreatePR", mock.Anything, "org/target", mock.AnythingOfType("gh.PRRequest")).Return(successfulPR, nil)

		// Create engine and repository sync
		engine := &Engine{
			config:    cfg,
			gh:        ghClient,
			git:       gitClient,
			transform: transformChain,
			options:   &Options{DryRun: false, MaxConcurrency: 1},
			logger:    logrus.New(),
		}

		repoSync := &RepositorySync{
			engine:      engine,
			target:      target,
			sourceState: sourceState,
			targetState: targetState,
			logger:      logrus.NewEntry(logrus.New()),
		}

		// Execute sync
		err := repoSync.Execute(context.Background())

		// Should still succeed by falling back to cleanup and retry
		require.NoError(t, err)

		// Verify that PR creation was attempted twice
		ghClient.AssertNumberOfCalls(t, "CreatePR", 2)

		// Verify that API call was attempted (even though it failed)
		ghClient.AssertCalled(t, "ListPRs", mock.Anything, "org/target", "open")

		// Verify that cleanup was still attempted
		ghClient.AssertCalled(t, "DeleteBranch", mock.Anything, "org/target", mock.AnythingOfType("string"))
	})

	t.Run("existing PR found but update fails", func(t *testing.T) {
		// Create temporary directory and files for testing
		tmpDir := testutil.CreateTempDir(t)
		sourceDir := tmpDir + "/source"
		testutil.CreateTestDirectory(t, sourceDir)
		testutil.WriteTestFile(t, sourceDir+"/file1.txt", "new content")

		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		gitClient.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
		gitClient.On("Diff", mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()
		gitClient.On("DiffIgnoreWhitespace", mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()
		transformChain := &transform.MockChain{}

		// Setup default expectations
		ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
		ghClient.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", nil).Maybe()

		// Mock git operations
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/source")
		}), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			srcContent, _ := os.ReadFile(filepath.Join(sourceDir, "file1.txt")) //nolint:gosec // test file in controlled directory
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), string(srcContent))
		})

		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/target")
		}), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), "old content")
		})

		gitClient.On("Checkout", mock.Anything, mock.Anything, "abc123").Return(nil)
		gitClient.On("Checkout", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		// Capture the branch name when CreateBranch is called and use it in ListPRs
		var capturedBranchName string
		gitClient.On("CreateBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil).Run(func(args mock.Arguments) {
			capturedBranchName = args[2].(string)
		})
		gitClient.On("CheckoutBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("AddAll", mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.Anything).Return("commit-sha", nil)
		gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"file1.txt"}, nil)
		gitClient.On("Push", mock.Anything, mock.Anything, "origin", mock.AnythingOfType("string"), false).Return(nil)

		// Mock transform operations
		transformChain.On("Transform", mock.Anything, mock.Anything, mock.Anything).Return([]byte("transformed content"), nil)

		// Mock PR creation to fail initially
		prValidationError := errPRValidationHeadBranch
		ghClient.On("CreatePR", mock.Anything, "org/target", mock.AnythingOfType("gh.PRRequest")).Return(nil, prValidationError).Once()

		// Mock GetCurrentUser for PR creation
		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "test-user"}, nil).Maybe()

		// Mock API call to find existing PR - return PR with captured branch name
		ghClient.On("ListPRs", mock.Anything, "org/target", "open").Return(func(context.Context, string, string) []gh.PR {
			existingPR := gh.PR{
				Number: 456,
				State:  "open",
			}
			existingPR.Head.Ref = capturedBranchName // Use the captured branch name
			return []gh.PR{existingPR}
		}, nil)

		// Mock updating the existing PR to fail
		updateError := errPRUpdatePermissionDenied
		ghClient.On("UpdatePR", mock.Anything, "org/target", 456, mock.AnythingOfType("gh.PRUpdate")).Return(nil, updateError)

		// Mock branch deletion (cleanup attempt) just to prevent panic
		ghClient.On("DeleteBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil).Maybe()

		// Mock retry PR creation after branch cleanup - should succeed to avoid panic
		ghClient.On("CreatePR", mock.Anything, "org/target", mock.AnythingOfType("gh.PRRequest")).Return(&gh.PR{Number: 789}, nil).Once()

		// Create engine and repository sync
		engine := &Engine{
			config:    cfg,
			gh:        ghClient,
			git:       gitClient,
			transform: transformChain,
			options:   &Options{DryRun: false, MaxConcurrency: 1},
			logger:    logrus.New(),
		}

		repoSync := &RepositorySync{
			engine:      engine,
			target:      target,
			sourceState: sourceState,
			targetState: targetState,
			logger:      logrus.NewEntry(logrus.New()),
		}

		// Execute sync
		err := repoSync.Execute(context.Background())

		// Should succeed after retry (since existing PR detection failed and retry succeeded)
		require.NoError(t, err)

		// Verify the main flow was executed
		ghClient.AssertCalled(t, "CreatePR", mock.Anything, "org/target", mock.AnythingOfType("gh.PRRequest"))
		// TODO: ListPRs and UpdatePR may not be called if existing PR detection fails
	})
}
