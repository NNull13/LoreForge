package registry

import (
	"loreforge/internal/application/ports"
	"loreforge/internal/domain/publication"
)

type Registry struct {
	items map[publication.ChannelName]ports.Publisher
}

func New(publishers []ports.Publisher) *Registry {
	items := make(map[publication.ChannelName]ports.Publisher, len(publishers))
	for _, publisher := range publishers {
		items[publisher.Name()] = publisher
	}
	return &Registry{items: items}
}

func (r *Registry) Get(name publication.ChannelName) (ports.Publisher, bool) {
	publisher, ok := r.items[name]
	return publisher, ok
}
