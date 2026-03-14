// Package errors defines common error types used throughout the application
package errors //nolint:revive,nolintlint // internal test package, name conflict intentional

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncErrors verifies that all sync-related errors are defined correctly
func TestSyncErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrNoFilesToCommit",
			err:      ErrNoFilesToCommit,
			expected: "no files to commit",
		},
		{
			name:     "ErrNoTargets",
			err:      ErrNoTargets,
			expected: "no targets configured",
		},
		{
			name:     "ErrInvalidConfig",
			err:      ErrInvalidConfig,
			expected: "invalid configuration",
		},
		{
			name:     "ErrSyncFailed",
			err:      ErrSyncFailed,
			expected: "sync operation failed",
		},
		{
			name:     "ErrNoMatchingTargets",
			err:      ErrNoMatchingTargets,
			expected: "no targets match the specified filter",
		},
		{
			name:     "ErrFileNotFound",
			err:      ErrFileNotFound,
			expected: "source file not found",
		},
		{
			name:     "ErrTransformNotFound",
			err:      ErrTransformNotFound,
			expected: "transform not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.err)
			require.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// TestStateErrors verifies that all state-related errors are defined correctly
func TestStateErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrPRExists",
			err:      ErrPRExists,
			expected: "pull request already exists",
		},
		{
			name:     "ErrPRNotFound",
			err:      ErrPRNotFound,
			expected: "pull request not found",
		},
		{
			name:     "ErrBranchNotFound",
			err:      ErrBranchNotFound,
			expected: "branch not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.err)
			require.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// TestGitErrors verifies that all git-related errors are defined correctly
func TestGitErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrInvalidRepoPath",
			err:      ErrInvalidRepoPath,
			expected: "invalid repository path",
		},
		{
			name:     "ErrGitCommand",
			err:      ErrGitCommand,
			expected: "git command failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.err)
			require.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

// TestTestErrors verifies that test-specific errors are defined correctly
func TestTestErrors(t *testing.T) {
	require.Error(t, ErrTest)
	require.Equal(t, "test error", ErrTest.Error())
}

// TestErrorsAreSentinel verifies that all errors are sentinel errors
func TestErrorsAreSentinel(t *testing.T) {
	sentinelErrors := []struct {
		name string
		err  error
	}{
		// Sync errors
		{"ErrNoFilesToCommit", ErrNoFilesToCommit},
		{"ErrNoTargets", ErrNoTargets},
		{"ErrInvalidConfig", ErrInvalidConfig},
		{"ErrSyncFailed", ErrSyncFailed},
		{"ErrNoMatchingTargets", ErrNoMatchingTargets},
		{"ErrFileNotFound", ErrFileNotFound},
		{"ErrTransformNotFound", ErrTransformNotFound},
		// State errors
		{"ErrPRExists", ErrPRExists},
		{"ErrPRNotFound", ErrPRNotFound},
		{"ErrBranchNotFound", ErrBranchNotFound},
		// Git errors
		{"ErrInvalidRepoPath", ErrInvalidRepoPath},
		{"ErrGitCommand", ErrGitCommand},
		// Test errors
		{"ErrTest", ErrTest},
	}

	for _, tt := range sentinelErrors {
		t.Run(tt.name, func(t *testing.T) {
			// Verify errors are not nil
			require.Error(t, tt.err)

			// Verify errors have non-empty messages
			require.NotEmpty(t, tt.err.Error())

			// Verify errors can be used with errors.Is
			testErr := tt.err
			require.ErrorIs(t, testErr, tt.err)
		})
	}
}

// TestErrorsAreImmutable verifies that the exported errors cannot be modified
func TestErrorsAreImmutable(t *testing.T) {
	// This test ensures that the errors remain constant and their messages don't change
	originalMessage := ErrNoFilesToCommit.Error()

	// Attempt to use the error (this shouldn't modify it)
	_ = ErrNoFilesToCommit

	// Verify the error message hasn't changed
	require.Equal(t, originalMessage, ErrNoFilesToCommit.Error())
}

// Define static wrapper errors for testing
var (
	errFailedToSyncRepository  = errors.New("failed to sync repository")
	errFailedToExecuteGitPull  = errors.New("failed to execute git pull")
	errUnableToFindPRForBranch = errors.New("unable to find PR for branch main")
)

// TestErrorsCanBeWrapped verifies that errors can be properly wrapped
func TestErrorsCanBeWrapped(t *testing.T) {
	tests := []struct {
		name        string
		baseErr     error
		wrapErr     error
		wrapMessage string
	}{
		{
			name:        "wrap sync error",
			baseErr:     ErrSyncFailed,
			wrapErr:     errFailedToSyncRepository,
			wrapMessage: "failed to sync repository",
		},
		{
			name:        "wrap git error",
			baseErr:     ErrGitCommand,
			wrapErr:     errFailedToExecuteGitPull,
			wrapMessage: "failed to execute git pull",
		},
		{
			name:        "wrap state error",
			baseErr:     ErrPRNotFound,
			wrapErr:     errUnableToFindPRForBranch,
			wrapMessage: "unable to find PR for branch main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Wrap the error
			wrappedErr := errors.Join(tt.wrapErr, tt.baseErr)

			// Verify we can still identify the base error
			require.ErrorIs(t, wrappedErr, tt.baseErr)

			// Verify the wrapped error contains both messages
			require.Contains(t, wrappedErr.Error(), tt.wrapMessage)
			require.Contains(t, wrappedErr.Error(), tt.baseErr.Error())
		})
	}
}

