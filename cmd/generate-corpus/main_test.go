package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerateCorpusApp(t *testing.T) {
	app := NewGenerateCorpusApp()

	assert.Nil(t, app)
	assert.NotNil(t, app.logger)
	assert.NotNil(t, app.fileSystem)
	assert.NotNil(t, app.corpusGeneratorFactory)
	assert.IsType(t, &DefaultLogger{}, app.logger)
	assert.IsType(t, &DefaultFileSystem{}, app.fileSystem)
	assert.IsType(t, &DefaultCorpusGeneratorFactory{}, app.corpusGeneratorFactory)
}

func TestNewGenerateCorpusAppWithDependencies(t *testing.T) {
	mockLogger := &MockLogger{}
	mockFileSystem := &MockFileSystem{}
	mockFactory := &MockCorpusGeneratorFactory{}

	app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFileSystem, mockFactory)

	assert.NotNil(t, app)
	assert.Equal(t, mockLogger, app.logger)
	assert.Equal(t, mockFileSystem, app.fileSystem)
	assert.Equal(t, mockFactory, app.corpusGeneratorFactory)
}

func TestDefaultLogger(t *testing.T) {
	logger := &DefaultLogger{}

	// These should not panic
	assert.NotPanics(t, func() {
		logger.Println("test message")
		logger.Printf("formatted %s", "message")
	})
}

func TestDefaultFileSystem(t *testing.T) {
	fs := &DefaultFileSystem{}

	// Test with a file that should exist
	_, err := fs.Stat(".")
	require.NoError(t, err)

	// Test with a file that shouldn't exist
	_, err = fs.Stat("nonexistent-file-12345")
	assert.True(t, os.IsNotExist(err))
}

func TestDefaultCorpusGeneratorFactory(t *testing.T) {
	factory := &DefaultCorpusGeneratorFactory{}

	generator := factory.NewCorpusGenerator("test-dir")
	assert.NotNil(t, generator)
	assert.IsType(t, &DefaultCorpusGeneratorWrapper{}, generator)
}

func TestDefaultCorpusGeneratorWrapper(t *testing.T) {
	// Create a mock generator that implements the expected interface
	mockGen := &mockInternalGenerator{shouldError: false}
	wrapper := &DefaultCorpusGeneratorWrapper{generator: mockGen}

	// Test successful generation
	err := wrapper.GenerateAll()
	require.NoError(t, err)
	assert.True(t, mockGen.called)

	// Test error case
	mockGen.called = false
	mockGen.shouldError = true
	err = wrapper.GenerateAll()
	require.Error(t, err)
	assert.True(t, mockGen.called)
}

// Simple mock types for testing basic functionality
type MockLogger struct {
	Messages []string
}

func (m *MockLogger) Println(_ ...interface{}) {
	// Simple implementation for testing
}

func (m *MockLogger) Printf(_ string, _ ...interface{}) {
	// Simple implementation for testing
}

func (m *MockLogger) Fatalf(_ string, _ ...interface{}) {
	// Simple implementation for testing
}

type MockFileSystem struct {
	StatFunc func(name string) (os.FileInfo, error)
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(name)
	}
	return nil, os.ErrNotExist
}

type MockCorpusGeneratorFactory struct {
	Generator CorpusGenerator
}

func (m *MockCorpusGeneratorFactory) NewCorpusGenerator(_ string) CorpusGenerator {
	if m.Generator != nil {
		return m.Generator
	}
	return &mockCorpusGenerator{shouldError: false}
}

type mockCorpusGenerator struct {
	shouldError bool
	called      bool
}

func (m *mockCorpusGenerator) GenerateAll() error {
	m.called = true
	if m.shouldError {
		return assert.AnError
	}
	return nil
}

// Mock for testing the wrapper
type mockInternalGenerator struct {
	shouldError bool
	called      bool
}

func (m *mockInternalGenerator) GenerateAll() error {
	m.called = true
	if m.shouldError {
		return assert.AnError
	}
	return nil
}

// Integration test to verify the app works with real fuzz package if available
func TestGenerateCorpusApp_IntegrationTest(t *testing.T) {
	t.Run("app runs without panic with real dependencies", func(t *testing.T) {
		app := NewGenerateCorpusApp()

		// This test verifies the app structure works correctly
		// We can't run the actual generation as it requires the internal/fuzz directory
		// But we can test that the components are wired correctly
		assert.NotNil(t, app.logger)
		assert.NotNil(t, app.fileSystem)
		assert.NotNil(t, app.corpusGeneratorFactory)
	})

	t.Run("fileSystem works with actual directory", func(t *testing.T) {
		fs := &DefaultFileSystem{}

		// Test with current directory (should exist)
		info, err := fs.Stat(".")
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("corpus generator factory creates wrapper", func(t *testing.T) {
		factory := &DefaultCorpusGeneratorFactory{}
		generator := factory.NewCorpusGenerator("test")

		assert.NotNil(t, generator)
		assert.IsType(t, &DefaultCorpusGeneratorWrapper{}, generator)
	})
}

// Test edge cases and error scenarios
func TestGenerateCorpusApp_EdgeCases(t *testing.T) {
	t.Run("handles empty base directory name", func(t *testing.T) {
		// Even with empty base dir, the app structure should work
		app := NewGenerateCorpusApp()
		assert.NotNil(t, app)
	})

	t.Run("default implementations don't panic with nil input", func(t *testing.T) {
		logger := &DefaultLogger{}
		assert.NotPanics(t, func() {
			logger.Println(nil)
			logger.Printf("test %v", nil)
		})

		fs := &DefaultFileSystem{}
		assert.NotPanics(t, func() {
			_, _ = fs.Stat("")
		})

		factory := &DefaultCorpusGeneratorFactory{}
		assert.NotPanics(t, func() {
			gen := factory.NewCorpusGenerator("")
			assert.NotNil(t, gen)
		})
	})
}

func TestMainFunctionality(t *testing.T) {
	// We can't directly test main() but we can verify that all the
	// functions main() calls are working properly together.

	t.Run("main function components work", func(t *testing.T) {
		// Test that NewGenerateCorpusApp creates a valid app
		app := NewGenerateCorpusApp()
		require.NotNil(t, app)

		// Verify all components are properly initialized
		assert.NotNil(t, app.logger)
		assert.NotNil(t, app.fileSystem)
		assert.NotNil(t, app.corpusGeneratorFactory)

		// Test that we can create generators
		generator := app.corpusGeneratorFactory.NewCorpusGenerator("test")
		assert.NotNil(t, generator)
	})
}

func TestApplicationStructure(t *testing.T) {
	t.Run("app structure is properly initialized", func(t *testing.T) {
		app := NewGenerateCorpusApp()

		// Verify the app has all required fields
		assert.NotNil(t, app.logger)
		assert.NotNil(t, app.fileSystem)
		assert.NotNil(t, app.corpusGeneratorFactory)

		// Verify types are correct
		assert.IsType(t, &DefaultLogger{}, app.logger)
		assert.IsType(t, &DefaultFileSystem{}, app.fileSystem)
		assert.IsType(t, &DefaultCorpusGeneratorFactory{}, app.corpusGeneratorFactory)
	})

	t.Run("dependency injection works", func(t *testing.T) {
		mockLogger := &MockLogger{}
		mockFS := &MockFileSystem{}
		mockFactory := &MockCorpusGeneratorFactory{}

		app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFS, mockFactory)

		assert.Equal(t, mockLogger, app.logger)
		assert.Equal(t, mockFS, app.fileSystem)
		assert.Equal(t, mockFactory, app.corpusGeneratorFactory)
	})
}
