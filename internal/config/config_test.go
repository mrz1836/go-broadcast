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
source:
  repo: "org/template-repo"
  branch: "master"
defaults:
  branch_prefix: "sync/template"
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
	assert.Equal(t, "org/template-repo", config.Source.Repo)
	assert.Equal(t, "master", config.Source.Branch)
	assert.Equal(t, "sync/template", config.Defaults.BranchPrefix)
	assert.Equal(t, []string{"automated-sync", "template"}, config.Defaults.PRLabels)

	require.Len(t, config.Targets, 1)
	assert.Equal(t, "org/service-a", config.Targets[0].Repo)
	require.Len(t, config.Targets[0].Files, 1)
	assert.Equal(t, ".github/workflows/ci.yml", config.Targets[0].Files[0].Src)
	assert.Equal(t, ".github/workflows/ci.yml", config.Targets[0].Files[0].Dest)
	assert.True(t, config.Targets[0].Transform.RepoName)
	assert.Equal(t, "service-a", config.Targets[0].Transform.Variables["SERVICE"])
}

func TestLoadFromReader_MinimalConfig(t *testing.T) {
	yamlContent := `
version: 1
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

	// Check defaults were applied
	assert.Equal(t, "master", config.Source.Branch)
	assert.Equal(t, "sync/template", config.Defaults.BranchPrefix)
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
		Source: SourceConfig{
			Repo:   "org/template",
			Branch: "master",
		},
		Defaults: DefaultConfig{
			BranchPrefix: "sync/template",
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
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	assert.NoError(t, err)
}

func TestValidate_InvalidVersion(t *testing.T) {
	config := &Config{
		Version: 2,
		Source:  SourceConfig{Repo: "org/repo", Branch: "master"},
		Targets: []TargetConfig{{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}}},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config version: 2")
}

func TestValidate_MissingSourceRepo(t *testing.T) {
	config := &Config{
		Version: 1,
		Source:  SourceConfig{Branch: "master"},
		Targets: []TargetConfig{{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}}},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source repository is required")
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
				Source:  SourceConfig{Repo: tc.repo, Branch: "master"},
				Targets: []TargetConfig{{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}}},
			}

			err := config.ValidateWithLogging(context.Background(), nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid repository format")
		})
	}
}

func TestValidate_InvalidBranch(t *testing.T) {
	config := &Config{
		Version: 1,
		Source:  SourceConfig{Repo: "org/repo", Branch: ""},
		Targets: []TargetConfig{{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}}},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source branch is required")
}

func TestValidate_NoTargets(t *testing.T) {
	config := &Config{
		Version: 1,
		Source:  SourceConfig{Repo: "org/repo", Branch: "master"},
		Targets: []TargetConfig{},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one target repository must be specified")
}

func TestValidate_DuplicateTargets(t *testing.T) {
	config := &Config{
		Version: 1,
		Source:  SourceConfig{Repo: "org/repo", Branch: "master"},
		Targets: []TargetConfig{
			{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}},
			{Repo: "org/target", Files: []FileMapping{{Src: "f2", Dest: "f2"}}},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate target repository: org/target")
}

func TestValidate_NoFileMappings(t *testing.T) {
	config := &Config{
		Version: 1,
		Source:  SourceConfig{Repo: "org/repo", Branch: "master"},
		Targets: []TargetConfig{
			{Repo: "org/target", Files: []FileMapping{}},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one file mapping is required")
}

func TestValidate_InvalidFilePaths(t *testing.T) {
	testCases := []struct {
		name string
		src  string
		dest string
		err  string
	}{
		{"empty src", "", "file", "source file path is required"},
		{"empty dest", "file", "", "destination file path is required"},
		{"absolute src", "/absolute/path", "file", "invalid source path"},
		{"absolute dest", "file", "/absolute/path", "invalid destination path"},
		{"escape src", "../escape", "file", "invalid source path"},
		{"escape dest", "file", "../escape", "invalid destination path"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &Config{
				Version: 1,
				Source:  SourceConfig{Repo: "org/repo", Branch: "master"},
				Targets: []TargetConfig{
					{
						Repo:  "org/target",
						Files: []FileMapping{{Src: tc.src, Dest: tc.dest}},
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
		Source:  SourceConfig{Repo: "org/repo", Branch: "master"},
		Targets: []TargetConfig{
			{
				Repo: "org/target",
				Files: []FileMapping{
					{Src: "file1.txt", Dest: "same.txt"},
					{Src: "file2.txt", Dest: "same.txt"},
				},
			},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate destination file: same.txt")
}

func TestValidate_EmptyPRLabel(t *testing.T) {
	config := &Config{
		Version: 1,
		Source:  SourceConfig{Repo: "org/repo", Branch: "master"},
		Defaults: DefaultConfig{
			PRLabels: []string{"valid", "  ", "another"},
		},
		Targets: []TargetConfig{
			{Repo: "org/target", Files: []FileMapping{{Src: "f", Dest: "f"}}},
		},
	}

	err := config.ValidateWithLogging(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PR label cannot be empty")
}

func TestLoad_FileNotFound(t *testing.T) {
	config, err := Load("/non/existent/file.yaml")
	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to open config file")
}
