package makewrap

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
)

// LintCheck runs golangci-lint via make
type LintCheck struct {
	repoRoot string
}

// NewLintCheck creates a new lint check
func NewLintCheck() *LintCheck {
	return &LintCheck{}
}

// Name returns the name of the check
func (c *LintCheck) Name() string {
	return "lint"
}

// Description returns a brief description of the check
func (c *LintCheck) Description() string {
	return "Run golangci-lint"
}

// Run executes the lint check
func (c *LintCheck) Run(ctx context.Context, files []string) error {
	// Find repository root
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}
	c.repoRoot = strings.TrimSpace(string(output))

	// Check if make lint is available
	if !c.hasMakeTarget(ctx, "lint") {
		// Fall back to direct golangci-lint if available
		return c.runDirectLint(ctx, files)
	}

	// Run make lint
	cmd = exec.CommandContext(ctx, "make", "lint")
	cmd.Dir = c.repoRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make lint failed: %w\n%s", err, stderr.String())
	}

	return nil
}

// FilterFiles filters to only Go files
func (c *LintCheck) FilterFiles(files []string) []string {
	var filtered []string
	for _, file := range files {
		if strings.HasSuffix(file, ".go") {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// hasMakeTarget checks if a make target exists
func (c *LintCheck) hasMakeTarget(ctx context.Context, target string) bool {
	cmd := exec.CommandContext(ctx, "make", "-n", target)
	cmd.Dir = c.repoRoot
	return cmd.Run() == nil
}

// runDirectLint runs golangci-lint directly on files
func (c *LintCheck) runDirectLint(ctx context.Context, files []string) error {
	// Check if golangci-lint is available
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		return fmt.Errorf("%w: golangci-lint not found in PATH and make lint not available", prerrors.ErrToolNotFound)
	}

	// Build absolute paths
	absFiles := make([]string, len(files))
	for i, file := range files {
		absFiles[i] = filepath.Join(c.repoRoot, file)
	}

	// Run golangci-lint
	args := append([]string{"run", "--new-from-rev=HEAD~1"}, absFiles...)
	cmd := exec.CommandContext(ctx, "golangci-lint", args...) //nolint:gosec // Command arguments are validated
	cmd.Dir = c.repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// golangci-lint returns non-zero exit code when it finds issues
		// Check if it's actual failure or just linting issues
		output := stdout.String() + stderr.String()
		if strings.Contains(output, "error") || strings.Contains(output, "failed") {
			return fmt.Errorf("golangci-lint failed: %w\n%s", err, output)
		}
		// Otherwise, it's just linting issues
		return fmt.Errorf("%w:\n%s", prerrors.ErrLintingIssues, output)
	}

	return nil
}
