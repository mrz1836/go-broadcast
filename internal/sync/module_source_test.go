package sync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/git"
)

func TestModuleSourceResolver_GetSourceAtVersion(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	t.Run("successfully clones at version", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "module-source-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		mockGit := git.NewMockClient()
		resolver := NewModuleSourceResolver(mockGit, logger, nil)

		// Set up mock to create a directory when CloneAtTag is called
		mockGit.On("CloneAtTag", mock.Anything, "https://github.com/owner/repo", mock.MatchedBy(func(path string) bool {
			// Create the directory to simulate clone
			if mkdirErr := os.MkdirAll(path, 0o750); mkdirErr != nil {
				return false
			}
			// Create a test file to verify
			testFile := filepath.Join(path, "main.go")
			return os.WriteFile(testFile, []byte("package main"), 0o600) == nil
		}), "v1.0.0", (*git.CloneOptions)(nil)).Return(nil)

		source, err := resolver.GetSourceAtVersion(
			context.Background(),
			"https://github.com/owner/repo",
			"v1.0.0",
			"",
			tempDir,
		)

		require.NoError(t, err)
		require.NotNil(t, source)
		assert.Equal(t, "v1.0.0", source.ResolvedVersion)
		assert.DirExists(t, source.Path)
		assert.NotNil(t, source.CleanupFunc)

		// Verify the test file exists
		assert.FileExists(t, filepath.Join(source.Path, "main.go"))

		// Cleanup
		source.CleanupFunc()
		assert.NoDirExists(t, source.RepoPath)

		mockGit.AssertExpectations(t)
	})

	t.Run("handles subdirectory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "module-source-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		mockGit := git.NewMockClient()
		resolver := NewModuleSourceResolver(mockGit, logger, nil)

		// Set up mock to create a directory with subdirectory
		mockGit.On("CloneAtTag", mock.Anything, "https://github.com/owner/repo", mock.MatchedBy(func(path string) bool {
			// Create directory with subdirectory
			subdir := filepath.Join(path, "pkg", "lib")
			if mkdirErr := os.MkdirAll(subdir, 0o750); mkdirErr != nil {
				return false
			}
			return os.WriteFile(filepath.Join(subdir, "lib.go"), []byte("package lib"), 0o600) == nil
		}), "v2.0.0", (*git.CloneOptions)(nil)).Return(nil)

		source, err := resolver.GetSourceAtVersion(
			context.Background(),
			"https://github.com/owner/repo",
			"v2.0.0",
			"pkg/lib",
			tempDir,
		)

		require.NoError(t, err)
		require.NotNil(t, source)
		assert.Contains(t, source.Path, "pkg/lib")
		assert.FileExists(t, filepath.Join(source.Path, "lib.go"))

		// Cleanup
		source.CleanupFunc()

		mockGit.AssertExpectations(t)
	})

	t.Run("returns error for missing subdirectory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "module-source-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		mockGit := git.NewMockClient()
		resolver := NewModuleSourceResolver(mockGit, logger, nil)

		// Set up mock to create a directory without the expected subdirectory
		mockGit.On("CloneAtTag", mock.Anything, "https://github.com/owner/repo", mock.MatchedBy(func(path string) bool {
			return os.MkdirAll(path, 0o750) == nil
		}), "v1.0.0", (*git.CloneOptions)(nil)).Return(nil)

		source, err := resolver.GetSourceAtVersion(
			context.Background(),
			"https://github.com/owner/repo",
			"v1.0.0",
			"nonexistent/subdir",
			tempDir,
		)

		require.Error(t, err)
		assert.Nil(t, source)
		require.ErrorIs(t, err, ErrSubdirMissing)
		assert.Contains(t, err.Error(), "nonexistent/subdir")

		mockGit.AssertExpectations(t)
	})

	t.Run("returns error for empty repo URL", func(t *testing.T) {
		mockGit := git.NewMockClient()
		resolver := NewModuleSourceResolver(mockGit, logger, nil)

		source, err := resolver.GetSourceAtVersion(
			context.Background(),
			"",
			"v1.0.0",
			"",
			"/tmp",
		)

		require.Error(t, err)
		assert.Nil(t, source)
		assert.ErrorIs(t, err, ErrEmptyRepoURL)
	})

	t.Run("returns error for empty version", func(t *testing.T) {
		mockGit := git.NewMockClient()
		resolver := NewModuleSourceResolver(mockGit, logger, nil)

		source, err := resolver.GetSourceAtVersion(
			context.Background(),
			"https://github.com/owner/repo",
			"",
			"",
			"/tmp",
		)

		require.Error(t, err)
		assert.Nil(t, source)
		assert.ErrorIs(t, err, ErrEmptyVersion)
	})

	t.Run("returns error when clone fails", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "module-source-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		mockGit := git.NewMockClient()
		resolver := NewModuleSourceResolver(mockGit, logger, nil)

		mockGit.On("CloneAtTag", mock.Anything, "https://github.com/owner/repo", mock.Anything, "v1.0.0", (*git.CloneOptions)(nil)).Return(git.ErrGitCommand)

		source, err := resolver.GetSourceAtVersion(
			context.Background(),
			"https://github.com/owner/repo",
			"v1.0.0",
			"",
			tempDir,
		)

		require.Error(t, err)
		assert.Nil(t, source)
		require.ErrorIs(t, err, ErrCloneFailed)

		mockGit.AssertExpectations(t)
	})

	t.Run("generates unique directory names", func(t *testing.T) {
		// Call sanitizeVersion and generateShortID multiple times
		v1 := sanitizeVersion("v1.0.0")
		v2 := sanitizeVersion("v1.0.0")
		assert.Equal(t, v1, v2, "sanitizeVersion should be deterministic")

		// generateShortID uses PID, so it's consistent within the same process
		id1 := generateShortID()
		id2 := generateShortID()
		assert.Equal(t, id1, id2, "generateShortID uses PID so should be consistent")
	})
}

