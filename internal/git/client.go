// Package git provides Git repository operations
package git

import "context"

// Client defines the interface for Git operations
type Client interface {
	// Clone clones a repository to the specified path
	Clone(ctx context.Context, url, path string) error

	// Checkout switches to the specified branch
	Checkout(ctx context.Context, repoPath, branch string) error

	// CreateBranch creates a new branch from the current HEAD
	CreateBranch(ctx context.Context, repoPath, branch string) error

	// Add stages files for commit. Paths are relative to repo root.
	// Use "." to stage all changes
	Add(ctx context.Context, repoPath string, paths ...string) error

	// Commit creates a commit with the specified message
	Commit(ctx context.Context, repoPath, message string) error

	// Push pushes the current branch to the remote
	// If force is true, uses --force flag
	Push(ctx context.Context, repoPath, remote, branch string, force bool) error

	// Diff returns the diff of staged changes
	Diff(ctx context.Context, repoPath string, staged bool) (string, error)

	// GetCurrentBranch returns the name of the current branch
	GetCurrentBranch(ctx context.Context, repoPath string) (string, error)

	// GetRemoteURL returns the URL of the specified remote
	GetRemoteURL(ctx context.Context, repoPath, remote string) (string, error)
}
