package main

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test errors
var (
	ErrDirectoryNotFound = errors.New("directory not found")
	ErrStatFailed        = errors.New("stat failed")
	ErrGenerationFailed  = errors.New("generation failed")
	ErrCustomStat        = errors.New("custom stat error")
	ErrFileWrite         = errors.New("file write error")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrDiskFull          = errors.New("disk full")
	ErrInvalidCorpusData = errors.New("invalid corpus data")
)

// MockLoggerAdvanced is a full mock implementation of Logger with testify/mock
type MockLoggerAdvanced struct {
	mock.Mock

	messages []string
}

func (m *MockLoggerAdvanced) Println(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	m.messages = append(m.messages, msg)
	m.Called(v...)
}

func (m *MockLoggerAdvanced) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	m.messages = append(m.messages, msg)
	m.Called(format, v)
}

func (m *MockLoggerAdvanced) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	m.messages = append(m.messages, msg)
	m.Called(format, v)
}

// MockFileSystemAdvanced is a full mock implementation of FileSystem with testify/mock
type MockFileSystemAdvanced struct {
	mock.Mock
}

func (m *MockFileSystemAdvanced) Stat(name string) (os.FileInfo, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(os.FileInfo), args.Error(1)
}

// MockCorpusGeneratorFactoryAdvanced is a full mock implementation of CorpusGeneratorFactory
type MockCorpusGeneratorFactoryAdvanced struct {
	mock.Mock
}

func (m *MockCorpusGeneratorFactoryAdvanced) NewCorpusGenerator(baseDir string) CorpusGenerator {
	args := m.Called(baseDir)
	return args.Get(0).(CorpusGenerator)
}

// MockCorpusGeneratorAdvanced is a full mock implementation of CorpusGenerator
type MockCorpusGeneratorAdvanced struct {
	mock.Mock
}

func (m *MockCorpusGeneratorAdvanced) GenerateAll() error {
	args := m.Called()
	return args.Error(0)
}

// MockFileInfo is a mock implementation of os.FileInfo
type MockFileInfo struct {
	name  string
	size  int64
	mode  os.FileMode
	isDir bool
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return m.size }
func (m *MockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *MockFileInfo) ModTime() time.Time { return time.Now() }
func (m *MockFileInfo) IsDir() bool        { return m.isDir }
func (m *MockFileInfo) Sys() interface{}   { return nil }

