---
allowed-tools: Task, Bash(go mod:*), Bash(git diff:*), Read, Grep
description: Analyze dependency updates for breaking changes
argument-hint: <dependency-name@version or PR number>
---
# 🔍 Breaking Change Analysis

I need to perform deep analysis of dependency updates to identify potential breaking changes.

**Use the breaking-change-detector agent** to:

1. **Dependency analysis**:
   - Identify the specific dependency to analyze: $ARGUMENTS
   - Compare current vs proposed versions
   - Fetch changelogs and release notes
   - Analyze API differences

2. **Code impact assessment**:
   - Search codebase for usage of changed APIs
   - Identify deprecated function calls
   - Find removed or renamed methods
   - Check type signature changes

3. **Compatibility verification**:
   - Review semantic versioning compliance
   - Check for interface changes
   - Analyze struct field modifications
   - Verify constant/variable changes

4. **Migration planning**:
   - Generate list of required code changes
   - Estimate migration complexity
   - Provide code examples for updates
   - Suggest phased migration approach

5. **Risk assessment**:
   - Calculate risk score (LOW/MEDIUM/HIGH)
   - Identify potential runtime issues
   - Flag performance implications
   - Note security considerations

**Output includes:**
- Detailed breaking change report
- Affected code locations
- Migration checklist
- Recommended update strategy

The agent will provide a comprehensive analysis to help make informed decisions about dependency updates.
