# go-broadcast Configuration Implementation Plan

## Executive Summary

This document outlines a comprehensive plan to implement go-broadcast's configuration structure where configurations contain groups, and each group has its own source-target mappings. Since go-broadcast is unreleased software, this will be the standard and only configuration format, enabling organizations to manage repository synchronization with multiple template sources, priority-based execution, and granular control over sync operations.

**Key Configuration Features:**
- **Group-Based Structure**: Configuration organized into named groups
- **Priority-Based Execution**: Groups execute in priority order (lower number = higher priority)
- **Group Dependencies**: Groups can depend on other groups completing first
- **Enable/Disable Control**: Groups can be toggled on/off without removing configuration
- **Enhanced Metadata**: Each group has name, ID, and optional description for clarity
- **Source per Group**: Each group defines its own source repository and targets
- **Module-Aware Sync**: Smart handling of Go modules with version tracking
- **Stateless Operation**: Maintains go-broadcast's core principle of stateless sync

## Vision Statement

go-broadcast is designed with a group-based configuration structure as its core architecture, providing:
- **Complex Sync Scenarios**: Different template sources for different parts of the organization
- **Phased Rollouts**: Priority-based execution for controlled deployments
- **Dependency Management**: Groups can depend on successful completion of other groups
- **Module Intelligence**: Automatically detect and sync Go modules by version
- **Organizational Structure**: Groups aligned with teams, projects, or environments
- **Operational Control**: Enable/disable groups for maintenance or testing
- **Clear Documentation**: Self-documenting configurations with names and descriptions

This is the fundamental design of go-broadcast, not an enhancement or migration path.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   go-broadcast Sync Engine                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────────────────┐               │
│  │ Config       │───▶│ Group Orchestrator       │               │
│  │ (multi-group)│    │                          │               │
│  └──────────────┘    │ ├─ resolveDependencies() │               │
│                      │ ├─ sortByPriority()      │               │
│                      │ ├─ filterEnabled()       │               │
│                      │ └─ executeGroups()       │               │
│                      └──────────┬───────────────┘               │
│                                 │                               │
│                                 ▼                               │
│  ┌─────────────────────────────────────────────┐                │
│  │          Group Execution Engine             │                │
│  │                                             │                │
│  │  For each enabled group (by priority):      │                │
│  │  ├─ Clone source repository                 │                │
│  │  ├─ Detect Go modules in directories        │                │
│  │  ├─ Resolve module versions                 │                │
│  │  ├─ Process all targets in group            │                │
│  │  ├─ Apply group-level defaults              │                │
│  │  ├─ Track group execution status            │                │
│  │  └─ Report group-level metrics              │                │
│  └─────────────────────────────────────────────┘                │
│                                                                 │
│  State Tracking Enhancement:                                    │
│  - Branch: chore/sync-files-{group_id}-20250130-143052-abc123f  │
│  - PR Metadata: Includes group name, ID, and execution context  │
└─────────────────────────────────────────────────────────────────┘

New Configuration Structure:
version: 1
name: "Platform Repository Sync"           # Optional top-level name
id: "platform-sync-2025"                  # Optional top-level ID
groups:
  - name: "Core Infrastructure"
    id: "core-infra"
    description: "Syncs core CI/CD and build infrastructure"
    priority: 1                           # Executes first
    enabled: true
    source:
      repo: "company/infrastructure-templates"
      branch: "main"
    global:                               # Group-level global settings
      pr_labels: ["infrastructure", "automated"]
    targets:
      - repo: "company/service-a"
        files: [...]
        directories: [...]

  - name: "Security Policies"
    id: "security"
    description: "Syncs security policies and configurations"
    priority: 2                           # Executes second
    enabled: true
    depends_on: ["core-infra"]            # Wait for core-infra to complete
    source:
      repo: "company/security-templates"
      branch: "main"
    targets: [...]
```

## Implementation Roadmap

### Phase 1: Configuration Types Implementation
**Objective**: Implement configuration types with group-based structure

**Implementation Steps:**
1. Update `Config` type to use group-based structure
2. Add `Group` type with all required fields
3. Implement validation logic for groups
4. Update parser to handle group-based configuration
5. Add comprehensive tests for configuration structure

**Files to Create/Modify:**
- `internal/config/types.go` - Define group-based configuration types
- `internal/config/validator.go` - Add group validation
- `internal/config/parser.go` - Parse group-based configuration
- `internal/config/config_test.go` - Tests for configuration structure

**Type Definitions:**
```go
// Config represents the complete sync configuration
type Config struct {
    Version int       `yaml:"version"`          // Config version (1)
    Name    string    `yaml:"name,omitempty"`   // Optional config name
    ID      string    `yaml:"id,omitempty"`     // Optional config ID
    Groups   []Group   `yaml:"groups"`          // List of sync groups
}

