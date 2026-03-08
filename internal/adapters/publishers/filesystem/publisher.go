package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"loreforge/internal/domain/publication"
)

type Publisher struct {
	OutputDir string
}

func (p Publisher) Name() publication.ChannelName { return publication.ChannelFilesystem }

func (p Publisher) Publish(_ context.Context, item publication.Item) (publication.Result, error) {
	if err := os.MkdirAll(p.OutputDir, 0o755); err != nil {
		return publication.Result{}, err
	}
	stamp := time.Now().Format("20060102-150405")
	if item.AssetPath != "" {
		content, err := os.ReadFile(item.AssetPath)
		if err != nil {
			return publication.Result{}, err
		}
		ext := filepath.Ext(item.AssetPath)
		if ext == "" {
			switch item.OutputType {
			case "video":
				ext = ".mp4"
			case "image":
				ext = ".png"
			default:
				ext = ".bin"
			}
		}
		target := filepath.Join(p.OutputDir, fmt.Sprintf("%s-%s%s", stamp, item.EpisodeID, ext))
		if err := os.WriteFile(target, content, 0o644); err != nil {
			return publication.Result{}, err
		}
		return publication.Result{Channel: string(p.Name()), Success: true, ExternalID: target, Message: "published asset"}, nil
	}
	target := filepath.Join(p.OutputDir, fmt.Sprintf("%s-%s.txt", stamp, item.EpisodeID))
	if err := os.WriteFile(target, []byte(item.Content), 0o644); err != nil {
		return publication.Result{}, err
	}
	result := publication.Result{Channel: string(p.Name()), Success: true, ExternalID: target, Message: "published text"}
	if len(item.Parts) > 0 {
		threadPath := filepath.Join(p.OutputDir, fmt.Sprintf("%s-%s-thread.json", stamp, item.EpisodeID))
		body, err := json.MarshalIndent(map[string]any{
			"episode_id": item.EpisodeID,
			"format":     item.Format,
			"parts":      item.Parts,
		}, "", "  ")
		if err != nil {
			return publication.Result{}, err
		}
		if err := os.WriteFile(threadPath, body, 0o644); err != nil {
			return publication.Result{}, err
		}
		result.Metadata = map[string]any{"thread_path": threadPath}
	}
	return result, nil
}
