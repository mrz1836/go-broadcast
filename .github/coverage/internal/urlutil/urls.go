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
