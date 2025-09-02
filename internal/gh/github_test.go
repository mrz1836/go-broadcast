package gh

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
