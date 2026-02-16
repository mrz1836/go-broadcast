package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBroadcastSyncRepo_CreateSyncRun tests creating a sync run
func TestBroadcastSyncRepo_CreateSyncRun(t *testing.T) {
	db := TestDB(t)
	repo := NewBroadcastSyncRepo(db)
	ctx := context.Background()

	t.Run("create valid run", func(t *testing.T) {
		run := &BroadcastSyncRun{
			ExternalID:   "SR-20260215-abc123",
			StartedAt:    time.Now(),
			Status:       BroadcastSyncRunStatusRunning,
			Trigger:      BroadcastSyncRunTriggerManual,
			TotalTargets: 3,
		}

		err := repo.CreateSyncRun(ctx, run)
		require.NoError(t, err)
		assert.NotZero(t, run.ID)
	})

	t.Run("missing external ID", func(t *testing.T) {
		run := &BroadcastSyncRun{
			StartedAt:    time.Now(),
			Status:       BroadcastSyncRunStatusRunning,
			TotalTargets: 1,
		}

		err := repo.CreateSyncRun(ctx, run)
		assert.ErrorIs(t, err, ErrMissingExternalID)
	})

	t.Run("duplicate external ID", func(t *testing.T) {
		run1 := &BroadcastSyncRun{
			ExternalID:   "SR-20260215-dup",
			StartedAt:    time.Now(),
			Status:       BroadcastSyncRunStatusRunning,
			TotalTargets: 1,
		}

		err := repo.CreateSyncRun(ctx, run1)
		require.NoError(t, err)

		run2 := &BroadcastSyncRun{
			ExternalID:   "SR-20260215-dup",
			StartedAt:    time.Now(),
			Status:       BroadcastSyncRunStatusRunning,
			TotalTargets: 1,
		}

		err = repo.CreateSyncRun(ctx, run2)
		assert.Error(t, err)
	})
}

// TestBroadcastSyncRepo_UpdateSyncRun tests updating a sync run
func TestBroadcastSyncRepo_UpdateSyncRun(t *testing.T) {
	db := TestDB(t)

	repo := NewBroadcastSyncRepo(db)
	ctx := context.Background()

	t.Run("update existing run", func(t *testing.T) {
		run := &BroadcastSyncRun{
			ExternalID:   "SR-20260215-update",
			StartedAt:    time.Now(),
			Status:       BroadcastSyncRunStatusRunning,
			TotalTargets: 2,
		}

		err := repo.CreateSyncRun(ctx, run)
		require.NoError(t, err)

		// Update status
		endedAt := time.Now()
		run.EndedAt = &endedAt
		run.Status = BroadcastSyncRunStatusSuccess
		run.SuccessfulTargets = 2
		run.DurationMs = endedAt.Sub(run.StartedAt).Milliseconds()

		err = repo.UpdateSyncRun(ctx, run)
		require.NoError(t, err)

		// Verify update
		retrieved, err := repo.GetSyncRunByID(ctx, run.ID)
		require.NoError(t, err)
		assert.Equal(t, BroadcastSyncRunStatusSuccess, retrieved.Status)
		assert.Equal(t, 2, retrieved.SuccessfulTargets)
		assert.NotNil(t, retrieved.EndedAt)
	})

	t.Run("update without ID", func(t *testing.T) {
		run := &BroadcastSyncRun{
			ExternalID: "SR-20260215-noid",
			StartedAt:  time.Now(),
			Status:     BroadcastSyncRunStatusRunning,
		}

		err := repo.UpdateSyncRun(ctx, run)
		assert.ErrorIs(t, err, ErrMissingSyncRunID)
	})
}

