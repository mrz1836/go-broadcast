// Package mocks - Example showing code reduction with mock factory
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// ExampleClient shows original mock implementation
type ExampleClient struct {
	mock.Mock
}

// MethodReturningString - Original implementation (6 lines)
func (m *ExampleClient) MethodReturningString(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	return args.Get(0).(string), args.Error(1)
}

// MethodReturningBool - Original implementation (6 lines)
func (m *ExampleClient) MethodReturningBool(ctx context.Context, flag bool) (bool, error) {
	args := m.Called(ctx, flag)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}
	return args.Get(0).(bool), args.Error(1)
}

// MethodReturningError - Original implementation (3 lines)
func (m *ExampleClient) MethodReturningError(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// ExampleClientRefactored shows refactored mock using factory
type ExampleClientRefactored struct {
	MockBase
}

// NewExampleClientRefactored creates a new refactored mock
func NewExampleClientRefactored() *ExampleClientRefactored {
	return &ExampleClientRefactored{
		MockBase: *NewMockBase(),
	}
}

// MethodReturningString - Refactored implementation (2 lines)
func (m *ExampleClientRefactored) MethodReturningString(ctx context.Context, id string) (string, error) {
	return m.String(ctx, id)
}

// MethodReturningBool - Refactored implementation (2 lines)
func (m *ExampleClientRefactored) MethodReturningBool(ctx context.Context, flag bool) (bool, error) {
	return m.Bool(ctx, flag)
}

// MethodReturningError - Refactored implementation (2 lines)
func (m *ExampleClientRefactored) MethodReturningError(ctx context.Context) error {
	return m.Error(ctx)
}

// ComplexExampleClient shows more complex types
type ComplexExampleClient struct {
	mock.Mock

	sliceHandler *SliceHandler[string]
	mapHandler   *MapHandler[string, int]
}

// NewComplexExampleClient creates a new complex example client
func NewComplexExampleClient() *ComplexExampleClient {
	client := &ComplexExampleClient{}
	client.sliceHandler = NewSliceHandler[string](&client.Mock)
	client.mapHandler = NewMapHandler[string, int](&client.Mock)
	return client
}

// GetItems - Refactored slice return (2 lines)
func (m *ComplexExampleClient) GetItems(ctx context.Context) ([]string, error) {
	return m.sliceHandler.HandleSlice(ctx)
}

// GetCounts - Refactored map return (2 lines)
func (m *ComplexExampleClient) GetCounts(ctx context.Context) (map[string]int, error) {
	return m.mapHandler.HandleMap(ctx)
}

// Code reduction summary:
// Original mock method: 3-6 lines per method
// Refactored mock method: 2 lines per method
// Reduction: 50-67% per method

// For a mock with 20 methods:
// Original: ~100 lines
// Refactored: ~40 lines
// Total reduction: 60%
