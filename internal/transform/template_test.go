package transform

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestTemplateTransformer_Name(t *testing.T) {
	logger := logrus.New()
	transformer := NewTemplateTransformer(logger, nil)
	assert.Equal(t, "template-variable-replacer", transformer.Name())
}

func TestTemplateTransformer_Transform(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name        string
		content     string
		variables   map[string]string
		wantContent string
	}{
		{
			name:    "replace double brace variables",
			content: `Service: {{SERVICE_NAME}}\nPort: {{PORT}}`,
			variables: map[string]string{
				"SERVICE_NAME": "my-service",
				"PORT":         "8080",
			},
			wantContent: `Service: my-service\nPort: 8080`,
		},
		{
			name:    "replace dollar brace variables",
			content: `export SERVICE=${SERVICE_NAME}\nexport PORT=${PORT}`,
			variables: map[string]string{
				"SERVICE_NAME": "my-service",
				"PORT":         "8080",
			},
			wantContent: `export SERVICE=my-service\nexport PORT=8080`,
		},
		{
			name:    "mixed variable styles",
			content: `Name: {{SERVICE_NAME}}, Port: ${PORT}`,
			variables: map[string]string{
				"SERVICE_NAME": "my-service",
				"PORT":         "8080",
			},
			wantContent: `Name: my-service, Port: 8080`,
		},
		{
			name:        "no variables to replace",
			content:     `This has no variables`,
			variables:   map[string]string{},
			wantContent: `This has no variables`,
		},
		{
			name:    "variable not in map",
			content: `Service: {{SERVICE_NAME}}, Unknown: {{UNKNOWN_VAR}}`,
			variables: map[string]string{
				"SERVICE_NAME": "my-service",
			},
			wantContent: `Service: my-service, Unknown: {{UNKNOWN_VAR}}`,
		},
		{
			name:    "nested variable names",
			content: `{{SERVICE_NAME}} and {{SERVICE}}`,
			variables: map[string]string{
				"SERVICE_NAME": "my-service-name",
				"SERVICE":      "my-service",
			},
			wantContent: `my-service-name and my-service`,
		},
		{
			name:    "variables with special characters in values",
			content: `Path: {{PATH}}\nRegex: {{PATTERN}}`,
			variables: map[string]string{
				"PATH":    "/usr/local/bin",
				"PATTERN": "^[a-z]+$",
			},
			wantContent: `Path: /usr/local/bin\nRegex: ^[a-z]+$`,
		},
		{
			name:    "multiple occurrences of same variable",
			content: `{{APP}} is running. Check {{APP}} status. {{APP}} logs:`,
			variables: map[string]string{
				"APP": "my-app",
			},
			wantContent: `my-app is running. Check my-app status. my-app logs:`,
		},
		{
			name:    "empty variable value",
			content: `Prefix: {{PREFIX}}value`,
			variables: map[string]string{
				"PREFIX": "",
			},
			wantContent: `Prefix: value`,
		},
		{
			name:    "variable names with underscores and numbers",
			content: `{{VAR_1}} and {{VAR_2}} and {{LONG_VAR_NAME_123}}`,
			variables: map[string]string{
				"VAR_1":             "first",
				"VAR_2":             "second",
				"LONG_VAR_NAME_123": "third",
			},
			wantContent: `first and second and third`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := NewTemplateTransformer(logger, nil)
			ctx := Context{
				SourceRepo: "org/source",
				TargetRepo: "org/target",
				FilePath:   "test.txt",
				Variables:  tt.variables,
			}

			result, err := transformer.Transform([]byte(tt.content), ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantContent, string(result))
		})
	}
}

func TestTemplateTransformer_FindUnreplacedVariables(t *testing.T) {
	logger := logrus.New()
	transformer := NewTemplateTransformer(logger, nil).(*templateTransformer)

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "find double brace variables",
			content:  `{{VAR1}} and {{VAR2}}`,
			expected: []string{"VAR1", "VAR2"},
		},
		{
			name:     "find dollar brace variables",
			content:  `${VAR1} and ${VAR2}`,
			expected: []string{"VAR1", "VAR2"},
		},
		{
			name:     "mixed styles",
			content:  `{{VAR1}} and ${VAR2}}`,
			expected: []string{"VAR1", "VAR2"},
		},
		{
			name:     "duplicate variables",
			content:  `{{VAR1}} and {{VAR1}} and ${VAR1}`,
			expected: []string{"VAR1"},
		},
		{
			name:     "no variables",
			content:  `This has no variables`,
			expected: []string{},
		},
		{
			name:     "invalid variable names ignored",
			content:  `{{lower}} and {{123}} and {{VAR-NAME}}`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := transformer.findUnreplacedVariables(tt.content)

			// Sort for consistent comparison
			assert.ElementsMatch(t, tt.expected, vars)
		})
	}
}