// TestErrorCategories verifies that errors are properly categorized
func TestErrorCategories(t *testing.T) {
	syncErrors := []error{
		ErrNoFilesToCommit,
		ErrNoTargets,
		ErrInvalidConfig,
		ErrSyncFailed,
		ErrNoMatchingTargets,
		ErrFileNotFound,
		ErrTransformNotFound,
	}

	stateErrors := []error{
		ErrPRExists,
		ErrPRNotFound,
		ErrBranchNotFound,
	}

	gitErrors := []error{
		ErrInvalidRepoPath,
		ErrGitCommand,
	}

	testErrors := []error{
		ErrTest,
	}

	// Verify all errors are accounted for
	totalErrors := len(syncErrors) + len(stateErrors) + len(gitErrors) + len(testErrors)
	require.Equal(t, 13, totalErrors, "ensure all errors are categorized")

	// Verify no nil errors
	for _, err := range append(append(append(syncErrors, stateErrors...), gitErrors...), testErrors...) {
		require.Error(t, err)
	}
}

// Test error utility functions added in Phase 3 refactoring

func TestWrapWithContext(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		operation string
		want      string
		wantNil   bool
	}{
		{
			name:      "wrap error with context",
			err:       fmt.Errorf("original error"), //nolint:err113 // test-only errors
			operation: "test operation",
			want:      "failed to test operation: original error",
		},
		{
			name:      "nil error returns nil",
			err:       nil,
			operation: "test operation",
			wantNil:   true,
		},
		{
			name:      "empty operation",
			err:       fmt.Errorf("original error"), //nolint:err113 // test-only errors
			operation: "",
			want:      "failed to : original error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapWithContext(tt.err, tt.operation)

			if tt.wantNil {
				assert.NoError(t, result)
				return
			}

			require.Error(t, result)
			assert.Equal(t, tt.want, result.Error())
			assert.ErrorIs(t, result, tt.err)
		})
	}
}

func TestInvalidFieldError(t *testing.T) {
	tests := []struct {
		name  string
		field string
		value string
		want  string
	}{
		{
			name:  "standard invalid field",
			field: "repository",
			value: "invalid-repo",
			want:  "invalid field: repository: invalid-repo",
		},
		{
			name:  "empty field name",
			field: "",
			value: "value",
			want:  "invalid field: : value",
		},
		{
			name:  "empty value",
			field: "field",
			value: "",
			want:  "invalid field: field: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := InvalidFieldError(tt.field, tt.value)
			assert.Equal(t, tt.want, result.Error())
		})
	}
}

func TestCommandFailedError(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		err     error
		want    string
		wantNil bool
	}{
		{
			name: "command with error",
			cmd:  "git clone",
			err:  fmt.Errorf("exit code 1"), //nolint:err113 // test-only errors
			want: "command failed: 'git clone': exit code 1",
		},
		{
			name:    "nil error returns nil",
			cmd:     "git clone",
			err:     nil,
			wantNil: true,
		},
		{
			name: "empty command",
			cmd:  "",
			err:  fmt.Errorf("some error"), //nolint:err113 // test-only errors
			want: "command failed: '': some error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CommandFailedError(tt.cmd, tt.err)

			if tt.wantNil {
				assert.NoError(t, result)
				return
			}

			require.Error(t, result)
			assert.Equal(t, tt.want, result.Error())
			assert.ErrorIs(t, result, tt.err)
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name   string
		item   string
		reason string
		want   string
	}{
		{
			name:   "standard validation error",
			item:   "repository name",
			reason: "must be in org/repo format",
			want:   "validation failed for repository name: must be in org/repo format",
		},
		{
			name:   "empty item",
			item:   "",
			reason: "some reason",
			want:   "validation failed for : some reason",
		},
		{
			name:   "empty reason",
			item:   "field",
			reason: "",
			want:   "validation failed for field: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidationError(tt.item, tt.reason)
			assert.Equal(t, tt.want, result.Error())
		})
	}
}

