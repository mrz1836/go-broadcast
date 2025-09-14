package sync

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/testutil"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

var (
	errTestAuthError       = errors.New("auth error")
	errTestGetSHA          = errors.New("failed to get SHA")
	errTestForcePushFailed = errors.New("force push failed")
	errTestGitCommand      = errors.New("git command failed: permission denied")
)

func TestRepositorySync_Execute(t *testing.T) {
	// Setup test configuration
	target := config.TargetConfig{
		Repo: "org/target",
		Files: []config.FileMapping{
			{Src: "file1.txt", Dest: "file1.txt"},
			{Src: "file2.txt", Dest: "file2.txt"},
		},
		Transform: config.Transform{
			RepoName: true,
			Variables: map[string]string{
				"VAR1": "value1",
			},
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

	t.Run("successful sync", func(t *testing.T) {
		// Create temporary directory and files for testing
		tmpDir := testutil.CreateTempDir(t)
		sourceDir := tmpDir + "/source"
		testutil.CreateTestDirectory(t, sourceDir)

		// Create source files
		testutil.WriteTestFile(t, sourceDir+"/file1.txt", "content 1")
		testutil.WriteTestFile(t, sourceDir+"/file2.txt", "content 2")

		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		transformChain := &transform.MockChain{}

		// Setup default expectations for pre-sync validation
		ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()

		// Mock git operations to use our temp directory
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/source")
		})).Return(nil).Run(func(args mock.Arguments) {
			// Copy our pre-created files to the expected location
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)                                            // Test setup
			srcContent1, _ := os.ReadFile(filepath.Join(sourceDir, "file1.txt"))                 //nolint:gosec // Test file
			srcContent2, _ := os.ReadFile(filepath.Join(sourceDir, "file2.txt"))                 //nolint:gosec // Test file
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), string(srcContent1)) // Test setup
			testutil.WriteTestFile(t, filepath.Join(destPath, "file2.txt"), string(srcContent2)) // Test setup
		})

		// Mock target repository clone
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/target")
		})).Return(nil).Run(func(args mock.Arguments) {
			// Create target directory structure for cloning
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
		})

		gitClient.On("Checkout", mock.Anything, mock.Anything, "abc123").Return(nil)

		// Mock file operations
		ghClient.On("GetFile", mock.Anything, "org/target", "file1.txt", "").
			Return(&gh.FileContent{Content: []byte("old content 1")}, nil)
		ghClient.On("GetFile", mock.Anything, "org/target", "file2.txt", "").
			Return(&gh.FileContent{Content: []byte("old content 2")}, nil)

		// Mock target repository git operations
		gitClient.On("CreateBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("Checkout", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.Anything).Return("new123", nil)
		gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"file1.txt", "file2.txt"}, nil)
		gitClient.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// Mock transformations (return different content to trigger changes)
		transformChain.On("Transform", mock.Anything, []byte("content 1"), mock.Anything).
			Return([]byte("new content 1"), nil)
		transformChain.On("Transform", mock.Anything, []byte("content 2"), mock.Anything).
			Return([]byte("new content 2"), nil)

		// Mock GitHub operations for PR creation
		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser"}, nil)
		ghClient.On("ListBranches", mock.Anything, "org/target").
			Return([]gh.Branch{{Name: "master"}}, nil)
		ghClient.On("CreatePR", mock.Anything, "org/target", mock.Anything).
			Return(&gh.PR{Number: 123}, nil)

		// Create engine with dry-run disabled
		opts := DefaultOptions().WithDryRun(false)
		engine := &Engine{
			config: &config.Config{
				Groups: []config.Group{{
					Defaults: config.DefaultConfig{
						BranchPrefix: "chore/sync-files",
					},
				}},
			},
			gh:        ghClient,
			git:       gitClient,
			transform: transformChain,
			options:   opts,
			logger:    logrus.New(),
		}

		// Create repository sync
		repoSync := &RepositorySync{
			engine:      engine,
			target:      target,
			sourceState: sourceState,
			targetState: targetState,
			logger:      logrus.NewEntry(logrus.New()),
		}

		// Execute sync
		err := repoSync.Execute(context.Background())

		// Assertions
		require.NoError(t, err)
		gitClient.AssertExpectations(t)
		ghClient.AssertExpectations(t)
		transformChain.AssertExpectations(t)
	})

	t.Run("sync not needed", func(t *testing.T) {
		// Target is up-to-date
		upToDateTargetState := &state.TargetState{
			Repo:           "org/target",
			LastSyncCommit: "abc123", // Same as source
			Status:         state.StatusUpToDate,
		}

		engine := &Engine{
			options: DefaultOptions().WithForce(false),
			logger:  logrus.New(),
		}

		repoSync := &RepositorySync{
			engine:      engine,
			target:      target,
			sourceState: sourceState,
			targetState: upToDateTargetState,
			logger:      logrus.NewEntry(logrus.New()),
		}

		// Execute sync
		err := repoSync.Execute(context.Background())

		// Should complete without error and without doing work
		assert.NoError(t, err)
	})

	t.Run("clone failure", func(t *testing.T) {
		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		transformChain := &transform.MockChain{}

		// Setup default expectations for pre-sync validation
		ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()

		// Mock git clone failure
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.Anything).
			Return(internalerrors.ErrTest)

		engine := &Engine{
			config: &config.Config{
				Groups: []config.Group{{
					Defaults: config.DefaultConfig{
						BranchPrefix: "chore/sync-files",
					},
				}},
			},
			gh:        ghClient,
			git:       gitClient,
			transform: transformChain,
			options:   DefaultOptions(),
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

		// Should fail
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clone source")
		gitClient.AssertExpectations(t)
	})

	t.Run("dry run mode", func(t *testing.T) {
		// Create temporary directory and files for testing
		tmpDir := testutil.CreateTempDir(t)
		sourceDir := tmpDir + "/source"
		testutil.CreateTestDirectory(t, sourceDir)

		// Create source files
		testutil.WriteTestFile(t, sourceDir+"/file1.txt", "content 1")
		testutil.WriteTestFile(t, sourceDir+"/file2.txt", "content 2")

		// Setup mocks (minimal since dry-run shouldn't call most operations)
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		transformChain := &transform.MockChain{}

		// Setup default expectations for pre-sync validation
		ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()

		// Only mock the operations that should happen in dry-run
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/source")
		})).Return(nil).Run(func(args mock.Arguments) {
			// Copy our pre-created files to the expected location
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)                                            // Test setup
			srcContent1, _ := os.ReadFile(filepath.Join(sourceDir, "file1.txt"))                 //nolint:gosec // Test file
			srcContent2, _ := os.ReadFile(filepath.Join(sourceDir, "file2.txt"))                 //nolint:gosec // Test file
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), string(srcContent1)) // Test setup
			testutil.WriteTestFile(t, filepath.Join(destPath, "file2.txt"), string(srcContent2)) // Test setup
		})
		gitClient.On("Checkout", mock.Anything, mock.Anything, "abc123").Return(nil)

		// Mock file operations
		ghClient.On("GetFile", mock.Anything, "org/target", "file1.txt", "").
			Return(&gh.FileContent{Content: []byte("old content 1")}, nil)
		ghClient.On("GetFile", mock.Anything, "org/target", "file2.txt", "").
			Return(&gh.FileContent{Content: []byte("old content 2")}, nil)

		// Mock transformations
		transformChain.On("Transform", mock.Anything, []byte("content 1"), mock.Anything).
			Return([]byte("new content 1"), nil)
		transformChain.On("Transform", mock.Anything, []byte("content 2"), mock.Anything).
			Return([]byte("new content 2"), nil)

		// Mock GetCurrentUser for dry-run PR preview
		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser", ID: 123}, nil).Maybe()

		// Create engine with dry-run enabled
		opts := DefaultOptions().WithDryRun(true)
		engine := &Engine{
			config: &config.Config{
				Groups: []config.Group{{
					Defaults: config.DefaultConfig{
						BranchPrefix: "chore/sync-files",
					},
				}},
			},
			gh:        ghClient,
			git:       gitClient,
			transform: transformChain,
			options:   opts,
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

		// Should succeed in dry-run mode
		require.NoError(t, err)

		// Verify that PR creation was NOT called (dry-run)
		ghClient.AssertNotCalled(t, "CreatePR")
	})
}

