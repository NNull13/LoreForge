package text

import (
	"context"
	"fmt"
	"strings"

	"loreforge/pkg/contracts"
)

type Agent struct {
	Provider contracts.TextProvider
	Model    string
}

func (a Agent) Name() string       { return "text-agent" }
func (a Agent) OutputType() string { return "text" }

func (a Agent) Generate(ctx context.Context, brief contracts.EpisodeBrief, _ contracts.EpisodeState) (contracts.EpisodeOutput, error) {
	prompt := buildPrompt(brief)
	resp, err := a.Provider.GenerateText(ctx, contracts.TextRequest{
		Prompt:      prompt,
		Temperature: 0.8,
		MaxTokens:   500,
	})
	if err != nil {
		return contracts.EpisodeOutput{}, err
	}
	return contracts.EpisodeOutput{
		Content:  strings.TrimSpace(resp.Content),
		Provider: a.Provider.Name(),
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

func buildPrompt(brief contracts.EpisodeBrief) string {
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
	if strings.TrimSpace(brief.TemplateBody) != "" {
		return fmt.Sprintf("%s\n\n%s", brief.TemplateBody, contextBlock)
	}
	return fmt.Sprintf("Create a short narrative. Type: %s. World: %s. Characters: %s. Event: %s. Tone: %s. Objective: %s. Rules: %s", brief.EpisodeType, brief.WorldID, strings.Join(brief.CharacterIDs, ", "), brief.EventID, brief.Tone, brief.Objective, strings.Join(brief.CanonRules, " | "))
}
