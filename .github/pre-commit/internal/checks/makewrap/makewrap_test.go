package makewrap

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/pre-commit/internal/shared"
)

func TestNewFumptCheck(t *testing.T) {
	check := NewFumptCheck()
	assert.NotNil(t, check)
	assert.IsType(t, &FumptCheck{}, check)
}

func TestFumptCheck(t *testing.T) {
	check := &FumptCheck{}

	assert.Equal(t, "fumpt", check.Name())
	assert.Equal(t, "Format Go code with gofumpt", check.Description())
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

	// Initialize git repository
	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

	check := NewFumptCheck()
	ctx := context.Background()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	// The new implementation should return a CheckError for missing tools
	assert.Contains(t, err.Error(), "gofumpt not found")
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

	// Initialize git repository
	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

	// Create a Makefile without fumpt target
	makefile := `
test:
	@echo "test"
`
	err = os.WriteFile("Makefile", []byte(makefile), 0o600)
	require.NoError(t, err)

	check := NewFumptCheck()
	ctx := context.Background()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	// The new implementation should return a CheckError for missing make target or tool
	assert.Contains(t, err.Error(), "not found")
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

	check := NewLintCheck()
	ctx := context.Background()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	// Lint check will fail to find repository root first
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

	check := NewModTidyCheck()
	ctx := context.Background()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	// ModTidy check will fail to find repository root first
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

	check := NewModTidyCheck()
	ctx := context.Background()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	// ModTidy check will fail to find repository root first
	assert.Contains(t, err.Error(), "failed to find repository root")
}

// Comprehensive test suites for each check type

type FumptCheckTestSuite struct {
	suite.Suite

	tempDir string
	oldDir  string
}

func TestFumptCheckSuite(t *testing.T) {
	suite.Run(t, new(FumptCheckTestSuite))
}

func (s *FumptCheckTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "fumpt_test_*")
	s.Require().NoError(err)

	s.oldDir, err = os.Getwd()
	s.Require().NoError(err)

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Initialize git repo
	s.initGitRepo()
}

func (s *FumptCheckTestSuite) TearDownTest() {
	if s.oldDir != "" {
		err := os.Chdir(s.oldDir)
		s.Require().NoError(err)
	}
	if s.tempDir != "" {
		err := os.RemoveAll(s.tempDir)
		s.Require().NoError(err)
	}
}

func (s *FumptCheckTestSuite) initGitRepo() {
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "init").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())
}

func (s *FumptCheckTestSuite) TestNewFumptCheckWithSharedContext() {
	sharedCtx := shared.NewContext()
	check := NewFumptCheckWithSharedContext(sharedCtx)
	s.NotNil(check)
	s.Equal(sharedCtx, check.sharedCtx)
}

func (s *FumptCheckTestSuite) TestNewFumptCheckWithConfig() {
	sharedCtx := shared.NewContext()
	timeout := 30 * time.Second
	check := NewFumptCheckWithConfig(sharedCtx, timeout)
	s.NotNil(check)
	s.Equal(sharedCtx, check.sharedCtx)
	s.Equal(timeout, check.timeout)
}

func (s *FumptCheckTestSuite) TestRunMakeFumpt() {
	// Skip if make is not available
	if _, err := exec.LookPath("make"); err != nil {
		s.T().Skip("make not available")
	}

	// Create a Makefile that doesn't require gofumpt to be installed
	makefileContent := `fumpt:
	@echo "Running fumpt check"
	@echo "No gofumpt issues found"
`
	err := os.WriteFile("Makefile", []byte(makefileContent), 0o600)
	s.Require().NoError(err)

	check := NewFumptCheck()
	err = check.Run(context.Background(), []string{"test.go"})

	// Should succeed with make target available
	s.NoError(err)
}

