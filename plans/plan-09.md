# GoFortress Internal Coverage System - Go-Native Implementation Plan

## Executive Summary

This document outlines a comprehensive plan for a self-hosted, Go-native coverage system integrated directly into the GoFortress CI/CD pipeline. Built entirely in Go as a **bolt-on solution completely encapsulated within the `.github` folder**, this solution provides professional coverage tracking, badge generation, and reporting while maintaining the simplicity and performance that Go developers expect. The system leverages GitHub Pages for hosting static content and operates with zero external service dependencies.

**Key Architecture Decision**: The entire coverage system resides within `.github/coverage/` making it a portable, self-contained bolt-on that can be copied to any repository without polluting the main codebase.

## Vision Statement

Create a best-in-class Go-native coverage system that embodies Go's philosophy of simplicity and performance:
- **Go-First Design**: Built by Go developers, for Go developers
- **Single Binary**: One compiled tool that does everything - no runtime dependencies
- **Lightning Fast**: Leverage Go's performance for instant badge generation and reporting
- **Professional Quality**: GitHub-style badges and clean, accessible reports
- **Zero Dependencies**: Pure Go implementation with minimal external packages
- **Developer Friendly**: Simple CLI interface following Unix philosophy
- **Bolt-On Architecture**: Completely self-contained within `.github/coverage/`
- **Portable**: Can be copied to any repository as a complete unit
- **Non-Invasive**: Does not pollute the main repository structure

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     GoFortress CI/CD Pipeline                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────────────────┐               │
│  │ Go Test      │───▶│ gofortress-coverage      │               │
│  │ -coverprofile│    │ (Single Go Binary)       │               │
│  └──────────────┘    │                          │               │
│                      │ ├─ parse                 │               │
│                      │ ├─ badge                 │               │
│                      │ ├─ report                │               │
│                      │ ├─ history               │               │
│                      │ └─ comment               │               │
│                      └──────────┬───────────────┘               │
│                                 │                               │
│                                 ▼                               │
│                      ┌──────────────────────┐                   │
│                      │ GitHub Pages Deploy  │                   │
│                      │ (Static Files Only)  │                   │
│                      └──────────────────────┘                   │
└─────────────────────────────────────────────────────────────────┘

GitHub Pages Structure:
├── badges/
│   ├── main.svg
│   ├── develop.svg
│   └── pr-{number}.svg
├── reports/
│   ├── main/index.html
│   ├── develop/index.html
│   └── pr-{number}/index.html
├── api/
│   ├── coverage-summary.json
│   └── history.json
└── index.html (Dashboard)
```

## Implementation Roadmap

Each phase is designed to be completed in a single Claude Code session with clear deliverables and verification steps. The phases build upon each other, so they should be completed in order.

**IMPORTANT**: After completing each phase, update the status tracking document at `plans/plan-09-status.md` with:
- Mark the phase as completed (✓)
- Note any deviations from the plan
- Record actual metrics and timings
- Document any issues encountered
- Update the "Next Steps" section

### Phase 1: Foundation & Configuration (Session 1)
**Objective**: Establish infrastructure and remove Codecov dependencies

**Implementation Steps:**
1. Add coverage environment variables to `.github/.env.shared`
2. Create directory structure for coverage scripts
3. Remove all Codecov dependencies
4. Update documentation references

**Files to Modify:**
- `.github/.env.shared` - Add new variables
- `.github/workflows/fortress-test-suite.yml` - Remove codecov-action
- `.github/workflows/fortress.yml` - Remove codecov-token secret
- `.github/dependabot.yml` - Add coverage tool Go module monitoring
- `README.md` - Update badge URLs
- Delete: `codecov.yml`

**Verification Steps:**
```bash
# 1. Verify environment variables are added
grep "ENABLE_INTERNAL_COVERAGE" .github/.env.shared

# 2. Verify directory structure
ls -la .github/coverage/lib/

# 3. Verify Codecov removal
! grep -r "codecov" .github/workflows/ --exclude="*-status.md"
! test -f codecov.yml

# 4. Run lint and tests
make lint
make test
```

**Success Criteria:**
- ✅ All 25+ coverage environment variables present in .env.shared
- ✅ Directory structure created with proper permissions
- ✅ No references to codecov in workflows
- ✅ codecov.yml deleted
- ✅ All tests pass with no lint errors

**Status Update Required:**
After completing this phase, update `plans/plan-09-status.md`:
- Mark Phase 1 as completed (✓)
- Record actual implementation details
- Note any configuration changes made

#### 1.1 Environment Configuration Enhancement
Add to `.github/.env.shared`:

```bash
# ───────────────────────────────────────────────────────────────────────────────
# ENV: Internal Coverage System Configuration
# ───────────────────────────────────────────────────────────────────────────────
ENABLE_INTERNAL_COVERAGE=true                   # Enable internal coverage system (replaces Codecov)
COVERAGE_BADGE_STYLE=flat                       # Badge style: flat, flat-square, for-the-badge
COVERAGE_BADGE_LABEL=coverage                   # Badge label text
COVERAGE_BADGE_LOGO=go                          # Badge logo: go, github, custom URL
COVERAGE_BADGE_LOGO_COLOR=white                 # Logo color
COVERAGE_THRESHOLD_EXCELLENT=90                 # Coverage % for green badge
COVERAGE_THRESHOLD_GOOD=80                      # Coverage % for yellow-green badge
COVERAGE_THRESHOLD_ACCEPTABLE=70                # Coverage % for yellow badge
COVERAGE_THRESHOLD_LOW=60                       # Coverage % for orange badge (below is red)
COVERAGE_ENFORCE_THRESHOLD=false                # Fail builds below threshold
COVERAGE_FAIL_UNDER=70                          # Minimum acceptable coverage %
COVERAGE_PAGES_BRANCH=gh-pages                  # GitHub Pages branch name
COVERAGE_PAGES_AUTO_CREATE=true                 # Auto-create gh-pages branch if missing
COVERAGE_HISTORY_RETENTION_DAYS=90              # Days to retain coverage history
COVERAGE_REPORT_TITLE=GoFortress Coverage       # HTML report title
COVERAGE_REPORT_THEME=github-dark               # Report theme: github-light, github-dark, custom
COVERAGE_PR_COMMENT_ENABLED=true                # Enable PR coverage comments
COVERAGE_PR_COMMENT_BEHAVIOR=update             # Comment behavior: new, update, delete-and-new (prevents spam)
COVERAGE_PR_COMMENT_SHOW_TREE=true              # Show file tree in PR comments
COVERAGE_PR_COMMENT_SHOW_MISSING=true           # Highlight uncovered lines in PR
COVERAGE_SLACK_WEBHOOK_ENABLED=false            # Enable Slack notifications
COVERAGE_SLACK_WEBHOOK_URL=                     # Slack webhook URL (secret)
COVERAGE_BADGE_BRANCHES=master,development      # Branches to generate badges for
COVERAGE_CLEANUP_PR_AFTER_DAYS=7                # Clean up PR coverage data after merge
COVERAGE_ENABLE_TREND_ANALYSIS=true             # Enable historical trend tracking
COVERAGE_ENABLE_PACKAGE_BREAKDOWN=true          # Show package-level coverage
COVERAGE_ENABLE_COMPLEXITY_ANALYSIS=false       # Analyze code complexity (future)
ENABLE_INTERNAL_COVERAGE_TESTS=true             # Run coverage tool tests in CI

# Coverage Exclusion Configuration
COVERAGE_EXCLUDE_PATHS=test/,vendor/,examples/,third_party/,testdata/  # Comma-separated paths to exclude
COVERAGE_EXCLUDE_FILES=*_test.go,*.pb.go,*_mock.go,mock_*.go          # Comma-separated file patterns to exclude
COVERAGE_EXCLUDE_PACKAGES=                      # Additional packages to exclude (comma-separated)
COVERAGE_INCLUDE_ONLY_PATHS=                    # If set, only include these paths (comma-separated)
COVERAGE_EXCLUDE_GENERATED=true                 # Exclude generated files (detected by header)
COVERAGE_EXCLUDE_TEST_FILES=true                # Exclude test files from coverage
COVERAGE_MIN_FILE_LINES=10                      # Minimum lines in file to include in coverage

# Logging and Debugging Configuration
COVERAGE_LOG_LEVEL=info                         # debug, info, warn, error
COVERAGE_LOG_FORMAT=json                        # json, text, pretty
COVERAGE_LOG_FILE=/tmp/coverage.log             # Log file path
COVERAGE_LOG_MAX_SIZE=10MB                      # Max log file size
COVERAGE_LOG_RETENTION_DAYS=7                   # Log retention
COVERAGE_DEBUG_MODE=false                       # Enable verbose debugging
COVERAGE_TRACE_ERRORS=true                      # Include stack traces
COVERAGE_LOG_PERFORMANCE=true                   # Log timing metrics
COVERAGE_LOG_MEMORY_USAGE=true                  # Log memory consumption

# Monitoring and Metrics
COVERAGE_METRICS_ENABLED=true                   # Enable metrics collection
COVERAGE_METRICS_ENDPOINT=                      # Optional metrics endpoint
COVERAGE_METRICS_INCLUDE_ERRORS=true            # Track error metrics
COVERAGE_METRICS_INCLUDE_PERFORMANCE=true       # Track performance metrics
COVERAGE_METRICS_INCLUDE_USAGE=true             # Track usage metrics

# Error Injection for Testing
COVERAGE_TEST_MODE=false                        # Enable test mode
COVERAGE_INJECT_ERRORS=                         # Error injection: parser,api,storage
COVERAGE_ERROR_RATE=0                           # Error injection rate (0-1)
```

#### 1.2 Directory Structure Creation - Encapsulated Architecture
```bash
.github/
├── coverage/                     # Self-contained coverage system (bolt-on)
│   ├── cmd/
│   │   └── gofortress-coverage/  # Main CLI tool
│   │       ├── main.go           # Entry point with subcommands
│   │       ├── go.mod            # Separate Go module
│   │       ├── go.sum
│   │       └── cmd/              # Command implementations
│   │           ├── root.go       # Root command setup
│   │           ├── parse.go      # Parse coverage subcommand
│   │           ├── badge.go      # Generate badge subcommand
│   │           ├── report.go     # Generate report subcommand
│   │           ├── history.go    # Update history subcommand
│   │           └── comment.go    # PR comment subcommand
│   ├── internal/                 # Internal packages (Go conventions)
│   │   ├── parser/               # Coverage parsing logic
│   │   │   ├── parser.go
│   │   │   ├── parser_test.go
│   │   │   └── testdata/         # Test fixtures
│   │   ├── badge/                # SVG badge generation
│   │   │   ├── generator.go
│   │   │   ├── generator_test.go
│   │   │   └── templates.go      # Embedded SVG templates
│   │   ├── report/               # HTML report generation
│   │   │   ├── generator.go
│   │   │   ├── generator_test.go
│   │   │   └── templates/        # Embedded HTML templates
│   │   ├── history/              # Historical data tracking
│   │   │   ├── tracker.go
│   │   │   └── tracker_test.go
│   │   ├── github/               # GitHub API integration
│   │   │   ├── client.go
│   │   │   ├── pr_comment.go
│   │   │   └── client_test.go
│   │   └── config/               # Configuration handling
│   │       ├── config.go
│   │       └── config_test.go
│   └── README.md                 # Coverage system documentation
├── .env.shared                   # Environment variables
└── workflows/
    └── fortress-coverage.yml     # New coverage workflow
```

**Key Benefits of This Structure:**
- ✅ **Complete Encapsulation**: Everything coverage-related lives in `.github/coverage/`
- ✅ **Portable**: The entire `coverage/` folder can be copied to any repository
- ✅ **Separate Module**: Coverage tool has its own `go.mod` for isolated dependencies
- ✅ **Clean Separation**: Main repository remains uncluttered
- ✅ **Self-Documenting**: Coverage system includes its own README.md

#### 1.4 Label Configuration
The Go coverage tool itself has no external dependencies to track, but we need to add a label for tracking coverage-related PRs.

Add to `.github/labels.yml`:
```yaml
- name: "coverage-system"
  description: "Internal coverage system related"
  color: 1f6feb
