# GoFortress Pre-commit Hook Manager - Go-Native Implementation Plan

## Executive Summary

This document outlines a comprehensive plan to replace the Python-based pre-commit framework with a self-hosted, Go-native git hooks manager integrated directly into the GoFortress CI/CD pipeline. Built entirely in Go as a **bolt-on solution completely encapsulated within the `.github` folder**, this solution eliminates Python dependencies, provides better performance through parallel execution, and maintains simplicity while being fully portable between projects.

**Key Architecture Decision**: The entire pre-commit system resides within `.github/hooks/` making it a portable, self-contained bolt-on that can be copied to any repository without polluting the main codebase.

## Vision Statement

Create a best-in-class Go-native git hooks manager that embodies Go's philosophy of simplicity and performance:
- **Go-First Design**: Built by Go developers, for Go developers
- **Single Binary**: One compiled tool that does everything - no runtime dependencies
- **Lightning Fast**: Leverage Go's concurrency for parallel hook execution
- **Professional Quality**: Clean CLI interface with helpful error messages
- **Zero Dependencies**: Pure Go implementation with minimal external packages
- **Developer Friendly**: Simple YAML configuration with sensible defaults
- **Bolt-On Architecture**: Completely self-contained within `.github/hooks/`
- **Portable**: Can be copied to any repository as a complete unit
- **Non-Invasive**: Does not pollute the main repository structure

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     GoFortress Git Hooks System                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────────────────┐               │
│  │ Git Command  │───▶│ gofortress-hooks         │               │
│  │ (commit/push)│    │ (Single Go Binary)       │               │
│  └──────────────┘    │                          │               │
│                      │ ├─ install               │               │
│                      │ ├─ run                   │               │
│                      │ ├─ list                  │               │
│                      │ ├─ validate              │               │
│                      │ └─ version               │               │
│                      └──────────┬───────────────┘               │
│                                 │                               │
│                                 ▼                               │
│                      ┌──────────────────────┐                   │
│                      │ Parallel Execution   │                   │
│                      │ Engine (Goroutines)  │                   │
│                      └──────────────────────┘                   │
└─────────────────────────────────────────────────────────────────┘

Directory Structure:
.github/
└── hooks/
    ├── cmd/
    │   └── gofortress-hooks/     # Main CLI tool
    ├── internal/                  # Internal packages
    ├── config/
    │   └── hooks.yaml            # Hook configuration
    ├── scripts/                  # External hook scripts
    └── README.md                 # Documentation
```

## Implementation Roadmap

Each phase is designed to be completed in a single Claude Code session with clear deliverables and verification steps. The phases build upon each other, so they should be completed in order.

**IMPORTANT**: 
1. **All files must be created within the `.github` directory** - the entire hooks system is self-contained in `.github/hooks/`
2. After completing each phase, update the status tracking document at `plans/plan-10-status.md` with:
   - Mark the phase as completed (✓)
   - Note any deviations from the plan
   - Record actual metrics and timings
   - Document any issues encountered
   - Update the "Next Steps" section

### Phase 1: Foundation & Configuration (Session 1)
**Objective**: Establish infrastructure for the new hooks system

**Implementation Steps:**
1. Add hook system environment variables to `.github/.env.shared`
2. Create directory structure for hooks system
3. Create initial configuration schema
4. Add labels for hook-related PRs

**Files to Create/Modify:**
- `.github/.env.shared` - Add new variables
- `.github/hooks/` - Create directory structure
- `.github/hooks/README.md` - Initial documentation
- `.github/hooks/config/hooks.yaml` - Example configuration
- `.github/labels.yml` - Add hook-system label
- `.github/dependabot.yml` - Add hooks tool Go module monitoring
- `plans/plan-10-status.md` - Create status tracking document

**Verification Steps:**
```bash
# 1. Verify environment variables are added
grep "ENABLE_GO_HOOKS" .github/.env.shared

# 2. Verify directory structure
ls -la .github/hooks/

# 3. Verify configuration schema
yamllint .github/hooks/config/hooks.yaml

# 4. Run existing pre-commit for baseline
pre-commit run --all-files

