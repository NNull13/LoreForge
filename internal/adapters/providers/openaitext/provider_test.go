package openaitext

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/config"
	"loreforge/internal/domain/episode"
)

func TestGenerateTextStructured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"output_text":"{\"parts\":[\"tweet one\",\"tweet two\"]}","output":[{"status":"completed","content":[{"type":"output_text","text":"{\"parts\":[\"tweet one\",\"tweet two\"]}"}]}]}`))
	}))
	defer server.Close()

	if err := os.Setenv("OPENAI_API_KEY", "test-token"); err != nil {
		t.Fatalf("Setenv failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("OPENAI_API_KEY") })

	provider := Provider{
		Config: config.ProviderDriver{
			Driver:    "openai_text",
			Model:     "gpt-5-mini",
			APIKeyEnv: "OPENAI_API_KEY",
			BaseURL:   server.URL,
		},
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
