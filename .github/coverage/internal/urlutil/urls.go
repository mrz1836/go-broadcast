// Package urlutil provides utility functions for URL generation and formatting
package urlutil

import (
	"fmt"
	"strings"
)

// BuildGitHubCommitURL builds a GitHub commit URL from repository info and commit SHA
func BuildGitHubCommitURL(owner, repo, commitSHA string) string {
	if owner == "" || repo == "" || commitSHA == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s/commit/%s", owner, repo, commitSHA)
}

// BuildCoverageBadgeURL builds a coverage badge URL using shields.io
func BuildCoverageBadgeURL(percentage float64) string {
	// Determine color based on percentage
	var color string
	switch {
	case percentage >= 90:
		color = "brightgreen"
	case percentage >= 80:
		color = "green"
	case percentage >= 70:
		color = "yellowgreen"
	case percentage >= 60:
		color = "yellow"
	case percentage >= 50:
		color = "orange"
	default:
		color = "red"
	}

	return fmt.Sprintf("https://img.shields.io/badge/coverage-%.1f%%25-%s", percentage, color)
}

// BuildGitHubRepoURL builds a GitHub repository URL from owner and repo name
func BuildGitHubRepoURL(owner, repo string) string {
	if owner == "" || repo == "" {
		return ""
	}
	return fmt.Sprintf("https://github.com/%s/%s", owner, repo)
}

// ExtractRepoNameFromURL extracts just the repository name from a full repository name
func ExtractRepoNameFromURL(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return fullName
}

// CleanModulePath removes the Go module prefix from a file path and cleans up any erroneous prefixes
// e.g., "github.com/mrz1836/go-broadcast/internal/algorithms/file.go" -> "internal/algorithms/file.go"
// e.g., "github.com/mrz1836/go-broadcast/cli/internal/cli/cancel.go" -> "internal/cli/cancel.go"
func CleanModulePath(fullPath string) string {
	// Try to match the pattern github.com/owner/repo/...
	// We want to strip everything up to and including the repo name
	parts := strings.Split(fullPath, "/")

	// Look for github.com pattern
	for i := 0; i < len(parts); i++ {
		if parts[i] == "github.com" && i+2 < len(parts) {
			// Skip github.com/owner/repo and return the rest
			if i+3 < len(parts) {
				remainder := strings.Join(parts[i+3:], "/")

				// Special handling for paths that might have extra prefixes
				// like "cli/internal/cli/cancel.go" -> "internal/cli/cancel.go"
				// This happens when the Go module analysis includes extra path components
				return cleanupExtraPathPrefixes(remainder)
			}
		}
	}

	// If no github.com pattern found, return the original path
	return fullPath
}

// BuildGitHubFileURL builds a GitHub file URL with automatic path cleaning
func BuildGitHubFileURL(owner, repo, branch, filePath string) string {
	if owner == "" || repo == "" || branch == "" || filePath == "" {
		return ""
	}
	cleanPath := CleanModulePath(filePath)
	return fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, branch, cleanPath)
}

// cleanupExtraPathPrefixes removes common erroneous path prefixes
// that can appear in Go module analysis
func cleanupExtraPathPrefixes(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return path
	}

	// If we have a pattern like "cli/internal/cli/..." where the first
	// component is repeated later, remove the first occurrence
	if len(parts) >= 3 {
		firstPart := parts[0]
		// Look for the first part appearing again in a reasonable position
		for i := 1; i < len(parts)-1; i++ {
			if parts[i] == firstPart {
				// Found a repetition, return everything from the second occurrence
				return strings.Join(parts[1:], "/")
			}
		}
	}

	return path
}
