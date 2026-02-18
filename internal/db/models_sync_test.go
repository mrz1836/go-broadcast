package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBroadcastSyncRunValidation tests sync run model validation
func TestBroadcastSyncRunValidation(t *testing.T) {
	t.Run("valid run", func(t *testing.T) {
		run := &BroadcastSyncRun{
			ExternalID:   "SR-20260215-abc123",
			StartedAt:    time.Now(),
			Status:       BroadcastSyncRunStatusRunning,
			Trigger:      BroadcastSyncRunTriggerManual,
			TotalTargets: 3,
		}

		assert.NotEmpty(t, run.ExternalID)
		assert.Equal(t, BroadcastSyncRunStatusRunning, run.Status)
		assert.Equal(t, 3, run.TotalTargets)
	})

	t.Run("status constants", func(t *testing.T) {
		statuses := []string{
			BroadcastSyncRunStatusPending,
			BroadcastSyncRunStatusRunning,
			BroadcastSyncRunStatusSuccess,
			BroadcastSyncRunStatusPartial,
			BroadcastSyncRunStatusFailed,
			BroadcastSyncRunStatusSkipped,
		}

		for _, status := range statuses {
			assert.NotEmpty(t, status, "Status constant should not be empty")
		}
	})

	t.Run("trigger constants", func(t *testing.T) {
		triggers := []string{
			BroadcastSyncRunTriggerManual,
			BroadcastSyncRunTriggerCron,
			BroadcastSyncRunTriggerCI,
		}

		for _, trigger := range triggers {
			assert.NotEmpty(t, trigger, "Trigger constant should not be empty")
		}
	})

	t.Run("computed duration", func(t *testing.T) {
		startedAt := time.Now()
		endedAt := startedAt.Add(5 * time.Second)

		run := &BroadcastSyncRun{
			ExternalID: "SR-20260215-test",
			StartedAt:  startedAt,
			EndedAt:    &endedAt,
			DurationMs: endedAt.Sub(startedAt).Milliseconds(),
		}

		assert.Equal(t, int64(5000), run.DurationMs)
	})
}

// TestBroadcastSyncTargetResultValidation tests target result model validation
func TestBroadcastSyncTargetResultValidation(t *testing.T) {
	t.Run("valid target result", func(t *testing.T) {
		result := &BroadcastSyncTargetResult{
			BroadcastSyncRunID: 1,
			TargetID:           2,
			RepoID:             3,
			StartedAt:          time.Now(),
			Status:             BroadcastSyncTargetStatusSuccess,
			FilesChanged:       5,
			LinesAdded:         100,
			LinesRemoved:       50,
		}

		assert.Equal(t, uint(1), result.BroadcastSyncRunID)
		assert.Equal(t, uint(2), result.TargetID)
		assert.Equal(t, uint(3), result.RepoID)
		assert.Equal(t, 5, result.FilesChanged)
	})

	t.Run("status constants", func(t *testing.T) {
		statuses := []string{
			BroadcastSyncTargetStatusPending,
			BroadcastSyncTargetStatusSuccess,
			BroadcastSyncTargetStatusFailed,
			BroadcastSyncTargetStatusSkipped,
			BroadcastSyncTargetStatusNoChanges,
		}

		for _, status := range statuses {
			assert.NotEmpty(t, status, "Status constant should not be empty")
		}
	})

	t.Run("with PR info", func(t *testing.T) {
		prNumber := 42
		result := &BroadcastSyncTargetResult{
			BroadcastSyncRunID: 1,
			TargetID:           2,
			RepoID:             3,
			StartedAt:          time.Now(),
			Status:             BroadcastSyncTargetStatusSuccess,
			PRNumber:           &prNumber,
			PRURL:              "https://github.com/org/repo/pull/42",
			PRState:            "open",
		}

		require.NotNil(t, result.PRNumber)
		assert.Equal(t, 42, *result.PRNumber)
		assert.Equal(t, "https://github.com/org/repo/pull/42", result.PRURL)
		assert.Equal(t, "open", result.PRState)
	})

	t.Run("with error details", func(t *testing.T) {
		result := &BroadcastSyncTargetResult{
			BroadcastSyncRunID: 1,
			TargetID:           2,
			RepoID:             3,
			StartedAt:          time.Now(),
			Status:             BroadcastSyncTargetStatusFailed,
			ErrorMessage:       "connection timeout",
			ErrorDetails: Metadata{
				"error_type": "timeout",
				"retryable":  true,
			},
		}

		assert.Equal(t, BroadcastSyncTargetStatusFailed, result.Status)
		assert.Equal(t, "connection timeout", result.ErrorMessage)
		assert.NotNil(t, result.ErrorDetails)
		assert.Equal(t, "timeout", result.ErrorDetails["error_type"])
	})
}

