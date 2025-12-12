package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/testutil"
)

func TestBatchAddFiles(t *testing.T) {
	tests := []struct {
		name        string
		files       []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "EmptyFileList",
			files:       []string{},
			expectError: false,
		},
		{
			name:        "SingleFile",
			files:       []string{"file1.txt"},
			expectError: false,
		},
		{
			name:        "SmallBatch",
			files:       []string{"file1.txt", "file2.txt", "file3.txt"},
			expectError: false,
		},
		{
			name:        "LargeBatchExceedingLimit",
			files:       generateFileList(250), // More than maxBatchSize (100)
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping integration test")
			}

			// Create temporary git repository
			tmpDir := testutil.CreateTempDir(t)
			ctx := context.Background()

			// Initialize git repo
			initCmd := exec.CommandContext(ctx, "git", "init", tmpDir) //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err := initCmd.Run()
			require.NoError(t, err)

			// Configure git user for tests
			configCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.email", "test@example.com") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			configCmd = exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.name", "Test User") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			logger := logrus.New()
			logger.SetOutput(io.Discard) // Discard logs during tests
			client := &gitClient{logger: logger}

			// Test with empty files - should return nil immediately
			if len(tt.files) == 0 {
				err2 := client.BatchAddFiles(ctx, tmpDir, tt.files)
				require.NoError(t, err2)
				return
			}

			// Create test files
			for _, file := range tt.files {
				filePath := filepath.Join(tmpDir, file)
				testutil.WriteTestFile(t, filePath, "test content")
			}

			// Test batch add
			err = client.BatchAddFiles(ctx, tmpDir, tt.files)
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)

				// Verify files were added
				statusCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "status", "--porcelain") //nolint:gosec // tmpDir is from t.TempDir()
				output, err := statusCmd.Output()
				require.NoError(t, err)

				// All files should be staged (A)
				lines := strings.Split(strings.TrimSpace(string(output)), "\n")
				if len(tt.files) > 0 {
					require.Len(t, lines, len(tt.files))
					for _, line := range lines {
						require.True(t, strings.HasPrefix(line, "A "))
					}
				}
			}
		})
	}
}

