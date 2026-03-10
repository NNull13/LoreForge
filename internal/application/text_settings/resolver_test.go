package text_settings

import (
	"testing"

	"loreforge/internal/config"
)

func TestResolveTextSettingsPrecedence(t *testing.T) {
	t.Parallel()

	temp := 0.9
	cfg := config.Config{
		Text: config.TextGenerationConfig{
			Formats: map[string]config.TextFormatConfig{
				"tweet_thread": {
					TargetParts:     4,
					MaxOutputTokens: 700,
				},
			},
		},
	}
	artist := config.ArtistConfig{
		ID:   "tweet-thread-artist",
		Type: "tweet_thread",
		Provider: config.ProviderDriver{
			Driver:  "openai_text",
			Model:   "gpt-5-mini",
			Options: map[string]any{"max_output_tokens": 600, "temperature": 0.7},
		},
		Options: map[string]any{
			"text": map[string]any{
				"temperature":        temp,
				"target_parts":       5,
				"max_chars_per_part": 280,
			},
		},
	}

	resolved, err := ResolveTextSettings(cfg, artist)
	if err != nil {
		t.Fatalf("ResolveTextSettings returned error: %v", err)
	}
	if resolved.TargetParts != 5 {
		t.Fatalf("expected artist override to win, got %d", resolved.TargetParts)
	}
	if resolved.Temperature != 0.9 {
		t.Fatalf("expected artist temperature override, got %f", resolved.Temperature)
	}
	if resolved.MaxOutputTokens != 700 {
		t.Fatalf("expected format override for max_output_tokens, got %d", resolved.MaxOutputTokens)
	}
}
