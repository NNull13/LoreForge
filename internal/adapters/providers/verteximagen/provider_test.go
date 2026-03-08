package verteximagen

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	providercontracts "loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/config"
)

func TestGenerateImageDecodesPrediction(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte("imgdata"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"predictions":[{"bytesBase64Encoded":"` + raw + `","mimeType":"image/png","prompt":"enhanced"}]}`))
	}))
	defer server.Close()

	t.Setenv("GOOGLE_CLOUD_PROJECT", "project-1")
	t.Setenv("GOOGLE_CLOUD_ACCESS_TOKEN", "token-1")
	provider := Provider{
		Config: config.ProviderDriver{
			Driver:       "vertex_imagen",
			Model:        "imagen-4.0-fast-generate-001",
			ProjectIDEnv: "GOOGLE_CLOUD_PROJECT",
			Location:     "us-central1",
			BaseURL:      server.URL,
		},
		HTTP: server.Client(),
	}

	resp, err := provider.GenerateImage(context.Background(), providercontracts.ImageRequest{Prompt: "forest"})
	if err != nil {
		t.Fatalf("GenerateImage returned error: %v", err)
	}
	if resp.AssetPath == "" {
		t.Fatal("expected asset path")
	}
	if resp.RevisedPrompt != "enhanced" {
		t.Fatalf("unexpected prompt: %s", resp.RevisedPrompt)
	}
}
