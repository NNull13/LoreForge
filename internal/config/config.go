package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App        AppConfig            `yaml:"app"`
	Universe   UniverseConfig       `yaml:"universe"`
	Scheduler  SchedulerConfig      `yaml:"scheduler"`
	Generation GenerationConfig     `yaml:"generation"`
	Text       TextGenerationConfig `yaml:"text"`
	Providers  ProvidersConfig      `yaml:"providers"`
	Artists    []ArtistConfig       `yaml:"artists"`
	Channels   ChannelsConfig       `yaml:"channels"`
	Memory     MemoryConfig         `yaml:"memory"`
	Logging    LoggingConfig        `yaml:"logging"`
}

type AppConfig struct {
	Name string `yaml:"name"`
	Env  string `yaml:"env"`
}

type UniverseConfig struct {
	Path string `yaml:"path"`
}

type SchedulerConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Mode          string `yaml:"mode"`
	MinInterval   string `yaml:"min_interval"`
	MaxInterval   string `yaml:"max_interval"`
	FixedInterval string `yaml:"fixed_interval"`
	Seed          int64  `yaml:"seed"`
	Timezone      string `yaml:"timezone"`
}

type GenerationConfig struct {
	EnabledAgents []string       `yaml:"enabled_agents"`
	Weights       map[string]int `yaml:"weights"`
	MaxRetries    int            `yaml:"max_retries"`
	RecencyWindow int            `yaml:"recency_window"`
}

type TextFormatConfig struct {
	MinWords           int      `yaml:"min_words" json:"min_words"`
	MaxWords           int      `yaml:"max_words" json:"max_words"`
	MinParts           int      `yaml:"min_parts" json:"min_parts"`
	MaxParts           int      `yaml:"max_parts" json:"max_parts"`
	MaxCharsPerPart    int      `yaml:"max_chars_per_part" json:"max_chars_per_part"`
	RequireEntityMatch *bool    `yaml:"require_entity_match" json:"require_entity_match"`
	RequireStructured  *bool    `yaml:"require_structured" json:"require_structured"`
	Temperature        *float64 `yaml:"temperature" json:"temperature"`
	MaxOutputTokens    int      `yaml:"max_output_tokens" json:"max_output_tokens"`
	TargetParts        int      `yaml:"target_parts" json:"target_parts"`
	TargetLineCount    int      `yaml:"target_line_count" json:"target_line_count"`
	TargetSceneCount   int      `yaml:"target_scene_count" json:"target_scene_count"`
	TemplateStrictness string   `yaml:"template_strictness" json:"template_strictness"`
	TwitterPublishable *bool    `yaml:"twitter_publishable" json:"twitter_publishable"`
}

type TextGenerationConfig struct {
	Formats map[string]TextFormatConfig `yaml:"formats"`
}

type ProviderDriver struct {
	Driver       string         `yaml:"driver"`
	Model        string         `yaml:"model"`
	APIKeyEnv    string         `yaml:"api_key_env"`
	BaseURL      string         `yaml:"base_url"`
	ProjectIDEnv string         `yaml:"project_id_env"`
	Location     string         `yaml:"location"`
	BucketURI    string         `yaml:"bucket_uri"`
	PollInterval string         `yaml:"poll_interval"`
	Timeout      string         `yaml:"timeout"`
	Version      string         `yaml:"version"`
	Options      map[string]any `yaml:"options"`
}

type ProvidersConfig struct {
	Text  ProviderDriver `yaml:"text"`
	Video ProviderDriver `yaml:"video"`
	Image ProviderDriver `yaml:"image"`
}

