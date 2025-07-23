package transform

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// templateTransformer replaces template variables in content
type templateTransformer struct {
	logger *logrus.Logger
}

// NewTemplateTransformer creates a new template variable transformer
func NewTemplateTransformer(logger *logrus.Logger) Transformer {
	return &templateTransformer{
		logger: logger,
	}
}

// Name returns the name of this transformer
func (t *templateTransformer) Name() string {
	return "template-variable-replacer"
}

// Transform replaces template variables in the content
func (t *templateTransformer) Transform(content []byte, ctx Context) ([]byte, error) {
	if len(ctx.Variables) == 0 {
		return content, nil
	}

	result := string(content)
	replacedVars := make([]string, 0)

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

		for _, pattern := range patterns {
			// Escape special regex characters in the pattern
			escapedPattern := regexp.QuoteMeta(pattern)
			re := regexp.MustCompile(escapedPattern)

			oldResult := result
			result = re.ReplaceAllString(result, value)

			if result != oldResult {
				replaced = true
			}
		}

		if replaced {
			replacedVars = append(replacedVars, varName)
		}
	}

	if len(replacedVars) > 0 {
		t.logger.WithFields(logrus.Fields{
			"file_path": ctx.FilePath,
			"variables": strings.Join(replacedVars, ", "),
		}).Debug("Replaced template variables")
	}

	// Check for any remaining unreplaced variables and log warnings
	remainingVars := t.findUnreplacedVariables(result)
	if len(remainingVars) > 0 {
		t.logger.WithFields(logrus.Fields{
			"file_path":       ctx.FilePath,
			"unreplaced_vars": strings.Join(remainingVars, ", "),
			"available_vars":  strings.Join(varKeys, ", "),
		}).Warn("Found unreplaced template variables")
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
