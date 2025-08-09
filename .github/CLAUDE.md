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

## 🚀 go-broadcast Developer Workflow Guide

This section provides go-broadcast specific workflows while maintaining `AGENTS.md` as the authoritative source for general standards.

### 🔧 Essential Development Commands

**Quick Setup:**
```bash
# Install dependencies and validate environment
make mod-download
make install-stdlib

# Build the application locally
make build-go
# Or build directly: go build -o go-broadcast ./cmd/go-broadcast
```

**Core Development Workflow:**
```bash
# Standard development cycle - run before every commit
make test           # Fast linting + unit tests
make test-race      # Unit tests with race detector (slower)
make bench          # Run performance benchmarks

# Quality assurance
make lint           # Run golangci-lint
make coverage       # Generate and view test coverage
make govulncheck    # Scan for security vulnerabilities
```

### ⚡ Testing and Validation

**Unit Testing:**
```bash
# Fast testing (recommended for development)
make test-no-lint   # Skip linting, run only tests
make test-short     # Skip integration tests

# Comprehensive testing
make test-ci        # Full CI test suite with race detection
make test-cover     # Unit tests with coverage report
make test-all-modules-race  # Test all modules with race detection
```

**Integration Testing:**
```bash
# Phase-specific integration tests
make test-integration-complex    # Phase 1: Complex workflows
make test-integration-advanced   # Phase 2: Advanced scenarios
make test-integration-network    # Phase 3: Network edge cases
make test-integration-all        # All integration test phases
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
make test-fuzz

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

**Benchmark Execution:**
```bash
# Run all benchmarks with memory profiling
make bench

# Run specific component benchmarks
go test -bench=. -benchmem ./internal/cache
go test -bench=. -benchmem ./internal/algorithms
go test -bench=. -benchmem ./internal/transform

# Performance profiling with built-in demo
go run ./cmd/profile_demo
# Results saved to: ./profiles/final_demo/
```

**Benchmark Categories:**
- **Core algorithms**: Binary detection, content comparison, batch processing
- **Cache operations**: TTL cache performance across different hit rates
- **API simulation**: GitHub API latency simulation with various response times
- **Concurrency**: Goroutine performance across different worker pool sizes
- **Memory usage**: Allocation patterns and memory efficiency

**Performance Analysis:**
```bash
# Generate performance reports
go test -bench=. -benchmem -cpuprofile=cpu.prof ./internal/worker
go test -bench=. -benchmem -memprofile=mem.prof ./internal/git

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

### 🛠️ Troubleshooting Quick Reference

**Common Development Issues:**

1. **Build Failures:**
   ```bash
   # Clean and rebuild
   make clean-mods
   make mod-download
   make build-go
   ```

2. **Test Failures:**
   ```bash
   # Run tests with verbose output
   go test -v ./...

   # Run specific failing test
   go test -v -run TestSpecificFunction ./internal/package
   ```

3. **Linting Errors:**
   ```bash
   # Fix formatting issues
   make fumpt          # Apply gofumpt formatting
   make goimports      # Fix import statements

   # Check linting rules
   make lint-version   # Show linter version
   make lint           # Run all linters
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

**Troubleshooting Coverage Issues:**
- **404 on main badge**: Wait for next push to master branch
- **Missing branch coverage**: Ensure the branch has pushed after coverage setup
- **PR badge not showing**: Check that `COVERAGE_PR_COMMENT_ENABLED=true` in `.env.shared`

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
- ⚙️ **Environment-driven** - All configuration via `.github/.env.shared`
- 🎯 **Production ready** - Comprehensive test coverage and validation
- 🔗 **External tool** - Maintained at [github.com/mrz1836/go-pre-commit](https://github.com/mrz1836/go-pre-commit)

**Available Checks (5 MVP checks):**
1. **fumpt** - Code formatting via `make fumpt`
2. **lint** - Linting via `make lint`
3. **mod-tidy** - Module tidying via `make mod-tidy`
4. **whitespace** - Trailing whitespace removal (built-in)
5. **eof** - End-of-file newline enforcement (built-in)

**Configuration in `.github/.env.shared`:**
```bash
# Enable the system
ENABLE_PRE_COMMIT_SYSTEM=true
GO_PRE_COMMIT_VERSION=v1.0.0  # Version of external tool to use

# Individual check control
PRE_COMMIT_SYSTEM_ENABLE_FUMPT=true
PRE_COMMIT_SYSTEM_ENABLE_LINT=true
PRE_COMMIT_SYSTEM_ENABLE_MOD_TIDY=true
PRE_COMMIT_SYSTEM_ENABLE_WHITESPACE=true
PRE_COMMIT_SYSTEM_ENABLE_EOF=true

# Performance tuning
PRE_COMMIT_SYSTEM_PARALLEL_WORKERS=2  # Number of parallel workers
PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=120  # Timeout in seconds
PRE_COMMIT_SYSTEM_FAIL_FAST=false
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
PRE_COMMIT_SYSTEM_SKIP=all git commit -m "hotfix: critical fix"

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
- **"go-pre-commit not found"**: Run `go install github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@v1.0.0`
- **"Hook already exists"**: Use `go-pre-commit install --force`
- **Slow execution**: Increase timeout with `PRE_COMMIT_SYSTEM_TIMEOUT_SECONDS=180`

**Fumpt Check Failures (Tower/SourceTree Git GUIs):**
- **"fumpt check failed"**: The system uses pinned gofumpt version from `.env.shared`
- **"make: gofumpt: No such file or directory"**: Run `make fumpt` once manually to install correct version
- **PATH issues in git GUIs**: The tool automatically manages GOPATH/bin in PATH during execution
- **Version conflicts**: Ensure `PRE_COMMIT_SYSTEM_FUMPT_VERSION=v0.7.0` is set in `.env.shared`

**Environment Verification:**
```bash
# Verify go-pre-commit installation
go-pre-commit --version               # Check installed version
make fumpt                           # Install and run gofumpt
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
- **Performance** (3 agents): performance-profiler, benchmark-analyst, coverage-maintainer
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
make lint

# Run linters on all modules (including coverage)
make lint-all-modules

# Run tests on all modules with race detection
make test-all-modules-race

# Fix common formatting issues
make fumpt
goimports -w .
```

The project maintains zero linter issues across all enabled linters. When adding code, follow existing patterns and address linter feedback promptly.

---

## ✅ Pre-Development Checklist

Before starting any development work:

1. **Read `AGENTS.md` thoroughly** - Understand all conventions and standards
2. **Set up development environment:**
   ```bash
   make mod-download
   make install-stdlib
   cd .github/pre-commit && make build && ./gofortress-pre-commit install  # Optional but recommended
   ```
3. **Validate environment:**
   ```bash
   make test          # Ensure all tests pass
   gh auth status     # Verify GitHub authentication
   ```
4. **Review relevant documentation** based on your planned changes

## 🎯 Development Workflow Summary

1. **Study `AGENTS.md`** - Make sure every change respects established standards
2. **Follow branch‑prefix and commit‑message standards** - Required for CI automation
3. **Never tag releases** - Only repository code‑owners handle releases
4. **Run comprehensive testing:**
   ```bash
   make test         # Fast linting + unit tests
   make test-race    # Race condition detection
   make govulncheck  # Security vulnerability scan
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
