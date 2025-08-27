# Module-Aware Synchronization

## Overview

Module-aware synchronization is a powerful feature in go-broadcast that intelligently handles versioned modules, particularly for Go projects. Instead of blindly copying directories, go-broadcast can detect modules, resolve version constraints, and ensure your synchronized code maintains version consistency across repositories.

## What is Module-Aware Sync?

Traditional directory synchronization copies files without understanding their context. Module-aware sync:

- **Detects module boundaries** by finding `go.mod` files
- **Resolves version constraints** using semantic versioning
- **Fetches specific versions** from git tags
- **Updates module references** in dependent code
- **Caches version information** for performance

## Benefits

### For Go Projects

- **Version consistency** - Ensure all services use the same module versions
- **Controlled updates** - Specify exact versions or use semantic constraints
- **Dependency management** - Automatically update go.mod references
- **Vendor synchronization** - Sync specific versions to vendor directories

### For Enterprise Teams

- **Gradual rollouts** - Different services can use different versions
- **Version pinning** - Lock critical modules to specific versions
- **Compliance tracking** - Know exactly which versions are deployed where
- **Reduced conflicts** - Version-aware updates prevent breaking changes

## Configuration

### Basic Module Configuration

Add a `module` field to any directory mapping:

```yaml
groups:
  - name: "Shared Libraries"
    id: "shared-libs"
    source:
      repo: "company/go-modules"
    targets:
      - repo: "company/service"
        directories:
          - src: "pkg/errors"
            dest: "vendor/github.com/company/errors"
            module:
              type: "go"           # Module type
              version: "v1.2.3"    # Version specification
```

### Module Configuration Options

| Field         | Type    | Required | Description                                        |
|---------------|---------|----------|----------------------------------------------------|
| `type`        | string  | Yes      | Module type (currently only "go" supported)        |
| `version`     | string  | Yes      | Version constraint or keyword                      |
| `check_tags`  | boolean | No       | Use git tags for version discovery (default: true) |
| `update_refs` | boolean | No       | Update go.mod references (default: false)          |

## Version Specifications

### Exact Versions

Specify an exact version to sync:

```yaml
module:
  type: "go"
  version: "v1.2.3"  # Exact version
```

### Latest Version

Always use the highest available version:

```yaml
module:
  type: "go"
  version: "latest"  # Highest version from git tags
```

### Semantic Version Constraints

Use semantic versioning constraints for flexible updates:

```yaml
# Tilde Range (~)
# Allows patch-level changes
module:
  type: "go"
  version: "~1.2.3"  # Allows 1.2.3, 1.2.4, but not 1.3.0

# Caret Range (^)
# Allows minor and patch-level changes
module:
  type: "go"
  version: "^1.2.3"  # Allows 1.2.3, 1.3.0, but not 2.0.0

# Comparison Operators
module:
  type: "go"
  version: ">=1.2.0"  # Version 1.2.0 or higher

module:
  type: "go"
  version: "<2.0.0"   # Any version below 2.0.0

# Combined Constraints
module:
  type: "go"
  version: ">=1.2.0 <2.0.0"  # Version 1.2.0 up to (but not including) 2.0.0
```

## Module Detection

### Automatic Detection

go-broadcast automatically detects Go modules by looking for `go.mod` files:

```yaml
directories:
  - src: "shared/auth"      # If auth/go.mod exists, it's treated as a module
    dest: "pkg/auth"
    module:
      type: "go"
      version: "latest"
```

### Module Structure

When go-broadcast detects a module:

1. **Finds module root** - Locates the go.mod file
2. **Parses module info** - Extracts module name and current version
3. **Resolves version** - Finds the requested version from git tags
4. **Syncs module files** - Copies the specific version to destination

### Nested Modules

go-broadcast handles nested modules correctly:

```
repository/
├── go.mod                 # Root module
├── pkg/
│   ├── auth/
│   │   └── go.mod        # Nested module - detected separately
│   └── database/
│       └── go.mod        # Another nested module
```

## Version Resolution

### How Versions are Discovered

