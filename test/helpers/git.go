package helpers

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// SkipIfNoGit skips the test if git is not available
func SkipIfNoGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
}

// InitGitRepo initializes a git repository in the given directory
func InitGitRepo(dir, initialCommitMsg string) error {
	ctx := context.Background()
	// Initialize git repository
	cmd := exec.CommandContext(ctx, "git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Configure git user
	cmd = exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Create initial commit
	cmd = exec.CommandContext(ctx, "git", "commit", "--allow-empty", "-m", initialCommitMsg) //nolint:gosec // G204: exec uses trusted git command with controlled arguments
	cmd.Dir = dir
	return cmd.Run()
}

// CommitChanges adds and commits all changes in the repository
func CommitChanges(dir, commitMsg string) error {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.CommandContext(ctx, "git", "commit", "-m", commitMsg) //nolint:gosec // G204: exec uses trusted git command with controlled arguments
	cmd.Dir = dir
	return cmd.Run()
}

// GetLatestCommit returns the SHA of the latest commit
func GetLatestCommit(dir string) (string, error) {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// CreateBranch creates a new git branch
func CreateBranch(dir, branchName string) error {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "checkout", "-b", branchName) //nolint:gosec // G204: exec uses trusted git command with controlled arguments
	cmd.Dir = dir
	return cmd.Run()
}

// FileExists checks if a file exists
func FileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath) //nolint:gosec // G703: filePath comes from the caller which validates the path
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
