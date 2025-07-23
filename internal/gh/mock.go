package gh

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

// ListBranches mock implementation
func (m *MockClient) ListBranches(ctx context.Context, repo string) ([]Branch, error) {
	args := m.Called(ctx, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Branch), args.Error(1)
}

// GetBranch mock implementation
func (m *MockClient) GetBranch(ctx context.Context, repo, branch string) (*Branch, error) {
	args := m.Called(ctx, repo, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Branch), args.Error(1)
}

// CreatePR mock implementation
func (m *MockClient) CreatePR(ctx context.Context, repo string, req PRRequest) (*PR, error) {
	args := m.Called(ctx, repo, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PR), args.Error(1)
}

// GetPR mock implementation
func (m *MockClient) GetPR(ctx context.Context, repo string, number int) (*PR, error) {
	args := m.Called(ctx, repo, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PR), args.Error(1)
}

// ListPRs mock implementation
func (m *MockClient) ListPRs(ctx context.Context, repo, state string) ([]PR, error) {
	args := m.Called(ctx, repo, state)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]PR), args.Error(1)
}

// GetFile mock implementation
func (m *MockClient) GetFile(ctx context.Context, repo, path, ref string) (*FileContent, error) {
	args := m.Called(ctx, repo, path, ref)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*FileContent), args.Error(1)
}

// GetCommit mock implementation
func (m *MockClient) GetCommit(ctx context.Context, repo, sha string) (*Commit, error) {
	args := m.Called(ctx, repo, sha)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*Commit), args.Error(1)
}
