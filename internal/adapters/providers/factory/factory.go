package factory

import (
	"fmt"
	"strings"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/adapters/providers/lmstudio_text"
	"loreforge/internal/adapters/providers/mock"
	"loreforge/internal/adapters/providers/openai_image"
	"loreforge/internal/adapters/providers/openai_text"
	"loreforge/internal/adapters/providers/runway_video"
	"loreforge/internal/adapters/providers/vertex_imagen"
	"loreforge/internal/adapters/providers/vertex_veo"
	"loreforge/internal/config"
)

func NewTextProvider(cfg config.ProviderDriver) (contracts.TextProvider, error) {
	switch normalizeDriver(cfg.Driver) {
	case "mock":
		return mock.TextProvider{Model: cfg.Model}, nil
	case "openai_text":
		return openai_text.Provider{Config: cfg}, nil
	case "lmstudio_text":
		return lmstudio_text.Provider{Config: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported text provider driver: %s", cfg.Driver)
	}
}

func NewVideoProvider(cfg config.ProviderDriver) (contracts.VideoProvider, error) {
	switch normalizeDriver(cfg.Driver) {
	case "mock":
		return mock.VideoProvider{Model: cfg.Model}, nil
	case "vertex_veo":
		return vertex_veo.Provider{Config: cfg}, nil
	case "runway_gen4":
		return runway_video.Provider{Config: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported video provider driver: %s", cfg.Driver)
	}
}

func NewImageProvider(cfg config.ProviderDriver) (contracts.ImageProvider, error) {
	switch normalizeDriver(cfg.Driver) {
	case "mock":
		return mock.ImageProvider{Model: cfg.Model}, nil
	case "openai_image":
		return openai_image.Provider{Config: cfg}, nil
	case "vertex_imagen":
		return vertex_imagen.Provider{Config: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported image provider driver: %s", cfg.Driver)
	}
}

func normalizeDriver(driver string) string {
	value := strings.TrimSpace(strings.ToLower(driver))
	if value == "" {
		return "mock"
	}
	return value
}
