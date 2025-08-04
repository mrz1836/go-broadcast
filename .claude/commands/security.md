---
allowed-tools: Task, Bash(govulncheck:*), Bash(nancy:*), Bash(gitleaks:*), WebFetch
description: Comprehensive security audit with multiple scanners
argument-hint: [specific package/file or leave empty for full scan]
---
# 🛡️ Comprehensive Security Audit

I need to perform a thorough security audit using multiple specialized agents working in parallel.

**Parallel execution of three security agents:**

1. **Use the security-auditor agent** to:
   - Run govulncheck for Go vulnerability database checks
   - Execute nancy for dependency vulnerability scanning
   - Run gitleaks for secret detection
   - Perform OSSAR static analysis
   - Generate consolidated security report

2. **Use the compliance-checker agent** to:
   - Check OpenSSF Scorecard compliance
   - Verify branch protection rules
   - Audit security policies
   - Ensure supply chain security
   - Track security metrics over time

3. **Use the dependabot-coordinator agent** to:
   - Review pending security updates
   - Identify high-priority patches
   - Check for available security fixes
   - Coordinate with breaking changes

**Consolidation phase:**
- Merge findings from all agents
- Prioritize critical vulnerabilities
- Generate actionable fix recommendations
- Create security improvement roadmap
- Update security documentation

## Security Scan Target: $ARGUMENTS

All three agents will work simultaneously to provide the most comprehensive security assessment possible, with automatic fixes applied where safe to do so.
