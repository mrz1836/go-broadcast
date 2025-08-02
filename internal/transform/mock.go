package transform

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockTransformer is a mock implementation of the Transformer interface
type MockTransformer struct {
	mock.Mock
}

// NewMockTransformer creates a new MockTransformer (backward compatibility)
func NewMockTransformer() *MockTransformer {
	return &MockTransformer{}
}

// Name mock implementation
func (m *MockTransformer) Name() string {
	args := m.Called()
	return args.String(0)
}

// Transform mock implementation
func (m *MockTransformer) Transform(content []byte, ctx Context) ([]byte, error) {
	args := m.Called(content, ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// MockChain is a mock implementation of the Chain interface
type MockChain struct {
	mock.Mock
}

// NewMockChain creates a new MockChain (backward compatibility)
func NewMockChain() *MockChain {
	return &MockChain{}
}

// Add mock implementation
func (m *MockChain) Add(transformer Transformer) Chain {
	args := m.Called(transformer)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(Chain)
}

// Transform mock implementation
func (m *MockChain) Transform(ctx context.Context, content []byte, transformCtx Context) ([]byte, error) {
	args := m.Called(ctx, content, transformCtx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// Transformers mock implementation
func (m *MockChain) Transformers() []Transformer {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]Transformer)
}
