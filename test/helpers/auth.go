// Package helpers provides test utilities for the go-broadcast integration tests.
package helpers

import (
	"os"
	"testing"
)

// GetGitHubToken returns the GitHub token from environment variables.
// It checks GH_PAT_TOKEN first (preferred), then falls back to GITHUB_TOKEN.
func GetGitHubToken() string {
	if token := os.Getenv("GH_PAT_TOKEN"); token != "" {
		return token
	}
	return os.Getenv("GITHUB_TOKEN")
}

// SkipIfNoGitHubAuth skips the test if GitHub authentication is not available.
// This allows tests to run in CI environments without GitHub authentication.
func SkipIfNoGitHubAuth(t *testing.T) {
	t.Helper()
	if GetGitHubToken() == "" {
		t.Skip("GH_PAT_TOKEN or GITHUB_TOKEN not set, skipping test that requires GitHub authentication")
	}
}
