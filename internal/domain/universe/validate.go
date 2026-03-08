package universe

import (
	"fmt"
	"path/filepath"
	"strings"
)

func Validate(u Universe) error {
	if u.Universe.ID == "" {
		return fmt.Errorf("universe/universe.md missing id")
	}
	if u.Universe.Type != "universe" {
		return fmt.Errorf("universe/universe.md type must be 'universe'")
	}
	if len(u.Worlds) == 0 {
		return fmt.Errorf("at least one world is required")
	}
	if len(u.Characters) == 0 {
		return fmt.Errorf("at least one character is required")
	}
	if len(u.Events) == 0 {
		return fmt.Errorf("at least one event is required")
	}
	if len(u.Templates) == 0 {
		return fmt.Errorf("at least one template is required")
	}
	for _, event := range u.Events {
		if worlds, ok := event.Data["compatible_worlds"]; ok {
			for _, id := range ToStringSlice(worlds) {
				if _, exists := u.Worlds[id]; !exists {
					return fmt.Errorf("event %s references unknown world %s", event.ID, id)
				}
			}
		}
		if chars, ok := event.Data["compatible_characters"]; ok {
			for _, id := range ToStringSlice(chars) {
				if _, exists := u.Characters[id]; !exists {
					return fmt.Errorf("event %s references unknown character %s", event.ID, id)
				}
			}
		}
	}
	for _, character := range u.Characters {
		if worlds, ok := character.Data["world_affinities"]; ok {
			for _, id := range ToStringSlice(worlds) {
				if _, exists := u.Worlds[id]; !exists {
					return fmt.Errorf("character %s references unknown world %s", character.ID, id)
				}
			}
		}
	}
	for _, entity := range allEntities(u) {
		if err := validateAssets(entity); err != nil {
			return err
		}
	}
	return nil
}

func validateAssets(entity Entity) error {
	seen := map[string]bool{}
	for _, asset := range entity.Assets.Items {
		if strings.TrimSpace(asset.ID) == "" {
			return fmt.Errorf("entity %s has asset without id", entity.ID)
		}
		if seen[asset.ID] {
			return fmt.Errorf("entity %s has duplicate asset id %s", entity.ID, asset.ID)
		}
		seen[asset.ID] = true
		if !validAssetUsage(asset.Usage) {
			return fmt.Errorf("entity %s asset %s has invalid usage %s", entity.ID, asset.ID, asset.Usage)
		}
		if asset.Weight < 0 {
			return fmt.Errorf("entity %s asset %s has negative weight", entity.ID, asset.ID)
		}
		if asset.Path == "" {
			return fmt.Errorf("entity %s asset %s missing path", entity.ID, asset.ID)
		}
		if asset.MediaType == "" {
			return fmt.Errorf("entity %s asset %s missing media type", entity.ID, asset.ID)
		}
		if !mediaTypeMatchesExtension(asset.MediaType, asset.Path) {
			return fmt.Errorf("entity %s asset %s media_type does not match extension", entity.ID, asset.ID)
		}
		if asset.Usage == "video_prompt_image" && asset.MediaType != "image" {
			return fmt.Errorf("entity %s asset %s usage video_prompt_image requires media_type=image", entity.ID, asset.ID)
		}
		for driver := range asset.ModelRoles {
			if !validProviderDriver(driver) {
				return fmt.Errorf("entity %s asset %s has invalid model role driver %s", entity.ID, asset.ID, driver)
			}
		}
	}
	return nil
}

func allEntities(u Universe) []Entity {
	out := []Entity{u.Universe}
	for _, item := range u.Rules {
		out = append(out, item)
	}
	for _, item := range u.Worlds {
		out = append(out, item)
	}
	for _, item := range u.Characters {
		out = append(out, item)
	}
	for _, item := range u.Events {
		out = append(out, item)
	}
	for _, item := range u.Templates {
		out = append(out, item)
	}
	return out
}

func validAssetUsage(v string) bool {
	switch v {
	case "character_reference", "style_reference", "environment_reference", "prop_reference", "pose_reference", "continuity_reference", "video_prompt_image":
		return true
	default:
		return false
	}
}

func validProviderDriver(v string) bool {
	switch v {
	case "mock", "openai_image", "vertex_imagen", "vertex_veo", "runway_gen4":
		return true
	default:
		return false
	}
}

func mediaTypeMatchesExtension(mediaType string, path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch mediaType {
	case "image":
		return ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".webp"
	case "video":
		return ext == ".mp4" || ext == ".mov" || ext == ".webm"
	default:
		return false
	}
}

func ToStringSlice(v any) []string {
	list, ok := v.([]any)
	if !ok {
		if strList, ok := v.([]string); ok {
			return strList
		}
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}
