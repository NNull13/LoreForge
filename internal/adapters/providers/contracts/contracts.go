package contracts

import "context"

type ReferenceImage struct {
	URI           string `json:"uri,omitempty"`
	Base64        string `json:"base64,omitempty"`
	MIMEType      string `json:"mime_type,omitempty"`
	ReferenceType string `json:"reference_type"`
}

type TextRequest struct {
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

type TextResponse struct {
	Content string `json:"content"`
	Model   string `json:"model"`
}

type TextProvider interface {
	GenerateText(ctx context.Context, input TextRequest) (TextResponse, error)
	Name() string
}

type VideoRequest struct {
	Prompt          string           `json:"prompt"`
	PromptImage     string           `json:"prompt_image,omitempty"`
	ReferenceImages []ReferenceImage `json:"reference_images,omitempty"`
	Duration        int              `json:"duration_seconds"`
	Seed            int64            `json:"seed,omitempty"`
	AspectRatio     string           `json:"aspect_ratio,omitempty"`
	Resolution      string           `json:"resolution,omitempty"`
	Count           int              `json:"count,omitempty"`
	Options         map[string]any   `json:"options,omitempty"`
}

type VideoResponse struct {
	AssetPath string         `json:"asset_path,omitempty"`
	URL       string         `json:"url,omitempty"`
	JobID     string         `json:"job_id,omitempty"`
	Model     string         `json:"model"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type VideoProvider interface {
	GenerateVideo(ctx context.Context, input VideoRequest) (VideoResponse, error)
	Name() string
}

type ImageRequest struct {
	Prompt         string         `json:"prompt"`
	NegativePrompt string         `json:"negative_prompt,omitempty"`
	Width          int            `json:"width,omitempty"`
	Height         int            `json:"height,omitempty"`
	AspectRatio    string         `json:"aspect_ratio,omitempty"`
	Seed           int64          `json:"seed,omitempty"`
	Count          int            `json:"count,omitempty"`
	Quality        string         `json:"quality,omitempty"`
	OutputFormat   string         `json:"output_format,omitempty"`
	Background     string         `json:"background,omitempty"`
	ReferenceImage string         `json:"reference_image,omitempty"`
	MaskImage      string         `json:"mask_image,omitempty"`
	Options        map[string]any `json:"options,omitempty"`
}

type ImageResponse struct {
	AssetPath     string         `json:"asset_path,omitempty"`
	URL           string         `json:"url,omitempty"`
	MIMEType      string         `json:"mime_type,omitempty"`
	Model         string         `json:"model"`
	RevisedPrompt string         `json:"revised_prompt,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type ImageProvider interface {
	GenerateImage(ctx context.Context, input ImageRequest) (ImageResponse, error)
	Name() string
}
