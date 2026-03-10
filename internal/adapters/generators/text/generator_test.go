package text

import (
	"context"
	"testing"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/application/text_settings"
	"loreforge/internal/domain/episode"
)

func TestGenerateIncludesStructuredTextAndReferences(t *testing.T) {
	t.Parallel()

	generator := Generator{
		GeneratorID: "short-story-artist",
		Format:      episode.OutputTypeShortStory,
		Settings:    text_settings.SystemTextDefaults[episode.OutputTypeShortStory],
		Provider:    fakeTextProvider{},
	}
	output, err := generator.Generate(context.Background(), episode.Brief{
		EpisodeType:  episode.OutputTypeShortStory,
		WorldID:      "glass-kingdom",
		CharacterIDs: []string{"red-wanderer"},
		EventID:      "lost-artifact",
		Tone:         "lyrical",
		Objective:    "Expand canon",
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
	if output.Text == nil || output.Text.Body == "" {
		t.Fatal("expected structured text artifact")
	}
	if output.ProviderRequest["json_schema"] == nil {
		t.Fatal("expected json schema in provider request")
	}
}

type fakeTextProvider struct{}

func (fakeTextProvider) Name() string { return "fake-text" }

func (fakeTextProvider) GenerateText(context.Context, contracts.TextRequest) (contracts.TextResponse, error) {
	return contracts.TextResponse{
		Title:   "Ash Garden",
		Content: "Red Wanderer crossed the ash bridge and The Architect answered from the gate.",
		Model:   "fake-text-v1",
	}, nil
}
