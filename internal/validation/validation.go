// Package validation provides shared validation utilities and patterns used throughout the application.
// This package consolidates validation logic that was previously scattered across multiple packages.
package validation

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mrz1836/go-broadcast/internal/errors"
)

// Validation patterns compiled once for efficiency
var (
	// repoNamePattern validates repository names in org/repo format
	repoNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][\w.-]*/[a-zA-Z0-9][\w.-]*$`)

	// branchNamePattern validates branch names with allowed characters
	branchNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][\w./\-]*$`)

	// branchPrefixPattern validates branch prefixes (same as branch names)
	branchPrefixPattern = regexp.MustCompile(`^[a-zA-Z0-9][\w./\-]*$`)
)

// ValidateRepoName validates repository name format.
// Expects org/repo format and ensures no path traversal attempts.
func ValidateRepoName(name string) error {
	if name == "" {
		return errors.EmptyFieldError("repository name")
	}

	if !repoNamePattern.MatchString(name) {
		return errors.FormatError("repository name", name, "org/repo")
	}

	if strings.Contains(name, "..") {
		return errors.PathTraversalError(name)
	}

	return nil
}

// ValidateBranchName validates branch name format.
// Ensures branch names contain only allowed characters.
func ValidateBranchName(name string) error {
	if name == "" {
		return errors.EmptyFieldError("branch name")
	}

	if !branchNamePattern.MatchString(name) {
		return errors.InvalidFieldError("branch name", name)
	}

	return nil
}

// ValidateBranchPrefix validates branch prefix format.
// Uses the same rules as branch names but allows empty values.
func ValidateBranchPrefix(prefix string) error {
	// Empty prefix is allowed (will use default)
	if prefix == "" {
		return nil
	}

	if !branchPrefixPattern.MatchString(prefix) {
		return errors.InvalidFieldError("branch prefix", prefix)
	}

	return nil
}

// ValidateFilePath validates file paths for security and format.
// Ensures paths are relative and don't escape the repository via path traversal.
func ValidateFilePath(path, fieldName string) error {
	if path == "" {
		return errors.RequiredFieldError(fieldName + " path")
	}

	// Clean the path and check for security issues
	cleanPath := filepath.Clean(path)

	// Check for absolute paths
	if filepath.IsAbs(cleanPath) {
		return errors.ValidationError(fieldName+" path", "must be relative, not absolute")
	}

	// Check for path traversal attempts
	if strings.HasPrefix(cleanPath, "..") {
		return errors.PathTraversalError(path)
	}

	return nil
}

// ValidateNonEmpty validates that a string field is not empty or whitespace-only.
// This is commonly used for required string fields.
func ValidateNonEmpty(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.EmptyFieldError(field)
	}
	return nil
}

// SanitizeInput performs basic input sanitization for string values.
// Trims whitespace and can be extended with additional sanitization rules.
func SanitizeInput(input string) string {
	return strings.TrimSpace(input)
}

// Result represents the result of a validation operation.
type Result struct {
	Valid  bool
	Errors []error
}

// AddError adds an error to the validation result.
func (vr *Result) AddError(err error) {
	if err != nil {
		vr.Valid = false
		vr.Errors = append(vr.Errors, err)
	}
}

// FirstError returns the first validation error or nil if valid.
func (vr *Result) FirstError() error {
	if len(vr.Errors) > 0 {
		return vr.Errors[0]
	}
	return nil
}

// AllErrors returns all validation errors as a single combined error.
func (vr *Result) AllErrors() error {
	if len(vr.Errors) == 0 {
		return nil
	}

	if len(vr.Errors) == 1 {
		return vr.Errors[0]
	}

	// Combine multiple errors
	messages := make([]string, 0, len(vr.Errors))
	for _, err := range vr.Errors {
		messages = append(messages, err.Error())
	}

	return errors.ValidationError("multiple fields", strings.Join(messages, "; "))
}

// NewValidationResult creates a new validation result initialized as valid.
func NewValidationResult() *Result {
	return &Result{
		Valid:  true,
		Errors: make([]error, 0),
	}
}

// ValidateSourceConfig validates source repository configuration.
// This consolidates validation logic from config package.
func ValidateSourceConfig(repo, branch string) error {
	result := NewValidationResult()

	result.AddError(ValidateRepoName(repo))
	result.AddError(ValidateBranchName(branch))

	return result.FirstError()
}

// ValidateTargetConfig validates target repository configuration.
// This consolidates validation logic for target repositories.
func ValidateTargetConfig(repo string, fileMappings []FileMapping) error {
	result := NewValidationResult()

	// Validate repository name
	result.AddError(ValidateRepoName(repo))

	// Validate file mappings
	if len(fileMappings) == 0 {
		result.AddError(errors.ValidationError("target repository", "at least one file mapping is required"))
	}

	// Check for duplicate destinations
	seenDest := make(map[string]bool)
	for i, mapping := range fileMappings {
		result.AddError(ValidateFileMapping(mapping))

		if seenDest[mapping.Dest] {
			result.AddError(errors.ValidationError("file mappings", "duplicate destination: "+mapping.Dest))
		}
		seenDest[mapping.Dest] = true

		// Add context to errors for specific file mapping
		if len(result.Errors) > 0 {
			lastErr := result.Errors[len(result.Errors)-1]
			result.Errors[len(result.Errors)-1] = errors.WrapWithContext(lastErr, fmt.Sprintf("validate file mapping[%d]", i))
		}
	}

	return result.FirstError()
}

// FileMapping represents a file mapping for validation.
type FileMapping struct {
	Src  string
	Dest string
}

// ValidateFileMapping validates a single file mapping.
// This consolidates file mapping validation logic.
func ValidateFileMapping(mapping FileMapping) error {
	result := NewValidationResult()

	result.AddError(ValidateFilePath(mapping.Src, "source"))
	result.AddError(ValidateFilePath(mapping.Dest, "destination"))

	return result.FirstError()
}
