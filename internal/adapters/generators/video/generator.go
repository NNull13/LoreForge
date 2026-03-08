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
	if state.Metadata != nil {
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
			"prompt":       prompt,
			"duration":     req.Duration,
			"seed":         seed,
			"prompt_image": req.PromptImage,
			"aspect_ratio": req.AspectRatio,
			"resolution":   req.Resolution,
			"count":        req.Count,
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
	if strings.TrimSpace(brief.TemplateBody) != "" {
		return fmt.Sprintf("%s\n\n%s", brief.TemplateBody, contextBlock)
	}
	return fmt.Sprintf(
		"Create a short cinematic scene. World: %s. Characters: %s. Event: %s. Tone: %s. Keep canon rules: %s",
		brief.WorldID,
		strings.Join(brief.CharacterIDs, ", "),
		brief.EventID,
		brief.Tone,
		strings.Join(brief.CanonRules, " | "),
	)
}
