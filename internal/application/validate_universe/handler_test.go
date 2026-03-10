package validate_universe

import (
	"context"
	"errors"
	"testing"

	domainuniverse "loreforge/internal/domain/universe"
)

func TestHandleWrapsUniverseErrors(t *testing.T) {
	t.Parallel()

	handler := Handler{UniverseRepo: fakeValidateUniverseRepo{err: errors.New("bad universe")}}
	if err := handler.Handle(context.Background()); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestHandleAcceptsValidUniverse(t *testing.T) {
	t.Parallel()

	handler := Handler{UniverseRepo: fakeValidateUniverseRepo{universe: domainuniverse.Universe{}}}
	if err := handler.Handle(context.Background()); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
}

type fakeValidateUniverseRepo struct {
	universe domainuniverse.Universe
	err      error
}

func (f fakeValidateUniverseRepo) Load(context.Context) (domainuniverse.Universe, error) {
	return f.universe, f.err
}
