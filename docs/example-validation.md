# Example Configuration Validation Guide

This guide documents the validation testing performed for all go-broadcast example configurations, including the directory sync examples created for Phase 6 of the directory sync implementation.

## Validation Script

The [`scripts/validate-examples.sh`](../scripts/validate-examples.sh) script provides automated validation of all example configurations.

### Usage

```bash
# Make script executable (if needed)
chmod +x scripts/validate-examples.sh

# Run validation for all examples
./scripts/validate-examples.sh

# Run with verbose output
./scripts/validate-examples.sh --verbose

# Show help
./scripts/validate-examples.sh --help
```

## Example Configurations Tested

### File Sync Examples (Existing)
- ✅ **minimal.yaml** - Basic file sync configuration
- ✅ **sync.yaml** - Complete example with all features
- ✅ **microservices.yaml** - Microservices architecture sync
- ✅ **multi-language.yaml** - Multi-language project sync
- ✅ **ci-cd-only.yaml** - CI/CD pipeline synchronization
- ✅ **documentation.yaml** - Documentation template sync

### Directory Sync Examples (New)
- 🔄 **directory-sync.yaml** - Comprehensive directory sync examples
- 🔄 **github-workflows.yaml** - GitHub infrastructure sync
- 🔄 **large-directories.yaml** - Large directory management
- 🔄 **exclusion-patterns.yaml** - Exclusion pattern showcase
- 🔄 **github-complete.yaml** - Complete GitHub directory sync

**Note**: Directory sync examples require the directory sync feature to be fully implemented in the go-broadcast binary. Current validation shows `field directories not found in type config.TargetConfig` which is expected until the implementation is complete.

## Manual Validation Steps

### 1. Configuration Syntax Validation

```bash
# Test each configuration file
go-broadcast validate --config examples/minimal.yaml
go-broadcast validate --config examples/directory-sync.yaml
go-broadcast validate --config examples/github-workflows.yaml
go-broadcast validate --config examples/large-directories.yaml
go-broadcast validate --config examples/exclusion-patterns.yaml  
go-broadcast validate --config examples/github-complete.yaml
```

**Expected Results**:
- ✅ File-only examples: "Configuration is valid!"
- 🔄 Directory examples: Will be valid once directory sync implementation is complete

### 2. Dry-Run Testing

```bash
# Test dry-run mode (requires valid repositories)
go-broadcast sync --dry-run --config examples/minimal.yaml
go-broadcast sync --dry-run --config examples/directory-sync.yaml
```

**Expected Results**:
- Configuration parsing works correctly
- Remote repository validation may fail (expected for example configs with placeholder names)
- No actual changes made to repositories

### 3. Command Testing

```bash
# Test all documented commands
go-broadcast --version
go-broadcast --help
go-broadcast validate --help
go-broadcast sync --help
go-broadcast status --help
go-broadcast diagnose --help
go-broadcast cancel --help
```

**Expected Results**:
- All commands should display help/version information correctly
- No errors in command parsing

## Validation Results

### Current Status (Phase 6 Implementation)

#### File Sync Examples ✅
All existing file sync examples validate successfully:

```
✅ VALID: examples/minimal.yaml
✅ VALID: examples/sync.yaml  
✅ VALID: examples/microservices.yaml
✅ VALID: examples/multi-language.yaml
✅ VALID: examples/ci-cd-only.yaml
✅ VALID: examples/documentation.yaml
```

**Sample validation output**:
```
Validating configuration file: examples/minimal.yaml
✓ Configuration is valid!

Configuration Summary:
  Version: 1
  Source: org/template-repo (branch: master)
  Targets: 1 repositories
    1. org/target-repo
       Files: 1 mappings

Total file mappings: 1

Additional checks:
  ✓ Repository name format
  ✓ File paths
  ✓ No duplicate targets
  ✓ No duplicate file destinations
```

#### Directory Sync Examples 🔄  
Directory sync examples show expected validation errors until implementation is complete:

```
❌ VALIDATION PENDING: examples/directory-sync.yaml
❌ VALIDATION PENDING: examples/github-workflows.yaml
❌ VALIDATION PENDING: examples/large-directories.yaml
❌ VALIDATION PENDING: examples/exclusion-patterns.yaml
❌ VALIDATION PENDING: examples/github-complete.yaml
```

