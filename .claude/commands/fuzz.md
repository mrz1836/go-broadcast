---
allowed-tools: Task, Bash(go test -fuzz:*), Bash(ls testdata:*), Edit, MultiEdit
description: Run security-focused fuzz testing to discover edge cases
argument-hint: [specific function to fuzz or leave empty for all]
---
# 🛡️ Security Fuzz Testing

I need to run comprehensive fuzz testing to discover security vulnerabilities and edge cases.

**Use the fuzz-test-guardian agent** to:

1. **Identify fuzzable functions** in the codebase, especially:
   - Parsing and validation logic
   - Input handlers and transformers
   - Security-critical code paths
   - Functions processing external data

2. **Generate intelligent fuzz corpus**:
   - Create seed inputs based on code analysis
   - Include edge cases and boundary values
   - Add malformed inputs for security testing
   - Incorporate previously discovered crashes

3. **Execute fuzz tests**:
   - Run with appropriate time limits
   - Monitor for crashes and panics
   - Track code coverage during fuzzing
   - Save interesting inputs that increase coverage

4. **Fix discovered issues**:
   - Analyze crash reports
   - Implement proper input validation
   - Add bounds checking where needed
   - Create regression tests for fixed issues

## Fuzz Target: $ARGUMENTS

The agent will ensure all security-critical code is thoroughly fuzz tested and any discovered vulnerabilities are fixed with appropriate safeguards.