type ArtistConfig struct {
	ID              string                           `yaml:"id"`
	ProfileID       string                           `yaml:"profile_id"`
	Type            string                           `yaml:"type"`
	Enabled         *bool                            `yaml:"enabled"`
	Style           string                           `yaml:"style"`
	Provider        ProviderDriver                   `yaml:"provider"`
	Options         map[string]any                   `yaml:"options"`
	PromptOverrides ArtistPromptOverrideConfig       `yaml:"prompt_overrides"`
	Presentation    ArtistPresentationOverrideConfig `yaml:"presentation"`
	PublishTargets  []string                         `yaml:"publish_targets"`
	Scheduler       SchedulerConfig                  `yaml:"scheduler"`
}

type ArtistPromptOverrideConfig struct {
	ExtraSystemRules []string `yaml:"extra_system_rules"`
	TonalBiases      []string `yaml:"tonal_biases"`
	LexicalCues      []string `yaml:"lexical_cues"`
	Forbidden        []string `yaml:"forbidden"`
}

type ArtistPresentationOverrideConfig struct {
	Enabled         *bool    `yaml:"enabled"`
	SignatureMode   string   `yaml:"signature_mode"`
	SignatureText   string   `yaml:"signature_text"`
	FramingMode     string   `yaml:"framing_mode"`
	IntroTemplate   string   `yaml:"intro_template"`
	OutroTemplate   string   `yaml:"outro_template"`
	AllowedChannels []string `yaml:"allowed_channels"`
}

type FilesystemChannelConfig struct {
	Enabled   bool   `yaml:"enabled"`
	OutputDir string `yaml:"output_dir"`
}

type TwitterChannelConfig struct {
	Enabled        bool   `yaml:"enabled"`
	DryRun         bool   `yaml:"dry_run"`
	BearerTokenEnv string `yaml:"bearer_token_env"`
	BaseURL        string `yaml:"base_url"`
}

type ChannelsConfig struct {
	Filesystem FilesystemChannelConfig `yaml:"filesystem"`
	Twitter    TwitterChannelConfig    `yaml:"twitter"`
}

type MemoryConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func Load(path string) (Config, error) {
	cfg := Config{
		Scheduler: SchedulerConfig{Enabled: true},
		Channels: ChannelsConfig{
			Filesystem: FilesystemChannelConfig{Enabled: true},
			Twitter:    TwitterChannelConfig{DryRun: true},
		},
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config yaml: %w", err)
	}

	if err := cfg.Validate(filepath.Dir(path)); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c *Config) Validate(configDir string) error {
	if c.Universe.Path == "" {
		return errors.New("universe.path is required")
	}
	if !filepath.IsAbs(c.Universe.Path) {
		candidate := filepath.Clean(filepath.Join(configDir, c.Universe.Path))
		if _, err := os.Stat(candidate); err == nil {
			c.Universe.Path = candidate
		} else {
			c.Universe.Path = filepath.Clean(c.Universe.Path)
		}
	}

	if c.Scheduler.Mode == "" {
		c.Scheduler.Mode = "random_window"
	}
	if c.Scheduler.Seed == 0 {
		c.Scheduler.Seed = 42
	}
	if c.Scheduler.Timezone == "" {
		c.Scheduler.Timezone = "UTC"
	}
	if err := validateScheduler(c.Scheduler, c.Scheduler.Enabled); err != nil {
		return err
	}

	if c.Generation.MaxRetries < 0 {
		return errors.New("generation.max_retries cannot be negative")
	}
	if c.Generation.MaxRetries == 0 {
		c.Generation.MaxRetries = 2
	}
	if c.Generation.RecencyWindow <= 0 {
		c.Generation.RecencyWindow = 20
	}
	c.applyProviderDefaults()
	if len(c.Generation.EnabledAgents) == 0 {
		c.Generation.EnabledAgents = []string{"short_story", "video", "image"}
	}
	if len(c.Generation.Weights) == 0 {
		c.Generation.Weights = map[string]int{"short_story": 60, "video": 25, "image": 15}
	} else {
		if c.Generation.Weights["short_story"] == 0 {
			c.Generation.Weights["short_story"] = 60
		}
		if c.Generation.Weights["video"] == 0 {
			c.Generation.Weights["video"] = 25
		}
		if c.Generation.Weights["image"] == 0 {
			c.Generation.Weights["image"] = 15
		}
	}

	if c.Memory.DSN == "" {
		c.Memory.DSN = "./data/universe.db"
	}
	if c.Memory.Driver == "" {
		c.Memory.Driver = "sqlite"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}

	if c.Channels.Filesystem.Enabled && c.Channels.Filesystem.OutputDir == "" {
		return errors.New("channels.filesystem.output_dir is required when enabled")
	}
	if c.Channels.Twitter.BaseURL == "" {
		c.Channels.Twitter.BaseURL = "https://api.twitter.com"
	}
	if c.Channels.Twitter.BearerTokenEnv == "" {
		c.Channels.Twitter.BearerTokenEnv = "TWITTER_BEARER_TOKEN"
	}

	c.normalizeArtistsFromLegacy()
	if err := c.validateArtists(); err != nil {
		return err
	}
	return nil
}

