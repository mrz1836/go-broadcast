package testutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisableAIEnv(t *testing.T) {
	t.Run("clears provider keys and go-broadcast AI settings", func(t *testing.T) {
		t.Setenv("GO_BROADCAST_AI_ENABLED", "true")
		t.Setenv("GO_BROADCAST_AI_API_KEY", "secret")
		t.Setenv("GO_BROADCAST_AI_PROVIDER", "anthropic")
		t.Setenv("ANTHROPIC_API_KEY", "secret")
		t.Setenv("OPENAI_API_KEY", "secret")
		t.Setenv("GEMINI_API_KEY", "secret")

		restore := DisableAIEnv()

		assert.Equal(t, "false", os.Getenv("GO_BROADCAST_AI_ENABLED"))

		for _, key := range []string{
			"GO_BROADCAST_AI_API_KEY",
			"GO_BROADCAST_AI_PROVIDER",
			"ANTHROPIC_API_KEY",
			"OPENAI_API_KEY",
			"GEMINI_API_KEY",
		} {
			_, present := os.LookupEnv(key)
			assert.False(t, present, "%s must be cleared so tests cannot reach a live provider", key)
		}

		restore()

		assert.Equal(t, "secret", os.Getenv("ANTHROPIC_API_KEY"))
		assert.Equal(t, "secret", os.Getenv("GO_BROADCAST_AI_API_KEY"))
		assert.Equal(t, "true", os.Getenv("GO_BROADCAST_AI_ENABLED"))
	})

	t.Run("clears unknown GO_BROADCAST_AI_ variables via prefix scan", func(t *testing.T) {
		// Stands in for a setting added after this helper was written
		t.Setenv("GO_BROADCAST_AI_SOME_FUTURE_SETTING", "value")

		restore := DisableAIEnv()

		_, present := os.LookupEnv("GO_BROADCAST_AI_SOME_FUTURE_SETTING")
		assert.False(t, present, "prefix scan should cover settings added later")

		restore()

		assert.Equal(t, "value", os.Getenv("GO_BROADCAST_AI_SOME_FUTURE_SETTING"))
	})

	t.Run("leaves no residue when the variable was absent", func(t *testing.T) {
		// t.Setenv registers restoration of the ambient value, then we clear it to
		// model a machine with no AI configuration at all
		t.Setenv("GO_BROADCAST_AI_ENABLED", "placeholder")
		require.NoError(t, os.Unsetenv("GO_BROADCAST_AI_ENABLED"))

		restore := DisableAIEnv()
		require.NotNil(t, restore)

		assert.Equal(t, "false", os.Getenv("GO_BROADCAST_AI_ENABLED"))

		restore()

		// The synthetic "false" must not linger as a real value
		_, present := os.LookupEnv("GO_BROADCAST_AI_ENABLED")
		assert.False(t, present, "restore should leave an absent variable absent")
	})
}
