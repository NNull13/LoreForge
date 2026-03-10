package scheduler_state_fs

import (
	"context"
	"os"
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


func TestRepositoryRejectsEmptyGeneratorID(t *testing.T) {
	t.Parallel()

	repo := Repository{BaseDir: t.TempDir()}
	if err := repo.Save(context.Background(), "", scheduling.State{}); err == nil {
		t.Fatal("expected empty generator id error")
	}
}
