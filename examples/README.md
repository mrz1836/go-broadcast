# Configuration Examples

This directory contains example configuration files for different use cases of go-broadcast.

## Available Examples

### [`sync.yaml`](sync.yaml) - Complete Example
A comprehensive example showing all available configuration options, including:
- Multiple target repositories
- File transformations (repository name replacement, template variables)
- Different file mapping patterns
- Custom branch prefix and PR labels

**Use case**: Understanding all available features and options.

### [`minimal.yaml`](minimal.yaml) - Minimal Configuration
The simplest possible configuration for syncing files between repositories.
- Single source and target repository
- One file mapping
- No transformations
- Uses all default settings

**Use case**: Getting started quickly or syncing simple files.

### [`microservices.yaml`](microservices.yaml) - Microservices Architecture
Configuration for syncing common tooling across microservices:
- Multiple service repositories
- Common CI/CD pipelines and development tools
- Service-specific template variables
- Repository name transformations for Go modules

**Use case**: Maintaining consistency across microservices architecture.

### [`multi-language.yaml`](multi-language.yaml) - Multi-Language Projects
Syncs platform-wide standards across different technology stacks:
- Go, Python, Node.js, and Rust projects
- Language-specific tooling and configurations
- Common documentation and security policies
- Technology-specific template variables

**Use case**: Organizations using multiple programming languages.

### [`ci-cd-only.yaml`](ci-cd-only.yaml) - CI/CD Pipeline Sync
Focuses exclusively on synchronizing GitHub Actions workflows:
- Frontend, backend, and infrastructure repositories
- Technology-specific CI/CD pipelines
- Security workflows and dependency management
- Environment-specific deployment configurations

**Use case**: Standardizing CI/CD pipelines across projects.

### [`documentation.yaml`](documentation.yaml) - Documentation Templates
Syncs documentation standards and templates:
- Service, frontend, library, and infrastructure repositories
- Standard documentation files (README, CONTRIBUTING, SECURITY)
- Team-specific documentation templates
- Compliance and architecture documentation

**Use case**: Maintaining documentation standards across repositories.

### [`directory-sync.yaml`](directory-sync.yaml) - Directory Synchronization
Comprehensive directory sync examples showing all directory mapping options:
- Basic directory sync with smart defaults
- Custom exclusion patterns for development artifacts
- Mixed file and directory configurations
- Advanced directory options (structure preservation, hidden files)
- Transform application to directory files

**Use case**: Syncing entire directories like `.github`, `configs`, or `docs`.

### [`github-workflows.yaml`](github-workflows.yaml) - GitHub Infrastructure Sync
Real-world example for syncing `.github` directory structures:
- Complete GitHub Actions workflow synchronization
- Coverage system and issue template distribution
- Service-specific workflow customization
- Performance-optimized for large directory structures
- Global PR assignment configuration

**Use case**: Maintaining GitHub repository infrastructure across microservices.

### [`large-directories.yaml`](large-directories.yaml) - Large Directory Management
Demonstrates efficient handling of large directory structures:
- Performance targets for 1000+ files (~32ms processing)
- Directory segmentation strategies
- API optimization techniques (90%+ call reduction)
- Memory usage optimization (~1.2MB per 1000 files)
- Progress reporting for large operations

**Use case**: Organizations with large directory structures requiring high performance.

### [`exclusion-patterns.yaml`](exclusion-patterns.yaml) - Exclusion Pattern Showcase
Comprehensive guide to all exclusion pattern types:
- File extension patterns and wildcards
- Directory and path-based exclusions
- Development artifact patterns (beyond smart defaults)
- Security-sensitive file exclusions
- Complex multi-pattern exclusion strategies

**Use case**: Fine-tuned control over which files are synchronized.

### [`github-complete.yaml`](github-complete.yaml) - Complete GitHub Directory Sync
Enterprise-scale GitHub repository infrastructure synchronization:
- Production service complete `.github` sync (149 files)
- Performance metrics and API optimization examples
- Service-type specific configurations
- Legacy system gradual migration patterns
- Comprehensive real-world performance documentation

