package gh

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestNewCommandRunner(t *testing.T) {
	logger := logrus.New()
	logConfig := &logging.LogConfig{
		Debug: logging.DebugFlags{
			API: true,
		},
	}

	runner := NewCommandRunner(logger, logConfig)
	require.NotNil(t, runner)

	// Check that it returns the correct type
	_, ok := runner.(*realCommandRunner)
	require.True(t, ok)
}

func TestRealCommandRunner_Run(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
	}{
		{
			name:        "SimpleEchoCommand",
			command:     "echo",
			args:        []string{"hello"},
			expectError: false,
		},
		{
			name:        "NonExistentCommand",
			command:     "nonexistentcommand12345",
			args:        []string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &realCommandRunner{
				logger:    logrus.New(),
				logConfig: &logging.LogConfig{},
			}

			output, err := runner.Run(context.Background(), tt.command, tt.args...)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, output)
			}
		})
	}
}

func TestRealCommandRunner_RunWithInput(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		args           []string
		input          []byte
		logConfig      *logging.LogConfig
		expectError    bool
		expectInOutput string
	}{
		{
			name:           "EchoWithNoInput",
			command:        "echo",
			args:           []string{"test"},
			input:          nil,
			logConfig:      &logging.LogConfig{},
			expectError:    false,
			expectInOutput: "test",
		},
		{
			name:           "CatWithInput",
			command:        "cat",
			args:           []string{},
			input:          []byte("hello world"),
			logConfig:      &logging.LogConfig{},
			expectError:    false,
			expectInOutput: "hello world",
		},
		{
			name:    "CommandWithDebugLogging",
			command: "echo",
			args:    []string{"debug test"},
			input:   nil,
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{
					API: true,
				},
			},
			expectError:    false,
			expectInOutput: "debug test",
		},
		{
			name:           "CommandWithMultipleArgs",
			command:        "echo",
			args:           []string{"-n", "no", "newline"},
			input:          nil,
			logConfig:      &logging.LogConfig{},
			expectError:    false,
			expectInOutput: "no newline",
		},
		{
			name:    "NonExistentCommandWithDebug",
			command: "nonexistentcmd",
			args:    []string{"arg1"},
			input:   []byte("test input"),
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{
					API: true,
				},
			},
			expectError: true,
		},
		{
			name:        "CommandWithStderr",
			command:     "sh",
			args:        []string{"-c", "echo 'error' >&2; exit 1"},
			input:       nil,
			logConfig:   &logging.LogConfig{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a logger with a buffer to capture logs
			var logBuffer bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&logBuffer)
			logger.SetLevel(logrus.TraceLevel)

			runner := &realCommandRunner{
				logger:    logger,
				logConfig: tt.logConfig,
			}

			ctx := context.Background()
			output, err := runner.RunWithInput(ctx, tt.input, tt.command, tt.args...)

			if tt.expectError {
				require.Error(t, err)
				// Check for CommandError type
				var cmdErr *CommandError
				if errors.As(err, &cmdErr) {
					require.Equal(t, tt.command, cmdErr.Command)
					require.Equal(t, tt.args, cmdErr.Args)
				}
			} else {
				require.NoError(t, err)
				require.Contains(t, string(output), tt.expectInOutput)
			}

			// Verify debug logging behavior
			logs := logBuffer.String()
			if tt.logConfig != nil && tt.logConfig.Debug.API {
				require.Contains(t, logs, "GitHub CLI request")
				if tt.input != nil {
					require.Contains(t, logs, "Request input")
				}
				require.Contains(t, logs, "GitHub CLI response")
			}
		})
	}
}

func TestRealCommandRunner_RunWithInput_Context(t *testing.T) {
	runner := &realCommandRunner{
		logger:    logrus.New(),
		logConfig: &logging.LogConfig{},
	}

	// Test with canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := runner.RunWithInput(ctx, nil, "sleep", "10")
	require.Error(t, err)
}

func TestRealCommandRunner_RunWithInput_Timeout(t *testing.T) {
	runner := &realCommandRunner{
		logger:    logrus.New(),
		logConfig: &logging.LogConfig{},
	}

	// Test with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := runner.RunWithInput(ctx, nil, "sleep", "1")
	require.Error(t, err)
}

func TestRealCommandRunner_RunWithInput_LargeOutput(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logBuffer)
	logger.SetLevel(logrus.TraceLevel)

	runner := &realCommandRunner{
		logger: logger,
		logConfig: &logging.LogConfig{
			Debug: logging.DebugFlags{
				API: true,
			},
		},
	}

	// Generate output larger than 1024 bytes
	largeString := string(bytes.Repeat([]byte("a"), 2000))
	output, err := runner.RunWithInput(context.Background(), nil, "echo", largeString)
	require.NoError(t, err)
	require.Contains(t, string(output), largeString)

	// Check that large response is not logged in trace
	logs := logBuffer.String()
	require.NotContains(t, logs, "Response body")
}

