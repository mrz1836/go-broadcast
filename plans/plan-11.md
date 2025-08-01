# go-broadcast Directory Sync Feature - Implementation Plan

## Executive Summary

This document outlines a comprehensive plan to add directory synchronization support to go-broadcast while maintaining its stateless, Git-based architecture. The feature will allow users to sync entire directories with support for transformations and exclusion patterns, significantly reducing configuration verbosity for projects with many files.

**Key Architecture Decisions**:
- **Stateless Operation**: Directory contents resolved dynamically at sync time
- **Backward Compatible**: Additive feature with no breaking changes
- **Transform Support**: All existing transformations work on directory files
- **Exclusion Patterns**: Gitignore-style patterns with smart defaults
- **Performance First**: Concurrent processing with batching and caching
- **Audit Trail**: Complete tracking via branch names and PR metadata
- **Smart Defaults**: Automatic exclusion of development artifacts (.out, .test, binaries)

## Vision Statement

Enhance go-broadcast with directory synchronization capabilities that embody the tool's core principles:
- **Stateless Architecture**: No stored state about directory contents
- **Git-Native**: All state tracked through branches and commits
- **Developer Friendly**: Intuitive configuration with sensible defaults
- **Performance Optimized**: Parallel processing for large directories
- **Fully Auditable**: Complete history in Git
- **Zero Breaking Changes**: Existing configurations continue to work

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                   go-broadcast Sync Engine                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────────────────┐               │
│  │ Config       │───▶│ Sync Engine              │               │
│  │ (files +     │    │                          │               │
│  │  directories)│    │ ├─ processFiles()        │               │
│  └──────────────┘    │ ├─ processDirectories()  │◀── NEW        │
│                      │ └─ processFile()         │               │
│                      └──────────┬───────────────┘               │
│                                 │                               │
│                                 ▼                               │
│  ┌─────────────────────────────────────────────┐                │
│  │             Directory Processor             │                │
│  │                                             │                │
│  │  ├─ Concurrent directory walker             │                │
│  │  ├─ Smart exclusion patterns + defaults     │                │
│  │  ├─ Batch file processing (10 concurrent)   │                │
│  │  ├─ Progress reporting (>50 files)          │                │
│  │  ├─ GitHub API optimization (tree API)      │                │
│  │  └─ Transform pipeline with worker pool     │                │
│  └─────────────────────────────────────────────┘                │
│                                                                 │
│  Performance Features:                                          │
│  - Smart caching for unchanged content                          │
│  - Batch API operations to avoid rate limits                    │
│                                                                 │
│                                                                 │
│  State Tracking (Unchanged):                                    │
│  - Branch: chore/sync-files-20250130-143052-abc123f             │
│  - PR Metadata: Complete audit trail with directory info        │
└─────────────────────────────────────────────────────────────────┘

Configuration Structure:
targets:
  - repo: "org/service"
    files:           # Existing
      - src: "Makefile"
        dest: "Makefile"
    directories:     # NEW
      - src: ".github/workflows"
        dest: ".github/workflows"
        exclude: ["*.tmp", "test-*"]
        transform:
          repo_name: true
```

## Implementation Roadmap

### Phase 1: Configuration Layer Enhancement
**Objective**: Extend configuration types to support directory mappings

**Implementation Agent**: Use go-expert-developer for all Go code implementation

**Implementation Steps:**
1. Add `DirectoryMapping` type to config package
2. Update `TargetConfig` with `Directories` field
3. Implement directory configuration validation
4. Add directory support to config parser
5. Create comprehensive tests for new configuration

**Files to Create/Modify:**
- `internal/config/types.go` - Add DirectoryMapping struct
- `internal/config/validator.go` - Add directory validation logic
- `internal/config/parser.go` - Update parsing for directories
- `internal/config/config_test.go` - Add directory config tests
- `examples/directory-sync.yaml` - New example configuration

**DirectoryMapping Structure:**
```go
// DirectoryMapping defines source to destination directory mapping
type DirectoryMapping struct {
    Src              string            `yaml:"src"`                          // Source directory path
    Dest             string            `yaml:"dest"`                         // Destination directory path
    Exclude          []string          `yaml:"exclude,omitempty"`            // Glob patterns to exclude
    Transform        Transform         `yaml:"transform,omitempty"`          // Apply to all files
    PreserveStructure bool             `yaml:"preserve_structure,omitempty"` // Keep nested structure (default: true)
    IncludeHidden    bool              `yaml:"include_hidden,omitempty"`     // Include hidden files (default: true)
}

