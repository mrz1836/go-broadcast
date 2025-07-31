# GoFortress Pre-commit Hook Manager - Go-Native Implementation Plan

## Executive Summary

This document outlines a comprehensive plan to replace the Python-based pre-commit framework with a self-hosted, Go-native git hooks manager integrated directly into the GoFortress CI/CD pipeline. Built entirely in Go as a **bolt-on solution encapsulated within the `.github` folder**, this solution eliminates Python dependencies, provides better performance through parallel execution, and maintains simplicity while being fully portable between projects.

**Key Architecture Decision**: The pre-commit system follows the successful coverage system pattern:
- Configuration centralized in `.github/.env.shared` (single source of truth)
- Implementation self-contained in `.github/pre-commit/` (bolt-on architecture)
- MVP-first approach focusing on essential Go hooks
- Convention over configuration with sensible defaults

## Vision Statement

Create a best-in-class Go-native git hooks manager that embodies Go's philosophy of simplicity and performance:
- **Go-First Design**: Built by Go developers, for Go developers
- **Single Binary**: One compiled tool that does everything - no runtime dependencies
- **Lightning Fast**: Leverage Go's concurrency for parallel hook execution
- **Professional Quality**: Clean CLI interface with helpful error messages
- **Zero Dependencies**: Pure Go implementation with minimal external packages
- **Environment-Driven**: Configuration via `.github/.env.shared` (like coverage system)
- **MVP Approach**: Start with essential hooks, expand based on needs
- **Bolt-On Architecture**: Self-contained within `.github/pre-commit/` (except config)
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
├── .env.shared                   # Centralized configuration (all pre-commit settings)
└── pre-commit/                   # Self-contained GoFortress Pre-commit System
    ├── cmd/
    │   └── gofortress-hooks/     # Main CLI tool
    ├── internal/                 # Internal packages
    └── README.md                 # Documentation
```

## Implementation Roadmap

Each phase is designed to be completed in a single Claude Code session with clear deliverables and verification steps. The phases build upon each other, so they should be completed in order.

**IMPORTANT**: 
1. **All files must be created within the `.github` directory** - the entire GoFortress Pre-commit System is self-contained in `.github/pre-commit/`
2. After completing each phase, update the status tracking document at `plans/plan-10-status.md` with:
   - Mark the phase as completed (✓)
   - Note any deviations from the plan
   - Record actual metrics and timings
   - Document any issues encountered
   - Update the "Next Steps" section

### Phase 1: Foundation & Configuration (Session 1)
**Objective**: Establish infrastructure for the new GoFortress Pre-commit System

**Implementation Steps:**
1. Add pre-commit system environment variables to `.github/.env.shared`
2. Create directory structure for GoFortress Pre-commit System
3. Create initial configuration schema
4. Add labels for pre-commit-related PRs

**Files to Create/Modify:**
- `.github/.env.shared` - Add new variables (MVP configuration)
- `.github/pre-commit/` - Create directory structure
- `.github/pre-commit/.gitignore` - Ignore build artifacts
- `.github/pre-commit/README.md` - Initial documentation
- `.github/labels.yml` - Add hook-system label
- `.github/dependabot.yml` - Add hooks tool Go module monitoring
- `plans/plan-10-status.md` - Update status tracking document

**Verification Steps:**
```bash
# 1. Verify environment variables are added
grep "ENABLE_PRE_COMMIT" .github/.env.shared

# 2. Verify directory structure
ls -la .github/pre-commit/

# 3. Verify .env.shared configuration
grep "HOOKS_" .github/.env.shared | wc -l

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
# ENV: GoFortress Pre-commit System Configuration (MVP)
# ───────────────────────────────────────────────────────────────────────────────
ENABLE_PRE_COMMIT=false                  # Enable GoFortress Pre-commit System
HOOKS_PARALLEL_EXECUTION=true            # Run hooks in parallel
HOOKS_TIMEOUT_SECONDS=120                # Global timeout for hooks (2 min)
HOOKS_COLOR_OUTPUT=true                  # Use colored terminal output
HOOKS_LOG_LEVEL=info                     # Log level: debug, info, warn, error