```

#### 1.5 Dependabot Configuration for Coverage Tool
Since the coverage tool will be a separate Go module in `.github/coverage/cmd/gofortress-coverage/`, we need to add it to dependabot monitoring.

Add to `.github/dependabot.yml` after the main gomod entry:
```yaml
  # ──────────────────────────────────────────────────────────────
  # 1b. Coverage Tool Go Module (.github/coverage/cmd/gofortress-coverage)
  # ──────────────────────────────────────────────────────────────
  - package-ecosystem: "gomod"
    directory: "/.github/coverage/cmd/gofortress-coverage"
    target-branch: "master"
    schedule:
      interval: "weekly"
      day: "monday"
      time: "09:00"
      timezone: "America/New_York"
    allow:
      - dependency-type: "direct"
    groups:
      coverage-deps:
        patterns:
          - "*"
        update-types: ["minor", "patch"]
    open-pull-requests-limit: 5
    assignees: ["mrz1836"]
    labels: ["chore", "dependencies", "gomod", "coverage-system"]
    commit-message:
      prefix: "chore"
      include: "scope"
```

This ensures the coverage tool's dependencies (cobra, template libraries, etc.) stay up-to-date and secure.

#### 1.3 Codecov Removal Tasks
- Remove `codecov-token` from workflow secrets
- Delete `codecov.yml` configuration file
- Update README.md badge URLs
- Remove codecov-action from fortress-test-suite.yml
- Update documentation references
- Create comprehensive coverage tool as single Go binary

### Phase 2: Core Coverage Engine (Session 2)
**Objective**: Build the Go-native coverage processing tool with comprehensive testing and benchmarking

**Implementation Steps:**
1. Create main CLI application with cobra for subcommands
2. Implement coverage parser package (`internal/coverage/parser`)
3. Implement badge generator package (`internal/coverage/badge`)
4. Implement report generator package (`internal/coverage/report`)
5. Implement history tracker package (`internal/coverage/history`)
6. Write comprehensive unit tests with >90% coverage
7. Add benchmarks for all performance-critical paths
8. Set up proper error handling with context wrapping

**Files to Create (Encapsulated Structure):**
```
.github/coverage/
├── cmd/
│   └── gofortress-coverage/
│       ├── main.go              # CLI entry point
│       ├── go.mod               # Separate Go module
│       ├── go.sum               # Module dependencies
│       ├── cmd/                 # Command implementations
│       │   ├── root.go          # Root command setup
│       │   ├── parse.go         # Parse coverage subcommand
│       │   ├── badge.go         # Generate badge subcommand
│       │   ├── report.go        # Generate report subcommand
│       │   ├── history.go       # Update history subcommand
│       │   └── comment.go       # PR comment subcommand
│       └── cmd_test.go          # CLI integration tests
├── internal/
│   ├── parser/
│   │   ├── parser.go            # Coverage file parsing
│   │   ├── parser_test.go       # Unit tests
│   │   ├── parser_bench_test.go # Benchmarks
│   │   └── testdata/            # Test fixtures
│   │       ├── coverage.txt     # Sample coverage data
│   │       └── complex.txt      # Complex scenarios
│   ├── badge/
│   │   ├── generator.go         # SVG badge generation
│   │   ├── generator_test.go    # Unit tests
│   │   ├── generator_bench_test.go # Benchmarks
│   │   └── templates.go         # Embedded SVG templates
│   ├── report/
│   │   ├── generator.go         # HTML report generation
│   │   ├── generator_test.go    # Unit tests
│   │   ├── generator_bench_test.go # Benchmarks
│   │   └── templates/           # Embedded HTML templates
│   │       ├── report.html.tmpl
│   │       └── dashboard.html.tmpl
│   ├── history/
│   │   ├── tracker.go           # Historical data management
│   │   ├── tracker_test.go      # Unit tests
│   │   └── tracker_bench_test.go # Benchmarks
│   ├── github/
│   │   ├── client.go            # GitHub API client
│   │   ├── pr_comment.go        # PR commenting logic
│   │   └── client_test.go       # Unit tests with mocks
│   └── config/
│       ├── config.go            # Configuration handling
│       └── config_test.go       # Unit tests
└── README.md                    # Coverage system documentation
```

**Go Module Dependencies (.github/coverage/cmd/gofortress-coverage/go.mod):**
```go
// go.mod - Separate module for coverage tool
module github.com/mrz1836/go-broadcast/coverage

go 1.24

require (
    github.com/spf13/cobra v1.8.0
    github.com/google/go-github/v58 v58.0.0
    github.com/stretchr/testify v1.8.4
)
```

**Key Benefits of Separate Module:**
- ✅ **Isolated Dependencies**: Coverage tool dependencies don't affect main project
- ✅ **Version Independence**: Can update coverage tool dependencies separately
- ✅ **Reduced Complexity**: Main go.mod stays clean
- ✅ **Security**: Easier to audit coverage tool dependencies separately

**Verification Steps (Updated Paths):**
```bash
# 1. Build the tool (from encapsulated location)
cd .github/coverage/cmd/gofortress-coverage
go build -o gofortress-coverage

# 2. Run all tests with coverage (from coverage root)
cd .github/coverage
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -html=coverage.out -o coverage.html

# 3. Verify test coverage is >90%
go tool cover -func=coverage.out | grep total | awk '{print $3}'

# 4. Run benchmarks
go test -bench=. -benchmem ./internal/...

# 5. Run linting (from coverage tool directory)
cd cmd/gofortress-coverage
golangci-lint run ./...

# 6. Test CLI commands
./gofortress-coverage parse --file ../../internal/parser/testdata/coverage.txt
./gofortress-coverage badge --coverage 85.5 --output badge.svg
./gofortress-coverage report --file testdata/coverage.txt --output report.html

# 7. Run race detector tests
go test -race ./...

# 8. Check for vulnerabilities
govulncheck ./...
```

**Success Criteria:**
- ✅ Single Go binary compiles without errors
- ✅ All packages follow Go project layout standards
- ✅ Coverage parser correctly processes Go coverage format with proper exclusions
- ✅ Badge generator creates valid SVG files <100ms
- ✅ Report generator creates clean HTML reports <500ms
- ✅ All tests pass with >90% code coverage
- ✅ Benchmarks show performance meets targets (<2s badge, <10s report)
- ✅ No race conditions detected
- ✅ Zero security vulnerabilities from govulncheck
- ✅ All code passes golangci-lint with project settings
- ✅ Context propagation throughout call stack
- ✅ Proper error wrapping and handling

**Status Update Required:**
After completing this phase, update `plans/plan-09-status.md`:
- Mark Phase 2 as completed (✓)
- Document module performance metrics
- Note any design decisions or changes

#### 2.1 Coverage Parser (`internal/coverage/parser/parser.go`)
```go
// Package parser handles parsing of Go coverage profiles
package parser

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "path/filepath"
    "regexp"
    "strings"
)

// Parser processes Go coverage profiles and calculates metrics
type Parser struct {
    config *Config
}

    ExcludePaths      []string
    ExcludeFiles      []string  
    ExcludePackages   []string
    IncludeOnlyPaths  []string
    ExcludeGenerated  bool
    ExcludeTestFiles  bool
    MinimumFileLines  int
}

// FileCoverage represents coverage data for a single file
type FileCoverage struct {
    Path       string
    Statements int
    Covered    int
    Percentage float64
}

// New creates a new coverage parser
func New(cfg *Config) *Parser {
    if cfg == nil {
        cfg = &Config{
            MinimumFileLines: 10,
        }
    }
    return &Parser{config: cfg}
}

// Parse processes a coverage profile from the given reader
func (p *Parser) Parse(ctx context.Context, r io.Reader) (*CoverageData, error) {
    scanner := bufio.NewScanner(r)
    
    // Read mode line
    if !scanner.Scan() {
        return nil, fmt.Errorf("empty coverage profile")
    }
    
    modeLine := scanner.Text()
    mode := parseMode(modeLine)
    if mode == "" {
        return nil, fmt.Errorf("invalid mode line: %s", modeLine)
    }
    
    data := &CoverageData{
        Mode:  mode,
        Files: make(map[string]*FileCoverage),
    }
    
    // Parse coverage lines with context cancellation support
    for scanner.Scan() {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
        
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }
        
        if err := p.parseLine(line, data); err != nil {
            return nil, fmt.Errorf("parsing line: %w", err)
        }
    }
    
    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("reading coverage: %w", err)
    }
    
    return data, nil
}

// parseLine processes a single coverage line
func (p *Parser) parseLine(line string, data *CoverageData) error {
    // Format: filename:start.col,end.col statements count
    parts := strings.Fields(line)
    if len(parts) != 3 {
        return fmt.Errorf("invalid line format")
    }
    
    // Extract filename and positions
    filePos := strings.Split(parts[0], ":")
    if len(filePos) != 2 {
        return fmt.Errorf("invalid file position format")
    }
    
    filename := filePos[0]
    
    // Check exclusions
    if p.shouldExclude(filename) {
        return nil
    }
    
    // Parse statement count and coverage count
    stmts, err := strconv.Atoi(parts[1])
    if err != nil {
        return fmt.Errorf("invalid statement count: %w", err)
    }
    
    count, err := strconv.Atoi(parts[2])
    if err != nil {
        return fmt.Errorf("invalid coverage count: %w", err)
    }
    
    // Update file coverage
    if _, ok := data.Files[filename]; !ok {
        data.Files[filename] = &FileCoverage{
            Path: filename,
        }
    }
    
    file := data.Files[filename]
    file.Statements += stmts
    if count > 0 {
        file.Covered += stmts
    }
    
    return nil
}
```

#### 2.2 Badge Generator (`internal/coverage/badge/generator.go`)
```go
// Package badge generates SVG coverage badges
package badge

import (
    "bytes"
    "context"
    "embed"
    "fmt"
    "html/template"
    "io"
)

//go:embed templates/*.svg
var templates embed.FS

// Generator creates coverage badges in SVG format
type Generator struct {
    tmpl *template.Template
}

// New creates a new badge generator
func New() (*Generator, error) {
    tmpl, err := template.ParseFS(templates, "templates/*.svg")
    if err != nil {
        return nil, fmt.Errorf("parsing templates: %w", err)
    }
    
    return &Generator{tmpl: tmpl}, nil
}

// Generate creates an SVG badge for the given coverage percentage
func (g *Generator) Generate(ctx context.Context, percentage float64, style string) ([]byte, error) {
    color := g.getColor(percentage)
    
    data := struct {
        Label      string
        Percentage string
        Color      string
        Style      string
    }{
        Label:      "coverage",
        Percentage: fmt.Sprintf("%.1f%%", percentage),
        Color:      color,
        Style:      style,
    }
    
    var buf bytes.Buffer
    templateName := fmt.Sprintf("%s.svg", style)
    if err := g.tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
        return nil, fmt.Errorf("executing template: %w", err)
    }
    
    return buf.Bytes(), nil
}

// getColor returns the appropriate color based on coverage percentage
func (g *Generator) getColor(percentage float64) string {
    switch {
    case percentage >= 90:
        return "#3fb950" // Bright green
    case percentage >= 80:
        return "#90c978" // Green
    case percentage >= 70:
        return "#d29922" // Yellow
    case percentage >= 60:
        return "#f85149" // Orange
    default:
        return "#da3633" // Red
    }
}
```

#### 2.3 Report Generator (`internal/coverage/report/generator.go`)
```go
// Package report generates HTML coverage reports
package report

import (
    "context"
    "embed"
    "fmt"
    "html/template"
    "io"
    "time"
)

//go:embed templates/*
var templates embed.FS

// Generator creates HTML coverage reports
type Generator struct {
    tmpl *template.Template
}

// New creates a new report generator
func New() (*Generator, error) {
    funcs := template.FuncMap{
        "formatTime": func(t time.Time) string {
            return t.Format("2006-01-02 15:04:05")
        },
        "formatFloat": func(f float64) string {
            return fmt.Sprintf("%.2f", f)
        },
    }
    
    tmpl, err := template.New("report").Funcs(funcs).ParseFS(templates, "templates/*.html")
    if err != nil {
        return nil, fmt.Errorf("parsing templates: %w", err)
    }
    
    return &Generator{tmpl: tmpl}, nil
}

