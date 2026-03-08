package episodestore

import (
	"context"
	"os"
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
			ArtistID:     "short-story-artist",
			OutputType:   "short_story",
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

func TestRecentReferencesByGenerator(t *testing.T) {
	t.Parallel()

	repo := New(filepath.Join(t.TempDir(), "universe.db"))
	assetPath := filepath.Join(t.TempDir(), "frame.png")
	if err := os.WriteFile(assetPath, []byte("not-a-real-image"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	records := []episode.Record{
		{
			Manifest: episode.Manifest{
				EpisodeID:  "ep-1",
				CreatedAt:  time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC),
				ArtistID:   "image-artist",
				OutputType: "image",
			},
			Prompt:          "prompt one",
			OutputAssetPath: assetPath,
		},
		{
			Manifest: episode.Manifest{
				EpisodeID:  "ep-2",
				CreatedAt:  time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
				ArtistID:   "image-artist",
				OutputType: "short_story",
			},
			Prompt:     "prompt two",
			OutputText: "Aria returns to the gate with a silver lantern and hears the old city breathe again.",
		},
	}
	for _, record := range records {
		if _, err := repo.Save(context.Background(), record); err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
	}

	refs, err := repo.RecentReferencesByGenerator(context.Background(), "image-artist", 5)
	if err != nil {
		t.Fatalf("RecentReferencesByGenerator returned error: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("unexpected reference count: %d", len(refs))
	}
	if refs[0].EpisodeID != "ep-2" {
		t.Fatalf("expected newest episode first, got %s", refs[0].EpisodeID)
	}
	if refs[0].Summary == "" {
		t.Fatal("expected text summary for textual output")
	}
	if refs[1].OutputAssetPath == "" {
		t.Fatal("expected visual asset path for image output")
	}
}
