package referenceselector

import (
	"testing"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/episode"
	domainuniverse "loreforge/internal/domain/universe"
)

func TestSelectModes(t *testing.T) {
	t.Parallel()

	brief := episode.Brief{
		WorldID:      "glass-kingdom",
		CharacterIDs: []string{"red-wanderer"},
		EventID:      "lost-artifact",
		TemplateID:   "short-story",
	}
	u := domainuniverse.Universe{
		Universe: domainuniverse.Entity{
			ID:   "no-name-universe",
			Type: "universe",
			Assets: domainuniverse.AssetSet{Items: []domainuniverse.Asset{
				{ID: "skyline", FileName: "skyline.png", Path: "/tmp/skyline.png", MediaType: "image", Usage: "environment_reference", Weight: 60},
			}},
		},
		Worlds: map[string]domainuniverse.Entity{
			"glass-kingdom": {
				ID:   "glass-kingdom",
				Type: "world",
				Assets: domainuniverse.AssetSet{Items: []domainuniverse.Asset{
					{ID: "city", FileName: "city.png", Path: "/tmp/city.png", MediaType: "image", Usage: "environment_reference", Weight: 80},
				}},
			},
		},
		Characters: map[string]domainuniverse.Entity{
			"red-wanderer": {
				ID:   "red-wanderer",
				Type: "character",
				Assets: domainuniverse.AssetSet{Items: []domainuniverse.Asset{
					{ID: "hero", FileName: "hero.png", Path: "/tmp/hero.png", MediaType: "image", Usage: "video_prompt_image", Weight: 100},
				}},
			},
		},
		Events:    map[string]domainuniverse.Entity{"lost-artifact": {ID: "lost-artifact", Type: "event"}},
		Templates: map[string]domainuniverse.Entity{"short-story": {ID: "short-story", Type: "template"}},
	}
	continuity := []episode.ContinuityReference{
		{EpisodeID: "ep-1", Summary: "The wanderer crossed the ash bridge."},
		{EpisodeID: "ep-0", Summary: "The architect hid the map in glass."},
	}

	result := Select(brief, u, ports.GeneratorConfig{
		ProviderDriver:     "runway_gen4",
		ReferenceMode:      "continuity_plus_assets",
		MaxContinuityItems: 1,
		MaxAssetReferences: 2,
	}, continuity)
	if len(result.VisualReferences) != 2 {
		t.Fatalf("unexpected visual ref count: %d", len(result.VisualReferences))
	}
	if result.VisualReferences[0].ModelRole != "prompt_image" {
		t.Fatalf("expected prompt_image role, got %s", result.VisualReferences[0].ModelRole)
	}
	if len(result.ContinuityReferences) != 1 || result.ContinuityReferences[0].EpisodeID != "ep-1" {
		t.Fatalf("unexpected continuity refs: %#v", result.ContinuityReferences)
	}

	result = Select(brief, u, ports.GeneratorConfig{
		ProviderDriver:      "runway_gen4",
		ReferenceMode:       "assets_only",
		MaxAssetReferences:  5,
		AssetUsageAllowlist: []string{"video_prompt_image"},
	}, continuity)
	if len(result.ContinuityReferences) != 0 {
		t.Fatal("expected no continuity refs for assets_only")
	}
	if len(result.VisualReferences) != 1 || result.VisualReferences[0].AssetID != "hero" {
		t.Fatalf("unexpected allowlisted refs: %#v", result.VisualReferences)
	}

	result = Select(brief, u, ports.GeneratorConfig{
		ProviderDriver: "runway_gen4",
		ReferenceMode:  "creative",
	}, continuity)
	if len(result.VisualReferences) != 0 || len(result.ContinuityReferences) != 0 {
		t.Fatal("expected creative mode to suppress all refs")
	}
}
