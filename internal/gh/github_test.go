package gh

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	internalerrors "github.com/mrz1836/go-broadcast/internal/errors"
)

var errTestAPIError = errors.New("API error")

func TestListBranches(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	branches := []Branch{
		{Name: "master", Protected: true},
		{Name: "develop", Protected: false},
	}
	output, err := json.Marshal(branches)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/branches", "--paginate"}).
		Return(output, nil)

	result, err := client.ListBranches(ctx, "org/repo")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "master", result[0].Name)
	assert.True(t, result[0].Protected)

	mockRunner.AssertExpectations(t)
}

func TestListBranches_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/branches", "--paginate"}).
		Return(nil, internalerrors.ErrTest)

	result, err := client.ListBranches(ctx, "org/repo")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list branches")

	mockRunner.AssertExpectations(t)
}

func TestGetBranch(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	branch := Branch{Name: "main", Protected: true}
	output, err := json.Marshal(branch)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/branches/main"}).
		Return(output, nil)

	result, err := client.GetBranch(ctx, "org/repo", "main")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "main", result.Name)
	assert.True(t, result.Protected)

	mockRunner.AssertExpectations(t)
}

func TestGetBranch_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/branches/nonexistent"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	result, err := client.GetBranch(ctx, "org/repo", "nonexistent")
	require.Error(t, err)
	assert.Equal(t, ErrBranchNotFound, err)
	assert.Nil(t, result)

	mockRunner.AssertExpectations(t)
}

func TestCreatePR(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title: "Test PR",
		Body:  "Test description",
		Head:  "feature",
		Base:  "master",
	}

	pr := PR{
		Number: 42,
		Title:  req.Title,
		Body:   req.Body,
		State:  "open",
	}
	output, err := json.Marshal(pr)
	require.NoError(t, err)

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(output, nil)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)
	assert.Equal(t, "Test PR", result.Title)

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_HeadFormatting tests that the head branch is properly formatted with owner prefix
func TestCreatePR_HeadFormatting(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title: "Test PR",
		Body:  "Test description",
		Head:  "feature-branch",
		Base:  "master",
	}

	pr := PR{
		Number: 42,
		Title:  req.Title,
		Body:   req.Body,
		State:  "open",
	}
	output, err := json.Marshal(pr)
	require.NoError(t, err)

	// Capture the JSON data to verify head formatting
	var capturedJSON []byte
	mockRunner.On("RunWithInput", ctx, mock.MatchedBy(func(jsonData []byte) bool {
		capturedJSON = jsonData
		var prData map[string]interface{}
		if unmarshalErr := json.Unmarshal(jsonData, &prData); unmarshalErr != nil {
			return false
		}
		// Verify that head is formatted as "org:feature-branch"
		head, ok := prData["head"].(string)
		return ok && head == "org:feature-branch"
	}), "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(output, nil)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)

	// Verify the captured JSON has the correctly formatted head
	var prData map[string]interface{}
	err = json.Unmarshal(capturedJSON, &prData)
	require.NoError(t, err)
	assert.Equal(t, "org:feature-branch", prData["head"])

	mockRunner.AssertExpectations(t)
}

func TestGetPR(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	pr := PR{
		Number: 42,
		Title:  "Test PR",
		State:  "open",
	}
	output, err := json.Marshal(pr)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/42"}).
		Return(output, nil)

	result, err := client.GetPR(ctx, "org/repo", 42)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)
	assert.Equal(t, "Test PR", result.Title)

	mockRunner.AssertExpectations(t)
}

func TestListPRs(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	prs := []PR{
		{Number: 1, Title: "PR 1", State: "open"},
		{Number: 2, Title: "PR 2", State: "open"},
	}
	output, err := json.Marshal(prs)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls?state=open", "--paginate"}).
		Return(output, nil)

	result, err := client.ListPRs(ctx, "org/repo", "open")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].Number)

	mockRunner.AssertExpectations(t)
}

func TestGetFile(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	file := File{
		Path:    "README.md",
		Content: "IyBUZXN0IENvbnRlbnQ=", // Base64 encoded "# Test Content"
		SHA:     "abc123",
	}
	output, err := json.Marshal(file)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/contents/README.md?ref=main"}).
		Return(output, nil)

	result, err := client.GetFile(ctx, "org/repo", "README.md", "main")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "README.md", result.Path)
	assert.Equal(t, "# Test Content", string(result.Content))
	assert.Equal(t, "abc123", result.SHA)

	mockRunner.AssertExpectations(t)
}

func TestGetCommit(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	commit := Commit{
		SHA: "abc123",
	}
	commit.Commit.Message = "Test commit"
	commit.Commit.Author.Name = "Test User"
	commit.Commit.Author.Email = "test@example.com"
	output, err := json.Marshal(commit)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/abc123"}).
		Return(output, nil)

	result, err := client.GetCommit(ctx, "org/repo", "abc123")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "abc123", result.SHA)
	assert.Equal(t, "Test commit", result.Commit.Message)

	mockRunner.AssertExpectations(t)
}

func TestNewClient_NotAuthenticated(t *testing.T) {
	// Save original PATH
	oldPath := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	// Create a temporary directory with a fake gh that fails auth
	tmpDir := t.TempDir()
	fakeGH := filepath.Join(tmpDir, "gh")
	script := `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
    exit 1
fi
`
	err := os.WriteFile(fakeGH, []byte(script), 0o700) //nolint:gosec // Executable script needs execute permission
	require.NoError(t, err)

	// Add temp dir to PATH
	err = os.Setenv("PATH", tmpDir+":"+oldPath)
	require.NoError(t, err)

	client, err := NewClient(context.Background(), logrus.New(), nil)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

// TestNewClient_GHNotFound tests behavior when gh CLI is not found in PATH
func TestNewClient_GHNotFound(t *testing.T) {
	// Save original PATH
	oldPath := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	// Set PATH to empty directory
	tmpDir := t.TempDir()
	err := os.Setenv("PATH", tmpDir)
	require.NoError(t, err)

	client, err := NewClient(context.Background(), logrus.New(), nil)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.ErrorIs(t, err, ErrGHNotFound)
}

// TestListBranches_JSONUnmarshalError tests error handling when JSON unmarshaling fails
func TestListBranches_JSONUnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Return invalid JSON
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/branches", "--paginate"}).
		Return([]byte("invalid json"), nil)

	result, err := client.ListBranches(ctx, "org/repo")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse branches")

	mockRunner.AssertExpectations(t)
}

