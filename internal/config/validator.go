package config

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

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
	// ErrGroupNameEmpty indicates group name is empty
	ErrGroupNameEmpty = errors.New("group name cannot be empty")
	// ErrGroupIDEmpty indicates group ID is empty
	ErrGroupIDEmpty = errors.New("group id cannot be empty")
	// ErrDuplicateDestPath indicates destination path is used by multiple mappings
	ErrDuplicateDestPath = errors.New("destination path already in use")
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

	// Debug logging when --debug-config flag is enabled
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.Operation: logging.OperationTypes.ConfigValidate,
			"version":                        c.Version,
			"group_count":                    len(c.Groups),
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

	// Validate groups
	if len(c.Groups) == 0 {
		if logConfig != nil && logConfig.Debug.Config {
			logger.Error("No groups specified")
		}
		return ErrNoTargets
	}

	// Validate all groups
	for i, group := range c.Groups {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("validation canceled: %w", ctx.Err())
		default:
		}

		// Validate group name and ID
		if group.Name == "" {
			return fmt.Errorf("group[%d]: %w", i, ErrGroupNameEmpty)
		}
		if group.ID == "" {
			return fmt.Errorf("group[%d]: %w", i, ErrGroupIDEmpty)
		}

		// Validate source
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"group_index": i,
				"group_name":  group.Name,
				"group_id":    group.ID,
			}).Debug("Validating group source configuration")
		}

		if err := c.validateGroupSourceWithLogging(ctx, logConfig, group); err != nil {
			return fmt.Errorf("invalid group[%d] (%s) source configuration: %w", i, group.Name, err)
		}

		// Validate global
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"group_index": i,
				"group_name":  group.Name,
			}).Debug("Validating group global configuration")
		}

		if err := c.validateGroupGlobalWithLogging(ctx, logConfig, group); err != nil {
			return fmt.Errorf("invalid group[%d] (%s) global configuration: %w", i, group.Name, err)
		}

		// Validate defaults
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"group_index": i,
				"group_name":  group.Name,
			}).Debug("Validating group defaults configuration")
		}

		if err := c.validateGroupDefaultsWithLogging(ctx, logConfig, group); err != nil {
			return fmt.Errorf("invalid group[%d] (%s) defaults configuration: %w", i, group.Name, err)
		}

		// Validate targets
		if len(group.Targets) == 0 {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithFields(logrus.Fields{
					"group_index": i,
					"group_name":  group.Name,
				}).Error("No target repositories specified in group")
			}
			return fmt.Errorf("group[%d] (%s): %w", i, group.Name, ErrNoTargets)
		}

		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"group_index":                      i,
				"group_name":                       group.Name,
				logging.StandardFields.TargetCount: len(group.Targets),
			}).Debug("Validating group target repositories")
		}

		for j, target := range group.Targets {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				return fmt.Errorf("validation canceled: %w", ctx.Err())
			default:
			}

			targetLogger := logger
			if logConfig != nil && logConfig.Debug.Config {
				targetLogger = logger.WithFields(logrus.Fields{
					"group_index":                     i,
					"group_name":                      group.Name,
					"target_index":                    j,
					logging.StandardFields.TargetRepo: target.Repo,
				})
				targetLogger.Trace("Validating target repository")
			}

			if err := target.validateWithLogging(ctx, logConfig, targetLogger); err != nil {
				return fmt.Errorf("invalid group[%d] (%s) target[%d] configuration: %w", i, group.Name, j, err)
			}
		}

		// Check for duplicate target repositories in group
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				"group_index": i,
				"group_name":  group.Name,
			}).Debug("Checking for duplicate target repositories in group")
		}

		seen := make(map[string]bool)
		for _, target := range group.Targets {
			if seen[target.Repo] {
				if logConfig != nil && logConfig.Debug.Config {
					logger.WithFields(logrus.Fields{
						"group_index":                    i,
						"group_name":                     group.Name,
						"duplicate_repo":                 target.Repo,
						logging.StandardFields.ErrorType: "duplicate_target",
					}).Error("Duplicate target repository found in group")
				}
				return fmt.Errorf("group[%d] (%s): %w: %s", i, group.Name, ErrDuplicateTarget, target.Repo)
			}

			seen[target.Repo] = true
		}
	}

	// Log successful validation completion
	duration := time.Since(start)
	if logConfig != nil && logConfig.Debug.Config {
		totalTargets := 0
		for _, group := range c.Groups {
			totalTargets += len(group.Targets)
		}

		logger.WithFields(logrus.Fields{
			logging.StandardFields.DurationMs: duration.Milliseconds(),
			"groups_valid":                    len(c.Groups),
			"targets_valid":                   totalTargets,
			logging.StandardFields.Status:     "completed",
		}).Debug("Configuration validation completed successfully")
	}

	return nil
}

