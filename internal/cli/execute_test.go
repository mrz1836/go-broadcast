package cli

import (
	"bytes"
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/output"
)

// TestExecute tests the Execute function
func TestExecute(t *testing.T) {
	// Save original command and args
	originalCmd := rootCmd
	originalArgs := os.Args
	defer func() {
		rootCmd = originalCmd
		os.Args = originalArgs
	}()

	testCases := []struct {
		name           string
		args           []string
		setupFunc      func()
		expectError    bool
		expectExitCode int
		verifyOutput   func(*testing.T, string, string)
	}{
		{
			name: "Help command",
			args: []string{"go-broadcast", "--help"},
			verifyOutput: func(t *testing.T, stdout, _ string) {
				assert.Contains(t, stdout, "go-broadcast is a stateless File Sync Orchestrator")
				assert.Contains(t, stdout, "Available Commands:")
				assert.Contains(t, stdout, "sync")
				assert.Contains(t, stdout, "validate")
			},
		},
		{
			name: "Version flag",
			args: []string{"go-broadcast", "--version"},
			setupFunc: func() {
				// Set version info for test
				SetVersionInfo("test-version", "test-commit", "test-date")
			},
			verifyOutput: func(t *testing.T, stdout, _ string) {
				assert.Contains(t, stdout, "go-broadcast test-version")
				assert.Contains(t, stdout, "test-commit")
				assert.Contains(t, stdout, "test-date")
			},
		},
		{
			name:        "Invalid command",
			args:        []string{"go-broadcast", "invalid-command"},
			expectError: true,
			verifyOutput: func(_ *testing.T, stdout, stderr string) {
				// Since SilenceErrors is true, cobra won't print error messages
				// The error is returned but not printed - this is the expected behavior
				// We just verify that the test captured output (even if empty)
				_ = stdout
				_ = stderr
				// The important thing is the command returned an error, which we check above
			},
		},
		{
			name:        "Invalid flag",
			args:        []string{"go-broadcast", "--invalid-flag"},
			expectError: true,
			verifyOutput: func(_ *testing.T, stdout, stderr string) {
				// Since SilenceErrors is true, cobra won't print error messages
				// The error is returned but not printed - this is the expected behavior
				// We just verify that the test captured output (even if empty)
				_ = stdout
				_ = stderr
				// The important thing is the command returned an error, which we check above
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// For error tests, we test the underlying command execution instead of the full Execute flow
			// to avoid os.Exit being called
			if tc.expectError {
				// Create new root command for isolation
				testCmd := NewRootCmd()
				testCmd.SetArgs(tc.args[1:]) // Remove program name

				// Capture both cobra's output and our output package output
				var stdoutBuf, stderrBuf bytes.Buffer

				// Set cobra's output writers
				testCmd.SetOut(&stdoutBuf)
				testCmd.SetErr(&stderrBuf)

				// Also capture our output package output
				originalStdout := output.Stdout()
				originalStderr := output.Stderr()
				output.SetStdout(&stdoutBuf)
				output.SetStderr(&stderrBuf)
				defer func() {
					output.SetStdout(originalStdout)
					output.SetStderr(originalStderr)
				}()

				// Execute command and expect error
				err := testCmd.Execute()
				require.Error(t, err)

				// Verify output if test case provides verification function
				if tc.verifyOutput != nil {
					tc.verifyOutput(t, stdoutBuf.String(), stderrBuf.String())
				}
				return
			}

			// Setup
			if tc.setupFunc != nil {
				tc.setupFunc()
			}

			// Create new root command for isolation
			testCmd := NewRootCmd()
			testCmd.SetArgs(tc.args[1:]) // Remove program name

			// Capture both cobra's output and our output package output
			var stdoutBuf, stderrBuf bytes.Buffer

			// Set cobra's output writers
			testCmd.SetOut(&stdoutBuf)
			testCmd.SetErr(&stderrBuf)

			// Also capture our output package output
			originalStdout := output.Stdout()
			originalStderr := output.Stderr()
			output.SetStdout(&stdoutBuf)
			output.SetStderr(&stderrBuf)
			defer func() {
				output.SetStdout(originalStdout)
				output.SetStderr(originalStderr)
			}()

			// Execute command
			err := testCmd.Execute()
			require.NoError(t, err)

			// Verify output if test case provides verification function
			if tc.verifyOutput != nil {
				tc.verifyOutput(t, stdoutBuf.String(), stderrBuf.String())
			}
		})
	}
}

