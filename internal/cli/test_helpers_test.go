package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestTestValidConfig(t *testing.T) {
	t.Run("valid YAML structure", func(t *testing.T) {
		require.NotEmpty(t, TestValidConfig, "TestValidConfig should not be empty")

		// Test that it's valid YAML
		var config map[string]interface{}
		err := yaml.Unmarshal([]byte(TestValidConfig), &config)
		require.NoError(t, err, "TestValidConfig should be valid YAML")
	})

	t.Run("contains required fields", func(t *testing.T) {
		assert.Contains(t, TestValidConfig, "version:", "Should contain version field")
		assert.Contains(t, TestValidConfig, "source:", "Should contain source field")
		assert.Contains(t, TestValidConfig, "targets:", "Should contain targets field")
	})

	t.Run("version field validation", func(t *testing.T) {
		assert.Contains(t, TestValidConfig, "version: 1", "Should specify version 1")
	})

	t.Run("source configuration", func(t *testing.T) {
		assert.Contains(t, TestValidConfig, "repo: org/template", "Should contain source repo")
		assert.Contains(t, TestValidConfig, "branch: main", "Should contain source branch")
	})

	t.Run("target configuration", func(t *testing.T) {
		assert.Contains(t, TestValidConfig, "repo: org/target1", "Should contain target repo")
		assert.Contains(t, TestValidConfig, "files:", "Should contain files section")
		assert.Contains(t, TestValidConfig, "src: README.md", "Should contain source file")
		assert.Contains(t, TestValidConfig, "dest: README.md", "Should contain destination file")
	})

	t.Run("YAML parsing validation", func(t *testing.T) {
		// Parse the configuration and validate structure
		var config struct {
			Version int `yaml:"version"`
			Source  struct {
				Repo   string `yaml:"repo"`
				Branch string `yaml:"branch"`
			} `yaml:"source"`
			Targets []struct {
				Repo  string `yaml:"repo"`
				Files []struct {
					Src  string `yaml:"src"`
					Dest string `yaml:"dest"`
				} `yaml:"files"`
			} `yaml:"targets"`
		}

		err := yaml.Unmarshal([]byte(TestValidConfig), &config)
		require.NoError(t, err, "Should parse as expected structure")

		// Validate parsed values
		assert.Equal(t, 1, config.Version, "Version should be 1")
		assert.Equal(t, "org/template", config.Source.Repo, "Source repo should be org/template")
		assert.Equal(t, "master", config.Source.Branch, "Source branch should be main")

		require.Len(t, config.Targets, 1, "Should have one target")
		assert.Equal(t, "org/target1", config.Targets[0].Repo, "Target repo should be org/target1")

		require.Len(t, config.Targets[0].Files, 1, "Should have one file mapping")
		assert.Equal(t, "README.md", config.Targets[0].Files[0].Src, "Source file should be README.md")
		assert.Equal(t, "README.md", config.Targets[0].Files[0].Dest, "Destination file should be README.md")
	})

	t.Run("indentation consistency", func(t *testing.T) {
		lines := strings.Split(TestValidConfig, "\n")

		// Check that indentation is consistent (using spaces, not tabs)
		for i, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue // Skip empty lines
			}

			// Count leading spaces
			leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))

			// Ensure indentation is in multiples of 2 (common YAML convention)
			if leadingSpaces > 0 {
				assert.Equal(t, 0, leadingSpaces%2, "Line %d should have even number of leading spaces for consistent indentation", i+1)
			}

			// Ensure no tabs are used
			assert.NotContains(t, line, "\t", "Line %d should not contain tabs", i+1)
		}
	})

	t.Run("realistic configuration values", func(t *testing.T) {
		// Check that the configuration uses realistic values that would work in tests
		assert.Contains(t, TestValidConfig, "org/", "Should use realistic organization structure")
		assert.Contains(t, TestValidConfig, "README.md", "Should use common file types")
		assert.Contains(t, TestValidConfig, "master", "Should use common branch name")
	})

	t.Run("minimal but complete configuration", func(t *testing.T) {
		// Ensure the config is minimal but has all required fields for testing
		configLines := strings.Split(strings.TrimSpace(TestValidConfig), "\n")

		// Should be concise (not too many lines for a test helper)
		assert.LessOrEqual(t, len(configLines), 20, "Test config should be concise")
		assert.GreaterOrEqual(t, len(configLines), 8, "Test config should have minimum required structure")

		// Should not contain comments (keep it simple for testing)
		for _, line := range configLines {
			assert.NotContains(t, line, "#", "Test config should not contain comments for simplicity")
		}
	})

	t.Run("configuration usefulness in tests", func(t *testing.T) {
		// Ensure the configuration would be useful for common test scenarios

		// Should have different source and target repos (useful for transformation tests)
		assert.Contains(t, TestValidConfig, "org/template", "Should have template source")
		assert.Contains(t, TestValidConfig, "org/target1", "Should have different target")

		// Should have file mapping (useful for file operation tests)
		assert.Contains(t, TestValidConfig, "src:", "Should have source file mapping")
		assert.Contains(t, TestValidConfig, "dest:", "Should have destination file mapping")

		// Should use version 1 (current version)
		assert.Contains(t, TestValidConfig, "version: 1", "Should use current version")
	})

	t.Run("string constant properties", func(t *testing.T) {
		// Verify it's properly defined as a string constant
		assert.IsType(t, "", TestValidConfig, "TestValidConfig should be a string")

		// Should start and end cleanly (no extraneous whitespace)
		trimmed := strings.TrimSpace(TestValidConfig)
		assert.Equal(t, TestValidConfig, trimmed, "TestValidConfig should not have leading/trailing whitespace")

		// Should start with version field
		assert.True(t, strings.HasPrefix(TestValidConfig, "version:"), "Should start with version field")
	})
}

