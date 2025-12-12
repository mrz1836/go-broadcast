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

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// getMainBranches returns the list of main branches from environment variable or default
func getMainBranches() []string {
	mainBranches := os.Getenv("MAIN_BRANCHES")
	if mainBranches == "" {
		mainBranches = "master,main"
	}

	branches := strings.Split(mainBranches, ",")
	for i, branch := range branches {
		branches[i] = strings.TrimSpace(branch)
	}

	return branches
}

// configureGitUser sets up git user configuration for tests
func configureGitUser(ctx context.Context, t *testing.T, repoPath string) {
	t.Helper()

	// Configure git user email
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "config", "user.email", "test@example.com")
	err := cmd.Run()
	require.NoError(t, err)

	// Configure git user name
	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "config", "user.name", "Test User")
	err = cmd.Run()
	require.NoError(t, err)
}

// TestGetCurrentCommitSHA tests the GetCurrentCommitSHA function
func TestGetCurrentCommitSHA(t *testing.T) {
	client, err := NewClient(logrus.New(), &logging.LogConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("ValidRepository", func(t *testing.T) {
		// Create a test repository
		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "test-repo")

		// Initialize repository
		cmd := exec.CommandContext(ctx, "git", "init", repoPath) //nolint:gosec // Git command with safe static args
		require.NoError(t, cmd.Run())

		// Configure git user
		configureGitUser(ctx, t, repoPath)

		// Create a file and commit
		testFile := filepath.Join(repoPath, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))

		cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "add", "test.txt") //nolint:gosec // Git command with safe static args
		require.NoError(t, cmd.Run())

		cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", "Initial commit") //nolint:gosec // Git command with safe static args
		require.NoError(t, cmd.Run())

		// Get commit SHA
		sha, err := client.GetCurrentCommitSHA(ctx, repoPath)
		require.NoError(t, err)
		assert.NotEmpty(t, sha)
		assert.Len(t, sha, 40) // Git SHA-1 is 40 characters
		assert.Regexp(t, "^[a-f0-9]{40}$", sha)
	})

	t.Run("NonExistentRepository", func(t *testing.T) {
		sha, err := client.GetCurrentCommitSHA(ctx, "/non/existent/path")
		require.Error(t, err)
		assert.Empty(t, sha)
		assert.Contains(t, err.Error(), "get current commit SHA")
	})

	t.Run("EmptyRepository", func(t *testing.T) {
		// Create an empty repository
		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "empty-repo")

		cmd := exec.CommandContext(ctx, "git", "init", repoPath) //nolint:gosec // Git command with safe static args
		require.NoError(t, cmd.Run())

		sha, err := client.GetCurrentCommitSHA(ctx, repoPath)
		require.Error(t, err)
		assert.Empty(t, sha)
	})
}

