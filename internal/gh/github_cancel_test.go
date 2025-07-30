package gh

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGitHubClient_ClosePR(t *testing.T) {
	tests := []struct {
		name        string
		repo        string
		number      int
		comment     string
		mockSetup   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful PR close without comment",
			repo:    "owner/repo",
			number:  123,
			comment: "",
			mockSetup: func(mr *MockCommandRunner) {
				// Only expect UpdatePR call (no comment)
				mr.On("RunWithInput", mock.Anything, mock.MatchedBy(func(data []byte) bool {
					return string(data) == `{"state":"closed"}`
				}), "gh", []string{"api", "repos/owner/repo/pulls/123", "--method", "PATCH", "--input", "-"}).Return([]byte(""), nil)
			},
			wantErr: false,
		},
		{
			name:    "successful PR close with comment",
			repo:    "owner/repo",
			number:  456,
			comment: "Closing this PR due to issues",
			mockSetup: func(mr *MockCommandRunner) {
				// Expect comment API call first
				mr.On("RunWithInput", mock.Anything, mock.MatchedBy(func(data []byte) bool {
					return string(data) == `{"body":"Closing this PR due to issues"}`
				}), "gh", []string{"api", "repos/owner/repo/issues/456/comments", "--method", "POST", "--input", "-"}).Return([]byte(""), nil)

				// Then expect UpdatePR call
				mr.On("RunWithInput", mock.Anything, mock.MatchedBy(func(data []byte) bool {
					return string(data) == `{"state":"closed"}`
				}), "gh", []string{"api", "repos/owner/repo/pulls/456", "--method", "PATCH", "--input", "-"}).Return([]byte(""), nil)
			},
			wantErr: false,
		},
		{
			name:    "UpdatePR fails",
			repo:    "owner/repo",
			number:  789,
			comment: "",
			mockSetup: func(mr *MockCommandRunner) {
				mr.On("RunWithInput", mock.Anything, mock.Anything, "gh", []string{"api", "repos/owner/repo/pulls/789", "--method", "PATCH", "--input", "-"}).Return([]byte(""), assert.AnError)
			},
			wantErr:     true,
			errContains: "update PR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := &MockCommandRunner{}
			tt.mockSetup(mockRunner)

			client := NewClientWithRunner(mockRunner, nil)

			err := client.ClosePR(context.Background(), tt.repo, tt.number, tt.comment)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			mockRunner.AssertExpectations(t)
		})
	}
}

func TestGitHubClient_DeleteBranch(t *testing.T) {
	tests := []struct {
		name        string
		repo        string
		branch      string
		mockSetup   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful branch deletion",
			repo:   "owner/repo",
			branch: "feature-branch",
			mockSetup: func(mr *MockCommandRunner) {
				mr.On("Run", mock.Anything, "gh", []string{"api", "repos/owner/repo/git/refs/heads/feature-branch", "--method", "DELETE"}).Return([]byte(""), nil)
			},
			wantErr: false,
		},
		{
			name:   "branch not found",
			repo:   "owner/repo",
			branch: "nonexistent-branch",
			mockSetup: func(mr *MockCommandRunner) {
				mr.On("Run", mock.Anything, "gh", []string{"api", "repos/owner/repo/git/refs/heads/nonexistent-branch", "--method", "DELETE"}).Return([]byte(""), &mockNotFoundError{})
			},
			wantErr:     true,
			errContains: "branch not found",
		},
		{
			name:   "API error",
			repo:   "owner/repo",
			branch: "protected-branch",
			mockSetup: func(mr *MockCommandRunner) {
				mr.On("Run", mock.Anything, "gh", []string{"api", "repos/owner/repo/git/refs/heads/protected-branch", "--method", "DELETE"}).Return([]byte(""), assert.AnError)
			},
			wantErr:     true,
			errContains: "delete branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := &MockCommandRunner{}
			tt.mockSetup(mockRunner)

			client := NewClientWithRunner(mockRunner, nil)

			err := client.DeleteBranch(context.Background(), tt.repo, tt.branch)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			mockRunner.AssertExpectations(t)
		})
	}
}

func TestGitHubClient_UpdatePR(t *testing.T) {
	tests := []struct {
		name        string
		repo        string
		number      int
		updates     PRUpdate
		mockSetup   func(*MockCommandRunner)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful PR update - close",
			repo:   "owner/repo",
			number: 123,
			updates: PRUpdate{
				State: stringPtr("closed"),
			},
			mockSetup: func(mr *MockCommandRunner) {
				mr.On("RunWithInput", mock.Anything, mock.MatchedBy(func(data []byte) bool {
					return string(data) == `{"state":"closed"}`
				}), "gh", []string{"api", "repos/owner/repo/pulls/123", "--method", "PATCH", "--input", "-"}).Return([]byte(""), nil)
			},
			wantErr: false,
		},
		{
			name:   "successful PR update - body",
			repo:   "owner/repo",
			number: 456,
			updates: PRUpdate{
				Body: stringPtr("Updated description"),
			},
			mockSetup: func(mr *MockCommandRunner) {
				mr.On("RunWithInput", mock.Anything, mock.MatchedBy(func(data []byte) bool {
					return string(data) == `{"body":"Updated description"}`
				}), "gh", []string{"api", "repos/owner/repo/pulls/456", "--method", "PATCH", "--input", "-"}).Return([]byte(""), nil)
			},
			wantErr: false,
		},
		{
			name:   "PR not found",
			repo:   "owner/repo",
			number: 999,
			updates: PRUpdate{
				State: stringPtr("closed"),
			},
			mockSetup: func(mr *MockCommandRunner) {
				mr.On("RunWithInput", mock.Anything, mock.Anything, "gh", []string{"api", "repos/owner/repo/pulls/999", "--method", "PATCH", "--input", "-"}).Return([]byte(""), &mockNotFoundError{})
			},
			wantErr:     true,
			errContains: "pull request not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := &MockCommandRunner{}
			tt.mockSetup(mockRunner)

			client := NewClientWithRunner(mockRunner, nil)

			err := client.UpdatePR(context.Background(), tt.repo, tt.number, tt.updates)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}

			mockRunner.AssertExpectations(t)
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// mockNotFoundError implements an error that looks like a GitHub 404
type mockNotFoundError struct{}

func (e *mockNotFoundError) Error() string {
	return "HTTP 404: Not Found"
}