// TestBroadcastSyncFileChangeValidation tests file change model validation
func TestBroadcastSyncFileChangeValidation(t *testing.T) {
	t.Run("valid file change", func(t *testing.T) {
		change := &BroadcastSyncFileChange{
			BroadcastSyncTargetResultID: 1,
			FilePath:                    "README.md",
			ChangeType:                  BroadcastSyncFileChangeTypeModified,
			LinesAdded:                  10,
			LinesRemoved:                5,
			SizeBytes:                   1024,
			Position:                    0,
		}

		assert.Equal(t, uint(1), change.BroadcastSyncTargetResultID)
		assert.Equal(t, "README.md", change.FilePath)
		assert.Equal(t, BroadcastSyncFileChangeTypeModified, change.ChangeType)
	})

	t.Run("change type constants", func(t *testing.T) {
		changeTypes := []string{
			BroadcastSyncFileChangeTypeAdded,
			BroadcastSyncFileChangeTypeModified,
			BroadcastSyncFileChangeTypeDeleted,
		}

		for _, changeType := range changeTypes {
			assert.NotEmpty(t, changeType, "Change type constant should not be empty")
		}
	})

	t.Run("file added", func(t *testing.T) {
		change := &BroadcastSyncFileChange{
			BroadcastSyncTargetResultID: 1,
			FilePath:                    "new-file.go",
			ChangeType:                  BroadcastSyncFileChangeTypeAdded,
			LinesAdded:                  100,
			LinesRemoved:                0,
			SizeBytes:                   2048,
		}

		assert.Equal(t, BroadcastSyncFileChangeTypeAdded, change.ChangeType)
		assert.Equal(t, 100, change.LinesAdded)
		assert.Equal(t, 0, change.LinesRemoved)
	})

	t.Run("file deleted", func(t *testing.T) {
		change := &BroadcastSyncFileChange{
			BroadcastSyncTargetResultID: 1,
			FilePath:                    "old-file.go",
			ChangeType:                  BroadcastSyncFileChangeTypeDeleted,
			LinesAdded:                  0,
			LinesRemoved:                50,
			SizeBytes:                   0,
		}

		assert.Equal(t, BroadcastSyncFileChangeTypeDeleted, change.ChangeType)
		assert.Equal(t, 0, change.LinesAdded)
		assert.Equal(t, 50, change.LinesRemoved)
	})
}

// TestBroadcastSyncRunRelationships tests model relationships
func TestBroadcastSyncRunRelationships(t *testing.T) {
	t.Run("run with target results", func(t *testing.T) {
		run := &BroadcastSyncRun{
			ExternalID:   "SR-20260215-test",
			StartedAt:    time.Now(),
			Status:       BroadcastSyncRunStatusSuccess,
			TotalTargets: 2,
			TargetResults: []BroadcastSyncTargetResult{
				{
					TargetID:     1,
					RepoID:       1,
					StartedAt:    time.Now(),
					Status:       BroadcastSyncTargetStatusSuccess,
					FilesChanged: 3,
				},
				{
					TargetID:     2,
					RepoID:       2,
					StartedAt:    time.Now(),
					Status:       BroadcastSyncTargetStatusSuccess,
					FilesChanged: 5,
				},
			},
		}

		assert.Len(t, run.TargetResults, 2)
		assert.Equal(t, 2, run.TotalTargets)
	})

	t.Run("target result with file changes", func(t *testing.T) {
		result := &BroadcastSyncTargetResult{
			BroadcastSyncRunID: 1,
			TargetID:           1,
			RepoID:             1,
			StartedAt:          time.Now(),
			Status:             BroadcastSyncTargetStatusSuccess,
			FilesChanged:       2,
			FileChanges: []BroadcastSyncFileChange{
				{
					FilePath:   "file1.go",
					ChangeType: BroadcastSyncFileChangeTypeModified,
				},
				{
					FilePath:   "file2.go",
					ChangeType: BroadcastSyncFileChangeTypeAdded,
				},
			},
		}

		assert.Len(t, result.FileChanges, 2)
		assert.Equal(t, 2, result.FilesChanged)
	})
}