// TestGetRepositoryInfo tests the GetRepositoryInfo function
func TestGetRepositoryInfo(t *testing.T) {
	client, err := NewClient(logrus.New(), &logging.LogConfig{})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("GitHubSSHURL", func(t *testing.T) {
		// Create a test repository with remote
		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "test-repo")

		cmd := exec.CommandContext(ctx, "git", "init", repoPath) //nolint:gosec // Git command with safe static args
		require.NoError(t, cmd.Run())

		// Add a remote
		cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "add", "origin", "git@github.com:owner/repo.git") //nolint:gosec // Git command with safe static args
		require.NoError(t, cmd.Run())

		info, err := client.GetRepositoryInfo(ctx, repoPath)
		require.NoError(t, err)
		assert.Equal(t, "repo", info.Name)
		assert.Equal(t, "owner", info.Owner)
		assert.Equal(t, "owner/repo", info.FullName)
		assert.Equal(t, "git@github.com:owner/repo.git", info.URL)
		assert.True(t, info.IsGitHub)
	})

	t.Run("GitHubHTTPSURL", func(t *testing.T) {
		// Create a test repository with HTTPS remote
		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "test-repo")

		cmd := exec.CommandContext(ctx, "git", "init", repoPath) //nolint:gosec // Git command with safe static args
		require.NoError(t, cmd.Run())

		// Add a remote
		cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "add", "origin", "https://github.com/owner/repo.git") //nolint:gosec // Git command with safe static args
		require.NoError(t, cmd.Run())

		info, err := client.GetRepositoryInfo(ctx, repoPath)
		require.NoError(t, err)
		assert.Equal(t, "repo", info.Name)
		assert.Equal(t, "owner", info.Owner)
		assert.Equal(t, "owner/repo", info.FullName)
		assert.Equal(t, "https://github.com/owner/repo.git", info.URL)
		assert.True(t, info.IsGitHub)
	})

	t.Run("NoRemote", func(t *testing.T) {
		// Create a repository without remote
		tmpDir := testutil.CreateTempDir(t)
		repoPath := filepath.Join(tmpDir, "test-repo")

		cmd := exec.CommandContext(ctx, "git", "init", repoPath) //nolint:gosec // Git command with safe static args
		require.NoError(t, cmd.Run())

		info, err := client.GetRepositoryInfo(ctx, repoPath)
		require.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "get repository info")
	})

	t.Run("NonExistentRepository", func(t *testing.T) {
		info, err := client.GetRepositoryInfo(ctx, "/non/existent/path")
		require.Error(t, err)
		assert.Nil(t, info)
	})
}

// TestParseRepositoryURL tests the parseRepositoryURL function
func TestParseRepositoryURL(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected *RepositoryInfo
		wantErr  bool
	}{
		{
			name: "GitHubSSH",
			url:  "git@github.com:owner/repo.git",
			expected: &RepositoryInfo{
				Name:     "repo",
				Owner:    "owner",
				FullName: "owner/repo",
				URL:      "git@github.com:owner/repo.git",
				IsGitHub: true,
			},
		},
		{
			name: "GitHubSSHWithoutGitExtension",
			url:  "git@github.com:owner/repo",
			expected: &RepositoryInfo{
				Name:     "repo",
				Owner:    "owner",
				FullName: "owner/repo",
				URL:      "git@github.com:owner/repo",
				IsGitHub: true,
			},
		},
		{
			name: "GitHubHTTPS",
			url:  "https://github.com/owner/repo.git",
			expected: &RepositoryInfo{
				Name:     "repo",
				Owner:    "owner",
				FullName: "owner/repo",
				URL:      "https://github.com/owner/repo.git",
				IsGitHub: true,
			},
		},
		{
			name: "GitHubHTTPSWithoutGitExtension",
			url:  "https://github.com/owner/repo",
			expected: &RepositoryInfo{
				Name:     "repo",
				Owner:    "owner",
				FullName: "owner/repo",
				URL:      "https://github.com/owner/repo",
				IsGitHub: true,
			},
		},
		{
			name: "GitLabSSH",
			url:  "git@gitlab.com:owner/repo.git",
			expected: &RepositoryInfo{
				Name:     "repo",
				Owner:    "owner",
				FullName: "owner/repo",
				URL:      "git@gitlab.com:owner/repo.git",
				IsGitHub: false,
			},
		},
		{
			name: "BitbucketHTTPS",
			url:  "https://bitbucket.org/owner/repo.git",
			expected: &RepositoryInfo{
				Name:     "repo",
				Owner:    "owner",
				FullName: "owner/repo",
				URL:      "https://bitbucket.org/owner/repo.git",
				IsGitHub: false,
			},
		},
		{
			name:    "InvalidURL",
			url:     "not-a-valid-url",
			wantErr: true,
		},
		{
			name:    "EmptyURL",
			url:     "",
			wantErr: true,
		},
		{
			name: "ComplexRepoName",
			url:  "git@github.com:owner/repo-with-dashes_and_underscores.git",
			expected: &RepositoryInfo{
				Name:     "repo-with-dashes_and_underscores",
				Owner:    "owner",
				FullName: "owner/repo-with-dashes_and_underscores",
				URL:      "git@github.com:owner/repo-with-dashes_and_underscores.git",
				IsGitHub: true,
			},
		},
		{
			name: "NestedOwner",
			url:  "git@github.com:org/team/repo.git",
			expected: &RepositoryInfo{
				Name:     "team/repo",
				Owner:    "org",
				FullName: "org/team/repo",
				URL:      "git@github.com:org/team/repo.git",
				IsGitHub: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parseRepositoryURL(tc.url)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, info)
			} else {
				require.NoError(t, err)
				require.NotNil(t, info)
				assert.Equal(t, tc.expected.Name, info.Name)
				assert.Equal(t, tc.expected.Owner, info.Owner)
				assert.Equal(t, tc.expected.FullName, info.FullName)
				assert.Equal(t, tc.expected.URL, info.URL)
				assert.Equal(t, tc.expected.IsGitHub, info.IsGitHub)
			}
		})
	}
}

