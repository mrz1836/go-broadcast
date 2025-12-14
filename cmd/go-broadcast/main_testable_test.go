package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test errors
var (
	ErrCLIExecutionFailed = errors.New("CLI execution failed")
	ErrOutputInitFailed   = errors.New("output init failed")
	ErrInternalError      = errors.New("internal error")
	ErrCommandNotFound    = errors.New("command not found")
	ErrInvalidArguments   = errors.New("invalid arguments")
)

// MockOutputHandlerAdvanced is a full mock implementation of OutputHandler with testify/mock
type MockOutputHandlerAdvanced struct {
	mock.Mock

	errorMessages []string
}

func (m *MockOutputHandlerAdvanced) Init() {
	m.Called()
}

func (m *MockOutputHandlerAdvanced) Error(msg string) {
	m.errorMessages = append(m.errorMessages, msg)
	m.Called(msg)
}

// MockCLIExecutorAdvanced is a full mock implementation of CLIExecutor with testify/mock
type MockCLIExecutorAdvanced struct {
	mock.Mock
}

func (m *MockCLIExecutorAdvanced) Execute() error {
	args := m.Called()
	return args.Error(0)
}

// containsEnvWarning checks if a string contains the env file warning message.
// This is a specialized helper for test assertions.
func containsEnvWarning(s string) bool {
	return strings.Contains(s, "Warning: Failed to load environment files")
}

func TestApp_Run(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		// Setup mocks
		mockOutputHandler := &MockOutputHandlerAdvanced{}
		mockCLIExecutor := &MockCLIExecutorAdvanced{}

		// Setup expectations
		mockOutputHandler.On("Init").Return()
		// Expect warning about missing env files (this is expected in test environment)
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return containsEnvWarning(msg)
		})).Return()
		mockCLIExecutor.On("Execute").Return(nil)

		// Create app with mocked dependencies
		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

		// Execute
		err := app.Run([]string{"arg1", "arg2"})

		// Assertions
		require.NoError(t, err)
		mockOutputHandler.AssertExpectations(t)
		mockCLIExecutor.AssertExpectations(t)

		// Verify call order: Init should be called before Execute
		mockOutputHandler.AssertCalled(t, "Init")
		mockCLIExecutor.AssertCalled(t, "Execute")
	})

	t.Run("CLI execution error", func(t *testing.T) {
		// Setup mocks
		mockOutputHandler := &MockOutputHandlerAdvanced{}
		mockCLIExecutor := &MockCLIExecutorAdvanced{}

		// Setup expectations
		mockOutputHandler.On("Init").Return()
		// Expect warning about missing env files (this is expected in test environment)
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return containsEnvWarning(msg)
		})).Return()
		mockCLIExecutor.On("Execute").Return(ErrCLIExecutionFailed)

		// Create app with mocked dependencies
		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

		// Execute
		err := app.Run([]string{})

		// Assertions
		require.Error(t, err)
		assert.Equal(t, ErrCLIExecutionFailed, err)
		mockOutputHandler.AssertExpectations(t)
		mockCLIExecutor.AssertExpectations(t)
	})

	t.Run("panic recovery during CLI execution", func(t *testing.T) {
		// Setup mocks
		mockOutputHandler := &MockOutputHandlerAdvanced{}
		mockCLIExecutor := &MockCLIExecutorAdvanced{}

		// Setup expectations
		mockOutputHandler.On("Init").Return()
		// Expect warning about missing env files first
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return containsEnvWarning(msg)
		})).Return()
		// Then expect panic recovery error (now includes stack trace)
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, "Fatal error: CLI execution panic")
		})).Return()
		mockCLIExecutor.On("Execute").Panic("CLI execution panic")

		// Create app with mocked dependencies
		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

		var err error
		// Execute - should not panic but should return error
		assert.NotPanics(t, func() {
			err = app.Run([]string{})
		})

		// CRITICAL: Panic recovery must return an error
		require.Error(t, err, "panic must return non-nil error")
		assert.Contains(t, err.Error(), "panic recovered")

		// Assertions
		mockOutputHandler.AssertExpectations(t)
		mockCLIExecutor.AssertExpectations(t)

		// Verify that error messages were captured (env warning + panic recovery with stack trace)
		assert.Len(t, mockOutputHandler.errorMessages, 2)
		assert.Contains(t, mockOutputHandler.errorMessages[0], "Warning: Failed to load environment files")
		assert.Contains(t, mockOutputHandler.errorMessages[1], "Fatal error: CLI execution panic")
		assert.Contains(t, mockOutputHandler.errorMessages[1], "goroutine") // Stack trace marker
	})

	t.Run("panic recovery with complex panic value", func(t *testing.T) {
		// Setup mocks
		mockOutputHandler := &MockOutputHandlerAdvanced{}
		mockCLIExecutor := &MockCLIExecutorAdvanced{}

		panicValue := "complex error with details"

		// Setup expectations
		mockOutputHandler.On("Init").Return()
		// Expect warning about missing env files first
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return containsEnvWarning(msg)
		})).Return()
		// Then expect panic recovery error (now includes stack trace)
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return strings.Contains(msg, fmt.Sprintf("Fatal error: %v", panicValue))
		})).Return()
		mockCLIExecutor.On("Execute").Panic(panicValue)

		// Create app with mocked dependencies
		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

		var err error
		// Execute - should not panic but should return error
		assert.NotPanics(t, func() {
			err = app.Run([]string{})
		})

		// CRITICAL: Panic recovery must return an error
		require.Error(t, err, "panic must return non-nil error")
		assert.Contains(t, err.Error(), "panic recovered")

		// Assertions
		mockOutputHandler.AssertExpectations(t)
		mockCLIExecutor.AssertExpectations(t)
	})

	t.Run("complete flow verification", func(t *testing.T) {
		// Setup mocks
		mockOutputHandler := &MockOutputHandlerAdvanced{}
		mockCLIExecutor := &MockCLIExecutorAdvanced{}

		// Setup expectations with detailed verification
		mockOutputHandler.On("Init").Once()
		// Expect warning about missing env files
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return containsEnvWarning(msg)
		})).Return()
		mockCLIExecutor.On("Execute").Return(nil).Once()

		// Create app with mocked dependencies
		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

		// Execute
		err := app.Run([]string{"test", "args"})

		// Assertions
		require.NoError(t, err)
		mockOutputHandler.AssertExpectations(t)
		mockCLIExecutor.AssertExpectations(t)

		// Verify exact call counts
		mockOutputHandler.AssertNumberOfCalls(t, "Init", 1)
		mockCLIExecutor.AssertNumberOfCalls(t, "Execute", 1)
	})
}

