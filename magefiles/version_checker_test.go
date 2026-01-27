package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test errors for linting compliance (err113)
var (
	errNotFound         = errors.New("not found")
	errRateLimited      = errors.New("rate limited")
	errPermissionDenied = errors.New("permission denied")
	errDiskFull         = errors.New("disk full")
	errFileNotFound     = errors.New("file not found")
)

// MockVersionChecker is a mock implementation of VersionChecker for testing.
type MockVersionChecker struct {
	versions map[string]string // repoURL -> version
	errors   map[string]error  // repoURL -> error
	calls    []string          // Track calls
}

// NewMockVersionChecker creates a new mock version checker.
func NewMockVersionChecker() *MockVersionChecker {
	return &MockVersionChecker{
		versions: make(map[string]string),
		errors:   make(map[string]error),
		calls:    make([]string, 0),
	}
}

// CheckLatestVersion returns the mocked version or error.
func (m *MockVersionChecker) CheckLatestVersion(_ context.Context, repoURL, goModulePath string) (string, error) {
	// Use module path as key for Go proxy tools, otherwise use repo URL
	key := repoURL
	if goModulePath != "" {
		key = goModulePath
	}
	m.calls = append(m.calls, key)
	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if version, ok := m.versions[key]; ok {
		return version, nil
	}
	return "", errNotFound
}

// SetVersion sets the version to return for a repo.
func (m *MockVersionChecker) SetVersion(repoURL, version string) {
	m.versions[repoURL] = version
}

// SetError sets the error to return for a repo.
func (m *MockVersionChecker) SetError(repoURL string, err error) {
	m.errors[repoURL] = err
}

// GetCalls returns the list of calls made.
func (m *MockVersionChecker) GetCalls() []string {
	return m.calls
}

// MockFileUpdater is a mock implementation of FileUpdater for testing.
type MockFileUpdater struct {
	content      []byte
	readError    error
	writeError   error
	backupError  error
	writtenPath  string
	writtenData  []byte
	backedUpPath string
}

// NewMockFileUpdater creates a new mock file updater.
func NewMockFileUpdater() *MockFileUpdater {
	return &MockFileUpdater{}
}

// ReadFile returns the mocked content or error.
func (m *MockFileUpdater) ReadFile(_ string) ([]byte, error) {
	if m.readError != nil {
		return nil, m.readError
	}
	return m.content, nil
}

// WriteFile stores the written data.
func (m *MockFileUpdater) WriteFile(path string, content []byte, _ os.FileMode) error {
	if m.writeError != nil {
		return m.writeError
	}
	m.writtenPath = path
	m.writtenData = content
	return nil
}

// BackupFile records the backup.
func (m *MockFileUpdater) BackupFile(path string) error {
	if m.backupError != nil {
		return m.backupError
	}
	m.backedUpPath = path
	return nil
}

// SetContent sets the content to return on read.
func (m *MockFileUpdater) SetContent(content []byte) {
	m.content = content
}

// GetWrittenData returns the data that was written.
func (m *MockFileUpdater) GetWrittenData() []byte {
	return m.writtenData
}

// MockLogger is a mock implementation of VersionLogger for testing.
type MockLogger struct {
	infoMessages  []string
	errorMessages []string
	warnMessages  []string
}

// NewMockLogger creates a new mock logger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		infoMessages:  make([]string, 0),
		errorMessages: make([]string, 0),
		warnMessages:  make([]string, 0),
	}
}

// Info logs an info message.
func (m *MockLogger) Info(msg string) {
	m.infoMessages = append(m.infoMessages, msg)
}

// Error logs an error message.
func (m *MockLogger) Error(msg string) {
	m.errorMessages = append(m.errorMessages, msg)
}

// Warn logs a warning message.
func (m *MockLogger) Warn(msg string) {
	m.warnMessages = append(m.warnMessages, msg)
}

// GetInfoMessages returns all info messages.
func (m *MockLogger) GetInfoMessages() []string {
	return m.infoMessages
}

