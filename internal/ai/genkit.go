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
func (p *GenkitProvider) GenerateText(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if p.gk == nil {
		return nil, ErrProviderNotConfigured
	}

	start := time.Now()

	// Build generation options - config is provider-specific, so we only use prompt
	// The compat_oai plugins don't accept GenerationCommonConfig
	opts := []genkitai.GenerateOption{
		genkitai.WithPrompt(req.Prompt),
	}

	// Generate response
	resp, err := genkit.Generate(ctx, p.gk, opts...)
	if err != nil {
		return nil, GenerationError(p.provider, "generate text", err)
	}

	content := resp.Text()
	if content == "" {
		return nil, ErrEmptyResponse
	}

	return &GenerateResponse{
		Content:      content,
		TokensUsed:   getTokenCount(resp),
		FinishReason: getFinishReason(resp),
		Duration:     time.Since(start),
	}, nil
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
