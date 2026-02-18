// Package suite provides common test suite setup functionality for go-broadcast tests
package suite

import (
	"context"
	"fmt"
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

// SetupTempDir creates and manages a temporary directory for tests.
// Caller must ensure CleanupTempDir is called, typically in TearDownSuite.
// For automatic cleanup even on panic, use SetupTempDirWithCleanup instead.
func (s *Helper) SetupTempDir(prefix string) {
	tempDir, err := os.MkdirTemp("", prefix)
	s.Require().NoError(err)
	s.TempDir = tempDir
}

// SetupTempDirWithCleanup creates a temporary directory and registers automatic cleanup.
// This ensures the directory is removed even if the test panics.
func (s *Helper) SetupTempDirWithCleanup(prefix string) {
	s.SetupTempDir(prefix)
	s.T().Cleanup(func() {
		s.CleanupTempDir()
	})
}

// CleanupTempDir removes the temporary directory.
// Safe to call multiple times; subsequent calls are no-ops.
func (s *Helper) CleanupTempDir() {
	if s.TempDir != "" {
		path := s.TempDir
		s.TempDir = "" // Clear first to prevent double-cleanup attempts
		err := os.RemoveAll(path)
		s.Require().NoError(err, "failed to cleanup temp dir: %s", path)
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

// defaultTestTimeout is the default timeout for test contexts to prevent hanging tests.
const defaultTestTimeout = 30 * time.Second

// TestContext returns a context with a default timeout for testing.
// The default timeout prevents tests from hanging indefinitely.
// For proper resource management, use TestContextWithCancel instead.
func TestContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	// Schedule cancel to run when context deadline is reached
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return ctx
}

// TestContextWithCancel returns a context with default timeout and its cancel function.
// Caller should defer cancel() to properly release resources.
func TestContextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTestTimeout)
}

// TestContextWithTimeout returns a context with a custom timeout for testing.
// For proper resource management, use TestContextWithTimeoutAndCancel instead.
func TestContextWithTimeout(timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// Schedule cancel to run when context deadline is reached
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return ctx
}

// TestContextWithTimeoutAndCancel returns a context with custom timeout and its cancel function.
// Caller should defer cancel() to properly release resources.
func TestContextWithTimeoutAndCancel(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
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

// CreateTempFileE creates a temporary file with given content, returning error on failure.
// This is the error-returning variant for use in test helpers that need error handling.
func CreateTempFileE(t *testing.T, dir, pattern, content string) (string, error) {
	t.Helper()

	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	name := file.Name()

	if content != "" {
		if _, err := file.WriteString(content); err != nil {
			_ = file.Close()
			_ = os.Remove(name) //nolint:gosec // G703: path from os.CreateTemp or trusted source, not user input
			return "", fmt.Errorf("write to temp file %s: %w", name, err)
		}
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(name) //nolint:gosec // G703: path from os.CreateTemp or trusted source, not user input
		return "", fmt.Errorf("close temp file %s: %w", name, err)
	}

	return name, nil
}

// CreateTempFile creates a temporary file with given content for testing.
// Fails the test on error. For error handling, use CreateTempFileE instead.
func CreateTempFile(t *testing.T, dir, pattern, content string) string {
	t.Helper()

	name, err := CreateTempFileE(t, dir, pattern, content)
	if err != nil {
		t.Fatalf("CreateTempFile: %v", err)
	}

	return name
}
