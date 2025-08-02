// Package errors - API operation error utilities
package errors

import (
	"errors"
	"fmt"
)

// Error templates for API operations
var (
	errGitOperationTemplate   = errors.New("git operation failed")
	errGitHubAPITemplate      = errors.New("GitHub API operation failed")
	errAPIResponseTemplate    = errors.New("API response error")
	errRateLimitTemplate      = errors.New("rate limit exceeded")
	errAuthenticationTemplate = errors.New("authentication failed")
)

// GitOperationError creates a standardized git command error.
// This consolidates patterns like fmt.Errorf("git %s failed: %w", operation, err).
//
// Example usage:
//
//	return GitOperationError("clone", "user/repo", err)
//	// Returns: "git operation failed: clone 'user/repo': <original error>"
func GitOperationError(operation, context string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s '%s': %w", errGitOperationTemplate, operation, context, err)
}

// GitCloneError is a convenience function for git clone errors.
func GitCloneError(repo string, err error) error {
	return GitOperationError("clone", repo, err)
}

// GitCheckoutError is a convenience function for git checkout errors.
func GitCheckoutError(branch string, err error) error {
	return GitOperationError("checkout", branch, err)
}

// GitAddError is a convenience function for git add errors.
func GitAddError(files string, err error) error {
	return GitOperationError("add", files, err)
}

// GitCommitError is a convenience function for git commit errors.
func GitCommitError(message string, err error) error {
	return GitOperationError("commit", message, err)
}

// GitPushError is a convenience function for git push errors.
func GitPushError(branch string, err error) error {
	return GitOperationError("push", branch, err)
}

// GitHubAPIError creates a standardized GitHub API error.
// This consolidates patterns like fmt.Errorf("GitHub API %s failed: %w", operation, err).
//
// Example usage:
//
//	return GitHubAPIError("create pull request", "user/repo", err)
//	// Returns: "GitHub API operation failed: create pull request 'user/repo': <original error>"
func GitHubAPIError(operation, context string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s '%s': %w", errGitHubAPITemplate, operation, context, err)
}

// GitHubListError is a convenience function for GitHub list operations.
func GitHubListError(resource, context string, err error) error {
	return GitHubAPIError("list "+resource, context, err)
}

// GitHubGetError is a convenience function for GitHub get operations.
func GitHubGetError(resource, context string, err error) error {
	return GitHubAPIError("get "+resource, context, err)
}

// GitHubCreateError is a convenience function for GitHub create operations.
func GitHubCreateError(resource, context string, err error) error {
	return GitHubAPIError("create "+resource, context, err)
}

// GitHubUpdateError is a convenience function for GitHub update operations.
func GitHubUpdateError(resource, context string, err error) error {
	return GitHubAPIError("update "+resource, context, err)
}

// APIResponseError creates a standardized API response error.
// This is for unexpected API responses or status codes.
//
// Example usage:
//
//	return APIResponseError(404, "repository not found")
//	// Returns: "API response error: status 404: repository not found"
func APIResponseError(statusCode int, message string) error {
	return fmt.Errorf("%w: status %d: %s", errAPIResponseTemplate, statusCode, message)
}

// RateLimitError creates a standardized rate limit error.
// This provides consistent rate limit error messages.
//
// Example usage:
//
//	return RateLimitError("GitHub API", resetTime)
//	// Returns: "rate limit exceeded: GitHub API: resets at <time>"
func RateLimitError(service string, resetTime string) error {
	return fmt.Errorf("%w: %s: resets at %s", errRateLimitTemplate, service, resetTime)
}

// AuthenticationError creates a standardized authentication error.
// This provides consistent authentication error messages.
//
// Example usage:
//
//	return AuthenticationError("GitHub", "invalid token")
//	// Returns: "authentication failed: GitHub: invalid token"
func AuthenticationError(service, reason string) error {
	return fmt.Errorf("%w: %s: %s", errAuthenticationTemplate, service, reason)
}
