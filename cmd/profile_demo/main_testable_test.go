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

	"github.com/mrz1836/go-broadcast/internal/profiling"
)

// Test errors
var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrProfilingFailed  = errors.New("profiling failed")
	ErrReportGenFailed  = errors.New("report generation failed")
	ErrStopFailed       = errors.New("stop failed")
)

// MockLogger is a mock implementation of Logger
type MockLogger struct {
	mock.Mock

	messages []string
}

func (m *MockLogger) Println(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	m.messages = append(m.messages, msg)
	m.Called(v...)
}

func (m *MockLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	m.messages = append(m.messages, msg)
	m.Called(format, v)
}

// MockDirectoryManager is a mock implementation of DirectoryManager
type MockDirectoryManager struct {
	mock.Mock
}

func (m *MockDirectoryManager) MkdirAll(path string, perm os.FileMode) error {
	args := m.Called(path, perm)
	return args.Error(0)
}

// MockProfileSuiteFactory is a mock implementation of ProfileSuiteFactory
type MockProfileSuiteFactory struct {
	mock.Mock
}

func (m *MockProfileSuiteFactory) NewProfileSuite(profilesDir string) ProfileSuite {
	args := m.Called(profilesDir)
	return args.Get(0).(ProfileSuite)
}

// MockProfileSuite is a mock implementation of ProfileSuite
type MockProfileSuite struct {
	mock.Mock
}

func (m *MockProfileSuite) Configure(config profiling.ProfileConfig) {
	m.Called(config)
}

func (m *MockProfileSuite) StartProfiling(sessionName string) error {
	args := m.Called(sessionName)
	return args.Error(0)
}

func (m *MockProfileSuite) StopProfiling() error {
	args := m.Called()
	return args.Error(0)
}

// MockTestRunner is a mock implementation of TestRunner
type MockTestRunner struct {
	mock.Mock
}

func (m *MockTestRunner) TestWorkerPool() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockTestRunner) TestTTLCache() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockTestRunner) TestAlgorithmOptimizations() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func (m *MockTestRunner) TestBatchProcessing() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

// MockReportGenerator is a mock implementation of ReportGenerator
type MockReportGenerator struct {
	mock.Mock
}

func (m *MockReportGenerator) GenerateFinalReport(metrics map[string]float64, profilesDir string) error {
	args := m.Called(metrics, profilesDir)
	return args.Error(0)
}

