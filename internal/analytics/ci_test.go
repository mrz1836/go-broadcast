package analytics

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// ============================================================
// Artifact Parsing Unit Tests
// ============================================================

func TestParseLOCJSON(t *testing.T) {
	t.Run("valid LOC JSON", func(t *testing.T) {
		data := []byte(`{
			"go_files_loc": 5678,
			"test_files_loc": 1234,
			"go_files_count": 90,
			"test_files_count": 56,
			"total_loc": 6912,
			"total_files_count": 146
		}`)

		loc := parseLOCJSON(data)
		require.NotNil(t, loc)
		assert.Equal(t, 5678, loc.GoFilesLOC)
		assert.Equal(t, 1234, loc.TestFilesLOC)
		assert.Equal(t, 90, loc.GoFilesCount)
		assert.Equal(t, 56, loc.TestFilesCount)
	})

	t.Run("all zeros returns nil", func(t *testing.T) {
		data := []byte(`{
			"go_files_loc": 0,
			"test_files_loc": 0,
			"go_files_count": 0,
			"test_files_count": 0
		}`)

		loc := parseLOCJSON(data)
		assert.Nil(t, loc)
	})

	t.Run("invalid JSON returns nil", func(t *testing.T) {
		loc := parseLOCJSON([]byte(`not json`))
		assert.Nil(t, loc)
	})

	t.Run("empty JSON object returns nil", func(t *testing.T) {
		loc := parseLOCJSON([]byte(`{}`))
		assert.Nil(t, loc)
	})

	t.Run("partial data is accepted", func(t *testing.T) {
		data := []byte(`{"go_files_loc": 100}`)
		loc := parseLOCJSON(data)
		require.NotNil(t, loc)
		assert.Equal(t, 100, loc.GoFilesLOC)
		assert.Equal(t, 0, loc.TestFilesLOC)
	})
}

func TestParseLOCFromMarkdown(t *testing.T) {
	t.Run("standard GoFortress table format", func(t *testing.T) {
		md := []byte("### Lines of Code Summary\n" +
			"| Type | Lines of Code | Files | Total Size | Avg Size | Date |\n" +
			"|------|---------------|-------|------------|----------|------|\n" +
			"| Test Files | 1,234 | 56 | 120KB | 2.1KB | 2024-01-15 |\n" +
			"| Go Files | 5,678 | 90 | 450KB | 5.0KB | 2024-01-15 |\n" +
			"| **Total** | **6,912** | **146** | **570KB** | | |\n")

		goLOC, testLOC, goFiles, testFiles := parseLOCFromMarkdown(md)
		assert.Equal(t, 5678, goLOC)
		assert.Equal(t, 1234, testLOC)
		assert.Equal(t, 90, goFiles)
		assert.Equal(t, 56, testFiles)
	})

	t.Run("no matching rows", func(t *testing.T) {
		md := []byte("### Some other section\n| Column A | Column B |\n|----------|----------|\n| data1    | data2    |\n")

		goLOC, testLOC, goFiles, testFiles := parseLOCFromMarkdown(md)
		assert.Equal(t, 0, goLOC)
		assert.Equal(t, 0, testLOC)
		assert.Equal(t, 0, goFiles)
		assert.Equal(t, 0, testFiles)
	})

	t.Run("empty input", func(t *testing.T) {
		goLOC, testLOC, goFiles, testFiles := parseLOCFromMarkdown([]byte{})
		assert.Equal(t, 0, goLOC)
		assert.Equal(t, 0, testLOC)
		assert.Equal(t, 0, goFiles)
		assert.Equal(t, 0, testFiles)
	})

	t.Run("numbers without commas", func(t *testing.T) {
		md := []byte("| Type | Lines of Code | Files |\n" +
			"|------|---------------|-------|\n" +
			"| Test Files | 500 | 10 |\n" +
			"| Go Files | 2000 | 30 |\n")

		goLOC, testLOC, goFiles, testFiles := parseLOCFromMarkdown(md)
		assert.Equal(t, 2000, goLOC)
		assert.Equal(t, 500, testLOC)
		assert.Equal(t, 30, goFiles)
		assert.Equal(t, 10, testFiles)
	})
}

