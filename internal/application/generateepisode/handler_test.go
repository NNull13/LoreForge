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
						ID:               "short-story-artist",
						ProfileID:        "ash-chorister",
						Type:             episode.OutputTypeShortStory,
						Style:            "lyrical-canon",
						SchedulerEnabled: true,
						PublishTargets:   []publication.Target{{Channel: publication.ChannelFilesystem}},
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

	result, err := handler.Handle(context.Background(), Request{Generator: "short-story-artist", MaxRetries: 1, RecencyWindow: 5})
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
						ID:               "short-story-artist",
						ProfileID:        "ash-chorister",
						Type:             episode.OutputTypeShortStory,
						SchedulerEnabled: true,
						Scheduler:        scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"},
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

	result, err := handler.Handle(context.Background(), Request{Generator: "short-story-artist", MaxRetries: 2, RecencyWindow: 5})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if result.Record.Manifest.RetryCount != 1 {
		t.Fatalf("expected one retry, got %d", result.Record.Manifest.RetryCount)
	}
}

func TestHandleReturnsNoGeneratorsDue(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	handler := Handler{
		UniverseRepo:       fakeUniverseRepo{universe: testUniverse()},
		EpisodeRepo:        &fakeEpisodeRepo{},
		SchedulerStateRepo: &fakeSchedulerStateRepo{states: map[string]scheduling.State{"short-story-artist": {NextRunAt: now.Add(time.Hour)}}},
		GeneratorRegistry: fakeGeneratorRegistry{
			items: []ports.RegisteredGenerator{
				{
					Generator: fakeGenerator{id: "short-story-artist", outputType: episode.OutputTypeShortStory, content: validStory()},
					Config: ports.GeneratorConfig{
						ID:               "short-story-artist",
						ProfileID:        "ash-chorister",
						Type:             episode.OutputTypeShortStory,
						SchedulerEnabled: true,
						Scheduler:        scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"},
					},
				},
			},
		},
		PublisherRegistry: fakePublisherRegistry{},
		Clock:             fakeClock{now: now},
		IDGenerator:       fakeIDGen{id: "ep-due"},
		Hasher:            fakeHasher{value: "hash"},
		Planner:           planner.New(planner.Config{Weights: map[string]int{"short_story": 100}, Seed: 3, RecencyWindow: 5}),
	}

	_, err := handler.Handle(context.Background(), Request{MaxRetries: 1, RecencyWindow: 5})
	if !errors.Is(err, episode.ErrNoGeneratorsDue) {
		t.Fatalf("err = %v, want no generators due", err)
	}
}

func TestHandleReturnsSchedulerDisabledWhenAutoRunHasNoEnabledArtists(t *testing.T) {
	t.Parallel()

	handler := Handler{
		UniverseRepo:       fakeUniverseRepo{universe: testUniverse()},
		EpisodeRepo:        &fakeEpisodeRepo{},
		SchedulerStateRepo: &fakeSchedulerStateRepo{},
		GeneratorRegistry: fakeGeneratorRegistry{
			items: []ports.RegisteredGenerator{
				{
					Generator: fakeGenerator{id: "short-story-artist", outputType: episode.OutputTypeShortStory, content: validStory()},
					Config: ports.GeneratorConfig{
						ID:               "short-story-artist",
						ProfileID:        "ash-chorister",
						Type:             episode.OutputTypeShortStory,
						SchedulerEnabled: false,
					},
				},
			},
		},
		PublisherRegistry: fakePublisherRegistry{},
		Clock:             fakeClock{now: time.Now().UTC()},
		IDGenerator:       fakeIDGen{id: "ep-disabled"},
		Hasher:            fakeHasher{value: "hash"},
		Planner:           planner.New(planner.Config{Weights: map[string]int{"short_story": 100}, Seed: 4, RecencyWindow: 5}),
	}

	_, err := handler.Handle(context.Background(), Request{MaxRetries: 1, RecencyWindow: 5})
	if !errors.Is(err, episode.ErrSchedulerDisabled) {
		t.Fatalf("err = %v, want scheduler disabled", err)
	}
}

