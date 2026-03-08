package contracts

import (
	providercontracts "loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/publication"
)

type EpisodeBrief = episode.Brief
type EpisodeState = episode.State
type EpisodeOutput = episode.Output
type EpisodeManifest = episode.Manifest
type EpisodeRecord = episode.Record
type TextRequest = providercontracts.TextRequest
type TextResponse = providercontracts.TextResponse
type VideoRequest = providercontracts.VideoRequest
type VideoResponse = providercontracts.VideoResponse
type ImageRequest = providercontracts.ImageRequest
type ImageResponse = providercontracts.ImageResponse
type PublishableContent = publication.Item
type PublishResult = publication.Result
