package config

import (
	"context"
	"testing"

	"github.com/mrz1836/go-broadcast/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig_Validate tests the basic configuration validation
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantError   bool
		expectedErr error
	}{
		{
			name: "valid multi-source configuration",
			config: &Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							ID:     "test-source",
							Repo:   "org/template",
							Branch: "master",
						},
						Defaults: &DefaultConfig{
							BranchPrefix: "chore/sync-files",
							PRLabels:     []string{"automated-sync"},
						},
						Targets: []TargetConfig{
							{
								Repo: "org/service",
								Files: []FileMapping{
									{Src: "file.txt", Dest: "dest.txt"},
								},
							},
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "invalid version",
			config: &Config{
				Version: 2,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							ID:     "test-source",
							Repo:   "org/template",
							Branch: "master",
						},
						Targets: []TargetConfig{
							{
								Repo: "org/service",
								Files: []FileMapping{
									{Src: "file.txt", Dest: "dest.txt"},
								},
							},
						},
					},
				},
			},
			wantError:   true,
			expectedErr: ErrUnsupportedVersion,
		},
		{
			name: "no targets",
			config: &Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							ID:     "test-source",
							Repo:   "org/template",
							Branch: "master",
						},
						Targets: []TargetConfig{},
					},
				},
			},
			wantError: true,
		},
		{
			name: "duplicate target within same source mapping",
			config: &Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							ID:     "test-source",
							Repo:   "org/template",
							Branch: "master",
						},
						Targets: []TargetConfig{
							{
								Repo: "org/service",
								Files: []FileMapping{
									{Src: "file1.txt", Dest: "dest1.txt"},
								},
							},
							{
								Repo: "org/service",
								Files: []FileMapping{
									{Src: "file2.txt", Dest: "dest2.txt"},
								},
							},
						},
					},
				},
			},
			wantError:   true,
			expectedErr: ErrDuplicateTarget,
		},
		{
			name: "multiple sources targeting same repository (should be allowed)",
			config: &Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							ID:     "source1",
							Repo:   "org/template1",
							Branch: "master",
						},
						Targets: []TargetConfig{
							{
								Repo: "org/service",
								Files: []FileMapping{
									{Src: "file1.txt", Dest: "dest1.txt"},
								},
							},
						},
					},
					{
						Source: SourceConfig{
							ID:     "source2",
							Repo:   "org/template2",
							Branch: "master",
						},
						Targets: []TargetConfig{
							{
								Repo: "org/service", // Same repo as above - should be allowed
								Files: []FileMapping{
									{Src: "file2.txt", Dest: "dest2.txt"},
								},
							},
						},
					},
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantError {
				require.Error(t, err)
				if tt.expectedErr != nil {
					require.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestConfig_ValidateWithLogging tests validation with logging enabled
func TestConfig_ValidateWithLogging(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		logConfig *logging.LogConfig
		wantError bool
	}{
		{
			name: "valid configuration with debug logging",
			config: &Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							ID:     "test-source",
							Repo:   "org/template",
							Branch: "master",
						},
						Targets: []TargetConfig{
							{
								Repo: "org/service",
								Files: []FileMapping{
									{Src: "file.txt", Dest: "dest.txt"},
								},
							},
						},
					},
				},
			},
			logConfig: &logging.LogConfig{
				Debug: logging.DebugFlags{
					Config: true,
				},
			},
			wantError: false,
		},
		{
			name: "valid configuration with nil log config",
			config: &Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{
							ID:     "test-source",
							Repo:   "org/template",
							Branch: "master",
						},
						Targets: []TargetConfig{
							{
								Repo: "org/service",
								Files: []FileMapping{
									{Src: "file.txt", Dest: "dest.txt"},
								},
							},
						},
					},
				},
			},
			logConfig: nil,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.config.ValidateWithLogging(ctx, tt.logConfig)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestConfig_ValidateDirectories tests directory validation (the problematic test)
func TestConfig_ValidateDirectories(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "valid directory configuration",
			config: Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{ID: "test-source", Repo: "org/repo", Branch: "master"},
						Targets: []TargetConfig{{
							Repo: "org/target",
							Directories: []DirectoryMapping{{
								Src:     ".github/workflows",
								Dest:    ".github/workflows",
								Exclude: []string{"*.tmp", "test-*"},
							}},
						}},
					},
				},
			},
			wantErr: "",
		},
		{
			name: "valid mixed configuration",
			config: Config{
				Version: 1,
				Mappings: []SourceMapping{
					{
						Source: SourceConfig{ID: "test-source", Repo: "org/repo", Branch: "master"},
						Targets: []TargetConfig{{
							Repo: "org/target",
							Files: []FileMapping{{
								Src:  "Makefile",
								Dest: "Makefile",
							}},
							Directories: []DirectoryMapping{{
								Src:  ".github",
								Dest: ".github",
							}},
						}},
					},
				},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
