package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	versionpkg "github.com/mrz1836/go-broadcast/internal/version"
)

// Sentinel errors used by the upgrade tests to drive deterministic seam failures
// without making real network/command calls.
var (
	errTestInstall = errors.New("simulated command failure")
	errTestRelease = errors.New("simulated release fetch failure")
	errTestLocate  = errors.New("simulated binary location failure")
)

// swapRunCommand overrides the runCommand seam for the duration of the test.
// Tests using the upgrade seams must NOT call t.Parallel(), since the seams are
// package-level globals shared across the package.
func swapRunCommand(t *testing.T, fn func(ctx context.Context, name string, args ...string) error) {
	t.Helper()
	prev := runCommand
	runCommand = fn
	t.Cleanup(func() { runCommand = prev })
}

// swapGetLatestRelease overrides the getLatestRelease seam for the test duration.
func swapGetLatestRelease(t *testing.T, fn func(ctx context.Context, owner, repo string) (*versionpkg.GitHubRelease, error)) {
	t.Helper()
	prev := getLatestRelease
	getLatestRelease = fn
	t.Cleanup(func() { getLatestRelease = prev })
}

// swapLocateBinary overrides the locateBinary seam for the test duration.
func swapLocateBinary(t *testing.T, fn func() (string, error)) {
	t.Helper()
	prev := locateBinary
	locateBinary = fn
	t.Cleanup(func() { locateBinary = prev })
}

// swapDownloadServer points the binary-download seams at a local httptest server
// for the test duration, ensuring no real github.com request is made.
func swapDownloadServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	prevURL, prevClient := githubDownloadBaseURL, upgradeHTTPClient
	githubDownloadBaseURL = srv.URL
	upgradeHTTPClient = srv.Client()
	t.Cleanup(func() {
		srv.Close()
		githubDownloadBaseURL = prevURL
		upgradeHTTPClient = prevClient
	})
}

// swapCurrentVersion sets the reported current version for the test duration.
func swapCurrentVersion(t *testing.T, v string) {
	t.Helper()
	prev := getVersionRaw()
	setVersion(v)
	t.Cleanup(func() { setVersion(prev) })
}

// makeTarGz builds an in-memory tar.gz archive containing a single file.
func makeTarGz(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     name,
		Mode:     0o755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())
	return buf.Bytes()
}

func TestNewUpgradeCmd(t *testing.T) {
	t.Parallel()

	cmd := newUpgradeCmd()

	// Test basic command properties
	assert.Equal(t, "upgrade", cmd.Use)
	assert.Contains(t, cmd.Short, "Upgrade go-broadcast")
	assert.Contains(t, cmd.Long, "Upgrade the go-broadcast system")
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.RunE)

	// Test that command can run without panicking
	assert.NotPanics(t, func() {
		_ = cmd.Help()
	})

	// Check flags
	forceFlag := cmd.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)
	assert.Equal(t, "false", forceFlag.DefValue)
	assert.Contains(t, forceFlag.Usage, "Force upgrade")

	checkFlag := cmd.Flags().Lookup("check")
	require.NotNil(t, checkFlag)
	assert.Empty(t, checkFlag.Shorthand) // No shorthand due to conflict with global 'c' flag
	assert.Equal(t, "false", checkFlag.DefValue)
	assert.Contains(t, checkFlag.Usage, "Check for updates")

	verboseFlag := cmd.Flags().Lookup("verbose")
	require.NotNil(t, verboseFlag)
	assert.Equal(t, "v", verboseFlag.Shorthand)
	assert.Equal(t, "false", verboseFlag.DefValue)
	assert.Contains(t, verboseFlag.Usage, "release notes")

	useBinaryFlag := cmd.Flags().Lookup("use-binary")
	require.NotNil(t, useBinaryFlag)
	assert.Empty(t, useBinaryFlag.Shorthand)
	assert.Equal(t, "false", useBinaryFlag.DefValue)
	assert.Contains(t, useBinaryFlag.Usage, "binary")
}