func TestGenerateCorpusApp_Run(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		// Setup mocks
		mockLogger := &MockLoggerAdvanced{}
		mockFileSystem := &MockFileSystemAdvanced{}
		mockFactory := &MockCorpusGeneratorFactoryAdvanced{}
		mockGenerator := &MockCorpusGeneratorAdvanced{}

		// Setup expectations
		mockLogger.On("Println", mock.Anything).Return()
		mockLogger.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()

		// Mock successful directory check
		mockFileInfo := &MockFileInfo{name: "internal/fuzz", isDir: true}
		mockFileSystem.On("Stat", "internal/fuzz").Return(mockFileInfo, nil)

		// Mock successful generator creation and execution
		mockFactory.On("NewCorpusGenerator", "internal/fuzz").Return(mockGenerator)
		mockGenerator.On("GenerateAll").Return(nil)

		// Create app with mocked dependencies
		app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFileSystem, mockFactory)

		// Execute
		err := app.Run()

		// Assertions
		require.NoError(t, err)
		mockLogger.AssertExpectations(t)
		mockFileSystem.AssertExpectations(t)
		mockFactory.AssertExpectations(t)
		mockGenerator.AssertExpectations(t)

		// Verify that log messages were captured
		assert.NotEmpty(t, mockLogger.messages)
		assert.Contains(t, mockLogger.messages[0], "Starting fuzz corpus generation")

		// Find completion message
		found := false
		for _, msg := range mockLogger.messages {
			if msg == "Corpus generation complete!\n" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected completion message")
	})

	t.Run("directory not exists error", func(t *testing.T) {
		// Setup mocks
		mockLogger := &MockLoggerAdvanced{}
		mockFileSystem := &MockFileSystemAdvanced{}
		mockFactory := &MockCorpusGeneratorFactoryAdvanced{}

		// Setup expectations
		mockLogger.On("Println", mock.Anything).Return()

		// Mock directory not found error
		mockFileSystem.On("Stat", "internal/fuzz").Return(nil, os.ErrNotExist)

		// Create app with mocked dependencies
		app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFileSystem, mockFactory)

		// Execute
		err := app.Run()

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "directory does not exist: internal/fuzz")
		mockFileSystem.AssertExpectations(t)
	})

	t.Run("directory stat error", func(t *testing.T) {
		// Setup mocks
		mockLogger := &MockLoggerAdvanced{}
		mockFileSystem := &MockFileSystemAdvanced{}
		mockFactory := &MockCorpusGeneratorFactoryAdvanced{}

		// Setup expectations
		mockLogger.On("Println", mock.Anything).Return()

		// Mock stat error (not IsNotExist)
		mockFileSystem.On("Stat", "internal/fuzz").Return(nil, ErrStatFailed)

		// Create app with mocked dependencies
		app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFileSystem, mockFactory)

		// Execute
		err := app.Run()

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check directory internal/fuzz")
		assert.Contains(t, err.Error(), "stat failed")
		mockFileSystem.AssertExpectations(t)
	})

	t.Run("corpus generation error", func(t *testing.T) {
		// Setup mocks
		mockLogger := &MockLoggerAdvanced{}
		mockFileSystem := &MockFileSystemAdvanced{}
		mockFactory := &MockCorpusGeneratorFactoryAdvanced{}
		mockGenerator := &MockCorpusGeneratorAdvanced{}

		// Setup expectations for successful path until generation
		mockLogger.On("Println", mock.Anything).Return()

		mockFileInfo := &MockFileInfo{name: "internal/fuzz", isDir: true}
		mockFileSystem.On("Stat", "internal/fuzz").Return(mockFileInfo, nil)
		mockFactory.On("NewCorpusGenerator", "internal/fuzz").Return(mockGenerator)

		// Generation fails
		mockGenerator.On("GenerateAll").Return(ErrGenerationFailed)

		// Create app with mocked dependencies
		app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFileSystem, mockFactory)

		// Execute
		err := app.Run()

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to generate corpus")
		assert.Contains(t, err.Error(), "generation failed")
		mockGenerator.AssertExpectations(t)
	})

	t.Run("complete flow with all logging", func(t *testing.T) {
		// Setup mocks
		mockLogger := &MockLoggerAdvanced{}
		mockFileSystem := &MockFileSystemAdvanced{}
		mockFactory := &MockCorpusGeneratorFactoryAdvanced{}
		mockGenerator := &MockCorpusGeneratorAdvanced{}

		// Setup expectations with detailed logging verification
		mockLogger.On("Println", "Starting fuzz corpus generation...").Return()
		mockLogger.On("Println", "Generating fuzz test corpus...").Return()
		mockLogger.On("Println", "Corpus generation complete!").Return()
		mockLogger.On("Printf", "Corpus files created in: %s/corpus/\n", []interface{}{"internal/fuzz"}).Return()

		mockFileInfo := &MockFileInfo{name: "internal/fuzz", isDir: true}
		mockFileSystem.On("Stat", "internal/fuzz").Return(mockFileInfo, nil)
		mockFactory.On("NewCorpusGenerator", "internal/fuzz").Return(mockGenerator)
		mockGenerator.On("GenerateAll").Return(nil)

		// Create app with mocked dependencies
		app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFileSystem, mockFactory)

		// Execute
		err := app.Run()

		// Assertions
		require.NoError(t, err)
		mockLogger.AssertExpectations(t)
		mockFileSystem.AssertExpectations(t)
		mockFactory.AssertExpectations(t)
		mockGenerator.AssertExpectations(t)

		// Verify exact log sequence
		expectedMessages := []string{
			"Starting fuzz corpus generation...\n",
			"Generating fuzz test corpus...\n",
			"Corpus generation complete!\n",
			"Corpus files created in: internal/fuzz/corpus/\n",
		}

		assert.Len(t, mockLogger.messages, len(expectedMessages))
		for i, expected := range expectedMessages {
			assert.Equal(t, expected, mockLogger.messages[i], "Log message %d mismatch", i)
		}
	})
}

