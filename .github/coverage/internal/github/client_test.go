package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	token := "test-token"
	client := New(token)

	assert.NotNil(t, client)
	assert.Equal(t, token, client.token)
	assert.Equal(t, "https://api.github.com", client.baseURL)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
	assert.NotNil(t, client.config)
	assert.Equal(t, "go-broadcast-coverage/1.0", client.config.UserAgent)
}

func TestNewWithConfig(t *testing.T) {
	config := &Config{
		Token:      "custom-token",
		BaseURL:    "https://custom.api.com",
		Timeout:    60 * time.Second,
		RetryCount: 5,
		UserAgent:  "custom-agent/2.0",
	}

	client := NewWithConfig(config)

	assert.NotNil(t, client)
	assert.Equal(t, config.Token, client.token)
	assert.Equal(t, config.BaseURL, client.baseURL)
	assert.Equal(t, config.Timeout, client.httpClient.Timeout)
	assert.Equal(t, config, client.config)
}

func TestCreateComment(t *testing.T) {
	tests := []struct {
		name             string
		existingComments []Comment
		body             string
		expectedAction   string
		expectedComment  *Comment
		expectError      bool
	}{
		{
			name:             "create new comment when none exists",
			existingComments: []Comment{},
			body:             "<!-- coverage-comment -->\nNew coverage report",
			expectedAction:   "create",
			expectedComment: &Comment{
				ID:   123,
				Body: "<!-- coverage-comment -->\nNew coverage report",
			},
		},
		{
			name: "update existing coverage comment",
			existingComments: []Comment{
				{ID: 456, Body: "<!-- coverage-comment -->\nOld coverage report"},
				{ID: 789, Body: "Some other comment"},
			},
			body:           "<!-- coverage-comment -->\nUpdated coverage report",
			expectedAction: "update",
			expectedComment: &Comment{
				ID:   456,
				Body: "<!-- coverage-comment -->\nUpdated coverage report",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == "GET" && strings.Contains(r.URL.Path, "/comments"):
					// Return existing comments
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(tt.existingComments)
				case r.Method == "POST" && strings.Contains(r.URL.Path, "/comments"):
					// Create new comment
					var req CommentRequest
					_ = json.NewDecoder(r.Body).Decode(&req)
					comment := Comment{
						ID:   123,
						Body: req.Body,
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(comment)
				case r.Method == "PATCH" && strings.Contains(r.URL.Path, "/comments/"):
					// Update existing comment
					var req CommentRequest
					_ = json.NewDecoder(r.Body).Decode(&req)
					comment := Comment{
						ID:   456,
						Body: req.Body,
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(comment)
				default:
					t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			client := &Client{
				token:   "test-token",
				baseURL: server.URL,
				httpClient: &http.Client{
					Timeout: 30 * time.Second,
				},
				config: &Config{
					UserAgent: "test-agent",
				},
			}

			ctx := context.Background()
			comment, err := client.CreateComment(ctx, "owner", "repo", 123, tt.body)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, comment)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, comment)
				assert.Equal(t, tt.expectedComment.ID, comment.ID)
				assert.Equal(t, tt.expectedComment.Body, comment.Body)
			}
		})
	}
}

