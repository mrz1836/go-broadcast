package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRateLimitPreflightDefaults(t *testing.T) {
	t.Run("nil config uses documented defaults", func(t *testing.T) {
		enabled, margin, reserve, failClosed := ResolveRateLimitPreflight(nil)
		assert.True(t, enabled)
		assert.Equal(t, DefaultRateLimitPrimaryMarginPercent, margin)
		assert.Equal(t, DefaultRateLimitSecondaryReserve, reserve)
		assert.False(t, failClosed)
	})

	t.Run("empty block uses documented defaults", func(t *testing.T) {
		enabled, margin, reserve, failClosed := ResolveRateLimitPreflight(&Config{})
		assert.True(t, enabled)
		assert.Equal(t, 20, margin)
		assert.Equal(t, 10, reserve)
		assert.False(t, failClosed)
	})

	t.Run("explicit values override defaults", func(t *testing.T) {
		disabled := false
		cfg := &Config{RateLimitPreflight: RateLimitPreflightConfig{
			Enabled:              &disabled,
			PrimaryMarginPercent: 50,
			SecondaryReserve:     25,
			FailClosed:           true,
		}}
		enabled, margin, reserve, failClosed := ResolveRateLimitPreflight(cfg)
		assert.False(t, enabled)
		assert.Equal(t, 50, margin)
		assert.Equal(t, 25, reserve)
		assert.True(t, failClosed)
	})

	t.Run("zero margin and reserve fall back to defaults", func(t *testing.T) {
		cfg := &Config{RateLimitPreflight: RateLimitPreflightConfig{
			PrimaryMarginPercent: 0,
			SecondaryReserve:     0,
		}}
		_, margin, reserve, _ := ResolveRateLimitPreflight(cfg)
		assert.Equal(t, DefaultRateLimitPrimaryMarginPercent, margin)
		assert.Equal(t, DefaultRateLimitSecondaryReserve, reserve)
	})
}

func TestApplyDefaultsRateLimitPreflight(t *testing.T) {
	cfg := &Config{}
	applyDefaults(cfg)

	require.NotNil(t, cfg.RateLimitPreflight.Enabled)
	assert.True(t, *cfg.RateLimitPreflight.Enabled)
	assert.Equal(t, DefaultRateLimitPrimaryMarginPercent, cfg.RateLimitPreflight.PrimaryMarginPercent)
	assert.Equal(t, DefaultRateLimitSecondaryReserve, cfg.RateLimitPreflight.SecondaryReserve)
	assert.False(t, cfg.RateLimitPreflight.FailClosed)
}

func TestApplyDefaultsRateLimitPreflightPreservesExplicit(t *testing.T) {
	disabled := false
	cfg := &Config{RateLimitPreflight: RateLimitPreflightConfig{
		Enabled:              &disabled,
		PrimaryMarginPercent: 33,
		SecondaryReserve:     7,
		FailClosed:           true,
	}}
	applyDefaults(cfg)

	require.NotNil(t, cfg.RateLimitPreflight.Enabled)
	assert.False(t, *cfg.RateLimitPreflight.Enabled)
	assert.Equal(t, 33, cfg.RateLimitPreflight.PrimaryMarginPercent)
	assert.Equal(t, 7, cfg.RateLimitPreflight.SecondaryReserve)
	assert.True(t, cfg.RateLimitPreflight.FailClosed)
}

func TestValidateRateLimitPreflight(t *testing.T) {
	tests := []struct {
		name    string
		block   RateLimitPreflightConfig
		wantErr error
	}{
		{name: "defaults valid", block: RateLimitPreflightConfig{PrimaryMarginPercent: 20, SecondaryReserve: 10}},
		{name: "zero valid (means default)", block: RateLimitPreflightConfig{}},
		{name: "margin 100 valid", block: RateLimitPreflightConfig{PrimaryMarginPercent: 100}},
		{name: "margin negative invalid", block: RateLimitPreflightConfig{PrimaryMarginPercent: -1}, wantErr: ErrInvalidRateLimitMargin},
		{name: "margin over 100 invalid", block: RateLimitPreflightConfig{PrimaryMarginPercent: 101}, wantErr: ErrInvalidRateLimitMargin},
		{name: "reserve negative invalid", block: RateLimitPreflightConfig{SecondaryReserve: -5}, wantErr: ErrInvalidRateLimitReserve},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{RateLimitPreflight: tt.block}
			err := cfg.validateRateLimitPreflight()
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRejectsBadRateLimitPreflight(t *testing.T) {
	// Full Validate() should surface the rate-limit error before group checks.
	cfg := &Config{
		Version:            1,
		RateLimitPreflight: RateLimitPreflightConfig{PrimaryMarginPercent: 250},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRateLimitMargin)
}

func TestLoadRateLimitPreflightFromYAML(t *testing.T) {
	yamlData := `
version: 1
rate_limit_preflight:
  enabled: false
  primary_margin_percent: 40
  secondary_reserve: 15
  fail_closed: true
groups:
  - name: core
    id: core
    source:
      repo: org/source
      branch: main
    targets:
      - repo: org/target
        files:
          - src: a.txt
            dest: a.txt
`
	cfg, err := LoadFromReader(strings.NewReader(yamlData))
	require.NoError(t, err)
	require.NotNil(t, cfg.RateLimitPreflight.Enabled)
	assert.False(t, *cfg.RateLimitPreflight.Enabled)
	assert.Equal(t, 40, cfg.RateLimitPreflight.PrimaryMarginPercent)
	assert.Equal(t, 15, cfg.RateLimitPreflight.SecondaryReserve)
	assert.True(t, cfg.RateLimitPreflight.FailClosed)
}
