package registry

import (
	"context"
	"testing"

	"loreforge/internal/application/ports"
	"loreforge/internal/domain/publication"
)

func TestRegistryGet(t *testing.T) {
	t.Parallel()

	reg := New([]ports.Publisher{
		stubPublisher{name: publication.ChannelFilesystem},
		stubPublisher{name: publication.ChannelTwitter},
	})
	if _, ok := reg.Get(publication.ChannelFilesystem); !ok {
		t.Fatal("expected filesystem publisher")
	}
	if _, ok := reg.Get("missing"); ok {
		t.Fatal("expected missing publisher lookup to fail")
	}
}

type stubPublisher struct {
	name publication.ChannelName
}

func (s stubPublisher) Name() publication.ChannelName { return s.name }

func (s stubPublisher) Publish(context.Context, publication.Item) (publication.Result, error) {
	return publication.Result{Channel: string(s.name), Success: true}, nil
}
