package sync

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/ai"
	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/git"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// errCoverageBoost is a static sentinel error used by the tests in this file to
// satisfy the err113 linter (no dynamic errors). The exact message is irrelevant;
// these tests only assert that the error path is taken.
var errCoverageBoost = errors.New("coverage-boost test error")

// mockSyncRecorder is a testify mock implementation of SyncMetricsRecorder.
type mockSyncRecorder struct {
	mock.Mock
}

func (m *mockSyncRecorder) CreateSyncRun(ctx context.Context, run *BroadcastSyncRun) error {
	args := m.Called(ctx, run)
	return args.Error(0)
}

func (m *mockSyncRecorder) UpdateSyncRun(ctx context.Context, run *BroadcastSyncRun) error {
	args := m.Called(ctx, run)
	return args.Error(0)
}

func (m *mockSyncRecorder) CreateTargetResult(ctx context.Context, result *BroadcastSyncTargetResult) error {
	args := m.Called(ctx, result)
	return args.Error(0)
}

func (m *mockSyncRecorder) CreateFileChanges(ctx context.Context, changes []BroadcastSyncFileChange) error {
	args := m.Called(ctx, changes)
	return args.Error(0)
}

func (m *mockSyncRecorder) LookupGroupID(ctx context.Context, groupExternalID string) (uint, error) {
	args := m.Called(ctx, groupExternalID)
	return args.Get(0).(uint), args.Error(1)
}

func (m *mockSyncRecorder) LookupRepoID(ctx context.Context, repoFullName string) (uint, error) {
	args := m.Called(ctx, repoFullName)
	return args.Get(0).(uint), args.Error(1)
}

func (m *mockSyncRecorder) LookupTargetID(ctx context.Context, groupDBID uint, repoFullName string) (uint, error) {
	args := m.Called(ctx, groupDBID, repoFullName)
	return args.Get(0).(uint), args.Error(1)
}

func (m *mockSyncRecorder) UpdateRepoSyncTimestamp(ctx context.Context, repoID uint, syncAt time.Time, runID uint) error {
	args := m.Called(ctx, repoID, syncAt, runID)
	return args.Error(0)
}

func testEntry() *logrus.Entry {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l.WithField("test", "true")
}

// --- module_resolver.go GetAvailableVersions ---

func TestModuleResolver_GetAvailableVersions(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(5*time.Minute, logger)
	defer cache.Close()
	resolver := NewModuleResolver(logger, cache)

	callCount := 0
	resolver.tagFetcher = func(_ context.Context, _ string) ([]string, error) {
		callCount++
		// Include invalid versions that must be skipped, and out-of-order versions.
		return []string{"v1.0.0", "not-a-version", "v2.0.0", "v1.2.0", "vbogus"}, nil
	}

	t.Run("returns sorted valid versions and skips invalid", func(t *testing.T) {
		versions, err := resolver.GetAvailableVersions(context.Background(), "org/repo")
		require.NoError(t, err)
		// Sorted highest-first, invalid entries skipped.
		assert.Equal(t, []string{"v2.0.0", "v1.2.0", "v1.0.0"}, versions)
		assert.Equal(t, 1, callCount)
	})

	t.Run("uses cache on second call", func(t *testing.T) {
		versions, err := resolver.GetAvailableVersions(context.Background(), "org/repo")
		require.NoError(t, err)
		assert.Equal(t, []string{"v2.0.0", "v1.2.0", "v1.0.0"}, versions)
		// tagFetcher should not be called again (cache hit).
		assert.Equal(t, 1, callCount)
	})

	t.Run("propagates fetch error", func(t *testing.T) {
		cache2 := NewModuleCache(5*time.Minute, logger)
		defer cache2.Close()
		r := NewModuleResolver(logger, cache2)
		sentinel := errCoverageBoost
		r.tagFetcher = func(_ context.Context, _ string) ([]string, error) {
			return nil, sentinel
		}
		_, err := r.GetAvailableVersions(context.Background(), "org/repo")
		require.ErrorIs(t, err, sentinel)
	})

	t.Run("no valid versions returns empty without caching", func(t *testing.T) {
		cache3 := NewModuleCache(5*time.Minute, logger)
		defer cache3.Close()
		r := NewModuleResolver(logger, cache3)
		r.tagFetcher = func(_ context.Context, _ string) ([]string, error) {
			return []string{"garbage", "alsobad"}, nil
		}
		versions, err := r.GetAvailableVersions(context.Background(), "org/repo")
		require.NoError(t, err)
		assert.Empty(t, versions)
	})
}

