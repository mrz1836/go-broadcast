// Package jsonutil provides type-safe JSON utilities with standardized error handling.
// This package consolidates JSON processing patterns to reduce code duplication
// across the go-broadcast codebase.
package jsonutil

import (
	"bytes"
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
// Returns an error if data is nil or empty.
func UnmarshalJSON[T any](data []byte) (T, error) {
	var result T
	if len(data) == 0 {
		return result, appErrors.WrapWithContext(errEmptyInput, "unmarshal JSON")
	}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, appErrors.WrapWithContext(err, "unmarshal JSON")
	}
	return result, nil
}

// Static errors for validation
var (
	errNegativeCount = errors.New("count must be non-negative")
	errCountTooLarge = errors.New("count exceeds maximum allowed")
	errEmptyInput    = errors.New("empty input")
	errNilJSONData   = errors.New("nil JSON data")
	errEmptyJSONData = errors.New("empty JSON data")
	errNullJSON      = errors.New("JSON is null, expected object")
)

// MaxGenerateCount is the maximum number of items GenerateTestJSON will create.
// This prevents potential OOM conditions from extremely large count values.
const MaxGenerateCount = 100_000

// GenerateTestJSON creates test JSON data for benchmarks and tests.
// It generates an array of count items based on the provided template.
// This is useful for creating consistent test data across benchmark tests.
//
// All items share the same template reference. If the template
// contains mutable types (pointers, maps, slices), they will be shared.
// For independent copies, callers should use value types or copy manually.
//
// Returns an error if count is negative or exceeds MaxGenerateCount.
func GenerateTestJSON(count int, template interface{}) ([]byte, error) {
	if count < 0 {
		return nil, fmt.Errorf("%w: got %d", errNegativeCount, count)
	}
	if count > MaxGenerateCount {
		return nil, fmt.Errorf("%w: %d (max: %d)", errCountTooLarge, count, MaxGenerateCount)
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
// Returns an error if data is nil or empty, or if it's not valid JSON.
func CompactJSON(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, appErrors.WrapWithContext(errEmptyInput, "compact JSON")
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		return nil, appErrors.WrapWithContext(err, "compact JSON")
	}
	return buf.Bytes(), nil
}

// MergeJSON merges multiple JSON objects into a single object.
// Later values override earlier values for the same keys.
// All inputs must be valid JSON objects (not arrays, primitives, or null).
// Returns an error if any input is nil, empty, or not a JSON object.
func MergeJSON(jsons ...[]byte) ([]byte, error) {
	result := make(map[string]interface{})

	for i, data := range jsons {
		if data == nil {
			return nil, fmt.Errorf("%w at index %d", errNilJSONData, i)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("%w at index %d", errEmptyJSONData, i)
		}

		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON at index %d: %w", i, err)
		}

		// Check for JSON null which unmarshals to nil map
		if obj == nil {
			return nil, fmt.Errorf("%w at index %d", errNullJSON, i)
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
