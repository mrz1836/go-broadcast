# GoFortress Coverage API & CLI Documentation

Complete reference for the GoFortress Internal Coverage System CLI tools, API endpoints, and automation capabilities.

## CLI Reference

The `gofortress-coverage` CLI tool provides comprehensive coverage processing, analytics, and automation capabilities.

### Installation & Build

The CLI tool is built from the `.github/coverage/` directory:

```bash
cd .github/coverage
go build -o gofortress-coverage ./cmd/gofortress-coverage/
```

### Global Options

All commands support these global flags:

```bash
gofortress-coverage [command] [flags]

Global Flags:
  --config string     Configuration file path (default: ".github/.env.shared")
  --verbose          Enable verbose logging
  --debug            Enable debug mode with detailed output
  --dry-run          Preview operations without making changes
  --help            Show help information
  --version         Show version information
```

## Core Commands

### `complete` - Full Pipeline Processing

Executes the complete coverage processing pipeline in a single command.

```bash
gofortress-coverage complete [flags]

Flags:
  --input string      Input coverage file (default: "coverage.out")
  --output string     Output directory for reports (default: "coverage-reports")
  --branch string     Git branch name
  --commit string     Git commit SHA
  --pr string         Pull request number
  --threshold float   Coverage threshold percentage (default: from config)
  --fail-under        Fail if coverage is below threshold
  
Examples:
  # Basic usage
  gofortress-coverage complete --input coverage.out
  
  # With branch and commit info
  gofortress-coverage complete \
    --input coverage.out \
    --branch main \
    --commit abc123def \
    --threshold 80
    
  # For pull request
  gofortress-coverage complete \
    --input coverage.out \
    --pr 123 \
    --fail-under
```

#### Pipeline Steps

The `complete` command executes these steps in order:

1. **Parse**: Analyze coverage file and extract metrics
2. **Badge**: Generate SVG badges for the branch/PR
3. **Report**: Create interactive HTML reports
4. **History**: Update historical trend data
5. **Deploy**: Deploy to GitHub Pages
6. **Comment**: Add PR comment (if applicable)
7. **Status**: Update GitHub status checks

### `parse` - Coverage File Analysis

Parses Go coverage files and extracts detailed metrics.

```bash
gofortress-coverage parse [flags]

Flags:
  --file string       Coverage file to parse (default: "coverage.out")
  --output string     Output JSON file (default: stdout)
  --format string     Output format: json, yaml, table (default: "json")
  --exclude strings   Additional exclusion patterns
  --include strings   Include only these patterns (whitelist mode)
  
Examples:
  # Parse and output to stdout
  gofortress-coverage parse --file coverage.out
  
  # Save to file with custom exclusions
  gofortress-coverage parse \
    --file coverage.out \
    --output metrics.json \
    --exclude "test/*,mock/*"
    
  # Human-readable table format
  gofortress-coverage parse \
    --file coverage.out \
    --format table
```

#### Parse Output Format

```json
{
  "overall_coverage": 87.2,
  "total_lines": 1432,
  "covered_lines": 1249,
  "packages": [
    {
      "name": "internal/parser",
      "coverage": 95.8,
      "lines": 1240,
      "covered": 1187,
      "files": 8,
      "functions": 45
    }
  ],
  "files": [
    {
      "name": "internal/parser/parser.go",
      "coverage": 92.5,
      "lines": 324,
      "covered": 300,
      "functions": 12,
      "missing_lines": [45, 78, 123]
    }
  ],
  "quality_score": 92,
  "risk_assessment": "low"
}
```

### `badge` - SVG Badge Generation

Creates professional SVG badges for coverage visualization.

```bash
gofortress-coverage badge [flags]

Flags:
  --coverage float    Coverage percentage (required)
  --output string     Output SVG file (default: stdout)
  --style string      Badge style: flat, flat-square, for-the-badge (default: "flat")
  --label string      Badge label text (default: "coverage")
  --logo string       Logo name or URL (default: "go")
  --logo-color string Logo color (default: "white")
  --trend string      Trend indicator: up, down, stable
  --branch string     Branch name for badge caching
  
Examples:
  # Basic badge
  gofortress-coverage badge --coverage 87.2 --output badge.svg
  
  # Custom style and branding
  gofortress-coverage badge \
    --coverage 92.5 \
    --style for-the-badge \
    --label "quality" \
    --logo github \
    --logo-color blue
    
  # With trend indicator
  gofortress-coverage badge \
    --coverage 89.1 \
    --trend up \
    --output trending-badge.svg
```

#### Badge Types

