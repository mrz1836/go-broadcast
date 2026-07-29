package testutil

import (
	"os"
	"strings"
)

// aiEnvPrefix covers every go-broadcast AI setting, including ones added later
const aiEnvPrefix = "GO_BROADCAST_AI_"

// providerAPIKeyVars are the provider-specific keys the AI config falls back to
// when GO_BROADCAST_AI_API_KEY is unset.
var providerAPIKeyVars = []string{ //nolint:gochecknoglobals // fixed list of env var names
	"ANTHROPIC_API_KEY",
	"OPENAI_API_KEY",
	"GEMINI_API_KEY",
}

// DisableAIEnv clears every environment variable that could switch on live AI text
// generation, and returns a function that restores them.
//
// Call it from TestMain. Without it, a developer or CI machine that exports real
// credentials makes the test suite issue live, billable requests to a provider
// (api.anthropic.com and friends) whenever a test builds a sync Engine - which is
// slow, flaky, and depends on someone else's uptime. Tests that need AI behavior
// should point a provider at a local mock server instead.
//
// The prefix scan means a newly added GO_BROADCAST_AI_* setting is covered
// automatically rather than silently reopening the hole.
func DisableAIEnv() func() {
	saved := make(map[string]string)

	save := func(key string) {
		if value, ok := os.LookupEnv(key); ok {
			saved[key] = value
			_ = os.Unsetenv(key)
		}
	}

	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if found && strings.HasPrefix(key, aiEnvPrefix) {
			save(key)
		}
	}

	for _, key := range providerAPIKeyVars {
		save(key)
	}

	// Belt and braces: an explicit "off" beats an absent variable if any future
	// code path treats "unset" as a reason to probe elsewhere.
	_ = os.Setenv(aiEnvPrefix+"ENABLED", "false")

	return func() {
		_ = os.Unsetenv(aiEnvPrefix + "ENABLED")
		for key, value := range saved {
			_ = os.Setenv(key, value)
		}
	}
}
