# GoFortress Internal Coverage System - Complete Implementation Plan

## Executive Summary

This document outlines a comprehensive plan to replace Codecov with a self-hosted, enterprise-grade coverage badge and reporting system integrated directly into the GoFortress CI/CD pipeline. The solution leverages GitHub Pages for hosting, provides branch-specific badges, detailed coverage reports, and operates with zero external dependencies.

## Vision Statement

Create a best-in-class coverage system that surpasses traditional third-party solutions by providing:
- **Complete Control**: No external service dependencies or API rate limits
- **Enhanced Visibility**: Rich, interactive coverage reports with historical tracking
- **Professional Aesthetics**: GitHub-quality badges and beautiful report interfaces
- **Developer Delight**: Fast, accurate, and insightful coverage analytics
- **Cutting-Edge UX**: Modern, responsive, and delightful user experience that rivals industry leaders

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     GoFortress CI/CD Pipeline                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │ Go Test      │───▶│ Coverage     │───▶│ Badge        │       │
│  │ -coverprofile│    │ Processor    │    │ Generator    │       │
│  └──────────────┘    └──────────────┘    └──────────────┘       │
│                              │                     │            │
│                              ▼                     ▼            │
│                      ┌──────────────┐    ┌──────────────┐       │
│                      │ Report       │    │ GitHub       │       │
│                      │ Generator    │───▶│ Pages Deploy │       │
│                      └──────────────┘    └──────────────┘       │
│                              │                                  │
│                              ▼                                  │
│                      ┌──────────────┐                           │
│                      │ PR Commenter │                           │
│                      └──────────────┘                           │
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
COVERAGE_BADGE_BRANCHES=main,develop            # Branches to generate badges for
COVERAGE_CLEANUP_PR_AFTER_DAYS=7                # Clean up PR coverage data after merge
COVERAGE_ENABLE_TREND_ANALYSIS=true             # Enable historical trend tracking
COVERAGE_ENABLE_PACKAGE_BREAKDOWN=true          # Show package-level coverage
COVERAGE_ENABLE_COMPLEXITY_ANALYSIS=false       # Analyze code complexity (future)
ENABLE_INTERNAL_COVERAGE_TESTS=true             # Run JavaScript tests for coverage system

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

#### 1.2 Directory Structure Creation
```bash
.github/
├── coverage/
│       ├── lib/
│       │   ├── badge-generator.js      # SVG badge generation
│       │   ├── coverage-parser.js      # Parse Go coverage data
│       │   ├── report-generator.js     # HTML report generation
│       │   ├── history-tracker.js      # Historical data management
│       │   └── pr-commenter.js         # PR comment formatting
│       ├── generate-badge.js           # Main badge generation script
│       ├── process-coverage.js         # Main coverage processing script
│       ├── generate-report.js          # Main report generation script
│       ├── update-history.js           # History update script
│       └── comment-pr.js               # PR commenting script
└── workflows/
    └── fortress-coverage.yml           # New coverage workflow
```

#### 1.4 Dependabot Configuration Update
Add to `.github/dependabot.yml`:
```yaml
  # JavaScript dependencies for coverage system
  - package-ecosystem: "npm"
    directory: "/.github/coverage"
    schedule:
      interval: "weekly"
      day: "monday"
      time: "04:00"
    reviewers:
      - "YOUR_GITHUB_TEAM"
    labels:
      - "dependencies"
      - "npm"
      - "coverage-system"
    open-pull-requests-limit: 5
    groups:
      coverage-deps:
        patterns:
          - "*"
        update-types:
          - "minor"
          - "patch"
```

Add to `.github/labels.yml`:
```yaml
- name: "coverage-system"
  description: "Internal coverage system related"
  color: 1f6feb
```

