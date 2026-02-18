// Package errors defines common error types and utilities used throughout the application
package errors //nolint:revive,nolintlint // internal package, name conflict intentional

import (
	"errors"
	"fmt"
)

// Common errors used across the application
var (
	// Sync errors
	ErrNoFilesToCommit   = errors.New("no files to commit")
	ErrNoChangesToSync   = errors.New("no changes to sync - files are already synchronized")
	ErrNoTargets         = errors.New("no targets configured")
	ErrInvalidConfig     = errors.New("invalid configuration")
	ErrSyncFailed        = errors.New("sync operation failed")
	ErrNoMatchingTargets = errors.New("no targets match the specified filter")
	ErrFileNotFound      = errors.New("source file not found")
	ErrFileTooLarge      = errors.New("file exceeds size limit")
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

// Error templates for static error definitions (satisfies err113 linter)
var (
	errInvalidFieldTemplate     = errors.New("invalid field")
	errCommandFailedTemplate    = errors.New("command failed")
	errValidationFailedTemplate = errors.New("validation failed")
	errPathTraversalTemplate    = errors.New("path traversal detected")
	errEmptyFieldTemplate       = errors.New("field cannot be empty")
	errRequiredFieldTemplate    = errors.New("field is required")
	errInvalidFormatTemplate    = errors.New("invalid format")
)

// Error utility functions for standardized error creation and context wrapping

// WrapWithContext wraps an error with operation context using consistent formatting.
// This replaces manual fmt.Errorf("failed to %s: %w", operation, err) patterns.
func WrapWithContext(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("failed to %s: %w", operation, err)
}

// InvalidFieldError creates a standardized invalid field error.
// This replaces manual fmt.Errorf("invalid %s: %s", field, value) patterns.
func InvalidFieldError(field, value string) error {
	return fmt.Errorf("%w: %s: %s", errInvalidFieldTemplate, field, value)
}

// CommandFailedError creates a standardized command failure error.
// This standardizes command execution error reporting across git, gh, and other packages.
func CommandFailedError(cmd string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: '%s': %w", errCommandFailedTemplate, cmd, err)
}

// ValidationError creates a standardized validation error.
// This provides consistent validation error messages across all validation functions.
func ValidationError(item, reason string) error {
	return fmt.Errorf("%w for %s: %s", errValidationFailedTemplate, item, reason)
}

// PathTraversalError creates a specific error for path traversal attempts.
// This standardizes security-related path validation errors.
func PathTraversalError(path string) error {
	return fmt.Errorf("%w: invalid path '%s'", errPathTraversalTemplate, path)
}

// EmptyFieldError creates a standardized empty field validation error.
// This replaces various "field cannot be empty" error patterns.
func EmptyFieldError(field string) error {
	return fmt.Errorf("%w: %s", errEmptyFieldTemplate, field)
}

// RequiredFieldError creates a standardized required field error.
// This provides consistent messaging for missing required fields.
func RequiredFieldError(field string) error {
	return fmt.Errorf("%w: %s", errRequiredFieldTemplate, field)
}

// FormatError creates a standardized format validation error.
// This provides consistent messaging for format validation failures.
func FormatError(field, value, expectedFormat string) error {
	return fmt.Errorf("%w: %s '%s': expected %s", errInvalidFormatTemplate, field, value, expectedFormat)
}
