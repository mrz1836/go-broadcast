package transform

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/sirupsen/logrus"
)

// templateTransformer replaces template variables in content
type templateTransformer struct {
	logger    *logrus.Logger
	logConfig *logging.LogConfig
}

// NewTemplateTransformer creates a new template variable transformer.
//
// Parameters:
// - logger: Logger instance for general logging
// - logConfig: Configuration for debug logging and verbose settings
//
// Returns:
// - Transformer interface implementation for template variable replacement
func NewTemplateTransformer(logger *logrus.Logger, logConfig *logging.LogConfig) Transformer {
	return &templateTransformer{
		logger:    logger,
		logConfig: logConfig,
	}
}

// Name returns the name of this transformer
func (t *templateTransformer) Name() string {
	return "template-variable-replacer"
}

// Transform replaces template variables in the content with comprehensive debug logging support.
//
// This method provides detailed visibility into template transformation when debug logging is enabled,
// including before/after content, variable substitution details, timing metrics, and content size analysis.
//
// Parameters:
// - content: The original file content to transform
// - ctx: Transform context containing variables and configuration
//
// Returns:
// - Transformed content as byte slice
// - Error if transformation fails
//
// Side Effects:
// - Logs detailed transformation information when --debug-transform flag is enabled
// - Records transformation timing and content size metrics
func (t *templateTransformer) Transform(content []byte, ctx Context) ([]byte, error) {
	logger := logging.WithStandardFields(t.logger, t.logConfig, logging.ComponentNames.Transform)
	start := time.Now()

	// Enhanced debug logging when --debug-transform flag is enabled
	if t.logConfig != nil && t.logConfig.Debug.Transform {
		logger.WithFields(logrus.Fields{
			logging.StandardFields.Operation:     logging.OperationTypes.FileTransform,
			logging.StandardFields.FilePath:      ctx.FilePath,
			logging.StandardFields.SourceRepo:    ctx.SourceRepo,
			logging.StandardFields.TargetRepo:    ctx.TargetRepo,
			logging.StandardFields.VariableCount: len(ctx.Variables),
			logging.StandardFields.ContentSize:   len(content),
		}).Debug("Starting template transformation")

		// Log original content for small files (with size limits)
		if len(content) > 0 && len(content) < 2048 {
			logger.WithField("content", string(content)).Trace("Original content")
		}

		// Log available variables
		if len(ctx.Variables) > 0 {
			for varName, value := range ctx.Variables {
				logger.WithFields(logrus.Fields{
					logging.StandardFields.Variable:      varName,
					logging.StandardFields.VariableValue: value,
				}).Trace("Available variable")
			}
		}
	}

	if len(ctx.Variables) == 0 {
		// Log completion for empty variable case
		if t.logConfig != nil && t.logConfig.Debug.Transform {
			duration := time.Since(start)
			logger.WithFields(logrus.Fields{
				logging.StandardFields.DurationMs: duration.Milliseconds(),
				"changes":                         0,
				logging.StandardFields.Status:     "completed_no_variables",
			}).Debug("Template transformation completed (no variables)")
		}
		return content, nil
	}

	result := string(content)
	replacedVars := make([]string, 0)
	replacementCount := 0

	// Sort variables by length (longest first) to avoid partial replacements
	// e.g., replace {{SERVICE_NAME}} before {{SERVICE}}
	varKeys := make([]string, 0, len(ctx.Variables))
	for k := range ctx.Variables {
		varKeys = append(varKeys, k)
	}

	// Simple bubble sort by length (descending)
	for i := 0; i < len(varKeys); i++ {
		for j := i + 1; j < len(varKeys); j++ {
			if len(varKeys[j]) > len(varKeys[i]) {
				varKeys[i], varKeys[j] = varKeys[j], varKeys[i]
			}
		}
	}

	// Replace each variable
	for _, varName := range varKeys {
		value := ctx.Variables[varName]

		// Support both {{VAR}} and ${VAR} syntax
		patterns := []string{
			fmt.Sprintf("{{%s}}", varName),
			fmt.Sprintf("${%s}", varName),
		}

		replaced := false
		patternReplacements := 0

		for _, pattern := range patterns {
			// Escape special regex characters in the pattern
			escapedPattern := regexp.QuoteMeta(pattern)
			re := regexp.MustCompile(escapedPattern)

			oldResult := result
			result = re.ReplaceAllString(result, value)

			if result != oldResult {
				replaced = true
				// Count replacements for this pattern
				currentReplacements := strings.Count(oldResult, pattern)
				patternReplacements += currentReplacements
			}
		}

		if replaced {
			replacedVars = append(replacedVars, varName)
			replacementCount += patternReplacements

			// Enhanced debug logging for individual variable replacements
			if t.logConfig != nil && t.logConfig.Debug.Transform {
				logger.WithFields(logrus.Fields{
					logging.StandardFields.Variable:      varName,
					logging.StandardFields.VariableValue: value,
					logging.StandardFields.Replacements:  patternReplacements,
				}).Trace("Variable substitution")
			}
		}
	}

	// Calculate transformation timing and metrics
	duration := time.Since(start)
	contentSizeChange := len(result) - len(content)

	if len(replacedVars) > 0 {
		if t.logConfig != nil && t.logConfig.Debug.Transform {
			// Enhanced transformation completion logging
			logger.WithFields(logrus.Fields{
				logging.StandardFields.FilePath:   ctx.FilePath,
				"variables":                       strings.Join(replacedVars, ", "),
				"total_replacements":              replacementCount,
				logging.StandardFields.DurationMs: duration.Milliseconds(),
				logging.StandardFields.SizeChange: contentSizeChange,
				"original_size":                   len(content),
				"final_size":                      len(result),
				logging.StandardFields.Status:     "completed",
			}).Debug("Template variables replaced")

			// Log final content for small files (with size limits)
			if len(result) > 0 && len(result) < 2048 {
				logger.WithField("content", result).Trace("Transformed content")
			}
		} else {
			// Basic logging for backwards compatibility
			t.logger.WithFields(logrus.Fields{
				logging.StandardFields.Component: logging.ComponentNames.Transform,
				logging.StandardFields.FilePath:  ctx.FilePath,
				"variables":                      strings.Join(replacedVars, ", "),
				logging.StandardFields.Status:    "completed",
			}).Debug("Replaced template variables")
		}
	} else {
		// Log completion for no replacements case
		if t.logConfig != nil && t.logConfig.Debug.Transform {
			logger.WithFields(logrus.Fields{
				logging.StandardFields.DurationMs: duration.Milliseconds(),
				"changes":                         0,
				logging.StandardFields.Status:     "completed_no_replacements",
			}).Debug("Template transformation completed (no replacements)")
		}
	}

	// Check for any remaining unreplaced variables and log warnings
	remainingVars := t.findUnreplacedVariables(result)
	if len(remainingVars) > 0 {
		if t.logConfig != nil && t.logConfig.Debug.Transform {
			// Enhanced unreplaced variable warning
			logger.WithFields(logrus.Fields{
				logging.StandardFields.FilePath: ctx.FilePath,
				"unreplaced_vars":               strings.Join(remainingVars, ", "),
				"available_vars":                strings.Join(varKeys, ", "),
				"unreplaced_count":              len(remainingVars),
				logging.StandardFields.Status:   "warning_unreplaced_vars",
			}).Warn("Found unreplaced template variables")
		} else {
			// Basic logging for backwards compatibility
			t.logger.WithFields(logrus.Fields{
				logging.StandardFields.Component: logging.ComponentNames.Transform,
				logging.StandardFields.FilePath:  ctx.FilePath,
				"unreplaced_vars":                strings.Join(remainingVars, ", "),
				"available_vars":                 strings.Join(varKeys, ", "),
				logging.StandardFields.Status:    "warning_unreplaced_vars",
			}).Warn("Found unreplaced template variables")
		}
	}

	return []byte(result), nil
}

// findUnreplacedVariables finds any remaining template variables in the content
func (t *templateTransformer) findUnreplacedVariables(content string) []string {
	vars := make(map[string]bool)

	// Find {{VAR}} style variables
	re1 := regexp.MustCompile(`\{\{([A-Z_][A-Z0-9_]*)\}\}`)

	matches1 := re1.FindAllStringSubmatch(content, -1)
	for _, match := range matches1 {
		if len(match) > 1 {
			vars[match[1]] = true
		}
	}

	// Find ${VAR} style variables
	re2 := regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

	matches2 := re2.FindAllStringSubmatch(content, -1)
	for _, match := range matches2 {
		if len(match) > 1 {
			vars[match[1]] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(vars))
	for v := range vars {
		result = append(result, v)
	}

	return result
}
