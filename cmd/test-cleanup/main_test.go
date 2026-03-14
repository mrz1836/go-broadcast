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
	ErrInvalidFlag = errors.New("invalid flag")
	ErrDirNotFound = errors.New("directory not found")
)

// MockFlagParser is a mock implementation of FlagParser
type MockFlagParser struct {
	mock.Mock
}

func (m *MockFlagParser) ParseFlags(args []string) (TestCleanupConfig, error) {
	mockArgs := m.Called(args)
	return mockArgs.Get(0).(TestCleanupConfig), mockArgs.Error(1)
}

// MockFileWalker is a mock implementation of FileWalker
type MockFileWalker struct {
	mock.Mock
}

func (m *MockFileWalker) Walk(root string, walkFunc WalkFunc) error {
	args := m.Called(root, walkFunc)
	return args.Error(0)
}

// MockLogger is a mock implementation of Logger
type MockLogger struct {
	mock.Mock

	messages []string
}

func (m *MockLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	m.messages = append(m.messages, msg)
	m.Called(format, v)
}

func (m *MockLogger) Fatalf(format string, v ...interface{}) {
	m.Called(format, v)
	panic(fmt.Sprintf(format, v...))
}

// MockFileRemover is a mock implementation of FileRemover
type MockFileRemover struct {
	mock.Mock
}

func (m *MockFileRemover) Remove(filename string) error {
	args := m.Called(filename)
	return args.Error(0)
}

func (m *MockFileRemover) Stat(filename string) (os.FileInfo, error) {
	args := m.Called(filename)
	return args.Get(0).(os.FileInfo), args.Error(1)
}

// MockFileInfo is a simple implementation of os.FileInfo for testing
type MockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m *MockFileInfo) Name() string       { return m.name }
func (m *MockFileInfo) Size() int64        { return m.size }
func (m *MockFileInfo) Mode() os.FileMode  { return 0o644 }
func (m *MockFileInfo) ModTime() time.Time { return time.Now() }
func (m *MockFileInfo) IsDir() bool        { return m.isDir }
func (m *MockFileInfo) Sys() interface{}   { return nil }

func TestTestCleanupApp_Run(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		// Setup mocks
		mockFlagParser := &MockFlagParser{}
		mockFileWalker := &MockFileWalker{}
		mockLogger := &MockLogger{}
		mockFileRemover := &MockFileRemover{}

		// Setup expectations
		config := TestCleanupConfig{
			RootDir:     ".",
			DryRun:      true,
			Verbose:     true,
			Patterns:    []string{"*.test"},
			ExcludeDirs: []string{".git"},
		}

		mockFlagParser.On("ParseFlags", []string{"test-cleanup", "--dry-run"}).Return(config, nil)
		mockLogger.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
		mockFileWalker.On("Walk", ".", mock.AnythingOfType("main.WalkFunc")).Return(nil)

		// Create app with mocked dependencies
		app := NewTestCleanupAppWithDependencies(mockFlagParser, mockFileWalker, mockLogger, mockFileRemover)

		// Execute
		err := app.Run([]string{"test-cleanup", "--dry-run"})

		// Assertions
		require.NoError(t, err)
		mockFlagParser.AssertExpectations(t)
		mockFileWalker.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
	})

	t.Run("flag parsing error", func(t *testing.T) {
		// Setup mocks
		mockFlagParser := &MockFlagParser{}
		mockFileWalker := &MockFileWalker{}
		mockLogger := &MockLogger{}
		mockFileRemover := &MockFileRemover{}

		// Setup expectations
		mockFlagParser.On("ParseFlags", []string{"test-cleanup", "--invalid"}).Return(TestCleanupConfig{}, ErrInvalidFlag)

		// Create app with mocked dependencies
		app := NewTestCleanupAppWithDependencies(mockFlagParser, mockFileWalker, mockLogger, mockFileRemover)

		// Execute
		err := app.Run([]string{"test-cleanup", "--invalid"})

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse flags")
		mockFlagParser.AssertExpectations(t)
	})

	t.Run("file walking error", func(t *testing.T) {
		// Setup mocks
		mockFlagParser := &MockFlagParser{}
		mockFileWalker := &MockFileWalker{}
		mockLogger := &MockLogger{}
		mockFileRemover := &MockFileRemover{}

		// Setup expectations
		config := TestCleanupConfig{
			RootDir:     "./invalid",
			DryRun:      false,
			Verbose:     false,
			Patterns:    []string{"*.test"},
			ExcludeDirs: []string{},
		}

		mockFlagParser.On("ParseFlags", mock.Anything).Return(config, nil)
		mockFileWalker.On("Walk", "./invalid", mock.AnythingOfType("main.WalkFunc")).Return(ErrDirNotFound)

		// Create app with mocked dependencies
		app := NewTestCleanupAppWithDependencies(mockFlagParser, mockFileWalker, mockLogger, mockFileRemover)

		// Execute
		err := app.Run([]string{"test-cleanup"})

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to walk directory tree")
		mockFlagParser.AssertExpectations(t)
		mockFileWalker.AssertExpectations(t)
	})
}