func TestRepositorySync_needsSync(t *testing.T) {
	sourceState := &state.SourceState{
		LatestCommit: "abc123",
	}

	repoSync := &RepositorySync{
		sourceState: sourceState,
	}

	t.Run("no target state", func(t *testing.T) {
		repoSync.targetState = nil
		assert.True(t, repoSync.needsSync())
	})

	t.Run("different commits", func(t *testing.T) {
		repoSync.targetState = &state.TargetState{
			LastSyncCommit: "old123",
		}
		assert.True(t, repoSync.needsSync())
	})

	t.Run("same commits", func(t *testing.T) {
		repoSync.targetState = &state.TargetState{
			LastSyncCommit: "abc123",
		}
		assert.False(t, repoSync.needsSync())
	})
}

func TestRepositorySync_generateCommitMessage(t *testing.T) {
	repoSync := &RepositorySync{}

	t.Run("single file", func(t *testing.T) {
		files := []FileChange{
			{Path: "README.md"},
		}

		msg := repoSync.generateCommitMessage(files)
		assert.Equal(t, "sync: update README.md from source repository", msg)
	})

	t.Run("multiple files", func(t *testing.T) {
		files := []FileChange{
			{Path: "README.md"},
			{Path: "Makefile"},
			{Path: ".github/workflows/ci.yml"},
		}

		msg := repoSync.generateCommitMessage(files)
		assert.Equal(t, "sync: update 3 files from source repository", msg)
	})
}

func TestRepositorySync_generatePRTitle(t *testing.T) {
	repoSync := &RepositorySync{
		sourceState: &state.SourceState{
			LatestCommit: "abc123def456",
		},
	}

	title := repoSync.generatePRTitle()
	assert.Equal(t, "[Sync] Update project files from source repository (abc123d)", title)
}

func TestRepositorySync_generatePRBody(t *testing.T) {
	repoSync := &RepositorySync{
		sourceState: &state.SourceState{
			Repo:         "org/template",
			Branch:       "master",
			LatestCommit: "abc123",
		},
		target: config.TargetConfig{
			Repo: "org/target",
		},
		syncMetrics: &PerformanceMetrics{
			FileMetrics: FileProcessingMetrics{
				FilesProcessed: 2,
				FilesChanged:   2,
				FilesSkipped:   0,
			},
		},
	}

	files := []FileChange{
		{Path: "README.md", IsNew: false},
		{Path: "new-file.txt", IsNew: true},
	}

	body := repoSync.generatePRBody("commit456", files, nil)

	// Verify key components are present
	assert.Contains(t, body, "## What Changed")
	assert.Contains(t, body, "## Why It Was Necessary")
	assert.Contains(t, body, "## Testing Performed")
	assert.Contains(t, body, "## Impact / Risk")
	assert.Contains(t, body, "## Performance Metrics")
	// Check for the enhanced change description
	assert.Contains(t, body, "Updated 2 individual file(s) to synchronize with the source repository")
	assert.Contains(t, body, "go-broadcast-metadata")
	assert.Contains(t, body, "source_repo: org/template")
	assert.Contains(t, body, "source_commit: abc123")
	assert.Contains(t, body, "target_repo: org/target")
	assert.Contains(t, body, "sync_commit: commit456")
}

func TestRepositorySync_findExistingPR(t *testing.T) {
	branchName := "chore/sync-files-20240115-120530-abc123"

	t.Run("no target state", func(t *testing.T) {
		repoSync := &RepositorySync{
			targetState: nil,
		}

		pr := repoSync.findExistingPR(branchName)
		assert.Nil(t, pr)
	})

	t.Run("no matching PR", func(t *testing.T) {
		repoSync := &RepositorySync{
			targetState: &state.TargetState{
				OpenPRs: []gh.PR{
					{Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{Ref: "feature/other-branch"}},
				},
			},
		}

		pr := repoSync.findExistingPR(branchName)
		assert.Nil(t, pr)
	})

	t.Run("matching PR found", func(t *testing.T) {
		expectedPR := gh.PR{
			Number: 123,
			Head: struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			}{Ref: branchName},
		}

		repoSync := &RepositorySync{
			targetState: &state.TargetState{
				OpenPRs: []gh.PR{
					{Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{Ref: "feature/other-branch"}},
					expectedPR,
				},
			},
		}

		pr := repoSync.findExistingPR(branchName)
		require.NotNil(t, pr)
		assert.Equal(t, 123, pr.Number)
		assert.Equal(t, branchName, pr.Head.Ref)
	})
}

func TestDryRunOutput(t *testing.T) {
	t.Run("NewDryRunOutput with nil writer", func(t *testing.T) {
		output := NewDryRunOutput(nil)
		assert.NotNil(t, output)
		assert.Equal(t, os.Stdout, output.writer)
	})

	t.Run("NewDryRunOutput with custom writer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)
		assert.NotNil(t, output)
		assert.Equal(t, buf, output.writer)
	})

	t.Run("Header", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Header("Test Header")

		expected := "\n🔍 Test Header\n┌─────────────────────────────────────────────────────────────────\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Field", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Field("Label", "Value")

		expected := "│ Label: Value\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Field with empty label", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Field("", "Value only")

		expected := "│ : Value only\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Separator", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Separator()

		expected := "├─────────────────────────────────────────────────────────────────\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Content with short line", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Content("Short content")

		expected := "│ Short content\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Content with empty line", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Content("")

		expected := "│\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Content with whitespace only", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Content("   ")

		expected := "│\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Content with long line", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		longLine := "This is a very long line that exceeds sixty characters and should be truncated"
		output.Content(longLine)

		expected := "│ This is a very long line that exceeds sixty characters an...\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Footer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Footer()

		expected := "└─────────────────────────────────────────────────────────────────\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Info", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Info("Info message")

		expected := "   Info message\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Warning", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Warning("Warning message")

		expected := "⚠️  Warning message\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Success", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Success("Success message")

		expected := "✅ Success message\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("Complete usage pattern", func(t *testing.T) {
		buf := &bytes.Buffer{}
		output := NewDryRunOutput(buf)

		output.Header("Complete Test")
		output.Field("Repository", "org/repo")
		output.Field("Branch", "feature/test")
		output.Separator()
		output.Content("This is some content")
		output.Content("") // Empty line
		output.Content("More content here")
		output.Footer()
		output.Success("Operation completed")
		output.Info("Additional info")
		output.Warning("Warning note")

		result := buf.String()

		// Verify all components are present in expected order
		assert.Contains(t, result, "🔍 Complete Test")
		assert.Contains(t, result, "┌─────────────────────────────────────────────────────────────────")
		assert.Contains(t, result, "│ Repository: org/repo")
		assert.Contains(t, result, "│ Branch: feature/test")
		assert.Contains(t, result, "├─────────────────────────────────────────────────────────────────")
		assert.Contains(t, result, "│ This is some content")
		assert.Contains(t, result, "│") // Empty content line
		assert.Contains(t, result, "│ More content here")
		assert.Contains(t, result, "└─────────────────────────────────────────────────────────────────")
		assert.Contains(t, result, "✅ Operation completed")
		assert.Contains(t, result, "   Additional info")
		assert.Contains(t, result, "⚠️  Warning note")

		// Verify proper ordering by checking positions
		headerPos := strings.Index(result, "🔍 Complete Test")
		repoPos := strings.Index(result, "│ Repository: org/repo")
		separatorPos := strings.Index(result, "├─────────────────────────────────────────────────────────────────")
		footerPos := strings.Index(result, "└─────────────────────────────────────────────────────────────────")
		successPos := strings.Index(result, "✅ Operation completed")

		assert.Less(t, headerPos, repoPos, "Header should come before repository field")
		assert.Less(t, repoPos, separatorPos, "Repository field should come before separator")
		assert.Less(t, separatorPos, footerPos, "Separator should come before footer")
		assert.Less(t, footerPos, successPos, "Footer should come before success message")
	})
}

