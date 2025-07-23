package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExampleConfigLoadsAndValidates(t *testing.T) {
	// Test that our example configuration is valid
	config, err := Load("../../examples/sync.yaml")
	require.NoError(t, err)
	require.NotNil(t, config)

	// Validate the configuration
	err = config.Validate()
	require.NoError(t, err)

	// Verify it loaded correctly
	assert.Equal(t, 1, config.Version)
	assert.Equal(t, "org/template-repo", config.Source.Repo)
	assert.Equal(t, "master", config.Source.Branch)
	assert.Len(t, config.Targets, 3)
}
