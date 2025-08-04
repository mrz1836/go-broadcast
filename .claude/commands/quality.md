---
allowed-tools: Task, Bash(make lint:*), Bash(golangci-lint:*), Edit, MultiEdit
description: Enforce code quality with 60+ linters and automatic fixes
argument-hint: [specific file/package or leave empty for entire codebase]
---
!make lint 2>&1 | grep -E "(FAIL|Error|Warning)" | head -10

# 🎯 Comprehensive Code Quality Enforcement

I need to ensure the codebase meets all quality standards using our extensive linting suite.

**Primary: Use the go-quality-enforcer agent** to:

1. **Run all 60+ configured linters**:
   - golangci-lint with custom configuration
   - gofumpt for consistent formatting
   - Security linters (gosec, etc.)
   - Performance linters
   - Style and convention checkers

2. **Automatically fix issues**:
   - Apply gofumpt formatting
   - Fix simple linting violations
   - Update deprecated code patterns
   - Ensure consistent naming conventions

3. **Handle complex issues**:
   - Refactor code to eliminate linting errors
   - Add necessary comments for exported functions
   - Fix cyclomatic complexity issues
   - Resolve duplicate code warnings

**In parallel: Use the test-commander agent** to:
- Verify tests still pass after quality fixes
- Ensure no regressions from auto-fixes

## Quality Check Scope: $ARGUMENTS

The agents will work together to achieve 100% linting compliance while maintaining all functionality. Any issues that cannot be auto-fixed will be clearly documented with recommended solutions.
