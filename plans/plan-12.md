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

## Implementation Strategy

This plan uses a gradual implementation approach to ensure:
- Tests continue passing at each phase
- Each phase is independently testable
- Easy rollback if issues arise
- Clear transition path for all code
- No breaking changes until everything is ready

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

Configuration Structure:
version: 1
name: "Platform Repository Sync"          # Optional top-level name
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

### Phase 0: Code Audit and Impact Analysis
**Objective**: Identify all code locations that will be affected by the configuration change
**Duration**: 2-3 hours

**Implementation Steps:**
1. Scan codebase for all references to `config.Source`
2. Scan codebase for all references to `config.Targets`
3. Identify all test files that create Config structs
4. Document all example YAML files
5. List all commands that use configuration
6. Identify integration points in sync engine

**Deliverables:**
- `plans/plan-12-audit.md` - Complete list of affected files and code locations
- Decision on implementation approach based on findings

**Success Criteria:**
- ✅ Complete inventory of all affected code
- ✅ Clear understanding of scope
- ✅ No surprises during implementation
- ✅ Final todo: Update the @plans/plan-12-status.md file with the results of the implementation, make sure success was hit

### Phase 1: Configuration Types with Compatibility
**Objective**: Add new types while maintaining compatibility with existing code
**Duration**: 3-4 hours

**Implementation Steps:**
1. Add new types (Config, Group, ModuleConfig) alongside existing ones
2. Add compatibility methods to Config type
3. Update DirectoryMapping with Module field
4. Add dependency management utilities
5. Create test helpers for both formats

**Files to Create/Modify:**
- `internal/config/types.go` - Add new types without removing old ones
- `internal/config/compatibility.go` - NEW: Compatibility layer
- `internal/config/types_test.go` - Tests for new types

