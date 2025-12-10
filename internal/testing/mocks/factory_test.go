package mocks

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test sentinel errors for err113 compliance.
var (
	errBase   = errors.New("base error")
	errBool   = errors.New("bool error")
	errString = errors.New("string error")
)

func TestNewCallHandler(t *testing.T) {
	t.Parallel()

	m := &mock.Mock{}
	handler := NewCallHandler[string](m)

	require.NotNil(t, handler)
	assert.NotNil(t, handler.mock)
}

func TestNewCallHandler_GenericTypes(t *testing.T) {
	t.Parallel()

	t.Run("StringHandler", func(t *testing.T) {
		t.Parallel()
		m := &mock.Mock{}
		handler := NewCallHandler[string](m)
		require.NotNil(t, handler)
	})

	t.Run("IntHandler", func(t *testing.T) {
		t.Parallel()
		m := &mock.Mock{}
		handler := NewCallHandler[int](m)
		require.NotNil(t, handler)
	})

	t.Run("BoolHandler", func(t *testing.T) {
		t.Parallel()
		m := &mock.Mock{}
		handler := NewCallHandler[bool](m)
		require.NotNil(t, handler)
	})

	t.Run("SliceHandler", func(t *testing.T) {
		t.Parallel()
		m := &mock.Mock{}
		handler := NewCallHandler[[]string](m)
		require.NotNil(t, handler)
	})

	t.Run("StructHandler", func(t *testing.T) {
		t.Parallel()
		type customStruct struct {
			Name  string
			Value int
		}
		m := &mock.Mock{}
		handler := NewCallHandler[customStruct](m)
		require.NotNil(t, handler)
	})
}

func TestNewErrorHandler(t *testing.T) {
	t.Parallel()

	m := &mock.Mock{}
	handler := NewErrorHandler(m)

	require.NotNil(t, handler)
	assert.NotNil(t, handler.mock)
}

func TestNewBoolHandler(t *testing.T) {
	t.Parallel()

	m := &mock.Mock{}
	handler := NewBoolHandler(m)

	require.NotNil(t, handler)
	assert.NotNil(t, handler.mock)
}

func TestNewStringHandler(t *testing.T) {
	t.Parallel()

	m := &mock.Mock{}
	handler := NewStringHandler(m)

	require.NotNil(t, handler)
	assert.NotNil(t, handler.mock)
}

func TestNewSliceHandler(t *testing.T) {
	t.Parallel()

	m := &mock.Mock{}
	handler := NewSliceHandler[string](m)

	require.NotNil(t, handler)
	assert.NotNil(t, handler.mock)
}

func TestNewSliceHandler_GenericTypes(t *testing.T) {
	t.Parallel()

	t.Run("StringSlice", func(t *testing.T) {
		t.Parallel()
		m := &mock.Mock{}
		handler := NewSliceHandler[string](m)
		require.NotNil(t, handler)
	})

	t.Run("IntSlice", func(t *testing.T) {
		t.Parallel()
		m := &mock.Mock{}
		handler := NewSliceHandler[int](m)
		require.NotNil(t, handler)
	})

	t.Run("StructSlice", func(t *testing.T) {
		t.Parallel()
		type item struct {
			ID   int
			Name string
		}
		m := &mock.Mock{}
		handler := NewSliceHandler[item](m)
		require.NotNil(t, handler)
	})
}

func TestNewMapHandler(t *testing.T) {
	t.Parallel()

	m := &mock.Mock{}
	handler := NewMapHandler[string, int](m)

	require.NotNil(t, handler)
	assert.NotNil(t, handler.mock)
}

func TestNewMapHandler_GenericTypes(t *testing.T) {
	t.Parallel()

	t.Run("StringIntMap", func(t *testing.T) {
		t.Parallel()
		m := &mock.Mock{}
		handler := NewMapHandler[string, int](m)
		require.NotNil(t, handler)
	})

	t.Run("IntStringMap", func(t *testing.T) {
		t.Parallel()
		m := &mock.Mock{}
		handler := NewMapHandler[int, string](m)
		require.NotNil(t, handler)
	})

	t.Run("StringSliceMap", func(t *testing.T) {
		t.Parallel()
		m := &mock.Mock{}
		handler := NewMapHandler[string, []string](m)
		require.NotNil(t, handler)
	})
}

func TestNewMockBase(t *testing.T) {
	t.Parallel()

	base := NewMockBase()

	require.NotNil(t, base)
	assert.NotNil(t, base.ErrorHandler)
	assert.NotNil(t, base.BoolHandler)
	assert.NotNil(t, base.StringHandler)
}

func TestMockBase_HandlersAreInitialized(t *testing.T) {
	t.Parallel()

	base := NewMockBase()

	// Verify all handlers are properly connected to the mock
	assert.Equal(t, &base.Mock, base.ErrorHandler.mock)
	assert.Equal(t, &base.Mock, base.BoolHandler.mock)
	assert.Equal(t, &base.Mock, base.StringHandler.mock)
}

