package state

import (
	"context"
	"fmt"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/stretchr/testify/mock"
)

// MockDiscoverer is a mock implementation of the Discoverer interface
type MockDiscoverer struct {
	mock.Mock
}

// DiscoverState mock implementation
func (m *MockDiscoverer) DiscoverState(ctx context.Context, cfg *config.Config) (*State, error) {
	args := m.Called(ctx, cfg)

	// Check if we have enough arguments to avoid panic
	if len(args) < 2 {
		// Fallback for incorrectly configured mocks
		if len(args) == 1 {
			if err, ok := args.Get(0).(error); ok {
				return nil, err
			}
		}
		// Return an error instead of nil,nil to avoid nil pointer dereference
		return nil, fmt.Errorf("mock not properly configured: expected 2 return values, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*State), args.Error(1)
}

// DiscoverTargetState mock implementation
func (m *MockDiscoverer) DiscoverTargetState(ctx context.Context, repo string) (*TargetState, error) {
	args := m.Called(ctx, repo)

	// Check if we have enough arguments to avoid panic
	if len(args) < 2 {
		// Fallback for incorrectly configured mocks
		if len(args) == 1 {
			if err, ok := args.Get(0).(error); ok {
				return nil, err
			}
		}
		// Return an error instead of nil,nil to avoid nil pointer dereference
		return nil, fmt.Errorf("mock not properly configured: expected 2 return values, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*TargetState), args.Error(1)
}

// ParseBranchName mock implementation
func (m *MockDiscoverer) ParseBranchName(name string) (*BranchMetadata, error) {
	args := m.Called(name)

	// Check if we have enough arguments to avoid panic
	if len(args) < 2 {
		// Fallback for incorrectly configured mocks
		if len(args) == 1 {
			if err, ok := args.Get(0).(error); ok {
				return nil, err
			}
		}
		// Return an error instead of nil,nil to avoid nil pointer dereference
		return nil, fmt.Errorf("mock not properly configured: expected 2 return values, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BranchMetadata), args.Error(1)
}
