package db

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AnalyticsRepo provides database operations for analytics data
type AnalyticsRepo interface {
	// Organizations
	UpsertOrganization(ctx context.Context, org *Organization) error
	GetOrganization(ctx context.Context, login string) (*Organization, error)
	ListOrganizations(ctx context.Context) ([]Organization, error)

	// Repositories
	UpsertRepository(ctx context.Context, repo *AnalyticsRepository) error
	GetRepository(ctx context.Context, fullName string) (*AnalyticsRepository, error)
	ListRepositories(ctx context.Context, orgLogin string) ([]AnalyticsRepository, error)

	// Snapshots
	CreateSnapshot(ctx context.Context, snap *RepositorySnapshot) error
	GetLatestSnapshot(ctx context.Context, repoID uint) (*RepositorySnapshot, error)
	GetSnapshotHistory(ctx context.Context, repoID uint, since time.Time) ([]RepositorySnapshot, error)

	// Alerts
	UpsertAlert(ctx context.Context, alert *SecurityAlert) error
	GetOpenAlerts(ctx context.Context, repoID uint, severity string) ([]SecurityAlert, error)
	GetAlertCounts(ctx context.Context, repoID uint) (map[string]int, error)

	// SyncRuns
	CreateSyncRun(ctx context.Context, run *SyncRun) error
	UpdateSyncRun(ctx context.Context, run *SyncRun) error
	GetLatestSyncRun(ctx context.Context) (*SyncRun, error)
}

// analyticsRepo implements AnalyticsRepo using GORM
type analyticsRepo struct {
	db *gorm.DB
}

// NewAnalyticsRepo creates a new analytics repository
func NewAnalyticsRepo(db *gorm.DB) AnalyticsRepo {
	return &analyticsRepo{db: db}
}

// ============================================================
// Organizations
// ============================================================

// UpsertOrganization creates or updates an organization
func (r *analyticsRepo) UpsertOrganization(ctx context.Context, org *Organization) error {
	// Try to find existing
	var existing Organization
	err := r.db.WithContext(ctx).
		Where("name = ?", org.Name).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Create new
		return r.db.WithContext(ctx).Create(org).Error
	}
	if err != nil {
		return err
	}

	// Update existing
	org.ID = existing.ID
	return r.db.WithContext(ctx).Save(org).Error
}

// GetOrganization retrieves an organization by login name
func (r *analyticsRepo) GetOrganization(ctx context.Context, login string) (*Organization, error) {
	var org Organization
	err := r.db.WithContext(ctx).
		Where("name = ?", login).
		First(&org).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// ListOrganizations retrieves all organizations
func (r *analyticsRepo) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var orgs []Organization
	err := r.db.WithContext(ctx).
		Order("name ASC").
		Find(&orgs).Error
	return orgs, err
}

// ============================================================
// Repositories
// ============================================================

// UpsertRepository creates or updates an analytics repository
func (r *analyticsRepo) UpsertRepository(ctx context.Context, repo *AnalyticsRepository) error {
	// Try to find existing
	var existing AnalyticsRepository
	err := r.db.WithContext(ctx).
		Where("full_name = ?", repo.FullName).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// Create new
		return r.db.WithContext(ctx).Create(repo).Error
	}
	if err != nil {
		return err
	}

	// Update existing
	repo.ID = existing.ID
	return r.db.WithContext(ctx).Save(repo).Error
}