func TestHandleMarksPublishFailedAndPersistsRecord(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	repo := &fakeEpisodeRepo{}
	handler := Handler{
		UniverseRepo:       fakeUniverseRepo{universe: testUniverse()},
		EpisodeRepo:        repo,
		SchedulerStateRepo: &fakeSchedulerStateRepo{},
		GeneratorRegistry: fakeGeneratorRegistry{
			items: []ports.RegisteredGenerator{
				{
					Generator: fakeGenerator{id: "short-story-artist", outputType: episode.OutputTypeShortStory, content: validStory()},
					Config: ports.GeneratorConfig{
						ID:               "short-story-artist",
						ProfileID:        "ash-chorister",
						Type:             episode.OutputTypeShortStory,
						SchedulerEnabled: true,
						PublishTargets:   []publication.Target{{Channel: publication.ChannelFilesystem}},
						Scheduler:        scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"},
					},
				},
			},
		},
		PublisherRegistry: fakePublisherRegistry{
			publisher: fakeStaticPublisher{
				name: publication.ChannelFilesystem,
				err:  errors.New("disk full"),
			},
		},
		Clock:       fakeClock{now: now},
		IDGenerator: fakeIDGen{id: "ep-fail"},
		Hasher:      fakeHasher{value: "hash"},
		Planner:     planner.New(planner.Config{Weights: map[string]int{"short_story": 100}, Seed: 5, RecencyWindow: 5}),
	}

	result, err := handler.Handle(context.Background(), Request{Generator: "short-story-artist", MaxRetries: 1, RecencyWindow: 5})
	if !errors.Is(err, episode.ErrPublishFailed) {
		t.Fatalf("err = %v, want publish failed", err)
	}
	if result.Stored.Path == "" {
		t.Fatal("expected persisted result even on publish failure")
	}
	if repo.saved.Manifest.State != string(episode.StatusPublishFailed) {
		t.Fatalf("state = %s, want publish_failed", repo.saved.Manifest.State)
	}
}

func TestHandleAcceptsPartialPublishSuccess(t *testing.T) {
	t.Parallel()

	handler := Handler{
		UniverseRepo:       fakeUniverseRepo{universe: testUniverse()},
		EpisodeRepo:        &fakeEpisodeRepo{},
		SchedulerStateRepo: &fakeSchedulerStateRepo{},
		GeneratorRegistry: fakeGeneratorRegistry{
			items: []ports.RegisteredGenerator{
				{
					Generator: fakeGenerator{id: "short-story-artist", outputType: episode.OutputTypeShortStory, content: validStory()},
					Config: ports.GeneratorConfig{
						ID:               "short-story-artist",
						ProfileID:        "ash-chorister",
						Type:             episode.OutputTypeShortStory,
						SchedulerEnabled: true,
						PublishTargets: []publication.Target{
							{Channel: publication.ChannelFilesystem},
							{Channel: publication.ChannelTwitter, Account: "base"},
						},
						Scheduler: scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"},
					},
				},
			},
		},
		PublisherRegistry: fakePublisherRegistry{
			publishers: map[publication.ChannelName]ports.Publisher{
				publication.ChannelFilesystem: fakeStaticPublisher{
					name: publication.ChannelFilesystem,
					result: publication.Result{
						Channel:    "filesystem",
						Success:    true,
						ExternalID: "/tmp/out.txt",
					},
				},
				publication.ChannelTwitter: fakeStaticPublisher{
					name: publication.ChannelTwitter,
					err:  errors.New("twitter down"),
				},
			},
		},
		Clock:       fakeClock{now: time.Now().UTC()},
		IDGenerator: fakeIDGen{id: "ep-partial"},
		Hasher:      fakeHasher{value: "hash"},
		Planner:     planner.New(planner.Config{Weights: map[string]int{"short_story": 100}, Seed: 6, RecencyWindow: 5}),
	}

	result, err := handler.Handle(context.Background(), Request{Generator: "short-story-artist", MaxRetries: 1, RecencyWindow: 5})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !result.Record.Manifest.Published || result.Record.Manifest.State != string(episode.StatusPublished) {
		t.Fatalf("unexpected publish state: %#v", result.Record.Manifest)
	}
	if len(result.Record.Manifest.Channels) != 1 || result.Record.Manifest.Channels[0] != "filesystem" {
		t.Fatalf("unexpected published channels: %#v", result.Record.Manifest.Channels)
	}
}