// validateGroupSourceWithLogging validates group source configuration with debug logging support.
func (c *Config) validateGroupSourceWithLogging(ctx context.Context, logConfig *logging.LogConfig, group Group) error {
	logger := logging.WithStandardFields(logrus.StandardLogger(), logConfig, "config-group-source")

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.RepoName:   group.Source.Repo,
			logging.StandardFields.BranchName: group.Source.Branch,
			"group_name":                      group.Name,
			"group_id":                        group.ID,
		}).Trace("Validating group source repository configuration")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("group source validation canceled: %w", ctx.Err())
	default:
	}

	// Use centralized validation for source configuration
	if err := validation.ValidateSourceConfig(group.Source.Repo, group.Source.Branch); err != nil {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				logging.StandardFields.RepoName:   group.Source.Repo,
				logging.StandardFields.BranchName: group.Source.Branch,
				logging.StandardFields.ErrorType:  "validation_failed",
				"group_name":                      group.Name,
				"group_id":                        group.ID,
			}).Error("Group source configuration validation failed")
		}
		return err
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Group source configuration validation completed successfully")
	}

	return nil
}

// validateGroupGlobalWithLogging validates group global configuration with debug logging support.
func (c *Config) validateGroupGlobalWithLogging(ctx context.Context, logConfig *logging.LogConfig, group Group) error {
	logger := logrus.WithField("component", "config-group-global")

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			"pr_labels":         group.Global.PRLabels,
			"pr_assignees":      group.Global.PRAssignees,
			"pr_reviewers":      group.Global.PRReviewers,
			"pr_team_reviewers": group.Global.PRTeamReviewers,
			"group_name":        group.Name,
			"group_id":          group.ID,
		}).Trace("Validating group global configuration")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("group global validation canceled: %w", ctx.Err())
	default:
	}

	// Validate PR labels
	for i, label := range group.Global.PRLabels {
		if err := validation.ValidateNonEmpty("group global PR label", label); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("label_index", i).Error("Empty group global PR label found")
			}
			return err
		}
	}

	// Validate PR assignees, reviewers, team reviewers
	for i, assignee := range group.Global.PRAssignees {
		if err := validation.ValidateNonEmpty("group global PR assignee", assignee); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("assignee_index", i).Error("Empty group global PR assignee found")
			}
			return err
		}
	}

	for i, reviewer := range group.Global.PRReviewers {
		if err := validation.ValidateNonEmpty("group global PR reviewer", reviewer); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("reviewer_index", i).Error("Empty group global PR reviewer found")
			}
			return err
		}
	}

	for i, teamReviewer := range group.Global.PRTeamReviewers {
		if err := validation.ValidateNonEmpty("group global PR team reviewer", teamReviewer); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("team_reviewer_index", i).Error("Empty group global PR team reviewer found")
			}
			return err
		}
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Group global configuration validation completed successfully")
	}

	return nil
}

// validateGroupDefaultsWithLogging validates group defaults configuration with debug logging support.
func (c *Config) validateGroupDefaultsWithLogging(ctx context.Context, logConfig *logging.LogConfig, group Group) error {
	logger := logrus.WithField("component", "config-group-defaults")

	if logConfig != nil && logConfig.Debug.Config {
		logger.WithFields(logrus.Fields{
			"branch_prefix": group.Defaults.BranchPrefix,
			"pr_labels":     group.Defaults.PRLabels,
			"group_name":    group.Name,
			"group_id":      group.ID,
		}).Trace("Validating group defaults configuration")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("group defaults validation canceled: %w", ctx.Err())
	default:
	}

	// Validate branch prefix using centralized validation
	if err := validation.ValidateBranchPrefix(group.Defaults.BranchPrefix); err != nil {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithField("branch_prefix", group.Defaults.BranchPrefix).Error("Invalid group branch prefix format")
		}
		return err
	}

	// Validate PR labels
	for i, label := range group.Defaults.PRLabels {
		if err := validation.ValidateNonEmpty("group PR label", label); err != nil {
			if logConfig != nil && logConfig.Debug.Config {
				logger.WithField("label_index", i).Error("Empty group PR label found")
			}
			return err
		}
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Group defaults configuration validation completed successfully")
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
