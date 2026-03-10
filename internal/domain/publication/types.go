package publication

import "time"

type ChannelName string

const (
	ChannelFilesystem ChannelName = "filesystem"
	ChannelTwitter    ChannelName = "twitter"
)

type Target struct {
	Channel ChannelName `json:"channel"`
	Account string      `json:"account,omitempty"`
}

type Item struct {
	EpisodeID     string         `json:"episode_id"`
	GeneratorID   string         `json:"generator_id,omitempty"`
	GeneratorType string         `json:"generator_type,omitempty"`
	Target        Target         `json:"target"`
	OutputType    string         `json:"output_type"`
	Format        string         `json:"format,omitempty"`
	Content       string         `json:"content,omitempty"`
	Parts         []string       `json:"parts,omitempty"`
	AssetPath     string         `json:"asset_path,omitempty"`
	Caption       string         `json:"caption,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type Result struct {
	Channel    string         `json:"channel"`
	Success    bool           `json:"success"`
	ExternalID string         `json:"external_id,omitempty"`
	Message    string         `json:"message,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}
