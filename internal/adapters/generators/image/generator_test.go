package image

import (
	"context"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/domain/episode"
)

func TestGenerateIncludesReferenceMetadata(t *testing.T) {
	t.Parallel()

	provider := fakeImageProvider{}
	generator := Generator{GeneratorID: "image-artist", Provider: provider, Seed: 42}
	output, err := generator.Generate(context.Background(), episode.Brief{
		WorldID:      "glass-kingdom",
		CharacterIDs: []string{"red-wanderer"},
		EventID:      "lost-artifact",
		Tone:         "mysterious",
		CanonRules:   []string{"Keep continuity"},
		VisualReferences: []episode.VisualReference{
			{AssetID: "hero-base", Usage: "character_reference", Description: "Red scarf silhouette."},
		},
		ContinuityReferences: []episode.ContinuityReference{
			{EpisodeID: "ep-1", Summary: "The wanderer crossed the ash bridge."},
		},
	}, episode.State{})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if output.ProviderRequest["visual_references"] == nil {
		t.Fatal("expected visual refs in provider request")
	}
	if output.ProviderRequest["continuity_references"] == nil {
		t.Fatal("expected continuity refs in provider request")
	}
}

type fakeImageProvider struct{}

func (fakeImageProvider) Name() string { return "fake-image" }

func (fakeImageProvider) GenerateImage(context.Context, contracts.ImageRequest) (contracts.ImageResponse, error) {
	return contracts.ImageResponse{AssetPath: "/tmp/out.png", Model: "fake-image-v1"}, nil
}
