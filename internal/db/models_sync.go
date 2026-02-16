package db

import (
	"time"
)

// BroadcastSyncRun represents a single broadcast sync invocation
// (distinct from analytics SyncRun which tracks GitHub data collection)
type BroadcastSyncRun struct {
	BaseModel

	// User-facing ID (SR-{YYYYMMDD}-{random6})
	ExternalID string `gorm:"uniqueIndex;type:text;not null" json:"external_id"`

	// Link to group being synced (nullable for full sync)
	GroupID *uint `gorm:"index" json:"group_id,omitempty"`

	// Source repo at time of sync
	SourceRepoID *uint  `gorm:"index" json:"source_repo_id,omitempty"`
	SourceBranch string `gorm:"type:text" json:"source_branch"`
	SourceCommit string `gorm:"type:text" json:"source_commit"` // SHA at sync time

	// Timing
	StartedAt  time.Time  `gorm:"not null" json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	DurationMs int64      `json:"duration_ms"`

	// Status: pending, running, success, partial, failed, skipped
	Status string `gorm:"type:text;not null;default:'pending';index" json:"status"`

	// Trigger: manual, cron, ci
	Trigger string `gorm:"type:text;default:'manual'" json:"trigger"`

	// Options used for this run (JSON)
	Options Metadata `gorm:"type:text" json:"options,omitempty"`

	// Aggregate stats
	TotalTargets      int `gorm:"default:0" json:"total_targets"`
	SuccessfulTargets int `gorm:"default:0" json:"successful_targets"`
	FailedTargets     int `gorm:"default:0" json:"failed_targets"`
	SkippedTargets    int `gorm:"default:0" json:"skipped_targets"`

	// Total file stats (aggregated from target results)
	TotalFilesChanged int `gorm:"default:0" json:"total_files_changed"`
	TotalLinesAdded   int `gorm:"default:0" json:"total_lines_added"`
	TotalLinesRemoved int `gorm:"default:0" json:"total_lines_removed"`

	// Error summary (if failed)
	ErrorSummary string `gorm:"type:text" json:"error_summary,omitempty"`

	// Relationships
	TargetResults []BroadcastSyncTargetResult `gorm:"foreignKey:BroadcastSyncRunID" json:"target_results,omitempty"`
}

// BroadcastSyncRunStatus constants
const (
	BroadcastSyncRunStatusPending = "pending"
	BroadcastSyncRunStatusRunning = "running"
	BroadcastSyncRunStatusSuccess = "success"
	BroadcastSyncRunStatusPartial = "partial" // Some targets succeeded, some failed
	BroadcastSyncRunStatusFailed  = "failed"
	BroadcastSyncRunStatusSkipped = "skipped"
)

// BroadcastSyncRunTrigger constants
const (
	BroadcastSyncRunTriggerManual = "manual"
	BroadcastSyncRunTriggerCron   = "cron"
	BroadcastSyncRunTriggerCI     = "ci"
)

// BroadcastSyncTargetResult represents the result of syncing to a single target repo
type BroadcastSyncTargetResult struct {
	BaseModel

	// Parent run
	BroadcastSyncRunID uint `gorm:"index;not null" json:"broadcast_sync_run_id"`

	// Target being synced
	TargetID uint `gorm:"index;not null" json:"target_id"`
	RepoID   uint `gorm:"index;not null" json:"repo_id"`

	// Timing
	StartedAt  time.Time  `gorm:"not null" json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	DurationMs int64      `json:"duration_ms"`

	// Status: success, failed, skipped, no_changes
	Status string `gorm:"type:text;not null;default:'pending';index" json:"status"`

	// Branch info
	BranchName string `gorm:"type:text" json:"branch_name"`

	// Commit SHA of source at time of sync
	SourceCommitSHA string `gorm:"type:text" json:"source_commit_sha"`

	// File stats
	FilesProcessed int `gorm:"default:0" json:"files_processed"`
	FilesChanged   int `gorm:"default:0" json:"files_changed"`
	FilesDeleted   int `gorm:"default:0" json:"files_deleted"`

	// Diff stats (aggregated from file changes)
	LinesAdded   int   `gorm:"default:0" json:"lines_added"`
	LinesRemoved int   `gorm:"default:0" json:"lines_removed"`
	BytesChanged int64 `gorm:"default:0" json:"bytes_changed"`

	// PR info (nullable - only set if PR created)
	PRNumber *int   `json:"pr_number,omitempty"`
	PRURL    string `gorm:"type:text" json:"pr_url,omitempty"`
	PRState  string `gorm:"type:text" json:"pr_state,omitempty"` // open, merged, closed

	// Error info (if failed)
	ErrorMessage string   `gorm:"type:text" json:"error_message,omitempty"`
	ErrorDetails Metadata `gorm:"type:text" json:"error_details,omitempty"`

	// Relationships
	FileChanges []BroadcastSyncFileChange `gorm:"foreignKey:BroadcastSyncTargetResultID" json:"file_changes,omitempty"`
}

// BroadcastSyncTargetStatus constants
const (
	BroadcastSyncTargetStatusPending   = "pending"
	BroadcastSyncTargetStatusSuccess   = "success"
	BroadcastSyncTargetStatusFailed    = "failed"
	BroadcastSyncTargetStatusSkipped   = "skipped"
	BroadcastSyncTargetStatusNoChanges = "no_changes"
)

// BroadcastSyncFileChange represents a single file change within a target result
type BroadcastSyncFileChange struct {
	BaseModel

	// Parent target result
	BroadcastSyncTargetResultID uint `gorm:"index;not null" json:"broadcast_sync_target_result_id"`

	// File info
	FilePath   string `gorm:"type:text;not null;index" json:"file_path"`
	SourcePath string `gorm:"type:text" json:"source_path,omitempty"` // If different from dest

	// Change type: added, modified, deleted
	ChangeType string `gorm:"type:text;not null" json:"change_type"`

	// Diff stats
	LinesAdded   int   `gorm:"default:0" json:"lines_added"`
	LinesRemoved int   `gorm:"default:0" json:"lines_removed"`
	SizeBytes    int64 `gorm:"default:0" json:"size_bytes"`

	// Ordering
	Position int `gorm:"default:0" json:"position"`
}

// BroadcastSyncFileChangeType constants
const (
	BroadcastSyncFileChangeTypeAdded    = "added"
	BroadcastSyncFileChangeTypeModified = "modified"
	BroadcastSyncFileChangeTypeDeleted  = "deleted"
)
