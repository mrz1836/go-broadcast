package config

import (
	"context"
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
	err = config.ValidateWithLogging(context.Background(), nil)
	require.NoError(t, err)

	// Verify it loaded correctly
	assert.Equal(t, 1, config.Version)

	// After normalization, source is in mappings format
	require.Len(t, config.Mappings, 1)
	assert.Equal(t, "org/template-repo", config.Mappings[0].Source.Repo)
	assert.Equal(t, "master", config.Mappings[0].Source.Branch)
	assert.Len(t, config.Mappings[0].Targets, 3)
}
