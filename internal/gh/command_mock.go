package gh

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockCommandRunner is a mock implementation of CommandRunner
type MockCommandRunner struct {
	mock.Mock
}

// Run mocks command execution
func (m *MockCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	called := m.Called(ctx, name, args)
	if called.Get(0) == nil {
		return nil, called.Error(1)
	}
	return called.Get(0).([]byte), called.Error(1)
}

// RunWithInput mocks command execution with input
func (m *MockCommandRunner) RunWithInput(ctx context.Context, input []byte, name string, args ...string) ([]byte, error) {
	called := m.Called(ctx, input, name, args)
	if called.Get(0) == nil {
		return nil, called.Error(1)
	}
	return called.Get(0).([]byte), called.Error(1)
}