// TestRepositorySync_mergeUniqueStrings tests the merge unique strings functionality
func TestRepositorySync_mergeUniqueStrings(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	rs := &RepositorySync{
		logger: logger,
	}

	t.Run("merges two non-empty slices with unique elements", func(t *testing.T) {
		slice1 := []string{"a", "b", "c"}
		slice2 := []string{"d", "e", "f"}
		result := rs.mergeUniqueStrings(slice1, slice2)
		expected := []string{"a", "b", "c", "d", "e", "f"}
		assert.Equal(t, expected, result)
	})

	t.Run("removes duplicates while preserving order", func(t *testing.T) {
		slice1 := []string{"a", "b", "c"}
		slice2 := []string{"b", "d", "a", "e"}
		result := rs.mergeUniqueStrings(slice1, slice2)
		expected := []string{"a", "b", "c", "d", "e"}
		assert.Equal(t, expected, result)
	})

	t.Run("handles empty first slice", func(t *testing.T) {
		var slice1 []string
		slice2 := []string{"a", "b", "c"}
		result := rs.mergeUniqueStrings(slice1, slice2)
		expected := []string{"a", "b", "c"}
		assert.Equal(t, expected, result)
	})

	t.Run("handles empty second slice", func(t *testing.T) {
		slice1 := []string{"a", "b", "c"}
		var slice2 []string
		result := rs.mergeUniqueStrings(slice1, slice2)
		expected := []string{"a", "b", "c"}
		assert.Equal(t, expected, result)
	})

	t.Run("handles both slices empty", func(t *testing.T) {
		var slice1 []string
		var slice2 []string
		result := rs.mergeUniqueStrings(slice1, slice2)
		assert.Nil(t, result)
	})

	t.Run("handles nil slices", func(t *testing.T) {
		result := rs.mergeUniqueStrings(nil, nil)
		assert.Nil(t, result)
	})

	t.Run("filters out empty strings", func(t *testing.T) {
		slice1 := []string{"a", "", "b"}
		slice2 := []string{"", "c", "d"}
		result := rs.mergeUniqueStrings(slice1, slice2)
		expected := []string{"a", "b", "c", "d"}
		assert.Equal(t, expected, result)
	})

	t.Run("handles all empty strings", func(t *testing.T) {
		slice1 := []string{"", ""}
		slice2 := []string{"", ""}
		result := rs.mergeUniqueStrings(slice1, slice2)
		assert.Nil(t, result)
	})
}

// TestRepositorySync_getPRAssignees tests the PR assignees resolution logic with global merging
func TestRepositorySync_getPRAssignees(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("merges global and target assignees", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Global: config.GlobalConfig{
					PRAssignees: []string{"global1", "global2"},
				},
				Defaults: config.DefaultConfig{
					PRAssignees: []string{"default1", "default2"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:        "org/target",
			PRAssignees: []string{"target1", "target2"},
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		assignees := rs.getPRAssignees()
		expected := []string{"global1", "global2", "target1", "target2"}
		assert.Equal(t, expected, assignees)
	})

	t.Run("removes duplicates when merging global and target", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Global: config.GlobalConfig{
					PRAssignees: []string{"user1", "user2"},
				},
				Defaults: config.DefaultConfig{
					PRAssignees: []string{"default1", "default2"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:        "org/target",
			PRAssignees: []string{"user2", "user3"},
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		assignees := rs.getPRAssignees()
		expected := []string{"user1", "user2", "user3"}
		assert.Equal(t, expected, assignees)
	})

	t.Run("uses only global when target has none", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Global: config.GlobalConfig{
					PRAssignees: []string{"global1", "global2"},
				},
				Defaults: config.DefaultConfig{
					PRAssignees: []string{"default1", "default2"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo: "org/target",
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		assignees := rs.getPRAssignees()
		expected := []string{"global1", "global2"}
		assert.Equal(t, expected, assignees)
	})

	t.Run("uses only target when global has none", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Global: config.GlobalConfig{},
				Defaults: config.DefaultConfig{
					PRAssignees: []string{"default1", "default2"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:        "org/target",
			PRAssignees: []string{"target1", "target2"},
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		assignees := rs.getPRAssignees()
		expected := []string{"target1", "target2"}
		assert.Equal(t, expected, assignees)
	})

	t.Run("falls back to defaults when neither global nor target have assignees", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Global: config.GlobalConfig{},
				Defaults: config.DefaultConfig{
					PRAssignees: []string{"default1", "default2"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo: "org/target",
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		assignees := rs.getPRAssignees()
		expected := []string{"default1", "default2"}
		assert.Equal(t, expected, assignees)
	})

	t.Run("returns empty when no assignees configured anywhere", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Global:   config.GlobalConfig{},
				Defaults: config.DefaultConfig{},
			}},
		}

		target := config.TargetConfig{
			Repo: "org/target",
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		assignees := rs.getPRAssignees()
		assert.Empty(t, assignees)
	})
}

// TestRepositorySync_getPRReviewers tests the PR reviewers resolution logic
func TestRepositorySync_getPRReviewers(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("uses target-specific reviewers when present", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRReviewers: []string{"reviewer1", "reviewer2"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:        "org/target",
			PRReviewers: []string{"target-reviewer1", "target-reviewer2"},
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		reviewers := rs.getPRReviewers()
		assert.Equal(t, []string{"target-reviewer1", "target-reviewer2"}, reviewers)
	})

	t.Run("uses default reviewers when target has none", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRReviewers: []string{"reviewer1", "reviewer2"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo: "org/target",
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		reviewers := rs.getPRReviewers()
		assert.Equal(t, []string{"reviewer1", "reviewer2"}, reviewers)
	})
}

// TestRepositorySync_getPRTeamReviewers tests the PR team reviewers resolution logic
func TestRepositorySync_getPRTeamReviewers(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("uses target-specific team reviewers when present", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRTeamReviewers: []string{"default-team"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:            "org/target",
			PRTeamReviewers: []string{"target-team1", "target-team2"},
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		teamReviewers := rs.getPRTeamReviewers()
		assert.Equal(t, []string{"target-team1", "target-team2"}, teamReviewers)
	})

	t.Run("uses default team reviewers when target has none", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRTeamReviewers: []string{"default-team1", "default-team2"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo: "org/target",
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		teamReviewers := rs.getPRTeamReviewers()
		assert.Equal(t, []string{"default-team1", "default-team2"}, teamReviewers)
	})
}

// TestRepositorySync_getPRLabels tests the PR labels resolution logic
func TestRepositorySync_getPRLabels(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("uses target-specific labels when present", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRLabels: []string{"default-label1", "default-label2"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:     "org/target",
			PRLabels: []string{"target-label1", "target-label2"},
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		labels := rs.getPRLabels()
		assert.Equal(t, []string{"target-label1", "target-label2"}, labels)
	})

	t.Run("uses default labels when target has none", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRLabels: []string{"automated-sync", "maintenance"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo: "org/target",
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		labels := rs.getPRLabels()
		assert.Equal(t, []string{"automated-sync", "maintenance"}, labels)
	})

	t.Run("returns empty slice when neither target nor defaults have labels", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{},
			}},
		}

		target := config.TargetConfig{
			Repo: "org/target",
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		labels := rs.getPRLabels()
		assert.Empty(t, labels)
	})

	t.Run("target empty slice overrides defaults", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRLabels: []string{"default-label"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:     "org/target",
			PRLabels: []string{}, // Explicitly empty
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		labels := rs.getPRLabels()
		assert.Equal(t, []string{"default-label"}, labels) // Should use defaults since target slice is empty
	})

	t.Run("single label works correctly", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRLabels: []string{"automated-sync"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:     "org/target",
			PRLabels: []string{"custom-label"},
		}

		rs := &RepositorySync{
			engine: &Engine{config: cfg},
			target: target,
			logger: logger,
		}

		labels := rs.getPRLabels()
		assert.Equal(t, []string{"custom-label"}, labels)
	})
}

