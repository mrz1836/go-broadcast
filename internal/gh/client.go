package gh

import "context"

// Client defines the interface for GitHub operations
type Client interface {
	// ListBranches returns all branches for a repository
	ListBranches(ctx context.Context, repo string) ([]Branch, error)

	// GetBranch returns details for a specific branch
	GetBranch(ctx context.Context, repo, branch string) (*Branch, error)

	// CreatePR creates a new pull request
	CreatePR(ctx context.Context, repo string, req PRRequest) (*PR, error)

	// GetPR retrieves a pull request by number
	GetPR(ctx context.Context, repo string, number int) (*PR, error)

	// ListPRs lists pull requests for a repository
	// Can filter by state (open, closed, all)
	ListPRs(ctx context.Context, repo, state string) ([]PR, error)

	// GetFile retrieves file contents from a repository
	// ref can be a branch name, tag, or commit SHA
	GetFile(ctx context.Context, repo, path, ref string) (*FileContent, error)

	// GetCommit retrieves commit details
	GetCommit(ctx context.Context, repo, sha string) (*Commit, error)

	// ClosePR closes a pull request with an optional comment
	ClosePR(ctx context.Context, repo string, number int, comment string) error

	// DeleteBranch deletes a branch from the repository
	DeleteBranch(ctx context.Context, repo, branch string) error

	// UpdatePR updates a pull request (e.g., to add comments)
	UpdatePR(ctx context.Context, repo string, number int, updates PRUpdate) error

	// GetCurrentUser returns the authenticated user
	GetCurrentUser(ctx context.Context) (*User, error)

	// GetGitTree retrieves the Git tree for a repository
	// recursive=true will fetch all files in the repository
	GetGitTree(ctx context.Context, repo, treeSHA string, recursive bool) (*GitTree, error)

	// GetRepository retrieves repository details including merge settings
	GetRepository(ctx context.Context, repo string) (*Repository, error)

	// ReviewPR submits an approving review for a pull request
	ReviewPR(ctx context.Context, repo string, number int, message string) error

	// MergePR merges a pull request using the specified method
	MergePR(ctx context.Context, repo string, number int, method MergeMethod) error

	// BypassMergePR merges a pull request using admin privileges to bypass branch protection
	// This requires the user to have admin access to the repository
	BypassMergePR(ctx context.Context, repo string, number int, method MergeMethod) error

	// EnableAutoMergePR enables auto-merge for a pull request
	// This allows the PR to merge automatically when all required checks pass
	EnableAutoMergePR(ctx context.Context, repo string, number int, method MergeMethod) error

	// SearchAssignedPRs searches for all open, non-draft pull requests assigned to the current user
	SearchAssignedPRs(ctx context.Context) ([]PR, error)

	// GetPRReviews retrieves all reviews for a pull request
	GetPRReviews(ctx context.Context, repo string, number int) ([]Review, error)

	// HasApprovedReview checks if a specific user has already submitted an approving review for a PR
	HasApprovedReview(ctx context.Context, repo string, number int, username string) (bool, error)

	// AddPRComment adds a comment to a pull request (for cases where a review cannot be submitted)
	AddPRComment(ctx context.Context, repo string, number int, comment string) error

	// GetPRCheckStatus retrieves the status of all check runs for a PR's head commit
	// Returns a summary of check statuses including running, passed, failed, and skipped counts
	GetPRCheckStatus(ctx context.Context, repo string, number int) (*CheckStatusSummary, error)

	// DiscoverOrgRepos returns all repositories for an organization
	// Uses REST API with pagination to fetch all repos
	DiscoverOrgRepos(ctx context.Context, org string) ([]RepoInfo, error)

	// ExecuteGraphQL executes a GraphQL query and returns the raw response data
	ExecuteGraphQL(ctx context.Context, query string) (map[string]interface{}, error)

	// GetDependabotAlerts retrieves Dependabot security alerts for a repository
	// Returns empty slice if Dependabot is not enabled (404 response)
	GetDependabotAlerts(ctx context.Context, repo string) ([]DependabotAlert, error)

	// GetCodeScanningAlerts retrieves code scanning alerts for a repository
	// Returns empty slice if code scanning is not enabled (404 response)
	GetCodeScanningAlerts(ctx context.Context, repo string) ([]CodeScanningAlert, error)

	// GetSecretScanningAlerts retrieves secret scanning alerts for a repository
	// Returns empty slice if secret scanning is not enabled (404 response)
	GetSecretScanningAlerts(ctx context.Context, repo string) ([]SecretScanningAlert, error)

	// ListWorkflows returns all workflows for a repository
	ListWorkflows(ctx context.Context, repo string) ([]Workflow, error)

	// GetWorkflowRuns returns recent runs for a specific workflow
	GetWorkflowRuns(ctx context.Context, repo string, workflowID int64, count int) ([]WorkflowRun, error)

	// GetRunArtifacts returns all artifacts for a workflow run
	GetRunArtifacts(ctx context.Context, repo string, runID int64) ([]Artifact, error)

	// DownloadRunArtifact downloads a named artifact from a workflow run to the specified directory
	DownloadRunArtifact(ctx context.Context, repo string, runID int64, artifactName, destDir string) error
}