The system generates multiple badge types:

- **Coverage Badge**: Standard coverage percentage
- **Trend Badge**: Coverage direction (↗ ↘ →)
- **Status Badge**: Quality gate status (PASS/FAIL)
- **Comparison Badge**: PR coverage vs base
- **Quality Badge**: Overall quality score (A+, B, C, etc.)

### `report` - HTML Report Generation

Creates interactive HTML coverage reports.

```bash
gofortress-coverage report [flags]

Flags:
  --data string       JSON data file from parse command (required)
  --output string     Output directory (default: "report")
  --template string   Report template: standard, minimal, detailed (default: "standard")
  --theme string      Color theme: github-dark, github-light (default: "github-dark")
  --title string      Report title (default: "GoFortress Coverage")
  --include-history   Include historical trend charts
  --branch string     Branch name for report organization
  
Examples:
  # Basic report generation
  gofortress-coverage report --data metrics.json
  
  # Custom themed report
  gofortress-coverage report \
    --data metrics.json \
    --output reports/main \
    --theme github-light \
    --title "Main Branch Coverage"
    
  # Detailed report with history
  gofortress-coverage report \
    --data metrics.json \
    --template detailed \
    --include-history \
    --branch main
```

#### Report Features

Generated reports include:

- **Interactive Dashboard**: Real-time metrics and charts
- **Package Breakdown**: Detailed package-level analysis
- **File Coverage Map**: Visual file tree with coverage colors
- **Missing Lines**: Highlighted uncovered code sections
- **Trend Analysis**: Historical coverage patterns
- **Quality Metrics**: Code quality assessments

### `history` - Trend Analysis

Manages historical coverage data and trend analysis.

```bash
gofortress-coverage history [subcommand] [flags]

Subcommands:
  add         Add coverage data point to history
  analyze     Analyze historical trends
  predict     Generate coverage predictions
  export      Export historical data
  clean       Clean old historical data
  
Examples:
  # Add current coverage to history
  gofortress-coverage history add \
    --coverage 87.2 \
    --branch main \
    --commit abc123
    
  # Analyze recent trends
  gofortress-coverage history analyze \
    --branch main \
    --days 30
    
  # Generate predictions
  gofortress-coverage history predict \
    --branch main \
    --horizon 7 \
    --output predictions.json
```

#### History Subcommands

##### `history add`
```bash
Flags:
  --coverage float    Coverage percentage (required)
  --branch string     Branch name (required)
  --commit string     Commit SHA
  --pr string         Pull request number
  --author string     Commit author
  --message string    Commit message
```

##### `history analyze`
```bash
Flags:
  --branch string     Branch to analyze (default: "main")
  --days int          Days of history to analyze (default: 30)
  --output string     Output analysis file
  --format string     Output format: json, yaml, table (default: "json")
```

##### `history predict`
```bash
Flags:
  --branch string     Branch to predict for (default: "main")
  --horizon int       Prediction horizon in days (default: 7)
  --model string      Prediction model: linear, polynomial, seasonal (default: "linear")
  --confidence float  Confidence interval (default: 0.95)
```

### `comment` - PR Comment Management

Manages pull request coverage comments with comprehensive analysis and templates.

```bash
gofortress-coverage comment [flags]

Flags:
  -p, --pr int              Pull request number (defaults to GITHUB_PR_NUMBER)
  -c, --coverage string     Coverage data file
  --base-coverage string    Base coverage data file for comparison
  --badge-url string        Badge URL (auto-generated if not provided)
  --report-url string       Report URL (auto-generated if not provided)
  --status                  Create enhanced status checks
  --block-merge             Block PR merge on coverage failure
  --generate-badges         Generate PR-specific badges
  --enable-analysis         Enable detailed coverage analysis and comparison (default true)
  --anti-spam               Enable anti-spam features (default true)
  --dry-run                 Show preview of comment without posting
  
Examples:
  # Basic coverage comment
  gofortress-coverage comment \
    --pr 123 \
    --coverage coverage.json
    
  # Enhanced comment with analysis
  gofortress-coverage comment \
    --pr 123 \
    --coverage pr-coverage.json \
    --base-coverage main-coverage.json \
    --enable-analysis \
    --generate-badges
    
  # Comment with status checks
  gofortress-coverage comment \
    --pr 123 \
    --coverage coverage.json \
    --status \
    --block-merge
    
  # Dry run preview
  gofortress-coverage comment \
    --pr 123 \
    --coverage coverage.json \
    --dry-run
```

#### Comment Templates

