package sync

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Static error variables for err113 linter compliance.
var (
	errMockCreate      = errors.New("mock create error")
	errMockUpdate      = errors.New("mock update error")
	errMockResult      = errors.New("mock result error")
	errMockFileChanges = errors.New("mock file changes error")
)

// Static error variables for lookup mock methods.
var (
	errMockLookup = errors.New("mock lookup error")
)

// mockSyncMetricsRecorder is a minimal mock implementing SyncMetricsRecorder.
type mockSyncMetricsRecorder struct{}

func (m *mockSyncMetricsRecorder) CreateSyncRun(_ context.Context, _ *BroadcastSyncRun) error {
	return errMockCreate
}

func (m *mockSyncMetricsRecorder) UpdateSyncRun(_ context.Context, _ *BroadcastSyncRun) error {
	return errMockUpdate
}

func (m *mockSyncMetricsRecorder) CreateTargetResult(_ context.Context, _ *BroadcastSyncTargetResult) error {
	return errMockResult
}

func (m *mockSyncMetricsRecorder) CreateFileChanges(_ context.Context, _ []BroadcastSyncFileChange) error {
	return errMockFileChanges
}

func (m *mockSyncMetricsRecorder) LookupGroupID(_ context.Context, _ string) (uint, error) {
	return 0, errMockLookup
}

func (m *mockSyncMetricsRecorder) LookupRepoID(_ context.Context, _ string) (uint, error) {
	return 0, errMockLookup
}

func (m *mockSyncMetricsRecorder) LookupTargetID(_ context.Context, _ uint, _ string) (uint, error) {
	return 0, errMockLookup
}

func (m *mockSyncMetricsRecorder) UpdateRepoSyncTimestamp(_ context.Context, _ uint, _ time.Time, _ uint) error {
	return nil
}

func TestEngine_MetricsRecorder(t *testing.T) {
	t.Parallel()

	t.Run("initially has no recorder", func(t *testing.T) {
		t.Parallel()

		engine := &Engine{
			logger: logrus.New(),
		}
		assert.False(t, engine.HasMetricsRecorder(), "new engine should not have a metrics recorder")
	})

	t.Run("set recorder makes HasMetricsRecorder true", func(t *testing.T) {
		t.Parallel()

		engine := &Engine{
			logger: logrus.New(),
		}
		require.False(t, engine.HasMetricsRecorder())

		recorder := &mockSyncMetricsRecorder{}
		engine.SetSyncMetricsRecorder(recorder)
		assert.True(t, engine.HasMetricsRecorder(), "engine should have a metrics recorder after setting one")
	})

	t.Run("set nil recorder makes HasMetricsRecorder false again", func(t *testing.T) {
		t.Parallel()

		engine := &Engine{
			logger: logrus.New(),
		}
		engine.SetSyncMetricsRecorder(&mockSyncMetricsRecorder{})
		require.True(t, engine.HasMetricsRecorder())

		engine.SetSyncMetricsRecorder(nil)
		assert.False(t, engine.HasMetricsRecorder(), "engine should not have a metrics recorder after setting nil")
	})
}

func TestEngine_CurrentRun(t *testing.T) {
	t.Parallel()

	t.Run("initially nil", func(t *testing.T) {
		t.Parallel()

		engine := &Engine{
			logger: logrus.New(),
		}
		assert.Nil(t, engine.GetCurrentRun(), "new engine should have nil current run")
	})

	t.Run("set and get run", func(t *testing.T) {
		t.Parallel()

		engine := &Engine{
			logger: logrus.New(),
		}
		run := &BroadcastSyncRun{
			ExternalID: "SR-20260219-abc123",
			Status:     SyncRunStatusRunning,
		}

		engine.setCurrentRun(run)
		got := engine.GetCurrentRun()
		require.NotNil(t, got)
		assert.Equal(t, "SR-20260219-abc123", got.ExternalID)
		assert.Equal(t, SyncRunStatusRunning, got.Status)
	})

	t.Run("set nil clears current run", func(t *testing.T) {
		t.Parallel()

		engine := &Engine{
			logger: logrus.New(),
		}
		engine.setCurrentRun(&BroadcastSyncRun{ExternalID: "SR-20260219-abc123"})
		require.NotNil(t, engine.GetCurrentRun())

		engine.setCurrentRun(nil)
		assert.Nil(t, engine.GetCurrentRun(), "current run should be nil after setting nil")
	})
}

func TestEngine_Options(t *testing.T) {
	t.Parallel()

	t.Run("returns options passed at construction", func(t *testing.T) {
		t.Parallel()

		opts := DefaultOptions().WithDryRun(true).WithMaxConcurrency(7)
		engine := &Engine{
			options: opts,
			logger:  logrus.New(),
		}
		got := engine.Options()
		require.NotNil(t, got)
		assert.True(t, got.DryRun)
		assert.Equal(t, 7, got.MaxConcurrency)
		assert.Same(t, opts, got, "Options() should return the same pointer")
	})

	t.Run("returns nil when no options set", func(t *testing.T) {
		t.Parallel()

		engine := &Engine{
			logger: logrus.New(),
		}
		assert.Nil(t, engine.Options())
	})
}

func TestEngine_RecordTargetStats(t *testing.T) {
	t.Parallel()

	t.Run("nil currentRun — no panic, no state change", func(t *testing.T) {
		t.Parallel()

		engine := &Engine{
			logger: logrus.New(),
		}
		// currentRun is nil — calling RecordTargetStats must not panic
		require.NotPanics(t, func() {
			engine.RecordTargetStats(5, 10, 3)
		})
		assert.Nil(t, engine.GetCurrentRun())
	})

	t.Run("active currentRun — totals accumulate correctly", func(t *testing.T) {
		t.Parallel()

		engine := &Engine{
			logger: logrus.New(),
		}
		engine.setCurrentRun(&BroadcastSyncRun{ExternalID: "SR-test", Status: SyncRunStatusRunning})

		engine.RecordTargetStats(3, 10, 2)
		engine.RecordTargetStats(2, 5, 1)

		run := engine.GetCurrentRun()
		require.NotNil(t, run)
		assert.Equal(t, 5, run.TotalFilesChanged)
		assert.Equal(t, 15, run.TotalLinesAdded)
		assert.Equal(t, 3, run.TotalLinesRemoved)
	})

	t.Run("concurrent calls — race-safe accumulation", func(t *testing.T) {
		t.Parallel()

		engine := &Engine{
			logger: logrus.New(),
		}
		engine.setCurrentRun(&BroadcastSyncRun{ExternalID: "SR-race", Status: SyncRunStatusRunning})

		const goroutines = 10
		var wg sync.WaitGroup
		wg.Add(goroutines)
		for range goroutines {
			go func() {
				defer wg.Done()
				engine.RecordTargetStats(1, 1, 1)
			}()
		}
		wg.Wait()

		run := engine.GetCurrentRun()
		require.NotNil(t, run)
		assert.Equal(t, goroutines, run.TotalFilesChanged)
		assert.Equal(t, goroutines, run.TotalLinesAdded)
		assert.Equal(t, goroutines, run.TotalLinesRemoved)
	})
}
