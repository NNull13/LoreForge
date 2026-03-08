package openaitext

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/config"
	"loreforge/internal/domain/episode"
)

func TestGenerateTextStructured(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"output_text":"{\"parts\":[\"tweet one\",\"tweet two\"]}","output":[{"status":"completed","content":[{"type":"output_text","text":"{\"parts\":[\"tweet one\",\"tweet two\"]}"}]}]}`)),
		}, nil
	})}

	if err := os.Setenv("OPENAI_API_KEY", "test-token"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("OPENAI_API_KEY") })

	provider := Provider{
		Config: config.ProviderDriver{
			Driver:    "openai_text",
			Model:     "gpt-5-mini",
			APIKeyEnv: "OPENAI_API_KEY",
			BaseURL:   "https://api.openai.test/v1",
		},
		HTTP: client,
	}

	resp, err := provider.GenerateText(context.Background(), contracts.TextRequest{
		Format:       episode.OutputTypeTweetThread,
		SystemPrompt: "system",
		Prompt:       "prompt",
		JSONSchema:   map[string]any{"type": "object"},
	})
	if err != nil {
		t.Fatalf("GenerateText returned error: %v", err)
	}
	if len(resp.Parts) != 2 {
		t.Fatalf("unexpected parts: %#v", resp.Parts)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
