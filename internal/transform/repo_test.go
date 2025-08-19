package transform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoTransformer_Name(t *testing.T) {
	transformer := NewRepoTransformer()
	assert.Equal(t, "repository-name-replacer", transformer.Name())
}

func TestRepoTransformer_Transform(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		ctx         Context
		wantContent string
		wantError   bool
	}{
		{
			name: "go.mod file transformation",
			content: `module github.com/oldorg/oldrepo

go 1.21

require (
	github.com/oldorg/oldrepo/pkg v1.0.0
	github.com/other/dependency v2.0.0
)`,
			ctx: Context{
				SourceRepo: "oldorg/oldrepo",
				TargetRepo: "neworg/newrepo",
				FilePath:   "go.mod",
			},
			wantContent: `module github.com/neworg/newrepo

go 1.21

require (
	github.com/neworg/newrepo/pkg v1.0.0
	github.com/other/dependency v2.0.0
)`,
			wantError: false,
		},
		{
			name: "go source file transformation",
			content: `package main

import (
	"fmt"
	"github.com/oldorg/oldrepo/pkg/util"
	"github.com/oldorg/oldrepo/internal/config"
)

func main() {
	fmt.Println("github.com/oldorg/oldrepo")
}`,
			ctx: Context{
				SourceRepo: "oldorg/oldrepo",
				TargetRepo: "neworg/newrepo",
				FilePath:   "main.go",
			},
			wantContent: `package main

import (
	"fmt"
	"github.com/neworg/newrepo/pkg/util"
	"github.com/neworg/newrepo/internal/config"
)

func main() {
	fmt.Println("github.com/neworg/newrepo")
}`,
			wantError: false,
		},
		{
			name: "markdown documentation transformation",
			content: `# oldrepo

[![Build Status](https://github.com/oldorg/oldrepo/workflows/test/badge.svg)](https://github.com/oldorg/oldrepo/actions)

## Installation

` + "```bash" + `
go get github.com/oldorg/oldrepo
` + "```" + `

Visit https://github.com/oldorg/oldrepo for more info.`,
			ctx: Context{
				SourceRepo: "oldorg/oldrepo",
				TargetRepo: "neworg/newrepo",
				FilePath:   "README.md",
			},
			wantContent: `# newrepo

[![Build Status](https://github.com/neworg/newrepo/workflows/test/badge.svg)](https://github.com/neworg/newrepo/actions)

## Installation

` + "```bash" + `
go get github.com/neworg/newrepo
` + "```" + `

Visit https://github.com/neworg/newrepo for more info.`,
			wantError: false,
		},
		{
			name: "yaml configuration transformation",
			content: `name: CI
repository: oldorg/oldrepo
settings:
  repo: "oldrepo"
  full_name: "oldorg/oldrepo"`,
			ctx: Context{
				SourceRepo: "oldorg/oldrepo",
				TargetRepo: "neworg/newrepo",
				FilePath:   "config.yaml",
			},
			wantContent: `name: CI
repository: neworg/newrepo
settings:
  repo: "newrepo"
  full_name: "neworg/newrepo"`,
			wantError: false,
		},
		{
			name:    "no transformation when repos are the same",
			content: `module github.com/org/repo`,
			ctx: Context{
				SourceRepo: "org/repo",
				TargetRepo: "org/repo",
				FilePath:   "go.mod",
			},
			wantContent: `module github.com/org/repo`,
			wantError:   false,
		},
		{
			name:    "invalid source repo format",
			content: `test content`,
			ctx: Context{
				SourceRepo: "invalid-repo-format",
				TargetRepo: "neworg/newrepo",
				FilePath:   "test.txt",
			},
			wantContent: "",
			wantError:   true,
		},
		{
			name:    "invalid target repo format",
			content: `test content`,
			ctx: Context{
				SourceRepo: "oldorg/oldrepo",
				TargetRepo: "invalid-repo-format",
				FilePath:   "test.txt",
			},
			wantContent: "",
			wantError:   true,
		},
		{
			name: "general file transformation",
			content: `This project uses oldorg/oldrepo as its base.
See oldorg/oldrepo for details.`,
			ctx: Context{
				SourceRepo: "oldorg/oldrepo",
				TargetRepo: "neworg/newrepo",
				FilePath:   "notes.txt",
			},
			wantContent: `This project uses neworg/newrepo as its base.
See neworg/newrepo for details.`,
			wantError: false,
		},
		{
			name: "avoid over-replacement in go files",
			content: `package oldrepo

// Package oldrepo provides utilities
var repoName = "oldrepo"`,
			ctx: Context{
				SourceRepo: "oldorg/oldrepo",
				TargetRepo: "neworg/newrepo",
				FilePath:   "doc.go",
			},
			wantContent: `package oldrepo

// Package oldrepo provides utilities
var repoName = "oldrepo"`,
			wantError: false,
		},
		{
			name: "funding yml with URL parameters",
			content: `# These are supported funding model platforms

github: mrz1836
custom: https://mrz1818.com/?tab=tips&utm_source=github&utm_medium=sponsor-link&utm_campaign=go-broadcast&utm_term=go-broadcast&utm_content=go-broadcast`,
			ctx: Context{
				SourceRepo: "mrz1836/go-broadcast",
				TargetRepo: "mrz1836/go-coverage",
				FilePath:   ".github/FUNDING.yml",
			},
			wantContent: `# These are supported funding model platforms

github: mrz1836
custom: https://mrz1818.com/?tab=tips&utm_source=github&utm_medium=sponsor-link&utm_campaign=go-coverage&utm_term=go-coverage&utm_content=go-coverage`,
			wantError: false,
		},
		{
			name: "vscode settings with repo names",
			content: `{
  "formatting.local": "github.com/mrz1836/go-broadcast",
  "go.lintTool": "golangci-lint"
}`,
			ctx: Context{
				SourceRepo: "mrz1836/go-broadcast",
				TargetRepo: "mrz1836/go-coverage",
				FilePath:   ".vscode/settings.json",
			},
			wantContent: `{
  "formatting.local": "github.com/mrz1836/go-coverage",
  "go.lintTool": "golangci-lint"
}`,
			wantError: false,
		},
		{
			name: "general text file with standalone repo names",
			content: `This project is called go-broadcast and uses go-broadcast for testing.
The go-broadcast tool is amazing!`,
			ctx: Context{
				SourceRepo: "mrz1836/go-broadcast",
				TargetRepo: "mrz1836/go-coverage",
				FilePath:   "notes.txt",
			},
			wantContent: `This project is called go-coverage and uses go-coverage for testing.
The go-coverage tool is amazing!`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := NewRepoTransformer()
			result, err := transformer.Transform([]byte(tt.content), tt.ctx)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantContent, string(result))
			}
		})
	}
}

func TestRepoTransformer_SpecialCases(t *testing.T) {
	transformer := NewRepoTransformer()

	t.Run("handles repos with special regex characters", func(t *testing.T) {
		content := `module github.com/old.org/old-repo.v2`
		ctx := Context{
			SourceRepo: "old.org/old-repo.v2",
			TargetRepo: "new.org/new-repo.v2",
			FilePath:   "go.mod",
		}

		result, err := transformer.Transform([]byte(content), ctx)
		require.NoError(t, err)
		assert.Equal(t, `module github.com/new.org/new-repo.v2`, string(result))
	})

	t.Run("preserves import paths correctly", func(t *testing.T) {
		content := `import (
	"github.com/oldorg/oldrepo/pkg/util"
	"github.com/oldorg/oldrepo-client/api"
)`
		ctx := Context{
			SourceRepo: "oldorg/oldrepo",
			TargetRepo: "neworg/newrepo",
			FilePath:   "main.go",
		}

		result, err := transformer.Transform([]byte(content), ctx)
		require.NoError(t, err)
		// Should only replace exact matches, not partial
		assert.Contains(t, string(result), "github.com/neworg/newrepo/pkg/util")
		assert.Contains(t, string(result), "github.com/oldorg/oldrepo-client/api")
	})
}
