package text

import (
	"context"
	"strings"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/application/textprompt"
	"loreforge/internal/application/textsettings"
	"loreforge/internal/domain/episode"
)

type Generator struct {
	GeneratorID string
	Format      episode.OutputType
	Settings    textsettings.ResolvedTextSettings
	Provider    contracts.TextProvider
}

func (g Generator) ID() string { return g.GeneratorID }

func (g Generator) Type() episode.OutputType { return g.Format }

func (g Generator) Generate(ctx context.Context, brief episode.Brief, _ episode.State) (episode.Output, error) {
	bundle := textprompt.Build(brief, g.Format, g.Settings)
	resp, err := g.Provider.GenerateText(ctx, contracts.TextRequest{
		Format:          g.Format,
		SystemPrompt:    bundle.SystemPrompt,
		Prompt:          bundle.UserPrompt,
		Temperature:     g.Settings.Temperature,
		MaxOutputTokens: g.Settings.MaxOutputTokens,
		JSONSchema:      bundle.JSONSchema,
		Options: map[string]any{
			"template_strictness": g.Settings.TemplateStrictness,
		},
	})
	if err != nil {
		return episode.Output{}, err
	}
	textArtifact := buildTextArtifact(resp)
	content := flattenTextArtifact(textArtifact, strings.TrimSpace(resp.Content))
	return episode.Output{
		Content:  content,
		Text:     textArtifact,
		Provider: g.Provider.Name(),
		Model:    resp.Model,
		Prompt:   strings.TrimSpace(bundle.SystemPrompt + "\n\n" + bundle.UserPrompt),
		ProviderRequest: map[string]any{
			"format":            g.Format,
			"system_prompt":     bundle.SystemPrompt,
			"prompt":            bundle.UserPrompt,
			"temperature":       g.Settings.Temperature,
			"max_output_tokens": g.Settings.MaxOutputTokens,
			"json_schema":       bundle.JSONSchema,
		},
		ProviderResponse: map[string]any{
			"content":       resp.Content,
			"parts":         resp.Parts,
			"title":         resp.Title,
			"finish_reason": resp.FinishReason,
			"metadata":      resp.Metadata,
		},
	}, nil
}

func buildTextArtifact(resp contracts.TextResponse) *episode.TextArtifact {
	artifact := &episode.TextArtifact{
		Title: strings.TrimSpace(resp.Title),
	}
	if len(resp.Parts) > 0 {
		artifact.Parts = make([]episode.TextPart, 0, len(resp.Parts))
		for i, part := range resp.Parts {
			artifact.Parts = append(artifact.Parts, episode.TextPart{Index: i, Content: strings.TrimSpace(part)})
		}
		artifact.Body = strings.Join(resp.Parts, "\n\n")
	} else {
		artifact.Body = strings.TrimSpace(resp.Content)
	}
	artifact.WordCount = len(strings.Fields(artifact.Body))
	artifact.CharacterCount = len([]rune(artifact.Body))
	return artifact
}

func flattenTextArtifact(artifact *episode.TextArtifact, fallback string) string {
	if artifact == nil {
		return fallback
	}
	if len(artifact.Parts) > 0 {
		parts := make([]string, 0, len(artifact.Parts))
		for _, part := range artifact.Parts {
			if strings.TrimSpace(part.Content) != "" {
				parts = append(parts, strings.TrimSpace(part.Content))
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n\n")
		}
	}
	if strings.TrimSpace(artifact.Body) != "" {
		return strings.TrimSpace(artifact.Body)
	}
	return fallback
}
