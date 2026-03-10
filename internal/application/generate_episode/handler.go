package generate_episode

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"loreforge/internal/application/artist_presentation"
	"loreforge/internal/application/ports"
	"loreforge/internal/application/reference_selector"
	"loreforge/internal/application/scheduler"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/publication"
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
	UniverseRepo      ports.UniverseRepository
	EpisodeRepo       ports.EpisodeRepository
	Scheduler         scheduler.Service
	GeneratorRegistry ports.GeneratorRegistry
	PublisherRegistry ports.PublisherRegistry
	Clock             ports.Clock
	IDGenerator       ports.IDGenerator
	Hasher            ports.Hasher
	Planner           *planner.Planner
}

func (h Handler) Handle(ctx context.Context, req Request) (Result, error) {
	u, err := h.UniverseRepo.Load(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %v", episode.ErrUniverseInvalid, err)
	}
	def, err := h.resolveGenerator(ctx, req.Generator)
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
	brief, err := h.Planner.BuildBriefForType(u, string(def.Config.Type), toPlannerHistory(recent))
	if err != nil {
		return Result{}, err
	}
	brief.TemplateID = resolveTemplateForType(u, string(def.Config.Type), brief.TemplateID)
	if strings.TrimSpace(brief.TemplateID) == "" {
		return Result{}, fmt.Errorf("%w: no template for output type %s", episode.ErrUniverseInvalid, def.Config.Type)
	}
	artistProfile, ok := u.Artists[def.Config.ProfileID]
	if !ok {
		return Result{}, fmt.Errorf("%w: unknown artist profile %s", episode.ErrUniverseInvalid, def.Config.ProfileID)
	}
	brief = enrichBriefWithUniverseData(brief, u)
	brief.Artist = buildArtistLens(artistProfile, def.Config)
	brief.TextConstraints = def.Config.TextConstraints
	continuityRefs, err := h.EpisodeRepo.RecentReferencesByGenerator(ctx, def.Config.ID, def.Config.MaxContinuityItems)
	if err != nil {
		return Result{}, err
	}
	selected := reference_selector.Select(brief, u, def.Config, continuityRefs)
	brief.VisualReferences = selected.VisualReferences
	brief.ContinuityReferences = selected.ContinuityReferences
	brief.Artist.VisualRefs = artistVisualReferences(brief.VisualReferences, brief.Artist.ID)
	if !def.Config.IncludeTextMemories {
		brief.ContinuityReferences = nil
	}
	universeVersion, err := h.Hasher.Hash(ctx)
	if err != nil {
		return Result{}, err
	}
	state := episode.State{
		UniverseVersion:  universeVersion,
		RecentEpisodeIDs: combosToKeys(recentByGenerator),
		Metadata:         map[string]any{},
	}
	var bootstrapImage episode.Output
	if def.Config.ProviderDriver == "runway_gen4" {
		bootstrapImage, err = h.bootstrapRunwayImage(ctx, req, brief, state, def)
		if err != nil {
			return Result{}, err
		}
		if bootstrapImage.AssetPath != "" {
			state.Metadata["prompt_image"] = bootstrapImage.AssetPath
		}
	}
	now := h.Clock.Now()
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
			// Advance the scheduler even on failure to prevent tight retry loops
			if def.Config.SchedulerEnabled {
				_ = h.Scheduler.AdvanceAfterRun(ctx, def, now)
			}
			return Result{}, err
		}
	}
	episodeID := h.IDGenerator.NewEpisodeID()
	record := episode.Record{
		Manifest: episode.Manifest{
			EpisodeID:              episodeID,
			CreatedAt:              now,
			Agent:                  def.Config.ID,
			ArtistID:               def.Config.ID,
			ArtistType:             string(def.Config.Type),
			ArtistStyle:            def.Config.Style,
			ArtistProfileID:        brief.Artist.ID,
			ArtistName:             brief.Artist.Name,
			ArtistRole:             brief.Artist.Role,
			ArtistPresentationMode: brief.Artist.Presentation.SignatureMode + ":" + brief.Artist.Presentation.FramingMode,
			OutputType:             string(brief.EpisodeType),
			UniverseVersion:        universeVersion,
			WorldIDs:               []string{brief.WorldID},
			CharacterIDs:           brief.CharacterIDs,
			EventID:                brief.EventID,
			TemplateID:             brief.TemplateID,
			PromptInput:            brief.Objective,
			PromptFinal:            out.Prompt,
			Provider:               out.Provider,
			Model:                  out.Model,
			Seed:                   def.Config.Seed,
			RetryCount:             retries,
			Scores: map[string]any{
				"length_ok":         len(out.Content) >= 50 || out.AssetPath != "",
				"contains_entities": episode.ContainsEntities(out.Content, brief.CharacterIDs),
			},
			State: string(episode.StatusGenerated),
		},
		Context: map[string]any{
			"brief":                          brief,
			"universe":                       u.Universe.ID,
			"generator_id":                   def.Config.ID,
			"collab_recent":                  recent,
			"generator_recent":               recentByGenerator,
			"generator_targets":              def.Config.PublishTargets,
			"provider_driver":                def.Config.ProviderDriver,
			"artist_profile":                 brief.Artist,
			"artist_prompt_snapshot":         artistPromptSnapshot(brief.Artist),
			"artist_visual_references":       brief.Artist.VisualRefs,
			"artist_non_diegetic":            brief.Artist.NonDiegetic,
			"reference_mode":                 def.Config.ReferenceMode,
			"selected_visual_references":     brief.VisualReferences,
			"selected_continuity_references": brief.ContinuityReferences,
		},
		Prompt:           out.Prompt,
		ProviderRequest:  sanitizeSecrets(out.ProviderRequest),
		ProviderResponse: sanitizeSecrets(out.ProviderResponse),
		OutputText:       out.Content,
		OutputParts:      out.OutputParts(),
		OutputAssetPath:  out.AssetPath,
		ArtistSnapshot:   artistSnapshot(artistProfile),
	}
	record.Publish, record.Presentation = h.publish(ctx, record, brief.Artist, def.Config.PublishTargets)
	record.Context["artist_presentation_applied"] = record.Presentation
	if bootstrapImage.AssetPath != "" {
		record.Context["visual_pipeline"] = map[string]any{
			"bootstrap_image_asset_path": bootstrapImage.AssetPath,
			"bootstrap_provider":         bootstrapImage.Provider,
			"bootstrap_generator_id":     state.Metadata["bootstrap_generator_id"],
		}
	}
	record.Manifest.Channels = publishedChannels(record.Publish)
	publishErr := publishFailureError(record.Publish, def.Config.PublishTargets)
	switch {
	case len(record.Manifest.Channels) > 0:
		record.Manifest.Published = true
		record.Manifest.State = string(episode.StatusPublished)
	case len(def.Config.PublishTargets) > 0:
		record.Manifest.State = string(episode.StatusPublishFailed)
	}
	stored, err := h.EpisodeRepo.Save(ctx, record)
	if err != nil {
		return Result{}, err
	}
	if def.Config.SchedulerEnabled {
		_ = h.Scheduler.AdvanceAfterRun(ctx, def, now)
	}
	result := Result{Record: record, Stored: stored}
	if publishErr != nil {
		return result, publishErr
	}
	return result, nil
}