// Group represents a sync group with its own source and targets
type Group struct {
    Name        string         `yaml:"name"`                    // Friendly name
    ID          string         `yaml:"id"`                      // Unique identifier
    Description string         `yaml:"description,omitempty"`   // Optional description
    Priority    int            `yaml:"priority,omitempty"`      // Execution order (default: 0)
    DependsOn   []string       `yaml:"depends_on,omitempty"`    // Group IDs this group depends on
    Enabled     *bool          `yaml:"enabled,omitempty"`       // Toggle on/off (default: true)
    Source      SourceConfig   `yaml:"source"`                  // Source repository
    Global      GlobalConfig   `yaml:"global,omitempty"`        // Group-level globals
    Defaults    DefaultConfig  `yaml:"defaults,omitempty"`      // Group-level defaults
    Targets     []TargetConfig `yaml:"targets"`                 // Target repositories
}

// DirectoryMapping defines source to destination directory mapping
type DirectoryMapping struct {
    Src               string         `yaml:"src"`                          // Source directory path
    Dest              string         `yaml:"dest"`                         // Destination directory path
    Exclude           []string       `yaml:"exclude,omitempty"`            // Glob patterns to exclude
    IncludeOnly       []string       `yaml:"include_only,omitempty"`       // Glob patterns to include
    Transform         Transform      `yaml:"transform,omitempty"`          // Apply to all files
    PreserveStructure *bool          `yaml:"preserve_structure,omitempty"` // Keep nested structure (default: true)
    IncludeHidden     *bool          `yaml:"include_hidden,omitempty"`     // Include hidden files (default: true)
    Module            *ModuleConfig  `yaml:"module,omitempty"`             // Module-aware sync settings
}

// ModuleConfig defines module-aware sync settings
type ModuleConfig struct {
    Type        string `yaml:"type,omitempty"`         // Module type: "go" (future: "npm", "python")
    Version     string `yaml:"version"`                // Version constraint (exact, latest, or semver)
    CheckTags   *bool  `yaml:"check_tags,omitempty"`   // Use git tags for versions (default: true)
    UpdateRefs  bool   `yaml:"update_refs,omitempty"`  // Update go.mod references
}
```

**Validation Rules:**
- Group names must be unique within a config
- Group IDs must be unique and valid (alphanumeric + hyphens)
- Priority values should be positive integers
- At least one enabled group required
- Source and targets required for each group
- Dependencies must reference valid group IDs
- No circular dependencies allowed
- Module version must be valid (exact version, "latest", or semver constraint)
- Module type must be supported (currently only "go")

**Success Criteria:**
- ✅ Types properly defined and documented
- ✅ Configuration parsing handles group-based structure
- ✅ Validation catches all error cases
- ✅ Tests cover all configuration scenarios
- ✅ Examples updated to show group-based structure


### Phase 2: Module Version Resolver Implementation
**Objective**: Implement module version detection and resolution system

**Implementation Steps:**
1. Create module detector to identify Go modules in directories
2. Implement version resolver for git tags and semantic versioning
3. Add version constraint evaluation (exact, latest, semver ranges)
4. Create version cache to optimize API calls
5. Implement module metadata extraction from go.mod files

**Files to Create/Modify:**
- `internal/sync/module_detector.go` - Detect and analyze Go modules
- `internal/sync/module_resolver.go` - Resolve module versions from git
- `internal/sync/module_cache.go` - Cache version lookups
- `internal/sync/semver.go` - Semantic version constraint handling
- `internal/sync/module_test.go` - Module functionality tests

**Module Detector Implementation:**
```go
// ModuleDetector identifies and analyzes modules in directories
type ModuleDetector struct {
    logger *logrus.Entry
}

// DetectModule checks if a directory contains a Go module
func (md *ModuleDetector) DetectModule(path string) (*ModuleInfo, error) {
    goModPath := filepath.Join(path, "go.mod")
    if _, err := os.Stat(goModPath); err != nil {
        return nil, nil // Not a module
    }

    // Parse go.mod to extract module information
    modFile, err := modfile.Parse(goModPath, nil, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to parse go.mod: %w", err)
    }

    return &ModuleInfo{
        Path:    modFile.Module.Mod.Path,
        Version: modFile.Module.Mod.Version,
        GoMod:   goModPath,
    }, nil
}
```

**Version Resolver Implementation:**
```go
// ModuleResolver resolves module versions from git repositories
type ModuleResolver struct {
    client *github.Client
    cache  *ModuleCache
    logger *logrus.Entry
}

// ResolveVersion finds the appropriate version based on constraint
func (mr *ModuleResolver) ResolveVersion(ctx context.Context, repo string, constraint string) (string, error) {
    // Check cache first
    if cached, found := mr.cache.Get(repo, constraint); found {
        return cached, nil
    }

    switch constraint {
    case "latest":
        return mr.resolveLatestTag(ctx, repo)
    case "":
        return "", errors.New("version constraint required")
    default:
        if strings.HasPrefix(constraint, "^") || strings.HasPrefix(constraint, "~") {
            return mr.resolveSemverConstraint(ctx, repo, constraint)
        }
        // Exact version
        return mr.validateTag(ctx, repo, constraint)
    }
}