func TestTemplateTransformer_LogsWarnings(t *testing.T) {
	// Create logger with hook to capture logs
	logger := logrus.New()
	hook := &testLogHook{}
	logger.AddHook(hook)
	logger.SetLevel(logrus.DebugLevel)

	transformer := NewTemplateTransformer(logger, nil)

	content := `Service: {{SERVICE_NAME}}, Missing: {{MISSING_VAR}}`
	ctx := Context{
		SourceRepo: "org/source",
		TargetRepo: "org/target",
		FilePath:   "config.yaml",
		Variables: map[string]string{
			"SERVICE_NAME": "my-service",
		},
	}

	result, err := transformer.Transform([]byte(content), ctx)
	require.NoError(t, err)
	assert.Equal(t, `Service: my-service, Missing: {{MISSING_VAR}}`, string(result))

	// Check that warning was logged
	found := false

	for _, entry := range hook.entries {
		if entry.Level == logrus.WarnLevel {
			assert.Contains(t, entry.Message, "unreplaced template variables")
			assert.Contains(t, entry.Data["unreplaced_vars"], "MISSING_VAR")

			found = true

			break
		}
	}

	assert.True(t, found, "Expected warning log for unreplaced variables")
}

// testLogHook captures log entries for testing
type testLogHook struct {
	entries []logrus.Entry
}

func (h *testLogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *testLogHook) Fire(entry *logrus.Entry) error {
	h.entries = append(h.entries, *entry)
	return nil
}

