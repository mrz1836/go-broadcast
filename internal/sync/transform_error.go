// Package sync provides specialized error handling for transform operations
package sync

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
)

// TransformErrorCategory categorizes different types of transform errors
type TransformErrorCategory string

const (
	// CategoryBinaryFile indicates errors related to binary file processing
	CategoryBinaryFile TransformErrorCategory = "binary_file"

	// CategoryTemplateParse indicates template parsing errors
	CategoryTemplateParse TransformErrorCategory = "template_parse"

	// CategoryVariableSubstitution indicates variable substitution errors
	CategoryVariableSubstitution TransformErrorCategory = "variable_substitution"

	// CategoryRepoNameFormat indicates repository name format errors
	CategoryRepoNameFormat TransformErrorCategory = "repo_name_format"

	// CategoryGenericTransform indicates generic transform errors
	CategoryGenericTransform TransformErrorCategory = "generic_transform"

	// CategoryFileSystem indicates file system related errors
	CategoryFileSystem TransformErrorCategory = "file_system"

	// CategoryTimeout indicates timeout errors during transformation
	CategoryTimeout TransformErrorCategory = "timeout"

	// CategoryContext indicates context cancellation errors
	CategoryContext TransformErrorCategory = "context"
)

// TransformError provides detailed error context for transform operations
type TransformError struct {
	// Underlying error
	err error

	// File being transformed
	filePath string

	// Source and target repositories
	sourceRepo string
	targetRepo string

	// Transform type information
	transformType string

	// Directory operation context
	isFromDirectory bool
	relativePath    string

	// Error categorization
	category TransformErrorCategory

	// Recovery information
	recoverable bool
	retryable   bool

	// Timing information
	timestamp time.Time
	duration  time.Duration

	// Additional context
	metadata map[string]interface{}
}

// Error implements the error interface
func (te *TransformError) Error() string {
	var parts []string

	// Base error message
	if te.err != nil {
		parts = append(parts, fmt.Sprintf("transform failed: %s", te.err.Error()))
	} else {
		parts = append(parts, "transform failed")
	}

	// Add file context
	if te.filePath != "" {
		if te.isFromDirectory && te.relativePath != "" {
			parts = append(parts, fmt.Sprintf("file: %s (relative: %s)", te.filePath, te.relativePath))
		} else {
			parts = append(parts, fmt.Sprintf("file: %s", te.filePath))
		}
	}

	// Add repository context
	if te.sourceRepo != "" && te.targetRepo != "" {
		parts = append(parts, fmt.Sprintf("repos: %s -> %s", te.sourceRepo, te.targetRepo))
	}

	// Add transform type
	if te.transformType != "" {
		parts = append(parts, fmt.Sprintf("transform: %s", te.transformType))
	}

	// Add category
	parts = append(parts, fmt.Sprintf("category: %s", te.category))

	return strings.Join(parts, " | ")
}

// Unwrap returns the underlying error for error.Is and error.As
func (te *TransformError) Unwrap() error {
	return te.err
}

// Is implements error comparison for errors.Is
func (te *TransformError) Is(target error) bool {
	var targetTE *TransformError
	if errors.As(target, &targetTE) {
		return te.category == targetTE.category
	}
	return errors.Is(te.err, target)
}

// NewTransformError creates a new transform error with comprehensive context
func NewTransformError(
	err error,
	filePath string,
	sourceRepo, targetRepo string,
	transformType string,
) *TransformError {
	return &TransformError{
		err:           err,
		filePath:      filePath,
		sourceRepo:    sourceRepo,
		targetRepo:    targetRepo,
		transformType: transformType,
		category:      categorizeError(err),
		recoverable:   isRecoverableError(err),
		retryable:     isRetryableError(err),
		timestamp:     time.Now(),
		metadata:      make(map[string]interface{}),
	}
}

