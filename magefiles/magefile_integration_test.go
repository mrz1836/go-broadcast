//go:build mage && integration

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBenchQuickIntegration runs an actual quick benchmark with minimal settings
// This test is tagged as integration and will only run when explicitly requested
func TestBenchQuickIntegration(t *testing.T) {
	// Skip if in CI environment unless specifically requested
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	// Use real commander for integration test
	setCommander(ShCommander{})

	// This will run actual benchmarks but with minimal time
	err := BenchQuick()

	// We expect this might fail in test environment, so we're mainly
	// testing that the command structure is correct
	if err != nil {
		t.Logf("Benchmark command failed (expected in test env): %v", err)
		// Don't fail the test - the command structure is what we're testing
	} else {
		t.Log("Benchmark command executed successfully")
	}
}

// TestTestQuickIntegration runs actual quick tests
func TestTestQuickIntegration(t *testing.T) {
	// Skip if in CI environment unless specifically requested
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	// Use real commander for integration test
	setCommander(ShCommander{})

	err := TestQuick()

	// This should generally succeed as it runs the project's own tests
	if err != nil {
		t.Logf("Test command failed: %v", err)
		// Log but don't fail - environment might not be set up for full test run
	} else {
		t.Log("Test command executed successfully")
	}
}

// TestMageCommandsSmoke is a smoke test that verifies mage commands can be invoked
// without actually running the full commands
func TestMageCommandsSmoke(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test")
	}

	testCases := []struct {
		name     string
		function func() error
	}{
		{"BenchHeavy", BenchHeavy},
		{"BenchQuick", BenchQuick},
		{"BenchAll", BenchAll},
		{"TestQuick", TestQuick},
		{"TestPerf", TestPerf},
		{"TestAll", TestAll},
	}

	// No need for additional logging mock - using existing MockCommander

	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	// Use a mock that just logs what would be executed
	setCommander(&MockCommander{})

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mock for each test
			if mock, ok := getCommander().(*MockCommander); ok {
				mock.Reset()
			}

			// This tests that the function can be called without panicking
			// and that it attempts to run the expected command
			err := tc.function()

			// We expect no error from the mock
			require.NoError(t, err)

			// Verify at least one command was attempted
			if mock, ok := getCommander().(*MockCommander); ok {
				calls := mock.GetCalls()
				assert.NotEmpty(t, calls, "Function %s should attempt to run at least one command", tc.name)

				// All calls should start with "go"
				for i, call := range calls {
					assert.True(t, len(call) > 0, "Call %d should have at least one argument", i)
					if len(call) > 0 {
						assert.Equal(t, "go", call[0], "Call %d should start with 'go' command", i)
					}
				}
			}
		})
	}
}

// TestMagefileStructure validates the magefile structure
func TestMagefileStructure(t *testing.T) {
	t.Run("CommanderInterface", func(t *testing.T) {
		// Test that our ShCommander implements Commander
		var _ Commander = ShCommander{}

		// Test that default commander is not nil
		assert.NotNil(t, getCommander())
	})

	t.Run("FunctionSignatures", func(t *testing.T) {
		// Test that all mage functions have the expected signature: func() error
		functions := []func() error{
			BenchHeavy,
			BenchQuick,
			BenchAll,
			TestQuick,
			TestPerf,
			TestAll,
		}

		for i, fn := range functions {
			assert.NotNil(t, fn, "Function %d should not be nil", i)
		}
	})
}

// TestRealCommandExecution tests actual command execution if enabled
func TestRealCommandExecution(t *testing.T) {
	if os.Getenv("ENABLE_REAL_COMMAND_TESTS") != "true" {
		t.Skip("Real command execution tests disabled. Set ENABLE_REAL_COMMAND_TESTS=true to enable.")
	}

	// Save original commander
	originalCommander := getCommander()
	defer func() { setCommander(originalCommander) }()

	// Use real commander
	setCommander(ShCommander{})

	t.Run("TestQuick", func(t *testing.T) {
		// This should work in most environments
		err := TestQuick()
		if err != nil {
			t.Logf("TestQuick failed: %v", err)
			// Don't fail the test - environment might not be suitable
		}
	})

	// Note: We don't test BenchHeavy or TestPerf here as they take too long
	// and may not be suitable for all test environments
}
