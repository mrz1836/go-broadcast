// Package makewrap provides pre-commit checks that wrap make commands
package makewrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/shared"
)

// FumptCheck runs gofumpt via make
type FumptCheck struct {
	sharedCtx *shared.Context
	timeout   time.Duration
}

// NewFumptCheck creates a new fumpt check
func NewFumptCheck() *FumptCheck {
	return &FumptCheck{
		sharedCtx: shared.NewContext(),
		timeout:   30 * time.Second, // 30 second timeout for fumpt
	}
}

// NewFumptCheckWithSharedContext creates a new fumpt check with shared context
func NewFumptCheckWithSharedContext(sharedCtx *shared.Context) *FumptCheck {
	return &FumptCheck{
		sharedCtx: sharedCtx,
		timeout:   30 * time.Second,
	}
}

// NewFumptCheckWithConfig creates a new fumpt check with shared context and custom timeout
func NewFumptCheckWithConfig(sharedCtx *shared.Context, timeout time.Duration) *FumptCheck {
	return &FumptCheck{
		sharedCtx: sharedCtx,
		timeout:   timeout,
	}
}

// Name returns the name of the check
func (c *FumptCheck) Name() string {
	return "fumpt"
}

// Description returns a brief description of the check
func (c *FumptCheck) Description() string {
	return "Format Go code with gofumpt"
}

// Metadata returns comprehensive metadata about the check
func (c *FumptCheck) Metadata() interface{} {
	return CheckMetadata{
		Name:              "fumpt",
		Description:       "Format Go code with gofumpt (stricter gofmt)",
		FilePatterns:      []string{"*.go"},
		EstimatedDuration: 3 * time.Second,
		Dependencies:      []string{"fumpt"}, // make target
		DefaultTimeout:    c.timeout,
		Category:          "formatting",
		RequiresFiles:     true,
	}
}

// Run executes the fumpt check
func (c *FumptCheck) Run(ctx context.Context, files []string) error {
	// Early return if no files to process
	if len(files) == 0 {
		return nil
	}

	// Check if make fumpt is available
	if c.sharedCtx.HasMakeTarget(ctx, "fumpt") {
		// Run make fumpt with timeout
		return c.runMakeFumpt(ctx)
	}

	// Fall back to direct gofumpt if available
	return c.runDirectFumpt(ctx, files)
}