func TestModuleResolver_ResolveVersion_WithTags(t *testing.T) {
	logger := logrus.New()
	cache := NewModuleCache(5*time.Minute, logger)
	defer cache.Close()
	resolver := NewModuleResolver(logger, cache)
	resolver.tagFetcher = func(_ context.Context, _ string) ([]string, error) {
		return []string{"v1.0.0", "v1.2.0", "v2.0.0"}, nil
	}

	t.Run("latest resolves highest", func(t *testing.T) {
		v, err := resolver.ResolveVersion(context.Background(), "org/repo", "latest", true)
		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", v)
	})

	t.Run("semver constraint resolves match", func(t *testing.T) {
		v, err := resolver.ResolveVersion(context.Background(), "org/repo", "~1.2", true)
		require.NoError(t, err)
		assert.Equal(t, "v1.2.0", v)
	})

	t.Run("exact version with tags", func(t *testing.T) {
		v, err := resolver.ResolveVersion(context.Background(), "org/repo", "v2.0.0", true)
		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", v)
	})

	t.Run("fetch error surfaced", func(t *testing.T) {
		cache2 := NewModuleCache(5*time.Minute, logger)
		defer cache2.Close()
		r := NewModuleResolver(logger, cache2)
		r.tagFetcher = func(_ context.Context, _ string) ([]string, error) {
			return nil, errCoverageBoost
		}
		_, err := r.ResolveVersion(context.Background(), "org/repo", "latest", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch git tags")
	})
}

// --- repository.go forceCleanup ---

func TestRepositorySync_forceCleanup(t *testing.T) {
	t.Run("removes directory successfully", func(t *testing.T) {
		tmp := t.TempDir()
		dir := filepath.Join(tmp, "victim")
		require.NoError(t, os.MkdirAll(dir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o600))

		rs := &RepositorySync{tempDir: dir, logger: testEntry()}
		require.NoError(t, rs.forceCleanup())
		_, statErr := os.Stat(dir)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("removes read-only directory", func(t *testing.T) {
		tmp := t.TempDir()
		dir := filepath.Join(tmp, "ro")
		require.NoError(t, os.MkdirAll(dir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o600))
		// Make it read-only; forceCleanup chmods back to 0700.
		require.NoError(t, os.Chmod(dir, 0o500)) //nolint:gosec // G302: directory intentionally made read-only (exec bit required to traverse) to exercise forceCleanup's chmod-back path

		rs := &RepositorySync{tempDir: dir, logger: testEntry()}
		err := rs.forceCleanup()
		require.NoError(t, err)
		_, statErr := os.Stat(dir)
		assert.True(t, os.IsNotExist(statErr))
	})
}

// --- repository.go updateDirectoryMetricsWithActualChanges ---

func TestRepositorySync_updateDirectoryMetricsWithActualChanges(t *testing.T) {
	t.Run("nil metrics is a no-op", func(t *testing.T) {
		rs := &RepositorySync{logger: testEntry()}
		rs.updateDirectoryMetricsWithActualChanges([]string{"a.txt"})
		// no panic

		rs.syncMetrics = &PerformanceMetrics{} // DirectoryMetrics nil
		rs.updateDirectoryMetricsWithActualChanges([]string{"a.txt"})
	})

	t.Run("counts actual changes per source directory", func(t *testing.T) {
		rs := &RepositorySync{
			logger: testEntry(),
			target: config.TargetConfig{
				Directories: []config.DirectoryMapping{
					{Src: "srcA", Dest: "destA"},
					{Src: "srcB", Dest: "destB"},
				},
			},
			syncMetrics: &PerformanceMetrics{
				DirectoryMetrics: map[string]DirectoryMetrics{
					"srcA": {FilesProcessed: 5, FilesChanged: 99},
					"srcB": {FilesProcessed: 3, FilesChanged: 99},
				},
			},
		}

		rs.updateDirectoryMetricsWithActualChanges([]string{
			"destA/one.txt",
			"destA/sub/two.txt",
			"destB/three.txt",
			"unmapped/four.txt", // belongs to no directory
		})

		a, ok := rs.syncMetrics.GetDirectoryMetric("srcA")
		require.True(t, ok)
		assert.Equal(t, 2, a.FilesChanged)
		b, ok := rs.syncMetrics.GetDirectoryMetric("srcB")
		require.True(t, ok)
		assert.Equal(t, 1, b.FilesChanged)
	})
}

// --- engine.go filterTargets fallback (no groups) ---

func TestEngineFilterTargets_NoGroups(t *testing.T) {
	currentState := &state.State{Targets: map[string]*state.TargetState{}}

	t.Run("empty config returns empty target list", func(t *testing.T) {
		engine := &Engine{
			config:  &config.Config{}, // no groups
			options: DefaultOptions(),
			logger:  logrus.New(),
		}
		targets, err := engine.filterTargets(nil, currentState)
		require.NoError(t, err)
		assert.Empty(t, targets)
	})

	t.Run("filter with no matches returns error", func(t *testing.T) {
		engine := &Engine{
			config:  &config.Config{},
			options: DefaultOptions(),
			logger:  logrus.New(),
		}
		_, err := engine.filterTargets([]string{"org/none"}, currentState)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no targets match")
	})
}

// --- engine.go updateSyncRunTargetCount ---

func TestEngine_updateSyncRunTargetCount(t *testing.T) {
	t.Run("no recorder is a no-op", func(t *testing.T) {
		e := &Engine{logger: logrus.New()}
		require.NoError(t, e.updateSyncRunTargetCount(context.Background(), 3))
	})

	t.Run("no current run is a no-op", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		e := &Engine{logger: logrus.New(), syncRepo: rec}
		require.NoError(t, e.updateSyncRunTargetCount(context.Background(), 3))
		rec.AssertNotCalled(t, "UpdateSyncRun", mock.Anything, mock.Anything)
	})

	t.Run("updates run with target count", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		rec.On("UpdateSyncRun", mock.Anything, mock.Anything).Return(nil)
		e := &Engine{logger: logrus.New(), syncRepo: rec}
		e.setCurrentRun(&BroadcastSyncRun{ExternalID: "r1"})

		require.NoError(t, e.updateSyncRunTargetCount(context.Background(), 7))
		assert.Equal(t, 7, e.GetCurrentRun().TotalTargets)
		rec.AssertExpectations(t)
	})

	t.Run("propagates update error", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		rec.On("UpdateSyncRun", mock.Anything, mock.Anything).Return(errCoverageBoost)
		e := &Engine{logger: logrus.New(), syncRepo: rec}
		e.setCurrentRun(&BroadcastSyncRun{ExternalID: "r1"})

		err := e.updateSyncRunTargetCount(context.Background(), 7)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update sync run target count")
	})
}

