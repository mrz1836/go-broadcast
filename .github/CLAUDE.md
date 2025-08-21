# CLAUDE.md

## 🤖 Welcome, Claude

This repository uses **`AGENTS.md`** as the entry point to our modular technical conventions:

* The main **`AGENTS.md`** provides an overview and directory structure
* Technical standards are organized in **`.github/tech-conventions/`** for portability:
  * **Core Development**: Go essentials, testing, documentation
  * **Version Control**: Commits, branches, pull requests, releases
  * **Infrastructure**: CI/CD, dependencies, security, workflows
  * **Project Management**: Labeling conventions

> **TL;DR:** **Start with `AGENTS.md`**, then explore specific conventions in **`tech-conventions/`**.
> All technical questions are answered in these focused documents.

---

## 🔧 MAGE-X Build System

go-broadcast uses **[MAGE-X](https://github.com/mrz1836/mage-x)** - a zero-boilerplate build automation system for Go that provides **241+ built-in commands** plus project-specific custom commands.

### What is MAGE-X?

MAGE-X revolutionizes Go build automation with TRUE zero-configuration. Unlike traditional Mage which requires writing wrapper functions, MAGE-X provides all commands instantly through the `magex` binary:

- **Zero Setup Required**: No magefile.go needed for basic operations
- **241+ Built-in Commands**: Complete build, test, lint, release, and deployment workflows
- **Hybrid Execution**: Built-in commands execute directly for speed; custom commands from magefile.go
- **Smart Configuration**: Uses `.mage.yaml` for project-specific settings
- **Parameter Support**: Modern parameter syntax: `magex command param=value`

### MAGE-X in go-broadcast

**Built-in Commands Used:**
- **Build**: `magex build`, `magex build:clean`, `magex build:install`
- **Testing**: `magex test`, `magex test:race`, `magex test:cover`, `magex test:coverrace`
- **Quality**: `magex lint`, `magex format:fix`, `magex deps:audit`
- **Benchmarks**: `magex bench time=50ms`, `magex bench:cpu`, `magex bench:mem`
- **Version**: `magex version:bump bump=patch push`, `magex version:show`
- **Tools**: `magex tools:update`, `magex deps:update`, `magex update:install`

**Custom Commands (from magefile.go):**
- `magex benchheavy`: Intensive benchmarks (10-30 minutes, 1000+ concurrent tasks)
- `magex benchall`: All benchmarks including heavy ones (30-60 minutes total)
- `magex benchquick`: Quick benchmarks only (same as `magex bench`)

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
    - "-X main.commit={{.Commit}}"
    - "-X main.buildDate={{.Date}}"
  flags:
    - "-trimpath"
  output: "./cmd/go-broadcast"
```

**Parameter Syntax:**
```bash
# New parameter format (MAGE-X)
magex bench time=50ms count=3     # Quick benchmarks with custom timing
magex version:bump bump=minor push # Bump minor version and push tag
magex test:fuzz time=30s          # Run fuzz tests for 30 seconds

# Available parameter types
magex command param=value         # Key-value parameters
magex command flag                # Standalone flags
magex command dry-run             # Preview mode
```

### MAGE-X Namespace Architecture

MAGE-X organizes commands into **37 built-in namespaces**. go-broadcast actively uses these key namespaces:

**Core Development Namespaces:**
- **Build** (`build:`): Compilation, cross-platform building, Docker integration
- **Test** (`test:`): Unit testing, integration testing, coverage, race detection, fuzz testing
- **Lint** (`lint:`): Code quality, 60+ linters via golangci-lint, automatic fixes
- **Format** (`format:`): Code formatting (gofumpt, imports, etc.)
- **Deps** (`deps:`): Dependency management, security audits, updates

**Productivity Namespaces:**
- **Bench** (`bench:`): Performance benchmarking, profiling (CPU, memory, trace)
- **Tools** (`tools:`): Development tool management and updates
- **Docs** (`docs:`): Documentation generation (pkgsite/godoc hybrid)
- **Version** (`version:`): Semantic versioning, changelog generation
- **Git** (`git:`): Git operations, tagging, commit management

**Enterprise Namespaces (available but not actively used):**
- **Audit** (`audit:`): Activity tracking and compliance reporting
- **Security** (`security:`): Security scanning and policy enforcement
- **Workflow** (`workflow:`): Build automation and pipeline orchestration
- **Enterprise** (`enterprise:`): Governance and enterprise management
- **Analytics** (`analytics:`): Usage analytics and reporting

**Interface-Based Architecture:**
```go
// Example: Access namespaces programmatically
build := mage.NewBuildNamespace()
test := mage.NewTestNamespace()
bench := mage.NewBenchNamespace()

// Execute with parameters
err := bench.DefaultWithArgs("time=50ms", "count=3")
```

**Cache Management:**
- MAGE-X uses `.mage-cache/` for build artifacts, dependency caching, and metadata
- Automatically managed - no manual intervention required
- Improves build performance through intelligent caching

### Release Process

go-broadcast uses a **tag-based release workflow**:

1. **Create Release Tag**: `magex version:bump bump=patch push`
   - Creates and pushes a new semantic version tag
   - Triggers GitHub Actions Fortress workflow automatically
2. **Automated Release**: GitHub Actions handles the actual release creation
   - Builds multi-platform binaries
   - Creates GitHub release with artifacts
   - Updates documentation and changelog

**Important**: Never use `magex release` directly - releases are handled by CI/CD after tag creation.

---

## 🚀 go-broadcast Developer Workflow Guide

This section provides go-broadcast specific workflows while maintaining `AGENTS.md` as the authoritative source for general standards.

### 🔧 Essential Development Commands

**Quick Setup:**
```bash
# Install dependencies and validate environment
magex build
```

**Core Development Workflow:**
```bash
# Standard development cycle - run before every commit
magex test           # Fast linting + unit tests
magex test:race      # Unit tests with race detector (slower)
magex bench          # Run performance benchmarks

# Quality assurance
magex lint           # Run golangci-lint
magex test:cover     # Generate and view test coverage
magex deps:audit     # Scan for security vulnerabilities
```

### 📋 MAGE-X Quick Reference

**Most Frequently Used Commands:**
```bash
# Development Cycle
magex test:coverrace             # Full CI test suite with race detection
magex lint                       # Run all 60+ linters
magex format:fix                 # Auto-fix code formatting
magex deps:audit                 # Security vulnerability scan
magex build                      # Build go-broadcast binary

# Benchmarking & Performance
magex bench                      # Default benchmarks (CI timing)
magex bench time=50ms            # Quick benchmarks for fast feedback
magex bench time=10s count=3     # Comprehensive benchmarks
magex benchquick                 # Custom: quick benchmarks only
magex benchheavy                 # Custom: intensive benchmarks (10-30min)
magex benchall                   # Custom: all benchmarks (30-60min)

# Testing Variations
magex test                       # Fast linting + unit tests
magex test:unit                  # Unit tests only (skip linting)
magex test:race                  # Race condition detection
magex test:cover                 # Coverage analysis
magex test:fuzz                  # Fuzz testing (default duration)
magex test:fuzz time=30s         # Fuzz testing with custom duration

# Version & Release Management
magex version:show               # Display current version
magex version:bump bump=patch push    # Create patch release tag (triggers CI release)
magex version:bump bump=minor push    # Create minor release tag
magex version:bump bump=major push    # Create major release tag (with confirmation)

# Dependencies & Tools
magex deps:update                # Update dependencies safely
magex deps:tidy                  # Clean up go.mod and go.sum
magex tools:update               # Update development tools
magex update:install             # Update magex itself

# Documentation
magex docs:serve                 # Serve documentation locally
magex docs:generate              # Generate package documentation
magex docs:update                # Update pkg.go.dev documentation

# Profiling & Analysis
magex bench:cpu time=30s         # CPU profiling with custom duration
magex bench:mem time=2s          # Memory profiling
magex bench:profile              # General performance profiling
```

**Parameter Syntax:**
- **Key-Value**: `magex command param=value` (e.g., `time=50ms`, `bump=patch`)
- **Flags**: `magex command flag` (e.g., `push`, `dry-run`)
- **Multiple**: `magex command param1=value1 param2=value2 flag`

### ⚡ Testing and Validation

**Unit Testing:**
```bash
# Fast testing (recommended for development)
magex test:unit   # Skip linting, run only tests
magex test:short  # Skip integration tests

# Comprehensive testing
magex test:coverrace    # Full CI test suite with race detection
magex test:cover        # Unit tests with coverage report
```

**Configuration Validation:**
```bash
# Validate go-broadcast configurations
./go-broadcast validate --config examples/minimal.yaml
./go-broadcast validate --config examples/sync.yaml

# Test dry-run functionality
./go-broadcast sync --dry-run --config examples/minimal.yaml

# Check configuration status
./go-broadcast status --config examples/minimal.yaml

# Cancel active sync operations
./go-broadcast cancel --dry-run --config examples/minimal.yaml
./go-broadcast cancel --config examples/minimal.yaml
```

### 🧪 Fuzz Testing Workflow

go-broadcast includes comprehensive fuzz testing for critical components:

**Run Fuzz Tests:**
```bash
# Run all fuzz tests (limited iterations for speed)
magex test:fuzz

# Run fuzz tests with custom duration (using MAGE-X parameter syntax)
magex test:fuzz time=30s

# Run specific fuzz tests with more iterations
go test -fuzz=FuzzConfigParsing -fuzztime=30s ./internal/config
go test -fuzz=FuzzTransformChain -fuzztime=30s ./internal/transform
go test -fuzz=FuzzRegexReplacement -fuzztime=30s ./internal/transform

# Generate new corpus entries
go run ./cmd/generate-corpus
```

**Fuzz Test Coverage:**
- **Config parsing** (`internal/config`): YAML validation, repository name validation
- **Transformations** (`internal/transform`): Variable substitution, regex replacement chains
- **GitHub API** (`internal/gh`): Response parsing and error handling
- **Git operations** (`internal/git`): Command parsing and output handling

### 📊 Performance Testing and Benchmarking

**Quick Benchmarks (CI Default):**
```bash
# Run fast benchmarks only (completes in <5 minutes)
magex bench                    # Standard CI benchmarks (default timing)
magex bench time=50ms          # Quick benchmarks for fast feedback
magex benchquick               # Custom command: quick benchmarks only

# Run specific component benchmarks
go test -bench=. -benchmem ./internal/cache
go test -bench=. -benchmem ./internal/algorithms
go test -bench=. -benchmem ./internal/config
```

**Heavy Benchmarks (Manual Execution):**
```bash
# Run intensive benchmarks (10-30 minutes)
magex benchheavy               # Custom command: Worker pools, large datasets, real-world scenarios

# Run all benchmarks (30-60 minutes)
magex benchall                 # Custom command: Quick + heavy benchmarks

# Run with custom parameters using MAGE-X syntax
magex bench time=10s count=3   # Comprehensive benchmarks with custom timing
go test -bench=. -benchmem -tags=bench_heavy -benchtime=1s ./...
```

**Benchmark Categories:**
- **Quick (CI)**: Core algorithms, basic cache operations, config parsing
- **Heavy (Manual)**: Worker pool stress tests (1000+ tasks), large directory sync, memory efficiency with large datasets, real-world scenario simulations
- **Performance profiling**: Built-in demo via `go run ./cmd/profile_demo`

**Performance Analysis:**
```bash
# Profile benchmarks with MAGE-X commands
magex bench:cpu time=30s               # CPU profiling with custom duration
magex bench:mem time=2s                # Memory profiling
magex bench:profile                    # General performance profiling

# Profile heavy benchmarks for detailed analysis (manual)
go test -bench=. -benchmem -cpuprofile=cpu.prof -tags=bench_heavy ./internal/worker
go test -bench=. -benchmem -memprofile=mem.prof -tags=bench_heavy ./internal/git

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof

# Built-in profiling demo
go run ./cmd/profile_demo    # Results saved to: ./profiles/final_demo/
```

**Note:** Heavy benchmarks are excluded from CI to prevent timeouts. Use `magex benchheavy` for intensive performance testing during development.

### 🛠️ Troubleshooting Quick Reference

**Common Development Issues:**

1. **Build Failures:**
   ```bash
   # Clean and rebuild
   magex clean
   magex build
   ```

2. **Test Failures:**
   ```bash
   # Run tests with verbose output
   magex test

   # Run specific failing test
   go test -v -run TestSpecificFunction ./internal/package
   ```

3. **Linting Errors:**
   ```bash
   # Fix formatting issues
   magex format:fix    # Apply gofumpt formatting and fmt

   # Check linting rules
   magex lint           # Run all linters
   ```

4. **Performance Issues:**
   ```bash
   # Profile memory usage
   go test -bench=. -memprofile=mem.prof ./internal/component
   go tool pprof mem.prof

   # Monitor goroutines
   go test -bench=. -trace=trace.out ./internal/worker
   go tool trace trace.out
   ```

**Debugging go-broadcast:**
```bash
# Enable debug logging
./go-broadcast sync --log-level debug --config examples/minimal.yaml

# Collect diagnostic information
./go-broadcast diagnose > diagnostics.json

# Cancel active sync operations when needed
./go-broadcast cancel --dry-run --config examples/minimal.yaml  # Preview cancellations
./go-broadcast cancel --config examples/minimal.yaml            # Cancel all active syncs
./go-broadcast cancel org/repo1 --config examples/minimal.yaml  # Cancel specific repository

# Upgrade go-broadcast when needed
go-broadcast upgrade                     # Upgrade to latest version
go-broadcast upgrade --check             # Check for updates without upgrading
go-broadcast upgrade --force             # Force upgrade even if already on latest
go-broadcast upgrade --verbose           # Show release notes after upgrade
```

**Note**: Component-specific debug flags (`--debug-git`, `--debug-api`, etc.) are planned features not yet implemented.

**Environment Issues:**
```bash
# Check GitHub authentication
gh auth status

# Verify Go environment
go version
go env GOPATH GOROOT

# Validate dependencies
go mod verify
govulncheck ./...
```

### 🚀 GitHub Pages Setup for New Repositories

If you're setting up GoFortress coverage system for a new repository or encountering GitHub Pages deployment issues, you may need to configure environment protection rules.

**Quick Setup:**
```bash
# Run the automated setup script
./.github/coverage/scripts/setup-github-pages-env.sh

# Or specify repository explicitly
./.github/coverage/scripts/setup-github-pages-env.sh owner/repo-name
```

**What the script does:**
1. Creates/configures the `github-pages` environment
2. Sets up deployment branch policies for `master`, `gh-pages`, and `dependabot/*` branches
3. Configures environment protection rules
4. Verifies the setup

**Manual Setup (if script fails):**
1. Go to your repository Settings → Environments → github-pages
2. Under "Deployment branches", select "Selected branches and tags"
3. Add deployment branch rules for:
   - `master` (for main deployments)
   - `gh-pages` (GitHub Pages default)
   - `dependabot/*` (for automated dependency updates)
4. Save the changes

**Requirements:**
- GitHub CLI (`gh`) installed and authenticated
- Repository admin permissions
- Personal Access Token with repo scope (for private repos)

**Troubleshooting:**
- **"Branch not allowed to deploy"**: Run the setup script or manually configure branch rules
- **"Environment protection rules"**: Ensure you have admin permissions to the repository
- **Script fails**: Check GitHub CLI authentication with `gh auth status`

**Verification:**
After setup, coverage reports will be available at:
`https://[owner].github.io/[repo-name]/`

### 📊 Coverage System URLs and Deployment

The GoFortress coverage system uses an **incremental deployment strategy** that preserves coverage data across different branches and pull requests.

**Coverage URLs Structure:**

**Main Branch Coverage (deployed from master):**
- Dashboard: `https://[owner].github.io/[repo-name]/`
- Report: `https://[owner].github.io/[repo-name]/coverage.html`
- Badge: `https://[owner].github.io/[repo-name]/coverage.svg`

**Branch-Specific Coverage:**
- Dashboard: `https://[owner].github.io/[repo-name]/coverage/branch/[branch-name]/`
- Report: `https://[owner].github.io/[repo-name]/coverage/branch/[branch-name]/coverage.html`
- Badge: `https://[owner].github.io/[repo-name]/coverage/branch/[branch-name]/coverage.svg`

**Pull Request Coverage:**
- Badge: `https://[owner].github.io/[repo-name]/coverage/pr/[pr-number]/coverage.svg`
- All branches index: `https://[owner].github.io/[repo-name]/branches.html`

**Important Deployment Notes:**
- The deployment is **incremental** - new deployments don't overwrite existing branch/PR data
- Each branch gets its own persistent directory under `/coverage/branch/`
- PR badges are stored under `/coverage/pr/` and persist across deployments
- The main branch deployment updates the root files while preserving all subdirectories
- A `branches.html` index is generated when deploying from the main branch

### 🪝 GoFortress Pre-commit System

The GoFortress Pre-commit System uses the external **[go-pre-commit](https://github.com/mrz1836/go-pre-commit)** tool - a production-ready, high-performance Go-native pre-commit framework that delivers 17x faster execution than traditional Python-based solutions.

**Quick Setup:**
```bash
# Install the external tool
go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@v1.0.0

# Install hooks in your repository
go-pre-commit install

# Normal development - hooks run automatically
git add .
git commit -m "feat: new feature"
# ✅ All checks passed in <2s
```

**Key Features:**
- ⚡ **17x faster execution** - <2 second commits with parallel processing
- 📦 **Zero Python dependencies** - Pure Go binary, no runtime requirements
- 🔧 **Make integration** - Wraps existing Makefile targets (fumpt, lint, mod-tidy)
- ⚙️ **Environment-driven** - All configuration via `.github/.env.base` (custom via `.env.custom`)
- 🎯 **Production ready** - Comprehensive test coverage and validation
- 🔗 **External tool** - Maintained at [github.com/mrz1836/go-pre-commit](https://github.com/mrz1836/go-pre-commit)

**Available Checks (5 MVP checks):**
1. **fumpt** - Code formatting via `magex format`
2. **lint** - Linting via `magex lint`
3. **mod-tidy** - Module tidying via `magex tidy`
4. **whitespace** - Trailing whitespace removal (built-in)
5. **eof** - End-of-file newline enforcement (built-in)

**Example Configuration in `.github/.env.base`:**
```bash
# Enable the system
ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_VERSION=v1.0.0  # Version of external tool to use

# Individual check control
GO_PRE_COMMIT_ENABLE_FUMPT=true
GO_PRE_COMMIT_ENABLE_LINT=true
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
GO_PRE_COMMIT_ENABLE_EOF=true

# Performance tuning
GO_PRE_COMMIT_PARALLEL_WORKERS=2  # Number of parallel workers
GO_PRE_COMMIT_TIMEOUT_SECONDS=120  # Timeout in seconds
GO_PRE_COMMIT_FAIL_FAST=false
```

**Development Commands:**
```bash
# Manual execution
go-pre-commit run                    # All checks on staged files
go-pre-commit run --all-files        # All checks on all files
go-pre-commit run --checks fumpt,lint  # Specific checks only
go-pre-commit run --verbose          # Debug output

# Skip functionality
SKIP=lint git commit -m "wip: work in progress"
GO_PRE_COMMIT_SKIP=all git commit -m "hotfix: critical fix"

# Status and management
go-pre-commit status --verbose       # Installation status
go-pre-commit uninstall              # Remove hooks
```

**CI/CD Integration:**
```yaml
# Automatic integration via fortress-pre-commit.yml
pre-commit:
  name: 🪝 Pre-commit Checks
  if: needs.setup.outputs.pre-commit-enabled == 'true'
  uses: ./.github/workflows/fortress-pre-commit.yml
```

The workflow automatically installs the external tool using the version specified in `GO_PRE_COMMIT_VERSION`.

**Performance Benchmarks:**
- **Total pipeline**: <2s (17x faster than baseline)
- **fumpt**: 6ms (37% faster)
- **lint**: 68ms (94% faster)
- **mod-tidy**: 110ms (53% faster)
- **Text processing**: <1ms (built-in speed)

**Troubleshooting:**
- **"go-pre-commit not found"**: Run `go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@latest`
- **"Hook already exists"**: Use `go-pre-commit install --force`
- **Slow execution**: Increase timeout with `GO_PRE_COMMIT_TIMEOUT_SECONDS=180`

**Environment Verification:**
```bash
# Verify go-pre-commit installation
go-pre-commit --version               # Check installed version
go-pre-commit run fumpt --verbose    # Test pre-commit fumpt check
go env GOPATH                        # Check GOPATH is set correctly
echo $PATH | grep "$(go env GOPATH)/bin"  # Verify GOPATH/bin in PATH
```

**📚 Complete Documentation:** [github.com/mrz1836/go-pre-commit](https://github.com/mrz1836/go-pre-commit)

### 🤖 AI Sub-Agents Team

go-broadcast includes a comprehensive team of **26 specialized AI sub-agents** designed to manage all aspects of repository lifecycle. These agents work autonomously and collaboratively to maintain code quality, security, and performance.

**Quick Overview:**
- **Core Operations** (4 agents): sync-orchestrator, config-validator, github-sync-api, directory-sync-specialist
- **Testing & QA** (5 agents): test-commander, benchmark-runner, fuzz-test-guardian, integration-test-manager, go-quality-enforcer
- **Dependencies** (3 agents): dependabot-coordinator, dependency-upgrader, breaking-change-detector
- **Performance** (2 agents): performance-profiler, benchmark-analyst
- **Security** (2 agents): security-auditor, compliance-checker
- **And more...** (11 additional agents for automation, diagnostics, documentation, and refactoring)

**Using Sub-Agents:**
```bash
# Automatic agent selection
"Fix the failing tests and improve coverage"

# Explicit agent invocation
"Use the security-auditor agent to scan for vulnerabilities"
"Have the release-manager prepare version 1.2.0"
```

**Agent Collaboration:**
- Agents work in parallel groups for efficiency (e.g., Quality Group runs 3 agents simultaneously)
- Sequential workflows for complex tasks (e.g., Release Flow: changelog → release → docs)
- Event-driven triggers activate relevant agents automatically

**Performance Targets:**
- Binary detection: 587M+ ops/sec
- Test coverage: >85%
- Security scans: <60s
- PR automation: <2s response

**📚 Complete Documentation:** [`docs/sub-agents.md`](../docs/sub-agents.md)

### 📚 Documentation Navigation

**Core Documentation:**
- [`README.md`](../README.md) - Project overview and quick start
- [`AGENTS.md`](AGENTS.md) - **Primary authority** for all development standards
- [`CONTRIBUTING.md`](CONTRIBUTING.md) - Contribution guidelines
- [`CODE_STANDARDS.md`](CODE_STANDARDS.md) - Detailed coding standards

**Technical Documentation:**
- [`docs/logging.md`](../docs/logging.md) - Complete logging documentation with quick reference and comprehensive guide
- [`docs/troubleshooting.md`](../docs/troubleshooting.md) - Enhanced troubleshooting guide with operational procedures

**Performance and Optimization:**
- [`docs/performance-guide.md`](../docs/performance-guide.md) - Complete performance guide with benchmarking, profiling, and optimization

**Configuration Examples:**
- [`examples/`](../examples) - Complete configuration examples directory
- [`examples/README.md`](../examples/README.md) - Example configurations overview
- [`examples/minimal.yaml`](../examples/minimal.yaml) - Simplest configuration
- [`examples/sync.yaml`](../examples/sync.yaml) - Comprehensive example

---

## 🔧 Code Quality Guidelines

This project uses 60+ linters via golangci-lint with strict standards. Key areas:

### Essential Linting Practices

**Security & Best Practices:**
- Use appropriate file permissions: `0600` for sensitive files, `0750` for directories
- Always check error returns: `if err := foo(); err != nil { ... }`
- Use context-aware functions: `DialContext`, `CommandContext`
- Create static error variables and wrap with context

**Code Quality:**
- Add comments to all exported functions, types, and constants
- Use `_` for intentionally unused parameters
- Avoid redefining built-in functions (`max`, `min`, etc.)
- Pre-allocate slices when size is known: `make([]Type, 0, knownSize)`

**Common Patterns:**
- Format with `gofumpt` before committing
- Use `fmt.Fprintf(w, format, args...)` for efficient string building
- Add `//nolint:linter // reason` only when necessary with clear explanation

### Running Linters

```bash
# Run all linters on main module
magex lint

# Fix common formatting issues
magex format:fix
```

The project maintains zero linter issues across all enabled linters. When adding code, follow existing patterns and address linter feedback promptly.

---

## ✅ Pre-Development Checklist

Before starting any development work:

1. **Read `AGENTS.md` thoroughly** - Understand all conventions and standards
2. **Set up development environment:**
   ```bash
   magex install:stdlib
   cd .github/pre-commit && magex build && ./gofortress-pre-commit install  # Optional but recommended
   ```
3. **Validate environment:**
   ```bash
   magex test          # Ensure all tests pass
   gh auth status     # Verify GitHub authentication
   ```
4. **Review relevant documentation** based on your planned changes

## 🎯 Development Workflow Summary

1. **Study `AGENTS.md`** - Make sure every change respects established standards
2. **Follow branch‑prefix and commit‑message standards** - Required for CI automation
3. **Never tag releases** - Only repository code‑owners handle releases
4. **Run comprehensive testing:**
   ```bash
   magex test         # Fast linting + unit tests
   magex test:race    # Race condition detection
   ```
5. **Validate go-broadcast functionality:**
   ```bash
   ./go-broadcast validate --config examples/minimal.yaml
   ./go-broadcast sync --dry-run --config examples/minimal.yaml
   ./go-broadcast cancel --dry-run --config examples/minimal.yaml
   ```

## 🚨 Important Reminders

- **`AGENTS.md` is the ultimate authority** - When in doubt, refer to it first
- **Test thoroughly** - Use both unit tests and go-broadcast validation commands
- **Follow Go conventions** - Context-first design, interface composition, no global state
- **Security first** - Run `govulncheck` and validate all external dependencies
- **Performance matters** - Use benchmarks to validate optimizations

If you encounter conflicting guidance elsewhere, `AGENTS.md` wins.
Questions or ambiguities? Open a discussion or ping a maintainer instead of guessing.

---

Happy hacking! 🚀
