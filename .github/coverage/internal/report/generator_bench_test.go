package report

import (
	"context"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

// BenchmarkGenerate benchmarks basic report generation performance
func BenchmarkGenerate(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()
	coverage := createBenchmarkCoverageData(10, 50) // 10 packages, 50 files total

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, coverage)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateSmall benchmarks small report generation
func BenchmarkGenerateSmall(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()
	coverage := createBenchmarkCoverageData(3, 10) // 3 packages, 10 files total

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, coverage)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateLarge benchmarks large report generation
func BenchmarkGenerateLarge(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()
	coverage := createBenchmarkCoverageData(50, 500) // 50 packages, 500 files total

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, coverage)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateVeryLarge benchmarks very large report generation
func BenchmarkGenerateVeryLarge(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()
	coverage := createBenchmarkCoverageData(100, 1000) // 100 packages, 1000 files total

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, coverage)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGenerateWithAllOptions benchmarks generation with all options
func BenchmarkGenerateWithAllOptions(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()
	coverage := createBenchmarkCoverageData(20, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(ctx, coverage,
			WithTheme("github-dark"),
			WithTitle("Complex Coverage Report"),
			WithPackages(true),
			WithFiles(true),
			WithMissing(true),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBuildReportData benchmarks report data construction
func BenchmarkBuildReportData(b *testing.B) { //nolint:revive // function naming
	generator := New()
	coverage := createBenchmarkCoverageData(20, 100)
	config := generator.config

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.buildReportData(coverage, config)
	}
}

// BenchmarkRenderHTML benchmarks HTML template rendering
func BenchmarkRenderHTML(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()
	coverage := createBenchmarkCoverageData(20, 100)
	reportData := generator.buildReportData(coverage, generator.config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.renderHTML(ctx, reportData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBuildLineReports benchmarks line report construction
func BenchmarkBuildLineReports(b *testing.B) { //nolint:revive // function naming
	generator := New()
	fileCov := createBenchmarkFileCoverage(1000) // File with 1000 statements

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.buildLineReports(fileCov)
	}
}

// BenchmarkGetStatusClass benchmarks status class calculation
func BenchmarkGetStatusClass(b *testing.B) { //nolint:revive // function naming
	generator := New()
	percentages := []float64{0.0, 25.5, 50.0, 75.3, 100.0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		percentage := percentages[i%len(percentages)]
		_ = generator.getStatusClass(percentage)
	}
}

// BenchmarkExtractFileName benchmarks file name extraction
func BenchmarkExtractFileName(b *testing.B) { //nolint:revive // function naming
	generator := New()
	paths := []string{
		"github.com/example/internal/config/config.go",
		"github.com/example/pkg/utils/helper.go",
		"github.com/example/cmd/server/main.go",
		"simple.go",
		"very/deeply/nested/path/to/file.go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		_ = generator.extractFileName(path)
	}
}

// BenchmarkGetHTMLTemplate benchmarks template retrieval
func BenchmarkGetHTMLTemplate(b *testing.B) { //nolint:revive // function naming
	generator := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.getHTMLTemplate()
	}
}

// BenchmarkMemoryAllocation benchmarks memory allocation during report generation
func BenchmarkMemoryAllocation(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()
	coverage := createBenchmarkCoverageData(10, 50)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		report, err := generator.Generate(ctx, coverage)
		if err != nil {
			b.Fatal(err)
		}
		_ = report // Prevent optimization
	}
}

// BenchmarkConcurrentGeneration benchmarks concurrent report generation
func BenchmarkConcurrentGeneration(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()
	coverage := createBenchmarkCoverageData(10, 50)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := generator.Generate(ctx, coverage)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkDifferentConfigurations benchmarks generation with different configurations
func BenchmarkDifferentConfigurations(b *testing.B) { //nolint:revive // function naming
	coverage := createBenchmarkCoverageData(10, 50)
	ctx := context.Background()

	configs := []*Config{
		{Theme: "github-dark", ShowPackages: true, ShowFiles: true},
		{Theme: "light", ShowPackages: false, ShowFiles: false},
		{Theme: "github-dark", ShowPackages: true, ShowFiles: false},
		{Theme: "light", ShowPackages: false, ShowFiles: true},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := configs[i%len(configs)]
		generator := NewWithConfig(config)
		_, err := generator.Generate(ctx, coverage)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVariousReportSizes benchmarks generation with different report sizes
func BenchmarkVariousReportSizes(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()

	// Different report sizes: (packages, files)
	sizes := []struct{ packages, files int }{
		{1, 5},    // Very small
		{5, 25},   // Small
		{10, 50},  // Medium
		{25, 125}, // Large
		{50, 250}, // Very large
	}

	coverageData := make([]*parser.CoverageData, len(sizes))
	for i, size := range sizes {
		coverageData[i] = createBenchmarkCoverageData(size.packages, size.files)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coverage := coverageData[i%len(coverageData)]
		_, err := generator.Generate(ctx, coverage)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkComplexLineReports benchmarks line report generation with complex statements
func BenchmarkComplexLineReports(b *testing.B) { //nolint:revive // function naming
	generator := New()

	// Create file coverage with overlapping statements
	fileCov := &parser.FileCoverage{
		Path: "complex.go",
		Statements: []parser.Statement{
			{StartLine: 1, EndLine: 50, Count: 1},    // Large block
			{StartLine: 25, EndLine: 75, Count: 0},   // Overlapping uncovered
			{StartLine: 50, EndLine: 100, Count: 2},  // Another overlap
			{StartLine: 150, EndLine: 200, Count: 1}, // Separate block
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.buildLineReports(fileCov)
	}
}

// BenchmarkHTMLTemplateExecution benchmarks template execution with large data
func BenchmarkHTMLTemplateExecution(b *testing.B) { //nolint:revive // function naming
	generator := New()
	ctx := context.Background()

	// Create large report data
	coverage := createBenchmarkCoverageData(50, 500)
	reportData := generator.buildReportData(coverage, generator.config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.renderHTML(ctx, reportData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReportDataConstruction benchmarks only the data construction phase
func BenchmarkReportDataConstruction(b *testing.B) { //nolint:revive // function naming
	generator := New()
	coverage := createBenchmarkCoverageData(25, 125)
	config := generator.config

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.buildReportData(coverage, config)
	}
}

// BenchmarkPackageSorting benchmarks package sorting performance
func BenchmarkPackageSorting(b *testing.B) { //nolint:revive // function naming
	generator := New()
	coverage := createBenchmarkCoverageData(100, 1000) // 100 packages

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generator.buildReportData(coverage, generator.config)
	}
}

// Helper function to create benchmark coverage data
func createBenchmarkCoverageData(numPackages, totalFiles int) *parser.CoverageData {
	packages := make(map[string]*parser.PackageCoverage)
	filesPerPackage := totalFiles / numPackages
	if filesPerPackage == 0 {
		filesPerPackage = 1
	}

	totalLines := 0
	totalCovered := 0

	for i := 0; i < numPackages; i++ {
		pkgName := "pkg" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		files := make(map[string]*parser.FileCoverage)

		pkgLines := 0
		pkgCovered := 0

		for j := 0; j < filesPerPackage; j++ {
			fileName := pkgName + "/file" + string(rune('0'+j)) + ".go"
			fileLines := 20 + (j * 5)                                            // Varying file sizes
			fileCovered := int(float64(fileLines) * (0.6 + float64(i%40)/100.0)) // Varying coverage

			files[fileName] = &parser.FileCoverage{
				Path:         fileName,
				Percentage:   float64(fileCovered) / float64(fileLines) * 100,
				TotalLines:   fileLines,
				CoveredLines: fileCovered,
				Statements:   createBenchmarkStatements(fileLines, fileCovered),
			}

			pkgLines += fileLines
			pkgCovered += fileCovered
		}

		packages[pkgName] = &parser.PackageCoverage{
			Name:         pkgName,
			Percentage:   float64(pkgCovered) / float64(pkgLines) * 100,
			TotalLines:   pkgLines,
			CoveredLines: pkgCovered,
			Files:        files,
		}

		totalLines += pkgLines
		totalCovered += pkgCovered
	}

	return &parser.CoverageData{
		Mode:         "atomic",
		Percentage:   float64(totalCovered) / float64(totalLines) * 100,
		TotalLines:   totalLines,
		CoveredLines: totalCovered,
		Timestamp:    time.Now(),
		Packages:     packages,
	}
}

// Helper function to create benchmark file coverage with many statements
func createBenchmarkFileCoverage(numStatements int) *parser.FileCoverage {
	statements := make([]parser.Statement, numStatements)
	covered := 0

	for i := 0; i < numStatements; i++ {
		count := 0
		if i%3 != 0 { // ~66% coverage
			count = i%5 + 1
			covered++
		}

		statements[i] = parser.Statement{
			StartLine: i*2 + 1,
			EndLine:   i*2 + 2,
			Count:     count,
		}
	}

	return &parser.FileCoverage{
		Path:         "benchmark.go",
		Percentage:   float64(covered) / float64(numStatements) * 100,
		TotalLines:   numStatements,
		CoveredLines: covered,
		Statements:   statements,
	}
}

// Helper function to create statements for benchmark data
func createBenchmarkStatements(totalLines, coveredLines int) []parser.Statement {
	statements := make([]parser.Statement, totalLines)

	for i := 0; i < totalLines; i++ {
		count := 0
		if i < coveredLines {
			count = (i % 5) + 1 // Varying hit counts
		}

		statements[i] = parser.Statement{
			StartLine: i + 1,
			EndLine:   i + 1,
			Count:     count,
		}
	}

	return statements
}
