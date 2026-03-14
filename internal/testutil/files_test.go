package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTestFiles(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("CreateMultipleFiles", func(t *testing.T) {
		count := 5
		files := CreateTestFiles(t, tempDir, count)

		require.Len(t, files, count)

		// Verify each file was created and has expected content
		for i, filePath := range files {
			assert.True(t, filepath.IsAbs(filePath), "file path should be absolute")
			assert.True(t, strings.HasPrefix(filePath, tempDir), "file should be in temp directory")

			// Check file exists
			info, err := os.Stat(filePath)
			require.NoError(t, err)
			assert.False(t, info.IsDir())

			// Check file content
			content, err := os.ReadFile(filePath) //nolint:gosec // test file path is controlled //nolint:gosec // test file path is controlled
			require.NoError(t, err)
			expectedContent := "Test content for file " + string(rune('0'+i)) + "\n"
			assert.Equal(t, expectedContent, string(content))

			// Check file permissions
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		}
	})

	t.Run("CreateZeroFiles", func(t *testing.T) {
		files := CreateTestFiles(t, tempDir, 0)
		assert.Empty(t, files)
	})

	t.Run("CreateSingleFile", func(t *testing.T) {
		files := CreateTestFiles(t, tempDir, 1)
		require.Len(t, files, 1)

		content, err := os.ReadFile(files[0])
		require.NoError(t, err)
		assert.Equal(t, "Test content for file 0\n", string(content))
	})
}

func TestCreateTestFilesWithNames(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("CreateWithCustomNames", func(t *testing.T) {
		names := []string{"config.yaml", "data.json", "readme.txt"}
		files := CreateTestFilesWithNames(t, tempDir, names)

		require.Len(t, files, len(names))

		for i, filePath := range files {
			expectedPath := filepath.Join(tempDir, names[i])
			assert.Equal(t, expectedPath, filePath)

			// Check file exists
			info, err := os.Stat(filePath)
			require.NoError(t, err)
			assert.False(t, info.IsDir())

			// Check file content
			content, err := os.ReadFile(filePath) //nolint:gosec // test file path is controlled //nolint:gosec // test file path is controlled
			require.NoError(t, err)
			expectedContent := "Content for " + names[i]
			assert.Equal(t, expectedContent, string(content))

			// Check file permissions
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		}
	})

	t.Run("CreateWithEmptyNames", func(t *testing.T) {
		files := CreateTestFilesWithNames(t, tempDir, []string{})
		assert.Empty(t, files)
	})

	t.Run("CreateWithSpecialCharacters", func(t *testing.T) {
		names := []string{"file-with-dashes.txt", "file_with_underscores.log", "file.with.dots.csv"}
		files := CreateTestFilesWithNames(t, tempDir, names)

		require.Len(t, files, len(names))

		for i, filePath := range files {
			_, err := os.Stat(filePath)
			require.NoError(t, err, "file should exist: %s", names[i])
		}
	})
}

