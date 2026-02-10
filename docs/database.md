<div align="center">

# 🗄️ Database Backend

**Structured Configuration Storage for go-broadcast**

[![SQLite](https://img.shields.io/badge/SQLite-3-003B57?logo=sqlite&logoColor=white)](https://sqlite.org/)
[![GORM v2](https://img.shields.io/badge/GORM-v2-00ADD8?logo=go&logoColor=white)](https://gorm.io/)
[![Pure Go](https://img.shields.io/badge/Pure%20Go-No%20CGO-00ADD8?logo=go&logoColor=white)](https://go.dev/)

</div>

---

## 📑 Navigation

| | | |
|:---:|:---:|:---:|
| [🚀 Quick Start](#-quick-start) | [📊 Schema Reference](#-schema-reference) | [🔄 Import/Export](#-importexport) |
| [⌨️ CLI Commands](#-cli-commands) | [🔍 Queries](#-queries) | [🧪 Testing](#-testing) |
| [⚡ Performance](#-performance) | [🏗️ Architecture](#-architecture) | [⚙️ Configuration](#-configuration) |

---

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

# Import with validation
go-broadcast db import sync.yaml --validate

# Merge with existing data
go-broadcast db import sync.yaml --merge
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
go-broadcast db export --output sync.yaml

# Export single group
go-broadcast db export --group mrz-tools --output group.yaml

# Export to stdout
go-broadcast db export --stdout
```

### Sync from Database

```bash
# Use database instead of YAML file
go-broadcast sync --from-db

# Sync specific groups from database
go-broadcast sync --from-db --groups "core,security"

# Dry-run with database configuration
go-broadcast sync --from-db --dry-run
```

---

## 📊 Schema Reference

### Entity Relationship Diagram

```
┌──────────────┐
│   configs    │
└──────┬───────┘
       │
       ├─────────────────┐
       │                 │
       ▼                 ▼
┌──────────────┐  ┌──────────────┐
│    groups    │  │  file_lists  │
└──────┬───────┘  └──────┬───────┘
       │                 │
       ├────┐            │
       │    │            ▼
       ▼    ▼     ┌─────────────────┐
┌─────────┐ │     │ file_mappings   │
│ sources │ │     │  (polymorphic)  │
└─────────┘ │     └─────────────────┘
            │
            ▼
     ┌──────────────┐
     │   targets    │◄────┐
     └──────┬───────┘     │
            │             │
            ├─────────────┤
            │             │
            ▼             │
     ┌──────────────┐     │
     │ transforms   │     │
     │ (polymorphic)│     │
     └──────────────┘     │
                          │
     ┌─────────────────────┤
     │ target_file_list_refs
     └──────────────────────┘
```

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
<summary><b>sources</b> — Source repository configuration</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `group_id` | uint | Foreign key to `groups` (1:1) |
| `repo` | text | Repository (org/name) |
| `branch` | text | Branch name |
| `blob_size_limit` | text | Max blob size |
| `security_email` | text | Security contact |
| `support_email` | text | Support contact |
| `metadata` | text | JSON metadata |

**Validation:**
- `repo`: Must match `org/name` format
- `branch`: Must be valid branch name
- Emails: Must be valid email addresses

</details>

<details>
<summary><b>targets</b> — Target repository configuration</summary>

| Column | Type | Description |
|--------|------|-------------|
| `id` | uint | Primary key |
| `group_id` | uint | Foreign key to `groups` |
| `repo` | text | Repository (org/name) |
| `branch` | text | Branch name |
| `pr_labels` | text | JSON array of PR labels |
| `pr_assignees` | text | JSON array of assignees |
| `pr_reviewers` | text | JSON array of reviewers |
| `pr_team_reviewers` | text | JSON array of team reviewers |
| `position` | int | Ordering for export |
| `metadata` | text | JSON metadata |

**Relations:**
- Belongs to `groups`
- Has many `file_mappings` (polymorphic)
- Has many `directory_mappings` (polymorphic)
- Has zero or one `transforms` (polymorphic)
- Many-to-many with `file_lists` via `target_file_list_refs`
- Many-to-many with `directory_lists` via `target_directory_list_refs`

**Indexes:**
- Composite: `(group_id, repo)` for fast lookups

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

---

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

| Flag | Behavior |
|------|----------|
| (default) | **Replace**: Delete all existing data, import fresh |
| `--merge` | **Merge**: Add new groups/targets, update existing |
| `--validate` | Validate YAML before import (exit on error) |

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

---

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

# Merge with existing data
go-broadcast db import sync.yaml --merge

# Validate before import
go-broadcast db import sync.yaml --validate

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
go-broadcast db query --file .github/workflows/ci.yml --format json
```

</details>

<details>
<summary><code>db validate</code> — Validate database consistency</summary>

```bash
# Run all validation checks
go-broadcast db validate

# Verbose output
go-broadcast db validate --verbose
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

# Machine-readable output
go-broadcast db diff sync.yaml --format json
```

</details>

---

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

---

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

### Key Test Scenarios

**Metadata Round-Trip:**
- Create model with complex metadata → read back → verify
- Tested on all 15 models

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

---

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

---

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

---

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

**Override via environment:**
```bash
export GO_BROADCAST_DB=/custom/path.db
go-broadcast db status
```

**Precedence:**
1. `--db-path` flag (highest)
2. `GO_BROADCAST_DB` environment variable
3. Default path (lowest)

### Connection Settings

Configured in `internal/db/sqlite.go`:

```go
// Connection pool
MaxOpenConns: 1          // SQLite single-writer
MaxIdleConns: 1
ConnMaxLifetime: 0       // No connection reuse limit
```

---

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

---

## 🔗 Related Documentation

- [README](../README.md) — Project overview
- [Configuration Guide](./configuration.md) — YAML format reference
- [CLI Reference](./cli.md) — All commands
- [Development Guide](./development.md) — Contributing

---

<div align="center">

**Questions or Issues?**

[Open an issue](https://github.com/mrz1836/go-broadcast/issues) • [View source](https://github.com/mrz1836/go-broadcast)

</div>
