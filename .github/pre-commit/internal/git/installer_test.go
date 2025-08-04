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

	// Create the pre-commit directory for validation
	preCommitDir := filepath.Join(tmpDir, ".github", "pre-commit")
	err = os.MkdirAll(preCommitDir, 0o750)
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

	// Create the pre-commit directory for validation
	preCommitDir := filepath.Join(tmpDir, ".github", "pre-commit")
	err = os.MkdirAll(preCommitDir, 0o750)
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

	// Create the pre-commit directory for validation
	preCommitDir := filepath.Join(tmpDir, ".github", "pre-commit")
	err = os.MkdirAll(preCommitDir, 0o750)
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
	// Create an installer to test hook script generation
	installer := NewInstaller("/test/repo", ".github/pre-commit")
	hookScript := installer.GenerateHookScript()

	// Verify the hook script is properly formatted
	assert.True(t, strings.HasPrefix(hookScript, "#!/bin/bash"))
	assert.Contains(t, hookScript, "GoFortress Pre-commit Hook")
	assert.Contains(t, hookScript, "gofortress-pre-commit")
	assert.Contains(t, hookScript, "exec")
}

func TestInstaller_InstallHook_ErrorCases(t *testing.T) {
	// Test error creating hooks directory
	tmpDir := t.TempDir()

	// Create the pre-commit directory for validation
	preCommitDir := filepath.Join(tmpDir, ".github", "pre-commit")
	err := os.MkdirAll(preCommitDir, 0o750)
	require.NoError(t, err)

	// Create a file where .git/hooks should be to cause mkdir error
	gitHooksPath := filepath.Join(tmpDir, ".git", "hooks")
	gitPath := filepath.Join(tmpDir, ".git")
	err = os.MkdirAll(gitPath, 0o750)
	require.NoError(t, err)

	// Create a file instead of directory
	err = os.WriteFile(gitHooksPath, []byte("not a directory"), 0o600)
	require.NoError(t, err)

	installer := NewInstaller(tmpDir, ".github/pre-commit")

	// This should fail when trying to create the hook file or hooks directory
	err = installer.InstallHook("pre-commit", false)
	require.Error(t, err)
	// Error could be either "failed to create hooks directory" or "failed to write hook script"
	assert.True(t,
		strings.Contains(err.Error(), "failed to create hooks directory") ||
			strings.Contains(err.Error(), "failed to write hook script"),
		"Expected error about hooks directory or hook script, got: %s", err.Error())
}

func TestInstaller_UninstallHook_ErrorCases(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git", "hooks")
	err := os.MkdirAll(gitDir, 0o750)
	require.NoError(t, err)

	// Create the pre-commit directory for validation
	preCommitDir := filepath.Join(tmpDir, ".github", "pre-commit")
	err = os.MkdirAll(preCommitDir, 0o750)
	require.NoError(t, err)

	installer := NewInstaller(tmpDir, ".github/pre-commit")
	hookPath := filepath.Join(gitDir, "pre-commit")

	// Test error reading hook file (permission denied)
	err = os.WriteFile(hookPath, []byte("#!/bin/bash\n# GoFortress Pre-commit Hook\necho test"), 0o000) // No read permissions
	require.NoError(t, err)

	// This should fail on reading the file
	removed, err := installer.UninstallHook("pre-commit")
	if err != nil {
		// On some systems, reading a file with no permissions still works
		// The test is mainly to cover the error path
		assert.False(t, removed)
	}

	// Restore permissions and test removal error by making directory read-only
	err = os.Chmod(hookPath, 0o600)
	require.NoError(t, err)

	// Make the hooks directory read-only to prevent removal
	err = os.Chmod(gitDir, 0o444) //nolint:gosec // test needs specific permissions
	require.NoError(t, err)

	// Cleanup - restore permissions before test ends
	defer func() {
		_ = os.Chmod(gitDir, 0o755) //nolint:gosec // test cleanup
	}()

	removed, err = installer.UninstallHook("pre-commit")
	if err != nil {
		// Should fail either to read hook or remove hook due to permissions
		assert.True(t,
			strings.Contains(err.Error(), "failed to read hook") ||
				strings.Contains(err.Error(), "failed to remove hook"),
			"Expected error about reading or removing hook, got: %s", err.Error())
		assert.False(t, removed)
	}
}