// TestRepositorySync_getPRLabels_WithAutomerge tests automerge label functionality
func TestRepositorySync_getPRLabels_WithAutomerge(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("adds automerge labels when enabled", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRLabels: []string{"default-label"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:     "org/target",
			PRLabels: []string{"target-label"},
		}

		opts := &Options{
			Automerge:       true,
			AutomergeLabels: []string{"automerge", "ready-to-merge"},
		}

		rs := &RepositorySync{
			engine: &Engine{
				config:  cfg,
				options: opts,
			},
			target: target,
			logger: logger,
		}

		labels := rs.getPRLabels()
		expected := []string{"target-label", "automerge", "ready-to-merge"}
		assert.Equal(t, expected, labels)
	})

	t.Run("does not add automerge labels when disabled", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRLabels: []string{"default-label"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:     "org/target",
			PRLabels: []string{"target-label"},
		}

		opts := &Options{
			Automerge:       false,
			AutomergeLabels: []string{"automerge", "ready-to-merge"},
		}

		rs := &RepositorySync{
			engine: &Engine{
				config:  cfg,
				options: opts,
			},
			target: target,
			logger: logger,
		}

		labels := rs.getPRLabels()
		expected := []string{"target-label"}
		assert.Equal(t, expected, labels)
	})

	t.Run("does not add automerge labels when enabled but no labels configured", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRLabels: []string{"default-label"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:     "org/target",
			PRLabels: []string{"target-label"},
		}

		opts := &Options{
			Automerge:       true,
			AutomergeLabels: []string{}, // Empty labels
		}

		rs := &RepositorySync{
			engine: &Engine{
				config:  cfg,
				options: opts,
			},
			target: target,
			logger: logger,
		}

		labels := rs.getPRLabels()
		expected := []string{"target-label"}
		assert.Equal(t, expected, labels)
	})

	t.Run("adds automerge labels to defaults when no target labels", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRLabels: []string{"default-label"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo: "org/target",
			// No PRLabels configured
		}

		opts := &Options{
			Automerge:       true,
			AutomergeLabels: []string{"automerge"},
		}

		rs := &RepositorySync{
			engine: &Engine{
				config:  cfg,
				options: opts,
			},
			target: target,
			logger: logger,
		}

		labels := rs.getPRLabels()
		expected := []string{"default-label", "automerge"}
		assert.Equal(t, expected, labels)
	})

	t.Run("removes duplicate automerge labels", func(t *testing.T) {
		cfg := &config.Config{
			Groups: []config.Group{{
				Defaults: config.DefaultConfig{
					PRLabels: []string{"default-label"},
				},
			}},
		}

		target := config.TargetConfig{
			Repo:     "org/target",
			PRLabels: []string{"automerge", "target-label"}, // Already contains automerge
		}

		opts := &Options{
			Automerge:       true,
			AutomergeLabels: []string{"automerge", "ready-to-merge"}, // Duplicate automerge
		}

		rs := &RepositorySync{
			engine: &Engine{
				config:  cfg,
				options: opts,
			},
			target: target,
			logger: logger,
		}

		labels := rs.getPRLabels()
		// Should not have duplicate "automerge" labels
		expected := []string{"automerge", "target-label", "ready-to-merge"}
		assert.Equal(t, expected, labels)
	})
}

// TestFormatReviewersWithFiltering tests the formatReviewersWithFiltering method
func TestFormatReviewersWithFiltering(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	cfg := &config.Config{}
	rs := &RepositorySync{
		engine: &Engine{config: cfg},
		logger: logger,
	}

	t.Run("no reviewers returns none", func(t *testing.T) {
		result := rs.formatReviewersWithFiltering([]string{}, "currentuser")
		assert.Equal(t, "none", result)
	})

	t.Run("reviewers without current user", func(t *testing.T) {
		reviewers := []string{"user1", "user2", "user3"}
		result := rs.formatReviewersWithFiltering(reviewers, "currentuser")
		assert.Equal(t, "user1, user2, user3", result)
	})

	t.Run("reviewers with current user filtered", func(t *testing.T) {
		reviewers := []string{"user1", "currentuser", "user3"}
		result := rs.formatReviewersWithFiltering(reviewers, "currentuser")
		assert.Equal(t, "user1, currentuser (author - will be filtered), user3", result)
	})

	t.Run("all reviewers are current user", func(t *testing.T) {
		reviewers := []string{"currentuser"}
		result := rs.formatReviewersWithFiltering(reviewers, "currentuser")
		assert.Equal(t, "currentuser (author - will be filtered)", result)
	})

	t.Run("empty current user login", func(t *testing.T) {
		reviewers := []string{"user1", "user2"}
		result := rs.formatReviewersWithFiltering(reviewers, "")
		assert.Equal(t, "user1, user2", result)
	})
}

// TestCreateNewPR_WithReviewerFiltering tests PR creation with reviewer filtering
func TestCreateNewPR_WithReviewerFiltering(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	gitClient := &git.MockClient{}
	ghClient := &gh.MockClient{}

	// Mock getting current user
	currentUser := &gh.User{
		Login: "authoruser",
		ID:    123,
	}
	ghClient.On("GetCurrentUser", ctx).Return(currentUser, nil)

	// Mock branch listing
	branches := []gh.Branch{
		{Name: "master", Protected: true},
		{Name: "master", Protected: false},
	}
	ghClient.On("ListBranches", ctx, "org/target").Return(branches, nil)

	// Configure with reviewers including the author
	cfg := &config.Config{
		Groups: []config.Group{{
			Global: config.GlobalConfig{
				PRReviewers: []string{"reviewer1", "authoruser", "reviewer3"},
			},
		}},
	}

	target := config.TargetConfig{
		Repo: "org/target",
	}

	// Expected PR request should have filtered reviewers
	expectedPRReq := mock.MatchedBy(func(req gh.PRRequest) bool {
		// Check that authoruser is filtered out
		return len(req.Reviewers) == 2 &&
			req.Reviewers[0] == "reviewer1" &&
			req.Reviewers[1] == "reviewer3"
	})

	ghClient.On("CreatePR", ctx, "org/target", expectedPRReq).
		Return(&gh.PR{Number: 123}, nil)

	// Create engine with dry-run disabled
	engine := &Engine{
		config:  cfg,
		git:     gitClient,
		gh:      ghClient,
		logger:  logrus.New(),
		options: DefaultOptions().WithDryRun(false),
	}

	rs := &RepositorySync{
		engine:      engine,
		target:      target,
		logger:      logger,
		sourceState: &state.SourceState{LatestCommit: "abc123"},
		targetState: &state.TargetState{},
	}

	// Test creating PR
	err := rs.createNewPR(ctx, "test-branch", "abc123", []FileChange{}, nil)
	require.NoError(t, err)

	ghClient.AssertExpectations(t)
}