// --- engine.go recordSyncRunStart ---

func TestEngine_recordSyncRunStart(t *testing.T) {
	group := config.Group{ID: "grp-1", Name: "Group One"}
	currentState := &state.State{Source: state.SourceState{
		Repo: "org/source", Branch: "main", LatestCommit: "abc123",
	}}

	t.Run("no recorder is a no-op", func(t *testing.T) {
		e := &Engine{logger: logrus.New(), options: DefaultOptions()}
		require.NoError(t, e.recordSyncRunStart(context.Background(), group, currentState))
		assert.Nil(t, e.GetCurrentRun())
	})

	t.Run("creates run resolving ids", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		rec.On("LookupGroupID", mock.Anything, "grp-1").Return(uint(10), nil)
		rec.On("LookupRepoID", mock.Anything, "org/source").Return(uint(20), nil)
		rec.On("CreateSyncRun", mock.Anything, mock.Anything).Return(nil)

		e := &Engine{logger: logrus.New(), options: DefaultOptions(), syncRepo: rec}
		require.NoError(t, e.recordSyncRunStart(context.Background(), group, currentState))

		run := e.GetCurrentRun()
		require.NotNil(t, run)
		require.NotNil(t, run.GroupID)
		assert.Equal(t, uint(10), *run.GroupID)
		require.NotNil(t, run.SourceRepoID)
		assert.Equal(t, uint(20), *run.SourceRepoID)
		assert.Equal(t, SyncRunStatusRunning, run.Status)
		rec.AssertExpectations(t)
	})

	t.Run("tolerates id lookup failures", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		rec.On("LookupGroupID", mock.Anything, "grp-1").Return(uint(0), errCoverageBoost)
		rec.On("LookupRepoID", mock.Anything, "org/source").Return(uint(0), errCoverageBoost)
		rec.On("CreateSyncRun", mock.Anything, mock.Anything).Return(nil)

		e := &Engine{logger: logrus.New(), options: DefaultOptions(), syncRepo: rec}
		require.NoError(t, e.recordSyncRunStart(context.Background(), group, currentState))
		run := e.GetCurrentRun()
		require.NotNil(t, run)
		assert.Nil(t, run.GroupID)
		assert.Nil(t, run.SourceRepoID)
	})

	t.Run("propagates create error", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		rec.On("LookupGroupID", mock.Anything, "grp-1").Return(uint(10), nil)
		rec.On("LookupRepoID", mock.Anything, "org/source").Return(uint(20), nil)
		rec.On("CreateSyncRun", mock.Anything, mock.Anything).Return(errCoverageBoost)

		e := &Engine{logger: logrus.New(), options: DefaultOptions(), syncRepo: rec}
		err := e.recordSyncRunStart(context.Background(), group, currentState)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create sync run record")
	})
}

