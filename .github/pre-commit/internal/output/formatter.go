// Package output provides utilities for formatting user-facing output and messages
package output

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Formatter handles all output formatting for the pre-commit system
type Formatter struct {
	colorEnabled bool
	out          io.Writer
	err          io.Writer
}

// Options for configuring the formatter
type Options struct {
	ColorEnabled bool
	Out          io.Writer
	Err          io.Writer
}

// New creates a new formatter with the given options
func New(opts Options) *Formatter {
	f := &Formatter{
		colorEnabled: opts.ColorEnabled,
		out:          opts.Out,
		err:          opts.Err,
	}

	// Default to stdout/stderr if not specified
	if f.out == nil {
		f.out = os.Stdout
	}
	if f.err == nil {
		f.err = os.Stderr
	}

	// Set up color configuration
	color.NoColor = !f.colorEnabled

	return f
}

// NewDefault creates a formatter with default settings, respecting environment variables
func NewDefault() *Formatter {
	// Check for color disable flags
	colorEnabled := !(os.Getenv("NO_COLOR") != "" ||
		os.Getenv("PRE_COMMIT_SYSTEM_COLOR_OUTPUT") == "false")

	return New(Options{
		ColorEnabled: colorEnabled,
		Out:          os.Stdout,
		Err:          os.Stderr,
	})
}

// Success prints a success message with green checkmark
func (f *Formatter) Success(format string, args ...interface{}) {
	if f.colorEnabled {
		color.New(color.FgGreen).Fprintf(f.out, "✓ "+format+"\n", args...)
	} else {
		fmt.Fprintf(f.out, "✓ "+format+"\n", args...)
	}
}

// Error prints an error message with red X
func (f *Formatter) Error(format string, args ...interface{}) {
	if f.colorEnabled {
		color.New(color.FgRed).Fprintf(f.err, "✗ "+format+"\n", args...)
	} else {
		fmt.Fprintf(f.err, "✗ "+format+"\n", args...)
	}
}

// Warning prints a warning message with yellow warning symbol
func (f *Formatter) Warning(format string, args ...interface{}) {
	if f.colorEnabled {
		color.New(color.FgYellow).Fprintf(f.err, "⚠ "+format+"\n", args...)
	} else {
		fmt.Fprintf(f.err, "⚠ "+format+"\n", args...)
	}
}

// Info prints an info message with blue info symbol
func (f *Formatter) Info(format string, args ...interface{}) {
	if f.colorEnabled {
		color.New(color.FgBlue).Fprintf(f.out, "ℹ "+format+"\n", args...)
	} else {
		fmt.Fprintf(f.out, "ℹ "+format+"\n", args...)
	}
}

// Progress prints a progress message with spinning indicator
func (f *Formatter) Progress(format string, args ...interface{}) {
	if f.colorEnabled {
		color.New(color.FgCyan).Fprintf(f.out, "⏳ "+format+"\n", args...)
	} else {
		fmt.Fprintf(f.out, "⏳ "+format+"\n", args...)
	}
}

// Header prints a section header
func (f *Formatter) Header(text string) {
	if f.colorEnabled {
		color.New(color.FgCyan, color.Bold).Fprintf(f.out, "\n%s\n", text)
		color.New(color.FgCyan).Fprintf(f.out, "%s\n", strings.Repeat("─", len(text)))
	} else {
		fmt.Fprintf(f.out, "\n%s\n%s\n", text, strings.Repeat("─", len(text)))
	}
}

// Subheader prints a subsection header
func (f *Formatter) Subheader(text string) {
	if f.colorEnabled {
		color.New(color.FgWhite, color.Bold).Fprintf(f.out, "\n%s:\n", text)
	} else {
		fmt.Fprintf(f.out, "\n%s:\n", text)
	}
}

// Detail prints detailed information with indentation
func (f *Formatter) Detail(format string, args ...interface{}) {
	fmt.Fprintf(f.out, "  "+format+"\n", args...)
}

// Duration formats and prints a duration
func (f *Formatter) Duration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// ParseMakeError analyzes make command output and provides context-aware suggestions
func (f *Formatter) ParseMakeError(command, output string) (message, suggestion string) {
	output = strings.TrimSpace(output)

	switch command {
	case "make lint":
		return f.parseLintError(output)
	case "make fumpt":
		return f.parseFumptError(output)
	case "make mod-tidy":
		return f.parseModTidyError(output)
	default:
		return f.parseGenericMakeError(command, output)
	}
}

// parseLintError analyzes golangci-lint output
func (f *Formatter) parseLintError(output string) (string, string) {
	if strings.Contains(output, "no such file or directory") {
		return "golangci-lint binary not found",
			"Install golangci-lint or ensure it's in your PATH. Run 'make install-lint' if available."
	}

	if strings.Contains(output, "config file") {
		return "golangci-lint configuration issue",
			"Check your .golangci.yml file for syntax errors."
	}

	if strings.Contains(output, "timeout") {
		return "golangci-lint timed out",
			"Increase timeout with PRE_COMMIT_SYSTEM_LINT_TIMEOUT or run 'golangci-lint run' manually."
	}

	// Check for actual linting issues
	issuePatterns := []string{
		`\w+:\d+:\d+:`, // file:line:col: pattern
		`level=error`,  // structured log error
		`ERRO`,         // ERROR level logs
	}

	for _, pattern := range issuePatterns {
		if matched, _ := regexp.MatchString(pattern, output); matched {
			lines := strings.Split(output, "\n")
			issueCount := 0
			for _, line := range lines {
				if matched, _ := regexp.MatchString(pattern, line); matched {
					issueCount++
				}
			}

			return fmt.Sprintf("Found %d linting issue(s)", issueCount),
				"Fix the issues shown above. Run 'make lint' or 'golangci-lint run' to see full details."
		}
	}

	return "Linting failed with unknown error",
		"Run 'make lint' manually to see detailed output."
}

