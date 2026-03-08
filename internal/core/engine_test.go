package core

import (
	"strings"
	"testing"

	"loreforge/internal/config"
)

func TestNewRejectsUnsupportedProviderDriver(t *testing.T) {
	t.Parallel()

	cfg := testConfig(t)
	cfg.Artists[0].Provider.Driver = "openai"

	_, err := New(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported provider driver")
	}
	if !strings.Contains(err.Error(), "unsupported text provider driver: openai") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewBuildsMockArtistRuntime(t *testing.T) {
	t.Parallel()

	cfg := testConfig(t)

	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	rt, ok := eng.artists["text-artist"]
	if !ok {
		t.Fatal("expected text-artist runtime to be registered")
	}
	if rt.artist == nil {
		t.Fatal("expected artist implementation to be initialized")
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()

	return config.Config{
		Artists: []config.ArtistConfig{
			{
				ID:   "text-artist",
				Type: "text",
				Provider: config.ProviderDriver{
					Driver: "mock",
					Model:  "mock-text-v1",
				},
				Scheduler: config.SchedulerConfig{
					Enabled:       true,
					Mode:          "fixed_interval",
					FixedInterval: "1h",
					Timezone:      "UTC",
				},
			},
		},
		Memory: config.MemoryConfig{
			DSN: t.TempDir() + "/universe.db",
		},
	}
}