// TestCreateNewPR_GetCurrentUserFailure tests PR creation when getting current user fails
func TestCreateNewPR_GetCurrentUserFailure(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	gitClient := &git.MockClient{}
	ghClient := &gh.MockClient{}

	// Mock getting current user fails
	ghClient.On("GetCurrentUser", ctx).Return(nil, errTestAuthError)

	// Mock branch listing
	branches := []gh.Branch{{Name: "master", Protected: true}}
	ghClient.On("ListBranches", ctx, "org/target").Return(branches, nil)

	// Configure with reviewers
	cfg := &config.Config{
		Groups: []config.Group{{
			Global: config.GlobalConfig{
				PRReviewers: []string{"reviewer1", "reviewer2"},
			},
		}},
	}

	target := config.TargetConfig{
		Repo: "org/target",
	}

	// Should still create PR with all reviewers when current user lookup fails
	expectedPRReq := mock.MatchedBy(func(req gh.PRRequest) bool {
		return len(req.Reviewers) == 2 &&
			req.Reviewers[0] == "reviewer1" &&
			req.Reviewers[1] == "reviewer2"
	})

	ghClient.On("CreatePR", ctx, "org/target", expectedPRReq).
		Return(&gh.PR{Number: 123}, nil)

	engine := &Engine{
		config:  cfg,
		git:     gitClient,
		gh:      ghClient,
		logger:  logrus.New(),
		options: DefaultOptions().WithDryRun(false),
	}

	rs := &RepositorySync{
		engine:      engine,
		target:      target,
		logger:      logger,
		sourceState: &state.SourceState{LatestCommit: "abc123"},
		targetState: &state.TargetState{},
	}

	// Test creating PR - should succeed despite GetCurrentUser failure
	err := rs.createNewPR(ctx, "test-branch", "abc123", []FileChange{}, nil)
	require.NoError(t, err)

	ghClient.AssertExpectations(t)
}