// TestGetBranch_JSONUnmarshalError tests error handling when JSON unmarshaling fails
func TestGetBranch_JSONUnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Return invalid JSON
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/branches/main"}).
		Return([]byte("invalid json"), nil)

	result, err := client.GetBranch(ctx, "org/repo", "main")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse branch")

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_JSONMarshalError tests error handling when JSON marshaling fails
func TestCreatePR_JSONMarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Create a request that would cause JSON marshal issues
	// In practice, this is very unlikely to happen with simple structs
	// but we test the error path for completeness
	req := PRRequest{
		Title: "Test PR",
		Body:  "Test description",
		Head:  "feature",
		Base:  "master",
	}

	// For this test, we'll override the marshaling by testing the response parsing error instead
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return([]byte("invalid json response"), nil)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse PR response")

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_RunWithInputError tests error handling when RunWithInput fails
func TestCreatePR_RunWithInputError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title: "Test PR",
		Body:  "Test description",
		Head:  "feature",
		Base:  "master",
	}

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(nil, internalerrors.ErrTest)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create PR")

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_ValidationFailedError tests error handling for HTTP 422 validation failures
func TestCreatePR_ValidationFailedError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title: "Test PR",
		Body:  "Test body",
		Head:  "feature-branch",
		Base:  "main",
	}

	// Mock a 422 validation failed error
	validationErr := &CommandError{
		Stderr: "gh: Validation Failed (HTTP 422)\nA pull request already exists for mrz1836:feature-branch.",
	}

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(nil, validationErr)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.Error(t, err)
	assert.Nil(t, result)
	require.ErrorIs(t, err, ErrPRValidationFailed)
	assert.Contains(t, err.Error(), "failed to create PR with head 'org:feature-branch' and base 'main'")

	mockRunner.AssertExpectations(t)
}

// TestGetPR_NotFound tests error handling when PR is not found
func TestGetPR_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/999"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	result, err := client.GetPR(ctx, "org/repo", 999)
	require.Error(t, err)
	assert.Equal(t, ErrPRNotFound, err)
	assert.Nil(t, result)

	mockRunner.AssertExpectations(t)
}

// TestGetPR_JSONUnmarshalError tests error handling when JSON unmarshaling fails
func TestGetPR_JSONUnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/42"}).
		Return([]byte("invalid json"), nil)

	result, err := client.GetPR(ctx, "org/repo", 42)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse PR")

	mockRunner.AssertExpectations(t)
}

// TestListPRs_Error tests error handling when listing PRs fails
func TestListPRs_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls", "--paginate"}).
		Return(nil, internalerrors.ErrTest)

	result, err := client.ListPRs(ctx, "org/repo", "all")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list PRs")

	mockRunner.AssertExpectations(t)
}

// TestListPRs_JSONUnmarshalError tests error handling when JSON unmarshaling fails
func TestListPRs_JSONUnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls", "--paginate"}).
		Return([]byte("invalid json"), nil)

	result, err := client.ListPRs(ctx, "org/repo", "all")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse PRs")

	mockRunner.AssertExpectations(t)
}

// TestListPRs_WithEmptyState tests listing PRs with empty state parameter
func TestListPRs_WithEmptyState(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	prs := []PR{
		{Number: 1, Title: "PR 1", State: "open"},
	}
	output, err := json.Marshal(prs)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls", "--paginate"}).
		Return(output, nil)

	result, err := client.ListPRs(ctx, "org/repo", "")
	require.NoError(t, err)
	assert.Len(t, result, 1)

	mockRunner.AssertExpectations(t)
}

// TestGetFile_NotFound tests error handling when file is not found
func TestGetFile_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/contents/nonexistent.txt"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	result, err := client.GetFile(ctx, "org/repo", "nonexistent.txt", "")
	require.Error(t, err)
	assert.Equal(t, ErrFileNotFound, err)
	assert.Nil(t, result)

	mockRunner.AssertExpectations(t)
}

// TestGetFile_JSONUnmarshalError tests error handling when JSON unmarshaling fails
func TestGetFile_JSONUnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/contents/file.txt"}).
		Return([]byte("invalid json"), nil)

	result, err := client.GetFile(ctx, "org/repo", "file.txt", "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse file")

	mockRunner.AssertExpectations(t)
}

// TestGetFile_Base64DecodeError tests error handling when base64 decoding fails
func TestGetFile_Base64DecodeError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	file := File{
		Path:    "test.txt",
		Content: "invalid base64!@#$%^&*()",
		SHA:     "abc123",
	}
	output, err := json.Marshal(file)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/contents/test.txt"}).
		Return(output, nil)

	result, err := client.GetFile(ctx, "org/repo", "test.txt", "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode file content")

	mockRunner.AssertExpectations(t)
}

// TestGetFile_WithRef tests getting file with specific ref
func TestGetFile_WithRef(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	file := File{
		Path:    "README.md",
		Content: "IyBUZXN0IENvbnRlbnQ=", // Base64 encoded "# Test Content"
		SHA:     "abc123",
	}
	output, err := json.Marshal(file)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/contents/README.md?ref=develop"}).
		Return(output, nil)

	result, err := client.GetFile(ctx, "org/repo", "README.md", "develop")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "README.md", result.Path)
	assert.Equal(t, "# Test Content", string(result.Content))

	mockRunner.AssertExpectations(t)
}

// TestGetCommit_NotFound tests error handling when commit is not found
func TestGetCommit_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/nonexistent"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	result, err := client.GetCommit(ctx, "org/repo", "nonexistent")
	require.Error(t, err)
	assert.Equal(t, ErrCommitNotFound, err)
	assert.Nil(t, result)

	mockRunner.AssertExpectations(t)
}

// TestGetCommit_JSONUnmarshalError tests error handling when JSON unmarshaling fails
func TestGetCommit_JSONUnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/abc123"}).
		Return([]byte("invalid json"), nil)

	result, err := client.GetCommit(ctx, "org/repo", "abc123")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse commit")

	mockRunner.AssertExpectations(t)
}

