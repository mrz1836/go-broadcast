package config

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/validation"
	"github.com/sirupsen/logrus"
)

var (
	// ErrUnsupportedVersion indicates the configuration version is not supported
	ErrUnsupportedVersion = errors.New("unsupported config version")
	// ErrNoTargets indicates no target repositories were specified
	ErrNoTargets = errors.New("at least one target repository must be specified")
	// ErrDuplicateTarget indicates a target repository is specified multiple times
	ErrDuplicateTarget = errors.New("duplicate target repository")
	// ErrNoMappings indicates no file or directory mappings were specified
	ErrNoMappings = errors.New("at least one file or directory mapping is required")
	// ErrEmptySourcePath indicates a directory source path is empty
	ErrEmptySourcePath = errors.New("source path cannot be empty")
	// ErrEmptyDestPath indicates a directory destination path is empty
	ErrEmptyDestPath = errors.New("destination path cannot be empty")
	// ErrPathTraversal indicates path traversal is not allowed
	ErrPathTraversal = errors.New("path traversal not allowed")
	// ErrDuplicateDestPath indicates destination path is used by multiple mappings
	ErrDuplicateDestPath = errors.New("destination path already in use")
	// ErrNoSourceMapping indicates no source mappings were specified
	ErrNoSourceMapping = errors.New("at least one source mapping must be specified")
	// ErrDuplicateSourceID indicates a source ID is used multiple times
	ErrDuplicateSourceID = errors.New("duplicate source ID")
	// ErrMissingSourceID indicates a source ID is required for conflict resolution
	ErrMissingSourceID = errors.New("source ID is required when using conflict resolution")

	// sourceIDRegex validates source ID format
	sourceIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// isValidSourceID checks if a source ID contains only valid characters
func isValidSourceID(id string) bool {
	return sourceIDRegex.MatchString(id)
}

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

	// Debug logging when --debug-config flag is enabled
	if logConfig != nil && logConfig.Debug.Config {
		fields := logrus.Fields{
			logging.StandardFields.Operation: logging.OperationTypes.ConfigValidate,
			"version":                        c.Version,
		}

		// Handle multi-source configurations
		fields["mappings_count"] = len(c.Mappings)
		// Count total unique targets
		targetMap := make(map[string]bool)
		for _, mapping := range c.Mappings {
			for _, target := range mapping.Targets {
				targetMap[target.Repo] = true
			}
		}
		fields[logging.StandardFields.TargetCount] = len(targetMap)

		logger.WithFields(fields).Debug("Starting configuration validation")
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

	// Validate source configuration
	if len(c.Mappings) > 0 {
		if logConfig != nil && logConfig.Debug.Config {
			logger.Debug("Validating source mappings")
		}

		// Validate each source in mappings
		for i, mapping := range c.Mappings {
			if err := validation.ValidateSourceConfig(mapping.Source.Repo, mapping.Source.Branch); err != nil {
				return fmt.Errorf("invalid source configuration in mapping %d: %w", i+1, err)
			}

			// Validate source ID if provided
			if mapping.Source.ID != "" && !isValidSourceID(mapping.Source.ID) {
				return appErrors.InvalidSourceIDError(mapping.Source.ID, i)
			}
		}
	} else {
		return appErrors.NoSourceConfigFoundError()
	}

	// Validate global
	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Validating global configuration")
	}

	if err := c.validateGlobalWithLogging(ctx, logConfig); err != nil {
		return fmt.Errorf("invalid global configuration: %w", err)
	}

	// Validate defaults
	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Validating defaults configuration")
	}

	if err := c.validateDefaultsWithLogging(ctx, logConfig); err != nil {
		return fmt.Errorf("invalid defaults configuration: %w", err)
	}

	// Validate targets
	allTargets := []TargetConfig{}

	if len(c.Mappings) > 0 {
		// Collect all targets from mappings
		for _, mapping := range c.Mappings {
			if len(mapping.Targets) == 0 {
				if logConfig != nil && logConfig.Debug.Config {
					logger.Error("No targets specified in mapping")
				}
				return appErrors.MappingNoTargetsError(mapping.Source.Repo)
			}
			allTargets = append(allTargets, mapping.Targets...)
		}
	} else {
		return appErrors.NoSourceConfigFoundError()
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField(logging.StandardFields.TargetCount, len(allTargets)).Debug("Validating target repositories")
	}

	for i, target := range allTargets {
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

	// Check for duplicate target repositories within each source mapping
	// (Multi-source configurations allow different sources to target the same repository)
	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Checking for duplicate target repositories within source mappings")
	}

	for mappingIdx, mapping := range c.Mappings {
		seen := make(map[string]bool)
		for _, target := range mapping.Targets {
			if seen[target.Repo] {
				if logConfig != nil && logConfig.Debug.Config {
					logger.WithFields(logrus.Fields{
						"duplicate_repo":                 target.Repo,
						"source_mapping_index":           mappingIdx,
						"source_repo":                    mapping.Source.Repo,
						logging.StandardFields.ErrorType: "duplicate_target_in_source",
					}).Error("Duplicate target repository found within source mapping")
				}
				return fmt.Errorf("%w: %s (within source mapping %d)", ErrDuplicateTarget, target.Repo, mappingIdx)
			}
			seen[target.Repo] = true
		}
	}

	// Log successful validation completion
	duration := time.Since(start)
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.DurationMs: duration.Milliseconds(),
			"targets_valid":                   len(allTargets),
			logging.StandardFields.Status:     "completed",
		}).Debug("Configuration validation completed successfully")
	}

	return nil
}

