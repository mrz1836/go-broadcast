// Package output provides colored output functions for the CLI.
package output

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
)

// Writer defines the interface for output operations
type Writer interface {
	Success(msg string)
	Successf(format string, args ...interface{})
	Info(msg string)
	Infof(format string, args ...interface{})
	Warn(msg string)
	Warnf(format string, args ...interface{})
	Error(msg string)
	Errorf(format string, args ...interface{})
	Plain(msg string)
	Plainf(format string, args ...interface{})
}

// Compile-time interface check
var _ Writer = (*ColoredWriter)(nil)

// ColoredWriter implements Writer with colored output
type ColoredWriter struct {
	successColor *color.Color
	infoColor    *color.Color
	warnColor    *color.Color
	errorColor   *color.Color
	stdout       io.Writer
	stderr       io.Writer
	mu           sync.Mutex
}

// NewColoredWriter creates a new ColoredWriter instance.
// If stdout or stderr is nil, io.Discard is used as a safe default.
func NewColoredWriter(stdout, stderr io.Writer) *ColoredWriter {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return &ColoredWriter{
		successColor: color.New(color.FgGreen, color.Bold),
		infoColor:    color.New(color.FgCyan),
		warnColor:    color.New(color.FgYellow),
		errorColor:   color.New(color.FgRed, color.Bold),
		stdout:       stdout,
		stderr:       stderr,
	}
}

// Success prints a success message in green
func (w *ColoredWriter) Success(msg string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = w.successColor.Fprintln(w.stdout, msg)
}

// Successf prints a formatted success message
func (w *ColoredWriter) Successf(format string, args ...interface{}) {
	w.Success(fmt.Sprintf(format, args...))
}

// Info prints an info message in cyan
func (w *ColoredWriter) Info(msg string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = w.infoColor.Fprintln(w.stdout, msg)
}

// Infof prints a formatted info message
func (w *ColoredWriter) Infof(format string, args ...interface{}) {
	w.Info(fmt.Sprintf(format, args...))
}

// Warn prints a warning message in yellow
func (w *ColoredWriter) Warn(msg string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = w.warnColor.Fprintln(w.stderr, msg)
}

// Warnf prints a formatted warning message
func (w *ColoredWriter) Warnf(format string, args ...interface{}) {
	w.Warn(fmt.Sprintf(format, args...))
}

// Error prints an error message in red
func (w *ColoredWriter) Error(msg string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = w.errorColor.Fprintln(w.stderr, msg)
}

// Errorf prints a formatted error message
func (w *ColoredWriter) Errorf(format string, args ...interface{}) {
	w.Error(fmt.Sprintf(format, args...))
}

// Plain prints a message without color
func (w *ColoredWriter) Plain(msg string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = fmt.Fprintln(w.stdout, msg)
}

// Plainf prints a formatted message without color
func (w *ColoredWriter) Plainf(format string, args ...interface{}) {
	w.Plain(fmt.Sprintf(format, args...))
}

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

// Scope provides test-isolated output capture.
// It captures stdout and stderr in thread-safe buffers while the scope is active.
// Call Restore() in defer to restore original writers.
type Scope struct {
	prevStdout io.Writer
	prevStderr io.Writer
	Stdout     *bytes.Buffer
	Stderr     *bytes.Buffer
}

// CaptureOutput creates a new output scope for testing.
// This function atomically captures the current stdout/stderr and replaces them
// with buffers for test isolation. Call scope.Restore() in defer to restore.
//
// Usage:
//
//	scope := output.CaptureOutput()
//	defer scope.Restore()
//	// ... run code that produces output ...
//	assert.Contains(t, scope.Stdout.String(), "expected")
func CaptureOutput() *Scope {
	mu.Lock()
	defer mu.Unlock()
	scope := &Scope{
		prevStdout: stdout,
		prevStderr: stderr,
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
	}
	stdout = scope.Stdout
	stderr = scope.Stderr
	return scope
}

// Restore restores the original output writers.
// This should be called in a defer immediately after CaptureOutput().
func (s *Scope) Restore() {
	mu.Lock()
	defer mu.Unlock()
	stdout = s.prevStdout
	stderr = s.prevStderr
}

// Progress represents a progress indicator.
// It is safe to call Stop() multiple times or without calling Start().
type Progress struct {
	message string
	spinner []string
	index   int
	done    chan struct{} // Use struct{} for signal channels (zero allocation)
	started atomic.Bool   // Prevents double-start goroutine leak
	stopped atomic.Bool   // Prevents double-stop and deadlock
	mu      sync.Mutex
}

// NewProgress creates a new progress indicator
func NewProgress(message string) *Progress {
	return &Progress{
		message: message,
		spinner: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:    make(chan struct{}),
	}
}

// Start begins showing the progress indicator.
// Calling Start() multiple times is safe; subsequent calls are no-ops.
func (p *Progress) Start() {
	// Prevent double-start which would leak goroutines
	if p.started.Swap(true) {
		return
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-p.done:
				return
			case <-ticker.C:
				p.mu.Lock()
				mu.Lock()
				_, _ = fmt.Fprintf(stdout, "\r%s %s", p.spinner[p.index], p.message)
				mu.Unlock()
				p.index = (p.index + 1) % len(p.spinner)
				p.mu.Unlock()
			}
		}
	}()
}

// Stop stops the progress indicator.
// It is safe to call Stop() multiple times or without calling Start().
func (p *Progress) Stop() {
	// Prevent deadlock: only stop if started and not already stopped
	if !p.started.Load() || p.stopped.Swap(true) {
		return
	}
	close(p.done) // Close instead of send - idiomatic Go
	mu.Lock()
	_, _ = fmt.Fprint(stdout, "\r\033[K") // Clear line
	mu.Unlock()
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
