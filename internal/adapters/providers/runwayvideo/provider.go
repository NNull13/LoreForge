package runwayvideo

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
	sharedpolling "loreforge/internal/adapters/providers/shared/polling"
	"loreforge/internal/config"
)

type Provider struct {
	Config config.ProviderDriver
	HTTP   *http.Client
}

func (p Provider) Name() string { return "runway-gen4" }

func (p Provider) GenerateVideo(ctx context.Context, input providercontracts.VideoRequest) (providercontracts.VideoResponse, error) {
	if strings.TrimSpace(input.PromptImage) == "" {
		return providercontracts.VideoResponse{}, fmt.Errorf("runway_gen4 requires prompt_image")
	}
	apiKey, err := sharedauth.BearerTokenFromEnv(p.Config.APIKeyEnv)
	if err != nil {
		return providercontracts.VideoResponse{}, err
	}
	baseURL := strings.TrimRight(p.Config.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.dev.runwayml.com/v1"
	}
	timeout, _ := time.ParseDuration(p.Config.Timeout)
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	pollInterval, _ := time.ParseDuration(p.Config.PollInterval)
	if pollInterval < 5*time.Second {
		pollInterval = 5 * time.Second
	}
	client := sharedhttp.New(timeout)
	if p.HTTP != nil {
		client.HTTP = p.HTTP
	}
	asset := input.PromptImage
	if !isRemoteAsset(asset) {
		asset, err = sharedfiles.ToDataURI(asset)
		if err != nil {
			return providercontracts.VideoResponse{}, err
		}
	}
	payload := map[string]any{
		"model":       p.Config.Model,
		"promptText":  input.Prompt,
		"promptImage": asset,
		"duration":    maxInt(input.Duration, 5),
		"ratio":       firstNonEmpty(input.AspectRatio, "1280:720"),
	}
	_, body, err := client.JSON(ctx, http.MethodPost, baseURL+"/image_to_video", map[string]string{
		"Authorization":    "Bearer " + apiKey,
		"X-Runway-Version": firstNonEmpty(p.Config.Version, "2024-11-06"),
	}, payload)
	if err != nil {
		return providercontracts.VideoResponse{}, err
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		return providercontracts.VideoResponse{}, err
	}
	if created.ID == "" {
		return providercontracts.VideoResponse{}, fmt.Errorf("runway task id missing")
	}
	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var outputURL string
	err = sharedpolling.Until(pollCtx, pollInterval, func(inner context.Context) (bool, error) {
		_, taskBody, err := client.JSON(inner, http.MethodGet, baseURL+"/tasks/"+created.ID, map[string]string{
			"Authorization":    "Bearer " + apiKey,
			"X-Runway-Version": firstNonEmpty(p.Config.Version, "2024-11-06"),
		}, nil)
		if err != nil {
			return false, err
		}
		var task struct {
			Status  string   `json:"status"`
			Output  []string `json:"output"`
			Failure string   `json:"failure"`
		}
		if err := json.Unmarshal(taskBody, &task); err != nil {
			return false, err
		}
		switch strings.ToUpper(task.Status) {
		case "SUCCEEDED":
			if len(task.Output) == 0 {
				return false, fmt.Errorf("runway task succeeded without output")
			}
			outputURL = task.Output[0]
			return true, nil
		case "FAILED", "CANCELLED":
			return false, fmt.Errorf("runway task failed: %s", task.Failure)
		default:
			return false, nil
		}
	})
	if err != nil {
		return providercontracts.VideoResponse{}, err
	}
	path, _, err := sharedfiles.DownloadToTemp(ctx, client.HTTP, outputURL, "runway-video", nil)
	if err != nil {
		return providercontracts.VideoResponse{}, err
	}
	return providercontracts.VideoResponse{
		AssetPath: path,
		URL:       outputURL,
		JobID:     created.ID,
		Model:     p.Config.Model,
		Metadata:  map[string]any{"driver": p.Config.Driver},
	}, nil
}

func isRemoteAsset(value string) bool {
	return strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "data:")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func maxInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
