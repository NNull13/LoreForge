package contracts

import (
	"context"
	"time"
)

type EpisodeBrief struct {
	EpisodeType   string                    `json:"episode_type"`
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

type EpisodeState struct {
	RecentEpisodeIDs []string `json:"recent_episode_ids"`
	UniverseVersion  string   `json:"universe_version"`
}

type EpisodeOutput struct {
	Content          string         `json:"content"`
	AssetPath        string         `json:"asset_path,omitempty"`
	Provider         string         `json:"provider"`
	Model            string         `json:"model"`
	Prompt           string         `json:"prompt"`
	ProviderRequest  map[string]any `json:"provider_request,omitempty"`
	ProviderResponse map[string]any `json:"provider_response,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type Agent interface {
	Name() string
	OutputType() string
	Generate(ctx context.Context, brief EpisodeBrief, state EpisodeState) (EpisodeOutput, error)
}

type TextRequest struct {
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

type TextResponse struct {
	Content string `json:"content"`
	Model   string `json:"model"`
}

type VideoRequest struct {
	Prompt   string `json:"prompt"`
	Duration int    `json:"duration_seconds"`
	Seed     int64  `json:"seed"`
}

type VideoResponse struct {
	AssetPath string `json:"asset_path"`
	Model     string `json:"model"`
}

type TextProvider interface {
	GenerateText(ctx context.Context, input TextRequest) (TextResponse, error)
	Name() string
}

type VideoProvider interface {
	GenerateVideo(ctx context.Context, input VideoRequest) (VideoResponse, error)
	Name() string
}

type PublishableContent struct {
	EpisodeID  string    `json:"episode_id"`
	OutputType string    `json:"output_type"`
	Content    string    `json:"content,omitempty"`
	AssetPath  string    `json:"asset_path,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type PublishResult struct {
	Channel    string `json:"channel"`
	Success    bool   `json:"success"`
	ExternalID string `json:"external_id,omitempty"`
	Message    string `json:"message,omitempty"`
}

type Channel interface {
	Name() string
	Publish(ctx context.Context, item PublishableContent) (PublishResult, error)
}
