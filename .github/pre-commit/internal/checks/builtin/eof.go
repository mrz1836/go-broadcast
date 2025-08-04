package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
)

// EOFCheck ensures files end with a newline
type EOFCheck struct {
	timeout time.Duration
}

// NewEOFCheck creates a new EOF check
func NewEOFCheck() *EOFCheck {
	return &EOFCheck{
		timeout: 30 * time.Second, // Default 30 second timeout
	}
}

// NewEOFCheckWithTimeout creates a new EOF check with custom timeout
func NewEOFCheckWithTimeout(timeout time.Duration) *EOFCheck {
	return &EOFCheck{
		timeout: timeout,
	}
}

// Name returns the name of the check
func (c *EOFCheck) Name() string {
	return "eof"
}

// Description returns a brief description of the check
func (c *EOFCheck) Description() string {
	return "Ensure files end with newline"
}

// Metadata returns comprehensive metadata about the check
func (c *EOFCheck) Metadata() interface{} {
	return CheckMetadata{
		Name:              "eof",
		Description:       "Ensure text files end with a newline character",
		FilePatterns:      []string{"*.go", "*.md", "*.txt", "*.yml", "*.yaml", "*.json", "Makefile"},
		EstimatedDuration: 1 * time.Second,
		Dependencies:      []string{}, // No external dependencies
		DefaultTimeout:    c.timeout,
		Category:          "formatting",
		RequiresFiles:     true,
	}
}

// Run executes the EOF check
func (c *EOFCheck) Run(ctx context.Context, files []string) error {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var errors []string
	var foundIssues bool

	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			modified, err := c.processFile(file)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", file, err))
			} else if modified {
				foundIssues = true
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%w:\n%s", prerrors.ErrEOFIssues, strings.Join(errors, "\n"))
	}

	if foundIssues {
		return prerrors.ErrEOFIssues
	}

	return nil
}

// FilterFiles filters to only text files
func (c *EOFCheck) FilterFiles(files []string) []string {
	var filtered []string
	for _, file := range files {
		if isTextFile(file) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// processFile ensures a file ends with a newline
func (c *EOFCheck) processFile(filename string) (bool, error) {
	// Read file
	content, err := os.ReadFile(filename) //nolint:gosec // File from user input
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	// Skip empty files
	if len(content) == 0 {
		return false, nil
	}

	// Check if file ends with newline
	if content[len(content)-1] != '\n' {
		// Add newline
		content = append(content, '\n')

		if err := os.WriteFile(filename, content, 0o600); err != nil {
			return false, fmt.Errorf("failed to write file: %w", err)
		}
		return true, nil
	}

	return false, nil
}
