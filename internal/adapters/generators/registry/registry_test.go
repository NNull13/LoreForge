package registry

import (
	"context"
	"testing"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/episode"
)

func TestRegistryGetAndList(t *testing.T) {
	t.Parallel()

	first := ports.RegisteredGenerator{
		Generator: stubGenerator{id: "b", outputType: episode.OutputTypeShortStory},
		Config:    ports.GeneratorConfig{ID: "b", Type: episode.OutputTypeShortStory},
	}
	second := ports.RegisteredGenerator{
		Generator: stubGenerator{id: "a", outputType: episode.OutputTypeImage},
		Config:    ports.GeneratorConfig{ID: "a", Type: episode.OutputTypeImage},
	}
	duplicateType := ports.RegisteredGenerator{
		Generator: stubGenerator{id: "c", outputType: episode.OutputTypeShortStory},
		Config:    ports.GeneratorConfig{ID: "c", Type: episode.OutputTypeShortStory},
	}

	reg := New([]ports.RegisteredGenerator{first, second, duplicateType})
	if got, ok := reg.GetByID("a"); !ok || got.Config.ID != "a" {
		t.Fatalf("GetByID returned %#v, %v", got, ok)
	}
	if got, ok := reg.GetByType(episode.OutputTypeShortStory); !ok || got.Config.ID != "b" {
		t.Fatalf("GetByType returned %#v, %v", got, ok)
	}
	if _, ok := reg.GetByID("missing"); ok {
		t.Fatal("expected missing generator lookup to fail")
	}

	list := reg.List()
	if len(list) != 3 || list[0].Config.ID != "a" || list[1].Config.ID != "b" || list[2].Config.ID != "c" {
		t.Fatalf("unexpected sorted list: %#v", list)
	}
}

type stubGenerator struct {
	id         string
	outputType episode.OutputType
}

func (s stubGenerator) ID() string { return s.id }

func (s stubGenerator) Type() episode.OutputType { return s.outputType }

func (s stubGenerator) Generate(context.Context, episode.Brief, episode.State) (episode.Output, error) {
	return episode.Output{}, nil
}
