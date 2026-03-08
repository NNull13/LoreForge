package video

import (
	"context"
	"fmt"
	"strings"
	"time"

	providercontracts "loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/domain/episode"
)

type Generator struct {
	GeneratorID string
	Provider    providercontracts.VideoProvider
	Seed        int64
}

func (g Generator) ID() string { return g.GeneratorID }

func (g Generator) Type() episode.OutputType { return episode.OutputTypeVideo }

func (g Generator) Generate(ctx context.Context, brief episode.Brief, _ episode.State) (episode.Output, error) {
	return g.GenerateWithState(ctx, brief, episode.State{})
}

func (g Generator) GenerateWithState(ctx context.Context, brief episode.Brief, state episode.State) (episode.Output, error) {
	prompt := buildPrompt(brief)
	seed := g.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	req := providercontracts.VideoRequest{
		Prompt:      prompt,
		Duration:    15,
		Seed:        seed,
		AspectRatio: "1280:720",
		Resolution:  "720p",
		Count:       1,
	}
	req.PromptImage = bestPromptImage(brief.VisualReferences)
	req.ReferenceImages = buildReferenceImages(brief.VisualReferences)
	if req.PromptImage == "" && state.Metadata != nil {
		if v, ok := state.Metadata["prompt_image"].(string); ok {
			req.PromptImage = v
		}
	}
	resp, err := g.Provider.GenerateVideo(ctx, req)
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
			"duration":              req.Duration,
			"seed":                  seed,
			"prompt_image":          req.PromptImage,
			"reference_images":      req.ReferenceImages,
			"aspect_ratio":          req.AspectRatio,
			"resolution":            req.Resolution,
			"count":                 req.Count,
			"visual_references":     brief.VisualReferences,
			"continuity_references": brief.ContinuityReferences,
		},
		ProviderResponse: map[string]any{
			"asset_path": resp.AssetPath,
			"url":        resp.URL,
			"job_id":     resp.JobID,
			"metadata":   resp.Metadata,
		},
	}, nil
}

func buildPrompt(brief episode.Brief) string {
	contextBlock := fmt.Sprintf(
		"Context:\n- World: %s\n- Characters: %s\n- Event: %s\n- Tone: %s\n- Rules: %s\n- WorldData: %v\n- EventData: %v\n- CharacterData: %v",
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
		contextBlock += "\n\nContinuity Memories:\n" + refs
	}
	if refs := formatVisualReferences(brief.VisualReferences); refs != "" {
		contextBlock += "\n\nVisual References:\n" + refs
	}
	if strings.TrimSpace(brief.TemplateBody) != "" {
		return fmt.Sprintf("%s\n\n%s", brief.TemplateBody, contextBlock)
	}
	additions := promptAdditions(brief)
	return fmt.Sprintf(
		"Create a short cinematic scene. World: %s. Characters: %s. Event: %s. Tone: %s. Keep canon rules: %s%s",
		brief.WorldID,
		strings.Join(brief.CharacterIDs, ", "),
		brief.EventID,
		brief.Tone,
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

func bestPromptImage(refs []episode.VisualReference) string {
	for _, ref := range refs {
		if ref.MediaType == "image" && ref.ModelRole == "prompt_image" {
			return ref.Path
		}
	}
	return ""
}

func buildReferenceImages(refs []episode.VisualReference) []providercontracts.ReferenceImage {
	out := make([]providercontracts.ReferenceImage, 0, len(refs))
	for _, ref := range refs {
		if ref.MediaType != "image" {
			continue
		}
		if ref.ModelRole != "asset" {
			continue
		}
		out = append(out, providercontracts.ReferenceImage{
			URI:           ref.Path,
			MIMEType:      "image/" + imageExtension(ref.Path),
			ReferenceType: "asset",
		})
	}
	return out
}

func imageExtension(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return "png"
	}
	return strings.ToLower(parts[len(parts)-1])
}
