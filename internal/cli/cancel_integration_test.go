package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/gh"
	"github.com/mrz1836/go-broadcast/internal/output"
	"github.com/mrz1836/go-broadcast/internal/state"
)

// TestRunCancel tests the runCancel function
func TestRunCancel(t *testing.T) {
	// Save original flags (thread-safe)
	originalFlags := GetGlobalFlags()
	defer func() {
		SetFlags(originalFlags)
	}()

	testCases := []struct {
		name          string
		setupConfig   func() (string, func())
		args          []string
		dryRun        bool
		expectError   bool
		errorContains string
		verifyOutput  func(*testing.T, string)
	}{
		{
			name: "Config file not found",
			setupConfig: func() (string, func()) {
				SetFlags(&Flags{
					ConfigFile: "/non/existent/config.yml",
					LogLevel:   "info",
				})
				return "", func() {}
			},
			expectError:   true,
			errorContains: "failed to load configuration",
		},
		{
			name: "Invalid config file",
			setupConfig: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "invalid-*.yml")
				require.NoError(t, err)

				invalidConfig := `invalid: yaml: content:`
				_, err = tmpFile.WriteString(invalidConfig)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				SetFlags(&Flags{
					ConfigFile: tmpFile.Name(),
					LogLevel:   "info",
				})
				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			expectError:   true,
			errorContains: "failed to load configuration",
		},
		{
			name: "Valid config - dry run mode",
			setupConfig: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "valid-*.yml")
				require.NoError(t, err)

				validConfig := TestValidConfig

				_, err = tmpFile.WriteString(validConfig)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				SetFlags(&Flags{
					ConfigFile: tmpFile.Name(),
					DryRun:     true,
					LogLevel:   "info",
				})
				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			dryRun:        true,
			expectError:   true, // Will fail because gh.NewClient requires actual GitHub CLI
			errorContains: "cancel operation failed",
		},
		{
			name: "With specific target repos",
			setupConfig: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "valid-*.yml")
				require.NoError(t, err)

				validConfig := `version: 1
groups:
  - name: "test-group"
    id: "test-group-1"
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
          - src: LICENSE
            dest: LICENSE`

				_, err = tmpFile.WriteString(validConfig)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				SetFlags(&Flags{
					ConfigFile: tmpFile.Name(),
					LogLevel:   "info",
				})
				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			args:          []string{"org/target1"},
			expectError:   true, // Will fail because gh.NewClient requires actual GitHub CLI
			errorContains: "cancel operation failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configPath, cleanup := tc.setupConfig()
			defer cleanup()

			// Create command
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Capture output (would need proper output capture mechanism)
			var outputBuf bytes.Buffer

			// Execute command
			err := runCancel(cmd, tc.args)

			// Verify results
			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify output if checker provided
			if tc.verifyOutput != nil {
				tc.verifyOutput(t, outputBuf.String())
			}

			_ = configPath
		})
	}
}