# 5. Check labels
grep "hook-system" .github/labels.yml
```

**Success Criteria:**
- ✅ All hook environment variables present in .env.shared
- ✅ Directory structure created with proper permissions
- ✅ Configuration schema documented and valid
- ✅ Labels and dependabot configuration updated

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 1 as completed (✓)
- Record actual implementation details
- Note any configuration changes made

#### 1.1 Environment Configuration Enhancement
Add to `.github/.env.shared`:

```bash
# ───────────────────────────────────────────────────────────────────────────────
# ENV: Go Hooks System Configuration
# ───────────────────────────────────────────────────────────────────────────────
ENABLE_GO_HOOKS=false                           # Enable Go-native hooks system (replaces pre-commit)
HOOKS_PARALLEL_EXECUTION=true                   # Run hooks in parallel
HOOKS_MAX_WORKERS=0                             # Max parallel workers (0 = NumCPU)
HOOKS_TIMEOUT_SECONDS=300                       # Global timeout for all hooks
HOOKS_FAIL_FAST=false                           # Stop on first hook failure
HOOKS_VERBOSE_OUTPUT=false                      # Show detailed hook output
HOOKS_COLOR_OUTPUT=true                         # Use colored terminal output
HOOKS_PROGRESS_BAR=true                         # Show progress bar during execution
HOOKS_SKIP_ON_CI=false                          # Skip hooks in CI environment
HOOKS_ALLOW_SKIP=true                           # Allow SKIP environment variable
HOOKS_CONFIG_FILE=.github/hooks/config/hooks.yaml  # Path to hooks configuration
HOOKS_CACHE_ENABLED=true                        # Cache hook results
HOOKS_CACHE_DIR=.github/hooks/.cache            # Cache directory
HOOKS_CACHE_TTL_MINUTES=60                      # Cache time-to-live
HOOKS_LOG_LEVEL=info                            # Log level: debug, info, warn, error
HOOKS_LOG_FILE=                                 # Optional log file path
HOOKS_METRICS_ENABLED=true                      # Track hook execution metrics
HOOKS_UPDATE_CHECK=true                         # Check for hook updates
HOOKS_AUTO_INSTALL=true                         # Auto-install hooks on first run
HOOKS_PROTECT_BRANCHES=main,master,development  # Branches with strict hook enforcement
HOOKS_CUSTOM_SCRIPTS_DIR=.github/hooks/scripts  # Directory for custom scripts
HOOKS_DRY_RUN=false                             # Enable dry-run mode for testing
HOOKS_STRICT_MODE=false                         # Fail on any hook warning

# Hook-specific Configuration
HOOKS_GO_FMT_ENABLED=true                       # Enable gofmt hook
HOOKS_GO_IMPORTS_ENABLED=true                   # Enable goimports hook
HOOKS_GO_VET_ENABLED=true                       # Enable go vet hook
HOOKS_GOLANGCI_LINT_ENABLED=true                # Enable golangci-lint hook
HOOKS_GO_MOD_TIDY_ENABLED=true                  # Enable go mod tidy hook
HOOKS_GO_TEST_ENABLED=false                     # Enable go test hook (pre-push only)
HOOKS_GITLEAKS_ENABLED=true                     # Enable gitleaks secret scanning
HOOKS_TRAILING_WHITESPACE_ENABLED=true          # Enable trailing whitespace fixer
HOOKS_END_OF_FILE_FIXER_ENABLED=true            # Enable end-of-file fixer
HOOKS_CHECK_MERGE_CONFLICT_ENABLED=true         # Check for merge conflict markers
HOOKS_REVIVE_ENABLED=true                       # Enable revive for comment style
HOOKS_COMMIT_MSG_ENABLED=true                   # Enable commit message validation
HOOKS_FILE_SIZE_LIMIT_ENABLED=true              # Check for large files
HOOKS_FILE_SIZE_LIMIT_MB=10                     # Max file size in MB