// TestIsNotFoundError tests the isNotFoundError helper function
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "404 error",
			err:      &CommandError{Stderr: "HTTP 404: Not Found"},
			expected: true,
		},
		{
			name:     "Not Found error",
			err:      &CommandError{Stderr: "Resource Not Found"},
			expected: true,
		},
		{
			name:     "Could not resolve error",
			err:      &CommandError{Stderr: "could not resolve repository"},
			expected: true,
		},
		{
			name:     "Different error",
			err:      &CommandError{Stderr: "500 Internal Server Error"},
			expected: false,
		},
		{
			name:     "Regular error",
			err:      internalerrors.ErrTest,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsValidationFailedError tests the isValidationFailedError helper function
func TestIsValidationFailedError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "HTTP 422 error",
			err:      &CommandError{Stderr: "HTTP 422: Unprocessable Entity"},
			expected: true,
		},
		{
			name:     "Validation Failed error",
			err:      &CommandError{Stderr: "Validation Failed"},
			expected: true,
		},
		{
			name:     "Unprocessable Entity error",
			err:      &CommandError{Stderr: "Unprocessable Entity"},
			expected: true,
		},
		{
			name:     "422 in error message",
			err:      &CommandError{Stderr: "gh: Validation Failed (HTTP 422)"},
			expected: true,
		},
		{
			name:     "Different error",
			err:      &CommandError{Stderr: "500 Internal Server Error"},
			expected: false,
		},
		{
			name:     "404 error",
			err:      &CommandError{Stderr: "HTTP 404: Not Found"},
			expected: false,
		},
		{
			name:     "Regular error",
			err:      internalerrors.ErrTest,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidationFailedError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsBranchProtectionError tests the IsBranchProtectionError helper function
func TestIsBranchProtectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Base branch policy prohibits merge",
			err:      &CommandError{Stderr: "X Pull request owner/repo#123 is not mergeable: the base branch policy prohibits the merge.\nTo have the pull request merged after all the requirements have been met, add the `--auto` flag."},
			expected: true,
		},
		{
			name:     "Add auto flag suggestion",
			err:      &CommandError{Stderr: "merge failed: add the `--auto` flag to enable auto-merge"},
			expected: true,
		},
		{
			name:     "Not mergeable base branch policy",
			err:      &CommandError{Stderr: "not mergeable: the base branch policy requires status checks"},
			expected: true,
		},
		{
			name:     "Required status checks",
			err:      &CommandError{Stderr: "required status checks have not passed"},
			expected: true,
		},
		{
			name:     "Required reviews",
			err:      &CommandError{Stderr: "required reviews are not satisfied"},
			expected: true,
		},
		{
			name:     "Different error - not found",
			err:      &CommandError{Stderr: "404 Not Found"},
			expected: false,
		},
		{
			name:     "Different error - validation",
			err:      &CommandError{Stderr: "422 Validation Failed"},
			expected: false,
		},
		{
			name:     "Regular error",
			err:      internalerrors.ErrTest,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBranchProtectionError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetFile_WithBase64Whitespace tests file content with base64 whitespace
func TestGetFile_WithBase64Whitespace(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	file := File{
		Path:    "test.txt",
		Content: "  IyBUZXN0IENvbnRlbnQ=  \n\t", // Base64 with whitespace
		SHA:     "abc123",
	}
	output, err := json.Marshal(file)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/contents/test.txt"}).
		Return(output, nil)

	result, err := client.GetFile(ctx, "org/repo", "test.txt", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "# Test Content", string(result.Content))

	mockRunner.AssertExpectations(t)
}

// TestNewClient_AuthCheckError tests when gh auth status command itself fails
func TestNewClient_AuthCheckError(t *testing.T) {
	// Save original PATH
	oldPath := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	// Create a temporary directory with a fake gh that returns auth error
	tmpDir := t.TempDir()
	fakeGH := filepath.Join(tmpDir, "gh")
	script := `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
    echo "authentication failed" >&2
    exit 1
fi
`
	err := os.WriteFile(fakeGH, []byte(script), 0o700) //nolint:gosec // Executable script needs execute permission
	require.NoError(t, err)

	// Add temp dir to PATH
	err = os.Setenv("PATH", tmpDir+":"+oldPath)
	require.NoError(t, err)

	client, err := NewClient(context.Background(), logrus.New(), nil)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "gh auth status failed")
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

// TestNewClient_Success tests successful client creation
func TestNewClient_Success(t *testing.T) {
	// Save original PATH
	oldPath := os.Getenv("PATH")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	// Create a temporary directory with a fake gh that succeeds
	tmpDir := t.TempDir()
	fakeGH := filepath.Join(tmpDir, "gh")
	script := `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
    echo "Logged in to github.com"
    exit 0
fi
`
	err := os.WriteFile(fakeGH, []byte(script), 0o700) //nolint:gosec // Executable script needs execute permission
	require.NoError(t, err)

	// Add temp dir to PATH
	err = os.Setenv("PATH", tmpDir+":"+oldPath)
	require.NoError(t, err)

	client, err := NewClient(context.Background(), logrus.New(), nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// TestGetBranch_OtherCommandError tests GetBranch with non-404 command errors
func TestGetBranch_OtherCommandError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Return a command error that's not a 404
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/branches/main"}).
		Return(nil, &CommandError{Stderr: "500 Internal Server Error"})

	result, err := client.GetBranch(ctx, "org/repo", "main")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get branch")
	assert.NotEqual(t, ErrBranchNotFound, err) // Should be different from not found error

	mockRunner.AssertExpectations(t)
}

// TestGetPR_OtherCommandError tests GetPR with non-404 command errors
func TestGetPR_OtherCommandError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Return a command error that's not a 404
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/42"}).
		Return(nil, &CommandError{Stderr: "401 Unauthorized"})

	result, err := client.GetPR(ctx, "org/repo", 42)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get PR")
	assert.NotEqual(t, ErrPRNotFound, err) // Should be different from not found error

	mockRunner.AssertExpectations(t)
}

// TestGetFile_OtherCommandError tests GetFile with non-404 command errors
func TestGetFile_OtherCommandError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Return a command error that's not a 404
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/contents/test.txt"}).
		Return(nil, &CommandError{Stderr: "403 Forbidden"})

	result, err := client.GetFile(ctx, "org/repo", "test.txt", "")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get file")
	assert.NotEqual(t, ErrFileNotFound, err) // Should be different from not found error

	mockRunner.AssertExpectations(t)
}

// TestGetCommit_OtherCommandError tests GetCommit with non-404 command errors
func TestGetCommit_OtherCommandError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Return a command error that's not a 404
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/abc123"}).
		Return(nil, &CommandError{Stderr: "422 Unprocessable Entity"})

	result, err := client.GetCommit(ctx, "org/repo", "abc123")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get commit")
	assert.NotEqual(t, ErrCommitNotFound, err) // Should be different from not found error

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_WithAssignees tests PR creation with assignees
func TestCreatePR_WithAssignees(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title:     "Test PR",
		Body:      "Test description",
		Head:      "feature",
		Base:      "master",
		Assignees: []string{"user1", "user2"},
	}

	pr := PR{
		Number: 42,
		Title:  req.Title,
		Body:   req.Body,
		State:  "open",
	}
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Expect PR creation call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(prOutput, nil)

	// Expect assignees call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/issues/42/assignees", "--method", "POST", "--input", "-"}).
		Return([]byte("{}"), nil)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)
	assert.Equal(t, "Test PR", result.Title)

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_WithReviewers tests PR creation with reviewers
func TestCreatePR_WithReviewers(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title:         "Test PR",
		Body:          "Test description",
		Head:          "feature",
		Base:          "master",
		Reviewers:     []string{"reviewer1", "reviewer2"},
		TeamReviewers: []string{"team1"},
	}

	pr := PR{
		Number: 42,
		Title:  req.Title,
		Body:   req.Body,
		State:  "open",
	}
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Expect PR creation call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(prOutput, nil)

	// Expect reviewers call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls/42/requested_reviewers", "--method", "POST", "--input", "-"}).
		Return([]byte("{}"), nil)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)
	assert.Equal(t, "Test PR", result.Title)

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_WithAssigneesAndReviewers tests PR creation with both assignees and reviewers
func TestCreatePR_WithAssigneesAndReviewers(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title:         "Test PR",
		Body:          "Test description",
		Head:          "feature",
		Base:          "master",
		Assignees:     []string{"assignee1"},
		Reviewers:     []string{"reviewer1"},
		TeamReviewers: []string{"team1"},
	}

	pr := PR{
		Number: 42,
		Title:  req.Title,
		Body:   req.Body,
		State:  "open",
	}
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Expect PR creation call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(prOutput, nil)

	// Expect assignees call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/issues/42/assignees", "--method", "POST", "--input", "-"}).
		Return([]byte("{}"), nil)

	// Expect reviewers call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls/42/requested_reviewers", "--method", "POST", "--input", "-"}).
		Return([]byte("{}"), nil)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)
	assert.Equal(t, "Test PR", result.Title)

	mockRunner.AssertExpectations(t)
}

