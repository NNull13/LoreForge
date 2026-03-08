package configrefresh

import (
	"context"
	"sort"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/scheduling"
)

type Result struct {
	Active    int
	Created   []string
	Preserved []string
	Orphaned  []string
}

type Handler struct {
	Registry           ports.GeneratorRegistry
	SchedulerStateRepo ports.SchedulerStateRepository
	Clock              ports.Clock
}

func (h Handler) Handle(ctx context.Context) (Result, error) {
	now := h.Clock.Now()
	items := h.Registry.List()
	active := make(map[string]bool, len(items))
	result := Result{Active: len(items)}
	for _, item := range items {
		active[item.Config.ID] = true
		state, err := h.SchedulerStateRepo.Load(ctx, item.Config.ID)
		if err != nil {
			return Result{}, err
		}
		if state.LastRunAt != nil || !state.NextRunAt.IsZero() {
			result.Preserved = append(result.Preserved, item.Config.ID)
			continue
		}
		scheduler, err := scheduling.NewScheduler(item.Config.Scheduler)
		if err != nil {
			return Result{}, err
		}
		next, err := scheduler.NextRun(now)
		if err != nil {
			return Result{}, err
		}
		if err := h.SchedulerStateRepo.Save(ctx, item.Config.ID, scheduling.State{NextRunAt: next}); err != nil {
			return Result{}, err
		}
		result.Created = append(result.Created, item.Config.ID)
	}
	existing, err := h.SchedulerStateRepo.ListGeneratorIDs(ctx)
	if err != nil {
		return Result{}, err
	}
	for _, id := range existing {
		if !active[id] {
			result.Orphaned = append(result.Orphaned, id)
		}
	}
	sort.Strings(result.Created)
	sort.Strings(result.Preserved)
	sort.Strings(result.Orphaned)
	return result, nil
}
