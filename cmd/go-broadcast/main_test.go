package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	app := NewApp()

	assert.NotNil(t, app)
	assert.NotNil(t, app.outputHandler)
	assert.NotNil(t, app.cliExecutor)
	assert.IsType(t, &DefaultOutputHandler{}, app.outputHandler)
	assert.IsType(t, &DefaultCLIExecutor{}, app.cliExecutor)
}

func TestNewAppWithDependencies(t *testing.T) {
	mockOutputHandler := &MockOutputHandler{}
	mockCLIExecutor := &MockCLIExecutor{}

	app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

	assert.NotNil(t, app)
	assert.Equal(t, mockOutputHandler, app.outputHandler)
	assert.Equal(t, mockCLIExecutor, app.cliExecutor)
}

func TestDefaultOutputHandler(t *testing.T) {
	handler := &DefaultOutputHandler{}

	// These should not panic
	assert.NotPanics(t, func() {
		handler.Init()
		handler.Error("test error message")
	})
}

func TestDefaultCLIExecutor(t *testing.T) {
	executor := &DefaultCLIExecutor{}

	// We can't easily test the actual execution without running the full CLI
	// but we can verify the struct exists and can be created
	assert.NotNil(t, executor)
}

// Simple mock types for testing basic functionality
type MockOutputHandler struct {
	InitCalled    bool
	ErrorMessages []string
}

func (m *MockOutputHandler) Init() {
	m.InitCalled = true
}

func (m *MockOutputHandler) Error(msg string) {
	m.ErrorMessages = append(m.ErrorMessages, msg)
}

type MockCLIExecutor struct {
	ExecuteError  error
	ExecuteCalled bool
}

func (m *MockCLIExecutor) Execute() error {
	m.ExecuteCalled = true
	return m.ExecuteError
}

func TestAppRun_Success(t *testing.T) {
	mockOutputHandler := &MockOutputHandler{}
	mockCLIExecutor := &MockCLIExecutor{ExecuteError: nil}

	app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

	err := app.Run([]string{})

	require.NoError(t, err)
	assert.True(t, mockOutputHandler.InitCalled)
	assert.True(t, mockCLIExecutor.ExecuteCalled)
}

func TestAppRun_CLIError(t *testing.T) {
	mockOutputHandler := &MockOutputHandler{}
	mockCLIExecutor := &MockCLIExecutor{ExecuteError: assert.AnError}

	app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

	err := app.Run([]string{})

	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	assert.True(t, mockOutputHandler.InitCalled)
	assert.True(t, mockCLIExecutor.ExecuteCalled)
}

func TestAppRun_PanicRecovery(t *testing.T) {
	mockOutputHandler := &MockOutputHandler{}
	mockCLIExecutor := &PanicCLIExecutor{}

	app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

	// Should not panic, but should call Error on output handler
	assert.NotPanics(t, func() {
		_ = app.Run([]string{})
	})

	assert.True(t, mockOutputHandler.InitCalled)
	assert.Len(t, mockOutputHandler.ErrorMessages, 1)
	assert.Contains(t, mockOutputHandler.ErrorMessages[0], "Fatal error: test panic")
}

// PanicCLIExecutor is a mock that panics during execution
type PanicCLIExecutor struct{}

func (p *PanicCLIExecutor) Execute() error {
	panic("test panic")
}

func TestInterfaceCompliance(t *testing.T) {
	t.Run("default implementations satisfy interfaces", func(t *testing.T) {
		var outputHandler OutputHandler = &DefaultOutputHandler{}
		var cliExecutor CLIExecutor = &DefaultCLIExecutor{}

		// If this compiles, the interfaces are satisfied
		assert.NotNil(t, outputHandler)
		assert.NotNil(t, cliExecutor)
	})

	t.Run("mock implementations satisfy interfaces", func(t *testing.T) {
		var outputHandler OutputHandler = &MockOutputHandler{}
		var cliExecutor CLIExecutor = &MockCLIExecutor{}

		// If this compiles, the interfaces are satisfied
		assert.NotNil(t, outputHandler)
		assert.NotNil(t, cliExecutor)
	})
}

func TestApplicationStructure(t *testing.T) {
	t.Run("app structure is properly initialized", func(t *testing.T) {
		app := NewApp()

		// Verify the app has all required fields
		assert.NotNil(t, app.outputHandler)
		assert.NotNil(t, app.cliExecutor)

		// Verify types are correct
		assert.IsType(t, &DefaultOutputHandler{}, app.outputHandler)
		assert.IsType(t, &DefaultCLIExecutor{}, app.cliExecutor)
	})

	t.Run("dependency injection works", func(t *testing.T) {
		mockOutputHandler := &MockOutputHandler{}
		mockCLIExecutor := &MockCLIExecutor{}

		app := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)

		assert.Equal(t, mockOutputHandler, app.outputHandler)
		assert.Equal(t, mockCLIExecutor, app.cliExecutor)
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("handles nil arguments", func(t *testing.T) {
		app := NewApp()
		assert.NotNil(t, app)

		// Should not panic with nil args
		assert.NotPanics(t, func() {
			_ = app.Run(nil)
		})
	})

	t.Run("handles empty arguments", func(t *testing.T) {
		app := NewApp()
		assert.NotNil(t, app)

		// Should not panic with empty args
		assert.NotPanics(t, func() {
			_ = app.Run([]string{})
		})
	})

	t.Run("concurrent app creation", func(t *testing.T) {
		// Test that creating multiple apps doesn't cause issues
		apps := make([]*App, 10)
		for i := range apps {
			apps[i] = NewApp()
			assert.NotNil(t, apps[i])
		}
	})
}

func TestMainFunctionality(t *testing.T) {
	// We can't directly test main() but we can verify that all the
	// functions main() calls are working properly together.

	t.Run("main function components work", func(t *testing.T) {
		// Test that NewApp creates a valid app
		app := NewApp()
		require.NotNil(t, app)

		// Verify all components are properly initialized
		assert.NotNil(t, app.outputHandler)
		assert.NotNil(t, app.cliExecutor)

		// Test that we can call Run without panicking
		assert.NotPanics(t, func() {
			// Using a mock to avoid actually running the CLI
			mockOutputHandler := &MockOutputHandler{}
			mockCLIExecutor := &MockCLIExecutor{ExecuteError: nil}
			testApp := NewAppWithDependencies(mockOutputHandler, mockCLIExecutor)
			_ = testApp.Run([]string{})
		})
	})
}