// resolveSemverConstraint finds best matching version for semver constraint
func (mr *ModuleResolver) resolveSemverConstraint(ctx context.Context, repo string, constraint string) (string, error) {
    tags, err := mr.listTags(ctx, repo)
    if err != nil {
        return "", err
    }

    // Parse constraint and find best match
    c, err := semver.NewConstraint(constraint)
    if err != nil {
        return "", fmt.Errorf("invalid constraint %s: %w", constraint, err)
    }

    var bestVersion *semver.Version
    for _, tag := range tags {
        v, err := semver.NewVersion(tag)
        if err != nil {
            continue // Skip non-semver tags
        }

        if c.Check(v) && (bestVersion == nil || v.GreaterThan(bestVersion)) {
            bestVersion = v
        }
    }

    if bestVersion == nil {
        return "", fmt.Errorf("no version matches constraint %s", constraint)
    }

    return "v" + bestVersion.String(), nil
}
```

**Success Criteria:**
- ✅ Module detection works for Go modules
- ✅ Version resolver handles exact, latest, and semver constraints
- ✅ Git tags are efficiently retrieved and cached
- ✅ Semantic version matching works correctly
- ✅ API calls are minimized through caching


### Phase 3: Sync Engine Implementation
**Objective**: Implement sync engine with group-based execution and module awareness

**Implementation Steps:**
1. Create group orchestrator with dependency resolution
2. Implement topological sort for dependency ordering
3. Integrate module detection into directory processing
4. Add module-aware sync logic that checks versions before syncing
5. Refactor sync engine to process groups based on dependencies and priority
6. Add group-level state tracking with dependency status
7. Implement group-level error handling that respects dependencies
8. Add progress reporting per group with dependency chain visibility

**Files to Create/Modify:**
- `internal/sync/orchestrator.go` - New group orchestrator
- `internal/sync/engine.go` - Refactor for group support
- `internal/sync/repository.go` - Update for group context
- `internal/state/types.go` - Add group tracking
- `internal/sync/progress.go` - Group-level progress

**Group Orchestrator Implementation:**
```go
// GroupOrchestrator manages execution of multiple sync groups
type GroupOrchestrator struct {
    config        *config.Config
    engine        *Engine
    logger        *logrus.Logger
    groupStatus   map[string]GroupStatus  // Track group execution status
}

type GroupStatus struct {
    State     string    // pending, running, success, failed, skipped
    StartTime time.Time
    EndTime   time.Time
    Error     error
}

// ExecuteGroups runs all enabled groups respecting dependencies and priority
func (go *GroupOrchestrator) ExecuteGroups(ctx context.Context) error {
    // Resolve dependencies and get execution order
    executionOrder, err := go.resolveDependencies()
    if err != nil {
        return fmt.Errorf("failed to resolve dependencies: %w", err)
    }

    // Initialize group status tracking
    go.initializeGroupStatus()

    // Execute groups in resolved order
    for _, group := range executionOrder {
        // Check if dependencies completed successfully
        if !go.areDependenciesSatisfied(group) {
            go.logger.WithField("group_id", group.ID).Info("Skipping group due to failed dependencies")
            go.groupStatus[group.ID] = GroupStatus{State: "skipped"}
            continue
        }

        go.logger.WithFields(logrus.Fields{
            "group_name": group.Name,
            "group_id":   group.ID,
            "priority":   group.Priority,
            "depends_on": group.DependsOn,
        }).Info("Starting group sync")

        go.groupStatus[group.ID] = GroupStatus{State: "running", StartTime: time.Now()}

        if err := go.executeGroup(ctx, group); err != nil {
            go.groupStatus[group.ID].State = "failed"
            go.groupStatus[group.ID].Error = err
            go.logger.WithError(err).Error("Group sync failed")
            // Continue with groups that don't depend on this one
        } else {
            go.groupStatus[group.ID].State = "success"
        }

        go.groupStatus[group.ID].EndTime = time.Now()
    }

    return go.reportFinalStatus()
}

// resolveDependencies performs topological sort considering both dependencies and priority
func (go *GroupOrchestrator) resolveDependencies() ([]config.Group, error) {
    // Build dependency graph
    graph := go.buildDependencyGraph()

    // Check for circular dependencies
    if err := go.detectCircularDependencies(graph); err != nil {
        return nil, err
    }

    // Perform topological sort with priority consideration
    return go.topologicalSort(graph)
}

// executeGroup processes a single group
func (go *GroupOrchestrator) executeGroup(ctx context.Context, group config.Group) error {
    // Create group-specific engine instance
    groupEngine := go.engine.ForGroup(group)

    // Clone source repository for this group
    if err := groupEngine.cloneSource(ctx); err != nil {
        return fmt.Errorf("failed to clone source for group %s: %w", group.ID, err)
    }

    // Process all targets in the group
    return groupEngine.processTargets(ctx)
}