#### 1.3 Codecov Removal Tasks
- Remove `codecov-token` from workflow secrets
- Delete `codecov.yml` configuration file
- Update README.md badge URLs
- Remove codecov-action from fortress-test-suite.yml
- Update documentation references
- Update `.github/dependabot.yml` to include npm ecosystem for coverage scripts

### Phase 2: Core Coverage Engine (Session 2)
**Objective**: Build the coverage processing and badge generation system with comprehensive testing

**Implementation Steps:**
1. Create coverage parser module (`lib/coverage-parser.js`)
2. Create badge generator module (`lib/badge-generator.js`)
3. Create report generator module (`lib/report-generator.js`)
4. Create main executable scripts
5. Add package.json for Node.js dependencies
6. Write comprehensive tests for all modules using Vitest
7. Set up ESLint and Prettier for code quality

**Files to Create:**
```
.github/coverage/
├── package.json                    # Node.js dependencies
├── lib/
│   ├── coverage-parser.js         # Parse Go coverage data
│   ├── badge-generator.js         # Generate SVG badges
│   ├── report-generator.js        # Generate HTML reports
│   ├── history-tracker.js         # Track coverage history
│   └── pr-commenter.js           # Format PR comments
├── test/
│   ├── coverage-parser.test.js   # Tests for coverage parser
│   ├── badge-generator.test.js   # Tests for badge generator
│   ├── report-generator.test.js  # Tests for report generator
│   ├── history-tracker.test.js   # Tests for history tracker
│   ├── pr-commenter.test.js      # Tests for PR commenter
│   └── fixtures/                  # Test fixtures
│       ├── coverage.txt           # Sample Go coverage data
│       └── coverage-complex.txt   # Complex coverage scenarios
├── generate-badge.js              # CLI: node generate-badge.js
├── process-coverage.js            # CLI: node process-coverage.js
├── generate-report.js             # CLI: node generate-report.js
├── vitest.config.js               # Vitest configuration
├── .eslintrc.json                 # ESLint configuration
└── .prettierrc.json               # Prettier configuration
```

**package.json Dependencies:**
```json
{
  "name": "gofortress-coverage",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "test": "vitest run",
    "test:watch": "vitest",
    "test:coverage": "vitest run --coverage",
    "test:ui": "vitest --ui",
    "lint": "eslint lib/*.js *.js",
    "lint:fix": "eslint lib/*.js *.js --fix",
    "format": "prettier --write \"**/*.{js,json,md}\"",
    "format:check": "prettier --check \"**/*.{js,json,md}\""
  },
  "dependencies": {
    "chart.js": "^4.4.0",
    "handlebars": "^4.7.8"
  },
  "devDependencies": {
    "@vitest/coverage-v8": "^1.2.0",
    "@vitest/ui": "^1.2.0",
    "eslint": "^8.56.0",
    "eslint-config-prettier": "^9.1.0",
    "jsdom": "^23.2.0",
    "prettier": "^3.2.0",
    "vitest": "^1.2.0"
  }
}
```

**Verification Steps:**
```bash
# 1. Install dependencies
cd .github/coverage && npm install

# 2. Run tests with coverage
npm run test:coverage

# 3. Run linting
npm run lint

# 4. Test coverage parser CLI
echo "mode: set
github.com/org/repo/main.go:10.2,12.16 2 1" > test.coverage
node process-coverage.js < test.coverage

# 5. Test badge generation CLI
COVERAGE=85 node generate-badge.js
test -f coverage.svg && echo "Badge generated"

# 6. Test report generation CLI
node generate-report.js < test.coverage
test -f coverage-report.html && echo "Report generated"
```

**Success Criteria:**
- ✅ All JavaScript modules created and syntax-valid
- ✅ Coverage parser correctly processes Go coverage format
- ✅ Badge generator creates valid SVG files
- ✅ Report generator creates HTML with proper styling
- ✅ All scripts executable with proper error handling
- ✅ 100% test coverage for all JavaScript modules
- ✅ All tests pass in CI environment
- ✅ ESLint and Prettier checks pass

