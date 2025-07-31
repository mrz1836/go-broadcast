package testutil

import (
	"errors"
	"testing"
)

func TestAssertNoError(t *testing.T) {
	// Test with nil error - should pass
	AssertNoError(t, nil)
	AssertNoError(t, nil, "with message")
	AssertNoError(t, nil, "with", "multiple", "args")

	// Test with actual error - we can't directly test failure
	// but we can ensure the function compiles with various argument combinations
	err := errors.New("test error") //nolint:err113 // test-only errors
	_ = err                         // Prevent unused variable error
}

func TestAssertError(t *testing.T) {
	// Test with actual error - should pass
	err := errors.New("test error") //nolint:err113 // test-only errors
	AssertError(t, err)
	AssertError(t, err, "with message")
	AssertError(t, err, "with", "multiple", "args")
}

func TestAssertErrorContains(t *testing.T) {
	// Test with error containing expected message
	err := errors.New("this is a test error message") //nolint:err113 // test-only errors
	AssertErrorContains(t, err, "test error")

	// Test with exact match
	AssertErrorContains(t, err, "this is a test error message")

	// Test with empty expected message (should pass)
	AssertErrorContains(t, err, "")
}

func TestAssertEqual(t *testing.T) {
	// Test with equal values - should pass
	AssertEqual(t, 42, 42)
	AssertEqual(t, "hello", "hello")
	AssertEqual(t, true, true)
	AssertEqual(t, 3.14, 3.14, "float comparison")
	AssertEqual(t, 3.14, 3.14, "with", "multiple", "args")

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
	AssertNotEqual(t, 3.14, 2.71, "with", "multiple", "args")
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

	RunTableTests(t, tests, func(t testing.TB, tc TestCase[input, output]) {
		// Example test implementation
		result := tc.Input.a + tc.Input.b
		AssertEqual(t, tc.Expected.sum, result)
		if tc.WantErr {
			t.(*testing.T).Error("this test expected an error but addition doesn't produce errors")
		}
	})
}

// TestRunBenchmarkCases verifies that RunBenchmarkCases works correctly
// by using it in an actual benchmark function
func TestRunBenchmarkCases(t *testing.T) {
	// Since RunBenchmarkCases requires *testing.B, we'll verify it works
	// by checking that it's used correctly in BenchmarkRunBenchmarkCases
	// The actual test is in the benchmark function below
	t.Log("RunBenchmarkCases is tested via BenchmarkRunBenchmarkCases")
}

// BenchmarkRunBenchmarkCases tests the actual RunBenchmarkCases function
func BenchmarkRunBenchmarkCases(b *testing.B) {
	setupCalled := false
	cleanupCalled := false

	cases := []BenchmarkCase{
		{
			Name: "WithSetup",
			Size: 10,
			Setup: func() func() {
				setupCalled = true
				return func() {
					cleanupCalled = true
				}
			},
		},
		{
			Name: "WithoutSetup",
			Size: 20,
		},
	}

	RunBenchmarkCases(b, cases, func(b *testing.B, bc BenchmarkCase) {
		// Simple operation for benchmarking
		for i := 0; i < b.N; i++ {
			_ = bc.Size * 2
		}
	})

	// Verify setup and cleanup were called
	if !setupCalled {
		b.Error("setup function was not called")
	}
	if !cleanupCalled {
		b.Error("cleanup function was not called")
	}
}

// TestTestCaseStructure verifies the TestCase structure works correctly
func TestTestCaseStructure(t *testing.T) {
	// Test with string input/output
	stringCase := TestCase[string, int]{
		Name:     "string_length",
		Input:    "hello",
		Expected: 5,
		WantErr:  false,
		ErrMsg:   "",
	}

	AssertEqual(t, "string_length", stringCase.Name)
	AssertEqual(t, "hello", stringCase.Input)
	AssertEqual(t, 5, stringCase.Expected)
	AssertEqual(t, false, stringCase.WantErr)

	// Test with complex types
	type ComplexInput struct {
		Name  string
		Value int
	}
	type ComplexOutput struct {
		Result string
	}

	complexCase := TestCase[ComplexInput, ComplexOutput]{
		Name: "complex_test",
		Input: ComplexInput{
			Name:  "test",
			Value: 42,
		},
		Expected: ComplexOutput{
			Result: "test:42",
		},
		WantErr: true,
		ErrMsg:  "validation error",
	}

	AssertEqual(t, "complex_test", complexCase.Name)
	AssertEqual(t, "test", complexCase.Input.Name)
	AssertEqual(t, 42, complexCase.Input.Value)
	AssertEqual(t, "test:42", complexCase.Expected.Result)
	AssertEqual(t, true, complexCase.WantErr)
	AssertEqual(t, "validation error", complexCase.ErrMsg)
}

// TestBenchmarkCaseStructure verifies the BenchmarkCase structure
func TestBenchmarkCaseStructure(t *testing.T) {
	setupCalled := false
	cleanupCalled := false

	bc := BenchmarkCase{
		Name: "test_benchmark",
		Size: 100,
		Setup: func() func() {
			setupCalled = true
			return func() {
				cleanupCalled = true
			}
		},
	}

	AssertEqual(t, "test_benchmark", bc.Name)
	AssertEqual(t, 100, bc.Size)

	// Test setup function
	if bc.Setup != nil {
		cleanup := bc.Setup()
		AssertEqual(t, true, setupCalled)

		if cleanup != nil {
			cleanup()
			AssertEqual(t, true, cleanupCalled)
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
		t.Log("not in short mode")
	})

	t.Run("SkipIfNoNetwork", func(t *testing.T) {
		// This would skip if network is not available
		SkipIfNoNetwork(t)
		// If we get here, network tests are allowed
		t.Log("network tests allowed")
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
