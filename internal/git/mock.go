package git

import (
	"context"
	"fmt"

	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	mock.Mock
}

// Clone mock implementation
func (m *MockClient) Clone(ctx context.Context, url, path string) error {
	args := m.Called(ctx, url, path)

	// Check if we have enough arguments to avoid panic
	if len(args) < 1 {
		// Return an error instead of nil to avoid nil pointer dereference
		return fmt.Errorf("mock not properly configured: expected 1 return value, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	// Handle nil return value (which is a valid error value)
	if args.Get(0) == nil {
		return nil
	}

	// Try to cast to error, fallback to generic error if not possible
	if err, ok := args.Get(0).(error); ok {
		return err
	}

	// If not an error type, return a generic error
	return fmt.Errorf("mock returned non-error type: %T", args.Get(0)) //nolint:err113 // defensive error for test mock
}

// Checkout mock implementation
func (m *MockClient) Checkout(ctx context.Context, repoPath, branch string) error {
	args := m.Called(ctx, repoPath, branch)

	// Check if we have enough arguments to avoid panic
	if len(args) < 1 {
		// Return an error instead of nil to avoid nil pointer dereference
		return fmt.Errorf("mock not properly configured: expected 1 return value, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	// Handle nil return value (which is a valid error value)
	if args.Get(0) == nil {
		return nil
	}

	// Try to cast to error, fallback to generic error if not possible
	if err, ok := args.Get(0).(error); ok {
		return err
	}

	// If not an error type, return a generic error
	return fmt.Errorf("mock returned non-error type: %T", args.Get(0)) //nolint:err113 // defensive error for test mock
}

// CreateBranch mock implementation
func (m *MockClient) CreateBranch(ctx context.Context, repoPath, branch string) error {
	args := m.Called(ctx, repoPath, branch)

	// Check if we have enough arguments to avoid panic
	if len(args) < 1 {
		// Return an error instead of nil to avoid nil pointer dereference
		return fmt.Errorf("mock not properly configured: expected 1 return value, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	// Handle nil return value (which is a valid error value)
	if args.Get(0) == nil {
		return nil
	}

	// Try to cast to error, fallback to generic error if not possible
	if err, ok := args.Get(0).(error); ok {
		return err
	}

	// If not an error type, return a generic error
	return fmt.Errorf("mock returned non-error type: %T", args.Get(0)) //nolint:err113 // defensive error for test mock
}

// Add mock implementation
func (m *MockClient) Add(ctx context.Context, repoPath string, paths ...string) error {
	args := m.Called(ctx, repoPath, paths)

	// Check if we have enough arguments to avoid panic
	if len(args) < 1 {
		// Return an error instead of nil to avoid nil pointer dereference
		return fmt.Errorf("mock not properly configured: expected 1 return value, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	// Handle nil return value (which is a valid error value)
	if args.Get(0) == nil {
		return nil
	}

	// Try to cast to error, fallback to generic error if not possible
	if err, ok := args.Get(0).(error); ok {
		return err
	}

	// If not an error type, return a generic error
	return fmt.Errorf("mock returned non-error type: %T", args.Get(0)) //nolint:err113 // defensive error for test mock
}

// Commit mock implementation
func (m *MockClient) Commit(ctx context.Context, repoPath, message string) error {
	args := m.Called(ctx, repoPath, message)

	// Check if we have enough arguments to avoid panic
	if len(args) < 1 {
		// Return an error instead of nil to avoid nil pointer dereference
		return fmt.Errorf("mock not properly configured: expected 1 return value, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	// Handle nil return value (which is a valid error value)
	if args.Get(0) == nil {
		return nil
	}

	// Try to cast to error, fallback to generic error if not possible
	if err, ok := args.Get(0).(error); ok {
		return err
	}

	// If not an error type, return a generic error
	return fmt.Errorf("mock returned non-error type: %T", args.Get(0)) //nolint:err113 // defensive error for test mock
}

// Push mock implementation
func (m *MockClient) Push(ctx context.Context, repoPath, remote, branch string, force bool) error {
	args := m.Called(ctx, repoPath, remote, branch, force)

	// Check if we have enough arguments to avoid panic
	if len(args) < 1 {
		// Return an error instead of nil to avoid nil pointer dereference
		return fmt.Errorf("mock not properly configured: expected 1 return value, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	// Handle nil return value (which is a valid error value)
	if args.Get(0) == nil {
		return nil
	}

	// Try to cast to error, fallback to generic error if not possible
	if err, ok := args.Get(0).(error); ok {
		return err
	}

	// If not an error type, return a generic error
	return fmt.Errorf("mock returned non-error type: %T", args.Get(0)) //nolint:err113 // defensive error for test mock
}

// Diff mock implementation
func (m *MockClient) Diff(ctx context.Context, repoPath string, staged bool) (string, error) {
	args := m.Called(ctx, repoPath, staged)

	// Check if we have enough arguments to avoid panic
	if len(args) < 2 {
		// Fallback for incorrectly configured mocks
		if len(args) == 1 {
			if err, ok := args.Get(0).(error); ok {
				return "", err
			}
		}
		// Return an error instead of nil,nil to avoid nil pointer dereference
		return "", fmt.Errorf("mock not properly configured: expected 2 return values, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	return args.String(0), args.Error(1)
}

// GetCurrentBranch mock implementation
func (m *MockClient) GetCurrentBranch(ctx context.Context, repoPath string) (string, error) {
	args := m.Called(ctx, repoPath)

	// Check if we have enough arguments to avoid panic
	if len(args) < 2 {
		// Fallback for incorrectly configured mocks
		if len(args) == 1 {
			if err, ok := args.Get(0).(error); ok {
				return "", err
			}
		}
		// Return an error instead of nil,nil to avoid nil pointer dereference
		return "", fmt.Errorf("mock not properly configured: expected 2 return values, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	return args.String(0), args.Error(1)
}

// GetRemoteURL mock implementation
func (m *MockClient) GetRemoteURL(ctx context.Context, repoPath, remote string) (string, error) {
	args := m.Called(ctx, repoPath, remote)

	// Check if we have enough arguments to avoid panic
	if len(args) < 2 {
		// Fallback for incorrectly configured mocks
		if len(args) == 1 {
			if err, ok := args.Get(0).(error); ok {
				return "", err
			}
		}
		// Return an error instead of nil,nil to avoid nil pointer dereference
		return "", fmt.Errorf("mock not properly configured: expected 2 return values, got %d", len(args)) //nolint:err113 // defensive error for test mock
	}

	return args.String(0), args.Error(1)
}
