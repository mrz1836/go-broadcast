package transform

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateTransformer_Name(t *testing.T) {
	logger := logrus.New()
	transformer := NewTemplateTransformer(logger)
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
			transformer := NewTemplateTransformer(logger)
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
	transformer := NewTemplateTransformer(logger).(*templateTransformer)

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

	transformer := NewTemplateTransformer(logger)

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
