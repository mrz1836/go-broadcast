// Package makewrap provides pre-commit checks that wrap make commands
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

// FumptCheck runs gofumpt via make
type FumptCheck struct {
	repoRoot string
}

// NewFumptCheck creates a new fumpt check
func NewFumptCheck() *FumptCheck {
	return &FumptCheck{}
}

// Name returns the name of the check
func (c *FumptCheck) Name() string {
	return "fumpt"
}

// Description returns a brief description of the check
func (c *FumptCheck) Description() string {
	return "Format code with gofumpt"
}

// Run executes the fumpt check
func (c *FumptCheck) Run(ctx context.Context, files []string) error {
	// Find repository root
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}
	c.repoRoot = strings.TrimSpace(string(output))

	// Check if make fumpt is available
	if !c.hasMakeTarget(ctx, "fumpt") {
		// Fall back to direct gofumpt if available
		return c.runDirectFumpt(ctx, files)
	}

	// Run make fumpt
	cmd = exec.CommandContext(ctx, "make", "fumpt")
	cmd.Dir = c.repoRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make fumpt failed: %w\n%s", err, stderr.String())
	}

	return nil
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

// hasMakeTarget checks if a make target exists
func (c *FumptCheck) hasMakeTarget(ctx context.Context, target string) bool {
	cmd := exec.CommandContext(ctx, "make", "-n", target)
	cmd.Dir = c.repoRoot
	return cmd.Run() == nil
}

// runDirectFumpt runs gofumpt directly on files
func (c *FumptCheck) runDirectFumpt(ctx context.Context, files []string) error {
	// Check if gofumpt is available
	if _, err := exec.LookPath("gofumpt"); err != nil {
		return fmt.Errorf("%w: gofumpt not found in PATH and make fumpt not available", prerrors.ErrToolNotFound)
	}

	// Build absolute paths
	absFiles := make([]string, len(files))
	for i, file := range files {
		absFiles[i] = filepath.Join(c.repoRoot, file)
	}

	// Run gofumpt
	args := append([]string{"-w"}, absFiles...)
	cmd := exec.CommandContext(ctx, "gofumpt", args...) //nolint:gosec // Command arguments are validated
	cmd.Dir = c.repoRoot

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gofumpt failed: %w\n%s", err, stderr.String())
	}

	return nil
}