# Performance Tuning
HOOKS_BATCH_SIZE=50                             # Files to process per batch
HOOKS_DEBOUNCE_MS=100                           # Debounce time for file watchers
HOOKS_MAX_FILE_SIZE_BYTES=10485760              # Skip hooks for files larger than this
HOOKS_EXCLUDE_PATTERNS=vendor/,*.min.js         # Global exclude patterns
```

#### 1.2 Directory Structure Creation - Encapsulated Architecture
```bash
.github/
├── hooks/                         # Self-contained hooks system (bolt-on)
│   ├── cmd/
│   │   └── gofortress-hooks/      # Main CLI tool
│   │       ├── main.go            # Entry point with subcommands
│   │       ├── go.mod             # Separate Go module
│   │       ├── go.sum
│   │       └── cmd/               # Command implementations
│   │           ├── root.go        # Root command setup
│   │           ├── install.go     # Install git hooks
│   │           ├── run.go         # Run hooks
│   │           ├── list.go        # List available hooks
│   │           ├── validate.go    # Validate configuration
│   │           └── version.go     # Version command
│   ├── internal/                  # Internal packages (Go conventions)
│   │   ├── config/                # Configuration handling
│   │   │   ├── config.go
│   │   │   ├── config_test.go
│   │   │   └── schema.go          # YAML schema validation
│   │   ├── runner/                # Hook execution engine
│   │   │   ├── runner.go
│   │   │   ├── runner_test.go
│   │   │   ├── parallel.go       # Parallel execution
│   │   │   └── cache.go           # Result caching
│   │   ├── installer/             # Git hook installation
│   │   │   ├── installer.go
│   │   │   └── installer_test.go
│   │   ├── hooks/                 # Built-in hook implementations
│   │   │   ├── hook.go            # Hook interface
│   │   │   ├── go_fmt.go         # Go formatter
│   │   │   ├── go_imports.go     # Import organizer
│   │   │   ├── go_vet.go         # Go vet
│   │   │   ├── golangci_lint.go  # Linter
│   │   │   ├── go_mod_tidy.go    # Module tidy
│   │   │   ├── gitleaks.go       # Secret scanner
│   │   │   ├── whitespace.go     # Whitespace fixer
│   │   │   ├── eof_fixer.go      # EOF fixer
│   │   │   ├── merge_conflict.go # Conflict checker
│   │   │   └── commit_msg.go     # Commit validator
│   │   └── git/                   # Git integration
│   │       ├── client.go
│   │       └── diff.go            # Get changed files
│   ├── config/
│   │   └── hooks.yaml             # Hook configuration
│   ├── scripts/                   # External hook scripts
│   │   ├── check-large-files.sh  # Check for accidentally committed large files
│   │   └── validate-yaml.sh      # YAML validation script
│   ├── templates/                 # Git hook templates
│   │   ├── pre-commit             # Pre-commit hook
│   │   ├── pre-push              # Pre-push hook
│   │   └── commit-msg            # Commit message hook
│   ├── .gitignore                # Ignore cache and builds
│   ├── Makefile                  # Build commands
│   └── README.md                 # Hooks system documentation
├── .env.shared                   # Environment variables
└── workflows/
    └── fortress-hooks.yml        # Hooks CI workflow
```

**Key Benefits of This Structure:**
- ✅ **Complete Encapsulation**: Everything hooks-related lives in `.github/hooks/`
- ✅ **Portable**: The entire `hooks/` folder can be copied to any repository
- ✅ **Separate Module**: Hooks tool has its own `go.mod` for isolated dependencies
- ✅ **Clean Separation**: Main repository remains uncluttered
- ✅ **Self-Documenting**: Hooks system includes its own README.md

#### 1.3 Hook Configuration Schema
Create `.github/hooks/config/hooks.yaml`:
```yaml
# GoFortress Hooks Configuration
# This file defines all git hooks and their behavior
# Aligned with fortress workflows and AGENTS.md conventions

version: 1.0

# Global settings apply to all hooks
global:
  parallel: true              # Run hooks in parallel
  fail_fast: false           # Continue running hooks after failure
  verbose: false             # Show detailed output
  exclude_patterns:          # Global file exclusions
    - vendor/
    - "*.min.js"
    - "*.generated.go"
    - third_party/
    - testdata/

# Hook definitions
hooks:
  pre-commit:
    # Go-specific hooks (using make commands from fortress workflows)
    - id: go-fumpt
      name: "Go Format (fumpt)"
      description: "Format Go source code with gofumpt"
      enabled: true
      files: '\.go$'
      command: make
      args: [fumpt]
      pass_filenames: false  # make fumpt handles all files
      parallel_safe: false   # make command handles its own parallelism
      
    - id: go-lint
      name: "Go Linter"
      description: "Run golangci-lint via make"
      enabled: true
      files: '\.go$'
      command: make
      args: [lint]
      pass_filenames: false  # make lint handles all files
      timeout: 5m
      
    - id: go-vet
      name: "Go Vet"
      description: "Run go vet static analysis"
      enabled: true
      files: '\.go$'
      command: make
      args: [vet-parallel]
      pass_filenames: false
      
    - id: go-mod-tidy
      name: "Go Mod Tidy"
      description: "Ensure go.mod is tidy"
      enabled: true
      files: '^go\.mod$|^go\.sum$'
      command: make
      args: [mod-tidy]
      pass_filenames: false
      
    # General hooks
    - id: trailing-whitespace
      name: "Trim Trailing Whitespace"
      description: "Remove trailing whitespace"
      enabled: true
      files: '.*'
      exclude: '\.(md|markdown)$'
      builtin: true  # Use built-in implementation
      
    - id: end-of-file-fixer
      name: "Fix End of Files"
      description: "Ensure files end with newline"
      enabled: true
      files: '.*'
      exclude: '\.(jpg|png|gif|ico|bin)$'
      builtin: true  # Use built-in implementation
      
    - id: check-merge-conflict
      name: "Check Merge Conflicts"
      description: "Check for merge conflict markers"
      enabled: true
      files: '.*'
      builtin: true
      
    # Security hooks
    - id: gitleaks
      name: "Secret Scanner"
      description: "Scan for secrets and credentials"
      enabled: true
      files: '.*'
      command: gitleaks
      args: [detect, --verbose, --no-git]
      pass_filenames: false
      stage_on_error: false  # Don't stage files if secrets found
      
    - id: govulncheck
      name: "Vulnerability Scanner"
      description: "Scan for known vulnerabilities"
      enabled: true
      files: '^go\.mod$|^go\.sum$'
      command: make
      args: [govulncheck]
      pass_filenames: false
      
    # Comment style enforcement
    - id: revive-comments
      name: "Check Go Comments"
      description: "Enforce GoDoc comment style from AGENTS.md"
      enabled: true
      files: '\.go$'
      command: revive
      args: [-config, .revive.toml]
      pass_filenames: true
      continue_on_error: false

  pre-push:
    - id: go-test
      name: "Go Tests"
      description: "Run Go tests"
      enabled: true
      command: make
      args: [test-no-lint]  # Tests without linting (already done in pre-commit)
      pass_filenames: false
      timeout: 10m
      
    - id: go-test-race
      name: "Race Detection"
      description: "Run tests with race detector"
      enabled: true
      command: make
      args: [test-race]
      pass_filenames: false
      timeout: 15m
      
  commit-msg:
    - id: conventional-commits
      name: "Conventional Commits"
      description: "Enforce commit format from AGENTS.md"
      enabled: true
      builtin: true
      args:
        # Types from AGENTS.md
        types: [feat, fix, docs, test, refactor, chore, build, ci]
        # Common scopes in the project
        scopes: [api, cli, config, hooks, deps, internal, cmd]
        require_scope: false
        max_length: 50  # Short description ≤ 50 chars
        imperative: true  # Enforce imperative mood

