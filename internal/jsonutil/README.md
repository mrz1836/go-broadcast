# jsonutil Package

The `jsonutil` package provides type-safe JSON utilities with standardized error handling for the go-broadcast codebase. This package consolidates JSON processing patterns to reduce code duplication and ensure consistent error handling across the project.

## Features

- **Type-safe JSON marshaling and unmarshaling** - Generic functions that preserve type safety
- **Standardized error handling** - Consistent error messages with proper wrapping
- **Test data generation** - Utilities for creating JSON test data for benchmarks
- **Pretty printing** - Human-readable JSON formatting
- **JSON compaction** - Remove unnecessary whitespace from JSON
- **JSON merging** - Merge multiple JSON objects into one

## Functions

### MarshalJSON[T any](v T) ([]byte, error)

Marshals any type to JSON with standardized error handling.

```go
type Person struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

person := Person{Name: "Alice", Age: 30}
data, err := jsonutil.MarshalJSON(person)
if err != nil {
    // Handle error
}
```

### UnmarshalJSON[T any](data []byte) (T, error)

Unmarshals JSON data to any type with standardized error handling.

```go
data := []byte(`{"name":"Bob","age":25}`)
person, err := jsonutil.UnmarshalJSON[Person](data)
if err != nil {
    // Handle error
}
```

### GenerateTestJSON(count int, template interface{}) ([]byte, error)

Creates test JSON data for benchmarks and tests by generating an array of items based on the provided template.

```go
template := map[string]interface{}{
    "id":   123,
    "name": "test",
}
data, err := jsonutil.GenerateTestJSON(100, template)
// Creates an array with 100 identical objects
```

### PrettyPrint(v interface{}) (string, error)

Formats JSON for human-readable output with proper indentation.

```go
config := map[string]interface{}{
    "host": "localhost",
    "port": 8080,
}
pretty, err := jsonutil.PrettyPrint(config)
// Output:
// {
//   "host": "localhost",
//   "port": 8080
// }
```

### CompactJSON(data []byte) ([]byte, error)

Removes unnecessary whitespace from JSON data to minimize size.

```go
input := []byte(`{
    "name": "test",
    "value": 42
}`)
compact, err := jsonutil.CompactJSON(input)
// Output: {"name":"test","value":42}
```

### MergeJSON(jsons ...[]byte) ([]byte, error)

Merges multiple JSON objects into a single object. Later values override earlier values for the same keys.

```go
json1 := []byte(`{"a":1,"b":2}`)
json2 := []byte(`{"b":3,"c":4}`)
merged, err := jsonutil.MergeJSON(json1, json2)
// Output: {"a":1,"b":3,"c":4}
```

## Migration Guide

To migrate existing code to use jsonutil:

### Before:
```go
import "encoding/json"

// Marshaling
data, err := json.Marshal(obj)
if err != nil {
    return fmt.Errorf("failed to marshal: %w", err)
}

// Unmarshaling
var result MyType
if err := json.Unmarshal(data, &result); err != nil {
    return fmt.Errorf("failed to unmarshal: %w", err)
}
```

### After:
```go
import "github.com/mrz1836/go-broadcast/internal/jsonutil"

// Marshaling
data, err := jsonutil.MarshalJSON(obj)
if err != nil {
    return err // Error already wrapped
}

// Unmarshaling
result, err := jsonutil.UnmarshalJSON[MyType](data)
if err != nil {
    return err // Error already wrapped
}
```

## Performance

The jsonutil functions add minimal overhead compared to direct json package usage. The main benefits are:
- Consistent error handling
- Type safety for unmarshaling
- Reduced boilerplate code

Benchmark results show negligible performance difference compared to standard library functions.

## Usage in Tests

The package is particularly useful in tests and benchmarks:

```go
func BenchmarkProcessJSON(b *testing.B) {
    // Generate test data
    testData, _ := jsonutil.GenerateTestJSON(1000, map[string]int{"value": 42})
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Process the JSON data
        var items []map[string]int
        _, _ = jsonutil.UnmarshalJSON[[]map[string]int](testData)
    }
}
```

## Error Handling

All functions return errors that are already wrapped with context using the internal errors package. This means you don't need to wrap the errors again:

```go
data, err := jsonutil.MarshalJSON(obj)
if err != nil {
    // err already contains context like "failed to marshal to JSON: <original error>"
    return err
}
```