# Hook Enable/Disable Flags (MVP - just essentials)
HOOKS_FUMPT_ENABLED=true                 # Enable gofumpt formatting (make fumpt)
HOOKS_LINT_ENABLED=true                  # Enable golangci-lint (make lint)
HOOKS_MOD_TIDY_ENABLED=true              # Enable go mod tidy (make mod-tidy)
HOOKS_TRAILING_WHITESPACE_ENABLED=true   # Fix trailing whitespace (built-in)
HOOKS_END_OF_FILE_FIXER_ENABLED=true     # Ensure files end with newline (built-in)

# Future enhancement placeholders (post-MVP)
# HOOKS_CACHE_ENABLED=false              # Cache hook results (future)
# HOOKS_PROGRESS_BAR=false               # Progress indicators (future)
# HOOKS_VET_ENABLED=false                # Go vet (future)
# HOOKS_GITLEAKS_ENABLED=false           # Secret scanning (future)
# HOOKS_GOVULNCHECK_ENABLED=false        # Vulnerability scanning (future)
```

#### 1.2 Directory Structure Creation - MVP Architecture
```bash
.github/
├── .env.shared                    # Centralized configuration (like coverage)
├── pre-commit/                    # Self-contained GoFortress Pre-commit System (bolt-on)
│   ├── go.mod                     # Go module at root (like coverage)
│   ├── go.sum
│   ├── cmd/
│   │   └── gofortress-hooks/      # Main CLI tool
│   │       ├── main.go            # Entry point
│   │       └── cmd/               # Command implementations (MVP: 3 commands)
│   │           ├── install.go     # Install git hooks
│   │           ├── run.go         # Run hooks
│   │           └── uninstall.go   # Uninstall hooks
│   ├── internal/                  # Internal packages
│   │   ├── config/                # Configuration from .env.shared
│   │   │   ├── env.go             # Read env configuration
│   │   │   └── config.go          # Config structure
│   │   ├── runner/                # Hook execution engine
│   │   │   ├── runner.go          # Simple parallel runner
│   │   │   └── runner_test.go
│   │   ├── hooks/                 # Built-in hook implementations (MVP: 5 hooks)
│   │   │   ├── hook.go            # Hook interface
│   │   │   ├── fumpt.go           # Make fumpt wrapper
│   │   │   ├── lint.go            # Make lint wrapper
│   │   │   ├── mod_tidy.go        # Make mod-tidy wrapper
│   │   │   ├── whitespace.go      # Trailing whitespace fixer
│   │   │   └── eof_fixer.go       # End-of-file fixer
│   │   └── git/                   # Git integration
│   │       ├── installer.go       # Hook installation
│   │       └── files.go           # Get changed files
│   ├── .gitignore                # Ignore builds and artifacts
│   └── README.md                 # Hooks system documentation
└── workflows/
    └── fortress-test-suite.yml   # Updated with hooks integration
```

**Key Benefits of This Structure:**
- ✅ **Complete Encapsulation**: Everything pre-commit-related lives in `.github/pre-commit/`
- ✅ **Portable**: The entire `pre-commit/` folder can be copied to any repository
- ✅ **Separate Module**: Pre-commit tool has its own `go.mod` for isolated dependencies
- ✅ **Clean Separation**: Main repository remains uncluttered
- ✅ **Self-Documenting**: GoFortress Pre-commit System includes its own README.md

#### 1.2.1 Create .gitignore for GoFortress Pre-commit System
Create `.github/pre-commit/.gitignore`:
```gitignore
# Build artifacts
gofortress-hooks
cmd/gofortress-hooks/gofortress-hooks
*.exe
*.dll
*.so
*.dylib

# Test and coverage artifacts
*.test
*.out
coverage.html
coverage.xml
*.prof