func TestGetToolDefinitions(t *testing.T) {
	tools := GetToolDefinitions()

	// Test that all expected tools are present
	expectedTools := []string{
		"go",
		"go-coverage",
		"mage-x",
		"mage",
		"gitleaks",
		"gofumpt",
		"golangci-lint",
		"goreleaser",
		"govulncheck",
		"mockgen",
		"nancy",
		"staticcheck",
		"swag",
		"yamlfmt",
		"go-pre-commit",
		"benchstat",
		"act",
		"actionlint",
		"go-sarif",
	}

	assert.Len(t, tools, len(expectedTools), "should have correct number of tools")

	for _, toolName := range expectedTools {
		tool, ok := tools[toolName]
		require.True(t, ok, "tool %s should exist", toolName)
		assert.NotEmpty(t, tool.EnvVars, "tool %s should have env vars", toolName)
		// Go proxy-based tools (like benchstat) don't have GitHub repo info
		if tool.GoModulePath == "" {
			assert.NotEmpty(t, tool.RepoURL, "tool %s should have repo URL", toolName)
			assert.NotEmpty(t, tool.RepoOwner, "tool %s should have repo owner", toolName)
			assert.NotEmpty(t, tool.RepoName, "tool %s should have repo name", toolName)
		} else {
			assert.NotEmpty(t, tool.GoModulePath, "tool %s should have Go module path", toolName)
		}
	}

	// Test specific tool configurations
	t.Run("gitleaks has multiple env vars", func(t *testing.T) {
		tool := tools["gitleaks"]
		assert.Contains(t, tool.EnvVars, "MAGE_X_GITLEAKS_VERSION")
		assert.Contains(t, tool.EnvVars, "GITLEAKS_VERSION")
		assert.Contains(t, tool.EnvVars, "GO_PRE_COMMIT_GITLEAKS_VERSION")
		assert.Equal(t, "gitleaks", tool.RepoOwner)
		assert.Equal(t, "gitleaks", tool.RepoName)
	})

	t.Run("golangci-lint has multiple env vars", func(t *testing.T) {
		tool := tools["golangci-lint"]
		assert.Contains(t, tool.EnvVars, "MAGE_X_GOLANGCI_LINT_VERSION")
		assert.Contains(t, tool.EnvVars, "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION")
	})
}

func TestVersionUpdateService_ExtractVersions(t *testing.T) {
	t.Run("extracts versions from env vars", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

		content := []byte(`# Comment line
GO_COVERAGE_VERSION=v1.1.15
MAGE_X_VERSION=v1.8.7
MAGE_X_GITLEAKS_VERSION=8.29.1
GITLEAKS_VERSION=8.29.1
NANCY_VERSION=v1.0.52
`)

		tools := GetToolDefinitions()
		versions := service.extractVersions(content, tools)

		assert.Equal(t, "v1.1.15", versions["go-coverage"])
		assert.Equal(t, "v1.8.7", versions["mage-x"])
		assert.Equal(t, "8.29.1", versions["gitleaks"])
		assert.Equal(t, "v1.0.52", versions["nancy"])
	})

	t.Run("keeps first version when env vars diverge", func(t *testing.T) {
		// When multiple env vars for the same tool have different versions,
		// extract the first one found to detect if any needs updating
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

		content := []byte(`# Simulating diverged versions (first one is older)
MAGE_X_GITLEAKS_VERSION=8.29.1
GITLEAKS_VERSION=8.29.1
GO_PRE_COMMIT_GITLEAKS_VERSION=v8.30.0
`)

		tools := GetToolDefinitions()
		versions := service.extractVersions(content, tools)

		// Should keep the first version found (8.29.1), not the last (v8.30.0)
		assert.Equal(t, "8.29.1", versions["gitleaks"])
	})
}

