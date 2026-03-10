package list_artists

import (
	"context"
	"sort"
	"time"

	"loreforge/internal/application/ports"
	"loreforge/internal/application/scheduler"
	"loreforge/internal/domain/universe"
)

type Item struct {
	GeneratorID      string
	ProfileID        string
	ArtistName       string
	Type             string
	ProviderDriver   string
	ProviderModel    string
	SchedulerEnabled bool
	NextRun          *time.Time
	PublishTargets   []string
}

type Handler struct {
	Registry  ports.GeneratorRegistry
	Scheduler scheduler.Service
	Clock     ports.Clock
	Universe  universe.Universe
}

func (h Handler) Handle(ctx context.Context) ([]Item, error) {
	now := h.Clock.Now()
	items := h.Registry.List()
	out := make([]Item, 0, len(items))
	for _, item := range items {
		var next *time.Time
		if item.Config.SchedulerEnabled {
			scheduled, err := h.Scheduler.NextRun(ctx, item, now)
			if err != nil {
				return nil, err
			}
			next = &scheduled
		}
		name := item.Config.ProfileID
		if artist, ok := h.Universe.Artists[item.Config.ProfileID]; ok && artist.Name != "" {
			name = artist.Name
		}
		targets := make([]string, 0, len(item.Config.PublishTargets))
		for _, target := range item.Config.PublishTargets {
			label := string(target.Channel)
			if target.Account != "" {
				label += "(" + target.Account + ")"
			}
			targets = append(targets, label)
		}
		out = append(out, Item{
			GeneratorID:      item.Config.ID,
			ProfileID:        item.Config.ProfileID,
			ArtistName:       name,
			Type:             string(item.Config.Type),
			ProviderDriver:   item.Config.ProviderDriver,
			ProviderModel:    item.Config.ProviderModel,
			SchedulerEnabled: item.Config.SchedulerEnabled,
			NextRun:          next,
			PublishTargets:   targets,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GeneratorID < out[j].GeneratorID })
	return out, nil
}
