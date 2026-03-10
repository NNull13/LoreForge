package universe_fs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	domainuniverse "loreforge/internal/domain/universe"

	"gopkg.in/yaml.v3"
)

type Repository struct {
	Root string
}

type assetMetadataFile struct {
	Assets []assetMetadata `yaml:"assets"`
}

type assetMetadata struct {
	FileName    string            `yaml:"file"`
	ID          string            `yaml:"id"`
	MediaType   string            `yaml:"media_type"`
	Usage       string            `yaml:"usage"`
	Description string            `yaml:"description"`
	Tags        []string          `yaml:"tags"`
	Weight      int               `yaml:"weight"`
	Optional    *bool             `yaml:"optional"`
	ModelRoles  map[string]string `yaml:"model_roles"`
}

type entityDirectoryLoad struct {
	dir          string
	expectedType string
	dst          map[string]domainuniverse.Entity
}

func (r Repository) Load(_ context.Context) (domainuniverse.Universe, error) {
	u := domainuniverse.Universe{
		SourcePath: r.Root,
		Artists:    map[string]domainuniverse.Artist{},
		Rules:      map[string]domainuniverse.Entity{},
		Worlds:     map[string]domainuniverse.Entity{},
		Characters: map[string]domainuniverse.Entity{},
		Events:     map[string]domainuniverse.Entity{},
		Templates:  map[string]domainuniverse.Entity{},
	}
	if _, err := os.Stat(r.Root); err != nil {
		return u, fmt.Errorf("universe path: %w", err)
	}

	rootEntityDir := filepath.Join(r.Root, "universe")
	entity, err := loadEntityDirectory(rootEntityDir, "universe", "")
	if err != nil {
		return u, err
	}
	u.Universe = entity

	entityLoads := []entityDirectoryLoad{
		{dir: "rules", expectedType: "rule", dst: u.Rules},
		{dir: "worlds", expectedType: "world", dst: u.Worlds},
		{dir: "characters", expectedType: "character", dst: u.Characters},
		{dir: "events", expectedType: "event", dst: u.Events},
		{dir: "templates", expectedType: "template", dst: u.Templates},
	}
	for _, load := range entityLoads {
		if err = loadEntityDirectories(filepath.Join(r.Root, load.dir), load.expectedType, load.dst); err != nil {
			return u, err
		}
	}
	if err = loadArtistDirectories(filepath.Join(r.Root, "artists"), u.Artists); err != nil {
		return u, err
	}

	if err = domainuniverse.Validate(u); err != nil {
		return u, err
	}
	return u, nil
}

func loadEntityDirectories(dir, expectedType string, dst map[string]domainuniverse.Entity) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read %s: %w", dir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return fmt.Errorf("%s only supports entity directories, found file %s", dir, entry.Name())
		}
		entity, err := loadEntityDirectory(filepath.Join(dir, entry.Name()), expectedType, entry.Name())
		if err != nil {
			return err
		}
		dst[entity.ID] = entity
	}
	return nil
}

func loadArtistDirectories(dir string, dst map[string]domainuniverse.Artist) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read %s: %w", dir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			return fmt.Errorf("%s only supports artist directories, found file %s", dir, entry.Name())
		}
		artist, err := loadArtistDirectory(filepath.Join(dir, entry.Name()), entry.Name())
		if err != nil {
			return err
		}
		dst[artist.ID] = artist
	}
	return nil
}

func loadEntityDirectory(dir, expectedType, expectedID string) (domainuniverse.Entity, error) {
	fileBase := expectedID
	if fileBase == "" {
		fileBase = filepath.Base(dir)
	}
	mdPath := filepath.Join(dir, fileBase+".md")
	entity, err := loadEntity(mdPath)
	if err != nil {
		return domainuniverse.Entity{}, err
	}
	if expectedID != "" && entity.ID != expectedID {
		return domainuniverse.Entity{}, fmt.Errorf("%s id mismatch: folder=%s id=%s", dir, expectedID, entity.ID)
	}
	if expectedType != "" && entity.Type != expectedType {
		return domainuniverse.Entity{}, fmt.Errorf("%s expects type=%s, got %s", dir, expectedType, entity.Type)
	}
	if err := ensureOnlyExpectedMarkdown(dir, fileBase+".md"); err != nil {
		return domainuniverse.Entity{}, err
	}
	assets, err := loadAssets(dir, entity.Type)
	if err != nil {
		return domainuniverse.Entity{}, err
	}
	entity.Assets = assets
	return entity, nil
}

