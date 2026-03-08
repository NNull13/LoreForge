package episode

import "time"

type OutputType string

const (
	OutputTypeText  OutputType = "text"
	OutputTypeVideo OutputType = "video"
	OutputTypeImage OutputType = "image"
)

type Status string

const (
	StatusGenerated Status = "generated"
	StatusPublished Status = "published"
)

type Brief struct {
	EpisodeType   OutputType                `json:"episode_type"`
	WorldID       string                    `json:"world_id"`
	CharacterIDs  []string                  `json:"character_ids"`
	EventID       string                    `json:"event_id"`
	TemplateID    string                    `json:"template_id"`
	TemplateBody  string                    `json:"template_body,omitempty"`
	Tone          string                    `json:"tone"`
	Objective     string                    `json:"objective"`
	CanonRules    []string                  `json:"canon_rules"`
	CharacterData map[string]map[string]any `json:"character_data,omitempty"`
	WorldData     map[string]any            `json:"world_data,omitempty"`
	EventData     map[string]any            `json:"event_data,omitempty"`
}

type State struct {
	RecentEpisodeIDs []string `json:"recent_episode_ids"`
	UniverseVersion  string   `json:"universe_version"`
}

type Output struct {
	Content          string         `json:"content"`
	AssetPath        string         `json:"asset_path,omitempty"`
	Provider         string         `json:"provider"`
	Model            string         `json:"model"`
	Prompt           string         `json:"prompt"`
	ProviderRequest  map[string]any `json:"provider_request,omitempty"`
	ProviderResponse map[string]any `json:"provider_response,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type Manifest struct {
	EpisodeID       string         `json:"episode_id"`
	CreatedAt       time.Time      `json:"created_at"`
	Agent           string         `json:"agent"`
	ArtistID        string         `json:"artist_id,omitempty"`
	ArtistType      string         `json:"artist_type,omitempty"`
	ArtistStyle     string         `json:"artist_style,omitempty"`
	OutputType      string         `json:"output_type"`
	UniverseVersion string         `json:"universe_version"`
	WorldIDs        []string       `json:"world_ids"`
	CharacterIDs    []string       `json:"character_ids"`
	EventID         string         `json:"event_id"`
	TemplateID      string         `json:"template_id"`
	PromptInput     string         `json:"prompt_input"`
	PromptFinal     string         `json:"prompt_final"`
	Provider        string         `json:"provider"`
	Model           string         `json:"model"`
	Seed            int64          `json:"seed"`
	RetryCount      int            `json:"retry_count"`
	Published       bool           `json:"published"`
	Channels        []string       `json:"channels"`
	Scores          map[string]any `json:"scores"`
	State           string         `json:"state"`
}

type Record struct {
	Manifest         Manifest       `json:"manifest"`
	Context          map[string]any `json:"context"`
	Prompt           string         `json:"prompt"`
	ProviderRequest  map[string]any `json:"provider_request"`
	ProviderResponse map[string]any `json:"provider_response"`
	OutputText       string         `json:"output_text,omitempty"`
	OutputAssetPath  string         `json:"output_asset_path,omitempty"`
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
