package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	generatorimage "loreforge/internal/adapters/generators/image"
	generatorregistry "loreforge/internal/adapters/generators/registry"
	generatortext "loreforge/internal/adapters/generators/text"
	generatorvideo "loreforge/internal/adapters/generators/video"
	providerfactory "loreforge/internal/adapters/providers/factory"
	publisherfilesystem "loreforge/internal/adapters/publishers/filesystem"
	publisherregistry "loreforge/internal/adapters/publishers/registry"
	publishertwitter "loreforge/internal/adapters/publishers/twitter"
	"loreforge/internal/adapters/repositories/episodestore"
	"loreforge/internal/adapters/repositories/schedulerstatefs"
	"loreforge/internal/adapters/repositories/universefs"
	"loreforge/internal/application/generateepisode"
	"loreforge/internal/application/nextrun"
	"loreforge/internal/application/ports"
	"loreforge/internal/application/showepisode"
	"loreforge/internal/application/textsettings"
	"loreforge/internal/application/validateuniverse"
	"loreforge/internal/config"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/publication"
	"loreforge/internal/domain/scheduling"
	"loreforge/internal/planner"
	"loreforge/internal/platform/hashutil"
	"loreforge/internal/platform/idgen"
	"loreforge/internal/platform/timeutil"
)

type app struct {
	generate generateepisode.Handler
	validate validateuniverse.Handler
	show     showepisode.Handler
	nextRun  nextrun.Handler
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "validate":
		validateCmd(os.Args[2:])
	case "generate":
		generateCmd(os.Args[2:])
	case "episode":
		episodeCmd(os.Args[2:])
	case "universe":
		universeCmd(os.Args[2:])
	case "scheduler":
		schedulerCmd(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "./universes/config.yaml", "path to config yaml")
	_ = fs.Parse(args)
	cfg := loadConfigOrExit(*configPath)
	app, err := buildApp(cfg)
	must(err)
	res, err := app.generate.Handle(context.Background(), generateepisode.Request{
		MaxRetries:    cfg.Generation.MaxRetries,
		RecencyWindow: cfg.Generation.RecencyWindow,
	})
	must(err)
	fmt.Printf("run complete: episode=%s state=%s type=%s\n", res.Record.Manifest.EpisodeID, res.Record.Manifest.State, res.Record.Manifest.OutputType)
}

func validateCmd(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	configPath := fs.String("config", "./universes/config.yaml", "path to config yaml")
	_ = fs.Parse(args)
	cfg := loadConfigOrExit(*configPath)
	app, err := buildApp(cfg)
	must(err)
	must(app.validate.Handle(context.Background()))
	fmt.Println("validate ok")
}

func generateCmd(args []string) {
	if len(args) == 0 || args[0] != "once" {
		fmt.Fprintln(os.Stderr, "usage: loreforge generate once [--artist short-story-artist|tweet-thread-artist] --config ./config.yaml")
		os.Exit(1)
	}
	fs := flag.NewFlagSet("generate once", flag.ExitOnError)
	artist := fs.String("artist", "", "artist id or type")
	agent := fs.String("agent", "", "legacy alias for generator type")
	configPath := fs.String("config", "./universes/config.yaml", "path to config yaml")
	_ = fs.Parse(args[1:])
	cfg := loadConfigOrExit(*configPath)
	app, err := buildApp(cfg)
	must(err)
	selected := *artist
	if selected == "" {
		selected = *agent
	}
	res, err := app.generate.Handle(context.Background(), generateepisode.Request{
		Generator:     selected,
		MaxRetries:    cfg.Generation.MaxRetries,
		RecencyWindow: cfg.Generation.RecencyWindow,
	})
	must(err)
	fmt.Printf("generated: episode=%s type=%s artist=%s\n", res.Record.Manifest.EpisodeID, res.Record.Manifest.OutputType, res.Record.Manifest.ArtistID)
}

func episodeCmd(args []string) {
	if len(args) < 2 || args[0] != "show" {
		fmt.Fprintln(os.Stderr, "usage: loreforge episode show <id> --config ./config.yaml")
		os.Exit(1)
	}
	epID := args[1]
	fs := flag.NewFlagSet("episode show", flag.ExitOnError)
	configPath := fs.String("config", "./universes/config.yaml", "path to config yaml")
	_ = fs.Parse(args[2:])
	cfg := loadConfigOrExit(*configPath)
	app, err := buildApp(cfg)
	must(err)
	res, err := app.show.Handle(context.Background(), showepisode.Request{EpisodeID: epID})
	must(err)
	b, _ := json.MarshalIndent(res.Manifest, "", "  ")
	fmt.Printf("episode path: %s\n%s\n", res.Path, string(b))
}

func universeCmd(args []string) {
	if len(args) < 2 || args[0] != "lint" {
		fmt.Fprintln(os.Stderr, "usage: loreforge universe lint ./universe")
		os.Exit(1)
	}
	path := args[1]
	repo := universefs.Repository{Root: path}
	_, err := repo.Load(context.Background())
	must(err)
	fmt.Println("universe lint ok")
}