func TestTestCleanupApp_cleanupTestFiles(t *testing.T) {
	t.Run("dry run mode", func(t *testing.T) {
		// Setup mocks
		mockFileWalker := &MockFileWalker{}
		mockLogger := &MockLogger{}
		mockFileRemover := &MockFileRemover{}

		// Create app
		app := NewTestCleanupAppWithDependencies(nil, mockFileWalker, mockLogger, mockFileRemover)

		// Setup test config
		config := TestCleanupConfig{
			RootDir:     ".",
			DryRun:      true,
			Verbose:     true,
			Patterns:    []string{"*.test"},
			ExcludeDirs: []string{".git"},
		}

		// Setup expectations - simulate finding one test file
		mockFileWalker.On("Walk", ".", mock.AnythingOfType("main.WalkFunc")).Run(func(args mock.Arguments) {
			walkFunc := args.Get(1).(WalkFunc)
			// Simulate finding a test file
			testFile := &MockFileInfo{name: "example.test", size: 1024, isDir: false}
			_ = walkFunc("./example.test", testFile, nil)
		}).Return(nil)

		mockLogger.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()

		// Execute
		err := app.cleanupTestFiles(config)

		// Assertions
		require.NoError(t, err)
		mockFileWalker.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
		// FileRemover should not be called in dry-run mode
		mockFileRemover.AssertNotCalled(t, "Remove")
	})

	t.Run("actual cleanup mode", func(t *testing.T) {
		// Setup mocks
		mockFileWalker := &MockFileWalker{}
		mockLogger := &MockLogger{}
		mockFileRemover := &MockFileRemover{}

		// Create app
		app := NewTestCleanupAppWithDependencies(nil, mockFileWalker, mockLogger, mockFileRemover)

		// Setup test config
		config := TestCleanupConfig{
			RootDir:     ".",
			DryRun:      false,
			Verbose:     false,
			Patterns:    []string{"*.test"},
			ExcludeDirs: []string{".git"},
		}

		// Setup expectations - simulate finding and removing one test file
		mockFileWalker.On("Walk", ".", mock.AnythingOfType("main.WalkFunc")).Run(func(args mock.Arguments) {
			walkFunc := args.Get(1).(WalkFunc)
			// Simulate finding a test file
			testFile := &MockFileInfo{name: "example.test", size: 1024, isDir: false}
			_ = walkFunc("./example.test", testFile, nil)
		}).Return(nil)

		mockFileRemover.On("Remove", "./example.test").Return(nil)
		mockLogger.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()

		// Execute
		err := app.cleanupTestFiles(config)

		// Assertions
		require.NoError(t, err)
		mockFileWalker.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
		mockFileRemover.AssertExpectations(t)
	})
}

func TestNewTestCleanupApp(t *testing.T) {
	app := NewTestCleanupApp()

	assert.NotNil(t, app)
	assert.NotNil(t, app.flagParser)
	assert.NotNil(t, app.fileWalker)
	assert.NotNil(t, app.logger)
	assert.NotNil(t, app.fileRemover)
	assert.IsType(t, &DefaultFlagParser{}, app.flagParser)
	assert.IsType(t, &DefaultFileWalker{}, app.fileWalker)
	assert.IsType(t, &DefaultLogger{}, app.logger)
	assert.IsType(t, &DefaultFileRemover{}, app.fileRemover)
}

func TestNewTestCleanupAppWithDependencies(t *testing.T) {
	mockFlagParser := &MockFlagParser{}
	mockFileWalker := &MockFileWalker{}
	mockLogger := &MockLogger{}
	mockFileRemover := &MockFileRemover{}

	app := NewTestCleanupAppWithDependencies(mockFlagParser, mockFileWalker, mockLogger, mockFileRemover)

	assert.NotNil(t, app)
	assert.Equal(t, mockFlagParser, app.flagParser)
	assert.Equal(t, mockFileWalker, app.fileWalker)
	assert.Equal(t, mockLogger, app.logger)
	assert.Equal(t, mockFileRemover, app.fileRemover)
}

func TestDefaultFlagParser_ParseFlags(t *testing.T) {
	parser := &DefaultFlagParser{}

	t.Run("default values", func(t *testing.T) {
		config, err := parser.ParseFlags([]string{"test-cleanup"})

		require.NoError(t, err)
		assert.Equal(t, ".", config.RootDir)
		assert.False(t, config.DryRun)
		assert.False(t, config.Verbose)
		assert.Equal(t, []string{"*.test", "*.out", "*.prof"}, config.Patterns)
		assert.Equal(t, []string{".git", "vendor", "node_modules"}, config.ExcludeDirs)
	})

	t.Run("custom values", func(t *testing.T) {
		args := []string{
			"test-cleanup",
			"--root=/tmp",
			"--dry-run",
			"--verbose",
			"--patterns=*.log,*.tmp",
			"--exclude-dirs=.git,build",
		}

		config, err := parser.ParseFlags(args)

		require.NoError(t, err)
		assert.Equal(t, "/tmp", config.RootDir)
		assert.True(t, config.DryRun)
		assert.True(t, config.Verbose)
		assert.Equal(t, []string{"*.log", "*.tmp"}, config.Patterns)
		assert.Equal(t, []string{".git", "build"}, config.ExcludeDirs)
	})
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("input_%d", test.input), func(t *testing.T) {
			result := formatBytes(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}
