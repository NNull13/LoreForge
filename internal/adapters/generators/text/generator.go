package text

import (
	"context"
	"fmt"
	"strings"

	providercontracts "loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/domain/episode"
)

type Generator struct {
	GeneratorID string
	Provider    providercontracts.TextProvider
}

func (g Generator) ID() string { return g.GeneratorID }

func (g Generator) Type() episode.OutputType { return episode.OutputTypeText }

func (g Generator) Generate(ctx context.Context, brief episode.Brief, _ episode.State) (episode.Output, error) {
	prompt := buildPrompt(brief)
	resp, err := g.Provider.GenerateText(ctx, providercontracts.TextRequest{
		Prompt:      prompt,
		Temperature: 0.8,
		MaxTokens:   500,
	})
	if err != nil {
		return episode.Output{}, err
	}
	return episode.Output{
		Content:  strings.TrimSpace(resp.Content),
		Provider: g.Provider.Name(),
		Model:    resp.Model,
		Prompt:   prompt,
		ProviderRequest: map[string]any{
			"prompt": prompt,
		},
		ProviderResponse: map[string]any{
			"content": resp.Content,
		},
	}, nil
}

func buildPrompt(brief episode.Brief) string {
	contextBlock := fmt.Sprintf(
		"Context:\n- Type: %s\n- World: %s\n- Characters: %s\n- Event: %s\n- Tone: %s\n- Objective: %s\n- Rules: %s\n- WorldData: %v\n- EventData: %v\n- CharacterData: %v",
		brief.EpisodeType,
		brief.WorldID,
		strings.Join(brief.CharacterIDs, ", "),
		brief.EventID,
		brief.Tone,
		brief.Objective,
		strings.Join(brief.CanonRules, " | "),
		brief.WorldData,
		brief.EventData,
		brief.CharacterData,
	)
	if refs := formatContinuityReferences(brief.ContinuityReferences); refs != "" {
		contextBlock += "\n\nContinuity Memories:\n" + refs
	}
	if refs := formatVisualReferences(brief.VisualReferences); refs != "" {
		contextBlock += "\n\nVisual Canon References:\n" + refs
	}
	if strings.TrimSpace(brief.TemplateBody) != "" {
		return fmt.Sprintf("%s\n\n%s", brief.TemplateBody, contextBlock)
	}
	additions := promptAdditions(brief)
	return fmt.Sprintf(
		"Create a short narrative. Type: %s. World: %s. Characters: %s. Event: %s. Tone: %s. Objective: %s. Rules: %s%s",
		brief.EpisodeType,
		brief.WorldID,
		strings.Join(brief.CharacterIDs, ", "),
		brief.EventID,
		brief.Tone,
		brief.Objective,
		strings.Join(brief.CanonRules, " | "),
		additions,
	)
}

func promptAdditions(brief episode.Brief) string {
	sections := make([]string, 0, 2)
	if refs := formatContinuityReferences(brief.ContinuityReferences); refs != "" {
		sections = append(sections, "Continuity Memories:\n"+refs)
	}
	if refs := formatVisualReferences(brief.VisualReferences); refs != "" {
		sections = append(sections, "Visual Canon References:\n"+refs)
	}
	if len(sections) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(sections, "\n\n")
}

func formatContinuityReferences(refs []episode.ContinuityReference) string {
	lines := make([]string, 0, len(refs))
	for _, ref := range refs {
		summary := strings.TrimSpace(ref.Summary)
		if summary == "" {
			summary = strings.TrimSpace(ref.OutputText)
		}
		if summary == "" {
			summary = strings.TrimSpace(ref.Prompt)
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
