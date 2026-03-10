package episode_store

import (
	"context"
	"errors"
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

func TestRecentCombosAndByGenerator(t *testing.T) {
	t.Parallel()

	repo := New(filepath.Join(t.TempDir(), "universe.db"))
	records := []episode.Record{
		{
			Manifest: episode.Manifest{
				EpisodeID:    "ep-1",
				CreatedAt:    time.Date(2026, 3, 8, 8, 0, 0, 0, time.UTC),
				ArtistID:     "story-a",
				WorldIDs:     []string{"world-1"},
				CharacterIDs: []string{"aria"},
				EventID:      "event-1",
			},
			Prompt:           "prompt 1",
			ProviderRequest:  map[string]any{"prompt": 1},
			ProviderResponse: map[string]any{"ok": true},
		},
		{
			Manifest: episode.Manifest{
				EpisodeID:    "ep-2",
				CreatedAt:    time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC),
				ArtistID:     "story-b",
				WorldIDs:     []string{"world-2"},
				CharacterIDs: []string{"kade", "aria"},
				EventID:      "event-2",
			},
			Prompt:           "prompt 2",
			ProviderRequest:  map[string]any{"prompt": 2},
			ProviderResponse: map[string]any{"ok": true},
		},
		{
			Manifest: episode.Manifest{
				EpisodeID:    "ep-3",
				CreatedAt:    time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
				ArtistID:     "story-a",
				WorldIDs:     []string{"world-3"},
				CharacterIDs: []string{"mira"},
				EventID:      "event-3",
			},
			Prompt:           "prompt 3",
			ProviderRequest:  map[string]any{"prompt": 3},
			ProviderResponse: map[string]any{"ok": true},
		},
	}
	for _, record := range records {
		if _, err := repo.Save(context.Background(), record); err != nil {
			t.Fatalf("Save returned error: %v", err)
		}
	}

	if combos, err := repo.RecentCombos(context.Background(), 0); err != nil || combos != nil {
		t.Fatalf("RecentCombos limit 0 = %#v, %v; want nil, nil", combos, err)
	}

	combos, err := repo.RecentCombos(context.Background(), 2)
	if err != nil {
		t.Fatalf("RecentCombos returned error: %v", err)
	}
	if len(combos) != 2 || combos[0].WorldID != "world-3" || combos[1].WorldID != "world-2" {
		t.Fatalf("unexpected combos: %#v", combos)
	}

	byGenerator, err := repo.RecentCombosByGenerator(context.Background(), "story-a", 10)
	if err != nil {
		t.Fatalf("RecentCombosByGenerator returned error: %v", err)
	}
	if len(byGenerator) != 2 || byGenerator[0].WorldID != "world-3" || byGenerator[1].WorldID != "world-1" {
		t.Fatalf("unexpected by-generator combos: %#v", byGenerator)
	}
}

func TestRecentCombosRejectsInvalidCharacterJSON(t *testing.T) {
	t.Parallel()

	repo := New(filepath.Join(t.TempDir(), "universe.db"))
	if err := repo.ensureDB(); err != nil {
		t.Fatalf("ensureDB returned error: %v", err)
	}
	_, err := repo.db.Exec(`
		INSERT INTO episodes(id, path, generator_id, created_at, world_id, character_ids, event_id)
		VALUES(?, ?, ?, ?, ?, ?, ?)`,
		"ep-bad",
		filepath.Join(t.TempDir(), "bad"),
		"story-a",
		time.Date(2026, 3, 8, 11, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		"world-1",
		"{bad-json}",
		"event-1",
	)
	if err != nil {
		t.Fatalf("insert returned error: %v", err)
	}

	if _, err := repo.RecentCombos(context.Background(), 1); err == nil {
		t.Fatal("expected invalid character json error")
	}
}

func TestSaveWritesOptionalArtifactsAndCopiesAsset(t *testing.T) {
	t.Parallel()

	repo := New(filepath.Join(t.TempDir(), "universe.db"))
	sourceAsset := filepath.Join(t.TempDir(), "frame.png")
	if err := os.WriteFile(sourceAsset, []byte("png-data"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	record := episode.Record{
		Manifest: episode.Manifest{
			EpisodeID:  "ep-optional",
			CreatedAt:  time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
			ArtistID:   "image-artist",
			OutputType: "image",
			Scores:     map[string]any{"length_ok": true},
		},
		Context:          map[string]any{"ok": true},
		Prompt:           "prompt",
		ProviderRequest:  map[string]any{"prompt": "draw"},
		ProviderResponse: map[string]any{"ok": true},
		OutputText:       "Rendered output summary",
		OutputParts:      []string{"part-1", "part-2"},
		OutputAssetPath:  sourceAsset,
		ArtistSnapshot:   map[string]any{"id": "image-artist"},
		Presentation:     map[string]any{"filesystem": map[string]any{"intro": "Intro"}},
		Publish:          map[string]any{"filesystem": map[string]any{"success": true}},
	}

	stored, err := repo.Save(context.Background(), record)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	for _, name := range []string{
		"manifest.json",
		"context.json",
		"prompt.txt",
		"provider_request.json",
		"provider_response.json",
		"score.json",
		"publish.json",
		"presentation.json",
		"artist_snapshot.json",
		"output.txt",
		"output_parts.json",
		"frame.png",
	} {
		if _, err := os.Stat(filepath.Join(stored.Path, name)); err != nil {
			t.Fatalf("expected snapshot file %s: %v", name, err)
		}
	}

	if got := filepath.Base(findOutputAsset(stored.Path)); got != "frame.png" {
		t.Fatalf("findOutputAsset = %q, want frame.png", got)
	}
}

func TestFindByIDErrors(t *testing.T) {
	t.Parallel()

	repo := New(filepath.Join(t.TempDir(), "universe.db"))
	if _, err := repo.FindByID(context.Background(), "missing"); !errors.Is(err, episode.ErrEpisodeNotFound) {
		t.Fatalf("FindByID missing err = %v, want episode not found", err)
	}

	stored, err := repo.Save(context.Background(), episode.Record{
		Manifest: episode.Manifest{
			EpisodeID: "ep-corrupt",
			CreatedAt: time.Date(2026, 3, 8, 13, 0, 0, 0, time.UTC),
			ArtistID:  "story-a",
		},
		Prompt:           "prompt",
		ProviderRequest:  map[string]any{"prompt": "hello"},
		ProviderResponse: map[string]any{"ok": true},
	})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stored.Path, "manifest.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("corrupt manifest: %v", err)
	}
	if _, err := repo.FindByID(context.Background(), "ep-corrupt"); err == nil {
		t.Fatal("expected invalid manifest error")
	}
}

func TestBaseDirFromDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{name: "empty uses data dir", dsn: "", want: "./data"},
		{name: "basename uses data dir", dsn: "universe.db", want: "./data"},
		{name: "relative path keeps parent", dsn: "./state/universe.db", want: "state"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BaseDirFromDSN(tt.dsn); got != tt.want {
				t.Fatalf("BaseDirFromDSN(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}
