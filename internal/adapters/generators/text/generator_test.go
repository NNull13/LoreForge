package text

import (
	"strings"
	"testing"

	"loreforge/internal/domain/episode"
)

func TestBuildPromptIncludesReferences(t *testing.T) {
	t.Parallel()

	prompt := buildPrompt(episode.Brief{
		EpisodeType:  episode.OutputTypeText,
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
	})

	if !strings.Contains(prompt, "Continuity Memories") {
		t.Fatal("expected continuity section in prompt")
	}
	if !strings.Contains(prompt, "Visual Canon References") {
		t.Fatal("expected visual refs section in prompt")
	}
}