func loadArtistDirectory(dir, expectedID string) (domainuniverse.Artist, error) {
	mdPath := filepath.Join(dir, "artist.md")
	entity, err := loadEntity(mdPath)
	if err != nil {
		return domainuniverse.Artist{}, err
	}
	if entity.ID != expectedID {
		return domainuniverse.Artist{}, fmt.Errorf("%s id mismatch: folder=%s id=%s", dir, expectedID, entity.ID)
	}
	if err := ensureOnlyExpectedMarkdown(dir, "artist.md"); err != nil {
		return domainuniverse.Artist{}, err
	}
	assets, err := loadAssets(dir, "artist")
	if err != nil {
		return domainuniverse.Artist{}, err
	}
	data := entity.Data
	voice := mapStringAny(data["voice"])
	mission := mapStringAny(data["mission"])
	prompting := mapStringAny(data["prompting"])
	presentation := mapStringAny(data["presentation"])
	future := mapStringAny(data["future"])
	artist := domainuniverse.Artist{
		ID:          entity.ID,
		Name:        coalesce(asString(data["name"]), entity.DisplayName, entity.ID),
		Title:       asString(data["title"]),
		Role:        asString(data["role"]),
		Summary:     entity.Summary,
		Body:        entity.Body,
		NonDiegetic: boolWithDefault(data["non_diegietic"], true),
		Voice: domainuniverse.ArtistVoice{
			Register:    asString(voice["register"]),
			Cadence:     asString(voice["cadence"]),
			Diction:     asString(voice["diction"]),
			Stance:      asString(voice["stance"]),
			Perspective: asString(voice["perspective"]),
			Intensity:   asString(voice["intensity"]),
		},
		Mission: domainuniverse.ArtistMission{
			Purpose:    asString(mission["purpose"]),
			Priorities: stringSlice(mission["priorities"]),
		},
		Prompting: domainuniverse.ArtistPrompting{
			SystemIdentity: asString(prompting["system_identity"]),
			SystemRules:    stringSlice(prompting["system_rules"]),
			TonalBiases:    stringSlice(prompting["tonal_biases"]),
			LexicalCues:    stringSlice(prompting["lexical_cues"]),
			Forbidden:      stringSlice(prompting["forbidden"]),
		},
		Presentation: domainuniverse.ArtistPresentation{
			Enabled:         boolWithDefault(presentation["enabled"], false),
			SignatureMode:   coalesce(asString(presentation["signature_mode"]), "presentation_only"),
			SignatureText:   asString(presentation["signature_text"]),
			FramingMode:     coalesce(asString(presentation["framing_mode"]), "none"),
			IntroTemplate:   asString(presentation["intro_template"]),
			OutroTemplate:   asString(presentation["outro_template"]),
			AllowedChannels: stringSlice(presentation["allowed_channels"]),
		},
		Future: domainuniverse.ArtistFuture{
			MemoryMode: coalesce(asString(future["memory_mode"]), "reserved"),
		},
		Assets: assets,
		Path:   mdPath,
		Data:   data,
	}
	return artist, nil
}

func ensureOnlyExpectedMarkdown(dir, expected string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".md") && entry.Name() != expected {
			return fmt.Errorf("%s contains unexpected markdown file %s", dir, entry.Name())
		}
	}
	return nil
}

