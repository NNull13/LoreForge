package universe

import "testing"

func TestValidateAcceptsAssets(t *testing.T) {
	t.Parallel()

	u := Universe{
		Universe: Entity{
			ID:   "no-name-universe",
			Type: "universe",
			Path: "/tmp/universe/universe.md",
			Assets: AssetSet{Items: []Asset{
				{ID: "skyline", FileName: "skyline.png", Path: "/tmp/universe/skyline.png", MediaType: "image", Usage: "environment_reference", Weight: 100},
			}},
		},
		Artists: map[string]Artist{
			"ash-chorister": {
				ID:          "ash-chorister",
				Name:        "Ash Chorister",
				Role:        "chronicler",
				Summary:     "A solemn editorial witness.",
				NonDiegetic: true,
				Prompting:   ArtistPrompting{SystemIdentity: "You are The Ash Chorister."},
				Presentation: ArtistPresentation{
					Enabled:       true,
					SignatureMode: "presentation_only",
					FramingMode:   "none",
				},
			},
		},
		Rules: map[string]Entity{
			"global-rules": {ID: "global-rules", Type: "rule", Path: "/tmp/rules/global-rules/global-rules.md"},
		},
		Worlds: map[string]Entity{
			"glass-kingdom": {ID: "glass-kingdom", Type: "world", Path: "/tmp/worlds/glass-kingdom/glass-kingdom.md"},
		},
		Characters: map[string]Entity{
			"red-wanderer": {ID: "red-wanderer", Type: "character", Path: "/tmp/characters/red-wanderer/red-wanderer.md"},
		},
		Events: map[string]Entity{
			"lost-artifact": {ID: "lost-artifact", Type: "event", Path: "/tmp/events/lost-artifact/lost-artifact.md"},
		},
		Templates: map[string]Entity{
			"short-story": {ID: "short-story", Type: "template", Path: "/tmp/templates/short-story/short-story.md"},
		},
	}

	if err := Validate(u); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateRejectsInvalidAssetUsage(t *testing.T) {
	t.Parallel()

	u := Universe{
		Universe: Entity{
			ID:   "no-name-universe",
			Type: "universe",
			Path: "/tmp/universe/universe.md",
			Assets: AssetSet{Items: []Asset{
				{ID: "bad", FileName: "bad.png", Path: "/tmp/universe/bad.png", MediaType: "image", Usage: "bad_usage"},
			}},
		},
		Artists: map[string]Artist{
			"ash-chorister": {
				ID:          "ash-chorister",
				Name:        "Ash Chorister",
				Role:        "chronicler",
				Summary:     "A solemn editorial witness.",
				NonDiegetic: true,
				Prompting:   ArtistPrompting{SystemIdentity: "You are The Ash Chorister."},
				Presentation: ArtistPresentation{
					Enabled:       true,
					SignatureMode: "presentation_only",
					FramingMode:   "none",
				},
			},
		},
		Rules:      map[string]Entity{"global-rules": {ID: "global-rules", Type: "rule", Path: "/tmp/rules/global-rules/global-rules.md"}},
		Worlds:     map[string]Entity{"glass-kingdom": {ID: "glass-kingdom", Type: "world", Path: "/tmp/worlds/glass-kingdom/glass-kingdom.md"}},
		Characters: map[string]Entity{"red-wanderer": {ID: "red-wanderer", Type: "character", Path: "/tmp/characters/red-wanderer/red-wanderer.md"}},
		Events:     map[string]Entity{"lost-artifact": {ID: "lost-artifact", Type: "event", Path: "/tmp/events/lost-artifact/lost-artifact.md"}},
		Templates:  map[string]Entity{"short-story": {ID: "short-story", Type: "template", Path: "/tmp/templates/short-story/short-story.md"}},
	}

	if err := Validate(u); err == nil {
		t.Fatal("expected validation error for invalid asset usage")
	}
}
