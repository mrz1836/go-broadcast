// Package output provides colored output functions for the CLI.
package output

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/fatih/color"
)

//nolint:gochecknoglobals // Output package requires package-level state for consistent formatting
var (
	// Color functions
	successColor = color.New(color.FgGreen, color.Bold)
	infoColor    = color.New(color.FgCyan)
	warnColor    = color.New(color.FgYellow)
	errorColor   = color.New(color.FgRed, color.Bold)

	// Output writers
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr

	// Mutex for thread-safe output
	mu sync.Mutex
)

// Init initializes the output system
func Init() {
	// Enable color output
	color.NoColor = false
}

// SetStdout sets the standard output writer (useful for testing)
func SetStdout(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()

	stdout = w
}

// SetStderr sets the standard error writer (useful for testing)
func SetStderr(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()

	stderr = w
}

// Stdout returns the current stdout writer
func Stdout() io.Writer {
	mu.Lock()
	defer mu.Unlock()

	return stdout
}

// Stderr returns the current stderr writer
func Stderr() io.Writer {
	mu.Lock()
	defer mu.Unlock()

	return stderr
}

// Success prints a success message in green
func Success(msg string) {
	mu.Lock()
	defer mu.Unlock()
	_, _ = successColor.Fprintln(stdout, msg)
}

// Successf prints a formatted success message
func Successf(format string, args ...interface{}) {
	Success(fmt.Sprintf(format, args...))
}

// Info prints an info message in cyan
func Info(msg string) {
	mu.Lock()
	defer mu.Unlock()
	_, _ = infoColor.Fprintln(stdout, msg)
}

// Infof prints a formatted info message
func Infof(format string, args ...interface{}) {
	Info(fmt.Sprintf(format, args...))
}

// Warn prints a warning message in yellow
func Warn(msg string) {
	mu.Lock()
	defer mu.Unlock()
	_, _ = warnColor.Fprintln(stderr, msg)
}

// Warnf prints a formatted warning message
func Warnf(format string, args ...interface{}) {
	Warn(fmt.Sprintf(format, args...))
}

// Error prints an error message in red
func Error(msg string) {
	mu.Lock()
	defer mu.Unlock()
	_, _ = errorColor.Fprintln(stderr, msg)
}

// Errorf prints a formatted error message
func Errorf(format string, args ...interface{}) {
	Error(fmt.Sprintf(format, args...))
}

// Plain prints a message without color
func Plain(msg string) {
	mu.Lock()
	defer mu.Unlock()
	_, _ = fmt.Fprintln(stdout, msg)
}

// Plainf prints a formatted message without color
func Plainf(format string, args ...interface{}) {
	Plain(fmt.Sprintf(format, args...))
}

// Progress represents a progress indicator
type Progress struct {
	message string
	spinner []string
	index   int
	done    chan bool
	mu      sync.Mutex
}

// NewProgress creates a new progress indicator
func NewProgress(message string) *Progress {
	return &Progress{
		message: message,
		spinner: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:    make(chan bool),
	}
}

// Start begins showing the progress indicator
func (p *Progress) Start() {
	go func() {
		for {
			select {
			case <-p.done:
				return
			default:
				p.mu.Lock()
				_, _ = fmt.Fprintf(stdout, "\r%s %s", p.spinner[p.index], p.message)
				p.index = (p.index + 1) % len(p.spinner)
				p.mu.Unlock()
			}
		}
	}()
}

// Stop stops the progress indicator
func (p *Progress) Stop() {
	p.done <- true
	_, _ = fmt.Fprint(stdout, "\r\033[K") // Clear line
}

// StopWithSuccess stops with a success message
func (p *Progress) StopWithSuccess(msg string) {
	p.Stop()
	Success(msg)
}

// StopWithError stops with an error message
func (p *Progress) StopWithError(msg string) {
	p.Stop()
	Error(msg)
}