1. **Git Tags** - Fetches tags from the source repository
2. **Semantic Parsing** - Parses tags as semantic versions
3. **Constraint Matching** - Finds versions matching constraints
4. **Selection** - Chooses the best matching version

### Git Tag Requirements

For version resolution to work:
- Tags must follow semantic versioning (e.g., `v1.2.3`)
- Tags must be pushed to the remote repository
- The source repository must be accessible

### Version Cache

go-broadcast caches version information for performance:

```yaml
# Cache configuration (automatic)
cache:
  ttl: 5m                  # Cache TTL (5 minutes default)
  max_entries: 1000        # Maximum cache entries
```

The cache stores:
- Available versions for each module
- Resolution results for constraints
- Module metadata

## Examples

### Basic Module Sync

Sync a specific version of a shared library:

```yaml
version: 1
groups:
  - name: "Error Package Sync"
    id: "error-pkg"
    source:
      repo: "company/go-commons"
      branch: "main"
    targets:
      - repo: "company/user-service"
        directories:
          - src: "pkg/errors"
            dest: "internal/errors"
            module:
              type: "go"
              version: "v1.2.3"
```

### Multi-Version Strategy

Different services using different versions:

```yaml
groups:
  - name: "Stable Services"
    id: "stable"
    source:
      repo: "company/go-modules"
    targets:
      - repo: "company/payment-service"
        directories:
          - src: "pkg/crypto"
            dest: "vendor/crypto"
            module:
              type: "go"
              version: "v1.0.0"  # Stable version

  - name: "Beta Services"
    id: "beta"
    source:
      repo: "company/go-modules"
    targets:
      - repo: "company/experimental-service"
        directories:
          - src: "pkg/crypto"
            dest: "vendor/crypto"
            module:
              type: "go"
              version: "v2.0.0-beta"  # Beta version
```

### Vendor Directory Management

Sync specific versions to vendor directories:

```yaml
groups:
  - name: "Vendor Dependencies"
    id: "vendor-deps"
    source:
      repo: "company/go-modules"
    targets:
      - repo: "company/monolith"
        directories:
          # Auth module v1.5.x
          - src: "pkg/auth"
            dest: "vendor/github.com/company/auth"
            module:
              type: "go"
              version: "~1.5.0"

          # Database module v2.x
          - src: "pkg/database"
            dest: "vendor/github.com/company/database"
            module:
              type: "go"
              version: "^2.0.0"

          # Logger - always latest
          - src: "pkg/logger"
            dest: "vendor/github.com/company/logger"
            module:
              type: "go"
              version: "latest"
```

### Gradual Version Migration

Migrate services to new versions gradually:

```yaml
groups:
  # Phase 1: Update non-critical services
  - name: "Phase 1 - Non-Critical"
    id: "phase1"
    priority: 1
    enabled: true
    source:
      repo: "company/modules"
    targets:
      - repo: "company/logging-service"
      - repo: "company/metrics-service"
    directories:
      - src: "pkg/framework"
        dest: "vendor/framework"
        module:
          type: "go"
          version: "v2.0.0"

  # Phase 2: Update critical services (disabled initially)
  - name: "Phase 2 - Critical"
    id: "phase2"
    priority: 2
    enabled: false  # Enable after phase 1 validation
    depends_on: ["phase1"]
    source:
      repo: "company/modules"
    targets:
      - repo: "company/payment-service"
      - repo: "company/auth-service"
    directories:
      - src: "pkg/framework"
        dest: "vendor/framework"
        module:
          type: "go"
          version: "v2.0.0"
```

## CLI Commands

### List Modules

View all configured modules:

```bash
go-broadcast modules list

# Output:
# === Configured Modules ===
#
# Group: Shared Go Libraries (go-libs)
#   Module 1:
#     Source: pkg/errors
#     Target: company/service-a -> vendor/github.com/company/errors
#     Type: go
#     Version: v1.2.3
#
# Total modules configured: 1
```

### Show Module Details

Get details about a specific module:

