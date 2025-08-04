---
allowed-tools: Task, Bash(go-broadcast:*), Read, Edit
description: Validate and optimize go-broadcast sync operations
argument-hint: [validate|dry-run|optimize]
---
# 🔄 Sync Operations Management

I need to manage go-broadcast sync operations: ${ARGUMENTS:-validate}

**Parallel execution of sync agents:**

1. **Use the sync-orchestrator agent** to:
   - Load and parse sync configuration
   - Coordinate sync workflow
   - Execute dry-run operations
   - Handle sync state management
   - Process sync errors gracefully
   - Generate operation reports

2. **In parallel, use the config-validator agent** to:
   - Validate YAML syntax
   - Check repository access permissions
   - Verify transformation rules
   - Validate file/directory mappings
   - Test exclusion patterns
   - Ensure configuration consistency

**Operation modes:**
- **validate**: Check configuration validity
- **dry-run**: Preview changes without applying
- **optimize**: Improve sync performance

**Sync optimization targets:**
- Minimize GitHub API calls
- Optimize directory processing
- Improve exclusion efficiency
- Enhance transformation speed
- Reduce memory usage
- Parallelize operations

**Quality checks:**
- Configuration best practices
- Security considerations
- Performance bottlenecks
- Error handling robustness
- State consistency

The agents will ensure sync operations are valid, efficient, and reliable with comprehensive reporting.
