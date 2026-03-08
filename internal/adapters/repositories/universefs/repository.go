package universefs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	domainuniverse "loreforge/internal/domain/universe"
)

type Repository struct {
	Root string
}

func (r Repository) Load(_ context.Context) (domainuniverse.Universe, error) {
	u := domainuniverse.Universe{
		SourcePath: r.Root,
		Rules:      map[string]domainuniverse.Entity{},
		Worlds:     map[string]domainuniverse.Entity{},
		Characters: map[string]domainuniverse.Entity{},
		Events:     map[string]domainuniverse.Entity{},
		Templates:  map[string]domainuniverse.Entity{},
	}
	if _, err := os.Stat(r.Root); err != nil {
		return u, fmt.Errorf("universe path: %w", err)
	}
	entity, err := loadEntity(filepath.Join(r.Root, "universe.md"))
	if err != nil {
		return u, fmt.Errorf("load universe.md: %w", err)
	}
	u.Universe = entity
	if err := loadDirEntities(filepath.Join(r.Root, "rules"), "rule", u.Rules); err != nil {
		return u, err
	}
	if err := loadDirEntities(filepath.Join(r.Root, "worlds"), "world", u.Worlds); err != nil {
		return u, err
	}
	if err := loadDirEntities(filepath.Join(r.Root, "characters"), "character", u.Characters); err != nil {
		return u, err
	}
	if err := loadDirEntities(filepath.Join(r.Root, "events"), "event", u.Events); err != nil {
		return u, err
	}
	if err := loadDirEntities(filepath.Join(r.Root, "templates"), "template", u.Templates); err != nil {
		return u, err
	}
	if err := domainuniverse.Validate(u); err != nil {
		return u, err
	}
	return u, nil
}

func loadDirEntities(dir, expectedType string, dst map[string]domainuniverse.Entity) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		entity, err := loadEntity(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		if expectedType != "" && entity.Type != expectedType {
			return fmt.Errorf("%s expects type=%s, got %s (%s)", dir, expectedType, entity.Type, entry.Name())
		}
		dst[entity.ID] = entity
	}
	return nil
}

func loadEntity(path string) (domainuniverse.Entity, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return domainuniverse.Entity{}, err
	}
	fm, body, err := splitFrontmatter(string(content))
	if err != nil {
		return domainuniverse.Entity{}, fmt.Errorf("%s: %w", path, err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(fm), &raw); err != nil {
		return domainuniverse.Entity{}, fmt.Errorf("%s: invalid frontmatter: %w", path, err)
	}
	data := normalizeYAMLMap(raw)
	id := asString(data["id"])
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	entityType := asString(data["type"])
	if entityType == "" && strings.HasSuffix(path, "universe.md") {
		entityType = "universe"
	}
	return domainuniverse.Entity{
		ID:          id,
		Type:        entityType,
		DisplayName: coalesce(asString(data["display_name"]), asString(data["name"])),
		Summary:     asString(data["summary"]),
		Body:        strings.TrimSpace(body),
		Data:        data,
		Path:        path,
	}, nil
}

func splitFrontmatter(content string) (fm string, body string, err error) {
	if !strings.HasPrefix(content, "---\n") {
		return "", "", errors.New("missing frontmatter start '---'")
	}
	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		return "", "", errors.New("missing frontmatter closing '---'")
	}
	return rest[:idx], rest[idx+5:], nil
}

func normalizeYAMLMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = normalizeYAMLValue(value)
	}
	return out
}

func normalizeYAMLValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return normalizeYAMLMap(typed)
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			out[fmt.Sprint(key)] = normalizeYAMLValue(nested)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, normalizeYAMLValue(item))
		}
		return out
	default:
		return value
	}
}

func asString(value any) string {
	s, _ := value.(string)
	return s
}

func coalesce(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