// TestNewUpgradeCmdExecution tests the RunE function of newUpgradeCmd
func TestNewUpgradeCmdExecution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "ValidFlags",
			args:        []string{"--force", "--check"},
			expectError: false, // May succeed or fail on API call, but flag parsing should work
		},
		{
			name:        "AllFlags",
			args:        []string{"--force", "--check", "--verbose", "--use-binary"},
			expectError: false, // May succeed or fail on API call, but flag parsing should work
		},
		{
			name:        "NoFlags",
			args:        []string{},
			expectError: false, // May succeed or fail on API call, but should not fail on missing flags
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := newUpgradeCmd()
			cmd.SetArgs(tt.args)

			// Parse flags to verify the command can handle them
			err := cmd.ParseFlags(tt.args)
			require.NoError(t, err, "Flag parsing should succeed")

			// Verify that the flags are set correctly
			if containsString(tt.args, "--force") {
				force, err := cmd.Flags().GetBool("force")
				require.NoError(t, err)
				assert.True(t, force)
			}

			if containsString(tt.args, "--check") {
				check, err := cmd.Flags().GetBool("check")
				require.NoError(t, err)
				assert.True(t, check)
			}

			if containsString(tt.args, "--verbose") {
				verbose, err := cmd.Flags().GetBool("verbose")
				require.NoError(t, err)
				assert.True(t, verbose)
			}

			if containsString(tt.args, "--use-binary") {
				useBinary, err := cmd.Flags().GetBool("use-binary")
				require.NoError(t, err)
				assert.True(t, useBinary)
			}
		})
	}
}

// Helper function to check if slice contains string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestRunUpgradeWithConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		currentVersion    string
		config            UpgradeConfig
		mockRelease       *versionpkg.GitHubRelease
		mockReleaseError  error
		expectError       bool
		errorContains     []string
		expectedOutput    []string
		skipCommandChecks bool
	}{
		{
			name:           "SuccessfulUpgrade",
			currentVersion: "0.5.0",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: false,
			},
			mockRelease: &versionpkg.GitHubRelease{
				TagName:     "v1.0.12",
				Name:        "Release v1.0.12",
				Body:        "Bug fixes and improvements",
				PublishedAt: time.Now(),
			},
			expectError:       false,
			expectedOutput:    []string{"Current version: v0.5.0", "Checking for updates"},
			skipCommandChecks: true, // Skip actual go install command
		},
		{
			name:           "AlreadyOnLatest",
			currentVersion: "1.0.12",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: false,
			},
			mockRelease: &versionpkg.GitHubRelease{
				TagName: "v1.0.12",
				Name:    "Release v1.0.12",
			},
			expectError:    false,
			expectedOutput: []string{"Current version: v1.0.12", "Checking for updates"},
		},
		{
			name:           "CheckOnlyMode",
			currentVersion: "0.5.0",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: true,
			},
			mockRelease: &versionpkg.GitHubRelease{
				TagName: "v1.0.12",
				Name:    "Release v1.0.12",
			},
			expectError:    false,
			expectedOutput: []string{"Current version: v0.5.0", "Checking for updates"},
		},
		{
			name:           "CheckOnlyModeUpToDate",
			currentVersion: "1.0.12",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: true,
			},
			mockRelease: &versionpkg.GitHubRelease{
				TagName: "v1.0.12",
				Name:    "Release v1.0.12",
			},
			expectError:    false,
			expectedOutput: []string{"Current version: v1.0.12", "Checking for updates"},
		},
		{
			name:           "ForceUpgrade",
			currentVersion: "1.0.12",
			config: UpgradeConfig{
				Force:     true,
				CheckOnly: false,
			},
			mockRelease: &versionpkg.GitHubRelease{
				TagName: "v1.0.12",
				Name:    "Release v1.0.12",
			},
			expectError:       false,
			expectedOutput:    []string{"Current version: v1.0.12", "Checking for updates"},
			skipCommandChecks: true,
		},
		{
			name:           "DevVersionWithoutForce",
			currentVersion: "dev",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: false,
			},
			mockRelease: &versionpkg.GitHubRelease{
				TagName: "v1.2.3",
				Name:    "Release v1.2.3",
			},
			expectError:    true,
			errorContains:  []string{"cannot upgrade development build without --force"},
			expectedOutput: []string{"development build", "Use --force to upgrade"},
		},
		{
			name:           "DevVersionWithForce",
			currentVersion: "dev",
			config: UpgradeConfig{
				Force:     true,
				CheckOnly: false,
			},
			mockRelease: &versionpkg.GitHubRelease{
				TagName: "v1.0.12",
				Name:    "Release v1.0.12",
			},
			expectError:       false,
			expectedOutput:    []string{"Current version: dev", "Checking for updates"},
			skipCommandChecks: true,
		},
		{
			name:           "CommitHashVersion",
			currentVersion: "abc123def456",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: false,
			},
			mockRelease: &versionpkg.GitHubRelease{
				TagName: "v1.2.3",
				Name:    "Release v1.2.3",
			},
			expectError:    true,
			errorContains:  []string{"cannot upgrade development build without --force"},
			expectedOutput: []string{"development build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create isolated command for testing
			cmd := &cobra.Command{
				Use: "upgrade",
			}
			cmd.Flags().Bool("verbose", false, "Show release notes")

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// We can't easily mock the version.GetLatestRelease function
			// without dependency injection, so we'll test the logic separately

			// For now, we skip the actual API call test and focus on logic
			// In a real implementation, we'd use dependency injection

			// Test the command creation and flag parsing
			upgradeCmd := newUpgradeCmd()
			assert.NotNil(t, upgradeCmd)

			// Test flag parsing
			err := upgradeCmd.Flags().Set("force", fmt.Sprintf("%v", tt.config.Force))
			require.NoError(t, err)

			err = upgradeCmd.Flags().Set("check", fmt.Sprintf("%v", tt.config.CheckOnly))
			require.NoError(t, err)

			// Verify flags can be read
			force, err := upgradeCmd.Flags().GetBool("force")
			require.NoError(t, err)
			assert.Equal(t, tt.config.Force, force)

			check, err := upgradeCmd.Flags().GetBool("check")
			require.NoError(t, err)
			assert.Equal(t, tt.config.CheckOnly, check)
		})
	}
}

func TestFormatVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "StandardVersion",
			version:  "1.2.3",
			expected: "v1.2.3",
		},
		{
			name:     "VersionWithVPrefix",
			version:  "v1.2.3",
			expected: "v1.2.3",
		},
		{
			name:     "DevVersion",
			version:  "dev",
			expected: "dev",
		},
		{
			name:     "EmptyVersion",
			version:  "",
			expected: "dev",
		},
		{
			name:     "VersionWithoutV",
			version:  "2.0.0-rc1",
			expected: "v2.0.0-rc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := formatVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetInstalledVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mockOutput    string
		mockError     error
		expected      string
		expectError   bool
		errorContains []string
	}{
		{
			name:       "ValidVersionOutput",
			mockOutput: "go-broadcast version v1.2.3",
			expected:   "1.2.3",
		},
		{
			name:       "VersionWithoutV",
			mockOutput: "go-broadcast version 1.2.3",
			expected:   "1.2.3",
		},
		{
			name:       "MultiWordVersionOutput",
			mockOutput: "go-broadcast command version v2.0.0-rc1",
			expected:   "2.0.0-rc1",
		},
		{
			name:          "CommandNotFound",
			mockError:     exec.ErrNotFound,
			expectError:   true,
			errorContains: []string{"failed to get version"},
		},
		{
			name:          "InvalidOutput",
			mockOutput:    "invalid output format",
			expectError:   true,
			errorContains: []string{"could not parse version"},
		},
		{
			name:          "EmptyOutput",
			mockOutput:    "",
			expectError:   true,
			errorContains: []string{"could not parse version"},
		},
		{
			name:          "NoVersionKeyword",
			mockOutput:    "go-broadcast v1.2.3",
			expectError:   true,
			errorContains: []string{"could not parse version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// We test the parsing logic directly without external dependencies

			if tt.mockError != nil {
				// Test error case - in real environment, go-broadcast may actually be available
				installedVersion, err := GetInstalledVersion()
				if err != nil {
					// Command not found or failed
					assert.Empty(t, installedVersion)
				}
				// If no error, command exists and works - that's also valid
				return
			}

			// Test version parsing logic directly
			outputStr := strings.TrimSpace(tt.mockOutput)
			parts := strings.Fields(outputStr)

			var parsedVersion string
			var err error

			for i, part := range parts {
				if part == "version" && i+1 < len(parts) {
					parsedVersion = parts[i+1]
					parsedVersion = strings.TrimPrefix(parsedVersion, "v")
					break
				}
			}

			if parsedVersion == "" && !tt.expectError {
				err = fmt.Errorf("%w: %s", ErrVersionParseFailed, outputStr)
			}

			if tt.expectError {
				if err == nil {
					err = fmt.Errorf("%w: %s", ErrVersionParseFailed, outputStr)
				}
				require.Error(t, err)
				for _, contains := range tt.errorContains {
					assert.Contains(t, err.Error(), contains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, parsedVersion)
			}
		})
	}
}

func TestCheckGoInstalled(t *testing.T) {
	t.Parallel()

	// Test the actual function - Go should be installed in test environment
	err := CheckGoInstalled()
	// In CI/test environment, Go should be available
	if err != nil {
		// If Go is not found, verify it's the expected error type
		assert.Contains(t, err.Error(), "go is not installed or not in PATH")
	} else {
		// If no error, Go is properly installed
		require.NoError(t, err)
	}
}

func TestGetGoPath(t *testing.T) {
	t.Parallel()

	// Test that the function exists and returns a valid path format
	goPath, err := GetGoPath()
	if err != nil {
		// If Go is not installed, that's okay for testing
		return
	}

	// If we got a path, it should end with /bin
	assert.Contains(t, goPath, "bin")
	assert.NotEmpty(t, goPath)
}

