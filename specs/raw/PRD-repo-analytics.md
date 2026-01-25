# PRD: Repository Analytics Engine

> **Codename:** `broadcast analytics` (subcommand of go-broadcast)
> **Status:** Draft
> **Author:** ZAI
> **Date:** 2026-01-25

---

## Executive Summary

Replace n8n-based repository analytics with a native Go implementation that provides:
- **Time-series statistics** (hourly/daily snapshots)
- **Comprehensive security monitoring** (Dependabot, code scanning, secret scanning)
- **Multi-org support** across personal and organizational repositories
- **GORM-powered storage** (SQLite for local, Postgres for production/Retool)
- **Smart caching** to minimize API calls and respect rate limits
- **Authenticated GitHub connections** for 5,000 req/hour vs 60 unauthenticated

---

## Problem Statement

The current n8n workflows (`Populate-Repos` and `Enrich-Repos-With-Security-Data`) work but have limitations:

1. **No historical data** — Only stores current state, overwrites previous values
2. **No time-series** — Can't track trends (stars over time, alert resolution rates)
3. **External dependency** — Requires n8n instance, Postgres connection, manual triggers
4. **Limited caching** — Re-fetches everything on each run
5. **No local option** — Always needs Postgres (Retool)
6. **Fragile orchestration** — Split across two workflows with timing dependencies

---

## Goals

| Goal | Description |
|------|-------------|
| **Historical tracking** | Store hourly/daily snapshots to track trends |
| **Comprehensive data** | Capture everything GitHub exposes via REST + GraphQL |
| **Security focus** | Track all security alerts with severity, state, resolution |
| **Portable storage** | SQLite for local/dev, Postgres for production |
| **Smart sync** | Incremental updates, ETag caching, conditional requests |
| **Rate limit aware** | Authenticated requests, exponential backoff, parallel batching |
| **CLI integration** | `broadcast analytics` subcommand with TUI dashboard option |

---

## Non-Goals

- Real-time webhooks (batch sync is sufficient)
- Public API server (CLI tool, not a service)
- Multi-tenant (single user/org focus)
- Web dashboard (use Retool or external tools)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      broadcast analytics                            │
├─────────────────────────────────────────────────────────────────────┤
│  CLI Layer (cobra)                                                  │
│  ├── analytics sync [--full|--incremental|--security-only]          │
│  ├── analytics status [repo]                                        │
│  ├── analytics history [repo] [--since] [--metric]                  │
│  ├── analytics alerts [--severity] [--state]                        │
│  ├── analytics export [--format json|csv]                           │
│  └── analytics dashboard (TUI)                                      │
├─────────────────────────────────────────────────────────────────────┤
│  Core Engine                                                        │
│  ├── sync/           — Orchestrates full/incremental syncs          │
│  ├── github/         — REST + GraphQL clients with auth             │
│  ├── cache/          — ETag cache, conditional requests             │
│  ├── models/         — GORM models for all entities                 │
│  └── metrics/        — Time-series aggregation                      │
├─────────────────────────────────────────────────────────────────────┤
│  Storage (GORM)                                                     │
│  ├── SQLite  — ~/.config/broadcast/analytics.db                     │
│  └── Postgres — configurable DSN for Retool/production              │
├─────────────────────────────────────────────────────────────────────┤
│  GitHub API                                                         │
│  ├── REST v3  — Repos, security alerts, rate limits                 │
│  ├── GraphQL v4 — Rich queries, nested data, efficient batching     │
│  └── Auth — PAT or GitHub App for 5,000 req/hr                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Data Model (GORM)

### Core Tables

