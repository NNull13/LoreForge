package artists

import (
	"context"
	"fmt"
	"strings"
	"time"

	"loreforge/pkg/contracts"
)

type ImageArtist struct {
	Provider contracts.ImageProvider
	Seed     int64
}

func (a ImageArtist) Name() string       { return "image-artist" }
func (a ImageArtist) OutputType() string { return "image" }

func (a ImageArtist) Generate(ctx context.Context, brief contracts.EpisodeBrief, _ contracts.EpisodeState) (contracts.EpisodeOutput, error) {
	prompt := buildImagePrompt(brief)
	seed := a.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	resp, err := a.Provider.GenerateImage(ctx, contracts.ImageRequest{
		Prompt: prompt,
		Width:  1024,
		Height: 1024,
		Seed:   seed,
	})
	if err != nil {
		return contracts.EpisodeOutput{}, err
	}
	return contracts.EpisodeOutput{
		AssetPath: resp.AssetPath,
		Provider:  a.Provider.Name(),
		Model:     resp.Model,
		Prompt:    prompt,
		ProviderRequest: map[string]any{
			"prompt": prompt,
			"width":  1024,
			"height": 1024,
			"seed":   seed,
		},
		ProviderResponse: map[string]any{
			"asset_path": resp.AssetPath,
		},
	}, nil
}

func buildImagePrompt(brief contracts.EpisodeBrief) string {
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
	return fmt.Sprintf("Create a still image concept. World: %s. Characters: %s. Event: %s. Tone: %s. Keep canon rules: %s", brief.WorldID, strings.Join(brief.CharacterIDs, ", "), brief.EventID, brief.Tone, strings.Join(brief.CanonRules, " | "))
}
