package openaiimage

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/config"
)

func TestGenerateImageFromBase64(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte("pngdata"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"` + raw + `","revised_prompt":"revised"}]}`))
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	provider := Provider{
		Config: config.ProviderDriver{
			Driver:    "openai_image",
			Model:     "gpt-image-1.5",
			APIKeyEnv: "OPENAI_API_KEY",
			BaseURL:   server.URL,
			Options:   map[string]any{"response_format": "b64_json"},
		},
		HTTP: server.Client(),
	}

	resp, err := provider.GenerateImage(context.Background(), contracts.ImageRequest{Prompt: "castle"})
	if err != nil {
		t.Fatalf("GenerateImage returned error: %v", err)
	}
	if resp.AssetPath == "" {
		t.Fatal("expected asset path")
	}
	if resp.RevisedPrompt != "revised" {
		t.Fatalf("unexpected revised prompt: %s", resp.RevisedPrompt)
	}
}