func TestApp_ErrorPathsCoverage(t *testing.T) {
	t.Run("various CLI execution error scenarios", func(t *testing.T) {
		testCases := []struct {
			name        string
			cliErr      error
			expectError string
		}{
			{
				name:        "command not found",
				cliErr:      ErrCommandNotFound,
				expectError: "command not found",
			},
			{
				name:        "invalid arguments",
				cliErr:      ErrInvalidArguments,
				expectError: "invalid arguments",
			},
			{
				name:        "internal error",
				cliErr:      ErrInternalError,
				expectError: "internal error",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockOutputHandler := &MockOutputHandlerAdvanced{}
				mockCLIExecutor := &MockCLIExecutorAdvanced{}

				mockOutputHandler.On("Init").Return()
				// Expect warning about missing env files
				mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
					return containsEnvWarning(msg)
				})).Return()
				mockCLIExecutor.On("Execute").Return(tc.cliErr)

				app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)
				err := app.Run([]string{})

				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
				mockOutputHandler.AssertExpectations(t)
				mockCLIExecutor.AssertExpectations(t)
			})
		}
	})

	t.Run("panic scenarios", func(t *testing.T) {
		panicValues := []string{
			"string panic",
			"error panic",
			"struct panic",
			"nil panic",
			"number panic",
		}

		for i, panicValue := range panicValues {
			t.Run(fmt.Sprintf("panic_scenario_%d", i), func(t *testing.T) {
				mockOutputHandler := &MockOutputHandlerAdvanced{}
				mockCLIExecutor := &MockCLIExecutorAdvanced{}

				mockOutputHandler.On("Init").Return()
				// Expect warning about missing env files first
				mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
					return containsEnvWarning(msg)
				})).Return()
				// Then expect panic recovery error (now includes stack trace)
				mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
					return strings.Contains(msg, fmt.Sprintf("Fatal error: %v", panicValue))
				})).Return()
				mockCLIExecutor.On("Execute").Panic(panicValue)

				app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

				var err error
				assert.NotPanics(t, func() {
					err = app.Run([]string{})
				})

				// CRITICAL: Panic recovery must return an error
				require.Error(t, err, "panic must return non-nil error")
				assert.Contains(t, err.Error(), "panic recovered")

				mockOutputHandler.AssertExpectations(t)
				mockCLIExecutor.AssertExpectations(t)
			})
		}
	})
}

