package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// repoRegex validates org/repo format
	repoRegex = regexp.MustCompile(`^[a-zA-Z0-9][\w.-]*/[a-zA-Z0-9][\w.-]*$`)

	// branchRegex validates branch names
	branchRegex = regexp.MustCompile(`^[a-zA-Z0-9][\w./\-]*$`)

	// ErrUnsupportedVersion indicates the configuration version is not supported
	ErrUnsupportedVersion = errors.New("unsupported config version")
	// ErrNoTargets indicates no target repositories were specified
	ErrNoTargets = errors.New("at least one target repository must be specified")
	// ErrDuplicateTarget indicates a target repository is specified multiple times
	ErrDuplicateTarget = errors.New("duplicate target repository")
	// ErrSourceRepoRequired indicates the source repository is missing
	ErrSourceRepoRequired = errors.New("source repository is required")
	// ErrInvalidRepoFormat indicates a repository name is not in org/repo format
	ErrInvalidRepoFormat = errors.New("invalid repository format (expected: org/repo)")
	// ErrSourceBranchRequired indicates the source branch is missing
	ErrSourceBranchRequired = errors.New("source branch is required")
	// ErrInvalidBranchName indicates a branch name contains invalid characters
	ErrInvalidBranchName = errors.New("invalid branch name")
	// ErrInvalidBranchPrefix indicates the branch prefix contains invalid characters
	ErrInvalidBranchPrefix = errors.New("invalid branch prefix")
	// ErrEmptyPRLabel indicates a PR label is empty or whitespace only
	ErrEmptyPRLabel = errors.New("PR label cannot be empty")
	// ErrRepoRequired indicates a target repository is missing
	ErrRepoRequired = errors.New("repository is required")
	// ErrNoFileMappings indicates a target has no file mappings
	ErrNoFileMappings = errors.New("at least one file mapping is required")
	// ErrDuplicateDestination indicates multiple files map to the same destination
	ErrDuplicateDestination = errors.New("duplicate destination file")
	// ErrSourcePathRequired indicates a file mapping has no source path
	ErrSourcePathRequired = errors.New("source file path is required")
	// ErrDestPathRequired indicates a file mapping has no destination path
	ErrDestPathRequired = errors.New("destination file path is required")
	// ErrInvalidSourcePath indicates a source path is absolute or escapes the repository
	ErrInvalidSourcePath = errors.New("invalid source path (must be relative and within repository)")
	// ErrInvalidDestPath indicates a destination path is absolute or escapes the repository
	ErrInvalidDestPath = errors.New("invalid destination path (must be relative and within repository)")
)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate version
	if c.Version != 1 {
		return fmt.Errorf("%w: %d (only version 1 is supported)", ErrUnsupportedVersion, c.Version)
	}

	// Validate source
	if err := c.validateSource(); err != nil {
		return fmt.Errorf("invalid source configuration: %w", err)
	}

	// Validate defaults
	if err := c.validateDefaults(); err != nil {
		return fmt.Errorf("invalid defaults configuration: %w", err)
	}

	// Validate targets
	if len(c.Targets) == 0 {
		return ErrNoTargets
	}

	for i, target := range c.Targets {
		if err := target.validate(); err != nil {
			return fmt.Errorf("invalid target[%d] configuration: %w", i, err)
		}
	}

	// Check for duplicate target repositories
	seen := make(map[string]bool)
	for _, target := range c.Targets {
		if seen[target.Repo] {
			return fmt.Errorf("%w: %s", ErrDuplicateTarget, target.Repo)
		}

		seen[target.Repo] = true
	}

	return nil
}

// validateSource validates source configuration
func (c *Config) validateSource() error {
	if c.Source.Repo == "" {
		return ErrSourceRepoRequired
	}

	if !repoRegex.MatchString(c.Source.Repo) {
		return fmt.Errorf("%w: %s", ErrInvalidRepoFormat, c.Source.Repo)
	}

	if c.Source.Branch == "" {
		return ErrSourceBranchRequired
	}

	if !branchRegex.MatchString(c.Source.Branch) {
		return fmt.Errorf("%w: %s", ErrInvalidBranchName, c.Source.Branch)
	}

	return nil
}

// validateDefaults validates default configuration
func (c *Config) validateDefaults() error {
	if c.Defaults.BranchPrefix != "" && !branchRegex.MatchString(c.Defaults.BranchPrefix) {
		return fmt.Errorf("%w: %s", ErrInvalidBranchPrefix, c.Defaults.BranchPrefix)
	}

	for _, label := range c.Defaults.PRLabels {
		if strings.TrimSpace(label) == "" {
			return ErrEmptyPRLabel
		}
	}

	return nil
}

// validate validates a target configuration
func (t *TargetConfig) validate() error {
	if t.Repo == "" {
		return ErrRepoRequired
	}

	if !repoRegex.MatchString(t.Repo) {
		return fmt.Errorf("%w: %s", ErrInvalidRepoFormat, t.Repo)
	}

	if len(t.Files) == 0 {
		return fmt.Errorf("%w for repository: %s", ErrNoFileMappings, t.Repo)
	}

	// Validate file mappings
	seenDest := make(map[string]bool)

	for i, file := range t.Files {
		if err := file.validate(); err != nil {
			return fmt.Errorf("invalid file mapping[%d]: %w", i, err)
		}

		// Check for duplicate destinations
		if seenDest[file.Dest] {
			return fmt.Errorf("%w: %s", ErrDuplicateDestination, file.Dest)
		}

		seenDest[file.Dest] = true
	}

	return nil
}

// validate validates a file mapping
func (f *FileMapping) validate() error {
	if f.Src == "" {
		return ErrSourcePathRequired
	}

	if f.Dest == "" {
		return ErrDestPathRequired
	}

	// Ensure paths are clean and don't escape repository
	cleanSrc := filepath.Clean(f.Src)
	if strings.HasPrefix(cleanSrc, "..") || filepath.IsAbs(cleanSrc) {
		return fmt.Errorf("%w: %s", ErrInvalidSourcePath, f.Src)
	}

	cleanDest := filepath.Clean(f.Dest)
	if strings.HasPrefix(cleanDest, "..") || filepath.IsAbs(cleanDest) {
		return fmt.Errorf("%w: %s", ErrInvalidDestPath, f.Dest)
	}

	return nil
}
