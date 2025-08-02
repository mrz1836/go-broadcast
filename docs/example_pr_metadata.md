# Enhanced PR Metadata Example

This document shows an example of the enhanced PR body with directory synchronization information and performance metrics.

## Example PR Body

```markdown
## What Changed
* Updated 3 individual file(s) to synchronize with the source repository
* Synchronized 61 file(s) from directory mappings
* Applied file transformations and updates based on sync configuration
* Brought target repository in line with source repository state at commit abc123d

## Directory Synchronization Details
The following directories were synchronized:

### `.github/coverage` → `.github/coverage`
* **Files synced**: 61
* **Files excluded**: 26
* **Processing time**: 1523ms
* **Binary files skipped**: 3 (1.50 KB)
* **Exclusion patterns**: `*.out`, `*.test`, `gofortress-coverage`

## Performance Metrics
* **Total sync time**: 2.5s
* **Files processed**: 3 (3 changed, 0 skipped)
* **File processing time**: 250ms
* **Directory files processed**: 61 (26 excluded)
* **API calls saved**: 72 (through optimization)
* **Cache hit rate**: 62.5% (45 hits, 27 misses)

## Why It Was Necessary
This synchronization ensures the target repository stays up-to-date with the latest changes from the configured source repository. The sync operation identifies and applies only the necessary file changes while maintaining consistency across repositories.

## Testing Performed
* Validated sync configuration and file mappings
* Verified file transformations applied correctly
* Confirmed no unintended changes were introduced
* All automated checks and linters passed

## Impact / Risk
* **Low Risk**: Standard sync operation with established patterns
* **No Breaking Changes**: File updates maintain backward compatibility
* **Performance**: No impact on application performance
* **Dependencies**: No dependency changes included in this sync

<!-- go-broadcast-metadata
sync_metadata:
  source_repo: org/template
  source_commit: abc123def456
  target_repo: org/target
  sync_commit: def456abc123
  sync_time: 2025-08-01T19:05:00-04:00
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
-->
```

## Key Features

### Enhanced Change Description
- Distinguishes between individual file changes and directory synchronization
- Provides specific counts for each type of change
- Makes it clear what transformations were applied

### Directory Synchronization Details
- Shows each directory mapping with source and destination
- Includes detailed metrics for each directory processed
- Lists exclusion patterns used during synchronization
- Reports binary files that were skipped with size information
- Shows processing time per directory

### Performance Metrics
- Overall sync timing information
- File processing breakdown (processed, changed, skipped)
- Directory processing statistics
- API optimization metrics (calls saved through caching/tree API)
- Cache performance statistics with hit rate calculation

### Machine-Parseable Metadata
- YAML format within HTML comment block
- Complete directory mapping information with metrics
- Performance data for programmatic analysis
- Backwards compatible with existing parsers

This enhanced format provides both human-readable insights and machine-parseable data for automation and monitoring purposes.