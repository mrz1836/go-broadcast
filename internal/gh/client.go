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
}