// Generate creates an HTML report from coverage data
func (g *Generator) Generate(ctx context.Context, data *CoverageData, w io.Writer) error {
    reportData := struct {
        Title      string
        Timestamp  time.Time
        Coverage   *CoverageData
        Metrics    *Metrics
    }{
        Title:     "Coverage Report",
        Timestamp: time.Now(),
        Coverage:  data,
        Metrics:   calculateMetrics(data),
    }
    
    if err := g.tmpl.ExecuteTemplate(w, "report.html", reportData); err != nil {
        return fmt.Errorf("executing template: %w", err)
    }
    
    return nil
}
```

#### 2.4 Testing Requirements

All Go packages must include comprehensive tests and benchmarks:

**Example Test (`internal/coverage/parser/parser_test.go`):**
```go
package parser_test

import (
    "context"
    "strings"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/YOUR_ORG/YOUR_REPO/internal/coverage/parser"
)

func TestParser_Parse(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *parser.CoverageData
        wantErr bool
    }{
        {
            name: "valid coverage data",
            input: `mode: set
github.com/org/repo/main.go:10.2,12.16 2 1
github.com/org/repo/main.go:12.16,14.3 1 0`,
            want: &parser.CoverageData{
                Mode: "set",
                Files: map[string]*parser.FileCoverage{
                    "github.com/org/repo/main.go": {
                        Path:       "github.com/org/repo/main.go",
                        Statements: 3,
                        Covered:    2,
                        Percentage: 66.67,
                    },
                },
            },
        },
        {
            name:    "empty coverage",
            input:   "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := parser.New(nil)
            ctx := context.Background()
            
            got, err := p.Parse(ctx, strings.NewReader(tt.input))
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            
            require.NoError(t, err)
            assert.Equal(t, tt.want.Mode, got.Mode)
            assert.Equal(t, len(tt.want.Files), len(got.Files))
        })
    }
}

func TestParser_ShouldExclude(t *testing.T) {
    cfg := &parser.Config{
        ExcludePaths: []string{"vendor/", "test/"},
        ExcludeFiles: []string{"*_test.go", "*.pb.go"},
    }
    
    p := parser.New(cfg)
    
    tests := []struct {
        path     string
        excluded bool
    }{
        {"vendor/github.com/pkg/errors/errors.go", true},
        {"internal/parser/parser_test.go", true},
        {"internal/parser/parser.go", false},
        {"api/v1/service.pb.go", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.path, func(t *testing.T) {
            got := p.ShouldExclude(tt.path)
            assert.Equal(t, tt.excluded, got)
        })
    }
}
```

**Example Benchmark (`internal/coverage/parser/parser_bench_test.go`):**
```go
package parser_test

import (
    "context"
    "strings"
    "testing"
    
    "github.com/YOUR_ORG/YOUR_REPO/internal/coverage/parser"
)

