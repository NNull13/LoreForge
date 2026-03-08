package image

import (
	"context"
	"fmt"
	"strings"
	"time"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/domain/episode"
)

type Generator struct {
	GeneratorID string
	Provider    contracts.ImageProvider
	Seed        int64
}

func (g Generator) ID() string { return g.GeneratorID }

func (g Generator) Type() episode.OutputType { return episode.OutputTypeImage }

func (g Generator) Generate(ctx context.Context, brief episode.Brief, _ episode.State) (episode.Output, error) {
	prompt := buildPrompt(brief)
	seed := g.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	req := contracts.ImageRequest{
		Prompt:       prompt,
		Width:        1024,
		Height:       1024,
		AspectRatio:  "1:1",
		Seed:         seed,
		Count:        1,
		OutputFormat: "png",
		Quality:      "auto",
	}
	resp, err := g.Provider.GenerateImage(ctx, req)
	if err != nil {
		return episode.Output{}, err
	}
	return episode.Output{
		AssetPath: resp.AssetPath,
		Provider:  g.Provider.Name(),
		Model:     resp.Model,
		Prompt:    prompt,
		ProviderRequest: map[string]any{
			"prompt":                prompt,
			"width":                 req.Width,
			"height":                req.Height,
			"seed":                  seed,
			"aspect_ratio":          req.AspectRatio,
			"count":                 req.Count,
			"output_format":         req.OutputFormat,
			"quality":               req.Quality,
			"visual_references":     brief.VisualReferences,
			"continuity_references": brief.ContinuityReferences,
		},
		ProviderResponse: map[string]any{
			"asset_path":     resp.AssetPath,
			"url":            resp.URL,
			"mime_type":      resp.MIMEType,
			"revised_prompt": resp.RevisedPrompt,
			"metadata":       resp.Metadata,
		},
	}, nil
}

func buildPrompt(brief episode.Brief) string {
	contextBlock := fmt.Sprintf(
		"Context:\n- Artist: %s\n- Artist Mission: %s\n- Artist Tone Biases: %s\n- Artist Lexical Cues: %s\n- Artist Non-Diegetic: %t\n- World: %s\n- Characters: %s\n- Event: %s\n- Tone: %s\n- Rules: %s\n- WorldData: %v\n- EventData: %v\n- CharacterData: %v",
		brief.Artist.Name,
		brief.Artist.Mission,
		strings.Join(brief.Artist.TonalBiases, ", "),
		strings.Join(brief.Artist.LexicalCues, ", "),
		brief.Artist.NonDiegetic,
		brief.WorldID,
		strings.Join(brief.CharacterIDs, ", "),
		brief.EventID,
		brief.Tone,
		strings.Join(brief.CanonRules, " | "),
		brief.WorldData,
		brief.EventData,
		brief.CharacterData,
	)
	if refs := formatContinuityReferences(brief.ContinuityReferences); refs != "" {
		contextBlock += "\n\nReference Memories:\n" + refs
	}
	if refs := formatVisualReferences(brief.VisualReferences); refs != "" {
		contextBlock += "\n\nVisual References:\n" + refs
	}
	if strings.TrimSpace(brief.TemplateBody) != "" {
		return fmt.Sprintf("%s\n\n%s", brief.TemplateBody, contextBlock)
	}
	additions := promptAdditions(brief)
	return fmt.Sprintf(
		"Create a still image concept through the editorial lens of %s. World: %s. Characters: %s. Event: %s. Tone: %s. Artist tonal biases: %s. Artist lexical cues: %s. Keep canon rules: %s%s",
		brief.Artist.Name,
		brief.WorldID,
		strings.Join(brief.CharacterIDs, ", "),
		brief.EventID,
		brief.Tone,
		strings.Join(brief.Artist.TonalBiases, ", "),
		strings.Join(brief.Artist.LexicalCues, ", "),
		strings.Join(brief.CanonRules, " | "),
		additions,
	)
}

func promptAdditions(brief episode.Brief) string {
	sections := make([]string, 0, 2)
	if refs := formatContinuityReferences(brief.ContinuityReferences); refs != "" {
		sections = append(sections, "Reference Memories:\n"+refs)
	}
	if refs := formatVisualReferences(brief.VisualReferences); refs != "" {
		sections = append(sections, "Visual References:\n"+refs)
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
