package lmstudio_text

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/config"
	"loreforge/internal/domain/episode"
)

func TestGenerateTextStructuredChat(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"choices":[{"finish_reason":"stop","message":{"content":"{\"body\":\"Verse 1\\nRed Wanderer walks the ash road\\n\\nChorus\\nCarry the name into dawn\"}"}}]}`)),
		}, nil
	})}

	provider := Provider{
		Config: config.ProviderDriver{
			Driver:  "lmstudio_text",
			Model:   "qwen2.5-7b-instruct",
			BaseURL: "https://lmstudio.test/v1",
			Options: map[string]any{"endpoint_mode": "structured_chat"},
		},
		HTTP: client,
	}

	resp, err := provider.GenerateText(context.Background(), contracts.TextRequest{
		Format:       episode.OutputTypeSongLyrics,
		SystemPrompt: "system",
		Prompt:       "prompt",
		JSONSchema:   map[string]any{"type": "object"},
	})
	if err != nil {
		t.Fatalf("GenerateText returned error: %v", err)
	}
	if resp.Content == "" {
		t.Fatal("expected textual content")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
