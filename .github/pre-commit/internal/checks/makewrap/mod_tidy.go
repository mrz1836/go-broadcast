package makewrap

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
	"github.com/mrz1836/go-broadcast/pre-commit/internal/shared"
)

// ModTidyCheck ensures go.mod and go.sum are tidy
type ModTidyCheck struct {
	sharedCtx *shared.Context
	timeout   time.Duration
}

// NewModTidyCheck creates a new mod tidy check
func NewModTidyCheck() *ModTidyCheck {
	return &ModTidyCheck{
		sharedCtx: shared.NewContext(),
		timeout:   30 * time.Second, // 30 second timeout for mod tidy
	}
}

// NewModTidyCheckWithSharedContext creates a new mod tidy check with shared context
func NewModTidyCheckWithSharedContext(sharedCtx *shared.Context) *ModTidyCheck {
	return &ModTidyCheck{
		sharedCtx: sharedCtx,
		timeout:   30 * time.Second,
	}
}

// NewModTidyCheckWithConfig creates a new mod tidy check with shared context and custom timeout
func NewModTidyCheckWithConfig(sharedCtx *shared.Context, timeout time.Duration) *ModTidyCheck {
	return &ModTidyCheck{
		sharedCtx: sharedCtx,
		timeout:   timeout,
	}
}

// Name returns the name of the check
func (c *ModTidyCheck) Name() string {
	return "mod-tidy"
}

// Description returns a brief description of the check
func (c *ModTidyCheck) Description() string {
	return "Ensure go.mod and go.sum are tidy"
}

// Metadata returns comprehensive metadata about the check
func (c *ModTidyCheck) Metadata() interface{} {
	return CheckMetadata{
		Name:              "mod-tidy",
		Description:       "Ensure go.mod and go.sum are up to date and tidy",
		FilePatterns:      []string{"*.go", "go.mod", "go.sum"},
		EstimatedDuration: 5 * time.Second,
		Dependencies:      []string{"mod-tidy"}, // make target
		DefaultTimeout:    c.timeout,
		Category:          "dependencies",
		RequiresFiles:     false, // Can run even with no staged files
	}
}

// Run executes the mod tidy check
func (c *ModTidyCheck) Run(ctx context.Context, files []string) error {
	// Early return if no files to process
	if len(files) == 0 {
		return nil
	}

	// Check if make mod-tidy is available
	if c.sharedCtx.HasMakeTarget(ctx, "mod-tidy") {
		return c.runMakeModTidy(ctx)
	}

	// Fall back to direct go mod tidy
	return c.runDirectModTidy(ctx)
}

// FilterFiles filters to only go.mod and go.sum files or when .go files change
func (c *ModTidyCheck) FilterFiles(files []string) []string {
	var hasGoMod, hasGoFiles bool
	var filtered []string

	for _, file := range files {
		// Check for go.mod/go.sum changes
		if file == "go.mod" || file == "go.sum" || strings.HasSuffix(file, "/go.mod") || strings.HasSuffix(file, "/go.sum") {
			hasGoMod = true
			filtered = append(filtered, file)
		}
		// Check for .go file changes
		if strings.HasSuffix(file, ".go") {
			hasGoFiles = true
		}
	}

	// If we have go.mod/go.sum changes, run on those
	if hasGoMod {
		return filtered
	}

	// If we have .go files but no go.mod/go.sum changes, still run mod-tidy
	// because imports might have changed
	if hasGoFiles {
		return []string{"go.mod"} // Dummy entry to trigger the check
	}

	// No relevant files
	return []string{}
}

