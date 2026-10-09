package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
	"github.com/mrz1836/go-broadcast/internal/logging"
)

// Sparse clone errors
var (
	// ErrSparseCheckout indicates the repository was cloned but its sparse
	// checkout could not be set up. The clone directory has been removed.
	ErrSparseCheckout = errors.New("sparse checkout setup failed")

	// ErrInvalidSparsePath indicates a path cannot be used as a sparse checkout path
	ErrInvalidSparsePath = errors.New("invalid sparse checkout path")
)

// sparsePatternSpecialChars are the characters escaped with a backslash when a
// file path is turned into a sparse-checkout pattern. Non-cone patterns use
// gitignore syntax, where a backslash makes any following character literal.
const sparsePatternSpecialChars = `\*?[]!# `

// cloneFilterArgs returns the `git clone` arguments that control which objects
// are downloaded and whether a checkout happens.
//
// Sparse clones skip the initial checkout (--no-checkout) so that
// configureSparseCheckout can populate only the requested paths; otherwise git
// would fetch every blob of the branch tip to fill the working tree.
func cloneFilterArgs(opts *CloneOptions) []string {
	switch {
	case opts == nil:
		return nil
	case len(opts.SparsePaths) > 0:
		return []string{"--depth", "1", "--filter=blob:none", "--no-checkout", "--single-branch", "--no-tags"}
	case opts.BlobSizeLimit != "" && opts.BlobSizeLimit != "0":
		return []string{"--filter=blob:limit=" + opts.BlobSizeLimit}
	default:
		return nil
	}
}

// sparsePatternsFor returns the sparse-checkout patterns for opts, or an empty
// string when opts does not request a sparse clone.
func sparsePatternsFor(opts *CloneOptions) (string, error) {
	if opts == nil || len(opts.SparsePaths) == 0 {
		return "", nil
	}
	return sparseCheckoutPatterns(opts.SparsePaths)
}

// sparseCheckoutPatterns converts repository-relative file paths into
// newline-separated non-cone sparse-checkout patterns. Each path is anchored to
// the repository root and has gitignore metacharacters escaped, so it matches
// exactly that one path. Duplicate paths are collapsed.
func sparseCheckoutPatterns(paths []string) (string, error) {
	seen := make(map[string]struct{}, len(paths))
	var b strings.Builder

	for _, p := range paths {
		clean, err := cleanSparsePath(p)
		if err != nil {
			return "", err
		}
		if _, dup := seen[clean]; dup {
			continue
		}
		seen[clean] = struct{}{}

		b.WriteByte('/')
		for _, r := range clean {
			if strings.ContainsRune(sparsePatternSpecialChars, r) {
				b.WriteByte('\\')
			}
			b.WriteRune(r)
		}
		b.WriteByte('\n')
	}

	return b.String(), nil
}

// cleanSparsePath normalizes a repository-relative path the same way
// filepath.Join(repoRoot, p) resolves it, so the pattern matches the file the
// caller actually writes. Leading slashes are stripped. Paths that are empty,
// resolve to the repository root, escape the repository, or contain line breaks
// (which a pattern file cannot express) are rejected.
func cleanSparsePath(p string) (string, error) {
	if strings.ContainsAny(p, "\n\r") {
		return "", fmt.Errorf("%w: %q contains a line break", ErrInvalidSparsePath, p)
	}

	clean := strings.TrimLeft(path.Clean(filepath.ToSlash(p)), "/")
	if clean == "" || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("%w: %q", ErrInvalidSparsePath, p)
	}

	return clean, nil
}

// configureSparseCheckout restricts the working tree of a freshly created
// --no-checkout clone to patterns and then populates the index and working tree
// from HEAD, fetching only the blobs of matching paths in a single batch.
//
// It finally verifies that the index matches HEAD. An unpopulated index would
// make a later `git add` stage the deletion of every file in the repository, so
// this invariant is checked before the clone is handed back to the caller.
//
// On failure the clone directory is removed and the error wraps ErrSparseCheckout.
func (g *gitClient) configureSparseCheckout(ctx context.Context, repoPath, patterns string) error {
	steps := []struct {
		args  []string
		stdin string
		desc  string
	}{
		{args: []string{"sparse-checkout", "set", "--no-cone", "--stdin"}, stdin: patterns, desc: "set sparse checkout paths"},
		{args: []string{"read-tree", "-mu", "HEAD"}, desc: "populate sparse working tree"},
		{args: []string{"diff-index", "--cached", "--quiet", "HEAD"}, desc: "verify sparse index matches HEAD"},
	}

	for _, step := range steps {
		args := append([]string{"-C", repoPath}, step.args...)
		cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // args are fixed git subcommands; repoPath is the caller's clone destination
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		cmd.Stdin = strings.NewReader(step.stdin)

		if err := g.runCommand(cmd); err != nil {
			if cleanupErr := os.RemoveAll(repoPath); cleanupErr != nil {
				logging.WithStandardFields(g.logger, g.logConfig, logging.ComponentNames.Git).
					WithError(cleanupErr).Debug("Failed to clean up clone after sparse checkout failure")
			}
			return fmt.Errorf("%w: %w", ErrSparseCheckout, appErrors.WrapWithContext(err, step.desc))
		}
	}

	return nil
}