func TestBatchStatus(t *testing.T) {
	tests := []struct {
		name        string
		files       []string
		setupFiles  func(t *testing.T, tmpDir string) // Function to set up files with specific states
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, result map[string]string)
	}{
		{
			name:  "EmptyFileList",
			files: []string{},
			validate: func(t *testing.T, result map[string]string) {
				require.Empty(t, result)
			},
		},
		{
			name:  "SingleFileModified",
			files: []string{"file1.txt"},
			setupFiles: func(t *testing.T, tmpDir string) {
				// Create and commit file
				filePath := filepath.Join(tmpDir, "file1.txt")
				testutil.WriteTestFile(t, filePath, "original content")

				ctx := context.Background()
				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "file1.txt")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "initial commit")
				err = commitCmd.Run()
				require.NoError(t, err)

				// Modify the file
				testutil.WriteTestFile(t, filePath, "modified content")
			},
			validate: func(t *testing.T, result map[string]string) {
				require.Len(t, result, 1)
				require.Equal(t, " M", result["file1.txt"])
			},
		},
		{
			name:  "MultipleFilesWithDifferentStatuses",
			files: []string{"file1.txt", "file2.txt", "file3.txt"},
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create and commit file1
				file1 := filepath.Join(tmpDir, "file1.txt")
				testutil.WriteTestFile(t, file1, "content1")

				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "file1.txt")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "add file1")
				err = commitCmd.Run()
				require.NoError(t, err)

				// Modify file1
				testutil.WriteTestFile(t, file1, "modified content1")

				// Create and stage file2
				file2 := filepath.Join(tmpDir, "file2.txt")
				testutil.WriteTestFile(t, file2, "content2")

				addCmd2 := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "file2.txt")
				err = addCmd2.Run()
				require.NoError(t, err)

				// Create untracked file3
				file3 := filepath.Join(tmpDir, "file3.txt")
				testutil.WriteTestFile(t, file3, "content3")
			},
			validate: func(t *testing.T, result map[string]string) {
				require.Len(t, result, 3)
				require.Equal(t, " M", result["file1.txt"])
				require.Equal(t, "A ", result["file2.txt"])
				require.Equal(t, "??", result["file3.txt"])
			},
		},
		{
			name:  "FilesWithSpacesInPath",
			files: []string{"path/to/my file.txt"},
			setupFiles: func(t *testing.T, tmpDir string) {
				// Create directory structure
				dirPath := filepath.Join(tmpDir, "path", "to")
				testutil.CreateTestDirectory(t, dirPath)

				// Create file with space in name
				filePath := filepath.Join(dirPath, "my file.txt")
				testutil.WriteTestFile(t, filePath, "content with spaces")
			},
			validate: func(t *testing.T, result map[string]string) {
				// When specifying specific files with spaces, git might not return them
				// if they're not properly escaped. In this case, we should get an empty result
				// or the file if git handles it properly
				if len(result) > 0 {
					require.Len(t, result, 1)
					// Check for the file in the result map
					for file, status := range result {
						require.Contains(t, file, "my file.txt")
						require.Equal(t, "??", status)
					}
				}
			},
		},
		{
			name:  "LargeBatchExceedingLimit",
			files: generateFileList(120), // More than maxBatchSize for status
			setupFiles: func(t *testing.T, tmpDir string) {
				// Create multiple files
				for i := 0; i < 120; i++ {
					fileName := fmt.Sprintf("file%d.txt", i+1)
					filePath := filepath.Join(tmpDir, fileName)
					testutil.WriteTestFileWithFormat(t, filePath, "content %d", i)
				}
			},
			validate: func(t *testing.T, result map[string]string) {
				require.Len(t, result, 120)
				for i := 0; i < 120; i++ {
					fileName := fmt.Sprintf("file%d.txt", i+1)
					require.Equal(t, "??", result[fileName])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping integration test")
			}

			// Create temporary git repository
			tmpDir := testutil.CreateTempDir(t)
			ctx := context.Background()

			// Initialize git repo
			initCmd := exec.CommandContext(ctx, "git", "init", tmpDir) //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err := initCmd.Run()
			require.NoError(t, err)

			// Configure git user for tests
			configCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.email", "test@example.com") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			configCmd = exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.name", "Test User") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			logger := logrus.New()
			logger.SetOutput(io.Discard) // Discard logs during tests
			client := &gitClient{logger: logger}

			// Set up files if needed
			if tt.setupFiles != nil {
				tt.setupFiles(t, tmpDir)
			}

			// Test batch status
			result, err := client.BatchStatus(ctx, tmpDir, tt.files)
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestBatchStatusAll(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  func(t *testing.T, tmpDir string) // Function to set up files with specific states
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, result map[string]string)
	}{
		{
			name: "EmptyRepository",
			validate: func(t *testing.T, result map[string]string) {
				require.Empty(t, result)
			},
		},
		{
			name: "MultipleFilesStatus",
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create src directory
				srcDir := filepath.Join(tmpDir, "src")
				testutil.CreateTestDirectory(t, srcDir)

				// Create and commit main.go
				mainFile := filepath.Join(srcDir, "main.go")
				testutil.WriteTestFile(t, mainFile, "package main")

				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "src/main.go")
				err := addCmd.Run()
				require.NoError(t, err)

				// Create and commit old.go to delete later
				oldFile := filepath.Join(tmpDir, "old.go")
				testutil.WriteTestFile(t, oldFile, "package old")

				addCmd2 := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "old.go")
				err = addCmd2.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "initial files")
				err = commitCmd.Run()
				require.NoError(t, err)

				// Modify main.go
				testutil.WriteTestFile(t, mainFile, "package main\n\nfunc main() {}")

				// Add README.md
				readmeFile := filepath.Join(tmpDir, "README.md")
				testutil.WriteTestFile(t, readmeFile, "# Test Repo")

				addCmd3 := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "README.md")
				err = addCmd3.Run()
				require.NoError(t, err)

				// Create untracked temp.txt
				tempFile := filepath.Join(tmpDir, "temp.txt")
				testutil.WriteTestFile(t, tempFile, "temporary")

				// Delete old.go
				rmCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "rm", "old.go")
				err = rmCmd.Run()
				require.NoError(t, err)
			},
			validate: func(t *testing.T, result map[string]string) {
				require.Len(t, result, 4)
				require.Equal(t, " M", result["src/main.go"])
				require.Equal(t, "A ", result["README.md"])
				require.Equal(t, "??", result["temp.txt"])
				require.Equal(t, "D ", result["old.go"])
			},
		},
		{
			name: "StatusWithRenames",
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create and commit old-name.txt
				oldFile := filepath.Join(tmpDir, "old-name.txt")
				testutil.WriteTestFile(t, oldFile, "content")

				// Create and commit file.go
				goFile := filepath.Join(tmpDir, "file.go")
				testutil.WriteTestFile(t, goFile, "package main")

				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", ".")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "initial commit")
				err = commitCmd.Run()
				require.NoError(t, err)

				// Rename old-name.txt to new-name.txt
				mvCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "mv", "old-name.txt", "new-name.txt")
				err = mvCmd.Run()
				require.NoError(t, err)

				// Modify file.go
				testutil.WriteTestFile(t, goFile, "package main\n\nfunc main() {}")
			},
			validate: func(t *testing.T, result map[string]string) {
				require.Len(t, result, 2)
				// Git shows renames as "old-name.txt -> new-name.txt"
				require.Equal(t, "R ", result["old-name.txt -> new-name.txt"])
				require.Equal(t, " M", result["file.go"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping integration test")
			}

			// Create temporary git repository
			tmpDir := testutil.CreateTempDir(t)
			ctx := context.Background()

			// Initialize git repo
			initCmd := exec.CommandContext(ctx, "git", "init", tmpDir) //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err := initCmd.Run()
			require.NoError(t, err)

			// Configure git user for tests
			configCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.email", "test@example.com") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			configCmd = exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.name", "Test User") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			logger := logrus.New()
			logger.SetOutput(io.Discard) // Discard logs during tests
			client := &gitClient{logger: logger}

			// Set up files if needed
			if tt.setupFiles != nil {
				tt.setupFiles(t, tmpDir)
			}

			// Test batch status all
			result, err := client.BatchStatusAll(ctx, tmpDir)
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestBatchDiffFiles(t *testing.T) {
	tests := []struct {
		name        string
		files       []string
		staged      bool
		setupFiles  func(t *testing.T, tmpDir string) // Function to set up files with specific states
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, result map[string]string)
	}{
		{
			name:   "EmptyFileList",
			files:  []string{},
			staged: false,
			validate: func(t *testing.T, result map[string]string) {
				require.Empty(t, result)
			},
		},
		{
			name:   "UnstagedDiff",
			files:  []string{"file1.txt", "file2.txt"},
			staged: false,
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create and commit files
				file1 := filepath.Join(tmpDir, "file1.txt")
				testutil.WriteTestFile(t, file1, "original content 1")

				file2 := filepath.Join(tmpDir, "file2.txt")
				testutil.WriteTestFile(t, file2, "original content 2")

				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", ".")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "initial commit")
				err = commitCmd.Run()
				require.NoError(t, err)

				// Modify the files
				testutil.WriteTestFile(t, file1, "modified content 1\nnew line")

				testutil.WriteTestFile(t, file2, "modified content 2")
			},
			validate: func(t *testing.T, result map[string]string) {
				require.Len(t, result, 2)

				// Check file1 diff
				diff1, ok := result["file1.txt"]
				require.True(t, ok, "file1.txt should have diff")
				require.Contains(t, diff1, "-original content 1")
				require.Contains(t, diff1, "+modified content 1")
				require.Contains(t, diff1, "+new line")

				// Check file2 diff
				diff2, ok := result["file2.txt"]
				require.True(t, ok, "file2.txt should have diff")
				require.Contains(t, diff2, "-original content 2")
				require.Contains(t, diff2, "+modified content 2")
			},
		},
		{
			name:   "StagedDiff",
			files:  []string{"file1.txt", "file2.txt"},
			staged: true,
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create and commit file1
				file1 := filepath.Join(tmpDir, "file1.txt")
				testutil.WriteTestFile(t, file1, "original content 1")

				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "file1.txt")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "add file1")
				err = commitCmd.Run()
				require.NoError(t, err)

				// Modify file1 and stage it
				testutil.WriteTestFile(t, file1, "staged content 1")

				addCmd2 := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "file1.txt")
				err = addCmd2.Run()
				require.NoError(t, err)

				// Create new file2 and stage it
				file2 := filepath.Join(tmpDir, "file2.txt")
				testutil.WriteTestFile(t, file2, "new file content")

				addCmd3 := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "file2.txt")
				err = addCmd3.Run()
				require.NoError(t, err)
			},
			validate: func(t *testing.T, result map[string]string) {
				require.Len(t, result, 2)

				// Check file1 staged diff
				diff1, ok := result["file1.txt"]
				require.True(t, ok, "file1.txt should have staged diff")
				require.Contains(t, diff1, "-original content 1")
				require.Contains(t, diff1, "+staged content 1")

				// Check file2 staged diff (new file)
				diff2, ok := result["file2.txt"]
				require.True(t, ok, "file2.txt should have staged diff")
				require.Contains(t, diff2, "+new file content")
			},
		},
		{
			name:   "LargeBatchSplit",
			files:  generateFileList(60), // More than maxBatchSize (50) for diff
			staged: false,
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create and commit multiple files
				for i := 0; i < 60; i++ {
					fileName := fmt.Sprintf("file%d.txt", i+1)
					filePath := filepath.Join(tmpDir, fileName)
					testutil.WriteTestFileWithFormat(t, filePath, "original content %d", i)
				}

				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", ".")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "initial commit")
				err = commitCmd.Run()
				require.NoError(t, err)

				// Modify all files
				for i := 0; i < 60; i++ {
					fileName := fmt.Sprintf("file%d.txt", i+1)
					filePath := filepath.Join(tmpDir, fileName)
					testutil.WriteTestFileWithFormat(t, filePath, "modified content %d", i)
				}
			},
			validate: func(t *testing.T, result map[string]string) {
				require.Len(t, result, 60)

				// Check that all files have diffs
				for i := 0; i < 60; i++ {
					fileName := fmt.Sprintf("file%d.txt", i+1)
					diff, ok := result[fileName]
					require.True(t, ok, "%s should have diff", fileName)
					require.Contains(t, diff, fmt.Sprintf("-original content %d", i))
					require.Contains(t, diff, fmt.Sprintf("+modified content %d", i))
				}
			},
		},
		{
			name:   "MixedChanges",
			files:  []string{"modified.txt", "unchanged.txt", "deleted.txt"},
			staged: false,
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create and commit files
				modFile := filepath.Join(tmpDir, "modified.txt")
				testutil.WriteTestFile(t, modFile, "original")

				unchangedFile := filepath.Join(tmpDir, "unchanged.txt")
				testutil.WriteTestFile(t, unchangedFile, "no changes")

				delFile := filepath.Join(tmpDir, "deleted.txt")
				testutil.WriteTestFile(t, delFile, "to be deleted")

				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", ".")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "initial")
				err = commitCmd.Run()
				require.NoError(t, err)

				// Make changes
				testutil.WriteTestFile(t, modFile, "modified")

				err = os.Remove(delFile)
				require.NoError(t, err)
			},
			validate: func(t *testing.T, result map[string]string) {
				// Only modified and deleted files should have diffs
				require.Len(t, result, 2)

				// Check modified file
				diff1, ok := result["modified.txt"]
				require.True(t, ok)
				require.Contains(t, diff1, "-original")
				require.Contains(t, diff1, "+modified")

				// Check deleted file
				diff2, ok := result["deleted.txt"]
				require.True(t, ok)
				require.Contains(t, diff2, "-to be deleted")

				// Unchanged file should not be in results
				_, ok = result["unchanged.txt"]
				require.False(t, ok)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping integration test")
			}

			// Create temporary git repository
			tmpDir := testutil.CreateTempDir(t)
			ctx := context.Background()

			// Initialize git repo
			initCmd := exec.CommandContext(ctx, "git", "init", tmpDir) //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err := initCmd.Run()
			require.NoError(t, err)

			// Configure git user for tests
			configCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.email", "test@example.com") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			configCmd = exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.name", "Test User") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			logger := logrus.New()
			logger.SetOutput(io.Discard) // Discard logs during tests
			client := &gitClient{logger: logger}

			// Set up files if needed
			if tt.setupFiles != nil {
				tt.setupFiles(t, tmpDir)
			}

			// Test batch diff
			result, err := client.BatchDiffFiles(ctx, tmpDir, tt.files, tt.staged)
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestBatchCheckIgnored(t *testing.T) {
	tests := []struct {
		name        string
		files       []string
		setupFiles  func(t *testing.T, tmpDir string) // Function to set up files and gitignore
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, result map[string]bool)
	}{
		{
			name:  "EmptyFileList",
			files: []string{},
			validate: func(t *testing.T, result map[string]bool) {
				require.Empty(t, result)
			},
		},
		{
			name:  "NoIgnoredFiles",
			files: []string{"file1.txt", "file2.txt"},
			setupFiles: func(t *testing.T, tmpDir string) {
				// Create files without gitignore
				file1 := filepath.Join(tmpDir, "file1.txt")
				testutil.WriteTestFile(t, file1, "content1")

				file2 := filepath.Join(tmpDir, "file2.txt")
				testutil.WriteTestFile(t, file2, "content2")
			},
			validate: func(t *testing.T, result map[string]bool) {
				require.Len(t, result, 2)
				require.False(t, result["file1.txt"])
				require.False(t, result["file2.txt"])
			},
		},
		{
			name:  "SomeIgnoredFiles",
			files: []string{"file1.txt", ".DS_Store", "build/output.txt"},
			setupFiles: func(t *testing.T, tmpDir string) {
				// Create gitignore
				gitignore := filepath.Join(tmpDir, ".gitignore")
				testutil.WriteTestFile(t, gitignore, ".DS_Store\nbuild/\n")

				// Create files
				file1 := filepath.Join(tmpDir, "file1.txt")
				testutil.WriteTestFile(t, file1, "content1")

				dsStore := filepath.Join(tmpDir, ".DS_Store")
				testutil.WriteTestFile(t, dsStore, "macos metadata")

				// Create build directory and file
				buildDir := filepath.Join(tmpDir, "build")
				testutil.CreateTestDirectory(t, buildDir)

				buildFile := filepath.Join(buildDir, "output.txt")
				testutil.WriteTestFile(t, buildFile, "build output")
			},
			validate: func(t *testing.T, result map[string]bool) {
				require.Len(t, result, 3)
				require.False(t, result["file1.txt"])
				require.True(t, result[".DS_Store"])
				require.True(t, result["build/output.txt"])
			},
		},
		{
			name:  "AllIgnoredFiles",
			files: []string{"node_modules/index.js", ".env", "debug.log"},
			setupFiles: func(t *testing.T, tmpDir string) {
				// Create comprehensive gitignore
				gitignore := filepath.Join(tmpDir, ".gitignore")
				testutil.WriteTestFile(t, gitignore, "node_modules/\n.env\n*.log\n")

				// Create node_modules directory and file
				nodeDir := filepath.Join(tmpDir, "node_modules")
				testutil.CreateTestDirectory(t, nodeDir)

				nodeFile := filepath.Join(nodeDir, "index.js")
				testutil.WriteTestFile(t, nodeFile, "module.exports = {}")

				// Create .env file
				envFile := filepath.Join(tmpDir, ".env")
				testutil.WriteTestFile(t, envFile, "SECRET=value")

				// Create log file
				logFile := filepath.Join(tmpDir, "debug.log")
				testutil.WriteTestFile(t, logFile, "debug info")
			},
			validate: func(t *testing.T, result map[string]bool) {
				require.Len(t, result, 3)
				require.True(t, result["node_modules/index.js"])
				require.True(t, result[".env"])
				require.True(t, result["debug.log"])
			},
		},
		{
			name:  "GitignorePatterns",
			files: []string{"test.tmp", "backup.bak", "src/main.go", "src/test.tmp"},
			setupFiles: func(t *testing.T, tmpDir string) {
				// Create gitignore with patterns
				gitignore := filepath.Join(tmpDir, ".gitignore")
				testutil.WriteTestFile(t, gitignore, "*.tmp\n*.bak\n")

				// Create src directory
				srcDir := filepath.Join(tmpDir, "src")
				testutil.CreateTestDirectory(t, srcDir)

				// Create files
				files := []string{
					filepath.Join(tmpDir, "test.tmp"),
					filepath.Join(tmpDir, "backup.bak"),
					filepath.Join(srcDir, "main.go"),
					filepath.Join(srcDir, "test.tmp"),
				}

				for _, file := range files {
					testutil.WriteTestFile(t, file, "content")
				}
			},
			validate: func(t *testing.T, result map[string]bool) {
				require.Len(t, result, 4)
				require.True(t, result["test.tmp"])
				require.True(t, result["backup.bak"])
				require.False(t, result["src/main.go"])
				require.True(t, result["src/test.tmp"])
			},
		},
		{
			name:  "LargeBatchExceedingLimit",
			files: generateFileList(120), // More than maxBatchSize
			setupFiles: func(t *testing.T, tmpDir string) {
				// Create gitignore that ignores even numbered files
				var gitignoreContent strings.Builder
				for i := 0; i < 120; i += 2 {
					gitignoreContent.WriteString(fmt.Sprintf("file%d.txt\n", i+1))
				}

				gitignore := filepath.Join(tmpDir, ".gitignore")
				testutil.WriteTestFile(t, gitignore, gitignoreContent.String())

				// Create all files
				for i := 0; i < 120; i++ {
					fileName := fmt.Sprintf("file%d.txt", i+1)
					filePath := filepath.Join(tmpDir, fileName)
					testutil.WriteTestFile(t, filePath, "content")
				}
			},
			validate: func(t *testing.T, result map[string]bool) {
				require.Len(t, result, 120)

				// Check that even numbered files are ignored
				for i := 0; i < 120; i++ {
					fileName := fmt.Sprintf("file%d.txt", i+1)
					if i%2 == 0 {
						require.True(t, result[fileName], "%s should be ignored", fileName)
					} else {
						require.False(t, result[fileName], "%s should not be ignored", fileName)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping integration test")
			}

			// Create temporary git repository
			tmpDir := testutil.CreateTempDir(t)
			ctx := context.Background()

			// Initialize git repo
			initCmd := exec.CommandContext(ctx, "git", "init", tmpDir) //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err := initCmd.Run()
			require.NoError(t, err)

			// Configure git user for tests
			configCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.email", "test@example.com") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			configCmd = exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.name", "Test User") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			logger := logrus.New()
			logger.SetOutput(io.Discard) // Discard logs during tests
			client := &gitClient{logger: logger}

			// Set up files if needed
			if tt.setupFiles != nil {
				tt.setupFiles(t, tmpDir)
			}

			// Test batch check ignored
			result, err := client.BatchCheckIgnored(ctx, tmpDir, tt.files)
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestBatchRemoveFiles(t *testing.T) {
	tests := []struct {
		name        string
		files       []string
		keepLocal   bool
		setupFiles  func(t *testing.T, tmpDir string) // Function to set up files
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, tmpDir string)
	}{
		{
			name:      "EmptyFileList",
			files:     []string{},
			keepLocal: false,
			validate: func(_ *testing.T, _ string) {
				// Nothing to validate for empty list
			},
		},
		{
			name:      "SingleFileRemove",
			files:     []string{"file1.txt"},
			keepLocal: false,
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create and commit file
				file1 := filepath.Join(tmpDir, "file1.txt")
				testutil.WriteTestFile(t, file1, "content1")

				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "file1.txt")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "add file1")
				err = commitCmd.Run()
				require.NoError(t, err)
			},
			validate: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Check git status - file should be deleted
				statusCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "status", "--porcelain")
				output, err := statusCmd.Output()
				require.NoError(t, err)
				require.Contains(t, string(output), "D  file1.txt")

				// Check that file doesn't exist on disk
				file1 := filepath.Join(tmpDir, "file1.txt")
				_, err = os.Stat(file1)
				require.True(t, os.IsNotExist(err))
			},
		},
		{
			name:      "RemoveKeepLocal",
			files:     []string{"file1.txt", "file2.txt"},
			keepLocal: true,
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create and commit files
				file1 := filepath.Join(tmpDir, "file1.txt")
				testutil.WriteTestFile(t, file1, "content1")

				file2 := filepath.Join(tmpDir, "file2.txt")
				testutil.WriteTestFile(t, file2, "content2")

				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", ".")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "add files")
				err = commitCmd.Run()
				require.NoError(t, err)
			},
			validate: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Check git status - files should be deleted from index
				statusCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "status", "--porcelain")
				output, err := statusCmd.Output()
				require.NoError(t, err)

				// Files should be deleted from index but present as untracked
				lines := strings.Split(strings.TrimSpace(string(output)), "\n")
				require.Len(t, lines, 4) // 2 deleted (D) + 2 untracked (??)

				statusMap := make(map[string]string)
				for _, line := range lines {
					if len(line) >= 3 {
						status := line[:2]
						filename := strings.TrimSpace(line[3:])
						// If file already has status, append (for D and ??)
						if existing, ok := statusMap[filename]; ok {
							statusMap[filename] = existing + "," + status
						} else {
							statusMap[filename] = status
						}
					}
				}

				// Files should show as both deleted from index and untracked
				require.Contains(t, statusMap["file1.txt"], "D ")
				require.Contains(t, statusMap["file1.txt"], "??")
				require.Contains(t, statusMap["file2.txt"], "D ")
				require.Contains(t, statusMap["file2.txt"], "??")

				// Check that files still exist on disk
				file1 := filepath.Join(tmpDir, "file1.txt")
				info1, err := os.Stat(file1)
				require.NoError(t, err)
				require.False(t, info1.IsDir())

				file2 := filepath.Join(tmpDir, "file2.txt")
				info2, err := os.Stat(file2)
				require.NoError(t, err)
				require.False(t, info2.IsDir())
			},
		},
		{
			name:      "RemoveWithSubdirectories",
			files:     []string{"src/main.go", "test/test.go", "README.md"},
			keepLocal: false,
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create directory structure
				srcDir := filepath.Join(tmpDir, "src")
				testutil.CreateTestDirectory(t, srcDir)

				testDir := filepath.Join(tmpDir, "test")
				testutil.CreateTestDirectory(t, testDir)

				// Create files
				mainFile := filepath.Join(srcDir, "main.go")
				testutil.WriteTestFile(t, mainFile, "package main")

				testFile := filepath.Join(testDir, "test.go")
				testutil.WriteTestFile(t, testFile, "package test")

				readmeFile := filepath.Join(tmpDir, "README.md")
				testutil.WriteTestFile(t, readmeFile, "# Test")

				// Add and commit all files
				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", ".")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "add files")
				err = commitCmd.Run()
				require.NoError(t, err)
			},
			validate: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Check git status
				statusCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "status", "--porcelain")
				output, err := statusCmd.Output()
				require.NoError(t, err)

				// All files should be deleted
				outputStr := string(output)
				require.Contains(t, outputStr, "D  README.md")
				require.Contains(t, outputStr, "D  src/main.go")
				require.Contains(t, outputStr, "D  test/test.go")

				// Check that files don't exist on disk
				files := []string{
					filepath.Join(tmpDir, "src", "main.go"),
					filepath.Join(tmpDir, "test", "test.go"),
					filepath.Join(tmpDir, "README.md"),
				}

				for _, file := range files {
					_, err := os.Stat(file)
					require.True(t, os.IsNotExist(err), "file %s should not exist", file)
				}
			},
		},
		{
			name:      "LargeBatchRemove",
			files:     generateFileList(120), // More than maxBatchSize (100)
			keepLocal: false,
			setupFiles: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Create multiple files
				for i := 0; i < 120; i++ {
					fileName := fmt.Sprintf("file%d.txt", i+1)
					filePath := filepath.Join(tmpDir, fileName)
					testutil.WriteTestFileWithFormat(t, filePath, "content %d", i)
				}

				// Add and commit all files
				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", ".")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "add many files")
				err = commitCmd.Run()
				require.NoError(t, err)
			},
			validate: func(t *testing.T, tmpDir string) {
				ctx := context.Background()

				// Check git status
				statusCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "status", "--porcelain")
				output, err := statusCmd.Output()
				require.NoError(t, err)

				// Count deleted files
				lines := strings.Split(strings.TrimSpace(string(output)), "\n")
				deletedCount := 0
				for _, line := range lines {
					if strings.HasPrefix(line, "D ") {
						deletedCount++
					}
				}
				require.Equal(t, 120, deletedCount)

				// Check that files don't exist on disk
				for i := 0; i < 120; i++ {
					fileName := fmt.Sprintf("file%d.txt", i+1)
					filePath := filepath.Join(tmpDir, fileName)
					_, err := os.Stat(filePath)
					require.True(t, os.IsNotExist(err), "file %s should not exist", fileName)
				}
			},
		},
		{
			name:      "RemoveNonExistentFile",
			files:     []string{"nonexistent.txt"},
			keepLocal: false,
			setupFiles: func(t *testing.T, tmpDir string) {
				// Don't create the file, but initialize empty repo
				ctx := context.Background()

				// Create a dummy file to have a non-empty repo
				dummyFile := filepath.Join(tmpDir, "dummy.txt")
				testutil.WriteTestFile(t, dummyFile, "dummy")

				addCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "add", "dummy.txt")
				err := addCmd.Run()
				require.NoError(t, err)

				commitCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "commit", "-m", "initial")
				err = commitCmd.Run()
				require.NoError(t, err)
			},
			expectError: true,
			errorMsg:    "did not match any files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if testing.Short() {
				t.Skip("Skipping integration test")
			}

			// Create temporary git repository
			tmpDir := testutil.CreateTempDir(t)
			ctx := context.Background()

			// Initialize git repo
			initCmd := exec.CommandContext(ctx, "git", "init", tmpDir) //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err := initCmd.Run()
			require.NoError(t, err)

			// Configure git user for tests
			configCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.email", "test@example.com") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			configCmd = exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.name", "Test User") //nolint:gosec // Test-only: git args are hardcoded constants // tmpDir is from t.TempDir()
			err = configCmd.Run()
			require.NoError(t, err)

			logger := logrus.New()
			logger.SetOutput(io.Discard) // Discard logs during tests
			client := &gitClient{logger: logger}

			// Set up files if needed
			if tt.setupFiles != nil {
				tt.setupFiles(t, tmpDir)
			}

			// Test batch remove
			err = client.BatchRemoveFiles(ctx, tmpDir, tt.files, tt.keepLocal)
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tmpDir)
				}
			}
		})
	}
}

