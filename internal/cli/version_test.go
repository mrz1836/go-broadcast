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
	version = "dev"
	commit = "unknown"
	buildDate = "unknown"

	// Capture output
	var buf bytes.Buffer
	output.SetStdout(&buf)
	defer output.SetStdout(os.Stdout)

	// Run the function
	err := printVersion(false)
	require.NoError(t, err)

	// Verify output contains default values
	outputStr := buf.String()
	require.Contains(t, outputStr, "go-broadcast dev")
	require.Contains(t, outputStr, "Commit:     unknown")
	require.Contains(t, outputStr, "Build Date: unknown")
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
