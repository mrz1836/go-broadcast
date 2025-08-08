# go-broadcast Configuration Guide

## Overview

go-broadcast uses a **group-based configuration structure** that enables organizations to manage repository synchronization with multiple template sources, priority-based execution, and granular control over sync operations. Each configuration file contains one or more groups, where each group defines its own source repository and target repositories.

## Configuration Structure

### Top-Level Configuration

```yaml
version: 1                    # Configuration version (required)
name: "My Sync Config"        # Optional configuration name
id: "sync-2025"              # Optional configuration identifier
groups:                      # List of sync groups (required)
  - ...                      # Group definitions
```

### Group Structure

Each group is a self-contained sync unit with its own source and targets:

```yaml
groups:
  - name: "Infrastructure Templates"     # Friendly name for the group
    id: "infra-templates"                # Unique identifier
    description: "Syncs CI/CD files"     # Optional description
    priority: 1                          # Execution order (lower = higher priority)
    enabled: true                        # Enable/disable without removing
    depends_on: ["other-group"]         # Dependencies on other group IDs
    source:                              # Source repository for this group
      repo: "org/templates"
      branch: "main"
    global:                              # Group-level global settings
      pr_labels: ["automated"]
    defaults:                            # Group-level defaults
      branch_prefix: "sync"
    targets:                             # Target repositories
      - repo: "org/service-a"
        files: [...]
        directories: [...]
```

## Group Properties

### Basic Properties

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `name` | string | Yes | Human-readable name for the group |
| `id` | string | Yes | Unique identifier used for dependencies |
| `description` | string | No | Detailed description of the group's purpose |
| `enabled` | boolean | No | Whether to execute this group (default: true) |

### Execution Control

#### Priority

Groups execute in priority order, with lower numbers executing first:

```yaml
groups:
  - name: "Critical Updates"
    id: "critical"
    priority: 1        # Executes first

  - name: "Standard Updates"
    id: "standard"
    priority: 10       # Executes after priority 1

  - name: "Optional Updates"
    id: "optional"
    priority: 100      # Executes last
```

#### Dependencies

Groups can depend on successful completion of other groups:

```yaml
groups:
  - name: "Base Configuration"
    id: "base-config"
    priority: 1
    source:
      repo: "org/base-templates"
    targets: [...]

  - name: "Extended Configuration"
    id: "extended-config"
    priority: 2
    depends_on: ["base-config"]  # Only runs if base-config succeeds
    source:
      repo: "org/extended-templates"
    targets: [...]
```

**Dependency Rules:**
- Dependencies must reference valid group IDs
- Circular dependencies are detected and rejected
- If a dependency fails, dependent groups are skipped
- Dependencies are resolved before priority ordering

### Source Configuration

Each group has its own source repository:

```yaml
source:
  repo: "organization/repository"  # GitHub repository (required)
  branch: "main"                   # Branch to sync from (optional, default: main)
  ref: "abc123"                    # Specific commit/tag (optional)
```

### Target Configuration

Targets define where files are synchronized to:

```yaml
targets:
  - repo: "org/target-repo"
    files:                         # Individual file mappings
      - src: "file.yml"
        dest: "file.yml"
        transform:                 # File-specific transformations
          variables:
            KEY: "value"
    directories:                   # Directory mappings
      - src: "configs"
        dest: "configs"
        exclude: ["*.local"]
        module:                    # Module-aware sync (Go modules)
          type: "go"
          version: "v1.2.3"
    transform:                     # Target-level transformations
      repo_name: true
    pr_labels: ["service-specific"]
    pr_assignees: ["owner"]
```

## Settings Hierarchy

go-broadcast uses a three-level settings hierarchy within each group:

### 1. Global Settings

Applied to ALL targets in the group:

```yaml
global:
  pr_labels: ["automated-sync", "group-name"]
  pr_assignees: ["platform-team"]
  pr_reviewers: ["tech-lead"]
  pr_team_reviewers: ["architecture"]
```

### 2. Default Settings

Used as fallback when neither global nor target-specific settings exist:

```yaml
defaults:
  branch_prefix: "chore/sync"
  pr_labels: ["maintenance"]
  pr_assignees: ["bot"]
```

### 3. Target Settings

Merged with global settings for specific targets:

```yaml
targets:
  - repo: "org/critical-service"
    pr_labels: ["high-priority"]     # Added to global labels
    pr_assignees: ["service-owner"]  # Added to global assignees
```

**Merge Order:** Global + Target → Defaults (as fallback)

## Module-Aware Synchronization

go-broadcast can intelligently sync Go modules with version management:

```yaml
directories:
  - src: "pkg/errors"
    dest: "vendor/github.com/company/errors"
    module:
      type: "go"              # Module type (currently only "go")
      version: "v1.2.3"       # Version constraint
      check_tags: true        # Use git tags for versions
      update_refs: true       # Update go.mod references
```

### Version Constraints

- **Exact version:** `v1.2.3` - Use specific version
- **Latest:** `latest` - Use highest available version
- **Semantic:** `~1.2`, `^1.2.0`, `>=1.2.0` - Semantic version constraints

## Advanced Patterns

### Multi-Group with Dependencies

