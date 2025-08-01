package github

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

func TestNewPRCommentManager(t *testing.T) {
	client := New("test-token")

	tests := []struct {
		name     string
		config   *PRCommentConfig
		expected *PRCommentConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
			expected: &PRCommentConfig{
				MinUpdateIntervalMinutes: 5,
				MaxCommentsPerPR:         1,
				CommentSignature:         "gofortress-coverage-v1",
				IncludeTrend:             true,
				IncludeCoverageDetails:   true,
				IncludeFileAnalysis:      false,
				ShowCoverageHistory:      true,
				GeneratePRBadges:         true,
				BadgeStyle:               "flat",
				EnableStatusChecks:       true,
				FailBelowThreshold:       true,
				BlockMergeOnFailure:      false,
			},
		},
		{
			name: "custom config",
			config: &PRCommentConfig{
				MinUpdateIntervalMinutes: 10,
				MaxCommentsPerPR:         2,
				CommentSignature:         "custom-signature",
				IncludeTrend:             false,
				GeneratePRBadges:         false,
			},
			expected: &PRCommentConfig{
				MinUpdateIntervalMinutes: 10,
				MaxCommentsPerPR:         2,
				CommentSignature:         "custom-signature",
				IncludeTrend:             false,
				GeneratePRBadges:         false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewPRCommentManager(client, tt.config)
			require.NotNil(t, manager)
			require.Equal(t, client, manager.client)

			if tt.config == nil {
				require.Equal(t, tt.expected.MinUpdateIntervalMinutes, manager.config.MinUpdateIntervalMinutes)
				require.Equal(t, tt.expected.MaxCommentsPerPR, manager.config.MaxCommentsPerPR)
				require.Equal(t, tt.expected.CommentSignature, manager.config.CommentSignature)
				require.Equal(t, tt.expected.IncludeTrend, manager.config.IncludeTrend)
				require.Equal(t, tt.expected.GeneratePRBadges, manager.config.GeneratePRBadges)
			} else {
				require.Equal(t, tt.expected.MinUpdateIntervalMinutes, manager.config.MinUpdateIntervalMinutes)
				require.Equal(t, tt.expected.MaxCommentsPerPR, manager.config.MaxCommentsPerPR)
				require.Equal(t, tt.expected.CommentSignature, manager.config.CommentSignature)
				require.Equal(t, tt.expected.IncludeTrend, manager.config.IncludeTrend)
				require.Equal(t, tt.expected.GeneratePRBadges, manager.config.GeneratePRBadges)
			}

			require.NotNil(t, manager.logger)
		})
	}
}

func TestCreateOrUpdatePRComment(t *testing.T) {
	tests := []struct {
		name           string
		setupMockFn    func() *httptest.Server
		owner          string
		repo           string
		prNumber       int
		commentBody    string
		comparison     *CoverageComparison
		expectedError  string
		expectedAction string
	}{
		{
			name: "successful comment creation",
			setupMockFn: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/repos/testowner/testrepo/pulls/123":
						// Mock PR response
						pr := map[string]interface{}{
							"number": 123,
							"title":  "Test PR",
							"state":  "open",
							"head": map[string]interface{}{
								"sha": "abc123",
							},
						}
						w.Header().Set("Content-Type", "application/json")
						assert.NoError(t, json.NewEncoder(w).Encode(pr))
					case "/repos/testowner/testrepo/issues/123/comments":
						switch r.Method {
						case "GET":
							// Mock existing comments (empty)
							w.Header().Set("Content-Type", "application/json")
							assert.NoError(t, json.NewEncoder(w).Encode([]interface{}{}))
						case "POST":
							// Mock comment creation
							comment := map[string]interface{}{
								"id":   456,
								"body": "test comment",
							}
							w.Header().Set("Content-Type", "application/json")
							assert.NoError(t, json.NewEncoder(w).Encode(comment))
						}
					default:
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			owner:       "testowner",
			repo:        "testrepo",
			prNumber:    123,
			commentBody: "test comment",
			comparison: &CoverageComparison{
				PRCoverage: CoverageData{Percentage: 85.0},
			},
			expectedAction: "created",
		},
		{
			name: "PR not found error",
			setupMockFn: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/repos/testowner/testrepo/pulls/123" {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			owner:         "testowner",
			repo:          "testrepo",
			prNumber:      123,
			commentBody:   "test comment",
			comparison:    &CoverageComparison{},
			expectedError: "failed to get PR information",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupMockFn()
			defer server.Close()

			client := NewWithConfig(&Config{
				Token:      "test-token",
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				RetryCount: 1,
				UserAgent:  "test-agent",
			})

			manager := NewPRCommentManager(client, nil)
			ctx := context.Background()

			result, err := manager.CreateOrUpdatePRComment(ctx, tt.owner, tt.repo, tt.prNumber, tt.commentBody, tt.comparison)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.expectedAction != "" {
					require.Equal(t, tt.expectedAction, result.Action)
				}
			}
		})
	}
}

