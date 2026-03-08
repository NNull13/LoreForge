package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"loreforge/internal/adapters/repositories/episodestore"
	"loreforge/internal/adapters/repositories/schedulerstatefs"
	"loreforge/internal/config"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/publication"
	"loreforge/internal/domain/scheduling"
	"loreforge/internal/domain/universe"
)

func TestToSchedulingConfigAndSchedulerEnabled(t *testing.T) {
	cfg, err := toSchedulingConfig(config.SchedulerConfig{
		Mode:          "fixed_interval",
		FixedInterval: "2h",
		Timezone:      "UTC",
		Seed:          9,
	})
	if err != nil {
		t.Fatalf("toSchedulingConfig returned error: %v", err)
	}
	if cfg.FixedInterval != 2*time.Hour || !isSchedulerEnabled(config.SchedulerConfig{}) {
		t.Fatalf("unexpected scheduling config: %#v", cfg)
	}
	if _, err := toSchedulingConfig(config.SchedulerConfig{Mode: "fixed_interval", FixedInterval: "bad", Timezone: "UTC"}); err == nil {
		t.Fatal("expected invalid duration error")
	}
}

func TestBuildGeneratorRegistryAndPublishers(t *testing.T) {
	cfg := config.Config{
		Text: config.TextGenerationConfig{
			Formats: map[string]config.TextFormatConfig{
				"short_story": {MinWords: 10, MaxWords: 100, RequireStructured: testBoolPtr(true), RequireEntityMatch: testBoolPtr(true), MaxOutputTokens: 100},
			},
		},
		Artists: []config.ArtistConfig{
			{
				ID:        "story-artist",
				ProfileID: "ash-chorister",
				Type:      "short_story",
				Provider:  config.ProviderDriver{Driver: "mock", Model: "mock-text-v1"},
				Scheduler: config.SchedulerConfig{Enabled: testBoolPtr(true), Mode: "fixed_interval", FixedInterval: "1h", Timezone: "UTC"},
			},
			{
				ID:        "image-artist",
				ProfileID: "signal-cartographer",
				Type:      "image",
				Provider:  config.ProviderDriver{Driver: "mock", Model: "mock-image-v1"},
				Scheduler: config.SchedulerConfig{Enabled: testBoolPtr(false), Mode: "fixed_interval", FixedInterval: "1h", Timezone: "UTC"},
			},
		},
		Channels: config.ChannelsConfig{
			Filesystem: config.FilesystemChannelConfig{Enabled: true, OutputDir: t.TempDir()},
			Twitter:    config.TwitterChannelConfig{Enabled: true, DryRun: true},
		},
	}
	u := universe.Universe{
		Artists: map[string]universe.Artist{
			"ash-chorister":       {ID: "ash-chorister", Name: "Ash", Role: "chronicler", Summary: "summary", Prompting: universe.ArtistPrompting{SystemIdentity: "id"}},
			"signal-cartographer": {ID: "signal-cartographer", Name: "Signal", Role: "editor", Summary: "summary", Prompting: universe.ArtistPrompting{SystemIdentity: "id"}},
		},
	}

	registry, err := buildGeneratorRegistry(cfg, u)
	if err != nil {
		t.Fatalf("buildGeneratorRegistry returned error: %v", err)
	}
	if len(registry.List()) != 2 {
		t.Fatalf("unexpected generator count: %d", len(registry.List()))
	}
	textDef, ok := registry.GetByID("story-artist")
	if !ok || textDef.Config.Type != episode.OutputTypeShortStory || !textDef.Config.SchedulerEnabled {
		t.Fatalf("unexpected text generator: %#v", textDef)
	}
	imageDef, ok := registry.GetByID("image-artist")
	if !ok || imageDef.Config.SchedulerEnabled {
		t.Fatalf("unexpected image generator: %#v", imageDef)
	}

	publishers := buildPublisherRegistry(cfg)
	if _, ok := publishers.Get(publication.ChannelFilesystem); !ok {
		t.Fatal("expected filesystem publisher")
	}
	if _, ok := publishers.Get(publication.ChannelTwitter); !ok {
		t.Fatal("expected twitter publisher")
	}
}

func TestHelperMapsAndProductionEnv(t *testing.T) {
	if !isProductionEnv("production") || isProductionEnv("dev") {
		t.Fatal("unexpected production env detection")
	}
	if got := promptOverridesMap(config.ArtistPromptOverrideConfig{Forbidden: []string{"slang"}}); got["forbidden"] == nil {
		t.Fatalf("unexpected prompt overrides: %#v", got)
	}
	if got := presentationOverridesMap(config.ArtistPresentationOverrideConfig{Enabled: testBoolPtr(true), AllowedChannels: []string{"filesystem"}}); got["enabled"] != true {
		t.Fatalf("unexpected presentation overrides: %#v", got)
	}
	if got := providerConfigMap(config.ProviderDriver{Driver: "mock", Model: "m"}); got["driver"] != "mock" {
		t.Fatalf("unexpected provider config map: %#v", got)
	}
}