func TestVersionUpdateService_NormalizeVersion(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with v prefix", "v1.2.3", "1.2.3"},
		{"without v prefix", "1.2.3", "1.2.3"},
		{"with multiple digits", "v10.20.30", "10.20.30"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.normalizeVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersionUpdateService_CheckVersions(t *testing.T) {
	t.Run("all up to date", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

		// Mock versions
		checker.SetVersion("https://github.com/mrz1836/go-coverage", "v1.1.15")
		checker.SetVersion("https://github.com/mrz1836/mage-x", "v1.8.7")

		tools := map[string]*ToolInfo{
			"go-coverage": {
				EnvVars:   []string{"GO_COVERAGE_VERSION"},
				RepoURL:   "https://github.com/mrz1836/go-coverage",
				RepoOwner: "mrz1836",
				RepoName:  "go-coverage",
			},
			"mage-x": {
				EnvVars:   []string{"MAGE_X_VERSION"},
				RepoURL:   "https://github.com/mrz1836/mage-x",
				RepoOwner: "mrz1836",
				RepoName:  "mage-x",
			},
		}

		currentVersions := map[string]string{
			"go-coverage": "v1.1.15",
			"mage-x":      "v1.8.7",
		}

		ctx := context.Background()
		results := service.checkVersions(ctx, tools, currentVersions)

		require.Len(t, results, 2)
		assert.Equal(t, "up-to-date", results[0].Status)
		assert.Equal(t, "up-to-date", results[1].Status)
	})

	t.Run("updates available", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

		// Mock versions with updates
		checker.SetVersion("https://github.com/mrz1836/go-coverage", "v1.1.16")
		checker.SetVersion("https://github.com/mrz1836/mage-x", "v1.8.8")

		tools := map[string]*ToolInfo{
			"go-coverage": {
				EnvVars:   []string{"GO_COVERAGE_VERSION"},
				RepoURL:   "https://github.com/mrz1836/go-coverage",
				RepoOwner: "mrz1836",
				RepoName:  "go-coverage",
			},
			"mage-x": {
				EnvVars:   []string{"MAGE_X_VERSION"},
				RepoURL:   "https://github.com/mrz1836/mage-x",
				RepoOwner: "mrz1836",
				RepoName:  "mage-x",
			},
		}

		currentVersions := map[string]string{
			"go-coverage": "v1.1.15",
			"mage-x":      "v1.8.7",
		}

		ctx := context.Background()
		results := service.checkVersions(ctx, tools, currentVersions)

		require.Len(t, results, 2)
		assert.Equal(t, "update-available", results[0].Status)
		assert.Equal(t, "update-available", results[1].Status)
		assert.Equal(t, "v1.1.16", results[0].LatestVersion)
		assert.Equal(t, "v1.8.8", results[1].LatestVersion)
	})

	t.Run("version check errors", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

		// Mock error
		checker.SetError("https://github.com/mrz1836/go-coverage", errRateLimited)

		tools := map[string]*ToolInfo{
			"go-coverage": {
				EnvVars:   []string{"GO_COVERAGE_VERSION"},
				RepoURL:   "https://github.com/mrz1836/go-coverage",
				RepoOwner: "mrz1836",
				RepoName:  "go-coverage",
			},
		}

		currentVersions := map[string]string{
			"go-coverage": "v1.1.15",
		}

		ctx := context.Background()
		results := service.checkVersions(ctx, tools, currentVersions)

		require.Len(t, results, 1)
		assert.Equal(t, "error", results[0].Status)
		assert.Error(t, results[0].Error)
	})

	t.Run("normalizes version comparison", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

		// Mock version with v prefix vs without
		checker.SetVersion("https://github.com/gitleaks/gitleaks", "v8.29.1")

		tools := map[string]*ToolInfo{
			"gitleaks": {
				EnvVars:   []string{"GITLEAKS_VERSION"},
				RepoURL:   "https://github.com/gitleaks/gitleaks",
				RepoOwner: "gitleaks",
				RepoName:  "gitleaks",
			},
		}

		// Current version without v prefix
		currentVersions := map[string]string{
			"gitleaks": "8.29.1",
		}

		ctx := context.Background()
		results := service.checkVersions(ctx, tools, currentVersions)

		require.Len(t, results, 1)
		assert.Equal(t, "up-to-date", results[0].Status)
	})
}

