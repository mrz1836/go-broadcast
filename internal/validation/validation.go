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

// Length limits for validation to prevent DoS attacks
const (
	// MaxRepoNameLength is the maximum length for repository names (org/repo)
	MaxRepoNameLength = 200

	// MaxOrgNameLength is the maximum length for organization names
	MaxOrgNameLength = 100

	// MaxRepoShortNameLength is the maximum length for repository short names (without org prefix)
	MaxRepoShortNameLength = 100

	// MaxBranchNameLength is the maximum length for branch names (Git limit is ~255)
	MaxBranchNameLength = 255

	// MaxEmailLength is the maximum length for email addresses
	MaxEmailLength = 254 // RFC 5321 limit

	// MaxFilePathLength is the maximum length for file paths
	MaxFilePathLength = 4096 // Common OS limit
)

// Validation patterns compiled once for efficiency
var (
	// repoNamePattern validates repository names in org/repo format
	repoNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][\w.-]*/[a-zA-Z0-9][\w.-]*$`)

	// nameSegmentPattern validates a single name segment (org name or repo short name)
	nameSegmentPattern = regexp.MustCompile(`^[a-zA-Z0-9][\w.-]*$`)

	// branchNamePattern validates branch names with allowed characters
	branchNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][\w./\-]*$`)

	// branchPrefixPattern validates branch prefixes (same as branch names)
	branchPrefixPattern = regexp.MustCompile(`^[a-zA-Z0-9][\w./\-]*$`)

	// emailPattern validates email addresses using a basic pattern
	// This follows RFC 5322 simplified pattern for practical use
	// Rejects consecutive dots and leading/trailing dots in local part
	emailPattern = regexp.MustCompile(`^[a-zA-Z0-9_%+\-]([a-zA-Z0-9._%+\-]*[a-zA-Z0-9_%+\-])?@[a-zA-Z0-9]([a-zA-Z0-9.\-]*[a-zA-Z0-9])?\.([a-zA-Z]{2,})$`)
)

// ValidateRepoName validates repository name format.
// Expects org/repo format and ensures no path traversal attempts.
func ValidateRepoName(name string) error {
	if name == "" {
		return errors.EmptyFieldError("repository name")
	}

	if len(name) > MaxRepoNameLength {
		return errors.ValidationError("repository name", fmt.Sprintf("exceeds maximum length of %d characters", MaxRepoNameLength))
	}

	if !repoNamePattern.MatchString(name) {
		return errors.FormatError("repository name", name, "org/repo")
	}

	if strings.Contains(name, "..") {
		return errors.PathTraversalError(name)
	}

	// Check for trailing dots (GitHub doesn't allow)
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if strings.HasSuffix(part, ".") {
			return errors.InvalidFieldError("repository name", name+" (segment ends with '.')")
		}
	}

	return nil
}

// ValidateOrgName validates an organization name (a single segment, no slash).
func ValidateOrgName(name string) error {
	if name == "" {
		return errors.EmptyFieldError("organization name")
	}

	if len(name) > MaxOrgNameLength {
		return errors.ValidationError("organization name", fmt.Sprintf("exceeds maximum length of %d characters", MaxOrgNameLength))
	}

	if strings.Contains(name, "/") {
		return errors.InvalidFieldError("organization name", name+" (contains '/')")
	}

	if strings.Contains(name, "..") {
		return errors.PathTraversalError(name)
	}

	if !nameSegmentPattern.MatchString(name) {
		return errors.FormatError("organization name", name, "alphanumeric with hyphens, dots, underscores")
	}

	if strings.HasSuffix(name, ".") {
		return errors.InvalidFieldError("organization name", name+" (ends with '.')")
	}

	return nil
}

// ValidateRepoShortName validates a repository short name (without org prefix, no slash).
func ValidateRepoShortName(name string) error {
	if name == "" {
		return errors.EmptyFieldError("repository name")
	}

	if len(name) > MaxRepoShortNameLength {
		return errors.ValidationError("repository name", fmt.Sprintf("exceeds maximum length of %d characters", MaxRepoShortNameLength))
	}

	if strings.Contains(name, "/") {
		return errors.InvalidFieldError("repository name", name+" (contains '/')")
	}

	if strings.Contains(name, "..") {
		return errors.PathTraversalError(name)
	}

	if !nameSegmentPattern.MatchString(name) {
		return errors.FormatError("repository name", name, "alphanumeric with hyphens, dots, underscores")
	}

	if strings.HasSuffix(name, ".") {
		return errors.InvalidFieldError("repository name", name+" (ends with '.')")
	}

	return nil
}

