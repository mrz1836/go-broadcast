# testutil Package

The `testutil` package provides shared test utilities and patterns for consistent testing across the go-broadcast codebase. This package helps reduce test code duplication and establish consistent patterns for writing tests.

## Features

- **Mock utilities** - Standardized mock validation and result extraction
- **File helpers** - Common file creation and cleanup utilities for tests
- **Test patterns** - Generic test case structures and table-driven test helpers
- **Assertion helpers** - Simplified assertions with clear error messages
- **Benchmark utilities** - Consistent benchmark case management

## Mock Utilities (mock.go)

### ValidateArgs
Validates that mock arguments match the expected count:
```go
func ValidateArgs(args mock.Arguments, expectedCount int) error
```

### ExtractResult
Extracts a typed result from mock arguments with type safety:
```go
result, err := testutil.ExtractResult[string](args, 0)
if err != nil {
    return err
}
```

### ExtractError
Extracts an error from mock arguments (typically the last argument):
```go
err := testutil.ExtractError(args)
if err != nil {
    return err
}
```

## File Utilities (files.go)

### CreateTestFiles
Creates multiple test files with default content:
```go
files := testutil.CreateTestFiles(t, "/tmp/test", 5)
// Creates 5 files: test_file_0.txt through test_file_4.txt
```

### WriteTestFile
Creates a single test file with custom content:
```go
testutil.WriteTestFile(t, "/tmp/test/config.json", `{"key": "value"}`)
```

### CreateTestDirectory
Creates a temporary test directory with cleanup:
```go
dir := testutil.CreateTestDirectory(t)
// Directory is automatically cleaned up after test
```

### CreateBenchmarkFiles
Creates files specifically for benchmark testing:
```go
files := testutil.CreateBenchmarkFiles(b, dir, 100)
```

### WriteBenchmarkFile
Creates a single file for benchmark testing:
```go
testutil.WriteBenchmarkFile(b, "data.json", largeJSON)
```

## Test Patterns (patterns.go)

### Generic Test Case Structure
Use `TestCase` for organizing table-driven tests with type safety:
```go
tests := []testutil.TestCase[Input, Output]{
    {
        Name:     "valid input",
        Input:    Input{Value: 42},
        Expected: Output{Result: 84},
        WantErr:  false,
    },
    {
        Name:    "invalid input",
        Input:   Input{Value: -1},
        WantErr: true,
        ErrMsg:  "invalid value",
    },
}
```

### RunTableTests
Execute table-driven tests with consistent patterns:
```go
testutil.RunTableTests(t, tests, func(t *testing.T, tc testutil.TestCase[Input, Output]) {
    result, err := ProcessInput(tc.Input)
    
    if tc.WantErr {
        testutil.AssertError(t, err)
        if tc.ErrMsg != "" {
            testutil.AssertErrorContains(t, err, tc.ErrMsg)
        }
    } else {
        testutil.AssertNoError(t, err)
        testutil.AssertEqual(t, tc.Expected, result)
    }
})
```

### Assertion Helpers
Simple assertions with clear error messages:
```go
// Check for no error
testutil.AssertNoError(t, err)

// Check that error occurred
testutil.AssertError(t, err)

// Check error message
testutil.AssertErrorContains(t, err, "expected substring")

// Check equality
testutil.AssertEqual(t, expected, actual)

// Check inequality
testutil.AssertNotEqual(t, unexpected, actual)
```

### Benchmark Utilities
Organize benchmarks with consistent patterns:
```go
cases := []testutil.BenchmarkCase{
    {Name: "small", Size: 10},
    {Name: "medium", Size: 100},
    {Name: "large", Size: 1000},
}

testutil.RunBenchmarkCases(b, cases, func(b *testing.B, bc testutil.BenchmarkCase) {
    for i := 0; i < b.N; i++ {
        ProcessData(bc.Size)
    }
})
```

### Skip Helpers
Conditionally skip tests:
```go
// Skip if running with -short flag
testutil.SkipIfShort(t)

// Skip tests requiring network
testutil.SkipIfNoNetwork(t)
```

## Migration Guide

### From Manual Mock Validation
Before:
```go
if len(args) != 2 {
    return nil, fmt.Errorf("expected 2 args, got %d", len(args))
}
result, ok := args[0].(string)
if !ok {
    return nil, fmt.Errorf("arg 0 is not string")
}
```

After:
```go
result, err := testutil.ExtractResult[string](args, 0)
if err != nil {
    return nil, err
}
```

### From Manual File Creation
Before:
```go
for i := 0; i < 10; i++ {
    path := filepath.Join(dir, fmt.Sprintf("file_%d.txt", i))
    err := os.WriteFile(path, []byte("content"), 0600)
    require.NoError(t, err)
}
```

After:
```go
files := testutil.CreateTestFiles(t, dir, 10)
```

### From Ad-hoc Test Structures
Before:
```go
tests := []struct {
    name    string
    input   string
    want    int
    wantErr bool
}{
    // test cases
}
```

After:
```go
tests := []testutil.TestCase[string, int]{
    {
        Name:     "valid input",
        Input:    "42",
        Expected: 42,
        WantErr:  false,
    },
}
```

## Best Practices

1. **Use table-driven tests** - Leverage `TestCase` and `RunTableTests` for comprehensive test coverage
2. **Prefer helpers over repetition** - Use assertion helpers instead of repeating error checks
3. **Clean up resources** - File utilities handle cleanup automatically
4. **Type safety** - Use generic functions for type-safe operations
5. **Descriptive names** - Use clear test case names that describe the scenario

## Examples

See `example_patterns_test.go` for comprehensive examples of using these utilities.