func TestVersionUpdateService_HasUpdates(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

	t.Run("has updates", func(t *testing.T) {
		results := []CheckResult{
			{Status: "up-to-date"},
			{Status: "update-available"},
			{Status: "up-to-date"},
		}
		assert.True(t, service.hasUpdates(results))
	})

	t.Run("no updates", func(t *testing.T) {
		results := []CheckResult{
			{Status: "up-to-date"},
			{Status: "up-to-date"},
			{Status: "error"},
		}
		assert.False(t, service.hasUpdates(results))
	})
}

func TestVersionUpdateService_UpdateFile(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		originalContent := []byte(`# Configuration
GO_COVERAGE_VERSION=v1.1.15
MAGE_X_VERSION=v1.8.7
GITLEAKS_VERSION=8.29.1
MAGE_X_GITLEAKS_VERSION=8.29.1
`)

		updater.SetContent(originalContent)

		results := []CheckResult{
			{
				Tool:           "go-coverage",
				EnvVars:        []string{"GO_COVERAGE_VERSION"},
				CurrentVersion: "v1.1.15",
				LatestVersion:  "v1.1.16",
				Status:         "update-available",
			},
			{
				Tool:           "gitleaks",
				EnvVars:        []string{"GITLEAKS_VERSION", "MAGE_X_GITLEAKS_VERSION"},
				CurrentVersion: "8.29.1",
				LatestVersion:  "8.30.0",
				Status:         "update-available",
			},
			{
				Tool:           "mage-x",
				EnvVars:        []string{"MAGE_X_VERSION"},
				CurrentVersion: "v1.8.7",
				LatestVersion:  "v1.8.7",
				Status:         "up-to-date",
			},
		}

		err := service.updateFile(".github/.env.base", originalContent, results)
		require.NoError(t, err)

		// Verify backup was created
		assert.Equal(t, ".github/.env.base", updater.backedUpPath)

		// Verify file was written
		writtenData := string(updater.GetWrittenData())
		assert.Contains(t, writtenData, "GO_COVERAGE_VERSION=v1.1.16")
		assert.Contains(t, writtenData, "GITLEAKS_VERSION=8.30.0")
		assert.Contains(t, writtenData, "MAGE_X_GITLEAKS_VERSION=8.30.0")
		assert.Contains(t, writtenData, "MAGE_X_VERSION=v1.8.7") // Unchanged
	})

	t.Run("backup failure", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		updater.backupError = errPermissionDenied

		results := []CheckResult{
			{Status: "update-available"},
		}

		err := service.updateFile(".github/.env.base", []byte{}, results)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create backup")
	})

	t.Run("write failure", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		updater.writeError = errDiskFull

		results := []CheckResult{
			{
				EnvVars:        []string{"GO_COVERAGE_VERSION"},
				CurrentVersion: "v1.1.15",
				LatestVersion:  "v1.1.16",
				Status:         "update-available",
			},
		}

		err := service.updateFile(".github/.env.base", []byte("GO_COVERAGE_VERSION=v1.1.15"), results)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write file")
	})

	t.Run("diverged versions are all updated preserving v-prefix format", func(t *testing.T) {
		// Test that all env vars for a tool are updated even when they have different current versions,
		// and that the v-prefix format of each env var is preserved
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		// Simulate diverged versions: different env vars have different current values and formats
		originalContent := []byte(`# Configuration
GITLEAKS_VERSION=8.29.1
MAGE_X_GITLEAKS_VERSION=8.29.1
GO_PRE_COMMIT_GITLEAKS_VERSION=v8.30.0
`)

		updater.SetContent(originalContent)

		results := []CheckResult{
			{
				Tool:           "gitleaks",
				EnvVars:        []string{"GITLEAKS_VERSION", "MAGE_X_GITLEAKS_VERSION", "GO_PRE_COMMIT_GITLEAKS_VERSION"},
				CurrentVersion: "8.29.1",
				LatestVersion:  "v8.31.0", // Latest has v-prefix
				Status:         "update-available",
			},
		}

		err := service.updateFile(".github/.env.base", originalContent, results)
		require.NoError(t, err)

		// Verify ALL env vars were updated, preserving their original v-prefix format
		writtenData := string(updater.GetWrittenData())
		assert.Contains(t, writtenData, "GITLEAKS_VERSION=8.31.0")                // No v (original had no v)
		assert.Contains(t, writtenData, "MAGE_X_GITLEAKS_VERSION=8.31.0")         // No v (original had no v)
		assert.Contains(t, writtenData, "GO_PRE_COMMIT_GITLEAKS_VERSION=v8.31.0") // Has v (original had v)
	})
}

