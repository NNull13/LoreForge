package scheduler

import (
	"context"
	"testing"
	"time"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/scheduling"
)

func TestNextRunUsesPersistedStateWithoutSaving(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	repo := &fakeSchedulerStateRepo{
		states: map[string]scheduling.State{
			"artist": {NextRunAt: now.Add(30 * time.Minute)},
		},
	}
	service := Service{StateRepo: repo}
	def := fixedIntervalGenerator("artist", 2*time.Hour)

	next, err := service.NextRun(context.Background(), def, now)
	if err != nil {
		t.Fatalf("NextRun returned error: %v", err)
	}
	if want := now.Add(30 * time.Minute); !next.Equal(want) {
		t.Fatalf("next run = %s, want %s", next, want)
	}
	if repo.saveCalls != 0 {
		t.Fatalf("save calls = %d, want 0", repo.saveCalls)
	}
}

func TestNextRunBootstrapsMissingStateAndPersists(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	repo := &fakeSchedulerStateRepo{}
	service := Service{StateRepo: repo}
	def := fixedIntervalGenerator("artist", 2*time.Hour)

	next, err := service.NextRun(context.Background(), def, now)
	if err != nil {
		t.Fatalf("NextRun returned error: %v", err)
	}
	if want := now.Add(2 * time.Hour); !next.Equal(want) {
		t.Fatalf("next run = %s, want %s", next, want)
	}
	if repo.saveCalls != 1 {
		t.Fatalf("save calls = %d, want 1", repo.saveCalls)
	}
	if got := repo.saved["artist"]; got.NextRunAt.IsZero() {
		t.Fatal("expected bootstrapped state to be saved")
	}
}

func TestAdvanceAfterRunPersistsLastRunAndNextRun(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	repo := &fakeSchedulerStateRepo{}
	service := Service{StateRepo: repo}
	def := fixedIntervalGenerator("artist", time.Hour)

	if err := service.AdvanceAfterRun(context.Background(), def, now); err != nil {
		t.Fatalf("AdvanceAfterRun returned error: %v", err)
	}
	state := repo.saved["artist"]
	if state.LastRunAt == nil || !state.LastRunAt.Equal(now) {
		t.Fatalf("last run = %v, want %s", state.LastRunAt, now)
	}
	if want := now.Add(time.Hour); !state.NextRunAt.Equal(want) {
		t.Fatalf("next run = %s, want %s", state.NextRunAt, want)
	}
}

func fixedIntervalGenerator(id string, interval time.Duration) ports.RegisteredGenerator {
	return ports.RegisteredGenerator{
		Config: ports.GeneratorConfig{
			ID: id,
			Scheduler: scheduling.Config{
				Mode:          scheduling.ModeFixedInterval,
				FixedInterval: interval,
				Timezone:      "UTC",
			},
		},
	}
}

type fakeSchedulerStateRepo struct {
	states    map[string]scheduling.State
	saved     map[string]scheduling.State
	saveCalls int
}

func (f *fakeSchedulerStateRepo) Load(_ context.Context, generatorID string) (scheduling.State, error) {
	if state, ok := f.states[generatorID]; ok {
		return state, nil
	}
	return scheduling.State{}, nil
}

func (f *fakeSchedulerStateRepo) Save(_ context.Context, generatorID string, state scheduling.State) error {
	if f.saved == nil {
		f.saved = map[string]scheduling.State{}
	}
	f.saveCalls++
	f.saved[generatorID] = state
	return nil
}

func (f *fakeSchedulerStateRepo) ListGeneratorIDs(_ context.Context) ([]string, error) {
	return nil, nil
}