**Status Update Required:**
After completing this phase, update `plans/plan-09-status.md`:
- Mark Phase 2 as completed (✓)
- Document module performance metrics
- Note any design decisions or changes

#### 2.1 Coverage Parser (`lib/coverage-parser.js`)
```javascript
/**
 * Parse Go coverage profile and extract metrics
 * Supports multiple profile formats: set, count, atomic
 */
class GoCoverageParser {
  constructor(options = {}) {
    this.options = {
      excludePaths: this.parseExcludeList(process.env.COVERAGE_EXCLUDE_PATHS),
      excludeFiles: this.parseExcludeList(process.env.COVERAGE_EXCLUDE_FILES),
      excludePackages: this.parseExcludeList(process.env.COVERAGE_EXCLUDE_PACKAGES),
      includeOnlyPaths: this.parseExcludeList(process.env.COVERAGE_INCLUDE_ONLY_PATHS),
      excludeGenerated: process.env.COVERAGE_EXCLUDE_GENERATED === 'true',
      excludeTestFiles: process.env.COVERAGE_EXCLUDE_TEST_FILES === 'true',
      minimumFileLines: parseInt(process.env.COVERAGE_MIN_FILE_LINES) || 10,
      ...options
    };
  }

  parseExcludeList(envVar) {
    return envVar ? envVar.split(',').map(s => s.trim()).filter(Boolean) : [];
  }

  shouldExcludeFile(filePath, fileContent) {
    // Check exclude paths
    if (this.options.excludePaths.some(path => filePath.includes(path))) {
      return true;
    }

    // Check exclude file patterns
    if (this.options.excludeFiles.some(pattern => {
      const regex = new RegExp(pattern.replace('*', '.*'));
      return regex.test(filePath);
    })) {
      return true;
    }

    // Check include only paths (if specified)
    if (this.options.includeOnlyPaths.length > 0) {
      if (!this.options.includeOnlyPaths.some(path => filePath.includes(path))) {
        return true;
      }
    }

    // Check for generated files
    if (this.options.excludeGenerated && this.isGeneratedFile(fileContent)) {
      return true;
    }

    // Check minimum file size
    const lineCount = fileContent.split('\n').length;
    if (lineCount < this.options.minimumFileLines) {
      return true;
    }

    return false;
  }

  isGeneratedFile(content) {
    const generatedPatterns = [
      /^\/\/ Code generated .* DO NOT EDIT\./m,
      /^\/\/ @generated/m,
      /^\/\* Generated by .* \*\//m,
      /^# Generated by/m
    ];
    return generatedPatterns.some(pattern => pattern.test(content));
  }

  parse(coverageData) {
    const lines = coverageData.split('\n');
    const mode = lines[0].match(/^mode: (\w+)/)?.[1] || 'set';
    const coverage = {};

    for (let i = 1; i < lines.length; i++) {
      const line = lines[i].trim();
      if (!line) continue;

      const match = line.match(/^(.+?):(\d+)\.(\d+),(\d+)\.(\d+) (\d+) (\d+)$/);
      if (!match) continue;

      const [, file, startLine, startCol, endLine, endCol, stmts, count] = match;

      // Apply exclusion rules
      if (this.shouldExcludeFile(file, '')) {
        continue;
      }

      if (!coverage[file]) {
        coverage[file] = { lines: {}, statements: 0, covered: 0 };
      }

      // Track coverage data...
    }

    return coverage;
  }

  calculateMetrics(parsedData) {
    // Total coverage percentage
    // Package-level breakdown
    // File-level statistics
    // Uncovered line ranges
  }

  generateSummary(metrics) {
    // Human-readable summary
    // JSON API format
    // Badge data format
  }
}
```

#### 2.5 Test Infrastructure (Vitest + JSDOM)

