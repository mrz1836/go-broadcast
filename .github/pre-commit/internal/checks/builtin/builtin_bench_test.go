package builtin

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// BenchmarkWhitespaceCheck_Run_SmallFile measures performance on small files
func BenchmarkWhitespaceCheck_Run_SmallFile(b *testing.B) {
	check := &WhitespaceCheck{}
	ctx := context.Background()

	// Create a temporary file with whitespace issues
	tmpDir := b.TempDir()
	testFile := fmt.Sprintf("%s/test.txt", tmpDir)
	content := "line 1\nline 2 \t \nline 3\n"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		b.Fatal(err)
	}

	files := []string{testFile}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Run(ctx, files)
	}
}

// BenchmarkWhitespaceCheck_Run_LargeFile measures performance on large files
func BenchmarkWhitespaceCheck_Run_LargeFile(b *testing.B) {
	check := &WhitespaceCheck{}
	ctx := context.Background()

	// Create a large temporary file
	tmpDir := b.TempDir()
	testFile := fmt.Sprintf("%s/large.txt", tmpDir)

	// Generate 10KB of content with whitespace issues
	var content string
	for i := 0; i < 1000; i++ {
		content += fmt.Sprintf("line %d \t \n", i)
	}

	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		b.Fatal(err)
	}

	files := []string{testFile}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Run(ctx, files)
	}
}

// BenchmarkWhitespaceCheck_Run_MultipleFiles measures performance with multiple files
func BenchmarkWhitespaceCheck_Run_MultipleFiles(b *testing.B) {
	check := &WhitespaceCheck{}
	ctx := context.Background()

	tmpDir := b.TempDir()
	var files []string

	// Create 10 test files
	for i := 0; i < 10; i++ {
		testFile := fmt.Sprintf("%s/test%d.txt", tmpDir, i)
		content := "line 1 \nline 2\t\nline 3 \t \n"
		err := os.WriteFile(testFile, []byte(content), 0o600)
		if err != nil {
			b.Fatal(err)
		}
		files = append(files, testFile)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Run(ctx, files)
	}
}

// BenchmarkEOFCheck_Run_SmallFile measures EOF check performance on small files
func BenchmarkEOFCheck_Run_SmallFile(b *testing.B) {
	check := &EOFCheck{}
	ctx := context.Background()

	// Create a temporary file without final newline
	tmpDir := b.TempDir()
	testFile := fmt.Sprintf("%s/test.txt", tmpDir)
	content := "line 1\nline 2\nline 3"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		b.Fatal(err)
	}

	files := []string{testFile}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Run(ctx, files)
	}
}

// BenchmarkEOFCheck_Run_LargeFile measures EOF check performance on large files
func BenchmarkEOFCheck_Run_LargeFile(b *testing.B) {
	check := &EOFCheck{}
	ctx := context.Background()

	// Create a large temporary file
	tmpDir := b.TempDir()
	testFile := fmt.Sprintf("%s/large.txt", tmpDir)

	// Generate 100KB of content without final newline
	var content string
	for i := 0; i < 10000; i++ {
		content += fmt.Sprintf("line %d\n", i)
	}
	content = content[:len(content)-1] // Remove final newline

	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		b.Fatal(err)
	}

	files := []string{testFile}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = check.Run(ctx, files)
	}
}

// BenchmarkWhitespaceCheck_FilterFiles measures file filtering performance
func BenchmarkWhitespaceCheck_FilterFiles(b *testing.B) {
	check := &WhitespaceCheck{}

	// Create a large list of files with mixed extensions
	files := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		switch i % 4 {
		case 0:
			files[i] = fmt.Sprintf("file%d.go", i)
		case 1:
			files[i] = fmt.Sprintf("file%d.txt", i)
		case 2:
			files[i] = fmt.Sprintf("file%d.bin", i) // Binary file
		case 3:
			files[i] = fmt.Sprintf("file%d.md", i)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		check.FilterFiles(files)
	}
}

// BenchmarkEOFCheck_FilterFiles measures EOF filter performance
func BenchmarkEOFCheck_FilterFiles(b *testing.B) {
	check := &EOFCheck{}

	// Create a large list of files
	files := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		files[i] = fmt.Sprintf("file%d.txt", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		check.FilterFiles(files)
	}
}

// BenchmarkWhitespaceCheck_Parallel measures parallel performance
func BenchmarkWhitespaceCheck_Parallel(b *testing.B) {
	check := &WhitespaceCheck{}
	ctx := context.Background()

	// Create test file once
	tmpDir := b.TempDir()
	testFile := fmt.Sprintf("%s/test.txt", tmpDir)
	content := "line 1 \nline 2\t\nline 3 \t \n"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	if err != nil {
		b.Fatal(err)
	}

	files := []string{testFile}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = check.Run(ctx, files)
		}
	})
}
