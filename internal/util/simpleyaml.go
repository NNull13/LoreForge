package util

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseSimpleYAMLFile(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSimpleYAML(string(b)), nil
}

func ParseSimpleYAML(input string) map[string]string {
	res := map[string]string{}
	type level struct {
		indent int
		key    string
	}
	stack := []level{{indent: -1, key: ""}}
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		raw := scanner.Text()
		line := stripComment(raw)
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := countIndent(line)
		trimmed := strings.TrimSpace(line)

		for len(stack) > 1 && indent <= stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
		}

		if strings.HasPrefix(trimmed, "- ") {
			if len(stack) < 2 {
				continue
			}
			key := stack[len(stack)-1].key
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			item = trimQuotes(item)
			if old := res[key]; old != "" {
				res[key] = old + "," + item
			} else {
				res[key] = item
			}
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		fullKey := key
		if parent := stack[len(stack)-1].key; parent != "" {
			fullKey = parent + "." + key
		}

		if val == "" {
			stack = append(stack, level{indent: indent, key: fullKey})
			continue
		}
		res[fullKey] = trimQuotes(val)
	}
	return res
}

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

func MustInt(v string, def int) int {
	i, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return i
}

func MustInt64(v string, def int64) int64 {
	i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return def
	}
	return i
}

func MustBool(v string, def bool) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return b
}

func RequireKey(m map[string]string, key string) (string, error) {
	v := strings.TrimSpace(m[key])
	if v == "" {
		return "", fmt.Errorf("missing key: %s", key)
	}
	return v, nil
}

func countIndent(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
			continue
		}
		break
	}
	return n
}

func stripComment(s string) string {
	idx := strings.Index(s, "#")
	if idx < 0 {
		return s
	}
	return s[:idx]
}

func trimQuotes(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	s = strings.Trim(s, "'")
	return s
}