// validateGlobalWithLogging validates global configuration with debug logging support.
//
// Parameters:
// - ctx: Context for cancellation control
// - logConfig: Configuration for debug logging
//
// Returns:
// - Error if global configuration is invalid
//
// Side Effects:
// - Logs detailed global validation when --debug-config flag is enabled
func (c *Config) validateGlobalWithLogging(ctx context.Context, logConfig *logging.LogConfig) error {
	logger := logrus.WithField("component", "config-global")

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			"pr_labels":         c.Global.PRLabels,
			"pr_assignees":      c.Global.PRAssignees,
			"pr_reviewers":      c.Global.PRReviewers,
			"pr_team_reviewers": c.Global.PRTeamReviewers,
		}).Trace("Validating global configuration")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("global validation canceled: %w", ctx.Err())
	default:
	}

	// Validate PR labels
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField("label_count", len(c.Global.PRLabels)).Trace("Validating global PR labels")
	}

	for i, label := range c.Global.PRLabels {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"label_index": i,
				"label":       label,
			}).Trace("Validating global PR label")
		}

		if err := validation.ValidateNonEmpty("global PR label", label); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("label_index", i).Error("Empty global PR label found")
			}
			return err
		}
	}

	// Validate PR assignees (basic non-empty validation)
	for i, assignee := range c.Global.PRAssignees {
		if err := validation.ValidateNonEmpty("global PR assignee", assignee); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("assignee_index", i).Error("Empty global PR assignee found")
			}
			return err
		}
	}

	// Validate PR reviewers (basic non-empty validation)
	for i, reviewer := range c.Global.PRReviewers {
		if err := validation.ValidateNonEmpty("global PR reviewer", reviewer); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("reviewer_index", i).Error("Empty global PR reviewer found")
			}
			return err
		}
	}

	// Validate PR team reviewers (basic non-empty validation)
	for i, teamReviewer := range c.Global.PRTeamReviewers {
		if err := validation.ValidateNonEmpty("global PR team reviewer", teamReviewer); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("team_reviewer_index", i).Error("Empty global PR team reviewer found")
			}
			return err
		}
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Global configuration validation completed successfully")
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

	// Validate branch prefix using centralized validation
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField("branch_prefix", c.Defaults.BranchPrefix).Trace("Validating branch prefix format")
	}

	if err := validation.ValidateBranchPrefix(c.Defaults.BranchPrefix); err != nil {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithField("branch_prefix", c.Defaults.BranchPrefix).Error("Invalid branch prefix format")
		}
		return err
	}

	if c.Defaults.BranchPrefix == "" && logConfig != nil && logConfig.Debug.Config {
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

		if err := validation.ValidateNonEmpty("PR label", label); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("label_index", i).Error("Empty PR label found")
			}
			return err
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

	// Validate that we have at least one file or directory mapping
	if len(t.Files) == 0 && len(t.Directories) == 0 {
		return ErrNoMappings
	}

	// Convert file mappings to validation format
	fileMappings := make([]validation.FileMapping, 0, len(t.Files))
	for _, file := range t.Files {
		fileMappings = append(fileMappings, validation.FileMapping{
			Src:  file.Src,
			Dest: file.Dest,
		})
	}

	// Use centralized validation for target configuration only if we have files
	if len(fileMappings) > 0 {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithField("repo_format", t.Repo).Trace("Validating target repository configuration")
		}

		if err := validation.ValidateTargetConfig(t.Repo, fileMappings); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithFields(logrus.Fields{
					"repo":       t.Repo,
					"file_count": len(t.Files),
				}).Error("Target repository validation failed")
			}
			return err
		}
	} else {
		// Validate repo name when we only have directories
		if err := validation.ValidateRepoName(t.Repo); err != nil {
			return err
		}
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField("file_count", len(t.Files)).Debug("File mappings validated via centralized validation")
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

	// Validate PR labels for this target
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField("label_count", len(t.PRLabels)).Trace("Validating target-specific PR labels")
	}

	for i, label := range t.PRLabels {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"label_index": i,
				"label":       label,
			}).Trace("Validating target PR label")
		}

		if err := validation.ValidateNonEmpty("target PR label", label); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("label_index", i).Error("Empty target PR label found")
			}
			return err
		}
	}

	// Validate directories if present
	if len(t.Directories) > 0 {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithField("directory_count", len(t.Directories)).Debug("Validating directory mappings")
		}

		if err := t.validateDirectories(ctx, logger); err != nil {
			return fmt.Errorf("invalid directory configuration: %w", err)
		}
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Target configuration validation completed successfully")
	}

	return nil
}