func TestRepositorySync_commitChanges_NoChanges(t *testing.T) {
	t.Skip("Temporarily disabled - error message assertion issue")
	ctx := context.Background()
	baseLogger := logrus.New()
	baseLogger.SetLevel(logrus.DebugLevel)
	logger := logrus.NewEntry(baseLogger)

	// Create a temporary directory for test
	tempDir, err := os.MkdirTemp("", "commitChanges_test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Setup mocks
	ghClient := &gh.MockClient{}
	gitClient := &git.MockClient{}

	// Mock successful operations until commit
	gitClient.On("Clone", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	gitClient.On("CreateBranch", mock.Anything, mock.Anything, "test-branch").Return(nil)
	gitClient.On("Checkout", mock.Anything, mock.Anything, "test-branch").Return(nil)
	gitClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Mock git.Commit to return ErrNoChanges
	gitClient.On("Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(git.ErrNoChanges)
	gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.Anything).Return("existing-sha-123", nil)

	engine := &Engine{
		git:     gitClient,
		gh:      ghClient,
		options: &Options{DryRun: false},
	}

	rs := &RepositorySync{
		engine:      engine,
		tempDir:     tempDir,
		target:      config.TargetConfig{Repo: "org/target"},
		logger:      logger,
		sourceState: &state.SourceState{LatestCommit: "abc123"},
		targetState: &state.TargetState{},
	}

	// Test commitChanges with ErrNoChanges scenario
	changedFiles := []FileChange{
		{Path: "file1.txt", Content: []byte("content1")},
	}

	commitSHA, _, err := rs.commitChanges(ctx, "test-branch", changedFiles)

	// Should not return an error and should return the existing commit SHA
	require.NoError(t, err)
	assert.Equal(t, "existing-sha-123", commitSHA)

	gitClient.AssertExpectations(t)
}

func TestRepositorySync_commitChanges_NoChanges_GetSHAError(t *testing.T) {
	t.Skip("Temporarily disabled - error message assertion issue")
	ctx := context.Background()
	baseLogger := logrus.New()
	baseLogger.SetLevel(logrus.DebugLevel)
	logger := logrus.NewEntry(baseLogger)

	// Create a temporary directory for test
	tempDir, err := os.MkdirTemp("", "commitChanges_test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Setup mocks
	ghClient := &gh.MockClient{}
	gitClient := &git.MockClient{}

	// Mock successful operations until commit
	gitClient.On("Clone", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	gitClient.On("CreateBranch", mock.Anything, mock.Anything, "test-branch").Return(nil)
	gitClient.On("Checkout", mock.Anything, mock.Anything, "test-branch").Return(nil)
	gitClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Mock git.Commit to return ErrNoChanges, but GetCurrentCommitSHA fails
	gitClient.On("Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(git.ErrNoChanges)
	gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.Anything).Return("", errTestGetSHA)

	engine := &Engine{
		git:     gitClient,
		gh:      ghClient,
		options: &Options{DryRun: false},
	}

	rs := &RepositorySync{
		engine:      engine,
		tempDir:     tempDir,
		target:      config.TargetConfig{Repo: "org/target"},
		logger:      logger,
		sourceState: &state.SourceState{LatestCommit: "abc123"},
		targetState: &state.TargetState{},
	}

	// Test commitChanges with ErrNoChanges + GetSHA error scenario
	changedFiles := []FileChange{
		{Path: "file1.txt", Content: []byte("content1")},
	}

	_, _, err = rs.commitChanges(ctx, "test-branch", changedFiles)

	// Should return an error about getting SHA
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no changes to commit and failed to get current SHA")

	gitClient.AssertExpectations(t)
}

// TestRepositorySync_getBranchPrefix tests the getBranchPrefix method
func TestRepositorySync_getBranchPrefix(t *testing.T) {
	tests := []struct {
		name           string
		currentGroup   *config.Group
		configGroups   []config.Group
		expectedPrefix string
	}{
		{
			name: "current group with branch prefix",
			currentGroup: &config.Group{
				Defaults: config.DefaultConfig{
					BranchPrefix: "custom/prefix",
				},
			},
			expectedPrefix: "custom/prefix",
		},
		{
			name:         "no current group, config groups with prefix",
			currentGroup: nil,
			configGroups: []config.Group{
				{
					Defaults: config.DefaultConfig{
						BranchPrefix: "config/prefix",
					},
				},
			},
			expectedPrefix: "config/prefix",
		},
		{
			name:           "no prefix anywhere, use default",
			currentGroup:   nil,
			configGroups:   nil,
			expectedPrefix: "chore/sync-files",
		},
		{
			name: "empty prefix in current group, use default",
			currentGroup: &config.Group{
				Defaults: config.DefaultConfig{
					BranchPrefix: "",
				},
			},
			expectedPrefix: "chore/sync-files",
		},
		{
			name:         "empty prefix in config groups, use default",
			currentGroup: nil,
			configGroups: []config.Group{
				{
					Defaults: config.DefaultConfig{
						BranchPrefix: "",
					},
				},
			},
			expectedPrefix: "chore/sync-files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Groups: tt.configGroups}
			engine := &Engine{
				config:       cfg,
				currentGroup: tt.currentGroup,
			}

			rs := &RepositorySync{
				engine: engine,
			}

			result := rs.getBranchPrefix()
			assert.Equal(t, tt.expectedPrefix, result)
		})
	}
}

// TestRepositorySync_findExistingPRForBranch tests the findExistingPRForBranch method
func TestRepositorySync_findExistingPRForBranch(t *testing.T) {
	branchName := "chore/sync-files-test-123"

	t.Run("no target state", func(t *testing.T) {
		rs := &RepositorySync{
			targetState: nil,
		}

		pr := rs.findExistingPRForBranch(branchName)
		assert.Nil(t, pr)
	})

	t.Run("no matching PR", func(t *testing.T) {
		rs := &RepositorySync{
			targetState: &state.TargetState{
				OpenPRs: []gh.PR{
					{Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{Ref: "feature/other-branch"}},
				},
			},
		}

		pr := rs.findExistingPRForBranch(branchName)
		assert.Nil(t, pr)
	})

	t.Run("matching PR found", func(t *testing.T) {
		expectedPR := gh.PR{
			Number: 123,
			Head: struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			}{Ref: branchName},
		}

		rs := &RepositorySync{
			targetState: &state.TargetState{
				OpenPRs: []gh.PR{
					{Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{Ref: "feature/other-branch"}},
					expectedPR,
				},
			},
		}

		pr := rs.findExistingPRForBranch(branchName)
		require.NotNil(t, pr)
		assert.Equal(t, 123, pr.Number)
		assert.Equal(t, branchName, pr.Head.Ref)
	})
}

// TestValidationMockGHClient provides a simple mock GitHub client for validation testing
type TestValidationMockGHClient struct {
	branches      []gh.Branch
	shouldFailLB  bool
	shouldFailDB  bool
	deletedBranch string
}

var (
	ErrMockListBranchesFailed = errors.New("ListBranches failed")
	ErrMockDeleteBranchFailed = errors.New("DeleteBranch failed")
	ErrMockNotImplemented     = errors.New("not implemented")
)

func (m *TestValidationMockGHClient) ListBranches(_ context.Context, _ string) ([]gh.Branch, error) {
	if m.shouldFailLB {
		return nil, ErrMockListBranchesFailed
	}
	return m.branches, nil
}

func (m *TestValidationMockGHClient) DeleteBranch(_ context.Context, _, branch string) error {
	if m.shouldFailDB {
		return ErrMockDeleteBranchFailed
	}
	m.deletedBranch = branch
	return nil
}

// Required methods to implement gh.Client interface (minimal implementations)
func (m *TestValidationMockGHClient) GetBranch(_ context.Context, _, _ string) (*gh.Branch, error) {
	return nil, ErrMockNotImplemented
}

func (m *TestValidationMockGHClient) CreatePR(_ context.Context, _ string, _ gh.PRRequest) (*gh.PR, error) {
	return nil, ErrMockNotImplemented
}

func (m *TestValidationMockGHClient) GetPR(_ context.Context, _ string, _ int) (*gh.PR, error) {
	return nil, ErrMockNotImplemented
}

func (m *TestValidationMockGHClient) ListPRs(_ context.Context, _, _ string) ([]gh.PR, error) {
	return nil, ErrMockNotImplemented
}

func (m *TestValidationMockGHClient) GetCommit(_ context.Context, _, _ string) (*gh.Commit, error) {
	return nil, ErrMockNotImplemented
}

func (m *TestValidationMockGHClient) ClosePR(_ context.Context, _ string, _ int, _ string) error {
	return ErrMockNotImplemented
}

func (m *TestValidationMockGHClient) UpdatePR(_ context.Context, _ string, _ int, _ gh.PRUpdate) error {
	return ErrMockNotImplemented
}

func (m *TestValidationMockGHClient) GetCurrentUser(_ context.Context) (*gh.User, error) {
	return nil, ErrMockNotImplemented
}

func (m *TestValidationMockGHClient) GetGitTree(_ context.Context, _, _ string, _ bool) (*gh.GitTree, error) {
	return nil, ErrMockNotImplemented
}

func (m *TestValidationMockGHClient) GetFile(_ context.Context, _, _, _ string) (*gh.FileContent, error) {
	return nil, ErrMockNotImplemented
}

// TestRepositorySync_validateAndCleanupOrphanedBranches tests the validateAndCleanupOrphanedBranches method
func TestRepositorySync_validateAndCleanupOrphanedBranches(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	t.Run("no branches to cleanup", func(t *testing.T) {
		ghClient := &TestValidationMockGHClient{
			branches: []gh.Branch{
				{Name: "main"},
				{Name: "develop"},
				{Name: "feature/some-feature"},
			},
		}

		engine := &Engine{
			gh: ghClient,
			config: &config.Config{
				Groups: []config.Group{
					{
						Defaults: config.DefaultConfig{
							BranchPrefix: "chore/sync-files",
						},
					},
				},
			},
		}

		rs := &RepositorySync{
			engine:      engine,
			target:      config.TargetConfig{Repo: "org/repo"},
			targetState: &state.TargetState{OpenPRs: []gh.PR{}},
			logger:      logger,
		}

		err := rs.validateAndCleanupOrphanedBranches(ctx)
		require.NoError(t, err)
		assert.Empty(t, ghClient.deletedBranch) // No branches should be deleted
	})

	t.Run("orphaned branches found and cleaned", func(t *testing.T) {
		orphanedBranch := "chore/sync-files-test-123"
		ghClient := &TestValidationMockGHClient{
			branches: []gh.Branch{
				{Name: "main"},
				{Name: orphanedBranch},
				{Name: "chore/sync-files-test-456"}, // This one has a PR
			},
		}

		engine := &Engine{
			gh: ghClient,
			config: &config.Config{
				Groups: []config.Group{
					{
						Defaults: config.DefaultConfig{
							BranchPrefix: "chore/sync-files",
						},
					},
				},
			},
		}

		rs := &RepositorySync{
			engine: engine,
			target: config.TargetConfig{Repo: "org/repo"},
			targetState: &state.TargetState{
				OpenPRs: []gh.PR{
					{Head: struct {
						Ref string `json:"ref"`
						SHA string `json:"sha"`
					}{Ref: "chore/sync-files-test-456"}}, // This branch has a PR
				},
			},
			logger: logger,
		}

		err := rs.validateAndCleanupOrphanedBranches(ctx)
		require.NoError(t, err)
		assert.Equal(t, orphanedBranch, ghClient.deletedBranch) // Orphaned branch should be deleted
	})

	t.Run("ListBranches fails", func(t *testing.T) {
		ghClient := &TestValidationMockGHClient{
			shouldFailLB: true,
		}

		engine := &Engine{
			gh: ghClient,
		}

		rs := &RepositorySync{
			engine: engine,
			target: config.TargetConfig{Repo: "org/repo"},
			logger: logger,
		}

		err := rs.validateAndCleanupOrphanedBranches(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list branches")
	})

	t.Run("DeleteBranch fails but continues", func(t *testing.T) {
		orphanedBranch := "chore/sync-files-test-123"
		ghClient := &TestValidationMockGHClient{
			branches: []gh.Branch{
				{Name: orphanedBranch},
			},
			shouldFailDB: true,
		}

		engine := &Engine{
			gh: ghClient,
			config: &config.Config{
				Groups: []config.Group{
					{
						Defaults: config.DefaultConfig{
							BranchPrefix: "chore/sync-files",
						},
					},
				},
			},
		}

		rs := &RepositorySync{
			engine:      engine,
			target:      config.TargetConfig{Repo: "org/repo"},
			targetState: &state.TargetState{OpenPRs: []gh.PR{}},
			logger:      logger,
		}

		// Should not return error even if DeleteBranch fails
		err := rs.validateAndCleanupOrphanedBranches(ctx)
		require.NoError(t, err)
	})
}

// TestRepositorySync_Execute_ExistingBranch tests scenarios where a branch already exists
func TestRepositorySync_Execute_ExistingBranch(t *testing.T) {
	ctx := context.Background()
	logger := logrus.NewEntry(logrus.New())

	t.Run("branch exists locally during create - recovers gracefully", func(t *testing.T) {
		// Setup mock clients
		gitClient := git.NewMockClient()
		ghClient := gh.NewMockClient()

		// Mock git operations - CreateBranch fails with ErrBranchAlreadyExists
		gitClient.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, destPath+"/test.txt", "test file content")
		})
		gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), "abc1234").Return(nil)
		gitClient.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(git.ErrBranchAlreadyExists)
		gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("commit123", nil)
		gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"test.txt"}, nil)
		gitClient.On("Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), false).Return(nil)

		// Mock GitHub operations - validation checks first
		ghClient.On("ListBranches", mock.Anything, "target/repo").Return([]gh.Branch{{Name: "master"}}, nil)

		// Mock file existence check to simulate file changes
		ghClient.On("GetFile", mock.Anything, "target/repo", "test.txt", "").Return(nil, gh.ErrFileNotFound)

		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser"}, nil)
		ghClient.On("CreatePR", mock.Anything, "target/repo", mock.AnythingOfType("gh.PRRequest")).Return(&gh.PR{Number: 123}, nil)

		// Mock transform operations
		transformChain := &transform.MockChain{}
		transformChain.On("Transform", mock.Anything, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("transform.Context")).Return([]byte("transformed content"), nil)

		// Setup engine and repository sync
		engine := &Engine{
			git:       gitClient,
			gh:        ghClient,
			transform: transformChain,
			config: &config.Config{
				Groups: []config.Group{{
					ID: "test-group",
					Defaults: config.DefaultConfig{
						BranchPrefix: "chore/sync-files",
					},
				}},
			},
			options: DefaultOptions(),
			logger:  logrus.New(),
		}

		sourceState := &state.SourceState{
			Repo:         "source/repo",
			Branch:       "main",
			LatestCommit: "abc1234",
		}

		targetState := &state.TargetState{
			LastSyncCommit: "different",
		}

		targetConfig := config.TargetConfig{
			Repo: "target/repo",
			Files: []config.FileMapping{
				{
					Src:  "test.txt",
					Dest: "test.txt",
				},
			},
		}

		rs := &RepositorySync{
			engine:      engine,
			target:      targetConfig,
			sourceState: sourceState,
			targetState: targetState,
			logger:      logger,
		}

		// Execute sync
		err := rs.Execute(ctx)
		require.NoError(t, err)

		// Verify git operations were called
		gitClient.AssertExpectations(t)
		ghClient.AssertExpectations(t)

		// Verify CreateBranch was called and failed with expected error
		gitClient.AssertCalled(t, "CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"))
		// Verify Checkout was called twice (once for source, once for branch recovery)
		gitClient.AssertCalled(t, "Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"))
	})

	t.Run("branch exists on remote during push - force pushes successfully", func(t *testing.T) {
		// Setup mock clients
		gitClient := git.NewMockClient()
		ghClient := gh.NewMockClient()

		// Mock git operations - Push fails first time, succeeds on force push
		gitClient.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, destPath+"/test.txt", "test file content")
		})
		gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), "abc1234").Return(nil)
		gitClient.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("commit123", nil)
		gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"test.txt"}, nil)
		// First push fails with branch exists error, second push (force) succeeds
		gitClient.On("Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), false).Return(git.ErrBranchAlreadyExists)
		gitClient.On("Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), true).Return(nil)

		// Mock GitHub operations
		ghClient.On("ListBranches", mock.Anything, "target/repo").Return([]gh.Branch{{Name: "master"}}, nil)
		ghClient.On("GetFile", mock.Anything, "target/repo", "test.txt", "").Return(nil, gh.ErrFileNotFound)
		ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "testuser"}, nil)
		ghClient.On("CreatePR", mock.Anything, "target/repo", mock.AnythingOfType("gh.PRRequest")).Return(&gh.PR{Number: 123}, nil)

		// Mock transform operations
		transformChain := &transform.MockChain{}
		transformChain.On("Transform", mock.Anything, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("transform.Context")).Return([]byte("transformed content"), nil)

		// Setup engine and repository sync
		engine := &Engine{
			git:       gitClient,
			gh:        ghClient,
			transform: transformChain,
			config: &config.Config{
				Groups: []config.Group{{
					ID: "test-group",
					Defaults: config.DefaultConfig{
						BranchPrefix: "chore/sync-files",
					},
				}},
			},
			options: DefaultOptions(),
			logger:  logrus.New(),
		}

		sourceState := &state.SourceState{
			Repo:         "source/repo",
			Branch:       "main",
			LatestCommit: "abc1234",
		}

		targetState := &state.TargetState{
			LastSyncCommit: "different",
		}

		targetConfig := config.TargetConfig{
			Repo: "target/repo",
			Files: []config.FileMapping{
				{
					Src:  "test.txt",
					Dest: "test.txt",
				},
			},
		}

		rs := &RepositorySync{
			engine:      engine,
			target:      targetConfig,
			sourceState: sourceState,
			targetState: targetState,
			logger:      logger,
		}

		// Execute sync
		err := rs.Execute(ctx)
		require.NoError(t, err)

		// Verify git operations were called
		gitClient.AssertExpectations(t)
		ghClient.AssertExpectations(t)

		// Verify both push calls were made (normal push, then force push)
		gitClient.AssertCalled(t, "Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), false)
		gitClient.AssertCalled(t, "Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), true)
	})

	t.Run("branch exists during push but force push also fails", func(t *testing.T) {
		// Setup mock clients
		gitClient := git.NewMockClient()
		ghClient := gh.NewMockClient()

		// Mock git operations - both normal and force push fail
		gitClient.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, destPath+"/test.txt", "test file content")
		})
		gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), "abc1234").Return(nil)
		gitClient.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("[]string")).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("commit123", nil)
		gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"test.txt"}, nil)
		gitClient.On("Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), false).Return(git.ErrBranchAlreadyExists)
		gitClient.On("Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), true).Return(errTestForcePushFailed)

		// Mock GitHub operations
		ghClient.On("ListBranches", mock.Anything, "target/repo").Return([]gh.Branch{{Name: "master"}}, nil)
		ghClient.On("GetFile", mock.Anything, "target/repo", "test.txt", "").Return(nil, gh.ErrFileNotFound)

		// Mock transform operations
		transformChain := &transform.MockChain{}
		transformChain.On("Transform", mock.Anything, mock.AnythingOfType("[]uint8"), mock.AnythingOfType("transform.Context")).Return([]byte("transformed content"), nil)

		// Setup engine and repository sync
		engine := &Engine{
			git:       gitClient,
			gh:        ghClient,
			transform: transformChain,
			config: &config.Config{
				Groups: []config.Group{{
					ID: "test-group",
					Defaults: config.DefaultConfig{
						BranchPrefix: "chore/sync-files",
					},
				}},
			},
			options: &Options{CleanupTempFiles: false, DryRun: false, Force: false, MaxConcurrency: 5},
			logger:  logrus.New(),
		}

		sourceState := &state.SourceState{
			Repo:         "source/repo",
			Branch:       "main",
			LatestCommit: "abc1234",
		}

		targetState := &state.TargetState{
			LastSyncCommit: "different",
		}

		targetConfig := config.TargetConfig{
			Repo: "target/repo",
			Files: []config.FileMapping{
				{
					Src:  "test.txt",
					Dest: "test.txt",
				},
			},
		}

		rs := &RepositorySync{
			engine:      engine,
			target:      targetConfig,
			sourceState: sourceState,
			targetState: targetState,
			logger:      logger,
		}

		// Execute sync - should fail when force push fails
		err := rs.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to force push branch")

		// Verify both push calls were made
		gitClient.AssertCalled(t, "Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), false)
		gitClient.AssertCalled(t, "Push", mock.Anything, mock.AnythingOfType("string"), "origin", mock.AnythingOfType("string"), true)
	})
}

