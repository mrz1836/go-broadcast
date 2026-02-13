package gh

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/jsonutil"
	"github.com/mrz1836/go-broadcast/internal/logging"
)

// Common errors
var (
	ErrNotAuthenticated       = errors.New("gh CLI not authenticated")
	ErrGHNotFound             = errors.New("gh CLI not found in PATH")
	ErrRateLimited            = errors.New("GitHub API rate limit exceeded")
	ErrBranchNotFound         = errors.New("branch not found")
	ErrPRNotFound             = errors.New("pull request not found")
	ErrPRValidationFailed     = errors.New("PR validation failed - branch may already have PR or conflict exists")
	ErrFileNotFound           = errors.New("file not found")
	ErrCommitNotFound         = errors.New("commit not found")
	ErrGitTreeNotFound        = errors.New("git tree not found")
	ErrRepositoryNotFound     = errors.New("repository not found")
	ErrUnsupportedMergeMethod = errors.New("unsupported merge method")
	ErrOrganizationNotFound   = errors.New("organization not found")
	ErrGraphQLError           = errors.New("GraphQL query failed")
)

// githubClient implements the Client interface using gh CLI
type githubClient struct {
	runner      CommandRunner
	logger      *logrus.Logger
	currentUser *User        // Cache for current user
	mu          sync.RWMutex // Protects currentUser
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
		runner:      runner,
		logger:      logger,
		currentUser: nil,
	}, nil
}

// ListBranches returns all branches for a repository
func (g *githubClient) ListBranches(ctx context.Context, repo string) ([]Branch, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/branches", repo), "--paginate")
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "list branches")
	}

	branches, err := jsonutil.UnmarshalJSON[[]Branch](output)
	if err != nil {
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

	b, err := jsonutil.UnmarshalJSON[Branch](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse branch")
	}

	return &b, nil
}

// CreatePR creates a new pull request
func (g *githubClient) CreatePR(ctx context.Context, repo string, req PRRequest) (*PR, error) {
	// Initialize audit logger for security event tracking
	auditLogger := logging.NewAuditLogger()

	// Extract owner from repo (format: owner/repo)
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return nil, appErrors.WrapWithContext(appErrors.FormatError("repository", repo, "owner/repo"), "parse repo")
	}
	owner := parts[0]

	// Format head branch with owner prefix for cross-repository PRs
	headRef := req.Head
	if !strings.Contains(headRef, ":") {
		headRef = fmt.Sprintf("%s:%s", owner, req.Head)
	}

	// Create PR using gh api
	prData := map[string]interface{}{
		"title": req.Title,
		"body":  req.Body,
		"head":  headRef,
		"base":  req.Base,
	}

	jsonData, err := jsonutil.MarshalJSON(prData)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "marshal PR data")
	}

	output, err := g.runner.RunWithInput(ctx, jsonData, "gh", "api", fmt.Sprintf("repos/%s/pulls", repo), "--method", "POST", "--input", "-")
	if err != nil {
		// Log failed repository access
		auditLogger.LogRepositoryAccess("github_cli", repo, "pr_create_failed")

		// Handle validation failures (HTTP 422) which commonly occur when:
		// - A PR already exists for the branch
		// - The branch doesn't exist
		// - There are conflicts or validation issues
		if isValidationFailedError(err) {
			return nil, appErrors.WrapWithContext(ErrPRValidationFailed, fmt.Sprintf("failed to create PR with head '%s' and base '%s': %v", headRef, req.Base, err))
		}

		return nil, appErrors.WrapWithContext(fmt.Errorf("failed to create PR with head '%s' and base '%s': %w", headRef, req.Base, err), "create PR")
	}

	pr, err := jsonutil.UnmarshalJSON[PR](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse PR response")
	}

	// Log successful repository access for PR creation
	auditLogger.LogRepositoryAccess("github_cli", repo, "pr_create")

	// Set assignees if provided
	if len(req.Assignees) > 0 {
		if err := g.setAssignees(ctx, repo, pr.Number, req.Assignees); err != nil {
			if g.logger != nil {
				g.logger.WithError(err).Warn("Failed to set PR assignees")
			}
		}
	}

	// Set reviewers if provided
	if len(req.Reviewers) > 0 || len(req.TeamReviewers) > 0 {
		if err := g.setReviewers(ctx, repo, pr.Number, req.Reviewers, req.TeamReviewers); err != nil {
			if g.logger != nil {
				g.logger.WithError(err).Warn("Failed to set PR reviewers")
			}
		}
	}

	// Set labels if provided
	if len(req.Labels) > 0 {
		if err := g.setLabels(ctx, repo, pr.Number, req.Labels); err != nil {
			if g.logger != nil {
				g.logger.WithError(err).Warn("Failed to set PR labels")
			}
		}
	}

	return &pr, nil
}

