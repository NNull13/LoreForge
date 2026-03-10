package runway_video

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/config"
)

func TestGenerateVideoRequiresPromptImage(t *testing.T) {
	t.Parallel()

	provider := Provider{Config: config.ProviderDriver{Driver: "runway_gen4", Model: "gen4_turbo", APIKeyEnv: "RUNWAY_API_KEY"}}
	if _, err := provider.GenerateVideo(context.Background(), contracts.VideoRequest{Prompt: "storm"}); err == nil {
		t.Fatal("expected error when prompt image is missing")
	}
}

func TestGenerateVideoPollsTaskAndDownloadsOutput(t *testing.T) {
	var polls int32
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/image_to_video":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"id":"task-1"}`)),
			}, nil
		case r.Method == http.MethodGet && r.URL.Path == "/v1/tasks/task-1":
			if atomic.AddInt32(&polls, 1) == 1 {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"status":"PENDING"}`)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"status":"SUCCEEDED","output":["https://runway.test/download/video.mp4"]}`)),
			}, nil
		case r.Method == http.MethodGet && r.URL.Path == "/download/video.mp4":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"video/mp4"}},
				Body:       io.NopCloser(strings.NewReader("video")),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
				Body:       io.NopCloser(strings.NewReader("not found")),
			}, nil
		}
	})}

	t.Setenv("RUNWAY_API_KEY", "runway-token")
	provider := Provider{
		Config: config.ProviderDriver{
			Driver:       "runway_gen4",
			Model:        "gen4_turbo",
			APIKeyEnv:    "RUNWAY_API_KEY",
			BaseURL:      "https://runway.test/v1",
			Version:      "2024-11-06",
			PollInterval: "1ms",
			Timeout:      "7s",
		},
		HTTP: client,
	}

	resp, err := provider.GenerateVideo(context.Background(), contracts.VideoRequest{
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
