package listartists

import (
	"context"
	"testing"
	"time"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/scheduling"
	domainuniverse "loreforge/internal/domain/universe"
)

func TestHandleReturnsArtistProfilesAndNextRun(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	handler := Handler{
		Registry: fakeListRegistry{
			items: []ports.RegisteredGenerator{
				{
					Config: ports.GeneratorConfig{
						ID:             "short-story-artist",
						ProfileID:      "ash-chorister",
						Type:           episode.OutputTypeShortStory,
						ProviderDriver: "openai_text",
						ProviderModel:  "gpt-5-mini",
						PublishTargets: nil,
						Scheduler:      scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"},
					},
				},
			},
		},
		SchedulerStateRepo: fakeListSchedulerRepo{
			state: scheduling.State{NextRunAt: now.Add(3 * time.Hour)},
		},
		Clock: fakeListClock{now: now},
		Universe: domainuniverse.Universe{
			Artists: map[string]domainuniverse.Artist{
				"ash-chorister": {ID: "ash-chorister", Name: "The Ash Chorister"},
			},
		},
	}

	items, err := handler.Handle(context.Background())
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].ArtistName != "The Ash Chorister" {
		t.Fatalf("unexpected artist name: %s", items[0].ArtistName)
	}
	if !items[0].NextRun.Equal(now.Add(3 * time.Hour)) {
		t.Fatalf("unexpected next run: %s", items[0].NextRun)
	}
}

type fakeListRegistry struct {
	items []ports.RegisteredGenerator
}

func (f fakeListRegistry) GetByID(string) (ports.RegisteredGenerator, bool) {
	return ports.RegisteredGenerator{}, false
}

func (f fakeListRegistry) GetByType(episode.OutputType) (ports.RegisteredGenerator, bool) {
	return ports.RegisteredGenerator{}, false
}

func (f fakeListRegistry) List() []ports.RegisteredGenerator { return f.items }

type fakeListSchedulerRepo struct {
	state scheduling.State
}

func (f fakeListSchedulerRepo) Load(_ context.Context, _ string) (scheduling.State, error) {
	return f.state, nil
}

func (f fakeListSchedulerRepo) Save(_ context.Context, _ string, _ scheduling.State) error {
	return nil
}

func (f fakeListSchedulerRepo) ListGeneratorIDs(_ context.Context) ([]string, error) {
	return nil, nil
}

type fakeListClock struct {
	now time.Time
}

func (f fakeListClock) Now() time.Time { return f.now }