func TestTemplateTransformerEdgeCasesWithDebugLogging(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	hook := &testLogHook{}
	logger.AddHook(hook)

	logConfig := &logging.LogConfig{
		Debug: logging.DebugFlags{
			Transform: true,
		},
	}

	transformer := NewTemplateTransformer(logger, logConfig)

	tests := []struct {
		name            string
		content         string
		variables       map[string]string
		wantContent     string
		wantLogContains []string
	}{
		{
			name:    "debug logging for large content",
			content: strings.Repeat("a", 3000) + "{{VAR}}",
			variables: map[string]string{
				"VAR": "replaced",
			},
			wantContent:     strings.Repeat("a", 3000) + "replaced",
			wantLogContains: []string{"Starting template transformation", "Template variables replaced"},
		},
		{
			name:    "debug logging for empty variables",
			content: "{{VAR}}",
			variables: map[string]string{
				"VAR": "",
			},
			wantContent:     "",
			wantLogContains: []string{"Variable substitution"},
		},
		{
			name:            "no variables case with debug logging",
			content:         "No variables here",
			variables:       map[string]string{},
			wantContent:     "No variables here",
			wantLogContains: []string{"completed_no_variables"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook.entries = nil

			ctx := Context{
				SourceRepo: "org/source",
				TargetRepo: "org/target",
				FilePath:   "test.txt",
				Variables:  tt.variables,
			}

			result, err := transformer.Transform([]byte(tt.content), ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantContent, string(result))

			// Check that expected log messages were generated
			for _, expectedLog := range tt.wantLogContains {
				found := false
				for _, entry := range hook.entries {
					if strings.Contains(entry.Message, expectedLog) ||
						strings.Contains(formatLogData(entry.Data), expectedLog) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected log message not found: %s", expectedLog)
			}
		})
	}
}

func TestTemplateTransformerComplexVariablePatterns(t *testing.T) {
	logger := logrus.New()
	transformer := NewTemplateTransformer(logger, nil)

	tests := []struct {
		name        string
		content     string
		variables   map[string]string
		wantContent string
	}{
		{
			name:    "variables with special regex characters",
			content: `Pattern: {{REGEX_PATTERN}}, Path: {{FILE_PATH}}`,
			variables: map[string]string{
				"REGEX_PATTERN": `^[a-z]+\d*$`,
				"FILE_PATH":     `/usr/local/bin/script.sh`,
			},
			wantContent: `Pattern: ^[a-z]+\d*$, Path: /usr/local/bin/script.sh`,
		},
		{
			name:    "variables with dollar signs and backslashes",
			content: `Price: {{PRICE}}, Path: {{WINDOWS_PATH}}`,
			variables: map[string]string{
				"PRICE":        "$99.99",
				"WINDOWS_PATH": `C:\Users\Admin\Documents`,
			},
			wantContent: `Price: $99.99, Path: C:\Users\Admin\Documents`,
		},
		{
			name:    "variables with quotes and apostrophes",
			content: `Message: {{MESSAGE}}, Quote: {{QUOTE}}`,
			variables: map[string]string{
				"MESSAGE": `Don't forget to "escape" properly`,
				"QUOTE":   `She said, "Hello!"`,
			},
			wantContent: `Message: Don't forget to "escape" properly, Quote: She said, "Hello!"`,
		},
		{
			name:    "variables with newlines and tabs",
			content: `Content: {{MULTILINE}}, Tabbed: {{TABBED}}`,
			variables: map[string]string{
				"MULTILINE": "Line 1\nLine 2\nLine 3",
				"TABBED":    "Col1\tCol2\tCol3",
			},
			wantContent: `Content: Line 1
Line 2
Line 3, Tabbed: Col1	Col2	Col3`,
		},
		{
			name:    "variables with JSON content",
			content: `Config: {{JSON_CONFIG}}`,
			variables: map[string]string{
				"JSON_CONFIG": `{"key": "value", "nested": {"array": [1, 2, 3]}}`,
			},
			wantContent: `Config: {"key": "value", "nested": {"array": [1, 2, 3]}}`,
		},
		{
			name:    "variables with HTML/XML content",
			content: `Template: {{HTML_TEMPLATE}}`,
			variables: map[string]string{
				"HTML_TEMPLATE": `<div class="container"><p>Hello & welcome!</p></div>`,
			},
			wantContent: `Template: <div class="container"><p>Hello & welcome!</p></div>`,
		},
		{
			name:    "variables with Unicode characters",
			content: `Greeting: {{GREETING}}, Emoji: {{EMOJI}}`,
			variables: map[string]string{
				"GREETING": "Hello, 世界", //nolint:gosmopolitan // Testing Unicode support
				"EMOJI":    "🚀 Launch! 🎉",
			},
			wantContent: `Greeting: Hello, 世界, Emoji: 🚀 Launch! 🎉`, //nolint:gosmopolitan // Testing Unicode support
		},
		{
			name:    "extremely long variable names",
			content: `Value: {{VERY_LONG_VARIABLE_NAME_THAT_EXCEEDS_NORMAL_EXPECTATIONS_BUT_IS_STILL_VALID}}`,
			variables: map[string]string{
				"VERY_LONG_VARIABLE_NAME_THAT_EXCEEDS_NORMAL_EXPECTATIONS_BUT_IS_STILL_VALID": "works",
			},
			wantContent: `Value: works`,
		},
		{
			name:    "adjacent variables without separators",
			content: `{{VAR1}}{{VAR2}}{{VAR3}}`,
			variables: map[string]string{
				"VAR1": "A",
				"VAR2": "B",
				"VAR3": "C",
			},
			wantContent: `ABC`,
		},
		{
			name:    "nested-looking patterns that aren't actually nested",
			content: `{{OUTER_{{INNER}}_VAR}}`,
			variables: map[string]string{
				"INNER":              "TEST",
				"OUTER_{{INNER}_VAR": "not replaced",
				"OUTER_TEST_VAR":     "should not match",
			},
			wantContent: `{{OUTER_TEST_VAR}}`,
		},
		{
			name:    "mixed syntax with partial matches",
			content: `{{VAR}} ${VAR} {VAR} $VAR {{VAR}`,
			variables: map[string]string{
				"VAR": "X",
			},
			wantContent: `X X {VAR} $VAR {{VAR}`,
		},
		{
			name:    "variable replacement that creates new patterns",
			content: `{{PREFIX}}VAR}}`,
			variables: map[string]string{
				"PREFIX": "{{NEW_",
				"VAR":    "should_not_match",
			},
			wantContent: `{{NEW_VAR}}`,
		},
		{
			name:    "variables with numbers at different positions",
			content: `{{VAR1}} ${2VAR} {{VAR_3}} {{VAR_4_END}}`,
			variables: map[string]string{
				"VAR1":      "first",
				"2VAR":      "wont_match", // Variables starting with numbers are invalid
				"VAR_3":     "third",
				"VAR_4_END": "fourth",
			},
			wantContent: `first wont_match third fourth`, // ${2VAR} gets replaced literally without regex validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Context{
				SourceRepo: "org/source",
				TargetRepo: "org/target",
				FilePath:   "test.txt",
				Variables:  tt.variables,
			}

			result, err := transformer.Transform([]byte(tt.content), ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantContent, string(result))
		})
	}
}

