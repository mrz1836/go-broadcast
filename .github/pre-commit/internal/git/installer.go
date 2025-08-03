package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/config"
)

// hookScriptTemplate is the template for generating git hook scripts
const hookScriptTemplate = `#!/bin/bash
# GoFortress Pre-commit Hook
# This hook is managed by GoFortress pre-commit system
# Generated automatically - do not edit manually

# Configuration
REPO_ROOT="%s"
PRE_COMMIT_DIR="%s"
BINARY_NAME="gofortress-pre-commit"
CONFIG_FILE="$REPO_ROOT/.github/.env.shared"

# CI Environment Detection
if [[ -n "$CI" || -n "$GITHUB_ACTIONS" || -n "$GITLAB_CI" || -n "$JENKINS_URL" ]]; then
    CI_ENV=true
else
    CI_ENV=false
fi

# Check if pre-commit system is enabled
if [[ -f "$CONFIG_FILE" ]]; then
    # Source config to check if enabled
    if grep -q "^ENABLE_PRE_COMMIT_SYSTEM=false" "$CONFIG_FILE" 2>/dev/null; then
        if [[ "$CI_ENV" != "true" ]]; then
            echo "GoFortress pre-commit system is disabled (ENABLE_PRE_COMMIT_SYSTEM=false)"
        fi
        exit 0
    fi
fi

# Find the gofortress-pre-commit binary
BINARY_PATH=""

# Search locations in order of preference
SEARCH_PATHS=(
    "$PRE_COMMIT_DIR/$BINARY_NAME"  # Built binary in pre-commit dir
    "$(command -v $BINARY_NAME 2>/dev/null)"  # In PATH
    "$(go env GOPATH 2>/dev/null)/bin/$BINARY_NAME"  # GOPATH/bin
    "./bin/$BINARY_NAME"  # Local bin directory
    "$PRE_COMMIT_DIR/cmd/gofortress-pre-commit/$BINARY_NAME"  # Development location
)

for path in "${SEARCH_PATHS[@]}"; do
    if [[ -n "$path" && -x "$path" ]]; then
        BINARY_PATH="$path"
        break
    fi
done

if [[ -z "$BINARY_PATH" ]]; then
    echo "Error: gofortress-pre-commit binary not found"
    echo "Searched locations:"
    for path in "${SEARCH_PATHS[@]}"; do
        if [[ -n "$path" ]]; then
            echo "  - $path"
        fi
    done
    echo ""
    echo "To fix this issue:"
    echo "  1. Build the binary: cd $PRE_COMMIT_DIR && go build -o gofortress-pre-commit ./cmd/gofortress-pre-commit"
    echo "  2. Or install to PATH: cd $PRE_COMMIT_DIR && go install ./cmd/gofortress-pre-commit"
    echo "  3. Or run: make install (if Makefile exists)"
    exit 1
fi

# Change to repository root for execution
cd "$REPO_ROOT" || {
    echo "Error: Could not change to repository root: $REPO_ROOT"
    exit 1
}

# Execute the pre-commit system
# Pass through environment variables including SKIP
exec "$BINARY_PATH" run
`

// Installer handles git hook installation
type Installer struct {
	repoRoot     string
	preCommitDir string
	config       *config.Config
}

// NewInstaller creates a new hook installer
func NewInstaller(repoRoot, preCommitDir string) *Installer {
	return &Installer{
		repoRoot:     repoRoot,
		preCommitDir: preCommitDir,
	}
}

// NewInstallerWithConfig creates a new hook installer with configuration
func NewInstallerWithConfig(repoRoot, preCommitDir string, cfg *config.Config) *Installer {
	return &Installer{
		repoRoot:     repoRoot,
		preCommitDir: preCommitDir,
		config:       cfg,
	}
}