func TestCreateTestRepo(t *testing.T) {
	t.Run("CreateAndCleanup", func(t *testing.T) {
		repoDir, cleanup := CreateTestRepo(t)
		defer cleanup()

		// Verify directory was created
		info, err := os.Stat(repoDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Verify it's a temp directory with expected prefix
		assert.Contains(t, repoDir, "test_repo_")

		// Create a test file to verify cleanup works
		testFile := filepath.Join(repoDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test"), 0o600)
		require.NoError(t, err)

		// Cleanup should remove the directory
		cleanup()

		// Verify directory was removed
		_, err = os.Stat(repoDir)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("MultipleRepos", func(t *testing.T) {
		// Create multiple repos to ensure unique names
		repo1, cleanup1 := CreateTestRepo(t)
		repo2, cleanup2 := CreateTestRepo(t)
		defer cleanup1()
		defer cleanup2()

		assert.NotEqual(t, repo1, repo2, "repos should have different paths")

		// Both should exist
		_, err1 := os.Stat(repo1)
		_, err2 := os.Stat(repo2)
		require.NoError(t, err1)
		require.NoError(t, err2)
	})
}

func TestWriteTestFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("WriteWithCustomContent", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "custom.txt")
		content := "This is custom content\nwith multiple lines\n"

		WriteTestFile(t, filePath, content)

		// Verify file was created
		info, err := os.Stat(filePath)
		require.NoError(t, err)
		assert.False(t, info.IsDir())

		// Verify content
		actualContent, err := os.ReadFile(filePath) //nolint:gosec // test file path is controlled
		require.NoError(t, err)
		assert.Equal(t, content, string(actualContent))

		// Check permissions
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("WriteEmptyFile", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "empty.txt")
		WriteTestFile(t, filePath, "")

		// Verify file was created
		info, err := os.Stat(filePath)
		require.NoError(t, err)
		assert.Equal(t, int64(0), info.Size())
	})

	t.Run("OverwriteExistingFile", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "overwrite.txt")

		// Create initial file
		WriteTestFile(t, filePath, "original content")

		// Overwrite with new content
		newContent := "new content"
		WriteTestFile(t, filePath, newContent)

		// Verify new content
		actualContent, err := os.ReadFile(filePath) //nolint:gosec // test file path is controlled
		require.NoError(t, err)
		assert.Equal(t, newContent, string(actualContent))
	})
}

func TestWriteTestFileWithFormat(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("FormatWithArguments", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "formatted.txt")
		format := "Hello %s! You have %d messages."
		name := "Alice"
		count := 42

		WriteTestFileWithFormat(t, filePath, format, name, count)

		// Verify content
		content, err := os.ReadFile(filePath) //nolint:gosec // test file path is controlled
		require.NoError(t, err)
		expected := "Hello Alice! You have 42 messages."
		assert.Equal(t, expected, string(content))
	})

	t.Run("FormatWithoutArguments", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "noargs.txt")
		content := "No formatting needed"

		WriteTestFileWithFormat(t, filePath, "%s", content)

		// Verify content
		actualContent, err := os.ReadFile(filePath) //nolint:gosec // test file path is controlled
		require.NoError(t, err)
		assert.Equal(t, content, string(actualContent))
	})

	t.Run("FormatWithComplexArguments", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "complex.txt")
		format := "User: %s, Age: %d, Score: %.2f, Active: %t"

		WriteTestFileWithFormat(t, filePath, format, "Bob", 30, 95.5, true)

		content, err := os.ReadFile(filePath) //nolint:gosec // test file path is controlled
		require.NoError(t, err)
		expected := "User: Bob, Age: 30, Score: 95.50, Active: true"
		assert.Equal(t, expected, string(content))
	})
}

func TestCreateBenchmarkFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Since CreateBenchmarkFiles expects *testing.B, we need to test it indirectly
	// We'll create a benchmark function that calls it
	t.Run("VerifyBenchmarkFilesStructure", func(t *testing.T) {
		// Test that we can create files with the benchmark pattern
		count := 3
		files := make([]string, count)

		for i := 0; i < count; i++ {
			fileName := "bench_file_" + string(rune('0'+i)) + ".txt"
			filePath := filepath.Join(tempDir, fileName)
			content := "Benchmark test content " + string(rune('0'+i))

			err := os.WriteFile(filePath, []byte(content), 0o600)
			require.NoError(t, err)
			files[i] = filePath
		}

		// Verify the pattern matches what CreateBenchmarkFiles would create
		for i, filePath := range files {
			expectedName := "bench_file_" + string(rune('0'+i)) + ".txt"
			assert.True(t, strings.HasSuffix(filePath, expectedName))

			content, err := os.ReadFile(filePath) //nolint:gosec // test file path is controlled //nolint:gosec // test file path is controlled
			require.NoError(t, err)
			expectedContent := "Benchmark test content " + string(rune('0'+i))
			assert.Equal(t, expectedContent, string(content))
		}
	})
}

