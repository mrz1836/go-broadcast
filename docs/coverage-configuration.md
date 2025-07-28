# GoFortress Coverage Configuration Reference

Complete configuration guide for the GoFortress Internal Coverage System with all 45+ environment variables, examples, and best practices.

## Configuration Overview

The GoFortress Coverage System uses environment variables defined in [`.github/.env.shared`](../.github/.env.shared) for all configuration. This approach provides:

- **Version Control**: All settings tracked in git
- **Environment Isolation**: Different settings per deployment
- **Zero Secrets**: No sensitive data in configuration
- **CI/CD Integration**: Seamless workflow integration

## Quick Start Configuration

### Minimal Setup
Add these variables to `.github/.env.shared` to get started:

```bash
# Enable the internal coverage system
ENABLE_INTERNAL_COVERAGE=true

# Basic thresholds
COVERAGE_FAIL_UNDER=80
COVERAGE_THRESHOLD_EXCELLENT=90

# Enable core features
COVERAGE_PR_COMMENT_ENABLED=true
COVERAGE_PAGES_AUTO_CREATE=true
```

### Recommended Setup
For production use, add these additional settings:

```bash
# Badge configuration
COVERAGE_BADGE_STYLE=flat
COVERAGE_BADGE_LOGO=go
COVERAGE_BADGE_BRANCHES=master,development

# Analytics and trends
COVERAGE_ENABLE_TREND_ANALYSIS=true
COVERAGE_ENABLE_PACKAGE_BREAKDOWN=true
COVERAGE_HISTORY_RETENTION_DAYS=90

# Cleanup and maintenance
COVERAGE_CLEANUP_PR_AFTER_DAYS=7
COVERAGE_PAGES_BRANCH=gh-pages
```

## Core Configuration

### System Enable/Disable

#### `ENABLE_INTERNAL_COVERAGE`
**Type**: Boolean  
**Default**: `false`  
**Description**: Master switch to enable the internal coverage system

```bash
ENABLE_INTERNAL_COVERAGE=true   # Enable coverage system
ENABLE_INTERNAL_COVERAGE=false  # Use external service (Codecov, etc.)
```

#### `ENABLE_INTERNAL_COVERAGE_TESTS`
**Type**: Boolean  
**Default**: `true`  
**Description**: Run coverage tool tests in CI

```bash
ENABLE_INTERNAL_COVERAGE_TESTS=true   # Test coverage tool
ENABLE_INTERNAL_COVERAGE_TESTS=false  # Skip coverage tool tests
```

## Coverage Thresholds

### Quality Gates

#### `COVERAGE_FAIL_UNDER`
**Type**: Float (0-100)  
**Default**: `70`  
**Description**: Minimum acceptable coverage percentage

```bash
COVERAGE_FAIL_UNDER=80    # Fail builds below 80%
COVERAGE_FAIL_UNDER=70    # More lenient threshold
COVERAGE_FAIL_UNDER=90    # Strict quality requirements
```

#### `COVERAGE_ENFORCE_THRESHOLD`
**Type**: Boolean  
**Default**: `false`  
**Description**: Whether to fail builds below threshold

```bash
COVERAGE_ENFORCE_THRESHOLD=true   # Fail builds on low coverage
COVERAGE_ENFORCE_THRESHOLD=false  # Warning only
```

### Badge Color Thresholds

#### `COVERAGE_THRESHOLD_EXCELLENT`
**Type**: Float (0-100)  
**Default**: `90`  
**Description**: Coverage percentage for bright green badges

#### `COVERAGE_THRESHOLD_GOOD`
**Type**: Float (0-100)  
**Default**: `80`  
**Description**: Coverage percentage for green badges

#### `COVERAGE_THRESHOLD_ACCEPTABLE`
**Type**: Float (0-100)  
**Default**: `70`  
**Description**: Coverage percentage for yellow badges

#### `COVERAGE_THRESHOLD_LOW`
**Type**: Float (0-100)  
**Default**: `60`  
**Description**: Coverage percentage for orange badges (below = red)

```bash
# Custom color thresholds
COVERAGE_THRESHOLD_EXCELLENT=95  # Bright green at 95%+
COVERAGE_THRESHOLD_GOOD=85       # Green at 85-94%
COVERAGE_THRESHOLD_ACCEPTABLE=75 # Yellow at 75-84%
COVERAGE_THRESHOLD_LOW=65        # Orange at 65-74%, red below
```

## Badge Configuration

### Badge Style and Appearance

