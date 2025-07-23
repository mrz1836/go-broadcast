package gh

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// CommandRunner interface for executing system commands
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	RunWithInput(ctx context.Context, input []byte, name string, args ...string) ([]byte, error)
}

// realCommandRunner executes actual system commands
type realCommandRunner struct {
	logger *logrus.Logger
}

// NewCommandRunner creates a new command runner
func NewCommandRunner(logger *logrus.Logger) CommandRunner {
	return &realCommandRunner{logger: logger}
}

// Run executes a command and returns its output
func (r *realCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return r.RunWithInput(ctx, nil, name, args...)
}

// RunWithInput executes a command with input and returns its output
func (r *realCommandRunner) RunWithInput(ctx context.Context, input []byte, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	if r.logger != nil && r.logger.IsLevelEnabled(logrus.DebugLevel) {
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

	err := cmd.Run()
	if err != nil {
		// Include stderr in error message for better debugging
		if stderr.Len() > 0 {
			if r.logger != nil {
				r.logger.WithFields(logrus.Fields{
					"command": name,
					"args":    args,
					"stderr":  stderr.String(),
				}).Error("Command failed")
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

	if r.logger != nil && r.logger.IsLevelEnabled(logrus.DebugLevel) {
		r.logger.WithFields(logrus.Fields{
			"command": name,
			"args":    args,
			"output":  stdout.String(),
		}).Debug("Command completed successfully")
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
	return e.Stderr
}

func (e *CommandError) Unwrap() error {
	return e.Err
}
