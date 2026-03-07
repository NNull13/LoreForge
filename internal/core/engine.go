package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"loreforge/internal/agents/text"
	"loreforge/internal/agents/video"
	"loreforge/internal/channels"
	"loreforge/internal/config"
	"loreforge/internal/memory"
	"loreforge/internal/planner"
	"loreforge/internal/providers"
	"loreforge/internal/scheduler"
	"loreforge/internal/universe"
	"loreforge/internal/util"
	"loreforge/pkg/contracts"
)

type Engine struct {
	cfg       config.Config
	store     *memory.Store
	planner   *planner.Planner
	scheduler *scheduler.Scheduler
	agents    map[string]contracts.Agent
	channels  []contracts.Channel
}

func New(cfg config.Config) (*Engine, error) {
	minInt, _ := time.ParseDuration(cfg.Scheduler.MinInterval)
	maxInt, _ := time.ParseDuration(cfg.Scheduler.MaxInterval)
	fixInt, _ := time.ParseDuration(cfg.Scheduler.FixedInterval)
	sch, err := scheduler.New(scheduler.Config{
		Mode:          cfg.Scheduler.Mode,
		MinInterval:   minInt,
		MaxInterval:   maxInt,
		FixedInterval: fixInt,
		Seed:          cfg.Scheduler.Seed,
		Timezone:      cfg.Scheduler.Timezone,
	})
	if err != nil {
		return nil, err
	}

	txtProvider := providers.MockTextProvider{Model: cfg.Providers.Text.Model}
	vidProvider := providers.MockVideoProvider{Model: cfg.Providers.Video.Model}

	agents := map[string]contracts.Agent{
		"text":  text.Agent{Provider: txtProvider},
		"video": video.Agent{Provider: vidProvider, Seed: cfg.Scheduler.Seed},
	}

	var outChannels []contracts.Channel
	if cfg.Channels.Filesystem.Enabled {
		outChannels = append(outChannels, channels.FilesystemChannel{OutputDir: cfg.Channels.Filesystem.OutputDir})
	}

	return &Engine{
		cfg:       cfg,
		store:     memory.New("./data"),
		planner:   planner.New(planner.Config{Weights: cfg.Generation.Weights, RecencyWindow: cfg.Generation.RecencyWindow, Seed: cfg.Scheduler.Seed}),
		scheduler: sch,
		agents:    agents,
		channels:  outChannels,
	}, nil
}

func (e *Engine) ValidateUniverse() error {
	_, err := universe.Load(e.cfg.Universe.Path)
	return err
}

func (e *Engine) NextRun(now time.Time) (time.Time, error) {
	return e.scheduler.NextRun(now)
}

