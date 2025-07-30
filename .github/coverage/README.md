# GoFortress Coverage System

A self-contained, Go-native coverage system built as a complete bolt-on solution for any Go repository.

## Architecture

This coverage system is designed as a portable, self-contained unit that lives entirely within the `.github/coverage/` directory. It can be copied to any repository without polluting the main codebase structure.

### Directory Structure

```
.github/coverage/
├── cmd/
│   └── gofortress-coverage/        # Main CLI tool
│       ├── main.go                 # Entry point
│       ├── go.mod                  # Separate Go module
│       └── cmd/                    # Command implementations
├── internal/                       # Internal packages
│   ├── parser/                     # Coverage parsing logic
│   ├── badge/                      # SVG badge generation
│   ├── report/                     # HTML report generation
│   ├── history/                    # Historical data tracking
│   ├── github/                     # GitHub API integration
│   └── templates/                  # HTML templates (source files)
│       ├── coverage-report.html    # Template for detailed coverage reports
│       ├── dashboard.html          # Template for coverage dashboard
│       └── favicon.ico             # Favicon for web interface
├── scripts/                        # Setup and maintenance scripts
│   ├── setup-github-pages-env.sh   # GitHub Pages environment configuration
│   └── README.md                   # Scripts documentation
├── history/                        # Coverage history tracking (generated)
└── README.md                       # This file

# Generated directories (not in repository)
coverage-output/                    # Generated coverage reports
test-coverage-output/               # Generated test coverage reports
```

**Note**: The `coverage-output/` and `test-coverage-output/` directories are generated during the build process and are excluded from version control via `.gitignore`. Only the template source files in `internal/templates/` are stored in the repository.

## Features

- **Go-Native**: Single binary with no runtime dependencies
- **Bolt-On Architecture**: Complete encapsulation within `.github/coverage/`
- **Portable**: Can be copied to any repository as a complete unit
- **Zero External Dependencies**: No reliance on external services
- **Professional Quality**: GitHub-style badges and clean reports
- **CI/CD Integration**: Seamless integration with GitHub Actions
- **Enhanced PR Comments**: Beautiful, spam-free PR coverage comments with intelligent updates
- **Anti-Spam Protection**: Only one comment per PR with smart update logic
- **Multiple Templates**: Comprehensive, detailed, compact, summary, and minimal styles
- **Coverage Comparison**: Visual diff between base and PR branch coverage

## Implementation Status

- **Phase 1**: ✅ Foundation & Configuration (Current)
- **Phase 2**: ⏳ Core Coverage Engine
- **Phase 3**: ⏳ Fortress Workflow Integration
- **Phase 4**: ⏳ GitHub Pages & Storage
- **Phase 5**: ⏳ Pull Request Integration

## Setup

### New Repository Setup

When setting up the GoFortress coverage system for a new repository, you'll need to configure GitHub Pages environment settings:

```bash
# From repository root
./.github/coverage/scripts/setup-github-pages-env.sh
```

This script configures the necessary environment protection rules for GitHub Pages deployment. See [`scripts/README.md`](scripts/README.md) for detailed documentation.

### Requirements

- GitHub CLI (`gh`) installed and authenticated
- Repository admin permissions
- GitHub Pages enabled in repository settings

## Usage

The CLI tool will be built and used in GitHub Actions workflows:

```bash
# Parse coverage data
./gofortress-coverage parse --file coverage.out --output coverage.json

# Generate badge
./gofortress-coverage badge --coverage 85.5 --output badge.svg

# Generate report
./gofortress-coverage report --data coverage.json --output report.html

# Update history
./gofortress-coverage history --add coverage.json --branch main --commit abc123

# Create basic PR comment (compact template)
./gofortress-coverage comment --pr 123 --coverage coverage.json --template compact

# Create enhanced PR comment with anti-spam and templates
./gofortress-coverage comment --pr 123 --coverage coverage.out \
  --template comprehensive \
  --badge-url "https://owner.github.io/repo/coverage/badge.svg" \
  --report-url "https://owner.github.io/repo/coverage/"
```

## Configuration

The system is configured through environment variables in `.github/.env.shared`. See the main repository documentation for complete configuration options.

## Template System

The coverage system uses two main HTML templates stored in `internal/templates/`:

1. **`coverage-report.html`** - Generates detailed coverage reports showing line-by-line coverage
2. **`dashboard.html`** - Creates an interactive dashboard with coverage trends and metrics

During the coverage generation process:
- Templates are processed with coverage data
- Output files are generated in `coverage-output/` directory
- For GitHub Pages deployment, `dashboard.html` is also copied as `index.html` for convenience

All generated HTML files are excluded from version control and created fresh during each build.

## Enhanced PR Comments

The enhanced PR comment feature provides beautiful, informative coverage comments with anti-spam protection:

### Features
- **Single Comment Per PR**: Automatically updates existing comments instead of creating new ones
- **Anti-Spam Protection**: 
  - Maximum 1 comment per PR
  - Minimum 5-minute update interval
  - Only updates on significant changes (>1% coverage change)
- **Multiple Templates**:
  - `comprehensive`: Full details with badges, metrics, and trends
  - `detailed`: Deep analysis with file-level breakdowns
  - `compact`: Clean, minimal design
  - `summary`: High-level overview
  - `minimal`: Just the essentials
- **Beautiful Design**: Matches GitHub's style with emojis, progress bars, and tables
- **Coverage Comparison**: Shows diff between base branch and PR branch
- **Inline Badge**: Displays coverage badge directly in the comment
- **Direct Links**: Quick access to full reports and PR-specific coverage

### Usage Examples

```bash
# Basic usage (auto-detects PR context from environment)
./gofortress-coverage comment --coverage coverage.out

# With specific template
./gofortress-coverage comment --pr 123 --coverage coverage.out --template detailed

# With base comparison
./gofortress-coverage comment --pr 123 \
  --coverage pr-coverage.out \
  --base-coverage main-coverage.out \
  --template comprehensive

# Dry run to preview comment
./gofortress-coverage comment --pr 123 --coverage coverage.out --dry-run
```

### Template Examples

The comprehensive template includes:
- Overall coverage percentage with trend emoji
- Coverage metrics table with grades
- Progress bar visualization
- File-level changes (significant files only)
- Quality assessment with strengths/weaknesses
- Actionable recommendations
- Resource links with inline badge

## Development

This is a separate Go module with its own dependencies. To work on the coverage tool:

```bash
cd .github/coverage/cmd/gofortress-coverage
go mod tidy
go build -o gofortress-coverage
go test ./...
```

## License

This coverage system inherits the license from the parent repository.