# Cache directories
.cache/
*.tmp

# IDE and editor files
.idea/
.vscode/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Debug files
__debug_bin
*.log
```

#### 1.3 Configuration Reader Implementation
Create configuration reader that loads from `.env.shared` (following coverage pattern):

```go
// internal/config/env.go
package config

import (
    "os"
    "path/filepath"
    "strconv"
    "github.com/joho/godotenv"
)

// LoadFromEnv loads configuration from .github/.env.shared
func LoadFromEnv() (*Config, error) {
    // Load .env.shared file (like coverage system)
    envPath := filepath.Join(".github", ".env.shared")
    if err := godotenv.Load(envPath); err != nil {
        // Not an error if file doesn't exist (use defaults)
        if !os.IsNotExist(err) {
            return nil, fmt.Errorf("failed to load .env.shared: %w", err)
        }
    }
    
    // Parse configuration from environment
    cfg := &Config{
        Enabled:     getEnvBool("ENABLE_PRE_COMMIT", false),
        Parallel:    getEnvBool("HOOKS_PARALLEL_EXECUTION", true),
        Timeout:     getEnvInt("HOOKS_TIMEOUT_SECONDS", 120),
        ColorOutput: getEnvBool("HOOKS_COLOR_OUTPUT", true),
        LogLevel:    getEnvString("HOOKS_LOG_LEVEL", "info"),
    }
    
    // Load hook-specific settings (MVP hooks only)
    cfg.Hooks = HooksConfig{
        Fumpt:              getEnvBool("HOOKS_FUMPT_ENABLED", true),
        Lint:               getEnvBool("HOOKS_LINT_ENABLED", true),
        ModTidy:            getEnvBool("HOOKS_MOD_TIDY_ENABLED", true),
        TrailingWhitespace: getEnvBool("HOOKS_TRAILING_WHITESPACE_ENABLED", true),
        EndOfFile:          getEnvBool("HOOKS_END_OF_FILE_FIXER_ENABLED", true),
    }
    
    return cfg, nil
}
```

**Note**: No YAML configuration files in MVP - all configuration via environment variables in `.env.shared`

#### 1.4 Label Configuration
Add to `.github/labels.yml`:
```yaml
- name: "hook-system"
  description: "Git hooks system related"
  color: 4a5aba
```

#### 1.5 Dependabot Configuration for GoFortress Pre-commit Tool
Since the pre-commit tool will be a separate Go module in `.github/pre-commit/`, we need to add it to dependabot monitoring.

Add to `.github/dependabot.yml` after the main gomod entry:
```yaml
  # ──────────────────────────────────────────────────────────────
  # 1c. GoFortress Pre-commit Tool Go Module (.github/pre-commit)
  # ──────────────────────────────────────────────────────────────
  - package-ecosystem: "gomod"
    directory: "/.github/pre-commit"
    target-branch: "master"
    schedule:
      interval: "weekly"
      day: "monday"
      time: "09:00"
      timezone: "America/New_York"
    allow:
      - dependency-type: "direct"
    groups:
      pre-commit-deps:
        patterns:
          - "*"
        update-types: ["minor", "patch"]
    open-pull-requests-limit: 5
    assignees: ["mrz1836"]
    labels: ["chore", "dependencies", "gomod", "pre-commit-system"]
    commit-message:
      prefix: "chore"
      include: "scope"
