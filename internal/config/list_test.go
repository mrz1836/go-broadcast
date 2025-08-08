package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileListOverride tests that inline files override list files with same destination
func TestFileListOverride(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "defaults"
    name: "Default Files"
    files:
      - src: "templates/.editorconfig"
        dest: ".editorconfig"
      - src: "templates/.gitignore"
        dest: ".gitignore"
      - src: "templates/LICENSE"
        dest: "LICENSE"
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        file_list_refs: ["defaults"]
        files:
          # This should override the one from the list
          - src: "custom/LICENSE"
            dest: "LICENSE"
          # This is a new file
          - src: "README.md"
            dest: "README.md"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)
	require.NotNil(t, config)

	target := config.Groups[0].Targets[0]

	// Should have 4 files total: 2 from list (not overridden) + 1 override + 1 new
	require.Len(t, target.Files, 4)

	// Check that LICENSE was overridden
	var licenseSrc string
	for _, file := range target.Files {
		if file.Dest == "LICENSE" {
			licenseSrc = file.Src
			break
		}
	}
	assert.Equal(t, "custom/LICENSE", licenseSrc, "LICENSE should be overridden by inline file")

	// Check other files are present
	destMap := make(map[string]string)
	for _, file := range target.Files {
		destMap[file.Dest] = file.Src
	}

	assert.Equal(t, "templates/.editorconfig", destMap[".editorconfig"])
	assert.Equal(t, "templates/.gitignore", destMap[".gitignore"])
	assert.Equal(t, "custom/LICENSE", destMap["LICENSE"])
	assert.Equal(t, "README.md", destMap["README.md"])
}

