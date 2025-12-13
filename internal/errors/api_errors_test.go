package errors //nolint:revive,nolintlint // internal test package, name conflict intentional

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test error variables for use in tests
var (
	errTestCommand = errors.New("command failed")
	errTestAPI     = errors.New("API error")
	errTestGeneric = errors.New("test error")
)

func TestGitOperationError(t *testing.T) {
	baseErr := errTestCommand

	tests := []struct {
		name      string
		operation string
		context   string
		err       error
		want      string
		wantNil   bool
	}{
		{
			name:      "clone operation error",
			operation: "clone",
			context:   "user/repo",
			err:       baseErr,
			want:      "git operation failed: clone 'user/repo': command failed",
		},
		{
			name:      "checkout operation error",
			operation: "checkout",
			context:   "feature-branch",
			err:       baseErr,
			want:      "git operation failed: checkout 'feature-branch': command failed",
		},
		{
			name:      "nil error returns nil",
			operation: "clone",
			context:   "user/repo",
			err:       nil,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GitOperationError(tt.operation, tt.context, tt.err)
			if tt.wantNil {
				assert.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.want)
				require.ErrorIs(t, err, errGitOperationTemplate)
				require.ErrorIs(t, err, baseErr)
			}
		})
	}
}

func TestGitConvenienceFunctions(t *testing.T) {
	baseErr := errTestGeneric

	tests := []struct {
		name     string
		fn       func(string, error) error
		context  string
		err      error
		wantText string
	}{
		{
			name:     "GitCloneError",
			fn:       GitCloneError,
			context:  "user/repo",
			err:      baseErr,
			wantText: "git operation failed: clone 'user/repo': test error",
		},
		{
			name:     "GitCheckoutError",
			fn:       GitCheckoutError,
			context:  "main",
			err:      baseErr,
			wantText: "git operation failed: checkout 'main': test error",
		},
		{
			name:     "GitAddError",
			fn:       GitAddError,
			context:  "*.go",
			err:      baseErr,
			wantText: "git operation failed: add '*.go': test error",
		},
		{
			name:     "GitCommitError",
			fn:       GitCommitError,
			context:  "feat: new feature",
			err:      baseErr,
			wantText: "git operation failed: commit 'feat: new feature': test error",
		},
		{
			name:     "GitPushError",
			fn:       GitPushError,
			context:  "origin main",
			err:      baseErr,
			wantText: "git operation failed: push 'origin main': test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.context, tt.err)
			require.EqualError(t, err, tt.wantText)
			require.ErrorIs(t, err, errGitOperationTemplate)
			require.ErrorIs(t, err, baseErr)
		})
	}
}

func TestGitHubAPIError(t *testing.T) {
	baseErr := errTestAPI

	tests := []struct {
		name      string
		operation string
		context   string
		err       error
		want      string
		wantNil   bool
	}{
		{
			name:      "create PR error",
			operation: "create pull request",
			context:   "user/repo",
			err:       baseErr,
			want:      "GitHub API operation failed: create pull request 'user/repo': API error",
		},
		{
			name:      "list branches error",
			operation: "list branches",
			context:   "user/repo",
			err:       baseErr,
			want:      "GitHub API operation failed: list branches 'user/repo': API error",
		},
		{
			name:      "nil error returns nil",
			operation: "get file",
			context:   "user/repo",
			err:       nil,
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GitHubAPIError(tt.operation, tt.context, tt.err)
			if tt.wantNil {
				assert.NoError(t, err)
			} else {
				require.EqualError(t, err, tt.want)
				require.ErrorIs(t, err, errGitHubAPITemplate)
				require.ErrorIs(t, err, baseErr)
			}
		})
	}
}