// TestPerformCancel tests the performCancel function with mocked dependencies
func TestPerformCancel(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Groups: []config.Group{{
			Source: config.SourceConfig{
				Repo:   "org/source",
				Branch: "master",
			},
			Targets: []config.TargetConfig{
				{Repo: "org/target1"},
				{Repo: "org/target2"},
			},
		}},
	}

	testCases := []struct {
		name          string
		targetRepos   []string
		clientError   error
		stateError    error
		setupState    func() *state.State
		expectError   bool
		errorContains string
		verifySummary func(*testing.T, *CancelSummary)
	}{
		{
			name:        "Nil config panics",
			expectError: true,
			// This test would cause a panic, so we skip it
		},
		{
			name:          "GitHub CLI not found",
			clientError:   gh.ErrGHNotFound,
			expectError:   true,
			errorContains: "gh CLI not found in PATH",
		},
		{
			name:          "Not authenticated",
			clientError:   gh.ErrNotAuthenticated,
			expectError:   true,
			errorContains: "gh CLI not authenticated",
		},
		{
			name:          "Client initialization error",
			clientError:   errors.New("network error"), //nolint:err113 // test-only error
			expectError:   true,
			errorContains: "network error",
		},
		{
			name:          "State discovery error",
			stateError:    errors.New("API error"), //nolint:err113 // test-only error
			expectError:   true,
			errorContains: "failed to discover sync state",
		},
		{
			name:        "Filter nonexistent target",
			targetRepos: []string{"org/nonexistent"},
			setupState: func() *state.State {
				return &state.State{
					Targets: map[string]*state.TargetState{
						"org/target1": {Repo: "org/target1"},
						"org/target2": {Repo: "org/target2"},
					},
				}
			},
			expectError:   true,
			errorContains: "failed to filter targets",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "Nil config panics" {
				// Test panic recovery for performCancelWithDiscoverer
				defer func() {
					if r := recover(); r != nil {
						assert.Equal(t, "config cannot be nil", r)
					}
				}()
				mockClient := &gh.MockClient{}
				mockDiscoverer := &state.MockDiscoverer{}
				_, _ = performCancelWithDiscoverer(context.Background(), nil, nil, mockClient, mockDiscoverer)
				return
			}

			// Create mocks
			mockClient := &gh.MockClient{}
			mockDiscoverer := &state.MockDiscoverer{}

			// Set up mock expectations based on test case
			if tc.clientError != nil {
				// Simulate client errors by having the discoverer return the appropriate error
				mockDiscoverer.On("DiscoverState", context.Background(), cfg).Return(nil, tc.clientError)

				summary, err := performCancelWithDiscoverer(context.Background(), cfg, tc.targetRepos, mockClient, mockDiscoverer)

				require.Error(t, err)
				assert.Nil(t, summary)
				assert.Contains(t, err.Error(), tc.errorContains)
				mockDiscoverer.AssertExpectations(t)
				return
			}

			if tc.stateError != nil {
				// Mock discoverer returns state error
				mockDiscoverer.On("DiscoverState", context.Background(), cfg).Return(nil, tc.stateError)

				summary, err := performCancelWithDiscoverer(context.Background(), cfg, tc.targetRepos, mockClient, mockDiscoverer)

				require.Error(t, err)
				assert.Nil(t, summary)
				assert.Contains(t, err.Error(), tc.errorContains)
				mockDiscoverer.AssertExpectations(t)
				return
			}

			if tc.setupState != nil {
				// Mock discoverer returns the test state, then test filtering
				testState := tc.setupState()
				mockDiscoverer.On("DiscoverState", context.Background(), cfg).Return(testState, nil)

				summary, err := performCancelWithDiscoverer(context.Background(), cfg, tc.targetRepos, mockClient, mockDiscoverer)

				require.Error(t, err)
				assert.Nil(t, summary)
				assert.Contains(t, err.Error(), tc.errorContains)
				mockDiscoverer.AssertExpectations(t)
				return
			}

			// For any remaining cases, mock a successful state discovery with no active syncs
			emptyState := &state.State{
				Targets: map[string]*state.TargetState{
					"org/target1": {
						Repo:         "org/target1",
						OpenPRs:      []gh.PR{},
						SyncBranches: []state.SyncBranch{},
					},
					"org/target2": {
						Repo:         "org/target2",
						OpenPRs:      []gh.PR{},
						SyncBranches: []state.SyncBranch{},
					},
				},
			}
			mockDiscoverer.On("DiscoverState", context.Background(), cfg).Return(emptyState, nil)

			summary, err := performCancelWithDiscoverer(context.Background(), cfg, tc.targetRepos, mockClient, mockDiscoverer)

			if tc.expectError {
				require.Error(t, err)
				assert.Nil(t, summary)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, summary)
				assert.Equal(t, 0, summary.TotalTargets) // No active syncs to cancel
			}
			mockDiscoverer.AssertExpectations(t)
		})
	}

	// Use cfg to avoid unused variable error
	_ = cfg
}

