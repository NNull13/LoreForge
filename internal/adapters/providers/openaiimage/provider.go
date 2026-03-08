package openaiimage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	providercontracts "loreforge/internal/adapters/providers/contracts"
	sharedauth "loreforge/internal/adapters/providers/shared/auth"
	sharedfiles "loreforge/internal/adapters/providers/shared/files"
	sharedhttp "loreforge/internal/adapters/providers/shared/httpclient"
	"loreforge/internal/config"
)

type Provider struct {
	Config config.ProviderDriver
	HTTP   *http.Client
}

func (p Provider) Name() string { return "openai-image" }

func (p Provider) GenerateImage(ctx context.Context, input providercontracts.ImageRequest) (providercontracts.ImageResponse, error) {
	token, err := sharedauth.BearerTokenFromEnv(p.Config.APIKeyEnv)
	if err != nil {
		return providercontracts.ImageResponse{}, err
	}
	baseURL := strings.TrimRight(p.Config.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	timeout, _ := time.ParseDuration(p.Config.Timeout)
	client := sharedhttp.New(timeout)
	if p.HTTP != nil {
		client.HTTP = p.HTTP
	}
	payload := map[string]any{
		"model":           p.Config.Model,
		"prompt":          input.Prompt,
		"response_format": optionString(p.Config.Options, "response_format", "b64_json"),
	}
	if input.Quality != "" {
		payload["quality"] = input.Quality
	} else if v := optionString(p.Config.Options, "quality", ""); v != "" {
		payload["quality"] = v
	}
	if input.Background != "" {
		payload["background"] = input.Background
	} else if v := optionString(p.Config.Options, "background", ""); v != "" {
		payload["background"] = v
	}
	if input.ReferenceImage != "" || input.MaskImage != "" {
		return providercontracts.ImageResponse{}, fmt.Errorf("openai image edits are not implemented yet")
	}
	resp, body, err := client.JSON(ctx, http.MethodPost, baseURL+"/images/generations", map[string]string{
		"Authorization": "Bearer " + token,
	}, payload)
	if err != nil {
		return providercontracts.ImageResponse{}, err
	}
	var parsed struct {
		Data []struct {
			B64JSON       string `json:"b64_json"`
			URL           string `json:"url"`
			RevisedPrompt string `json:"revised_prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return providercontracts.ImageResponse{}, err
	}
	if len(parsed.Data) == 0 {
		return providercontracts.ImageResponse{}, fmt.Errorf("openai image response missing data")
	}
	item := parsed.Data[0]
	out := providercontracts.ImageResponse{
		Model:         p.Config.Model,
		RevisedPrompt: item.RevisedPrompt,
		Metadata: map[string]any{
			"status_code": resp.StatusCode,
			"driver":      p.Config.Driver,
		},
	}
	if strings.HasPrefix(strings.ToLower(p.Config.Model), "dall-e-") {
		out.Metadata["warning"] = "DALL-E 2 and DALL-E 3 are deprecated by OpenAI and documented as supported only until May 12, 2026."
	}
	if item.B64JSON != "" {
		path, err := sharedfiles.WriteBase64Temp("openai-image", "image/png", item.B64JSON)
		if err != nil {
			return providercontracts.ImageResponse{}, err
		}
		out.AssetPath = path
		out.MIMEType = "image/png"
		return out, nil
	}
	if item.URL != "" {
		path, mimeType, err := sharedfiles.DownloadToTemp(ctx, client.HTTP, item.URL, "openai-image", nil)
		if err != nil {
			return providercontracts.ImageResponse{}, err
		}
		out.URL = item.URL
		out.AssetPath = path
		out.MIMEType = mimeType
		return out, nil
	}
	return providercontracts.ImageResponse{}, fmt.Errorf("openai image response missing b64_json and url")
}

func optionString(options map[string]any, key, fallback string) string {
	if v, ok := options[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}
