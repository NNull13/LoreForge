package universe

import "fmt"

func Validate(u Universe) error {
	if u.Universe.ID == "" {
		return fmt.Errorf("universe.md missing id")
	}
	if u.Universe.Type != "universe" {
		return fmt.Errorf("universe.md type must be 'universe'")
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
	return nil
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