// DefaultExclusions returns smart default exclusions for development artifacts
func DefaultExclusions() []string {
    return []string{
        "*.out",           // Coverage outputs
        "*.test",          // Go test binaries
        "*.exe",           // Windows executables
        "**/.DS_Store",    // macOS files
        "**/tmp/*",        // Temporary files
        "**/.git",         // Git directories
    }
}
```

**Verification Steps:**
```bash
# 1. Verify configuration parsing
go test ./internal/config/... -run TestDirectoryConfig

# 2. Validate example configuration
go-broadcast validate --config examples/directory-sync.yaml

# 3. Check validation errors
# Test invalid directory paths, circular references, etc.

# 4. Verify backward compatibility
go-broadcast validate --config sync.yaml  # Existing configs still work
```

**Success Criteria:**
- ✅ DirectoryMapping type properly defined
- ✅ Configuration parsing handles directories field
- ✅ Validation catches invalid directory configurations
- ✅ Existing configurations remain valid
- ✅ Tests cover all new functionality
- ✅ Final todo: Update the @plans/plan-11-status.md file with the results of the implementation, make sure success was hit


### Phase 2: Directory Processing Engine
**Objective**: Implement core directory traversal and file resolution with performance optimizations

**Implementation Agent**: Use go-expert-developer for all Go code implementation

**Implementation Steps:**
1. Create concurrent directory walker with worker pool
2. Implement gitignore-style pattern matching with caching
3. Add batch file processing for transformations
4. Implement progress reporting for large directories
5. Add GitHub API optimization with tree API

**Files to Create/Modify:**
- `internal/sync/directory.go` - New directory processing logic
- `internal/sync/exclusion.go` - Pattern matching for exclusions
- `internal/sync/batch.go` - Batch processing utilities
- `internal/sync/progress.go` - Progress reporting
- `internal/sync/directory_test.go` - Comprehensive tests
- `internal/sync/repository.go` - Integrate processDirectories()

**Core Implementation:**
```go
// processDirectories handles all directory mappings for a target with concurrency
func (rs *RepositorySync) processDirectories(ctx context.Context) ([]FileChange, error) {
    var allChanges []FileChange
    sourcePath := filepath.Join(rs.tempDir, "source")
    
    // Process directories concurrently
    var wg sync.WaitGroup
    changesCh := make(chan []FileChange, len(rs.target.Directories))
    errorsCh := make(chan error, len(rs.target.Directories))
    
    for _, dirMapping := range rs.target.Directories {
        wg.Add(1)
        go func(dm DirectoryMapping) {
            defer wg.Done()
            
            // Add progress reporting for large directories
            changes, err := rs.processDirectoryWithProgress(ctx, sourcePath, dm)
            if err != nil {
                errorsCh <- fmt.Errorf("failed to process directory %s: %w", dm.Src, err)
                return
            }
            changesCh <- changes
        }(dirMapping)
    }
    
    wg.Wait()
    close(changesCh)
    close(errorsCh)
    
    // Check for errors
    if err := <-errorsCh; err != nil {
        return nil, err
    }
    
    // Collect all changes
    for changes := range changesCh {
        allChanges = append(allChanges, changes...)
    }
    
    return allChanges, nil
}

// BatchProcessor handles concurrent file processing
type BatchProcessor struct {
    maxConcurrent int
    batchSize     int
}

func (bp *BatchProcessor) ProcessFiles(ctx context.Context, files []string, fn func(string) error) error {
    sem := make(chan struct{}, bp.maxConcurrent)
    var wg sync.WaitGroup
    
    for _, file := range files {
        wg.Add(1)
        go func(f string) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()
            
            if err := fn(f); err != nil {
                rs.logger.WithError(err).Errorf("Failed to process file: %s", f)
            }
        }(file)
    }
    
    wg.Wait()
    return nil
}
```

**Exclusion Pattern Matching:**
```go
// matchesExclusion checks if a file path matches any exclusion pattern
func matchesExclusion(path string, patterns []string) bool {
    for _, pattern := range patterns {
        matched, _ := filepath.Match(pattern, filepath.Base(path))
        if matched {
            return true
        }
        // Also check against full path for directory patterns
        matched, _ = filepath.Match(pattern, path)
        if matched {
            return true
        }
    }
    return false
}
```

**Verification Steps:**
```bash
# 1. Test directory traversal
go test ./internal/sync/... -run TestProcessDirectory

