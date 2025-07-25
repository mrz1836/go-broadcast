// Package cli provides command-line interface functionality for go-broadcast.
package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGetConfigFile verifies that GetConfigFile returns the correct config file path
func TestGetConfigFile(t *testing.T) {
	tests := []struct {
		name           string
		setupFlags     *Flags
		expectedConfig string
	}{
		{
			name:           "default config file",
			setupFlags:     nil,
			expectedConfig: "sync.yaml",
		},
		{
			name: "custom config file",
			setupFlags: &Flags{
				ConfigFile: "custom-config.yaml",
				DryRun:     false,
				LogLevel:   "info",
			},
			expectedConfig: "custom-config.yaml",
		},
		{
			name: "absolute path config file",
			setupFlags: &Flags{
				ConfigFile: "/etc/myapp/config.yaml",
				DryRun:     false,
				LogLevel:   "info",
			},
			expectedConfig: "/etc/myapp/config.yaml",
		},
		{
			name: "empty config file",
			setupFlags: &Flags{
				ConfigFile: "",
				DryRun:     false,
				LogLevel:   "info",
			},
			expectedConfig: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current flags
			originalFlags := globalFlags

			// Reset flags after test
			defer func() {
				globalFlags = originalFlags
			}()

			// Setup test flags if provided
			if tt.setupFlags != nil {
				SetFlags(tt.setupFlags)
			} else {
				// Reset to nil for default test
				globalFlags = nil
			}

			// Test GetConfigFile
			result := GetConfigFile()
			require.Equal(t, tt.expectedConfig, result)
		})
	}
}

// TestIsDryRun verifies that IsDryRun returns the correct dry-run state
func TestIsDryRun(t *testing.T) {
	tests := []struct {
		name           string
		setupFlags     *Flags
		expectedDryRun bool
	}{
		{
			name:           "default dry-run (false)",
			setupFlags:     nil,
			expectedDryRun: false,
		},
		{
			name: "dry-run enabled",
			setupFlags: &Flags{
				ConfigFile: "sync.yaml",
				DryRun:     true,
				LogLevel:   "info",
			},
			expectedDryRun: true,
		},
		{
			name: "dry-run explicitly disabled",
			setupFlags: &Flags{
				ConfigFile: "sync.yaml",
				DryRun:     false,
				LogLevel:   "info",
			},
			expectedDryRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current flags
			originalFlags := globalFlags

			// Reset flags after test
			defer func() {
				globalFlags = originalFlags
			}()

			// Setup test flags if provided
			if tt.setupFlags != nil {
				SetFlags(tt.setupFlags)
			} else {
				// Reset to nil for default test
				globalFlags = nil
			}

			// Test IsDryRun
			result := IsDryRun()
			require.Equal(t, tt.expectedDryRun, result)
		})
	}
}

// TestSetFlags verifies that SetFlags correctly updates the global flags
func TestSetFlags(t *testing.T) {
	tests := []struct {
		name     string
		newFlags *Flags
	}{
		{
			name: "update all flags",
			newFlags: &Flags{
				ConfigFile: "production.yaml",
				DryRun:     true,
				LogLevel:   "debug",
			},
		},
		{
			name: "update with empty config",
			newFlags: &Flags{
				ConfigFile: "",
				DryRun:     false,
				LogLevel:   "error",
			},
		},
		{
			name: "update with verbose log level",
			newFlags: &Flags{
				ConfigFile: "test.yaml",
				DryRun:     true,
				LogLevel:   "trace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current flags
			originalFlags := globalFlags

			// Reset flags after test
			defer func() {
				globalFlags = originalFlags
			}()

			// Set new flags
			SetFlags(tt.newFlags)

			// Verify all fields were updated
			require.Equal(t, tt.newFlags.ConfigFile, globalFlags.ConfigFile)
			require.Equal(t, tt.newFlags.DryRun, globalFlags.DryRun)
			require.Equal(t, tt.newFlags.LogLevel, globalFlags.LogLevel)

			// Verify that the functions return the new values
			require.Equal(t, tt.newFlags.ConfigFile, GetConfigFile())
			require.Equal(t, tt.newFlags.DryRun, IsDryRun())
		})
	}
}