```

This ensures the GoFortress Pre-commit tool's dependencies (cobra, godotenv, etc.) stay up-to-date and secure.

### Phase 2: Core Pre-commit Engine (Session 2)
**Objective**: Build the MVP Go-native pre-commit processing tool

**Implementation Steps:**
1. Create simple CLI with just 3 commands: install, run, uninstall
2. Implement env-based configuration parser (`internal/config`)
3. Implement basic parallel runner (`internal/runner`)
4. Implement 5 MVP hooks (3 make wrappers + 2 built-in)
5. Create git integration (`internal/git`)
6. Write focused tests for core functionality
7. Ensure compatibility with CI environment

**Files to Create (MVP Structure):**
```
.github/pre-commit/
├── go.mod                       # Go module at root (like coverage)
├── go.sum                       # Module dependencies
├── cmd/
│   └── gofortress-hooks/
│       ├── main.go              # CLI entry point
│       └── cmd/                 # Command implementations (MVP: 3 only)
│           ├── install.go       # Install hooks command
│           ├── run.go           # Run hooks command
│           └── uninstall.go     # Uninstall hooks command
├── internal/
│   ├── config/
│   │   ├── env.go               # Load from .env.shared
│   │   ├── config.go            # Configuration structure
│   │   └── config_test.go       # Unit tests
│   ├── runner/
│   │   ├── runner.go            # Simple parallel execution
│   │   └── runner_test.go       # Unit tests
│   ├── hooks/
│   │   ├── hook.go              # Hook interface
│   │   ├── fumpt.go             # Make fumpt wrapper
│   │   ├── lint.go              # Make lint wrapper
│   │   ├── mod_tidy.go          # Make mod-tidy wrapper
│   │   ├── whitespace.go        # Trailing whitespace fixer
│   │   ├── eof_fixer.go         # End-of-file fixer
│   │   └── hooks_test.go        # Tests for all hooks
│   └── git/
│       ├── installer.go         # Install/uninstall logic
│       └── files.go             # Get changed files
└── Makefile                     # Build commands
```

**Go Module Dependencies (.github/pre-commit/go.mod):**
```go
module github.com/mrz1836/go-broadcast/pre-commit

go 1.24

require (
    github.com/spf13/cobra v1.8.0      // CLI framework
    github.com/joho/godotenv v1.5.1    // Read .env.shared
    github.com/stretchr/testify v1.8.4 // Testing
    github.com/fatih/color v1.16.0     // Colored output
)
```

**Key Benefits of Separate Module:**
- ✅ **Isolated Dependencies**: Pre-commit tool dependencies don't affect main project
- ✅ **Version Independence**: Can update pre-commit tool dependencies separately
- ✅ **Reduced Complexity**: Main go.mod stays clean
- ✅ **Security**: Easier to audit pre-commit tool dependencies separately

**Verification Steps:**
```bash
# 1. Build the tool
cd .github/pre-commit
go build -o cmd/gofortress-hooks/gofortress-hooks ./cmd/gofortress-hooks

# 2. Run all tests with coverage
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out -o coverage.html

# 3. Verify test coverage
go tool cover -func=coverage.out | grep total | awk '{print $3}'

# 4. Run benchmarks
go test -bench=. -benchmem ./internal/...

# 5. Run linting
golangci-lint run ./...

# 6. Test CLI commands
./cmd/gofortress-hooks/gofortress-hooks --help
./cmd/gofortress-hooks/gofortress-hooks install
./cmd/gofortress-hooks/gofortress-hooks run pre-commit

# 7. Run race detector tests
go test -race ./...

# 8. Check for vulnerabilities
govulncheck ./...
```

**Success Criteria:**
- ✅ Single Go binary compiles without errors
- ✅ Configuration loads from .env.shared (like coverage)
- ✅ 3 CLI commands work: install, run, uninstall
- ✅ 5 MVP hooks execute correctly
- ✅ Hooks run in parallel
- ✅ Works identically in local and CI environments
- ✅ Clean error messages
- ✅ < 2 second execution for typical commit

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 2 as completed (✓)
- Document performance metrics
- Note any design decisions or changes

### Phase 3: Pre-commit Hook Implementations (Session 3)
**Objective**: Implement MVP pre-commit hooks only

**Implementation Steps:**
1. Implement 3 make command wrappers (fumpt, lint, mod-tidy)
2. Implement 2 built-in hooks (whitespace, EOF)
3. Create simple hook registry
4. Add basic tests for each hook
5. Ensure make command output is properly captured

**Files to Create/Modify:**
- Implement 5 MVP pre-commit hooks in `internal/hooks/`
- Create shared make command executor
- Create tests for each hook
- Focus on correctness over performance

**Testing Each Hook:**
```bash
# Test MVP hooks
./gofortress-hooks run fumpt
./gofortress-hooks run lint
./gofortress-hooks run mod-tidy

