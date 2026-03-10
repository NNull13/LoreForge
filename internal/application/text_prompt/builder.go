package text_prompt

import (
	"fmt"
	"strings"

	"loreforge/internal/application/text_settings"
	"loreforge/internal/domain/episode"
)

type PromptBundle struct {
	SystemPrompt string
	UserPrompt   string
	JSONSchema   map[string]any
}

func Build(brief episode.Brief, format episode.OutputType, settings text_settings.ResolvedTextSettings) PromptBundle {
	user := buildUserPrompt(brief, format, settings)
	return PromptBundle{
		SystemPrompt: buildSystemPrompt(format, settings),
		UserPrompt:   user,
		JSONSchema:   schemaFor(format, settings),
	}
}

func buildSystemPrompt(format episode.OutputType, settings text_settings.ResolvedTextSettings) string {
	switch format {
	case episode.OutputTypeTweetShort:
		return "You write one concise tweet in canon voice. Universe canon has priority over artistic style. Return valid JSON only."
	case episode.OutputTypeTweetThread:
		return fmt.Sprintf("You write a tweet thread with %d to %d tweets. Each tweet must stand alone and flow sequentially. Universe canon has priority over artistic style. Return valid JSON only.", settings.MinParts, settings.MaxParts)
	case episode.OutputTypeShortStory:
		return "You write a short story scene with setup, escalation, and consequence. Universe canon has priority over artistic style. Return valid JSON only."
	case episode.OutputTypeLongStory:
		return "You write a long-form story with sustained continuity and strong scene progression. Universe canon has priority over artistic style. Return valid JSON only."
	case episode.OutputTypePoem:
		return "You write a poem with vivid imagery and line breaks. Universe canon has priority over artistic style. Return valid JSON only."
	case episode.OutputTypeSongLyrics:
		return "You write song lyrics with Verse and Chorus sections. Universe canon has priority over artistic style. Return valid JSON only."
	case episode.OutputTypeScreenplaySeries:
		return "You write a series screenplay excerpt using scene headings, action, and dialogue. Universe canon has priority over artistic style. Return valid JSON only."
	default:
		return "You write canon-consistent creative text. Universe canon has priority over artistic style. Return valid JSON only."
	}
}

func buildUserPrompt(brief episode.Brief, format episode.OutputType, settings text_settings.ResolvedTextSettings) string {
	sections := []string{
		fmt.Sprintf("Format: %s", format),
		fmt.Sprintf("Artist Identity: %s", artistIdentity(brief.Artist)),
		fmt.Sprintf("Artist Mission: %s", brief.Artist.Mission),
		fmt.Sprintf("World: %s", brief.WorldID),
		fmt.Sprintf("Characters: %s", strings.Join(brief.CharacterIDs, ", ")),
		fmt.Sprintf("Event: %s", brief.EventID),
		fmt.Sprintf("Tone: %s", brief.Tone),
		fmt.Sprintf("Objective: %s", brief.Objective),
		fmt.Sprintf("Rules: %s", strings.Join(brief.CanonRules, " | ")),
		fmt.Sprintf("Word bounds: %d-%d", settings.MinWords, settings.MaxWords),
	}
	if voice := artistVoice(brief.Artist); voice != "" {
		sections = append(sections, "Artist Voice:\n"+voice)
	}
	if len(brief.Artist.PromptingRules) > 0 {
		sections = append(sections, "Artist Prompting Rules:\n- "+strings.Join(brief.Artist.PromptingRules, "\n- "))
	}
	if brief.Artist.NonDiegetic {
		sections = append(sections, "Artist Non-Diegetic Constraint:\nThe artist is an editorial lens, not a character in the story, and must never appear diegetically unless explicitly instructed.")
	}
	if policy := artistSignaturePolicy(brief.Artist); policy != "" {
		sections = append(sections, "Artist Signature Policy:\n"+policy)
	}
	if settings.MinParts > 0 || settings.MaxParts > 0 {
		sections = append(sections, fmt.Sprintf("Part bounds: %d-%d", settings.MinParts, settings.MaxParts))
	}
	if settings.TargetLineCount > 0 {
		sections = append(sections, fmt.Sprintf("Target line count: %d", settings.TargetLineCount))
	}
	if settings.TargetSceneCount > 0 {
		sections = append(sections, fmt.Sprintf("Target scene count: %d", settings.TargetSceneCount))
	}
	if strings.TrimSpace(brief.TemplateBody) != "" {
		sections = append(sections, "Template:\n"+strings.TrimSpace(brief.TemplateBody))
	}
	if refs := formatContinuityReferences(brief.ContinuityReferences); refs != "" {
		sections = append(sections, "Continuity Memories:\n"+refs)
	}
	if refs := formatVisualReferences(brief.VisualReferences); refs != "" {
		sections = append(sections, "Visual Canon References:\n"+refs)
	}
	return strings.Join(sections, "\n\n")
}

func artistIdentity(artist episode.ArtistLens) string {
	if artist.Title != "" {
		return fmt.Sprintf("%s (%s)", artist.Name, artist.Title)
	}
	if artist.Name != "" {
		return artist.Name
	}
	return artist.ID
}

func artistVoice(artist episode.ArtistLens) string {
	if len(artist.Voice) == 0 {
		return ""
	}
	order := []string{"register", "cadence", "diction", "stance", "perspective", "intensity"}
	lines := make([]string, 0, len(order))
	for _, key := range order {
		if value := strings.TrimSpace(artist.Voice[key]); value != "" {
			lines = append(lines, fmt.Sprintf("- %s: %s", key, value))
		}
	}
	return strings.Join(lines, "\n")
}

func artistSignaturePolicy(artist episode.ArtistLens) string {
	if !artist.Presentation.Enabled {
		return "No visible artist framing should be embedded in the body unless channel presentation applies later."
	}
	return fmt.Sprintf("signature_mode=%s framing_mode=%s signature_text=%s", artist.Presentation.SignatureMode, artist.Presentation.FramingMode, artist.Presentation.SignatureText)
}

func schemaFor(format episode.OutputType, settings text_settings.ResolvedTextSettings) map[string]any {
	switch format {
	case episode.OutputTypeTweetShort:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"body": map[string]any{"type": "string"},
			},
			"required": []string{"body"},
		}
	case episode.OutputTypeTweetThread:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"parts": map[string]any{
					"type":     "array",
					"minItems": settings.MinParts,
					"maxItems": settings.MaxParts,
					"items":    map[string]any{"type": "string"},
				},
			},
			"required": []string{"parts"},
		}
	default:
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{"type": "string"},
				"body":  map[string]any{"type": "string"},
			},
			"required": []string{"body"},
		}
	}
}

func formatContinuityReferences(refs []episode.ContinuityReference) string {
	lines := make([]string, 0, len(refs))
	for _, ref := range refs {
		summary := strings.TrimSpace(ref.Summary)
		if summary == "" {
			summary = strings.TrimSpace(ref.OutputText)
		}
		if summary == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- Episode %s: %s", ref.EpisodeID, summary))
	}
	return strings.Join(lines, "\n")
}

func formatVisualReferences(refs []episode.VisualReference) string {
	lines := make([]string, 0, len(refs))
	for _, ref := range refs {
		label := ref.AssetID
		if label == "" {
			label = ref.Path
		}
		if strings.TrimSpace(ref.Description) != "" {
			lines = append(lines, fmt.Sprintf("- %s (%s): %s", label, ref.Usage, ref.Description))
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s (%s)", label, ref.Usage))
	}
	return strings.Join(lines, "\n")
}
