package gh

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

// CommandRunner interface for executing system commands
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	RunWithInput(ctx context.Context, input []byte, name string, args ...string) ([]byte, error)
}

// realCommandRunner executes actual system commands
type realCommandRunner struct {
	logger    *logrus.Logger
	logConfig *logging.LogConfig
}

// NewCommandRunner creates a new command runner.
//
// Parameters:
// - logger: Logger instance for general logging
// - logConfig: Configuration for debug logging and verbose settings
//
// Returns:
// - CommandRunner interface implementation for executing system commands
func NewCommandRunner(logger *logrus.Logger, logConfig *logging.LogConfig) CommandRunner {
	return &realCommandRunner{
		logger:    logger,
		logConfig: logConfig,
	}
}

// Run executes a command and returns its output
func (r *realCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.RunWithInput(ctx, nil, name, args...)
}

// RunWithInput executes a command with input and returns its output.
//
// This method provides detailed visibility into GitHub CLI command execution when
// API debug logging is enabled, including request details, timing, response size,
// and selective response body logging for troubleshooting.
//
// Parameters:
// - ctx: Context for cancellation and timeout control
// - input: Optional input data to pass to the command
// - name: Command name to execute
// - args: Command arguments
//
// Returns:
// - Command output as byte slice
// - Error if command execution fails
//
// Side Effects:
// - Logs detailed request/response information when --debug-api flag is enabled
// - Records command timing and response size metrics
func (r *realCommandRunner) RunWithInput(ctx context.Context, input []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	// Create logger entry only if logger is not nil to avoid panic
	var logger *logrus.Entry
	if r.logger != nil {
		logger = logging.WithStandardFields(r.logger, r.logConfig, logging.ComponentNames.API)
	}

	// Debug logging when --debug-api flag is enabled
	if r.logConfig != nil && r.logConfig.Debug.API && logger != nil {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.Operation: logging.OperationTypes.APIRequest,
			"args":                           args,
			logging.StandardFields.Timestamp: time.Now().Format(time.RFC3339),
		}).Debug("GitHub CLI request")

		// Log request body/input if present
		if input != nil {
			logger.WithFields(logrus.Fields{
				logging.StandardFields.ContentSize: len(input),
				"input":                            string(input),
			}).Trace("Request input")
		}

		// Log request field parsing for GitHub CLI -f parameters
		for i, arg := range args {
			if arg == "-f" && i+1 < len(args) {
				logger.WithField("field", args[i+1]).Trace("Request field")
			}
		}
	} else if r.logger != nil && r.logger.IsLevelEnabled(logrus.DebugLevel) {
		// Basic logging for backwards compatibility
		r.logger.WithFields(logrus.Fields{
			"command": name,
			"args":    args,
		}).Debug("Executing command")
	}

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}

	// Execute command with timing
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	// Response logging when --debug-api flag is enabled
	if r.logConfig != nil && r.logConfig.Debug.API && logger != nil {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.DurationMs:  duration.Milliseconds(),
			logging.StandardFields.ContentSize: stdout.Len(),
			logging.StandardFields.Error:       err,
			logging.StandardFields.Status:      "response_received",
		}).Debug("GitHub CLI response")

		// Log response body for small responses (with size limits)
		if err == nil && stdout.Len() > 0 && stdout.Len() < 1024 {
			logger.WithField("response", stdout.String()).Trace("Response body")
		}

		// Log stderr for debugging even on success
		if stderr.Len() > 0 {
			logger.WithField("stderr", stderr.String()).Trace("Response stderr")
		}
	}

	if err != nil {
		// Include stderr in error message for better debugging
		if stderr.Len() > 0 {
			if r.logger != nil {
				// 404 errors are expected for many operations (disabled features, missing resources)
				// Log them at DEBUG level to avoid confusion
				stderrStr := stderr.String()
				is404 := strings.Contains(stderrStr, "404") ||
					strings.Contains(stderrStr, "Not Found") ||
					strings.Contains(stderrStr, "could not resolve")

				if r.logConfig != nil && r.logConfig.Debug.API {
					// Error logging with timing context
					fields := logrus.Fields{
						"command":                         name,
						"args":                            args,
						"stderr":                          stderr.String(),
						logging.StandardFields.DurationMs: duration.Milliseconds(),
					}

					if is404 {
						fields[logging.StandardFields.Status] = "not_found"
						logger.WithFields(fields).Debug("Resource not found (HTTP 404)")
					} else {
						fields[logging.StandardFields.Status] = "failed"
						logger.WithFields(fields).Error("GitHub CLI command failed")
					}
				} else {
					// Basic error logging for backwards compatibility
					fields := logrus.Fields{
						logging.StandardFields.Component: logging.ComponentNames.API,
						"command":                        name,
						"args":                           args,
						"stderr":                         stderr.String(),
					}

					if is404 {
						fields[logging.StandardFields.Status] = "not_found"
						r.logger.WithFields(fields).Debug("Resource not found (HTTP 404)")
					} else {
						fields[logging.StandardFields.Status] = "failed"
						r.logger.WithFields(fields).Error("Command failed")
					}
				}
			}
			return nil, &CommandError{
				Command: name,
				Args:    args,
				Stderr:  stderr.String(),
				Err:     err,
			}
		}
		return nil, err
	}

	// Log successful completion
	if r.logConfig == nil || !r.logConfig.Debug.API {
		// Basic logging for backwards compatibility
		if r.logger != nil && r.logger.IsLevelEnabled(logrus.DebugLevel) {
			r.logger.WithFields(logrus.Fields{
				"command": name,
				"args":    args,
				"output":  stdout.String(),
			}).Debug("Command completed successfully")
		}
	}

	return stdout.Bytes(), nil
}

// CommandError provides detailed error information from command execution
type CommandError struct {
	Command string
	Args    []string
	Stderr  string
	Err     error
}

func (e *CommandError) Error() string {
	if e.Stderr != "" {
		return e.Stderr
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("command %s failed", e.Command)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}
