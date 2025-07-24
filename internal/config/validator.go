package config

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/sirupsen/logrus"
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
	return c.ValidateWithLogging(context.Background(), nil)
}

// ValidateWithLogging checks if the configuration is valid with comprehensive debug logging support.
//
// This method provides detailed visibility into configuration validation when debug logging is enabled,
// including step-by-step validation progress, warnings for potential issues, and detailed error context.
//
// Parameters:
// - ctx: Context for cancellation control
// - logConfig: Configuration for debug logging and verbose settings
//
// Returns:
// - Error if validation fails
//
// Side Effects:
// - Logs detailed validation progress when --debug-config flag is enabled
// - Records validation timing and warning information
func (c *Config) ValidateWithLogging(ctx context.Context, logConfig *logging.LogConfig) error {
	logger := logging.WithStandardFields(logrus.StandardLogger(), logConfig, logging.ComponentNames.Config)
	start := time.Now()

	// Enhanced debug logging when --debug-config flag is enabled
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.Operation:   logging.OperationTypes.ConfigValidate,
			"version":                          c.Version,
			logging.StandardFields.SourceRepo:  c.Source.Repo,
			logging.StandardFields.BranchName:  c.Source.Branch,
			logging.StandardFields.TargetCount: len(c.Targets),
		}).Debug("Starting configuration validation")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("validation canceled: %w", ctx.Err())
	default:
	}

	// Validate version
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField("version", c.Version).Trace("Validating configuration version")
	}

	if c.Version != 1 {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"expected":                       1,
				"actual":                         c.Version,
				logging.StandardFields.ErrorType: "unsupported_version",
			}).Error("Unsupported configuration version")
		}
		return fmt.Errorf("%w: %d (only version 1 is supported)", ErrUnsupportedVersion, c.Version)
	}

	// Validate source
	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Validating source configuration")
	}

	if err := c.validateSourceWithLogging(ctx, logConfig); err != nil {
		return fmt.Errorf("invalid source configuration: %w", err)
	}

	// Validate defaults
	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Validating defaults configuration")
	}

	if err := c.validateDefaultsWithLogging(ctx, logConfig); err != nil {
		return fmt.Errorf("invalid defaults configuration: %w", err)
	}

	// Validate targets
	if len(c.Targets) == 0 {
		if logConfig != nil && logConfig.Debug.Config {
			logger.Error("No target repositories specified")
		}
		return ErrNoTargets
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField(logging.StandardFields.TargetCount, len(c.Targets)).Debug("Validating target repositories")
	}

	for i, target := range c.Targets {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("validation canceled: %w", ctx.Err())
		default:
		}

		targetLogger := logger
		if logConfig != nil && logConfig.Debug.Config {
			targetLogger = logger.WithFields(logrus.Fields{
				"target_index":                    i,
				logging.StandardFields.TargetRepo: target.Repo,
			})
			targetLogger.Trace("Validating target repository")
		}

		if err := target.validateWithLogging(ctx, logConfig, targetLogger); err != nil {
			return fmt.Errorf("invalid target[%d] configuration: %w", i, err)
		}
	}

	// Check for duplicate target repositories
	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Checking for duplicate target repositories")
	}

	seen := make(map[string]bool)
	for _, target := range c.Targets {
		if seen[target.Repo] {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithFields(logrus.Fields{
					"duplicate_repo":                 target.Repo,
					logging.StandardFields.ErrorType: "duplicate_target",
				}).Error("Duplicate target repository found")
			}
			return fmt.Errorf("%w: %s", ErrDuplicateTarget, target.Repo)
		}

		seen[target.Repo] = true
	}

	// Log successful validation completion
	duration := time.Since(start)
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.DurationMs: duration.Milliseconds(),
			"targets_valid":                   len(c.Targets),
			logging.StandardFields.Status:     "completed",
		}).Debug("Configuration validation completed successfully")
	}

	return nil
}

