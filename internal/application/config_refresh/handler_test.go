package config_refresh

import (
	"context"
	"testing"
	"time"

	"loreforge/internal/application/ports"
	"loreforge/internal/application/scheduler"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/scheduling"
)

func TestHandlePreservesExistingAndCreatesMissing(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	repo := &fakeSchedulerRepo{
		states: map[string]scheduling.State{
			"existing-artist": {NextRunAt: now.Add(2 * time.Hour)},
		},
		list: []string{"existing-artist", "orphaned-artist"},
	}
	handler := Handler{
		Registry: fakeRefreshRegistry{
			items: []ports.RegisteredGenerator{
				{Config: ports.GeneratorConfig{ID: "existing-artist", SchedulerEnabled: true, Scheduler: scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"}}},
				{Config: ports.GeneratorConfig{ID: "new-artist", SchedulerEnabled: true, Scheduler: scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"}}},
			},
		},
		Scheduler:          scheduler.Service{StateRepo: repo},
		SchedulerStateRepo: repo,
		Clock:              fakeRefreshClock{now: now},
	}

	result, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(result.Created) != 1 || result.Created[0] != "new-artist" {
		t.Fatalf("unexpected created list: %#v", result.Created)
	}
	if len(result.Preserved) != 1 || result.Preserved[0] != "existing-artist" {
		t.Fatalf("unexpected preserved list: %#v", result.Preserved)
	}
	if len(result.Orphaned) != 1 || result.Orphaned[0] != "orphaned-artist" {
		t.Fatalf("unexpected orphaned list: %#v", result.Orphaned)
	}
	if _, ok := repo.saved["new-artist"]; !ok {
		t.Fatal("expected scheduler state to be created for new artist")
	}
}

func TestHandleSkipsDisabledSchedulersWithoutExistingState(t *testing.T) {
	t.Parallel()

	repo := &fakeSchedulerRepo{}
	handler := Handler{
		Registry: fakeRefreshRegistry{
			items: []ports.RegisteredGenerator{
				{Config: ports.GeneratorConfig{ID: "disabled-artist", SchedulerEnabled: false}},
			},
		},
		Scheduler:          scheduler.Service{StateRepo: repo},
		SchedulerStateRepo: repo,
		Clock:              fakeRefreshClock{now: time.Now().UTC()},
	}

	result, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(result.Created) != 0 || len(repo.saved) != 0 {
		t.Fatalf("expected disabled scheduler to skip creation: %#v %#v", result, repo.saved)
	}
}

type fakeRefreshRegistry struct {
	items []ports.RegisteredGenerator
}

func (f fakeRefreshRegistry) GetByID(string) (ports.RegisteredGenerator, bool) {
	return ports.RegisteredGenerator{}, false
}

func (f fakeRefreshRegistry) GetByType(episode.OutputType) (ports.RegisteredGenerator, bool) {
	return ports.RegisteredGenerator{}, false
}

func (f fakeRefreshRegistry) List() []ports.RegisteredGenerator { return f.items }

type fakeSchedulerRepo struct {
	states map[string]scheduling.State
	saved  map[string]scheduling.State
	list   []string
}

func (f *fakeSchedulerRepo) Load(_ context.Context, generatorID string) (scheduling.State, error) {
	if state, ok := f.states[generatorID]; ok {
		return state, nil
	}
	return scheduling.State{}, nil
}

func (f *fakeSchedulerRepo) Save(_ context.Context, generatorID string, state scheduling.State) error {
	if f.saved == nil {
		f.saved = map[string]scheduling.State{}
	}
	f.saved[generatorID] = state
	return nil
}

func (f *fakeSchedulerRepo) ListGeneratorIDs(_ context.Context) ([]string, error) {
	return append([]string(nil), f.list...), nil
}

type fakeRefreshClock struct {
	now time.Time
}

func (f fakeRefreshClock) Now() time.Time { return f.now }
