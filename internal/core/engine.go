package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"loreforge/internal/artists"
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

type artistRuntime struct {
	cfg       config.ArtistConfig
	artist    artists.Artist
	scheduler *scheduler.Scheduler
}

type Engine struct {
	cfg      config.Config
	store    *memory.Store
	planner  *planner.Planner
	artists  map[string]artistRuntime
	channels map[string]contracts.Channel
}

func New(cfg config.Config) (*Engine, error) {
	channelsMap := map[string]contracts.Channel{}
	if cfg.Channels.Filesystem.Enabled {
		channelsMap["filesystem"] = channels.FilesystemChannel{OutputDir: cfg.Channels.Filesystem.OutputDir}
	}
	if cfg.Channels.Twitter.Enabled {
		channelsMap["twitter"] = channels.TwitterChannel{
			DryRun:         cfg.Channels.Twitter.DryRun,
			BearerTokenEnv: cfg.Channels.Twitter.BearerTokenEnv,
			BaseURL:        cfg.Channels.Twitter.BaseURL,
		}
	}

	artists := map[string]artistRuntime{}
	for _, ac := range cfg.Artists {
		if ac.Enabled != nil && !*ac.Enabled {
			continue
		}
		rt, err := buildArtistRuntime(ac)
		if err != nil {
			return nil, err
		}
		artists[ac.ID] = rt
	}
	if len(artists) == 0 {
		return nil, errors.New("no enabled artists configured")
	}

	return &Engine{
		cfg:   cfg,
		store: memory.New(cfg.Memory.DSN),
		planner: planner.New(planner.Config{
			Weights:        cfg.Generation.Weights,
			RecencyWindow:  cfg.Generation.RecencyWindow,
			Seed:           cfg.Scheduler.Seed,
			ProductionMode: isProductionEnv(cfg.App.Env),
		}),
		artists:  artists,
		channels: channelsMap,
	}, nil
}

func buildArtistRuntime(ac config.ArtistConfig) (artistRuntime, error) {
	minInt, _ := time.ParseDuration(ac.Scheduler.MinInterval)
	maxInt, _ := time.ParseDuration(ac.Scheduler.MaxInterval)
	fixInt, _ := time.ParseDuration(ac.Scheduler.FixedInterval)
	sch, err := scheduler.New(scheduler.Config{
		Mode:          ac.Scheduler.Mode,
		MinInterval:   minInt,
		MaxInterval:   maxInt,
		FixedInterval: fixInt,
		Seed:          ac.Scheduler.Seed,
		Timezone:      ac.Scheduler.Timezone,
	})
	if err != nil {
		return artistRuntime{}, fmt.Errorf("artist %s scheduler: %w", ac.ID, err)
	}

	switch ac.Type {
	case "text":
		p, err := providers.NewTextProvider(ac.Provider)
		if err != nil {
			return artistRuntime{}, fmt.Errorf("artist %s provider: %w", ac.ID, err)
		}
		return artistRuntime{cfg: ac, artist: artists.TextArtist{Provider: p}, scheduler: sch}, nil
	case "video":
		p, err := providers.NewVideoProvider(ac.Provider)
		if err != nil {
			return artistRuntime{}, fmt.Errorf("artist %s provider: %w", ac.ID, err)
		}
		return artistRuntime{cfg: ac, artist: artists.VideoArtist{Provider: p, Seed: ac.Scheduler.Seed}, scheduler: sch}, nil
	case "image":
		p, err := providers.NewImageProvider(ac.Provider)
		if err != nil {
			return artistRuntime{}, fmt.Errorf("artist %s provider: %w", ac.ID, err)
		}
		return artistRuntime{cfg: ac, artist: artists.ImageArtist{Provider: p, Seed: ac.Scheduler.Seed}, scheduler: sch}, nil
	default:
		return artistRuntime{}, fmt.Errorf("artist %s has unsupported type: %s", ac.ID, ac.Type)
	}
}

func (e *Engine) ValidateUniverse() error {
	_, err := universe.Load(e.cfg.Universe.Path)
	return err
}

func (e *Engine) NextRun(now time.Time) (time.Time, error) {
	ids := e.artistIDs()
	if len(ids) == 0 {
		return time.Time{}, errors.New("no artists configured")
	}
	var best time.Time
	for _, id := range ids {
		next, err := e.NextRunForArtist(id, now)
		if err != nil {
			return time.Time{}, err
		}
		if best.IsZero() || next.Before(best) {
			best = next
		}
	}
	return best, nil
}