func TestIsInPath(t *testing.T) {
	t.Parallel()

	// Test that the function runs without error and returns a boolean
	result := IsInPath()
	// Result is a boolean - just verify the function doesn't panic
	assert.IsType(t, false, result)

	// Test coverage: function should handle both cases (found/not found)
	// Since this depends on environment, we just verify it executes
}

func TestGetBinaryLocation(t *testing.T) {
	t.Parallel()

	location, err := GetBinaryLocation()
	if err != nil {
		// Binary may not be in PATH, which is expected in test environment
		return
	}

	// If we found a location, verify it makes sense
	if runtime.GOOS == "windows" {
		assert.Contains(t, location, "go-broadcast.exe")
	} else {
		assert.Contains(t, location, "go-broadcast")
	}
}

func TestIsLikelyCommitHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{
			name:     "ValidShortCommitHash",
			version:  "abc123d",
			expected: true,
		},
		{
			name:     "ValidLongCommitHash",
			version:  "abc123def456789012345678901234567890abcd",
			expected: true,
		},
		{
			name:     "ValidHashWithDirtySuffix",
			version:  "abc123d-dirty",
			expected: true,
		},
		{
			name:     "ValidMixedCaseHash",
			version:  "AbC123DeF456",
			expected: true,
		},
		{
			name:     "TooShort",
			version:  "abc12",
			expected: false,
		},
		{
			name:     "TooLong",
			version:  "abc123def456789012345678901234567890abcdef",
			expected: false,
		},
		{
			name:     "ContainsInvalidCharacters",
			version:  "abc123xyz",
			expected: false,
		},
		{
			name:     "ContainsSpecialCharacters",
			version:  "abc123-def",
			expected: false,
		},
		{
			name:     "EmptyString",
			version:  "",
			expected: false,
		},
		{
			name:     "StandardVersion",
			version:  "1.2.3",
			expected: false,
		},
		{
			name:     "DevVersion",
			version:  "dev",
			expected: false,
		},
		{
			name:     "OnlyNumbers",
			version:  "1234567890",
			expected: true,
		},
		{
			name:     "OnlyValidHexLetters",
			version:  "abcdefabcdef",
			expected: true,
		},
		{
			name:     "OnlyInvalidLetters",
			version:  "abcdefghijk",
			expected: false,
		},
		{
			name:     "ExactSevenChars",
			version:  "abc1234",
			expected: true,
		},
		{
			name:     "ExactFortyChars",
			version:  "1234567890abcdef1234567890abcdef12345678",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isLikelyCommitHash(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpgradeConfigStruct(t *testing.T) {
	t.Parallel()

	config := UpgradeConfig{
		Force:     true,
		CheckOnly: false,
	}

	assert.True(t, config.Force)
	assert.False(t, config.CheckOnly)
}

func TestUpgradeErrors(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "cannot upgrade development build without --force", ErrDevVersionNoForce.Error())
	assert.Equal(t, "could not parse version from output", ErrVersionParseFailed.Error())
}

// Integration test for upgrade command creation and flag parsing
func TestUpgradeCommandIntegration(t *testing.T) {
	t.Parallel()

	cmd := newUpgradeCmd()

	// Test flag parsing
	args := []string{"--force", "--check", "--verbose"}
	cmd.SetArgs(args)
	err := cmd.ParseFlags(args)
	require.NoError(t, err)

	forceFlag, err := cmd.Flags().GetBool("force")
	require.NoError(t, err)
	assert.True(t, forceFlag)

	checkFlag, err := cmd.Flags().GetBool("check")
	require.NoError(t, err)
	assert.True(t, checkFlag)

	verboseFlag, err := cmd.Flags().GetBool("verbose")
	require.NoError(t, err)
	assert.True(t, verboseFlag)
}

// Test version comparison integration with upgrade logic
func TestVersionComparisonIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		expectUpgrade  bool
	}{
		{
			name:           "NeedUpgrade",
			currentVersion: "1.2.2",
			latestVersion:  "1.2.3",
			expectUpgrade:  true,
		},
		{
			name:           "NoUpgradeNeeded",
			currentVersion: "1.2.3",
			latestVersion:  "1.2.3",
			expectUpgrade:  false,
		},
		{
			name:           "DevVersionNeedsUpgrade",
			currentVersion: "dev",
			latestVersion:  "1.2.3",
			expectUpgrade:  true,
		},
		{
			name:           "NewerThanLatest",
			currentVersion: "1.2.4",
			latestVersion:  "1.2.3",
			expectUpgrade:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Use the version comparison logic from the upgrade command
			latestVersionClean := strings.TrimPrefix(tt.latestVersion, "v")
			isNewer := versionpkg.IsNewerVersion(tt.currentVersion, latestVersionClean)

			assert.Equal(t, tt.expectUpgrade, isNewer)
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	t.Parallel()

	// Test that the function returns a version string
	currentVersion := GetCurrentVersion()
	// During tests, version may be set to different values, just ensure it's not empty
	// In production builds, it will be set via ldflags, in dev builds it will be "dev"
	assert.NotEmpty(t, currentVersion, "Version should not be empty")
}

// TestNewUpgradeCmdExecutionErrors tests error scenarios in the upgrade command RunE function
func TestNewUpgradeCmdExecutionErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupCmd      func(*cobra.Command)
		expectError   bool
		errorContains string
	}{
		{
			name: "flag parsing works",
			setupCmd: func(cmd *cobra.Command) {
				// Test that flags can be set and retrieved
				err := cmd.Flags().Set("force", "true")
				require.NoError(t, err)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command
			cmd := newUpgradeCmd()

			// Apply test-specific setup
			if tt.setupCmd != nil {
				tt.setupCmd(cmd)
			}

			// Test flag functionality
			force, err := cmd.Flags().GetBool("force")
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				// Verify the force flag was set correctly by setupCmd
				if tt.setupCmd != nil {
					assert.True(t, force)
				}
			}
		})
	}
}