**vitest.config.js:**
```javascript
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'test/',
        '*.config.js',
        'coverage/'
      ],
      thresholds: {
        lines: 100,
        functions: 100,
        branches: 100,
        statements: 100
      }
    },
    globals: true,
    setupFiles: ['./test/setup.js']
  }
});
```

**Example Test - coverage-parser.test.js:**
```javascript
import { describe, it, expect, beforeEach } from 'vitest';
import { GoCoverageParser } from '../lib/coverage-parser.js';

describe('GoCoverageParser', () => {
  let parser;
  
  beforeEach(() => {
    parser = new GoCoverageParser({
      excludePaths: ['vendor/', 'test/'],
      excludeFiles: ['*_test.go', '*.pb.go'],
      minimumFileLines: 10
    });
  });

  describe('parse()', () => {
    it('should parse basic coverage data', () => {
      const coverageData = `mode: set
github.com/org/repo/main.go:10.2,12.16 2 1
github.com/org/repo/main.go:12.16,14.3 1 0`;
      
      const result = parser.parse(coverageData);
      
      expect(result['github.com/org/repo/main.go']).toBeDefined();
      expect(result['github.com/org/repo/main.go'].statements).toBe(3);
      expect(result['github.com/org/repo/main.go'].covered).toBe(2);
    });

    it('should exclude vendor files', () => {
      const coverageData = `mode: set
github.com/org/repo/vendor/lib.go:10.2,12.16 2 1
github.com/org/repo/main.go:10.2,12.16 2 1`;
      
      const result = parser.parse(coverageData);
      
      expect(result['github.com/org/repo/vendor/lib.go']).toBeUndefined();
      expect(result['github.com/org/repo/main.go']).toBeDefined();
    });

    it('should detect generated files', () => {
      const content = '// Code generated by protoc-gen-go. DO NOT EDIT.';
      expect(parser.isGeneratedFile(content)).toBe(true);
    });
  });

  describe('calculateMetrics()', () => {
    it('should calculate correct coverage percentage', () => {
      const parsedData = {
        'main.go': { statements: 100, covered: 85 },
        'util.go': { statements: 50, covered: 45 }
      };
      
      const metrics = parser.calculateMetrics(parsedData);
      
      expect(metrics.totalStatements).toBe(150);
      expect(metrics.totalCovered).toBe(130);
      expect(metrics.percentage).toBe(86.67);
    });
  });
});
```

**Testing Best Practices:**
- 100% code coverage requirement for all modules
- Unit tests for each function/method
- Integration tests for CLI commands
- Snapshot tests for SVG and HTML generation
- Mock file system operations
- Test error conditions and edge cases
- Performance benchmarks for large coverage files

#### 2.2 Badge Generator (`lib/badge-generator.js`)
```javascript
/**
 * Generate professional SVG badges matching GitHub's design language
 */
class CoverageBadgeGenerator {
  constructor(style = 'flat') {
    this.style = style;
    this.colors = {
      excellent: '#3fb950',  // Bright green (90%+)
      good: '#90c978',       // Green (80%+)
      acceptable: '#d29922', // Yellow (70%+)
      low: '#f85149',        // Orange (60%+)
      poor: '#da3633'        // Red (<60%)
    };
  }

  generate(percentage, options = {}) {
    // Calculate color based on thresholds
    // Generate SVG with proper styling
    // Support for logos and custom labels
    // Accessibility features (aria-labels)
  }

  generateTrendBadge(current, previous) {
    // Show coverage trend (↑ ↓ →)
    // Difference percentage
    // Animated indicators (optional)
  }
}
```

