package makewrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
)

// ModTidyCheck ensures go.mod and go.sum are tidy
type ModTidyCheck struct {
	repoRoot string
}

// NewModTidyCheck creates a new mod tidy check
func NewModTidyCheck() *ModTidyCheck {
	return &ModTidyCheck{}
}

// Name returns the name of the check
func (c *ModTidyCheck) Name() string {
	return "mod-tidy"
}

// Description returns a brief description of the check
func (c *ModTidyCheck) Description() string {
	return "Ensure go.mod and go.sum are tidy"
}

// Run executes the mod tidy check
func (c *ModTidyCheck) Run(ctx context.Context, _ []string) error {
	// Find repository root
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}
	c.repoRoot = strings.TrimSpace(string(output))

	// Check if make mod-tidy is available
	if c.hasMakeTarget(ctx, "mod-tidy") {
		return c.runMakeModTidy(ctx)
	}

	// Fall back to direct go mod tidy
	return c.runDirectModTidy(ctx)
}

// FilterFiles filters to only go.mod and go.sum files
func (c *ModTidyCheck) FilterFiles(files []string) []string {
	var filtered []string
	for _, file := range files {
		if file == "go.mod" || file == "go.sum" || strings.HasSuffix(file, "/go.mod") || strings.HasSuffix(file, "/go.sum") {
			filtered = append(filtered, file)
		}
	}
	// If no go.mod/go.sum in the changeset, but we have .go files, still run
	if len(filtered) == 0 {
		for _, file := range files {
			if strings.HasSuffix(file, ".go") {
				// Return a dummy entry to trigger the check
				return []string{"go.mod"}
			}
		}
	}
	return filtered
}

// hasMakeTarget checks if a make target exists
func (c *ModTidyCheck) hasMakeTarget(ctx context.Context, target string) bool {
	cmd := exec.CommandContext(ctx, "make", "-n", target)
	cmd.Dir = c.repoRoot
	return cmd.Run() == nil
}

// runMakeModTidy runs make mod-tidy
func (c *ModTidyCheck) runMakeModTidy(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "make", "mod-tidy")
	cmd.Dir = c.repoRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make mod-tidy failed: %w\n%s", err, stderr.String())
	}

	// Check if there are uncommitted changes
	return c.checkUncommittedChanges(ctx)
}

// runDirectModTidy runs go mod tidy directly
func (c *ModTidyCheck) runDirectModTidy(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = c.repoRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w\n%s", err, stderr.String())
	}

	// Check if there are uncommitted changes
	return c.checkUncommittedChanges(ctx)
}

// checkUncommittedChanges checks if go mod tidy made any changes
func (c *ModTidyCheck) checkUncommittedChanges(ctx context.Context) error {
	// Check if go.mod or go.sum have changes
	cmd := exec.CommandContext(ctx, "git", "diff", "--exit-code", "go.mod", "go.sum")
	cmd.Dir = c.repoRoot

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		// Exit code 1 means there are differences
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return prerrors.ErrNotTidy
		}
		return fmt.Errorf("failed to check for changes: %w", err)
	}

	return nil
}