// ValidateBranchName validates branch name format.
// Ensures branch names contain only allowed characters and follow Git branch naming rules.
func ValidateBranchName(name string) error {
	if name == "" {
		return errors.EmptyFieldError("branch name")
	}

	if len(name) > MaxBranchNameLength {
		return errors.ValidationError("branch name", fmt.Sprintf("exceeds maximum length of %d characters", MaxBranchNameLength))
	}

	if !branchNamePattern.MatchString(name) {
		return errors.InvalidFieldError("branch name", name)
	}

	// Git-specific validations
	if strings.Contains(name, "..") {
		return errors.InvalidFieldError("branch name", name+" (contains '..')")
	}

	if strings.HasSuffix(name, "/") {
		return errors.InvalidFieldError("branch name", name+" (ends with '/')")
	}

	if strings.Contains(name, "//") {
		return errors.InvalidFieldError("branch name", name+" (contains '//')")
	}

	if strings.HasSuffix(name, ".lock") {
		return errors.InvalidFieldError("branch name", name+" (ends with '.lock')")
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

	if len(prefix) > MaxBranchNameLength {
		return errors.ValidationError("branch prefix", fmt.Sprintf("exceeds maximum length of %d characters", MaxBranchNameLength))
	}

	if !branchPrefixPattern.MatchString(prefix) {
		return errors.InvalidFieldError("branch prefix", prefix)
	}

	// Git-specific validations (same as branch names)
	if strings.Contains(prefix, "..") {
		return errors.InvalidFieldError("branch prefix", prefix+" (contains '..')")
	}

	if strings.HasSuffix(prefix, "/") {
		return errors.InvalidFieldError("branch prefix", prefix+" (ends with '/')")
	}

	if strings.Contains(prefix, "//") {
		return errors.InvalidFieldError("branch prefix", prefix+" (contains '//')")
	}

	if strings.HasSuffix(prefix, ".lock") {
		return errors.InvalidFieldError("branch prefix", prefix+" (ends with '.lock')")
	}

	return nil
}

// ValidateEmail validates email address format.
// Uses a simplified RFC 5322 pattern for practical email validation.
// Empty emails are allowed as they are optional configuration.
func ValidateEmail(email, fieldName string) error {
	// Empty email is allowed (optional field)
	if email == "" {
		return nil
	}

	if len(email) > MaxEmailLength {
		return errors.ValidationError(fieldName, fmt.Sprintf("exceeds maximum length of %d characters", MaxEmailLength))
	}

	// Check for consecutive dots (not allowed in email addresses)
	if strings.Contains(email, "..") {
		return errors.InvalidFieldError(fieldName, email+" (contains consecutive dots)")
	}

	if !emailPattern.MatchString(email) {
		return errors.InvalidFieldError(fieldName, email)
	}

	return nil
}

// ValidateFilePath validates file paths for security and format.
// Ensures paths are relative and don't escape the repository via path traversal.
// Rejects null bytes and control characters for security.
func ValidateFilePath(path, fieldName string) error {
	if path == "" {
		return errors.RequiredFieldError(fieldName + " path")
	}

	if len(path) > MaxFilePathLength {
		return errors.ValidationError(fieldName+" path", fmt.Sprintf("exceeds maximum length of %d characters", MaxFilePathLength))
	}

	// Check for null bytes and control characters (security)
	for i := 0; i < len(path); i++ {
		if path[i] < 0x20 { // Control characters including null byte
			return errors.ValidationError(fieldName+" path", "contains invalid control characters")
		}
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
// Result is NOT safe for concurrent use. If concurrent access is needed,
// external synchronization must be provided.
type Result struct {
	Valid  bool
	Errors []error
}

// MaxErrorsInMessage is the maximum number of errors included in AllErrors message.
// Additional errors are truncated with a count.
const MaxErrorsInMessage = 10

// AddError adds an error to the validation result.
// Safe to call on nil receiver (no-op).
func (vr *Result) AddError(err error) {
	if vr == nil || err == nil {
		return
	}
	vr.Valid = false
	vr.Errors = append(vr.Errors, err)
}

// FirstError returns the first validation error or nil if valid.
// Safe to call on nil receiver (returns nil).
func (vr *Result) FirstError() error {
	if vr == nil || len(vr.Errors) == 0 {
		return nil
	}
	return vr.Errors[0]
}

// AllErrors returns all validation errors as a single combined error.
// If there are more than MaxErrorsInMessage errors, the message is truncated.
// Safe to call on nil receiver (returns nil).
func (vr *Result) AllErrors() error {
	if vr == nil || len(vr.Errors) == 0 {
		return nil
	}

	if len(vr.Errors) == 1 {
		return vr.Errors[0]
	}

	// Limit errors in message to prevent unbounded output
	errorCount := len(vr.Errors)
	displayCount := errorCount
	if displayCount > MaxErrorsInMessage {
		displayCount = MaxErrorsInMessage
	}

	// Combine multiple errors
	messages := make([]string, 0, displayCount+1)
	for i := 0; i < displayCount; i++ {
		messages = append(messages, vr.Errors[i].Error())
	}

	if errorCount > MaxErrorsInMessage {
		messages = append(messages, fmt.Sprintf("... and %d more errors", errorCount-MaxErrorsInMessage))
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

	// Check for duplicate destinations (normalize paths for comparison)
	seenDest := make(map[string]bool)
	for i, mapping := range fileMappings {
		// Track error count before this iteration
		errorCountBefore := len(result.Errors)

		result.AddError(ValidateFileMapping(mapping))

		// Normalize destination path for duplicate detection
		normalizedDest := filepath.Clean(mapping.Dest)
		if seenDest[normalizedDest] {
			result.AddError(errors.ValidationError("file mappings", "duplicate destination: "+mapping.Dest))
		}
		seenDest[normalizedDest] = true

		// Wrap only errors added in this iteration with mapping context
		for j := errorCountBefore; j < len(result.Errors); j++ {
			result.Errors[j] = errors.WrapWithContext(result.Errors[j], fmt.Sprintf("validate file mapping[%d]", i))
		}
	}

	return result.FirstError()
}

// FileMapping represents a file mapping for validation.
type FileMapping struct {
	Src    string
	Dest   string
	Delete bool
}

// ValidateFileMapping validates a single file mapping.
// This consolidates file mapping validation logic.
func ValidateFileMapping(mapping FileMapping) error {
	result := NewValidationResult()

	// For deletions, source path can be empty
	if !mapping.Delete {
		result.AddError(ValidateFilePath(mapping.Src, "source"))
	}
	result.AddError(ValidateFilePath(mapping.Dest, "destination"))

	return result.FirstError()
}