# 2. Test exclusion patterns
go test ./internal/sync/... -run TestExclusionPatterns

# 3. Benchmark directory processing
go test -bench=BenchmarkDirectoryWalk ./internal/sync/...

# 4. Test with large directory structures
# Create test directory with 1000+ files
```

**Success Criteria:**
- ✅ Directory walker correctly traverses source directories
- ✅ Exclusion patterns work with gitignore syntax
- ✅ Hidden files handled according to configuration
- ✅ Performance targets met:
  - < 500ms for directories with < 50 files
  - < 2s for .github/coverage (87 files)
  - < 5s for directories with 1000 files
- ✅ Progress reporting shows for directories > 50 files
- ✅ Batch processing reduces API calls by 80%
- ✅ Proper error handling and recovery
- ✅ Final todo: Update the @plans/plan-11-status.md file with the results of the implementation, make sure success was hit


### Phase 3: Transform Integration
**Objective**: Ensure transformations work correctly on directory files

**Implementation Agent**: Use go-expert-developer for all Go code implementation

**Implementation Steps:**
1. Apply transformations to each file in directory
2. Maintain transformation context per file
3. Handle binary file detection
4. Ensure transform errors don't fail entire directory
5. Add transform debugging support

**Files to Modify:**
- `internal/sync/directory.go` - Add transformation logic
- `internal/transform/context.go` - Enhance context for directories
- `internal/sync/directory_test.go` - Add transform tests

**Transform Application:**
```go
// applyDirectoryTransforms applies configured transforms to directory files
func (rs *RepositorySync) applyDirectoryTransforms(
    content []byte, 
    srcPath string,
    dirMapping DirectoryMapping,
) ([]byte, error) {
    if !dirMapping.Transform.RepoName && len(dirMapping.Transform.Variables) == 0 {
        return content, nil // No transforms configured
    }
    
    // Create transform context with directory awareness
    ctx := transform.Context{
        SourceRepo: rs.sourceState.Repo,
        TargetRepo: rs.target.Repo,
        FilePath:   srcPath,
        Variables:  dirMapping.Transform.Variables,
        IsFromDirectory: true,
        DirectoryMapping: dirMapping,
    }
    
    return rs.engine.transform.Transform(ctx, content, ctx)
}
```

**Verification Steps:**
```bash
# 1. Test repo_name transform on directory files
go test ./internal/sync/... -run TestDirectoryTransform

# 2. Test variable substitution
go test ./internal/sync/... -run TestDirectoryVariables

# 3. Verify binary files are skipped
go test ./internal/sync/... -run TestDirectoryBinarySkip

# 4. Test transform error handling
```

**Success Criteria:**
- ✅ All transforms work on directory files
- ✅ Binary files detected and handled appropriately
- ✅ Transform errors logged but don't fail directory
- ✅ Performance remains acceptable
- ✅ Transform context includes directory information
- ✅ Final todo: Update the @plans/plan-11-status.md file with the results of the implementation, make sure success was hit


### Phase 4: State Tracking & GitHub API Optimization
**Objective**: Extend state tracking and optimize GitHub API usage

**Implementation Agent**: Use go-expert-developer for all Go code implementation

**Implementation Steps:**
1. Enhance PR metadata to include directory information
2. Implement GitHub tree API for bulk file operations
3. Add content caching with TTL for unchanged files
4. Batch API calls for file existence checks
5. Maintain complete audit trail with performance metrics

**Files to Modify:**
- `internal/sync/repository.go` - Update metadata generation
- `internal/sync/github_api.go` - Add tree API support
- `internal/sync/cache.go` - Content caching layer
- `internal/state/types.go` - Add directory tracking fields
- `internal/state/pr.go` - Parse directory metadata from PRs

**GitHub API Optimization:**
```go
// BatchCheckFiles checks multiple files existence using GitHub tree API
func (rs *RepositorySync) batchCheckFiles(ctx context.Context, paths []string) (map[string]bool, error) {
    tree, err := rs.engine.gh.GetTree(ctx, rs.target.Repo, "HEAD")
    if err != nil {
        return nil, err
    }
    
    // Build map for O(1) lookups
    exists := make(map[string]bool)
    for _, entry := range tree.Entries {
        exists[entry.Path] = true
    }
    
    result := make(map[string]bool)
    for _, path := range paths {
        result[path] = exists[path]
    }
    
    return result, nil
}
```

**Enhanced PR Metadata:**
```yaml
<!-- go-broadcast metadata
source:
  repo: company/template-repo
  branch: master
  commit: abc123f7890