// processDirectory handles directory sync with module awareness
func (rs *RepositorySync) processDirectory(ctx context.Context, dirMapping config.DirectoryMapping) ([]FileChange, error) {
    sourcePath := filepath.Join(rs.tempDir, "source", dirMapping.Src)

    // Check if this is a module-aware sync
    if dirMapping.Module != nil {
        return rs.processModuleDirectory(ctx, dirMapping, sourcePath)
    }

    // Regular directory processing
    return rs.processRegularDirectory(ctx, dirMapping, sourcePath)
}

// processModuleDirectory handles module-aware directory sync
func (rs *RepositorySync) processModuleDirectory(ctx context.Context, dirMapping config.DirectoryMapping, sourcePath string) ([]FileChange, error) {
    // Detect module in source directory
    moduleInfo, err := rs.moduleDetector.DetectModule(sourcePath)
    if err != nil {
        return nil, fmt.Errorf("failed to detect module: %w", err)
    }

    if moduleInfo == nil && dirMapping.Module.Type == "go" {
        return nil, fmt.Errorf("directory %s is not a Go module but module sync requested", dirMapping.Src)
    }

    // Resolve target version based on constraint
    targetVersion, err := rs.moduleResolver.ResolveVersion(ctx, rs.source.Repo, dirMapping.Module.Version)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve version %s: %w", dirMapping.Module.Version, err)
    }

    // Check if target already has this version
    if rs.isModuleVersionCurrent(ctx, dirMapping.Dest, targetVersion) {
        rs.logger.WithFields(logrus.Fields{
            "module":  dirMapping.Src,
            "version": targetVersion,
        }).Info("Module already at target version, skipping")
        return nil, nil
    }

    // Proceed with sync
    rs.logger.WithFields(logrus.Fields{
        "module":         dirMapping.Src,
        "target_version": targetVersion,
    }).Info("Syncing module to version")

    return rs.processRegularDirectory(ctx, dirMapping, sourcePath)
}
```

**State Tracking Enhancement:**
Branch naming pattern for groups:
```
chore/sync-files-{group_id}-20250130-143052-abc123f
```

Where:
- `{group_id}` - Unique identifier from the group configuration
- `20250130-143052` - Timestamp (YYYYMMDD-HHMMSS)
- `abc123f` - Source commit SHA (7 chars)

This maintains stateless tracking while identifying which group performed the sync.

**Success Criteria:**
- ✅ Groups execute respecting both dependencies and priority
- ✅ Dependency resolution with topological sort
- ✅ Circular dependency detection and prevention
- ✅ Disabled groups are skipped
- ✅ Groups with failed dependencies are skipped
- ✅ Each group has isolated execution context
- ✅ Group-level errors don't block unrelated groups
- ✅ Progress reported per group with dependency status
- ✅ State tracking includes group and dependency information


### Phase 4: Command Interface Updates
**Objective**: Update CLI commands to support group operations and module features

**Implementation Steps:**
1. Add group filtering to sync command
2. Add group listing to status command
3. Update validate command for groups and modules
4. Add group-specific dry-run
5. Enhance cancel command for groups
6. Add module-specific commands and flags
7. Add module version checking command

**Files to Modify:**
- `cmd/go-broadcast/sync.go` - Add group and module flags
- `cmd/go-broadcast/status.go` - Show group and module status
- `cmd/go-broadcast/validate.go` - Validate groups and modules
- `cmd/go-broadcast/cancel.go` - Cancel by group
- `cmd/go-broadcast/modules.go` - New module commands

**New Command Options:**
```bash
# Sync specific groups only
go-broadcast sync --groups "core-infra,security" --config sync.yaml

# Sync single group
go-broadcast sync --group-id "core-infra" --config sync.yaml

# List all groups and their status
go-broadcast status --show-groups --config sync.yaml

# Dry run for specific group
go-broadcast sync --dry-run --group-id "security" --config sync.yaml

# Cancel sync for specific group
go-broadcast cancel --group-id "core-infra" --config sync.yaml

# Check module versions without syncing
go-broadcast modules check --config sync.yaml

# List all modules and their versions
go-broadcast modules list --config sync.yaml

# Update module version constraints
go-broadcast modules update --module "github.com/pkg/errors" --version "^0.9.0"
```

**Module Command Implementation:**
```go
// cmd/go-broadcast/modules.go
type ModulesCmd struct {
    Config string `help:"Path to config file" required:""`
}

type ModulesCheckCmd struct {
    GroupID string `help:"Check modules for specific group"`
    All     bool   `help:"Check all modules across all groups"`
}