```go
// Organization tracks GitHub orgs/users being monitored
type Organization struct {
    gorm.Model
    Login       string `gorm:"uniqueIndex;not null"`
    Type        string // "Organization" or "User"
    AvatarURL   string
    Description string
    Monitoring  bool   `gorm:"default:true"`

    Repositories []Repository
}

// Repository is the central entity
type Repository struct {
    gorm.Model
    GitHubID        int64  `gorm:"uniqueIndex;not null"`
    OrganizationID  uint   `gorm:"index"`
    Organization    Organization

    // Identifiers
    Owner           string `gorm:"index;not null"`
    Name            string `gorm:"not null"`
    FullName        string `gorm:"uniqueIndex;not null"` // owner/name

    // Metadata
    Description     string
    Language        string
    DefaultBranch   string
    Homepage        string
    License         string
    Topics          datatypes.JSON // []string

    // Flags
    IsPrivate       bool
    IsFork          bool
    IsArchived      bool
    IsDisabled      bool
    Monitoring      bool `gorm:"default:true;index"`

    // Timestamps (from GitHub)
    GitHubCreatedAt time.Time
    GitHubUpdatedAt time.Time
    GitHubPushedAt  time.Time

    // Parent fork info (if IsFork)
    ParentOwner     string
    ParentName      string

    // Sync metadata
    LastSyncedAt           time.Time `gorm:"index"`
    LastSecuritySyncAt     time.Time
    LastStatsSyncAt        time.Time
    RawJSON                datatypes.JSON // Full API response

    // Relations
    Snapshots        []RepositorySnapshot
    SecurityAlerts   []SecurityAlert
    Contributors     []Contributor `gorm:"many2many:repository_contributors;"`
}

// RepositorySnapshot stores point-in-time metrics
type RepositorySnapshot struct {
    gorm.Model
    RepositoryID    uint      `gorm:"index;not null"`
    Repository      Repository

    // Timestamp for this snapshot
    SnapshotAt      time.Time `gorm:"index;not null"`
    SnapshotType    string    `gorm:"index"` // "hourly" | "daily" | "manual"

    // Popularity metrics
    Stars           int
    Forks           int
    Watchers        int

    // Activity metrics
    OpenIssues      int
    OpenPRs         int
    Branches        int

    // Release info
    LatestReleaseTag    string
    LatestReleaseAt     *time.Time
    LatestTagName       string
    LatestTagAt         *time.Time

    // Calculated deltas (vs previous snapshot)
    StarsDelta      int
    ForksDelta      int
    IssuesDelta     int

    // Security summary at this point
    DependabotOpen      int
    DependabotCritical  int
    DependabotHigh      int
    CodeScanningOpen    int
    SecretScanningOpen  int
}

// SecurityAlert tracks individual security alerts
type SecurityAlert struct {
    gorm.Model
    RepositoryID    uint   `gorm:"index;not null"`
    Repository      Repository

    // Alert identification
    GitHubAlertID   int64  `gorm:"uniqueIndex:idx_repo_alert;not null"`
    AlertType       string `gorm:"index;not null"` // "dependabot" | "code_scanning" | "secret_scanning"

    // Common fields
    State           string `gorm:"index"` // "open" | "dismissed" | "fixed"
    Severity        string `gorm:"index"` // "critical" | "high" | "medium" | "low"
    HTMLURL         string

    // Timestamps
    GitHubCreatedAt time.Time
    GitHubUpdatedAt time.Time
    FixedAt         *time.Time
    DismissedAt     *time.Time
    DismissedBy     string
    DismissedReason string

    // Dependabot specific
    PackageName     string
    PackageEcosystem string // "npm" | "pip" | "go" | etc
    VulnerableRange string
    PatchedVersion  string
    CVEID           string
    CVSSScore       float64

    // Code scanning specific
    RuleName        string
    RuleDescription string
    RuleSeverity    string
    ToolName        string
    FilePath        string
    StartLine       int
    EndLine         int

    // Secret scanning specific
    SecretType      string // "github_token" | "aws_key" | etc
    SecretProvider  string
    IsPubliclyLeaked bool

    // Raw data
    RawJSON         datatypes.JSON
}

// Contributor tracks repo contributors
type Contributor struct {
    gorm.Model
    GitHubID        int64  `gorm:"uniqueIndex;not null"`
    Login           string `gorm:"uniqueIndex;not null"`
    AvatarURL       string
    Type            string // "User" | "Bot"

    Repositories    []Repository `gorm:"many2many:repository_contributors;"`
}

// RepositoryContributor is the join table with contribution count
type RepositoryContributor struct {
    RepositoryID    uint `gorm:"primaryKey"`
    ContributorID   uint `gorm:"primaryKey"`
    Contributions   int  // commit count
    LastContributedAt time.Time
}

// SyncLog tracks sync operations
type SyncLog struct {
    gorm.Model
    SyncType        string    // "full" | "incremental" | "security"
    StartedAt       time.Time
    CompletedAt     *time.Time
    Status          string    // "running" | "completed" | "failed"
    ReposProcessed  int
    ReposSkipped    int
    AlertsFound     int
    ErrorCount      int
    ErrorMessages   datatypes.JSON // []string
    Duration        time.Duration
}

// CacheEntry stores ETags and conditional request data
type CacheEntry struct {
    gorm.Model
    CacheKey        string `gorm:"uniqueIndex;not null"` // "repo:{owner}/{name}" or "org:{login}"
    ETag            string
    LastModified    string
    ExpiresAt       time.Time `gorm:"index"`
    ResponseHash    string    // SHA256 of response for change detection
}
```

### Indexes for Query Performance

```go
// Auto-migrate will create these, but explicit for documentation
db.Migrator().CreateIndex(&RepositorySnapshot{}, "idx_repo_snapshot_at")
db.Migrator().CreateIndex(&SecurityAlert{}, "idx_alert_repo_type_state")
```

---

## GitHub API Integration

### Authentication

```go
type GitHubClient struct {
    restClient    *github.Client      // google/go-github
    graphqlClient *githubv4.Client    // shurcooL/githubv4
    token         string
    rateLimiter   *rate.Limiter
    cache         *CacheStore
}

// NewClient creates authenticated client
// PAT gets 5,000 requests/hour vs 60 unauthenticated
func NewClient(token string) (*GitHubClient, error) {
    ctx := context.Background()
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    tc := oauth2.NewClient(ctx, ts)

    return &GitHubClient{
        restClient:    github.NewClient(tc),
        graphqlClient: githubv4.NewClient(tc),
        token:         token,
        rateLimiter:   rate.NewLimiter(rate.Every(720*time.Millisecond), 1), // ~5000/hr
        cache:         NewCacheStore(),
    }, nil
}
```

### GraphQL Query (Efficient Batching)

