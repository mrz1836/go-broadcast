package sync

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
)

// Static errors for module resolution
var (
	ErrNoVersionsAvailable      = errors.New("no versions available")
	ErrNoValidSemanticVersions  = errors.New("no valid semantic versions found")
	ErrVersionNotFound          = errors.New("version not found")
	ErrNoVersionMatches         = errors.New("no version matches constraint")
	ErrCannotResolveWithoutTags = errors.New("cannot resolve version constraint without git tags")
	ErrInvalidSemverConstraint  = errors.New("invalid semver constraint")
)

// ModuleResolver resolves module version constraints to concrete versions
type ModuleResolver struct {
	logger *logrus.Logger
	cache  *ModuleCache
}

// NewModuleResolver creates a new module version resolver
func NewModuleResolver(logger *logrus.Logger, cache *ModuleCache) *ModuleResolver {
	return &ModuleResolver{
		logger: logger,
		cache:  cache,
	}
}

// ResolveVersion resolves a version constraint to a concrete version
// Supports:
// - Exact versions: "v1.2.3"
// - Latest: "latest"
// - Semver constraints: "~1.2", "^1.2", ">=1.2.0", etc.
func (r *ModuleResolver) ResolveVersion(ctx context.Context, repoPath, constraint string, checkTags bool) (string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", repoPath, constraint)
	if cached, found := r.cache.Get(cacheKey); found {
		r.logger.WithFields(logrus.Fields{
			"repo":       repoPath,
			"constraint": constraint,
			"version":    cached,
		}).Debug("Using cached version resolution")
		return cached, nil
	}

	// Get available versions
	var availableVersions []string
	var err error

	if checkTags {
		availableVersions, err = r.fetchGitTags(ctx, repoPath)
		if err != nil {
			return "", fmt.Errorf("failed to fetch git tags: %w", err)
		}
	} else {
		// Without tags, we can't resolve versions
		if constraint != "latest" && !strings.HasPrefix(constraint, "v") {
			return "", ErrCannotResolveWithoutTags
		}
		// For exact versions without tag checking, just return as-is
		if strings.HasPrefix(constraint, "v") {
			r.cache.Set(cacheKey, constraint)
			return constraint, nil
		}
	}

	// Resolve based on constraint type
	var resolved string
	switch constraint {
	case "latest":
		resolved, err = r.resolveLatest(availableVersions)
	case "":
		// Empty constraint means latest
		resolved, err = r.resolveLatest(availableVersions)
	default:
		if strings.HasPrefix(constraint, "v") && !strings.ContainsAny(constraint, "~^<>=*") {
			// Exact version
			resolved, err = r.resolveExact(availableVersions, constraint)
		} else {
			// Semver constraint
			resolved, err = r.resolveSemver(availableVersions, constraint)
		}
	}

	if err != nil {
		return "", err
	}

	// Cache the result
	r.cache.Set(cacheKey, resolved)

	r.logger.WithFields(logrus.Fields{
		"repo":       repoPath,
		"constraint": constraint,
		"resolved":   resolved,
	}).Info("Resolved module version")

	return resolved, nil
}

// fetchGitTags fetches all git tags from a repository
func (r *ModuleResolver) fetchGitTags(ctx context.Context, repoPath string) ([]string, error) {
	// Use git ls-remote to get tags
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--tags", repoPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git ls-remote: %w", err)
	}

	var tags []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Parse tag from ls-remote output
		// Format: <hash>\trefs/tags/<tag>
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}

		tagRef := parts[1]
		if !strings.HasPrefix(tagRef, "refs/tags/") {
			continue
		}

		// Skip annotated tag markers (^{})
		if strings.HasSuffix(tagRef, "^{}") {
			continue
		}

		tag := strings.TrimPrefix(tagRef, "refs/tags/")

		// Only include semver-like tags
		if strings.HasPrefix(tag, "v") {
			tags = append(tags, tag)
		}
	}

	r.logger.WithFields(logrus.Fields{
		"repo":  repoPath,
		"count": len(tags),
	}).Debug("Fetched git tags")

	return tags, nil
}

// resolveLatest finds the latest version from available versions
func (r *ModuleResolver) resolveLatest(versions []string) (string, error) {
	if len(versions) == 0 {
		return "", ErrNoVersionsAvailable
	}

	// Parse all versions
	semverList := make([]*semver.Version, 0, len(versions))
	for _, v := range versions {
		parsed, err := semver.NewVersion(v)
		if err != nil {
			r.logger.WithField("version", v).Debug("Skipping invalid semver")
			continue
		}
		semverList = append(semverList, parsed)
	}

	if len(semverList) == 0 {
		return "", ErrNoValidSemanticVersions
	}

	// Sort by version (highest first)
	sort.Sort(sort.Reverse(semver.Collection(semverList)))

	// Return the highest version
	return "v" + semverList[0].String(), nil
}

// resolveExact checks if an exact version exists
func (r *ModuleResolver) resolveExact(versions []string, target string) (string, error) {
	for _, v := range versions {
		if v == target {
			return v, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrVersionNotFound, target)
}

// resolveSemver resolves a semver constraint to a concrete version
func (r *ModuleResolver) resolveSemver(versions []string, constraint string) (string, error) {
	// Parse the constraint
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return "", fmt.Errorf("%w %s: %w", ErrInvalidSemverConstraint, constraint, err)
	}

	// Find all matching versions
	var matches []*semver.Version
	for _, v := range versions {
		parsed, err := semver.NewVersion(v)
		if err != nil {
			r.logger.WithField("version", v).Debug("Skipping invalid semver")
			continue
		}

		if c.Check(parsed) {
			matches = append(matches, parsed)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("%w: %s", ErrNoVersionMatches, constraint)
	}

	// Sort by version (highest first)
	sort.Sort(sort.Reverse(semver.Collection(matches)))

	// Return the highest matching version
	return "v" + matches[0].String(), nil
}

// GetAvailableVersions returns all available versions for a repository
func (r *ModuleResolver) GetAvailableVersions(ctx context.Context, repoPath string) ([]string, error) {
	// Check cache for available versions
	cacheKey := fmt.Sprintf("%s:_versions", repoPath)
	if cached, found := r.cache.Get(cacheKey); found {
		// Parse cached versions (stored as comma-separated)
		return strings.Split(cached, ","), nil
	}

	versions, err := r.fetchGitTags(ctx, repoPath)
	if err != nil {
		return nil, err
	}

	// Parse and validate versions
	var validVersions []string
	for _, v := range versions {
		if _, err := semver.NewVersion(v); err == nil {
			validVersions = append(validVersions, v)
		}
	}

	// Sort versions
	sort.Slice(validVersions, func(i, j int) bool {
		vi, _ := semver.NewVersion(validVersions[i])
		vj, _ := semver.NewVersion(validVersions[j])
		return vi.GreaterThan(vj)
	})

	// Cache the result
	if len(validVersions) > 0 {
		r.cache.Set(cacheKey, strings.Join(validVersions, ","))
	}

	return validVersions, nil
}

// IsVersionConstraint checks if a string is a version constraint
func (r *ModuleResolver) IsVersionConstraint(s string) bool {
	// Check for exact version
	if strings.HasPrefix(s, "v") && !strings.ContainsAny(s, "~^<>=*") {
		_, err := semver.NewVersion(s)
		return err == nil
	}

	// Check for "latest"
	if s == "latest" || s == "" {
		return true
	}

	// Check for semver constraint
	_, err := semver.NewConstraint(s)
	return err == nil
}
