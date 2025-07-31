# Performance Baseline Metrics

Generated: 2025-07-24

## Current Benchmark Results

### Config Package (`internal/config`)

| Benchmark             | Iterations | ns/op     | B/op    | allocs/op | Notes              |
|-----------------------|------------|-----------|---------|-----------|--------------------|
| LoadFromReader        | 34,320     | 35,063    | 24,456  | 394       | Small config       |
| LoadFromReader_Large  | 849        | 1,411,101 | 782,392 | 15,832    | 100 targets        |
| Validate              | 473,331    | 2,428     | 1,521   | 14        | Small config       |
| Validate_Large        | 21,516     | 56,975    | 8,223   | 23        | 100 targets        |
| LoadAndValidate       | 31,468     | 38,130    | 25,999  | 408       | Combined operation |
| LoadAndValidate_Large | 813        | 1,483,731 | 791,326 | 15,856    | 100 targets        |

**Key Observations:**
- Large configs (100 targets) are ~40x slower than small configs
- Memory usage scales linearly with config size
- Validation is relatively fast compared to parsing

### Transform Package (`internal/transform`)

| Benchmark                  | Iterations | ns/op   | B/op      | allocs/op | Notes         |
|----------------------------|------------|---------|-----------|-----------|---------------|
| TemplateTransform_Small    | 29,791     | 40,499  | 68,456    | 583       | ~300 bytes    |
| TemplateTransform_Large    | 1,245      | 968,229 | 1,985,048 | 712       | ~30KB         |
| RepoTransform_GoFile       | 82,586     | 14,250  | 21,804    | 133       | Small Go file |
| RepoTransform_GoFile_Large | 2,146      | 563,744 | 279,363   | 352       | ~14KB Go file |
| RepoTransform_Markdown     | 55,273     | 21,575  | 20,263    | 125       | README size   |
| BinaryDetection/Text       | 33,849,976 | 35.26   | 0         | 0         | Plain text    |
| BinaryDetection/Binary     | 73,101,453 | 16.24   | 0         | 0         | PNG header    |
| BinaryDetection/LargeText  | 215,406    | 5,443   | 0         | 0         | ~12KB text    |
| ChainTransform             | 42,892     | 28,851  | 44,978    | 354       | Full pipeline |
| ChainTransform_Binary      | 331,166    | 3,598   | 6,558     | 61        | Binary bypass |

**Key Observations:**
- Template transform has high allocation count (583 for small file)
- Binary detection is very efficient (zero allocations)
- Chain transform with binary files is 8x faster due to early exit

### State Package (`internal/state`)

| Benchmark            | Iterations | ns/op | B/op | allocs/op | Notes               |
|----------------------|------------|-------|------|-----------|---------------------|
| BranchParsing        | 996,838    | 1,198 | 544  | 12        | Regex parsing       |
| PRParsing            | 25,734,987 | 46.38 | 0    | 0         | Simple field access |
| StateComparison      | 27,177,913 | 45.33 | 0    | 0         | Commit comparison   |
| SyncBranchGeneration | 10,497,003 | 114.9 | 64   | 2         | String formatting   |
| StateAggregation     | 513,632    | 2,343 | 0    | 0         | 100 targets         |

**Key Observations:**
- Branch parsing allocates due to regex operations
- PR and state comparison are allocation-free
- State aggregation scales well with target count

### Sync Package (`internal/sync`)

| Benchmark                | Iterations  | ns/op | B/op   | allocs/op | Notes               |
|--------------------------|-------------|-------|--------|-----------|---------------------|
| FilterTargets            | 299,329     | 3,589 | 16,208 | 8         | 100 targets         |
| FilterTargets_WithFilter | 7,259,798   | 171.4 | 352    | 4         | 3 targets, filtered |
| NeedsSync/UpToDate       | 144,084,021 | 8.304 | 0      | 0         | Fast path           |
| NeedsSync/Behind         | 142,957,431 | 8.267 | 0      | 0         | Needs sync          |
| NeedsSync/Pending        | 137,130,776 | 8.735 | 0      | 0         | PR pending          |

**Key Observations:**
- FilterTargets allocates significantly with many targets
- NeedsSync is extremely efficient (no allocations)
- Progress tracking benchmarks need cleanup (logging noise)

## Performance Bottlenecks Identified

### 1. High Allocation Areas
- **Template Transform**: 583 allocations for small files
- **Config Parsing**: 394 allocations for small configs
- **Repo Transform**: 133 allocations per file

### 2. Scaling Issues
- **Large Config Loading**: 40x slower with 100 targets
- **Template Transform**: Memory usage grows faster than file size

### 3. Missing Benchmarks
Critical paths without benchmarks:
- Git command execution
- GitHub API calls
- File I/O operations
- Concurrent operations
- Logging overhead

## Optimization Opportunities

### Quick Wins
1. **Regex Caching**: Branch parsing does 12 allocations per parse
2. **Buffer Pooling**: Template transform could reuse buffers
3. **String Building**: Sync branch generation allocates unnecessarily

### Medium Term
1. **Streaming**: Large file transforms load entire file into memory
2. **Batch Operations**: FilterTargets could batch allocations
3. **Lazy Evaluation**: Config validation could defer work

### Long Term
1. **Memory Layout**: Optimize struct layouts for cache efficiency
2. **Zero-Copy**: Implement zero-copy transforms where possible
3. **Custom Allocators**: Pool frequently allocated objects

## Next Steps

1. Implement benchmarks for missing critical paths (git, gh, logging)
2. Profile CPU and memory usage during real sync operations
3. Implement quick win optimizations with before/after metrics
4. Set up continuous benchmark tracking to prevent regressions
