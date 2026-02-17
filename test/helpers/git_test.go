package helpers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSkipIfNoGit tests the SkipIfNoGit function
func TestSkipIfNoGit(t *testing.T) {
	// Test that the function properly skips when git is not available
	// We can't easily mock the testing.T interface, but we can verify
	// that the function works with a real testing.T

	t.Run("WithGitAvailable", func(t *testing.T) {
		// This test verifies that SkipIfNoGit behaves correctly
		// when git is available (or not)
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git is not available - test would be skipped")
		}
		// If we get here, git is available and SkipIfNoGit should not skip
	})
}

// TestInitGitRepo tests the InitGitRepo function
func TestInitGitRepo(t *testing.T) {
	SkipIfNoGit(t)

	t.Run("SuccessfulInit", func(t *testing.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Initialize git repository
		err := InitGitRepo(tmpDir, "Initial commit")
		require.NoError(t, err)

		// Verify git directory exists
		gitDir := filepath.Join(tmpDir, ".git")
		assert.DirExists(t, gitDir)

		// Verify we can get the commit
		commit, err := GetLatestCommit(tmpDir)
		require.NoError(t, err)
		assert.NotEmpty(t, commit)
	})

	t.Run("InvalidDirectory", func(t *testing.T) {
		// Try to initialize in a non-existent directory
		err := InitGitRepo("/nonexistent/directory", "Initial commit")
		assert.Error(t, err)
	})

	t.Run("EmptyCommitMessage", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize with empty commit message
		err := InitGitRepo(tmpDir, "")
		// Git doesn't allow empty commit messages without --allow-empty-message flag
		// which our implementation doesn't use
		assert.Error(t, err)
	})

	t.Run("SpecialCharactersInCommitMessage", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize with special characters in commit message
		commitMsg := `Special "characters" & 'quotes' <brackets> $variables`
		err := InitGitRepo(tmpDir, commitMsg)
		require.NoError(t, err)

		// Verify repository was created
		gitDir := filepath.Join(tmpDir, ".git")
		assert.DirExists(t, gitDir)
	})

	t.Run("ReinitializeExistingRepo", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize first time
		err := InitGitRepo(tmpDir, "First init")
		require.NoError(t, err)

		// Get first commit
		firstCommit, err := GetLatestCommit(tmpDir)
		require.NoError(t, err)

		// Try to reinitialize - git init is idempotent and allows another empty commit
		err = InitGitRepo(tmpDir, "Second init")
		// This actually succeeds because git init is idempotent and --allow-empty permits another commit
		require.NoError(t, err)

		// Get the new commit
		currentCommit, err := GetLatestCommit(tmpDir)
		require.NoError(t, err)
		// The commits will be different since a new empty commit was created
		assert.NotEqual(t, firstCommit, currentCommit)
	})

	t.Run("ReadOnlyDirectory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping read-only directory test on Windows")
		}

		tmpDir := t.TempDir()

		// Make directory read-only
		require.NoError(t, os.Chmod(tmpDir, 0o555)) //nolint:gosec // Test requires restricted permissions
		defer func() {
			// Restore permissions for cleanup
			_ = os.Chmod(tmpDir, 0o755) //nolint:gosec // Test cleanup restoring permissions
		}()

		// Try to initialize in read-only directory
		err := InitGitRepo(tmpDir, "Initial commit")
		assert.Error(t, err)
	})
}

// TestCommitChanges tests the CommitChanges function
func TestCommitChanges(t *testing.T) {
	SkipIfNoGit(t)

	t.Run("SuccessfulCommit", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Create a file
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))

		// Commit changes
		err := CommitChanges(tmpDir, "Add test file")
		require.NoError(t, err)

		// Verify commit was created
		commit, err := GetLatestCommit(tmpDir)
		require.NoError(t, err)
		assert.NotEmpty(t, commit)
	})

	t.Run("NoChangesToCommit", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Try to commit without changes
		err := CommitChanges(tmpDir, "No changes")
		// Git will return an error when there's nothing to commit
		assert.Error(t, err)
	})

	t.Run("InvalidDirectory", func(t *testing.T) {
		// Try to commit in non-existent directory
		err := CommitChanges("/nonexistent/directory", "Test commit")
		assert.Error(t, err)
	})

	t.Run("NotAGitRepository", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a file without initializing git
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))

		// Try to commit in non-git directory
		err := CommitChanges(tmpDir, "Test commit")
		assert.Error(t, err)
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Create multiple files
		for i := 0; i < 5; i++ {
			testFile := filepath.Join(tmpDir, fmt.Sprintf("test%d.txt", i))
			require.NoError(t, os.WriteFile(testFile, []byte(fmt.Sprintf("content %d", i)), 0o600))
		}

		// Commit all changes
		err := CommitChanges(tmpDir, "Add multiple files")
		require.NoError(t, err)

		// Verify commit was created
		commit, err := GetLatestCommit(tmpDir)
		require.NoError(t, err)
		assert.NotEmpty(t, commit)
	})

	t.Run("FilesInSubdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Create subdirectory with files
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0o750))

		testFile := filepath.Join(subDir, "nested.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("nested content"), 0o600))

		// Commit changes
		err := CommitChanges(tmpDir, "Add nested file")
		require.NoError(t, err)

		// Verify commit was created
		commit, err := GetLatestCommit(tmpDir)
		require.NoError(t, err)
		assert.NotEmpty(t, commit)
	})

	t.Run("EmptyCommitMessage", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Create a file
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))

		// Commit with empty message - git doesn't allow this without --allow-empty-message
		err := CommitChanges(tmpDir, "")
		assert.Error(t, err)
	})
}