// TestResetGlobalFlags verifies that ResetGlobalFlags restores default values
func TestResetGlobalFlags(t *testing.T) {
	// Save current flags
	originalFlags := globalFlags

	// Reset flags after test
	defer func() {
		globalFlags = originalFlags
	}()

	// Set non-default values
	SetFlags(&Flags{
		ConfigFile: "custom.yaml",
		DryRun:     true,
		LogLevel:   "trace",
	})

	// Verify flags were changed
	require.Equal(t, "custom.yaml", GetConfigFile())
	require.True(t, IsDryRun())
	require.Equal(t, "trace", globalFlags.LogLevel)

	// Reset flags
	ResetGlobalFlags()

	// Verify all flags are back to defaults
	require.Equal(t, "sync.yaml", GetConfigFile())
	require.False(t, IsDryRun())
	require.Equal(t, "info", globalFlags.LogLevel)
}

// TestFlagsConcurrency verifies that flag operations are safe for concurrent use
func TestFlagsConcurrency(t *testing.T) {
	// Save current flags
	originalFlags := globalFlags

	// Reset flags after test
	defer func() {
		globalFlags = originalFlags
	}()

	// Test concurrent reads
	t.Run("concurrent reads", func(t *testing.T) {
		// Set known values
		SetFlags(&Flags{
			ConfigFile: "concurrent.yaml",
			DryRun:     true,
			LogLevel:   "debug",
		})

		// Run concurrent reads
		done := make(chan bool, 100)
		for i := 0; i < 100; i++ {
			go func() {
				config := GetConfigFile()
				dryRun := IsDryRun()

				// Verify values are consistent
				if config != "concurrent.yaml" {
					t.Errorf("expected config 'concurrent.yaml', got %s", config)
				}
				if !dryRun {
					t.Error("expected dryRun to be true")
				}

				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 100; i++ {
			<-done
		}
	})
}

// TestFlagsStructFields verifies the Flags struct fields
func TestFlagsStructFields(t *testing.T) {
	// Test creating a new Flags instance
	f := &Flags{
		ConfigFile: "test.yaml",
		DryRun:     true,
		LogLevel:   "warn",
	}

	require.Equal(t, "test.yaml", f.ConfigFile)
	require.True(t, f.DryRun)
	require.Equal(t, "warn", f.LogLevel)
}

// TestFlagsDefaultValues verifies the default values of global flags
func TestFlagsDefaultValues(t *testing.T) {
	// Save current flags
	originalFlags := globalFlags

	// Reset flags after test
	defer func() {
		globalFlags = originalFlags
	}()

	// Reset to ensure we're testing defaults
	ResetGlobalFlags()

	// Verify default values
	require.Equal(t, "sync.yaml", globalFlags.ConfigFile)
	require.False(t, globalFlags.DryRun)
	require.Equal(t, "info", globalFlags.LogLevel)
}

// TestFlagsPointerBehavior verifies pointer behavior when setting flags
func TestFlagsPointerBehavior(t *testing.T) {
	// Save current flags
	originalFlags := globalFlags

	// Reset flags after test
	defer func() {
		globalFlags = originalFlags
	}()

	// Create a new flags instance
	newFlags := &Flags{
		ConfigFile: "pointer.yaml",
		DryRun:     true,
		LogLevel:   "error",
	}

	// Set the flags
	SetFlags(newFlags)

	// Modify the original flags object
	newFlags.ConfigFile = "modified.yaml"
	newFlags.DryRun = false

	// Verify that globalFlags points to the same object
	require.Equal(t, "modified.yaml", GetConfigFile())
	require.False(t, IsDryRun())
}
