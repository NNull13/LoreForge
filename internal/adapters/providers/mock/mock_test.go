package mock

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/domain/episode"
)

func TestTextProviderFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		format episode.OutputType
		want   string
	}{
		{format: episode.OutputTypeTweetShort, want: "Red Wanderer crosses"},
		{format: episode.OutputTypeTweetThread, want: "Red Wanderer reaches"},
		{format: episode.OutputTypePoem, want: "Ash Gate"},
		{format: episode.OutputTypeSongLyrics, want: "Lantern Chorus"},
		{format: episode.OutputTypeScreenplaySeries, want: "Gate of Glass"},
		{format: episode.OutputTypeLongStory, want: "The Oath Returns"},
		{format: episode.OutputTypeShortStory, want: "Ash Garden"},
	}

	provider := TextProvider{}
	if provider.Name() != "mock-text" {
		t.Fatalf("unexpected provider name: %s", provider.Name())
	}
	for _, tt := range cases {
		resp, err := provider.GenerateText(context.Background(), contracts.TextRequest{Format: tt.format})
		if err != nil {
			t.Fatalf("GenerateText(%s) returned error: %v", tt.format, err)
		}
		joined := resp.Title + " " + resp.Content + " " + strings.Join(resp.Parts, " ")
		if !strings.Contains(joined, tt.want) {
			t.Fatalf("GenerateText(%s) = %#v, want substring %q", tt.format, resp, tt.want)
		}
	}
}

func TestImageAndVideoProvidersWriteAssets(t *testing.T) {
	temp := t.TempDir()
	t.Chdir(temp)

	image := ImageProvider{}
	if image.Name() != "mock-image" {
		t.Fatalf("unexpected image provider name: %s", image.Name())
	}
	imageResp, err := image.GenerateImage(context.Background(), contracts.ImageRequest{Prompt: "embers"})
	if err != nil {
		t.Fatalf("GenerateImage returned error: %v", err)
	}
	if imageResp.MIMEType != "image/png" {
		t.Fatalf("unexpected image mime type: %s", imageResp.MIMEType)
	}
	imageBytes, err := os.ReadFile(imageResp.AssetPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(imageBytes), "embers") || filepath.Ext(imageResp.AssetPath) != ".png" {
		t.Fatalf("unexpected image asset: %s %q", imageResp.AssetPath, imageBytes)
	}

	video := VideoProvider{}
	if video.Name() != "mock-video" {
		t.Fatalf("unexpected video provider name: %s", video.Name())
	}
	videoResp, err := video.GenerateVideo(context.Background(), contracts.VideoRequest{Prompt: "lantern"})
	if err != nil {
		t.Fatalf("GenerateVideo returned error: %v", err)
	}
	videoBytes, err := os.ReadFile(videoResp.AssetPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(videoBytes), "lantern") || filepath.Ext(videoResp.AssetPath) != ".mp4" {
		t.Fatalf("unexpected video asset: %s %q", videoResp.AssetPath, videoBytes)
	}
	if videoResp.Metadata["driver"] != "mock" {
		t.Fatalf("unexpected video metadata: %#v", videoResp.Metadata)
	}
}
