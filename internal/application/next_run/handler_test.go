package next_run

import (
	"context"
	"errors"
	"testing"
	"time"

	"loreforge/internal/application/ports"
	"loreforge/internal/application/scheduler"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/scheduling"
)

func TestHandleReturnsEarliestEnabledNextRun(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	repo := &fakeNextRunSchedulerRepo{}
	handler := Handler{
		Registry: fakeNextRunRegistry{
			items: []ports.RegisteredGenerator{
				{Config: ports.GeneratorConfig{ID: "disabled", SchedulerEnabled: false}},
				{Config: ports.GeneratorConfig{ID: "b", SchedulerEnabled: true, Scheduler: scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: 2 * time.Hour, Timezone: "UTC"}}},
				{Config: ports.GeneratorConfig{ID: "a", SchedulerEnabled: true, Scheduler: scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"}}},
			},
		},
		Scheduler: scheduler.Service{StateRepo: repo},
		Clock:     fakeNextRunClock{now: now},
	}

	next, err := handler.Handle(context.Background(), Request{})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if got, want := next, now.Add(time.Hour); !got.Equal(want) {
		t.Fatalf("next run = %s, want %s", got, want)
	}
}

func TestHandleReturnsSchedulerDisabled(t *testing.T) {
	t.Parallel()

	repo := &fakeNextRunSchedulerRepo{}
	handler := Handler{
		Registry: fakeNextRunRegistry{
			items: []ports.RegisteredGenerator{
				{Config: ports.GeneratorConfig{ID: "disabled", SchedulerEnabled: false}},
			},
		},
		Scheduler: scheduler.Service{StateRepo: repo},
		Clock:     fakeNextRunClock{now: time.Now().UTC()},
	}

	_, err := handler.Handle(context.Background(), Request{})
	if !errors.Is(err, episode.ErrSchedulerDisabled) {
		t.Fatalf("err = %v, want scheduler disabled", err)
	}
}

func TestHandleRejectsExplicitDisabledArtist(t *testing.T) {
	t.Parallel()

	repo := &fakeNextRunSchedulerRepo{}
	handler := Handler{
		Registry: fakeNextRunRegistry{
			items: []ports.RegisteredGenerator{
				{Config: ports.GeneratorConfig{ID: "artist", SchedulerEnabled: false}},
			},
		},
		Scheduler: scheduler.Service{StateRepo: repo},
		Clock:     fakeNextRunClock{now: time.Now().UTC()},
	}

	_, err := handler.Handle(context.Background(), Request{GeneratorID: "artist"})
	if !errors.Is(err, episode.ErrSchedulerDisabled) {
		t.Fatalf("err = %v, want scheduler disabled", err)
	}
}

type fakeNextRunRegistry struct {
	items []ports.RegisteredGenerator
}

func (f fakeNextRunRegistry) GetByID(id string) (ports.RegisteredGenerator, bool) {
	for _, item := range f.items {
		if item.Config.ID == id {
			return item, true
		}
	}
	return ports.RegisteredGenerator{}, false
}

func (f fakeNextRunRegistry) GetByType(episode.OutputType) (ports.RegisteredGenerator, bool) {
	return ports.RegisteredGenerator{}, false
}

func (f fakeNextRunRegistry) List() []ports.RegisteredGenerator { return f.items }

type fakeNextRunSchedulerRepo struct {
	state scheduling.State
}

func (f *fakeNextRunSchedulerRepo) Load(context.Context, string) (scheduling.State, error) {
	return f.state, nil
}

func (f *fakeNextRunSchedulerRepo) Save(context.Context, string, scheduling.State) error { return nil }

func (f *fakeNextRunSchedulerRepo) ListGeneratorIDs(context.Context) ([]string, error) {
	return nil, nil
}

type fakeNextRunClock struct {
	now time.Time
}

func (f fakeNextRunClock) Now() time.Time { return f.now }
