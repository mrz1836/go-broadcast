// Package errors defines common error types used throughout the application
package errors

import "errors"

// Common errors used across the application
var (
	// Sync errors
	ErrNoFilesToCommit   = errors.New("no files to commit")
	ErrNoTargets         = errors.New("no targets configured")
	ErrInvalidConfig     = errors.New("invalid configuration")
	ErrSyncFailed        = errors.New("sync operation failed")
	ErrNoMatchingTargets = errors.New("no targets match the specified filter")
	ErrFileNotFound      = errors.New("source file not found")
	ErrTransformNotFound = errors.New("transform not found")

	// State errors
	ErrPRExists       = errors.New("pull request already exists")
	ErrPRNotFound     = errors.New("pull request not found")
	ErrBranchNotFound = errors.New("branch not found")

	// Git errors
	ErrInvalidRepoPath = errors.New("invalid repository path")
	ErrGitCommand      = errors.New("git command failed")

	// Test errors (only used in tests)
	ErrTest = errors.New("test error")
)
