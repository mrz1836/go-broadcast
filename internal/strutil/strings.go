// Package strutil provides common string and path utility functions used throughout the application.
// This package consolidates utility logic that was previously scattered across multiple packages.
package strutil

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// IsEmpty checks if a string is empty or contains only whitespace.
// This consolidates the common pattern: strings.TrimSpace(value) == ""
func IsEmpty(value string) bool {
	return strings.TrimSpace(value) == ""
}

// IsNotEmpty checks if a string is not empty and contains non-whitespace characters.
// This consolidates the common pattern: strings.TrimSpace(value) != ""
func IsNotEmpty(value string) bool {
	return strings.TrimSpace(value) != ""
}

// EmptyToDefault returns the defaultValue if the input is empty or whitespace-only.
// This consolidates the common pattern of providing defaults for empty strings.
func EmptyToDefault(value, defaultValue string) string {
	if IsEmpty(value) {
		return defaultValue
	}
	return strings.TrimSpace(value)
}

// TrimAndLower trims whitespace and converts to lowercase.
// This consolidates a common pattern for normalizing user input.
func TrimAndLower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// ContainsAny checks if a string contains any of the provided substrings.
// This consolidates repetitive multiple strings.Contains calls.
func ContainsAny(text string, substrings ...string) bool {
	for _, substring := range substrings {
		if strings.Contains(text, substring) {
			return true
		}
	}
	return false
}

// HasAnyPrefix checks if a string has any of the provided prefixes.
// This consolidates repetitive multiple strings.HasPrefix calls.
func HasAnyPrefix(text string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

// HasAnySuffix checks if a string has any of the provided suffixes.
// This consolidates repetitive multiple strings.HasSuffix calls.
func HasAnySuffix(text string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(text, suffix) {
			return true
		}
	}
	return false
}

// FormatRepoName formats a repository name as "org/repo".
// This consolidates the common pattern: fmt.Sprintf("org/%s", name)
func FormatRepoName(org, repo string) string {
	return fmt.Sprintf("%s/%s", org, repo)
}

// FormatFilePath formats a file path with proper separators.
// This consolidates common path formatting patterns.
func FormatFilePath(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}
	return filepath.Join(parts...)
}

// NormalizePath normalizes a path by cleaning it and converting to forward slashes.
// This consolidates the common patterns of filepath.Clean and filepath.ToSlash.
func NormalizePath(path string) string {
	return filepath.ToSlash(filepath.Clean(path))
}

// SanitizeForFilename sanitizes a string to be safe for use as a filename.
// This consolidates the common pattern of replacing problematic characters.
func SanitizeForFilename(name string) string {
	// Replace common problematic characters
	sanitized := strings.ReplaceAll(name, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, "\\", "-")
	sanitized = strings.ReplaceAll(sanitized, ":", "-")
	sanitized = strings.ReplaceAll(sanitized, "\"", "-")
	sanitized = strings.ReplaceAll(sanitized, "<", "-")
	sanitized = strings.ReplaceAll(sanitized, ">", "-")
	sanitized = strings.ReplaceAll(sanitized, "|", "-")
	sanitized = strings.ReplaceAll(sanitized, "?", "-")
	sanitized = strings.ReplaceAll(sanitized, "*", "-")
	return strings.TrimSpace(sanitized)
}

// IsValidGitHubURL validates if a URL is a valid GitHub URL.
// This consolidates the common pattern of validating GitHub URLs.
func IsValidGitHubURL(rawURL string) bool {
	if IsEmpty(rawURL) {
		return false
	}

	// Check for path traversal attempts
	if strings.Contains(rawURL, "..") {
		return false
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Must be HTTPS and github.com
	return parsedURL.Scheme == "https" && parsedURL.Host == "github.com"
}

// ReplaceTemplateVars replaces template variables in content.
// This consolidates the common pattern of multiple strings.ReplaceAll calls.
func ReplaceTemplateVars(content string, replacements map[string]string) string {
	result := content
	for placeholder, replacement := range replacements {
		result = strings.ReplaceAll(result, placeholder, replacement)
	}
	return result
}
