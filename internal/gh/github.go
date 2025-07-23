package gh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

// Common errors
var (
	ErrNotAuthenticated = errors.New("gh CLI not authenticated")
	ErrGHNotFound       = errors.New("gh CLI not found in PATH")
	ErrRateLimited      = errors.New("GitHub API rate limit exceeded")
)

// githubClient implements the Client interface using gh CLI
type githubClient struct {
	runner CommandRunner
	logger *logrus.Logger
}

// NewClient creates a new GitHub client using gh CLI
func NewClient(logger *logrus.Logger) (Client, error) {
	// Check if gh is available
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, ErrGHNotFound
	}

	runner := NewCommandRunner(logger)

	// Check authentication status
	ctx := context.Background()
	if _, err := runner.Run(ctx, "gh", "auth", "status"); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotAuthenticated, err)
	}

	return &githubClient{
		runner: runner,
		logger: logger,
	}, nil
}

// ListBranches returns all branches for a repository
func (g *githubClient) ListBranches(ctx context.Context, repo string) ([]Branch, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/branches", repo), "--paginate")
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	var branches []Branch
	if err := json.Unmarshal(output, &branches); err != nil {
		return nil, fmt.Errorf("failed to parse branches: %w", err)
	}

	return branches, nil
}

// GetBranch returns details for a specific branch
func (g *githubClient) GetBranch(ctx context.Context, repo, branch string) (*Branch, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/branches/%s", repo, branch))
	if err != nil {
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}

	var b Branch
	if err := json.Unmarshal(output, &b); err != nil {
		return nil, fmt.Errorf("failed to parse branch: %w", err)
	}

	return &b, nil
}

// CreatePR creates a new pull request
func (g *githubClient) CreatePR(ctx context.Context, repo string, req PRRequest) (*PR, error) {
	// Create PR using gh api
	prData := map[string]interface{}{
		"title": req.Title,
		"body":  req.Body,
		"head":  req.Head,
		"base":  req.Base,
	}

	jsonData, err := json.Marshal(prData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PR data: %w", err)
	}

	output, err := g.runner.RunWithInput(ctx, jsonData, "gh", "api", fmt.Sprintf("repos/%s/pulls", repo), "--method", "POST", "--input", "-")
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	var pr PR
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR response: %w", err)
	}

	return &pr, nil
}

// GetPR retrieves a pull request by number
func (g *githubClient) GetPR(ctx context.Context, repo string, number int) (*PR, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/pulls/%d", repo, number))
	if err != nil {
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	var pr PR
	if err := json.Unmarshal(output, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR: %w", err)
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
		return nil, fmt.Errorf("failed to list PRs: %w", err)
	}

	var prs []PR
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse PRs: %w", err)
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
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	var file File
	if err := json.Unmarshal(output, &file); err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(file.Content))
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
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
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	var commit Commit
	if err := json.Unmarshal(output, &commit); err != nil {
		return nil, fmt.Errorf("failed to parse commit: %w", err)
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

// isRateLimitError checks if the error is due to rate limiting
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "403") && strings.Contains(errStr, "API rate limit exceeded")
}

// NewClientWithRunner creates a GitHub client with a custom command runner (for testing)
func NewClientWithRunner(runner CommandRunner, logger *logrus.Logger) Client {
	return &githubClient{
		runner: runner,
		logger: logger,
	}
}

