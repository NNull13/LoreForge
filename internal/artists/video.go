package artists

import (
	"context"
	"fmt"
	"strings"
	"time"

	"loreforge/pkg/contracts"
)

type VideoArtist struct {
	Provider contracts.VideoProvider
	Seed     int64
}

func (a VideoArtist) Name() string       { return "video-artist" }
func (a VideoArtist) OutputType() string { return "video" }

func (a VideoArtist) Generate(ctx context.Context, brief contracts.EpisodeBrief, _ contracts.EpisodeState) (contracts.EpisodeOutput, error) {
	prompt := buildVideoPrompt(brief)
	seed := a.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	resp, err := a.Provider.GenerateVideo(ctx, contracts.VideoRequest{
		Prompt:   prompt,
		Duration: 15,
		Seed:     seed,
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
			"prompt":   prompt,
			"duration": 15,
			"seed":     seed,
		},
		ProviderResponse: map[string]any{
			"asset_path": resp.AssetPath,
		},
	}, nil
}

func buildVideoPrompt(brief contracts.EpisodeBrief) string {
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
	return fmt.Sprintf("Create a short cinematic scene. World: %s. Characters: %s. Event: %s. Tone: %s. Keep canon rules: %s", brief.WorldID, strings.Join(brief.CharacterIDs, ", "), brief.EventID, brief.Tone, strings.Join(brief.CanonRules, " | "))
}
