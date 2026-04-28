# Presets — Discoverability and Seeding

This document covers how `go-broadcast` discovers settings presets, what is
shipped out of the box, and how to layer your own preset definitions on top.

For the complete preset schema and the `scaffold` / `settings apply` /
`settings audit` workflows, see [`settings.md`](./settings.md).

## Resolution Chain

When any command needs a preset (for example `--preset go-lib`), the
following sources are consulted in order:

1. **Database** (`SettingsPreset` rows in the local SQLite DB)
2. **Config file** (entries under `settings_presets:` in `sync.yaml`, or
   whichever path is supplied via `--config`)
3. **Bundled defaults** compiled into the binary (`mvp`, `go-lib`)

The first source to yield a match wins. If no source has the requested id,
the command exits with:

```
unknown preset: <id> (run 'go-broadcast presets list' to see available)
```

There is **no silent fallback** for unknown ids — typos surface immediately
instead of being papered over with a placeholder preset.

### Reserved `default` id

The literal id `default` is reserved and always resolves to the hardcoded
fallback returned by `config.DefaultPreset()`. Use it to opt out of the
discovery chain when you explicitly want the built-in defaults regardless of
DB or config-file overrides.

```bash
go-broadcast scaffold owner/example "Demo" --preset default
```

## Bundled Defaults

Two generic presets ship with the binary:

| ID       | Description                                                      |
|----------|------------------------------------------------------------------|
| `mvp`    | Minimal preset for any new repository (issues + squash merge)    |
| `go-lib` | Standard Go library preset (labels, branch and tag protection)   |

Both are intentionally generic so any fork of the project gets sensible
out-of-the-box behavior. They are also auto-seeded into the local
database the first time a preset-resolving command runs against an empty
preset table:

```
INFO  auto-seeded bundled preset(s) into empty DB count=2
```

After auto-seeding, the bundled defaults are stored as ordinary DB rows and
can be edited, overridden, or deleted like any other preset.

## Listing Presets

```bash
# Plain id list
go-broadcast presets list

# Annotate each entry with its source
go-broadcast presets list --show-source

# Machine-readable
go-broadcast presets list --json
```

The `--show-source` annotations are:

- `db` — stored in the local database
- `config-file:<path>` — defined in the supplied config file
- `bundled-default` — compiled into the binary

When the same id appears in multiple sources, every occurrence is listed so
the resolution chain is fully visible.

## Seeding Custom Presets

Use `presets seed` to keep the database in sync with a directory of preset
definitions:

```bash
# Re-seed the bundled defaults (idempotent — existing rows are kept)
go-broadcast presets seed

# Layer your own presets on top
go-broadcast presets seed --from /path/to/preset-yamls/
```

`--from <dir>` walks the directory and parses every `*.yaml`, `*.yml`, and
`*.json` file as a single preset. Each file produces one row, keyed by the
`id` field. When a file's id matches an existing row (including a bundled
default), the row is overridden and the change is logged at INFO. Re-running
the command on the same directory is safe — the result converges to the
current contents of the directory.

### Example preset YAML

```yaml
id: my-team
name: My Team Preset
description: Team-wide defaults for new repositories
has_issues: true
has_wiki: false
allow_squash_merge: true
allow_merge_commit: false
delete_branch_on_merge: true
squash_merge_commit_title: PR_TITLE
squash_merge_commit_message: COMMIT_MESSAGES
labels:
  - name: bug
    color: d73a4a
    description: Something isn't working
  - name: chore
    color: cccccc
    description: Maintenance work
rulesets:
  - name: branch-protection
    target: branch
    enforcement: active
    include:
      - "~DEFAULT_BRANCH"
    rules:
      - deletion
      - pull_request
```

Place the file in your seed directory and run:

```bash
go-broadcast presets seed --from /path/to/preset-yamls/
go-broadcast presets list --show-source   # confirm the new row shows source=db
```

## Where to Look Next

- `go-broadcast db status` — prints DB table counts and points at
  `presets list` for the cross-source inventory
- [`settings.md`](./settings.md) — full preset schema, scaffold/apply/audit
  workflows, and `db preset` CRUD commands
