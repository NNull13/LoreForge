package contracts

import (
	"time"

	providercontracts "loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/publication"
)

type EpisodeBrief = episode.Brief
type EpisodeState = episode.State
type EpisodeOutput = episode.Output
type TextRequest = providercontracts.TextRequest
type TextResponse = providercontracts.TextResponse
type VideoRequest = providercontracts.VideoRequest
type VideoResponse = providercontracts.VideoResponse
type ImageRequest = providercontracts.ImageRequest
type ImageResponse = providercontracts.ImageResponse

type PublishableContent struct {
	EpisodeID  string    `json:"episode_id"`
	ArtistID   string    `json:"artist_id,omitempty"`
	ArtistType string    `json:"artist_type,omitempty"`
	OutputType string    `json:"output_type"`
	Content    string    `json:"content,omitempty"`
	AssetPath  string    `json:"asset_path,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type PublishResult = publication.Result
