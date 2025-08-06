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

// NewCallHandler creates a new CallHandler for the given mock
func NewCallHandler[T any](m *mock.Mock) *CallHandler[T] {
	return &CallHandler[T]{mock: m}
}

// HandleCall handles a mock call that returns (T, error)
func (h *CallHandler[T]) HandleCall(_ string, args ...interface{}) (T, error) {
	callArgs := h.mock.Called(args...)
	return testutil.ExtractResult[T](callArgs, 0)
}

// HandleCallWithIndex handles a mock call where the result is at a specific index
func (h *CallHandler[T]) HandleCallWithIndex(_ string, index int, args ...interface{}) (T, error) {
	callArgs := h.mock.Called(args...)
	return testutil.ExtractResult[T](callArgs, index)
}

// ErrorHandler provides consistent error handling for mock methods
type ErrorHandler struct {
	mock *mock.Mock
}

// NewErrorHandler creates a new ErrorHandler
func NewErrorHandler(m *mock.Mock) *ErrorHandler {
	return &ErrorHandler{mock: m}
}

// HandleError handles a mock call that returns only error
func (h *ErrorHandler) HandleError(_ string, args ...interface{}) error {
	callArgs := h.mock.Called(args...)
	return testutil.ExtractError(callArgs)
}

// BoolHandler provides type-safe handling for boolean returns
type BoolHandler struct {
	mock *mock.Mock
}

// NewBoolHandler creates a new BoolHandler
func NewBoolHandler(m *mock.Mock) *BoolHandler {
	return &BoolHandler{mock: m}
}

// HandleBool handles a mock call that returns (bool, error)
func (h *BoolHandler) HandleBool(_ string, args ...interface{}) (bool, error) {
	callArgs := h.mock.Called(args...)
	return testutil.HandleTwoValueReturn[bool](callArgs)
}

// StringHandler provides type-safe handling for string returns
type StringHandler struct {
	mock *mock.Mock
}

// NewStringHandler creates a new StringHandler
func NewStringHandler(m *mock.Mock) *StringHandler {
	return &StringHandler{mock: m}
}

// HandleString handles a mock call that returns (string, error)
func (h *StringHandler) HandleString(_ string, args ...interface{}) (string, error) {
	callArgs := h.mock.Called(args...)
	return testutil.HandleTwoValueReturn[string](callArgs)
}

// SliceHandler provides type-safe handling for slice returns
type SliceHandler[T any] struct {
	mock *mock.Mock
}

// NewSliceHandler creates a new SliceHandler
func NewSliceHandler[T any](m *mock.Mock) *SliceHandler[T] {
	return &SliceHandler[T]{mock: m}
}

// HandleSlice handles a mock call that returns ([]T, error)
func (h *SliceHandler[T]) HandleSlice(_ string, args ...interface{}) ([]T, error) {
	callArgs := h.mock.Called(args...)
	return testutil.HandleTwoValueReturn[[]T](callArgs)
}

// MapHandler provides type-safe handling for map returns
type MapHandler[K comparable, V any] struct {
	mock *mock.Mock
}

// NewMapHandler creates a new MapHandler
func NewMapHandler[K comparable, V any](m *mock.Mock) *MapHandler[K, V] {
	return &MapHandler[K, V]{mock: m}
}

// HandleMap handles a mock call that returns (map[K]V, error)
func (h *MapHandler[K, V]) HandleMap(_ string, args ...interface{}) (map[K]V, error) {
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

// Error is a convenience method for error-only returns
func (m *MockBase) Error(args ...interface{}) error {
	return m.ErrorHandler.HandleError("", args...)
}

// Bool is a convenience method for bool returns
func (m *MockBase) Bool(args ...interface{}) (bool, error) {
	return m.BoolHandler.HandleBool("", args...)
}

// String is a convenience method for string returns
func (m *MockBase) String(args ...interface{}) (string, error) {
	return m.StringHandler.HandleString("", args...)
}
