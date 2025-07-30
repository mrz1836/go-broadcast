package state

import (
	"context"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/testutil"
	"github.com/stretchr/testify/mock"
)

// MockDiscoverer is a mock implementation of the Discoverer interface
type MockDiscoverer struct {
	mock.Mock
}

// DiscoverState mock implementation
func (m *MockDiscoverer) DiscoverState(ctx context.Context, cfg *config.Config) (*State, error) {
	args := m.Called(ctx, cfg)
	return testutil.HandleTwoValueReturn[*State](args)
}

// DiscoverTargetState mock implementation
func (m *MockDiscoverer) DiscoverTargetState(ctx context.Context, repo string, branchPrefix string) (*TargetState, error) {
	args := m.Called(ctx, repo, branchPrefix)
	return testutil.HandleTwoValueReturn[*TargetState](args)
}

// ParseBranchName mock implementation
func (m *MockDiscoverer) ParseBranchName(name string) (*BranchMetadata, error) {
	args := m.Called(name)
	return testutil.HandleTwoValueReturn[*BranchMetadata](args)
}
