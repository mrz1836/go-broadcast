# Code Quality Plan 07: Identifying and Reducing Code Duplication

## Executive Summary

This document outlines a comprehensive plan to identify and reduce code duplication across the go-broadcast Go codebase. The goal is to eliminate unnecessary repetition, improve maintainability, and create reusable patterns while preserving all existing functionality and ensuring all tests continue to pass.

## Objectives

1. **Eliminate Code Duplication**: Remove ~500+ lines of duplicated code across packages
2. **Improve Maintainability**: Create shared utilities and patterns for common operations
3. **Preserve Functionality**: Ensure all existing behavior remains unchanged
4. **Maintain Test Coverage**: All tests must continue to pass throughout refactoring
5. **Enhance Code Quality**: Standardize error handling, validation, and test patterns

## Technical Approach

### Duplication Analysis Framework
- **Static Analysis**: Identify repeated code patterns and similar function signatures
- **Semantic Analysis**: Find functional duplication beyond exact text matches  
- **Impact Assessment**: Prioritize refactoring based on lines saved and maintainability gains
- **Risk Evaluation**: Ensure refactoring preserves behavior and doesn't break tests

### Refactoring Principles
1. **Shared Over Duplicated**: Create reusable components where repetition is high
2. **Backward Compatible**: All existing APIs must continue to work
3. **Test-Driven**: Verify existing tests pass before and after each change
4. **Incremental**: Small, focused changes that can be validated independently

### Quality Gates
- **After Every Phase**: Run `make lint` and `make test-race` to ensure code quality and race condition detection
- **Before Integration**: Run full test suite including `make test-fuzz` and benchmarks
- **Regression Prevention**: Compare test output before and after changes

## Code Duplication Analysis

### High-Impact Duplication Found

#### 1. Mock Implementation Patterns (~170 lines)
**Locations**: `internal/gh/mock.go`, `internal/git/mock.go`, `internal/state/mock.go`
- Identical argument validation logic across all mock implementations
- Repeated error type assertions and fallback error creation
- Same pattern: `fmt.Errorf("mock not properly configured: expected %d return values, got %d")`

#### 2. Benchmark Test Setup (~200+ lines)  
**Locations**: Multiple `*_benchmark_test.go` files
- Memory allocation tracking setup duplication
- Repeated `b.ResetTimer()` / `b.StopTimer()` patterns
- File creation for testing: `os.WriteFile(filepath, []byte("test content"), 0o600)`
- Size-based test iterations (Small/Medium/Large patterns)

#### 3. Test File Creation (50+ occurrences)
**Locations**: `internal/git/`, `internal/transform/`, various test files
- Identical file creation patterns across multiple test files
- Repeated error handling for file operations in tests
- Similar cleanup patterns for temporary files

#### 4. Error Handling Patterns (~50+ instances)
**Locations**: Throughout codebase
- Repetitive error wrapping: `fmt.Errorf("failed to [operation]: %w", err)`
- Similar validation errors: `fmt.Errorf("invalid [field]: %s", value)`
- Consistent command failure error patterns

#### 5. JSON/YAML Processing
**Locations**: `internal/logging/`, `internal/transform/`, test files
- Similar marshaling/unmarshaling with identical error handling
- Repeated patterns for handling serialization errors
- Duplicated test data generation for JSON structures

#### 6. Validation Logic Patterns
**Locations**: `internal/config/validator.go`, `internal/cli/validate.go`
- Similar validation patterns for repository names, configurations
- Repeated input sanitization logic
- Scattered validation rules that could be centralized

## Implementation Phases

### Phase 1: Infrastructure Setup and Mock Consolidation (Days 1-2)

