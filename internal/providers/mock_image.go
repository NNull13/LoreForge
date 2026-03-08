package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"loreforge/pkg/contracts"
)

type MockImageProvider struct {
	Model string
}

func (m MockImageProvider) Name() string { return "mock-image" }

func (m MockImageProvider) GenerateImage(_ context.Context, input contracts.ImageRequest) (contracts.ImageResponse, error) {
	outDir := filepath.Join("./out", "image-assets")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return contracts.ImageResponse{}, err
	}
	path := filepath.Join(outDir, fmt.Sprintf("image-%d.png", time.Now().UnixNano()))
	stub := []byte("PNG PLACEHOLDER\n" + input.Prompt + "\n")
	if err := os.WriteFile(path, stub, 0o644); err != nil {
		return contracts.ImageResponse{}, err
	}
	return contracts.ImageResponse{AssetPath: path, Model: coalesceImage(m.Model, "mock-image-v1")}, nil
}

func coalesceImage(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
