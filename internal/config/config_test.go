// Package config provides configuration parsing and validation for go-broadcast
package config

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromReader_ValidConfig(t *testing.T) {
	yamlContent := `
version: 1
defaults:
  branch_prefix: "chore/sync-files"
  pr_labels: ["automated-sync", "template"]
mappings:
  - source:
      repo: "org/template-repo"
      branch: "master"
      id: "template"
    targets:
      - repo: "org/service-a"
        files:
          - src: ".github/workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
        transform:
          repo_name: true
          variables:
            SERVICE: "service-a"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, config)

	// Check parsed values
	assert.Equal(t, 1, config.Version)
	assert.Equal(t, "chore/sync-files", config.Defaults.BranchPrefix)
	assert.Equal(t, []string{"automated-sync", "template"}, config.Defaults.PRLabels)

	// Multi-source config should be parsed directly
	require.Len(t, config.Mappings, 1)
	assert.Equal(t, "org/template-repo", config.Mappings[0].Source.Repo)
	assert.Equal(t, "master", config.Mappings[0].Source.Branch)
	assert.Equal(t, "template", config.Mappings[0].Source.ID)

	require.Len(t, config.Mappings[0].Targets, 1)
	assert.Equal(t, "org/service-a", config.Mappings[0].Targets[0].Repo)
	require.Len(t, config.Mappings[0].Targets[0].Files, 1)
	assert.Equal(t, ".github/workflows/ci.yml", config.Mappings[0].Targets[0].Files[0].Src)
	assert.Equal(t, ".github/workflows/ci.yml", config.Mappings[0].Targets[0].Files[0].Dest)
	assert.True(t, config.Mappings[0].Targets[0].Transform.RepoName)
	assert.Equal(t, "service-a", config.Mappings[0].Targets[0].Transform.Variables["SERVICE"])
}

func TestLoadFromReader_MinimalConfig(t *testing.T) {
	yamlContent := `
version: 1
mappings:
  - source:
      repo: "org/template"
      id: "template"
    targets:
      - repo: "org/service"
        files:
          - src: "file.txt"
            dest: "file.txt"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, config)

	// Check defaults were applied
	require.Len(t, config.Mappings, 1)
	assert.Equal(t, "main", config.Mappings[0].Source.Branch)
	assert.Equal(t, "chore/sync-files", config.Defaults.BranchPrefix)
	assert.Equal(t, []string{"automated-sync"}, config.Defaults.PRLabels)
}

func TestLoadFromReader_InvalidYAML(t *testing.T) {
	yamlContent := `
version: 1
source:
  repo: [invalid
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

func TestLoadFromReader_UnknownFields(t *testing.T) {
	yamlContent := `
version: 1
unknown_field: "value"
mappings:
  - source:
      repo: "org/repo"
      id: "repo"
    targets:
      - repo: "org/target"
        files:
          - src: "file"
            dest: "file"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.Error(t, err)
	assert.Nil(t, config)
}