func TestFindExistingCoverageComments(t *testing.T) {
	tests := []struct {
		name          string
		setupMockFn   func() *httptest.Server
		expectedCount int
		expectedError string
	}{
		{
			name: "no existing comments",
			setupMockFn: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/repos/testowner/testrepo/issues/123/comments" && r.Method == "GET" {
						w.Header().Set("Content-Type", "application/json")
						assert.NoError(t, json.NewEncoder(w).Encode([]interface{}{}))
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			expectedCount: 0,
		},
		{
			name: "existing coverage comments",
			setupMockFn: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/repos/testowner/testrepo/issues/123/comments" && r.Method == "GET" {
						comments := []map[string]interface{}{
							{
								"id":   1,
								"body": "<!-- gofortress-coverage-v1 --> Some coverage comment",
							},
							{
								"id":   2,
								"body": "Regular comment without signature",
							},
							{
								"id":   3,
								"body": "<!-- gofortress-coverage-v1 --> Another coverage comment",
							},
						}
						w.Header().Set("Content-Type", "application/json")
						assert.NoError(t, json.NewEncoder(w).Encode(comments))
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			expectedCount: 2,
		},
		{
			name: "API error",
			setupMockFn: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			expectedError: "GitHub API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupMockFn()
			defer server.Close()

			client := NewWithConfig(&Config{
				Token:      "test-token",
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				RetryCount: 1,
				UserAgent:  "test-agent",
			})

			manager := NewPRCommentManager(client, nil)
			ctx := context.Background()

			comments, err := manager.findExistingCoverageComments(ctx, "testowner", "testrepo", 123)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				require.Len(t, comments, tt.expectedCount)
			}
		})
	}
}

