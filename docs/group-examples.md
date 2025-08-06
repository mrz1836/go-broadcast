# go-broadcast Group Configuration Examples

This document provides practical examples of group-based configurations for various scenarios.

## Table of Contents

- [Simple Single-Group Configuration](#simple-single-group-configuration)
- [Multi-Group with Priorities](#multi-group-with-priorities)
- [Groups with Dependencies](#groups-with-dependencies)
- [Module-Aware Configurations](#module-aware-configurations)
- [Complex Enterprise Scenarios](#complex-enterprise-scenarios)
- [Environment-Based Groups](#environment-based-groups)
- [Team-Based Organization](#team-based-organization)
- [Phased Rollout Strategy](#phased-rollout-strategy)

## Simple Single-Group Configuration

The most basic configuration with a single group:

```yaml
version: 1
name: "Basic Repository Sync"
id: "basic-sync"
groups:
  - name: "Default Sync"
    id: "default"
    description: "Synchronizes common files across repositories"
    priority: 1
    enabled: true
    source:
      repo: "organization/template-repo"
      branch: "main"
    targets:
      - repo: "organization/service-a"
        files:
          - src: ".github/workflows/ci.yml"
            dest: ".github/workflows/ci.yml"
          - src: "Makefile"
            dest: "Makefile"
        directories:
          - src: ".github/actions"
            dest: ".github/actions"
```

## Multi-Group with Priorities

Multiple groups executing in priority order:

```yaml
version: 1
name: "Platform Configuration Suite"
groups:
  # Critical security policies - highest priority
  - name: "Security Policies"
    id: "security"
    description: "Security configurations and policies"
    priority: 1  # Executes first
    enabled: true
    source:
      repo: "platform/security-templates"
      branch: "main"
    targets:
      - repo: "platform/payment-service"
      - repo: "platform/auth-service"
      - repo: "platform/user-service"

  # CI/CD configurations - medium priority
  - name: "CI/CD Pipelines"
    id: "ci-cd"
    description: "GitHub Actions workflows and build configs"
    priority: 10  # Executes after security
    enabled: true
    source:
      repo: "platform/ci-templates"
      branch: "main"
    targets:
      - repo: "platform/payment-service"
      - repo: "platform/auth-service"
      - repo: "platform/user-service"

  # Optional monitoring - lowest priority
  - name: "Monitoring Setup"
    id: "monitoring"
    description: "Observability and monitoring configurations"
    priority: 100  # Executes last
    enabled: true
    source:
      repo: "platform/monitoring-templates"
      branch: "main"
    targets:
      - repo: "platform/payment-service"
        files:
          - src: "datadog/monitors.yml"
            dest: ".monitoring/datadog.yml"
```

## Groups with Dependencies

Groups that depend on successful completion of others:

```yaml
version: 1
name: "Dependent Infrastructure Setup"
groups:
  # Base infrastructure must succeed first
  - name: "Base Infrastructure"
    id: "base-infra"
    priority: 1
    enabled: true
    source:
      repo: "infra/base-templates"
      branch: "stable"
    targets:
      - repo: "app/service-a"
        files:
          - src: "docker/Dockerfile.base"
            dest: "Dockerfile"
          - src: "docker-compose.yml"
            dest: "docker-compose.yml"

  # Networking depends on base infrastructure
  - name: "Network Configuration"
    id: "network"
    priority: 2
    depends_on: ["base-infra"]  # Only runs if base-infra succeeds
    enabled: true
    source:
      repo: "infra/network-templates"
      branch: "stable"
    targets:
      - repo: "app/service-a"
        files:
          - src: "nginx/nginx.conf"
            dest: "config/nginx.conf"

  # Application config depends on both base and network
  - name: "Application Configuration"
    id: "app-config"
    priority: 3
    depends_on: ["base-infra", "network"]  # Requires both to succeed
    enabled: true
    source:
      repo: "infra/app-templates"
      branch: "stable"
    targets:
      - repo: "app/service-a"
        directories:
          - src: "config"
            dest: "config"
            exclude: ["*.local", "*.dev"]
```

## Module-Aware Configurations

Synchronizing Go modules with version management:

```yaml
version: 1
name: "Go Module Distribution"
groups:
  - name: "Shared Go Libraries"
    id: "go-libs"
    description: "Distributes versioned Go modules across services"
    priority: 1
    enabled: true
    source:
      repo: "company/go-commons"
      branch: "main"
    targets:
      # Production service - stable versions
      - repo: "company/payment-service"
        directories:
          - src: "pkg/errors"
            dest: "vendor/github.com/company/errors"
            module:
              type: "go"
              version: "v1.2.3"  # Exact stable version
          - src: "pkg/logging"
            dest: "vendor/github.com/company/logging"
            module:
              type: "go"
              version: "~1.5.0"  # Allow patch updates

      # Development service - latest versions
      - repo: "company/experimental-service"
        directories:
          - src: "pkg/errors"
            dest: "vendor/github.com/company/errors"
            module:
              type: "go"
              version: "latest"  # Always use latest
          - src: "pkg/logging"
            dest: "vendor/github.com/company/logging"
            module:
              type: "go"
              version: "^2.0.0"  # Allow minor updates
```

## Complex Enterprise Scenarios

Large-scale enterprise configuration with multiple teams and environments:

```yaml
version: 1
name: "Enterprise Platform Synchronization"
id: "enterprise-sync-2025"
groups:
  # Platform team - core infrastructure
  - name: "Platform Core"
    id: "platform-core"
    description: "Core platform configurations managed by Platform Team"
    priority: 1
    enabled: true
    source:
      repo: "enterprise/platform-core"
      branch: "release/2025.1"
    global:
      pr_labels: ["platform", "automated"]
      pr_assignees: ["platform-team"]
      pr_reviewers: ["platform-lead", "platform-architect"]
    defaults:
      branch_prefix: "platform/sync"
    targets:
      - repo: "enterprise/service-a"
      - repo: "enterprise/service-b"
      - repo: "enterprise/service-c"
        pr_labels: ["critical"]  # Additional label for critical service

  # Security team - compliance and policies
  - name: "Security Compliance"
    id: "security-compliance"
    description: "Security policies and compliance configurations"
    priority: 2
    depends_on: ["platform-core"]
    enabled: true
    source:
      repo: "enterprise/security-policies"
      branch: "compliance/q1-2025"
    global:
      pr_labels: ["security", "compliance", "requires-approval"]
      pr_reviewers: ["security-team", "compliance-officer"]
      pr_team_reviewers: ["security-board"]
    targets:
      - repo: "enterprise/service-a"
        files:
          - src: "policies/security.yml"
            dest: ".security/policy.yml"
          - src: "policies/compliance.md"
            dest: "COMPLIANCE.md"

  # DevOps team - CI/CD and monitoring
  - name: "DevOps Tooling"
    id: "devops"
    description: "CI/CD pipelines and monitoring setup"
    priority: 10
    depends_on: ["platform-core"]
    enabled: true
    source:
      repo: "enterprise/devops-templates"
      branch: "main"
    targets:
      - repo: "enterprise/service-a"
        directories:
          - src: ".github"
            dest: ".github"
            exclude: ["*.local", "workflows/experimental-*"]
        transform:
          variables:
            ENVIRONMENT: "production"
            REGION: "us-east-1"

  # Data team - optional data pipeline configs
  - name: "Data Pipeline Configuration"
    id: "data-pipeline"
    description: "Data pipeline and ETL configurations"
    priority: 100
    enabled: false  # Disabled by default, enable per service
    source:
      repo: "enterprise/data-templates"
      branch: "stable"
    targets:
      - repo: "enterprise/analytics-service"
        directories:
          - src: "airflow/dags"
            dest: "dags"
          - src: "spark/configs"
            dest: "spark-configs"
```

## Environment-Based Groups

Different configurations for different environments:

```yaml
version: 1
name: "Environment-Specific Configurations"
groups:
  # Development environment
  - name: "Development Environment"
    id: "env-dev"
    description: "Development environment configurations"
    priority: 1
    enabled: true
    source:
      repo: "config/dev-templates"
      branch: "develop"
    defaults:
      pr_labels: ["dev", "auto-merge"]
    targets:
      - repo: "app/service-dev"
        files:
          - src: "config/app.yml"
            dest: "config/app.yml"
        transform:
          variables:
            ENVIRONMENT: "development"
            DEBUG: "true"
            LOG_LEVEL: "debug"

  # Staging environment
  - name: "Staging Environment"
    id: "env-staging"
    description: "Staging environment configurations"
    priority: 1
    enabled: true
    source:
      repo: "config/staging-templates"
      branch: "staging"
    defaults:
      pr_labels: ["staging", "qa-review"]
      pr_reviewers: ["qa-team"]
    targets:
      - repo: "app/service-staging"
        files:
          - src: "config/app.yml"
            dest: "config/app.yml"
        transform:
          variables:
            ENVIRONMENT: "staging"
            DEBUG: "false"
            LOG_LEVEL: "info"

  # Production environment
  - name: "Production Environment"
    id: "env-prod"
    description: "Production environment configurations"
    priority: 1
    enabled: true
    source:
      repo: "config/prod-templates"
      branch: "production"
    global:
      pr_labels: ["production", "requires-approval", "no-auto-merge"]
      pr_reviewers: ["tech-lead", "security-team"]
      pr_team_reviewers: ["production-approvers"]
    targets:
      - repo: "app/service-prod"
        files:
          - src: "config/app.yml"
            dest: "config/app.yml"
        transform:
          variables:
            ENVIRONMENT: "production"
            DEBUG: "false"
            LOG_LEVEL: "error"
```

## Team-Based Organization

Organizing groups by team ownership:

```yaml
version: 1
name: "Team-Based Repository Management"
groups:
  # Frontend team configurations
  - name: "Frontend Standards"
    id: "frontend"
    description: "Frontend team's standard configurations"
    priority: 10
    enabled: true
    source:
      repo: "frontend/templates"
      branch: "main"
    global:
      pr_assignees: ["frontend-team"]
      pr_reviewers: ["frontend-lead"]
    targets:
      - repo: "frontend/web-app"
      - repo: "frontend/mobile-app"
      - repo: "frontend/admin-portal"

  # Backend team configurations
  - name: "Backend Standards"
    id: "backend"
    description: "Backend team's service configurations"
    priority: 10
    enabled: true
    source:
      repo: "backend/templates"
      branch: "main"
    global:
      pr_assignees: ["backend-team"]
      pr_reviewers: ["backend-lead"]
    targets:
      - repo: "backend/api-gateway"
      - repo: "backend/user-service"
      - repo: "backend/payment-service"

  # Mobile team configurations
  - name: "Mobile Standards"
    id: "mobile"
    description: "Mobile team's build and deployment configs"
    priority: 10
    enabled: true
    source:
      repo: "mobile/templates"
      branch: "main"
    global:
      pr_assignees: ["mobile-team"]
      pr_reviewers: ["mobile-lead"]
    targets:
      - repo: "mobile/ios-app"
      - repo: "mobile/android-app"

  # Cross-team shared configurations
  - name: "Shared Standards"
    id: "shared"
    description: "Cross-team shared configurations"
    priority: 1  # Higher priority than team-specific
    enabled: true
    source:
      repo: "platform/shared-templates"
      branch: "main"
    global:
      pr_labels: ["shared", "cross-team"]
      pr_reviewers: ["tech-council"]
    targets:
      - repo: "frontend/web-app"
      - repo: "backend/api-gateway"
      - repo: "mobile/ios-app"
```

## Phased Rollout Strategy

Gradual rollout with validation gates:

```yaml
version: 1
name: "Phased Feature Rollout"
groups:
  # Phase 1: Canary deployment (5% of services)
  - name: "Phase 1 - Canary"
    id: "rollout-canary"
    description: "Initial canary deployment to test services"
    priority: 1
    enabled: true
    source:
      repo: "features/new-framework"
      branch: "release/v2.0"
    global:
      pr_labels: ["canary", "experimental"]
      pr_assignees: ["release-manager"]
    targets:
      - repo: "services/test-service-1"
        transform:
          variables:
            FEATURE_FLAG: "canary"
            ROLLOUT_PERCENTAGE: "5"

  # Phase 2: Early adopters (25% of services)
  - name: "Phase 2 - Early Adopters"
    id: "rollout-early"
    description: "Rollout to early adopter services"
    priority: 2
    depends_on: ["rollout-canary"]
    enabled: false  # Enable after canary validation
    source:
      repo: "features/new-framework"
      branch: "release/v2.0"
    global:
      pr_labels: ["early-adopter", "beta"]
    targets:
      - repo: "services/beta-service-1"
      - repo: "services/beta-service-2"
      - repo: "services/beta-service-3"
        transform:
          variables:
            FEATURE_FLAG: "beta"
            ROLLOUT_PERCENTAGE: "25"

  # Phase 3: General availability (50% of services)
  - name: "Phase 3 - General Availability"
    id: "rollout-ga"
    description: "General availability rollout"
    priority: 3
    depends_on: ["rollout-early"]
    enabled: false  # Enable after early adopter validation
    source:
      repo: "features/new-framework"
      branch: "release/v2.0"
    targets:
      - repo: "services/prod-service-1"
      - repo: "services/prod-service-2"
      - repo: "services/prod-service-3"
      - repo: "services/prod-service-4"
      - repo: "services/prod-service-5"
        transform:
          variables:
            FEATURE_FLAG: "ga"
            ROLLOUT_PERCENTAGE: "50"

  # Phase 4: Full rollout (100% of services)
  - name: "Phase 4 - Full Rollout"
    id: "rollout-full"
    description: "Complete rollout to all services"
    priority: 4
    depends_on: ["rollout-ga"]
    enabled: false  # Enable for final rollout
    source:
      repo: "features/new-framework"
      branch: "release/v2.0"
    global:
      pr_labels: ["production", "full-rollout"]
      pr_reviewers: ["tech-lead", "product-owner"]
    targets:
      - repo: "services/prod-service-6"
      - repo: "services/prod-service-7"
      - repo: "services/prod-service-8"
      - repo: "services/prod-service-9"
      - repo: "services/prod-service-10"
        transform:
          variables:
            FEATURE_FLAG: "enabled"
            ROLLOUT_PERCENTAGE: "100"
```

## See Also

- [Configuration Guide](configuration-guide.md) - Complete configuration reference
- [Module-Aware Sync](module-sync.md) - Module versioning and management
- [Directory Sync](directory-sync.md) - Directory synchronization details
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