// TestGetCurrentUser tests getting the authenticated user
func TestGetCurrentUser(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	user := User{
		Login: "testuser",
		ID:    12345,
		Name:  "Test User",
		Email: "test@example.com",
	}
	userOutput, err := json.Marshal(user)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "user"}).
		Return(userOutput, nil)

	result, err := client.GetCurrentUser(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "testuser", result.Login)
	assert.Equal(t, 12345, result.ID)
	assert.Equal(t, "Test User", result.Name)
	assert.Equal(t, "test@example.com", result.Email)

	// Second call should use cached value
	result2, err := client.GetCurrentUser(ctx)
	require.NoError(t, err)
	require.NotNil(t, result2)
	assert.Equal(t, result.Login, result2.Login)

	// Verify API was only called once due to caching
	mockRunner.AssertNumberOfCalls(t, "Run", 1)
}

// TestGetCurrentUser_Error tests error handling when getting current user fails
func TestGetCurrentUser_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "user"}).
		Return([]byte{}, errTestAPIError)

	result, err := client.GetCurrentUser(ctx)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get current user")
}

// TestCreatePR_AssigneesFailure tests that PR creation succeeds even if setting assignees fails
func TestCreatePR_AssigneesFailure(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title:     "Test PR",
		Body:      "Test description",
		Head:      "feature",
		Base:      "master",
		Assignees: []string{"user1"},
	}

	pr := PR{
		Number: 42,
		Title:  req.Title,
		Body:   req.Body,
		State:  "open",
	}
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Expect PR creation call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(prOutput, nil)

	// Expect assignees call to fail
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/issues/42/assignees", "--method", "POST", "--input", "-"}).
		Return(nil, internalerrors.ErrTest)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err) // Should still succeed
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)

	mockRunner.AssertExpectations(t)
}

func TestGetRepository(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	repo := Repository{
		Name:             "test-repo",
		FullName:         "org/test-repo",
		DefaultBranch:    "main",
		AllowSquashMerge: true,
		AllowMergeCommit: true,
		AllowRebaseMerge: false,
	}
	output, err := json.Marshal(repo)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/test-repo"}).
		Return(output, nil)

	result, err := client.GetRepository(ctx, "org/test-repo")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "test-repo", result.Name)
	assert.Equal(t, "org/test-repo", result.FullName)
	assert.Equal(t, "main", result.DefaultBranch)
	assert.True(t, result.AllowSquashMerge)
	assert.True(t, result.AllowMergeCommit)
	assert.False(t, result.AllowRebaseMerge)

	mockRunner.AssertExpectations(t)
}

func TestGetRepository_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/nonexistent"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	result, err := client.GetRepository(ctx, "org/nonexistent")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "repository not found")

	mockRunner.AssertExpectations(t)
}

func TestGetRepository_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/test-repo"}).
		Return(nil, errTestAPIError)

	result, err := client.GetRepository(ctx, "org/test-repo")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get repository")

	mockRunner.AssertExpectations(t)
}

func TestReviewPR(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "review", "123", "--repo", "org/repo", "--approve", "--body", "LGTM"}).
		Return([]byte(""), nil)

	err := client.ReviewPR(ctx, "org/repo", 123, "LGTM")
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

func TestReviewPR_EmptyMessage(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "review", "456", "--repo", "org/repo", "--approve"}).
		Return([]byte(""), nil)

	err := client.ReviewPR(ctx, "org/repo", 456, "")
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

func TestReviewPR_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "review", "999", "--repo", "org/repo", "--approve", "--body", "LGTM"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	err := client.ReviewPR(ctx, "org/repo", 999, "LGTM")
	require.Error(t, err)
	assert.Equal(t, ErrPRNotFound, err)

	mockRunner.AssertExpectations(t)
}

func TestReviewPR_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "review", "123", "--repo", "org/repo", "--approve", "--body", "LGTM"}).
		Return(nil, errTestAPIError)

	err := client.ReviewPR(ctx, "org/repo", 123, "LGTM")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "review PR #123")

	mockRunner.AssertExpectations(t)
}

func TestMergePR_Squash(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "123", "--repo", "org/repo", "--squash"}).
		Return([]byte(""), nil)

	err := client.MergePR(ctx, "org/repo", 123, MergeMethodSquash)
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

func TestMergePR_Merge(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "456", "--repo", "org/repo", "--merge"}).
		Return([]byte(""), nil)

	err := client.MergePR(ctx, "org/repo", 456, MergeMethodMerge)
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

func TestMergePR_Rebase(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "789", "--repo", "org/repo", "--rebase"}).
		Return([]byte(""), nil)

	err := client.MergePR(ctx, "org/repo", 789, MergeMethodRebase)
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

func TestMergePR_InvalidMethod(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	err := client.MergePR(ctx, "org/repo", 123, MergeMethod("invalid"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported merge method")

	// Should not call runner for invalid method
	mockRunner.AssertExpectations(t)
}

func TestMergePR_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "999", "--repo", "org/repo", "--squash"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	err := client.MergePR(ctx, "org/repo", 999, MergeMethodSquash)
	require.Error(t, err)
	assert.Equal(t, ErrPRNotFound, err)

	mockRunner.AssertExpectations(t)
}

func TestMergePR_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "123", "--repo", "org/repo", "--squash"}).
		Return(nil, errTestAPIError)

	err := client.MergePR(ctx, "org/repo", 123, MergeMethodSquash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "merge PR #123")

	mockRunner.AssertExpectations(t)
}

func TestEnableAutoMergePR_Squash(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "123", "--repo", "org/repo", "--auto", "--squash"}).
		Return(nil, nil)

	err := client.EnableAutoMergePR(ctx, "org/repo", 123, MergeMethodSquash)
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

func TestEnableAutoMergePR_Merge(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "123", "--repo", "org/repo", "--auto", "--merge"}).
		Return(nil, nil)

	err := client.EnableAutoMergePR(ctx, "org/repo", 123, MergeMethodMerge)
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

func TestEnableAutoMergePR_Rebase(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "123", "--repo", "org/repo", "--auto", "--rebase"}).
		Return(nil, nil)

	err := client.EnableAutoMergePR(ctx, "org/repo", 123, MergeMethodRebase)
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

func TestEnableAutoMergePR_InvalidMethod(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	err := client.EnableAutoMergePR(ctx, "org/repo", 123, MergeMethod("invalid"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported merge method")

	// Should not call runner for invalid method
	mockRunner.AssertExpectations(t)
}

func TestEnableAutoMergePR_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "123", "--repo", "org/repo", "--auto", "--squash"}).
		Return(nil, errTestAPIError)

	err := client.EnableAutoMergePR(ctx, "org/repo", 123, MergeMethodSquash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enable auto-merge for PR #123")

	mockRunner.AssertExpectations(t)
}

// TestGetGitTree tests successful retrieval of a Git tree
func TestGetGitTree(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	gitTree := GitTree{
		SHA:       "abc123def456",
		Truncated: false,
		Tree: []GitTreeNode{
			{Path: "README.md", Mode: "100644", Type: "blob", SHA: "sha1"},
			{Path: "src", Mode: "040000", Type: "tree", SHA: "sha2"},
		},
	}
	output, err := json.Marshal(gitTree)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/git/trees/abc123def456"}).
		Return(output, nil)

	result, err := client.GetGitTree(ctx, "org/repo", "abc123def456", false)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "abc123def456", result.SHA)
	assert.False(t, result.Truncated)
	assert.Len(t, result.Tree, 2)
	assert.Equal(t, "README.md", result.Tree[0].Path)

	mockRunner.AssertExpectations(t)
}

