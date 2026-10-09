package config

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/logging"
)

func TestValidateCloneMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{name: "empty uses default", mode: ""},
		{name: "sparse", mode: CloneModeSparse},
		{name: "full", mode: CloneModeFull},
		{name: "unknown value", mode: "shallow", wantErr: true},
		{name: "wrong case", mode: "Full", wantErr: true},
		{name: "surrounding whitespace", mode: " full", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCloneMode(tt.mode)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidCloneMode)
				assert.Contains(t, err.Error(), tt.mode)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestResolveCloneMode(t *testing.T) {
	assert.Equal(t, CloneModeSparse, ResolveCloneMode(""))
	assert.Equal(t, CloneModeSparse, ResolveCloneMode(CloneModeSparse))
	assert.Equal(t, CloneModeFull, ResolveCloneMode(CloneModeFull))
}

func TestTargetConfig_ValidateCloneMode(t *testing.T) {
	newTarget := func(mode string) *TargetConfig {
		return &TargetConfig{
			Repo:      "org/target",
			CloneMode: mode,
			Files:     []FileMapping{{Src: "file.txt", Dest: "dest.txt"}},
		}
	}
	logger := logrus.WithField("test", "clone_mode")
	debugConfig := &logging.LogConfig{Debug: logging.DebugFlags{Config: true}}

	for _, mode := range []string{"", CloneModeSparse, CloneModeFull} {
		require.NoError(t, newTarget(mode).validateWithLogging(context.Background(), nil, logger), "mode %q", mode)
	}

	err := newTarget("partial").validateWithLogging(context.Background(), nil, logger)
	require.ErrorIs(t, err, ErrInvalidCloneMode)

	err = newTarget("partial").validateWithLogging(context.Background(), debugConfig, logger)
	require.ErrorIs(t, err, ErrInvalidCloneMode)
}

func TestLoadFromReader_TargetCloneMode(t *testing.T) {
	const yamlConfig = `version: 1
groups:
  - name: "Group"
    id: "group"
    source:
      repo: "org/template"
    targets:
      - repo: "org/huge-repo"
        clone_mode: "full"
        files:
          - src: "a.txt"
            dest: "a.txt"
      - repo: "org/normal-repo"
        files:
          - src: "a.txt"
            dest: "a.txt"
`
	cfg, err := LoadFromReader(strings.NewReader(yamlConfig))
	require.NoError(t, err)
	require.Len(t, cfg.Groups, 1)
	require.Len(t, cfg.Groups[0].Targets, 2)

	assert.Equal(t, CloneModeFull, cfg.Groups[0].Targets[0].CloneMode)
	assert.Empty(t, cfg.Groups[0].Targets[1].CloneMode, "unset clone_mode must stay empty so the default applies")
	require.NoError(t, cfg.Validate())
}
