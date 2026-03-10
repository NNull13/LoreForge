package episode

import "time"

type OutputType string

const (
	OutputTypeTweetShort       OutputType = "tweet_short"
	OutputTypeTweetThread      OutputType = "tweet_thread"
	OutputTypeShortStory       OutputType = "short_story"
	OutputTypeLongStory        OutputType = "long_story"
	OutputTypePoem             OutputType = "poem"
	OutputTypeSongLyrics       OutputType = "song_lyrics"
	OutputTypeScreenplaySeries OutputType = "screenplay_series"
	OutputTypeVideo            OutputType = "video"
	OutputTypeImage            OutputType = "image"
)

type Status string

const (
	StatusGenerated     Status = "generated"
	StatusPublished     Status = "published"
	StatusPublishFailed Status = "publish_failed"
)

type Brief struct {
	EpisodeType          OutputType                `json:"episode_type"`
	WorldID              string                    `json:"world_id"`
	CharacterIDs         []string                  `json:"character_ids"`
	EventID              string                    `json:"event_id"`
	TemplateID           string                    `json:"template_id"`
	TemplateBody         string                    `json:"template_body,omitempty"`
	Tone                 string                    `json:"tone"`
	Objective            string                    `json:"objective"`
	CanonRules           []string                  `json:"canon_rules"`
	CharacterData        map[string]map[string]any `json:"character_data,omitempty"`
	WorldData            map[string]any            `json:"world_data,omitempty"`
	EventData            map[string]any            `json:"event_data,omitempty"`
	Artist               ArtistLens                `json:"artist"`
	VisualReferences     []VisualReference         `json:"visual_references,omitempty"`
	ContinuityReferences []ContinuityReference     `json:"continuity_references,omitempty"`
	TextConstraints      *TextConstraints          `json:"text_constraints,omitempty"`
}

type ArtistLens struct {
	ID             string                     `json:"id"`
	Name           string                     `json:"name"`
	Title          string                     `json:"title,omitempty"`
	Role           string                     `json:"role"`
	Summary        string                     `json:"summary,omitempty"`
	Body           string                     `json:"body,omitempty"`
	NonDiegetic    bool                       `json:"non_diegetic"`
	Voice          map[string]string          `json:"voice,omitempty"`
	Mission        string                     `json:"mission,omitempty"`
	PromptingRules []string                   `json:"prompting_rules,omitempty"`
	TonalBiases    []string                   `json:"tonal_biases,omitempty"`
	LexicalCues    []string                   `json:"lexical_cues,omitempty"`
	Forbidden      []string                   `json:"forbidden,omitempty"`
	Presentation   ArtistPresentationSnapshot `json:"presentation"`
	VisualRefs     []VisualReference          `json:"visual_refs,omitempty"`
}

type ArtistPresentationSnapshot struct {
	Enabled         bool     `json:"enabled"`
	SignatureMode   string   `json:"signature_mode,omitempty"`
	SignatureText   string   `json:"signature_text,omitempty"`
	FramingMode     string   `json:"framing_mode,omitempty"`
	IntroTemplate   string   `json:"intro_template,omitempty"`
	OutroTemplate   string   `json:"outro_template,omitempty"`
	AllowedChannels []string `json:"allowed_channels,omitempty"`
}

type VisualReference struct {
	Source      string `json:"source"`
	EntityType  string `json:"entity_type"`
	EntityID    string `json:"entity_id"`
	AssetID     string `json:"asset_id"`
	Path        string `json:"path"`
	MediaType   string `json:"media_type"`
	Usage       string `json:"usage"`
	Description string `json:"description"`
	Weight      int    `json:"weight"`
	ModelRole   string `json:"model_role"`
}

type ContinuityReference struct {
	EpisodeID       string    `json:"episode_id"`
	GeneratorID     string    `json:"generator_id"`
	CreatedAt       time.Time `json:"created_at"`
	Prompt          string    `json:"prompt,omitempty"`
	OutputText      string    `json:"output_text,omitempty"`
	OutputAssetPath string    `json:"output_asset_path,omitempty"`
	Summary         string    `json:"summary,omitempty"`
}