func TestParseCoverageJSON(t *testing.T) {
	t.Run("coverage_percentage field", func(t *testing.T) {
		data := []byte(`{"coverage_percentage": 85.5, "provider": "codecov"}`)
		cov := parseCoverageJSON(data)
		require.NotNil(t, cov)
		assert.InDelta(t, 85.5, *cov, 0.01)
	})

	t.Run("coverage_percent field", func(t *testing.T) {
		data := []byte(`{"coverage_percent": 92.3}`)
		cov := parseCoverageJSON(data)
		require.NotNil(t, cov)
		assert.InDelta(t, 92.3, *cov, 0.01)
	})

	t.Run("string coverage value", func(t *testing.T) {
		data := []byte(`{"coverage_percentage": "78.9"}`)
		cov := parseCoverageJSON(data)
		require.NotNil(t, cov)
		assert.InDelta(t, 78.9, *cov, 0.01)
	})

	t.Run("no coverage field returns nil", func(t *testing.T) {
		data := []byte(`{"provider": "codecov", "files_processed": 50}`)
		cov := parseCoverageJSON(data)
		assert.Nil(t, cov)
	})

	t.Run("invalid JSON returns nil", func(t *testing.T) {
		cov := parseCoverageJSON([]byte(`not json`))
		assert.Nil(t, cov)
	})

	t.Run("N/A string value returns nil", func(t *testing.T) {
		data := []byte(`{"coverage_percentage": "N/A"}`)
		cov := parseCoverageJSON(data)
		assert.Nil(t, cov)
	})

	t.Run("null value returns nil", func(t *testing.T) {
		data := []byte(`{"coverage_percentage": null}`)
		cov := parseCoverageJSON(data)
		assert.Nil(t, cov)
	})
}

func TestParseBenchStatsJSON(t *testing.T) {
	t.Run("valid benchmark stats", func(t *testing.T) {
		data := []byte(`{
			"name": "unit-benchmarks",
			"benchmark_count": 25,
			"duration_seconds": 120,
			"status": "success"
		}`)

		count := parseBenchStatsJSON(data)
		assert.Equal(t, 25, count)
	})

	t.Run("zero benchmarks", func(t *testing.T) {
		data := []byte(`{"benchmark_count": 0}`)
		count := parseBenchStatsJSON(data)
		assert.Equal(t, 0, count)
	})

	t.Run("missing benchmark_count", func(t *testing.T) {
		data := []byte(`{"name": "bench", "status": "success"}`)
		count := parseBenchStatsJSON(data)
		assert.Equal(t, 0, count)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		count := parseBenchStatsJSON([]byte(`broken`))
		assert.Equal(t, 0, count)
	})
}

func TestParseTestCountFromMarkdown(t *testing.T) {
	t.Run("tests keyword", func(t *testing.T) {
		md := []byte("### Test Results\n" +
			"| Suite | Count | Status |\n" +
			"|-------|-------|--------|\n" +
			"| Unit  | 150 tests | pass |\n")

		count := parseTestCountFromMarkdown(md)
		assert.Equal(t, 150, count)
	})

	t.Run("total keyword", func(t *testing.T) {
		md := []byte("200 total tests run\n50 passed\n1 failed")
		count := parseTestCountFromMarkdown(md)
		assert.Equal(t, 200, count)
	})

	t.Run("formatted number with commas", func(t *testing.T) {
		md := []byte("1,234 tests passed")
		count := parseTestCountFromMarkdown(md)
		assert.Equal(t, 1234, count)
	})

	t.Run("no test count found", func(t *testing.T) {
		md := []byte("No relevant content here")
		count := parseTestCountFromMarkdown(md)
		assert.Equal(t, 0, count)
	})

	t.Run("empty input", func(t *testing.T) {
		count := parseTestCountFromMarkdown([]byte{})
		assert.Equal(t, 0, count)
	})

	t.Run("picks largest count", func(t *testing.T) {
		md := []byte("50 tests in suite A\n150 tests in suite B\n")
		count := parseTestCountFromMarkdown(md)
		assert.Equal(t, 150, count)
	})
}

func TestParseFormattedInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"plain number", "1234", 1234},
		{"with commas", "1,234", 1234},
		{"large number", "1,234,567", 1234567},
		{"zero", "0", 0},
		{"bold markdown", "**1,234**", 1234},
		{"with spaces", " 42 ", 42},
		{"empty", "", 0},
		{"non-numeric", "abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseFormattedInt(tt.input))
		})
	}
}

// ============================================================
// File Helper Tests
// ============================================================

func TestFindAndReadJSON(t *testing.T) {
	t.Run("finds JSON file", func(t *testing.T) {
		dir := t.TempDir()
		expected := []byte(`{"key": "value"}`)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "data.json"), expected, 0o600))

		data, err := findAndReadJSON(dir)
		require.NoError(t, err)
		assert.Equal(t, expected, data)
	})

	t.Run("no JSON file", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "data.txt"), []byte("text"), 0o600))

		_, err := findAndReadJSON(dir)
		require.Error(t, err)
		assert.ErrorIs(t, err, errFileNotFound)
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := findAndReadJSON("/nonexistent/path")
		require.Error(t, err)
	})
}

func TestFindAndReadMarkdown(t *testing.T) {
	t.Run("finds markdown file", func(t *testing.T) {
		dir := t.TempDir()
		expected := []byte("# Title\nContent")
		require.NoError(t, os.WriteFile(filepath.Join(dir, "report.md"), expected, 0o600))

		data, err := findAndReadMarkdown(dir)
		require.NoError(t, err)
		assert.Equal(t, expected, data)
	})

	t.Run("no markdown file", func(t *testing.T) {
		dir := t.TempDir()
		_, err := findAndReadMarkdown(dir)
		require.Error(t, err)
	})
}

// ============================================================
// CICollector Integration Tests (with mocks)
// ============================================================

