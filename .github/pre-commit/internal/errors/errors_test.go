package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ErrorTestSuite struct {
	suite.Suite
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}

// TestCommonErrors tests that all common errors are properly defined
func (s *ErrorTestSuite) TestCommonErrors() {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrChecksFailed", ErrChecksFailed, "checks failed"},
		{"ErrNoChecksToRun", ErrNoChecksToRun, "no checks to run"},
		{"ErrEnvFileNotFound", ErrEnvFileNotFound, ".github/.env.shared not found in any parent directory"},
		{"ErrRepositoryRootNotFound", ErrRepositoryRootNotFound, "unable to determine repository root"},
		{"ErrToolNotFound", ErrToolNotFound, "required tool not found"},
		{"ErrLintingIssues", ErrLintingIssues, "linting issues found"},
		{"ErrNotTidy", ErrNotTidy, "go.mod or go.sum are not tidy"},
		{"ErrWhitespaceIssues", ErrWhitespaceIssues, "whitespace issues found"},
		{"ErrEOFIssues", ErrEOFIssues, "EOF issues found"},
		{"ErrMakeTargetNotFound", ErrMakeTargetNotFound, "make target not found"},
		{"ErrToolExecutionFailed", ErrToolExecutionFailed, "tool execution failed"},
		{"ErrGracefulSkip", ErrGracefulSkip, "check gracefully skipped"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Error(tt.err)
			s.Equal(tt.expected, tt.err.Error())
		})
	}
}

// TestCheckErrorConstructor tests the CheckError constructor
func (s *ErrorTestSuite) TestCheckErrorConstructor() {
	baseErr := errors.New("base error")
	message := "something went wrong"
	suggestion := "try this fix"

	checkErr := NewCheckError(baseErr, message, suggestion)

	s.NotNil(checkErr)
	s.Equal(baseErr, checkErr.Err)
	s.Equal(message, checkErr.Message)
	s.Equal(suggestion, checkErr.Suggestion)
	s.False(checkErr.CanSkip)
}

// TestCheckErrorError tests the Error method
func (s *ErrorTestSuite) TestCheckErrorError() {
	tests := []struct {
		name     string
		checkErr *CheckError
		expected string
	}{
		{
			name: "message takes precedence",
			checkErr: &CheckError{
				Err:     errors.New("base error"),
				Message: "custom message",
			},
			expected: "custom message",
		},
		{
			name: "falls back to base error",
			checkErr: &CheckError{
				Err: errors.New("base error"),
			},
			expected: "base error",
		},
		{
			name:     "unknown error when both are empty",
			checkErr: &CheckError{},
			expected: "unknown error",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.expected, tt.checkErr.Error())
		})
	}
}

// TestCheckErrorUnwrap tests the Unwrap method
func (s *ErrorTestSuite) TestCheckErrorUnwrap() {
	baseErr := errors.New("base error")
	checkErr := &CheckError{Err: baseErr}

	unwrapped := checkErr.Unwrap()
	s.Equal(baseErr, unwrapped)
}

// TestCheckErrorIs tests the Is method
func (s *ErrorTestSuite) TestCheckErrorIs() {
	baseErr := ErrToolNotFound
	checkErr := &CheckError{Err: baseErr}

	s.True(checkErr.Is(ErrToolNotFound))
	s.False(checkErr.Is(ErrLintingIssues))
}

// TestNewToolNotFoundError tests the tool not found error constructor
func (s *ErrorTestSuite) TestNewToolNotFoundError() {
	tool := "golangci-lint"
	alternative := "try installing with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

	err := NewToolNotFoundError(tool, alternative)

	s.NotNil(err)
	s.True(err.Is(ErrToolNotFound))
	s.Equal("golangci-lint not found", err.Message)
	s.Equal(alternative, err.Suggestion)
	s.True(err.CanSkip)
}

// TestNewMakeTargetNotFoundError tests the make target not found error constructor
func (s *ErrorTestSuite) TestNewMakeTargetNotFoundError() {
	target := "lint"
	alternative := "try running golangci-lint directly"

	err := NewMakeTargetNotFoundError(target, alternative)

	s.NotNil(err)
	s.True(err.Is(ErrMakeTargetNotFound))
	s.Equal("make target 'lint' not found", err.Message)
	s.Equal(alternative, err.Suggestion)
	s.True(err.CanSkip)
}

// TestNewToolExecutionError tests the tool execution error constructor
func (s *ErrorTestSuite) TestNewToolExecutionError() {
	command := "golangci-lint run"
	output := "some error output"
	suggestion := "fix the issues found"

	err := NewToolExecutionError(command, output, suggestion)

	s.NotNil(err)
	s.True(err.Is(ErrToolExecutionFailed))
	s.Equal("command 'golangci-lint run' failed", err.Message)
	s.Equal(command, err.Command)
	s.Equal(output, err.Output)
	s.Equal(suggestion, err.Suggestion)
	s.False(err.CanSkip)
}

