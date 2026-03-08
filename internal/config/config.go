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
	App        AppConfig        `yaml:"app"`
	Universe   UniverseConfig   `yaml:"universe"`
	Scheduler  SchedulerConfig  `yaml:"scheduler"`
	Generation GenerationConfig `yaml:"generation"`
	Providers  ProvidersConfig  `yaml:"providers"`
	Artists    []ArtistConfig   `yaml:"artists"`
	Channels   ChannelsConfig   `yaml:"channels"`
	Memory     MemoryConfig     `yaml:"memory"`
	Logging    LoggingConfig    `yaml:"logging"`
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
	ID             string          `yaml:"id"`
	Type           string          `yaml:"type"`
	Enabled        *bool           `yaml:"enabled"`
	Style          string          `yaml:"style"`
	Provider       ProviderDriver  `yaml:"provider"`
	Options        map[string]any  `yaml:"options"`
	PublishTargets []string        `yaml:"publish_targets"`
	Scheduler      SchedulerConfig `yaml:"scheduler"`
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
		c.Generation.EnabledAgents = []string{"text", "video", "image"}
	}
	if len(c.Generation.Weights) == 0 {
		c.Generation.Weights = map[string]int{"text": 60, "video": 25, "image": 15}
	} else {
		if c.Generation.Weights["text"] == 0 {
			c.Generation.Weights["text"] = 60
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
		switch typ {
		case "text":
			artist.Provider = c.Providers.Text
		case "video":
			artist.Provider = c.Providers.Video
		case "image":
			artist.Provider = c.Providers.Image
		}
		c.Artists = append(c.Artists, artist)
	}
}

func (c *Config) applyArtistDefaults(a *ArtistConfig) {
	if a.ID == "" {
		a.ID = a.Type + "-artist"
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
		switch a.Type {
		case "text":
			a.Provider = c.Providers.Text
		case "video":
			a.Provider = c.Providers.Video
		case "image":
			a.Provider = c.Providers.Image
		}
	}
	if a.Provider.Driver == "" {
		switch a.Type {
		case "text":
			a.Provider.Driver = c.Providers.Text.Driver
		case "video":
			a.Provider.Driver = c.Providers.Video.Driver
		case "image":
			a.Provider.Driver = c.Providers.Image.Driver
		}
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
		if seen[a.ID] {
			return fmt.Errorf("duplicate artist id: %s", a.ID)
		}
		seen[a.ID] = true
		if a.Type != "text" && a.Type != "video" && a.Type != "image" {
			return fmt.Errorf("artist %s has invalid type: %s", a.ID, a.Type)
		}
		if a.Provider.Driver == "" {
			return fmt.Errorf("artist %s provider.driver is required", a.ID)
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

func (c *Config) applyProviderDefaults() {
	if c.Providers.Text.Driver == "" {
		c.Providers.Text.Driver = "mock"
	}
	if c.Providers.Text.Model == "" {
		c.Providers.Text.Model = "mock-text-v1"
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
