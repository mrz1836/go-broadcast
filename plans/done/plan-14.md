# Go Coverage System Migration Plan

## Executive Summary

This document outlines a streamlined plan to migrate the Go Coverage System from an embedded module within go-broadcast to a standalone, reusable Go ecosystem tool. Following the successful pattern of the go-pre-commit migration, this plan emphasizes simplicity, focusing on `go install` as the primary distribution method and maintaining `.env.shared` configuration without introducing new formats initially.

**Key Migration Features:**
- **Standalone Repository**: Independent `mrz1836/go-coverage` with zero go-broadcast dependencies
- **Simple Configuration**: `.env.shared` compatibility maintained, no new config formats initially
- **Primary Installation**: Focus on `go install` first, other methods added incrementally
- **Zero Breaking Changes**: Exact same environment variables and behavior for existing users
- **Enhanced Portability**: Works independently or as part of fortress ecosystem
- **Asset Embedding**: Static assets (CSS, JS, images) embedded using Go 1.16+ embed package
- **GitHub Integration**: Maintained support for PR comments, status checks, and Pages deployment
- **Performance Preservation**: Maintains <5s badge generation and efficient report creation

## Vision Statement

The Go Coverage System will evolve from an embedded tool to become the premier Go ecosystem coverage solution, providing:
- **Universal Adoption**: Any Go project can use it via simple `go install`
- **Configuration Simplicity**: Environment variables only initially, following go-pre-commit's success
- **Zero External Dependencies**: Single binary with embedded assets
- **Professional Quality**: GitHub-style badges, beautiful reports, and actionable insights
- **Production Ready**: Ship v1.0.0 when core functionality is solid
- **CI/CD First**: Direct integration in any CI/CD pipeline via go install
- **Incremental Features**: Start with core, add advanced features based on user feedback
- **Independence**: Works perfectly without fortress, while maintaining seamless fortress integration

This migration establishes go-coverage as the Go community's premier coverage visualization and tracking tool, following the proven go-pre-commit pattern of simplicity first.

## Implementation Strategy

This plan uses a phased migration approach to ensure:
- Continuous functionality throughout migration
- Zero downtime for existing users
- Backward compatibility at every phase
- Clear rollback points if issues arise
- Incremental testing and validation
- Smooth transition for all workflows

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Go Coverage Architecture                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────────────────┐               │
│  │ Config       │───▶│ Coverage Engine          │               │
│  │ (.env.shared)│    │                          │               │
│  │ (env vars)   │    │ ├─ Parser                │               │
│  └──────────────┘    │ ├─ Analyzer              │               │
│                      │ ├─ History Tracker       │               │
│                      │ └─ Metrics Calculator    │               │
│                      └──────────┬───────────────┘               │
│                                 │                               │
│                                 ▼                               │
│  ┌─────────────────────────────────────────────┐                │
│  │          Output Generators                  │                │
│  │                                             │                │
│  │  ├─ Badge Generator (SVG)                   │                │
│  │  ├─ HTML Report Builder                     │                │
│  │  ├─ Dashboard Creator                       │                │
│  │  ├─ PR Comment Formatter                    │                │
│  │  └─ Analytics Exporter                      │                │
│  └─────────────────────────────────────────────┘                │
│                                                                 │
│  ┌─────────────────────────────────────────────┐                │
│  │          Embedded Assets (go:embed)         │                │
│  │                                             │                │
│  │  ├─ CSS Styles                              │                │
│  │  ├─ JavaScript                              │                │
│  │  ├─ Images & Icons                          │                │
│  │  └─ HTML Templates                          │                │
│  └─────────────────────────────────────────────┘                │
│                                                                 │
│  Primary Installation Method:                                   │
│  - go install github.com/mrz1836/go-coverage@latest             │
│                                                                 │
│  Future Installation Methods (post-v1.0.0):                     │
│  - Homebrew, Docker, Binary releases                            │
└─────────────────────────────────────────────────────────────────┘

