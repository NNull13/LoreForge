package universe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"loreforge/internal/util"
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
	m := util.ParseSimpleYAML(fm)

	id := m["id"]
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	typ := m["type"]
	if typ == "" && strings.HasSuffix(path, "universe.md") {
		typ = "universe"
	}
	data := map[string]any{}
	for k, v := range m {
		if strings.HasPrefix(strings.TrimSpace(v), "[") {
			data[k] = util.ParseInlineList(v)
			continue
		}
		if strings.Contains(v, ",") && (strings.HasSuffix(k, "_worlds") || strings.HasSuffix(k, "_characters") || strings.HasSuffix(k, "_rules") || strings.HasSuffix(k, "affinities")) {
			data[k] = util.ParseStringListValue(v)
			continue
		}
		data[k] = v
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

func coalesce(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
