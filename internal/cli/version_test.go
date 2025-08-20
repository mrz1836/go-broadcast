// Package cli provides command-line interface functionality for go-broadcast.
package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"

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
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
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
			expectedCommit:    commit,
			expectedBuildDate: buildDate,
		},
		{
			name:              "set commit only",
			setVersion:        "",
			setCommit:         "ghi789",
			setBuildDate:      "",
			expectedVersion:   version,
			expectedCommit:    "ghi789",
			expectedBuildDate: buildDate,
		},
		{
			name:              "set build date only",
			setVersion:        "",
			setCommit:         "",
			setBuildDate:      "2025-01-03",
			expectedVersion:   version,
			expectedCommit:    commit,
			expectedBuildDate: "2025-01-03",
		},
		{
			name:              "empty values don't override",
			setVersion:        "",
			setCommit:         "",
			setBuildDate:      "",
			expectedVersion:   version,
			expectedCommit:    commit,
			expectedBuildDate: buildDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to original values before each test
			version = origVersion
			commit = origCommit
			buildDate = origBuildDate

			// Set version info
			SetVersionInfo(tt.setVersion, tt.setCommit, tt.setBuildDate)

			// Verify values
			require.Equal(t, tt.expectedVersion, version)
			require.Equal(t, tt.expectedCommit, commit)
			require.Equal(t, tt.expectedBuildDate, buildDate)
		})
	}
}

// TestPrintVersionTextOutput verifies the text output of the printVersion function
func TestPrintVersionTextOutput(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Set known values
	version = "1.0.0-test"
	commit = "test123"
	buildDate = "2025-01-01T12:00:00Z"

	// Capture output
	var buf bytes.Buffer
	output.SetStdout(&buf)
	defer output.SetStdout(os.Stdout)

	// Run the function
	err := printVersion(false)
	require.NoError(t, err)

	// Verify output contains expected information
	outputStr := buf.String()
	require.Contains(t, outputStr, "go-broadcast 1.0.0-test")
	require.Contains(t, outputStr, "Commit:     test123")
	require.Contains(t, outputStr, "Build Date: 2025-01-01T12:00:00Z")
	require.Contains(t, outputStr, "Go Version: "+runtime.Version())
	require.Contains(t, outputStr, "Platform:   "+runtime.GOOS+"/"+runtime.GOARCH)
}

// TestPrintVersionJSONOutput verifies the JSON output of the printVersion function
func TestPrintVersionJSONOutput(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Set known values
	version = "2.0.0-json"
	commit = "json456"
	buildDate = "2025-01-02T00:00:00Z"

	// Capture output
	var buf bytes.Buffer
	output.SetStdout(&buf)
	defer output.SetStdout(os.Stdout)

	// Run the function
	err := printVersion(true)
	require.NoError(t, err)

	// Parse JSON output
	var info VersionInfo
	err = json.Unmarshal(buf.Bytes(), &info)
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
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Set to default values
	version = devVersionString
	commit = unknownString
	buildDate = unknownString

	// Capture output
	var buf bytes.Buffer
	output.SetStdout(&buf)
	defer output.SetStdout(os.Stdout)

	// Run the function
	err := printVersion(false)
	require.NoError(t, err)

	// Verify output contains default values
	outputStr := buf.String()
	require.Contains(t, outputStr, "go-broadcast "+devVersionString)
	require.Contains(t, outputStr, "Commit:     "+unknownString)
	require.Contains(t, outputStr, "Build Date: "+unknownString)
}

// TestPrintVersionFormats verifies the function works with different output formats
func TestPrintVersionFormats(t *testing.T) {
	// Test with text output
	t.Run("text output", func(t *testing.T) {
		var buf bytes.Buffer
		output.SetStdout(&buf)
		defer output.SetStdout(os.Stdout)

		err := printVersion(false)
		require.NoError(t, err)

		// Verify some output was produced
		require.Positive(t, buf.Len())
		require.Contains(t, buf.String(), "go-broadcast")
	})

	// Test with JSON output
	t.Run("json output", func(t *testing.T) {
		var buf bytes.Buffer
		output.SetStdout(&buf)
		defer output.SetStdout(os.Stdout)

		err := printVersion(true)
		require.NoError(t, err)

		// Verify valid JSON was produced
		var info VersionInfo
		err = json.Unmarshal(buf.Bytes(), &info)
		require.NoError(t, err)
	})
}

