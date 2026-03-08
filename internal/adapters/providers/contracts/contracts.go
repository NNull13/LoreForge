package contracts

import "context"

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
	Prompt   string `json:"prompt"`
	Duration int    `json:"duration_seconds"`
	Seed     int64  `json:"seed"`
}

type VideoResponse struct {
	AssetPath string `json:"asset_path"`
	Model     string `json:"model"`
}

type VideoProvider interface {
	GenerateVideo(ctx context.Context, input VideoRequest) (VideoResponse, error)
	Name() string
}

type ImageRequest struct {
	Prompt string `json:"prompt"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Seed   int64  `json:"seed"`
}

type ImageResponse struct {
	AssetPath string `json:"asset_path"`
	Model     string `json:"model"`
}

type ImageProvider interface {
	GenerateImage(ctx context.Context, input ImageRequest) (ImageResponse, error)
	Name() string
}
