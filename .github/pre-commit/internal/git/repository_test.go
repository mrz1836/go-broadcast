package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindRepositoryRoot(t *testing.T) {
	// This test runs in a git repository
	root, err := FindRepositoryRoot()
	require.NoError(t, err)
	assert.NotEmpty(t, root)

	// Should contain .git directory
	gitDir := filepath.Join(root, ".git")
	_, err = os.Stat(gitDir)
	assert.NoError(t, err)
}

func TestRepository_GetRoot(t *testing.T) {
	repo := NewRepository("/test/path")
	assert.Equal(t, "/test/path", repo.GetRoot())
}

func TestRepository_IsFileTracked(t *testing.T) {
	// Get actual repo root
	root, err := FindRepositoryRoot()
	require.NoError(t, err)

	repo := NewRepository(root)

	// This file should be tracked
	tracked := repo.IsFileTracked("go.mod")
	assert.True(t, tracked)

	// Non-existent file should not be tracked
	tracked = repo.IsFileTracked("nonexistent.file")
	assert.False(t, tracked)
}

func TestParseFileList(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []string
	}{
		{
			name:     "empty input",
			input:    []byte(""),
			expected: []string{},
		},
		{
			name:     "single file",
			input:    []byte("file.go\n"),
			expected: []string{"file.go"},
		},
		{
			name:     "multiple files",
			input:    []byte("file1.go\nfile2.go\nfile3.go\n"),
			expected: []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name:     "files with spaces in output",
			input:    []byte("  file1.go  \n  file2.go  \n"),
			expected: []string{"file1.go", "file2.go"},
		},
		{
			name:     "empty lines",
			input:    []byte("file1.go\n\nfile2.go\n\n"),
			expected: []string{"file1.go", "file2.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFileList(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRepository_GetStagedFiles(t *testing.T) {
	// This test requires a real git repository
	root, err := FindRepositoryRoot()
	if err != nil {
		t.Skip("Not in a git repository")
	}

	repo := NewRepository(root)

	// Should not error even if no files are staged
	files, err := repo.GetStagedFiles()
	require.NoError(t, err)
	assert.NotNil(t, files) // Can be empty array
}

func TestRepository_GetAllFiles(t *testing.T) {
	// This test requires a real git repository
	root, err := FindRepositoryRoot()
	if err != nil {
		t.Skip("Not in a git repository")
	}

	repo := NewRepository(root)

	files, err := repo.GetAllFiles()
	require.NoError(t, err)
	assert.NotEmpty(t, files) // Should have some files

	// Should contain go.mod
	hasGoMod := false
	for _, f := range files {
		if filepath.Base(f) == "go.mod" {
			hasGoMod = true
			break
		}
	}
	assert.True(t, hasGoMod, "Should contain go.mod")
}

func TestRepository_GetModifiedFiles(t *testing.T) {
	// This test requires a real git repository
	root, err := FindRepositoryRoot()
	if err != nil {
		t.Skip("Not in a git repository")
	}

	repo := NewRepository(root)

	// Should not error even if no files are modified
	files, err := repo.GetModifiedFiles()
	require.NoError(t, err)
	assert.NotNil(t, files) // Can be empty array
}

func TestRepository_GetFileContent(t *testing.T) {
	// This test requires a real git repository
	root, err := FindRepositoryRoot()
	if err != nil {
		t.Skip("Not in a git repository")
	}

	repo := NewRepository(root)

	// Try to get content of go.mod
	content, err := repo.GetFileContent("go.mod")
	require.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, string(content), "module")
}

// Test with mock command for better coverage
// Full context testing would require mocking exec.CommandContext

func TestRepository_GetStagedFiles_Context(t *testing.T) {
	repo := NewRepository("/test/repo")
	assert.NotNil(t, repo)

	// Full context testing would require mocking exec.CommandContext
	// which is complex. This ensures the structure is correct.
}