// validateSourceWithLogging validates source configuration with debug logging support.
//
// Parameters:
// - ctx: Context for cancellation control
// - logConfig: Configuration for debug logging
//
// Returns:
// - Error if source configuration is invalid
//
// Side Effects:
// - Logs detailed source validation when --debug-config flag is enabled
func (c *Config) validateSourceWithLogging(ctx context.Context, logConfig *logging.LogConfig) error {
	logger := logging.WithStandardFields(logrus.StandardLogger(), logConfig, "config-source")

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.RepoName:   c.Source.Repo,
			logging.StandardFields.BranchName: c.Source.Branch,
		}).Trace("Validating source repository configuration")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("source validation canceled: %w", ctx.Err())
	default:
	}

	// Validate repository field
	if c.Source.Repo == "" {
		if logConfig != nil && logConfig.Debug.Config {
			logger.Error("Source repository is required but not specified")
		}
		return ErrSourceRepoRequired
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField("repo_format", c.Source.Repo).Trace("Validating source repository format")
	}

	if !repoRegex.MatchString(c.Source.Repo) {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				logging.StandardFields.RepoName:  c.Source.Repo,
				"expected_format":                "org/repo",
				"regex_pattern":                  repoRegex.String(),
				logging.StandardFields.ErrorType: "invalid_repo_format",
			}).Error("Invalid source repository format")
		}
		return fmt.Errorf("%w: %s", ErrInvalidRepoFormat, c.Source.Repo)
	}

	// Validate branch field
	if c.Source.Branch == "" {
		if logConfig != nil && logConfig.Debug.Config {
			logger.Error("Source branch is required but not specified")
		}
		return ErrSourceBranchRequired
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField(logging.StandardFields.BranchName, c.Source.Branch).Trace("Validating source branch name")
	}

	if !branchRegex.MatchString(c.Source.Branch) {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				logging.StandardFields.BranchName: c.Source.Branch,
				"regex_pattern":                   branchRegex.String(),
				logging.StandardFields.ErrorType:  "invalid_branch_name",
			}).Error("Invalid source branch name")
		}
		return fmt.Errorf("%w: %s", ErrInvalidBranchName, c.Source.Branch)
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Source configuration validation completed successfully")
	}

	return nil
}

// validateDefaultsWithLogging validates default configuration with debug logging support.
//
// Parameters:
// - ctx: Context for cancellation control
// - logConfig: Configuration for debug logging
//
// Returns:
// - Error if defaults configuration is invalid
//
// Side Effects:
// - Logs detailed defaults validation when --debug-config flag is enabled
func (c *Config) validateDefaultsWithLogging(ctx context.Context, logConfig *logging.LogConfig) error {
	logger := logrus.WithField("component", "config-defaults")

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			"branch_prefix": c.Defaults.BranchPrefix,
			"pr_labels":     c.Defaults.PRLabels,
		}).Trace("Validating defaults configuration")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("defaults validation canceled: %w", ctx.Err())
	default:
	}

	// Validate branch prefix
	if c.Defaults.BranchPrefix != "" {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithField("branch_prefix", c.Defaults.BranchPrefix).Trace("Validating branch prefix format")
		}

		if !branchRegex.MatchString(c.Defaults.BranchPrefix) {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithFields(logrus.Fields{
					"branch_prefix": c.Defaults.BranchPrefix,
					"regex_pattern": branchRegex.String(),
				}).Error("Invalid branch prefix format")
			}
			return fmt.Errorf("%w: %s", ErrInvalidBranchPrefix, c.Defaults.BranchPrefix)
		}
	} else if logConfig != nil && logConfig.Debug.Config {
		logger.Trace("No branch prefix specified, will use default")
	}

	// Validate PR labels
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField("label_count", len(c.Defaults.PRLabels)).Trace("Validating PR labels")
	}

	for i, label := range c.Defaults.PRLabels {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"label_index": i,
				"label":       label,
			}).Trace("Validating PR label")
		}

		if strings.TrimSpace(label) == "" {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("label_index", i).Error("Empty PR label found")
			}
			return ErrEmptyPRLabel
		}
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Defaults configuration validation completed successfully")
	}

	return nil
}

