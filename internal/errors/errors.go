// Package errors defines common error types and utilities used throughout the application
package errors

import (
	"errors"
	"fmt"
)

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

// Error templates for static error definitions (satisfies err113 linter)
var (
	errInvalidFieldTemplate       = errors.New("invalid field")
	errCommandFailedTemplate      = errors.New("command failed")
	errValidationFailedTemplate   = errors.New("validation failed")
	errPathTraversalTemplate      = errors.New("path traversal detected")
	errEmptyFieldTemplate         = errors.New("field cannot be empty")
	errRequiredFieldTemplate      = errors.New("field is required")
	errInvalidFormatTemplate      = errors.New("invalid format")
	errNoSourceConfigFound        = errors.New("no source configuration found")
	errConflictDetected           = errors.New("conflict detected")
	errNoSourcesInConflict        = errors.New("no sources in conflict")
	errPriorityStrategyNoPriority = errors.New("priority strategy configured but no priority list provided")
	errSourceStateNotFound        = errors.New("source state not found")
	errMappingNoTargets           = errors.New("mapping has no targets")
	errInvalidSourceID            = errors.New("invalid source ID")
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

// NoSourceConfigFoundError creates a standardized no source configuration error.
func NoSourceConfigFoundError() error {
	return errNoSourceConfigFound
}

// ConflictDetectedError creates a standardized conflict detection error.
func ConflictDetectedError(targetFile string, sourceCount int) error {
	return fmt.Errorf("%w for %s: %d sources want to sync this file", errConflictDetected, targetFile, sourceCount)
}

// NoSourcesInConflictError creates a standardized no sources in conflict error.
func NoSourcesInConflictError() error {
	return errNoSourcesInConflict
}

// PriorityStrategyNoPriorityError creates a standardized priority strategy error.
func PriorityStrategyNoPriorityError() error {
	return errPriorityStrategyNoPriority
}

// SourceStateNotFoundError creates a standardized source state not found error.
func SourceStateNotFoundError(repo string) error {
	return fmt.Errorf("%w for %s", errSourceStateNotFound, repo)
}

// MappingNoTargetsError creates a standardized mapping no targets error.
func MappingNoTargetsError(sourceRepo string) error {
	return fmt.Errorf("%w with source %s", errMappingNoTargets, sourceRepo)
}

// InvalidSourceIDError creates a standardized invalid source ID error.
func InvalidSourceIDError(sourceID string, mappingIndex int) error {
	return fmt.Errorf("%w '%s' in mapping %d: must contain only alphanumeric characters, hyphens, and underscores", errInvalidSourceID, sourceID, mappingIndex+1)
}
