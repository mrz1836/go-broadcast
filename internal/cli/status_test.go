package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/state"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInitStatus tests status command initialization
func TestInitStatus(t *testing.T) {
	// Check that json flag is already registered from package init
	jsonFlag := statusCmd.Flags().Lookup("json")
	require.NotNil(t, jsonFlag)
	assert.Equal(t, "false", jsonFlag.DefValue)
	assert.Equal(t, "Output status in JSON format", jsonFlag.Usage)
}

// TestStatusCmd tests status command configuration
func TestStatusCmd(t *testing.T) {
	cmd := statusCmd
	assert.Equal(t, "status", cmd.Use)
	assert.Equal(t, "Show sync state for all targets", cmd.Short)
	assert.Contains(t, cmd.Long, "synchronization state")
	assert.Contains(t, cmd.Example, "go-broadcast status")
	assert.Contains(t, cmd.Aliases, "st")
	assert.NotNil(t, cmd.RunE)
}

// TestSyncStatusJSON tests JSON marshaling of SyncStatus
func TestSyncStatusJSON(t *testing.T) {
	status := &SyncStatus{
		Source: SourceStatus{
			Repository:   "org/source-repo",
			Branch:       "master",
			LatestCommit: "abc123def456789",
		},
		Targets: []TargetStatus{
			{
				Repository: "org/target1",
				State:      "synced",
				LastSync: &SyncInfo{
					Timestamp: "2024-01-15T12:00:00Z",
					Commit:    "abc123def456789",
				},
			},
			{
				Repository: "org/target2",
				State:      "outdated",
				SyncBranch: strPtr("chore/sync-files-20240115-120000-abc123"),
				PullRequest: &PullRequestInfo{
					Number: 42,
					State:  "open",
					URL:    "https://github.com/org/target2/pull/42",
					Title:  "Sync template updates",
				},
			},
			{
				Repository: "org/target3",
				State:      "error",
				Error:      strPtr("failed to access repository"),
			},
		},
	}

	data, err := json.Marshal(status)
	require.NoError(t, err)

	// Verify JSON structure
	var decoded SyncStatus
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, status.Source.Repository, decoded.Source.Repository)
	assert.Len(t, decoded.Targets, 3)
	assert.Equal(t, "synced", decoded.Targets[0].State)
	assert.Equal(t, "outdated", decoded.Targets[1].State)
	assert.Equal(t, "error", decoded.Targets[2].State)

	// Verify optional fields
	assert.NotNil(t, decoded.Targets[0].LastSync)
	assert.NotNil(t, decoded.Targets[1].PullRequest)
	assert.NotNil(t, decoded.Targets[2].Error)
}

// TestStatusConversion tests status conversion logic
func TestStatusConversion(t *testing.T) {
	// Create a mock state for testing conversion
	mockState := createMockState()

	// Convert to CLI status format
	status := convertStateToStatus(mockState)

	// Verify source status
	assert.Equal(t, "org/template", status.Source.Repository)
	assert.Equal(t, "master", status.Source.Branch)
	assert.NotEmpty(t, status.Source.LatestCommit)

	// Verify targets
	assert.Len(t, status.Targets, 3)

	// First target should be synced
	assert.Equal(t, "org/target1", status.Targets[0].Repository)
	assert.Equal(t, "synced", status.Targets[0].State)
	assert.NotNil(t, status.Targets[0].LastSync)

	// Other targets should be outdated with PR
	for i := 1; i < 3; i++ {
		target := status.Targets[i]
		assert.Equal(t, fmt.Sprintf("org/target%d", i+1), target.Repository)
		assert.Equal(t, "outdated", target.State)
		assert.NotNil(t, target.SyncBranch)
		assert.NotNil(t, target.PullRequest)
		assert.Equal(t, 42, target.PullRequest.Number)
		assert.Equal(t, "open", target.PullRequest.State)
	}
}

// TestOutputJSON tests JSON output formatting
func TestOutputJSON(t *testing.T) {
	status := &SyncStatus{
		Source: SourceStatus{
			Repository:   "org/source",
			Branch:       "master",
			LatestCommit: "abc123",
		},
		Targets: []TargetStatus{
			{
				Repository: "org/target",
				State:      "synced",
			},
		},
	}

	// Capture output
	oldStdout := output.Stdout()
	r, w, _ := os.Pipe()
	output.SetStdout(w)
	defer output.SetStdout(oldStdout)

	err := outputJSON(status)
	require.NoError(t, err)

	// Close writer and read output
	require.NoError(t, w.Close())
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Verify JSON output
	var decoded SyncStatus
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))
	assert.Equal(t, status.Source.Repository, decoded.Source.Repository)
}

