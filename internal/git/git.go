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
	"sync"
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
	ErrNilLogger           = errors.New("logger cannot be nil")
)

// errorPatterns maps git error message patterns to sentinel errors.
// Using lowercase patterns for case-insensitive matching.
//
//nolint:gochecknoglobals // Package-level lookup table for error pattern matching - immutable after init
var errorPatterns = map[*error][]string{
	&ErrBranchAlreadyExists: {
		"already exists",
		"updates were rejected",
		"non-fast-forward",
		"fetch first",
	},
	&ErrNoChanges: {
		"nothing to commit",
		"no changes",
		"working tree clean",
		"nothing added to commit",
	},
	&ErrNotARepository: {
		"not a git repository",
	},
}

// detectGitError maps git command output to sentinel errors.
// It performs case-insensitive matching against known error patterns.
func detectGitError(errMsg string) error {
	if errMsg == "" {
		return nil
	}
	normalizedMsg := strings.ToLower(errMsg)
	for sentinelErr, patterns := range errorPatterns {
		for _, pattern := range patterns {
			if strings.Contains(normalizedMsg, pattern) {
				return *sentinelErr
			}
		}
	}
	return nil
}

// sshURLPattern is a pre-compiled regex for parsing SSH git URLs.
// Compiled at package level for better performance.
var sshURLPattern = regexp.MustCompile(`^git@([^:]+):([^/]+)/(.+?)(?:\.git)?$`)

// gitClient implements the Client interface using git commands
type gitClient struct {
	logger    *logrus.Logger
	logConfig *logging.LogConfig
}

// NewClient creates a new Git client.
//
// Parameters:
// - logger: Logger instance for general logging (required, cannot be nil)
// - logConfig: Configuration for debug logging and verbose settings
//
// Returns:
// - Git client interface implementation
// - Error if logger is nil or git command is not available in PATH
func NewClient(logger *logrus.Logger, logConfig *logging.LogConfig) (Client, error) {
	// Validate logger is not nil to prevent panics in logging calls
	if logger == nil {
		return nil, ErrNilLogger
	}

	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return nil, ErrGitNotFound
	}

	return &gitClient{
		logger:    logger,
		logConfig: logConfig,
	}, nil
}