#### 2.3 Report Generator (`lib/report-generator.js`)
```javascript
/**
 * Generate beautiful, interactive HTML coverage reports with cutting-edge UX
 */
class CoverageReportGenerator {
  constructor(theme = 'github-dark') {
    this.theme = theme;
    this.templates = this.loadTemplates();
    this.designSystem = {
      colors: {
        // Modern color palette with proper contrast ratios
        primary: '#1f6feb',
        success: '#3fb950',
        warning: '#d29922',
        danger: '#f85149',
        // Glassmorphism backgrounds
        glass: 'rgba(255, 255, 255, 0.05)',
        glassHover: 'rgba(255, 255, 255, 0.08)'
      },
      animations: {
        // Smooth micro-interactions
        fadeIn: 'fadeIn 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
        slideUp: 'slideUp 0.4s cubic-bezier(0.34, 1.56, 0.64, 1)',
        pulse: 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite'
      }
    };
  }

  generateReport(coverageData, options = {}) {
    // Modern dashboard with animated metrics
    // Real-time search with instant results
    // Keyboard navigation (j/k for files, / for search)
    // Smooth transitions between views
    // Progressive enhancement for performance
    // Virtual scrolling for large file lists
    // Code minimap for quick navigation
  }

  generatePackageView(packageData) {
    // Interactive package cards with hover effects
    // Animated coverage rings (like GitHub's contribution graph)
    // Sortable/filterable file tables
    // Coverage heatmap visualization
    // Quick actions menu on hover
  }

  generateFileView(fileData) {
    // Syntax highlighting with VS Code themes
    // Inline coverage annotations
    // Gutter indicators with tooltips
    // Coverage diff mode (show changes)
    // Sticky file header while scrolling
    // Line blame integration (who wrote uncovered code)
    // Quick jump to test files
  }

  generateInteractiveElements() {
    // Smooth scroll-spy navigation
    // Breadcrumb trail with dropdowns
    // Command palette (Cmd+K)
    // Toast notifications for actions
    // Contextual help tooltips
    // Keyboard shortcut overlay (?)
  }
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

      - name: 📦 Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: '.github/coverage/package-lock.json'

      - name: 📦 Install dependencies
        run: |
          cd .github/coverage
          npm ci

      - name: 🧪 Run JavaScript tests
        if: env.ENABLE_INTERNAL_COVERAGE_TESTS == 'true'
        run: |
          cd .github/coverage
          npm run test:coverage
          npm run lint

      - name: 🔍 Parse coverage data
        # Run coverage parser
        # Calculate all metrics

      - name: 🎨 Generate badge
        # Create SVG badge
        # Multiple styles if configured

      - name: 📝 Generate report
        # Create HTML report
        # Package breakdown
        # Historical comparison

      - name: 📈 Update history
        # Track coverage over time
        # Maintain trend data

      - name: 🚀 Deploy to GitHub Pages
        # Commit to gh-pages branch
        # Organize by branch/PR

      - name: 💬 Comment on PR
        if: inputs.pr-number != ''
        # Post/update PR comment
        # Show coverage changes
        # Link to full report
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

**Files to Create:**
```
.github/coverage/
├── setup-pages.js              # Initialize gh-pages branch
├── deploy-to-pages.js          # Deploy coverage data
└── templates/
    ├── dashboard.html          # Main dashboard template
    ├── dashboard.css           # Dashboard styling
    └── dashboard.js            # Dashboard interactivity
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
# 1. Test setup script
node .github/coverage/setup-pages.js

# 2. Verify gh-pages branch exists
git ls-remote --heads origin gh-pages

# 3. Test deployment script
BRANCH_NAME=test node .github/coverage/deploy-to-pages.js

# 4. Check GitHub Pages URL
curl -I https://USERNAME.github.io/REPO/
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

