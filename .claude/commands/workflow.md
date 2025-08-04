---
allowed-tools: Task, Read, Edit, MultiEdit, Bash(gh workflow:*)
description: Fix and optimize GitHub Actions workflows
argument-hint: [specific workflow or leave empty for all]
---
!gh workflow list --all | head -10

# ⚙️ GitHub Actions Workflow Optimization

I need to analyze and optimize GitHub Actions workflows.

**Use the workflow-optimizer agent** to:

1. **Workflow analysis**:
   - Scan .github/workflows/: ${ARGUMENTS:-all}
   - Check for deprecated actions
   - Identify slow-running jobs
   - Find redundant steps
   - Analyze failure patterns
   - Review .env.shared configuration

2. **Security updates**:
   - Update action versions to latest
   - Fix security warnings
   - Implement dependency pinning
   - Add security scanning
   - Review permissions

3. **Performance optimization**:
   - Implement job parallelization
   - Add caching strategies
   - Optimize matrix builds
   - Reduce redundant work
   - Improve runner selection

4. **Reliability improvements**:
   - Add retry logic for flaky tests
   - Implement timeout controls
   - Improve error handling
   - Add status badges
   - Enhance notifications

5. **Best practices**:
   - Use composite actions for reuse
   - Implement proper secrets handling
   - Add workflow documentation
   - Create reusable workflows
   - Optimize trigger conditions

**Focus areas:**
- CI/CD pipeline efficiency
- Security compliance
- Cost optimization
- Developer experience
- Monitoring and alerts

The agent will transform workflows into efficient, secure, and maintainable CI/CD pipelines.