// TestNewGracefulSkipError tests the graceful skip error constructor
func (s *ErrorTestSuite) TestNewGracefulSkipError() {
	reason := "make target not available"

	err := NewGracefulSkipError(reason)

	s.NotNil(err)
	s.True(err.Is(ErrGracefulSkip))
	s.Equal(reason, err.Message)
	s.Equal("This check was skipped to allow other checks to continue", err.Suggestion)
	s.True(err.CanSkip)
}

// TestCheckErrorChaining tests error chaining and wrapping
func (s *ErrorTestSuite) TestCheckErrorChaining() {
	originalErr := errors.New("original error")
	wrappedErr := NewCheckError(originalErr, "wrapped message", "fix suggestion")

	// Test that we can unwrap to the original error
	s.Require().ErrorIs(wrappedErr, originalErr)
	s.Equal(originalErr, errors.Unwrap(wrappedErr))

	// Test error chaining with standard library
	var checkErr *CheckError
	s.ErrorAs(wrappedErr, &checkErr)
	s.Equal(wrappedErr, checkErr)
}

// TestCheckErrorFields tests all fields of CheckError
func (s *ErrorTestSuite) TestCheckErrorFields() {
	err := &CheckError{
		Err:        ErrLintingIssues,
		Message:    "custom message",
		Suggestion: "fix the linting issues",
		Command:    "golangci-lint run",
		Output:     "error output",
		Files:      []string{"main.go", "test.go"},
		CanSkip:    true,
	}

	s.Equal(ErrLintingIssues, err.Err)
	s.Equal("custom message", err.Message)
	s.Equal("fix the linting issues", err.Suggestion)
	s.Equal("golangci-lint run", err.Command)
	s.Equal("error output", err.Output)
	s.Equal([]string{"main.go", "test.go"}, err.Files)
	s.True(err.CanSkip)
}

// Unit tests for edge cases
func TestCheckErrorNilWrapping(t *testing.T) {
	checkErr := &CheckError{Err: nil}
	assert.NoError(t, checkErr.Unwrap())
	assert.False(t, checkErr.Is(ErrToolNotFound))
}

func TestCheckErrorEmptyMessage(t *testing.T) {
	checkErr := &CheckError{
		Err:     ErrToolNotFound,
		Message: "",
	}
	assert.Equal(t, "required tool not found", checkErr.Error())
}

func TestErrorComparisons(t *testing.T) {
	// Test that our predefined errors are distinct
	assert.NotEqual(t, ErrChecksFailed, ErrNoChecksToRun)
	assert.NotEqual(t, ErrToolNotFound, ErrLintingIssues)
	assert.NotEqual(t, ErrMakeTargetNotFound, ErrToolExecutionFailed)
}

func TestErrorWrappingWithStandardLibrary(t *testing.T) {
	originalErr := ErrToolNotFound
	wrappedErr := NewCheckError(originalErr, "custom message", "fix it")

	// Test with errors.Is
	assert.ErrorIs(t, wrappedErr, ErrToolNotFound)
	assert.NotErrorIs(t, wrappedErr, ErrLintingIssues)

	// Test with errors.As
	var checkErr *CheckError
	assert.ErrorAs(t, wrappedErr, &checkErr)
	assert.Equal(t, "custom message", checkErr.Message)
}

func TestAllConstructors(t *testing.T) {
	tests := []struct {
		name        string
		constructor func() *CheckError
		expectedErr error
		canSkip     bool
	}{
		{
			name:        "NewCheckError",
			constructor: func() *CheckError { return NewCheckError(ErrLintingIssues, "msg", "suggestion") },
			expectedErr: ErrLintingIssues,
			canSkip:     false,
		},
		{
			name:        "NewToolNotFoundError",
			constructor: func() *CheckError { return NewToolNotFoundError("tool", "alt") },
			expectedErr: ErrToolNotFound,
			canSkip:     true,
		},
		{
			name:        "NewMakeTargetNotFoundError",
			constructor: func() *CheckError { return NewMakeTargetNotFoundError("target", "alt") },
			expectedErr: ErrMakeTargetNotFound,
			canSkip:     true,
		},
		{
			name:        "NewToolExecutionError",
			constructor: func() *CheckError { return NewToolExecutionError("cmd", "output", "suggestion") },
			expectedErr: ErrToolExecutionFailed,
			canSkip:     false,
		},
		{
			name:        "NewGracefulSkipError",
			constructor: func() *CheckError { return NewGracefulSkipError("reason") },
			expectedErr: ErrGracefulSkip,
			canSkip:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			assert.NotNil(t, err)
			assert.True(t, err.Is(tt.expectedErr))
			assert.Equal(t, tt.canSkip, err.CanSkip)
			assert.NotEmpty(t, err.Error())
		})
	}
}