func (m *ModulesCheckCmd) Run(ctx *Context) error {
    // Load configuration
    config, err := loadConfig(m.Config)
    if err != nil {
        return err
    }

    // Initialize module resolver
    resolver := sync.NewModuleResolver(ctx.githubClient, cache, logger)
    detector := sync.NewModuleDetector(logger)

    // Check modules for each group
    for _, group := range config.Group {
        if m.GroupID != "" && group.ID != m.GroupID {
            continue
        }

        logger.WithField("group", group.Name).Info("Checking modules")

        for _, target := range group.Targets {
            for _, dir := range target.Directories {
                if dir.Module != nil {
                    // Check if source is a module
                    moduleInfo, err := detector.DetectModule(dir.Src)
                    if err != nil {
                        logger.WithError(err).Error("Failed to detect module")
                        continue
                    }

                    if moduleInfo != nil {
                        // Resolve version
                        version, err := resolver.ResolveVersion(ctx, group.Source.Repo, dir.Module.Version)
                        if err != nil {
                            logger.WithError(err).Error("Failed to resolve version")
                            continue
                        }

                        logger.WithFields(logrus.Fields{
                            "module":    moduleInfo.Path,
                            "version":   version,
                            "directory": dir.Src,
                        }).Info("Module version resolved")
                    }
                }
            }
        }
    }

    return nil
}
```

**Success Criteria:**
- ✅ Commands support group filtering
- ✅ Status shows group information
- ✅ Module commands work correctly
- ✅ Version checking without sync
- ✅ Validation reports group-specific and module issues
- ✅ Help text updated with examples
- ✅ Clean command interface for group and module operations


### Phase 5: State Discovery Implementation
**Objective**: Implement state discovery for group-based configurations with module tracking

**Implementation Steps:**
1. Update branch naming pattern to include group ID
2. Enhance PR metadata generation with group and module information
3. Update PR body format with group and module details
4. Implement group-aware state comparison
5. Add group execution history tracking
6. Track module versions in PR metadata
7. Add module update detection in state discovery

**Files to Modify:**
- `internal/state/discovery.go` - Group-aware discovery
- `internal/state/parser.go` - Parse group metadata from PRs
- `internal/state/types.go` - Add group state types
- `internal/sync/metadata.go` - Generate enhanced PR metadata
- `internal/sync/pr_body.go` - Format PR body with group information

**Enhanced PR Metadata Format:**
The PR body will include human-readable content followed by machine-parseable metadata:

```markdown
## What Changed
* Synchronized files for group "Core Infrastructure" (core-infra)
* Updated 3 individual file(s) and 61 file(s) from directory mappings
* Applied transformations based on group configuration

## Group Information
**Group**: Core Infrastructure (`core-infra`)
**Priority**: 1 (executed 1 of 3 groups)
**Dependencies**: None
**Source**: company/infrastructure-templates @ abc123f

## Directory Synchronization Details
### `.github/coverage` → `.github/coverage`
* **Files synced**: 61
* **Files excluded**: 26
* **Processing time**: 1523ms
* **Modules detected**: 1 (github.com/company/utils v1.2.3)
* **Exclusion patterns**: `*.out`, `*.test`, `gofortress-coverage`

## Performance Metrics
* **Group execution time**: 2.1s
* **Total files processed**: 87
* **API calls saved**: 72
* **Cache hit rate**: 62.5%

<!-- go-broadcast metadata
config:
  name: "Platform Repository Sync"
  id: "platform-sync-2025"
groups:
  name: "Core Infrastructure"
  id: "core-infra"
  description: "Syncs core CI/CD and build infrastructure"
  priority: 1
  depends_on: []
  execution_order: "1 of 3"
source:
  repo: company/infrastructure-templates
  branch: main
  commit: abc123f7890
files:
  - src: .github/workflows/ci.yml
    dest: .github/workflows/ci.yml
directories:
  - src: .github/coverage
    dest: .github/coverage
    excluded: ["*.out", "*.test", "gofortress-coverage"]
    files_synced: 61
    files_excluded: 26
    processing_time_ms: 1523
  - src: pkg/utils
    dest: pkg/utils
    module:
      type: go
      version: "^1.2.0"
      resolved_version: "v1.2.3"
      check_tags: true
    files_synced: 12
    processing_time_ms: 450
modules:
  - path: "github.com/company/utils"
    source_version: "v1.2.3"
    target_version: "v1.1.0"
    updated: true
performance:
  group_execution_time_ms: 2145
  total_files: 87
  api_calls_saved: 72
  cache_hits: 45
  module_version_checks: 3
  module_cache_hits: 2
timestamp: 2025-01-30T14:30:52Z
-->
```

**State Discovery Components:**
1. **Branch Pattern**: `chore/sync-files-{group_id}-{timestamp}-{commit}`
2. **PR Title**: `[Sync] {Group Name} - Update from {source_repo} ({commit})`
3. **PR Metadata**: Complete group information in YAML format
4. **State Tracking**: Group-specific sync history and status

**Success Criteria:**
- ✅ Branch names include group ID for identification
- ✅ PR metadata includes complete group information
- ✅ PR body shows human-readable group details
- ✅ State discovery can identify group from branch/PR
- ✅ Status command shows per-group state
- ✅ History tracking works per group
- ✅ No conflicts between groups


### Phase 6: Integration Testing
**Objective**: Comprehensive testing of group-based and module functionality

**Implementation Steps:**
1. Create integration tests for multi-group sync
2. Test priority-based execution
3. Test dependency resolution and execution order
4. Test circular dependency detection
5. Test enable/disable functionality
6. Test group isolation and failure propagation
7. Test module detection and version resolution
8. Test module-aware directory sync
9. Performance testing with multiple groups and modules

**Files to Create/Modify:**
- `test/integration/multi_group_test.go` - Group tests
- `test/integration/priority_test.go` - Priority tests
- `test/integration/group_state_test.go` - State tests
- `test/integration/module_sync_test.go` - Module tests
- `internal/sync/orchestrator_test.go` - Unit tests
- `internal/sync/module_resolver_test.go` - Module resolver tests

**Test Scenarios:**
```go
func TestMultiGroupExecution(t *testing.T) {
    // Test multiple groups execute in priority order
    // Test disabled groups are skipped
    // Test group isolation (source repos don't conflict)
}

