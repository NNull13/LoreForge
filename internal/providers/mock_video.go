package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"loreforge/pkg/contracts"
)

type MockVideoProvider struct {
	Model string
}

func (m MockVideoProvider) Name() string { return "mock-video" }

func (m MockVideoProvider) GenerateVideo(_ context.Context, input contracts.VideoRequest) (contracts.VideoResponse, error) {
	outDir := filepath.Join("./out", "video-assets")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return contracts.VideoResponse{}, err
	}
	path := filepath.Join(outDir, fmt.Sprintf("video-%d.mp4", time.Now().UnixNano()))
	stub := []byte("MVP video placeholder\n" + input.Prompt + "\n")
	if err := os.WriteFile(path, stub, 0o644); err != nil {
		return contracts.VideoResponse{}, err
	}
	return contracts.VideoResponse{AssetPath: path, Model: coalesceVideo(m.Model, "mock-video-v1")}, nil
}

func coalesceVideo(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