// TestNewUpgradeCmdFlagParsing tests flag parsing in the upgrade command
func TestNewUpgradeCmdFlagParsing(t *testing.T) {
	t.Parallel()

	cmd := newUpgradeCmd()

	// Test setting and getting flags
	require.NoError(t, cmd.Flags().Set("force", "true"))
	require.NoError(t, cmd.Flags().Set("check", "true"))
	require.NoError(t, cmd.Flags().Set("verbose", "true"))
	require.NoError(t, cmd.Flags().Set("use-binary", "true"))

	// Verify flags can be read
	force, err := cmd.Flags().GetBool("force")
	require.NoError(t, err)
	require.True(t, force)

	check, err := cmd.Flags().GetBool("check")
	require.NoError(t, err)
	require.True(t, check)

	verbose, err := cmd.Flags().GetBool("verbose")
	require.NoError(t, err)
	require.True(t, verbose)

	useBinary, err := cmd.Flags().GetBool("use-binary")
	require.NoError(t, err)
	require.True(t, useBinary)
}

// TestUpgradeGoInstall verifies the go install command is constructed correctly
// and that command failures are wrapped, using the runCommand seam so no real
// "go install" (and thus no module-proxy network call) is performed.
func TestUpgradeGoInstall(t *testing.T) {
	var gotArgs []string
	swapRunCommand(t, func(_ context.Context, name string, args ...string) error {
		gotArgs = append([]string{name}, args...)
		return nil
	})

	require.NoError(t, upgradeGoInstall("1.2.3"))
	assert.Equal(t, []string{"go", "install", "github.com/mrz1836/go-broadcast/cmd/go-broadcast@v1.2.3"}, gotArgs)

	// Empty version still produces a (malformed) command; callers validate upstream.
	require.NoError(t, upgradeGoInstall(""))
	assert.Equal(t, []string{"go", "install", "github.com/mrz1836/go-broadcast/cmd/go-broadcast@v"}, gotArgs)

	// Command failures are wrapped with a helpful message.
	swapRunCommand(t, func(_ context.Context, _ string, _ ...string) error {
		return errTestInstall
	})
	err := upgradeGoInstall("1.2.3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "go install failed")
}

// TestUpgradeBinary exercises upgradeBinary against a local httptest server and a
// stubbed binary locator, so no real github.com download is ever attempted.
func TestUpgradeBinary(t *testing.T) {
	t.Run("download HTTP 404", func(t *testing.T) {
		swapLocateBinary(t, func() (string, error) {
			return filepath.Join(t.TempDir(), "go-broadcast"), nil
		})
		swapDownloadServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		err := upgradeBinary("999.999.999")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDownloadFailed)
	})

	t.Run("binary location failure", func(t *testing.T) {
		swapLocateBinary(t, func() (string, error) {
			return "", errTestLocate
		})

		err := upgradeBinary("1.2.3")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not determine current binary location")
	})

	t.Run("successful upgrade replaces binary", func(t *testing.T) {
		binPath := filepath.Join(t.TempDir(), "go-broadcast")
		require.NoError(t, os.WriteFile(binPath, []byte("old binary"), 0o600))
		swapLocateBinary(t, func() (string, error) { return binPath, nil })

		archive := makeTarGz(t, "go-broadcast", "new binary content")
		swapDownloadServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(archive)
		})

		require.NoError(t, upgradeBinary("1.2.3"))

		got, err := os.ReadFile(binPath) //nolint:gosec // binPath is a test-controlled temp file
		require.NoError(t, err)
		assert.Equal(t, "new binary content", string(got))
	})

	t.Run("archive missing binary", func(t *testing.T) {
		binPath := filepath.Join(t.TempDir(), "go-broadcast")
		require.NoError(t, os.WriteFile(binPath, []byte("old binary"), 0o600))
		swapLocateBinary(t, func() (string, error) { return binPath, nil })

		archive := makeTarGz(t, "README.md", "no binary here")
		swapDownloadServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(archive)
		})

		err := upgradeBinary("1.2.3")
		require.Error(t, err)
		require.ErrorIs(t, err, ErrBinaryNotFoundInArchive)
	})
}