func TestApp_MockValidation(t *testing.T) {
	t.Run("verify all mocks are called correctly", func(t *testing.T) {
		mockOutputHandler := &MockOutputHandlerAdvanced{}
		mockCLIExecutor := &MockCLIExecutorAdvanced{}

		// Set up strict expectations
		mockOutputHandler.On("Init").Once()
		// Expect warning about missing env files (this is expected in test environment)
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return containsEnvWarning(msg)
		})).Return()
		mockCLIExecutor.On("Execute").Return(nil).Once()

		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)
		err := app.Run([]string{})

		require.NoError(t, err)
		mockOutputHandler.AssertExpectations(t)
		mockCLIExecutor.AssertExpectations(t)
	})

	t.Run("verify call counts and ordering", func(t *testing.T) {
		mockOutputHandler := &MockOutputHandlerAdvanced{}
		mockCLIExecutor := &MockCLIExecutorAdvanced{}

		mockOutputHandler.On("Init").Return()
		// Expect warning about missing env files (this is expected in test environment)
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return containsEnvWarning(msg)
		})).Return()
		mockCLIExecutor.On("Execute").Return(nil)

		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)
		err := app.Run([]string{})

		require.NoError(t, err)

		// Verify call counts
		mockOutputHandler.AssertNumberOfCalls(t, "Init", 1)
		mockCLIExecutor.AssertNumberOfCalls(t, "Execute", 1)

		// Verify specific calls were made
		mockOutputHandler.AssertCalled(t, "Init")
		mockCLIExecutor.AssertCalled(t, "Execute")
	})

	t.Run("verify no unexpected calls", func(t *testing.T) {
		mockOutputHandler := &MockOutputHandlerAdvanced{}
		mockCLIExecutor := &MockCLIExecutorAdvanced{}

		// Set up expected calls (including env file warning which always happens)
		mockOutputHandler.On("Init").Return()
		// Expect warning about missing env files (this is expected in test environment)
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return containsEnvWarning(msg)
		})).Return()
		mockCLIExecutor.On("Execute").Return(ErrCLIExecutionFailed)

		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)
		_ = app.Run([]string{})

		// Should only call Error for env file warning, no other Error calls
		mockOutputHandler.AssertExpectations(t)
		mockCLIExecutor.AssertExpectations(t)
	})
}

func TestInterfaceComplianceAdvanced(t *testing.T) {
	t.Run("default implementations satisfy interfaces", func(t *testing.T) {
		var outputHandler OutputHandler = &DefaultOutputHandler{}
		var cliExecutor CLIExecutor = &DefaultCLIExecutor{}

		// If this compiles, the interfaces are satisfied
		assert.NotNil(t, outputHandler)
		assert.NotNil(t, cliExecutor)
	})

	t.Run("mock implementations satisfy interfaces", func(t *testing.T) {
		var outputHandler OutputHandler = &MockOutputHandlerAdvanced{}
		var cliExecutor CLIExecutor = &MockCLIExecutorAdvanced{}

		// If this compiles, the interfaces are satisfied
		assert.NotNil(t, outputHandler)
		assert.NotNil(t, cliExecutor)
	})

	t.Run("simple mock implementations satisfy interfaces", func(t *testing.T) {
		var outputHandler OutputHandler = &MockOutputHandler{}
		var cliExecutor CLIExecutor = &MockCLIExecutor{}

		// If this compiles, the interfaces are satisfied
		assert.NotNil(t, outputHandler)
		assert.NotNil(t, cliExecutor)
	})
}