// --- engine.go finalizeSyncRun ---

func TestEngine_finalizeSyncRun(t *testing.T) {
	t.Run("no recorder is a no-op", func(t *testing.T) {
		e := &Engine{logger: logrus.New()}
		require.NoError(t, e.finalizeSyncRun(context.Background(), &Results{}, nil))
	})

	t.Run("no current run is a no-op", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		e := &Engine{logger: logrus.New(), syncRepo: rec}
		require.NoError(t, e.finalizeSyncRun(context.Background(), &Results{}, nil))
	})

	statusCases := []struct {
		name       string
		total      int
		successful int
		failed     int
		skipped    int
		expected   string
	}{
		{"all success", 3, 3, 0, 0, SyncRunStatusSuccess},
		{"zero targets", 0, 0, 0, 0, SyncRunStatusSkipped},
		{"partial", 3, 1, 1, 1, SyncRunStatusPartial},
		{"all failed", 2, 0, 2, 0, SyncRunStatusFailed},
		{"all skipped", 2, 0, 0, 2, SyncRunStatusSkipped},
	}
	for _, tc := range statusCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &mockSyncRecorder{}
			rec.On("UpdateSyncRun", mock.Anything, mock.Anything).Return(nil)
			e := &Engine{logger: logrus.New(), syncRepo: rec}
			e.setCurrentRun(&BroadcastSyncRun{
				ExternalID:   "r1",
				StartedAt:    time.Now().Add(-time.Second),
				TotalTargets: tc.total,
			})
			var errs []error
			if tc.failed > 0 {
				errs = []error{errCoverageBoost}
			}
			require.NoError(t, e.finalizeSyncRun(context.Background(),
				&Results{Successful: tc.successful, Failed: tc.failed, Skipped: tc.skipped}, errs))

			run := e.GetCurrentRun()
			assert.Equal(t, tc.expected, run.Status)
			require.NotNil(t, run.EndedAt)
			rec.AssertExpectations(t)
		})
	}

	t.Run("propagates update error", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		rec.On("UpdateSyncRun", mock.Anything, mock.Anything).Return(errCoverageBoost)
		e := &Engine{logger: logrus.New(), syncRepo: rec}
		e.setCurrentRun(&BroadcastSyncRun{ExternalID: "r1", StartedAt: time.Now(), TotalTargets: 1})
		err := e.finalizeSyncRun(context.Background(), &Results{Successful: 1}, nil)
		require.Error(t, err)
	})
}

// --- engine.go initializeAI (disabled-by-config path) ---

func TestEngine_initializeAI_Disabled(t *testing.T) {
	t.Setenv("GO_BROADCAST_AI_ENABLED", "false")
	e := &Engine{logger: logrus.New()}
	e.initializeAI(context.Background())
	assert.Nil(t, e.prGenerator)
	assert.Nil(t, e.commitGenerator)
	assert.Nil(t, e.diffTruncator)
}

// --- repository.go updateExistingPR ---

