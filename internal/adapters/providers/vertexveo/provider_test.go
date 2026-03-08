package vertexveo

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"name":"operations/test-op"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/operations/test-op":
			_, _ = w.Write([]byte(`{"done":true,"response":{"videos":[{"gcsUri":"gs://bucket/video.mp4"}]}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/storage/v1/b/bucket/o/video.mp4":
			w.Header().Set("Content-Type", "video/mp4")
			_, _ = w.Write([]byte("video-data"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GOOGLE_CLOUD_PROJECT", "project-1")
	t.Setenv("GOOGLE_CLOUD_ACCESS_TOKEN", "token-1")
	provider := Provider{
		Config: config.ProviderDriver{
			Driver:       "vertex_veo",
			Model:        "veo-3.1-fast-generate-001",
			ProjectIDEnv: "GOOGLE_CLOUD_PROJECT",
			Location:     "us-central1",
			BucketURI:    "gs://bucket/out",
			BaseURL:      server.URL + "/v1",
			PollInterval: "1ms",
			Timeout:      "2s",
			Options:      map[string]any{"gcs_base_url": server.URL},
		},
		HTTP: server.Client(),
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