func loadEntity(path string) (domainuniverse.Entity, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return domainuniverse.Entity{}, fmt.Errorf("load %s: %w", path, err)
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
	if entityType == "" && filepath.Base(path) == "universe.md" {
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

func loadAssets(dir, entityType string) (domainuniverse.AssetSet, error) {
	declared, err := loadAssetsMetadata(dir)
	if err != nil {
		return domainuniverse.AssetSet{}, err
	}
	discovered, err := discoverAssets(dir, entityType)
	if err != nil {
		return domainuniverse.AssetSet{}, err
	}
	items, err := mergeAssets(declared, discovered)
	if err != nil {
		return domainuniverse.AssetSet{}, err
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Weight != items[j].Weight {
			return items[i].Weight > items[j].Weight
		}
		return items[i].ID < items[j].ID
	})
	return domainuniverse.AssetSet{Items: items}, nil
}

func loadAssetsMetadata(dir string) ([]domainuniverse.Asset, error) {
	path := filepath.Join(dir, "assets.yaml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file assetMetadataFile
	if err := yaml.Unmarshal(content, &file); err != nil {
		return nil, fmt.Errorf("%s: invalid yaml: %w", path, err)
	}
	out := make([]domainuniverse.Asset, 0, len(file.Assets))
	for _, item := range file.Assets {
		if err := validateDeclaredAssetFileName(item.FileName); err != nil {
			return nil, fmt.Errorf("%s asset %q invalid: %w", path, item.FileName, err)
		}
		assetPath := filepath.Join(dir, item.FileName)
		if _, err := os.Stat(assetPath); err != nil {
			return nil, fmt.Errorf("%s references missing asset %s", path, item.FileName)
		}
		out = append(out, domainuniverse.Asset{
			ID:          coalesce(item.ID, strings.TrimSuffix(item.FileName, filepath.Ext(item.FileName))),
			FileName:    item.FileName,
			Path:        assetPath,
			MediaType:   coalesce(item.MediaType, mediaTypeFromExt(item.FileName)),
			Usage:       item.Usage,
			Description: item.Description,
			Tags:        append([]string(nil), item.Tags...),
			Weight:      item.Weight,
			Optional:    item.Optional == nil || *item.Optional,
			ModelRoles:  cloneStringMap(item.ModelRoles),
		})
	}
	return out, nil
}

func discoverAssets(dir, entityType string) ([]domainuniverse.Asset, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]domainuniverse.Asset, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.EqualFold(name, "assets.yaml") || strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		mediaType := mediaTypeFromExt(name)
		if mediaType == "" {
			continue
		}
		out = append(out, domainuniverse.Asset{
			ID:          strings.TrimSuffix(name, filepath.Ext(name)),
			FileName:    name,
			Path:        filepath.Join(dir, name),
			MediaType:   mediaType,
			Usage:       defaultUsage(entityType),
			Description: "",
			Weight:      50,
			Optional:    true,
		})
	}
	return out, nil
}

func mergeAssets(declared []domainuniverse.Asset, discovered []domainuniverse.Asset) ([]domainuniverse.Asset, error) {
	byFile := map[string]domainuniverse.Asset{}
	for _, item := range discovered {
		byFile[item.FileName] = item
	}
	for _, item := range declared {
		byFile[item.FileName] = item
	}
	out := make([]domainuniverse.Asset, 0, len(byFile))
	for _, item := range byFile {
		out = append(out, item)
	}
	return out, nil
}

func defaultUsage(entityType string) string {
	switch entityType {
	case "character":
		return "character_reference"
	case "world":
		return "environment_reference"
	case "artist":
		return "style_reference"
	default:
		return "continuity_reference"
	}
}

func mediaTypeFromExt(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".webp":
		return "image"
	case ".mp4", ".mov", ".webm":
		return "video"
	default:
		return ""
	}
}

func validateDeclaredAssetFileName(name string) error {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return errors.New("file is required")
	case filepath.IsAbs(name):
		return errors.New("absolute paths are not allowed")
	case filepath.Base(name) != name:
		return errors.New("subdirectories are not allowed")
	case strings.Contains(name, ".."):
		return errors.New("path traversal is not allowed")
	default:
		return nil
	}
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

func mapStringAny(value any) map[string]any {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func stringSlice(value any) []string {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func boolWithDefault(value any, fallback bool) bool {
	if value == nil {
		return fallback
	}
	if b, ok := value.(bool); ok {
		return b
	}
	return fallback
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
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
