package db

import (
	"time"
)

// =====================
// Analytics Models (4 new models)
// NOTE: Reuses existing Organization model from T-19 (models.go)
// =====================

// AnalyticsRepository tracks individual repo metrics (separate from config.Target)
// Links to existing Organization model from T-19 via OrganizationID.
// This is distinct from config Targets - analytics tracks ALL repos in an org,
// while Targets are specific repos selected for syncing.
type AnalyticsRepository struct {
	BaseModel

	OrganizationID uint       `gorm:"index" json:"organization_id"`                                        // FK to existing Organization table
	Owner          string     `gorm:"type:text;not null;index:idx_analytics_repo_owner_name" json:"owner"` // GitHub owner (org/user)
	Name           string     `gorm:"type:text;not null;index:idx_analytics_repo_owner_name" json:"name"`  // Repo name
	FullName       string     `gorm:"uniqueIndex;type:text;not null" json:"full_name"`                     // owner/name
	Description    string     `gorm:"type:text" json:"description"`                                        // Repo description
	DefaultBranch  string     `gorm:"type:text" json:"default_branch"`                                     // Default branch (main/master)
	Language       string     `gorm:"type:text" json:"language"`                                           // Primary language
	IsPrivate      bool       `json:"is_private"`                                                          // Visibility
	IsFork         bool       `json:"is_fork"`                                                             // Is this a fork?
	IsArchived     bool       `json:"is_archived"`                                                         // Is archived?
	URL            string     `gorm:"type:text" json:"url"`                                                // HTML URL
	MetadataETag   string     `gorm:"type:text" json:"metadata_etag"`                                      // ETag for conditional metadata requests
	SecurityETag   string     `gorm:"type:text" json:"security_etag"`                                      // ETag for conditional security requests
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`                                              // Last sync timestamp
	LastSyncRunID  *uint      `json:"last_sync_run_id,omitempty"`                                          // Links to the SyncRun that last processed this repo

	// Relationships
	Organization *Organization        `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Snapshots    []RepositorySnapshot `gorm:"foreignKey:RepositoryID" json:"snapshots,omitempty"`
	Alerts       []SecurityAlert      `gorm:"foreignKey:RepositoryID" json:"alerts,omitempty"`
}

// RepositorySnapshot captures point-in-time metrics for a repository.
// Uses timestamp (not date-only) to support future hourly snapshots.
type RepositorySnapshot struct {
	BaseModel

	RepositoryID uint      `gorm:"index;not null" json:"repository_id"`
	SnapshotAt   time.Time `gorm:"index;not null" json:"snapshot_at"` // Timestamp (not date-only) for future hourly support

	// Core metrics (from GraphQL batched queries)
	Stars       int `json:"stars"`        // Stargazers count
	Forks       int `json:"forks"`        // Forks count
	Watchers    int `json:"watchers"`     // Watchers count
	OpenIssues  int `json:"open_issues"`  // Open issues count
	OpenPRs     int `json:"open_prs"`     // Open pull requests count
	BranchCount int `json:"branch_count"` // Total branches

	// Release information
	LatestRelease   string     `gorm:"type:text" json:"latest_release,omitempty"` // Latest release tag name
	LatestReleaseAt *time.Time `json:"latest_release_at,omitempty"`               // Latest release published date
	LatestTag       string     `gorm:"type:text" json:"latest_tag,omitempty"`     // Latest tag name
	LatestTagAt     *time.Time `json:"latest_tag_at,omitempty"`                   // Latest tag date

	// Activity timestamps
	RepoUpdatedAt *time.Time `json:"repo_updated_at,omitempty"` // Last activity on repo

	// Raw data for future expansion (store full GraphQL response)
	RawData Metadata `gorm:"type:text" json:"raw_data,omitempty"`
}

// SecurityAlert tracks dependabot, code scanning, and secret scanning alerts.
// Collected via concurrent REST API calls (errgroup pattern).
type SecurityAlert struct {
	BaseModel

	RepositoryID    uint       `gorm:"index;not null" json:"repository_id"`
	AlertType       string     `gorm:"type:text;not null;index" json:"alert_type"`  // dependabot, code_scanning, secret_scanning
	AlertNumber     int        `gorm:"index" json:"alert_number"`                   // Alert number from GitHub API
	State           string     `gorm:"type:text;index" json:"state"`                // open, fixed, dismissed
	Severity        string     `gorm:"type:text;index" json:"severity"`             // critical, high, medium, low
	Summary         string     `gorm:"type:text" json:"summary"`                    // Short summary
	Description     string     `gorm:"type:text" json:"description"`                // Full description
	HTMLURL         string     `gorm:"type:text" json:"html_url"`                   // Link to alert
	CreatedAt       time.Time  `json:"created_at"`                                  // When alert was created
	FixedAt         *time.Time `json:"fixed_at,omitempty"`                          // When alert was fixed
	DismissedAt     *time.Time `json:"dismissed_at,omitempty"`                      // When alert was dismissed
	DismissedReason string     `gorm:"type:text" json:"dismissed_reason,omitempty"` // Reason for dismissal

	// Type-specific fields stored as JSON (CVSS score, CWE, package info, etc.)
	AlertData Metadata `gorm:"type:text" json:"alert_data,omitempty"`
}

// SyncRun tracks each analytics sync execution for observability and debugging.
// Provides detailed metrics about API usage, processing time, errors, etc.
type SyncRun struct {
	BaseModel

	StartedAt   time.Time  `gorm:"not null" json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Status      string     `gorm:"type:text;not null;index" json:"status"` // running, completed, failed, partial
	SyncType    string     `gorm:"type:text;not null" json:"sync_type"`    // full, security_only, metadata_only
	OrgFilter   string     `gorm:"type:text" json:"org_filter,omitempty"`  // empty = all orgs
	RepoFilter  string     `gorm:"type:text" json:"repo_filter,omitempty"` // empty = all repos

	// Processing counters
	ReposProcessed   int   `json:"repos_processed"`   // Total repos processed
	ReposSkipped     int   `json:"repos_skipped"`     // Skipped via change detection
	ReposFailed      int   `json:"repos_failed"`      // Failed to process
	SnapshotsCreated int   `json:"snapshots_created"` // New snapshots written
	AlertsUpserted   int   `json:"alerts_upserted"`   // Alerts created/updated
	APICallsMade     int   `json:"api_calls_made"`    // Total GitHub API calls
	DurationMs       int64 `json:"duration_ms"`       // Total execution time

	// Error tracking (JSON array of error details)
	Errors Metadata `gorm:"type:text" json:"errors,omitempty"`

	// Resume support (for future incremental syncs)
	LastProcessedRepo string `gorm:"type:text" json:"last_processed_repo,omitempty"` // For future resume support
}
