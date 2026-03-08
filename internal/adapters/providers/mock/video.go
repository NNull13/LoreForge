package mock

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	providercontracts "loreforge/internal/adapters/providers/contracts"
)

type VideoProvider struct {
	Model string
}

func (p VideoProvider) Name() string { return "mock-video" }

func (p VideoProvider) GenerateVideo(_ context.Context, input providercontracts.VideoRequest) (providercontracts.VideoResponse, error) {
	outDir := filepath.Join("./out", "video-assets")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return providercontracts.VideoResponse{}, err
	}
	path := filepath.Join(outDir, fmt.Sprintf("video-%d.mp4", time.Now().UnixNano()))
	stub := []byte("MVP video placeholder\n" + input.Prompt + "\n")
	if err := os.WriteFile(path, stub, 0o644); err != nil {
		return providercontracts.VideoResponse{}, err
	}
	return providercontracts.VideoResponse{
		AssetPath: path,
		Model:     coalesce(p.Model, "mock-video-v1"),
		Metadata:  map[string]any{"driver": "mock"},
	}, nil
}