func validateScheduler(s SchedulerConfig, enabled bool) error {
	if !enabled {
		return nil
	}
	switch s.Mode {
	case "random_window":
		if _, err := time.ParseDuration(s.MinInterval); err != nil {
			return fmt.Errorf("invalid scheduler.min_interval: %w", err)
		}
		if _, err := time.ParseDuration(s.MaxInterval); err != nil {
			return fmt.Errorf("invalid scheduler.max_interval: %w", err)
		}
	case "fixed_interval":
		if _, err := time.ParseDuration(s.FixedInterval); err != nil {
			return fmt.Errorf("invalid scheduler.fixed_interval: %w", err)
		}
	default:
		return fmt.Errorf("invalid scheduler.mode: %s", s.Mode)
	}
	if s.Timezone == "" {
		s.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(s.Timezone); err != nil {
		return fmt.Errorf("invalid scheduler.timezone: %w", err)
	}
	return nil
}

func (c *Config) normalizeArtistsFromLegacy() {
	if len(c.Artists) > 0 {
		for i := range c.Artists {
			c.applyArtistDefaults(&c.Artists[i])
		}
		return
	}

	targets := []string{}
	if c.Channels.Filesystem.Enabled {
		targets = append(targets, "filesystem")
	}
	for _, typ := range c.Generation.EnabledAgents {
		id := typ + "-artist"
		artist := ArtistConfig{
			ID:             id,
			Type:           typ,
			Enabled:        boolPtr(true),
			Style:          "default",
			PublishTargets: append([]string(nil), targets...),
			Scheduler:      c.Scheduler,
		}
		switch {
		case isTextArtistType(typ):
			artist.Provider = c.Providers.Text
		case typ == "video":
			artist.Provider = c.Providers.Video
		case typ == "image":
			artist.Provider = c.Providers.Image
		}
		c.Artists = append(c.Artists, artist)
	}
}

func (c *Config) applyArtistDefaults(a *ArtistConfig) {
	if a.ID == "" {
		a.ID = a.Type + "-artist"
	}
	if a.ProfileID == "" {
		a.ProfileID = a.ID
	}
	if a.Style == "" {
		a.Style = "default"
	}
	if a.Enabled == nil {
		a.Enabled = boolPtr(true)
	}
	if a.Scheduler.Mode == "" {
		a.Scheduler = c.Scheduler
	}
	if a.Scheduler.Seed == 0 {
		a.Scheduler.Seed = c.Scheduler.Seed
	}
	if a.Scheduler.Timezone == "" {
		a.Scheduler.Timezone = c.Scheduler.Timezone
	}
	if len(a.PublishTargets) == 0 {
		if c.Channels.Filesystem.Enabled {
			a.PublishTargets = []string{"filesystem"}
		}
	}
	if a.Provider.Model == "" {
		switch {
		case isTextArtistType(a.Type):
			a.Provider = c.Providers.Text
		case a.Type == "video":
			a.Provider = c.Providers.Video
		case a.Type == "image":
			a.Provider = c.Providers.Image
		}
	}
	if a.Provider.Driver == "" {
		switch {
		case isTextArtistType(a.Type):
			a.Provider.Driver = c.Providers.Text.Driver
		case a.Type == "video":
			a.Provider.Driver = c.Providers.Video.Driver
		case a.Type == "image":
			a.Provider.Driver = c.Providers.Image.Driver
		}
	}
	if a.Options == nil {
		a.Options = map[string]any{}
	}
	if _, ok := a.Options["reference_mode"]; !ok {
		switch {
		case isTextArtistType(a.Type):
			a.Options["reference_mode"] = "continuity_only"
		case a.Type == "image", a.Type == "video":
			a.Options["reference_mode"] = "continuity_plus_assets"
		}
	}
	if _, ok := a.Options["continuity_scope"]; !ok {
		a.Options["continuity_scope"] = "same_artist"
	}
	if _, ok := a.Options["max_continuity_items"]; !ok {
		a.Options["max_continuity_items"] = 3
	}
	if _, ok := a.Options["max_asset_references"]; !ok {
		a.Options["max_asset_references"] = 4
	}
	if _, ok := a.Options["include_text_memories"]; !ok {
		a.Options["include_text_memories"] = true
	}
}

func boolPtr(v bool) *bool { return &v }

func (c *Config) validateArtists() error {
	if len(c.Artists) == 0 {
		return errors.New("at least one artist is required")
	}
	seen := map[string]bool{}
	for i := range c.Artists {
		a := &c.Artists[i]
		c.applyArtistDefaults(a)
		if a.ID == "" {
			return fmt.Errorf("artists[%d].id is required", i)
		}
		if a.ProfileID == "" {
			return fmt.Errorf("artist %s profile_id is required", a.ID)
		}
		if seen[a.ID] {
			return fmt.Errorf("duplicate artist id: %s", a.ID)
		}
		seen[a.ID] = true
		if !isTextArtistType(a.Type) && a.Type != "video" && a.Type != "image" {
			return fmt.Errorf("artist %s has invalid type: %s", a.ID, a.Type)
		}
		if a.Provider.Driver == "" {
			return fmt.Errorf("artist %s provider.driver is required", a.ID)
		}
		if err := validateArtistOptions(*a); err != nil {
			return err
		}
		if err := validateArtistOverrides(*a); err != nil {
			return err
		}
		if err := validateProviderDriver(*a); err != nil {
			return err
		}
		if err := validateScheduler(a.Scheduler, true); err != nil {
			return fmt.Errorf("artist %s scheduler invalid: %w", a.ID, err)
		}
	}
	sort.Slice(c.Artists, func(i, j int) bool { return c.Artists[i].ID < c.Artists[j].ID })
	return nil
}

func validateArtistOptions(a ArtistConfig) error {
	mode, _ := a.Options["reference_mode"].(string)
	switch mode {
	case "creative", "continuity_only", "continuity_plus_assets", "assets_only":
	default:
		return fmt.Errorf("artist %s options.reference_mode invalid: %s", a.ID, mode)
	}
	scope, _ := a.Options["continuity_scope"].(string)
	if scope != "same_artist" {
		return fmt.Errorf("artist %s options.continuity_scope invalid: %s", a.ID, scope)
	}
	if v, ok := intFromAny(a.Options["max_continuity_items"]); !ok || v < 0 {
		return fmt.Errorf("artist %s options.max_continuity_items must be >= 0", a.ID)
	}
	if v, ok := intFromAny(a.Options["max_asset_references"]); !ok || v < 0 {
		return fmt.Errorf("artist %s options.max_asset_references must be >= 0", a.ID)
	}
	if list, ok := stringSliceFromAny(a.Options["asset_usage_allowlist"]); ok {
		for _, item := range list {
			switch item {
			case "character_reference", "style_reference", "environment_reference", "prop_reference", "pose_reference", "continuity_reference", "video_prompt_image":
			default:
				return fmt.Errorf("artist %s options.asset_usage_allowlist contains invalid usage %s", a.ID, item)
			}
		}
	}
	return nil
}

func validateArtistOverrides(a ArtistConfig) error {
	if a.Presentation.SignatureMode != "" {
		switch a.Presentation.SignatureMode {
		case "none", "presentation_only", "append", "prepend":
		default:
			return fmt.Errorf("artist %s presentation.signature_mode invalid: %s", a.ID, a.Presentation.SignatureMode)
		}
	}
	if a.Presentation.FramingMode != "" {
		switch a.Presentation.FramingMode {
		case "none", "intro", "outro", "intro_outro":
		default:
			return fmt.Errorf("artist %s presentation.framing_mode invalid: %s", a.ID, a.Presentation.FramingMode)
		}
	}
	return nil
}

func intFromAny(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	default:
		return 0, false
	}
}

