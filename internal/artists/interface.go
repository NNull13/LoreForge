package artists

import (
	"context"

	"loreforge/pkg/contracts"
)

// Artist defines the common generation contract for all artist types.
type Artist interface {
	Name() string
	OutputType() string
	Generate(ctx context.Context, brief contracts.EpisodeBrief, state contracts.EpisodeState) (contracts.EpisodeOutput, error)
}
