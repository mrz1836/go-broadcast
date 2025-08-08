package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMixedListsAndInlineFiles tests complex scenarios mixing lists and inline files
func TestMixedListsAndInlineFiles(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "base"
    name: "Base Files"
    files:
      - src: "base/file1.txt"
        dest: "file1.txt"
      - src: "base/file2.txt"
        dest: "file2.txt"
      - src: "base/file3.txt"
        dest: "file3.txt"
  - id: "override"
    name: "Override Files"
    files:
      - src: "override/file2.txt"
        dest: "file2.txt"  # Overrides base
      - src: "override/file4.txt"
        dest: "file4.txt"
groups:
  - name: "Test"
    id: "test"
    source:
      repo: "org/template"
    targets:
      # Target 1: Lists only
      - repo: "org/service1"
        file_list_refs: ["base", "override"]

      # Target 2: Lists with inline override
      - repo: "org/service2"
        file_list_refs: ["base", "override"]
        files:
          - src: "custom/file1.txt"
            dest: "file1.txt"  # Overrides base
          - src: "custom/file5.txt"
            dest: "file5.txt"  # New file

      # Target 3: Empty lists should work
      - repo: "org/service3"
        file_list_refs: []
        files:
          - src: "only-inline.txt"
            dest: "only-inline.txt"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)
	require.Len(t, config.Groups[0].Targets, 3)

	// Target 1: Should have base files with override for file2
	target1 := config.Groups[0].Targets[0]
	require.Len(t, target1.Files, 4) // file1, file2(override), file3, file4

	files1 := make(map[string]string)
	for _, f := range target1.Files {
		files1[f.Dest] = f.Src
	}
	assert.Equal(t, "base/file1.txt", files1["file1.txt"])
	assert.Equal(t, "override/file2.txt", files1["file2.txt"], "Should be overridden by override list")
	assert.Equal(t, "base/file3.txt", files1["file3.txt"])
	assert.Equal(t, "override/file4.txt", files1["file4.txt"])

	// Target 2: Should have inline overrides
	target2 := config.Groups[0].Targets[1]
	require.Len(t, target2.Files, 5) // file1(custom), file2(override), file3, file4, file5

	files2 := make(map[string]string)
	for _, f := range target2.Files {
		files2[f.Dest] = f.Src
	}
	assert.Equal(t, "custom/file1.txt", files2["file1.txt"], "Should be overridden by inline")
	assert.Equal(t, "override/file2.txt", files2["file2.txt"])
	assert.Equal(t, "base/file3.txt", files2["file3.txt"])
	assert.Equal(t, "override/file4.txt", files2["file4.txt"])
	assert.Equal(t, "custom/file5.txt", files2["file5.txt"])

	// Target 3: Should only have inline file
	target3 := config.Groups[0].Targets[2]
	require.Len(t, target3.Files, 1)
	assert.Equal(t, "only-inline.txt", target3.Files[0].Src)
}

// TestListReferencesOrder tests that order of list references matters
func TestListReferencesOrder(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "list1"
    name: "List 1"
    files:
      - src: "v1/shared.txt"
        dest: "shared.txt"
      - src: "v1/unique1.txt"
        dest: "unique1.txt"
  - id: "list2"
    name: "List 2"
    files:
      - src: "v2/shared.txt"
        dest: "shared.txt"
      - src: "v2/unique2.txt"
        dest: "unique2.txt"
groups:
  - name: "Test"
    id: "test"
    source:
      repo: "org/template"
    targets:
      # Order matters - list2 overrides list1 for shared.txt
      - repo: "org/service1"
        file_list_refs: ["list1", "list2"]

      # Reversed order - list1 overrides list2 for shared.txt
      - repo: "org/service2"
        file_list_refs: ["list2", "list1"]
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	// Service1: list2 should win for shared.txt
	target1 := config.Groups[0].Targets[0]
	files1 := make(map[string]string)
	for _, f := range target1.Files {
		files1[f.Dest] = f.Src
	}
	assert.Equal(t, "v2/shared.txt", files1["shared.txt"], "list2 should override list1")
	assert.Equal(t, "v1/unique1.txt", files1["unique1.txt"])
	assert.Equal(t, "v2/unique2.txt", files1["unique2.txt"])

	// Service2: list1 should win for shared.txt
	target2 := config.Groups[0].Targets[1]
	files2 := make(map[string]string)
	for _, f := range target2.Files {
		files2[f.Dest] = f.Src
	}
	assert.Equal(t, "v1/shared.txt", files2["shared.txt"], "list1 should override list2")
	assert.Equal(t, "v1/unique1.txt", files2["unique1.txt"])
	assert.Equal(t, "v2/unique2.txt", files2["unique2.txt"])
}