#### 4.1 GitHub Pages Auto-Setup Script
```javascript
// scripts/coverage/setup-pages.js
async function setupGitHubPages() {
  // Check if gh-pages branch exists
  // Create if missing with initial structure
  // Set up directory hierarchy
  // Create index.html dashboard
  // Initialize history tracking
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
**Objective**: Enhance PR workflow with intelligent coverage feedback that avoids comment spam

**Implementation Steps:**
1. Create PR comment formatter (`lib/pr-commenter.js`)
2. Implement coverage comparison logic
3. Add PR-specific badge generation
4. Create GitHub API integration for comments
5. Test with mock PR data

**Files to Create/Modify:**
- Create: `.github/coverage/lib/pr-commenter.js`
- Create: `.github/coverage/comment-pr.js`
- Modify: `.github/workflows/fortress-coverage.yml` (add PR comment step)

**Testing PR Comments:**
```bash
# Mock PR comment generation
export GITHUB_TOKEN="test"
export PR_NUMBER="123"
export CURRENT_COVERAGE="85.5"
export BASE_COVERAGE="83.2"
node .github/coverage/comment-pr.js

# Test coverage comparison
echo '{"current": 85.5, "base": 83.2}' | \
  node .github/coverage/lib/pr-commenter.js
```

**Verification Steps:**
```bash
# 1. Test comment formatting
node -e "
  const commenter = require('.github/coverage/lib/pr-commenter.js');
  console.log(commenter.format(85.5, 83.2));
"

# 2. Verify GitHub API integration
# Create a test PR and run the workflow

# 3. Check comment updates vs new comments
# Verify existing comments are updated, not duplicated
```

**Success Criteria:**
- ✅ PR comments show coverage changes clearly
- ✅ Comments are updated, not duplicated (prevents spam on multiple pushes)
- ✅ Existing comments are edited when coverage changes
- ✅ Coverage comparison is accurate
- ✅ Links to full reports work
- ✅ Handles first-time PRs gracefully
- ✅ Smart comment updates: only update if coverage changes significantly (>0.1%)

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
1. Create trend visualization with Chart.js
2. Implement coverage history tracking
3. Add notification system (optional)
4. Create coverage prediction logic
5. Build advanced analytics dashboard

**Files to Create:**
```
.github/coverage/
├── lib/
│   ├── trend-analyzer.js       # Analyze coverage trends
│   ├── notifier.js            # Send notifications
│   └── predictor.js           # Predict coverage impact
└── templates/
    └── trends.html            # Trend visualization page
```

**Feature Implementation:**
```javascript
// Test trend analysis
const history = [
  {date: '2024-01-01', coverage: 80},
  {date: '2024-01-02', coverage: 82},
  {date: '2024-01-03', coverage: 81}
];
node -e "
  const analyzer = require('./lib/trend-analyzer.js');
  console.log(analyzer.analyze(history));
"
```

**Verification Steps:**
```bash
# 1. Test trend visualization
node .github/coverage/generate-trends.js

# 2. Verify Chart.js integration
grep "Chart" .github/coverage/templates/dashboard.html

# 3. Test notification system (if enabled)
COVERAGE_SLACK_WEBHOOK_ENABLED=true \
  node .github/coverage/lib/notifier.js
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