func TestTemplateTransformerPerformanceEdgeCases(t *testing.T) {
	logger := logrus.New()
	transformer := NewTemplateTransformer(logger, nil)

	t.Run("many variables with overlapping names", func(t *testing.T) {
		// Create variables with overlapping names
		variables := make(map[string]string)
		content := ""
		expected := ""

		// Add variables like SERVICE, SERVICE_NAME, SERVICE_NAME_LONG
		for i := 1; i <= 10; i++ {
			varName := "VAR"
			for j := 1; j <= i; j++ {
				varName += "_PART"
			}
			variables[varName] = varName + "_VALUE"
			content += "{{" + varName + "}} "
			expected += varName + "_VALUE "
		}

		ctx := Context{
			SourceRepo: "org/source",
			TargetRepo: "org/target",
			FilePath:   "test.txt",
			Variables:  variables,
		}

		result, err := transformer.Transform([]byte(content), ctx)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(string(result)))
	})

	t.Run("large content with many replacements", func(t *testing.T) {
		// Generate content with many variable occurrences
		var contentBuilder strings.Builder
		var expectedBuilder strings.Builder

		for i := 0; i < 100; i++ {
			contentBuilder.WriteString("Line " + string(rune(i)) + ": {{VAR1}} and {{VAR2}} and {{VAR3}}\n")
			expectedBuilder.WriteString("Line " + string(rune(i)) + ": value1 and value2 and value3\n")
		}

		variables := map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
		}

		ctx := Context{
			SourceRepo: "org/source",
			TargetRepo: "org/target",
			FilePath:   "test.txt",
			Variables:  variables,
		}

		result, err := transformer.Transform([]byte(contentBuilder.String()), ctx)
		require.NoError(t, err)
		assert.Equal(t, expectedBuilder.String(), string(result))
	})
}

func TestTemplateTransformerRegressionCases(t *testing.T) {
	logger := logrus.New()
	transformer := NewTemplateTransformer(logger, nil)

	tests := []struct {
		name        string
		content     string
		variables   map[string]string
		wantContent string
	}{
		{
			name:    "variable value that looks like a variable pattern",
			content: `Replace: {{META_VAR}}`,
			variables: map[string]string{
				"META_VAR": "{{ANOTHER_VAR}}",
			},
			wantContent: `Replace: {{ANOTHER_VAR}}`,
		},
		{
			name:    "empty content with variables defined",
			content: ``,
			variables: map[string]string{
				"VAR": "value",
			},
			wantContent: ``,
		},
		{
			name:    "only whitespace content",
			content: "\n\t  \r\n",
			variables: map[string]string{
				"VAR": "value",
			},
			wantContent: "\n\t  \r\n",
		},
		{
			name:    "binary-looking content",
			content: "\x00\x01\x02{{VAR}}\x03\x04",
			variables: map[string]string{
				"VAR": "TEST",
			},
			wantContent: "\x00\x01\x02TEST\x03\x04",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Context{
				SourceRepo: "org/source",
				TargetRepo: "org/target",
				FilePath:   "test.txt",
				Variables:  tt.variables,
			}

			result, err := transformer.Transform([]byte(tt.content), ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantContent, string(result))
		})
	}
}

func TestFindUnreplacedVariablesEdgeCases(t *testing.T) {
	logger := logrus.New()
	transformer := NewTemplateTransformer(logger, nil).(*templateTransformer)

	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "variables at content boundaries",
			content:  `{{START_VAR}} middle {{END_VAR}}`,
			expected: []string{"START_VAR", "END_VAR"},
		},
		{
			name:     "malformed variable patterns",
			content:  `{{VALID}} {{ SPACES }} {{lower}} {{123STARTS_WITH_NUMBER}} {{-DASH}}`,
			expected: []string{"VALID"},
		},
		{
			name:     "very long variable names",
			content:  `{{A_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_LONG_VARIABLE_NAME_THAT_IS_STILL_VALID}}`,
			expected: []string{"A_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_VERY_LONG_VARIABLE_NAME_THAT_IS_STILL_VALID"},
		},
		{
			name:     "variables in comments and strings",
			content:  `// Comment with {{VAR_IN_COMMENT}}\nconst str = "String with {{VAR_IN_STRING}}"`,
			expected: []string{"VAR_IN_COMMENT", "VAR_IN_STRING"},
		},
		{
			name:     "escaped-looking patterns",
			content:  `\{{ESCAPED}} \\{{DOUBLE_ESCAPED}}`,
			expected: []string{"ESCAPED", "DOUBLE_ESCAPED"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := transformer.findUnreplacedVariables(tt.content)
			assert.ElementsMatch(t, tt.expected, vars)
		})
	}
}

// Helper function to format log data for comparison
func formatLogData(data logrus.Fields) string {
	parts := make([]string, 0, len(data))
	for k, v := range data {
		parts = append(parts, k+"="+formatValue(v))
	}
	return strings.Join(parts, " ")
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		return "value"
	}
}
