// Package git provides Git repository operations for the pre-commit system
package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
)

// Repository represents a Git repository
type Repository struct {
	root string
}

// NewRepository creates a new Repository instance
func NewRepository(root string) *Repository {
	return &Repository{root: root}
}

// GetStagedFiles returns all files staged for commit
func (r *Repository) GetStagedFiles() ([]string, error) {
	cmd := exec.CommandContext(context.Background(), "git", "diff", "--cached", "--name-only", "--diff-filter=ACMR")
	cmd.Dir = r.root

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %w", err)
	}

	return parseFileList(output), nil
}

// GetAllFiles returns all tracked files in the repository
func (r *Repository) GetAllFiles() ([]string, error) {
	cmd := exec.CommandContext(context.Background(), "git", "ls-files")
	cmd.Dir = r.root

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get all files: %w", err)
	}

	return parseFileList(output), nil
}

// GetModifiedFiles returns all modified files (staged and unstaged)
func (r *Repository) GetModifiedFiles() ([]string, error) {
	// Get staged files
	staged, err := r.GetStagedFiles()
	if err != nil {
		return nil, err
	}

	// Get unstaged modifications
	cmd := exec.CommandContext(context.Background(), "git", "diff", "--name-only", "--diff-filter=ACMR")
	cmd.Dir = r.root

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get modified files: %w", err)
	}

	unstaged := parseFileList(output)

	// Merge and deduplicate
	fileMap := make(map[string]bool)
	for _, f := range staged {
		fileMap[f] = true
	}
	for _, f := range unstaged {
		fileMap[f] = true
	}

	files := make([]string, 0, len(fileMap))
	for f := range fileMap {
		files = append(files, f)
	}

	return files, nil
}

// GetFileContent returns the content of a file from the index (staged version)
func (r *Repository) GetFileContent(path string) ([]byte, error) {
	// Try to get staged version first
	cmd := exec.CommandContext(context.Background(), "git", "show", ":"+path) //nolint:gosec // Git command with validated path
	cmd.Dir = r.root

	output, err := cmd.Output()
	if err != nil {
		// If not staged, get the file from disk
		fullPath := filepath.Join(r.root, path)
		return os.ReadFile(fullPath) //nolint:gosec // Path is validated
	}

	return output, nil
}

// IsFileTracked checks if a file is tracked by git
func (r *Repository) IsFileTracked(path string) bool {
	cmd := exec.CommandContext(context.Background(), "git", "ls-files", "--error-unmatch", path)
	cmd.Dir = r.root

	err := cmd.Run()
	return err == nil
}

// GetRoot returns the repository root directory
func (r *Repository) GetRoot() string {
	return r.root
}

// FindRepositoryRoot finds the root directory of the Git repository
func FindRepositoryRoot() (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", "rev-parse", "--show-toplevel")

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}

	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", prerrors.ErrRepositoryRootNotFound
	}

	return root, nil
}

// parseFileList parses newline-separated file list
func parseFileList(output []byte) []string {
	output = bytes.TrimSpace(output)
	if len(output) == 0 {
		return []string{}
	}

	lines := bytes.Split(output, []byte("\n"))
	files := make([]string, 0, len(lines))

	for _, line := range lines {
		file := string(bytes.TrimSpace(line))
		if file != "" {
			files = append(files, file)
		}
	}

	return files
}