func BenchmarkParser_Parse(b *testing.B) {
    // Generate test data
    var sb strings.Builder
    sb.WriteString("mode: set\n")
    for i := 0; i < 1000; i++ {
        fmt.Fprintf(&sb, "github.com/org/repo/file%d.go:10.2,12.16 2 1\n", i)
    }
    
    input := sb.String()
    p := parser.New(nil)
    ctx := context.Background()
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := p.Parse(ctx, strings.NewReader(input))
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkBadgeGeneration(b *testing.B) {
    g, err := badge.New()
    if err != nil {
        b.Fatal(err)
    }
    
    ctx := context.Background()
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, err := g.Generate(ctx, 85.5, "flat")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

#### 2.5 Fortress Integration Testing Requirements

**Add to fortress-test-suite.yml:**
```yaml
- name: Test Coverage Tool
  run: |
    # Build the Go coverage tool
    go build -o gofortress-coverage ./cmd/gofortress-coverage
    
    # Run unit tests for the coverage tool
    go test -v -race ./internal/coverage/...
    
    # Run integration tests
    go test -v -tags=integration ./tests/integration/...
```

**Example Test - parser_test.go:**
```go
package parser

import (
    "context"
    "strings"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGoCoverageParser(t *testing.T) {
    t.Run("Parse", func(t *testing.T) {
        parser := NewParser(Config{
            ExcludePaths: []string{"vendor/", "test/"},
            ExcludeFiles: []string{"*_test.go", "*.pb.go"},
            MinimumFileLines: 10,
        })
        
        t.Run("should parse basic coverage data", func(t *testing.T) {
            coverageData := `mode: set
github.com/org/repo/main.go:10.2,12.16 2 1
github.com/org/repo/main.go:12.16,14.3 1 0`
            
            result, err := parser.Parse(context.Background(), strings.NewReader(coverageData))
            require.NoError(t, err)
            
            mainFile, exists := result.Files["github.com/org/repo/main.go"]
            assert.True(t, exists)
            assert.Equal(t, 3, mainFile.Statements)
            assert.Equal(t, 2, mainFile.Covered)
        })
        
        t.Run("should exclude vendor files", func(t *testing.T) {
            coverageData := `mode: set
github.com/org/repo/vendor/lib.go:10.2,12.16 2 1
github.com/org/repo/main.go:10.2,12.16 2 1`
            
            result, err := parser.Parse(context.Background(), strings.NewReader(coverageData))
            require.NoError(t, err)
            
            _, vendorExists := result.Files["github.com/org/repo/vendor/lib.go"]
            assert.False(t, vendorExists)
            
            _, mainExists := result.Files["github.com/org/repo/main.go"]
            assert.True(t, mainExists)
        })
        
        t.Run("should detect generated files", func(t *testing.T) {
            content := "// Code generated by protoc-gen-go. DO NOT EDIT."
            assert.True(t, parser.isGeneratedFile(content))
        })
    })
    
    t.Run("CalculateMetrics", func(t *testing.T) {
        t.Run("should calculate correct coverage percentage", func(t *testing.T) {
            parsedData := &ParsedData{
                Files: map[string]*FileData{
                    "main.go": {Statements: 100, Covered: 85},
                    "util.go": {Statements: 50, Covered: 45},
                },
            }
            
            metrics := CalculateMetrics(parsedData)
            
            assert.Equal(t, 150, metrics.TotalStatements)
            assert.Equal(t, 130, metrics.TotalCovered)
            assert.Equal(t, 86.67, metrics.Percentage)
        })
    })
}
```

**Testing Best Practices:**
- 100% code coverage requirement for all modules
- Unit tests for each function/method
- Integration tests for CLI commands
- Snapshot tests for SVG and HTML generation
- Mock file system operations
- Test error conditions and edge cases
- Performance benchmarks for large coverage files

#### 2.2 Badge Generator (`internal/coverage/badge/generator.go`)
```go
package badge

import (
    "fmt"
    "strings"
)

// Generator creates professional SVG badges matching GitHub's design language
type Generator struct {
    style  string
    colors map[string]string
}

// NewGenerator creates a badge generator with the specified style
func NewGenerator(style string) *Generator {
    return &Generator{
        style: style,
        colors: map[string]string{
            "excellent":  "#3fb950", // Bright green (90%+)
            "good":       "#90c978", // Green (80%+)
            "acceptable": "#d29922", // Yellow (70%+)
            "low":        "#f85149", // Orange (60%+)
            "poor":       "#da3633", // Red (<60%)
        },
    }
}

// Generate creates an SVG badge for the given coverage percentage
func (g *Generator) Generate(percentage float64, options Options) string {
    color := g.getColorForPercentage(percentage)
    label := options.Label
    if label == "" {
        label = "coverage"
    }
    
    // Generate SVG with proper styling
    return g.renderSVG(BadgeData{
        Label:      label,
        Message:    fmt.Sprintf("%.1f%%", percentage),
        Color:      color,
        Style:      g.style,
        Logo:       options.Logo,
        LogoColor:  options.LogoColor,
        AriaLabel:  fmt.Sprintf("Code coverage: %.1f percent", percentage),
    })
}

// GenerateTrendBadge creates a badge showing coverage trend
func (g *Generator) GenerateTrendBadge(current, previous float64) string {
    diff := current - previous
    var trend, color string
    
    switch {
    case diff > 0:
        trend = fmt.Sprintf("↑ +%.1f%%", diff)
        color = g.colors["good"]
    case diff < 0:
        trend = fmt.Sprintf("↓ %.1f%%", diff)
        color = g.colors["low"]
    default:
        trend = "→ 0%"
        color = "#8b949e" // neutral gray
    }
    
    return g.renderSVG(BadgeData{
        Label:   "coverage trend",
        Message: trend,
        Color:   color,
        Style:   g.style,
    })
}
```

#### 2.3 Report Generator (`internal/coverage/report/generator.go`)
```go
package report

import (
    "context"
    "embed"
    "fmt"
    "html/template"
    "strings"
    
    "github.com/go-broadcast/internal/coverage/parser"
)

//go:embed templates/*
var templates embed.FS

// Generator creates beautiful, interactive HTML coverage reports with cutting-edge UX
type Generator struct {
    theme       string
    templates   *template.Template
    designSystem DesignSystem
}

// DesignSystem defines the visual design tokens
type DesignSystem struct {
    Colors struct {
        // Modern color palette with proper contrast ratios
        Primary    string
        Success    string
        Warning    string
        Danger     string
        // Glassmorphism backgrounds
        Glass      string
        GlassHover string
    }
    Animations struct {
        // CSS animation definitions
        FadeIn  string
        SlideUp string
        Pulse   string
    }
}

// NewGenerator creates a report generator with the specified theme
func NewGenerator(theme string) (*Generator, error) {
    tmpl, err := template.ParseFS(templates, "templates/*.html")
    if err != nil {
        return nil, fmt.Errorf("parsing templates: %w", err)
    }
    
    g := &Generator{
        theme:     theme,
        templates: tmpl,
    }
    
    // Initialize design system based on theme
    g.initDesignSystem()
    
    return g, nil
}

// GenerateReport creates a comprehensive coverage report
func (g *Generator) GenerateReport(ctx context.Context, coverageData *parser.ParsedData, options ReportOptions) (string, error) {
    // Modern dashboard with server-rendered content
    // Progressive enhancement - works without JavaScript
    // Accessible keyboard navigation hints
    // Optimized for fast initial render
    
    data := struct {
        Coverage     *parser.ParsedData
        Options      ReportOptions
        Theme        string
        DesignSystem DesignSystem
        Features     Features
    }{
        Coverage:     coverageData,
        Options:      options,
        Theme:        g.theme,
        DesignSystem: g.designSystem,
        Features: Features{
            Search:           true,
            KeyboardNav:      true,
            VirtualScrolling: len(coverageData.Files) > 100,
            CodeMinimap:      options.ShowMinimap,
        },
    }
    
    var buf strings.Builder
    if err := g.templates.ExecuteTemplate(&buf, "report.html", data); err != nil {
        return "", fmt.Errorf("executing template: %w", err)
    }
    
    return buf.String(), nil
}

// GeneratePackageView creates the package-level view
func (g *Generator) GeneratePackageView(ctx context.Context, packageData PackageData) (string, error) {
    // Server-rendered package cards with CSS hover effects
    // SVG coverage rings (like GitHub's contribution graph)
    // Sortable tables with data attributes
    // Coverage heatmap using CSS gradients
    // Accessible quick actions
    
    var buf strings.Builder
    if err := g.templates.ExecuteTemplate(&buf, "package.html", packageData); err != nil {
        return "", fmt.Errorf("executing package template: %w", err)
    }
    
    return buf.String(), nil
}

// GenerateFileView creates the file-level coverage view
func (g *Generator) GenerateFileView(ctx context.Context, fileData FileData) (string, error) {
    // Syntax highlighting using chroma (Go library)
    // Coverage annotations in HTML
    // Accessible gutter indicators
    // Coverage diff mode support
    // Semantic HTML structure
    
    highlighted, err := g.highlightCode(fileData.Content, fileData.Language)
    if err != nil {
        return "", fmt.Errorf("highlighting code: %w", err)
    }
    
    fileData.HighlightedContent = highlighted
    
    var buf strings.Builder
    if err := g.templates.ExecuteTemplate(&buf, "file.html", fileData); err != nil {
        return "", fmt.Errorf("executing file template: %w", err)
    }
    
    return buf.String(), nil
}

// initDesignSystem sets up the design tokens based on theme
func (g *Generator) initDesignSystem() {
    switch g.theme {
    case "github-dark":
        g.designSystem.Colors.Primary = "#1f6feb"
        g.designSystem.Colors.Success = "#3fb950"
        g.designSystem.Colors.Warning = "#d29922"
        g.designSystem.Colors.Danger = "#f85149"
        g.designSystem.Colors.Glass = "rgba(255, 255, 255, 0.05)"
        g.designSystem.Colors.GlassHover = "rgba(255, 255, 255, 0.08)"
    default:
        // Light theme colors
        g.designSystem.Colors.Primary = "#0969da"
        g.designSystem.Colors.Success = "#1a7f37"
        g.designSystem.Colors.Warning = "#9a6700"
        g.designSystem.Colors.Danger = "#d1242f"
        g.designSystem.Colors.Glass = "rgba(0, 0, 0, 0.03)"
        g.designSystem.Colors.GlassHover = "rgba(0, 0, 0, 0.05)"
    }
    
    // Animations are CSS strings embedded in templates
    g.designSystem.Animations.FadeIn = "fadeIn 0.3s cubic-bezier(0.4, 0, 0.2, 1)"
    g.designSystem.Animations.SlideUp = "slideUp 0.4s cubic-bezier(0.34, 1.56, 0.64, 1)"
    g.designSystem.Animations.Pulse = "pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite"
}
```

#### 2.4 Coverage Exclusion System
The coverage system provides comprehensive exclusion capabilities to ensure accurate and meaningful coverage metrics:

**Exclusion Types:**
1. **Path Exclusions**: Entire directories (e.g., `vendor/`, `test/`, `examples/`)
2. **File Pattern Exclusions**: Wildcards (e.g., `*_test.go`, `*.pb.go`, `mock_*.go`)
3. **Package Exclusions**: Specific Go packages
4. **Generated File Detection**: Automatic detection via file headers
5. **Size-based Exclusions**: Files with fewer than N lines
6. **Include-only Mode**: Whitelist specific paths

**Configuration Examples:**
```bash
# Exclude common non-production code
COVERAGE_EXCLUDE_PATHS=test/,vendor/,examples/,third_party/,testdata/,docs/

# Exclude test files and generated code
COVERAGE_EXCLUDE_FILES=*_test.go,*.pb.go,*_mock.go,mock_*.go,bindata.go

# Exclude specific packages
COVERAGE_EXCLUDE_PACKAGES=github.com/org/repo/internal/testutil

# Only include specific paths (exclusive with exclude paths)
COVERAGE_INCLUDE_ONLY_PATHS=internal/,cmd/

# Automatic exclusions
COVERAGE_EXCLUDE_GENERATED=true      # Detects "DO NOT EDIT" headers
COVERAGE_EXCLUDE_TEST_FILES=true     # Excludes *_test.go files
COVERAGE_MIN_FILE_LINES=10          # Skip trivial files
```

**How Exclusions Work:**
1. **During Coverage Collection**: Go test already excludes test files by default
2. **During Processing**: The coverage parser filters out excluded files/packages
3. **In Reports**: Excluded files don't appear in coverage calculations or reports
4. **In PR Comments**: Only production code coverage is shown

This ensures that coverage metrics reflect only the production code that matters, not test helpers, generated code, or examples.

### Phase 3: Fortress Workflow Integration (Session 3)
**Objective**: Seamlessly integrate with existing GoFortress workflows

**Implementation Steps:**
1. Create new `fortress-coverage.yml` workflow
2. Modify `fortress-test-suite.yml` to call coverage workflow
3. Update `fortress.yml` to pass coverage token
4. Test workflow integration locally using act
5. Verify GitHub Pages permissions

**Files to Create/Modify:**
- Create: `.github/workflows/fortress-coverage.yml`
- Modify: `.github/workflows/fortress-test-suite.yml`
- Modify: `.github/workflows/fortress.yml`

**Local Testing with act:**
```bash
# Test coverage workflow locally
act -W .github/workflows/fortress-coverage.yml \
    --var ENABLE_INTERNAL_COVERAGE=true \
    --var COVERAGE_BADGE_STYLE=flat

# Test full pipeline
act push -W .github/workflows/fortress.yml
```

**Verification Steps:**
```bash
# 1. Verify workflow syntax
actionlint .github/workflows/fortress-coverage.yml

# 2. Check workflow integration
grep "fortress-coverage.yml" .github/workflows/fortress-test-suite.yml

# 3. Verify environment variable passing
grep "env-json" .github/workflows/fortress-coverage.yml

# 4. Test coverage artifact handling
# Run a test workflow and verify artifacts are created
```

**Success Criteria:**
- ✅ fortress-coverage.yml passes syntax validation
- ✅ Workflow is called from fortress-test-suite.yml
- ✅ Environment variables properly passed between workflows
- ✅ Coverage artifacts uploaded and downloaded correctly
- ✅ No regression in existing test suite

**Status Update Required:**
After completing this phase, update `plans/plan-09-status.md`:
- Mark Phase 3 as completed (✓)
- Record workflow execution times
- Document any integration challenges

#### 3.1 New Workflow: `fortress-coverage.yml`
```yaml
name: GoFortress (Coverage System)

on:
  workflow_call:
    inputs:
      coverage-file:
        description: "Path to coverage profile"
        required: true
        type: string
      branch-name:
        description: "Current branch name"
        required: true
        type: string
      pr-number:
        description: "PR number if applicable"
        required: false
        type: string
      commit-sha:
        description: "Commit SHA"
        required: true
        type: string
      env-json:
        description: "Environment configuration"
        required: true
        type: string

jobs:
  process-coverage:
    name: 📊 Process Coverage
    runs-on: ubuntu-latest
    steps:
      - name: 🔧 Setup environment
        # Parse env-json and set variables

      - name: 📥 Download coverage artifact
        # Get coverage data from test job

      - name: 📦 Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
          cache: true

      - name: 🔨 Build coverage tool
        run: |
          cd .github/coverage/cmd/gofortress-coverage
          go build -o gofortress-coverage

      - name: 🧪 Run coverage tool tests
        if: env.ENABLE_INTERNAL_COVERAGE_TESTS == 'true'
        run: |
          go test -race -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out

      - name: 🔍 Parse coverage data
        run: |
          ./.github/coverage/cmd/gofortress-coverage/gofortress-coverage parse \
            --file ${{ inputs.coverage-file }} \
            --output coverage-data.json

      - name: 🚨 Check coverage threshold
        if: env.COVERAGE_ENFORCE_THRESHOLD == 'true'
        run: |
          COVERAGE=$(jq -r '.percentage' coverage-data.json)
          THRESHOLD=${{ env.COVERAGE_FAIL_UNDER }}
          
          # Use awk for decimal comparison
          if awk -v cov="$COVERAGE" -v thresh="$THRESHOLD" 'BEGIN {exit (cov >= thresh)}'; then
            echo "❌ Coverage ${COVERAGE}% is below threshold ${THRESHOLD}%"
            echo "::error::Coverage ${COVERAGE}% is below the required threshold of ${THRESHOLD}%"
            exit 1
          else
            echo "✅ Coverage ${COVERAGE}% meets threshold ${THRESHOLD}%"
          fi

      - name: 🎨 Generate badge
        run: |
          COVERAGE=$(jq -r '.percentage' coverage-data.json)
          ./.github/coverage/cmd/gofortress-coverage/gofortress-coverage badge \
            --coverage $COVERAGE \
            --style ${{ env.COVERAGE_BADGE_STYLE }} \
            --output badge.svg

      - name: 📝 Generate report
        run: |
          ./.github/coverage/cmd/gofortress-coverage/gofortress-coverage report \
            --data coverage-data.json \
            --output report.html \
            --theme ${{ env.COVERAGE_REPORT_THEME }}

      - name: 📈 Update history
        run: |
          ./.github/coverage/cmd/gofortress-coverage/gofortress-coverage history \
            --add coverage-data.json \
            --branch ${{ inputs.branch-name }} \
            --commit ${{ inputs.commit-sha }}

      - name: 🚀 Deploy to GitHub Pages
        uses: peaceiris/actions-gh-pages@v3
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./.github/coverage/coverage-output
          destination_dir: ${{ inputs.branch-name }}
          keep_files: true

      - name: 💬 Comment on PR
        if: inputs.pr-number != ''
        run: |
          ./.github/coverage/cmd/gofortress-coverage/gofortress-coverage comment \
            --pr ${{ inputs.pr-number }} \
            --coverage coverage-data.json \
            --badge-url https://${{ github.repository_owner }}.github.io/${{ github.event.repository.name }}/badges/${{ inputs.branch-name }}.svg \
            --report-url https://${{ github.repository_owner }}.github.io/${{ github.event.repository.name }}/reports/${{ inputs.branch-name }}/
```

#### 3.2 Modify `fortress-test-suite.yml`
```yaml
# Replace codecov upload section with:
- name: 📊 Process internal coverage
  if: inputs.code-coverage-enabled == 'true'
  uses: ./.github/workflows/fortress-coverage.yml
  with:
    coverage-file: coverage.txt
    branch-name: ${{ github.ref_name }}
    pr-number: ${{ github.event.pull_request.number || '' }}
    commit-sha: ${{ github.sha }}
    env-json: ${{ inputs.env-json }}
```

### Phase 4: GitHub Pages & Storage (Session 4)
**Objective**: Implement robust storage and public hosting

**Implementation Steps:**
1. Create GitHub Pages setup script
2. Initialize gh-pages branch structure
3. Create main dashboard HTML/CSS/JS
4. Implement automatic deployment in workflow
5. Configure GitHub Pages in repository settings

**Files to Create (Encapsulated Structure):**
The Go coverage tool will handle GitHub Pages deployment directly. Static assets will be embedded in the binary:
```
.github/coverage/internal/report/templates/
├── dashboard.html              # Main dashboard template (embedded)
├── report.html                 # Coverage report template (embedded)
└── assets/
    └── style.css              # Minimal CSS (embedded in HTML)
```

**Manual Setup Required:**
```bash
# 1. Create orphan gh-pages branch (if not exists)
git checkout --orphan gh-pages
git rm -rf .
echo "# Coverage Reports" > README.md
git add README.md
git commit -m "Initialize gh-pages"
git push origin gh-pages

# 2. Enable GitHub Pages
# Go to Settings > Pages > Source: Deploy from branch > Branch: gh-pages
```

**Verification Steps:**
```bash
# 1. Test GitHub Pages setup
./gofortress-coverage pages setup --branch gh-pages

# 2. Verify gh-pages branch exists
git ls-remote --heads origin gh-pages

# 3. Test deployment
./gofortress-coverage pages deploy \
  --branch main \
  --badge badge.svg \
  --report report.html

# 4. Check GitHub Pages URL
curl -I https://USERNAME.github.io/REPO/badges/main.svg
curl -I https://USERNAME.github.io/REPO/reports/main/
```

**Success Criteria:**
- ✅ gh-pages branch created with proper structure
- ✅ Dashboard accessible via GitHub Pages URL
- ✅ Badge URLs resolve correctly
- ✅ Reports organized by branch/PR
- ✅ Automatic cleanup of old PR data works

**Status Update Required:**
After completing this phase, update `plans/plan-09-status.md`:
- Mark Phase 4 as completed (✓)
- Record GitHub Pages URLs
- Note storage usage patterns

#### 4.1 GitHub Pages Setup Command
The Go tool includes a `pages` subcommand for GitHub Pages management:
```go
// .github/coverage/cmd/gofortress-coverage/cmd/pages.go
package cmd

import (
    "context"
    "fmt"
    
    "github.com/spf13/cobra"
)

var pagesCmd = &cobra.Command{
    Use:   "pages",
    Short: "Manage GitHub Pages deployment",
    Long:  `Setup and deploy coverage reports to GitHub Pages`,
}

var pagesSetupCmd = &cobra.Command{
    Use:   "setup",
    Short: "Initialize GitHub Pages branch",
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx := context.Background()
        // Initialize gh-pages branch with proper structure
        // Create index.html dashboard
        // Set up directory hierarchy
        return setupGitHubPages(ctx)
    },
}
```

#### 4.2 Storage Structure & Organization
```
gh-pages/
├── badges/
│   ├── main.svg
│   ├── main-trend.svg
│   ├── develop.svg
│   ├── develop-trend.svg
│   └── pr/
│       ├── 123.svg
│       └── 124.svg
├── reports/
│   ├── main/
│   │   ├── index.html
│   │   ├── assets/
│   │   └── data/
│   ├── develop/
│   └── pr/
│       ├── 123/
│       └── 124/
├── api/
│   ├── summary.json
│   ├── history.json
│   └── trends.json
├── assets/
│   ├── css/
│   ├── js/
│   └── fonts/
└── index.html
```

#### 4.3 Dashboard Implementation - Modern, Professional UX
```html
<!-- index.html - Main Coverage Dashboard with Cutting-Edge Design -->
<!DOCTYPE html>
<html lang="en" data-theme="auto">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Coverage Dashboard | GoFortress</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="assets/css/dashboard.css">
  <link rel="icon" type="image/svg+xml" href="assets/icons/coverage.svg">
</head>
<body>
  <!-- Modern glass-morphism header with blur backdrop -->
  <header class="header-glass">
    <nav class="nav-container">
      <div class="logo-section">
        <svg class="logo-icon" viewBox="0 0 24 24"><!-- Custom logo --></svg>
        <h1 class="logo-text">Coverage<span class="accent">Hub</span></h1>
      </div>
      
      <!-- Global search with command palette -->
      <div class="search-container">
        <input type="search" placeholder="Search files, packages... (⌘K)" class="global-search">
        <div class="search-shortcuts">
          <kbd>⌘K</kbd>
        </div>
      </div>
      
      <!-- Theme switcher and settings -->
      <div class="header-actions">
        <button class="theme-toggle" aria-label="Toggle theme">
          <svg class="sun-icon"><!-- Sun icon --></svg>
          <svg class="moon-icon"><!-- Moon icon --></svg>
        </button>
        <button class="settings-btn" aria-label="Settings">
          <svg><!-- Gear icon --></svg>
        </button>
      </div>
    </nav>
  </header>

  <main class="dashboard-container">
    <!-- Hero section with animated metrics -->
    <section class="hero-section">
      <div class="hero-content">
        <h2 class="hero-title">Coverage Overview</h2>
        <p class="hero-subtitle">Track, analyze, and improve your code coverage</p>
      </div>
      
      <!-- Animated metric cards with gradients -->
      <div class="metrics-grid">
        <article class="metric-card metric-card--primary">
          <div class="metric-icon">
            <svg><!-- Coverage icon --></svg>
          </div>
          <div class="metric-content">
            <h3 class="metric-label">Total Coverage</h3>
            <div class="metric-value-container">
              <span class="metric-value" data-count="85.4">0</span>
              <span class="metric-unit">%</span>
            </div>
            <div class="metric-trend trend-up">
              <svg><!-- Trend arrow --></svg>
              <span>+2.3% from last week</span>
            </div>
          </div>
          <!-- Animated progress ring -->
          <svg class="progress-ring" viewBox="0 0 120 120">
            <circle class="progress-ring-bg"/>
            <circle class="progress-ring-fill" style="--progress: 85.4"/>
          </svg>
        </article>
        
        <!-- More metric cards with different styles -->
        <article class="metric-card metric-card--success"><!-- Files covered --></article>
        <article class="metric-card metric-card--warning"><!-- Lines to cover --></article>
        <article class="metric-card metric-card--info"><!-- Packages tracked --></article>
      </div>
    </section>

    <!-- Interactive branch selector with live preview -->
    <section class="branch-section">
      <header class="section-header">
        <h2 class="section-title">Branch Coverage</h2>
        <div class="branch-selector">
          <button class="branch-dropdown">
            <svg><!-- Git branch icon --></svg>
            <span>main</span>
            <svg><!-- Chevron --></svg>
          </button>
        </div>
      </header>
      
      <!-- Branch cards with hover effects and quick actions -->
      <div class="branch-grid">
        <article class="branch-card" data-branch="main">
          <div class="branch-header">
            <h3 class="branch-name">main</h3>
            <div class="branch-badge">Protected</div>
          </div>
          <div class="branch-coverage">
            <img src="badges/main.svg" alt="Main coverage" class="coverage-badge">
            <div class="coverage-details">
              <div class="coverage-bar">
                <div class="coverage-fill" style="--coverage: 85.4"></div>
              </div>
              <div class="coverage-stats">
                <span>2,451 / 2,867 lines</span>
              </div>
            </div>
          </div>
          <div class="branch-actions">
            <a href="reports/main/" class="action-link">View Report →</a>
            <button class="action-menu">⋯</button>
          </div>
        </article>
        <!-- More branch cards -->
      </div>
    </section>

    <!-- Beautiful trend visualization with Chart.js -->
    <section class="trends-section">
      <header class="section-header">
        <h2 class="section-title">Coverage Trends</h2>
        <div class="trend-controls">
          <div class="time-selector">
            <button class="time-btn active">1W</button>
            <button class="time-btn">1M</button>
            <button class="time-btn">3M</button>
            <button class="time-btn">1Y</button>
            <button class="time-btn">All</button>
          </div>
        </div>
      </header>
      
      <div class="chart-container">
        <canvas id="trendChart" class="trend-chart"></canvas>
        <!-- Tooltip overlay for detailed info -->
        <div class="chart-tooltip" style="display: none;">
          <div class="tooltip-date"></div>
          <div class="tooltip-coverage"></div>
          <div class="tooltip-commit"></div>
        </div>
      </div>
    </section>

    <!-- Recent PRs with live updates via WebSocket -->
    <section class="prs-section">
      <header class="section-header">
        <h2 class="section-title">Recent Pull Requests</h2>
        <button class="refresh-btn">
          <svg><!-- Refresh icon --></svg>
        </button>
      </header>
      
      <div class="pr-list">
        <article class="pr-card">
          <div class="pr-status">
            <div class="pr-coverage-indicator coverage-up"></div>
          </div>
          <div class="pr-content">
            <h4 class="pr-title">
              <a href="#123">#123</a> Add user authentication system
            </h4>
            <div class="pr-meta">
              <img src="avatar.jpg" class="pr-author-avatar" alt="Author">
              <span class="pr-author">john.doe</span>
              <span class="pr-time">2 hours ago</span>
            </div>
            <div class="pr-coverage-change">
              <span class="coverage-delta positive">+3.2%</span>
              <span class="coverage-current">88.6%</span>
            </div>
          </div>
          <div class="pr-actions">
            <a href="reports/pr-123/" class="pr-link">View Details</a>
          </div>
        </article>
        <!-- More PR cards with real-time updates -->
      </div>
    </section>

    <!-- Package explorer with search and filters -->
    <section class="packages-section">
      <header class="section-header">
        <h2 class="section-title">Package Coverage</h2>
        <input type="search" placeholder="Filter packages..." class="package-filter">
      </header>
      
      <div class="package-tree">
        <!-- Interactive file tree with expand/collapse -->
      </div>
    </section>
  </main>

  <!-- Command palette overlay -->
  <div class="command-palette" role="dialog" aria-hidden="true">
    <div class="command-input-wrapper">
      <input type="text" class="command-input" placeholder="Type a command or search...">
    </div>
    <div class="command-results">
      <!-- Dynamic command results -->
    </div>
  </div>

  <!-- Modern toast notifications -->
  <div class="toast-container" aria-live="polite"></div>
  
  <script type="module" src="assets/js/dashboard.js"></script>
</body>
</html>
```

#### 4.4 Modern CSS Design System
```css
/* dashboard.css - Professional, cutting-edge design system */
:root {
  /* Modern color palette */
  --color-primary: #1f6feb;
  --color-primary-hover: #388bfd;
  --color-success: #3fb950;
  --color-warning: #d29922;
  --color-danger: #f85149;
  
  /* Sophisticated neutrals */
  --color-bg: #0d1117;
  --color-bg-secondary: #161b22;
  --color-bg-tertiary: #21262d;
  --color-border: #30363d;
  --color-text: #c9d1d9;
  --color-text-secondary: #8b949e;
  
  /* Glass morphism */
  --glass-bg: rgba(255, 255, 255, 0.05);
  --glass-border: rgba(255, 255, 255, 0.1);
  --backdrop-blur: 12px;
  
  /* Smooth animations */
  --transition-base: 200ms cubic-bezier(0.4, 0, 0.2, 1);
  --transition-smooth: 300ms cubic-bezier(0.4, 0, 0.2, 1);
  
  /* Professional typography */
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  --font-mono: 'SF Mono', Monaco, Consolas, monospace;
}

/* Smooth theme transitions */
* {
  transition: background-color var(--transition-base),
              color var(--transition-base),
              border-color var(--transition-base);
}

/* Glass morphism header */
.header-glass {
  position: sticky;
  top: 0;
  z-index: 100;
  background: var(--glass-bg);
  backdrop-filter: blur(var(--backdrop-blur));
  border-bottom: 1px solid var(--glass-border);
}

/* Animated metric cards */
.metric-card {
  position: relative;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 24px;
  transition: all var(--transition-smooth);
  overflow: hidden;
}

.metric-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, var(--gradient-start), var(--gradient-end));
  transform: translateX(-100%);
  animation: shimmer 2s infinite;
}

.metric-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

/* Animated progress rings */
.progress-ring {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 60px;
  height: 60px;
}

.progress-ring-fill {
  stroke-dasharray: calc(var(--progress) * 3.14) 314;
  transition: stroke-dasharray 1s cubic-bezier(0.4, 0, 0.2, 1);
  animation: rotate 2s linear infinite;
}

/* Smooth hover effects */
.branch-card {
  transition: all var(--transition-smooth);
  cursor: pointer;
}

.branch-card:hover {
  background: var(--color-bg-tertiary);
  border-color: var(--color-primary);
}

/* Modern tooltips */
.tooltip {
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border);
  backdrop-filter: blur(var(--backdrop-blur));
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  animation: fadeIn var(--transition-base);
}

/* Command palette (Cmd+K) */
.command-palette {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.8);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 20vh;
  opacity: 0;
  visibility: hidden;
  transition: all var(--transition-base);
}

.command-palette[data-open="true"] {
  opacity: 1;
  visibility: visible;
}

/* Mobile responsive with touch-optimized interactions */
@media (max-width: 768px) {
  .metrics-grid {
    grid-template-columns: 1fr;
  }
  
  .metric-card {
    padding: 16px;
  }
  
  /* Touch-friendly tap targets */
  button, a {
    min-height: 44px;
    min-width: 44px;
  }
}

/* Dark mode with smooth transitions */
[data-theme="light"] {
  --color-bg: #ffffff;
  --color-bg-secondary: #f6f8fa;
  --color-text: #24292f;
  /* ... light theme colors ... */
}

/* Accessibility - focus styles */
:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

/* Reduced motion support */
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

### Phase 5: Pull Request Integration (Session 5)
**Objective**: Enhance PR workflow with intelligent coverage feedback that avoids comment spam and provides status checks

**Implementation Steps:**
1. Implement PR comment formatter in `internal/coverage/github/pr_comment.go`
2. Add coverage comparison logic with proper base branch handling
3. Create PR-specific badge generation with unique naming
4. Use go-github client for GitHub API integration
5. Write comprehensive tests with mocked GitHub API
6. Leverage the threshold enforcement from Phase 3 to create GitHub status checks that can block PR merging

**Files to Create/Modify:**
- Implement: `internal/coverage/github/pr_comment.go`
- Add tests: `internal/coverage/github/pr_comment_test.go`
- Already modified: `.github/workflows/fortress-coverage.yml` (PR comment step added)

**Testing PR Comments:**
```bash
# Test PR comment generation locally
export GITHUB_TOKEN="test"
./gofortress-coverage comment \
  --pr 123 \
  --current 85.5 \
  --base 83.2 \
  --dry-run

# Test with actual coverage data
./gofortress-coverage comment \
  --pr 123 \
  --coverage coverage-data.json \
  --base-branch main \
  --dry-run
```

**Verification Steps:**
```bash
# 1. Test comment formatting
./gofortress-coverage comment \
  --pr 123 \
  --current 85.5 \
  --base 83.2 \
  --format-only

# 2. Test GitHub API integration with mock
go test ./internal/coverage/github -run TestPRComment

# 3. Integration test with real PR (non-destructive)
./gofortress-coverage comment \
  --pr $TEST_PR_NUMBER \
  --coverage coverage-data.json \
  --dry-run \
  --verbose
```

**Success Criteria:**
- ✅ PR comments show coverage changes clearly
- ✅ Comments are updated, not duplicated (prevents spam on multiple pushes)
- ✅ Existing comments are edited when coverage changes
- ✅ Coverage comparison is accurate
- ✅ Links to full reports work
- ✅ Handles first-time PRs gracefully
- ✅ Smart comment updates: only update if coverage changes significantly (>0.1%)
- ✅ GitHub status check created that can block PR merge when coverage drops below threshold
- ✅ Status check shows as "Coverage Check" with pass/fail based on COVERAGE_ENFORCE_THRESHOLD and COVERAGE_FAIL_UNDER

**Status Update Required:**
After completing this phase, update `plans/plan-09-status.md`:
- Mark Phase 5 as completed (✓)
- Document PR comment performance
- Note any GitHub API limitations

#### 5.1 PR Comment Template
```markdown
## Coverage Report 📊

**Current Coverage:** {{current}}% {{trend_emoji}}
**Target Coverage:** {{target}}%
**Change:** {{change_symbol}}{{change}}%

### Summary
{{#if coverage_increased}}
✅ Coverage increased by {{change}}% - great work!
{{else if coverage_decreased}}
⚠️ Coverage decreased by {{change}}% - please add tests for new code
{{else}}
✅ Coverage remained stable
{{/if}}

### File Changes
| File | Coverage | Change | Status |
|------|----------|--------|--------|
{{#each files}}
| {{name}} | {{coverage}}% | {{change}}% | {{status_emoji}} |
{{/each}}

### Uncovered Lines
{{#if uncovered_lines}}
<details>
<summary>Click to see uncovered lines</summary>

{{#each uncovered_lines}}
**{{file}}**
- Lines: {{lines}}
{{/each}}
</details>
{{/if}}

📈 [View Full Report]({{report_url}}) | 🏷️ [Coverage Badge]({{badge_url}})

---
<sub>Generated by GoFortress Coverage System</sub>
```

#### 5.2 Smart PR Analysis & Comment Management
- Detect new uncovered code
- Suggest test locations
- Compare against base branch
- Track coverage velocity
- **Anti-spam features**:
  - Find and update existing comment instead of creating new ones
  - Only update comment if coverage changes by >0.1%
  - Batch multiple rapid pushes (wait 30s before updating)
  - Use GitHub's comment reactions for minor updates
  - Delete outdated comments after PR merge

### Phase 6: Advanced Features (Session 6)
**Objective**: Add professional enhancements and analytics

**Implementation Steps:**
1. Add trend analysis to history tracker with JSON output
2. Enhance coverage history with time-series data
3. Add notification support via webhook (optional)
4. Implement simple coverage prediction based on trends
5. Generate static trend charts as SVG

**Files to Create:**
```
internal/coverage/
├── history/
│   ├── analyzer.go            # Analyze coverage trends
│   └── analyzer_test.go       # Trend analysis tests
├── notify/
│   ├── notifier.go           # Send notifications (webhooks)
│   └── notifier_test.go      # Notification tests
└── report/templates/
    └── trends.svg.tmpl       # SVG trend chart template
```

**Feature Implementation:**
```go
// internal/coverage/history/analyzer.go
package history

import (
    "context"
    "time"
)

type TrendAnalyzer struct {
    history []DataPoint
}

type DataPoint struct {
    Date     time.Time
    Coverage float64
    Branch   string
}

// AnalyzeTrends calculates coverage trends and predictions
func (ta *TrendAnalyzer) AnalyzeTrends(ctx context.Context) (*TrendReport, error) {
    // Calculate moving averages
    // Identify trend direction
    // Simple linear regression for prediction
    return &TrendReport{
        Direction:   "improving",
        Change:      2.3,
        Prediction:  87.5,
    }, nil
}
```

**Verification Steps:**
```bash
# 1. Test trend analysis
./gofortress-coverage history analyze \
  --branch main \
  --days 30 \
  --output trend-report.json

# 2. Generate trend chart
./gofortress-coverage report trends \
  --data trend-report.json \
  --output trends.svg

# 3. Test notification system (if enabled)
COVERAGE_SLACK_WEBHOOK_ENABLED=true \
COVERAGE_SLACK_WEBHOOK_URL=$TEST_WEBHOOK \
  ./gofortress-coverage notify \
  --event milestone \
  --coverage 90.0
```

**Success Criteria:**
- ✅ Trend charts display correctly
- ✅ Historical data tracked accurately
- ✅ Notifications sent for milestones (if enabled)
- ✅ Coverage predictions reasonable
- ✅ Analytics provide actionable insights

**Status Update Required:**
After completing this phase, update `plans/plan-09-status.md`:
- Mark Phase 6 as completed (✓)
- Record feature adoption metrics
- Document user feedback

#### 6.1 Trend Visualization with SVG Charts
Since we're building a Go-native solution, trend visualization will be done through server-side generated SVG charts that are embedded in the static HTML reports:

```go
// internal/coverage/report/chart.go
package report

import (
    "context"
    "fmt"
    "math"
    "strings"
    "time"
    
    "github.com/go-broadcast/internal/coverage/history"
)

// ChartOptions configures SVG chart generation
type ChartOptions struct {
    Width      int
    Height     int
    ShowGrid   bool
    ShowLegend bool
    Theme      Theme
}

// Theme defines chart colors and styles
type Theme struct {
    Background  string
    Text        string
    GridColor   string
    LineColor   string
    TargetColor string
}

// GenerateTrendChart creates an SVG trend chart from historical data
func GenerateTrendChart(ctx context.Context, data []history.DataPoint, opts ChartOptions) (string, error) {
    // Server-side SVG generation - no JavaScript dependencies
    // Clean, accessible, performant
    
    if len(data) == 0 {
        return "", fmt.Errorf("no data points provided")
    }
    
    // Calculate chart dimensions and scaling
    padding := 40
    chartWidth := opts.Width - 2*padding
    chartHeight := opts.Height - 2*padding
    
    // Find data bounds
    minCov, maxCov := findCoverageBounds(data)
    xScale := float64(chartWidth) / float64(len(data)-1)
    yScale := float64(chartHeight) / (maxCov - minCov)
    
    // Build SVG
    var svg strings.Builder
    svg.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">`, opts.Width, opts.Height))
    
    // Background
    svg.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="%s"/>`, opts.Width, opts.Height, opts.Theme.Background))
    
    // Grid lines
    if opts.ShowGrid {
        svg.WriteString(generateGridLines(padding, chartWidth, chartHeight, opts.Theme.GridColor))
    }
    
    // Coverage line path
    path := generateLinePath(data, padding, xScale, yScale, minCov, chartHeight)
    svg.WriteString(fmt.Sprintf(`<path d="%s" fill="none" stroke="%s" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>`, path, opts.Theme.LineColor))
    
    // Data points
    for i, point := range data {
        x := padding + int(float64(i)*xScale)
        y := padding + chartHeight - int((point.Coverage-minCov)*yScale)
        
        // Invisible larger hit area for accessibility
        svg.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="20" fill="transparent" class="hit-area"/>`, x, y))
        
        // Visible point on hover
        svg.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="4" fill="%s" class="data-point" opacity="0">`, x, y, opts.Theme.LineColor))
        svg.WriteString(`<animate attributeName="opacity" begin="mouseover" dur="0.1s" fill="freeze" to="1"/>`)
        svg.WriteString(`<animate attributeName="opacity" begin="mouseout" dur="0.1s" fill="freeze" to="0"/>`)
        svg.WriteString(`</circle>`)
        
        // Tooltip text
        tooltip := fmt.Sprintf("%.1f%% on %s", point.Coverage, point.Date.Format("Jan 2"))
        svg.WriteString(fmt.Sprintf(`<title>%s</title>`, tooltip))
    }
    
    // Axis labels
    svg.WriteString(generateAxisLabels(data, padding, chartWidth, chartHeight, opts.Theme.Text))
    
    // Legend
    if opts.ShowLegend {
        svg.WriteString(generateLegend(opts.Width, padding, opts.Theme))
    }
    
    svg.WriteString(`</svg>`)
    return svg.String(), nil
}

// Helper functions for clean code organization
func findCoverageBounds(data []history.DataPoint) (min, max float64) {
    min, max = 100.0, 0.0
    for _, point := range data {
        if point.Coverage < min {
            min = point.Coverage
        }
        if point.Coverage > max {
            max = point.Coverage
        }
    }
    // Add padding to bounds
    min = math.Max(0, min-5)
    max = math.Min(100, max+5)
    return
}

func generateLinePath(data []history.DataPoint, padding int, xScale, yScale, minCov float64, chartHeight int) string {
    var path strings.Builder
    
    for i, point := range data {
        x := padding + int(float64(i)*xScale)
        y := padding + chartHeight - int((point.Coverage-minCov)*yScale)
        
        if i == 0 {
            path.WriteString(fmt.Sprintf("M %d %d", x, y))
        } else {
            // Smooth curve using quadratic bezier
            prevX := padding + int(float64(i-1)*xScale)
            midX := (prevX + x) / 2
            prevY := padding + chartHeight - int((data[i-1].Coverage-minCov)*yScale)
            path.WriteString(fmt.Sprintf(" Q %d %d %d %d", midX, prevY, x, y))
        }
    }
    
    return path.String()
}

// Generate interactive HTML report with embedded SVG charts
func GenerateTrendReport(ctx context.Context, history []history.DataPoint, opts ReportOptions) (string, error) {
    // Generate multiple views of the data
    last30Days := filterLastNDays(history, 30)
    last90Days := filterLastNDays(history, 90)
    allTime := history
    
    // Generate charts for each time period
    chart30, err := GenerateTrendChart(ctx, last30Days, ChartOptions{
        Width: 800, Height: 400,
        ShowGrid: true, ShowLegend: false,
        Theme: opts.Theme,
    })
    if err != nil {
        return "", fmt.Errorf("generating 30-day chart: %w", err)
    }
    
    // Build HTML report with embedded charts
    html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Coverage Trends - %s</title>
    <style>
        %s
    </style>
</head>
<body>
    <div class="container">
        <h1>Coverage Trend Analysis</h1>
        
        <div class="chart-container">
            <h2>Last 30 Days</h2>
            %s
        </div>
        
        <div class="insights">
            %s
        </div>
    </div>
</body>
</html>
    `, opts.ProjectName, getCSSStyles(opts.Theme), chart30, generateInsights(last30Days))
    
    return html, nil
}
```

#### 6.2 Coverage Prediction Engine
```go
// internal/coverage/prediction/predictor.go
package prediction

import (
    "context"
    "fmt"
    "math"
    
    "github.com/go-broadcast/internal/coverage/history"
)

// Predictor analyzes coverage trends and predicts future coverage
type Predictor struct {
    history []history.DataPoint
}

// AnalyzeImpact estimates coverage impact of proposed changes
func (p *Predictor) AnalyzeImpact(ctx context.Context, diffData DiffData) (*ImpactAnalysis, error) {
    analysis := &ImpactAnalysis{
        EstimatedImpact: p.calculateImpact(diffData),
        RiskScore:       p.assessRisk(diffData),
        SuggestedTests:  p.suggestPriorityTests(diffData),
    }
    
    return analysis, nil
}

// PredictTrend uses simple linear regression for coverage prediction
func (p *Predictor) PredictTrend(days int) (float64, error) {
    if len(p.history) < 2 {
        return 0, fmt.Errorf("insufficient history for prediction")
    }
    
    // Simple linear regression
    var sumX, sumY, sumXY, sumX2 float64
    n := float64(len(p.history))
    
    for i, point := range p.history {
        x := float64(i)
        y := point.Coverage
        sumX += x
        sumY += y
        sumXY += x * y
        sumX2 += x * x
    }
    
    // Calculate slope and intercept
    slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
    intercept := (sumY - slope*sumX) / n
    
    // Predict future value
    futureX := float64(len(p.history) + days)
    prediction := slope*futureX + intercept
    
    // Clamp between 0 and 100
    return math.Max(0, math.Min(100, prediction)), nil
}

type ImpactAnalysis struct {
    EstimatedImpact float64
    RiskScore       int // 1-10
    SuggestedTests  []string
}

type DiffData struct {
    FilesChanged   []string
    LinesAdded     int
    LinesRemoved   int
    TestsModified  bool
}
```

#### 6.3 Notification System
```go
// internal/coverage/notify/notifier.go
package notify

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"
)

// Notifier handles multi-channel notifications
type Notifier struct {
    slackWebhook   string
    slackEnabled   bool
    emailEnabled   bool
    webhookEnabled bool
}

// NewNotifier creates a notification handler from environment config
func NewNotifier() *Notifier {
    return &Notifier{
        slackWebhook:   os.Getenv("COVERAGE_SLACK_WEBHOOK_URL"),
        slackEnabled:   os.Getenv("COVERAGE_SLACK_WEBHOOK_ENABLED") == "true",
        emailEnabled:   os.Getenv("COVERAGE_EMAIL_NOTIFICATIONS_ENABLED") == "true",
        webhookEnabled: os.Getenv("COVERAGE_GENERIC_WEBHOOK_ENABLED") == "true",
    }
}

// NotifyEvent sends notifications for coverage events
func (n *Notifier) NotifyEvent(ctx context.Context, event string, data NotificationData) error {
    switch event {
    case "milestone_reached":
        return n.notifyMilestone(ctx, data)
    case "coverage_dropped":
        return n.notifyCoverageDrop(ctx, data)
    case "weekly_summary":
        return n.notifyWeeklySummary(ctx, data)
    default:
        return fmt.Errorf("unknown event type: %s", event)
    }
}

func (n *Notifier) notifyMilestone(ctx context.Context, data NotificationData) error {
    if !n.slackEnabled {
        return nil
    }
    
    message := SlackMessage{
        Text: fmt.Sprintf("🎉 Coverage milestone reached: %.1f%%!", data.Coverage),
        Attachments: []SlackAttachment{{
            Color: "good",
            Fields: []SlackField{
                {Title: "Project", Value: data.Project, Short: true},
                {Title: "Branch", Value: data.Branch, Short: true},
                {Title: "Previous", Value: fmt.Sprintf("%.1f%%", data.PreviousCoverage), Short: true},
                {Title: "Current", Value: fmt.Sprintf("%.1f%%", data.Coverage), Short: true},
            },
        }},
    }
    
    return n.sendSlackMessage(ctx, message)
}

func (n *Notifier) sendSlackMessage(ctx context.Context, msg SlackMessage) error {
    payload, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("marshaling slack message: %w", err)
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", n.slackWebhook, bytes.NewReader(payload))
    if err != nil {
        return fmt.Errorf("creating request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("sending slack message: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
    }
    
    return nil
}

// Types for Slack integration
type SlackMessage struct {
    Text        string            `json:"text"`
    Attachments []SlackAttachment `json:"attachments,omitempty"`
}

type SlackAttachment struct {
    Color  string       `json:"color"`
    Fields []SlackField `json:"fields"`
}

type SlackField struct {
    Title string `json:"title"`
    Value string `json:"value"`
    Short bool   `json:"short"`
}

type NotificationData struct {
    Project          string
    Branch           string
    Coverage         float64
    PreviousCoverage float64
    URL              string
    Timestamp        time.Time
}
```

### Phase 7: Production Deployment & Testing (Session 7)
**Objective**: Deploy to production with comprehensive testing

**Implementation Steps:**
1. Perform end-to-end testing
2. Deploy to production
3. Monitor initial rollout
4. Validate all systems operational

**Files to Modify:**
- `.github/.env.shared` - Enable internal coverage
- Workflow files - Remove Codecov references

**End-to-End Testing:**
```bash
# 1. Run full test suite with coverage
make test-cover

# 2. Verify coverage processing with Go tool
./gofortress-coverage parse --input coverage.txt --output coverage.json

# 3. Check badge generation
./gofortress-coverage badge generate --coverage 85 --output coverage.svg
test -f coverage.svg

# 4. Test GitHub Pages deployment
git checkout gh-pages
ls -la badges/ reports/

# 5. Verify all URLs work
curl -I https://USERNAME.github.io/REPO/badges/main.svg
```

**Production Deployment Checklist:**
```bash
# 1. Enable internal coverage system
sed -i 's/ENABLE_INTERNAL_COVERAGE=false/ENABLE_INTERNAL_COVERAGE=true/' .github/.env.shared

# 2. Disable Codecov
sed -i 's/ENABLE_CODE_COVERAGE=true/ENABLE_CODE_COVERAGE=false/' .github/.env.shared

# 3. Commit and push changes
git add -A
git commit -m "feat: replace Codecov with internal coverage system"
git push

# 4. Monitor first workflow run
gh run watch
```

**Verification Steps:**
```bash
# 1. Check new badges are displayed
curl -s https://raw.githubusercontent.com/USERNAME/REPO/main/README.md | grep "github.io"

# 2. Verify coverage reports accessible
curl -I https://USERNAME.github.io/REPO/reports/main/

# 3. Test PR comment functionality
# Create a test PR and verify comment appears

# 4. Validate no Codecov references remain
! grep -r "codecov" . --exclude-dir=.git --exclude="*-status.md"
```

**Success Criteria:**
- ✅ All documentation updated with new URLs
- ✅ Migration guide clear and complete
- ✅ End-to-end tests pass
- ✅ Production deployment successful
- ✅ First workflow run completes without errors
- ✅ Badges and reports publicly accessible
- ✅ PR comments working correctly

**Status Update Required:**
After completing this phase, update `plans/plan-09-status.md`:
- Mark Phase 7 as completed (✓)
- Document deployment metrics
- Note any issues encountered

### Phase 8: Documentation & Feature Showcase (Session 8)
**Objective**: Create comprehensive documentation showcasing all features

**Implementation Steps:**
1. Update README.md with feature showcase
2. Create detailed documentation in /docs/
3. Add migration guide from Codecov
4. Update CONTRIBUTING.md with coverage requirements
5. Create interactive feature demos

**Files to Create/Modify:**
- Modify: `README.md` - Add comprehensive feature section
- Create: `.github/coverage/docs/coverage-system.md` - Complete system documentation
- Create: `.github/coverage/docs/coverage-features.md` - Feature showcase with screenshots
- Create: `.github/coverage/docs/coverage-configuration.md` - Configuration reference
- Create: `.github/coverage/docs/coverage-api.md` - API documentation
- Create: `.github/coverage/docs/migrating-from-codecov.md` - Migration guide
- Modify: `CONTRIBUTING.md` - Add coverage guidelines

**Verification Steps:**
```bash
# 1. Verify all documentation links work
find docs -name '*.md' -exec grep -l 'http' {} \; | xargs -I {} sh -c 'echo "Checking {}" && grep -oE "https?://[^\)\s]+" {} | xargs -I URL curl -I -s URL | head -1'

# 2. Check for broken internal links
grep -r '\](' docs/ README.md | grep -v http | grep -oE '\([^\)]+\)' | sort | uniq

# 3. Validate markdown formatting
find docs -name '*.md' -exec markdownlint {} \;
```

**Success Criteria:**
- ✅ README.md showcases all major features
- ✅ Complete documentation in /docs/ directory
- ✅ Migration guide tested and validated
- ✅ All links functional
- ✅ Screenshots/demos included
- ✅ API documentation complete

**Status Update Required:**
After completing this phase, update `plans/plan-09-status.md`:
- Mark Phase 8 as completed (✓)
- Update overall project status to "Complete"
- Record final metrics and lessons learned
- Archive any temporary resources

#### 8.1 README.md Feature Showcase
```markdown
## 🚀 Coverage System Features

### 🎯 Core Features

#### Professional Coverage Badges
- **GitHub-style badges** with multiple themes
- **Real-time updates** on every push
- **Branch-specific badges** for master, development, and PRs
- **Trend indicators** showing coverage direction

![Coverage Demo](docs/images/badge-showcase.png)

#### Interactive Coverage Dashboard
- **Modern, responsive UI** with glass-morphism design
- **Command palette** (Cmd+K) for quick navigation
- **Dark/light theme** with automatic detection
- **Mobile-optimized** with touch gestures

[View Live Demo](https://mrz1836.github.io/go-broadcast)

#### Intelligent PR Comments
- **Smart updates** - edits existing comments instead of spamming
- **Visual diffs** showing coverage changes
- **File-level breakdown** with uncovered lines
- **Actionable insights** and suggestions

![PR Comment Example](docs/images/pr-comment.png)

#### Advanced Analytics
- **Historical trends** with interactive charts
- **Package-level analysis** with drill-down
- **Coverage predictions** for PR impact
- **Export capabilities** (PNG, SVG, PDF)

### 🛠️ Configuration

```bash
# Enable the coverage system
ENABLE_INTERNAL_COVERAGE=true

# Customize appearance
COVERAGE_BADGE_STYLE=flat-square
COVERAGE_REPORT_THEME=github-dark

# Set thresholds
COVERAGE_THRESHOLD_EXCELLENT=90
COVERAGE_ENFORCE_THRESHOLD=true
```

[Full Configuration Guide](.github/coverage/docs/coverage-configuration.md)

### 📊 Exclusion System

Intelligently exclude non-production code:
- Test files (`*_test.go`)
- Generated code (protobuf, mocks)
- Vendor dependencies
- Example code
- Small utility files

### 🔧 Go Testing

100% test coverage for all coverage system modules:
```bash
# Run all tests with coverage
go test -race -cover ./cmd/gofortress-coverage/...
go test -race -cover ./internal/coverage/...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 📈 Performance

- Badge generation: <2 seconds
- Report generation: <10 seconds
- PR comments: <5 seconds
- Dashboard load: <1 second

### 🌐 Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- Mobile browsers

### ♿ Accessibility

- WCAG 2.1 Level AA compliant
- Full keyboard navigation
- Screen reader optimized
- Reduced motion support
```

#### 8.2 Documentation Structure
```
docs/
├── coverage-system.md          # Complete system overview
├── coverage-features.md        # Detailed feature documentation
├── coverage-configuration.md   # Configuration reference
├── coverage-api.md            # API documentation
├── coverage-troubleshooting.md # Common issues and solutions
├── migrating-from-codecov.md  # Migration guide
└── images/
    ├── badge-showcase.png     # Badge examples
    ├── dashboard-hero.png     # Dashboard screenshot
    ├── pr-comment.png         # PR comment example
    ├── trend-chart.png        # Analytics visualization
    └── architecture.png       # System architecture diagram
```

#### 8.3 .github/coverage/docs/coverage-system.md Template
```markdown
# GoFortress Internal Coverage System

## Overview

The GoFortress Internal Coverage System is a comprehensive, self-hosted solution for code coverage tracking, visualization, and analysis. Built as a modern replacement for third-party services like Codecov, it provides complete control over your coverage data while delivering a superior user experience.

## Key Benefits

- **Zero External Dependencies**: All data stays within your GitHub organization
- **No Rate Limits**: Process coverage as often as needed
- **Cost Effective**: No subscription fees or usage limits
- **Privacy First**: Your code metrics never leave your infrastructure
- **Customizable**: Tailor the system to your specific needs

## Architecture

![Architecture Diagram](images/architecture.png)

The system consists of several key components:

1. **Coverage Parser**: Processes Go test coverage data
2. **Badge Generator**: Creates SVG badges for README files
3. **Report Generator**: Builds interactive HTML reports
4. **GitHub Pages Integration**: Hosts all public-facing content
5. **PR Commenter**: Provides intelligent feedback on pull requests

## Getting Started

### Prerequisites

- Go 1.21+
- GitHub repository with Actions enabled
- GitHub Pages enabled (automatic setup available)

### Quick Start

1. **Enable the system**:
   ```bash
   # In .github/.env.shared
   ENABLE_INTERNAL_COVERAGE=true
   ```

2. **Run the setup**:
   ```bash
   make coverage-setup
   ```

3. **View your coverage**:
   - Badge: `https://YOUR-ORG.github.io/YOUR-REPO/badges/main.svg`
   - Dashboard: `https://YOUR-ORG.github.io/YOUR-REPO/`

## Features

### Coverage Tracking
- Line-by-line coverage analysis
- Package-level breakdowns
- Historical trend tracking
- Branch comparison

### Visualization
- Interactive dashboards
- Animated charts and graphs
- Heat maps for quick insights
- Mobile-responsive design

### Integration
- Seamless CI/CD integration
- PR status checks
- Slack/Discord notifications
- API for custom integrations

### Developer Experience
- Fast processing (<1 minute)
- Intelligent caching
- Offline support
- Keyboard shortcuts

## Configuration

See [Configuration Guide](coverage-configuration.md) for detailed options.

## Troubleshooting

Common issues and solutions in our [Troubleshooting Guide](coverage-troubleshooting.md).

## Contributing

We welcome contributions! See our [Contributing Guidelines](../CONTRIBUTING.md).
```

#### 8.4 Testing & Validation
- Documentation link validation
- Screenshot generation
- Interactive demo creation
- Cross-browser testing
- Accessibility audit

## Success Metrics

### Performance Targets
- Badge generation: <2 seconds
- Report generation: <10 seconds
- PR comment: <5 seconds
- Full workflow: <1 minute

### Quality Metrics
- 100% badge availability
- <0.1% error rate
- 99.9% accuracy vs go test
- <24hr historical data lag

### User Experience Excellence

#### Visual Design
- **Modern Aesthetics**: GitHub-inspired design language with refined typography
- **Glass Morphism**: Subtle transparency effects with backdrop blur
- **Smooth Animations**: 60fps transitions with spring physics
- **Professional Color Palette**: Carefully chosen colors with WCAG AAA contrast
- **Dark/Light Themes**: Automatic theme detection with smooth transitions

#### Interaction Design
- **Instant Feedback**: <100ms response for all interactions
- **Micro-interactions**: Delightful hover states, loading animations
- **Keyboard Navigation**: Full keyboard support (j/k navigation, / for search)
- **Command Palette**: Cmd+K for quick actions (like VS Code)
- **Touch Optimized**: Gesture support, 44px minimum touch targets

#### Performance
- **Lightning Fast**: <1s initial load, <100ms subsequent navigation
- **Progressive Enhancement**: Works without JavaScript, enhanced with it
- **Virtual Scrolling**: Handle 10,000+ files without lag
- **Optimistic Updates**: Instant UI updates with background sync
- **Web Workers**: Heavy processing off the main thread

#### Accessibility
- **WCAG 2.1 Level AA**: Full compliance
- **Screen Reader Support**: Proper ARIA labels and live regions
- **Keyboard Navigation**: Complete keyboard accessibility
- **Focus Management**: Clear focus indicators, logical tab order
- **Reduced Motion**: Respects prefers-reduced-motion

#### Mobile Experience
- **Responsive Design**: Optimized layouts for all screen sizes
- **Touch Gestures**: Swipe to navigate, pinch to zoom charts
- **Offline Support**: Service worker for offline viewing
- **Native Feel**: Add to home screen, fullscreen support

#### Data Visualization
- **Interactive Charts**: Zoom, pan, hover for details
- **Real-time Updates**: Live coverage changes via WebSocket
- **Beautiful Animations**: Smooth data transitions
- **Multiple Views**: List, grid, tree, and graph visualizations
- **Export Options**: Download as PNG, SVG, or PDF

## Risk Mitigation

### Technical Risks
1. **GitHub Pages Limits**
   - Mitigation: Implement cleanup, compression
   - Fallback: External CDN for assets

2. **Large Repository Performance**
   - Mitigation: Incremental processing
   - Fallback: Sampling for approximation

3. **Concurrent Updates**
   - Mitigation: Lock mechanisms
   - Fallback: Retry with backoff

### Operational Risks
1. **Data Loss**
   - Mitigation: Regular backups
   - Fallback: Reconstruct from CI logs

2. **Breaking Changes**
   - Mitigation: Versioned APIs
   - Fallback: Legacy compatibility mode

## Maintenance Schedule

### Daily
- Automated PR cleanup
- Cache invalidation
- Health checks

### Weekly
- History optimization
- Performance reports
- Usage analytics

### Monthly
- Security updates
- Feature evaluation
- User feedback review

### Quarterly
- Major version updates
- Architecture review
- Capacity planning
- Dependabot security updates review

## Innovation Roadmap

### Near Term (3 months)
- Test coverage suggestions
- Complexity analysis
- Performance profiling

### Medium Term (6 months)
- AI-powered test generation
- Cross-repository analytics
- Team dashboards

### Long Term (12 months)
- Predictive coverage modeling
- Automated test prioritization
- Industry benchmarking

### Phase 9: GoFortress Dashboard - Unified Project Health (Session 9)
**Objective**: Transform the coverage dashboard into a comprehensive "GoFortress Dashboard" that aggregates all project health metrics

**Implementation Steps:**
1. Extend the dashboard architecture to support multiple metric types
2. Integrate GitHub Actions API to pull workflow data
3. Parse and display benchmark results from fortress-benchmarks.yml
4. Aggregate security scan results (CodeQL, OSSAR, OpenSSF Scorecard)
5. Create unified health score algorithm
6. Implement plugin architecture for future metrics

**Files to Create/Modify:**
```
internal/coverage/
└── dashboard/                    # New dashboard package
    ├── aggregator.go            # Metrics aggregation logic
    ├── github_actions.go        # GitHub Actions API integration
    ├── benchmarks.go            # Benchmark parsing and trending
    ├── security.go              # Security scan aggregation
    ├── health_score.go          # Unified project health scoring
    └── templates/
        ├── fortress-dashboard.html
        └── components/          # Reusable dashboard components

cmd/gofortress-coverage/cmd/
└── dashboard.go                 # New dashboard subcommand
```

**Dashboard Features:**
1. **Coverage Metrics** (existing)
   - Test coverage trends
   - Package-level breakdown
   - PR impact analysis

2. **Build & CI Metrics** (new)
   - Build success rates
   - Workflow execution times
   - Failure patterns analysis
   - Resource usage trends

3. **Performance Metrics** (new)
   - Benchmark results over time
   - Performance regression detection
   - Operation-specific trends
   - Memory allocation patterns

4. **Security Metrics** (new)
   - CodeQL scan results
   - OSSAR findings
   - OpenSSF Scorecard trends
   - Vulnerability timeline

5. **Code Quality Metrics** (new)
   - Go Report Card integration
   - Linting statistics
   - Complexity trends
   - Technical debt indicators

6. **Release Metrics** (new)
   - Release frequency
   - Changelog summaries
   - Version adoption rates
   - Breaking change tracking

**Implementation Details:**
```go
// internal/coverage/dashboard/aggregator.go
package dashboard

import (
    "context"
    "time"
)

// MetricType defines the types of metrics we can aggregate
type MetricType string

const (
    MetricTypeCoverage    MetricType = "coverage"
    MetricTypeBuild       MetricType = "build"
    MetricTypeBenchmark   MetricType = "benchmark"
    MetricTypeSecurity    MetricType = "security"
    MetricTypeQuality     MetricType = "quality"
    MetricTypeRelease     MetricType = "release"
)

// DashboardAggregator collects metrics from various sources
type DashboardAggregator struct {
    githubClient *github.Client
    metrics      map[MetricType]MetricCollector
}

// AggregateMetrics collects all metrics for the dashboard
func (da *DashboardAggregator) AggregateMetrics(ctx context.Context) (*DashboardData, error) {
    data := &DashboardData{
        Timestamp: time.Now(),
        Metrics:   make(map[MetricType]interface{}),
    }
    
    // Collect from each metric source in parallel
    for metricType, collector := range da.metrics {
        metric, err := collector.Collect(ctx)
        if err != nil {
            // Log but don't fail - show what we can
            continue
        }
        data.Metrics[metricType] = metric
    }
    
    // Calculate unified health score
    data.HealthScore = da.calculateHealthScore(data.Metrics)
    
    return data, nil
}
```

**Dashboard UI Enhancements:**
```html
<!-- GoFortress Dashboard - Unified View -->
<div class="fortress-dashboard">
  <!-- Castle-themed health indicator -->
  <div class="fortress-health">
    <div class="castle-icon">
      <svg><!-- Animated castle with health-based colors --></svg>
    </div>
    <div class="health-score">
      <span class="score-value">92</span>
      <span class="score-label">Fortress Health</span>
    </div>
  </div>
  
  <!-- Metric cards grid -->
  <div class="metrics-overview">
    <div class="metric-card coverage-card">
      <h3>Coverage</h3>
      <div class="metric-value">85.4%</div>
      <div class="metric-trend">↑ 2.3%</div>
    </div>
    
    <div class="metric-card build-card">
      <h3>Build Success</h3>
      <div class="metric-value">98.2%</div>
      <div class="metric-trend">→ stable</div>
    </div>
    
    <div class="metric-card benchmark-card">
      <h3>Performance</h3>
      <div class="metric-value">+5.2%</div>
      <div class="metric-trend">↑ faster</div>
    </div>
    
    <div class="metric-card security-card">
      <h3>Security Score</h3>
      <div class="metric-value">A+</div>
      <div class="metric-trend">0 critical</div>
    </div>
  </div>
  
  <!-- Detailed metric sections -->
  <div class="metric-details">
    <!-- Each metric type gets its own detailed view -->
  </div>
</div>
```

**Verification Steps:**
```bash
# 1. Test metrics aggregation
./gofortress-coverage dashboard aggregate \
  --repo $GITHUB_REPOSITORY \
  --token $GITHUB_TOKEN

# 2. Generate dashboard
./gofortress-coverage dashboard generate \
  --data metrics.json \
  --output dashboard.html

# 3. Test GitHub Actions integration
./gofortress-coverage dashboard actions \
  --workflow fortress.yml \
  --days 30

# 4. Verify unified health score
./gofortress-coverage dashboard health \
  --verbose
```

**Success Criteria:**
- ✅ Dashboard displays multiple metric types in unified view
- ✅ GitHub Actions data successfully integrated
- ✅ Benchmark trends visualized with performance indicators
- ✅ Security scan results aggregated and displayed
- ✅ Unified health score accurately reflects project state
- ✅ Plugin architecture allows easy addition of new metrics
- ✅ Dashboard remains performant with all metrics enabled
- ✅ Mobile-responsive design maintained

**Status Update Required:**
After completing this phase, update `plans/plan-09-status.md`:
- Mark Phase 9 as completed (✓)
- Document integrated metric sources
- Record performance impact
- Note future metric opportunities

## Conclusion

This comprehensive plan transforms coverage reporting from a third-party dependency into a powerful, integrated feature of the GoFortress CI/CD system. By controlling every aspect of the coverage pipeline, we can deliver a superior developer experience while maintaining complete data sovereignty and zero external dependencies.

The phased approach ensures smooth implementation with minimal disruption, while the modular architecture allows for continuous enhancement and customization. This isn't just a replacement for Codecov—it's an evolution in how we think about code coverage in modern development workflows.

With the addition of the GoFortress Dashboard in Phase 9, the system evolves beyond coverage into a comprehensive project health monitoring solution, showcasing the full power of the GoFortress CI/CD system.

## Implementation Timeline

- **Phase 1-2**: Foundation & Core Engine (2 sessions)
- **Phase 3-4**: Integration & Storage (2 sessions)
- **Phase 5-6**: PR Features & Analytics (2 sessions)
- **Phase 7-8**: Deployment & Documentation (2 sessions)
- **Phase 9**: GoFortress Dashboard (1 session)
- **Total**: 9 development sessions

Each phase builds upon the previous, ensuring a stable and tested system at every stage. The GoFortress Dashboard in Phase 9 can be implemented after the core coverage system is operational, allowing it to serve as a showcase for the entire GoFortress CI/CD ecosystem.