// Clone clones a repository to the specified path with retry logic for network errors.
// opts can be nil to use default behavior (no blob filtering).
func (g *gitClient) Clone(ctx context.Context, url, path string, opts *CloneOptions) error {
	// Check if path already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w: %s", ErrRepositoryExists, path)
	}

	// Build clone arguments
	args := []string{"clone"}

	// Add blob filter if specified and not "0"
	if opts != nil && opts.BlobSizeLimit != "" && opts.BlobSizeLimit != "0" {
		args = append(args, "--filter=blob:limit="+opts.BlobSizeLimit)
	}

	args = append(args, url, path)

	// Retry logic for network errors
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		err := g.runCommand(cmd)
		if err == nil {
			return nil // Success
		}

		// Check if it's a retryable network error
		if isRetryableNetworkError(err) && attempt < maxRetries {
			logger := logging.WithStandardFields(g.logger, g.logConfig, logging.ComponentNames.Git)
			logger.WithFields(logrus.Fields{
				"attempt":     attempt,
				"max_retries": maxRetries,
				"url":         url,
				"error":       err.Error(),
			}).Warn("Network error during git clone - retrying")

			// Clean up failed partial clone
			if cleanupErr := os.RemoveAll(path); cleanupErr != nil {
				logger.WithError(cleanupErr).Debug("Failed to clean up partial clone")
			}

			// Brief delay before retry
			select {
			case <-time.After(time.Duration(attempt) * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		// Non-retryable error or max retries exceeded
		return appErrors.WrapWithContext(err, "clone repository")
	}

	return fmt.Errorf("%w: clone failed after %d attempts", ErrGitCommand, maxRetries)
}

// CloneWithBranch clones a repository to the specified path with a specific branch.
// If branch is empty, behaves like Clone.
// opts can be nil to use default behavior (no blob filtering).
func (g *gitClient) CloneWithBranch(ctx context.Context, url, path, branch string, opts *CloneOptions) error {
	// Check if path already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w: %s", ErrRepositoryExists, path)
	}

	// If no branch specified, use regular clone
	if branch == "" {
		return g.Clone(ctx, url, path, opts)
	}

	logger := logging.WithStandardFields(g.logger, g.logConfig, logging.ComponentNames.Git)
	logger.WithFields(logrus.Fields{
		"url":    url,
		"path":   path,
		"branch": branch,
	}).Debug("Cloning repository with specific branch")

	// Build clone arguments
	args := []string{"clone"}

	// Add blob filter if specified and not "0"
	if opts != nil && opts.BlobSizeLimit != "" && opts.BlobSizeLimit != "0" {
		args = append(args, "--filter=blob:limit="+opts.BlobSizeLimit)
	}

	args = append(args, "--branch", branch, url, path)

	// Retry logic for network errors
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		err := g.runCommand(cmd)
		if err == nil {
			logger.WithField("branch", branch).Info("Successfully cloned repository with branch")
			return nil // Success
		}

		// Check if it's a retryable network error
		if isRetryableNetworkError(err) && attempt < maxRetries {
			logger.WithFields(logrus.Fields{
				"attempt":     attempt,
				"max_retries": maxRetries,
				"url":         url,
				"branch":      branch,
				"error":       err.Error(),
			}).Warn("Network error during git clone with branch - retrying")

			// Clean up failed partial clone
			if cleanupErr := os.RemoveAll(path); cleanupErr != nil {
				logger.WithError(cleanupErr).Debug("Failed to clean up partial clone")
			}

			// Brief delay before retry
			select {
			case <-time.After(time.Duration(attempt) * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		// Non-retryable error or max retries exceeded
		return appErrors.WrapWithContext(err, fmt.Sprintf("clone repository with branch %s", branch))
	}

	return fmt.Errorf("%w: clone with branch %s failed after %d attempts", ErrGitCommand, branch, maxRetries)
}

// CloneAtTag clones a repository at a specific tag with a shallow clone (depth 1).
// This is optimized for fetching a specific version without full history.
// opts can be nil to use default behavior.
func (g *gitClient) CloneAtTag(ctx context.Context, url, path, tag string, opts *CloneOptions) error {
	// Check if path already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%w: %s", ErrRepositoryExists, path)
	}

	// Tag is required for this method
	if tag == "" {
		return fmt.Errorf("%w: tag cannot be empty", ErrGitCommand)
	}

	logger := logging.WithStandardFields(g.logger, g.logConfig, logging.ComponentNames.Git)
	logger.WithFields(logrus.Fields{
		"url":  url,
		"path": path,
		"tag":  tag,
	}).Debug("Cloning repository at specific tag (shallow)")

	// Build clone arguments with --depth 1 for shallow clone
	args := []string{"clone", "--depth", "1", "--branch", tag}

	// Add blob filter if specified and not "0"
	if opts != nil && opts.BlobSizeLimit != "" && opts.BlobSizeLimit != "0" {
		args = append(args, "--filter=blob:limit="+opts.BlobSizeLimit)
	}

	args = append(args, url, path)

	// Retry logic for network errors
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed from validated tag/url inputs
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		err := g.runCommand(cmd)
		if err == nil {
			logger.WithField("tag", tag).Info("Successfully cloned repository at tag")
			return nil // Success
		}

		// Check if it's a retryable network error
		if isRetryableNetworkError(err) && attempt < maxRetries {
			logger.WithFields(logrus.Fields{
				"attempt":     attempt,
				"max_retries": maxRetries,
				"url":         url,
				"tag":         tag,
				"error":       err.Error(),
			}).Warn("Network error during git clone at tag - retrying")

			// Clean up failed partial clone
			if cleanupErr := os.RemoveAll(path); cleanupErr != nil {
				logger.WithError(cleanupErr).Debug("Failed to clean up partial clone")
			}

			// Brief delay before retry
			select {
			case <-time.After(time.Duration(attempt) * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		// Non-retryable error or max retries exceeded
		return appErrors.WrapWithContext(err, fmt.Sprintf("clone repository at tag %s", tag))
	}

	return fmt.Errorf("%w: clone at tag %s failed after %d attempts", ErrGitCommand, tag, maxRetries)
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
		// Check for known error patterns
		if sentinel := detectGitError(err.Error()); sentinel != nil {
			return sentinel
		}
		return appErrors.WrapWithContext(err, fmt.Sprintf("create branch %s", branch))
	}

	return nil
}

// Add stages files for commit
func (g *gitClient) Add(ctx context.Context, repoPath string, paths ...string) error {
	args := make([]string, 0, 3+len(paths))
	args = append(args, "-C", repoPath, "add")
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
		// Check for known error patterns
		if sentinel := detectGitError(err.Error()); sentinel != nil {
			return sentinel
		}
		return appErrors.WrapWithContext(err, "commit")
	}

	return nil
}

// Push pushes the current branch to the remote with retry logic for network errors.
func (g *gitClient) Push(ctx context.Context, repoPath, remote, branch string, force bool) error {
	args := []string{"-C", repoPath, "push", remote, branch}
	if force {
		args = append(args, "--force")
	}

	// Retry logic for network errors (same pattern as Clone)
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

		err := g.runCommand(cmd)
		if err == nil {
			return nil // Success
		}

		// Check for known error patterns (sentinel errors are not retried)
		if sentinel := detectGitError(err.Error()); sentinel != nil {
			return sentinel
		}

		// Check if it's a retryable network error
		if isRetryableNetworkError(err) && attempt < maxRetries {
			logger := logging.WithStandardFields(g.logger, g.logConfig, logging.ComponentNames.Git)
			logger.WithFields(logrus.Fields{
				"attempt":     attempt,
				"max_retries": maxRetries,
				"branch":      branch,
				"remote":      remote,
				"repo_path":   repoPath,
				"error":       err.Error(),
			}).Warn("Network error during git push - retrying")

			// Brief delay before retry
			select {
			case <-time.After(time.Duration(attempt) * time.Second):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		// Non-retryable error or max retries exceeded
		return appErrors.WrapWithContext(err, "push")
	}

	return fmt.Errorf("%w: push failed after %d attempts", ErrGitCommand, maxRetries)
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

// DiffIgnoreWhitespace returns the diff ignoring whitespace changes.
// Uses -w flag to ignore all whitespace (spaces, tabs, line endings).
// This is useful for AI context where line ending normalization can mask real changes.
func (g *gitClient) DiffIgnoreWhitespace(ctx context.Context, repoPath string, staged bool) (string, error) {
	args := []string{"-C", repoPath, "diff", "-w"}
	if staged {
		args = append(args, "--staged")
	}

	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

	output, err := cmd.Output()
	if err != nil {
		return "", appErrors.WrapWithContext(err, "get diff ignoring whitespace")
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
	if matches := sshURLPattern.FindStringSubmatch(remoteURL); len(matches) == 4 {
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

// GetChangedFiles returns the list of files that changed in the last commit
func (g *gitClient) GetChangedFiles(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "diff", "--name-only", "HEAD~1")

	output, err := cmd.Output()
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "get changed files")
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Filter out empty strings
	var result []string
	for _, file := range files {
		if file != "" {
			result = append(result, file)
		}
	}

	return result, nil
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
		// Safely extract args, skipping command name if present
		var args []string
		if len(cmd.Args) > 1 {
			args = cmd.Args[1:]
		}
		logger.WithFields(logrus.Fields{
			logging.StandardFields.Operation: logging.OperationTypes.GitCommand,
			"command":                        cmd.Path,
			"args":                           args,
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

	// Extract error messages - works for both debug writers and regular buffers
	if dw, ok := cmd.Stderr.(*debugWriter); ok {
		errMsg = dw.String()
	} else if buf, ok := cmd.Stderr.(*bytes.Buffer); ok {
		errMsg = buf.String()
	}

	if dw, ok := cmd.Stdout.(*debugWriter); ok {
		outMsg = dw.String()
	} else if buf, ok := cmd.Stdout.(*bytes.Buffer); ok {
		outMsg = buf.String()
	}

	if g.logConfig != nil && g.logConfig.Debug.Git {
		// When using debug logging, command details already logged above
		logger.WithError(err).Error("Git command failed")
	} else if g.logger != nil {
		g.logger.WithFields(logrus.Fields{
			logging.StandardFields.Component: logging.ComponentNames.Git,
			"command":                        strings.Join(cmd.Args, " "),
			logging.StandardFields.Error:     errMsg,
			"output":                         outMsg,
			logging.StandardFields.Status:    "failed",
		}).Error("Git command failed")
	}

	// Check for common error patterns using centralized detection
	if sentinel := detectGitError(errMsg); sentinel != nil {
		return sentinel
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
// visibility into git command execution. It also buffers content for later
// error message extraction.
type debugWriter struct {
	logger *logrus.Entry
	prefix string
	buffer bytes.Buffer
	mu     sync.Mutex
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
// - Buffers the content for later error extraction
// - Logs the output content at trace level with stream identification
func (w *debugWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	w.buffer.Write(p)
	w.mu.Unlock()
	w.logger.WithField("stream", w.prefix).Trace(string(p))
	return len(p), nil
}

// String returns the buffered content for error message extraction.
func (w *debugWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}

// isRetryableNetworkError checks if an error is a transient network error that should be retried
func isRetryableNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "early eof") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "couldn't connect")
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