// validateWithLogging validates a target configuration with debug logging support.
//
// Parameters:
// - ctx: Context for cancellation control
// - logConfig: Configuration for debug logging
// - logger: Logger entry for output
//
// Returns:
// - Error if target configuration is invalid
//
// Side Effects:
// - Logs detailed target validation when --debug-config flag is enabled
func (t *TargetConfig) validateWithLogging(ctx context.Context, logConfig *logging.LogConfig, logger *logrus.Entry) error {
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			"repo":                    t.Repo,
			"file_count":              len(t.Files),
			"has_transform_repo_name": t.Transform.RepoName,
			"has_transform_variables": len(t.Transform.Variables) > 0,
		}).Trace("Validating target repository configuration")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("target validation canceled: %w", ctx.Err())
	default:
	}

	// Validate repository field
	if t.Repo == "" {
		if logConfig != nil && logConfig.Debug.Config {
			logger.Error("Target repository is required but not specified")
		}
		return ErrRepoRequired
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField("repo_format", t.Repo).Trace("Validating target repository format")
	}

	if !repoRegex.MatchString(t.Repo) {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"repo":            t.Repo,
				"expected_format": "org/repo",
				"regex_pattern":   repoRegex.String(),
			}).Error("Invalid target repository format")
		}
		return fmt.Errorf("%w: %s", ErrInvalidRepoFormat, t.Repo)
	}

	// Validate file mappings exist
	if len(t.Files) == 0 {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithField("repo", t.Repo).Error("No file mappings specified for target repository")
		}
		return fmt.Errorf("%w for repository: %s", ErrNoFileMappings, t.Repo)
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField("file_count", len(t.Files)).Debug("Validating file mappings")
	}

	// Validate file mappings
	seenDest := make(map[string]bool)

	for i, file := range t.Files {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("file mapping validation canceled: %w", ctx.Err())
		default:
		}

		fileLogger := logger
		if logConfig != nil && logConfig.Debug.Config {
			fileLogger = logger.WithFields(logrus.Fields{
				"file_index": i,
				"src":        file.Src,
				"dest":       file.Dest,
			})
			fileLogger.Trace("Validating file mapping")
		}

		if err := file.validateWithLogging(ctx, logConfig, fileLogger); err != nil {
			return fmt.Errorf("invalid file mapping[%d]: %w", i, err)
		}

		// Check for duplicate destinations
		if seenDest[file.Dest] {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithFields(logrus.Fields{
					"duplicate_dest": file.Dest,
					"file_index":     i,
				}).Error("Duplicate destination file found")
			}
			return fmt.Errorf("%w: %s", ErrDuplicateDestination, file.Dest)
		}

		seenDest[file.Dest] = true
	}

	// Log transform configuration if present
	if logConfig != nil && logConfig.Debug.Config {
		if t.Transform.RepoName || len(t.Transform.Variables) > 0 {
			logger.WithFields(logrus.Fields{
				"repo_name_transform": t.Transform.RepoName,
				"variable_count":      len(t.Transform.Variables),
			}).Debug("Transform configuration detected")

			if len(t.Transform.Variables) > 0 {
				for key, value := range t.Transform.Variables {
					logger.WithFields(logrus.Fields{
						"variable": key,
						"value":    value,
					}).Trace("Transform variable")
				}
			}
		}
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Target configuration validation completed successfully")
	}

	return nil
}

// validateWithLogging validates a file mapping with debug logging support.
//
// Parameters:
// - ctx: Context for cancellation control
// - logConfig: Configuration for debug logging
// - logger: Logger entry for output
//
// Returns:
// - Error if file mapping is invalid
//
// Side Effects:
// - Logs detailed file mapping validation when --debug-config flag is enabled
func (f *FileMapping) validateWithLogging(ctx context.Context, logConfig *logging.LogConfig, logger *logrus.Entry) error {
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			"src":  f.Src,
			"dest": f.Dest,
		}).Trace("Validating file mapping")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("file mapping validation canceled: %w", ctx.Err())
	default:
	}

	// Validate source path
	if f.Src == "" {
		if logConfig != nil && logConfig.Debug.Config {
			logger.Error("Source file path is required but not specified")
		}
		return ErrSourcePathRequired
	}

	// Validate destination path
	if f.Dest == "" {
		if logConfig != nil && logConfig.Debug.Config {
			logger.Error("Destination file path is required but not specified")
		}
		return ErrDestPathRequired
	}

	// Ensure source path is clean and doesn't escape repository
	cleanSrc := filepath.Clean(f.Src)
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			"original_src": f.Src,
			"clean_src":    cleanSrc,
		}).Trace("Validating source path safety")
	}

	if strings.HasPrefix(cleanSrc, "..") || filepath.IsAbs(cleanSrc) {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"src":         f.Src,
				"clean_src":   cleanSrc,
				"is_absolute": filepath.IsAbs(cleanSrc),
				"has_dotdot":  strings.HasPrefix(cleanSrc, ".."),
			}).Error("Invalid source path: must be relative and within repository")
		}
		return fmt.Errorf("%w: %s", ErrInvalidSourcePath, f.Src)
	}

	// Ensure destination path is clean and doesn't escape repository
	cleanDest := filepath.Clean(f.Dest)
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			"original_dest": f.Dest,
			"clean_dest":    cleanDest,
		}).Trace("Validating destination path safety")
	}

	if strings.HasPrefix(cleanDest, "..") || filepath.IsAbs(cleanDest) {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"dest":        f.Dest,
				"clean_dest":  cleanDest,
				"is_absolute": filepath.IsAbs(cleanDest),
				"has_dotdot":  strings.HasPrefix(cleanDest, ".."),
			}).Error("Invalid destination path: must be relative and within repository")
		}
		return fmt.Errorf("%w: %s", ErrInvalidDestPath, f.Dest)
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("File mapping validation completed successfully")
	}

	return nil
}
