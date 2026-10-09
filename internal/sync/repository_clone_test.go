package sync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/git"
)

var (
	errTestRepoNotFound    = errors.New("repository not found")
	errTestFullCloneFailed = errors.New("full clone failed")
)

const testCloneTargetURL = "https://github.com/org/target.git"

// newCloneTestSync builds a RepositorySync wired to a mock git client for
// exercising the target clone logic in isolation.
func newCloneTestSync(t *testing.T, target config.TargetConfig, moduleUpdates ...ModuleUpdateInfo) (*RepositorySync, *git.MockClient, string) {
	t.Helper()

	gitClient := git.NewMockClient()
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	rs := &RepositorySync{
		engine:        &Engine{git: gitClient},
		target:        target,
		logger:        logger.WithField("test", t.Name()),
		moduleUpdates: moduleUpdates,
	}
	return rs, gitClient, filepath.Join(t.TempDir(), "target")
}

func sparseOpts(paths ...string) any {
	return mock.MatchedBy(func(opts *git.CloneOptions) bool {
		return opts != nil && opts.BlobSizeLimit == "" && assert.ObjectsAreEqual(paths, opts.SparsePaths)
	})
}

func fullCloneOpts() *git.CloneOptions {
	return &git.CloneOptions{BlobSizeLimit: "0"}
}

func TestRepositorySync_TargetSparsePaths(t *testing.T) {
	rs, _, _ := newCloneTestSync(t, config.TargetConfig{Repo: "org/target"},
		ModuleUpdateInfo{DestPath: filepath.Join("libs", "shared", "go.mod"), ModuleName: "example.com/shared", Version: "v1.2.3"},
	)

	paths := rs.targetSparsePaths([]FileChange{
		{Path: ".github/workflows/ci.yml"},
		{Path: "docs/new-file.md", IsNew: true},
		{Path: "old/removed.txt", IsDeleted: true},
	})

	assert.Equal(t, []string{
		".github/workflows/ci.yml",
		"docs/new-file.md",
		"old/removed.txt",
		filepath.Join("libs", "shared", "go.mod"),
	}, paths, "every path commitChanges touches, including deletions and go.mod updates, must be checked out")
}

func TestRepositorySync_CloneTarget(t *testing.T) {
	ctx := context.Background()
	changes := []FileChange{{Path: "README.md"}, {Path: "gone.txt", IsDeleted: true}}

	t.Run("default mode clones the target branch sparsely", func(t *testing.T) {
		rs, gitClient, targetPath := newCloneTestSync(t, config.TargetConfig{Repo: "org/target", Branch: "development"},
			ModuleUpdateInfo{DestPath: "go.mod"},
		)
		gitClient.On("CloneWithBranch", ctx, testCloneTargetURL, targetPath, "development",
			sparseOpts("README.md", "gone.txt", "go.mod")).Return(nil).Once()

		require.NoError(t, rs.cloneTarget(ctx, testCloneTargetURL, targetPath, changes))
		gitClient.AssertExpectations(t)
	})

	t.Run("explicit sparse mode without a branch uses the default branch", func(t *testing.T) {
		rs, gitClient, targetPath := newCloneTestSync(t, config.TargetConfig{Repo: "org/target", CloneMode: config.CloneModeSparse})
		gitClient.On("Clone", ctx, testCloneTargetURL, targetPath, sparseOpts("README.md", "gone.txt")).Return(nil).Once()

		require.NoError(t, rs.cloneTarget(ctx, testCloneTargetURL, targetPath, changes))
		gitClient.AssertExpectations(t)
	})

	t.Run("full mode clones everything", func(t *testing.T) {
		rs, gitClient, targetPath := newCloneTestSync(t, config.TargetConfig{Repo: "org/target", Branch: "main", CloneMode: config.CloneModeFull})
		gitClient.On("CloneWithBranch", ctx, testCloneTargetURL, targetPath, "main", fullCloneOpts()).Return(nil).Once()

		require.NoError(t, rs.cloneTarget(ctx, testCloneTargetURL, targetPath, changes))
		gitClient.AssertExpectations(t)
	})

	t.Run("full mode without a branch uses the default branch", func(t *testing.T) {
		rs, gitClient, targetPath := newCloneTestSync(t, config.TargetConfig{Repo: "org/target", CloneMode: config.CloneModeFull})
		gitClient.On("Clone", ctx, testCloneTargetURL, targetPath, fullCloneOpts()).Return(nil).Once()

		require.NoError(t, rs.cloneTarget(ctx, testCloneTargetURL, targetPath, changes))
		gitClient.AssertExpectations(t)
	})

	t.Run("falls back to a full clone when sparse checkout setup fails", func(t *testing.T) {
		rs, gitClient, targetPath := newCloneTestSync(t, config.TargetConfig{Repo: "org/target", Branch: "main"})

		gitClient.On("CloneWithBranch", ctx, testCloneTargetURL, targetPath, "main", sparseOpts("README.md", "gone.txt")).
			Run(func(mock.Arguments) {
				// Simulate a leftover directory from the failed attempt
				require.NoError(t, os.MkdirAll(targetPath, 0o750))
				require.NoError(t, os.WriteFile(filepath.Join(targetPath, "partial"), []byte("x"), 0o600))
			}).
			Return(fmt.Errorf("clone: %w: read-tree failed", git.ErrSparseCheckout)).Once()
		gitClient.On("CloneWithBranch", ctx, testCloneTargetURL, targetPath, "main", fullCloneOpts()).
			Run(func(mock.Arguments) {
				assert.NoDirExists(t, targetPath, "the failed sparse clone must be removed before the full clone")
			}).
			Return(nil).Once()

		require.NoError(t, rs.cloneTarget(ctx, testCloneTargetURL, targetPath, changes))
		gitClient.AssertExpectations(t)
	})

	t.Run("returns the full clone error when the fallback also fails", func(t *testing.T) {
		rs, gitClient, targetPath := newCloneTestSync(t, config.TargetConfig{Repo: "org/target"})
		gitClient.On("Clone", ctx, testCloneTargetURL, targetPath, sparseOpts("README.md", "gone.txt")).
			Return(git.ErrSparseCheckout).Once()
		gitClient.On("Clone", ctx, testCloneTargetURL, targetPath, fullCloneOpts()).
			Return(errTestFullCloneFailed).Once()

		err := rs.cloneTarget(ctx, testCloneTargetURL, targetPath, changes)
		require.ErrorIs(t, err, errTestFullCloneFailed)
		assert.Contains(t, err.Error(), "failed to clone target repository")
		gitClient.AssertExpectations(t)
	})

	t.Run("does not fall back on other clone errors", func(t *testing.T) {
		rs, gitClient, targetPath := newCloneTestSync(t, config.TargetConfig{Repo: "org/target", Branch: "main"})
		gitClient.On("CloneWithBranch", ctx, testCloneTargetURL, targetPath, "main", mock.Anything).
			Return(errTestRepoNotFound).Once()

		err := rs.cloneTarget(ctx, testCloneTargetURL, targetPath, changes)
		require.ErrorIs(t, err, errTestRepoNotFound)
		assert.Contains(t, err.Error(), "failed to clone target repository with branch main")
		gitClient.AssertNumberOfCalls(t, "CloneWithBranch", 1)
	})
}