func TestHandlePersistsResolvedPublishAccountMetadata(t *testing.T) {
	t.Parallel()

	repo := &fakeEpisodeRepo{}
	handler := Handler{
		UniverseRepo:       fakeUniverseRepo{universe: testUniverse()},
		EpisodeRepo:        repo,
		SchedulerStateRepo: &fakeSchedulerStateRepo{},
		GeneratorRegistry: fakeGeneratorRegistry{
			items: []ports.RegisteredGenerator{
				{
					Generator: fakeGenerator{id: "tweet-artist", outputType: episode.OutputTypeShortStory, content: validStory()},
					Config: ports.GeneratorConfig{
						ID:               "tweet-artist",
						ProfileID:        "ash-chorister",
						Type:             episode.OutputTypeShortStory,
						SchedulerEnabled: true,
						PublishTargets:   []publication.Target{{Channel: publication.ChannelTwitter, Account: "artist_a"}},
						Scheduler:        scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"},
					},
				},
			},
		},
		PublisherRegistry: fakePublisherRegistry{
			publisher: fakeStaticPublisher{
				name: publication.ChannelTwitter,
				result: publication.Result{
					Channel:    "twitter",
					Success:    true,
					ExternalID: "tweet-123",
					Metadata:   map[string]any{"account": "artist_a"},
				},
			},
		},
		Clock:       fakeClock{now: time.Now().UTC()},
		IDGenerator: fakeIDGen{id: "ep-twitter"},
		Hasher:      fakeHasher{value: "hash"},
		Planner:     planner.New(planner.Config{Weights: map[string]int{"short_story": 100}, Seed: 7, RecencyWindow: 5}),
	}

	result, err := handler.Handle(context.Background(), Request{Generator: "tweet-artist", MaxRetries: 1, RecencyWindow: 5})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	published, ok := result.Record.Publish["twitter"].(publication.Result)
	if !ok {
		t.Fatalf("expected twitter publish result, got %#v", result.Record.Publish["twitter"])
	}
	if published.Metadata["account"] != "artist_a" {
		t.Fatalf("expected account metadata to persist, got %#v", published.Metadata)
	}
	if saved, ok := repo.saved.Publish["twitter"].(publication.Result); !ok || saved.Metadata["account"] != "artist_a" {
		t.Fatalf("expected persisted twitter publish metadata, got %#v", repo.saved.Publish["twitter"])
	}
}

func TestResolveTemplateForTypeKeepsCompatibleFallback(t *testing.T) {
	t.Parallel()

	u := testUniverse()
	u.Templates["alt-template"] = domainuniverse.Entity{
		ID:   "alt-template",
		Type: "template",
		Data: map[string]any{"output_type": "short_story"},
	}
	if got := resolveTemplateForType(u, "short_story", "alt-template"); got != "alt-template" {
		t.Fatalf("resolveTemplateForType = %q, want alt-template", got)
	}
}

func TestBuildArtistLensPreservesNonDiegeticFlag(t *testing.T) {
	t.Parallel()

	profile := testUniverse().Artists["ash-chorister"]
	profile.NonDiegetic = false

	lens := buildArtistLens(profile, ports.GeneratorConfig{})
	if lens.NonDiegetic {
		t.Fatal("expected artist lens to preserve non-diegetic=false")
	}
}

func TestSanitizeSecretsRedactsNestedValues(t *testing.T) {
	t.Parallel()

	sanitized := sanitizeSecrets(map[string]any{
		"api_key": "secret",
		"nested": map[string]any{
			"authorization": "Bearer token",
			"value":         "keep",
		},
		"items": []any{
			map[string]any{"token": "secret"},
			"safe",
		},
	})
	if sanitized["api_key"] != "***" {
		t.Fatalf("expected top-level key redaction: %#v", sanitized)
	}
	nested := sanitized["nested"].(map[string]any)
	if nested["authorization"] != "***" || nested["value"] != "keep" {
		t.Fatalf("unexpected nested redaction: %#v", nested)
	}
	items := sanitized["items"].([]any)
	if items[0].(map[string]any)["token"] != "***" || items[1] != "safe" {
		t.Fatalf("unexpected array redaction: %#v", items)
	}
}