func TestIsCoverageComment(t *testing.T) {
	manager := NewPRCommentManager(New("test-token"), nil)

	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "valid coverage comment",
			body:     "<!-- gofortress-coverage-v1 --> Coverage report",
			expected: true,
		},
		{
			name:     "coverage comment with different content",
			body:     "Some text\n<!-- gofortress-coverage-v1 -->\nMore content",
			expected: true,
		},
		{
			name:     "regular comment",
			body:     "This is just a regular PR comment",
			expected: false,
		},
		{
			name:     "different signature",
			body:     "<!-- different-signature --> Not our comment",
			expected: false,
		},
		{
			name:     "empty comment",
			body:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.isCoverageComment(tt.body)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDetermineCommentAction(t *testing.T) {
	manager := NewPRCommentManager(New("test-token"), &PRCommentConfig{
		MinUpdateIntervalMinutes: 5,
		MaxCommentsPerPR:         1,
	})

	now := time.Now()
	recentTime := now.Add(-2 * time.Minute) // 2 minutes ago
	oldTime := now.Add(-10 * time.Minute)   // 10 minutes ago

	tests := []struct {
		name              string
		existingComments  []Comment
		coverageData      *CoverageComparison
		expectedAction    string
		expectedCommentID int
	}{
		{
			name:             "no existing comments - create new",
			existingComments: []Comment{},
			coverageData:     &CoverageComparison{PRCoverage: CoverageData{Percentage: 85.0}},
			expectedAction:   "create",
		},
		{
			name: "recent comment - skip update",
			existingComments: []Comment{
				{
					ID:        123,
					Body:      "<!-- gofortress-coverage-v1 --> Previous comment",
					CreatedAt: recentTime.Format(time.RFC3339),
					UpdatedAt: recentTime.Format(time.RFC3339),
				},
			},
			coverageData:   &CoverageComparison{PRCoverage: CoverageData{Percentage: 85.0}},
			expectedAction: "skipped",
		},
		{
			name: "old comment - update",
			existingComments: []Comment{
				{
					ID:        123,
					Body:      "<!-- gofortress-coverage-v1 --> Previous comment",
					CreatedAt: oldTime.Format(time.RFC3339),
					UpdatedAt: oldTime.Format(time.RFC3339),
				},
			},
			coverageData:      &CoverageComparison{PRCoverage: CoverageData{Percentage: 85.0}},
			expectedAction:    "update",
			expectedCommentID: 123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, shouldUpdate, reason := manager.determineCommentAction(tt.existingComments, tt.coverageData)
			require.Equal(t, tt.expectedAction, action)
			switch tt.expectedAction {
			case "create", "update":
				require.True(t, shouldUpdate)
			default:
				require.False(t, shouldUpdate)
			}
			require.NotEmpty(t, reason)
		})
	}
}