func TestRealCommandRunner_RunWithInput_FieldParsing(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logBuffer)
	logger.SetLevel(logrus.TraceLevel)

	runner := &realCommandRunner{
		logger: logger,
		logConfig: &logging.LogConfig{
			Debug: logging.DebugFlags{
				API: true,
			},
		},
	}

	// Test with -f flag parsing
	output, err := runner.RunWithInput(context.Background(), nil, "echo", "-f", "field1=value1", "-f", "field2=value2")
	require.NoError(t, err)
	require.NotNil(t, output)

	// Check that fields are logged
	logs := logBuffer.String()
	require.Contains(t, logs, "Request field")
	require.Contains(t, logs, "field1=value1")
	require.Contains(t, logs, "field2=value2")
}

func TestCommandError_Error(t *testing.T) {
	err := &CommandError{
		Command: "git",
		Args:    []string{"status"},
		Stderr:  "fatal: not a git repository",
		Err:     errors.New("exit status 128"), //nolint:err113 // test error
	}

	require.Equal(t, "fatal: not a git repository", err.Error())
}

func TestCommandError_Unwrap(t *testing.T) {
	baseErr := errors.New("exit status 128") //nolint:err113 // test error
	err := &CommandError{
		Command: "git",
		Args:    []string{"status"},
		Stderr:  "fatal: not a git repository",
		Err:     baseErr,
	}

	unwrapped := err.Unwrap()
	require.Equal(t, baseErr, unwrapped)
}

func TestRealCommandRunner_BackwardsCompatibilityLogging(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logBuffer)
	logger.SetLevel(logrus.DebugLevel)

	// Test with nil logConfig (backwards compatibility)
	runner := &realCommandRunner{
		logger:    logger,
		logConfig: nil,
	}

	output, err := runner.RunWithInput(context.Background(), nil, "echo", "backwards")
	require.NoError(t, err)
	require.Contains(t, string(output), "backwards")

	// Should still log basic debug info
	logs := logBuffer.String()
	require.Contains(t, logs, "Executing command")
	require.Contains(t, logs, "Command completed successfully")
}

func TestRealCommandRunner_RunWithInput_NilLogger(t *testing.T) {
	// Test with nil logger - should not panic
	runner := &realCommandRunner{
		logger:    nil,
		logConfig: nil,
	}

	output, err := runner.RunWithInput(context.Background(), nil, "echo", "test")
	require.NoError(t, err)
	require.Contains(t, string(output), "test")
}

func TestRealCommandRunner_RunWithInput_NilLoggerWithDebug(t *testing.T) {
	// Test with nil logger but debug config - should not panic
	runner := &realCommandRunner{
		logger: nil,
		logConfig: &logging.LogConfig{
			Debug: logging.DebugFlags{
				API: true,
			},
		},
	}

	output, err := runner.RunWithInput(context.Background(), nil, "echo", "test")
	require.NoError(t, err)
	require.Contains(t, string(output), "test")
}

func TestCommandError_Error_EmptyStderr(t *testing.T) {
	// Test with empty stderr - should fallback to underlying error
	baseErr := errors.New("exit status 1") //nolint:err113 // test error
	err := &CommandError{
		Command: "git",
		Args:    []string{"status"},
		Stderr:  "",
		Err:     baseErr,
	}

	require.Equal(t, "exit status 1", err.Error())
}

func TestCommandError_Error_EmptyStderrNilErr(t *testing.T) {
	// Test with empty stderr and nil error - should return command failed message
	err := &CommandError{
		Command: "git",
		Args:    []string{"status"},
		Stderr:  "",
		Err:     nil,
	}

	require.Equal(t, "command git failed", err.Error())
}

func TestCommandError_Error_StderrPriority(t *testing.T) {
	// Test that stderr takes priority when present
	baseErr := errors.New("exit status 1") //nolint:err113 // test error
	err := &CommandError{
		Command: "git",
		Args:    []string{"status"},
		Stderr:  "fatal: not a git repository",
		Err:     baseErr,
	}

	require.Equal(t, "fatal: not a git repository", err.Error())
}

