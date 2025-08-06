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
	require.Len(t, config.Groups, 1)
	group := config.Groups[0]
	assert.Equal(t, "org/template-repo", group.Source.Repo)
	assert.Equal(t, "master", group.Source.Branch)
	assert.Len(t, group.Targets, 3)
}
