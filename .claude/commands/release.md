---
allowed-tools: Task, Bash(git tag:*), Bash(make release:*), Edit, MultiEdit
description: Comprehensive release preparation workflow
argument-hint: <version> (e.g., 1.2.0)
model: opus
---
# 🚀 Complete Release Workflow

I need to prepare a comprehensive release for version: $ARGUMENTS

**Sequential release workflow with multiple agents:**

1. **Use the test-commander agent** to:
   - Run full test suite with race detection
   - Ensure 100% test passage
   - Verify coverage >85%
   - Run integration tests
   - Validate all examples work

2. **Use the security-auditor agent** to:
   - Perform final security scan
   - Check for vulnerabilities
   - Verify no secrets exposed
   - Validate security policies
   - Generate security report

3. **Run "make tag version=$ARGUMENTS"** to create a Git tag for the release.
   - Run "make tag version=$ARGUMENTS" to create a Git tag for the release.
   - Nothing else will be needed since the CI will handle the rest of the release process.

**Release checklist:**
- ✅ All tests passing
- ✅ Security scan clean
- ✅ Tag created

The agents will work in sequence to ensure a flawless release process.
