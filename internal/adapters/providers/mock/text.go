package mock

import (
	"context"

	"loreforge/internal/adapters/providers/contracts"
)

type TextProvider struct {
	Model string
}

func (p TextProvider) Name() string { return "mock-text" }

func (p TextProvider) GenerateText(_ context.Context, input contracts.TextRequest) (contracts.TextResponse, error) {
	switch input.Format {
	case "tweet_short":
		return contracts.TextResponse{
			Content: "Red Wanderer crosses the glass gate while the city remembers his oath.",
			Model:   coalesce(p.Model, "mock-text-v1"),
		}, nil
	case "tweet_thread":
		return contracts.TextResponse{
			Parts: []string{
				"Red Wanderer reaches the ash gate at dawn and the hinges sing.",
				"The Architect waits beyond the glass tower, counting sparks like debts.",
				"By nightfall, the city remembers both names and refuses to forget again.",
			},
			Model: coalesce(p.Model, "mock-text-v1"),
		}, nil
	case "poem":
		return contracts.TextResponse{
			Title:   "Ash Gate",
			Content: "Red Wanderer walks where mirrors burn,\nGlass remembers every turn,\nThe Architect keeps silver time,\nAnd rust begins to speak in rhyme.",
			Model:   coalesce(p.Model, "mock-text-v1"),
		}, nil
	case "song_lyrics":
		return contracts.TextResponse{
			Title:   "Lantern Chorus",
			Content: "Verse 1\nRed Wanderer keeps the ember low\nThe city hums beneath the stone\n\nChorus\nCarry the lantern, carry the name\nGlass and iron answer the flame",
			Model:   coalesce(p.Model, "mock-text-v1"),
		}, nil
	case "screenplay_series":
		return contracts.TextResponse{
			Title:   "Gate of Glass",
			Content: "INT. GLASS KINGDOM GATE - NIGHT\nRed Wanderer studies the sealed arch.\n\nTHE ARCHITECT\nYou came back for the oath.\n\nRED WANDERER\nI came back for the truth.",
			Model:   coalesce(p.Model, "mock-text-v1"),
		}, nil
	case "long_story":
		return contracts.TextResponse{
			Title:   "The Oath Returns",
			Content: longStoryContent(),
			Model:   coalesce(p.Model, "mock-text-v1"),
		}, nil
	default:
		return contracts.TextResponse{
			Title:   "Ash Garden",
			Content: shortStoryContent(),
			Model:   coalesce(p.Model, "mock-text-v1"),
		}, nil
	}
}

func coalesce(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func shortStoryContent() string {
	segment := "Red Wanderer crossed the ash garden at dusk, following the heat that lived beneath the cracked tiles. The city held its breath while The Architect waited near the gate of black glass, silver mask dimmed by soot and distance. Between them stood the oath they had once buried under iron dust. When the first lantern sparked, the old machinery in the walls began to hum. Red Wanderer spoke first, not with anger but with the careful tone reserved for ruins that might still answer. The Architect listened, then opened one hand to reveal the missing seal of the gate. In that quiet exchange the city understood that memory had returned, and with it the cost of keeping the frontier alive."
	return segment + " " + segment + " " + segment + " " + segment
}

func longStoryContent() string {
	return shortStoryContent() + " " + shortStoryContent() + " " + shortStoryContent() + " " + shortStoryContent()
}
