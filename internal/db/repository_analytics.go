package db

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	// analyticsDefaultClientName is the name of the auto-created client for analytics-discovered orgs
	analyticsDefaultClientName = "analytics"
)

// ErrInvalidSyncRunID is returned when attempting to update a sync run with ID 0
var ErrInvalidSyncRunID = errors.New("cannot update sync run with ID 0")

// AnalyticsRepo provides database operations for analytics data
type AnalyticsRepo interface {
	// Organizations
	UpsertOrganization(ctx context.Context, org *Organization) error
	GetOrganization(ctx context.Context, login string) (*Organization, error)
	ListOrganizations(ctx context.Context) ([]Organization, error)

	// Repositories (uses unified Repo model)
	UpsertRepository(ctx context.Context, repo *Repo) error
	GetRepository(ctx context.Context, fullName string) (*Repo, error)
	ListRepositories(ctx context.Context, orgLogin string) ([]Repo, error)

	// Snapshots
	CreateSnapshot(ctx context.Context, snap *RepositorySnapshot) error
	GetLatestSnapshot(ctx context.Context, repoID uint) (*RepositorySnapshot, error)
	GetSnapshotHistory(ctx context.Context, repoID uint, since time.Time) ([]RepositorySnapshot, error)
	UpdateSnapshotAlertCounts(ctx context.Context, snap *RepositorySnapshot) error

	// Alerts
	UpsertAlert(ctx context.Context, alert *SecurityAlert) error
	GetOpenAlerts(ctx context.Context, repoID uint, severity string) ([]SecurityAlert, error)
	GetAlertCounts(ctx context.Context, repoID uint) (map[string]int, error)
	GetAlertCountsByType(ctx context.Context, repoID uint) (map[string]int, error)

	// CI Snapshots
	CreateCISnapshot(ctx context.Context, snap *CIMetricsSnapshot) error
	GetLatestCISnapshot(ctx context.Context, repoID uint) (*CIMetricsSnapshot, error)

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

// ensureDefaultClient creates or retrieves the default "analytics" client for discovered orgs.
// Organizations require a ClientID FK; this provides one for orgs discovered via the GitHub API
// that don't have a pre-existing client from config import.
func (r *analyticsRepo) ensureDefaultClient(ctx context.Context) (uint, error) {
	var client Client
	err := r.db.WithContext(ctx).
		Where("name = ?", analyticsDefaultClientName).
		First(&client).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		client = Client{
			Name:        analyticsDefaultClientName,
			Description: "Auto-created client for analytics-discovered organizations",
		}
		if createErr := r.db.WithContext(ctx).Create(&client).Error; createErr != nil {
			return 0, createErr
		}
		return client.ID, nil
	}
	if err != nil {
		return 0, err
	}
	return client.ID, nil
}

// UpsertOrganization creates or updates an organization.
// If ClientID is 0, a default "analytics" client is auto-created and assigned.
func (r *analyticsRepo) UpsertOrganization(ctx context.Context, org *Organization) error {
	// Ensure ClientID is set for new orgs discovered via analytics
	if org.ClientID == 0 {
		clientID, err := r.ensureDefaultClient(ctx)
		if err != nil {
			return err
		}
		org.ClientID = clientID
	}

	// Try to find existing
	var existing Organization
	err := r.db.WithContext(ctx).
		Where("name = ?", org.Name).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
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

// UpsertRepository creates or updates a repository.
// Uses Unscoped to find soft-deleted records and restore them on upsert,
// avoiding UNIQUE constraint failures when a repo was previously soft-deleted.
func (r *analyticsRepo) UpsertRepository(ctx context.Context, repo *Repo) error {
	// Try to find existing (including soft-deleted rows to avoid unique constraint conflicts)
	var existing Repo
	err := r.db.WithContext(ctx).Unscoped().
		Where("full_name = ?", repo.FullNameStr).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new
		return r.db.WithContext(ctx).Create(repo).Error
	}
	if err != nil {
		return err
	}

	// Update existing (restore if soft-deleted)
	repo.ID = existing.ID
	if existing.DeletedAt.Valid {
		// Clear soft-delete: restore the record
		repo.DeletedAt = gorm.DeletedAt{Valid: false}
	}
	return r.db.WithContext(ctx).Unscoped().Save(repo).Error
}