**Use case**: Enterprise GitHub infrastructure management at scale.

## Using These Examples

### 1. Copy and Customize
```bash
# Copy an example that matches your use case
cp examples/microservices.yaml my-sync.yaml

# Edit the configuration
vim my-sync.yaml

# Validate the configuration
go-broadcast validate --config my-sync.yaml
```

### 2. Test with Dry Run
```bash
# See what would happen without making changes
go-broadcast sync --config my-sync.yaml --dry-run
```

### 3. Sync Specific Repositories
```bash
# Test with a single repository first
go-broadcast sync org/test-repo --config my-sync.yaml

# Sync all configured repositories
go-broadcast sync --config my-sync.yaml
```

## Configuration Structure

All examples follow this basic structure:

```yaml
version: 1  # Required: configuration version

source:     # Required: source repository
  repo: "org/template-repo"
  branch: "master"

defaults:   # Optional: default settings
  branch_prefix: "chore/sync-files"
  pr_labels: ["automated-sync"]

targets:    # Required: target repositories
  - repo: "org/target-repo"
    files:  # Optional: individual file mappings
      - src: "source/file.txt"
        dest: "destination/file.txt"
    directories:  # Optional: directory mappings
      - src: ".github/workflows"
        dest: ".github/workflows"
        exclude: ["*-local.yml", "*.disabled"]
        preserve_structure: true  # default: true
        include_hidden: true      # default: true
    transform:  # Optional: applies to files and directories
      repo_name: true
      variables:
        KEY: "value"
```

## Common Patterns

### Repository Name Transformation
When `repo_name: true` is set, go-broadcast will replace repository names in files:
```go
// Before (in template repo: org/template-repo)
module github.com/org/template-repo

// After (in target repo: org/service-a)
module github.com/org/service-a
```

### Template Variables
Define variables that will be replaced in files:
```yaml
transform:
  variables:
    SERVICE_NAME: "my-service"
    PORT: "8080"
```

Files can use these variables:
```bash
# Before
SERVICE_NAME={{SERVICE_NAME}}
PORT=${PORT}

# After
SERVICE_NAME=my-service
PORT=8080
```

### File Path Mapping
Files can be renamed or moved during sync:
```yaml
files:
  # Copy to same location
  - src: "Makefile"
    dest: "Makefile"

  # Rename during copy
  - src: "README.template.md"
    dest: "README.md"

  # Move to different directory
  - src: "configs/default.yaml"
    dest: "config/app.yaml"
```

### Directory Sync Patterns

#### Basic Directory Sync
```yaml
directories:
  # Sync entire directory with smart defaults
  - src: ".github/workflows"
    dest: ".github/workflows"
    # Smart defaults automatically exclude: *.out, *.test, *.exe, **/.DS_Store, **/tmp/*, **/.git
```

#### Directory Sync with Custom Exclusions
```yaml
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

#### Mixed Files and Directories
```yaml
files:
  - src: "Makefile"
    dest: "Makefile"
directories:
  - src: "configs"
    dest: "configs"
    exclude: ["*.local", "*.secret"]
transform:
  variables:
    SERVICE_NAME: "my-service"  # Applied to both files and directories
```

#### Advanced Directory Options
```yaml
directories:
  # Preserve nested structure (default)
  - src: "docs/api/v1"
    dest: "api-docs"
    preserve_structure: true       # Results in: api-docs/nested/file.md
    include_hidden: true           # Include .gitignore, .env, etc.

  # Flatten directory structure
  - src: "templates/production"
    dest: "templates"
    preserve_structure: false      # Results in: templates/file.md (no nesting)
    include_hidden: false          # Skip hidden files
```

#### Performance-Optimized Large Directories
```yaml
directories:
  # Segment large directories for better performance
  - src: ".github/workflows"       # 24 files - ~1.5ms
    dest: ".github/workflows"
  - src: ".github/actions"        # 87 files - ~4ms
    dest: ".github/actions"
    exclude: ["*.out", "*.test", "go-coverage"]
  # Expected total: ~5.5ms for 111 files with 90%+ API call reduction