func (s *FumptCheckTestSuite) TestRunDirectFumpt() {
	// Skip if gofumpt is not available
	if _, err := exec.LookPath("gofumpt"); err != nil {
		s.T().Skip("gofumpt not available")
	}

	// Create a Go file that needs formatting
	goFile := `package main

import "fmt"

func main() {
fmt.Println("hello")
}
`
	err := os.WriteFile("main.go", []byte(goFile), 0o600)
	s.Require().NoError(err)

	check := NewFumptCheck()
	err = check.Run(context.Background(), []string{"main.go"})

	// Should succeed when gofumpt is available
	s.NoError(err)
}

func (s *FumptCheckTestSuite) TestRunWithTimeout() {
	timeout := 1 * time.Millisecond // Very short timeout
	check := NewFumptCheckWithConfig(shared.NewContext(), timeout)

	// Create a Makefile with a slow target
	makefileContent := `fumpt:
	@sleep 10
`
	err := os.WriteFile("Makefile", []byte(makefileContent), 0o600)
	s.Require().NoError(err)

	err = check.Run(context.Background(), []string{"test.go"})
	s.Error(err)
}

type LintCheckTestSuite struct {
	suite.Suite

	tempDir string
	oldDir  string
}

func TestLintCheckSuite(t *testing.T) {
	suite.Run(t, new(LintCheckTestSuite))
}

func (s *LintCheckTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "lint_test_*")
	s.Require().NoError(err)

	s.oldDir, err = os.Getwd()
	s.Require().NoError(err)

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Initialize git repo
	s.initGitRepo()
}

func (s *LintCheckTestSuite) TearDownTest() {
	if s.oldDir != "" {
		err := os.Chdir(s.oldDir)
		s.Require().NoError(err)
	}
	if s.tempDir != "" {
		err := os.RemoveAll(s.tempDir)
		s.Require().NoError(err)
	}
}

func (s *LintCheckTestSuite) initGitRepo() {
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "init").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())
}

func (s *LintCheckTestSuite) TestNewLintCheckWithSharedContext() {
	sharedCtx := shared.NewContext()
	check := NewLintCheckWithSharedContext(sharedCtx)
	s.NotNil(check)
	s.Equal(sharedCtx, check.sharedCtx)
}

func (s *LintCheckTestSuite) TestNewLintCheckWithConfig() {
	sharedCtx := shared.NewContext()
	timeout := 30 * time.Second
	check := NewLintCheckWithConfig(sharedCtx, timeout)
	s.NotNil(check)
	s.Equal(sharedCtx, check.sharedCtx)
	s.Equal(timeout, check.timeout)
}

func (s *LintCheckTestSuite) TestRunMakeLint() {
	// Skip if make is not available
	if _, err := exec.LookPath("make"); err != nil {
		s.T().Skip("make not available")
	}

	// Create a Makefile that doesn't require golangci-lint to be installed
	makefileContent := `lint:
	@echo "Running lint check"
	@echo "No linting issues found"
`
	err := os.WriteFile("Makefile", []byte(makefileContent), 0o600)
	s.Require().NoError(err)

	check := NewLintCheck()
	err = check.Run(context.Background(), []string{"test.go"})

	// Should succeed with make target available
	s.NoError(err)
}

func (s *LintCheckTestSuite) TestRunDirectLint() {
	// Skip if golangci-lint is not available
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		s.T().Skip("golangci-lint not available")
	}

	// Create a basic Go module
	goMod := "module test\n\ngo 1.21\n"
	err := os.WriteFile("go.mod", []byte(goMod), 0o600)
	s.Require().NoError(err)

	// Create a Go file
	goFile := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	err = os.WriteFile("main.go", []byte(goFile), 0o600)
	s.Require().NoError(err)

	check := NewLintCheck()
	err = check.Run(context.Background(), []string{"main.go"})

	// Should succeed when golangci-lint is available
	s.NoError(err)
}

type ModTidyCheckTestSuite struct {
	suite.Suite

	tempDir string
	oldDir  string
}

func TestModTidyCheckSuite(t *testing.T) {
	suite.Run(t, new(ModTidyCheckTestSuite))
}