// TestBroadcastSyncRepo_GetSyncRun tests retrieving sync runs
func TestBroadcastSyncRepo_GetSyncRun(t *testing.T) {
	db := TestDB(t)

	repo := NewBroadcastSyncRepo(db)
	ctx := context.Background()

	// Create test run
	run := &BroadcastSyncRun{
		ExternalID:   "SR-20260215-get",
		StartedAt:    time.Now(),
		Status:       BroadcastSyncRunStatusSuccess,
		TotalTargets: 1,
	}
	err := repo.CreateSyncRun(ctx, run)
	require.NoError(t, err)

	t.Run("get by ID", func(t *testing.T) {
		retrieved, err := repo.GetSyncRunByID(ctx, run.ID)
		require.NoError(t, err)
		assert.Equal(t, run.ExternalID, retrieved.ExternalID)
		assert.Equal(t, run.Status, retrieved.Status)
	})

	t.Run("get by external ID", func(t *testing.T) {
		retrieved, err := repo.GetSyncRunByExternalID(ctx, run.ExternalID)
		require.NoError(t, err)
		assert.Equal(t, run.ID, retrieved.ID)
		assert.Equal(t, run.Status, retrieved.Status)
	})

	t.Run("get non-existent", func(t *testing.T) {
		_, err := repo.GetSyncRunByID(ctx, 99999)
		assert.Error(t, err)
	})
}

// TestBroadcastSyncRepo_ListRecentSyncRuns tests listing recent runs
func TestBroadcastSyncRepo_ListRecentSyncRuns(t *testing.T) {
	db := TestDB(t)

	repo := NewBroadcastSyncRepo(db)
	ctx := context.Background()

	// Create test runs at different times
	now := time.Now()
	runs := []BroadcastSyncRun{
		{
			ExternalID: "SR-20260215-001",
			StartedAt:  now.Add(-1 * time.Hour),
			Status:     BroadcastSyncRunStatusSuccess,
		},
		{
			ExternalID: "SR-20260215-002",
			StartedAt:  now.Add(-2 * time.Hour),
			Status:     BroadcastSyncRunStatusSuccess,
		},
		{
			ExternalID: "SR-20260215-003",
			StartedAt:  now.Add(-25 * time.Hour),
			Status:     BroadcastSyncRunStatusSuccess,
		},
	}

	for i := range runs {
		err := repo.CreateSyncRun(ctx, &runs[i])
		require.NoError(t, err)
	}

	t.Run("list last 24 hours", func(t *testing.T) {
		since := now.Add(-24 * time.Hour)
		retrieved, err := repo.ListRecentSyncRuns(ctx, since, 100)
		require.NoError(t, err)
		assert.Len(t, retrieved, 2) // Only the first two
	})

	t.Run("list last 48 hours", func(t *testing.T) {
		since := now.Add(-48 * time.Hour)
		retrieved, err := repo.ListRecentSyncRuns(ctx, since, 100)
		require.NoError(t, err)
		assert.Len(t, retrieved, 3) // All three
	})

	t.Run("with limit", func(t *testing.T) {
		since := now.Add(-48 * time.Hour)
		retrieved, err := repo.ListRecentSyncRuns(ctx, since, 2)
		require.NoError(t, err)
		assert.Len(t, retrieved, 2)
	})
}

// TestBroadcastSyncRepo_GetSummaryStats tests summary statistics
func TestBroadcastSyncRepo_GetSummaryStats(t *testing.T) {
	db := TestDB(t)

	repo := NewBroadcastSyncRepo(db)
	ctx := context.Background()

	t.Run("empty database", func(t *testing.T) {
		stats, err := repo.GetSyncRunSummaryStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.TotalRuns)
	})

	t.Run("with runs", func(t *testing.T) {
		now := time.Now()
		runs := []BroadcastSyncRun{
			{
				ExternalID: "SR-20260215-s1",
				StartedAt:  now,
				Status:     BroadcastSyncRunStatusSuccess,
				DurationMs: 5000,
			},
			{
				ExternalID: "SR-20260215-s2",
				StartedAt:  now,
				Status:     BroadcastSyncRunStatusSuccess,
				DurationMs: 3000,
			},
			{
				ExternalID: "SR-20260215-s3",
				StartedAt:  now,
				Status:     BroadcastSyncRunStatusFailed,
				DurationMs: 1000,
			},
		}

		for i := range runs {
			err := repo.CreateSyncRun(ctx, &runs[i])
			require.NoError(t, err)
		}

		stats, err := repo.GetSyncRunSummaryStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(3), stats.TotalRuns)
		assert.InDelta(t, 66.67, stats.SuccessRate, 0.1)  // 2/3 = 66.67%
		assert.Equal(t, int64(3000), stats.AvgDurationMs) // (5000+3000+1000)/3 = 3000
	})
}

