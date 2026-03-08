package mock

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"loreforge/internal/adapters/providers/contracts"
)

type ImageProvider struct {
	Model string
}

func (p ImageProvider) Name() string { return "mock-image" }

func (p ImageProvider) GenerateImage(_ context.Context, input contracts.ImageRequest) (contracts.ImageResponse, error) {
	outDir := filepath.Join("./out", "image-assets")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return contracts.ImageResponse{}, err
	}
	path := filepath.Join(outDir, fmt.Sprintf("image-%d.png", time.Now().UnixNano()))
	stub := []byte("PNG PLACEHOLDER\n" + input.Prompt + "\n")
	if err := os.WriteFile(path, stub, 0o644); err != nil {
		return contracts.ImageResponse{}, err
	}
	return contracts.ImageResponse{
		AssetPath: path,
		Model:     coalesce(p.Model, "mock-image-v1"),
		MIMEType:  "image/png",
	}, nil
}
