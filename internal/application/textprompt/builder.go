package textprompt

import (
	"fmt"
	"strings"

	"loreforge/internal/application/textsettings"
	"loreforge/internal/domain/episode"
)

type PromptBundle struct {
	SystemPrompt string
	UserPrompt   string
	JSONSchema   map[string]any
}

func Build(brief episode.Brief, format episode.OutputType, settings textsettings.ResolvedTextSettings) PromptBundle {
	user := buildUserPrompt(brief, format, settings)
	return PromptBundle{
		SystemPrompt: buildSystemPrompt(format, settings),
		UserPrompt:   user,
		JSONSchema:   schemaFor(format, settings),
	}
}

func buildSystemPrompt(format episode.OutputType, settings textsettings.ResolvedTextSettings) string {
	switch format {
	case episode.OutputTypeTweetShort:
		return "You write one concise tweet in canon voice. Return valid JSON only."
	case episode.OutputTypeTweetThread:
		return fmt.Sprintf("You write a tweet thread with %d to %d tweets. Each tweet must stand alone and flow sequentially. Return valid JSON only.", settings.MinParts, settings.MaxParts)
	case episode.OutputTypeShortStory:
		return "You write a short story scene with setup, escalation, and consequence. Return valid JSON only."
	case episode.OutputTypeLongStory:
		return "You write a long-form story with sustained continuity and strong scene progression. Return valid JSON only."
	case episode.OutputTypePoem:
		return "You write a poem with vivid imagery and line breaks. Return valid JSON only."
	case episode.OutputTypeSongLyrics:
		return "You write song lyrics with Verse and Chorus sections. Return valid JSON only."
	case episode.OutputTypeScreenplaySeries:
		return "You write a series screenplay excerpt using scene headings, action, and dialogue. Return valid JSON only."
	default:
		return "You write canon-consistent creative text. Return valid JSON only."
	}
}

func buildUserPrompt(brief episode.Brief, format episode.OutputType, settings textsettings.ResolvedTextSettings) string {
	sections := []string{
		fmt.Sprintf("Format: %s", format),
		fmt.Sprintf("World: %s", brief.WorldID),
		fmt.Sprintf("Characters: %s", strings.Join(brief.CharacterIDs, ", ")),
		fmt.Sprintf("Event: %s", brief.EventID),
		fmt.Sprintf("Tone: %s", brief.Tone),
		fmt.Sprintf("Objective: %s", brief.Objective),
		fmt.Sprintf("Rules: %s", strings.Join(brief.CanonRules, " | ")),
		fmt.Sprintf("Word bounds: %d-%d", settings.MinWords, settings.MaxWords),
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

func schemaFor(format episode.OutputType, settings textsettings.ResolvedTextSettings) map[string]any {
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