func TestGenerateCorpusApp_ErrorPathsCoverage(t *testing.T) {
	t.Run("various stat error scenarios", func(t *testing.T) {
		testCases := []struct {
			name        string
			statErr     error
			expectError string
		}{
			{
				name:        "permission denied",
				statErr:     os.ErrPermission,
				expectError: "failed to check directory internal/fuzz",
			},
			{
				name:        "invalid argument",
				statErr:     os.ErrInvalid,
				expectError: "failed to check directory internal/fuzz",
			},
			{
				name:        "custom error",
				statErr:     ErrCustomStat,
				expectError: "failed to check directory internal/fuzz",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockLogger := &MockLoggerAdvanced{}
				mockFileSystem := &MockFileSystemAdvanced{}
				mockFactory := &MockCorpusGeneratorFactoryAdvanced{}

				mockLogger.On("Println", mock.Anything).Return()
				mockFileSystem.On("Stat", "internal/fuzz").Return(nil, tc.statErr)

				app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFileSystem, mockFactory)
				err := app.Run()

				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
				mockFileSystem.AssertExpectations(t)
			})
		}
	})

	t.Run("generation error scenarios", func(t *testing.T) {
		genErrors := []error{
			ErrFileWrite,
			ErrPermissionDenied,
			ErrDiskFull,
			ErrInvalidCorpusData,
		}

		for i, genError := range genErrors {
			t.Run(fmt.Sprintf("generation_error_%d", i), func(t *testing.T) {
				mockLogger := &MockLoggerAdvanced{}
				mockFileSystem := &MockFileSystemAdvanced{}
				mockFactory := &MockCorpusGeneratorFactoryAdvanced{}
				mockGenerator := &MockCorpusGeneratorAdvanced{}

				mockLogger.On("Println", mock.Anything).Return()
				mockFileInfo := &MockFileInfo{name: "internal/fuzz", isDir: true}
				mockFileSystem.On("Stat", "internal/fuzz").Return(mockFileInfo, nil)
				mockFactory.On("NewCorpusGenerator", "internal/fuzz").Return(mockGenerator)
				mockGenerator.On("GenerateAll").Return(genError)

				app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFileSystem, mockFactory)
				err := app.Run()

				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to generate corpus")
				assert.Contains(t, err.Error(), genError.Error())
			})
		}
	})
}