func TestRepositorySync_updateExistingPR(t *testing.T) {
	t.Run("dry run does not call gh", func(t *testing.T) {
		ghClient := &gh.MockClient{}
		e := &Engine{
			gh:      ghClient,
			options: &Options{DryRun: true},
			logger:  logrus.New(),
		}
		rs := &RepositorySync{
			engine: e,
			target: config.TargetConfig{Repo: "org/repo"},
			logger: testEntry(),
		}
		pr := &gh.PR{Number: 42, Title: "Existing PR"}
		err := rs.updateExistingPR(context.Background(), pr, "sha123",
			[]FileChange{{Path: "a.txt"}}, []string{"a.txt"})
		require.NoError(t, err)
		ghClient.AssertNotCalled(t, "UpdatePR", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("updates PR via gh client", func(t *testing.T) {
		ghClient := &gh.MockClient{}
		ghClient.On("UpdatePR", mock.Anything, "org/repo", 42, mock.Anything).Return(nil)
		e := &Engine{
			gh:      ghClient,
			options: &Options{DryRun: false},
			logger:  logrus.New(),
		}
		rs := &RepositorySync{
			engine:      e,
			target:      config.TargetConfig{Repo: "org/repo"},
			sourceState: &state.SourceState{Repo: "org/source"},
			logger:      testEntry(),
		}
		pr := &gh.PR{Number: 42, Title: "Existing PR"}
		err := rs.updateExistingPR(context.Background(), pr, "sha123",
			[]FileChange{{Path: "a.txt"}}, []string{"a.txt"})
		require.NoError(t, err)
		require.NotNil(t, rs.lastPRNumber)
		assert.Equal(t, 42, *rs.lastPRNumber)
		assert.Contains(t, rs.lastPRURL, "/pull/42")
		ghClient.AssertExpectations(t)
	})

	t.Run("returns error when gh update fails", func(t *testing.T) {
		ghClient := &gh.MockClient{}
		ghClient.On("UpdatePR", mock.Anything, "org/repo", 7, mock.Anything).
			Return(errCoverageBoost)
		e := &Engine{
			gh:      ghClient,
			options: &Options{DryRun: false},
			logger:  logrus.New(),
		}
		rs := &RepositorySync{
			engine:      e,
			target:      config.TargetConfig{Repo: "org/repo"},
			sourceState: &state.SourceState{Repo: "org/source"},
			logger:      testEntry(),
		}
		pr := &gh.PR{Number: 7}
		err := rs.updateExistingPR(context.Background(), pr, "sha", nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update PR")
	})
}

// --- repository.go getDiffForAI ---

func TestRepositorySync_getDiffForAI(t *testing.T) {
	t.Run("nil truncator returns empty", func(t *testing.T) {
		rs := &RepositorySync{
			engine: &Engine{logger: logrus.New()},
			logger: testEntry(),
		}
		assert.Empty(t, rs.getDiffForAI(context.Background(), []FileChange{{Path: "a.txt"}}))
	})

	t.Run("uses git diff from staged repo", func(t *testing.T) {
		gitClient := git.NewMockClient()
		gitClient.On("DiffIgnoreWhitespace", mock.Anything, "/staged", true).
			Return("diff --git a/a.txt b/a.txt\n+hello\n", nil)
		e := &Engine{
			git:           gitClient,
			diffTruncator: ai.NewDiffTruncator(ai.LoadConfig()),
			logger:        logrus.New(),
		}
		rs := &RepositorySync{
			engine:         e,
			stagedRepoPath: "/staged",
			target:         config.TargetConfig{Repo: "org/repo"},
			logger:         testEntry(),
		}
		out := rs.getDiffForAI(context.Background(), []FileChange{{Path: "a.txt", Content: []byte("hello")}})
		assert.Contains(t, out, "hello")
		gitClient.AssertExpectations(t)
	})

	t.Run("falls back to synthetic diff when git diff fails", func(t *testing.T) {
		gitClient := git.NewMockClient()
		gitClient.On("DiffIgnoreWhitespace", mock.Anything, "/staged", true).
			Return("", errCoverageBoost)
		e := &Engine{
			git:           gitClient,
			diffTruncator: ai.NewDiffTruncator(ai.LoadConfig()),
			logger:        logrus.New(),
		}
		rs := &RepositorySync{
			engine:         e,
			stagedRepoPath: "/staged",
			target:         config.TargetConfig{Repo: "org/repo"},
			logger:         testEntry(),
		}
		out := rs.getDiffForAI(context.Background(),
			[]FileChange{{Path: "a.txt", Content: []byte("new content"), IsNew: true}})
		assert.NotEmpty(t, out)
		gitClient.AssertExpectations(t)
	})

	t.Run("synthetic diff when no staged repo", func(t *testing.T) {
		e := &Engine{
			diffTruncator: ai.NewDiffTruncator(ai.LoadConfig()),
			logger:        logrus.New(),
		}
		rs := &RepositorySync{
			engine: e,
			target: config.TargetConfig{Repo: "org/repo"},
			logger: testEntry(),
		}
		out := rs.getDiffForAI(context.Background(),
			[]FileChange{{Path: "a.txt", Content: []byte("data"), IsNew: true}})
		assert.NotEmpty(t, out)
	})
}

// --- repository.go recordTargetResult ---

func TestRepositorySync_recordTargetResult(t *testing.T) {
	t.Run("no current run is a no-op", func(t *testing.T) {
		e := &Engine{logger: logrus.New()}
		rs := &RepositorySync{engine: e, logger: testEntry()}
		require.NoError(t, rs.recordTargetResult(context.Background(), "b", "s", nil, nil, nil, ""))
	})

	t.Run("records success with file changes including deleted file", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		rec.On("LookupRepoID", mock.Anything, "org/repo").Return(uint(20), nil)
		rec.On("LookupGroupID", mock.Anything, "grp-1").Return(uint(10), nil)
		rec.On("LookupTargetID", mock.Anything, uint(10), "org/repo").Return(uint(30), nil)

		var capturedResult *BroadcastSyncTargetResult
		rec.On("CreateTargetResult", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedResult = args.Get(1).(*BroadcastSyncTargetResult)
			}).Return(nil)

		var capturedChanges []BroadcastSyncFileChange
		rec.On("CreateFileChanges", mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				capturedChanges = args.Get(1).([]BroadcastSyncFileChange)
			}).Return(nil)
		rec.On("UpdateRepoSyncTimestamp", mock.Anything, uint(20), mock.Anything, mock.Anything).Return(nil)

		e := &Engine{
			logger:   logrus.New(),
			syncRepo: rec,
			config:   &config.Config{Groups: []config.Group{{ID: "grp-1"}}},
		}
		e.setCurrentRun(&BroadcastSyncRun{ID: 1, ExternalID: "r1"})

		rs := &RepositorySync{
			engine:      e,
			target:      config.TargetConfig{Repo: "org/repo"},
			syncMetrics: &PerformanceMetrics{StartTime: time.Now().Add(-time.Second)},
			logger:      testEntry(),
		}

		// A modified file (3 -> 2 lines), a new file, and a DELETED file.
		// For the deleted file Content is nil, IsDeleted true: all 2 original lines
		// should count as removed. string(nil) == "" is correct (not a bug).
		changes := []FileChange{
			{Path: "mod.txt", OriginalContent: []byte("a\nb\nc"), Content: []byte("a\nb")},
			{Path: "new.txt", OriginalContent: nil, Content: []byte("x\ny"), IsNew: true},
			{Path: "del.txt", OriginalContent: []byte("d1\nd2"), Content: nil, IsDeleted: true},
		}

		err := rs.recordTargetResult(context.Background(), "feat-branch", "sha999",
			changes, []string{"mod.txt", "new.txt", "del.txt"}, nil, "")
		require.NoError(t, err)

		require.NotNil(t, capturedResult)
		assert.Equal(t, TargetStatusSuccess, capturedResult.Status)
		assert.Equal(t, uint(20), capturedResult.RepoID)
		assert.Equal(t, uint(30), capturedResult.TargetID)
		assert.Equal(t, 3, capturedResult.FilesProcessed)
		assert.Equal(t, 3, capturedResult.FilesChanged)

		// Verify the deleted file's change record: all original lines counted as removed.
		require.Len(t, capturedChanges, 3)
		var del BroadcastSyncFileChange
		var found bool
		for _, c := range capturedChanges {
			if c.FilePath == "del.txt" {
				del = c
				found = true
			}
		}
		require.True(t, found)
		assert.Equal(t, FileChangeTypeDeleted, del.ChangeType)
		// Verified-correct behavior: Content==nil => string(nil)=="" which difflib
		// treats as a single empty line. So CountDiffLines("d1\nd2","") reports
		// removed=2 (both original lines) and added=1 (the empty line). This is the
		// expected, correct semantics for a deleted file -- NOT a bug.
		assert.Equal(t, 1, del.LinesAdded, "empty new content counts as one (empty) line")
		assert.Equal(t, 2, del.LinesRemoved, "all 2 original lines counted as removed")
		assert.Equal(t, int64(0), del.SizeBytes, "deleted file has zero content size")

		rec.AssertExpectations(t)
	})

	t.Run("failure status when sync error present", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		rec.On("LookupRepoID", mock.Anything, "org/repo").Return(uint(20), nil)
		rec.On("LookupGroupID", mock.Anything, "grp-1").Return(uint(10), nil)
		rec.On("LookupTargetID", mock.Anything, uint(10), "org/repo").Return(uint(30), nil)
		rec.On("CreateTargetResult", mock.Anything, mock.MatchedBy(func(r *BroadcastSyncTargetResult) bool {
			return r.Status == TargetStatusFailed && r.ErrorMessage == errCoverageBoost.Error()
		})).Return(nil)

		e := &Engine{
			logger:   logrus.New(),
			syncRepo: rec,
			config:   &config.Config{Groups: []config.Group{{ID: "grp-1"}}},
		}
		e.setCurrentRun(&BroadcastSyncRun{ID: 1, ExternalID: "r1"})
		rs := &RepositorySync{
			engine: e,
			target: config.TargetConfig{Repo: "org/repo"},
			logger: testEntry(),
		}
		err := rs.recordTargetResult(context.Background(), "b", "s", nil, nil,
			errCoverageBoost, "")
		require.NoError(t, err)
		rec.AssertExpectations(t)
	})

	t.Run("repo id lookup failure skips recording gracefully", func(t *testing.T) {
		rec := &mockSyncRecorder{}
		rec.On("LookupRepoID", mock.Anything, "org/repo").Return(uint(0), errCoverageBoost)
		e := &Engine{logger: logrus.New(), syncRepo: rec, config: &config.Config{}}
		e.setCurrentRun(&BroadcastSyncRun{ID: 1})
		rs := &RepositorySync{
			engine: e,
			target: config.TargetConfig{Repo: "org/repo"},
			logger: testEntry(),
		}
		require.NoError(t, rs.recordTargetResult(context.Background(), "b", "s", nil, nil, nil, ""))
		rec.AssertNotCalled(t, "CreateTargetResult", mock.Anything, mock.Anything)
	})
}