func TestValidate_ValidConfig(t *testing.T) {
	config := &Config{
		Version: 1,
		Defaults: DefaultConfig{
			BranchPrefix: "chore/sync-files",
			PRLabels:     []string{"automated"},
		},
		Mappings: []SourceMapping{
			{
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "master",
					ID:     "template",
				},
				Targets: []TargetConfig{
					{
						Repo: "org/service",
						Files: []FileMapping{
							{Src: "file.txt", Dest: "file.txt"},
						},
					},
				},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	assert.NoError(t, err)
}

func TestValidate_InvalidVersion(t *testing.T) {
	config := &Config{
		Version: 2,
		Mappings: []SourceMapping{
			{
				Source:  SourceConfig{Repo: "org/repo", Branch: "master", ID: "repo"},
				Targets: []TargetConfig{{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}}},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config version: 2")
}

func TestValidate_MissingSourceRepo(t *testing.T) {
	config := &Config{
		Version: 1,
		Mappings: []SourceMapping{
			{
				Source:  SourceConfig{Branch: "master", ID: "empty"},
				Targets: []TargetConfig{{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}}},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field cannot be empty: repository name")
}

func TestValidate_InvalidRepoFormat(t *testing.T) {
	testCases := []struct {
		name string
		repo string
	}{
		{"no slash", "invalid-repo"},
		{"multiple slashes", "org/repo/extra"},
		{"empty org", "/repo"},
		{"empty repo", "org/"},
		{"starts with dash", "-org/repo"},
		{"starts with dot", ".org/repo"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source:  SourceConfig{Repo: tc.repo, Branch: "master", ID: "test"},
						Targets: []TargetConfig{{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}}},
					},
				},
			}

			err := config.ValidateWithLogging(context.Background(), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid format: repository name")
		})
	}
}

func TestValidate_InvalidBranch(t *testing.T) {
	config := &Config{
		Version: 1,
		Mappings: []SourceMapping{
			{
				Source:  SourceConfig{Repo: "org/repo", Branch: "", ID: "repo"},
				Targets: []TargetConfig{{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}}},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field cannot be empty: branch name")
}

func TestValidate_NoTargets(t *testing.T) {
	config := &Config{
		Version: 1,
		Mappings: []SourceMapping{
			{
				Source:  SourceConfig{Repo: "org/repo", Branch: "master", ID: "repo"},
				Targets: []TargetConfig{},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mapping has no targets")
}

func TestValidate_DuplicateTargets(t *testing.T) {
	config := &Config{
		Version: 1,
		Mappings: []SourceMapping{
			{
				Source: SourceConfig{Repo: "org/repo", Branch: "master", ID: "repo"},
				Targets: []TargetConfig{
					{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}},
					{Repo: "org/target", Files: []FileMapping{{Src: "f2", Dest: "f2"}}},
				},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate target repository: org/target")
}

func TestValidate_NoFileMappings(t *testing.T) {
	config := &Config{
		Version: 1,
		Mappings: []SourceMapping{
			{
				Source: SourceConfig{Repo: "org/repo", Branch: "master", ID: "repo"},
				Targets: []TargetConfig{
					{Repo: "org/target", Files: []FileMapping{}},
				},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one file or directory mapping is required")
}

func TestValidate_InvalidFilePaths(t *testing.T) {
	testCases := []struct {
		name string
		src  string
		dest string
		err  string
	}{
		{"empty src", "", "file", "field is required: source path"},
		{"empty dest", "file", "", "field is required: destination path"},
		{"absolute src", "/absolute/path", "file", "must be relative"},
		{"absolute dest", "file", "/absolute/path", "must be relative"},
		{"escape src", "../escape", "file", "path traversal detected"},
		{"escape dest", "file", "../escape", "path traversal detected"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{Repo: "org/repo", Branch: "master"},
						Targets: []TargetConfig{
							{
								Repo:  "org/target",
								Files: []FileMapping{{Src: tc.src, Dest: tc.dest}},
							},
						},
					},
				},
			}

			err := config.ValidateWithLogging(context.Background(), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.err)
		})
	}
}

func TestValidate_DuplicateDestinations(t *testing.T) {
	config := &Config{
		Version: 1,
		Mappings: []SourceMapping{
			{
				Source: SourceConfig{Repo: "org/repo", Branch: "master"},
				Targets: []TargetConfig{
					{
						Repo: "org/target",
						Files: []FileMapping{
							{Src: "file1.txt", Dest: "same.txt"},
							{Src: "file2.txt", Dest: "same.txt"},
						},
					},
				},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate destination: same.txt")
}

func TestValidate_EmptyPRLabel(t *testing.T) {
	config := &Config{
		Version: 1,
		Defaults: DefaultConfig{
			PRLabels: []string{"valid", "  ", "another"},
		},
		Mappings: []SourceMapping{
			{
				Source: SourceConfig{Repo: "org/repo", Branch: "master", ID: "repo"},
				Targets: []TargetConfig{
					{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}},
				},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestValidate_EmptyTargetPRLabel(t *testing.T) {
	config := &Config{
		Version: 1,
		Defaults: DefaultConfig{
			PRLabels: []string{"default-label"},
		},
		Mappings: []SourceMapping{
			{
				Source: SourceConfig{Repo: "org/repo", Branch: "master", ID: "repo"},
				Targets: []TargetConfig{
					{
						Repo:     "org/target",
						Files:    []FileMapping{{Src: "f", Dest: "f"}},
						PRLabels: []string{"valid", "  ", "another"}, // Empty label should cause validation error
					},
				},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target PR label")
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestLoad_FileNotFound(t *testing.T) {
	config, err := Load("/non/existent/file.yaml")
	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to open config file")
}

// TestApplyDefaults tests the applyDefaults function behavior
func TestApplyDefaults(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected Config
	}{
		{
			name: "ApplyAllDefaults",
			input: `
version: 1
mappings:
  - source:
      repo: "org/template"
      id: "template"
    targets:
      - repo: "org/service"
        files:
          - src: "file.txt"
            dest: "file.txt"
`,
			expected: Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							Repo:   "org/template",
							Branch: "main",
							ID:     "template",
						},
					},
				},
				Defaults: DefaultConfig{
					BranchPrefix: "chore/sync-files",
					PRLabels:     []string{"automated-sync"},
				},
			},
		},
		{
			name: "KeepExistingValues",
			input: `
version: 1
defaults:
  branch_prefix: "custom/prefix"
  pr_labels: ["custom-label", "another"]
mappings:
  - source:
      repo: "org/template"
      id: "template"
    targets:
      - repo: "org/service"
        files:
          - src: "file.txt"
            dest: "file.txt"
`,
			expected: Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							Repo:   "org/template",
							Branch: "main",
							ID:     "template",
						},
					},
				},
				Defaults: DefaultConfig{
					BranchPrefix: "custom/prefix",
					PRLabels:     []string{"custom-label", "another"},
				},
			},
		},
		{
			name: "PartialDefaults",
			input: `
version: 1
defaults:
  pr_labels: ["my-label"]
mappings:
  - source:
      repo: "org/template"
      branch: "develop"
      id: "template"
    targets:
      - repo: "org/service"
        files:
          - src: "file.txt"
            dest: "file.txt"
`,
			expected: Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							Repo:   "org/template",
							Branch: "develop",
							ID:     "template",
						},
					},
				},
				Defaults: DefaultConfig{
					BranchPrefix: "chore/sync-files",
					PRLabels:     []string{"my-label"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := LoadFromReader(strings.NewReader(tc.input))
			require.NoError(t, err)
			require.NotNil(t, config)

			assert.Equal(t, tc.expected.Version, config.Version)
			assert.Equal(t, tc.expected.Defaults.BranchPrefix, config.Defaults.BranchPrefix)
			assert.Equal(t, tc.expected.Defaults.PRLabels, config.Defaults.PRLabels)

			// Multi-source config should be parsed directly
			require.Len(t, config.Mappings, 1)
			assert.Equal(t, tc.expected.Mappings[0].Source.Repo, config.Mappings[0].Source.Repo)
			assert.Equal(t, tc.expected.Mappings[0].Source.Branch, config.Mappings[0].Source.Branch)
			assert.Equal(t, tc.expected.Mappings[0].Source.ID, config.Mappings[0].Source.ID)
		})
	}
}

// TestLoadFromReader_StrictParsing tests that strict YAML parsing is enforced
func TestLoadFromReader_StrictParsing(t *testing.T) {
	yamlContent := `
version: 1
unknown_field: "should fail"
mappings:
  - source:
      repo: "org/repo"
      id: "repo"
    targets:
      - repo: "org/target"
        files:
          - src: "file"
            dest: "file"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

// TestLoadFromReader_EmptyInput tests behavior with empty input
func TestLoadFromReader_EmptyInput(t *testing.T) {
	config, err := LoadFromReader(strings.NewReader(""))
	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}

// TestLoadFromReader_InvalidInput tests various invalid inputs
func TestLoadFromReader_InvalidInput(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "InvalidYAMLSyntax",
			input: "version: 1\nsource:\n  repo: [invalid",
		},
		{
			name:  "NotYAML",
			input: "<xml>not yaml</xml>",
		},
		{
			name:  "MixedTypes",
			input: "version: \"string instead of int\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := LoadFromReader(strings.NewReader(tc.input))
			require.Error(t, err)
			assert.Nil(t, config)
			assert.Contains(t, err.Error(), "failed to parse YAML")
		})
	}
}

func TestLoadFromReader_DirectoryConfig(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid directory config with defaults",
			yaml: `
version: 1
mappings:
  - source:
      repo: "org/template"
      id: "template"
    targets:
      - repo: "org/service"
        directories:
          - src: ".github/workflows"
            dest: ".github/workflows"
`,
			check: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				require.Len(t, cfg.Mappings[0].Targets[0].Directories, 1)
				dir := cfg.Mappings[0].Targets[0].Directories[0]
				assert.Equal(t, ".github/workflows", dir.Src)
				assert.Equal(t, ".github/workflows", dir.Dest)
				assert.Equal(t, DefaultExclusions(), dir.Exclude)
				require.NotNil(t, dir.PreserveStructure)
				assert.True(t, *dir.PreserveStructure)
				require.NotNil(t, dir.IncludeHidden)
				assert.True(t, *dir.IncludeHidden)
			},
		},
		{
			name: "directory with custom exclusions",
			yaml: `
version: 1
mappings:
  - source:
      repo: "org/template"
      id: "template"
    targets:
      - repo: "org/service"
        directories:
          - src: "configs"
            dest: "configs"
            exclude: ["*.local", "*.secret"]
            preserve_structure: false
            include_hidden: false
`,
			check: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				dir := cfg.Mappings[0].Targets[0].Directories[0]
				assert.Equal(t, []string{"*.local", "*.secret"}, dir.Exclude)
				require.NotNil(t, dir.PreserveStructure)
				assert.False(t, *dir.PreserveStructure)
				require.NotNil(t, dir.IncludeHidden)
				assert.False(t, *dir.IncludeHidden)
			},
		},
		{
			name: "directory with transform",
			yaml: `
version: 1
mappings:
  - source:
      repo: "org/template"
      id: "template"
    targets:
      - repo: "org/service"
        directories:
          - src: "templates"
            dest: "templates"
            transform:
              repo_name: true
              variables:
                ENV: "prod"
`,
			check: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				dir := cfg.Mappings[0].Targets[0].Directories[0]
				assert.True(t, dir.Transform.RepoName)
				assert.Equal(t, "prod", dir.Transform.Variables["ENV"])
			},
		},
		{
			name: "mixed files and directories",
			yaml: `
version: 1
mappings:
  - source:
      repo: "org/template"
      id: "template"
    targets:
      - repo: "org/service"
        files:
          - src: "Makefile"
            dest: "Makefile"
        directories:
          - src: ".github"
            dest: ".github"
`,
			check: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Mappings, 1)
				require.Len(t, cfg.Mappings[0].Targets, 1)
				assert.Len(t, cfg.Mappings[0].Targets[0].Files, 1)
				assert.Len(t, cfg.Mappings[0].Targets[0].Directories, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadFromReader(strings.NewReader(tt.yaml))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
