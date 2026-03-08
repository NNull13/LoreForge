package planner

import (
	"testing"

	"loreforge/internal/universe"
)

func TestBuildBrief_RespectsWorldCompatibility(t *testing.T) {
	u := universe.Universe{
		Universe: universe.Entity{ID: "u1", Type: "universe", Data: map[string]any{"creator_presence": "steady"}},
		Worlds: map[string]universe.Entity{
			"w1": {ID: "w1", Type: "world"},
			"w2": {ID: "w2", Type: "world"},
		},
		Characters: map[string]universe.Entity{
			"c1": {ID: "c1", Type: "character", Data: map[string]any{"world_affinities": []any{"w1"}}},
			"c2": {ID: "c2", Type: "character", Data: map[string]any{"world_affinities": []any{"w2"}}},
		},
		Events: map[string]universe.Entity{
			"e1": {ID: "e1", Type: "event", Data: map[string]any{"compatible_worlds": []any{"w1"}}},
			"e2": {ID: "e2", Type: "event", Data: map[string]any{"compatible_worlds": []any{"w2"}}},
		},
		Templates: map[string]universe.Entity{
			"t1": {ID: "t1", Type: "template", Data: map[string]any{"output_type": "text"}},
		},
		Rules: map[string]universe.Entity{},
	}

	p := New(Config{Weights: map[string]int{"text": 100}, RecencyWindow: 5, Seed: 7})
	for i := 0; i < 20; i++ {
		brief, err := p.BuildBrief(u, nil)
		if err != nil {
			t.Fatalf("BuildBrief failed: %v", err)
		}
		for _, cid := range brief.CharacterIDs {
			ch := u.Characters[cid]
			aff := toStringSlice(ch.Data["world_affinities"])
			if len(aff) > 0 && !containsString(aff, brief.WorldID) {
				t.Fatalf("character %s incompatible with world %s", cid, brief.WorldID)
			}
		}
		ev := u.Events[brief.EventID]
		cw := toStringSlice(ev.Data["compatible_worlds"])
		if len(cw) > 0 && !containsString(cw, brief.WorldID) {
			t.Fatalf("event %s incompatible with world %s", brief.EventID, brief.WorldID)
		}
	}
}

func TestNew_DeterministicWhenSeededAndNotProduction(t *testing.T) {
	u := universe.Universe{
		Universe:   universe.Entity{ID: "u1", Type: "universe", Data: map[string]any{"creator_presence": "steady"}},
		Worlds:     map[string]universe.Entity{"w1": {ID: "w1", Type: "world"}},
		Characters: map[string]universe.Entity{"c1": {ID: "c1", Type: "character"}},
		Events:     map[string]universe.Entity{"e1": {ID: "e1", Type: "event"}},
		Templates:  map[string]universe.Entity{"t1": {ID: "t1", Type: "template", Data: map[string]any{"output_type": "text"}}},
		Rules:      map[string]universe.Entity{},
	}
	cfg := Config{Weights: map[string]int{"text": 100}, RecencyWindow: 5, Seed: 1234, ProductionMode: false}
	p1 := New(cfg)
	p2 := New(cfg)

	b1, err := p1.BuildBrief(u, nil)
	if err != nil {
		t.Fatalf("BuildBrief p1 failed: %v", err)
	}
	b2, err := p2.BuildBrief(u, nil)
	if err != nil {
		t.Fatalf("BuildBrief p2 failed: %v", err)
	}

	if comboKey(b1.WorldID, b1.CharacterIDs, b1.EventID) != comboKey(b2.WorldID, b2.CharacterIDs, b2.EventID) {
		t.Fatalf("expected deterministic output for same seed")
	}
}
