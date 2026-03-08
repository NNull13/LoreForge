package universe

import (
	"strings"
	"testing"
)

func TestValidate_FailsOnUnknownReferencedWorld(t *testing.T) {
	u := Universe{
		Universe: Entity{ID: "u1", Type: "universe"},
		Worlds: map[string]Entity{
			"w1": {ID: "w1", Type: "world"},
		},
		Characters: map[string]Entity{
			"c1": {ID: "c1", Type: "character"},
		},
		Events: map[string]Entity{
			"e1": {ID: "e1", Type: "event", Data: map[string]any{"compatible_worlds": []any{"missing-world"}}},
		},
		Templates: map[string]Entity{
			"t1": {ID: "t1", Type: "template"},
		},
		Rules: map[string]Entity{},
	}

	err := Validate(u)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "unknown world") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_PassesForConsistentReferences(t *testing.T) {
	u := Universe{
		Universe: Entity{ID: "u1", Type: "universe"},
		Worlds: map[string]Entity{
			"w1": {ID: "w1", Type: "world"},
		},
		Characters: map[string]Entity{
			"c1": {ID: "c1", Type: "character", Data: map[string]any{"world_affinities": []any{"w1"}}},
		},
		Events: map[string]Entity{
			"e1": {ID: "e1", Type: "event", Data: map[string]any{"compatible_worlds": []any{"w1"}, "compatible_characters": []any{"c1"}}},
		},
		Templates: map[string]Entity{
			"t1": {ID: "t1", Type: "template"},
		},
		Rules: map[string]Entity{},
	}

	if err := Validate(u); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
