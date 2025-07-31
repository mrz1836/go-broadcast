package config

import (
	"context"
	"errors"
	"fmt"
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

	// Use centralized validation for source configuration
	if logConfig != nil && logConfig.Debug.Config {
		logger.WithField("repo_format", c.Source.Repo).Trace("Validating source repository format")
	}

	if err := validation.ValidateSourceConfig(c.Source.Repo, c.Source.Branch); err != nil {
		if logConfig != nil && logConfig.Debug.Config {
			logger.WithFields(logrus.Fields{
				logging.StandardFields.RepoName:   c.Source.Repo,
				logging.StandardFields.BranchName: c.Source.Branch,
				logging.StandardFields.ErrorType:  "validation_failed",
			}).Error("Source configuration validation failed")
		}
		return err
	}

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Source configuration validation completed successfully")
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

	// Convert file mappings to validation format
	fileMappings := make([]validation.FileMapping, 0, len(t.Files))
	for _, file := range t.Files {
		fileMappings = append(fileMappings, validation.FileMapping{
			Src:  file.Src,
			Dest: file.Dest,
		})
	}

	// Use centralized validation for target configuration
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

	if logConfig != nil && logConfig.Debug.Config {
		logger.Debug("Target configuration validation completed successfully")
	}

	return nil
}
