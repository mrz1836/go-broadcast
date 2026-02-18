package db

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// BroadcastSyncRepo provides database operations for broadcast sync tracking
type BroadcastSyncRepo interface {
	// Sync Runs
	CreateSyncRun(ctx context.Context, run *BroadcastSyncRun) error
	UpdateSyncRun(ctx context.Context, run *BroadcastSyncRun) error
	GetSyncRunByID(ctx context.Context, id uint) (*BroadcastSyncRun, error)
	GetSyncRunByExternalID(ctx context.Context, extID string) (*BroadcastSyncRun, error)
	ListRecentSyncRuns(ctx context.Context, since time.Time, limit int) ([]BroadcastSyncRun, error)
	ListSyncRunsByRepo(ctx context.Context, repoID uint, limit int) ([]BroadcastSyncRun, error)
	GetSyncRunSummaryStats(ctx context.Context) (*SyncRunSummaryStats, error)

	// Target Results
	CreateTargetResult(ctx context.Context, result *BroadcastSyncTargetResult) error
	UpdateTargetResult(ctx context.Context, result *BroadcastSyncTargetResult) error
	GetTargetResultsByRunID(ctx context.Context, runID uint) ([]BroadcastSyncTargetResult, error)

	// File Changes
	CreateFileChange(ctx context.Context, change *BroadcastSyncFileChange) error
	CreateFileChanges(ctx context.Context, changes []BroadcastSyncFileChange) error
	GetFileChangesByTargetResultID(ctx context.Context, targetResultID uint) ([]BroadcastSyncFileChange, error)
}

// SyncRunSummaryStats holds aggregate sync statistics
type SyncRunSummaryStats struct {
	TotalRuns            int64      `json:"total_runs"`
	SuccessRate          float64    `json:"success_rate_pct"`
	AvgDurationMs        int64      `json:"avg_duration_ms"`
	RunsThisWeek         int64      `json:"runs_this_week"`
	FilesChangedThisWeek int64      `json:"files_changed_this_week"`
	LastRunAt            *time.Time `json:"last_run_at,omitempty"`
}

// broadcastSyncRepo implements BroadcastSyncRepo using GORM
type broadcastSyncRepo struct {
	db *gorm.DB
}

// NewBroadcastSyncRepo creates a new broadcast sync repository
func NewBroadcastSyncRepo(db *gorm.DB) BroadcastSyncRepo {
	return &broadcastSyncRepo{db: db}
}

// ============================================================
// Sync Runs
// ============================================================

// CreateSyncRun creates a new sync run
func (r *broadcastSyncRepo) CreateSyncRun(ctx context.Context, run *BroadcastSyncRun) error {
	if run.ExternalID == "" {
		return ErrMissingExternalID
	}
	return r.db.WithContext(ctx).Create(run).Error
}

// UpdateSyncRun updates an existing sync run
func (r *broadcastSyncRepo) UpdateSyncRun(ctx context.Context, run *BroadcastSyncRun) error {
	if run.ID == 0 {
		return ErrMissingSyncRunID
	}
	return r.db.WithContext(ctx).Save(run).Error
}