# Test all hooks together
./gofortress-hooks run pre-commit

# Verify make command execution
make fumpt  # Direct make command
./gofortress-hooks run fumpt  # Should produce identical output
```

**Verification Steps:**
```bash
# 1. Test each MVP hook
for hook in fumpt lint mod-tidy; do
    ./gofortress-hooks run $hook
done

# 2. Verify make command execution
./gofortress-hooks run lint 2>&1 | grep "make lint"

# 3. Test that hooks match CI behavior
make lint  # Direct make command
./gofortress-hooks run lint  # Should produce identical output

# 4. Test built-in hooks
echo "test  " > test.txt
./gofortress-hooks run trailing-whitespace
cat test.txt  # Should have no trailing spaces
```

**Success Criteria:**
- ✅ MVP pre-commit hooks produce same results as make commands
- ✅ Make command output properly captured
- ✅ Built-in hooks work correctly
- ✅ Pre-commit hooks match CI behavior
- ✅ Performance: <2s for typical commit
- ✅ Clear output without verbose flags

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 3 as completed (✓)
- Record pre-commit hook performance benchmarks
- Document any compatibility issues

### Phase 4: Git Integration & Installation (Session 4)
**Objective**: Simple git integration for MVP

**Implementation Steps:**
1. Create basic installer that sets up pre-commit hook
2. Create simple uninstaller
3. Support SKIP environment variable
4. Detect CI environment (skip if ENABLE_PRE_COMMIT=false)

**Files to Create/Modify:**
- `internal/git/installer.go` - Install/uninstall logic
- Generate simple hook script on install

**Installation Process:**
```bash
# Install hooks (creates .git/hooks/pre-commit)
./gofortress-hooks install

# Uninstall hooks
./gofortress-hooks uninstall
```

**Verification Steps:**
```bash
# 1. Test installation
./gofortress-hooks install
ls -la .git/hooks/pre-commit

# 2. Test actual git commit
echo "test" > test.txt
git add test.txt
git commit -m "test: hook execution"

# 3. Test skip functionality
SKIP=lint git commit -m "test: skip linter"

# 4. Test uninstall
./gofortress-hooks uninstall
ls .git/hooks/pre-commit  # Should not exist
```

**Success Criteria:**
- ✅ Hooks install with single command
- ✅ Git commits trigger hooks
- ✅ SKIP environment variable works
- ✅ Respects ENABLE_PRE_COMMIT from .env.shared
- ✅ Clean uninstall

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 4 as completed (✓)
- Document installation process
- Note any git version compatibility issues

### Phase 5: CI/CD Integration (Session 5)
**Objective**: Create dedicated fortress-pre-commit.yml workflow

**Implementation Steps:**
1. Create new fortress-pre-commit.yml reusable workflow
2. Follow GoFortress patterns (verbose logging, status checks, summaries)
3. Include fallback to make commands when hooks system not available
4. Integrate with existing GoFortress workflow orchestration

**Files to Create/Modify:**
- Create `.github/workflows/fortress-pre-commit.yml`
- Update calling workflows to include pre-commit step
- Ensure conditional execution based on ENABLE_PRE_COMMIT

**Fortress Pre-commit Workflow Features:**
- **Status Checks**: Verify hooks system exists before attempting to build
- **Verbose Logging**: Display configuration from .env.shared
- **Build & Execute**: Build gofortress-hooks and run on all files
- **Fallback Mode**: Use make commands if hooks system not available
- **Job Summary**: Detailed summary with execution results

**Workflow Structure:**
```yaml
name: GoFortress (Pre-commit Hooks)