func stringSliceFromAny(v any) ([]string, bool) {
	switch t := v.(type) {
	case []string:
		return t, true
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}

func (c *Config) applyProviderDefaults() {
	if c.Providers.Text.Driver == "" {
		c.Providers.Text.Driver = "mock"
	}
	if c.Providers.Text.Model == "" && c.Providers.Text.Driver == "mock" {
		c.Providers.Text.Model = "mock-text-v1"
	}
	if c.Providers.Text.Model == "" && c.Providers.Text.Driver == "openai_text" {
		c.Providers.Text.Model = "gpt-5-mini"
	}
	if c.Providers.Text.Model == "" && c.Providers.Text.Driver == "lmstudio_text" {
		c.Providers.Text.Model = "qwen2.5-7b-instruct"
	}
	if c.Providers.Video.Driver == "" {
		c.Providers.Video.Driver = "mock"
	}
	if c.Providers.Video.Model == "" {
		c.Providers.Video.Model = "mock-video-v1"
	}
	if c.Providers.Video.PollInterval == "" {
		c.Providers.Video.PollInterval = "10s"
	}
	if c.Providers.Video.Timeout == "" {
		c.Providers.Video.Timeout = "10m"
	}
	if c.Providers.Image.Driver == "" {
		c.Providers.Image.Driver = "mock"
	}
	if c.Providers.Image.Model == "" {
		c.Providers.Image.Model = "mock-image-v1"
	}
	if c.Providers.Image.Timeout == "" {
		c.Providers.Image.Timeout = "2m"
	}
	if c.Providers.Text.Timeout == "" {
		c.Providers.Text.Timeout = "2m"
	}
	if c.Providers.Text.Driver == "lmstudio_text" && c.Providers.Text.BaseURL == "" {
		c.Providers.Text.BaseURL = "http://localhost:1234/v1"
	}
	if c.Providers.Image.Location == "" {
		c.Providers.Image.Location = "us-central1"
	}
	if c.Providers.Video.Location == "" {
		c.Providers.Video.Location = "us-central1"
	}
	if c.Providers.Video.Version == "" {
		c.Providers.Video.Version = "2024-11-06"
	}
}

func validateProviderDriver(a ArtistConfig) error {
	driver := a.Provider.Driver
	switch driver {
	case "mock":
		return nil
	case "openai_text":
		if !isTextArtistType(a.Type) {
			return fmt.Errorf("artist %s provider.driver openai_text only supports textual types", a.ID)
		}
		if a.Provider.APIKeyEnv == "" {
			return fmt.Errorf("artist %s provider.api_key_env is required for openai_text", a.ID)
		}
		if a.Provider.Model == "" {
			return fmt.Errorf("artist %s provider.model is required for openai_text", a.ID)
		}
		if a.Provider.Timeout != "" {
			if _, err := time.ParseDuration(a.Provider.Timeout); err != nil {
				return fmt.Errorf("artist %s provider.timeout invalid: %w", a.ID, err)
			}
		}
	case "lmstudio_text":
		if !isTextArtistType(a.Type) {
			return fmt.Errorf("artist %s provider.driver lmstudio_text only supports textual types", a.ID)
		}
		if a.Provider.Model == "" {
			return fmt.Errorf("artist %s provider.model is required for lmstudio_text", a.ID)
		}
		if a.Provider.Timeout != "" {
			if _, err := time.ParseDuration(a.Provider.Timeout); err != nil {
				return fmt.Errorf("artist %s provider.timeout invalid: %w", a.ID, err)
			}
		}
	case "openai_image":
		if a.Provider.APIKeyEnv == "" {
			return fmt.Errorf("artist %s provider.api_key_env is required for openai_image", a.ID)
		}
		if a.Provider.Model == "" {
			return fmt.Errorf("artist %s provider.model is required for openai_image", a.ID)
		}
		if v, ok := a.Provider.Options["response_format"].(string); ok && v != "" && v != "b64_json" && v != "url" {
			return fmt.Errorf("artist %s provider.options.response_format must be b64_json or url", a.ID)
		}
	case "vertex_imagen":
		if a.Provider.ProjectIDEnv == "" {
			return fmt.Errorf("artist %s provider.project_id_env is required for vertex_imagen", a.ID)
		}
		if a.Provider.Location == "" {
			return fmt.Errorf("artist %s provider.location is required for vertex_imagen", a.ID)
		}
		if a.Provider.Model == "" {
			return fmt.Errorf("artist %s provider.model is required for vertex_imagen", a.ID)
		}
		if a.Provider.Timeout != "" {
			if _, err := time.ParseDuration(a.Provider.Timeout); err != nil {
				return fmt.Errorf("artist %s provider.timeout invalid: %w", a.ID, err)
			}
		}
	case "vertex_veo":
		if a.Provider.ProjectIDEnv == "" {
			return fmt.Errorf("artist %s provider.project_id_env is required for vertex_veo", a.ID)
		}
		if a.Provider.Location == "" {
			return fmt.Errorf("artist %s provider.location is required for vertex_veo", a.ID)
		}
		if a.Provider.BucketURI == "" {
			return fmt.Errorf("artist %s provider.bucket_uri is required for vertex_veo", a.ID)
		}
		if a.Provider.Model == "" {
			return fmt.Errorf("artist %s provider.model is required for vertex_veo", a.ID)
		}
		if _, err := time.ParseDuration(a.Provider.PollInterval); err != nil {
			return fmt.Errorf("artist %s provider.poll_interval invalid: %w", a.ID, err)
		}
		if _, err := time.ParseDuration(a.Provider.Timeout); err != nil {
			return fmt.Errorf("artist %s provider.timeout invalid: %w", a.ID, err)
		}
	case "runway_gen4":
		if a.Provider.APIKeyEnv == "" {
			return fmt.Errorf("artist %s provider.api_key_env is required for runway_gen4", a.ID)
		}
		if a.Provider.Model == "" {
			return fmt.Errorf("artist %s provider.model is required for runway_gen4", a.ID)
		}
		if a.Provider.Version == "" {
			return fmt.Errorf("artist %s provider.version is required for runway_gen4", a.ID)
		}
		if _, err := time.ParseDuration(a.Provider.PollInterval); err != nil {
			return fmt.Errorf("artist %s provider.poll_interval invalid: %w", a.ID, err)
		}
		if _, err := time.ParseDuration(a.Provider.Timeout); err != nil {
			return fmt.Errorf("artist %s provider.timeout invalid: %w", a.ID, err)
		}
	default:
		return fmt.Errorf("artist %s provider.driver unsupported: %s", a.ID, driver)
	}
	return nil
}

func isTextArtistType(value string) bool {
	switch value {
	case "tweet_short", "tweet_thread", "short_story", "long_story", "poem", "song_lyrics", "screenplay_series":
		return true
	default:
		return false
	}
}
