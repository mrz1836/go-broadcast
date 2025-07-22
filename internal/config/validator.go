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

	// Common validation errors
	ErrUnsupportedVersion   = errors.New("unsupported config version")
	ErrNoTargets            = errors.New("at least one target repository must be specified")
	ErrDuplicateTarget      = errors.New("duplicate target repository")
	ErrSourceRepoRequired   = errors.New("source repository is required")
	ErrInvalidRepoFormat    = errors.New("invalid repository format (expected: org/repo)")
	ErrSourceBranchRequired = errors.New("source branch is required")
	ErrInvalidBranchName    = errors.New("invalid branch name")
	ErrInvalidBranchPrefix  = errors.New("invalid branch prefix")
	ErrEmptyPRLabel         = errors.New("PR label cannot be empty")
	ErrRepoRequired         = errors.New("repository is required")
	ErrNoFileMappings       = errors.New("at least one file mapping is required")
	ErrDuplicateDestination = errors.New("duplicate destination file")
	ErrSourcePathRequired   = errors.New("source file path is required")
	ErrDestPathRequired     = errors.New("destination file path is required")
	ErrInvalidSourcePath    = errors.New("invalid source path (must be relative and within repository)")
	ErrInvalidDestPath      = errors.New("invalid destination path (must be relative and within repository)")
)

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate version
	if c.Version != 2 {
		return fmt.Errorf("%w: %d (only version 2 is supported)", ErrUnsupportedVersion, c.Version)
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