on:
  workflow_call:
    inputs:
      env-json:
        description: "JSON string of environment variables"
        required: true
        type: string
      pre-commit-enabled:
        description: "Whether GoFortress Pre-commit is enabled"
        required: true
        type: string
    outputs:
      hooks-version:
        description: "Version of gofortress-hooks used"
      hooks-executed:
        description: "List of hooks that were executed"
```

**Key Workflow Steps:**
1. Parse environment variables from env-json
2. Check if .github/hooks/ exists (status check)
3. Display hooks configuration from .env.shared
4. Build gofortress-hooks if system exists
5. Run hooks with verbose output
6. Fallback to make commands if needed
7. Generate detailed job summary

**Integration Pattern:**
```yaml
# In calling workflows (e.g., fortress-orchestrator.yml)
pre-commit:
  name: 🪝 Pre-commit Hooks
  needs: [setup]
  if: needs.setup.outputs.pre-commit-enabled == 'true'
  uses: ./.github/workflows/fortress-pre-commit.yml
  with:
    env-json: ${{ needs.setup.outputs.env-json }}
    primary-runner: ${{ needs.setup.outputs.primary-runner }}
    go-primary-version: ${{ needs.setup.outputs.go-primary-version }}
    pre-commit-enabled: ${{ needs.setup.outputs.pre-commit-enabled }}
  secrets:
    github-token: ${{ secrets.GITHUB_TOKEN }}
```

**Verification Steps:**
```bash
# 1. Test workflow locally with act
act -W .github/workflows/fortress-pre-commit.yml

# 2. Verify with ENABLE_PRE_COMMIT=true
ENABLE_PRE_COMMIT=true gh workflow run

# 3. Verify with ENABLE_PRE_COMMIT=false (should skip)
ENABLE_PRE_COMMIT=false gh workflow run

# 4. Check job summaries in GitHub UI
```

**Success Criteria:**
- ✅ Workflow follows GoFortress patterns
- ✅ Status checks detect hooks system presence
- ✅ Configuration displayed clearly
- ✅ Graceful fallback to make commands
- ✅ Detailed job summaries
- ✅ Respects ENABLE_PRE_COMMIT setting

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 5 as completed (✓)
- Document fortress-pre-commit.yml creation
- Note integration approach

### Phase 6: Documentation & Release (Session 6)
**Objective**: Document MVP and prepare for release

**Implementation Steps:**
1. Write clear README.md in .github/pre-commit/
2. Add basic usage examples
3. Document configuration via .env.shared
4. Create simple troubleshooting guide

**Files to Create/Modify:**
- `.github/pre-commit/README.md` - Complete documentation
- Update main `README.md` with brief mention, similiar to coverage system
  - make a section, add to feature list, but anything verbose make it in <details> tags
- Update `CLAUDE.md` with pre-commit information

**Documentation Requirements:**
1. **`.github/pre-commit/README.md`**:
   - Installation instructions
   - Configuration via .env.shared
   - Available pre-commit hooks (MVP: 5 hooks)
   - Usage examples
   - Troubleshooting

2. **Main README.md Update**:
   - Brief mention in "Features" section
   - Link to .github/pre-commit/README.md

**Basic Usage:**
```bash
# Install pre-commit hooks
cd .github/pre-commit
go build -o gofortress-hooks ./cmd/gofortress-hooks
./gofortress-hooks install

# Run pre-commit hooks manually
./gofortress-hooks run pre-commit

# Uninstall
./gofortress-hooks uninstall
```

**Verification Steps:**
```bash
# 1. Test installation and usage
./gofortress-hooks install
git add -A
git commit -m "test: hooks working"

# 2. Test configuration
HOOKS_LINT_ENABLED=false ./gofortress-hooks run pre-commit

