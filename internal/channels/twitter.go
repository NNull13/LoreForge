package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"loreforge/pkg/contracts"
)

type TwitterChannel struct {
	DryRun         bool
	BearerTokenEnv string
	BaseURL        string
	Client         *http.Client
}

func (t TwitterChannel) Name() string { return "twitter" }

func (t TwitterChannel) Publish(ctx context.Context, item contracts.PublishableContent) (contracts.PublishResult, error) {
	if t.DryRun {
		return contracts.PublishResult{
			Channel:    t.Name(),
			Success:    true,
			ExternalID: "dry-run",
			Message:    "twitter dry-run publish",
		}, nil
	}
	tokenVar := t.BearerTokenEnv
	if tokenVar == "" {
		tokenVar = "TWITTER_BEARER_TOKEN"
	}
	token := strings.TrimSpace(os.Getenv(tokenVar))
	if token == "" {
		return contracts.PublishResult{}, fmt.Errorf("twitter bearer token not set in %s", tokenVar)
	}
	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	base := strings.TrimRight(t.BaseURL, "/")
	if base == "" {
		base = "https://api.twitter.com"
	}

	text := strings.TrimSpace(item.Content)
	if text == "" {
		text = fmt.Sprintf("New %s piece by %s (%s).", item.OutputType, item.ArtistID, item.EpisodeID)
	}
	if len(text) > 280 {
		text = text[:280]
	}
	payload := map[string]any{"text": text}
	body, err := json.Marshal(payload)
	if err != nil {
		return contracts.PublishResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/2/tweets", bytes.NewReader(body))
	if err != nil {
		return contracts.PublishResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return contracts.PublishResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return contracts.PublishResult{}, fmt.Errorf("twitter publish failed: status %d", resp.StatusCode)
	}

	var parsed struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)

	return contracts.PublishResult{
		Channel:    t.Name(),
		Success:    true,
		ExternalID: parsed.Data.ID,
		Message:    "tweet published",
	}, nil
}