// GetRepository retrieves a repository by full name (owner/name)
func (r *analyticsRepo) GetRepository(ctx context.Context, fullName string) (*AnalyticsRepository, error) {
	var repo AnalyticsRepository
	err := r.db.WithContext(ctx).
		Where("full_name = ?", fullName).
		Preload("Organization").
		First(&repo).Error
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

// ListRepositories retrieves all repositories for an organization
func (r *analyticsRepo) ListRepositories(ctx context.Context, orgLogin string) ([]AnalyticsRepository, error) {
	var repos []AnalyticsRepository

	query := r.db.WithContext(ctx).
		Joins("JOIN organizations ON organizations.id = analytics_repositories.organization_id").
		Where("organizations.name = ?", orgLogin).
		Order("analytics_repositories.name ASC")

	err := query.Find(&repos).Error
	return repos, err
}

// ============================================================
// Snapshots
// ============================================================

// CreateSnapshot creates a new repository snapshot
func (r *analyticsRepo) CreateSnapshot(ctx context.Context, snap *RepositorySnapshot) error {
	return r.db.WithContext(ctx).Create(snap).Error
}

// GetLatestSnapshot retrieves the most recent snapshot for a repository
func (r *analyticsRepo) GetLatestSnapshot(ctx context.Context, repoID uint) (*RepositorySnapshot, error) {
	var snap RepositorySnapshot
	err := r.db.WithContext(ctx).
		Where("repository_id = ?", repoID).
		Order("snapshot_at DESC").
		First(&snap).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No previous snapshot
		}
		return nil, err
	}
	return &snap, nil
}

// GetSnapshotHistory retrieves snapshots for a repository since a given time
func (r *analyticsRepo) GetSnapshotHistory(ctx context.Context, repoID uint, since time.Time) ([]RepositorySnapshot, error) {
	var snaps []RepositorySnapshot
	err := r.db.WithContext(ctx).
		Where("repository_id = ? AND snapshot_at >= ?", repoID, since).
		Order("snapshot_at DESC").
		Find(&snaps).Error
	return snaps, err
}

// ============================================================
// Alerts
// ============================================================

// UpsertAlert creates or updates a security alert
// Matches by repository_id, alert_type, and alert_number
func (r *analyticsRepo) UpsertAlert(ctx context.Context, alert *SecurityAlert) error {
	result := r.db.WithContext(ctx).
		Where("repository_id = ? AND alert_type = ? AND alert_number = ?",
			alert.RepositoryID, alert.AlertType, alert.AlertNumber).
		Assign(alert).
		FirstOrCreate(alert)
	return result.Error
}

// GetOpenAlerts retrieves open alerts for a repository, optionally filtered by severity
func (r *analyticsRepo) GetOpenAlerts(ctx context.Context, repoID uint, severity string) ([]SecurityAlert, error) {
	var alerts []SecurityAlert

	query := r.db.WithContext(ctx).
		Where("repository_id = ? AND state = ?", repoID, "open")

	if severity != "" {
		query = query.Where("severity = ?", severity)
	}

	err := query.
		Order("severity DESC, created_at DESC").
		Find(&alerts).Error

	return alerts, err
}

// GetAlertCounts retrieves alert counts by severity for a repository
func (r *analyticsRepo) GetAlertCounts(ctx context.Context, repoID uint) (map[string]int, error) {
	var results []struct {
		Severity string
		Count    int
	}

	err := r.db.WithContext(ctx).
		Model(&SecurityAlert{}).
		Select("severity, count(*) as count").
		Where("repository_id = ? AND state = ?", repoID, "open").
		Group("severity").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, r := range results {
		counts[r.Severity] = r.Count
	}

	return counts, nil
}

// ============================================================
// SyncRuns
// ============================================================

// CreateSyncRun creates a new sync run record
func (r *analyticsRepo) CreateSyncRun(ctx context.Context, run *SyncRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

// UpdateSyncRun updates an existing sync run
func (r *analyticsRepo) UpdateSyncRun(ctx context.Context, run *SyncRun) error {
	if run.ID == 0 {
		return fmt.Errorf("cannot update sync run: ID is 0")
	}
	return r.db.WithContext(ctx).Save(run).Error
}

// GetLatestSyncRun retrieves the most recent sync run
func (r *analyticsRepo) GetLatestSyncRun(ctx context.Context) (*SyncRun, error) {
	var run SyncRun
	err := r.db.WithContext(ctx).
		Order("started_at DESC").
		First(&run).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &run, nil
}