# 3. Verify documentation
cat .github/hooks/README.md
```

**Success Criteria:**
- ✅ Clear documentation in .github/pre-commit/README.md
- ✅ Installation process documented
- ✅ Configuration via .env.shared explained
- ✅ Basic troubleshooting included

**Status Update Required:**
After completing this phase, update `plans/plan-10-status.md`:
- Mark Phase 6 as completed (✓)
- Confirm documentation complete
- Ready for team testing

## Configuration Examples

### MVP Configuration (in `.github/.env.shared`)
```bash
# Enable the GoFortress Pre-commit System
ENABLE_PRE_COMMIT=true

# Configure individual pre-commit hooks
HOOKS_FUMPT_ENABLED=true      # Format code with gofumpt
HOOKS_LINT_ENABLED=true       # Run golangci-lint
HOOKS_MOD_TIDY_ENABLED=true   # Keep go.mod tidy

# Disable specific pre-commit hooks if needed
# HOOKS_LINT_ENABLED=false    # Temporarily disable linting
```

### Hook Execution
```bash
# All pre-commit hooks run by default
./gofortress-hooks run pre-commit

# Skip specific pre-commit hooks
SKIP=lint ./gofortress-hooks run pre-commit

# Run individual pre-commit hooks
./gofortress-hooks run fumpt
./gofortress-hooks run lint
```

## MVP Implementation Summary

### Key Decisions
1. **Configuration via .env.shared** - Single source of truth (like coverage)
2. **Minimal commands** - Just install, run, uninstall
3. **5 essential hooks** - fumpt, lint, mod-tidy, whitespace, EOF
4. **Make command integration** - Consistency with CI
5. **No YAML files** - All config from environment

### Quick Start
```bash
# 1. Enable in .env.shared
ENABLE_PRE_COMMIT=true

# 2. Build and install
cd .github/pre-commit
go build -o gofortress-hooks ./cmd/gofortress-hooks
./gofortress-hooks install

# 3. Use git normally
git add .
git commit -m "feat: new feature"
```

## Phase 7: Python/Pre-commit Removal (Post-MVP)
**Objective**: Remove all Python and pre-commit dependencies

**Implementation Steps:**
1. Ensure GoFortress Pre-commit System is working reliably
2. Remove Python/pip dependencies
3. Remove old pre-commit configuration
4. Update workflows to remove old pre-commit references
5. Clean up any remaining Python scripts

**Files to Remove/Modify:**
- `.pre-commit-config.yaml` - Remove entirely
- `.github/pip/` - Remove entire directory
- `.github/workflows/update-pre-commit-hooks.yml` - Remove workflow
- `.github/workflows/update-python-dependencies.yml` - Remove workflow
- Any Python scripts (e.g., `scripts/comment_lint.py` if exists)
- Update `.github/.env.shared` to remove Python-related variables

**Verification Steps:**
```bash
# 1. Ensure GoFortress Pre-commit is working
./gofortress-hooks run pre-commit

# 2. Check for Python dependencies
find . -name "*.py" -type f
find . -name "requirements*.txt" -type f

# 3. Check for pre-commit references
grep -r "pre-commit" .github/workflows/