func TestMockBase_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		returnError error
		expectError bool
	}{
		{
			name:        "NoError",
			returnError: nil,
			expectError: false,
		},
		{
			name:        "WithError",
			returnError: errBase,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base := NewMockBase()
			// MockBase.Error calls ErrorHandler.HandleError which uses mock.Called
			// The method name from call stack is "HandleError"
			base.On("HandleError", "action", "param").Return(tt.returnError)

			err := base.Error("action", "param")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			base.AssertExpectations(t)
		})
	}
}

func TestMockBase_Bool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		returnBool    bool
		returnError   error
		expectedValue bool
		expectError   bool
	}{
		{
			name:          "TrueNoError",
			returnBool:    true,
			returnError:   nil,
			expectedValue: true,
			expectError:   false,
		},
		{
			name:          "FalseNoError",
			returnBool:    false,
			returnError:   nil,
			expectedValue: false,
			expectError:   false,
		},
		{
			name:          "WithError",
			returnBool:    false,
			returnError:   errBool,
			expectedValue: false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base := NewMockBase()
			// MockBase.Bool calls BoolHandler.HandleBool which uses mock.Called
			// The method name from call stack is "HandleBool"
			base.On("HandleBool", "check", "value").Return(tt.returnBool, tt.returnError)

			result, err := base.Bool("check", "value")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedValue, result)
			base.AssertExpectations(t)
		})
	}
}

func TestMockBase_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		returnString  string
		returnError   error
		expectedValue string
		expectError   bool
	}{
		{
			name:          "NonEmptyString",
			returnString:  "result",
			returnError:   nil,
			expectedValue: "result",
			expectError:   false,
		},
		{
			name:          "EmptyString",
			returnString:  "",
			returnError:   nil,
			expectedValue: "",
			expectError:   false,
		},
		{
			name:          "WithError",
			returnString:  "",
			returnError:   errString,
			expectedValue: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base := NewMockBase()
			// MockBase.String calls StringHandler.HandleString which uses mock.Called
			// The method name from call stack is "HandleString"
			base.On("HandleString", "fetch", "key").Return(tt.returnString, tt.returnError)

			result, err := base.String("fetch", "key")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedValue, result)
			base.AssertExpectations(t)
		})
	}
}

func TestMockBase_Error_MultipleArgs(t *testing.T) {
	t.Parallel()

	base := NewMockBase()
	// MockBase.Error calls ErrorHandler.HandleError which uses mock.Called
	base.On("HandleError", "arg1", "arg2", "arg3").Return(nil)

	err := base.Error("arg1", "arg2", "arg3")

	require.NoError(t, err)
	base.AssertExpectations(t)
}

func TestMockBase_Bool_MultipleArgs(t *testing.T) {
	t.Parallel()

	base := NewMockBase()
	// MockBase.Bool calls BoolHandler.HandleBool which uses mock.Called
	base.On("HandleBool", 1, 2, 3).Return(true, nil)

	result, err := base.Bool(1, 2, 3)

	require.NoError(t, err)
	assert.True(t, result)
	base.AssertExpectations(t)
}

func TestMockBase_String_MultipleArgs(t *testing.T) {
	t.Parallel()

	base := NewMockBase()
	// MockBase.String calls StringHandler.HandleString which uses mock.Called
	base.On("HandleString", "a", "b", "c").Return("combined", nil)

	result, err := base.String("a", "b", "c")

	require.NoError(t, err)
	assert.Equal(t, "combined", result)
	base.AssertExpectations(t)
}

// Benchmarks
func BenchmarkNewCallHandler(b *testing.B) {
	m := &mock.Mock{}
	for i := 0; i < b.N; i++ {
		_ = NewCallHandler[string](m)
	}
}

func BenchmarkNewErrorHandler(b *testing.B) {
	m := &mock.Mock{}
	for i := 0; i < b.N; i++ {
		_ = NewErrorHandler(m)
	}
}

func BenchmarkNewBoolHandler(b *testing.B) {
	m := &mock.Mock{}
	for i := 0; i < b.N; i++ {
		_ = NewBoolHandler(m)
	}
}

func BenchmarkNewStringHandler(b *testing.B) {
	m := &mock.Mock{}
	for i := 0; i < b.N; i++ {
		_ = NewStringHandler(m)
	}
}

func BenchmarkNewSliceHandler(b *testing.B) {
	m := &mock.Mock{}
	for i := 0; i < b.N; i++ {
		_ = NewSliceHandler[string](m)
	}
}

func BenchmarkNewMapHandler(b *testing.B) {
	m := &mock.Mock{}
	for i := 0; i < b.N; i++ {
		_ = NewMapHandler[string, int](m)
	}
}

func BenchmarkNewMockBase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewMockBase()
	}
}

func BenchmarkMockBase_Error(b *testing.B) {
	base := NewMockBase()
	base.On("HandleError", "action").Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = base.Error("action")
	}
}

func BenchmarkMockBase_Bool(b *testing.B) {
	base := NewMockBase()
	base.On("HandleBool", "check").Return(true, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = base.Bool("check")
	}
}

func BenchmarkMockBase_String(b *testing.B) {
	base := NewMockBase()
	base.On("HandleString", "fetch").Return("result", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = base.String("fetch")
	}
}