// runMakeModTidy runs make mod-tidy
func (c *ModTidyCheck) runMakeModTidy(ctx context.Context) error {
	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Add timeout for make command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "make", "mod-tidy")
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stdout.String() + stderr.String()

		// Check if it's a context timeout
		if ctx.Err() == context.DeadlineExceeded {
			return prerrors.NewToolExecutionError(
				"make mod-tidy",
				output,
				fmt.Sprintf("Mod tidy check timed out after %v. Consider increasing PRE_COMMIT_SYSTEM_MOD_TIDY_TIMEOUT or run 'make mod-tidy' manually.", c.timeout),
			)
		}

		// Parse the error for better context
		if strings.Contains(output, "No rule to make target") {
			return prerrors.NewMakeTargetNotFoundError(
				"mod-tidy",
				"Create a 'mod-tidy' target in your Makefile or disable mod-tidy with PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=false",
			)
		}

		if strings.Contains(output, "no go.mod file") {
			return prerrors.NewToolExecutionError(
				"make mod-tidy",
				output,
				"No go.mod file found. Initialize a Go module with 'go mod init <module-name>'.",
			)
		}

		if strings.Contains(output, "network") || strings.Contains(output, "timeout") {
			return prerrors.NewToolExecutionError(
				"make mod-tidy",
				output,
				"Network error downloading modules. Check your internet connection and proxy settings. Try running 'go mod tidy' manually.",
			)
		}

		if strings.Contains(output, "checksum mismatch") {
			return prerrors.NewToolExecutionError(
				"make mod-tidy",
				output,
				"Module checksum verification failed. Run 'go clean -modcache' and try again, or check for module security issues.",
			)
		}

		// Generic failure
		return prerrors.NewToolExecutionError(
			"make mod-tidy",
			output,
			"Run 'make mod-tidy' manually to see detailed error output. Check your Makefile and module dependencies.",
		)
	}

	// Check if there are uncommitted changes
	return c.checkUncommittedChanges(ctx, repoRoot)
}

// runDirectModTidy runs go mod tidy directly
func (c *ModTidyCheck) runDirectModTidy(ctx context.Context) error {
	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Add timeout for go mod tidy command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stdout.String() + stderr.String()

		// Check if it's a context timeout
		if ctx.Err() == context.DeadlineExceeded {
			return prerrors.NewToolExecutionError(
				"go mod tidy",
				output,
				fmt.Sprintf("Mod tidy timed out after %v. Consider increasing PRE_COMMIT_SYSTEM_MOD_TIDY_TIMEOUT.", c.timeout),
			)
		}

		if strings.Contains(output, "no go.mod file") {
			return prerrors.NewToolExecutionError(
				"go mod tidy",
				output,
				"No go.mod file found. Initialize a Go module with 'go mod init <module-name>'.",
			)
		}

		if strings.Contains(output, "network") || strings.Contains(output, "timeout") {
			return prerrors.NewToolExecutionError(
				"go mod tidy",
				output,
				"Network error downloading modules. Check your internet connection and proxy settings.",
			)
		}

		if strings.Contains(output, "checksum mismatch") {
			return prerrors.NewToolExecutionError(
				"go mod tidy",
				output,
				"Module checksum verification failed. Run 'go clean -modcache' and try again.",
			)
		}

		if strings.Contains(output, "not found") {
			return prerrors.NewToolExecutionError(
				"go mod tidy",
				output,
				"Module dependencies not found. Check that all imported modules exist and are accessible.",
			)
		}

		// Generic failure
		return prerrors.NewToolExecutionError(
			"go mod tidy",
			output,
			"Run 'go mod tidy' manually to see detailed error output.",
		)
	}

	// Check if there are uncommitted changes
	return c.checkUncommittedChanges(ctx, repoRoot)
}

// checkUncommittedChanges checks if go mod tidy made any changes
func (c *ModTidyCheck) checkUncommittedChanges(ctx context.Context, repoRoot string) error {
	// Add short timeout for git diff
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Check for new untracked files (like go.sum created for the first time)
	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "go.mod", "go.sum")
	statusCmd.Dir = repoRoot

	var statusOutput bytes.Buffer
	statusCmd.Stdout = &statusOutput
	statusCmd.Stderr = &statusOutput

	if err := statusCmd.Run(); err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	// If there are any changes or new files, that's an error
	if statusOutput.Len() > 0 {
		return prerrors.NewToolExecutionError(
			"git status",
			statusOutput.String(),
			"go.mod or go.sum were modified by 'go mod tidy'. Commit these changes to proceed. Run 'git add go.mod go.sum && git commit -m \"Update module dependencies\"'.",
		)
	}

	return nil
}
