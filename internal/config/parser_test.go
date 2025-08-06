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
			name: "valid config file",
			setupFile: func(t *testing.T) string {
				content := `
version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: "org/template-repo"
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
				require.Len(t, cfg.Groups, 1)
				group := cfg.Groups[0]
				assert.Equal(t, "org/template-repo", group.Source.Repo)
				assert.Equal(t, "sync/custom", group.Defaults.BranchPrefix)
				assert.Equal(t, []string{"sync", "automated"}, group.Defaults.PRLabels)
				require.Len(t, group.Targets, 1)
				assert.Equal(t, "org/service-a", group.Targets[0].Repo)
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
groups:
  - name: "Default Group"
    id: "default"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1, cfg.Version)
				require.Len(t, cfg.Groups, 1)
				group := cfg.Groups[0]
				assert.Equal(t, "org/source", group.Source.Repo)
				assert.Equal(t, "main", group.Source.Branch) // Default applied
				require.Len(t, group.Targets, 1)
				assert.Equal(t, "org/target", group.Targets[0].Repo)
			},
		},
		{
			name: "complete config with all fields",
			input: `
version: 2
groups:
  - name: "Complete Group"
    id: "complete"
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
				require.Len(t, cfg.Groups, 1)
				group := cfg.Groups[0]
				assert.Equal(t, "org/template", group.Source.Repo)
				assert.Equal(t, "develop", group.Source.Branch)

				// Defaults
				assert.Equal(t, "feature/sync", group.Defaults.BranchPrefix)
				assert.Equal(t, []string{"sync", "template", "automated"}, group.Defaults.PRLabels)

				// Targets
				require.Len(t, group.Targets, 2)

				// First target
				assert.Equal(t, "org/app1", group.Targets[0].Repo)
				require.Len(t, group.Targets[0].Files, 2)
				assert.Equal(t, "template/config.yaml", group.Targets[0].Files[0].Src)
				assert.Equal(t, "app/config.yaml", group.Targets[0].Files[0].Dest)
				assert.True(t, group.Targets[0].Transform.RepoName)
				assert.Equal(t, "app1", group.Targets[0].Transform.Variables["APP_NAME"])
				assert.Equal(t, "production", group.Targets[0].Transform.Variables["ENV"])

				// Second target
				assert.Equal(t, "org/app2", group.Targets[1].Repo)
				require.Len(t, group.Targets[1].Files, 1)
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
groups:
  - name: "test-group"
    id: "test-group-1"
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
				require.Len(t, cfg.Groups, 1)
				require.Len(t, cfg.Groups[0].Targets, 1)
				transform := cfg.Groups[0].Targets[0].Transform
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
				Groups: []Group{
					{
						Source: SourceConfig{
							Branch: "main",
						},
						Defaults: DefaultConfig{
							BranchPrefix: "chore/sync-files",
							PRLabels:     []string{"automated-sync"},
						},
					},
				},
			},
		},
		{
			name: "partial config preserves existing values",
			input: &Config{
				Groups: []Group{
					{
						Source: SourceConfig{
							Repo: "org/repo",
						},
						Defaults: DefaultConfig{
							BranchPrefix: "custom/prefix",
						},
					},
				},
			},
			expected: &Config{
				Groups: []Group{
					{
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
			},
		},
		{
			name: "existing PR labels not overwritten",
			input: &Config{
				Groups: []Group{
					{
						Defaults: DefaultConfig{
							PRLabels: []string{"custom", "labels"},
						},
					},
				},
			},
			expected: &Config{
				Groups: []Group{
					{
						Source: SourceConfig{
							Branch: "main",
						},
						Defaults: DefaultConfig{
							BranchPrefix: "chore/sync-files",
							PRLabels:     []string{"custom", "labels"}, // Not overwritten
						},
					},
				},
			},
		},
		{
			name: "empty PR labels gets default",
			input: &Config{
				Groups: []Group{
					{
						Defaults: DefaultConfig{
							PRLabels: []string{},
						},
					},
				},
			},
			expected: &Config{
				Groups: []Group{
					{
						Source: SourceConfig{
							Branch: "main",
						},
						Defaults: DefaultConfig{
							BranchPrefix: "chore/sync-files",
							PRLabels:     []string{"automated-sync"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.input
			applyDefaults(cfg)

			// Compare first group in both expected and actual
			if len(tt.expected.Groups) > 0 && len(cfg.Groups) > 0 {
				assert.Equal(t, tt.expected.Groups[0].Source.Branch, cfg.Groups[0].Source.Branch)
				assert.Equal(t, tt.expected.Groups[0].Defaults.BranchPrefix, cfg.Groups[0].Defaults.BranchPrefix)
				assert.Equal(t, tt.expected.Groups[0].Defaults.PRLabels, cfg.Groups[0].Defaults.PRLabels)
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
groups:
  - name: "Variable Test Group"
    id: "var-test"
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

	require.Len(t, cfg.Groups, 1)
	require.Len(t, cfg.Groups[0].Targets, 1)
	vars := cfg.Groups[0].Targets[0].Transform.Variables

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
groups:
  - name: "test-group"
    id: "test-group-1"
    source:
      repo: "org/source"
      branch: Final
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

		// Verify defaults in first group
		require.Len(t, cfg.Groups, 1)
		group := cfg.Groups[0]
		assert.Equal(t, []string{"default-label1", "default-label2"}, group.Defaults.PRLabels)

		// Verify targets
		require.Len(t, group.Targets, 2)

		// First target should have custom labels
		assert.Equal(t, "org/target1", group.Targets[0].Repo)
		assert.Equal(t, []string{"custom-label1", "custom-label2"}, group.Targets[0].PRLabels)

		// Second target should have no labels (will use defaults at runtime)
		assert.Equal(t, "org/target2", group.Targets[1].Repo)
		assert.Nil(t, group.Targets[1].PRLabels)
	})

	t.Run("target with empty pr_labels array", func(t *testing.T) {
		input := `
version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
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
		require.Len(t, cfg.Groups, 1)
		require.Len(t, cfg.Groups[0].Targets, 1)
		assert.NotNil(t, cfg.Groups[0].Targets[0].PRLabels)
		assert.Empty(t, cfg.Groups[0].Targets[0].PRLabels)
	})

	t.Run("target with single pr_label", func(t *testing.T) {
		input := `
version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
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

		require.Len(t, cfg.Groups, 1)
		require.Len(t, cfg.Groups[0].Targets, 1)
		assert.Equal(t, []string{"single-label"}, cfg.Groups[0].Targets[0].PRLabels)
	})

	t.Run("target with all PR fields", func(t *testing.T) {
		input := `
version: 1
groups:
  - name: "test-group"
    id: "test"
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

		require.Len(t, cfg.Groups, 1)
		require.Len(t, cfg.Groups[0].Targets, 1)
		target := cfg.Groups[0].Targets[0]
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
groups:
  - name: "test-group"
    id: "test"
    source:
      repo: "org/source"
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
				sb.WriteString("version: 1\ngroups:\n  - name: \"test-group\"\n    id: \"test\"\n    source:\n      repo: \"org/source\"\n    targets:\n")
				for i := 0; i < 100; i++ {
					sb.WriteString(fmt.Sprintf("      - repo: \"org/target%d\"\n", i))
				}
				return sb.String()
			}(),
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Groups, 1)
				assert.Len(t, cfg.Groups[0].Targets, 100)
			},
		},
		{
			name: "malformed YAML with tabs",
			input: `
version: 1
source:
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
groups:
  - name: "test-group"
    id: "test"
    source:
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
groups:
  - name: "test-group"
    id: "test"
    source:
      repo: "org/source"
      branch: null
    defaults:
      branch_prefix: null
      pr_labels: null
    targets:
      - repo: "org/target"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Groups, 1)
				assert.Equal(t, "main", cfg.Groups[0].Source.Branch) // Default applied
				assert.Equal(t, "chore/sync-files", cfg.Groups[0].Defaults.BranchPrefix)
				assert.Equal(t, []string{"automated-sync"}, cfg.Groups[0].Defaults.PRLabels)
			},
		},
		{
			name: "YAML with special characters in strings",
			input: `
version: 1
groups:
  - name: "test-group"
    id: "test"
    source:
      repo: "org/source-with-special-@#$%"
      branch: "feature/test-&-deploy"
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
				require.Len(t, cfg.Groups, 1)
				require.Len(t, cfg.Groups[0].Targets, 1)
				assert.Equal(t, "org/source-with-special-@#$%", cfg.Groups[0].Source.Repo)
				assert.Equal(t, "feature/test-&-deploy", cfg.Groups[0].Source.Branch)
				assert.Equal(t, "org/target!@#", cfg.Groups[0].Targets[0].Repo)
				// Using | preserves newlines exactly
				assert.Contains(t, cfg.Groups[0].Targets[0].Transform.Variables["SPECIAL"], "value with spaces, tabs\tand newlines\n")
				assert.Contains(t, cfg.Groups[0].Targets[0].Transform.Variables["SPECIAL"], "and special chars: !@#$%^&*()")
			},
		},
		{
			name: "YAML with Unicode characters",
			//nolint:gosmopolitan // Testing Unicode support
			input: `
version: 1
groups:
  - name: "test-group"
    id: "test"
    source:
      repo: "org/source-世界"
    targets:
      - repo: "org/target-🚀"
        transform:
          variables:
            GREETING: "Hello 世界 🌍"
            EMOJI: "🎉🎊🎈"
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Groups, 1)
				require.Len(t, cfg.Groups[0].Targets, 1)
				assert.Equal(t, "org/source-世界", cfg.Groups[0].Source.Repo) //nolint:gosmopolitan // Testing Unicode support
				assert.Equal(t, "org/target-🚀", cfg.Groups[0].Targets[0].Repo)
				assert.Equal(t, "Hello 世界 🌍", cfg.Groups[0].Targets[0].Transform.Variables["GREETING"]) //nolint:gosmopolitan // Testing Unicode support
				assert.Equal(t, "🎉🎊🎈", cfg.Groups[0].Targets[0].Transform.Variables["EMOJI"])
			},
		},
		{
			name: "YAML with very long strings",
			input: func() string {
				longString := strings.Repeat("a", 10000)
				return fmt.Sprintf(`
version: 1
groups:
  - name: "test-group"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
        transform:
          variables:
            LONG: "%s"
`, longString)
			}(),
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Groups, 1)
				require.Len(t, cfg.Groups[0].Targets, 1)
				assert.Len(t, cfg.Groups[0].Targets[0].Transform.Variables["LONG"], 10000)
			},
		},
		{
			name: "YAML with anchors and aliases",
			input: `
version: 1
groups:
  - name: "test-group"
    id: "test"
    source:
      repo: "org/source"
    defaults: &defaults
      branch_prefix: "sync/files"
      pr_labels: ["automated"]
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
groups:
  - name: "test-group"
    id: "test"
    source: {repo: "org/source", branch: Final}
    defaults: {branch_prefix: "sync", pr_labels: ["auto", "sync"]}
    targets:
      - {repo: "org/target", files: [{src: "a.txt", dest: "b.txt"}]}
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Groups, 1)
				require.Len(t, cfg.Groups[0].Targets, 1)
				assert.Equal(t, "org/source", cfg.Groups[0].Source.Repo)
				assert.Equal(t, "Final", cfg.Groups[0].Source.Branch)
				assert.Equal(t, []string{"auto", "sync"}, cfg.Groups[0].Defaults.PRLabels)
				assert.Len(t, cfg.Groups[0].Targets[0].Files, 1)
			},
		},
		{
			name: "YAML with comments everywhere",
			input: `
# Main config file
version: 1 # Version number
groups:
  - name: "test-group"
    id: "test"
    source: # Source repository
      repo: "org/source" # Repository name
      branch: Final # Branch to use
    # Default settings
    defaults:
      branch_prefix: "sync" # Prefix for branches
      pr_labels: # Labels to add
        - "automated" # First label
        - "sync" # Second label
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
				require.Len(t, cfg.Groups, 1)
				assert.Equal(t, "org/source", cfg.Groups[0].Source.Repo)
				assert.Len(t, cfg.Groups[0].Targets, 1)
			},
		},
		{
			name: "YAML with multiline strings",
			input: `
version: 1
groups:
  - name: "test-group"
    id: "test"
    source:
      repo: "org/source"
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
				require.Len(t, cfg.Groups, 1)
				require.Len(t, cfg.Groups[0].Targets, 1)
				literal := cfg.Groups[0].Targets[0].Transform.Variables["LITERAL"]
				assert.Contains(t, literal, "It preserves newlines\n")

				folded := cfg.Groups[0].Targets[0].Transform.Variables["FOLDED"]
				assert.Contains(t, folded, "It folds newlines into spaces")
				// Folded scalars keep final newline
				assert.True(t, strings.HasSuffix(folded, "\n"))
			},
		},
		{
			name: "empty targets array",
			input: `
version: 1
groups:
  - name: "test-group"
    id: "test"
    source:
      repo: "org/source"
    targets: []
`,
			expectError: false,
			validate: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Groups, 1)
				assert.NotNil(t, cfg.Groups[0].Targets)
				assert.Empty(t, cfg.Groups[0].Targets)
			},
		},
		{
			name: "version as string instead of int",
			input: `
version: "1"
groups:
  - name: "test-group"
    id: "test"
    source:
      repo: "org/source"
    targets:
      - repo: "org/target"
`,
			expectError: true,
			errorMsg:    "failed to parse YAML",
		},
		{
			name: "missing required version field",
			input: `
groups:
  - name: "test-group"
    id: "test"
    source:
      repo: "org/source"
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
groups:
  - name: "test-group"
    id: "test"
    source: &source
      repo: "org/source"
      branch: Final
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
				content := "version: 1\x00\nsource:\n  repo: \"org/source\""
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
source: &a
  repo: "org/source"
  extra: *a
targets:
  - repo: "org/target"
`,
		},
		{
			name:  "malformed unicode",
			input: "version: 1\nsource:\n  repo: \"org/source\xc3\x28\"\n",
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
