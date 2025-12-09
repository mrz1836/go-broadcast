// Package git provides Git repository operations
package git

import "context"

// RepositoryInfo contains information extracted from a Git repository
type RepositoryInfo struct {
	Name     string // Repository name (e.g., "go-broadcast")
	Owner    string // Repository owner (e.g., "mrz1836")
	FullName string // Full repository name (e.g., "mrz1836/go-broadcast")
	URL      string // Repository URL
	IsGitHub bool   // Whether this is a GitHub repository
}

// CloneOptions configures git clone behavior
type CloneOptions struct {
	// BlobSizeLimit sets the maximum blob size for partial clone.
	// Uses git's --filter=blob:limit=<size> option.
	// Examples: "10m" (10 megabytes), "1g" (1 gigabyte)
	// Use "0" or empty string to disable filtering (clone all blobs).
	BlobSizeLimit string
}

// Client defines the interface for Git operations
type Client interface {
	// Clone clones a repository to the specified path.
	// opts can be nil to use default behavior.
	Clone(ctx context.Context, url, path string, opts *CloneOptions) error

	// CloneWithBranch clones a repository to the specified path with a specific branch.
	// If branch is empty, behaves like Clone.
	// opts can be nil to use default behavior.
	CloneWithBranch(ctx context.Context, url, path, branch string, opts *CloneOptions) error

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

	// AddRemote adds a new remote to the repository
	AddRemote(ctx context.Context, repoPath, remoteName, remoteURL string) error

	// GetCurrentCommitSHA returns the SHA of the current commit
	GetCurrentCommitSHA(ctx context.Context, repoPath string) (string, error)

	// GetRepositoryInfo extracts repository information from Git remote
	GetRepositoryInfo(ctx context.Context, repoPath string) (*RepositoryInfo, error)

	// GetChangedFiles returns the list of files that changed in the last commit
	GetChangedFiles(ctx context.Context, repoPath string) ([]string, error)

	// BatchRemoveFiles removes multiple files from git tracking efficiently
	BatchRemoveFiles(ctx context.Context, repoPath string, files []string, keepLocal bool) error
}
