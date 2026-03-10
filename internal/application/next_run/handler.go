package next_run

import (
	"context"
	"fmt"
	"time"

	"loreforge/internal/application/ports"
	"loreforge/internal/application/scheduler"
	"loreforge/internal/domain/episode"
)

type Request struct {
	GeneratorID string
}

type Handler struct {
	Registry  ports.GeneratorRegistry
	Scheduler scheduler.Service
	Clock     ports.Clock
}

func (h Handler) Handle(ctx context.Context, req Request) (time.Time, error) {
	now := h.Clock.Now()
	if req.GeneratorID != "" {
		def, ok := h.Registry.GetByID(req.GeneratorID)
		if !ok {
			return time.Time{}, fmt.Errorf("%w: %s", episode.ErrGeneratorUnavailable, req.GeneratorID)
		}
		if !def.Config.SchedulerEnabled {
			return time.Time{}, fmt.Errorf("%w: %s", episode.ErrSchedulerDisabled, req.GeneratorID)
		}
		return h.Scheduler.NextRun(ctx, def, now)
	}
	items := h.Registry.List()
	if len(items) == 0 {
		return time.Time{}, episode.ErrNoGeneratorsAvailable
	}
	var best time.Time
	enabled := 0
	for _, item := range items {
		if !item.Config.SchedulerEnabled {
			continue
		}
		enabled++
		next, err := h.Scheduler.NextRun(ctx, item, now)
		if err != nil {
			return time.Time{}, err
		}
		if best.IsZero() || next.Before(best) {
			best = next
		}
	}
	if enabled == 0 {
		return time.Time{}, episode.ErrSchedulerDisabled
	}
	return best, nil
}
