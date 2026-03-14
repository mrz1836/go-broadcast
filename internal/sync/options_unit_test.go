package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions_AIEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{name: "enable AI", enabled: true, expected: true},
		{name: "disable AI", enabled: false, expected: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := DefaultOptions().WithAIEnabled(tc.enabled)
			assert.Equal(t, tc.expected, opts.AIEnabled)
		})
	}
}

func TestOptions_AIPREnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{name: "enable AI PR", enabled: true, expected: true},
		{name: "disable AI PR", enabled: false, expected: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := DefaultOptions().WithAIPREnabled(tc.enabled)
			assert.Equal(t, tc.expected, opts.AIPREnabled)
		})
	}
}

func TestOptions_AICommitEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{name: "enable AI commit", enabled: true, expected: true},
		{name: "disable AI commit", enabled: false, expected: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := DefaultOptions().WithAICommitEnabled(tc.enabled)
			assert.Equal(t, tc.expected, opts.AICommitEnabled)
		})
	}
}

func TestOptions_ClearModuleCache(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{name: "enable clear cache", enabled: true, expected: true},
		{name: "disable clear cache", enabled: false, expected: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opts := DefaultOptions().WithClearModuleCache(tc.enabled)
			assert.Equal(t, tc.expected, opts.ClearModuleCache)
		})
	}
}
