//nolint:err113 // Test files are allowed to use dynamic errors for testing purposes
package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TransformErrorTestSuite provides a test suite for transform error handling
type TransformErrorTestSuite struct {
	suite.Suite
}

// TestTransformErrorTestSuite runs the test suite
func TestTransformErrorTestSuite(t *testing.T) {
	suite.Run(t, new(TransformErrorTestSuite))
}

// TestNewTransformError tests creating a new transform error
func (suite *TransformErrorTestSuite) TestNewTransformError() {
	baseErr := errors.New("test error")
	filePath := "/path/to/file.go"
	sourceRepo := "org/source"
	targetRepo := "org/target"
	transformType := "repo_name"

	te := NewTransformError(baseErr, filePath, sourceRepo, targetRepo, transformType)

	suite.Equal(baseErr, te.err)
	suite.Equal(filePath, te.filePath)
	suite.Equal(sourceRepo, te.sourceRepo)
	suite.Equal(targetRepo, te.targetRepo)
	suite.Equal(transformType, te.transformType)
	suite.False(te.isFromDirectory)
	suite.Empty(te.relativePath)
	suite.NotZero(te.timestamp)
	suite.NotNil(te.metadata)
}

// TestNewDirectoryTransformError tests creating a directory transform error
func (suite *TransformErrorTestSuite) TestNewDirectoryTransformError() {
	baseErr := errors.New("directory test error")
	filePath := "/full/path/to/file.go"
	relativePath := "relative/file.go"
	sourceRepo := "org/source"
	targetRepo := "org/target"
	transformType := "template"

	te := NewDirectoryTransformError(baseErr, filePath, relativePath, sourceRepo, targetRepo, transformType)

	suite.Equal(baseErr, te.err)
	suite.Equal(filePath, te.filePath)
	suite.Equal(relativePath, te.relativePath)
	suite.True(te.isFromDirectory)
}

// TestTransformErrorError tests the Error method
func (suite *TransformErrorTestSuite) TestTransformErrorError() {
	baseErr := errors.New("base error")
	te := NewTransformError(baseErr, "/path/file.go", "org/source", "org/target", "repo_name")

	errorStr := te.Error()
	suite.Contains(errorStr, "transform failed: base error")
	suite.Contains(errorStr, "file: /path/file.go")
	suite.Contains(errorStr, "repos: org/source -> org/target")
	suite.Contains(errorStr, "transform: repo_name")
	suite.Contains(errorStr, "category:")
}

// TestTransformErrorDirectoryError tests the Error method for directory operations
func (suite *TransformErrorTestSuite) TestTransformErrorDirectoryError() {
	baseErr := errors.New("directory error")
	te := NewDirectoryTransformError(baseErr, "/full/path.go", "relative/path.go", "org/source", "org/target", "template")

	errorStr := te.Error()
	suite.Contains(errorStr, "file: /full/path.go (relative: relative/path.go)")
}

// TestTransformErrorUnwrap tests error unwrapping
func (suite *TransformErrorTestSuite) TestTransformErrorUnwrap() {
	baseErr := errors.New("wrapped error")
	te := NewTransformError(baseErr, "/path/file.go", "org/source", "org/target", "repo_name")

	suite.Equal(baseErr, te.Unwrap())
	suite.ErrorIs(te, baseErr)
}

// TestTransformErrorIs tests error comparison
func (suite *TransformErrorTestSuite) TestTransformErrorIs() {
	baseErr := errors.New("base error")
	te1 := NewTransformError(baseErr, "/path/file.go", "org/source", "org/target", "repo_name")
	te2 := NewTransformError(errors.New("other error"), "/path/file.go", "org/source", "org/target", "repo_name")

	// Should match if categories are the same
	suite.Require().ErrorIs(te1, te2)

	// Should match underlying error
	suite.ErrorIs(te1, baseErr)
}

// TestTransformErrorMethods tests various getter methods
func (suite *TransformErrorTestSuite) TestTransformErrorMethods() {
	baseErr := errors.New("test error")
	filePath := "/path/to/file.go"
	relativePath := "relative/file.go"
	sourceRepo := "org/source"
	targetRepo := "org/target"
	transformType := "template"
	duration := 100 * time.Millisecond

	te := NewDirectoryTransformError(baseErr, filePath, relativePath, sourceRepo, targetRepo, transformType)
	_ = te.WithDuration(duration).WithMetadata("key", "value")

	suite.Equal(filePath, te.GetFilePath())
	suite.Equal(relativePath, te.GetRelativePath())
	suite.Equal(sourceRepo, te.GetSourceRepo())
	suite.Equal(targetRepo, te.GetTargetRepo())
	suite.Equal(transformType, te.GetTransformType())
	suite.True(te.IsFromDirectory())
	suite.Equal(duration, te.GetDuration())
	suite.Equal("value", te.GetMetadata()["key"])
	suite.NotZero(te.GetTimestamp())
}

