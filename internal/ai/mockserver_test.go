package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// mockAIServer is a fake AI backend used to exercise the full GenkitProvider
// generation path without any live HTTP requests to a real provider.
//
// All three supported backends are redirectable to a local server purely via
// environment variables (no production code changes required):
//   - Anthropic: ANTHROPIC_BASE_URL  (honored by the genkit compat_oai plugin)
//   - OpenAI:    OPENAI_BASE_URL     (honored by openai-go's DefaultClientOptions)
//   - Google:    GOOGLE_GEMINI_BASE_URL (honored by the google.golang.org/genai SDK)
//
// Anthropic and OpenAI share the OpenAI-compatible /chat/completions wire
// format; Google uses the Gemini :generateContent format. The server serves
// whichever shape matches the request path.
type mockAIServer struct {
	*httptest.Server

	mu       sync.Mutex
	response string   // text the mock "generates" for every request
	requests int      // number of generation requests received
	bodies   []string // captured request bodies, for assertions
}

// newMockAIServer starts a mock AI backend that returns the given response text
// for every generation request. It is automatically shut down when the test ends.
func newMockAIServer(t *testing.T, response string) *mockAIServer {
	t.Helper()
	m := &mockAIServer{response: response}
	m.Server = httptest.NewServer(http.HandlerFunc(m.handle))
	t.Cleanup(m.Close)
	return m
}

func (m *mockAIServer) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	m.mu.Lock()
	m.requests++
	m.bodies = append(m.bodies, string(body))
	response := m.response
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	var payload any
	if strings.Contains(r.URL.Path, ":generateContent") {
		payload = geminiResponse(response) // Google (Gemini) format
	} else {
		payload = openAIResponse(response) // Anthropic + OpenAI (compat_oai) format
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// callCount returns how many generation requests the server has received.
func (m *mockAIServer) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requests
}

// openAIResponse builds a minimal OpenAI-compatible chat completion response.
func openAIResponse(content string) map[string]any {
	return map[string]any{
		"id":      "chatcmpl-mock",
		"object":  "chat.completion",
		"created": 0,
		"model":   "mock-model",
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": content},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     10,
			"completion_tokens": 5,
			"total_tokens":      15,
		},
	}
}

// geminiResponse builds a minimal Gemini generateContent response.
func geminiResponse(content string) map[string]any {
	return map[string]any{
		"candidates": []map[string]any{
			{
				"content":      map[string]any{"role": "model", "parts": []map[string]any{{"text": content}}},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     10,
			"candidatesTokenCount": 5,
			"totalTokenCount":      15,
		},
	}
}

// redirectProvider points the given provider's SDK base URL at the mock server
// via the environment variable each backend already honors. The variable is
// restored automatically when the test ends (t.Setenv).
func redirectProvider(t *testing.T, srv *mockAIServer, provider string) {
	t.Helper()
	switch provider {
	case ProviderAnthropic:
		t.Setenv("ANTHROPIC_BASE_URL", srv.URL)
	case ProviderOpenAI:
		t.Setenv("OPENAI_BASE_URL", srv.URL)
	case ProviderGoogle:
		t.Setenv("GOOGLE_GEMINI_BASE_URL", srv.URL)
	}
}

// newMockProvider builds a GenkitProvider whose backend is redirected to the
// given mock server. No live HTTP occurs during generation.
func newMockProvider(t *testing.T, srv *mockAIServer, cfg *Config) *GenkitProvider {
	t.Helper()
	redirectProvider(t, srv, cfg.Provider)
	provider, err := NewGenkitProvider(context.Background(), cfg, logrus.NewEntry(logrus.New()))
	require.NoError(t, err)
	return provider
}
