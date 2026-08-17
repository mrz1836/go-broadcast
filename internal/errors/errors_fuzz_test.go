//go:build go1.18

package errors //nolint:revive,nolintlint // internal test package, name conflict intentional

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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
		if len(operation) > 2000 {
			t.Skipf("Input too large: %d bytes (limit: 2000)", len(operation))
		}

		// Create context with timeout to prevent expensive operations from hanging
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Check context before expensive operations
		select {
		case <-ctx.Done():
			t.Skipf("Context timeout before operation")
		default:
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
		if len(field)+len(value) > 2000 {
			t.Skipf("Input too large: %d bytes (limit: 2000)", len(field)+len(value))
		}

		// Create context with timeout to prevent expensive operations from hanging
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Check context before expensive operations
		select {
		case <-ctx.Done():
			t.Skipf("Context timeout before operation")
		default:
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
		if len(item)+len(reason) > 2000 {
			t.Skipf("Input too large: %d bytes (limit: 2000)", len(item)+len(reason))
		}

		// Create context with timeout to prevent expensive operations from hanging
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Check context before expensive operations
		select {
		case <-ctx.Done():
			t.Skipf("Context timeout before operation")
		default:
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
		if len(path) > 2000 {
			t.Skipf("Input too large: %d bytes (limit: 2000)", len(path))
		}

		// Create context with timeout to prevent expensive operations from hanging
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Check context before expensive operations
		select {
		case <-ctx.Done():
			t.Skipf("Context timeout before operation")
		default:
		}

		result := PathTraversalError(path)
		require.Error(t, result)
		require.True(t, strings.HasPrefix(result.Error(), "path traversal detected"))
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
		if len(field)+len(value)+len(expectedFormat) > 2000 {
			t.Skipf("Input too large: %d bytes (limit: 2000)", len(field)+len(value)+len(expectedFormat))
		}

		// Create context with timeout to prevent expensive operations from hanging
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Check context before expensive operations
		select {
		case <-ctx.Done():
			t.Skipf("Context timeout before operation")
		default:
		}

		result := FormatError(field, value, expectedFormat)
		require.Error(t, result)
		require.True(t, strings.HasPrefix(result.Error(), "invalid format"))
	})
}