// NewDirectoryTransformError creates a transform error for directory operations
func NewDirectoryTransformError(
	err error,
	filePath, relativePath string,
	sourceRepo, targetRepo string,
	transformType string,
) *TransformError {
	te := NewTransformError(err, filePath, sourceRepo, targetRepo, transformType)
	te.isFromDirectory = true
	te.relativePath = relativePath
	return te
}

// WithDuration adds timing information to the error
func (te *TransformError) WithDuration(duration time.Duration) *TransformError {
	te.duration = duration
	return te
}

// WithMetadata adds metadata to the error
func (te *TransformError) WithMetadata(key string, value interface{}) *TransformError {
	te.metadata[key] = value
	return te
}

// GetCategory returns the error category
func (te *TransformError) GetCategory() TransformErrorCategory {
	return te.category
}

// IsRecoverable returns whether the error allows for fallback strategies
func (te *TransformError) IsRecoverable() bool {
	return te.recoverable
}

// ShouldRetry returns whether the operation should be retried
func (te *TransformError) ShouldRetry() bool {
	return te.retryable
}

// GetFilePath returns the file path being transformed
func (te *TransformError) GetFilePath() string {
	return te.filePath
}

// GetRelativePath returns the relative path within directory operations
func (te *TransformError) GetRelativePath() string {
	return te.relativePath
}

// IsFromDirectory returns whether this error is from a directory operation
func (te *TransformError) IsFromDirectory() bool {
	return te.isFromDirectory
}

// GetSourceRepo returns the source repository
func (te *TransformError) GetSourceRepo() string {
	return te.sourceRepo
}

// GetTargetRepo returns the target repository
func (te *TransformError) GetTargetRepo() string {
	return te.targetRepo
}

// GetTransformType returns the transform type
func (te *TransformError) GetTransformType() string {
	return te.transformType
}

// GetTimestamp returns when the error occurred
func (te *TransformError) GetTimestamp() time.Time {
	return te.timestamp
}

// GetDuration returns how long the transform took before failing
func (te *TransformError) GetDuration() time.Duration {
	return te.duration
}

// GetMetadata returns the error metadata
func (te *TransformError) GetMetadata() map[string]interface{} {
	return te.metadata
}

// categorizeError determines the error category based on the underlying error
func categorizeError(err error) TransformErrorCategory {
	if err == nil {
		return CategoryGenericTransform
	}

	errStr := strings.ToLower(err.Error())

	// Check for context errors first
	if errors.Is(err, context.Canceled) {
		return CategoryContext
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return CategoryTimeout
	}

	// Check for specific error types
	if errors.Is(err, internalerrors.ErrTransformNotFound) {
		return CategoryGenericTransform
	}

	// Check for binary file indicators
	if strings.Contains(errStr, "binary") || strings.Contains(errStr, "non-text") {
		return CategoryBinaryFile
	}

	// Check for template parsing errors
	if strings.Contains(errStr, "template") || strings.Contains(errStr, "parse") {
		return CategoryTemplateParse
	}

	// Check for variable substitution errors
	if strings.Contains(errStr, "variable") || strings.Contains(errStr, "substitution") {
		return CategoryVariableSubstitution
	}

	// Check for repository format errors
	if strings.Contains(errStr, "repository format") || strings.Contains(errStr, "invalid repo") {
		return CategoryRepoNameFormat
	}

	// Check for file system errors
	if strings.Contains(errStr, "no such file") || strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "file system") || strings.Contains(errStr, "path") {
		return CategoryFileSystem
	}

	// Check for timeout-related errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return CategoryTimeout
	}

	return CategoryGenericTransform
}

// isRecoverableError determines if an error allows for fallback strategies
func isRecoverableError(err error) bool {
	if err == nil {
		return true
	}

	// Non-recoverable error categories
	nonRecoverableCategories := []TransformErrorCategory{
		CategoryFileSystem,
		CategoryContext,
	}

	category := categorizeError(err)
	return !slices.Contains(nonRecoverableCategories, category)
}