# 4. Verify no pip dependencies
ls -la .github/pip/  # Should not exist
```

**Success Criteria:**
- ✅ All Python/pip dependencies removed
- ✅ Old pre-commit configuration removed
- ✅ Workflows updated to use GoFortress Pre-commit System
- ✅ No Python runtime required
- ✅ CI/CD continues to work correctly

## Future Enhancements (Post-MVP)

After the MVP proves successful and Python is removed, consider adding:

### Phase 1: Performance
- Smart caching for expensive operations (lint)
- Progress indicators for long operations
- Parallel execution optimization

### Phase 2: Additional Hooks
- Security scanning (gitleaks, govulncheck)
- Go vet integration
- Commit message validation
- File size limits

### Phase 3: Developer Experience
- Colored output with detailed errors
- Interactive mode for hook selection
- Performance profiling
- Shell completions

### Phase 4: Advanced Features
- Custom hook support
- Hook-specific configuration
- Metrics and reporting
- Pre-push and commit-msg hooks

## MVP Testing Strategy

### Essential Tests
1. **Configuration Loading**
   - Reads from .env.shared correctly
   - Respects ENABLE_PRE_COMMIT flag
   - Individual hook enable/disable works

2. **Hook Execution**
   - Make commands execute correctly
   - Output captured properly
   - Exit codes handled
   - Built-in hooks work

3. **Git Integration**
   - Install creates pre-commit hook
   - Hook runs on git commit
   - SKIP variable works
   - Uninstall removes hook

4. **CI Compatibility**
   - Works with CI=true
   - Respects environment configuration
   - Same behavior as local

## MVP Success Metrics

### Performance
- **Pre-commit Hook Execution**: <2s for typical commit
- **Installation**: <10 seconds
- **Binary Size**: <10MB
- **Memory Usage**: Minimal

### Reliability
- **Core Functionality**: 100% working
- **CI Integration**: Seamless
- **Error Handling**: Clear messages
- **Environment Config**: Reliable

### Developer Experience
- **Zero Configuration**: Works with defaults
- **Simple Commands**: Just 3 (install, run, uninstall)
- **Clear Documentation**: In .github/pre-commit/README.md
- **Consistent Behavior**: Same locally and in CI

## Implementation Timeline

- **Session 1**: Foundation & Configuration (Phase 1)
- **Session 2**: Core Engine Implementation (Phase 2)
- **Session 3**: MVP Hook Implementation (Phase 3)
- **Session 4**: Git Integration (Phase 4)
- **Session 5**: CI Integration (Phase 5)
- **Session 6**: Documentation & Release (Phase 6)
- **Post-MVP**: Python/Pre-commit Removal (Phase 7)

Total estimated time: 6 focused sessions for MVP, plus cleanup phase

## MVP Benefits

### Immediate Benefits
1. **No Python Dependencies**: Pure Go implementation
2. **Fast Execution**: Direct make command integration
3. **Single Binary**: No runtime dependencies
4. **Environment-Driven**: Configuration via .env.shared

### Consistency Benefits
1. **CI/CD Alignment**: Same make commands everywhere
2. **Centralized Config**: Single source of truth
3. **Portable**: Entire system in .github/
4. **Simple**: Just 5 essential hooks

### Developer Benefits
1. **Quick Setup**: < 1 minute to install
2. **Familiar**: Uses existing make commands
3. **Flexible**: Easy enable/disable via env
4. **Reliable**: No complex dependencies

## Risk Mitigation

### MVP Approach Reduces Risk
- **Minimal Scope**: Only 5 essential hooks
- **Proven Pattern**: Follows coverage system architecture
- **Simple Config**: Environment variables only
- **Easy Rollback**: Just uninstall and disable

### Technical Risks
- **Make Command Changes**: Hooks use stable make targets
- **CI Compatibility**: Tested in both environments
- **Performance**: Direct command execution

### Adoption Strategy
- **Opt-in**: ENABLE_PRE_COMMIT=false by default
- **Side-by-side**: Run both systems initially
- **Gradual Migration**: Team by team validation
- **Clean Removal**: Remove Python only after proven stable
- **Clear Benefits**: Faster, simpler, no Python runtime

## Conclusion

This MVP GoFortress Pre-commit System follows the successful pattern established by the coverage system:
- **Centralized configuration** in `.github/.env.shared`
- **Bolt-on architecture** in `.github/pre-commit/`
- **MVP-first approach** with just essential features
- **Convention over configuration** for simplicity

By starting with just 5 essential pre-commit hooks and 3 commands, we can deliver a working system quickly and enhance based on real usage. The use of environment variables for all configuration keeps the system simple and consistent with other GoFortress tools.

This approach eliminates Python dependencies while maintaining the quality checks developers expect, all in a package that installs in seconds and runs in milliseconds.
