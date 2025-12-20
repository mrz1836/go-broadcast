package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/mrz1836/go-broadcast/internal/git"
)

// Static errors for module source resolution
var (
	ErrEmptyRepoURL  = errors.New("repository URL cannot be empty")
	ErrEmptyVersion  = errors.New("version cannot be empty")
	ErrCloneFailed   = errors.New("failed to clone repository at tag")
	ErrSubdirMissing = errors.New("subdirectory does not exist in cloned repository")
)

// ModuleSourceResolver handles fetching source files at specific versions
type ModuleSourceResolver struct {
	git    git.Client
	logger *logrus.Logger
	cache  *ModuleCache
}

// NewModuleSourceResolver creates a new module source resolver
func NewModuleSourceResolver(gitClient git.Client, logger *logrus.Logger, cache *ModuleCache) *ModuleSourceResolver {
	return &ModuleSourceResolver{
		git:    gitClient,
		logger: logger,
		cache:  cache,
	}
}

// VersionedSource represents source files at a specific version
type VersionedSource struct {
	// Path is the local path to the versioned source (including subdir if specified)
	Path string

	// RepoPath is the path to the cloned repository root
	RepoPath string

	// ResolvedVersion is the actual version that was cloned
	ResolvedVersion string

	// CleanupFunc should be called after the source is no longer needed
	// to remove temporary directories
	CleanupFunc func()
}

// GetSourceAtVersion fetches source files from a repository at a specific tag/version.
// It clones the repository at the specified tag to a temporary location and returns
// the path to the source files.
//
// Parameters:
//   - ctx: Context for cancellation
//   - repoURL: Full URL to the repository (e.g., "https://github.com/owner/repo")
//   - version: The git tag to clone (e.g., "v1.2.3")
//   - subdir: Optional subdirectory within the repository to return
//   - tempDir: Base directory for temporary clones
//
// Returns:
//   - VersionedSource with path to the cloned source and cleanup function
//   - Error if cloning fails
func (r *ModuleSourceResolver) GetSourceAtVersion(
	ctx context.Context,
	repoURL string,
	version string,
	subdir string,
	tempDir string,
) (*VersionedSource, error) {
	if repoURL == "" {
		return nil, ErrEmptyRepoURL
	}
	if version == "" {
		return nil, ErrEmptyVersion
	}

	logger := r.logger.WithFields(logrus.Fields{
		"repo":    repoURL,
		"version": version,
		"subdir":  subdir,
	})

	// Create a unique directory for this version
	// Use a sanitized version string to create a valid directory name
	versionDir := filepath.Join(tempDir, "module-"+sanitizeVersion(version)+"-"+generateShortID())

	logger.WithField("clone_path", versionDir).Debug("Cloning repository at specific version")

	// Clone at the specific tag
	err := r.git.CloneAtTag(ctx, repoURL, versionDir, version, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCloneFailed, err)
	}

	// Determine the source path (with optional subdirectory)
	sourcePath := versionDir
	if subdir != "" {
		sourcePath = filepath.Join(versionDir, subdir)
		// Verify the subdirectory exists
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			// Clean up the clone since we can't use it
			_ = os.RemoveAll(versionDir)
			return nil, fmt.Errorf("%w: %s", ErrSubdirMissing, subdir)
		}
	}

	logger.WithField("source_path", sourcePath).Info("Successfully cloned repository at version")

	// Return the versioned source with cleanup function
	return &VersionedSource{
		Path:            sourcePath,
		RepoPath:        versionDir,
		ResolvedVersion: version,
		CleanupFunc: func() {
			if err := os.RemoveAll(versionDir); err != nil {
				r.logger.WithError(err).WithField("path", versionDir).Warn("Failed to cleanup versioned source")
			} else {
				r.logger.WithField("path", versionDir).Debug("Cleaned up versioned source")
			}
		},
	}, nil
}

// sanitizeVersion converts a version tag to a safe directory name
func sanitizeVersion(version string) string {
	// Replace problematic characters with hyphens
	safe := strings.ReplaceAll(version, "/", "-")
	safe = strings.ReplaceAll(safe, "\\", "-")
	safe = strings.ReplaceAll(safe, ":", "-")
	safe = strings.ReplaceAll(safe, "*", "-")
	safe = strings.ReplaceAll(safe, "?", "-")
	safe = strings.ReplaceAll(safe, "\"", "-")
	safe = strings.ReplaceAll(safe, "<", "-")
	safe = strings.ReplaceAll(safe, ">", "-")
	safe = strings.ReplaceAll(safe, "|", "-")
	return safe
}

// generateShortID generates a short unique ID for directory naming
func generateShortID() string {
	// Use current nanosecond time to create a simple unique suffix
	// This avoids collisions when cloning the same version multiple times
	return fmt.Sprintf("%d", os.Getpid())
}