# Hook-specific overrides for CI environments
ci_overrides:
  pre-commit:
    - id: go-lint
      timeout: 10m  # More time in CI
    - id: go-test-race
      enabled: false  # Race tests run in separate CI job
```

#### 1.4 Label Configuration
Add to `.github/labels.yml`:
```yaml
- name: "hook-system"
  description: "Git hooks system related"
  color: 4a5aba
```

#### 1.5 Dependabot Configuration for Hooks Tool
Since the hooks tool will be a separate Go module in `.github/hooks/cmd/gofortress-hooks/`, we need to add it to dependabot monitoring.

Add to `.github/dependabot.yml` after the main gomod entry:
```yaml
  # ──────────────────────────────────────────────────────────────
  # 1c. Hooks Tool Go Module (.github/hooks/cmd/gofortress-hooks)
  # ──────────────────────────────────────────────────────────────
  - package-ecosystem: "gomod"
    directory: "/.github/hooks/cmd/gofortress-hooks"
    target-branch: "master"
    schedule:
      interval: "weekly"
      day: "monday"
      time: "09:00"
      timezone: "America/New_York"
    allow:
      - dependency-type: "direct"
    groups:
      hooks-deps:
        patterns:
          - "*"
        update-types: ["minor", "patch"]
    open-pull-requests-limit: 5
    assignees: ["mrz1836"]
    labels: ["chore", "dependencies", "gomod", "hook-system"]
    commit-message:
      prefix: "chore"
      include: "scope"
```

This ensures the hooks tool's dependencies (cobra, YAML parser, etc.) stay up-to-date and secure.

### Phase 2: Core Hooks Engine (Session 2)
**Objective**: Build the Go-native hooks processing tool with comprehensive testing

**Implementation Steps:**
1. Create main CLI application with cobra for subcommands
2. Implement configuration parser (`internal/config`)
3. Implement hook runner with parallel execution (`internal/runner`)
4. Implement built-in hooks (`internal/hooks`)
5. Create git integration (`internal/git`)
6. Write comprehensive unit tests with >90% coverage
7. Add benchmarks for performance-critical paths
8. Set up proper error handling with context

**Files to Create (Encapsulated Structure):**
```
.github/hooks/
├── cmd/
│   └── gofortress-hooks/
│       ├── main.go              # CLI entry point
│       ├── go.mod               # Separate Go module
│       ├── go.sum               # Module dependencies
│       ├── cmd/                 # Command implementations
│       │   ├── root.go          # Root command setup
│       │   ├── install.go       # Install hooks command
│       │   ├── run.go           # Run hooks command
│       │   ├── list.go          # List hooks command
│       │   ├── validate.go      # Validate config command
│       │   └── version.go       # Version command
│       └── cmd_test.go          # CLI integration tests
├── internal/
│   ├── config/
│   │   ├── config.go            # Configuration parser
│   │   ├── config_test.go       # Unit tests
│   │   ├── schema.go            # Schema validation
│   │   └── testdata/            # Test fixtures
│   ├── runner/
│   │   ├── runner.go            # Hook execution engine
│   │   ├── runner_test.go       # Unit tests
│   │   ├── runner_bench_test.go # Benchmarks
│   │   ├── parallel.go          # Parallel execution
│   │   └── cache.go             # Result caching
│   ├── hooks/
│   │   ├── hook.go              # Hook interface
│   │   ├── registry.go          # Hook registry
│   │   ├── builtin_test.go      # Tests for all hooks
│   │   └── ... (individual hook files)
│   └── git/
│       ├── client.go            # Git operations
│       ├── client_test.go       # Unit tests
│       └── diff.go              # Changed files detection
└── Makefile                     # Build commands
```

**Go Module Dependencies (.github/hooks/cmd/gofortress-hooks/go.mod):**
```go
module github.com/mrz1836/go-broadcast/hooks

