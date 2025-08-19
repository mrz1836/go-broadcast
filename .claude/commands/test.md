---
allowed-tools: Task, Bash(magex test:*), TodoWrite
description: Run comprehensive tests with automatic fixes and coverage tracking
argument-hint: [specific test file, package, or leave empty for all tests]
---
!magex test:coverrace --no-cache 2>&1 | tail -20

# 🧪 Comprehensive Test Suite with Auto-Fix

I need to run comprehensive tests and ensure code quality. Here's what I'll do:

1. **Use the test-commander agent** to:
   - Run all tests with race detection enabled
   - Automatically fix any failing tests
   - Ensure all test patterns follow conventions
   - Generate detailed test reports

2. **In parallel, use the coverage-maintainer agent** to:
   - Track test coverage and ensure it stays above 85%
   - Generate coverage badges and reports
   - Update GitHub Pages with latest coverage data
   - Create PR comments with coverage analysis

3. **Coordinate with go-quality-enforcer** if needed to:
   - Fix any linting issues discovered during testing
   - Ensure code follows all 60+ configured linters

## Target: $ARGUMENTS

The agents will work together to ensure all tests pass, coverage meets requirements, and code quality standards are maintained. They'll automatically fix issues where possible and provide detailed reports of any manual interventions needed.
