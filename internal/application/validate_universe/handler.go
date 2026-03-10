package validate_universe

import (
	"context"
	"fmt"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/episode"
)

type Handler struct {
	UniverseRepo ports.UniverseRepository
}

func (h Handler) Handle(ctx context.Context) error {
	_, err := h.UniverseRepo.Load(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", episode.ErrUniverseInvalid, err)
	}
	return nil
}
