package helpers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetGitHubTokenPrefersPAT tests that GetGitHubToken prefers GH_PAT_TOKEN over GITHUB_TOKEN
func TestGetGitHubTokenPrefersPAT(t *testing.T) {
	// Save original environment
	originalPAT := os.Getenv("GH_PAT_TOKEN")
	originalGitHub := os.Getenv("GITHUB_TOKEN")

	// Restore environment after test
	defer func() {
		if originalPAT != "" {
			require.NoError(t, os.Setenv("GH_PAT_TOKEN", originalPAT))
		} else {
			require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
		}
		if originalGitHub != "" {
			require.NoError(t, os.Setenv("GITHUB_TOKEN", originalGitHub))
		} else {
			require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
		}
	}()

	// Test both tokens set - should prefer GH_PAT_TOKEN
	require.NoError(t, os.Setenv("GH_PAT_TOKEN", "pat_token_123"))
	require.NoError(t, os.Setenv("GITHUB_TOKEN", "github_token_456"))

	token := GetGitHubToken()
	assert.Equal(t, "pat_token_123", token)
}

// TestGetGitHubTokenFallsBackToGitHubToken tests fallback to GITHUB_TOKEN when GH_PAT_TOKEN is not set
func TestGetGitHubTokenFallsBackToGitHubToken(t *testing.T) {
	// Save original environment
	originalPAT := os.Getenv("GH_PAT_TOKEN")
	originalGitHub := os.Getenv("GITHUB_TOKEN")

	// Restore environment after test
	defer func() {
		if originalPAT != "" {
			require.NoError(t, os.Setenv("GH_PAT_TOKEN", originalPAT))
		} else {
			require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
		}
		if originalGitHub != "" {
			require.NoError(t, os.Setenv("GITHUB_TOKEN", originalGitHub))
		} else {
			require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
		}
	}()

	// Clear GH_PAT_TOKEN and set GITHUB_TOKEN
	require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
	require.NoError(t, os.Setenv("GITHUB_TOKEN", "github_token_456"))

	token := GetGitHubToken()
	assert.Equal(t, "github_token_456", token)
}

// TestGetGitHubTokenEmptyPATFallsBack tests that empty GH_PAT_TOKEN falls back to GITHUB_TOKEN
func TestGetGitHubTokenEmptyPATFallsBack(t *testing.T) {
	// Save original environment
	originalPAT := os.Getenv("GH_PAT_TOKEN")
	originalGitHub := os.Getenv("GITHUB_TOKEN")

	// Restore environment after test
	defer func() {
		if originalPAT != "" {
			require.NoError(t, os.Setenv("GH_PAT_TOKEN", originalPAT))
		} else {
			require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
		}
		if originalGitHub != "" {
			require.NoError(t, os.Setenv("GITHUB_TOKEN", originalGitHub))
		} else {
			require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
		}
	}()

	// Set empty GH_PAT_TOKEN and valid GITHUB_TOKEN
	require.NoError(t, os.Setenv("GH_PAT_TOKEN", ""))
	require.NoError(t, os.Setenv("GITHUB_TOKEN", "github_token_789"))

	token := GetGitHubToken()
	assert.Equal(t, "github_token_789", token)
}

// TestGetGitHubTokenBothEmpty tests that empty tokens return empty string
func TestGetGitHubTokenBothEmpty(t *testing.T) {
	// Save original environment
	originalPAT := os.Getenv("GH_PAT_TOKEN")
	originalGitHub := os.Getenv("GITHUB_TOKEN")

	// Restore environment after test
	defer func() {
		if originalPAT != "" {
			require.NoError(t, os.Setenv("GH_PAT_TOKEN", originalPAT))
		} else {
			require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
		}
		if originalGitHub != "" {
			require.NoError(t, os.Setenv("GITHUB_TOKEN", originalGitHub))
		} else {
			require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
		}
	}()

	// Clear both tokens
	require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))

	token := GetGitHubToken()
	assert.Empty(t, token)
}

// TestGetGitHubTokenBothEmptyStrings tests that both empty string tokens return empty string
func TestGetGitHubTokenBothEmptyStrings(t *testing.T) {
	// Save original environment
	originalPAT := os.Getenv("GH_PAT_TOKEN")
	originalGitHub := os.Getenv("GITHUB_TOKEN")

	// Restore environment after test
	defer func() {
		if originalPAT != "" {
			require.NoError(t, os.Setenv("GH_PAT_TOKEN", originalPAT))
		} else {
			require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
		}
		if originalGitHub != "" {
			require.NoError(t, os.Setenv("GITHUB_TOKEN", originalGitHub))
		} else {
			require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
		}
	}()

	// Set both to empty strings
	require.NoError(t, os.Setenv("GH_PAT_TOKEN", ""))
	require.NoError(t, os.Setenv("GITHUB_TOKEN", ""))

	token := GetGitHubToken()
	assert.Empty(t, token)
}