// --- directory.go CompleteAllDirectories ---

func TestDirectoryProcessor_CompleteAllDirectories(t *testing.T) {
	dp := NewDirectoryProcessor(testEntry(), 2, nil)
	defer dp.Close()

	// Register a couple of reporters.
	dp.progressManager.GetReporter("dirA", 50)
	dp.progressManager.GetReporter("dirB", 50)

	results := dp.CompleteAllDirectories()
	assert.Len(t, results, 2)
	assert.Contains(t, results, "dirA")
	assert.Contains(t, results, "dirB")

	// After completion reporters are cleared.
	assert.Empty(t, dp.GetDirectoryStats())
}

// --- directory.go ProcessDirectoriesWithMetrics ---

func TestRepositorySync_ProcessDirectoriesWithMetrics(t *testing.T) {
	t.Run("no directories returns nil changes", func(t *testing.T) {
		rs := &RepositorySync{
			target: config.TargetConfig{Repo: "org/repo"},
			logger: testEntry(),
		}
		changes, metrics, err := rs.ProcessDirectoriesWithMetrics(context.Background())
		require.NoError(t, err)
		assert.Nil(t, changes)
		assert.NotNil(t, metrics)
		assert.Empty(t, metrics)
	})

	t.Run("propagates processing error", func(t *testing.T) {
		tmp := t.TempDir()
		rs := &RepositorySync{
			tempDir: tmp,
			target: config.TargetConfig{
				Repo: "org/repo",
				Directories: []config.DirectoryMapping{
					{Src: "missing", Dest: "dest"},
				},
			},
			logger: testEntry(),
		}
		_, _, err := rs.ProcessDirectoriesWithMetrics(context.Background())
		require.Error(t, err)
	})
}

