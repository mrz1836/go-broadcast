package strutil

import (
	"path/filepath"
	"strings"
)

// JoinPath safely joins path elements and normalizes the result.
// This consolidates the common pattern: filepath.Join + filepath.Clean
func JoinPath(elements ...string) string {
	if len(elements) == 0 {
		return ""
	}
	return filepath.Clean(filepath.Join(elements...))
}

// GetBaseName extracts the base name from a path.
// This consolidates the common pattern: filepath.Base(path)
func GetBaseName(path string) string {
	if IsEmpty(path) {
		return ""
	}
	return filepath.Base(path)
}

// GetDirName extracts the directory name from a path.
// This consolidates the common pattern: filepath.Dir(path)
func GetDirName(path string) string {
	if IsEmpty(path) {
		return ""
	}
	return filepath.Dir(path)
}

// IsAbsolutePath checks if a path is absolute.
// This consolidates the common pattern: filepath.IsAbs(path)
func IsAbsolutePath(path string) bool {
	return filepath.IsAbs(path)
}

// HasPathTraversal checks if a path contains path traversal attempts.
// This consolidates the common security check pattern.
// A path is considered to have traversal if:
// - For absolute paths: any ".." component exists (even if it doesn't escape root)
// - For relative paths: it escapes upward from its starting point
func HasPathTraversal(path string) bool {
	// Normalize separators for consistent checking
	normalizedPath := filepath.ToSlash(path)

	// For absolute paths, any ".." component is suspicious
	if strings.HasPrefix(normalizedPath, "/") {
		parts := strings.Split(normalizedPath, "/")
		for _, part := range parts {
			if part == ".." {
				return true
			}
		}
		return false
	}

	// For relative paths: check if cleaned path escapes upward (starts with "..")
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "..") {
		return true
	}

	// Check for ".." as a component in cleaned path
	parts := strings.Split(cleanPath, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}

	return false
}

// IsHiddenFile checks if a file or directory is hidden (starts with dot).
// This consolidates the common pattern for detecting hidden files.
func IsHiddenFile(path string) bool {
	base := GetBaseName(path)
	return strings.HasPrefix(base, ".") && base != "." && base != ".."
}

// ToUnixPath converts a path to Unix-style forward slashes.
// This consolidates the common pattern: filepath.ToSlash(path)
func ToUnixPath(path string) string {
	return filepath.ToSlash(path)
}

// HasExtension checks if a path has any of the specified extensions.
// This consolidates the common pattern of checking multiple extensions.
func HasExtension(path string, extensions ...string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, validExt := range extensions {
		if strings.ToLower(validExt) == ext {
			return true
		}
	}
	return false
}

// EnsureTrailingSlash ensures a path ends with a forward slash (/).
// This consolidates the common pattern for directory paths and URLs.
// Always uses forward slash regardless of platform, suitable for URLs
// and normalized paths. Returns "/" for empty input.
func EnsureTrailingSlash(path string) string {
	if IsEmpty(path) {
		return "/"
	}
	if !strings.HasSuffix(path, "/") {
		return path + "/"
	}
	return path
}

// RemoveTrailingSlash removes all trailing slashes from a path.
// This consolidates the common pattern for normalizing paths.
func RemoveTrailingSlash(path string) string {
	return strings.TrimRight(path, "/")
}

// SplitPath splits a path into its directory components.
// This consolidates the common pattern: strings.Split(filepath.ToSlash(path), "/")
// For absolute paths (e.g., "/path/to/file"), the leading "/" is not preserved
// in the result. Use IsAbsolutePath to check if the original path was absolute.
// Returns nil for empty paths or paths with no components (e.g., "/").
func SplitPath(path string) []string {
	if IsEmpty(path) {
		return nil
	}

	normalizedPath := ToUnixPath(filepath.Clean(path))
	parts := strings.Split(normalizedPath, "/")

	// Filter out empty parts
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}

	// Return nil for paths with no components (e.g., "/" or ".")
	if len(result) == 0 {
		return nil
	}

	return result
}