**Expected error (until implementation complete)**:
```
Failed to parse configuration: yaml: unmarshal errors:
  line X: field directories not found in type config.TargetConfig
```

### Post-Implementation Validation Plan

Once directory sync implementation is complete, all examples should validate successfully:

#### Expected Validation Results

1. **directory-sync.yaml**:
   - ✅ 4 target repositories
   - ✅ Mixed files and directories configuration
   - ✅ Custom exclusion patterns
   - ✅ Transform application to directories

2. **github-workflows.yaml**:
   - ✅ GitHub infrastructure sync configuration
   - ✅ Global PR assignment settings
   - ✅ Service-specific customizations
   - ✅ Performance-optimized directory mappings

3. **large-directories.yaml**:
   - ✅ Large directory handling (1000+ files)
   - ✅ Performance optimization strategies
   - ✅ Memory usage configurations
   - ✅ API efficiency settings

4. **exclusion-patterns.yaml**:
   - ✅ Comprehensive exclusion pattern examples
   - ✅ Pattern syntax validation
   - ✅ Development artifact exclusions
   - ✅ Security-sensitive file patterns

5. **github-complete.yaml**:
   - ✅ Enterprise-scale configuration
   - ✅ Complete .github directory sync
   - ✅ Production service configurations
   - ✅ Performance metrics documentation

## Testing Checklist

### Pre-Implementation Testing ✅
- [x] File sync examples validate successfully
- [x] Basic commands work correctly
- [x] Validation script executes without errors
- [x] Documentation examples are syntactically correct
- [x] Help commands display correctly

### Post-Implementation Testing (Pending)
- [ ] Directory sync examples validate successfully
- [ ] Mixed file/directory configurations work
- [ ] Exclusion patterns compile correctly
- [ ] Transform application to directories works
- [ ] Performance configurations are recognized
- [ ] All documented commands work with directory configs

### Integration Testing (Pending)
- [ ] End-to-end directory sync works
- [ ] Performance targets are met
- [ ] API optimization functions correctly
- [ ] Error handling works as documented
- [ ] Progress reporting functions properly

## Troubleshooting Validation Issues

### Common Issues

1. **"field directories not found"** - Directory sync implementation not complete
2. **"Repository not found"** - Expected for example configs with placeholder names
3. **"Source file not found"** - Expected for example configs with placeholder files
4. **"Authentication failed"** - GitHub CLI not configured (expected)

### Resolution Steps

1. **For implementation issues**: Wait for directory sync implementation completion
2. **For repository issues**: Expected behavior with example configurations
3. **For authentication issues**: Not required for syntax validation
4. **For syntax errors**: Check YAML formatting and field names

## Performance Validation

Once implementation is complete, performance validation should verify:

### Expected Performance Metrics
- Small directories (<50 files): <3ms processing
- Medium directories (50-150 files): 1-7ms processing  
- Large directories (500+ files): 16-32ms processing
- Very large directories (1000+ files): ~32ms processing

### API Efficiency Metrics
- 90%+ reduction in GitHub API calls through tree API optimization
- Memory usage: ~1.2MB per 1000 files processed
- Cache hit rates: 50%+ for unchanged content

### Validation Commands
```bash
# Performance testing with debug logging
go-broadcast sync --log-level debug --config examples/large-directories.yaml 2>&1 | \
  grep -E "(processing_time_ms|duration|files_synced)"

# Memory usage monitoring
go-broadcast sync --config examples/github-complete.yaml &
PID=$!
while kill -0 $PID 2>/dev/null; do
  ps -o pid,vsz,rss,comm -p $PID
  sleep 1
done
```

## Continuous Validation

### Automated Testing
The validation script should be run:
- Before any release
- After configuration format changes  
- When adding new example files
- As part of CI/CD pipeline

### Integration with CI/CD
```yaml
# Example GitHub Actions step
- name: Validate Example Configurations
  run: |
    make build-go
    ./scripts/validate-examples.sh
    if [ $? -eq 0 ]; then
      echo "✅ All examples validated successfully"
    else
      echo "❌ Example validation failed"
      exit 1
    fi
```

## Related Documentation

- [Directory Sync Guide](directory-sync.md) - Complete directory sync documentation
- [Troubleshooting Guide](troubleshooting.md) - Common issues and solutions
- [Examples README](../examples/README.md) - Example usage and patterns
- [Performance Guide](../README.md#-performance) - Performance benchmarks and optimization