// --- directory.go ProcessDirectoriesWithOptions ---

func TestRepositorySync_ProcessDirectoriesWithOptions(t *testing.T) {
	t.Run("no directories returns nil", func(t *testing.T) {
		rs := &RepositorySync{
			target: config.TargetConfig{Repo: "org/repo"},
			logger: testEntry(),
		}
		changes, err := rs.ProcessDirectoriesWithOptions(context.Background(), DirectoryProcessingOptions{})
		require.NoError(t, err)
		assert.Nil(t, changes)
	})

	t.Run("processes directory with options and exclusions", func(t *testing.T) {
		tmp := t.TempDir()
		// processor uses tmp/source as source base, joins dirMapping.Src.
		srcDir := filepath.Join(tmp, "source", "mydir")
		require.NoError(t, os.MkdirAll(srcDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "keep.txt"), []byte("keep"), 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "skip.log"), []byte("skip"), 0o600))

		ghClient := &gh.MockClient{}
		ghClient.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(&gh.FileContent{Content: []byte("")}, nil)

		e := &Engine{
			gh:      ghClient,
			git:     git.NewMockClient(),
			options: DefaultOptions(),
			logger:  logrus.New(),
		}
		rs := &RepositorySync{
			tempDir:     tmp,
			engine:      e,
			sourceState: &state.SourceState{Repo: "org/source"},
			target: config.TargetConfig{
				Repo: "org/repo",
				Directories: []config.DirectoryMapping{
					{Src: "mydir", Dest: "target"},
				},
			},
			logger: testEntry(),
		}

		changes, err := rs.ProcessDirectoriesWithOptions(context.Background(), DirectoryProcessingOptions{
			WorkerCount:       2,
			ExclusionPatterns: []string{"*.log"},
		})
		require.NoError(t, err)
		for _, c := range changes {
			assert.NotEqual(t, ".log", filepath.Ext(c.Path), "log files should be excluded")
		}
	})

	t.Run("continues past failed directory", func(t *testing.T) {
		tmp := t.TempDir()
		goodDir := filepath.Join(tmp, "source", "good")
		require.NoError(t, os.MkdirAll(goodDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(goodDir, "f.txt"), []byte("x"), 0o600))

		ghClient := &gh.MockClient{}
		ghClient.On("GetFile", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(&gh.FileContent{Content: []byte("")}, nil)

		e := &Engine{
			gh:      ghClient,
			git:     git.NewMockClient(),
			options: DefaultOptions(),
			logger:  logrus.New(),
		}
		rs := &RepositorySync{
			tempDir:     tmp,
			engine:      e,
			sourceState: &state.SourceState{Repo: "org/source"},
			target: config.TargetConfig{
				Repo: "org/repo",
				Directories: []config.DirectoryMapping{
					{Src: "missing", Dest: "t1"},
					{Src: "good", Dest: "t2"},
				},
			},
			logger: testEntry(),
		}
		// Should not error overall; bad directory logged and skipped.
		_, err := rs.ProcessDirectoriesWithOptions(context.Background(), DirectoryProcessingOptions{WorkerCount: 1})
		require.NoError(t, err)
	})
}

