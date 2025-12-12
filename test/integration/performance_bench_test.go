//go:build performance

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
	syncpkg "github.com/mrz1836/go-broadcast/internal/sync"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// BenchmarkPerformanceRegression benchmarks performance regression scenarios
// Run with: go test -tags=performance -bench=BenchmarkPerformanceRegression ./test/integration
func BenchmarkPerformanceRegression(b *testing.B) {
	b.Run("baseline performance comparison", func(b *testing.B) {
		if testing.Short() {
			b.Skip("Skipping performance regression test in short mode")
		}

		ctx := context.Background()
		tmpDir := b.TempDir()

		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		require.NoError(b, os.MkdirAll(sourceDir, 0o750))
		require.NoError(b, os.MkdirAll(targetDir, 0o750))

		// Create test file
		require.NoError(b, os.WriteFile(
			filepath.Join(sourceDir, "test.txt"),
			[]byte("test content"),
			0o600,
		))

		initGitRepoPerf(b, sourceDir)
		initGitRepoPerf(b, targetDir)

		// Simple configuration for baseline
		cfg := &config.Config{
			Version: 1,
			Groups: []config.Group{
				{
					Name:     "Baseline Test",
					ID:       "baseline",
					Priority: 1,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   sourceDir,
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: targetDir,
							Files: []config.FileMapping{
								{
									Src:  "test.txt",
									Dest: "output.txt",
								},
							},
						},
					},
				},
			},
		}

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		// Run multiple times to get average
		var durations []time.Duration
		for i := 0; i < 5; i++ {
			opts := syncpkg.DefaultOptions().
				WithDryRun(true)

			// Setup mocks
			mockGH := &gh.MockClient{}
			mockGit := &git.MockClient{}
			// Add broad GetChangedFiles mock to handle all calls
			mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
			mockState := &state.MockDiscoverer{}
			mockTransform := &transform.MockChain{}

			// Mock state discovery
			currentState := &state.State{
				Source: state.SourceState{
					Repo:         sourceDir,
					Branch:       "main",
					LatestCommit: "abc123",
				},
				Targets: map[string]*state.TargetState{
					targetDir: {
						Repo:           targetDir,
						LastSyncCommit: "old123", // Outdated to trigger sync
						Status:         state.StatusBehind,
					},
				},
			}
			mockState.On("DiscoverState", mock.Anything, cfg).Return(currentState, nil)

			// Mock git operations
			mockGit.On("Clone", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Checkout", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Commit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			// Mock GitHub operations
			mockGH.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte("test content"), nil)
			mockGH.On("CreatePR", mock.Anything, mock.Anything).Return("https://github.com/org/repo/pull/1", nil)

			// Mock transform operations
			mockTransform.On("Apply", mock.Anything, mock.Anything).Return(mock.Anything, nil)

			engine := syncpkg.NewEngine(context.Background(), cfg, mockGH, mockGit, mockState, mockTransform, opts)

			start := time.Now()
			err := engine.Sync(ctx, nil)
			duration := time.Since(start)

			require.NoError(b, err)
			durations = append(durations, duration)
		}

		// Calculate average
		var total time.Duration
		for _, d := range durations {
			total += d
		}
		avg := total / time.Duration(len(durations))

		b.Logf("Average baseline performance: %v", avg)

		// Store as reference (in practice, this would be compared against historical data)
		// For now, just ensure it's reasonable
		// The race detector adds significant overhead, so we need different thresholds
		var threshold time.Duration
		if isRaceEnabled() {
			threshold = 5 * time.Second // Race detector can add 10x+ overhead
			b.Logf("Race detector enabled, using relaxed threshold: %v", threshold)
		} else {
			threshold = 500 * time.Millisecond
			b.Logf("Race detector disabled, using normal threshold: %v", threshold)
		}
		assert.Less(b, avg, threshold, "Baseline performance should be under %v", threshold)
	})

	b.Run("performance with increasing groups", func(b *testing.B) {
		if testing.Short() {
			b.Skip("Skipping performance regression test in short mode")
		}

		tmpDir := b.TempDir()
		sourceDir := filepath.Join(tmpDir, "source")
		targetDir := filepath.Join(tmpDir, "target")

		require.NoError(b, os.MkdirAll(sourceDir, 0o750))
		require.NoError(b, os.MkdirAll(targetDir, 0o750))

		require.NoError(b, os.WriteFile(
			filepath.Join(sourceDir, "test.txt"),
			[]byte("test"),
			0o600,
		))

		initGitRepoPerf(b, sourceDir)
		initGitRepoPerf(b, targetDir)

		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		// Test with increasing number of groups
		groupCounts := []int{1, 5, 10, 20, 50}
		var results []struct {
			count    int
			duration time.Duration
		}

		for _, count := range groupCounts {
			var groups []config.Group
			for i := 0; i < count; i++ {
				groups = append(groups, config.Group{
					Name:     fmt.Sprintf("Group %d", i),
					ID:       fmt.Sprintf("group-%d", i),
					Priority: i,
					Enabled:  boolPtr(true),
					Source: config.SourceConfig{
						Repo:   sourceDir,
						Branch: "main",
					},
					Targets: []config.TargetConfig{
						{
							Repo: targetDir,
							Files: []config.FileMapping{
								{
									Src:  "test.txt",
									Dest: fmt.Sprintf("output%d.txt", i),
								},
							},
						},
					},
				})
			}

			cfg := &config.Config{
				Version: 1,
				Groups:  groups,
			}

			ctx := context.Background()
			opts := syncpkg.DefaultOptions().
				WithDryRun(true)

			// Setup mocks
			mockGH := &gh.MockClient{}
			mockGit := &git.MockClient{}
			// Add broad GetChangedFiles mock to handle all calls
			mockGit.On("GetChangedFiles", mock.Anything, mock.Anything).Return([]string{"mocked-file.txt"}, nil).Maybe()
			mockState := &state.MockDiscoverer{}
			mockTransform := &transform.MockChain{}

			// Mock state discovery
			currentState := &state.State{
				Source: state.SourceState{
					Repo:         sourceDir,
					Branch:       "main",
					LatestCommit: "abc123",
				},
				Targets: map[string]*state.TargetState{
					targetDir: {
						Repo:           targetDir,
						LastSyncCommit: "old123", // Outdated to trigger sync
						Status:         state.StatusBehind,
					},
				},
			}
			mockState.On("DiscoverState", mock.Anything, mock.AnythingOfType("*config.Config")).Return(currentState, nil)

			// Mock git operations - with enough calls for multiple groups
			mockGit.On("Clone", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Checkout", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Add", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Commit", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			mockGit.On("Push", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			// Mock GitHub operations
			mockGH.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]byte("test content"), nil)
			mockGH.On("CreatePR", mock.Anything, mock.Anything).Return("https://github.com/org/repo/pull/1", nil)

			// Mock transform operations
			mockTransform.On("Apply", mock.Anything, mock.Anything).Return(mock.Anything, nil)

			engine := syncpkg.NewEngine(context.Background(), cfg, mockGH, mockGit, mockState, mockTransform, opts)

			start := time.Now()
			err := engine.Sync(ctx, nil)
			duration := time.Since(start)

			require.NoError(b, err)

			results = append(results, struct {
				count    int
				duration time.Duration
			}{count, duration})
		}

		// Log results
		b.Log("Performance scaling with group count:")
		for _, r := range results {
			perGroup := r.duration / time.Duration(r.count)
			b.Logf("  %2d groups: %10v total, %10v per group", r.count, r.duration, perGroup)
		}

		// Verify linear or better scaling
		// The time per group should not increase significantly
		firstPerGroup := results[0].duration / time.Duration(results[0].count)
		lastPerGroup := results[len(results)-1].duration / time.Duration(results[len(results)-1].count)

		// Allow up to 3x slowdown per group (should be much less in practice)
		assert.Less(b, lastPerGroup, firstPerGroup*3,
			"Performance should scale reasonably with group count")
	})
}
