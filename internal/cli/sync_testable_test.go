package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/output"
)

// Test errors
var (
	ErrConfigNotFound     = errors.New("config not found")
	ErrValidationFailed   = errors.New("validation failed")
	ErrCreateEngineFailed = errors.New("failed to create engine")
	ErrSyncFailed         = errors.New("sync failed")
)

// MockConfigLoader is a mock implementation of ConfigLoader
type MockConfigLoader struct {
	mock.Mock
}

func (m *MockConfigLoader) LoadConfig(configPath string) (*config.Config, error) {
	args := m.Called(configPath)
	return args.Get(0).(*config.Config), args.Error(1)
}

func (m *MockConfigLoader) ValidateConfig(cfg *config.Config) error {
	args := m.Called(cfg)
	return args.Error(0)
}

// MockSyncEngineFactory is a mock implementation of SyncEngineFactory
type MockSyncEngineFactory struct {
	mock.Mock
}

func (m *MockSyncEngineFactory) CreateSyncEngine(ctx context.Context, cfg *config.Config, flags *Flags, logger *logrus.Logger) (SyncService, error) {
	args := m.Called(ctx, cfg, flags, logger)
	return args.Get(0).(SyncService), args.Error(1)
}

// MockSyncService is a mock implementation of SyncService
type MockSyncService struct {
	mock.Mock
}

func (m *MockSyncService) Sync(ctx context.Context, targets []string) error {
	args := m.Called(ctx, targets)
	return args.Error(0)
}

// MockOutputWriter is a mock implementation of output.Writer
type MockOutputWriter struct {
	mock.Mock

	messages []string
}

func (m *MockOutputWriter) Success(msg string) {
	m.messages = append(m.messages, "SUCCESS: "+msg)
	m.Called(msg)
}

func (m *MockOutputWriter) Successf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockOutputWriter) Info(msg string) {
	m.messages = append(m.messages, "INFO: "+msg)
	m.Called(msg)
}

func (m *MockOutputWriter) Infof(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockOutputWriter) Warn(msg string) {
	m.messages = append(m.messages, "WARN: "+msg)
	m.Called(msg)
}

func (m *MockOutputWriter) Warnf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockOutputWriter) Error(msg string) {
	m.messages = append(m.messages, "ERROR: "+msg)
	m.Called(msg)
}

func (m *MockOutputWriter) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockOutputWriter) Plain(msg string) {
	m.messages = append(m.messages, "PLAIN: "+msg)
	m.Called(msg)
}

func (m *MockOutputWriter) Plainf(format string, args ...interface{}) {
	m.Called(format, args)
}

