package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/logging"
)

// Common errors
var (
	ErrGitNotFound         = errors.New("git command not found in PATH")
	ErrNotARepository      = errors.New("not a git repository")
	ErrRepositoryExists    = errors.New("repository already exists")
	ErrNoChanges           = errors.New("no changes to commit")
	ErrGitCommand          = errors.New("git command failed")
	ErrInvalidRepoURL      = errors.New("invalid repository URL format")
	ErrBranchAlreadyExists = errors.New("branch already exists on remote")
)

// gitClient implements the Client interface using git commands
type gitClient struct {
	logger    *logrus.Logger
	logConfig *logging.LogConfig
}

// NewClient creates a new Git client.
//
// Parameters:
// - logger: Logger instance for general logging
// - logConfig: Configuration for debug logging and verbose settings
//
// Returns:
// - Git client interface implementation
// - Error if git command is not available in PATH
func NewClient(logger *logrus.Logger, logConfig *logging.LogConfig) (Client, error) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return nil, ErrGitNotFound
	}

	return &gitClient{
		logger:    logger,
		logConfig: logConfig,
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
		return appErrors.WrapWithContext(err, "clone repository")
	}

	return nil
}

// Checkout switches to the specified branch
func (g *gitClient) Checkout(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", branch)

	if err := g.runCommand(cmd); err != nil {
		return appErrors.WrapWithContext(err, fmt.Sprintf("checkout branch %s", branch))
	}

	return nil
}

// CreateBranch creates a new branch from the current HEAD
func (g *gitClient) CreateBranch(ctx context.Context, repoPath, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", "-b", branch)

	if err := g.runCommand(cmd); err != nil {
		// Check if it's a branch already exists error
		errStr := err.Error()
		if strings.Contains(errStr, "already exists") {
			return ErrBranchAlreadyExists
		}
		return appErrors.WrapWithContext(err, fmt.Sprintf("create branch %s", branch))
	}

	return nil
}

// Add stages files for commit
func (g *gitClient) Add(ctx context.Context, repoPath string, paths ...string) error {
	args := []string{"-C", repoPath, "add"}
	args = append(args, paths...)

	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

	if err := g.runCommand(cmd); err != nil {
		return appErrors.WrapWithContext(err, "add files")
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
		return appErrors.WrapWithContext(err, "commit")
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
		// Check if it's a branch already exists error
		errStr := err.Error()
		if strings.Contains(errStr, "already exists") ||
			strings.Contains(errStr, "updates were rejected") ||
			strings.Contains(errStr, "non-fast-forward") ||
			strings.Contains(errStr, "fetch first") {
			return ErrBranchAlreadyExists
		}
		return appErrors.WrapWithContext(err, "push")
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
		return "", appErrors.WrapWithContext(err, "get diff")
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
			return "", appErrors.WrapWithContext(err, "get current branch")
		}
	}

	return strings.TrimSpace(string(output)), nil
}

// GetRemoteURL returns the URL of the specified remote
func (g *gitClient) GetRemoteURL(ctx context.Context, repoPath, remote string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "get-url", remote)

	output, err := cmd.Output()
	if err != nil {
		return "", appErrors.WrapWithContext(err, "get remote URL")
	}

	return strings.TrimSpace(string(output)), nil
}

// AddRemote adds a new remote to the repository
func (g *gitClient) AddRemote(ctx context.Context, repoPath, remoteName, remoteURL string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "remote", "add", remoteName, remoteURL)

	if err := g.runCommand(cmd); err != nil {
		return appErrors.WrapWithContext(err, "add remote")
	}

	return nil
}

// GetCurrentCommitSHA returns the SHA of the current commit
func (g *gitClient) GetCurrentCommitSHA(ctx context.Context, repoPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "rev-parse", "HEAD")

	output, err := cmd.Output()
	if err != nil {
		return "", appErrors.WrapWithContext(err, "get current commit SHA")
	}

	return strings.TrimSpace(string(output)), nil
}

// GetRepositoryInfo extracts repository information from Git remote
func (g *gitClient) GetRepositoryInfo(ctx context.Context, repoPath string) (*RepositoryInfo, error) {
	// Get the origin remote URL
	remoteURL, err := g.GetRemoteURL(ctx, repoPath, "origin")
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "get repository info")
	}

	return parseRepositoryURL(remoteURL)
}

