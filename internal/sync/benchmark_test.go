//go:build bench_heavy

package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
)

// BenchmarkDirectoryWalk benchmarks directory walking performance
func BenchmarkDirectoryWalk(b *testing.B) {
	testCases := []struct {
		name      string
		fileCount int
	}{
		{"SmallDirectory_10", 10},
		{"MediumDirectory_50", 50},
		{"LargeDirectory_100", 100},
		{"XLargeDirectory_500", 500},
		{"XXLargeDirectory_1000", 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create test directory
			tempDir := b.TempDir()
			sourceDir := filepath.Join(tempDir, "source")
			require.NoError(b, os.MkdirAll(sourceDir, 0o750))

			// Create test files
			for i := 0; i < tc.fileCount; i++ {
				dir := filepath.Join(sourceDir, "subdir", string('a'+rune(i%26)))
				require.NoError(b, os.MkdirAll(dir, 0o750))

				filename := filepath.Join(dir, fmt.Sprintf("file_%d.txt", i))
				require.NoError(b, os.WriteFile(filename, []byte("test content"), 0o600))
			}

			logger := logrus.NewEntry(logrus.New()).WithField("component", "benchmark")
			processor := &DirectoryProcessor{
				logger:          logger,
				progressManager: NewDirectoryProgressManager(logger),
			}

			dirMapping := config.DirectoryMapping{
				Src:  "source",
				Dest: "dest",
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				processor.exclusionEngine = NewExclusionEngine(dirMapping.Exclude)
				files, err := processor.discoverFiles(ctx, tempDir, dirMapping)
				require.NoError(b, err)
				require.Len(b, files, tc.fileCount)
			}
		})
	}
}

// BenchmarkExclusionEngine benchmarks the exclusion pattern matching
func BenchmarkExclusionEngine(b *testing.B) {
	engine := NewExclusionEngine([]string{
		"*.log",
		"*.tmp",
		"**/*.out",
		"node_modules/**",
		".git/**",
		"vendor/**",
	})

	testPaths := []string{
		"src/main.go",
		"vendor/github.com/pkg/errors/errors.go",
		"node_modules/react/index.js",
		"test/coverage.out",
		"logs/app.log",
		"build/output.tmp",
		".git/config",
		"internal/sync/sync.go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range testPaths {
			_ = engine.IsExcluded(path)
		}
	}
}

// treeOnlyClient serves one fixed tree and counts tree requests. The embedded mock
// satisfies gh.Client; the benchmark must not make any other request.
type treeOnlyClient struct {
	*gh.MockClient

	tree      *gh.GitTree
	treeCalls atomic.Int64
}

func (c *treeOnlyClient) GetGitTree(_ context.Context, _, _ string, _ bool) (*gh.GitTree, error) {
	c.treeCalls.Add(1)
	return c.tree, nil
}

// BenchmarkTargetTreeChangeDetection measures settling a target whose mapped files are
// all unchanged: one tree request, then a blob-ID comparison per file and no file requests.
func BenchmarkTargetTreeChangeDetection(b *testing.B) {
	for _, fileCount := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("Files_%d", fileCount), func(b *testing.B) {
			paths := make([]string, fileCount)
			contents := make([][]byte, fileCount)
			nodes := make([]gh.GitTreeNode, fileCount)
			for i := range fileCount {
				paths[i] = fmt.Sprintf("dir_%d/file_%d.go", i%20, i)
				contents[i] = fmt.Appendf(nil, "package dir%d\n\n// File %d holds benchmark content.\n", i%20, i)
				id, _ := gitBlobID(contents[i], 40)
				nodes[i] = gh.GitTreeNode{Path: paths[i], Mode: gitModeFile, Type: gitEntryTypeBlob, SHA: id}
			}
			client := &treeOnlyClient{MockClient: &gh.MockClient{}, tree: &gh.GitTree{Tree: nodes}}
			ctx := context.Background()

			var iterations, fileRequests int
			b.ReportAllocs()
			for b.Loop() {
				tree := newTargetTree(client, "owner/repo", "main", nil)
				for i := range fileCount {
					if _, err := tree.existingContent(ctx, paths[i], contents[i]); err != nil {
						b.Fatal(err)
					}
				}
				fileRequests += tree.stats().ContentRequests
				iterations++
			}

			b.ReportMetric(float64(client.treeCalls.Load())/float64(iterations), "tree-requests/op")
			b.ReportMetric(float64(fileRequests)/float64(iterations), "file-requests/op")
		})
	}
}

