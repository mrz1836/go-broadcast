// Package github provides GitHub API integration for coverage reporting
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Static error definitions
var (
	ErrGitHubAPIError   = errors.New("GitHub API error")
	ErrCommentNotFound  = errors.New("coverage comment not found")
	ErrWorkflowNotFound = errors.New("workflow not found")
)

// Client handles GitHub API operations for coverage reporting
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
	config     *Config
}

// Config holds GitHub client configuration
type Config struct {
	Token      string        // GitHub API token
	BaseURL    string        // GitHub API base URL
	Timeout    time.Duration // Request timeout
	RetryCount int           // Number of retries
	UserAgent  string        // User agent string
}

// CommentRequest represents a PR comment request
type CommentRequest struct {
	Body string `json:"body"`
}

// StatusRequest represents a commit status request
type StatusRequest struct {
	State       string `json:"state"`       // "error", "failure", "pending", "success"
	TargetURL   string `json:"target_url"`  // URL to details
	Description string `json:"description"` // Short description
	Context     string `json:"context"`     // Unique context identifier
}

// Comment represents a GitHub PR comment
type Comment struct {
	ID        int    `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Head   struct {
		SHA string `json:"sha"`
	} `json:"head"`
	Labels []Label `json:"labels"`
}

// Label represents a GitHub label
type Label struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID               int64     `json:"id"`
	NodeID           string    `json:"node_id"`
	Name             string    `json:"name"`
	HeadBranch       string    `json:"head_branch"`
	HeadSHA          string    `json:"head_sha"`
	Path             string    `json:"path"`
	DisplayTitle     string    `json:"display_title"`
	RunNumber        int       `json:"run_number"`
	Event            string    `json:"event"`
	Status           string    `json:"status"`     // "queued", "in_progress", "completed"
	Conclusion       string    `json:"conclusion"` // "success", "failure", "neutral", "canceled", "skipped", "timed_out", "action_required"
	WorkflowID       int64     `json:"workflow_id"`
	CheckSuiteID     int64     `json:"check_suite_id"`
	CheckSuiteNodeID string    `json:"check_suite_node_id"`
	URL              string    `json:"url"`
	HTMLURL          string    `json:"html_url"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	RunStartedAt     time.Time `json:"run_started_at"`
	JobsURL          string    `json:"jobs_url"`
	LogsURL          string    `json:"logs_url"`
	CheckSuiteURL    string    `json:"check_suite_url"`
	ArtifactsURL     string    `json:"artifacts_url"`
	CancelURL        string    `json:"cancel_url"`
	RerunURL         string    `json:"rerun_url"`
	WorkflowURL      string    `json:"workflow_url"`
}

// WorkflowRunsResponse represents the response from listing workflow runs
type WorkflowRunsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

