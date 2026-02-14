//go:build go1.18

package errors //nolint:revive,nolintlint // internal test package, name conflict intentional

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzWrapWithContext tests WrapWithContext with arbitrary string inputs.
// It verifies that the function never panics and always preserves error chain.
func FuzzWrapWithContext(f *testing.F) {
	// Add seed corpus
	f.Add("normal operation")
	f.Add("")
	f.Add("with\nnewline")
	f.Add("with\ttab")
	f.Add("with\x00null")
	f.Add(strings.Repeat("a", 10000))
	f.Add("special chars: @#$%^&*()")
	f.Add("unicode: 日本語 中文 한국어") //nolint:gosmopolitan // intentional unicode test data
	f.Add("emoji: 🚀🎉💻")
	f.Add("path/like/string")
	f.Add("quote's and \"doubles\"")

	baseErr := errors.New("base error") //nolint:err113 // test-only error for fuzz testing
	f.Fuzz(func(t *testing.T, operation string) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(operation) > 5000 {
			t.Skipf("Input too large: %d bytes (limit: 5000)", len(operation))
		}

		// Should never panic
		result := WrapWithContext(baseErr, operation)

		// Should always return non-nil for non-nil input
		require.Error(t, result)

		// Should always preserve error chain
		require.ErrorIs(t, result, baseErr)

		// Error message should be retrievable without panic
		_ = result.Error()
	})
}

// FuzzInvalidFieldError tests InvalidFieldError with arbitrary field and value strings.
func FuzzInvalidFieldError(f *testing.F) {
	// Add seed corpus
	f.Add("field", "value")
	f.Add("", "")
	f.Add("field\x00with\x00nulls", "value\x00with\x00nulls")
	f.Add("field with spaces", "value with spaces")
	f.Add(strings.Repeat("f", 5000), strings.Repeat("v", 5000))
	f.Add("field:with:colons", "value:with:colons")
	f.Add("unicode_field_日本語", "unicode_value_中文") //nolint:gosmopolitan // intentional unicode test data

	f.Fuzz(func(t *testing.T, field, value string) {
		// Skip long inputs to avoid timeout in CI with expensive error formatting
		if len(field)+len(value) > 5000 {
			t.Skipf("Input too large: %d bytes (limit: 5000)", len(field)+len(value))
		}

		// Should never panic
		result := InvalidFieldError(field, value)

		// Should always return non-nil
		require.Error(t, result)

		// Error message should be retrievable without panic
		msg := result.Error()

		// Message should contain expected prefix
		require.True(t, strings.HasPrefix(msg, "invalid field:"))
	})
}

// FuzzValidationError tests ValidationError with arbitrary inputs.
func FuzzValidationError(f *testing.F) {
	f.Add("item", "reason")
	f.Add("", "")
	f.Add("item\nwith\nnewlines", "reason\nwith\nnewlines")
	f.Add(strings.Repeat("i", 10000), strings.Repeat("r", 10000))

	f.Fuzz(func(t *testing.T, item, reason string) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(item)+len(reason) > 5000 {
			t.Skipf("Input too large: %d bytes (limit: 5000)", len(item)+len(reason))
		}

		result := ValidationError(item, reason)
		require.Error(t, result)
		require.True(t, strings.HasPrefix(result.Error(), "validation failed"))
	})
}

// FuzzPathTraversalError tests PathTraversalError with arbitrary path strings.
func FuzzPathTraversalError(f *testing.F) {
	f.Add("../../../etc/passwd")
	f.Add("")
	f.Add("..")
	f.Add("/absolute/path")
	f.Add("relative/path")
	f.Add("path with spaces/file.txt")
	f.Add("path\x00with\x00null")
	f.Add(strings.Repeat("a/", 1000))

	f.Fuzz(func(t *testing.T, path string) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(path) > 5000 {
			t.Skipf("Input too large: %d bytes (limit: 5000)", len(path))
		}

		result := PathTraversalError(path)
		require.Error(t, result)
		require.True(t, strings.HasPrefix(result.Error(), "path traversal detected"))
	})
}

// FuzzGitOperationError tests GitOperationError with arbitrary inputs.
func FuzzGitOperationError(f *testing.F) {
	f.Add("clone", "user/repo")
	f.Add("", "")
	f.Add("checkout", "feature-branch")
	f.Add("push", "origin main")
	f.Add("operation\x00null", "context\x00null")
	f.Add(strings.Repeat("o", 5000), strings.Repeat("c", 5000))

	baseErr := errors.New("git error") //nolint:err113 // test-only error for fuzz testing
	f.Fuzz(func(t *testing.T, operation, context string) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(operation)+len(context) > 5000 {
			t.Skipf("Input too large: %d bytes (limit: 5000)", len(operation)+len(context))
		}

		result := GitOperationError(operation, context, baseErr)
		require.Error(t, result)
		require.ErrorIs(t, result, baseErr)
		require.True(t, strings.HasPrefix(result.Error(), "git operation failed"))
	})
}

