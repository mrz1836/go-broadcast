package git

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

// Clone mock implementation
func (m *MockClient) Clone(ctx context.Context, url, path string) error {
	args := m.Called(ctx, url, path)
	return args.Error(0)
}

// Checkout mock implementation
func (m *MockClient) Checkout(ctx context.Context, repoPath, branch string) error {
	args := m.Called(ctx, repoPath, branch)
	return args.Error(0)
}

// CreateBranch mock implementation
func (m *MockClient) CreateBranch(ctx context.Context, repoPath, branch string) error {
	args := m.Called(ctx, repoPath, branch)
	return args.Error(0)
}

// Add mock implementation
func (m *MockClient) Add(ctx context.Context, repoPath string, paths ...string) error {
	args := m.Called(ctx, repoPath, paths)
	return args.Error(0)
}

// Commit mock implementation
func (m *MockClient) Commit(ctx context.Context, repoPath, message string) error {
	args := m.Called(ctx, repoPath, message)
	return args.Error(0)
}

// Push mock implementation
func (m *MockClient) Push(ctx context.Context, repoPath, remote, branch string, force bool) error {
	args := m.Called(ctx, repoPath, remote, branch, force)
	return args.Error(0)
}

// Diff mock implementation
func (m *MockClient) Diff(ctx context.Context, repoPath string, staged bool) (string, error) {
	args := m.Called(ctx, repoPath, staged)
	return args.String(0), args.Error(1)
}

// GetCurrentBranch mock implementation
func (m *MockClient) GetCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	args := m.Called(ctx, repoPath)
	return args.String(0), args.Error(1)
}

// GetRemoteURL mock implementation
func (m *MockClient) GetRemoteURL(ctx context.Context, repoPath, remote string) (string, error) {
	args := m.Called(ctx, repoPath, remote)
	return args.String(0), args.Error(1)
}