// InstallHook installs a git hook with enhanced validation and conflict resolution
func (i *Installer) InstallHook(hookType string, force bool) error {
	// Pre-installation validation
	if err := i.validateInstallation(hookType); err != nil {
		return fmt.Errorf("installation validation failed: %w", err)
	}

	hookPath := filepath.Join(i.repoRoot, ".git", "hooks", hookType)

	// Handle existing hooks
	if err := i.handleExistingHook(hookPath, force); err != nil {
		return err
	}

	// Create hooks directory if it doesn't exist
	hooksDir := filepath.Dir(hookPath)
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Generate dynamic hook script
	hookScript := i.GenerateHookScript()

	// Write hook script
	if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil { //nolint:gosec // Hook script must be executable
		return fmt.Errorf("failed to write hook script: %w", err)
	}

	// Post-installation verification
	if err := i.verifyInstallation(hookPath); err != nil {
		return fmt.Errorf("installation verification failed: %w", err)
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

	// Check for and restore backup if it exists
	if err := i.restoreBackupIfExists(hookPath); err != nil {
		// Log warning but don't fail uninstall
		fmt.Fprintf(os.Stderr, "Warning: failed to restore backup hook: %v\n", err)
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

// validateInstallation performs pre-installation validation
func (i *Installer) validateInstallation(hookType string) error {
	// Validate git repository
	gitDir := filepath.Join(i.repoRoot, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", i.repoRoot)
	}

	// Validate hook type
	validHookTypes := map[string]bool{
		"pre-commit":  true,
		"pre-push":    true,
		"commit-msg":  true,
		"post-commit": true,
	}
	if !validHookTypes[hookType] {
		return fmt.Errorf("unsupported hook type: %s", hookType)
	}

	// Validate pre-commit directory exists
	preCommitPath := i.preCommitDir
	if !filepath.IsAbs(preCommitPath) {
		preCommitPath = filepath.Join(i.repoRoot, i.preCommitDir)
	}
	if _, err := os.Stat(preCommitPath); os.IsNotExist(err) {
		return fmt.Errorf("pre-commit directory does not exist: %s", preCommitPath)
	}

	// Validate configuration if available
	if i.config != nil {
		if err := i.config.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	return nil
}

// handleExistingHook manages existing hook conflicts
func (i *Installer) handleExistingHook(hookPath string, force bool) error {
	if _, err := os.Stat(hookPath); err == nil {
		// Hook exists, read content
		content, readErr := os.ReadFile(hookPath) //nolint:gosec // Path is validated
		if readErr != nil {
			return fmt.Errorf("failed to read existing hook: %w", readErr)
		}

		// Check if it's our hook
		if strings.Contains(string(content), "GoFortress Pre-commit Hook") {
			// It's our hook, update it (this is safe)
			return nil
		}

		// It's not our hook
		if !force {
			return fmt.Errorf("%w: %s (use --force to overwrite)", os.ErrExist, hookPath)
		}

		// Backup existing hook if force is used
		backupPath := hookPath + ".backup." + fmt.Sprintf("%d", os.Getpid())
		if err := os.Rename(hookPath, backupPath); err != nil {
			return fmt.Errorf("failed to backup existing hook: %w", err)
		}
	}

	return nil
}

// GenerateHookScript creates a dynamic hook script based on current environment
func (i *Installer) GenerateHookScript() string {
	return fmt.Sprintf(hookScriptTemplate, i.repoRoot, i.preCommitDir)
}

// verifyInstallation checks that the installation was successful
func (i *Installer) verifyInstallation(hookPath string) error {
	// Check file exists and is executable
	info, err := os.Stat(hookPath)
	if err != nil {
		return fmt.Errorf("hook file not found after installation: %w", err)
	}

	// Check permissions
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("hook file is not executable: %s", hookPath)
	}

	// Check content contains our marker
	content, err := os.ReadFile(hookPath) //nolint:gosec // Path is validated
	if err != nil {
		return fmt.Errorf("failed to read installed hook: %w", err)
	}

	if !strings.Contains(string(content), "GoFortress Pre-commit Hook") {
		return fmt.Errorf("installed hook does not contain expected marker")
	}

	return nil
}

// restoreBackupIfExists restores a backed up hook if one exists
func (i *Installer) restoreBackupIfExists(hookPath string) error {
	// Look for backup files
	dir := filepath.Dir(hookPath)
	base := filepath.Base(hookPath)
	pattern := base + ".backup.*"

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return fmt.Errorf("failed to search for backup files: %w", err)
	}

	if len(matches) == 0 {
		return nil // No backup to restore
	}

	// Use the most recent backup (last in sorted order)
	backupPath := matches[len(matches)-1]

	// Restore the backup
	if err := os.Rename(backupPath, hookPath); err != nil {
		return fmt.Errorf("failed to restore backup from %s: %w", backupPath, err)
	}

	return nil
}

// GetInstallationStatus returns detailed information about hook installation status
func (i *Installer) GetInstallationStatus(hookType string) (*InstallationStatus, error) {
	hookPath := filepath.Join(i.repoRoot, ".git", "hooks", hookType)
	status := &InstallationStatus{
		HookType: hookType,
		HookPath: hookPath,
	}

	// Check if hook file exists
	info, err := os.Stat(hookPath)
	if os.IsNotExist(err) {
		status.Installed = false
		status.Message = "Hook not installed"
		return status, nil
	}
	if err != nil {
		return status, fmt.Errorf("failed to stat hook file: %w", err)
	}

	status.Executable = info.Mode()&0o111 != 0
	status.FileMode = info.Mode()
	status.ModTime = info.ModTime()

	// Read and analyze content
	content, err := os.ReadFile(hookPath) //nolint:gosec // Path is validated
	if err != nil {
		return status, fmt.Errorf("failed to read hook file: %w", err)
	}

	status.IsOurHook = strings.Contains(string(content), "GoFortress Pre-commit Hook")
	if status.IsOurHook {
		status.Installed = true
		status.Message = "GoFortress pre-commit hook installed and ready"
		if !status.Executable {
			status.Message += " (warning: not executable)"
		}
	} else {
		status.Installed = false
		status.Message = "Different hook installed (not GoFortress)"
		status.ConflictingHook = true
	}

	return status, nil
}

// InstallationStatus provides detailed information about hook installation
type InstallationStatus struct {
	HookType        string
	HookPath        string
	Installed       bool
	IsOurHook       bool
	Executable      bool
	ConflictingHook bool
	FileMode        os.FileMode
	ModTime         time.Time
	Message         string
}
