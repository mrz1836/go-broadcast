package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-broadcast/internal/config"
	"github.com/mrz1836/go-broadcast/internal/sync"
)

func TestMergeRateLimitPreflightConfigBase(t *testing.T) {
	// No CLI overrides: options come from config (with defaults applied).
	cfg := &config.Config{}
	require.NoError(t, config.ApplyDefaultsAndResolve(cfg))

	opts := mergeRateLimitPreflight(sync.DefaultOptions(), cfg, rateLimitPreflightOverrides{})

	assert.True(t, opts.RateLimitPreflightEnabled)
	assert.Equal(t, 20, opts.RateLimitPrimaryMarginPercent)
	assert.Equal(t, 10, opts.RateLimitSecondaryReserve)
	assert.False(t, opts.RateLimitFailClosed)
	assert.False(t, opts.IgnoreRateLimitPreflight)
}

func TestMergeRateLimitPreflightConfigValues(t *testing.T) {
	// Config block provides non-default values; no CLI overrides.
	cfg := &config.Config{RateLimitPreflight: config.RateLimitPreflightConfig{
		Enabled:              boolPtr(false),
		PrimaryMarginPercent: 45,
		SecondaryReserve:     5,
		FailClosed:           true,
	}}

	opts := mergeRateLimitPreflight(sync.DefaultOptions(), cfg, rateLimitPreflightOverrides{})

	assert.False(t, opts.RateLimitPreflightEnabled)
	assert.Equal(t, 45, opts.RateLimitPrimaryMarginPercent)
	assert.Equal(t, 5, opts.RateLimitSecondaryReserve)
	assert.True(t, opts.RateLimitFailClosed)
}

func TestMergeRateLimitPreflightFlagsOverrideConfig(t *testing.T) {
	// Config says enabled with a 45% margin; CLI overrides flip several fields.
	cfg := &config.Config{RateLimitPreflight: config.RateLimitPreflightConfig{
		Enabled:              boolPtr(true),
		PrimaryMarginPercent: 45,
		SecondaryReserve:     5,
		FailClosed:           false,
	}}

	ov := rateLimitPreflightOverrides{
		enabled:    boolPtr(false),
		ignore:     true,
		margin:     intPtr(10),
		reserve:    intPtr(2),
		failClosed: boolPtr(true),
	}

	opts := mergeRateLimitPreflight(sync.DefaultOptions(), cfg, ov)

	assert.False(t, opts.RateLimitPreflightEnabled)         // overridden
	assert.Equal(t, 10, opts.RateLimitPrimaryMarginPercent) // overridden
	assert.Equal(t, 2, opts.RateLimitSecondaryReserve)      // overridden
	assert.True(t, opts.RateLimitFailClosed)                // overridden
	assert.True(t, opts.IgnoreRateLimitPreflight)           // CLI-only escape hatch
}

func TestMergeRateLimitPreflightPartialOverride(t *testing.T) {
	// Only the margin is overridden; everything else stays at config values.
	cfg := &config.Config{RateLimitPreflight: config.RateLimitPreflightConfig{
		Enabled:              boolPtr(true),
		PrimaryMarginPercent: 45,
		SecondaryReserve:     5,
	}}

	opts := mergeRateLimitPreflight(sync.DefaultOptions(), cfg, rateLimitPreflightOverrides{margin: intPtr(30)})

	assert.True(t, opts.RateLimitPreflightEnabled)
	assert.Equal(t, 30, opts.RateLimitPrimaryMarginPercent) // overridden
	assert.Equal(t, 5, opts.RateLimitSecondaryReserve)      // from config
}

func TestCurrentRateLimitOverridesDefaultsNoChange(t *testing.T) {
	// With no flags explicitly changed, only the (always-carried) ignore value is
	// returned; the rest are nil so config provides the base.
	ResetGlobalFlags()
	ov := currentRateLimitOverrides()
	assert.Nil(t, ov.enabled)
	assert.Nil(t, ov.margin)
	assert.Nil(t, ov.reserve)
	assert.Nil(t, ov.failClosed)
}