// TestOutputCancelResultsIntegration tests the output functions
func TestOutputCancelResultsIntegration(t *testing.T) {
	testCases := []struct {
		name         string
		summary      *CancelSummary
		outputFormat string
		verifyOutput func(*testing.T, string)
	}{
		{
			name: "JSON output format",
			summary: &CancelSummary{
				TotalTargets:    2,
				PRsClosed:       1,
				BranchesDeleted: 1,
				Errors:          0,
				Results: []CancelResult{
					{
						Repository:    "org/target1",
						PRNumber:      intPtr(123),
						PRClosed:      true,
						BranchName:    "sync/test",
						BranchDeleted: true,
					},
					{
						Repository: "org/target2",
						Error:      "API error",
					},
				},
			},
			outputFormat: "json",
			verifyOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, `"total_targets": 2`)
				assert.Contains(t, output, `"prs_closed": 1`)
				assert.Contains(t, output, `"branches_deleted": 1`)
			},
		},
		{
			name: "Text output format",
			summary: &CancelSummary{
				TotalTargets:    3,
				PRsClosed:       2,
				BranchesDeleted: 2,
				Errors:          1,
				Results: []CancelResult{
					{
						Repository:    "org/target1",
						PRNumber:      intPtr(123),
						PRClosed:      true,
						BranchName:    "sync/test1",
						BranchDeleted: true,
					},
					{
						Repository:    "org/target2",
						PRNumber:      intPtr(456),
						PRClosed:      true,
						BranchName:    "sync/test2",
						BranchDeleted: true,
					},
					{
						Repository: "org/target3",
						Error:      "Permission denied",
					},
				},
			},
			outputFormat: "text",
			verifyOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "Canceled sync operations for 3 target(s)")
				assert.Contains(t, output, "📦 org/target1")
				assert.Contains(t, output, "PR #123")
				assert.Contains(t, output, "sync/test1")
				assert.Contains(t, output, "📦 org/target3")
				assert.Contains(t, output, "Summary:")
				assert.Contains(t, output, "PRs closed: 2")
				assert.Contains(t, output, "Branches deleted: 2")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up output capture
			var stdoutBuf bytes.Buffer
			originalStdout := output.Stdout()
			output.SetStdout(&stdoutBuf)
			defer output.SetStdout(originalStdout)

			// Set up JSON output flag (thread-safe)
			setJSONOutput(tc.outputFormat == "json")
			defer func() {
				setJSONOutput(false)
			}()

			// Execute output function
			err := outputCancelResults(tc.summary)
			require.NoError(t, err)

			// Verify output
			if tc.verifyOutput != nil {
				outputStr := stdoutBuf.String()
				tc.verifyOutput(t, outputStr)
			}
		})
	}
}

// TestOutputCancelPreviewIntegration tests the preview output for dry run
func TestOutputCancelPreviewIntegration(t *testing.T) {
	summary := &CancelSummary{
		TotalTargets: 2,
		Results: []CancelResult{
			{
				Repository:    "org/target1",
				PRNumber:      intPtr(123),
				PRClosed:      true,
				BranchName:    "sync/test1",
				BranchDeleted: true,
			},
			{
				Repository:    "org/target2",
				PRNumber:      intPtr(456),
				PRClosed:      true,
				BranchName:    "sync/test2",
				BranchDeleted: false, // Keep branches flag
			},
		},
		DryRun: true,
	}

	// Set up output capture
	var stdoutBuf bytes.Buffer
	originalStdout := output.Stdout()
	output.SetStdout(&stdoutBuf)
	defer output.SetStdout(originalStdout)

	// Execute preview
	err := outputCancelPreview(summary)
	require.NoError(t, err)

	// Verify output
	outputStr := stdoutBuf.String()
	assert.Contains(t, outputStr, "Would cancel sync operations for 2 target(s)")
	assert.Contains(t, outputStr, "📦 org/target1")
	assert.Contains(t, outputStr, "Would close PR #123")
	assert.Contains(t, outputStr, "Would delete branch: sync/test1")
	assert.Contains(t, outputStr, "📦 org/target2")
	assert.Contains(t, outputStr, "Would close PR #456")
	assert.Contains(t, outputStr, "Would delete branch: sync/test2")
	assert.Contains(t, outputStr, "Summary (would):")
}

// Multi-group integration tests