- **Comprehensive**: Full analysis with file details, quality gates, and recommendations
- **Compact**: Single line with coverage and link to detailed report
- **Detailed**: Enhanced analysis with risk assessment and complexity metrics
- **Summary**: Quick overview with key metrics and status
- **Minimal**: Ultra-compact coverage percentage only

## Analytics Commands

### `analytics` - Advanced Analysis

Provides comprehensive analytics and insights.

```bash
gofortress-coverage analytics [subcommand] [flags]

Subcommands:
  dashboard   Generate interactive analytics dashboard
  trends      Analyze coverage trends and patterns
  predictions Generate future coverage predictions
  impact      Analyze PR impact and risk assessment
  team        Generate team analytics and collaboration metrics
  export      Export analytics data in various formats
  chart       Generate standalone charts and visualizations
  notify      Send analytics notifications
```

#### `analytics dashboard`
```bash
gofortress-coverage analytics dashboard [flags]

Flags:
  --output string     Output directory (default: "analytics")
  --branch string     Branch to analyze (default: "main")
  --days int          Days of data to include (default: 90)
  --theme string      Dashboard theme (default: "github-dark")
  --include-team      Include team analytics
  --include-predictions Include prediction models
  
Examples:
  # Generate comprehensive dashboard
  gofortress-coverage analytics dashboard \
    --output dashboard \
    --days 90 \
    --include-team \
    --include-predictions
```

#### `analytics trends`
```bash
gofortress-coverage analytics trends [flags]

Flags:
  --branch string     Branch to analyze (default: "main")
  --days int          Days of history (default: 30)
  --output string     Output file for trend data
  --chart             Generate trend chart
  --forecast int      Forecast days ahead (default: 7)
  
Examples:
  # Analyze main branch trends
  gofortress-coverage analytics trends \
    --branch main \
    --days 60 \
    --chart \
    --forecast 14
```

#### `analytics team`
```bash
gofortress-coverage analytics team [flags]

Flags:
  --output string     Output directory (default: "team-analytics")
  --days int          Days of activity to analyze (default: 30)
  --format string     Output format: html, json, csv (default: "html")
  --include-individual Include individual contributor metrics
  --include-collaboration Include collaboration analysis
  
Examples:
  # Generate team analytics report
  gofortress-coverage analytics team \
    --days 60 \
    --include-individual \
    --include-collaboration
```

## GitHub Pages Commands

### `pages` - GitHub Pages Management

Manages GitHub Pages deployment and organization.

```bash
gofortress-coverage pages [subcommand] [flags]

Subcommands:
  setup       Initialize GitHub Pages branch and structure
  deploy      Deploy coverage reports to GitHub Pages
  clean       Clean up old reports and maintain storage limits
  status      Check GitHub Pages deployment status
  
Examples:
  # Initialize GitHub Pages
  gofortress-coverage pages setup --branch gh-pages
  
  # Deploy current reports
  gofortress-coverage pages deploy \
    --input coverage-reports \
    --branch main
    
  # Clean old reports
  gofortress-coverage pages clean --max-age 30
```

#### `pages setup`
```bash
Flags:
  --branch string     Pages branch name (default: "gh-pages")
  --force            Force recreation if branch exists
  --template string  Initial template: standard, minimal (default: "standard")
```

#### `pages deploy`
```bash
Flags:
  --input string      Input directory with reports (required)
  --branch string     Source branch name (required)
  --commit string     Commit SHA for organization
  --pr string         Pull request number (for PR-specific deployment)
  --message string    Commit message for deployment
```

#### `pages clean`
```bash
Flags:
  --max-age int       Maximum age in days (default: 30)
  --dry-run          Preview cleanup without deleting
  --size-limit string Maximum total size (default: "800MB")
  --keep-branches strings Branches to always keep
```

## Configuration Commands

### `config` - Configuration Management

Validates and manages configuration settings.

```bash
gofortress-coverage config [subcommand] [flags]

Subcommands:
  validate    Validate current configuration
  show        Display current configuration
  init        Initialize default configuration
  migrate     Migrate from external service configuration
  
Examples:
  # Validate configuration
  gofortress-coverage config validate
  
  # Show current settings
  gofortress-coverage config show --format table
  
  # Initialize with defaults
  gofortress-coverage config init --output .github/.env.shared
```

## API Endpoints

When deployed to GitHub Pages, the coverage system exposes several JSON API endpoints:

### Base URL Structure
```
https://{organization}.github.io/{repository}/api/
```

### Available Endpoints

#### `/api/coverage.json` - Latest Coverage Data
```http
GET /api/coverage.json?branch=main
```

