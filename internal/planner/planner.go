package planner

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"loreforge/internal/domain/episode"
	"loreforge/internal/domain/universe"
)

type HistoryCombo struct {
	WorldID      string
	CharacterIDs []string
	EventID      string
}

type Config struct {
	Weights        map[string]int
	RecencyWindow  int
	Seed           int64
	ProductionMode bool
}

type Planner struct {
	rng *rand.Rand
	cfg Config
}

func New(cfg Config) *Planner {
	if cfg.RecencyWindow <= 0 {
		cfg.RecencyWindow = 20
	}
	seed := cfg.Seed
	if seed == 0 || cfg.ProductionMode {
		seed = cfg.Seed ^ time.Now().UnixNano()
	}
	return &Planner{rng: rand.New(rand.NewSource(seed)), cfg: cfg}
}

func (p *Planner) BuildBrief(u universe.Universe, recent []HistoryCombo) (episode.Brief, error) {
	episodeType, err := weightedPick(p.rng, p.cfg.Weights)
	if err != nil {
		return episode.Brief{}, err
	}
	return p.BuildBriefForType(u, episodeType, recent)
}

func (p *Planner) BuildBriefForType(u universe.Universe, outputType string, recent []HistoryCombo) (episode.Brief, error) {
	worldID := pickKey(p.rng, u.Worlds)
	if worldID == "" {
		return episode.Brief{}, errors.New("no worlds available in universe")
	}

	eventID, err := pickOne(p.rng, compatibleEventIDs(u, worldID))
	if err != nil {
		return episode.Brief{}, err
	}
	templateID, err := p.pickTemplate(u, outputType)
	if err != nil {
		return episode.Brief{}, err
	}
	charIDs, err := p.pickCharactersFromIDs(compatibleCharacterIDs(u, worldID), 1+p.rng.Intn(2))
	if err != nil {
		return episode.Brief{}, err
	}

	candidate := comboKey(worldID, charIDs, eventID)
	window := recent
	if len(window) > p.cfg.RecencyWindow {
		window = window[len(window)-p.cfg.RecencyWindow:]
	}
	for i := 0; i < 5; i++ {
		if !containsCombo(window, candidate) {
			break
		}
		worldID = pickKey(p.rng, u.Worlds)
		eventID, err = pickOne(p.rng, compatibleEventIDs(u, worldID))
		if err != nil {
			return episode.Brief{}, err
		}
		charIDs, err = p.pickCharactersFromIDs(compatibleCharacterIDs(u, worldID), 1+p.rng.Intn(2))
		if err != nil {
			return episode.Brief{}, err
		}
		candidate = comboKey(worldID, charIDs, eventID)
	}

	brief := episode.Brief{
		EpisodeType:  episode.OutputType(outputType),
		WorldID:      worldID,
		CharacterIDs: charIDs,
		EventID:      eventID,
		TemplateID:   templateID,
		Tone:         p.universeTone(u),
		Objective:    "Expand canon while preserving coherence",
		CanonRules:   p.collectRules(u, outputType),
	}
	return brief, nil
}

func (p *Planner) pickTemplate(u universe.Universe, outputType string) (string, error) {
	ids := make([]string, 0)
	for id, t := range u.Templates {
		if ot, ok := t.Data["output_type"].(string); ok && ot == outputType {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return "", errors.New("no templates available for output type")
	}
	return ids[p.rng.Intn(len(ids))], nil
}

func (p *Planner) pickCharactersFromIDs(keys []string, n int) ([]string, error) {
	if len(keys) == 0 {
		return nil, errors.New("no compatible characters available")
	}
	if n >= len(keys) {
		out := append([]string(nil), keys...)
		sort.Strings(out)
		return out, nil
	}
	out := make([]string, 0, n)
	perm := p.rng.Perm(len(keys))
	for _, idx := range perm[:n] {
		out = append(out, keys[idx])
	}
	sort.Strings(out)
	return out, nil
}

func weightedPick(rng *rand.Rand, weights map[string]int) (string, error) {
	total := 0
	for _, w := range weights {
		if w > 0 {
			total += w
		}
	}
	if total <= 0 {
		return "", fmt.Errorf("invalid weights")
	}
	n := rng.Intn(total)
	run := 0
	for k, w := range weights {
		if w <= 0 {
			continue
		}
		run += w
		if n < run {
			return k, nil
		}
	}
	return "", fmt.Errorf("weighted selection failed")
}

func containsCombo(recent []HistoryCombo, key string) bool {
	for _, c := range recent {
		if comboKey(c.WorldID, c.CharacterIDs, c.EventID) == key {
			return true
		}
	}
	return false
}

func comboKey(world string, chars []string, event string) string {
	c := append([]string(nil), chars...)
	sort.Strings(c)
	return world + "|" + strings.Join(c, ",") + "|" + event
}

func pickKey[T any](rng *rand.Rand, m map[string]T) string {
	keys := sortedKeys(m)
	if len(keys) == 0 {
		return ""
	}
	return keys[rng.Intn(len(keys))]
}

func pickOne(rng *rand.Rand, items []string) (string, error) {
	if len(items) == 0 {
		return "", errors.New("no compatible items available")
	}
	return items[rng.Intn(len(items))], nil
}

func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (p *Planner) collectRules(u universe.Universe, outputType string) []string {
	rules := make([]string, 0)
	for _, r := range u.Rules {
		t, hasTarget := r.Data["target"].(string)
		if !hasTarget || t == "" {
			// Global rule: no specific target, always applies
			rules = append(rules, r.Body)
			continue
		}
		// Targeted rule: include only when the output type matches
		if t == outputType || (t == "textual" && episode.OutputType(outputType).IsTextual()) {
			rules = append(rules, r.Body)
		}
	}
	if gr, ok := u.Universe.Data["global_rules"]; ok {
		rules = append(rules, toStringSlice(gr)...)
	}
	return rules
}


func (p *Planner) universeTone(u universe.Universe) string {
	if v, ok := u.Universe.Data["creator_presence"].(string); ok && v != "" {
		return v
	}
	if v, ok := u.Universe.Data["summary"].(string); ok {
		return v
	}
	return "consistent"
}

func toStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		if s, ok := v.([]string); ok {
			return s
		}
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, it := range arr {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func compatibleCharacterIDs(u universe.Universe, worldID string) []string {
	out := make([]string, 0, len(u.Characters))
	for id, c := range u.Characters {
		aff := toStringSlice(c.Data["world_affinities"])
		if len(aff) == 0 || containsString(aff, worldID) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func compatibleEventIDs(u universe.Universe, worldID string) []string {
	out := make([]string, 0, len(u.Events))
	for id, ev := range u.Events {
		worlds := toStringSlice(ev.Data["compatible_worlds"])
		if len(worlds) == 0 || containsString(worlds, worldID) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func containsString(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
