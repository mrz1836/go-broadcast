// Package helpers provides test utilities for the go-broadcast integration tests.
package helpers

import (
	"os"
	"testing"
)

// SkipIfNoGitHubAuth skips the test if GitHub authentication is not available.
// This allows tests to run in CI environments without GitHub authentication.
func SkipIfNoGitHubAuth(t *testing.T) {
	t.Helper()
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN not set, skipping test that requires GitHub authentication")
	}
}