func TestCICollector_CollectCIMetrics(t *testing.T) {
	ctx := context.Background()

	t.Run("empty repos returns empty map", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		collector := NewCICollector(mockClient, logrus.New())

		result, err := collector.CollectCIMetrics(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("repo without GoFortress is skipped", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		collector := NewCICollector(mockClient, logrus.New())

		repos := []gh.RepoInfo{
			{FullName: "owner/repo1"},
		}

		mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
			Return([]gh.Workflow{
				{ID: 1, Name: "CI", Path: ".github/workflows/ci.yml"},
			}, nil)

		result, err := collector.CollectCIMetrics(ctx, repos)
		require.NoError(t, err)
		assert.Empty(t, result)

		mockClient.AssertExpectations(t)
	})

	t.Run("repo with GoFortress but no runs is skipped", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		collector := NewCICollector(mockClient, logrus.New())

		repos := []gh.RepoInfo{
			{FullName: "owner/repo1"},
		}

		mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
			Return([]gh.Workflow{
				{ID: 10, Name: "GoFortress", Path: ".github/workflows/fortress.yml"},
			}, nil)

		mockClient.On("GetWorkflowRuns", mock.Anything, "owner/repo1", int64(10), 1).
			Return([]gh.WorkflowRun{}, nil)

		result, err := collector.CollectCIMetrics(ctx, repos)
		require.NoError(t, err)
		assert.Empty(t, result)

		mockClient.AssertExpectations(t)
	})

	t.Run("workflow list error is handled gracefully", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		collector := NewCICollector(mockClient, logrus.New())

		repos := []gh.RepoInfo{
			{FullName: "owner/repo1"},
		}

		mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
			Return([]gh.Workflow(nil), assert.AnError)

		result, err := collector.CollectCIMetrics(ctx, repos)
		require.NoError(t, err) // Should not fail the batch
		assert.Empty(t, result)

		mockClient.AssertExpectations(t)
	})

	t.Run("successful collection with loc-stats and coverage artifacts", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		collector := NewCICollector(mockClient, logrus.New())

		repos := []gh.RepoInfo{
			{FullName: "owner/repo1"},
		}

		mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
			Return([]gh.Workflow{
				{ID: 10, Name: "GoFortress", Path: ".github/workflows/fortress.yml"},
			}, nil)

		mockClient.On("GetWorkflowRuns", mock.Anything, "owner/repo1", int64(10), 1).
			Return([]gh.WorkflowRun{
				{ID: 100, HeadBranch: "main", HeadSHA: "abc123", Status: "completed", Conclusion: "success"},
			}, nil)

		mockClient.On("GetRunArtifacts", mock.Anything, "owner/repo1", int64(100)).
			Return([]gh.Artifact{
				{ID: 200, Name: "loc-stats"},
				{ID: 201, Name: "coverage-stats-codecov"},
			}, nil)

		// Mock artifact downloads — write test data into the destination directory
		mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), "loc-stats", mock.AnythingOfType("string")).
			Run(func(args mock.Arguments) {
				destDir := args.String(4)
				_ = os.MkdirAll(destDir, 0o750)
				locJSON := `{"go_files_loc": 5000, "test_files_loc": 2000, "go_files_count": 80, "test_files_count": 40}`
				_ = os.WriteFile(filepath.Join(destDir, "loc-stats.json"), []byte(locJSON), 0o600)
			}).
			Return(nil)

		mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), "coverage-stats-codecov", mock.AnythingOfType("string")).
			Run(func(args mock.Arguments) {
				destDir := args.String(4)
				_ = os.MkdirAll(destDir, 0o750)
				covJSON := `{"coverage_percentage": 87.5, "provider": "codecov"}`
				_ = os.WriteFile(filepath.Join(destDir, "coverage-stats-codecov.json"), []byte(covJSON), 0o600)
			}).
			Return(nil)

		result, err := collector.CollectCIMetrics(ctx, repos)
		require.NoError(t, err)
		require.Len(t, result, 1)

		metrics := result["owner/repo1"]
		require.NotNil(t, metrics)
		assert.Equal(t, int64(100), metrics.WorkflowRunID)
		assert.Equal(t, "main", metrics.Branch)
		assert.Equal(t, "abc123", metrics.CommitSHA)
		assert.Equal(t, 5000, metrics.GoFilesLOC)
		assert.Equal(t, 2000, metrics.TestFilesLOC)
		assert.Equal(t, 80, metrics.GoFilesCount)
		assert.Equal(t, 40, metrics.TestFilesCount)
		require.NotNil(t, metrics.Coverage)
		assert.InDelta(t, 87.5, *metrics.Coverage, 0.01)

		mockClient.AssertExpectations(t)
	})

	t.Run("successful collection with bench-stats artifact", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		collector := NewCICollector(mockClient, logrus.New())

		repos := []gh.RepoInfo{
			{FullName: "owner/repo1"},
		}

		mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
			Return([]gh.Workflow{
				{ID: 10, Name: "GoFortress"},
			}, nil)

		mockClient.On("GetWorkflowRuns", mock.Anything, "owner/repo1", int64(10), 1).
			Return([]gh.WorkflowRun{
				{ID: 100, HeadBranch: "main", HeadSHA: "def456"},
			}, nil)

		mockClient.On("GetRunArtifacts", mock.Anything, "owner/repo1", int64(100)).
			Return([]gh.Artifact{
				{ID: 300, Name: "bench-stats-unit"},
			}, nil)

		mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), "bench-stats-unit", mock.AnythingOfType("string")).
			Run(func(args mock.Arguments) {
				destDir := args.String(4)
				_ = os.MkdirAll(destDir, 0o750)
				benchJSON := `{"name": "unit-benchmarks", "benchmark_count": 42, "status": "success"}`
				_ = os.WriteFile(filepath.Join(destDir, "bench-stats-unit.json"), []byte(benchJSON), 0o600)
			}).
			Return(nil)

		result, err := collector.CollectCIMetrics(ctx, repos)
		require.NoError(t, err)
		require.Len(t, result, 1)

		metrics := result["owner/repo1"]
		require.NotNil(t, metrics)
		assert.Equal(t, 42, metrics.BenchmarkCount)

		mockClient.AssertExpectations(t)
	})

	t.Run("successful collection with markdown fallback for LOC", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		collector := NewCICollector(mockClient, logrus.New())

		repos := []gh.RepoInfo{
			{FullName: "owner/repo1"},
		}

		mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
			Return([]gh.Workflow{
				{ID: 10, Name: "GoFortress"},
			}, nil)

		mockClient.On("GetWorkflowRuns", mock.Anything, "owner/repo1", int64(10), 1).
			Return([]gh.WorkflowRun{
				{ID: 100, HeadBranch: "main", HeadSHA: "xyz789"},
			}, nil)

		// No loc-stats artifact, but statistics-section is present
		mockClient.On("GetRunArtifacts", mock.Anything, "owner/repo1", int64(100)).
			Return([]gh.Artifact{
				{ID: 400, Name: "statistics-section"},
			}, nil)

		mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), "statistics-section", mock.AnythingOfType("string")).
			Run(func(args mock.Arguments) {
				destDir := args.String(4)
				_ = os.MkdirAll(destDir, 0o750)
				md := "### Lines of Code Summary\n" +
					"| Type | Lines of Code | Files |\n" +
					"|------|---------------|-------|\n" +
					"| Test Files | 1,000 | 20 |\n" +
					"| Go Files | 3,000 | 50 |\n"
				_ = os.WriteFile(filepath.Join(destDir, "statistics-section.md"), []byte(md), 0o600)
			}).
			Return(nil)

		result, err := collector.CollectCIMetrics(ctx, repos)
		require.NoError(t, err)
		require.Len(t, result, 1)

		metrics := result["owner/repo1"]
		require.NotNil(t, metrics)
		assert.Equal(t, 3000, metrics.GoFilesLOC)
		assert.Equal(t, 1000, metrics.TestFilesLOC)
		assert.Equal(t, 50, metrics.GoFilesCount)
		assert.Equal(t, 20, metrics.TestFilesCount)

		mockClient.AssertExpectations(t)
	})

	t.Run("artifact download failure is handled gracefully", func(t *testing.T) {
		mockClient := gh.NewMockClient()
		collector := NewCICollector(mockClient, logrus.New())

		repos := []gh.RepoInfo{
			{FullName: "owner/repo1"},
		}

		mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
			Return([]gh.Workflow{
				{ID: 10, Name: "GoFortress"},
			}, nil)

		mockClient.On("GetWorkflowRuns", mock.Anything, "owner/repo1", int64(10), 1).
			Return([]gh.WorkflowRun{
				{ID: 100, HeadBranch: "main", HeadSHA: "abc"},
			}, nil)

		mockClient.On("GetRunArtifacts", mock.Anything, "owner/repo1", int64(100)).
			Return([]gh.Artifact{
				{ID: 200, Name: "loc-stats"},
			}, nil)

		// Download fails
		mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), "loc-stats", mock.AnythingOfType("string")).
			Return(assert.AnError)

		result, err := collector.CollectCIMetrics(ctx, repos)
		require.NoError(t, err)
		// Metrics still created, just without LOC data
		require.Len(t, result, 1)
		metrics := result["owner/repo1"]
		require.NotNil(t, metrics)
		assert.Equal(t, 0, metrics.GoFilesLOC)

		mockClient.AssertExpectations(t)
	})
}

func TestCICollector_CollectCIMetrics_NilLogger(t *testing.T) {
	ctx := context.Background()
	mockClient := gh.NewMockClient()
	collector := NewCICollector(mockClient, nil) // nil logger

	repos := []gh.RepoInfo{
		{FullName: "owner/repo1"},
	}

	mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
		Return([]gh.Workflow(nil), assert.AnError)

	result, err := collector.CollectCIMetrics(ctx, repos)
	require.NoError(t, err)
	assert.Empty(t, result)

	mockClient.AssertExpectations(t)
}
