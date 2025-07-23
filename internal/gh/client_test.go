package gh

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockClient_ListBranches(t *testing.T) {
	mockClient := new(MockClient)
	ctx := context.Background()

	// Test successful case
	expectedBranches := []Branch{
		{Name: "master", Protected: true},
		{Name: "develop", Protected: false},
	}
	mockClient.On("ListBranches", ctx, "org/repo").Return(expectedBranches, nil)

	branches, err := mockClient.ListBranches(ctx, "org/repo")
	assert.NoError(t, err)
	assert.Equal(t, expectedBranches, branches)

	// Test error case
	testErr := errors.New("API error") //nolint:err113
	mockClient.On("ListBranches", ctx, "org/error").Return(nil, testErr)

	branches, err = mockClient.ListBranches(ctx, "org/error")
	assert.Error(t, err)
	assert.Nil(t, branches)

	mockClient.AssertExpectations(t)
}

func TestMockClient_CreatePR(t *testing.T) {
	mockClient := new(MockClient)
	ctx := context.Background()

	req := PRRequest{
		Title: "Test PR",
		Body:  "Test description",
		Head:  "feature-branch",
		Base:  "master",
	}

	expectedPR := &PR{
		Number: 123,
		State:  "open",
		Title:  req.Title,
		Body:   req.Body,
	}

	mockClient.On("CreatePR", ctx, "org/repo", req).Return(expectedPR, nil)

	pr, err := mockClient.CreatePR(ctx, "org/repo", req)
	assert.NoError(t, err)
	assert.Equal(t, expectedPR, pr)

	mockClient.AssertExpectations(t)
}

func TestMockClient_GetFile(t *testing.T) {
	mockClient := new(MockClient)
	ctx := context.Background()

	expectedContent := &FileContent{
		Path:    "README.md",
		Content: []byte("# Test Content"),
		SHA:     "abc123",
	}

	mockClient.On("GetFile", ctx, "org/repo", "README.md", "master").Return(expectedContent, nil)

	content, err := mockClient.GetFile(ctx, "org/repo", "README.md", "master")
	assert.NoError(t, err)
	assert.Equal(t, expectedContent, content)

	// Test nil return
	fileErr := errors.New("file not found") //nolint:err113
	mockClient.On("GetFile", ctx, "org/repo", "missing.txt", "master").Return(nil, fileErr)

	content, err = mockClient.GetFile(ctx, "org/repo", "missing.txt", "master")
	assert.Error(t, err)
	assert.Nil(t, content)

	mockClient.AssertExpectations(t)
}

// Verify interface compliance
var _ Client = (*MockClient)(nil)
