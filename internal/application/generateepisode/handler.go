package generateepisode

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/publication"
	"loreforge/internal/domain/scheduling"
	domainuniverse "loreforge/internal/domain/universe"
	"loreforge/internal/planner"
)

type Request struct {
	Generator     string
	MaxRetries    int
	RecencyWindow int
}

type Result struct {
	Record episode.Record
	Stored episode.StoredRecord
}

type Handler struct {
	UniverseRepo       ports.UniverseRepository
	EpisodeRepo        ports.EpisodeRepository
	SchedulerStateRepo ports.SchedulerStateRepository
	GeneratorRegistry  ports.GeneratorRegistry
	PublisherRegistry  ports.PublisherRegistry
	Clock              ports.Clock
	IDGenerator        ports.IDGenerator
	Hasher             ports.Hasher
	Planner            *planner.Planner
}

func (h Handler) Handle(ctx context.Context, req Request) (Result, error) {
	u, err := h.UniverseRepo.Load(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", episode.ErrUniverseInvalid, err)
	}
	def, err := h.resolveGenerator(req.Generator, ctx)
	if err != nil {
		return Result{}, err
	}
	recent, err := h.EpisodeRepo.RecentCombos(ctx, req.RecencyWindow)
	if err != nil {
		return Result{}, err
	}
	recentByGenerator, err := h.EpisodeRepo.RecentCombosByGenerator(ctx, def.Config.ID, req.RecencyWindow)
	if err != nil {
		return Result{}, err
	}
	brief, err := h.Planner.BuildBrief(u, toPlannerHistory(recent))
	if err != nil {
		return Result{}, err
	}
	brief.EpisodeType = def.Config.Type
	brief.TemplateID = pickTemplateForType(u, string(def.Config.Type), brief.TemplateID)
	brief = enrichBriefWithUniverseData(brief, u)
	universeVersion, err := h.Hasher.Hash(ctx)
	if err != nil {
		return Result{}, err
	}
	state := episode.State{
		UniverseVersion:  universeVersion,
		RecentEpisodeIDs: combosToKeys(recentByGenerator),
	}
	var out episode.Output
	retries := 0
	for {
		out, err = def.Generator.Generate(ctx, brief, state)
		if err == nil {
			if err = episode.ValidateOutput(out, brief); err == nil {
				break
			}
		}
		retries++
		if retries > req.MaxRetries {
			return Result{}, err
		}
	}
	now := h.Clock.Now()
	episodeID := h.IDGenerator.NewEpisodeID()
	record := episode.Record{
		Manifest: episode.Manifest{
			EpisodeID:       episodeID,
			CreatedAt:       now,
			Agent:           def.Config.ID,
			ArtistID:        def.Config.ID,
			ArtistType:      string(def.Config.Type),
			ArtistStyle:     def.Config.Style,
			OutputType:      string(brief.EpisodeType),
			UniverseVersion: universeVersion,
			WorldIDs:        []string{brief.WorldID},
			CharacterIDs:    brief.CharacterIDs,
			EventID:         brief.EventID,
			TemplateID:      brief.TemplateID,
			PromptInput:     brief.Objective,
			PromptFinal:     out.Prompt,
			Provider:        out.Provider,
			Model:           out.Model,
			Seed:            def.Config.Seed,
			RetryCount:      retries,
			Scores: map[string]any{
				"length_ok":         len(out.Content) >= 50 || out.AssetPath != "",
				"contains_entities": episode.ContainsEntities(out.Content, brief.CharacterIDs),
			},
			State: string(episode.StatusGenerated),
		},
		Context: map[string]any{
			"brief":             brief,
			"universe":          u.Universe.ID,
			"generator_id":      def.Config.ID,
			"artist_id":         def.Config.ID,
			"collab_recent":     recent,
			"generator_recent":  recentByGenerator,
			"artist_recent":     recentByGenerator,
			"generator_targets": def.Config.PublishTargets,
			"artist_targets":    def.Config.PublishTargets,
		},
		Prompt:           out.Prompt,
		ProviderRequest:  sanitizeSecrets(out.ProviderRequest),
		ProviderResponse: sanitizeSecrets(out.ProviderResponse),
		OutputText:       out.Content,
		OutputAssetPath:  out.AssetPath,
	}
	record.Publish = h.publish(ctx, record, def.Config.PublishTargets)
	record.Manifest.Channels = publishedChannels(record.Publish)
	record.Manifest.Published = len(record.Manifest.Channels) > 0
	if record.Manifest.Published {
		record.Manifest.State = string(episode.StatusPublished)
	}
	stored, err := h.EpisodeRepo.Save(ctx, record)
	if err != nil {
		return Result{}, err
	}
	scheduler, err := scheduling.NewScheduler(def.Config.Scheduler)
	if err == nil {
		nextRun, nextErr := scheduler.NextRun(now)
		if nextErr == nil {
			_ = h.SchedulerStateRepo.Save(ctx, def.Config.ID, scheduling.State{
				LastRunAt: &now,
				NextRunAt: nextRun,
			})
		}
	}
	return Result{Record: record, Stored: stored}, nil
}

