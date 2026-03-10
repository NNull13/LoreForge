package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadResolvesRelativePathsAndInheritsScheduler(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "universe"), 0o755); err != nil {
		t.Fatalf("mkdir universe: %v", err)
	}
	configPath := filepath.Join(root, "config.yaml")
	content := `universe:
  path: ./universe
scheduler:
  enabled: false
  mode: fixed_interval
  fixed_interval: 1h
  seed: 7
  timezone: UTC
providers:
  image:
    driver: mock
    model: mock-image-v1
channels:
  filesystem:
    enabled: true
    output_dir: ./out
memory:
  dsn: ./data/app.db
artists:
  - id: image-artist
    profile_id: artist_profile
    type: image
    provider:
      driver: mock
      model: mock-image-v1
    scheduler:
      enabled: true
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got, want := cfg.Universe.Path, filepath.Join(root, "universe"); got != want {
		t.Fatalf("universe path = %q, want %q", got, want)
	}
	if got, want := cfg.Memory.DSN, filepath.Join(root, "data", "app.db"); got != want {
		t.Fatalf("memory dsn = %q, want %q", got, want)
	}
	if got, want := cfg.Channels.Filesystem.OutputDir, filepath.Join(root, "out"); got != want {
		t.Fatalf("output dir = %q, want %q", got, want)
	}
	if cfg.Artists[0].Scheduler.Enabled == nil || !*cfg.Artists[0].Scheduler.Enabled {
		t.Fatal("expected artist scheduler to override root disabled scheduler")
	}
	if got := cfg.Artists[0].Scheduler.FixedInterval; got != "1h" {
		t.Fatalf("artist fixed interval = %q, want 1h", got)
	}
	if got := cfg.Artists[0].Scheduler.Seed; got != 7 {
		t.Fatalf("artist seed = %d, want 7", got)
	}
}

func TestValidateRejectsInvalidArtistIdentifiers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		id        string
		profileID string
	}{
		{name: "path traversal id", id: "../image-artist", profileID: "artist_profile"},
		{name: "slash id", id: "image/artist", profileID: "artist_profile"},
		{name: "space profile", id: "image-artist", profileID: "artist profile"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			cfg := validConfig(root)
			cfg.Artists[0].ID = tt.id
			cfg.Artists[0].ProfileID = tt.profileID

			if err := cfg.Validate(root); err == nil {
				t.Fatal("expected invalid identifier error")
			}
		})
	}
}

func TestValidateRequiresExplicitArtists(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := Config{
		Universe: UniverseConfig{Path: filepath.Join(root, "universe")},
		Scheduler: SchedulerConfig{
			Mode:          "fixed_interval",
			FixedInterval: "1h",
			Seed:          11,
			Timezone:      "UTC",
		},
		Providers: ProvidersConfig{
			Text:  ProviderDriver{Driver: "mock", Model: "mock-text-v1"},
			Image: ProviderDriver{Driver: "mock", Model: "mock-image-v1"},
		},
		Channels: ChannelsConfig{
			Filesystem: FilesystemChannelConfig{Enabled: true, OutputDir: "./out"},
		},
	}
	if err := os.MkdirAll(cfg.Universe.Path, 0o755); err != nil {
		t.Fatalf("mkdir universe: %v", err)
	}

	if err := cfg.Validate(root); err == nil || !strings.Contains(err.Error(), "at least one artist is required") {
		t.Fatalf("expected explicit artist validation error, got %v", err)
	}
}

func TestValidateLeavesSpecialMemoryDSNUntouched(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := validConfig(root)
	cfg.Memory.DSN = ":memory:"

	if err := cfg.Validate(root); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if cfg.Memory.DSN != ":memory:" {
		t.Fatalf("memory dsn = %q, want :memory:", cfg.Memory.DSN)
	}
}

func TestValidateProviderDriverAndHelpers(t *testing.T) {
	t.Parallel()

	validCases := []ArtistConfig{
		{ID: "text", Type: "short_story", Provider: ProviderDriver{Driver: "openai_text", Model: "gpt-5-mini", APIKeyEnv: "OPENAI_API_KEY", Timeout: "1m"}},
		{ID: "local", Type: "short_story", Provider: ProviderDriver{Driver: "lmstudio_text", Model: "qwen", Timeout: "1m"}},
		{ID: "image", Type: "image", Provider: ProviderDriver{Driver: "openai_image", Model: "gpt-image-1", APIKeyEnv: "OPENAI_API_KEY", Options: map[string]any{"response_format": "url"}}},
		{ID: "imagen", Type: "image", Provider: ProviderDriver{Driver: "vertex_imagen", Model: "imagen", ProjectIDEnv: "PROJECT_ID", Location: "us-central1", Timeout: "1m"}},
		{ID: "veo", Type: "video", Provider: ProviderDriver{Driver: "vertex_veo", Model: "veo", ProjectIDEnv: "PROJECT_ID", Location: "us-central1", BucketURI: "gs://bucket/out", PollInterval: "10s", Timeout: "5m"}},
		{ID: "runway", Type: "video", Provider: ProviderDriver{Driver: "runway_gen4", Model: "gen4", APIKeyEnv: "RUNWAY_API_KEY", Version: "2024-11-06", PollInterval: "10s", Timeout: "5m"}},
	}
	for _, artist := range validCases {
		if err := validateProviderDriver(artist); err != nil {
			t.Fatalf("validateProviderDriver(%s) returned error: %v", artist.Provider.Driver, err)
		}
	}

	if err := validateArtistOptions(ArtistConfig{ID: "bad", Options: map[string]any{"reference_mode": "bad", "max_continuity_items": 1, "max_asset_references": 1}}); err == nil {
		t.Fatal("expected invalid reference mode error")
	}
	if err := validateArtistOverrides(ArtistConfig{ID: "bad", Presentation: ArtistPresentationOverrideConfig{SignatureMode: "bad"}}); err == nil {
		t.Fatal("expected invalid presentation override error")
	}
	if got, ok := intFromAny(float64(3)); !ok || got != 3 {
		t.Fatalf("intFromAny unexpected result: %d %v", got, ok)
	}
	if got, ok := stringSliceFromAny([]any{"a", "b"}); !ok || len(got) != 2 {
		t.Fatalf("stringSliceFromAny unexpected result: %#v %v", got, ok)
	}
	if shouldResolveRelativePath("file:memory.db") {
		t.Fatal("expected file: DSN to skip relative resolution")
	}
}

func TestValidateProviderDriverRejectsInvalidConfigurations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		artist ArtistConfig
		want   string
	}{
		{
			name:   "unsupported driver",
			artist: ArtistConfig{ID: "bad", Type: "short_story", Provider: ProviderDriver{Driver: "unknown"}},
			want:   "unsupported",
		},
		{
			name:   "openai text wrong type",
			artist: ArtistConfig{ID: "bad", Type: "image", Provider: ProviderDriver{Driver: "openai_text", APIKeyEnv: "OPENAI_API_KEY", Model: "gpt"}},
			want:   "only supports textual types",
		},
		{
			name:   "openai text missing api key",
			artist: ArtistConfig{ID: "bad", Type: "short_story", Provider: ProviderDriver{Driver: "openai_text", Model: "gpt"}},
			want:   "api_key_env is required",
		},
		{
			name:   "openai text invalid timeout",
			artist: ArtistConfig{ID: "bad", Type: "short_story", Provider: ProviderDriver{Driver: "openai_text", APIKeyEnv: "OPENAI_API_KEY", Model: "gpt", Timeout: "forever"}},
			want:   "timeout invalid",
		},
		{
			name:   "lmstudio wrong type",
			artist: ArtistConfig{ID: "bad", Type: "video", Provider: ProviderDriver{Driver: "lmstudio_text", Model: "qwen"}},
			want:   "only supports textual types",
		},
		{
			name:   "lmstudio missing model",
			artist: ArtistConfig{ID: "bad", Type: "short_story", Provider: ProviderDriver{Driver: "lmstudio_text"}},
			want:   "model is required",
		},
		{
			name:   "lmstudio invalid timeout",
			artist: ArtistConfig{ID: "bad", Type: "short_story", Provider: ProviderDriver{Driver: "lmstudio_text", Model: "qwen", Timeout: "forever"}},
			want:   "timeout invalid",
		},
		{
			name:   "openai image missing api key",
			artist: ArtistConfig{ID: "bad", Type: "image", Provider: ProviderDriver{Driver: "openai_image", Model: "gpt-image-1"}},
			want:   "api_key_env is required",
		},
		{
			name:   "openai image invalid response format",
			artist: ArtistConfig{ID: "bad", Type: "image", Provider: ProviderDriver{Driver: "openai_image", APIKeyEnv: "OPENAI_API_KEY", Model: "gpt-image-1", Options: map[string]any{"response_format": "stream"}}},
			want:   "response_format must be b64_json or url",
		},
		{
			name:   "vertex imagen missing project",
			artist: ArtistConfig{ID: "bad", Type: "image", Provider: ProviderDriver{Driver: "vertex_imagen", Location: "us-central1", Model: "imagen"}},
			want:   "project_id_env is required",
		},
		{
			name:   "vertex imagen invalid timeout",
			artist: ArtistConfig{ID: "bad", Type: "image", Provider: ProviderDriver{Driver: "vertex_imagen", ProjectIDEnv: "PROJECT_ID", Location: "us-central1", Model: "imagen", Timeout: "later"}},
			want:   "timeout invalid",
		},
		{
			name:   "vertex veo missing bucket",
			artist: ArtistConfig{ID: "bad", Type: "video", Provider: ProviderDriver{Driver: "vertex_veo", ProjectIDEnv: "PROJECT_ID", Location: "us-central1", Model: "veo", PollInterval: "10s", Timeout: "5m"}},
			want:   "bucket_uri is required",
		},
		{
			name:   "vertex veo invalid poll",
			artist: ArtistConfig{ID: "bad", Type: "video", Provider: ProviderDriver{Driver: "vertex_veo", ProjectIDEnv: "PROJECT_ID", Location: "us-central1", BucketURI: "gs://bucket/out", Model: "veo", PollInterval: "soon", Timeout: "5m"}},
			want:   "poll_interval invalid",
		},
		{
			name:   "runway missing version",
			artist: ArtistConfig{ID: "bad", Type: "video", Provider: ProviderDriver{Driver: "runway_gen4", APIKeyEnv: "RUNWAY_API_KEY", Model: "gen4", PollInterval: "10s", Timeout: "5m"}},
			want:   "version is required",
		},
		{
			name:   "runway invalid timeout",
			artist: ArtistConfig{ID: "bad", Type: "video", Provider: ProviderDriver{Driver: "runway_gen4", APIKeyEnv: "RUNWAY_API_KEY", Model: "gen4", Version: "2024-11-06", PollInterval: "10s", Timeout: "never"}},
			want:   "timeout invalid",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateProviderDriver(tt.artist)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateProviderDriver(%s) err = %v, want substring %q", tt.artist.Provider.Driver, err, tt.want)
			}
		})
	}
}

func TestValidateArtistOptionAndHelperErrors(t *testing.T) {
	t.Parallel()

	if err := validateArtistOptions(ArtistConfig{
		ID: "bad",
		Options: map[string]any{
			"reference_mode":       "creative",
			"max_continuity_items": 1,
			"max_asset_references": 1,
		},
	}); err != nil {
		t.Fatalf("unexpected validation error, got %v", err)
	}

	if err := validateArtistOptions(ArtistConfig{
		ID: "bad",
		Options: map[string]any{
			"reference_mode":       "creative",
			"max_continuity_items": -1,
			"max_asset_references": 1,
		},
	}); err == nil || !strings.Contains(err.Error(), "max_continuity_items") {
		t.Fatalf("expected max_continuity_items validation error, got %v", err)
	}

	if err := validateArtistOptions(ArtistConfig{
		ID: "bad",
		Options: map[string]any{
			"reference_mode":       "assets_only",
			"max_continuity_items": 1,
			"max_asset_references": -1,
		},
	}); err == nil || !strings.Contains(err.Error(), "max_asset_references") {
		t.Fatalf("expected max_asset_references validation error, got %v", err)
	}

	if err := validateArtistOptions(ArtistConfig{
		ID: "bad",
		Options: map[string]any{
			"reference_mode":        "continuity_plus_assets",
			"max_continuity_items":  1,
			"max_asset_references":  1,
			"asset_usage_allowlist": []any{"invalid_usage"},
		},
	}); err == nil || !strings.Contains(err.Error(), "invalid usage") {
		t.Fatalf("expected asset allowlist validation error, got %v", err)
	}

	if got, ok := intFromAny(int64(4)); !ok || got != 4 {
		t.Fatalf("intFromAny(int64) = %d %v, want 4 true", got, ok)
	}
	if _, ok := intFromAny("4"); ok {
		t.Fatal("expected intFromAny to reject strings")
	}
	if _, ok := stringSliceFromAny([]any{"ok", 3}); ok {
		t.Fatal("expected stringSliceFromAny to reject mixed slices")
	}
}

func TestValidateSchedulerHelpersAndDefaults(t *testing.T) {
	t.Parallel()

	if err := validateScheduler(SchedulerConfig{Enabled: boolPtr(false), Mode: "fixed_interval"}, false); err != nil {
		t.Fatalf("disabled scheduler should skip validation, got %v", err)
	}
	if err := validateScheduler(SchedulerConfig{Mode: "fixed_interval", FixedInterval: "nope", Timezone: "UTC"}, true); err == nil {
		t.Fatal("expected invalid fixed interval")
	}
	if err := validateScheduler(SchedulerConfig{Mode: "random_window", MinInterval: "1h", MaxInterval: "bad", Timezone: "UTC"}, true); err == nil {
		t.Fatal("expected invalid random window")
	}
	if err := validateScheduler(SchedulerConfig{Mode: "weird", Timezone: "UTC"}, true); err == nil {
		t.Fatal("expected invalid scheduler mode")
	}
	if err := validateScheduler(SchedulerConfig{Mode: "fixed_interval", FixedInterval: "1h", Timezone: "Mars/Olympus"}, true); err == nil {
		t.Fatal("expected invalid scheduler timezone")
	}

	if err := validateIdentifier("id", ""); err == nil {
		t.Fatal("expected empty identifier error")
	}
	if err := validateIdentifier("id", "bad.value"); err == nil {
		t.Fatal("expected invalid character identifier error")
	}

	merged := mergeSchedulerConfig(
		SchedulerConfig{Enabled: boolPtr(true), Mode: "fixed_interval", FixedInterval: "1h", Seed: 1, Timezone: "UTC"},
		SchedulerConfig{Enabled: boolPtr(false), MinInterval: "2h", Seed: 5},
	)
	if merged.Enabled == nil || *merged.Enabled {
		t.Fatalf("expected merged enabled=false, got %#v", merged.Enabled)
	}
	if merged.FixedInterval != "1h" || merged.MinInterval != "2h" || merged.Seed != 5 {
		t.Fatalf("unexpected merged scheduler: %#v", merged)
	}

	cfg := Config{}
	cfg.applyProviderDefaults()
	if cfg.Providers.Text.Driver != "mock" || cfg.Providers.Video.Driver != "mock" || cfg.Providers.Image.Driver != "mock" {
		t.Fatalf("expected mock defaults, got %#v", cfg.Providers)
	}
	if cfg.Providers.Text.Timeout != "2m" || cfg.Providers.Video.Timeout != "10m" || cfg.Providers.Image.Timeout != "2m" {
		t.Fatalf("unexpected timeout defaults: %#v", cfg.Providers)
	}
	if cfg.Providers.Video.Version != "2024-11-06" || cfg.Providers.Image.Location != "us-central1" {
		t.Fatalf("unexpected provider defaults: %#v", cfg.Providers)
	}

	if got := resolveConfigPath("/tmp/config", "./memory.db"); got != filepath.Clean("/tmp/config/memory.db") {
		t.Fatalf("resolveConfigPath returned %q", got)
	}
	if !shouldResolveRelativePath("./memory.db") {
		t.Fatal("expected regular relative path to resolve")
	}
	if shouldResolveRelativePath("https://example.com/file") {
		t.Fatal("expected URL to skip relative resolution")
	}
}

func TestApplyArtistDefaultsSetsProviderAndOptions(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Scheduler: SchedulerConfig{
			Enabled:       boolPtr(true),
			Mode:          "fixed_interval",
			FixedInterval: time.Hour.String(),
			Seed:          99,
			Timezone:      "UTC",
		},
		Providers: ProvidersConfig{
			Text:  ProviderDriver{Driver: "mock", Model: "mock-text-v1"},
			Image: ProviderDriver{Driver: "mock", Model: "mock-image-v1"},
		},
		Channels: ChannelsConfig{
			Filesystem: FilesystemChannelConfig{Enabled: true, OutputDir: "./out"},
		},
	}

	artist := ArtistConfig{Type: "short_story"}
	cfg.applyArtistDefaults(&artist)

	if artist.ID != "short_story-artist" || artist.ProfileID != "short_story-artist" {
		t.Fatalf("unexpected generated identifiers: %#v", artist)
	}
	if artist.Provider.Driver != "mock" || artist.Provider.Model != "mock-text-v1" {
		t.Fatalf("unexpected provider defaults: %#v", artist.Provider)
	}
	if artist.Enabled == nil || !*artist.Enabled {
		t.Fatalf("expected artist enabled by default, got %#v", artist.Enabled)
	}
	if artist.Options["reference_mode"] != "continuity_only" {
		t.Fatalf("unexpected options defaults: %#v", artist.Options)
	}
	if got := artist.Publish; len(got) != 1 || got[0].Channel != "filesystem" {
		t.Fatalf("unexpected publish targets: %#v", got)
	}
	if artist.Scheduler.Enabled == nil || !*artist.Scheduler.Enabled || artist.Scheduler.Seed != 99 {
		t.Fatalf("unexpected scheduler defaults: %#v", artist.Scheduler)
	}
}

func TestValidatePublishTargetsAndTwitterAccounts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := validConfig(root)
	cfg.Channels.Twitter = TwitterChannelConfig{
		Enabled:        true,
		DefaultAccount: "base",
		Accounts: map[string]TwitterAccountConfig{
			"base": {DryRun: true},
		},
	}
	cfg.Artists[0].Publish = []ArtistPublishTargetConfig{
		{Channel: "filesystem"},
		{Channel: "twitter", Account: "base"},
	}

	if err := cfg.Validate(root); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if got := cfg.Artists[0].Publish; len(got) != 2 || got[1].Account != "base" {
		t.Fatalf("unexpected publish config: %#v", got)
	}
	if cfg.Channels.Twitter.Accounts["base"].BaseURL != "https://api.twitter.com" {
		t.Fatalf("expected twitter account defaults, got %#v", cfg.Channels.Twitter.Accounts["base"])
	}
}

func TestValidateSupportsLegacyPublishTargetsAndTwitterConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := validConfig(root)
	cfg.Channels.Twitter = TwitterChannelConfig{
		Enabled:        true,
		DryRun:         true,
		BearerTokenEnv: "TWITTER_BEARER_TOKEN",
		BaseURL:        "https://api.twitter.test",
	}
	cfg.Artists[0].Publish = nil
	cfg.Artists[0].PublishTargets = []string{"twitter"}

	if err := cfg.Validate(root); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if cfg.Channels.Twitter.DefaultAccount != "base" {
		t.Fatalf("expected legacy twitter config to synthesize base account, got %#v", cfg.Channels.Twitter)
	}
	if got := cfg.Artists[0].Publish; len(got) != 1 || got[0].Channel != "twitter" || got[0].Account != "base" {
		t.Fatalf("expected legacy publish targets to normalize, got %#v", got)
	}
	if cfg.Channels.Twitter.Accounts["base"].BaseURL != "https://api.twitter.test" {
		t.Fatalf("expected synthesized base account, got %#v", cfg.Channels.Twitter.Accounts["base"])
	}
}

func TestValidateRejectsInvalidPublishAccountConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	base := validConfig(root)
	base.Channels.Twitter = TwitterChannelConfig{
		Enabled:        true,
		DefaultAccount: "base",
		Accounts: map[string]TwitterAccountConfig{
			"base": {DryRun: true},
		},
	}

	cases := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{
			name: "unknown twitter account",
			mutate: func(cfg *Config) {
				cfg.Artists[0].Publish = []ArtistPublishTargetConfig{{Channel: "twitter", Account: "missing"}}
			},
			want: "unknown twitter account",
		},
		{
			name: "account on filesystem",
			mutate: func(cfg *Config) {
				cfg.Artists[0].Publish = []ArtistPublishTargetConfig{{Channel: "filesystem", Account: "base"}}
			},
			want: "not supported for channel filesystem",
		},
		{
			name: "duplicate publish channel",
			mutate: func(cfg *Config) {
				cfg.Artists[0].Publish = []ArtistPublishTargetConfig{{Channel: "twitter"}, {Channel: "twitter", Account: "base"}}
			},
			want: "duplicate publish channel",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			cfg.Artists = append([]ArtistConfig(nil), base.Artists...)
			cfg.Artists[0] = base.Artists[0]
			tt.mutate(&cfg)
			if err := cfg.Validate(root); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate err = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func validConfig(root string) Config {
	_ = os.MkdirAll(filepath.Join(root, "universe"), 0o755)
	return Config{
		Universe: UniverseConfig{Path: filepath.Join(root, "universe")},
		Scheduler: SchedulerConfig{
			Mode:          "fixed_interval",
			FixedInterval: "1h",
			Seed:          42,
			Timezone:      "UTC",
		},
		Providers: ProvidersConfig{
			Image: ProviderDriver{Driver: "mock", Model: "mock-image-v1"},
		},
		Channels: ChannelsConfig{
			Filesystem: FilesystemChannelConfig{Enabled: true, OutputDir: "./out"},
		},
		Artists: []ArtistConfig{
			{
				ID:        "image-artist",
				ProfileID: "artist_profile",
				Type:      "image",
				Provider:  ProviderDriver{Driver: "mock", Model: "mock-image-v1"},
			},
		},
	}
}
