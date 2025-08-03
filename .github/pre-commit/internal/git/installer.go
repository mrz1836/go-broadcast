package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const hookScript = `#!/bin/bash
# GoFortress Pre-commit Hook
# This hook is managed by GoFortress pre-commit system

# Find the gofortress-pre-commit binary
BINARY="gofortress-pre-commit"

# Check if binary is in PATH
if command -v "$BINARY" >/dev/null 2>&1; then
    exec "$BINARY" run
else
    # Try to find it in common locations
    for dir in "$(go env GOPATH)/bin" ".github/pre-commit" "./bin"; do
        if [ -x "$dir/$BINARY" ]; then
            exec "$dir/$BINARY" run
        fi
    done
    
    echo "Error: gofortress-pre-commit not found in PATH or common locations"
    echo "Please install it with: cd .github/pre-commit && make install"
    exit 1
fi
`

// Installer handles git hook installation
type Installer struct {
	repoRoot     string
	preCommitDir string
}

// NewInstaller creates a new hook installer
func NewInstaller(repoRoot, preCommitDir string) *Installer {
	return &Installer{
		repoRoot:     repoRoot,
		preCommitDir: preCommitDir,
	}
}

// InstallHook installs a git hook
func (i *Installer) InstallHook(hookType string, force bool) error {
	hookPath := filepath.Join(i.repoRoot, ".git", "hooks", hookType)

	// Check if hook already exists
	if _, err := os.Stat(hookPath); err == nil && !force {
		// Check if it's our hook
		content, readErr := os.ReadFile(hookPath) //nolint:gosec // Path is validated
		if readErr == nil && strings.Contains(string(content), "GoFortress Pre-commit Hook") {
			return nil // Already installed
		}
		return os.ErrExist
	}

	// Create hooks directory if it doesn't exist
	hooksDir := filepath.Dir(hookPath)
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Write hook script
	if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil { //nolint:gosec // Hook script must be executable
		return fmt.Errorf("failed to write hook script: %w", err)
	}

	return nil
}

// UninstallHook removes a git hook if it was installed by us
func (i *Installer) UninstallHook(hookType string) (bool, error) {
	hookPath := filepath.Join(i.repoRoot, ".git", "hooks", hookType)

	// Check if hook exists
	content, err := os.ReadFile(hookPath) //nolint:gosec // Path is validated
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // Hook doesn't exist
		}
		return false, fmt.Errorf("failed to read hook: %w", err)
	}

	// Check if it's our hook
	if !strings.Contains(string(content), "GoFortress Pre-commit Hook") {
		return false, nil // Not our hook
	}

	// Remove the hook
	if err := os.Remove(hookPath); err != nil {
		return false, fmt.Errorf("failed to remove hook: %w", err)
	}

	return true, nil
}

// IsHookInstalled checks if a hook is installed
func (i *Installer) IsHookInstalled(hookType string) bool {
	hookPath := filepath.Join(i.repoRoot, ".git", "hooks", hookType)

	content, err := os.ReadFile(hookPath) //nolint:gosec // Path is validated
	if err != nil {
		return false
	}

	return strings.Contains(string(content), "GoFortress Pre-commit Hook")
}
