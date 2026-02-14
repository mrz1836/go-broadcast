<div align="center">

# 🗄️ Database Backend

**Structured Configuration Storage for go-broadcast**

[![SQLite](https://img.shields.io/badge/SQLite-3-003B57?logo=sqlite&logoColor=white)](https://sqlite.org/)
[![GORM v2](https://img.shields.io/badge/GORM-v2-00ADD8?logo=go&logoColor=white)](https://gorm.io/)
[![Pure Go](https://img.shields.io/badge/Pure%20Go-No%20CGO-00ADD8?logo=go&logoColor=white)](https://go.dev/)

</div>

<br>

## 📑 Navigation

<table align="center">
  <tr>
    <td align="center" width="33%">
       🚀&nbsp;<a href="#-quick-start"><code>Quick&nbsp;Start</code></a>
    </td>
    <td align="center" width="33%">
       📊&nbsp;<a href="#-schema-reference"><code>Schema&nbsp;Reference</code></a>
    </td>
    <td align="center" width="33%">
       🔄&nbsp;<a href="#-importexport"><code>Import/Export</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
       ⌨️&nbsp;<a href="#-cli-commands"><code>CLI&nbsp;Commands</code></a>
    </td>
    <td align="center">
       🤖&nbsp;<a href="#crud-commands"><code>CRUD&nbsp;Commands</code></a>
    </td>
    <td align="center">
       🔍&nbsp;<a href="#-queries"><code>Queries</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
       🧪&nbsp;<a href="#-testing"><code>Testing</code></a>
    </td>
    <td align="center">
       ⚡&nbsp;<a href="#-performance"><code>Performance</code></a>
    </td>
    <td align="center">
       🏗️&nbsp;<a href="#-architecture"><code>Architecture</code></a>
    </td>
  </tr>
  <tr>
    <td align="center">
       ⚙️&nbsp;<a href="#-configuration"><code>Configuration</code></a>
    </td>
    <td align="center">
    </td>
    <td align="center">
    </td>
  </tr>
</table>

<br>

## 🚀 Quick Start

### Initialize Database

```bash
# Create a new database
go-broadcast db init

# Check database status
go-broadcast db status
```

### Import Existing YAML Configuration

```bash
# Import your existing sync.yaml
go-broadcast db import sync.yaml
```

### Query Configuration

```bash
# Which repos sync this file?
go-broadcast db query --file .github/workflows/ci.yml

# What files sync to this repo?
go-broadcast db query --repo mrz1836/my-repo

# Which targets use this file list?
go-broadcast db query --file-list ai-files
```

### Export to YAML

```bash
# Export entire configuration
go-broadcast db export --output sync-exported.yaml

# Export single group
go-broadcast db export --group mrz-tools --output group.yaml

# Export to stdout
go-broadcast db export --stdout
```

### Use Database Configuration

The `--from-db` flag works with all configuration-based commands:

```bash
# Sync from database
go-broadcast sync --from-db
go-broadcast sync --from-db --groups "core,security"
go-broadcast sync --from-db --dry-run

# Check status from database
go-broadcast status --from-db
go-broadcast status --from-db --json

# Validate database configuration
go-broadcast validate --from-db

# Cancel operations using database configuration
go-broadcast cancel --from-db --groups "bitcoin-schema"

# List modules from database
go-broadcast modules list --from-db
```

<br>

## 📊 Schema Reference

### Entity Relationship Diagram

```
    ┌──────────────┐
    │   clients    │
    └──────┬───────┘
           │ 1:N
           ▼
    ┌──────────────┐
    │organizations │
    └──────┬───────┘
           │ 1:N
           ▼
    ┌──────────────┐
    │    repos     │◄──────────────────────────────┐
    └──────────────┘                               │
           ▲                                       │
           │ repo_id FK                            │ repo_id FK
           │                                       │
           │ ┌──────────────┐
           │ │   configs    │
           │ └──────┬───────┘
           │        │
           ├────────┴───────────┐
           │                    │
           ▼                    ▼
    ┌──────────────┐       ┌──────────────┐
    │    groups    │       │  file_lists  │
    └──────┬───────┘       └──────┬───────┘
           │                      │
           ├────┐                 │
           │    │                 ▼
           ▼    ▼         ┌─────────────────┐
    ┌─────────┐ │         │ file_mappings   │
    │ sources │─┘         │  (polymorphic)  │
    └─────────┘           └─────────────────┘
           │
           ▼
    ┌──────────────┐
    │   targets    │◄────────┐
    └──────┬───────┘         │
           │                 │
           ├─────────────────┤
           │                 │
           ▼                 │
    ┌──────────────┐         │
    │  transforms  │         │
    │ (polymorphic)│         │
    └──────────────┘         │
                             │
    ┌────────────────────────┤
    │ target_file_list_refs  │
    └────────────────────────┘
```

### Analytics Entity Relationship Diagram

```
    ┌──────────────┐
    │organizations │
    └──────┬───────┘
           │ 1:N
           ▼
    ┌──────────────────────┐
    │analytics_repositories│
    └──────┬───────────────┘
           │
           ├── 1:N ──►  ┌──────────────────────┐
           │            │ repository_snapshots │
           │            └──────────────────────┘
           │
           ├── 1:N ──►  ┌─────────────────┐
           │            │ security_alerts │
           │            └─────────────────┘
           │
           └── 1:N ──►  ┌──────────────────────┐
                        │ ci_metrics_snapshots │
                        └──────────────────────┘

    ┌──────────────┐
    │  sync_runs   │  (standalone)
    └──────────────┘
```

<details>
<summary><b>clients</b> — Top-level owner entities</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `name` | text | Client name (unique) |
| `description` | text | Client description |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Has many `organizations`

</details>

<details>
<summary><b>organizations</b> — GitHub organizations</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `client_id` | uint | Foreign key to `clients` |
| `name` | text | Organization name (unique, e.g. "mrz1836") |
| `description` | text | Organization description |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Belongs to `clients`
- Has many `repos`

</details>

<details>
<summary><b>repos</b> — Repository records</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `organization_id` | uint | Foreign key to `organizations` |
| `name` | text | Short repo name (e.g. "go-broadcast") |
| `description` | text | Repository description |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Belongs to `organizations`

**Indexes:**
- Unique composite: `(organization_id, name)`

</details>

<details>
<summary><b>configs</b> — Root configuration container</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `external_id` | text | User-facing ID (unique) |
| `name` | text | Configuration name |
| `version` | int | Schema version |
| `metadata` | text | JSON metadata for extensibility |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Has many `groups`
- Has many `file_lists`
- Has many `directory_lists`

</details>

<details>
<summary><b>groups</b> — Sync group definitions</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `config_id` | uint | Foreign key to `configs` |
| `external_id` | text | User-facing ID (unique) |
| `name` | text | Group name |
| `description` | text | Group description |
| `priority` | int | Execution priority |
| `enabled` | *bool | Enable/disable group |
| `position` | int | Ordering for export |
| `metadata` | text | JSON metadata |

**Relations:**
- Belongs to `configs`
- Has one `sources` (source repository)
- Has one `group_globals` (global settings)
- Has one `group_defaults` (default PR settings)
- Has many `targets` (target repositories)
- Has many `group_dependencies` (dependency graph)

</details>

<details>
<summary><b>group_globals</b> — Group-level global PR settings</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `group_id` | uint | Foreign key to `groups` (1:1, unique) |
| `pr_labels` | text | JSON array of PR labels |
| `pr_assignees` | text | JSON array of assignees |
| `pr_reviewers` | text | JSON array of reviewers |
| `pr_team_reviewers` | text | JSON array of team reviewers |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Belongs to `groups` (1:1)

**Purpose:**
- Stores global PR settings that apply to all targets in a group
- Can be overridden by target-specific PR settings

</details>

<details>
<summary><b>group_defaults</b> — Group-level default settings</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `group_id` | uint | Foreign key to `groups` (1:1, unique) |
| `branch_prefix` | text | Default branch prefix for PRs |
| `pr_labels` | text | JSON array of default PR labels |
| `pr_assignees` | text | JSON array of default assignees |
| `pr_reviewers` | text | JSON array of default reviewers |
| `pr_team_reviewers` | text | JSON array of default team reviewers |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Belongs to `groups` (1:1)

**Purpose:**
- Provides default values for targets that don't specify their own settings

</details>

<details>
<summary><b>group_dependencies</b> — Group dependency graph</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `group_id` | uint | Foreign key to `groups` |
| `depends_on_id` | text | External ID of dependency group |
| `position` | int | Ordering for dependency resolution |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |

**Relations:**
- Belongs to `groups`

**Purpose:**
- Tracks which groups depend on other groups
- Used for execution ordering and circular dependency detection

**Indexes:**
- `group_id` for fast dependency lookups

</details>

<details>
<summary><b>sources</b> — Source repository configuration</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `group_id` | uint | Foreign key to `groups` (1:1) |
| `repo_id` | uint | Foreign key to `repos` |
| `branch` | text | Branch name |
| `blob_size_limit` | text | Max blob size |
| `security_email` | text | Security contact |
| `support_email` | text | Support contact |
| `metadata` | text | JSON metadata |

**Relations:**
- Belongs to `groups` (1:1)
- Belongs to `repos` (via `repo_id`)

**Validation:**
- `repo_id`: Must reference a valid repo
- `branch`: Must be valid branch name
- Emails: Must be valid email addresses

</details>

<details>
<summary><b>targets</b> — Target repository configuration</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `group_id` | uint | Foreign key to `groups` |
| `repo_id` | uint | Foreign key to `repos` |
| `branch` | text | Branch name |
| `blob_size_limit` | text | Max blob size limit |
| `security_email` | text | Security contact email |
| `support_email` | text | Support contact email |
| `pr_labels` | text | JSON array of PR labels |
| `pr_assignees` | text | JSON array of assignees |
| `pr_reviewers` | text | JSON array of reviewers |
| `pr_team_reviewers` | text | JSON array of team reviewers |
| `position` | int | Ordering for export |
| `metadata` | text | JSON metadata |

**Relations:**
- Belongs to `groups`
- Belongs to `repos` (via `repo_id`)
- Has many `file_mappings` (polymorphic)
- Has many `directory_mappings` (polymorphic)
- Has zero or one `transforms` (polymorphic)
- Many-to-many with `file_lists` via `target_file_list_refs`
- Many-to-many with `directory_lists` via `target_directory_list_refs`

</details>

<details>
<summary><b>file_lists</b> — Reusable file mapping lists</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `config_id` | uint | Foreign key to `configs` |
| `external_id` | text | User-facing ID (unique) |
| `name` | text | List name |
| `description` | text | List description |
| `position` | int | Ordering for export |
| `metadata` | text | JSON metadata |

**Relations:**
- Belongs to `configs`
- Has many `file_mappings` (polymorphic)
- Many-to-many with `targets` via `target_file_list_refs`

</details>

<details>
<summary><b>file_mappings</b> — File source/destination mappings (polymorphic)</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `owner_type` | text | "target" or "file_list" |
| `owner_id` | uint | Foreign key to owner |
| `src` | text | Source file path |
| `dest` | text | Destination file path |
| `delete_flag` | bool | Delete this file from target |
| `position` | int | Ordering for export |
| `metadata` | text | JSON metadata |

**Validation:**
- `src`: Required unless `delete_flag=true`
- `dest`: Always required
- Paths: No directory traversal (`../`)

**Indexes:**
- Composite: `(owner_type, owner_id)` for polymorphic lookups
- `dest` for query-by-file

</details>

<details>
<summary><b>directory_mappings</b> — Directory sync mappings (polymorphic)</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `owner_type` | text | "target" or "directory_list" |
| `owner_id` | uint | Foreign key to owner |
| `src` | text | Source directory path |
| `dest` | text | Destination directory path |
| `exclude` | text | JSON array of exclusion patterns |
| `include_only` | text | JSON array of inclusion patterns |
| `preserve_structure` | *bool | Preserve directory structure |
| `include_hidden` | *bool | Include hidden files |
| `delete_flag` | bool | Delete this directory from target |
| `module_config` | text | JSON module configuration |
| `position` | int | Ordering for export |
| `metadata` | text | JSON metadata |

**Relations:**
- Polymorphic: Belongs to `targets` or `directory_lists`
- Has zero or one `transforms` (polymorphic)

</details>

<details>
<summary><b>transforms</b> — Template variable transformations (polymorphic)</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `owner_type` | text | "target" or "directory_mapping" |
| `owner_id` | uint | Foreign key to owner |
| `repo_name` | bool | Enable repo name variable |
| `variables` | text | JSON map of custom variables |
| `metadata` | text | JSON metadata |

**Variables:**
- `{{ repo_name }}`: Target repository name
- Custom variables from `variables` JSON map

</details>

<details>
<summary><b>target_file_list_refs</b> — Many-to-many join table</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `target_id` | uint | Foreign key to `targets` |
| `file_list_id` | uint | Foreign key to `file_lists` |
| `position` | int | Ordering for export |
| `metadata` | text | JSON metadata |

**Indexes:**
- Unique composite: `(target_id, file_list_id)`

</details>

<details>
<summary><b>target_directory_list_refs</b> — Many-to-many join table for directory lists</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `target_id` | uint | Foreign key to `targets` |
| `directory_list_id` | uint | Foreign key to `directory_lists` |
| `position` | int | Ordering for export |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |

**Relations:**
- Belongs to `targets`
- Belongs to `directory_lists`

**Indexes:**
- Unique composite: `(target_id, directory_list_id)`

</details>

<details>
<summary><b>schema_migrations</b> — Schema version tracking</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `version` | text | Schema version string (unique) |
| `applied_at` | datetime | When migration was applied |
| `description` | text | Human-readable description |
| `checksum` | text | Integrity verification hash |

**Purpose:**
- Tracks applied schema migrations
- Enables safe schema evolution

**Indexes:**
- Unique index on `version`

</details>

<details>
<summary><b>analytics_repositories</b> — Repository records for analytics tracking</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `organization_id` | uint | Foreign key to `organizations` |
| `owner` | text | GitHub owner (org/user) |
| `name` | text | Repo name |
| `full_name` | text | Full name (owner/name, unique) |
| `description` | text | Repo description |
| `default_branch` | text | Default branch (main/master) |
| `language` | text | Primary language |
| `is_private` | bool | Visibility flag |
| `is_fork` | bool | Fork flag |
| `fork_source` | text | Parent repo full name if fork |
| `is_archived` | bool | Archived flag |
| `url` | text | HTML URL |
| `metadata_etag` | text | ETag for conditional metadata requests |
| `security_etag` | text | ETag for conditional security requests |
| `last_sync_at` | datetime | Last sync timestamp |
| `last_sync_run_id` | uint | Links to the SyncRun that last processed this repo |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Belongs to `organizations`
- Has many `repository_snapshots`
- Has many `security_alerts`
- Has many `ci_metrics_snapshots`

**Indexes:**
- Unique index on `full_name`
- Composite index on `(owner, name)`

</details>

<details>
<summary><b>repository_snapshots</b> — Point-in-time repository metrics</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `repository_id` | uint | Foreign key to `analytics_repositories` |
| `snapshot_at` | datetime | Timestamp of snapshot |
| `stars` | int | Stargazers count |
| `forks` | int | Forks count |
| `watchers` | int | Watchers count |
| `open_issues` | int | Open issues count |
| `open_prs` | int | Open pull requests count |
| `branch_count` | int | Total branches |
| `latest_release` | text | Latest release tag name |
| `latest_release_at` | datetime | Latest release date |
| `latest_tag` | text | Latest tag name |
| `latest_tag_at` | datetime | Latest tag date |
| `repo_updated_at` | datetime | Last activity on repo |
| `pushed_at` | datetime | Last code push timestamp |
| `dependabot_alert_count` | int | Open Dependabot alert count |
| `code_scanning_alert_count` | int | Open code scanning alert count |
| `secret_scanning_alert_count` | int | Open secret scanning alert count |
| `raw_data` | text | JSON raw GraphQL response |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Belongs to `analytics_repositories`

**Indexes:**
- Index on `repository_id`
- Index on `snapshot_at`

**Purpose:**
- Captures point-in-time metrics for trend analysis
- Uses timestamp (not date-only) to support future hourly snapshots

</details>

<details>
<summary><b>security_alerts</b> — Security alert tracking (Dependabot, code scanning, secret scanning)</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `repository_id` | uint | Foreign key to `analytics_repositories` |
| `alert_type` | text | Alert source: `dependabot`, `code_scanning`, `secret_scanning` |
| `alert_number` | int | Alert number from GitHub API |
| `state` | text | Alert state: `open`, `fixed`, `dismissed` |
| `severity` | text | Severity level: `critical`, `high`, `medium`, `low` |
| `summary` | text | Short summary |
| `description` | text | Full description |
| `html_url` | text | Link to alert on GitHub |
| `alert_created_at` | datetime | When GitHub alert was created |
| `fixed_at` | datetime | When alert was fixed |
| `dismissed_at` | datetime | When alert was dismissed |
| `dismissed_reason` | text | Reason for dismissal |
| `alert_data` | text | JSON blob of type-specific data (CVSS, CWE, package info) |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | DB record creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Belongs to `analytics_repositories`

**Indexes:**
- Index on `repository_id`
- Index on `alert_type`
- Index on `state`
- Index on `severity`
- Index on `alert_number`

</details>

<details>
<summary><b>ci_metrics_snapshots</b> — Point-in-time CI metrics from GoFortress workflow artifacts</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `repository_id` | uint | Foreign key to `analytics_repositories` |
| `snapshot_at` | datetime | Timestamp of snapshot |
| `workflow_run_id` | int64 | GitHub Actions workflow run ID |
| `branch` | text | Branch the workflow ran on |
| `commit_sha` | text | Commit SHA of the workflow run |
| `go_files_loc` | int | Lines of code in Go files |
| `test_files_loc` | int | Lines of code in test files |
| `go_files_count` | int | Number of Go source files |
| `test_files_count` | int | Number of test files |
| `test_count` | int | Total unit test count |
| `benchmark_count` | int | Total benchmark count |
| `coverage_percent` | float64 | Code coverage percentage (nullable) |
| `raw_data` | text | JSON raw artifact data |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Belongs to `analytics_repositories`

**Indexes:**
- Index on `repository_id`
- Index on `snapshot_at`
- Index on `workflow_run_id`

**Purpose:**
- Captures CI metrics from GoFortress workflow artifacts for trend analysis
- Sources: `loc-stats` (JSON), `statistics-section` (markdown fallback), `coverage-stats-codecov` (JSON), `tests-section` (markdown), `bench-stats-*` (JSON)
- Tracks code size, test coverage, and test/benchmark counts over time

</details>

<details>
<summary><b>sync_runs</b> — Analytics sync execution tracking</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `started_at` | datetime | When sync started |
| `completed_at` | datetime | When sync completed |
| `status` | text | Execution status: `running`, `completed`, `failed`, `partial` |
| `sync_type` | text | Sync type: `full`, `security_only`, `metadata_only` |
| `org_filter` | text | Organization filter (empty = all orgs) |
| `repo_filter` | text | Repository filter (empty = all repos) |
| `repos_processed` | int | Total repos processed |
| `repos_skipped` | int | Skipped via change detection |
| `repos_failed` | int | Failed to process |
| `snapshots_created` | int | New snapshots written |
| `alerts_upserted` | int | Alerts created/updated |
| `api_calls_made` | int | Total GitHub API calls |
| `duration_ms` | int | Total execution time in milliseconds |
| `errors` | text | JSON array of error details |
| `last_processed_repo` | text | For future resume support |
| `metadata` | text | JSON metadata |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last update timestamp |
| `deleted_at` | datetime | Soft delete timestamp |

**Relations:**
- Standalone (no foreign keys)

**Indexes:**
- Index on `status`

**Purpose:**
- Tracks each analytics sync execution for observability
- Provides detailed metrics about API usage, processing time, and errors
- Supports incremental sync via `last_processed_repo`

</details>

<br>

## 🔄 Import/Export

### Import Workflow

```bash
# 1. Validate YAML before import
go-broadcast validate

# 2. Initialize database (if needed)
go-broadcast db init

# 3. Import configuration
go-broadcast db import sync.yaml

# 4. Verify import
go-broadcast db status
```

### Import Behavior

By default, import performs a **full replacement**: all existing data is deleted and replaced with the imported configuration. All operations are performed within a transaction - if import fails, no changes are made to the database.

### Export Workflow

```bash
# Export entire configuration
go-broadcast db export --output backup.yaml

# Export single group for review
go-broadcast db export --group mrz-tools --output review.yaml

# Export to stdout (pipe to other tools)
go-broadcast db export --stdout | yq eval '.groups[0].name'
```

### Round-Trip Guarantee

The converter preserves:
- ✅ All field values (no data loss)
- ✅ Group ordering (via `position`)
- ✅ Target ordering within groups
- ✅ File/directory mapping order
- ✅ Metadata fields
- ✅ Transform configurations
- ✅ Dependency relationships

**Tested with:**
- Real `sync.yaml` (1857 lines, 50+ repos)
- Complex nested structures
- All field types

<br>

## ⌨️ CLI Commands

### Database Management

<details>
<summary><code>db init</code> — Initialize database</summary>

```bash
# Create new database
go-broadcast db init

# Force re-initialization (drops all data)
go-broadcast db init --force

# Use custom path
go-broadcast db init --db-path /tmp/test.db
```

**What it does:**
- Creates SQLite database file
- Runs auto-migrations (creates all tables)
- Applies WAL mode + pragmas
- Creates initial schema version

</details>

<details>
<summary><code>db status</code> — Show database information</summary>

```bash
# Show database status
go-broadcast db status
```

**Output:**
- Database path
- Connection status
- Schema version
- Record counts (configs, groups, targets, file lists, etc.)
- Size on disk

</details>

<details>
<summary><code>db import</code> — Import YAML configuration</summary>

```bash
# Basic import (replaces all data)
go-broadcast db import sync.yaml

# Custom database path
go-broadcast db import sync.yaml --db-path /tmp/test.db
```

**Transaction safety:**
- All-or-nothing: Import fails → no changes
- Validation runs before write
- Circular dependencies detected

</details>

<details>
<summary><code>db export</code> — Export to YAML</summary>

```bash
# Export to file
go-broadcast db export --output sync.yaml

# Export single group
go-broadcast db export --group mrz-tools --output group.yaml

# Export to stdout
go-broadcast db export --stdout
```

</details>

<details>
<summary><code>db query</code> — Query configuration</summary>

```bash
# Which repos sync this file?
go-broadcast db query --file .github/workflows/ci.yml

# What files sync to this repo?
go-broadcast db query --repo mrz1836/my-repo

# Which targets use this file list?
go-broadcast db query --file-list ai-files

# Search file paths (pattern matching)
go-broadcast db query --contains "workflows"

# JSON output for scripting
go-broadcast db query --file .github/workflows/ci.yml --json
```

</details>

<details>
<summary><code>db validate</code> — Validate database consistency</summary>

```bash
# Run all validation checks
go-broadcast db validate

# JSON output
go-broadcast db validate --json
```

**Checks:**
- Orphaned file list references
- Circular dependencies in groups
- Invalid foreign keys
- Broken polymorphic associations
- Constraint violations

</details>

<details>
<summary><code>db diff</code> — Compare YAML vs Database</summary>

```bash
# Show differences
go-broadcast db diff sync.yaml

# Detailed output
go-broadcast db diff sync.yaml --detail
```

</details>

### CRUD Commands

These commands provide granular, AI-agent-friendly access to individual database records. All commands support `--json` for structured output with a standard response envelope:

```json
{
  "success": true,
  "action": "created",
  "type": "group",
  "data": { ... },
  "count": 1,
  "error": "",
  "hint": ""
}
```

#### Group Management — `db group`

<details>
<summary><code>db group list</code> — List all groups</summary>

```bash
go-broadcast db group list
go-broadcast db group list --json
```

**Output:** Group ID, name, enabled state, and target count for each group.

</details>

<details>
<summary><code>db group get &lt;id&gt;</code> — Show group details</summary>

```bash
go-broadcast db group get mrz-tools
go-broadcast db group get mrz-tools --json
```

**Output:** Full group details including source repo, branch, global/default settings, and all targets.

</details>

<details>
<summary><code>db group create</code> — Create a new group</summary>

```bash
go-broadcast db group create \
  --id mrz-tools \
  --name "MRZ Tools" \
  --source-repo mrz1836/go-broadcast \
  --source-branch main \
  --json

# With optional fields
go-broadcast db group create \
  --id security-group \
  --name "Security Group" \
  --source-repo mrz1836/go-broadcast \
  --source-branch main \
  --description "Security-related files" \
  --priority 10 \
  --disabled \
  --json
```

**What it does:**
- Creates the group, source, group_globals, and group_defaults records atomically in a single transaction
- Auto-creates the organization and repo records if they don't exist

</details>

<details>
<summary><code>db group update &lt;id&gt;</code> — Update group fields</summary>

```bash
go-broadcast db group update mrz-tools --name "MRZ Tools v2" --json
go-broadcast db group update mrz-tools --description "Updated description" --priority 5 --json
```

**Updatable fields:** `--name`, `--description`, `--priority`

</details>

<details>
<summary><code>db group delete &lt;id&gt;</code> — Delete a group</summary>

```bash
# Soft delete (reversible)
go-broadcast db group delete mrz-tools --json

# Hard delete (permanent, requires no FK dependencies)
go-broadcast db group delete mrz-tools --hard --json
```

</details>

<details>
<summary><code>db group enable/disable &lt;id&gt;</code> — Toggle group enabled state</summary>

```bash
go-broadcast db group enable mrz-tools --json
go-broadcast db group disable mrz-tools --json
```

</details>

#### Target Management — `db target`

<details>
<summary><code>db target list</code> — List targets in a group</summary>

```bash
go-broadcast db target list --group mrz-tools --json
```

**Output:** All targets in the group with repo name, branch, and PR settings.

</details>

<details>
<summary><code>db target get</code> — Show target details</summary>

```bash
go-broadcast db target get --group mrz-tools --repo mrz1836/go-api --json
```

**Output:** Full target details including inline mappings, file list refs, directory list refs, and transforms.

</details>

<details>
<summary><code>db target add</code> — Add a target to a group</summary>

```bash
go-broadcast db target add --group mrz-tools --repo mrz1836/new-repo --json

# With optional settings
go-broadcast db target add \
  --group mrz-tools \
  --repo mrz1836/new-repo \
  --branch main \
  --json
```

**Idempotent:** Returns `"already_exists"` action if the target already exists in the group.

</details>

<details>
<summary><code>db target remove</code> — Remove a target from a group</summary>

```bash
# Soft delete
go-broadcast db target remove --group mrz-tools --repo mrz1836/old-repo --json

# Hard delete
go-broadcast db target remove --group mrz-tools --repo mrz1836/old-repo --hard --json
```

</details>

<details>
<summary><code>db target update</code> — Update target settings</summary>

```bash
go-broadcast db target update \
  --group mrz-tools \
  --repo mrz1836/go-api \
  --branch develop \
  --pr-labels "sync,automated" \
  --pr-assignees "mrz1836" \
  --pr-reviewers "reviewer1,reviewer2" \
  --json
```

**Updatable fields:** `--branch`, `--pr-labels`, `--pr-assignees`, `--pr-reviewers`

</details>

#### File List Management — `db file-list`

<details>
<summary><code>db file-list list</code> — List all file lists</summary>

```bash
go-broadcast db file-list list --json
```

</details>

<details>
<summary><code>db file-list get &lt;id&gt;</code> — Show file list with mappings</summary>

```bash
go-broadcast db file-list get ai-files --json
```

**Output:** File list details and all file mappings (src, dest, delete_flag).

</details>

<details>
<summary><code>db file-list create</code> — Create a new file list</summary>

```bash
go-broadcast db file-list create --id security-files --name "Security Files" --json
```

</details>

<details>
<summary><code>db file-list delete &lt;id&gt;</code> — Delete a file list</summary>

```bash
go-broadcast db file-list delete security-files --json
go-broadcast db file-list delete security-files --hard --json
```

</details>

<details>
<summary><code>db file-list add-file &lt;id&gt;</code> — Add a file mapping to a file list</summary>

```bash
go-broadcast db file-list add-file ai-files --src SECURITY.md --dest SECURITY.md --json

# With delete flag (marks file for deletion in targets)
go-broadcast db file-list add-file ai-files --src "" --dest old-file.txt --delete-flag --json
```

</details>

<details>
<summary><code>db file-list remove-file &lt;id&gt;</code> — Remove a file mapping by destination path</summary>

```bash
go-broadcast db file-list remove-file ai-files --dest SECURITY.md --json
```

</details>

#### Directory List Management — `db dir-list`

<details>
<summary><code>db dir-list list</code> — List all directory lists</summary>

```bash
go-broadcast db dir-list list --json
```

</details>

<details>
<summary><code>db dir-list get &lt;id&gt;</code> — Show directory list with mappings</summary>

```bash
go-broadcast db dir-list get github-workflows --json
```

</details>

<details>
<summary><code>db dir-list create</code> — Create a new directory list</summary>

```bash
go-broadcast db dir-list create --id ci-configs --name "CI Configurations" --json
```

</details>

<details>
<summary><code>db dir-list delete &lt;id&gt;</code> — Delete a directory list</summary>

```bash
go-broadcast db dir-list delete ci-configs --json
go-broadcast db dir-list delete ci-configs --hard --json
```

</details>

<details>
<summary><code>db dir-list add-dir &lt;id&gt;</code> — Add a directory mapping to a directory list</summary>

```bash
go-broadcast db dir-list add-dir github-workflows \
  --src .github/workflows \
  --dest .github/workflows \
  --json

# With options
go-broadcast db dir-list add-dir github-workflows \
  --src .github/workflows \
  --dest .github/workflows \
  --exclude "*.tmp,*.bak" \
  --preserve-structure \
  --json
```

</details>

<details>
<summary><code>db dir-list remove-dir &lt;id&gt;</code> — Remove a directory mapping by destination path</summary>

```bash
go-broadcast db dir-list remove-dir github-workflows --dest .github/workflows --json
```

</details>

#### Inline File Mappings — `db file`

Manage file mappings directly on targets (owner_type="target"):

<details>
<summary><code>db file list</code> — List inline file mappings on a target</summary>

```bash
go-broadcast db file list --group mrz-tools --repo mrz1836/go-api --json
```

</details>

<details>
<summary><code>db file add</code> — Add an inline file mapping to a target</summary>

```bash
go-broadcast db file add \
  --group mrz-tools \
  --repo mrz1836/go-api \
  --src .cursorrules \
  --dest .cursorrules \
  --json
```

</details>

<details>
<summary><code>db file remove</code> — Remove an inline file mapping from a target</summary>

```bash
go-broadcast db file remove \
  --group mrz-tools \
  --repo mrz1836/go-api \
  --dest .cursorrules \
  --json
```

</details>

#### Inline Directory Mappings — `db dir`

Manage directory mappings directly on targets (owner_type="target"):

<details>
<summary><code>db dir list</code> — List inline directory mappings on a target</summary>

```bash
go-broadcast db dir list --group mrz-tools --repo mrz1836/go-api --json
```

</details>

<details>
<summary><code>db dir add</code> — Add an inline directory mapping to a target</summary>

```bash
go-broadcast db dir add \
  --group mrz-tools \
  --repo mrz1836/go-api \
  --src .github/workflows \
  --dest .github/workflows \
  --exclude "*.tmp" \
  --preserve-structure \
  --json
```

</details>

<details>
<summary><code>db dir remove</code> — Remove an inline directory mapping from a target</summary>

```bash
go-broadcast db dir remove \
  --group mrz-tools \
  --repo mrz1836/go-api \
  --dest .github/workflows \
  --json
```

</details>

#### Reference Management — `db ref`

Attach or detach shared file lists and directory lists to/from individual targets:

<details>
<summary><code>db ref add-file-list</code> — Attach a file list to a target</summary>

```bash
go-broadcast db ref add-file-list \
  --group mrz-tools \
  --repo mrz1836/go-api \
  --file-list ai-files \
  --json
```

**Idempotent:** Returns `"already_attached"` action if the reference already exists.

</details>

<details>
<summary><code>db ref remove-file-list</code> — Detach a file list from a target</summary>

```bash
go-broadcast db ref remove-file-list \
  --group mrz-tools \
  --repo mrz1836/go-api \
  --file-list ai-files \
  --json
```

</details>

<details>
<summary><code>db ref add-dir-list</code> — Attach a directory list to a target</summary>

```bash
go-broadcast db ref add-dir-list \
  --group mrz-tools \
  --repo mrz1836/go-api \
  --dir-list github-workflows \
  --json
```

</details>

<details>
<summary><code>db ref remove-dir-list</code> — Detach a directory list from a target</summary>

```bash
go-broadcast db ref remove-dir-list \
  --group mrz-tools \
  --repo mrz1836/go-api \
  --dir-list github-workflows \
  --json
```

</details>

#### Bulk Operations — `db bulk`

Apply changes to all targets in a group at once:

<details>
<summary><code>db bulk add-file-list</code> — Attach a file list to all targets in a group</summary>

```bash
go-broadcast db bulk add-file-list --group mrz-tools --file-list ai-files --json
```

**Response includes:** Number of affected targets. Idempotent — skips targets that already have the reference.

</details>

<details>
<summary><code>db bulk remove-file-list</code> — Detach a file list from all targets in a group</summary>

```bash
go-broadcast db bulk remove-file-list --group mrz-tools --file-list ai-files --json
```

</details>

<details>
<summary><code>db bulk add-dir-list</code> — Attach a directory list to all targets in a group</summary>

```bash
go-broadcast db bulk add-dir-list --group mrz-tools --dir-list github-workflows --json
```

</details>

<details>
<summary><code>db bulk remove-dir-list</code> — Detach a directory list from all targets in a group</summary>

```bash
go-broadcast db bulk remove-dir-list --group mrz-tools --dir-list github-workflows --json
```

</details>

### Analytics Commands

<details>
<summary><code>analytics sync</code> — Collect repository analytics data</summary>

```bash
# Full sync for all organizations
go-broadcast analytics sync

# Sync specific organization only
go-broadcast analytics sync --org mrz1836

# Sync specific repository
go-broadcast analytics sync --repo mrz1836/go-broadcast

# Security alerts only (skip metadata)
go-broadcast analytics sync --security-only

# Force full sync (ignore change detection)
go-broadcast analytics sync --full

# Preview what would be synced
go-broadcast analytics sync --dry-run
```

**What it does:**
- Discovers all repos in configured organizations via GitHub REST API
- Collects repository metadata using batched GraphQL queries (25 repos per query)
- Fetches security alerts (Dependabot, code scanning, secret scanning) concurrently
- Creates point-in-time snapshots and upserts security alerts
- Tracks sync progress in `sync_runs` table

</details>

<details>
<summary><code>analytics status</code> — Show analytics status for repositories</summary>

```bash
# Show analytics status for all repositories
go-broadcast analytics status

# Show status for specific repository
go-broadcast analytics status mrz1836/go-broadcast
```

**Output:**
- Repository metrics (stars, forks, open issues, PRs)
- Security alert summary by severity
- Last sync timestamp and duration

</details>

<br>

## 🔍 Queries

### Common Query Patterns

**1. Which repos sync a specific file?**

```bash
go-broadcast db query --file .github/workflows/ci.yml
```

**Output:**
```
File: .github/workflows/ci.yml
Synced to 12 repositories:

Group: mrz-tools
  • mrz1836/go-api
  • mrz1836/go-broadcast
  • mrz1836/go-cache

Group: bsva-repos
  • bsvalias/go-bsvalias
  • bsvalias/paymail-server
```

**2. What files sync to a repository?**

```bash
go-broadcast db query --repo mrz1836/go-broadcast
```

**Output:**
```
Repository: mrz1836/go-broadcast
Group: mrz-tools

Inline Files:
  • .github/workflows/ci.yml → .github/workflows/ci.yml
  • .github/workflows/codeql.yml → .github/workflows/codeql.yml

From File Lists:
  [ai-files]
    • .github/copilot-instructions.md
    • .cursorrules

  [codecov-default]
    • .github/codecov.yml
```

**3. Which targets use a file list?**

```bash
go-broadcast db query --file-list ai-files
```

**Output:**
```
File List: ai-files (AI development guidelines)
Used by 47 targets:

Group: mrz-tools (15 targets)
Group: bsva-repos (32 targets)
```

**4. Search by pattern**

```bash
go-broadcast db query --contains "dependabot"
```

**Output:**
```
Found 3 file mappings matching "dependabot":

• .github/dependabot.yml
  → synced to 42 repos (via file list "github-security")

• .github/workflows/dependabot-auto-merge.yml
  → synced to 12 repos (group: mrz-tools)
```

<br>

## 🧪 Testing

### Run All Tests

```bash
# Full test suite
go test ./internal/db/... -v -race -cover

# With coverage report
go test ./internal/db/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Categories

| Category | Files | Coverage Target |
|----------|-------|-----------------|
| Unit Tests | `models_test.go` | >90% |
| Repository Tests | `repository_test.go` | >85% |
| Converter Tests | `converter_test.go` | >85% |
| Query Tests | `query_test.go` | >85% |
| Dependency Tests | `dependency_test.go` | >95% |
| Concurrent Tests | `concurrent_test.go` | >80% |
| Analytics Model Tests | `models_analytics_test.go` | >85% |
| Analytics Repository Tests | `repository_analytics_test.go` | >85% |
| File Mapping Repository | `repository_file_mapping_test.go` | >80% |
| Directory Mapping Repository | `repository_directory_mapping_test.go` | >80% |
| Bulk Repository | `repository_bulk_test.go` | >80% |
| CLI CRUD Tests | `db_crud_test.go` | >80% |

### Key Test Scenarios

**Metadata Round-Trip:**
- Create model with complex metadata → read back → verify
- Tested on all 18 models

**Validation Hooks:**
- Invalid repo format → error
- Path traversal → rejected
- Invalid email → error
- Delete flag → skip src validation

**Converter Round-Trip:**
- Import real `sync.yaml` (1857 lines)
- Export to config
- Assert structural equality
- Verify: ordering, refs, transforms

**Concurrent Access:**
- 100 goroutines reading simultaneously
- 20 goroutines writing to different groups
- No data corruption or deadlocks

<br>

## ⚡ Performance

### Benchmarks

Run with `bench_heavy` build tag:

```bash
go test ./internal/db/... -tags bench_heavy -bench=. -benchtime=10s
```

**Results** (Apple M3 Max, 36GB RAM):

| Operation | Latency | Throughput |
|-----------|---------|------------|
| Import sync.yaml (50 repos) | ~150ms | — |
| Export to YAML | ~80ms | — |
| Query by file | <1ms | 10,000+ ops/s |
| Query by repo | <1ms | 10,000+ ops/s |
| List all groups | <1ms | 10,000+ ops/s |
| Create target | <0.5ms | 20,000+ ops/s |

**Database Size:**
- sync.yaml: 1857 lines → 420KB SQLite
- 6 groups, 50+ targets → <1MB database

<br>

## 🏗️ Architecture

### Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **SQLite Driver** | `glebarez/sqlite` (pure Go) | No CGO → cross-compilation works |
| **Transform Storage** | Separate `transforms` table | Flexibility, future growth |
| **Metadata** | JSON `metadata` on every table | Extensibility without schema changes |
| **Polymorphic Associations** | `owner_type` + `owner_id` | Single table serves multiple parents |
| **Ordering** | `position int` on all child tables | Preserves YAML ordering |

### SQLite Configuration

**Write-Ahead Logging (WAL):**
```
PRAGMA journal_mode=WAL
```
- Concurrent reads while writing
- Better crash recovery
- ~3x faster for write-heavy workloads

**Pragmas:**
```sql
PRAGMA synchronous=NORMAL       -- Good durability with WAL
PRAGMA busy_timeout=5000        -- 5s lock contention timeout
PRAGMA foreign_keys=ON          -- Enforce FK constraints
PRAGMA cache_size=-20000        -- 20MB page cache
PRAGMA temp_store=MEMORY        -- Temp tables in memory
PRAGMA mmap_size=268435456      -- 256MB memory-mapped I/O
```

**Connection Pool:**
- `MaxOpenConns=1` (SQLite single-writer)
- `MaxIdleConns=1`

### Transaction Strategy

**Import:**
- Single transaction for entire import
- Any error → full rollback
- Reference resolution: Create all lists first → build ID map → resolve FK refs

**Export:**
- Read-only queries with preloading
- Ordered by `position` for faithful YAML reproduction

<br>

## ⚙️ Configuration

### Database Path

**Default:**
```
~/.config/go-broadcast/broadcast.db
```

**Override via flag:**
```bash
go-broadcast db <command> --db-path /custom/path.db
```

**Precedence:**
1. `--db-path` flag (if provided)
2. Default path

### Connection Settings

Configured in `internal/db/sqlite.go`:

```go
// Connection pool
MaxOpenConns: 1          // SQLite single-writer
MaxIdleConns: 1
ConnMaxLifetime: time.Hour  // 1 hour connection reuse limit
```

<br>

## 📚 Migration Guide

### From YAML-Only to Database-Backed

**Step 1: Backup**
```bash
cp sync.yaml sync.yaml.backup
```

**Step 2: Initialize Database**
```bash
go-broadcast db init
```

**Step 3: Import & Verify**
```bash
go-broadcast db import sync.yaml
go-broadcast db validate
```

**Step 4: Test Export**
```bash
go-broadcast db export --output test-export.yaml
diff sync.yaml test-export.yaml  # Should be structurally equivalent
```

**Step 5: Test Sync (Dry-Run)**
```bash
go-broadcast sync --from-db --dry-run
```

**Step 6: Switch to Database**
```bash
# Update your workflows to use --from-db
go-broadcast sync --from-db
```

### Rollback Plan

Database backend is purely additive:

1. **Keep YAML file:** Database doesn't replace it
2. **Export anytime:** `go-broadcast db export --output sync.yaml`
3. **Fall back:** Remove `--from-db` flag → uses YAML

<br>

## 🔗 Related Documentation

- [README](../README.md) — Project overview
- [Configuration Guide](./configuration.md) — YAML format reference
- [CLI Reference](./cli.md) — All commands
- [Development Guide](./development.md) — Contributing

<br>

<div align="center">

**Questions or Issues?**

[Open an issue](https://github.com/mrz1836/go-broadcast/issues) • [View source](https://github.com/mrz1836/go-broadcast)

</div>
