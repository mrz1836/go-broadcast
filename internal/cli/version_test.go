// Package cli provides command-line interface functionality for go-broadcast.
package cli

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/output"
)

// TestVersionInfo verifies the VersionInfo struct
func TestVersionInfo(t *testing.T) {
	info := VersionInfo{
		Version:   "1.2.3",
		Commit:    "abc123",
		BuildDate: "2025-01-01",
		GoVersion: "go1.24",
		OS:        "linux",
		Arch:      "amd64",
	}

	require.Equal(t, "1.2.3", info.Version)
	require.Equal(t, "abc123", info.Commit)
	require.Equal(t, "2025-01-01", info.BuildDate)
	require.Equal(t, "go1.24", info.GoVersion)
	require.Equal(t, "linux", info.OS)
	require.Equal(t, "amd64", info.Arch)
}

// TestSetVersionInfo verifies that SetVersionInfo correctly updates version variables
func TestSetVersionInfo(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	tests := []struct {
		name              string
		setVersion        string
		setCommit         string
		setBuildDate      string
		expectedVersion   string
		expectedCommit    string
		expectedBuildDate string
	}{
		{
			name:              "set all values",
			setVersion:        "2.0.0",
			setCommit:         "def456",
			setBuildDate:      "2025-01-02",
			expectedVersion:   "2.0.0",
			expectedCommit:    "def456",
			expectedBuildDate: "2025-01-02",
		},
		{
			name:              "set version only",
			setVersion:        "3.0.0",
			setCommit:         "",
			setBuildDate:      "",
			expectedVersion:   "3.0.0",
			expectedCommit:    origCommit,
			expectedBuildDate: origBuildDate,
		},
		{
			name:              "set commit only",
			setVersion:        "",
			setCommit:         "ghi789",
			setBuildDate:      "",
			expectedVersion:   origVersion,
			expectedCommit:    "ghi789",
			expectedBuildDate: origBuildDate,
		},
		{
			name:              "set build date only",
			setVersion:        "",
			setCommit:         "",
			setBuildDate:      "2025-01-03",
			expectedVersion:   origVersion,
			expectedCommit:    origCommit,
			expectedBuildDate: "2025-01-03",
		},
		{
			name:              "empty values don't override",
			setVersion:        "",
			setCommit:         "",
			setBuildDate:      "",
			expectedVersion:   origVersion,
			expectedCommit:    origCommit,
			expectedBuildDate: origBuildDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to original values before each test (thread-safe)
			setVersion(origVersion)
			setCommit(origCommit)
			setBuildDate(origBuildDate)

			// Set version info
			SetVersionInfo(tt.setVersion, tt.setCommit, tt.setBuildDate)

			// Verify values (thread-safe)
			require.Equal(t, tt.expectedVersion, getVersionRaw())
			require.Equal(t, tt.expectedCommit, getCommitRaw())
			require.Equal(t, tt.expectedBuildDate, getBuildDateRaw())
		})
	}
}

// TestPrintVersionTextOutput verifies the text output of the printVersion function
func TestPrintVersionTextOutput(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Set known values (thread-safe)
	setVersion("1.0.0-test")
	setCommit("test123")
	setBuildDate("2025-01-01T12:00:00Z")

	// Capture output (thread-safe)
	scope := output.CaptureOutput()
	defer scope.Restore()

	// Run the function
	err := printVersion(false)
	require.NoError(t, err)

	// Verify output contains expected information
	outputStr := scope.Stdout.String()
	require.Contains(t, outputStr, "go-broadcast 1.0.0-test")
	require.Contains(t, outputStr, "Commit:     test123")
	require.Contains(t, outputStr, "Build Date: 2025-01-01T12:00:00Z")
	require.Contains(t, outputStr, "Go Version: "+runtime.Version())
	require.Contains(t, outputStr, "Platform:   "+runtime.GOOS+"/"+runtime.GOARCH)
}