// TestGetGitTree_Recursive tests recursive Git tree retrieval
func TestGetGitTree_Recursive(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	gitTree := GitTree{
		SHA:       "abc123def456",
		Truncated: false,
		Tree: []GitTreeNode{
			{Path: "README.md", Mode: "100644", Type: "blob", SHA: "sha1"},
			{Path: "src/main.go", Mode: "100644", Type: "blob", SHA: "sha3"},
		},
	}
	output, err := json.Marshal(gitTree)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/git/trees/abc123def456?recursive=1"}).
		Return(output, nil)

	result, err := client.GetGitTree(ctx, "org/repo", "abc123def456", true)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Tree, 2)

	mockRunner.AssertExpectations(t)
}

// TestGetGitTree_NotFound tests error handling when tree is not found
func TestGetGitTree_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/git/trees/nonexistent"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	result, err := client.GetGitTree(ctx, "org/repo", "nonexistent", false)
	require.Error(t, err)
	assert.Nil(t, result)
	require.ErrorIs(t, err, ErrGitTreeNotFound)

	mockRunner.AssertExpectations(t)
}

// TestGetGitTree_Error tests error handling for API errors
func TestGetGitTree_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/git/trees/abc123"}).
		Return(nil, errTestAPIError)

	result, err := client.GetGitTree(ctx, "org/repo", "abc123", false)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get git tree")

	mockRunner.AssertExpectations(t)
}

// TestGetGitTree_JSONUnmarshalError tests error handling for invalid JSON
func TestGetGitTree_JSONUnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/git/trees/abc123"}).
		Return([]byte("invalid json"), nil)

	result, err := client.GetGitTree(ctx, "org/repo", "abc123", false)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parse git tree")

	mockRunner.AssertExpectations(t)
}

// TestSearchAssignedPRs tests successful PR search
func TestSearchAssignedPRs(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	searchResults := []struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		URL     string `json:"url"`
		State   string `json:"state"`
		IsDraft bool   `json:"isDraft"`
	}{
		{Number: 1, Title: "PR 1", URL: "https://github.com/org/repo/pull/1", State: "OPEN", IsDraft: false},
		{Number: 2, Title: "PR 2", URL: "https://github.com/org/repo/pull/2", State: "OPEN", IsDraft: false},
	}
	output, err := json.Marshal(searchResults)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"search", "prs", "--assignee", "@me", "--state", "open", "--limit", "1000", "--json", "number,title,url,isDraft,state"}).
		Return(output, nil)

	result, err := client.SearchAssignedPRs(ctx)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].Number)
	assert.Equal(t, "PR 1", result[0].Title)

	mockRunner.AssertExpectations(t)
}

// TestSearchAssignedPRs_FiltersDrafts tests that draft PRs are filtered out
func TestSearchAssignedPRs_FiltersDrafts(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	searchResults := []struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		URL     string `json:"url"`
		State   string `json:"state"`
		IsDraft bool   `json:"isDraft"`
	}{
		{Number: 1, Title: "PR 1", URL: "https://github.com/org/repo/pull/1", State: "OPEN", IsDraft: false},
		{Number: 2, Title: "Draft PR", URL: "https://github.com/org/repo/pull/2", State: "OPEN", IsDraft: true},
		{Number: 3, Title: "PR 3", URL: "https://github.com/org/repo/pull/3", State: "OPEN", IsDraft: false},
	}
	output, err := json.Marshal(searchResults)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"search", "prs", "--assignee", "@me", "--state", "open", "--limit", "1000", "--json", "number,title,url,isDraft,state"}).
		Return(output, nil)

	result, err := client.SearchAssignedPRs(ctx)
	require.NoError(t, err)
	assert.Len(t, result, 2) // Only non-draft PRs
	assert.Equal(t, 1, result[0].Number)
	assert.Equal(t, 3, result[1].Number)

	mockRunner.AssertExpectations(t)
}

// TestSearchAssignedPRs_Error tests error handling for search failures
func TestSearchAssignedPRs_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"search", "prs", "--assignee", "@me", "--state", "open", "--limit", "1000", "--json", "number,title,url,isDraft,state"}).
		Return(nil, errTestAPIError)

	result, err := client.SearchAssignedPRs(ctx)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "search assigned PRs")

	mockRunner.AssertExpectations(t)
}

// TestSearchAssignedPRs_JSONUnmarshalError tests error handling for invalid JSON
func TestSearchAssignedPRs_JSONUnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"search", "prs", "--assignee", "@me", "--state", "open", "--limit", "1000", "--json", "number,title,url,isDraft,state"}).
		Return([]byte("invalid json"), nil)

	result, err := client.SearchAssignedPRs(ctx)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parse search results")

	mockRunner.AssertExpectations(t)
}

// TestSearchAssignedPRs_InvalidURL tests handling of PRs with invalid URLs
func TestSearchAssignedPRs_InvalidURL(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	searchResults := []struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		URL     string `json:"url"`
		State   string `json:"state"`
		IsDraft bool   `json:"isDraft"`
	}{
		{Number: 1, Title: "PR 1", URL: "https://github.com/org/repo/pull/1", State: "OPEN", IsDraft: false},
		{Number: 2, Title: "Invalid URL PR", URL: "invalid-url", State: "OPEN", IsDraft: false}, // Invalid URL
	}
	output, err := json.Marshal(searchResults)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"search", "prs", "--assignee", "@me", "--state", "open", "--limit", "1000", "--json", "number,title,url,isDraft,state"}).
		Return(output, nil)

	result, err := client.SearchAssignedPRs(ctx)
	require.NoError(t, err)
	assert.Len(t, result, 1) // Only valid URL PR
	assert.Equal(t, 1, result[0].Number)

	mockRunner.AssertExpectations(t)
}

// TestGetPRReviews tests successful retrieval of PR reviews
func TestGetPRReviews(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	reviews := []Review{
		{ID: 1, User: User{Login: "reviewer1"}, State: "APPROVED", Body: "LGTM"},
		{ID: 2, User: User{Login: "reviewer2"}, State: "CHANGES_REQUESTED", Body: "Please fix issues"},
	}
	output, err := json.Marshal(reviews)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123/reviews"}).
		Return(output, nil)

	result, err := client.GetPRReviews(ctx, "org/repo", 123)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "reviewer1", result[0].User.Login)
	assert.Equal(t, "APPROVED", result[0].State)

	mockRunner.AssertExpectations(t)
}

// TestGetPRReviews_NotFound tests error handling when PR is not found
func TestGetPRReviews_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/999/reviews"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	result, err := client.GetPRReviews(ctx, "org/repo", 999)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrPRNotFound, err)

	mockRunner.AssertExpectations(t)
}

// TestGetPRReviews_Error tests error handling for API errors
func TestGetPRReviews_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123/reviews"}).
		Return(nil, errTestAPIError)

	result, err := client.GetPRReviews(ctx, "org/repo", 123)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get reviews for PR #123")

	mockRunner.AssertExpectations(t)
}

// TestGetPRReviews_JSONUnmarshalError tests error handling for invalid JSON
func TestGetPRReviews_JSONUnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123/reviews"}).
		Return([]byte("invalid json"), nil)

	result, err := client.GetPRReviews(ctx, "org/repo", 123)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parse PR reviews")

	mockRunner.AssertExpectations(t)
}