// TestGitClient_Checkout tests the Checkout function
func TestGitClient_Checkout(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	testCases := []struct {
		name         string
		logConfig    *logging.LogConfig
		setupRepo    func(t *testing.T, ctx context.Context, repoPath string)
		branchName   string
		expectError  bool
		errorMessage string
	}{
		{
			name: "CheckoutExistingBranch",
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{Git: false},
			},
			setupRepo: func(t *testing.T, ctx context.Context, repoPath string) {
				// Initialize repository
				cmd := exec.CommandContext(ctx, "git", "init", repoPath)
				require.NoError(t, cmd.Run())

				// Configure git user
				configureGitUser(ctx, t, repoPath)

				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))

				cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "add", "test.txt")
				require.NoError(t, cmd.Run())

				cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", "Initial commit")
				require.NoError(t, cmd.Run())

				// Create a new branch
				cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", "-b", "test-branch")
				require.NoError(t, cmd.Run())

				// Switch back to main/master
				cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", "-")
				require.NoError(t, cmd.Run())
			},
			branchName:  "test-branch",
			expectError: false,
		},
		{
			name: "CheckoutWithDebugLogging",
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{Git: true},
			},
			setupRepo: func(t *testing.T, ctx context.Context, repoPath string) {
				// Initialize repository
				cmd := exec.CommandContext(ctx, "git", "init", repoPath)
				require.NoError(t, cmd.Run())

				// Configure git user
				configureGitUser(ctx, t, repoPath)

				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))

				cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "add", "test.txt")
				require.NoError(t, cmd.Run())

				cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", "Initial commit")
				require.NoError(t, cmd.Run())

				// Create a new branch
				cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", "-b", "debug-branch")
				require.NoError(t, cmd.Run())

				// Switch back to main/master
				cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", "-")
				require.NoError(t, cmd.Run())
			},
			branchName:  "debug-branch",
			expectError: false,
		},
		{
			name: "CheckoutNonExistentBranch",
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{Git: false},
			},
			setupRepo: func(t *testing.T, ctx context.Context, repoPath string) {
				// Initialize repository
				cmd := exec.CommandContext(ctx, "git", "init", repoPath)
				require.NoError(t, cmd.Run())

				// Configure git user
				configureGitUser(ctx, t, repoPath)

				// Create initial commit
				testFile := filepath.Join(repoPath, "test.txt")
				require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))

				cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "add", "test.txt")
				require.NoError(t, cmd.Run())

				cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", "Initial commit")
				require.NoError(t, cmd.Run())
			},
			branchName:   "non-existent-branch",
			expectError:  true,
			errorMessage: "checkout branch non-existent-branch",
		},
		{
			name: "CheckoutInNonGitRepository",
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{Git: false},
			},
			setupRepo: func(t *testing.T, _ context.Context, repoPath string) {
				// Just create an empty directory, not a git repo
				require.NoError(t, os.MkdirAll(repoPath, 0o750))
			},
			branchName:   "any-branch",
			expectError:  true,
			errorMessage: "checkout branch any-branch",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewClient(logger, tc.logConfig)
			require.NoError(t, err)

			ctx := context.Background()
			tmpDir := testutil.CreateTempDir(t)
			repoPath := filepath.Join(tmpDir, "test-repo")

			tc.setupRepo(t, ctx, repoPath)

			err = client.Checkout(ctx, repoPath, tc.branchName)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMessage)
			} else {
				require.NoError(t, err)

				// Verify we're on the correct branch
				cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD") //nolint:gosec // Git command with safe static args
				output, err := cmd.Output()
				require.NoError(t, err)
				assert.Equal(t, tc.branchName, strings.TrimSpace(string(output)))
			}
		})
	}
}