```bash
go-broadcast modules show pkg/errors

# Output:
# === Module: pkg/errors ===
#
# Group: Shared Go Libraries (go-libs)
#   Source Repository: company/go-commons
#   Source Directory: pkg/errors
#   Target Repository: company/service-a
#   Target Directory: vendor/github.com/company/errors
#
# Module Configuration:
#   Type: go
#   Version: v1.2.3
```

### Check Available Versions

Fetch available versions for a module:

```bash
go-broadcast modules versions pkg/errors

# Output:
# === Available Versions for pkg/errors ===
# Source Repository: company/go-commons
#
# Available Versions:
#   • v2.0.0-beta
#   • v1.2.3
#   • v1.2.0
#   • v1.1.0
#   • v1.0.0
#
# Current Configuration: v1.2.3
#   Resolves to: v1.2.3
```

### Validate Module Configuration

Ensure all module configurations are valid:

```bash
go-broadcast modules validate

# Output:
# === Validating Module Configurations ===
#
#   ✓ Module pkg/errors: v1.2.3 -> v1.2.3
#   ✓ Module pkg/database: ^2.0.0 -> v2.3.1
#   Group 'Shared Go Libraries': 2 modules
#
# === Validation Summary ===
# Total Modules: 2
# Valid: 2
# All module configurations are valid!
```

## Performance Optimization

### Caching Strategy

The module resolver uses intelligent caching:

1. **Version Cache** - Caches available versions per repository
2. **Resolution Cache** - Caches constraint resolution results
3. **TTL-based Expiry** - Default 5-minute TTL
4. **Lazy Invalidation** - Cache cleaned on access

### API Call Optimization

To minimize GitHub API calls:

```yaml
# Good: Specify exact versions when possible
module:
  version: "v1.2.3"  # No API call needed if version exists

# Expensive: Always checking for latest
module:
  version: "latest"  # Requires API call to check tags
```

### Batch Processing

When syncing multiple modules from the same source:

```yaml
# Modules from same source are batched
targets:
  - repo: "company/service"
    directories:
      - src: "pkg/auth"
        module:
          version: "v1.0.0"
      - src: "pkg/database"  # Version fetched in same API call
        module:
          version: "v2.0.0"
```

## Troubleshooting

### Common Issues

**Version Not Found:**
```bash
Error: Version v3.0.0 not found for module pkg/auth
Available versions: v1.0.0, v1.1.0, v2.0.0
```
**Solution:** Check available versions with `go-broadcast modules versions`

**No Git Tags:**
```bash
Error: No versions found for module pkg/utils
```
**Solution:** Ensure the source repository has semantic version tags

**Module Detection Failed:**
```bash
Warning: No go.mod found in pkg/helpers, treating as regular directory
```
**Solution:** Module detection requires a go.mod file in the source directory

**Cache Issues:**
```bash
# Force cache refresh
go-broadcast sync --clear-cache
```

### Debug Logging

Enable debug logging to see module resolution details:

```bash
go-broadcast sync --log-level debug

# Shows:
# - Module detection process
# - Version resolution steps
# - Cache hits/misses
# - API calls made
```

## Best Practices

### 1. Use Semantic Versioning

Always tag modules with semantic versions:
```bash
git tag v1.2.3
git push origin v1.2.3
```

### 2. Pin Critical Dependencies

For production-critical modules, use exact versions:
```yaml
module:
  version: "v1.2.3"  # Exact version for stability
```

### 3. Use Constraints for Flexibility

For internal modules, use constraints:
```yaml
module:
  version: "~1.2.0"  # Allow patch updates
```

### 4. Document Version Choices

Add comments explaining version decisions:
```yaml
module:
  type: "go"
  version: "v1.2.3"  # Pinned due to breaking changes in v2.0.0
```

### 5. Test Version Updates

Use groups and enable/disable to test updates:
```yaml
groups:
  - name: "Test New Version"
    enabled: false  # Enable for testing
    module:
      version: "v2.0.0-rc1"
```


## See Also

- [Configuration Guide](configuration-guide.md)
- [Group Examples](group-examples.md)
- [Directory Synchronization](directory-sync.md)
- [Enhanced Troubleshooting Guide](troubleshooting.md)
- [Performance Guide](performance-guide.md)
