package db

import (
	"time"
)

// =====================
// Analytics Models
// Reuses existing Organization and Repo models from models.go
// =====================

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
	PushedAt      *time.Time `json:"pushed_at,omitempty"`       // Last code push timestamp

	// Denormalized security alert counts (for fast dashboard queries)
	DependabotAlertCount     int `json:"dependabot_alert_count"`
	CodeScanningAlertCount   int `json:"code_scanning_alert_count"`
	SecretScanningAlertCount int `json:"secret_scanning_alert_count"`
}

// SecurityAlert tracks dependabot, code scanning, and secret scanning alerts.
// Collected via concurrent REST API calls (errgroup pattern).
type SecurityAlert struct {
	BaseModel

	RepositoryID    uint       `gorm:"index;not null" json:"repository_id"`
	AlertType       string     `gorm:"type:text;not null;index" json:"alert_type"`      // dependabot, code_scanning, secret_scanning
	AlertNumber     int        `gorm:"index" json:"alert_number"`                       // Alert number from GitHub API
	State           string     `gorm:"type:text;index" json:"state"`                    // open, fixed, dismissed
	Severity        string     `gorm:"type:text;index" json:"severity"`                 // critical, high, medium, low
	Summary         string     `gorm:"type:text" json:"summary"`                        // Short summary
	Description     string     `gorm:"type:text" json:"description"`                    // Full description
	HTMLURL         string     `gorm:"type:text" json:"html_url"`                       // Link to alert
	AlertCreatedAt  time.Time  `gorm:"column:alert_created_at" json:"alert_created_at"` // When GitHub alert was created (distinct from BaseModel.CreatedAt)
	FixedAt         *time.Time `json:"fixed_at,omitempty"`                              // When alert was fixed
	DismissedAt     *time.Time `json:"dismissed_at,omitempty"`                          // When alert was dismissed
	DismissedReason string     `gorm:"type:text" json:"dismissed_reason,omitempty"`     // Reason for dismissal

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

// CIMetricsSnapshot captures CI build metrics from GoFortress workflow artifacts.
// Replaces the fragile shell script approach in zai/scripts/scorecard-collect.sh.
type CIMetricsSnapshot struct {
	BaseModel

	RepositoryID  uint      `gorm:"index;not null" json:"repository_id"`
	SnapshotAt    time.Time `gorm:"index;not null" json:"snapshot_at"`
	WorkflowRunID int64     `gorm:"index" json:"workflow_run_id"`
	Branch        string    `gorm:"type:text" json:"branch"`
	CommitSHA     string    `gorm:"type:text" json:"commit_sha"`

	// LOC (from loc-stats JSON or statistics-section markdown fallback)
	GoFilesLOC     int `json:"go_files_loc"`
	TestFilesLOC   int `json:"test_files_loc"`
	GoFilesCount   int `json:"go_files_count"`
	TestFilesCount int `json:"test_files_count"`

	// Testing (from tests-section markdown + bench-stats JSON)
	TestCount       int      `json:"test_count"`
	BenchmarkCount  int      `json:"benchmark_count"`
	CoveragePercent *float64 `json:"coverage_percent"` // nullable — not all repos report
}