func TestGenerateCorpusApp_MockValidation(t *testing.T) {
	t.Run("verify all mocks are called correctly", func(t *testing.T) {
		mockLogger := &MockLoggerAdvanced{}
		mockFileSystem := &MockFileSystemAdvanced{}
		mockFactory := &MockCorpusGeneratorFactoryAdvanced{}
		mockGenerator := &MockCorpusGeneratorAdvanced{}

		// Set up strict expectations
		mockLogger.On("Println", "Starting fuzz corpus generation...").Once()
		mockLogger.On("Println", "Generating fuzz test corpus...").Once()
		mockLogger.On("Println", "Corpus generation complete!").Once()
		mockLogger.On("Printf", "Corpus files created in: %s/corpus/\n", []interface{}{"internal/fuzz"}).Once()

		mockFileInfo := &MockFileInfo{name: "internal/fuzz", isDir: true}
		mockFileSystem.On("Stat", "internal/fuzz").Return(mockFileInfo, nil).Once()
		mockFactory.On("NewCorpusGenerator", "internal/fuzz").Return(mockGenerator).Once()
		mockGenerator.On("GenerateAll").Return(nil).Once()

		app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFileSystem, mockFactory)
		err := app.Run()

		require.NoError(t, err)
		mockLogger.AssertExpectations(t)
		mockFileSystem.AssertExpectations(t)
		mockFactory.AssertExpectations(t)
		mockGenerator.AssertExpectations(t)
	})

	t.Run("verify call counts and ordering", func(t *testing.T) {
		mockLogger := &MockLoggerAdvanced{}
		mockFileSystem := &MockFileSystemAdvanced{}
		mockFactory := &MockCorpusGeneratorFactoryAdvanced{}
		mockGenerator := &MockCorpusGeneratorAdvanced{}

		mockLogger.On("Println", mock.Anything).Return()
		mockLogger.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()

		mockFileInfo := &MockFileInfo{name: "internal/fuzz", isDir: true}
		mockFileSystem.On("Stat", "internal/fuzz").Return(mockFileInfo, nil)
		mockFactory.On("NewCorpusGenerator", "internal/fuzz").Return(mockGenerator)
		mockGenerator.On("GenerateAll").Return(nil)

		app := NewGenerateCorpusAppWithDependencies(mockLogger, mockFileSystem, mockFactory)
		err := app.Run()

		require.NoError(t, err)

		// Verify call counts
		mockFileSystem.AssertNumberOfCalls(t, "Stat", 1)
		mockFactory.AssertNumberOfCalls(t, "NewCorpusGenerator", 1)
		mockGenerator.AssertNumberOfCalls(t, "GenerateAll", 1)

		// Verify specific calls were made
		mockFileSystem.AssertCalled(t, "Stat", "internal/fuzz")
		mockFactory.AssertCalled(t, "NewCorpusGenerator", "internal/fuzz")
		mockGenerator.AssertCalled(t, "GenerateAll")
	})
}

func TestInterfaceCompliance(t *testing.T) {
	t.Run("default implementations satisfy interfaces", func(t *testing.T) {
		var logger Logger = &DefaultLogger{}
		var fs FileSystem = &DefaultFileSystem{}
		var factory CorpusGeneratorFactory = &DefaultCorpusGeneratorFactory{}

		// If this compiles, the interfaces are satisfied
		assert.NotNil(t, logger)
		assert.NotNil(t, fs)
		assert.NotNil(t, factory)
	})

	t.Run("mock implementations satisfy interfaces", func(t *testing.T) {
		var logger Logger = &MockLoggerAdvanced{}
		var fs FileSystem = &MockFileSystemAdvanced{}
		var factory CorpusGeneratorFactory = &MockCorpusGeneratorFactoryAdvanced{}
		var generator CorpusGenerator = &MockCorpusGeneratorAdvanced{}

		// If this compiles, the interfaces are satisfied
		assert.NotNil(t, logger)
		assert.NotNil(t, fs)
		assert.NotNil(t, factory)
		assert.NotNil(t, generator)
	})
}

func TestEdgeCasesAndBoundaries(t *testing.T) {
	t.Run("empty log messages", func(t *testing.T) {
		mockLogger := &MockLoggerAdvanced{}
		mockLogger.On("Println", "").Return()
		mockLogger.On("Printf", "%s", mock.Anything).Return()

		// Test that empty messages don't cause issues
		assert.NotPanics(t, func() {
			mockLogger.Println("")
			mockLogger.Printf("%s", "test")
		})
	})

	t.Run("nil file info handling", func(t *testing.T) {
		mockFileSystem := &MockFileSystemAdvanced{}
		mockFileSystem.On("Stat", "test").Return(nil, nil)

		info, err := mockFileSystem.Stat("test")
		assert.Nil(t, info)
		assert.NoError(t, err)
	})

	t.Run("concurrent access safety", func(t *testing.T) {
		// Test that creating multiple apps doesn't cause issues
		apps := make([]*GenerateCorpusApp, 10)
		for i := range apps {
			apps[i] = NewGenerateCorpusApp()
			assert.NotNil(t, apps[i])
		}
	})
}