// TestRunCommand tests the runCommand method with various logging configurations
func TestRunCommand(t *testing.T) {
	testCases := []struct {
		name        string
		logConfig   *logging.LogConfig
		loggerLevel logrus.Level
		command     []string
		expectError bool
	}{
		{
			name: "SuccessWithoutDebug",
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{Git: false},
			},
			loggerLevel: logrus.InfoLevel,
			command:     []string{"echo", "test"},
			expectError: false,
		},
		{
			name: "SuccessWithDebugGit",
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{Git: true},
			},
			loggerLevel: logrus.DebugLevel,
			command:     []string{"echo", "test with debug"},
			expectError: false,
		},
		{
			name:        "SuccessWithDebugLogLevel",
			logConfig:   nil,
			loggerLevel: logrus.DebugLevel,
			command:     []string{"echo", "test debug level"},
			expectError: false,
		},
		{
			name: "FailureWithDebug",
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{Git: true},
			},
			loggerLevel: logrus.DebugLevel,
			command:     []string{"false"}, // Command that always fails
			expectError: true,
		},
		{
			name: "FailureWithoutDebug",
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{Git: false},
			},
			loggerLevel: logrus.InfoLevel,
			command:     []string{"false"}, // Command that always fails
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger := logrus.New()
			logger.SetLevel(tc.loggerLevel)

			client, err := NewClient(logger, tc.logConfig)
			require.NoError(t, err)

			ctx := context.Background()
			cmd := exec.CommandContext(ctx, tc.command[0], tc.command[1:]...) //nolint:gosec // Test code with controlled input
			err = client.(*gitClient).runCommand(cmd)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// These are integration tests that require git to be installed
func TestGitClient_Clone(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := testutil.CreateTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Clone a small public repository
	err = client.Clone(ctx, "https://github.com/octocat/Hello-World.git", repoPath, nil)
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
	tmpDir := testutil.CreateTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create directory first
	testutil.CreateTestDirectory(t, repoPath)
	require.NoError(t, err)

	// Try to clone into existing directory
	err = client.Clone(ctx, "https://github.com/octocat/Hello-World.git", repoPath, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRepositoryExists)
}

func TestGitClient_CloneWithBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := testutil.CreateTempDir(t)

	tests := []struct {
		name       string
		branch     string
		expectFile string
	}{
		{
			name:       "clone with master branch",
			branch:     "master",
			expectFile: "README",
		},
		{
			name:       "clone with empty branch (fallback to default)",
			branch:     "",
			expectFile: "README",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := filepath.Join(tmpDir, "test-repo-"+tt.name)

			// Clone with specific branch
			err = client.CloneWithBranch(ctx, "https://github.com/octocat/Hello-World.git", repoPath, tt.branch, nil)
			require.NoError(t, err)

			// Verify the repository was cloned
			assert.DirExists(t, filepath.Join(repoPath, ".git"))
			assert.FileExists(t, filepath.Join(repoPath, tt.expectFile))

			// Verify we're on the expected branch (or default if empty)
			if tt.branch != "" {
				currentBranch, err := client.GetCurrentBranch(ctx, repoPath)
				require.NoError(t, err)
				assert.Equal(t, tt.branch, currentBranch)
			}
		})
	}
}

func TestGitClient_CloneWithBranch_AlreadyExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := testutil.CreateTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create directory first
	testutil.CreateTestDirectory(t, repoPath)
	require.NoError(t, err)

	// Try to clone with branch into existing directory
	err = client.CloneWithBranch(ctx, "https://github.com/octocat/Hello-World.git", repoPath, "master", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRepositoryExists)
}

