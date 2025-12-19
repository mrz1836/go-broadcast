# CLAUDE.md

## Welcome, Claude

This repository uses **`AGENTS.md`** as the entry point to our modular technical conventions:

- The main **`AGENTS.md`** provides an overview and directory structure
- Technical standards are organized in **`.github/tech-conventions/`**:
  - **Core Development**: Go essentials, testing, documentation
  - **Version Control**: Commits, branches, pull requests, releases
  - **Infrastructure**: CI/CD, dependencies, security, workflows

> **Start with `AGENTS.md`**, then explore specific conventions in `tech-conventions/`.

---

## Quick Reference

| Command | Purpose |
|---------|---------|
| `magex test` | Fast linting + unit tests |
| `magex test:coverrace` | Full CI suite with race detection |
| `magex lint` | Run 60+ linters via golangci-lint |
| `magex format:fix` | Auto-fix code formatting |
| `magex bench` | Run performance benchmarks |
| `magex build` | Build go-broadcast binary |
| `magex deps:audit` | Scan for security vulnerabilities |
| `magex version:bump bump=patch push` | Create release tag (triggers CI) |
| `./go-broadcast validate --config examples/minimal.yaml` | Validate configuration |
| `./go-broadcast sync --dry-run --config examples/minimal.yaml` | Test sync without changes |
| `./go-broadcast cancel --config examples/minimal.yaml` | Cancel active sync operations |
| `./go-broadcast status --config examples/minimal.yaml` | Check configuration status |

---

## MAGE-X Build System

go-broadcast uses **[MAGE-X](https://github.com/mrz1836/mage-x)** - a zero-config build automation system providing 150+ commands for testing, linting, building, and releasing Go projects.

**Configuration (`.mage.yaml`):**
```yaml
project:
  name: go-broadcast
  binary: go-broadcast
  module: github.com/mrz1836/go-broadcast
  main: ./cmd/go-broadcast/main.go

build:
  ldflags:
    - "-s -w"
    - "-X main.version={{.Version}}"
  output: "./cmd/go-broadcast"
```

**Parameter Syntax:**
```bash
magex bench time=50ms count=3     # Quick benchmarks with timing
magex version:bump bump=minor push # Bump version and push tag
magex test:fuzz time=30s          # Fuzz tests for 30 seconds
```

**Release Process:** Use `magex version:bump bump=patch push` to create a tag. GitHub Actions handles the actual release automatically.

---

## go-broadcast Workflows

**Development Cycle:**
```bash
magex test           # Fast linting + unit tests (before every commit)
magex test:race      # Unit tests with race detector
magex lint           # Run all linters
magex format:fix     # Fix formatting issues
```

**Configuration Validation:**
```bash
./go-broadcast validate --config examples/minimal.yaml
./go-broadcast sync --dry-run --config examples/minimal.yaml
./go-broadcast cancel --dry-run --config examples/minimal.yaml
```

**AI Text Generation (Optional):**
Enable via environment variables:
```bash
GO_BROADCAST_AI_ENABLED=true
ANTHROPIC_API_KEY=${{ secrets.ANTHROPIC_API_KEY }}
```
See [`.github/.env.base`](.env.base) for all options. Disabled by default; failures fall back to static templates.

---

## Testing & Benchmarks

**Unit Testing:**
```bash
magex test:unit       # Skip linting, run only tests
magex test:short      # Skip integration tests
magex test:coverrace  # Full CI suite with race detection
magex test:cover      # Unit tests with coverage report
```

**Fuzz Testing:**
```bash
magex test:fuzz time=30s  # Run all fuzz tests
go test -fuzz=FuzzConfigParsing -fuzztime=30s ./internal/config  # Specific fuzz test
```

**Benchmarks:**
| Type | Command | Duration |
|------|---------|----------|
| Quick (CI) | `magex bench` or `magex benchquick` | < 5 min |
| Heavy | `magex benchheavy` | 10-30 min |
| All | `magex benchall` | 30-60 min |

Custom commands (`benchheavy`, `benchall`) are defined in `magefile.go`.

---

## go-pre-commit

Fast Go-native pre-commit hooks (17x faster than Python alternatives).

```bash
go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@latest
go-pre-commit install
```

See [go-pre-commit documentation](https://github.com/mrz1836/go-pre-commit) for configuration and usage.

---

## Code Quality Guidelines

This project uses 60+ linters via golangci-lint with strict standards.

**Essential Practices:**
- Use `0600` for sensitive files, `0750` for directories
- Always check error returns: `if err := foo(); err != nil { ... }`
- Use context-aware functions: `DialContext`, `CommandContext`
- Create static error variables and wrap with context

**Code Quality:**
- Add comments to all exported functions, types, and constants
- Use `_` for intentionally unused parameters
- Avoid redefining built-in functions (`max`, `min`, etc.)
- Pre-allocate slices when size is known: `make([]Type, 0, knownSize)`

**Formatting:**
```bash
# Always use magex format:fix for code formatting
magex format:fix

# Never use gofumpt or fmt directly
```

**Common Patterns:**
- Use `fmt.Fprintf(w, format, args...)` for efficient string building
- Add `//nolint:linter // reason` only when necessary with clear explanation

**Running Linters:**
```bash
magex format:fix  # Fix formatting first
magex lint        # Run all linters
```

---

## Troubleshooting

**Debug Logging:**
```bash
./go-broadcast sync --log-level debug --config examples/minimal.yaml
./go-broadcast diagnose > diagnostics.json
```

**Environment Verification:**
```bash
gh auth status    # Check GitHub authentication
go mod verify     # Validate dependencies
govulncheck ./... # Security scan
```

---

## Checklist & Reminders

**Before Development:**
1. Read `AGENTS.md` thoroughly
2. Run `magex test` to ensure all tests pass
3. Verify GitHub authentication with `gh auth status`

**Key Rules:**
- **`AGENTS.md` is the ultimate authority** - When in doubt, refer to it first
- **Never tag releases** - Only repository code-owners handle releases
- **Security first** - Run `govulncheck` and validate external dependencies
- **Test thoroughly** - Use both unit tests and go-broadcast validation commands

---

## Documentation

| Document | Purpose |
|----------|---------|
| [`README.md`](../README.md) | Project overview and quick start |
| [`AGENTS.md`](AGENTS.md) | Primary authority for all development standards |
| [`CONTRIBUTING.md`](CONTRIBUTING.md) | Contribution guidelines |
| [`docs/`](../docs/) | Technical documentation |
| [`examples/`](../examples/) | Configuration examples |
