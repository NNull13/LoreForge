package episode

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrOutputInvalid         = errors.New("episode output invalid")
	ErrGeneratorUnavailable  = errors.New("generator unavailable")
	ErrEpisodeNotFound       = errors.New("episode not found")
	ErrUniverseInvalid       = errors.New("universe invalid")
	ErrNoGeneratorsAvailable = errors.New("no generators available")
)

func ValidateOutput(out Output, brief Brief) error {
	if brief.EpisodeType.IsTextual() {
		if err := validateTextualOutput(out, brief); err != nil {
			return err
		}
	}
	for _, bad := range []string{"api_key", "token=", "secret="} {
		if strings.Contains(strings.ToLower(out.Content), bad) {
			return fmt.Errorf("%w: forbidden term found", ErrOutputInvalid)
		}
	}
	for _, rule := range brief.CanonRules {
		const prefix = "FORBIDDEN:"
		if !strings.HasPrefix(strings.ToUpper(rule), prefix) {
			continue
		}
		term := strings.TrimSpace(rule[len(prefix):])
		if term == "" {
			continue
		}
		if strings.Contains(strings.ToLower(out.Content), strings.ToLower(term)) {
			return fmt.Errorf("%w: forbidden term found: %s", ErrOutputInvalid, term)
		}
	}
	return nil
}

func validateTextualOutput(out Output, brief Brief) error {
	content := strings.TrimSpace(out.Content)
	if len(content) < 40 {
		return fmt.Errorf("%w: text output too short", ErrOutputInvalid)
	}
	constraints := brief.TextConstraints
	if constraints == nil {
		constraints = &TextConstraints{Type: brief.EpisodeType}
	}
	if templateMax := TemplateMaxChars(brief.TemplateBody); templateMax > 0 && len(content) > templateMax {
		return fmt.Errorf("%w: text output exceeds template max chars (%d)", ErrOutputInvalid, templateMax)
	}
	parts := out.OutputParts()
	switch brief.EpisodeType {
	case OutputTypeTweetShort:
		if len(parts) == 0 {
			parts = []string{content}
		}
		if len(parts) != 1 {
			return fmt.Errorf("%w: tweet_short requires exactly one part", ErrOutputInvalid)
		}
		maxChars := constraints.MaxCharsPerPart
		if maxChars <= 0 {
			maxChars = 280
		}
		if len([]rune(strings.TrimSpace(parts[0]))) > maxChars {
			return fmt.Errorf("%w: tweet_short exceeds 280 chars", ErrOutputInvalid)
		}
	case OutputTypeTweetThread:
		minParts := constraints.MinParts
		if minParts <= 0 {
			minParts = 2
		}
		maxParts := constraints.MaxParts
		if maxParts <= 0 {
			maxParts = 5
		}
		if len(parts) < minParts || len(parts) > maxParts {
			return fmt.Errorf("%w: tweet_thread requires 2-5 parts", ErrOutputInvalid)
		}
		maxChars := constraints.MaxCharsPerPart
		if maxChars <= 0 {
			maxChars = 280
		}
		for _, part := range parts {
			if len([]rune(strings.TrimSpace(part))) > maxChars {
				return fmt.Errorf("%w: tweet_thread part exceeds 280 chars", ErrOutputInvalid)
			}
		}
	case OutputTypeShortStory:
		if constraints.MinWords > 0 && countWords(content) < constraints.MinWords {
			return fmt.Errorf("%w: short_story too short", ErrOutputInvalid)
		}
	case OutputTypeLongStory:
		if constraints.MinWords > 0 && countWords(content) < constraints.MinWords {
			return fmt.Errorf("%w: long_story too short", ErrOutputInvalid)
		}
	case OutputTypePoem:
		if constraints.TargetLineCount > 0 && countNonEmptyLines(content) < max(4, constraints.TargetLineCount/2) {
			return fmt.Errorf("%w: poem requires multiple lines", ErrOutputInvalid)
		}
	case OutputTypeSongLyrics:
		lower := strings.ToLower(content)
		if !strings.Contains(lower, "verse") || !strings.Contains(lower, "chorus") {
			return fmt.Errorf("%w: song_lyrics must include Verse and Chorus", ErrOutputInvalid)
		}
	case OutputTypeScreenplaySeries:
		lower := strings.ToLower(content)
		if !strings.Contains(lower, "int.") && !strings.Contains(lower, "ext.") {
			return fmt.Errorf("%w: screenplay_series must include scene headings", ErrOutputInvalid)
		}
	}
	if constraints.RequireEntityMatch && !ContainsEntities(content, brief.CharacterIDs) {
		return fmt.Errorf("%w: no character mentioned in output", ErrOutputInvalid)
	}
	if constraints.MaxWords > 0 && countWords(content) > constraints.MaxWords {
		return fmt.Errorf("%w: text output exceeds max words", ErrOutputInvalid)
	}
	return nil
}

func countWords(content string) int {
	return len(strings.Fields(content))
}

func countNonEmptyLines(content string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (o Output) OutputParts() []string {
	if o.Text == nil || len(o.Text.Parts) == 0 {
		return nil
	}
	out := make([]string, 0, len(o.Text.Parts))
	for _, part := range o.Text.Parts {
		out = append(out, part.Content)
	}
	return out
}

func ContainsEntities(content string, entities []string) bool {
	if len(entities) == 0 {
		return true
	}
	lower := strings.ToLower(content)
	for _, entity := range entities {
		normalized := strings.ToLower(strings.ReplaceAll(entity, "-", " "))
		if strings.Contains(lower, normalized) || strings.Contains(lower, strings.ToLower(entity)) {
			return true
		}
	}
	return false
}

func TemplateMaxChars(templateBody string) int {
	const marker = "MAX_CHARS:"
	for _, line := range strings.Split(templateBody, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToUpper(line), marker) {
			continue
		}
		rest := strings.TrimSpace(line[len(marker):])
		if rest == "" {
			return 0
		}
		var value int
		if _, err := fmt.Sscanf(rest, "%d", &value); err == nil && value > 0 {
			return value
		}
	}
	return 0
}
