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
   - Track test coverage and ensure it stays above 85%
   - Ensure all test patterns follow conventions
   - Generate detailed test reports

2. **Coordinate with go-quality-enforcer** if needed to:
   - Fix any linting issues discovered during testing
   - Ensure code follows all 60+ configured linters

## Target: $ARGUMENTS

The test-commander agent will ensure all tests pass, coverage meets requirements, and code quality standards are maintained. It will automatically fix issues where possible and provide detailed reports of any manual interventions needed.