func (s *ModTidyCheckTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "modtidy_test_*")
	s.Require().NoError(err)

	s.oldDir, err = os.Getwd()
	s.Require().NoError(err)

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Initialize git repo
	s.initGitRepo()
}

func (s *ModTidyCheckTestSuite) TearDownTest() {
	if s.oldDir != "" {
		err := os.Chdir(s.oldDir)
		s.Require().NoError(err)
	}
	if s.tempDir != "" {
		err := os.RemoveAll(s.tempDir)
		s.Require().NoError(err)
	}
}

func (s *ModTidyCheckTestSuite) initGitRepo() {
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "init").Run())
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "config", "user.email", "test@example.com").Run())
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "config", "user.name", "Test User").Run())
}

func (s *ModTidyCheckTestSuite) TestNewModTidyCheckWithSharedContext() {
	sharedCtx := shared.NewContext()
	check := NewModTidyCheckWithSharedContext(sharedCtx)
	s.NotNil(check)
	s.Equal(sharedCtx, check.sharedCtx)
}

func (s *ModTidyCheckTestSuite) TestNewModTidyCheckWithConfig() {
	sharedCtx := shared.NewContext()
	timeout := 30 * time.Second
	check := NewModTidyCheckWithConfig(sharedCtx, timeout)
	s.NotNil(check)
	s.Equal(sharedCtx, check.sharedCtx)
	s.Equal(timeout, check.timeout)
}

func (s *ModTidyCheckTestSuite) TestFilterFilesWithGoFiles() {
	check := &ModTidyCheck{}

	files := []string{
		"main.go",
		"go.mod",
		"go.sum",
		"doc.md",
		"Makefile",
		"internal/pkg.go",
	}

	// With go.mod/go.sum present, should return only those
	filtered := check.FilterFiles(files)
	expected := []string{"go.mod", "go.sum"}
	s.Equal(expected, filtered)
}

func (s *ModTidyCheckTestSuite) TestFilterFilesOnlyGoFiles() {
	check := &ModTidyCheck{}

	files := []string{
		"main.go",
		"doc.md",
		"Makefile",
		"internal/pkg.go",
	}

	// With only .go files, should return dummy go.mod to trigger check
	filtered := check.FilterFiles(files)
	expected := []string{"go.mod"}
	s.Equal(expected, filtered)
}

func (s *ModTidyCheckTestSuite) TestFilterFilesNoGoFiles() {
	check := &ModTidyCheck{}

	files := []string{
		"doc.md",
		"Makefile",
		"README.txt",
	}

	// Without go files, should return empty
	filtered := check.FilterFiles(files)
	s.Empty(filtered)
}

func (s *ModTidyCheckTestSuite) TestRunMakeModTidy() {
	// Skip if make is not available
	if _, err := exec.LookPath("make"); err != nil {
		s.T().Skip("make not available")
	}

	// Create a basic Go module
	goMod := "module test\n\ngo 1.21\n"
	err := os.WriteFile("go.mod", []byte(goMod), 0o600)
	s.Require().NoError(err)

	// Add and commit the go.mod to git
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "add", "go.mod").Run())
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "commit", "-m", "Add go.mod").Run())

	// Create a Makefile that doesn't actually modify files
	makefileContent := `mod-tidy:
	@echo "Running mod-tidy check"
	@echo "All modules are tidy"
`
	err = os.WriteFile("Makefile", []byte(makefileContent), 0o600)
	s.Require().NoError(err)

	check := NewModTidyCheck()
	err = check.Run(context.Background(), []string{"go.mod"})

	// Should succeed with make target available
	s.NoError(err)
}