func TestCreateCommentError(t *testing.T) { //nolint:revive // function naming
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:          "unauthorized",
			statusCode:    401,
			responseBody:  `{"message": "Bad credentials"}`,
			expectedError: "GitHub API error: 401",
		},
		{
			name:          "not found",
			statusCode:    404,
			responseBody:  `{"message": "Not Found"}`,
			expectedError: "GitHub API error: 404",
		},
		{
			name:          "rate limited",
			statusCode:    429,
			responseBody:  `{"message": "API rate limit exceeded"}`,
			expectedError: "GitHub API error: 429",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" && strings.Contains(r.URL.Path, "/comments") {
					// Return empty comments array
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode([]Comment{})
				} else {
					w.WriteHeader(tt.statusCode)
					_, _ = w.Write([]byte(tt.responseBody))
				}
			}))
			defer server.Close()

			client := &Client{
				token:   "test-token",
				baseURL: server.URL,
				httpClient: &http.Client{
					Timeout: 30 * time.Second,
				},
				config: &Config{
					UserAgent: "test-agent",
				},
			}

			ctx := context.Background()
			comment, err := client.CreateComment(ctx, "owner", "repo", 123, "test body")

			require.Error(t, err)
			assert.Nil(t, comment)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCreateStatus(t *testing.T) { //nolint:revive // function naming
	tests := []struct {
		name         string
		status       *StatusRequest
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name: "success status",
			status: &StatusRequest{
				State:       StatusSuccess,
				TargetURL:   "https://example.com/coverage",
				Description: "Coverage: 85.0%",
				Context:     ContextCoverage,
			},
			statusCode:   201,
			responseBody: `{"state": "success"}`,
			expectError:  false,
		},
		{
			name: "failure status",
			status: &StatusRequest{
				State:       StatusFailure,
				TargetURL:   "https://example.com/coverage",
				Description: "Coverage below threshold",
				Context:     ContextCoverage,
			},
			statusCode:   201,
			responseBody: `{"state": "failure"}`,
			expectError:  false,
		},
		{
			name: "error response",
			status: &StatusRequest{
				State:   StatusSuccess,
				Context: ContextCoverage,
			},
			statusCode:   422,
			responseBody: `{"message": "Validation Failed"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Contains(t, r.URL.Path, "/statuses/")
				assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "test-agent", r.Header.Get("User-Agent"))

				var status StatusRequest
				err := json.NewDecoder(r.Body).Decode(&status)
				assert.NoError(t, err)
				assert.Equal(t, tt.status.State, status.State)
				assert.Equal(t, tt.status.Context, status.Context)

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := &Client{
				token:   "test-token",
				baseURL: server.URL,
				httpClient: &http.Client{
					Timeout: 30 * time.Second,
				},
				config: &Config{
					UserAgent: "test-agent",
				},
			}

			ctx := context.Background()
			err := client.CreateStatus(ctx, "owner", "repo", "abc123", tt.status)

			if tt.expectError {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetPullRequest(t *testing.T) { //nolint:revive // function naming
	tests := []struct {
		name         string
		prNumber     int
		statusCode   int
		responseBody string
		expectedPR   *PullRequest
		expectError  bool
	}{
		{
			name:       "successful retrieval",
			prNumber:   123,
			statusCode: 200,
			responseBody: `{
				"number": 123,
				"title": "Test PR",
				"state": "open",
				"head": {
					"sha": "abc123def456"
				}
			}`,
			expectedPR: &PullRequest{
				Number: 123,
				Title:  "Test PR",
				State:  "open",
				Head: struct {
					SHA string `json:"sha"`
				}{SHA: "abc123def456"},
			},
			expectError: false,
		},
		{
			name:         "not found",
			prNumber:     999,
			statusCode:   404,
			responseBody: `{"message": "Not Found"}`,
			expectedPR:   nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Contains(t, r.URL.Path, fmt.Sprintf("/pulls/%d", tt.prNumber))
				assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
				assert.Equal(t, "test-agent", r.Header.Get("User-Agent"))

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := &Client{
				token:   "test-token",
				baseURL: server.URL,
				httpClient: &http.Client{
					Timeout: 30 * time.Second,
				},
				config: &Config{
					UserAgent: "test-agent",
				},
			}

			ctx := context.Background()
			pr, err := client.GetPullRequest(ctx, "owner", "repo", tt.prNumber)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, pr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, pr)
				assert.Equal(t, tt.expectedPR.Number, pr.Number)
				assert.Equal(t, tt.expectedPR.Title, pr.Title)
				assert.Equal(t, tt.expectedPR.State, pr.State)
				assert.Equal(t, tt.expectedPR.Head.SHA, pr.Head.SHA)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) { //nolint:revive // function naming
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id": 123}`))
	}))
	defer server.Close()

	client := &Client{
		token:   "test-token",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: &Config{
			UserAgent: "test-agent",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.CreateComment(ctx, "owner", "repo", 123, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestContainsCoverageMarker(t *testing.T) { //nolint:revive // function naming
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "contains coverage report marker",
			body:     "Some text\n## Coverage Report\nMore text",
			expected: true,
		},
		{
			name:     "contains HTML comment marker",
			body:     "<!-- coverage-comment -->\nCoverage data here",
			expected: true,
		},
		{
			name:     "contains emoji marker",
			body:     "📊 **Coverage** increased by 2%",
			expected: true,
		},
		{
			name:     "no coverage markers",
			body:     "This is just a regular comment",
			expected: false,
		},
		{
			name:     "empty body",
			body:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsCoverageMarker(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateCoverageComment(t *testing.T) { //nolint:revive // function naming
	tests := []struct {
		name       string
		percentage float64
		trend      string
		badgeURL   string
		contains   []string
	}{
		{
			name:       "upward trend",
			percentage: 85.5,
			trend:      "up",
			badgeURL:   "https://example.com/badge.svg",
			contains: []string{
				"<!-- coverage-comment -->",
				"📈 Coverage Report",
				"**Overall Coverage: 85.5%** 🟡", // 85.5% should be yellow (80-90)
				"📈 Coverage increased",
				"![Coverage Badge](https://example.com/badge.svg)",
				"GoFortress Coverage",
			},
		},
		{
			name:       "downward trend",
			percentage: 65.0,
			trend:      "down",
			badgeURL:   "https://example.com/badge.svg",
			contains: []string{
				"📉 Coverage Report",
				"**Overall Coverage: 65.0%** 🔴",
				"📉 Coverage decreased",
			},
		},
		{
			name:       "stable trend",
			percentage: 80.0,
			trend:      "stable",
			badgeURL:   "https://example.com/badge.svg",
			contains: []string{
				"📊 Coverage Report",
				"**Overall Coverage: 80.0%** 🟡",
				"📊 Coverage stable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New("test-token")
			comment := client.GenerateCoverageComment(tt.percentage, tt.trend, tt.badgeURL)

			for _, expected := range tt.contains {
				assert.Contains(t, comment, expected, "Comment should contain: %s", expected)
			}
		})
	}
}

func TestGetPercentageEmoji(t *testing.T) { //nolint:revive // function naming
	tests := []struct {
		percentage float64
		expected   string
	}{
		{95.0, "🟢"},
		{90.0, "🟢"},
		{85.0, "🟡"},
		{80.0, "🟡"},
		{75.0, "🟠"},
		{70.0, "🟠"},
		{65.0, "🔴"},
		{50.0, "🔴"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%.1f%%", tt.percentage), func(t *testing.T) {
			result := getPercentageEmoji(tt.percentage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsHelper(t *testing.T) { //nolint:revive // function naming
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"exact match", "hello", "hello", true},
		{"substring at start", "hello world", "hello", true},
		{"substring at end", "hello world", "world", true},
		{"substring in middle", "hello beautiful world", "beautiful", true},
		{"not found", "hello world", "goodbye", false},
		{"empty substring", "hello", "", true},
		{"empty string", "", "hello", false},
		{"both empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIndexOfHelper(t *testing.T) { //nolint:revive // function naming
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{"found at start", "hello world", "hello", 0},
		{"found at end", "hello world", "world", 6},
		{"found in middle", "hello beautiful world", "beautiful", 6},
		{"not found", "hello world", "goodbye", -1},
		{"empty substring", "hello", "", 0},
		{"empty string", "", "hello", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindCoverageCommentIntegration(t *testing.T) { //nolint:revive // function naming
	comments := []Comment{
		{ID: 1, Body: "Regular comment"},
		{ID: 2, Body: "<!-- coverage-comment -->\nOld coverage data"},
		{ID: 3, Body: "Another regular comment"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/comments")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comments)
	}))
	defer server.Close()

	client := &Client{
		token:   "test-token",
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: &Config{
			UserAgent: "test-agent",
		},
	}

	ctx := context.Background()
	comment, err := client.findCoverageComment(ctx, "owner", "repo", 123)

	require.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, 2, comment.ID)
	assert.Contains(t, comment.Body, "<!-- coverage-comment -->")
}

func TestStatusConstants(t *testing.T) { //nolint:revive // function naming
	assert.Equal(t, "success", StatusSuccess)
	assert.Equal(t, "failure", StatusFailure)
	assert.Equal(t, "error", StatusError)
	assert.Equal(t, "pending", StatusPending)

	assert.Equal(t, "coverage/total", ContextCoverage)
	assert.Equal(t, "coverage/trend", ContextTrend)
}