go 1.24

require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.2
    gopkg.in/yaml.v3 v3.0.1
    github.com/stretchr/testify v1.8.4
    github.com/fatih/color v1.16.0
    github.com/schollz/progressbar/v3 v3.14.1
)
```

**Key Benefits of Separate Module:**
- ✅ **Isolated Dependencies**: Hooks tool dependencies don't affect main project
- ✅ **Version Independence**: Can update hooks tool dependencies separately
- ✅ **Reduced Complexity**: Main go.mod stays clean
- ✅ **Security**: Easier to audit hooks tool dependencies separately

**Verification Steps:**
```bash
# 1. Build the tool (from encapsulated location)
cd .github/hooks/cmd/gofortress-hooks
go build -o gofortress-hooks

# 2. Run all tests with coverage
cd .github/hooks
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out -o coverage.html

# 3. Verify test coverage is >90%
go tool cover -func=coverage.out | grep total | awk '{print $3}'

# 4. Run benchmarks
go test -bench=. -benchmem ./internal/...

# 5. Run linting
cd cmd/gofortress-hooks
golangci-lint run ./...

# 6. Test CLI commands
./gofortress-hooks --help
./gofortress-hooks list
./gofortress-hooks validate --config ../../config/hooks.yaml

# 7. Run race detector tests
go test -race ./...

# 8. Check for vulnerabilities
govulncheck ./...
```

**Success Criteria:**
- ✅ Single Go binary compiles without errors
- ✅ All packages follow Go project layout standards
- ✅ Configuration parser correctly processes YAML with validation
- ✅ Hook runner executes hooks in parallel with proper concurrency control
- ✅ All tests pass with >90% code coverage
- ✅ Benchmarks show performance meets targets (<100ms overhead)
- ✅ No race conditions detected
- ✅ Zero security vulnerabilities from govulncheck
- ✅ All code passes golangci-lint with project settings
- ✅ Context propagation throughout call stack
- ✅ Proper error wrapping and handling

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 2 as completed (✓)
- Document performance metrics
- Note any design decisions or changes

### Phase 3: Hook Implementations (Session 3)
**Objective**: Implement all built-in hooks with make command integration

**Implementation Steps:**
1. Implement make command wrapper hooks (fumpt, lint, vet-parallel, mod-tidy)
2. Implement general hooks (whitespace, EOF, merge conflicts)
3. Implement security hooks (gitleaks, govulncheck via make)
4. Create hook registry with make command awareness
5. Add hook-specific tests and benchmarks
6. Implement caching for expensive operations (especially for make lint)

**Files to Create/Modify:**
- Implement all hooks in `internal/hooks/`
- Create make command wrapper in `internal/hooks/make_wrapper.go`
- Create comprehensive tests for each hook
- Add performance benchmarks
- Create utility scripts in `scripts/`

**Testing Each Hook:**
```bash
# Test individual hooks (using make commands)
./gofortress-hooks run go-fumpt --verbose
./gofortress-hooks run go-lint --verbose
./gofortress-hooks run go-vet --verbose

# Test security hooks
./gofortress-hooks run gitleaks --verbose
./gofortress-hooks run govulncheck --verbose

# Test parallel execution
./gofortress-hooks run pre-commit --parallel --verbose

# Test with dry-run
./gofortress-hooks run pre-commit --dry-run

# Test make command integration
./gofortress-hooks run --debug-make go-lint  # Shows make command execution
```

**Verification Steps:**
```bash
# 1. Test each hook individually (with make commands)
for hook in go-fumpt go-lint go-vet go-mod-tidy govulncheck; do
    ./gofortress-hooks run $hook --verbose
done

# 2. Verify make command execution
./gofortress-hooks run go-lint --debug 2>&1 | grep "Executing: make lint"

# 3. Benchmark hook performance
go test -bench=BenchmarkHook ./internal/hooks/...
go test -bench=BenchmarkMakeWrapper ./internal/hooks/...