// setAssignees sets assignees for a pull request
func (g *githubClient) setAssignees(ctx context.Context, repo string, prNumber int, assignees []string) error {
	assigneeData := map[string]interface{}{
		"assignees": assignees,
	}

	jsonData, err := jsonutil.MarshalJSON(assigneeData)
	if err != nil {
		return appErrors.WrapWithContext(err, "marshal assignee data")
	}

	_, err = g.runner.RunWithInput(ctx, jsonData, "gh", "api", fmt.Sprintf("repos/%s/issues/%d/assignees", repo, prNumber), "--method", "POST", "--input", "-")
	if err != nil {
		return appErrors.WrapWithContext(err, "set PR assignees")
	}

	return nil
}

// setReviewers sets reviewers and team reviewers for a pull request
func (g *githubClient) setReviewers(ctx context.Context, repo string, prNumber int, reviewers, teamReviewers []string) error {
	reviewerData := map[string]interface{}{}

	if len(reviewers) > 0 {
		reviewerData["reviewers"] = reviewers
	}

	if len(teamReviewers) > 0 {
		reviewerData["team_reviewers"] = teamReviewers
	}

	jsonData, err := jsonutil.MarshalJSON(reviewerData)
	if err != nil {
		return appErrors.WrapWithContext(err, "marshal reviewer data")
	}

	_, err = g.runner.RunWithInput(ctx, jsonData, "gh", "api", fmt.Sprintf("repos/%s/pulls/%d/requested_reviewers", repo, prNumber), "--method", "POST", "--input", "-")
	if err != nil {
		return appErrors.WrapWithContext(err, "set PR reviewers")
	}

	return nil
}

// setLabels sets labels for a pull request
func (g *githubClient) setLabels(ctx context.Context, repo string, prNumber int, labels []string) error {
	labelData := map[string]interface{}{
		"labels": labels,
	}

	jsonData, err := jsonutil.MarshalJSON(labelData)
	if err != nil {
		return appErrors.WrapWithContext(err, "marshal label data")
	}

	_, err = g.runner.RunWithInput(ctx, jsonData, "gh", "api", fmt.Sprintf("repos/%s/issues/%d/labels", repo, prNumber), "--method", "POST", "--input", "-")
	if err != nil {
		return appErrors.WrapWithContext(err, "set PR labels")
	}

	return nil
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

	pr, err := jsonutil.UnmarshalJSON[PR](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse PR")
	}

	return &pr, nil
}

