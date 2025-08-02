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
func HasPathTraversal(path string) bool {
	cleanPath := filepath.Clean(path)
	// Check if cleaned path starts with .. (escapes upward)
	if strings.HasPrefix(cleanPath, "..") {
		return true
	}
	// Check for .. as a path component (not just substring)
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

// EnsureTrailingSlash ensures a path ends with a trailing slash.
// This consolidates the common pattern for directory paths.
func EnsureTrailingSlash(path string) string {
	if IsEmpty(path) {
		return "/"
	}
	if !strings.HasSuffix(path, "/") {
		return path + "/"
	}
	return path
}

// RemoveTrailingSlash removes trailing slashes from a path.
// This consolidates the common pattern for normalizing paths.
func RemoveTrailingSlash(path string) string {
	return strings.TrimSuffix(path, "/")
}

// SplitPath splits a path into its directory components.
// This consolidates the common pattern: strings.Split(filepath.ToSlash(path), "/")
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
	return result
}