func TestVersionUpdateService_Run_DryRun(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 10*time.Millisecond)

	content := []byte(`GO_COVERAGE_VERSION=v1.1.15
MAGE_X_VERSION=v1.8.7
`)

	updater.SetContent(content)
	checker.SetVersion("https://github.com/mrz1836/go-coverage", "v1.1.16")
	checker.SetVersion("https://github.com/mrz1836/mage-x", "v1.8.8")

	ctx := context.Background()
	err := service.Run(ctx, ".github/.env.base")

	require.NoError(t, err)

	// In dry run mode, no files should be written
	assert.Empty(t, updater.writtenPath)
	assert.Empty(t, updater.backedUpPath)

	// Logger should have been used
	assert.NotEmpty(t, logger.GetInfoMessages())
}

func TestVersionUpdateService_Run_ActualUpdate(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, false, false, 10*time.Millisecond)

	content := []byte(`GO_COVERAGE_VERSION=v1.1.15
MAGE_X_VERSION=v1.8.7
`)

	updater.SetContent(content)
	checker.SetVersion("https://github.com/mrz1836/go-coverage", "v1.1.16")
	checker.SetVersion("https://github.com/mrz1836/mage-x", "v1.8.8")

	ctx := context.Background()
	err := service.Run(ctx, ".github/.env.base")

	require.NoError(t, err)

	// File should be backed up and written
	assert.Equal(t, ".github/.env.base", updater.backedUpPath)
	assert.Equal(t, ".github/.env.base", updater.writtenPath)

	// Verify updates
	writtenData := string(updater.GetWrittenData())
	assert.Contains(t, writtenData, "GO_COVERAGE_VERSION=v1.1.16")
	assert.Contains(t, writtenData, "MAGE_X_VERSION=v1.8.8")
}

func TestVersionUpdateService_Run_ReadError(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

	updater.readError = errFileNotFound

	ctx := context.Background()
	err := service.Run(ctx, ".github/.env.base")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestRunVersionUpdate(t *testing.T) {
	t.Run("dry run mode", func(t *testing.T) {
		// Save original service
		originalService := versionUpdateService
		defer func() {
			setVersionUpdateService(originalService)
			resetVersionUpdateService()
		}()

		// Create mock service
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		mockService := NewVersionUpdateService(checker, updater, logger, true, false, 0)

		// Set up mocks
		updater.SetContent([]byte("GO_COVERAGE_VERSION=v1.1.15\n"))
		checker.SetVersion("https://github.com/mrz1836/go-coverage", "v1.1.15")

		// Inject mock service
		setVersionUpdateService(mockService)

		err := RunVersionUpdate(true, false)
		require.NoError(t, err)
	})
}

func TestRealFileUpdater_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	updater := NewFileUpdater()

	// Create temp file
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.env"
	content := []byte("TEST=value\n")

	// Test write
	err := updater.WriteFile(testFile, content, 0o644)
	require.NoError(t, err)

	// Test read
	readContent, err := updater.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Test backup
	err = updater.BackupFile(testFile)
	require.NoError(t, err)

	// Verify backup exists
	backupContent, err := updater.ReadFile(testFile + ".backup")
	require.NoError(t, err)
	assert.Equal(t, content, backupContent)
}