**Response:**
```json
{
  "overall_coverage": 87.2,
  "branch": "main",
  "commit": "abc123def456",
  "timestamp": "2025-01-27T10:30:00Z",
  "quality_score": 92,
  "trend": "improving",
  "packages": [
    {
      "name": "internal/parser",
      "coverage": 95.8,
      "lines": 1240,
      "files": 8
    }
  ],
  "thresholds": {
    "minimum": 80,
    "excellent": 90,
    "status": "pass"
  }
}
```

#### `/api/history.json` - Historical Trends
```http
GET /api/history.json?branch=main&days=30
```

**Query Parameters:**
- `branch`: Branch name (default: main)
- `days`: Days of history (default: 30, max: 365)
- `granularity`: Data granularity (daily, weekly, monthly)

**Response:**
```json
{
  "branch": "main",
  "period": {
    "start": "2024-12-28T00:00:00Z",
    "end": "2025-01-27T00:00:00Z",
    "days": 30
  },
  "summary": {
    "current_coverage": 87.2,
    "start_coverage": 85.1,
    "change": 2.1,
    "trend": "improving",
    "volatility": 0.8,
    "momentum": 0.15
  },
  "data_points": [
    {
      "date": "2025-01-27",
      "coverage": 87.2,
      "commit": "abc123",
      "author": "developer@example.com"
    }
  ],
  "predictions": {
    "next_7_days": [
      {"date": "2025-01-28", "coverage": 87.5, "confidence": 0.85}
    ]
  }
}
```

#### `/api/analytics.json` - Advanced Analytics
```http
GET /api/analytics.json?branch=main
```

**Response:**
```json
{
  "branch": "main",
  "generated_at": "2025-01-27T10:30:00Z",
  "quality_assessment": {
    "overall_score": 92,
    "grade": "A",
    "risk_level": "low",
    "recommendations": [
      "Consider adding tests for parseComplexFormat() function"
    ]
  },
  "performance": {
    "large_files": 3,
    "complex_functions": 12,
    "test_coverage_gaps": 5
  },
  "trends": {
    "weekly_change": 1.2,
    "monthly_change": 3.1,
    "stability_score": 0.92
  },
  "team_metrics": {
    "contributors": 8,
    "avg_coverage_impact": 1.5,
    "top_contributors": [
      {"author": "alice@example.com", "impact": 2.3}
    ]
  }
}
```

#### `/api/health.json` - System Status
```http
GET /api/health.json
```

**Response:**
```json
{
  "status": "healthy",
  "last_update": "2025-01-27T10:30:00Z",
  "uptime": "99.95%",
  "metrics": {
    "total_reports": 1247,
    "total_branches": 15,
    "active_prs": 3,
    "storage_usage": "245MB"
  },
  "health_checks": {
    "badges": "ok",
    "reports": "ok",
    "github_pages": "ok",
    "api_endpoints": "ok"
  },
  "performance": {
    "avg_badge_generation": "1.2ms",
    "avg_report_generation": "8.5s",
    "avg_deployment_time": "45s"
  }
}
```

### API Authentication

All API endpoints are publicly accessible when deployed to GitHub Pages. For private repositories, GitHub Pages authentication applies automatically.

### Rate Limiting

API endpoints are served by GitHub Pages CDN with automatic caching:
- Cache TTL: 5 minutes for dynamic data
- Static assets: 24 hours
- No explicit rate limits (GitHub Pages limits apply)

## Automation Examples

### GitHub Actions Integration

#### Basic Workflow Integration
```yaml
name: Coverage Processing
on: [push, pull_request]

jobs:
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
          
      - name: Run Tests with Coverage
        run: go test -coverprofile=coverage.out ./...
        
      - name: Process Coverage
        run: |
          cd .github/coverage
          go run ./cmd/gofortress-coverage complete \
            --input ../../coverage.out \
            --branch ${{ github.ref_name }} \
            --commit ${{ github.sha }}
```

#### Advanced Workflow with PR Integration
```yaml
- name: Coverage Analysis
  env:
    ENABLE_INTERNAL_COVERAGE: true
    COVERAGE_FAIL_UNDER: 80
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: |
    cd .github/coverage
    go run ./cmd/gofortress-coverage complete \
      --input ../../coverage.out \
      --branch ${{ github.ref_name }} \
      --commit ${{ github.sha }} \
      --pr ${{ github.event.number }} \
      --fail-under
```

### Custom Automation Scripts

