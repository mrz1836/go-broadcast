// Package jsonutil provides type-safe JSON utilities with standardized error handling.
// This package consolidates JSON processing patterns to reduce code duplication
// across the go-broadcast codebase.
package jsonutil

import (
	"encoding/json"
	"errors"
	"fmt"

	appErrors "github.com/mrz1836/go-broadcast/internal/errors"
)

// MarshalJSON marshals any type to JSON with standardized error handling.
// It provides a type-safe wrapper around json.Marshal with consistent error messages.
func MarshalJSON[T any](v T) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "marshal to JSON")
	}
	return data, nil
}

// UnmarshalJSON unmarshals JSON data to any type with standardized error handling.
// It provides a type-safe wrapper around json.Unmarshal with consistent error messages.
func UnmarshalJSON[T any](data []byte) (T, error) {
	var result T
	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, appErrors.WrapWithContext(err, "unmarshal JSON")
	}
	return result, nil
}

// Static error for invalid count
var errNegativeCount = errors.New("count must be non-negative")

// GenerateTestJSON creates test JSON data for benchmarks and tests.
// It generates an array of count items based on the provided template.
// This is useful for creating consistent test data across benchmark tests.
func GenerateTestJSON(count int, template interface{}) ([]byte, error) {
	if count < 0 {
		return nil, fmt.Errorf("%w: got %d", errNegativeCount, count)
	}

	items := make([]interface{}, count)
	for i := 0; i < count; i++ {
		items[i] = template
	}

	data, err := json.Marshal(items)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "generate test JSON")
	}
	return data, nil
}

// PrettyPrint formats JSON for human-readable output with proper indentation.
// It returns a formatted string representation of the provided value.
func PrettyPrint(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", appErrors.WrapWithContext(err, "pretty print JSON")
	}
	return string(data), nil
}

// CompactJSON removes unnecessary whitespace from JSON data.
// This is useful for minimizing JSON size for storage or transmission.
func CompactJSON(data []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, appErrors.WrapWithContext(err, "parse JSON for compaction")
	}

	compact, err := json.Marshal(v)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "compact JSON")
	}
	return compact, nil
}

// MergeJSON merges multiple JSON objects into a single object.
// Later values override earlier values for the same keys.
// All inputs must be valid JSON objects (not arrays or primitives).
func MergeJSON(jsons ...[]byte) ([]byte, error) {
	result := make(map[string]interface{})

	for i, data := range jsons {
		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON at index %d: %w", i, err)
		}

		// Merge the object into result
		for k, v := range obj {
			result[k] = v
		}
	}

	merged, err := json.Marshal(result)
	if err != nil {
		return nil, appErrors.WrapWithContext(err, "marshal merged JSON")
	}
	return merged, nil
}
