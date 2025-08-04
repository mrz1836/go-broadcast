---
allowed-tools: Task, Bash(go version:*), Bash(git status:*), Bash(go env:*), Write
description: Collect comprehensive diagnostic information
argument-hint: [specific area or leave empty for full diagnostic]
---
!go version && echo "---" && git status --short | head -5

# 🔍 System Diagnostics Collection

I need to collect comprehensive diagnostic information for troubleshooting.

**Use the diagnostic-specialist agent** to gather:

1. **System information**:
   - Go version and environment
   - Git status and configuration
   - OS and architecture details
   - Available resources (CPU, memory)
   - Network connectivity status

2. **Project state**:
   - Current branch and commit
   - Modified files
   - Dependency versions
   - Build configuration
   - Recent error logs

3. **go-broadcast specific**:
   - Sync operation history
   - Configuration validation results
   - GitHub API rate limit status
   - Recent sync failures
   - Cache state

4. **Performance metrics**:
   - Recent benchmark results
   - Memory usage patterns
   - API call statistics
   - Operation timings
   - Resource utilization

5. **Error analysis**:
   - Recent test failures
   - CI/CD pipeline issues
   - Runtime errors
   - Configuration problems
   - Integration failures

**Output format:**
- Structured JSON report
- Markdown summary
- Actionable insights
- Troubleshooting recommendations

## Diagnostic Focus: $ARGUMENTS

The agent will create a comprehensive diagnostic report that can be shared with maintainers or used for self-service troubleshooting.