func TestSyncCommand_ExecuteSync(t *testing.T) {
	t.Run("successful sync", func(t *testing.T) {
		// Setup mocks
		mockConfigLoader := &MockConfigLoader{}
		mockSyncEngineFactory := &MockSyncEngineFactory{}
		mockSyncService := &MockSyncService{}
		mockOutputWriter := &MockOutputWriter{}

		// Setup test config
		testConfig := &config.Config{
			Groups: []config.Group{{
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{
					Repo: "org/target",
				}},
			}},
		}

		flags := &Flags{
			ConfigFile: "test-config.yaml",
			DryRun:     false,
		}

		// Setup expectations
		mockConfigLoader.On("LoadConfig", "test-config.yaml").Return(testConfig, nil)
		mockConfigLoader.On("ValidateConfig", testConfig).Return(nil)
		mockSyncEngineFactory.On("CreateSyncEngine", mock.Anything, testConfig, flags, mock.AnythingOfType("*logrus.Logger")).Return(mockSyncService, nil)
		mockSyncService.On("Sync", mock.Anything, []string{"org/target1"}).Return(nil)
		mockOutputWriter.On("Success", "Sync completed successfully").Return()

		// Create command with mocks
		cmd := NewSyncCommandWithDependencies(mockConfigLoader, mockSyncEngineFactory, mockOutputWriter)

		// Execute sync
		err := cmd.ExecuteSync(context.Background(), flags, []string{"org/target1"})

		// Assertions
		require.NoError(t, err)
		mockConfigLoader.AssertExpectations(t)
		mockSyncEngineFactory.AssertExpectations(t)
		mockSyncService.AssertExpectations(t)
		mockOutputWriter.AssertExpectations(t)
	})

	t.Run("config loading error", func(t *testing.T) {
		// Setup mocks
		mockConfigLoader := &MockConfigLoader{}
		mockSyncEngineFactory := &MockSyncEngineFactory{}
		mockOutputWriter := &MockOutputWriter{}

		flags := &Flags{
			ConfigFile: "nonexistent-config.yaml",
		}

		// Setup expectations
		mockConfigLoader.On("LoadConfig", "nonexistent-config.yaml").Return((*config.Config)(nil), ErrConfigNotFound)

		// Create command with mocks
		cmd := NewSyncCommandWithDependencies(mockConfigLoader, mockSyncEngineFactory, mockOutputWriter)

		// Execute sync
		err := cmd.ExecuteSync(context.Background(), flags, []string{})

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
		mockConfigLoader.AssertExpectations(t)
	})

	t.Run("config validation error", func(t *testing.T) {
		// Setup mocks
		mockConfigLoader := &MockConfigLoader{}
		mockSyncEngineFactory := &MockSyncEngineFactory{}
		mockOutputWriter := &MockOutputWriter{}

		testConfig := &config.Config{}
		flags := &Flags{
			ConfigFile: "invalid-config.yaml",
		}

		// Setup expectations
		mockConfigLoader.On("LoadConfig", "invalid-config.yaml").Return(testConfig, nil)
		mockConfigLoader.On("ValidateConfig", testConfig).Return(ErrValidationFailed)

		// Create command with mocks
		cmd := NewSyncCommandWithDependencies(mockConfigLoader, mockSyncEngineFactory, mockOutputWriter)

		// Execute sync
		err := cmd.ExecuteSync(context.Background(), flags, []string{})

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid configuration")
		mockConfigLoader.AssertExpectations(t)
	})

	t.Run("dry run mode", func(t *testing.T) {
		// Setup mocks
		mockConfigLoader := &MockConfigLoader{}
		mockSyncEngineFactory := &MockSyncEngineFactory{}
		mockSyncService := &MockSyncService{}
		mockOutputWriter := &MockOutputWriter{}

		testConfig := &config.Config{
			Groups: []config.Group{{
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{
					Repo: "org/target",
				}},
			}},
		}

		flags := &Flags{
			ConfigFile: "test-config.yaml",
			DryRun:     true, // Enable dry-run mode
		}

		// Setup expectations
		mockConfigLoader.On("LoadConfig", "test-config.yaml").Return(testConfig, nil)
		mockConfigLoader.On("ValidateConfig", testConfig).Return(nil)
		mockOutputWriter.On("Warn", "DRY-RUN MODE: No changes will be made to repositories").Return()
		mockSyncEngineFactory.On("CreateSyncEngine", mock.Anything, testConfig, flags, mock.AnythingOfType("*logrus.Logger")).Return(mockSyncService, nil)
		mockSyncService.On("Sync", mock.Anything, []string{}).Return(nil)
		mockOutputWriter.On("Success", "Sync completed successfully").Return()

		// Create command with mocks
		cmd := NewSyncCommandWithDependencies(mockConfigLoader, mockSyncEngineFactory, mockOutputWriter)

		// Execute sync
		err := cmd.ExecuteSync(context.Background(), flags, []string{})

		// Assertions
		require.NoError(t, err)
		mockOutputWriter.AssertExpectations(t) // Verify dry-run warning was shown
	})

	t.Run("sync engine creation error", func(t *testing.T) {
		// Setup mocks
		mockConfigLoader := &MockConfigLoader{}
		mockSyncEngineFactory := &MockSyncEngineFactory{}
		mockOutputWriter := &MockOutputWriter{}

		testConfig := &config.Config{}
		flags := &Flags{
			ConfigFile: "test-config.yaml",
		}

		// Setup expectations
		mockConfigLoader.On("LoadConfig", "test-config.yaml").Return(testConfig, nil)
		mockConfigLoader.On("ValidateConfig", testConfig).Return(nil)
		mockSyncEngineFactory.On("CreateSyncEngine", mock.Anything, testConfig, flags, mock.AnythingOfType("*logrus.Logger")).Return((*MockSyncService)(nil), ErrCreateEngineFailed)

		// Create command with mocks
		cmd := NewSyncCommandWithDependencies(mockConfigLoader, mockSyncEngineFactory, mockOutputWriter)

		// Execute sync
		err := cmd.ExecuteSync(context.Background(), flags, []string{})

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize sync engine")
		mockSyncEngineFactory.AssertExpectations(t)
	})

	t.Run("sync execution error", func(t *testing.T) {
		// Setup mocks
		mockConfigLoader := &MockConfigLoader{}
		mockSyncEngineFactory := &MockSyncEngineFactory{}
		mockSyncService := &MockSyncService{}
		mockOutputWriter := &MockOutputWriter{}

		testConfig := &config.Config{}
		flags := &Flags{
			ConfigFile: "test-config.yaml",
		}

		// Setup expectations
		mockConfigLoader.On("LoadConfig", "test-config.yaml").Return(testConfig, nil)
		mockConfigLoader.On("ValidateConfig", testConfig).Return(nil)
		mockSyncEngineFactory.On("CreateSyncEngine", mock.Anything, testConfig, flags, mock.AnythingOfType("*logrus.Logger")).Return(mockSyncService, nil)
		mockSyncService.On("Sync", mock.Anything, []string{}).Return(ErrSyncFailed)

		// Create command with mocks
		cmd := NewSyncCommandWithDependencies(mockConfigLoader, mockSyncEngineFactory, mockOutputWriter)

		// Execute sync
		err := cmd.ExecuteSync(context.Background(), flags, []string{})

		// Assertions
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sync failed")
		mockSyncService.AssertExpectations(t)
	})
}

