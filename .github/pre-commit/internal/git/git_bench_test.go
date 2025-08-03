package git

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkRepository_GetAllFiles measures performance of getting all tracked files
func BenchmarkRepository_GetAllFiles(b *testing.B) {
	// Find git repository root
	root, err := FindRepositoryRoot()
	if err != nil {
		b.Skip("Not in a git repository")
	}

	repo := NewRepository(root)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetAllFiles()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRepository_GetStagedFiles measures performance of getting staged files
func BenchmarkRepository_GetStagedFiles(b *testing.B) {
	// Find git repository root
	root, err := FindRepositoryRoot()
	if err != nil {
		b.Skip("Not in a git repository")
	}

	repo := NewRepository(root)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetStagedFiles()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRepository_GetModifiedFiles measures performance of getting modified files
func BenchmarkRepository_GetModifiedFiles(b *testing.B) {
	// Find git repository root
	root, err := FindRepositoryRoot()
	if err != nil {
		b.Skip("Not in a git repository")
	}

	repo := NewRepository(root)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetModifiedFiles()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRepository_IsFileTracked measures performance of file tracking checks
func BenchmarkRepository_IsFileTracked(b *testing.B) {
	// Find git repository root
	root, err := FindRepositoryRoot()
	if err != nil {
		b.Skip("Not in a git repository")
	}

	repo := NewRepository(root)

	// Use this test file as it should be tracked
	testFile := "repository_test.go"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.IsFileTracked(testFile)
	}
}

// BenchmarkRepository_GetFileContent measures performance of reading file content
func BenchmarkRepository_GetFileContent(b *testing.B) {
	// Find git repository root
	root, err := FindRepositoryRoot()
	if err != nil {
		b.Skip("Not in a git repository")
	}

	repo := NewRepository(root)

	// Use this test file
	testFile := "repository_test.go"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetFileContent(testFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFindRepositoryRoot measures performance of finding repository root
func BenchmarkFindRepositoryRoot(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := FindRepositoryRoot()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseFileList measures performance of parsing git output
func BenchmarkParseFileList(b *testing.B) {
	// Create a large git output sample
	var output string
	for i := 0; i < 1000; i++ {
		output += fmt.Sprintf("file%d.go\n", i)
		output += fmt.Sprintf("pkg/file%d.go\n", i)
		output += fmt.Sprintf("cmd/file%d.go\n", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseFileList([]byte(output))
	}
}

// BenchmarkInstaller_InstallHook measures performance of hook installation
func BenchmarkInstaller_InstallHook(b *testing.B) {
	// Create a temporary directory for the test
	tmpDir := b.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")

	err := os.MkdirAll(hooksDir, 0o755)
	if err != nil {
		b.Fatal(err)
	}

	installer := NewInstaller(tmpDir, "/tmp/gofortress")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Install and uninstall to measure the complete cycle
		err := installer.InstallHook("pre-commit", false)
		if err != nil {
			b.Fatal(err)
		}

		_, err = installer.UninstallHook("pre-commit")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkInstaller_IsHookInstalled measures performance of hook status check
func BenchmarkInstaller_IsHookInstalled(b *testing.B) {
	// Create a temporary directory with an installed hook
	tmpDir := b.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	hooksDir := filepath.Join(gitDir, "hooks")

	err := os.MkdirAll(hooksDir, 0o755)
	if err != nil {
		b.Fatal(err)
	}

	installer := NewInstaller(tmpDir, "/tmp/gofortress")

	// Install a hook once
	err = installer.InstallHook("pre-commit", false)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		installer.IsHookInstalled("pre-commit")
	}
}

// BenchmarkRepository_Parallel measures parallel git operations
func BenchmarkRepository_Parallel(b *testing.B) {
	// Find git repository root
	root, err := FindRepositoryRoot()
	if err != nil {
		b.Skip("Not in a git repository")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		repo := NewRepository(root)
		for pb.Next() {
			_, err := repo.GetAllFiles()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