func TestTestValidConfigUsability(t *testing.T) {
	t.Run("can be used in table-driven tests", func(t *testing.T) {
		// Simulate how this would be used in other tests
		testCases := []struct {
			name   string
			config string
			valid  bool
		}{
			{
				name:   "valid test config",
				config: TestValidConfig,
				valid:  true,
			},
			{
				name:   "empty config",
				config: "",
				valid:  false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.valid {
					var config map[string]interface{}
					err := yaml.Unmarshal([]byte(tc.config), &config)
					require.NoError(t, err, "Should parse successfully")
					assert.NotEmpty(t, config, "Should not be empty")
				} else {
					// Handle empty config case
					if tc.config == "" {
						assert.Empty(t, tc.config, "Empty config should be empty")
					}
				}
			})
		}
	})

	t.Run("can be modified for test variations", func(t *testing.T) {
		// Test that the base config can be easily modified for different test scenarios

		// Replace target repo for different test scenarios
		modifiedConfig := strings.ReplaceAll(TestValidConfig, "org/target1", "org/different-target")
		assert.Contains(t, modifiedConfig, "org/different-target", "Should be modifiable")
		assert.NotContains(t, modifiedConfig, "org/target1", "Original value should be replaced")

		// Replace file names for different test scenarios
		modifiedConfig = strings.ReplaceAll(TestValidConfig, "README.md", "config.yaml")
		assert.Contains(t, modifiedConfig, "config.yaml", "Should be modifiable")

		// Still should be valid YAML after modifications
		var config map[string]interface{}
		err := yaml.Unmarshal([]byte(modifiedConfig), &config)
		assert.NoError(t, err, "Modified config should still be valid YAML")
	})
}

func TestTestValidConfigBestPractices(t *testing.T) {
	t.Run("follows YAML best practices", func(t *testing.T) {
		lines := strings.Split(TestValidConfig, "\n")

		for i, line := range lines {
			// Skip empty lines
			if strings.TrimSpace(line) == "" {
				continue
			}

			// Check for trailing spaces (not ideal in YAML)
			assert.Equal(t, strings.TrimRight(line, " "), line, "Line %d should not have trailing spaces", i+1)

			// Ensure proper key-value separation
			if strings.Contains(line, ":") && !strings.HasSuffix(strings.TrimSpace(line), ":") {
				// Lines with values should have space after colon
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
					assert.True(t, strings.HasPrefix(parts[1], " "), "Line %d should have space after colon", i+1)
				}
			}
		}
	})

	t.Run("uses consistent naming conventions", func(t *testing.T) {
		// Check that repo names follow consistent patterns
		assert.Contains(t, TestValidConfig, "org/template", "Should use consistent org prefix")
		assert.Contains(t, TestValidConfig, "org/target1", "Should use consistent org prefix")

		// Check that field names are lowercase
		assert.Contains(t, TestValidConfig, "version:", "Field names should be lowercase")
		assert.Contains(t, TestValidConfig, "source:", "Field names should be lowercase")
		assert.Contains(t, TestValidConfig, "targets:", "Field names should be lowercase")
		assert.NotContains(t, TestValidConfig, "Version:", "Should not use capitalized field names")
		assert.NotContains(t, TestValidConfig, "Source:", "Should not use capitalized field names")
	})
}