// GetSyncRunByID retrieves a sync run by ID with all related data
func (r *broadcastSyncRepo) GetSyncRunByID(ctx context.Context, id uint) (*BroadcastSyncRun, error) {
	var run BroadcastSyncRun
	err := r.db.WithContext(ctx).
		Preload("TargetResults").
		Preload("TargetResults.FileChanges").
		First(&run, id).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// GetSyncRunByExternalID retrieves a sync run by external ID with related data
func (r *broadcastSyncRepo) GetSyncRunByExternalID(ctx context.Context, extID string) (*BroadcastSyncRun, error) {
	var run BroadcastSyncRun
	err := r.db.WithContext(ctx).
		Preload("TargetResults").
		Preload("TargetResults.FileChanges").
		Where("external_id = ?", extID).
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// ListRecentSyncRuns retrieves recent sync runs since the given time
func (r *broadcastSyncRepo) ListRecentSyncRuns(ctx context.Context, since time.Time, limit int) ([]BroadcastSyncRun, error) {
	if limit <= 0 {
		limit = 100 // default limit
	}

	var runs []BroadcastSyncRun
	err := r.db.WithContext(ctx).
		Where("started_at >= ?", since).
		Order("started_at DESC").
		Limit(limit).
		Find(&runs).Error
	return runs, err
}

// ListSyncRunsByRepo retrieves sync runs for a specific target repo
func (r *broadcastSyncRepo) ListSyncRunsByRepo(ctx context.Context, repoID uint, limit int) ([]BroadcastSyncRun, error) {
	if limit <= 0 {
		limit = 100 // default limit
	}

	var runs []BroadcastSyncRun
	err := r.db.WithContext(ctx).
		Joins("JOIN broadcast_sync_target_results ON broadcast_sync_target_results.broadcast_sync_run_id = broadcast_sync_runs.id").
		Where("broadcast_sync_target_results.repo_id = ?", repoID).
		Order("broadcast_sync_runs.started_at DESC").
		Limit(limit).
		Distinct().
		Find(&runs).Error
	return runs, err
}

// GetSyncRunSummaryStats retrieves aggregate statistics across all sync runs
func (r *broadcastSyncRepo) GetSyncRunSummaryStats(ctx context.Context) (*SyncRunSummaryStats, error) {
	stats := &SyncRunSummaryStats{}

	// Total runs
	if err := r.db.WithContext(ctx).
		Model(&BroadcastSyncRun{}).
		Count(&stats.TotalRuns).Error; err != nil {
		return nil, err
	}

	// Success rate (based on success + partial vs failed)
	var successCount int64
	if err := r.db.WithContext(ctx).
		Model(&BroadcastSyncRun{}).
		Where("status IN ?", []string{BroadcastSyncRunStatusSuccess, BroadcastSyncRunStatusPartial}).
		Count(&successCount).Error; err != nil {
		return nil, err
	}
	if stats.TotalRuns > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalRuns) * 100
	}

	// Average duration (only for completed runs)
	// Cast to INTEGER to avoid float64 scanning issues
	if err := r.db.WithContext(ctx).
		Model(&BroadcastSyncRun{}).
		Select("CAST(COALESCE(AVG(duration_ms), 0) AS INTEGER)").
		Where("status IN ?", []string{BroadcastSyncRunStatusSuccess, BroadcastSyncRunStatusPartial, BroadcastSyncRunStatusFailed}).
		Where("duration_ms > 0").
		Scan(&stats.AvgDurationMs).Error; err != nil {
		return nil, err
	}

	// This week stats (last 7 days)
	weekAgo := time.Now().AddDate(0, 0, -7)
	if err := r.db.WithContext(ctx).
		Model(&BroadcastSyncRun{}).
		Where("started_at >= ?", weekAgo).
		Count(&stats.RunsThisWeek).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Model(&BroadcastSyncRun{}).
		Select("COALESCE(SUM(total_files_changed), 0)").
		Where("started_at >= ?", weekAgo).
		Scan(&stats.FilesChangedThisWeek).Error; err != nil {
		return nil, err
	}

	// Last run timestamp
	var lastRun BroadcastSyncRun
	if err := r.db.WithContext(ctx).
		Model(&BroadcastSyncRun{}).
		Order("started_at DESC").
		First(&lastRun).Error; err == nil {
		stats.LastRunAt = &lastRun.StartedAt
	}
	// Ignore ErrRecordNotFound for LastRunAt

	return stats, nil
}

// ============================================================
// Target Results
// ============================================================

// CreateTargetResult creates a new target result
func (r *broadcastSyncRepo) CreateTargetResult(ctx context.Context, result *BroadcastSyncTargetResult) error {
	if result.BroadcastSyncRunID == 0 {
		return ErrMissingBroadcastSyncRunID
	}
	if result.TargetID == 0 {
		return ErrMissingTargetID
	}
	if result.RepoID == 0 {
		return ErrMissingRepoID
	}
	return r.db.WithContext(ctx).Create(result).Error
}

// UpdateTargetResult updates an existing target result
func (r *broadcastSyncRepo) UpdateTargetResult(ctx context.Context, result *BroadcastSyncTargetResult) error {
	if result.ID == 0 {
		return ErrMissingTargetResultID
	}
	return r.db.WithContext(ctx).Save(result).Error
}

// GetTargetResultsByRunID retrieves all target results for a specific sync run
func (r *broadcastSyncRepo) GetTargetResultsByRunID(ctx context.Context, runID uint) ([]BroadcastSyncTargetResult, error) {
	var results []BroadcastSyncTargetResult
	err := r.db.WithContext(ctx).
		Preload("FileChanges").
		Where("broadcast_sync_run_id = ?", runID).
		Order("started_at ASC").
		Find(&results).Error
	return results, err
}

// ============================================================
// File Changes
// ============================================================

// CreateFileChange creates a single file change record
func (r *broadcastSyncRepo) CreateFileChange(ctx context.Context, change *BroadcastSyncFileChange) error {
	if change.BroadcastSyncTargetResultID == 0 {
		return ErrMissingBroadcastSyncTargetResultID
	}
	if change.FilePath == "" {
		return ErrMissingFilePath
	}
	return r.db.WithContext(ctx).Create(change).Error
}

// CreateFileChanges batch creates multiple file change records
func (r *broadcastSyncRepo) CreateFileChanges(ctx context.Context, changes []BroadcastSyncFileChange) error {
	if len(changes) == 0 {
		return nil // nothing to do
	}

	// Validate all changes
	for _, change := range changes {
		if change.BroadcastSyncTargetResultID == 0 {
			return ErrMissingBroadcastSyncTargetResultID
		}
		if change.FilePath == "" {
			return ErrMissingFilePath
		}
	}

	// Batch insert with size 100
	return r.db.WithContext(ctx).CreateInBatches(changes, 100).Error
}

// GetFileChangesByTargetResultID retrieves all file changes for a specific target result
func (r *broadcastSyncRepo) GetFileChangesByTargetResultID(ctx context.Context, targetResultID uint) ([]BroadcastSyncFileChange, error) {
	var changes []BroadcastSyncFileChange
	err := r.db.WithContext(ctx).
		Where("broadcast_sync_target_result_id = ?", targetResultID).
		Order("position ASC").
		Find(&changes).Error
	return changes, err
}