// ListPRs lists pull requests for a repository
func (g *githubClient) ListPRs(ctx context.Context, repo, state string) ([]PR, error) {
	apiURL := fmt.Sprintf("repos/%s/pulls", repo)
	if state != "" && state != "all" {
		apiURL += fmt.Sprintf("?state=%s", state)
	}

	args := []string{"api", apiURL, "--paginate"}

	output, err := g.runner.Run(ctx, "gh", args...)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "list PRs")
	}

	prs, err := jsonutil.UnmarshalJSON[[]PR](output)
	if err != nil {
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

	file, unmarshalErr := jsonutil.UnmarshalJSON[File](output)
	if unmarshalErr != nil {
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

	commit, err := jsonutil.UnmarshalJSON[Commit](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse commit")
	}

	return &commit, nil
}

// ClosePR closes a pull request with an optional comment
func (g *githubClient) ClosePR(ctx context.Context, repo string, number int, comment string) error {
	// First, add a comment if provided
	if comment != "" {
		if err := g.AddPRComment(ctx, repo, number, comment); err != nil {
			if g.logger != nil {
				g.logger.WithError(err).Warn("Failed to add comment before closing PR")
			}
		}
	}

	// Close the PR by updating its state
	closed := "closed"
	updates := PRUpdate{
		State: &closed,
	}

	return g.UpdatePR(ctx, repo, number, updates)
}

// DeleteBranch deletes a branch from the repository
func (g *githubClient) DeleteBranch(ctx context.Context, repo, branch string) error {
	_, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/git/refs/heads/%s", repo, branch), "--method", "DELETE")
	if err != nil {
		if isNotFoundError(err) {
			return ErrBranchNotFound
		}
		return appErrors.WrapWithContext(err, "delete branch")
	}

	return nil
}

// UpdatePR updates a pull request
func (g *githubClient) UpdatePR(ctx context.Context, repo string, number int, updates PRUpdate) error {
	jsonData, err := jsonutil.MarshalJSON(updates)
	if err != nil {
		return appErrors.WrapWithContext(err, "marshal PR update")
	}

	_, err = g.runner.RunWithInput(ctx, jsonData, "gh", "api", fmt.Sprintf("repos/%s/pulls/%d", repo, number), "--method", "PATCH", "--input", "-")
	if err != nil {
		if isNotFoundError(err) {
			return ErrPRNotFound
		}
		return appErrors.WrapWithContext(err, "update PR")
	}

	return nil
}

// AddPRComment adds a comment to a pull request
func (g *githubClient) AddPRComment(ctx context.Context, repo string, number int, comment string) error {
	commentData := map[string]string{
		"body": comment,
	}

	jsonData, err := jsonutil.MarshalJSON(commentData)
	if err != nil {
		return appErrors.WrapWithContext(err, "marshal comment")
	}

	_, err = g.runner.RunWithInput(ctx, jsonData, "gh", "api", fmt.Sprintf("repos/%s/issues/%d/comments", repo, number), "--method", "POST", "--input", "-")
	if err != nil {
		return appErrors.WrapWithContext(err, "add PR comment")
	}

	return nil
}

// GetCurrentUser returns the authenticated user
func (g *githubClient) GetCurrentUser(ctx context.Context) (*User, error) {
	// Check cache with read lock
	g.mu.RLock()
	if g.currentUser != nil {
		user := g.currentUser
		g.mu.RUnlock()
		return user, nil
	}
	g.mu.RUnlock()

	output, err := g.runner.Run(ctx, "gh", "api", "user")
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "get current user")
	}

	user, err := jsonutil.UnmarshalJSON[User](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse user")
	}

	// Cache the user with write lock
	g.mu.Lock()
	g.currentUser = &user
	g.mu.Unlock()

	return &user, nil
}

// GetGitTree retrieves the Git tree for a repository
func (g *githubClient) GetGitTree(ctx context.Context, repo, treeSHA string, recursive bool) (*GitTree, error) {
	apiURL := fmt.Sprintf("repos/%s/git/trees/%s", repo, treeSHA)
	if recursive {
		apiURL += "?recursive=1"
	}

	output, err := g.runner.Run(ctx, "gh", "api", apiURL)
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("%w: %s", ErrGitTreeNotFound, treeSHA)
		}
		return nil, appErrors.WrapWithContext(err, "get git tree")
	}

	gitTree, err := jsonutil.UnmarshalJSON[GitTree](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse git tree")
	}

	return &gitTree, nil
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

// isValidationFailedError checks if the error is a 422 (validation failed) from GitHub API
func isValidationFailedError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common 422 patterns in error messages
	errStr := err.Error()
	return strings.Contains(errStr, "422") ||
		strings.Contains(errStr, "Validation Failed") ||
		strings.Contains(errStr, "Unprocessable Entity")
}

// IsBranchProtectionError checks if the error is due to branch protection policies
// blocking the merge. These errors indicate that auto-merge should be attempted instead.
func IsBranchProtectionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for patterns indicating branch protection is blocking the merge
	errStr := err.Error()
	return strings.Contains(errStr, "base branch policy prohibits the merge") ||
		strings.Contains(errStr, "add the `--auto` flag") ||
		strings.Contains(errStr, "not mergeable: the base branch policy") ||
		strings.Contains(errStr, "required status checks") ||
		strings.Contains(errStr, "required reviews")
}

// NewClientWithRunner creates a GitHub client with a custom command runner (for testing)
func NewClientWithRunner(runner CommandRunner, logger *logrus.Logger) Client {
	return &githubClient{
		runner:      runner,
		logger:      logger,
		currentUser: nil,
	}
}

// GetRepository retrieves repository details including merge settings
func (g *githubClient) GetRepository(ctx context.Context, repo string) (*Repository, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s", repo))
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("%w: %s", ErrRepositoryNotFound, repo)
		}
		return nil, appErrors.WrapWithContext(err, "get repository")
	}

	repository, err := jsonutil.UnmarshalJSON[Repository](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse repository")
	}

	return &repository, nil
}