func TestBootstrapRunwayImageUsesFallbackGenerator(t *testing.T) {
	t.Parallel()

	handler := Handler{
		GeneratorRegistry: fakeGeneratorRegistry{
			items: []ports.RegisteredGenerator{
				{
					Generator: fakeAssetGenerator{id: "image-artist", outputType: episode.OutputTypeImage, assetPath: "/tmp/bootstrap.png"},
					Config: ports.GeneratorConfig{
						ID:   "image-artist",
						Type: episode.OutputTypeImage,
					},
				},
			},
		},
	}
	output, err := handler.bootstrapRunwayImage(context.Background(), Request{}, episode.Brief{}, episode.State{Metadata: map[string]any{}}, ports.RegisteredGenerator{
		Config: ports.GeneratorConfig{
			ID:             "video-artist",
			ProviderDriver: "runway_gen4",
			Options:        map[string]any{"bootstrap_image_generator": "image-artist"},
		},
	})
	if err != nil {
		t.Fatalf("bootstrapRunwayImage returned error: %v", err)
	}
	if output.AssetPath != "/tmp/bootstrap.png" {
		t.Fatalf("unexpected bootstrap asset: %#v", output)
	}
}

func TestBootstrapRunwayImageUsesPromptImageReference(t *testing.T) {
	t.Parallel()

	handler := Handler{}
	output, err := handler.bootstrapRunwayImage(context.Background(), Request{}, episode.Brief{
		Objective: "Create scene",
		VisualReferences: []episode.VisualReference{
			{Path: "/tmp/reference.png", MediaType: "image", ModelRole: "prompt_image"},
		},
	}, episode.State{Metadata: map[string]any{}}, ports.RegisteredGenerator{})
	if err != nil {
		t.Fatalf("bootstrapRunwayImage returned error: %v", err)
	}
	if output.AssetPath != "/tmp/reference.png" || output.Provider != "universe_asset" {
		t.Fatalf("unexpected prompt image output: %#v", output)
	}
}

func TestResolveGeneratorAcceptsTypeAlias(t *testing.T) {
	t.Parallel()

	handler := Handler{
		GeneratorRegistry: fakeGeneratorRegistry{
			items: []ports.RegisteredGenerator{
				{
					Generator: fakeGenerator{id: "story-artist", outputType: episode.OutputTypeShortStory, content: validStory()},
					Config: ports.GeneratorConfig{
						ID:   "story-artist",
						Type: episode.OutputTypeShortStory,
					},
				},
			},
		},
	}
	def, err := handler.resolveGenerator(context.Background(), "short_story")
	if err != nil {
		t.Fatalf("resolveGenerator returned error: %v", err)
	}
	if def.Config.ID != "story-artist" {
		t.Fatalf("unexpected generator: %#v", def)
	}
}

func TestApplyArtistOverridesAndHelpers(t *testing.T) {
	t.Parallel()

	lens := episode.ArtistLens{
		Presentation: episode.ArtistPresentationSnapshot{},
	}
	applyArtistOverrides(&lens, ports.GeneratorConfig{
		PromptOverrides: map[string]any{
			"extra_system_rules": []any{"Rule A"},
			"tonal_biases":       []any{"ritual"},
			"lexical_cues":       []any{"ember"},
			"forbidden":          []any{"slang"},
		},
		PresentationOverrides: map[string]any{
			"enabled":          true,
			"signature_mode":   "append",
			"signature_text":   "Filed by the archive.",
			"framing_mode":     "intro",
			"intro_template":   "Intro",
			"outro_template":   "Outro",
			"allowed_channels": []any{"filesystem"},
		},
	})
	if len(lens.PromptingRules) != 1 || lens.Presentation.SignatureMode != "append" || !lens.Presentation.Enabled {
		t.Fatalf("unexpected overridden lens: %#v", lens)
	}
	if got := toStringSlice([]any{"a", "b"}); len(got) != 2 {
		t.Fatalf("unexpected toStringSlice: %#v", got)
	}
	if got := firstNonEmpty("", "value"); got != "value" {
		t.Fatalf("firstNonEmpty = %q, want value", got)
	}
}

