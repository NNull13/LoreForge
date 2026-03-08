package runwayvideo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	providercontracts "loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/config"
)

func TestGenerateVideoRequiresPromptImage(t *testing.T) {
	t.Parallel()

	provider := Provider{Config: config.ProviderDriver{Driver: "runway_gen4", Model: "gen4_turbo", APIKeyEnv: "RUNWAY_API_KEY"}}
	if _, err := provider.GenerateVideo(context.Background(), providercontracts.VideoRequest{Prompt: "storm"}); err == nil {
		t.Fatal("expected error when prompt image is missing")
	}
}

func TestGenerateVideoPollsTaskAndDownloadsOutput(t *testing.T) {
	var polls int32
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/image_to_video":
			_, _ = w.Write([]byte(`{"id":"task-1"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/task-1":
			if atomic.AddInt32(&polls, 1) == 1 {
				_, _ = w.Write([]byte(`{"status":"PENDING"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"SUCCEEDED","output":["` + baseURL + `/download/video.mp4"]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/download/video.mp4":
			w.Header().Set("Content-Type", "video/mp4")
			_, _ = w.Write([]byte("video"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	t.Setenv("RUNWAY_API_KEY", "runway-token")
	provider := Provider{
		Config: config.ProviderDriver{
			Driver:       "runway_gen4",
			Model:        "gen4_turbo",
			APIKeyEnv:    "RUNWAY_API_KEY",
			BaseURL:      server.URL,
			Version:      "2024-11-06",
			PollInterval: "1ms",
			Timeout:      "7s",
		},
		HTTP: server.Client(),
	}

	resp, err := provider.GenerateVideo(context.Background(), providercontracts.VideoRequest{
		Prompt:      "storm over ruins",
		PromptImage: "data:image/png;base64,AAAA",
	})
	if err != nil {
		t.Fatalf("GenerateVideo returned error: %v", err)
	}
	if resp.AssetPath == "" {
		t.Fatal("expected asset path")
	}
	if resp.JobID != "task-1" {
		t.Fatalf("unexpected job id: %s", resp.JobID)
	}
}