// isRetryableError determines if an operation should be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Retryable error categories
	retryableCategories := []TransformErrorCategory{
		CategoryTimeout,
		CategoryFileSystem, // Some file system errors are transient
	}

	category := categorizeError(err)

	// Context cancellation is not retryable
	if errors.Is(err, context.Canceled) {
		return false
	}

	return slices.Contains(retryableCategories, category)
}

// TransformCollectionError aggregates multiple transform errors from directory operations
type TransformCollectionError struct {
	errors      []*TransformError
	successful  int
	failed      int
	recoverable int
	retryable   int
}

// NewTransformCollectionError creates a new error collection
func NewTransformCollectionError() *TransformCollectionError {
	return &TransformCollectionError{
		errors: make([]*TransformError, 0),
	}
}

// Add adds a transform error to the collection
func (tec *TransformCollectionError) Add(err *TransformError) {
	if err == nil {
		return
	}

	tec.errors = append(tec.errors, err)
	tec.failed++

	if err.IsRecoverable() {
		tec.recoverable++
	}

	if err.ShouldRetry() {
		tec.retryable++
	}
}

// AddSuccess records a successful transform
func (tec *TransformCollectionError) AddSuccess() {
	tec.successful++
}

// GetErrors returns all errors in the collection
func (tec *TransformCollectionError) GetErrors() []*TransformError {
	return slices.Clone(tec.errors)
}

