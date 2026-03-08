package generateepisode

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/publication"
	"loreforge/internal/domain/scheduling"
	domainuniverse "loreforge/internal/domain/universe"
	"loreforge/internal/planner"
)

func TestHandleGeneratesAndPublishesEpisode(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	repo := &fakeEpisodeRepo{}
	schedulerRepo := &fakeSchedulerStateRepo{}
	publisher := fakePublisher{}
	handler := Handler{
		UniverseRepo:       fakeUniverseRepo{universe: testUniverse()},
		EpisodeRepo:        repo,
		SchedulerStateRepo: schedulerRepo,
		GeneratorRegistry: fakeGeneratorRegistry{
			items: []ports.RegisteredGenerator{
				{
					Generator: fakeGenerator{id: "short-story-artist", outputType: episode.OutputTypeShortStory, content: "Aria walks through the ash garden and hears Kade whisper from the gate while the city keeps their old oath alive."},
					Config: ports.GeneratorConfig{
						ID:             "short-story-artist",
						Type:           episode.OutputTypeShortStory,
						Style:          "lyrical-canon",
						PublishTargets: []publication.ChannelName{publication.ChannelFilesystem},
						Scheduler: scheduling.Config{
							Mode:          scheduling.ModeFixedInterval,
							FixedInterval: time.Hour,
							Timezone:      "UTC",
						},
					},
				},
			},
		},
		PublisherRegistry: fakePublisherRegistry{publisher: publisher},
		Clock:             fakeClock{now: now},
		IDGenerator:       fakeIDGen{id: "ep-123"},
		Hasher:            fakeHasher{value: "universe-hash"},
		Planner:           planner.New(planner.Config{Weights: map[string]int{"short_story": 100}, Seed: 1, RecencyWindow: 5}),
	}

	result, err := handler.Handle(context.Background(), Request{MaxRetries: 1, RecencyWindow: 5})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if result.Record.Manifest.EpisodeID != "ep-123" {
		t.Fatalf("unexpected episode id: %s", result.Record.Manifest.EpisodeID)
	}
	if !result.Record.Manifest.Published {
		t.Fatal("expected episode to be published")
	}
	if schedulerRepo.savedGeneratorID != "short-story-artist" {
		t.Fatalf("expected scheduler state to be saved for generator, got %s", schedulerRepo.savedGeneratorID)
	}
	if repo.saved.Manifest.EpisodeID != "ep-123" {
		t.Fatalf("expected record to be persisted")
	}
}

func TestHandleRetriesInvalidOutput(t *testing.T) {
	t.Parallel()

	handler := Handler{
		UniverseRepo:       fakeUniverseRepo{universe: testUniverse()},
		EpisodeRepo:        &fakeEpisodeRepo{},
		SchedulerStateRepo: &fakeSchedulerStateRepo{},
		GeneratorRegistry: fakeGeneratorRegistry{
			items: []ports.RegisteredGenerator{
				{
					Generator: &sequenceGenerator{
						id:         "short-story-artist",
						outputType: episode.OutputTypeShortStory,
						outputs: []episode.Output{
							{Content: "tiny", Prompt: "p1", Provider: "mock", Model: "mock-text"},
							{Content: "Aria finds Kade beside the ember gate and they trade the old oath in silence while the city listens for what comes next.", Prompt: "p2", Provider: "mock", Model: "mock-text"},
						},
					},
					Config: ports.GeneratorConfig{
						ID:        "short-story-artist",
						Type:      episode.OutputTypeShortStory,
						Scheduler: scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"},
					},
				},
			},
		},
		PublisherRegistry: fakePublisherRegistry{},
		Clock:             fakeClock{now: time.Now().UTC()},
		IDGenerator:       fakeIDGen{id: "ep-456"},
		Hasher:            fakeHasher{value: "hash"},
		Planner:           planner.New(planner.Config{Weights: map[string]int{"short_story": 100}, Seed: 2, RecencyWindow: 5}),
	}

	result, err := handler.Handle(context.Background(), Request{MaxRetries: 2, RecencyWindow: 5})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if result.Record.Manifest.RetryCount != 1 {
		t.Fatalf("expected one retry, got %d", result.Record.Manifest.RetryCount)
	}
}

type fakeUniverseRepo struct {
	universe domainuniverse.Universe
}

func (f fakeUniverseRepo) Load(_ context.Context) (domainuniverse.Universe, error) {
	return f.universe, nil
}

type fakeEpisodeRepo struct {
	saved      episode.Record
	references []episode.ContinuityReference
}

func (f *fakeEpisodeRepo) Save(_ context.Context, record episode.Record) (episode.StoredRecord, error) {
	f.saved = record
	return episode.StoredRecord{Path: "/tmp/" + record.Manifest.EpisodeID, Manifest: record.Manifest}, nil
}

func (f *fakeEpisodeRepo) FindByID(_ context.Context, _ string) (episode.StoredRecord, error) {
	return episode.StoredRecord{}, errors.New("not implemented")
}