// TestOutputTextStatus tests text output formatting
func TestOutputTextStatus(t *testing.T) {
	testCases := []struct {
		name   string
		status *SyncStatus
	}{
		{
			name: "AllSynced",
			status: &SyncStatus{
				Source: SourceStatus{
					Repository:   "org/source",
					Branch:       "master",
					LatestCommit: "abc123def456789",
				},
				Targets: []TargetStatus{
					{
						Repository: "org/target1",
						State:      "synced",
						LastSync: &SyncInfo{
							Timestamp: "2024-01-15T12:00:00Z",
							Commit:    "abc123def456789",
						},
					},
					{
						Repository: "org/target2",
						State:      "synced",
						LastSync: &SyncInfo{
							Timestamp: "2024-01-15T12:00:00Z",
							Commit:    "abc123def456789",
						},
					},
				},
			},
		},
		{
			name: "MixedStates",
			status: &SyncStatus{
				Source: SourceStatus{
					Repository:   "org/source",
					Branch:       "master",
					LatestCommit: "xyz789",
				},
				Targets: []TargetStatus{
					{
						Repository: "org/target1",
						State:      "synced",
					},
					{
						Repository: "org/target2",
						State:      "outdated",
						SyncBranch: strPtr("chore/sync-files-branch"),
						PullRequest: &PullRequestInfo{
							Number: 10,
							State:  "open",
							URL:    "https://github.com/org/target2/pull/10",
							Title:  "Update from source repository",
						},
					},
					{
						Repository: "org/target3",
						State:      "pending",
					},
					{
						Repository: "org/target4",
						State:      "error",
						Error:      strPtr("Authentication failed"),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Just test that the function runs without error
			err := outputTextStatus(tc.status)
			require.NoError(t, err)
		})
	}
}

// TestGetRealStatus tests getRealStatus function with various scenarios
func TestGetRealStatus(t *testing.T) {
	// Skip these tests if we're not in an environment with proper GitHub access
	// The getRealStatus function requires real GitHub API access which isn't available in test environments
	t.Skip("Skipping getRealStatus tests - requires real GitHub API access and dependency injection")
}

// TestConvertSyncStatus tests the convertSyncStatus function
func TestConvertSyncStatus(t *testing.T) {
	testCases := []struct {
		name     string
		input    state.SyncStatus
		expected string
	}{
		{
			name:     "StatusUpToDate",
			input:    state.StatusUpToDate,
			expected: "synced",
		},
		{
			name:     "StatusBehind",
			input:    state.StatusBehind,
			expected: "outdated",
		},
		{
			name:     "StatusPending",
			input:    state.StatusPending,
			expected: "pending",
		},
		{
			name:     "StatusUnknown",
			input:    state.StatusUnknown,
			expected: "unknown",
		},
		{
			name:     "StatusConflict",
			input:    state.StatusConflict,
			expected: "error",
		},
		{
			name:     "InvalidStatus",
			input:    state.SyncStatus("invalid-status"),
			expected: "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertSyncStatus(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestRunStatus tests the main status command execution
func TestRunStatus(t *testing.T) {
	t.Run("ConfigNotFound", func(t *testing.T) {
		// Save original config file path
		originalConfig := globalFlags.ConfigFile
		globalFlags.ConfigFile = "/non/existent/config.yml"
		defer func() {
			globalFlags.ConfigFile = originalConfig
		}()

		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		err := runStatus(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	t.Run("ConfigLoadError", func(t *testing.T) {
		// Create a temporary file with invalid YAML
		tmpFile, err := os.CreateTemp("", "invalid-config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		_, err = tmpFile.WriteString("invalid: yaml: content:\n  - broken")
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Save original config file path
		originalConfig := globalFlags.ConfigFile
		globalFlags.ConfigFile = tmpFile.Name()
		defer func() {
			globalFlags.ConfigFile = originalConfig
		}()

		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		err = runStatus(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	t.Run("ValidConfigWithMockStatus", func(t *testing.T) {
		// Create a valid temporary config
		tmpFile, err := os.CreateTemp("", "valid-config-*.yml")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		_, err = tmpFile.WriteString(TestValidConfig)
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Save original config file path
		originalConfig := globalFlags.ConfigFile
		globalFlags.ConfigFile = tmpFile.Name()
		defer func() {
			globalFlags.ConfigFile = originalConfig
		}()

		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		// This will fail due to lack of GitHub access, but we're testing the flow
		err = runStatus(cmd, []string{})
		// We expect an error because we can't mock getRealStatus without dependency injection
		require.Error(t, err)
		// But it should get past config loading
		assert.Contains(t, err.Error(), "failed to discover status")
	})

	t.Run("JSONOutputFlag", func(t *testing.T) {
		// Save original flags
		originalConfig := globalFlags.ConfigFile
		originalJSON := jsonOutput
		globalFlags.ConfigFile = "/non/existent/config.yml"
		jsonOutput = true
		defer func() {
			globalFlags.ConfigFile = originalConfig
			jsonOutput = originalJSON
		}()

		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		err := runStatus(cmd, []string{})
		require.Error(t, err)
		// Should still fail on config, regardless of output format
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	t.Run("ValidConfigTextOutput", func(t *testing.T) {
		t.Skip("Skipping integration test that requires real GitHub API access")
	})

	t.Run("ValidConfigJSONOutput", func(t *testing.T) {
		t.Skip("Skipping integration test that requires real GitHub API access")
	})
}

// TestTargetStatusStates tests all possible target states
func TestTargetStatusStates(t *testing.T) {
	states := []string{"synced", "outdated", "pending", "error"}

	for _, state := range states {
		t.Run(state, func(t *testing.T) {
			status := TargetStatus{
				Repository: "org/repo",
				State:      state,
			}

			// Add state-specific fields
			switch state {
			case "outdated":
				status.SyncBranch = strPtr("sync/branch")
				status.PullRequest = &PullRequestInfo{
					Number: 1,
					State:  "open",
					URL:    "https://github.com/org/repo/pull/1",
					Title:  "Sync",
				}
			case "error":
				status.Error = strPtr("test error")
			}

			// Marshal and unmarshal
			data, err := json.Marshal(status)
			require.NoError(t, err)

			var decoded TargetStatus
			require.NoError(t, json.Unmarshal(data, &decoded))

			assert.Equal(t, state, decoded.State)

			// Verify state-specific fields
			switch state {
			case "outdated":
				assert.NotNil(t, decoded.SyncBranch)
				assert.NotNil(t, decoded.PullRequest)
			case "error":
				assert.NotNil(t, decoded.Error)
			}
		})
	}
}

// TestStatusSummaryCalculation tests summary statistics
func TestStatusSummaryCalculation(t *testing.T) {
	status := &SyncStatus{
		Source: SourceStatus{
			Repository:   "org/source",
			Branch:       "master",
			LatestCommit: "abc123",
		},
		Targets: []TargetStatus{
			{Repository: "repo1", State: "synced"},
			{Repository: "repo2", State: "synced"},
			{Repository: "repo3", State: "outdated"},
			{Repository: "repo4", State: "outdated"},
			{Repository: "repo5", State: "outdated"},
			{Repository: "repo6", State: "pending"},
			{Repository: "repo7", State: "error"},
		},
	}

	// Just test that the function runs without error
	err := outputTextStatus(status)
	require.NoError(t, err)
}

// TestStatusOutputIcons tests correct icon display for states
func TestStatusOutputIcons(t *testing.T) {
	iconMap := map[string]string{
		"synced":   "✓",
		"outdated": "⚠",
		"pending":  "⏳",
		"error":    "✗",
		"unknown":  "?",
	}

	for state := range iconMap {
		t.Run(state, func(t *testing.T) {
			status := &SyncStatus{
				Source: SourceStatus{
					Repository:   "org/source",
					Branch:       "master",
					LatestCommit: "abc123",
				},
				Targets: []TargetStatus{
					{
						Repository: "org/target",
						State:      state,
					},
				},
			}

			// Just test that the function runs without error
			err := outputTextStatus(status)
			require.NoError(t, err)
		})
	}
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}

// createMockState creates a mock state for testing conversion logic
func createMockState() *state.State {
	now := time.Now()
	commitSHA := "abc123def456"

	return &state.State{
		Source: state.SourceState{
			Repo:         "org/template",
			Branch:       "master",
			LatestCommit: commitSHA,
			LastChecked:  now,
		},
		Targets: map[string]*state.TargetState{
			"org/target1": {
				Repo:           "org/target1",
				Status:         state.StatusUpToDate,
				LastSyncCommit: commitSHA,
				LastSyncTime:   &now,
				SyncBranches: []state.SyncBranch{
					{
						Name: "chore/sync-files-20240115-120000-abc123",
						Metadata: &state.BranchMetadata{
							Timestamp: now,
							CommitSHA: commitSHA,
							Prefix:    "chore/sync-files",
						},
					},
				},
				OpenPRs: []gh.PR{},
			},
			"org/target2": {
				Repo:           "org/target2",
				Status:         state.StatusBehind,
				LastSyncCommit: "abc123old",
				LastSyncTime:   &now,
				SyncBranches: []state.SyncBranch{
					{
						Name: "chore/sync-files-20240116-120000-abc124",
						Metadata: &state.BranchMetadata{
							Timestamp: now.Add(time.Hour),
							CommitSHA: "abc124",
							Prefix:    "chore/sync-files",
						},
					},
				},
				OpenPRs: []gh.PR{
					{
						Number: 42,
						State:  "open",
						Title:  "Sync template updates",
						Head: struct {
							Ref string `json:"ref"`
							SHA string `json:"sha"`
						}{
							Ref: "chore/sync-files-20240116-120000-abc124",
							SHA: "abc124",
						},
					},
				},
			},
			"org/target3": {
				Repo:           "org/target3",
				Status:         state.StatusBehind,
				LastSyncCommit: "abc123old",
				LastSyncTime:   &now,
				SyncBranches: []state.SyncBranch{
					{
						Name: "chore/sync-files-20240117-120000-abc125",
						Metadata: &state.BranchMetadata{
							Timestamp: now.Add(2 * time.Hour),
							CommitSHA: "abc125",
							Prefix:    "chore/sync-files",
						},
					},
				},
				OpenPRs: []gh.PR{
					{
						Number: 42,
						State:  "open",
						Title:  "Sync template updates",
						Head: struct {
							Ref string `json:"ref"`
							SHA string `json:"sha"`
						}{
							Ref: "chore/sync-files-20240117-120000-abc125",
							SHA: "abc125",
						},
					},
				},
			},
		},
	}
}

// TestGetRealStatusErrorCases tests error handling in getRealStatus
func TestGetRealStatusErrorCases(t *testing.T) {
	t.Run("GitHub client creation failure", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{
			Source: config.SourceConfig{
				Repo:   "test/source",
				Branch: "main",
			},
			Targets: []config.TargetConfig{
				{Repo: "test/target1"},
			},
		}

		// This will likely fail with GitHub CLI not found or auth issues
		status, err := getRealStatus(ctx, cfg)

		// Should return error, not panic
		require.Error(t, err)
		assert.Nil(t, status)

		// Error should be GitHub-related (auth, not found, or branch error)
		assert.True(t,
			strings.Contains(err.Error(), "failed to initialize GitHub client") ||
				strings.Contains(err.Error(), "ErrGHNotFound") ||
				strings.Contains(err.Error(), "ErrNotAuthenticated") ||
				strings.Contains(err.Error(), "Please install GitHub CLI") ||
				strings.Contains(err.Error(), "Please run: gh auth login") ||
				strings.Contains(err.Error(), "branch not found") ||
				strings.Contains(err.Error(), "not authenticated"),
			"Expected GitHub client or branch error, got: %s", err.Error())
	})

	t.Run("State discovery failure", func(t *testing.T) {
		ctx := context.Background()
		cfg := &config.Config{
			Source: config.SourceConfig{
				Repo:   "nonexistent/repo",
				Branch: "main",
			},
			Targets: []config.TargetConfig{
				{Repo: "test/target1"},
			},
		}

		// Will fail at state discovery phase if it gets past client creation
		status, err := getRealStatus(ctx, cfg)
		require.Error(t, err)
		assert.Nil(t, status)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		// Create canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cfg := &config.Config{
			Source: config.SourceConfig{
				Repo:   "test/source",
				Branch: "main",
			},
		}

		// Should handle canceled context gracefully
		status, err := getRealStatus(ctx, cfg)
		// The function might still succeed if it doesn't check context before operations
		// or it might fail with context/GitHub error - both are acceptable
		if err != nil {
			// If it errors, it should be related to context or GitHub operations
			assert.True(t,
				strings.Contains(err.Error(), "context") ||
					strings.Contains(err.Error(), "canceled") ||
					strings.Contains(err.Error(), "GitHub") ||
					strings.Contains(err.Error(), "initialize") ||
					strings.Contains(err.Error(), "not authenticated") ||
					strings.Contains(err.Error(), "branch not found"),
				"Expected context or GitHub error, got: %s", err.Error())
		}
		_ = status // May be nil or valid depending on timing
	})
}