func TestGroupDependencyExecution(t *testing.T) {
    // Test groups execute in dependency order
    // Test groups with both dependencies and priority
    // Test skipping groups when dependencies fail
    // Test independent groups continue when others fail
}

func TestCircularDependencies(t *testing.T) {
    // Test detection of direct circular dependencies (A→B→A)
    // Test detection of indirect circular dependencies (A→B→C→A)
    // Test validation fails with clear error message
}

func TestGroupPriorityExecution(t *testing.T) {
    // Test groups with priority 1, 2, 3 execute in order
    // Test groups with same priority but different dependencies
    // Test priority 0 executes first (unless dependencies)
}

func TestGroupStateIsolation(t *testing.T) {
    // Test each group gets its own branch
    // Test group PRs are independent
    // Test one group failure doesn't affect unrelated groups
    // Test dependent groups are skipped on failure
}

func TestModuleSync(t *testing.T) {
    // Test module detection in source directories
    // Test version resolution from git tags
    // Test semantic version constraint matching
    // Test module sync skips when version is current
    // Test module sync updates when version differs
}

func TestModuleVersionConstraints(t *testing.T) {
    // Test exact version matching (v1.2.3)
    // Test latest version resolution
    // Test caret constraints (^1.2.0)
    // Test tilde constraints (~1.2.0)
    // Test invalid constraint handling
}