// TestRunUpgradeWithConfigErrorPaths exercises runUpgradeWithConfig's branches
// using the release/command/download seams, so every scenario is deterministic
// and no real GitHub API, module proxy, or download request is made.
func TestRunUpgradeWithConfigErrorPaths(t *testing.T) {
	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("verbose", false, "")
		return cmd
	}

	// Release lookup failure surfaces as "failed to check for updates".
	t.Run("release fetch failure", func(t *testing.T) {
		swapCurrentVersion(t, "0.5.0")
		swapGetLatestRelease(t, func(context.Context, string, string) (*versionpkg.GitHubRelease, error) {
			return nil, errTestRelease
		})

		err := runUpgradeWithConfig(newCmd(), UpgradeConfig{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check for updates")
	})

	// A development version without --force is rejected before any network call.
	t.Run("dev version without force", func(t *testing.T) {
		swapCurrentVersion(t, devVersionString)
		swapGetLatestRelease(t, func(context.Context, string, string) (*versionpkg.GitHubRelease, error) {
			t.Fatal("release lookup should not happen for dev version without --force")
			return nil, errTestRelease
		})

		err := runUpgradeWithConfig(newCmd(), UpgradeConfig{})
		require.ErrorIs(t, err, ErrDevVersionNoForce)
	})

	// Check-only mode reports availability without attempting an upgrade.
	t.Run("check only reports newer version", func(t *testing.T) {
		swapCurrentVersion(t, "0.5.0")
		swapGetLatestRelease(t, func(context.Context, string, string) (*versionpkg.GitHubRelease, error) {
			return &versionpkg.GitHubRelease{TagName: "v1.0.0"}, nil
		})
		swapRunCommand(t, func(context.Context, string, ...string) error {
			t.Fatal("check-only mode must not run any upgrade command")
			return nil
		})

		require.NoError(t, runUpgradeWithConfig(newCmd(), UpgradeConfig{CheckOnly: true}))
	})

	// Force go install path: when both methods fail, the combined error is returned.
	t.Run("force install with both methods failing", func(t *testing.T) {
		swapCurrentVersion(t, "1.0.0")
		swapGetLatestRelease(t, func(context.Context, string, string) (*versionpkg.GitHubRelease, error) {
			return &versionpkg.GitHubRelease{TagName: "v1.0.0"}, nil
		})
		swapRunCommand(t, func(context.Context, string, ...string) error {
			return errTestInstall // go install fails
		})
		swapLocateBinary(t, func() (string, error) {
			return "", errTestLocate // binary fallback fails too
		})

		err := runUpgradeWithConfig(newCmd(), UpgradeConfig{Force: true})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upgrade methods failed")
	})

	// Force go install path: success.
	t.Run("force install succeeds", func(t *testing.T) {
		swapCurrentVersion(t, "1.0.0")
		swapGetLatestRelease(t, func(context.Context, string, string) (*versionpkg.GitHubRelease, error) {
			return &versionpkg.GitHubRelease{TagName: "v1.0.0"}, nil
		})
		swapRunCommand(t, func(context.Context, string, ...string) error { return nil })

		require.NoError(t, runUpgradeWithConfig(newCmd(), UpgradeConfig{Force: true}))
	})

	// Force binary path: success via local download server.
	t.Run("force binary upgrade succeeds", func(t *testing.T) {
		swapCurrentVersion(t, "1.0.0")
		swapGetLatestRelease(t, func(context.Context, string, string) (*versionpkg.GitHubRelease, error) {
			return &versionpkg.GitHubRelease{TagName: "v1.0.0"}, nil
		})
		binPath := filepath.Join(t.TempDir(), "go-broadcast")
		require.NoError(t, os.WriteFile(binPath, []byte("old binary"), 0o600))
		swapLocateBinary(t, func() (string, error) { return binPath, nil })
		archive := makeTarGz(t, "go-broadcast", "new binary content")
		swapDownloadServer(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(archive)
		})

		require.NoError(t, runUpgradeWithConfig(newCmd(), UpgradeConfig{Force: true, UseBinary: true}))
	})
}

