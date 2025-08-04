package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/pre-commit/cmd/gofortress-pre-commit/cmd"
)

var errCommandExited = errors.New("command exited")

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
		if err := os.Remove("gofortress-pre-commit-test"); err != nil {
			t.Logf("Failed to remove test binary: %v", err)
		}
	}()

	// Test various command scenarios
	tests := []struct {
		name     string
		args     []string
		wantExit int
	}{
		{
			name:     "help command",
			args:     []string{"--help"},
			wantExit: 0,
		},
		{
			name:     "version flag",
			args:     []string{"--version"},
			wantExit: 0,
		},
		{
			name:     "show checks",
			args:     []string{"run", "--show-checks"},
			wantExit: 0,
		},
		{
			name:     "invalid command",
			args:     []string{"invalid-command"},
			wantExit: 1,
		},
		{
			name:     "status command",
			args:     []string{"status"},
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.CommandContext(ctx, "./gofortress-pre-commit-test", tt.args...) // #nosec G204 - test code with controlled input
			err := cmd.Run()

			if tt.wantExit == 0 {
				assert.NoError(t, err, "Expected successful exit")
			} else {
				assert.Error(t, err, "Expected error exit")
			}
		})
	}
}

// Test the main function directly with command setup
func TestMainFunction(t *testing.T) {
	// Save original args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "help flag",
			args:        []string{"gofortress-pre-commit", "--no-color", "--help"},
			expectError: false,
		},
		{
			name:        "version flag",
			args:        []string{"gofortress-pre-commit", "--no-color", "--version"},
			expectError: false,
		},
		{
			name:        "show checks",
			args:        []string{"gofortress-pre-commit", "--no-color", "run", "--show-checks"},
			expectError: false,
		},
		{
			name:        "invalid command",
			args:        []string{"gofortress-pre-commit", "--no-color", "invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset command for each test
			cmd.ResetCommand()
			cmd.SetVersionInfo("test", "test-commit", "test-date")

			// Set up args
			os.Args = tt.args

			// Capture output
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			// Execute command and capture panic
			var cmdErr error
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Main calls os.Exit(1) on error, which we can't intercept
						// So we expect a panic in test environment
						cmdErr = fmt.Errorf("%w: %v", errCommandExited, r)
					}
				}()

				// Call the command directly
				cmdErr = cmd.Execute()
			}()

			// Restore stdout/stderr
			if err := w.Close(); err != nil {
				t.Logf("Failed to close pipe writer: %v", err)
			}
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			// Read output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Logf("Failed to copy output: %v", err)
			}
			output := buf.String()

			if tt.expectError {
				require.Error(t, cmdErr, "Expected command to fail")
			} else {
				require.NoError(t, cmdErr, "Expected command to succeed")
			}

			// Verify some output was produced (except for invalid command which might be silenced)
			if tt.name != "invalid command" {
				assert.NotEmpty(t, output, "Expected some output")
			}
		})
	}
}

// Test direct execution scenarios
func TestDirectExecution(t *testing.T) {
	// Save original values
	oldArgs := os.Args
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Test successful execution
	t.Run("successful help", func(t *testing.T) {
		os.Args = []string{"gofortress-pre-commit", "--no-color", "--help"}

		// Capture output
		r, w, _ := os.Pipe()
		os.Stdout = w
		os.Stderr = w

		// Reset and set version info
		cmd.ResetCommand()
		cmd.SetVersionInfo("test", "test-commit", "test-date")

		// Execute
		err := cmd.Execute()

		// Close and read output
		if closeErr := w.Close(); closeErr != nil {
			t.Logf("Failed to close pipe writer: %v", closeErr)
		}
		var buf bytes.Buffer
		if _, copyErr := io.Copy(&buf, r); copyErr != nil {
			t.Logf("Failed to copy output: %v", copyErr)
		}

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "GoFortress Pre-commit System")
	})
}

