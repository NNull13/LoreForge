package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"loreforge/internal/domain/publication"
)

type Publisher struct {
	DryRun         bool
	BearerTokenEnv string
	BaseURL        string
	Client         *http.Client
}

func (p Publisher) Name() publication.ChannelName { return publication.ChannelTwitter }

func (p Publisher) Publish(ctx context.Context, item publication.Item) (publication.Result, error) {
	if p.DryRun {
		if len(item.Parts) > 0 {
			ids := make([]string, 0, len(item.Parts))
			for i := range item.Parts {
				ids = append(ids, fmt.Sprintf("dry-run-%d", i+1))
			}
			return publication.Result{
				Channel:    string(p.Name()),
				Success:    true,
				ExternalID: ids[0],
				Message:    "twitter dry-run thread publish",
				Metadata:   map[string]any{"tweet_ids": ids},
			}, nil
		}
		return publication.Result{
			Channel:    string(p.Name()),
			Success:    true,
			ExternalID: "dry-run",
			Message:    "twitter dry-run publish",
		}, nil
	}
	tokenVar := p.BearerTokenEnv
	if tokenVar == "" {
		tokenVar = "TWITTER_BEARER_TOKEN"
	}
	token := strings.TrimSpace(os.Getenv(tokenVar))
	if token == "" {
		return publication.Result{}, fmt.Errorf("twitter bearer token not set in %s", tokenVar)
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = "https://api.twitter.com"
	}
	if len(item.Parts) > 0 {
		return p.publishThread(ctx, client, base, token, item)
	}
	text := strings.TrimSpace(item.Content)
	if text == "" {
		text = fmt.Sprintf("New %s piece by %s (%s).", item.OutputType, item.GeneratorID, item.EpisodeID)
	}
	if len(text) > 280 {
		text = text[:280]
	}
	payload := map[string]any{"text": text}
	return p.publishSingle(ctx, client, base, token, payload)
}

func (p Publisher) publishThread(ctx context.Context, client *http.Client, base, token string, item publication.Item) (publication.Result, error) {
	var parentID string
	tweetIDs := make([]string, 0, len(item.Parts))
	for idx, part := range item.Parts {
		text := strings.TrimSpace(part)
		if len([]rune(text)) > 280 {
			text = string([]rune(text)[:280])
		}
		payload := map[string]any{"text": text}
		if parentID != "" {
			payload["reply"] = map[string]any{"in_reply_to_tweet_id": parentID}
		}
		result, err := p.publishSingle(ctx, client, base, token, payload)
		if err != nil {
			return publication.Result{
				Channel: string(p.Name()),
				Success: false,
				Message: fmt.Sprintf("twitter thread failed after %d tweets", idx),
				Metadata: map[string]any{
					"tweet_ids": tweetIDs,
				},
			}, err
		}
		parentID = result.ExternalID
		tweetIDs = append(tweetIDs, result.ExternalID)
	}
	return publication.Result{
		Channel:    string(p.Name()),
		Success:    true,
		ExternalID: firstTweetID(tweetIDs),
		Message:    "twitter thread published",
		Metadata:   map[string]any{"tweet_ids": tweetIDs},
	}, nil
}

func (p Publisher) publishSingle(ctx context.Context, client *http.Client, base, token string, payload map[string]any) (publication.Result, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return publication.Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/2/tweets", bytes.NewReader(body))
	if err != nil {
		return publication.Result{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return publication.Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return publication.Result{}, fmt.Errorf("twitter publish failed: status %d", resp.StatusCode)
	}
	var parsed struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	return publication.Result{
		Channel:    string(p.Name()),
		Success:    true,
		ExternalID: parsed.Data.ID,
		Message:    "tweet published",
	}, nil
}

func firstTweetID(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}
