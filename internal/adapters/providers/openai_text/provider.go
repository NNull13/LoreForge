package openai_text

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/adapters/providers/shared/auth"
	"loreforge/internal/adapters/providers/shared/http_client"
	"loreforge/internal/adapters/providers/shared/text_parse"
	"loreforge/internal/config"
)

type Provider struct {
	Config config.ProviderDriver
	HTTP   *http.Client
}

func (p Provider) Name() string { return "openai-text" }

func (p Provider) GenerateText(ctx context.Context, input contracts.TextRequest) (contracts.TextResponse, error) {
	token, err := auth.BearerTokenFromEnv(p.Config.APIKeyEnv)
	if err != nil {
		return contracts.TextResponse{}, err
	}
	baseURL := strings.TrimRight(p.Config.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	timeout, _ := time.ParseDuration(p.Config.Timeout)
	client := http_client.New(timeout)
	if p.HTTP != nil {
		client.HTTP = p.HTTP
	}
	payload := map[string]any{
		"model": p.Config.Model,
		"input": []map[string]any{
			{
				"role": "system",
				"content": []map[string]any{
					{"type": "input_text", "text": input.SystemPrompt},
				},
			},
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": input.Prompt},
				},
			},
		},
	}
	if input.Temperature > 0 {
		payload["temperature"] = input.Temperature
	}
	if input.MaxOutputTokens > 0 {
		payload["max_output_tokens"] = input.MaxOutputTokens
	}
	if input.JSONSchema != nil {
		payload["text"] = map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "text_output",
				"schema": input.JSONSchema,
				"strict": true,
			},
		}
	}
	resp, body, err := client.JSON(ctx, http.MethodPost, baseURL+"/responses", map[string]string{
		"Authorization": "Bearer " + token,
	}, payload)
	if err != nil {
		return contracts.TextResponse{}, err
	}
	content, finishReason, err := responseOutputText(body)
	if err != nil {
		return contracts.TextResponse{}, err
	}
	out := contracts.TextResponse{
		Content:      content,
		Model:        p.Config.Model,
		FinishReason: finishReason,
		Metadata: map[string]any{
			"driver":      p.Config.Driver,
			"status_code": resp.StatusCode,
		},
	}
	if input.JSONSchema != nil {
		content, parts, title, metadata, err := text_parse.ParseStructuredContent(content)
		if err != nil {
			return contracts.TextResponse{}, err
		}
		out.Content = content
		out.Parts = parts
		out.Title = title
		out.Metadata["structured"] = metadata
	}
	return out, nil
}

func responseOutputText(body []byte) (string, string, error) {
	var parsed struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Status  string `json:"status"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(parsed.OutputText) != "" {
		return strings.TrimSpace(parsed.OutputText), firstStatus(parsed.Output), nil
	}
	parts := make([]string, 0)
	for _, item := range parsed.Output {
		for _, content := range item.Content {
			if strings.TrimSpace(content.Text) != "" {
				parts = append(parts, strings.TrimSpace(content.Text))
			}
		}
	}
	if len(parts) == 0 {
		return "", "", fmt.Errorf("openai text response missing output text")
	}
	return strings.Join(parts, "\n"), firstStatus(parsed.Output), nil
}

func firstStatus(items []struct {
	Status  string `json:"status"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}) string {
	if len(items) == 0 {
		return ""
	}
	return items[0].Status
}