```yaml
version: 1
name: "Enterprise Sync Configuration"
groups:
  # Base infrastructure - runs first
  - name: "Core Infrastructure"
    id: "core-infra"
    priority: 1
    enabled: true
    source:
      repo: "company/infra-templates"
    targets:
      - repo: "company/service-a"
      - repo: "company/service-b"

  # Security policies - depends on infrastructure
  - name: "Security Policies"
    id: "security"
    priority: 2
    depends_on: ["core-infra"]
    source:
      repo: "company/security-templates"
    targets:
      - repo: "company/service-a"
      - repo: "company/service-b"

  # Optional monitoring - can be disabled
  - name: "Monitoring Setup"
    id: "monitoring"
    priority: 3
    enabled: false  # Temporarily disabled
    depends_on: ["core-infra", "security"]
    source:
      repo: "company/monitoring-templates"
    targets: [...]
```

### Environment-Specific Groups

```yaml
groups:
  # Development environment
  - name: "Development Config"
    id: "dev-config"
    priority: 1
    enabled: true
    source:
      repo: "company/dev-templates"
    targets:
      - repo: "company/service-dev"
        transform:
          variables:
            ENVIRONMENT: "development"

  # Production environment (higher priority)
  - name: "Production Config"
    id: "prod-config"
    priority: 1
    enabled: true
    source:
      repo: "company/prod-templates"
    global:
      pr_labels: ["production", "requires-approval"]
      pr_reviewers: ["senior-team", "security-team"]
    targets:
      - repo: "company/service-prod"
        transform:
          variables:
            ENVIRONMENT: "production"
```

### Phased Rollout Pattern

```yaml
groups:
  # Phase 1: Canary deployment
  - name: "Canary Rollout"
    id: "canary"
    priority: 1
    enabled: true
    source:
      repo: "company/new-templates"
    targets:
      - repo: "company/canary-service"

  # Phase 2: Staging (after canary validation)
  - name: "Staging Rollout"
    id: "staging"
    priority: 2
    enabled: false  # Enable after canary validation
    depends_on: ["canary"]
    source:
      repo: "company/new-templates"
    targets:
      - repo: "company/staging-service-1"
      - repo: "company/staging-service-2"

  # Phase 3: Production (after staging validation)
  - name: "Production Rollout"
    id: "production"
    priority: 3
    enabled: false  # Enable after staging validation
    depends_on: ["staging"]
    source:
      repo: "company/new-templates"
    targets:
      - repo: "company/prod-service-1"
      - repo: "company/prod-service-2"
```

## CLI Integration

### Group Filtering

Execute specific groups:

```bash
# Sync only specific groups
go-broadcast sync --groups "core-infra,security"

# Skip specific groups
go-broadcast sync --skip-groups "optional,experimental"
```

### Status and Validation

```bash
# View group status
go-broadcast status

# Validate configuration including dependencies
go-broadcast validate

# List configured modules
go-broadcast modules list

# Check module versions
go-broadcast modules versions pkg/errors
```

## Best Practices

### 1. Use Descriptive Names and IDs

```yaml
groups:
  - name: "GitHub Actions CI/CD Workflows"  # Clear, descriptive name
    id: "gh-actions-ci-cd"                  # Unique, meaningful ID
    description: "Synchronizes GitHub Actions workflows for CI/CD pipeline"
```

### 2. Organize by Priority

- **Priority 1-10:** Critical infrastructure and security
- **Priority 11-50:** Core application configurations
- **Priority 51-100:** Optional or experimental features

### 3. Leverage Dependencies

Use dependencies to ensure proper sequencing:
- Base configurations before extensions
- Infrastructure before applications
- Security policies before deployment configs

### 4. Use Enable/Disable for Testing

```yaml
groups:
  - name: "Experimental Features"
    id: "experimental"
    enabled: false  # Disable in production, enable for testing
```

### 5. Group by Logical Units

Organize groups by:
- **Source:** Groups sharing the same template source
- **Purpose:** Security, CI/CD, monitoring, etc.
- **Environment:** Development, staging, production
- **Team:** Platform, security, frontend, backend

## Troubleshooting

### Common Issues

**Circular Dependencies:**
```yaml
# This will be rejected
groups:
  - id: "group-a"
    depends_on: ["group-b"]
  - id: "group-b"
    depends_on: ["group-a"]  # Creates circular dependency
```

**Missing Dependencies:**
```yaml
groups:
  - id: "group-a"
    depends_on: ["group-x"]  # Error: group-x doesn't exist
```

**Priority Conflicts:**
```yaml
# These execute in random order (same priority)
groups:
  - id: "group-a"
    priority: 1
  - id: "group-b"
    priority: 1
    # Add explicit dependency if order matters
    depends_on: ["group-a"]
```

## Migration from Legacy Configuration

If migrating from an older single-source configuration, wrap your existing configuration in a group:

**Before:**
```yaml
version: 1
source:
  repo: "org/templates"
targets:
  - repo: "org/service"
```

**After:**
```yaml
version: 1
groups:
  - name: "Default Sync"
    id: "default"
    priority: 1
    enabled: true
    source:
      repo: "org/templates"
    targets:
      - repo: "org/service"
```

## See Also

- [Module-Aware Synchronization Guide](module-sync.md)
- [Group Configuration Examples](group-examples.md)
- [Directory Synchronization](directory-sync.md)
- [Enhanced Troubleshooting Guide](troubleshooting.md)
- [Performance Guide](performance-guide.md)
- [Enhanced Logging Guide](logging.md)
