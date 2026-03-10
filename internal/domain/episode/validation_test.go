package episode

import (
	"strings"
	"testing"
)

func TestValidateOutputAcceptsValidTweetThread(t *testing.T) {
	t.Parallel()

	out := Output{
		Content: "Aria walks the ember road with Kade while the city listens.\n\nAria answers the gate and the oath returns with them.",
		Text: &TextArtifact{
			Parts: []TextPart{
				{Index: 0, Content: "Aria walks the ember road with Kade while the city listens."},
				{Index: 1, Content: "Aria answers the gate and the oath returns with them."},
			},
		},
	}
	brief := Brief{
		EpisodeType:  OutputTypeTweetThread,
		CharacterIDs: []string{"aria"},
		TextConstraints: &TextConstraints{
			Type:               OutputTypeTweetThread,
			MinParts:           2,
			MaxParts:           3,
			MaxCharsPerPart:    280,
			RequireEntityMatch: true,
		},
	}

	if err := ValidateOutput(out, brief); err != nil {
		t.Fatalf("ValidateOutput returned error: %v", err)
	}
}

func TestValidateOutputRejectsInvalidTextualFormats(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		out   Output
		brief Brief
	}{
		{
			name: "tweet short too long",
			out: Output{
				Content: "Aria " + repeatWord("ember", 80),
				Text:    &TextArtifact{Parts: []TextPart{{Index: 0, Content: "Aria " + repeatWord("ember", 80)}}},
			},
			brief: Brief{EpisodeType: OutputTypeTweetShort, TextConstraints: &TextConstraints{Type: OutputTypeTweetShort, MaxCharsPerPart: 20}},
		},
		{
			name: "entity missing",
			out:  Output{Content: "The archive trembles while the oath burns brighter than ever before in the city of glass."},
			brief: Brief{
				EpisodeType:  OutputTypeShortStory,
				CharacterIDs: []string{"aria"},
				TextConstraints: &TextConstraints{
					Type:               OutputTypeShortStory,
					RequireEntityMatch: true,
				},
			},
		},
		{
			name:  "song lyrics missing chorus",
			out:   Output{Content: "Verse 1\nThe city wakes beneath the ash and iron."},
			brief: Brief{EpisodeType: OutputTypeSongLyrics, TextConstraints: &TextConstraints{Type: OutputTypeSongLyrics}},
		},
		{
			name:  "screenplay missing heading",
			out:   Output{Content: "Aria crosses the gate while Kade waits beside the lantern and the old machines hum again."},
			brief: Brief{EpisodeType: OutputTypeScreenplaySeries, TextConstraints: &TextConstraints{Type: OutputTypeScreenplaySeries}},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateOutput(tt.out, tt.brief); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestContainsEntitiesAndTemplateMaxChars(t *testing.T) {
	t.Parallel()

	if !ContainsEntities("Red Wanderer sees the gate.", []string{"red-wanderer"}) {
		t.Fatal("expected hyphenated entity match")
	}
	if got := TemplateMaxChars("MAX_CHARS: 120\nBody"); got != 120 {
		t.Fatalf("TemplateMaxChars = %d, want 120", got)
	}
}

func TestValidateOutputTweetThreadMessageReflectsConfiguredLimits(t *testing.T) {
	t.Parallel()

	// When constraints deviate from defaults, the error message must show the
	// actual configured values, not the hardcoded "2-5 parts" string.
	out := Output{
		Content: "Part one of many.\n\nPart two keeps going.\n\nPart three is too much.",
		Text: &TextArtifact{
			Parts: []TextPart{
				{Index: 0, Content: "Part one of many."},
				{Index: 1, Content: "Part two keeps going."},
				{Index: 2, Content: "Part three is too much."},
			},
		},
	}
	brief := Brief{
		EpisodeType: OutputTypeTweetThread,
		TextConstraints: &TextConstraints{
			Type:     OutputTypeTweetThread,
			MinParts: 4,
			MaxParts: 8,
		},
	}

	err := ValidateOutput(out, brief)
	if err == nil {
		t.Fatal("expected validation error for thread below MinParts")
	}
	if !strings.Contains(err.Error(), "4") || !strings.Contains(err.Error(), "8") {
		t.Fatalf("error message %q does not include configured limits 4-8", err.Error())
	}
}

func repeatWord(word string, count int) string {
	out := ""
	for i := 0; i < count; i++ {
		if out != "" {
			out += " "
		}
		out += word
	}
	return out
}
