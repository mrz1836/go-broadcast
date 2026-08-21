package ai

import (
	"context"
	"net/http"
	"time"

	genkitai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/anthropic"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/openai/openai-go/option"
	"github.com/sirupsen/logrus"
)

// GenkitProvider implements Provider using the Genkit unified interface.
// Supports: Anthropic Claude, OpenAI GPT, Google Gemini.
// Thread-safe for concurrent use.
type GenkitProvider struct {
	gk       *genkit.Genkit
	config   *Config
	provider string
	logger   *logrus.Entry
}

// NewGenkitProvider creates a provider based on configuration.
// It initializes the appropriate backend (anthropic, openai, or google).
func NewGenkitProvider(ctx context.Context, cfg *Config, logger *logrus.Entry) (*GenkitProvider, error) {
	if cfg.APIKey == "" {
		return nil, ErrAPIKeyMissing
	}

	var gk *genkit.Genkit

	switch cfg.Provider {
	case ProviderAnthropic:
		gk = initAnthropicProvider(ctx, cfg)
	case ProviderOpenAI:
		gk = initOpenAIProvider(ctx, cfg)
	case ProviderGoogle:
		gk = initGoogleProvider(ctx, cfg)
	default:
		return nil, ErrUnsupportedProvider
	}

	return &GenkitProvider{
		gk:       gk,
		config:   cfg,
		provider: cfg.Provider,
		logger:   logger,
	}, nil
}

// DefaultRequestTimeout is the fallback per-attempt request timeout when the
// config does not specify one.
const DefaultRequestTimeout = 30 * time.Second

// perRequestTimeout returns the timeout applied to a single request against any
// provider endpoint. It is deliberately shorter than the overall generation
// budget (Config.Timeout) so a stalled connection fails fast and leaves room for
// a retry within the same deadline, instead of the first hung request consuming
// the entire budget and forcing a silent fallback. Two-thirds of the budget stays
// comfortably above normal generation latency while guaranteeing at least one
// retry can fit. Every provider derives its per-request bound from this value so
// resilience behavior is identical across Anthropic, OpenAI, and Google.
func perRequestTimeout(cfg *Config) time.Duration {
	base := cfg.Timeout
	if base <= 0 {
		base = DefaultRequestTimeout
	}
	return base * 2 / 3
}

// compatRequestOpts returns the request options shared by the OpenAI-compatible
// providers (Anthropic, OpenAI). The per-attempt timeout bounds each HTTP request
// so transient network stalls cannot swallow the whole generation budget; the
// SDK's built-in retry (default two) then re-issues the request on connection
// errors, giving flaky networks a chance to self-heal before falling back.
func compatRequestOpts(cfg *Config) []option.RequestOption {
	return []option.RequestOption{
		option.WithRequestTimeout(perRequestTimeout(cfg)),
	}
}

// perRequestHTTPClient returns an HTTP client whose timeout bounds a single
// request to a Google GenAI endpoint. The googlegenai plugin exposes no
// per-request timeout option like the OpenAI-compatible plugins, so the bound is
// applied at the transport layer instead - giving Google the same fail-fast
// behavior as the other providers. Supplying an HTTPClient opts out of the
// plugin's default OpenTelemetry-instrumented transport; text generation does not
// depend on that tracing.
func perRequestHTTPClient(cfg *Config) *http.Client {
	return &http.Client{Timeout: perRequestTimeout(cfg)}
}

// initAnthropicProvider initializes the Anthropic/Claude backend.
//
// The API key is set on the plugin's APIKey field (not just via Opts) so BOTH
// surfaces the plugin speaks authenticate: the OpenAI-compatible chat endpoint
// (bearer token) and the native models-list endpoint (x-api-key header). Passing
// the key only through Opts leaves model listing unauthenticated.
func initAnthropicProvider(ctx context.Context, cfg *Config) *genkit.Genkit {
	plugin := &anthropic.Anthropic{
		APIKey: cfg.APIKey,
		Opts:   compatRequestOpts(cfg),
	}
	return genkit.Init(
		ctx,
		genkit.WithPlugins(plugin),
		genkit.WithDefaultModel(getModelPath(cfg)),
	)
}

