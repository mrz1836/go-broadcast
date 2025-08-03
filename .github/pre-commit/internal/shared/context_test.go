package shared

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ContextTestSuite struct {
	suite.Suite
	tempDir string
}

func TestContextSuite(t *testing.T) {
	suite.Run(t, new(ContextTestSuite))
}

func (s *ContextTestSuite) SetupTest() {
	// Create a temporary directory for testing
	var err error
	s.tempDir, err = os.MkdirTemp("", "context_test_*")
	s.Require().NoError(err)

	// Initialize git repository
	s.Require().NoError(s.initGitRepo())
}

func (s *ContextTestSuite) TearDownTest() {
	if s.tempDir != "" {
		err := os.RemoveAll(s.tempDir)
		s.Require().NoError(err)
	}
}

func (s *ContextTestSuite) initGitRepo() error {
	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	if err := os.Chdir(s.tempDir); err != nil {
		return err
	}

	// Initialize git repo
	if err := exec.Command("git", "init").Run(); err != nil {
		return err
	}

	// Configure git
	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		return err
	}
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		return err
	}

	// Create and add a test file
	testFile := filepath.Join(s.tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		return err
	}

	if err := exec.Command("git", "add", "test.txt").Run(); err != nil {
		return err
	}

	if err := exec.Command("git", "commit", "-m", "Initial commit").Run(); err != nil {
		return err
	}

	return nil
}

func (s *ContextTestSuite) createMakefile(targets []string) {
	makefileContent := "# Test Makefile\n\n"
	for _, target := range targets {
		makefileContent += target + ":\n\t@echo \"Running " + target + "\"\n\n"
	}

	makefilePath := filepath.Join(s.tempDir, "Makefile")
	err := os.WriteFile(makefilePath, []byte(makefileContent), 0o644)
	s.Require().NoError(err)
}

// TestNewContext tests the constructor
func (s *ContextTestSuite) TestNewContext() {
	ctx := NewContext()
	s.Assert().NotNil(ctx)
	s.Assert().NotNil(ctx.makeTargets)
	s.Assert().Equal("", ctx.repoRoot)
}

// TestGetRepoRoot tests repository root discovery
func (s *ContextTestSuite) TestGetRepoRoot() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		err := os.Chdir(oldDir)
		s.Require().NoError(err)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Test successful repository root discovery
	root, err := ctx.GetRepoRoot(context.Background())
	s.Assert().NoError(err)

	// Resolve symlinks to handle macOS /var -> /private/var
	expectedPath, err := filepath.EvalSymlinks(s.tempDir)
	s.Require().NoError(err)
	actualPath, err := filepath.EvalSymlinks(root)
	s.Require().NoError(err)
	s.Assert().Equal(expectedPath, actualPath)

	// Test that subsequent calls return cached result
	root2, err2 := ctx.GetRepoRoot(context.Background())
	s.Assert().NoError(err2)
	s.Assert().Equal(root, root2)
}

// TestGetRepoRootError tests error handling when not in git repository
func (s *ContextTestSuite) TestGetRepoRootError() {
	ctx := NewContext()

	// Create a non-git directory
	nonGitDir, err := os.MkdirTemp("", "non_git_*")
	s.Require().NoError(err)
	defer func() {
		err := os.RemoveAll(nonGitDir)
		s.Require().NoError(err)
	}()

	// Change to non-git directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		err := os.Chdir(oldDir)
		s.Require().NoError(err)
	}()

	err = os.Chdir(nonGitDir)
	s.Require().NoError(err)

	// Test error case
	root, err := ctx.GetRepoRoot(context.Background())
	s.Assert().Error(err)
	s.Assert().Equal("", root)

	// Test that error is cached
	root2, err2 := ctx.GetRepoRoot(context.Background())
	s.Assert().Error(err2)
	s.Assert().Equal("", root2)
	s.Assert().Equal(err, err2)
}

// TestGetRepoRootTimeout tests timeout handling
func (s *ContextTestSuite) TestGetRepoRootTimeout() {
	ctx := NewContext()

	// Create a context that times out immediately
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Allow some time for the context to timeout
	time.Sleep(10 * time.Millisecond)

	root, err := ctx.GetRepoRoot(timeoutCtx)
	s.Assert().Error(err)
	s.Assert().Equal("", root)
}

// TestHasMakeTarget tests make target detection
func (s *ContextTestSuite) TestHasMakeTarget() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		err := os.Chdir(oldDir)
		s.Require().NoError(err)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create Makefile with test targets
	s.createMakefile([]string{"lint", "test", "build"})

	// Test existing target
	hasLint := ctx.HasMakeTarget(context.Background(), "lint")
	s.Assert().True(hasLint)

	// Test non-existing target
	hasFoo := ctx.HasMakeTarget(context.Background(), "nonexistent")
	s.Assert().False(hasFoo)

	// Test that results are cached
	hasLint2 := ctx.HasMakeTarget(context.Background(), "lint")
	s.Assert().True(hasLint2)

	hasFoo2 := ctx.HasMakeTarget(context.Background(), "nonexistent")
	s.Assert().False(hasFoo2)
}

