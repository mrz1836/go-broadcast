package gh

import (
	"context"

	"github.com/mrz1836/go-broadcast/internal/testutil"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

// ListBranches mock implementation
func (m *MockClient) ListBranches(ctx context.Context, repo string) ([]Branch, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[[]Branch](args)
}

// GetBranch mock implementation
func (m *MockClient) GetBranch(ctx context.Context, repo, branch string) (*Branch, error) {
	args := m.Called(ctx, repo, branch)
	return testutil.HandleTwoValueReturn[*Branch](args)
}

// CreatePR mock implementation
func (m *MockClient) CreatePR(ctx context.Context, repo string, req PRRequest) (*PR, error) {
	args := m.Called(ctx, repo, req)
	return testutil.HandleTwoValueReturn[*PR](args)
}

// GetPR mock implementation
func (m *MockClient) GetPR(ctx context.Context, repo string, number int) (*PR, error) {
	args := m.Called(ctx, repo, number)
	return testutil.HandleTwoValueReturn[*PR](args)
}

// ListPRs mock implementation
func (m *MockClient) ListPRs(ctx context.Context, repo, state string) ([]PR, error) {
	args := m.Called(ctx, repo, state)
	return testutil.HandleTwoValueReturn[[]PR](args)
}

// GetFile mock implementation
func (m *MockClient) GetFile(ctx context.Context, repo, path, ref string) (*FileContent, error) {
	args := m.Called(ctx, repo, path, ref)
	return testutil.HandleTwoValueReturn[*FileContent](args)
}

// GetCommit mock implementation
func (m *MockClient) GetCommit(ctx context.Context, repo, sha string) (*Commit, error) {
	args := m.Called(ctx, repo, sha)
	return testutil.HandleTwoValueReturn[*Commit](args)
}

// ClosePR mock implementation
func (m *MockClient) ClosePR(ctx context.Context, repo string, number int, comment string) error {
	args := m.Called(ctx, repo, number, comment)
	return args.Error(0)
}

// DeleteBranch mock implementation
func (m *MockClient) DeleteBranch(ctx context.Context, repo, branch string) error {
	args := m.Called(ctx, repo, branch)
	return args.Error(0)
}

// UpdatePR mock implementation
func (m *MockClient) UpdatePR(ctx context.Context, repo string, number int, updates PRUpdate) error {
	args := m.Called(ctx, repo, number, updates)
	return args.Error(0)
}