```graphql
query RepositoryData($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {
    id
    databaseId
    name
    nameWithOwner
    description
    createdAt
    updatedAt
    pushedAt
    url
    homepageUrl

    isPrivate
    isFork
    isArchived
    isDisabled

    primaryLanguage { name }
    licenseInfo { spdxId name }

    stargazerCount
    forkCount
    watchers { totalCount }

    issues(states: [OPEN]) { totalCount }
    pullRequests(states: [OPEN]) { totalCount }

    defaultBranchRef { name }
    refs(refPrefix: "refs/heads/") { totalCount }

    latestRelease {
      tagName
      name
      publishedAt
      isPrerelease
    }

    # Latest tag
    tags: refs(refPrefix: "refs/tags/", last: 1, orderBy: {field: TAG_COMMIT_DATE, direction: DESC}) {
      nodes {
        name
        target {
          ... on Commit { committedDate }
          ... on Tag {
            tagger { date }
            target { ... on Commit { committedDate } }
          }
        }
      }
    }

    # Repository topics
    repositoryTopics(first: 20) {
      nodes { topic { name } }
    }

    # Fork parent
    parent {
      owner { login }
      name
    }

    # Vulnerability alerts summary
    vulnerabilityAlerts(states: [OPEN], first: 100) {
      totalCount
      nodes {
        id
        createdAt
        dismissedAt
        securityVulnerability {
          severity
          package { name ecosystem }
          advisory { cvss { score } identifiers { type value } }
        }
      }
    }
  }

  # Rate limit info
  rateLimit {
    limit
    cost
    remaining
    resetAt
  }
}
```

### REST Endpoints for Security

```go
// Security alerts require REST API (not all available via GraphQL)
func (c *GitHubClient) GetDependabotAlerts(ctx context.Context, owner, repo string) ([]DependabotAlert, error) {
    // GET /repos/{owner}/{repo}/dependabot/alerts?state=open&per_page=100
}

func (c *GitHubClient) GetCodeScanningAlerts(ctx context.Context, owner, repo string) ([]CodeScanningAlert, error) {
    // GET /repos/{owner}/{repo}/code-scanning/alerts?state=open&per_page=100
}

func (c *GitHubClient) GetSecretScanningAlerts(ctx context.Context, owner, repo string) ([]SecretScanningAlert, error) {
    // GET /repos/{owner}/{repo}/secret-scanning/alerts?state=open&per_page=100
    // Also query with secret_type filter for generic patterns
}
```

---

## Caching Strategy

### ETag-Based Conditional Requests

```go
type CacheStore struct {
    db *gorm.DB
}

func (c *CacheStore) GetWithCache(ctx context.Context, key string, fetcher func() (*http.Response, error)) ([]byte, bool, error) {
    // 1. Check cache entry
    var entry CacheEntry
    if err := c.db.Where("cache_key = ?", key).First(&entry).Error; err == nil {
        // 2. Make conditional request with If-None-Match
        req.Header.Set("If-None-Match", entry.ETag)
    }

    resp, err := fetcher()
    if err != nil {
        return nil, false, err
    }

    // 3. Handle 304 Not Modified
    if resp.StatusCode == http.StatusNotModified {
        return nil, false, nil // No changes, use cached
    }

    // 4. Update cache with new ETag
    newEntry := CacheEntry{
        CacheKey:     key,
        ETag:         resp.Header.Get("ETag"),
        LastModified: resp.Header.Get("Last-Modified"),
        ExpiresAt:    time.Now().Add(1 * time.Hour),
    }
    c.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&newEntry)

    return body, true, nil // Changed, process new data
}
```

### Incremental Sync Logic

```go
func (s *Syncer) IncrementalSync(ctx context.Context) error {
    // 1. Get repos updated since last sync
    var repos []Repository
    s.db.Where("last_synced_at < github_updated_at OR last_synced_at IS NULL").
        Where("monitoring = ?", true).
        Find(&repos)

    // 2. For each repo, check if data changed (ETag)
    for _, repo := range repos {
        changed, data, err := s.cache.GetWithCache(ctx,
            fmt.Sprintf("repo:%s", repo.FullName),
            func() { return s.github.GetRepository(ctx, repo.Owner, repo.Name) },
        )

        if !changed {
            continue // Skip, no changes
        }

        // 3. Update repo and create snapshot
        s.updateRepository(repo, data)
        s.createSnapshot(repo, "incremental")
    }

    return nil
}
```

---

## Sync Modes

### Full Sync
```bash
broadcast analytics sync --full
```
- Fetches all repos from all monitored orgs
- Creates snapshots for all repos
- Fetches all security alerts
- Rebuilds contributor data
- Use for initial setup or recovery

### Incremental Sync
```bash
broadcast analytics sync
broadcast analytics sync --incremental
```
- Only fetches repos with `updated_at` > `last_synced_at`
- Uses ETag caching to skip unchanged repos
- Default mode for scheduled runs

### Security-Only Sync
```bash
broadcast analytics sync --security-only
```
- Only fetches security alerts for monitored repos
- Faster than full sync
- Good for security dashboard updates

### Scheduled Sync
```yaml
# config.yaml
analytics:
  schedule:
    snapshots: "0 * * * *"      # Hourly snapshots
    security: "0 */4 * * *"    # Security every 4 hours
    full: "0 0 * * 0"          # Weekly full sync
```

---

