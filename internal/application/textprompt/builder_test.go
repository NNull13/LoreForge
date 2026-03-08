package textprompt

import (
	"strings"
	"testing"

	"loreforge/internal/application/textsettings"
	"loreforge/internal/domain/episode"
)

func TestBuildTweetThreadIncludesSchemaAndReferences(t *testing.T) {
	t.Parallel()

	bundle := Build(episode.Brief{
		EpisodeType:  episode.OutputTypeTweetThread,
		WorldID:      "ember-city",
		CharacterIDs: []string{"aria"},
		EventID:      "gate-whisper",
		Objective:    "Expand canon",
		Tone:         "ritual",
		CanonRules:   []string{"Keep continuity"},
		TemplateBody: "Use numbered parts.",
		Artist: episode.ArtistLens{
			ID:             "ash-chorister",
			Name:           "The Ash Chorister",
			Mission:        "Preserve canon.",
			PromptingRules: []string{"Never contradict canon."},
			NonDiegetic:    true,
			Presentation: episode.ArtistPresentationSnapshot{
				Enabled:       true,
				SignatureMode: "append",
				FramingMode:   "intro_outro",
				SignatureText: "Filed by the archive.",
			},
		},
		ContinuityReferences: []episode.ContinuityReference{{EpisodeID: "ep-1", Summary: "Aria crossed the ash bridge."}},
		VisualReferences:     []episode.VisualReference{{AssetID: "hero", Usage: "character_reference", Description: "Portrait reference"}},
	}, episode.OutputTypeTweetThread, textsettings.ResolvedTextSettings{
		Type:            episode.OutputTypeTweetThread,
		MinParts:        2,
		MaxParts:        4,
		MaxCharsPerPart: 280,
	})

	if !strings.Contains(bundle.SystemPrompt, "tweet thread") {
		t.Fatalf("unexpected system prompt: %s", bundle.SystemPrompt)
	}
	properties := bundle.JSONSchema["properties"].(map[string]any)
	parts := properties["parts"].(map[string]any)
	if parts["minItems"].(int) != 2 || parts["maxItems"].(int) != 4 {
		t.Fatalf("unexpected schema bounds: %#v", parts)
	}
	for _, fragment := range []string{"Artist Prompting Rules", "Continuity Memories", "Visual Canon References", "Template:"} {
		if !strings.Contains(bundle.UserPrompt, fragment) {
			t.Fatalf("user prompt missing %q: %s", fragment, bundle.UserPrompt)
		}
	}
}

func TestBuildShortStoryIncludesPresentationPolicyWhenDisabled(t *testing.T) {
	t.Parallel()

	bundle := Build(episode.Brief{
		EpisodeType:  episode.OutputTypeShortStory,
		WorldID:      "ember-city",
		CharacterIDs: []string{"aria"},
		EventID:      "gate-whisper",
		Objective:    "Expand canon",
		Tone:         "steady",
		CanonRules:   []string{"Keep continuity"},
		Artist: episode.ArtistLens{
			ID:          "ash-chorister",
			Name:        "The Ash Chorister",
			Mission:     "Preserve canon.",
			NonDiegetic: true,
			Presentation: episode.ArtistPresentationSnapshot{
				Enabled: false,
			},
		},
	}, episode.OutputTypeShortStory, textsettings.ResolvedTextSettings{
		Type:     episode.OutputTypeShortStory,
		MinWords: 400,
		MaxWords: 1200,
	})

	if !strings.Contains(bundle.UserPrompt, "No visible artist framing") {
		t.Fatalf("unexpected user prompt: %s", bundle.UserPrompt)
	}
	if bundle.JSONSchema["type"] != "object" {
		t.Fatalf("unexpected schema: %#v", bundle.JSONSchema)
	}
}

func TestBuildSystemPromptCoversFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		format episode.OutputType
		want   string
	}{
		{format: episode.OutputTypeTweetShort, want: "one concise tweet"},
		{format: episode.OutputTypeTweetThread, want: "tweet thread"},
		{format: episode.OutputTypeShortStory, want: "short story scene"},
		{format: episode.OutputTypeLongStory, want: "long-form story"},
		{format: episode.OutputTypePoem, want: "poem"},
		{format: episode.OutputTypeSongLyrics, want: "song lyrics"},
		{format: episode.OutputTypeScreenplaySeries, want: "series screenplay excerpt"},
		{format: episode.OutputTypeImage, want: "canon-consistent creative text"},
	}

	for _, tt := range cases {
		if got := buildSystemPrompt(tt.format, textsettings.ResolvedTextSettings{MinParts: 2, MaxParts: 4}); !strings.Contains(got, tt.want) {
			t.Fatalf("buildSystemPrompt(%s) = %q, want substring %q", tt.format, got, tt.want)
		}
	}
}

func TestPromptHelperFormatting(t *testing.T) {
	t.Parallel()

	if got := artistIdentity(episode.ArtistLens{Name: "Ash", Title: "Archivist"}); got != "Ash (Archivist)" {
		t.Fatalf("artistIdentity with title = %q", got)
	}
	if got := artistIdentity(episode.ArtistLens{Name: "Ash"}); got != "Ash" {
		t.Fatalf("artistIdentity with name = %q", got)
	}
	if got := artistIdentity(episode.ArtistLens{ID: "ash-id"}); got != "ash-id" {
		t.Fatalf("artistIdentity fallback = %q", got)
	}

	voice := artistVoice(episode.ArtistLens{
		Voice: map[string]string{
			"cadence":   "ritual",
			"register":  "elevated",
			"intensity": "high",
		},
	})
	for _, fragment := range []string{"- register: elevated", "- cadence: ritual", "- intensity: high"} {
		if !strings.Contains(voice, fragment) {
			t.Fatalf("artistVoice missing %q in %q", fragment, voice)
		}
	}
	if got := artistVoice(episode.ArtistLens{}); got != "" {
		t.Fatalf("artistVoice empty = %q, want empty", got)
	}

	if got := artistSignaturePolicy(episode.ArtistLens{Presentation: episode.ArtistPresentationSnapshot{Enabled: false}}); !strings.Contains(got, "No visible artist framing") {
		t.Fatalf("artistSignaturePolicy disabled = %q", got)
	}
	if got := artistSignaturePolicy(episode.ArtistLens{Presentation: episode.ArtistPresentationSnapshot{Enabled: true, SignatureMode: "append", FramingMode: "intro", SignatureText: "Signed"}}); !strings.Contains(got, "signature_mode=append") {
		t.Fatalf("artistSignaturePolicy enabled = %q", got)
	}

	continuity := formatContinuityReferences([]episode.ContinuityReference{
		{EpisodeID: "ep-1", Summary: "Aria crossed the bridge."},
		{EpisodeID: "ep-2", OutputText: "Kade answered the gate."},
		{EpisodeID: "ep-3"},
	})
	if !strings.Contains(continuity, "Episode ep-1") || !strings.Contains(continuity, "Kade answered the gate.") || strings.Contains(continuity, "ep-3") {
		t.Fatalf("unexpected continuity formatting: %q", continuity)
	}

	visual := formatVisualReferences([]episode.VisualReference{
		{AssetID: "hero", Usage: "character_reference", Description: "Portrait"},
		{Path: "/tmp/scene.png", Usage: "environment_reference"},
	})
	if !strings.Contains(visual, "hero (character_reference): Portrait") || !strings.Contains(visual, "/tmp/scene.png (environment_reference)") {
		t.Fatalf("unexpected visual formatting: %q", visual)
	}

	shortSchema := schemaFor(episode.OutputTypeTweetShort, textsettings.ResolvedTextSettings{})
	if shortSchema["type"] != "object" {
		t.Fatalf("unexpected short schema: %#v", shortSchema)
	}
	defaultSchema := schemaFor(episode.OutputTypeShortStory, textsettings.ResolvedTextSettings{})
	if required := defaultSchema["required"].([]string); len(required) != 1 || required[0] != "body" {
		t.Fatalf("unexpected default schema required fields: %#v", required)
	}
}
