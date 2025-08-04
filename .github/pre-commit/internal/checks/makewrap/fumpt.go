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

		if strings.Contains(output, "gofumpt") && strings.Contains(output, "not found") {
			return prerrors.NewToolNotFoundError(
				"gofumpt",
				"Install gofumpt: 'go install mvdan.cc/gofumpt@latest' or add an install target to your Makefile",
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

		// Generic failure
		return prerrors.NewToolExecutionError(
			"make fumpt",
			output,
			"Run 'make fumpt' manually to see detailed error output. Check your Makefile and gofumpt installation.",
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
