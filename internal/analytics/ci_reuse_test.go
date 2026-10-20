package analytics

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// reusableBaseline returns metrics collected in full from run 100, attempt 1.
func reusableBaseline() *CIMetrics {
	coverage := 91.2
	return &CIMetrics{
		WorkflowRunID:      100,
		WorkflowRunAttempt: 1,
		Branch:             "main",
		CommitSHA:          "abc123",
		GoFilesLOC:         1500,
		TestFilesLOC:       5500,
		GoFilesCount:       20,
		TestFilesCount:     50,
		TestCount:          321,
		BenchmarkCount:     12,
		Coverage:           &coverage,
		ArtifactsComplete:  true,
	}
}

// writeArtifact simulates gh run download by writing one file into the artifact directory.
func writeArtifact(fileName, content string) func(mock.Arguments) {
	return func(args mock.Arguments) {
		destDir := args.String(4)
		_ = os.MkdirAll(destDir, 0o750)
		_ = os.WriteFile(filepath.Join(destDir, fileName), []byte(content), 0o600)
	}
}

func TestReusableCIMetrics(t *testing.T) {
	newest := gh.WorkflowRun{ID: 100, RunAttempt: 1}

	tests := []struct {
		name     string
		baseline func() *CIMetrics
		newest   gh.WorkflowRun
		reusable bool
	}{
		{name: "same run and attempt", baseline: reusableBaseline, newest: newest, reusable: true},
		{name: "no baseline", baseline: func() *CIMetrics { return nil }, newest: newest},
		{name: "newer run", baseline: reusableBaseline, newest: gh.WorkflowRun{ID: 101, RunAttempt: 1}},
		{name: "re-run attempt", baseline: reusableBaseline, newest: gh.WorkflowRun{ID: 100, RunAttempt: 2}},
		{
			name: "no coverage",
			baseline: func() *CIMetrics {
				m := reusableBaseline()
				m.Coverage = nil
				return m
			},
			newest: newest,
		},
		{
			name: "incomplete artifacts",
			baseline: func() *CIMetrics {
				m := reusableBaseline()
				m.ArtifactsComplete = false
				return m
			},
			newest: newest,
		},
		{
			name: "unknown attempt",
			baseline: func() *CIMetrics {
				m := reusableBaseline()
				m.WorkflowRunAttempt = 0
				return m
			},
			newest: gh.WorkflowRun{ID: 100, RunAttempt: 0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			baseline := tc.baseline()
			reused := reusableCIMetrics(baseline, tc.newest)

			if !tc.reusable {
				assert.Nil(t, reused)
				return
			}

			require.NotNil(t, reused)
			assert.True(t, reused.Reused)
			assert.False(t, baseline.Reused, "the baseline itself must not be modified")

			expected := *baseline
			expected.Reused = true
			assert.Equal(t, expected, *reused)
			assert.NotSame(t, baseline.Coverage, reused.Coverage, "coverage must be copied, not shared")
		})
	}
}

func TestCICollectorReusesBaselineForUnchangedRun(t *testing.T) {
	ctx := context.Background()
	mockClient := gh.NewMockClient()
	mockClient.Test(t) // Any artifact request fails the test
	mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
		Return([]gh.Workflow{{ID: 10, Name: GoFortressWorkflowName}}, nil).Once()
	mockClient.On("GetWorkflowRuns", mock.Anything, "owner/repo1", int64(10), maxWorkflowRunsToCheck).
		Return([]gh.WorkflowRun{{ID: 100, RunAttempt: 1, HeadBranch: "main", HeadSHA: "abc123"}}, nil).Once()

	collector := NewCICollector(mockClient, logrus.New(), nil)
	baseline := reusableBaseline()

	result, err := collector.CollectCIMetricsWithBaselines(ctx,
		[]gh.RepoInfo{{FullName: "owner/repo1"}},
		map[string]*CIMetrics{"owner/repo1": baseline})

	require.NoError(t, err)
	metrics := result["owner/repo1"]
	require.NotNil(t, metrics)
	assert.True(t, metrics.Reused)
	assert.Equal(t, 1500, metrics.GoFilesLOC)
	assert.Equal(t, 321, metrics.TestCount)
	assert.Equal(t, 12, metrics.BenchmarkCount)
	require.NotNil(t, metrics.Coverage)
	assert.InDelta(t, 91.2, *metrics.Coverage, 0.0001)
	mockClient.AssertExpectations(t)
}

