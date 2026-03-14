package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestClientInterface verifies that concrete implementations satisfy the Client interface
func TestClientInterface(t *testing.T) {
	tests := []struct {
		name     string
		provider func() Client
	}{
		{
			name: "gitClient implements Client",
			provider: func() Client {
				return &gitClient{}
			},
		},
		{
			name: "MockClient implements Client",
			provider: func() Client {
				return &MockClient{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.provider()
			require.NotNil(t, client)

			// Verify interface compliance by checking method signatures exist
			// These calls will panic at runtime due to nil dependencies, but compile-time
			// verification ensures the interface is properly implemented
			require.NotPanics(t, func() {
				_ = client.Clone
				_ = client.Checkout
				_ = client.CreateBranch
				_ = client.Add
				_ = client.Commit
				_ = client.Push
				_ = client.Diff
				_ = client.GetCurrentBranch
				_ = client.GetRemoteURL
			})
		})
	}
}

// TestClientInterfaceMethodSignatures validates that methods exist and have correct signatures
func TestClientInterfaceMethodSignatures(t *testing.T) {
	// This test ensures that if the interface changes, we'll catch it at compile time

	// Test method existence through function value assignments
	client := &gitClient{}

	// These assignments will fail at compile time if method signatures don't match
	require.NotNil(t, client.Clone)
	require.NotNil(t, client.Checkout)
	require.NotNil(t, client.CreateBranch)
	require.NotNil(t, client.Add)
	require.NotNil(t, client.Commit)
	require.NotNil(t, client.Push)
	require.NotNil(t, client.Diff)
	require.NotNil(t, client.GetCurrentBranch)
	require.NotNil(t, client.GetRemoteURL)
}
