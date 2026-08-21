package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
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

// goRequirement is a mocked go.mod 'go' directive (major.minor) for a module version.
type goRequirement struct {
	major int
	minor int
	err   error
}

// MockVersionChecker is a mock implementation of VersionChecker for testing.
type MockVersionChecker struct {
	versions       map[string]string        // repoURL -> version
	errors         map[string]error         // repoURL -> error
	calls          []string                 // Track calls
	goRequirements map[string]goRequirement // "module@version" -> go.mod 'go' directive
	goReqCalls     []string                 // Track CheckModuleGoRequirement calls ("module@version")
}

// NewMockVersionChecker creates a new mock version checker.
func NewMockVersionChecker() *MockVersionChecker {
	return &MockVersionChecker{
		versions:       make(map[string]string),
		errors:         make(map[string]error),
		calls:          make([]string, 0),
		goRequirements: make(map[string]goRequirement),
		goReqCalls:     make([]string, 0),
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

// CheckModuleGoRequirement returns the mocked go.mod 'go' directive for a module version.
// Unset versions default to (0, 0, nil), i.e. unconstrained/compatible.
func (m *MockVersionChecker) CheckModuleGoRequirement(_ context.Context, goModulePath, version string) (int, int, error) {
	key := goModulePath + "@" + version
	m.goReqCalls = append(m.goReqCalls, key)
	if req, ok := m.goRequirements[key]; ok {
		return req.major, req.minor, req.err
	}
	return 0, 0, nil
}

// SetVersion sets the version to return for a repo.
func (m *MockVersionChecker) SetVersion(repoURL, version string) {
	m.versions[repoURL] = version
}

// SetGoRequirement sets the mocked go.mod 'go' directive (major.minor) for a module version.
func (m *MockVersionChecker) SetGoRequirement(goModulePath, version string, major, minor int) {
	m.goRequirements[goModulePath+"@"+version] = goRequirement{major: major, minor: minor}
}

// SetGoRequirementError sets an error to return when resolving a module version's go.mod.
func (m *MockVersionChecker) SetGoRequirementError(goModulePath, version string, err error) {
	m.goRequirements[goModulePath+"@"+version] = goRequirement{err: err}
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
	contents      map[string][]byte // path -> content (multi-file support)
	readError     error             // global read error (applies to all reads)
	writeError    error             // global write error (applies to all writes)
	backupError   error             // global backup error
	writtenFiles  map[string][]byte // path -> written data (tracks all writes)
	backedUpPaths []string          // list of backed up paths
}

// NewMockFileUpdater creates a new mock file updater.
func NewMockFileUpdater() *MockFileUpdater {
	return &MockFileUpdater{
		contents:      make(map[string][]byte),
		writtenFiles:  make(map[string][]byte),
		backedUpPaths: make([]string, 0),
	}
}

// ReadFile returns the mocked content or error.
func (m *MockFileUpdater) ReadFile(path string) ([]byte, error) {
	if m.readError != nil {
		return nil, m.readError
	}
	if content, ok := m.contents[path]; ok {
		return content, nil
	}
	// Return empty content for unknown paths (no version vars will be found)
	return []byte{}, nil
}

// WriteFile stores the written data.
func (m *MockFileUpdater) WriteFile(path string, content []byte, _ os.FileMode) error {
	if m.writeError != nil {
		return m.writeError
	}
	m.writtenFiles[path] = content
	return nil
}

// BackupFile records the backup.
func (m *MockFileUpdater) BackupFile(path string) error {
	if m.backupError != nil {
		return m.backupError
	}
	m.backedUpPaths = append(m.backedUpPaths, path)
	return nil
}

// SetContent sets the content to return on read for a specific file path.
func (m *MockFileUpdater) SetContent(path string, content []byte) {
	m.contents[path] = content
}

// GetWrittenData returns the data that was written for a specific file path.
func (m *MockFileUpdater) GetWrittenData(path string) []byte {
	return m.writtenFiles[path]
}

// GetAllWrittenPaths returns all paths that were written to.
func (m *MockFileUpdater) GetAllWrittenPaths() []string {
	paths := make([]string, 0, len(m.writtenFiles))
	for p := range m.writtenFiles {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// WasBackedUp returns true if the given path was backed up.
func (m *MockFileUpdater) WasBackedUp(path string) bool {
	for _, p := range m.backedUpPaths {
		if p == path {
			return true
		}
	}
	return false
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
		"osv-scanner",
		"staticcheck",
		"swag",
		"yamlfmt",
		"go-pre-commit",
		"benchstat",
		"benchstat-go125",
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

	t.Run("osv-scanner has multiple env vars", func(t *testing.T) {
		tool := tools["osv-scanner"]
		assert.Contains(t, tool.EnvVars, "MAGE_X_OSV_SCANNER_VERSION")
		assert.Contains(t, tool.EnvVars, "OSV_SCANNER_VERSION")
		assert.Equal(t, "google", tool.RepoOwner)
		assert.Equal(t, "osv-scanner", tool.RepoName)
	})

	// The version source must match how each tool is installed: go-install tools
	// resolve versions via the Go module proxy (git tags), so they carry a
	// GoModulePath; binary-download and CalVer tools resolve via GitHub Releases and
	// must leave GoModulePath empty. Regressing these silently drifts "latest" away
	// from what the pinned installer actually resolves (see govulncheck/swag).
	t.Run("go-install tools resolve via Go module proxy (git tags)", func(t *testing.T) {
		proxyTools := map[string]string{
			"go-coverage": "github.com/mrz1836/go-coverage",
			"gofumpt":     "mvdan.cc/gofumpt",
			// govulncheck: golang.org/x/vuln is tagged to v1.7.0 while its newest
			// GitHub Release is a stale v1.1.4 — the proxy is the only correct source.
			"govulncheck": "golang.org/x/vuln",
			"mockgen":     "go.uber.org/mock",
			// nancy/osv-scanner are /v2 modules installed via `go install .../v2/...`.
			"nancy":       "github.com/sonatype-nexus-community/nancy/v2",
			"osv-scanner": "github.com/google/osv-scanner/v2",
			// swag's "latest" GitHub Release is a pre-release (v2.0.0-rc5); the proxy
			// correctly resolves to the stable v1.16.6 that `go install @latest` uses.
			"swag":    "github.com/swaggo/swag",
			"yamlfmt": "github.com/google/yamlfmt",
			"mage":    "github.com/magefile/mage",
			// benchstat is split into two proxy pins (latest + Go 1.25-held); see
			// GetToolDefinitions. Both resolve golang.org/x/perf via the proxy.
			"benchstat":       "golang.org/x/perf",
			"benchstat-go125": "golang.org/x/perf",
		}
		for name, wantModule := range proxyTools {
			assert.Equal(t, wantModule, tools[name].GoModulePath, "tool %s must resolve via the Go module proxy", name)
		}
	})

	t.Run("binary-download and CalVer tools resolve via GitHub Releases", func(t *testing.T) {
		// staticcheck ships CalVer tags (2026.1); its module semver (v0.7.0) diverges,
		// so it must use GitHub Releases, not the proxy. The rest are release binaries.
		releaseTools := []string{
			"staticcheck", "mage-x", "go-pre-commit", "gitleaks",
			"golangci-lint", "goreleaser", "act", "actionlint", "go-sarif",
		}
		for _, name := range releaseTools {
			assert.Empty(t, tools[name].GoModulePath, "tool %s must resolve via GitHub Releases", name)
			assert.NotEmpty(t, tools[name].RepoURL, "tool %s should have a GitHub repo URL", name)
		}
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
OSV_SCANNER_VERSION=v2.4.0
`)

		tools := GetToolDefinitions()
		versions := service.extractVersions(content, tools)

		assert.Equal(t, "v1.1.15", versions["go-coverage"])
		assert.Equal(t, "v1.8.7", versions["mage-x"])
		assert.Equal(t, "8.29.1", versions["gitleaks"])
		assert.Equal(t, "v1.0.52", versions["nancy"])
		assert.Equal(t, "v2.4.0", versions["osv-scanner"])
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

func TestVersionUpdateService_UpdateFiles(t *testing.T) {
	t.Run("successful update across multiple files", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		coverageContent := []byte(`# Coverage config
GO_COVERAGE_VERSION=v1.1.15
`)
		securityContent := []byte(`# Security config
GITLEAKS_VERSION=8.29.1
`)
		mageXContent := []byte(`# Mage-X config
MAGE_X_VERSION=v1.8.7
MAGE_X_GITLEAKS_VERSION=8.29.1
`)

		fileContents := map[string][]byte{
			".github/env/10-coverage.env": coverageContent,
			".github/env/10-security.env": securityContent,
			".github/env/10-mage-x.env":   mageXContent,
		}

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

		err := service.updateFiles(fileContents, results)
		require.NoError(t, err)

		// Verify coverage file was updated
		assert.True(t, updater.WasBackedUp(".github/env/10-coverage.env"))
		writtenCoverage := string(updater.GetWrittenData(".github/env/10-coverage.env"))
		assert.Contains(t, writtenCoverage, "GO_COVERAGE_VERSION=v1.1.16")

		// Verify security file was updated
		assert.True(t, updater.WasBackedUp(".github/env/10-security.env"))
		writtenSecurity := string(updater.GetWrittenData(".github/env/10-security.env"))
		assert.Contains(t, writtenSecurity, "GITLEAKS_VERSION=8.30.0")

		// Verify mage-x file was updated (gitleaks version changed)
		assert.True(t, updater.WasBackedUp(".github/env/10-mage-x.env"))
		writtenMageX := string(updater.GetWrittenData(".github/env/10-mage-x.env"))
		assert.Contains(t, writtenMageX, "MAGE_X_GITLEAKS_VERSION=8.30.0")
		assert.Contains(t, writtenMageX, "MAGE_X_VERSION=v1.8.7") // Unchanged
	})

	t.Run("backup failure", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		updater.backupError = errPermissionDenied

		fileContents := map[string][]byte{
			".github/env/10-coverage.env": []byte("GO_COVERAGE_VERSION=v1.1.15\n"),
		}

		results := []CheckResult{
			{
				EnvVars:        []string{"GO_COVERAGE_VERSION"},
				CurrentVersion: "v1.1.15",
				LatestVersion:  "v1.1.16",
				Status:         "update-available",
			},
		}

		err := service.updateFiles(fileContents, results)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create backup")
	})

	t.Run("write failure", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		updater.writeError = errDiskFull

		fileContents := map[string][]byte{
			".github/env/10-coverage.env": []byte("GO_COVERAGE_VERSION=v1.1.15\n"),
		}

		results := []CheckResult{
			{
				EnvVars:        []string{"GO_COVERAGE_VERSION"},
				CurrentVersion: "v1.1.15",
				LatestVersion:  "v1.1.16",
				Status:         "update-available",
			},
		}

		err := service.updateFiles(fileContents, results)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write file")
	})

	t.Run("diverged versions are all updated preserving v-prefix format", func(t *testing.T) {
		// Test that all env vars for a tool are updated even when they have different current versions,
		// and that the v-prefix format of each env var is preserved across multiple files
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		securityContent := []byte(`# Security config
GITLEAKS_VERSION=8.29.1
`)
		mageXContent := []byte(`# Mage-X config
MAGE_X_GITLEAKS_VERSION=8.29.1
`)
		preCommitContent := []byte(`# Pre-commit config
GO_PRE_COMMIT_GITLEAKS_VERSION=v8.30.0
`)

		fileContents := map[string][]byte{
			".github/env/10-security.env":   securityContent,
			".github/env/10-mage-x.env":     mageXContent,
			".github/env/10-pre-commit.env": preCommitContent,
		}

		results := []CheckResult{
			{
				Tool:           "gitleaks",
				EnvVars:        []string{"GITLEAKS_VERSION", "MAGE_X_GITLEAKS_VERSION", "GO_PRE_COMMIT_GITLEAKS_VERSION"},
				CurrentVersion: "8.29.1",
				LatestVersion:  "v8.31.0", // Latest has v-prefix
				Status:         "update-available",
			},
		}

		err := service.updateFiles(fileContents, results)
		require.NoError(t, err)

		// Verify ALL env vars were updated, preserving their original v-prefix format
		writtenSecurity := string(updater.GetWrittenData(".github/env/10-security.env"))
		assert.Contains(t, writtenSecurity, "GITLEAKS_VERSION=8.31.0") // No v (original had no v)

		writtenMageX := string(updater.GetWrittenData(".github/env/10-mage-x.env"))
		assert.Contains(t, writtenMageX, "MAGE_X_GITLEAKS_VERSION=8.31.0") // No v (original had no v)

		writtenPreCommit := string(updater.GetWrittenData(".github/env/10-pre-commit.env"))
		assert.Contains(t, writtenPreCommit, "GO_PRE_COMMIT_GITLEAKS_VERSION=v8.31.0") // Has v (original had v)
	})

	t.Run("only changed files are backed up and written", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		coverageContent := []byte(`# Coverage config
GO_COVERAGE_VERSION=v1.1.15
`)
		redisContent := []byte(`# Redis config
ENABLE_REDIS_SERVICE=false
REDIS_VERSION=7-alpine
`)
		workflowContent := []byte(`# Workflow config
STALE_DAYS_BEFORE_STALE=60
`)

		fileContents := map[string][]byte{
			".github/env/10-coverage.env":  coverageContent,
			".github/env/20-redis.env":     redisContent,
			".github/env/20-workflows.env": workflowContent,
		}

		results := []CheckResult{
			{
				Tool:           "go-coverage",
				EnvVars:        []string{"GO_COVERAGE_VERSION"},
				CurrentVersion: "v1.1.15",
				LatestVersion:  "v1.1.16",
				Status:         "update-available",
			},
		}

		err := service.updateFiles(fileContents, results)
		require.NoError(t, err)

		// Only coverage file should be backed up and written
		writtenPaths := updater.GetAllWrittenPaths()
		assert.Equal(t, []string{".github/env/10-coverage.env"}, writtenPaths)
		assert.True(t, updater.WasBackedUp(".github/env/10-coverage.env"))
		assert.False(t, updater.WasBackedUp(".github/env/20-redis.env"))
		assert.False(t, updater.WasBackedUp(".github/env/20-workflows.env"))
	})
}

func TestVersionUpdateService_Run_DryRun(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 10*time.Millisecond)

	coverageFile := ".github/env/10-coverage.env"
	mageXFile := ".github/env/10-mage-x.env"

	updater.SetContent(coverageFile, []byte("GO_COVERAGE_VERSION=v1.1.15\n"))
	updater.SetContent(mageXFile, []byte("MAGE_X_VERSION=v1.8.7\n"))
	// go-coverage is a go-install tool, so the checker is keyed by its Go module path.
	checker.SetVersion("github.com/mrz1836/go-coverage", "v1.1.16")
	checker.SetVersion("https://github.com/mrz1836/mage-x", "v1.8.8")

	ctx := context.Background()
	err := service.Run(ctx, []string{coverageFile, mageXFile})

	require.NoError(t, err)

	// In dry run mode, no files should be written
	assert.Empty(t, updater.GetAllWrittenPaths())
	assert.Empty(t, updater.backedUpPaths)

	// Logger should have been used
	assert.NotEmpty(t, logger.GetInfoMessages())
}

func TestVersionUpdateService_Run_ActualUpdate(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, false, false, 10*time.Millisecond)

	coverageFile := ".github/env/10-coverage.env"
	mageXFile := ".github/env/10-mage-x.env"

	updater.SetContent(coverageFile, []byte("GO_COVERAGE_VERSION=v1.1.15\n"))
	updater.SetContent(mageXFile, []byte("MAGE_X_VERSION=v1.8.7\n"))
	// go-coverage is a go-install tool, so the checker is keyed by its Go module path.
	checker.SetVersion("github.com/mrz1836/go-coverage", "v1.1.16")
	checker.SetVersion("https://github.com/mrz1836/mage-x", "v1.8.8")

	ctx := context.Background()
	err := service.Run(ctx, []string{coverageFile, mageXFile})

	require.NoError(t, err)

	// Both files should be backed up and written
	assert.True(t, updater.WasBackedUp(coverageFile))
	assert.True(t, updater.WasBackedUp(mageXFile))

	// Verify updates
	writtenCoverage := string(updater.GetWrittenData(coverageFile))
	assert.Contains(t, writtenCoverage, "GO_COVERAGE_VERSION=v1.1.16")

	writtenMageX := string(updater.GetWrittenData(mageXFile))
	assert.Contains(t, writtenMageX, "MAGE_X_VERSION=v1.8.8")
}

func TestVersionUpdateService_Run_ReadError(t *testing.T) {
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, true, false, 0)

	updater.readError = errFileNotFound

	ctx := context.Background()
	err := service.Run(ctx, []string{".github/env/10-coverage.env"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestVersionUpdateService_Run_MultiFile(t *testing.T) {
	// End-to-end test: versions spread across multiple files
	checker := NewMockVersionChecker()
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

	coreFile := ".github/env/00-core.env"
	coverageFile := ".github/env/10-coverage.env"
	mageXFile := ".github/env/10-mage-x.env"
	securityFile := ".github/env/10-security.env"

	updater.SetContent(coreFile, []byte("GOVULNCHECK_GO_VERSION=1.25.7\n"))
	updater.SetContent(coverageFile, []byte("GO_COVERAGE_VERSION=v1.2.0\n"))
	updater.SetContent(mageXFile, []byte("MAGE_X_VERSION=v1.19.0\nMAGE_X_GITLEAKS_VERSION=8.29.0\n"))
	updater.SetContent(securityFile, []byte("GITLEAKS_VERSION=8.29.0\n"))

	checker.SetVersion("https://go.dev", "go1.25.7")               // up-to-date
	checker.SetVersion("github.com/mrz1836/go-coverage", "v1.3.0") // go-install tool: keyed by module path
	checker.SetVersion("https://github.com/mrz1836/mage-x", "v1.20.0")
	checker.SetVersion("https://github.com/gitleaks/gitleaks", "v8.30.0")

	ctx := context.Background()
	envFiles := []string{coreFile, coverageFile, mageXFile, securityFile}
	err := service.Run(ctx, envFiles)
	require.NoError(t, err)

	// Core file should NOT be written (Go version up-to-date)
	assert.False(t, updater.WasBackedUp(coreFile))

	// Coverage file should be updated
	writtenCoverage := string(updater.GetWrittenData(coverageFile))
	assert.Contains(t, writtenCoverage, "GO_COVERAGE_VERSION=v1.3.0")

	// Mage-X file should be updated
	writtenMageX := string(updater.GetWrittenData(mageXFile))
	assert.Contains(t, writtenMageX, "MAGE_X_VERSION=v1.20.0")
	assert.Contains(t, writtenMageX, "MAGE_X_GITLEAKS_VERSION=8.30.0") // no v prefix preserved

	// Security file should be updated
	writtenSecurity := string(updater.GetWrittenData(securityFile))
	assert.Contains(t, writtenSecurity, "GITLEAKS_VERSION=8.30.0") // no v prefix preserved
}

func TestRunVersionUpdate(t *testing.T) {
	t.Run("dry run mode", func(t *testing.T) {
		// RunVersionUpdate uses relative path .github/env, so chdir to project root
		t.Chdir("..")

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

		// Set up mocks - ReadFile returns empty for undiscovered paths
		checker.SetVersion("https://github.com/mrz1836/go-coverage", "v1.1.15")

		// Inject mock service
		setVersionUpdateService(mockService)

		err := RunVersionUpdate(true, false)
		require.NoError(t, err)
	})
}

func TestDiscoverEnvFiles(t *testing.T) {
	t.Run("discovers and sorts env files", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create test env files
		for _, name := range []string{"00-core.env", "10-mage-x.env", "10-security.env", "20-redis.env"} {
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, name), []byte("# test"), 0o600))
		}

		files, err := discoverEnvFiles(tmpDir)
		require.NoError(t, err)
		require.Len(t, files, 4)
		assert.Equal(t, filepath.Join(tmpDir, "00-core.env"), files[0])
		assert.Equal(t, filepath.Join(tmpDir, "10-mage-x.env"), files[1])
		assert.Equal(t, filepath.Join(tmpDir, "10-security.env"), files[2])
		assert.Equal(t, filepath.Join(tmpDir, "20-redis.env"), files[3])
	})

	t.Run("excludes 90- and 99- prefixed files", func(t *testing.T) {
		tmpDir := t.TempDir()
		for _, name := range []string{"00-core.env", "90-project.env", "99-local.env"} {
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, name), []byte("# test"), 0o600))
		}

		files, err := discoverEnvFiles(tmpDir)
		require.NoError(t, err)
		require.Len(t, files, 1)
		assert.Equal(t, filepath.Join(tmpDir, "00-core.env"), files[0])
	})

	t.Run("excludes non-env files", func(t *testing.T) {
		tmpDir := t.TempDir()
		for _, name := range []string{"00-core.env", "load-env.sh", "README.md"} {
			require.NoError(t, os.WriteFile(filepath.Join(tmpDir, name), []byte("# test"), 0o600))
		}

		files, err := discoverEnvFiles(tmpDir)
		require.NoError(t, err)
		require.Len(t, files, 1)
		assert.Equal(t, filepath.Join(tmpDir, "00-core.env"), files[0])
	})

	t.Run("returns error when no eligible files found", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Only excluded files
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "90-project.env"), []byte("# test"), 0o600))

		_, err := discoverEnvFiles(tmpDir)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoEnvFiles)
	})

	t.Run("returns error when directory is empty", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, err := discoverEnvFiles(tmpDir)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoEnvFiles)
	})

	t.Run("returns error when directory does not exist", func(t *testing.T) {
		_, err := discoverEnvFiles("/nonexistent/path")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read env directory")
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

func TestVersionUpdateService_UpdateFiles_SkipsMajorUpgrades(t *testing.T) {
	t.Run("major-skipped status not updated", func(t *testing.T) {
		checker := NewMockVersionChecker()
		updater := NewMockFileUpdater()
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

		mageXContent := []byte(`# Configuration
MAGE_X_SWAG_VERSION=v1.16.6
MAGE_X_VERSION=v1.15.5
`)

		fileContents := map[string][]byte{
			".github/env/10-mage-x.env": mageXContent,
		}

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

		err := service.updateFiles(fileContents, results)
		require.NoError(t, err)

		writtenData := string(updater.GetWrittenData(".github/env/10-mage-x.env"))
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

	mageXFile := ".github/env/10-mage-x.env"
	updater.SetContent(mageXFile, []byte("MAGE_X_SWAG_VERSION=v1.16.6\nMAGE_X_VERSION=v1.15.5\n"))
	// Major upgrade (swag is a go-install tool, so the checker is keyed by module path)
	checker.SetVersion("github.com/swaggo/swag", "v2.0.0")
	// Minor upgrade
	checker.SetVersion("https://github.com/mrz1836/mage-x", "v1.15.6")

	ctx := context.Background()
	err := service.Run(ctx, []string{mageXFile})

	require.NoError(t, err)

	// Both should be updated when allowMajorUpgrades=true
	writtenData := string(updater.GetWrittenData(mageXFile))
	assert.Contains(t, writtenData, "MAGE_X_SWAG_VERSION=v2.0.0")
	assert.Contains(t, writtenData, "MAGE_X_VERSION=v1.15.6")
}

// --- benchstat two-pin / Go-version-held coverage -------------------------------------

func TestGetToolDefinitions_Benchstat(t *testing.T) {
	tools := GetToolDefinitions()

	t.Run("latest pin tracks the newest build unconstrained", func(t *testing.T) {
		tool, ok := tools["benchstat"]
		require.True(t, ok)
		assert.Equal(t, []string{"MAGE_X_BENCHSTAT_VERSION_LATEST"}, tool.EnvVars)
		assert.Equal(t, "golang.org/x/perf", tool.GoModulePath)
		assert.Equal(t, 0, tool.MaxGoMinor, "latest pin must not be Go-version-constrained")
	})

	t.Run("go125 pin is held to Go 1.25", func(t *testing.T) {
		tool, ok := tools["benchstat-go125"]
		require.True(t, ok)
		assert.Equal(t, []string{"MAGE_X_BENCHSTAT_VERSION"}, tool.EnvVars)
		assert.Equal(t, "golang.org/x/perf", tool.GoModulePath)
		assert.Equal(t, 25, tool.MaxGoMinor, "go125 pin must be held to Go 1.25")
	})
}

func TestVersionUpdateService_CheckVersions_GoVersionHeld(t *testing.T) {
	const (
		perf    = "golang.org/x/perf"
		current = "v0.0.0-20260813145340-fd4a688df892" // requires go 1.25
		latest  = "v0.0.0-20260819171926-ebcb4798430d" // requires go 1.26
	)

	newService := func(checker *MockVersionChecker) *VersionUpdateService {
		return NewVersionUpdateService(checker, NewMockFileUpdater(), NewMockLogger(), true, false, 0)
	}
	go125Tool := map[string]*ToolInfo{
		"benchstat-go125": {
			EnvVars:      []string{"MAGE_X_BENCHSTAT_VERSION"},
			GoModulePath: perf,
			MaxGoMinor:   25,
		},
	}

	t.Run("holds when latest requires a newer Go than allowed", func(t *testing.T) {
		checker := NewMockVersionChecker()
		checker.SetVersion(perf, latest)
		checker.SetGoRequirement(perf, latest, 1, 26)
		service := newService(checker)

		results := service.checkVersions(context.Background(), go125Tool, map[string]string{"benchstat-go125": current})

		require.Len(t, results, 1)
		assert.Equal(t, "go-version-held", results[0].Status)
		// The real (incompatible) latest is still surfaced for visibility...
		assert.Equal(t, latest, results[0].LatestVersion)
		assert.Equal(t, current, results[0].CurrentVersion)
		// ...but it must not count as an update, so no file write happens.
		assert.False(t, service.hasUpdates(results))
	})

	t.Run("updates when latest still supports the allowed Go", func(t *testing.T) {
		checker := NewMockVersionChecker()
		checker.SetVersion(perf, latest)
		checker.SetGoRequirement(perf, latest, 1, 25) // still Go 1.25
		service := newService(checker)

		results := service.checkVersions(context.Background(), go125Tool, map[string]string{"benchstat-go125": current})

		require.Len(t, results, 1)
		assert.Equal(t, "update-available", results[0].Status)
		assert.Equal(t, latest, results[0].LatestVersion)
	})

	t.Run("holds conservatively when the Go requirement cannot be resolved", func(t *testing.T) {
		checker := NewMockVersionChecker()
		checker.SetVersion(perf, latest)
		checker.SetGoRequirementError(perf, latest, errNotFound)
		service := newService(checker)

		results := service.checkVersions(context.Background(), go125Tool, map[string]string{"benchstat-go125": current})

		require.Len(t, results, 1)
		assert.Equal(t, "go-version-held", results[0].Status)
	})

	t.Run("holds when latest requires a new major Go", func(t *testing.T) {
		checker := NewMockVersionChecker()
		checker.SetVersion(perf, latest)
		checker.SetGoRequirement(perf, latest, 2, 0) // hypothetical Go 2.0
		service := newService(checker)

		results := service.checkVersions(context.Background(), go125Tool, map[string]string{"benchstat-go125": current})

		require.Len(t, results, 1)
		assert.Equal(t, "go-version-held", results[0].Status)
	})

	t.Run("no go.mod lookup when already up to date", func(t *testing.T) {
		checker := NewMockVersionChecker()
		checker.SetVersion(perf, current) // latest == current
		service := newService(checker)

		results := service.checkVersions(context.Background(), go125Tool, map[string]string{"benchstat-go125": current})

		require.Len(t, results, 1)
		assert.Equal(t, "up-to-date", results[0].Status)
		assert.Empty(t, checker.goReqCalls, "no Go requirement lookup needed when unchanged")
	})

	t.Run("unconstrained tools never trigger a go.mod lookup", func(t *testing.T) {
		checker := NewMockVersionChecker()
		checker.SetVersion(perf, latest)
		service := newService(checker)

		latestTool := map[string]*ToolInfo{
			"benchstat": {EnvVars: []string{"MAGE_X_BENCHSTAT_VERSION_LATEST"}, GoModulePath: perf}, // MaxGoMinor 0
		}
		results := service.checkVersions(context.Background(), latestTool, map[string]string{"benchstat": current})

		require.Len(t, results, 1)
		assert.Equal(t, "update-available", results[0].Status)
		assert.Empty(t, checker.goReqCalls)
	})
}

func TestVersionUpdateService_HasUpdates_ExcludesGoVersionHeld(t *testing.T) {
	service := NewVersionUpdateService(NewMockVersionChecker(), NewMockFileUpdater(), NewMockLogger(), true, false, 0)

	assert.False(t, service.hasUpdates([]CheckResult{{Status: "up-to-date"}, {Status: "go-version-held"}}))
	assert.True(t, service.hasUpdates([]CheckResult{{Status: "go-version-held"}, {Status: "update-available"}}))
}

func TestVersionUpdateService_UpdateFiles_SkipsGoVersionHeld(t *testing.T) {
	updater := NewMockFileUpdater()
	service := NewVersionUpdateService(NewMockVersionChecker(), updater, NewMockLogger(), false, false, 0)

	const file = ".github/env/10-mage-x.env"
	updater.SetContent(file, []byte("MAGE_X_BENCHSTAT_VERSION=v0.0.0-20260813145340-fd4a688df892\n"))

	results := []CheckResult{{
		Tool:           "benchstat-go125",
		EnvVars:        []string{"MAGE_X_BENCHSTAT_VERSION"},
		CurrentVersion: "v0.0.0-20260813145340-fd4a688df892",
		LatestVersion:  "v0.0.0-20260819171926-ebcb4798430d",
		Status:         "go-version-held",
	}}

	require.NoError(t, service.updateFiles(map[string][]byte{file: updater.contents[file]}, results))
	// Held pins must be left untouched on disk.
	assert.Empty(t, updater.GetWrittenData(file), "held pin should not be written")
}

func TestExtractEnvValue(t *testing.T) {
	content := []byte("# comment\nMAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO=1.26\nOTHER=x\n")

	assert.Equal(t, "1.26", extractEnvValue(content, "MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO"))
	assert.Equal(t, "x", extractEnvValue(content, "OTHER"))
	assert.Empty(t, extractEnvValue(content, "MISSING"))
	// Must not match a variable whose name is a prefix of the queried one.
	assert.Empty(t, extractEnvValue([]byte("MAGE_X_BENCHSTAT_VERSION=v1\n"), "MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO"))
}

func TestVersionUpdateService_MaintainBenchstatMinGo(t *testing.T) {
	const (
		perf   = "golang.org/x/perf"
		latest = "v0.0.0-20260819171926-ebcb4798430d"
	)
	benchstatResult := CheckResult{Tool: "benchstat", Status: "update-available", LatestVersion: latest}

	t.Run("appends update when boundary changed", func(t *testing.T) {
		checker := NewMockVersionChecker()
		checker.SetGoRequirement(perf, latest, 1, 26)
		service := NewVersionUpdateService(checker, NewMockFileUpdater(), NewMockLogger(), true, false, 0)

		content := []byte("MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO=1.25\n")
		out := service.maintainBenchstatMinGo(context.Background(), []CheckResult{benchstatResult}, content)

		require.Len(t, out, 2)
		boundary := out[1]
		assert.Equal(t, []string{"MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO"}, boundary.EnvVars)
		assert.Equal(t, "1.25", boundary.CurrentVersion)
		assert.Equal(t, "1.26", boundary.LatestVersion)
		assert.Equal(t, "update-available", boundary.Status)
	})

	t.Run("appends up-to-date when boundary already correct", func(t *testing.T) {
		checker := NewMockVersionChecker()
		checker.SetGoRequirement(perf, latest, 1, 26)
		service := NewVersionUpdateService(checker, NewMockFileUpdater(), NewMockLogger(), true, false, 0)

		content := []byte("MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO=1.26\n")
		out := service.maintainBenchstatMinGo(context.Background(), []CheckResult{benchstatResult}, content)

		require.Len(t, out, 2)
		assert.Equal(t, "up-to-date", out[1].Status)
	})

	t.Run("no-op when env var absent from files", func(t *testing.T) {
		checker := NewMockVersionChecker()
		checker.SetGoRequirement(perf, latest, 1, 26)
		service := NewVersionUpdateService(checker, NewMockFileUpdater(), NewMockLogger(), true, false, 0)

		out := service.maintainBenchstatMinGo(context.Background(), []CheckResult{benchstatResult}, []byte("UNRELATED=1\n"))
		assert.Len(t, out, 1, "should not append a boundary result when the var is not present")
		assert.Empty(t, checker.goReqCalls, "should not fetch go.mod when the var is absent")
	})

	t.Run("no-op when benchstat check errored", func(t *testing.T) {
		checker := NewMockVersionChecker()
		service := NewVersionUpdateService(checker, NewMockFileUpdater(), NewMockLogger(), true, false, 0)

		content := []byte("MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO=1.26\n")
		errored := []CheckResult{{Tool: "benchstat", Status: "error", LatestVersion: ""}}
		out := service.maintainBenchstatMinGo(context.Background(), errored, content)
		assert.Len(t, out, 1)
	})

	t.Run("leaves boundary unchanged when go.mod lookup fails", func(t *testing.T) {
		checker := NewMockVersionChecker()
		checker.SetGoRequirementError(perf, latest, errNotFound)
		logger := NewMockLogger()
		service := NewVersionUpdateService(checker, NewMockFileUpdater(), logger, true, false, 0)

		content := []byte("MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO=1.26\n")
		out := service.maintainBenchstatMinGo(context.Background(), []CheckResult{benchstatResult}, content)
		assert.Len(t, out, 1, "boundary result should not be appended on lookup failure")
		assert.NotEmpty(t, logger.warnMessages, "a warning should be logged")
	})
}

func TestVersionUpdateService_Run_MaintainsBenchstatMinGo(t *testing.T) {
	const (
		perf   = "golang.org/x/perf"
		go125  = "v0.0.0-20260813145340-fd4a688df892"
		latest = "v0.0.0-20260819171926-ebcb4798430d"
	)
	checker := NewMockVersionChecker()
	// Proxy @latest is the newest (Go 1.26) build; both benchstat pins resolve it.
	checker.SetVersion(perf, latest)
	checker.SetGoRequirement(perf, latest, 1, 26)
	updater := NewMockFileUpdater()
	logger := NewMockLogger()
	service := NewVersionUpdateService(checker, updater, logger, false, false, 0)

	const file = ".github/env/10-mage-x.env"
	updater.SetContent(file, []byte(
		"MAGE_X_BENCHSTAT_VERSION="+go125+"\n"+
			"MAGE_X_BENCHSTAT_VERSION_LATEST="+go125+"\n"+
			"MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO=1.25\n",
	))

	require.NoError(t, service.Run(context.Background(), []string{file}))

	written := string(updater.GetWrittenData(file))
	// LATEST advances to the newest build.
	assert.Contains(t, written, "MAGE_X_BENCHSTAT_VERSION_LATEST="+latest)
	// The Go 1.25 pin is held because latest now requires Go 1.26.
	assert.Contains(t, written, "MAGE_X_BENCHSTAT_VERSION="+go125)
	// The boundary is refreshed to the latest build's requirement.
	assert.Contains(t, written, "MAGE_X_BENCHSTAT_VERSION_LATEST_MIN_GO=1.26")
}

func TestVersionChecker_ModuleGoRequirement_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	checker := NewVersionChecker(false)
	ctx := context.Background()

	// Pinned historical benchstat builds with stable go.mod 'go' directives.
	cases := []struct {
		version   string
		wantMajor int
		wantMinor int
	}{
		{"v0.0.0-20260813145340-fd4a688df892", 1, 25},
		{"v0.0.0-20260819171926-ebcb4798430d", 1, 26},
	}
	for _, tc := range cases {
		major, minor, err := checker.CheckModuleGoRequirement(ctx, "golang.org/x/perf", tc.version)
		if err != nil {
			t.Logf("Network error (expected in some envs): %v", err)
			continue
		}
		assert.Equal(t, tc.wantMajor, major, "major for %s", tc.version)
		assert.Equal(t, tc.wantMinor, minor, "minor for %s", tc.version)
	}
}