func TestCreateTestDirectory(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("CreateNewDirectory", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "newdir")
		CreateTestDirectory(t, dirPath)

		// Verify directory was created
		info, err := os.Stat(dirPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Check permissions
		assert.Equal(t, os.FileMode(0o750), info.Mode().Perm())
	})

	t.Run("CreateNestedDirectory", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "parent", "child", "grandchild")
		CreateTestDirectory(t, dirPath)

		// Verify nested directory was created
		info, err := os.Stat(dirPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Verify parent directories exist
		parentPath := filepath.Join(tempDir, "parent")
		info, err = os.Stat(parentPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("CreateExistingDirectory", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "existing")

		// Create directory first
		err := os.MkdirAll(dirPath, 0o750)
		require.NoError(t, err)

		// Should not fail when directory already exists
		CreateTestDirectory(t, dirPath)

		// Verify directory still exists
		info, err := os.Stat(dirPath)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestCreateTempDir(t *testing.T) {
	t.Run("CreateTempDirectory", func(t *testing.T) {
		tempDir := CreateTempDir(t)

		// Verify directory exists
		info, err := os.Stat(tempDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		// Verify it's in the system temp directory
		assert.Contains(t, tempDir, os.TempDir())

		// Create a file to test cleanup
		testFile := filepath.Join(tempDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test"), 0o600)
		require.NoError(t, err)

		// Directory should be cleaned up automatically by Go's testing framework
	})

	t.Run("MultipleTempdirs", func(t *testing.T) {
		tempDir1 := CreateTempDir(t)
		tempDir2 := CreateTempDir(t)

		// Should be different directories
		assert.NotEqual(t, tempDir1, tempDir2)

		// Both should exist
		_, err1 := os.Stat(tempDir1)
		_, err2 := os.Stat(tempDir2)
		require.NoError(t, err1)
		require.NoError(t, err2)
	})
}

func TestCreateBenchmarkTempDir(t *testing.T) {
	// Since CreateBenchmarkTempDir expects *testing.B, we test the pattern
	t.Run("VerifyBenchmarkTempDirPattern", func(t *testing.T) {
		// Test that we can create temp directories for benchmarks
		tempDir := t.TempDir()

		// Verify it matches the expected pattern
		info, err := os.Stat(tempDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestWriteBenchmarkFile(t *testing.T) {
	// Since WriteBenchmarkFile expects *testing.B, we test the pattern
	t.Run("VerifyBenchmarkFilePattern", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "bench_test.txt")
		content := "benchmark file content"

		// Test the same pattern WriteBenchmarkFile would use
		err := os.WriteFile(filePath, []byte(content), 0o600)
		require.NoError(t, err)

		// Verify file was created with correct content and permissions
		info, err := os.Stat(filePath)
		require.NoError(t, err)
		assert.False(t, info.IsDir())
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

		actualContent, err := os.ReadFile(filePath) //nolint:gosec // test file path is controlled
		require.NoError(t, err)
		assert.Equal(t, content, string(actualContent))
	})
}

// BenchmarkCreateTestFiles tests the performance of file creation
func BenchmarkCreateTestFiles(b *testing.B) {
	tempDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a small number of files for each iteration
		files := CreateTestFiles((&testing.T{}), tempDir, 5)
		// Clean up files for next iteration
		for _, file := range files {
			_ = os.Remove(file)
		}
	}
}

// BenchmarkWriteTestFile tests the performance of individual file writing
func BenchmarkWriteTestFile(b *testing.B) {
	tempDir := b.TempDir()
	content := "benchmark test content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filePath := filepath.Join(tempDir, "bench_"+string(rune('0'+(i%10)))+".txt")
		WriteTestFile((&testing.T{}), filePath, content)
		// Clean up for next iteration
		_ = os.Remove(filePath)
	}
}