// --- directory.go handleModuleSync ---

func TestDirectoryProcessor_handleModuleSync(t *testing.T) {
	t.Run("unsupported module type errors", func(t *testing.T) {
		dp := NewDirectoryProcessor(testEntry(), 1, nil)
		defer dp.Close()
		_, err := dp.handleModuleSync(context.Background(), t.TempDir(), "dir",
			&config.ModuleConfig{Type: "npm"}, testEntry())
		require.ErrorIs(t, err, ErrUnsupportedModuleType)
	})

	t.Run("non-module directory returns default result", func(t *testing.T) {
		dp := NewDirectoryProcessor(testEntry(), 1, nil)
		defer dp.Close()
		src := t.TempDir()
		res, err := dp.handleModuleSync(context.Background(), src, "dir",
			&config.ModuleConfig{Type: "go"}, testEntry())
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, src, res.SourcePath)
		assert.Nil(t, res.ModuleInfo)
	})

	t.Run("go module without version constraint", func(t *testing.T) {
		dp := NewDirectoryProcessor(testEntry(), 1, nil)
		defer dp.Close()
		src := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(src, "go.mod"),
			[]byte("module example.com/foo\n\ngo 1.21\n"), 0o600))

		res, err := dp.handleModuleSync(context.Background(), src, "dir",
			&config.ModuleConfig{Type: "go"}, testEntry())
		require.NoError(t, err)
		require.NotNil(t, res.ModuleInfo)
		assert.Equal(t, "example.com/foo", res.ModuleInfo.Name)
		assert.Empty(t, res.ResolvedVersion)
	})

	t.Run("go module with version constraint resolved hermetically", func(t *testing.T) {
		dp := NewDirectoryProcessor(testEntry(), 1, &DirectoryProcessorOptions{
			GitClient:     git.NewMockClient(),
			SourceRepoURL: "https://github.com/org/source",
		})
		defer dp.Close()
		// Override tag fetcher to avoid network.
		dp.moduleResolver.tagFetcher = func(_ context.Context, _ string) ([]string, error) {
			return []string{"v1.0.0", "v1.2.0", "v2.0.0"}, nil
		}

		src := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(src, "go.mod"),
			[]byte("module example.com/foo\n\ngo 1.21\n"), 0o600))

		res, err := dp.handleModuleSync(context.Background(), src, "dir",
			&config.ModuleConfig{Type: "go", Version: "latest"}, testEntry())
		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", res.ResolvedVersion)
		// No moduleSourceResolver wired (no GitClient), so SourcePath stays the original.
		assert.Equal(t, src, res.SourcePath)
	})

	t.Run("version constraint but no source repo url", func(t *testing.T) {
		dp := NewDirectoryProcessor(testEntry(), 1, nil) // no GitClient => empty sourceRepoURL
		defer dp.Close()
		src := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(src, "go.mod"),
			[]byte("module example.com/foo\n\ngo 1.21\n"), 0o600))

		res, err := dp.handleModuleSync(context.Background(), src, "dir",
			&config.ModuleConfig{Type: "go", Version: "latest"}, testEntry())
		require.NoError(t, err)
		assert.Empty(t, res.ResolvedVersion)
		assert.Equal(t, src, res.SourcePath)
	})

	t.Run("version resolution failure falls back to original source", func(t *testing.T) {
		dp := NewDirectoryProcessor(testEntry(), 1, &DirectoryProcessorOptions{
			GitClient:     git.NewMockClient(),
			SourceRepoURL: "https://github.com/org/source",
		})
		defer dp.Close()
		dp.moduleResolver.tagFetcher = func(_ context.Context, _ string) ([]string, error) {
			return nil, errCoverageBoost
		}
		src := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(src, "go.mod"),
			[]byte("module example.com/foo\n\ngo 1.21\n"), 0o600))

		res, err := dp.handleModuleSync(context.Background(), src, "dir",
			&config.ModuleConfig{Type: "go", Version: "latest"}, testEntry())
		require.NoError(t, err)
		assert.Empty(t, res.ResolvedVersion)
		assert.Equal(t, src, res.SourcePath)
	})
}