// TestBroadcastSyncRepo_CreateTargetResult tests creating target results
func TestBroadcastSyncRepo_CreateTargetResult(t *testing.T) {
	db := TestDB(t)

	repo := NewBroadcastSyncRepo(db)
	ctx := context.Background()

	// Create parent run
	run := &BroadcastSyncRun{
		ExternalID: "SR-20260215-parent",
		StartedAt:  time.Now(),
		Status:     BroadcastSyncRunStatusRunning,
	}
	err := repo.CreateSyncRun(ctx, run)
	require.NoError(t, err)

	t.Run("create valid target result", func(t *testing.T) {
		result := &BroadcastSyncTargetResult{
			BroadcastSyncRunID: run.ID,
			TargetID:           1,
			RepoID:             1,
			StartedAt:          time.Now(),
			Status:             BroadcastSyncTargetStatusSuccess,
			FilesChanged:       5,
		}

		err := repo.CreateTargetResult(ctx, result)
		require.NoError(t, err)
		assert.NotZero(t, result.ID)
	})

	t.Run("missing run ID", func(t *testing.T) {
		result := &BroadcastSyncTargetResult{
			TargetID:  1,
			RepoID:    1,
			StartedAt: time.Now(),
			Status:    BroadcastSyncTargetStatusSuccess,
		}

		err := repo.CreateTargetResult(ctx, result)
		assert.ErrorIs(t, err, ErrMissingBroadcastSyncRunID)
	})

	t.Run("missing target ID", func(t *testing.T) {
		result := &BroadcastSyncTargetResult{
			BroadcastSyncRunID: run.ID,
			RepoID:             1,
			StartedAt:          time.Now(),
			Status:             BroadcastSyncTargetStatusSuccess,
		}

		err := repo.CreateTargetResult(ctx, result)
		assert.ErrorIs(t, err, ErrMissingTargetID)
	})

	t.Run("missing repo ID", func(t *testing.T) {
		result := &BroadcastSyncTargetResult{
			BroadcastSyncRunID: run.ID,
			TargetID:           1,
			StartedAt:          time.Now(),
			Status:             BroadcastSyncTargetStatusSuccess,
		}

		err := repo.CreateTargetResult(ctx, result)
		assert.ErrorIs(t, err, ErrMissingRepoID)
	})
}

