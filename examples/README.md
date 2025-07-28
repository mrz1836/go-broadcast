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
  branch_prefix: "sync/template"
  pr_labels: ["automated-sync"]

targets:    # Required: target repositories
  - repo: "org/target-repo"
    files:
      - src: "source/file.txt"
        dest: "destination/file.txt"
    transform:  # Optional: file transformations
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

## Best Practices

1. **Start Small**: Begin with a minimal configuration and add complexity gradually
2. **Use Dry Run**: Always test with `--dry-run` before running actual sync
3. **Test with One Repository**: Test configuration changes with a single repository first
4. **Validate Regularly**: Use `go-broadcast validate` to check configuration syntax
5. **Version Control**: Keep your sync configurations in version control
6. **Document Variables**: Comment your template variables and their purposes
7. **Group by Purpose**: Organize configurations by the type of files being synced

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
- [Troubleshooting Guide](../docs/troubleshooting.md) - Detailed troubleshooting procedures
- [Configuration Reference](../README.md#configuration) - Complete configuration options
- [CLAUDE.md Developer Workflows](../.github/CLAUDE.md) - Comprehensive development workflows