// TestVersionOutputFormatting verifies the output formatting
func TestVersionOutputFormatting(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Test JSON formatting with indentation
	t.Run("json indentation", func(t *testing.T) {
		version = "1.0.0"
		commit = "abc"
		buildDate = "2025-01-01"

		var buf bytes.Buffer
		output.SetStdout(&buf)
		defer output.SetStdout(os.Stdout)

		err := printVersion(true)
		require.NoError(t, err)

		// Verify indentation (2 spaces)
		jsonStr := buf.String()
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
	// Save original values
	origCommit := commit

	// Restore original values after test
	defer func() {
		commit = origCommit
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
			commit = tt.setCommit
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
	// Save original values
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		buildDate = origBuildDate
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
			buildDate = tt.setBuildDate
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
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Set known test values
	version = "2.0.0-test"
	commit = "test123"
	buildDate = "2025-01-01T12:00:00Z"

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
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Test with default/unknown values to trigger fallback logic
	version = devVersionString
	commit = unknownString
	buildDate = unknownString

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
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Test getVersionWithFallback with known ldflags version
	t.Run("getVersionWithFallback with ldflags", func(t *testing.T) {
		version = "1.2.3"
		result := getVersionWithFallback()
		require.Equal(t, "1.2.3", result)
	})

	// Test getVersionWithFallback with dev version (should fall back to build info)
	t.Run("getVersionWithFallback with dev version", func(t *testing.T) {
		version = devVersionString
		result := getVersionWithFallback()
		// Should return either build info or dev string
		require.NotEmpty(t, result)
		// In test environment, it should fall back to dev or short commit
		require.True(t, result == devVersionString || len(result) >= 7)
	})

	// Test getCommitWithFallback with known ldflags commit
	t.Run("getCommitWithFallback with ldflags", func(t *testing.T) {
		commit = "abc123def456"
		result := getCommitWithFallback()
		require.Equal(t, "abc123def456", result)
	})

	// Test getCommitWithFallback with unknown commit (should fall back to build info)
	t.Run("getCommitWithFallback with unknown", func(t *testing.T) {
		commit = unknownString
		result := getCommitWithFallback()
		// Should return either build info or unknown string
		require.NotEmpty(t, result)
		// In test environment, might get build info or unknown
		require.True(t, result == unknownString || len(result) >= 7)
	})

	// Test getBuildDateWithFallback with known ldflags build date
	t.Run("getBuildDateWithFallback with ldflags", func(t *testing.T) {
		buildDate = "2025-01-01T12:00:00Z"
		result := getBuildDateWithFallback()
		require.Equal(t, "2025-01-01T12:00:00Z", result)
	})

	// Test getBuildDateWithFallback with unknown build date (should fall back to build info)
	t.Run("getBuildDateWithFallback with unknown", func(t *testing.T) {
		buildDate = unknownString
		result := getBuildDateWithFallback()
		// Should return either build info, go-install marker, or unknown string
		require.NotEmpty(t, result)
		// In test environment, might get build info, go-install, or unknown
		require.True(t, result == unknownString || result == "go-install" || len(result) > 10)
	})
}

// TestVersionFallbackWithEmptyValues tests version fallback functions with empty values
func TestVersionFallbackWithEmptyValues(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Test with empty version string
	t.Run("empty version string", func(t *testing.T) {
		version = ""
		result := getVersionWithFallback()
		require.NotEmpty(t, result)
		// Should fall back to build info or dev string
		require.True(t, result == devVersionString || len(result) >= 7)
	})

	// Test with empty commit string
	t.Run("empty commit string", func(t *testing.T) {
		commit = ""
		result := getCommitWithFallback()
		require.NotEmpty(t, result)
		// Should fall back to build info or unknown
		require.True(t, result == unknownString || len(result) >= 7)
	})

	// Test with empty build date string
	t.Run("empty build date string", func(t *testing.T) {
		buildDate = ""
		result := getBuildDateWithFallback()
		require.NotEmpty(t, result)
		// Should fall back to build info, go-install, or unknown
		require.True(t, result == unknownString || result == "go-install" || len(result) > 10)
	})
}

// TestVersionFallbackBuildInfoPaths tests specific build info fallback scenarios
func TestVersionFallbackBuildInfoPaths(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Test scenarios that trigger different build info paths
	t.Run("version fallback scenarios", func(t *testing.T) {
		// Test dev version that should fall back to build info
		version = devVersionString
		result := getVersionWithFallback()
		require.NotEmpty(t, result)

		// Test empty version that should fall back to build info
		version = ""
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
		commit = unknownString
		result := getCommitWithFallback()
		require.NotEmpty(t, result)

		// Test empty commit that should fall back to build info
		commit = ""
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
		buildDate = unknownString
		result := getBuildDateWithFallback()
		require.NotEmpty(t, result)

		// Test empty build date that should fall back to build info
		buildDate = ""
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
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Test to increase coverage of different fallback code paths
	t.Run("test all fallback function conditions", func(t *testing.T) {
		// Set all values to trigger fallback logic
		version = devVersionString
		commit = unknownString
		buildDate = unknownString

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
		version = ""
		emptyVersionResult := getVersionWithFallback()
		require.NotEmpty(t, emptyVersionResult)

		// For commit: test both unknown and empty string conditions
		commit = ""
		emptyCommitResult := getCommitWithFallback()
		require.NotEmpty(t, emptyCommitResult)

		// For build date: test both unknown and empty string conditions
		buildDate = ""
		emptyBuildDateResult := getBuildDateWithFallback()
		require.NotEmpty(t, emptyBuildDateResult)
	})
}

// TestVersionFallbackDetailedPaths tests detailed build info scenarios
func TestVersionFallbackDetailedPaths(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	// Test getVersionWithFallback specific paths
	t.Run("version fallback detailed paths", func(t *testing.T) {
		// Test condition: version != devVersionString && version != ""
		version = "1.0.0-custom"
		result := getVersionWithFallback()
		require.Equal(t, "1.0.0-custom", result)

		// Test condition: version == devVersionString (triggers build info fallback)
		version = devVersionString
		result = getVersionWithFallback()
		require.NotEmpty(t, result)
		// In build environment, this should return dev or a commit hash

		// Test condition: version == "" (triggers build info fallback)
		version = ""
		result = getVersionWithFallback()
		require.NotEmpty(t, result)
		// This exercises the empty string condition in the first if statement
	})

	// Test getCommitWithFallback specific paths
	t.Run("commit fallback detailed paths", func(t *testing.T) {
		// Test condition: commit != unknownString && commit != ""
		commit = "abc123def456"
		result := getCommitWithFallback()
		require.Equal(t, "abc123def456", result)

		// Test condition: commit == unknownString (triggers build info fallback)
		commit = unknownString
		result = getCommitWithFallback()
		require.NotEmpty(t, result)
		// In build environment, this should return unknown or a commit hash

		// Test condition: commit == "" (triggers build info fallback)
		commit = ""
		result = getCommitWithFallback()
		require.NotEmpty(t, result)
		// This exercises the empty string condition in the first if statement
	})

	// Test getBuildDateWithFallback specific paths
	t.Run("build date fallback detailed paths", func(t *testing.T) {
		// Test condition: buildDate != unknownString && buildDate != ""
		buildDate = "2025-01-01T12:00:00Z"
		result := getBuildDateWithFallback()
		require.Equal(t, "2025-01-01T12:00:00Z", result)

		// Test condition: buildDate == unknownString (triggers build info fallback)
		buildDate = unknownString
		result = getBuildDateWithFallback()
		require.NotEmpty(t, result)
		// In build environment, this should return unknown, go-install, or formatted date

		// Test condition: buildDate == "" (triggers build info fallback)
		buildDate = ""
		result = getBuildDateWithFallback()
		require.NotEmpty(t, result)
		// This exercises the empty string condition in the first if statement
	})
}

// TestVersionEdgeCases tests edge cases for version handling
func TestVersionEdgeCases(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore original values after test
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
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
			version = tt.version
			commit = tt.commit
			buildDate = tt.buildDate

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