# 4. Test hook caching (especially important for make lint)
time ./gofortress-hooks run go-lint --all-files
time ./gofortress-hooks run go-lint --all-files  # Should use cache

# 5. Test that hooks match CI behavior
make lint  # Direct make command
./gofortress-hooks run go-lint  # Should produce identical output
```

**Success Criteria:**
- ✅ All hooks produce same results as fortress workflows
- ✅ Make commands properly integrated and wrapped
- ✅ General hooks handle edge cases correctly
- ✅ Security hooks (gitleaks, govulncheck) detect issues
- ✅ Hooks match exact CI/CD behavior
- ✅ Performance: <2s for typical commit
- ✅ Caching reduces repeated operations by >50%
- ✅ Make command output properly captured and displayed

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 3 as completed (✓)
- Record hook performance benchmarks
- Document any compatibility issues

### Phase 4: Git Integration & Installation (Session 4)
**Objective**: Seamlessly integrate with git workflow

**Implementation Steps:**
1. Create git hook installer with template generation
2. Implement smart hook detection (detect existing hooks)
3. Create uninstaller with backup/restore
4. Add support for multiple hook paths
5. Implement hook skip functionality (SKIP env var)
6. Create CI detection and behavior adjustment

**Files to Create/Modify:**
- `internal/installer/` - Installation logic
- `templates/` - Git hook templates
- Update CLI commands for installation

**Installation Process:**
```bash
# Install hooks (sets core.hooksPath)
./gofortress-hooks install

# Install with custom path
./gofortress-hooks install --path .git/hooks

# Install specific hooks only
./gofortress-hooks install --hooks pre-commit,pre-push

# Uninstall and restore original hooks
./gofortress-hooks uninstall --restore
```

**Verification Steps:**
```bash
# 1. Test installation
./gofortress-hooks install --verbose
git config core.hooksPath

# 2. Verify hooks are executable
ls -la .github/hooks/templates/
test -x .github/hooks/templates/pre-commit

# 3. Test actual git commit
echo "test" > test.txt
git add test.txt
git commit -m "test: hook execution"

# 4. Test skip functionality
SKIP=golangci-lint git commit -m "test: skip linter"

# 5. Test in CI environment
CI=true ./gofortress-hooks run pre-commit

# 6. Test uninstall
./gofortress-hooks uninstall
git config core.hooksPath  # Should be empty
```

**Success Criteria:**
- ✅ Hooks install with single command
- ✅ Existing hooks backed up safely
- ✅ Git commands trigger hooks correctly
- ✅ SKIP environment variable works
- ✅ CI environment detected and handled
- ✅ Uninstall restores original state

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 4 as completed (✓)
- Document installation process
- Note any git version compatibility issues

### Phase 5: CI/CD & Workflow Integration (Session 5)
**Objective**: Integrate hooks system with CI/CD pipeline

**Implementation Steps:**
1. Create fortress-hooks.yml workflow
2. Update existing workflows to use new system
3. Add workflow documentation
4. Test CI/CD integration
5. Add performance monitoring
6. Create deployment scripts

**Files to Create/Modify:**
- `.github/workflows/fortress-hooks.yml` - New workflow
- Update fortress-test-suite.yml
- Create CI/CD integration guide
- Add performance monitoring scripts

**CI/CD Integration:**
```bash
# Validate hooks configuration
./gofortress-hooks validate --config hooks.yaml

# Run hooks in CI environment
./gofortress-hooks run --ci --all-files

# Generate performance report
./gofortress-hooks report --format json > hooks-report.json

# Check hook compliance
./gofortress-hooks check --strict
```

**Verification Steps:**
```bash
# 1. Test CI integration
act -W .github/workflows/fortress-hooks.yml

# 2. Verify workflow execution
./gofortress-hooks run --ci --verbose

# 3. Test performance monitoring
./gofortress-hooks run --profile --all-files

# 4. Validate configuration
./gofortress-hooks validate --strict

# 5. Check deployment
./gofortress-hooks version
./gofortress-hooks check --self-test
```

**Success Criteria:**
- ✅ CI workflow passes all checks
- ✅ Hooks run successfully in CI environment
- ✅ Performance monitoring operational
- ✅ Documentation complete
- ✅ All workflows updated
- ✅ Deployment automated

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 5 as completed (✓)
- Document CI/CD integration details
- Record performance metrics

### Phase 6: Developer Experience & Polish (Session 6)
**Objective**: Create exceptional developer experience

**Implementation Steps:**
1. Add colored output with progress indicators
2. Implement helpful error messages with fixes
3. Create interactive mode for hook selection
4. Add performance profiling and reporting
5. Create comprehensive documentation
6. Add shell completions (bash, zsh, fish)

**Files to Create/Modify (all within .github directory):**
- Add progress bars to runner
- Enhance error messages throughout
- Create shell completion generators
- Write user and developer documentation

**Documentation Requirements:**
1. **README.md Update** (brief getting started section):
   - Add a "Git Hooks System" section after "Coverage System Features"
   - Include quick installation command: `cd .github/hooks && make install`
   - Link to detailed documentation: `docs/hooks-system.md`
   - Keep it concise (5-10 lines max)

2. **docs/hooks-system.md** (comprehensive guide):
   - Full installation and configuration guide
   - Architecture overview and design decisions
   - Hook development guide
   - CI/CD integration guide
   - Performance tuning and troubleshooting
   - API reference for custom hooks
   - Examples and best practices

**Developer Features:**
```bash
# Interactive mode
./gofortress-hooks run --interactive