func testBoolPtr(v bool) *bool { return &v }

func TestCommandsWorkAgainstTempUniverse(t *testing.T) {
	cfgPath, cfg := writeCLIConfig(t)

	captureStdout(t, usage)
	captureStdout(t, func() { validateCmd([]string{"--config", cfgPath}) })
	captureStdout(t, func() { universeCmd([]string{"lint", cfg.Universe.Path}) })
	captureStdout(t, func() { artistsCmd([]string{"list", "--config", cfgPath}) })
	captureStdout(t, func() { configCmd([]string{"refresh", "--config", cfgPath}) })
	captureStdout(t, func() { schedulerCmd([]string{"next-run", "--artist", "story-artist", "--config", cfgPath}) })
	captureStdout(t, func() { generateCmd([]string{"once", "--artist", "story-artist", "--config", cfgPath}) })

	episodeID := firstEpisodeID(t, episodestore.BaseDirFromDSN(cfg.Memory.DSN))
	captureStdout(t, func() { episodeCmd([]string{"show", episodeID, "--config", cfgPath}) })

	schedulerRepo := schedulerstatefs.Repository{BaseDir: episodestore.BaseDirFromDSN(cfg.Memory.DSN)}
	if err := schedulerRepo.Save(context.Background(), "story-artist", scheduling.State{NextRunAt: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatalf("scheduler save: %v", err)
	}
	captureStdout(t, func() { runCmd([]string{"--config", cfgPath}) })
}

func writeCLIConfig(t *testing.T) (string, config.Config) {
	t.Helper()

	root := t.TempDir()
	universePath := filepath.Join(root, "universe")
	writeUniverseFixture(t, universePath)

	cfgPath := filepath.Join(root, "config.yaml")
	dsn := filepath.Join(root, "data", "universe.db")
	outDir := filepath.Join(root, "out")
	content := "app:\n  name: loreforge\n  env: dev\n" +
		"universe:\n  path: " + universePath + "\n" +
		"scheduler:\n  mode: fixed_interval\n  fixed_interval: 1h\n  seed: 42\n  timezone: UTC\n" +
		"generation:\n  weights:\n    short_story: 100\n  max_retries: 1\n  recency_window: 5\n" +
		"providers:\n  text:\n    driver: mock\n    model: mock-text-v1\n" +
		"channels:\n  filesystem:\n    enabled: true\n    output_dir: " + outDir + "\n" +
		"memory:\n  dsn: " + dsn + "\n" +
		"artists:\n  - id: story-artist\n    profile_id: ash-chorister\n    type: short_story\n    provider:\n      driver: mock\n      model: mock-text-v1\n    publish_targets: [filesystem]\n    scheduler:\n      enabled: true\n      mode: fixed_interval\n      fixed_interval: 1h\n      timezone: UTC\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("config.Load returned error: %v", err)
	}
	return cfgPath, cfg
}

func writeUniverseFixture(t *testing.T, root string) {
	t.Helper()

	writeFixtureFile(t, filepath.Join(root, "universe", "universe.md"), `---
id: ember-archive
type: universe
name: Ember Archive
---
Root body.`)
	writeFixtureFile(t, filepath.Join(root, "artists", "ash-chorister", "artist.md"), `---
id: ash-chorister
name: Ash Chorister
role: chronicler
summary: A solemn witness.
non_diegietic: true
mission:
  purpose: Preserve canon.
prompting:
  system_identity: You are The Ash Chorister.
presentation:
  enabled: true
  signature_mode: presentation_only
  framing_mode: none
---
Artist body.`)
	writeFixtureFile(t, filepath.Join(root, "worlds", "ember-city", "ember-city.md"), `---
id: ember-city
type: world
---
World body.`)
	writeFixtureFile(t, filepath.Join(root, "characters", "red-wanderer", "red-wanderer.md"), `---
id: red-wanderer
type: character
world_affinities: [ember-city]
---
Character body.`)
	writeFixtureFile(t, filepath.Join(root, "characters", "the-architect", "the-architect.md"), `---
id: the-architect
type: character
world_affinities: [ember-city]
---
Character body.`)
	writeFixtureFile(t, filepath.Join(root, "events", "gate-whisper", "gate-whisper.md"), `---
id: gate-whisper
type: event
compatible_worlds: [ember-city]
compatible_characters: [red-wanderer, the-architect]
---
Event body.`)
	writeFixtureFile(t, filepath.Join(root, "templates", "short-story", "short-story.md"), `---
id: short-story
type: template
output_type: short_story
---
MAX_CHARS: 5000`)
	writeFixtureFile(t, filepath.Join(root, "rules", "global-rules", "global-rules.md"), `---
id: global-rules
type: rule
target: all
---
Keep continuity.`)
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func firstEpisodeID(t *testing.T, baseDir string) string {
	t.Helper()

	pattern := filepath.Join(baseDir, "episodes", "*", "*", "*", "manifest.json")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		t.Fatalf("manifest glob failed: %v %v", err, matches)
	}
	content, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest episode.Manifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	return manifest.EpisodeID
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	content, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(content)
}