// GetRepository retrieves a repository by full name (owner/name)
func (r *analyticsRepo) GetRepository(ctx context.Context, fullName string) (*Repo, error) {
	var repo Repo
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
func (r *analyticsRepo) ListRepositories(ctx context.Context, orgLogin string) ([]Repo, error) {
	var repos []Repo

	query := r.db.WithContext(ctx)

	if orgLogin != "" {
		query = query.
			Joins("JOIN organizations ON organizations.id = repos.organization_id").
			Where("organizations.name = ?", orgLogin)
	}

	err := query.
		Order("repos.name ASC").
		Find(&repos).Error
	return repos, err
}

// ============================================================
// Snapshots
// ============================================================

// CreateSnapshot creates a new repository snapshot
func (r *analyticsRepo) CreateSnapshot(ctx context.Context, snap *RepositorySnapshot) error {
	return r.db.WithContext(ctx).Create(snap).Error
}

// UpdateSnapshotAlertCounts updates only the alert count fields on an existing snapshot
func (r *analyticsRepo) UpdateSnapshotAlertCounts(ctx context.Context, snap *RepositorySnapshot) error {
	return r.db.WithContext(ctx).
		Model(snap).
		Updates(map[string]interface{}{
			"dependabot_alert_count":      snap.DependabotAlertCount,
			"code_scanning_alert_count":   snap.CodeScanningAlertCount,
			"secret_scanning_alert_count": snap.SecretScanningAlertCount,
		}).Error
}

// GetLatestSnapshot retrieves the most recent snapshot for a repository
// Returns gorm.ErrRecordNotFound if no snapshot exists
func (r *analyticsRepo) GetLatestSnapshot(ctx context.Context, repoID uint) (*RepositorySnapshot, error) {
	var snap RepositorySnapshot
	err := r.db.WithContext(ctx).
		Where("repository_id = ?", repoID).
		Order("snapshot_at DESC").
		First(&snap).Error
	if err != nil {
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
		Order("severity DESC, alert_created_at DESC").
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
		return ErrInvalidSyncRunID
	}
	return r.db.WithContext(ctx).Save(run).Error
}

// GetLatestSyncRun retrieves the most recent sync run
// Returns gorm.ErrRecordNotFound if no sync run exists
func (r *analyticsRepo) GetLatestSyncRun(ctx context.Context) (*SyncRun, error) {
	var run SyncRun
	err := r.db.WithContext(ctx).
		Order("started_at DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// GetAlertCountsByType retrieves open alert counts grouped by alert_type for a repository.
// Returns a map of alert_type ("dependabot", "code_scanning", "secret_scanning") to count.
func (r *analyticsRepo) GetAlertCountsByType(ctx context.Context, repoID uint) (map[string]int, error) {
	var results []struct {
		AlertType string
		Count     int
	}

	err := r.db.WithContext(ctx).
		Model(&SecurityAlert{}).
		Select("alert_type, count(*) as count").
		Where("repository_id = ? AND state = ?", repoID, "open").
		Group("alert_type").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, r := range results {
		counts[r.AlertType] = r.Count
	}

	return counts, nil
}

// ============================================================
// CI Snapshots
// ============================================================

// CreateCISnapshot creates a new CI metrics snapshot
func (r *analyticsRepo) CreateCISnapshot(ctx context.Context, snap *CIMetricsSnapshot) error {
	return r.db.WithContext(ctx).Create(snap).Error
}

// GetLatestCISnapshot retrieves the most recent CI snapshot for a repository
// Returns gorm.ErrRecordNotFound if no snapshot exists
func (r *analyticsRepo) GetLatestCISnapshot(ctx context.Context, repoID uint) (*CIMetricsSnapshot, error) {
	var snap CIMetricsSnapshot
	err := r.db.WithContext(ctx).
		Where("repository_id = ?", repoID).
		Order("snapshot_at DESC").
		First(&snap).Error
	if err != nil {
		return nil, err
	}
	return &snap, nil
}