// TestCategorizeError tests error categorization
func (suite *TransformErrorTestSuite) TestCategorizeError() {
	testCases := []struct {
		name     string
		err      error
		expected TransformErrorCategory
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: CategoryGenericTransform,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: CategoryContext,
		},
		{
			name:     "context timeout",
			err:      context.DeadlineExceeded,
			expected: CategoryTimeout,
		},
		{
			name:     "binary file error",
			err:      errors.New("binary file detected"),
			expected: CategoryBinaryFile,
		},
		{
			name:     "template parse error",
			err:      errors.New("template parsing failed"),
			expected: CategoryTemplateParse,
		},
		{
			name:     "variable substitution error",
			err:      errors.New("variable substitution failed"),
			expected: CategoryVariableSubstitution,
		},
		{
			name:     "repository format error",
			err:      errors.New("invalid repository format"),
			expected: CategoryRepoNameFormat,
		},
		{
			name:     "file system error",
			err:      errors.New("no such file or directory"),
			expected: CategoryFileSystem,
		},
		{
			name:     "timeout error",
			err:      errors.New("operation timeout"),
			expected: CategoryTimeout,
		},
		{
			name:     "generic error",
			err:      errors.New("some other error"),
			expected: CategoryGenericTransform,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := categorizeError(tc.err)
			suite.Equal(tc.expected, result)
		})
	}
}

// TestIsRecoverableError tests error recoverability
func (suite *TransformErrorTestSuite) TestIsRecoverableError() {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: true,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "file system error",
			err:      errors.New("no such file"),
			expected: false,
		},
		{
			name:     "template error",
			err:      errors.New("template parsing failed"),
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := isRecoverableError(tc.err)
			suite.Equal(tc.expected, result)
		})
	}
}

// TestIsRetryableError tests error retry ability
func (suite *TransformErrorTestSuite) TestIsRetryableError() {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "timeout error",
			err:      errors.New("timeout occurred"),
			expected: true,
		},
		{
			name:     "file system error",
			err:      errors.New("permission denied"),
			expected: true,
		},
		{
			name:     "template error",
			err:      errors.New("template parsing failed"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := isRetryableError(tc.err)
			suite.Equal(tc.expected, result)
		})
	}
}

// TestTransformErrorCollection tests error collection functionality
func (suite *TransformErrorTestSuite) TestTransformErrorCollection() {
	collection := NewTransformCollectionError()

	// Test empty collection
	suite.False(collection.HasErrors())
	suite.Equal("no transform errors", collection.Error())

	// Add successful operations
	collection.AddSuccess()
	collection.AddSuccess()

	// Add errors
	err1 := NewTransformError(errors.New("error 1"), "/file1.go", "org/source", "org/target", "repo_name")
	err2 := NewTransformError(context.Canceled, "/file2.go", "org/source", "org/target", "template")
	err3 := NewTransformError(errors.New("timeout"), "/file3.go", "org/source", "org/target", "variable")

	collection.Add(err1)
	collection.Add(err2)
	collection.Add(err3)

	suite.True(collection.HasErrors())
	suite.Len(collection.GetErrors(), 3)

	// Test filtering by category
	contextErrors := collection.GetErrorsByCategory(CategoryContext)
	suite.Len(contextErrors, 1)
	suite.Equal(err2, contextErrors[0])

	// Test summary
	summary := collection.GetSummary()
	suite.Equal(3, summary.Total)
	suite.Equal(2, summary.Successful)
	suite.Positive(summary.Recoverable)
	suite.NotEmpty(summary.Categories)
	suite.NotEmpty(summary.FileErrors)

	// Test error string
	errorStr := collection.Error()
	suite.Contains(errorStr, "transform errors")
	suite.Contains(errorStr, "3 total")
	suite.Contains(errorStr, "2 successful")
}

// TestGetRecoveryStrategy tests recovery strategy determination
func (suite *TransformErrorTestSuite) TestGetRecoveryStrategy() {
	testCases := []struct {
		name     string
		err      *TransformError
		expected TransformRecoveryStrategy
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: RecoveryStrategyNone,
		},
		{
			name:     "context error",
			err:      NewTransformError(context.Canceled, "/file.go", "org/source", "org/target", "repo_name"),
			expected: RecoveryStrategyNone,
		},
		{
			name:     "binary file error",
			err:      NewTransformError(errors.New("binary file"), "/file.bin", "org/source", "org/target", "repo_name"),
			expected: RecoveryStrategySkipFile,
		},
		{
			name:     "timeout error",
			err:      NewTransformError(errors.New("timeout"), "/file.go", "org/source", "org/target", "repo_name"),
			expected: RecoveryStrategyRetry,
		},
		{
			name:     "template error",
			err:      NewTransformError(errors.New("template error"), "/file.go", "org/source", "org/target", "template"),
			expected: RecoveryStrategyUseOriginal,
		},
		{
			name:     "file system error",
			err:      NewTransformError(errors.New("no such file"), "/file.go", "org/source", "org/target", "repo_name"),
			expected: RecoveryStrategyRetry, // File system errors are retryable
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			strategy := GetRecoveryStrategy(tc.err)
			suite.Equal(tc.expected, strategy)
		})
	}
}

