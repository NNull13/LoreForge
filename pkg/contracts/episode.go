package contracts

import "time"

type EpisodeManifest struct {
	EpisodeID       string         `json:"episode_id"`
	CreatedAt       time.Time      `json:"created_at"`
	Agent           string         `json:"agent"`
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

type EpisodeRecord struct {
	Manifest         EpisodeManifest `json:"manifest"`
	Context          map[string]any  `json:"context"`
	Prompt           string          `json:"prompt"`
	ProviderRequest  map[string]any  `json:"provider_request"`
	ProviderResponse map[string]any  `json:"provider_response"`
	OutputText       string          `json:"output_text,omitempty"`
	OutputAssetPath  string          `json:"output_asset_path,omitempty"`
	Publish          map[string]any  `json:"publish,omitempty"`
}
