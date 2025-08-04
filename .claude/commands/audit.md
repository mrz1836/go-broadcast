---
allowed-tools: Task, Bash(govulncheck:*), Bash(go list:*), Read
description: Quick vulnerability scan for CI/CD pipelines
argument-hint: [specific module or leave empty for all]
---
!govulncheck ./... 2>&1 | grep -E "(Vulnerability|affected)" | head -10

# ⚡ Quick Security Audit

I need to perform a fast security audit suitable for CI/CD pipelines.

**Use the security-auditor agent** to perform a focused security check:

1. **Vulnerability scanning**:
   - Run govulncheck for known Go vulnerabilities
   - Quick scan of direct dependencies
   - Check for critical security issues only
   - Skip time-consuming deep analysis

2. **Rapid assessment**:
   - Focus on HIGH and CRITICAL vulnerabilities
   - Check against latest vulnerability database
   - Verify no exposed secrets in recent commits
   - Quick supply chain check

3. **CI/CD optimized output**:
   - Exit with appropriate status codes
   - Clear pass/fail indicators
   - Minimal output for clean logs
   - Actionable fix suggestions only

4. **If vulnerabilities found**:
   - List affected packages
   - Show upgrade paths
   - Indicate severity levels
   - Provide quick remediation steps

## Audit Scope: $ARGUMENTS

This lightweight audit is designed for speed while still catching critical security issues that could block deployments.
