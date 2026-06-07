package cli

import "errors"

// exitCodeError carries a requested process exit code for CLI paths that need
// a specific status while still returning errors from testable command handlers.
type exitCodeError struct {
	code int
	err  error
}

func (e *exitCodeError) Error() string {
	return e.err.Error()
}

func (e *exitCodeError) Unwrap() error {
	return e.err
}

func (e *exitCodeError) ExitCode() int {
	return e.code
}

func newExitCodeError(code int, err error) error {
	return &exitCodeError{code: code, err: err}
}

// ExitCodeForError returns the process exit code requested by a CLI error.
func ExitCodeForError(err error) int {
	var exitErr *exitCodeError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

// Common CLI errors
var (
	// ErrConfigFileNotFound indicates the configuration file was not found
	ErrConfigFileNotFound = errors.New("configuration file not found")

	// ErrNoMatchingTargets indicates no targets matched the filter
	ErrNoMatchingTargets = errors.New("no matching targets found")

	// ErrNilConfig indicates a nil configuration was passed to a function that requires a valid config
	ErrNilConfig = errors.New("config cannot be nil")

	// ErrDatabaseFileNotFound indicates the database file was not found
	ErrDatabaseFileNotFound = errors.New("database file not found")
)
