package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App        AppConfig        `yaml:"app"`
	Universe   UniverseConfig   `yaml:"universe"`
	Scheduler  SchedulerConfig  `yaml:"scheduler"`
	Generation GenerationConfig `yaml:"generation"`
	Providers  ProvidersConfig  `yaml:"providers"`
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
	Driver    string `yaml:"driver"`
	Model     string `yaml:"model"`
	APIKeyEnv string `yaml:"api_key_env"`
}

type ProvidersConfig struct {
	Text  ProviderDriver `yaml:"text"`
	Video ProviderDriver `yaml:"video"`
}

type FilesystemChannelConfig struct {
	Enabled   bool   `yaml:"enabled"`
	OutputDir string `yaml:"output_dir"`
}

type ChannelsConfig struct {
	Filesystem FilesystemChannelConfig `yaml:"filesystem"`
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
		Channels:  ChannelsConfig{Filesystem: FilesystemChannelConfig{Enabled: true}},
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
	if c.Generation.MaxRetries < 0 {
		return errors.New("generation.max_retries cannot be negative")
	}
	if c.Generation.MaxRetries == 0 {
		c.Generation.MaxRetries = 2
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
	} else {
		if c.Generation.Weights["text"] == 0 {
			c.Generation.Weights["text"] = 70
		}
		if c.Generation.Weights["video"] == 0 {
			c.Generation.Weights["video"] = 30
		}
	}
	return nil
}
