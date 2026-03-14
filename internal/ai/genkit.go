package ai

import (
	"context"
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

// initAnthropicProvider initializes the Anthropic/Claude backend.
func initAnthropicProvider(ctx context.Context, cfg *Config) *genkit.Genkit {
	plugin := &anthropic.Anthropic{
		Opts: []option.RequestOption{
			option.WithAPIKey(cfg.APIKey),
		},
	}
	return genkit.Init(ctx,
		genkit.WithPlugins(plugin),
		genkit.WithDefaultModel(getModelPath(cfg)),
	)
}

// initOpenAIProvider initializes the OpenAI backend.
func initOpenAIProvider(ctx context.Context, cfg *Config) *genkit.Genkit {
	plugin := &openai.OpenAI{
		Opts: []option.RequestOption{
			option.WithAPIKey(cfg.APIKey),
		},
	}
	return genkit.Init(ctx,
		genkit.WithPlugins(plugin),
		genkit.WithDefaultModel(getModelPath(cfg)),
	)
}

// initGoogleProvider initializes the Google Gemini backend.
func initGoogleProvider(ctx context.Context, cfg *Config) *genkit.Genkit {
	plugin := &googlegenai.GoogleAI{
		APIKey: cfg.APIKey,
	}
	return genkit.Init(ctx,
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