func TestModulePerformance(t *testing.T) {
    // Test version cache effectiveness
    // Test API call minimization
    // Test concurrent module resolution
    // Test large directory module detection
}
```

**Success Criteria:**
- ✅ All integration tests pass
- ✅ Priority execution verified
- ✅ Group isolation confirmed
- ✅ Performance acceptable with 10+ groups
- ✅ CI/CD pipeline updated


### Phase 7: Documentation Revision
**Objective**: Revise all existing documentation to reflect group-based configuration as the standard way go-broadcast works

**Important Context**: go-broadcast is unreleased software. This phase is about updating existing documentation to present group-based configuration as the only way the system works, not as a "new feature".

**Implementation Steps:**
1. Revise README.md to present group-based configuration as the standard
2. Update configuration guide to show only group-based examples
3. Remove any references to single-source configuration
4. Revise all example files to use group structure
5. Update CLAUDE.md to reflect current system behavior
6. Ensure no documentation implies this is a "migration" or "new feature"

**Files to Revise:**
- `README.md` - Complete revision showing group-based config as standard
- `docs/configuration-guide.md` - Update to show only group-based structure
- `docs/directory-sync.md` - Add module-aware sync as built-in capability
- `docs/module-sync.md` - New file documenting module functionality
- `examples/*.yaml` - Convert ALL examples to group format
- `.github/CLAUDE.md` - Update development instructions for group-based system

**Documentation Approach:**
1. **Present as Standard**: Write as if groups have always been the way go-broadcast works
2. **Remove Legacy References**: No mentions of "old" or "previous" config formats
3. **Clear Examples**: Show group configuration in all examples
4. **Module Integration**: Present module-aware sync as a core feature
5. **Consistent Terminology**: Use "groups" throughout, not "targets"

**Key Documentation Updates:**
```markdown
# README.md Example Update

## Configuration

go-broadcast uses a group-based configuration structure where each group
defines its own source repository and target mappings:

```yaml
version: 1
name: "My Repository Sync"
groups:
  - name: "Core Templates"
    id: "core"
    source:
      repo: "org/templates"
      branch: "main"
    targets:
      - repo: "org/service-a"
        files:
          - src: "Makefile"
            dest: "Makefile"
```

Each group executes independently with its own source repository...
```

**Success Criteria:**
- ✅ All documentation presents group-based config as the standard
- ✅ No references to "migration" or "old formats"
- ✅ Examples consistently use group structure
- ✅ Module features documented as built-in capabilities
- ✅ Clear, cohesive documentation that reads as if written from scratch
- ✅ CLAUDE.md updated with group-based development workflow


## Configuration Examples

### Basic Group Configuration
```yaml
version: 1
name: "Company-wide Repository Sync"
id: "company-sync"
groups:
  - name: "Core Infrastructure"
    id: "core"
    priority: 1
    enabled: true
    source:
      repo: "company/infra-templates"
      branch: "main"
    targets:
      - repo: "company/service-a"
        files:
          - src: ".github/workflows/ci.yml"
            dest: ".github/workflows/ci.yml"

  - name: "Documentation Standards"
    id: "docs"
    priority: 2
    enabled: true
    source:
      repo: "company/doc-templates"
      branch: "main"
    targets:
      - repo: "company/service-a"
        directories:
          - src: "docs/templates"
            dest: "docs"
```

### Enterprise Multi-Team Configuration
```yaml
version: 1
name: "Enterprise Platform Sync"
id: "enterprise-platform-2025"
groups:
  # Platform team templates (highest priority)
  - name: "Platform Team Standards"
    id: "platform"
    description: "Core platform team standards and tooling"
    priority: 1
    enabled: true
    source:
      repo: "platform/templates"
      branch: "stable"
    global:
      pr_labels: ["platform", "automated-sync"]
      pr_assignees: ["platform-team"]
    targets:
      - repo: "company/service-a"
        files:
          - src: "Makefile"
            dest: "Makefile"
        directories:
          - src: ".github"
            dest: ".github"
            exclude: ["workflows/experimental-*"]

  # Security team templates
  - name: "Security Compliance"
    id: "security"
    description: "Security policies and scanning configurations"
    priority: 2
    enabled: true
    source:
      repo: "security/policies"
      branch: "main"
    global:
      pr_labels: ["security", "compliance"]
      pr_reviewers: ["security-team"]
    targets:
      - repo: "company/service-a"
        files:
          - src: ".gitleaks.toml"
            dest: ".gitleaks.toml"
          - src: "SECURITY.md"
            dest: "SECURITY.md"

  # QA team templates (depends on security being in place)
  - name: "QA Standards"
    id: "qa"
    description: "Testing frameworks and QA configurations"
    priority: 3
    enabled: true
    depends_on: ["security"]  # QA configs need security policies first
    source:
      repo: "qa/templates"
      branch: "main"
    targets:
      - repo: "company/service-a"
        directories:
          - src: "test/templates"
            dest: "test"
```

### Groups with Dependencies
```yaml
version: 1
name: "Application Deployment Pipeline"
groups:
  # Base configuration must be applied first
  - name: "Base Configuration"
    id: "base-config"
    priority: 1
    enabled: true
    source:
      repo: "company/base-templates"
      branch: "main"
    targets:
      - repo: "company/app-service"
        files:
          - src: "Dockerfile.base"
            dest: "Dockerfile"

  # Application code depends on base configuration
  - name: "Application Code"
    id: "app-code"
    priority: 2
    enabled: true
    depends_on: ["base-config"]  # Needs base Dockerfile first
    source:
      repo: "company/app-templates"
      branch: "main"
    targets:
      - repo: "company/app-service"
        directories:
          - src: "src"
            dest: "src"

  # Tests depend on application code being in place
  - name: "Test Suite"
    id: "tests"
    priority: 3
    enabled: true
    depends_on: ["app-code"]  # Needs application code first
    source:
      repo: "company/test-templates"
      branch: "main"
    targets:
      - repo: "company/app-service"
        directories:
          - src: "tests"
            dest: "tests"
```

### Module-Aware Configuration
```yaml
version: 1
name: "Module-Aware Repository Sync"
groups:
  - name: "Shared Libraries"
    id: "shared-libs"
    description: "Sync shared Go modules with version management"
    priority: 1
    enabled: true
    source:
      repo: "company/go-modules"
      branch: "main"
    targets:
      - repo: "company/service-a"
        directories:
          # Sync specific version of error handling module
          - src: "pkg/errors"
            dest: "vendor/github.com/company/errors"
            module:
              type: "go"
              version: "v1.2.3"  # Exact version
              check_tags: true

          # Sync latest stable version of utils
          - src: "pkg/utils"
            dest: "vendor/github.com/company/utils"
            module:
              type: "go"
              version: "latest"
              check_tags: true

          # Sync with semantic version constraint
          - src: "pkg/logger"
            dest: "vendor/github.com/company/logger"
            module:
              type: "go"
              version: "^2.1.0"  # Any 2.x version >= 2.1.0
              check_tags: true
              update_refs: true  # Update import paths in go.mod

  - name: "Service Libraries"
    id: "service-libs"
    description: "Service-specific module synchronization"
    priority: 2
    enabled: true
    depends_on: ["shared-libs"]
    source:
      repo: "company/service-modules"
      branch: "main"
    targets:
      - repo: "company/service-a"
        directories:
          # Sync with tilde constraint (patch versions only)
          - src: "internal/auth"
            dest: "internal/auth"
            module:
              type: "go"
              version: "~1.5.2"  # 1.5.2 <= version < 1.6.0
              check_tags: true
```

### Enterprise Module Management
```yaml
version: 1
name: "Enterprise Module Distribution"
id: "enterprise-modules-2025"
groups:
  - name: "Core Modules"
    id: "core-modules"
    description: "Company-wide core Go modules"
    priority: 1
    enabled: true
    source:
      repo: "company/core-modules"
      branch: "stable"
    global:
      pr_labels: ["module-update", "automated"]
    targets:
      # Update multiple services with same module versions
      - repo: "company/api-gateway"
        directories:
          - src: "modules/authentication"
            dest: "pkg/auth"
            module:
              type: "go"
              version: "v3.0.0"
              check_tags: true

      - repo: "company/user-service"
        directories:
          - src: "modules/authentication"
            dest: "pkg/auth"
            module:
              type: "go"
              version: "v3.0.0"
              check_tags: true
```

## Implementation Timeline

- **Session 1**: Configuration Types (Phase 1) - 3-4 hours
  - Define group and module types
  - Implement validation logic
  - Create parser updates

- **Session 2**: Module Version Resolver (Phase 2) - 4-5 hours
  - Implement module detector
  - Create version resolver with git integration
  - Build caching system
  - Handle semantic versioning

- **Session 3**: Sync Engine Refactoring (Phase 3) - 4-5 hours
  - Create group orchestrator
  - Implement dependency resolution
  - Integrate module-aware sync
  - Add group state tracking

- **Session 4**: Command Interface (Phase 4) - 3-4 hours
  - Add group commands
  - Implement module commands
  - Update existing commands

- **Session 5**: State Discovery (Phase 5) - 3-4 hours
  - Enhanced PR metadata
  - Module version tracking
  - Group state management

- **Session 6**: Integration Testing (Phase 6) - 4-5 hours
  - Group functionality tests
  - Module sync tests
  - Performance validation

- **Session 7**: Documentation & Examples (Phase 7) - 3-4 hours
  - Complete documentation
  - Create examples
  - Best practices guide

Total estimated time: 24-31 hours across 7 focused sessions

## Success Metrics

### Functionality
- **Group Support**: Groups execute successfully
- **Priority Execution**: Groups execute in defined order
- **Dependency Resolution**: Groups respect dependencies
- **Enable/Disable**: Groups can be toggled without removal
- **Group Isolation**: No interference between groups
- **State Tracking**: Complete audit trail per group
- **Module Detection**: Go modules automatically detected
- **Version Resolution**: Correct versions resolved from tags
- **Smart Sync**: Modules skip when version is current

### Performance
- **Execution Time**: Linear with number of groups
- **Memory Usage**: Isolated per group execution
- **API Efficiency**: No additional API calls per group
- **Module Caching**: 90%+ cache hit rate for versions
- **Concurrent Processing**: Within groups maintained
- **File Comparison**: Avoided for unchanged modules

### Developer Experience
- **Configuration**: Intuitive group and module structure
- **Simplicity**: Single, clear configuration format
- **Debugging**: Group and module-specific logging
- **Flexibility**: Support various organizational patterns
- **Version Control**: Clear semantic versioning support

## Risk Mitigation

### Technical Risks
- **Complexity**: Phased implementation with thorough testing
- **State Conflicts**: Unique branch names per group
- **Performance**: Sequential group execution by design
- **Error Handling**: Group failures isolated

### Adoption Risks
- **Learning Curve**: Comprehensive documentation
- **Configuration Understanding**: Clear examples and guides
- **Configuration Size**: Groups reduce redundancy

## Module Version Resolution Examples

### Version Constraint Behavior
```yaml
# Exact version - must match exactly
module:
  version: "v1.2.3"  # Only v1.2.3 will be used

# Latest version - highest available tag
module:
  version: "latest"  # Resolves to newest tag (e.g., v2.5.1)

# Caret constraint - compatible versions
module:
  version: "^1.2.3"  # Allows 1.2.3, 1.2.4, 1.3.0, but not 2.0.0

# Tilde constraint - patch versions only
module:
  version: "~1.2.3"  # Allows 1.2.3, 1.2.4, but not 1.3.0

# Major version constraint
module:
  version: "^2.0.0"  # Any 2.x.x version
```

### Module Sync Workflow
1. **Detection**: Check if source directory contains go.mod
2. **Resolution**: Find appropriate version from git tags
3. **Comparison**: Check if target already has this version
4. **Decision**: Skip if current, sync if different
5. **Update**: Optionally update go.mod references

## Future Enhancements

After initial implementation (DO NOT implement yet), consider the following enhancements:

### Future Features
- Support for other module systems (npm, Python)
- Conditional groups (run only if condition met)
- Parallel group execution (opt-in)
- Group-level rollback capability
- Group-level metrics and monitoring
- Scheduled group syncs (cron-like)

## Conclusion

This implementation plan establishes go-broadcast's core architecture with group-based configuration as its foundation. Since go-broadcast is unreleased software, this is not a migration or enhancement - this IS how go-broadcast works. The implementation provides:

- **Organizational Flexibility**: Groups align with team structures
- **Operational Control**: Priority and enable/disable features
- **Clear Semantics**: Named groups with descriptions
- **Scalability**: Handles complex enterprise scenarios
- **Maintainability**: Self-documenting configurations
- **Module Intelligence**: Built-in Go module version management

The group-based structure with module awareness defines go-broadcast as an enterprise-ready solution for managing repository synchronization at scale across complex organizational structures. All documentation and examples will present this as the standard and only way to configure go-broadcast.
