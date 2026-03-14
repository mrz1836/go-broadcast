// Package mocks provides factory functions and utilities for creating mock objects
package mocks

import (
	"github.com/stretchr/testify/mock"

	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// CallHandler provides a type-safe way to handle mock calls with consistent error handling
type CallHandler[T any] struct {
	mock *mock.Mock
}

// NewCallHandler creates a new CallHandler for the given mock.
// Panics if m is nil to fail fast rather than defer to runtime.
func NewCallHandler[T any](m *mock.Mock) *CallHandler[T] {
	if m == nil {
		panic("mocks: nil mock.Mock passed to NewCallHandler")
	}
	return &CallHandler[T]{mock: m}
}

// HandleCall handles a mock call that returns (T, error)
func (h *CallHandler[T]) HandleCall(args ...interface{}) (T, error) {
	callArgs := h.mock.Called(args...)
	return testutil.ExtractResult[T](callArgs, 0)
}

// HandleCallWithIndex handles a mock call where the result is at a specific index
func (h *CallHandler[T]) HandleCallWithIndex(index int, args ...interface{}) (T, error) {
	callArgs := h.mock.Called(args...)
	return testutil.ExtractResult[T](callArgs, index)
}

// ErrorHandler provides consistent error handling for mock methods
type ErrorHandler struct {
	mock *mock.Mock
}

// NewErrorHandler creates a new ErrorHandler.
// Panics if m is nil to fail fast rather than defer to runtime.
func NewErrorHandler(m *mock.Mock) *ErrorHandler {
	if m == nil {
		panic("mocks: nil mock.Mock passed to NewErrorHandler")
	}
	return &ErrorHandler{mock: m}
}

// HandleError handles a mock call that returns only error
func (h *ErrorHandler) HandleError(args ...interface{}) error {
	callArgs := h.mock.Called(args...)
	return testutil.ExtractError(callArgs)
}

// BoolHandler provides type-safe handling for boolean returns
type BoolHandler struct {
	mock *mock.Mock
}

// NewBoolHandler creates a new BoolHandler.
// Panics if m is nil to fail fast rather than defer to runtime.
func NewBoolHandler(m *mock.Mock) *BoolHandler {
	if m == nil {
		panic("mocks: nil mock.Mock passed to NewBoolHandler")
	}
	return &BoolHandler{mock: m}
}

// HandleBool handles a mock call that returns (bool, error)
func (h *BoolHandler) HandleBool(args ...interface{}) (bool, error) {
	callArgs := h.mock.Called(args...)
	return testutil.HandleTwoValueReturn[bool](callArgs)
}

// StringHandler provides type-safe handling for string returns
type StringHandler struct {
	mock *mock.Mock
}

// NewStringHandler creates a new StringHandler.
// Panics if m is nil to fail fast rather than defer to runtime.
func NewStringHandler(m *mock.Mock) *StringHandler {
	if m == nil {
		panic("mocks: nil mock.Mock passed to NewStringHandler")
	}
	return &StringHandler{mock: m}
}

// HandleString handles a mock call that returns (string, error)
func (h *StringHandler) HandleString(args ...interface{}) (string, error) {
	callArgs := h.mock.Called(args...)
	return testutil.HandleTwoValueReturn[string](callArgs)
}

// SliceHandler provides type-safe handling for slice returns
type SliceHandler[T any] struct {
	mock *mock.Mock
}

// NewSliceHandler creates a new SliceHandler.
// Panics if m is nil to fail fast rather than defer to runtime.
func NewSliceHandler[T any](m *mock.Mock) *SliceHandler[T] {
	if m == nil {
		panic("mocks: nil mock.Mock passed to NewSliceHandler")
	}
	return &SliceHandler[T]{mock: m}
}

// HandleSlice handles a mock call that returns ([]T, error)
func (h *SliceHandler[T]) HandleSlice(args ...interface{}) ([]T, error) {
	callArgs := h.mock.Called(args...)
	return testutil.HandleTwoValueReturn[[]T](callArgs)
}

// MapHandler provides type-safe handling for map returns
type MapHandler[K comparable, V any] struct {
	mock *mock.Mock
}

// NewMapHandler creates a new MapHandler.
// Panics if m is nil to fail fast rather than defer to runtime.
func NewMapHandler[K comparable, V any](m *mock.Mock) *MapHandler[K, V] {
	if m == nil {
		panic("mocks: nil mock.Mock passed to NewMapHandler")
	}
	return &MapHandler[K, V]{mock: m}
}

// HandleMap handles a mock call that returns (map[K]V, error)
func (h *MapHandler[K, V]) HandleMap(args ...interface{}) (map[K]V, error) {
	callArgs := h.mock.Called(args...)
	return testutil.HandleTwoValueReturn[map[K]V](callArgs)
}

// MockBase provides common functionality for all mocks
type MockBase struct {
	mock.Mock

	ErrorHandler  *ErrorHandler
	BoolHandler   *BoolHandler
	StringHandler *StringHandler
}

// NewMockBase creates a new MockBase
func NewMockBase() *MockBase {
	base := &MockBase{}
	base.ErrorHandler = NewErrorHandler(&base.Mock)
	base.BoolHandler = NewBoolHandler(&base.Mock)
	base.StringHandler = NewStringHandler(&base.Mock)
	return base
}

// Error is a convenience method for error-only returns.
// Panics if MockBase was not initialized via NewMockBase().
func (m *MockBase) Error(args ...interface{}) error {
	if m.ErrorHandler == nil {
		panic("mocks: MockBase not initialized - use NewMockBase()")
	}
	return m.ErrorHandler.HandleError(args...)
}

// Bool is a convenience method for bool returns.
// Panics if MockBase was not initialized via NewMockBase().
func (m *MockBase) Bool(args ...interface{}) (bool, error) {
	if m.BoolHandler == nil {
		panic("mocks: MockBase not initialized - use NewMockBase()")
	}
	return m.BoolHandler.HandleBool(args...)
}

// String is a convenience method for string returns.
// Panics if MockBase was not initialized via NewMockBase().
func (m *MockBase) String(args ...interface{}) (string, error) {
	if m.StringHandler == nil {
		panic("mocks: MockBase not initialized - use NewMockBase()")
	}
	return m.StringHandler.HandleString(args...)
}