## CLI Commands

```bash
# Sync commands
broadcast analytics sync                    # Incremental sync
broadcast analytics sync --full             # Full sync
broadcast analytics sync --security-only    # Security alerts only
broadcast analytics sync --org bsv-blockchain  # Specific org only

# Status commands
broadcast analytics status                  # Summary of all repos
broadcast analytics status mrz1836/go-whatsonchain  # Specific repo
broadcast analytics status --alerts         # Show security alert summary

# History commands
broadcast analytics history mrz1836/go-whatsonchain --since 7d
broadcast analytics history mrz1836/go-whatsonchain --metric stars
broadcast analytics history --all --metric alerts --format csv

# Alert commands
broadcast analytics alerts                  # All open alerts
broadcast analytics alerts --severity critical,high
broadcast analytics alerts --type dependabot
broadcast analytics alerts --repo mrz1836/go-whatsonchain

# Export commands
broadcast analytics export --format json > analytics.json
broadcast analytics export --format csv --since 30d > monthly.csv
broadcast analytics export --to-postgres "postgres://..."  # Sync to Retool

# Dashboard (TUI)
broadcast analytics dashboard
```

---

## Configuration

```yaml
# ~/.config/broadcast/config.yaml (or broadcast.yaml in repo)
analytics:
  # Storage
  database:
    driver: sqlite                           # sqlite | postgres
    dsn: ~/.config/broadcast/analytics.db   # SQLite path
    # dsn: postgres://user:pass@host/db     # Postgres DSN

  # GitHub
  github:
    token: ${GITHUB_TOKEN}                   # From env var
    # Or: token_file: ~/.config/broadcast/github-token

  # Organizations to monitor
  organizations:
    - login: mrz1836
      type: user
      monitoring: true
    - login: BitcoinSchema
      type: org
      monitoring: true
    - login: bsv-blockchain
      type: org
      monitoring: true
    - login: bitcoin-sv
      type: org
      monitoring: true
    - login: skyetel
      type: org
      monitoring: true

  # Sync settings
  sync:
    snapshot_interval: hourly   # hourly | daily | manual
    security_interval: 4h       # How often to check security alerts
    batch_size: 10              # Repos to process in parallel
    rate_limit_buffer: 100      # Keep this many requests in reserve

  # Caching
  cache:
    enabled: true
    ttl: 1h                     # How long to trust cached data

  # Retention
  retention:
    hourly_snapshots: 7d        # Keep hourly for 7 days
    daily_snapshots: 90d        # Keep daily for 90 days
    security_history: 365d      # Keep security data for 1 year
```

---

## Data Collection Summary

### From Current n8n (Preserved)

| Data Point | Source | Notes |
|------------|--------|-------|
| repo_id, name, owner, url | REST/GraphQL | Unique identifier |
| created_at, updated_at | GraphQL | GitHub timestamps |
| raw_repository_record | REST | Full JSON blob |
| dependabot_alerts | REST | `/dependabot/alerts` |
| code_scanning_alerts | REST | `/code-scanning/alerts` |
| secret_scanning_alerts | REST | `/secret-scanning/alerts` |
| open_issues_count | GraphQL | Real-time count |
| open_pull_requests_count | GraphQL | Real-time count |
| stargazers_count, forks_count | GraphQL | Popularity |
| branches_count, default_branch | GraphQL | Branch info |
| latest_release_*, latest_tag_* | GraphQL | Release tracking |

### New Data (Enhanced)

| Data Point | Source | Value |
|------------|--------|-------|
| **Time-series snapshots** | Calculated | Track trends over time |
| watchers_count | GraphQL | Currently missing |
| topics | GraphQL | Repository tags |
| license | GraphQL | SPDX license ID |
| homepage_url | GraphQL | Project homepage |
| is_archived, is_disabled | GraphQL | Repo status flags |
| pushed_at | GraphQL | Last push timestamp |
| parent (fork source) | GraphQL | Track fork origins |
| contributors | REST | Contributor list with commit counts |
| languages | REST | Language breakdown |
| commit_activity | REST | Weekly commit stats |
| **Alert severity breakdown** | Calculated | critical/high/medium/low counts |
| **Alert resolution metrics** | Calculated | Time to fix, fix rate |
| **Delta calculations** | Calculated | Stars gained, issues opened |

### Security Data (Comprehensive)

```
Dependabot Alerts:
├── package, ecosystem, vulnerable_versions
├── severity (critical/high/medium/low)
├── CVSS score
├── CVE ID
├── patched_version
├── state, dismissed_reason, fixed_at

Code Scanning Alerts:
├── rule_id, rule_name, rule_description
├── severity (error/warning/note)
├── tool_name (CodeQL, etc.)
├── file_path, start_line, end_line
├── state, dismissed_reason

Secret Scanning Alerts:
├── secret_type (github_token, aws_key, etc.)
├── secret_provider
├── push_protection_bypassed
├── validity (active/inactive/unknown)
├── state, resolution
```

---

## Go-Broadcast Patterns (Consistency)

The analytics module MUST follow existing go-broadcast patterns for consistency:

### Package Structure