// TestRunCancel_MultiGroupConfig tests the full cancel command with multi-group configurations
func TestRunCancel_MultiGroupConfig(t *testing.T) {
	// Save original flags (thread-safe)
	originalFlags := GetGlobalFlags()
	defer func() {
		SetFlags(originalFlags)
	}()

	testCases := []struct {
		name          string
		setupConfig   func() (string, func())
		args          []string
		dryRun        bool
		expectError   bool
		errorContains string
		verifyOutput  func(*testing.T, string)
	}{
		{
			name: "Multi-group config - dry run mode",
			setupConfig: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "multi-group-*.yml")
				require.NoError(t, err)

				multiGroupConfig := `version: 1
groups:
  - name: "group-1"
    id: "group1"
    source:
      repo: org/source1
      branch: main
    targets:
      - repo: org/target1
        files:
          - src: README.md
            dest: README.md
  - name: "group-2"
    id: "group2"
    source:
      repo: org/source2
      branch: main
    targets:
      - repo: org/target2
        files:
          - src: LICENSE
            dest: LICENSE
  - name: "group-3"
    id: "group3"
    source:
      repo: org/source3
      branch: main
    targets:
      - repo: org/target3
        files:
          - src: CONTRIBUTING.md
            dest: CONTRIBUTING.md
  - name: "skyetel-go"
    id: "skyetel-go"
    source:
      repo: skyetel/template
      branch: development
    targets:
      - repo: skyetel/reach
        files:
          - src: .github/workflows/ci.yml
            dest: .github/workflows/ci.yml`

				_, err = tmpFile.WriteString(multiGroupConfig)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				SetFlags(&Flags{
					ConfigFile: tmpFile.Name(),
					DryRun:     true,
					LogLevel:   "info",
				})
				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			dryRun:        true,
			expectError:   true, // Will fail because gh.NewClient requires actual GitHub CLI
			errorContains: "cancel operation failed",
		},
		{
			name: "Target specific repo from 4th group (skyetel-go regression test)",
			setupConfig: func() (string, func()) {
				tmpFile, err := os.CreateTemp("", "skyetel-regression-*.yml")
				require.NoError(t, err)

				skyetelConfig := `version: 1
groups:
  - name: "mrz-tools"
    id: "mrz-tools"
    source:
      repo: mrz1836/go-broadcast
      branch: master
    targets:
      - repo: mrz1836/tool1
        files:
          - src: .gitignore
            dest: .gitignore
  - name: "mrz-libraries"
    id: "mrz-libraries"
    source:
      repo: mrz1836/go-broadcast
      branch: master
    targets:
      - repo: mrz1836/lib1
        files:
          - src: LICENSE
            dest: LICENSE
  - name: "mrz-fun-projects"
    id: "mrz-fun-projects"
    source:
      repo: mrz1836/go-broadcast
      branch: master
    targets:
      - repo: mrz1836/fun1
        files:
          - src: README.md
            dest: README.md
  - name: "skyetel-go"
    id: "skyetel-go"
    source:
      repo: skyetel/go-template
      branch: development
    targets:
      - repo: skyetel/reach
        files:
          - src: .github/workflows/ci.yml
            dest: .github/workflows/ci.yml`

				_, err = tmpFile.WriteString(skyetelConfig)
				require.NoError(t, err)
				require.NoError(t, tmpFile.Close())

				SetFlags(&Flags{
					ConfigFile: tmpFile.Name(),
					LogLevel:   "info",
				})
				return tmpFile.Name(), func() { _ = os.Remove(tmpFile.Name()) }
			},
			args:          []string{"skyetel/reach"}, // Target the 4th group specifically
			expectError:   true,                      // Will fail because gh.NewClient requires actual GitHub CLI
			errorContains: "cancel operation failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configPath, cleanup := tc.setupConfig()
			defer cleanup()

			// Create command
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			// Execute command
			err := runCancel(cmd, tc.args)

			// Verify results
			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
			} else {
				require.NoError(t, err)
			}

			_ = configPath
		})
	}
}

