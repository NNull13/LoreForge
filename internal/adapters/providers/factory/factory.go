package factory

import (
	"fmt"
	"strings"

	"loreforge/internal/adapters/providers/contracts"
	"loreforge/internal/adapters/providers/lmstudiotext"
	"loreforge/internal/adapters/providers/mock"
	"loreforge/internal/adapters/providers/openaiimage"
	"loreforge/internal/adapters/providers/openaitext"
	"loreforge/internal/adapters/providers/runwayvideo"
	"loreforge/internal/adapters/providers/verteximagen"
	"loreforge/internal/adapters/providers/vertexveo"
	"loreforge/internal/config"
)

func NewTextProvider(cfg config.ProviderDriver) (contracts.TextProvider, error) {
	switch normalizeDriver(cfg.Driver) {
	case "mock":
		return mock.TextProvider{Model: cfg.Model}, nil
	case "openai_text":
		return openaitext.Provider{Config: cfg}, nil
	case "lmstudio_text":
		return lmstudiotext.Provider{Config: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported text provider driver: %s", cfg.Driver)
	}
}

func NewVideoProvider(cfg config.ProviderDriver) (contracts.VideoProvider, error) {
	switch normalizeDriver(cfg.Driver) {
	case "mock":
		return mock.VideoProvider{Model: cfg.Model}, nil
	case "vertex_veo":
		return vertexveo.Provider{Config: cfg}, nil
	case "runway_gen4":
		return runwayvideo.Provider{Config: cfg}, nil
	default:
		return nil, fmt.Errorf("unsupported video provider driver: %s", cfg.Driver)
	}
}

func NewImageProvider(cfg config.ProviderDriver) (contracts.ImageProvider, error) {
	switch normalizeDriver(cfg.Driver) {
	case "mock":
		return mock.ImageProvider{Model: cfg.Model}, nil
	case "openai_image":
		return openaiimage.Provider{Config: cfg}, nil
	case "vertex_imagen":
		return verteximagen.Provider{Config: cfg}, nil
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
