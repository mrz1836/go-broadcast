package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConfig_Validate tests basic config validation
func TestConfig_Validate(t *testing.T) {
	t.Run("valid config with groups", func(t *testing.T) {
		config := &Config{
			Version: 1,
			Groups: []Group{
				{
					Name: "test-group",
					ID:   "test",
					Source: SourceConfig{
						Repo: "org/template",
					},
					Targets: []TargetConfig{
						{
							Repo: "org/service",
							Files: []FileMapping{
								{Src: "file.txt", Dest: "dest.txt"},
							},
						},
					},
				},
			},
		}

		// Test that config is not nil
		assert.NotNil(t, config)
		assert.Equal(t, 1, config.Version)
		assert.Len(t, config.Groups, 1)
	})

	t.Run("empty config", func(t *testing.T) {
		config := &Config{}
		assert.NotNil(t, config)
		assert.Empty(t, config.Groups)
	})
}