Configuration Loading (Simplified):
1. Command-line flags (highest priority)
2. Runtime environment variables
3. Project .github/.env.shared file (fortress compatibility)
4. Built-in defaults (lowest priority)
Note: YAML config support to be added based on user demand
```

## Implementation Roadmap

### Phase 0: Pre-Migration Analysis
**Objective**: Quick analysis of current implementation
**Duration**: 2-3 hours

**Implementation Steps:**
1. Verify zero go-broadcast imports in `.github/coverage/`
2. List all static assets for embedding (CSS, JS, images)
3. Document all `COVERAGE_*` environment variables
4. Note GitHub API integration points

**Success Criteria:**
- ✅ Zero go-broadcast imports confirmed
- ✅ All assets cataloged for embedding
- ✅ Environment variables documented

### Phase 1: Repository Creation and Initial Setup
**Objective**: Create new repository with clean structure
**Duration**: 2-3 hours

**Implementation Steps:**
1. Create `mrz1836/go-coverage` repository on GitHub
2. Initialize with MIT license and README (follow go-pre-commit pattern)
3. Set up basic GitHub Actions
4. Create simple project structure

**Repository Structure (Simplified):**
```
mrz1836/go-coverage/
├── .github/
│   ├── workflows/
│   │   └── ci.yml              # Basic test and build
│   └── dependabot.yml
├── cmd/
│   └── go-coverage/            # Main binary
│       └── main.go
├── internal/
│   ├── analysis/               # Coverage analysis
│   ├── badge/                  # SVG badge generation
│   ├── config/                 # Configuration (env vars only)
│   ├── github/                 # GitHub API client
│   ├── parser/                 # Coverage file parsing
│   └── report/                 # HTML report generation
├── assets/                     # Embedded static assets
│   ├── css/
│   ├── js/
│   └── images/
├── examples/                   # Usage examples
├── .goreleaser.yml             # Release configuration
├── go.mod                      # Module definition
├── Makefile                    # Development tasks
└── README.md                   # Comprehensive docs
```

**Success Criteria:**
- ✅ Repository created
- ✅ Basic CI/CD working
- ✅ Clean structure in place

### Phase 2: Code Migration with Asset Embedding
**Objective**: Migrate code and embed all static assets
**Duration**: 4-5 hours

**Implementation Steps:**
1. Copy all code from `.github/coverage/` to new repository
2. Update module path to `github.com/mrz1836/go-coverage`
3. Rename binary: `gofortress-coverage` → `go-coverage`
4. Embed static assets using `embed` package
5. Update asset references to use embedded content
6. Verify tests pass

**Asset Embedding Strategy:**
```go
package assets

import "embed"

//go:embed css/*.css js/*.js images/*
var EmbeddedAssets embed.FS

// Simple getters for assets
func GetCSS(filename string) ([]byte, error) {
    return EmbeddedAssets.ReadFile("css/" + filename)
}
```

**Success Criteria:**
- ✅ Code migrated
- ✅ Module paths updated
- ✅ Assets embedded
- ✅ Tests passing

### Phase 3: Configuration System (Simplified)
**Objective**: Keep configuration simple - env vars only initially
**Duration**: 2-3 hours

**Implementation Steps:**
1. Maintain `.env.shared` compatibility for fortress users
2. Support pure environment variables for standalone use
3. Add basic command-line flags
4. Skip YAML config for now (add later if users request)

**Configuration Loading Logic (Simple):**
```go
// LoadConfig loads configuration from environment
func LoadConfig() (*Config, error) {
    cfg := NewDefaultConfig()

    // Try .github/.env.shared for fortress compatibility
    if envPath := findEnvShared(); envPath != "" {
        godotenv.Load(envPath) // Optional, ignore errors
    }

    // Load from environment variables
    cfg.ThresholdExcellent = getEnvInt("COVERAGE_THRESHOLD_EXCELLENT", 90)
    cfg.ThresholdGood = getEnvInt("COVERAGE_THRESHOLD_GOOD", 80)
    // ... etc for all COVERAGE_* vars

    return cfg, nil
}
```

**Environment Variables (Unchanged):**
```bash
# All existing COVERAGE_* variables continue to work
COVERAGE_THRESHOLD_EXCELLENT=90
COVERAGE_THRESHOLD_GOOD=80
COVERAGE_BADGE_STYLE=flat
COVERAGE_REPORT_THEME=github-dark
# ... all 40+ existing variables maintained
```

**Success Criteria:**
- ✅ Fortress compatibility maintained
- ✅ Environment variables working
- ✅ Command-line flags added

### Phase 4: Installation (Focus on go install)
**Objective**: Set up primary installation method
**Duration**: 3-4 hours

**Implementation Steps:**
1. Ensure `go install` works perfectly
2. Set up GoReleaser for binary releases
3. Create simple install script
4. Document installation clearly

**Primary Installation Method:**
```bash
# This is the focus - make it work perfectly
go install github.com/mrz1836/go-coverage@latest
```

**Future Installation Methods (post-v1.0.0):**
- Homebrew formula
- Docker images
- Direct binary downloads

**Simple GoReleaser Config:**
```yaml
builds:
  - id: go-coverage
    main: ./cmd/go-coverage
    binary: go-coverage
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
```

**Success Criteria:**
- ✅ `go install` works perfectly
- ✅ GoReleaser configured
- ✅ Basic install script created

### Phase 5: CI/CD Integration Updates
**Objective**: Update workflows for external tool
**Duration**: 3-4 hours

**Implementation Steps:**
1. Update fortress-coverage.yml to use `go install`
2. Ensure GitHub Pages deployment continues
3. Test PR comment functionality
4. Add basic CI/CD examples

**Updated fortress-coverage.yml (Simple):**
```yaml
- name: Install go-coverage
  run: go install github.com/mrz1836/go-coverage@latest