// TestHasMakeTargetNoGitRepo tests behavior when not in git repository
func (s *ContextTestSuite) TestHasMakeTargetNoGitRepo() {
	ctx := NewContext()

	// Create a non-git directory
	nonGitDir, err := os.MkdirTemp("", "non_git_*")
	s.Require().NoError(err)
	defer func() {
		err := os.RemoveAll(nonGitDir)
		s.Require().NoError(err)
	}()

	// Change to non-git directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		err := os.Chdir(oldDir)
		s.Require().NoError(err)
	}()

	err = os.Chdir(nonGitDir)
	s.Require().NoError(err)

	// Test that it returns false when git repo root cannot be found
	hasTarget := ctx.HasMakeTarget(context.Background(), "lint")
	s.Assert().False(hasTarget)
}

// TestHasMakeTargetTimeout tests timeout handling
func (s *ContextTestSuite) TestHasMakeTargetTimeout() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		err := os.Chdir(oldDir)
		s.Require().NoError(err)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create Makefile
	s.createMakefile([]string{"lint"})

	// Test with a very short timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Allow some time for the context to timeout
	time.Sleep(10 * time.Millisecond)

	hasTarget := ctx.HasMakeTarget(timeoutCtx, "lint")
	// Should return false due to timeout
	s.Assert().False(hasTarget)
}

// TestExecuteMakeTarget tests make target execution
func (s *ContextTestSuite) TestExecuteMakeTarget() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		err := os.Chdir(oldDir)
		s.Require().NoError(err)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create Makefile with test targets
	s.createMakefile([]string{"test", "build"})

	// Test successful execution
	err = ctx.ExecuteMakeTarget(context.Background(), "test", 5*time.Second)
	s.Assert().NoError(err)

	// Test execution of non-existent target
	err = ctx.ExecuteMakeTarget(context.Background(), "nonexistent", 5*time.Second)
	s.Assert().Error(err)
}

// TestExecuteMakeTargetNoGitRepo tests execution when not in git repository
func (s *ContextTestSuite) TestExecuteMakeTargetNoGitRepo() {
	ctx := NewContext()

	// Create a non-git directory
	nonGitDir, err := os.MkdirTemp("", "non_git_*")
	s.Require().NoError(err)
	defer func() {
		err := os.RemoveAll(nonGitDir)
		s.Require().NoError(err)
	}()

	// Change to non-git directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		err := os.Chdir(oldDir)
		s.Require().NoError(err)
	}()

	err = os.Chdir(nonGitDir)
	s.Require().NoError(err)

	// Test that it returns error when git repo root cannot be found
	err = ctx.ExecuteMakeTarget(context.Background(), "test", 5*time.Second)
	s.Assert().Error(err)
}

// TestExecuteMakeTargetTimeout tests timeout handling
func (s *ContextTestSuite) TestExecuteMakeTargetTimeout() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		err := os.Chdir(oldDir)
		s.Require().NoError(err)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create Makefile with a slow target
	makefileContent := `# Test Makefile

slow:
	@sleep 10
`
	makefilePath := filepath.Join(s.tempDir, "Makefile")
	err = os.WriteFile(makefilePath, []byte(makefileContent), 0o644)
	s.Require().NoError(err)

	// Test execution with short timeout
	err = ctx.ExecuteMakeTarget(context.Background(), "slow", 100*time.Millisecond)
	s.Assert().Error(err)
}

// TestConcurrentAccess tests concurrent access to the context
func (s *ContextTestSuite) TestConcurrentAccess() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		err := os.Chdir(oldDir)
		s.Require().NoError(err)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create Makefile
	s.createMakefile([]string{"lint", "test", "build"})

	// Run multiple goroutines checking make targets
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			target := "lint"
			if i%2 == 0 {
				target = "test"
			}
			hasTarget := ctx.HasMakeTarget(context.Background(), target)
			s.Assert().True(hasTarget)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			s.Fail("Timeout waiting for goroutines")
		}
	}
}

// Unit tests for simple cases
func TestNewContextUnit(t *testing.T) {
	ctx := NewContext()
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.makeTargets)
	assert.Equal(t, "", ctx.repoRoot)
}

func TestContextCaching(t *testing.T) {
	ctx := NewContext()

	// Test that make targets cache is properly initialized
	assert.NotNil(t, ctx.makeTargets)
	assert.Equal(t, 0, len(ctx.makeTargets))
}

// Benchmark tests
func BenchmarkHasMakeTarget(b *testing.B) {
	ctx := NewContext()

	// Create temporary git repo
	tempDir, err := os.MkdirTemp("", "bench_*")
	require.NoError(b, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Initialize git repo
	oldDir, err := os.Getwd()
	require.NoError(b, err)
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	require.NoError(b, os.Chdir(tempDir))
	require.NoError(b, exec.Command("git", "init").Run())
	require.NoError(b, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(b, exec.Command("git", "config", "user.name", "Test User").Run())

	// Create Makefile
	makefileContent := "lint:\n\t@echo lint\n"
	require.NoError(b, os.WriteFile(filepath.Join(tempDir, "Makefile"), []byte(makefileContent), 0o644))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.HasMakeTarget(context.Background(), "lint")
	}
}
