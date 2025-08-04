---
allowed-tools: Task, Bash(go tool:*), Bash(git log:*), Read, Grep
description: Deep debug analysis for complex issues
argument-hint: <issue description or error message>
---
# 🐛 Deep Debugging Analysis

I need to perform comprehensive debugging for the issue: $ARGUMENTS

**Sequential debugging workflow:**

1. **Use the diagnostic-specialist agent** to:
   - Collect all relevant diagnostic information
   - Analyze error logs and stack traces
   - Review recent code changes
   - Check system state and configuration
   - Identify potential root causes
   - Create initial troubleshooting report

2. **Then use the debugging-expert agent** to:
   - Perform deep-dive debugging based on diagnostics
   - Analyze race conditions if applicable
   - Check for memory leaks or corruption
   - Inspect goroutine behavior
   - Review mutex/channel usage
   - Use advanced debugging techniques

**Debugging techniques applied:**
- Stack trace analysis
- Memory profiling
- Goroutine inspection
- Race condition detection
- Deadlock analysis
- Performance bottleneck identification

**Resolution steps:**
- Identify root cause with evidence
- Provide specific fix recommendations
- Include code examples
- Add defensive programming suggestions
- Create tests to prevent regression

The agents will work sequentially, with the diagnostic specialist gathering information that the debugging expert uses for deep analysis and resolution.