func (h Handler) bootstrapRunwayImage(ctx context.Context, req Request, brief episode.Brief, state episode.State, def ports.RegisteredGenerator) (episode.Output, error) {
	for _, ref := range brief.VisualReferences {
		if ref.MediaType == "image" && ref.ModelRole == "prompt_image" && strings.TrimSpace(ref.Path) != "" {
			state.Metadata["bootstrap_generator_id"] = "universe_asset"
			return episode.Output{
				AssetPath: ref.Path,
				Provider:  "universe_asset",
				Model:     "reference",
				Prompt:    brief.Objective,
			}, nil
		}
	}
	bootstrapID, _ := def.Config.Options["bootstrap_image_generator"].(string)
	if strings.TrimSpace(bootstrapID) == "" {
		if providerDriver, ok := def.Config.Options["bootstrap_image_provider"].(string); ok && strings.TrimSpace(providerDriver) != "" {
			for _, item := range h.GeneratorRegistry.List() {
				if item.Config.Type == episode.OutputTypeImage && item.Config.ProviderDriver == providerDriver {
					bootstrapID = item.Config.ID
					break
				}
			}
		}
	}
	if strings.TrimSpace(bootstrapID) == "" {
		return episode.Output{}, fmt.Errorf("runway_gen4 requires options.bootstrap_image_generator or options.bootstrap_image_provider")
	}
	bootstrap, ok := h.GeneratorRegistry.GetByID(bootstrapID)
	if !ok {
		return episode.Output{}, fmt.Errorf("bootstrap image generator not available: %s", bootstrapID)
	}
	if bootstrap.Config.Type != episode.OutputTypeImage {
		return episode.Output{}, fmt.Errorf("bootstrap generator %s is not an image generator", bootstrapID)
	}
	state.Metadata["bootstrap_generator_id"] = bootstrapID
	return bootstrap.Generator.Generate(ctx, brief, state)
}

