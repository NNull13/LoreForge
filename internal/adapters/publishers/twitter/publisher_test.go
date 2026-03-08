package twitter

import (
	"context"
	"testing"

	"loreforge/internal/domain/publication"
)

func TestPublishDryRunThreadReturnsTweetIDs(t *testing.T) {
	t.Parallel()

	publisher := Publisher{DryRun: true}
	result, err := publisher.Publish(context.Background(), publication.Item{
		EpisodeID:  "ep-1",
		OutputType: "tweet_thread",
		Parts:      []string{"one", "two", "three"},
	})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	ids, ok := result.Metadata["tweet_ids"].([]string)
	if !ok {
		t.Fatalf("expected tweet_ids metadata, got %#v", result.Metadata)
	}
	if len(ids) != 3 {
		t.Fatalf("unexpected dry-run thread ids: %#v", ids)
	}
}