func TestGitHubConvenienceFunctions(t *testing.T) {
	baseErr := errTestGeneric

	tests := []struct {
		name     string
		fn       func(string, string, error) error
		resource string
		context  string
		err      error
		wantText string
	}{
		{
			name:     "GitHubListError",
			fn:       GitHubListError,
			resource: "branches",
			context:  "user/repo",
			err:      baseErr,
			wantText: "GitHub API operation failed: list branches 'user/repo': test error",
		},
		{
			name:     "GitHubGetError",
			fn:       GitHubGetError,
			resource: "file",
			context:  "user/repo/file.txt",
			err:      baseErr,
			wantText: "GitHub API operation failed: get file 'user/repo/file.txt': test error",
		},
		{
			name:     "GitHubCreateError",
			fn:       GitHubCreateError,
			resource: "pull request",
			context:  "user/repo",
			err:      baseErr,
			wantText: "GitHub API operation failed: create pull request 'user/repo': test error",
		},
		{
			name:     "GitHubUpdateError",
			fn:       GitHubUpdateError,
			resource: "issue",
			context:  "user/repo#123",
			err:      baseErr,
			wantText: "GitHub API operation failed: update issue 'user/repo#123': test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn(tt.resource, tt.context, tt.err)
			require.EqualError(t, err, tt.wantText)
			require.ErrorIs(t, err, errGitHubAPITemplate)
			require.ErrorIs(t, err, baseErr)
		})
	}
}

func TestAPIResponseError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		want       string
	}{
		{
			name:       "404 not found",
			statusCode: 404,
			message:    "repository not found",
			want:       "API response error: status 404: repository not found",
		},
		{
			name:       "401 unauthorized",
			statusCode: 401,
			message:    "invalid authentication credentials",
			want:       "API response error: status 401: invalid authentication credentials",
		},
		{
			name:       "500 server error",
			statusCode: 500,
			message:    "internal server error",
			want:       "API response error: status 500: internal server error",
		},
		{
			name:       "100 continue (minimum valid)",
			statusCode: 100,
			message:    "continue",
			want:       "API response error: status 100: continue",
		},
		{
			name:       "599 (maximum valid)",
			statusCode: 599,
			message:    "custom error",
			want:       "API response error: status 599: custom error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := APIResponseError(tt.statusCode, tt.message)
			require.EqualError(t, err, tt.want)
			require.ErrorIs(t, err, errAPIResponseTemplate)
		})
	}
}

func TestAPIResponseError_InvalidStatusCodes(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		wantContain string
	}{
		{
			name:        "negative status code",
			statusCode:  -1,
			wantContain: "invalid status -1",
		},
		{
			name:        "zero status code",
			statusCode:  0,
			wantContain: "invalid status 0",
		},
		{
			name:        "status code too low (99)",
			statusCode:  99,
			wantContain: "invalid status 99",
		},
		{
			name:        "status code too high (600)",
			statusCode:  600,
			wantContain: "invalid status 600",
		},
		{
			name:        "very large status code",
			statusCode:  9999,
			wantContain: "invalid status 9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := APIResponseError(tt.statusCode, "message")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantContain)
			// Should still wrap the template error
			require.ErrorIs(t, err, errAPIResponseTemplate)
		})
	}
}

func TestRateLimitError(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		resetTime string
		want      string
	}{
		{
			name:      "GitHub API rate limit",
			service:   "GitHub API",
			resetTime: "2024-01-01 15:00:00 UTC",
			want:      "rate limit exceeded: GitHub API: resets at 2024-01-01 15:00:00 UTC",
		},
		{
			name:      "Custom API rate limit",
			service:   "Custom API",
			resetTime: "in 30 minutes",
			want:      "rate limit exceeded: Custom API: resets at in 30 minutes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RateLimitError(tt.service, tt.resetTime)
			require.EqualError(t, err, tt.want)
			require.ErrorIs(t, err, errRateLimitTemplate)
		})
	}
}

func TestAuthenticationError(t *testing.T) {
	tests := []struct {
		name    string
		service string
		reason  string
		want    string
	}{
		{
			name:    "GitHub invalid token",
			service: "GitHub",
			reason:  "invalid token",
			want:    "authentication failed: GitHub: invalid token",
		},
		{
			name:    "API key expired",
			service: "API",
			reason:  "API key expired",
			want:    "authentication failed: API: API key expired",
		},
		{
			name:    "Missing credentials",
			service: "Service",
			reason:  "credentials not provided",
			want:    "authentication failed: Service: credentials not provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AuthenticationError(tt.service, tt.reason)
			require.EqualError(t, err, tt.want)
			require.ErrorIs(t, err, errAuthenticationTemplate)
		})
	}
}
