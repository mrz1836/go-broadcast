package parser

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// BenchmarkParse benchmarks the parsing of coverage data
func BenchmarkParse(b *testing.B) {
	parser := New()
	ctx := context.Background()

	// Create sample coverage data
	coverageData := generateCoverageData(100) // 100 files with coverage statements

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(coverageData)
		_, err := parser.Parse(ctx, reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseSmall benchmarks parsing small coverage files
func BenchmarkParseSmall(b *testing.B) {
	parser := New()
	ctx := context.Background()

	coverageData := generateCoverageData(10) // 10 files

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(coverageData)
		_, err := parser.Parse(ctx, reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseLarge benchmarks parsing large coverage files
func BenchmarkParseLarge(b *testing.B) {
	parser := New()
	ctx := context.Background()

	coverageData := generateCoverageData(1000) // 1000 files

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(coverageData)
		_, err := parser.Parse(ctx, reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseStatement benchmarks the parsing of individual statements
func BenchmarkParseStatement(b *testing.B) {
	parser := New()
	statement := "github.com/example/pkg/file.go:10.5,12.10 2 1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := parser.parseStatement(statement)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParsePosition benchmarks position parsing
func BenchmarkParsePosition(b *testing.B) {
	parser := New()
	position := "10.15"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := parser.parsePosition(position)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkShouldExcludeFile benchmarks file exclusion logic
func BenchmarkShouldExcludeFile(b *testing.B) {
	parser := New()
	filename := "internal/config/config.go"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.shouldExcludeFile(filename)
	}
}

// BenchmarkExtractPackageName benchmarks package name extraction
func BenchmarkExtractPackageName(b *testing.B) {
	parser := New()
	filename := "github.com/example/internal/config/config.go"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.extractPackageName(filename)
	}
}

// BenchmarkCalculateFileCoverage benchmarks coverage calculation
func BenchmarkCalculateFileCoverage(b *testing.B) {
	parser := New()
	statements := []Statement{
		{StartLine: 10, NumStmt: 2, Count: 1},
		{StartLine: 15, NumStmt: 3, Count: 0},
		{StartLine: 20, NumStmt: 1, Count: 2},
		{StartLine: 25, NumStmt: 4, Count: 1},
		{StartLine: 30, NumStmt: 2, Count: 0},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.calculateFileCoverage("test.go", statements)
	}
}

// BenchmarkBuildCoverageData benchmarks the final data structure building
func BenchmarkBuildCoverageData(b *testing.B) {
	parser := New()
	statements := generateStatements(100) // 100 statements across multiple files

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.buildCoverageData("atomic", statements)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseWithExclusions benchmarks parsing with various exclusion patterns
func BenchmarkParseWithExclusions(b *testing.B) {
	config := &Config{
		ExcludePaths:     []string{"test/", "vendor/", "examples/", "third_party/", "testdata/", "generated/"},
		ExcludeFiles:     []string{"*_test.go", "*.pb.go", "*_mock.go", "mock_*.go", "*.gen.go"},
		ExcludeGenerated: true,
		ExcludeTestFiles: true,
		MinFileLines:     10,
	}
	parser := NewWithConfig(config)
	ctx := context.Background()

	// Generate coverage data with mixed file types (some should be excluded)
	coverageData := generateMixedCoverageData(200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(coverageData)
		_, err := parser.Parse(ctx, reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// generateCoverageData creates sample coverage data for benchmarking
func generateCoverageData(numFiles int) string {
	var builder strings.Builder
	builder.WriteString("mode: atomic\n")

	for i := 0; i < numFiles; i++ {
		pkg := fmt.Sprintf("pkg%d", i%10) // 10 different packages
		file := fmt.Sprintf("github.com/example/%s/file%d.go", pkg, i)

		// Generate 5-10 statements per file
		for j := 0; j < 5+i%6; j++ {
			line := 10 + j*5
			builder.WriteString(fmt.Sprintf("%s:%d.1,%d.10 %d %d\n",
				file, line, line+2, 1+j%3, i%3))
		}
	}

	return builder.String()
}

// generateMixedCoverageData creates coverage data with files that should be excluded
func generateMixedCoverageData(numFiles int) string {
	var builder strings.Builder
	builder.WriteString("mode: atomic\n")

	fileTypes := []string{
		"github.com/example/internal/config.go",
		"github.com/example/internal/config_test.go", // should be excluded
		"github.com/example/vendor/lib/external.go",  // should be excluded
		"github.com/example/internal/service.pb.go",  // should be excluded
		"github.com/example/mocks/mock_service.go",   // should be excluded
		"github.com/example/pkg/utils.go",
		"github.com/example/testdata/helper.go", // should be excluded
	}

	for i := 0; i < numFiles; i++ {
		file := fileTypes[i%len(fileTypes)]
		file = strings.Replace(file, ".go", fmt.Sprintf("%d.go", i), 1)

		// Generate statements
		for j := 0; j < 3; j++ {
			line := 10 + j*5
			builder.WriteString(fmt.Sprintf("%s:%d.1,%d.10 1 %d\n",
				file, line, line+2, i%2))
		}
	}

	return builder.String()
}

// generateStatements creates test statements for benchmarking
func generateStatements(numStatements int) []StatementWithFile {
	statements := make([]StatementWithFile, numStatements)

	for i := 0; i < numStatements; i++ {
		pkg := fmt.Sprintf("pkg%d", i%5)
		filename := fmt.Sprintf("github.com/example/%s/file%d.go", pkg, i%20)

		statements[i] = StatementWithFile{
			Statement: Statement{
				StartLine: 10 + i%50,
				StartCol:  1,
				EndLine:   12 + i%50,
				EndCol:    10,
				NumStmt:   1 + i%3,
				Count:     i % 3,
			},
			Filename: filename,
		}
	}

	return statements
}

// BenchmarkMemoryAllocation benchmarks memory allocation during parsing
func BenchmarkMemoryAllocation(b *testing.B) {
	parser := New()
	ctx := context.Background()
	coverageData := generateCoverageData(50)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(coverageData)
		coverage, err := parser.Parse(ctx, reader)
		if err != nil {
			b.Fatal(err)
		}
		_ = coverage // Prevent optimization
	}
}

// BenchmarkConcurrentParsing benchmarks concurrent parsing operations
func BenchmarkConcurrentParsing(b *testing.B) {
	parser := New()
	ctx := context.Background()
	coverageData := generateCoverageData(30)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reader := strings.NewReader(coverageData)
			_, err := parser.Parse(ctx, reader)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
