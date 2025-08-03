package makewrap

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFumptCheck(t *testing.T) {
	check := NewFumptCheck()
	assert.NotNil(t, check)
	assert.IsType(t, &FumptCheck{}, check)
}

func TestFumptCheck(t *testing.T) {
	check := &FumptCheck{}

	assert.Equal(t, "fumpt", check.Name())
	assert.Equal(t, "Format code with gofumpt", check.Description())
}

func TestFumptCheck_FilterFiles(t *testing.T) {
	check := &FumptCheck{}

	files := []string{
		"main.go",
		"test.go",
		"doc.md",
		"Makefile",
		"test.txt",
		"pkg/foo.go",
	}

	filtered := check.FilterFiles(files)
	expected := []string{"main.go", "test.go", "pkg/foo.go"}
	assert.Equal(t, expected, filtered)
}

func TestFumptCheck_Run_NoMake(t *testing.T) {
	// Create a temporary directory without Makefile
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			t.Logf("Failed to restore directory: %v", chErr)
		}
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	check := &FumptCheck{repoRoot: tmpDir}
	ctx := context.Background()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find repository root")
}

func TestFumptCheck_Run_NoTarget(t *testing.T) {
	// Skip if make is not available
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not available")
	}

	// Create a temporary directory with Makefile but no fumpt target
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			t.Logf("Failed to restore directory: %v", chErr)
		}
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a Makefile without fumpt target
	makefile := `
test:
	@echo "test"
`
	err = os.WriteFile("Makefile", []byte(makefile), 0o600)
	require.NoError(t, err)

	check := &FumptCheck{repoRoot: tmpDir}
	ctx := context.Background()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find repository root")
}

func TestNewLintCheck(t *testing.T) {
	check := NewLintCheck()
	assert.NotNil(t, check)
	assert.IsType(t, &LintCheck{}, check)
}

func TestLintCheck(t *testing.T) {
	check := &LintCheck{}

	assert.Equal(t, "lint", check.Name())
	assert.Equal(t, "Run golangci-lint", check.Description())
}

func TestLintCheck_FilterFiles(t *testing.T) {
	check := &LintCheck{}

	files := []string{
		"main.go",
		"test.go",
		"doc.md",
		"Makefile",
		"test.txt",
		"pkg/foo.go",
	}

	filtered := check.FilterFiles(files)
	expected := []string{"main.go", "test.go", "pkg/foo.go"}
	assert.Equal(t, expected, filtered)
}

func TestLintCheck_Run_NoMake(t *testing.T) {
	// Create a temporary directory without Makefile
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			t.Logf("Failed to restore directory: %v", chErr)
		}
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	check := &LintCheck{repoRoot: tmpDir}
	ctx := context.Background()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find repository root")
}

func TestNewModTidyCheck(t *testing.T) {
	check := NewModTidyCheck()
	assert.NotNil(t, check)
	assert.IsType(t, &ModTidyCheck{}, check)
}

func TestModTidyCheck(t *testing.T) {
	check := &ModTidyCheck{}

	assert.Equal(t, "mod-tidy", check.Name())
	assert.Equal(t, "Ensure go.mod and go.sum are tidy", check.Description())
}

func TestModTidyCheck_FilterFiles(t *testing.T) {
	check := &ModTidyCheck{}

	files := []string{
		"main.go",
		"go.mod",
		"go.sum",
		"doc.md",
		"Makefile",
	}

	// Only returns go.mod and go.sum
	filtered := check.FilterFiles(files)
	expected := []string{"go.mod", "go.sum"}
	assert.Equal(t, expected, filtered)
}

func TestModTidyCheck_Run_NoGoMod(t *testing.T) {
	// Create a temporary directory without go.mod
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			t.Logf("Failed to restore directory: %v", chErr)
		}
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	check := &ModTidyCheck{repoRoot: tmpDir}
	ctx := context.Background()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find repository root")
}

func TestModTidyCheck_Run_NoMake(t *testing.T) {
	// Create a temporary directory with go.mod but no Makefile
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			t.Logf("Failed to restore directory: %v", chErr)
		}
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a go.mod file
	gomod := `module test

go 1.21
`
	err = os.WriteFile("go.mod", []byte(gomod), 0o600)
	require.NoError(t, err)

	check := &ModTidyCheck{repoRoot: tmpDir}
	ctx := context.Background()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find repository root")
}