// BenchmarkCacheHitRates benchmarks cache hit rates during directory processing
func BenchmarkCacheHitRates(b *testing.B) {
	testCases := []struct {
		name          string
		cacheSize     int
		accessPattern string // "sequential", "random", "hotspot"
		fileCount     int
	}{
		{"Sequential_SmallCache_100files", 50, "sequential", 100},
		{"Sequential_LargeCache_100files", 200, "sequential", 100},
		{"Random_SmallCache_100files", 50, "random", 100},
		{"Random_LargeCache_100files", 200, "random", 100},
		{"Hotspot_SmallCache_100files", 50, "hotspot", 100},
		{"Hotspot_LargeCache_100files", 200, "hotspot", 100},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			logger := logrus.NewEntry(logrus.New()).WithField("component", "benchmark")

			// Create cache with specified size
			cache := NewContentCache(5*time.Minute, int64(tc.cacheSize*1024), logger)

			// Pre-populate some entries for hotspot pattern
			if tc.accessPattern == "hotspot" {
				ctx := context.Background()
				for i := 0; i < 10; i++ {
					key := fmt.Sprintf("hotspot-file-%d.txt", i)
					err := cache.Put(ctx, "owner/repo", "master", key, fmt.Sprintf("content-%d", i))
					require.NoError(b, err)
				}
			}

			b.ResetTimer()
			var hits, misses int64

			for i := 0; i < b.N; i++ {
				var fileIndex int
				switch tc.accessPattern {
				case "sequential":
					fileIndex = i % tc.fileCount
				case "random":
					fileIndex = i % tc.fileCount // Simplified for deterministic results
				case "hotspot":
					// 80% access to first 10 files, 20% to others
					if i%5 < 4 {
						fileIndex = i % 10
					} else {
						fileIndex = 10 + (i % (tc.fileCount - 10))
					}
				}

				fileName := fmt.Sprintf("file-%d.txt", fileIndex)
				ctx := context.Background()
				_, hit, err := cache.Get(ctx, "owner/repo", "master", fileName)
				require.NoError(b, err)

				if hit {
					atomic.AddInt64(&hits, 1)
				} else {
					atomic.AddInt64(&misses, 1)
					// Simulate cache miss by setting the content
					err := cache.Put(ctx, "owner/repo", "master", fileName, fmt.Sprintf("content-%d", fileIndex))
					require.NoError(b, err)
				}
			}

			stats := cache.GetStats()
			hitRate := float64(hits) / float64(hits+misses) * 100

			b.ReportMetric(hitRate, "cache-hit-rate-%")
			b.ReportMetric(float64(stats.Size), "cache-size")
			b.ReportMetric(float64(stats.MemoryUsage), "memory-usage-bytes")
		})
	}
}

