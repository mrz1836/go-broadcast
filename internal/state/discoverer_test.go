package state

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDiscovererInterface verifies that concrete implementations satisfy the Discoverer interface
func TestDiscovererInterface(t *testing.T) {
	tests := []struct {
		name     string
		provider func() Discoverer
	}{
		{
			name: "discoveryService implements Discoverer",
			provider: func() Discoverer {
				return NewDiscoverer(nil, nil, nil)
			},
		},
		{
			name: "MockDiscoverer implements Discoverer",
			provider: func() Discoverer {
				return &MockDiscoverer{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discoverer := tt.provider()
			require.NotNil(t, discoverer)

			// Verify interface compliance by checking method signatures exist
			// These calls will panic at runtime due to nil clients, but compile-time
			// verification ensures the interface is properly implemented
			require.NotPanics(t, func() {
				_ = discoverer.DiscoverState
				_ = discoverer.DiscoverTargetState
				_ = discoverer.ParseBranchName
			})
		})
	}
}

// TestDiscovererInterfaceMethodSignatures validates that methods exist and have correct signatures
func TestDiscovererInterfaceMethodSignatures(t *testing.T) {
	// This test ensures that if the interface changes, we'll catch it at compile time

	// Test method existence through function value assignments
	var discoverer Discoverer = &MockDiscoverer{}

	// These assignments will fail to compile if method signatures change
	discoverStateFunc := discoverer.DiscoverState
	discoverTargetStateFunc := discoverer.DiscoverTargetState
	parseBranchNameFunc := discoverer.ParseBranchName

	// Verify functions are not nil (compile-time check)
	require.NotNil(t, discoverStateFunc, "DiscoverState method should exist")
	require.NotNil(t, discoverTargetStateFunc, "DiscoverTargetState method should exist")
	require.NotNil(t, parseBranchNameFunc, "ParseBranchName method should exist")
}

// TestDiscovererInterfaceCompilance ensures the interface methods have the correct types
func TestDiscovererInterfaceCompliance(t *testing.T) {
	// Test that both concrete and mock implementations can be assigned to interface
	var discoverers []Discoverer

	// Add concrete implementation
	discoverers = append(discoverers, NewDiscoverer(nil, nil, nil))

	// Add mock implementation
	discoverers = append(discoverers, &MockDiscoverer{})

	require.Len(t, discoverers, 2)

	// Verify all implementations satisfy the interface
	for i, d := range discoverers {
		require.NotNil(t, d, "Discoverer at index %d should not be nil", i)

		// Verify interface compliance by accessing interface methods
		require.NotNil(t, d.DiscoverState, "DiscoverState method should exist")
		require.NotNil(t, d.DiscoverTargetState, "DiscoverTargetState method should exist")
		require.NotNil(t, d.ParseBranchName, "ParseBranchName method should exist")
	}
}
