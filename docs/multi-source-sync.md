# Multi-Source Synchronization Guide

## Overview

go-broadcast provides powerful multi-source synchronization capabilities, allowing you to sync files from multiple template repositories to the same target repositories. This feature enables different teams to maintain their own template repositories while ensuring consistent deployment across your organization.

## Table of Contents

- [Why Multi-Source?](#why-multi-source)
- [Configuration Structure](#configuration-structure)
- [Conflict Resolution](#conflict-resolution)
- [Advanced Patterns](#advanced-patterns)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Why Multi-Source?

Modern organizations often have specialized teams managing different aspects of their infrastructure:

- **DevOps Team** maintains CI/CD workflows and build configurations
- **Security Team** manages security policies and compliance configurations
- **Platform Team** owns development standards and tooling
- **Documentation Team** maintains documentation templates and standards

Multi-source synchronization allows each team to:
- Maintain their own template repositories
- Deploy their configurations independently
- Avoid conflicts through intelligent resolution strategies
- Keep clear ownership boundaries

## Configuration Structure

### Basic Multi-Source Configuration

```yaml
version: 1

# Multiple source-to-target mappings
mappings:
  # First source repository
  - source:
      repo: "company/ci-templates"
      branch: "master"
      id: "ci"  # Unique identifier for this source
    targets:
      - repo: "company/service-a"
        files:
          - src: "workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
      - repo: "company/service-b"
        files:
          - src: "workflows/ci.yml"
            dest: ".github/workflows/ci.yml"

  # Second source repository
  - source:
      repo: "company/security-templates"
      branch: "master"
      id: "security"
    targets:
      - repo: "company/service-a"
        files:
          - src: "policies/security.yml"
            dest: "security/policies.yml"
      - repo: "company/service-b"
        files:
          - src: "policies/security.yml"
            dest: "security/policies.yml"

# Global settings apply to all mappings
global:
  pr_labels: ["automated-sync", "multi-source"]
  pr_assignees: ["platform-team"]
```

### Key Components

#### Source Configuration

Each source must have:
- **repo**: Repository path (e.g., "org/repo")
- **branch**: Branch to sync from
- **id**: Unique identifier used in branch names and conflict resolution

```yaml
source:
  repo: "company/templates"
  branch: "main"
  id: "templates"  # Must be unique across all sources
```

#### Mapping-Specific Defaults

Each mapping can have its own default settings:

```yaml
mappings:
  - source:
      repo: "company/security-templates"
      id: "security"
    targets:
      - repo: "company/service"
    # These defaults only apply to this mapping
    defaults:
      branch_prefix: "security/sync"
      pr_labels: ["security", "compliance"]
      pr_reviewers: ["security-lead"]
      pr_team_reviewers: ["security-team"]
```

## Conflict Resolution

When multiple sources target the same file in a repository, go-broadcast provides three resolution strategies:

### 1. Last-Wins Strategy (Default)

The last mapping in the configuration file wins all conflicts:

```yaml
version: 1

conflict_resolution:
  strategy: "last-wins"  # Default if not specified

mappings:
  - source:
      repo: "company/base-templates"
      id: "base"
    targets:
      - repo: "company/my-service"
        files:
          - src: "Makefile"
            dest: "Makefile"  # Will be overwritten by golang mapping

  - source:
      repo: "company/golang-templates"
      id: "golang"
    targets:
      - repo: "company/my-service"
        files:
          - src: "go/Makefile"
            dest: "Makefile"  # This one wins!
```

### 2. Priority-Based Resolution

Explicitly define priority order for sources:

```yaml
version: 1

conflict_resolution:
  strategy: "priority"
  priority: ["security", "platform", "team", "base"]  # Higher priority first

mappings:
  # Base configuration (lowest priority)
  - source:
      repo: "company/base-config"
      id: "base"
    targets:
      - repo: "company/service"
        files:
          - src: "config.yml"
            dest: "config.yml"

  # Security configuration (highest priority - wins!)
  - source:
      repo: "company/security-config"
      id: "security"
    targets:
      - repo: "company/service"
        files:
          - src: "secure/config.yml"
            dest: "config.yml"  # Wins due to priority
```

### 3. Error on Conflict

Fail the sync if any conflicts are detected:

```yaml
version: 1

conflict_resolution:
  strategy: "error"  # Strict mode - no conflicts allowed

mappings:
  # This configuration will fail if multiple sources
  # try to sync to the same file in any target
```

## Advanced Patterns

### Team-Based Repository Management

Structure your sources by team ownership:

```yaml
version: 1

conflict_resolution:
  strategy: "priority"
  priority: ["security", "platform", "devops", "docs"]

mappings:
  # Security team (highest priority)
  - source:
      repo: "company/security-templates"
      id: "security"
    targets:
      - repo: "company/service-*"  # Glob patterns supported
    defaults:
      pr_team_reviewers: ["security-team"]
      pr_labels: ["security", "compliance"]

  # Platform team
  - source:
      repo: "company/platform-standards"
      id: "platform"
    targets:
      - repo: "company/service-*"
    defaults:
      pr_team_reviewers: ["platform-team"]
      pr_labels: ["platform", "standards"]

  # DevOps team
  - source:
      repo: "company/ci-cd-templates"
      id: "devops"
    targets:
      - repo: "company/service-*"
    defaults:
      pr_team_reviewers: ["devops-team"]
      pr_labels: ["ci-cd", "automation"]
```

### Mixed File and Directory Sync

Combine file and directory synchronization from multiple sources:

```yaml
version: 1

mappings:
  # Base configuration with directories
  - source:
      repo: "company/base-config"
      id: "base"
    targets:
      - repo: "company/service"
        directories:
          - src: "configs"
            dest: "configs"
            exclude: ["*.local", "*.secret"]
          - src: "scripts"
            dest: "scripts"

  # Override specific files from directories
  - source:
      repo: "company/service-specific"
      id: "custom"
    targets:
      - repo: "company/service"
        files:
          # These override files from the base directory sync
          - src: "custom/database.yml"
            dest: "configs/database.yml"
          - src: "custom/deploy.sh"
            dest: "scripts/deploy.sh"
```

### Conditional Synchronization

Use transforms and variables for environment-specific configurations:

```yaml
version: 1

mappings:
  # Development environment templates
  - source:
      repo: "company/dev-templates"
      id: "dev"
    targets:
      - repo: "company/service-dev"
        transform:
          variables:
            ENVIRONMENT: "development"
            DEBUG: "true"

  # Production environment templates
  - source:
      repo: "company/prod-templates"
      id: "prod"
    targets:
      - repo: "company/service-prod"
        transform:
          variables:
            ENVIRONMENT: "production"
            DEBUG: "false"
```

## Best Practices

### 1. Use Descriptive Source IDs

Source IDs appear in branch names and logs. Make them meaningful:

```yaml
# Good
source:
  id: "security"      # Clear purpose
  id: "ci-cd"         # Team ownership obvious
  id: "docs"          # Self-explanatory

# Avoid
source:
  id: "src1"          # Not descriptive
  id: "a"             # Too short
  id: "my-templates"  # Vague
```

### 2. Organize by Team Ownership

Structure your configuration to reflect organizational boundaries:

```yaml
mappings:
  # Group related sources together
  # Security team sources
  - source:
      repo: "security/policies"
      id: "security-policies"
  - source:
      repo: "security/scanning"
      id: "security-scanning"

  # Platform team sources
  - source:
      repo: "platform/standards"
      id: "platform-standards"
  - source:
      repo: "platform/tools"
      id: "platform-tools"
```

### 3. Use Priority Strategy for Clear Precedence

When you have a clear hierarchy of importance:

```yaml
conflict_resolution:
  strategy: "priority"
  priority: [
    "critical-security",  # Security always wins
    "compliance",         # Compliance requirements next
    "platform",           # Platform standards
    "team",               # Team customizations
    "base"                # Base templates last
  ]
```

### 4. Document Conflict Expectations

Add comments explaining why certain sources should win conflicts:

```yaml
mappings:
  - source:
      repo: "company/base-templates"
      id: "base"
    # Base templates provide defaults that teams can override

  - source:
      repo: "company/team-overrides"
      id: "team"
    # Team overrides should win over base templates
    # but lose to security requirements
```

### 5. Test Conflict Resolution

Always test your configuration with dry-run:

```bash
# See what would happen without making changes
go-broadcast sync --dry-run --config multi-source.yaml

# Check specific repository
go-broadcast sync company/my-service --dry-run --config multi-source.yaml

# Validate configuration
go-broadcast validate --config multi-source.yaml
```

## Troubleshooting

### Common Issues

#### 1. Unexpected File Overwrites

**Symptom**: Files are being overwritten by the wrong source

**Solution**: Check your conflict resolution strategy:
```bash
# Enable debug logging to see conflict resolution
go-broadcast sync --log-level debug --config multi-source.yaml
```

Look for logs like:
```
DEBUG Conflict detected for file: Makefile
DEBUG Sources competing: [base, golang, custom]
DEBUG Resolution strategy: last-wins
DEBUG Winner: custom (mapping index: 2)
```

#### 2. Source ID Conflicts

**Symptom**: Error about duplicate source IDs

**Solution**: Ensure all source IDs are unique:
```yaml
mappings:
  - source:
      id: "ci"  # Must be unique
  - source:
      id: "ci"  # ERROR: Duplicate ID!
```

#### 3. Priority Not Working

**Symptom**: Priority strategy not selecting expected source

**Solution**: Verify source IDs match priority list exactly:
```yaml
conflict_resolution:
  strategy: "priority"
  priority: ["security", "ci"]  # IDs must match exactly

mappings:
  - source:
      id: "Security"  # ERROR: Case mismatch!
  - source:
      id: "ci"        # Correct
```

#### 4. Branches Not Found

**Symptom**: Can't find sync branches for multi-source

**Solution**: Check branch naming pattern:
- Single source: `chore/sync-files-20250130-143052-abc123f`
- Multi-source: `chore/sync-files-security-20250130-143052-abc123f`

The source ID is included after "sync-files" in multi-source branch names.

### Debug Commands

```bash
# Check current sync status
go-broadcast status --config multi-source.yaml

# See what files would be synced
go-broadcast sync --dry-run --config multi-source.yaml

# Enable debug logging
go-broadcast sync --log-level debug --config multi-source.yaml

# Validate configuration
go-broadcast validate --config multi-source.yaml

# Cancel problematic syncs
go-broadcast cancel --dry-run --config multi-source.yaml
```

## See Also

- [Configuration Reference](../README.md#configuration-reference)
- [Example: multi-source.yaml](../examples/multi-source.yaml)
- [Example: multi-source-conflicts.yaml](../examples/multi-source-conflicts.yaml)
- [Troubleshooting Guide](troubleshooting.md)