// TestValidateTransformContext tests transform context validation
func (suite *TransformErrorTestSuite) TestValidateTransformContext() {
	testCases := []struct {
		name       string
		filePath   string
		sourceRepo string
		targetRepo string
		expectErr  bool
		errType    error
	}{
		{
			name:       "valid context",
			filePath:   "/path/to/file.go",
			sourceRepo: "org/source",
			targetRepo: "org/target",
			expectErr:  false,
		},
		{
			name:       "empty file path",
			filePath:   "",
			sourceRepo: "org/source",
			targetRepo: "org/target",
			expectErr:  true,
		},
		{
			name:       "empty source repo",
			filePath:   "/path/to/file.go",
			sourceRepo: "",
			targetRepo: "org/target",
			expectErr:  true,
		},
		{
			name:       "empty target repo",
			filePath:   "/path/to/file.go",
			sourceRepo: "org/source",
			targetRepo: "",
			expectErr:  true,
		},
		{
			name:       "invalid source repo format",
			filePath:   "/path/to/file.go",
			sourceRepo: "invalid-format",
			targetRepo: "org/target",
			expectErr:  true,
		},
		{
			name:       "invalid target repo format",
			filePath:   "/path/to/file.go",
			sourceRepo: "org/source",
			targetRepo: "invalid/format/too/many/parts",
			expectErr:  true,
		},
		{
			name:       "path traversal",
			filePath:   "/path/../../../etc/passwd",
			sourceRepo: "org/source",
			targetRepo: "org/target",
			expectErr:  true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := ValidateTransformContext(tc.filePath, tc.sourceRepo, tc.targetRepo)
			if tc.expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
}

// TestWrapTransformError tests error wrapping
func (suite *TransformErrorTestSuite) TestWrapTransformError() {
	// Test nil error
	suite.Require().NoError(WrapTransformError(nil, "/file.go", "org/source", "org/target", "repo_name"))

	// Test wrapping regular error
	baseErr := errors.New("base error")
	wrapped := WrapTransformError(baseErr, "/file.go", "org/source", "org/target", "repo_name")

	var te *TransformError
	suite.Require().ErrorAs(wrapped, &te)
	suite.Equal(baseErr, te.err)

	// Test wrapping already wrapped error
	alreadyWrapped := WrapTransformError(wrapped, "/file.go", "org/source", "org/target", "repo_name")
	suite.Equal(wrapped, alreadyWrapped)
}

// TestWrapDirectoryTransformError tests directory error wrapping
func (suite *TransformErrorTestSuite) TestWrapDirectoryTransformError() {
	// Test nil error
	suite.Require().NoError(WrapDirectoryTransformError(nil, "/file.go", "rel.go", "org/source", "org/target", "template"))

	// Test wrapping regular error
	baseErr := errors.New("directory error")
	wrapped := WrapDirectoryTransformError(baseErr, "/full/file.go", "rel/file.go", "org/source", "org/target", "template")

	var te *TransformError
	suite.Require().ErrorAs(wrapped, &te)
	suite.True(te.IsFromDirectory())
	suite.Equal("rel/file.go", te.GetRelativePath())

	// Test wrapping already wrapped error
	regularTE := NewTransformError(errors.New("regular"), "/file.go", "org/source", "org/target", "repo_name")
	dirWrapped := WrapDirectoryTransformError(regularTE, "/full/file.go", "rel/file.go", "org/source", "org/target", "template")

	var dirTE *TransformError
	suite.Require().ErrorAs(dirWrapped, &dirTE)
	suite.True(dirTE.IsFromDirectory())
	suite.Equal("rel/file.go", dirTE.GetRelativePath())
}

// TestIsValidRepoFormat tests repository format validation
func TestIsValidRepoFormat(t *testing.T) {
	testCases := []struct {
		name     string
		repo     string
		expected bool
	}{
		{
			name:     "valid format",
			repo:     "org/repo",
			expected: true,
		},
		{
			name:     "empty string",
			repo:     "",
			expected: false,
		},
		{
			name:     "no slash",
			repo:     "orgRepo",
			expected: false,
		},
		{
			name:     "too many parts",
			repo:     "org/repo/extra",
			expected: false,
		},
		{
			name:     "empty org",
			repo:     "/repo",
			expected: false,
		},
		{
			name:     "empty repo",
			repo:     "org/",
			expected: false,
		},
		{
			name:     "contains space",
			repo:     "org name/repo",
			expected: false,
		},
		{
			name:     "contains tab",
			repo:     "org/repo\tname",
			expected: false,
		},
		{
			name:     "contains newline",
			repo:     "org/repo\n",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidRepoFormat(tc.repo)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestIsBinaryFileExtension tests binary file extension detection
func TestIsBinaryFileExtension(t *testing.T) {
	testCases := []struct {
		name      string
		extension string
		expected  bool
	}{
		{
			name:      "go file",
			extension: ".go",
			expected:  false,
		},
		{
			name:      "text file",
			extension: ".txt",
			expected:  false,
		},
		{
			name:      "executable",
			extension: ".exe",
			expected:  true,
		},
		{
			name:      "image",
			extension: ".jpg",
			expected:  true,
		},
		{
			name:      "pdf",
			extension: ".pdf",
			expected:  true,
		},
		{
			name:      "archive",
			extension: ".zip",
			expected:  true,
		},
		{
			name:      "font",
			extension: ".woff",
			expected:  true,
		},
		{
			name:      "unknown extension",
			extension: ".unknown",
			expected:  false,
		},
		{
			name:      "no extension",
			extension: "",
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsBinaryFileExtension(tc.extension)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestGetFileExtension tests file extension extraction
func TestGetFileExtension(t *testing.T) {
	testCases := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "go file",
			filePath: "/path/to/file.go",
			expected: ".go",
		},
		{
			name:     "multiple dots",
			filePath: "/path/to/file.test.go",
			expected: ".go",
		},
		{
			name:     "uppercase extension",
			filePath: "/path/to/FILE.TXT",
			expected: ".txt",
		},
		{
			name:     "no extension",
			filePath: "/path/to/file",
			expected: "",
		},
		{
			name:     "hidden file with extension",
			filePath: "/path/to/.gitignore",
			expected: ".gitignore",
		},
		{
			name:     "hidden file with double extension",
			filePath: "/path/to/.env.local",
			expected: ".local",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := GetFileExtension(tc.filePath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// BenchmarkTransformError benchmarks transform error creation and operations
func BenchmarkTransformError(b *testing.B) {
	baseErr := errors.New("benchmark error")

	b.Run("NewTransformError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewTransformError(baseErr, "/path/file.go", "org/source", "org/target", "repo_name")
		}
	})

	b.Run("ErrorString", func(b *testing.B) {
		te := NewTransformError(baseErr, "/path/file.go", "org/source", "org/target", "repo_name")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = te.Error()
		}
	})

	b.Run("CategorizeError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = categorizeError(baseErr)
		}
	})
}

// TestTransformErrorIntegration tests integration with existing error patterns
func (suite *TransformErrorTestSuite) TestTransformErrorIntegration() {
	// Test integration with internal errors
	te := NewTransformError(internalerrors.ErrTransformNotFound, "/file.go", "org/source", "org/target", "repo_name")
	suite.Require().ErrorIs(te, internalerrors.ErrTransformNotFound)

	// Test context integration
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	contextTE := NewTransformError(ctx.Err(), "/file.go", "org/source", "org/target", "template")
	suite.Equal(CategoryContext, contextTE.GetCategory())
	suite.False(contextTE.IsRecoverable())
	suite.False(contextTE.ShouldRetry())
}

// ExampleTransformError demonstrates basic usage of TransformError
func ExampleTransformError() {
	// Create a transform error
	baseErr := errors.New("template parsing failed")
	te := NewTransformError(baseErr, "/templates/service.yaml", "org/template", "org/service", "template")

	// Add timing and metadata
	_ = te.WithDuration(50*time.Millisecond).WithMetadata("template_vars", 5)

	// Check error properties
	_ = te.IsRecoverable() // true
	_ = te.ShouldRetry()   // false
	_ = te.GetCategory()   // CategoryTemplateParse

	// Get recovery strategy
	strategy := GetRecoveryStrategy(te)
	_ = strategy // RecoveryStrategyUseOriginal
}

// ExampleTransformCollectionError demonstrates error collection usage
func ExampleTransformCollectionError() {
	collection := NewTransformCollectionError()

	// Record successful operations
	collection.AddSuccess()
	collection.AddSuccess()

	// Add errors
	err1 := NewTransformError(errors.New("timeout"), "/file1.go", "org/source", "org/target", "repo_name")
	err2 := NewTransformError(errors.New("binary file"), "/file2.bin", "org/source", "org/target", "repo_name")

	collection.Add(err1)
	collection.Add(err2)

	// Get summary
	summary := collection.GetSummary()
	_ = summary.Total      // 2
	_ = summary.Successful // 2
	_ = summary.Retryable  // 1 (timeout is retryable)
}