- name: Load fortress config
  run: |
    if [ -f .github/.env.shared ]; then
      set -a
      source .github/.env.shared
      set +a
      env | grep ^COVERAGE_ >> $GITHUB_ENV
    fi

- name: Generate coverage reports
  run: go-coverage complete --input coverage.txt --output coverage-output
```

**GitHub Actions Example:**
```yaml
- name: Coverage
  run: |
    go test -coverprofile=coverage.txt ./...
    go install github.com/mrz1836/go-coverage@latest
    go-coverage complete --input coverage.txt
```

**Success Criteria:**
- ✅ Fortress workflows updated
- ✅ GitHub Pages working
- ✅ PR comments functional

### Phase 6: go-broadcast Integration Update
**Objective**: Update go-broadcast to use external tool
**Duration**: 3-4 hours

**Implementation Steps:**
1. Remove `.github/coverage/` directory
2. Update documentation
3. Keep `.github/.env.shared` unchanged
4. Update CI/CD workflows
5. Create simple migration script

**Migration Script (migrate-coverage.sh):**
```bash
#!/bin/bash
echo "🔄 Migrating to external go-coverage..."

# Backup existing setup
if [ -d ".github/coverage" ]; then
    echo "📦 Backing up coverage directory..."
    tar -czf coverage-backup-$(date +%Y%m%d).tar.gz .github/coverage
fi

# Install new tool
echo "📥 Installing go-coverage..."
go install github.com/mrz1836/go-coverage@latest