func (h Handler) resolveGenerator(requested string, ctx context.Context) (ports.RegisteredGenerator, error) {
	if requested != "" {
		if def, ok := h.GeneratorRegistry.GetByID(requested); ok {
			return def, nil
		}
		if def, ok := h.GeneratorRegistry.GetByType(episode.OutputType(requested)); ok {
			return def, nil
		}
		return ports.RegisteredGenerator{}, fmt.Errorf("%w: %s", episode.ErrGeneratorUnavailable, requested)
	}
	return h.nextDueGenerator(ctx)
}

func (h Handler) nextDueGenerator(ctx context.Context) (ports.RegisteredGenerator, error) {
	items := h.GeneratorRegistry.List()
	if len(items) == 0 {
		return ports.RegisteredGenerator{}, episode.ErrNoGeneratorsAvailable
	}
	now := h.Clock.Now()
	best := items[0]
	var bestTime time.Time
	for _, item := range items {
		next, err := nextRunForGenerator(ctx, h.SchedulerStateRepo, item, now)
		if err != nil {
			return ports.RegisteredGenerator{}, err
		}
		if bestTime.IsZero() || next.Before(bestTime) {
			best = item
			bestTime = next
		}
	}
	return best, nil
}

func nextRunForGenerator(ctx context.Context, repo ports.SchedulerStateRepository, def ports.RegisteredGenerator, now time.Time) (time.Time, error) {
	state, err := repo.Load(ctx, def.Config.ID)
	if err == nil && !state.NextRunAt.IsZero() {
		return state.NextRunAt, nil
	}
	scheduler, err := scheduling.NewScheduler(def.Config.Scheduler)
	if err != nil {
		return time.Time{}, err
	}
	next, err := scheduler.NextRun(now)
	if err != nil {
		return time.Time{}, err
	}
	return next, nil
}

func toPlannerHistory(combos []episode.Combo) []planner.HistoryCombo {
	out := make([]planner.HistoryCombo, 0, len(combos))
	for _, combo := range combos {
		out = append(out, planner.HistoryCombo{
			WorldID:      combo.WorldID,
			CharacterIDs: combo.CharacterIDs,
			EventID:      combo.EventID,
		})
	}
	return out
}

func publishedChannels(results map[string]any) []string {
	out := make([]string, 0)
	for key, value := range results {
		result, ok := value.(publication.Result)
		if ok && result.Success {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func (h Handler) publish(ctx context.Context, record episode.Record, targets []publication.ChannelName) map[string]any {
	results := make(map[string]any, len(targets))
	for _, target := range targets {
		publisher, ok := h.PublisherRegistry.Get(target)
		if !ok {
			results[string(target)] = map[string]any{"success": false, "error": "channel not configured"}
			continue
		}
		res, err := publisher.Publish(ctx, publication.Item{
			EpisodeID:     record.Manifest.EpisodeID,
			GeneratorID:   record.Manifest.ArtistID,
			ArtistID:      record.Manifest.ArtistID,
			GeneratorType: record.Manifest.ArtistType,
			ArtistType:    record.Manifest.ArtistType,
			OutputType:    record.Manifest.OutputType,
			Content:       record.OutputText,
			AssetPath:     record.OutputAssetPath,
			CreatedAt:     record.Manifest.CreatedAt,
		})
		if err != nil {
			results[string(target)] = map[string]any{"success": false, "error": err.Error()}
			continue
		}
		results[string(target)] = res
	}
	return results
}

func pickTemplateForType(u domainuniverse.Universe, outputType, fallback string) string {
	for id, tmpl := range u.Templates {
		if v, ok := tmpl.Data["output_type"].(string); ok && v == outputType {
			return id
		}
	}
	return fallback
}

func enrichBriefWithUniverseData(brief episode.Brief, u domainuniverse.Universe) episode.Brief {
	if tmpl, ok := u.Templates[brief.TemplateID]; ok {
		brief.TemplateBody = strings.TrimSpace(tmpl.Body)
	}
	if world, ok := u.Worlds[brief.WorldID]; ok {
		brief.WorldData = cloneAnyMap(world.Data)
	}
	if eventData, ok := u.Events[brief.EventID]; ok {
		brief.EventData = cloneAnyMap(eventData.Data)
	}
	if len(brief.CharacterIDs) > 0 {
		brief.CharacterData = make(map[string]map[string]any, len(brief.CharacterIDs))
		for _, characterID := range brief.CharacterIDs {
			if character, ok := u.Characters[characterID]; ok {
				brief.CharacterData[characterID] = cloneAnyMap(character.Data)
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
	for key, value := range in {
		out[key] = value
	}
	return out
}

func combosToKeys(combos []episode.Combo) []string {
	out := make([]string, 0, len(combos))
	for _, combo := range combos {
		characters := append([]string(nil), combo.CharacterIDs...)
		sort.Strings(characters)
		out = append(out, combo.WorldID+"|"+strings.Join(characters, ",")+"|"+combo.EventID)
	}
	return out
}

func sanitizeSecrets(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") {
			out[key] = "***"
			continue
		}
		out[key] = value
	}
	return out
}