func TestMockVersionChecker_CallTracking(t *testing.T) {
	mock := NewMockVersionChecker()
	mock.SetVersion("https://github.com/owner/repo1", "v1.0.0")
	mock.SetVersion("https://github.com/owner/repo2", "v2.0.0")

	ctx := context.Background()

	// Make calls
	_, _ = mock.CheckLatestVersion(ctx, "https://github.com/owner/repo1", "")
	_, _ = mock.CheckLatestVersion(ctx, "https://github.com/owner/repo2", "")
	_, _ = mock.CheckLatestVersion(ctx, "https://github.com/owner/repo1", "")

	// Verify calls
	calls := mock.GetCalls()
	require.Len(t, calls, 3)
	assert.Equal(t, "https://github.com/owner/repo1", calls[0])
	assert.Equal(t, "https://github.com/owner/repo2", calls[1])
	assert.Equal(t, "https://github.com/owner/repo1", calls[2])
}

func TestConsoleLogger(t *testing.T) {
	// Just verify it doesn't panic
	logger := NewConsoleLogger()
	require.NotNil(t, logger)

	// These should not panic
	logger.Info("test info")
	logger.Error("test error")
	logger.Warn("test warn")
}

func TestVersionChecker_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Test without gh CLI
	checker := NewVersionChecker(false)
	ctx := context.Background()

	// Test with a known stable repo (GitHub releases)
	version, err := checker.CheckLatestVersion(ctx, "https://github.com/magefile/mage", "")
	if err != nil {
		// Network errors are ok in integration tests
		t.Logf("Network error (expected in some envs): %v", err)
		return
	}

	assert.NotEmpty(t, version)
	t.Logf("Found version: %s", version)
}

func TestVersionChecker_GoProxy_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Test Go proxy API
	checker := NewVersionChecker(false)
	ctx := context.Background()

	// Test with a Go module that uses pseudo-versions
	version, err := checker.CheckLatestVersion(ctx, "", "golang.org/x/perf")
	if err != nil {
		// Network errors are ok in integration tests
		t.Logf("Network error (expected in some envs): %v", err)
		return
	}

	assert.NotEmpty(t, version)
	assert.Contains(t, version, "v0.0.0-", "should be a pseudo-version")
	t.Logf("Found Go proxy version: %s", version)
}

func TestVersionUpdateService_CheckVersions_PinRecommended(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

	// Mock Go proxy response for benchstat
	checker.SetVersion("golang.org/x/perf", "v0.0.0-20251208221838-04cf7a2dca90")

	tools := map[string]*ToolInfo{
		"benchstat": {
			EnvVars:      []string{"MAGE_X_BENCHSTAT_VERSION"},
			GoModulePath: "golang.org/x/perf",
		},
	}

	currentVersions := map[string]string{
		"benchstat": "latest",
	}

	ctx := context.Background()
	results := service.checkVersions(ctx, tools, currentVersions)

	require.Len(t, results, 1)
	assert.Equal(t, "pin-recommended", results[0].Status)
	assert.Equal(t, "v0.0.0-20251208221838-04cf7a2dca90", results[0].LatestVersion)
	assert.Equal(t, "latest", results[0].CurrentVersion)
}

func TestVersionUpdateService_HasUpdates_IncludesPinRecommended(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

	t.Run("has pin-recommended", func(t *testing.T) {
		results := []CheckResult{
			{Status: "up-to-date"},
			{Status: "pin-recommended"},
			{Status: "up-to-date"},
		}
		assert.True(t, service.hasUpdates(results))
	})

	t.Run("only up-to-date and errors", func(t *testing.T) {
		results := []CheckResult{
			{Status: "up-to-date"},
			{Status: "up-to-date"},
			{Status: "error"},
		}
		assert.False(t, service.hasUpdates(results))
	})
}

