---
allowed-tools: Task, Bash(go test:*), Read, Edit
description: Run comprehensive integration tests
argument-hint: [phase: basic|network|full or specific test]
---
# 🧩 Phased Integration Testing

I need to run comprehensive integration tests: ${ARGUMENTS:-full}

**Use the integration-test-manager agent** to execute phased testing:**

1. **Phase 1: Basic Integration** (fast)
   - Core functionality tests
   - Basic sync operations
   - Configuration validation
   - State management
   - Error handling paths

2. **Phase 2: Network Integration** (medium)
   - GitHub API interactions
   - Rate limit handling
   - Network error recovery
   - Concurrent operations
   - Authentication flows

3. **Phase 3: Advanced Scenarios** (comprehensive)
   - Large repository handling
   - Complex transformations
   - Edge case scenarios
   - Performance under load
   - Failure recovery

**Test categories:**
- **Sync workflows**: Multi-repo synchronization
- **Transformation**: Variable substitution, repo name updates
- **Directory sync**: Large directory handling, exclusions
- **API optimization**: Tree API usage, caching
- **Error scenarios**: Network failures, permissions

**Quality metrics:**
- Test execution time
- Coverage of edge cases
- API call efficiency
- Memory usage patterns
- Error handling robustness

**Integration points:**
- GitHub API
- File system operations
- Git operations
- Configuration parsing
- State management

The agent will ensure all integration points work correctly in real-world scenarios.
