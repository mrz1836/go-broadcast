package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid config file with mappings",
			setupFile: func(t *testing.T) string {
				content := `
version: 1
defaults:
  branch_prefix: "sync/custom"
  pr_labels: ["sync", "automated"]
mappings:
  - source:
      repo: "org/template-repo"
      branch: "master"
      id: "main"
    targets:
      - repo: "org/service-a"
        files:
          - src: "README.md"
            dest: "README.md"
`
				tmpFile := filepath.Join(testutil.CreateTempDir(t), "config.yaml")
				testutil.WriteTestFile(t, tmpFile, content)
				return tmpFile
			},
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1, cfg.Version)
				assert.Equal(t, "sync/custom", cfg.Defaults.BranchPrefix)
				assert.Equal(t, []string{"sync", "automated"}, cfg.Defaults.PRLabels)

				require.Len(t, cfg.Mappings, 1)
				assert.Equal(t, "org/template-repo", cfg.Mappings[0].Source.Repo)
				assert.Equal(t, "master", cfg.Mappings[0].Source.Branch)
				assert.Equal(t, "main", cfg.Mappings[0].Source.ID)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				assert.Equal(t, "org/service-a", cfg.Mappings[0].Targets[0].Repo)
			},
		},
		{
			name: "file not found",
			setupFile: func(_ *testing.T) string {
				return "/path/does/not/exist/config.yaml"
			},
			expectError: true,
			errorMsg:    "failed to open config file",
		},
		{
			name: "invalid YAML syntax",
			setupFile: func(t *testing.T) string {
				content := `
version: 1
mappings:
  - source:
      repo: "org/template-repo
`
				tmpFile := filepath.Join(testutil.CreateTempDir(t), "invalid.yaml")
				testutil.WriteTestFile(t, tmpFile, content)
				return tmpFile
			},
			expectError: true,
			errorMsg:    "failed to parse YAML",
		},
		{
			name: "empty file",
			setupFile: func(t *testing.T) string {
				tmpFile := filepath.Join(testutil.CreateTempDir(t), "empty.yaml")
				testutil.WriteTestFile(t, tmpFile, "")
				return tmpFile
			},
			expectError: true,
			errorMsg:    "failed to parse YAML: EOF",
		},
		{
			name: "permission denied",
			setupFile: func(t *testing.T) string {
				tmpFile := filepath.Join(testutil.CreateTempDir(t), "noperm.yaml")
				err := os.WriteFile(tmpFile, []byte("version: 1"), 0o000)
				require.NoError(t, err)
				return tmpFile
			},
			expectError: true,
			errorMsg:    "failed to open config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupFile(t)
			cfg, err := Load(configPath)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestLoadFromReader(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid basic config with mappings",
			input: `
version: 1
mappings:
  - source:
      repo: "org/source"
      branch: "main"
      id: "primary"
    targets:
      - repo: "org/target"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1, cfg.Version)
				require.Len(t, cfg.Mappings, 1)
				assert.Equal(t, "org/source", cfg.Mappings[0].Source.Repo)
				assert.Equal(t, "main", cfg.Mappings[0].Source.Branch)
				assert.Equal(t, "primary", cfg.Mappings[0].Source.ID)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				assert.Equal(t, "org/target", cfg.Mappings[0].Targets[0].Repo)
			},
		},
		{
			name: "complete config with all fields",
			input: `
version: 1
defaults:
  branch_prefix: "feature/sync"
  pr_labels: ["sync", "template", "automated"]
global:
  pr_labels: ["global-label"]
  pr_assignees: ["global-assignee"]
mappings:
  - source:
      repo: "org/template"
      branch: "develop"
      id: "templates"
    targets:
      - repo: "org/app1"
        files:
          - src: "template/config.yaml"
            dest: "app/config.yaml"
          - src: "README.md"
            dest: "docs/README.md"
        transform:
          repo_name: true
          variables:
            APP_NAME: "app1"
            ENV: "production"
      - repo: "org/app2"
        files:
          - src: "config.yaml"
            dest: "config.yaml"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1, cfg.Version)

				// Defaults
				assert.Equal(t, "feature/sync", cfg.Defaults.BranchPrefix)
				assert.Equal(t, []string{"sync", "template", "automated"}, cfg.Defaults.PRLabels)

				// Global
				assert.Equal(t, []string{"global-label"}, cfg.Global.PRLabels)
				assert.Equal(t, []string{"global-assignee"}, cfg.Global.PRAssignees)

				// Mappings
				require.Len(t, cfg.Mappings, 1)
				assert.Equal(t, "org/template", cfg.Mappings[0].Source.Repo)
				assert.Equal(t, "develop", cfg.Mappings[0].Source.Branch)
				assert.Equal(t, "templates", cfg.Mappings[0].Source.ID)

				// Targets
				require.Len(t, cfg.Mappings[0].Targets, 2)

				// First target
				assert.Equal(t, "org/app1", cfg.Mappings[0].Targets[0].Repo)
				require.Len(t, cfg.Mappings[0].Targets[0].Files, 2)
				assert.Equal(t, "template/config.yaml", cfg.Mappings[0].Targets[0].Files[0].Src)
				assert.Equal(t, "app/config.yaml", cfg.Mappings[0].Targets[0].Files[0].Dest)
				assert.True(t, cfg.Mappings[0].Targets[0].Transform.RepoName)
				assert.Equal(t, "app1", cfg.Mappings[0].Targets[0].Transform.Variables["APP_NAME"])
				assert.Equal(t, "production", cfg.Mappings[0].Targets[0].Transform.Variables["ENV"])

				// Second target
				assert.Equal(t, "org/app2", cfg.Mappings[0].Targets[1].Repo)
				require.Len(t, cfg.Mappings[0].Targets[1].Files, 1)
			},
		},
		{
			name: "multi-source configuration",
			input: `
version: 1
mappings:
  - source:
      repo: "org/source1"
      branch: "main"
      id: "source1"
    targets:
      - repo: "org/target1"
        files:
          - src: "file1.txt"
            dest: "file1.txt"
  - source:
      repo: "org/source2"
      branch: "develop"
      id: "source2"
    targets:
      - repo: "org/target2"
        files:
          - src: "file2.txt"
            dest: "file2.txt"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1, cfg.Version)
				require.Len(t, cfg.Mappings, 2)

				// First mapping
				assert.Equal(t, "org/source1", cfg.Mappings[0].Source.Repo)
				assert.Equal(t, "source1", cfg.Mappings[0].Source.ID)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				assert.Equal(t, "org/target1", cfg.Mappings[0].Targets[0].Repo)

				// Second mapping
				assert.Equal(t, "org/source2", cfg.Mappings[1].Source.Repo)
				assert.Equal(t, "source2", cfg.Mappings[1].Source.ID)
				require.Len(t, cfg.Mappings[1].Targets, 1)
				assert.Equal(t, "org/target2", cfg.Mappings[1].Targets[0].Repo)
			},
		},
		{
			name: "invalid YAML syntax",
			input: `
version: 1
mappings:
  - source:
      repo: "unclosed quote
`,
			expectError: true,
			errorMsg:    "failed to parse YAML",
		},
		{
			name: "unknown fields rejected",
			input: `
version: 1
mappings:
  - source:
      repo: "org/source"
      unknown_field: "value"
    targets:
      - repo: "org/target"
`,
			expectError: true,
			errorMsg:    "failed to parse YAML",
		},
		{
			name:        "empty reader",
			input:       "",
			expectError: true,
			errorMsg:    "failed to parse YAML: EOF",
		},
		{
			name: "transform config with variables",
			input: `
version: 1
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
        transform:
          repo_name: true
          variables:
            SERVICE: "api"
            PORT: "8080"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				transform := cfg.Mappings[0].Targets[0].Transform
				assert.True(t, transform.RepoName)
				assert.Equal(t, "api", transform.Variables["SERVICE"])
				assert.Equal(t, "8080", transform.Variables["PORT"])
			},
		},
		{
			name: "conflict resolution config",
			input: `
version: 1
conflict_resolution:
  strategy: "error"
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.NotNil(t, cfg.ConflictResolution)
				assert.Equal(t, "error", cfg.ConflictResolution.Strategy)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			cfg, err := LoadFromReader(reader)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestParserApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    *Config
		expected *Config
	}{
		{
			name:  "empty config gets all defaults",
			input: &Config{},
			expected: &Config{
				Defaults: DefaultConfig{
					BranchPrefix: "chore/sync-files",
					PRLabels:     []string{"automated-sync"},
				},
			},
		},
		{
			name: "partial config preserves existing values",
			input: &Config{
				Defaults: DefaultConfig{
					BranchPrefix: "custom/prefix",
				},
			},
			expected: &Config{
				Defaults: DefaultConfig{
					BranchPrefix: "custom/prefix",            // Not overwritten
					PRLabels:     []string{"automated-sync"}, // Default applied
				},
			},
		},
		{
			name: "existing PR labels not overwritten",
			input: &Config{
				Defaults: DefaultConfig{
					PRLabels: []string{"custom", "labels"},
				},
			},
			expected: &Config{
				Defaults: DefaultConfig{
					BranchPrefix: "chore/sync-files",
					PRLabels:     []string{"custom", "labels"}, // Not overwritten
				},
			},
		},
		{
			name: "empty PR labels gets default",
			input: &Config{
				Defaults: DefaultConfig{
					PRLabels: []string{},
				},
			},
			expected: &Config{
				Defaults: DefaultConfig{
					BranchPrefix: "chore/sync-files",
					PRLabels:     []string{"automated-sync"},
				},
			},
		},
		{
			name: "mappings get source branch defaults",
			input: &Config{
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							Repo: "org/source",
							// Branch not set
						},
					},
				},
			},
			expected: &Config{
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							Repo:   "org/source",
							Branch: "main", // Default applied
						},
					},
				},
				Defaults: DefaultConfig{
					BranchPrefix: "chore/sync-files",
					PRLabels:     []string{"automated-sync"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.input
			applyDefaults(cfg)

			// Test defaults
			assert.Equal(t, tt.expected.Defaults.BranchPrefix, cfg.Defaults.BranchPrefix)
			assert.Equal(t, tt.expected.Defaults.PRLabels, cfg.Defaults.PRLabels)

			// Test mappings if present
			if len(tt.expected.Mappings) > 0 {
				require.Len(t, cfg.Mappings, len(tt.expected.Mappings))
				for i, mapping := range tt.expected.Mappings {
					assert.Equal(t, mapping.Source.Branch, cfg.Mappings[i].Source.Branch)
				}
			}
		})
	}
}

var errTestRead = errors.New("read error")

func TestLoadFromReaderIOError(t *testing.T) {
	// Test with a reader that returns an error
	reader := &errorReader{err: errTestRead}

	cfg, err := LoadFromReader(reader)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
	assert.Nil(t, cfg)
}

func TestLoadFromReaderVariableTypes(t *testing.T) {
	// Since Transform.Variables is map[string]string, non-string values will be converted to strings
	input := `
version: 1
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
        transform:
          variables:
            STRING: "value"
            NUMBER: "42"
            PORT: "8080"
            ENABLED: "true"
`

	reader := strings.NewReader(input)
	cfg, err := LoadFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Mappings, 1)
	require.Len(t, cfg.Mappings[0].Targets, 1)
	vars := cfg.Mappings[0].Targets[0].Transform.Variables

	assert.Equal(t, "value", vars["STRING"])
	assert.Equal(t, "42", vars["NUMBER"])
	assert.Equal(t, "8080", vars["PORT"])
	assert.Equal(t, "true", vars["ENABLED"])
}

// TestLoadFromReader_TargetPRLabels tests parsing of target-level PR labels
func TestLoadFromReader_TargetPRLabels(t *testing.T) {
	t.Run("target with pr_labels overrides defaults", func(t *testing.T) {
		input := `
version: 1
defaults:
  pr_labels: ["default-label1", "default-label2"]
mappings:
  - source:
      repo: "org/source"
      branch: "main"
      id: "main"
    targets:
      - repo: "org/target1"
        pr_labels: ["custom-label1", "custom-label2"]
        files:
          - src: "file.txt"
            dest: "file.txt"
      - repo: "org/target2"
        files:
          - src: "file.txt"
            dest: "file.txt"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Verify defaults
		assert.Equal(t, []string{"default-label1", "default-label2"}, cfg.Defaults.PRLabels)

		// Verify targets
		require.Len(t, cfg.Mappings, 1)
		require.Len(t, cfg.Mappings[0].Targets, 2)

		// First target should have custom labels
		assert.Equal(t, "org/target1", cfg.Mappings[0].Targets[0].Repo)
		assert.Equal(t, []string{"custom-label1", "custom-label2"}, cfg.Mappings[0].Targets[0].PRLabels)

		// Second target should have no labels (will use defaults at runtime)
		assert.Equal(t, "org/target2", cfg.Mappings[0].Targets[1].Repo)
		assert.Nil(t, cfg.Mappings[0].Targets[1].PRLabels)
	})

	t.Run("target with empty pr_labels array", func(t *testing.T) {
		input := `
version: 1
defaults:
  pr_labels: ["default-label"]
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
        pr_labels: []
        files:
          - src: "file.txt"
            dest: "file.txt"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		// Target should have empty slice (not nil)
		require.Len(t, cfg.Mappings, 1)
		require.Len(t, cfg.Mappings[0].Targets, 1)
		assert.NotNil(t, cfg.Mappings[0].Targets[0].PRLabels)
		assert.Empty(t, cfg.Mappings[0].Targets[0].PRLabels)
	})

	t.Run("target with single pr_label", func(t *testing.T) {
		input := `
version: 1
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
        pr_labels: ["single-label"]
        files:
          - src: "file.txt"
            dest: "file.txt"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		require.Len(t, cfg.Mappings, 1)
		require.Len(t, cfg.Mappings[0].Targets, 1)
		assert.Equal(t, []string{"single-label"}, cfg.Mappings[0].Targets[0].PRLabels)
	})

	t.Run("target with all PR fields", func(t *testing.T) {
		input := `
version: 1
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
        pr_labels: ["label1", "label2"]
        pr_assignees: ["user1", "user2"]
        pr_reviewers: ["reviewer1"]
        pr_team_reviewers: ["team1", "team2"]
        files:
          - src: "file.txt"
            dest: "file.txt"
`

		reader := strings.NewReader(input)
		cfg, err := LoadFromReader(reader)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		require.Len(t, cfg.Mappings, 1)
		require.Len(t, cfg.Mappings[0].Targets, 1)
		target := cfg.Mappings[0].Targets[0]
		assert.Equal(t, []string{"label1", "label2"}, target.PRLabels)
		assert.Equal(t, []string{"user1", "user2"}, target.PRAssignees)
		assert.Equal(t, []string{"reviewer1"}, target.PRReviewers)
		assert.Equal(t, []string{"team1", "team2"}, target.PRTeamReviewers)
	})
}