func TestNewModuleSourceResolver(t *testing.T) {
	logger := logrus.New()
	mockGit := git.NewMockClient()
	cache := NewModuleCache(0, logger) // 0 uses default TTL

	resolver := NewModuleSourceResolver(mockGit, logger, cache)

	assert.NotNil(t, resolver)
	assert.Equal(t, mockGit, resolver.git)
	assert.Equal(t, logger, resolver.logger)
	assert.Equal(t, cache, resolver.cache)
}

func TestVersionedSource_CleanupFunc(t *testing.T) {
	logger := logrus.New()

	t.Run("cleanup removes directory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cleanup-test-*")
		require.NoError(t, err)

		// Create a nested structure to ensure cleanup removes all
		nestedDir := filepath.Join(tempDir, "subdir", "nested")
		require.NoError(t, os.MkdirAll(nestedDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "file.txt"), []byte("test"), 0o600))

		source := &VersionedSource{
			Path:            nestedDir,
			RepoPath:        tempDir,
			ResolvedVersion: "v1.0.0",
			CleanupFunc: func() {
				if err := os.RemoveAll(tempDir); err != nil {
					logger.WithError(err).Warn("Cleanup failed")
				}
			},
		}

		// Verify directory exists
		assert.DirExists(t, tempDir)

		// Call cleanup
		source.CleanupFunc()

		// Verify directory is removed
		assert.NoDirExists(t, tempDir)
	})

	t.Run("cleanup is safe to call multiple times", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cleanup-test-*")
		require.NoError(t, err)

		source := &VersionedSource{
			Path:            tempDir,
			RepoPath:        tempDir,
			ResolvedVersion: "v1.0.0",
			CleanupFunc: func() {
				_ = os.RemoveAll(tempDir)
			},
		}

		// Call cleanup multiple times - should not panic
		source.CleanupFunc()
		source.CleanupFunc()
		source.CleanupFunc()
	})
}
