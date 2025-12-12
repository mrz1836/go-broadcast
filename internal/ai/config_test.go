package ai

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear all AI-related env vars to test defaults
	envVars := []string{
		"GO_BROADCAST_AI_ENABLED",
		"GO_BROADCAST_AI_PR_ENABLED",
		"GO_BROADCAST_AI_COMMIT_ENABLED",
		"GO_BROADCAST_AI_PROVIDER",
		"GO_BROADCAST_AI_API_KEY",
		"GO_BROADCAST_AI_MODEL",
		"GO_BROADCAST_AI_MAX_TOKENS",
		"GO_BROADCAST_AI_TIMEOUT",
		"GO_BROADCAST_AI_TEMPERATURE",
		"GO_BROADCAST_AI_DIFF_MAX_CHARS",
		"GO_BROADCAST_AI_DIFF_MAX_LINES_PER_FILE",
		"GO_BROADCAST_AI_CACHE_ENABLED",
		"GO_BROADCAST_AI_CACHE_TTL",
		"GO_BROADCAST_AI_CACHE_MAX_SIZE",
		"GO_BROADCAST_AI_RETRY_MAX_ATTEMPTS",
		"GO_BROADCAST_AI_RETRY_INITIAL_DELAY",
		"GO_BROADCAST_AI_RETRY_MAX_DELAY",
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"GEMINI_API_KEY",
	}

	for _, v := range envVars {
		t.Setenv(v, "")
	}

	cfg := LoadConfig()

	require.NotNil(t, cfg)
	// Master switch defaults to false
	assert.False(t, cfg.Enabled)
	assert.False(t, cfg.PREnabled)
	assert.False(t, cfg.CommitEnabled)

	// Provider defaults
	assert.Equal(t, ProviderAnthropic, cfg.Provider)
	assert.Empty(t, cfg.APIKey)
	assert.Equal(t, "claude-sonnet-4-20250514", cfg.Model) // default model for anthropic

	// Generation parameters
	assert.Equal(t, 2000, cfg.MaxTokens)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.InDelta(t, 0.3, cfg.Temperature, 0.001)

	// Diff truncation
	assert.Equal(t, 4000, cfg.DiffMaxChars)
	assert.Equal(t, 50, cfg.DiffMaxLinesPerFile)

	// Cache defaults
	assert.True(t, cfg.CacheEnabled)
	assert.Equal(t, 1*time.Hour, cfg.CacheTTL)
	assert.Equal(t, 1000, cfg.CacheMaxSize)

	// Retry defaults
	assert.Equal(t, 3, cfg.RetryMaxAttempts)
	assert.Equal(t, 1*time.Second, cfg.RetryInitialDelay)
	assert.Equal(t, 10*time.Second, cfg.RetryMaxDelay)
}

