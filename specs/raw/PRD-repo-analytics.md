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
