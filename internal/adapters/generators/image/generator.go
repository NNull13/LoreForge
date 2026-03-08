package image

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
	Provider    providercontracts.ImageProvider
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
	req := providercontracts.ImageRequest{
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
			"prompt":        prompt,
			"width":         req.Width,
			"height":        req.Height,
			"seed":          seed,
			"aspect_ratio":  req.AspectRatio,
			"count":         req.Count,
			"output_format": req.OutputFormat,
			"quality":       req.Quality,
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
	if strings.TrimSpace(brief.TemplateBody) != "" {
		return fmt.Sprintf("%s\n\n%s", brief.TemplateBody, contextBlock)
	}
	return fmt.Sprintf(
		"Create a still image concept. World: %s. Characters: %s. Event: %s. Tone: %s. Keep canon rules: %s",
		brief.WorldID,
		strings.Join(brief.CharacterIDs, ", "),
		brief.EventID,
		brief.Tone,
		strings.Join(brief.CanonRules, " | "),
	)
}