func (f *fakeEpisodeRepo) RecentCombos(_ context.Context, _ int) ([]episode.Combo, error) {
	return nil, nil
}

func (f *fakeEpisodeRepo) RecentCombosByGenerator(_ context.Context, _ string, _ int) ([]episode.Combo, error) {
	return nil, nil
}

func (f *fakeEpisodeRepo) RecentReferencesByGenerator(_ context.Context, _ string, _ int) ([]episode.ContinuityReference, error) {
	return f.references, nil
}

type fakeSchedulerStateRepo struct {
	savedGeneratorID string
	state            scheduling.State
}

func (f *fakeSchedulerStateRepo) Load(_ context.Context, _ string) (scheduling.State, error) {
	return scheduling.State{}, nil
}

func (f *fakeSchedulerStateRepo) Save(_ context.Context, generatorID string, state scheduling.State) error {
	f.savedGeneratorID = generatorID
	f.state = state
	return nil
}

type fakeGeneratorRegistry struct {
	items []ports.RegisteredGenerator
}

func (f fakeGeneratorRegistry) GetByID(id string) (ports.RegisteredGenerator, bool) {
	for _, item := range f.items {
		if item.Config.ID == id {
			return item, true
		}
	}
	return ports.RegisteredGenerator{}, false
}

func (f fakeGeneratorRegistry) GetByType(outputType episode.OutputType) (ports.RegisteredGenerator, bool) {
	for _, item := range f.items {
		if item.Config.Type == outputType {
			return item, true
		}
	}
	return ports.RegisteredGenerator{}, false
}

func (f fakeGeneratorRegistry) List() []ports.RegisteredGenerator {
	return f.items
}

type fakeGenerator struct {
	id         string
	outputType episode.OutputType
	content    string
}

func (f fakeGenerator) ID() string { return f.id }

func (f fakeGenerator) Type() episode.OutputType { return f.outputType }

func (f fakeGenerator) Generate(context.Context, episode.Brief, episode.State) (episode.Output, error) {
	return episode.Output{
		Content:  f.content,
		Prompt:   "prompt",
		Provider: "mock-text",
		Model:    "mock-text-v1",
		Text: &episode.TextArtifact{
			Body:           f.content,
			WordCount:      len(strings.Fields(f.content)),
			CharacterCount: len([]rune(f.content)),
		},
	}, nil
}

type sequenceGenerator struct {
	id         string
	outputType episode.OutputType
	outputs    []episode.Output
	index      int
}

func (s *sequenceGenerator) ID() string { return s.id }

func (s *sequenceGenerator) Type() episode.OutputType { return s.outputType }

func (s *sequenceGenerator) Generate(context.Context, episode.Brief, episode.State) (episode.Output, error) {
	current := s.outputs[s.index]
	if s.index < len(s.outputs)-1 {
		s.index++
	}
	return current, nil
}

type fakePublisherRegistry struct {
	publisher ports.Publisher
}

func (f fakePublisherRegistry) Get(name publication.ChannelName) (ports.Publisher, bool) {
	if f.publisher == nil {
		return nil, false
	}
	if f.publisher.Name() != name {
		return nil, false
	}
	return f.publisher, true
}

type fakePublisher struct{}

func (fakePublisher) Name() publication.ChannelName { return publication.ChannelFilesystem }

func (fakePublisher) Publish(context.Context, publication.Item) (publication.Result, error) {
	return publication.Result{Channel: "filesystem", Success: true, ExternalID: "/tmp/out.txt"}, nil
}

type fakeClock struct {
	now time.Time
}

func (f fakeClock) Now() time.Time { return f.now }

type fakeIDGen struct {
	id string
}

func (f fakeIDGen) NewEpisodeID() string { return f.id }

type fakeHasher struct {
	value string
}

func (f fakeHasher) Hash(context.Context) (string, error) { return f.value, nil }

func testUniverse() domainuniverse.Universe {
	return domainuniverse.Universe{
		Universe: domainuniverse.Entity{
			ID:   "universe-1",
			Type: "universe",
			Data: map[string]any{"creator_presence": "steady"},
		},
		Worlds: map[string]domainuniverse.Entity{
			"ember-city": {ID: "ember-city", Type: "world", Data: map[string]any{}},
		},
		Characters: map[string]domainuniverse.Entity{
			"aria": {ID: "aria", Type: "character", Data: map[string]any{"world_affinities": []any{"ember-city"}}},
			"kade": {ID: "kade", Type: "character", Data: map[string]any{"world_affinities": []any{"ember-city"}}},
		},
		Events: map[string]domainuniverse.Entity{
			"gate-whisper": {ID: "gate-whisper", Type: "event", Data: map[string]any{"compatible_worlds": []any{"ember-city"}}},
		},
		Templates: map[string]domainuniverse.Entity{
			"short-story-template": {ID: "short-story-template", Type: "template", Body: "MAX_CHARS: 400", Data: map[string]any{"output_type": "short_story"}},
		},
		Rules: map[string]domainuniverse.Entity{},
	}
}