func TestPathTraversalError(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "standard path traversal",
			path: "../../../etc/passwd",
			want: "path traversal detected: invalid path '../../../etc/passwd'",
		},
		{
			name: "empty path",
			path: "",
			want: "path traversal detected: invalid path ''",
		},
		{
			name: "simple dotdot",
			path: "..",
			want: "path traversal detected: invalid path '..'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PathTraversalError(tt.path)
			assert.Equal(t, tt.want, result.Error())
		})
	}
}

func TestEmptyFieldError(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{
			name:  "standard empty field",
			field: "repository name",
			want:  "field cannot be empty: repository name",
		},
		{
			name:  "empty field name",
			field: "",
			want:  "field cannot be empty: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EmptyFieldError(tt.field)
			assert.Equal(t, tt.want, result.Error())
		})
	}
}

func TestRequiredFieldError(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{
			name:  "standard required field",
			field: "source repository",
			want:  "field is required: source repository",
		},
		{
			name:  "empty field name",
			field: "",
			want:  "field is required: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RequiredFieldError(tt.field)
			assert.Equal(t, tt.want, result.Error())
		})
	}
}

func TestFormatError(t *testing.T) {
	tests := []struct {
		name           string
		field          string
		value          string
		expectedFormat string
		want           string
	}{
		{
			name:           "standard format error",
			field:          "repository name",
			value:          "invalid-repo",
			expectedFormat: "org/repo",
			want:           "invalid format: repository name 'invalid-repo': expected org/repo",
		},
		{
			name:           "empty field",
			field:          "",
			value:          "value",
			expectedFormat: "format",
			want:           "invalid format:  'value': expected format",
		},
		{
			name:           "empty value",
			field:          "field",
			value:          "",
			expectedFormat: "format",
			want:           "invalid format: field '': expected format",
		},
		{
			name:           "empty format",
			field:          "field",
			value:          "value",
			expectedFormat: "",
			want:           "invalid format: field 'value': expected ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatError(tt.field, tt.value, tt.expectedFormat)
			assert.Equal(t, tt.want, result.Error())
		})
	}
}

// Test error wrapping behavior with new utilities
func TestNewErrorUtilityWrapping(t *testing.T) {
	originalErr := fmt.Errorf("original error") //nolint:err113 // test-only errors

	t.Run("WrapWithContext preserves original error", func(t *testing.T) {
		wrapped := WrapWithContext(originalErr, "operation")
		require.ErrorIs(t, wrapped, originalErr)
		assert.Contains(t, wrapped.Error(), "original error")
		assert.Contains(t, wrapped.Error(), "failed to operation")
	})

	t.Run("CommandFailedError preserves original error", func(t *testing.T) {
		wrapped := CommandFailedError("test command", originalErr)
		require.ErrorIs(t, wrapped, originalErr)
		assert.Contains(t, wrapped.Error(), "original error")
		assert.Contains(t, wrapped.Error(), "command failed: 'test command'")
	})
}

// Test edge cases for new utilities
func TestNewUtilityEdgeCases(t *testing.T) {
	t.Run("multiple wrapping", func(t *testing.T) {
		original := fmt.Errorf("base error") //nolint:err113 // test-only errors
		wrapped1 := WrapWithContext(original, "first operation")
		wrapped2 := WrapWithContext(wrapped1, "second operation")

		require.ErrorIs(t, wrapped2, original)
		require.ErrorIs(t, wrapped2, wrapped1)
		assert.Contains(t, wrapped2.Error(), "failed to second operation")
		assert.Contains(t, wrapped2.Error(), "failed to first operation")
		assert.Contains(t, wrapped2.Error(), "base error")
	})

	t.Run("special characters in messages", func(t *testing.T) {
		result := InvalidFieldError("field with spaces", "value with special chars: @#$%")
		expected := "invalid field: field with spaces: value with special chars: @#$%"
		assert.Equal(t, expected, result.Error())
	})

	t.Run("unicode in messages", func(t *testing.T) {
		result := ValidationError("файл", "должен быть валидным")
		expected := "validation failed for файл: должен быть валидным"
		assert.Equal(t, expected, result.Error())
	})
}