func TestRealCommandRunner_404ErrorLogging(t *testing.T) {
	tests := []struct {
		name              string
		stderr            string
		expectDebugLevel  bool
		expectErrorLevel  bool
		expectedLogStatus string
	}{
		{
			name:              "404NotFound",
			stderr:            "gh: Not Found (HTTP 404)\n",
			expectDebugLevel:  true,
			expectErrorLevel:  false,
			expectedLogStatus: "not_found",
		},
		{
			name:              "404InMessage",
			stderr:            "Error: 404 page not found",
			expectDebugLevel:  true,
			expectErrorLevel:  false,
			expectedLogStatus: "not_found",
		},
		{
			name:              "CouldNotResolve",
			stderr:            "Error: could not resolve repository",
			expectDebugLevel:  true,
			expectErrorLevel:  false,
			expectedLogStatus: "not_found",
		},
		{
			name:              "RealError",
			stderr:            "Error: authentication failed",
			expectDebugLevel:  false,
			expectErrorLevel:  true,
			expectedLogStatus: "failed",
		},
		{
			name:              "RateLimitError",
			stderr:            "Error: API rate limit exceeded",
			expectDebugLevel:  false,
			expectErrorLevel:  true,
			expectedLogStatus: "failed",
		},
		{
			name:              "NetworkError",
			stderr:            "Error: network timeout",
			expectDebugLevel:  false,
			expectErrorLevel:  true,
			expectedLogStatus: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a logger with a buffer to capture logs
			var logBuffer bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&logBuffer)
			logger.SetLevel(logrus.DebugLevel)
			logger.SetFormatter(&logrus.TextFormatter{
				DisableColors:    true,
				DisableTimestamp: true,
			})

			runner := &realCommandRunner{
				logger:    logger,
				logConfig: &logging.LogConfig{},
			}

			// Simulate a failed command with stderr
			ctx := context.Background()
			_, err := runner.RunWithInput(ctx, nil, "sh", "-c", "echo '"+tt.stderr+"' >&2; exit 1")

			// Should always return an error
			require.Error(t, err)

			logs := logBuffer.String()

			if tt.expectDebugLevel {
				// Should log at DEBUG level with "Resource not found (HTTP 404)" message
				require.Contains(t, logs, "level=debug")
				require.Contains(t, logs, "Resource not found (HTTP 404)")
				require.Contains(t, logs, "status=not_found")
				require.NotContains(t, logs, "level=error")
			}

			if tt.expectErrorLevel {
				// Should log at ERROR level with "Command failed" message
				require.Contains(t, logs, "level=error")
				require.Contains(t, logs, "Command failed")
				require.Contains(t, logs, "status=failed")
				require.NotContains(t, logs, "Resource not found")
			}
		})
	}
}

func TestRealCommandRunner_404ErrorLogging_WithDebugAPI(t *testing.T) {
	tests := []struct {
		name              string
		stderr            string
		expectDebugLevel  bool
		expectErrorLevel  bool
		expectedLogStatus string
	}{
		{
			name:              "404WithDebugAPI",
			stderr:            "gh: Not Found (HTTP 404)\n",
			expectDebugLevel:  true,
			expectErrorLevel:  false,
			expectedLogStatus: "not_found",
		},
		{
			name:              "RealErrorWithDebugAPI",
			stderr:            "Error: authentication failed",
			expectDebugLevel:  false,
			expectErrorLevel:  true,
			expectedLogStatus: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a logger with a buffer to capture logs
			var logBuffer bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&logBuffer)
			logger.SetLevel(logrus.DebugLevel)
			logger.SetFormatter(&logrus.TextFormatter{
				DisableColors:    true,
				DisableTimestamp: true,
			})

			runner := &realCommandRunner{
				logger: logger,
				logConfig: &logging.LogConfig{
					Debug: logging.DebugFlags{
						API: true,
					},
				},
			}

			// Simulate a failed command with stderr
			ctx := context.Background()
			_, err := runner.RunWithInput(ctx, nil, "sh", "-c", "echo '"+tt.stderr+"' >&2; exit 1")

			// Should always return an error
			require.Error(t, err)

			logs := logBuffer.String()

			if tt.expectDebugLevel {
				// Should log at DEBUG level with "Resource not found (HTTP 404)" message
				require.Contains(t, logs, "level=debug")
				require.Contains(t, logs, "Resource not found (HTTP 404)")
				require.Contains(t, logs, "status=not_found")
				require.NotContains(t, logs, "level=error")
				// Should include timing info when debug API is enabled
				require.Contains(t, logs, "duration_ms")
			}

			if tt.expectErrorLevel {
				// Should log at ERROR level
				require.Contains(t, logs, "level=error")
				require.Contains(t, logs, "GitHub CLI command failed")
				require.Contains(t, logs, "status=failed")
				require.NotContains(t, logs, "Resource not found")
				// Should include timing info when debug API is enabled
				require.Contains(t, logs, "duration_ms")
			}
		})
	}
}
