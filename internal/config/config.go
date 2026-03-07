package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"loreforge/internal/util"
)

type Config struct {
	App        AppConfig
	Universe   UniverseConfig
	Scheduler  SchedulerConfig
	Generation GenerationConfig
	Providers  ProvidersConfig
	Channels   ChannelsConfig
	Memory     MemoryConfig
	Logging    LoggingConfig
}

type AppConfig struct {
	Name string
	Env  string
}

type UniverseConfig struct {
	Path string
}

type SchedulerConfig struct {
	Enabled       bool
	Mode          string
	MinInterval   string
	MaxInterval   string
	FixedInterval string
	Seed          int64
	Timezone      string
}

type GenerationConfig struct {
	EnabledAgents []string
	Weights       map[string]int
	MaxRetries    int
	RecencyWindow int
}

type ProviderDriver struct {
	Driver    string
	Model     string
	APIKeyEnv string
}

type ProvidersConfig struct {
	Text  ProviderDriver
	Video ProviderDriver
}

type FilesystemChannelConfig struct {
	Enabled   bool
	OutputDir string
}

type ChannelsConfig struct {
	Filesystem FilesystemChannelConfig
}

type MemoryConfig struct {
	Driver string
	DSN    string
}

type LoggingConfig struct {
	Level string
}

func Load(path string) (Config, error) {
	var cfg Config
	m, err := util.ParseSimpleYAMLFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	cfg = Config{
		App:      AppConfig{Name: m["app.name"], Env: m["app.env"]},
		Universe: UniverseConfig{Path: m["universe.path"]},
		Scheduler: SchedulerConfig{
			Enabled:       util.MustBool(m["scheduler.enabled"], true),
			Mode:          m["scheduler.mode"],
			MinInterval:   m["scheduler.min_interval"],
			MaxInterval:   m["scheduler.max_interval"],
			FixedInterval: m["scheduler.fixed_interval"],
			Seed:          util.MustInt64(m["scheduler.seed"], 42),
			Timezone:      m["scheduler.timezone"],
		},
		Generation: GenerationConfig{
			EnabledAgents: util.ParseStringListValue(m["generation.enabled_agents"]),
			Weights: map[string]int{
				"text":  util.MustInt(m["generation.weights.text"], 70),
				"video": util.MustInt(m["generation.weights.video"], 30),
			},
			MaxRetries:    util.MustInt(m["generation.max_retries"], 2),
			RecencyWindow: util.MustInt(m["generation.recency_window"], 20),
		},
		Providers: ProvidersConfig{
			Text:  ProviderDriver{Driver: m["providers.text.driver"], Model: m["providers.text.model"], APIKeyEnv: m["providers.text.api_key_env"]},
			Video: ProviderDriver{Driver: m["providers.video.driver"], Model: m["providers.video.model"], APIKeyEnv: m["providers.video.api_key_env"]},
		},
		Channels: ChannelsConfig{Filesystem: FilesystemChannelConfig{
			Enabled:   util.MustBool(m["channels.filesystem.enabled"], true),
			OutputDir: m["channels.filesystem.output_dir"],
		}},
		Memory:  MemoryConfig{Driver: m["memory.driver"], DSN: m["memory.dsn"]},
		Logging: LoggingConfig{Level: m["logging.level"]},
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
	if c.Generation.MaxRetries < 0 {
		return errors.New("generation.max_retries cannot be negative")
	}
	if c.Generation.RecencyWindow <= 0 {
		c.Generation.RecencyWindow = 20
	}
	if c.Scheduler.Enabled {
		switch c.Scheduler.Mode {
		case "random_window":
			if _, err := time.ParseDuration(c.Scheduler.MinInterval); err != nil {
				return fmt.Errorf("invalid scheduler.min_interval: %w", err)
			}
			if _, err := time.ParseDuration(c.Scheduler.MaxInterval); err != nil {
				return fmt.Errorf("invalid scheduler.max_interval: %w", err)
			}
		case "fixed_interval":
			if _, err := time.ParseDuration(c.Scheduler.FixedInterval); err != nil {
				return fmt.Errorf("invalid scheduler.fixed_interval: %w", err)
			}
		default:
			return fmt.Errorf("invalid scheduler.mode: %s", c.Scheduler.Mode)
		}
	}
	if c.Scheduler.Timezone == "" {
		c.Scheduler.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(c.Scheduler.Timezone); err != nil {
		return fmt.Errorf("invalid scheduler.timezone: %w", err)
	}
	if c.Channels.Filesystem.Enabled && c.Channels.Filesystem.OutputDir == "" {
		return errors.New("channels.filesystem.output_dir is required when enabled")
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
	if len(c.Generation.EnabledAgents) == 0 {
		c.Generation.EnabledAgents = []string{"text", "video"}
	}
	if len(c.Generation.Weights) == 0 {
		c.Generation.Weights = map[string]int{"text": 70, "video": 30}
	}
	return nil
}