func (s *ModTidyCheckTestSuite) TestRunDirectModTidy() {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		s.T().Skip("go not available")
	}

	// Create a basic Go module
	goMod := "module test\n\ngo 1.21\n"
	err := os.WriteFile("go.mod", []byte(goMod), 0o600)
	s.Require().NoError(err)

	// Create a Go file
	goFile := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
	err = os.WriteFile("main.go", []byte(goFile), 0o600)
	s.Require().NoError(err)

	// Add and commit the files to git
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "commit", "-m", "Add initial files").Run())

	check := NewModTidyCheck()
	err = check.Run(context.Background(), []string{"go.mod"})

	// Should succeed when go is available
	s.NoError(err)
}

func (s *ModTidyCheckTestSuite) TestCheckUncommittedChanges() {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		s.T().Skip("go not available")
	}

	// Create a basic Go module
	goMod := "module test\n\ngo 1.21\n"
	err := os.WriteFile("go.mod", []byte(goMod), 0o600)
	s.Require().NoError(err)

	// Create a Go file that uses an external dependency
	goFile := `package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
)

func main() {
	fmt.Println("hello")
	assert.True(nil, true)
}
`
	err = os.WriteFile("main.go", []byte(goFile), 0o600)
	s.Require().NoError(err)

	// Add and commit the files
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(context.Background(), "git", "commit", "-m", "Initial commit").Run())

	// Now run mod tidy which should modify go.mod and go.sum
	check := NewModTidyCheck()
	err = check.Run(context.Background(), []string{"go.mod"})

	// Should return error indicating uncommitted changes
	s.Error(err)
}

// Edge case and error condition tests
func TestFumptCheckEdgeCases(t *testing.T) {
	t.Run("empty files list", func(t *testing.T) {
		check := NewFumptCheck()

		// Create temp dir and change to it for clean test
		tmpDir := t.TempDir()
		oldDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(oldDir)
		}()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Initialize git repository for tests that need it
		require.NoError(t, exec.CommandContext(context.Background(), "git", "init").Run())
		require.NoError(t, exec.CommandContext(context.Background(), "git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.CommandContext(context.Background(), "git", "config", "user.name", "Test User").Run())

		err = check.Run(context.Background(), []string{})
		assert.NoError(t, err) // Should succeed with no files
	})

	t.Run("non-go files", func(t *testing.T) {
		check := NewFumptCheck()
		files := []string{"README.md", "Makefile", "config.yml"}
		filtered := check.FilterFiles(files)
		assert.Empty(t, filtered)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		check := NewFumptCheck()
		err := check.Run(ctx, []string{"test.go"})
		assert.Error(t, err)
	})
}

func TestLintCheckEdgeCases(t *testing.T) {
	t.Run("empty files list", func(t *testing.T) {
		check := NewLintCheck()

		// Create temp dir and change to it for clean test
		tmpDir := t.TempDir()
		oldDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(oldDir)
		}()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Initialize git repository for tests that need it
		require.NoError(t, exec.CommandContext(context.Background(), "git", "init").Run())
		require.NoError(t, exec.CommandContext(context.Background(), "git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.CommandContext(context.Background(), "git", "config", "user.name", "Test User").Run())

		err = check.Run(context.Background(), []string{})
		assert.NoError(t, err) // Should succeed with no files
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		check := NewLintCheck()
		err := check.Run(ctx, []string{"test.go"})
		assert.Error(t, err)
	})
}

func TestModTidyCheckEdgeCases(t *testing.T) {
	t.Run("empty files list", func(t *testing.T) {
		check := NewModTidyCheck()

		// Create temp dir and change to it for clean test
		tmpDir := t.TempDir()
		oldDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(oldDir)
		}()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Initialize git repository for tests that need it
		require.NoError(t, exec.CommandContext(context.Background(), "git", "init").Run())
		require.NoError(t, exec.CommandContext(context.Background(), "git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.CommandContext(context.Background(), "git", "config", "user.name", "Test User").Run())

		err = check.Run(context.Background(), []string{})
		assert.NoError(t, err) // Should succeed with no files
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		check := NewModTidyCheck()
		err := check.Run(ctx, []string{"go.mod"})
		assert.Error(t, err)
	})
}