// FilterFiles filters to only Go files
func (c *FumptCheck) FilterFiles(files []string) []string {
	var filtered []string
	for _, file := range files {
		if strings.HasSuffix(file, ".go") {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// runMakeFumpt runs make fumpt with proper error handling
func (c *FumptCheck) runMakeFumpt(ctx context.Context) error {
	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Add timeout for make command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "make", "fumpt")
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stdout.String() + stderr.String()

		// Check if it's a context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolExecutionError(
				"make fumpt",
				output,
				fmt.Sprintf("Fumpt check timed out after %v. Consider increasing PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT or run 'make fumpt' manually.", c.timeout),
			)
		}

		// Parse the error for better context
		if strings.Contains(output, "No rule to make target") {
			return prerrors.NewMakeTargetNotFoundError(
				"fumpt",
				"Create a 'fumpt' target in your Makefile or disable fumpt with PRE_COMMIT_SYSTEM_ENABLE_FUMPT=false",
			)
		}

		// Enhanced gofumpt detection and PATH diagnostics
		if strings.Contains(output, "gofumpt") && (strings.Contains(output, "not found") || strings.Contains(output, "command not found")) {
			// Try to provide better diagnostics
			gopath, err := exec.LookPath("go")
			if err != nil {
				return prerrors.NewToolNotFoundError(
					"go",
					"Go is not installed or not in PATH. Install Go first: https://golang.org/doc/install",
				)
			}

			// Check if gofumpt exists in common locations
			diagnostics := []string{
				"gofumpt is not available in the current PATH.",
				"Common causes:",
				"1. gofumpt is not installed - run: go install mvdan.cc/gofumpt@v0.7.0",
				"2. GOPATH/bin or GOROOT/bin is not in PATH",
				"3. Different environment between terminal and git GUI",
				"",
				"Current diagnostics:",
				fmt.Sprintf("- Go binary found at: %s", gopath),
			}

			// Try to detect GOPATH
			goCmd := exec.CommandContext(ctx, "go", "env", "GOPATH")
			if gopathBytes, err := goCmd.Output(); err == nil {
				gopath := strings.TrimSpace(string(gopathBytes))
				diagnostics = append(diagnostics, fmt.Sprintf("- GOPATH: %s", gopath))
				diagnostics = append(diagnostics, fmt.Sprintf("- Expected gofumpt location: %s/bin/gofumpt", gopath))
			}

			return prerrors.NewToolNotFoundError(
				"gofumpt",
				strings.Join(diagnostics, "\n"),
			)
		}

		// Enhanced PATH-related error detection
		if strings.Contains(output, "installation failed or not in PATH") {
			return prerrors.NewToolExecutionError(
				"make fumpt",
				output,
				"gofumpt installation succeeded but the binary is not accessible. This commonly happens in git GUI applications where PATH differs from terminal. Solutions:\n1. Add GOPATH/bin to your system PATH\n2. Restart your git GUI application\n3. Use terminal for git operations\n4. Check that $(go env GOPATH)/bin is in PATH",
			)
		}

		if strings.Contains(output, "permission denied") {
			return prerrors.NewToolExecutionError(
				"make fumpt",
				output,
				"Permission denied. Check file permissions and ensure you have write access to all Go files.",
			)
		}

		if strings.Contains(output, "syntax error") || strings.Contains(output, "invalid Go syntax") {
			return prerrors.NewToolExecutionError(
				"make fumpt",
				output,
				"Go syntax errors prevent formatting. Fix syntax errors in your Go files before running fumpt.",
			)
		}

		// Enhanced generic failure with better context
		envHints := []string{
			"Run 'make fumpt' manually to see detailed error output.",
			"Check your Makefile and gofumpt installation.",
			"If using a git GUI (Tower, SourceTree, etc.), try using terminal instead.",
			"Ensure PRE_COMMIT_SYSTEM_FUMPT_VERSION is set correctly in .env.shared",
		}

		return prerrors.NewToolExecutionError(
			"make fumpt",
			output,
			strings.Join(envHints, "\n"),
		)
	}

	return nil
}

// runDirectFumpt runs gofumpt directly on files
func (c *FumptCheck) runDirectFumpt(ctx context.Context, files []string) error {
	// Check if gofumpt is available
	if _, err := exec.LookPath("gofumpt"); err != nil {
		return prerrors.NewToolNotFoundError(
			"gofumpt",
			"Install gofumpt: 'go install mvdan.cc/gofumpt@latest'",
		)
	}

	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Build absolute paths
	absFiles := make([]string, len(files))
	for i, file := range files {
		absFiles[i] = filepath.Join(repoRoot, file)
	}

	// Add timeout for gofumpt command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Run gofumpt
	args := append([]string{"-w"}, absFiles...)
	cmd := exec.CommandContext(ctx, "gofumpt", args...) //nolint:gosec // Command arguments are validated
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stdout.String() + stderr.String()

		// Check if it's a context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolExecutionError(
				"gofumpt",
				output,
				fmt.Sprintf("Fumpt timed out after %v. Consider running on fewer files or increasing PRE_COMMIT_SYSTEM_FUMPT_TIMEOUT.", c.timeout),
			)
		}

		if strings.Contains(output, "permission denied") {
			return prerrors.NewToolExecutionError(
				"gofumpt",
				output,
				"Permission denied. Check file permissions and ensure you have write access to all Go files.",
			)
		}

		if strings.Contains(output, "syntax error") || strings.Contains(output, "invalid Go syntax") {
			return prerrors.NewToolExecutionError(
				"gofumpt",
				output,
				"Go syntax errors prevent formatting. Fix syntax errors in your Go files before running fumpt.",
			)
		}

		// Generic failure
		return prerrors.NewToolExecutionError(
			"gofumpt",
			output,
			"Run 'gofumpt -w <files>' manually to see detailed error output.",
		)
	}

	return nil
}
