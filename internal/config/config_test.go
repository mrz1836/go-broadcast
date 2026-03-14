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
groups:
  - name: "Test Group"
    id: "test-group"
    description: "Test configuration group"
    priority: 1
    enabled: true
    source:
      repo: "org/template-repo"
      branch: "master"
    defaults:
      branch_prefix: "chore/sync-files"
      pr_labels: ["automated-sync", "template"]
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
	require.Len(t, config.Groups, 1)

	group := config.Groups[0]
	assert.Equal(t, "Test Group", group.Name)
	assert.Equal(t, "test-group", group.ID)
	assert.Equal(t, "Test configuration group", group.Description)
	assert.Equal(t, 1, group.Priority)
	assert.NotNil(t, group.Enabled)
	assert.True(t, *group.Enabled)

	assert.Equal(t, "org/template-repo", group.Source.Repo)
	assert.Equal(t, "master", group.Source.Branch)
	assert.Equal(t, "chore/sync-files", group.Defaults.BranchPrefix)
	assert.Equal(t, []string{"automated-sync", "template"}, group.Defaults.PRLabels)

	require.Len(t, group.Targets, 1)
	assert.Equal(t, "org/service-a", group.Targets[0].Repo)
	require.Len(t, group.Targets[0].Files, 1)
	assert.Equal(t, ".github/workflows/ci.yml", group.Targets[0].Files[0].Src)
	assert.Equal(t, ".github/workflows/ci.yml", group.Targets[0].Files[0].Dest)
	assert.True(t, group.Targets[0].Transform.RepoName)
	assert.Equal(t, "service-a", group.Targets[0].Transform.Variables["SERVICE"])
}

