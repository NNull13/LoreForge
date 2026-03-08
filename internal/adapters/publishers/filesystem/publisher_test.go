package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"loreforge/internal/domain/publication"
)

func TestPublishWritesThreadJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	publisher := Publisher{OutputDir: dir}
	result, err := publisher.Publish(context.Background(), publication.Item{
		EpisodeID:  "ep-1",
		OutputType: "tweet_thread",
		Format:     "tweet_thread",
		Content:    "one\n\ntwo",
		Parts:      []string{"one", "two"},
	})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	threadPath, _ := result.Metadata["thread_path"].(string)
	if threadPath == "" {
		t.Fatal("expected thread_path metadata")
	}
	if _, err := os.Stat(threadPath); err != nil {
		t.Fatalf("thread json missing: %v", err)
	}
	if filepath.Ext(threadPath) != ".json" {
		t.Fatalf("unexpected thread file extension: %s", filepath.Ext(threadPath))
	}
}