// TestHasApprovedReview tests checking for approved review
func TestHasApprovedReview(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	reviews := []Review{
		{ID: 1, User: User{Login: "reviewer1"}, State: "APPROVED"},
		{ID: 2, User: User{Login: "reviewer2"}, State: "CHANGES_REQUESTED"},
	}
	output, err := json.Marshal(reviews)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123/reviews"}).
		Return(output, nil)

	hasApproval, err := client.HasApprovedReview(ctx, "org/repo", 123, "reviewer1")
	require.NoError(t, err)
	assert.True(t, hasApproval)

	mockRunner.AssertExpectations(t)
}

// TestHasApprovedReview_NoApproval tests when user has not approved
func TestHasApprovedReview_NoApproval(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	reviews := []Review{
		{ID: 1, User: User{Login: "reviewer1"}, State: "CHANGES_REQUESTED"},
		{ID: 2, User: User{Login: "reviewer2"}, State: "APPROVED"},
	}
	output, err := json.Marshal(reviews)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123/reviews"}).
		Return(output, nil)

	hasApproval, err := client.HasApprovedReview(ctx, "org/repo", 123, "reviewer1")
	require.NoError(t, err)
	assert.False(t, hasApproval)

	mockRunner.AssertExpectations(t)
}

// TestHasApprovedReview_UserNotReviewed tests when user has not reviewed at all
func TestHasApprovedReview_UserNotReviewed(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	reviews := []Review{
		{ID: 1, User: User{Login: "other_reviewer"}, State: "APPROVED"},
	}
	output, err := json.Marshal(reviews)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123/reviews"}).
		Return(output, nil)

	hasApproval, err := client.HasApprovedReview(ctx, "org/repo", 123, "reviewer1")
	require.NoError(t, err)
	assert.False(t, hasApproval)

	mockRunner.AssertExpectations(t)
}

// TestHasApprovedReview_LatestReviewMatters tests that only the latest review matters
func TestHasApprovedReview_LatestReviewMatters(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// User first approved, then requested changes
	reviews := []Review{
		{ID: 1, User: User{Login: "reviewer1"}, State: "APPROVED"},
		{ID: 2, User: User{Login: "reviewer1"}, State: "CHANGES_REQUESTED"},
	}
	output, err := json.Marshal(reviews)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123/reviews"}).
		Return(output, nil)

	hasApproval, err := client.HasApprovedReview(ctx, "org/repo", 123, "reviewer1")
	require.NoError(t, err)
	assert.False(t, hasApproval) // Latest review is CHANGES_REQUESTED

	mockRunner.AssertExpectations(t)
}

// TestHasApprovedReview_Error tests error handling
func TestHasApprovedReview_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123/reviews"}).
		Return(nil, errTestAPIError)

	hasApproval, err := client.HasApprovedReview(ctx, "org/repo", 123, "reviewer1")
	require.Error(t, err)
	assert.False(t, hasApproval)

	mockRunner.AssertExpectations(t)
}

// TestAddPRComment tests adding a comment to a PR
func TestAddPRComment(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("RunWithInput", ctx, mock.MatchedBy(func(data []byte) bool {
		return string(data) == `{"body":"Test comment"}`
	}), "gh", []string{"api", "repos/org/repo/issues/123/comments", "--method", "POST", "--input", "-"}).
		Return([]byte(`{"id": 1}`), nil)

	err := client.AddPRComment(ctx, "org/repo", 123, "Test comment")
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

// TestAddPRComment_Error tests error handling when adding comment fails
func TestAddPRComment_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/issues/123/comments", "--method", "POST", "--input", "-"}).
		Return(nil, errTestAPIError)

	err := client.AddPRComment(ctx, "org/repo", 123, "Test comment")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add PR comment")

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_WithLabels tests PR creation with labels
func TestCreatePR_WithLabels(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title:  "Test PR",
		Body:   "Test description",
		Head:   "feature",
		Base:   "master",
		Labels: []string{"bug", "high-priority"},
	}

	pr := PR{
		Number: 42,
		Title:  req.Title,
		Body:   req.Body,
		State:  "open",
	}
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Expect PR creation call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(prOutput, nil)

	// Expect labels call
	mockRunner.On("RunWithInput", ctx, mock.MatchedBy(func(data []byte) bool {
		return string(data) == `{"labels":["bug","high-priority"]}`
	}), "gh", []string{"api", "repos/org/repo/issues/42/labels", "--method", "POST", "--input", "-"}).
		Return([]byte("{}"), nil)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_LabelsFailure tests that PR creation succeeds even if setting labels fails
func TestCreatePR_LabelsFailure(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title:  "Test PR",
		Body:   "Test description",
		Head:   "feature",
		Base:   "master",
		Labels: []string{"bug"},
	}

	pr := PR{
		Number: 42,
		Title:  req.Title,
		Body:   req.Body,
		State:  "open",
	}
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Expect PR creation call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(prOutput, nil)

	// Expect labels call to fail
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/issues/42/labels", "--method", "POST", "--input", "-"}).
		Return(nil, internalerrors.ErrTest)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err) // Should still succeed
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_InvalidRepoFormat tests error handling for invalid repo format
func TestCreatePR_InvalidRepoFormat(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	req := PRRequest{
		Title: "Test PR",
		Body:  "Test description",
		Head:  "feature",
		Base:  "master",
	}

	result, err := client.CreatePR(ctx, "invalid-repo-format", req)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parse repo")

	// Should not call runner for invalid repo format
	mockRunner.AssertExpectations(t)
}

// TestGetCurrentUser_Concurrent tests that concurrent calls to GetCurrentUser are safe
func TestGetCurrentUser_Concurrent(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	user := User{
		Login: "testuser",
		ID:    12345,
		Name:  "Test User",
		Email: "test@example.com",
	}
	userOutput, err := json.Marshal(user)
	require.NoError(t, err)

	// Allow any number of calls since we're testing concurrency
	mockRunner.On("Run", ctx, "gh", []string{"api", "user"}).
		Return(userOutput, nil).Maybe()

	// Run concurrent goroutines to test race condition safety
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			result, err := client.GetCurrentUser(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, "testuser", result.Login)
		}()
	}

	wg.Wait()
}

// TestHasApprovedReview_EmptyUserLogin tests that reviews with empty user login are skipped
func TestHasApprovedReview_EmptyUserLogin(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Reviews with one having empty user login
	reviews := []Review{
		{
			ID:    1,
			User:  User{Login: ""}, // Empty login should be skipped
			State: "APPROVED",
		},
		{
			ID:    2,
			User:  User{Login: "testuser"},
			State: "APPROVED",
		},
	}
	reviewOutput, err := json.Marshal(reviews)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/42/reviews"}).
		Return(reviewOutput, nil)

	// Should find testuser's approval, but not panic on empty login
	hasApproval, err := client.HasApprovedReview(ctx, "org/repo", 42, "testuser")
	require.NoError(t, err)
	assert.True(t, hasApproval)

	// Empty login user should not be found
	hasApproval, err = client.HasApprovedReview(ctx, "org/repo", 42, "")
	require.NoError(t, err)
	assert.False(t, hasApproval)

	mockRunner.AssertExpectations(t)
}

