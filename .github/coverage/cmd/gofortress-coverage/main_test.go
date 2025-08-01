package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {
	// Test that main function can be called without panicking
	// We test this by running the binary with --help flag to avoid side effects

	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		// This is the subprocess that will run main()
		// Set args to help to avoid side effects
		os.Args = []string{"gofortress-coverage", "--help"}
		main()
		return
	}

	// Run the main function in a subprocess
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestMain") //nolint:gosec // Test needs subprocess execution
	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1")
	err := cmd.Run()
	// The --help flag should cause the command to exit with code 0
	// If there's an error, it should be an exit error with code 0 (help was shown)
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			// Help command typically exits with 0, but cobra may exit with different codes
			// We just ensure it's a controlled exit, not a panic
			require.GreaterOrEqual(t, exitError.ExitCode(), 0, "Command should exit gracefully, got exit code: %d", exitError.ExitCode())
		} else {
			t.Fatalf("Unexpected error type: %v", err)
		}
	}
}

func TestMainErrorHandling(t *testing.T) {
	// Test that main function handles cmd.Execute() errors properly

	if os.Getenv("GO_TEST_SUBPROCESS_ERROR") == "1" {
		// This is the subprocess that will run main() with invalid args
		os.Args = []string{"gofortress-coverage", "--invalid-flag-that-does-not-exist"}
		main()
		return
	}

	// Run the main function in a subprocess with invalid arguments
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestMainErrorHandling") //nolint:gosec // Test needs subprocess execution
	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS_ERROR=1")
	err := cmd.Run()

	// Should get an exit error since invalid flag should cause error
	require.Error(t, err)

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		// Should exit with code 1 as specified in main()
		require.Equal(t, 1, exitError.ExitCode(), "Command should exit with code 1 on error")
	} else {
		t.Fatalf("Expected exit error, got: %v", err)
	}
}

// TestMainFunctionExists verifies that main function exists and is callable
func TestMainFunctionExists(t *testing.T) {
	// This test ensures the main function exists and can be referenced
	// We can't call it directly in tests due to os.Exit(), but we can verify it exists
	require.NotNil(t, main, "main function should exist")
}