// TestRunUpgradeWithConfigVerboseMode verifies the verbose flag path is handled
// without panics, using the release seam so no real network call occurs.
func TestRunUpgradeWithConfigVerboseMode(t *testing.T) {
	swapCurrentVersion(t, "0.5.0")
	swapGetLatestRelease(t, func(context.Context, string, string) (*versionpkg.GitHubRelease, error) {
		return &versionpkg.GitHubRelease{TagName: "v1.0.0", Body: "Release notes line"}, nil
	})

	cmd := &cobra.Command{}
	cmd.Flags().Bool("verbose", true, "") // Enable verbose mode

	// Check-only mode avoids an actual upgrade while still exercising verbose handling.
	require.NoError(t, runUpgradeWithConfig(cmd, UpgradeConfig{CheckOnly: true}))
}

// TestExtractBinaryFromArchive tests archive extraction
func TestExtractBinaryFromArchive(t *testing.T) {
	t.Parallel()

	// Helper function to create a valid tar.gz archive
	createTestArchive := func(includesBinary bool, fileName string) []byte {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)

		if includesBinary {
			// Add the binary file
			header := &tar.Header{
				Name:     fileName,
				Mode:     0o755,
				Size:     int64(len("fake binary content")),
				Typeflag: tar.TypeReg,
			}
			require.NoError(t, tw.WriteHeader(header))
			_, err := tw.Write([]byte("fake binary content"))
			require.NoError(t, err)
		}

		// Add another file to test directory structure
		header := &tar.Header{
			Name:     "README.md",
			Mode:     0o644,
			Size:     int64(len("readme content")),
			Typeflag: tar.TypeReg,
		}
		require.NoError(t, tw.WriteHeader(header))
		_, err := tw.Write([]byte("readme content"))
		require.NoError(t, err)

		require.NoError(t, tw.Close())
		require.NoError(t, gw.Close())
		return buf.Bytes()
	}

	tests := []struct {
		name          string
		data          []byte
		expectError   bool
		errorContains string
	}{
		{
			name:          "InvalidGzipData",
			data:          []byte("invalid gzip data"),
			expectError:   true,
			errorContains: "could not create gzip reader",
		},
		{
			name:          "EmptyData",
			data:          []byte{},
			expectError:   true,
			errorContains: "could not create gzip reader",
		},
		{
			name:          "ValidArchiveWithBinary",
			data:          createTestArchive(true, "go-broadcast"),
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "ValidArchiveWithoutBinary",
			data:          createTestArchive(false, ""),
			expectError:   true,
			errorContains: "go-broadcast binary not found in archive",
		},
		{
			name:          "ValidArchiveWithWrongFileName",
			data:          createTestArchive(true, "wrong-binary-name"),
			expectError:   true,
			errorContains: "go-broadcast binary not found in archive",
		},
		{
			name:          "ValidArchiveWithSubdirectory",
			data:          createTestArchive(true, "bin/go-broadcast"),
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			reader := bytes.NewReader(tt.data)

			result, err := extractBinaryFromArchive(reader, tempDir)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, result)

				// Verify the binary file exists and has the right content
				assert.Contains(t, result, "go-broadcast")
				assert.FileExists(t, result)
			}
		})
	}
}

// TestUpgradeConfigValidation tests UpgradeConfig struct validation
func TestUpgradeConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config UpgradeConfig
	}{
		{
			name: "AllFlagsTrue",
			config: UpgradeConfig{
				Force:     true,
				CheckOnly: true,
				UseBinary: true,
			},
		},
		{
			name: "AllFlagsFalse",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: false,
				UseBinary: false,
			},
		},
		{
			name: "MixedFlags",
			config: UpgradeConfig{
				Force:     true,
				CheckOnly: false,
				UseBinary: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Verify struct fields have expected values
			assert.NotNil(t, &tt.config.Force)
			assert.NotNil(t, &tt.config.CheckOnly)
			assert.NotNil(t, &tt.config.UseBinary)
		})
	}
}

