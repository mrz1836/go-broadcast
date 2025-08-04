---
allowed-tools: Task, Grep, Read, Write, Bash(gh issue create:*)
description: Technical debt analysis and tracking
argument-hint: [analyze|report|track]
---
# 💳 Technical Debt Management

I need to analyze and track technical debt in the codebase.

**Use the tech-debt-tracker agent** to:

1. **Debt identification**:
   - Scan for TODO/FIXME comments
   - Identify deprecated code usage
   - Find overly complex functions
   - Detect poor test coverage areas
   - Analyze performance bottlenecks
   - Review outdated dependencies

2. **Debt categorization**:
   - **🔴 Critical**: Security risks, major bugs
   - **🟡 High**: Performance issues, bad patterns
   - **🟢 Medium**: Code smells, missing tests
   - **⚪ Low**: Style issues, minor improvements

3. **Impact analysis**:
   - Development velocity impact
   - Maintenance burden score
   - Risk assessment
   - Effort estimation
   - Business impact evaluation

4. **Debt metrics**:
   - Total debt items by category
   - Debt accumulation rate
   - Average resolution time
   - Debt per module/package
   - Trend analysis

5. **Action items**:
   - Create prioritized roadmap
   - Generate GitHub issues
   - Assign debt budgets
   - Track resolution progress
   - Schedule debt sprints

**Output options** based on $ARGUMENTS:
- **analyze**: Full debt analysis report
- **report**: Executive summary with metrics
- **track**: Create/update GitHub issues

The agent will provide actionable insights to systematically reduce technical debt.