func TestPublishFailureErrorAndPublishedChannels(t *testing.T) {
	t.Parallel()

	results := map[string]any{
		"filesystem": publication.Result{Channel: "filesystem", Success: true},
		"twitter":    map[string]any{"success": false, "error": "timeout"},
	}
	channels := publishedChannels(results)
	if len(channels) != 1 || channels[0] != "filesystem" {
		t.Fatalf("unexpected published channels: %#v", channels)
	}
	if err := publishFailureError(results, []publication.Target{{Channel: publication.ChannelFilesystem}, {Channel: publication.ChannelTwitter, Account: "base"}}); err != nil {
		t.Fatalf("expected partial success to suppress publish failure error: %v", err)
	}
	if err := publishFailureError(map[string]any{"twitter": map[string]any{"error": "timeout"}}, []publication.Target{{Channel: publication.ChannelTwitter}}); !errors.Is(err, episode.ErrPublishFailed) {
		t.Fatalf("unexpected publish failure error: %v", err)
	}
}

func TestArtistVisualReferencesAndCombosToKeys(t *testing.T) {
	t.Parallel()

	refs := artistVisualReferences([]episode.VisualReference{
		{EntityType: "artist", EntityID: "ash-chorister", AssetID: "brand"},
		{EntityType: "character", EntityID: "aria", AssetID: "hero"},
	}, "ash-chorister")
	if len(refs) != 1 || refs[0].AssetID != "brand" {
		t.Fatalf("unexpected artist refs: %#v", refs)
	}
	keys := combosToKeys([]episode.Combo{{WorldID: "ember-city", CharacterIDs: []string{"kade", "aria"}, EventID: "gate-whisper"}})
	if len(keys) != 1 || keys[0] != "ember-city|aria,kade|gate-whisper" {
		t.Fatalf("unexpected combo keys: %#v", keys)
	}
}

func TestNextRunForGeneratorUsesSavedStateAndSchedulerFallback(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	def := ports.RegisteredGenerator{
		Config: ports.GeneratorConfig{
			ID:               "story-artist",
			SchedulerEnabled: true,
			Scheduler: scheduling.Config{
				Mode:          scheduling.ModeFixedInterval,
				FixedInterval: 2 * time.Hour,
				Timezone:      "UTC",
			},
		},
	}

	next, err := nextRunForGenerator(context.Background(), &fakeSchedulerStateRepo{
		states: map[string]scheduling.State{
			"story-artist": {NextRunAt: now.Add(30 * time.Minute)},
		},
	}, def, now)
	if err != nil {
		t.Fatalf("nextRunForGenerator with state returned error: %v", err)
	}
	if want := now.Add(30 * time.Minute); !next.Equal(want) {
		t.Fatalf("next run = %s, want %s", next, want)
	}

	next, err = nextRunForGenerator(context.Background(), &fakeSchedulerStateRepo{}, def, now)
	if err != nil {
		t.Fatalf("nextRunForGenerator fallback returned error: %v", err)
	}
	if want := now.Add(2 * time.Hour); !next.Equal(want) {
		t.Fatalf("next run = %s, want %s", next, want)
	}
}

