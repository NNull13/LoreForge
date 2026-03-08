package showepisode

import (
	"context"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/episode"
)

type Request struct {
	EpisodeID string
}

type Result struct {
	Path     string
	Manifest episode.Manifest
}

type Handler struct {
	EpisodeRepo ports.EpisodeRepository
}

func (h Handler) Handle(ctx context.Context, req Request) (Result, error) {
	stored, err := h.EpisodeRepo.FindByID(ctx, req.EpisodeID)
	if err != nil {
		return Result{}, err
	}
	return Result{Path: stored.Path, Manifest: stored.Manifest}, nil
}