func (e *Engine) GenerateOnce(ctx context.Context, requestedAgent string) (contracts.EpisodeRecord, error) {
	u, err := universe.Load(e.cfg.Universe.Path)
	if err != nil {
		return contracts.EpisodeRecord{}, err
	}
	recent, err := e.store.RecentCombos(e.cfg.Generation.RecencyWindow)
	if err != nil {
		return contracts.EpisodeRecord{}, err
	}
	brief, err := e.planner.BuildBrief(u, recent)
	if err != nil {
		return contracts.EpisodeRecord{}, err
	}
	if requestedAgent != "" {
		brief.EpisodeType = requestedAgent
		brief.TemplateID = pickTemplateForType(u, requestedAgent, brief.TemplateID)
	}
	agent, ok := e.agents[brief.EpisodeType]
	if !ok {
		return contracts.EpisodeRecord{}, fmt.Errorf("agent not available: %s", brief.EpisodeType)
	}

	uHash, err := util.HashDir(e.cfg.Universe.Path)
	if err != nil {
		return contracts.EpisodeRecord{}, err
	}

	state := contracts.EpisodeState{UniverseVersion: uHash}
	var out contracts.EpisodeOutput
	var retry int
	for {
		out, err = agent.Generate(ctx, brief, state)
		if err == nil {
			verr := validateOutput(out, brief)
			if verr == nil {
				break
			}
			err = verr
		}
		retry++
		if retry > e.cfg.Generation.MaxRetries {
			return contracts.EpisodeRecord{}, err
		}
	}

	epID := util.NewEpisodeID()
	now := time.Now()
	record := contracts.EpisodeRecord{
		Manifest: contracts.EpisodeManifest{
			EpisodeID:       epID,
			CreatedAt:       now,
			Agent:           agent.Name(),
			OutputType:      brief.EpisodeType,
			UniverseVersion: uHash,
			WorldIDs:        []string{brief.WorldID},
			CharacterIDs:    brief.CharacterIDs,
			EventID:         brief.EventID,
			TemplateID:      brief.TemplateID,
			PromptInput:     brief.Objective,
			PromptFinal:     out.Prompt,
			Provider:        out.Provider,
			Model:           out.Model,
			Seed:            e.cfg.Scheduler.Seed,
			RetryCount:      retry,
			Scores: map[string]any{
				"length_ok":         len(out.Content) >= 50 || out.AssetPath != "",
				"contains_entities": containsEntities(out.Content, brief.CharacterIDs),
			},
			State: "generated",
		},
		Context: map[string]any{
			"brief":    brief,
			"universe": u.Universe.ID,
		},
		Prompt:           out.Prompt,
		ProviderRequest:  memory.SanitizeSecrets(out.ProviderRequest),
		ProviderResponse: memory.SanitizeSecrets(out.ProviderResponse),
		OutputText:       out.Content,
		OutputAssetPath:  out.AssetPath,
	}

	publishedChannels := make([]string, 0)
	publishResult := map[string]any{}
	for _, ch := range e.channels {
		res, err := ch.Publish(ctx, contracts.PublishableContent{
			EpisodeID:  epID,
			OutputType: brief.EpisodeType,
			Content:    out.Content,
			AssetPath:  out.AssetPath,
			CreatedAt:  now,
		})
		if err != nil {
			publishResult[ch.Name()] = map[string]any{"success": false, "error": err.Error()}
			continue
		}
		publishedChannels = append(publishedChannels, ch.Name())
		publishResult[ch.Name()] = res
	}
	record.Manifest.Published = len(publishedChannels) > 0
	record.Manifest.Channels = publishedChannels
	if record.Manifest.Published {
		record.Manifest.State = "published"
	}
	record.Publish = publishResult

	if _, err := e.store.SaveEpisode(record); err != nil {
		return contracts.EpisodeRecord{}, err
	}
	next, err := e.scheduler.NextRun(now)
	if err == nil {
		_ = e.store.SaveSchedulerState(memory.SchedulerState{LastRunAt: &now, NextRunAt: next})
	}
	return record, nil
}

func (e *Engine) ShowEpisode(episodeID string) (string, contracts.EpisodeManifest, error) {
	return e.store.FindEpisode(episodeID)
}

func validateOutput(out contracts.EpisodeOutput, brief contracts.EpisodeBrief) error {
	if brief.EpisodeType == "text" {
		if len(strings.TrimSpace(out.Content)) < 40 {
			return errors.New("validation failed: text output too short")
		}
	}
	for _, bad := range []string{"api_key", "token=", "secret="} {
		if strings.Contains(strings.ToLower(out.Content), bad) {
			return errors.New("validation failed: forbidden term found")
		}
	}
	return nil
}

func containsEntities(content string, entities []string) bool {
	if len(entities) == 0 {
		return true
	}
	l := strings.ToLower(content)
	for _, e := range entities {
		if strings.Contains(l, strings.ToLower(strings.ReplaceAll(e, "-", " "))) || strings.Contains(l, strings.ToLower(e)) {
			return true
		}
	}
	return false
}

func pickTemplateForType(u universe.Universe, outputType, fallback string) string {
	for id, t := range u.Templates {
		if v, ok := t.Data["output_type"].(string); ok && v == outputType {
			return id
		}
	}
	return fallback
}