files:
  - src: .github/workflows/ci.yml
    dest: .github/workflows/ci.yml
    from: file  # Individual file mapping
  - src: Makefile
    dest: Makefile
    from: file
directories:
  - src: .github/coverage
    dest: .github/coverage
    excluded: ["*.out", "*.test", "gofortress-coverage"]
    files_synced: 61
    files_excluded: 26
    processing_time_ms: 1523
performance:
  total_files: 87
  api_calls_saved: 72
  cache_hits: 45
timestamp: 2025-01-30T14:30:52Z
-->
```

**Verification Steps:**
```bash
# 1. Test metadata generation
go test ./internal/sync/... -run TestDirectoryMetadata

# 2. Verify state discovery
go test ./internal/state/... -run TestDirectoryStateDiscovery

# 3. Check PR descriptions
# Manually verify PR includes directory information

# 4. Test audit trail completeness
```

**Success Criteria:**
- ✅ PR metadata includes directory sync details with performance metrics
- ✅ GitHub tree API reduces API calls by 80%+
- ✅ Content caching works with 50%+ hit rate
- ✅ State discovery recognizes directory-synced files
- ✅ Complete audit trail maintained
- ✅ Branch names remain compatible
- ✅ PR descriptions clearly show directory operations
- ✅ Final todo: Update the @plans/plan-11-status.md file with the results of the implementation, make sure success was hit


### Phase 5: Integration Testing
**Objective**: Comprehensive testing of directory sync feature

**Implementation Agent**: Use go-expert-developer for all Go code implementation

**Implementation Steps:**
1. Create integration tests for directory sync
2. Test mixed file and directory configurations
3. Verify CI/CD compatibility
4. Performance testing with real repositories
5. Edge case handling

**Files to Create/Modify:**
- `test/integration/directory_sync_test.go` - New integration tests
- `internal/sync/benchmark_test.go` - Add directory benchmarks
- `.github/workflows/test.yml` - Ensure tests run in CI

**Integration Test Scenarios:**
```go
// Test mixed configuration with files and directories
func TestMixedFileAndDirectorySync(t *testing.T) {
    // Test configuration with both files and directories
    // Verify no conflicts or duplicates
    // Ensure proper precedence
}

// Test large directory sync
func TestLargeDirectorySync(t *testing.T) {
    // Create directory with 1000+ files
    // Verify performance and correctness
    // Check memory usage
}

// Test complex exclusion patterns
func TestComplexExclusionPatterns(t *testing.T) {
    // Test gitignore-style patterns
    // Verify nested directory exclusions
    // Test wildcard patterns
}
```

**Verification Steps:**
```bash
# 1. Run all integration tests
go test ./test/integration/... -v

# 2. Test with real repository
go-broadcast sync --dry-run --config examples/directory-sync.yaml

# 3. Performance benchmarks
go test -bench=. ./internal/sync/...

# 4. Memory profiling
go test -memprofile=mem.prof ./internal/sync/...