func TestLoadConfig_AllEnvVars(t *testing.T) {
	// Set all environment variables
	t.Setenv("GO_BROADCAST_AI_ENABLED", "true")
	t.Setenv("GO_BROADCAST_AI_PR_ENABLED", "true")
	t.Setenv("GO_BROADCAST_AI_COMMIT_ENABLED", "false")
	t.Setenv("GO_BROADCAST_AI_PROVIDER", "openai")
	t.Setenv("GO_BROADCAST_AI_API_KEY", "test-api-key")
	t.Setenv("GO_BROADCAST_AI_MODEL", "gpt-4-turbo")
	t.Setenv("GO_BROADCAST_AI_MAX_TOKENS", "4000")
	t.Setenv("GO_BROADCAST_AI_TIMEOUT", "60")
	t.Setenv("GO_BROADCAST_AI_TEMPERATURE", "0.7")
	t.Setenv("GO_BROADCAST_AI_DIFF_MAX_CHARS", "8000")
	t.Setenv("GO_BROADCAST_AI_DIFF_MAX_LINES_PER_FILE", "100")
	t.Setenv("GO_BROADCAST_AI_CACHE_ENABLED", "false")
	t.Setenv("GO_BROADCAST_AI_CACHE_TTL", "7200")
	t.Setenv("GO_BROADCAST_AI_CACHE_MAX_SIZE", "500")
	t.Setenv("GO_BROADCAST_AI_RETRY_MAX_ATTEMPTS", "5")
	t.Setenv("GO_BROADCAST_AI_RETRY_INITIAL_DELAY", "2")
	t.Setenv("GO_BROADCAST_AI_RETRY_MAX_DELAY", "30")

	cfg := LoadConfig()

	require.NotNil(t, cfg)
	assert.True(t, cfg.Enabled)
	assert.True(t, cfg.PREnabled)
	assert.False(t, cfg.CommitEnabled)
	assert.Equal(t, "openai", cfg.Provider)
	assert.Equal(t, "test-api-key", cfg.APIKey)
	assert.Equal(t, "gpt-4-turbo", cfg.Model)
	assert.Equal(t, 4000, cfg.MaxTokens)
	assert.Equal(t, 60*time.Second, cfg.Timeout)
	assert.InDelta(t, 0.7, cfg.Temperature, 0.001)
	assert.Equal(t, 8000, cfg.DiffMaxChars)
	assert.Equal(t, 100, cfg.DiffMaxLinesPerFile)
	assert.False(t, cfg.CacheEnabled)
	assert.Equal(t, 2*time.Hour, cfg.CacheTTL)
	assert.Equal(t, 500, cfg.CacheMaxSize)
	assert.Equal(t, 5, cfg.RetryMaxAttempts)
	assert.Equal(t, 2*time.Second, cfg.RetryInitialDelay)
	assert.Equal(t, 30*time.Second, cfg.RetryMaxDelay)
}

func TestLoadConfig_ProviderAPIKeyFallback(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		providerEnvVar string
		expectedKey    string
	}{
		{
			name:           "Anthropic fallback",
			provider:       "anthropic",
			providerEnvVar: "ANTHROPIC_API_KEY",
			expectedKey:    "anthropic-key",
		},
		{
			name:           "OpenAI fallback",
			provider:       "openai",
			providerEnvVar: "OPENAI_API_KEY",
			expectedKey:    "openai-key",
		},
		{
			name:           "Google fallback",
			provider:       "google",
			providerEnvVar: "GEMINI_API_KEY",
			expectedKey:    "gemini-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear main API key
			t.Setenv("GO_BROADCAST_AI_API_KEY", "")
			t.Setenv("GO_BROADCAST_AI_PROVIDER", tt.provider)
			t.Setenv(tt.providerEnvVar, tt.expectedKey)

			// Clear other provider keys
			otherKeys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GEMINI_API_KEY"}
			for _, k := range otherKeys {
				if k != tt.providerEnvVar {
					t.Setenv(k, "")
				}
			}

			cfg := LoadConfig()

			assert.Equal(t, tt.expectedKey, cfg.APIKey)
		})
	}
}

func TestLoadConfig_MainAPIKeyTakesPrecedence(t *testing.T) {
	t.Setenv("GO_BROADCAST_AI_API_KEY", "main-api-key")
	t.Setenv("GO_BROADCAST_AI_PROVIDER", "anthropic")
	t.Setenv("ANTHROPIC_API_KEY", "provider-specific-key")

	cfg := LoadConfig()

	assert.Equal(t, "main-api-key", cfg.APIKey, "main API key should take precedence")
}

