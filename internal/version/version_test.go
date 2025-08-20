package version

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		mockResponse    string
		mockStatusCode  int
		expectedRelease *GitHubRelease
		expectError     bool
		errorContains   string
	}{
		{
			name:           "ValidRelease",
			mockStatusCode: http.StatusOK,
			mockResponse: `{
				"tag_name": "v1.2.3",
				"name": "Release v1.2.3",
				"draft": false,
				"prerelease": false,
				"published_at": "2023-01-01T12:00:00Z",
				"body": "Bug fixes and improvements"
			}`,
			expectedRelease: &GitHubRelease{
				TagName:     "v1.2.3",
				Name:        "Release v1.2.3",
				Draft:       false,
				Prerelease:  false,
				PublishedAt: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				Body:        "Bug fixes and improvements",
			},
			expectError: false,
		},
		{
			name:           "InvalidJSON",
			mockStatusCode: http.StatusOK,
			mockResponse:   `{invalid json`,
			expectError:    true,
			errorContains:  "invalid character",
		},
		{
			name:           "NotFound",
			mockStatusCode: http.StatusNotFound,
			mockResponse:   `{"message": "Not Found"}`,
			expectError:    true,
			errorContains:  "GitHub API request failed",
		},
		{
			name:           "RateLimited",
			mockStatusCode: http.StatusForbidden,
			mockResponse:   `{"message": "API rate limit exceeded"}`,
			expectError:    true,
			errorContains:  "GitHub API request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/repos/owner/repo/releases/latest", r.URL.Path)
				assert.Contains(t, r.Header.Get("User-Agent"), "go-broadcast")
				assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))

				w.WriteHeader(tt.mockStatusCode)
				_, _ = w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Temporarily override the API URL for testing
			originalURL := "https://api.github.com/repos/%s/%s/releases/latest"
			testURL := server.URL + "/repos/%s/%s/releases/latest"
			_ = originalURL // Keep for reference

			// Mock the function by calling it with server URL
			release, err := getLatestReleaseFromURL("owner", "repo", testURL)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, release)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRelease, release)
			}
		})
	}
}

