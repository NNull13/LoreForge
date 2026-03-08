package schedulerstatefs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"loreforge/internal/domain/scheduling"
)

func TestRepositoryRoundTripsEncodedIDs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repo := Repository{BaseDir: dir}
	state := scheduling.State{NextRunAt: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)}

	if err := repo.Save(context.Background(), "../artist", state); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("unexpected file count: %d", len(entries))
	}
	if name := entries[0].Name(); strings.Contains(name, "..") || strings.Contains(name, "/") {
		t.Fatalf("unsafe scheduler state filename: %s", name)
	}

	loaded, err := repo.Load(context.Background(), "../artist")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !loaded.NextRunAt.Equal(state.NextRunAt) {
		t.Fatalf("loaded next run = %s, want %s", loaded.NextRunAt, state.NextRunAt)
	}

	ids, err := repo.ListGeneratorIDs(context.Background())
	if err != nil {
		t.Fatalf("ListGeneratorIDs returned error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "../artist" {
		t.Fatalf("unexpected ids: %#v", ids)
	}
}

func TestRepositoryLoadsLegacyStateFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "scheduler_state_legacy-artist.json")
	content, err := json.MarshalIndent(scheduling.State{
		NextRunAt: time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC),
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	repo := Repository{BaseDir: dir}
	state, err := repo.Load(context.Background(), "legacy-artist")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if state.NextRunAt.IsZero() {
		t.Fatal("expected legacy next run to be loaded")
	}
	ids, err := repo.ListGeneratorIDs(context.Background())
	if err != nil {
		t.Fatalf("ListGeneratorIDs returned error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "legacy-artist" {
		t.Fatalf("unexpected ids: %#v", ids)
	}
}

func TestRepositoryRejectsEmptyGeneratorID(t *testing.T) {
	t.Parallel()

	repo := Repository{BaseDir: t.TempDir()}
	if err := repo.Save(context.Background(), "", scheduling.State{}); err == nil {
		t.Fatal("expected empty generator id error")
	}
}