// Tests for major version upgrade detection

func TestVersionUpdateService_ExtractMajorVersion(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

	tests := []struct {
		name        string
		input       string
		expected    string
		expectValid bool
	}{
		{"standard semver", "1.2.3", "1", true},
		{"with v prefix", "v1.2.3", "1", true},
		{"double digit major", "v10.20.30", "10", true},
		{"with go prefix", "go1.25.5", "1", true},
		{"release candidate", "v2.0.0-rc5", "2", true},
		{"pseudo-version", "v0.0.0-20251208221838-04cf7a2dca90", "0", true},
		{"single number", "5", "5", true},
		{"empty string", "", "", false},
		{"no valid number", "latest", "", false},
		{"non-numeric start", "abc.1.2", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := service.extractMajorVersion(tt.input)
			assert.Equal(t, tt.expectValid, valid, "validity mismatch")
			if tt.expectValid {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestVersionUpdateService_IsMajorUpgrade(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{"v1 to v2", "v1.16.6", "v2.0.0-rc5", true},
		{"v1 to v2 no prefix", "1.16.6", "2.0.0", true},
		{"v0 to v1", "v0.9.2", "v1.0.0", true},
		{"same major different minor", "v1.15.5", "v1.15.6", false},
		{"same major different patch", "v2.8.0", "v2.8.1", false},
		{"downgrade major", "v2.0.0", "v1.0.0", false},
		{"same version", "v1.0.0", "v1.0.0", false},
		{"with go prefix", "go1.25.5", "go2.0.0", true},
		{"go minor update", "go1.25.5", "go1.26.0", false},
		{"pseudo-version same major", "v0.0.0-20251208221838-04cf7a2dca90", "v0.0.0-20260101000000-abc123", false},
		{"invalid current", "latest", "v1.0.0", false},
		{"invalid latest", "v1.0.0", "latest", false},
		{"both invalid", "latest", "stable", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isMajorUpgrade(tt.current, tt.latest)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersionUpdateService_CheckVersions_MajorUpgradeSkipped(t *testing.T) {
	t.Run("major upgrade skipped by default", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		// allowMajorUpgrades = false
		service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

		// Mock versions with major upgrade
		checker.SetVersion("https://github.com/swaggo/swag", "v2.0.0-rc5")

		tools := map[string]*ToolInfo{
			"swag": {
				EnvVars:   []string{"MAGE_X_SWAG_VERSION"},
				RepoURL:   "https://github.com/swaggo/swag",
				RepoOwner: "swaggo",
				RepoName:  "swag",
			},
		}

		currentVersions := map[string]string{
			"swag": "v1.16.6",
		}

		ctx := context.Background()
		results := service.checkVersions(ctx, tools, currentVersions)

		require.Len(t, results, 1)
		assert.Equal(t, "major-skipped", results[0].Status)
		assert.Equal(t, "v1.16.6", results[0].CurrentVersion)
		assert.Equal(t, "v2.0.0-rc5", results[0].LatestVersion)
	})

	t.Run("major upgrade allowed when flag set", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		// allowMajorUpgrades = true
		service := NewVersionUpdateService(checker, updater, logger, true, true, 0)

		// Mock versions with major upgrade
		checker.SetVersion("https://github.com/swaggo/swag", "v2.0.0-rc5")

		tools := map[string]*ToolInfo{
			"swag": {
				EnvVars:   []string{"MAGE_X_SWAG_VERSION"},
				RepoURL:   "https://github.com/swaggo/swag",
				RepoOwner: "swaggo",
				RepoName:  "swag",
			},
		}

		currentVersions := map[string]string{
			"swag": "v1.16.6",
		}

		ctx := context.Background()
		results := service.checkVersions(ctx, tools, currentVersions)

		require.Len(t, results, 1)
		assert.Equal(t, "update-available", results[0].Status)
		assert.Equal(t, "v2.0.0-rc5", results[0].LatestVersion)
	})

	t.Run("minor update not affected by major flag", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		// allowMajorUpgrades = false, but this is a minor update
		service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

		checker.SetVersion("https://github.com/mrz1836/mage-x", "v1.15.6")

		tools := map[string]*ToolInfo{
			"mage-x": {
				EnvVars:   []string{"MAGE_X_VERSION"},
				RepoURL:   "https://github.com/mrz1836/mage-x",
				RepoOwner: "mrz1836",
				RepoName:  "mage-x",
			},
		}

		currentVersions := map[string]string{
			"mage-x": "v1.15.5",
		}

		ctx := context.Background()
		results := service.checkVersions(ctx, tools, currentVersions)

		require.Len(t, results, 1)
		assert.Equal(t, "update-available", results[0].Status)
	})
}

func TestVersionUpdateService_HasUpdates_ExcludesMajorSkipped(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

	t.Run("major-skipped does not count as update", func(t *testing.T) {
		results := []CheckResult{
			{Status: "up-to-date"},
			{Status: "major-skipped"},
			{Status: "up-to-date"},
		}
		assert.False(t, service.hasUpdates(results))
	})

	t.Run("major-skipped with minor update available", func(t *testing.T) {
		results := []CheckResult{
			{Status: "up-to-date"},
			{Status: "major-skipped"},
			{Status: "update-available"},
		}
		assert.True(t, service.hasUpdates(results))
	})
}

func TestVersionUpdateService_UpdateFile_SkipsMajorUpgrades(t *testing.T) {
	t.Run("major-skipped status not updated", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		originalContent := []byte(`# Configuration
MAGE_X_SWAG_VERSION=v1.16.6
MAGE_X_VERSION=v1.15.5
`)

		updater.SetContent(originalContent)

		results := []CheckResult{
			{
				Tool:           "swag",
				EnvVars:        []string{"MAGE_X_SWAG_VERSION"},
				CurrentVersion: "v1.16.6",
				LatestVersion:  "v2.0.0-rc5",
				Status:         "major-skipped", // Should not be updated
			},
			{
				Tool:           "mage-x",
				EnvVars:        []string{"MAGE_X_VERSION"},
				CurrentVersion: "v1.15.5",
				LatestVersion:  "v1.15.6",
				Status:         "update-available", // Should be updated
			},
		}

		err := service.updateFile(".github/.env.base", originalContent, results)
		require.NoError(t, err)

		writtenData := string(updater.GetWrittenData())
		// swag should NOT be updated (major upgrade skipped)
		assert.Contains(t, writtenData, "MAGE_X_SWAG_VERSION=v1.16.6")
		// mage-x SHOULD be updated (minor update)
		assert.Contains(t, writtenData, "MAGE_X_VERSION=v1.15.6")
	})
}

func TestVersionUpdateService_Run_WithMajorUpgradesAllowed(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	// allowMajorUpgrades = true
	service := NewVersionUpdateService(checker, updater, logger, false, true, 10*time.Millisecond)

	content := []byte(`MAGE_X_SWAG_VERSION=v1.16.6
MAGE_X_VERSION=v1.15.5
`)

	updater.SetContent(content)
	// Major upgrade
	checker.SetVersion("https://github.com/swaggo/swag", "v2.0.0")
	// Minor upgrade
	checker.SetVersion("https://github.com/mrz1836/mage-x", "v1.15.6")

	ctx := context.Background()
	err := service.Run(ctx, ".github/.env.base")

	require.NoError(t, err)

	// Both should be updated when allowMajorUpgrades=true
	writtenData := string(updater.GetWrittenData())
	assert.Contains(t, writtenData, "MAGE_X_SWAG_VERSION=v2.0.0")
	assert.Contains(t, writtenData, "MAGE_X_VERSION=v1.15.6")
}