func (e *Engine) NextRunForArtist(artistID string, now time.Time) (time.Time, error) {
	rt, ok := e.artists[artistID]
	if !ok {
		return time.Time{}, fmt.Errorf("artist not available: %s", artistID)
	}
	st, err := e.store.LoadSchedulerState(artistID)
	if err == nil && !st.NextRunAt.IsZero() {
		return st.NextRunAt, nil
	}
	return rt.scheduler.NextRun(now)
}

func (e *Engine) GenerateOnce(ctx context.Context, requestedArtist string) (contracts.EpisodeRecord, error) {
	u, err := universe.Load(e.cfg.Universe.Path)
	if err != nil {
		return contracts.EpisodeRecord{}, err
	}
	artistID, rt, err := e.resolveArtist(requestedArtist, time.Now())
	if err != nil {
		return contracts.EpisodeRecord{}, err
	}

	recent, err := e.store.RecentCombos(e.cfg.Generation.RecencyWindow)
	if err != nil {
		return contracts.EpisodeRecord{}, err
	}
	recentByArtist, err := e.store.RecentCombosByArtist(artistID, e.cfg.Generation.RecencyWindow)
	if err != nil {
		return contracts.EpisodeRecord{}, err
	}

	brief, err := e.planner.BuildBrief(u, recent)
	if err != nil {
		return contracts.EpisodeRecord{}, err
	}
	brief.EpisodeType = rt.cfg.Type
	brief.TemplateID = pickTemplateForType(u, rt.cfg.Type, brief.TemplateID)
	brief = enrichBriefWithUniverseData(brief, u)

	uHash, err := util.HashDir(e.cfg.Universe.Path)
	if err != nil {
		return contracts.EpisodeRecord{}, err
	}

	state := contracts.EpisodeState{UniverseVersion: uHash}
	state.RecentEpisodeIDs = combosToKeys(recentByArtist)
	var out contracts.EpisodeOutput
	var retry int
	for {
		out, err = rt.artist.Generate(ctx, brief, state)
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
			Agent:           artistID,
			ArtistID:        artistID,
			ArtistType:      rt.cfg.Type,
			ArtistStyle:     rt.cfg.Style,
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
			Seed:            rt.cfg.Scheduler.Seed,
			RetryCount:      retry,
			Scores: map[string]any{
				"length_ok":         len(out.Content) >= 50 || out.AssetPath != "",
				"contains_entities": containsEntities(out.Content, brief.CharacterIDs),
			},
			State: "generated",
		},
		Context: map[string]any{
			"brief":          brief,
			"universe":       u.Universe.ID,
			"artist_id":      artistID,
			"collab_recent":  recent,
			"artist_recent":  recentByArtist,
			"artist_targets": rt.cfg.PublishTargets,
		},
		Prompt:           out.Prompt,
		ProviderRequest:  memory.SanitizeSecrets(out.ProviderRequest),
		ProviderResponse: memory.SanitizeSecrets(out.ProviderResponse),
		OutputText:       out.Content,
		OutputAssetPath:  out.AssetPath,
	}

	publishedChannels := make([]string, 0)
	publishResult := map[string]any{}
	for _, target := range rt.cfg.PublishTargets {
		ch, ok := e.channels[target]
		if !ok {
			publishResult[target] = map[string]any{"success": false, "error": "channel not configured"}
			continue
		}
		res, err := ch.Publish(ctx, contracts.PublishableContent{
			EpisodeID:  epID,
			ArtistID:   artistID,
			ArtistType: rt.cfg.Type,
			OutputType: brief.EpisodeType,
			Content:    out.Content,
			AssetPath:  out.AssetPath,
			CreatedAt:  now,
		})
		if err != nil {
			publishResult[target] = map[string]any{"success": false, "error": err.Error()}
			continue
		}
		publishedChannels = append(publishedChannels, target)
		publishResult[target] = res
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
	next, err := rt.scheduler.NextRun(now)
	if err == nil {
		_ = e.store.SaveSchedulerState(artistID, memory.SchedulerState{LastRunAt: &now, NextRunAt: next})
	}
	return record, nil
}

func (e *Engine) ShowEpisode(episodeID string) (string, contracts.EpisodeManifest, error) {
	return e.store.FindEpisode(episodeID)
}