#### 1.1 Create Shared Mock Utilities Package
```go
// internal/testutil/mock.go
package testutil

import (
    "fmt"
    "github.com/stretchr/testify/mock"
)

// ValidateArgs validates mock arguments count
func ValidateArgs(args mock.Arguments, expectedCount int) error {
    if len(args) != expectedCount {
        return fmt.Errorf("mock not properly configured: expected %d return values, got %d", expectedCount, len(args))
    }
    return nil
}

// ExtractResult extracts a typed result from mock arguments
func ExtractResult[T any](args mock.Arguments, index int) (T, error) {
    var zero T
    if err := ValidateArgs(args, index+1); err != nil {
        return zero, err
    }
    
    result, ok := args[index].(T)
    if !ok {
        return zero, fmt.Errorf("mock result at index %d is not of expected type", index)
    }
    return result, nil
}

// ExtractError extracts error from mock arguments (typically last argument)
func ExtractError(args mock.Arguments) error {
    if len(args) == 0 {
        return nil
    }
    
    lastArg := args[len(args)-1]
    if lastArg == nil {
        return nil
    }
    
    err, ok := lastArg.(error)
    if !ok {
        return fmt.Errorf("last argument is not an error type")
    }
    return err
}
```

#### 1.2 Refactor Existing Mock Files
- Update `internal/gh/mock.go` to use shared utilities
- Update `internal/git/mock.go` to use shared utilities  
- Update `internal/state/mock.go` to use shared utilities
- Remove duplicated validation logic from each file

#### 1.3 Create Test File Utilities Package
```go
// internal/testutil/files.go
package testutil

import (
    "os"
    "path/filepath"
    "testing"
)

// CreateTestFiles creates multiple test files with default content
func CreateTestFiles(t *testing.T, dir string, count int) []string {
    t.Helper()
    
    files := make([]string, count)
    for i := 0; i < count; i++ {
        filePath := filepath.Join(dir, fmt.Sprintf("test_file_%d.txt", i))
        err := os.WriteFile(filePath, []byte("test content"), 0o600)
        if err != nil {
            t.Fatalf("failed to create test file %s: %v", filePath, err)
        }
        files[i] = filePath
    }
    return files
}

// CreateTestRepo creates a temporary repository directory with cleanup
func CreateTestRepo(t *testing.T) (string, func()) {
    t.Helper()
    
    dir, err := os.MkdirTemp("", "test_repo_*")
    if err != nil {
        t.Fatalf("failed to create temp directory: %v", err)
    }
    
    cleanup := func() {
        os.RemoveAll(dir)
    }
    
    return dir, cleanup
}

// WriteTestFile creates a single test file with custom content
func WriteTestFile(t *testing.T, filePath, content string) {
    t.Helper()
    
    err := os.WriteFile(filePath, []byte(content), 0o600)
    if err != nil {
        t.Fatalf("failed to create test file %s: %v", filePath, err)
    }
}
```

#### Phase 1 Deliverables
- [x] New `internal/testutil/mock.go` with shared mock utilities
- [x] New `internal/testutil/files.go` with test file helpers
- [x] Refactored `internal/gh/mock.go` using shared utilities
- [x] Refactored `internal/git/mock.go` using shared utilities
- [x] Refactored `internal/state/mock.go` using shared utilities
- [x] All tests pass with `make test-race`
- [x] Code quality maintained with `make lint`
- [x] Fix any linting errors or race conditions that arise

### Phase 2: Test Helper Consolidation (Days 3-4)

#### 2.1 Extend Benchmark Helpers
```go
// internal/benchmark/helpers.go (extend existing)

// SetupBenchmarkFiles creates files for benchmark testing
func SetupBenchmarkFiles(b *testing.B, dir string, count int) []string {
    b.Helper()
    
    files := make([]string, count)
    for i := 0; i < count; i++ {
        filePath := filepath.Join(dir, fmt.Sprintf("bench_file_%d.txt", i))
        err := os.WriteFile(filePath, []byte("benchmark test content"), 0o600)
        if err != nil {
            b.Fatalf("failed to create benchmark file: %v", err)
        }
        files[i] = filePath
    }
    return files
}

// WithMemoryTracking runs benchmark with memory allocation tracking
func WithMemoryTracking(b *testing.B, fn func()) {
    b.Helper()
    b.ReportAllocs()
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        fn()
    }
    
    b.StopTimer()
}

// StandardSizes returns consistent size configurations for benchmarks
func StandardSizes() []BenchmarkSize {
    return []BenchmarkSize{
        {Name: "Small", FileCount: 10, FileSize: 1024},
        {Name: "Medium", FileCount: 100, FileSize: 10240}, 
        {Name: "Large", FileCount: 1000, FileSize: 102400},
    }
}
```