// GetErrorsByCategory returns errors filtered by category
func (tec *TransformCollectionError) GetErrorsByCategory(category TransformErrorCategory) []*TransformError {
	var filtered []*TransformError
	for _, err := range tec.errors {
		if err.GetCategory() == category {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// HasErrors returns whether the collection contains any errors
func (tec *TransformCollectionError) HasErrors() bool {
	return len(tec.errors) > 0
}

// Error implements the error interface for the collection
func (tec *TransformCollectionError) Error() string {
	if !tec.HasErrors() {
		return "no transform errors"
	}

	categories := make(map[TransformErrorCategory]int)
	for _, err := range tec.errors {
		categories[err.GetCategory()]++
	}

	parts := make([]string, 0, len(categories))
	for category, count := range categories {
		parts = append(parts, fmt.Sprintf("%s: %d", category, count))
	}

	return fmt.Sprintf("transform errors (%d total, %d successful, %d recoverable, %d retryable): %s",
		tec.failed, tec.successful, tec.recoverable, tec.retryable, strings.Join(parts, ", "))
}

// GetSummary returns a summary of the error collection
func (tec *TransformCollectionError) GetSummary() TransformErrorSummary {
	categories := make(map[TransformErrorCategory]int)
	files := make(map[string]int)

	for _, err := range tec.errors {
		categories[err.GetCategory()]++

		filePath := err.GetFilePath()
		if err.IsFromDirectory() && err.GetRelativePath() != "" {
			filePath = err.GetRelativePath()
		}
		if filePath != "" {
			files[filePath]++
		}
	}

	return TransformErrorSummary{
		Total:       tec.failed,
		Successful:  tec.successful,
		Recoverable: tec.recoverable,
		Retryable:   tec.retryable,
		Categories:  categories,
		FileErrors:  files,
	}
}

// TransformErrorSummary provides a summary of transform errors
type TransformErrorSummary struct {
	Total       int                            `json:"total"`
	Successful  int                            `json:"successful"`
	Recoverable int                            `json:"recoverable"`
	Retryable   int                            `json:"retryable"`
	Categories  map[TransformErrorCategory]int `json:"categories"`
	FileErrors  map[string]int                 `json:"file_errors"`
}

// TransformRecoveryStrategy defines strategies for handling transform errors
type TransformRecoveryStrategy int

const (
	// RecoveryStrategyNone performs no recovery
	RecoveryStrategyNone TransformRecoveryStrategy = iota

	// RecoveryStrategyUseOriginal uses the original content without transformation
	RecoveryStrategyUseOriginal

	// RecoveryStrategySkipFile skips the file entirely
	RecoveryStrategySkipFile

	// RecoveryStrategyRetry retries the transformation
	RecoveryStrategyRetry
)

// GetRecoveryStrategy determines the appropriate recovery strategy for an error
func GetRecoveryStrategy(err *TransformError) TransformRecoveryStrategy {
	if err == nil {
		return RecoveryStrategyNone
	}

	// Context cancellation should not be recovered
	if err.GetCategory() == CategoryContext {
		return RecoveryStrategyNone
	}

	// Binary files should be skipped rather than transformed
	if err.GetCategory() == CategoryBinaryFile {
		return RecoveryStrategySkipFile
	}

	// Retryable errors should be retried first
	if err.ShouldRetry() {
		return RecoveryStrategyRetry
	}

	// Recoverable errors can use original content
	if err.IsRecoverable() {
		return RecoveryStrategyUseOriginal
	}

	// Non-recoverable errors should skip the file
	return RecoveryStrategySkipFile
}

// ValidateTransformContext validates transform context for common error conditions
func ValidateTransformContext(filePath, sourceRepo, targetRepo string) error {
	if filePath == "" {
		return internalerrors.EmptyFieldError("file_path")
	}

	if sourceRepo == "" {
		return internalerrors.EmptyFieldError("source_repo")
	}

	if targetRepo == "" {
		return internalerrors.EmptyFieldError("target_repo")
	}

	// Validate repository format
	if !isValidRepoFormat(sourceRepo) {
		return internalerrors.FormatError("source_repo", sourceRepo, "org/repo")
	}

	if !isValidRepoFormat(targetRepo) {
		return internalerrors.FormatError("target_repo", targetRepo, "org/repo")
	}

	// Check for path traversal
	if strings.Contains(filePath, "..") {
		return internalerrors.PathTraversalError(filePath)
	}

	return nil
}

// isValidRepoFormat checks if a repository string follows the org/repo format
func isValidRepoFormat(repo string) bool {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return false
	}

	// Check that both parts are non-empty and don't contain invalid characters
	for _, part := range parts {
		if part == "" || strings.ContainsAny(part, " \t\n\r") {
			return false
		}
	}

	return true
}

// WrapTransformError wraps a regular error as a TransformError if it isn't already
func WrapTransformError(err error, filePath, sourceRepo, targetRepo, transformType string) error {
	if err == nil {
		return nil
	}

	// If it's already a TransformError, return as-is
	var te *TransformError
	if errors.As(err, &te) {
		return err
	}

	// Wrap as new TransformError
	return NewTransformError(err, filePath, sourceRepo, targetRepo, transformType)
}

// WrapDirectoryTransformError wraps a regular error as a directory TransformError
func WrapDirectoryTransformError(err error, filePath, relativePath, sourceRepo, targetRepo, transformType string) error {
	if err == nil {
		return nil
	}

	// If it's already a TransformError, update directory context and return
	var te *TransformError
	if errors.As(err, &te) {
		if !te.isFromDirectory {
			te.isFromDirectory = true
			te.relativePath = relativePath
		}
		return err
	}

	// Wrap as new directory TransformError
	return NewDirectoryTransformError(err, filePath, relativePath, sourceRepo, targetRepo, transformType)
}

// GetFileExtension returns the file extension from a path, used for error categorization
func GetFileExtension(filePath string) string {
	return strings.ToLower(filepath.Ext(filePath))
}

// IsBinaryFileExtension checks if a file extension typically indicates a binary file
func IsBinaryFileExtension(ext string) bool {
	binaryExtensions := []string{
		".exe", ".dll", ".so", ".dylib", ".a", ".o", ".obj",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".ico", ".svg",
		".mp3", ".mp4", ".avi", ".mov", ".wav", ".ogg",
		".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".class", ".jar", ".war", ".ear",
		".woff", ".woff2", ".ttf", ".otf", ".eot",
	}

	return slices.Contains(binaryExtensions, ext)
}