# Detailed performance report
./gofortress-hooks run pre-commit --profile

# Fix mode
./gofortress-hooks run --fix-only

# Verbose debugging
./gofortress-hooks run --debug --trace

# Shell completions
./gofortress-hooks completion bash > hooks-completion.bash
```

**Verification Steps:**
```bash
# 1. Test colored output
./gofortress-hooks run pre-commit --color=always

# 2. Test progress indicators
./gofortress-hooks run pre-commit --all-files

# 3. Test error messages
echo "invalid go code" > test.go
./gofortress-hooks run go-fmt --files test.go

# 4. Test interactive mode
./gofortress-hooks run --interactive

# 5. Test shell completions
source <(./gofortress-hooks completion bash)
./gofortress-hooks <TAB><TAB>

# 6. Test performance profiling
./gofortress-hooks run pre-commit --profile > profile.txt
```

**Success Criteria:**
- ✅ Beautiful, informative output
- ✅ Clear error messages with solutions
- ✅ Interactive mode intuitive
- ✅ Performance data actionable
- ✅ Documentation comprehensive
- ✅ Shell completions work correctly

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 6 as completed (✓)
- Include screenshots of output
- Document user feedback
- Confirm README.md was updated with brief getting started
- Confirm docs/hooks-system.md was created with full documentation

## Configuration Examples

### Basic Configuration (`.github/hooks/config/hooks.yaml`)
```yaml
version: 1.0

hooks:
  pre-commit:
    - id: go-fmt
      name: "Format Go Code"
      files: '\.go$'
      command: gofmt
      args: [-w]
      pass_filenames: true
      
    - id: trailing-whitespace
      name: "Trim Whitespace"
      builtin: true
```

### Advanced Configuration
```yaml
version: 1.0

global:
  parallel: true
  fail_fast: false
  exclude_patterns:
    - vendor/
    - .git/
    - "*.pb.go"
    - third_party/
    - testdata/

hooks:
  pre-commit:
    - id: go-fumpt
      name: "Format with fumpt"
      files: '\.go$'
      command: make
      args: [fumpt]
      pass_filenames: false
      timeout: 30s
      env:
        GOFLAGS: "-mod=readonly"
      
    - id: go-lint
      name: "Run Linters"
      files: '\.go$'
      command: make
      args: [lint]
      pass_filenames: false
      stages: [pre-commit, pre-push]
      timeout: 5m
      
  pre-push:
    - id: go-test-full
      name: "Run Full Test Suite"
      command: make
      args: [test-ci]  # Full CI test suite with coverage
      pass_filenames: false
      timeout: 15m
      env:
        CGO_ENABLED: "1"
    
    - id: bench-quick
      name: "Quick Benchmarks"
      command: make
      args: [bench-quick]
      pass_filenames: false
      enabled: false  # Optional, enable for performance-critical changes
```

## Installation & Setup

### Quick Start
```bash
# Build the hooks system
cd .github/hooks/cmd/gofortress-hooks
go build -o gofortress-hooks

# Install hooks
./gofortress-hooks install

# Verify installation
./gofortress-hooks check
```

### Configuration
```bash
# Validate configuration
./gofortress-hooks validate --config hooks.yaml

# List available hooks
./gofortress-hooks list

# Run specific hooks
./gofortress-hooks run go-fmt --all-files
```

### CI/CD Setup
```bash
# Add to CI workflow
- name: Run Hooks
  run: |
    cd .github/hooks/cmd/gofortress-hooks
    go build -o gofortress-hooks
    ./gofortress-hooks run --ci --all-files
```

## Performance Optimization

### Parallel Execution
```go
// internal/runner/parallel.go
type ParallelRunner struct {
    workers   int
    taskQueue chan Task
    results   chan Result
}

func (r *ParallelRunner) Run(ctx context.Context, hooks []Hook, files []string) error {
    // Create worker pool
    for i := 0; i < r.workers; i++ {
        go r.worker(ctx)
    }
    
    // Distribute tasks
    for _, hook := range hooks {
        if hook.ParallelSafe {
            r.taskQueue <- Task{Hook: hook, Files: files}
        }
    }
}
```

### Intelligent Caching
```go
// internal/runner/cache.go
type Cache struct {
    store map[string]CacheEntry
    ttl   time.Duration
}

