package gh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/sirupsen/logrus"
)

// Common errors
var (
	ErrNotAuthenticated = errors.New("gh CLI not authenticated")
	ErrGHNotFound       = errors.New("gh CLI not found in PATH")
	ErrRateLimited      = errors.New("GitHub API rate limit exceeded")
	ErrBranchNotFound   = errors.New("branch not found")
	ErrPRNotFound       = errors.New("pull request not found")
	ErrFileNotFound     = errors.New("file not found")
	ErrCommitNotFound   = errors.New("commit not found")
)

// githubClient implements the Client interface using gh CLI
type githubClient struct {
	runner CommandRunner
	logger *logrus.Logger
}

// NewClient creates a new GitHub client using gh CLI.
//
// Parameters:
// - ctx: Context for authentication check and cancellation
// - logger: Logger instance for general logging
// - logConfig: Configuration for debug logging and verbose settings
//
// Returns:
// - GitHub client interface implementation
// - Error if gh CLI is not available or not authenticated
func NewClient(ctx context.Context, logger *logrus.Logger, logConfig *logging.LogConfig) (Client, error) {
	// Initialize audit logger for security event tracking
	auditLogger := logging.NewAuditLogger()

	// Check if gh is available
	if _, err := exec.LookPath("gh"); err != nil {
		auditLogger.LogAuthentication("system", "github_cli", false)
		return nil, ErrGHNotFound
	}

	runner := NewCommandRunner(logger, logConfig)

	// Check authentication status
	if _, err := runner.Run(ctx, "gh", "auth", "status"); err != nil {
		auditLogger.LogAuthentication("unknown", "github_cli", false)
		return nil, fmt.Errorf("%w: gh auth status failed", ErrNotAuthenticated)
	}

	// Log successful authentication
	auditLogger.LogAuthentication("github_cli", "github_token", true)

	return &githubClient{
		runner: runner,
		logger: logger,
	}, nil
}

// ListBranches returns all branches for a repository
func (g *githubClient) ListBranches(ctx context.Context, repo string) ([]Branch, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/branches", repo), "--paginate")
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "list branches")
	}

	var branches []Branch
	if err := json.Unmarshal(output, &branches); err != nil {
		return nil, appErrors.WrapWithContext(err, "parse branches")
	}

	return branches, nil
}

// GetBranch returns details for a specific branch
func (g *githubClient) GetBranch(ctx context.Context, repo, branch string) (*Branch, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/branches/%s", repo, branch))
	if err != nil {
		if isNotFoundError(err) {
			return nil, ErrBranchNotFound
		}
		return nil, appErrors.WrapWithContext(err, "get branch")
	}

	var b Branch
	if err := json.Unmarshal(output, &b); err != nil {
		return nil, appErrors.WrapWithContext(err, "parse branch")
	}

	return &b, nil
}

// CreatePR creates a new pull request
func (g *githubClient) CreatePR(ctx context.Context, repo string, req PRRequest) (*PR, error) {
	// Initialize audit logger for security event tracking
	auditLogger := logging.NewAuditLogger()

	// Create PR using gh api
	prData := map[string]interface{}{
		"title": req.Title,
		"body":  req.Body,
		"head":  req.Head,
		"base":  req.Base,
	}

	jsonData, err := json.Marshal(prData)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "marshal PR data")
	}

	output, err := g.runner.RunWithInput(ctx, jsonData, "gh", "api", fmt.Sprintf("repos/%s/pulls", repo), "--method", "POST", "--input", "-")
	if err != nil {
		// Log failed repository access
		auditLogger.LogRepositoryAccess("github_cli", repo, "pr_create_failed")
		return nil, appErrors.WrapWithContext(err, "create PR")
	}

	var pr PR
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, appErrors.WrapWithContext(err, "parse PR response")
	}

	// Log successful repository access for PR creation
	auditLogger.LogRepositoryAccess("github_cli", repo, "pr_create")

	return &pr, nil
}

// GetPR retrieves a pull request by number
func (g *githubClient) GetPR(ctx context.Context, repo string, number int) (*PR, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/pulls/%d", repo, number))
	if err != nil {
		if isNotFoundError(err) {
			return nil, ErrPRNotFound
		}
		return nil, appErrors.WrapWithContext(err, "get PR")
	}

	var pr PR
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, appErrors.WrapWithContext(err, "parse PR")
	}

	return &pr, nil
}

// ListPRs lists pull requests for a repository
func (g *githubClient) ListPRs(ctx context.Context, repo, state string) ([]PR, error) {
	args := []string{"api", fmt.Sprintf("repos/%s/pulls", repo), "--paginate"}
	if state != "" && state != "all" {
		args = append(args, "-f", fmt.Sprintf("state=%s", state))
	}

	output, err := g.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "list PRs")
	}

	var prs []PR
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, appErrors.WrapWithContext(err, "parse PRs")
	}

	return prs, nil
}

// GetFile retrieves file contents from a repository
func (g *githubClient) GetFile(ctx context.Context, repo, path, ref string) (*FileContent, error) {
	url := fmt.Sprintf("repos/%s/contents/%s", repo, path)
	if ref != "" {
		url += fmt.Sprintf("?ref=%s", ref)
	}

	output, err := g.runner.Run(ctx, "gh", "api", url)
	if err != nil {
		if isNotFoundError(err) {
			return nil, ErrFileNotFound
		}
		return nil, appErrors.WrapWithContext(err, "get file")
	}

	var file File
	if unmarshalErr := json.Unmarshal(output, &file); unmarshalErr != nil {
		return nil, appErrors.WrapWithContext(unmarshalErr, "parse file")
	}

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(file.Content))
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "decode file content")
	}

	return &FileContent{
		Path:    file.Path,
		Content: content,
		SHA:     file.SHA,
	}, nil
}

// GetCommit retrieves commit details
func (g *githubClient) GetCommit(ctx context.Context, repo, sha string) (*Commit, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/commits/%s", repo, sha))
	if err != nil {
		if isNotFoundError(err) {
			return nil, ErrCommitNotFound
		}
		return nil, appErrors.WrapWithContext(err, "get commit")
	}

	var commit Commit
	if err := json.Unmarshal(output, &commit); err != nil {
		return nil, appErrors.WrapWithContext(err, "parse commit")
	}

	return &commit, nil
}

// isNotFoundError checks if the error is a 404 from GitHub API
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common 404 patterns in error messages
	errStr := err.Error()
	return strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "Not Found") ||
		strings.Contains(errStr, "could not resolve")
}

// NewClientWithRunner creates a GitHub client with a custom command runner (for testing)
func NewClientWithRunner(runner CommandRunner, logger *logrus.Logger) Client {
	return &githubClient{
		runner: runner,
		logger: logger,
	}
}
