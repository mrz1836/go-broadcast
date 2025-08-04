---
allowed-tools: Task, Bash(git tag:*), Bash(make release:*), Edit, MultiEdit
description: Comprehensive release preparation workflow
argument-hint: <version> (e.g., v1.2.0)
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

3. **Use the changelog-generator agent** to:
   - Generate changelog since last release
   - Group changes by category
   - Highlight breaking changes
   - Create migration guide
   - Format release notes

4. **Use the release-manager agent** to:
   - Update version numbers
   - Tag the release
   - Run goreleaser
   - Create GitHub release
   - Upload artifacts
   - Publish documentation

5. **Use the documentation-maintainer agent** to:
   - Update version references
   - Refresh installation docs
   - Update compatibility matrix
   - Sync API documentation
   - Publish to pkg.go.dev

**Release checklist:**
- ✅ All tests passing
- ✅ Security scan clean
- ✅ Changelog updated
- ✅ Version bumped
- ✅ Documentation current
- ✅ Release notes ready
- ✅ Artifacts built

The agents will work in sequence to ensure a flawless release process.
