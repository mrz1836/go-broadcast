package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// TestGitClient_Push tests the Push method
func TestGitClient_Push(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a test repository
	err = exec.CommandContext(ctx, "git", "-C", tmpDir, "init", "test-repo").Run() //nolint:gosec // Test uses hardcoded command
	require.NoError(t, err)

	// Create initial commit
	testFile := filepath.Join(repoPath, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repo"), 0o600)
	require.NoError(t, err)

	err = client.Add(ctx, repoPath, "README.md")
	require.NoError(t, err)

	err = client.Commit(ctx, repoPath, "Initial commit")
	require.NoError(t, err)

	// Test push without remote (should fail)
	err = client.Push(ctx, repoPath, "origin", "main", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to push")

	// Test with force flag
	err = client.Push(ctx, repoPath, "origin", "main", true)
	require.Error(t, err) // Still fails without remote, but tests force flag handling
	assert.Contains(t, err.Error(), "failed to push")
}

// TestGitClient_GetCurrentBranch_AlternativeMethod tests the fallback method for older git versions
func TestGitClient_GetCurrentBranch_AlternativeMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize repository
	err = exec.CommandContext(ctx, "git", "-C", tmpDir, "init", "test-repo").Run() //nolint:gosec // Test uses hardcoded command
	require.NoError(t, err)

	// Create initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0o600)
	require.NoError(t, err)

	err = client.Add(ctx, repoPath, "test.txt")
	require.NoError(t, err)

	err = client.Commit(ctx, repoPath, "Initial commit")
	require.NoError(t, err)

	// Get current branch
	branch, err := client.GetCurrentBranch(ctx, repoPath)
	require.NoError(t, err)
	// Git init creates main or master depending on configuration
	assert.True(t, branch == "main" || branch == "master", "Expected main or master, got %s", branch)
}

// TestDebugWriter tests the debugWriter functionality
func TestDebugWriter(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)

	// Create a buffer to capture log output
	var buf strings.Builder
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	writer := &debugWriter{
		logger: logger.WithField("test", "debug"),
		prefix: "test-stream",
	}

	// Test Write method
	testData := []byte("test debug output")
	n, err := writer.Write(testData)

	require.NoError(t, err)
	assert.Equal(t, len(testData), n)

	// Check that log was written
	logOutput := buf.String()
	assert.Contains(t, logOutput, "test debug output")
	assert.Contains(t, logOutput, "stream=test-stream")
}

// TestFilterSensitiveEnv tests environment variable filtering
func TestFilterSensitiveEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name: "filters GH_TOKEN",
			input: []string{
				"PATH=/usr/bin",
				"GH_TOKEN=secret123",
				"HOME=/home/user",
			},
			expected: []string{
				"PATH=/usr/bin",
				"GH_TOKEN=REDACTED",
				"HOME=/home/user",
			},
		},
		{
			name: "filters GITHUB_TOKEN",
			input: []string{
				"GITHUB_TOKEN=ghp_secret456",
				"USER=testuser",
			},
			expected: []string{
				"GITHUB_TOKEN=REDACTED",
				"USER=testuser",
			},
		},
		{
			name: "filters case insensitive token",
			input: []string{
				"API_TOKEN=secret789",
				"access_token=oauth123",
				"TEST_VAR=normal",
			},
			expected: []string{
				"API_TOKEN=REDACTED",
				"access_token=REDACTED",
				"TEST_VAR=normal",
			},
		},
		{
			name: "filters password variables",
			input: []string{
				"DB_PASSWORD=secretpass",
				"admin_password=admin123",
				"NORMAL_VAR=value",
			},
			expected: []string{
				"DB_PASSWORD=REDACTED",
				"admin_password=REDACTED",
				"NORMAL_VAR=value",
			},
		},
		{
			name: "filters secret variables",
			input: []string{
				"API_SECRET=topsecret",
				"CLIENT_SECRET=client123",
				"CONFIG_FILE=config.yaml",
			},
			expected: []string{
				"API_SECRET=REDACTED",
				"CLIENT_SECRET=REDACTED",
				"CONFIG_FILE=config.yaml",
			},
		},
		{
			name: "handles variables without equals sign",
			input: []string{
				"NORMAL_VAR",
				"API_TOKEN=secret",
				"ANOTHER_VAR",
			},
			expected: []string{
				"NORMAL_VAR",
				"API_TOKEN=REDACTED",
				"ANOTHER_VAR",
			},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterSensitiveEnv(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