```
internal/
├── analytics/           # NEW: Analytics engine
│   ├── client.go        # GitHub API client (REST + GraphQL)
│   ├── client_test.go
│   ├── models.go        # GORM models
│   ├── sync.go          # Sync orchestrator
│   ├── sync_test.go
│   ├── progress.go      # Progress reporting (like directory_progress.go)
│   ├── cache.go         # ETag caching
│   ├── retry.go         # Backoff/retry logic
│   └── export.go        # CSV/JSON/Postgres export
├── cli/
│   ├── analytics.go     # NEW: analytics subcommand
│   └── analytics_test.go
└── logging/             # EXISTING: Reuse logging patterns
```

### Logging Standards

Use `internal/logging` patterns with `logrus` and `StandardFields`:

```go
import (
    "github.com/sirupsen/logrus"
    "github.com/mrz1836/go-broadcast/internal/logging"
)

// Define component name
const componentAnalytics = "analytics"

// Use structured logging with standard fields
func (s *Syncer) syncRepository(ctx context.Context, repo Repository) error {
    log := logging.WithStandardFields(s.logger, s.logConfig, componentAnalytics)

    log.WithFields(logrus.Fields{
        logging.StandardFields.RepoName:   repo.FullName,
        logging.StandardFields.Operation:  "sync",
        logging.StandardFields.Phase:      "start",
    }).Info("Syncing repository")

    start := time.Now()
    // ... sync logic ...

    log.WithFields(logrus.Fields{
        logging.StandardFields.RepoName:   repo.FullName,
        logging.StandardFields.Operation:  "sync",
        logging.StandardFields.Phase:      "complete",
        logging.StandardFields.DurationMs: time.Since(start).Milliseconds(),
    }).Info("Repository synced successfully")

    return nil
}
```

### Interfaces (Testability)

Follow the interface pattern from `cli/sync.go`:

```go
// GitHubClient interface for mocking in tests
type GitHubClient interface {
    GetRepository(ctx context.Context, owner, name string) (*Repository, error)
    GetOrganizationRepos(ctx context.Context, org string) ([]Repository, error)
    GetDependabotAlerts(ctx context.Context, owner, repo string) ([]DependabotAlert, error)
    GetCodeScanningAlerts(ctx context.Context, owner, repo string) ([]CodeScanningAlert, error)
    GetSecretScanningAlerts(ctx context.Context, owner, repo string) ([]SecretScanningAlert, error)
}

// Storage interface for database operations
type Storage interface {
    SaveRepository(ctx context.Context, repo *Repository) error
    SaveSnapshot(ctx context.Context, snapshot *RepositorySnapshot) error
    SaveAlerts(ctx context.Context, repoID uint, alerts []SecurityAlert) error
    GetRepositories(ctx context.Context, filter RepositoryFilter) ([]Repository, error)
}

// AnalyticsService is the main service interface
type AnalyticsService interface {
    Sync(ctx context.Context, opts SyncOptions) error
    GetStatus(ctx context.Context, repoName string) (*RepositoryStatus, error)
    GetHistory(ctx context.Context, repoName string, since time.Time) ([]RepositorySnapshot, error)
    GetAlerts(ctx context.Context, filter AlertFilter) ([]SecurityAlert, error)
}
```

---

## Resilience & Error Handling

### Exponential Backoff with Jitter

```go
package analytics

import (
    "context"
    "math/rand"
    "time"

    "github.com/sirupsen/logrus"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
    MaxRetries     int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    Multiplier     float64
    JitterFactor   float64 // 0.0 to 1.0
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() RetryConfig {
    return RetryConfig{
        MaxRetries:     5,
        InitialBackoff: 1 * time.Second,
        MaxBackoff:     60 * time.Second,
        Multiplier:     2.0,
        JitterFactor:   0.3, // ±30% jitter
    }
}

// RetryableFunc is a function that can be retried
type RetryableFunc func(ctx context.Context) error

// WithRetry executes a function with exponential backoff
func WithRetry(ctx context.Context, log *logrus.Entry, cfg RetryConfig, operation string, fn RetryableFunc) error {
    var lastErr error
    backoff := cfg.InitialBackoff

    for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
        if attempt > 0 {
            // Add jitter: backoff * (1 ± jitterFactor)
            jitter := 1.0 + (rand.Float64()*2-1)*cfg.JitterFactor
            sleepTime := time.Duration(float64(backoff) * jitter)

            log.WithFields(logrus.Fields{
                "attempt":    attempt,
                "max":        cfg.MaxRetries,
                "backoff_ms": sleepTime.Milliseconds(),
                "operation":  operation,
            }).Warn("Retrying after backoff")

            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(sleepTime):
            }

            // Increase backoff for next attempt
            backoff = time.Duration(float64(backoff) * cfg.Multiplier)
            if backoff > cfg.MaxBackoff {
                backoff = cfg.MaxBackoff
            }
        }

        err := fn(ctx)
        if err == nil {
            if attempt > 0 {
                log.WithFields(logrus.Fields{
                    "attempt":   attempt,
                    "operation": operation,
                }).Info("Retry succeeded")
            }
            return nil
        }

        lastErr = err

        // Check if error is retryable
        if !isRetryableError(err) {
            log.WithFields(logrus.Fields{
                "error":     err.Error(),
                "operation": operation,
            }).Error("Non-retryable error")
            return err
        }

        log.WithFields(logrus.Fields{
            "attempt":   attempt,
            "error":     err.Error(),
            "operation": operation,
        }).Warn("Retryable error occurred")
    }

    return fmt.Errorf("max retries exceeded for %s: %w", operation, lastErr)
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
    // Rate limit errors
    if strings.Contains(err.Error(), "rate limit") {
        return true
    }
    // Temporary network errors
    if strings.Contains(err.Error(), "timeout") ||
       strings.Contains(err.Error(), "connection refused") ||
       strings.Contains(err.Error(), "EOF") {
        return true
    }
    // GitHub 5xx errors
    if strings.Contains(err.Error(), "502") ||
       strings.Contains(err.Error(), "503") ||
       strings.Contains(err.Error(), "504") {
        return true
    }
    return false
}
```