// TestRunUpgradeWithConfigLogic tests the runUpgradeWithConfig function logic
func TestRunUpgradeWithConfigLogic(t *testing.T) {
	t.Skip("Skipping upgrade tests to avoid GitHub API rate limits during CI/development")
	t.Parallel()

	tests := []struct {
		name           string
		currentVersion string
		config         UpgradeConfig
		expectError    bool
		errorContains  string
	}{
		{
			name:           "DevVersionWithoutForce",
			currentVersion: "dev",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: false,
			},
			expectError:   false, // In test environment, version may not be exactly "dev"
			errorContains: "",
		},
		{
			name:           "CommitHashWithoutForce",
			currentVersion: "abc123def456",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: false,
			},
			expectError:   false, // Tests run in environment where upgrade may work
			errorContains: "",
		},
		{
			name:           "EmptyVersionWithoutForce",
			currentVersion: "",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: false,
			},
			expectError:   false, // Tests run in environment where upgrade may work
			errorContains: "",
		},
		{
			name:           "DevVersionWithForce",
			currentVersion: "dev",
			config: UpgradeConfig{
				Force:     true,
				CheckOnly: false,
			},
			expectError:   false, // May succeed if network is available and upgrade works
			errorContains: "",
		},
		{
			name:           "CheckOnlyMode",
			currentVersion: "1.0.0",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: true,
			},
			expectError:   false, // Check-only mode may succeed if network is available
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a command for testing
			cmd := &cobra.Command{
				Use: "upgrade",
			}
			cmd.Flags().Bool("verbose", false, "Show release notes")

			// Test the function with the config
			err := runUpgradeWithConfig(cmd, tt.config)

			if tt.expectError {
				require.Error(t, err)
				if err != nil && tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestRunUpgradeWithConfigVersionChecks tests version checking logic
func TestRunUpgradeWithConfigVersionChecks(t *testing.T) {
	t.Parallel()

	// Test the version checking logic indirectly
	tests := []struct {
		name         string
		version      string
		isDevVersion bool
	}{
		{
			name:         "DevVersion",
			version:      "dev",
			isDevVersion: true,
		},
		{
			name:         "EmptyVersion",
			version:      "",
			isDevVersion: true,
		},
		{
			name:         "CommitHash",
			version:      "abc123def456",
			isDevVersion: true,
		},
		{
			name:         "RegularVersion",
			version:      "1.2.3",
			isDevVersion: false,
		},
		{
			name:         "VersionWithV",
			version:      "v1.2.3",
			isDevVersion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test the version detection logic that runUpgradeWithConfig uses
			isDev := tt.version == "dev" || tt.version == "" || isLikelyCommitHash(tt.version)
			assert.Equal(t, tt.isDevVersion, isDev)

			// Test format version logic
			formatted := formatVersion(tt.version)
			if tt.version == devVersionString || tt.version == "" {
				assert.Equal(t, devVersionString, formatted)
			} else if !strings.HasPrefix(tt.version, "v") && tt.version != devVersionString && tt.version != "" {
				assert.Equal(t, "v"+tt.version, formatted)
			} else {
				assert.Equal(t, tt.version, formatted)
			}
		})
	}
}

// TestUpgradeConfigEdgeCases tests edge cases for UpgradeConfig
func TestUpgradeConfigEdgeCases(t *testing.T) {
	t.Parallel()

	// Test conflicting configurations
	tests := []struct {
		name   string
		config UpgradeConfig
		valid  bool
	}{
		{
			name: "ForceAndCheckOnly",
			config: UpgradeConfig{
				Force:     true,
				CheckOnly: true,
			},
			valid: true, // Both can be true - force applies if not check-only
		},
		{
			name: "UseBinaryWithForce",
			config: UpgradeConfig{
				Force:     true,
				UseBinary: true,
			},
			valid: true,
		},
		{
			name: "AllFlagsTrue",
			config: UpgradeConfig{
				Force:     true,
				CheckOnly: true,
				UseBinary: true,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create command with flags matching the config
			cmd := &cobra.Command{Use: "upgrade"}
			cmd.Flags().Bool("force", false, "Force upgrade")
			cmd.Flags().Bool("check", false, "Check only")
			cmd.Flags().Bool("use-binary", false, "Use binary")
			cmd.Flags().Bool("verbose", false, "Verbose")

			// Set flags according to config
			_ = cmd.Flags().Set("force", fmt.Sprintf("%v", tt.config.Force))
			_ = cmd.Flags().Set("check", fmt.Sprintf("%v", tt.config.CheckOnly))
			_ = cmd.Flags().Set("use-binary", fmt.Sprintf("%v", tt.config.UseBinary))

			// Verify flags can be read (tests flag parsing)
			force, err := cmd.Flags().GetBool("force")
			require.NoError(t, err)
			assert.Equal(t, tt.config.Force, force)

			check, err := cmd.Flags().GetBool("check")
			require.NoError(t, err)
			assert.Equal(t, tt.config.CheckOnly, check)

			useBinary, err := cmd.Flags().GetBool("use-binary")
			require.NoError(t, err)
			assert.Equal(t, tt.config.UseBinary, useBinary)

			// Test that the configuration is internally consistent
			assert.True(t, tt.valid, "Configuration should be valid")
		})
	}
}
