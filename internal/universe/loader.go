package universe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func Load(root string) (Universe, error) {
	u := Universe{
		SourcePath: root,
		Rules:      map[string]Entity{},
		Worlds:     map[string]Entity{},
		Characters: map[string]Entity{},
		Events:     map[string]Entity{},
		Templates:  map[string]Entity{},
	}
	if _, err := os.Stat(root); err != nil {
		return u, fmt.Errorf("universe path: %w", err)
	}

	if ent, err := loadEntity(filepath.Join(root, "universe.md")); err != nil {
		return u, fmt.Errorf("load universe.md: %w", err)
	} else {
		u.Universe = ent
	}

	if err := loadDirEntities(filepath.Join(root, "rules"), "rule", u.Rules); err != nil {
		return u, err
	}
	if err := loadDirEntities(filepath.Join(root, "worlds"), "world", u.Worlds); err != nil {
		return u, err
	}
	if err := loadDirEntities(filepath.Join(root, "characters"), "character", u.Characters); err != nil {
		return u, err
	}
	if err := loadDirEntities(filepath.Join(root, "events"), "event", u.Events); err != nil {
		return u, err
	}
	if err := loadDirEntities(filepath.Join(root, "templates"), "template", u.Templates); err != nil {
		return u, err
	}

	return u, Validate(u)
}

func loadDirEntities(dir, expectedType string, dst map[string]Entity) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		ent, err := loadEntity(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		if expectedType != "" && ent.Type != expectedType {
			return fmt.Errorf("%s expects type=%s, got %s (%s)", dir, expectedType, ent.Type, e.Name())
		}
		dst[ent.ID] = ent
	}
	return nil
}

func loadEntity(path string) (Entity, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Entity{}, err
	}
	text := string(b)
	fm, body, err := splitFrontmatter(text)
	if err != nil {
		return Entity{}, fmt.Errorf("%s: %w", path, err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(fm), &raw); err != nil {
		return Entity{}, fmt.Errorf("%s: invalid frontmatter: %w", path, err)
	}
	data := normalizeYAMLMap(raw)

	id := asString(data["id"])
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	typ := asString(data["type"])
	if typ == "" && strings.HasSuffix(path, "universe.md") {
		typ = "universe"
	}

	return Entity{
		ID:          id,
		Type:        typ,
		DisplayName: coalesce(asString(data["display_name"]), asString(data["name"])),
		Summary:     asString(data["summary"]),
		Body:        strings.TrimSpace(body),
		Data:        data,
		Path:        path,
	}, nil
}

func splitFrontmatter(content string) (fm, body string, err error) {
	if !strings.HasPrefix(content, "---\n") {
		return "", "", errors.New("missing frontmatter start '---'")
	}
	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		return "", "", errors.New("missing frontmatter closing '---'")
	}
	fm = rest[:idx]
	body = rest[idx+5:]
	return fm, body, nil
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func normalizeYAMLMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = normalizeYAMLValue(v)
	}
	return out
}

func normalizeYAMLValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return normalizeYAMLMap(t)
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, v2 := range t {
			out[fmt.Sprint(k)] = normalizeYAMLValue(v2)
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, normalizeYAMLValue(item))
		}
		return out
	default:
		return v
	}
}

func coalesce(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