type CacheEntry struct {
    Result    Result
    Timestamp time.Time
    FileHash  string
}

func (c *Cache) Get(hook Hook, file string) (Result, bool) {
    key := fmt.Sprintf("%s:%s", hook.ID, file)
    entry, exists := c.store[key]
    
    if !exists || time.Since(entry.Timestamp) > c.ttl {
        return Result{}, false
    }
    
    // Verify file hasn't changed
    if hash := hashFile(file); hash != entry.FileHash {
        return Result{}, false
    }
    
    return entry.Result, true
}
```

## Testing Strategy

### Unit Tests
```go
// internal/hooks/go_fmt_test.go
func TestGoFmtHook(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "unformatted code",
            input:    "package main\nimport \"fmt\"\nfunc main(){fmt.Println(\"hello\")}",
            expected: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}",
        },
    }
    
    hook := NewGoFmtHook()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := hook.Run(context.Background(), tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests
```go
// tests/integration/hooks_test.go
func TestFullHookExecution(t *testing.T) {
    // Create test repository
    repo := createTestRepo(t)
    defer repo.Cleanup()
    
    // Add test files
    repo.AddFile("main.go", unformattedCode)
    repo.AddFile("test.txt", "trailing whitespace   \n")
    
    // Run hooks
    cmd := exec.Command("gofortress-hooks", "run", "pre-commit")
    cmd.Dir = repo.Path
    
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)
    
    // Verify fixes applied
    content := repo.ReadFile("main.go")
    assert.Contains(t, content, "func main() {")
    
    content = repo.ReadFile("test.txt")
    assert.Equal(t, "trailing whitespace\n", content)
}
```

## Success Metrics

### Performance
- **Hook Execution**: <2s for typical commit (10-20 files)
- **Parallel Speedup**: 3-5x faster than sequential
- **Cache Hit Rate**: >80% for unchanged files
- **Memory Usage**: <50MB for large repositories

### Reliability
- **Test Coverage**: >90% across all packages
- **Zero Panics**: Graceful error handling
- **CI Integration**: 100% workflow compatibility
- **Cross-Platform**: Linux, macOS, Windows support

### Developer Experience
- **Installation Time**: <10 seconds
- **Migration Success**: 100% hook compatibility
- **Error Clarity**: Actionable error messages
- **Documentation**: Comprehensive guides and examples

## Implementation Timeline

- **Week 1**: Foundation & Configuration (Phase 1)
- **Week 2**: Core Engine & Hook Implementation (Phases 2-3)
- **Week 3**: Git Integration & Migration (Phases 4-5)
- **Week 4**: Polish & Documentation (Phase 6)
- **Week 5**: Team rollout and training
- **Week 6**: Complete migration and cleanup

## Benefits Summary

### Immediate Benefits
1. **No Python Dependencies**: Eliminate Python, pip, and related security concerns
2. **Better Performance**: 3-5x faster execution through parallelization
3. **Single Binary**: Simple distribution and installation
4. **Native Go Integration**: Direct integration with Go toolchain

### Long-term Benefits
1. **Reduced Maintenance**: No more Python dependency updates
2. **Better Security**: Fewer supply chain attack vectors
3. **Improved Reliability**: Type-safe Go implementation
4. **Team Ownership**: Full control over hook behavior

### Developer Benefits
1. **Faster Feedback**: Sub-second hook execution
2. **Better Errors**: Clear, actionable error messages
3. **Flexible Configuration**: Fine-grained control
4. **Native Feel**: Designed for Go developers

## Risk Mitigation

### Technical Risks
- **Initial Development**: Mitigated by phased approach and comprehensive testing
- **Feature Completeness**: Ensure all necessary hooks are implemented
- **Performance**: Extensive benchmarking and profiling

### Organizational Risks
- **Team Adoption**: Clear documentation and training
- **Workflow Disruption**: Side-by-side operation during transition
- **Rollback Plan**: Keep pre-commit config for 30 days

## Conclusion

This Go-native hooks system represents a significant improvement over the current Python-based pre-commit setup. By leveraging Go's strengths - performance, simplicity, and single-binary distribution - we create a solution that is faster, more reliable, and easier to maintain. The bolt-on architecture ensures the system remains portable and non-invasive, while the phased implementation approach ensures high quality and reliability.

The investment in building our own hooks system pays dividends through reduced dependencies, improved performance, and complete control over our development workflow. This aligns perfectly with the GoFortress philosophy of minimal external dependencies and maximum developer productivity.