# Migrate history if present
if [ -d ".github/coverage/history" ]; then
    mkdir -p .coverage-history
    cp -r .github/coverage/history/* .coverage-history/
fi

echo "✅ Migration complete!"
echo "Test with: go-coverage complete --input coverage.txt"
```

**Success Criteria:**
- ✅ go-broadcast updated
- ✅ Migration script working
- ✅ CI/CD functioning

### Phase 7: Testing and Documentation
**Objective**: Test everything and create great docs
**Duration**: 4-5 hours

**Implementation Steps:**
1. Test all core functionality
2. Write comprehensive README (follow go-pre-commit pattern)
3. Add quickstart guide
4. Create migration guide
5. Add CI/CD examples

**README Structure (like go-pre-commit):**
- 🚀 Quickstart (30 seconds to working)
- ⚙️ Configuration (env vars only initially)
- 📊 Features (badges, reports, PR comments)
- 📚 Documentation links
- Badges for quality metrics

**Test Coverage:**
- Badge generation
- Report creation
- PR comments
- GitHub Pages deployment
- History tracking

**Success Criteria:**
- ✅ Core features tested
- ✅ README comprehensive
- ✅ Migration guide clear
- ✅ Examples provided

## Future Enhancements (Post-v1.0.0)

After the initial release proves stable, consider adding:

### Advanced Features
- **Multiple coverage formats**: Cobertura, LCOV, JaCoCo support
- **YAML configuration**: For users who prefer config files over env vars
- **Plugin system**: For custom analyzers and extensions
- **Service integrations**: Slack, Discord, webhook notifications
- **Coverage goals**: Per-package thresholds and trend enforcement

### Additional Installation Methods
- Homebrew formula
- Docker images
- GitHub Action (composite action)
- Package managers (apt, yum, etc.)

### Enhanced Documentation
- Video tutorials
- Advanced usage guides
- Plugin development docs
- Community examples


## Configuration Examples

### Configuration (Environment Variables Only Initially)

**Fortress Users (.github/.env.shared):**
```bash
# All existing COVERAGE_* variables continue to work
COVERAGE_THRESHOLD_EXCELLENT=90
COVERAGE_THRESHOLD_GOOD=80
COVERAGE_BADGE_STYLE=flat
COVERAGE_REPORT_THEME=github-dark
# ... all 40+ variables unchanged
```

**Standalone Users:**
```bash
# Set environment variables directly
export COVERAGE_THRESHOLD_EXCELLENT=90
export COVERAGE_THRESHOLD_GOOD=80
export COVERAGE_BADGE_STYLE=flat

# Or use in CI/CD
COVERAGE_THRESHOLD_GOOD=80 go-coverage complete --input coverage.txt
```

Note: YAML configuration to be added in future versions based on user demand.

### Usage Examples

**GitHub Actions:**
```yaml
- name: Coverage
  run: |
    go test -coverprofile=coverage.txt ./...
    go install github.com/mrz1836/go-coverage@latest
    go-coverage complete --input coverage.txt
```

**GitLab CI:**
```yaml
coverage:
  script:
    - go test -coverprofile=coverage.txt ./...
    - go install github.com/mrz1836/go-coverage@latest
    - go-coverage complete --input coverage.txt
```

**Local Development:**
```bash
# Generate coverage
go test -coverprofile=coverage.txt ./...

# Install tool
go install github.com/mrz1836/go-coverage@latest

# Generate reports
go-coverage complete --input coverage.txt
```

## Implementation Timeline (Streamlined)

- **Phase 0**: Pre-Migration Analysis (2-3 hours)
- **Phase 1**: Repository Creation (2-3 hours)
- **Phase 2**: Code Migration & Asset Embedding (4-5 hours)
- **Phase 3**: Configuration System - Env vars only (2-3 hours)
- **Phase 4**: Installation - Focus on go install (3-4 hours)
- **Phase 5**: CI/CD Integration (3-4 hours)
- **Phase 6**: go-broadcast Integration (3-4 hours)
- **Phase 7**: Testing and Documentation (4-5 hours)

**Total: 23-30 hours** (reduced from original 44-56 hours)

Advanced features and additional installation methods to be added incrementally after v1.0.0 release based on user feedback.

## Success Metrics

### Initial Release (v1.0.0)
- ✅ Zero breaking changes for existing users
- ✅ All `COVERAGE_*` env vars continue working
- ✅ `go install` works perfectly
- ✅ Assets properly embedded
- ✅ GitHub Pages deployment functional
- ✅ PR comments working
- ✅ Migration script tested

### Quality Targets
- Test coverage 80%+ (match go-pre-commit)
- Performance: <5s badge generation maintained
- Documentation: Comprehensive README with quickstart
- Zero external dependencies (single binary)

## Key Simplifications from Original Plan

1. **Configuration**: Environment variables only initially (no YAML)
2. **Installation**: Focus on `go install` first
3. **Features**: Core functionality only for v1.0.0
4. **Documentation**: Simple README following go-pre-commit pattern
5. **Timeline**: 23-30 hours vs 44-56 hours originally

## Lessons from go-pre-commit Success

1. **Simple binary name**: `go-coverage` not `gofortress-coverage`
2. **Keep `.env.shared`**: Don't introduce new config formats
3. **Focus on `go install`**: Primary distribution method
4. **Ship v1.0.0 early**: When core functionality is solid
5. **Add features incrementally**: Based on user feedback

## Conclusion

Following the successful pattern of the go-pre-commit migration, this streamlined plan transforms the Go Coverage System into a standalone tool while maintaining simplicity and focusing on core functionality first.

The refined approach ensures:
- **Zero breaking changes** - All `COVERAGE_*` environment variables continue working
- **Simplicity first** - No new config formats initially, just env vars
- **Fast delivery** - 23-30 hours vs 44-56 hours originally planned
- **Proven pattern** - Following go-pre-commit's successful migration approach
- **Incremental growth** - Ship v1.0.0 with core features, add more based on feedback

By focusing on `go install` as the primary distribution method and keeping configuration simple, we can deliver a working solution quickly and iterate based on real user needs, just as go-pre-commit successfully did.