```

#### Exclusion Pattern Examples
```yaml
directories:
  - src: "project-files"
    dest: "project-files"
    exclude:
      # File patterns
      - "*.tmp"                    # All .tmp files
      - "*-local.*"                # Files with -local in name
      - "experimental-*"           # Files starting with experimental-

      # Directory patterns
      - "draft/**"                 # Everything under draft/
      - "**/temp/**"               # temp directories at any depth
      - "cache/*"                  # Files directly in cache/

      # Development artifacts (beyond smart defaults)
      - "**/*.pyc"                 # Python compiled files
      - "**/node_modules/**"       # Node.js dependencies
      - "**/__pycache__/**"        # Python cache directories

      # Security-sensitive files
      - "**/*.key"                 # Private keys
      - "**/*secret*"              # Files containing 'secret'
      - "**/.env.local"            # Local environment files
```

## Best Practices

### General Configuration
1. **Start Small**: Begin with a minimal configuration and add complexity gradually
2. **Use Dry Run**: Always test with `--dry-run` before running actual sync
3. **Test with One Repository**: Test configuration changes with a single repository first
4. **Validate Regularly**: Use `go-broadcast validate` to check configuration syntax
5. **Version Control**: Keep your sync configurations in version control
6. **Document Variables**: Comment your template variables and their purposes
7. **Group by Purpose**: Organize configurations by the type of files being synced

### Directory Sync Best Practices
8. **Trust Smart Defaults**: Smart defaults exclude common artifacts (*.out, *.test, binaries, .DS_Store, tmp files)
9. **Use Specific Exclusions**: Add custom exclusions on top of smart defaults for your specific needs
10. **Test Exclusion Patterns**: Use debug logging to verify patterns match expected files:
    ```bash
    go-broadcast sync --log-level debug --config sync.yaml 2>&1 | grep -i "excluded"
    ```
11. **Segment Large Directories**: For >500 files, consider splitting into logical subdirectories
12. **Monitor Performance**: Expected performance targets:
    - <50 files: <3ms
    - 50-150 files: 1-7ms
    - 500+ files: 16-32ms
    - 1000+ files: ~32ms
13. **Leverage API Optimization**: Directory sync uses GitHub tree API for 90%+ call reduction
14. **Preserve Structure by Default**: Use `preserve_structure: false` only when flattening is specifically needed
15. **Include Hidden Files Safely**: Default `include_hidden: true` is safe due to smart defaults excluding risky hidden files

## Troubleshooting

If you encounter issues with these examples:

1. **Validate the configuration**:
   ```bash
   go-broadcast validate --config examples/sync.yaml
   ```

2. **Check repository access**:
   ```bash
   gh repo view org/repository-name
   ```

3. **Test GitHub authentication**:
   ```bash
   gh auth status
   ```

4. **Enable debug logging**:
   ```bash
   go-broadcast sync --config examples/sync.yaml --log-level debug
   ```

**Note**: Example configurations use placeholder repository names (e.g., "org/template-repo", "company/service-name"). Replace these with actual repository names in your organization when creating your configuration.

## Developer Workflow Integration

For comprehensive go-broadcast development workflows and testing procedures, see [CLAUDE.md](../.github/CLAUDE.md#-essential-development-commands) which includes:
- **Configuration validation workflows** for testing your sync configurations
- **Debugging procedures** for troubleshooting sync issues
- **Testing commands** for validating changes before deployment
- **Performance testing** for optimizing large-scale synchronizations

## Related Documentation

- [Main Documentation](../README.md) - Project overview and quick start guide
- [Directory Sync Guide](../docs/directory-sync.md) - Complete guide to directory synchronization
- [Troubleshooting Guide](../docs/troubleshooting.md) - Detailed troubleshooting procedures (includes directory sync)
- [Configuration Reference](../README.md#configuration) - Complete configuration options
- [Performance Guide](../README.md#-performance) - Performance benchmarks and optimization
- [CLAUDE.md Developer Workflows](../.github/CLAUDE.md) - Comprehensive development workflows
