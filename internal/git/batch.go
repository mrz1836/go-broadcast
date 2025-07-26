package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// BatchClient extends the basic Client interface with batch operations
type BatchClient interface {
	Client
	BatchAddFiles(ctx context.Context, repoPath string, files []string) error
	BatchStatus(ctx context.Context, repoPath string, files []string) (map[string]string, error)
}

// Ensure gitClient implements BatchClient
var _ BatchClient = (*gitClient)(nil)

// BatchAddFiles adds multiple files in optimized batches to avoid command line length limits
func (g *gitClient) BatchAddFiles(ctx context.Context, repoPath string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	// Batch files to avoid command line length limits
	const maxBatchSize = 100

	for i := 0; i < len(files); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		args := []string{"-C", repoPath, "add"}
		args = append(args, batch...)

		cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

		if err := g.runCommand(cmd); err != nil {
			return fmt.Errorf("batch add failed for files %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// BatchStatus gets status for multiple files efficiently using a single git command
func (g *gitClient) BatchStatus(ctx context.Context, repoPath string, files []string) (map[string]string, error) {
	if len(files) == 0 {
		return make(map[string]string), nil
	}

	args := []string{"-C", repoPath, "status", "--porcelain", "--"}
	args = append(args, files...)

	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("batch status failed: %w", err)
	}

	statuses := make(map[string]string)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		status := line[:2]
		file := strings.TrimSpace(line[3:])
		statuses[file] = status
	}

	return statuses, nil
}

// BatchStatusAll gets status for all files in the repository
func (g *gitClient) BatchStatusAll(ctx context.Context, repoPath string) (map[string]string, error) {
	args := []string{"-C", repoPath, "status", "--porcelain"}

	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("batch status all failed: %w", err)
	}

	statuses := make(map[string]string)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		status := line[:2]
		file := strings.TrimSpace(line[3:])
		statuses[file] = status
	}

	return statuses, nil
}

// BatchDiffFiles gets diff for multiple files efficiently
func (g *gitClient) BatchDiffFiles(ctx context.Context, repoPath string, files []string, staged bool) (map[string]string, error) {
	if len(files) == 0 {
		return make(map[string]string), nil
	}

	diffs := make(map[string]string)

	// Process files in smaller batches to avoid command line length limits
	const maxBatchSize = 50

	for i := 0; i < len(files); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		args := []string{"-C", repoPath, "diff", "--name-only"}
		if staged {
			args = append(args, "--staged")
		}
		args = append(args, "--")
		args = append(args, batch...)

		cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("batch diff failed for files %d-%d: %w", i, end-1, err)
		}

		// Get individual diffs for files that have changes
		changedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, file := range changedFiles {
			if file == "" {
				continue
			}

			// Get the actual diff for this file
			diffArgs := []string{"-C", repoPath, "diff"}
			if staged {
				diffArgs = append(diffArgs, "--staged")
			}
			diffArgs = append(diffArgs, "--", file)

			diffCmd := exec.CommandContext(ctx, "git", diffArgs...) //nolint:gosec // Arguments are safely constructed
			diffOutput, err := diffCmd.Output()
			if err != nil {
				return nil, fmt.Errorf("failed to get diff for file %s: %w", file, err)
			}

			diffs[file] = string(diffOutput)
		}
	}

	return diffs, nil
}

// BatchCheckIgnored checks if multiple files are ignored by git
func (g *gitClient) BatchCheckIgnored(ctx context.Context, repoPath string, files []string) (map[string]bool, error) {
	if len(files) == 0 {
		return make(map[string]bool), nil
	}

	args := []string{"-C", repoPath, "check-ignore"}
	args = append(args, files...)

	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

	output, err := cmd.Output()
	// git check-ignore returns exit code 1 when files are not ignored, which is not an error
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			// Files are not ignored - this is expected, continue processing
			err = nil
		}
		if err != nil {
			return nil, fmt.Errorf("batch check-ignore failed: %w", err)
		}
	}

	ignored := make(map[string]bool)

	// Initialize all files as not ignored
	for _, file := range files {
		ignored[file] = false
	}

	// Mark ignored files as true
	if string(output) != "" {
		ignoredFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, file := range ignoredFiles {
			if file != "" {
				ignored[file] = true
			}
		}
	}

	return ignored, nil
}

// BatchRemoveFiles removes multiple files from git tracking efficiently
func (g *gitClient) BatchRemoveFiles(ctx context.Context, repoPath string, files []string, keepLocal bool) error {
	if len(files) == 0 {
		return nil
	}

	// Batch files to avoid command line length limits
	const maxBatchSize = 100

	for i := 0; i < len(files); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		args := []string{"-C", repoPath, "rm"}
		if keepLocal {
			args = append(args, "--cached")
		}
		args = append(args, batch...)

		cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Arguments are safely constructed

		if err := g.runCommand(cmd); err != nil {
			return fmt.Errorf("batch remove failed for files %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}
