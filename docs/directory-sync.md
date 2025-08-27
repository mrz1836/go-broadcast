# Directory Sync Guide

Complete guide to directory synchronization in go-broadcast, covering concepts, configuration, performance optimization, and real-world usage patterns.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Configuration Reference](#configuration-reference)
- [Exclusion Patterns](#exclusion-patterns)
- [Performance Characteristics](#performance-characteristics)
- [Real-World Use Cases](#real-world-use-cases)
- [Integration with File Sync](#integration-with-file-sync)
- [Troubleshooting](#troubleshooting)
- [Advanced Topics](#advanced-topics)

## Overview

go-broadcast supports synchronizing entire directories alongside individual files, providing powerful capabilities for repository management at scale. Directory sync maintains go-broadcast's core principles:

- **Stateless operation** - Directory contents resolved dynamically at sync time
- **Smart exclusions** - Automatic filtering of development artifacts with custom patterns
- **High performance** - Concurrent processing with API optimization (90%+ call reduction)
- **Complete audit trail** - All directory operations tracked in Git history
- **Transform support** - All transformations work on directory files

### Key Benefits

✅ **Reduced Configuration** - Sync hundreds of files with single directory mapping
✅ **Smart Defaults** - Automatic exclusion of `*.out`, `*.test`, binaries, and temp files
✅ **Exceptional Performance** - 1000+ files processed in ~32ms
✅ **API Efficient** - GitHub tree API reduces calls by 90%+
✅ **Production Ready** - Battle-tested with real .github directories (149 files)

## Quick Start

### Basic Directory Sync

```yaml
version: 1
groups:
  - name: "Workflow Templates"
    id: "workflow-sync"
    priority: 1
    enabled: true
    source:
      repo: "company/templates"
      branch: "master"
    targets:
      - repo: "company/service"
        directories:
          - src: ".github/workflows"
            dest: ".github/workflows"
            # Smart defaults automatically exclude: *.out, *.test, *.exe, **/.DS_Store, **/tmp/*, **/.git
```

### Directory Sync with Custom Exclusions

```yaml
groups:
  - name: "Coverage Configuration"
    id: "coverage-sync"
    priority: 1
    enabled: true
    source:
      repo: "company/templates"
      branch: "master"
    targets:
      - repo: "company/service"
        directories:
          - src: ".github/actions"
            dest: ".github/actions"
            exclude:
              - "*.out"                    # Coverage output files
              - "go-coverage"              # Binary executable
              - "**/testdata/**"           # Test fixtures
            transform:
              repo_name: true
```

### Mixed Files and Directories

```yaml
groups:
  - name: "Service Configuration"
    id: "service-config"
    priority: 1
    enabled: true
    source:
      repo: "company/templates"
      branch: "master"
    targets:
      - repo: "company/service"
        files:
          - src: "Makefile"
            dest: "Makefile"
        directories:
          - src: "configs"
            dest: "configs"
            exclude: ["*.local", "*.secret"]
        transform:
          variables:
            SERVICE_NAME: "my-service"
```

## Core Concepts

### Directory Mapping

Directory mapping defines the relationship between source and destination directories:

```yaml
groups:
  - name: "Directory Sync Group"
    id: "dir-sync"
    source:
      repo: "company/templates"
    targets:
      - repo: "company/service"
        directories:
          - src: "source/path"              # Source directory in template repo
            dest: "destination/path"        # Destination directory in target repo
            exclude: ["pattern1", "pattern2"] # Optional exclusion patterns
            preserve_structure: true        # Keep nested structure (default: true)
            include_hidden: true           # Include hidden files (default: true)
            module:                        # Module-aware sync (for Go projects)
              type: "go"
              version: "v1.2.3"
            transform:                     # Apply to all files in directory
              repo_name: true
              variables:
                KEY: "value"
```

### Smart Defaults

All directories automatically exclude common development artifacts:

```yaml
# Automatically applied to all directories:
default_exclusions:
  - "*.out"           # Go coverage files
  - "*.test"          # Go test binaries
  - "*.exe"           # Executables
  - "**/.DS_Store"    # macOS system files
  - "**/tmp/*"        # Temporary files
  - "**/.git"         # Git directories
```

### State Tracking

Directory sync integrates seamlessly with go-broadcast's stateless architecture:

- **Branch names** include directory metadata: `chore/sync-files-20250130-143052-abc123f`
- **PR metadata** includes directory performance metrics and file counts
- **State discovery** recognizes directory-synced files through GitHub API
- **Audit trail** maintains complete history of directory operations

## Configuration Reference

### DirectoryMapping Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `src` | string | required | Source directory path in template repository |
| `dest` | string | required | Destination directory path in target repository |
| `exclude` | []string | [] | Glob patterns to exclude (in addition to smart defaults) |
| `preserve_structure` | bool | true | Keep nested directory structure |
| `include_hidden` | bool | true | Include hidden files (starting with .) |
| `transform` | Transform | {} | Apply transformations to all files |
| `module` | ModuleConfig | {} | Module-aware sync configuration (for Go projects) |

### Module-Aware Directory Sync

Directory sync can intelligently handle versioned modules, particularly for Go projects:

```yaml
groups:
  - name: "Shared Go Libraries"
    id: "go-libs"
    source:
      repo: "company/go-modules"
      branch: "main"
    targets:
      - repo: "company/service"
        directories:
          - src: "pkg/errors"
            dest: "vendor/github.com/company/errors"
            module:
              type: "go"              # Module type
              version: "v1.2.3"       # Exact version
              check_tags: true        # Use git tags
              update_refs: false      # Update go.mod references
```

Module sync features:
- **Version Resolution**: Fetches specific versions from git tags
- **Semantic Versioning**: Supports constraints like `~1.2.0`, `^1.2.0`, `latest`
- **Caching**: Caches version lookups for performance
- **Go.mod Updates**: Optionally updates module references

See [Module-Aware Synchronization Guide](module-sync.md) for complete details.

### Exclude Patterns

Custom exclusion patterns are applied **in addition to** smart defaults:

```yaml
directories:
  - src: ".github"
    dest: ".github"
    exclude:
      # File patterns
      - "*.tmp"                    # All .tmp files
      - "*-local.*"                # Files with -local in name
      - "experimental-*"           # Files starting with experimental-

      # Directory patterns
      - "draft/**"                 # Everything under draft/
      - "**/temp/**"               # temp directories at any depth
      - "local/*"                  # Files directly in local/

      # Path-specific patterns
      - "workflows/*-dev.yml"      # Dev workflows in workflows/
      - "coverage/*.out"           # Coverage files in coverage/
```

### Transform Application

Transformations apply to **all files** within the directory:

```yaml
directories:
  - src: "configs"
    dest: "configs"
    transform:
      repo_name: true              # Updates Go module paths in all files
      variables:
        SERVICE: "user-service"    # Replaces {{SERVICE}} in all files
        ENV: "production"          # Replaces ${ENV} in all files
```

### Advanced Configuration

#### Structure Preservation

```yaml
directories:
  # Keep nested structure (default)
  - src: "docs/api/v1"
    dest: "api-docs/v1"
    preserve_structure: true       # Results in: api-docs/v1/nested/file.md

  # Flatten structure
  - src: "templates/production"
    dest: "templates"
    preserve_structure: false      # Results in: templates/file.md (no nesting)
```

#### Hidden Files

```yaml
directories:
  # Include hidden files (default)
  - src: "configs"
    dest: "configs"
    include_hidden: true           # Syncs .env, .gitignore, etc.

  # Skip hidden files
  - src: "public"
    dest: "public"
    include_hidden: false          # Skips .DS_Store, .gitkeep, etc.
```

## Exclusion Patterns

### Pattern Syntax

go-broadcast uses gitignore-style patterns with high-performance compiled matching:

| Pattern | Matches | Example |
|---------|---------|---------|
| `*.ext` | Files with extension | `*.log` matches `app.log` |
| `prefix-*` | Files starting with prefix | `temp-*` matches `temp-file.txt` |
| `*-suffix` | Files ending with suffix | `*-local` matches `config-local` |
| `*keyword*` | Files containing keyword | `*secret*` matches `my-secret.key` |
| `dir/` | Directory at any level | `temp/` matches any `temp` directory |
| `*/file` | File one level deep | `*/config.yml` matches `app/config.yml` |
| `**/pattern` | Pattern at any depth | `**/node_modules` matches anywhere |
| `path/to/file` | Specific path | `src/temp.go` matches exactly |

### Performance Characteristics

- **Zero allocations** - Pattern matching uses pre-compiled engines
- **Cached compilation** - Patterns compiled once per directory sync
- **Fast evaluation** - 107 ns/op for exclusion engine
- **Smart ordering** - Most specific patterns evaluated first

### Common Patterns

#### Development Artifacts

```yaml
exclude:
  # Go development
  - "**/*.test"                    # Test binaries
  - "**/*.out"                     # Coverage files
  - "**/vendor/**"                 # Dependency cache
  - "**/*.prof"                    # Profiling files

  # Node.js development
  - "**/node_modules/**"           # Dependencies
  - "**/dist/**"                   # Build outputs
  - "**/.npm/**"                   # NPM cache
  - "**/npm-debug.log*"            # Debug logs

  # Python development
  - "**/__pycache__/**"            # Python cache
  - "**/*.pyc"                     # Compiled Python
  - "**/venv/**"                   # Virtual environments
  - "**/*.egg-info/**"             # Package metadata
```

#### Environment-Specific Files

```yaml
exclude:
  # Environment variants
  - "**/*-local.*"                 # Local development files
  - "**/*-dev.*"                   # Development files
  - "**/*-staging.*"               # Staging files
  - "configs/local/**"             # Local configuration directory
  - "configs/dev/**"               # Development configuration directory
```

#### Security-Sensitive Files

```yaml
exclude:
  # Credentials and secrets
  - "**/*.key"                     # Private keys
  - "**/*.pem"                     # PEM certificates
  - "**/*secret*"                  # Files with 'secret' in name
  - "**/*password*"                # Files with 'password' in name
  - "**/.env.local"                # Local environment files
  - "**/secrets/**"                # Secrets directories
```

## Performance Characteristics

### Benchmark Results

Directory sync achieves exceptional performance through concurrent processing and API optimization:

| Directory Size | Processing Time | Memory Usage | API Calls |
|----------------|-----------------|--------------|-----------|
| <50 files | <3ms | <1MB | 1 (tree API) |
| 50-150 files | 1-7ms | ~1.2MB | 1 (tree API) |
| 500+ files | 16-32ms | ~1.2MB per 1000 | 1 (tree API) |
| 1000+ files | ~32ms | ~1.2MB | 1 (tree API) |

### Real-World Performance

Based on production usage with actual repositories:

```yaml
# .github/workflows (24 files) - ~1.5ms
# .github/actions (87 files) - ~4ms
# Full .github (149 files with exclusions) - ~7ms
# Documentation (1000+ files) - ~32ms
```

### API Optimization

**Traditional Approach:**
- N API calls for N files
- Rate limit pressure with large directories
- Sequential file existence checks

**go-broadcast Approach:**
- 1 tree API call for all file existence checks
- 90%+ reduction in total API calls
- Batch content operations
- Result: **149 files sync with only 2-3 API calls**

### Memory Efficiency

- **Linear scaling** - ~1.2MB per 1000 files processed
- **Zero allocation patterns** - Exclusion engine uses no allocations
- **Concurrent safe** - Worker pools with controlled memory usage
- **Garbage collection friendly** - Minimal allocation during hot paths

### Progress Reporting

Automatic progress reporting for large directories:

```bash
# Automatically enabled for directories >50 files
Processing .github directory: 87 files found
Applying exclusions: 26 files excluded
Processing files: [████████████████████] 61/61 (100%) - 4ms
Directory sync complete: 61 files synced in 4ms
```

## Real-World Use Cases

### GitHub Infrastructure Sync

Perfect for syncing `.github` directories across organization repositories:

```yaml
version: 1
groups:
  - name: "GitHub Infrastructure"
    id: "github-infra"
    priority: 1
    enabled: true
    source:
      repo: "company/github-templates"
      branch: "main"
    targets:
      - repo: "company/service-a"
      - repo: "company/service-b"
    # Real-world .github sync (149 files total)
    directories:
      - src: ".github/workflows"       # 24 files - ~1.5ms
        dest: ".github/workflows"
        exclude: ["*-local.yml", "*.disabled"]

      - src: ".github/actions"        # 87 files - ~4ms
        dest: ".github/actions"
        exclude: ["*.out", "*.test", "go-coverage"]

      - src: ".github/ISSUE_TEMPLATE"  # Template files
        dest: ".github/ISSUE_TEMPLATE"

# Expected total performance: ~7ms for 149 files
```

### Documentation Management

Sync documentation with structure preservation:

```yaml
directories:
  - src: "docs"
    dest: "docs"
    exclude:
      - "**/_build/**"             # Sphinx builds
      - "**/node_modules/**"       # Node dependencies
      - "**/.vuepress/dist/**"     # VuePress builds
    preserve_structure: true
    transform:
      variables:
        VERSION: "v2.0"
        COMPANY: "ACME Corp"
```

### Configuration Management

Distribute configuration templates:

```yaml
directories:
  - src: "configs/production"
    dest: "configs"
    exclude:
      - "*.local.*"                # Local overrides
      - "*.secret.*"               # Sensitive data
      - "dev/**"                   # Development configs
    transform:
      variables:
        ENV: "production"
        SERVICE_NAME: "user-api"
```

### CI/CD Pipeline Distribution

Share CI/CD workflows across services:

```yaml
directories:
  - src: ".github/workflows"
    dest: ".github/workflows"
    exclude:
      - "*-local.yml"              # Local workflows
      - "experimental-*"           # Experimental workflows
    transform:
      repo_name: true
      variables:
        SERVICE_TYPE: "api"
        DEPLOY_ENV: "production"
```

## Integration with File Sync

Directory sync works seamlessly with existing file sync configurations:

### Mixed Configuration

```yaml
groups:
  - name: "Complete Service Setup"
    id: "complete-setup"
    priority: 1
    enabled: true
    source:
      repo: "company/templates"
      branch: "main"
    targets:
      - repo: "company/service"
        # Individual files (existing functionality)
        files:
          - src: "Makefile"
            dest: "Makefile"
          - src: "README.md"
            dest: "README.md"

        # Directories (enhanced functionality)
        directories:
          - src: ".github/workflows"
            dest: ".github/workflows"
          - src: "configs"
            dest: "configs"
            exclude: ["*.local"]

        # Transforms apply to both files and directories
        transform:
          repo_name: true
          variables:
            SERVICE: "user-service"
```

### Precedence and Conflicts

- **No conflicts** - Files and directories can coexist in same target
- **Transform inheritance** - Directory files inherit target-level transforms
- **Exclusion isolation** - Directory exclusions don't affect individual files
- **State tracking** - Both files and directories tracked in same PR metadata

### Performance Impact

Mixed configurations maintain optimal performance:

```yaml
# Example: 2 individual files + 1 directory (87 files)
# Total: 89 files processed
# API calls: 3 (2 for individual files + 1 tree API for directory)
# Processing time: ~6ms total
```

## Troubleshooting

### Common Issues

#### 1. Empty Directory Results

**Problem:** Directory sync completes but no files are synced.

**Diagnosis:**
```bash
# Use debug logging to see exclusion details
go-broadcast sync --log-level debug --config sync.yaml
```

**Common causes:**
- All files matched exclusion patterns (including smart defaults)
- Source directory doesn't exist in template repository
- Include/exclude patterns too restrictive

**Solution:**
```yaml
# Check what smart defaults exclude
# Default exclusions: *.out, *.test, *.exe, **/.DS_Store, **/tmp/*, **/.git

# Override if needed (rarely recommended)
directories:
  - src: "test-data"
    dest: "test-data"
    exclude: []              # Custom exclusions only
    # Note: Smart defaults still apply for safety
```

#### 2. Performance Issues

**Problem:** Directory sync takes longer than expected.

**Expected performance:**
- <50 files: <3ms
- 50-150 files: 1-7ms
- 500+ files: 16-32ms
- 1000+ files: ~32ms

**Diagnosis:**
```bash
# Enable debug logging for timing details
go-broadcast sync --log-level debug --config sync.yaml 2>&1 | grep -i "duration"
```

**Optimization strategies:**
```yaml
directories:
  - src: "large-directory"
    dest: "large-directory"
    exclude:
      # Exclude large subdirectories early
      - "large-subdir/**"        # Process separately
      - "**/*.zip"               # Skip large archives
      - "**/*.tar.gz"            # Skip compressed files
      - "**/node_modules/**"     # Skip dependencies
```

#### 3. API Rate Limiting

**Problem:** GitHub API rate limits exceeded during large directory sync.

**Understanding limits:**
- Primary rate limit: 5000 requests/hour
- Search API: 30 requests/minute
- go-broadcast typically uses <2% of limits

**Diagnosis:**
```bash
# Check API usage
go-broadcast sync --log-level debug --config sync.yaml 2>&1 | grep -i "api\\|rate"
```

**Solutions:**
- Tree API optimization should prevent this (90%+ call reduction)
- Check for configuration issues causing excessive calls
- Consider splitting very large directories

#### 4. Transform Failures

**Problem:** Transforms fail on some files in directory.

**Behavior:** Transform errors don't fail entire directory sync (by design).

**Diagnosis:**
```bash
# Check transform error details
go-broadcast sync --log-level debug --config sync.yaml 2>&1 | grep -i "transform"
```

**Common causes:**
- Binary files processed as text (handled automatically)
- Invalid variable substitution patterns
- Malformed Go module paths for repo_name transform

**Solution:**
```yaml
directories:
  - src: "mixed-files"
    dest: "mixed-files"
    exclude:
      - "**/*.bin"               # Skip known binary files
      - "**/*.exe"               # Already in smart defaults
      - "**/*.jpg"               # Skip images
    transform:
      repo_name: true            # Only applied to text files
```

### Debugging Tools

#### Dry Run Mode

```bash
# Preview directory sync without making changes
go-broadcast sync --dry-run --config sync.yaml

# Shows:
# - Files that would be synced
# - Files that would be excluded
# - Transform preview
# - Performance estimates
```

#### Debug Logging

```bash
# Comprehensive debug information
go-broadcast sync --log-level debug --config sync.yaml

# Debug output includes:
# - Directory traversal details
# - Exclusion pattern matching
# - Transform application results
# - Performance timing
# - API call details
```

#### Configuration Validation

```bash
# Validate directory configuration
go-broadcast validate --config sync.yaml

# Checks:
# - Directory mapping syntax
# - Exclusion pattern validity
# - Transform configuration
# - Repository access
```

## Advanced Topics

### Custom Exclusion Strategies

#### Include-Only Patterns (Simulated)

```yaml
# Sync only specific file types
directories:
  - src: "mixed-content"
    dest: "filtered-content"
    exclude:
      - "*"                      # Exclude everything
      # Then selectively include by excluding the exclude pattern
      # (Pattern inversion - advanced technique)
```

**Better approach - separate directories:**
```yaml
# More maintainable: separate by content type
directories:
  - src: "docs/markdown"         # Only markdown docs
    dest: "docs"
  - src: "configs/yaml"          # Only YAML configs
    dest: "configs"
```

### Performance Optimization

#### Directory Segmentation

For very large directories (>1000 files), consider segmentation:

```yaml
# Instead of syncing entire large directory
directories:
  # Split into logical segments
  - src: "large-dir/core"
    dest: "large-dir/core"
  - src: "large-dir/modules"
    dest: "large-dir/modules"
  - src: "large-dir/utils"
    dest: "large-dir/utils"

# Benefits:
# - Better granular control
# - Improved error isolation
# - Clearer change tracking
# - Faster individual processing
```

#### Exclusion Pattern Optimization

```yaml
# Optimized pattern ordering (most specific first)
exclude:
  # Exact matches (fastest)
  - "node_modules"
  - ".DS_Store"

  # Extension patterns (very fast)
  - "*.tmp"
  - "*.log"

  # Prefix/suffix patterns (fast)
  - "temp-*"
  - "*-backup"

  # Wildcard patterns (slower)
  - "*secret*"
  - "*password*"

  # Deep recursive patterns (slowest, use sparingly)
  - "**/cache/**"
  - "**/temp/**"
```

### Integration Patterns

#### Multi-Stage Directory Sync

```yaml
version: 1
groups:
  # Stage 1: Core infrastructure (higher priority)
  - name: "Core Infrastructure"
    id: "core-infra"
    priority: 1
    enabled: true
    source:
      repo: "company/core-templates"
      branch: "main"
    targets:
      - repo: "company/service"
        directories:
          - src: ".github/workflows/core"
            dest: ".github/workflows"

  # Stage 2: Service-specific additions (lower priority, depends on stage 1)
  - name: "Service Additions"
    id: "service-additions"
    priority: 2
    depends_on: ["core-infra"]
    enabled: true
    source:
      repo: "company/service-templates"
      branch: "main"
    targets:
      - repo: "company/service"
        directories:
          - src: ".github/workflows/service"
            dest: ".github/workflows"
            exclude: ["core-*"]        # Don't conflict with stage 1
```

#### Conditional Directory Sync

```yaml
# Use different configs for different repository types
# config-api.yaml
directories:
  - src: ".github/workflows/api"
    dest: ".github/workflows"

# config-frontend.yaml
directories:
  - src: ".github/workflows/frontend"
    dest: ".github/workflows"
```

### Monitoring and Observability

#### Performance Tracking

Monitor directory sync performance over time:

```bash
# Extract performance metrics from PR metadata
curl -H "Authorization: token $GITHUB_TOKEN" \
  "https://api.github.com/repos/company/service/pulls" | \
  jq '.[] | select(.title | contains("[Sync]")) | .body' | \
  grep -E "(processing_time_ms|files_synced|api_calls_saved)"
```

#### Success Rate Monitoring

Track sync success rates:

```bash
# Monitor branch creation success
git ls-remote --heads origin | grep "chore/sync-files" | wc -l

# Monitor PR creation success
gh pr list --label "automated-sync" --state all --limit 100
```

---

**Next Steps:**
- Review [Enhanced Troubleshooting Guide](troubleshooting.md) for additional debugging techniques
- Explore [configuration examples](../examples/) for more real-world scenarios
- Check [Performance Guide](performance-guide.md) for advanced optimization techniques

**Related Documentation:**
- [Quick Start Guide](../README.md#-quick-start)
- [Configuration Guide](configuration-guide.md)
- [Module-Aware Sync](module-sync.md)
- [Group Examples](group-examples.md)
- [Enhanced Logging Guide](logging.md)
- [Performance Benchmarks](../README.md#-performance)
- [Example Configurations](../examples/)
