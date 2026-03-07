package channels

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"loreforge/pkg/contracts"
)

type FilesystemChannel struct {
	OutputDir string
}

func (f FilesystemChannel) Name() string { return "filesystem" }

func (f FilesystemChannel) Publish(_ context.Context, item contracts.PublishableContent) (contracts.PublishResult, error) {
	if err := os.MkdirAll(f.OutputDir, 0o755); err != nil {
		return contracts.PublishResult{}, err
	}
	stamp := time.Now().Format("20060102-150405")
	if item.OutputType == "video" && item.AssetPath != "" {
		b, err := os.ReadFile(item.AssetPath)
		if err != nil {
			return contracts.PublishResult{}, err
		}
		target := filepath.Join(f.OutputDir, fmt.Sprintf("%s-%s.mp4", stamp, item.EpisodeID))
		if err := os.WriteFile(target, b, 0o644); err != nil {
			return contracts.PublishResult{}, err
		}
		return contracts.PublishResult{Channel: f.Name(), Success: true, ExternalID: target, Message: "published video"}, nil
	}
	target := filepath.Join(f.OutputDir, fmt.Sprintf("%s-%s.txt", stamp, item.EpisodeID))
	if err := os.WriteFile(target, []byte(item.Content), 0o644); err != nil {
		return contracts.PublishResult{}, err
	}
	return contracts.PublishResult{Channel: f.Name(), Success: true, ExternalID: target, Message: "published text"}, nil
}
