package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errExampleTest = errors.New("example test error")

func TestExampleClient_MethodReturningString(t *testing.T) {
	t.Parallel()

	t.Run("returns string value", func(t *testing.T) {
		t.Parallel()

		client := &ExampleClient{}
		client.On("MethodReturningString", mock.Anything, "test-id").Return("result", nil)

		result, err := client.MethodReturningString(context.Background(), "test-id")
		require.NoError(t, err)
		assert.Equal(t, "result", result)
		client.AssertExpectations(t)
	})

	t.Run("returns error with nil value", func(t *testing.T) {
		t.Parallel()

		client := &ExampleClient{}
		client.On("MethodReturningString", mock.Anything, "bad-id").Return(nil, errExampleTest)

		result, err := client.MethodReturningString(context.Background(), "bad-id")
		require.ErrorIs(t, err, errExampleTest)
		assert.Empty(t, result)
		client.AssertExpectations(t)
	})
}

func TestExampleClient_MethodReturningBool(t *testing.T) {
	t.Parallel()

	t.Run("returns true", func(t *testing.T) {
		t.Parallel()

		client := &ExampleClient{}
		client.On("MethodReturningBool", mock.Anything, true).Return(true, nil)

		result, err := client.MethodReturningBool(context.Background(), true)
		require.NoError(t, err)
		assert.True(t, result)
		client.AssertExpectations(t)
	})

	t.Run("returns error with nil value", func(t *testing.T) {
		t.Parallel()

		client := &ExampleClient{}
		client.On("MethodReturningBool", mock.Anything, false).Return(nil, errExampleTest)

		result, err := client.MethodReturningBool(context.Background(), false)
		require.ErrorIs(t, err, errExampleTest)
		assert.False(t, result)
		client.AssertExpectations(t)
	})
}

func TestExampleClient_MethodReturningError(t *testing.T) {
	t.Parallel()

	t.Run("returns nil error", func(t *testing.T) {
		t.Parallel()

		client := &ExampleClient{}
		client.On("MethodReturningError", mock.Anything).Return(nil)

		err := client.MethodReturningError(context.Background())
		require.NoError(t, err)
		client.AssertExpectations(t)
	})

	t.Run("returns error", func(t *testing.T) {
		t.Parallel()

		client := &ExampleClient{}
		client.On("MethodReturningError", mock.Anything).Return(errExampleTest)

		err := client.MethodReturningError(context.Background())
		require.ErrorIs(t, err, errExampleTest)
		client.AssertExpectations(t)
	})
}

func TestNewExampleClientRefactored(t *testing.T) {
	t.Parallel()

	client := NewExampleClientRefactored()
	require.NotNil(t, client)
	assert.NotNil(t, client.ErrorHandler)
	assert.NotNil(t, client.BoolHandler)
	assert.NotNil(t, client.StringHandler)
}

func TestNewComplexExampleClient(t *testing.T) {
	t.Parallel()

	client := NewComplexExampleClient()
	require.NotNil(t, client)
	assert.NotNil(t, client.sliceHandler)
	assert.NotNil(t, client.mapHandler)
}

func TestCallHandler_HandleCall(t *testing.T) {
	t.Parallel()

	t.Run("returns typed value", func(t *testing.T) {
		t.Parallel()

		m := &mock.Mock{}
		handler := NewCallHandler[string](m)
		m.On("HandleCall", "arg1").Return("value", nil)

		result, err := handler.HandleCall("arg1")
		require.NoError(t, err)
		assert.Equal(t, "value", result)
		m.AssertExpectations(t)
	})

	t.Run("returns error", func(t *testing.T) {
		t.Parallel()

		m := &mock.Mock{}
		handler := NewCallHandler[string](m)
		m.On("HandleCall", "bad").Return(nil, errExampleTest)

		result, err := handler.HandleCall("bad")
		require.ErrorIs(t, err, errExampleTest)
		assert.Empty(t, result)
		m.AssertExpectations(t)
	})
}

func TestCallHandler_HandleCallWithIndex(t *testing.T) {
	t.Parallel()

	t.Run("returns value at index", func(t *testing.T) {
		t.Parallel()

		m := &mock.Mock{}
		handler := NewCallHandler[int](m)
		// index param is not passed to mock.Called, only variadic args are
		m.On("HandleCallWithIndex", "arg1").Return(42, nil)

		result, err := handler.HandleCallWithIndex(0, "arg1")
		require.NoError(t, err)
		assert.Equal(t, 42, result)
		m.AssertExpectations(t)
	})

	t.Run("returns error", func(t *testing.T) {
		t.Parallel()

		m := &mock.Mock{}
		handler := NewCallHandler[int](m)
		m.On("HandleCallWithIndex", "bad").Return(nil, errExampleTest)

		result, err := handler.HandleCallWithIndex(0, "bad")
		require.ErrorIs(t, err, errExampleTest)
		assert.Equal(t, 0, result)
		m.AssertExpectations(t)
	})
}

func TestSliceHandler_HandleSlice(t *testing.T) {
	t.Parallel()

	t.Run("returns slice", func(t *testing.T) {
		t.Parallel()

		m := &mock.Mock{}
		handler := NewSliceHandler[int](m)
		m.On("HandleSlice", "arg1").Return([]int{1, 2, 3}, nil)

		result, err := handler.HandleSlice("arg1")
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, result)
		m.AssertExpectations(t)
	})

	t.Run("returns error", func(t *testing.T) {
		t.Parallel()

		m := &mock.Mock{}
		handler := NewSliceHandler[int](m)
		m.On("HandleSlice", "bad").Return(nil, errExampleTest)

		result, err := handler.HandleSlice("bad")
		require.ErrorIs(t, err, errExampleTest)
		assert.Nil(t, result)
		m.AssertExpectations(t)
	})
}

func TestMapHandler_HandleMap(t *testing.T) {
	t.Parallel()

	t.Run("returns map", func(t *testing.T) {
		t.Parallel()

		m := &mock.Mock{}
		handler := NewMapHandler[string, bool](m)
		expected := map[string]bool{"a": true, "b": false}
		m.On("HandleMap", "arg1").Return(expected, nil)

		result, err := handler.HandleMap("arg1")
		require.NoError(t, err)
		assert.Equal(t, expected, result)
		m.AssertExpectations(t)
	})

	t.Run("returns error", func(t *testing.T) {
		t.Parallel()

		m := &mock.Mock{}
		handler := NewMapHandler[string, bool](m)
		m.On("HandleMap", "bad").Return(nil, errExampleTest)

		result, err := handler.HandleMap("bad")
		require.ErrorIs(t, err, errExampleTest)
		assert.Nil(t, result)
		m.AssertExpectations(t)
	})
}