#### `COVERAGE_BADGE_STYLE`
**Type**: String  
**Default**: `flat`  
**Options**: `flat`, `flat-square`, `for-the-badge`  
**Description**: Visual style of coverage badges

```bash
COVERAGE_BADGE_STYLE=flat          # Standard GitHub style
COVERAGE_BADGE_STYLE=flat-square   # Square corners
COVERAGE_BADGE_STYLE=for-the-badge # Large, prominent style
```

#### `COVERAGE_BADGE_LABEL`
**Type**: String  
**Default**: `coverage`  
**Description**: Left-side text of the badge

```bash
COVERAGE_BADGE_LABEL=coverage     # Default label
COVERAGE_BADGE_LABEL=tests        # Custom label
COVERAGE_BADGE_LABEL=quality      # Alternative text
```

#### `COVERAGE_BADGE_LOGO`
**Type**: String  
**Default**: `go`  
**Options**: `go`, `github`, custom URL  
**Description**: Logo displayed on badge

```bash
COVERAGE_BADGE_LOGO=go                                    # Go gopher logo
COVERAGE_BADGE_LOGO=github                               # GitHub logo
COVERAGE_BADGE_LOGO=https://example.com/custom-logo.svg # Custom logo
```

#### `COVERAGE_BADGE_LOGO_COLOR`
**Type**: String  
**Default**: `white`  
**Description**: Color of the logo on the badge

```bash
COVERAGE_BADGE_LOGO_COLOR=white   # White logo
COVERAGE_BADGE_LOGO_COLOR=blue    # Blue logo
COVERAGE_BADGE_LOGO_COLOR=#ff0000 # Custom hex color
```

### Badge Generation

#### `COVERAGE_BADGE_BRANCHES`
**Type**: Comma-separated list  
**Default**: `master,development`  
**Description**: Branches to generate badges for

```bash
COVERAGE_BADGE_BRANCHES=master,development           # Standard branches
COVERAGE_BADGE_BRANCHES=master,staging,prod    # Custom branches
COVERAGE_BADGE_BRANCHES=master                   # Single branch only
```

## GitHub Pages Configuration

### Pages Setup

#### `COVERAGE_PAGES_BRANCH`
**Type**: String  
**Default**: `gh-pages`  
**Description**: GitHub Pages branch name

```bash
COVERAGE_PAGES_BRANCH=gh-pages    # Standard Pages branch
COVERAGE_PAGES_BRANCH=docs        # Custom Pages branch
```

#### `COVERAGE_PAGES_AUTO_CREATE`
**Type**: Boolean  
**Default**: `true`  
**Description**: Automatically create GitHub Pages branch if missing

```bash
COVERAGE_PAGES_AUTO_CREATE=true   # Auto-create branch
COVERAGE_PAGES_AUTO_CREATE=false  # Manual setup required
```

## Report Configuration

### Report Generation

#### `COVERAGE_REPORT_TITLE`
**Type**: String  
**Default**: `GoFortress Coverage`  
**Description**: Title displayed in HTML reports

```bash
COVERAGE_REPORT_TITLE="GoFortress Coverage"     # Default title
COVERAGE_REPORT_TITLE="My Project Coverage"    # Custom title
COVERAGE_REPORT_TITLE="Quality Dashboard"      # Alternative title
```

#### `COVERAGE_REPORT_THEME`
**Type**: String  
**Default**: `github-dark`  
**Options**: `github-dark`, `github-light`, `custom`  
**Description**: Visual theme for reports and dashboard

```bash
COVERAGE_REPORT_THEME=github-dark   # Dark theme (default)
COVERAGE_REPORT_THEME=github-light  # Light theme
COVERAGE_REPORT_THEME=custom        # Custom theme (requires CSS)
```

## Pull Request Integration

### PR Comments

#### `COVERAGE_PR_COMMENT_ENABLED`
**Type**: Boolean  
**Default**: `true`  
**Description**: Enable automatic PR coverage comments

```bash
COVERAGE_PR_COMMENT_ENABLED=true   # Enable PR comments
COVERAGE_PR_COMMENT_ENABLED=false  # Disable PR comments
```

#### `COVERAGE_PR_COMMENT_BEHAVIOR`
**Type**: String  
**Default**: `update`  
**Options**: `new`, `update`, `delete-and-new`  
**Description**: How to handle multiple PR updates

```bash
COVERAGE_PR_COMMENT_BEHAVIOR=update        # Update existing comment (recommended)
COVERAGE_PR_COMMENT_BEHAVIOR=new           # Create new comment each time
COVERAGE_PR_COMMENT_BEHAVIOR=delete-and-new # Delete old, create new
```

