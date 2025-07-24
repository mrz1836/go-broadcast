package gh

import (
	"context"
	"fmt"

	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

// ListBranches mock implementation
func (m *MockClient) ListBranches(ctx context.Context, repo string) ([]Branch, error) {
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
	return args.Get(0).([]Branch), args.Error(1)
}

// GetBranch mock implementation
func (m *MockClient) GetBranch(ctx context.Context, repo, branch string) (*Branch, error) {
	args := m.Called(ctx, repo, branch)

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
	return args.Get(0).(*Branch), args.Error(1)
}

// CreatePR mock implementation
func (m *MockClient) CreatePR(ctx context.Context, repo string, req PRRequest) (*PR, error) {
	args := m.Called(ctx, repo, req)

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
	return args.Get(0).(*PR), args.Error(1)
}

// GetPR mock implementation
func (m *MockClient) GetPR(ctx context.Context, repo string, number int) (*PR, error) {
	args := m.Called(ctx, repo, number)

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
	return args.Get(0).(*PR), args.Error(1)
}

// ListPRs mock implementation
func (m *MockClient) ListPRs(ctx context.Context, repo, state string) ([]PR, error) {
	args := m.Called(ctx, repo, state)

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

	return args.Get(0).([]PR), args.Error(1)
}

// GetFile mock implementation
func (m *MockClient) GetFile(ctx context.Context, repo, path, ref string) (*FileContent, error) {
	args := m.Called(ctx, repo, path, ref)

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

	return args.Get(0).(*FileContent), args.Error(1)
}

// GetCommit mock implementation
func (m *MockClient) GetCommit(ctx context.Context, repo, sha string) (*Commit, error) {
	args := m.Called(ctx, repo, sha)

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

	return args.Get(0).(*Commit), args.Error(1)
}