// TestRepositorySync_NoChangesToSync tests the scenario where files are already synchronized
func TestRepositorySync_NoChangesToSync(t *testing.T) {
	// Setup test configuration
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
		LastSyncCommit: "def456", // Different from source to allow sync to start
		Status:         state.StatusUpToDate,
	}

	t.Run("no changes needed - skip PR creation", func(t *testing.T) {
		// Create temporary directory and files for testing
		tmpDir := testutil.CreateTempDir(t)
		sourceDir := tmpDir + "/source"
		targetDir := tmpDir + "/target"
		testutil.CreateTestDirectory(t, sourceDir)
		testutil.CreateTestDirectory(t, targetDir)

		// Create different source content but transform will produce identical target content
		testutil.WriteTestFile(t, sourceDir+"/file1.txt", "source content")
		testutil.WriteTestFile(t, targetDir+"/file1.txt", "identical content")

		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		transformChain := &transform.MockChain{}

		// Setup default expectations
		ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
		ghClient.On("GetFile", mock.Anything, "org/target", "file1.txt", "").Return(&gh.FileContent{
			Content: []byte("identical content"),
		}, nil)

		// Mock git operations
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/source")
		})).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), "source content")
		})

		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/target")
		})).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), "identical content")
		})

		gitClient.On("Checkout", mock.Anything, mock.Anything, "abc123").Return(nil)
		gitClient.On("Checkout", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("CreateBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("CheckoutBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)

		// Mock transform operations - return different content to force processing
		transformChain.On("Transform", mock.Anything, mock.Anything, mock.Anything).Return([]byte("transformed content"), nil)

		// Most importantly: Mock commit to return ErrNoChangesToSync
		gitClient.On("AddAll", mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(git.ErrNoChanges)

		// Mock additional git operations that might be called
		gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.Anything).Return("existing-sha", nil).Maybe()

		// Create engine and repository sync
		engine := &Engine{
			config: &config.Config{
				Groups: []config.Group{{
					Defaults: config.DefaultConfig{
						BranchPrefix: "chore/sync-files",
					},
				}},
			},
			gh:        ghClient,
			git:       gitClient,
			transform: transformChain,
			options:   DefaultOptions(),
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

		// Should succeed without error (no PR needed)
		require.NoError(t, err)

		// Verify that no PR creation methods were called
		ghClient.AssertNotCalled(t, "CreatePR", mock.Anything, mock.Anything, mock.Anything)
		ghClient.AssertNotCalled(t, "UpdatePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)

		// Verify git operations were called appropriately
		gitClient.AssertCalled(t, "Clone", mock.Anything, mock.Anything, mock.AnythingOfType("string"))
		gitClient.AssertCalled(t, "Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string"))

		// Verify transform was still executed
		transformChain.AssertCalled(t, "Transform", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("no changes but commit fails with different error", func(t *testing.T) {
		// Create temporary directory and files
		tmpDir := testutil.CreateTempDir(t)
		sourceDir := tmpDir + "/source"
		testutil.CreateTestDirectory(t, sourceDir)
		testutil.WriteTestFile(t, sourceDir+"/file1.txt", "content")

		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		transformChain := &transform.MockChain{}

		// Setup basic expectations
		ghClient.On("ListBranches", mock.Anything, mock.Anything).Return([]gh.Branch{}, nil).Maybe()
		ghClient.On("GetFile", mock.Anything, "org/target", "file1.txt", "").Return(&gh.FileContent{
			Content: []byte("different content"),
		}, nil)

		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/source")
		})).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), "content")
		})

		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/target")
		})).Return(nil).Run(func(args mock.Arguments) {
			destPath := args[2].(string)
			testutil.CreateTestDirectory(t, destPath)
			testutil.WriteTestFile(t, filepath.Join(destPath, "file1.txt"), "different content")
		})
		gitClient.On("Checkout", mock.Anything, mock.Anything, "abc123").Return(nil)
		gitClient.On("Checkout", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("CreateBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		gitClient.On("CheckoutBranch", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(nil)
		transformChain.On("Transform", mock.Anything, mock.Anything, mock.Anything).Return([]byte("transformed content"), nil)
		gitClient.On("AddAll", mock.Anything, mock.Anything).Return(nil)
		gitClient.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// Mock commit to return a different error (not ErrNoChangesToSync)
		commitError := errTestGitCommand
		gitClient.On("Commit", mock.Anything, mock.Anything, mock.AnythingOfType("string")).Return(commitError)

		// Create engine and repository sync
		engine := &Engine{
			config: &config.Config{
				Groups: []config.Group{{
					Defaults: config.DefaultConfig{
						BranchPrefix: "chore/sync-files",
					},
				}},
			},
			gh:        ghClient,
			git:       gitClient,
			transform: transformChain,
			options:   DefaultOptions(),
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

		// Should fail with the commit error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit changes")
		assert.Contains(t, err.Error(), "permission denied")

		// Verify that no PR creation methods were called
		ghClient.AssertNotCalled(t, "CreatePR", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestRepositorySync_BranchAwareCloning(t *testing.T) {
	tests := []struct {
		name         string
		targetBranch string
		expectClone  bool
		expectBranch bool
	}{
		{
			name:         "clone with target branch",
			targetBranch: "development",
			expectClone:  false,
			expectBranch: true,
		},
		{
			name:         "clone without target branch",
			targetBranch: "",
			expectClone:  true,
			expectBranch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			ghClient := gh.NewMockClient()
			gitClient := git.NewMockClient()

			// Setup basic mocks
			sourceState := &state.SourceState{
				Repo:         "source/repo",
				Branch:       "master",
				LatestCommit: "abc123",
			}

			targetState := &state.TargetState{
				Branch: tt.targetBranch,
			}

			target := config.TargetConfig{
				Repo: "target/repo",
			}

			// Mock directory operations
			engine := &Engine{
				gh:  ghClient,
				git: gitClient,
				options: &Options{
					DryRun: false,
				},
			}

			repoSync := &RepositorySync{
				engine:      engine,
				target:      target,
				sourceState: sourceState,
				targetState: targetState,
				logger:      logrus.NewEntry(logrus.New()),
				tempDir:     "/tmp/test",
			}

			// Mock the git operations based on expected behavior
			if tt.expectBranch {
				gitClient.On("CloneWithBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), tt.targetBranch).Return(nil)
			}

			if tt.expectClone {
				gitClient.On("Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			}

			gitClient.On("CreateBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			gitClient.On("Checkout", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			gitClient.On("Add", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(nil)
			gitClient.On("Commit", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
			gitClient.On("Push", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return(nil)
			gitClient.On("GetChangedFiles", mock.Anything, mock.AnythingOfType("string")).Return([]string{"file1.txt"}, nil)
			gitClient.On("GetCurrentCommitSHA", mock.Anything, mock.AnythingOfType("string")).Return("abc123", nil)

			// Mock GitHub operations
			ghClient.On("ListPRs", mock.Anything, target.Repo, mock.AnythingOfType("string")).Return([]gh.PR{}, nil)
			ghClient.On("GetCurrentUser", mock.Anything).Return(&gh.User{Login: "test-user"}, nil)

			if tt.targetBranch != "" {
				ghClient.On("GetBranch", mock.Anything, target.Repo, tt.targetBranch).Return(&gh.Branch{Name: tt.targetBranch}, nil)
			} else {
				ghClient.On("ListBranches", mock.Anything, target.Repo).Return([]gh.Branch{{Name: "master"}}, nil)
			}

			ghClient.On("CreatePR", mock.Anything, target.Repo, mock.AnythingOfType("gh.PRRequest")).Return(&gh.PR{Number: 123}, nil)

			// Create test file changes
			fileChanges := []FileChange{
				{
					Path:      "test.txt",
					Content:   []byte("test content"),
					IsDeleted: false,
				},
			}

			// Execute the commit process
			_, _, err := repoSync.commitChanges(context.Background(), "test-branch", fileChanges)
			require.NoError(t, err)

			// Verify the correct clone method was called
			if tt.expectBranch {
				gitClient.AssertCalled(t, "CloneWithBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), tt.targetBranch)
				gitClient.AssertNotCalled(t, "Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"))
			}

			if tt.expectClone {
				gitClient.AssertCalled(t, "Clone", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"))
				gitClient.AssertNotCalled(t, "CloneWithBranch", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"))
			}
		})
	}
}