func TestGitClient_CloneWithBranch_InvalidBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := testutil.CreateTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Try to clone with non-existent branch
	err = client.CloneWithBranch(ctx, "https://github.com/octocat/Hello-World.git", repoPath, "non-existent-branch", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clone repository with branch non-existent-branch")
}

func TestGitClient_Operations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := testutil.CreateTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a new repository
	cmd := exec.CommandContext(ctx, "git", "init", repoPath) //nolint:gosec // Test uses hardcoded command
	err = cmd.Run()
	require.NoError(t, err)

	// Configure git user for tests
	configureGitUser(ctx, t, repoPath)

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
	testutil.WriteTestFile(t, testFile, "test content")

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
	tmpDir := testutil.CreateTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Clone a repository with a remote
	err = client.Clone(ctx, "https://github.com/octocat/Hello-World.git", repoPath, nil)
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

	client, err := NewClient(logrus.New(), nil)
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
	tmpDir := testutil.CreateTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize a test repository
	err = exec.CommandContext(ctx, "git", "-C", tmpDir, "init", "test-repo").Run() //nolint:gosec // Test uses hardcoded command
	require.NoError(t, err)

	// Configure git user for tests
	configureGitUser(ctx, t, repoPath)

	// Create initial commit
	testFile := filepath.Join(repoPath, "README.md")
	testutil.WriteTestFile(t, testFile, "# Test Repo")

	err = client.Add(ctx, repoPath, "README.md")
	require.NoError(t, err)

	err = client.Commit(ctx, repoPath, "Initial commit")
	require.NoError(t, err)

	// Test push without remote (should fail)
	err = client.Push(ctx, repoPath, "origin", "master", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to push")

	// Test with force flag
	err = client.Push(ctx, repoPath, "origin", "master", true)
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
	tmpDir := testutil.CreateTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Initialize repository
	err = exec.CommandContext(ctx, "git", "-C", tmpDir, "init", "test-repo").Run() //nolint:gosec // Test uses hardcoded command
	require.NoError(t, err)

	// Configure git user for tests
	configureGitUser(ctx, t, repoPath)

	// Create initial commit
	testFile := filepath.Join(repoPath, "test.txt")
	testutil.WriteTestFile(t, testFile, "test")

	err = client.Add(ctx, repoPath, "test.txt")
	require.NoError(t, err)

	err = client.Commit(ctx, repoPath, "Initial commit")
	require.NoError(t, err)

	// Get current branch
	branch, err := client.GetCurrentBranch(ctx, repoPath)
	require.NoError(t, err)
	// Git init creates main or master depending on configuration
	mainBranches := getMainBranches()
	isMainBranch := false
	for _, mainBranch := range mainBranches {
		if branch == mainBranch {
			isMainBranch = true
			break
		}
	}
	assert.True(t, isMainBranch, "Expected one of %v, got %s", mainBranches, branch)
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

// TestGitClient_CloneError tests clone error scenarios
func TestGitClient_CloneError(t *testing.T) {
	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test clone with invalid URL
	err = client.Clone(ctx, "invalid://url", "/tmp/test-clone-error", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clone repository")
}

// TestGitClient_CheckoutError tests checkout error scenarios
func TestGitClient_CheckoutError(t *testing.T) {
	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test checkout on non-existent repository
	err = client.Checkout(ctx, "/nonexistent/repo", "main")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to checkout branch main")
}

// TestGitClient_CreateBranchError tests create branch error scenarios
func TestGitClient_CreateBranchError(t *testing.T) {
	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test create branch on non-existent repository
	err = client.CreateBranch(ctx, "/nonexistent/repo", "test-branch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create branch test-branch")
}

// TestGitClient_AddError tests add error scenarios
func TestGitClient_AddError(t *testing.T) {
	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test add on non-existent repository
	err = client.Add(ctx, "/nonexistent/repo", "file.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add files")
}

// TestGitClient_CommitError tests commit error scenarios
func TestGitClient_CommitError(t *testing.T) {
	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test commit on non-existent repository
	err = client.Commit(ctx, "/nonexistent/repo", "test commit")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit")
}

// TestGitClient_PushError tests push error scenarios
func TestGitClient_PushError(t *testing.T) {
	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test push on non-existent repository
	err = client.Push(ctx, "/nonexistent/repo", "origin", "master", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to push")
}

// TestGitClient_DiffError tests diff error scenarios
func TestGitClient_DiffError(t *testing.T) {
	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test diff on non-existent repository
	_, err = client.Diff(ctx, "/nonexistent/repo", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get diff")
}

// TestGitClient_GetCurrentBranchError tests current branch error scenarios
func TestGitClient_GetCurrentBranchError(t *testing.T) {
	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test get current branch on non-existent repository
	_, err = client.GetCurrentBranch(ctx, "/nonexistent/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current branch")
}

// TestGitClient_GetRemoteURLError tests remote URL error scenarios
func TestGitClient_GetRemoteURLError(t *testing.T) {
	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test get remote URL on non-existent repository
	_, err = client.GetRemoteURL(ctx, "/nonexistent/repo", "origin")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get remote URL")
}

// TestGitClient_RunCommandNotARepository tests git error recognition
func TestGitClient_RunCommandNotARepository(t *testing.T) {
	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := testutil.CreateTempDir(t)

	// Try to run git command in directory that's not a repository
	err = client.Checkout(ctx, tmpDir, "master")
	require.Error(t, err)
	// The error should be recognized as "not a git repository"
	assert.ErrorIs(t, err, ErrNotARepository)
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

// TestGitClient_RunCommandWithDebugLogging tests debug logging functionality
func TestGitClient_RunCommandWithDebugLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)

	// Create debug config
	debugConfig := &logging.LogConfig{
		Debug: logging.DebugFlags{
			Git: true,
		},
	}

	client, err := NewClient(logger, debugConfig)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := testutil.CreateTempDir(t)
	repoPath := filepath.Join(tmpDir, "debug-test-repo")

	// Initialize repository to test debug logging
	cmd := exec.CommandContext(ctx, "git", "init", repoPath) //nolint:gosec // Test uses hardcoded command
	err = cmd.Run()
	require.NoError(t, err)

	// Configure git user for tests
	configureGitUser(ctx, t, repoPath)

	// Test operation with debug logging enabled
	branch, err := client.GetCurrentBranch(ctx, repoPath)
	require.NoError(t, err)
	assert.NotEmpty(t, branch)
}

// TestGitClient_CommitNoChangesVariations tests different "no changes" error messages
func TestGitClient_CommitNoChangesVariations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client, err := NewClient(logrus.New(), nil)
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := testutil.CreateTempDir(t)
	repoPath := filepath.Join(tmpDir, "no-changes-repo")

	// Initialize repository
	cmd := exec.CommandContext(ctx, "git", "init", repoPath) //nolint:gosec // Test uses hardcoded command
	err = cmd.Run()
	require.NoError(t, err)

	// Configure git user for tests
	configureGitUser(ctx, t, repoPath)

	// Create and commit initial file
	testFile := filepath.Join(repoPath, "test.txt")
	testutil.WriteTestFile(t, testFile, "initial content")

	err = client.Add(ctx, repoPath, "test.txt")
	require.NoError(t, err)

	err = client.Commit(ctx, repoPath, "Initial commit")
	require.NoError(t, err)

	// Try to commit again with no changes - should return ErrNoChanges
	err = client.Commit(ctx, repoPath, "Empty commit")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoChanges)
}

// TestGitClient_FilterSensitiveEnvEdgeCases tests edge cases in environment filtering
func TestGitClient_FilterSensitiveEnvEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name: "multiple equals signs",
			input: []string{
				"NORMAL_VAR=value=with=equals",
				"API_TOKEN=secret=with=equals",
			},
			expected: []string{
				"NORMAL_VAR=value=with=equals",
				"API_TOKEN=REDACTED",
			},
		},
		{
			name: "mixed case sensitivity",
			input: []string{
				"API_Token=secret123",
				"User_Password=pass456",
				"CLIENT_Secret=client789",
			},
			expected: []string{
				"API_Token=REDACTED",
				"User_Password=REDACTED",
				"CLIENT_Secret=REDACTED",
			},
		},
		{
			name: "empty values",
			input: []string{
				"API_TOKEN=",
				"NORMAL_VAR=",
			},
			expected: []string{
				"API_TOKEN=REDACTED",
				"NORMAL_VAR=",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterSensitiveEnv(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetChangedFiles tests the GetChangedFiles function
func TestGetChangedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testcases := []struct {
		name        string
		setupRepo   func(t *testing.T, repoPath string, client Client) // Setup repo with multiple commits
		expectFiles []string
		expectError bool
	}{
		{
			name: "single file changed in last commit",
			setupRepo: func(t *testing.T, repoPath string, client Client) {
				ctx := context.Background()
				configureGitUser(ctx, t, repoPath)

				// First commit
				err := os.WriteFile(filepath.Join(repoPath, "file1.txt"), []byte("initial content"), 0o600)
				require.NoError(t, err)
				err = client.Add(ctx, repoPath, ".")
				require.NoError(t, err)
				err = client.Commit(ctx, repoPath, "Initial commit")
				require.NoError(t, err)

				// Second commit with changes
				err = os.WriteFile(filepath.Join(repoPath, "file2.txt"), []byte("new file"), 0o600)
				require.NoError(t, err)
				err = client.Add(ctx, repoPath, ".")
				require.NoError(t, err)
				err = client.Commit(ctx, repoPath, "Add file2")
				require.NoError(t, err)
			},
			expectFiles: []string{"file2.txt"},
			expectError: false,
		},
		{
			name: "multiple files changed in last commit",
			setupRepo: func(t *testing.T, repoPath string, client Client) {
				ctx := context.Background()
				configureGitUser(ctx, t, repoPath)

				// First commit
				err := os.WriteFile(filepath.Join(repoPath, "file1.txt"), []byte("initial content"), 0o600)
				require.NoError(t, err)
				err = client.Add(ctx, repoPath, ".")
				require.NoError(t, err)
				err = client.Commit(ctx, repoPath, "Initial commit")
				require.NoError(t, err)

				// Second commit with multiple file changes
				err = os.WriteFile(filepath.Join(repoPath, "file2.txt"), []byte("new file"), 0o600)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(repoPath, "file3.txt"), []byte("another file"), 0o600)
				require.NoError(t, err)
				// Modify existing file
				err = os.WriteFile(filepath.Join(repoPath, "file1.txt"), []byte("modified content"), 0o600)
				require.NoError(t, err)
				err = client.Add(ctx, repoPath, ".")
				require.NoError(t, err)
				err = client.Commit(ctx, repoPath, "Add multiple files and modify existing")
				require.NoError(t, err)
			},
			expectFiles: []string{"file1.txt", "file2.txt", "file3.txt"},
			expectError: false,
		},
		{
			name: "no files changed scenario",
			setupRepo: func(t *testing.T, repoPath string, client Client) {
				ctx := context.Background()
				configureGitUser(ctx, t, repoPath)

				// First commit
				err := os.WriteFile(filepath.Join(repoPath, "file1.txt"), []byte("initial content"), 0o600)
				require.NoError(t, err)
				err = client.Add(ctx, repoPath, ".")
				require.NoError(t, err)
				err = client.Commit(ctx, repoPath, "Initial commit")
				require.NoError(t, err)

				// Second commit with same content (this would not happen in practice but tests the edge case)
				// Use direct git command for --allow-empty since the client doesn't support it directly
				cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "--allow-empty", "-m", "Empty commit")
				err = cmd.Run()
				require.NoError(t, err)
			},
			expectFiles: []string{},
			expectError: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Initialize repository
			ctx := context.Background()
			cmd := exec.CommandContext(ctx, "git", "-C", tempDir, "init") // #nosec G204
			err := cmd.Run()
			require.NoError(t, err)

			// Create client and setup repo
			logger := logrus.New()
			client, err := NewClient(logger, nil)
			require.NoError(t, err)
			tc.setupRepo(t, tempDir, client)

			// Test GetChangedFiles
			files, err := client.GetChangedFiles(ctx, tempDir)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				// Sort both slices for comparison since git may return files in different order
				assert.ElementsMatch(t, tc.expectFiles, files)
			}
		})
	}
}

