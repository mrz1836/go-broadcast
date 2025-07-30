package config

import (
	"errors"
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
			name: "valid config file",
			setupFile: func(t *testing.T) string {
				content := `
version: 1
source:
  repo: "org/template-repo"
  branch: "main"
defaults:
  branch_prefix: "sync/custom"
  pr_labels: ["sync", "automated"]
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
				assert.Equal(t, "org/template-repo", cfg.Source.Repo)
				assert.Equal(t, "main", cfg.Source.Branch)
				assert.Equal(t, "sync/custom", cfg.Defaults.BranchPrefix)
				assert.Equal(t, []string{"sync", "automated"}, cfg.Defaults.PRLabels)
				require.Len(t, cfg.Targets, 1)
				assert.Equal(t, "org/service-a", cfg.Targets[0].Repo)
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
source:
  repo: "org/template-repo
  branch: "main"
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
			name: "valid basic config",
			input: `
version: 1
source:
  repo: "org/source"
targets:
  - repo: "org/target"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1, cfg.Version)
				assert.Equal(t, "org/source", cfg.Source.Repo)
				assert.Equal(t, "master", cfg.Source.Branch) // Default applied
				require.Len(t, cfg.Targets, 1)
				assert.Equal(t, "org/target", cfg.Targets[0].Repo)
			},
		},
		{
			name: "complete config with all fields",
			input: `
version: 2
source:
  repo: "org/template"
  branch: "develop"
defaults:
  branch_prefix: "feature/sync"
  pr_labels: ["sync", "template", "automated"]
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
				assert.Equal(t, 2, cfg.Version)
				assert.Equal(t, "org/template", cfg.Source.Repo)
				assert.Equal(t, "develop", cfg.Source.Branch)

				// Defaults
				assert.Equal(t, "feature/sync", cfg.Defaults.BranchPrefix)
				assert.Equal(t, []string{"sync", "template", "automated"}, cfg.Defaults.PRLabels)

				// Targets
				require.Len(t, cfg.Targets, 2)

				// First target
				assert.Equal(t, "org/app1", cfg.Targets[0].Repo)
				require.Len(t, cfg.Targets[0].Files, 2)
				assert.Equal(t, "template/config.yaml", cfg.Targets[0].Files[0].Src)
				assert.Equal(t, "app/config.yaml", cfg.Targets[0].Files[0].Dest)
				assert.True(t, cfg.Targets[0].Transform.RepoName)
				assert.Equal(t, "app1", cfg.Targets[0].Transform.Variables["APP_NAME"])
				assert.Equal(t, "production", cfg.Targets[0].Transform.Variables["ENV"])

				// Second target
				assert.Equal(t, "org/app2", cfg.Targets[1].Repo)
				require.Len(t, cfg.Targets[1].Files, 1)
			},
		},
		{
			name: "invalid YAML syntax",
			input: `
version: 1
source:
  repo: "unclosed quote
`,
			expectError: true,
			errorMsg:    "failed to parse YAML",
		},
		{
			name: "unknown fields rejected",
			input: `
version: 1
source:
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
source:
  repo: "org/source"
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
				require.Len(t, cfg.Targets, 1)
				transform := cfg.Targets[0].Transform
				assert.True(t, transform.RepoName)
				assert.Equal(t, "api", transform.Variables["SERVICE"])
				assert.Equal(t, "8080", transform.Variables["PORT"])
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
				Source: SourceConfig{
					Branch: "master",
				},
				Defaults: DefaultConfig{
					BranchPrefix: "sync/template",
					PRLabels:     []string{"automated-sync"},
				},
			},
		},
		{
			name: "partial config preserves existing values",
			input: &Config{
				Source: SourceConfig{
					Repo:   "org/repo",
					Branch: "main",
				},
				Defaults: DefaultConfig{
					BranchPrefix: "custom/prefix",
				},
			},
			expected: &Config{
				Source: SourceConfig{
					Repo:   "org/repo",
					Branch: "main", // Not overwritten
				},
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
				Source: SourceConfig{
					Branch: "master",
				},
				Defaults: DefaultConfig{
					BranchPrefix: "sync/template",
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
				Source: SourceConfig{
					Branch: "master",
				},
				Defaults: DefaultConfig{
					BranchPrefix: "sync/template",
					PRLabels:     []string{"automated-sync"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.input
			applyDefaults(cfg)

			assert.Equal(t, tt.expected.Source.Branch, cfg.Source.Branch)
			assert.Equal(t, tt.expected.Defaults.BranchPrefix, cfg.Defaults.BranchPrefix)
			assert.Equal(t, tt.expected.Defaults.PRLabels, cfg.Defaults.PRLabels)
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
source:
  repo: "org/source"
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

	require.Len(t, cfg.Targets, 1)
	vars := cfg.Targets[0].Transform.Variables

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
source:
  repo: "org/source"
  branch: "main"
defaults:
  pr_labels: ["default-label1", "default-label2"]
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
		require.Len(t, cfg.Targets, 2)

		// First target should have custom labels
		assert.Equal(t, "org/target1", cfg.Targets[0].Repo)
		assert.Equal(t, []string{"custom-label1", "custom-label2"}, cfg.Targets[0].PRLabels)

		// Second target should have no labels (will use defaults at runtime)
		assert.Equal(t, "org/target2", cfg.Targets[1].Repo)
		assert.Nil(t, cfg.Targets[1].PRLabels)
	})

	t.Run("target with empty pr_labels array", func(t *testing.T) {
		input := `
version: 1
source:
  repo: "org/source"
defaults:
  pr_labels: ["default-label"]
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
		require.Len(t, cfg.Targets, 1)
		assert.NotNil(t, cfg.Targets[0].PRLabels)
		assert.Empty(t, cfg.Targets[0].PRLabels)
	})

	t.Run("target with single pr_label", func(t *testing.T) {
		input := `
version: 1
source:
  repo: "org/source"
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

		require.Len(t, cfg.Targets, 1)
		assert.Equal(t, []string{"single-label"}, cfg.Targets[0].PRLabels)
	})

	t.Run("target with all PR fields", func(t *testing.T) {
		input := `
version: 1
source:
  repo: "org/source"
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

		require.Len(t, cfg.Targets, 1)
		target := cfg.Targets[0]
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
