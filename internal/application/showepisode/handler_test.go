package showepisode

import (
	"context"
	"testing"
	"time"

	"loreforge/internal/domain/episode"
)

func TestHandleReturnsStoredEpisode(t *testing.T) {
	t.Parallel()

	handler := Handler{
		EpisodeRepo: fakeShowEpisodeRepo{
			stored: episode.StoredRecord{
				Path: "/tmp/ep-1",
				Manifest: episode.Manifest{
					EpisodeID: "ep-1",
					CreatedAt: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
				},
			},
		},
	}

	result, err := handler.Handle(context.Background(), Request{EpisodeID: "ep-1"})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if result.Path != "/tmp/ep-1" || result.Manifest.EpisodeID != "ep-1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

type fakeShowEpisodeRepo struct {
	stored episode.StoredRecord
}

func (f fakeShowEpisodeRepo) Save(context.Context, episode.Record) (episode.StoredRecord, error) {
	return episode.StoredRecord{}, nil
}

func (f fakeShowEpisodeRepo) FindByID(context.Context, string) (episode.StoredRecord, error) {
	return f.stored, nil
}

func (f fakeShowEpisodeRepo) RecentCombos(context.Context, int) ([]episode.Combo, error) {
	return nil, nil
}

func (f fakeShowEpisodeRepo) RecentCombosByGenerator(context.Context, string, int) ([]episode.Combo, error) {
	return nil, nil
}

func (f fakeShowEpisodeRepo) RecentReferencesByGenerator(context.Context, string, int) ([]episode.ContinuityReference, error) {
	return nil, nil
}