// ReviewPR submits an approving review for a pull request
func (g *githubClient) ReviewPR(ctx context.Context, repo string, number int, message string) error {
	// Use gh pr review command for approving
	args := []string{"pr", "review", fmt.Sprintf("%d", number), "--repo", repo, "--approve"}

	// Add message/body if provided
	if message != "" {
		args = append(args, "--body", message)
	}

	_, err := g.runner.Run(ctx, "gh", args...)
	if err != nil {
		if isNotFoundError(err) {
			return ErrPRNotFound
		}
		return appErrors.WrapWithContext(err, fmt.Sprintf("review PR #%d", number))
	}

	return nil
}

// MergePR merges a pull request using the specified method
func (g *githubClient) MergePR(ctx context.Context, repo string, number int, method MergeMethod) error {
	// Build merge command
	args := []string{"pr", "merge", fmt.Sprintf("%d", number), "--repo", repo}

	// Add merge method flag
	switch method {
	case MergeMethodSquash:
		args = append(args, "--squash")
	case MergeMethodRebase:
		args = append(args, "--rebase")
	case MergeMethodMerge:
		args = append(args, "--merge")
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedMergeMethod, method)
	}

	_, err := g.runner.Run(ctx, "gh", args...)
	if err != nil {
		if isNotFoundError(err) {
			return ErrPRNotFound
		}
		return appErrors.WrapWithContext(err, fmt.Sprintf("merge PR #%d with method %s", number, method))
	}

	return nil
}

// BypassMergePR merges a pull request using admin privileges to bypass branch protection
func (g *githubClient) BypassMergePR(ctx context.Context, repo string, number int, method MergeMethod) error {
	// Build merge command with --admin flag
	args := []string{"pr", "merge", fmt.Sprintf("%d", number), "--repo", repo, "--admin"}

	// Add merge method flag
	switch method {
	case MergeMethodSquash:
		args = append(args, "--squash")
	case MergeMethodRebase:
		args = append(args, "--rebase")
	case MergeMethodMerge:
		args = append(args, "--merge")
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedMergeMethod, method)
	}

	_, err := g.runner.Run(ctx, "gh", args...)
	if err != nil {
		if isNotFoundError(err) {
			return ErrPRNotFound
		}
		return appErrors.WrapWithContext(err, fmt.Sprintf("bypass merge PR #%d with method %s", number, method))
	}

	return nil
}

// EnableAutoMergePR enables auto-merge for a pull request
func (g *githubClient) EnableAutoMergePR(ctx context.Context, repo string, number int, method MergeMethod) error {
	// Build auto-merge command
	args := []string{"pr", "merge", fmt.Sprintf("%d", number), "--repo", repo, "--auto"}

	// Add merge method flag
	switch method {
	case MergeMethodSquash:
		args = append(args, "--squash")
	case MergeMethodRebase:
		args = append(args, "--rebase")
	case MergeMethodMerge:
		args = append(args, "--merge")
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedMergeMethod, method)
	}

	_, err := g.runner.Run(ctx, "gh", args...)
	if err != nil {
		if isNotFoundError(err) {
			return ErrPRNotFound
		}
		return appErrors.WrapWithContext(err, fmt.Sprintf("enable auto-merge for PR #%d with method %s", number, method))
	}

	return nil
}

// SearchAssignedPRs searches for all open, non-draft pull requests assigned to the current user
func (g *githubClient) SearchAssignedPRs(ctx context.Context) ([]PR, error) {
	// Use GitHub search API to find PRs assigned to current user
	// The gh search prs command has limited fields compared to gh pr view
	// We only need basic info here - full details will be fetched when processing each PR
	output, err := g.runner.Run(ctx, "gh", "search", "prs", "--assignee", "@me", "--state", "open", "--limit", "1000", "--json", "number,title,url,isDraft,state")
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "search assigned PRs")
	}

	// Parse the search results - only fields available from gh search prs
	type searchResult struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		URL     string `json:"url"`
		State   string `json:"state"`
		IsDraft bool   `json:"isDraft"`
	}

	results, err := jsonutil.UnmarshalJSON[[]searchResult](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse search results")
	}

	// Convert search results to PR objects and filter out drafts
	prs := make([]PR, 0, len(results))
	for _, result := range results {
		// Skip draft PRs
		if result.IsDraft {
			continue
		}

		// Parse owner/repo from URL (format: https://github.com/owner/repo/pull/123)
		parts := strings.Split(result.URL, "/")
		if len(parts) < 7 {
			// Invalid URL format, log warning and skip this PR
			if g.logger != nil {
				g.logger.WithField("url", result.URL).Warn("Skipping PR with invalid URL format")
			}
			continue
		}
		owner := parts[3]
		repoName := parts[4]
		repo := fmt.Sprintf("%s/%s", owner, repoName)

		// Create minimal PR info - full details will be fetched later by GetPR when processing
		pr := PR{
			Number: result.Number,
			State:  strings.ToLower(result.State),
			Title:  result.Title,
			Draft:  result.IsDraft,
			Repo:   repo, // Store repo in dedicated field for cross-repository operations
		}

		prs = append(prs, pr)
	}

	return prs, nil
}