#### Bash Script for Local Development
```bash
#!/bin/bash
# local-coverage.sh - Run coverage analysis locally

set -e

echo "Running tests with coverage..."
go test -coverprofile=coverage.out ./...

echo "Processing coverage..."
cd .github/coverage

echo "Generating reports..."
go run ./cmd/gofortress-coverage complete \
  --input ../../coverage.out \
  --output ../../coverage-reports \
  --verbose

echo "Coverage reports generated in coverage-reports/"
echo "Open coverage-reports/index.html to view dashboard"
```

#### Python Integration Script
```python
#!/usr/bin/env python3
# coverage-integration.py - Custom integration example

import subprocess
import json
import sys

def run_coverage_analysis():
    """Run Go tests and process coverage"""
    # Run tests
    subprocess.run(["go", "test", "-coverprofile=coverage.out", "./..."], check=True)
    
    # Process with GoFortress
    result = subprocess.run([
        "go", "run", "./.github/coverage/cmd/gofortress-coverage",
        "parse", "--file", "coverage.out", "--format", "json"
    ], capture_output=True, text=True, check=True)
    
    # Parse coverage data
    coverage_data = json.loads(result.stdout)
    
    # Custom processing
    if coverage_data["overall_coverage"] < 80:
        print(f"❌ Coverage {coverage_data['overall_coverage']:.1f}% below threshold")
        sys.exit(1)
    else:
        print(f"✅ Coverage {coverage_data['overall_coverage']:.1f}% meets requirements")

if __name__ == "__main__":
    run_coverage_analysis()
```

### CI/CD Integration Patterns

#### GitLab CI Integration
```yaml
coverage:
  stage: test
  script:
    - go test -coverprofile=coverage.out ./...
    - cd .github/coverage
    - go run ./cmd/gofortress-coverage complete --input ../../coverage.out
  artifacts:
    reports:
      coverage_report:
        coverage_format: cobertura
        path: coverage-reports/cobertura.xml
```

#### Jenkins Pipeline
```groovy
pipeline {
    agent any
    stages {
        stage('Test') {
            steps {
                sh 'go test -coverprofile=coverage.out ./...'
            }
        }
        stage('Coverage') {
            steps {
                dir('.github/coverage') {
                    sh '''
                        go run ./cmd/gofortress-coverage complete \
                            --input ../../coverage.out \
                            --output ../../coverage-reports
                    '''
                }
                publishHTML([
                    allowMissing: false,
                    alwaysLinkToLastBuild: true,
                    keepAll: true,
                    reportDir: 'coverage-reports',
                    reportFiles: 'index.html',
                    reportName: 'Coverage Report'
                ])
            }
        }
    }
}
```

## Error Handling & Debugging

### Common Exit Codes

- `0`: Success
- `1`: General error
- `2`: Configuration error
- `3`: Coverage below threshold (when --fail-under is used)
- `4`: GitHub API error
- `5`: File system error
- `6`: Parse error

### Debug Mode

Enable detailed debugging with:

```bash
gofortress-coverage --debug [command]
```

Debug mode provides:
- Detailed operation logs
- Performance timing
- Memory usage tracking
- Configuration validation
- API request/response details

### Verbose Logging

```bash
gofortress-coverage --log-level debug [command]
```

Verbose mode includes:
- Progress indicators
- Step-by-step operations
- Warning messages
- Performance metrics

### Troubleshooting

#### Common Issues

1. **Coverage file not found**
   ```bash
   Error: coverage file 'coverage.out' not found
   Solution: Ensure tests generate coverage.out file
   ```

2. **GitHub API rate limits**
   ```bash
   Error: GitHub API rate limit exceeded
   Solution: Wait or use authenticated requests
   ```

3. **GitHub Pages deployment failure**
   ```bash
   Error: failed to push to gh-pages branch
   Solution: Check repository permissions and branch existence
   ```

#### Debug Commands

```bash
# Validate configuration
gofortress-coverage config validate --log-level debug

# Test GitHub API connectivity
gofortress-coverage --debug comment --pr 123 --dry-run

# Verify file parsing
gofortress-coverage parse --file coverage.out --log-level debug
```

---

## Related Documentation

- [📖 System Overview](coverage-system.md) - Architecture and components
- [🎯 Feature Showcase](coverage-features.md) - Detailed feature examples  
- [⚙️ Configuration Guide](coverage-configuration.md) - Complete configuration reference

## Support & Contributing

- **Issues**: Report bugs and request features via GitHub Issues
- **Documentation**: Complete guides available in `/docs/` directory
- **Contributing**: See [CONTRIBUTING.md](../CONTRIBUTING.md) for development guidelines
- **CLI Help**: Use `gofortress-coverage --help` for command-specific help