#### 2.2 Refactor Benchmark Tests
- Update `internal/git/benchmark_test.go` to use shared helpers
- Update `internal/logging/benchmark_test.go` to use shared helpers
- Update `internal/transform/benchmark_test.go` to use shared helpers
- Update remaining benchmark tests to use consistent patterns

#### 2.3 Refactor Test File Creation
- Update `internal/git/concurrent_benchmark_test.go` (15+ file creation instances)
- Update `internal/git/batch_test.go` (10+ file creation instances)
- Replace manual file creation with `testutil.CreateTestFiles()`
- Replace manual repo setup with `testutil.CreateTestRepo()`

#### Phase 2 Deliverables
- [x] Extended `internal/benchmark/helpers.go` with file creation utilities
- [x] Refactored all benchmark tests to use shared helpers
- [x] Replaced 50+ file creation patterns with centralized utilities
- [x] All benchmark tests pass and show consistent performance
- [x] All tests pass with `make test-race`
- [x] Code quality maintained with `make lint`
- [x] Fix any linting errors or race conditions that arise

### Phase 3: Error Handling and Validation Standardization (Days 5-6)

#### 3.1 Extend Error Utilities
```go
// internal/errors/errors.go (extend existing)

// WrapWithContext wraps an error with operation context
func WrapWithContext(err error, operation string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("failed to %s: %w", operation, err)
}

// InvalidFieldError creates a standardized invalid field error
func InvalidFieldError(field, value string) error {
    return fmt.Errorf("invalid %s: %s", field, value)
}

// CommandFailedError creates a standardized command failure error
func CommandFailedError(cmd string, err error) error {
    return fmt.Errorf("command '%s' failed: %w", cmd, err)
}

// ValidationError creates a standardized validation error
func ValidationError(item, reason string) error {
    return fmt.Errorf("validation failed for %s: %s", item, reason)
}
```

#### 3.2 Create Shared Validation Package
```go
// internal/validation/validation.go
package validation

import (
    "regexp"
    "strings"
    "github.com/yourusername/go-broadcast/internal/errors"
)

var (
    repoNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`)
    branchNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
)

// ValidateRepoName validates repository name format
func ValidateRepoName(name string) error {
    if name == "" {
        return errors.InvalidFieldError("repository name", "cannot be empty")
    }
    
    if !repoNamePattern.MatchString(name) {
        return errors.InvalidFieldError("repository name", "must be in format owner/repo")
    }
    
    if strings.Contains(name, "..") {
        return errors.ValidationError("repository name", "contains path traversal")
    }
    
    return nil
}

// ValidateBranchName validates branch name format
func ValidateBranchName(name string) error {
    if name == "" {
        return errors.InvalidFieldError("branch name", "cannot be empty")
    }
    
    if !branchNamePattern.MatchString(name) {
        return errors.InvalidFieldError("branch name", "contains invalid characters")
    }
    
    return nil
}

// ValidateConfig validates configuration objects using reflection
func ValidateConfig(cfg interface{}) error {
    // Implementation would use reflection to validate struct fields
    // This consolidates validation logic from multiple packages
    return nil
}
```

#### 3.3 Refactor Error Handling Patterns
- Update `internal/git/git.go` to use standardized error utilities
- Update `internal/gh/github.go` to use standardized error utilities
- Update `internal/sync/engine.go` to use standardized error utilities
- Update `internal/config/validator.go` to use shared validation

#### Phase 3 Deliverables
- [x] Extended `internal/errors/errors.go` with context and validation utilities
- [x] New `internal/validation/validation.go` with shared validation logic
- [x] Refactored 50+ error creation patterns to use standardized utilities
- [x] Consolidated validation logic from multiple packages
- [x] All validation tests pass with consistent error messages
- [x] All tests pass with `make test-race`
- [x] Code quality maintained with `make lint`
- [x] Fix any linting errors or race conditions that arise

### Phase 4: JSON Processing and Benchmark Optimization (Days 7-8)

#### 4.1 Create JSON Utilities Package
```go
// internal/jsonutil/jsonutil.go
package jsonutil