func TestNewSyncCommand(t *testing.T) {
	cmd := NewSyncCommand()

	assert.NotNil(t, cmd)
	assert.NotNil(t, cmd.configLoader)
	assert.NotNil(t, cmd.syncEngineFactory)
	assert.NotNil(t, cmd.outputWriter)
	assert.IsType(t, &DefaultConfigLoader{}, cmd.configLoader)
	assert.IsType(t, &DefaultSyncEngineFactory{}, cmd.syncEngineFactory)
	assert.IsType(t, &output.ColoredWriter{}, cmd.outputWriter)
}

func TestNewSyncCommandWithDependencies(t *testing.T) {
	mockConfigLoader := &MockConfigLoader{}
	mockSyncEngineFactory := &MockSyncEngineFactory{}
	mockOutputWriter := &MockOutputWriter{}

	cmd := NewSyncCommandWithDependencies(mockConfigLoader, mockSyncEngineFactory, mockOutputWriter)

	assert.NotNil(t, cmd)
	assert.Equal(t, mockConfigLoader, cmd.configLoader)
	assert.Equal(t, mockSyncEngineFactory, cmd.syncEngineFactory)
	assert.Equal(t, mockOutputWriter, cmd.outputWriter)
}

func TestDefaultConfigLoader(t *testing.T) {
	t.Run("LoadConfig - file not found", func(t *testing.T) {
		loader := &DefaultConfigLoader{}

		cfg, err := loader.LoadConfig("/nonexistent/config.yaml")

		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "configuration file not found")
	})

	t.Run("ValidateConfig", func(t *testing.T) {
		loader := &DefaultConfigLoader{}

		// Create a minimal valid config
		cfg := &config.Config{
			Version: 1, // Add version to make it valid
			Groups: []config.Group{{
				Name: "test-group",   // Add required name field
				ID:   "test-group-1", // Add required ID field
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{
					Repo: "org/target",
					Files: []config.FileMapping{{
						Src:  "README.md",
						Dest: "README.md",
					}},
				}},
			}},
		}

		err := loader.ValidateConfig(cfg)

		// This should not error for a valid config
		assert.NoError(t, err)
	})
}

func TestDefaultSyncEngineFactory(t *testing.T) {
	t.Run("CreateSyncEngine", func(t *testing.T) {
		factory := &DefaultSyncEngineFactory{}

		cfg := &config.Config{
			Groups: []config.Group{{
				Source: config.SourceConfig{
					Repo:   "org/source",
					Branch: "main",
				},
				Targets: []config.TargetConfig{{
					Repo: "org/target",
				}},
			}},
		}

		flags := &Flags{}
		logger := logrus.New()

		// This may succeed or fail depending on system configuration
		// but should not panic
		assert.NotPanics(t, func() {
			_, _ = factory.CreateSyncEngine(context.Background(), cfg, flags, logger)
		})
	})
}
