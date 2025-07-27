package history

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/coverage/internal/parser"
)

// BenchmarkRecord benchmarks recording coverage entries
func BenchmarkRecord(b *testing.B) { //nolint:revive // function naming
	tempDir, err := os.MkdirTemp("", "history_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()
	coverage := createBenchmarkCoverage()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := tracker.Record(ctx, coverage, WithBranch("main"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRecordWithOptions benchmarks recording with all options
func BenchmarkRecordWithOptions(b *testing.B) { //nolint:revive // function naming
	tempDir, err := os.MkdirTemp("", "history_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()
	coverage := createBenchmarkCoverage()

	buildInfo := &BuildInfo{
		GoVersion:    "1.21.0",
		Platform:     "linux",
		Architecture: "amd64",
		BuildTime:    "2024-01-01T12:00:00Z",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := tracker.Record(ctx, coverage,
			WithBranch("main"),
			WithCommit("abc123", "https://github.com/test/repo/commit/abc123"),
			WithMetadata("project", "test"),
			WithMetadata("build", "123"),
			WithBuildInfo(buildInfo),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetTrend benchmarks trend retrieval
func BenchmarkGetTrend(b *testing.B) { //nolint:revive // function naming
	tempDir, err := os.MkdirTemp("", "history_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Pre-populate with entries
	for i := 0; i < 50; i++ {
		coverage := createBenchmarkCoverage()
		coverage.Percentage = float64(60 + i)
		err := tracker.Record(ctx, coverage, WithBranch("main"))
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tracker.GetTrend(ctx, WithTrendBranch("main"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetTrendLarge benchmarks trend retrieval with large dataset
func BenchmarkGetTrendLarge(b *testing.B) { //nolint:revive // function naming
	tempDir, err := os.MkdirTemp("", "history_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Pre-populate with many entries
	for i := 0; i < 500; i++ {
		coverage := createBenchmarkCoverage()
		coverage.Percentage = float64(50 + (i % 50))
		err := tracker.Record(ctx, coverage,
			WithBranch("main"),
			WithCommit("commit"+string(rune('0'+i%10)), ""),
		)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tracker.GetTrend(ctx,
			WithTrendBranch("main"),
			WithTrendDays(30),
			WithMaxDataPoints(100),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetLatestEntry benchmarks latest entry retrieval
func BenchmarkGetLatestEntry(b *testing.B) { //nolint:revive // function naming
	tempDir, err := os.MkdirTemp("", "history_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Pre-populate with entries
	for i := 0; i < 20; i++ {
		coverage := createBenchmarkCoverage()
		err := tracker.Record(ctx, coverage, WithBranch("main"))
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tracker.GetLatestEntry(ctx, "main")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCleanup benchmarks cleanup operations
func BenchmarkCleanup(b *testing.B) { //nolint:revive // function naming
	config := &Config{
		RetentionDays: 30,
		MaxEntries:    100,
		AutoCleanup:   true,
	}

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		tempDir, err := os.MkdirTemp("", "history_bench_*")
		if err != nil {
			b.Fatal(err)
		}

		config.StoragePath = tempDir
		tracker := NewWithConfig(config)
		ctx := context.Background()

		// Create many entries for cleanup
		for j := 0; j < 150; j++ {
			coverage := createBenchmarkCoverage()
			recordErr := tracker.Record(ctx, coverage, WithBranch("main"))
			if recordErr != nil {
				b.Fatal(recordErr)
			}
		}

		b.StartTimer()
		err = tracker.Cleanup(ctx)
		if err != nil {
			b.Fatal(err)
		}
		b.StopTimer()

		_ = os.RemoveAll(tempDir)
	}
}

// BenchmarkGetStatistics benchmarks statistics calculation
func BenchmarkGetStatistics(b *testing.B) { //nolint:revive // function naming
	tempDir, err := os.MkdirTemp("", "history_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Pre-populate with entries across multiple branches and projects
	branches := []string{"main", "develop", "feature-a", "feature-b"}
	projects := []string{"project-1", "project-2", "project-3"}

	for i := 0; i < 100; i++ {
		coverage := createBenchmarkCoverage()
		branch := branches[i%len(branches)]
		project := projects[i%len(projects)]

		err := tracker.Record(ctx, coverage,
			WithBranch(branch),
			WithMetadata("project", project),
		)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tracker.GetStatistics(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCalculateFileHashes benchmarks file hash calculation
func BenchmarkCalculateFileHashes(b *testing.B) { //nolint:revive // function naming
	tracker := New()
	coverage := createBenchmarkCoverageComplex()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.calculateFileHashes(coverage)
	}
}

// BenchmarkCalculatePackageStats benchmarks package statistics calculation
func BenchmarkCalculatePackageStats(b *testing.B) { //nolint:revive // function naming
	tracker := New()
	coverage := createBenchmarkCoverageComplex()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.calculatePackageStats(coverage, "main")
	}
}

// BenchmarkCalculateSummary benchmarks summary calculation
func BenchmarkCalculateSummary(b *testing.B) { //nolint:revive // function naming
	tracker := New()
	entries := createBenchmarkEntries(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.calculateSummary(entries)
	}
}

// BenchmarkAnalyzeEntries benchmarks trend analysis
func BenchmarkAnalyzeEntries(b *testing.B) { //nolint:revive // function naming
	tracker := New()
	entries := createBenchmarkEntries(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.analyzeEntries(entries)
	}
}

// BenchmarkAnalyzePeriod benchmarks period analysis
func BenchmarkAnalyzePeriod(b *testing.B) { //nolint:revive // function naming
	tracker := New()
	entries := createBenchmarkEntries(30)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.analyzePeriod(entries, 7)
	}
}

// BenchmarkCalculateVolatility benchmarks volatility calculation
func BenchmarkCalculateVolatility(b *testing.B) { //nolint:revive // function naming
	tracker := New()
	entries := createBenchmarkEntries(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.calculateVolatility(entries)
	}
}

// BenchmarkCalculateMomentum benchmarks momentum calculation
func BenchmarkCalculateMomentum(b *testing.B) { //nolint:revive // function naming
	tracker := New()
	entries := createBenchmarkEntries(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.calculateMomentum(entries)
	}
}

// BenchmarkGeneratePrediction benchmarks prediction generation
func BenchmarkGeneratePrediction(b *testing.B) { //nolint:revive // function naming
	tracker := New()
	entries := createBenchmarkEntries(15)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tracker.generatePrediction(entries)
	}
}

// BenchmarkLoadAllEntries benchmarks loading all entries from storage
func BenchmarkLoadAllEntries(b *testing.B) { //nolint:revive // function naming
	tempDir, err := os.MkdirTemp("", "history_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Pre-populate with entries
	for i := 0; i < 100; i++ {
		coverage := createBenchmarkCoverage()
		err := tracker.Record(ctx, coverage, WithBranch("main"))
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tracker.loadAllEntries(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMemoryAllocation benchmarks memory allocation during operations
func BenchmarkMemoryAllocation(b *testing.B) { //nolint:revive // function naming
	tempDir, err := os.MkdirTemp("", "history_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()
	coverage := createBenchmarkCoverage()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := tracker.Record(ctx, coverage, WithBranch("main"))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentRecord benchmarks concurrent recording
func BenchmarkConcurrentRecord(b *testing.B) { //nolint:revive // function naming
	tempDir, err := os.MkdirTemp("", "history_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()
	coverage := createBenchmarkCoverage()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := tracker.Record(ctx, coverage, WithBranch("main"))
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentGetTrend benchmarks concurrent trend retrieval
func BenchmarkConcurrentGetTrend(b *testing.B) { //nolint:revive // function naming
	tempDir, err := os.MkdirTemp("", "history_bench_*")
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	config := &Config{StoragePath: tempDir}
	tracker := NewWithConfig(config)
	ctx := context.Background()

	// Pre-populate with entries
	for i := 0; i < 50; i++ {
		coverage := createBenchmarkCoverage()
		err := tracker.Record(ctx, coverage, WithBranch("main"))
		if err != nil {
			b.Fatal(err)
		}
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := tracker.GetTrend(ctx, WithTrendBranch("main"))
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Helper functions for benchmarks

func createBenchmarkCoverage() *parser.CoverageData {
	return &parser.CoverageData{
		Mode:         "atomic",
		Percentage:   75.0,
		TotalLines:   200,
		CoveredLines: 150,
		Timestamp:    time.Now(),
		Packages: map[string]*parser.PackageCoverage{
			"main": {
				Name:         "main",
				Percentage:   75.0,
				TotalLines:   200,
				CoveredLines: 150,
				Files: map[string]*parser.FileCoverage{
					"main.go": {
						Path:         "main.go",
						Percentage:   75.0,
						TotalLines:   200,
						CoveredLines: 150,
						Statements: []parser.Statement{
							{StartLine: 1, EndLine: 50, Count: 1},
							{StartLine: 51, EndLine: 100, Count: 0},
							{StartLine: 101, EndLine: 200, Count: 2},
						},
					},
				},
			},
		},
	}
}

func createBenchmarkCoverageComplex() *parser.CoverageData {
	packages := make(map[string]*parser.PackageCoverage)

	for i := 0; i < 10; i++ {
		pkgName := "package" + string(rune('A'+i))
		files := make(map[string]*parser.FileCoverage)

		for j := 0; j < 5; j++ {
			fileName := pkgName + "/file" + string(rune('0'+j)) + ".go"
			files[fileName] = &parser.FileCoverage{
				Path:         fileName,
				Percentage:   float64(70 + j*5),
				TotalLines:   100,
				CoveredLines: 70 + j*5,
				Statements: []parser.Statement{
					{StartLine: 1, EndLine: 25, Count: 1},
					{StartLine: 26, EndLine: 50, Count: 0},
					{StartLine: 51, EndLine: 75, Count: 2},
					{StartLine: 76, EndLine: 100, Count: 1},
				},
			}
		}

		packages[pkgName] = &parser.PackageCoverage{
			Name:         pkgName,
			Percentage:   75.0,
			TotalLines:   500,
			CoveredLines: 375,
			Files:        files,
		}
	}

	return &parser.CoverageData{
		Mode:         "atomic",
		Percentage:   75.0,
		TotalLines:   5000,
		CoveredLines: 3750,
		Timestamp:    time.Now(),
		Packages:     packages,
	}
}

func createBenchmarkEntries(count int) []Entry {
	entries := make([]Entry, count)

	for i := 0; i < count; i++ {
		entries[i] = Entry{
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
			Branch:    "main",
			CommitSHA: "commit" + string(rune('0'+i%10)),
			Coverage: &parser.CoverageData{
				Percentage:   float64(60 + (i % 40)), // Varying coverage
				TotalLines:   1000,
				CoveredLines: 600 + (i % 400),
			},
			Metadata: map[string]string{
				"project": "benchmark-project",
				"build":   string(rune('0' + i%10)),
			},
		}
	}

	return entries
}
