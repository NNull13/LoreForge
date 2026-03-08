package lmstudiotext

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"loreforge/internal/adapters/providers/contracts"
	sharedhttp "loreforge/internal/adapters/providers/shared/httpclient"
	sharedtextparse "loreforge/internal/adapters/providers/shared/textparse"
	"loreforge/internal/config"
)

type Provider struct {
	Config config.ProviderDriver
	HTTP   *http.Client
}

func (p Provider) Name() string { return "lmstudio-text" }

func (p Provider) GenerateText(ctx context.Context, input contracts.TextRequest) (contracts.TextResponse, error) {
	mode := optionString(p.Config.Options, "endpoint_mode", "structured_chat")
	if mode == "compat_responses" {
		return p.generateResponses(ctx, input)
	}
	return p.generateChat(ctx, input)
}

func (p Provider) generateChat(ctx context.Context, input contracts.TextRequest) (contracts.TextResponse, error) {
	baseURL := strings.TrimRight(p.Config.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:1234/v1"
	}
	timeout, _ := time.ParseDuration(p.Config.Timeout)
	client := sharedhttp.New(timeout)
	if p.HTTP != nil {
		client.HTTP = p.HTTP
	}
	payload := map[string]any{
		"model": p.Config.Model,
		"messages": []map[string]any{
			{"role": "system", "content": input.SystemPrompt},
			{"role": "user", "content": input.Prompt},
		},
	}
	if input.Temperature > 0 {
		payload["temperature"] = input.Temperature
	}
	if input.MaxOutputTokens > 0 {
		payload["max_tokens"] = input.MaxOutputTokens
	}
	if input.JSONSchema != nil {
		payload["response_format"] = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "text_output",
				"schema": input.JSONSchema,
				"strict": true,
			},
		}
	}
	headers := optionalAuthHeader(p.Config.APIKeyEnv)
	_, body, err := client.JSON(ctx, http.MethodPost, baseURL+"/chat/completions", headers, payload)
	if err != nil {
		return contracts.TextResponse{}, err
	}
	var parsed struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return contracts.TextResponse{}, err
	}
	if len(parsed.Choices) == 0 {
		return contracts.TextResponse{}, fmt.Errorf("lmstudio chat response missing choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	out := contracts.TextResponse{
		Content:      content,
		Model:        p.Config.Model,
		FinishReason: parsed.Choices[0].FinishReason,
		Metadata:     map[string]any{"driver": p.Config.Driver, "endpoint_mode": "structured_chat"},
	}
	if input.JSONSchema != nil {
		content, parts, title, metadata, err := sharedtextparse.ParseStructuredContent(content)
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

func (p Provider) generateResponses(ctx context.Context, input contracts.TextRequest) (contracts.TextResponse, error) {
	baseURL := strings.TrimRight(p.Config.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:1234/v1"
	}
	timeout, _ := time.ParseDuration(p.Config.Timeout)
	client := sharedhttp.New(timeout)
	if p.HTTP != nil {
		client.HTTP = p.HTTP
	}
	payload := map[string]any{
		"model": p.Config.Model,
		"input": []map[string]any{
			{"role": "system", "content": input.SystemPrompt},
			{"role": "user", "content": input.Prompt},
		},
	}
	if input.MaxOutputTokens > 0 {
		payload["max_output_tokens"] = input.MaxOutputTokens
	}
	if input.Temperature > 0 {
		payload["temperature"] = input.Temperature
	}
	headers := optionalAuthHeader(p.Config.APIKeyEnv)
	_, body, err := client.JSON(ctx, http.MethodPost, baseURL+"/responses", headers, payload)
	if err != nil {
		return contracts.TextResponse{}, err
	}
	var parsed struct {
		OutputText string `json:"output_text"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return contracts.TextResponse{}, err
	}
	return contracts.TextResponse{
		Content:  strings.TrimSpace(parsed.OutputText),
		Model:    p.Config.Model,
		Metadata: map[string]any{"driver": p.Config.Driver, "endpoint_mode": "compat_responses"},
	}, nil
}

func optionalAuthHeader(envVar string) map[string]string {
	if strings.TrimSpace(envVar) == "" {
		return nil
	}
	value := strings.TrimSpace(os.Getenv(envVar))
	if value == "" {
		return nil
	}
	return map[string]string{"Authorization": "Bearer " + value}
}

func optionString(options map[string]any, key, fallback string) string {
	if v, ok := options[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