**Type Definitions with Compatibility:**
```go
// Config represents the complete sync configuration
type Config struct {
    Version int       `yaml:"version"`          // Config version (1)
    Name    string    `yaml:"name,omitempty"`   // Optional config name
    ID      string    `yaml:"id,omitempty"`     // Optional config ID

    // Structure we are moving towards
    Groups  []Group   `yaml:"groups,omitempty"` // List of sync groups

    // Existing fields for compatibility during transition
    Source   SourceConfig   `yaml:"source,omitempty"`
    Global   GlobalConfig   `yaml:"global,omitempty"`
    Defaults DefaultConfig  `yaml:"defaults,omitempty"`
    Targets  []TargetConfig `yaml:"targets,omitempty"`
}

// GetGroups returns groups, converting from old format if needed (temporary function)
func (c *Config) GetGroups() []Group {
    if len(c.Groups) > 0 {
        return c.Groups
    }

    // Convert old format to group format for compatibility
    if c.Source.Repo != "" {
        return []Group{{
            Name:     "default",
            ID:       "default",
            Priority: 0,
            Enabled:  boolPtr(true),
            Source:   c.Source,
            Global:   c.Global,
            Defaults: c.Defaults,
            Targets:  c.Targets,
        }}
    }

    return nil
}

// IsGroupBased returns true if using new group format (temporary function)
func (c *Config) IsGroupBased() bool {
    return len(c.Groups) > 0
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

**Success Criteria:**
- ✅ New types defined with compatibility methods
- ✅ Existing tests still pass
- ✅ Both config formats can be loaded
- ✅ Clear path for code to use GetGroups()
- ✅ Final todo: Update the @plans/plan-12-status.md file with the results of the implementation, make sure success was hit

### Phase 2: Update Code to Use Compatibility Layer
**Objective**: Modify all code to work with both configuration formats
**Duration**: 4-5 hours

**Implementation Steps:**
1. Update sync engine to use config.GetGroups()
2. Update commands to use config.GetGroups()
3. Update validators to handle both formats
4. Update state discovery to work with groups
5. Ensure all tests pass with both formats

**Files to Modify:**
- `internal/sync/engine.go` - Use GetGroups() instead of direct access
- `internal/sync/repository.go` - Work with group context
- `cmd/go-broadcast/*.go` - Update all commands
- `internal/config/validator.go` - Validate both formats
- `internal/state/discovery.go` - Handle group-based branches

**Example Code Updates:**
```go
// Before (in engine.go):
func (e *Engine) Execute(ctx context.Context) error {
    // Clone source
    if err := e.cloneSource(ctx, e.config.Source); err != nil {
        return err
    }

    // Process targets
    for _, target := range e.config.Targets {
        // ...
    }
}

// After (using compatibility):
func (e *Engine) Execute(ctx context.Context) error {
    groups := e.config.GetGroups()

    // Create orchestrator
    orch := NewGroupOrchestrator(e.config, e, e.logger)

    // Execute groups
    return orch.ExecuteGroups(ctx, groups)
}
```

**Success Criteria:**
- ✅ All code uses GetGroups() method
- ✅ No direct access to Source/Targets fields
- ✅ All existing tests pass
- ✅ Both config formats work correctly
- ✅ Final todo: Update the @plans/plan-12-status.md file with the results of the implementation, make sure success was hit

### Phase 3: Add Group Orchestration
**Objective**: Implement group orchestrator with dependency resolution
**Duration**: 4-5 hours

**Implementation Steps:**
1. Create GroupOrchestrator
2. Implement dependency resolution
3. Add priority sorting
4. Implement group isolation
5. Add group-level error handling

**Files to Create:**
- `internal/sync/orchestrator.go` - Group orchestration logic
- `internal/sync/orchestrator_test.go` - Orchestrator tests
- `internal/sync/dependency.go` - Dependency resolution
- `internal/sync/dependency_test.go` - Dependency tests

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
func (o *GroupOrchestrator) ExecuteGroups(ctx context.Context, groups []config.Group) error {
    // Resolve dependencies and get execution order
    executionOrder, err := o.resolveDependencies(groups)
    if err != nil {
        return fmt.Errorf("failed to resolve dependencies: %w", err)
    }

    // Initialize group status tracking
    o.initializeGroupStatus(groups)

    // Execute groups in resolved order
    for _, group := range executionOrder {
        // Check if dependencies completed successfully
        if !o.areDependenciesSatisfied(group) {
            o.logger.WithField("group_id", group.ID).Info("Skipping group due to failed dependencies")
            o.groupStatus[group.ID] = GroupStatus{State: "skipped"}
            continue
        }

        o.logger.WithFields(logrus.Fields{
            "group_name": group.Name,
            "group_id":   group.ID,
            "priority":   group.Priority,
            "depends_on": group.DependsOn,
        }).Info("Starting group sync")

        o.groupStatus[group.ID] = GroupStatus{State: "running", StartTime: time.Now()}

        if err := o.executeGroup(ctx, group); err != nil {
            o.groupStatus[group.ID].State = "failed"
            o.groupStatus[group.ID].Error = err
            o.logger.WithError(err).Error("Group sync failed")
            // Continue with groups that don't depend on this one
        } else {
            o.groupStatus[group.ID].State = "success"
        }

        o.groupStatus[group.ID].EndTime = time.Now()
    }

    return o.reportFinalStatus()
}
```

**Success Criteria:**
- ✅ Groups execute in priority order
- ✅ Dependencies are respected
- ✅ Circular dependencies detected
- ✅ Group failures isolated
- ✅ All tests pass
- ✅ Final todo: Update the @plans/plan-12-status.md file with the results of the implementation, make sure success was hit

### Phase 4: Module Version Resolver
**Objective**: Implement module detection and version resolution
**Duration**: 4-5 hours

**Implementation Steps:**
1. Add semver dependency to go.mod
2. Create module detector
3. Implement version resolver
4. Add version caching
5. Integrate with directory sync

**Files to Create:**
- `internal/sync/module_detector.go` - Module detection logic
- `internal/sync/module_resolver.go` - Version resolution
- `internal/sync/module_cache.go` - Version caching
- `internal/sync/module_test.go` - Module tests

**Success Criteria:**
- ✅ Module detection works
- ✅ Version resolution from git tags
- ✅ Semantic version constraints work
- ✅ Caching reduces API calls
- ✅ Integration with directory sync
- ✅ Final todo: Update the @plans/plan-12-status.md file with the results of the implementation, make sure success was hit

### Phase 5: Command Interface Updates
**Objective**: Update commands to support group operations
**Duration**: 3-4 hours

**Implementation Steps:**
1. Add group filtering to sync command
2. Update status command for groups
3. Update validate command
4. Add module commands
5. Update help text

**Files to Modify:**
- `cmd/go-broadcast/sync.go` - Add group flags
- `cmd/go-broadcast/status.go` - Show group status
- `cmd/go-broadcast/validate.go` - Validate groups
- `cmd/go-broadcast/modules.go` - NEW: Module commands

**Success Criteria:**
- ✅ Commands work with groups
- ✅ Backward compatibility maintained
- ✅ Help text updated
- ✅ Module commands functional
- ✅ Final todo: Update the @plans/plan-12-status.md file with the results of the implementation, make sure success was hit

### Phase 6: Remove Compatibility Layer
**Objective**: Clean up code to use only group-based configuration
**Duration**: 3-4 hours

**Implementation Steps:**
1. Remove old fields from Config struct
2. Remove GetGroups() compatibility method
3. Update all example configurations
4. Update all test configurations
5. Clean up any remaining references

**Files to Modify:**
- `internal/config/types.go` - Remove old fields
- `internal/config/compatibility.go` - DELETE this file
- `examples/*.yaml` - Update to group format
- All test files with example configs

**Final Config Structure:**
```go
// Config represents the complete sync configuration
type Config struct {
    Version int       `yaml:"version"`          // Config version (1)
    Name    string    `yaml:"name,omitempty"`   // Optional config name
    ID      string    `yaml:"id,omitempty"`     // Optional config ID
    Groups  []Group   `yaml:"groups"`           // List of sync groups
}
```

**Success Criteria:**
- ✅ Only group-based config remains
- ✅ All tests pass
- ✅ All examples use groups
- ✅ No compatibility code remains
- ✅ Final todo: Update the @plans/plan-12-status.md file with the results of the implementation, make sure success was hit

### Phase 7: Integration Testing
**Objective**: Comprehensive testing of complete implementation
**Duration**: 4-5 hours

**Implementation Steps:**
1. Create integration tests for multi-group sync
2. Test priority and dependency execution
3. Test module sync functionality
4. Performance testing
5. Update CI/CD pipeline

**Files to Create:**
- `test/integration/multi_group_test.go` - Group tests
- `test/integration/module_sync_test.go` - Module tests
- `test/integration/performance_test.go` - Performance tests

**Success Criteria:**
- ✅ All integration tests pass
- ✅ Performance targets met
- ✅ CI/CD pipeline green
- ✅ Ready for production use
- ✅ Final todo: Update the @plans/plan-12-status.md file with the results of the implementation, make sure success was hit

### Phase 8: Documentation Update
**Objective**: Update all documentation to reflect group-based configuration
**Duration**: 3-4 hours

**Implementation Steps:**
1. Update README.md
2. Update configuration guide
3. Create module sync documentation
4. Update all examples
5. Update Claude commands and agents

**Files to Update:**
- `README.md` - Show group-based config as standard
- `docs/configuration-guide.md` - Group configuration guide
- `docs/module-sync.md` - NEW: Module sync guide
- `.claude/commands/*.md` - Update for groups
- `.claude/agents/*.md` - Update using meta-agent

**Success Criteria:**
- ✅ Documentation presents groups as standard
- ✅ No references to old formats
- ✅ Clear examples provided
- ✅ Module features documented
- ✅ Commands and agents updated
- ✅ Final todo: Update the @plans/plan-12-status.md file with the results of the implementation, make sure success was hit

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
          - src: "pkg/errors"
            dest: "vendor/github.com/company/errors"
            module:
              type: "go"
              version: "v1.2.3"  # Exact version
              check_tags: true
```

## Implementation Timeline

- **Phase 0**: Code Audit (2-3 hours)
- **Phase 1**: Configuration Types with Compatibility (3-4 hours)
- **Phase 2**: Update Code to Use Compatibility (4-5 hours)
- **Phase 3**: Add Group Orchestration (4-5 hours)
- **Phase 4**: Module Version Resolver (4-5 hours)
- **Phase 5**: Command Interface Updates (3-4 hours)
- **Phase 6**: Remove Compatibility Layer (3-4 hours)
- **Phase 7**: Integration Testing (4-5 hours)
- **Phase 8**: Documentation Update (3-4 hours)

Total estimated time: 28-37 hours across 9 phases

## Success Metrics

### Quality
- Tests pass at every phase
- No breaking changes until Phase 6
- Smooth transition path
- Clear rollback capability

### Functionality
- Group-based configuration works
- Dependencies resolved correctly
- Module sync operational
- All commands updated

### Performance
- Linear scaling with groups
- Module caching effective
- No performance regression

## Risk Mitigation

### Technical Risks
- **Compatibility Issues**: Addressed by gradual approach
- **Test Failures**: Each phase ensures tests pass
- **Integration Problems**: Compatibility layer provides safety

### Implementation Risks
- **Scope Creep**: Clear phase boundaries
- **Time Overruns**: Each phase is time-boxed
- **Quality Issues**: Tests at every phase

## Conclusion

This implementation plan establishes go-broadcast's core architecture with group-based configuration as its foundation. The gradual approach ensures a smooth transition while maintaining code quality and test coverage throughout. Since go-broadcast is unreleased software, the final result will present groups as the standard and only way to configure go-broadcast, with no traces of the implementation journey remaining in the codebase or documentation.