#### `COVERAGE_PR_COMMENT_SHOW_TREE`
**Type**: Boolean  
**Default**: `true`  
**Description**: Show file tree in PR comments

#### `COVERAGE_PR_COMMENT_SHOW_MISSING`
**Type**: Boolean  
**Default**: `true`  
**Description**: Highlight uncovered lines in PR comments

```bash
COVERAGE_PR_COMMENT_SHOW_TREE=true      # Show file tree
COVERAGE_PR_COMMENT_SHOW_MISSING=true   # Highlight missing coverage
```

## Analytics and History

### Trend Analysis

#### `COVERAGE_ENABLE_TREND_ANALYSIS`
**Type**: Boolean  
**Default**: `true`  
**Description**: Enable historical trend tracking and predictions

#### `COVERAGE_ENABLE_PACKAGE_BREAKDOWN`
**Type**: Boolean  
**Default**: `true`  
**Description**: Show package-level coverage analysis

#### `COVERAGE_ENABLE_COMPLEXITY_ANALYSIS`
**Type**: Boolean  
**Default**: `false`  
**Description**: Analyze code complexity (future feature)

```bash
COVERAGE_ENABLE_TREND_ANALYSIS=true      # Enable trends
COVERAGE_ENABLE_PACKAGE_BREAKDOWN=true   # Package details
COVERAGE_ENABLE_COMPLEXITY_ANALYSIS=false # Complexity (planned)
```

### Data Retention

#### `COVERAGE_HISTORY_RETENTION_DAYS`
**Type**: Integer  
**Default**: `90`  
**Description**: Days to retain historical coverage data

#### `COVERAGE_CLEANUP_PR_AFTER_DAYS`
**Type**: Integer  
**Default**: `7`  
**Description**: Clean up PR coverage data after merge/close

```bash
COVERAGE_HISTORY_RETENTION_DAYS=90   # Keep 3 months of history
COVERAGE_CLEANUP_PR_AFTER_DAYS=7     # Clean PR data after 1 week
```

## Notification Configuration

### Slack Integration

#### `COVERAGE_SLACK_WEBHOOK_ENABLED`
**Type**: Boolean  
**Default**: `false`  
**Description**: Enable Slack notifications

#### `COVERAGE_SLACK_WEBHOOK_URL`
**Type**: String (Secret)  
**Default**: `""`  
**Description**: Slack webhook URL (store in GitHub Secrets)

```bash
COVERAGE_SLACK_WEBHOOK_ENABLED=true                     # Enable Slack
COVERAGE_SLACK_WEBHOOK_URL=${{ secrets.SLACK_WEBHOOK }} # Use secret
```

**Note**: Store the actual webhook URL in GitHub Secrets, not in the configuration file.

## Exclusion Configuration

### Path Exclusions

#### `COVERAGE_EXCLUDE_PATHS`
**Type**: Comma-separated list  
**Default**: `test/,vendor/,examples/,third_party/,testdata/`  
**Description**: Directory paths to exclude from coverage

```bash
# Default exclusions
COVERAGE_EXCLUDE_PATHS=test/,vendor/,examples/,third_party/,testdata/

# Custom exclusions
COVERAGE_EXCLUDE_PATHS=vendor/,docs/,scripts/,proto/

# Minimal exclusions
COVERAGE_EXCLUDE_PATHS=vendor/
```

#### `COVERAGE_EXCLUDE_FILES`
**Type**: Comma-separated list  
**Default**: `*_test.go,*.pb.go,*_mock.go,mock_*.go`  
**Description**: File patterns to exclude from coverage

```bash
# Default exclusions
COVERAGE_EXCLUDE_FILES=*_test.go,*.pb.go,*_mock.go,mock_*.go

# Custom exclusions
COVERAGE_EXCLUDE_FILES=*_test.go,*.generated.go,*_pb.go

# Add more patterns
COVERAGE_EXCLUDE_FILES=*_test.go,*.pb.go,*_mock.go,mock_*.go,*_gen.go
```

#### `COVERAGE_EXCLUDE_PACKAGES`
**Type**: Comma-separated list  
**Default**: `""`  
**Description**: Additional packages to exclude

```bash
COVERAGE_EXCLUDE_PACKAGES=internal/generated,pkg/proto
```

### Advanced Exclusions

#### `COVERAGE_INCLUDE_ONLY_PATHS`
**Type**: Comma-separated list  
**Default**: `""`  
**Description**: If set, only include these paths (whitelist mode)