// TestPerformCancelWithDiscoverer_MultiGroupIntegration tests multi-group scenarios with mocked discoverer
func TestPerformCancelWithDiscoverer_MultiGroupIntegration(t *testing.T) {
	// Create realistic multi-group config similar to the actual sync.yaml
	cfg := &config.Config{
		Groups: []config.Group{
			{
				Name: "mrz-tools",
				ID:   "mrz-tools",
				Source: config.SourceConfig{
					Repo:   "mrz1836/go-broadcast",
					Branch: "master",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/tool1"},
					{Repo: "mrz1836/tool2"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "chore/sync-files",
				},
			},
			{
				Name: "mrz-libraries",
				ID:   "mrz-libraries",
				Source: config.SourceConfig{
					Repo:   "mrz1836/go-broadcast",
					Branch: "master",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/lib1"},
					{Repo: "mrz1836/lib2"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "chore/sync-files",
				},
			},
			{
				Name: "mrz-fun-projects",
				ID:   "mrz-fun-projects",
				Source: config.SourceConfig{
					Repo:   "mrz1836/go-broadcast",
					Branch: "master",
				},
				Targets: []config.TargetConfig{
					{Repo: "mrz1836/fun1"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "chore/sync-files",
				},
			},
			{
				Name: "skyetel-go",
				ID:   "skyetel-go",
				Source: config.SourceConfig{
					Repo:   "skyetel/go-template",
					Branch: "development",
				},
				Targets: []config.TargetConfig{
					{Repo: "skyetel/reach"},
				},
				Defaults: config.DefaultConfig{
					BranchPrefix: "chore/sync-files",
				},
			},
		},
	}

	tests := []struct {
		name          string
		targetRepos   []string
		setupState    func() *state.State
		setupMocks    func(*gh.MockClient)
		verifySummary func(*testing.T, *CancelSummary)
	}{
		{
			name:        "skyetel-go 4th group regression - full integration",
			targetRepos: []string{"skyetel/reach"}, // This was the repo that wasn't being found
			setupState: func() *state.State {
				return &state.State{
					Targets: map[string]*state.TargetState{
						"mrz1836/tool1": {Repo: "mrz1836/tool1"},
						"mrz1836/tool2": {Repo: "mrz1836/tool2"},
						"mrz1836/lib1":  {Repo: "mrz1836/lib1"},
						"mrz1836/lib2":  {Repo: "mrz1836/lib2"},
						"mrz1836/fun1":  {Repo: "mrz1836/fun1"},
						"skyetel/reach": { // The problematic 4th group target
							Repo: "skyetel/reach",
							OpenPRs: []gh.PR{
								{Number: 430, State: "open", Title: "Sync from skyetel/go-template"},
							},
							SyncBranches: []state.SyncBranch{
								{
									Name: "chore/sync-files-skyetel-go-20250112-145757-561a06e",
									Metadata: &state.BranchMetadata{
										Timestamp: time.Date(2025, time.January, 12, 14, 57, 57, 0, time.UTC),
										CommitSHA: "561a06e",
										GroupID:   "skyetel-go",
									},
								},
							},
						},
					},
					Source: state.SourceState{
						Repo:         "mrz1836/go-broadcast",
						Branch:       "master",
						LatestCommit: "561a06e",
					},
				}
			},
			setupMocks: func(mockClient *gh.MockClient) {
				mockClient.On("ClosePR", mock.Anything, "skyetel/reach", 430, mock.AnythingOfType("string")).Return(nil)
				mockClient.On("DeleteBranch", mock.Anything, "skyetel/reach", "chore/sync-files-skyetel-go-20250112-145757-561a06e").Return(nil)
			},
			verifySummary: func(t *testing.T, summary *CancelSummary) {
				assert.Equal(t, 1, summary.TotalTargets)
				assert.Equal(t, 1, summary.PRsClosed)
				assert.Equal(t, 1, summary.BranchesDeleted)
				assert.Equal(t, 0, summary.Errors)

				// Verify the specific result
				require.Len(t, summary.Results, 1)
				result := summary.Results[0]
				assert.Equal(t, "skyetel/reach", result.Repository)
				assert.Equal(t, 430, *result.PRNumber)
				assert.Equal(t, "chore/sync-files-skyetel-go-20250112-145757-561a06e", result.BranchName)
				assert.True(t, result.PRClosed)
				assert.True(t, result.BranchDeleted)
			},
		},
		{
			name:        "cancel all active syncs across all 4 groups",
			targetRepos: []string{},
			setupState: func() *state.State {
				return &state.State{
					Targets: map[string]*state.TargetState{
						"mrz1836/tool1": {
							Repo: "mrz1836/tool1",
							OpenPRs: []gh.PR{
								{Number: 101, State: "open"},
							},
							SyncBranches: []state.SyncBranch{
								{
									Name: "chore/sync-files-mrz-tools-20250110-100000-abc123",
									Metadata: &state.BranchMetadata{
										Timestamp: time.Date(2025, time.January, 10, 10, 0, 0, 0, time.UTC),
									},
								},
							},
						},
						"mrz1836/lib2": {
							Repo: "mrz1836/lib2",
							SyncBranches: []state.SyncBranch{
								{
									Name: "chore/sync-files-mrz-libraries-20250111-110000-def456",
									Metadata: &state.BranchMetadata{
										Timestamp: time.Date(2025, time.January, 11, 11, 0, 0, 0, time.UTC),
									},
								},
							},
						},
						"skyetel/reach": {
							Repo: "skyetel/reach",
							OpenPRs: []gh.PR{
								{Number: 430, State: "open"},
							},
							SyncBranches: []state.SyncBranch{
								{
									Name: "chore/sync-files-skyetel-go-20250112-145757-561a06e",
									Metadata: &state.BranchMetadata{
										Timestamp: time.Date(2025, time.January, 12, 14, 57, 57, 0, time.UTC),
									},
								},
							},
						},
					},
					Source: state.SourceState{
						Repo:         "mrz1836/go-broadcast",
						Branch:       "master",
						LatestCommit: "latest123",
					},
				}
			},
			setupMocks: func(mockClient *gh.MockClient) {
				// Mock PRs
				mockClient.On("ClosePR", mock.Anything, "mrz1836/tool1", 101, mock.AnythingOfType("string")).Return(nil)
				mockClient.On("ClosePR", mock.Anything, "skyetel/reach", 430, mock.AnythingOfType("string")).Return(nil)

				// Mock branches
				mockClient.On("DeleteBranch", mock.Anything, "mrz1836/tool1", "chore/sync-files-mrz-tools-20250110-100000-abc123").Return(nil)
				mockClient.On("DeleteBranch", mock.Anything, "mrz1836/lib2", "chore/sync-files-mrz-libraries-20250111-110000-def456").Return(nil)
				mockClient.On("DeleteBranch", mock.Anything, "skyetel/reach", "chore/sync-files-skyetel-go-20250112-145757-561a06e").Return(nil)
			},
			verifySummary: func(t *testing.T, summary *CancelSummary) {
				assert.Equal(t, 3, summary.TotalTargets)    // 3 repos with active syncs
				assert.Equal(t, 2, summary.PRsClosed)       // 2 had open PRs
				assert.Equal(t, 3, summary.BranchesDeleted) // All 3 had sync branches
				assert.Equal(t, 0, summary.Errors)

				// Verify skyetel/reach is included (regression test)
				skyetelFound := false
				for _, result := range summary.Results {
					if result.Repository == "skyetel/reach" {
						skyetelFound = true
						assert.Equal(t, 430, *result.PRNumber)
						assert.True(t, result.PRClosed)
						assert.True(t, result.BranchDeleted)
						break
					}
				}
				assert.True(t, skyetelFound, "skyetel/reach should be included in results")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure dry run is off (thread-safe)
			originalFlags := GetGlobalFlags()
			SetFlags(&Flags{ConfigFile: originalFlags.ConfigFile, DryRun: false, LogLevel: originalFlags.LogLevel})
			defer func() { SetFlags(originalFlags) }()

			// Create mocks
			mockClient := &gh.MockClient{}
			mockDiscoverer := &state.MockDiscoverer{}

			// Set up state
			testState := tt.setupState()
			mockDiscoverer.On("DiscoverState", context.Background(), cfg).Return(testState, nil)

			// Set up GitHub client mocks
			tt.setupMocks(mockClient)

			// Execute
			summary, err := performCancelWithDiscoverer(context.Background(), cfg, tt.targetRepos, mockClient, mockDiscoverer)

			// Verify
			require.NoError(t, err)
			assert.NotNil(t, summary)
			tt.verifySummary(t, summary)

			mockClient.AssertExpectations(t)
			mockDiscoverer.AssertExpectations(t)
		})
	}
}
