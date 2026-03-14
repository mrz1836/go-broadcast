package strutil

import "strings"

// IsEmptySlice checks if a slice is nil or has zero length.
// This consolidates the common pattern: len(slice) == 0
func IsEmptySlice[T any](slice []T) bool {
	return len(slice) == 0
}

// IsNotEmptySlice checks if a slice is not nil and has at least one element.
// This consolidates the common pattern: len(slice) > 0
func IsNotEmptySlice[T any](slice []T) bool {
	return len(slice) > 0
}

// SafeSliceAccess safely accesses a slice element with bounds checking.
// Returns the element and true if index is valid, zero value and false otherwise.
func SafeSliceAccess[T any](slice []T, index int) (T, bool) {
	var zero T
	if index < 0 || index >= len(slice) {
		return zero, false
	}
	return slice[index], true
}

// FilterNonEmpty filters out empty strings from a slice and trims whitespace.
// Both filtering and trimming are applied: empty/whitespace-only strings are removed,
// and remaining strings have leading/trailing whitespace trimmed.
// Returns nil for nil or empty input slices.
func FilterNonEmpty(slice []string) []string {
	if IsEmptySlice(slice) {
		return nil
	}

	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if IsNotEmpty(item) {
			result = append(result, strings.TrimSpace(item))
		}
	}
	return result
}

// UniqueStrings removes duplicate strings from a slice while preserving order.
// This consolidates the common pattern of deduplicating slices.
func UniqueStrings(slice []string) []string {
	if IsEmptySlice(slice) {
		return nil
	}

	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

// ChunkSlice splits a slice into chunks of the specified size.
// This consolidates the common pattern of batch processing slices.
// Each returned chunk is an independent copy, safe to modify without affecting
// the original slice or other chunks.
func ChunkSlice[T any](slice []T, chunkSize int) [][]T {
	if IsEmptySlice(slice) || chunkSize <= 0 {
		return nil
	}

	chunks := make([][]T, 0, (len(slice)+chunkSize-1)/chunkSize)
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		// Create independent copy to prevent mutations from affecting original
		chunk := make([]T, end-i)
		copy(chunk, slice[i:end])
		chunks = append(chunks, chunk)
	}
	return chunks
}