func (h Handler) resolveGenerator(ctx context.Context, requested string) (ports.RegisteredGenerator, error) {
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

	// Track the most-overdue eligible generator (next <= now).
	var dueCandidate ports.RegisteredGenerator
	var dueCandidateTime time.Time
	foundDue := false

	// Track the earliest upcoming run for a useful error message when nothing is due.
	var earliestFutureTime time.Time

	enabled := 0
	for _, item := range items {
		if !item.Config.SchedulerEnabled {
			continue
		}
		enabled++
		next, err := h.Scheduler.NextRun(ctx, item, now)
		if err != nil {
			return ports.RegisteredGenerator{}, err
		}
		if next.After(now) {
			if earliestFutureTime.IsZero() || next.Before(earliestFutureTime) {
				earliestFutureTime = next
			}
			continue
		}
		// Generator is due (next <= now); prefer the most overdue.
		if !foundDue || next.Before(dueCandidateTime) {
			dueCandidate = item
			dueCandidateTime = next
			foundDue = true
		}
	}
	if enabled == 0 {
		return ports.RegisteredGenerator{}, episode.ErrSchedulerDisabled
	}
	if foundDue {
		return dueCandidate, nil
	}
	if !earliestFutureTime.IsZero() {
		return ports.RegisteredGenerator{}, fmt.Errorf("%w: next run at %s", episode.ErrNoGeneratorsDue, earliestFutureTime.Format(time.RFC3339))
	}
	return ports.RegisteredGenerator{}, episode.ErrNoGeneratorsDue
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
	out := make([]string, 0, len(results))
	for key, value := range results {
		if result, ok := value.(publication.Result); ok && result.Success {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func (h Handler) publish(ctx context.Context, record episode.Record, artist episode.ArtistLens, targets []publication.Target) (map[string]any, map[string]any) {
	results := make(map[string]any, len(targets))
	presentation := make(map[string]any, len(targets))
	for _, target := range targets {
		ch := string(target.Channel)
		publisher, ok := h.PublisherRegistry.Get(target.Channel)
		if !ok {
			results[ch] = publication.Result{Channel: ch, Success: false, Message: "channel not configured"}
			presentation[ch] = artist_presentation.Applied{Channel: ch}
			continue
		}
		item, applied := artist_presentation.Compose(publication.Item{
			EpisodeID:     record.Manifest.EpisodeID,
			GeneratorID:   record.Manifest.ArtistID,
			GeneratorType: record.Manifest.ArtistType,
			Target:        target,
			OutputType:    record.Manifest.OutputType,
			Format:        record.Manifest.OutputType,
			Content:       record.OutputText,
			Parts:         append([]string(nil), record.OutputParts...),
			AssetPath:     record.OutputAssetPath,
			CreatedAt:     record.Manifest.CreatedAt,
		}, artist, target.Channel)
		res, err := publisher.Publish(ctx, item)
		presentation[ch] = applied
		if err != nil {
			results[ch] = publication.Result{Channel: ch, Success: false, Message: err.Error()}
			continue
		}
		results[ch] = res
	}
	return results, presentation
}

func resolveTemplateForType(u domainuniverse.Universe, outputType, fallback string) string {
	if tmpl, ok := u.Templates[fallback]; ok && templateMatchesType(tmpl, outputType) {
		return fallback
	}
	matches := make([]string, 0)
	for id, tmpl := range u.Templates {
		if templateMatchesType(tmpl, outputType) {
			matches = append(matches, id)
		}
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func templateMatchesType(tmpl domainuniverse.Entity, outputType string) bool {
	v, ok := tmpl.Data["output_type"].(string)
	return ok && v == outputType
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

func buildArtistLens(profile domainuniverse.Artist, cfg ports.GeneratorConfig) episode.ArtistLens {
	lens := episode.ArtistLens{
		ID:          profile.ID,
		Name:        profile.Name,
		Title:       profile.Title,
		Role:        profile.Role,
		Summary:     profile.Summary,
		Body:        profile.Body,
		NonDiegetic: profile.NonDiegetic,
		Voice: map[string]string{
			"register":    profile.Voice.Register,
			"cadence":     profile.Voice.Cadence,
			"diction":     profile.Voice.Diction,
			"stance":      profile.Voice.Stance,
			"perspective": profile.Voice.Perspective,
			"intensity":   profile.Voice.Intensity,
		},
		Mission:        profile.Mission.Purpose,
		PromptingRules: append([]string(nil), profile.Prompting.SystemRules...),
		TonalBiases:    append([]string(nil), profile.Prompting.TonalBiases...),
		LexicalCues:    append([]string(nil), profile.Prompting.LexicalCues...),
		Forbidden:      append([]string(nil), profile.Prompting.Forbidden...),
		Presentation: episode.ArtistPresentationSnapshot{
			Enabled:         profile.Presentation.Enabled,
			SignatureMode:   profile.Presentation.SignatureMode,
			SignatureText:   profile.Presentation.SignatureText,
			FramingMode:     profile.Presentation.FramingMode,
			IntroTemplate:   profile.Presentation.IntroTemplate,
			OutroTemplate:   profile.Presentation.OutroTemplate,
			AllowedChannels: append([]string(nil), profile.Presentation.AllowedChannels...),
		},
	}
	applyArtistOverrides(&lens, cfg)
	return lens
}

func applyArtistOverrides(lens *episode.ArtistLens, cfg ports.GeneratorConfig) {
	if len(cfg.PromptOverrides) > 0 {
		lens.PromptingRules = append(lens.PromptingRules, toStringSlice(cfg.PromptOverrides["extra_system_rules"])...)
		lens.TonalBiases = append(lens.TonalBiases, toStringSlice(cfg.PromptOverrides["tonal_biases"])...)
		lens.LexicalCues = append(lens.LexicalCues, toStringSlice(cfg.PromptOverrides["lexical_cues"])...)
		lens.Forbidden = append(lens.Forbidden, toStringSlice(cfg.PromptOverrides["forbidden"])...)
	}
	if len(cfg.PresentationOverrides) > 0 {
		if enabled, ok := cfg.PresentationOverrides["enabled"].(bool); ok {
			lens.Presentation.Enabled = enabled
		}
		if value, ok := cfg.PresentationOverrides["signature_mode"].(string); ok && value != "" {
			lens.Presentation.SignatureMode = value
		}
		if value, ok := cfg.PresentationOverrides["signature_text"].(string); ok && value != "" {
			lens.Presentation.SignatureText = value
		}
		if value, ok := cfg.PresentationOverrides["framing_mode"].(string); ok && value != "" {
			lens.Presentation.FramingMode = value
		}
		if value, ok := cfg.PresentationOverrides["intro_template"].(string); ok && value != "" {
			lens.Presentation.IntroTemplate = value
		}
		if value, ok := cfg.PresentationOverrides["outro_template"].(string); ok && value != "" {
			lens.Presentation.OutroTemplate = value
		}
		if value := toStringSlice(cfg.PresentationOverrides["allowed_channels"]); len(value) > 0 {
			lens.Presentation.AllowedChannels = value
		}
	}
}

func artistVisualReferences(refs []episode.VisualReference, artistID string) []episode.VisualReference {
	out := make([]episode.VisualReference, 0)
	for _, ref := range refs {
		if ref.EntityType == "artist" && ref.EntityID == artistID {
			out = append(out, ref)
		}
	}
	return out
}

func artistSnapshot(artist domainuniverse.Artist) map[string]any {
	return map[string]any{
		"id":            artist.ID,
		"name":          artist.Name,
		"title":         artist.Title,
		"role":          artist.Role,
		"summary":       artist.Summary,
		"body":          artist.Body,
		"non_diegetic": artist.NonDiegetic,
		"voice":         artist.Voice,
		"mission":       artist.Mission,
		"prompting":     artist.Prompting,
		"presentation":  artist.Presentation,
		"future":        artist.Future,
		"assets":        artist.Assets,
	}
}

func artistPromptSnapshot(artist episode.ArtistLens) map[string]any {
	return map[string]any{
		"id":              artist.ID,
		"name":            artist.Name,
		"mission":         artist.Mission,
		"voice":           artist.Voice,
		"prompting_rules": append([]string(nil), artist.PromptingRules...),
		"tonal_biases":    append([]string(nil), artist.TonalBiases...),
		"lexical_cues":    append([]string(nil), artist.LexicalCues...),
		"forbidden":       append([]string(nil), artist.Forbidden...),
		"presentation":    artist.Presentation,
	}
}

func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
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
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = sanitizeSecretValue(key, value)
	}
	return out
}

func sanitizeSecretValue(key string, value any) any {
	if secretKey(key) {
		return "***"
	}
	return sanitizeNestedSecrets(value)
}

func sanitizeNestedSecrets(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			out[key] = sanitizeSecretValue(key, nested)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, sanitizeNestedSecrets(item))
		}
		return out
	default:
		return value
	}
}

// exactSecretKeys is a package-level set of exact key names that should be redacted.
// Defined at package level to avoid re-allocating the map on every call.
var exactSecretKeys = map[string]bool{
	"authorization": true,
	"api_key":       true,
	"api_key_env":   true,
	"secret":        true,
	"token":         true,
	"password":      true,
	"passwd":        true,
	"bearer_token":  true,
	"access_token":  true,
	"refresh_token": true,
	"private_key":   true,
	"client_secret": true,
	"credential":    true,
	"credentials":   true,
}

func secretKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if exactSecretKeys[lower] {
		return true
	}
	for _, suffix := range []string{"_key", "_token", "_secret", "_password", "_passwd", "_credential"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	for _, prefix := range []string{"api_key", "secret_", "private_"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func publishFailureError(results map[string]any, targets []publication.Target) error {
	if len(targets) == 0 || len(publishedChannels(results)) > 0 {
		return nil
	}
	failures := make([]string, 0, len(targets))
	for _, target := range targets {
		targetName := string(target.Channel)
		if target.Account != "" {
			targetName += "(" + target.Account + ")"
		}
		value, ok := results[string(target.Channel)]
		if !ok {
			failures = append(failures, targetName+": publish result missing")
			continue
		}
		result, ok := value.(publication.Result)
		if !ok {
			failures = append(failures, targetName+": publish failed")
			continue
		}
		if !result.Success {
			failures = append(failures, targetName+": "+firstNonEmpty(result.Message, "publish failed"))
		}
	}
	if len(failures) == 0 {
		return episode.ErrPublishFailed
	}
	sort.Strings(failures)
	return fmt.Errorf("%w: %s", episode.ErrPublishFailed, strings.Join(failures, "; "))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
