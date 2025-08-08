package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleDetector_IsGoModule(t *testing.T) {
	logger := logrus.New()
	detector := NewModuleDetector(logger)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "module-detector-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	t.Run("directory with go.mod is a module", func(t *testing.T) {
		// Create go.mod file
		goModPath := filepath.Join(tempDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test.com/example\n\ngo 1.21\n"), 0o600)
		require.NoError(t, err)

		assert.True(t, detector.IsGoModule(tempDir))
	})

	t.Run("directory without go.mod is not a module", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0o750)
		require.NoError(t, err)

		assert.False(t, detector.IsGoModule(emptyDir))
	})

	t.Run("non-existent directory is not a module", func(t *testing.T) {
		assert.False(t, detector.IsGoModule(filepath.Join(tempDir, "non-existent")))
	})
}

func TestModuleDetector_DetectModule(t *testing.T) {
	logger := logrus.New()
	detector := NewModuleDetector(logger)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "module-detector-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	t.Run("detects module with valid go.mod", func(t *testing.T) {
		moduleDir := filepath.Join(tempDir, "module1")
		err := os.MkdirAll(moduleDir, 0o750)
		require.NoError(t, err)

		// Create go.mod file
		goModContent := `module github.com/example/testmodule

go 1.21

require (
	github.com/sirupsen/logrus v1.9.0
)
`
		goModPath := filepath.Join(moduleDir, "go.mod")
		err = os.WriteFile(goModPath, []byte(goModContent), 0o600)
		require.NoError(t, err)

		moduleInfo, err := detector.DetectModule(moduleDir)
		require.NoError(t, err)
		require.NotNil(t, moduleInfo)

		assert.Equal(t, "github.com/example/testmodule", moduleInfo.Name)
		assert.Equal(t, moduleDir, moduleInfo.Path)
		assert.Equal(t, goModPath, moduleInfo.GoMod)
	})

	t.Run("returns nil for non-module directory", func(t *testing.T) {
		nonModuleDir := filepath.Join(tempDir, "non-module")
		err := os.MkdirAll(nonModuleDir, 0o750)
		require.NoError(t, err)

		moduleInfo, err := detector.DetectModule(nonModuleDir)
		require.NoError(t, err)
		assert.Nil(t, moduleInfo)
	})

	t.Run("handles invalid go.mod gracefully", func(t *testing.T) {
		invalidDir := filepath.Join(tempDir, "invalid")
		err := os.MkdirAll(invalidDir, 0o750)
		require.NoError(t, err)

		// Create invalid go.mod file (missing module directive)
		goModPath := filepath.Join(invalidDir, "go.mod")
		err = os.WriteFile(goModPath, []byte("go 1.21\n"), 0o600)
		require.NoError(t, err)

		moduleInfo, err := detector.DetectModule(invalidDir)
		require.Error(t, err)
		assert.Nil(t, moduleInfo)
		assert.Contains(t, err.Error(), "does not contain a module directive")
	})
}

func TestModuleDetector_DetectModules(t *testing.T) {
	logger := logrus.New()
	detector := NewModuleDetector(logger)

	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "module-detector-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create multiple modules
	module1Dir := filepath.Join(tempDir, "module1")
	err = os.MkdirAll(module1Dir, 0o750)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(module1Dir, "go.mod"), []byte("module test.com/module1\ngo 1.21\n"), 0o600)
	require.NoError(t, err)

	module2Dir := filepath.Join(tempDir, "subdir", "module2")
	err = os.MkdirAll(module2Dir, 0o750)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(module2Dir, "go.mod"), []byte("module test.com/module2\ngo 1.21\n"), 0o600)
	require.NoError(t, err)

	// Create non-module directory
	nonModuleDir := filepath.Join(tempDir, "non-module")
	err = os.MkdirAll(nonModuleDir, 0o750)
	require.NoError(t, err)

	t.Run("detects all modules in directory tree", func(t *testing.T) {
		modules, err := detector.DetectModules(tempDir)
		require.NoError(t, err)
		require.Len(t, modules, 2)

		// Check module names
		moduleNames := make(map[string]bool)
		for _, m := range modules {
			moduleNames[m.Name] = true
		}
		assert.True(t, moduleNames["test.com/module1"])
		assert.True(t, moduleNames["test.com/module2"])
	})

	t.Run("skips subdirectories of modules", func(t *testing.T) {
		// Create a subdirectory within module1 with its own go.mod (should be skipped)
		nestedDir := filepath.Join(module1Dir, "nested")
		err := os.MkdirAll(nestedDir, 0o750)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(nestedDir, "go.mod"), []byte("module test.com/nested\ngo 1.21\n"), 0o600)
		require.NoError(t, err)

		modules, err := detector.DetectModules(tempDir)
		require.NoError(t, err)
		// Should still be 2 modules (nested is skipped)
		assert.Len(t, modules, 2)
	})
}

func TestModuleDetector_GetModuleName(t *testing.T) {
	logger := logrus.New()
	detector := NewModuleDetector(logger)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "module-detector-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	t.Run("extracts module name from go.mod", func(t *testing.T) {
		goModPath := filepath.Join(tempDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module github.com/test/example\ngo 1.21\n"), 0o600)
		require.NoError(t, err)

		name, err := detector.GetModuleName(goModPath)
		require.NoError(t, err)
		assert.Equal(t, "github.com/test/example", name)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		name, err := detector.GetModuleName(filepath.Join(tempDir, "non-existent.mod"))
		require.Error(t, err)
		assert.Empty(t, name)
	})
}

func TestModuleDetector_FindGoModInParents(t *testing.T) {
	logger := logrus.New()
	detector := NewModuleDetector(logger)

	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "module-detector-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create go.mod in root
	rootGoMod := filepath.Join(tempDir, "go.mod")
	err = os.WriteFile(rootGoMod, []byte("module test.com/root\ngo 1.21\n"), 0o600)
	require.NoError(t, err)

	// Create nested directories
	nestedDir := filepath.Join(tempDir, "pkg", "nested", "deep")
	err = os.MkdirAll(nestedDir, 0o750)
	require.NoError(t, err)

	t.Run("finds go.mod in parent directory", func(t *testing.T) {
		moduleRoot, err := detector.FindGoModInParents(nestedDir)
		require.NoError(t, err)
		assert.Equal(t, tempDir, moduleRoot)
	})

	t.Run("finds go.mod in current directory", func(t *testing.T) {
		moduleRoot, err := detector.FindGoModInParents(tempDir)
		require.NoError(t, err)
		assert.Equal(t, tempDir, moduleRoot)
	})

	t.Run("returns error when no go.mod found", func(t *testing.T) {
		noModuleDir, err := os.MkdirTemp("", "no-module-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(noModuleDir) }()

		moduleRoot, err := detector.FindGoModInParents(noModuleDir)
		require.Error(t, err)
		assert.Empty(t, moduleRoot)
		assert.Contains(t, err.Error(), "no go.mod found")
	})
}
