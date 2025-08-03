package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	prerrors "github.com/mrz1836/go-broadcast/pre-commit/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhitespaceCheck(t *testing.T) {
	check := NewWhitespaceCheck()

	assert.Equal(t, "whitespace", check.Name())
	assert.Equal(t, "Fix trailing whitespace", check.Description())
}

func TestWhitespaceCheck_Run(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	goodFile := filepath.Join(tmpDir, "good.txt")
	err := os.WriteFile(goodFile, []byte("no trailing spaces\nclean line\n"), 0o600)
	require.NoError(t, err)

	badFile := filepath.Join(tmpDir, "bad.txt")
	err = os.WriteFile(badFile, []byte("trailing spaces   \nclean line\nmore spaces \t\n"), 0o600)
	require.NoError(t, err)

	check := NewWhitespaceCheck()
	ctx := context.Background()

	// Test with good file
	err = check.Run(ctx, []string{goodFile})
	require.NoError(t, err)

	// Test with bad file
	err = check.Run(ctx, []string{badFile})
	require.Error(t, err)
	require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)

	// Verify file was fixed
	content, err := os.ReadFile(badFile) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "trailing spaces\nclean line\nmore spaces\n", string(content))

	// Test with no files
	err = check.Run(ctx, []string{})
	require.NoError(t, err)

	// Test with non-existent file
	err = check.Run(ctx, []string{"/nonexistent/file.txt"})
	assert.Error(t, err)
}

func TestWhitespaceCheck_BinaryFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a binary file
	binaryFile := filepath.Join(tmpDir, "binary.bin")
	err := os.WriteFile(binaryFile, []byte{0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}, 0o600)
	require.NoError(t, err)

	check := NewWhitespaceCheck()
	ctx := context.Background()

	// Filter files first (as the runner would do)
	filteredFiles := check.FilterFiles([]string{binaryFile})
	assert.Empty(t, filteredFiles, "Binary file should be filtered out")

	// Should skip binary files when filtered
	err = check.Run(ctx, filteredFiles)
	assert.NoError(t, err)
}

func TestEOFCheck(t *testing.T) {
	check := NewEOFCheck()

	assert.Equal(t, "eof", check.Name())
	assert.Equal(t, "Ensure files end with newline", check.Description())
}

func TestEOFCheck_Run(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	goodFile := filepath.Join(tmpDir, "good.txt")
	err := os.WriteFile(goodFile, []byte("content\n"), 0o600)
	require.NoError(t, err)

	badFile := filepath.Join(tmpDir, "bad.txt")
	err = os.WriteFile(badFile, []byte("no newline at end"), 0o600)
	require.NoError(t, err)

	emptyFile := filepath.Join(tmpDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte(""), 0o600)
	require.NoError(t, err)

	check := NewEOFCheck()
	ctx := context.Background()

	// Test with good file
	err = check.Run(ctx, []string{goodFile})
	require.NoError(t, err)

	// Test with bad file
	err = check.Run(ctx, []string{badFile})
	require.Error(t, err)
	require.ErrorIs(t, err, prerrors.ErrEOFIssues)

	// Verify file was fixed
	content, err := os.ReadFile(badFile) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "no newline at end\n", string(content))

	// Test with empty file (should be skipped)
	err = check.Run(ctx, []string{emptyFile})
	require.NoError(t, err)

	// Test with mixed files
	// Reset bad file
	err = os.WriteFile(badFile, []byte("no newline"), 0o600)
	require.NoError(t, err)

	err = check.Run(ctx, []string{goodFile, badFile, emptyFile})
	require.Error(t, err)

	// Test with no files
	err = check.Run(ctx, []string{})
	require.NoError(t, err)

	// Test with non-existent file
	err = check.Run(ctx, []string{"/nonexistent/file.txt"})
	assert.Error(t, err)
}

func TestEOFCheck_BinaryFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a binary file
	binaryFile := filepath.Join(tmpDir, "binary.bin")
	err := os.WriteFile(binaryFile, []byte{0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}, 0o600)
	require.NoError(t, err)

	check := NewEOFCheck()
	ctx := context.Background()

	// Filter files first (as the runner would do)
	filteredFiles := check.FilterFiles([]string{binaryFile})
	assert.Empty(t, filteredFiles, "Binary file should be filtered out")

	// Should skip binary files when filtered
	err = check.Run(ctx, filteredFiles)
	assert.NoError(t, err)
}

// isBinary is not exported, so we test it indirectly through the checks
// Binary files are tested in TestWhitespaceCheck_BinaryFile and TestEOFCheck_BinaryFile
