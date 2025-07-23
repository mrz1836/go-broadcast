package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

// Common errors
var (
	ErrGitNotFound      = errors.New("git command not found in PATH")
	ErrNotARepository   = errors.New("not a git repository")
	ErrRepositoryExists = errors.New("repository already exists")
	ErrNoChanges        = errors.New("no changes to commit")
	ErrGitCommand       = errors.New("git command failed")
)

// gitClient implements the Client interface using git commands
type gitClient struct {
	logger *logrus.Logger
}

// NewClient creates a new Git client
func NewClient(logger *logrus.Logger) (Client, error) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return nil, ErrGitNotFound
	}

	return &gitClient{
		logger: logger,
	}, nil
}

// Clone clones a repository to the specified path
func (g *gitClient) Clone(ctx context.Context, url, path string) error {
	// Check if path already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w: %s", ErrRepositoryExists, path)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", url, path)

	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	if err := g.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

// Checkout switches to the specified branch
func (g *gitClient) Checkout(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", branch)

	if err := g.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branch, err)
	}

	return nil
}

// CreateBranch creates a new branch from the current HEAD
func (g *gitClient) CreateBranch(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", "-b", branch)

	if err := g.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branch, err)
	}

	return nil
}

// Add stages files for commit
func (g *gitClient) Add(ctx context.Context, repoPath string, paths ...string) error {
	args := []string{"-C", repoPath, "add"}
	args = append(args, paths...)

	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

	if err := g.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	return nil
}

// Commit creates a commit with the given message
func (g *gitClient) Commit(ctx context.Context, repoPath, message string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "commit", "-m", message)

	if err := g.runCommand(cmd); err != nil {
		// Check if it's because there are no changes
		errStr := err.Error()
		if strings.Contains(errStr, "nothing to commit") ||
			strings.Contains(errStr, "no changes") ||
			strings.Contains(errStr, "working tree clean") ||
			strings.Contains(errStr, "nothing added to commit") {
			return ErrNoChanges
		}
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// Push pushes the current branch to the remote
func (g *gitClient) Push(ctx context.Context, repoPath, remote, branch string, force bool) error {
	args := []string{"-C", repoPath, "push", remote, branch}
	if force {
		args = append(args, "--force")
	}

	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	if err := g.runCommand(cmd); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// Diff returns the diff of changes
func (g *gitClient) Diff(ctx context.Context, repoPath string, staged bool) (string, error) {
	args := []string{"-C", repoPath, "diff"}
	if staged {
		args = append(args, "--staged")
	}

	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// GetCurrentBranch returns the name of the current branch
func (g *gitClient) GetCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "--show-current")

	output, err := cmd.Output()
	if err != nil {
		// Try alternative method for older git versions
		cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "symbolic-ref", "--short", "HEAD")

		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w", err)
		}
	}

	return strings.TrimSpace(string(output)), nil
}

// GetRemoteURL returns the URL of the specified remote
func (g *gitClient) GetRemoteURL(ctx context.Context, repoPath, remote string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "get-url", remote)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// runCommand executes a command and logs the output
func (g *gitClient) runCommand(cmd *exec.Cmd) error {
	if g.logger != nil && g.logger.IsLevelEnabled(logrus.DebugLevel) {
		g.logger.WithField("command", strings.Join(cmd.Args, " ")).Debug("Executing git command")
	}

	var stderr bytes.Buffer
	var stdout bytes.Buffer

	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err == nil {
		return nil
	}

	errMsg := stderr.String()
	outMsg := stdout.String()
	if g.logger != nil {
		g.logger.WithFields(logrus.Fields{
			"command": strings.Join(cmd.Args, " "),
			"error":   errMsg,
			"output":  outMsg,
		}).Error("Git command failed")
	}

	// Check for common error patterns
	if strings.Contains(errMsg, "not a git repository") {
		return ErrNotARepository
	}

	// Return error with stderr content (or stdout if stderr is empty)
	if errMsg != "" {
		return fmt.Errorf("%w: %s", ErrGitCommand, errMsg)
	}
	if outMsg != "" {
		return fmt.Errorf("%w: %s", ErrGitCommand, outMsg)
	}
	return err
}
