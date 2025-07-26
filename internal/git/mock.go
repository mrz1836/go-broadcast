package git

import (
	"context"

	"github.com/mrz1836/go-broadcast/internal/testutil"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

// Clone mock implementation
func (m *MockClient) Clone(ctx context.Context, url, path string) error {
	args := m.Called(ctx, url, path)
	return testutil.ExtractError(args)
}

// Checkout mock implementation
func (m *MockClient) Checkout(ctx context.Context, repoPath, branch string) error {
	args := m.Called(ctx, repoPath, branch)
	return testutil.ExtractError(args)
}

// CreateBranch mock implementation
func (m *MockClient) CreateBranch(ctx context.Context, repoPath, branch string) error {
	args := m.Called(ctx, repoPath, branch)
	return testutil.ExtractError(args)
}

// Add mock implementation
func (m *MockClient) Add(ctx context.Context, repoPath string, paths ...string) error {
	args := m.Called(ctx, repoPath, paths)
	return testutil.ExtractError(args)
}

// Commit mock implementation
func (m *MockClient) Commit(ctx context.Context, repoPath, message string) error {
	args := m.Called(ctx, repoPath, message)
	return testutil.ExtractError(args)
}

// Push mock implementation
func (m *MockClient) Push(ctx context.Context, repoPath, remote, branch string, force bool) error {
	args := m.Called(ctx, repoPath, remote, branch, force)
	return testutil.ExtractError(args)
}

// Diff mock implementation
func (m *MockClient) Diff(ctx context.Context, repoPath string, staged bool) (string, error) {
	args := m.Called(ctx, repoPath, staged)
	return testutil.ExtractStringResult(args)
}

// GetCurrentBranch mock implementation
func (m *MockClient) GetCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	args := m.Called(ctx, repoPath)
	return testutil.ExtractStringResult(args)
}

// GetRemoteURL mock implementation
func (m *MockClient) GetRemoteURL(ctx context.Context, repoPath, remote string) (string, error) {
	args := m.Called(ctx, repoPath, remote)
	return testutil.ExtractStringResult(args)
}