func TestNextDueGeneratorAndPublishHelpers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	handler := Handler{
		GeneratorRegistry: fakeGeneratorRegistry{
			items: []ports.RegisteredGenerator{
				{Config: ports.GeneratorConfig{ID: "disabled", SchedulerEnabled: false}},
				{
					Config: ports.GeneratorConfig{
						ID:               "future",
						SchedulerEnabled: true,
						Scheduler:        scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"},
					},
				},
				{
					Config: ports.GeneratorConfig{
						ID:               "due",
						SchedulerEnabled: true,
						Scheduler:        scheduling.Config{Mode: scheduling.ModeFixedInterval, FixedInterval: time.Hour, Timezone: "UTC"},
					},
				},
			},
		},
		SchedulerStateRepo: &fakeSchedulerStateRepo{
			states: map[string]scheduling.State{
				"future": {NextRunAt: now.Add(time.Hour)},
				"due":    {NextRunAt: now.Add(-time.Minute)},
			},
		},
		PublisherRegistry: fakePublisherRegistry{},
		Clock:             fakeClock{now: now},
	}

	def, err := handler.nextDueGenerator(context.Background())
	if err != nil {
		t.Fatalf("nextDueGenerator returned error: %v", err)
	}
	if def.Config.ID != "due" {
		t.Fatalf("nextDueGenerator selected %q, want due", def.Config.ID)
	}

	noGenerators := Handler{
		GeneratorRegistry:  fakeGeneratorRegistry{},
		SchedulerStateRepo: &fakeSchedulerStateRepo{},
		Clock:              fakeClock{now: now},
	}
	if _, err := noGenerators.nextDueGenerator(context.Background()); !errors.Is(err, episode.ErrNoGeneratorsAvailable) {
		t.Fatalf("nextDueGenerator err = %v, want no generators available", err)
	}

	results, presentation := handler.publish(context.Background(), episode.Record{
		Manifest: episode.Manifest{
			EpisodeID:  "ep-1",
			ArtistID:   "due",
			ArtistType: "short_story",
			OutputType: "short_story",
			CreatedAt:  now,
		},
		OutputText: "Aria keeps the old oath alive in the ash city.",
	}, episode.ArtistLens{}, []publication.Target{{Channel: publication.ChannelFilesystem}})
	if results["filesystem"].(map[string]any)["error"] != "channel not configured" {
		t.Fatalf("unexpected publish result: %#v", results)
	}
	if presentation["filesystem"] == nil {
		t.Fatalf("expected presentation snapshot for missing channel: %#v", presentation)
	}
}

func TestTemplateUniverseDataAndSnapshotHelpers(t *testing.T) {
	t.Parallel()

	u := testUniverse()
	u.Templates["a-template"] = domainuniverse.Entity{ID: "a-template", Type: "template", Body: "alpha", Data: map[string]any{"output_type": "short_story"}}
	u.Templates["b-template"] = domainuniverse.Entity{ID: "b-template", Type: "template", Body: "beta", Data: map[string]any{"output_type": "short_story"}}

	if got := resolveTemplateForType(u, "short_story", "missing"); got != "a-template" {
		t.Fatalf("resolveTemplateForType sorted fallback = %q, want a-template", got)
	}
	if got := resolveTemplateForType(u, "video", "missing"); got != "" {
		t.Fatalf("resolveTemplateForType non-match = %q, want empty", got)
	}
	if templateMatchesType(domainuniverse.Entity{Data: map[string]any{"output_type": "image"}}, "video") {
		t.Fatal("expected templateMatchesType to reject mismatched type")
	}

	brief := enrichBriefWithUniverseData(episode.Brief{
		TemplateID:   "short-story-template",
		WorldID:      "ember-city",
		EventID:      "gate-whisper",
		CharacterIDs: []string{"aria"},
	}, u)
	if brief.TemplateBody == "" || brief.EventData == nil || brief.CharacterData["aria"] == nil {
		t.Fatalf("unexpected enriched brief: %#v", brief)
	}

	snapshot := artistSnapshot(u.Artists["ash-chorister"])
	if snapshot["id"] != "ash-chorister" || snapshot["non_diegietic"] != true {
		t.Fatalf("unexpected artist snapshot: %#v", snapshot)
	}
	promptSnapshot := artistPromptSnapshot(buildArtistLens(u.Artists["ash-chorister"], ports.GeneratorConfig{}))
	if promptSnapshot["id"] != "ash-chorister" || promptSnapshot["presentation"] == nil {
		t.Fatalf("unexpected prompt snapshot: %#v", promptSnapshot)
	}
	if cloneAnyMap(nil) != nil {
		t.Fatal("expected cloneAnyMap(nil) to return nil")
	}
	if !secretKey("Authorization") || secretKey("content") {
		t.Fatal("unexpected secretKey classification")
	}
}

func TestHandleWrapsUniverseLoadError(t *testing.T) {
	t.Parallel()

	handler := Handler{
		UniverseRepo:       fakeUniverseRepo{err: errors.New("bad universe")},
		EpisodeRepo:        &fakeEpisodeRepo{},
		SchedulerStateRepo: &fakeSchedulerStateRepo{},
		GeneratorRegistry:  fakeGeneratorRegistry{},
		PublisherRegistry:  fakePublisherRegistry{},
		Clock:              fakeClock{now: time.Now().UTC()},
		IDGenerator:        fakeIDGen{id: "ep-err"},
		Hasher:             fakeHasher{value: "hash"},
		Planner:            planner.New(planner.Config{Weights: map[string]int{"short_story": 100}, Seed: 9, RecencyWindow: 5}),
	}

	if _, err := handler.Handle(context.Background(), Request{Generator: "missing"}); !errors.Is(err, episode.ErrUniverseInvalid) {
		t.Fatalf("Handle err = %v, want universe invalid", err)
	}
}

