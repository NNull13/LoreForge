package text_parse

import (
	"encoding/json"
	"fmt"
	"strings"
)

func ParseStructuredContent(raw string) (content string, parts []string, title string, metadata map[string]any, err error) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", nil, "", nil, err
	}
	metadata = parsed
	if value, ok := parsed["title"].(string); ok {
		title = strings.TrimSpace(value)
	}
	if arr, ok := parsed["parts"].([]any); ok {
		parts = make([]string, 0, len(arr))
		for _, item := range arr {
			s, ok := item.(string)
			if !ok {
				return "", nil, "", nil, fmt.Errorf("invalid parts item")
			}
			parts = append(parts, strings.TrimSpace(s))
		}
		content = strings.Join(parts, "\n\n")
		return content, parts, title, metadata, nil
	}
	if body, ok := parsed["body"].(string); ok {
		content = strings.TrimSpace(body)
		return content, nil, title, metadata, nil
	}
	return strings.TrimSpace(raw), nil, title, metadata, nil
}