func TestLoadConfig_InvalidValues(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		invalidValue string
		checkField   func(*Config) interface{}
		expectValue  interface{}
	}{
		{
			name:         "invalid max tokens uses default",
			envVar:       "GO_BROADCAST_AI_MAX_TOKENS",
			invalidValue: "not-a-number",
			checkField:   func(c *Config) interface{} { return c.MaxTokens },
			expectValue:  2000,
		},
		{
			name:         "invalid timeout uses default",
			envVar:       "GO_BROADCAST_AI_TIMEOUT",
			invalidValue: "invalid",
			checkField:   func(c *Config) interface{} { return c.Timeout },
			expectValue:  30 * time.Second,
		},
		{
			name:         "invalid temperature uses default",
			envVar:       "GO_BROADCAST_AI_TEMPERATURE",
			invalidValue: "abc",
			checkField:   func(c *Config) interface{} { return c.Temperature },
			expectValue:  0.3,
		},
		{
			name:         "invalid cache TTL uses default",
			envVar:       "GO_BROADCAST_AI_CACHE_TTL",
			invalidValue: "xyz",
			checkField:   func(c *Config) interface{} { return c.CacheTTL },
			expectValue:  1 * time.Hour,
		},
		{
			name:         "invalid retry max attempts uses default",
			envVar:       "GO_BROADCAST_AI_RETRY_MAX_ATTEMPTS",
			invalidValue: "three",
			checkField:   func(c *Config) interface{} { return c.RetryMaxAttempts },
			expectValue:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envVar, tt.invalidValue)

			cfg := LoadConfig()

			assert.Equal(t, tt.expectValue, tt.checkField(cfg))
		})
	}
}

func TestLoadConfig_DefaultModel(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		expectedModel string
	}{
		{
			name:          "Anthropic default model",
			provider:      "anthropic",
			expectedModel: "claude-sonnet-4-20250514",
		},
		{
			name:          "OpenAI default model",
			provider:      "openai",
			expectedModel: "gpt-4o",
		},
		{
			name:          "Google default model",
			provider:      "google",
			expectedModel: "gemini-2.5-flash",
		},
		{
			name:          "Unknown provider empty model",
			provider:      "unknown",
			expectedModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GO_BROADCAST_AI_PROVIDER", tt.provider)
			t.Setenv("GO_BROADCAST_AI_MODEL", "") // Clear to test default

			cfg := LoadConfig()

			assert.Equal(t, tt.expectedModel, cfg.Model)
		})
	}
}

func TestLoadConfig_CustomModelOverridesDefault(t *testing.T) {
	t.Setenv("GO_BROADCAST_AI_PROVIDER", "anthropic")
	t.Setenv("GO_BROADCAST_AI_MODEL", "claude-3-opus")

	cfg := LoadConfig()

	assert.Equal(t, "claude-3-opus", cfg.Model)
}

func TestConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		apiKey     string
		wantResult bool
	}{
		{
			name:       "disabled returns false",
			enabled:    false,
			apiKey:     "key",
			wantResult: false,
		},
		{
			name:       "enabled without API key returns false",
			enabled:    true,
			apiKey:     "",
			wantResult: false,
		},
		{
			name:       "enabled with API key returns true",
			enabled:    true,
			apiKey:     "key",
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Enabled: tt.enabled,
				APIKey:  tt.apiKey,
			}

			assert.Equal(t, tt.wantResult, cfg.IsEnabled())
		})
	}
}

func TestConfig_IsPREnabled(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		apiKey     string
		prEnabled  bool
		wantResult bool
	}{
		{
			name:       "all enabled returns true",
			enabled:    true,
			apiKey:     "key",
			prEnabled:  true,
			wantResult: true,
		},
		{
			name:       "PR disabled returns false",
			enabled:    true,
			apiKey:     "key",
			prEnabled:  false,
			wantResult: false,
		},
		{
			name:       "master disabled returns false",
			enabled:    false,
			apiKey:     "key",
			prEnabled:  true,
			wantResult: false,
		},
		{
			name:       "no API key returns false",
			enabled:    true,
			apiKey:     "",
			prEnabled:  true,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Enabled:   tt.enabled,
				APIKey:    tt.apiKey,
				PREnabled: tt.prEnabled,
			}

			assert.Equal(t, tt.wantResult, cfg.IsPREnabled())
		})
	}
}

