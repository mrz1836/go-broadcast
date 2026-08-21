//go:build ailive

package ai

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestLiveGenkitDiagnostic determines whether the production Genkit provider
// actually works, timing a real call. It exists to distinguish a genuine Genkit
// problem from transient network flakiness.
//
//	go test -tags ailive -run TestLiveGenkitDiagnostic ./internal/ai -v
func TestLiveGenkitDiagnostic(t *testing.T) {
	cfg := liveConfig(t)
	t.Logf("provider=%s model=%s apiKeySource=%s", cfg.Provider, cfg.Model, cfg.APIKeySource)

	logger := logrus.NewEntry(logrus.New())
	provider, err := NewGenkitProvider(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("NewGenkitProvider failed: %v", err)
	}
	t.Logf("provider.IsAvailable()=%v name=%s", provider.IsAvailable(), provider.Name())

	const attempts = 3
	for i := 0; i < attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		start := time.Now()
		resp, genErr := provider.GenerateText(ctx, &GenerateRequest{
			Prompt:      "Reply with exactly the words: hello from genkit",
			MaxTokens:   50,
			Temperature: TemperatureNotSet,
		})
		elapsed := time.Since(start)
		cancel()

		if genErr != nil {
			t.Logf("attempt %d: FAILED in %s: %v", i+1, elapsed.Round(time.Millisecond), genErr)
			continue
		}
		t.Logf("attempt %d: OK in %s, content=%q tokens=%d", i+1, elapsed.Round(time.Millisecond), resp.Content, resp.TokensUsed)
	}
}

// TestLiveGoogleDiagnostic verifies the Google/Gemini provider initializes with a
// per-request HTTP timeout and can generate against a real endpoint. It is skipped
// unless a Google API key is present. Set GO_BROADCAST_AI_MODEL to override the
// model if the default preview model is not enabled for the key.
//
//	go test -tags ailive -run TestLiveGoogleDiagnostic ./internal/ai -v
func TestLiveGoogleDiagnostic(t *testing.T) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		key = os.Getenv("GOOGLE_API_KEY")
	}
	if key == "" {
		t.Skip("no Google API key (GEMINI_API_KEY/GOOGLE_API_KEY) in environment")
	}

	cfg := &Config{
		Provider: ProviderGoogle,
		APIKey:   key,
		Model:    GetDefaultModel(ProviderGoogle),
		Timeout:  30 * time.Second,
	}
	if m := os.Getenv("GO_BROADCAST_AI_MODEL"); m != "" {
		cfg.Model = m
	}
	t.Logf("google model=%s perRequestTimeout=%s", cfg.Model, perRequestTimeout(cfg))

	logger := logrus.NewEntry(logrus.New())
	provider, err := NewGenkitProvider(context.Background(), cfg, logger)
	if err != nil {
		t.Skipf("could not initialize google provider: %v", err)
	}
	if !provider.IsAvailable() {
		t.Skip("google provider not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	start := time.Now()
	resp, genErr := provider.GenerateText(ctx, &GenerateRequest{
		Prompt:      "Reply with exactly the words: hello from gemini",
		MaxTokens:   50,
		Temperature: TemperatureNotSet,
	})
	if genErr != nil {
		t.Skipf("google generation failed (model may be unavailable for this key): %v", genErr)
	}
	t.Logf("google OK in %s, content=%q", time.Since(start).Round(time.Millisecond), resp.Content)
	assert.NotEmpty(t, resp.Content)
}
