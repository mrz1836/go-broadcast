// Package suite provides common test suite setup functionality for go-broadcast tests
package suite

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/mrz1836/go-broadcast/internal/transform"
)

// Helper provides common test suite setup functionality
type Helper struct {
	suite.Suite

	TempDir       string
	Logger        *logrus.Entry
	MockGH        *gh.MockClient
	MockTransform *transform.MockChain
	SourceState   *state.SourceState
	TargetConfig  config.TargetConfig
}

// SetupTempDir creates and manages a temporary directory for tests
func (s *Helper) SetupTempDir(prefix string) {
	tempDir, err := os.MkdirTemp("", prefix)
	s.Require().NoError(err)
	s.TempDir = tempDir
}

// CleanupTempDir removes the temporary directory
func (s *Helper) CleanupTempDir() {
	if s.TempDir != "" {
		err := os.RemoveAll(s.TempDir)
		s.Require().NoError(err)
	}
}

// SetupLogger creates a configured test logger
func (s *Helper) SetupLogger(component string) {
	s.Logger = CreateTestLogger(component)
}

// SetupLoggerWithLevel creates a test logger with specific log level
func (s *Helper) SetupLoggerWithLevel(component string, level logrus.Level) {
	logger := logrus.New()
	logger.SetLevel(level)
	s.Logger = logger.WithField("component", component)
}

// SetupMocks initializes common mock objects
func (s *Helper) SetupMocks() {
	s.MockGH = &gh.MockClient{}
	s.MockTransform = &transform.MockChain{}
}

// SetupSourceState creates a default source state for testing
func (s *Helper) SetupSourceState(repo, branch, commit string) {
	s.SourceState = &state.SourceState{
		Repo:         repo,
		Branch:       branch,
		LatestCommit: commit,
		LastChecked:  time.Now(),
	}
}

// SetupTargetConfig creates a basic target configuration
func (s *Helper) SetupTargetConfig(repo string, files []config.FileMapping) {
	s.TargetConfig = config.TargetConfig{
		Repo:  repo,
		Files: files,
	}
}

// SetupStandardSuite performs standard suite setup with all common components
func (s *Helper) SetupStandardSuite(tempDirPrefix, component, sourceRepo, targetRepo string) {
	s.SetupTempDir(tempDirPrefix)
	s.SetupLogger(component)
	s.SetupMocks()
	s.SetupSourceState(sourceRepo, "main", "abc123")
	s.SetupTargetConfig(targetRepo, []config.FileMapping{
		{Src: "file1.txt", Dest: "file1.txt"},
		{Src: "file2.txt", Dest: "file2.txt"},
	})
}

// TestLogger creates a logger suitable for testing
func TestLogger(_ *testing.T, component string) *logrus.Entry {
	return CreateTestLogger(component)
}

// TestLoggerWithLevel creates a logger with specific level for testing
func TestLoggerWithLevel(_ *testing.T, component string, level logrus.Level) *logrus.Entry {
	logger := logrus.New()
	logger.SetLevel(level)
	return logger.WithField("component", component)
}

// TestContext returns a background context for testing
func TestContext() context.Context {
	return context.Background()
}

// CreateTestLogger creates a logger configured for testing with standard settings
func CreateTestLogger(component string) *logrus.Entry {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	return logger.WithField("component", component)
}

// CreateTestLoggerWithConfig creates a logger with custom logging configuration
func CreateTestLoggerWithConfig(component string, logConfig logging.LogConfig) *logrus.Entry {
	logger := logrus.New()

	// Apply log configuration
	switch logConfig.LogLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	if logConfig.LogFormat == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})
	}

	return logger.WithField("component", component)
}

// CreateTempFile creates a temporary file with given content for testing
func CreateTempFile(t *testing.T, dir, pattern, content string) string {
	t.Helper()

	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if content != "" {
		if _, err := file.WriteString(content); err != nil {
			_ = file.Close()
			_ = os.Remove(file.Name())
			t.Fatalf("failed to write content to temp file: %v", err)
		}
	}

	if err := file.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	return file.Name()
}