func schedulerCmd(args []string) {
	if len(args) == 0 || args[0] != "next-run" {
		fmt.Fprintln(os.Stderr, "usage: loreforge scheduler next-run [--artist short-story-artist] --config ./config.yaml")
		os.Exit(1)
	}
	fs := flag.NewFlagSet("scheduler next-run", flag.ExitOnError)
	artist := fs.String("artist", "", "artist id")
	configPath := fs.String("config", "./universes/config.yaml", "path to config yaml")
	_ = fs.Parse(args[1:])
	cfg := loadConfigOrExit(*configPath)
	app, err := buildApp(cfg)
	must(err)
	if *artist != "" {
		next, err := app.nextRun.Handle(context.Background(), nextrun.Request{GeneratorID: *artist})
		must(err)
		fmt.Printf("next run (%s): %s\n", *artist, next.Format(time.RFC3339))
		return
	}
	next, err := app.nextRun.Handle(context.Background(), nextrun.Request{})
	must(err)
	fmt.Printf("next run (any artist): %s\n", next.Format(time.RFC3339))
}

func loadConfigOrExit(path string) config.Config {
	cfg, err := config.Load(path)
	must(err)
	return cfg
}

func usage() {
	fmt.Println(strings.TrimSpace(`
Usage:
  loreforge run --config ./config.yaml
  loreforge validate --config ./config.yaml
  loreforge generate once --artist short-story-artist --config ./config.yaml
  loreforge generate once --artist tweet-thread-artist --config ./config.yaml
  loreforge episode show <id> --config ./config.yaml
  loreforge universe lint ./universe
  loreforge scheduler next-run --artist short-story-artist --config ./config.yaml
`))
}