// TestPrintVersionJSONOutput verifies the JSON output of the printVersion function
func TestPrintVersionJSONOutput(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Set known values (thread-safe)
	setVersion("2.0.0-json")
	setCommit("json456")
	setBuildDate("2025-01-02T00:00:00Z")

	// Capture output (thread-safe)
	scope := output.CaptureOutput()
	defer scope.Restore()

	// Run the function
	err := printVersion(true)
	require.NoError(t, err)

	// Parse JSON output
	var info VersionInfo
	err = json.Unmarshal(scope.Stdout.Bytes(), &info)
	require.NoError(t, err)

	// Verify JSON data
	require.Equal(t, "2.0.0-json", info.Version)
	require.Equal(t, "json456", info.Commit)
	require.Equal(t, "2025-01-02T00:00:00Z", info.BuildDate)
	require.Equal(t, runtime.Version(), info.GoVersion)
	require.Equal(t, runtime.GOOS, info.OS)
	require.Equal(t, runtime.GOARCH, info.Arch)
}

// TestVersionInfoJSON verifies JSON marshaling of VersionInfo
func TestVersionInfoJSON(t *testing.T) {
	info := VersionInfo{
		Version:   "1.2.3",
		Commit:    "abc123",
		BuildDate: "2025-01-01",
		GoVersion: "go1.24",
		OS:        "linux",
		Arch:      "amd64",
	}

	// Marshal to JSON
	data, err := json.Marshal(info)
	require.NoError(t, err)

	// Verify JSON contains expected fields
	jsonStr := string(data)
	require.Contains(t, jsonStr, `"version":"1.2.3"`)
	require.Contains(t, jsonStr, `"commit":"abc123"`)
	require.Contains(t, jsonStr, `"build_date":"2025-01-01"`)
	require.Contains(t, jsonStr, `"go_version":"go1.24"`)
	require.Contains(t, jsonStr, `"os":"linux"`)
	require.Contains(t, jsonStr, `"arch":"amd64"`)

	// Unmarshal back
	var decoded VersionInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, info, decoded)
}

// TestPrintVersionDefaultValues verifies behavior with default build values
func TestPrintVersionDefaultValues(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Set to default values (thread-safe)
	setVersion(devVersionString)
	setCommit(unknownString)
	setBuildDate(unknownString)

	// Capture output (thread-safe)
	scope := output.CaptureOutput()
	defer scope.Restore()

	// Run the function
	err := printVersion(false)
	require.NoError(t, err)

	// Verify output contains default values
	outputStr := scope.Stdout.String()
	require.Contains(t, outputStr, "go-broadcast "+devVersionString)
	require.Contains(t, outputStr, "Commit:     "+unknownString)
	require.Contains(t, outputStr, "Build Date: "+unknownString)
}

// TestPrintVersionFormats verifies the function works with different output formats
func TestPrintVersionFormats(t *testing.T) {
	// Test with text output
	t.Run("text output", func(t *testing.T) {
		scope := output.CaptureOutput()
		defer scope.Restore()

		err := printVersion(false)
		require.NoError(t, err)

		// Verify some output was produced
		require.Positive(t, scope.Stdout.Len())
		require.Contains(t, scope.Stdout.String(), "go-broadcast")
	})

	// Test with JSON output
	t.Run("json output", func(t *testing.T) {
		scope := output.CaptureOutput()
		defer scope.Restore()

		err := printVersion(true)
		require.NoError(t, err)

		// Verify valid JSON was produced
		var info VersionInfo
		err = json.Unmarshal(scope.Stdout.Bytes(), &info)
		require.NoError(t, err)
	})
}

