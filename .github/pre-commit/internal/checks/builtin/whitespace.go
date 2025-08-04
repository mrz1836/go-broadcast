// Package builtin provides built-in pre-commit checks
package builtin

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
)

// WhitespaceCheck removes trailing whitespace from files
type WhitespaceCheck struct {
	timeout   time.Duration
	config    *config.Config
	autoStage bool
}

// NewWhitespaceCheck creates a new whitespace check
func NewWhitespaceCheck() *WhitespaceCheck {
	return &WhitespaceCheck{
		timeout:   30 * time.Second, // Default 30 second timeout
		config:    nil,
		autoStage: false,
	}
}

// NewWhitespaceCheckWithTimeout creates a new whitespace check with custom timeout
func NewWhitespaceCheckWithTimeout(timeout time.Duration) *WhitespaceCheck {
	return &WhitespaceCheck{
		timeout:   timeout,
		config:    nil,
		autoStage: false,
	}
}

// NewWhitespaceCheckWithConfig creates a new whitespace check with full configuration
func NewWhitespaceCheckWithConfig(cfg *config.Config) *WhitespaceCheck {
	timeout := 30 * time.Second
	autoStage := false

	if cfg != nil {
		timeout = time.Duration(cfg.CheckTimeouts.Whitespace) * time.Second
		autoStage = cfg.CheckBehaviors.WhitespaceAutoStage
	}

	return &WhitespaceCheck{
		timeout:   timeout,
		config:    cfg,
		autoStage: autoStage,
	}
}

// Name returns the name of the check
func (c *WhitespaceCheck) Name() string {
	return "whitespace"
}

// Description returns a brief description of the check
func (c *WhitespaceCheck) Description() string {
	return "Fix trailing whitespace"
}

// Metadata returns comprehensive metadata about the check
func (c *WhitespaceCheck) Metadata() interface{} {
	return CheckMetadata{
		Name:              "whitespace",
		Description:       "Remove trailing whitespace from text files",
		FilePatterns:      []string{"*.go", "*.md", "*.txt", "*.yml", "*.yaml", "*.json", "Makefile"},
		EstimatedDuration: 1 * time.Second,
		Dependencies:      []string{}, // No external dependencies
		DefaultTimeout:    c.timeout,
		Category:          "formatting",
		RequiresFiles:     true,
	}
}

// Run executes the whitespace check
func (c *WhitespaceCheck) Run(ctx context.Context, files []string) error {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var errors []string
	var foundIssues bool
	var modifiedFiles []string

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
				modifiedFiles = append(modifiedFiles, file)
			}
		}
	}

	// Stage modified files if auto-staging is enabled
	if c.autoStage && len(modifiedFiles) > 0 {
		if err := c.stageFiles(ctx, modifiedFiles); err != nil {
			// Log warning but don't fail the check
			errors = append(errors, fmt.Sprintf("auto-staging failed: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%w:\n%s", prerrors.ErrWhitespaceIssues, strings.Join(errors, "\n"))
	}

	if foundIssues {
		return prerrors.ErrWhitespaceIssues
	}

	return nil
}

// FilterFiles filters to only text files
func (c *WhitespaceCheck) FilterFiles(files []string) []string {
	var filtered []string
	for _, file := range files {
		if isTextFile(file) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// processFile removes trailing whitespace from a single file
func (c *WhitespaceCheck) processFile(filename string) (bool, error) {
	// Read file
	content, err := os.ReadFile(filename) //nolint:gosec // File from user input
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	// Process lines
	var modified bool
	var output bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimRight(line, " \t")

		if line != trimmed {
			modified = true
		}

		output.WriteString(trimmed)
		output.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("error scanning file: %w", err)
	}

	// Only write if modified
	if modified {
		// Remove trailing newline added by loop
		result := output.Bytes()
		if len(result) > 0 && result[len(result)-1] == '\n' {
			result = result[:len(result)-1]
		}

		// Preserve original file ending
		if len(content) > 0 && content[len(content)-1] == '\n' {
			result = append(result, '\n')
		}

		if err := os.WriteFile(filename, result, 0o600); err != nil {
			return false, fmt.Errorf("failed to write file: %w", err)
		}
	}

	return modified, nil
}

// stageFiles adds modified files to git staging area
func (c *WhitespaceCheck) stageFiles(ctx context.Context, files []string) error {
	if len(files) == 0 {
		return nil
	}

	// Build git add command with all modified files
	args := append([]string{"add"}, files...)
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // git add with controlled file list

	// Set working directory to repository root if possible
	if c.config != nil && c.config.Directory != "" {
		// Go up from pre-commit directory to repository root
		repoRoot := filepath.Dir(filepath.Dir(c.config.Directory))
		cmd.Dir = repoRoot
	}

	// Run git add command
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage files: %w (output: %s)", err, string(output))
	}

	return nil
}

// isTextFile checks if a file is likely a text file based on extension
func isTextFile(filename string) bool {
	// Common text file extensions
	textExtensions := map[string]bool{
		".go":     true,
		".mod":    true,
		".sum":    true,
		".md":     true,
		".txt":    true,
		".yml":    true,
		".yaml":   true,
		".json":   true,
		".xml":    true,
		".toml":   true,
		".ini":    true,
		".cfg":    true,
		".conf":   true,
		".sh":     true,
		".bash":   true,
		".zsh":    true,
		".fish":   true,
		".ps1":    true,
		".py":     true,
		".rb":     true,
		".js":     true,
		".ts":     true,
		".jsx":    true,
		".tsx":    true,
		".css":    true,
		".scss":   true,
		".sass":   true,
		".less":   true,
		".html":   true,
		".htm":    true,
		".vue":    true,
		".java":   true,
		".c":      true,
		".cpp":    true,
		".cc":     true,
		".cxx":    true,
		".h":      true,
		".hpp":    true,
		".rs":     true,
		".swift":  true,
		".kt":     true,
		".scala":  true,
		".r":      true,
		".R":      true,
		".sql":    true,
		".proto":  true,
		".thrift": true,
		".env":    true,
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if textExtensions[ext] {
		return true
	}

	// Check for files without extensions that are commonly text
	base := filepath.Base(filename)
	textFiles := map[string]bool{
		"Makefile":      true,
		"Dockerfile":    true,
		"Jenkinsfile":   true,
		"Vagrantfile":   true,
		".gitignore":    true,
		".dockerignore": true,
		".editorconfig": true,
		"LICENSE":       true,
		"README":        true,
		"CHANGELOG":     true,
		"AUTHORS":       true,
		"CONTRIBUTORS":  true,
		"MAINTAINERS":   true,
		"TODO":          true,
		"NOTES":         true,
	}

	return textFiles[base]
}
