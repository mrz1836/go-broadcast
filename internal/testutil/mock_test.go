package testutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          mock.Arguments
		expectedCount int
		wantErr       bool
		errMsg        string
	}{
		{
			name:          "ValidArgsCount",
			args:          mock.Arguments{"result", nil},
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name:          "TooFewArgs",
			args:          mock.Arguments{"result"},
			expectedCount: 2,
			wantErr:       true,
			errMsg:        "expected 2 return values, got 1",
		},
		{
			name:          "TooManyArgs",
			args:          mock.Arguments{"result", nil, "extra"},
			expectedCount: 2,
			wantErr:       true,
			errMsg:        "expected 2 return values, got 3",
		},
		{
			name:          "EmptyArgsExpectedEmpty",
			args:          mock.Arguments{},
			expectedCount: 0,
			wantErr:       false,
		},
		{
			name:          "EmptyArgsExpectedOne",
			args:          mock.Arguments{},
			expectedCount: 1,
			wantErr:       true,
			errMsg:        "expected 1 return values, got 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArgs(tt.args, tt.expectedCount)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExtractResult(t *testing.T) {
	testErr := errors.New("test error") //nolint:err113 // test-only error

	t.Run("SuccessfulExtraction", func(t *testing.T) {
		args := mock.Arguments{"hello", nil}
		result, err := ExtractResult[string](args, 0)

		require.NoError(t, err)
		assert.Equal(t, "hello", result)
	})

	t.Run("ExtractionWithError", func(t *testing.T) {
		args := mock.Arguments{"hello", testErr}
		result, err := ExtractResult[string](args, 0)

		assert.Equal(t, "hello", result)
		assert.Equal(t, testErr, err)
	})

	t.Run("NilResultWithError", func(t *testing.T) {
		args := mock.Arguments{nil, testErr}
		result, err := ExtractResult[string](args, 0)

		assert.Empty(t, result) // zero value for string
		assert.Equal(t, testErr, err)
	})

	t.Run("NilResultWithoutError", func(t *testing.T) {
		args := mock.Arguments{nil, nil}
		result, err := ExtractResult[string](args, 0)

		assert.Empty(t, result) // zero value for string
		require.NoError(t, err)
	})

	t.Run("WrongType", func(t *testing.T) {
		args := mock.Arguments{42, nil} // int instead of string
		result, err := ExtractResult[string](args, 0)

		assert.Empty(t, result) // zero value for string
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not of expected type")
	})

	t.Run("WrongArgsCount", func(t *testing.T) {
		args := mock.Arguments{"hello"} // only 1 arg instead of 2
		result, err := ExtractResult[string](args, 0)

		assert.Empty(t, result) // zero value for string
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected 2 return values, got 1")
	})

	t.Run("IntType", func(t *testing.T) {
		args := mock.Arguments{42, nil}
		result, err := ExtractResult[int](args, 0)

		require.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("BoolType", func(t *testing.T) {
		args := mock.Arguments{true, nil}
		result, err := ExtractResult[bool](args, 0)

		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("SliceType", func(t *testing.T) {
		expectedSlice := []string{"a", "b", "c"}
		args := mock.Arguments{expectedSlice, nil}
		result, err := ExtractResult[[]string](args, 0)

		require.NoError(t, err)
		assert.Equal(t, expectedSlice, result)
	})

	t.Run("StructType", func(t *testing.T) {
		type testStruct struct {
			Name string
			Age  int
		}
		expected := testStruct{Name: "Alice", Age: 30}
		args := mock.Arguments{expected, nil}
		result, err := ExtractResult[testStruct](args, 0)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})
}

func TestExtractError(t *testing.T) {
	testErr := errors.New("test error") //nolint:err113 // test-only error

	tests := []struct {
		name    string
		args    mock.Arguments
		wantErr bool
		errMsg  string
	}{
		{
			name:    "NoError",
			args:    mock.Arguments{nil},
			wantErr: false,
		},
		{
			name:    "WithError",
			args:    mock.Arguments{testErr},
			wantErr: true,
			errMsg:  "test error",
		},
		{
			name:    "WrongArgsCount",
			args:    mock.Arguments{},
			wantErr: true,
			errMsg:  "expected 1 return values, got 0",
		},
		{
			name:    "TooManyArgs",
			args:    mock.Arguments{nil, "extra"},
			wantErr: true,
			errMsg:  "expected 1 return values, got 2",
		},
		{
			name:    "NonErrorType",
			args:    mock.Arguments{"not an error"},
			wantErr: true,
			errMsg:  "mock returned non-error type",
		},
		{
			name:    "IntType",
			args:    mock.Arguments{42},
			wantErr: true,
			errMsg:  "mock returned non-error type: int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExtractError(tt.args)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExtractStringResult(t *testing.T) {
	testErr := errors.New("test error") //nolint:err113 // test-only error

	tests := []struct {
		name         string
		args         mock.Arguments
		expectedStr  string
		expectedErr  error
		wantErr      bool
		errMsgSubstr string
	}{
		{
			name:        "SuccessfulExtraction",
			args:        mock.Arguments{"hello", nil},
			expectedStr: "hello",
			expectedErr: nil,
			wantErr:     false,
		},
		{
			name:        "WithError",
			args:        mock.Arguments{"hello", testErr},
			expectedStr: "hello",
			expectedErr: testErr,
			wantErr:     true,
		},
		{
			name:         "TooFewArgs",
			args:         mock.Arguments{},
			expectedStr:  "",
			wantErr:      true,
			errMsgSubstr: "expected 2 return values, got 0",
		},
		{
			name:         "OneArgError",
			args:         mock.Arguments{testErr},
			expectedStr:  "",
			expectedErr:  testErr,
			wantErr:      true,
			errMsgSubstr: "test error",
		},
		{
			name:         "OneArgNonError",
			args:         mock.Arguments{"hello"},
			expectedStr:  "",
			wantErr:      true,
			errMsgSubstr: "expected 2 return values, got 1",
		},
		{
			name:        "EmptyString",
			args:        mock.Arguments{"", nil},
			expectedStr: "",
			expectedErr: nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractStringResult(tt.args)

			assert.Equal(t, tt.expectedStr, result)

			if tt.wantErr {
				require.Error(t, err)
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				}
				if tt.errMsgSubstr != "" {
					assert.Contains(t, err.Error(), tt.errMsgSubstr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHandleTwoValueReturn(t *testing.T) {
	testErr := errors.New("test error") //nolint:err113 // test-only error

	t.Run("StringType", func(t *testing.T) {
		tests := []struct {
			name         string
			args         mock.Arguments
			expectedStr  string
			expectedErr  error
			wantErr      bool
			errMsgSubstr string
		}{
			{
				name:        "SuccessfulExtraction",
				args:        mock.Arguments{"hello", nil},
				expectedStr: "hello",
				expectedErr: nil,
				wantErr:     false,
			},
			{
				name:        "WithError",
				args:        mock.Arguments{"hello", testErr},
				expectedStr: "hello",
				expectedErr: testErr,
				wantErr:     true,
			},
			{
				name:        "NilResult",
				args:        mock.Arguments{nil, nil},
				expectedStr: "",
				expectedErr: nil,
				wantErr:     false,
			},
			{
				name:         "WrongType",
				args:         mock.Arguments{42, nil},
				expectedStr:  "",
				wantErr:      true,
				errMsgSubstr: "not of expected type",
			},
			{
				name:         "TooFewArgs",
				args:         mock.Arguments{"hello"},
				expectedStr:  "",
				wantErr:      true,
				errMsgSubstr: "expected 2 return values, got 1",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := HandleTwoValueReturn[string](tt.args)

				assert.Equal(t, tt.expectedStr, result)

				if tt.wantErr {
					require.Error(t, err)
					if tt.expectedErr != nil {
						assert.Equal(t, tt.expectedErr, err)
					}
					if tt.errMsgSubstr != "" {
						assert.Contains(t, err.Error(), tt.errMsgSubstr)
					}
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("IntType", func(t *testing.T) {
		args := mock.Arguments{42, nil}
		result, err := HandleTwoValueReturn[int](args)

		require.NoError(t, err)
		assert.Equal(t, 42, result)
	})

	t.Run("BoolType", func(t *testing.T) {
		args := mock.Arguments{true, testErr}
		result, err := HandleTwoValueReturn[bool](args)

		assert.True(t, result)
		assert.Equal(t, testErr, err)
	})

	t.Run("SliceType", func(t *testing.T) {
		expectedSlice := []int{1, 2, 3}
		args := mock.Arguments{expectedSlice, nil}
		result, err := HandleTwoValueReturn[[]int](args)

		require.NoError(t, err)
		assert.Equal(t, expectedSlice, result)
	})

	t.Run("PointerType", func(t *testing.T) {
		value := "test"
		expected := &value
		args := mock.Arguments{expected, nil}
		result, err := HandleTwoValueReturn[*string](args)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
		assert.Equal(t, "test", *result)
	})

	t.Run("StructType", func(t *testing.T) {
		type testStruct struct {
			Name string
			Age  int
		}
		expected := testStruct{Name: "Bob", Age: 25}
		args := mock.Arguments{expected, nil}
		result, err := HandleTwoValueReturn[testStruct](args)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("MapType", func(t *testing.T) {
		expectedMap := map[string]int{"a": 1, "b": 2}
		args := mock.Arguments{expectedMap, nil}
		result, err := HandleTwoValueReturn[map[string]int](args)

		require.NoError(t, err)
		assert.Equal(t, expectedMap, result)
	})

	t.Run("InterfaceType", func(t *testing.T) {
		var expected interface{} = "hello"
		args := mock.Arguments{expected, nil}
		result, err := HandleTwoValueReturn[interface{}](args)

		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("EmptyArgs", func(t *testing.T) {
		args := mock.Arguments{}
		result, err := HandleTwoValueReturn[string](args)

		assert.Empty(t, result) // zero value
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected 2 return values, got 0")
	})

	t.Run("OneArgError", func(t *testing.T) {
		args := mock.Arguments{testErr}
		result, err := HandleTwoValueReturn[string](args)

		assert.Empty(t, result) // zero value
		assert.Equal(t, testErr, err)
	})

	t.Run("OneArgNonError", func(t *testing.T) {
		args := mock.Arguments{"hello"}
		result, err := HandleTwoValueReturn[string](args)

		assert.Empty(t, result) // zero value
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected 2 return values, got 1")
	})
}

// TestMockUtilitiesIntegration tests the utilities working together
func TestMockUtilitiesIntegration(t *testing.T) {
	t.Run("RealWorldScenario", func(t *testing.T) {
		// Simulate a mock that returns (string, error)
		testErr := errors.New("integration test error") //nolint:err113 // test-only error

		// Test successful case
		successArgs := mock.Arguments{"success", nil}

		// Test with HandleTwoValueReturn
		result1, err1 := HandleTwoValueReturn[string](successArgs)
		assert.Equal(t, "success", result1)
		require.NoError(t, err1)

		// Test with ExtractResult
		result2, err2 := ExtractResult[string](successArgs, 0)
		assert.Equal(t, "success", result2)
		require.NoError(t, err2)

		// Test with ExtractStringResult
		result3, err3 := ExtractStringResult(successArgs)
		assert.Equal(t, "success", result3)
		require.NoError(t, err3)

		// Test error case
		errorArgs := mock.Arguments{"failure", testErr}

		result4, err4 := HandleTwoValueReturn[string](errorArgs)
		assert.Equal(t, "failure", result4)
		assert.Equal(t, testErr, err4)

		result5, err5 := ExtractResult[string](errorArgs, 0)
		assert.Equal(t, "failure", result5)
		assert.Equal(t, testErr, err5)

		result6, err6 := ExtractStringResult(errorArgs)
		assert.Equal(t, "failure", result6)
		assert.Equal(t, testErr, err6)
	})

	t.Run("ErrorOnlyScenario", func(t *testing.T) {
		// Simulate a mock that returns only error
		testErr := errors.New("error only test") //nolint:err113 // test-only error

		// Test successful case (no error)
		successArgs := mock.Arguments{nil}
		err1 := ExtractError(successArgs)
		require.NoError(t, err1)

		// Test error case
		errorArgs := mock.Arguments{testErr}
		err2 := ExtractError(errorArgs)
		assert.Equal(t, testErr, err2)
	})
}

// BenchmarkValidateArgs tests the performance of argument validation
func BenchmarkValidateArgs(b *testing.B) {
	args := mock.Arguments{"result", nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateArgs(args, 2)
	}
}

// BenchmarkExtractResult tests the performance of result extraction
func BenchmarkExtractResult(b *testing.B) {
	args := mock.Arguments{"hello", nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractResult[string](args, 0)
	}
}

// BenchmarkHandleTwoValueReturn tests the performance of two-value return handling
func BenchmarkHandleTwoValueReturn(b *testing.B) {
	args := mock.Arguments{"hello", nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HandleTwoValueReturn[string](args)
	}
}
