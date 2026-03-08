package verteximagen

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/config"
)

func TestGenerateImageDecodesPrediction(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte("imgdata"))
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.Contains(r.URL.Path, ":predict") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"predictions":[{"bytesBase64Encoded":"` + raw + `","mimeType":"image/png","prompt":"enhanced"}]}`)),
		}, nil
	})}

	t.Setenv("GOOGLE_CLOUD_PROJECT", "project-1")
	t.Setenv("GOOGLE_CLOUD_ACCESS_TOKEN", "token-1")
	provider := Provider{
		Config: config.ProviderDriver{
			Driver:       "vertex_imagen",
			Model:        "imagen-4.0-fast-generate-001",
			ProjectIDEnv: "GOOGLE_CLOUD_PROJECT",
			Location:     "us-central1",
			BaseURL:      "https://vertex.test/v1",
		},
		HTTP: client,
	}

	resp, err := provider.GenerateImage(context.Background(), contracts.ImageRequest{Prompt: "forest"})
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