// TestExecuteWithInterrupt tests signal handling
func TestExecuteWithInterrupt(t *testing.T) {
	// This test demonstrates how signal handling works
	// In a real test, we would need to mock os.Signal handling

	t.Run("Signal handling setup", func(t *testing.T) {
		// The Execute function sets up signal handling for SIGINT and SIGTERM
		// We can't easily test the actual signal handling without complex mocking
		// but we can verify the setup code compiles and runs

		// Create a context that we can cancel
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Simulate what Execute does with signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		defer signal.Stop(sigChan)

		// Verify we can send signals to the channel
		done := make(chan bool)
		go func() {
			select {
			case sig := <-sigChan:
				assert.NotNil(t, sig)
				done <- true
			case <-ctx.Done():
				done <- false
			}
		}()

		// Send a test signal
		sigChan <- os.Interrupt

		// Wait for handler
		handled := <-done
		assert.True(t, handled, "Signal should be handled")
	})
}

// TestExecuteContextCancellation tests context cancellation behavior
func TestExecuteContextCancellation(t *testing.T) {
	// Save original
	originalCmd := rootCmd
	defer func() {
		rootCmd = originalCmd
	}()

	t.Run("Context cancellation propagation", func(t *testing.T) {
		// Create a command that checks context cancellation
		var wg sync.WaitGroup
		contextChecked := false

		testCmd := NewRootCmd()
		testCmd.RunE = func(cmd *cobra.Command, _ []string) error {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-cmd.Context().Done()
				contextChecked = true
			}()

			// Simulate work
			time.Sleep(100 * time.Millisecond)
			return nil
		}

		// Execute with cancellable context
		ctx, cancel := context.WithCancel(context.Background())

		// Run command in goroutine
		go func() {
			_ = testCmd.ExecuteContext(ctx)
		}()

		// Cancel context after short delay
		time.Sleep(50 * time.Millisecond)
		cancel()

		// Wait for goroutine to notice cancellation
		wg.Wait()

		assert.True(t, contextChecked, "Context cancellation should be detected")
	})
}

// TestExecuteErrorHandling tests error handling and exit behavior
func TestExecuteErrorHandling(t *testing.T) {
	// This test would normally verify that Execute calls os.Exit(1) on error
	// However, testing os.Exit is complex, so we test the error flow instead

	t.Run("Command error handling", func(t *testing.T) {
		// Save original
		originalCmd := rootCmd
		defer func() {
			rootCmd = originalCmd
		}()

		// Create command that returns an error
		testCmd := NewRootCmd()
		errorCmd := &cobra.Command{
			Use: "error-test",
			RunE: func(_ *cobra.Command, _ []string) error {
				return assert.AnError
			},
		}
		testCmd.AddCommand(errorCmd)
		rootCmd = testCmd

		// Capture output
		var outputBuf bytes.Buffer
		// output.SetWriter(&outputBuf) // Not available in output package
		// defer output.SetWriter(os.Stdout)
		_ = outputBuf

		// Set args
		os.Args = []string{"go-broadcast", "error-test"}

		// We can't test Execute directly because it calls os.Exit
		// Instead, test the underlying ExecuteContext
		err := rootCmd.ExecuteContext(context.Background())
		require.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})
}

// TestExecuteIntegration provides integration test examples
func TestExecuteIntegration(t *testing.T) {
	testCases := []struct {
		name         string
		args         []string
		expectError  bool
		verifyOutput func(*testing.T, string, string)
	}{
		{
			name: "Help command integration",
			args: []string{"go-broadcast", "--help"},
			verifyOutput: func(t *testing.T, stdout, stderr string) {
				assert.Contains(t, stdout, "go-broadcast is a stateless File Sync Orchestrator")
				assert.Contains(t, stdout, "Available Commands:")
				assert.Contains(t, stdout, "sync")
				assert.Contains(t, stdout, "validate")
				assert.Empty(t, stderr)
			},
		},
		{
			name: "Version flag integration",
			args: []string{"go-broadcast", "--version"},
			verifyOutput: func(t *testing.T, stdout, stderr string) {
				assert.Contains(t, stdout, "go-broadcast")
				assert.Contains(t, stdout, "Go Version:")
				assert.Empty(t, stderr)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping integration test in short mode")
			}

			// Create new root command for isolation
			testCmd := NewRootCmd()
			testCmd.SetArgs(tc.args[1:]) // Remove program name

			// Capture both cobra's output and our output package output
			var stdoutBuf, stderrBuf bytes.Buffer

			// Set cobra's output writers
			testCmd.SetOut(&stdoutBuf)
			testCmd.SetErr(&stderrBuf)

			// Also capture our output package output
			originalStdout := output.Stdout()
			originalStderr := output.Stderr()
			output.SetStdout(&stdoutBuf)
			output.SetStderr(&stderrBuf)
			defer func() {
				output.SetStdout(originalStdout)
				output.SetStderr(originalStderr)
			}()

			// Execute command
			err := testCmd.Execute()

			// Verify error expectation
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify output if test case provides verification function
			if tc.verifyOutput != nil {
				tc.verifyOutput(t, stdoutBuf.String(), stderrBuf.String())
			}
		})
	}
}