// FuzzFileOperationError tests FileOperationError with arbitrary inputs.
func FuzzFileOperationError(f *testing.F) {
	f.Add("read", "/path/to/file.txt")
	f.Add("", "")
	f.Add("write", "/path with spaces/file.txt")
	f.Add("operation", "path\x00with\x00null")
	f.Add(strings.Repeat("o", 5000), strings.Repeat("p", 5000))

	baseErr := errors.New("file error") //nolint:err113 // test-only error for fuzz testing
	f.Fuzz(func(t *testing.T, operation, path string) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(operation)+len(path) > 5000 {
			t.Skipf("Input too large: %d bytes (limit: 5000)", len(operation)+len(path))
		}

		result := FileOperationError(operation, path, baseErr)
		require.Error(t, result)
		require.ErrorIs(t, result, baseErr)
		require.True(t, strings.HasPrefix(result.Error(), "file operation failed"))
	})
}

// FuzzBatchOperationError tests BatchOperationError with arbitrary inputs.
// This is particularly important to verify the range validation logic.
func FuzzBatchOperationError(f *testing.F) {
	// Valid ranges
	f.Add("process", 0, 10)
	f.Add("validate", 5, 15)
	f.Add("", 0, 1)

	// Invalid ranges (should produce "invalid range" message)
	f.Add("process", 10, 5)  // start > end
	f.Add("process", 0, 0)   // zero range
	f.Add("process", -1, 5)  // negative start
	f.Add("process", 0, -1)  // negative end
	f.Add("process", -5, -1) // both negative

	// Edge cases
	f.Add("op", 0, 1)             // single item
	f.Add("op", 1000000, 1000001) // large numbers
	f.Add("op", 2147483647, 0)    // max int with 0

	baseErr := errors.New("batch error") //nolint:err113 // test-only error for fuzz testing
	f.Fuzz(func(t *testing.T, operation string, start, end int) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(operation) > 5000 {
			t.Skipf("Input too large: %d bytes (limit: 5000)", len(operation))
		}

		// Should never panic regardless of input
		result := BatchOperationError(operation, start, end, baseErr)

		// Should always return non-nil for non-nil error input
		require.Error(t, result)

		// Should always preserve error chain
		require.ErrorIs(t, result, baseErr)

		// Error message should be retrievable without panic
		msg := result.Error()

		// Message should have expected prefix
		require.True(t, strings.HasPrefix(msg, "batch operation failed"))

		// Verify invalid range detection
		if start < 0 || end < 0 || start > end {
			require.Contains(t, msg, "invalid range")
		}
	})
}

// FuzzAPIResponseError tests APIResponseError with arbitrary status codes.
func FuzzAPIResponseError(f *testing.F) {
	// Valid status codes
	f.Add(200, "OK")
	f.Add(404, "Not Found")
	f.Add(500, "Internal Server Error")
	f.Add(100, "Continue")
	f.Add(599, "Custom")

	// Invalid status codes
	f.Add(-1, "negative")
	f.Add(0, "zero")
	f.Add(99, "too low")
	f.Add(600, "too high")
	f.Add(2147483647, "max int")
	f.Add(-2147483648, "min int")

	f.Fuzz(func(t *testing.T, statusCode int, message string) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(message) > 1000 {
			t.Skipf("Input too large: %d bytes (limit: 1000)", len(message))
		}

		// Should never panic
		result := APIResponseError(statusCode, message)

		// Should always return non-nil
		require.Error(t, result)

		// Error message should be retrievable without panic
		msg := result.Error()

		// Message should have expected prefix
		require.True(t, strings.HasPrefix(msg, "API response error"))

		// Verify invalid status detection
		if statusCode < 100 || statusCode > 599 {
			require.Contains(t, msg, "invalid status")
		}
	})
}

// FuzzCommandFailedError tests CommandFailedError with arbitrary command strings.
func FuzzCommandFailedError(f *testing.F) {
	f.Add("git clone")
	f.Add("")
	f.Add("command with spaces and 'quotes'")
	f.Add("command\x00with\x00null")
	f.Add(strings.Repeat("c", 10000))

	baseErr := errors.New("command error") //nolint:err113 // test-only error for fuzz testing
	f.Fuzz(func(t *testing.T, cmd string) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(cmd) > 5000 {
			t.Skipf("Input too large: %d bytes (limit: 5000)", len(cmd))
		}

		result := CommandFailedError(cmd, baseErr)
		require.Error(t, result)
		require.ErrorIs(t, result, baseErr)
		require.True(t, strings.HasPrefix(result.Error(), "command failed"))
	})
}

// FuzzFormatError tests FormatError with arbitrary inputs.
func FuzzFormatError(f *testing.F) {
	f.Add("field", "value", "expected format")
	f.Add("", "", "")
	f.Add("repository name", "invalid-repo", "org/repo")
	f.Add("field\x00null", "value\x00null", "format\x00null")

	f.Fuzz(func(t *testing.T, field, value, expectedFormat string) {
		// Skip extremely long inputs to avoid resource exhaustion
		if len(field)+len(value)+len(expectedFormat) > 5000 {
			t.Skipf("Input too large: %d bytes (limit: 5000)", len(field)+len(value)+len(expectedFormat))
		}

		result := FormatError(field, value, expectedFormat)
		require.Error(t, result)
		require.True(t, strings.HasPrefix(result.Error(), "invalid format"))
	})
}