// TestSearchAssignedPRs_InvalidURLFiltering tests that invalid PR URLs are filtered out gracefully
func TestSearchAssignedPRs_InvalidURLFiltering(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Search results with some invalid URLs
	searchResults := `[
		{"number": 1, "title": "Valid PR", "url": "https://github.com/owner/repo/pull/1", "state": "OPEN", "isDraft": false},
		{"number": 2, "title": "Invalid URL PR", "url": "invalid-url", "state": "OPEN", "isDraft": false},
		{"number": 3, "title": "Short URL PR", "url": "https://github.com", "state": "OPEN", "isDraft": false}
	]`

	mockRunner.On("Run", ctx, "gh", []string{"search", "prs", "--assignee", "@me", "--state", "open", "--limit", "1000", "--json", "number,title,url,isDraft,state"}).
		Return([]byte(searchResults), nil)

	prs, err := client.SearchAssignedPRs(ctx)
	require.NoError(t, err)

	// Only the valid PR should be returned
	assert.Len(t, prs, 1)
	assert.Equal(t, 1, prs[0].Number)
	assert.Equal(t, "owner/repo", prs[0].Repo)

	mockRunner.AssertExpectations(t)
}

// TestCreatePR_NilLogger tests that PR creation works when logger is nil
func TestCreatePR_NilLogger(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	// Create client with nil logger
	client := NewClientWithRunner(mockRunner, nil)

	req := PRRequest{
		Title:     "Test PR",
		Body:      "Test description",
		Head:      "feature",
		Base:      "master",
		Assignees: []string{"user1"}, // This will fail but should not panic
	}

	pr := PR{
		Number: 42,
		Title:  req.Title,
		Body:   req.Body,
		State:  "open",
	}
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Expect PR creation call
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(prOutput, nil)

	// Expect assignees call to fail - should not panic even with nil logger
	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/issues/42/assignees", "--method", "POST", "--input", "-"}).
		Return(nil, internalerrors.ErrTest)

	// Should not panic with nil logger
	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)

	mockRunner.AssertExpectations(t)
}

// TestGetPRCheckStatus_AllPassed tests check status when all checks have passed
func TestGetPRCheckStatus_AllPassed(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Mock PR to get head SHA
	pr := PR{
		Number: 123,
		Title:  "Test PR",
		State:  "open",
	}
	pr.Head.SHA = "abc123def456"
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Mock check runs response - all passed
	checkRunsResponse := CheckRunsResponse{
		TotalCount: 3,
		CheckRuns: []CheckRun{
			{ID: 1, Name: "CI / Build", Status: "completed", Conclusion: "success"},
			{ID: 2, Name: "CI / Tests", Status: "completed", Conclusion: "success"},
			{ID: 3, Name: "CI / Lint", Status: "completed", Conclusion: "neutral"},
		},
	}
	checkRunsOutput, err := json.Marshal(checkRunsResponse)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123"}).
		Return(prOutput, nil)
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/abc123def456/check-runs"}).
		Return(checkRunsOutput, nil)

	result, err := client.GetPRCheckStatus(ctx, "org/repo", 123)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, 3, result.Completed)
	assert.Equal(t, 3, result.Passed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 0, result.Running)
	assert.Equal(t, 0, result.Skipped)
	assert.True(t, result.AllPassed())
	assert.False(t, result.HasRunningChecks())
	assert.False(t, result.HasFailedChecks())

	mockRunner.AssertExpectations(t)
}

// TestGetPRCheckStatus_SomeRunning tests check status when some checks are still running
func TestGetPRCheckStatus_SomeRunning(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Mock PR to get head SHA
	pr := PR{
		Number: 123,
		Title:  "Test PR",
		State:  "open",
	}
	pr.Head.SHA = "abc123def456"
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Mock check runs response - some running
	checkRunsResponse := CheckRunsResponse{
		TotalCount: 4,
		CheckRuns: []CheckRun{
			{ID: 1, Name: "CI / Build", Status: "completed", Conclusion: "success"},
			{ID: 2, Name: "CI / Tests", Status: "in_progress", Conclusion: ""},
			{ID: 3, Name: "CI / Lint", Status: "queued", Conclusion: ""},
			{ID: 4, Name: "CI / Deploy", Status: "completed", Conclusion: "success"},
		},
	}
	checkRunsOutput, err := json.Marshal(checkRunsResponse)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123"}).
		Return(prOutput, nil)
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/abc123def456/check-runs"}).
		Return(checkRunsOutput, nil)

	result, err := client.GetPRCheckStatus(ctx, "org/repo", 123)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 4, result.Total)
	assert.Equal(t, 2, result.Completed)
	assert.Equal(t, 2, result.Passed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 2, result.Running)
	assert.False(t, result.AllPassed())
	assert.True(t, result.HasRunningChecks())
	assert.False(t, result.HasFailedChecks())

	// Verify running check names
	runningNames := result.RunningCheckNames()
	assert.Len(t, runningNames, 2)
	assert.Contains(t, runningNames, "CI / Tests")
	assert.Contains(t, runningNames, "CI / Lint")

	mockRunner.AssertExpectations(t)
}

// TestGetPRCheckStatus_SomeFailed tests check status when some checks have failed
func TestGetPRCheckStatus_SomeFailed(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Mock PR to get head SHA
	pr := PR{
		Number: 123,
		Title:  "Test PR",
		State:  "open",
	}
	pr.Head.SHA = "abc123def456"
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Mock check runs response - some failed
	checkRunsResponse := CheckRunsResponse{
		TotalCount: 5,
		CheckRuns: []CheckRun{
			{ID: 1, Name: "CI / Build", Status: "completed", Conclusion: "success"},
			{ID: 2, Name: "CI / Tests", Status: "completed", Conclusion: "failure"},
			{ID: 3, Name: "CI / Lint", Status: "completed", Conclusion: "canceled"},
			{ID: 4, Name: "CI / Security", Status: "completed", Conclusion: "timed_out"},
			{ID: 5, Name: "CI / Deploy", Status: "completed", Conclusion: "action_required"},
		},
	}
	checkRunsOutput, err := json.Marshal(checkRunsResponse)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123"}).
		Return(prOutput, nil)
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/abc123def456/check-runs"}).
		Return(checkRunsOutput, nil)

	result, err := client.GetPRCheckStatus(ctx, "org/repo", 123)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 5, result.Total)
	assert.Equal(t, 5, result.Completed)
	assert.Equal(t, 1, result.Passed)
	assert.Equal(t, 4, result.Failed)
	assert.Equal(t, 0, result.Running)
	assert.False(t, result.AllPassed())
	assert.False(t, result.HasRunningChecks())
	assert.True(t, result.HasFailedChecks())

	// Verify failed check names
	failedNames := result.FailedCheckNames()
	assert.Len(t, failedNames, 4)
	assert.Contains(t, failedNames, "CI / Tests")
	assert.Contains(t, failedNames, "CI / Lint")
	assert.Contains(t, failedNames, "CI / Security")
	assert.Contains(t, failedNames, "CI / Deploy")

	mockRunner.AssertExpectations(t)
}