// BenchmarkMemoryAllocationPatterns benchmarks memory allocation for different directory sizes
func BenchmarkMemoryAllocationPatterns(b *testing.B) {
	testCases := []struct {
		name      string
		fileCount int
		dirDepth  int
	}{
		{"SmallFlat_50files_1level", 50, 1},
		{"SmallDeep_50files_5levels", 50, 5},
		{"MediumFlat_200files_1level", 200, 1},
		{"MediumDeep_200files_5levels", 200, 5},
		{"LargeFlat_1000files_1level", 1000, 1},
		{"LargeDeep_1000files_10levels", 1000, 10},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create test directory structure
			tempDir := b.TempDir()
			sourceDir := filepath.Join(tempDir, "source")
			require.NoError(b, os.MkdirAll(sourceDir, 0o750))

			// Create files with specified depth
			for i := 0; i < tc.fileCount; i++ {
				// Create nested directory structure
				var dirPath string
				if tc.dirDepth > 1 {
					subDirs := make([]string, tc.dirDepth-1)
					for d := 0; d < tc.dirDepth-1; d++ {
						subDirs[d] = fmt.Sprintf("level_%d", d)
					}
					dirPath = filepath.Join(sourceDir, filepath.Join(subDirs...))
				} else {
					dirPath = sourceDir
				}

				require.NoError(b, os.MkdirAll(dirPath, 0o750))

				filename := filepath.Join(dirPath, fmt.Sprintf("file_%d.txt", i))
				content := fmt.Sprintf("test content for file %d\nwith multiple lines\nto simulate realistic file sizes", i)
				require.NoError(b, os.WriteFile(filename, []byte(content), 0o600))
			}

			logger := logrus.NewEntry(logrus.New()).WithField("component", "benchmark")
			processor := &DirectoryProcessor{
				logger:          logger,
				progressManager: NewDirectoryProgressManager(logger),
			}

			dirMapping := config.DirectoryMapping{
				Src:  "source",
				Dest: "dest",
			}

			// Force garbage collection before benchmark
			runtime.GC()
			var m1, m2 runtime.MemStats
			runtime.ReadMemStats(&m1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				processor.exclusionEngine = NewExclusionEngine(dirMapping.Exclude)
				files, err := processor.discoverFiles(ctx, tempDir, dirMapping)
				require.NoError(b, err)
				require.Len(b, files, tc.fileCount)
			}
			b.StopTimer()

			runtime.ReadMemStats(&m2)

			// Report memory statistics
			if b.N > 0 {
				allocPerOp := (m2.TotalAlloc - m1.TotalAlloc) / uint64(b.N)
				b.ReportMetric(float64(allocPerOp), "bytes-alloc-per-op")
				b.ReportMetric(float64(m2.Mallocs-m1.Mallocs)/float64(b.N), "mallocs-per-op")
			}
		})
	}
}

