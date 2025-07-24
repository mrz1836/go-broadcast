package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These are integration tests that require git to be installed
func TestGitClient_Clone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Clone a small public repository
	err = client.Clone(ctx, "https://github.com/octocat/Hello-World.git", repoPath)
	require.NoError(t, err)

	// Verify the repository was cloned
	assert.DirExists(t, filepath.Join(repoPath, ".git"))
	assert.FileExists(t, filepath.Join(repoPath, "README"))
}

func TestGitClient_Clone_AlreadyExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create directory first
	err = os.MkdirAll(repoPath, 0o700)
	require.NoError(t, err)

	// Try to clone into existing directory
	err = client.Clone(ctx, "https://github.com/octocat/Hello-World.git", repoPath)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRepositoryExists)
}

func TestGitClient_Operations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a new repository
	cmd := exec.CommandContext(ctx, "git", "init", repoPath) //nolint:gosec // Test uses hardcoded command
	err = cmd.Run()
	require.NoError(t, err)

	// Configure git user for tests
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "config", "user.email", "test@example.com") //nolint:gosec // Test uses hardcoded command
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "config", "user.name", "Test User") //nolint:gosec // Test uses hardcoded command
	err = cmd.Run()
	require.NoError(t, err)

	// Test GetCurrentBranch (should be on default branch)
	branch, err := client.GetCurrentBranch(ctx, repoPath)
	require.NoError(t, err)
	assert.NotEmpty(t, branch)

	// Create and checkout a new branch
	err = client.CreateBranch(ctx, repoPath, "test-branch")
	require.NoError(t, err)

	branch, err = client.GetCurrentBranch(ctx, repoPath)
	require.NoError(t, err)
	assert.Equal(t, "test-branch", branch)

	// Create a test file
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0o600)
	require.NoError(t, err)

	// Add the file
	err = client.Add(ctx, repoPath, "test.txt")
	require.NoError(t, err)

	// Get staged diff
	diff, err := client.Diff(ctx, repoPath, true)
	require.NoError(t, err)
	assert.Contains(t, diff, "+test content")

	// Commit the changes
	err = client.Commit(ctx, repoPath, "Test commit")
	require.NoError(t, err)

	// Try to commit with no changes
	err = client.Commit(ctx, repoPath, "Empty commit")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoChanges)

	// Checkout back to original branch
	originalBranch := "master"
	if branch == "main" {
		originalBranch = "main"
	}

	// Create initial commit on original branch first
	err = client.Checkout(ctx, repoPath, originalBranch)
	if err != nil {
		// If checkout fails, try creating the branch
		cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", "-b", originalBranch) //nolint:gosec // Test uses hardcoded command
		err = cmd.Run()
		require.NoError(t, err)
	}
}

func TestGitClient_GetRemoteURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Clone a repository with a remote
	err = client.Clone(ctx, "https://github.com/octocat/Hello-World.git", repoPath)
	require.NoError(t, err)

	// Get remote URL
	url, err := client.GetRemoteURL(ctx, repoPath, "origin")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/octocat/Hello-World.git", url)
}

func TestNewClient_GitNotFound(t *testing.T) {
	// Save original PATH
	oldPath := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	// Set PATH to empty to simulate git not being found
	_ = os.Setenv("PATH", "")

	client, err := NewClient(nil, nil)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.ErrorIs(t, err, ErrGitNotFound)
}