// Workflow represents a GitHub Actions workflow
type Workflow struct {
	ID        int64     `json:"id"`
	NodeID    string    `json:"node_id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	URL       string    `json:"url"`
	HTMLURL   string    `json:"html_url"`
	BadgeURL  string    `json:"badge_url"`
}

// New creates a new GitHub client with default configuration
func New(token string) *Client {
	return &Client{
		token:   token,
		baseURL: "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: &Config{
			Token:      token,
			BaseURL:    "https://api.github.com",
			Timeout:    30 * time.Second,
			RetryCount: 3,
			UserAgent:  "coverage-system/1.0",
		},
	}
}

// NewWithConfig creates a new GitHub client with custom configuration
func NewWithConfig(config *Config) *Client {
	return &Client{
		token:   config.Token,
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		config: config,
	}
}

// CreateComment creates or updates a PR comment with coverage information
func (c *Client) CreateComment(ctx context.Context, owner, repo string, pr int, body string) (*Comment, error) {
	// First, try to find existing coverage comment
	existing, err := c.findCoverageComment(ctx, owner, repo, pr)
	if err != nil && !errors.Is(err, ErrCommentNotFound) {
		return nil, fmt.Errorf("failed to find existing comment: %w", err)
	}

	if existing != nil {
		// Update existing comment
		return c.updateComment(ctx, owner, repo, existing.ID, body)
	}

	// Create new comment
	return c.createComment(ctx, owner, repo, pr, body)
}

// CreateStatus creates a commit status for coverage
func (c *Client) CreateStatus(ctx context.Context, owner, repo, sha string, status *StatusRequest) error {
	url := fmt.Sprintf("%s/repos/%s/%s/statuses/%s", c.baseURL, owner, repo, sha)

	jsonData, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create status: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: %d %s", ErrGitHubAPIError, resp.StatusCode, string(body))
	}

	return nil
}

// GetPullRequest retrieves PR information
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, pr int) (*PullRequest, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.baseURL, owner, repo, pr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %d %s", ErrGitHubAPIError, resp.StatusCode, string(body))
	}

	var pullRequest PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pullRequest); err != nil {
		return nil, fmt.Errorf("failed to decode PR: %w", err)
	}

	return &pullRequest, nil
}

// Helper methods

func (c *Client) findCoverageComment(ctx context.Context, owner, repo string, pr int) (*Comment, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, pr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %d %s", ErrGitHubAPIError, resp.StatusCode, string(body))
	}

	var comments []Comment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, fmt.Errorf("failed to decode comments: %w", err)
	}

	// Look for existing coverage comment
	for _, comment := range comments {
		if containsCoverageMarker(comment.Body) {
			return &comment, nil
		}
	}

	return nil, ErrCommentNotFound
}

func (c *Client) createComment(ctx context.Context, owner, repo string, pr int, body string) (*Comment, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, pr)

	commentReq := CommentRequest{Body: body}
	jsonData, err := json.Marshal(commentReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal comment: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %d %s", ErrGitHubAPIError, resp.StatusCode, string(body))
	}

	var comment Comment
	if err := json.NewDecoder(resp.Body).Decode(&comment); err != nil {
		return nil, fmt.Errorf("failed to decode comment: %w", err)
	}

	return &comment, nil
}

func (c *Client) updateComment(ctx context.Context, owner, repo string, commentID int, body string) (*Comment, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/comments/%d", c.baseURL, owner, repo, commentID)

	commentReq := CommentRequest{Body: body}
	jsonData, err := json.Marshal(commentReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal comment: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %d %s", ErrGitHubAPIError, resp.StatusCode, string(body))
	}

	var comment Comment
	if err := json.NewDecoder(resp.Body).Decode(&comment); err != nil {
		return nil, fmt.Errorf("failed to decode comment: %w", err)
	}

	return &comment, nil
}

func containsCoverageMarker(body string) bool {
	// Look for coverage report markers
	markers := []string{
		"## Coverage Report",
		"<!-- coverage-comment -->",
		"📊 **Coverage**",
	}

	for _, marker := range markers {
		if contains(body, marker) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GenerateCoverageComment generates a formatted coverage comment for PR
func (c *Client) GenerateCoverageComment(percentage float64, trend string, badgeURL string) string {
	var trendIcon string
	var trendText string

	switch trend {
	case "up":
		trendIcon = "📈"
		trendText = "Coverage increased"
	case "down":
		trendIcon = "📉"
		trendText = "Coverage decreased"
	default:
		trendIcon = "📊"
		trendText = "Coverage stable"
	}

	comment := fmt.Sprintf(`<!-- coverage-comment -->
## %s Coverage Report

**Overall Coverage: %.1f%%** %s

%s %s

![Coverage Badge](%s)

---
*Generated by GoFortress Coverage System 🤖*
`, trendIcon, percentage, getPercentageEmoji(percentage), trendIcon, trendText, badgeURL)

	return comment
}

func getPercentageEmoji(percentage float64) string {
	switch {
	case percentage >= 90:
		return "🟢"
	case percentage >= 80:
		return "🟡"
	case percentage >= 70:
		return "🟠"
	default:
		return "🔴"
	}
}

// Status constants for commit statuses
const (
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusError   = "error"
	StatusPending = "pending"
)

// Coverage status contexts
const (
	ContextCoverage = "coverage/total"
	ContextTrend    = "coverage/trend"
)

// GetWorkflowRuns retrieves the latest workflow runs for a repository
func (c *Client) GetWorkflowRuns(ctx context.Context, owner, repo string, limit int) (*WorkflowRunsResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs", c.baseURL, owner, repo)
	if limit > 0 {
		url += fmt.Sprintf("?per_page=%d", limit)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow runs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %d %s", ErrGitHubAPIError, resp.StatusCode, string(body))
	}

	var response WorkflowRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode workflow runs response: %w", err)
	}

	return &response, nil
}

// GetWorkflowRunsByWorkflow retrieves workflow runs for a specific workflow
func (c *Client) GetWorkflowRunsByWorkflow(ctx context.Context, owner, repo, workflowName string, limit int) (*WorkflowRunsResponse, error) {
	// First, get the workflow ID by name
	workflowID, err := c.getWorkflowIDByName(ctx, owner, repo, workflowName)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow ID: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/actions/workflows/%d/runs", c.baseURL, owner, repo, workflowID)
	if limit > 0 {
		url += fmt.Sprintf("?per_page=%d", limit)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow runs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %d %s", ErrGitHubAPIError, resp.StatusCode, string(body))
	}

	var response WorkflowRunsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode workflow runs response: %w", err)
	}

	return &response, nil
}

// GetWorkflowRun retrieves a specific workflow run by ID
func (c *Client) GetWorkflowRun(ctx context.Context, owner, repo string, runID int64) (*WorkflowRun, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs/%d", c.baseURL, owner, repo, runID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow run: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %d %s", ErrGitHubAPIError, resp.StatusCode, string(body))
	}

	var workflowRun WorkflowRun
	if err := json.NewDecoder(resp.Body).Decode(&workflowRun); err != nil {
		return nil, fmt.Errorf("failed to decode workflow run response: %w", err)
	}

	return &workflowRun, nil
}

// getWorkflowIDByName finds a workflow ID by its name
func (c *Client) getWorkflowIDByName(ctx context.Context, owner, repo, workflowName string) (int64, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/workflows", c.baseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.config.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get workflows: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("%w: %d %s", ErrGitHubAPIError, resp.StatusCode, string(body))
	}

	var response struct {
		TotalCount int        `json:"total_count"`
		Workflows  []Workflow `json:"workflows"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf("failed to decode workflows response: %w", err)
	}

	for _, workflow := range response.Workflows {
		if workflow.Name == workflowName {
			return workflow.ID, nil
		}
	}

	return 0, fmt.Errorf("%w: %s", ErrWorkflowNotFound, workflowName)
}