// Test version info functionality using subprocess
func TestVersionInfo(t *testing.T) {
	// Save and restore working directory to prevent interference from other tests
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	// Ensure we're in the correct directory for building
	expectedWD := "/Users/mrz/projects/go-broadcast/.github/pre-commit/cmd/gofortress-pre-commit"
	if originalWD != expectedWD {
		err = os.Chdir(expectedWD)
		require.NoError(t, err, "Failed to change to correct directory")
	}

	// Build a test binary
	ctx := context.Background()
	testBinary := "./test-version-binary"
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", testBinary, ".")

	// Add debug output for build failures
	var stdout, stderr bytes.Buffer
	buildCmd.Stdout = &stdout
	buildCmd.Stderr = &stderr

	err = buildCmd.Run()
	if err != nil {
		t.Logf("Build failed. stdout: %s, stderr: %s", stdout.String(), stderr.String())
		t.Logf("Working directory: %s", func() string {
			wd, _ := os.Getwd()
			return wd
		}())
	}
	require.NoError(t, err)

	defer func() {
		if removeErr := os.Remove(testBinary); removeErr != nil {
			t.Logf("Failed to remove test binary: %v", removeErr)
		}
	}()

	// Run with version flag and no-color flag
	testCmd := exec.CommandContext(ctx, testBinary, "--no-color", "--version")
	output, err := testCmd.Output()
	require.NoError(t, err)

	outputStr := string(output)
	t.Logf("Version command output: %q", outputStr)

	// Check that version command works and outputs version information
	assert.Contains(t, outputStr, "gofortress-pre-commit")
	assert.Contains(t, outputStr, "version")
	// The default version values should be present
	assert.Contains(t, outputStr, "dev")
	assert.Contains(t, outputStr, "none")
	assert.Contains(t, outputStr, "unknown")
}

// Test that main calls os.Exit(1) on error - using subprocess
func TestMainExitOnError(t *testing.T) {
	// This test verifies that main() calls os.Exit(1) on error
	// by running it as a subprocess

	// Save and restore working directory to prevent interference from other tests
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}()

	// Ensure we're in the correct directory for building
	expectedWD := "/Users/mrz/projects/go-broadcast/.github/pre-commit/cmd/gofortress-pre-commit"
	if originalWD != expectedWD {
		err = os.Chdir(expectedWD)
		require.NoError(t, err, "Failed to change to correct directory")
	}

	// Reset command state before test
	cmd.ResetCommand()
	defer cmd.ResetCommand() // Reset after test too

	// Build a test binary
	ctx := context.Background()
	testBinary := "./test-exit-binary"
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", testBinary, ".")

	// Add debug output for build failures
	var stdout, stderr bytes.Buffer
	buildCmd.Stdout = &stdout
	buildCmd.Stderr = &stderr

	err = buildCmd.Run()
	if err != nil {
		t.Logf("Build failed. stdout: %s, stderr: %s", stdout.String(), stderr.String())
		t.Logf("Working directory: %s", func() string {
			wd, _ := os.Getwd()
			return wd
		}())
	}
	require.NoError(t, err)

	defer func() {
		if removeErr := os.Remove(testBinary); removeErr != nil {
			t.Logf("Failed to remove test binary: %v", removeErr)
		}
	}()

	// Run with invalid command and no-color flag
	testCmd := exec.CommandContext(ctx, testBinary, "--no-color", "invalid-command")
	err = testCmd.Run()

	// Should exit with status 1
	require.Error(t, err)
	var exitError *exec.ExitError
	ok := errors.As(err, &exitError)
	if ok {
		assert.Equal(t, 1, exitError.ExitCode())
	}
}

// Benchmark main execution
func BenchmarkMain(b *testing.B) {
	// Save original values
	oldArgs := os.Args
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Discard output during benchmark
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull
	os.Stderr = devNull
	defer func() {
		if err := devNull.Close(); err != nil {
			b.Logf("Failed to close devNull: %v", err)
		}
	}()

	// Set args for list command (fast operation)
	os.Args = []string{"gofortress-pre-commit", "list"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cmd.Execute(); err != nil {
			b.Logf("Command failed: %v", err)
		}
	}
}

// Example showing how to use the pre-commit system
func Example_main() {
	// The gofortress-pre-commit tool manages Git hooks for code quality
	// It integrates with your existing Makefile targets

	// Usage:
	// gofortress-pre-commit install          # Install Git hooks
	// gofortress-pre-commit uninstall        # Remove Git hooks
	// gofortress-pre-commit run              # Run checks on staged files
	// gofortress-pre-commit run --all-files  # Run checks on all files
	// gofortress-pre-commit list             # List available checks
	// gofortress-pre-commit status           # Show installation status

	fmt.Println("GoFortress Pre-commit System")
}
