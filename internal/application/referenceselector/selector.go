package referenceselector

import (
	"sort"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/universe"
)

type Result struct {
	VisualReferences     []episode.VisualReference
	ContinuityReferences []episode.ContinuityReference
}

func Select(brief episode.Brief, u universe.Universe, cfg ports.GeneratorConfig, continuity []episode.ContinuityReference) Result {
	visual := collectVisualReferences(brief, u, cfg)
	continuity = limitContinuity(continuity, cfg.MaxContinuityItems)
	switch cfg.ReferenceMode {
	case "creative":
		return Result{}
	case "continuity_only":
		return Result{ContinuityReferences: continuity}
	case "assets_only":
		return Result{VisualReferences: visual}
	default:
		return Result{
			VisualReferences:     visual,
			ContinuityReferences: continuity,
		}
	}
}

func collectVisualReferences(brief episode.Brief, u universe.Universe, cfg ports.GeneratorConfig) []episode.VisualReference {
	candidates := make([]episode.VisualReference, 0)
	appendEntityAssets := func(source, entityType, entityID string, entity universe.Entity) {
		for _, asset := range entity.Assets.Items {
			if len(cfg.AssetUsageAllowlist) > 0 && !contains(cfg.AssetUsageAllowlist, asset.Usage) {
				continue
			}
			candidates = append(candidates, episode.VisualReference{
				Source:      source,
				EntityType:  entityType,
				EntityID:    entityID,
				AssetID:     asset.ID,
				Path:        asset.Path,
				MediaType:   asset.MediaType,
				Usage:       asset.Usage,
				Description: asset.Description,
				Weight:      asset.Weight,
				ModelRole:   modelRole(asset, cfg.ProviderDriver),
			})
		}
	}
	appendEntityAssets("universe_asset", "universe", u.Universe.ID, u.Universe)
	if world, ok := u.Worlds[brief.WorldID]; ok {
		appendEntityAssets("world_asset", "world", brief.WorldID, world)
	}
	for _, id := range brief.CharacterIDs {
		if character, ok := u.Characters[id]; ok {
			appendEntityAssets("character_asset", "character", id, character)
		}
	}
	if event, ok := u.Events[brief.EventID]; ok {
		appendEntityAssets("event_asset", "event", brief.EventID, event)
	}
	if tmpl, ok := u.Templates[brief.TemplateID]; ok {
		appendEntityAssets("template_asset", "template", brief.TemplateID, tmpl)
	}
	if artist, ok := u.Artists[cfg.ProfileID]; ok {
		for _, asset := range artist.Assets.Items {
			if len(cfg.AssetUsageAllowlist) > 0 && !contains(cfg.AssetUsageAllowlist, asset.Usage) {
				continue
			}
			candidates = append(candidates, episode.VisualReference{
				Source:      "artist_asset",
				EntityType:  "artist",
				EntityID:    artist.ID,
				AssetID:     asset.ID,
				Path:        asset.Path,
				MediaType:   asset.MediaType,
				Usage:       asset.Usage,
				Description: asset.Description,
				Weight:      asset.Weight,
				ModelRole:   modelRole(asset, cfg.ProviderDriver),
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Weight != candidates[j].Weight {
			return candidates[i].Weight > candidates[j].Weight
		}
		if sourcePriority(candidates[i].Source) != sourcePriority(candidates[j].Source) {
			return sourcePriority(candidates[i].Source) < sourcePriority(candidates[j].Source)
		}
		if candidates[i].Usage != candidates[j].Usage {
			return candidates[i].Usage < candidates[j].Usage
		}
		return candidates[i].AssetID < candidates[j].AssetID
	})
	limit := cfg.MaxAssetReferences
	if limit <= 0 || limit > len(candidates) {
		limit = len(candidates)
	}
	return append([]episode.VisualReference(nil), candidates[:limit]...)
}

func sourcePriority(source string) int {
	switch source {
	case "world_asset", "character_asset", "event_asset", "template_asset":
		return 0
	case "artist_asset":
		return 1
	case "universe_asset":
		return 2
	default:
		return 3
	}
}

func limitContinuity(in []episode.ContinuityReference, limit int) []episode.ContinuityReference {
	if limit <= 0 || limit > len(in) {
		limit = len(in)
	}
	return append([]episode.ContinuityReference(nil), in[:limit]...)
}

func modelRole(asset universe.Asset, driver string) string {
	if asset.ModelRoles != nil {
		if role := asset.ModelRoles[driver]; role != "" {
			return role
		}
	}
	switch driver {
	case "runway_gen4":
		if asset.Usage == "video_prompt_image" || asset.Usage == "character_reference" {
			return "prompt_image"
		}
		return "reference"
	case "vertex_veo":
		return "asset"
	case "vertex_imagen":
		return "reference"
	case "openai_image":
		return "textual_reference"
	default:
		return ""
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
