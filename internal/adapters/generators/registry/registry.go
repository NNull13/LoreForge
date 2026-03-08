package registry

import (
	"sort"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/episode"
)

type Registry struct {
	byID   map[string]ports.RegisteredGenerator
	byType map[episode.OutputType]ports.RegisteredGenerator
}

func New(defs []ports.RegisteredGenerator) *Registry {
	byID := make(map[string]ports.RegisteredGenerator, len(defs))
	byType := make(map[episode.OutputType]ports.RegisteredGenerator, len(defs))
	for _, def := range defs {
		byID[def.Config.ID] = def
		if _, exists := byType[def.Config.Type]; !exists {
			byType[def.Config.Type] = def
		}
	}
	return &Registry{byID: byID, byType: byType}
}

func (r *Registry) GetByID(id string) (ports.RegisteredGenerator, bool) {
	def, ok := r.byID[id]
	return def, ok
}

func (r *Registry) GetByType(outputType episode.OutputType) (ports.RegisteredGenerator, bool) {
	def, ok := r.byType[outputType]
	return def, ok
}

func (r *Registry) List() []ports.RegisteredGenerator {
	keys := make([]string, 0, len(r.byID))
	for key := range r.byID {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]ports.RegisteredGenerator, 0, len(keys))
	for _, key := range keys {
		out = append(out, r.byID[key])
	}
	return out
}