func must(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func buildApp(cfg config.Config) (app, error) {
	generators, err := buildGeneratorRegistry(cfg)
	if err != nil {
		return app{}, err
	}
	publishers := buildPublisherRegistry(cfg)
	universeRepo := universefs.Repository{Root: cfg.Universe.Path}
	episodeRepo := episodestore.New(cfg.Memory.DSN)
	schedulerRepo := schedulerstatefs.Repository{BaseDir: episodestore.BaseDirFromDSN(cfg.Memory.DSN)}
	plannerSvc := planner.New(planner.Config{
		Weights:        cfg.Generation.Weights,
		RecencyWindow:  cfg.Generation.RecencyWindow,
		Seed:           cfg.Scheduler.Seed,
		ProductionMode: isProductionEnv(cfg.App.Env),
	})
	clock := timeutil.RealClock{}
	return app{
		generate: generateepisode.Handler{
			UniverseRepo:       universeRepo,
			EpisodeRepo:        episodeRepo,
			SchedulerStateRepo: schedulerRepo,
			GeneratorRegistry:  generators,
			PublisherRegistry:  publishers,
			Clock:              clock,
			IDGenerator:        idgen.CryptoIDGenerator{},
			Hasher:             hashutil.DirHasher{Root: cfg.Universe.Path},
			Planner:            plannerSvc,
		},
		validate: validateuniverse.Handler{UniverseRepo: universeRepo},
		show:     showepisode.Handler{EpisodeRepo: episodeRepo},
		nextRun: nextrun.Handler{
			Registry:           generators,
			SchedulerStateRepo: schedulerRepo,
			Clock:              clock,
		},
	}, nil
}

func buildGeneratorRegistry(cfg config.Config) (ports.GeneratorRegistry, error) {
	defs := make([]ports.RegisteredGenerator, 0, len(cfg.Artists))
	for _, ac := range cfg.Artists {
		if ac.Enabled != nil && !*ac.Enabled {
			continue
		}
		schedulerCfg, err := toSchedulingConfig(ac.Scheduler)
		if err != nil {
			return nil, fmt.Errorf("generator %s scheduler: %w", ac.ID, err)
		}
		def := ports.RegisteredGenerator{
			Config: ports.GeneratorConfig{
				ID:                  ac.ID,
				Type:                episode.OutputType(ac.Type),
				Style:               ac.Style,
				PublishTargets:      toPublishTargets(ac.PublishTargets),
				Scheduler:           schedulerCfg,
				Seed:                ac.Scheduler.Seed,
				ProviderDriver:      ac.Provider.Driver,
				ProviderModel:       ac.Provider.Model,
				ProviderConfig:      providerConfigMap(ac.Provider),
				Options:             cloneAnyMap(ac.Options),
				ReferenceMode:       optionString(ac.Options, "reference_mode", "creative"),
				ContinuityScope:     optionString(ac.Options, "continuity_scope", "same_artist"),
				MaxContinuityItems:  optionInt(ac.Options, "max_continuity_items", 3),
				MaxAssetReferences:  optionInt(ac.Options, "max_asset_references", 4),
				IncludeTextMemories: optionBool(ac.Options, "include_text_memories", true),
				AssetUsageAllowlist: optionStringSlice(ac.Options, "asset_usage_allowlist"),
			},
		}
		switch {
		case isTextArtistType(ac.Type):
			provider, err := providerfactory.NewTextProvider(ac.Provider)
			if err != nil {
				return nil, fmt.Errorf("generator %s provider: %w", ac.ID, err)
			}
			settings, err := textsettings.ResolveTextSettings(cfg, ac)
			if err != nil {
				return nil, fmt.Errorf("generator %s text settings: %w", ac.ID, err)
			}
			def.Config.TextConstraints = settings.ToConstraints()
			def.Generator = generatortext.Generator{GeneratorID: ac.ID, Format: episode.OutputType(ac.Type), Settings: settings, Provider: provider}
		case ac.Type == "video":
			provider, err := providerfactory.NewVideoProvider(ac.Provider)
			if err != nil {
				return nil, fmt.Errorf("generator %s provider: %w", ac.ID, err)
			}
			def.Generator = generatorvideo.Generator{GeneratorID: ac.ID, Provider: provider, Seed: ac.Scheduler.Seed}
		case ac.Type == "image":
			provider, err := providerfactory.NewImageProvider(ac.Provider)
			if err != nil {
				return nil, fmt.Errorf("generator %s provider: %w", ac.ID, err)
			}
			def.Generator = generatorimage.Generator{GeneratorID: ac.ID, Provider: provider, Seed: ac.Scheduler.Seed}
		default:
			return nil, fmt.Errorf("generator %s has unsupported type: %s", ac.ID, ac.Type)
		}
		defs = append(defs, def)
	}
	return generatorregistry.New(defs), nil
}

func buildPublisherRegistry(cfg config.Config) ports.PublisherRegistry {
	var items []ports.Publisher
	if cfg.Channels.Filesystem.Enabled {
		items = append(items, publisherfilesystem.Publisher{OutputDir: cfg.Channels.Filesystem.OutputDir})
	}
	if cfg.Channels.Twitter.Enabled {
		items = append(items, publishertwitter.Publisher{
			DryRun:         cfg.Channels.Twitter.DryRun,
			BearerTokenEnv: cfg.Channels.Twitter.BearerTokenEnv,
			BaseURL:        cfg.Channels.Twitter.BaseURL,
		})
	}
	return publisherregistry.New(items)
}

func toSchedulingConfig(cfg config.SchedulerConfig) (scheduling.Config, error) {
	minInt, err := time.ParseDuration(cfg.MinInterval)
	if cfg.Mode == "random_window" && err != nil {
		return scheduling.Config{}, err
	}
	maxInt, err := time.ParseDuration(cfg.MaxInterval)
	if cfg.Mode == "random_window" && err != nil {
		return scheduling.Config{}, err
	}
	fixedInt, err := time.ParseDuration(cfg.FixedInterval)
	if cfg.Mode == "fixed_interval" && err != nil {
		return scheduling.Config{}, err
	}
	return scheduling.Config{
		Mode:          scheduling.Mode(cfg.Mode),
		MinInterval:   minInt,
		MaxInterval:   maxInt,
		FixedInterval: fixedInt,
		Seed:          cfg.Seed,
		Timezone:      cfg.Timezone,
	}, nil
}

func toPublishTargets(values []string) []publication.ChannelName {
	out := make([]publication.ChannelName, 0, len(values))
	for _, value := range values {
		out = append(out, publication.ChannelName(value))
	}
	return out
}

func isProductionEnv(env string) bool {
	value := strings.ToLower(strings.TrimSpace(env))
	return value == "prod" || value == "production"
}

func providerConfigMap(cfg config.ProviderDriver) map[string]any {
	return map[string]any{
		"driver":         cfg.Driver,
		"model":          cfg.Model,
		"api_key_env":    cfg.APIKeyEnv,
		"base_url":       cfg.BaseURL,
		"project_id_env": cfg.ProjectIDEnv,
		"location":       cfg.Location,
		"bucket_uri":     cfg.BucketURI,
		"poll_interval":  cfg.PollInterval,
		"timeout":        cfg.Timeout,
		"version":        cfg.Version,
		"options":        cloneAnyMap(cfg.Options),
	}
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func optionString(options map[string]any, key, fallback string) string {
	if v, ok := options[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func optionBool(options map[string]any, key string, fallback bool) bool {
	if v, ok := options[key].(bool); ok {
		return v
	}
	return fallback
}

func optionInt(options map[string]any, key string, fallback int) int {
	switch v := options[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return fallback
	}
}

func optionStringSlice(options map[string]any, key string) []string {
	switch v := options[key].(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func isTextArtistType(value string) bool {
	switch value {
	case "tweet_short", "tweet_thread", "short_story", "long_story", "poem", "song_lyrics", "screenplay_series":
		return true
	default:
		return false
	}
}