func TestCICollectorTracksArtifactCompleteness(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T, secondBenchErr error) *gh.MockClient {
		t.Helper()
		mockClient := gh.NewMockClient()
		mockClient.Test(t)
		mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
			Return([]gh.Workflow{{ID: 10, Name: GoFortressWorkflowName}}, nil).Once()
		mockClient.On("GetWorkflowRuns", mock.Anything, "owner/repo1", int64(10), maxWorkflowRunsToCheck).
			Return([]gh.WorkflowRun{{ID: 100, RunAttempt: 2, HeadBranch: "main", HeadSHA: "abc123"}}, nil).Once()
		mockClient.On("GetRunArtifacts", mock.Anything, "owner/repo1", int64(100)).Return([]gh.Artifact{
			{ID: 1, Name: artifactLOCStats},
			{ID: 2, Name: artifactCoverageInternal},
			{ID: 3, Name: "bench-stats-unit"},
			{ID: 4, Name: "bench-stats-race"},
		}, nil).Once()

		mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), artifactLOCStats, mock.AnythingOfType("string")).
			Run(writeArtifact("loc-stats.json",
				`{"go_files_loc": 1500, "test_files_loc": 5500, "go_files_count": 20, "test_files_count": 50}`)).
			Return(nil).Once()
		mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), artifactCoverageInternal, mock.AnythingOfType("string")).
			Run(writeArtifact("coverage.json", `{"coverage_percentage": 91.2}`)).
			Return(nil).Once()
		mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), "bench-stats-unit", mock.AnythingOfType("string")).
			Run(writeArtifact("bench.json", `{"benchmark_count": 7}`)).
			Return(nil).Once()

		race := mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), "bench-stats-race", mock.AnythingOfType("string"))
		if secondBenchErr != nil {
			race.Return(secondBenchErr).Once()
		} else {
			race.Run(writeArtifact("bench.json", `{"benchmark_count": 5}`)).Return(nil).Once()
		}
		return mockClient
	}

	t.Run("every artifact downloaded", func(t *testing.T) {
		mockClient := setup(t, nil)
		result, err := NewCICollector(mockClient, logrus.New(), nil).CollectCIMetrics(ctx, []gh.RepoInfo{{FullName: "owner/repo1"}})

		require.NoError(t, err)
		metrics := result["owner/repo1"]
		require.NotNil(t, metrics)
		assert.True(t, metrics.ArtifactsComplete)
		assert.False(t, metrics.Reused)
		assert.Equal(t, 2, metrics.WorkflowRunAttempt)
		assert.Equal(t, 12, metrics.BenchmarkCount)
		mockClient.AssertExpectations(t)
	})

	t.Run("a failed download marks the metrics incomplete", func(t *testing.T) {
		mockClient := setup(t, assert.AnError)
		result, err := NewCICollector(mockClient, logrus.New(), nil).CollectCIMetrics(ctx, []gh.RepoInfo{{FullName: "owner/repo1"}})

		require.NoError(t, err)
		metrics := result["owner/repo1"]
		require.NotNil(t, metrics)
		assert.False(t, metrics.ArtifactsComplete, "a missing benchmark artifact must block reuse of these metrics")
		assert.Equal(t, 7, metrics.BenchmarkCount)
		mockClient.AssertExpectations(t)
	})
}

// TestCICollectorPicksCIResultsDeterministically verifies a run with several
// ci-results artifacts always reports the same test count, so re-collecting a run
// can never change its metrics.
func TestCICollectorPicksCIResultsDeterministically(t *testing.T) {
	ctx := context.Background()
	writeResults := func(total int) func(mock.Arguments) {
		return func(args mock.Arguments) {
			destDir := filepath.Join(args.String(4), ".mage-x")
			_ = os.MkdirAll(destDir, 0o750)
			line := `{"type":"summary","summary":{"unique_total":` + strconv.Itoa(total) + `}}`
			_ = os.WriteFile(filepath.Join(destDir, "ci-results.jsonl"), []byte(line+"\n"), 0o600)
		}
	}

	for range 5 {
		mockClient := gh.NewMockClient()
		mockClient.Test(t) // Downloading the second ci-results artifact fails the test
		mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
			Return([]gh.Workflow{{ID: 10, Name: GoFortressWorkflowName}}, nil).Once()
		mockClient.On("GetWorkflowRuns", mock.Anything, "owner/repo1", int64(10), maxWorkflowRunsToCheck).
			Return([]gh.WorkflowRun{{ID: 100, RunAttempt: 1}}, nil).Once()
		mockClient.On("GetRunArtifacts", mock.Anything, "owner/repo1", int64(100)).Return([]gh.Artifact{
			{ID: 1, Name: "ci-results-ubuntu-1.26.x"},
			{ID: 2, Name: "ci-results-macos-1.26.x"},
		}, nil).Once()
		mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), "ci-results-macos-1.26.x", mock.AnythingOfType("string")).
			Run(writeResults(200)).Return(nil).Once()

		result, err := NewCICollector(mockClient, logrus.New(), nil).CollectCIMetrics(ctx, []gh.RepoInfo{{FullName: "owner/repo1"}})

		require.NoError(t, err)
		require.NotNil(t, result["owner/repo1"])
		assert.Equal(t, 200, result["owner/repo1"].TestCount)
		mockClient.AssertExpectations(t)
	}
}