// BenchmarkConcurrentDirectoryProcessing benchmarks memory usage during concurrent processing
func BenchmarkConcurrentDirectoryProcessing(b *testing.B) {
	testCases := []struct {
		name        string
		workerCount int
		filesPerDir int
		dirCount    int
	}{
		{"LowConcurrency_2workers_100files_5dirs", 2, 100, 5},
		{"MediumConcurrency_5workers_200files_10dirs", 5, 200, 10},
		{"HighConcurrency_10workers_500files_20dirs", 10, 500, 20},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create test directory structure
			tempDir := b.TempDir()

			// Create multiple source directories
			var dirMappings []config.DirectoryMapping
			for d := 0; d < tc.dirCount; d++ {
				srcDir := fmt.Sprintf("source_%d", d)
				destDir := fmt.Sprintf("dest_%d", d)

				fullSrcDir := filepath.Join(tempDir, srcDir)
				require.NoError(b, os.MkdirAll(fullSrcDir, 0o750))

				// Create files in each directory
				for f := 0; f < tc.filesPerDir; f++ {
					filename := filepath.Join(fullSrcDir, fmt.Sprintf("file_%d.txt", f))
					content := fmt.Sprintf("content for file %d in directory %d", f, d)
					require.NoError(b, os.WriteFile(filename, []byte(content), 0o600))
				}

				dirMappings = append(dirMappings, config.DirectoryMapping{
					Src:  srcDir,
					Dest: destDir,
				})
			}

			logger := logrus.NewEntry(logrus.New()).WithField("component", "benchmark")
			processor := NewDirectoryProcessor(logger, tc.workerCount, nil)
			defer processor.Close()

			// Force garbage collection before benchmark
			runtime.GC()
			var m1, m2 runtime.MemStats
			runtime.ReadMemStats(&m1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				var wg sync.WaitGroup

				for _, dirMapping := range dirMappings {
					wg.Add(1)
					go func(dm config.DirectoryMapping) {
						defer wg.Done()

						processor.exclusionEngine = NewExclusionEngine(dm.Exclude)
						files, err := processor.discoverFiles(ctx, tempDir, dm)
						if err != nil {
							b.Errorf("Failed to discover files: %v", err)
							return
						}
						if len(files) != tc.filesPerDir {
							b.Errorf("Expected %d files, got %d", tc.filesPerDir, len(files))
						}
					}(dirMapping)
				}

				wg.Wait()
			}
			b.StopTimer()

			runtime.ReadMemStats(&m2)

			// Report memory statistics
			var allocPerOp uint64
			if b.N > 0 {
				bNUint64 := uint64(b.N) //nolint:gosec // G115: Safe conversion in benchmark test
				if bNUint64 > 0 {
					allocPerOp = (m2.TotalAlloc - m1.TotalAlloc) / bNUint64
				}
				b.ReportMetric(float64(allocPerOp), "bytes-alloc-per-op")
				b.ReportMetric(float64(m2.Mallocs-m1.Mallocs)/float64(b.N), "mallocs-per-op")
			}
			peakMemory := m2.Sys

			b.ReportMetric(float64(allocPerOp), "bytes-alloc-per-op")
			b.ReportMetric(float64(peakMemory), "peak-memory-bytes")
			b.ReportMetric(float64(tc.workerCount), "worker-count")
		})
	}
}

// BenchmarkExclusionPatternMemory benchmarks memory efficiency of exclusion pattern matching
func BenchmarkExclusionPatternMemory(b *testing.B) {
	testCases := []struct {
		name         string
		patternCount int
		testPaths    int
	}{
		{"FewPatterns_10patterns_100paths", 10, 100},
		{"ManyPatterns_50patterns_500paths", 50, 500},
		{"ComplexPatterns_100patterns_1000paths", 100, 1000},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Generate exclusion patterns
			var patterns []string
			for i := 0; i < tc.patternCount; i++ {
				patterns = append(
					patterns,
					fmt.Sprintf("**/*.tmp_%d", i),
					fmt.Sprintf("**/temp_%d/**", i),
					fmt.Sprintf("build_%d/**/*.out", i),
				)
			}

			// Generate test paths
			var testPaths []string
			for i := 0; i < tc.testPaths; i++ {
				testPaths = append(
					testPaths,
					fmt.Sprintf("src/module_%d/file_%d.go", i%10, i),
					fmt.Sprintf("build_%d/output_%d.out", i%5, i),
					fmt.Sprintf("temp_%d/cache_%d.tmp_%d", i%3, i, i%7),
				)
			}

			// Force garbage collection before benchmark
			runtime.GC()
			var m1, m2 runtime.MemStats
			runtime.ReadMemStats(&m1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				engine := NewExclusionEngine(patterns)

				matchCount := 0
				for _, path := range testPaths {
					if engine.IsExcluded(path) {
						matchCount++
					}
				}

				// Prevent compiler optimization
				_ = matchCount
			}
			b.StopTimer()

			runtime.ReadMemStats(&m2)

			// Report memory statistics
			var allocPerOp uint64
			if b.N > 0 {
				bNUint64 := uint64(b.N)
				if bNUint64 > 0 {
					allocPerOp = (m2.TotalAlloc - m1.TotalAlloc) / bNUint64
				}
				b.ReportMetric(float64(allocPerOp), "bytes-alloc-per-op")
				b.ReportMetric(float64(m2.Mallocs-m1.Mallocs)/float64(b.N), "mallocs-per-op")
			}
			b.ReportMetric(float64(allocPerOp), "bytes-alloc-per-op")
			b.ReportMetric(float64(tc.patternCount), "pattern-count")
			b.ReportMetric(float64(tc.testPaths), "test-path-count")
		})
	}
}

