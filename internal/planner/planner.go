package planner

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"loreforge/internal/universe"
	"loreforge/pkg/contracts"
)

type HistoryCombo struct {
	WorldID      string
	CharacterIDs []string
	EventID      string
}

type Config struct {
	Weights       map[string]int
	RecencyWindow int
	Seed          int64
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
	if seed == 0 {
		seed = 42
	}
	return &Planner{rng: rand.New(rand.NewSource(seed)), cfg: cfg}
}

func (p *Planner) BuildBrief(u universe.Universe, recent []HistoryCombo) (contracts.EpisodeBrief, error) {
	episodeType, err := weightedPick(p.rng, p.cfg.Weights)
	if err != nil {
		return contracts.EpisodeBrief{}, err
	}
	worldID := pickKey(p.rng, u.Worlds)
	eventID := pickKey(p.rng, u.Events)
	templateID, err := p.pickTemplate(u, episodeType)
	if err != nil {
		return contracts.EpisodeBrief{}, err
	}
	charIDs := p.pickCharacters(u, 1+p.rng.Intn(2))

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
		eventID = pickKey(p.rng, u.Events)
		charIDs = p.pickCharacters(u, 1+p.rng.Intn(2))
		candidate = comboKey(worldID, charIDs, eventID)
	}

	brief := contracts.EpisodeBrief{
		EpisodeType:  episodeType,
		WorldID:      worldID,
		CharacterIDs: charIDs,
		EventID:      eventID,
		TemplateID:   templateID,
		Tone:         p.universeTone(u),
		Objective:    "Expand canon while preserving coherence",
		CanonRules:   p.collectRules(u, episodeType),
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
		for id := range u.Templates {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return "", errors.New("no templates available")
	}
	return ids[p.rng.Intn(len(ids))], nil
}

func (p *Planner) pickCharacters(u universe.Universe, n int) []string {
	keys := sortedKeys(u.Characters)
	if n >= len(keys) {
		return keys
	}
	out := make([]string, 0, n)
	perm := p.rng.Perm(len(keys))
	for _, idx := range perm[:n] {
		out = append(out, keys[idx])
	}
	sort.Strings(out)
	return out
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
	return keys[rng.Intn(len(keys))]
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
		rules = append(rules, r.Body)
		if t, ok := r.Data["target"].(string); ok && t == outputType {
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