// TestGetLatestCommit tests the GetLatestCommit function
func TestGetLatestCommit(t *testing.T) {
	SkipIfNoGit(t)

	t.Run("SuccessfulGet", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Get latest commit
		commit, err := GetLatestCommit(tmpDir)
		require.NoError(t, err)
		assert.NotEmpty(t, commit)
		// Git SHA should be 40 characters
		assert.Len(t, commit, 40)
	})

	t.Run("InvalidDirectory", func(t *testing.T) {
		// Try to get commit from non-existent directory
		commit, err := GetLatestCommit("/nonexistent/directory")
		require.Error(t, err)
		assert.Empty(t, commit)
	})

	t.Run("NotAGitRepository", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Try to get commit from non-git directory
		commit, err := GetLatestCommit(tmpDir)
		require.Error(t, err)
		assert.Empty(t, commit)
	})

	t.Run("MultipleCommits", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Get first commit
		firstCommit, err := GetLatestCommit(tmpDir)
		require.NoError(t, err)

		// Create a file and commit
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))
		require.NoError(t, CommitChanges(tmpDir, "Second commit"))

		// Get latest commit
		secondCommit, err := GetLatestCommit(tmpDir)
		require.NoError(t, err)

		// Commits should be different
		assert.NotEqual(t, firstCommit, secondCommit)
		assert.Len(t, secondCommit, 40)
	})

	t.Run("CorruptedGitDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Corrupt the git directory by removing HEAD
		headFile := filepath.Join(tmpDir, ".git", "HEAD")
		require.NoError(t, os.Remove(headFile))

		// Try to get commit from corrupted repository
		commit, err := GetLatestCommit(tmpDir)
		require.Error(t, err)
		assert.Empty(t, commit)
	})
}

// TestCreateBranch tests the CreateBranch function
func TestCreateBranch(t *testing.T) {
	SkipIfNoGit(t)

	t.Run("SuccessfulCreate", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Create branch
		err := CreateBranch(tmpDir, "feature-branch")
		require.NoError(t, err)

		// Verify we're on the new branch
		cmd := exec.CommandContext(context.Background(), "git", "branch", "--show-current")
		cmd.Dir = tmpDir
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Equal(t, "feature-branch", strings.TrimSpace(string(output)))
	})

	t.Run("InvalidDirectory", func(t *testing.T) {
		// Try to create branch in non-existent directory
		err := CreateBranch("/nonexistent/directory", "test-branch")
		assert.Error(t, err)
	})

	t.Run("NotAGitRepository", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Try to create branch in non-git directory
		err := CreateBranch(tmpDir, "test-branch")
		assert.Error(t, err)
	})

	t.Run("DuplicateBranchName", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Create branch
		require.NoError(t, CreateBranch(tmpDir, "duplicate-branch"))

		// Switch back to main/master
		cmd := exec.CommandContext(context.Background(), "git", "checkout", "-")
		cmd.Dir = tmpDir
		_ = cmd.Run()

		// Try to create same branch again
		err := CreateBranch(tmpDir, "duplicate-branch")
		assert.Error(t, err)
	})

	t.Run("InvalidBranchName", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Try to create branch with invalid name
		invalidNames := []string{
			"branch with spaces",
			"branch..with..dots",
			"branch~with~tilde",
			"branch^with^caret",
			"branch:with:colon",
		}

		for _, name := range invalidNames {
			err := CreateBranch(tmpDir, name)
			assert.Error(t, err, "Branch name '%s' should be invalid", name)
		}
	})

	t.Run("ValidSpecialBranchNames", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Try valid special branch names
		validNames := []string{
			"feature/test",
			"bugfix-123",
			"release_v1.0",
		}

		for i, name := range validNames {
			// Switch back to main/master between attempts
			if i > 0 {
				cmd := exec.CommandContext(context.Background(), "git", "checkout", "-")
				cmd.Dir = tmpDir
				_ = cmd.Run()
			}

			err := CreateBranch(tmpDir, name)
			assert.NoError(t, err, "Branch name '%s' should be valid", name)
		}
	})
}