// initOpenAIProvider initializes the OpenAI backend.
//
// The OpenAI plugin requires the key on its APIKey field (or the OPENAI_API_KEY
// env var) and panics during Init otherwise - it does not consult Opts for that
// check. Setting the field directly (as the Google plugin does) ensures the
// provider works with a config-supplied key without requiring OPENAI_API_KEY.
func initOpenAIProvider(ctx context.Context, cfg *Config) *genkit.Genkit {
	plugin := &openai.OpenAI{
		APIKey: cfg.APIKey,
		Opts:   compatRequestOpts(cfg),
	}
	return genkit.Init(
		ctx,
		genkit.WithPlugins(plugin),
		genkit.WithDefaultModel(getModelPath(cfg)),
	)
}

// initGoogleProvider initializes the Google Gemini backend.
//
// A per-request HTTP client timeout bounds each call so a stalled connection
// fails fast and go-broadcast's retry can recover within the generation budget -
// matching the fail-fast behavior of the OpenAI-compatible providers.
func initGoogleProvider(ctx context.Context, cfg *Config) *genkit.Genkit {
	plugin := &googlegenai.GoogleAI{
		APIKey:     cfg.APIKey,
		HTTPClient: perRequestHTTPClient(cfg),
	}
	return genkit.Init(
		ctx,
		genkit.WithPlugins(plugin),
		genkit.WithDefaultModel(getModelPath(cfg)),
	)
}

// getModelPath returns the full model path for Genkit.
func getModelPath(cfg *Config) string {
	model := cfg.Model
	if model == "" {
		model = GetDefaultModel(cfg.Provider)
	}

	// Genkit uses provider prefix format
	switch cfg.Provider {
	case ProviderAnthropic:
		return "anthropic/" + model
	case ProviderOpenAI:
		return "openai/" + model
	case ProviderGoogle:
		return "googleai/" + model
	default:
		return model
	}
}

// Name returns the provider identifier.
func (p *GenkitProvider) Name() string {
	return p.provider
}

// GenerateText generates text based on the given prompt.
//
// IMPORTANT: Due to limitations in the Genkit compat_oai plugins, the following
// GenerateRequest fields are NOT used by this implementation:
//   - MaxTokens: Ignored - uses model defaults
//   - Temperature: Ignored - uses model defaults
//
// Only req.Prompt is passed to the underlying Genkit provider.
// The model defaults are configured at provider initialization time via getModelPath().
// Future Genkit versions may support per-request configuration.
func (p *GenkitProvider) GenerateText(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if p.gk == nil {
		return nil, ErrProviderNotConfigured
	}

	start := time.Now()

	// Build generation options.
	// The compat_oai plugins (anthropic, openai) don't accept GenerationCommonConfig,
	// so we cannot pass MaxTokens or Temperature per-request. These fields in GenerateRequest
	// are provided for interface consistency and potential future use with native plugins.
	opts := []genkitai.GenerateOption{
		genkitai.WithPrompt(req.Prompt),
	}

	// Use a channel to get the result so we can respect context cancellation
	// even if the underlying Genkit SDK doesn't properly handle it.
	type generateResult struct {
		resp *genkitai.ModelResponse
		err  error
	}
	resultCh := make(chan generateResult, 1)

	go func() {
		resp, err := genkit.Generate(ctx, p.gk, opts...)
		resultCh <- generateResult{resp, err}
	}()

	// Wait for either result or context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-resultCh:
		if r.err != nil {
			return nil, GenerationError(p.provider, "generate text", r.err)
		}

		content := r.resp.Text()
		if content == "" {
			return nil, ErrEmptyResponse
		}

		return &GenerateResponse{
			Content:      content,
			TokensUsed:   getTokenCount(r.resp),
			FinishReason: getFinishReason(r.resp),
			Duration:     time.Since(start),
		}, nil
	}
}

// IsAvailable checks if the provider is properly configured and ready.
func (p *GenkitProvider) IsAvailable() bool {
	return p.gk != nil && p.config.APIKey != ""
}

// getTokenCount extracts token count from response if available.
func getTokenCount(resp *genkitai.ModelResponse) int {
	if resp == nil || resp.Usage == nil {
		return 0
	}
	return resp.Usage.TotalTokens
}

// getFinishReason extracts finish reason from response if available.
func getFinishReason(resp *genkitai.ModelResponse) string {
	if resp == nil {
		return ""
	}
	return string(resp.FinishReason)
}

// Close releases resources held by the provider.
// Should be called when the provider is no longer needed to prevent resource leaks.
func (p *GenkitProvider) Close() error {
	// Genkit doesn't expose a cleanup method, but we clear the reference
	// to allow garbage collection and prevent further use.
	p.gk = nil
	return nil
}