func (e *Engine) resolveArtist(requested string, now time.Time) (string, artistRuntime, error) {
	if requested != "" {
		if rt, ok := e.artists[requested]; ok {
			return requested, rt, nil
		}
		for id, rt := range e.artists {
			if rt.cfg.Type == requested {
				return id, rt, nil
			}
		}
		return "", artistRuntime{}, fmt.Errorf("artist not available: %s", requested)
	}
	return e.nextDueArtist(now)
}

func (e *Engine) nextDueArtist(now time.Time) (string, artistRuntime, error) {
	ids := e.artistIDs()
	if len(ids) == 0 {
		return "", artistRuntime{}, errors.New("no artists configured")
	}
	bestID := ""
	var bestAt time.Time
	for _, id := range ids {
		next, err := e.NextRunForArtist(id, now)
		if err != nil {
			return "", artistRuntime{}, err
		}
		if bestID == "" || next.Before(bestAt) {
			bestID = id
			bestAt = next
		}
	}
	rt := e.artists[bestID]
	return bestID, rt, nil
}

func (e *Engine) artistIDs() []string {
	ids := make([]string, 0, len(e.artists))
	for id := range e.artists {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func validateOutput(out contracts.EpisodeOutput, brief contracts.EpisodeBrief) error {
	if brief.EpisodeType == "text" {
		content := strings.TrimSpace(out.Content)
		if len(content) < 40 {
			return errors.New("validation failed: text output too short")
		}
		if !containsEntities(content, brief.CharacterIDs) {
			return errors.New("validation failed: no character mentioned in output")
		}
		if maxLen := templateMaxChars(brief.TemplateBody); maxLen > 0 && len(content) > maxLen {
			return fmt.Errorf("validation failed: text output exceeds template max chars (%d)", maxLen)
		}
	}
	for _, bad := range []string{"api_key", "token=", "secret="} {
		if strings.Contains(strings.ToLower(out.Content), bad) {
			return errors.New("validation failed: forbidden term found")
		}
	}
	for _, rule := range brief.CanonRules {
		const prefix = "FORBIDDEN:"
		if !strings.HasPrefix(strings.ToUpper(rule), prefix) {
			continue
		}
		term := strings.TrimSpace(rule[len(prefix):])
		if term == "" {
			continue
		}
		if strings.Contains(strings.ToLower(out.Content), strings.ToLower(term)) {
			return fmt.Errorf("validation failed: forbidden term found: %s", term)
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

func enrichBriefWithUniverseData(brief contracts.EpisodeBrief, u universe.Universe) contracts.EpisodeBrief {
	if t, ok := u.Templates[brief.TemplateID]; ok {
		brief.TemplateBody = strings.TrimSpace(t.Body)
	}
	if w, ok := u.Worlds[brief.WorldID]; ok {
		brief.WorldData = cloneAnyMap(w.Data)
	}
	if ev, ok := u.Events[brief.EventID]; ok {
		brief.EventData = cloneAnyMap(ev.Data)
	}
	if len(brief.CharacterIDs) > 0 {
		brief.CharacterData = make(map[string]map[string]any, len(brief.CharacterIDs))
		for _, cid := range brief.CharacterIDs {
			if ch, ok := u.Characters[cid]; ok {
				brief.CharacterData[cid] = cloneAnyMap(ch.Data)
			}
		}
	}
	return brief
}

func cloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func templateMaxChars(templateBody string) int {
	const marker = "MAX_CHARS:"
	for _, line := range strings.Split(templateBody, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToUpper(line), marker) {
			continue
		}
		n := strings.TrimSpace(line[len(marker):])
		if n == "" {
			return 0
		}
		var v int
		if _, err := fmt.Sscanf(n, "%d", &v); err == nil && v > 0 {
			return v
		}
	}
	return 0
}

func isProductionEnv(env string) bool {
	e := strings.ToLower(strings.TrimSpace(env))
	return e == "prod" || e == "production"
}

func combosToKeys(combos []planner.HistoryCombo) []string {
	out := make([]string, 0, len(combos))
	for _, c := range combos {
		chars := append([]string(nil), c.CharacterIDs...)
		sort.Strings(chars)
		out = append(out, c.WorldID+"|"+strings.Join(chars, ",")+"|"+c.EventID)
	}
	return out
}