# 5. CI/CD verification
git push  # Trigger CI workflows
```

**Success Criteria:**
- ✅ All integration tests pass
- ✅ Performance targets achieved:
  - .github directory (149 files): ~2s with exclusions
  - .github/coverage (87 files): ~1.5s
  - Large test directory (1000 files): < 5s
- ✅ Memory usage linear with file count
- ✅ CI/CD workflows succeed
- ✅ Edge cases handled gracefully
- ✅ No GitHub API rate limit issues
- ✅ Final todo: Update the @plans/plan-11-status.md file with the results of the implementation, make sure success was hit


### Phase 6: Documentation & Examples
**Objective**: Comprehensive documentation for directory sync feature

**Implementation Agent**: Use go-expert-developer for all Go code implementation

**Important**: Present directory sync as an existing v1 feature. Do not use language suggesting this is "new" or "added". Integrate naturally into existing documentation.

**Implementation Steps:**
1. Update README with directory sync information
2. Create detailed examples for common use cases
3. Document exclusion pattern syntax
4. Add troubleshooting guide
5. Update configuration reference

**Files to Create/Modify:**
- `README.md` - Update with directory sync documentation
- `examples/directory-sync.yaml` - Comprehensive example
- `docs/directory-sync.md` - Detailed documentation
- `examples/README.md` - Update with directory examples

**Example Configurations:**
```yaml
# Sync entire workflows directory
targets:
  - repo: "company/service"
    directories:
      - src: ".github/workflows"
        dest: ".github/workflows"
        exclude: ["*-local.yml", "*.tmp"]

# Sync documentation with flattening
targets:
  - repo: "company/docs"
    directories:
      - src: "docs/api"
        dest: "api-docs"
        preserve_structure: false
        transform:
          variables:
            VERSION: "v2"

# Mixed file and directory sync
targets:
  - repo: "company/service"
    files:
      - src: "Makefile"
        dest: "Makefile"
    directories:
      - src: "configs"
        dest: "configs"
        exclude: ["*.local", "*.secret"]
        include_hidden: true
```

**Documentation Sections:**
1. **Quick Start**: Simple directory sync example
2. **Configuration Reference**: All DirectoryMapping options
3. **Exclusion Patterns**: Gitignore-style pattern guide
4. **Use Cases**: Common scenarios and solutions
5. **Performance**: Tips for large directories
6. **Troubleshooting**: Common issues and solutions

**Verification Steps:**
```bash
# 1. Test all examples
for example in examples/*.yaml; do
    go-broadcast validate --config "$example"
done

# 2. Verify documentation accuracy
# Manual review of all documentation

# 3. Test documented commands
# Run through quick start guide

# 4. Check for completeness
# Ensure all features documented
```

**Success Criteria:**
- ✅ Clear, comprehensive documentation
- ✅ Working examples for common use cases
- ✅ Exclusion patterns well documented
- ✅ Performance considerations included
- ✅ Troubleshooting covers common issues
- ✅ Final todo: Update the @plans/plan-11-status.md file with the results of the implementation, make sure success was hit


## Configuration Examples

### Basic Directory Sync
```yaml
version: 1
source:
  repo: "company/templates"
  branch: "main"
targets:
  - repo: "company/service"
    directories:
      - src: ".github"
        dest: ".github"
        # Default exclusions automatically applied:
        # *.out, *.test, *.exe, **/.DS_Store, **/tmp/*, **/.git
```

### Real-World Example: Syncing .github Directory
```yaml
targets:
  - repo: "company/service"
    directories:
      # Sync coverage module (87 files)
      - src: ".github/coverage"
        dest: ".github/coverage"
        exclude:
          - "*.out"              # Already in defaults
          - "gofortress-coverage" # Binary files
          - "**/testdata/*"      # Test fixtures
        transform:
          repo_name: true
      
      # Sync workflows separately
      - src: ".github/workflows"
        dest: ".github/workflows"
        exclude:
          - "*-local.yml"
          - "*.disabled"

### Advanced Directory Configuration
```yaml
targets:
  - repo: "company/service"
    directories:
      # Sync with progress reporting (>50 files)
      - src: ".github"
        dest: ".github"
        exclude:
          - "coverage/**"     # Synced separately above
          - "workflows/**"    # Synced separately above
          - "pip/**"         # Python dependencies
          - "*.toml"         # Tool configs
        preserve_structure: true
```

### Mixed Files and Directories
```yaml
targets:
  - repo: "company/service"
    # Individual files (existing)
    files:
      - src: "Makefile"
        dest: "Makefile"
      - src: "README.md"
        dest: "README.md"
    # Directories (new)
    directories:
      - src: ".github/workflows"
        dest: ".github/workflows"
      - src: "scripts"
        dest: "scripts"
        exclude: ["*.pyc", "__pycache__"]
```

## Implementation Timeline

