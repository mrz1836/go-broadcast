package gh

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

// GetCurrentUser mock implementation
func (m *MockClient) GetCurrentUser(ctx context.Context) (*User, error) {
	args := m.Called(ctx)
	return testutil.HandleTwoValueReturn[*User](args)
}

// GetGitTree mock implementation
func (m *MockClient) GetGitTree(ctx context.Context, repo, treeSHA string, recursive bool) (*GitTree, error) {
	args := m.Called(ctx, repo, treeSHA, recursive)
	return testutil.HandleTwoValueReturn[*GitTree](args)
}

// GetRepository mock implementation
func (m *MockClient) GetRepository(ctx context.Context, repo string) (*Repository, error) {
	args := m.Called(ctx, repo)
	return testutil.HandleTwoValueReturn[*Repository](args)
}

// ReviewPR mock implementation
func (m *MockClient) ReviewPR(ctx context.Context, repo string, number int, message string) error {
	args := m.Called(ctx, repo, number, message)
	return args.Error(0)
}

// MergePR mock implementation
func (m *MockClient) MergePR(ctx context.Context, repo string, number int, method MergeMethod) error {
	args := m.Called(ctx, repo, number, method)
	return args.Error(0)
}

// EnableAutoMergePR mock implementation
func (m *MockClient) EnableAutoMergePR(ctx context.Context, repo string, number int, method MergeMethod) error {
	args := m.Called(ctx, repo, number, method)
	return args.Error(0)
}

// SearchAssignedPRs mock implementation
func (m *MockClient) SearchAssignedPRs(ctx context.Context) ([]PR, error) {
	args := m.Called(ctx)
	return testutil.HandleTwoValueReturn[[]PR](args)
}

// GetPRReviews mock implementation
func (m *MockClient) GetPRReviews(ctx context.Context, repo string, number int) ([]Review, error) {
	args := m.Called(ctx, repo, number)
	return testutil.HandleTwoValueReturn[[]Review](args)
}

// HasApprovedReview mock implementation
func (m *MockClient) HasApprovedReview(ctx context.Context, repo string, number int, username string) (bool, error) {
	args := m.Called(ctx, repo, number, username)
	return args.Bool(0), args.Error(1)
}