// validateDirectories validates directory mappings
func (t *TargetConfig) validateDirectories(_ context.Context, _ *logrus.Entry) error {
	// Check for empty directories
	for i, dir := range t.Directories {
		if dir.Src == "" {
			return fmt.Errorf("directory[%d]: %w", i, ErrEmptySourcePath)
		}
		if dir.Dest == "" {
			return fmt.Errorf("directory[%d]: %w", i, ErrEmptyDestPath)
		}

		// Validate paths don't contain path traversal
		if strings.Contains(dir.Src, "..") || strings.Contains(dir.Dest, "..") {
			return fmt.Errorf("directory[%d]: %w", i, ErrPathTraversal)
		}

		// Validate exclusion patterns
		for _, pattern := range dir.Exclude {
			if _, err := filepath.Match(pattern, "test"); err != nil {
				return fmt.Errorf("directory[%d]: invalid exclusion pattern %q: %w", i, pattern, err)
			}
		}
	}

	// Check for conflicts between files and directories
	return t.validateFileDirectoryConflicts()
}

// validateFileDirectoryConflicts ensures no conflicts between file and directory mappings
func (t *TargetConfig) validateFileDirectoryConflicts() error {
	// Build map of all destination paths
	destPaths := make(map[string]string)

	// Add file destinations
	for _, file := range t.Files {
		destPaths[file.Dest] = "file"
	}

	// Check directory destinations don't conflict
	for _, dir := range t.Directories {
		if existing, exists := destPaths[dir.Dest]; exists {
			return fmt.Errorf("destination path %q used by both %s and directory: %w", dir.Dest, existing, ErrDuplicateDestPath)
		}
		destPaths[dir.Dest] = "directory"
	}

	return nil
}
