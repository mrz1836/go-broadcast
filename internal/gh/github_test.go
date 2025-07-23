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
)

func TestListBranches(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	branches := []Branch{
		{Name: "main", Protected: true},
		{Name: "develop", Protected: false},
	}
	output, _ := json.Marshal(branches)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/branches", "--paginate"}).
		Return(output, nil)

	result, err := client.ListBranches(ctx, "org/repo")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "main", result[0].Name)
	assert.True(t, result[0].Protected)

	mockRunner.AssertExpectations(t)
}

func TestListBranches_Error(t *testing.T) {
	ctx := context.Background()
	mockRunner := new(MockCommandRunner)
	client := NewClientWithRunner(mockRunner, logrus.New())

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/branches", "--paginate"}).
		Return(nil, errors.New("API error"))

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
	output, _ := json.Marshal(branch)

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
	require.NoError(t, err)
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
		Base:  "main",
	}

	pr := PR{
		Number: 42,
		Title:  req.Title,
		Body:   req.Body,
		State:  "open",
	}
	output, _ := json.Marshal(pr)

	mockRunner.On("RunWithInput", ctx, mock.Anything, "gh", []string{"api", "repos/org/repo/pulls", "--method", "POST", "--input", "-"}).
		Return(output, nil)

	result, err := client.CreatePR(ctx, "org/repo", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 42, result.Number)
	assert.Equal(t, "Test PR", result.Title)

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
	output, _ := json.Marshal(pr)

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
	output, _ := json.Marshal(prs)

	mockRunner.On("Run", ctx, "gh", []string{"api", "repos/org/repo/pulls", "--paginate", "-f", "state=open"}).
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
	output, _ := json.Marshal(file)

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
	output, _ := json.Marshal(commit)

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
	defer os.Setenv("PATH", oldPath)

	// Create a temporary directory with a fake gh that fails auth
	tmpDir := t.TempDir()
	fakeGH := filepath.Join(tmpDir, "gh")
	script := `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
    exit 1
fi
`
	err := os.WriteFile(fakeGH, []byte(script), 0755)
	require.NoError(t, err)

	// Add temp dir to PATH
	os.Setenv("PATH", tmpDir+":"+oldPath)

	client, err := NewClient(nil)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.ErrorIs(t, err, ErrNotAuthenticated)
}