import (
    "encoding/json"
    "fmt"
)

// MarshalJSON marshals any type to JSON with standardized error handling
func MarshalJSON[T any](v T) ([]byte, error) {
    data, err := json.Marshal(v)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
    }
    return data, nil
}

// UnmarshalJSON unmarshals JSON data to any type with standardized error handling
func UnmarshalJSON[T any](data []byte) (T, error) {
    var result T
    err := json.Unmarshal(data, &result)
    if err != nil {
        return result, fmt.Errorf("failed to unmarshal JSON: %w", err)
    }
    return result, nil
}

// GenerateTestJSON creates test JSON data for benchmarks and tests
func GenerateTestJSON(count int, template interface{}) []byte {
    items := make([]interface{}, count)
    for i := 0; i < count; i++ {
        items[i] = template
    }
    
    data, _ := json.Marshal(items)
    return data
}

// PrettyPrint formats JSON for human-readable output
func PrettyPrint(v interface{}) (string, error) {
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil {
        return "", fmt.Errorf("failed to pretty print JSON: %w", err)
    }
    return string(data), nil
}
```

#### 4.2 Optimize Benchmark Test Patterns
- Update benchmark tests to use `benchmark.WithMemoryTracking()`
- Replace manual `b.ResetTimer()` patterns with shared utilities
- Standardize benchmark naming and size configurations
- Use `benchmark.SetupBenchmarkFiles()` for consistent file creation

#### 4.3 Refactor JSON Processing
- Update `internal/logging/formatter.go` to use JSON utilities
- Update test files with JSON processing to use shared utilities
- Replace manual JSON marshaling/unmarshaling with type-safe utilities
- Update benchmark fixtures to use `GenerateTestJSON()`

#### Phase 4 Deliverables
- [x] New `internal/jsonutil/jsonutil.go` with type-safe JSON utilities
- [x] Optimized benchmark test patterns using shared utilities
- [x] Refactored JSON processing to use centralized utilities
- [x] Reduced JSON-related code duplication across packages
- [x] All JSON processing tests pass with improved error handling
- [x] Benchmark tests show consistent performance patterns
- [x] All tests pass with `make test-race`
- [x] Code quality maintained with `make lint`
- [x] Fix any linting errors or race conditions that arise

### Phase 5: Final Cleanup and Validation (Days 9-10)

#### 5.1 String Building Optimization
- Audit codebase for manual string concatenation with "+"
- Replace with optimized `transform.BuildPath()` where appropriate
- Promote usage of existing string building utilities
- Create additional string utilities if needed

#### 5.2 Test Pattern Standardization
```go
// internal/testutil/patterns.go
package testutil

import "testing"

// TestCase represents a generic test case structure
type TestCase[T any] struct {
    Name     string
    Input    T
    Expected interface{}
    WantErr  bool
}