func TestProfileDemoApp_Run(t *testing.T) {
	t.Run("successful execution", func(t *testing.T) {
		// Setup mocks
		mockLogger := &MockLogger{}
		mockDirManager := &MockDirectoryManager{}
		mockProfileSuiteFactory := &MockProfileSuiteFactory{}
		mockProfileSuite := &MockProfileSuite{}
		mockTestRunner := &MockTestRunner{}
		mockReportGenerator := &MockReportGenerator{}

		// Setup expectations
		mockLogger.On("Println", mock.Anything).Return()
		mockLogger.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
		mockDirManager.On("MkdirAll", "./profiles/final_demo", os.FileMode(0o750)).Return(nil)
		mockProfileSuiteFactory.On("NewProfileSuite", "./profiles/final_demo").Return(mockProfileSuite)
		mockProfileSuite.On("Configure", mock.AnythingOfType("profiling.ProfileConfig")).Return()
		mockProfileSuite.On("StartProfiling", "final_optimization_demo").Return(nil)
		mockProfileSuite.On("StopProfiling").Return(nil)

		// Set up test durations
		mockTestRunner.On("TestWorkerPool").Return(time.Duration(100) * time.Millisecond)
		mockTestRunner.On("TestTTLCache").Return(time.Duration(200) * time.Millisecond)
		mockTestRunner.On("TestAlgorithmOptimizations").Return(time.Duration(150) * time.Millisecond)
		mockTestRunner.On("TestBatchProcessing").Return(time.Duration(300) * time.Millisecond)

		mockReportGenerator.On("GenerateFinalReport", mock.AnythingOfType("map[string]float64"), "./profiles/final_demo").Return(nil)

		// Create app with mocked dependencies
		app := NewProfileDemoAppWithDependencies(mockLogger, mockDirManager, mockProfileSuiteFactory, mockTestRunner, mockReportGenerator)

		// Execute
		err := app.Run()

		// Assertions
		require.NoError(t, err)
		mockLogger.AssertExpectations(t)
		mockDirManager.AssertExpectations(t)
		mockProfileSuiteFactory.AssertExpectations(t)
		mockProfileSuite.AssertExpectations(t)
		mockTestRunner.AssertExpectations(t)
		mockReportGenerator.AssertExpectations(t)

		// Verify that log messages were captured
		assert.NotEmpty(t, mockLogger.messages)
		assert.Contains(t, mockLogger.messages[0], "Starting comprehensive profiling demonstration")
	})

	t.Run("directory creation error", func(t *testing.T) {
		// Setup mocks
		mockLogger := &MockLogger{}
		mockDirManager := &MockDirectoryManager{}
		mockProfileSuiteFactory := &MockProfileSuiteFactory{}
		mockTestRunner := &MockTestRunner{}
		mockReportGenerator := &MockReportGenerator{}

		// Setup expectations
		mockLogger.On("Println", mock.Anything).Return()
		mockDirManager.On("MkdirAll", "./profiles/final_demo", os.FileMode(0o750)).Return(ErrPermissionDenied)

		// Create app with mocked dependencies
		app := NewProfileDemoAppWithDependencies(mockLogger, mockDirManager, mockProfileSuiteFactory, mockTestRunner, mockReportGenerator)

		// Execute
		err := app.Run()

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create profiles directory")
		mockDirManager.AssertExpectations(t)
	})

	t.Run("profiling start error", func(t *testing.T) {
		// Setup mocks
		mockLogger := &MockLogger{}
		mockDirManager := &MockDirectoryManager{}
		mockProfileSuiteFactory := &MockProfileSuiteFactory{}
		mockProfileSuite := &MockProfileSuite{}
		mockTestRunner := &MockTestRunner{}
		mockReportGenerator := &MockReportGenerator{}

		// Setup expectations
		mockLogger.On("Println", mock.Anything).Return()
		mockDirManager.On("MkdirAll", "./profiles/final_demo", os.FileMode(0o750)).Return(nil)
		mockProfileSuiteFactory.On("NewProfileSuite", "./profiles/final_demo").Return(mockProfileSuite)
		mockProfileSuite.On("Configure", mock.AnythingOfType("profiling.ProfileConfig")).Return()
		mockProfileSuite.On("StartProfiling", "final_optimization_demo").Return(ErrProfilingFailed)

		// Create app with mocked dependencies
		app := NewProfileDemoAppWithDependencies(mockLogger, mockDirManager, mockProfileSuiteFactory, mockTestRunner, mockReportGenerator)

		// Execute
		err := app.Run()

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to start profiling")
		mockProfileSuite.AssertExpectations(t)
	})

	t.Run("report generation error", func(t *testing.T) {
		// Setup mocks
		mockLogger := &MockLogger{}
		mockDirManager := &MockDirectoryManager{}
		mockProfileSuiteFactory := &MockProfileSuiteFactory{}
		mockProfileSuite := &MockProfileSuite{}
		mockTestRunner := &MockTestRunner{}
		mockReportGenerator := &MockReportGenerator{}

		// Setup expectations for successful path until report generation
		mockLogger.On("Println", mock.Anything).Return()
		mockLogger.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
		mockDirManager.On("MkdirAll", "./profiles/final_demo", os.FileMode(0o750)).Return(nil)
		mockProfileSuiteFactory.On("NewProfileSuite", "./profiles/final_demo").Return(mockProfileSuite)
		mockProfileSuite.On("Configure", mock.AnythingOfType("profiling.ProfileConfig")).Return()
		mockProfileSuite.On("StartProfiling", "final_optimization_demo").Return(nil)
		mockProfileSuite.On("StopProfiling").Return(nil)

		mockTestRunner.On("TestWorkerPool").Return(time.Duration(100) * time.Millisecond)
		mockTestRunner.On("TestTTLCache").Return(time.Duration(200) * time.Millisecond)
		mockTestRunner.On("TestAlgorithmOptimizations").Return(time.Duration(150) * time.Millisecond)
		mockTestRunner.On("TestBatchProcessing").Return(time.Duration(300) * time.Millisecond)

		// Report generation fails
		mockReportGenerator.On("GenerateFinalReport", mock.AnythingOfType("map[string]float64"), "./profiles/final_demo").Return(ErrReportGenFailed)

		// Create app with mocked dependencies
		app := NewProfileDemoAppWithDependencies(mockLogger, mockDirManager, mockProfileSuiteFactory, mockTestRunner, mockReportGenerator)

		// Execute
		err := app.Run()

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to generate report")
		mockReportGenerator.AssertExpectations(t)
	})

	t.Run("profiling stop warning", func(t *testing.T) {
		// Setup mocks
		mockLogger := &MockLogger{}
		mockDirManager := &MockDirectoryManager{}
		mockProfileSuiteFactory := &MockProfileSuiteFactory{}
		mockProfileSuite := &MockProfileSuite{}
		mockTestRunner := &MockTestRunner{}
		mockReportGenerator := &MockReportGenerator{}

		// Setup expectations
		mockLogger.On("Println", mock.Anything).Return()
		mockLogger.On("Printf", mock.AnythingOfType("string"), mock.Anything).Return()
		mockDirManager.On("MkdirAll", "./profiles/final_demo", os.FileMode(0o750)).Return(nil)
		mockProfileSuiteFactory.On("NewProfileSuite", "./profiles/final_demo").Return(mockProfileSuite)
		mockProfileSuite.On("Configure", mock.AnythingOfType("profiling.ProfileConfig")).Return()
		mockProfileSuite.On("StartProfiling", "final_optimization_demo").Return(nil)
		mockProfileSuite.On("StopProfiling").Return(ErrStopFailed) // This should just log a warning

		mockTestRunner.On("TestWorkerPool").Return(time.Duration(100) * time.Millisecond)
		mockTestRunner.On("TestTTLCache").Return(time.Duration(200) * time.Millisecond)
		mockTestRunner.On("TestAlgorithmOptimizations").Return(time.Duration(150) * time.Millisecond)
		mockTestRunner.On("TestBatchProcessing").Return(time.Duration(300) * time.Millisecond)

		mockReportGenerator.On("GenerateFinalReport", mock.AnythingOfType("map[string]float64"), "./profiles/final_demo").Return(nil)

		// Create app with mocked dependencies
		app := NewProfileDemoAppWithDependencies(mockLogger, mockDirManager, mockProfileSuiteFactory, mockTestRunner, mockReportGenerator)

		// Execute - should succeed despite stop profiling error
		err := app.Run()

		// Assertions
		require.NoError(t, err) // Should not fail due to stop profiling error
		mockProfileSuite.AssertExpectations(t)

		// Check that warning was logged
		found := false
		for _, msg := range mockLogger.messages {
			if msg == "Warning: failed to stop profiling: stop failed" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected warning message about failed profiling stop")
	})
}

func TestNewProfileDemoApp(t *testing.T) {
	app := NewProfileDemoApp()

	assert.NotNil(t, app)
	assert.NotNil(t, app.logger)
	assert.NotNil(t, app.dirManager)
	assert.NotNil(t, app.profileSuiteFactory)
	assert.NotNil(t, app.testRunner)
	assert.NotNil(t, app.reportGenerator)
	assert.IsType(t, &DefaultLogger{}, app.logger)
	assert.IsType(t, &DefaultDirectoryManager{}, app.dirManager)
	assert.IsType(t, &DefaultProfileSuiteFactory{}, app.profileSuiteFactory)
	assert.IsType(t, &DefaultTestRunner{}, app.testRunner)
	assert.IsType(t, &DefaultReportGenerator{}, app.reportGenerator)
}

func TestNewProfileDemoAppWithDependencies(t *testing.T) {
	mockLogger := &MockLogger{}
	mockDirManager := &MockDirectoryManager{}
	mockProfileSuiteFactory := &MockProfileSuiteFactory{}
	mockTestRunner := &MockTestRunner{}
	mockReportGenerator := &MockReportGenerator{}

	app := NewProfileDemoAppWithDependencies(mockLogger, mockDirManager, mockProfileSuiteFactory, mockTestRunner, mockReportGenerator)

	assert.NotNil(t, app)
	assert.Equal(t, mockLogger, app.logger)
	assert.Equal(t, mockDirManager, app.dirManager)
	assert.Equal(t, mockProfileSuiteFactory, app.profileSuiteFactory)
	assert.Equal(t, mockTestRunner, app.testRunner)
	assert.Equal(t, mockReportGenerator, app.reportGenerator)
}

func TestDefaultLogger(t *testing.T) {
	logger := &DefaultLogger{}

	// These should not panic
	assert.NotPanics(t, func() {
		logger.Println("test message")
		logger.Printf("formatted %s", "message")
	})
}

func TestDefaultDirectoryManager(t *testing.T) {
	manager := &DefaultDirectoryManager{}

	// Test with a temporary directory
	tmpDir := t.TempDir()
	testPath := tmpDir + "/test/nested/dir"

	err := manager.MkdirAll(testPath, 0o755)
	require.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(testPath)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDefaultTestRunner(t *testing.T) {
	runner := &DefaultTestRunner{}

	// Test that all methods return positive durations and don't panic
	assert.NotPanics(t, func() {
		duration := runner.TestWorkerPool()
		assert.Greater(t, duration, time.Duration(0))
	})

	// We're not testing the other methods in unit tests as they
	// require complex setup and are integration-level functionality
}

func TestDefaultReportGenerator(t *testing.T) {
	generator := &DefaultReportGenerator{}

	// Create temporary directory for test
	tmpDir := t.TempDir()

	metrics := map[string]float64{
		"test_duration_ms": 100.5,
	}

	// This should not panic - the actual report generation may fail
	// due to missing dependencies, but the wrapper should not panic
	assert.NotPanics(t, func() {
		err := generator.GenerateFinalReport(metrics, tmpDir)
		// We don't assert on error here as the implementation may fail
		// due to missing profiling data, which is expected in unit tests
		_ = err
	})
}