// TestGetPRCheckStatus_NoChecks tests check status when no checks are configured
func TestGetPRCheckStatus_NoChecks(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Mock PR to get head SHA
	pr := PR{
		Number: 123,
		Title:  "Test PR",
		State:  "open",
	}
	pr.Head.SHA = "abc123def456"
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Mock check runs response - no checks
	checkRunsResponse := CheckRunsResponse{
		TotalCount: 0,
		CheckRuns:  []CheckRun{},
	}
	checkRunsOutput, err := json.Marshal(checkRunsResponse)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123"}).
		Return(prOutput, nil)
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/abc123def456/check-runs"}).
		Return(checkRunsOutput, nil)

	result, err := client.GetPRCheckStatus(ctx, "org/repo", 123)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.Total)
	assert.True(t, result.NoChecks())
	assert.False(t, result.AllPassed()) // No checks = not all passed
	assert.False(t, result.HasRunningChecks())
	assert.False(t, result.HasFailedChecks())
	assert.Equal(t, "no checks configured", result.Summary())

	mockRunner.AssertExpectations(t)
}

// TestGetPRCheckStatus_WithSkipped tests check status with skipped checks
func TestGetPRCheckStatus_WithSkipped(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Mock PR to get head SHA
	pr := PR{
		Number: 123,
		Title:  "Test PR",
		State:  "open",
	}
	pr.Head.SHA = "abc123def456"
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	// Mock check runs response - with skipped
	checkRunsResponse := CheckRunsResponse{
		TotalCount: 3,
		CheckRuns: []CheckRun{
			{ID: 1, Name: "CI / Build", Status: "completed", Conclusion: "success"},
			{ID: 2, Name: "CI / Tests", Status: "completed", Conclusion: "success"},
			{ID: 3, Name: "CI / Deploy", Status: "completed", Conclusion: "skipped"},
		},
	}
	checkRunsOutput, err := json.Marshal(checkRunsResponse)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123"}).
		Return(prOutput, nil)
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/abc123def456/check-runs"}).
		Return(checkRunsOutput, nil)

	result, err := client.GetPRCheckStatus(ctx, "org/repo", 123)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, result.Total)
	assert.Equal(t, 3, result.Completed)
	assert.Equal(t, 2, result.Passed)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 0, result.Running)
	assert.True(t, result.AllPassed()) // Skipped counts as passed for AllPassed check

	mockRunner.AssertExpectations(t)
}

// TestGetPRCheckStatus_PRNotFound tests error when PR is not found
func TestGetPRCheckStatus_PRNotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/999"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	result, err := client.GetPRCheckStatus(ctx, "org/repo", 999)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get PR #999 for check status")

	mockRunner.AssertExpectations(t)
}

// TestGetPRCheckStatus_CheckRunsAPIError tests error when check runs API fails
func TestGetPRCheckStatus_CheckRunsAPIError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Mock PR to get head SHA
	pr := PR{
		Number: 123,
		Title:  "Test PR",
		State:  "open",
	}
	pr.Head.SHA = "abc123def456"
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123"}).
		Return(prOutput, nil)
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/abc123def456/check-runs"}).
		Return(nil, errTestAPIError)

	result, err := client.GetPRCheckStatus(ctx, "org/repo", 123)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "get check runs for PR #123")

	mockRunner.AssertExpectations(t)
}

// TestGetPRCheckStatus_JSONUnmarshalError tests error when check runs JSON is invalid
func TestGetPRCheckStatus_JSONUnmarshalError(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	// Mock PR to get head SHA
	pr := PR{
		Number: 123,
		Title:  "Test PR",
		State:  "open",
	}
	pr.Head.SHA = "abc123def456"
	prOutput, err := json.Marshal(pr)
	require.NoError(t, err)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls/123"}).
		Return(prOutput, nil)
	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/commits/abc123def456/check-runs"}).
		Return([]byte("invalid json"), nil)

	result, err := client.GetPRCheckStatus(ctx, "org/repo", 123)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parse check runs response")

	mockRunner.AssertExpectations(t)
}

// TestCheckStatusSummary_Summary tests the Summary method
func TestCheckStatusSummary_Summary(t *testing.T) {
	tests := []struct {
		name     string
		summary  CheckStatusSummary
		expected string
	}{
		{
			name:     "No checks",
			summary:  CheckStatusSummary{Total: 0},
			expected: "no checks configured",
		},
		{
			name: "All passed",
			summary: CheckStatusSummary{
				Total: 3, Completed: 3, Passed: 3,
			},
			expected: "3/3 checks complete (3 passed)",
		},
		{
			name: "With running",
			summary: CheckStatusSummary{
				Total: 5, Completed: 3, Passed: 3, Running: 2,
			},
			expected: "3/5 checks complete (3 passed, 2 running)",
		},
		{
			name: "With failed",
			summary: CheckStatusSummary{
				Total: 4, Completed: 4, Passed: 2, Failed: 2,
			},
			expected: "4/4 checks complete (2 passed, 2 failed)",
		},
		{
			name: "With skipped",
			summary: CheckStatusSummary{
				Total: 3, Completed: 3, Passed: 2, Skipped: 1,
			},
			expected: "3/3 checks complete (2 passed, 1 skipped)",
		},
		{
			name: "Mixed state",
			summary: CheckStatusSummary{
				Total: 6, Completed: 4, Passed: 2, Skipped: 1, Failed: 1, Running: 2,
			},
			expected: "4/6 checks complete (2 passed, 1 skipped, 1 failed, 2 running)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.summary.Summary()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBypassMergePR_Squash tests bypass merge with squash method
func TestBypassMergePR_Squash(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "123", "--repo", "org/repo", "--admin", "--squash"}).
		Return([]byte(""), nil)

	err := client.BypassMergePR(ctx, "org/repo", 123, MergeMethodSquash)
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

// TestBypassMergePR_Rebase tests bypass merge with rebase method
func TestBypassMergePR_Rebase(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "123", "--repo", "org/repo", "--admin", "--rebase"}).
		Return([]byte(""), nil)

	err := client.BypassMergePR(ctx, "org/repo", 123, MergeMethodRebase)
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

// TestBypassMergePR_Merge tests bypass merge with merge method
func TestBypassMergePR_Merge(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "123", "--repo", "org/repo", "--admin", "--merge"}).
		Return([]byte(""), nil)

	err := client.BypassMergePR(ctx, "org/repo", 123, MergeMethodMerge)
	require.NoError(t, err)

	mockRunner.AssertExpectations(t)
}

// TestBypassMergePR_InvalidMethod tests bypass merge with invalid method
func TestBypassMergePR_InvalidMethod(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	err := client.BypassMergePR(ctx, "org/repo", 123, MergeMethod("invalid"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported merge method")

	mockRunner.AssertExpectations(t)
}

// TestBypassMergePR_NotFound tests bypass merge when PR not found
func TestBypassMergePR_NotFound(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "999", "--repo", "org/repo", "--admin", "--squash"}).
		Return(nil, &CommandError{Stderr: "404 Not Found"})

	err := client.BypassMergePR(ctx, "org/repo", 999, MergeMethodSquash)
	require.Error(t, err)
	assert.Equal(t, ErrPRNotFound, err)

	mockRunner.AssertExpectations(t)
}

// TestBypassMergePR_Error tests bypass merge when API returns error
func TestBypassMergePR_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"pr", "merge", "123", "--repo", "org/repo", "--admin", "--squash"}).
		Return(nil, errTestAPIError)

	err := client.BypassMergePR(ctx, "org/repo", 123, MergeMethodSquash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bypass merge PR #123")

	mockRunner.AssertExpectations(t)
}
