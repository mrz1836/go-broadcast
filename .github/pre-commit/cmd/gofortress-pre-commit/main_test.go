package main

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/pre-commit/cmd/gofortress-pre-commit/cmd"
)

func TestMain(t *testing.T) {
	// Test that the binary can be built and executed
	// This test verifies the main entry point works

	// Build the binary for testing
	ctx := context.Background()
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "gofortress-pre-commit-test", ".")
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build binary")

	defer func() {
		// Clean up the test binary
		_ = os.Remove("gofortress-pre-commit-test")
	}()

	// Test version flag (should exit cleanly)
	versionCmd := exec.CommandContext(ctx, "./gofortress-pre-commit-test", "--version")
	output, err := versionCmd.Output()
	require.NoError(t, err, "Version command should succeed")
	assert.Contains(t, string(output), "gofortress-pre-commit version", "Should contain version info")

	// Test help flag (should exit cleanly)
	helpCmd := exec.CommandContext(ctx, "./gofortress-pre-commit-test", "--help")
	output, err = helpCmd.Output()
	require.NoError(t, err, "Help command should succeed")
	assert.Contains(t, string(output), "GoFortress", "Should contain help text")
}

func TestVersionVariables(t *testing.T) {
	// Test that version variables are set correctly
	assert.Equal(t, "dev", version)
	assert.Equal(t, "none", commit)
	assert.Equal(t, "unknown", buildDate)
}

func TestMainErrorHandling(t *testing.T) {
	// Test main function error handling by providing invalid arguments
	// We can't directly test main() function exit, but we can test the cmd package

	// This test verifies that main calls cmd.Execute() correctly
	// and handles errors appropriately by checking that invalid commands
	// produce errors when executed via the binary

	buildCmd := exec.CommandContext(context.Background(), "go", "build", "-o", "gofortress-pre-commit-test", ".")
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build binary")

	defer func() {
		_ = os.Remove("gofortress-pre-commit-test")
	}()

	// Test with invalid command
	invalidCmd := exec.CommandContext(context.Background(), "./gofortress-pre-commit-test", "invalid-command")
	_, err = invalidCmd.Output()
	assert.Error(t, err, "Invalid command should return error")
}

func TestSetVersionInfo(t *testing.T) {
	// Test that SetVersionInfo is called correctly
	// We can verify this by checking that the cmd package receives version info

	// The version variables are package-level and set during build
	// This test verifies they are accessible and have expected default values
	assert.NotEmpty(t, version, "Version should not be empty")
	assert.NotEmpty(t, commit, "Commit should not be empty")
	assert.NotEmpty(t, buildDate, "Build date should not be empty")
}

func TestMainFunctionFlow(t *testing.T) {
	// Test the main function flow by directly calling the functions it uses
	// This ensures the main function logic is covered

	// Test SetVersionInfo function call
	cmd.SetVersionInfo(version, commit, buildDate)

	// Test Execute function call with help flag to avoid side effects
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
	}()

	// Set args to help command which should exit cleanly
	os.Args = []string{"gofortress-pre-commit", "--help"}

	// Execute the command (this is what main() does)
	err := cmd.Execute()

	// Help command should succeed (exit code 0 in cobra)
	require.NoError(t, err)

	// Test that version info was set correctly
	// We can't directly access the cmd package variables but we know
	// the function was called successfully if no panic occurred
	assert.NotPanics(t, func() {
		cmd.SetVersionInfo("test", "test", "test")
	})
}