func TestLoadFromReader_MinimalConfig(t *testing.T) {
	yamlContent := `
version: 1
groups:
  - name: "Default Group"
    id: "default"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        files:
          - src: "file.txt"
            dest: "file.txt"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, config)

	// Check defaults were applied to the group
	require.Len(t, config.Groups, 1)
	group := config.Groups[0]
	assert.Equal(t, "main", group.Source.Branch)
	assert.Equal(t, "chore/sync-files", group.Defaults.BranchPrefix)
	assert.Equal(t, []string{"automated-sync"}, group.Defaults.PRLabels)
}

func TestLoadFromReader_WithFileLists(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "common-files"
    name: "Common Configuration Files"
    description: "Shared configuration files across projects"
    files:
      - src: ".editorconfig"
        dest: ".editorconfig"
      - src: ".gitattributes"
        dest: ".gitattributes"
directory_lists:
  - id: "github-dirs"
    name: "GitHub Directories"
    description: "Standard GitHub configuration directories"
    directories:
      - src: ".github/workflows"
        dest: ".github/workflows"
      - src: ".github/actions"
        dest: ".github/actions"
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template-repo"
    targets:
      - repo: "org/service-a"
        file_list_refs: ["common-files"]
        directory_list_refs: ["github-dirs"]
        files:
          - src: "LICENSE"
            dest: "LICENSE"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, config)

	// Check file lists
	require.Len(t, config.FileLists, 1)
	assert.Equal(t, "common-files", config.FileLists[0].ID)
	assert.Equal(t, "Common Configuration Files", config.FileLists[0].Name)
	assert.Len(t, config.FileLists[0].Files, 2)

	// Check directory lists
	require.Len(t, config.DirectoryLists, 1)
	assert.Equal(t, "github-dirs", config.DirectoryLists[0].ID)
	assert.Equal(t, "GitHub Directories", config.DirectoryLists[0].Name)
	assert.Len(t, config.DirectoryLists[0].Directories, 2)

	// Check that references were resolved
	require.Len(t, config.Groups, 1)
	require.Len(t, config.Groups[0].Targets, 1)
	target := config.Groups[0].Targets[0]

	// Should have 3 files total: 2 from list + 1 inline
	assert.Len(t, target.Files, 3)

	// Check files are present (order may vary due to map iteration)
	fileMap := make(map[string]string)
	for _, file := range target.Files {
		fileMap[file.Dest] = file.Src
	}
	assert.Equal(t, ".editorconfig", fileMap[".editorconfig"])
	assert.Equal(t, ".gitattributes", fileMap[".gitattributes"])
	assert.Equal(t, "LICENSE", fileMap["LICENSE"])

	// Should have 2 directories from list
	assert.Len(t, target.Directories, 2)

	// Check directories are present (order may vary due to map iteration)
	dirDests := make(map[string]string)
	for _, dir := range target.Directories {
		dirDests[dir.Dest] = dir.Src
	}
	assert.Equal(t, ".github/workflows", dirDests[".github/workflows"])
	assert.Equal(t, ".github/actions", dirDests[".github/actions"])
}

func TestLoadFromReader_InvalidListReferences(t *testing.T) {
	yamlContent := `
version: 1
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template-repo"
    targets:
      - repo: "org/service-a"
        file_list_refs: ["non-existent-list"]
        files:
          - src: "file.txt"
            dest: "file.txt"
`

	_, err := LoadFromReader(strings.NewReader(yamlContent))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list reference not found")
}

func TestLoadFromReader_DuplicateListIDs(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "duplicate-id"
    name: "First List"
    files:
      - src: "file1.txt"
        dest: "file1.txt"
  - id: "duplicate-id"
    name: "Second List"
    files:
      - src: "file2.txt"
        dest: "file2.txt"
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template-repo"
    targets:
      - repo: "org/service-a"
        files:
          - src: "file.txt"
            dest: "file.txt"
`

	_, err := LoadFromReader(strings.NewReader(yamlContent))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate list ID")
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
source:
  repo: "org/repo"
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
		Groups: []Group{
			{
				Name: "Test Group",
				ID:   "test-group",
				Source: SourceConfig{
					Repo:   "org/template",
					Branch: "master",
				},
				Defaults: DefaultConfig{
					BranchPrefix: "chore/sync-files",
					PRLabels:     []string{"automated"},
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
		Groups: []Group{
			{
				Name:    "Test Group",
				ID:      "test-group",
				Source:  SourceConfig{Repo: "org/repo", Branch: "master"},
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
		Groups: []Group{
			{
				Name:    "Test Group",
				ID:      "test-group",
				Source:  SourceConfig{Branch: "master"},
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
				Groups: []Group{
					{
						Name:    "Test Group",
						ID:      "test-group",
						Source:  SourceConfig{Repo: tc.repo, Branch: "master"},
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
		Groups: []Group{
			{
				Name:    "Test Group",
				ID:      "test-group",
				Source:  SourceConfig{Repo: "org/repo", Branch: ""},
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
		Groups: []Group{
			{
				Name:    "Test Group",
				ID:      "test-group",
				Source:  SourceConfig{Repo: "org/repo", Branch: "master"},
				Targets: []TargetConfig{},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one target repository must be specified")
}

func TestValidate_DuplicateTargets(t *testing.T) {
	config := &Config{
		Version: 1,
		Groups: []Group{
			{
				Name:   "Test Group",
				ID:     "test-group",
				Source: SourceConfig{Repo: "org/repo", Branch: "master"},
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
		Groups: []Group{
			{
				Name:   "Test Group",
				ID:     "test-group",
				Source: SourceConfig{Repo: "org/repo", Branch: "master"},
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
				Groups: []Group{
					{
						Name:   "Test Group",
						ID:     "test-group",
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
		Groups: []Group{
			{
				Name:   "Test Group",
				ID:     "test-group",
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
		Groups: []Group{
			{
				Name:   "Test Group",
				ID:     "test-group",
				Source: SourceConfig{Repo: "org/repo", Branch: "master"},
				Defaults: DefaultConfig{
					PRLabels: []string{"valid", "  ", "another"},
				},
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
		Groups: []Group{
			{
				Name:   "Test Group",
				ID:     "test-group",
				Source: SourceConfig{Repo: "org/repo", Branch: "master"},
				Defaults: DefaultConfig{
					PRLabels: []string{"default-label"},
				},
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

// TestApplyDefaults tests the applyDefaults function behavior with group-based configs
func TestApplyDefaults(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name: "ApplyAllDefaults",
			input: `
version: 1
groups:
  - name: "Default Group"
    id: "default"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        files:
          - src: "file.txt"
            dest: "file.txt"
`,
		},
		{
			name: "KeepExistingValues",
			input: `
version: 1
groups:
  - name: "Custom Group"
    id: "custom"
    source:
      repo: "org/template"
    defaults:
      branch_prefix: "custom/prefix"
      pr_labels: ["custom-label", "another"]
    targets:
      - repo: "org/service"
        files:
          - src: "file.txt"
            dest: "file.txt"
`,
		},
		{
			name: "PartialDefaults",
			input: `
version: 1
groups:
  - name: "Partial Group"
    id: "partial"
    source:
      repo: "org/template"
      branch: "develop"
    defaults:
      pr_labels: ["my-label"]
    targets:
      - repo: "org/service"
        files:
          - src: "file.txt"
            dest: "file.txt"
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := LoadFromReader(strings.NewReader(tc.input))
			require.NoError(t, err)
			require.NotNil(t, config)
			require.Len(t, config.Groups, 1)

			group := config.Groups[0]
			assert.Equal(t, 1, config.Version)

			// Check that defaults were applied appropriately
			switch tc.name {
			case "ApplyAllDefaults":
				assert.Equal(t, "org/template", group.Source.Repo)
				assert.Equal(t, "main", group.Source.Branch) // Default should be applied
				assert.Equal(t, "chore/sync-files", group.Defaults.BranchPrefix)
				assert.Equal(t, []string{"automated-sync"}, group.Defaults.PRLabels)
			case "KeepExistingValues":
				assert.Equal(t, "org/template", group.Source.Repo)
				assert.Equal(t, "main", group.Source.Branch) // Default should be applied
				assert.Equal(t, "custom/prefix", group.Defaults.BranchPrefix)
				assert.Equal(t, []string{"custom-label", "another"}, group.Defaults.PRLabels)
			case "PartialDefaults":
				assert.Equal(t, "org/template", group.Source.Repo)
				assert.Equal(t, "develop", group.Source.Branch)                  // User specified value
				assert.Equal(t, "chore/sync-files", group.Defaults.BranchPrefix) // Default applied
				assert.Equal(t, []string{"my-label"}, group.Defaults.PRLabels)
			}
		})
	}
}

// TestLoadFromReader_StrictParsing tests that strict YAML parsing is enforced
func TestLoadFromReader_StrictParsing(t *testing.T) {
	yamlContent := `
version: 1
unknown_field: "should fail"
source:
  repo: "org/repo"
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
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        directories:
          - src: ".github/workflows"
            dest: ".github/workflows"
`,
			check: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Groups, 1)
				require.Len(t, cfg.Groups[0].Targets, 1)
				require.Len(t, cfg.Groups[0].Targets[0].Directories, 1)
				dir := cfg.Groups[0].Targets[0].Directories[0]
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
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
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
				require.Len(t, cfg.Groups, 1)
				require.Len(t, cfg.Groups[0].Targets, 1)
				require.Len(t, cfg.Groups[0].Targets[0].Directories, 1)
				dir := cfg.Groups[0].Targets[0].Directories[0]
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
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
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
				require.Len(t, cfg.Groups, 1)
				require.Len(t, cfg.Groups[0].Targets, 1)
				require.Len(t, cfg.Groups[0].Targets[0].Directories, 1)
				dir := cfg.Groups[0].Targets[0].Directories[0]
				assert.True(t, dir.Transform.RepoName)
				assert.Equal(t, "prod", dir.Transform.Variables["ENV"])
			},
		},
		{
			name: "mixed files and directories",
			yaml: `
version: 1
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
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
				require.Len(t, cfg.Groups, 1)
				require.Len(t, cfg.Groups[0].Targets, 1)
				assert.Len(t, cfg.Groups[0].Targets[0].Files, 1)
				assert.Len(t, cfg.Groups[0].Targets[0].Directories, 1)
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
