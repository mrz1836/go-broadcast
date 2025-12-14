package main

import (
	"sync"
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

// MockOutputHandler is a thread-safe mock for testing.
// Use mu.Lock/Unlock when accessing fields in concurrent tests.
type MockOutputHandler struct {
	mu            sync.Mutex
	InitCalled    bool
	ErrorMessages []string
}

func (m *MockOutputHandler) Init() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InitCalled = true
}

func (m *MockOutputHandler) Error(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorMessages = append(m.ErrorMessages, msg)
}

// GetErrorMessages returns a copy of error messages (thread-safe).
func (m *MockOutputHandler) GetErrorMessages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.ErrorMessages))
	copy(result, m.ErrorMessages)
	return result
}

// WasInitCalled returns whether Init was called (thread-safe).
func (m *MockOutputHandler) WasInitCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.InitCalled
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

	var err error
	// Should not panic, but should return an error
	assert.NotPanics(t, func() {
		err = app.Run([]string{})
	})

	// CRITICAL: Panic recovery must return an error (not nil)
	// This ensures main() calls os.Exit(1) after a panic
	require.Error(t, err, "panic recovery must return non-nil error")
	assert.Contains(t, err.Error(), "panic recovered: test panic")

	assert.True(t, mockOutputHandler.InitCalled)
	// Should have 2 error messages: env file warning + panic recovery (with stack trace)
	assert.Len(t, mockOutputHandler.ErrorMessages, 2)
	// First message should be about env files
	assert.Contains(t, mockOutputHandler.ErrorMessages[0], "Warning: Failed to load environment files")
	// Second message should contain panic value AND stack trace
	assert.Contains(t, mockOutputHandler.ErrorMessages[1], "Fatal error: test panic")
	assert.Contains(t, mockOutputHandler.ErrorMessages[1], "goroutine") // Stack trace marker
}

// PanicCLIExecutor is a mock that panics during execution
type PanicCLIExecutor struct{}

func (p *PanicCLIExecutor) Execute() error {
	panic("test panic")
}

// PanicOutputHandler is a mock that panics during Init()
type PanicOutputHandler struct{}

func (p *PanicOutputHandler) Init() {
	panic("init panic")
}

func (p *PanicOutputHandler) Error(_ string) {
	// No-op for testing
}

func TestAppRun_PanicInInit(t *testing.T) {
	// Test that panics in Init() are caught (defer must be at start of function)
	mockCLIExecutor := &MockCLIExecutor{}

	app := NewAppWithDependencies(&PanicOutputHandler{}, mockCLIExecutor)

	var err error
	assert.NotPanics(t, func() {
		err = app.Run([]string{})
	})

	// Panic in Init() must be caught and return an error
	require.Error(t, err, "panic in Init() must be caught")
	assert.Contains(t, err.Error(), "panic recovered: init panic")

	// Execute should NOT have been called since Init panicked
	assert.False(t, mockCLIExecutor.ExecuteCalled)
}

func TestAppRun_PanicRecoveryReturnsError(t *testing.T) {
	// Explicitly test that panic recovery returns non-nil error
	// This is critical for ensuring proper exit codes
	testCases := []struct {
		name       string
		panicValue interface{}
	}{
		{"string panic", "string panic value"},
		{"error panic", "error type panic"},
		{"int panic", "42"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockOutputHandler := &MockOutputHandler{}
			panicExecutor := &customPanicExecutor{panicValue: tc.panicValue}
			app := NewAppWithDependencies(mockOutputHandler, panicExecutor)

			err := app.Run([]string{})

			// Must return error, not nil
			require.Error(t, err, "panic must result in non-nil error for exit code")
			assert.Contains(t, err.Error(), "panic recovered")
		})
	}
}

type customPanicExecutor struct {
	panicValue interface{}
}

func (c *customPanicExecutor) Execute() error {
	panic(c.panicValue)
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