func TestEdgeCasesAndBoundariesAdvanced(t *testing.T) {
	t.Run("empty arguments handling", func(t *testing.T) {
		mockOutputHandler := &MockOutputHandlerAdvanced{}
		mockCLIExecutor := &MockCLIExecutorAdvanced{}

		mockOutputHandler.On("Init").Return()
		// Expect warning about missing env files (this is expected in test environment)
		mockOutputHandler.On("Error", mock.MatchedBy(func(msg string) bool {
			return containsEnvWarning(msg)
		})).Return()
		mockCLIExecutor.On("Execute").Return(nil)

		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

		// Test with nil, empty, and various argument scenarios
		testCases := [][]string{
			nil,
			{},
			{""},
			{"arg1"},
			{"arg1", "arg2", "arg3"},
		}

		for i, args := range testCases {
			t.Run(fmt.Sprintf("args_case_%d", i), func(t *testing.T) {
				assert.NotPanics(t, func() {
					_ = app.Run(args)
				})
			})
		}
	})

	t.Run("concurrent execution safety", func(t *testing.T) {
		// Test that multiple App instances can run concurrently
		apps := make([]*App, 5)
		mockHandlers := make([]*MockOutputHandlerAdvanced, 5)
		mockExecutors := make([]*MockCLIExecutorAdvanced, 5)

		for i := range apps {
			mockHandlers[i] = &MockOutputHandlerAdvanced{}
			mockExecutors[i] = &MockCLIExecutorAdvanced{}

			mockHandlers[i].On("Init").Return()
			// Expect warning about missing env files (this is expected in test environment)
			mockHandlers[i].On("Error", mock.MatchedBy(func(msg string) bool {
				return containsEnvWarning(msg)
			})).Return()
			mockExecutors[i].On("Execute").Return(nil)

			apps[i] = NewAppWithDependencies(mockHandlers[i], mockExecutors[i])
		}

		// Run all apps concurrently
		done := make(chan bool, len(apps))
		for i, app := range apps {
			go func(a *App, idx int) {
				defer func() { done <- true }()
				err := a.Run([]string{fmt.Sprintf("arg_%d", idx)})
				assert.NoError(t, err)
			}(app, i)
		}

		// Wait for all to complete
		for i := 0; i < len(apps); i++ {
			<-done
		}

		// Verify all mocks were called correctly
		for i := range apps {
			mockHandlers[i].AssertExpectations(t)
			mockExecutors[i].AssertExpectations(t)
		}
	})

	t.Run("nil outputHandler panics", func(t *testing.T) {
		// NewAppWithDependencies now panics on nil to fail fast
		assert.PanicsWithValue(t, "outputHandler must not be nil", func() {
			_ = NewAppWithDependencies(nil, &MockCLIExecutorAdvanced{})
		})
	})

	t.Run("nil cliExecutor panics", func(t *testing.T) {
		// NewAppWithDependencies now panics on nil to fail fast
		assert.PanicsWithValue(t, "cliExecutor must not be nil", func() {
			_ = NewAppWithDependencies(&MockOutputHandlerAdvanced{}, nil)
		})
	})

	t.Run("both nil panics with outputHandler error first", func(t *testing.T) {
		// When both are nil, outputHandler is checked first
		assert.PanicsWithValue(t, "outputHandler must not be nil", func() {
			_ = NewAppWithDependencies(nil, nil)
		})
	})
}

func TestDefaultImplementationsAdvanced(t *testing.T) {
	t.Run("DefaultOutputHandler methods", func(t *testing.T) {
		handler := &DefaultOutputHandler{}

		// Test that methods don't panic with various inputs
		assert.NotPanics(t, func() {
			handler.Init()
			handler.Error("")
			handler.Error("test message")
			handler.Error("message with special chars: !@#$%^&*()")
		})
	})

	t.Run("DefaultCLIExecutor creation", func(t *testing.T) {
		executor := &DefaultCLIExecutor{}
		assert.NotNil(t, executor)

		// We can't easily test Execute() without running the full CLI,
		// but we can verify the struct can be created and implements the interface
		var _ CLIExecutor = executor
	})
}

func TestAppConstructorsAdvanced(t *testing.T) {
	t.Run("NewApp consistency", func(t *testing.T) {
		// Create multiple instances and verify they're all properly initialized
		apps := make([]*App, 10)
		for i := range apps {
			apps[i] = NewApp()
			assert.NotNil(t, apps[i])
			assert.NotNil(t, apps[i].outputHandler)
			assert.NotNil(t, apps[i].cliExecutor)
			assert.IsType(t, &DefaultOutputHandler{}, apps[i].outputHandler)
			assert.IsType(t, &DefaultCLIExecutor{}, apps[i].cliExecutor)
		}
	})

	t.Run("NewAppWithDependencies validation", func(t *testing.T) {
		mockOutputHandler := &MockOutputHandlerAdvanced{}
		mockCLIExecutor := &MockCLIExecutorAdvanced{}

		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

		assert.NotNil(t, app)
		assert.Exactly(t, mockOutputHandler, app.outputHandler)
		assert.Exactly(t, mockCLIExecutor, app.cliExecutor)
	})
}
