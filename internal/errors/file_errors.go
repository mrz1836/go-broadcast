// Package errors - file operation error utilities
package errors

import (
	"errors"
	"fmt"
)

// Error templates for file operations
var (
	errFileOperationTemplate      = errors.New("file operation failed")
	errDirectoryOperationTemplate = errors.New("directory operation failed")
	errJSONOperationTemplate      = errors.New("JSON operation failed")
	errBatchOperationTemplate     = errors.New("batch operation failed")
)

// FileOperationError creates a standardized file operation error.
// This consolidates patterns like fmt.Errorf("failed to %s file %s: %w", operation, path, err).
//
// Example usage:
//
//	return FileOperationError("read", "/path/to/file.txt", err)
//	// Returns: "file operation failed: read '/path/to/file.txt': <original error>"
func FileOperationError(operation, path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s '%s': %w", errFileOperationTemplate, operation, path, err)
}

// DirectoryOperationError creates a standardized directory operation error.
// This consolidates patterns like fmt.Errorf("failed to %s directory %s: %w", operation, path, err).
//
// Example usage:
//
//	return DirectoryOperationError("create", "/path/to/dir", err)
//	// Returns: "directory operation failed: create '/path/to/dir': <original error>"
func DirectoryOperationError(operation, path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s '%s': %w", errDirectoryOperationTemplate, operation, path, err)
}

// FileReadError is a convenience function for file read operations.
// This replaces fmt.Errorf("failed to read file %s: %w", path, err).
func FileReadError(path string, err error) error {
	return FileOperationError("read", path, err)
}

// FileWriteError is a convenience function for file write operations.
// This replaces fmt.Errorf("failed to write file %s: %w", path, err).
func FileWriteError(path string, err error) error {
	return FileOperationError("write", path, err)
}

// FileOpenError is a convenience function for file open operations.
// This replaces fmt.Errorf("failed to open file %s: %w", path, err).
func FileOpenError(path string, err error) error {
	return FileOperationError("open", path, err)
}

// FileCreateError is a convenience function for file creation operations.
// This replaces fmt.Errorf("failed to create file %s: %w", path, err).
func FileCreateError(path string, err error) error {
	return FileOperationError("create", path, err)
}

// FileDeleteError is a convenience function for file deletion operations.
// This replaces fmt.Errorf("failed to delete file %s: %w", path, err).
func FileDeleteError(path string, err error) error {
	return FileOperationError("delete", path, err)
}

// DirectoryCreateError is a convenience function for directory creation.
// This replaces fmt.Errorf("failed to create directory %s: %w", path, err).
func DirectoryCreateError(path string, err error) error {
	return DirectoryOperationError("create", path, err)
}

// DirectoryWalkError is a convenience function for directory walking errors.
// This replaces fmt.Errorf("failed to walk directory %s: %w", path, err).
func DirectoryWalkError(path string, err error) error {
	return DirectoryOperationError("walk", path, err)
}

// JSONOperationError creates a standardized JSON processing error.
// This consolidates patterns like fmt.Errorf("failed to %s JSON: %w", operation, err).
//
// Example usage:
//
//	return JSONOperationError("marshal", "user data", err)
//	// Returns: "JSON operation failed: marshal 'user data': <original error>"
func JSONOperationError(operation, context string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s '%s': %w", errJSONOperationTemplate, operation, context, err)
}

// JSONMarshalError is a convenience function for JSON marshal errors.
// This replaces fmt.Errorf("failed to marshal %s: %w", context, err).
func JSONMarshalError(context string, err error) error {
	return JSONOperationError("marshal", context, err)
}

// JSONUnmarshalError is a convenience function for JSON unmarshal errors.
// This replaces fmt.Errorf("failed to unmarshal %s: %w", context, err).
func JSONUnmarshalError(context string, err error) error {
	return JSONOperationError("unmarshal", context, err)
}

// BatchOperationError creates a standardized batch operation error.
// This consolidates patterns for batch processing failures.
//
// Example usage:
//
//	return BatchOperationError("process", 0, 10, err)
//	// Returns: "batch operation failed: process items 0-9: <original error>"
func BatchOperationError(operation string, start, end int, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s items %d-%d: %w", errBatchOperationTemplate, operation, start, end-1, err)
}
