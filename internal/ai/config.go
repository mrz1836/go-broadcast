package ai

import (
	"fmt"
	"strconv"
	"time"

	"github.com/mrz1836/go-broadcast/internal/env"
)

// configLogger is an optional package-level logger for configuration warnings.
// Set via SetConfigLogger() to enable logging of configuration parsing issues.
//
//nolint:gochecknoglobals // Intentional - allows optional logging without requiring dependency injection
var configLogger func(msg string, args ...interface{})

// SetConfigLogger sets an optional logger for configuration warnings.
// The logger function receives a format string and arguments.
// Pass nil to disable logging (default).
func SetConfigLogger(logger func(msg string, args ...interface{})) {
	configLogger = logger
}

// logConfigWarning logs a configuration warning if a logger is set.
func logConfigWarning(format string, args ...interface{}) {
	if configLogger != nil {
		configLogger(format, args...)
	}
}

// Config holds AI provider configuration loaded from environment variables.
type Config struct {
	// Enabled is the master switch that enables AI infrastructure.
	Enabled bool

	// PREnabled enables AI for PR body generation (defaults to Enabled value).
	PREnabled bool

	// CommitEnabled enables AI for commit message generation (defaults to Enabled value).
	CommitEnabled bool

	// Provider specifies which AI provider to use: "anthropic", "openai", or "google".
	Provider string

	// APIKey is the API key for the selected provider.
	APIKey string

	// Model specifies which model to use (provider-specific defaults apply if empty).
	Model string

	// MaxTokens limits the response length.
	MaxTokens int

	// Timeout is the maximum time to wait for AI generation.
	Timeout time.Duration

	// Temperature controls randomness (0.0-1.0).
	Temperature float64

	// DiffMaxChars is the maximum characters in diff context (default: 4000).
	DiffMaxChars int

	// DiffMaxLinesPerFile is the maximum lines per file in diff (default: 50).
	DiffMaxLinesPerFile int

	// CacheEnabled enables diff-based response caching (default: true).
	CacheEnabled bool

	// CacheTTL is the cache time-to-live (default: 1 hour).
	CacheTTL time.Duration

	// CacheMaxSize is the maximum number of cached entries (default: 1000).
	CacheMaxSize int

	// RetryMaxAttempts is the maximum number of retry attempts (default: 3).
	RetryMaxAttempts int

	// RetryInitialDelay is the initial delay between retries (default: 1s).
	RetryInitialDelay time.Duration

	// RetryMaxDelay is the maximum delay between retries (default: 10s).
	RetryMaxDelay time.Duration
}

// LoadConfig reads AI configuration from environment variables.
// All settings have sensible defaults and the feature is disabled by default.
func LoadConfig() *Config {
	enabled := env.GetEnvWithFallback("GO_BROADCAST_AI_ENABLED", "false") == "true"

	cfg := &Config{
		Enabled:       enabled,
		PREnabled:     parseBoolWithDefault("GO_BROADCAST_AI_PR_ENABLED", false),
		CommitEnabled: parseBoolWithDefault("GO_BROADCAST_AI_COMMIT_ENABLED", false),
		Provider:      env.GetEnvWithFallback("GO_BROADCAST_AI_PROVIDER", ProviderAnthropic),
		APIKey:        env.GetEnvWithFallback("GO_BROADCAST_AI_API_KEY", ""),
		Model:         env.GetEnvWithFallback("GO_BROADCAST_AI_MODEL", ""),
		MaxTokens:     parseIntWithDefault("GO_BROADCAST_AI_MAX_TOKENS", 2000),
		Timeout:       parseDurationSecondsWithDefault("GO_BROADCAST_AI_TIMEOUT", 30*time.Second),
		Temperature:   parseFloatWithDefault("GO_BROADCAST_AI_TEMPERATURE", 0.3),

		// Diff truncation
		DiffMaxChars:        parseIntWithDefault("GO_BROADCAST_AI_DIFF_MAX_CHARS", 4000),
		DiffMaxLinesPerFile: parseIntWithDefault("GO_BROADCAST_AI_DIFF_MAX_LINES_PER_FILE", 50),

		// Cache (enabled by default for cost savings)
		CacheEnabled: env.GetEnvWithFallback("GO_BROADCAST_AI_CACHE_ENABLED", "true") == "true",
		CacheTTL:     parseDurationSecondsWithDefault("GO_BROADCAST_AI_CACHE_TTL", 1*time.Hour),
		CacheMaxSize: parseIntWithDefault("GO_BROADCAST_AI_CACHE_MAX_SIZE", 1000),

		// Retry
		RetryMaxAttempts:  parseIntWithDefault("GO_BROADCAST_AI_RETRY_MAX_ATTEMPTS", 3),
		RetryInitialDelay: parseDurationSecondsWithDefault("GO_BROADCAST_AI_RETRY_INITIAL_DELAY", 1*time.Second),
		RetryMaxDelay:     parseDurationSecondsWithDefault("GO_BROADCAST_AI_RETRY_MAX_DELAY", 10*time.Second),
	}

	// Fall back to provider-specific API key env vars if GO_BROADCAST_AI_API_KEY is not set
	if cfg.APIKey == "" {
		switch cfg.Provider {
		case ProviderAnthropic:
			cfg.APIKey = env.GetEnvWithFallback("ANTHROPIC_API_KEY", "")
		case ProviderOpenAI:
			cfg.APIKey = env.GetEnvWithFallback("OPENAI_API_KEY", "")
		case ProviderGoogle:
			cfg.APIKey = env.GetEnvWithFallback("GEMINI_API_KEY", "")
		}
	}

	// Apply default model if not specified
	if cfg.Model == "" {
		cfg.Model = GetDefaultModel(cfg.Provider)
	}

	return cfg
}

