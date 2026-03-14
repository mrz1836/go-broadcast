package testutil_test

import (
	"errors"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// Example of using testutil patterns for a simple function
func Add(a, b int) (int, error) {
	if a < 0 || b < 0 {
		return 0, errors.New("negative numbers not allowed") //nolint:err113 // example test-only errors
	}
	return a + b, nil
}

// ExampleTestWithPatterns demonstrates using the testutil patterns
func TestAddWithPatterns(t *testing.T) {
	// Define test cases using generic TestCase structure
	tests := []testutil.TestCase[struct{ a, b int }, int]{
		{
			Name:     "positive numbers",
			Input:    struct{ a, b int }{2, 3},
			Expected: 5,
			WantErr:  false,
		},
		{
			Name:     "zero values",
			Input:    struct{ a, b int }{0, 0},
			Expected: 0,
			WantErr:  false,
		},
		{
			Name:     "negative first number",
			Input:    struct{ a, b int }{-1, 5},
			Expected: 0,
			WantErr:  true,
			ErrMsg:   "negative numbers not allowed",
		},
		{
			Name:     "negative second number",
			Input:    struct{ a, b int }{5, -1},
			Expected: 0,
			WantErr:  true,
			ErrMsg:   "negative numbers not allowed",
		},
	}

	// Run table-driven tests with consistent patterns
	testutil.RunTableTests(t, tests, func(t testing.TB, tc testutil.TestCase[struct{ a, b int }, int]) {
		result, err := Add(tc.Input.a, tc.Input.b)

		if tc.WantErr {
			testutil.AssertError(t, err, "expected error for inputs", tc.Input)
			if tc.ErrMsg != "" {
				testutil.AssertErrorContains(t, err, tc.ErrMsg)
			}
		} else {
			testutil.AssertNoError(t, err, "unexpected error for inputs", tc.Input)
			testutil.AssertEqual(t, tc.Expected, result, "incorrect result")
		}
	})
}

// Example of simple assertions without table-driven tests
func TestSimpleAssertions(t *testing.T) {
	// Test successful addition
	result, err := Add(10, 20)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, 30, result)

	// Test error case
	_, err = Add(-5, 10)
	testutil.AssertError(t, err)
	testutil.AssertErrorContains(t, err, "negative")

	// Test not equal assertion
	result2, _ := Add(5, 5)
	testutil.AssertNotEqual(t, 0, result2, "result should not be zero")
}

// Example benchmark using BenchmarkCase
func BenchmarkAddOperations(b *testing.B) {
	cases := []testutil.BenchmarkCase{
		{
			Name: "small_numbers",
			Size: 10,
		},
		{
			Name: "medium_numbers",
			Size: 100,
		},
		{
			Name: "large_numbers",
			Size: 1000,
			Setup: func() func() {
				// Optional setup
				return nil // No cleanup needed
			},
		},
	}

	testutil.RunBenchmarkCases(b, cases, func(b *testing.B, bc testutil.BenchmarkCase) {
		for i := 0; i < b.N; i++ {
			_, _ = Add(bc.Size, bc.Size)
		}
	})
}

// Example of using skip functions
func TestNetworkOperation(t *testing.T) {
	testutil.SkipIfNoNetwork(t)
	// Network-dependent test code would go here
}

func TestLongRunningOperation(t *testing.T) {
	testutil.SkipIfShort(t)
	// Time-consuming test code would go here
}
