package video

import (
	"context"
	"testing"

	providercontracts "loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/domain/episode"
)

func TestGeneratePrefersSelectedPromptImage(t *testing.T) {
	t.Parallel()

	provider := &fakeVideoProvider{}
	generator := Generator{GeneratorID: "video-artist", Provider: provider, Seed: 42}
	_, err := generator.Generate(context.Background(), episode.Brief{
		WorldID:      "glass-kingdom",
		CharacterIDs: []string{"red-wanderer"},
		EventID:      "lost-artifact",
		Tone:         "cinematic",
		VisualReferences: []episode.VisualReference{
			{AssetID: "hero-base", Path: "/tmp/hero.png", MediaType: "image", Usage: "video_prompt_image", ModelRole: "prompt_image"},
			{AssetID: "city", Path: "/tmp/city.png", MediaType: "image", Usage: "environment_reference", ModelRole: "asset"},
		},
		ContinuityReferences: []episode.ContinuityReference{
			{EpisodeID: "ep-1", Summary: "The wanderer crossed the ash bridge."},
		},
	}, episode.State{Metadata: map[string]any{"prompt_image": "/tmp/fallback.png"}})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if provider.request.PromptImage != "/tmp/hero.png" {
		t.Fatalf("expected selected asset prompt image, got %s", provider.request.PromptImage)
	}
	if len(provider.request.ReferenceImages) != 1 || provider.request.ReferenceImages[0].URI != "/tmp/city.png" {
		t.Fatalf("unexpected reference images: %#v", provider.request.ReferenceImages)
	}
}

type fakeVideoProvider struct {
	request providercontracts.VideoRequest
}

func (f *fakeVideoProvider) Name() string { return "fake-video" }

func (f *fakeVideoProvider) GenerateVideo(_ context.Context, req providercontracts.VideoRequest) (providercontracts.VideoResponse, error) {
	f.request = req
	return providercontracts.VideoResponse{AssetPath: "/tmp/out.mp4", Model: "fake-video-v1"}, nil
}