### Rate Limit Handling

```go
// RateLimitHandler manages GitHub rate limits
type RateLimitHandler struct {
    remaining  int
    resetAt    time.Time
    buffer     int // Keep this many in reserve
    mu         sync.RWMutex
    log        *logrus.Entry
}

// WaitIfNeeded blocks if rate limit is near exhaustion
func (r *RateLimitHandler) WaitIfNeeded(ctx context.Context) error {
    r.mu.RLock()
    remaining := r.remaining
    resetAt := r.resetAt
    r.mu.RUnlock()

    if remaining > r.buffer {
        return nil // Plenty of requests remaining
    }

    sleepTime := time.Until(resetAt)
    if sleepTime <= 0 {
        return nil // Reset already happened
    }

    r.log.WithFields(logrus.Fields{
        "remaining":   remaining,
        "buffer":      r.buffer,
        "reset_in":    sleepTime.Round(time.Second).String(),
    }).Warn("Rate limit buffer reached, waiting for reset")

    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(sleepTime):
        return nil
    }
}

// UpdateFromResponse updates rate limit info from API response
func (r *RateLimitHandler) UpdateFromResponse(resp *http.Response) {
    r.mu.Lock()
    defer r.mu.Unlock()

    if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
        r.remaining, _ = strconv.Atoi(remaining)
    }
    if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
        resetUnix, _ := strconv.ParseInt(reset, 10, 64)
        r.resetAt = time.Unix(resetUnix, 0)
    }

    r.log.WithFields(logrus.Fields{
        "remaining": r.remaining,
        "reset_at":  r.resetAt.Format(time.RFC3339),
    }).Debug("Rate limit updated")
}
```

---

## User Experience & Progress Logging

### Progress Reporter (Following directory_progress.go Pattern)

