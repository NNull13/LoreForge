package vertex_imagen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/adapters/providers/shared/auth"
	"loreforge/internal/adapters/providers/shared/files"
	"loreforge/internal/adapters/providers/shared/http_client"
	"loreforge/internal/config"
)

type Provider struct {
	Config config.ProviderDriver
	HTTP   *http.Client
}

func (p Provider) Name() string { return "vertex-imagen" }

func (p Provider) GenerateImage(ctx context.Context, input contracts.ImageRequest) (contracts.ImageResponse, error) {
	projectID, err := auth.RequiredEnv(p.Config.ProjectIDEnv)
	if err != nil {
		return contracts.ImageResponse{}, err
	}
	token, err := auth.GoogleAccessToken()
	if err != nil {
		return contracts.ImageResponse{}, err
	}
	baseURL := strings.TrimRight(p.Config.BaseURL, "/")
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1", p.Config.Location)
	}
	timeout, _ := time.ParseDuration(p.Config.Timeout)
	client := http_client.New(timeout)
	if p.HTTP != nil {
		client.HTTP = p.HTTP
	}
	aspectRatio := input.AspectRatio
	if aspectRatio == "" && input.Width > 0 && input.Height > 0 {
		aspectRatio = fmt.Sprintf("%d:%d", input.Width, input.Height)
	}
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	payload := map[string]any{
		"instances": []map[string]any{{"prompt": input.Prompt}},
		"parameters": map[string]any{
			"sampleCount":      maxInt(input.Count, 1),
			"aspectRatio":      aspectRatio,
			"addWatermark":     optionBool(p.Config.Options, "add_watermark", true),
			"enhancePrompt":    optionBool(p.Config.Options, "enhance_prompt", false),
			"personGeneration": optionString(p.Config.Options, "person_generation", "allow_adult"),
			"safetySetting":    optionString(p.Config.Options, "safety_setting", "block_medium_and_above"),
			"outputOptions": map[string]any{
				"mimeType": optionString(p.Config.Options, "mime_type", "image/png"),
			},
		},
	}
	if input.Seed != 0 {
		payload["parameters"].(map[string]any)["seed"] = input.Seed
	}
	endpoint := fmt.Sprintf("%s/projects/%s/locations/%s/publishers/google/models/%s:predict", baseURL, projectID, p.Config.Location, p.Config.Model)
	resp, body, err := client.JSON(ctx, http.MethodPost, endpoint, map[string]string{
		"Authorization": "Bearer " + token,
	}, payload)
	if err != nil {
		return contracts.ImageResponse{}, err
	}
	var parsed struct {
		Predictions []struct {
			BytesBase64Encoded string `json:"bytesBase64Encoded"`
			MIMEType           string `json:"mimeType"`
			Prompt             string `json:"prompt"`
		} `json:"predictions"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return contracts.ImageResponse{}, err
	}
	if len(parsed.Predictions) == 0 {
		return contracts.ImageResponse{}, fmt.Errorf("vertex imagen response missing predictions")
	}
	item := parsed.Predictions[0]
	path, err := files.WriteBase64Temp("vertex-imagen", item.MIMEType, item.BytesBase64Encoded)
	if err != nil {
		return contracts.ImageResponse{}, err
	}
	return contracts.ImageResponse{
		AssetPath:     path,
		MIMEType:      item.MIMEType,
		Model:         p.Config.Model,
		RevisedPrompt: item.Prompt,
		Metadata: map[string]any{
			"driver":      p.Config.Driver,
			"status_code": resp.StatusCode,
			"endpoint":    endpoint,
		},
	}, nil
}

func optionString(options map[string]any, key, fallback string) string {
	if v, ok := options[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func optionBool(options map[string]any, key string, fallback bool) bool {
	if v, ok := options[key].(bool); ok {
		return v
	}
	return fallback
}

func maxInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