func TestConfig_IsCommitEnabled(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		apiKey        string
		commitEnabled bool
		wantResult    bool
	}{
		{
			name:          "all enabled returns true",
			enabled:       true,
			apiKey:        "key",
			commitEnabled: true,
			wantResult:    true,
		},
		{
			name:          "commit disabled returns false",
			enabled:       true,
			apiKey:        "key",
			commitEnabled: false,
			wantResult:    false,
		},
		{
			name:          "master disabled returns false",
			enabled:       false,
			apiKey:        "key",
			commitEnabled: true,
			wantResult:    false,
		},
		{
			name:          "no API key returns false",
			enabled:       true,
			apiKey:        "",
			commitEnabled: true,
			wantResult:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Enabled:       tt.enabled,
				APIKey:        tt.apiKey,
				CommitEnabled: tt.commitEnabled,
			}

			assert.Equal(t, tt.wantResult, cfg.IsCommitEnabled())
		})
	}
}

func TestGetDefaultModel(t *testing.T) {
	tests := []struct {
		provider      string
		expectedModel string
	}{
		{ProviderAnthropic, "claude-sonnet-4-20250514"},
		{ProviderOpenAI, "gpt-4o"},
		{ProviderGoogle, "gemini-2.5-flash"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			model := GetDefaultModel(tt.provider)
			assert.Equal(t, tt.expectedModel, model)
		})
	}
}

func TestParseIntWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue int
		expected     int
	}{
		{"empty uses default", "", 100, 100},
		{"valid int parsed", "42", 100, 42},
		{"invalid uses default", "not-int", 100, 100},
		{"zero is valid", "0", 100, 0},
		{"negative is valid", "-5", 100, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_INT_VAR", tt.value)
			result := parseIntWithDefault("TEST_INT_VAR", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFloatWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue float64
		expected     float64
	}{
		{"empty uses default", "", 0.5, 0.5},
		{"valid float parsed", "0.7", 0.5, 0.7},
		{"invalid uses default", "not-float", 0.5, 0.5},
		{"zero is valid", "0", 0.5, 0.0},
		{"integer as float", "1", 0.5, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_FLOAT_VAR", tt.value)
			result := parseFloatWithDefault("TEST_FLOAT_VAR", tt.defaultValue)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestParseDurationSecondsWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{"empty uses default", "", 30 * time.Second, 30 * time.Second},
		{"valid seconds parsed", "60", 30 * time.Second, 60 * time.Second},
		{"invalid uses default", "not-number", 30 * time.Second, 30 * time.Second},
		{"zero is valid", "0", 30 * time.Second, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_DURATION_VAR", tt.value)
			result := parseDurationSecondsWithDefault("TEST_DURATION_VAR", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseBoolWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue bool
		expected     bool
	}{
		{"empty uses default true", "", true, true},
		{"empty uses default false", "", false, false},
		{"true string", "true", false, true},
		{"false string", "false", true, false},
		{"non-true string is false", "yes", true, false},
		{"1 is not true", "1", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_BOOL_VAR", tt.value)
			result := parseBoolWithDefault("TEST_BOOL_VAR", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	// Helper to create a valid config
	validConfig := func() *Config {
		return &Config{
			Provider:            ProviderAnthropic,
			Temperature:         0.3,
			MaxTokens:           2000,
			Timeout:             30 * time.Second,
			DiffMaxChars:        4000,
			DiffMaxLinesPerFile: 50,
			CacheEnabled:        true,
			CacheMaxSize:        1000,
			CacheTTL:            time.Hour,
			RetryMaxAttempts:    3,
			RetryInitialDelay:   time.Second,
			RetryMaxDelay:       10 * time.Second,
		}
	}

	t.Run("valid config passes validation", func(t *testing.T) {
		cfg := validConfig()
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("valid config with all providers", func(t *testing.T) {
		for _, provider := range []string{ProviderAnthropic, ProviderOpenAI, ProviderGoogle} {
			cfg := validConfig()
			cfg.Provider = provider
			err := cfg.Validate()
			assert.NoError(t, err, "provider %s should be valid", provider)
		}
	})

	t.Run("invalid provider", func(t *testing.T) {
		cfg := validConfig()
		cfg.Provider = "invalid-provider"
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "provider")
	})

	t.Run("temperature too low", func(t *testing.T) {
		cfg := validConfig()
		cfg.Temperature = -0.1
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "temperature")
	})

	t.Run("temperature too high", func(t *testing.T) {
		cfg := validConfig()
		cfg.Temperature = 2.1
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "temperature")
	})

	t.Run("temperature zero is valid", func(t *testing.T) {
		cfg := validConfig()
		cfg.Temperature = 0.0
		err := cfg.Validate()
		assert.NoError(t, err, "temperature 0.0 should be valid")
	})

	t.Run("temperature 2.0 is valid", func(t *testing.T) {
		cfg := validConfig()
		cfg.Temperature = 2.0
		err := cfg.Validate()
		assert.NoError(t, err, "temperature 2.0 should be valid")
	})

	t.Run("max tokens zero", func(t *testing.T) {
		cfg := validConfig()
		cfg.MaxTokens = 0
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_tokens")
	})

	t.Run("max tokens negative", func(t *testing.T) {
		cfg := validConfig()
		cfg.MaxTokens = -100
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max_tokens")
	})

	t.Run("timeout zero", func(t *testing.T) {
		cfg := validConfig()
		cfg.Timeout = 0
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("timeout negative", func(t *testing.T) {
		cfg := validConfig()
		cfg.Timeout = -time.Second
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("diff max chars zero", func(t *testing.T) {
		cfg := validConfig()
		cfg.DiffMaxChars = 0
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "diff_max_chars")
	})

	t.Run("diff max lines per file zero", func(t *testing.T) {
		cfg := validConfig()
		cfg.DiffMaxLinesPerFile = 0
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "diff_max_lines_per_file")
	})

	t.Run("cache max size zero when cache enabled", func(t *testing.T) {
		cfg := validConfig()
		cfg.CacheEnabled = true
		cfg.CacheMaxSize = 0
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache_max_size")
	})

	t.Run("cache max size zero when cache disabled is ok", func(t *testing.T) {
		cfg := validConfig()
		cfg.CacheEnabled = false
		cfg.CacheMaxSize = 0
		err := cfg.Validate()
		assert.NoError(t, err, "cache_max_size=0 should be valid when cache is disabled")
	})

	t.Run("cache ttl zero when cache enabled", func(t *testing.T) {
		cfg := validConfig()
		cfg.CacheEnabled = true
		cfg.CacheTTL = 0
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache_ttl")
	})

	t.Run("cache ttl zero when cache disabled is ok", func(t *testing.T) {
		cfg := validConfig()
		cfg.CacheEnabled = false
		cfg.CacheTTL = 0
		err := cfg.Validate()
		assert.NoError(t, err, "cache_ttl=0 should be valid when cache is disabled")
	})

	t.Run("retry max attempts zero", func(t *testing.T) {
		cfg := validConfig()
		cfg.RetryMaxAttempts = 0
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "retry_max_attempts")
	})

	t.Run("retry initial delay zero", func(t *testing.T) {
		cfg := validConfig()
		cfg.RetryInitialDelay = 0
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "retry_initial_delay")
	})

	t.Run("retry max delay less than initial delay", func(t *testing.T) {
		cfg := validConfig()
		cfg.RetryInitialDelay = 10 * time.Second
		cfg.RetryMaxDelay = 5 * time.Second
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "retry_max_delay")
	})

	t.Run("retry max delay equal to initial delay is ok", func(t *testing.T) {
		cfg := validConfig()
		cfg.RetryInitialDelay = 5 * time.Second
		cfg.RetryMaxDelay = 5 * time.Second
		err := cfg.Validate()
		assert.NoError(t, err)
	})
}

func TestSetConfigLogger(t *testing.T) {
	t.Run("logs warning when set", func(t *testing.T) {
		var logged string
		SetConfigLogger(func(msg string, _ ...interface{}) {
			logged = msg
		})
		defer SetConfigLogger(nil)

		logConfigWarning("test message")
		assert.Contains(t, logged, "test message")
	})

	t.Run("no panic when logger is nil", func(t *testing.T) {
		SetConfigLogger(nil)
		assert.NotPanics(t, func() {
			logConfigWarning("should not panic")
		})
	})
}