// BenchmarkRealWorldScenarios benchmarks realistic directory sync scenarios
func BenchmarkRealWorldScenarios(b *testing.B) {
	// Use actual test fixtures for realistic scenarios
	fixturesPath := filepath.Join("..", "..", "test", "fixtures", "directories")

	testCases := []struct {
		name        string
		fixturePath string
		patterns    []string
	}{
		{
			"GitHubWorkflows",
			filepath.Join(fixturesPath, "github"),
			[]string{"*.log", "*.tmp", ".DS_Store"},
		},
		{
			"ComplexStructure",
			filepath.Join(fixturesPath, "complex"),
			[]string{"*.log", "*.tmp", "**/*.bak", "**/Thumbs.db", "**/desktop.ini"},
		},
		{
			"LargeRepository",
			filepath.Join(fixturesPath, "large"),
			[]string{"**/*.tmp", "**/*.log", "**/node_modules/**", "**/.git/**"},
		},
		{
			"MixedContent",
			filepath.Join(fixturesPath, "mixed"),
			[]string{"*.tmp", "**/*.exe", "**/*.dll", "Thumbs.db", "desktop.ini"},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Check if fixture path exists
			if _, err := os.Stat(tc.fixturePath); os.IsNotExist(err) {
				b.Skipf("Fixture path %s does not exist", tc.fixturePath)
				return
			}

			logger := logrus.NewEntry(logrus.New()).WithField("component", "benchmark")
			processor := &DirectoryProcessor{
				logger:          logger,
				progressManager: NewDirectoryProgressManager(logger),
			}

			dirMapping := config.DirectoryMapping{
				Src:     tc.fixturePath,
				Dest:    "dest",
				Exclude: tc.patterns,
			}

			// Count total files for validation
			var expectedFiles int
			err := filepath.WalkDir(tc.fixturePath, func(_ string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !d.IsDir() {
					expectedFiles++
				}
				return nil
			})
			require.NoError(b, err)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				processor.exclusionEngine = NewExclusionEngine(dirMapping.Exclude)

				// Use parent directory of fixture path
				parentDir := filepath.Dir(tc.fixturePath)
				relativeMapping := config.DirectoryMapping{
					Src:     filepath.Base(tc.fixturePath),
					Dest:    dirMapping.Dest,
					Exclude: dirMapping.Exclude,
				}

				files, err := processor.discoverFiles(ctx, parentDir, relativeMapping)
				require.NoError(b, err)

				// Report discovered file count
				b.ReportMetric(float64(len(files)), "files-discovered")
			}
		})
	}
}