// GetPRReviews retrieves all reviews for a pull request
func (g *githubClient) GetPRReviews(ctx context.Context, repo string, number int) ([]Review, error) {
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/pulls/%d/reviews", repo, number))
	if err != nil {
		if isNotFoundError(err) {
			return nil, ErrPRNotFound
		}
		return nil, appErrors.WrapWithContext(err, fmt.Sprintf("get reviews for PR #%d", number))
	}

	reviews, err := jsonutil.UnmarshalJSON[[]Review](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse PR reviews")
	}

	return reviews, nil
}

// HasApprovedReview checks if a specific user has already submitted an approving review for a PR
func (g *githubClient) HasApprovedReview(ctx context.Context, repo string, number int, username string) (bool, error) {
	reviews, err := g.GetPRReviews(ctx, repo, number)
	if err != nil {
		return false, err
	}

	// Check if the user has submitted an "APPROVED" review
	// We only check the latest review from this user, as users can update their reviews
	userReviews := make(map[string]string) // username -> latest review state
	for _, review := range reviews {
		// Skip reviews with missing user data to avoid panic
		if review.User.Login == "" {
			continue
		}
		// Track the latest review state for each user
		userReviews[review.User.Login] = review.State
	}

	// Check if the specified user has an "APPROVED" review as their latest review
	if state, exists := userReviews[username]; exists && state == "APPROVED" {
		return true, nil
	}

	return false, nil
}

// GetPRCheckStatus retrieves the status of all check runs for a PR's head commit
func (g *githubClient) GetPRCheckStatus(ctx context.Context, repo string, number int) (*CheckStatusSummary, error) {
	// First, get the PR to obtain the head SHA
	pr, err := g.GetPR(ctx, repo, number)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, fmt.Sprintf("get PR #%d for check status", number))
	}

	// Get check runs for the head commit
	output, err := g.runner.Run(ctx, "gh", "api", fmt.Sprintf("repos/%s/commits/%s/check-runs", repo, pr.Head.SHA))
	if err != nil {
		return nil, appErrors.WrapWithContext(err, fmt.Sprintf("get check runs for PR #%d", number))
	}

	response, err := jsonutil.UnmarshalJSON[CheckRunsResponse](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse check runs response")
	}

	// Build the summary
	summary := &CheckStatusSummary{
		Total:  response.TotalCount,
		Checks: response.CheckRuns,
	}

	// Categorize each check run
	for _, check := range response.CheckRuns {
		switch check.Status {
		case "completed":
			summary.Completed++
			switch check.Conclusion {
			case "success", "neutral":
				summary.Passed++
			case "skipped":
				summary.Skipped++
			case "failure", "canceled", "timed_out", "action_required":
				summary.Failed++
			}
		case "queued", "in_progress":
			summary.Running++
		}
	}

	return summary, nil
}

// GraphQLResponse represents a GraphQL API response
type GraphQLResponse struct {
	Data   map[string]interface{} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// ExecuteGraphQL executes a GraphQL query and returns the raw response data
func (g *githubClient) ExecuteGraphQL(ctx context.Context, query string) (map[string]interface{}, error) {
	// Execute GraphQL query via gh api
	output, err := g.runner.RunWithInput(ctx, []byte(query), "gh", "api", "graphql", "-F", "query=@-")
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "execute GraphQL query")
	}

	// Parse response - GraphQL returns { "data": {...}, "errors": [...] }
	response, err := jsonutil.UnmarshalJSON[GraphQLResponse](output)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "parse GraphQL response")
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("%w: %v", ErrGraphQLError, response.Errors)
	}

	return response.Data, nil
}
