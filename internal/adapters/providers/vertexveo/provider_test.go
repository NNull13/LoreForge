package vertexveo

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/config"
)

func TestGenerateVideoRejectsStyleReferenceForVeo31(t *testing.T) {
	provider := Provider{
		Config: config.ProviderDriver{
			Driver:       "vertex_veo",
			Model:        "veo-3.1-fast-generate-001",
			ProjectIDEnv: "GOOGLE_CLOUD_PROJECT",
			Location:     "us-central1",
			BucketURI:    "gs://bucket/out",
			PollInterval: "10ms",
			Timeout:      "1s",
		},
	}
	t.Setenv("GOOGLE_CLOUD_PROJECT", "project-1")
	t.Setenv("GOOGLE_CLOUD_ACCESS_TOKEN", "token-1")
	_, err := provider.GenerateVideo(context.Background(), contracts.VideoRequest{
		Prompt: "city",
		ReferenceImages: []contracts.ReferenceImage{
			{URI: "gs://bucket/ref.png", ReferenceType: "style"},
		},
	})
	if err == nil {
		t.Fatal("expected style reference error")
	}
}

func TestGenerateVideoPollsOperationAndDownloadsFromGCSAPI(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodPost:
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"name":"operations/test-op"}`)),
			}, nil
		case r.Method == http.MethodGet && r.URL.Path == "/v1/operations/test-op":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"done":true,"response":{"videos":[{"gcsUri":"gs://bucket/video.mp4"}]}}`)),
			}, nil
		case r.Method == http.MethodGet && r.URL.Host == "storage.test" && r.URL.Path == "/storage/v1/b/bucket/o/video.mp4":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"video/mp4"}},
				Body:       io.NopCloser(strings.NewReader("video-data")),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(strings.NewReader("not found")),
			}, nil
		}
	})}

	t.Setenv("GOOGLE_CLOUD_PROJECT", "project-1")
	t.Setenv("GOOGLE_CLOUD_ACCESS_TOKEN", "token-1")
	provider := Provider{
		Config: config.ProviderDriver{
			Driver:       "vertex_veo",
			Model:        "veo-3.1-fast-generate-001",
			ProjectIDEnv: "GOOGLE_CLOUD_PROJECT",
			Location:     "us-central1",
			BucketURI:    "gs://bucket/out",
			BaseURL:      "https://vertex.test/v1",
			PollInterval: "1ms",
			Timeout:      "2s",
			Options:      map[string]any{"gcs_base_url": "https://storage.test"},
		},
		HTTP: client,
	}

	resp, err := provider.GenerateVideo(context.Background(), contracts.VideoRequest{Prompt: "storm"})
	if err != nil {
		t.Fatalf("GenerateVideo returned error: %v", err)
	}
	if resp.AssetPath == "" {
		t.Fatal("expected asset path")
	}
	if resp.JobID != "operations/test-op" {
		t.Fatalf("unexpected job id: %s", resp.JobID)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
