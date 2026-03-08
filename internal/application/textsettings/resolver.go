package textsettings

import (
	"encoding/json"
	"fmt"
	"strings"

	"loreforge/internal/config"
	"loreforge/internal/domain/episode"
)

func ResolveTextSettings(cfg config.Config, artist config.ArtistConfig) (ResolvedTextSettings, error) {
	outputType := episode.OutputType(artist.Type)
	base, ok := SystemTextDefaults[outputType]
	if !ok {
		return ResolvedTextSettings{}, fmt.Errorf("unsupported text type: %s", artist.Type)
	}
	base.Type = outputType
	applyProviderOptions(&base, artist.Provider.Options)
	if global, ok := cfg.Text.Formats[artist.Type]; ok {
		applyConfig(&base, global)
	}
	if nested, ok := decodeTextFormatConfig(artist.Options["text"]); ok {
		applyConfig(&base, nested)
	}
	if err := validate(base); err != nil {
		return ResolvedTextSettings{}, fmt.Errorf("text settings for %s invalid: %w", artist.ID, err)
	}
	return base, nil
}

func (r ResolvedTextSettings) ToConstraints() *episode.TextConstraints {
	return &episode.TextConstraints{
		Type:               r.Type,
		MinWords:           r.MinWords,
		MaxWords:           r.MaxWords,
		MinParts:           r.MinParts,
		MaxParts:           r.MaxParts,
		MaxCharsPerPart:    r.MaxCharsPerPart,
		RequireEntityMatch: r.RequireEntityMatch,
		RequireStructured:  r.RequireStructured,
		Temperature:        r.Temperature,
		MaxOutputTokens:    r.MaxOutputTokens,
		TargetParts:        r.TargetParts,
		TargetLineCount:    r.TargetLineCount,
		TargetSceneCount:   r.TargetSceneCount,
		TemplateStrictness: r.TemplateStrictness,
		TwitterPublishable: r.TwitterPublishable,
	}
}

func applyProviderOptions(dst *ResolvedTextSettings, options map[string]any) {
	if v, ok := floatFromAny(options["temperature"]); ok {
		dst.Temperature = v
	}
	if v, ok := intFromAny(options["max_output_tokens"]); ok {
		dst.MaxOutputTokens = v
	}
	if v, ok := boolFromAny(options["structured_output"]); ok {
		dst.RequireStructured = v
	}
}

func applyConfig(dst *ResolvedTextSettings, cfg config.TextFormatConfig) {
	if cfg.MinWords > 0 {
		dst.MinWords = cfg.MinWords
	}
	if cfg.MaxWords > 0 {
		dst.MaxWords = cfg.MaxWords
	}
	if cfg.MinParts > 0 {
		dst.MinParts = cfg.MinParts
	}
	if cfg.MaxParts > 0 {
		dst.MaxParts = cfg.MaxParts
	}
	if cfg.MaxCharsPerPart > 0 {
		dst.MaxCharsPerPart = cfg.MaxCharsPerPart
	}
	if cfg.RequireEntityMatch != nil {
		dst.RequireEntityMatch = *cfg.RequireEntityMatch
	}
	if cfg.RequireStructured != nil {
		dst.RequireStructured = *cfg.RequireStructured
	}
	if cfg.Temperature != nil {
		dst.Temperature = *cfg.Temperature
	}
	if cfg.MaxOutputTokens > 0 {
		dst.MaxOutputTokens = cfg.MaxOutputTokens
	}
	if cfg.TargetParts > 0 {
		dst.TargetParts = cfg.TargetParts
	}
	if cfg.TargetLineCount > 0 {
		dst.TargetLineCount = cfg.TargetLineCount
	}
	if cfg.TargetSceneCount > 0 {
		dst.TargetSceneCount = cfg.TargetSceneCount
	}
	if strings.TrimSpace(cfg.TemplateStrictness) != "" {
		dst.TemplateStrictness = cfg.TemplateStrictness
	}
	if cfg.TwitterPublishable != nil {
		dst.TwitterPublishable = *cfg.TwitterPublishable
	}
}

func validate(s ResolvedTextSettings) error {
	if s.MinWords > 0 && s.MaxWords > 0 && s.MinWords > s.MaxWords {
		return fmt.Errorf("min_words > max_words")
	}
	if s.MinParts > 0 && s.MaxParts > 0 && s.MinParts > s.MaxParts {
		return fmt.Errorf("min_parts > max_parts")
	}
	if s.TargetParts > 0 && ((s.MinParts > 0 && s.TargetParts < s.MinParts) || (s.MaxParts > 0 && s.TargetParts > s.MaxParts)) {
		return fmt.Errorf("target_parts outside allowed range")
	}
	if s.TwitterPublishable && s.MaxCharsPerPart <= 0 {
		return fmt.Errorf("twitter_publishable requires max_chars_per_part")
	}
	switch s.Type {
	case episode.OutputTypeTweetShort:
		if s.MinParts != 1 || s.MaxParts != 1 {
			return fmt.Errorf("tweet_short must resolve to exactly one part")
		}
	case episode.OutputTypeTweetThread:
		if s.MaxParts < 2 {
			return fmt.Errorf("tweet_thread requires max_parts >= 2")
		}
	case episode.OutputTypeSongLyrics, episode.OutputTypeScreenplaySeries:
		if !s.RequireStructured {
			return fmt.Errorf("%s requires structured output", s.Type)
		}
	}
	return nil
}

func decodeTextFormatConfig(v any) (config.TextFormatConfig, bool) {
	if v == nil {
		return config.TextFormatConfig{}, false
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return config.TextFormatConfig{}, false
	}
	var out config.TextFormatConfig
	if err := json.Unmarshal(raw, &out); err != nil {
		return config.TextFormatConfig{}, false
	}
	return out, true
}

func intFromAny(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	default:
		return 0, false
	}
}

func floatFromAny(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	default:
		return 0, false
	}
}

func boolFromAny(v any) (bool, bool) {
	value, ok := v.(bool)
	return value, ok
}