type State struct {
	RecentEpisodeIDs []string       `json:"recent_episode_ids"`
	UniverseVersion  string         `json:"universe_version"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type TextPart struct {
	Index   int    `json:"index"`
	Content string `json:"content"`
}

type TextArtifact struct {
	Title          string     `json:"title,omitempty"`
	Body           string     `json:"body,omitempty"`
	Parts          []TextPart `json:"parts,omitempty"`
	WordCount      int        `json:"word_count,omitempty"`
	CharacterCount int        `json:"character_count,omitempty"`
}

type TextConstraints struct {
	Type               OutputType `json:"type"`
	MinWords           int        `json:"min_words,omitempty"`
	MaxWords           int        `json:"max_words,omitempty"`
	MinParts           int        `json:"min_parts,omitempty"`
	MaxParts           int        `json:"max_parts,omitempty"`
	MaxCharsPerPart    int        `json:"max_chars_per_part,omitempty"`
	RequireEntityMatch bool       `json:"require_entity_match,omitempty"`
	RequireStructured  bool       `json:"require_structured,omitempty"`
	Temperature        float64    `json:"temperature,omitempty"`
	MaxOutputTokens    int        `json:"max_output_tokens,omitempty"`
	TargetParts        int        `json:"target_parts,omitempty"`
	TargetLineCount    int        `json:"target_line_count,omitempty"`
	TargetSceneCount   int        `json:"target_scene_count,omitempty"`
	TemplateStrictness string     `json:"template_strictness,omitempty"`
}

type Output struct {
	Content          string         `json:"content"`
	AssetPath        string         `json:"asset_path,omitempty"`
	Text             *TextArtifact  `json:"text,omitempty"`
	Provider         string         `json:"provider"`
	Model            string         `json:"model"`
	Prompt           string         `json:"prompt"`
	ProviderRequest  map[string]any `json:"provider_request,omitempty"`
	ProviderResponse map[string]any `json:"provider_response,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type Manifest struct {
	EpisodeID              string         `json:"episode_id"`
	CreatedAt              time.Time      `json:"created_at"`
	Agent                  string         `json:"agent"`
	ArtistID               string         `json:"artist_id,omitempty"`
	ArtistType             string         `json:"artist_type,omitempty"`
	ArtistStyle            string         `json:"artist_style,omitempty"`
	ArtistProfileID        string         `json:"artist_profile_id,omitempty"`
	ArtistName             string         `json:"artist_name,omitempty"`
	ArtistRole             string         `json:"artist_role,omitempty"`
	ArtistPresentationMode string         `json:"artist_presentation_mode,omitempty"`
	OutputType             string         `json:"output_type"`
	UniverseVersion        string         `json:"universe_version"`
	WorldIDs               []string       `json:"world_ids"`
	CharacterIDs           []string       `json:"character_ids"`
	EventID                string         `json:"event_id"`
	TemplateID             string         `json:"template_id"`
	PromptInput            string         `json:"prompt_input"`
	PromptFinal            string         `json:"prompt_final"`
	Provider               string         `json:"provider"`
	Model                  string         `json:"model"`
	Seed                   int64          `json:"seed"`
	RetryCount             int            `json:"retry_count"`
	Published              bool           `json:"published"`
	Channels               []string       `json:"channels"`
	Scores                 map[string]any `json:"scores"`
	State                  string         `json:"state"`
}

type Record struct {
	Manifest         Manifest       `json:"manifest"`
	Context          map[string]any `json:"context"`
	Prompt           string         `json:"prompt"`
	ProviderRequest  map[string]any `json:"provider_request"`
	ProviderResponse map[string]any `json:"provider_response"`
	OutputText       string         `json:"output_text,omitempty"`
	OutputParts      []string       `json:"output_parts,omitempty"`
	OutputAssetPath  string         `json:"output_asset_path,omitempty"`
	ArtistSnapshot   map[string]any `json:"artist_snapshot,omitempty"`
	Presentation     map[string]any `json:"presentation,omitempty"`
	Publish          map[string]any `json:"publish,omitempty"`
}

type StoredRecord struct {
	Path     string
	Manifest Manifest
}

type Combo struct {
	WorldID      string
	CharacterIDs []string
	EventID      string
}

func (t OutputType) IsTextual() bool {
	switch t {
	case OutputTypeTweetShort, OutputTypeTweetThread, OutputTypeShortStory, OutputTypeLongStory, OutputTypePoem, OutputTypeSongLyrics, OutputTypeScreenplaySeries:
		return true
	default:
		return false
	}
}