```bash
COVERAGE_INCLUDE_ONLY_PATHS=internal/,pkg/    # Only include these paths
COVERAGE_INCLUDE_ONLY_PATHS=""                # Include all (default)
```

#### `COVERAGE_EXCLUDE_GENERATED`
**Type**: Boolean  
**Default**: `true`  
**Description**: Exclude generated files (detected by header comments)

#### `COVERAGE_EXCLUDE_TEST_FILES`
**Type**: Boolean  
**Default**: `true`  
**Description**: Exclude test files from coverage analysis

#### `COVERAGE_MIN_FILE_LINES`
**Type**: Integer  
**Default**: `10`  
**Description**: Minimum lines in file to include in coverage

```bash
COVERAGE_EXCLUDE_GENERATED=true    # Skip generated files
COVERAGE_EXCLUDE_TEST_FILES=true   # Skip test files
COVERAGE_MIN_FILE_LINES=10         # Skip tiny files
```

## Logging and Debugging

### Log Configuration

#### `COVERAGE_LOG_LEVEL`
**Type**: String  
**Default**: `info`  
**Options**: `debug`, `info`, `warn`, `error`  
**Description**: Logging verbosity level

#### `COVERAGE_LOG_FORMAT`
**Type**: String  
**Default**: `json`  
**Options**: `json`, `text`, `pretty`  
**Description**: Log output format

```bash
COVERAGE_LOG_LEVEL=debug     # Verbose logging
COVERAGE_LOG_FORMAT=pretty   # Human-readable format
```

#### `COVERAGE_LOG_FILE`
**Type**: String  
**Default**: `/tmp/coverage.log`  
**Description**: Log file path

#### `COVERAGE_LOG_MAX_SIZE`
**Type**: String  
**Default**: `10MB`  
**Description**: Maximum log file size before rotation

#### `COVERAGE_LOG_RETENTION_DAYS`
**Type**: Integer  
**Default**: `7`  
**Description**: Days to retain log files

```bash
COVERAGE_LOG_FILE=/var/log/coverage.log   # Custom log location
COVERAGE_LOG_MAX_SIZE=50MB                # Larger log files
COVERAGE_LOG_RETENTION_DAYS=30            # Keep logs longer
```

### Debug Features

#### `COVERAGE_DEBUG_MODE`
**Type**: Boolean  
**Default**: `false`  
**Description**: Enable verbose debugging output

#### `COVERAGE_TRACE_ERRORS`
**Type**: Boolean  
**Default**: `true`  
**Description**: Include stack traces in error logs

#### `COVERAGE_LOG_PERFORMANCE`
**Type**: Boolean  
**Default**: `true`  
**Description**: Log timing and performance metrics

#### `COVERAGE_LOG_MEMORY_USAGE`
**Type**: Boolean  
**Default**: `true`  
**Description**: Log memory consumption statistics

```bash
COVERAGE_DEBUG_MODE=true           # Enable debug mode
COVERAGE_TRACE_ERRORS=true         # Include stack traces
COVERAGE_LOG_PERFORMANCE=true      # Performance metrics
COVERAGE_LOG_MEMORY_USAGE=true     # Memory tracking
```

## Monitoring and Metrics

### Metrics Collection

#### `COVERAGE_METRICS_ENABLED`
**Type**: Boolean  
**Default**: `true`  
**Description**: Enable metrics collection and reporting

#### `COVERAGE_METRICS_ENDPOINT`
**Type**: String  
**Default**: `""`  
**Description**: Optional external metrics endpoint

#### `COVERAGE_METRICS_INCLUDE_ERRORS`
**Type**: Boolean  
**Default**: `true`  
**Description**: Track error metrics and rates

#### `COVERAGE_METRICS_INCLUDE_PERFORMANCE`
**Type**: Boolean  
**Default**: `true`  
**Description**: Track performance and timing metrics

#### `COVERAGE_METRICS_INCLUDE_USAGE`
**Type**: Boolean  
**Default**: `true`  
**Description**: Track feature usage and adoption

```bash
COVERAGE_METRICS_ENABLED=true               # Enable metrics
COVERAGE_METRICS_ENDPOINT=https://metrics.example.com # Optional endpoint
COVERAGE_METRICS_INCLUDE_ERRORS=true        # Error tracking
COVERAGE_METRICS_INCLUDE_PERFORMANCE=true   # Performance tracking
COVERAGE_METRICS_INCLUDE_USAGE=true         # Usage analytics
```

