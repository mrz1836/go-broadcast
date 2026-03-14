package gh

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockCommandRunner_Run(t *testing.T) {
	tests := []struct {
		name           string
		expectedOutput []byte
		expectedError  error
		setupMock      func(*MockCommandRunner)
	}{
		{
			name:           "successful command execution",
			expectedOutput: []byte("command output"),
			expectedError:  nil,
			setupMock: func(m *MockCommandRunner) {
				m.On("Run", context.Background(), "test-command", []string{"arg1", "arg2"}).
					Return([]byte("command output"), nil)
			},
		},
		{
			name:           "command execution with error",
			expectedOutput: nil,
			expectedError:  assert.AnError,
			setupMock: func(m *MockCommandRunner) {
				m.On("Run", context.Background(), "failing-command", []string{"arg1"}).
					Return(nil, assert.AnError)
			},
		},
		{
			name:           "empty command output",
			expectedOutput: []byte{},
			expectedError:  nil,
			setupMock: func(m *MockCommandRunner) {
				m.On("Run", context.Background(), "empty-command", []string(nil)).
					Return([]byte{}, nil)
			},
		},
		{
			name:           "nil output with error",
			expectedOutput: nil,
			expectedError:  assert.AnError,
			setupMock: func(m *MockCommandRunner) {
				m.On("Run", context.Background(), "nil-command", []string{"test"}).
					Return(nil, assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := &MockCommandRunner{}
			tt.setupMock(mockRunner)

			ctx := context.Background()
			var output []byte
			var err error

			switch tt.name {
			case "successful command execution":
				output, err = mockRunner.Run(ctx, "test-command", "arg1", "arg2")
			case "command execution with error":
				output, err = mockRunner.Run(ctx, "failing-command", "arg1")
			case "empty command output":
				output, err = mockRunner.Run(ctx, "empty-command")
			case "nil output with error":
				output, err = mockRunner.Run(ctx, "nil-command", "test")
			}

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedOutput, output)
			mockRunner.AssertExpectations(t)
		})
	}
}

func TestMockCommandRunner_RunWithInput(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		command        string
		args           []string
		expectedOutput []byte
		expectedError  error
		setupMock      func(*MockCommandRunner)
	}{
		{
			name:           "successful command execution with input",
			input:          []byte("test input"),
			command:        "process-input",
			args:           []string{"--format", "json"},
			expectedOutput: []byte("processed output"),
			expectedError:  nil,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunWithInput", context.Background(), []byte("test input"), "process-input", []string{"--format", "json"}).
					Return([]byte("processed output"), nil)
			},
		},
		{
			name:           "command execution with nil input",
			input:          nil,
			command:        "no-input-command",
			args:           []string{"arg1"},
			expectedOutput: []byte("no input result"),
			expectedError:  nil,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunWithInput", context.Background(), []byte(nil), "no-input-command", []string{"arg1"}).
					Return([]byte("no input result"), nil)
			},
		},
		{
			name:           "command execution with input error",
			input:          []byte("bad input"),
			command:        "failing-input-command",
			args:           []string{},
			expectedOutput: nil,
			expectedError:  assert.AnError,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunWithInput", context.Background(), []byte("bad input"), "failing-input-command", []string{}).
					Return(nil, assert.AnError)
			},
		},
		{
			name:           "empty input and output",
			input:          []byte{},
			command:        "empty-command",
			args:           []string{"test"},
			expectedOutput: []byte{},
			expectedError:  nil,
			setupMock: func(m *MockCommandRunner) {
				m.On("RunWithInput", context.Background(), []byte{}, "empty-command", []string{"test"}).
					Return([]byte{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRunner := &MockCommandRunner{}
			tt.setupMock(mockRunner)

			ctx := context.Background()
			output, err := mockRunner.RunWithInput(ctx, tt.input, tt.command, tt.args...)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedOutput, output)
			mockRunner.AssertExpectations(t)
		})
	}
}

func TestMockCommandRunner_ImplementsInterface(t *testing.T) {
	// Test that MockCommandRunner implements CommandRunner interface
	var _ CommandRunner = (*MockCommandRunner)(nil)

	// Test instantiation
	mockRunner := &MockCommandRunner{}
	require.NotNil(t, mockRunner)

	// Test that methods exist and can be called
	mockRunner.On("Run", context.Background(), "test", []string(nil)).Return([]byte("test"), nil)
	mockRunner.On("RunWithInput", context.Background(), []byte("input"), "test", []string(nil)).Return([]byte("test"), nil)

	// Verify method calls work
	output, err := mockRunner.Run(context.Background(), "test")
	require.NoError(t, err)
	assert.Equal(t, []byte("test"), output)

	output, err = mockRunner.RunWithInput(context.Background(), []byte("input"), "test")
	require.NoError(t, err)
	assert.Equal(t, []byte("test"), output)

	mockRunner.AssertExpectations(t)
}

func TestMockCommandRunner_NilHandling(t *testing.T) {
	t.Run("Run method handles nil output correctly", func(t *testing.T) {
		mockRunner := &MockCommandRunner{}
		mockRunner.On("Run", context.Background(), "nil-test", []string(nil)).Return(nil, assert.AnError)

		output, err := mockRunner.Run(context.Background(), "nil-test")
		require.Error(t, err)
		assert.Nil(t, output)
		mockRunner.AssertExpectations(t)
	})

	t.Run("RunWithInput method handles nil output correctly", func(t *testing.T) {
		mockRunner := &MockCommandRunner{}
		mockRunner.On("RunWithInput", context.Background(), []byte("test"), "nil-test", []string(nil)).Return(nil, assert.AnError)

		output, err := mockRunner.RunWithInput(context.Background(), []byte("test"), "nil-test")
		require.Error(t, err)
		assert.Nil(t, output)
		mockRunner.AssertExpectations(t)
	})
}

func TestMockCommandRunner_ContextPassing(t *testing.T) {
	t.Run("context is correctly passed to Run method", func(t *testing.T) {
		mockRunner := &MockCommandRunner{}
		type contextKey string
		ctx := context.WithValue(context.Background(), contextKey("test-key"), "test-value")

		mockRunner.On("Run", ctx, "context-test", []string{"arg"}).Return([]byte("success"), nil)

		output, err := mockRunner.Run(ctx, "context-test", "arg")
		require.NoError(t, err)
		assert.Equal(t, []byte("success"), output)
		mockRunner.AssertExpectations(t)
	})

	t.Run("context is correctly passed to RunWithInput method", func(t *testing.T) {
		mockRunner := &MockCommandRunner{}
		type contextKey string
		ctx := context.WithValue(context.Background(), contextKey("test-key"), "test-value")

		mockRunner.On("RunWithInput", ctx, []byte("input"), "context-test", []string{"arg"}).Return([]byte("success"), nil)

		output, err := mockRunner.RunWithInput(ctx, []byte("input"), "context-test", "arg")
		require.NoError(t, err)
		assert.Equal(t, []byte("success"), output)
		mockRunner.AssertExpectations(t)
	})
}