// errorReader is a mock reader that always returns an error
type errorReader struct {
	err error
}

func (r *errorReader) Read(_ []byte) (n int, err error) {
	return 0, r.err
}

func TestParserEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, cfg *Config)
	}{
		{
			name: "deeply nested YAML structure",
			input: `
version: 1
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
        transform:
          variables:
            LEVEL1:
              LEVEL2:
                LEVEL3:
                  LEVEL4: "deep_value"
`,
			expectError: true,
			errorMsg:    "failed to parse YAML", // Variables expects map[string]string
		},
		{
			name: "very large number of targets",
			input: func() string {
				var sb strings.Builder
				sb.WriteString("version: 1\nmappings:\n  - source:\n      repo: \"org/source\"\n      id: \"main\"\n    targets:\n")
				for i := 0; i < 100; i++ {
					sb.WriteString(fmt.Sprintf("      - repo: \"org/target%d\"\n", i))
				}
				return sb.String()
			}(),
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				assert.Len(t, cfg.Mappings[0].Targets, 100)
			},
		},
		{
			name: "malformed YAML with tabs",
			input: `
version: 1
mappings:
  - source:
	repo: "org/source"  # Tab instead of spaces
    targets:
      - repo: "org/target"
`,
			expectError: true,
			errorMsg:    "failed to parse YAML",
		},
		{
			name: "YAML with duplicate keys",
			input: `
version: 1
mappings:
  - source:
      repo: "org/source"
      repo: "org/duplicate"
    targets:
      - repo: "org/target"
`,
			expectError: true, // Strict YAML parser rejects duplicate keys
			errorMsg:    "failed to parse YAML",
		},
		{
			name: "YAML with null values",
			input: `
version: 1
defaults:
  branch_prefix: null
  pr_labels: null
mappings:
  - source:
      repo: "org/source"
      branch: null
      id: "main"
    targets:
      - repo: "org/target"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				assert.Equal(t, "main", cfg.Mappings[0].Source.Branch) // Default applied
				assert.Equal(t, "chore/sync-files", cfg.Defaults.BranchPrefix)
				assert.Equal(t, []string{"automated-sync"}, cfg.Defaults.PRLabels)
			},
		},
		{
			name: "YAML with special characters in strings",
			input: `
version: 1
mappings:
  - source:
      repo: "org/source-with-special-@#$%"
      branch: "feature/test-&-deploy"
      id: "special"
    targets:
      - repo: "org/target!@#"
        transform:
          variables:
            SPECIAL: |
              value with spaces, tabs	and newlines
              and special chars: !@#$%^&*()
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				assert.Equal(t, "org/source-with-special-@#$%", cfg.Mappings[0].Source.Repo)
				assert.Equal(t, "feature/test-&-deploy", cfg.Mappings[0].Source.Branch)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				assert.Equal(t, "org/target!@#", cfg.Mappings[0].Targets[0].Repo)
				// Using | preserves newlines exactly
				assert.Contains(t, cfg.Mappings[0].Targets[0].Transform.Variables["SPECIAL"], "value with spaces, tabs\tand newlines\n")
				assert.Contains(t, cfg.Mappings[0].Targets[0].Transform.Variables["SPECIAL"], "and special chars: !@#$%^&*()")
			},
		},
		{
			name: "YAML with Unicode characters",
			//nolint:gosmopolitan // Testing Unicode support
			input: `
version: 1
mappings:
  - source:
      repo: "org/source-世界"
      id: "unicode"
    targets:
      - repo: "org/target-🚀"
        transform:
          variables:
            GREETING: "Hello 世界 🌍"
            EMOJI: "🎉🎊🎈"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				assert.Equal(t, "org/source-世界", cfg.Mappings[0].Source.Repo) //nolint:gosmopolitan // Testing Unicode support
				require.Len(t, cfg.Mappings[0].Targets, 1)
				assert.Equal(t, "org/target-🚀", cfg.Mappings[0].Targets[0].Repo)
				assert.Equal(t, "Hello 世界 🌍", cfg.Mappings[0].Targets[0].Transform.Variables["GREETING"]) //nolint:gosmopolitan // Testing Unicode support
				assert.Equal(t, "🎉🎊🎈", cfg.Mappings[0].Targets[0].Transform.Variables["EMOJI"])
			},
		},
		{
			name: "YAML with very long strings",
			input: func() string {
				longString := strings.Repeat("a", 10000)
				return fmt.Sprintf(`
version: 1
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
        transform:
          variables:
            LONG: "%s"
`, longString)
			}(),
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				assert.Len(t, cfg.Mappings[0].Targets[0].Transform.Variables["LONG"], 10000)
			},
		},
		{
			name: "YAML with anchors and aliases",
			input: `
version: 1
defaults: &defaults
  branch_prefix: "sync/files"
  pr_labels: ["automated"]
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target1"
        pr_labels: *defaults.pr_labels
      - repo: "org/target2"
        pr_labels: ["custom"]
`,
			expectError: true, // YAML doesn't support partial anchor references
			errorMsg:    "failed to parse YAML",
		},
		{
			name: "YAML with flow style",
			input: `
version: 1
defaults: {branch_prefix: "sync", pr_labels: ["auto", "sync"]}
mappings:
  - {source: {repo: "org/source", branch: "main", id: "main"}, targets: [{repo: "org/target", files: [{src: "a.txt", dest: "b.txt"}]}]}
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				assert.Equal(t, "org/source", cfg.Mappings[0].Source.Repo)
				assert.Equal(t, "main", cfg.Mappings[0].Source.Branch)
				assert.Equal(t, []string{"auto", "sync"}, cfg.Defaults.PRLabels)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				assert.Len(t, cfg.Mappings[0].Targets[0].Files, 1)
			},
		},
		{
			name: "YAML with comments everywhere",
			input: `
# Main config file
version: 1 # Version number
defaults: # Default settings
  branch_prefix: "sync" # Prefix for branches
  pr_labels: # Labels to add
    - "automated" # First label
    - "sync" # Second label
mappings: # Source to target mappings
  - source: # Source repository
      repo: "org/source" # Repository name
      branch: "main" # Branch to use
      id: "main" # Source ID
    targets: # Target repositories
      # First target
      - repo: "org/target1" # Target repo name
        files: # Files to sync
          - src: "file.txt" # Source file
            dest: "file.txt" # Destination
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1, cfg.Version)
				require.Len(t, cfg.Mappings, 1)
				assert.Equal(t, "org/source", cfg.Mappings[0].Source.Repo)
				require.Len(t, cfg.Mappings[0].Targets, 1)
			},
		},
		{
			name: "YAML with multiline strings",
			input: `
version: 1
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
        transform:
          variables:
            LITERAL: |
              This is a literal block scalar
              It preserves newlines
              And indentation
            FOLDED: >
              This is a folded block scalar
              It folds newlines into spaces
              Unless there's a blank line
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				literal := cfg.Mappings[0].Targets[0].Transform.Variables["LITERAL"]
				assert.Contains(t, literal, "It preserves newlines\n")

				folded := cfg.Mappings[0].Targets[0].Transform.Variables["FOLDED"]
				assert.Contains(t, folded, "It folds newlines into spaces")
				// Folded scalars keep final newline
				assert.True(t, strings.HasSuffix(folded, "\n"))
			},
		},
		{
			name: "empty mappings array",
			input: `
version: 1
mappings: []
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.NotNil(t, cfg.Mappings)
				assert.Empty(t, cfg.Mappings)
			},
		},
		{
			name: "version as string instead of int",
			input: `
version: "1"
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
`,
			expectError: true,
			errorMsg:    "failed to parse YAML",
		},
		{
			name: "missing required version field",
			input: `
mappings:
  - source:
      repo: "org/source"
      id: "main"
    targets:
      - repo: "org/target"
`,
			expectError: false, // Version will be 0
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 0, cfg.Version)
			},
		},
		{
			name: "circular reference attempt",
			input: `
version: 1
mappings:
  - source: &source
      repo: "org/source"
      branch: "main"
      id: "main"
    targets:
      - repo: "org/target"
        source: *source
`,
			expectError: true, // Unknown field 'source' in target
			errorMsg:    "failed to parse YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			cfg, err := LoadFromReader(reader)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestLoadSecurityEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError bool
		errorMsg    string
	}{
		{
			name: "symlink to sensitive file",
			setupFile: func(t *testing.T) string {
				tmpDir := testutil.CreateTempDir(t)
				configPath := filepath.Join(tmpDir, "config.yaml")
				sensitiveFile := filepath.Join(tmpDir, "sensitive.txt")

				// Create a sensitive file
				testutil.WriteTestFile(t, sensitiveFile, "sensitive data")

				// Create symlink
				err := os.Symlink(sensitiveFile, configPath)
				if err != nil {
					t.Skip("Cannot create symlinks on this system")
				}
				return configPath
			},
			expectError: true,
			errorMsg:    "failed to parse YAML",
		},
		{
			name: "directory instead of file",
			setupFile: func(t *testing.T) string {
				tmpDir := testutil.CreateTempDir(t)
				return tmpDir // Return directory path instead of file
			},
			expectError: true,
			errorMsg:    "failed to parse YAML", // os.Open succeeds but YAML parsing fails
		},
		{
			name: "file with null bytes",
			setupFile: func(t *testing.T) string {
				tmpFile := filepath.Join(testutil.CreateTempDir(t), "null.yaml")
				content := "version: 1\x00\nmappings:\n  - source:\n      repo: \"org/source\""
				testutil.WriteTestFile(t, tmpFile, content)
				return tmpFile
			},
			expectError: true,
			errorMsg:    "failed to parse YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupFile(t)
			cfg, err := Load(configPath)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
			}
		})
	}
}

func TestLoadFromReaderPanicRecovery(t *testing.T) {
	// Test that panics in YAML parsing are handled gracefully
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "recursive alias",
			input: `
version: 1
mappings:
  - source: &a
      repo: "org/source"
      extra: *a
    targets:
      - repo: "org/target"
`,
		},
		{
			name:  "malformed unicode",
			input: "version: 1\nmappings:\n  - source:\n      repo: \"org/source\xc3\x28\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)

			// Should not panic
			cfg, err := LoadFromReader(reader)

			// May or may not error depending on YAML parser behavior
			if err != nil {
				assert.Contains(t, err.Error(), "failed to parse YAML")
			} else {
				assert.NotNil(t, cfg)
			}
		})
	}
}
