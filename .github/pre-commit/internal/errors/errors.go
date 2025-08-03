// Package errors defines common errors for the pre-commit system
package errors

import "errors"

// Common errors
var (
	// ErrChecksFailed is returned when one or more checks fail
	ErrChecksFailed = errors.New("checks failed")

	// ErrNoChecksToRun is returned when no checks are configured to run
	ErrNoChecksToRun = errors.New("no checks to run")

	// ErrEnvFileNotFound is returned when .env.shared cannot be found
	ErrEnvFileNotFound = errors.New(".github/.env.shared not found in any parent directory")

	// ErrRepositoryRootNotFound is returned when git repository root cannot be determined
	ErrRepositoryRootNotFound = errors.New("unable to determine repository root")

	// ErrToolNotFound is returned when a required tool is not available
	ErrToolNotFound = errors.New("required tool not found")

	// ErrLintingIssues is returned when linting finds issues
	ErrLintingIssues = errors.New("linting issues found")

	// ErrNotTidy is returned when go.mod/go.sum are not tidy
	ErrNotTidy = errors.New("go.mod or go.sum are not tidy")

	// ErrWhitespaceIssues is returned when whitespace issues are found
	ErrWhitespaceIssues = errors.New("whitespace issues found")

	// ErrEOFIssues is returned when EOF issues are found
	ErrEOFIssues = errors.New("EOF issues found")
)