```go
// SyncProgressReporter provides real-time feedback during sync
type SyncProgressReporter struct {
    logger         *logrus.Entry
    updateInterval time.Duration
    lastUpdate     time.Time
    mu             sync.RWMutex

    // Metrics
    metrics SyncMetrics
}

// SyncMetrics tracks sync progress
type SyncMetrics struct {
    // Discovery
    OrgsDiscovered   int
    ReposDiscovered  int
    ReposMonitored   int

    // Progress
    ReposProcessed   int
    ReposSkipped     int // Unchanged (ETag match)
    ReposErrored     int

    // Data collected
    SnapshotsCreated int
    AlertsFound      int
    AlertsCritical   int
    AlertsHigh       int

    // Timing
    StartTime        time.Time
    LastRepoTime     time.Duration
    TotalAPITime     time.Duration
    TotalDBTime      time.Duration
}

// LogProgress outputs current progress (rate-limited to avoid spam)
func (p *SyncProgressReporter) LogProgress(force bool) {
    p.mu.Lock()
    defer p.mu.Unlock()

    now := time.Now()
    if !force && now.Sub(p.lastUpdate) < p.updateInterval {
        return
    }
    p.lastUpdate = now

    elapsed := time.Since(p.metrics.StartTime)
    remaining := p.metrics.ReposMonitored - p.metrics.ReposProcessed - p.metrics.ReposSkipped

    p.logger.WithFields(logrus.Fields{
        "progress":    fmt.Sprintf("%d/%d", p.metrics.ReposProcessed+p.metrics.ReposSkipped, p.metrics.ReposMonitored),
        "processed":   p.metrics.ReposProcessed,
        "skipped":     p.metrics.ReposSkipped,
        "remaining":   remaining,
        "elapsed":     elapsed.Round(time.Second).String(),
        "alerts":      p.metrics.AlertsFound,
        "critical":    p.metrics.AlertsCritical,
    }).Info("📊 Sync progress")
}

// LogRepoStart logs when starting to process a repo
func (p *SyncProgressReporter) LogRepoStart(repo string) {
    p.logger.WithFields(logrus.Fields{
        "repo":     repo,
        "progress": fmt.Sprintf("%d/%d", p.metrics.ReposProcessed+1, p.metrics.ReposMonitored),
    }).Info("🔄 Processing repository")
}

// LogRepoComplete logs when a repo is done
func (p *SyncProgressReporter) LogRepoComplete(repo string, stats RepoSyncStats) {
    p.mu.Lock()
    p.metrics.ReposProcessed++
    p.metrics.SnapshotsCreated++
    p.metrics.AlertsFound += stats.AlertsFound
    p.metrics.AlertsCritical += stats.AlertsCritical
    p.metrics.AlertsHigh += stats.AlertsHigh
    p.metrics.LastRepoTime = stats.Duration
    p.mu.Unlock()

    fields := logrus.Fields{
        "repo":        repo,
        "duration_ms": stats.Duration.Milliseconds(),
        "stars":       stats.Stars,
        "forks":       stats.Forks,
    }

    // Only add alert info if there are alerts
    if stats.AlertsFound > 0 {
        fields["alerts"] = stats.AlertsFound
        if stats.AlertsCritical > 0 {
            fields["critical"] = stats.AlertsCritical
        }
    }

    p.logger.WithFields(fields).Info("✅ Repository synced")
}

// LogRepoSkipped logs when a repo is skipped (no changes)
func (p *SyncProgressReporter) LogRepoSkipped(repo string, reason string) {
    p.mu.Lock()
    p.metrics.ReposSkipped++
    p.mu.Unlock()

    p.logger.WithFields(logrus.Fields{
        "repo":   repo,
        "reason": reason,
    }).Debug("⏭️  Repository skipped (no changes)")
}

// LogRepoError logs when a repo fails
func (p *SyncProgressReporter) LogRepoError(repo string, err error) {
    p.mu.Lock()
    p.metrics.ReposErrored++
    p.mu.Unlock()

    p.logger.WithFields(logrus.Fields{
        "repo":  repo,
        "error": err.Error(),
    }).Error("❌ Repository sync failed")
}

// LogSummary outputs final summary
func (p *SyncProgressReporter) LogSummary() {
    p.mu.RLock()
    m := p.metrics
    p.mu.RUnlock()

    elapsed := time.Since(m.StartTime)

    p.logger.WithFields(logrus.Fields{
        "duration":     elapsed.Round(time.Second).String(),
        "repos_synced": m.ReposProcessed,
        "repos_skipped": m.ReposSkipped,
        "repos_errored": m.ReposErrored,
        "snapshots":    m.SnapshotsCreated,
        "alerts_total": m.AlertsFound,
        "alerts_critical": m.AlertsCritical,
        "alerts_high":  m.AlertsHigh,
    }).Info("🎉 Sync complete")

    // Print human-readable summary
    fmt.Println()
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("📊 ANALYTICS SYNC SUMMARY")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Printf("   Duration:        %s\n", elapsed.Round(time.Second))
    fmt.Printf("   Repos synced:    %d\n", m.ReposProcessed)
    fmt.Printf("   Repos skipped:   %d (unchanged)\n", m.ReposSkipped)
    if m.ReposErrored > 0 {
        fmt.Printf("   Repos errored:   %d ⚠️\n", m.ReposErrored)
    }
    fmt.Printf("   Snapshots:       %d\n", m.SnapshotsCreated)
    fmt.Println()
    if m.AlertsFound > 0 {
        fmt.Println("🔒 SECURITY ALERTS")
        fmt.Printf("   Total open:      %d\n", m.AlertsFound)
        if m.AlertsCritical > 0 {
            fmt.Printf("   Critical:        %d 🔴\n", m.AlertsCritical)
        }
        if m.AlertsHigh > 0 {
            fmt.Printf("   High:            %d 🟠\n", m.AlertsHigh)
        }
    } else {
        fmt.Println("🔒 No security alerts found ✅")
    }
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
```

### CLI Output Examples

**Normal sync:**
```
$ broadcast analytics sync

🚀 Starting analytics sync...
   Organizations: mrz1836, BitcoinSchema, bsv-blockchain, bitcoin-sv, skyetel
   Mode: incremental

🔄 Processing repository mrz1836/go-whatsonchain [1/87]
✅ Repository synced                    repo=mrz1836/go-whatsonchain duration_ms=234 stars=15 forks=8
🔄 Processing repository mrz1836/go-broadcast [2/87]
✅ Repository synced                    repo=mrz1836/go-broadcast duration_ms=312 stars=42 alerts=2
⏭️  Repository skipped (no changes)     repo=mrz1836/go-api-router
📊 Sync progress                        progress=25/87 elapsed=1m12s alerts=5 critical=1

... (continues) ...

🎉 Sync complete                        duration=4m32s repos_synced=72 repos_skipped=15 alerts_total=23

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 ANALYTICS SYNC SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   Duration:        4m32s
   Repos synced:    72
   Repos skipped:   15 (unchanged)
   Snapshots:       72

🔒 SECURITY ALERTS
   Total open:      23
   Critical:        3 🔴
   High:            7 🟠
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**With --verbose flag:**
```
$ broadcast analytics sync --verbose

🚀 Starting analytics sync...
   Config: ~/.config/broadcast/config.yaml
   Database: ~/.config/broadcast/analytics.db (SQLite)
   Organizations: 5
   Repositories: 87 monitored

📡 Checking rate limit...               remaining=4823 reset_in=48m

🔄 Processing repository mrz1836/go-whatsonchain [1/87]
   → Fetching via GraphQL...            cost=1
   → Stars: 15 (+0), Forks: 8 (+0), Issues: 3
   → Fetching Dependabot alerts...      found=0
   → Fetching code scanning alerts...   found=0
   → Creating snapshot...               id=1234
✅ Repository synced                    duration_ms=234