#### 6.1 Interactive Trend Dashboard with Professional Visualizations
```javascript
// Coverage trend visualization with cutting-edge UX
class CoverageTrendDashboard {
  constructor(containerId) {
    this.container = document.getElementById(containerId);
    this.chart = null;
    this.theme = this.detectTheme();
    
    // Professional chart configuration
    this.chartConfig = {
      type: 'line',
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: {
          mode: 'index',
          intersect: false,
        },
        plugins: {
          legend: {
            display: false // Custom legend instead
          },
          tooltip: {
            enabled: false // Custom tooltip
          },
          annotation: {
            annotations: {} // Commit markers
          }
        },
        scales: {
          x: {
            grid: {
              color: 'rgba(255, 255, 255, 0.1)',
              drawBorder: false
            },
            ticks: {
              color: '#8b949e',
              font: {
                family: 'Inter',
                size: 11
              }
            }
          },
          y: {
            beginAtZero: true,
            max: 100,
            grid: {
              color: 'rgba(255, 255, 255, 0.05)',
              drawBorder: false
            },
            ticks: {
              color: '#8b949e',
              callback: value => value + '%'
            }
          }
        },
        elements: {
          line: {
            tension: 0.4, // Smooth curves
            borderWidth: 3,
            borderCapStyle: 'round',
            borderJoinStyle: 'round'
          },
          point: {
            radius: 0, // Hidden by default
            hoverRadius: 6,
            hitRadius: 30, // Large touch target
            hoverBorderWidth: 2
          }
        },
        animation: {
          duration: 750,
          easing: 'easeInOutQuart'
        }
      }
    };
  }

  async loadData() {
    // Fetch with loading states
    this.showLoadingState();
    
    try {
      const data = await this.fetchHistoricalData();
      
      // Process with web workers for performance
      const processed = await this.processInWorker(data);
      
      // Add smooth data interpolation
      this.data = this.interpolateData(processed);
      
      // Calculate insights
      this.insights = this.calculateInsights(this.data);
      
    } catch (error) {
      this.showErrorState(error);
    }
  }

  render() {
    // Create gradient fills
    const gradient = this.createGradient();
    
    // Initialize Chart.js with custom styling
    this.chart = new Chart(this.container, {
      ...this.chartConfig,
      data: {
        labels: this.data.labels,
        datasets: [{
          label: 'Total Coverage',
          data: this.data.values,
          borderColor: '#1f6feb',
          backgroundColor: gradient,
          fill: true
        }, {
          label: 'Target',
          data: this.data.target,
          borderColor: '#3fb950',
          borderDash: [5, 5],
          borderWidth: 2,
          fill: false
        }]
      }
    });
    
    // Add interactive features
    this.addInteractivity();
    this.renderCustomTooltip();
    this.renderInsights();
    this.addKeyboardNavigation();
  }

  addInteractivity() {
    // Smooth zoom with mouse wheel
    this.container.addEventListener('wheel', (e) => {
      if (e.ctrlKey || e.metaKey) {
        e.preventDefault();
        this.handleZoom(e.deltaY);
      }
    });
    
    // Pan with drag
    let isPanning = false;
    this.container.addEventListener('mousedown', () => isPanning = true);
    this.container.addEventListener('mousemove', (e) => {
      if (isPanning) this.handlePan(e.movementX);
    });
    this.container.addEventListener('mouseup', () => isPanning = false);
    
    // Touch gestures for mobile
    this.addTouchGestures();
  }

  renderCustomTooltip() {
    // Beautiful custom tooltip that follows cursor
    const tooltip = document.createElement('div');
    tooltip.className = 'chart-tooltip';
    tooltip.innerHTML = `
      <div class="tooltip-header">
        <span class="tooltip-date"></span>
        <span class="tooltip-time"></span>
      </div>
      <div class="tooltip-body">
        <div class="tooltip-metric">
          <span class="tooltip-label">Coverage</span>
          <span class="tooltip-value"></span>
        </div>
        <div class="tooltip-commit">
          <img class="tooltip-avatar" src="" alt="">
          <div class="tooltip-commit-info">
            <span class="tooltip-commit-message"></span>
            <span class="tooltip-commit-author"></span>
          </div>
        </div>
      </div>
      <div class="tooltip-arrow"></div>
    `;
    
    this.container.appendChild(tooltip);
  }

  renderInsights() {
    // AI-powered insights panel
    const insights = document.createElement('div');
    insights.className = 'chart-insights';
    
    // Animated insight cards
    this.insights.forEach((insight, index) => {
      const card = this.createInsightCard(insight);
      card.style.animationDelay = `${index * 100}ms`;
      insights.appendChild(card);
    });
    
    this.container.parentElement.appendChild(insights);
  }

  // Smooth theme transitions
  detectTheme() {
    const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
    return {
      background: isDark ? '#0d1117' : '#ffffff',
      text: isDark ? '#c9d1d9' : '#24292f',
      grid: isDark ? 'rgba(255, 255, 255, 0.1)' : 'rgba(0, 0, 0, 0.1)'
    };
  }

  // Performance optimizations
  processInWorker(data) {
    return new Promise((resolve) => {
      const worker = new Worker('assets/js/coverage-worker.js');
      worker.postMessage({ type: 'process', data });
      worker.onmessage = (e) => {
        resolve(e.data);
        worker.terminate();
      };
    });
  }
}

// Initialize with smooth loading
document.addEventListener('DOMContentLoaded', () => {
  // Intersection observer for lazy loading
  const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        const dashboard = new CoverageTrendDashboard('trendChart');
        dashboard.loadData().then(() => dashboard.render());
        observer.unobserve(entry.target);
      }
    });
  });
  
  observer.observe(document.getElementById('trendChart'));
});
```