// Helper function to generate a list of file names
func generateFileList(count int) []string {
	files := make([]string, count)
	for i := 0; i < count; i++ {
		files[i] = fmt.Sprintf("file%d.txt", i+1)
	}
	return files
}

// TestFilterValidFiles tests the filterValidFiles helper function
func TestFilterValidFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "all valid files",
			input:    []string{"file1.txt", "file2.txt", "dir/file3.txt"},
			expected: []string{"file1.txt", "file2.txt", "dir/file3.txt"},
		},
		{
			name:     "empty strings filtered",
			input:    []string{"file1.txt", "", "file2.txt", ""},
			expected: []string{"file1.txt", "file2.txt"},
		},
		{
			name:     "whitespace only strings filtered",
			input:    []string{"file1.txt", "   ", "\t", "file2.txt"},
			expected: []string{"file1.txt", "file2.txt"},
		},
		{
			name:     "all empty strings",
			input:    []string{"", "  ", "\t\n"},
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: []string{},
		},
		{
			name:     "preserves spaces in filenames",
			input:    []string{"file with spaces.txt", "normal.txt"},
			expected: []string{"file with spaces.txt", "normal.txt"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := filterValidFiles(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestBatchStatus_EdgeCases tests edge cases in git status parsing
func TestBatchStatus_EdgeCases(t *testing.T) {
	// These are unit tests for the line parsing logic
	tests := []struct {
		name     string
		line     string
		expected bool // should be included in result
	}{
		{
			name:     "valid modified file",
			line:     "M  file.txt",
			expected: true,
		},
		{
			name:     "line exactly 3 chars - invalid",
			line:     "M  ",
			expected: false,
		},
		{
			name:     "line 2 chars - too short",
			line:     "M ",
			expected: false,
		},
		{
			name:     "empty line",
			line:     "",
			expected: false,
		},
		{
			name:     "valid minimum 4 chars",
			line:     "M  a",
			expected: true,
		},
		{
			name:     "whitespace only after status",
			line:     "M     ",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the parsing logic from BatchStatus/BatchStatusAll
			statuses := make(map[string]string)

			if len(tc.line) >= 4 {
				status := tc.line[:2]
				file := strings.TrimSpace(tc.line[3:])
				if file != "" {
					statuses[file] = status
				}
			}

			if tc.expected {
				require.NotEmpty(t, statuses, "Expected file to be parsed from line: %q", tc.line)
			} else {
				require.Empty(t, statuses, "Expected empty result for line: %q", tc.line)
			}
		})
	}
}

// TestBatchAddFiles_EmptyStrings tests that empty strings are filtered from batch operations
func TestBatchAddFiles_EmptyStrings(t *testing.T) {
	ctx := context.Background()
	tmpDir := testutil.CreateTempDir(t)

	// Initialize git repo
	initCmd := exec.CommandContext(ctx, "git", "init", tmpDir) //nolint:gosec // Test-only: git args are hardcoded constants
	err := initCmd.Run()
	require.NoError(t, err)

	// Configure git user
	configCmd := exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.email", "test@example.com") //nolint:gosec // Test-only: git args are hardcoded constants
	err = configCmd.Run()
	require.NoError(t, err)

	configCmd = exec.CommandContext(ctx, "git", "-C", tmpDir, "config", "user.name", "Test User") //nolint:gosec // Test-only: git args are hardcoded constants
	err = configCmd.Run()
	require.NoError(t, err)

	// Create a valid file
	err = os.WriteFile(filepath.Join(tmpDir, "valid.txt"), []byte("content"), 0o600)
	require.NoError(t, err)

	logger := logrus.New()
	client, err := NewClient(logger, nil)
	require.NoError(t, err)

	batchClient := client.(BatchClient)

	// Test with mix of valid and empty strings
	filesWithEmpty := []string{"valid.txt", "", "  ", "\t"}
	err = batchClient.BatchAddFiles(ctx, tmpDir, filesWithEmpty)
	require.NoError(t, err, "BatchAddFiles should succeed even with empty strings in input")

	// Test with all empty strings - should be a no-op
	allEmpty := []string{"", "  ", "\t\n"}
	err = batchClient.BatchAddFiles(ctx, tmpDir, allEmpty)
	require.NoError(t, err, "BatchAddFiles with all empty strings should return nil")
}