- **Session 1**: Configuration Layer (Phase 1) - 2-3 hours
- **Session 2**: Directory Processing Engine (Phase 2) - 3-4 hours
- **Session 3**: Transform Integration (Phase 3) - 2-3 hours
- **Session 4**: State Tracking Enhancement (Phase 4) - 2-3 hours
- **Session 5**: Integration Testing (Phase 5) - 3-4 hours
- **Session 6**: Documentation & Examples (Phase 6) - 2-3 hours

Total estimated time: 14-20 hours across 6 focused sessions

## Success Metrics

### Functionality
- **Directory Sync**: Correctly syncs all files in directories
- **Exclusions**: Gitignore-style patterns with smart defaults
- **Transformations**: Apply correctly to all files
- **State Tracking**: Complete audit trail with performance metrics

### Performance (Optimized Targets)
- **Small Directories** (<50 files): < 500ms
- **Medium Directories** (50-150 files): 1-2s
  - .github/workflows (24 files): ~400ms
  - .github/coverage (87 files): ~1.5s
- **Large Directories** (150-500 files): 2-4s
  - Full .github (149 files with exclusions): ~2s
- **Very Large Directories** (500-1000 files): < 5s
- **Memory Usage**: Linear with file count (~1MB per 100 files)

### API Efficiency
- **Batch Operations**: 80%+ reduction in API calls
- **Cache Hit Rate**: 50%+ for unchanged content
- **Rate Limit Safety**: Never exceeds 50% of limit
- **Tree API Usage**: Single call for file existence checks

### Compatibility
- **Backward Compatible**: All existing configs work
- **CI/CD**: No changes required
- **State Discovery**: Handles mixed configurations
- **Git Operations**: No additional overhead

### Developer Experience
- **Configuration**: Intuitive with smart defaults
- **Progress Reporting**: Automatic for >50 files
- **Error Messages**: Include performance hints
- **Examples**: Real-world scenarios (e.g., .github sync)

## Risk Mitigation

### Technical Risks
- **Large Directories**: Add file count warnings
- **Deep Nesting**: Implement recursion limits
- **Symbolic Links**: Follow Go's filepath.Walk behavior
- **Performance**: Concurrent processing within directories

### Compatibility Risks
- **No Breaking Changes**: Purely additive feature
- **State Format**: Extended but backward compatible
- **Branch Naming**: Unchanged format
- **PR Metadata**: Additional fields only

### Adoption Strategy
- **Gradual Rollout**: Start with small directories
- **Clear Benefits**: Document time savings
- **Migration Guide**: Show before/after configs
- **Team Training**: Simple examples first

## Future Enhancements

After the initial implementation proves successful:

### Phase 1: Advanced Patterns
- Support for `**` glob patterns
- Regex-based exclusions
- Include-only patterns
- Pattern precedence rules

### Phase 2: Performance Optimizations
- Parallel directory processing
- Smart caching for unchanged directories
- Incremental sync for large directories
- Progress reporting for long operations

### Phase 3: Enhanced Features
- Directory-specific transformations
- File renaming during sync
- Permission preservation options
- Symlink handling options

### Phase 4: Developer Experience
- Interactive directory preview
- Exclusion pattern testing tool
- Directory diff visualization
- Performance profiling per directory

## Conclusion

This enhanced directory sync implementation delivers a performant v1 that can efficiently handle real-world directory structures like `.github` (149 files) while maintaining go-broadcast's core principles.

**Key improvements in this plan:**
- **Smart Defaults**: Automatic exclusion of development artifacts (.out, .test, binaries)
- **Concurrent Processing**: Worker pools for file discovery and transformation
- **API Optimization**: GitHub tree API reduces calls by 80%+
- **Progress Reporting**: Automatic feedback for directories >50 files
- **Performance Guarantees**: .github/coverage (87 files) syncs in ~1.5s

The implementation follows go-broadcast's established patterns:
- **Stateless architecture** with dynamic directory resolution
- **Complete backward compatibility** with existing configurations
- **Comprehensive audit trail** with performance metrics
- **Optimized for Go modules** like .github/coverage
- **Developer-friendly** with smart defaults and clear feedback

This positions go-broadcast as a production-ready solution for repository synchronization, efficiently handling everything from individual files to complex directory structures with nested Go modules.
