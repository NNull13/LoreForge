package twitter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
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

func TestPublishRequiresTokenWhenLive(t *testing.T) {
	t.Parallel()

	publisher := Publisher{DryRun: false}
	if _, err := publisher.Publish(context.Background(), publication.Item{EpisodeID: "ep-1", Content: "hello"}); err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestPublishSingleAndThread(t *testing.T) {
	type request struct {
		Text  string         `json:"text"`
		Reply map[string]any `json:"reply"`
	}
	var requests []request
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/2/tweets" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload request
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		requests = append(requests, payload)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"data":{"id":"tweet-id"}}`)),
		}, nil
	})}

	t.Setenv("TWITTER_BEARER_TOKEN", "secret")
	publisher := Publisher{
		DryRun:  false,
		BaseURL: "https://twitter.test",
		Client:  client,
	}

	result, err := publisher.Publish(context.Background(), publication.Item{
		EpisodeID:  "ep-2",
		OutputType: "tweet_short",
		Content:    strings.Repeat("A", 400),
	})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if result.ExternalID != "tweet-id" {
		t.Fatalf("unexpected external id: %s", result.ExternalID)
	}

	threadResult, err := publisher.Publish(context.Background(), publication.Item{
		EpisodeID:  "ep-3",
		OutputType: "tweet_thread",
		Parts:      []string{"one", "two"},
	})
	if err != nil {
		t.Fatalf("Publish thread returned error: %v", err)
	}
	if threadResult.ExternalID != "tweet-id" || len(requests) != 3 {
		t.Fatalf("unexpected thread publish result: %#v %#v", threadResult, requests)
	}
	if requests[2].Reply["in_reply_to_tweet_id"] != "tweet-id" {
		t.Fatalf("expected reply chain in thread payload: %#v", requests[2])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
