package analytics

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/gh"
)

// TestBuildBatchQuery_IncludesAllNewFields verifies all enhanced fields are in the GraphQL query
func TestBuildBatchQuery_IncludesAllNewFields(t *testing.T) {
	repos := []gh.RepoInfo{
		{
			Owner: struct {
				Login string `json:"login"`
			}{Login: "mrz1836"},
			Name: "go-broadcast",
		},
	}

	query := BuildBatchQuery(repos)

	// Basic fields (already tested in existing tests)
	assert.Contains(t, query, "nameWithOwner")
	assert.Contains(t, query, "stargazerCount")

	// NEW: Verify all enhanced fields
	assert.Contains(t, query, "primaryLanguage", "Missing language field")
	assert.Contains(t, query, "createdAt", "Missing creation timestamp")
	assert.Contains(t, query, "homepageUrl", "Missing homepage URL")
	assert.Contains(t, query, "diskUsage", "Missing disk usage")
	assert.Contains(t, query, "licenseInfo", "Missing license info")
	assert.Contains(t, query, "repositoryTopics", "Missing topics")
	assert.Contains(t, query, "hasIssuesEnabled", "Missing issues flag")
	assert.Contains(t, query, "hasWikiEnabled", "Missing wiki flag")
	assert.Contains(t, query, "hasDiscussionsEnabled", "Missing discussions flag")
	assert.Contains(t, query, "sshUrl", "Missing SSH URL")
	assert.Contains(t, query, "url", "Missing HTML URL")

	// Verify nested structures (allow for formatting differences)
	assert.Contains(t, query, "primaryLanguage", "Language field missing")
	assert.Contains(t, query, "licenseInfo", "License field missing")
	assert.Contains(t, query, "repositoryTopics(first: 10)", "Topics pagination missing")
}

// TestParseBatchResponse_AllNewFields verifies parsing of all enhanced metadata fields
func TestParseBatchResponse_AllNewFields(t *testing.T) {
	testCases := []struct {
		name     string
		data     map[string]interface{}
		expected func(*testing.T, *RepoMetadata)
	}{
		{
			name: "Complete metadata with all fields",
			data: map[string]interface{}{
				"repo0": map[string]interface{}{
					"nameWithOwner": "mrz1836/go-broadcast",
					"primaryLanguage": map[string]interface{}{
						"name": "Go",
					},
					"createdAt":   "2023-01-15T10:00:00Z",
					"homepageUrl": "https://example.com",
					"diskUsage":   1024.0,
					"licenseInfo": map[string]interface{}{
						"key":  "mit",
						"name": "MIT License",
					},
					"repositoryTopics": map[string]interface{}{
						"nodes": []interface{}{
							map[string]interface{}{
								"topic": map[string]interface{}{"name": "golang"},
							},
							map[string]interface{}{
								"topic": map[string]interface{}{"name": "cli"},
							},
						},
					},
					"hasIssuesEnabled":      true,
					"hasWikiEnabled":        false,
					"hasDiscussionsEnabled": true,
					"url":                   "https://github.com/mrz1836/go-broadcast",
					"sshUrl":                "git@github.com:mrz1836/go-broadcast.git",
					"isPrivate":             false,
					"isArchived":            false,
					"isFork":                true,
					"parent": map[string]interface{}{
						"nameWithOwner": "upstream/original",
					},
				},
			},
			expected: func(t *testing.T, m *RepoMetadata) {
				assert.Equal(t, "Go", m.Language)
				assert.Equal(t, "2023-01-15T10:00:00Z", m.CreatedAt)
				assert.Equal(t, "https://example.com", m.HomepageURL)
				assert.Equal(t, "mit", m.License)
				assert.Equal(t, "MIT License", m.LicenseName)
				assert.Equal(t, 1024, m.DiskUsageKB)
				assert.Equal(t, []string{"golang", "cli"}, m.Topics)
				assert.True(t, m.HasIssuesEnabled)
				assert.False(t, m.HasWikiEnabled)
				assert.True(t, m.HasDiscussionsEnabled)
				assert.Equal(t, "https://github.com/mrz1836/go-broadcast", m.HTMLURL)
				assert.Equal(t, "git@github.com:mrz1836/go-broadcast.git", m.SSHURL)
				assert.Equal(t, "https://github.com/mrz1836/go-broadcast.git", m.CloneURL)
				assert.True(t, m.IsFork)
				assert.Equal(t, "upstream/original", m.ForkParent)
			},
		},
		{
			name: "Minimal metadata (null optional fields)",
			data: map[string]interface{}{
				"repo0": map[string]interface{}{
					"nameWithOwner": "test/minimal",
					// No primaryLanguage
					// No licenseInfo
					// No topics
					"hasIssuesEnabled": false,
					"url":              "https://github.com/test/minimal",
					"sshUrl":           "git@github.com:test/minimal.git",
				},
			},
			expected: func(t *testing.T, m *RepoMetadata) {
				assert.Empty(t, m.Language, "Empty language should be empty string")
				assert.Empty(t, m.License, "Null license should be empty string")
				assert.Empty(t, m.Topics, "Null topics should be empty array")
				assert.False(t, m.HasIssuesEnabled)
				assert.Equal(t, "https://github.com/test/minimal", m.HTMLURL)
				assert.Equal(t, "git@github.com:test/minimal.git", m.SSHURL)
				assert.Equal(t, "https://github.com/test/minimal.git", m.CloneURL)
			},
		},
		{
			name: "Topics with 10+ items (pagination test)",
			data: map[string]interface{}{
				"repo0": map[string]interface{}{
					"nameWithOwner": "test/many-topics",
					"repositoryTopics": map[string]interface{}{
						"nodes": buildTopicNodes(12), // 12 topics
					},
					"url":    "https://github.com/test/many-topics",
					"sshUrl": "git@github.com:test/many-topics.git",
				},
			},
			expected: func(t *testing.T, m *RepoMetadata) {
				assert.Len(t, m.Topics, 12, "Should handle 12 topics")
			},
		},
		{
			name: "Empty homepage URL",
			data: map[string]interface{}{
				"repo0": map[string]interface{}{
					"nameWithOwner": "test/no-homepage",
					"homepageUrl":   "",
					"url":           "https://github.com/test/no-homepage",
					"sshUrl":        "git@github.com:test/no-homepage.git",
				},
			},
			expected: func(t *testing.T, m *RepoMetadata) {
				assert.Empty(t, m.HomepageURL, "Empty homepage should be empty string")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repos := []gh.RepoInfo{{FullName: "mrz1836/go-broadcast"}}
			result, err := ParseBatchResponse(tc.data, repos)

			require.NoError(t, err)
			require.Contains(t, result, "mrz1836/go-broadcast")

			metadata := result["mrz1836/go-broadcast"]
			tc.expected(t, metadata)
		})
	}
}

