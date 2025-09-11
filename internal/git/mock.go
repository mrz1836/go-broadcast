package git

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

// NewMockClient creates a new MockClient
func NewMockClient() *MockClient {
	return &MockClient{}
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

// AddRemote mock implementation
func (m *MockClient) AddRemote(ctx context.Context, repoPath, remoteName, remoteURL string) error {
	args := m.Called(ctx, repoPath, remoteName, remoteURL)
	return testutil.ExtractError(args)
}

// GetCurrentCommitSHA mock implementation
func (m *MockClient) GetCurrentCommitSHA(ctx context.Context, repoPath string) (string, error) {
	args := m.Called(ctx, repoPath)
	return testutil.ExtractStringResult(args)
}

// GetRepositoryInfo mock implementation
func (m *MockClient) GetRepositoryInfo(ctx context.Context, repoPath string) (*RepositoryInfo, error) {
	args := m.Called(ctx, repoPath)
	if args.Get(0) == nil {
		return nil, testutil.ExtractError(args)
	}
	return args.Get(0).(*RepositoryInfo), testutil.ExtractError(args)
}

// GetChangedFiles mock implementation
func (m *MockClient) GetChangedFiles(ctx context.Context, repoPath string) ([]string, error) {
	args := m.Called(ctx, repoPath)
	if args.Get(0) == nil {
		return nil, testutil.ExtractError(args)
	}
	return args.Get(0).([]string), testutil.ExtractError(args)
}

// BatchRemoveFiles mock implementation
func (m *MockClient) BatchRemoveFiles(ctx context.Context, repoPath string, files []string, keepLocal bool) error {
	args := m.Called(ctx, repoPath, files, keepLocal)
	return testutil.ExtractError(args)
}
