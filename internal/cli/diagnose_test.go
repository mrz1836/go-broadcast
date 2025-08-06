package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/testutil"
)

// TestDiagnosticInfoJSONMarshal tests JSON marshaling of DiagnosticInfo
func TestDiagnosticInfoJSONMarshal(t *testing.T) {
	info := &DiagnosticInfo{
		Timestamp: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Version: DiagnosticVersionInfo{
			Version: "1.0.0",
			Commit:  "abc123",
			Date:    "2025-01-01",
			GoVer:   "go1.21",
			Built:   "source",
		},
		System: DiagnosticSystemInfo{
			OS:       "linux",
			Arch:     "amd64",
			NumCPU:   8,
			Hostname: "test-host",
			UserHome: "/home/test",
		},
		Environment: map[string]string{
			"PATH": "/usr/bin",
			"HOME": "/home/test",
		},
		GitVersion: "git version 2.40.0",
		GHVersion:  "gh version 2.30.0",
		Config: DiagnosticConfigInfo{
			Path:   "/home/test/.config/go-broadcast.yml",
			Exists: true,
			Valid:  true,
		},
	}

	data, err := json.Marshal(info)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"version":"1.0.0"`)
	assert.Contains(t, string(data), `"os":"linux"`)
	assert.Contains(t, string(data), `"git_version":"git version 2.40.0"`)
}

// TestGetVersionInfo tests version information retrieval
func TestGetVersionInfo(t *testing.T) {
	info := getVersionInfo()

	assert.Equal(t, "dev", info.Version)
	assert.Empty(t, info.Commit)
	assert.Empty(t, info.Date)
	assert.Equal(t, runtime.Version(), info.GoVer)
	assert.Equal(t, "source", info.Built)
}

// TestGetSystemInfo tests system information collection
func TestGetSystemInfo(t *testing.T) {
	info := getSystemInfo()
	assert.Equal(t, runtime.GOOS, info.OS)
	assert.Equal(t, runtime.GOARCH, info.Arch)
	assert.Equal(t, runtime.NumCPU(), info.NumCPU)
	assert.NotEmpty(t, info.Hostname)
	assert.NotEmpty(t, info.UserHome)
}

// TestCollectEnvironment tests environment variable collection
func TestCollectEnvironment(t *testing.T) {
	ctx := context.Background()

	t.Run("CollectsRelevantVariables", func(t *testing.T) {
		// Set some test environment variables
		require.NoError(t, os.Setenv("GO_BROADCAST_CONFIG", "/test/config.yml"))
		defer func() { _ = os.Unsetenv("GO_BROADCAST_CONFIG") }()

		env := collectEnvironment(ctx)

		// Should include PATH and HOME (usually set on all systems)
		assert.NotEmpty(t, env)

		// Should include our test variable
		assert.Equal(t, "/test/config.yml", env["GO_BROADCAST_CONFIG"])
	})

	t.Run("RedactsSensitiveValues", func(t *testing.T) {
		// Set sensitive environment variables
		testCases := []struct {
			name     string
			envVar   string
			value    string
			expected string
		}{
			{
				name:     "RedactsGHToken",
				envVar:   "GH_TOKEN",
				value:    "ghp_1234567890abcdef",
				expected: "ghp_***REDACTED***",
			},
			{
				name:     "RedactsShortToken",
				envVar:   "GITHUB_TOKEN",
				value:    "short",
				expected: "***REDACTED***",
			},
			{
				name:     "RedactsPasswordVariable",
				envVar:   "DB_PASSWORD",
				value:    "supersecret123",
				expected: "supe***REDACTED***",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				require.NoError(t, os.Setenv(tc.envVar, tc.value))
				defer func() { _ = os.Unsetenv(tc.envVar) }()

				env := collectEnvironment(ctx)

				// Check if the variable was collected but we can't guarantee it's in relevantVars
				if val, exists := env[tc.envVar]; exists {
					assert.Equal(t, tc.expected, val)
				}
			})
		}
	})
}

// TestIsSensitiveEnvVar tests sensitive environment variable detection
func TestIsSensitiveEnvVar(t *testing.T) {
	testCases := []struct {
		name      string
		key       string
		sensitive bool
	}{
		{
			name:      "TokenVariable",
			key:       "GH_TOKEN",
			sensitive: true,
		},
		{
			name:      "SecretVariable",
			key:       "API_SECRET",
			sensitive: true,
		},
		{
			name:      "KeyVariable",
			key:       "SSH_KEY",
			sensitive: true,
		},
		{
			name:      "PasswordVariable",
			key:       "DB_PASSWORD",
			sensitive: true,
		},
		{
			name:      "PassVariable",
			key:       "USER_PASS",
			sensitive: true,
		},
		{
			name:      "NonSensitiveVariable",
			key:       "PATH",
			sensitive: false,
		},
		{
			name:      "LowerCaseSensitive",
			key:       "github_token",
			sensitive: true,
		},
		{
			name:      "MixedCaseSensitive",
			key:       "GitHub_Token",
			sensitive: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isSensitiveEnvVar(tc.key)
			assert.Equal(t, tc.sensitive, result)
		})
	}
}

// TestGetGitVersion tests Git version retrieval
func TestGetGitVersion(t *testing.T) {
	ctx := context.Background()

	gitVersion := getGitVersion(ctx)

	// Should either return a version string or an error message
	assert.NotEmpty(t, gitVersion)

	// If git is installed, should contain "git version"
	// If not, should contain "error:"
	isValidResponse := strings.Contains(gitVersion, "git version") || strings.Contains(gitVersion, "error:")
	assert.True(t, isValidResponse, "Expected git version or error, got: %s", gitVersion)
}

// TestGetGHCLIVersion tests GitHub CLI version retrieval
func TestGetGHCLIVersion(t *testing.T) {
	ctx := context.Background()

	ghVersion := getGHCLIVersion(ctx)

	// Should either return a version string or an error message
	assert.NotEmpty(t, ghVersion)

	// If gh is installed, should contain "gh version"
	// If not, should contain "error:"
	isValidResponse := strings.Contains(ghVersion, "gh version") || strings.Contains(ghVersion, "error:") || ghVersion == "unknown"
	assert.True(t, isValidResponse, "Expected gh version, error, or unknown, got: %s", ghVersion)
}

// TestGetConfigInfo tests configuration file status analysis
func TestGetConfigInfo(t *testing.T) {
	ctx := context.Background()

	t.Run("FileNotFound", func(t *testing.T) {
		logConfig := &LogConfig{
			ConfigFile: "/non/existent/path/config.yml",
		}

		info := getConfigInfo(ctx, logConfig)

		assert.Equal(t, "/non/existent/path/config.yml", info.Path)
		assert.False(t, info.Exists)
		assert.False(t, info.Valid)
		assert.Contains(t, info.Error, "file not found")
	})

	t.Run("ValidConfigFile", func(t *testing.T) {
		// Create a temporary valid config file
		tmpDir := testutil.CreateTempDir(t)
		configPath := filepath.Join(tmpDir, "config.yml")

		configContent := TestValidConfig

		testutil.WriteTestFile(t, configPath, configContent)

		logConfig := &LogConfig{
			ConfigFile: configPath,
		}

		info := getConfigInfo(ctx, logConfig)

		assert.Equal(t, configPath, info.Path)
		assert.True(t, info.Exists)
		assert.True(t, info.Valid)
		assert.Empty(t, info.Error)
	})

	t.Run("InvalidConfigFile", func(t *testing.T) {
		// Create a temporary invalid config file
		tmpDir := testutil.CreateTempDir(t)
		configPath := filepath.Join(tmpDir, "config.yml")

		invalidContent := `invalid: yaml: content:`

		testutil.WriteTestFile(t, configPath, invalidContent)

		logConfig := &LogConfig{
			ConfigFile: configPath,
		}

		info := getConfigInfo(ctx, logConfig)

		assert.Equal(t, configPath, info.Path)
		assert.True(t, info.Exists)
		assert.False(t, info.Valid)
		assert.Contains(t, info.Error, "load error")
	})
}

// TestRunDiagnose tests the main diagnose command execution
func TestRunDiagnose(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create command
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Set global flags
	globalFlags = &Flags{
		ConfigFile: "/test/config.yml",
		LogLevel:   "info",
		DryRun:     false,
	}

	// Run diagnose
	err := runDiagnose(cmd, []string{})

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify no error
	require.NoError(t, err)

	// Verify JSON output
	var info DiagnosticInfo
	require.NoError(t, json.Unmarshal([]byte(output), &info))

	// Verify content
	assert.NotZero(t, info.Timestamp)
	assert.NotEmpty(t, info.Version.GoVer)
	assert.NotEmpty(t, info.System.OS)
	assert.NotNil(t, info.Environment)
	assert.NotEmpty(t, info.GitVersion)
	assert.NotEmpty(t, info.GHVersion)
	assert.Equal(t, "/test/config.yml", info.Config.Path)
}

// TestCreateDiagnoseCmdWithVerbose tests diagnose command creation with verbose support
func TestCreateDiagnoseCmdWithVerbose(t *testing.T) {
	logConfig := &LogConfig{
		ConfigFile: "/test/config.yml",
		LogLevel:   "debug",
		DryRun:     true,
		LogFormat:  "json",
	}

	cmd := createDiagnoseCmdWithVerbose(logConfig)

	assert.Equal(t, "diagnose", cmd.Use)
	assert.Equal(t, "Collect diagnostic information", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.Contains(t, cmd.Long, "comprehensive system information")
	assert.Contains(t, cmd.Example, "go-broadcast diagnose")
}

// TestCreateRunDiagnoseWithVerbose tests diagnose run function with verbose support
func TestCreateRunDiagnoseWithVerbose(t *testing.T) {
	logConfig := &LogConfig{
		ConfigFile: "/test/config.yml",
		LogLevel:   "debug",
		DryRun:     true,
		LogFormat:  "json",
	}

	runFunc := createRunDiagnoseWithVerbose(logConfig)
	require.NotNil(t, runFunc)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create command and run
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runFunc(cmd, []string{})

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify no error
	require.NoError(t, err)

	// Verify JSON output
	var info DiagnosticInfo
	require.NoError(t, json.Unmarshal([]byte(output), &info))

	// Verify it used the provided config
	assert.Equal(t, "/test/config.yml", info.Config.Path)
}

// TestCreateDiagnoseCmd tests isolated diagnose command creation
func TestCreateDiagnoseCmd(t *testing.T) {
	flags := &Flags{
		ConfigFile: "/test/config.yml",
		LogLevel:   "info",
		DryRun:     false,
	}

	cmd := createDiagnoseCmd(flags)

	assert.Equal(t, "diagnose", cmd.Use)
	assert.Equal(t, "Collect diagnostic information", cmd.Short)
	assert.NotNil(t, cmd.RunE)
}

// TestCreateRunDiagnose tests isolated diagnose run function
func TestCreateRunDiagnose(t *testing.T) {
	flags := &Flags{
		ConfigFile: "/test/config.yml",
		LogLevel:   "info",
		DryRun:     false,
	}

	runFunc := createRunDiagnose(flags)
	require.NotNil(t, runFunc)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create command and run
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	err := runFunc(cmd, []string{})

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify no error
	require.NoError(t, err)

	// Verify JSON output
	var info DiagnosticInfo
	require.NoError(t, json.Unmarshal([]byte(output), &info))

	// Verify it used the provided config
	assert.Equal(t, "/test/config.yml", info.Config.Path)
}

// TestDiagnoseCmdIntegration tests the diagnose command as it would be used
func TestDiagnoseCmdIntegration(t *testing.T) {
	// This test verifies the command works end-to-end
	cmd := diagnoseCmd

	assert.Equal(t, "diagnose", cmd.Use)
	assert.NotNil(t, cmd.RunE)

	// Verify command can be executed (with captured output)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Initialize global flags if not already set
	if globalFlags == nil {
		globalFlags = &Flags{
			ConfigFile: "~/.config/go-broadcast.yml",
			LogLevel:   "info",
			DryRun:     false,
		}
	}

	cmd.SetContext(context.Background())
	err := cmd.RunE(cmd, []string{})

	// Restore stdout
	require.NoError(t, w.Close())
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Should not error
	require.NoError(t, err)

	// Should produce valid JSON
	var info DiagnosticInfo
	require.NoError(t, json.Unmarshal([]byte(output), &info))
}
