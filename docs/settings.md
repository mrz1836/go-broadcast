# Repository Settings Management

go-broadcast includes commands to manage GitHub repository settings at scale using reusable presets.

## Overview

- **Presets** define a standard set of repository settings (12 managed fields), labels, and rulesets
- **`scaffold`** creates new repositories with settings from a preset
- **`settings apply`** applies preset settings to an existing repository
- **`settings audit`** audits repositories against their assigned preset
- **`db preset`** manages presets in the database

## Presets

A preset defines:
- 10 boolean settings: `has_issues`, `has_wiki`, `has_projects`, `has_discussions`, `allow_squash_merge`, `allow_merge_commit`, `allow_rebase_merge`, `delete_branch_on_merge`, `allow_auto_merge`, `allow_update_branch`
- 2 string settings: `squash_merge_commit_title`, `squash_merge_commit_message`
- Labels (name, color, description)
- Rulesets (branch protection, tag protection)

Presets are fully user-defined. Define them in your `sync.yaml` configuration file under the `settings_presets` key, or create them directly in the database via `db preset create`.

A hardcoded `mvp` default is used as a fallback when no preset is found.

### Preset Resolution Order

When a command needs a preset, it resolves in this order:
1. Database lookup by external ID
2. Config file (`sync.yaml`) lookup
3. Hardcoded default (mvp)

## Commands

### scaffold

Create a new private GitHub repository with settings from a preset.

```bash
# Create with default preset
go-broadcast scaffold <owner/repo> "Project description"

# Create with specific preset
go-broadcast scaffold <owner/repo> "Go library" --preset my-preset

# Preview what would be created
go-broadcast scaffold <owner/repo> "Test" --dry-run

# Create with topics
go-broadcast scaffold <owner/repo> "Test" --topics "go,library,tools"
```

Flags:
- `--preset` — Settings preset ID (default: mvp)
- `--topics` — Comma-separated topics
- `--no-clone` — Don't clone repository after creation
- `--no-files` — Skip initial file creation
- `--dry-run` — Preview changes without creating

### settings apply

Apply preset settings to an existing repository.

```bash
# Apply with default preset
go-broadcast settings apply <owner/repo>

# Apply specific preset
go-broadcast settings apply <owner/repo> --preset my-preset

# Preview changes
go-broadcast settings apply <owner/repo> --dry-run

# Apply with topics and description
go-broadcast settings apply <owner/repo> --topics "go,library" --description "A Go library"
```

Flags:
- `--preset` — Settings preset ID (default: mvp)
- `--topics` — Comma-separated topics to set
- `--description` — Repository description to set
- `--force` — Skip confirmation
- `--dry-run` — Preview changes without applying

The command is idempotent: re-running with no changes produces no API calls.

### settings audit

Audit repositories against their assigned preset.

```bash
# Audit a single repo
go-broadcast settings audit <owner/repo>

# Audit with specific preset
go-broadcast settings audit <owner/repo> --preset my-preset

# Audit all repos in database
go-broadcast settings audit --all

# Audit all repos in an organization
go-broadcast settings audit --org <org-name>

# Audit and save results to database
go-broadcast settings audit <owner/repo> --save

# JSON output for CI
go-broadcast settings audit <owner/repo> --json
```

Flags:
- `--preset` — Settings preset ID
- `--org` — Audit all repos in an organization
- `--all` — Audit all repos in database
- `--save` — Save audit results to database
- `--format` — Output format: table, json
- `--dry-run` — Show what would be audited
- `--json` — Output as JSON

Exit code 1 if any checks fail (CI-friendly).

### db preset

Manage settings presets in the database.

```bash
# List all presets
go-broadcast db preset list

# Show preset details
go-broadcast db preset show <preset-id>

# Create a new preset with defaults
go-broadcast db preset create <preset-id> --name "My Preset"

# Delete a preset
go-broadcast db preset delete <preset-id>

# Assign a preset to a repository
go-broadcast db preset assign <preset-id> <owner/repo>

# Import presets from sync.yaml
go-broadcast db preset import
```

## Database Schema

Four tables support settings management:

| Table | Purpose |
|-------|---------|
| `settings_presets` | Preset definitions (12 settings + metadata) |
| `settings_preset_labels` | Labels belonging to a preset |
| `settings_preset_rulesets` | Rulesets belonging to a preset |
| `repo_settings_audits` | Audit results (score, checks) |

The `repos` table is extended with merge settings fields that are automatically populated during `analytics sync`.

## CI Integration

Use `settings audit` in CI to detect settings drift:

```yaml
- name: Audit repository settings
  run: go-broadcast settings audit ${{ github.repository }} --json
```

The command exits with code 1 if any checks fail, making it suitable for CI gates.

## Workflow

1. Define presets in your `sync.yaml` under `settings_presets`
2. Import presets to database: `go-broadcast db preset import`
3. Assign presets to repos: `go-broadcast db preset assign <preset-id> <owner/repo>`
4. Apply settings: `go-broadcast settings apply <owner/repo>`
5. Audit periodically: `go-broadcast settings audit --all --save`

The `analytics sync` command automatically backfills merge settings for all repos at zero additional API cost.
