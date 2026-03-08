package episodestore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"loreforge/internal/domain/episode"
)

func TestSaveAndFindByID(t *testing.T) {
	t.Parallel()

	repo := New(filepath.Join(t.TempDir(), "universe.db"))
	record := episode.Record{
		Manifest: episode.Manifest{
			EpisodeID:    "ep-abc",
			CreatedAt:    time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
			ArtistID:     "text-artist",
			OutputType:   "text",
			WorldIDs:     []string{"world-1"},
			CharacterIDs: []string{"aria"},
			EventID:      "event-1",
		},
		Prompt:           "prompt",
		ProviderRequest:  map[string]any{"prompt": "prompt"},
		ProviderResponse: map[string]any{"content": "content"},
		OutputText:       "Aria opens the iron gate and hears the city answer back.",
	}

	stored, err := repo.Save(context.Background(), record)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if stored.Path == "" {
		t.Fatal("expected stored path")
	}

	found, err := repo.FindByID(context.Background(), "ep-abc")
	if err != nil {
		t.Fatalf("FindByID returned error: %v", err)
	}
	if found.Path != stored.Path {
		t.Fatalf("unexpected path: got %s want %s", found.Path, stored.Path)
	}
	if found.Manifest.EpisodeID != "ep-abc" {
		t.Fatalf("unexpected manifest id: %s", found.Manifest.EpisodeID)
	}
}