// BenchmarkPerformanceRegression establishes baseline metrics for performance regression detection
func BenchmarkPerformanceRegression(b *testing.B) {
	// Baseline scenario representing typical usage
	baselineScenario := struct {
		fileCount    int
		dirDepth     int
		workerCount  int
		patternCount int
	}{
		fileCount:    500,
		dirDepth:     5,
		workerCount:  10,
		patternCount: 20,
	}

	b.Run("BaselineDirectoryProcessing", func(b *testing.B) {
		// Create test directory structure
		tempDir := b.TempDir()
		sourceDir := filepath.Join(tempDir, "source")
		require.NoError(b, os.MkdirAll(sourceDir, 0o750))

		// Create realistic directory structure
		for i := 0; i < baselineScenario.fileCount; i++ {
			// Create nested directories
			depth := i % baselineScenario.dirDepth
			dirPath := sourceDir
			for d := 0; d < depth; d++ {
				dirPath = filepath.Join(dirPath, fmt.Sprintf("level_%d", d))
			}
			require.NoError(b, os.MkdirAll(dirPath, 0o750))

			// Create file with realistic content
			filename := filepath.Join(dirPath, fmt.Sprintf("file_%d.go", i))
			content := fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"log"
)

// Function%d demonstrates functionality
func Function%d(ctx context.Context) error {
	fmt.Printf("Processing item %%d\\n", %d)
	return nil
}

func main() {
	if err := Function%d(context.Background()); err != nil {
		log.Fatal(err)
	}
}
`, i, i, i, i)
			require.NoError(b, os.WriteFile(filename, []byte(content), 0o600))
		}

		// Create realistic exclusion patterns
		patterns := make([]string, baselineScenario.patternCount)
		for i := 0; i < baselineScenario.patternCount; i++ {
			patterns[i] = fmt.Sprintf("**/*.tmp_%d", i)
		}

		logger := logrus.NewEntry(logrus.New()).WithField("component", "benchmark")
		processor := NewDirectoryProcessor(logger, baselineScenario.workerCount, nil)
		defer processor.Close()

		dirMapping := config.DirectoryMapping{
			Src:     "source",
			Dest:    "dest",
			Exclude: patterns,
		}

		// Measure baseline performance
		b.ResetTimer()
		var totalFiles int
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			processor.exclusionEngine = NewExclusionEngine(dirMapping.Exclude)
			files, err := processor.discoverFiles(ctx, tempDir, dirMapping)
			require.NoError(b, err)
			totalFiles = len(files)
		}

		// Report baseline metrics
		b.ReportMetric(float64(totalFiles), "baseline-files-processed")
		b.ReportMetric(float64(baselineScenario.fileCount), "baseline-total-files")
		b.ReportMetric(float64(baselineScenario.workerCount), "baseline-worker-count")

		// Calculate processing efficiency
		efficiency := float64(totalFiles) / float64(baselineScenario.fileCount) * 100
		b.ReportMetric(efficiency, "baseline-processing-efficiency-%")
	})

	b.Run("BaselineCacheEffectiveness", func(b *testing.B) {
		logger := logrus.NewEntry(logrus.New()).WithField("component", "benchmark")
		cache := NewContentCache(5*time.Minute, 100*1024*1024, logger)

		// Simulate realistic access pattern
		files := make([]string, 200)
		for i := range files {
			files[i] = fmt.Sprintf("src/module_%d/file_%d.go", i%10, i)
		}

		b.ResetTimer()
		var hits, misses int64

		for i := 0; i < b.N; i++ {
			// Simulate 80/20 access pattern (80% of accesses to 20% of files)
			for j := 0; j < 1000; j++ {
				var fileIndex int
				if j%5 < 4 { // 80% of the time
					fileIndex = j % 40 // Access first 20% of files
				} else { // 20% of the time
					fileIndex = 40 + (j % 160) // Access remaining 80% of files
				}

				fileName := files[fileIndex]
				ctx := context.Background()
				_, hit, err := cache.Get(ctx, "owner/repo", "master", fileName)
				if err != nil {
					b.Fatalf("Cache get failed: %v", err)
				}

				if hit {
					atomic.AddInt64(&hits, 1)
				} else {
					atomic.AddInt64(&misses, 1)
					err := cache.Put(ctx, "owner/repo", "master", fileName, fmt.Sprintf("content-%d", fileIndex))
					if err != nil {
						b.Fatalf("Cache put failed: %v", err)
					}
				}
			}
		}

		stats := cache.GetStats()
		overallHitRate := float64(hits) / float64(hits+misses) * 100

		b.ReportMetric(overallHitRate, "baseline-cache-hit-rate-%")
		b.ReportMetric(float64(stats.Size), "baseline-cache-size")
		b.ReportMetric(float64(stats.MemoryUsage), "baseline-memory-usage-bytes")
	})
}
