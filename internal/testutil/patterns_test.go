package testutil

import (
	"errors"
	"testing"
)

func TestAssertNoError(t *testing.T) {
	// Test with nil error - should pass
	AssertNoError(t, nil)
	AssertNoError(t, nil, "with message")

	// Test with actual error - should fail
	// We can't directly test failure, but we can ensure the function exists and compiles
}

func TestAssertError(t *testing.T) {
	// Test with actual error - should pass
	err := errors.New("test error") //nolint:err113 // test-only errors
	AssertError(t, err)
	AssertError(t, err, "with message")

	// Test with nil error - should fail
	// We can't directly test failure, but we can ensure the function exists and compiles
}

func TestAssertErrorContains(t *testing.T) {
	// Test with error containing expected message
	err := errors.New("this is a test error message") //nolint:err113 // test-only errors
	AssertErrorContains(t, err, "test error")

	// Test with exact match
	AssertErrorContains(t, err, "this is a test error message")
}

func TestAssertEqual(t *testing.T) {
	// Test with equal values - should pass
	AssertEqual(t, 42, 42)
	AssertEqual(t, "hello", "hello")
	AssertEqual(t, true, true)
	AssertEqual(t, 3.14, 3.14, "float comparison")

	// Test with slices of same type
	type myStruct struct{ value int }
	AssertEqual(t, myStruct{42}, myStruct{42})
}

func TestAssertNotEqual(t *testing.T) {
	// Test with different values - should pass
	AssertNotEqual(t, 42, 43)
	AssertNotEqual(t, "hello", "world")
	AssertNotEqual(t, true, false)
	AssertNotEqual(t, 3.14, 2.71, "float comparison")
}

func TestRunTableTests(t *testing.T) {
	// Example test case using our patterns
	type input struct {
		a, b int
	}
	type output struct {
		sum int
	}

	tests := []TestCase[input, output]{
		{
			Name:     "positive numbers",
			Input:    input{a: 2, b: 3},
			Expected: output{sum: 5},
			WantErr:  false,
		},
		{
			Name:     "negative numbers",
			Input:    input{a: -2, b: -3},
			Expected: output{sum: -5},
			WantErr:  false,
		},
		{
			Name:     "zero",
			Input:    input{a: 0, b: 0},
			Expected: output{sum: 0},
			WantErr:  false,
		},
	}

	RunTableTests(t, tests, func(t *testing.T, tc TestCase[input, output]) {
		// Example test implementation
		result := tc.Input.a + tc.Input.b
		AssertEqual(t, tc.Expected.sum, result)
		if tc.WantErr {
			t.Error("this test expected an error but addition doesn't produce errors")
		}
	})
}

func TestRunBenchmarkCases(t *testing.T) {
	// Convert testing.T to testing.B for demonstration
	// In real usage, this would be in a benchmark function
	cases := []BenchmarkCase{
		{
			Name: "small",
			Size: 10,
			Setup: func() func() {
				// Setup code
				return func() {
					// Cleanup code
				}
			},
		},
		{
			Name: "medium",
			Size: 100,
		},
		{
			Name: "large",
			Size: 1000,
		},
	}

	// Verify the structure compiles and can be used
	for _, bc := range cases {
		if bc.Size <= 0 {
			t.Errorf("invalid benchmark case size: %d", bc.Size)
		}
	}
}

func TestSkipFunctions(t *testing.T) {
	// These functions will skip in certain conditions
	// We're just verifying they compile and can be called

	t.Run("SkipIfShort", func(t *testing.T) {
		// This would skip if -short flag is used
		SkipIfShort(t)
		// If we get here, we're not in short mode
	})

	t.Run("SkipIfNoNetwork", func(t *testing.T) {
		// This would skip if network is not available
		SkipIfNoNetwork(t)
		// If we get here, network tests are allowed
	})
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"contains substring", "hello world", "world", true},
		{"contains at start", "hello world", "hello", true},
		{"contains at end", "hello world", "world", true},
		{"exact match", "hello", "hello", true},
		{"doesn't contain", "hello world", "xyz", false},
		{"empty substring", "hello world", "", false},
		{"empty string", "", "hello", false},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			AssertEqual(t, tt.expected, result)
		})
	}
}
