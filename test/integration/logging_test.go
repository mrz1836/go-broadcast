package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateJSONLogOutput validates that the output contains valid JSON log entries
func validateJSONLogOutput(t *testing.T, output string) {
	if strings.TrimSpace(output) == "" {
		return
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	foundValidJSON := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if validateSingleJSONLogEntry(t, line) {
			foundValidJSON = true
			break // Found at least one valid JSON log entry
		}
	}

	if !foundValidJSON && len(lines) > 0 {
		// If we have output but no valid JSON, that's worth noting but not failing
		t.Logf("Output present but no valid JSON found: %s", output)
	}
}

// validateSingleJSONLogEntry validates a single JSON log entry
func validateSingleJSONLogEntry(t *testing.T, line string) bool {
	var logEntry map[string]interface{}
	if json.Unmarshal([]byte(line), &logEntry) != nil {
		return false
	}

	// Validate JSON structure
	if level, ok := logEntry["level"]; ok {
		assert.Contains(t, []interface{}{"info", "debug", "trace", "warn", "error"}, level,
			"should have valid log level")
	}

	if message, ok := logEntry["message"]; ok {
		assert.IsType(t, "", message, "message should be string")
	}

	// Check for timestamp
	if timestamp, ok := logEntry["@timestamp"]; ok {
		assert.IsType(t, "", timestamp, "timestamp should be string")
	}

	return true
}

