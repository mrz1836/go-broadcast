package sync

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// Static errors for module detection
var (
	ErrNoModuleDirective = errors.New("go.mod does not contain a module directive")
	ErrNoGoModFound      = errors.New("no go.mod found in directory or parent directories")
)

// ModuleInfo contains information about a Go module
type ModuleInfo struct {
	Name    string // Module name from go.mod
	Path    string // Path to the module directory
	Version string // Current version if specified in go.mod
	GoMod   string // Path to go.mod file
}

// ModuleDetector detects and analyzes Go modules in directories
type ModuleDetector struct {
	logger *logrus.Logger
}

// NewModuleDetector creates a new module detector
func NewModuleDetector(logger *logrus.Logger) *ModuleDetector {
	return &ModuleDetector{
		logger: logger,
	}
}

// IsGoModule checks if a directory contains a Go module
func (d *ModuleDetector) IsGoModule(dir string) bool {
	goModPath := filepath.Join(dir, "go.mod")
	info, err := os.Stat(goModPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DetectModule detects and parses module information from a directory
func (d *ModuleDetector) DetectModule(dir string) (*ModuleInfo, error) {
	if !d.IsGoModule(dir) {
		return nil, nil //nolint:nilnil // Not a module is a valid state, not an error
	}

	goModPath := filepath.Join(dir, "go.mod")
	moduleInfo, err := d.parseGoMod(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	moduleInfo.Path = dir
	moduleInfo.GoMod = goModPath

	d.logger.WithFields(logrus.Fields{
		"module":  moduleInfo.Name,
		"path":    moduleInfo.Path,
		"version": moduleInfo.Version,
	}).Debug("Detected Go module")

	return moduleInfo, nil
}

// DetectModules finds all Go modules in a directory tree
func (d *ModuleDetector) DetectModules(rootDir string) ([]*ModuleInfo, error) {
	var modules []*ModuleInfo

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Check if this directory contains a module
		if moduleInfo, err := d.DetectModule(path); err != nil {
			d.logger.WithError(err).WithField("path", path).Warn("Failed to detect module")
		} else if moduleInfo != nil {
			modules = append(modules, moduleInfo)
			// Skip subdirectories of modules (modules can't be nested)
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory tree: %w", err)
	}

	d.logger.WithField("count", len(modules)).Info("Detected Go modules")
	return modules, nil
}

// parseGoMod parses a go.mod file to extract module information
func (d *ModuleDetector) parseGoMod(goModPath string) (*ModuleInfo, error) {
	file, err := os.Open(goModPath) //nolint:gosec // Input is validated by caller
	if err != nil {
		return nil, fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer func() { _ = file.Close() }()

	info := &ModuleInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "//") || line == "" {
			continue
		}

		// Parse module directive
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				info.Name = parts[1]
			}
		}

		// Check for retract directive with version (indicates current version)
		// This is a simple heuristic; real version would come from git tags
		// or be specified in the module config

		// Stop at the first require block
		if strings.HasPrefix(line, "require") {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan go.mod: %w", err)
	}

	if info.Name == "" {
		return nil, ErrNoModuleDirective
	}

	return info, nil
}

// GetModuleName returns the module name from a go.mod file
func (d *ModuleDetector) GetModuleName(goModPath string) (string, error) {
	info, err := d.parseGoMod(goModPath)
	if err != nil {
		return "", err
	}
	return info.Name, nil
}

// FindGoModInParents searches for a go.mod file in parent directories
func (d *ModuleDetector) FindGoModInParents(startDir string) (string, error) {
	absPath, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	current := absPath
	for {
		goModPath := filepath.Join(current, "go.mod")
		if info, err := os.Stat(goModPath); err == nil && !info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root directory
			break
		}
		current = parent
	}

	return "", fmt.Errorf("%w: %s", ErrNoGoModFound, startDir)
}