// TestVersionOutputFormatting verifies the output formatting
func TestVersionOutputFormatting(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Test JSON formatting with indentation
	t.Run("json indentation", func(t *testing.T) {
		setVersion("1.0.0")
		setCommit("abc")
		setBuildDate("2025-01-01")

		scope := output.CaptureOutput()
		defer scope.Restore()

		err := printVersion(true)
		require.NoError(t, err)

		// Verify indentation (2 spaces)
		jsonStr := scope.Stdout.String()
		lines := strings.Split(jsonStr, "\n")

		// Check that fields are indented
		for _, line := range lines {
			if strings.Contains(line, `"version"`) ||
				strings.Contains(line, `"commit"`) ||
				strings.Contains(line, `"build_date"`) {
				require.True(t, strings.HasPrefix(line, "  "), "JSON fields should be indented with 2 spaces")
			}
		}
	})
}

// TestGetCommit tests the GetCommit function
func TestGetCommit(t *testing.T) {
	// Save original values (thread-safe)
	origCommit := getCommitRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setCommit(origCommit)
	}()

	tests := []struct {
		name           string
		setCommit      string
		expectedResult string
	}{
		{
			name:           "with set commit",
			setCommit:      "abc123def456",
			expectedResult: "abc123def456",
		},
		{
			name:           "with short commit",
			setCommit:      "abc123",
			expectedResult: "abc123",
		},
		{
			name:           "with unknown commit",
			setCommit:      unknownString,
			expectedResult: "abc123d", // Will use fallback from build info if available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setCommit(tt.setCommit)
			result := GetCommit()

			if tt.setCommit == unknownString {
				// For unknown, we expect either build info fallback or "unknown"
				require.NotEmpty(t, result)
			} else {
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

// TestGetBuildDate tests the GetBuildDate function
func TestGetBuildDate(t *testing.T) {
	// Save original values (thread-safe)
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setBuildDate(origBuildDate)
	}()

	tests := []struct {
		name           string
		setBuildDate   string
		expectedResult string
	}{
		{
			name:           "with set build date",
			setBuildDate:   "2025-01-01T12:00:00Z",
			expectedResult: "2025-01-01T12:00:00Z",
		},
		{
			name:           "with formatted date",
			setBuildDate:   "2025-01-02_15:30:45_UTC",
			expectedResult: "2025-01-02_15:30:45_UTC",
		},
		{
			name:           "with unknown build date",
			setBuildDate:   unknownString,
			expectedResult: unknownString, // Will use fallback from build info if available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setBuildDate(tt.setBuildDate)
			result := GetBuildDate()

			if tt.setBuildDate == unknownString {
				// For unknown, we expect either build info fallback or "unknown"
				require.NotEmpty(t, result)
			} else {
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

// TestGetVersionInfoStruct tests the GetVersionInfo function
func TestGetVersionInfoStruct(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Set known test values (thread-safe)
	setVersion("2.0.0-test")
	setCommit("test123")
	setBuildDate("2025-01-01T12:00:00Z")

	info := GetVersionInfo()

	// Verify all fields are populated
	require.Equal(t, "2.0.0-test", info.Version)
	require.Equal(t, "test123", info.Commit)
	require.Equal(t, "2025-01-01T12:00:00Z", info.BuildDate)
	require.Equal(t, runtime.Version(), info.GoVersion)
	require.Equal(t, runtime.GOOS, info.OS)
	require.Equal(t, runtime.GOARCH, info.Arch)

	// Verify struct fields have correct json tags
	require.IsType(t, VersionInfo{}, info)
}

// TestVersionFallbackBehavior tests version fallback with default values
func TestVersionFallbackBehavior(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Test with default/unknown values to trigger fallback logic (thread-safe)
	setVersion(devVersionString)
	setCommit(unknownString)
	setBuildDate(unknownString)

	// Test getVersionWithFallback
	versionResult := getVersionWithFallback()
	require.NotEmpty(t, versionResult, "Version should not be empty")

	// Test getCommitWithFallback
	commitResult := getCommitWithFallback()
	require.NotEmpty(t, commitResult, "Commit should not be empty")

	// Test getBuildDateWithFallback
	buildDateResult := getBuildDateWithFallback()
	require.NotEmpty(t, buildDateResult, "Build date should not be empty")
}

// TestVersionFallbackFunctions tests version fallback functions with mocked build info
func TestVersionFallbackFunctions(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Test getVersionWithFallback with known ldflags version
	t.Run("getVersionWithFallback with ldflags", func(t *testing.T) {
		setVersion("1.2.3")
		result := getVersionWithFallback()
		require.Equal(t, "1.2.3", result)
	})

	// Test getVersionWithFallback with dev version (should fall back to build info)
	t.Run("getVersionWithFallback with dev version", func(t *testing.T) {
		setVersion(devVersionString)
		result := getVersionWithFallback()
		// Should return either build info or dev string
		require.NotEmpty(t, result)
		// In test environment, it should fall back to dev or short commit
		require.True(t, result == devVersionString || len(result) >= 7)
	})

	// Test getCommitWithFallback with known ldflags commit
	t.Run("getCommitWithFallback with ldflags", func(t *testing.T) {
		setCommit("abc123def456")
		result := getCommitWithFallback()
		require.Equal(t, "abc123def456", result)
	})

	// Test getCommitWithFallback with unknown commit (should fall back to build info)
	t.Run("getCommitWithFallback with unknown", func(t *testing.T) {
		setCommit(unknownString)
		result := getCommitWithFallback()
		// Should return either build info or unknown string
		require.NotEmpty(t, result)
		// In test environment, might get build info or unknown
		require.True(t, result == unknownString || len(result) >= 7)
	})

	// Test getBuildDateWithFallback with known ldflags build date
	t.Run("getBuildDateWithFallback with ldflags", func(t *testing.T) {
		setBuildDate("2025-01-01T12:00:00Z")
		result := getBuildDateWithFallback()
		require.Equal(t, "2025-01-01T12:00:00Z", result)
	})

	// Test getBuildDateWithFallback with unknown build date (should fall back to build info)
	t.Run("getBuildDateWithFallback with unknown", func(t *testing.T) {
		setBuildDate(unknownString)
		result := getBuildDateWithFallback()
		// Should return either build info, go-install marker, or unknown string
		require.NotEmpty(t, result)
		// In test environment, might get build info, go-install, or unknown
		require.True(t, result == unknownString || result == "go-install" || len(result) > 10)
	})
}

// TestVersionFallbackWithEmptyValues tests version fallback functions with empty values
func TestVersionFallbackWithEmptyValues(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Test with empty version string
	t.Run("empty version string", func(t *testing.T) {
		setVersion("")
		result := getVersionWithFallback()
		require.NotEmpty(t, result)
		// Should fall back to build info or dev string
		require.True(t, result == devVersionString || len(result) >= 7)
	})

	// Test with empty commit string
	t.Run("empty commit string", func(t *testing.T) {
		setCommit("")
		result := getCommitWithFallback()
		require.NotEmpty(t, result)
		// Should fall back to build info or unknown
		require.True(t, result == unknownString || len(result) >= 7)
	})

	// Test with empty build date string
	t.Run("empty build date string", func(t *testing.T) {
		setBuildDate("")
		result := getBuildDateWithFallback()
		require.NotEmpty(t, result)
		// Should fall back to build info, go-install, or unknown
		require.True(t, result == unknownString || result == "go-install" || len(result) > 10)
	})
}

// TestVersionFallbackBuildInfoPaths tests specific build info fallback scenarios
func TestVersionFallbackBuildInfoPaths(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Test scenarios that trigger different build info paths
	t.Run("version fallback scenarios", func(t *testing.T) {
		// Test dev version that should fall back to build info
		setVersion(devVersionString)
		result := getVersionWithFallback()
		require.NotEmpty(t, result)

		// Test empty version that should fall back to build info
		setVersion("")
		result2 := getVersionWithFallback()
		require.NotEmpty(t, result2)

		// In test environment, build info fallback should work
		// The result should be either dev string or a commit hash from build info
		if result != devVersionString {
			// Should be a commit hash from VCS info
			require.GreaterOrEqual(t, len(result), 7, "Expected commit hash or dev string, got: %s", result)
		}
	})

	t.Run("commit fallback scenarios", func(t *testing.T) {
		// Test unknown commit that should fall back to build info
		setCommit(unknownString)
		result := getCommitWithFallback()
		require.NotEmpty(t, result)

		// Test empty commit that should fall back to build info
		setCommit("")
		result2 := getCommitWithFallback()
		require.NotEmpty(t, result2)

		// In test environment, build info fallback should work
		// The result should be either unknown string or a commit hash from build info
		if result != unknownString {
			// Should be a commit hash from VCS info
			require.GreaterOrEqual(t, len(result), 7, "Expected commit hash or unknown string, got: %s", result)
		}
	})

	t.Run("build date fallback scenarios", func(t *testing.T) {
		// Test unknown build date that should fall back to build info
		setBuildDate(unknownString)
		result := getBuildDateWithFallback()
		require.NotEmpty(t, result)

		// Test empty build date that should fall back to build info
		setBuildDate("")
		result2 := getBuildDateWithFallback()
		require.NotEmpty(t, result2)

		// In test environment, build info fallback should work
		// The result should be unknown, go-install marker, or formatted date
		if result != unknownString && result != "go-install" {
			// Should be a formatted date from VCS time
			require.Greater(t, len(result), 10, "Expected formatted date, go-install, or unknown, got: %s", result)
		}
	})
}

// TestVersionWithBuildInfoPathCoverage tests additional build info paths for coverage
func TestVersionWithBuildInfoPathCoverage(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Test to increase coverage of different fallback code paths
	t.Run("test all fallback function conditions", func(t *testing.T) {
		// Set all values to trigger fallback logic (thread-safe)
		setVersion(devVersionString)
		setCommit(unknownString)
		setBuildDate(unknownString)

		// Call all fallback functions to exercise their code paths
		versionResult := getVersionWithFallback()
		commitResult := getCommitWithFallback()
		buildDateResult := getBuildDateWithFallback()

		// Verify all functions return non-empty values
		require.NotEmpty(t, versionResult, "getVersionWithFallback should not return empty")
		require.NotEmpty(t, commitResult, "getCommitWithFallback should not return empty")
		require.NotEmpty(t, buildDateResult, "getBuildDateWithFallback should not return empty")

		// Test the various conditions in each function
		// For version: test both dev and empty string conditions
		setVersion("")
		emptyVersionResult := getVersionWithFallback()
		require.NotEmpty(t, emptyVersionResult)

		// For commit: test both unknown and empty string conditions
		setCommit("")
		emptyCommitResult := getCommitWithFallback()
		require.NotEmpty(t, emptyCommitResult)

		// For build date: test both unknown and empty string conditions
		setBuildDate("")
		emptyBuildDateResult := getBuildDateWithFallback()
		require.NotEmpty(t, emptyBuildDateResult)
	})
}

// TestVersionFallbackDetailedPaths tests detailed build info scenarios
func TestVersionFallbackDetailedPaths(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Test getVersionWithFallback specific paths
	t.Run("version fallback detailed paths", func(t *testing.T) {
		// Test condition: version != devVersionString && version != ""
		setVersion("1.0.0-custom")
		result := getVersionWithFallback()
		require.Equal(t, "1.0.0-custom", result)

		// Test condition: version == devVersionString (triggers build info fallback)
		setVersion(devVersionString)
		result = getVersionWithFallback()
		require.NotEmpty(t, result)
		// In build environment, this should return dev or a commit hash

		// Test condition: version == "" (triggers build info fallback)
		setVersion("")
		result = getVersionWithFallback()
		require.NotEmpty(t, result)
		// This exercises the empty string condition in the first if statement
	})

	// Test getCommitWithFallback specific paths
	t.Run("commit fallback detailed paths", func(t *testing.T) {
		// Test condition: commit != unknownString && commit != ""
		setCommit("abc123def456")
		result := getCommitWithFallback()
		require.Equal(t, "abc123def456", result)

		// Test condition: commit == unknownString (triggers build info fallback)
		setCommit(unknownString)
		result = getCommitWithFallback()
		require.NotEmpty(t, result)
		// In build environment, this should return unknown or a commit hash

		// Test condition: commit == "" (triggers build info fallback)
		setCommit("")
		result = getCommitWithFallback()
		require.NotEmpty(t, result)
		// This exercises the empty string condition in the first if statement
	})

	// Test getBuildDateWithFallback specific paths
	t.Run("build date fallback detailed paths", func(t *testing.T) {
		// Test condition: buildDate != unknownString && buildDate != ""
		setBuildDate("2025-01-01T12:00:00Z")
		result := getBuildDateWithFallback()
		require.Equal(t, "2025-01-01T12:00:00Z", result)

		// Test condition: buildDate == unknownString (triggers build info fallback)
		setBuildDate(unknownString)
		result = getBuildDateWithFallback()
		require.NotEmpty(t, result)
		// In build environment, this should return unknown, go-install, or formatted date

		// Test condition: buildDate == "" (triggers build info fallback)
		setBuildDate("")
		result = getBuildDateWithFallback()
		require.NotEmpty(t, result)
		// This exercises the empty string condition in the first if statement
	})
}

// TestVersionEdgeCases tests edge cases for version handling
func TestVersionEdgeCases(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	tests := []struct {
		name      string
		version   string
		commit    string
		buildDate string
	}{
		{
			name:      "empty values",
			version:   "",
			commit:    "",
			buildDate: "",
		},
		{
			name:      "whitespace values",
			version:   "   ",
			commit:    "   ",
			buildDate: "   ",
		},
		{
			name:      "special characters",
			version:   "v1.0.0-beta+build.123",
			commit:    "abc123-dirty",
			buildDate: "2025-01-01T12:00:00+00:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setVersion(tt.version)
			setCommit(tt.commit)
			setBuildDate(tt.buildDate)

			// Test that functions don't panic with edge case values
			versionResult := GetVersion()
			commitResult := GetCommit()
			buildDateResult := GetBuildDate()
			info := GetVersionInfo()

			// Verify all return non-empty values
			require.NotEmpty(t, versionResult)
			require.NotEmpty(t, commitResult)
			require.NotEmpty(t, buildDateResult)
			require.NotEmpty(t, info.Version)
			require.NotEmpty(t, info.Commit)
			require.NotEmpty(t, info.BuildDate)
		})
	}
}

// TestResetVersionInfo verifies that ResetVersionInfo restores default values
func TestResetVersionInfo(t *testing.T) {
	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	t.Run("resets from custom values to defaults", func(t *testing.T) {
		// Set custom values
		SetVersionInfo("9.8.7", "custom_commit_hash", "2099-12-31T23:59:59Z")

		// Verify custom values are set
		require.Equal(t, "9.8.7", getVersionRaw())
		require.Equal(t, "custom_commit_hash", getCommitRaw())
		require.Equal(t, "2099-12-31T23:59:59Z", getBuildDateRaw())

		// Reset
		ResetVersionInfo()

		// Verify defaults are restored
		assert.Equal(t, devVersionString, getVersionRaw())
		assert.Equal(t, unknownString, getCommitRaw())
		assert.Equal(t, unknownString, getBuildDateRaw())
	})

	t.Run("reset is idempotent", func(t *testing.T) {
		// Reset twice in a row
		ResetVersionInfo()
		ResetVersionInfo()

		assert.Equal(t, devVersionString, getVersionRaw())
		assert.Equal(t, unknownString, getCommitRaw())
		assert.Equal(t, unknownString, getBuildDateRaw())
	})

	t.Run("set then reset then verify getters use fallback", func(t *testing.T) {
		// Set known values
		SetVersionInfo("5.0.0", "abc1234", "2025-06-15")

		// Verify direct getters return set values
		assert.Equal(t, "5.0.0", GetVersion())
		assert.Equal(t, "abc1234", GetCommit())
		assert.Equal(t, "2025-06-15", GetBuildDate())

		// Reset to defaults
		ResetVersionInfo()

		// After reset, GetVersion/GetCommit/GetBuildDate use fallback logic.
		// The raw values should be the defaults.
		assert.Equal(t, devVersionString, getVersionRaw())
		assert.Equal(t, unknownString, getCommitRaw())
		assert.Equal(t, unknownString, getBuildDateRaw())

		// The public getters may return build info fallback, but should not be empty
		assert.NotEmpty(t, GetVersion())
		assert.NotEmpty(t, GetCommit())
		assert.NotEmpty(t, GetBuildDate())
	})
}

// TestGetVersion verifies GetVersion returns expected values
func TestGetVersion(t *testing.T) {
	t.Parallel()

	// Save original values (thread-safe)
	origVersion := getVersionRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
	}()

	t.Run("returns set version when explicitly set", func(t *testing.T) {
		setVersion("4.2.0-rc1")
		result := GetVersion()
		assert.Equal(t, "4.2.0-rc1", result)
	})

	t.Run("returns non-empty with dev default", func(t *testing.T) {
		setVersion(devVersionString)
		result := GetVersion()
		assert.NotEmpty(t, result)
		// Should be either devVersionString or a commit hash from build info
		assert.True(t, result == devVersionString || len(result) >= 7,
			"Expected dev string or commit hash, got: %s", result)
	})

	t.Run("returns non-empty with empty string", func(t *testing.T) {
		setVersion("")
		result := GetVersion()
		assert.NotEmpty(t, result)
	})

	t.Run("preserves semver format", func(t *testing.T) {
		setVersion("v1.2.3")
		result := GetVersion()
		assert.Equal(t, "v1.2.3", result)
	})
}

// TestSetVersionInfoAndReset tests the full lifecycle: set, verify, reset, verify
func TestSetVersionInfoAndReset(t *testing.T) {
	t.Parallel()

	// Save original values (thread-safe)
	origVersion := getVersionRaw()
	origCommit := getCommitRaw()
	origBuildDate := getBuildDateRaw()

	// Restore original values after test (thread-safe)
	defer func() {
		setVersion(origVersion)
		setCommit(origCommit)
		setBuildDate(origBuildDate)
	}()

	// Set custom values
	SetVersionInfo("10.0.0", "deadbeef1234", "2026-01-01T00:00:00Z")

	// Verify custom values through public API
	assert.Equal(t, "10.0.0", GetVersion())
	assert.Equal(t, "deadbeef1234", GetCommit())
	assert.Equal(t, "2026-01-01T00:00:00Z", GetBuildDate())

	// Verify GetVersionInfo struct reflects custom values
	info := GetVersionInfo()
	assert.Equal(t, "10.0.0", info.Version)
	assert.Equal(t, "deadbeef1234", info.Commit)
	assert.Equal(t, "2026-01-01T00:00:00Z", info.BuildDate)
	assert.Equal(t, runtime.Version(), info.GoVersion)
	assert.Equal(t, runtime.GOOS, info.OS)
	assert.Equal(t, runtime.GOARCH, info.Arch)

	// Reset and verify defaults
	ResetVersionInfo()
	assert.Equal(t, devVersionString, getVersionRaw())
	assert.Equal(t, unknownString, getCommitRaw())
	assert.Equal(t, unknownString, getBuildDateRaw())
}