// parseFumptError analyzes gofumpt output
func (f *Formatter) parseFumptError(output string) (string, string) {
	if strings.Contains(output, "no such file or directory") {
		return "gofumpt binary not found",
			"Install gofumpt with 'go install mvdan.cc/gofumpt@latest' or run 'make install-fumpt' if available."
	}

	if strings.Contains(output, "permission denied") {
		return "Permission denied writing files",
			"Check file permissions and ensure you can write to the affected files."
	}

	if strings.Contains(output, "syntax error") || strings.Contains(output, "invalid Go syntax") {
		return "Go syntax errors prevent formatting",
			"Fix syntax errors in your Go files before running fumpt."
	}

	return "Formatting failed",
		"Run 'gofumpt -w .' manually to see detailed errors."
}

// parseModTidyError analyzes go mod tidy output
func (f *Formatter) parseModTidyError(output string) (string, string) {
	if strings.Contains(output, "no go.mod file") {
		return "No go.mod file found",
			"Initialize a Go module with 'go mod init <module-name>'."
	}

	if strings.Contains(output, "network") || strings.Contains(output, "timeout") {
		return "Network error downloading modules",
			"Check your internet connection and proxy settings. Try running 'go mod tidy' manually."
	}

	if strings.Contains(output, "checksum mismatch") {
		return "Module checksum verification failed",
			"Run 'go clean -modcache' and try again, or check for module security issues."
	}

	if strings.Contains(output, "not found") {
		return "Module dependencies not found",
			"Check that all imported modules exist and are accessible."
	}

	return "Module tidy operation failed",
		"Run 'go mod tidy' manually to see detailed errors."
}

// parseGenericMakeError analyzes generic make command errors
func (f *Formatter) parseGenericMakeError(command, output string) (string, string) {
	target := strings.TrimPrefix(command, "make ")

	if strings.Contains(output, "No rule to make target") ||
		strings.Contains(output, "No such file or directory") {
		return fmt.Sprintf("Make target '%s' not found", target),
			fmt.Sprintf("Check your Makefile for the '%s' target or run 'make help' to see available targets.", target)
	}

	if strings.Contains(output, "Permission denied") {
		return "Permission denied",
			"Check file permissions and ensure you have write access to the project directory."
	}

	return fmt.Sprintf("Make command '%s' failed", command),
		fmt.Sprintf("Run '%s' manually to see detailed error output.", command)
}

// FormatFileList formats a list of files for display
func (f *Formatter) FormatFileList(files []string, maxFiles int) string {
	if len(files) == 0 {
		return "no files"
	}

	if len(files) == 1 {
		return files[0]
	}

	if len(files) <= maxFiles {
		return strings.Join(files, ", ")
	}

	shown := strings.Join(files[:maxFiles], ", ")
	return fmt.Sprintf("%s ... and %d more", shown, len(files)-maxFiles)
}

// FormatExecutionStats formats execution statistics
func (f *Formatter) FormatExecutionStats(passed, failed, skipped int, duration time.Duration, fileCount int) string {
	stats := []string{}

	if passed > 0 {
		if f.colorEnabled {
			stats = append(stats, color.GreenString("%d passed", passed))
		} else {
			stats = append(stats, fmt.Sprintf("%d passed", passed))
		}
	}

	if failed > 0 {
		if f.colorEnabled {
			stats = append(stats, color.RedString("%d failed", failed))
		} else {
			stats = append(stats, fmt.Sprintf("%d failed", failed))
		}
	}

	if skipped > 0 {
		if f.colorEnabled {
			stats = append(stats, color.YellowString("%d skipped", skipped))
		} else {
			stats = append(stats, fmt.Sprintf("%d skipped", skipped))
		}
	}

	result := strings.Join(stats, ", ")
	if fileCount > 0 {
		result += fmt.Sprintf(" on %d file(s)", fileCount)
	}
	result += fmt.Sprintf(" in %s", f.Duration(duration))

	return result
}

// Highlight highlights specific text within a string
func (f *Formatter) Highlight(text, highlight string) string {
	if !f.colorEnabled {
		return text
	}
	return strings.ReplaceAll(text, highlight, color.YellowString(highlight))
}

// CodeBlock formats text as a code block
func (f *Formatter) CodeBlock(text string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if f.colorEnabled {
			color.New(color.FgWhite, color.Faint).Fprintf(f.out, "    %s\n", line)
		} else {
			fmt.Fprintf(f.out, "    %s\n", line)
		}
	}
}

// SuggestAction prints an actionable suggestion
func (f *Formatter) SuggestAction(action string) {
	if f.colorEnabled {
		color.New(color.FgMagenta).Fprintf(f.out, "💡 %s\n", action)
	} else {
		fmt.Fprintf(f.out, "💡 %s\n", action)
	}
}
