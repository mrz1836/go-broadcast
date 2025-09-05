//go:build integration
// +build integration

package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestGitClient_FullWorkflow_Integration(t *testing.T) {
	client, err := NewClient(logrus.New(), &logging.LogConfig{})
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Clone a real repository
	t.Run("Clone", func(t *testing.T) {
		err := client.Clone(ctx, "https://github.com/octocat/Hello-World.git", repoPath)
		require.NoError(t, err)
		assert.DirExists(t, filepath.Join(repoPath, ".git"))
	})

	// Work with the cloned repository
	t.Run("GitOperations", func(t *testing.T) {
		// Get current branch
		branch, err := client.GetCurrentBranch(ctx, repoPath)
		require.NoError(t, err)
		assert.NotEmpty(t, branch)

		// Create a new branch
		err = client.CreateBranch(ctx, repoPath, "test-integration-branch")
		require.NoError(t, err)

		// Verify we're on the new branch
		currentBranch, err := client.GetCurrentBranch(ctx, repoPath)
		require.NoError(t, err)
		assert.Equal(t, "test-integration-branch", currentBranch)

		// Create a test file
		testFile := filepath.Join(repoPath, "integration-test.txt")
		err = os.WriteFile(testFile, []byte("Integration test content\n"), 0o644)
		require.NoError(t, err)

		// Add the file
		err = client.Add(ctx, repoPath, "integration-test.txt")
		require.NoError(t, err)

		// Check diff
		diff, err := client.Diff(ctx, repoPath, true)
		require.NoError(t, err)
		assert.Contains(t, diff, "Integration test content")

		// Commit changes
		err = client.Commit(ctx, repoPath, "Add integration test file")
		require.NoError(t, err)

		// Switch back to original branch
		err = client.Checkout(ctx, repoPath, branch)
		require.NoError(t, err)

		// Verify branch switch
		currentBranch, err = client.GetCurrentBranch(ctx, repoPath)
		require.NoError(t, err)
		assert.Equal(t, branch, currentBranch)
	})

	t.Run("RemoteOperations", func(t *testing.T) {
		// Get remote URL
		url, err := client.GetRemoteURL(ctx, repoPath, "origin")
		require.NoError(t, err)
		assert.Equal(t, "https://github.com/octocat/Hello-World.git", url)
	})
}
