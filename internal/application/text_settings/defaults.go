package text_settings

import "loreforge/internal/domain/episode"

type ResolvedTextSettings struct {
	Type               episode.OutputType
	MinWords           int
	MaxWords           int
	MinParts           int
	MaxParts           int
	MaxCharsPerPart    int
	RequireEntityMatch bool
	RequireStructured  bool
	Temperature        float64
	MaxOutputTokens    int
	TargetParts        int
	TargetLineCount    int
	TargetSceneCount   int
	TemplateStrictness string
}

var SystemTextDefaults = map[episode.OutputType]ResolvedTextSettings{
	episode.OutputTypeTweetShort: {
		Type:               episode.OutputTypeTweetShort,
		MinParts:           1,
		MaxParts:           1,
		MaxCharsPerPart:    280,
		RequireEntityMatch: true,
		RequireStructured:  true,
		Temperature:        0.6,
		MaxOutputTokens:    250,
		TargetParts:        1,
		TemplateStrictness: "strict",
	},
	episode.OutputTypeTweetThread: {
		Type:               episode.OutputTypeTweetThread,
		MinParts:           2,
		MaxParts:           5,
		MaxCharsPerPart:    280,
		RequireEntityMatch: true,
		RequireStructured:  true,
		Temperature:        0.6,
		MaxOutputTokens:    900,
		TargetParts:        3,
		TemplateStrictness: "strict",
	},
	episode.OutputTypeShortStory: {
		Type:               episode.OutputTypeShortStory,
		MinWords:           400,
		MaxWords:           1200,
		RequireEntityMatch: true,
		RequireStructured:  true,
		Temperature:        0.8,
		MaxOutputTokens:    1800,
		TemplateStrictness: "balanced",
	},
	episode.OutputTypeLongStory: {
		Type:               episode.OutputTypeLongStory,
		MinWords:           1800,
		MaxWords:           5000,
		RequireEntityMatch: true,
		RequireStructured:  true,
		Temperature:        0.85,
		MaxOutputTokens:    4000,
		TemplateStrictness: "balanced",
	},
	episode.OutputTypePoem: {
		Type:               episode.OutputTypePoem,
		MinWords:           60,
		MaxWords:           500,
		RequireEntityMatch: false,
		RequireStructured:  true,
		Temperature:        0.9,
		MaxOutputTokens:    900,
		TargetLineCount:    16,
		TemplateStrictness: "loose",
	},
	episode.OutputTypeSongLyrics: {
		Type:               episode.OutputTypeSongLyrics,
		MinWords:           120,
		MaxWords:           900,
		RequireEntityMatch: false,
		RequireStructured:  true,
		Temperature:        0.85,
		MaxOutputTokens:    1400,
		TargetLineCount:    24,
		TemplateStrictness: "strict",
	},
	episode.OutputTypeScreenplaySeries: {
		Type:               episode.OutputTypeScreenplaySeries,
		MinWords:           500,
		MaxWords:           2500,
		RequireEntityMatch: true,
		RequireStructured:  true,
		Temperature:        0.7,
		MaxOutputTokens:    1800,
		TargetSceneCount:   4,
		TemplateStrictness: "strict",
	},
}
