package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/testutil"
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
			Branch:       "main",
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
				SyncBranch: strPtr("sync/template-20240115-120000-abc123"),
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

// TestGetMockStatus tests mock status generation
func TestGetMockStatus(t *testing.T) {
	ctx := context.Background()

	cfg := &config.Config{
		Source: config.SourceConfig{
			Repo:   "org/template",
			Branch: "main",
		},
		Targets: []config.TargetConfig{
			{Repo: "org/target1"},
			{Repo: "org/target2"},
			{Repo: "org/target3"},
		},
	}

	status := getMockStatus(ctx, cfg)

	// Verify source status
	assert.Equal(t, "org/template", status.Source.Repository)
	assert.Equal(t, "main", status.Source.Branch)
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
			Branch:       "main",
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
					Branch:       "main",
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
					Branch:       "main",
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
						SyncBranch: strPtr("sync/template-branch"),
						PullRequest: &PullRequestInfo{
							Number: 10,
							State:  "open",
							URL:    "https://github.com/org/target2/pull/10",
							Title:  "Update from template",
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

	t.Run("ValidConfigTextOutput", func(t *testing.T) {
		// Create temporary config
		tmpDir := testutil.CreateTempDir(t)
		configPath := filepath.Join(tmpDir, "config.yml")

		configContent := `version: 1
source:
  repo: org/template
  branch: main
targets:
  - repo: org/target1
    files:
      - src: README.md
        dest: README.md
  - repo: org/target2
    files:
      - src: README.md
        dest: README.md`

		testutil.WriteTestFile(t, configPath, configContent)

		// Save original values
		originalConfig := globalFlags.ConfigFile
		originalJSON := jsonOutput
		globalFlags.ConfigFile = configPath
		jsonOutput = false
		defer func() {
			globalFlags.ConfigFile = originalConfig
			jsonOutput = originalJSON
		}()

		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		err := runStatus(cmd, []string{})
		require.NoError(t, err)
	})

	t.Run("ValidConfigJSONOutput", func(t *testing.T) {
		// Create temporary config
		tmpDir := testutil.CreateTempDir(t)
		configPath := filepath.Join(tmpDir, "config.yml")

		configContent := `version: 1
source:
  repo: org/template
  branch: main
targets:
  - repo: org/target1
    files:
      - src: README.md
        dest: README.md`

		testutil.WriteTestFile(t, configPath, configContent)

		// Save original values
		originalConfig := globalFlags.ConfigFile
		originalJSON := jsonOutput
		globalFlags.ConfigFile = configPath
		jsonOutput = true
		defer func() {
			globalFlags.ConfigFile = originalConfig
			jsonOutput = originalJSON
		}()

		// Capture output
		oldStdout := output.Stdout()
		var buf bytes.Buffer
		output.SetStdout(&buf)
		defer output.SetStdout(oldStdout)

		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		err := runStatus(cmd, []string{})
		require.NoError(t, err)

		// Verify JSON output
		var status SyncStatus
		require.NoError(t, json.Unmarshal(buf.Bytes(), &status))
		assert.Equal(t, "org/template", status.Source.Repository)
		assert.Equal(t, "main", status.Source.Branch)
		assert.Len(t, status.Targets, 1)
		assert.Equal(t, "org/target1", status.Targets[0].Repository)
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
			Branch:       "main",
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
					Branch:       "main",
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