// TestSkipIfNoGitHubAuthWithToken tests that SkipIfNoGitHubAuth does not skip when token is available
func TestSkipIfNoGitHubAuthWithToken(t *testing.T) {
	// Save original environment
	originalPAT := os.Getenv("GH_PAT_TOKEN")
	originalGitHub := os.Getenv("GITHUB_TOKEN")

	// Restore environment after test
	defer func() {
		if originalPAT != "" {
			require.NoError(t, os.Setenv("GH_PAT_TOKEN", originalPAT))
		} else {
			require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
		}
		if originalGitHub != "" {
			require.NoError(t, os.Setenv("GITHUB_TOKEN", originalGitHub))
		} else {
			require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
		}
	}()

	// Set a token
	require.NoError(t, os.Setenv("GH_PAT_TOKEN", "test_token"))

	// We can't easily test the actual Skip behavior without complex mocking,
	// but we can test the token retrieval logic that drives the decision
	token := GetGitHubToken()
	shouldSkip := token == ""

	assert.False(t, shouldSkip, "Test should not be skipped when token is available")
	assert.Equal(t, "test_token", token)
}

// TestSkipIfNoGitHubAuthWithoutToken tests the logic for when no token is available
func TestSkipIfNoGitHubAuthWithoutToken(t *testing.T) {
	// Save original environment
	originalPAT := os.Getenv("GH_PAT_TOKEN")
	originalGitHub := os.Getenv("GITHUB_TOKEN")

	// Restore environment after test
	defer func() {
		if originalPAT != "" {
			require.NoError(t, os.Setenv("GH_PAT_TOKEN", originalPAT))
		} else {
			require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
		}
		if originalGitHub != "" {
			require.NoError(t, os.Setenv("GITHUB_TOKEN", originalGitHub))
		} else {
			require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
		}
	}()

	// Clear both tokens
	require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))

	// Test the condition that would cause a skip
	token := GetGitHubToken()
	shouldSkip := token == ""

	assert.True(t, shouldSkip, "Test should be skipped when no token is available")
	assert.Empty(t, token)
}

// TestGetGitHubTokenWithWhitespace tests that tokens with whitespace are preserved
func TestGetGitHubTokenWithWhitespace(t *testing.T) {
	// Save original environment
	originalPAT := os.Getenv("GH_PAT_TOKEN")
	originalGitHub := os.Getenv("GITHUB_TOKEN")

	// Restore environment after test
	defer func() {
		if originalPAT != "" {
			require.NoError(t, os.Setenv("GH_PAT_TOKEN", originalPAT))
		} else {
			require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
		}
		if originalGitHub != "" {
			require.NoError(t, os.Setenv("GITHUB_TOKEN", originalGitHub))
		} else {
			require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
		}
	}()

	// Test GH_PAT_TOKEN with leading/trailing whitespace
	require.NoError(t, os.Setenv("GH_PAT_TOKEN", "  token_with_spaces  "))
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))

	token := GetGitHubToken()
	assert.Equal(t, "  token_with_spaces  ", token)
}

// TestGetGitHubTokenRealWorldTokenFormats tests realistic token formats
func TestGetGitHubTokenRealWorldTokenFormats(t *testing.T) {
	tests := []struct {
		name     string
		patToken string
		ghToken  string
		expected string
	}{
		{
			name:     "Classic PAT token format",
			patToken: "ghp_1234567890abcdef1234567890abcdef12345678",
			ghToken:  "",
			expected: "ghp_1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "Fine-grained PAT token format",
			patToken: "github_pat_11ABCDEFG0123456789_abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			ghToken:  "",
			expected: "github_pat_11ABCDEFG0123456789_abcdefghijklmnopqrstuvwxyz1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		},
		{
			name:     "GitHub token fallback",
			patToken: "",
			ghToken:  "ghs_1234567890abcdef1234567890abcdef",
			expected: "ghs_1234567890abcdef1234567890abcdef",
		},
		{
			name:     "Old format GitHub token",
			patToken: "",
			ghToken:  "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b",
			expected: "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalPAT := os.Getenv("GH_PAT_TOKEN")
			originalGitHub := os.Getenv("GITHUB_TOKEN")

			// Restore environment after test
			defer func() {
				if originalPAT != "" {
					require.NoError(t, os.Setenv("GH_PAT_TOKEN", originalPAT))
				} else {
					require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
				}
				if originalGitHub != "" {
					require.NoError(t, os.Setenv("GITHUB_TOKEN", originalGitHub))
				} else {
					require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
				}
			}()

			// Set test tokens
			if tt.patToken != "" {
				require.NoError(t, os.Setenv("GH_PAT_TOKEN", tt.patToken))
			} else {
				require.NoError(t, os.Unsetenv("GH_PAT_TOKEN"))
			}

			if tt.ghToken != "" {
				require.NoError(t, os.Setenv("GITHUB_TOKEN", tt.ghToken))
			} else {
				require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
			}

			token := GetGitHubToken()
			assert.Equal(t, tt.expected, token)
		})
	}
}
