package vertexveo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"loreforge/internal/adapters/providers/contracts"
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

func (p Provider) Name() string { return "vertex-veo" }

func (p Provider) GenerateVideo(ctx context.Context, input contracts.VideoRequest) (contracts.VideoResponse, error) {
	projectID, err := sharedauth.RequiredEnv(p.Config.ProjectIDEnv)
	if err != nil {
		return contracts.VideoResponse{}, err
	}
	token, err := sharedauth.GoogleAccessToken()
	if err != nil {
		return contracts.VideoResponse{}, err
	}
	baseURL := strings.TrimRight(p.Config.BaseURL, "/")
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1", p.Config.Location)
	}
	timeout, _ := time.ParseDuration(p.Config.Timeout)
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	pollInterval, _ := time.ParseDuration(p.Config.PollInterval)
	client := sharedhttp.New(timeout)
	if p.HTTP != nil {
		client.HTTP = p.HTTP
	}
	endpoint := fmt.Sprintf("%s/projects/%s/locations/%s/publishers/google/models/%s:predictLongRunning", baseURL, projectID, p.Config.Location, p.Config.Model)
	payload := map[string]any{
		"instances": []map[string]any{{"prompt": input.Prompt}},
		"parameters": map[string]any{
			"storageUri":      p.Config.BucketURI,
			"durationSeconds": maxInt(input.Duration, 8),
			"sampleCount":     maxInt(input.Count, 1),
			"resolution":      firstNonEmpty(input.Resolution, "720p"),
		},
	}
	if len(input.ReferenceImages) > 0 {
		refs := make([]map[string]any, 0, len(input.ReferenceImages))
		for _, ref := range input.ReferenceImages {
			if strings.HasPrefix(p.Config.Model, "veo-3.1") && ref.ReferenceType == "style" {
				return contracts.VideoResponse{}, fmt.Errorf("veo-3.1 does not support style reference images")
			}
			item := map[string]any{"referenceType": ref.ReferenceType}
			switch {
			case ref.URI != "":
				item["referenceImage"] = map[string]any{"gcsUri": ref.URI}
			case ref.Base64 != "":
				item["referenceImage"] = map[string]any{"bytesBase64Encoded": ref.Base64, "mimeType": ref.MIMEType}
			}
			refs = append(refs, item)
		}
		payload["parameters"].(map[string]any)["referenceImages"] = refs
	}
	_, body, err := client.JSON(ctx, http.MethodPost, endpoint, map[string]string{
		"Authorization": "Bearer " + token,
	}, payload)
	if err != nil {
		return contracts.VideoResponse{}, err
	}
	var op struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &op); err != nil {
		return contracts.VideoResponse{}, err
	}
	if op.Name == "" {
		return contracts.VideoResponse{}, fmt.Errorf("vertex veo response missing operation name")
	}
	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var outputURI string
	err = sharedpolling.Until(pollCtx, pollInterval, func(inner context.Context) (bool, error) {
		_, opBody, err := client.JSON(inner, http.MethodGet, baseURL+"/"+op.Name, map[string]string{
			"Authorization": "Bearer " + token,
		}, nil)
		if err != nil {
			return false, err
		}
		var status struct {
			Done     bool           `json:"done"`
			Error    map[string]any `json:"error"`
			Response struct {
				Videos []struct {
					GCSURI string `json:"gcsUri"`
				} `json:"videos"`
			} `json:"response"`
		}
		if err := json.Unmarshal(opBody, &status); err != nil {
			return false, err
		}
		if !status.Done {
			return false, nil
		}
		if len(status.Error) > 0 {
			return false, fmt.Errorf("vertex veo operation failed: %v", status.Error)
		}
		if len(status.Response.Videos) == 0 || status.Response.Videos[0].GCSURI == "" {
			return false, fmt.Errorf("vertex veo response missing output gcs uri")
		}
		outputURI = status.Response.Videos[0].GCSURI
		return true, nil
	})
	if err != nil {
		return contracts.VideoResponse{}, err
	}
	mediaURL, err := sharedfiles.GCSMediaURLWithBase(outputURI, firstNonEmpty(optionString(p.Config.Options, "gcs_base_url", ""), "https://storage.googleapis.com"))
	if err != nil {
		return contracts.VideoResponse{}, err
	}
	path, _, err := sharedfiles.DownloadToTemp(ctx, client.HTTP, mediaURL, "vertex-veo", map[string]string{
		"Authorization": "Bearer " + token,
	})
	if err != nil {
		return contracts.VideoResponse{}, err
	}
	return contracts.VideoResponse{
		AssetPath: path,
		JobID:     op.Name,
		Model:     p.Config.Model,
		Metadata: map[string]any{
			"gcs_uri": outputURI,
			"driver":  p.Config.Driver,
		},
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func optionString(options map[string]any, key, fallback string) string {
	if v, ok := options[key].(string); ok && strings.TrimSpace(v) != "" {
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