// TestGetChangedFiles_InitialCommit tests GetChangedFiles behavior with only one commit (no HEAD~1)
func TestGetChangedFiles_InitialCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tempDir := t.TempDir()
	ctx := context.Background()

	// Initialize repository
	cmd := exec.CommandContext(ctx, "git", "-C", tempDir, "init") // #nosec G204
	err := cmd.Run()
	require.NoError(t, err)
	configureGitUser(ctx, t, tempDir)

	// Create client
	logger := logrus.New()
	client, err := NewClient(logger, nil)
	require.NoError(t, err)

	// Only one commit (no HEAD~1 exists)
	err = os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("initial content"), 0o600)
	require.NoError(t, err)
	err = client.Add(ctx, tempDir, ".")
	require.NoError(t, err)
	err = client.Commit(ctx, tempDir, "Initial commit")
	require.NoError(t, err)

	// GetChangedFiles should handle this gracefully (error expected since HEAD~1 doesn't exist)
	files, err := client.GetChangedFiles(ctx, tempDir)
	// We expect an error here since there's no HEAD~1
	require.Error(t, err)
	assert.Nil(t, files)
}

// TestDetectGitError tests the centralized error detection function
func TestDetectGitError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected error
	}{
		// Branch already exists patterns
		{
			name:     "branch already exists lowercase",
			errMsg:   "error: branch 'feature' already exists",
			expected: ErrBranchAlreadyExists,
		},
		{
			name:     "branch already exists uppercase",
			errMsg:   "ERROR: BRANCH ALREADY EXISTS",
			expected: ErrBranchAlreadyExists,
		},
		{
			name:     "updates were rejected",
			errMsg:   "error: failed to push some refs, updates were rejected",
			expected: ErrBranchAlreadyExists,
		},
		{
			name:     "non-fast-forward",
			errMsg:   "hint: Updates were rejected because a pushed branch tip is behind non-fast-forward",
			expected: ErrBranchAlreadyExists,
		},
		{
			name:     "fetch first",
			errMsg:   "error: failed to push, fetch first",
			expected: ErrBranchAlreadyExists,
		},
		// No changes patterns
		{
			name:     "nothing to commit",
			errMsg:   "nothing to commit, working tree clean",
			expected: ErrNoChanges,
		},
		{
			name:     "no changes",
			errMsg:   "On branch main, no changes to commit",
			expected: ErrNoChanges,
		},
		{
			name:     "working tree clean",
			errMsg:   "nothing to commit, working tree clean",
			expected: ErrNoChanges,
		},
		{
			name:     "nothing added to commit",
			errMsg:   "nothing added to commit but untracked files present",
			expected: ErrNoChanges,
		},
		// Not a repository pattern
		{
			name:     "not a git repository",
			errMsg:   "fatal: not a git repository (or any of the parent directories): .git",
			expected: ErrNotARepository,
		},
		// Unknown error
		{
			name:     "unknown error",
			errMsg:   "some random error message",
			expected: nil,
		},
		{
			name:     "empty error message",
			errMsg:   "",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := detectGitError(tc.errMsg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestNewClient_NilLogger tests that NewClient returns an error when logger is nil
func TestNewClient_NilLogger(t *testing.T) {
	client, err := NewClient(nil, &logging.LogConfig{})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNilLogger)
	assert.Nil(t, client)
}

// TestNewClient_NilLogConfig tests that NewClient works with nil logConfig
func TestNewClient_NilLogConfig(t *testing.T) {
	logger := logrus.New()
	client, err := NewClient(logger, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
}
