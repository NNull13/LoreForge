package providers

import (
	"fmt"
	"strings"

	"loreforge/internal/config"
	"loreforge/pkg/contracts"
)

func NewTextProvider(cfg config.ProviderDriver) (contracts.TextProvider, error) {
	switch normalizeDriver(cfg.Driver, "mock") {
	case "mock":
		return MockTextProvider{Model: cfg.Model}, nil
	default:
		return nil, fmt.Errorf("unsupported text provider driver: %s", cfg.Driver)
	}
}

func NewVideoProvider(cfg config.ProviderDriver) (contracts.VideoProvider, error) {
	switch normalizeDriver(cfg.Driver, "mock") {
	case "mock":
		return MockVideoProvider{Model: cfg.Model}, nil
	default:
		return nil, fmt.Errorf("unsupported video provider driver: %s", cfg.Driver)
	}
}

func NewImageProvider(cfg config.ProviderDriver) (contracts.ImageProvider, error) {
	switch normalizeDriver(cfg.Driver, "mock") {
	case "mock":
		return MockImageProvider{Model: cfg.Model}, nil
	default:
		return nil, fmt.Errorf("unsupported image provider driver: %s", cfg.Driver)
	}
}

func normalizeDriver(value string, fallback string) string {
	driver := strings.TrimSpace(strings.ToLower(value))
	if driver == "" {
		return fallback
	}
	return driver
}
