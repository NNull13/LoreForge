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
	if brief.EpisodeType == OutputTypeText {
		content := strings.TrimSpace(out.Content)
		if len(content) < 40 {
			return fmt.Errorf("%w: text output too short", ErrOutputInvalid)
		}
		if !ContainsEntities(content, brief.CharacterIDs) {
			return fmt.Errorf("%w: no character mentioned in output", ErrOutputInvalid)
		}
		if maxLen := TemplateMaxChars(brief.TemplateBody); maxLen > 0 && len(content) > maxLen {
			return fmt.Errorf("%w: text output exceeds template max chars (%d)", ErrOutputInvalid, maxLen)
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