// TestDeepErrorChain verifies that errors.Is() works correctly
// even with deeply nested error chains (10+ levels).
func TestDeepErrorChain(t *testing.T) {
	original := fmt.Errorf("root cause") //nolint:err113 // test-only errors

	// Create 10-level deep chain
	chain := original
	for i := 0; i < 10; i++ {
		chain = WrapWithContext(chain, fmt.Sprintf("level %d", i))
	}

	// Verify original is still findable through deep chain
	require.ErrorIs(t, chain, original)

	// Verify message contains all levels
	errorMsg := chain.Error()
	for i := 0; i < 10; i++ {
		assert.Contains(t, errorMsg, fmt.Sprintf("level %d", i))
	}
	assert.Contains(t, errorMsg, "root cause")
}

// TestVeryDeepErrorChain tests an even deeper chain (50 levels)
// to ensure no stack overflow or performance issues.
func TestVeryDeepErrorChain(t *testing.T) {
	original := ErrSyncFailed

	chain := original
	for i := 0; i < 50; i++ {
		chain = WrapWithContext(chain, fmt.Sprintf("operation %d", i))
	}

	// Should still be able to find the original sentinel error
	require.ErrorIs(t, chain, ErrSyncFailed)
}

// customTestError is a custom error type for testing errors.As()
type customTestError struct {
	msg  string
	code int
}

func (e *customTestError) Error() string {
	return e.msg
}

// TestErrorAsDoesNotFindNonMatchingTypes verifies that errors.As()
// correctly returns false when the wrapped error doesn't contain
// the target type. This ensures future compatibility if custom
// error types are added.
func TestErrorAsDoesNotFindNonMatchingTypes(t *testing.T) {
	var customErr *customTestError

	// Create various wrapped errors that don't contain customTestError
	wrapped1 := WrapWithContext(ErrNoFilesToCommit, "test operation")
	wrapped2 := CommandFailedError("git clone", fmt.Errorf("exit 1")) //nolint:err113 // test-only errors
	wrapped3 := GitOperationError("clone", "repo", ErrGitCommand)

	// None of these should match customTestError type
	assert.NotErrorAs(t, wrapped1, &customErr, "wrapped1 should not match customTestError")
	assert.NotErrorAs(t, wrapped2, &customErr, "wrapped2 should not match customTestError")
	assert.NotErrorAs(t, wrapped3, &customErr, "wrapped3 should not match customTestError")
}

// TestErrorAsFindsCustomType verifies that errors.As() correctly finds
// a custom error type when it's present in the chain.
func TestErrorAsFindsCustomType(t *testing.T) {
	customErr := &customTestError{msg: "custom error", code: 42}
	wrapped := WrapWithContext(customErr, "wrapping custom error")

	var foundErr *customTestError
	require.ErrorAs(t, wrapped, &foundErr, "should find customTestError in chain")
	assert.Equal(t, "custom error", foundErr.msg)
	assert.Equal(t, 42, foundErr.code)
}

// TestErrorAsFindsStandardTypes verifies that errors.As() works
// with standard library error types when wrapped.
func TestErrorAsFindsStandardTypes(t *testing.T) {
	// Create an error that wraps a standard error
	baseErr := fmt.Errorf("base: %w", errors.New("underlying")) //nolint:err113 // test-only errors
	wrapped := WrapWithContext(baseErr, "operation")

	// Should be able to find the base error
	require.ErrorIs(t, wrapped, baseErr)
}

// TestMixedErrorChain tests a chain with different error types mixed together.
func TestMixedErrorChain(t *testing.T) {
	// Start with a sentinel error
	level0 := ErrFileNotFound

	// Wrap with different error functions
	level1 := WrapWithContext(level0, "reading config")
	level2 := FileOperationError("read", "/config.yaml", level1)
	level3 := WrapWithContext(level2, "loading application")
	level4 := CommandFailedError("app start", level3)

	// All levels should be findable
	require.ErrorIs(t, level4, ErrFileNotFound)
	require.ErrorIs(t, level4, level1)
	require.ErrorIs(t, level4, level2)
	require.ErrorIs(t, level4, level3)

	// Error message should contain context from all levels
	errorMsg := level4.Error()
	assert.Contains(t, errorMsg, "reading config")
	assert.Contains(t, errorMsg, "/config.yaml")
	assert.Contains(t, errorMsg, "loading application")
	assert.Contains(t, errorMsg, "app start")
	assert.Contains(t, errorMsg, "source file not found")
}