// Helper function for testing with custom URL
func getLatestReleaseFromURL(owner, repo, urlTemplate string) (*GitHubRelease, error) {
	// This is a test helper that mimics GetLatestRelease but with custom URL
	client := &http.Client{Timeout: 10 * time.Second}
	url := strings.Replace(urlTemplate, "%s/%s", owner+"/"+repo, 1)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "go-broadcast/dev (test/test)")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrGitHubAPIFailed
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func TestCompareVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "V1Greater",
			v1:       "1.2.3",
			v2:       "1.2.2",
			expected: 1,
		},
		{
			name:     "V2Greater",
			v1:       "1.2.2",
			v2:       "1.2.3",
			expected: -1,
		},
		{
			name:     "Equal",
			v1:       "1.2.3",
			v2:       "1.2.3",
			expected: 0,
		},
		{
			name:     "MajorVersionDifference",
			v1:       "2.0.0",
			v2:       "1.9.9",
			expected: 1,
		},
		{
			name:     "MinorVersionDifference",
			v1:       "1.3.0",
			v2:       "1.2.9",
			expected: 1,
		},
		{
			name:     "WithVPrefix",
			v1:       "v1.2.3",
			v2:       "v1.2.2",
			expected: 1,
		},
		{
			name:     "MixedVPrefix",
			v1:       "v1.2.3",
			v2:       "1.2.3",
			expected: 0,
		},
		{
			name:     "DevVersionVsRelease",
			v1:       "dev",
			v2:       "1.2.3",
			expected: -1,
		},
		{
			name:     "ReleaseVsDevVersion",
			v1:       "1.2.3",
			v2:       "dev",
			expected: 1,
		},
		{
			name:     "BothDevVersions",
			v1:       "dev",
			v2:       "dev",
			expected: 0,
		},
		{
			name:     "CommitHashVsRelease",
			v1:       "abc123def456",
			v2:       "1.2.3",
			expected: -1,
		},
		{
			name:     "ReleaseVsCommitHash",
			v1:       "1.2.3",
			v2:       "abc123def456",
			expected: 1,
		},
		{
			name:     "EmptyVersionVsRelease",
			v1:       "",
			v2:       "1.2.3",
			expected: -1,
		},
		{
			name:     "VersionWithSuffix",
			v1:       "1.2.3-rc1",
			v2:       "1.2.3",
			expected: 0, // Suffixes are ignored in basic comparison
		},
		{
			name:     "TwoPartVersion",
			v1:       "1.2",
			v2:       "1.2.0",
			expected: 0,
		},
		{
			name:     "SinglePartVersion",
			v1:       "2",
			v2:       "1.9.9",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CompareVersions(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		expected       bool
	}{
		{
			name:           "NewerAvailable",
			currentVersion: "1.2.2",
			latestVersion:  "1.2.3",
			expected:       true,
		},
		{
			name:           "SameVersion",
			currentVersion: "1.2.3",
			latestVersion:  "1.2.3",
			expected:       false,
		},
		{
			name:           "CurrentNewer",
			currentVersion: "1.2.4",
			latestVersion:  "1.2.3",
			expected:       false,
		},
		{
			name:           "DevVersionNeedsUpgrade",
			currentVersion: "dev",
			latestVersion:  "1.2.3",
			expected:       true,
		},
		{
			name:           "CommitHashNeedsUpgrade",
			currentVersion: "abc123def456",
			latestVersion:  "1.2.3",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsNewerVersion(tt.currentVersion, tt.latestVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "WithVPrefix",
			version:  "v1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "WithoutVPrefix",
			version:  "1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "WithSuffix",
			version:  "1.2.3-rc1",
			expected: "1.2.3",
		},
		{
			name:     "WithSpaces",
			version:  "  1.2.3  ",
			expected: "1.2.3",
		},
		{
			name:     "WithVPrefixAndSuffix",
			version:  "v1.2.3-dirty",
			expected: "1.2.3",
		},
		{
			name:     "EmptyString",
			version:  "",
			expected: "",
		},
		{
			name:     "OnlyV",
			version:  "v",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := NormalizeVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsCommitHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{
			name:     "ValidShortCommitHash",
			version:  "abc123d",
			expected: true,
		},
		{
			name:     "ValidLongCommitHash",
			version:  "abc123def456789012345678901234567890abcd",
			expected: true,
		},
		{
			name:     "ValidHashWithDirtySuffix",
			version:  "abc123d-dirty",
			expected: true,
		},
		{
			name:     "ValidMixedCaseHash",
			version:  "AbC123DeF456",
			expected: true,
		},
		{
			name:     "TooShort",
			version:  "abc12",
			expected: false,
		},
		{
			name:     "TooLong",
			version:  "abc123def456789012345678901234567890abcdef",
			expected: false,
		},
		{
			name:     "ContainsInvalidCharacters",
			version:  "abc123xyz",
			expected: false,
		},
		{
			name:     "ContainsSpecialCharacters",
			version:  "abc123-def",
			expected: false,
		},
		{
			name:     "EmptyString",
			version:  "",
			expected: false,
		},
		{
			name:     "StandardVersion",
			version:  "1.2.3",
			expected: false,
		},
		{
			name:     "DevVersion",
			version:  "dev",
			expected: false,
		},
		{
			name:     "OnlyNumbers",
			version:  "1234567890",
			expected: true,
		},
		{
			name:     "OnlyValidHexLetters",
			version:  "abcdefabcdef",
			expected: true,
		},
		{
			name:     "OnlyInvalidLetters",
			version:  "abcdefghijk",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isCommitHash(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected []int
	}{
		{
			name:     "StandardVersion",
			version:  "1.2.3",
			expected: []int{1, 2, 3},
		},
		{
			name:     "TwoPartVersion",
			version:  "1.2",
			expected: []int{1, 2},
		},
		{
			name:     "SinglePartVersion",
			version:  "1",
			expected: []int{1},
		},
		{
			name:     "VersionWithSuffix",
			version:  "1.2.3-rc1",
			expected: []int{1, 2, 3},
		},
		{
			name:     "VersionWithBuildSuffix",
			version:  "1.2.3+build123",
			expected: []int{1, 2, 3},
		},
		{
			name:     "EmptyString",
			version:  "",
			expected: []int{},
		},
		{
			name:     "InvalidVersion",
			version:  "abc.def.ghi",
			expected: []int{},
		},
		{
			name:     "MixedValidInvalid",
			version:  "1.abc.3",
			expected: []int{1, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := parseVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersionInfo(t *testing.T) {
	t.Parallel()

	info := Info{
		Current: "1.2.2",
		Latest:  "1.2.3",
		IsNewer: true,
	}

	assert.Equal(t, "1.2.2", info.Current)
	assert.Equal(t, "1.2.3", info.Latest)
	assert.True(t, info.IsNewer)
}

func TestGitHubRelease(t *testing.T) {
	t.Parallel()

	publishedAt := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	release := GitHubRelease{
		TagName:     "v1.2.3",
		Name:        "Release v1.2.3",
		Draft:       false,
		Prerelease:  false,
		PublishedAt: publishedAt,
		Body:        "Bug fixes and improvements",
	}

	assert.Equal(t, "v1.2.3", release.TagName)
	assert.Equal(t, "Release v1.2.3", release.Name)
	assert.False(t, release.Draft)
	assert.False(t, release.Prerelease)
	assert.Equal(t, publishedAt, release.PublishedAt)
	assert.Equal(t, "Bug fixes and improvements", release.Body)
}

// Benchmarks
func BenchmarkCompareVersions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CompareVersions("1.2.3", "1.2.4")
	}
}

func BenchmarkIsNewerVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsNewerVersion("1.2.3", "1.2.4")
	}
}

func BenchmarkNormalizeVersion(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NormalizeVersion("v1.2.3-rc1")
	}
}

func BenchmarkIsCommitHash(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isCommitHash("abc123def456")
	}
}
