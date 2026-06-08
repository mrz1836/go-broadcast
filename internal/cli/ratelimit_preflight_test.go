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

func TestCurrentRateLimitOverridesCapturesChangedFlags(t *testing.T) {
	// When the user explicitly sets the CLI flags, currentRateLimitOverrides must
	// capture each one (cobra's Changed() path) so they override config. Restore
	// the global flag state and the per-flag Changed markers afterwards so this
	// test does not leak into others that read currentRateLimitOverrides.
	t.Cleanup(func() {
		for _, name := range []string{
			flagRateLimitPreflight,
			flagRateLimitMarginPercent,
			flagRateLimitSecondaryReserve,
			flagRateLimitFailClosed,
			flagIgnoreRateLimitPreflight,
		} {
			if f := syncFlagSet.Lookup(name); f != nil {
				f.Changed = false
			}
		}
		rateLimitPreflight = true
		rateLimitMarginPercent = config.DefaultRateLimitPrimaryMarginPercent
		rateLimitSecondaryReserve = config.DefaultRateLimitSecondaryReserve
		rateLimitFailClosed = false
		ignoreRateLimitPreflight = false
	})

	require.NoError(t, syncFlagSet.Set(flagRateLimitPreflight, "false"))
	require.NoError(t, syncFlagSet.Set(flagRateLimitMarginPercent, "35"))
	require.NoError(t, syncFlagSet.Set(flagRateLimitSecondaryReserve, "7"))
	require.NoError(t, syncFlagSet.Set(flagRateLimitFailClosed, "true"))
	require.NoError(t, syncFlagSet.Set(flagIgnoreRateLimitPreflight, "true"))

	ov := currentRateLimitOverrides()

	require.NotNil(t, ov.enabled)
	assert.False(t, *ov.enabled)
	require.NotNil(t, ov.margin)
	assert.Equal(t, 35, *ov.margin)
	require.NotNil(t, ov.reserve)
	assert.Equal(t, 7, *ov.reserve)
	require.NotNil(t, ov.failClosed)
	assert.True(t, *ov.failClosed)
	assert.True(t, ov.ignore)
}

func TestCurrentRateLimitOverridesNilFlagSet(t *testing.T) {
	// If the flag set was never registered (an alternate command path), only the
	// always-carried ignore value is returned and the rest fall through to config.
	saved := syncFlagSet
	t.Cleanup(func() { syncFlagSet = saved })

	syncFlagSet = nil
	ov := currentRateLimitOverrides()
	assert.Nil(t, ov.enabled)
	assert.Nil(t, ov.margin)
	assert.Nil(t, ov.reserve)
	assert.Nil(t, ov.failClosed)
}