// TestBroadcastSyncRepo_CreateFileChanges tests creating file changes
func TestBroadcastSyncRepo_CreateFileChanges(t *testing.T) {
	db := TestDB(t)

	repo := NewBroadcastSyncRepo(db)
	ctx := context.Background()

	// Create parent run and result
	run := &BroadcastSyncRun{
		ExternalID: "SR-20260215-files",
		StartedAt:  time.Now(),
		Status:     BroadcastSyncRunStatusRunning,
	}
	err := repo.CreateSyncRun(ctx, run)
	require.NoError(t, err)

	result := &BroadcastSyncTargetResult{
		BroadcastSyncRunID: run.ID,
		TargetID:           1,
		RepoID:             1,
		StartedAt:          time.Now(),
		Status:             BroadcastSyncTargetStatusSuccess,
	}
	err = repo.CreateTargetResult(ctx, result)
	require.NoError(t, err)

	t.Run("create single file change", func(t *testing.T) {
		change := &BroadcastSyncFileChange{
			BroadcastSyncTargetResultID: result.ID,
			FilePath:                    "README.md",
			ChangeType:                  BroadcastSyncFileChangeTypeModified,
			LinesAdded:                  10,
			LinesRemoved:                5,
		}

		err := repo.CreateFileChange(ctx, change)
		require.NoError(t, err)
		assert.NotZero(t, change.ID)
	})

	t.Run("create multiple file changes", func(t *testing.T) {
		changes := []BroadcastSyncFileChange{
			{
				BroadcastSyncTargetResultID: result.ID,
				FilePath:                    "file1.go",
				ChangeType:                  BroadcastSyncFileChangeTypeAdded,
				LinesAdded:                  100,
			},
			{
				BroadcastSyncTargetResultID: result.ID,
				FilePath:                    "file2.go",
				ChangeType:                  BroadcastSyncFileChangeTypeModified,
				LinesAdded:                  50,
				LinesRemoved:                20,
			},
		}

		err := repo.CreateFileChanges(ctx, changes)
		require.NoError(t, err)
	})

	t.Run("empty changes slice", func(t *testing.T) {
		err := repo.CreateFileChanges(ctx, []BroadcastSyncFileChange{})
		require.NoError(t, err) // Should not error
	})

	t.Run("missing target result ID", func(t *testing.T) {
		change := &BroadcastSyncFileChange{
			FilePath:   "test.go",
			ChangeType: BroadcastSyncFileChangeTypeAdded,
		}

		err := repo.CreateFileChange(ctx, change)
		assert.ErrorIs(t, err, ErrMissingBroadcastSyncTargetResultID)
	})

	t.Run("missing file path", func(t *testing.T) {
		change := &BroadcastSyncFileChange{
			BroadcastSyncTargetResultID: result.ID,
			ChangeType:                  BroadcastSyncFileChangeTypeAdded,
		}

		err := repo.CreateFileChange(ctx, change)
		assert.ErrorIs(t, err, ErrMissingFilePath)
	})
}

// TestBroadcastSyncRepo_GetRelatedData tests loading related data
func TestBroadcastSyncRepo_GetRelatedData(t *testing.T) {
	db := TestDB(t)

	repo := NewBroadcastSyncRepo(db)
	ctx := context.Background()

	// Create a complete hierarchy
	run := &BroadcastSyncRun{
		ExternalID: "SR-20260215-related",
		StartedAt:  time.Now(),
		Status:     BroadcastSyncRunStatusSuccess,
	}
	err := repo.CreateSyncRun(ctx, run)
	require.NoError(t, err)

	result := &BroadcastSyncTargetResult{
		BroadcastSyncRunID: run.ID,
		TargetID:           1,
		RepoID:             1,
		StartedAt:          time.Now(),
		Status:             BroadcastSyncTargetStatusSuccess,
	}
	err = repo.CreateTargetResult(ctx, result)
	require.NoError(t, err)

	changes := []BroadcastSyncFileChange{
		{
			BroadcastSyncTargetResultID: result.ID,
			FilePath:                    "file1.go",
			ChangeType:                  BroadcastSyncFileChangeTypeAdded,
		},
		{
			BroadcastSyncTargetResultID: result.ID,
			FilePath:                    "file2.go",
			ChangeType:                  BroadcastSyncFileChangeTypeModified,
		},
	}
	err = repo.CreateFileChanges(ctx, changes)
	require.NoError(t, err)

	t.Run("get run with preloaded data", func(t *testing.T) {
		retrieved, err := repo.GetSyncRunByID(ctx, run.ID)
		require.NoError(t, err)
		assert.Len(t, retrieved.TargetResults, 1)
		assert.Len(t, retrieved.TargetResults[0].FileChanges, 2)
	})

	t.Run("get target results by run ID", func(t *testing.T) {
		results, err := repo.GetTargetResultsByRunID(ctx, run.ID)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Len(t, results[0].FileChanges, 2)
	})

	t.Run("get file changes by target result ID", func(t *testing.T) {
		fileChanges, err := repo.GetFileChangesByTargetResultID(ctx, result.ID)
		require.NoError(t, err)
		assert.Len(t, fileChanges, 2)
	})
}