// TestParseBatchResponse_LicenseVariations tests different license scenarios
func TestParseBatchResponse_LicenseVariations(t *testing.T) {
	testCases := []struct {
		name         string
		licenseInfo  interface{}
		expectedKey  string
		expectedName string
	}{
		{
			name: "MIT license",
			licenseInfo: map[string]interface{}{
				"key":  "mit",
				"name": "MIT License",
			},
			expectedKey:  "mit",
			expectedName: "MIT License",
		},
		{
			name: "Apache 2.0 license",
			licenseInfo: map[string]interface{}{
				"key":  "apache-2.0",
				"name": "Apache License 2.0",
			},
			expectedKey:  "apache-2.0",
			expectedName: "Apache License 2.0",
		},
		{
			name:         "No license",
			licenseInfo:  nil,
			expectedKey:  "",
			expectedName: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]interface{}{
				"repo0": map[string]interface{}{
					"nameWithOwner": "test/repo",
					"url":           "https://github.com/test/repo",
					"sshUrl":        "git@github.com:test/repo.git",
				},
			}

			if tc.licenseInfo != nil {
				data["repo0"].(map[string]interface{})["licenseInfo"] = tc.licenseInfo
			}

			repos := []gh.RepoInfo{{FullName: "test/repo"}}
			result, err := ParseBatchResponse(data, repos)

			require.NoError(t, err)
			require.Contains(t, result, "test/repo")

			metadata := result["test/repo"]
			assert.Equal(t, tc.expectedKey, metadata.License)
			assert.Equal(t, tc.expectedName, metadata.LicenseName)
		})
	}
}

// TestParseBatchResponse_TopicsJSON verifies topics are correctly extracted as array
func TestParseBatchResponse_TopicsJSON(t *testing.T) {
	data := map[string]interface{}{
		"repo0": map[string]interface{}{
			"nameWithOwner": "test/repo",
			"repositoryTopics": map[string]interface{}{
				"nodes": []interface{}{
					map[string]interface{}{
						"topic": map[string]interface{}{"name": "golang"},
					},
					map[string]interface{}{
						"topic": map[string]interface{}{"name": "testing"},
					},
					map[string]interface{}{
						"topic": map[string]interface{}{"name": "open-source"},
					},
				},
			},
			"url":    "https://github.com/test/repo",
			"sshUrl": "git@github.com:test/repo.git",
		},
	}

	repos := []gh.RepoInfo{{FullName: "test/repo"}}
	result, err := ParseBatchResponse(data, repos)

	require.NoError(t, err)
	require.Contains(t, result, "test/repo")

	metadata := result["test/repo"]
	assert.Equal(t, []string{"golang", "testing", "open-source"}, metadata.Topics)

	// Verify topics can be marshaled to JSON
	jsonBytes, err := json.Marshal(metadata.Topics)
	require.NoError(t, err)
	assert.JSONEq(t, `["golang","testing","open-source"]`, string(jsonBytes))
}

// buildTopicNodes is a helper function to create topic nodes for testing
func buildTopicNodes(count int) []interface{} {
	nodes := make([]interface{}, count)
	for i := 0; i < count; i++ {
		nodes[i] = map[string]interface{}{
			"topic": map[string]interface{}{
				"name": fmt.Sprintf("topic%d", i),
			},
		}
	}
	return nodes
}
