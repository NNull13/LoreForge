package lmstudiotext

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/config"
	"loreforge/internal/domain/episode"
)

func TestGenerateTextStructuredChat(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"choices":[{"finish_reason":"stop","message":{"content":"{\"body\":\"Verse 1\\nRed Wanderer walks the ash road\\n\\nChorus\\nCarry the name into dawn\"}"}}]}`))
	}))
	defer server.Close()

	provider := Provider{
		Config: config.ProviderDriver{
			Driver:  "lmstudio_text",
			Model:   "qwen2.5-7b-instruct",
			BaseURL: server.URL,
			Options: map[string]any{"endpoint_mode": "structured_chat"},
		},
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