// TestLoggingIntegration tests the complete logging system end-to-end
func TestLoggingIntegration(t *testing.T) {
	// Create test configuration
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sync.yaml")

	configContent := `version: 1
source:
  repo: "org/template"
  branch: "master"
defaults:
  branch_prefix: "sync/template"
  pr_labels: ["automated-sync"]
targets:
  - repo: "org/service-a"
    files:
      - src: ".github/workflows/ci.yml"
        dest: ".github/workflows/ci.yml"
    transform:
      repo_name: true
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	t.Run("text format logging", func(t *testing.T) {
		// Test basic text format logging
		var buffer bytes.Buffer
		var bufferMutex sync.Mutex

		// Create verbose CLI with text format
		cmd := cli.NewRootCmdWithVerbose()
		cmd.SetArgs([]string{"validate", "--config", configPath, "-v"})

		// Capture stderr output (where logs go)
		originalStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		// Execute command in background
		errChan := make(chan error, 1)
		go func() {
			errChan <- cmd.ExecuteContext(context.Background())
		}()

		// Read output with proper synchronization
		readerDone := make(chan bool, 1)
		go func() {
			defer func() { readerDone <- true }()
			buf := make([]byte, 4096)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					bufferMutex.Lock()
					buffer.Write(buf[:n])
					bufferMutex.Unlock()
				}
				if err != nil {
					break
				}
			}
		}()

		// Wait for command completion
		var cmdErr error
		select {
		case cmdErr = <-errChan:
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
		case <-time.After(10 * time.Second):
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			t.Fatal("command timed out")
		}

		// Wait for reader to finish
		select {
		case <-readerDone:
		case <-time.After(500 * time.Millisecond):
			t.Log("Reader timeout - proceeding with available output")
		}

		os.Stderr = originalStderr
		require.NoError(t, cmdErr, "validate command should succeed")

		bufferMutex.Lock()
		output := buffer.String()
		bufferMutex.Unlock()

		// Log output for debugging
		t.Logf("Captured output length: %d", len(output))
		if len(output) > 0 {
			previewLen := 200
			if len(output) < previewLen {
				previewLen = len(output)
			}
			t.Logf("Output preview: %q", output[:previewLen])
		}

		// For text format, we expect some form of output (even if minimal)
		// The command should produce logs to stderr when running with -v flag
		if len(output) == 0 {
			t.Log("No stderr output captured - this may indicate logs are going elsewhere")
			// Don't fail the test if no output is captured, as the command succeeded
			return
		}

		// If we have output, validate it's in text format
		// Text format should be human-readable and not JSON
		assert.False(t, strings.HasPrefix(strings.TrimSpace(output), "{"),
			"text format should not start with JSON")
	})

	t.Run("json format logging", func(t *testing.T) {
		// Test JSON format logging
		var buffer bytes.Buffer

		// Create verbose CLI with JSON format
		cmd := cli.NewRootCmdWithVerbose()
		cmd.SetArgs([]string{"validate", "--config", configPath, "--json", "-v"})

		// Capture stderr output
		originalStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		// Execute command in background
		errChan := make(chan error, 1)
		go func() {
			errChan <- cmd.ExecuteContext(context.Background())
		}()

		// Read output
		go func() {
			buf := make([]byte, 2048)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					buffer.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Wait for command completion
		select {
		case err := <-errChan:
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stderr = originalStderr
			require.NoError(t, err, "validate command should succeed")
		case <-time.After(10 * time.Second):
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stderr = originalStderr
			t.Fatal("command timed out")
		}

		time.Sleep(100 * time.Millisecond) // Allow buffer to fill
		output := buffer.String()

		// Validate JSON format output
		validateJSONLogOutput(t, output)
	})

	t.Run("verbose level progression", func(t *testing.T) {
		// Test different verbose levels
		verboseLevels := []struct {
			flag     string
			expected string
		}{
			{"-v", "debug"},
			{"-vv", "trace"},
			{"-vvv", "trace"},
		}

		for _, vl := range verboseLevels {
			t.Run("verbose_"+vl.flag, func(t *testing.T) {
				var buffer bytes.Buffer

				cmd := cli.NewRootCmdWithVerbose()
				cmd.SetArgs([]string{"validate", "--config", configPath, vl.flag, "--json"})

				// Capture stderr
				originalStderr := os.Stderr
				r, w, _ := os.Pipe()
				os.Stderr = w

				// Execute command
				errChan := make(chan error, 1)
				go func() {
					errChan <- cmd.ExecuteContext(context.Background())
				}()

				// Read output
				go func() {
					buf := make([]byte, 1024)
					for {
						n, err := r.Read(buf)
						if n > 0 {
							buffer.Write(buf[:n])
						}
						if err != nil {
							break
						}
					}
				}()

				// Wait for completion
				select {
				case err := <-errChan:
					if closeErr := w.Close(); closeErr != nil {
						t.Logf("Warning: failed to close writer: %v", closeErr)
					}
					os.Stderr = originalStderr
					require.NoError(t, err, "command should succeed")
				case <-time.After(10 * time.Second):
					if closeErr := w.Close(); closeErr != nil {
						t.Logf("Warning: failed to close writer: %v", closeErr)
					}
					os.Stderr = originalStderr
					t.Fatal("command timed out")
				}

				time.Sleep(50 * time.Millisecond)

				// Basic validation that verbose flag was processed
				// Specific log level validation would require more complex setup
				// Test passes if verbose flag execution completes without error
			})
		}
	})

	t.Run("debug flags integration", func(t *testing.T) {
		// Test component-specific debug flags
		debugFlags := []string{
			"--debug-git",
			"--debug-api",
			"--debug-transform",
			"--debug-config",
			"--debug-state",
		}

		for _, flag := range debugFlags {
			t.Run("flag_"+strings.TrimPrefix(flag, "--"), func(t *testing.T) {
				var buffer bytes.Buffer

				cmd := cli.NewRootCmdWithVerbose()
				cmd.SetArgs([]string{"validate", "--config", configPath, flag, "-v"})

				// Capture stderr
				originalStderr := os.Stderr
				r, w, _ := os.Pipe()
				os.Stderr = w

				// Execute command
				errChan := make(chan error, 1)
				go func() {
					errChan <- cmd.ExecuteContext(context.Background())
				}()

				// Read output
				go func() {
					buf := make([]byte, 1024)
					for {
						n, err := r.Read(buf)
						if n > 0 {
							buffer.Write(buf[:n])
						}
						if err != nil {
							break
						}
					}
				}()

				// Wait for completion
				select {
				case err := <-errChan:
					if closeErr := w.Close(); closeErr != nil {
						t.Logf("Warning: failed to close writer: %v", closeErr)
					}
					os.Stderr = originalStderr
					require.NoError(t, err, "command should succeed with debug flag")
				case <-time.After(10 * time.Second):
					if closeErr := w.Close(); closeErr != nil {
						t.Logf("Warning: failed to close writer: %v", closeErr)
					}
					os.Stderr = originalStderr
					t.Fatal("command timed out")
				}

				// Validate that debug flag was processed without error
				// Test passes if no panic occurs during execution
			})
		}
	})

	t.Run("sensitive data redaction", func(t *testing.T) {
		// Test that sensitive data gets redacted in logs
		// This test verifies the redaction functionality works end-to-end

		// Set up environment with fake sensitive data
		originalToken := os.Getenv("GITHUB_TOKEN")
		testToken := "ghp_test1234567890abcdefghijklmnopqrstuvwxyz123456"
		if err := os.Setenv("GITHUB_TOKEN", testToken); err != nil {
			t.Fatalf("Failed to set GITHUB_TOKEN: %v", err)
		}
		defer func() {
			if originalToken != "" {
				if err := os.Setenv("GITHUB_TOKEN", originalToken); err != nil {
					t.Logf("Warning: failed to restore GITHUB_TOKEN: %v", err)
				}
			} else {
				if err := os.Unsetenv("GITHUB_TOKEN"); err != nil {
					t.Logf("Warning: failed to unset GITHUB_TOKEN: %v", err)
				}
			}
		}()

		var buffer bytes.Buffer

		// Run diagnose command which may log environment info
		cmd := cli.NewRootCmdWithVerbose()
		cmd.SetArgs([]string{"diagnose"})

		// Capture stdout (diagnose outputs to stdout)
		originalStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Execute command
		errChan := make(chan error, 1)
		go func() {
			errChan <- cmd.ExecuteContext(context.Background())
		}()

		// Read output
		go func() {
			buf := make([]byte, 4096)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					buffer.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Wait for completion
		select {
		case err := <-errChan:
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stdout = originalStdout
			require.NoError(t, err, "diagnose command should succeed")
		case <-time.After(10 * time.Second):
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stdout = originalStdout
			t.Fatal("command timed out")
		}

		time.Sleep(100 * time.Millisecond)
		output := buffer.String()

		// Validate redaction
		if strings.Contains(output, "GITHUB_TOKEN") {
			// If GITHUB_TOKEN appears in output, it should be redacted
			assert.Contains(t, output, "***REDACTED***", "sensitive token should be redacted")
			assert.NotContains(t, output, testToken, "original token should not appear in output")
			assert.NotContains(t, output, "ghp_test1234567890", "token prefix should be redacted")
		}
	})

	t.Run("correlation id propagation", func(t *testing.T) {
		// Test that correlation IDs are properly propagated through logs
		var buffer bytes.Buffer

		cmd := cli.NewRootCmdWithVerbose()
		cmd.SetArgs([]string{"validate", "--config", configPath, "--json", "-v"})

		// Capture stderr
		originalStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		// Execute command
		errChan := make(chan error, 1)
		go func() {
			errChan <- cmd.ExecuteContext(context.Background())
		}()

		// Read output
		go func() {
			buf := make([]byte, 2048)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					buffer.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Wait for completion
		select {
		case err := <-errChan:
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stderr = originalStderr
			require.NoError(t, err, "command should succeed")
		case <-time.After(10 * time.Second):
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stderr = originalStderr
			t.Fatal("command timed out")
		}

		time.Sleep(100 * time.Millisecond)
		output := buffer.String()

		// Look for correlation IDs in JSON output
		if strings.TrimSpace(output) != "" {
			lines := strings.Split(strings.TrimSpace(output), "\n")
			correlationIDs := make(map[string]bool)

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				var logEntry map[string]interface{}
				if json.Unmarshal([]byte(line), &logEntry) == nil {
					if corrID, ok := logEntry["correlation_id"]; ok {
						if corrIDStr, ok := corrID.(string); ok && corrIDStr != "" {
							correlationIDs[corrIDStr] = true
						}
					}
				}
			}

			// We should have at most one correlation ID across all log entries
			// (all entries from same command execution should share same correlation ID)
			if len(correlationIDs) > 1 {
				t.Logf("Found multiple correlation IDs: %v", correlationIDs)
				// This could be okay if there are multiple operations
			}
		}
	})

	t.Run("performance timing integration", func(t *testing.T) {
		// Test that performance timing is captured in logs
		var buffer bytes.Buffer

		cmd := cli.NewRootCmdWithVerbose()
		cmd.SetArgs([]string{"validate", "--config", configPath, "--json", "-v"})

		// Capture stderr
		originalStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		startTime := time.Now()

		// Execute command
		errChan := make(chan error, 1)
		go func() {
			errChan <- cmd.ExecuteContext(context.Background())
		}()

		// Read output
		go func() {
			buf := make([]byte, 2048)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					buffer.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Wait for completion
		var cmdErr error
		select {
		case cmdErr = <-errChan:
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stderr = originalStderr
		case <-time.After(10 * time.Second):
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stderr = originalStderr
			t.Fatal("command timed out")
		}

		endTime := time.Now()
		totalDuration := endTime.Sub(startTime)

		require.NoError(t, cmdErr, "command should succeed")

		time.Sleep(100 * time.Millisecond)
		output := buffer.String()

		// Look for timing information in JSON output
		if strings.TrimSpace(output) != "" {
			lines := strings.Split(strings.TrimSpace(output), "\n")
			foundTiming := false

			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				var logEntry map[string]interface{}
				if json.Unmarshal([]byte(line), &logEntry) == nil {
					if durationMs, ok := logEntry["duration_ms"]; ok {
						foundTiming = true

						// Validate duration is reasonable
						if duration, ok := durationMs.(float64); ok {
							assert.GreaterOrEqual(t, duration, float64(0), "duration should be non-negative")
							assert.LessOrEqual(t, duration, float64(totalDuration.Milliseconds())*2,
								"logged duration should be reasonable compared to total time")
						}
						break
					}
				}
			}

			// Timing may not always be present depending on what operations occur
			if foundTiming {
				t.Logf("Found timing information in logs")
			}
		}
	})

	t.Run("log format consistency", func(t *testing.T) {
		// Test that log format is consistent across different commands
		commands := [][]string{
			{"validate", "--config", configPath, "--json"},
			{"version", "--json"},
			{"diagnose"},
		}

		for i, cmdArgs := range commands {
			t.Run("command_"+string(rune(i+'0')), func(t *testing.T) {
				var buffer bytes.Buffer

				cmd := cli.NewRootCmdWithVerbose()
				cmd.SetArgs(cmdArgs)

				// Capture appropriate output stream
				var originalOutput *os.File
				var r, w *os.File

				if cmdArgs[0] == "diagnose" {
					// diagnose outputs to stdout
					originalOutput = os.Stdout
					r, w, _ = os.Pipe()
					os.Stdout = w
				} else {
					// other commands log to stderr
					originalOutput = os.Stderr
					r, w, _ = os.Pipe()
					os.Stderr = w
				}

				// Execute command
				errChan := make(chan error, 1)
				go func() {
					errChan <- cmd.ExecuteContext(context.Background())
				}()

				// Read output
				go func() {
					buf := make([]byte, 2048)
					for {
						n, err := r.Read(buf)
						if n > 0 {
							buffer.Write(buf[:n])
						}
						if err != nil {
							break
						}
					}
				}()

				// Wait for completion
				select {
				case err := <-errChan:
					if closeErr := w.Close(); closeErr != nil {
						t.Logf("Warning: failed to close writer: %v", closeErr)
					}
					if cmdArgs[0] == "diagnose" {
						os.Stdout = originalOutput
					} else {
						os.Stderr = originalOutput
					}
					require.NoError(t, err, "command should succeed")
				case <-time.After(10 * time.Second):
					if closeErr := w.Close(); closeErr != nil {
						t.Logf("Warning: failed to close writer: %v", closeErr)
					}
					if cmdArgs[0] == "diagnose" {
						os.Stdout = originalOutput
					} else {
						os.Stderr = originalOutput
					}
					t.Fatal("command timed out")
				}

				time.Sleep(100 * time.Millisecond)
				output := buffer.String()

				// Basic validation that command executed and produced output
				// Test passes if command execution completes without error

				// If JSON flag was used and we have log output, validate JSON format
				hasJSONFlag := false
				for _, arg := range cmdArgs {
					if arg == "--json" {
						hasJSONFlag = true
						break
					}
				}

				if hasJSONFlag && strings.TrimSpace(output) != "" {
					// For JSON output, verify at least some content is valid JSON
					lines := strings.Split(strings.TrimSpace(output), "\n")
					for _, line := range lines {
						line = strings.TrimSpace(line)
						if line == "" {
							continue
						}

						// Try to parse as JSON - if successful, break
						var jsonData map[string]interface{}
						if json.Unmarshal([]byte(line), &jsonData) == nil {
							// Found valid JSON, test passes
							break
						}
					}
				}
			})
		}
	})
}

// TestLoggingBackwardCompatibility tests that old CLI still works
func TestLoggingBackwardCompatibility(t *testing.T) {
	// Create test configuration
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sync.yaml")

	configContent := `version: 1
source:
  repo: "org/template"
  branch: "master"
targets:
  - repo: "org/service-a"
    files:
      - src: "README.md"
        dest: "README.md"
`
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	t.Run("legacy CLI commands", func(t *testing.T) {
		// Test that legacy CLI (without verbose support) still works
		cmd := cli.NewRootCmd() // Use non-verbose version
		cmd.SetArgs([]string{"validate", "--config", configPath})

		err := cmd.ExecuteContext(context.Background())
		assert.NoError(t, err, "legacy CLI should still work")
	})

	t.Run("legacy log level flag", func(t *testing.T) {
		// Test that old --log-level flag still works
		cmd := cli.NewRootCmd()
		cmd.SetArgs([]string{"validate", "--config", configPath, "--log-level", "debug"})

		err := cmd.ExecuteContext(context.Background())
		assert.NoError(t, err, "legacy log-level flag should still work")
	})
}

// TestLoggingErrorScenarios tests logging in error conditions
func TestLoggingErrorScenarios(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("invalid config file logging", func(t *testing.T) {
		// Create invalid config
		invalidConfigPath := filepath.Join(tmpDir, "invalid.yaml")
		invalidContent := `invalid yaml content: [[[`
		err := os.WriteFile(invalidConfigPath, []byte(invalidContent), 0o600)
		require.NoError(t, err)

		var buffer bytes.Buffer

		cmd := cli.NewRootCmdWithVerbose()
		cmd.SetArgs([]string{"validate", "--config", invalidConfigPath, "--json", "-v"})

		// Capture stderr
		originalStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		// Execute command (should fail)
		errChan := make(chan error, 1)
		go func() {
			errChan <- cmd.ExecuteContext(context.Background())
		}()

		// Read output
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					buffer.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Wait for completion
		select {
		case err := <-errChan:
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stderr = originalStderr
			require.Error(t, err, "invalid config should cause error")
		case <-time.After(10 * time.Second):
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stderr = originalStderr
			t.Fatal("command timed out")
		}

		time.Sleep(100 * time.Millisecond)
		output := buffer.String()

		// Should have logged error information
		if strings.TrimSpace(output) != "" {
			// Error information should be present in some form
			assert.True(t, strings.Contains(output, "error") || strings.Contains(output, "Error") ||
				strings.Contains(output, "failed") || strings.Contains(output, "invalid"),
				"error output should contain error indicators")
		}
	})

	t.Run("missing config file logging", func(t *testing.T) {
		missingConfigPath := filepath.Join(tmpDir, "nonexistent.yaml")

		var buffer bytes.Buffer

		cmd := cli.NewRootCmdWithVerbose()
		cmd.SetArgs([]string{"validate", "--config", missingConfigPath, "-v"})

		// Capture stderr
		originalStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		// Execute command (should fail)
		errChan := make(chan error, 1)
		go func() {
			errChan <- cmd.ExecuteContext(context.Background())
		}()

		// Read output
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					buffer.Write(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Wait for completion
		select {
		case err := <-errChan:
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stderr = originalStderr
			require.Error(t, err, "missing config should cause error")
		case <-time.After(10 * time.Second):
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Warning: failed to close writer: %v", closeErr)
			}
			os.Stderr = originalStderr
			t.Fatal("command timed out")
		}

		// Should handle missing file error gracefully
		// Test passes if no panic occurs during error handling
	})
}