type fakeUniverseRepo struct {
	universe domainuniverse.Universe
	err      error
}

func (f fakeUniverseRepo) Load(_ context.Context) (domainuniverse.Universe, error) {
	if f.err != nil {
		return domainuniverse.Universe{}, f.err
	}
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
	states           map[string]scheduling.State
}

func (f *fakeSchedulerStateRepo) Load(_ context.Context, generatorID string) (scheduling.State, error) {
	if f.states != nil {
		if state, ok := f.states[generatorID]; ok {
			return state, nil
		}
		return scheduling.State{}, nil
	}
	return f.state, nil
}

func (f *fakeSchedulerStateRepo) Save(_ context.Context, generatorID string, state scheduling.State) error {
	f.savedGeneratorID = generatorID
	f.state = state
	return nil
}

func (f *fakeSchedulerStateRepo) ListGeneratorIDs(_ context.Context) ([]string, error) {
	return nil, nil
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
	publisher  ports.Publisher
	publishers map[publication.ChannelName]ports.Publisher
}

func (f fakePublisherRegistry) Get(name publication.ChannelName) (ports.Publisher, bool) {
	if f.publishers != nil {
		publisher, ok := f.publishers[name]
		return publisher, ok
	}
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

type fakeStaticPublisher struct {
	name   publication.ChannelName
	result publication.Result
	err    error
}

func (f fakeStaticPublisher) Name() publication.ChannelName { return f.name }

func (f fakeStaticPublisher) Publish(context.Context, publication.Item) (publication.Result, error) {
	return f.result, f.err
}

type fakeAssetGenerator struct {
	id         string
	outputType episode.OutputType
	assetPath  string
}

func (f fakeAssetGenerator) ID() string { return f.id }

func (f fakeAssetGenerator) Type() episode.OutputType { return f.outputType }

func (f fakeAssetGenerator) Generate(context.Context, episode.Brief, episode.State) (episode.Output, error) {
	return episode.Output{
		AssetPath: f.assetPath,
		Prompt:    "prompt",
		Provider:  "mock-image",
		Model:     "mock-image-v1",
	}, nil
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
		Artists: map[string]domainuniverse.Artist{
			"ash-chorister": {
				ID:          "ash-chorister",
				Name:        "The Ash Chorister",
				Title:       "Chronicler of the Ember Archive",
				Role:        "chronicler",
				Summary:     "A solemn editorial witness.",
				Body:        "Records the universe with ritual gravity.",
				NonDiegetic: true,
				Voice: domainuniverse.ArtistVoice{
					Register:    "elevated",
					Cadence:     "ritual",
					Diction:     "ceremonial",
					Stance:      "observant",
					Perspective: "editorial",
					Intensity:   "medium",
				},
				Mission: domainuniverse.ArtistMission{
					Purpose:    "Preserve canon through reflective narration.",
					Priorities: []string{"clarity", "continuity"},
				},
				Prompting: domainuniverse.ArtistPrompting{
					SystemIdentity: "You are The Ash Chorister.",
					SystemRules:    []string{"Never contradict canon.", "Never appear as an in-world character."},
					TonalBiases:    []string{"ritual", "restrained"},
					LexicalCues:    []string{"ember", "oath"},
					Forbidden:      []string{"internet slang"},
				},
				Presentation: domainuniverse.ArtistPresentation{
					Enabled:         true,
					SignatureMode:   "presentation_only",
					SignatureText:   "Filed by The Ash Chorister.",
					FramingMode:     "intro_outro",
					IntroTemplate:   "From the Ember Archive:",
					OutroTemplate:   "Filed by The Ash Chorister.",
					AllowedChannels: []string{"filesystem"},
				},
			},
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

func validStory() string {
	return "Aria walks through the ash garden and hears Kade whisper from the gate while the city keeps their old oath alive."
}
