package sync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
		tmpDir := t.TempDir()
		sourceDir := tmpDir + "/source"
		require.NoError(t, os.MkdirAll(sourceDir, 0o750))

		// Create source files
		require.NoError(t, os.WriteFile(sourceDir+"/file1.txt", []byte("content 1"), 0o600))
		require.NoError(t, os.WriteFile(sourceDir+"/file2.txt", []byte("content 2"), 0o600))

		// Setup mocks
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		transformChain := &transform.MockChain{}

		// Mock git operations to use our temp directory
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/source")
		})).Return(nil).Run(func(args mock.Arguments) {
			// Copy our pre-created files to the expected location
			destPath := args[2].(string)
			_ = os.MkdirAll(destPath, 0o750)                                           // Test setup
			srcContent1, _ := os.ReadFile(filepath.Join(sourceDir, "file1.txt"))       //nolint:gosec // Test file
			srcContent2, _ := os.ReadFile(filepath.Join(sourceDir, "file2.txt"))       //nolint:gosec // Test file
			_ = os.WriteFile(filepath.Join(destPath, "file1.txt"), srcContent1, 0o600) // Test setup
			_ = os.WriteFile(filepath.Join(destPath, "file2.txt"), srcContent2, 0o600) // Test setup
		})
		gitClient.On("Checkout", mock.Anything, mock.Anything, "abc123").Return(nil)

		// Mock file operations
		ghClient.On("GetFile", mock.Anything, "org/target", "file1.txt", "").
			Return(&gh.FileContent{Content: []byte("old content 1")}, nil)
		ghClient.On("GetFile", mock.Anything, "org/target", "file2.txt", "").
			Return(&gh.FileContent{Content: []byte("old content 2")}, nil)

		// Mock transformations (return different content to trigger changes)
		transformChain.On("Transform", mock.Anything, []byte("content 1"), mock.Anything).
			Return([]byte("new content 1"), nil)
		transformChain.On("Transform", mock.Anything, []byte("content 2"), mock.Anything).
			Return([]byte("new content 2"), nil)

		// Mock GitHub operations for PR creation
		ghClient.On("ListBranches", mock.Anything, "org/target").
			Return([]gh.Branch{{Name: "master"}}, nil)
		ghClient.On("CreatePR", mock.Anything, "org/target", mock.Anything).
			Return(&gh.PR{Number: 123}, nil)

		// Create engine with dry-run disabled
		opts := DefaultOptions().WithDryRun(false)
		engine := &Engine{
			config: &config.Config{
				Defaults: config.DefaultConfig{
					BranchPrefix: "sync/template",
				},
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

		// Mock git clone failure
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.Anything).
			Return(errors.ErrTest)

		engine := &Engine{
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
		tmpDir := t.TempDir()
		sourceDir := tmpDir + "/source"
		require.NoError(t, os.MkdirAll(sourceDir, 0o750))

		// Create source files
		require.NoError(t, os.WriteFile(sourceDir+"/file1.txt", []byte("content 1"), 0o600))
		require.NoError(t, os.WriteFile(sourceDir+"/file2.txt", []byte("content 2"), 0o600))

		// Setup mocks (minimal since dry-run shouldn't call most operations)
		ghClient := &gh.MockClient{}
		gitClient := &git.MockClient{}
		transformChain := &transform.MockChain{}

		// Only mock the operations that should happen in dry-run
		gitClient.On("Clone", mock.Anything, mock.Anything, mock.MatchedBy(func(path string) bool {
			return strings.HasSuffix(path, "/source")
		})).Return(nil).Run(func(args mock.Arguments) {
			// Copy our pre-created files to the expected location
			destPath := args[2].(string)
			_ = os.MkdirAll(destPath, 0o750)                                           // Test setup
			srcContent1, _ := os.ReadFile(filepath.Join(sourceDir, "file1.txt"))       //nolint:gosec // Test file
			srcContent2, _ := os.ReadFile(filepath.Join(sourceDir, "file2.txt"))       //nolint:gosec // Test file
			_ = os.WriteFile(filepath.Join(destPath, "file1.txt"), srcContent1, 0o600) // Test setup
			_ = os.WriteFile(filepath.Join(destPath, "file2.txt"), srcContent2, 0o600) // Test setup
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

		// Create engine with dry-run enabled
		opts := DefaultOptions().WithDryRun(true)
		engine := &Engine{
			config: &config.Config{
				Defaults: config.DefaultConfig{
					BranchPrefix: "sync/template",
				},
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
		assert.Equal(t, "sync: update README.md from template", msg)
	})

	t.Run("multiple files", func(t *testing.T) {
		files := []FileChange{
			{Path: "README.md"},
			{Path: "Makefile"},
			{Path: ".github/workflows/ci.yml"},
		}

		msg := repoSync.generateCommitMessage(files)
		assert.Equal(t, "sync: update 3 files from template", msg)
	})
}

func TestRepositorySync_generatePRTitle(t *testing.T) {
	repoSync := &RepositorySync{
		sourceState: &state.SourceState{
			LatestCommit: "abc123def456",
		},
	}

	title := repoSync.generatePRTitle()
	assert.Equal(t, "Sync files from template repository (abc123d)", title)
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
	}

	files := []FileChange{
		{Path: "README.md", IsNew: false},
		{Path: "new-file.txt", IsNew: true},
	}

	body := repoSync.generatePRBody("commit456", files)

	// Verify key components are present
	assert.Contains(t, body, "This pull request synchronizes files")
	assert.Contains(t, body, "go-broadcast-metadata")
	assert.Contains(t, body, "source_repo: org/template")
	assert.Contains(t, body, "source_commit: abc123")
	assert.Contains(t, body, "target_repo: org/target")
	assert.Contains(t, body, "sync_commit: commit456")
	assert.Contains(t, body, "## Changed Files")
	assert.Contains(t, body, "`README.md` (modified)")
	assert.Contains(t, body, "`new-file.txt` (added)")
	assert.Contains(t, body, "## Source Information")
}

func TestRepositorySync_findExistingPR(t *testing.T) {
	branchName := "sync/template-20240115-120530-abc123"

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