// TestCICollectorMarksUnreadArtifactsIncomplete verifies that every way an attempted
// artifact can go unread marks the metrics incomplete, so they are never reused.
func TestCICollectorMarksUnreadArtifactsIncomplete(t *testing.T) {
	ctx := context.Background()

	// Each case pairs the artifact under test with a healthy one, so the run still
	// yields usable metrics and the completeness flag can be observed.
	healthyLOC := gh.Artifact{ID: 1, Name: artifactLOCStats}
	healthyBench := gh.Artifact{ID: 2, Name: "bench-stats-healthy"}
	writeHealthy := map[string]func(mock.Arguments){
		artifactLOCStats:      writeArtifact("loc-stats.json", `{"go_files_loc": 100, "test_files_loc": 50, "go_files_count": 5, "test_files_count": 3}`),
		"bench-stats-healthy": writeArtifact("bench.json", `{"benchmark_count": 3}`),
	}

	tests := []struct {
		name     string
		artifact string
		healthy  gh.Artifact
		download error // nil means the download succeeds but writes nothing readable
	}{
		{name: "loc-stats unreadable", artifact: artifactLOCStats, healthy: healthyBench},
		{name: "statistics-section not downloaded", artifact: artifactStatistics, healthy: healthyBench, download: assert.AnError},
		{name: "statistics-section unreadable", artifact: artifactStatistics, healthy: healthyBench},
		{name: "coverage not downloaded", artifact: artifactCoverageInternal, healthy: healthyLOC, download: assert.AnError},
		{name: "coverage unreadable", artifact: artifactCoverageInternal, healthy: healthyLOC},
		{name: "tests-section not downloaded", artifact: artifactTestsSection, healthy: healthyLOC, download: assert.AnError},
		{name: "tests-section unreadable", artifact: artifactTestsSection, healthy: healthyLOC},
		{name: "ci-results not downloaded", artifact: "ci-results-ubuntu", healthy: healthyLOC, download: assert.AnError},
		{name: "ci-results unreadable", artifact: "ci-results-ubuntu", healthy: healthyLOC},
		{name: "bench-stats unreadable", artifact: "bench-stats-broken", healthy: healthyLOC},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := gh.NewMockClient()
			mockClient.Test(t)
			mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
				Return([]gh.Workflow{{ID: 10, Name: GoFortressWorkflowName}}, nil).Once()
			mockClient.On("GetWorkflowRuns", mock.Anything, "owner/repo1", int64(10), maxWorkflowRunsToCheck).
				Return([]gh.WorkflowRun{{ID: 100, RunAttempt: 1}}, nil).Once()
			mockClient.On("GetRunArtifacts", mock.Anything, "owner/repo1", int64(100)).
				Return([]gh.Artifact{{ID: 9, Name: tc.artifact}, tc.healthy}, nil).Once()
			mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), tc.healthy.Name, mock.AnythingOfType("string")).
				Run(writeHealthy[tc.healthy.Name]).Return(nil).Once()

			broken := mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), tc.artifact, mock.AnythingOfType("string"))
			if tc.download != nil {
				broken.Return(tc.download).Once()
			} else {
				broken.Return(nil).Once() // Downloads, but leaves no file to read
			}

			result, err := NewCICollector(mockClient, logrus.New(), nil).CollectCIMetrics(ctx, []gh.RepoInfo{{FullName: "owner/repo1"}})

			require.NoError(t, err)
			metrics := result["owner/repo1"]
			require.NotNil(t, metrics, "the healthy artifact should still produce metrics")
			assert.False(t, metrics.ArtifactsComplete)
			mockClient.AssertExpectations(t)
		})
	}
}

// TestCICollectorReadsTestsSection verifies the tests-section fallback is used when a run
// has no ci-results artifact, and that reading it keeps the metrics complete.
func TestCICollectorReadsTestsSection(t *testing.T) {
	ctx := context.Background()
	mockClient := gh.NewMockClient()
	mockClient.Test(t)
	mockClient.On("ListWorkflows", mock.Anything, "owner/repo1").
		Return([]gh.Workflow{{ID: 10, Name: GoFortressWorkflowName}}, nil).Once()
	mockClient.On("GetWorkflowRuns", mock.Anything, "owner/repo1", int64(10), maxWorkflowRunsToCheck).
		Return([]gh.WorkflowRun{{ID: 100, RunAttempt: 1}}, nil).Once()
	mockClient.On("GetRunArtifacts", mock.Anything, "owner/repo1", int64(100)).
		Return([]gh.Artifact{{ID: 1, Name: artifactTestsSection}}, nil).Once()
	mockClient.On("DownloadRunArtifact", mock.Anything, "owner/repo1", int64(100), artifactTestsSection, mock.AnythingOfType("string")).
		Run(writeArtifact("tests.md", "| Suite | Result |\n| unit | 42 tests |\n")).Return(nil).Once()

	result, err := NewCICollector(mockClient, logrus.New(), nil).CollectCIMetrics(ctx, []gh.RepoInfo{{FullName: "owner/repo1"}})

	require.NoError(t, err)
	metrics := result["owner/repo1"]
	require.NotNil(t, metrics)
	assert.Equal(t, 42, metrics.TestCount)
	assert.True(t, metrics.ArtifactsComplete)
	mockClient.AssertExpectations(t)
}
