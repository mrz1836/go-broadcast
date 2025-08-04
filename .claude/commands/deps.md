---
allowed-tools: Task, Bash(go list:*), Bash(go mod:*), Bash(gh pr list:*), Edit, MultiEdit
description: Review and update all dependencies intelligently
argument-hint: [specific dependency or leave empty for all]
---
!go list -u -m all 2>&1 | grep -E "\[.*\]" | head -10

# 📦 Comprehensive Dependency Management

I need to review and update dependencies using multiple specialized agents working in parallel.

**Parallel execution of dependency agents:**

1. **Use the dependabot-coordinator agent** to:
   - Review all pending Dependabot PRs
   - Group related updates intelligently
   - Make auto-merge decisions for patch updates
   - Flag major updates requiring review
   - Handle security updates with priority

2. **Use the dependency-upgrader agent** to:
   - Check for updates beyond Dependabot's scope
   - Update Go version if newer is available
   - Upgrade build tools and linters
   - Update indirect dependencies
   - Perform monthly dependency review

3. **Use the breaking-change-detector agent** to:
   - Analyze all proposed updates for breaking changes
   - Check API compatibility
   - Review semantic versioning compliance
   - Generate migration plans if needed
   - Test updates in isolated environment

**Coordination phase:**
- Merge safe updates automatically
- Create grouped PRs for related changes
- Document breaking changes clearly
- Update go.mod and go.sum properly
- Run tests after each update batch

## Dependency Focus: $ARGUMENTS

The agents will work together to ensure dependencies are up-to-date, secure, and compatible while minimizing disruption to the codebase.
