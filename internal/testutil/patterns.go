// Package testutil provides shared test utilities and patterns for consistent testing across the go-broadcast codebase.
package testutil

import (
	"testing"
)

// TestCase represents a generic test case structure for table-driven tests.
// It provides a consistent pattern for organizing test scenarios.
type TestCase[TInput any, TExpected any] struct {
	Name     string    // Test case name for t.Run()
	Input    TInput    // Input value(s) for the test
	Expected TExpected // Expected output/result
	WantErr  bool      // Whether an error is expected
	ErrMsg   string    // Expected error message substring (optional)
}

// RunTableTests runs table-driven tests with consistent patterns.
// It provides a standard way to execute multiple test cases.
func RunTableTests[TInput any, TExpected any](
	t testing.TB,
	tests []TestCase[TInput, TExpected],
	runner func(testing.TB, TestCase[TInput, TExpected]),
) {
	t.Helper()

	for _, tt := range tests {
		if tRunner, ok := t.(*testing.T); ok {
			tRunner.Run(tt.Name, func(t *testing.T) {
				runner(t, tt)
			})
		} else if bRunner, ok := t.(*testing.B); ok {
			bRunner.Run(tt.Name, func(b *testing.B) {
				runner(b, tt)
			})
		}
	}
}

// AssertNoError fails the test if err is not nil.
// It provides a consistent way to check for unexpected errors.
func AssertNoError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("unexpected error: %v, %v", err, msgAndArgs)
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// AssertError fails the test if err is nil when error is expected.
// It provides a consistent way to check that an error occurred.
func AssertError(t testing.TB, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("expected error but got nil: %v", msgAndArgs)
		} else {
			t.Fatal("expected error but got nil")
		}
	}
}

// AssertErrorContains checks that an error occurred and contains the expected message.
// It provides a consistent way to validate error messages.
func AssertErrorContains(t testing.TB, err error, expectedMsg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing '%s' but got nil", expectedMsg)
	}
	if expectedMsg != "" && !contains(err.Error(), expectedMsg) {
		t.Fatalf("expected error to contain '%s' but got: %v", expectedMsg, err)
	}
}

// AssertEqual checks that two values are equal.
// It provides a simple equality check with clear error messages.
func AssertEqual[T comparable](t testing.TB, expected, actual T, msgAndArgs ...interface{}) {
	t.Helper()
	if expected != actual {
		if len(msgAndArgs) > 0 {
			t.Fatalf("expected %v but got %v: %v", expected, actual, msgAndArgs)
		} else {
			t.Fatalf("expected %v but got %v", expected, actual)
		}
	}
}

// AssertNotEqual checks that two values are not equal.
// It provides a simple inequality check with clear error messages.
func AssertNotEqual[T comparable](t testing.TB, unexpected, actual T, msgAndArgs ...interface{}) {
	t.Helper()
	if unexpected == actual {
		if len(msgAndArgs) > 0 {
			t.Fatalf("expected value to not be %v: %v", unexpected, msgAndArgs)
		} else {
			t.Fatalf("expected value to not be %v", unexpected)
		}
	}
}

// BenchmarkCase represents a benchmark test case with size information.
type BenchmarkCase struct {
	Name  string        // Benchmark case name
	Size  int           // Size parameter for the benchmark
	Setup func() func() // Optional setup function that returns cleanup
}

// RunBenchmarkCases runs a set of benchmark cases with consistent patterns.
func RunBenchmarkCases(b *testing.B, cases []BenchmarkCase, runner func(*testing.B, BenchmarkCase)) {
	for _, bc := range cases {
		b.Run(bc.Name, func(b *testing.B) {
			if bc.Setup != nil {
				cleanup := bc.Setup()
				if cleanup != nil {
					defer cleanup()
				}
			}
			runner(b, bc)
		})
	}
}

// SkipIfShort skips a test if running in short mode.
func SkipIfShort(t testing.TB) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}

// SkipIfNoNetwork skips a test that requires network access.
func SkipIfNoNetwork(t testing.TB) {
	// This could be enhanced to actually check network availability
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

// containsSubstring checks if s contains substr.
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