// TestDirectoryListsWithComplexOverrides tests directory lists with complex override scenarios
func TestDirectoryListsWithComplexOverrides(t *testing.T) {
	yamlContent := `
version: 1
directory_lists:
  - id: "base-dirs"
    name: "Base Directories"
    directories:
      - src: ".github/workflows"
        dest: ".github/workflows"
        exclude: ["*.tmp"]
      - src: ".github/actions"
        dest: ".github/actions"
  - id: "enhanced-dirs"
    name: "Enhanced Directories"
    directories:
      - src: "enhanced/workflows"
        dest: ".github/workflows"
        exclude: ["*.tmp", "*.bak", "*.local"]
      - src: "docs"
        dest: "documentation"
groups:
  - name: "Test"
    id: "test"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        directory_list_refs: ["base-dirs", "enhanced-dirs"]
        directories:
          # Override actions directory
          - src: "custom/actions"
            dest: ".github/actions"
            exclude: ["test/*"]
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	target := config.Groups[0].Targets[0]
	require.Len(t, target.Directories, 3)

	dirMap := make(map[string]DirectoryMapping)
	for _, d := range target.Directories {
		dirMap[d.Dest] = d
	}

	// Workflows should be from enhanced-dirs
	workflows := dirMap[".github/workflows"]
	assert.Equal(t, "enhanced/workflows", workflows.Src)
	assert.Equal(t, []string{"*.tmp", "*.bak", "*.local"}, workflows.Exclude)

	// Actions should be from inline override
	actions := dirMap[".github/actions"]
	assert.Equal(t, "custom/actions", actions.Src)
	assert.Equal(t, []string{"test/*"}, actions.Exclude)

	// Documentation should be from enhanced-dirs
	docs := dirMap["documentation"]
	assert.Equal(t, "docs", docs.Src)
}

// TestNoListsDefinedButReferenced tests error when referencing non-existent lists
func TestNoListsDefinedButReferenced(t *testing.T) {
	yamlContent := `
version: 1
groups:
  - name: "Test"
    id: "test"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        file_list_refs: ["non-existent"]
`

	_, err := LoadFromReader(strings.NewReader(yamlContent))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list reference not found")
}

// TestEmptyFileListWithInlineFiles tests that empty file lists work correctly
func TestEmptyFileListWithInlineFiles(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "empty"
    name: "Empty List"
    files: []
groups:
  - name: "Test"
    id: "test"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service"
        file_list_refs: ["empty"]
        files:
          - src: "actual.txt"
            dest: "actual.txt"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	target := config.Groups[0].Targets[0]
	require.Len(t, target.Files, 1)
	assert.Equal(t, "actual.txt", target.Files[0].Src)
}

// TestMultipleTargetsUsingSameLists tests that multiple targets can reference same lists
func TestMultipleTargetsUsingSameLists(t *testing.T) {
	yamlContent := `
version: 1
file_lists:
  - id: "shared"
    name: "Shared Files"
    files:
      - src: "shared.txt"
        dest: "shared.txt"
groups:
  - name: "Test"
    id: "test"
    source:
      repo: "org/template"
    targets:
      - repo: "org/service1"
        file_list_refs: ["shared"]
        files:
          - src: "service1.txt"
            dest: "service1.txt"
      - repo: "org/service2"
        file_list_refs: ["shared"]
        files:
          - src: "service2.txt"
            dest: "service2.txt"
`

	config, err := LoadFromReader(strings.NewReader(yamlContent))
	require.NoError(t, err)

	// Both targets should have the shared file plus their own
	target1 := config.Groups[0].Targets[0]
	require.Len(t, target1.Files, 2)

	target2 := config.Groups[0].Targets[1]
	require.Len(t, target2.Files, 2)

	// Verify each has the shared file
	hasShared1 := false
	hasService1 := false
	for _, f := range target1.Files {
		if f.Dest == "shared.txt" {
			hasShared1 = true
		}
		if f.Dest == "service1.txt" {
			hasService1 = true
		}
	}
	assert.True(t, hasShared1, "Target1 should have shared file")
	assert.True(t, hasService1, "Target1 should have service1 file")

	hasShared2 := false
	hasService2 := false
	for _, f := range target2.Files {
		if f.Dest == "shared.txt" {
			hasShared2 = true
		}
		if f.Dest == "service2.txt" {
			hasService2 = true
		}
	}
	assert.True(t, hasShared2, "Target2 should have shared file")
	assert.True(t, hasService2, "Target2 should have service2 file")
}