#### 6.2 Coverage Prediction Engine
```javascript
// ML-based coverage impact prediction
class CoveragePrediction {
  analyzeChanges(diffData) {
    // Estimate coverage impact
    // Suggest priority test areas
    // Risk assessment score
  }
}
```

#### 6.3 Notification System
```javascript
// Multi-channel notifications
class CoverageNotifications {
  async notify(event, data) {
    switch(event) {
      case 'milestone_reached':
        // Celebrate coverage milestones
        break;
      case 'coverage_dropped':
        // Alert on significant drops
        break;
      case 'weekly_summary':
        // Send weekly reports
        break;
    }
  }
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

# 2. Verify coverage processing
cat coverage.txt | node .github/coverage/process-coverage.js

# 3. Check badge generation
COVERAGE=85 node .github/coverage/generate-badge.js
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
- Create: `docs/coverage-system.md` - Complete system documentation
- Create: `docs/coverage-features.md` - Feature showcase with screenshots
- Create: `docs/coverage-configuration.md` - Configuration reference
- Create: `docs/coverage-api.md` - API documentation
- Create: `docs/migrating-from-codecov.md` - Migration guide
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
- **Branch-specific badges** for main, develop, and PRs
- **Trend indicators** showing coverage direction

![Coverage Demo](docs/images/badge-showcase.png)

#### Interactive Coverage Dashboard
- **Modern, responsive UI** with glass-morphism design
- **Command palette** (Cmd+K) for quick navigation
- **Dark/light theme** with automatic detection
- **Mobile-optimized** with touch gestures

[View Live Demo](https://your-org.github.io/your-repo)

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

[Full Configuration Guide](docs/coverage-configuration.md)

### 📊 Exclusion System

Intelligently exclude non-production code:
- Test files (`*_test.go`)
- Generated code (protobuf, mocks)
- Vendor dependencies
- Example code
- Small utility files

### 🔧 JavaScript Testing

100% test coverage for all coverage system modules:
```bash
cd .github/coverage
npm test -- --coverage
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

#### 8.3 docs/coverage-system.md Template
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
- Node.js 20+
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

## Conclusion

This comprehensive plan transforms coverage reporting from a third-party dependency into a powerful, integrated feature of the GoFortress CI/CD system. By controlling every aspect of the coverage pipeline, we can deliver a superior developer experience while maintaining complete data sovereignty and zero external dependencies.

The phased approach ensures smooth implementation with minimal disruption, while the modular architecture allows for continuous enhancement and customization. This isn't just a replacement for Codecov—it's an evolution in how we think about code coverage in modern development workflows.

## Implementation Timeline

- **Phase 1-2**: Foundation & Core Engine (2 sessions)
- **Phase 3-4**: Integration & Storage (2 sessions)
- **Phase 5-6**: PR Features & Analytics (2 sessions)
- **Phase 7-8**: Deployment & Documentation (2 sessions)
- **Total**: 8 development sessions

Each phase builds upon the previous, ensuring a stable and tested system at every stage.
