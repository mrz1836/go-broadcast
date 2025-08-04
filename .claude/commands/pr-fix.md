---
allowed-tools: Task, Bash(gh pr view:*), Bash(make test:*), Edit, MultiEdit
description: Fix all issues in the current pull request
argument-hint: [PR number or leave empty for current branch]
---
!gh pr view --json state,mergeable,reviews | head -20

# 🔧 Pull Request Auto-Fix

I need to fix all issues preventing this PR from merging.

**Parallel execution of PR fix agents:**

1. **Use the pr-automation-manager agent** to:
   - Analyze PR status and blockers
   - Apply appropriate labels
   - Assign reviewers if missing
   - Update PR description
   - Handle merge conflicts
   - Manage PR metadata

2. **Use the test-commander agent** to:
   - Fix failing tests
   - Ensure coverage requirements
   - Run integration tests
   - Validate examples
   - Update test documentation

3. **Use the go-quality-enforcer agent** to:
   - Fix all linting issues
   - Apply code formatting
   - Resolve style violations
   - Update deprecated code
   - Ensure convention compliance

**Additional checks:**
- Verify CI/CD passes
- Check security scans
- Validate documentation
- Ensure changelog entry
- Review dependencies

**PR target: ${ARGUMENTS:-current branch}**

The agents will work simultaneously to resolve all issues and prepare the PR for successful merge.
