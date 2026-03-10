package openai_image

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

func TestGenerateImageFromBase64(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte("pngdata"))
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"b64_json":"` + raw + `","revised_prompt":"revised"}]}`)),
		}, nil
	})}

	t.Setenv("OPENAI_API_KEY", "test-key")
	provider := Provider{
		Config: config.ProviderDriver{
			Driver:    "openai_image",
			Model:     "gpt-image-1.5",
			APIKeyEnv: "OPENAI_API_KEY",
			BaseURL:   "https://api.openai.test/v1",
			Options:   map[string]any{"response_format": "b64_json"},
		},
		HTTP: client,
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
