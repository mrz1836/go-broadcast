package state

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// MockDiscoverer is a mock implementation of the Discoverer interface
type MockDiscoverer struct {
	mock.Mock
}

// NewMockDiscoverer creates a new MockDiscoverer
func NewMockDiscoverer() *MockDiscoverer {
	return &MockDiscoverer{}
}

// DiscoverState mock implementation
func (m *MockDiscoverer) DiscoverState(ctx context.Context, cfg *config.Config) (*State, error) {
	args := m.Called(ctx, cfg)
	return testutil.HandleTwoValueReturn[*State](args)
}

// DiscoverTargetState mock implementation
func (m *MockDiscoverer) DiscoverTargetState(ctx context.Context, repo, branchPrefix, targetBranch string) (*TargetState, error) {
	args := m.Called(ctx, repo, branchPrefix, targetBranch)
	return testutil.HandleTwoValueReturn[*TargetState](args)
}

// ParseBranchName mock implementation
func (m *MockDiscoverer) ParseBranchName(name string) (*BranchMetadata, error) {
	args := m.Called(name)
	return testutil.HandleTwoValueReturn[*BranchMetadata](args)
}