// RunTableTests runs table-driven tests with consistent patterns
func RunTableTests[T any](t *testing.T, tests []TestCase[T], runner func(*testing.T, TestCase[T])) {
    t.Helper()
    
    for _, tt := range tests {
        t.Run(tt.Name, func(t *testing.T) {
            runner(t, tt)
        })
    }
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

// AssertError fails the test if err is nil when error is expected
func AssertError(t *testing.T, err error) {
    t.Helper()
    if err == nil {
        t.Fatal("expected error but got nil")
    }
}
```

#### 5.3 Final Integration and Validation
- Run complete test suite: `make test-race`
- Run linting: `make lint`
- Run fuzzing tests: `make test-fuzz`
- Run benchmark tests to ensure performance is maintained
- Verify all refactored code maintains original behavior
- Fix any linting errors or race conditions that arise

#### 5.4 Documentation and Maintenance
- Update package documentation for new shared utilities
- Create usage examples for shared patterns
- Document refactoring decisions and patterns
- Establish guidelines for preventing future duplication

#### Phase 5 Deliverables
- [x] Replaced remaining manual string concatenation patterns
- [x] Standardized test patterns using shared utilities
- [x] Complete test suite passes (unit, integration, fuzz, benchmarks)
- [x] All tests pass with `make test-race`
- [x] All code quality checks pass (lint, format, vet)
- [x] Performance benchmarks show no regression
- [x] Documentation updated for new shared utilities
- [x] Maintenance guidelines established
- [x] Fix any linting errors or race conditions that arise

## Success Criteria

### Code Reduction Metrics
- **500+ lines** of duplicated code eliminated
- **170 lines** saved from mock consolidation
- **200+ lines** saved from benchmark test optimization
- **50+ instances** of error handling standardized
- **50+ file creation patterns** centralized

### Quality Metrics
- **100%** of existing tests continue to pass
- **0** performance regressions in benchmarks
- **100%** lint compliance maintained
- **0** breaking changes to public APIs

### Maintainability Metrics
- **Shared utilities** available for future use
- **Consistent patterns** across all packages
- **Centralized validation** and error handling
- **Reusable test helpers** for new test development

## Testing Strategy

### Regression Prevention
1. **Before each phase**: Record current test output and performance baselines
2. **During refactoring**: Run affected tests after each significant change
3. **After each phase**: Run `make lint` and `make test-race` to compare full test suite output to baseline
4. **Final validation**: Run extended test suite including integration and fuzz tests
5. **Error handling**: Address any linting errors or race conditions immediately when they arise

### Performance Validation
1. **Benchmark comparison**: Ensure refactored code maintains performance
2. **Memory usage**: Verify shared utilities don't increase memory overhead
3. **Test execution time**: Confirm test suite performance is maintained or improved

### Integration Testing
1. **Cross-package compatibility**: Verify shared utilities work across all packages
2. **Mock functionality**: Ensure consolidated mocks maintain all required behavior
3. **Error handling consistency**: Validate error messages and types remain consistent

## Implementation Timeline

### Week 1 (Days 1-5)
- **Day 1-2**: Infrastructure setup and mock consolidation (Phase 1)
- **Day 3-4**: Test helper consolidation and file operations (Phase 2)  
- **Day 5**: Error handling and validation standardization (Phase 3)

### Week 2 (Days 6-10)
- **Day 6-7**: JSON processing and benchmark optimization (Phase 4)
- **Day 8-9**: Final cleanup and string building optimization (Phase 5)
- **Day 10**: Complete validation, documentation, and maintenance guidelines

## Maintenance and Evolution

### Ongoing Practices
1. **Code Reviews**: Check for duplication patterns in new code
2. **Shared Utilities**: Encourage use of centralized patterns
3. **Regular Audits**: Quarterly reviews for new duplication opportunities
4. **Pattern Documentation**: Maintain examples of preferred patterns

### Prevention Strategies
1. **Pre-commit Hooks**: Automated checks for common duplication patterns
2. **Linting Rules**: Custom rules to detect repeated code patterns
3. **Team Guidelines**: Documentation of approved shared utilities
4. **Refactoring Opportunities**: Track and prioritize future consolidation

## Conclusion

This comprehensive code deduplication plan will eliminate over 500 lines of duplicated code while improving maintainability, consistency, and code quality across the go-broadcast project. By creating shared utilities for common patterns like mocking, testing, error handling, and JSON processing, we establish a foundation for cleaner, more maintainable code that follows consistent patterns throughout the codebase.

The phased approach ensures that all existing functionality is preserved while incrementally improving code quality. The extensive testing strategy and success criteria provide confidence that the refactoring will enhance rather than compromise the project's stability and performance.