func TestHasSignificantCoverageChange(t *testing.T) {
	manager := NewPRCommentManager(New("test-token"), nil)

	tests := []struct {
		name        string
		oldCoverage float64
		newCoverage float64
		expected    bool
	}{
		{
			name:        "significant increase",
			oldCoverage: 80.0,
			newCoverage: 85.5,
			expected:    true,
		},
		{
			name:        "significant decrease",
			oldCoverage: 85.0,
			newCoverage: 80.0,
			expected:    true,
		},
		{
			name:        "minimal change",
			oldCoverage: 85.0,
			newCoverage: 85.2,
			expected:    false,
		},
		{
			name:        "exactly 1% change",
			oldCoverage: 80.0,
			newCoverage: 81.0,
			expected:    false,
		},
		{
			name:        "no change",
			oldCoverage: 85.0,
			newCoverage: 85.0,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comparison := &CoverageComparison{
				BaseCoverage: CoverageData{Percentage: tt.oldCoverage},
				PRCoverage:   CoverageData{Percentage: tt.newCoverage},
				Difference:   tt.newCoverage - tt.oldCoverage,
			}
			result := manager.hasSignificantCoverageChange(comparison)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateCoverageStatusCheck(t *testing.T) {
	tests := []struct {
		name        string
		setupMockFn func() *httptest.Server
		comparison  *CoverageComparison
		expectError bool
	}{
		{
			name: "successful status check creation",
			setupMockFn: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.Path, "/statuses/") && r.Method == "POST" {
						status := map[string]interface{}{
							"id":          123,
							"state":       "success",
							"description": "Coverage check passed",
						}
						w.Header().Set("Content-Type", "application/json")
						assert.NoError(t, json.NewEncoder(w).Encode(status))
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			comparison: &CoverageComparison{
				PRCoverage: CoverageData{Percentage: 85.0},
			},
			expectError: false,
		},
		{
			name: "API error",
			setupMockFn: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			comparison: &CoverageComparison{
				PRCoverage: CoverageData{Percentage: 85.0},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupMockFn()
			defer server.Close()

			client := NewWithConfig(&Config{
				Token:      "test-token",
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				RetryCount: 1,
				UserAgent:  "test-agent",
			})

			manager := NewPRCommentManager(client, &PRCommentConfig{
				EnableStatusChecks: true,
			})

			ctx := context.Background()
			err := manager.createCoverageStatusCheck(ctx, "testowner", "testrepo", "abc123", tt.comparison)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExtractCommentMetadata(t *testing.T) {
	manager := NewPRCommentManager(New("test-token"), nil)

	tests := []struct {
		name     string
		body     string
		expected CommentMetadata
	}{
		{
			name: "comment with JSON metadata",
			body: `<!-- gofortress-coverage-v1 -->
Coverage Report
<!-- metadata: {"version": "1.0", "created_at": "2023-12-25T12:00:00Z"} -->`,
			expected: CommentMetadata{
				CommentVersion: "1.0",
				CreatedAt:      time.Date(2023, 12, 25, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "comment without metadata",
			body: "Regular comment without metadata",
			expected: CommentMetadata{
				CommentVersion: "unknown",
			},
		},
		{
			name: "comment with invalid JSON",
			body: `<!-- gofortress-coverage-v1 -->
Coverage Report
<!-- invalid json -->`,
			expected: CommentMetadata{
				CommentVersion: "unknown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.extractCommentMetadata(tt.body)
			if tt.expected.CommentVersion == "unknown" {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, tt.expected.CommentVersion, result.CommentVersion)
				if !tt.expected.CreatedAt.IsZero() {
					require.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
				}
			}
		})
	}
}

func TestDeletePRComments(t *testing.T) {
	tests := []struct {
		name        string
		setupMockFn func() *httptest.Server
		expectError bool
	}{
		{
			name: "successful deletion",
			setupMockFn: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/repos/testowner/testrepo/issues/123/comments" && r.Method == "GET" {
						comments := []map[string]interface{}{
							{
								"id":   1,
								"body": "<!-- gofortress-coverage-v1 --> Coverage comment 1",
							},
							{
								"id":   2,
								"body": "Regular comment",
							},
						}
						w.Header().Set("Content-Type", "application/json")
						assert.NoError(t, json.NewEncoder(w).Encode(comments))
					} else if strings.Contains(r.URL.Path, "/comments/") && r.Method == "DELETE" {
						w.WriteHeader(http.StatusNoContent)
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupMockFn()
			defer server.Close()

			client := NewWithConfig(&Config{
				Token:      "test-token",
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				RetryCount: 1,
				UserAgent:  "test-agent",
			})

			manager := NewPRCommentManager(client, nil)
			ctx := context.Background()

			err := manager.DeletePRComments(ctx, "testowner", "testrepo", 123)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetPRCommentStats(t *testing.T) {
	tests := []struct {
		name        string
		setupMockFn func() *httptest.Server
		expectError bool
	}{
		{
			name: "successful stats retrieval",
			setupMockFn: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/repos/testowner/testrepo/issues/123/comments" && r.Method == "GET" {
						comments := []map[string]interface{}{
							{
								"id":   1,
								"body": "<!-- gofortress-coverage-v1 --> Coverage comment 1",
							},
							{
								"id":   2,
								"body": "Regular comment",
							},
							{
								"id":   3,
								"body": "<!-- gofortress-coverage-v1 --> Coverage comment 2",
							},
						}
						w.Header().Set("Content-Type", "application/json")
						assert.NoError(t, json.NewEncoder(w).Encode(comments))
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
				}))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupMockFn()
			defer server.Close()

			client := NewWithConfig(&Config{
				Token:      "test-token",
				BaseURL:    server.URL,
				Timeout:    5 * time.Second,
				RetryCount: 1,
				UserAgent:  "test-agent",
			})

			manager := NewPRCommentManager(client, nil)
			ctx := context.Background()

			stats, err := manager.GetPRCommentStats(ctx, "testowner", "testrepo", 123)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, stats)
				require.Equal(t, 2, stats["total_comments"])  // Should find 2 coverage comments
				require.Equal(t, true, stats["has_comments"]) // Should have comments
			}
		})
	}
}
