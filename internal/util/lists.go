package util

import "strings"

func ParseInlineList(raw string) []string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		item := trimQuotes(strings.TrimSpace(p))
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func ParseStringListValue(v string) []string {
	if strings.HasPrefix(strings.TrimSpace(v), "[") {
		return ParseInlineList(v)
	}
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := trimQuotes(strings.TrimSpace(p))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func trimQuotes(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	s = strings.Trim(s, "'")
	return s
}
