package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
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
			require.NoError(b, os.MkdirAll(sourceDir, 0o755))

			// Create test files
			for i := 0; i < tc.fileCount; i++ {
				dir := filepath.Join(sourceDir, "subdir", string('a'+rune(i%26)))
				require.NoError(b, os.MkdirAll(dir, 0o755))

				filename := filepath.Join(dir, fmt.Sprintf("file_%d.txt", i))
				require.NoError(b, os.WriteFile(filename, []byte("test content"), 0o644))
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