// parseRepositoryURL parses a Git remote URL and extracts repository information
func parseRepositoryURL(remoteURL string) (*RepositoryInfo, error) {
	// Handle SSH URLs (git@github.com:owner/repo.git)
	sshPattern := regexp.MustCompile(`^git@([^:]+):([^/]+)/(.+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(remoteURL); len(matches) == 4 {
		host := matches[1]
		owner := matches[2]
		repo := matches[3]

		return &RepositoryInfo{
			Name:     repo,
			Owner:    owner,
			FullName: fmt.Sprintf("%s/%s", owner, repo),
			URL:      remoteURL,
			IsGitHub: host == "github.com",
		}, nil
	}

	// Handle HTTPS URLs
	parsedURL, err := url.Parse(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL: %w", err)
	}

	// Extract owner and repository from path
	path := strings.Trim(parsedURL.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRepoURL, remoteURL)
	}

	owner := parts[0]
	repo := parts[1]

	return &RepositoryInfo{
		Name:     repo,
		Owner:    owner,
		FullName: fmt.Sprintf("%s/%s", owner, repo),
		URL:      remoteURL,
		IsGitHub: parsedURL.Host == "github.com",
	}, nil
}

// runCommand executes a git command with comprehensive debug logging support.
//
// This method provides detailed visibility into git command execution when debug
// logging is enabled, including command details, timing, output capture, and
// environment variable filtering for security.
//
// Parameters:
// - cmd: The exec.Cmd to execute
//
// Returns:
// - Error if command execution fails
//
// Side Effects:
// - Logs command execution details when --debug-git flag is enabled
// - Captures and logs real-time stdout/stderr output at trace level
// - Records command timing and exit code information
func (g *gitClient) runCommand(cmd *exec.Cmd) error {
	logger := logging.WithStandardFields(g.logger, g.logConfig, logging.ComponentNames.Git)

	// Debug logging when --debug-git flag is enabled
	if g.logConfig != nil && g.logConfig.Debug.Git {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.Operation: logging.OperationTypes.GitCommand,
			"command":                        cmd.Path,
			"args":                           cmd.Args[1:], // Skip command name for cleaner logs
			"dir":                            cmd.Dir,
			"env":                            filterSensitiveEnv(cmd.Env),
		}).Debug("Executing git command")

		// Capture and log output in real-time using debug writers
		cmd.Stdout = &debugWriter{logger: logger, prefix: "stdout"}
		cmd.Stderr = &debugWriter{logger: logger, prefix: "stderr"}
	} else {
		// Fallback to buffer capture for error handling
		var stderr bytes.Buffer
		var stdout bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = &stdout
	}

	// Execute command with timing
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	// Log completion with timing and exit code
	if g.logConfig != nil && g.logConfig.Debug.Git {
		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}

		logger.WithFields(logrus.Fields{
			logging.StandardFields.DurationMs: duration.Milliseconds(),
			logging.StandardFields.ExitCode:   exitCode,
			logging.StandardFields.Status:     "completed",
		}).Debug("Git command completed")
	} else if g.logger != nil && g.logger.IsLevelEnabled(logrus.DebugLevel) {
		// Basic logging for backwards compatibility
		g.logger.WithField("command", strings.Join(cmd.Args, " ")).Debug("Executing git command")
	}

	// Handle success case
	if err == nil {
		return nil
	}

	// Handle error case with detailed logging
	var errMsg, outMsg string
	if g.logConfig == nil || !g.logConfig.Debug.Git {
		// Extract error messages from buffers when not using debug writers
		if stderr, ok := cmd.Stderr.(*bytes.Buffer); ok {
			errMsg = stderr.String()
		}
		if stdout, ok := cmd.Stdout.(*bytes.Buffer); ok {
			outMsg = stdout.String()
		}

		if g.logger != nil {
			g.logger.WithFields(logrus.Fields{
				logging.StandardFields.Component: logging.ComponentNames.Git,
				"command":                        strings.Join(cmd.Args, " "),
				logging.StandardFields.Error:     errMsg,
				"output":                         outMsg,
				logging.StandardFields.Status:    "failed",
			}).Error("Git command failed")
		}
	} else {
		// When using debug logging, command details already logged above
		logger.WithError(err).Error("Git command failed")
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

// debugWriter implements io.Writer for real-time git command output logging.
//
// This writer captures stdout/stderr output from git commands and logs each
// line at trace level when debug logging is enabled, providing real-time
// visibility into git command execution.
type debugWriter struct {
	logger *logrus.Entry
	prefix string
}

// Write implements io.Writer interface for capturing git command output.
//
// Parameters:
// - p: Byte slice containing the output data to log
//
// Returns:
// - Number of bytes written (always len(p))
// - Error (always nil in current implementation)
//
// Side Effects:
// - Logs the output content at trace level with stream identification
func (w *debugWriter) Write(p []byte) (n int, err error) {
	w.logger.WithField("stream", w.prefix).Trace(string(p))
	return len(p), nil
}

// filterSensitiveEnv filters environment variables to redact sensitive information.
//
// This function processes environment variables to identify and redact tokens,
// passwords, and other sensitive information before logging, ensuring security
// compliance while maintaining debugging visibility.
//
// Parameters:
// - env: Slice of environment variable strings in "KEY=VALUE" format
//
// Returns:
// - Filtered slice with sensitive values replaced with "REDACTED"
//
// Security:
// - Automatically redacts GH_TOKEN, GITHUB_TOKEN, and similar patterns
// - Preserves variable names for debugging while protecting values
func filterSensitiveEnv(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, "GH_TOKEN=") ||
			strings.HasPrefix(e, "GITHUB_TOKEN=") ||
			strings.Contains(strings.ToLower(e), "token=") ||
			strings.Contains(strings.ToLower(e), "password=") ||
			strings.Contains(strings.ToLower(e), "secret=") {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				filtered = append(filtered, parts[0]+"=REDACTED")
			} else {
				filtered = append(filtered, e)
			}
		} else {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