// TestDirectoryListOverride tests that inline directories override list directories
func TestDirectoryListOverride(t *testing.T) {
	yamlContent := `
version: 1
directory_lists:
  - id: "standard-dirs"
    name: "Standard Directories"
    directories:
      - src: ".github/workflows"
        dest: ".github/workflows"
        exclude: ["*.tmp"]
      - src: ".github/actions"
        dest: ".github/actions"
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        directory_list_refs: ["standard-dirs"]
        directories:
          # Override with different exclusions
          - src: "custom/workflows"
            dest: ".github/workflows"
            exclude: ["*.test", "*.tmp"]
          # Add new directory
          - src: "docs"
            dest: "documentation"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	target := config.Groups[0].Targets[0]
	require.Len(t, target.Directories, 3)

	// Check override
	for _, dir := range target.Directories {
		if dir.Dest == ".github/workflows" {
			assert.Equal(t, "custom/workflows", dir.Src)
			assert.Equal(t, []string{"*.test", "*.tmp"}, dir.Exclude)
			break
		}
	}
}

// TestMultipleFileListReferences tests referencing multiple file lists
func TestMultipleFileListReferences(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "base-files"
    name: "Base Files"
    files:
      - src: ".editorconfig"
        dest: ".editorconfig"
      - src: "LICENSE"
        dest: "LICENSE"
  - id: "dev-files"
    name: "Development Files"
    files:
      - src: ".gitignore"
        dest: ".gitignore"
      - src: "Makefile"
        dest: "Makefile"
  - id: "override-files"
    name: "Override Files"
    files:
      # This should override the one from base-files
      - src: "MIT-LICENSE"
        dest: "LICENSE"
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        # Later lists override earlier ones for same destination
        file_list_refs: ["base-files", "dev-files", "override-files"]
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	target := config.Groups[0].Targets[0]

	// Should have 4 unique destinations
	require.Len(t, target.Files, 4)

	destMap := make(map[string]string)
	for _, file := range target.Files {
		destMap[file.Dest] = file.Src
	}

	// LICENSE from override-files should win
	assert.Equal(t, "MIT-LICENSE", destMap["LICENSE"])
	assert.Equal(t, ".editorconfig", destMap[".editorconfig"])
	assert.Equal(t, ".gitignore", destMap[".gitignore"])
	assert.Equal(t, "Makefile", destMap["Makefile"])
}

// TestEmptyListReferences tests targets with only list references, no inline files
func TestEmptyListReferences(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "all-files"
    name: "All Files"
    files:
      - src: "file1.txt"
        dest: "file1.txt"
      - src: "file2.txt"
        dest: "file2.txt"
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        file_list_refs: ["all-files"]
        # No inline files
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	target := config.Groups[0].Targets[0]
	require.Len(t, target.Files, 2)

	// Check files are present (order doesn't matter since map iteration is not deterministic)
	fileMap := make(map[string]string)
	for _, file := range target.Files {
		fileMap[file.Dest] = file.Src
	}
	assert.Equal(t, "file1.txt", fileMap["file1.txt"])
	assert.Equal(t, "file2.txt", fileMap["file2.txt"])
}

// TestNoListReferences tests targets with only inline files
func TestNoListReferences(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "unused"
    name: "Unused List"
    files:
      - src: "unused.txt"
        dest: "unused.txt"
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        # No list references
        files:
          - src: "inline1.txt"
            dest: "inline1.txt"
          - src: "inline2.txt"
            dest: "inline2.txt"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	target := config.Groups[0].Targets[0]
	require.Len(t, target.Files, 2)
	assert.Equal(t, "inline1.txt", target.Files[0].Src)
	assert.Equal(t, "inline2.txt", target.Files[1].Src)
}

// TestComplexScenario tests a complex real-world scenario
func TestComplexScenario(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "github-base"
    name: "GitHub Base Files"
    files:
      - src: ".github/CODE_OF_CONDUCT.md"
        dest: ".github/CODE_OF_CONDUCT.md"
      - src: ".github/SECURITY.md"
        dest: ".github/SECURITY.md"
  - id: "editor-config"
    name: "Editor Config"
    files:
      - src: ".editorconfig"
        dest: ".editorconfig"
      - src: ".prettierrc"
        dest: ".prettierrc"
  - id: "go-specific"
    name: "Go Files"
    files:
      - src: "go.mod.template"
        dest: "go.mod"
      - src: ".golangci.yml"
        dest: ".golangci.yml"
directory_lists:
  - id: "workflows"
    name: "GitHub Workflows"
    directories:
      - src: ".github/workflows"
        dest: ".github/workflows"
        exclude: ["*.local"]
  - id: "docs"
    name: "Documentation"
    directories:
      - src: "docs/templates"
        dest: "docs"
        exclude: ["*.draft"]
groups:
  - name: "Go Services"
    id: "go-services"
    source:
      repo: "org/templates"
    targets:
      # Service A: Uses all lists
      - repo: "org/service-a"
        file_list_refs: ["github-base", "editor-config", "go-specific"]
        directory_list_refs: ["workflows", "docs"]
        files:
          - src: "LICENSE-APACHE"
            dest: "LICENSE"

      # Service B: Selective lists with overrides
      - repo: "org/service-b"
        file_list_refs: ["github-base", "editor-config"]
        directory_list_refs: ["workflows"]
        files:
          # Override editor config
          - src: "custom/.editorconfig"
            dest: ".editorconfig"
          - src: "LICENSE-MIT"
            dest: "LICENSE"
        directories:
          # Custom docs directory
          - src: "custom-docs"
            dest: "docs"

      # Service C: Minimal setup
      - repo: "org/service-c"
        file_list_refs: ["editor-config"]
        files:
          - src: "README.md"
            dest: "README.md"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)
	require.Len(t, config.Groups[0].Targets, 3)

	// Service A: Should have all files from lists plus LICENSE
	serviceA := config.Groups[0].Targets[0]
	assert.Len(t, serviceA.Files, 7) // 2 github + 2 editor + 2 go + 1 license
	assert.Len(t, serviceA.Directories, 2)

	// Check Service A has files from all lists
	aFiles := make(map[string]string)
	for _, f := range serviceA.Files {
		aFiles[f.Dest] = f.Src
	}
	assert.Equal(t, ".github/CODE_OF_CONDUCT.md", aFiles[".github/CODE_OF_CONDUCT.md"])
	assert.Equal(t, ".editorconfig", aFiles[".editorconfig"])
	assert.Equal(t, "go.mod.template", aFiles["go.mod"])
	assert.Equal(t, "LICENSE-APACHE", aFiles["LICENSE"])

	// Service B: Should have overridden editorconfig
	serviceB := config.Groups[0].Targets[1]
	assert.Len(t, serviceB.Files, 5)       // 2 github + 2 editor (1 overridden) + 1 license
	assert.Len(t, serviceB.Directories, 2) // 1 from list + 1 custom

	bFiles := make(map[string]string)
	for _, f := range serviceB.Files {
		bFiles[f.Dest] = f.Src
	}
	assert.Equal(t, "custom/.editorconfig", bFiles[".editorconfig"], "Should be overridden")
	assert.Equal(t, ".prettierrc", bFiles[".prettierrc"])
	assert.Equal(t, "LICENSE-MIT", bFiles["LICENSE"])

	// Service C: Minimal
	serviceC := config.Groups[0].Targets[2]
	assert.Len(t, serviceC.Files, 3) // 2 from editor-config + 1 readme
	assert.Empty(t, serviceC.Directories)
}

// TestDuplicatesWithinList tests handling of duplicates within a single list
func TestDuplicatesWithinList(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "duplicates"
    name: "List with Duplicates"
    files:
      - src: "first.txt"
        dest: "output.txt"
      - src: "second.txt"
        dest: "output.txt"  # Same dest as above
      - src: "third.txt"
        dest: "another.txt"
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        file_list_refs: ["duplicates"]
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	target := config.Groups[0].Targets[0]
	// Should only have 2 files (last one with output.txt wins due to override semantics)
	require.Len(t, target.Files, 2)

	for _, f := range target.Files {
		if f.Dest == "output.txt" {
			assert.Equal(t, "second.txt", f.Src, "Last occurrence should win (override semantics)")
			break
		}
	}
}

// TestEmptyLists tests empty file and directory lists
func TestEmptyLists(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "empty-files"
    name: "Empty File List"
    files: []
directory_lists:
  - id: "empty-dirs"
    name: "Empty Dir List"
    directories: []
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        file_list_refs: ["empty-files"]
        directory_list_refs: ["empty-dirs"]
        files:
          - src: "actual.txt"
            dest: "actual.txt"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	target := config.Groups[0].Targets[0]
	assert.Len(t, target.Files, 1)
	assert.Equal(t, "actual.txt", target.Files[0].Src)
	assert.Empty(t, target.Directories)
}

// TestListValidation tests validation of file and directory lists
func TestListValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "empty_list_id",
			yaml: `
version: 1
file_lists:
  - id: ""
    name: "No ID"
    files:
      - src: "file.txt"
        dest: "file.txt"
groups:
  - name: "Test"
    id: "test"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        files:
          - src: "test.txt"
            dest: "test.txt"
`,
			wantErr: "list ID cannot be empty",
		},
		{
			name: "empty_list_name",
			yaml: `
version: 1
file_lists:
  - id: "test-id"
    name: ""
    files:
      - src: "file.txt"
        dest: "file.txt"
groups:
  - name: "Test"
    id: "test"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        files:
          - src: "test.txt"
            dest: "test.txt"
`,
			wantErr: "list name cannot be empty",
		},
		{
			name: "path_traversal_in_list",
			yaml: `
version: 1
file_lists:
  - id: "bad-list"
    name: "Bad List"
    files:
      - src: "../../../etc/passwd"
        dest: "passwd"
groups:
  - name: "Test"
    id: "test"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        file_list_refs: ["bad-list"]
`,
			wantErr: "path traversal not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadFromReader(strings.NewReader(tt.yaml))
			require.NoError(t, err, "parsing should succeed")

			err = config.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestDirectoryListWithModuleConfig tests directory lists with module configurations
func TestDirectoryListWithModuleConfig(t *testing.T) {
	yamlContent := `
version: 1
directory_lists:
  - id: "module-dirs"
    name: "Module Directories"
    directories:
      - src: "pkg/shared"
        dest: "vendor/shared"
        module:
          type: "go"
          version: "v1.2.3"
          check_tags: true
groups:
  - name: "Test Group"
    id: "test-group"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        directory_list_refs: ["module-dirs"]
        directories:
          # Override with different module version
          - src: "pkg/custom"
            dest: "vendor/shared"
            module:
              type: "go"
              version: "v2.0.0"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	target := config.Groups[0].Targets[0]
	require.Len(t, target.Directories, 1)

	// Check that module config was overridden
	dir := target.Directories[0]
	assert.Equal(t, "pkg/custom", dir.Src)
	assert.Equal(t, "vendor/shared", dir.Dest)
	require.NotNil(t, dir.Module)
	assert.Equal(t, "v2.0.0", dir.Module.Version)
}