🔄 Processing repository mrz1836/go-broadcast [2/87]
   → Fetching via GraphQL...            cost=1
   → Stars: 42 (+2), Forks: 12 (+1), Issues: 5
   → Fetching Dependabot alerts...      found=2
     └─ critical: lodash <4.17.21 (CVE-2021-23337)
     └─ high: axios <0.21.1 (CVE-2020-28168)
   → Fetching code scanning alerts...   found=0
   → Creating snapshot...               id=1235
✅ Repository synced                    duration_ms=312 alerts=2
```

**Rate limit handling:**
```
$ broadcast analytics sync

🔄 Processing repository bitcoin-sv/go-sdk [45/87]
⚠️  Rate limit buffer reached           remaining=95 buffer=100 reset_in=12m34s
   Waiting for rate limit reset...
   (Press Ctrl+C to abort)
📡 Rate limit reset                     remaining=5000
🔄 Resuming sync...
```

**Error with retry:**
```
🔄 Processing repository skyetel/internal-api [67/87]
⚠️  Retryable error occurred            attempt=1 error="502 Bad Gateway"
   Retrying after backoff               attempt=2 backoff_ms=1200
⚠️  Retryable error occurred            attempt=2 error="502 Bad Gateway"
   Retrying after backoff               attempt=3 backoff_ms=2600
✅ Retry succeeded                      attempt=3
✅ Repository synced                    duration_ms=3842
```

---

## Rate Limit Management

### GitHub Rate Limits

| Type | Limit | Reset |
|------|-------|-------|
| REST (authenticated) | 5,000/hour | Rolling |
| REST (unauthenticated) | 60/hour | Rolling |
| GraphQL | 5,000 points/hour | Rolling |
| Search | 30/minute | Rolling |

### Strategy

```go
type RateLimiter struct {
    remaining  int
    resetAt    time.Time
    buffer     int // Keep this many in reserve
    mu         sync.Mutex
}

func (r *RateLimiter) Wait(ctx context.Context, cost int) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    // If we're under buffer, wait for reset
    if r.remaining - cost < r.buffer {
        sleepTime := time.Until(r.resetAt)
        if sleepTime > 0 {
            log.Printf("Rate limit buffer reached, sleeping %v", sleepTime)
            time.Sleep(sleepTime)
        }
    }

    r.remaining -= cost
    return nil
}

func (r *RateLimiter) Update(resp *http.Response) {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.remaining, _ = strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
    resetUnix, _ := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64)
    r.resetAt = time.Unix(resetUnix, 0)
}
```

---

## Roadmap

### Phase 1: Core Engine (MVP)
- [ ] GORM models + migrations (SQLite)
- [ ] GitHub REST client with auth
- [ ] Basic sync: fetch repos, store snapshots
- [ ] CLI: `sync`, `status`, `list`

### Phase 2: Security
- [ ] Dependabot alerts integration
- [ ] Code scanning alerts
- [ ] Secret scanning alerts
- [ ] Alert severity tracking

### Phase 3: GraphQL + Caching
- [ ] GraphQL client for rich queries
- [ ] ETag caching layer
- [ ] Incremental sync logic
- [ ] Rate limit management

### Phase 4: Time-Series
- [ ] Hourly/daily snapshot scheduling
- [ ] Delta calculations
- [ ] History queries
- [ ] Retention policies

### Phase 5: Production
- [ ] Postgres support
- [ ] Export to Retool (Postgres sync)
- [ ] TUI dashboard
- [ ] CSV/JSON export

### Phase 6: Advanced
- [ ] Contributor tracking
- [ ] Language breakdown
- [ ] Commit activity trends
- [ ] Alert resolution metrics

---

## Success Metrics

| Metric | Target |
|--------|--------|
| API efficiency | < 100 requests per full sync (with caching) |
| Sync time | < 5 minutes for 100 repos |
| Data freshness | Hourly snapshots, 4-hour security |
| Storage efficiency | < 100MB for 1 year of data |
| Query performance | < 100ms for dashboard queries |

---

## References

- [GitHub REST API v3](https://docs.github.com/en/rest)
- [GitHub GraphQL API v4](https://docs.github.com/en/graphql)
- [go-github](https://github.com/google/go-github) — REST client
- [githubv4](https://github.com/shurcooL/githubv4) — GraphQL client
- [GORM](https://gorm.io/) — ORM for Go
- Current n8n workflows: `./Populate-Repos.json`, `./Enrich-Repos-With-Security-Data.json`

---

## Appendix: Current n8n Schema (for migration)

```sql
-- Current table structure in Retool Postgres
CREATE TABLE broadcast_repositories (
    id SERIAL PRIMARY KEY,
    repo_id BIGINT UNIQUE NOT NULL,
    repo_name TEXT NOT NULL,
    repo_owner TEXT NOT NULL,
    repo_url TEXT NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP,
    monitoring BOOLEAN DEFAULT true,
    raw_repository_record JSONB,
    notes TEXT,
    last_synced_at TIMESTAMP,
    dependabot_alerts JSONB,
    code_scanning_alerts JSONB,
    secret_scanning_alerts JSONB,
    security_alerts_last_synced_at TIMESTAMP,
    maintainer_internal TEXT,
    maintainer_external TEXT
);
```

The new schema is a superset — all current data can be migrated and expanded.