## Testing and Error Injection

### Test Mode Configuration

#### `COVERAGE_TEST_MODE`
**Type**: Boolean  
**Default**: `false`  
**Description**: Enable test mode for development

#### `COVERAGE_INJECT_ERRORS`
**Type**: Comma-separated list  
**Default**: `""`  
**Options**: `parser`, `api`, `storage`  
**Description**: Components to inject errors in (testing only)

#### `COVERAGE_ERROR_RATE`
**Type**: Float (0-1)  
**Default**: `0`  
**Description**: Error injection rate for testing

```bash
# Development/testing only
COVERAGE_TEST_MODE=true
COVERAGE_INJECT_ERRORS=parser,api
COVERAGE_ERROR_RATE=0.1  # 10% error rate
```

**Warning**: Only use error injection in development environments.

## Environment-Specific Examples

### Development Environment
```bash
# Development configuration
ENABLE_INTERNAL_COVERAGE=true
COVERAGE_FAIL_UNDER=70
COVERAGE_DEBUG_MODE=true
COVERAGE_LOG_LEVEL=debug
COVERAGE_PR_COMMENT_ENABLED=true
COVERAGE_PAGES_AUTO_CREATE=true
COVERAGE_CLEANUP_PR_AFTER_DAYS=1
```

### Staging Environment
```bash
# Staging configuration
ENABLE_INTERNAL_COVERAGE=true
COVERAGE_FAIL_UNDER=80
COVERAGE_ENFORCE_THRESHOLD=false
COVERAGE_LOG_LEVEL=info
COVERAGE_ENABLE_TREND_ANALYSIS=true
COVERAGE_HISTORY_RETENTION_DAYS=30
```

### Production Environment
```bash
# Production configuration
ENABLE_INTERNAL_COVERAGE=true
COVERAGE_FAIL_UNDER=85
COVERAGE_ENFORCE_THRESHOLD=true
COVERAGE_LOG_LEVEL=warn
COVERAGE_ENABLE_TREND_ANALYSIS=true
COVERAGE_ENABLE_PACKAGE_BREAKDOWN=true
COVERAGE_HISTORY_RETENTION_DAYS=90
COVERAGE_SLACK_WEBHOOK_ENABLED=true
COVERAGE_METRICS_ENABLED=true
```

## Configuration Validation

### Common Issues

#### Missing Required Variables
```bash
# ❌ Minimal setup - may cause issues
ENABLE_INTERNAL_COVERAGE=true

# ✅ Recommended minimum
ENABLE_INTERNAL_COVERAGE=true
COVERAGE_FAIL_UNDER=80
COVERAGE_PR_COMMENT_ENABLED=true
```

#### Invalid Values
```bash
# ❌ Invalid threshold
COVERAGE_FAIL_UNDER=150  # Must be 0-100

# ❌ Invalid style
COVERAGE_BADGE_STYLE=invalid  # Must be flat, flat-square, or for-the-badge

# ❌ Invalid log level
COVERAGE_LOG_LEVEL=verbose  # Must be debug, info, warn, or error
```

### Validation Tools
```bash
# Validate configuration (when CLI is available)
gofortress-coverage validate --config

# Check environment variables
env | grep COVERAGE_ | sort
```

## Best Practices

### Performance Optimization
```bash
# Optimize for speed
COVERAGE_LOG_LEVEL=warn              # Reduce logging overhead
COVERAGE_DEBUG_MODE=false            # Disable debug mode
COVERAGE_LOG_PERFORMANCE=false       # Skip performance logging in production
```

### Security Considerations
```bash
# Security best practices
COVERAGE_DEBUG_MODE=false            # Disable debug in production
COVERAGE_TRACE_ERRORS=false          # Reduce information disclosure
COVERAGE_TEST_MODE=false             # Disable test mode
COVERAGE_INJECT_ERRORS=""            # No error injection
```

### Maintenance
```bash
# Balanced retention
COVERAGE_HISTORY_RETENTION_DAYS=90   # 3 months of history
COVERAGE_CLEANUP_PR_AFTER_DAYS=7     # Clean PR data weekly
COVERAGE_LOG_RETENTION_DAYS=14       # 2 weeks of logs
```

---

## Related Documentation

- [📖 System Overview](coverage-system.md) - Architecture and components
- [🎯 Feature Showcase](coverage-features.md) - Detailed feature examples
- [🔧 Migration Guide](migrating-from-codecov.md) - Migrate from external services
- [🛠️ API Reference](coverage-api.md) - CLI commands and automation
