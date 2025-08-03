package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstaller(t *testing.T) {
	installer := NewInstaller("/test/repo", ".github/pre-commit")
	assert.NotNil(t, installer)
	assert.Equal(t, "/test/repo", installer.repoRoot)
	assert.Equal(t, ".github/pre-commit", installer.preCommitDir)
}

func TestInstaller_InstallHook(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git", "hooks")
	err := os.MkdirAll(gitDir, 0o750)
	require.NoError(t, err)

	installer := NewInstaller(tmpDir, ".github/pre-commit")

	// Test installing a hook
	err = installer.InstallHook("pre-commit", false)
	require.NoError(t, err)

	// Check that the hook was created
	hookPath := filepath.Join(gitDir, "pre-commit")
	info, err := os.Stat(hookPath)
	require.NoError(t, err)
	assert.NotEqual(t, 0, info.Mode()&0o111, "Hook should be executable")

	// Read the hook content
	content, err := os.ReadFile(hookPath) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Contains(t, string(content), "GoFortress Pre-commit Hook")
	assert.Contains(t, string(content), "gofortress-pre-commit")

	// Test installing again without force (should not error - already our hook)
	err = installer.InstallHook("pre-commit", false)
	require.NoError(t, err)

	// Test with a non-GoFortress hook
	err = os.WriteFile(hookPath, []byte("#!/bin/bash\necho 'other hook'"), 0o600)
	require.NoError(t, err)

	// Should return ErrExist without force
	err = installer.InstallHook("pre-commit", false)
	require.ErrorIs(t, err, os.ErrExist)

	// Should succeed with force
	err = installer.InstallHook("pre-commit", true)
	require.NoError(t, err)

	// Verify it was replaced
	content, err = os.ReadFile(hookPath) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Contains(t, string(content), "GoFortress Pre-commit Hook")
}

func TestInstaller_UninstallHook(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git", "hooks")
	err := os.MkdirAll(gitDir, 0o750)
	require.NoError(t, err)

	installer := NewInstaller(tmpDir, ".github/pre-commit")
	hookPath := filepath.Join(gitDir, "pre-commit")

	// Test uninstalling non-existent hook
	removed, err := installer.UninstallHook("pre-commit")
	require.NoError(t, err)
	assert.False(t, removed)

	// Install a GoFortress hook
	err = installer.InstallHook("pre-commit", false)
	require.NoError(t, err)

	// Uninstall it
	removed, err = installer.UninstallHook("pre-commit")
	require.NoError(t, err)
	assert.True(t, removed)

	// Verify it was removed
	_, err = os.Stat(hookPath)
	assert.True(t, os.IsNotExist(err))

	// Test with a non-GoFortress hook
	err = os.WriteFile(hookPath, []byte("#!/bin/bash\necho 'other hook'"), 0o600)
	require.NoError(t, err)

	// Should not remove non-GoFortress hook
	removed, err = installer.UninstallHook("pre-commit")
	require.NoError(t, err)
	assert.False(t, removed)

	// Verify it still exists
	_, err = os.Stat(hookPath)
	assert.NoError(t, err)
}

func TestInstaller_IsHookInstalled(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git", "hooks")
	err := os.MkdirAll(gitDir, 0o750)
	require.NoError(t, err)

	installer := NewInstaller(tmpDir, ".github/pre-commit")

	// Test with non-existent hook
	installed := installer.IsHookInstalled("pre-commit")
	assert.False(t, installed)

	// Install a GoFortress hook
	err = installer.InstallHook("pre-commit", false)
	require.NoError(t, err)

	// Should be installed
	installed = installer.IsHookInstalled("pre-commit")
	assert.True(t, installed)

	// Test with a non-GoFortress hook
	hookPath := filepath.Join(gitDir, "pre-commit")
	err = os.WriteFile(hookPath, []byte("#!/bin/bash\necho 'other hook'"), 0o600)
	require.NoError(t, err)

	// Should not be considered installed
	installed = installer.IsHookInstalled("pre-commit")
	assert.False(t, installed)
}

func TestHookScript(t *testing.T) {
	// Verify the hook script is properly formatted
	assert.True(t, strings.HasPrefix(hookScript, "#!/bin/bash"))
	assert.Contains(t, hookScript, "GoFortress Pre-commit Hook")
	assert.Contains(t, hookScript, "gofortress-pre-commit")
	assert.Contains(t, hookScript, "exec")
}