// IsEnabled returns true if AI is enabled and properly configured.
func (c *Config) IsEnabled() bool {
	return c.Enabled && c.APIKey != ""
}

// IsPREnabled returns true if AI PR body generation is enabled.
func (c *Config) IsPREnabled() bool {
	return c.IsEnabled() && c.PREnabled
}

// IsCommitEnabled returns true if AI commit message generation is enabled.
func (c *Config) IsCommitEnabled() bool {
	return c.IsEnabled() && c.CommitEnabled
}

// Validate checks that all configuration values are within valid bounds.
// Returns nil if configuration is valid, or an error describing the first invalid value found.
func (c *Config) Validate() error {
	// Validate provider
	switch c.Provider {
	case ProviderAnthropic, ProviderOpenAI, ProviderGoogle:
		// Valid provider
	default:
		return ConfigError("provider", fmt.Sprintf("unsupported provider %q", c.Provider))
	}

	// Validate temperature (0.0 to 2.0 is valid for most providers)
	if c.Temperature < 0.0 || c.Temperature > 2.0 {
		return ConfigError("temperature", fmt.Sprintf("%v must be between 0.0 and 2.0", c.Temperature))
	}

	// Validate MaxTokens (must be positive)
	if c.MaxTokens <= 0 {
		return ConfigError("max_tokens", fmt.Sprintf("%d must be positive", c.MaxTokens))
	}

	// Validate Timeout (must be positive)
	if c.Timeout <= 0 {
		return ConfigError("timeout", fmt.Sprintf("%v must be positive", c.Timeout))
	}

	// Validate DiffMaxChars (must be positive)
	if c.DiffMaxChars <= 0 {
		return ConfigError("diff_max_chars", fmt.Sprintf("%d must be positive", c.DiffMaxChars))
	}

	// Validate DiffMaxLinesPerFile (must be positive)
	if c.DiffMaxLinesPerFile <= 0 {
		return ConfigError("diff_max_lines_per_file", fmt.Sprintf("%d must be positive", c.DiffMaxLinesPerFile))
	}

	// Validate CacheMaxSize (must be positive if cache is enabled)
	if c.CacheEnabled && c.CacheMaxSize <= 0 {
		return ConfigError("cache_max_size", fmt.Sprintf("%d must be positive when cache is enabled", c.CacheMaxSize))
	}

	// Validate CacheTTL (must be positive if cache is enabled)
	if c.CacheEnabled && c.CacheTTL <= 0 {
		return ConfigError("cache_ttl", fmt.Sprintf("%v must be positive when cache is enabled", c.CacheTTL))
	}

	// Validate RetryMaxAttempts (must be positive)
	if c.RetryMaxAttempts <= 0 {
		return ConfigError("retry_max_attempts", fmt.Sprintf("%d must be positive", c.RetryMaxAttempts))
	}

	// Validate RetryInitialDelay (must be positive)
	if c.RetryInitialDelay <= 0 {
		return ConfigError("retry_initial_delay", fmt.Sprintf("%v must be positive", c.RetryInitialDelay))
	}

	// Validate RetryMaxDelay (must be >= RetryInitialDelay)
	if c.RetryMaxDelay < c.RetryInitialDelay {
		return ConfigError("retry_max_delay", fmt.Sprintf("%v must be >= retry_initial_delay (%v)", c.RetryMaxDelay, c.RetryInitialDelay))
	}

	return nil
}

// parseIntWithDefault parses an environment variable as int with a default value.
// Logs a warning if the value is set but cannot be parsed.
func parseIntWithDefault(key string, defaultValue int) int {
	value := env.GetEnvWithFallback(key, "")
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		logConfigWarning("AI config: invalid integer for %s=%q, using default %d", key, value, defaultValue)
		return defaultValue
	}
	return parsed
}

// parseFloatWithDefault parses an environment variable as float64 with a default value.
// Logs a warning if the value is set but cannot be parsed.
func parseFloatWithDefault(key string, defaultValue float64) float64 {
	value := env.GetEnvWithFallback(key, "")
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		logConfigWarning("AI config: invalid float for %s=%q, using default %v", key, value, defaultValue)
		return defaultValue
	}
	return parsed
}

// parseDurationSecondsWithDefault parses an environment variable as seconds to Duration.
// Logs a warning if the value is set but cannot be parsed.
func parseDurationSecondsWithDefault(key string, defaultValue time.Duration) time.Duration {
	value := env.GetEnvWithFallback(key, "")
	if value == "" {
		return defaultValue
	}
	seconds, err := strconv.Atoi(value)
	if err != nil {
		logConfigWarning("AI config: invalid duration for %s=%q, using default %v", key, value, defaultValue)
		return defaultValue
	}
	return time.Duration(seconds) * time.Second
}

// parseBoolWithDefault parses an environment variable as bool with a default value.
// Returns true only if value is exactly "true". Logs a warning for unexpected non-boolean values.
func parseBoolWithDefault(key string, defaultValue bool) bool {
	value := env.GetEnvWithFallback(key, "")
	if value == "" {
		return defaultValue
	}
	// Log warning for non-standard boolean values (anything other than "true" or "false")
	if value != "true" && value != "false" {
		logConfigWarning("AI config: non-standard boolean for %s=%q (expected 'true' or 'false'), treating as false", key, value)
	}
	return value == "true"
}