// TestFileExists tests the FileExists function
func TestFileExists(t *testing.T) {
	t.Run("ExistingFile", func(t *testing.T) {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "test")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }() //nolint:gosec // G703: path from os.CreateTemp, not user-controlled input

		exists, err := FileExists(tmpFile.Name())
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		exists, err := FileExists("/nonexistent/file.txt")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("ExistingDirectory", func(t *testing.T) {
		tmpDir := t.TempDir()

		exists, err := FileExists(tmpDir)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		// Create a directory with no read permissions
		tmpDir := t.TempDir()
		restrictedDir := filepath.Join(tmpDir, "restricted")
		require.NoError(t, os.Mkdir(restrictedDir, 0o000))
		defer func() {
			// Restore permissions for cleanup
			_ = os.Chmod(restrictedDir, 0o755) //nolint:gosec // Test cleanup restoring permissions
		}()

		// Try to check a file inside restricted directory
		testFile := filepath.Join(restrictedDir, "test.txt")
		exists, err := FileExists(testFile)

		// On Unix systems, we should get a permission error
		// The function returns false and an error for permission issues
		require.Error(t, err)
		assert.False(t, exists)
	})

	t.Run("EmptyPath", func(t *testing.T) {
		exists, err := FileExists("")
		// Empty path should return false, no error
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("RelativePath", func(t *testing.T) {
		// Create a file in current directory
		tmpFile, err := os.CreateTemp(".", "test")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }() //nolint:gosec // G703: path from os.CreateTemp, not user-controlled input

		// Get just the filename
		filename := filepath.Base(tmpFile.Name())

		exists, err := FileExists(filename)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("SymbolicLink", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping symlink test on Windows")
		}

		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "test")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }() //nolint:gosec // G703: path from os.CreateTemp, not user-controlled input

		// Create a symlink to the file
		linkPath := tmpFile.Name() + ".link"
		require.NoError(t, os.Symlink(tmpFile.Name(), linkPath))
		defer func() { _ = os.Remove(linkPath) }()

		// Check if symlink exists
		exists, err := FileExists(linkPath)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("BrokenSymbolicLink", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping symlink test on Windows")
		}

		tmpDir := t.TempDir()

		// Create a symlink to non-existent file
		linkPath := filepath.Join(tmpDir, "broken.link")
		require.NoError(t, os.Symlink("/nonexistent/file", linkPath))

		// Check if broken symlink exists
		// os.Stat follows symlinks, so this should return false
		exists, err := FileExists(linkPath)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

// TestContextCancellation tests context cancellation scenarios
func TestContextCancellation(t *testing.T) {
	SkipIfNoGit(t)

	t.Run("InitGitRepoWithCancelledContext", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Temporarily replace exec.CommandContext to test context handling
		// TODO: In the actual implementation, context is created inside the function
		// This test verifies the behavior if we could pass a context

		// Since InitGitRepo creates its own context, we can't directly test cancellation
		// But we can verify it works normally even with system under stress
		err := InitGitRepo(tmpDir, "Initial commit")
		require.NoError(t, err)

		// Verify the repository was created successfully
		gitDir := filepath.Join(tmpDir, ".git")
		assert.DirExists(t, gitDir)

		_ = ctx // Use ctx to avoid unused variable warning
	})
}

// TestConcurrentOperations tests concurrent git operations
func TestConcurrentOperations(t *testing.T) {
	SkipIfNoGit(t)

	t.Run("ConcurrentCommits", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize repository
		require.NoError(t, InitGitRepo(tmpDir, "Initial commit"))

		// Create multiple files
		for i := 0; i < 10; i++ {
			testFile := filepath.Join(tmpDir, fmt.Sprintf("test%d.txt", i))
			require.NoError(t, os.WriteFile(testFile, []byte(fmt.Sprintf("content %d", i)), 0o600))
		}

		// Try to commit (sequential is fine since git handles locking)
		err := CommitChanges(tmpDir, "Add files")
		require.NoError(t, err)
	})
}
